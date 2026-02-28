package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetDomainVerificationActivity returns the verification activity level for a domain
// as a BPS value (0-1,000,000). Higher activity = more active verification (R31-4).
// Activity is computed from the ratio of completed rounds to domain carrying capacity.
func (k Keeper) GetDomainVerificationActivity(ctx context.Context, domain string) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil || params.FitnessEpochBlocks == 0 {
		return 0
	}

	height := uint64(sdkCtx.BlockHeight())
	epoch := height / params.FitnessEpochBlocks

	// Check current and previous epoch for diversity data (which tracks round counts)
	for try := uint64(0); try < 2; try++ {
		if epoch < try {
			break
		}
		checkEpoch := epoch - try
		rec, found, getErr := k.GetDomainDiversity(ctx, domain, checkEpoch)
		if getErr == nil && found && rec.RoundCount > 0 {
			// Activity = min(roundCount * 100_000, BPS)
			// Each round contributes 10% (100,000 BPS) of activity, capped at 100%
			activity := rec.RoundCount * 100_000
			if activity > BPS {
				activity = BPS
			}
			return activity
		}
	}

	return 0
}
