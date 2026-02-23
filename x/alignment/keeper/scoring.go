package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/alignment/types"
)

// ComputeScores calculates weighted dimension scores from an observation.
func (k Keeper) ComputeScores(ctx context.Context, obs *types.AlignmentObservation) *types.DimensionScores {
	params := k.GetParams(ctx)

	// Weighted composite: sum of (dimension * weight) / BPS.
	composite := (obs.KnowledgeQuality*params.WeightKnowledgeQuality +
		obs.EconomicStability*params.WeightEconomicStability +
		obs.GovernanceParticipation*params.WeightGovernanceParticipation +
		obs.NetworkSecurity*params.WeightNetworkSecurity +
		obs.StakingRatio*params.WeightStakingRatio) / types.BPS

	return &types.DimensionScores{
		Height:                  obs.Height,
		KnowledgeQuality:        obs.KnowledgeQuality,
		EconomicStability:       obs.EconomicStability,
		GovernanceParticipation: obs.GovernanceParticipation,
		NetworkSecurity:         obs.NetworkSecurity,
		StakingRatio:            obs.StakingRatio,
		Composite:               composite,
	}
}

// CategorizeHealth maps a composite score to a health category.
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

// BuildHealthIndex creates a complete health index from scores and corrections.
func (k Keeper) BuildHealthIndex(scores *types.DimensionScores, category string, numCorrections uint32) *types.AlignmentHealthIndex {
	return &types.AlignmentHealthIndex{
		Height:               scores.Height,
		CompositeScore:       scores.Composite,
		Category:             category,
		DimensionalScores:    scores,
		CorrectionsGenerated: numCorrections,
	}
}
