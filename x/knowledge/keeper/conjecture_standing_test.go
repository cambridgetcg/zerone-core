package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── The standing invariant ─────────────────────────────────────────────────
//
// One property, tested against every route an adversarial review found into
// it: a conjecture must never acquire standing it did not earn through
// verification of its truth.
//
// The first version of this feature keyed its guards on FACT_STATUS_PROVISIONAL
// and was defeated five separate ways, because Status is mutable by cheap
// public messages while ClaimType is not. Every test below drives the fact out
// of PROVISIONAL by a different route and then asserts the exemptions still
// hold. Each was written after watching the corresponding exploit succeed.

func TestStanding_ContradictionCannotLaunderAConjectureIntoVerified(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	conj := acceptConjecture(t, k, ctx, "s-1", "A conjecture someone wants promoted")

	// Route: MsgSubmitContradiction had no status gate and no minimum stake.
	// It flipped any fact to CONTESTED, and reverseContradictionsFromClaim
	// then wrote VERIFIED unconditionally on REJECT/MALFORMED/INCONCLUSIVE.
	_, err := ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{
		Submitter:    "zrn1attacker",
		FactId:       conj.Id,
		CounterClaim: "This conjecture is false, allegedly",
		Domain:       "mathematics",
		Stake:        "1",
	})
	require.Error(t, err, "a conjecture asserts nothing and must not be contradictable")
	require.Contains(t, err.Error(), "conjecture")

	after, found := k.GetFact(ctx, conj.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.Status)
}

func TestStanding_ReverseContradictionRestoresProvisionalNotVerified(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "s-2", "A conjecture reached by the reversal path")

	// Belt and braces: even if a conjecture somehow reaches CONTESTED (via a
	// pre-existing record, a migration, or a future writer), the reversal must
	// not mint it into a believed fact.
	conj.Status = types.FactStatus_FACT_STATUS_CONTESTED
	require.NoError(t, k.SetFact(ctx, conj))

	claim := &types.Claim{
		Id:          "rev-2",
		FactContent: "A contradicting claim that will be rejected",
		Domain:      "mathematics",
		Submitter:   "zrn1attacker",
		Stake:       "100000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Relations: []*types.ClaimRelation{{
			TargetFactId: conj.Id,
			Relation:     types.RelationType_RELATION_TYPE_CONTRADICTS,
		}},
	}
	require.NoError(t, k.SetClaim(ctx, claim))
	round := makeRoundInPhase("r-rev-2", "rev-2", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 90)
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict: types.Verdict_VERDICT_REJECT,
	}))

	after, found := k.GetFact(ctx, conj.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.Status,
		"the contradiction reversal hardcoded VERIFIED; on a conjecture that laundered a question into a believed fact using only honest panels")
}

func TestStanding_PatronageCannotBuyAConjectureIntoActive(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	conj := acceptConjecture(t, k, ctx, "s-3", "A conjecture someone wants to fund into standing")

	_, err := ms.PatronizeFact(ctx, &types.MsgPatronizeFact{
		Patron:         "zrn1patron",
		FactId:         conj.Id,
		Amount:         "1",
		DurationBlocks: 1_000_000,
	})
	require.Error(t, err, "patronage buys energy and energy converted to ACTIVE — 1 uzrn defeated every exemption")
	require.Contains(t, err.Error(), "conjecture")

	after, _ := k.GetFact(ctx, conj.Id)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.Status)
}

func TestStanding_MetabolicRecoveryRestoresProvisionalNotActive(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	params, perr := k.GetParams(ctx)
	require.NoError(t, perr)

	conj := acceptConjecture(t, k, ctx, "s-4", "A conjecture recovering from at-risk")

	// Put it in the recovery arm's precondition: at-risk, with healthy energy.
	conj.Status = types.FactStatus_FACT_STATUS_AT_RISK
	conj.AtRiskSinceEpoch = 1
	conj.Energy = params.MetabolismEnergyCap
	require.NoError(t, k.SetFact(ctx, conj))

	k.ApplyPatronageEnergyBoost(ctx, conj, 1_000_000, "zrn1patron")

	after, found := k.GetFact(ctx, conj.Id)
	require.True(t, found)
	require.NotEqual(t, types.FactStatus_FACT_STATUS_ACTIVE, after.Status,
		"the metabolism recovery arm wrote ACTIVE unconditionally — the single widest route into standing, reachable with no attacker at all")
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.Status)
}

