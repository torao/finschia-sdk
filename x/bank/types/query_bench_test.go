package types_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/line/lbm-sdk/crypto/keys/secp256k1"
	"github.com/line/lbm-sdk/simapp"
	"github.com/line/lbm-sdk/store/prefix"
	"github.com/line/lbm-sdk/types"
	authtypes "github.com/line/lbm-sdk/x/auth/types"
	bankkeeper "github.com/line/lbm-sdk/x/bank/keeper"
	banktypes "github.com/line/lbm-sdk/x/bank/types"
	stakingtypes "github.com/line/lbm-sdk/x/staking/types"
	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/log"
	ocproto "github.com/line/ostracon/proto/ostracon/types"
	db "github.com/line/tm-db/v2"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// % go test -bench=. ./x/bank/types -benchmem

const ACCOUNTS = 5000
const DENOM = "foocoin"

func BenchmarkPrefetchBeforeAccount(b *testing.B) {
	name := "query_bench_test"
	dir := os.TempDir()
	dbDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s.db", name))
	kvsGen := func() db.DB {
		if _, err := os.Stat(dbDir); !os.IsNotExist(err) {
			err := os.RemoveAll(dbDir)
			if err != nil {
				panic(err)
			}
		}
		if _, err := os.Stat(dbDir); !os.IsNotExist(err) {
			panic("")
		}
		kvs, err := types.NewLevelDB(name, dir)
		if err != nil {
			panic(err)
		}
		return kvs
	}

	seed := time.Now().UnixNano()

	app, _, ctx, addrs := prepare(kvsGen)
	rand.Seed(seed)
	b.ResetTimer()
	b.Run("AccountKeeper/AccountDirect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			j := rand.Intn(len(addrs))

			req := &authtypes.QueryAccountRequest{
				Address: addrs[j].String(),
			}
			_, err := app.AccountKeeper.Account(ctx, req)
			require.NoError(b, err)
		}
	})

	app, _, ctx, addrs = prepare(kvsGen)
	rand.Seed(seed)
	b.ResetTimer()
	b.Run("AccountKeeper/AccountAfterPrefetch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			j := rand.Intn(len(addrs))

			sdkCtx := types.UnwrapSDKContext(ctx)
			store := sdkCtx.KVStore(app.GetKey(banktypes.StoreKey))
			balancesStore := prefix.NewStore(store, banktypes.BalancesPrefix)
			accountStore := prefix.NewStore(balancesStore, bankkeeper.AddressToPrefixKey(addrs[j]))
			app.AccountKeeper.Prefetch(sdkCtx, addrs[j], true)
			accountStore.Prefetch([]byte(DENOM), true)

			req2 := &authtypes.QueryAccountRequest{
				Address: addrs[j].String(),
			}
			_, err := app.AccountKeeper.Account(ctx, req2)
			require.NoError(b, err)
		}
	})

	app, _, ctx, addrs = prepare(kvsGen)
	rand.Seed(seed)
	b.ResetTimer()
	b.Run("AccountKeeper/Prefetch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			j := rand.Intn(len(addrs))

			sdkCtx := types.UnwrapSDKContext(ctx)
			store := sdkCtx.KVStore(app.GetKey(banktypes.StoreKey))
			balancesStore := prefix.NewStore(store, banktypes.BalancesPrefix)
			accountStore := prefix.NewStore(balancesStore, bankkeeper.AddressToPrefixKey(addrs[j]))
			app.AccountKeeper.Prefetch(sdkCtx, addrs[j], true)
			accountStore.Prefetch([]byte(DENOM), true)
		}
	})

	rand.Seed(seed)
	b.ResetTimer()
	b.Run("AccountKeeper/Background", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var j = rand.Intn(len(addrs))
			var _ = &authtypes.QueryAccountRequest{
				Address: addrs[j].String(),
			}
		}
	})

	var _ = os.RemoveAll(dbDir)
}

func prepare(kvsGen func() db.DB) (*simapp.SimApp, *bankkeeper.BaseKeeper, context.Context, []types.AccAddress) {
	addrs := make([]types.AccAddress, ACCOUNTS)
	accounts := make([]authtypes.GenesisAccount, ACCOUNTS)
	balances := make([]banktypes.Balance, ACCOUNTS)
	for i := 0; i < ACCOUNTS; i++ {
		priv1 := secp256k1.GenPrivKey()
		addrs[i] = types.BytesToAccAddress(priv1.PubKey().Address())
		accounts[i] = &authtypes.BaseAccount{
			Address: addrs[i].String(),
		}
		coins := types.Coins{types.NewInt64Coin(DENOM, 0)}
		balances[i] = banktypes.Balance{
			Address: addrs[i].String(),
			Coins:   coins,
		}
	}

	// NOTE: This is the same as SetupWithGenesisAccounts() impl except that goleveldb is used instead of memdb
	kvs := kvsGen()
	app := setupWithGenesisAccounts(kvs, accounts, balances...)

	moduleAccAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		app.AppCodec(), app.GetKey(banktypes.StoreKey), app.AccountKeeper, app.GetSubspace(authtypes.ModuleName), map[string]bool{
			moduleAccAddr.String(): true,
		},
	)
	ctx := app.BaseApp.NewContext(false, ocproto.Header{})
	bk := bankkeeper.NewBaseKeeper(app.AppCodec(), app.GetKey(banktypes.StoreKey), app.AccountKeeper, app.GetSubspace(banktypes.ModuleName), map[string]bool{})

	return app, &bk, types.WrapSDKContext(ctx), addrs
}

func setup(db db.DB, withGenesis bool, invCheckPeriod uint) (*simapp.SimApp, simapp.GenesisState) {
	encCdc := simapp.MakeTestEncodingConfig()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, simapp.DefaultNodeHome, invCheckPeriod, encCdc, simapp.EmptyAppOptions{})
	if withGenesis {
		return app, simapp.NewDefaultGenesisState(encCdc.Marshaler)
	}
	return app, simapp.GenesisState{}
}

func setupWithGenesisAccounts(db db.DB, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *simapp.SimApp {
	app, genesisState := setup(db, true, 0)
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	totalSupply := types.NewCoins()
	for _, b := range balances {
		totalSupply = totalSupply.Add(b.Coins...)
	}

	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	app.InitChain(
		abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: simapp.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	app.Commit()
	app.BeginBlock(abci.RequestBeginBlock{Header: ocproto.Header{Height: app.LastBlockHeight() + 1}})

	return app
}
