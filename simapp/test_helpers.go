package simapp

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	ocabci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/log"
	octypes "github.com/line/ostracon/types"

	bam "github.com/line/lbm-sdk/baseapp"
	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/line/lbm-sdk/crypto/types"
	"github.com/line/lbm-sdk/simapp/helpers"
	"github.com/line/lbm-sdk/snapshots"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/types/errors"
	authtypes "github.com/line/lbm-sdk/x/auth/types"
	banktypes "github.com/line/lbm-sdk/x/bank/types"
	minttypes "github.com/line/lbm-sdk/x/mint/types"
)

// DefaultConsensusParams defines the default Tendermint consensus params used in
// SimApp testing.
var DefaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			octypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func setup(withGenesis bool, invCheckPeriod uint) (*SimApp, GenesisState) {
	randDir, _ := os.MkdirTemp(DefaultNodeHome, "")
	snapshotDir := filepath.Join(randDir, "data", "snapshots")
	snapshotDB := dbm.NewMemDB()

	snapshotStore, _ := snapshots.NewStore(snapshotDB, snapshotDir)
	baseAppOpts := []func(*bam.BaseApp){bam.SetSnapshotStore(snapshotStore), bam.SetSnapshotKeepRecent(2)}

	db := dbm.NewMemDB()
	encCdc := MakeTestEncodingConfig()
	app := NewSimApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, DefaultNodeHome, invCheckPeriod, encCdc, EmptyAppOptions{}, baseAppOpts...)
	if withGenesis {
		return app, NewDefaultGenesisState(encCdc.Marshaler)
	}
	return app, GenesisState{}
}

// Setup initializes a new SimApp. A Nop logger is set in SimApp.
func Setup(isCheckTx bool) *SimApp {
	app, genesisState := setup(!isCheckTx, 5)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		app.InitChain(
			abci.RequestInitChain{
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	return app
}

// SetupWithGenesisValSet initializes a new SimApp with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit (10^6) in the default token of the simapp from first genesis
// account. A Nop logger is set in SimApp.
//func SetupWithGenesisValSet(t *testing.T, valSet *octypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *SimApp {
//	app, genesisState := setup(true, 5)
//	// set genesis accounts
//	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
//	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)
//
//	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
//	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))
//
//	bondAmt := sdk.NewInt(1000000)
//
//	for _, val := range valSet.Validators {
//		pk, err := cryptocodec.FromOcPubKeyInterface(val.PubKey)
//		require.NoError(t, err)
//		pkAny, err := codectypes.NewAnyWithValue(pk)
//		require.NoError(t, err)
//		validator := stakingtypes.Validator{
//			OperatorAddress:   sdk.ValAddress(val.Address).String(),
//			ConsensusPubkey:   pkAny,
//			Jailed:            false,
//			Status:            stakingtypes.Bonded,
//			Tokens:            bondAmt,
//			DelegatorShares:   sdk.OneDec(),
//			Description:       stakingtypes.Description{},
//			UnbondingHeight:   int64(0),
//			UnbondingTime:     time.Unix(0, 0).UTC(),
//			Commission:        stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
//			MinSelfDelegation: sdk.ZeroInt(),
//		}
//		validators = append(validators, validator)
//		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))
//
//	}
//	// set validators and delegations
//	stakingGenesis := stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations)
//	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGenesis)
//
//	totalSupply := sdk.NewCoins()
//	for _, b := range balances {
//		// add genesis acc tokens and delegated tokens to total supply
//		totalSupply = totalSupply.Add(b.Coins.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))...)
//	}
//
//	// add bonded amount to bonded pool module account
//	balances = append(balances, banktypes.Balance{
//		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
//		Coins:   sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, bondAmt)},
//	})
//
//	// update total supply
//	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})
//	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)
//
//	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
//	require.NoError(t, err)
//
//	// init chain will set the validator set and initialize the genesis accounts
//	app.InitChain(
//		abci.RequestInitChain{
//			Validators:      []abci.ValidatorUpdate{},
//			ConsensusParams: DefaultConsensusParams,
//			AppStateBytes:   stateBytes,
//		},
//	)
//
//	// commit genesis changes
//	app.Commit()
//	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{
//		Height:             app.LastBlockHeight() + 1,
//		AppHash:            app.LastCommitID().Hash,
//		ValidatorsHash:     valSet.Hash(),
//		NextValidatorsHash: valSet.Hash(),
//	}})
//
//	return app
//}

// SetupWithGenesisAccounts initializes a new SimApp with the provided genesis
// accounts and possible balances.
func SetupWithGenesisAccounts(genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *SimApp {
	app, genesisState := setup(true, 0)
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		totalSupply = totalSupply.Add(b.Coins...)
	}

	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	app.InitChain(
		abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	app.Commit()
	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: app.LastBlockHeight() + 1}})

	return app
}

