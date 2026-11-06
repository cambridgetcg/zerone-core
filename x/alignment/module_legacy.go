package alignment

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/alignment/keeper"
	"github.com/zerone-chain/zerone/x/alignment/types"
)

// endBlockLegacy preserves the EndBlock body verbatim from
// 6a56b621082064483af1ced7718462766d9ae756. Do not route it through Q1 helpers.
func (am AppModule) endBlockLegacy(ctx context.Context) error {
	var err error
	ctx, err = am.keeper.SelectInputs(ctx)
	if err != nil {
		return err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	// Skip if disabled or halted.
	if !am.keeper.IsEnabled(ctx) {
		return nil
	}
	if am.keeper.IsHalted(ctx) {
		return nil
	}

	params := am.keeper.GetParams(ctx)
	if params.ObservationIntervalBlocks == 0 {
		return nil
	}

	// Compute effective interval: confidence-modulated, then halved when degraded frequency is active.
	state := am.keeper.GetState(ctx)
	effectiveInterval := am.keeper.GetEffectiveObservationInterval(ctx)
	if state.DegradedFrequencyActive && effectiveInterval > 1 {
		effectiveInterval = effectiveInterval / 2
	}
	// The modulo below panics on zero, and a panic in EndBlock halts the chain.
	if effectiveInterval == 0 {
		effectiveInterval = 1
	}

	if height%effectiveInterval != 0 {
		return nil
	}

	// 1. Observe
	obs := am.keeper.ObserveAll(ctx)
	am.keeper.SetObservation(ctx, obs)

	// 2. Score
	scores := am.keeper.ComputeScores(ctx, obs)
	am.keeper.SetScores(ctx, scores)

	// 2.5 Evaluate pending correction outcomes from previous observation (R29-4).
	am.keeper.EvaluatePendingCorrections(ctx, scores)

	// 3. Corrections
	corrections := am.keeper.GenerateCorrections(ctx, scores)
	am.keeper.ApplyCorrections(ctx, corrections)

	// 4. Health index
	category := am.keeper.CategorizeHealth(ctx, scores.Composite)
	hi := am.keeper.BuildHealthIndex(scores, category, uint32(len(corrections)))
	am.keeper.SetHealthIndex(ctx, hi)

	// 4.5 Emit correction confidence event (R29-4).
	confidence := am.keeper.GetCorrectionConfidence(ctx)
	effectiveMaxMag := am.keeper.GetEffectiveMaxMagnitude(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.alignment.correction_confidence_updated",
			sdk.NewAttribute("confidence_bps", fmt.Sprintf("%d", confidence)),
			sdk.NewAttribute("effective_max_magnitude", fmt.Sprintf("%d", effectiveMaxMag)),
			sdk.NewAttribute("category", keeper.CategorizeConfidence(confidence)),
		),
	)

	// 5. Health transition responses
	// Compute pacing for event enrichment (R29-6).
	creationPacing, analysisPacing := am.keeper.GetGlobalPacingMultiplier(ctx)

	previousCategory := state.PreviousCategory
	if previousCategory != "" && previousCategory != category {
		switch {
		case category == types.CategoryDegraded && previousCategory == types.CategoryHealthy:
			state.DegradedFrequencyActive = true
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.alignment.network_health_degraded",
					sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
					sdk.NewAttribute("composite", fmt.Sprintf("%d", scores.Composite)),
					sdk.NewAttribute("creation_multiplier_bps", fmt.Sprintf("%d", creationPacing)),
					sdk.NewAttribute("analysis_multiplier_bps", fmt.Sprintf("%d", analysisPacing)),
				),
			)
		case category == types.CategoryCritical:
			state.DegradedFrequencyActive = true
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.alignment.network_health_critical",
					sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
					sdk.NewAttribute("composite", fmt.Sprintf("%d", scores.Composite)),
					sdk.NewAttribute("creation_multiplier_bps", fmt.Sprintf("%d", creationPacing)),
					sdk.NewAttribute("analysis_multiplier_bps", fmt.Sprintf("%d", analysisPacing)),
				),
			)
		case category == types.CategoryHealthy && (previousCategory == types.CategoryDegraded || previousCategory == types.CategoryCritical):
			state.DegradedFrequencyActive = false
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.alignment.network_health_recovered",
					sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
					sdk.NewAttribute("composite", fmt.Sprintf("%d", scores.Composite)),
					sdk.NewAttribute("creation_multiplier_bps", fmt.Sprintf("%d", creationPacing)),
					sdk.NewAttribute("analysis_multiplier_bps", fmt.Sprintf("%d", analysisPacing)),
				),
			)
		}
	}
	state.PreviousCategory = category

	// 6. Update state
	state.LastObservationHeight = height
	state.ObservationCount++
	am.keeper.SetState(ctx, state)

	// 6.5 Prune old correction outcomes (R29-4).
	am.keeper.PruneOldOutcomes(ctx)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.alignment.observation_recorded",
			sdk.NewAttribute("height", fmt.Sprintf("%d", height)),
			sdk.NewAttribute("composite_score", fmt.Sprintf("%d", scores.Composite)),
			sdk.NewAttribute("category", category),
			sdk.NewAttribute("correction_count", fmt.Sprintf("%d", len(corrections))),
			sdk.NewAttribute("observation_count", fmt.Sprintf("%d", state.ObservationCount)),
			sdk.NewAttribute("creation_multiplier_bps", fmt.Sprintf("%d", creationPacing)),
			sdk.NewAttribute("analysis_multiplier_bps", fmt.Sprintf("%d", analysisPacing)),
		),
	)

	// Emit growth pressure event if applicable (R31-1)
	am.keeper.EmitGrowthPressureEvent(sdkCtx, obs.KnowledgeQuality)

	am.keeper.Logger(ctx).Info("alignment observation complete",
		"height", height,
		"composite", scores.Composite,
		"category", category,
		"corrections", len(corrections),
	)

	return nil
}
