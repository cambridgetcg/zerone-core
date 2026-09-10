package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

func nondecisiveChallengeFixture(t *testing.T, policy uint32, verdict types.Verdict) (handoffFixture, *types.Claim, *types.VerificationRound) {
	t.Helper()
	var f handoffFixture
	var claim *types.Claim
	var round *types.VerificationRound
	if policy == 1 {
		votes := []string{"accept", "reject", "malformed"}
		if verdict == types.Verdict_VERDICT_MALFORMED {
			votes = []string{"malformed", "malformed", "reject"}
		}
		f, claim, round = neutralReviewFixture(t, votes)
		claim.ProvisionalFactId = f.pending.FactId
		require.NoError(t, f.k.SetClaim(f.ctx, claim))
	} else {
		f, _ = bonusPolicyFixture(t, true)
		claim = &types.Claim{Id: "legacy-nondecisive", ProvisionalFactId: f.pending.FactId, Stake: "10000", Submitter: "recipient", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
		require.NoError(t, f.k.SetClaim(f.ctx, claim))
		round = &types.VerificationRound{Id: "legacy-nondecisive-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
		require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	}
	fact, found := f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	fact.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	fact.Energy = 123
	fact.CorroborationCount = 4
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	return f, claim, round
}

func TestReviewPolicyNondecisiveChallengeRestoresWithoutReward(t *testing.T) {
	for _, policy := range []uint32{0, 1} {
		for _, verdict := range []types.Verdict{types.Verdict_VERDICT_INCONCLUSIVE, types.Verdict_VERDICT_MALFORMED} {
			t.Run(fmt.Sprintf("%d/%s", policy, verdict), func(t *testing.T) {
				f, _, round := nondecisiveChallengeFixture(t, policy, verdict)
				require.NoError(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: verdict}))
				fact, found := f.k.GetFact(f.ctx, f.pending.FactId)
				require.True(t, found)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
				require.Equal(t, uint64(123), fact.Energy)
				require.Equal(t, uint64(4), fact.CorroborationCount)
				pending, found, err := f.k.getSurvivalPendingReward(f.ctx, f.pending.FactId)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, f.pending, pending)
				completed, found := f.k.GetVerificationRound(f.ctx, round.Id)
				require.True(t, found)
				require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, completed.Phase)
				require.Equal(t, verdict, completed.Verdict)
			})
		}
	}
}

func TestReviewPolicyRestorationRespectsOtherActiveChallenge(t *testing.T) {
	f, _, round := nondecisiveChallengeFixture(t, 1, types.Verdict_VERDICT_INCONCLUSIVE)
	other := &types.Claim{Id: "other-open-challenge", ProvisionalFactId: f.pending.FactId, Stake: "10000", Submitter: "recipient", Status: types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
	require.NoError(t, f.k.SetClaim(f.ctx, other))
	otherRound := &types.VerificationRound{Id: "other-open-round", ClaimId: other.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, otherRound))
	require.NoError(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	fact, found := f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, fact.Status)
	require.NoError(t, f.k.CompleteRound(f.ctx, otherRound, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	fact, found = f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
}

func TestReviewPolicyRestorationReadOrWriteFailureRollsBackCompletion(t *testing.T) {
	for _, failure := range []string{"target-read", "target-write", "active-round-orphan", "active-claim-corrupt"} {
		t.Run(failure, func(t *testing.T) {
			f, _, round := nondecisiveChallengeFixture(t, 1, types.Verdict_VERDICT_INCONCLUSIVE)
			switch failure {
			case "target-read":
				*f.kFault = handoffFault{op: "get", prefix: types.FactKeyPrefix}
			case "target-write":
				*f.kFault = handoffFault{op: "set", prefix: types.FactKeyPrefix, after: true}
			case "active-round-orphan":
				require.NoError(t, f.k.storeService.OpenKVStore(f.ctx).Set(activeRoundKey("orphan"), []byte{1}))
			case "active-claim-corrupt":
				require.NoError(t, f.k.SetClaim(f.ctx, &types.Claim{Id: "other-active", ProvisionalFactId: f.pending.FactId}))
				require.NoError(t, f.k.SetVerificationRound(f.ctx, &types.VerificationRound{Id: "other-active-round", ClaimId: "other-active", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}))
				require.NoError(t, f.k.storeService.OpenKVStore(f.ctx).Set(types.ClaimKey("other-active"), []byte{0xff}))
			}
			before := f.snapshot(t)
			events := len(f.ctx.EventManager().Events())
			beforeRound := proto.Clone(round)
			require.Error(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
			require.Equal(t, before, f.snapshot(t))
			require.Len(t, f.ctx.EventManager().Events(), events)
			require.True(t, proto.Equal(beforeRound, round))
		})
	}
}

func TestReviewPolicyRestorationKeepsAnotherRoundForSameLegacyClaim(t *testing.T) {
	f, claim, round := nondecisiveChallengeFixture(t, 0, types.Verdict_VERDICT_INCONCLUSIVE)
	// Historical import preserves both primary rounds even though the selected
	// claim→round index names one. This fixture must exercise that valid shape.
	other := &types.VerificationRound{Id: "other-round-same-legacy-claim", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
	raw, err := marshalOpts.Marshal(other)
	require.NoError(t, err)
	store := f.k.storeService.OpenKVStore(f.ctx)
	require.NoError(t, store.Set(types.RoundKey(other.Id), raw))
	require.NoError(t, store.Set(activeRoundKey(other.Id), []byte{1}))
	require.NoError(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	fact, found := f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, fact.Status)
	require.NoError(t, f.k.CompleteRound(f.ctx, other, &VerificationResult{Verdict: types.Verdict_VERDICT_INCONCLUSIVE}))
	fact, found = f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, fact.Status)
}
