package keeper

import (
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/x/rollup"
)

// Keeper defines the token module Keeper
type Keeper struct{}

// ExportGenesis returns a GenesisState for a given context.
func (k Keeper) ExportGenesis(ctx sdk.Context) *rollup.GenesisState {
	return &rollup.GenesisState{}
}

// NewKeeper returns a token keeper
func NewKeeper() Keeper {
	return Keeper{}
}
