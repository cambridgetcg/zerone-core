package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

func TestRecordHistorySetFactAtomicFailures(t *testing.T) {
	for _, tc := range []struct {
		op     string
		prefix []byte
	}{
		{"get", types.FactKeyPrefix}, {"get", types.StatusTransitionSeqKeyPrefix},
		{"set", types.StatusTransitionSeqKeyPrefix}, {"set", types.StatusTransitionKeyPrefix},
		{"set", types.FactKeyPrefix}, {"set", types.FactBySubmitterIndexPrefix}, {"set", types.DomainFactIndexPrefix},
		{"delete", types.FactBySubmitterIndexPrefix}, {"delete", types.DomainFactIndexPrefix},
	} {
		for _, after := range []bool{false, true} {
			if tc.op == "get" && after {
				continue
			}
			t.Run(fmt.Sprintf("%s/%x/after=%v", tc.op, tc.prefix, after), func(t *testing.T) {
				f := setupHandoff(t)
				require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
				fact, ok := f.k.GetFact(f.ctx, "fact")
				require.True(t, ok)
				fact.Domain = "old"
				require.NoError(t, f.k.SetFact(f.ctx, fact))
				before := f.snapshot(t)
				fact.Status = types.FactStatus_FACT_STATUS_ACTIVE
				fact.Submitter = "new"
				fact.Domain = "new"
				*f.kFault = handoffFault{tc.op, tc.prefix, after}
				require.Error(t, f.k.SetFact(f.ctx, fact))
				require.Equal(t, before, f.snapshot(t))
				require.Empty(t, f.ctx.EventManager().Events())
				*f.kFault = handoffFault{}
				require.NoError(t, f.k.SetFact(f.ctx, fact))
				require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
				histories, err := f.k.GetStatusHistoryChecked(f.ctx, "fact")
				require.NoError(t, err)
				require.Len(t, histories, 2)
				require.Nil(t, f.ctx.KVStore(f.keys[0]).Get(types.FactByDomainKey("old", "fact")))
			})
		}
	}
}

func TestRecordHistoryCascadeAtomicAndCorruptionRefusal(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	before := f.snapshot(t)
	event := &types.CascadeEvent{DisprovenFactId: "fact", DescendantFactId: "child", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CONTESTED}
	*f.kFault = handoffFault{"set", types.CascadeEventByDescendantPrefix, true}
	require.Error(t, f.k.RecordCascadeEvent(f.ctx, event))
	require.Zero(t, event.Seq)
	require.Equal(t, before, f.snapshot(t))
	*f.kFault = handoffFault{}
	require.NoError(t, f.k.RecordCascadeEvent(f.ctx, event))
	require.Equal(t, uint64(1), event.Seq)
	require.NoError(t, f.k.RecordCascadeEvent(f.ctx, event))
	require.Equal(t, uint64(2), event.Seq)
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
	f.ctx.KVStore(f.keys[0]).Set(types.StatusTransitionSeqKey("fact"), []byte{0x80, 0})
	before = f.snapshot(t)
	require.Error(t, f.k.RecordStatusTransition(f.ctx, &types.StatusTransition{FactId: "fact", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN}))
	require.Equal(t, before, f.snapshot(t))
	f.ctx.KVStore(f.keys[0]).Set(types.FactKey("fact"), []byte{0xff})
	before = f.snapshot(t)
	require.Error(t, f.k.SetFact(f.ctx, &types.Fact{Id: "fact"}))
	require.Equal(t, before, f.snapshot(t))
}