type GenerateAccountStrategy func(int) []sdk.AccAddress

// createRandomAccounts is a strategy used by addTestAddrs() in order to generated addresses in random order.
func createRandomAccounts(accNum int) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, accNum)
	for i := 0; i < accNum; i++ {
		pk := ed25519.GenPrivKey().PubKey()
		testAddrs[i] = sdk.AccAddress(pk.Address())
	}

	return testAddrs
}

// createIncrementalAccounts is a strategy used by addTestAddrs() in order to generated addresses in ascending order.
func createIncrementalAccounts(accNum int) []sdk.AccAddress {
	var addresses []sdk.AccAddress
	var buffer bytes.Buffer

	// start at 100 so we can make up to 999 test addresses with valid test addresses
	for i := 100; i < (accNum + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("A58856F0FD53BF058B4909A21AEC019107BA6") // base address string

		buffer.WriteString(numString) // adding on final two digits to make addresses unique
		res, _ := sdk.AccAddressFromHex(buffer.String())
		bech := res.String()
		addr, _ := TestAddr(buffer.String(), bech)

		addresses = append(addresses, addr)
		buffer.Reset()
	}

	return addresses
}

// AddTestAddrsFromPubKeys adds the addresses into the SimApp providing only the public keys.
func AddTestAddrsFromPubKeys(app *SimApp, ctx sdk.Context, pubKeys []cryptotypes.PubKey, accAmt sdk.Int) {
	initCoins := sdk.NewCoins(sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), accAmt))

	// fill all the addresses with some coins, set the loose pool tokens simultaneously
	for _, pk := range pubKeys {
		initAccountWithCoins(app, ctx, sdk.AccAddress(pk.Address()), initCoins)
	}
}

// SortAddresses - Sorts Addresses
func SortAddresses(addrs []sdk.AccAddress) {
	byteAddrs := make([][]byte, len(addrs))

	for i, addr := range addrs {
		byteAddrs[i] = addr.Bytes()
	}

	SortByteArrays(byteAddrs)

	for i, byteAddr := range byteAddrs {
		addrs[i] = sdk.AccAddress(byteAddr)
	}
}

// implement `Interface` in sort package.
type sortByteArrays [][]byte

func (b sortByteArrays) Len() int {
	return len(b)
}

func (b sortByteArrays) Less(i, j int) bool {
	// bytes package already implements Comparable for []byte.
	switch bytes.Compare(b[i], b[j]) {
	case -1:
		return true
	case 0, 1:
		return false
	default:
		panic("not fail-able with `bytes.Comparable` bounded [-1, 1].")
	}
}

func (b sortByteArrays) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

// SortByteArrays - sorts the provided byte array
func SortByteArrays(src [][]byte) [][]byte {
	sorted := sortByteArrays(src)
	sort.Sort(sorted)
	return sorted
}

// AddTestAddrs constructs and returns accNum amount of accounts with an
// initial balance of accAmt in random order
func AddTestAddrs(app *SimApp, ctx sdk.Context, accNum int, accAmt sdk.Int) []sdk.AccAddress {
	return addTestAddrs(app, ctx, accNum, accAmt, createRandomAccounts)
}

