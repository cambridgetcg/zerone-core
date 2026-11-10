package app

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
)

// consensusRepairsV2HealthShift states what consensus-repairs-v2 changes in
// alignment pacing (review finding F5). v2 switches Q1's not-observed
// neutrality and #122's bank-supply denominator together, so its first
// scheduled alignment observation can move the health category, and with it
// the knowledge claim cooldown and minimum review fee, the capture_defense
// risk-analysis interval, the alignment observation cadence and gov's
// expedited voting for knowledge, alignment and capture_defense parameter LIPs.
//
// The prediction enables both markers on a discarded cache and runs the same
// keeper steps as alignment's EndBlock (input selection, registry presence,
// observation, scores, category) against the stored previous category. It
// writes nothing that survives and emits no event that survives. Sensors read
// at this height; the real observation happens at the next scheduled height,
// so knowledge and governance readings can drift in between. An error here is
// one the first post-v2 observation would also meet, so the caller refuses.
func (app *ZeroneApp) consensusRepairsV2HealthShift(ctx sdk.Context, registered, active int, stake *big.Int) (string, error) {
	before := app.AlignmentKeeper.GetState(ctx)
	beforeGov := app.AlignmentKeeper.GetHealthCategory(ctx)
	beforeCreation, beforeAnalysis := app.AlignmentKeeper.GetGlobalPacingMultiplier(ctx)
	beforeComposite := "none"
	if latest := app.AlignmentKeeper.GetRecentHealthIndices(ctx, 1); len(latest) > 0 {
		beforeComposite = fmt.Sprint(latest[0].CompositeScore)
	}
	beforeFee := app.KnowledgeKeeper.GetEffectiveMinReviewFee(ctx)
	mint := app.VestingRewardsKeeper.GetTotalMinted(ctx)
	supply := app.BankKeeper.GetSupply(ctx, "uzrn").Amount.BigInt()

	cache, _ := ctx.CacheContext()
	if err := app.AlignmentKeeper.EnableUnobservedHealth(cache); err != nil {
		return "", err
	}
	if err := app.VestingRewardsKeeper.EnableAlignmentBankSupply(cache); err != nil {
		return "", err
	}
	selected, err := app.AlignmentKeeper.SelectInputs(cache)
	if err != nil {
		return "", err
	}
	params := app.AlignmentKeeper.GetParams(selected)
	scheduled := app.AlignmentKeeper.IsEnabled(selected) && params.ObservationIntervalBlocks != 0
	afterCategory, afterComposite, afterGov := before.PreviousCategory, beforeComposite, beforeGov
	if scheduled {
		observed, err := app.AlignmentKeeper.HasHealthObservation(selected)
		if err != nil {
			return "", fmt.Errorf("%w: %w", alignment.ErrHealthObservationFault, err)
		}
		scores := app.AlignmentKeeper.ComputeScores(selected, app.AlignmentKeeper.ObserveAll(selected))
		afterComposite = fmt.Sprint(scores.Composite)
		afterCategory = alignment.CategoryNotObserved
		if observed {
			afterCategory = app.AlignmentKeeper.CategorizeObservedHealth(selected, scores.Composite, before.PreviousCategory)
		}
		afterGov = afterCategory // the observation's health index is the newest one gov reads
		next := app.AlignmentKeeper.GetState(selected)
		next.PreviousCategory = afterCategory
		next.DegradedFrequencyActive = afterCategory == alignment.CategoryDegraded || afterCategory == alignment.CategoryCritical
		app.AlignmentKeeper.SetState(selected, next)
	}
	after := app.AlignmentKeeper.GetState(selected)
	afterCreation, afterAnalysis := app.AlignmentKeeper.GetGlobalPacingMultiplier(selected)
	afterFee := app.KnowledgeKeeper.GetEffectiveMinReviewFee(selected)

	knowledgeParams, err := app.KnowledgeKeeper.GetParams(ctx)
	if err != nil {
		return "", err
	}
	cooldown := func(creation uint64) uint64 { // x/knowledge GetEffectiveCooldown, before domain pressure
		base := knowledgeParams.ClaimCooldownBlocks
		if creation == 0 || creation == alignment.BPS {
			return base
		}
		return consensusRepairsMulDiv(base, alignment.BPS, creation)
	}
	riskInterval := func(analysis uint64) uint64 { // x/capture_defense EndBlock
		base := app.CaptureDefenseKeeper.GetParams(ctx).RiskAnalysisInterval
		if analysis == 0 || analysis == alignment.BPS {
			return base
		}
		if interval := consensusRepairsMulDiv(base, alignment.BPS, analysis); interval > 0 || base == 0 {
			return interval
		}
		return 1
	}
	expedited := func(category string) bool { // x/gov getEffectiveVotingPeriod
		return category == alignment.CategoryDegraded || category == alignment.CategoryCritical
	}
	voting := func(category string) uint64 {
		base := app.ZeroneGovKeeper.GetParams(ctx).VotingPeriodBlocks
		if !expedited(category) {
			return base
		}
		return max(base/2, 1)
	}
	return fmt.Sprintf("first scheduled observation after v2 (predicted at this height): category %s->%s composite %s->%s"+
		" creation_pacing_bps %d->%d analysis_pacing_bps %d->%d degraded_frequency %t->%t;"+
		" claim_cooldown_blocks %d->%d (before domain pressure) min_review_fee %s->%s risk_analysis_interval_blocks %d->%d;"+
		" knowledge/alignment/capture_defense parameter LIP voting_period_blocks %d->%d (expedited %t->%t; gov read %s before);"+
		" inputs registered=%d active=%d active_stake=%s mint_ledger=%s bank_supply=%s legacy_staking_ratio_bps=%d fixed_staking_ratio_bps=%d;"+
		" scheduled=%t; stored categories and observation history retained",
		before.PreviousCategory, afterCategory, beforeComposite, afterComposite,
		beforeCreation, afterCreation, beforeAnalysis, afterAnalysis, before.DegradedFrequencyActive, after.DegradedFrequencyActive,
		cooldown(beforeCreation), cooldown(afterCreation), beforeFee, afterFee, riskInterval(beforeAnalysis), riskInterval(afterAnalysis),
		voting(beforeGov), voting(afterGov), expedited(beforeGov), expedited(afterGov), beforeGov,
		registered, active, stake, mint, supply, consensusRepairsRatio(stake, mint), consensusRepairsRatio(stake, supply),
		scheduled), nil
}

func consensusRepairsMulDiv(a, b, c uint64) uint64 {
	result := new(big.Int).Mul(new(big.Int).SetUint64(a), new(big.Int).SetUint64(b))
	result.Div(result, new(big.Int).SetUint64(c))
	if !result.IsUint64() {
		return ^uint64(0)
	}
	return result.Uint64()
}
