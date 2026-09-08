package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestFeedbackChallengeConstructorsPreserveEvidenceAndReason(t *testing.T) {
	for _, conjecture := range []bool{false, true} {
		t.Run(fmt.Sprint(conjecture), func(t *testing.T) {
			k, ctx, _ := setupFeedbackStore(t)
			f := feedbackFact(t, k, ctx, "target")
			if conjecture {
				f.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
				f.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
				f.Confidence = 0
				require.NoError(t, k.SetFact(ctx, f))
			}
			ids := make([]string, 16)
			for i := range ids {
				ids[i] = fmt.Sprintf("evidence-%02d", i)
				feedbackFact(t, k, ctx, ids[i])
			}
			expected := append([]string(nil), ids...)
			reason := "Preserved reason — not a provenance citation. " + strings.Repeat("x", 100)
			m := keeper.NewMsgServerImpl(k)
			var roundID string
			if conjecture {
				r, err := m.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: reason, EvidenceIds: ids, Stake: "100000000"})
				require.NoError(t, err)
				roundID = r.ChallengeId
			} else {
				r, err := m.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: reason, EvidenceIds: ids, Stake: "100000000"})
				require.NoError(t, err)
				roundID = r.RoundId
			}
			ids[0] = "mutated-input"
			round, found := k.GetVerificationRound(ctx, roundID)
			require.True(t, found)
			claim, found := k.GetClaim(ctx, round.ClaimId)
			require.True(t, found)
			require.Equal(t, expected, claim.ChallengeEvidenceIds)
			require.Equal(t, reason, claim.ArgumentText)
			require.Empty(t, claim.References)
			require.Empty(t, claim.Relations)
		})
	}
}

func TestFeedbackChallengeEvidenceAdmissionIsAtomic(t *testing.T) {
	for _, provisional := range []bool{false, true} {
		for _, name := range []string{"missing", "duplicate", "too-many", "bad-id", "empty-reason", "long-reason", "invalid-utf8"} {
			t.Run(fmt.Sprintf("%t-%s", provisional, name), func(t *testing.T) {
				k, ctx, s := setupFeedbackStore(t)
				f := feedbackFact(t, k, ctx, "target")
				if provisional {
					f.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
					f.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
					require.NoError(t, k.SetFact(ctx, f))
				}
				feedbackFact(t, k, ctx, "evidence")
				ids := []string{"evidence"}
				reason := "reason"
				switch name {
				case "missing":
					ids = []string{"missing"}
				case "duplicate":
					ids = append(ids, "evidence")
				case "too-many":
					ids = make([]string, 17)
				case "bad-id":
					ids = []string{"bad/id"}
				case "empty-reason":
					reason = " "
				case "long-reason":
					reason = strings.Repeat("x", 4097)
				case "invalid-utf8":
					reason = string([]byte{255})
				}
				before := feedbackSnapshot(t, s, ctx)
				m := keeper.NewMsgServerImpl(k)
				var err error
				if provisional {
					_, err = m.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: reason, EvidenceIds: ids, Stake: "100000000"})
				} else {
					_, err = m.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: reason, EvidenceIds: ids, Stake: "100000000"})
				}
				require.Error(t, err)
				require.Equal(t, before, feedbackSnapshot(t, s, ctx))
			})
		}
	}
}

