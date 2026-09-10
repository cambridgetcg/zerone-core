package keeper_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

func TestReviewNeutralityFreezesQualificationFeedback(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	params.DecayMinSamples = 1
	params.DecayCheckIntervalBlocks = 1
	params.DecayProbationBps = 900_000
	params.DecaySuspensionBps = 500_000
	params.DecayRecoveryBps = 950_000
	k.SetParams(ctx, params)
	for _, status := range []types.QualificationStatus{
		types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
	} {
		q := &types.DomainQualification{
			Validator: testAddr(status.String()), Domain: "physics", Status: status,
			Weight: 80, ProbationUntil: 1,
			Metrics: &types.QualificationMetrics{TotalVerifications: 100, CorrectVerifications: 1, AccuracyBps: 10_000},
		}
		k.SetQualification(ctx, q)
	}
	before := k.ExportGenesis(ctx)
	k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return true, nil })
	adapter := keeper.NewKnowledgeDomainQualificationAdapter(k)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	for _, q := range before.Qualifications {
		require.NoError(t, adapter.RecordVerificationOutcome(ctx, q.Validator, q.Domain, true))
		require.NoError(t, adapter.RecordVerificationOutcome(ctx, q.Validator, q.Domain, false))
		require.ErrorContains(t, k.QualifyByTrackRecord(ctx, q.Validator, q.Domain), "retired")
	}
	// Missing records also remain absent: retirement never bootstraps a score.
	require.NoError(t, adapter.RecordVerificationOutcome(ctx, "missing", "physics", true))
	require.ErrorContains(t, k.QualifyByTrackRecord(ctx, "missing", "physics"), "retired")
	require.NoError(t, k.RunAccuracyDecay(ctx, 100, params))
	require.NoError(t, k.BeginBlocker(ctx))
	require.True(t, proto.Equal(before, k.ExportGenesis(ctx)))
	require.Empty(t, ctx.EventManager().Events())
}

func TestReviewNeutralityQualificationPolicyErrorRefusesBeforeEffects(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	k.SetQualification(ctx, &types.DomainQualification{Validator: testAddr("old"), Domain: "physics", Status: types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, ExpiresAt: 1})
	before := k.ExportGenesis(ctx)
	sentinel := errors.New("unreadable review policy")
	k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return false, sentinel })
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.ErrorIs(t, k.RecordVerificationOutcome(ctx, testAddr("old"), "physics", true), sentinel)
	require.ErrorIs(t, k.QualifyByTrackRecord(ctx, "new", "physics"), sentinel)
	require.ErrorIs(t, k.RunAccuracyDecay(ctx, 100, k.GetParams(ctx)), sentinel)
	require.ErrorIs(t, k.BeginBlocker(ctx), sentinel)
	require.True(t, proto.Equal(before, k.ExportGenesis(ctx)))
	require.Empty(t, ctx.EventManager().Events())
}

func TestReviewNeutralityPreservesQualificationCustody(t *testing.T) {
	for _, mode := range []string{"expiry", "withdrawal"} {
		t.Run(mode, func(t *testing.T) {
			k, ctx, bank, _ := setupKeeper(t)
			validator := testAddr(mode)
			bank.setBalance(validator, "uzrn", 200_000_000)
			require.NoError(t, k.QualifyByStake(ctx, validator, "physics", "100000000"))
			q, found := k.GetQualification(ctx, validator, "physics")
			require.True(t, found)
			params := k.GetParams(ctx)
			k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return true, nil })
			height := q.GrantedAt + params.StakeLockPeriod + 1
			if mode == "expiry" {
				q.Status = types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
				q.ProbationUntil = 1
				q.ExpiresAt = height
				k.SetQualification(ctx, q)
				require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(int64(height))))
				after, exists := k.GetQualification(ctx, validator, "physics")
				require.True(t, exists)
				require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, after.Status)
			} else {
				server := keeper.NewMsgServerImpl(k)
				_, err := server.WithdrawQualification(ctx, &types.MsgWithdrawQualification{Validator: validator, Domain: "physics"})
				require.Error(t, err, "the existing stake lock remains enforced")
				_, err = server.WithdrawQualification(ctx.WithBlockHeight(int64(height)), &types.MsgWithdrawQualification{Validator: validator, Domain: "physics"})
				require.NoError(t, err)
				_, exists := k.GetQualification(ctx, validator, "physics")
				require.False(t, exists)
			}
			require.EqualValues(t, 200_000_000, bank.balances[validator]["uzrn"])
			require.Zero(t, bank.moduleBalances[types.ModuleName]["uzrn"])
		})
	}
}
