package simulation

import (
	"math/rand"

	"github.com/line/lbm-sdk/baseapp"
	"github.com/line/lbm-sdk/codec"
	cdctypes "github.com/line/lbm-sdk/codec/types"
	"github.com/line/lbm-sdk/simapp/helpers"
	simappparams "github.com/line/lbm-sdk/simapp/params"
	sdk "github.com/line/lbm-sdk/types"
	simtypes "github.com/line/lbm-sdk/types/simulation"
	"github.com/line/lbm-sdk/x/authz"

	sdkerrors "github.com/line/lbm-sdk/types/errors"
	"github.com/line/lbm-sdk/x/authz/keeper"

	banktype "github.com/line/lbm-sdk/x/bank/types"
	"github.com/line/lbm-sdk/x/simulation"
)

// authz message types
var (
	TypeMsgGrant  = sdk.MsgTypeURL(&authz.MsgGrant{})
	TypeMsgRevoke = sdk.MsgTypeURL(&authz.MsgRevoke{})
	TypeMsgExec   = sdk.MsgTypeURL(&authz.MsgExec{})
)

// nolint:gosec
// Simulation operation weights constants
//
//nolint:gosec
const (
	OpWeightMsgGrant = "op_weight_msg_grant"
	OpWeightRevoke   = "op_weight_msg_revoke"
	OpWeightExec     = "op_weight_msg_execute"
)

// authz operations weights
const (
	WeightGrant  = 100
	WeightRevoke = 90
	WeightExec   = 90
)

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams simtypes.AppParams, cdc codec.JSONCodec, ak authz.AccountKeeper, bk authz.BankKeeper, k keeper.Keeper, appCdc cdctypes.AnyUnpacker,
) simulation.WeightedOperations {
	var (
		weightMsgGrant int
		weightRevoke   int
		weightExec     int
	)

	appParams.GetOrGenerate(cdc, OpWeightMsgGrant, &weightMsgGrant, nil,
		func(_ *rand.Rand) {
			weightMsgGrant = WeightGrant
		},
	)

	appParams.GetOrGenerate(cdc, OpWeightRevoke, &weightRevoke, nil,
		func(_ *rand.Rand) {
			weightRevoke = WeightRevoke
		},
	)

	appParams.GetOrGenerate(cdc, OpWeightExec, &weightExec, nil,
		func(_ *rand.Rand) {
			weightExec = WeightExec
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgGrant,
			SimulateMsgGrant(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightRevoke,
			SimulateMsgRevoke(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightExec,
			SimulateMsgExec(ak, bk, k, appCdc),
		),
	}
}

// SimulateMsgGrant generates a MsgGrant with random values.
func SimulateMsgGrant(ak authz.AccountKeeper, bk authz.BankKeeper, _ keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		granter, _ := simtypes.RandomAcc(r, accs)
		grantee, _ := simtypes.RandomAcc(r, accs)

		if granter.Address.Equals(grantee.Address) {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgGrant, "granter and grantee are same"), nil, nil
		}

		granterAcc := ak.GetAccount(ctx, granter.Address)
		spendableCoins := bk.SpendableCoins(ctx, granter.Address)
		fees, err := simtypes.RandomFees(r, ctx, spendableCoins)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgGrant, err.Error()), nil, err
		}

		spendLimit := spendableCoins.Sub(fees)
		if spendLimit == nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgGrant, "spend limit is nil"), nil, nil
		}

		expiration := ctx.BlockTime().AddDate(1, 0, 0)
		msg, err := authz.NewMsgGrant(granter.Address, grantee.Address, generateRandomAuthorization(r, spendLimit), expiration)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgGrant, err.Error()), nil, err
		}
		txCfg := simappparams.MakeTestEncodingConfig().TxConfig
		tx, err := helpers.GenSignedMockTx(
			r,
			txCfg,
			[]sdk.Msg{msg},
			fees,
			helpers.DefaultGenTxGas,
			chainID,
			[]uint64{granterAcc.GetAccountNumber()},
			[]uint64{granterAcc.GetSequence()},
			granter.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgGrant, "unable to generate mock tx"), nil, err
		}

		_, _, err = app.Deliver(txCfg.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, sdk.MsgTypeURL(msg), "unable to deliver tx"), nil, err
		}
		return simtypes.NewOperationMsg(msg, true, "", nil), nil, err
	}
}

func generateRandomAuthorization(r *rand.Rand, spendLimit sdk.Coins) authz.Authorization {
	authorizations := make([]authz.Authorization, 2)
	authorizations[0] = banktype.NewSendAuthorization(spendLimit)
	authorizations[1] = authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktype.MsgSend{}))

	return authorizations[r.Intn(len(authorizations))]
}