func TestFeedbackChallengeTerminalOutcomes(t *testing.T) {
	for _, conjecture := range []bool{false, true} {
		for _, verdict := range []types.Verdict{types.Verdict_VERDICT_ACCEPT, types.Verdict_VERDICT_REJECT, types.Verdict_VERDICT_INCONCLUSIVE, types.Verdict_VERDICT_MALFORMED} {
			t.Run(fmt.Sprintf("%t-%s", conjecture, verdict), func(t *testing.T) {
				k, ctx, _ := setupFeedbackStore(t)
				f := feedbackFact(t, k, ctx, "target")
				f.Energy = 12345
				f.CorroborationCount = 7
				f.LastCorroboratedBlock = 2
				if conjecture {
					f.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
					f.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
					f.Confidence = 0
				}
				require.NoError(t, k.SetFact(ctx, f))
				m := keeper.NewMsgServerImpl(k)
				var roundID string
				if conjecture {
					r, err := m.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: "Check a refutation", Stake: "100000000"})
					require.NoError(t, err)
					roundID = r.ChallengeId
				} else {
					r, err := m.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: f.Id, Reason: "Check the evidence", Stake: "100000000"})
					require.NoError(t, err)
					roundID = r.RoundId
				}
				round, found := k.GetVerificationRound(ctx, roundID)
				require.True(t, found)
				ctx = ctx.WithBlockHeight(15).WithEventManager(sdk.NewEventManager())
				require.NoError(t, k.CompleteRound(ctx, round, &keeper.VerificationResult{Verdict: verdict, Confidence: 800000}))
				updated, found := k.GetFact(ctx, f.Id)
				require.True(t, found)
				if verdict == types.Verdict_VERDICT_ACCEPT {
					require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, updated.Status)
				} else if conjecture {
					require.Equal(t, types.FactStatus_FACT_STATUS_PROVISIONAL, updated.Status)
					require.Zero(t, updated.Confidence)
				} else {
					require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, updated.Status)
				}
				if verdict == types.Verdict_VERDICT_INCONCLUSIVE || verdict == types.Verdict_VERDICT_MALFORMED {
					require.Equal(t, f.Energy, updated.Energy)
					require.Equal(t, f.Confidence, updated.Confidence)
					require.Equal(t, f.CorroborationCount, updated.CorroborationCount)
					require.Equal(t, f.LastCorroboratedBlock, updated.LastCorroboratedBlock)
					history := k.GetStatusHistory(ctx, f.Id)
					require.NotEmpty(t, history)
					last := history[len(history)-1]
					require.Equal(t, round.ClaimId, last.CauseId)
					require.Contains(t, last.CauseEventType, "completed_")
				}
			})
		}
	}
}

func TestFeedbackRestorationWriteFailuresPropagateAtomically(t *testing.T) {
	for _, path := range []string{"completed", "starved"} {
		for _, prefix := range [][]byte{types.FactKeyPrefix, types.StatusTransitionKeyPrefix, types.StatusTransitionSeqKeyPrefix} {
			t.Run(fmt.Sprintf("%s-%x", path, prefix), func(t *testing.T) {
				k, ctx, s := setupFeedbackStore(t)
				feedbackFact(t, k, ctx, "target")
				m := keeper.NewMsgServerImpl(k)
				r, err := m.ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: "target", Reason: "Evidence check", Stake: "100000000"})
				require.NoError(t, err)
				round, found := k.GetVerificationRound(ctx, r.RoundId)
				require.True(t, found)
				before := feedbackSnapshot(t, s, ctx)
				s.op, s.prefix = "set", prefix
				if path == "completed" {
					err = k.CompleteRound(ctx.WithBlockHeight(15), round, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE})
				} else {
					err = k.AdvanceRoundPhases(ctx.WithBlockHeight(int64(round.AggregationDeadline)))
				}
				require.Error(t, err)
				require.Equal(t, before, feedbackSnapshot(t, s, ctx))
			})
		}
	}
}

// Cross-lane regression: the history owner must make RecordStatusTransition
// propagate sequence reads/decoding/overflow. The feedback helper already
// propagates its return value; it cannot repair failures hidden by that API.
func TestFeedbackRestorationSequenceReadFailure(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	feedbackFact(t, k, ctx, "target")
	resp, err := keeper.NewMsgServerImpl(k).ChallengeFact(ctx, &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: "target", Reason: "Evidence check", Stake: "100000000"})
	require.NoError(t, err)
	round, found := k.GetVerificationRound(ctx, resp.RoundId)
	require.True(t, found)
	before := feedbackSnapshot(t, s, ctx)
	s.op, s.prefix = "get", types.StatusTransitionSeqKeyPrefix
	err = k.CompleteRound(ctx.WithBlockHeight(15), round, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE})
	require.ErrorIs(t, err, feedbackStoreErr)
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
}

