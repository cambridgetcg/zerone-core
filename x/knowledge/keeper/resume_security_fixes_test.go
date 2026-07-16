package keeper_test

// Drills for the CONTESTED/CHALLENGE-lock closure staged as a follow-up
// security upgrade for zerone-1 (framework-critique 2026-07-10). The doctrine
// resurrection made 47 previously-EXPIRED (lock-immune) facts VERIFIED again,
// so the starved-round lock attacks now have a live surface.
//
//	A starved verification round (reveals < MinVerifiers) previously hand-rolled
//	its termination and skipped the reversal/restore/record side effects that
//	CompleteRound performs — so a starved CONTRADICTS or CHALLENGE claim left
//	its target fact locked CONTESTED/CHALLENGED forever for the price of one
//	starved round, and the attempt was erased from the C2 calibration ledger.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestStarvedRound_UnlocksContestedFact(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// An established fact an attacker will try to bury.
	fact := makeTestFact(t, k, ctx, "fact-contested", "the sky is blue", "science", "empirical", "zrn1honest", 880000)
	// Reproduce SubmitClaim's immediate CONTESTED flip on a CONTRADICTS target.
	fact.Status = types.FactStatus_FACT_STATUS_CONTESTED
	require.NoError(t, k.SetFact(ctx, fact))

	// The attacker's 0.1-ZRN contradiction claim + its round.
	claim, round := makeTestClaim(t, k, ctx, "zrn1attacker", "the sky is not blue", "science", "empirical", "100000")
	claim.Status = types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION
	claim.Relations = []*types.ClaimRelation{{
		Relation:     types.RelationType_RELATION_TYPE_CONTRADICTS,
		TargetFactId: fact.Id,
	}}
	require.NoError(t, k.SetClaim(ctx, claim))

	// Starve the round: jump past the aggregation deadline with no reveals.
	ctx2 := ctx.WithBlockHeight(int64(round.AggregationDeadline) + 1)
	require.NoError(t, k.AdvanceRoundPhases(ctx2))

	got, found := k.GetFact(ctx2, fact.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, got.Status,
		"a starved contradiction must NOT leave the target fact locked CONTESTED forever")
}

func TestStarvedRound_UnlocksChallengedFact(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	fact := makeTestFact(t, k, ctx, "fact-challenged", "water is wet", "science", "empirical", "zrn1honest", 880000)
	// Reproduce ChallengeFact's flip to CHALLENGED at submission.
	fact.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	require.NoError(t, k.SetFact(ctx, fact))

	claim, round := makeTestClaim(t, k, ctx, "zrn1challenger", "water is dry", "science", "empirical", "11000000")
	claim.Status = types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION
	claim.ProvisionalFactId = fact.Id
	require.NoError(t, k.SetClaim(ctx, claim))

	ctx2 := ctx.WithBlockHeight(int64(round.AggregationDeadline) + 1)
	require.NoError(t, k.AdvanceRoundPhases(ctx2))

	got, found := k.GetFact(ctx2, fact.Id)
	require.True(t, found)
	require.NotEqual(t, types.FactStatus_FACT_STATUS_CHALLENGED, got.Status,
		"a starved challenge must NOT leave the target fact locked CHALLENGED forever")
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, got.Status,
		"restored with no survival credit, since no panel judged it")
}

func TestStarvedRound_RecordsInconclusiveOutcome(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	_, round := makeTestClaim(t, k, ctx, "zrn1starved", "an unwitnessed claim", "science", "empirical", "100000")

	ctx2 := ctx.WithBlockHeight(int64(round.AggregationDeadline) + 1)
	require.NoError(t, k.AdvanceRoundPhases(ctx2))

	cal, ok := k.GetAgentCalibration(ctx2, "zrn1starved")
	require.True(t, ok, "C2/COMPASSION: a starved round must still record the attempt in the calibration ledger")
	require.GreaterOrEqual(t, cal.Inconclusive, uint64(1),
		"the inconclusive outcome must be counted, not erased")
}

// The second CONTESTED entry path: SubmitContradiction. It previously flipped
// the target CONTESTED with an empty Relations slice, so the healing path had
// nothing to iterate and the fact stayed locked even after the fix to the
// starved SubmitClaim path.
func TestSubmitContradiction_StarvedDoesNotLockFact(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	fact := makeTestFact(t, k, ctx, "fact-sc", "gravity pulls down", "physics", "empirical", "zrn1honest", 880000)

	resp, err := ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{
		FactId:       fact.Id,
		Submitter:    makeValidBech32Addr("sc-attacker"),
		CounterClaim: "gravity pushes up",
		Domain:       "physics",
		Category:     "empirical",
		Stake:        "100000",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.CounterFactId)

	// The flip landed (established target).
	mid, _ := k.GetFact(ctx, fact.Id)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, mid.Status)

	// Starve every open round, then heal.
	ctx2 := ctx.WithBlockHeight(ctx.BlockHeight() + 10_000)
	require.NoError(t, k.AdvanceRoundPhases(ctx2))

	got, found := k.GetFact(ctx2, fact.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, got.Status,
		"a starved SubmitContradiction must NOT leave the target fact locked CONTESTED forever")
}
