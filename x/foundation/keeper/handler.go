package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/foundation"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func NewProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		if !k.GetEnabled(ctx) {
			return nil
		}

		switch c := content.(type) {
		case *foundation.UpdateFoundationParamsProposal:
			return k.handleUpdateFoundationParamsProposal(ctx, c)
		case *foundation.UpdateValidatorAuthsProposal:
			return k.handleUpdateValidatorAuthsProposal(ctx, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized foundation proposal content type: %T", c)
		}
	}
}
