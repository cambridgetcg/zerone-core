package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// Surge pricing tier constants.
const (
	TierEssential = "essential" // No surge.
	TierStandard  = "standard"  // Up to 2x cap.
	TierHeavy     = "heavy"     // Up to 10x cap.
)

// Surge cap constants (BPS).
const (
	essentialSurgeCap = types.BpsDenominator     // 1x (no surge)
	standardSurgeCap  = 2 * types.BpsDenominator // 2x
	heavySurgeCap     = 10 * types.BpsDenominator // 10x
)

// Heavy-tier exponential step constants.
const (
	heavyExpStepBps = 50_000 // Step size: 5% utilisation above critical
	heavyExpFactor  = 15     // 1.5× per step (numerator, /10 denominator)
	heavyExpDenom   = 10
)

// PricingTier maps a tool category to a pricing tier.
func PricingTier(category string) string {
	if types.EssentialCategories[category] {
		return TierEssential
	}
	switch category {
	case types.CategoryComputation, types.CategoryIntegration, types.CategoryComposite:
		return TierHeavy
	default:
		return TierStandard
	}
}

// tierMaxMultiplier returns the per-tier surge cap, respecting the MaxSurgeMultiplierBps param.
func tierMaxMultiplier(tier string, globalMax uint64) uint64 {
	switch tier {
	case TierEssential:
		return types.BpsDenominator
	case TierStandard:
		surgeCap := uint64(standardSurgeCap)
		if globalMax > 0 && surgeCap > globalMax {
			surgeCap = globalMax
		}
		return surgeCap
	case TierHeavy:
		surgeCap := uint64(heavySurgeCap)
		if globalMax > 0 && surgeCap > globalMax {
			surgeCap = globalMax
		}
		return surgeCap
	default:
		surgeCap := uint64(standardSurgeCap)
		if globalMax > 0 && surgeCap > globalMax {
			surgeCap = globalMax
		}
		return surgeCap
	}
}

// CalculateSurgeMultiplier computes the surge multiplier in BPS for a tool.
// Returns BpsDenominator (1x) if no surge applies.
func (k Keeper) CalculateSurgeMultiplier(ctx context.Context, tool *types.Tool) uint64 {
	params := k.GetParams(ctx)
	if !params.SurgeEnabled {
		return types.BpsDenominator
	}

	tier := PricingTier(tool.Category)
	if tier == TierEssential {
		return types.BpsDenominator
	}

	maxSurge := params.MaxSurgeMultiplierBps
	if maxSurge == 0 {
		maxSurge = types.DefaultMaxSurgeMultiplierBps
	}
	tierMax := tierMaxMultiplier(tier, maxSurge)

	// Per-tool utilisation.
	_, toolUtil := k.GetToolDemand(ctx, tool.Id)
	toolSurge := surgeCurve(toolUtil, tier, params, tierMax)

	// Global overlay — additive 50% blend.
	_, globalUtil := k.GetGlobalDemand(ctx)
	globalSurge := surgeCurve(globalUtil, tier, params, tierMax)

	// Blended = toolSurge + (globalSurge - 1.0×) / 2
	globalExcess := uint64(0)
	if globalSurge > types.BpsDenominator {
		globalExcess = (globalSurge - types.BpsDenominator) / 2
	}
	blended := toolSurge + globalExcess
	if blended > tierMax {
		blended = tierMax
	}

	return blended
}

// surgeCurve computes the surge multiplier for a given utilisation level.
// Essential: always 1.0×
// Standard: linear 1.0× → 2.0× between threshold and critical, capped at critical
// Heavy: linear 1.0× → 3.0× between threshold and critical, then exponential ×1.5 per step above critical
func surgeCurve(utilisationBps uint64, tier string, params *types.Params, tierMax uint64) uint64 {
	if tier == TierEssential {
		return types.BpsDenominator
	}

	threshold := params.SurgeThresholdBps
	if threshold == 0 {
		threshold = types.DefaultSurgeThresholdBps
	}
	critical := params.SurgeCriticalBps
	if critical == 0 {
		critical = types.DefaultSurgeCriticalBps
	}

	if utilisationBps <= threshold {
		return types.BpsDenominator // No surge below threshold.
	}

	switch tier {
	case TierStandard:
		return standardSurgeCurve(utilisationBps, threshold, critical, tierMax)
	case TierHeavy:
		return heavySurgeCurve(utilisationBps, threshold, critical, tierMax)
	default:
		return standardSurgeCurve(utilisationBps, threshold, critical, tierMax)
	}
}

// standardSurgeCurve: linear 1.0× → 2.0× between threshold and critical, capped at critical.
func standardSurgeCurve(util, threshold, critical, tierMax uint64) uint64 {
	if util >= critical {
		return tierMax
	}
	rng := critical - threshold
	if rng == 0 {
		return types.BpsDenominator
	}
	progress := util - threshold
	// Linear from 1× to 2×
	surgeRange := uint64(standardSurgeCap - types.BpsDenominator)
	surge := types.BpsDenominator + safeMulDiv(progress, surgeRange, rng)
	if surge > tierMax {
		surge = tierMax
	}
	return surge
}

// heavySurgeCurve: linear 1.0× → 3.0× between threshold and critical,
// then exponential ×1.5 per 50k step above critical, capped at tier max.
func heavySurgeCurve(util, threshold, critical, tierMax uint64) uint64 {
	if util <= critical {
		// Linear phase: 1.0× → 3.0×
		rng := critical - threshold
		if rng == 0 {
			return types.BpsDenominator
		}
		progress := util - threshold
		linearCap := uint64(3 * types.BpsDenominator) // 3.0×
		surgeRange := linearCap - types.BpsDenominator
		surge := types.BpsDenominator + safeMulDiv(progress, surgeRange, rng)
		if surge > tierMax {
			surge = tierMax
		}
		return surge
	}

	// Exponential phase above critical.
	// Start at 3.0×, multiply by 1.5 for each step of 50k BPS above critical.
	base := uint64(3 * types.BpsDenominator) // 3.0×
	excess := util - critical
	steps := excess / heavyExpStepBps

	surge := base
	for i := uint64(0); i < steps; i++ {
		surge = surge * heavyExpFactor / heavyExpDenom
		if surge >= tierMax {
			return tierMax
		}
	}
	if surge > tierMax {
		surge = tierMax
	}
	return surge
}

// ApplySurge applies a surge multiplier to a base price (both in BPS-compatible units).
// Short-circuits when multiplier is 1.0× or below.
func ApplySurge(basePriceUzrn uint64, surgeMultiplierBps uint64) uint64 {
	if surgeMultiplierBps <= types.BpsDenominator {
		return basePriceUzrn
	}
	return safeMulDiv(basePriceUzrn, surgeMultiplierBps, types.BpsDenominator)
}

// emitSurgeEvent emits a tool_surge_pricing event.
func emitSurgeEvent(ctx context.Context, toolID string, basePrice, effectivePrice, multiplierBps, utilisationBps uint64, tier string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.toolbox.surge_pricing",
			sdk.NewAttribute("tool_id", toolID),
			sdk.NewAttribute("base_price", strconv.FormatUint(basePrice, 10)),
			sdk.NewAttribute("effective_price", strconv.FormatUint(effectivePrice, 10)),
			sdk.NewAttribute("multiplier_bps", strconv.FormatUint(multiplierBps, 10)),
			sdk.NewAttribute("utilisation_bps", strconv.FormatUint(utilisationBps, 10)),
			sdk.NewAttribute("tier", tier),
		),
	)
}
