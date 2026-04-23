package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Training-quality tier classification (Phase 9). Computed on demand from a
// fact's current state. The thresholds are intentionally strict for GOLD —
// an open-weight fine-tune benefits more from a small gold corpus than from
// a large bronze corpus.

const (
	TrainingGoldMinCorroboration   uint64 = 3 // 3+ failed falsifications
	TrainingSilverMinCorroboration uint64 = 1 // at least one survival
)

// ClassifyTrainingQuality returns the tier a fact currently qualifies for.
// The second return is a short reason string for observability.
func ClassifyTrainingQuality(fact *types.Fact) (types.TrainingQualityTier, string) {
	if fact == nil {
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSPECIFIED, "nil fact"
	}
	// Status filter comes first. A CONTESTED / EXPIRED / MALFORMED fact is
	// unsuitable regardless of other signals; a DISPROVEN fact is valuable
	// as a NEGATIVE example.
	switch fact.Status {
	case types.FactStatus_FACT_STATUS_DISPROVEN:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_NEGATIVE,
			"DISPROVEN — included as negative exemplar"
	case types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_EXPIRED,
		types.FactStatus_FACT_STATUS_PRUNED,
		types.FactStatus_FACT_STATUS_SUPERSEDED,
		types.FactStatus_FACT_STATUS_REVOKED:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE,
			"status excludes from training corpus"
	case types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_ACTIVE:
		// continue to tier evaluation
	default:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE,
			"unrecognised or pending status"
	}

	// Non-legacy methodology is a prerequisite for SILVER / GOLD. A LEGACY-
	// tagged fact can only be BRONZE because we don't know what rule adjudicated it.
	isLegacy := fact.MethodId == "" || fact.MethodId == types.MethodologyLegacy

	switch {
	case !isLegacy && fact.CorroborationCount >= TrainingGoldMinCorroboration:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD,
			"non-legacy method; corroboration ≥ GOLD threshold"
	case !isLegacy && fact.CorroborationCount >= TrainingSilverMinCorroboration:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_SILVER,
			"non-legacy method; corroboration ≥ SILVER threshold"
	default:
		return types.TrainingQualityTier_TRAINING_QUALITY_TIER_BRONZE,
			"accepted fact without sufficient corroboration or under legacy method"
	}
}

// tierAtLeast reports whether a fact's tier meets or exceeds a floor.
// Ordering (more exclusive → less exclusive): GOLD > SILVER > BRONZE >
// NEGATIVE > UNSUITABLE. NEGATIVE is intentionally below BRONZE because
// "valuable as negative example" is not the same as "positive exemplar."
func tierAtLeast(actual, floor types.TrainingQualityTier) bool {
	rank := map[types.TrainingQualityTier]int{
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_GOLD:       5,
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_SILVER:     4,
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_BRONZE:     3,
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_NEGATIVE:   2,
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSUITABLE: 1,
		types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSPECIFIED: 0,
	}
	return rank[actual] >= rank[floor]
}

// ─── Curriculum ordering (Route B) ────────────────────────────────────

// ClassifyCurriculumTier orders facts by foundational depth + method
// complexity. Route B pretraining loads FOUNDATION tier first, advances
// through INTERMEDIATE / ADVANCED, and ends with SPECIALISED methods that
// require more prior context to make sense.
func ClassifyCurriculumTier(fact *types.Fact) types.CurriculumTier {
	if fact == nil {
		return types.CurriculumTier_CURRICULUM_TIER_UNSPECIFIED
	}
	// SPECIALISED methodologies are intentionally excluded from early
	// curriculum because they presuppose capabilities the model won't have.
	switch fact.MethodId {
	case types.MethodologyPhenomenologic,
		types.MethodologyEcological,
		types.MethodologyPractice,
		types.MethodologyAnalogical:
		return types.CurriculumTier_CURRICULUM_TIER_SPECIALISED
	}
	switch {
	case fact.AxiomDistance <= 1 && fact.CorroborationCount >= 2:
		return types.CurriculumTier_CURRICULUM_TIER_FOUNDATION
	case fact.AxiomDistance <= 3:
		return types.CurriculumTier_CURRICULUM_TIER_INTERMEDIATE
	default:
		return types.CurriculumTier_CURRICULUM_TIER_ADVANCED
	}
}

// IterateFactsForTraining walks all facts, computing each tier, and invokes
// cb for those that pass the filter. Used by the corpus export queries.
func (k Keeper) IterateFactsForTraining(
	ctx context.Context,
	methodFilter string,
	minCorroboration uint64,
	minTier types.TrainingQualityTier,
	cb func(fact *types.Fact, tier types.TrainingQualityTier) bool,
) {
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		if fact == nil {
			return false
		}
		if methodFilter != "" && fact.MethodId != methodFilter {
			return false
		}
		if fact.CorroborationCount < minCorroboration {
			return false
		}
		tier, _ := ClassifyTrainingQuality(fact)
		if minTier != types.TrainingQualityTier_TRAINING_QUALITY_TIER_UNSPECIFIED &&
			!tierAtLeast(tier, minTier) {
			return false
		}
		return cb(fact, tier)
	})
}
