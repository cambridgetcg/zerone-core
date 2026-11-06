package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/alignment/types"
)

// HasHealthObservation distinguishes an empty registry from registered but
// inactive validators. Call only on the Q1 path selected by EndBlock, before
// observation writes. Occupied malformed records are not an empty registry.
func (k Keeper) HasHealthObservation(ctx context.Context) (bool, error) {
	if k.stakingKeeper == nil {
		return false, nil
	}
	if registry, ok := k.stakingKeeper.(types.StakingRegistry); ok {
		count, err := registry.GetRegisteredValidatorCount(ctx)
		return count > 0, err
	}
	// An older sensor can prove presence with an active validator, but zero
	// cannot prove absence. Without the registry capability it is unobserved,
	// not a fabricated critical observation. The production adapter has it.
	return k.stakingKeeper.GetActiveValidatorCount(ctx) > 0, nil
}

// CategorizeObservedHealth uses all three thresholds. The interval between
// DegradedThreshold and HealthyThreshold is a hysteresis band: a previously
// healthy chain stays healthy, otherwise it stays degraded until recovery.
// A first genuine sample in that band is degraded, not asserted healthy.
// Critical is reserved for scores strictly below CriticalThreshold.
// The caller has already selected Q1 and established registry presence.
func (k Keeper) CategorizeObservedHealth(ctx context.Context, composite uint64, previous string) string {
	params := k.GetParams(ctx)
	switch {
	case composite < params.CriticalThreshold:
		return types.CategoryCritical
	case composite < params.DegradedThreshold:
		return types.CategoryDegraded
	case composite >= params.HealthyThreshold:
		return types.CategoryHealthy
	case previous == types.CategoryHealthy:
		return types.CategoryHealthy
	default:
		return types.CategoryDegraded
	}
}
