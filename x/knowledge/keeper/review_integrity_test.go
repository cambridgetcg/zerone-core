package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestRecordReviewCopiedCommitmentCannotBeRevealedByOtherSigner(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	ctx = ctx.WithChainID("origin-chain")
	ms := keeper.NewMsgServerImpl(k)
	a, b := makeValidBech32Addr("reviewer-A"), makeValidBech32Addr("reviewer-B")
	for _, v := range []string{a, b} {
		bk.balances[v] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 200_000_000))
	}
	round := makeRoundInPhase("bound-round", "claim-bound", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	round.CommitmentScheme = 2
	round.CommitmentChainId = "origin-chain"
	require.NoError(t, k.SetVerificationRound(ctx, round))
	att := &types.ReviewAttestation{Reason: "I ran the date-parser fixture α.", Scope: "Fixture only", EvidenceIds: []string{"sha256:public-fixture"}}
	salt := []byte("0123456789abcdef")
	hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, a, "accept", 800000, salt, att)
	require.NoError(t, err)
	for _, v := range []string{a, b} {
		_, err = ms.SubmitCommitment(ctx, &types.MsgSubmitCommitment{Verifier: v, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
	round, _ = k.GetVerificationRound(ctx, round.Id)
	round.Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
	require.NoError(t, k.SetVerificationRound(ctx, round))
	before := proto.Clone(round)
	events := len(ctx.EventManager().Events())
	_, err = ms.SubmitReveal(ctx, &types.MsgSubmitReveal{Verifier: b, RoundId: round.Id, Vote: "accept", Confidence: 800000, Salt: salt, Attestation: att})
	require.ErrorIs(t, err, types.ErrRevealMismatch)
	after, _ := k.GetVerificationRound(ctx, round.Id)
	require.True(t, proto.Equal(before, after))
	require.Len(t, ctx.EventManager().Events(), events)
	// Imported rounds keep their original preimage chain, even under a new tx chain.
	ctx = ctx.WithChainID("importing-chain")
	_, err = ms.SubmitReveal(ctx, &types.MsgSubmitReveal{Verifier: a, RoundId: round.Id, Vote: "accept", Confidence: 800000, Salt: salt, Attestation: att})
	require.NoError(t, err)
	after, _ = k.GetVerificationRound(ctx, round.Id)
	require.Len(t, after.Reveals, 1)
	require.Equal(t, uint64(800000), after.Reveals[0].Confidence)
	require.True(t, proto.Equal(att, after.Reveals[0].Attestation))
	require.Equal(t, "origin-chain", after.CommitmentChainId)
	for _, field := range []string{"scheme", "chain", "reason", "remove"} {
		t.Run(field, func(t *testing.T) {
			changed := proto.Clone(after).(*types.VerificationRound)
			switch field {
			case "scheme":
				changed.CommitmentScheme = 0
			case "chain":
				changed.CommitmentChainId = "other"
			case "reason":
				changed.Reveals[0].Attestation.Reason = "replacement"
			case "remove":
				changed.Reveals = nil
			}
			require.Error(t, k.SetVerificationRound(ctx, changed))
			current, _ := k.GetVerificationRound(ctx, round.Id)
			require.True(t, proto.Equal(after, current))
		})
	}
}

func TestRecordReviewLegacyRoundRetainsOriginalSchemeAfterActivation(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	a := makeValidBech32Addr("legacy-reviewer")
	salt := []byte("old-short-salt")
	round := makeRoundInPhase("legacy-round", "legacy-claim", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	require.NoError(t, k.SetVerificationRound(ctx, round))
	hash := types.ComputeCommitmentHash(round.Id, "accept", 123, salt)
	require.NoError(t, k.StoreCommitmentInRound(ctx, round.Id, &types.CommitEntry{Verifier: a, CommitHash: hash}))
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	round, _ = k.GetVerificationRound(ctx, round.Id)
	round.Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
	require.NoError(t, k.SetVerificationRound(ctx, round))
	require.Error(t, k.StoreRevealInRound(ctx, round.Id, &types.RevealEntry{Verifier: a, Vote: "accept", Salt: salt, Attestation: &types.ReviewAttestation{Reason: "unbound"}}, 123))
	require.NoError(t, k.StoreRevealInRound(ctx, round.Id, &types.RevealEntry{Verifier: a, Vote: "accept", Salt: salt}, 123))
	after, _ := k.GetVerificationRound(ctx, round.Id)
	require.Zero(t, after.CommitmentScheme)
	require.Zero(t, after.Reveals[0].Confidence)
	require.Nil(t, after.Reveals[0].Attestation)
}

func TestRecordClaimRetainsSignedMethodReasonAndRejectsUnboundInputs(t *testing.T) {
	k, ctx, bk := setupKnowledgeTestWithBank(t)
	ms := keeper.NewMsgServerImpl(k)
	msg := &types.MsgSubmitClaim{Submitter: makeValidBech32Addr("record-author"), FactContent: "A public date parser accepts the checked leap-day input.", Domain: "physics", Category: "computational", Stake: "1000000", MethodId: "M-COMPUTATIONAL", ReasoningTrace: "Exact submitted reason α."}
	_, err := ms.SubmitClaim(ctx, msg)
	require.Error(t, err)
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	ctx = ctx.WithChainID("record-chain")
	response, err := ms.SubmitClaim(ctx, msg)
	require.NoError(t, err)
	claim, found := k.GetClaim(ctx, response.ClaimId)
	require.True(t, found)
	require.Equal(t, msg.MethodId, claim.MethodId)
	require.Equal(t, msg.ReasoningTrace, claim.ReasoningTrace)
	round, found := k.GetRoundByClaimID(ctx, claim.Id)
	require.True(t, found)
	require.Equal(t, uint32(2), round.CommitmentScheme)
	require.Equal(t, "record-chain", round.CommitmentChainId)
	for _, name := range []string{"conjecture", "unknown-type", "unknown-category", "doctrine", "dangling-reference", "unknown-method", "huge-fee", "unknown-field"} {
		t.Run(name, func(t *testing.T) {
			bad := proto.Clone(msg).(*types.MsgSubmitClaim)
			bad.FactContent += " " + name
			switch name {
			case "conjecture":
				bad.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
			case "unknown-type":
				bad.ClaimType = types.ClaimType(100)
			case "unknown-category":
				bad.Category = "made-up"
			case "doctrine":
				bad.MethodId = types.DoctrineMethodId
			case "dangling-reference":
				bad.References = []string{"missing"}
			case "unknown-method":
				bad.MethodId = "M-NOT-REGISTERED"
			case "huge-fee":
				bad.Stake = "18446744073709551616"
			case "unknown-field":
				bad.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1})
			}
			calls := len(bk.sendCalls)
			events := len(ctx.EventManager().Events())
			_, err := ms.SubmitClaim(ctx, bad)
			require.Error(t, err)
			require.Len(t, bk.sendCalls, calls)
			require.Len(t, ctx.EventManager().Events(), events)
		})
	}
}

func TestRecordHistoricalRoundUpdatePreservesSelectedClaimIndex(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	first := makeRoundInPhase("historical-first", "historical-claim", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	second := makeRoundInPhase("historical-second", "historical-claim", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	require.NoError(t, k.SetVerificationRound(ctx, first))
	require.NoError(t, k.SetVerificationRound(ctx, second))
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	first.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
	require.NoError(t, k.SetVerificationRound(ctx, first))
	selected, found := k.GetRoundByClaimID(ctx, first.ClaimId)
	require.True(t, found)
	require.Equal(t, second.Id, selected.Id)
	third := makeRoundInPhase("new-third", first.ClaimId, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)
	require.Error(t, k.SetVerificationRound(ctx, third))
}

func TestRecordChallengesRetainExactSignedEvidence(t *testing.T) {
	for _, provisional := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "provisional"}[provisional], func(t *testing.T) {
			k, ctx, bk := setupKnowledgeTestWithBank(t)
			require.NoError(t, k.EnableRecordIntegrity(ctx))
			ctx = ctx.WithChainID("challenge-chain")
			ms := keeper.NewMsgServerImpl(k)
			fact := makeTestFact(t, k, ctx, "record-challenge-target", "Public fixture target for a reasoned challenge.", "physics", "empirical", makeValidBech32Addr("target-author"), 800000)
			fact.ClaimId = "original-claim"
			if provisional {
				fact.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
			}
			require.NoError(t, k.SetFact(ctx, fact))
			challenger := makeValidBech32Addr("record-challenger")
			reason := "Exact counterexample α, with declared limits."
			refs := []string{"sha256:first", "https://example.invalid/public-fixture"}
			var roundID string
			if provisional {
				bad := &types.MsgChallengeProvisionalFact{Challenger: challenger, FactId: fact.Id, ClaimId: "wrong", Stake: "19800000", Reason: reason, MethodId: "M-COMPUTATIONAL", EvidenceIds: refs, CounterClaim: "The boundary input is rejected."}
				calls := len(bk.sendCalls)
				_, err := ms.ChallengeProvisionalFact(ctx, bad)
				require.Error(t, err)
				require.Len(t, bk.sendCalls, calls)
				bad.ClaimId = fact.ClaimId
				response, err := ms.ChallengeProvisionalFact(ctx, bad)
				require.NoError(t, err)
				roundID = response.ChallengeId
			} else {
				response, err := ms.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: challenger, FactId: fact.Id, Stake: "19800000", Reason: reason, MethodId: "M-COMPUTATIONAL", EvidenceIds: refs})
				require.NoError(t, err)
				roundID = response.RoundId
			}
			round, found := k.GetVerificationRound(ctx, roundID)
			require.True(t, found)
			claim, found := k.GetClaim(ctx, round.ClaimId)
			require.True(t, found)
			require.Equal(t, reason, claim.ArgumentText)
			require.Equal(t, refs, claim.EvidenceIds)
			require.Equal(t, "M-COMPUTATIONAL", claim.MethodId)
			require.Equal(t, fact.Id, claim.ProvisionalFactId)
			if provisional {
				require.Equal(t, "original-claim", claim.ChallengedClaimId)
				require.Equal(t, "The boundary input is rejected.", claim.CounterClaim)
			}
		})
	}
}
