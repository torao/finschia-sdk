package keeper_test

import (
	"testing"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/foundation"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/stakingplus"
	"github.com/cosmos/cosmos-sdk/x/stakingplus/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

type KeeperTestSuite struct {
	suite.Suite
	ctx  sdk.Context

	app *simapp.SimApp
	keeper stakingkeeper.Keeper
	msgServer stakingtypes.MsgServer

	stranger sdk.AccAddress
	grantee sdk.AccAddress

	balance sdk.Int
}

func (s *KeeperTestSuite) SetupTest() {
	checkTx := false
	s.app = simapp.Setup(checkTx)
	s.ctx = s.app.BaseApp.NewContext(checkTx, tmproto.Header{})
	s.keeper = s.app.StakingKeeper

	s.msgServer = keeper.NewMsgServerImpl(s.keeper, s.app.FoundationKeeper)

	createAddress := func() sdk.AccAddress {
		return sdk.BytesToAccAddress(secp256k1.GenPrivKey().PubKey().Address())
	}

	s.stranger = createAddress()
	s.grantee = createAddress()

	s.balance = sdk.NewInt(1000000)
	holders := []sdk.AccAddress{
		s.stranger,
		s.grantee,
	}
	for _, holder := range holders {
		amount := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, s.balance))

		// using minttypes here introduces dependency on x/mint
		// the work around would be registering a new module account on this suite
		// because x/bank already has dependency on x/mint, and we must have dependency
		// on x/bank, it's OK to use x/mint here.
		minterName := minttypes.ModuleName
		err := s.app.BankKeeper.MintCoins(s.ctx, minterName, amount)
		s.Require().NoError(err)

		minter := s.app.AccountKeeper.GetModuleAccount(s.ctx, minterName).GetAddress()
		err = s.app.BankKeeper.SendCoins(s.ctx, minter, holder, amount)
		s.Require().NoError(err)
	}

	// allow Msg/CreateValidator
	s.app.FoundationKeeper.SetParams(s.ctx, &foundation.Params{
		Enabled: true,
		FoundationTax: sdk.ZeroDec(),
	})
	err := s.app.FoundationKeeper.Grant(s.ctx, govtypes.ModuleName, s.grantee, &stakingplus.CreateValidatorAuthorization{
		ValidatorAddress: s.grantee.ToValAddress().String(),
	})
	s.Require().NoError(err)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
