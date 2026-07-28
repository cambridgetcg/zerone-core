package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ========== Early Aggregation Tests (K-alpha) ==========
//
// GetExpectedPhase advances a REVEAL-phase round to AGGREGATION as soon as
// every committer has revealed and the commit set meets MinVerifiers, and is
// monotone: it never returns a phase behind the stored one, so a failed
// aggregation (round left standing in AGGREGATION) cannot be regressed to
// REVEAL by deadline arithmetic and reprocessed every block.

// earlyAggRound builds a round with n matching commit/reveal "accept" pairs.
// Deadlines follow testRound: commit 150, reveal 200, aggregation 250.
func earlyAggRound(id, claimID string, phase types.VerificationPhase, commits, reveals int) *types.VerificationRound {
	round := testRound(id, claimID, phase)
	for i := 0; i < commits; i++ {
		verifier := fmt.Sprintf("zrn1earlyaggval%dxxxxxxxxxxxxxxxxxxxx", i)
		salt := []byte(fmt.Sprintf("early-agg-salt-%d-%d", i, len(id)))
		round.SelectedVerifiers = append(round.SelectedVerifiers, verifier)
		round.Commits = append(round.Commits, &types.CommitEntry{
			Verifier:         verifier,
			CommitHash:       types.ComputeCommitmentHash(id, "accept", 800000, salt),
			CommittedAtBlock: uint64(60 + i),
		})
		if i < reveals {
			round.Reveals = append(round.Reveals, &types.RevealEntry{
				Verifier:        verifier,
				Vote:            "accept",
				Salt:            salt,
				RevealedAtBlock: uint64(155 + i),
			})
		}
	}
	return round
}

// earlyAggClaim stores a minimal claim so CompleteRound can settle the round.
func earlyAggClaim(t *testing.T, k keeper.Keeper, ctx sdk.Context, id string) {
	t.Helper()
	require.NoError(t, k.SetClaim(ctx, &types.Claim{
		Id:          id,
		FactContent: "Early aggregation regression fixture",
		Domain:      "physics",
		Category:    "empirical",
		Submitter:   "zrn1submitter000000000000000000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Stake:       "1000000",
	}))
}

// earlyAggCountEvents counts emitted events of the given type.
func earlyAggCountEvents(ctx sdk.Context, eventType string) int {
	count := 0
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == eventType {
			count++
		}
	}
	return count
}

// (a) Happy path: all reveals in => the round completes before the reveal
// deadline instead of waiting out the full window.
func TestEarlyAggregation_AllRevealsInCompletesBeforeDeadline(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	earlyAggClaim(t, k, ctx, "claim-early-happy")
	// 4 commits / 4 reveals: meets MinVerifiers (3) and the effective
	// domain minimum (base+1 = 4) so aggregation reaches a verdict.
	round := earlyAggRound("round-early-happy", "claim-early-happy", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// Inside the reveal window: commitDeadline 150 <= 160 < revealDeadline 200.
	ctx = ctx.WithBlockHeight(160).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	updated, found := k.GetVerificationRound(ctx, "round-early-happy")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, updated.Phase)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, updated.Verdict)
	require.Less(t, updated.VerdictBlock, round.RevealDeadline,
		"verdict must land before the reveal deadline")
	require.Equal(t, 1, earlyAggCountEvents(ctx, "zerone.knowledge.round_phase_changed"))
}

