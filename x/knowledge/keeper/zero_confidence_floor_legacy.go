package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// advanceConfidenceLegacy is verbatim from
// bc2a58fccb3e9bbb43647b0e51490fad034f34fd:epistemic_temperature.go.
func (k Keeper) advanceConfidenceLegacy(ctx context.Context) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	baseGrowthRate := params.ConfidenceGrowthPerEpochBps
	if baseGrowthRate == 0 {
		return nil // growth disabled
	}

	// Cache epistemic states per domain to avoid repeated lookups
	domainGrowthRates := make(map[string]uint64)

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		// Only grow confidence for active/verified facts.
		//
		// PROVISIONAL is deliberately excluded. Growth below is
		// `confidence += confidence * rate`, floored at a minimum of 1 —
		// so a node sitting at zero does not stay at zero, it climbs one
		// unit every epoch on the passage of time alone. For an ordinary
		// fact that floor is a rounding convenience. For a conjecture it
		// would be the chain manufacturing belief in a proposition nobody
		// has verified, which is the precise failure commitment 3 names:
		// popularity, not Popper. A conjecture's standing may only move
		// when something actually happens to it.
		//
		// This branch was unreachable before conjectures existed — nothing
		// in the module ever wrote PROVISIONAL — so no live fact changes
		// behaviour here.
		switch fact.Status {
		case types.FactStatus_FACT_STATUS_VERIFIED,
			types.FactStatus_FACT_STATUS_ACTIVE:
		default:
			return false
		}

		growthRate, ok := domainGrowthRates[fact.Domain]
		if !ok {
			growthRate = baseGrowthRate
			epistemicState, found, err := k.GetDomainEpistemicState(ctx, fact.Domain)
			if err == nil && found {
				// Hot domains: confidence grows faster
				if epistemicState.Temperature > 700_000 && params.EpistemicHotConfidenceGrowthBps > 0 {
					growthRate = safeMulDiv(growthRate, params.EpistemicHotConfidenceGrowthBps, BPS)
				}
				// Cold domains: confidence grows slower (50% rate)
				if epistemicState.Temperature < 300_000 {
					growthRate = safeMulDiv(growthRate, 500_000, BPS)
				}
			}
			domainGrowthRates[fact.Domain] = growthRate
		}

		// Apply growth: confidence += confidence * growthRate / BPS
		growth := safeMulDiv(fact.Confidence, growthRate, BPS)
		if growth == 0 {
			growth = 1 // minimum 1 BPS growth per epoch
		}
		fact.Confidence += growth

		// Clamp to effective cap (includes epistemic temperature modulation)
		fact.Confidence = k.ClampConfidence(ctx, fact.Confidence, fact.Domain)

		if setErr := k.SetFact(ctx, fact); setErr != nil {
			return false
		}
		return false
	})

	return nil
}
