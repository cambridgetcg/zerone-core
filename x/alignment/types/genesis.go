package types

import "fmt"

// BPS is the basis-point denominator (1,000,000 = 100%).
const BPS = uint64(1_000_000)

// DefaultParams returns the default alignment module parameters.
//
// These values express commitments 11 (per-system trust queryability)
// and 12 (chain pays for its own audit). See doc.go for the contract;
// the values below are the alignment layer's posture about its own
// reliability — equal-weighted sensors, conservative auto-apply,
// confidence-gated corrections.
func DefaultParams() Params {
	return Params{
		// ObservationInterval of 100 blocks (~4 minutes) means the
		// chain re-senses itself frequently without overwhelming
		// computation. Sensing is cheap; reacting on every reading
		// is what creates noise.
		ObservationIntervalBlocks:    100,

		// Sensor weights: equal 20% across all five dimensions. This
		// is the default posture — knowledge quality, economic
		// stability, governance participation, network security,
		// staking ratio. We do not pre-judge which dimension matters
		// most; governance can re-weight, but the prior is "all five
		// matter equally." The alternative — weighting one sensor
		// 60% — would make the chain blind to dimensions it had
		// pre-decided don't count.
		WeightKnowledgeQuality:       200_000, // 20% — does verification work cleanly?
		WeightEconomicStability:      200_000, // 20% — are rewards still motivating honest work?
		WeightGovernanceParticipation: 200_000, // 20% — is the chain governed or abandoned?
		WeightNetworkSecurity:        200_000, // 20% — is stake concentration acceptable?
		WeightStakingRatio:           200_000, // 20% — is enough capital committed?

		// Health bands: < 20% CRITICAL, < 40% DEGRADED, > 70%
		// HEALTHY. The wide gap (40%-70%) between DEGRADED and
		// HEALTHY is intentional — a chain whose composite score
		// fluctuates between 50% and 60% should not flip its
		// reported posture every observation. Only emerging fully
		// into HEALTHY territory or falling fully into DEGRADED
		// counts as a state change.
		CriticalThreshold:            200_000, // 20%
		DegradedThreshold:            400_000, // 40%
		HealthyThreshold:             700_000, // 70%
		Enabled:                      true,

		// MaxAutoApplyMagnitudeBps caps how much a single auto-apply
		// can shift things. 50% is the testnet default — conservative
		// enough that the controller cannot single-handedly halve or
		// double an emission rate, even if all the other safety bounds
		// failed. This is the last line of defence before
		// misconfigured sensors could damage the chain.
		MaxAutoApplyMagnitudeBps:             500_000,   // 50% — conservative testnet default

		// Correction confidence: corrections only auto-apply when
		// the controller has accumulated enough samples to trust the
		// signal. 50-sample window with 5-sample minimum means: do
		// not act on a single observation, do not act on the first
		// few observations, only act when the trend is established.
		CorrectionConfidenceWindowSize:       50,
		CorrectionConfidenceMinSamples:       5,
		MinConfidenceForAutoApply:            200_000,   // 20% confidence floor

		// Correction bounds: multipliers cannot move outside [30%, 200%].
		// The asymmetry is deliberate: the chain can be nudged toward
		// 30% as much as toward 200%, but it cannot be turned off
		// entirely (multiplier ≤ 0) or amplified to absurdity
		// (multiplier > 2×). Crisis response stays inside boundaries
		// the chain pre-committed to.
		CorrectionBoundsMinMultiplierBps:     300_000,   // 30% floor
		CorrectionBoundsMaxMultiplierBps:     2_000_000, // 200% ceiling

		// Correction banding (L7): below this magnitude, corrections
		// are advisory-only (logged + event), not forwarded to
		// autopoiesis. Below 3%, the chain still records what it
		// observed but does not act — small deviations should not
		// chatter the regulatory layer. The audit log captures
		// everything; the action gate is intentionally higher than
		// the observation gate.
		AdvisoryMagnitudeBps: 30_000, // 3% — observe but do not act
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params: &params,
		State: &AlignmentState{
			Enabled: true,
		},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates the module parameters.
func (p *Params) Validate() error {
	if p.ObservationIntervalBlocks == 0 {
		return ErrInvalidInterval
	}

	weightSum := p.WeightKnowledgeQuality +
		p.WeightEconomicStability +
		p.WeightGovernanceParticipation +
		p.WeightNetworkSecurity +
		p.WeightStakingRatio
	if weightSum != BPS {
		return ErrInvalidWeights
	}

	if p.CriticalThreshold > BPS || p.DegradedThreshold > BPS || p.HealthyThreshold > BPS {
		return ErrInvalidThreshold
	}

	if p.CriticalThreshold >= p.DegradedThreshold || p.DegradedThreshold >= p.HealthyThreshold {
		return ErrThresholdOrder
	}

	if p.MaxAutoApplyMagnitudeBps > BPS {
		return ErrInvalidMaxAutoApply
	}

	if p.CorrectionBoundsMinMultiplierBps > p.CorrectionBoundsMaxMultiplierBps {
		return ErrInvalidConfidenceBounds
	}

	// Cross-parameter safety (R30-2): max correction bounds × max magnitude must not exceed 100%.
	if p.CorrectionBoundsMaxMultiplierBps*p.MaxAutoApplyMagnitudeBps/BPS > BPS {
		return fmt.Errorf("max correction bounds multiplier * max_magnitude would exceed 100%%")
	}

	return nil
}
