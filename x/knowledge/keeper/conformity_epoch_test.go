package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Regression tests for the conformity-cooling epoch off-by-one (fixed 2026-08-03).
//
// The defect: at a fitness-epoch boundary height H, BeginBlocker computed
// epoch := H/FitnessEpochBlocks — the epoch BEGINNING at H — and both
// aggregated diversity for it (ProcessDiversity) and consumed it for
// conformity cooling (UpdateEpistemicTemperature, which independently
// re-derived the same value from height). But rounds finalized during the
// epoch that just ENDED carry epoch (H/E)-1 (RecordRoundDiversity labels a
// round with height/E at finalization). So the completed epoch's rounds were
// never aggregated, GetDomainDiversity found nothing, and the else-branch
// reset ConformityStreak every single epoch. Cooling could only fire for a
// round finalizing at EXACTLY the boundary block (~1 in FitnessEpochBlocks).
//
// These tests drive the real BeginBlocker wiring (phases.go), not the keeper
// internals, so they exercise the boundary epoch arithmetic itself. They were
// verified FAILING against the unfixed code before the fix landed.
//
// Harness facts they rely on: setupKnowledgeTest starts at height 100 with
// DefaultParams (FitnessEpochBlocks = 10,000, DiversityConformityAlertThreshold
// = 50,000, EpistemicConformityCoolingBps = 50,000) and DefaultGenesis domains
// (which include "mathematics").

// A full epoch of mid-epoch 4-0 unanimity must cool the domain at the next
// boundary. This is the inversion of the pre-fix behavior, where the epoch-1
// rounds were invisible at the height-20,000 boundary and the streak reset.
func TestConformityCooling_MidEpochUnanimityCoolsAtNextBoundary(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// Five fully unanimous rounds finalize mid-epoch 1 (heights 10,000..19,999).
	ctx = ctx.WithBlockHeight(15_000)
	for i := 0; i < 5; i++ {
		require.NoError(t, k.RecordRoundDiversity(ctx, fmt.Sprintf("r-unan-%d", i), "mathematics", 4, 0))
	}

	// Neutral starting temperature.
	require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{
		Domain:      "mathematics",
		Temperature: 500_000,
	}))

	// Next fitness boundary: the epoch that just COMPLETED is epoch 1.
	ctx = ctx.WithBlockHeight(20_000)
	require.NoError(t, k.BeginBlocker(ctx))

	// The completed epoch's rounds were aggregated.
	rec, found, err := k.GetDomainDiversity(ctx, "mathematics", 1)
	require.NoError(t, err)
	require.True(t, found, "boundary must aggregate the COMPLETED epoch (1), not the one beginning at the boundary")
	require.Equal(t, uint64(5), rec.RoundCount)
	require.Equal(t, uint64(0), rec.AvgEntropy)
	require.Equal(t, uint64(5), rec.UnanimousCount)

	// Conformity cooling fired: streak built, temperature dropped.
	state, sfound, err := k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.True(t, sfound)
	require.Equal(t, uint64(1), state.ConformityStreak,
		"an epoch of pure unanimity must increment the conformity streak")
	// Decay is a no-op at neutral; cooling = 50,000 * streak(1) / 10 = 5,000.
	require.Equal(t, uint64(495_000), state.Temperature,
		"cooling must fire off the completed epoch's aggregated unanimity")
}

// The streak must be able to BUILD across consecutive unanimous epochs —
// pre-fix it was reset to zero at every boundary, so scaled cooling
// (streak-proportional) could never engage and the cold-domain confidence
// cap could never bind.
func TestConformityCooling_StreakBuildsAcrossConsecutiveEpochs(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{
		Domain:      "mathematics",
		Temperature: 500_000,
	}))

	// Epoch 1: unanimous rounds mid-epoch, then the boundary at 20,000.
	ctx = ctx.WithBlockHeight(15_000)
	for i := 0; i < 3; i++ {
		require.NoError(t, k.RecordRoundDiversity(ctx, fmt.Sprintf("r-e1-%d", i), "mathematics", 4, 0))
	}
	ctx = ctx.WithBlockHeight(20_000)
	require.NoError(t, k.BeginBlocker(ctx))

	state, _, err := k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.Equal(t, uint64(1), state.ConformityStreak)
	tempAfterFirst := state.Temperature
	require.Less(t, tempAfterFirst, uint64(500_000))

	// Epoch 2: unanimity continues, then the boundary at 30,000.
	ctx = ctx.WithBlockHeight(25_000)
	for i := 0; i < 3; i++ {
		require.NoError(t, k.RecordRoundDiversity(ctx, fmt.Sprintf("r-e2-%d", i), "mathematics", 4, 0))
	}
	ctx = ctx.WithBlockHeight(30_000)
	require.NoError(t, k.BeginBlocker(ctx))

	state, _, err = k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.Equal(t, uint64(2), state.ConformityStreak,
		"consecutive low-diversity epochs must build the streak, not reset it")
	require.Less(t, state.Temperature, tempAfterFirst,
		"streak-scaled cooling must keep lowering the temperature")
}