// TestStanding_CitationWallHoldsInEveryStatus is the general form of the fix.
// The wall used to key on PROVISIONAL, so ANY exit from that status reopened
// the zero-floor escape.
func TestStanding_CitationWallHoldsInEveryStatus(t *testing.T) {
	statuses := []types.FactStatus{
		types.FactStatus_FACT_STATUS_PROVISIONAL,
		types.FactStatus_FACT_STATUS_CHALLENGED,
		types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_AT_RISK,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_EXPIRED,
	}
	for _, st := range statuses {
		t.Run(st.String(), func(t *testing.T) {
			k, ctx, _ := setupKnowledgeTestWithBank(t)
			ms := keeper.NewMsgServerImpl(k)

			conj := acceptConjecture(t, k, ctx, "s-cite", "A conjecture walked through the status graph")
			conj.Status = st
			require.NoError(t, k.SetFact(ctx, conj))

			_, err := ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
				Submitter:   "zrn1citer",
				FactContent: "A claim leaning on a conjecture for support",
				Domain:      "mathematics",
				Stake:       "100000",
				References:  []string{conj.Id},
			})
			require.Error(t, err, "the citation wall must key on ClaimType, not on the status the fact happens to hold")
			require.Contains(t, err.Error(), "conjecture")
		})
	}
}

// TestStanding_NoTrainingValueInAnyStatus — CHALLENGED and CONTESTED are
// TVW-bearing for ordinary facts (deliberately, so adjudication does not
// suspend revenue). A conjecture must not inherit that.
func TestStanding_NoTrainingValueInAnyStatus(t *testing.T) {
	for _, st := range []types.FactStatus{
		types.FactStatus_FACT_STATUS_PROVISIONAL,
		types.FactStatus_FACT_STATUS_CHALLENGED,
		types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_VERIFIED,
	} {
		t.Run(st.String(), func(t *testing.T) {
			k, ctx, _ := setupKnowledgeTestWithBank(t)
			conj := acceptConjecture(t, k, ctx, "s-tvw", "A conjecture in an earning status")
			conj.Status = st
			conj.CorroborationCount = 5
			require.NoError(t, k.SetFact(ctx, conj))

			tvw := k.ComputeTrainingValueWeight(ctx, conj.Id)
			require.Equal(t, uint64(0), tvw.Final,
				"a conjecture must earn zero training value in every status")

			tier, _ := keeper.ClassifyTrainingQuality(conj)
			require.NotEqual(t, types.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD, tier)
			require.NotEqual(t, types.TrainingQualityTier_TRAINING_QUALITY_TIER_SILVER, tier)
		})
	}
}

// TestStanding_RefutationDoorStaysOpenAfterDecay. The PROVISIONAL-only gate on
// MsgChallengeProvisionalFact meant metabolic decay to AT_RISK (~41 epochs,
// nobody acting) permanently closed the mechanism's single paid act.
func TestStanding_RefutationDoorStaysOpenAfterDecay(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	conj := acceptConjecture(t, k, ctx, "s-5", "A conjecture that decayed while nobody looked")
	conj.Status = types.FactStatus_FACT_STATUS_AT_RISK
	require.NoError(t, k.SetFact(ctx, conj))

	_, err := ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{
		Challenger: makeValidBech32Addr("refuter"),
		ClaimId:    "s-5",
		FactId:     conj.Id,
		Stake:      "11000000",
		Reason:     "Here is the counterexample",
	})
	require.NoError(t, err,
		"a decayed conjecture must still be refutable — the door keys on what the fact IS, not on the status it drifted into")

	after, _ := k.GetFact(ctx, conj.Id)
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, after.Status)
}

