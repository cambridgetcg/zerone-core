package keeper_test

import (
	"context"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// stubVestingKeeper exists so EscrowSubmitterReward is REACHABLE in these
// tests. It early-returns when vestingRewardsKeeper is nil, and the shared
// harnesses leave it nil — so without this stub a test asserting "no escrow
// was created" would pass whether or not the conjecture branch existed,
// which is a test that proves nothing.
type stubVestingKeeper struct{}

func (stubVestingKeeper) CreateVestingScheduleFromKnowledge(_ context.Context, _, _, _, _, _ string) error {
	return nil
}
func (stubVestingKeeper) DistributeFalsificationReward(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (stubVestingKeeper) GetEpochBlockRewardPool(_ context.Context, _ uint64) uint64 { return 0 }
func (stubVestingKeeper) PauseVestingByClaimId(_ context.Context, _ string) error    { return nil }
func (stubVestingKeeper) PauseAllVestingByRecipient(_ context.Context, _ string) int { return 0 }
func (stubVestingKeeper) DepositToResearchFund(_ context.Context, _ string, _ sdk.Coins) error {
	return nil
}
func (stubVestingKeeper) MintWithCap(_ context.Context, _ string, _ *big.Int) (*big.Int, error) {
	return big.NewInt(0), nil
}

// ─── Conjecture: the chain holds a question open ────────────────────────────
//
// These tests pin the properties that make a conjecture safe to admit, not the
// happy path. The dangerous direction is always the same one: a conjecture
// acquiring standing it never earned. Each test below closes one route by
// which that could happen.

func TestPostConjecture_OpensAndSettlesOrdinalPendingPair(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	proposer := makeValidBech32Addr("conjecture-pending")
	bk.balances[proposer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))

	resp, err := ms.PostConjecture(ctx, &types.MsgPostConjecture{
		Proposer:               proposer,
		Statement:              "There are infinitely many counterexamples beyond the observed range",
		FalsificationPredicate: "Exhibit a proof that the counterexample set is finite",
		Category:               "formal",
		Stake:                  "1000000",
	})
	require.NoError(t, err)

	opens := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_open")
	require.Len(t, opens, 1)
	require.Equal(t, proposer, opens[0]["beneficiary"])
	require.Equal(t, resp.ClaimId, opens[0]["ref_id"])
	require.Equal(t, "ORDINAL", opens[0]["state"],
		"a pending conjecture is a counter, never recognized standing")

	round, found := k.GetVerificationRound(ctx, resp.RoundId)
	require.True(t, found)
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict: types.Verdict_VERDICT_INCONCLUSIVE,
	}))

	settles := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_settle")
	require.Len(t, settles, 1)
	require.Equal(t, resp.ClaimId, settles[0]["ref_id"])
	require.Equal(t, "VERDICT_INCONCLUSIVE", settles[0]["verdict"])
}

// acceptConjecture drives a conjecture claim through a round to ACCEPT and
// returns the resulting fact.
func acceptConjecture(t *testing.T, k keeper.Keeper, ctx sdk.Context, id, content string) *types.Fact {
	t.Helper()

	claim := &types.Claim{
		Id:                     id,
		FactContent:            content,
		Domain:                 "mathematics",
		Category:               "formal",
		Submitter:              "zrn1proposer",
		Stake:                  "100000",
		Status:                 types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ClaimType:              types.ClaimType_CLAIM_TYPE_CONJECTURE,
		FalsificationPredicate: "exhibit a counterexample below 10^12",
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-"+id, id, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// The panel said "well-posed and falsifiable" — note the confidence it
	// returns is deliberately high, to prove the conjecture path ignores it.
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 900_000,
	}))

	var out *types.Fact
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.ClaimId == id {
			out = f
			return true
		}
		return false
	})
	require.NotNil(t, out, "accepted conjecture should create a fact")
	return out
}

func TestConjecture_EntersProvisionalWithZeroConfidence(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	fact := acceptConjecture(t, k, ctx, "c-1", "Every even integer > 2 is the sum of two primes")

	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, fact.Status,
		"a conjecture must not enter as VERIFIED — the panel judged well-posedness, not truth")
	require.Equal(t, uint64(0), fact.Confidence,
		"a conjecture asserts nothing; the panel's 900000 confidence in its WELL-POSEDNESS must not become confidence in its TRUTH")
	require.Equal(t, "exhibit a counterexample below 10^12", fact.FalsificationPredicate,
		"the killer must ride through to the fact so a refuter knows the target")
}

