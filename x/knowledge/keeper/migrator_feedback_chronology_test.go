package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The earlier REJECT uses the actual submission/completion handlers. Only the
// later terminal prestate is historical: old completion stranded its target.
func feedbackRejectedThenStranded(t *testing.T, rejects int, verdict types.Verdict) (keeper.Keeper, sdk.Context, *feedbackStoreService, *types.Claim, *types.VerificationRound, *types.Claim, *types.VerificationRound) {
	t.Helper()
	k, ctx, s := setupFeedbackStore(t)
	m := keeper.NewMsgServerImpl(k)
	feedbackFact(t, k, ctx, "target")
	var rejected *types.Claim
	var rejectedRound *types.VerificationRound
	for i := 0; i < rejects; i++ {
		height := int64(2 + i*2)
		first, err := m.ChallengeFact(ctx.WithBlockHeight(height), &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: "target", Reason: fmt.Sprintf("older challenge %d", i), Stake: "100000000"})
		require.NoError(t, err)
		var found bool
		rejectedRound, found = k.GetVerificationRound(ctx, first.RoundId)
		require.True(t, found)
		require.NoError(t, k.CompleteRound(ctx.WithBlockHeight(height+1), rejectedRound, &keeper.VerificationResult{Verdict: types.Verdict_VERDICT_REJECT}))
		rejected, found = k.GetClaim(ctx, rejectedRound.ClaimId)
		require.True(t, found)
		require.Equal(t, types.ClaimStatus_CLAIM_STATUS_REJECTED, rejected.Status)
	}
	second, err := m.ChallengeFact(ctx.WithBlockHeight(11), &types.MsgChallengeFact{Challenger: feedbackConsumer(1), FactId: "target", Reason: "later stranded challenge", Stake: "100000000"})
	require.NoError(t, err)
	strandedRound, found := k.GetVerificationRound(ctx, second.RoundId)
	require.True(t, found)
	stranded, found := k.GetClaim(ctx, strandedRound.ClaimId)
	require.True(t, found)
	stranded.Status = types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT
	if verdict == types.Verdict_VERDICT_MALFORMED {
		stranded.Status = types.ClaimStatus_CLAIM_STATUS_MALFORMED
	}
	strandedRound.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
	strandedRound.Verdict = verdict
	strandedRound.VerdictBlock = 12
	require.NoError(t, k.SetClaim(ctx, stranded))
	require.NoError(t, k.SetVerificationRound(ctx, strandedRound))
	return k, ctx, s, rejected, rejectedRound, stranded, strandedRound
}

func TestToKMigrationPriorRejectedChallengeChronology(t *testing.T) {
	for _, verdict := range []types.Verdict{types.Verdict_VERDICT_INCONCLUSIVE, types.Verdict_VERDICT_MALFORMED} {
		for _, rejects := range []int{1, 3} {
			t.Run(fmt.Sprintf("%s/%d_rejects", verdict, rejects), func(t *testing.T) {
				k, ctx, s, rejected, rejectedRound, stranded, strandedRound := feedbackRejectedThenStranded(t, rejects, verdict)
				beforeFact, found := k.GetFact(ctx, "target")
				require.True(t, found)
				require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, beforeFact.Status)
				beforeHistory := k.GetStatusHistory(ctx, "target")
				beforeCompleted := k.CountCompletedRoundsInWindow(ctx, 20, 20)
				require.EqualValues(t, rejects, beforeCompleted)
				require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)))
				afterFact, found := k.GetFact(ctx, "target")
				require.True(t, found)
				want := proto.Clone(beforeFact).(*types.Fact)
				want.Status = types.FactStatus_FACT_STATUS_ACTIVE
				want.AtRiskSinceEpoch = 0
				require.True(t, proto.Equal(want, afterFact), "repair must grant no energy, confidence, corroboration, survival or other fact credit")
				for _, before := range []*types.Claim{rejected, stranded} {
					after, found := k.GetClaim(ctx, before.Id)
					require.True(t, found)
					require.True(t, proto.Equal(before, after), "retained claims must not be rewritten")
				}
				for _, before := range []*types.VerificationRound{rejectedRound, strandedRound} {
					after, found := k.GetVerificationRound(ctx, before.Id)
					require.True(t, found)
					require.True(t, proto.Equal(before, after), "retained rounds must not be rewritten")
				}
				afterHistory := k.GetStatusHistory(ctx, "target")
				require.Len(t, afterHistory, len(beforeHistory)+1)
				transition := afterHistory[len(afterHistory)-1]
				require.Equal(t, "migration_tok_feedback_v1_terminal_challenge", transition.CauseEventType)
				require.Equal(t, stranded.Id, transition.CauseId)
				require.EqualValues(t, 20, transition.BlockHeight)
				require.Equal(t, beforeCompleted, k.CountCompletedRoundsInWindow(ctx, 20, 20), "migration must not invent completion metadata")
				beforeReplay := feedbackSnapshot(t, s, ctx)
				require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(21)))
				require.Equal(t, beforeReplay, feedbackSnapshot(t, s, ctx))
			})
		}
	}
}

