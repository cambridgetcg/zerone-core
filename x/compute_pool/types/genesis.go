package types

import "fmt"

// DefaultParams returns the default compute_pool module parameters.
func DefaultParams() *Params {
	return &Params{
		ComputePoolShareBps:      100000,     // 10%
		BaseCuPerVerification:    100,
		MinProviderStake:         "10000000", // 10 ZRN
		MinUptimeBps:             900000,     // 90%
		HeartbeatIntervalBlocks:  100,
		MaxPricePerCu:           "1000000", // 1 ZRN
		ProviderUnbondingBlocks:  10000,
		PriceChangeDelayBlocks:   500,
		MaxLatencyMs:             5000,
		SlaWindowBlocks:          1000,
		TargetUtilizationLowBps:  300000, // 30%
		TargetUtilizationHighBps: 800000, // 80%
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Providers: nil,
		Credits:   nil,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

// Validate validates the compute_pool module parameters.
func (p *Params) Validate() error {
	if p.HeartbeatIntervalBlocks == 0 {
		return fmt.Errorf("heartbeat_interval_blocks must be positive")
	}
	if p.TargetUtilizationLowBps >= p.TargetUtilizationHighBps {
		return fmt.Errorf("target_utilization_low must be less than high")
	}
	return nil
}