// AddTestAddrs constructs and returns accNum amount of accounts with an
// initial balance of accAmt in random order
func AddTestAddrsIncremental(app *SimApp, ctx sdk.Context, accNum int, accAmt sdk.Int) []sdk.AccAddress {
	return addTestAddrs(app, ctx, accNum, accAmt, createIncrementalAccounts)
}

func addTestAddrs(app *SimApp, ctx sdk.Context, accNum int, accAmt sdk.Int, strategy GenerateAccountStrategy) []sdk.AccAddress {
	testAddrs := strategy(accNum)

	initCoins := sdk.NewCoins(sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), accAmt))

	for _, addr := range testAddrs {
		initAccountWithCoins(app, ctx, addr, initCoins)
	}

	SortAddresses(testAddrs)
	return testAddrs
}

func initAccountWithCoins(app *SimApp, ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) {
	err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins)
	if err != nil {
		panic(err)
	}

	err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, coins)
	if err != nil {
		panic(err)
	}
}

// ConvertAddrsToValAddrs converts the provided addresses to ValAddress.
func ConvertAddrsToValAddrs(addrs []sdk.AccAddress) []sdk.ValAddress {
	valAddrs := make([]sdk.ValAddress, len(addrs))

	for i, addr := range addrs {
		valAddrs[i] = sdk.ValAddress(addr)
	}

	return valAddrs
}

func TestAddr(addr string, bech string) (sdk.AccAddress, error) {
	res, err := sdk.AccAddressFromHex(addr)
	if err != nil {
		return nil, err
	}
	bechexpected := res.String()
	if bech != bechexpected {
		return nil, fmt.Errorf("bech encoding doesn't match reference")
	}

	bechres, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(bechres, res) {
		return nil, err
	}

	return res, nil
}

// CheckBalance checks the balance of an account.
func CheckBalance(t *testing.T, app *SimApp, addr sdk.AccAddress, balances sdk.Coins) {
	ctxCheck := app.BaseApp.NewContext(true, tmproto.Header{})
	require.True(t, balances.IsEqual(app.BankKeeper.GetAllBalances(ctxCheck, addr)))
}

// SignCheckDeliver checks a generated signed transaction and simulates a
// block commitment with the given transaction. A test assertion is made using
// the parameter 'expPass' against the result. A corresponding result is
// returned.
func SignCheckDeliver(
	t *testing.T, txCfg client.TxConfig, app *bam.BaseApp, header tmproto.Header, msgs []sdk.Msg,
	chainID string, accNums, accSeqs []uint64, expSimPass, expPass bool, priv ...cryptotypes.PrivKey,
) (sdk.GasInfo, *sdk.Result, error) {
	tx, err := helpers.GenSignedMockTx(
		rand.New(rand.NewSource(time.Now().UnixNano())),
		txCfg,
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
		helpers.DefaultGenTxGas,
		chainID,
		accNums,
		accSeqs,
		priv...,
	)
	require.NoError(t, err)
	txBytes, err := txCfg.TxEncoder()(tx)
	require.Nil(t, err)

	// Must simulate now as CheckTx doesn't run Msgs anymore
	_, res, err := app.Simulate(txBytes)

	if expSimPass {
		require.NoError(t, err)
		require.NotNil(t, res)
	} else {
		require.Error(t, err)
		require.Nil(t, res)
	}

	// Simulate a sending a transaction and committing a block and recheck
	app.BeginBlock(ocabci.RequestBeginBlock{Header: header})
	gInfo, res, err := app.Deliver(txCfg.TxEncoder(), tx)

	if expPass {
		require.NoError(t, err)
		require.NotNil(t, res)
	} else {
		require.Error(t, err)
		require.Nil(t, res)
	}

	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	app.BeginRecheckTx(ocabci.RequestBeginRecheckTx{Header: header})
	app.EndRecheckTx(ocabci.RequestEndRecheckTx{})

	return gInfo, res, err
}