func TestFeedbackEpochBoundaryConsumesThenResetsAllStatuses(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FitnessEpochBlocks = 10
	p.MetabolismEnergyPerQuery = 50
	p.MetabolismBaseCost = 0
	require.NoError(t, k.SetParams(ctx, p))
	ctx = ctx.WithBlockHeight(9)
	active := &types.Fact{Id: "used", Status: types.FactStatus_FACT_STATUS_ACTIVE, Energy: 10000, QueryCount: 10, QueryCountEpoch: 10, SatisfactionUpEpoch: 2, SatisfactionDownEpoch: 1}
	excluded := proto.Clone(active).(*types.Fact)
	excluded.Id = "excluded"
	excluded.Status = types.FactStatus_FACT_STATUS_DISPROVEN
	require.NoError(t, k.SetFact(ctx, active))
	require.NoError(t, k.SetFact(ctx, excluded))
	require.NoError(t, k.RecordQueryReceipt(ctx, "legacy", "used"))
	require.NoError(t, k.BeginBlocker(ctx))
	got, found := k.GetFact(ctx, active.Id)
	require.True(t, found)
	require.Equal(t, uint64(10), got.QueryCountEpoch)
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(10)))
	got, found = k.GetFact(ctx, active.Id)
	require.True(t, found)
	require.Greater(t, got.Energy, active.Energy, "metabolism must see closed-epoch use before resetting")
	for _, id := range []string{active.Id, excluded.Id} {
		got, found := k.GetFact(ctx, id)
		require.True(t, found)
		require.Zero(t, got.QueryCountEpoch)
		require.Zero(t, got.SatisfactionUpEpoch)
		require.Zero(t, got.SatisfactionDownEpoch)
		require.Equal(t, uint64(10), got.QueryCount)
	}
	require.True(t, k.HasQueryReceipt(ctx, "legacy", "used"), "new epoch path does not scan legacy receipts")
	got.QueryCountEpoch = 3
	require.NoError(t, k.SetFact(ctx, got))
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(11)))
	got, found = k.GetFact(ctx, active.Id)
	require.True(t, found)
	require.Equal(t, uint64(3), got.QueryCountEpoch)
}

func TestFeedbackDiversityClosesPreviousEpoch(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FitnessEpochBlocks = 10
	require.NoError(t, k.SetParams(ctx, p))
	require.NoError(t, k.SetDomain(ctx, &types.Domain{Name: "feedback-domain"}))
	ctx = ctx.WithBlockHeight(9)
	require.NoError(t, k.RecordRoundDiversity(ctx, "before", "feedback-domain", 2, 2))
	// This round actually finalizes in AdvanceRoundPhases inside BeginBlocker
	// at H; it must be indexed in epoch 1 while only epoch 0 is aggregated.
	claim := &types.Claim{Id: "boundary-claim", FactContent: "Boundary completion fixture", Domain: "feedback-domain", Submitter: "boundary-submitter", Stake: "1000000", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
	require.NoError(t, k.SetClaim(ctx, claim))
	round := earlyAggRound("at-boundary", claim.Id, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, 4, 4)
	round.StartedAtBlock, round.CommitDeadline, round.RevealDeadline, round.AggregationDeadline = 1, 9, 10, 11
	for _, commit := range round.Commits {
		commit.CommittedAtBlock = 8
	}
	for _, reveal := range round.Reveals {
		reveal.RevealedAtBlock = 9
	}
	require.NoError(t, k.SetVerificationRound(ctx, round))
	boundary := ctx.WithBlockHeight(10)
	require.NoError(t, k.BeginBlocker(boundary))
	completed, ok := k.GetVerificationRound(boundary, round.Id)
	require.True(t, ok)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, completed.Phase)
	require.Equal(t, uint64(10), completed.VerdictBlock)
	closed, found, err := k.GetDomainDiversity(boundary, "feedback-domain", 0)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(1), closed.RoundCount)
	require.Equal(t, uint64(1000000), closed.AvgEntropy)
	_, found, err = k.GetDomainDiversity(boundary, "feedback-domain", 1)
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, k.RecordRoundDiversity(ctx.WithBlockHeight(11), "after", "feedback-domain", 2, 2))
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(20)))
	closed, found, err = k.GetDomainDiversity(ctx, "feedback-domain", 1)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(2), closed.RoundCount)
}
