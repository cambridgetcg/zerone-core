package cross_stack_test

import (
	"bytes"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	kt "github.com/zerone-chain/zerone/x/knowledge/types"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
)

// TestCorrectionAdmission_HistoryAndTopology characterizes two different public
// admission routes, not a proposed unification of their semantics. Unlike TC4's
// constructed claim/round, every claim, review, Fact and edge here is produced by
// message handlers and AdvanceRoundPhases. No SetClaim/SetFact/SetFactRelation or
// caller-supplied CompleteRound result supplies the connection under test.
//
// This is an in-memory application test: funded synthetic accounts and direct
// message/query calls, not signed transaction transport, independent reviewers,
// a live-network experiment, or proof of external-world truth.
func TestCorrectionAdmission_HistoryAndTopology(t *testing.T) {
	for _, route := range []string{"challenge", "contradiction"} {
		for _, vote := range []string{"accept", "reject"} {
			t.Run(route+"/"+vote, func(t *testing.T) {
				h := NewTestHarness(t)
				t.Cleanup(func() { require.NoError(t, h.App.Close()) })
				ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
				q := knowledgekeeper.NewQueryServerImpl(h.KnowledgeKeeper)
				// Select the current retained, neutral, funded policy explicitly in
				// this isolated fixture; do not rely on a legacy genesis default.
				require.NoError(t, h.KnowledgeKeeper.EnableRecordIntegrity(h.Ctx))
				require.NoError(t, h.KnowledgeKeeper.EnableReviewNeutrality(h.Ctx))
				require.NoError(t, h.KnowledgeKeeper.EnableFundSettlement(h.Ctx))
				params, err := h.KnowledgeKeeper.GetParams(h.Ctx)
				require.NoError(t, err)
				// Shorten deadlines only; retain the normal quorum and funding rules.
				params.CommitPhaseBlocks, params.RevealPhaseBlocks, params.AggregationPhaseBlocks = 2, 2, 1
				require.NoError(t, h.KnowledgeKeeper.SetParams(h.Ctx, params))
				const domain = "correction_admission"
				require.NoError(t, h.KnowledgeKeeper.SetDomain(h.Ctx, &kt.Domain{Name: domain, Status: kt.DomainStatus_DOMAIN_STATUS_ACTIVE}))
				h.App.ZeroneOntologyKeeper.SetDomain(h.Ctx, &ontologytypes.Domain{Name: domain, Status: "active", Stratum: uint32(ontologytypes.StratumEmpirical), Depth: 1})
				author := sdk.AccAddress(bytes.Repeat([]byte{71}, 20))
				challenger := sdk.AccAddress(bytes.Repeat([]byte{72}, 20))
				dependentAuthor := sdk.AccAddress(bytes.Repeat([]byte{76}, 20))
				var reviewers []string
				for _, seed := range []byte{71, 72, 73, 74, 75, 76} {
					addr := sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
					require.NoError(t, h.FundAccount(addr, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 500_000_000))))
					if seed >= 73 && seed <= 75 {
						reviewers = append(reviewers, addr.String())
					}
				}

				// Establish the original and its typed dependent through the same
				// admission/review path, so ClaimHistory has genuine retained roots.
				submitted, err := ms.SubmitClaim(h.Ctx, &kt.MsgSubmitClaim{Submitter: author.String(), FactContent: "The fixture observation holds under condition A.", Domain: domain, Category: "empirical", Stake: params.MinReviewFee})
				require.NoError(t, err)
				originalClaim, found := h.KnowledgeKeeper.GetClaim(h.Ctx, submitted.ClaimId)
				require.True(t, found)
				originalRound := completeCorrectionReview(t, h, originalClaim.VerificationRoundId, reviewers, "accept")
				originalID := knowledgekeeper.GenerateFactID(originalClaim.Id, uint64(h.Ctx.BlockHeight()))
				original, found := h.KnowledgeKeeper.GetFact(h.Ctx, originalID)
				require.True(t, found)
				submitted, err = ms.SubmitClaim(h.Ctx, &kt.MsgSubmitClaim{Submitter: dependentAuthor.String(), FactContent: "The derived fixture observation requires condition A.", Domain: domain, Category: "empirical", Stake: params.MinReviewFee, Relations: []*kt.ClaimRelation{{TargetFactId: originalID, Relation: kt.RelationType_RELATION_TYPE_REQUIRES, Inference: kt.InferenceType_INFERENCE_TYPE_DEDUCTIVE, InferenceStrengthBps: 1_000_000}}})
				require.NoError(t, err)
				dependentClaim, found := h.KnowledgeKeeper.GetClaim(h.Ctx, submitted.ClaimId)
				require.True(t, found)
				completeCorrectionReview(t, h, dependentClaim.VerificationRoundId, reviewers, "accept")
				dependentID := knowledgekeeper.GenerateFactID(dependentClaim.Id, uint64(h.Ctx.BlockHeight()))

				const reason = "A repeated fixture measurement contradicts condition A; scope is synthetic."
				evidence := []string{"https://example.invalid/correction-fixture"}
				stake := knowledgekeeper.EffectiveMinChallengeStake(params, original.Confidence).String()
				balanceBefore := h.GetBalance(challenger, "uzrn")
				var correctionID, roundID, linkField string
				if route == "challenge" {
					response, err := ms.ChallengeFact(h.Ctx, &kt.MsgChallengeFact{Challenger: challenger.String(), FactId: originalID, Stake: stake, Reason: reason, EvidenceIds: evidence, MethodId: "M-COMPUTATIONAL"})
					require.NoError(t, err)
					roundID = response.RoundId
					round, found := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
					require.True(t, found)
					correctionID, linkField = round.ClaimId, "provisional_fact_id"
				} else {
					response, err := ms.SubmitContradiction(h.Ctx, &kt.MsgSubmitContradiction{Submitter: challenger.String(), FactId: originalID, CounterClaim: "The fixture observation fails under condition A.", Category: "empirical", Stake: stake, Reason: reason, EvidenceIds: evidence})
					require.NoError(t, err)
					// Despite its wire name, CounterFactId is the admitted Claim ID.
					correctionID, linkField = response.CounterFactId, "relations.contradicts"
					round, found := h.KnowledgeKeeper.GetRoundByClaimID(h.Ctx, correctionID)
					require.True(t, found)
					roundID = round.Id
				}
				deposit, err := sdk.ParseCoinNormalized(stake + "uzrn")
				require.NoError(t, err)
				require.Equal(t, balanceBefore.Sub(deposit), h.GetBalance(challenger, "uzrn"), "actual test bank debit, not a mocked funding log")
				admitted, found := h.KnowledgeKeeper.GetClaim(h.Ctx, correctionID)
				require.True(t, found)
				require.NotNil(t, admitted.FundingTerms)
				require.Equal(t, reason, admitted.ArgumentText)
				require.Equal(t, evidence, admitted.EvidenceIds)
				require.Empty(t, admitted.ChallengedClaimId)
				pendingTarget, found := h.KnowledgeKeeper.GetFact(h.Ctx, originalID)
				require.True(t, found)
				if route == "challenge" {
					require.Equal(t, kt.FactStatus_FACT_STATUS_CHALLENGED, pendingTarget.Status)
					require.Equal(t, originalID, admitted.ProvisionalFactId)
					require.Empty(t, admitted.Relations, "challenge linkage is not an admitted graph edge")
					require.Equal(t, "M-COMPUTATIONAL", admitted.MethodId)
				} else {
					require.Equal(t, kt.FactStatus_FACT_STATUS_CONTESTED, pendingTarget.Status)
					require.Empty(t, admitted.ProvisionalFactId, "a contradiction is not the challenge-resolution route")
					require.Len(t, admitted.Relations, 1)
					require.Equal(t, originalID, admitted.Relations[0].TargetFactId)
					require.Equal(t, kt.RelationType_RELATION_TYPE_CONTRADICTS, admitted.Relations[0].Relation)
				}

				finished := completeCorrectionReview(t, h, roundID, reviewers, vote)
				history, err := q.ClaimHistory(h.Ctx, &kt.QueryClaimHistoryRequest{Id: originalClaim.Id})
				require.NoError(t, err)
				require.Equal(t, h.Ctx.ChainID(), history.ChainId)
				require.Equal(t, uint64(h.Ctx.BlockHeight()), history.BlockHeight)
				require.Len(t, history.Record.Rounds, 1)
				require.True(t, proto.Equal(originalRound, history.Record.Rounds[0]), "correction must not rewrite the original review")
				require.Len(t, history.RelatedClaims, 1)
				related := history.RelatedClaims[0]
				require.Equal(t, correctionID, related.Record.ClaimId)
				require.Len(t, related.Links, 1)
				require.Equal(t, linkField, related.Links[0].Field)
				require.Equal(t, originalID, related.Links[0].TargetId)
				require.Len(t, related.Record.Rounds, 1)
				require.True(t, proto.Equal(finished, related.Record.Rounds[0]), "history must retain the completed commit/reveal record")
				require.Equal(t, reason, related.Record.Claim.ArgumentText)
				require.Equal(t, evidence, related.Record.Claim.EvidenceIds)
				require.True(t, proto.Equal(admitted.FundingTerms, related.Record.Claim.FundingTerms), "completion preserves admitted funding terms")
				require.Len(t, history.Record.Facts, 1)
				root := history.Record.Facts[0].Fact
				dependent, found := h.KnowledgeKeeper.GetFact(h.Ctx, dependentID)
				require.True(t, found)
				if vote == "accept" {
					require.Equal(t, kt.ClaimStatus_CLAIM_STATUS_ACCEPTED, related.Record.Claim.Status)
					require.Len(t, related.Record.Facts, 1)
					accepted := related.Record.Facts[0]
					require.Equal(t, correctionID, accepted.Fact.ClaimId)
					if route == "challenge" {
						require.Equal(t, kt.FactStatus_FACT_STATUS_DISPROVEN, root.Status)
						require.Equal(t, kt.FactStatus_FACT_STATUS_CONTESTED, dependent.Status)
						require.Empty(t, accepted.OutgoingRelations, "successful challenge does not synthesize CONTRADICTS")
						cascade, err := q.BundleToK(h.Ctx, &kt.QueryBundleToKRequest{Selector: &kt.ToKSelector{Variant: &kt.ToKSelector_CascadeReplay{CascadeReplay: &kt.CascadeReplaySelector{DisprovenFactId: originalID, MaxDepth: 1, IncludeStatusHistory: true}}}})
						require.NoError(t, err)
						require.ElementsMatch(t, []string{originalID, dependentID}, cascade.Bundle.IncludedNodeIds)
						require.NotContains(t, cascade.Bundle.IncludedNodeIds, accepted.Fact.Id, "retained challenge history is not automatically extraction topology")
						require.Len(t, cascade.Bundle.CascadeEvents, 1)
						require.Equal(t, correctionID, cascade.Bundle.CascadeEvents[0].ChallengeClaimId)
						require.Equal(t, dependentID, cascade.Bundle.CascadeEvents[0].DescendantFactId)
						var disproof *kt.StatusTransition
						for _, transition := range cascade.Bundle.StatusHistory {
							if transition.FactId == originalID && transition.NewStatus == kt.FactStatus_FACT_STATUS_DISPROVEN {
								disproof = transition
							}
						}
						require.NotNil(t, disproof, "cascade bundle retains the actual target disproof")
						require.Equal(t, kt.FactStatus_FACT_STATUS_CHALLENGED, disproof.PriorStatus)
						require.Equal(t, uint64(h.Ctx.BlockHeight()), disproof.BlockHeight)
					} else {
						require.Equal(t, kt.FactStatus_FACT_STATUS_CONTESTED, root.Status, "accepted counterclaim alone does not execute challenge disproval")
						require.Equal(t, kt.FactStatus_FACT_STATUS_VERIFIED, dependent.Status)
						require.Len(t, accepted.OutgoingRelations, 1)
						require.Equal(t, accepted.Fact.Id, accepted.OutgoingRelations[0].SourceFactId)
						require.Equal(t, originalID, accepted.OutgoingRelations[0].TargetFactId)
						require.Equal(t, kt.RelationType_RELATION_TYPE_CONTRADICTS, accepted.OutgoingRelations[0].Relation)
					}
				} else {
					require.Equal(t, kt.ClaimStatus_CLAIM_STATUS_REJECTED, related.Record.Claim.Status)
					require.Empty(t, related.Record.Facts, "rejected correction must not create a Fact")
					expected := kt.FactStatus_FACT_STATUS_VERIFIED
					if route == "challenge" {
						expected = kt.FactStatus_FACT_STATUS_ACTIVE
					}
					require.Equal(t, expected, root.Status)
					require.Equal(t, kt.FactStatus_FACT_STATUS_VERIFIED, dependent.Status)
				}
				// The normal subtree follows support-bearing edges, not correction
				// history: both admissions retain the same two-node dependency view.
				subtree, err := q.BundleToK(h.Ctx, &kt.QueryBundleToKRequest{Selector: &kt.ToKSelector{Variant: &kt.ToKSelector_RootedSubtree{RootedSubtree: &kt.RootedSubtreeSelector{RootFactId: originalID, MaxDepth: 1}}}})
				require.NoError(t, err)
				require.ElementsMatch(t, []string{originalID, dependentID}, subtree.Bundle.IncludedNodeIds)
				require.Len(t, subtree.Bundle.IncludedEdges, 1)
				require.Equal(t, dependentID, subtree.Bundle.IncludedEdges[0].FromFactId)
				require.Equal(t, originalID, subtree.Bundle.IncludedEdges[0].ToFactId)
				require.Equal(t, kt.InferenceType_INFERENCE_TYPE_DEDUCTIVE.String(), subtree.Bundle.IncludedEdges[0].Inference)
				require.Equal(t, kt.RelationType_RELATION_TYPE_REQUIRES.String(), subtree.Bundle.IncludedEdges[0].Relation)
			})
		}
	}
}