// TestConjecture_MovesTheReviewFeeAndNothingElse asserts on the bank, not on a
// result struct. The framework critique's central lesson was that
// confidence.go computes reward.Amount and rounds.go discards it, and the unit
// tests reinforced the illusion by asserting on the struct that was computed
// rather than the money that moved.
func TestConjecture_MovesTheReviewFeeAndNothingElse(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	k.SetVestingRewardsKeeper(stubVestingKeeper{}) // make the escrow path reachable

	before := len(bk.sendCalls)
	mintedBefore := bk.minted.String()

	fact := acceptConjecture(t, k, ctx, "c-2", "P is not equal to NP")

	// No survival escrow was created — the proposer has no reward pending on
	// any path, so there is nothing to release later and nothing to farm.
	_, pending := k.GetSurvivalPendingReward(ctx, fact.Id)
	require.False(t, pending,
		"a conjecture must escrow no submitter reward — asking a question is not a way to earn")
	require.Equal(t, uint64(0), fact.ChallengeWindowEnd,
		"no challenge window is opened for a conjecture: there is no reward waiting on the other side of one")

	require.Equal(t, mintedBefore, bk.minted.String(),
		"accepting a conjecture must mint nothing")

	// Whatever moved, none of it went TO the proposer.
	for _, call := range bk.sendCalls[before:] {
		require.NotEqual(t, "zrn1proposer", call.to,
			"no value may flow to the proposer of a conjecture; saw %+v", call)
	}
}

// TestOrdinaryClaim_StillEscrows is the control for the test above. Without it,
// a bug that disabled escrow for EVERY claim would leave the conjecture test
// green and look like success.
func TestOrdinaryClaim_StillEscrows(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	k.SetVestingRewardsKeeper(stubVestingKeeper{})

	claim := &types.Claim{
		Id:          "ordinary-claim",
		FactContent: "An ordinary assertion that should escrow a reward",
		Domain:      "mathematics",
		Category:    "formal",
		Submitter:   "zrn1sub",
		Stake:       "100000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ClaimType:   types.ClaimType_CLAIM_TYPE_ASSERTION,
	}
	require.NoError(t, k.SetClaim(ctx, claim))
	round := makeRoundInPhase("r-ordinary", "ordinary-claim", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 850_000,
	}))

	var factID string
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.ClaimId == "ordinary-claim" {
			factID = f.Id
			return true
		}
		return false
	})
	require.NotEmpty(t, factID)

	_, pending := k.GetSurvivalPendingReward(ctx, factID)
	require.True(t, pending,
		"control: an ordinary accepted claim MUST still escrow its submitter reward — if this fails, the conjecture exemption has leaked onto every claim")
}

// TestConjecture_IsExcludedFromTrainingCorpus should pass WITHOUT any change to
// training_corpus.go: PROVISIONAL falls through to the default arm. If this
// ever fails, the assumption the whole design rests on is wrong and the
// conjecture path is leaking unverified propositions into the corpus.
func TestConjecture_IsExcludedFromTrainingCorpus(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	fact := acceptConjecture(t, k, ctx, "c-3", "The Riemann hypothesis holds")

	tier, reason := keeper.ClassifyTrainingQuality(fact)
	require.Equal(t, types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE, tier,
		"a conjecture must never be sold as training data (reason given: %q)", reason)

	tvw := k.ComputeTrainingValueWeight(ctx, fact.Id)
	require.True(t, tvw.StatusIneligible, "a conjecture must be status-ineligible for training value")
	require.Equal(t, uint64(0), tvw.Final, "a conjecture must accrue zero training value")
}

// TestConjecture_CannotBeCited closes the floor-escape route. computeProvenance
// takes the MINIMUM effective confidence across cited edges, and a conjecture
// sits at zero — so citing one alongside a solid fact would compute a
// dependency floor of zero, and a zero floor is read everywhere as "no floor",
// because the clamp is guarded by `floor > 0`. Citing a question would exempt a
// proof chain from the ceiling its real foundations impose.
func TestConjecture_CannotBeCited(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	conj := acceptConjecture(t, k, ctx, "c-4", "There are infinitely many twin primes")

	t.Run("via typed relation", func(t *testing.T) {
		_, err := ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
			Submitter:   "zrn1citer",
			FactContent: "A claim that leans on an open question for support",
			Domain:      "mathematics",
			Stake:       "100000",
			Relations: []*types.ClaimRelation{{
				TargetFactId: conj.Id,
				Relation:     types.RelationType_RELATION_TYPE_SUPPORTS,
			}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "conjecture")
	})

	t.Run("via untyped reference", func(t *testing.T) {
		_, err := ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
			Submitter:   "zrn1citer2",
			FactContent: "Another claim leaning on the same open question",
			Domain:      "mathematics",
			Stake:       "100000",
			References:  []string{conj.Id},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "conjecture")
	})
}