// SignAndDeliver signs and delivers a transaction. No simulation occurs as the
// ibc testing package causes checkState and deliverState to diverge in block time.
func SignAndDeliver(
	t *testing.T, txCfg client.TxConfig, app *bam.BaseApp, header tmproto.Header, msgs []sdk.Msg,
	chainID string, accNums, accSeqs []uint64, expSimPass, expPass bool, priv ...cryptotypes.PrivKey,
) (sdk.GasInfo, *sdk.Result, error) {
	tx, err := helpers.GenTx(
		txCfg,
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
		2*helpers.DefaultGenTxGas,
		chainID,
		accNums,
		accSeqs,
		priv...,
	)
	require.NoError(t, err)

	// Simulate a sending a transaction and committing a block
	app.BeginBlock(ocabci.RequestBeginBlock{Header: header})
	gInfo, res, err := app.Deliver(txCfg.TxEncoder(), tx)

	if expPass {
		require.NoError(t, err)
		require.NotNil(t, res)
	} else {
		require.Error(t, err)
		require.Nil(t, res)
	}

	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	return gInfo, res, err
}

// GenSequenceOfTxs generates a set of signed transactions of messages, such
// that they differ only by having the sequence numbers incremented between
// every transaction.
func GenSequenceOfTxs(txGen client.TxConfig, msgs []sdk.Msg, accNums []uint64, initSeqNums []uint64, numToGenerate int, priv ...cryptotypes.PrivKey) ([]sdk.Tx, error) {
	txs := make([]sdk.Tx, numToGenerate)
	var err error
	for i := 0; i < numToGenerate; i++ {
		txs[i], err = helpers.GenSignedMockTx(
			rand.New(rand.NewSource(time.Now().UnixNano())),
			txGen,
			msgs,
			sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
			helpers.DefaultGenTxGas,
			"",
			accNums,
			initSeqNums,
			priv...,
		)
		if err != nil {
			break
		}
		incrementAllSequenceNumbers(initSeqNums)
	}

	return txs, err
}

func incrementAllSequenceNumbers(initSeqNums []uint64) {
	for i := 0; i < len(initSeqNums); i++ {
		initSeqNums[i]++
	}
}

// CreateTestPubKeys returns a total of numPubKeys public keys in ascending order.
func CreateTestPubKeys(numPubKeys int) []cryptotypes.PubKey {
	var publicKeys []cryptotypes.PubKey
	var buffer bytes.Buffer

	// start at 10 to avoid changing 1 to 01, 2 to 02, etc
	for i := 100; i < (numPubKeys + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AF") // base pubkey string
		buffer.WriteString(numString)                                                       // adding on final two digits to make pubkeys unique
		publicKeys = append(publicKeys, NewPubKeyFromHex(buffer.String()))
		buffer.Reset()
	}

	return publicKeys
}

// NewPubKeyFromHex returns a PubKey from a hex string.
func NewPubKeyFromHex(pk string) (res cryptotypes.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	if len(pkBytes) != ed25519.PubKeySize {
		panic(errors.Wrap(errors.ErrInvalidPubKey, "invalid pubkey size"))
	}
	return &ed25519.PubKey{Key: pkBytes}
}

// EmptyAppOptions is a stub implementing AppOptions
type EmptyAppOptions struct{}

// Get implements AppOptions
func (ao EmptyAppOptions) Get(o string) interface{} {
	return nil
}

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
//
// TODO: Instead of using the mint module account, which has the
// permission of minting, create a "faucet" account. (@fdymylja)
func FundAccount(app *SimApp, ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address. This should be used for testing
// purposes only!
//
// TODO: Instead of using the mint module account, which has the
// permission of minting, create a "faucet" account. (@fdymylja)
func FundModuleAccount(app *SimApp, ctx sdk.Context, recipientMod string, amounts sdk.Coins) error {
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return app.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, recipientMod, amounts)
}
