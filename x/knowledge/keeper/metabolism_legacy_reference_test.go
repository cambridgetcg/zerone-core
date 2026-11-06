package keeper

// The f787860c metabolism and carrying-capacity path, verbatim apart from
// these four renamed functions and the calls between them. It is the
// reference DET-4's cached ProcessMetabolism is checked against: same store
// hash and same events from the same prestate. Generated from
// `git show f787860c:x/knowledge/keeper/{metabolism,carrying_capacity}.go`.

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ProcessMetabolism runs one epoch of energy accounting for all active facts.
// Deducts maintenance cost, adds energy income, and transitions fact status
// based on energy levels.
func (k Keeper) LegacyProcessMetabolismForTest(ctx context.Context, epoch uint64) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	// Collect facts to process (avoid modifying store during iteration)
	var factsToProcess []*types.Fact
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		// R-doctrine (2026-07-12 framework critique): doctrine lives by process —
		// the hash-pinned creed + gov-gated amendment LIP — not by metabolism.
		// Starvation is not falsification (the same "error is not deceit" shape
		// as commitment C2). Exempt the doctrinal stratum from the energy
		// lifecycle entirely. Match Category/MethodId (not Stratum) because
		// ontology stratum names are copied onto ordinary facts (rounds.go).
		if fact.Category == types.DoctrineCategory || fact.MethodId == types.DoctrineMethodId {
			return false
		}
		// Process all facts that are alive (not already pruned)
		if fact.Status == types.FactStatus_FACT_STATUS_VERIFIED ||
			fact.Status == types.FactStatus_FACT_STATUS_ACTIVE ||
			fact.Status == types.FactStatus_FACT_STATUS_PROVISIONAL ||
			fact.Status == types.FactStatus_FACT_STATUS_AT_RISK ||
			fact.Status == types.FactStatus_FACT_STATUS_EXPIRED {
			factsToProcess = append(factsToProcess, fact)
		}
		return false
	})

	domainCounts := k.CountFactsByDomain(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	changedDomains := make(map[string]bool) // track domains with status changes (R29-1)

	for _, fact := range factsToProcess {
		// ─── Calculate maintenance cost ───────────────────
		cost := k.calculateMaintenanceCost(fact, params, domainCounts)

		// Apply domain carrying capacity death pressure (R29-1)
		deathMultiplier := k.legacyDeathPressureMultiplierForTest(ctx, fact.Domain)
		cost = safeMulDiv(cost, deathMultiplier, BPSCapacity)

		// ─── Calculate energy income ──────────────────────
		income := k.calculateEnergyIncome(ctx, fact, params)

		// ─── Update energy ────────────────────────────────
		newEnergy := fact.Energy + income
		if cost > newEnergy {
			newEnergy = 0
		} else {
			newEnergy -= cost
		}
		if newEnergy > params.MetabolismEnergyCap {
			newEnergy = params.MetabolismEnergyCap
		}

		// ─── State transitions (multi-level thresholds) ──────────
		oldStatus := fact.Status

		if newEnergy >= params.MetabolismActiveThreshold {
			// Healthy — clear any at-risk state
			if fact.AtRiskSinceEpoch > 0 {
				fact.AtRiskSinceEpoch = 0
				fact.Status = RestoredStatusFor(fact)
			}
		} else if newEnergy < params.MetabolismActiveThreshold {
			// Below active threshold
			if fact.AtRiskSinceEpoch == 0 {
				// Just entered at-risk zone
				fact.AtRiskSinceEpoch = epoch
				fact.Status = types.FactStatus_FACT_STATUS_AT_RISK
			} else {
				// Already at risk — check for expiry/pruning
				atRiskDuration := epoch - fact.AtRiskSinceEpoch
				if atRiskDuration >= params.MetabolismAtRiskEpochs+params.MetabolismExpiredToPrunedEpochs {
					fact.Status = types.FactStatus_FACT_STATUS_PRUNED
				} else if atRiskDuration >= params.MetabolismAtRiskEpochs {
					fact.Status = types.FactStatus_FACT_STATUS_EXPIRED
				}
			}
		}

		fact.Energy = newEnergy
		fact.EnergyLastUpdated = uint64(sdkCtx.BlockHeight())
		if err := k.SetFact(ctx, fact); err != nil {
			k.Logger(ctx).Error("failed to update fact energy", "fact_id", fact.Id, "error", err)
			continue
		}

		// Emit event on status change and update domain stats (R29-1)
		if oldStatus != fact.Status {
			switch {
			case fact.Status == types.FactStatus_FACT_STATUS_AT_RISK:
				k.TransitionDomainFactStatus(ctx, fact.Domain, false) // active → at-risk
			case fact.Status == types.FactStatus_FACT_STATUS_ACTIVE &&
				(oldStatus == types.FactStatus_FACT_STATUS_AT_RISK || oldStatus == types.FactStatus_FACT_STATUS_EXPIRED):
				k.TransitionDomainFactStatus(ctx, fact.Domain, true) // at-risk → active
			case fact.Status == types.FactStatus_FACT_STATUS_EXPIRED ||
				fact.Status == types.FactStatus_FACT_STATUS_PRUNED:
				wasActive := oldStatus == types.FactStatus_FACT_STATUS_ACTIVE ||
					oldStatus == types.FactStatus_FACT_STATUS_VERIFIED ||
					oldStatus == types.FactStatus_FACT_STATUS_PROVISIONAL
				k.DecrementDomainFactCount(ctx, fact.Domain, wasActive, fact.Energy)
			}
			k.emitMetabolismStatusEvent(ctx, fact, oldStatus, epoch)
			changedDomains[fact.Domain] = true
		}
	}

	// Emit domain pressure events for domains that changed (R29-1)
	for domain := range changedDomains {
		k.EmitDomainPressureEvent(ctx, domain)
	}

	// Reset epoch citation counters for all facts
	k.ResetEpochCitationCounters(ctx)

	k.Logger(ctx).Info("metabolism processed",
		"facts_processed", len(factsToProcess),
		"epoch", epoch,
	)

	return nil
}