// TestConjecture_SurvivingAChallengeStaysProvisional is the sharpest test here.
// handleChallengeSurvival unconditionally set Status = ACTIVE before this
// change. ACTIVE is citable, enters the training corpus, earns TVW, and is no
// longer reachable by MsgChallengeProvisionalFact — so an attacker could
// launder any unverified proposition into a believed fact by challenging their
// own conjecture and losing on purpose.
func TestConjecture_SurvivingAChallengeStaysProvisional(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "c-5", "Collatz terminates for every positive integer")
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, conj.Status)

	// A challenge is opened against it and then FAILS (REJECT on the
	// challenge claim = the conjecture survived the probe).
	challenge := &types.Claim{
		Id:                "chal-5",
		FactContent:       "Provisional challenge of c-5",
		Domain:            "mathematics",
		Submitter:         "zrn1challenger",
		Stake:             "11000000",
		Status:            types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ProvisionalFactId: conj.Id,
	}
	require.NoError(t, k.SetClaim(ctx, challenge))
	round := makeRoundInPhase("r-chal-5", "chal-5", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 90)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_REJECT,
		Confidence: 0,
	}))

	after, found := k.GetFact(ctx, conj.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, after.Status,
		"surviving an attack is evidence THAT ATTACK failed, not that the conjecture is true; promoting it to ACTIVE would let anyone launder a question into a fact by losing a challenge on purpose")

	// It keeps what it legitimately earned: the hardening credit.
	require.Equal(t, uint64(1), after.CorroborationCount,
		"a survived probe still hardens the conjecture — that is the Popper premium")

	// And it remains attackable, which ACTIVE would have prevented.
	tier, _ := keeper.ClassifyTrainingQuality(after)
	require.Equal(t, types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE, tier,
		"a survived conjecture must still be excluded from the corpus")
}

// TestConjecture_ConfidenceDoesNotGrowOnTimeAlone pins the epistemic_temperature
// exclusion. AdvanceConfidence applies `confidence += confidence * rate` with a
// floor of 1 — so a node at zero climbs one unit per epoch forever on the
// passage of time. For a conjecture that would be the chain manufacturing
// belief in something nobody verified.
func TestConjecture_ConfidenceDoesNotGrowOnTimeAlone(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	conj := acceptConjecture(t, k, ctx, "c-6", "Every odd perfect number exceeds 10^1500")
	require.Equal(t, uint64(0), conj.Confidence)

	for i := 0; i < 25; i++ {
		require.NoError(t, k.AdvanceConfidence(ctx))
	}

	after, found := k.GetFact(ctx, conj.Id)
	require.True(t, found)
	require.Equal(t, uint64(0), after.Confidence,
		"a conjecture's standing may only move when something happens to it — never on the clock")
}

// TestConjecture_VerdictDoesNotTouchCalibration protects C2 (error is not
// deceit). Calibration is accepted-over-decisive and gates the training-fund
// mint; if conjecture verdicts fed it, asking a question the panel could not
// adjudicate would cost the proposer access to an unrelated fund.
func TestConjecture_VerdictDoesNotTouchCalibration(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	_, hadBefore := k.GetAgentCalibration(ctx, "zrn1proposer")
	require.False(t, hadBefore, "precondition: proposer has no calibration record yet")

	// A conjecture the panel returns MALFORMED — the worst outcome available.
	claim := &types.Claim{
		Id:                     "c-7",
		FactContent:            "Consciousness is fundamentally non-computable",
		Domain:                 "general",
		Submitter:              "zrn1proposer",
		Stake:                  "100000",
		Status:                 types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ClaimType:              types.ClaimType_CLAIM_TYPE_CONJECTURE,
		FalsificationPredicate: "exhibit a computable model reproducing it",
	}
	require.NoError(t, k.SetClaim(ctx, claim))
	round := makeRoundInPhase("r-c-7", "c-7", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict: types.Verdict_VERDICT_MALFORMED,
	}))

	_, hadAfter := k.GetAgentCalibration(ctx, "zrn1proposer")
	require.False(t, hadAfter,
		"a conjecture verdict must not enter the calibration ledger — being wrong in a well-posed way costs the fee and nothing else (COMPASSION C2)")

	// And no fact was created.
	var created bool
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.ClaimId == "c-7" {
			created = true
			return true
		}
		return false
	})
	require.False(t, created, "a MALFORMED conjecture must leave no node behind")
}

