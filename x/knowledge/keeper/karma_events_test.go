package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── K-alpha karma event tests ──────────────────────────────────────────────
//
// Covers: verifier_rewarded withheld arithmetic, cited edges addressed to the
// author, the corroborate pair on challenge survival, the pending open/settle
// pair including INCONCLUSIVE, the invitation_bonus_unfunded truth event, and
// the commit-seat backstop. (The expiry-path pending_settle, the challenge-
// side pending_open sites, and the capture-override early-aggregation gate
// are pinned in karma_review_fixes_test.go.)

// karmaEventAttrs flattens an event's attributes into a map.
func karmaEventAttrs(ev sdk.Event) map[string]string {
	attrs := make(map[string]string, len(ev.Attributes))
	for _, a := range ev.Attributes {
		attrs[a.Key] = a.Value
	}
	return attrs
}

// karmaEdgesOfKind returns attribute maps for every zerone.karma.edge event
// with the given kind, in emission order.
func karmaEdgesOfKind(events sdk.Events, kind string) []map[string]string {
	var out []map[string]string
	for _, ev := range events {
		if ev.Type != "zerone.karma.edge" {
			continue
		}
		attrs := karmaEventAttrs(ev)
		if attrs["kind"] == kind {
			out = append(out, attrs)
		}
	}
	return out
}

// karmaEventsOfType returns attribute maps for every event of the given type.
func karmaEventsOfType(events sdk.Events, typ string) []map[string]string {
	var out []map[string]string
	for _, ev := range events {
		if ev.Type == typ {
			out = append(out, karmaEventAttrs(ev))
		}
	}
	return out
}

// ─── verifier_rewarded ──────────────────────────────────────────────────────

func TestKarma_VerifierRewarded_WithheldArithmetic(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	verifier := makeValidBech32Addr("karma-verif-1")
	submitter := makeValidBech32Addr("karma-sub-1")

	// Full-conformity history: penalty = IndependenceRewardStrengthBps
	// (default 300_000), so 30% of the pool share is withheld.
	require.NoError(t, k.SetValidatorIndependence(ctx, verifier, keeper.ValidatorIndependenceRecord{
		Validator:  verifier,
		TotalVotes: 10,
		// MinorityVotes: 0 — always voted with the crowd
	}))

	claim := &types.Claim{
		Id:          "claim-karma-vr",
		FactContent: "Withheld arithmetic must be audible on-chain",
		Domain:      "physics",
		Submitter:   submitter,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-vr", "claim-karma-vr", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_ACCEPT,
		Confidence: 850_000,
		Rewards:    []keeper.VerifierReward{{Verifier: verifier, Amount: 3_000_000}},
	}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	// Pool = 55% of the 1_000_000 fee = 550_000; sole verifier takes it all.
	// Modulated = 550_000 × (1 − 1.0 conformity × 0.3 strength) = 385_000.
	rewarded := karmaEventsOfType(ctx.EventManager().Events(), "zerone.knowledge.verifier_rewarded")
	require.Len(t, rewarded, 1)
	require.Equal(t, verifier, rewarded[0]["verifier"])
	require.Equal(t, "r-karma-vr", rewarded[0]["round_id"])
	require.Equal(t, "385000", rewarded[0]["amount_uzrn"])
	require.Equal(t, "165000", rewarded[0]["withheld_uzrn"])

	// The verify recognition edge rides the same payout site.
	edges := karmaEdgesOfKind(ctx.EventManager().Events(), "verify")
	require.Len(t, edges, 1)
	require.Equal(t, verifier, edges[0]["beneficiary"])
	require.Equal(t, submitter, edges[0]["counterparty"])
	require.Equal(t, "RECOGNIZED", edges[0]["state"])
	require.Equal(t, "r-karma-vr", edges[0]["ref_id"])
	require.Equal(t, "physics", edges[0]["domain"])
	require.Equal(t, "true", edges[0]["correct"]) // ACCEPT verdict: rewarded == vote-matched
	require.NotContains(t, edges[0], "self")
	require.NotContains(t, edges[0], "graded") // fail-open bit unavailable at payout — omitted
}