func TestRecordHistoryGenesisPreservesMigrationMarkersAndGaps(t *testing.T) {
	f := setupHandoff(t)
	markers := map[string]string{"native_source": "current", "migration_v6_complete": "true", "migration_v7_complete": "true", "accounting_authority": "preserve"}
	for k, v := range markers {
		require.NoError(t, f.k.WriteMigrationMarker(f.ctx, k, v))
	}
	transition := &types.StatusTransition{FactId: "old", Seq: 3, PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED, NewStatus: types.FactStatus_FACT_STATUS_ACTIVE, BlockHeight: 70}
	cascade := &types.CascadeEvent{DisprovenFactId: "root", DescendantFactId: "old", Seq: 7, BlockHeight: 80}
	counters := []*types.StatusTransitionCounter{{FactId: "old", Sequence: 9}, {FactId: "unknown", Sequence: 4}}
	require.NoError(t, f.k.ImportKnowledgeHistory(f.ctx, []*types.StatusTransition{transition}, []*types.CascadeEvent{cascade}, counters))
	tr, ca, co, err := f.k.ExportKnowledgeHistory(f.ctx)
	require.NoError(t, err)
	require.Len(t, tr, 1)
	require.True(t, proto.Equal(transition, tr[0]))
	require.Len(t, ca, 1)
	require.True(t, proto.Equal(cascade, ca[0]))
	require.Len(t, co, 2)
	before := f.snapshot(t)
	require.NoError(t, f.k.ImportKnowledgeHistory(f.ctx, tr, ca, co))
	require.Equal(t, before, f.snapshot(t))
	for k, v := range markers {
		require.Equal(t, v, f.k.ReadMigrationMarker(f.ctx, k))
	}
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	next := &types.StatusTransition{FactId: "old", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN}
	require.NoError(t, f.k.RecordStatusTransition(f.ctx, next))
	require.Equal(t, uint64(10), next.Seq)
	before = f.snapshot(t)
	require.Error(t, f.k.ImportKnowledgeHistory(f.ctx, []*types.StatusTransition{transition, transition}, ca, co))
	require.Equal(t, before, f.snapshot(t))
	require.Error(t, f.k.ImportKnowledgeHistory(f.ctx, tr, ca, nil))
	require.Equal(t, before, f.snapshot(t))
	require.NoError(t, f.k.ImportKnowledgeHistory(f.ctx, nil, nil, nil))
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
	for k, v := range markers {
		require.Equal(t, v, f.k.ReadMigrationMarker(f.ctx, k))
	}
}

func TestRecordHistoryQueryBoundsHeightAndFaithfulTrace(t *testing.T) {
	f := setupHandoff(t)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("ancestor%d", i)
		require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: id, Domain: "physics", VerifiedAtBlock: uint64(i + 1)}))
		require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: "fact", TargetFactId: id, Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
	}
	ids, edges, err := f.k.GatherAncestorCone(f.ctx, &types.AncestorConeSelector{LeafFactId: "fact", MaxDepth: 3, MaxPaths: 2})
	require.NoError(t, err)
	require.Len(t, ids, 3)
	require.Len(t, edges, 2)
	ids, _, err = f.k.GatherFrontier(f.ctx, &types.FrontierSelector{Domain: "physics", Limit: 1})
	require.NoError(t, err)
	require.Equal(t, []string{"ancestor4"}, ids)
	sel := &types.ToKSelector{Variant: &types.ToKSelector_AncestorCone{AncestorCone: &types.AncestorConeSelector{LeafFactId: "fact", MaxDepth: 3, MaxPaths: 2}}}
	before := f.snapshot(t)
	bundle, err := f.k.AssembleToKBundle(f.ctx, sel, 99)
	require.Error(t, err)
	require.Nil(t, bundle)
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
	bundle, err = f.k.AssembleToKBundle(f.ctx, sel, 100)
	require.NoError(t, err)
	require.Equal(t, uint64(100), bundle.SnapshotBlock)
	require.Equal(t, ComputeToKSnapshotRoot(bundle.IncludedNodeIds, bundle.IncludedEdges), bundle.SnapshotRoot)
	fact, ok := f.k.GetFact(f.ctx, "fact")
	require.True(t, ok)
	fact.CorroborationCount = ^uint64(0)
	fact.Confidence = 900000
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	require.NoError(t, f.k.SetVindicationRecord(f.ctx, "fact", types.VindicationRecord{FactId: "fact", Verifier: "a", VindicatedAt: 80}))
	require.NoError(t, f.k.SetVindicationRecord(f.ctx, "fact", types.VindicationRecord{FactId: "fact", Verifier: "b", VindicatedAt: 70}))
	trace, found, err := f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
	require.NoError(t, err)
	require.True(t, found)
	require.Empty(t, trace.BeliefRevisions)
	require.Equal(t, uint64(70), trace.Vindication.VindicatedAtBlock)
	f.ctx.KVStore(f.keys[0]).Set(types.VindicationRecordKey("fact", "a"), []byte("broken"))
	trace, found, err = f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
	require.Error(t, err)
	require.False(t, found)
	require.Nil(t, trace)
	_, err = SerialiseToK_JSONL(&types.ToKBundle{Nodes: []*types.Fact{{Id: "big", Content: strings.Repeat("x", ToKMaxOutputBytes)}}})
	require.ErrorIs(t, err, ErrToKResourceLimit)
	g := &tokQueryGuard{entries: ToKMaxReadEntries}
	require.ErrorIs(t, g.read([]byte{0}, nil), ErrToKResourceLimit)
	g = &tokQueryGuard{bytes: ToKMaxReadBytes}
	require.ErrorIs(t, g.read([]byte{0}, nil), ErrToKResourceLimit)
}

