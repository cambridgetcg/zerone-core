package keeper_test

import (
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestToKMigrationDefaultsAtomicityAndIdempotence(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FactUseEnabled = false
	p.FactUseConsumers = nil
	p.FactUseMaxPerEpoch = 0
	p.FactUseMaxPerConsumerEpoch = 0
	require.NoError(t, k.SetParams(ctx, p))
	original := proto.Clone(p).(*types.Params)
	feedbackFact(t, k, ctx, "untouched")
	before := feedbackSnapshot(t, s, ctx)
	s.op, s.prefix = "set", types.FactUsePruningStateKey
	require.Error(t, keeper.NewMigrator(k).Migrate6to7(ctx))
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	s.op = ""
	require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx))
	updated, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.False(t, updated.FactUseEnabled)
	require.Empty(t, updated.FactUseConsumers)
	require.EqualValues(t, 100, updated.FactUseMaxPerConsumerEpoch)
	require.EqualValues(t, 1000, updated.FactUseMaxPerEpoch)
	original.FactUseMaxPerConsumerEpoch = 100
	original.FactUseMaxPerEpoch = 1000
	require.True(t, proto.Equal(original, updated), "migration may not overwrite old params")
	enableFeedback(t, k, ctx, feedbackConsumer(1))
	_, err = keeper.NewMsgServerImpl(k).ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(1), FactId: "untouched"})
	require.NoError(t, err)
	before = feedbackSnapshot(t, s, ctx)
	require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx))
	require.Equal(t, before, feedbackSnapshot(t, s, ctx), "replay must not clear latch or cohort")
}

func TestToKMigrationOnlyRepairsProvenTerminalChallenges(t *testing.T) {
	for _, scenario := range []string{"terminal", "conjecture", "live-other", "orphan-live", "missing-round", "ambiguous", "missing-history"} {
		t.Run(scenario, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			p, err := k.GetParams(ctx)
			require.NoError(t, err)
			p.FactUseEnabled = false
			p.FactUseConsumers = nil
			require.NoError(t, k.SetParams(ctx, p))
			f := feedbackFact(t, k, ctx, "target")
			f.Status = types.FactStatus_FACT_STATUS_CHALLENGED
			f.Energy = 432
			f.CorroborationCount = 17
			if scenario == "conjecture" {
				f.ClaimType = types.ClaimType_CLAIM_TYPE_CONJECTURE
				f.Confidence = 0
			}
			require.NoError(t, k.SetFactSkipTransition(ctx, f))
			claim := &types.Claim{Id: "challenge", ProvisionalFactId: "target", FactContent: "Challenge of fact target: recorded reason", Status: types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT}
			if scenario == "conjecture" {
				claim.FactContent = "Provisional challenge of fact target: recorded reason"
			}
			if scenario == "ambiguous" {
				claim.FactContent = "ordinary unresolved claim"
			}
			require.NoError(t, k.SetClaim(ctx, claim))
			if scenario != "missing-round" {
				require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{Id: "round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: types.Verdict_VERDICT_INCONCLUSIVE, VerdictBlock: 8}))
			}
			if scenario == "live-other" {
				require.NoError(t, k.SetClaim(ctx, &types.Claim{Id: "other", ProvisionalFactId: "target", Status: types.ClaimStatus_CLAIM_STATUS_PENDING}))
			}
			if scenario == "orphan-live" {
				require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{Id: "orphan", ClaimId: "absent", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}))
			}
			if scenario == "missing-history" {
				raw := s.KVStoreService.OpenKVStore(ctx)
				require.NoError(t, raw.Delete(types.StatusTransitionKey("target", 1)))
				require.NoError(t, raw.Delete(types.StatusTransitionSeqKey("target")))
			}
			previous := len(k.GetStatusHistory(ctx, f.Id))
			require.NoError(t, keeper.NewMigrator(k).Migrate6to7(ctx.WithBlockHeight(20)))
			actual, found := k.GetFact(ctx, "target")
			require.True(t, found)
			repaired := scenario == "terminal" || scenario == "conjecture" || scenario == "missing-history"
			if repaired {
				want := types.FactStatus_FACT_STATUS_ACTIVE
				if scenario == "conjecture" {
					want = types.FactStatus_FACT_STATUS_PROVISIONAL
				}
				require.Equal(t, want, actual.Status)
				h := k.GetStatusHistory(ctx, f.Id)
				require.Len(t, h, previous+1)
				require.Equal(t, "migration_tok_feedback_v1_terminal_challenge", h[len(h)-1].CauseEventType)
				require.EqualValues(t, 20, h[len(h)-1].BlockHeight)
			} else {
				require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, actual.Status)
				require.Len(t, k.GetStatusHistory(ctx, f.Id), previous)
			}
			require.Equal(t, f.Energy, actual.Energy)
			require.Equal(t, f.CorroborationCount, actual.CorroborationCount)
			require.Equal(t, f.Confidence, actual.Confidence)
			// Missing completion metadata must stay missing.
			require.Zero(t, k.CountCompletedRoundsInWindow(ctx, 20, 20))
		})
	}
}
