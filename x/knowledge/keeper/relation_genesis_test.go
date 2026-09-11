package keeper

import (
	"bytes"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func setupRelationGenesis(t *testing.T, genesis *types.GenesisState) handoffFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	fault := new(handoffFault)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := NewKeeper(handoffFaultService{runtime.NewKVStoreService(key), fault}, cdc, "authority", nil, nil)
	if genesis != nil {
		require.NoError(t, k.InitGenesis(ctx, genesis))
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	return handoffFixture{k: k, ctx: ctx, keys: []*storetypes.KVStoreKey{key}, kFault: fault}
}

func relationGenesisFacts() []*types.Fact {
	return []*types.Fact{
		{Id: "ordinary-a", Domain: "physics", Content: "observation A", Status: types.FactStatus_FACT_STATUS_ACTIVE, Submitter: "author-a"},
		{Id: "ordinary-b", Domain: "physics", Content: "observation B", Status: types.FactStatus_FACT_STATUS_VERIFIED, Submitter: "author-b"},
	}
}

func fullGenesisRelation() *types.FactRelation {
	return &types.FactRelation{
		SourceFactId: "ordinary-a", TargetFactId: "ordinary-b",
		Relation: types.RelationType_RELATION_TYPE_REQUIRES, CreatedAtBlock: 97,
		Creator: "relation-author", Inference: types.InferenceType_INFERENCE_TYPE_EMPIRICAL,
		InferenceStrengthBps: 8123, MethodId: "M-EXPERIMENT",
	}
}

func relationStoreSnapshot(t *testing.T, f handoffFixture) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	require.NoError(t, walkFactRelationGenesis(f.k.storeService.OpenKVStore(f.ctx), func(key, value []byte) error {
		result[string(key)] = bytes.Clone(value)
		return nil
	}))
	return result
}