type historyIteratorFaultService struct {
	corestore.KVStoreService
	mode string
}

func (s historyIteratorFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return historyIteratorFaultStore{s.KVStoreService.OpenKVStore(ctx), s.mode}
}

type historyIteratorFaultStore struct {
	corestore.KVStore
	mode string
}

func (s historyIteratorFaultStore) Iterator(a, b []byte) (corestore.Iterator, error) {
	it, err := s.KVStore.Iterator(a, b)
	if err != nil {
		return nil, err
	}
	return historyIteratorFault{it, s.mode}, nil
}

type historyIteratorFault struct {
	corestore.Iterator
	mode string
}

func (i historyIteratorFault) Error() error {
	if i.mode == "error" && !i.Iterator.Valid() {
		return errors.New("injected iterator error")
	}
	return i.Iterator.Error()
}
func (i historyIteratorFault) Close() error {
	err := i.Iterator.Close()
	if i.mode == "close" {
		return errors.Join(err, errors.New("injected iterator close"))
	}
	return err
}
func TestRecordHistoryIteratorErrorsRefuseWholeRead(t *testing.T) {
	for _, mode := range []string{"error", "close"} {
		t.Run(mode, func(t *testing.T) {
			f := setupHandoff(t)
			k := f.k
			k.storeService = historyIteratorFaultService{k.storeService, mode}
			rows, err := k.GetStatusHistoryChecked(f.ctx, "fact")
			require.Error(t, err)
			require.Nil(t, rows)
			_, _, err = k.GatherFrontier(f.ctx, &types.FrontierSelector{Domain: "physics"})
			require.Error(t, err)
		})
	}
	f := setupHandoff(t)
	cache, _ := f.ctx.CacheContext()
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(cache))
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
}

func TestRecordHistoryCounterOverflowAndMissingRefuse(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	store := f.ctx.KVStore(f.keys[0])
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], ^uint64(0))
	store.Set(types.StatusTransitionSeqKey("fact"), buf[:n])
	before := f.snapshot(t)
	t1 := &types.StatusTransition{FactId: "fact", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_DISPROVEN}
	require.Error(t, f.k.RecordStatusTransition(f.ctx, t1))
	require.Equal(t, before, f.snapshot(t))
	store.Delete(types.StatusTransitionSeqKey("fact"))
	require.Error(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
	require.Error(t, f.k.RecordStatusTransition(f.ctx, t1))
}

func TestRecordHistoryFailedOversizeBundleDoesNotEmitEvents(t *testing.T) {
	f := setupHandoff(t)
	fact, ok := f.k.GetFact(f.ctx, "fact")
	require.True(t, ok)
	fact.Content = strings.Repeat("x", ToKMaxOutputBytes/2+1000)
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	sel := &types.ToKSelector{Variant: &types.ToKSelector_RootedSubtree{RootedSubtree: &types.RootedSubtreeSelector{RootFactId: "fact", MaxDepth: 1}}}
	bundle, err := f.k.AssembleToKBundle(f.ctx, sel, 0)
	require.ErrorIs(t, err, ErrToKResourceLimit)
	require.Nil(t, bundle)
	require.Empty(t, f.ctx.EventManager().Events())
}

func TestRecordHistoryUnknownWireAndOrphanIndexRefuse(t *testing.T) {
	f := setupHandoff(t)
	store := f.ctx.KVStore(f.keys[0])
	key := types.StatusTransitionKey("fact", 1)
	raw := bytes.Clone(store.Get(key))
	store.Set(key, append(raw, 0xf8, 0x07, 0x01))
	require.Error(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
	_, err := f.k.GetStatusHistoryChecked(f.ctx, "fact")
	require.Error(t, err)
	store.Set(key, raw)
	store.Set(types.CascadeEventByDescendantKey("orphan", "root"), []byte{1})
	require.Error(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
}

func TestRecordHistoryWholeCascadeFailurePreservesEarlierDescendants(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	for _, id := range []string{"a", "b"} {
		require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: id, Status: types.FactStatus_FACT_STATUS_ACTIVE}))
		require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: id, TargetFactId: "fact", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
	}
	before := f.snapshot(t)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	*f.kFault = handoffFault{"set", types.FactKey("b"), true}
	require.Error(t, f.k.cascadeFalsificationChecked(f.ctx, "fact", "challenge"))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
	*f.kFault = handoffFault{}
	require.NoError(t, f.k.cascadeFalsificationChecked(f.ctx, "fact", "challenge"))
	events, err := f.k.GetCascadeEventsForDisproofChecked(f.ctx, "fact")
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.NoError(t, f.k.ValidateKnowledgeHistoryState(f.ctx))
	after := f.snapshot(t)
	require.NoError(t, f.k.cascadeFalsificationChecked(f.ctx, "fact", "challenge"))
	require.Equal(t, after, f.snapshot(t))
}

