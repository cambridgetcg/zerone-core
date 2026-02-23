package types

import "fmt"

// DefaultParams returns the default schedule module parameters.
func DefaultParams() *Params {
	return &Params{
		MaxActivePerAccount: 20,
		MaxGasPerBlock:      50000000,
		MinIntervalBlocks:   10,
		MinFeePerExecution:  "10000",
		MaxCompoundDepth:    3,
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Processes: nil,
		Sequence:  0,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

// Validate validates the schedule module parameters.
func (p *Params) Validate() error {
	if p.MaxActivePerAccount == 0 {
		return fmt.Errorf("max_active_per_account must be positive")
	}
	if p.MinIntervalBlocks == 0 {
		return fmt.Errorf("min_interval_blocks must be positive")
	}
	if p.MaxCompoundDepth == 0 {
		return fmt.Errorf("max_compound_depth must be positive")
	}
	return nil
}