func TestFactRelationGenesisCodecRoundTripPreservesCanonicalGraph(t *testing.T) {
	f := setupRelationGenesis(t, types.DefaultGenesis())
	facts := relationGenesisFacts()
	// An embedded proposal is Fact payload, never another canonical edge.
	facts[0].OutgoingRelations = []*types.FactRelation{{SourceFactId: "ordinary-a", TargetFactId: "embedded-only", Creator: "embedded-author"}}
	for _, fact := range facts {
		require.NoError(t, f.k.SetFact(f.ctx, fact))
	}
	full := fullGenesisRelation()
	reverse := &types.FactRelation{SourceFactId: full.TargetFactId, TargetFactId: full.SourceFactId, Relation: types.RelationType_RELATION_TYPE_CITES}
	self := &types.FactRelation{SourceFactId: full.SourceFactId, TargetFactId: full.SourceFactId}
	for _, rel := range []*types.FactRelation{full, reverse, self} {
		require.NoError(t, f.k.SetFactRelation(f.ctx, rel))
	}
	// Change one doctrine pair and remove another. Import must not reseed them
	// over the exported inventory merely because they were once defaults.
	seeded, err := f.k.ExportFactRelationGenesis(f.ctx)
	require.NoError(t, err)
	var doctrine []*types.FactRelation
	for _, rel := range seeded.Relations {
		if rel.SourceFactId != "ordinary-a" && rel.SourceFactId != "ordinary-b" {
			doctrine = append(doctrine, rel)
		}
	}
	require.GreaterOrEqual(t, len(doctrine), 2)
	changed := proto.Clone(doctrine[0]).(*types.FactRelation)
	changed.Relation = types.RelationType_RELATION_TYPE_CONTRADICTS
	changed.Creator = "later-relation-author"
	changed.CreatedAtBlock = 99
	changed.MethodId = "M-CORRECTION"
	require.NoError(t, f.k.SetFactRelation(f.ctx, changed))
	removed := doctrine[1]
	rawStore := f.ctx.KVStore(f.keys[0])
	rawStore.Delete(types.FactRelationKey(removed.SourceFactId, removed.TargetFactId))
	rawStore.Delete(types.FactRelationReverseKey(removed.TargetFactId, removed.SourceFactId))
	before := relationStoreSnapshot(t, f)
	beforeHistory, err := f.k.GetStatusHistoryChecked(f.ctx, facts[0].Id)
	require.NoError(t, err)
	require.NotEmpty(t, beforeHistory)
	selector := &types.ToKSelector{Variant: &types.ToKSelector_AncestorCone{AncestorCone: &types.AncestorConeSelector{LeafFactId: "ordinary-a", MaxDepth: 3, MaxPaths: 8}}}
	beforeBundle, err := f.k.AssembleToKBundle(f.ctx, selector, 100)
	require.NoError(t, err)
	exported := f.k.ExportGenesis(f.ctx)
	require.NotNil(t, exported.FactRelationState)
	last := ""
	for _, rel := range exported.FactRelationState.Relations {
		key := string(types.FactRelationKey(rel.SourceFactId, rel.TargetFactId))
		require.Greater(t, key, last)
		last = key
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	for _, format := range []string{"sdk-codec", "protobuf-json"} {
		t.Run(format, func(t *testing.T) {
			var data []byte
			var err error
			var imported types.GenesisState
			if format == "sdk-codec" {
				data, err = cdc.MarshalJSON(exported)
				require.NoError(t, err)
				err = cdc.UnmarshalJSON(data, &imported)
			} else {
				data, err = protojson.Marshal(exported)
				require.NoError(t, err)
				err = protojson.Unmarshal(data, &imported)
			}
			require.NoError(t, err)
			fresh := setupRelationGenesis(t, &imported)
			require.Equal(t, before, relationStoreSnapshot(t, fresh))
			for _, want := range facts {
				got, found := fresh.k.GetFact(fresh.ctx, want.Id)
				require.True(t, found)
				require.True(t, proto.Equal(want, got))
			}
			gotHistory, err := fresh.k.GetStatusHistoryChecked(fresh.ctx, facts[0].Id)
			require.NoError(t, err)
			require.Len(t, gotHistory, len(beforeHistory))
			for i := range beforeHistory {
				require.True(t, proto.Equal(beforeHistory[i], gotHistory[i]))
			}
			bundle, err := fresh.k.AssembleToKBundle(fresh.ctx, selector, 100)
			require.NoError(t, err)
			require.Equal(t, beforeBundle.SnapshotRoot, bundle.SnapshotRoot)
			again := fresh.k.ExportGenesis(fresh.ctx)
			require.True(t, proto.Equal(exported.FactRelationState, again.FactRelationState))
		})
	}
}

func TestFactRelationGenesisAbsentAndExplicitEmpty(t *testing.T) {
	legacy := types.DefaultGenesis()
	require.Nil(t, legacy.FactRelationState)
	f := setupRelationGenesis(t, legacy)
	require.NotEmpty(t, relationStoreSnapshot(t, f))
	for _, encoded := range []string{`{}`, `{"fact_relation_state":null}`, `{"factRelationState":{}}`, `{"fact_relation_state":{"relations":[]}}`} {
		t.Run(encoded, func(t *testing.T) {
			gs := types.DefaultGenesis()
			cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			require.NoError(t, cdc.UnmarshalJSON([]byte(encoded), gs))
			fresh := setupRelationGenesis(t, gs)
			state, err := fresh.k.ExportFactRelationGenesis(fresh.ctx)
			require.NoError(t, err)
			require.NotNil(t, state)
			if gs.FactRelationState == nil {
				require.NotEmpty(t, state.Relations)
			} else {
				require.Empty(t, state.Relations)
				require.Empty(t, relationStoreSnapshot(t, fresh))
				data, err := cdc.MarshalJSON(fresh.k.ExportGenesis(fresh.ctx))
				require.NoError(t, err)
				var restored types.GenesisState
				require.NoError(t, cdc.UnmarshalJSON(data, &restored))
				require.NotNil(t, restored.FactRelationState)
				again := setupRelationGenesis(t, &restored)
				require.Empty(t, relationStoreSnapshot(t, again))
			}
		})
	}
}

func TestFactRelationGenesisRejectsInvalidBeforeAnyWrites(t *testing.T) {
	f := setupRelationGenesis(t, nil)
	gs := types.DefaultGenesis()
	gs.Facts = relationGenesisFacts()
	gs.FactRelationState = &types.FactRelationGenesis{Relations: []*types.FactRelation{fullGenesisRelation(), fullGenesisRelation()}}
	before := f.snapshot(t)
	require.Error(t, f.k.InitGenesis(f.ctx, gs))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
}

func TestFactRelationGenesisReplacementFailuresAreAtomic(t *testing.T) {
	for _, op := range []string{"set", "delete"} {
		for _, prefix := range [][]byte{types.FactRelationPrefix, types.FactRelationReversePrefix} {
			for _, after := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%x/after=%v", op, prefix, after), func(t *testing.T) {
					f := setupRelationGenesis(t, types.DefaultGenesis())
					gs := f.k.ExportGenesis(f.ctx)
					gs.Facts = append(gs.Facts, relationGenesisFacts()...)
					gs.FactRelationState = &types.FactRelationGenesis{Relations: []*types.FactRelation{fullGenesisRelation()}}
					for _, fact := range relationGenesisFacts() {
						require.NoError(t, f.k.SetFact(f.ctx, fact))
					}
					f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
					before := f.snapshot(t)
					*f.kFault = handoffFault{op, prefix, after}
					require.Error(t, f.k.ImportFactRelationGenesis(f.ctx, gs))
					require.Equal(t, before, f.snapshot(t))
					require.Empty(t, f.ctx.EventManager().Events())
					*f.kFault = handoffFault{}
					require.NoError(t, f.k.ImportFactRelationGenesis(f.ctx, gs))
					state, err := f.k.ExportFactRelationGenesis(f.ctx)
					require.NoError(t, err)
					require.True(t, proto.Equal(gs.FactRelationState, state))
				})
			}
		}
	}
}

