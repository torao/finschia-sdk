package keeper

import (
	"math/rand"
	"testing"

	"github.com/tendermint/tendermint/libs/log"
	ostproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/line/tm-db/v2/memdb"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func genAddress() sdk.AccAddress {
	b := make([]byte, 20)
	rand.Read(b)
	return sdk.BytesToAccAddress(b)
}

func setupKeeper(storeKey *sdk.KVStoreKey) BaseKeeper {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	amino := codec.NewLegacyAmino()
	testTransientStoreKey := sdk.NewTransientStoreKey("test")

	accountStoreKey := sdk.NewKVStoreKey(accounttypes.StoreKey)
	accountSubspace := paramtypes.NewSubspace(cdc, amino, accountStoreKey, testTransientStoreKey, accounttypes.ModuleName)
	accountKeeper := accountkeeper.NewAccountKeeper(cdc, accountStoreKey, accountSubspace, accounttypes.ProtoBaseAccount, nil)

	bankSubspace := paramtypes.NewSubspace(cdc, amino, storeKey, testTransientStoreKey, banktypes.StoreKey)
	return NewBaseKeeper(cdc, storeKey, accountKeeper, bankSubspace, nil)
}

func setupContext(t *testing.T, storeKey *sdk.KVStoreKey) sdk.Context {
	db := memdb.NewDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	return sdk.NewContext(stateStore, ostproto.Header{}, false, log.NewNopLogger())
}

func TestInactiveAddr(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(banktypes.StoreKey)
	bankKeeper := setupKeeper(storeKey)
	ctx := setupContext(t, storeKey)

	addr := genAddress()

	require.Equal(t, 0, len(bankKeeper.inactiveAddrs))

	bankKeeper.addToInactiveAddr(ctx, addr)
	require.True(t, bankKeeper.isStoredInactiveAddr(ctx, addr))

	// duplicate addition, no error expects.
	bankKeeper.addToInactiveAddr(ctx, addr)
	require.True(t, bankKeeper.isStoredInactiveAddr(ctx, addr))

	bankKeeper.deleteFromInactiveAddr(ctx, addr)
	require.False(t, bankKeeper.isStoredInactiveAddr(ctx, addr))

	addr2 := genAddress()
	require.False(t, bankKeeper.isStoredInactiveAddr(ctx, addr2))

	// expect no error
	bankKeeper.deleteFromInactiveAddr(ctx, addr2)

	// test loadAllInactiveAddrs
	bankKeeper.addToInactiveAddr(ctx, addr)
	bankKeeper.addToInactiveAddr(ctx, addr2)
	require.Equal(t, 0, len(bankKeeper.inactiveAddrs))
	bankKeeper.loadAllInactiveAddrs(ctx)
	require.Equal(t, 2, len(bankKeeper.inactiveAddrs))
}
