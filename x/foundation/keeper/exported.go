package keeper

import (
	"github.com/line/lbm-sdk/baseapp"
	"github.com/line/lbm-sdk/codec"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/x/foundation"
	"github.com/line/lbm-sdk/x/foundation/keeper/internal"
	govtypes "github.com/line/lbm-sdk/x/gov/types"
)

type Keeper interface {
	GetAuthority() string
	Accept(ctx sdk.Context, grantee sdk.AccAddress, msg sdk.Msg) error

	InitGenesis(ctx sdk.Context, gs *foundation.GenesisState) error
	ExportGenesis(ctx sdk.Context) *foundation.GenesisState
}

type keeper struct {
	impl internal.Keeper
}

func NewKeeper(
	cdc codec.Codec,
	key sdk.StoreKey,
	router *baseapp.MsgServiceRouter,
	authKeeper foundation.AuthKeeper,
	bankKeeper foundation.BankKeeper,
	feeCollectorName string,
	config foundation.Config,
	authority string,
) Keeper {
	return &keeper{
		impl: internal.NewKeeper(
			cdc,
			key,
			router,
			authKeeper,
			bankKeeper,
			feeCollectorName,
			config,
			authority,
		),
	}
}

// GetAuthority returns the x/foundation module's authority.
func (k keeper) GetAuthority() string {
	return k.impl.GetAuthority()
}

func (k keeper) Accept(ctx sdk.Context, grantee sdk.AccAddress, msg sdk.Msg) error {
	return k.impl.Accept(ctx, grantee, msg)
}

func (k keeper) InitGenesis(ctx sdk.Context, gs *foundation.GenesisState) error {
	return k.impl.InitGenesis(ctx, gs)
}

func (k keeper) ExportGenesis(ctx sdk.Context) *foundation.GenesisState {
	return k.impl.ExportGenesis(ctx)
}

func NewMsgServer(k Keeper) foundation.MsgServer {
	impl := k.(*keeper).impl
	return internal.NewMsgServer(impl)
}

func NewQueryServer(k Keeper) foundation.QueryServer {
	impl := k.(*keeper).impl
	return internal.NewQueryServer(impl)
}

func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	impl := k.(*keeper).impl
	internal.RegisterInvariants(ir, impl)
}

func BeginBlocker(ctx sdk.Context, k Keeper) {
	impl := k.(*keeper).impl
	internal.BeginBlocker(ctx, impl)
}

func EndBlocker(ctx sdk.Context, k Keeper) {
	impl := k.(*keeper).impl
	internal.EndBlocker(ctx, impl)
}

func NewFoundationProposalsHandler(k Keeper) govtypes.Handler {
	impl := k.(*keeper).impl
	return internal.NewFoundationProposalsHandler(impl)
}
