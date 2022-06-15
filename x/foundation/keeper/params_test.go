package keeper_test

import (
	"testing"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/x/foundation"
)

func TestGetSetParams(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	k := app.FoundationKeeper

	params := &foundation.Params{
		Enabled: true,
	}
	k.SetParams(ctx, params)
	require.Equal(t, params, k.GetParams(ctx))
	require.Equal(t, params.Enabled, k.GetEnabled(ctx))
}
