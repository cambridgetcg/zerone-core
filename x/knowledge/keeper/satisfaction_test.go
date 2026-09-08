package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestRecordQueryReceipt(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	require.False(t, k.HasQueryReceipt(ctx, "legacy-rater", "fact"))
	require.NoError(t, k.RecordQueryReceipt(ctx, "legacy-rater", "fact"))
	require.True(t, k.HasQueryReceipt(ctx, "legacy-rater", "fact"))
	require.False(t, k.HasQueryReceipt(ctx, "other", "fact"))
}
func TestConsumeQueryReceipt(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	require.NoError(t, k.RecordQueryReceipt(ctx, "legacy-rater", "fact"))
	require.NoError(t, k.ConsumeQueryReceipt(ctx, "legacy-rater", "fact"))
	require.False(t, k.HasQueryReceipt(ctx, "legacy-rater", "fact"))
}
func TestClearQueryReceipts(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	for _, rater := range []string{"a", "b", "c"} {
		require.NoError(t, k.RecordQueryReceipt(ctx, rater, "fact"))
	}
	k.ClearQueryReceipts(ctx) // Explicit legacy helper; not called by BeginBlocker.
	for _, rater := range []string{"a", "b", "c"} {
		require.False(t, k.HasQueryReceipt(ctx, rater, "fact"))
	}
}
func TestRateFactPositiveAndNegative(t *testing.T) {
	for _, useful := range []bool{true, false} {
		t.Run(map[bool]string{true: "positive", false: "negative"}[useful], func(t *testing.T) {
			k, ctx, _ := setupFeedbackStore(t)
			f := feedbackFact(t, k, ctx, "fact")
			m := keeper.NewMsgServerImpl(k)
			consumer := feedbackConsumer(1)
			_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
			require.NoError(t, err)
			_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: f.Id, Useful: useful})
			require.NoError(t, err)
			updated, found := k.GetFact(ctx, f.Id)
			require.True(t, found)
			if useful {
				require.Equal(t, uint64(1), updated.SatisfactionUp)
				require.Equal(t, uint64(1), updated.SatisfactionUpEpoch)
				require.Zero(t, updated.SatisfactionDown)
			} else {
				require.Equal(t, uint64(1), updated.SatisfactionDown)
				require.Equal(t, uint64(1), updated.SatisfactionDownEpoch)
				require.Zero(t, updated.SatisfactionUp)
			}
		})
	}
}
func TestRateFactNoReceiptAndLegacyNotPromoted(t *testing.T) {
	k, ctx, _ := setupFeedbackStore(t)
	feedbackFact(t, k, ctx, "fact")
	m := keeper.NewMsgServerImpl(k)
	consumer := feedbackConsumer(1)
	for _, legacy := range []bool{false, true} {
		if legacy {
			require.NoError(t, k.RecordQueryReceipt(ctx, consumer, "fact"))
		}
		_, err := m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: "fact", Useful: true})
		require.ErrorContains(t, err, "current-epoch signed")
	}
	require.True(t, k.HasQueryReceipt(ctx, consumer, "fact"))
}
func TestSatisfactionFitnessImpact(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	high := makeTestFact(t, k, ctx, "high", "High satisfaction content", "physics", "empirical", "submitter", 800000)
	low := makeTestFact(t, k, ctx, "low", "Low satisfaction content.", "physics", "empirical", "submitter", 800000)
	high.QueryCountEpoch, low.QueryCountEpoch = 100, 100
	high.SatisfactionUpEpoch = 10
	low.SatisfactionDownEpoch = 10
	require.Greater(t, k.CalculateFitness(ctx, high, 1), k.CalculateFitness(ctx, low, 1)) // Legacy non-beta weights remain supported before any report.
}
func TestSatisfactionMinRatings(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	few := makeTestFact(t, k, ctx, "few", "Few ratings content here.", "physics", "empirical", "submitter", 800000)
	none := makeTestFact(t, k, ctx, "none", "No ratings content here.", "physics", "empirical", "submitter", 800000)
	few.QueryCountEpoch, none.QueryCountEpoch = 100, 100
	few.SatisfactionDownEpoch = 2
	require.Equal(t, k.CalculateFitness(ctx, few, 1), k.CalculateFitness(ctx, none, 1))
}
func TestSatisfactionEpochReset(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	f := makeTestFact(t, k, ctx, "fact-epoch", "Epoch reset content here.", "physics", "empirical", "submitter", 800000)
	f.SatisfactionUp, f.SatisfactionDown, f.SatisfactionUpEpoch, f.SatisfactionDownEpoch, f.QueryCountEpoch = 10, 3, 5, 2, 50
	require.NoError(t, k.SetFact(ctx, f))
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(int64(p.FitnessEpochBlocks))
	require.NoError(t, k.UpdateAllFitnessScores(ctx))
	updated, found := k.GetFact(ctx, f.Id)
	require.True(t, found)
	require.Equal(t, uint64(50), updated.QueryCountEpoch, "fitness must leave counters available for metabolism")
	require.NoError(t, k.ResetFactFeedbackEpochCounters(ctx))
	updated, found = k.GetFact(ctx, f.Id)
	require.True(t, found)
	require.Equal(t, uint64(10), updated.SatisfactionUp)
	require.Equal(t, uint64(3), updated.SatisfactionDown)
	require.Zero(t, updated.QueryCountEpoch)
	require.Zero(t, updated.SatisfactionUpEpoch)
	require.Zero(t, updated.SatisfactionDownEpoch)
}
func TestMemoTooLong(t *testing.T) {
	k, ctx, _ := setupFeedbackStore(t)
	feedbackFact(t, k, ctx, "fact")
	m := keeper.NewMsgServerImpl(k)
	consumer := feedbackConsumer(1)
	_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: "fact"})
	require.NoError(t, err)
	_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: "fact", Memo: strings.Repeat("x", 257)})
	require.ErrorContains(t, err, "256 bytes")
	_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: "fact", Memo: strings.Repeat("x", 256)})
	require.NoError(t, err)
}
