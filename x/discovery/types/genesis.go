package types

import "fmt"

// DefaultParams returns the default discovery module parameters.
func DefaultParams() *Params {
	return &Params{
		MinRegistrationStake:    "1000000", // 1 ZRN
		MaxCapabilitiesPerAgent: 20,
		ProfileExpiryBlocks:     100000,
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Profiles: nil,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

// Validate validates the discovery module parameters.
func (p *Params) Validate() error {
	if p.MaxCapabilitiesPerAgent == 0 {
		return fmt.Errorf("max_capabilities_per_agent must be positive")
	}
	if p.ProfileExpiryBlocks == 0 {
		return fmt.Errorf("profile_expiry_blocks must be positive")
	}
	return nil
}
