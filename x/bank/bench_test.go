package bank_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	ocabci "github.com/line/ostracon/abci/types"

	"github.com/line/lbm-sdk/simapp"
	simappparams "github.com/line/lbm-sdk/simapp/params"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/x/auth/types"
	authtypes "github.com/line/lbm-sdk/x/auth/types"
	stakingtypes "github.com/line/lbm-sdk/x/staking/types"
)

var moduleAccAddr = authtypes.NewModuleAddress(stakingtypes.BondedPoolName)

func BenchmarkOneBankSendTxPerBlock(b *testing.B) {
	b.ReportAllocs()
	// Add an account at genesis
	acc := authtypes.BaseAccount{
		Address: addr1.String(),
	}

	// construct genesis state
	genAccs := []types.GenesisAccount{&acc}
	benchmarkApp := simapp.SetupWithGenesisAccounts(genAccs)
	ctx := benchmarkApp.BaseApp.NewContext(false, tmproto.Header{})

	// some value conceivably higher than the benchmarks would ever go
	require.NoError(b, simapp.FundAccount(benchmarkApp, ctx, addr1, sdk.NewCoins(sdk.NewInt64Coin("foocoin", 100000000000))))

	benchmarkApp.Commit()
	txGen := simappparams.MakeTestEncodingConfig().TxConfig

	// Precompute all txs
	txs, err := simapp.GenSequenceOfTxs(txGen, []sdk.Msg{sendMsg1}, []uint64{0}, []uint64{uint64(0)}, b.N, priv1)
	require.NoError(b, err)
	b.ResetTimer()

	height := int64(3)

	// Run this with a profiler, so its easy to distinguish what time comes from
	// Committing, and what time comes from Check/Deliver Tx.
	for i := 0; i < b.N; i++ {
		benchmarkApp.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: height}})
		_, err := benchmarkApp.Check(txGen.TxEncoder(), txs[i])
		if err != nil {
			panic("something is broken in checking transaction")
		}

		_, _, err = benchmarkApp.Deliver(txGen.TxEncoder(), txs[i])
		require.NoError(b, err)
		benchmarkApp.EndBlock(abci.RequestEndBlock{Height: height})
		benchmarkApp.Commit()
		height++
	}
}

func BenchmarkOneBankMultiSendTxPerBlock(b *testing.B) {
	b.ReportAllocs()
	// Add an account at genesis
	acc := authtypes.BaseAccount{
		Address: addr1.String(),
	}

	// Construct genesis state
	genAccs := []authtypes.GenesisAccount{&acc}
	benchmarkApp := simapp.SetupWithGenesisAccounts(genAccs)
	ctx := benchmarkApp.BaseApp.NewContext(false, tmproto.Header{})

	// some value conceivably higher than the benchmarks would ever go
	require.NoError(b, simapp.FundAccount(benchmarkApp, ctx, addr1, sdk.NewCoins(sdk.NewInt64Coin("foocoin", 100000000000))))

	benchmarkApp.Commit()
	txGen := simappparams.MakeTestEncodingConfig().TxConfig

	// Precompute all txs
	txs, err := simapp.GenSequenceOfTxs(txGen, []sdk.Msg{multiSendMsg1}, []uint64{0}, []uint64{uint64(0)}, b.N, priv1)
	require.NoError(b, err)
	b.ResetTimer()

	height := int64(3)

	// Run this with a profiler, so its easy to distinguish what time comes from
	// Committing, and what time comes from Check/Deliver Tx.
	for i := 0; i < b.N; i++ {
		benchmarkApp.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: height}})
		_, err := benchmarkApp.Check(txGen.TxEncoder(), txs[i])
		if err != nil {
			panic("something is broken in checking transaction")
		}

		_, _, err = benchmarkApp.Deliver(txGen.TxEncoder(), txs[i])
		require.NoError(b, err)
		benchmarkApp.EndBlock(abci.RequestEndBlock{Height: height})
		benchmarkApp.Commit()
		height++
	}
}
