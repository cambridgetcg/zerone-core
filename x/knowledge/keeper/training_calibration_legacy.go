package keeper

import "context"

import "github.com/zerone-chain/zerone/x/knowledge/types"

// computeTrainingValueWeightLegacy is verbatim from
// bc2a58fccb3e9bbb43647b0e51490fad034f34fd:training_economics.go.
func (k Keeper) computeTrainingValueWeightLegacy(ctx context.Context, factID string) TrainingValueWeightBreakdown {
	var out TrainingValueWeightBreakdown

	// Is-ought wall: ids resolving to NormativeCommitments earn nothing.
	if _, ok := k.GetNormativeCommitment(ctx, factID); ok {
		out.BlockedByIsOught = true
		return out
	}

	fact, ok := k.GetFact(ctx, factID)
	if !ok || fact == nil {
		return out
	}

	// A conjecture earns no training value in ANY status. The status gate
	// below admits CONTESTED and CHALLENGED so ordinary facts stay payable
	// through adjudication — but a conjecture reaching either of those is
	// routine, not a dispute about something believed, and the expired-round
	// bypass can park it in CHALLENGED indefinitely.
	if IsConjecture(fact) {
		out.StatusIneligible = true
		return out
	}

	// Clawback state: disproven → revenue zeroed.
	if fact.Status == types.FactStatus_FACT_STATUS_DISPROVEN || fact.RevenueClawbackBlock > 0 {
		out.Disproven = true
		return out
	}

	// Status gate: only facts whose truth-chain cleared verification (or
	// survive mid-adjudication) can accrue training value. PENDING and
	// PROVISIONAL never passed a round; REVOKED / EXPIRED / PRUNED /
	// SUPERSEDED / AT_RISK have been invalidated or retired. UNSPECIFIED
	// is a malformed record. All earn zero TVW. CONTESTED and CHALLENGED
	// remain payable during their adjudication window — falsification
	// closes the door via the DISPROVEN branch above.
	switch fact.Status {
	case types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_CHALLENGED:
		// eligible
	default:
		out.StatusIneligible = true
		return out
	}

	// Base: survived falsification attempts + 1. Popper, not popularity.
	out.BaseWeight = fact.CorroborationCount + 1

	// Methodology normalization.
	out.MethodologyMultiplier = k.getMethodologyNormalizationBps(ctx, fact.MethodId)

	// Vindication multiplier: facts vindicated from minority status.
	out.VindicationMultiplier = bps
	if k.factWasVindicatedFromMinority(ctx, fact.Id) {
		params, _ := k.GetParams(ctx)
		mult := params.VindicationTvwMultiplierBps
		if mult == 0 {
			mult = 2_500_000
		}
		out.VindicationMultiplier = mult
	}

	// Calibration snapshot at submission (frozen at fact acceptance; non-gameable).
	// Facts created via createFactFromClaim always carry a snapshot. Legacy
	// records (pre-snapshot) or facts seeded via authority injection default
	// to neutral so they neither farm retroactive calibration nor punish the
	// submitter for it.
	out.SubmitterCalibration = fact.SubmitterCalibrationSnapshotBps
	if out.SubmitterCalibration == 0 {
		out.SubmitterCalibration = bps / 2
	}

	// Axiom proximity: axioms earn more than deep derivations. Simple
	// linear decay 1.0× at depth 0 → 0.5× at depth 10+.
	out.AxiomProximity = axiomProximityBps(fact.AxiomDistance)

	// Hardening: accelerating return on corroboration. BaseWeight already
	// scales linearly (each survived attack adds 1); hardening adds a
	// multiplicative bonus that tops out at the cap. The combination
	// rewards facts that have run a long gauntlet exponentially more
	// than facts that have barely been tested, encouraging submitters to
	// welcome stress-testing of their own claims.
	out.HardeningMultiplier = bps + fact.CorroborationCount*hardeningPerCorroboration
	if out.HardeningMultiplier > hardeningMaxBps {
		out.HardeningMultiplier = hardeningMaxBps
	}

	// Counterexample multiplier (commitment 15: counterexamples are
	// part of the corpus). Facts with at least one VALIDATED
	// counterexample carry alignment-by-structure context and earn a
	// configurable bonus (default 1.2× from x/counterexamples params).
	// Without the counterexamples module wired, the multiplier
	// degrades to 1.0 — the chain stays correct but offers no
	// alignment-by-structure premium.
	out.CounterexampleMultiplier = bps
	if k.counterexampleKeeper != nil && k.counterexampleKeeper.HasValidatedCounterexample(ctx, fact.Id) {
		mult := k.counterexampleKeeper.GetTvwMultiplierBps(ctx)
		if mult > 0 {
			out.CounterexampleMultiplier = mult
		}
	}

	// Compose: final = base × meth × vind × cal × proximity × hardening × counterexample.
	final := uint64(out.BaseWeight) * bps
	final = safeMulDivTVW(final, out.MethodologyMultiplier, bps)
	final = safeMulDivTVW(final, out.VindicationMultiplier, bps)
	final = safeMulDivTVW(final, out.SubmitterCalibration, bps)
	final = safeMulDivTVW(final, out.AxiomProximity, bps)
	final = safeMulDivTVW(final, out.HardeningMultiplier, bps)
	final = safeMulDivTVW(final, out.CounterexampleMultiplier, bps)
	out.Final = final
	return out
}
