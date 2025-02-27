package baseapp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	ocabci "github.com/line/ostracon/abci/types"

	sdk "github.com/line/lbm-sdk/types"
)

func TestGetBlockRentionHeight(t *testing.T) {
	logger := defaultLogger()
	db := dbm.NewMemDB()
	name := t.Name()

	testCases := map[string]struct {
		bapp         *BaseApp
		maxAgeBlocks int64
		commitHeight int64
		expected     int64
	}{
		"defaults": {
			bapp:         NewBaseApp(name, logger, db, nil),
			maxAgeBlocks: 0,
			commitHeight: 499000,
			expected:     0,
		},
		"pruning unbonding time only": {
			bapp:         NewBaseApp(name, logger, db, nil, SetMinRetainBlocks(1)),
			maxAgeBlocks: 362880,
			commitHeight: 499000,
			expected:     136120,
		},
		"pruning iavl snapshot only": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetPruning(sdk.PruningOptions{KeepEvery: 10000}),
				SetMinRetainBlocks(1),
			),
			maxAgeBlocks: 0,
			commitHeight: 499000,
			expected:     490000,
		},
		"pruning state sync snapshot only": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetSnapshotInterval(50000),
				SetSnapshotKeepRecent(3),
				SetMinRetainBlocks(1),
			),
			maxAgeBlocks: 0,
			commitHeight: 499000,
			expected:     349000,
		},
		"pruning min retention only": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetMinRetainBlocks(400000),
			),
			maxAgeBlocks: 0,
			commitHeight: 499000,
			expected:     99000,
		},
		"pruning all conditions": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetPruning(sdk.PruningOptions{KeepEvery: 10000}),
				SetMinRetainBlocks(400000),
				SetSnapshotInterval(50000), SetSnapshotKeepRecent(3),
			),
			maxAgeBlocks: 362880,
			commitHeight: 499000,
			expected:     99000,
		},
		"no pruning due to no persisted state": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetPruning(sdk.PruningOptions{KeepEvery: 10000}),
				SetMinRetainBlocks(400000),
				SetSnapshotInterval(50000), SetSnapshotKeepRecent(3),
			),
			maxAgeBlocks: 362880,
			commitHeight: 10000,
			expected:     0,
		},
		"disable pruning": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetPruning(sdk.PruningOptions{KeepEvery: 10000}),
				SetMinRetainBlocks(0),
				SetSnapshotInterval(50000), SetSnapshotKeepRecent(3),
			),
			maxAgeBlocks: 362880,
			commitHeight: 499000,
			expected:     0,
		},
		"iavl disable fast node": {
			bapp: NewBaseApp(
				name, logger, db, nil,
				SetIAVLDisableFastNode(true),
			),
			maxAgeBlocks: 0,
			commitHeight: 499000,
			expected:     0,
		},
	}

	for name, tc := range testCases {
		tc := tc

		tc.bapp.SetParamStore(&paramStore{db: dbm.NewMemDB()})
		tc.bapp.InitChain(abci.RequestInitChain{
			ConsensusParams: &abci.ConsensusParams{
				Evidence: &tmproto.EvidenceParams{
					MaxAgeNumBlocks: tc.maxAgeBlocks,
				},
			},
		})

		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.bapp.GetBlockRetentionHeight(tc.commitHeight))
		})
	}
}

// Test and ensure that invalid block heights always cause errors.
// See issues:
// - https://github.com/cosmos/cosmos-sdk/issues/11220
// - https://github.com/cosmos/cosmos-sdk/issues/7662
func TestBaseAppCreateQueryContext(t *testing.T) {
	t.Parallel()

	logger := defaultLogger()
	db := dbm.NewMemDB()
	name := t.Name()
	app := NewBaseApp(name, logger, db, nil)
	app.init()

	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
	app.Commit()

	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: 2}})
	app.Commit()

	testCases := []struct {
		name   string
		height int64
		prove  bool
		expErr bool
	}{
		{"valid height", 2, true, false},
		{"future height", 10, true, true},
		{"negative height, prove=true", -1, true, true},
		{"negative height, prove=false", -1, false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.createQueryContext(tc.height, tc.prove)
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test and ensure that consensus params has been updated.
// See:
// - https://github.com/line/lbm-sdk/pull/673
func TestBaseAppBeginBlockConsensusParams(t *testing.T) {
	t.Parallel()

	logger := defaultLogger()
	db := dbm.NewMemDB()
	name := t.Name()
	app := NewBaseApp(name, logger, db, nil)
	app.SetParamStore(&paramStore{db: dbm.NewMemDB()})
	app.InitChain(abci.RequestInitChain{
		ConsensusParams: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas: -1,
			},
		},
	})
	app.init()

	// set block params
	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
	ctx := app.deliverState.ctx
	maxGas := int64(123456789)
	app.paramStore.Set(ctx, ParamStoreKeyBlockParams,
		&abci.BlockParams{
			MaxGas: maxGas,
		})
	app.Commit()

	// confirm consensus params updated into the context
	app.BeginBlock(ocabci.RequestBeginBlock{Header: tmproto.Header{Height: 2}})
	newCtx := app.getContextForTx(app.checkState, []byte{})
	require.Equal(t, maxGas, newCtx.ConsensusParams().Block.MaxGas)
}

type paramStore struct {
	db *dbm.MemDB
}

func (ps *paramStore) Set(_ sdk.Context, key []byte, value interface{}) {
	bz, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	ps.db.Set(key, bz)
}

func (ps *paramStore) Has(_ sdk.Context, key []byte) bool {
	ok, err := ps.db.Has(key)
	if err != nil {
		panic(err)
	}

	return ok
}

func (ps *paramStore) Get(_ sdk.Context, key []byte, ptr interface{}) {
	bz, err := ps.db.Get(key)
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return
	}

	if err := json.Unmarshal(bz, ptr); err != nil {
		panic(err)
	}
}