func TestKarma_VerifierRewarded_NewVoterWithholdsNothing(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	verifier := makeValidBech32Addr("karma-verif-2") // no independence record

	claim := &types.Claim{
		Id:          "claim-karma-vr2",
		FactContent: "New voters get the full reward, withheld is zero",
		Domain:      "physics",
		Submitter:   makeValidBech32Addr("karma-sub-2"),
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-vr2", "claim-karma-vr2", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{
		Verdict:    types.Verdict_VERDICT_REJECT,
		Confidence: 0,
		Rewards:    []keeper.VerifierReward{{Verifier: verifier, Amount: 3_000_000}},
	}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	rewarded := karmaEventsOfType(ctx.EventManager().Events(), "zerone.knowledge.verifier_rewarded")
	require.Len(t, rewarded, 1)
	require.Equal(t, "550000", rewarded[0]["amount_uzrn"])
	require.Equal(t, "0", rewarded[0]["withheld_uzrn"])

	edges := karmaEdgesOfKind(ctx.EventManager().Events(), "verify")
	require.Len(t, edges, 1)
	require.Equal(t, "true", edges[0]["correct"]) // REJECT verdict: rewarded == vote-matched
}

// ─── cited ──────────────────────────────────────────────────────────────────

func TestKarma_CitedEdge_AuthorAddressed(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	author := makeValidBech32Addr("karma-author-1")
	citer := makeValidBech32Addr("karma-citer-1")

	target := makeTestFact(t, k, ctx, "fact-karma-cited", "A fact worth citing in later work", "physics", "empirical", author, 800_000)

	claim := &types.Claim{
		Id:          "claim-karma-cite",
		FactContent: "This accepted claim cites the target fact",
		Domain:      "physics",
		Submitter:   citer,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Relations: []*types.ClaimRelation{
			{TargetFactId: target.Id, Relation: types.RelationType_RELATION_TYPE_SUPPORTS},
		},
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-cite", "claim-karma-cite", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT, Confidence: 850_000}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	edges := karmaEdgesOfKind(ctx.EventManager().Events(), "cited")
	require.Len(t, edges, 1)
	require.Equal(t, author, edges[0]["beneficiary"], "cited recognition must address the target fact's author")
	require.Equal(t, citer, edges[0]["counterparty"])
	require.Equal(t, target.Id, edges[0]["ref_id"])
	require.Equal(t, "physics", edges[0]["domain"])
	require.Equal(t, "RECOGNIZED", edges[0]["state"])
	require.NotContains(t, edges[0], "self")

	// The accepted claim also settles its pending pair.
	settles := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_settle")
	require.Len(t, settles, 1)
	require.Equal(t, citer, settles[0]["beneficiary"])
	require.Equal(t, "VERDICT_ACCEPT", settles[0]["verdict"])
}

func TestKarma_CitedEdge_SelfCiteMarked(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	author := makeValidBech32Addr("karma-selfcite-1")
	target := makeTestFact(t, k, ctx, "fact-karma-selfcite", "An author may cite their own fact", "physics", "empirical", author, 800_000)

	claim := &types.Claim{
		Id:          "claim-karma-selfcite",
		FactContent: "Author cites their own earlier fact here",
		Domain:      "physics",
		Submitter:   author,
		Stake:       "1000000",
		Status:      types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		Relations: []*types.ClaimRelation{
			{TargetFactId: target.Id, Relation: types.RelationType_RELATION_TYPE_SUPPORTS},
		},
	}
	require.NoError(t, k.SetClaim(ctx, claim))

	round := makeRoundInPhase("r-karma-selfcite", "claim-karma-selfcite", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT, Confidence: 850_000}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	edges := karmaEdgesOfKind(ctx.EventManager().Events(), "cited")
	require.Len(t, edges, 1, "self-cites are recognized, not skipped")
	require.Equal(t, author, edges[0]["beneficiary"])
	require.Equal(t, author, edges[0]["counterparty"])
	require.Equal(t, "true", edges[0]["self"])
}

// ─── corroborate / corroborated ─────────────────────────────────────────────

func TestKarma_CorroboratePair_OnChallengeSurvival(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	defender := makeValidBech32Addr("karma-defender-1")
	challenger := makeValidBech32Addr("karma-challenger-1")

	fact := makeTestFact(t, k, ctx, "fact-karma-survives", "A fact that survives its challenge", "physics", "empirical", defender, 900_000)

	challenge := &types.Claim{
		Id:                "claim-karma-challenge",
		FactContent:       "This challenge will fail and the fact survives",
		Domain:            "physics",
		Submitter:         challenger,
		Stake:             "1000000",
		Status:            types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ProvisionalFactId: fact.Id,
	}
	require.NoError(t, k.SetClaim(ctx, challenge))

	round := makeRoundInPhase("r-karma-challenge", "claim-karma-challenge", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Confidence: 0}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	events := ctx.EventManager().Events()

	// Defender side: the fact hardened.
	corroborated := karmaEdgesOfKind(events, "corroborated")
	require.Len(t, corroborated, 1)
	require.Equal(t, defender, corroborated[0]["beneficiary"])
	require.Equal(t, challenger, corroborated[0]["counterparty"])
	require.Equal(t, fact.Id, corroborated[0]["ref_id"])
	require.Equal(t, "RECOGNIZED", corroborated[0]["state"])

	// Challenger side: the failed honest challenger is CREDITED (K-7).
	corroborate := karmaEdgesOfKind(events, "corroborate")
	require.Len(t, corroborate, 1)
	require.Equal(t, challenger, corroborate[0]["beneficiary"])
	require.Equal(t, defender, corroborate[0]["counterparty"])
	require.Equal(t, fact.Id, corroborate[0]["ref_id"])
}

func TestKarma_ConjectureSurvivalCarriesNoRecognitionStanding(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	conjecture := acceptConjecture(t, k, ctx, "karma-conjecture", "A question surviving one attempted refutation")
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	challenge := &types.Claim{
		Id:                "claim-karma-conjecture",
		FactContent:       "This attempted refutation fails",
		Domain:            conjecture.Domain,
		Submitter:         makeValidBech32Addr("karma-conjecture-challenger"),
		Stake:             "11000000",
		Status:            types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ProvisionalFactId: conjecture.Id,
	}
	require.NoError(t, k.SetClaim(ctx, challenge))

	round := makeRoundInPhase("r-karma-conjecture", challenge.Id, types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{
		Verdict: types.Verdict_VERDICT_REJECT,
	}))

	events := ctx.EventManager().Events()
	require.Empty(t, karmaEdgesOfKind(events, "corroborated"),
		"a question surviving one refutation must not grant its proposer recognition standing")
	require.Empty(t, karmaEdgesOfKind(events, "corroborate"),
		"a failed refutation of a question must not create a priced-looking recognition pair")

	got, found := k.GetFact(ctx, conjecture.Id)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, got.Status)
	require.Equal(t, uint64(1), got.CorroborationCount,
		"the Popperian survival count remains distinct from calibration and karma standing")
}

