package keeper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Test-only standalone adapters. Production dispatches the roundsExecution
// methods with marker snapshots selected at its entry point.
func (k Keeper) createFactFromClaim(ctx context.Context, claim *types.Claim, round *types.VerificationRound, confidence uint64) (string, error) {
	execution, err := k.selectRoundsExecution(ctx)
	if err != nil {
		return "", err
	}
	return execution.createFactFromClaim(ctx, claim, round, confidence)
}
func (k Keeper) completeRound(ctx context.Context, round *types.VerificationRound, result *VerificationResult, recordIntegrity bool) error {
	execution, err := k.selectRoundsExecution(ctx)
	if err != nil {
		return err
	}
	return execution.completeRound(ctx, round, result, recordIntegrity)
}
func (k Keeper) handleChallengeSurvival(ctx context.Context, claim *types.Claim) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).handleChallengeSurvival(ctx, claim)
}
func (k Keeper) handleChallengeDisproven(ctx context.Context, claim *types.Claim, factID string) error {
	enabled, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		return err
	}
	return (roundsExecution{Keeper: k, calibrationSmoothing: enabled}).handleChallengeDisproven(ctx, claim, factID)
}
func (k Keeper) beginBlockInconclusiveAccountingLegacy(ctx context.Context) error {
	return (roundsExecution{Keeper: k}).beginBlockInconclusiveAccountingLegacy(ctx)
}

func TestCR2GenuineZeroRequiresSubmissions(t *testing.T) {
	for _, submissions := range []uint64{0, 1} {
		for _, kind := range []types.ClaimType{types.ClaimType_CLAIM_TYPE_ASSERTION, types.ClaimType_CLAIM_TYPE_CONJECTURE} {
			f := newKR2Fixture(t, true, true, false)
			require.NoError(t, f.k.SetAgentCalibration(f.ctx, &types.AgentCalibration{Address: "author", TotalSubmissions: submissions}))
			id, err := f.k.createFactFromClaim(f.ctx, &types.Claim{Id: "claim", Submitter: "author", ClaimType: kind}, &types.VerificationRound{}, 850_000)
			require.NoError(t, err)
			fact, found := f.k.GetFact(f.ctx, id)
			require.True(t, found)
			require.Equal(t, submissions > 0, fact.HasCalibrationSnapshot)
			want := uint64(500_000)
			if submissions > 0 {
				want = 0
			}
			require.Equal(t, want, fact.SubmitterCalibrationSnapshotBps)
		}
	}
}