func TestFactRelationGenesisCorruptionRefusesWholeExport(t *testing.T) {
	for _, name := range []string{"malformed", "unknown", "duplicate", "overflow", "wrong-wire", "wrong-key", "missing-reverse", "orphan-reverse", "unequal-reverse", "missing-fact", "malformed-fact", "unknown-enum"} {
		t.Run(name, func(t *testing.T) {
			f := setupRelationGenesis(t, types.DefaultGenesis())
			for _, fact := range relationGenesisFacts() {
				require.NoError(t, f.k.SetFact(f.ctx, fact))
			}
			rel := fullGenesisRelation()
			require.NoError(t, f.k.SetFactRelation(f.ctx, rel))
			st := f.ctx.KVStore(f.keys[0])
			fk, rk := types.FactRelationKey(rel.SourceFactId, rel.TargetFactId), types.FactRelationReverseKey(rel.TargetFactId, rel.SourceFactId)
			original := bytes.Clone(st.Get(fk))
			switch name {
			case "malformed":
				st.Set(fk, []byte{0xff})
			case "unknown", "duplicate", "overflow", "wrong-wire":
				var extra []byte
				switch name {
				case "unknown":
					extra = protowire.AppendVarint(protowire.AppendTag(nil, 100, protowire.VarintType), 1)
				case "duplicate":
					extra = protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), rel.SourceFactId)
				case "overflow":
					extra = protowire.AppendVarint(protowire.AppendTag(nil, 3, protowire.VarintType), (1<<32)+uint64(rel.Relation))
				case "wrong-wire":
					extra = protowire.AppendString(protowire.AppendTag(nil, 3, protowire.BytesType), "1")
				}
				st.Set(fk, append(original, extra...))
				st.Set(rk, append(bytes.Clone(original), extra...))
			case "wrong-key":
				st.Delete(fk)
				st.Set(types.FactRelationKey("different", rel.TargetFactId), original)
			case "missing-reverse":
				st.Delete(rk)
			case "orphan-reverse":
				st.Delete(fk)
			case "unequal-reverse":
				rel.Creator = "different-author"
				value, err := marshalOpts.Marshal(rel)
				require.NoError(t, err)
				st.Set(rk, value)
			case "missing-fact":
				st.Delete(types.FactKey(rel.SourceFactId))
			case "malformed-fact":
				st.Set(types.FactKey(rel.SourceFactId), []byte{0xff})
			case "unknown-enum":
				rel.Relation = 999
				require.NoError(t, f.k.SetFactRelation(f.ctx, rel))
			}
			before := f.snapshot(t)
			state, err := f.k.ExportFactRelationGenesis(f.ctx)
			require.Error(t, err)
			require.Nil(t, state)
			require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
			require.Equal(t, before, f.snapshot(t))
		})
	}
}