func TestRecordHistoryDisproofDoesNotInventCausalFact(t *testing.T) {
	f := setupHandoff(t)
	fact, _ := f.k.GetFact(f.ctx, "fact")
	fact.Status = types.FactStatus_FACT_STATUS_DISPROVEN
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: "unrelated", TargetFactId: "fact", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, CreatedAtBlock: 12}))
	trace, ok, err := f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(100), trace.Disproval.DisprovenAtBlock)
	require.Empty(t, trace.Disproval.DisprovenByFactId)
	require.NoError(t, f.k.ImportKnowledgeHistory(f.ctx, nil, nil, nil))
	trace, ok, err = f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
	require.NoError(t, err)
	require.True(t, ok)
	require.Nil(t, trace.Disproval)
}

func TestRecordHistoryQueryRejectsAmbiguousVindicationJSON(t *testing.T) {
	for _, raw := range []string{`{"verifier":"a","verifier":"b","fact_id":"fact"}`, `{"Verifier":"a","fact_id":"fact"}`, `{"verifier":"a","fact_id":"fact"} {}`, `{"verifier":"a","fact_id":"fact","extra":1}`} {
		f := setupHandoff(t)
		f.ctx.KVStore(f.keys[0]).Set(types.VindicationRecordKey("fact", "a"), []byte(raw))
		trace, found, err := f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
		require.Error(t, err)
		require.False(t, found)
		require.Nil(t, trace)
	}
}

func TestRecordHistoryQueryGlobalVisitedCeiling(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "next"}))
	require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: "fact", TargetFactId: "next", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
	visited := map[string]bool{}
	for i := 0; i < ToKMaxNodes; i++ {
		visited[fmt.Sprintf("existing%d", i)] = true
	}
	var paths uint32
	err := f.k.gatherAncestorsRecursive(f.ctx, "fact", 0, 2, 2, &paths, visited, map[string]*types.ToKEdge{})
	require.ErrorIs(t, err, ErrToKResourceLimit)
}

func TestRecordHistoryNestedUnknownFieldsRefuse(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	fact, _ := f.k.GetFact(f.ctx, "fact")
	fact.Structure = &types.ClaimStructure{}
	fact.Structure.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 0x01})
	before := f.snapshot(t)
	require.Error(t, f.k.SetFact(f.ctx, fact))
	require.Equal(t, before, f.snapshot(t))
	raw, err := proto.Marshal(fact)
	require.NoError(t, err)
	f.ctx.KVStore(f.keys[0]).Set(types.FactKey("fact"), raw)
	trace, ok, err := f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "fact")
	require.Error(t, err)
	require.Nil(t, trace)
	require.False(t, ok)
}

func TestRecordHistoryDialecticVerdictsDescribeChallenge(t *testing.T) {
	f := setupHandoff(t)
	for _, tc := range []struct {
		outcome  string
		height   uint64
		verdict  types.StepVerdict
		children int
	}{
		{"disproven", 90, types.StepVerdict_STEP_VERDICT_SOUND, 2},
		{"survived", 90, types.StepVerdict_STEP_VERDICT_UNSOUND, 2},
		{"inconclusive", 90, types.StepVerdict_STEP_VERDICT_UNSPECIFIED, 2},
		{"pending", 0, types.StepVerdict_STEP_VERDICT_UNEXAMINED, 1},
	} {
		t.Run(tc.outcome, func(t *testing.T) {
			nodes := f.k.buildDialecticTree(f.ctx, &types.Fact{Id: "fact", Submitter: "author"}, []*types.TraceChallenge{{Outcome: tc.outcome, ResolvedBlock: tc.height, RebuttalText: "response"}})
			require.Len(t, nodes, 1)
			require.Zero(t, nodes[0].AtBlock)
			require.Equal(t, tc.verdict, nodes[0].NodeVerdict)
			require.Len(t, nodes[0].Children, tc.children)
			require.Zero(t, nodes[0].Children[0].AtBlock)
			if tc.height != 0 {
				require.Equal(t, tc.height, nodes[0].Children[1].AtBlock)
				require.Equal(t, tc.verdict, nodes[0].Children[1].NodeVerdict)
			}
		})
	}
}
