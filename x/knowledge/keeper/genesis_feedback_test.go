package keeper_test

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func openDurabilityStore(t *testing.T, dir string) (keeper.Keeper, sdk.Context, storetypes.CommitMultiStore, dbm.DB) {
	t.Helper()
	db, err := dbm.NewDB("durability", dbm.GoLevelDBBackend, dir)
	require.NoError(t, err)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, ms.LoadLatestVersion())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", nil, nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 50}, false, log.NewNopLogger())
	return k, ctx, ms, db
}
func TestDurableGenesisColdRoundTrip(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	gs := types.DefaultGenesis()
	gs.Params.FitnessEpochBlocks = 10
	gs.Params.FitnessWeightQueryBps = 0
	gs.Params.FitnessWeightSatisfactionBps = 0
	gs.Params.MetabolismEnergyPerQuery = 0
	require.NoError(t, k.InitGenesis(ctx, gs))
	feedbackFact(t, k, ctx, "ordinary")
	feedbackFact(t, k, ctx, "target")
	rel := &types.FactRelation{SourceFactId: "ordinary", TargetFactId: "target", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, CreatedAtBlock: 8, MethodId: "recorded-method", InferenceStrengthBps: 777, Creator: feedbackConsumer(1)}
	require.NoError(t, k.SetFactRelation(ctx, rel))
	// Preserve changed doctrine relation metadata rather than reseeding defaults.
	exported := k.ExportGenesis(ctx)
	require.NotEmpty(t, exported.FactRelations)
	doctrine := proto.Clone(exported.FactRelations[0]).(*types.FactRelation)
	doctrine.CreatedAtBlock = 4
	doctrine.MethodId = "old-doctrine-method"
	require.NoError(t, k.SetFactRelation(ctx, doctrine))
	raw := s.KVStoreService.OpenKVStore(ctx)
	require.NoError(t, raw.Set(types.StatusTransitionSeqKey("ordinary"), binary.AppendUvarint(nil, 12)))
	// Counter-only sparse marker must survive even when its earlier history does not.
	require.NoError(t, raw.Set(types.StatusTransitionSeqKey("missing-history"), binary.AppendUvarint(nil, 31)))
	require.NoError(t, k.RecordCascadeEvent(ctx, &types.CascadeEvent{DisprovenFactId: "ordinary", DescendantFactId: "target", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CONTESTED, BlockHeight: 4, EdgeRelation: "SUPPORTS"}))
	require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{Id: "legacy-complete", ClaimId: "old-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 7, Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{Id: "actual-complete", ClaimId: "actual-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 9}))
	require.NoError(t, k.IndexCompletedRound(ctx, 9, "actual-complete", &types.CompletedRoundMeta{Domain: "recorded-domain", HasDissent: true, DurationBlocks: 6}))
	receipts := []*types.FactUseReceipt{feedbackReceipt(0, feedbackConsumer(1), "ordinary"), feedbackReceipt(1, feedbackConsumer(1), "target")}
	receipts[0].Rating = types.FactUseRating_FACT_USE_RATING_USEFUL
	receipts[0].RatingHeight = receipts[0].UseHeight
	require.NoError(t, k.InitFactUseReceipts(ctx, receipts, &types.FactUsePruningState{EverReported: true, NextKey: types.FactUseReceiptKey(0, feedbackConsumer(1), "ordinary")}))
	before := k.ExportGenesis(ctx)
	wire, err := protojson.Marshal(before)
	require.NoError(t, err)
	decoded := new(types.GenesisState)
	require.NoError(t, protojson.Unmarshal(wire, decoded))
	dir := t.TempDir()
	restored, rctx, ms, db := openDurabilityStore(t, dir)
	require.NoError(t, restored.InitGenesis(rctx, decoded))
	ms.Commit()
	require.NoError(t, db.Close())
	reopened, cold, _, db2 := openDurabilityStore(t, dir)
	defer db2.Close()
	after := reopened.ExportGenesis(cold)
	require.True(t, proto.Equal(before, after), "cold export/import changed canonical typed state")
	require.Len(t, after.CompletedRounds, 2)
	require.Len(t, after.CompletedRoundRecords, 1, "missing old metadata must not be fabricated")
	tr := &types.StatusTransition{FactId: "ordinary", PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CHALLENGED}
	require.NoError(t, reopened.RecordStatusTransition(cold, tr))
	require.EqualValues(t, 13, tr.Seq)
	// Once all receipts expire/prune, the durable economic latch remains.
	require.NoError(t, reopened.PruneFactUseReceipts(cold))
	pruned := reopened.ExportGenesis(cold)
	require.Empty(t, pruned.FactUseReceipts)
	require.True(t, pruned.FactUsePruning.EverReported)
	other, oc, _, odb := openDurabilityStore(t, t.TempDir())
	defer odb.Close()
	require.NoError(t, other.InitGenesis(oc, pruned))
	require.True(t, other.ExportGenesis(oc).FactUsePruning.EverReported)
}

func TestDurableGenesisEmptyGraphDoesNotReseedDoctrine(t *testing.T) {
	k, ctx, _, db := openDurabilityStore(t, t.TempDir())
	defer db.Close()
	gs := types.DefaultGenesis()
	// Every new export includes a pruning singleton even when never reported.
	// Its presence distinguishes an authoritative empty graph from fresh seed.
	gs.FactUsePruning = &types.FactUsePruningState{}
	require.NoError(t, k.InitGenesis(ctx, gs))
	after := k.ExportGenesis(ctx)
	require.Empty(t, after.Facts)
	require.Empty(t, after.FactRelations)
	require.Empty(t, after.StatusTransitions)
}

func TestDurableGenesisRejectsBeforeWritesAndExportFailsClosed(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	gs := types.DefaultGenesis()
	gs.StatusTransitions = []*types.StatusTransition{{FactId: "f", Seq: 3, PriorStatus: types.FactStatus_FACT_STATUS_ACTIVE, NewStatus: types.FactStatus_FACT_STATUS_CONTESTED}}
	gs.StatusTransitionSequences = []*types.StatusTransitionSequence{{FactId: "f", LastSequence: 2}}
	before := feedbackSnapshot(t, s, ctx)
	require.Error(t, k.InitGenesis(ctx, gs))
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Set(types.StatusTransitionSeqKey("broken"), []byte{0x80}))
	require.Panics(t, func() { k.ExportGenesis(ctx) })
}
