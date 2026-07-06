package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetVerificationHealth returns verification metrics for the alignment module (R31-2: Fire -> Earth).
// Returns throughput (BPS relative to theoretical max), dispute rate (BPS), and avg round duration.
func (k Keeper) GetVerificationHealth(ctx context.Context) (throughputBps, disputeRateBps, avgRoundDurationBlocks uint64) {
	params, err := k.GetParams(ctx)
	if err != nil || params.CommitPhaseBlocks == 0 {
		return 0, 0, 0
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	windowBlocks := params.ObservationWindowBlocks
	if windowBlocks == 0 {
		windowBlocks = 10_000 // fallback default
	}

	completed := k.CountCompletedRoundsInWindow(ctx, height, windowBlocks)
	if completed == 0 {
		return 0, 0, 0
	}

	// Theoretical max: how many rounds could fit in the window
	roundCycleBlocks := params.CommitPhaseBlocks + params.RevealPhaseBlocks + params.AggregationPhaseBlocks
	if roundCycleBlocks == 0 {
		roundCycleBlocks = 1
	}
	theoreticalMax := windowBlocks / roundCycleBlocks
	if theoreticalMax == 0 {
		theoreticalMax = 1
	}

	throughputBps = completed * BPS / theoreticalMax
	if throughputBps > BPS {
		throughputBps = BPS
	}

	disputed := k.CountDisputedRoundsInWindow(ctx, height, windowBlocks)
	disputeRateBps = disputed * BPS / completed

	avgRoundDurationBlocks = k.GetAvgRoundDurationInWindow(ctx, height, windowBlocks)

	return throughputBps, disputeRateBps, avgRoundDurationBlocks
}

// GetEffectiveMinVerifiers returns the adjusted minimum verifiers for a domain,
// accounting for active capture-challenge overrides (R28-8: Metal -> Fire).
// The partnership-density modifier retired with x/partnerships (slim cut);
// this keeps the module's documented no-social-structure behavior (base + 1).
func (k Keeper) GetEffectiveMinVerifiers(ctx context.Context, domain string) uint32 {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 3 // safe default
	}
	base := uint32(params.MinVerifiers)

	// No social structure -> Fire burns unchecked.
	adjusted := base + 1

	// Apply capture-challenge override on top.
	if additional, active := k.GetVerificationThresholdOverride(ctx, domain); active {
		adjusted += additional
	}
	return adjusted
}