// ─── pending_open / pending_settle ──────────────────────────────────────────

func TestKarma_PendingPair_OpenThenSettleInconclusive(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	submitter := makeValidBech32Addr("karma-pending-1")
	bk.balances[submitter] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10_000_000))

	resp, err := ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
		Submitter:   submitter,
		FactContent: "A pending pair opens at submission and settles at verdict",
		Domain:      "",
		Stake:       "1000000",
	})
	require.NoError(t, err)

	opens := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_open")
	require.Len(t, opens, 1)
	require.Equal(t, submitter, opens[0]["beneficiary"])
	require.Equal(t, resp.ClaimId, opens[0]["ref_id"])
	// Pending edges are counters, not recognition: ORDINAL, the first-class
	// "not yet adjudicated" state — a fold-by-state consumer must never sum
	// them into RECOGNIZED totals (design §2.3 row 10).
	require.Equal(t, "ORDINAL", opens[0]["state"])
	require.Equal(t, "", opens[0]["counterparty"])

	// Settle the round INCONCLUSIVE — the pair still settles (K-2), and the
	// verdict class is on the edge.
	round, found := k.GetRoundByClaimID(ctx, resp.ClaimId)
	require.True(t, found)
	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE, Confidence: 0}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	settles := karmaEdgesOfKind(ctx.EventManager().Events(), "pending_settle")
	require.Len(t, settles, 1)
	require.Equal(t, submitter, settles[0]["beneficiary"])
	require.Equal(t, resp.ClaimId, settles[0]["ref_id"])
	require.Equal(t, "VERDICT_INCONCLUSIVE", settles[0]["verdict"])
}

// ─── invitation_bonus_unfunded ──────────────────────────────────────────────

