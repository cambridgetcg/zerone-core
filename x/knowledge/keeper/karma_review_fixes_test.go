package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── K-alpha review fixes ───────────────────────────────────────────────────
//
// Pins the fixes applied after the two-reviewer pass:
//   1. The round-expiry path (the one terminal close outside CompleteRound)
//      settles the pending pair.
//   2. Every round-creating message opens a pending pair — challenges and
//      contradictions included — so pending_settle never fires without a
//      matching pending_open.
//   3. Early aggregation is bound by the override-inclusive effective
//      minimum at the advance height: an active capture-challenge override
//      holds the advance; an expired one releases it.
//   4. A cited fact with no recorded submitter yields no edge (empty
//      beneficiary guard, matched with the substrate_bridge emitter).
//   5. Every karma edge carries the register confession.

// ─── 1. expiry-path pending_settle ──────────────────────────────────────────

func TestKarmaReview_PendingSettle_OnRoundExpiry(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	submitter := makeValidBech32Addr("karma-expiry-sub")
	claim := &types.Claim{
		Id:          "claim-karma-expiry",
		FactContent: "A starved round settles its pending pair on expiry",
		Domain:      "physics",
		Submitter:   submitter,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	// Round with zero reveals; deadlines from makeRoundInPhase:
	// commit 298, reveal 498, aggregation 548.
	round := makeRoundInPhase("r-karma-expiry", "claim-karma-expiry", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 98)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	ctx = ctx.WithBlockHeight(int64(round.AggregationDeadline)).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	stored, found := k.GetVerificationRound(ctx, "r-karma-expiry")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_EXPIRED, stored.Phase)
	require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, stored.Verdict)

	storedClaim, found := k.GetClaim(ctx, "claim-karma-expiry")
	require.True(t, found)
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, storedClaim.Status)

	// The pair settles on this route too — previously it stayed open forever.
	settles := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_settle")
	require.Len(t, settles, 1, "the expiry path must settle the pending pair")
	require.Equal(t, submitter, settles[0]["beneficiary"])
	require.Equal(t, "claim-karma-expiry", settles[0]["ref_id"])
	require.Equal(t, "physics", settles[0]["domain"])
	require.Equal(t, "ORDINAL", settles[0]["state"])
	require.Equal(t, "VERDICT_INCONCLUSIVE", settles[0]["verdict"])
	require.Equal(t, "priced-coherence", settles[0]["register"])
}

// ─── 2. challenge/contradiction pending_open ────────────────────────────────

func TestKarmaReview_PendingOpen_OnChallengeFact(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	defender := makeValidBech32Addr("karma-open-def")
	challenger := makeValidBech32Addr("karma-open-chal")
	fact := makeTestFact(t, k, ctx, "fact-karma-open", "A fact about to be challenged", "physics", "empirical", defender, 900_000)
	bk.balances[challenger] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))

	resp, err := ms.ChallengeFact(ctx, &types.MsgChallengeFact{
		FactId:     fact.Id,
		Challenger: challenger,
		Stake:      "11000000",
		Reason:     "probing the fact",
	})
	require.NoError(t, err)

	opens := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_open")
	require.Len(t, opens, 1, "a challenge opens a pending pair like any claim")
	require.Equal(t, challenger, opens[0]["beneficiary"])
	require.Equal(t, "physics", opens[0]["domain"])
	require.Equal(t, "ORDINAL", opens[0]["state"])

	// The pair balances: complete the challenge round and fold.
	round, found := k.GetVerificationRound(ctx, resp.RoundId)
	require.True(t, found)
	round.Phase = types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
	require.NoError(t, k.SetVerificationRound(ctx, round))
	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Confidence: 0}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	settles := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_settle")
	require.Len(t, settles, 1)
	require.Equal(t, challenger, settles[0]["beneficiary"])
	require.Equal(t, opens[0]["ref_id"], settles[0]["ref_id"], "settle must close the pair the challenge opened")
}

func TestKarmaReview_PendingOpen_OnChallengeProvisionalFact(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	defender := makeValidBech32Addr("karma-openprov-def")
	challenger := makeValidBech32Addr("karma-openprov-chal")
	fact := makeTestFact(t, k, ctx, "fact-karma-openprov", "A provisional fact about to be challenged", "physics", "empirical", defender, 700_000)
	fact.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
	require.NoError(t, k.SetFact(ctx, fact))
	bk.balances[challenger] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))

	_, err := ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{
		FactId:     fact.Id,
		Challenger: challenger,
		Stake:      "11000000",
		Reason:     "probing the provisional fact",
	})
	require.NoError(t, err)

	opens := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_open")
	require.Len(t, opens, 1)
	require.Equal(t, challenger, opens[0]["beneficiary"])
	require.Equal(t, "ORDINAL", opens[0]["state"])
}

func TestKarmaReview_PendingOpen_OnSubmitContradiction(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	author := makeValidBech32Addr("karma-openctr-author")
	contradictor := makeValidBech32Addr("karma-openctr-sub")
	fact := makeTestFact(t, k, ctx, "fact-karma-openctr", "A fact about to be contradicted", "physics", "empirical", author, 800_000)
	bk.balances[contradictor] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))

	resp, err := ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{
		FactId:       fact.Id,
		Submitter:    contradictor,
		CounterClaim: "The opposite is the case",
		Category:     "empirical",
		Stake:        "1000000",
	})
	require.NoError(t, err)

	opens := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_open")
	require.Len(t, opens, 1)
	require.Equal(t, contradictor, opens[0]["beneficiary"])
	require.Equal(t, resp.CounterFactId, opens[0]["ref_id"])
	require.Equal(t, "physics", opens[0]["domain"], "domain falls back to the target fact's domain")
	require.Equal(t, "ORDINAL", opens[0]["state"])
}