// completeCorrectionReview drives the real scheme-2 commitment/reveal handlers
// and the normal phase driver. It never edits a retained round or supplies a
// verdict to CompleteRound. Three funded test accounts are not three controllers.
func completeCorrectionReview(t *testing.T, h *TestHarness, roundID string, reviewers []string, vote string) *kt.VerificationRound {
	t.Helper()
	ms := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper)
	round, found := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
	require.True(t, found)
	require.Equal(t, kt.CommitmentSchemeReviewV2, round.CommitmentScheme)
	require.Equal(t, uint32(kt.ReviewPolicyNeutral), round.ReviewPolicyVersion)
	require.Equal(t, kt.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
	att := &kt.ReviewAttestation{Reason: "Reviewed the declared synthetic fixture and its condition A."}
	for i, reviewer := range reviewers {
		salt := []byte(fmt.Sprintf("correction-fixture-salt-%d", i))
		hash, err := kt.ComputeReviewCommitmentV2(round.CommitmentChainId, roundID, reviewer, vote, 0, salt, att)
		require.NoError(t, err)
		_, err = ms.SubmitCommitment(h.Ctx, &kt.MsgSubmitCommitment{Verifier: reviewer, RoundId: roundID, CommitHash: hash})
		require.NoError(t, err)
	}
	h.Ctx = h.Ctx.WithBlockHeight(int64(round.CommitDeadline) + 1)
	require.NoError(t, h.KnowledgeKeeper.AdvanceRoundPhases(h.Ctx))
	for i, reviewer := range reviewers {
		_, err := ms.SubmitReveal(h.Ctx, &kt.MsgSubmitReveal{Verifier: reviewer, RoundId: roundID, Vote: vote, Confidence: 0, Salt: []byte(fmt.Sprintf("correction-fixture-salt-%d", i)), Attestation: att})
		require.NoError(t, err)
	}
	h.Ctx = h.Ctx.WithBlockHeight(int64(round.RevealDeadline) + 1)
	require.NoError(t, h.KnowledgeKeeper.AdvanceRoundPhases(h.Ctx))
	finished, found := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, roundID)
	require.True(t, found)
	require.Equal(t, kt.VerificationPhase_VERIFICATION_PHASE_COMPLETE, finished.Phase)
	expected := kt.Verdict_VERDICT_ACCEPT
	if vote == "reject" {
		expected = kt.Verdict_VERDICT_REJECT
	}
	require.Equal(t, expected, finished.Verdict)
	require.Len(t, finished.Commits, len(reviewers))
	require.Len(t, finished.Reveals, len(reviewers))
	return finished
}
