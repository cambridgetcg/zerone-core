package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/alignment/types"
)

// CategorizeHealth retains the legacy body verbatim from
// 6a56b621082064483af1ced7718462766d9ae756. It does not select activation.
func (k Keeper) CategorizeHealth(ctx context.Context, composite uint64) string {
	params := k.GetParams(ctx)
	if composite < params.CriticalThreshold {
		return types.CategoryCritical
	}
	if composite < params.HealthyThreshold {
		return types.CategoryDegraded
	}
	return types.CategoryHealthy
}