// ─── 3. capture-challenge override binds early aggregation ──────────────────

// An active override raises the effective minimum above the commit count, so
// the fast-revealing panel cannot pick its aggregation height: the round
// holds in REVEAL and finalizes at the reveal deadline under exactly the
// threshold a fixed-deadline chain would have applied.
func TestKarmaReview_CaptureOverrideHoldsEarlyAggregation(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	earlyAggClaim(t, k, ctx, "claim-early-override")
	// 4 commits / 4 reveals: enough for the domain baseline (base+1 = 4)
	// but not for baseline + override (+2 = 6).
	round := earlyAggRound("round-early-override", "claim-early-override", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// Override active well past the reveal deadline (200).
	require.NoError(t, k.IncreaseVerificationThreshold(ctx, "physics", 2, 300))

	// Inside the reveal window: the advance is held, nothing is emitted.
	for h := int64(160); h < 200; h += 13 {
		ctx = ctx.WithBlockHeight(h).WithEventManager(sdk.NewEventManager())
		require.NoError(t, k.AdvanceRoundPhases(ctx))

		held, found := k.GetVerificationRound(ctx, "round-early-override")
		require.True(t, found)
		require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, held.Phase,
			"override must hold the early advance at height %d", h)
		require.Empty(t, ctx.EventManager().Events(), "held round emitted at height %d", h)
	}

	// At the reveal deadline the round aggregates under the override —
	// INCONCLUSIVE, exactly as under fixed deadlines.
	ctx = ctx.WithBlockHeight(200).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	done, found := k.GetVerificationRound(ctx, "round-early-override")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, done.Verdict,
		"4 reveals under an effective minimum of 6 must not reach a verdict")
}

// Once the override expires the early advance releases — before the reveal
// deadline — and the round reaches its decisive verdict.
func TestKarmaReview_ExpiredOverrideReleasesEarlyAggregation(t *testing.T) {
	k, ctx := setupKnowledgeKeeper(t)

	earlyAggClaim(t, k, ctx, "claim-early-release")
	round := earlyAggRound("round-early-release", "claim-early-release", types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// Override expires at height 170, well inside the reveal window.
	require.NoError(t, k.IncreaseVerificationThreshold(ctx, "physics", 2, 170))

	ctx = ctx.WithBlockHeight(165).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))
	held, found := k.GetVerificationRound(ctx, "round-early-release")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, held.Phase)

	ctx = ctx.WithBlockHeight(171).WithEventManager(sdk.NewEventManager())
	require.NoError(t, k.AdvanceRoundPhases(ctx))

	done, found := k.GetVerificationRound(ctx, "round-early-release")
	require.True(t, found)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, done.Verdict)
	require.Less(t, done.VerdictBlock, round.RevealDeadline,
		"after the override lapses the verdict still lands before the reveal deadline")
}

// ─── 4. empty-submitter cited guard ─────────────────────────────────────────

func TestKarmaReview_CitedEdge_EmptySubmitterSkipped(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// A fact with no recorded submitter (reachable only via state import).
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id:               "fact-karma-nosub",
		Content:          "An imported fact with no submitter",
		Domain:           "physics",
		Category:         "empirical",
		Confidence:       800_000,
		Status:           types.FactStatus_FACT_STATUS_VERIFIED,
		SubmittedAtBlock: 1,
	}))

	citer := makeValidBech32Addr("karma-nosub-citer")
	claim := &types.Claim{
		Id:          "claim-karma-nosub",
		FactContent: "This accepted claim cites the submitterless fact",
		Domain:      "physics",
		Submitter:   citer,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Relations: []*types.ClaimRelation{
			{TargetFactId: "fact-karma-nosub", Relation: types.RelationType_RELATION_TYPE_SUPPORTS},
		},
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-nosub", "claim-karma-nosub", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT, Confidence: 850_000}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	require.Empty(t, karmaEdgesOfKind(ctx.EventManager().Events(), "cited"),
		"an empty beneficiary would be an unattributable credit — no edge")
}

// ─── 5. the register confession rides every edge ────────────────────────────

func TestKarmaReview_RegisterRidesEveryEdge(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	author := makeValidBech32Addr("karma-reg-author")
	citer := makeValidBech32Addr("karma-reg-citer")
	target := makeTestFact(t, k, ctx, "fact-karma-reg", "A cited fact for the register test", "physics", "empirical", author, 800_000)

	claim := &types.Claim{
		Id:          "claim-karma-reg",
		FactContent: "An accepted claim whose edges all carry the register",
		Domain:      "physics",
		Submitter:   citer,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Relations: []*types.ClaimRelation{
			{TargetFactId: target.Id, Relation: types.RelationType_RELATION_TYPE_SUPPORTS},
		},
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-reg", "claim-karma-reg", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	verifier := makeValidBech32Addr("karma-reg-verif")
	result := &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 850_000,
		Rewards:    []keeper.VerifierReward{{Verifier: verifier, Amount: 3_000_000}},
	}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	var edges int
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != "zerone.karma.edge" {
			continue
		}
		edges++
		attrs := karmaEventAttrs(ev)
		require.Equal(t, "priced-coherence", attrs["register"],
			"every karma edge carries the circularity confession (kind %s)", attrs["kind"])
	}
	require.GreaterOrEqual(t, edges, 3, "expected verify, cited, and pending_settle edges")
}