func TestToKMigrationRejectedChallengeAmbiguitiesStayBlocked(t *testing.T) {
	for _, scenario := range []string{
		"live-round", "missing-round", "multiple-rejected-rounds", "multiple-stranded-rounds", "missing-rejected-claim",
		"pending-claim", "accepted-claim", "mismatched-verdict", "expired-reject", "unknown-constructor",
		"missing-rejected-round-reference", "wrong-rejected-round-reference", "missing-rejected-submission", "missing-rejected-start",
		"mismatched-rejected-start", "missing-rejected-verdict-height", "rejection-before-start",
		"rejection-at-strand-start", "rejection-after-strand-start", "rejection-after-strand-completion", "future-rejection",
		"missing-stranded-submission", "missing-stranded-start", "mismatched-stranded-start", "missing-stranded-round-reference",
		"wrong-stranded-round-reference", "missing-stranded-verdict-height", "stranded-verdict-before-start", "future-stranded-verdict",
		"another-live-challenge", "live-contradiction", "orphan-live-round",
	} {
		t.Run(scenario, func(t *testing.T) {
			k, ctx, s, rejected, r1, stranded, r2 := feedbackRejectedThenStranded(t, 1, types.Verdict_VERDICT_INCONCLUSIVE)
			switch scenario {
			case "live-round":
				r1.Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
			case "pending-claim":
				rejected.Status = types.ClaimStatus_CLAIM_STATUS_PENDING
			case "accepted-claim":
				rejected.Status = types.ClaimStatus_CLAIM_STATUS_ACCEPTED
			case "mismatched-verdict":
				r1.Verdict = types.Verdict_VERDICT_INCONCLUSIVE
			case "expired-reject":
				r1.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
			case "unknown-constructor":
				rejected.FactContent = "Not a proven challenge constructor"
			case "missing-rejected-round-reference":
				rejected.VerificationRoundId = ""
			case "wrong-rejected-round-reference":
				rejected.VerificationRoundId = "absent"
			case "missing-rejected-submission":
				rejected.SubmittedAtBlock = 0
			case "missing-rejected-start":
				r1.StartedAtBlock = 0
			case "mismatched-rejected-start":
				r1.StartedAtBlock++
			case "missing-rejected-verdict-height":
				r1.VerdictBlock = 0
			case "rejection-before-start":
				r1.VerdictBlock = 1
			case "rejection-at-strand-start":
				r1.VerdictBlock = 11
			case "rejection-after-strand-start":
				r1.VerdictBlock = 12
			case "rejection-after-strand-completion":
				r1.VerdictBlock = 13
			case "future-rejection":
				r1.VerdictBlock = 21
			case "missing-stranded-submission":
				stranded.SubmittedAtBlock = 0
			case "missing-stranded-start":
				r2.StartedAtBlock = 0
			case "mismatched-stranded-start":
				r2.StartedAtBlock++
			case "missing-stranded-round-reference":
				stranded.VerificationRoundId = ""
			case "wrong-stranded-round-reference":
				stranded.VerificationRoundId = "absent"
			case "missing-stranded-verdict-height":
				r2.VerdictBlock = 0
			case "stranded-verdict-before-start":
				r2.VerdictBlock = 10
			case "future-stranded-verdict":
				r2.VerdictBlock = 21
			case "another-live-challenge":
				require.NoError(t, k.SetClaim(ctx, &types.Claim{Id: "other", ProvisionalFactId: "target", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}))
			case "live-contradiction":
				require.NoError(t, k.SetClaim(ctx, &types.Claim{Id: "contradiction", Relations: []*types.ClaimRelation{{TargetFactId: "target", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}}}))
			case "orphan-live-round":
				require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{Id: "orphan", ClaimId: "absent", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}))
			case "multiple-rejected-rounds", "multiple-stranded-rounds":
				extra := proto.Clone(r1).(*types.VerificationRound)
				if scenario == "multiple-stranded-rounds" {
					extra = proto.Clone(r2).(*types.VerificationRound)
				}
				extra.Id = "extra-round"
				require.NoError(t, k.SetVerificationRound(ctx, extra))
			}
			require.NoError(t, k.SetClaim(ctx, rejected))
			require.NoError(t, k.SetClaim(ctx, stranded))
			require.NoError(t, k.SetVerificationRound(ctx, r1))
			require.NoError(t, k.SetVerificationRound(ctx, r2))
			if scenario == "missing-round" {
				require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Delete(types.RoundKey(r1.Id)))
			}
			if scenario == "missing-rejected-claim" {
				// Its orphan round is live: without the claim even its target is unknown.
				r1.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMMIT
				require.NoError(t, k.SetVerificationRound(ctx, r1))
				require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Delete(types.ClaimKey(rejected.Id)))
			}
			beforeFact, found := k.GetFact(ctx, "target")
			require.True(t, found)
			beforeHistory := k.GetStatusHistory(ctx, "target")
			require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)))
			after, found := k.GetFact(ctx, "target")
			require.True(t, found)
			require.True(t, proto.Equal(beforeFact, after), "ambiguity must leave target challenged without any credit")
			require.Equal(t, beforeHistory, k.GetStatusHistory(ctx, "target"))
			beforeReplay := feedbackSnapshot(t, s, ctx)
			require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(21)))
			require.Equal(t, beforeReplay, feedbackSnapshot(t, s, ctx))
		})
	}
}

