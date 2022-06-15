package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/token"
)

// Keeper defines the token module Keeper
type Keeper struct {
	accountKeeper token.AccountKeeper
	classKeeper   token.ClassKeeper

	// The (unexposed) keys used to access the stores from the Context.
	storeKey sdk.StoreKey

	// The codec for binary encoding/decoding.
	cdc codec.Codec
}

// NewKeeper returns a token keeper
func NewKeeper(
	cdc codec.Codec,
	key sdk.StoreKey,
	ak token.AccountKeeper,
	ck token.ClassKeeper,
) Keeper {
	return Keeper{
		accountKeeper: ak,
		classKeeper:   ck,
		storeKey:      key,
		cdc:           cdc,
	}
}