// (b) Aggregation-failure path: CompleteRound fails (claim missing — the
// phases.go failure branch logs and leaves the round standing in
// AGGREGATION). The round must not regress phase, and the BeginBlocker must
// not reprocess it every block: it parks until the aggregation deadline,
// where the existing EXPIRED branch retries — exactly the pre-early-advance
// failure semantics.
func TestEarlyAggregation_FailedAggregationParksWithoutLivelock(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	// No claim stored: performAggregation → CompleteRound errors on the
	// claim lookup after the round was already stored in AGGREGATION.
	round := earlyAggRound("round-early-fail", "claim-early-missing", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// Block 1: early advance fires, aggregation fails, round left standing.
	ctx = ctx.WithBlockHeight(160).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	updated, found := k.GetVerificationRound(ctx, "round-early-fail")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, updated.Phase)
	require.Equal(t, 1, earlyAggCountEvents(ctx, "zerone.knowledge.round_phase_changed"))

	// Every remaining block before the aggregation deadline (250): the round
	// stays parked in AGGREGATION — no phase regression, no transition
	// events, no re-aggregation.
	for h := int64(161); h < 250; h++ {
		ctx = ctx.WithBlockHeight(h).WithEventManager(sdk.NewEventManager())
		require.NoError(t, k.AdvanceRoundPhases(ctx))

		updated, found = k.GetVerificationRound(ctx, "round-early-fail")
		require.True(t, found)
		require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, updated.Phase,
			"phase regressed at height %d", h)
		require.Empty(t, ctx.EventManager().Events(),
			"round reprocessed at height %d", h)
	}

	// At the aggregation deadline the pre-existing EXPIRED branch retries the
	// late aggregation (still failing here). The round still never regresses.
	ctx = ctx.WithBlockHeight(250).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	updated, found = k.GetVerificationRound(ctx, "round-early-fail")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, updated.Phase)
}

// (c) Fewer reveals than commitments: no early advance — an outstanding
// reveal may still arrive inside the window.
func TestEarlyAggregation_MissingRevealDoesNotAdvance(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	round := earlyAggRound("round-early-partial", "claim-early-partial", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 3)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	ctx = ctx.WithBlockHeight(160).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	updated, found := k.GetVerificationRound(ctx, "round-early-partial")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, updated.Phase)
	require.Empty(t, ctx.EventManager().Events())
}

// (d) MinVerifiers unmet: a fully-revealed but undersized commit set does
// not advance early (DefaultParams.MinVerifiers = 3).
func TestEarlyAggregation_MinVerifiersUnmetDoesNotAdvance(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	round := earlyAggRound("round-early-small", "claim-early-small", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 2, 2)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	ctx = ctx.WithBlockHeight(160).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	updated, found := k.GetVerificationRound(ctx, "round-early-small")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, updated.Phase)
	require.Empty(t, ctx.EventManager().Events())
}

// The commit window is never shortened: even a synthetic round holding a
// full reveal set stays in COMMIT until the commit deadline passes.
func TestGetExpectedPhase_EarlyAdvanceNeverShortensCommit(t *testing.T) {
	params := types.DefaultParams()
	round := earlyAggRound("r-early-commit", "c-early-commit", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 4, 4)

	// height 100 < commitDeadline 150 → COMMIT, regardless of the sets.
	got := keeper.GetExpectedPhase(round, 100, &params)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, got)

	// At the commit deadline the closed reveal set advances immediately.
	got = keeper.GetExpectedPhase(round, round.CommitDeadline, &params)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, got)
}

// Phase monotonicity: deadline arithmetic lagging the stored phase never
// wins — the stored phase is the floor.
func TestGetExpectedPhase_MonotoneNeverRegresses(t *testing.T) {
	params := types.DefaultParams()

	// Stored AGGREGATION inside the reveal window with an open reveal set
	// (the shape left behind by a failed early aggregation once conditions
	// shift, e.g. a raised MinVerifiers): deadline math says REVEAL, the
	// stored phase must win.
	round := earlyAggRound("r-early-mono", "c-early-mono", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 4, 3)
	got := keeper.GetExpectedPhase(round, 160, &params)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, got)

	// Stored REVEAL inside the commit window: never back to COMMIT.
	round = earlyAggRound("r-early-mono2", "c-early-mono2", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 0, 0)
	got = keeper.GetExpectedPhase(round, 100, &params)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, got)

	// Expiry still outranks a stored AGGREGATION (the late-retry branch
	// must keep firing).
	round = earlyAggRound("r-early-mono3", "c-early-mono3", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 4, 3)
	got = keeper.GetExpectedPhase(round, round.AggregationDeadline, &params)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_EXPIRED, got)
}