func TestFactRelationGenesisStoreErrorsRefuseExportAndReplacement(t *testing.T) {
	for _, mode := range []string{"error", "close"} {
		t.Run(mode, func(t *testing.T) {
			f := setupRelationGenesis(t, types.DefaultGenesis())
			gs := f.k.ExportGenesis(f.ctx)
			before := f.snapshot(t)
			bad := f.k
			bad.storeService = historyIteratorFaultService{bad.storeService, mode}
			state, err := bad.ExportFactRelationGenesis(f.ctx)
			require.Error(t, err)
			require.Nil(t, state)
			require.Error(t, bad.ImportFactRelationGenesis(f.ctx, gs))
			require.Equal(t, before, f.snapshot(t))
		})
	}
	f := setupRelationGenesis(t, types.DefaultGenesis())
	before := f.snapshot(t)
	*f.kFault = handoffFault{"get", types.FactKeyPrefix, false}
	state, err := f.k.ExportFactRelationGenesis(f.ctx)
	require.Error(t, err)
	require.Nil(t, state)
	require.Equal(t, before, f.snapshot(t))
}

// A generated iterator exercises the real shared accounting without allocating
// 100,001 database rows. The byte fixture reuses one payload across three rows.
type relationBudgetStore struct {
	corestore.KVStore
	rows  int
	value []byte
}

func (s relationBudgetStore) Iterator(start, _ []byte) (corestore.Iterator, error) {
	rows := 0
	if bytes.Equal(start, types.FactRelationPrefix) {
		rows = s.rows
	}
	return &relationBudgetIterator{rows: rows, value: s.value}, nil
}

type relationBudgetIterator struct {
	corestore.Iterator
	rows, cursor int
	value        []byte
}

func (i *relationBudgetIterator) Valid() bool   { return i.cursor < i.rows }
func (i *relationBudgetIterator) Next()         { i.cursor++ }
func (i *relationBudgetIterator) Key() []byte   { return []byte{0x30, 'a', '/', 'b'} }
func (i *relationBudgetIterator) Value() []byte { return i.value }
func (i *relationBudgetIterator) Error() error  { return nil }
func (i *relationBudgetIterator) Close() error  { return nil }

func TestFactRelationGenesisSharedInventoryBounds(t *testing.T) {
	visit := func(_, _ []byte) error { return nil }
	require.NoError(t, walkFactRelationGenesis(relationBudgetStore{rows: types.MaxFactRelationGenesisEntries}, visit))
	require.Error(t, walkFactRelationGenesis(relationBudgetStore{rows: types.MaxFactRelationGenesisEntries + 1}, visit))
	large := make([]byte, types.MaxFactRelationGenesisBytes/2)
	require.NoError(t, walkFactRelationGenesis(relationBudgetStore{rows: 1, value: large}, visit))
	require.Error(t, walkFactRelationGenesis(relationBudgetStore{rows: 2, value: large}, visit))
}
