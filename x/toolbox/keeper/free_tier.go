package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// IsEssentialCategory returns true if the category is eligible for free-tier calls.
func IsEssentialCategory(category string) bool {
	return types.EssentialCategories[category]
}

// CheckFreeEligibility checks whether a caller is eligible for free-tier calls.
// Requirements: home keeper available, caller owns a home, home age >= MinHomeAgeBlocks.
// Returns (eligible, reason) where reason explains denial.
func (k Keeper) CheckFreeEligibility(ctx context.Context, caller string) (bool, string) {
	params := k.GetParams(ctx)
	if !params.FreeCallsEnabled {
		return false, "free_calls_disabled"
	}
	if k.homeKeeper == nil {
		return false, "home_keeper_unavailable"
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())

	homes, err := k.homeKeeper.GetHomesByOwner(ctx, caller)
	if err != nil || len(homes) == 0 {
		return false, "no_home_owned"
	}

	// Check that at least one home is old enough and active.
	for _, homeID := range homes {
		createdAt, err := k.homeKeeper.GetHomeCreatedAtBlock(ctx, homeID)
		if err != nil {
			continue
		}
		if blockHeight < createdAt+params.MinHomeAgeBlocks {
			continue
		}
		status, err := k.homeKeeper.GetHomeStatus(ctx, homeID)
		if err != nil {
			continue
		}
		if status == "active" {
			return true, ""
		}
	}
	return false, "no_eligible_home"
}

// TryConsumeFreeCall attempts to use a free-tier call for the given caller and tool.
// Returns true if a free call was consumed successfully.
func (k Keeper) TryConsumeFreeCall(ctx context.Context, caller string, tool *types.Tool) bool {
	params := k.GetParams(ctx)
	if !params.FreeCallsEnabled {
		return false
	}

	// Must be essential category.
	if !IsEssentialCategory(tool.Category) {
		return false
	}

	// Check eligibility.
	eligible, _ := k.CheckFreeEligibility(ctx, caller)
	if !eligible {
		return false
	}

	// Check and consume allowance.
	fa := k.GetFreeAllowance(ctx, caller)
	if fa.UsedCalls >= params.FreeCallsPerEpoch {
		return false
	}

	fa.UsedCalls++
	k.SetFreeAllowance(ctx, fa)

	// Emit free call event.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.toolbox.free_call",
			sdk.NewAttribute("caller", caller),
			sdk.NewAttribute("tool_id", tool.Id),
			sdk.NewAttribute("category", tool.Category),
		),
	)

	return true
}

// GetCurrentEpochForTest exports the current epoch for testing.
func (k Keeper) GetCurrentEpochForTest(ctx context.Context) uint64 {
	return k.getCurrentEpoch(ctx)
}