// SimulateMsgRevoke generates a MsgRevoke with random values.
func SimulateMsgRevoke(ak authz.AccountKeeper, bk authz.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var granterAddr, granteeAddr sdk.AccAddress
		var grant authz.Grant
		hasGrant := false

		k.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, g authz.Grant) bool {
			grant = g
			granterAddr = granter
			granteeAddr = grantee
			hasGrant = true
			return true
		})

		if !hasGrant {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "no grants"), nil, nil
		}

		granterAcc, ok := simtypes.FindAccount(accs, granterAddr)
		if !ok {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "account not found"), nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "account not found")
		}

		spendableCoins := bk.SpendableCoins(ctx, granterAddr)
		fees, err := simtypes.RandomFees(r, ctx, spendableCoins)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "fee error"), nil, err
		}

		a := grant.GetAuthorization()
		msg := authz.NewMsgRevoke(granterAddr, granteeAddr, a.MsgTypeURL())
		txCfg := simappparams.MakeTestEncodingConfig().TxConfig
		account := ak.GetAccount(ctx, granterAddr)
		tx, err := helpers.GenSignedMockTx(
			r,
			txCfg,
			[]sdk.Msg{&msg},
			fees,
			helpers.DefaultGenTxGas,
			chainID,
			[]uint64{account.GetAccountNumber()},
			[]uint64{account.GetSequence()},
			granterAcc.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, err.Error()), nil, err
		}

		_, _, err = app.Deliver(txCfg.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "unable to deliver tx"), nil, err
		}

		return simtypes.NewOperationMsg(&msg, true, "", nil), nil, nil
	}
}

// SimulateMsgExec generates a MsgExec with random values.
func SimulateMsgExec(ak authz.AccountKeeper, bk authz.BankKeeper, k keeper.Keeper, cdc cdctypes.AnyUnpacker) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		hasGrant := false
		var targetGrant authz.Grant
		var granterAddr sdk.AccAddress
		var granteeAddr sdk.AccAddress
		k.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
			targetGrant = grant
			granterAddr = granter
			granteeAddr = grantee
			hasGrant = true
			return true
		})

		if !hasGrant {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "no grant found"), nil, nil
		}

		if targetGrant.Expiration.Before(ctx.BlockHeader().Time) {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "grant expired"), nil, nil
		}

		grantee, ok := simtypes.FindAccount(accs, granteeAddr)
		if !ok {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "Account not found"), nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "grantee account not found")
		}

		if _, ok := simtypes.FindAccount(accs, granterAddr); !ok {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgRevoke, "Account not found"), nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "granter account not found")
		}

		granterspendableCoins := bk.SpendableCoins(ctx, granterAddr)
		coins := simtypes.RandSubsetCoins(r, granterspendableCoins)
		// if coins slice is empty, we can not create valid banktype.MsgSend
		if len(coins) == 0 {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "empty coins slice"), nil, nil
		}
		// Check send_enabled status of each sent coin denom
		if err := bk.IsSendEnabledCoins(ctx, coins...); err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, err.Error()), nil, nil
		}

		msg := []sdk.Msg{banktype.NewMsgSend(granterAddr, granteeAddr, coins)}
		authorization := targetGrant.GetAuthorization()

		sendAuth, ok := authorization.(*banktype.SendAuthorization)
		if !ok {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "not a send authorization"), nil, nil
		}

		_, err := sendAuth.Accept(ctx, msg[0])
		if err != nil {
			if sdkerrors.ErrInsufficientFunds.Is(err) {
				return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, err.Error()), nil, nil
			}
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, err.Error()), nil, err
		}

		msgExec := authz.NewMsgExec(granteeAddr, msg)
		granteeSpendableCoins := bk.SpendableCoins(ctx, granteeAddr)
		fees, err := simtypes.RandomFees(r, ctx, granteeSpendableCoins)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "fee error"), nil, err
		}

		txCfg := simappparams.MakeTestEncodingConfig().TxConfig
		granteeAcc := ak.GetAccount(ctx, granteeAddr)
		tx, err := helpers.GenSignedMockTx(
			r,
			txCfg,
			[]sdk.Msg{&msgExec},
			fees,
			helpers.DefaultGenTxGas,
			chainID,
			[]uint64{granteeAcc.GetAccountNumber()},
			[]uint64{granteeAcc.GetSequence()},
			grantee.PrivKey,
		)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, err.Error()), nil, err
		}

		_, _, err = app.Deliver(txCfg.TxEncoder(), tx)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, err.Error()), nil, err
		}

		err = msgExec.UnpackInterfaces(cdc)
		if err != nil {
			return simtypes.NoOpMsg(authz.ModuleName, TypeMsgExec, "unmarshal error"), nil, err
		}
		return simtypes.NewOperationMsg(&msgExec, true, "success", nil), nil, nil
	}
}