// TestOpenQuestions_ListsLiveConjecturesOldestFirst pins the ordering. Returning
// IterateFacts order would make `limit` an arbitrary SHA-256-ordered sample
// rather than a prefix — the exact defect FrontierSelector already carries.
func TestOpenQuestions_ListsLiveConjecturesOldestFirst(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	ctx = ctx.WithBlockHeight(100)
	first := acceptConjecture(t, k, ctx, "c-8", "The first open question, posted earliest")
	ctx = ctx.WithBlockHeight(200)
	second := acceptConjecture(t, k, ctx, "c-9", "The second open question, posted later")

	questions, total := k.OpenQuestionsForDomain(ctx, "mathematics", 0)
	require.Equal(t, uint32(2), total)
	require.Len(t, questions, 2)
	require.Equal(t, first.Id, questions[0].FactId, "oldest unanswered question first")
	require.Equal(t, second.Id, questions[1].FactId)
	require.Equal(t, "exhibit a counterexample below 10^12", questions[0].FalsificationPredicate)
	require.False(t, questions[0].UnderChallenge)

	// A domain filter that matches nothing returns nothing, not everything.
	none, noneTotal := k.OpenQuestionsForDomain(ctx, "physics", 0)
	require.Empty(t, none)
	require.Equal(t, uint32(0), noneTotal)
}

// TestOpenQuestions_ExcludesOrdinaryFacts guards against the surface quietly
// becoming a second listing of things the chain believes.
func TestOpenQuestions_ExcludesOrdinaryFacts(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id:         "ordinary-1",
		Content:    "An ordinary verified fact",
		Domain:     "mathematics",
		Status:     types.FactStatus_FACT_STATUS_VERIFIED,
		Confidence: 880_000,
		ClaimType:  types.ClaimType_CLAIM_TYPE_ASSERTION,
	}))

	questions, total := k.OpenQuestionsForDomain(ctx, "", 0)
	require.Equal(t, uint32(0), total)
	require.Empty(t, questions, "open-questions is what the chain does NOT know; a verified fact has no place in it")
}

// TestOpenQuestions_FollowsConjectureTerminality pins the shared definition of
// "open". Status transitions such as metabolic AT_RISK and EXPIRED do not
// settle a conjecture; only the terminal states in IsConjectureResolved do.
func TestOpenQuestions_FollowsConjectureTerminality(t *testing.T) {
	k, ctx, _ := setupKnowledgeTestWithBank(t)

	for _, tc := range []struct {
		id     string
		status types.FactStatus
		open   bool
	}{
		{"provisional", types.FactStatus_FACT_STATUS_PROVISIONAL, true},
		{"challenged", types.FactStatus_FACT_STATUS_CHALLENGED, true},
		{"at-risk", types.FactStatus_FACT_STATUS_AT_RISK, true},
		{"expired", types.FactStatus_FACT_STATUS_EXPIRED, true},
		{"disproven", types.FactStatus_FACT_STATUS_DISPROVEN, false},
		{"pruned", types.FactStatus_FACT_STATUS_PRUNED, false},
		{"revoked", types.FactStatus_FACT_STATUS_REVOKED, false},
	} {
		require.NoError(t, k.SetFact(ctx, &types.Fact{
			Id:                     tc.id,
			Content:                tc.id,
			Domain:                 "mathematics",
			Status:                 tc.status,
			ClaimType:              types.ClaimType_CLAIM_TYPE_CONJECTURE,
			FalsificationPredicate: "exhibit a counterexample",
			VerifiedAtBlock:        10,
		}))
	}

	questions, total := k.OpenQuestionsForDomain(ctx, "mathematics", 0)
	require.Equal(t, uint32(4), total)
	require.Len(t, questions, 4)

	got := make(map[string]bool, len(questions))
	for _, q := range questions {
		got[q.FactId] = true
	}
	for _, id := range []string{"provisional", "challenged", "at-risk", "expired"} {
		require.True(t, got[id], "%s remains challengeable and must stay visible", id)
	}
	for _, id := range []string{"disproven", "pruned", "revoked"} {
		require.False(t, got[id], "%s is terminal and must not be listed", id)
	}
}