// GetDeathPressureMultiplier returns the decay multiplier for a domain.
// >1M BPS = faster decay (overcrowded), <1M BPS = slower decay (sparse), 1M = normal.
func (k Keeper) legacyDeathPressureMultiplierForTest(ctx context.Context, domain string) uint64 {
	params, err := k.GetParams(ctx)
	if err != nil || params.DomainBaseCapacity == 0 {
		return BPSCapacity
	}
	pressure := k.legacyDomainPressureForTest(ctx, domain)

	if pressure > BPSCapacity {
		// Overcrowded: accelerate decay
		excess := pressure - BPSCapacity
		return BPSCapacity + safeMulDiv(excess, params.OvercrowdingDecayMultiplierBps-BPSCapacity, BPSCapacity)
	} else if pressure < BPSCapacity/2 {
		// Very sparse (< 50% capacity): slow decay to 75%
		return BPSCapacity * 3 / 4
	}
	return BPSCapacity // normal range
}

func (k Keeper) legacyDomainPressureForTest(ctx context.Context, domain string) uint64 {
	stats, _ := k.GetDomainStats(ctx, domain)
	capacity := k.legacyDomainCarryingCapacityForTest(ctx, domain)
	if capacity == 0 {
		return BPSCapacity
	}
	population := stats.ActiveCount + stats.AtRiskCount
	return safeMulDiv(population, BPSCapacity, capacity)
}

func (k Keeper) legacyDomainCarryingCapacityForTest(ctx context.Context, domain string) uint64 {
	params, _ := k.GetParams(ctx)
	base := params.DomainBaseCapacity
	if base == 0 {
		base = 1000 // safety default
	}
	inbound := k.GetInboundCrossDomainCitationCount(ctx, domain)
	bonus := inbound * params.DomainCapacityGrowthPerCitation

	// R31-5: Water → Wood — mentorship graduations expand carrying capacity.
	stats, _ := k.GetDomainStats(ctx, domain)
	mentorshipBonus := stats.MentorshipGraduations * params.MentorshipCapacityBonus

	total := base + bonus + mentorshipBonus

	// Metal controls Wood: capture flag penalty reduces capacity (R31-1)
	if k.captureDefenseKeeper != nil {
		flagged, penaltyBps := k.captureDefenseKeeper.GetDomainCapturePenalty(ctx, domain)
		if flagged {
			reduction := safeMulDiv(total, penaltyBps, BPSCapacity)
			if reduction >= total {
				return 1 // minimum capacity — can't go to zero
			}
			total = total - reduction
		}
	}

	// R31-4: Metal controls Wood — stratum depth constrains carrying capacity.
	stratumMultiplier := k.getStratumCapacityMultiplier(ctx, domain)
	if stratumMultiplier < BPSCapacity {
		effectiveCapacity := total * stratumMultiplier / BPSCapacity

		sdkCtx := sdk.UnwrapSDKContext(ctx)
		depth := uint32(1)
		if k.ontologyKeeper != nil {
			if d, err := k.ontologyKeeper.GetDepthForDomain(ctx, domain); err == nil {
				depth = d
			}
		}
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.stratum_capacity_applied",
			sdk.NewAttribute("domain", domain),
			sdk.NewAttribute("stratum_depth", fmt.Sprintf("%d", depth)),
			sdk.NewAttribute("capacity_multiplier_bps", fmt.Sprintf("%d", stratumMultiplier)),
			sdk.NewAttribute("effective_capacity", fmt.Sprintf("%d", effectiveCapacity)),
		))

		return effectiveCapacity
	}

	return total
}