func TestStanding_RefutationRequiresRealStake(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	conj := acceptConjecture(t, k, ctx, "s-6", "A conjecture attacked on the cheap")

	_, err := ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{
		Challenger: "zrn1cheap",
		ClaimId:    "s-6",
		FactId:     conj.Id,
		Stake:      "1",
		Reason:     "one uzrn buys the status transition",
	})
	require.Error(t, err, "refuting must never be cheaper than asking")
	require.Contains(t, err.Error(), "below minimum")
}

// TestStanding_RefutedConjectureDoesNotPenaliseProposer is COMPASSION C2. The
// first version claimed this property in its commit message and its own code
// violated it: vindication.go recorded a disproval against the proposer.
func TestStanding_RefutedConjectureDoesNotPenaliseProposer(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "s-7", "A conjecture that turns out to be false")

	challenge := &types.Claim{
		Id:                "chal-7",
		FactContent:       "Refutation of s-7",
		Domain:            "mathematics",
		Submitter:         "zrn1refuter",
		Stake:             "11000000",
		Status:            types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ProvisionalFactId: conj.Id,
	}
	require.NoError(t, k.SetClaim(ctx, challenge))
	round := makeRoundInPhase("r-chal-7", "chal-7", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 95)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// ACCEPT on the challenge claim = the conjecture is refuted.
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 850_000,
	}))

	cal, found := k.GetAgentCalibration(ctx, "zrn1proposer")
	if found && cal != nil {
		require.Zero(t, cal.DisprovenCount,
			"being refuted is the mechanism working, not the proposer erring — C2: error is not deceit")
	}

	// The refuter IS credited. That is the whole point of the mechanism.
	refuterCal, refuterFound := k.GetAgentCalibration(ctx, "zrn1refuter")
	require.True(t, refuterFound, "the refuter must be credited for the work")
	require.NotNil(t, refuterCal)
}

// TestStanding_ConjectureNotShippedToTrainers. GatherFrontier's only content
// filter is VerifiedAtBlock == 0, which a conjecture passes — so conjectures
// were shipping inside BundleToK, the headline training product, indexed beside
// verified knowledge.
func TestStanding_ConjectureNotShippedToTrainers(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "s-8", "A conjecture that must not reach a trainer as knowledge")

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id:              "real-8",
		Content:         "An ordinary verified fact in the same domain",
		Domain:          "mathematics",
		Status:          types.FactStatus_FACT_STATUS_VERIFIED,
		Confidence:      880_000,
		ClaimType:       types.ClaimType_CLAIM_TYPE_ASSERTION,
		VerifiedAtBlock: 100,
	}))

	nodeIDs, _, err := k.SelectToKIds(ctx, &types.ToKSelector{
		Variant: &types.ToKSelector_Frontier{Frontier: &types.FrontierSelector{
			Domain:     "mathematics",
			SinceBlock: 0,
			Limit:      100,
		}},
	})
	require.NoError(t, err)
	require.Contains(t, nodeIDs, "real-8", "verified knowledge must still ship")
	require.NotContains(t, nodeIDs, conj.Id,
		"a conjecture must not appear in the chain's headline training extraction — nothing in the payload distinguishes a question from a claim")
}

// TestStanding_DisprovenConjectureStillShips guards TC4 against the fix above:
// a refuted conjecture is an ANSWER, and the graph carries its disprovals.
func TestStanding_DisprovenConjectureStillShips(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "s-9", "A conjecture that was refuted and is now knowledge")
	conj.Status = types.FactStatus_FACT_STATUS_DISPROVEN
	require.NoError(t, k.SetFact(ctx, conj))

	nodeIDs, _, err := k.SelectToKIds(ctx, &types.ToKSelector{
		Variant: &types.ToKSelector_Frontier{Frontier: &types.FrontierSelector{
			Domain: "mathematics", SinceBlock: 0, Limit: 100,
		}},
	})
	require.NoError(t, err)
	require.Contains(t, nodeIDs, conj.Id,
		"TC4: a DISPROVEN conjecture is an answer and must still ship — excluding it would hide a falsification")
}
