package types

import "fmt"

// BPS scale: 1,000,000 = 100% = 1.0x multiplier.
const BPSScale = uint64(1_000_000)

// DefaultParams returns the default autopoiesis module parameters.
func DefaultParams() Params {
	return Params{
		EpochLengthBlocks:    100,
		MaxChangePerEpochBps: 10_000,    // 1% per epoch
		SlashMultiplierMin:   500_000,   // 0.5x floor
		SlashMultiplierMax:   2_000_000, // 2.0x ceiling
		SsiCriticalThreshold: 250_000,
		SsiStressedThreshold: 500_000,
		SsiHealthyThreshold:  750_000,
		Enabled:              true,

		// Damping & oscillation control (T8).
		SsiSmoothingAlphaBps:        200_000, // 0.2 = heavy smoothing
		TargetDeadZoneBps:           50_000,  // 5% dead zone
		OscillationWindowEpochs:     20,
		OscillationSignFlipThreshold: 10,     // 50% flip rate in window
		OscillationFreezeEpochs:     50,

		// Cross-module change budget (L7).
		MaxTotalChangeBpsPerEpoch: 20_000, // 2% total adjustment across all multipliers
	}
}

// Validate validates the module parameters.
func (p *Params) Validate() error {
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("epoch_length_blocks must be > 0")
	}
	if p.MaxChangePerEpochBps > BPSScale {
		return fmt.Errorf("max_change_per_epoch_bps must be <= %d, got %d", BPSScale, p.MaxChangePerEpochBps)
	}
	if p.SlashMultiplierMin > p.SlashMultiplierMax {
		return fmt.Errorf("slash_multiplier_min (%d) > slash_multiplier_max (%d)", p.SlashMultiplierMin, p.SlashMultiplierMax)
	}
	if p.SsiCriticalThreshold > p.SsiStressedThreshold {
		return fmt.Errorf("ssi_critical_threshold (%d) > ssi_stressed_threshold (%d)", p.SsiCriticalThreshold, p.SsiStressedThreshold)
	}
	if p.SsiStressedThreshold > p.SsiHealthyThreshold {
		return fmt.Errorf("ssi_stressed_threshold (%d) > ssi_healthy_threshold (%d)", p.SsiStressedThreshold, p.SsiHealthyThreshold)
	}
	if p.SsiHealthyThreshold > BPSScale {
		return fmt.Errorf("ssi_healthy_threshold (%d) > %d", p.SsiHealthyThreshold, BPSScale)
	}
	return nil
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:      &p,
		Multipliers: DefaultMultipliers(),
		Snapshots:   nil,
		Activated:   false,
	}
}

// DefaultMultipliers returns the initial multiplier set.
func DefaultMultipliers() []*MultiplierState {
	return []*MultiplierState{
		{
			Path:       "rewards.block",
			CurrentBps: BPSScale, // 1.0x
			TargetBps:  BPSScale,
			MinBps:     500_000,   // 0.5x
			MaxBps:     2_000_000, // 2.0x
		},
		{
			Path:       "slashing.severity",
			CurrentBps: BPSScale,
			TargetBps:  BPSScale,
			MinBps:     500_000,
			MaxBps:     2_000_000,
		},
		{
			Path:       "fees.base",
			CurrentBps: BPSScale,
			TargetBps:  BPSScale,
			MinBps:     500_000,
			MaxBps:     2_000_000,
		},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, m := range gs.Multipliers {
		if m.Path == "" {
			return fmt.Errorf("multiplier path cannot be empty")
		}
		if seen[m.Path] {
			return fmt.Errorf("duplicate multiplier path: %s", m.Path)
		}
		seen[m.Path] = true
		if m.MinBps > m.MaxBps {
			return fmt.Errorf("multiplier %s: min_bps (%d) > max_bps (%d)", m.Path, m.MinBps, m.MaxBps)
		}
		if m.CurrentBps < m.MinBps || m.CurrentBps > m.MaxBps {
			return fmt.Errorf("multiplier %s: current_bps (%d) outside [%d, %d]", m.Path, m.CurrentBps, m.MinBps, m.MaxBps)
		}
	}
	return nil
}

// AutopoiesisState tracks the module's runtime state.
type AutopoiesisState struct {
	Activated       bool   `json:"activated"`
	CurrentEpoch    uint64 `json:"current_epoch"`
	LastEpochHeight uint64 `json:"last_epoch_height"`

	// Damping / oscillation state (T8).
	SmoothedSsi                  uint64 `json:"smoothed_ssi,omitempty"`
	LastRawSsi                   uint64 `json:"last_raw_ssi,omitempty"`
	DeltaSignBitmap              uint64 `json:"delta_sign_bitmap,omitempty"` // bit i = 1 if epoch-i delta > 0
	OscillationFrozenUntilEpoch  uint64 `json:"oscillation_frozen_until_epoch,omitempty"`
}