// A round finalizing at EXACTLY the boundary block carries the epoch
// beginning there, so it belongs to the NEXT boundary's aggregation. Pre-fix
// this coincidence was the only live cooling path (consumed immediately at
// its own boundary); post-fix it must be deferred, not lost and not
// double-processed.
func TestConformityCooling_BoundaryBlockRoundDefersToNextEpoch(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{
		Domain:      "mathematics",
		Temperature: 500_000,
	}))

	// Round finalizes at exactly height 20,000 — RecordRoundDiversity labels
	// it epoch 2 (AdvanceRoundPhases runs before ProcessDiversity in the same
	// BeginBlocker, so the record exists when diversity is aggregated).
	ctx = ctx.WithBlockHeight(20_000)
	require.NoError(t, k.RecordRoundDiversity(ctx, "r-boundary", "mathematics", 4, 0))
	require.NoError(t, k.BeginBlocker(ctx))

	// This boundary aggregates completed epoch 1 (empty) — the boundary-block
	// round is NOT consumed early.
	_, found, err := k.GetDomainDiversity(ctx, "mathematics", 2)
	require.NoError(t, err)
	require.False(t, found, "epoch 2 must not be aggregated at its own opening boundary")
	state, _, err := k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.Equal(t, uint64(0), state.ConformityStreak,
		"a round opening epoch 2 must not cool at the height-20,000 boundary")
	require.Equal(t, uint64(500_000), state.Temperature)

	// At the next boundary the round is aggregated into epoch 2 and cools.
	ctx = ctx.WithBlockHeight(30_000)
	require.NoError(t, k.BeginBlocker(ctx))

	rec, found, err := k.GetDomainDiversity(ctx, "mathematics", 2)
	require.NoError(t, err)
	require.True(t, found, "the boundary-block round must be aggregated at the NEXT boundary")
	require.Equal(t, uint64(1), rec.RoundCount)
	state, _, err = k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.Equal(t, uint64(1), state.ConformityStreak)
	require.Less(t, state.Temperature, uint64(500_000))
}

// Healthy (split-vote) diversity in the completed epoch must be aggregated
// and visible at the boundary — with high entropy the streak resets and no
// cooling fires. Pre-fix the completed epoch's record was simply never
// written, so this record did not exist at all.
func TestProcessDiversity_BoundaryAggregatesCompletedEpoch(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// Seed a pre-existing streak so the healthy-epoch reset is observable.
	require.NoError(t, k.SetDomainEpistemicState(ctx, &types.DomainEpistemicState{
		Domain:           "mathematics",
		Temperature:      500_000,
		ConformityStreak: 3,
	}))

	// Three contested 2-2 rounds mid-epoch 1 (entropy 1,000,000 each).
	ctx = ctx.WithBlockHeight(15_000)
	for i := 0; i < 3; i++ {
		require.NoError(t, k.RecordRoundDiversity(ctx, fmt.Sprintf("r-split-%d", i), "mathematics", 2, 2))
	}

	ctx = ctx.WithBlockHeight(20_000)
	require.NoError(t, k.BeginBlocker(ctx))

	rec, found, err := k.GetDomainDiversity(ctx, "mathematics", 1)
	require.NoError(t, err)
	require.True(t, found, "the completed epoch's diverse rounds must be aggregated at the boundary")
	require.Equal(t, uint64(3), rec.RoundCount)
	require.Equal(t, uint64(1_000_000), rec.AvgEntropy)
	require.Equal(t, uint64(0), rec.UnanimousCount)

	// Healthy entropy: streak resets, no cooling (decay is a no-op at neutral).
	state, _, err := k.GetDomainEpistemicState(ctx, "mathematics")
	require.NoError(t, err)
	require.Equal(t, uint64(0), state.ConformityStreak)
	require.Equal(t, uint64(500_000), state.Temperature)
}