func TestToKMigrationPriorRejectionBeforeHistoricalStarvation(t *testing.T) {
	for _, scenario := range []string{"dated-starvation", "missing-deadline", "deadline-before-start", "future-deadline", "inconsistent-verdict-height"} {
		t.Run(scenario, func(t *testing.T) {
			k, ctx, _, _, _, _, round := feedbackRejectedThenStranded(t, 1, types.Verdict_VERDICT_INCONCLUSIVE)
			// Old starvation records have EXPIRED + INCONCLUSIVE but no VerdictBlock.
			round.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
			round.VerdictBlock = 0
			height := int64(round.AggregationDeadline + 1)
			switch scenario {
			case "missing-deadline":
				round.AggregationDeadline = 0
			case "deadline-before-start":
				round.AggregationDeadline = round.StartedAtBlock - 1
			case "future-deadline":
				round.AggregationDeadline = uint64(height + 1)
			case "inconsistent-verdict-height":
				round.VerdictBlock = 12
			}
			require.NoError(t, k.SetVerificationRound(ctx, round))
			require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(height)))
			fact, found := k.GetFact(ctx, "target")
			require.True(t, found)
			want := types.FactStatus_FACT_STATUS_CHALLENGED
			if scenario == "dated-starvation" {
				want = types.FactStatus_FACT_STATUS_ACTIVE
			}
			require.Equal(t, want, fact.Status)
		})
	}
}

func TestToKMigrationUsesLatestRejectionAndQualifyingStrand(t *testing.T) {
	for _, laterRejection := range []bool{false, true} {
		t.Run(fmt.Sprintf("later_rejection_%t", laterRejection), func(t *testing.T) {
			k, ctx, _, _, rejectedRound, stranded, round := feedbackRejectedThenStranded(t, 3, types.Verdict_VERDICT_INCONCLUSIVE)
			// An earlier strand sorts first but cannot explain the current challenge:
			// every retained rejection happened after that earlier submission.
			earlier := proto.Clone(stranded).(*types.Claim)
			earlier.Id, earlier.VerificationRoundId, earlier.SubmittedAtBlock = "!earlier-strand", "earlier-round", 1
			earlierRound := proto.Clone(round).(*types.VerificationRound)
			earlierRound.Id, earlierRound.ClaimId = earlier.VerificationRoundId, earlier.Id
			earlierRound.StartedAtBlock, earlierRound.VerdictBlock = 1, 2
			require.NoError(t, k.SetClaim(ctx, earlier))
			require.NoError(t, k.SetVerificationRound(ctx, earlierRound))
			if laterRejection {
				rejectedRound.VerdictBlock = 13 // other rejections still predate submission 11
				require.NoError(t, k.SetVerificationRound(ctx, rejectedRound))
			}
			before := k.GetStatusHistory(ctx, "target")
			require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)))
			fact, found := k.GetFact(ctx, "target")
			require.True(t, found)
			after := k.GetStatusHistory(ctx, "target")
			if laterRejection {
				require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, fact.Status)
				require.Equal(t, before, after)
			} else {
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
				require.Len(t, after, len(before)+1)
				require.Equal(t, stranded.Id, after[len(after)-1].CauseId, "only the later qualifying strand explains restoration")
			}
		})
	}
}

func TestToKMigrationPriorRejectedRepairFailureIsAtomic(t *testing.T) {
	k, ctx, s, _, _, _, _ := feedbackRejectedThenStranded(t, 1, types.Verdict_VERDICT_INCONCLUSIVE)
	before := feedbackSnapshot(t, s, ctx)
	s.op, s.prefix = "set", types.StatusTransitionKeyPrefix
	require.ErrorIs(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)), feedbackStoreErr)
	require.Equal(t, before, feedbackSnapshot(t, s, ctx), "failed history write must roll back target, sequence, params and migration marker")
	s.op = ""
	require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)))
	fact, found := k.GetFact(ctx, "target")
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
}
