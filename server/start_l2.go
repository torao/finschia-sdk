//go:build layer2
// +build layer2

package server

import (
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/codec"
	"github.com/line/lbm-sdk/server/api"
	"github.com/line/lbm-sdk/server/config"
	servergrpc "github.com/line/lbm-sdk/server/grpc"
	"github.com/line/lbm-sdk/server/rosetta"
	crgserver "github.com/line/lbm-sdk/server/rosetta/lib/server"
	"github.com/line/lbm-sdk/server/types"
	"github.com/line/ostracon/node"
	"github.com/line/ostracon/p2p"
)

func addLayer2Flags(cmd *cobra.Command) {
	rollconf.AddFlags(cmd)
}

func startInProcess(ctx *Context, clientCtx client.Context, appCreator types.AppCreator) error {
	cfg := ctx.Config
	home := cfg.RootDir
	var cpuProfileCleanup func()

	if cpuProfile := ctx.Viper.GetString(flagCPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}

		ctx.Logger.Info("starting CPU profiler", "profile", cpuProfile)
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		cpuProfileCleanup = func() {
			ctx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()
			f.Close()
		}
	}

	traceWriterFile := ctx.Viper.GetString(flagTraceStore)
	db, err := openDB(home)
	if err != nil {
		return err
	}

	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return err
	}

	config, err := config.GetConfig(ctx.Viper)
	if err != nil {
		return err
	}

	if err := config.ValidateBasic(); err != nil {
		ctx.Logger.Error("WARNING: The minimum-gas-prices config in app.toml is set to the empty string. " +
			"This defaults to 0 in the current version, but will error in the next version " +
			"(SDK v0.45). Please explicitly put the desired minimum-gas-prices in your app.toml.")
	}

	app := appCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}

	genDocProvider := node.DefaultGenesisDocProviderFunc(cfg)

	var (
		/*
			ocNode *node.Node
			/*/
		tmNode rollnode.Node
		server *rollrpc.Server
		//*/
		gRPCOnly = ctx.Viper.GetBool(flagGRPCOnly)
	)

	if gRPCOnly {
		ctx.Logger.Info("starting node in gRPC only mode; Ostracon is disabled")
		config.GRPC.Enable = true
	} else {
		/*
			ctx.Logger.Info("starting node with ABCI Ostracon in-process")

			pv := pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())

			ocNode, err = node.NewNode(
				cfg,
				pv,
				nodeKey,
				proxy.NewLocalClientCreator(app),
				genDocProvider,
				node.DefaultDBProvider,
				node.DefaultMetricsProvider(cfg.Instrumentation),
				ctx.Logger,
			)
			if err != nil {
				return err
			}
			ctx.Logger.Debug("initialization: ocNode created")
			if err := ocNode.Start(); err != nil {
				return err
			}
			ctx.Logger.Debug("initialization: ocNode started")
			/*/
		ctx.Logger.Info("starting node with ABCI rollmint in-process")

		// keys in rollmint format
		p2pKey, err := rollconv.GetNodeKey(nodeKey)
		if err != nil {
			return err
		}
		signingKey, err := rollconv.GetNodeKey(privValKey)
		if err != nil {
			return err
		}
		genesis, err := genDocProvider()
		if err != nil {
			return err
		}
		nodeConfig := rollconf.NodeConfig{}
		err = nodeConfig.GetViperConfig(ctx.Viper)
		if err != nil {
			return err
		}
		rollconv.GetNodeConfig(&nodeConfig, cfg)
		err = rollconv.TranslateAddresses(&nodeConfig)
		if err != nil {
			return err
		}
		tmNode, err = rollnode.NewNode(
			context.Background(),
			nodeConfig,
			p2pKey,
			signingKey,
			abciclient.NewLocalClient(nil, app),
			genesis,
			ctx.Logger,
		)
		if err != nil {
			return err
		}

		server = rollrpc.NewServer(tmNode, cfg.RPC, ctx.Logger)
		err = server.Start()
		if err != nil {
			return err
		}

		ctx.Logger.Debug("initialization: tmNode created")
		if err := tmNode.Start(); err != nil {
			return err
		}
		ctx.Logger.Debug("initialization: tmNode started")
		//*/
	}

	// Add the tx service to the gRPC router. We only need to register this
	// service if API or gRPC is enabled, and avoid doing so in the general
	// case, because it spawns a new local ostracon RPC client.
	/*
		if (config.API.Enable || config.GRPC.Enable) && ocNode != nil {
			clientCtx = clientCtx.WithClient(local.New(ocNode))
			/*/
	if config.API.Enable || config.GRPC.Enable {
		clientCtx = clientCtx.WithClient(server.Client())
		//*/

		app.RegisterTxService(clientCtx)
		app.RegisterTendermintService(clientCtx)

		if a, ok := app.(types.ApplicationQueryService); ok {
			a.RegisterNodeService(clientCtx)
		}
	}

	metrics, err := startTelemetry(config)
	if err != nil {
		return err
	}

	var apiSrv *api.Server
	if config.API.Enable {
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx := clientCtx.WithHomeDir(home).WithChainID(genDoc.ChainID)

		apiSrv = api.New(clientCtx, ctx.Logger.With("module", "api-server"))
		app.RegisterAPIRoutes(apiSrv, config.API)
		if config.Telemetry.Enabled {
			apiSrv.SetTelemetry(metrics)
		}
		errCh := make(chan error)

		go func() {
			if err := apiSrv.Start(config); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err

		case <-time.After(types.ServerStartTime): // assume server started successfully
		}
	}

	var (
		grpcSrv    *grpc.Server
		grpcWebSrv *http.Server
	)

	if config.GRPC.Enable {
		grpcSrv, err = servergrpc.StartGRPCServer(clientCtx, app, config.GRPC.Address)
		if err != nil {
			return err
		}

		if config.GRPCWeb.Enable {
			grpcWebSrv, err = servergrpc.StartGRPCWeb(grpcSrv, config)
			if err != nil {
				ctx.Logger.Error("failed to start grpc-web http server: ", err)
				return err
			}
		}
	}

	// At this point it is safe to block the process if we're in gRPC only mode as
	// we do not need to start Rosetta or handle any Tendermint related processes.
	if gRPCOnly {
		// wait for signal capture and gracefully return
		return WaitForQuitSignals()
	}

	var rosettaSrv crgserver.Server
	if config.Rosetta.Enable {
		offlineMode := config.Rosetta.Offline

		// If GRPC is not enabled rosetta cannot work in online mode, so it works in
		// offline mode.
		if !config.GRPC.Enable {
			offlineMode = true
		}

		conf := &rosetta.Config{
			Blockchain:        config.Rosetta.Blockchain,
			Network:           config.Rosetta.Network,
			TendermintRPC:     ctx.Config.RPC.ListenAddress,
			GRPCEndpoint:      config.GRPC.Address,
			Addr:              config.Rosetta.Address,
			Retries:           config.Rosetta.Retries,
			Offline:           offlineMode,
			Codec:             clientCtx.Codec.(*codec.ProtoCodec),
			InterfaceRegistry: clientCtx.InterfaceRegistry,
		}

		rosettaSrv, err = rosetta.ServerFromConfig(conf)
		if err != nil {
			return err
		}

		errCh := make(chan error)
		go func() {
			if err := rosettaSrv.Start(); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err

		case <-time.After(types.ServerStartTime): // assume server started successfully
		}
	}

	defer func() {
		if ocNode.IsRunning() {
			_ = ocNode.Stop()
		}

		if cpuProfileCleanup != nil {
			cpuProfileCleanup()
		}

		if apiSrv != nil {
			_ = apiSrv.Close()
		}

		if grpcSrv != nil {
			grpcSrv.Stop()
			if grpcWebSrv != nil {
				grpcWebSrv.Close()
			}
		}

		ctx.Logger.Info("exiting...")
	}()

	// wait for signal capture and gracefully return
	return WaitForQuitSignals()
}
