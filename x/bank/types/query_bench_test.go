package types_test

import (
	"github.com/line/lbm-sdk/crypto/keys/secp256k1"
	"github.com/line/lbm-sdk/simapp"
	"github.com/line/lbm-sdk/types"
	authtypes "github.com/line/lbm-sdk/x/auth/types"
	bankkeeper "github.com/line/lbm-sdk/x/bank/keeper"
	banktypes "github.com/line/lbm-sdk/x/bank/types"
	stakingtypes "github.com/line/lbm-sdk/x/staking/types"
	ocproto "github.com/line/ostracon/proto/ostracon/types"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

// % go test -bench=. ./x/bank/types -benchmem

func BenchmarkPrefetchEffect(b *testing.B) {
	const ACCOUNTS = 5000

	addrs := make([]types.AccAddress, ACCOUNTS)
	accounts := make([]authtypes.GenesisAccount, ACCOUNTS)
	balances := make([]banktypes.Balance, ACCOUNTS)
	for i := 0; i < ACCOUNTS; i++ {
		priv1 := secp256k1.GenPrivKey()
		addrs[i] = types.BytesToAccAddress(priv1.PubKey().Address())
		accounts[i] = &authtypes.BaseAccount{
			Address: addrs[i].String(),
		}
		coins := types.Coins{types.NewInt64Coin("foocoin", 0)}
		balances[i] = banktypes.Balance{
			Address: addrs[i].String(),
			Coins:   coins,
		}
	}
	app := simapp.SetupWithGenesisAccounts(accounts, balances...)

	moduleAccAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		app.AppCodec(), app.GetKey(banktypes.StoreKey), app.AccountKeeper, app.GetSubspace(authtypes.ModuleName), map[string]bool{
			moduleAccAddr.String(): true,
		},
	)
	ctx := app.BaseApp.NewContext(false, ocproto.Header{})
	context := types.WrapSDKContext(ctx)

	rand.Seed(time.Now().UnixNano())
	b.Run("BankKeeper/Balance", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			j := rand.Intn(ACCOUNTS)
			req := banktypes.NewQueryBalanceRequest(addrs[j], "foocoin")
			b.StartTimer()
			res, err := app.BankKeeper.Balance(context, req)
			b.StopTimer()
			require.NoError(b, err)
			require.Equal(b, res.Balance.Denom, "foocoin")
			time.Sleep(1 * time.Millisecond) // cede CPU time to other go routine to facilitate Prefetch()
		}
	})

	b.Run("AccountKeeper/Account", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			j := rand.Intn(ACCOUNTS)
			req1 := banktypes.NewQueryBalanceRequest(addrs[j], "foocoin")
			res1, err := app.BankKeeper.Balance(context, req1)
			require.NoError(b, err)
			require.Equal(b, res1.Balance.Denom, "foocoin")
			time.Sleep(1 * time.Millisecond) // cede CPU time to other go routine to facilitate Prefetch()

			req2 := &authtypes.QueryAccountRequest{
				Address: addrs[j].String(),
			}
			b.StartTimer()
			_, err = app.AccountKeeper.Account(context, req2)
			b.StopTimer()
			require.NoError(b, err)
		}
	})
}