func TestKarma_InvitationBonusUnfunded_EmitsTruthEvent(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	defender := makeValidBech32Addr("karma-invited-def")
	challenger := makeValidBech32Addr("karma-invited-chal")

	// Current invitation: stamped, never superseded by a corroboration.
	fact := makeTestFact(t, k, ctx, "fact-karma-invited", "An invited probe target with an empty pool", "physics", "empirical", defender, 900_000)
	fact.ProbeInvitedAtBlock = 90
	require.NoError(t, k.SetFact(ctx, fact))

	challenge := &types.Claim{
		Id:                "claim-karma-invited",
		FactContent:       "Answering the chain's probe invitation on the fact",
		Domain:            "physics",
		Submitter:         challenger,
		Stake:             "1000000",
		Status:            types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
		ProvisionalFactId: fact.Id,
	}
	require.NoError(t, k.SetClaim(ctx, challenge))

	round := makeRoundInPhase("r-karma-invited", "claim-karma-invited", types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, 80)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	// Probe bounty pool has no balance: the bonus promise cannot be honored.
	result := &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT, Confidence: 0}
	require.NoError(t, k.CompleteRound(ctx, round, result))

	events := ctx.EventManager().Events()
	require.Empty(t, karmaEventsOfType(events, "zerone.knowledge.invitation_bonus_paid"))

	unfunded := karmaEventsOfType(events, "zerone.knowledge.invitation_bonus_unfunded")
	require.Len(t, unfunded, 1, "silent non-payment must become an event")
	require.Equal(t, "claim-karma-invited", unfunded[0]["claim_id"])
	require.Equal(t, fact.Id, unfunded[0]["fact_id"])
	require.Equal(t, "500000", unfunded[0]["amount_uzrn"]) // default InvitationBonusAmount
}

// ─── commit-seat backstop ───────────────────────────────────────────────────

func TestKarma_SeatBackstop_RejectsCommitOverHardCap(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	round := makeRoundInPhase("r-karma-cap", "claim-cap", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	for i := 0; i < types.CommitSeatHardCap; i++ {
		round.Commits = append(round.Commits, &types.CommitEntry{
			Verifier:         makeValidBech32Addr(fmt.Sprintf("karma-seat-%d", i)),
			CommitHash:       []byte(fmt.Sprintf("h%d", i)),
			CommittedAtBlock: 99,
		})
	}
	require.NoError(t, k.SetVerificationRound(ctx, round))

	extra := makeValidBech32Addr("karma-seat-extra")
	bk.balances[extra] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))

	_, err := ms.SubmitCommitment(ctx, &types.MsgSubmitCommitment{
		Verifier:   extra,
		RoundId:    "r-karma-cap",
		CommitHash: computeMsgServerCommitHash("r-karma-cap", "accept", 0, []byte("salt-extra")),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrRoundSeatsFull)

	// The (cap+1)th commit must not have been stored.
	stored, found := k.GetVerificationRound(ctx, "r-karma-cap")
	require.True(t, found)
	require.Len(t, stored.Commits, types.CommitSeatHardCap)
}

// The backstop is deliberately NOT the tight MaxVerifiers seat cap: a round
// holding more commits than MaxVerifiers must keep admitting honest
// verifiers, or MaxVerifiers funded addresses could fill every seat the
// block a round opens, never reveal, and censor every claim on the chain
// (review finding — the tight cap waits for C-2 seat bonds).
func TestKarma_SeatBackstop_MaxVerifiersDoesNotExclude(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 22, params.MaxVerifiers, "default MaxVerifiers expected; prefill below assumes 30 > MaxVerifiers")

	round := makeRoundInPhase("r-karma-nocap", "claim-nocap", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	for i := 0; i < 30; i++ {
		round.Commits = append(round.Commits, &types.CommitEntry{
			Verifier:         makeValidBech32Addr(fmt.Sprintf("karma-nocap-%d", i)),
			CommitHash:       []byte(fmt.Sprintf("h%d", i)),
			CommittedAtBlock: 99,
		})
	}
	require.NoError(t, k.SetVerificationRound(ctx, round))

	extra := makeValidBech32Addr("karma-nocap-extra")
	bk.balances[extra] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))

	_, err = ms.SubmitCommitment(ctx, &types.MsgSubmitCommitment{
		Verifier:   extra,
		RoundId:    "r-karma-nocap",
		CommitHash: computeMsgServerCommitHash("r-karma-nocap", "accept", 0, []byte("salt-extra")),
	})
	require.NoError(t, err, "a seat-full-at-MaxVerifiers round must not exclude further verifiers")

	stored, found := k.GetVerificationRound(ctx, "r-karma-nocap")
	require.True(t, found)
	require.Len(t, stored.Commits, 31)
}
