package module

import (
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/codec"
	codectypes "github.com/line/lbm-sdk/codec/types"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/types/module"
	"github.com/line/lbm-sdk/x/rollup"
	"github.com/line/lbm-sdk/x/rollup/keeper"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
)

var _ module.AppModuleBasic = AppModuleBasic{}
var _ module.AppModule = AppModule{}
var _ module.AppModuleGenesis = AppModule{}

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return rollup.ModuleName
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(&rollup.GenesisState{})
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	return nil
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (b AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {}

type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

func (AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (am AppModule) Route() sdk.Route { return sdk.Route{} }

func (AppModule) QuerierRoute() string { return "" }

func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return nil
}

func (am AppModule) RegisterServices(cfg module.Configurator) {}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

func (AppModule) ConsensusVersion() uint64 { return 1 }

func NewAppModule(keeper keeper.Keeper) AppModule {
	return AppModule{
		keeper: keeper,
	}
}
