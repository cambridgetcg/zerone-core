package types

import "fmt"

func DefaultParams() Params {
	return Params{
		MaxBytecodeSize:         65536,     // 64KB
		MaxGasPerCall:           10000000,  // 10M
		MaxGasPerBlock:          100000000, // 100M
		MaxContractsPerCreator:  100,
		MaxStateEntries:         10000,
		DeployCost:              "5000000", // 5 ZRN
		MaxScheduleGas:          1000000,   // 1M
		ScheduleHorizonBlocks:   100000,
		CurrentBvmVersion:       1,
		MaxSchedulesPerContract: 100,
	}
}

func DefaultGenesisState() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params:    &p,
		Contracts: []*DeployedContract{},
		Codes:     []*ContractCode{},
		Schedules: []*ContractSchedule{},
		State:     []*ContractStateEntry{},
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params must not be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	for i, c := range gs.Contracts {
		if c.Address == "" {
			return fmt.Errorf("contract %d: empty address", i)
		}
		if c.CodeHash == "" {
			return fmt.Errorf("contract %d: empty code hash", i)
		}
		if c.Creator == "" {
			return fmt.Errorf("contract %d: empty creator", i)
		}
	}
	for i, code := range gs.Codes {
		if code.CodeHash == "" {
			return fmt.Errorf("code %d: empty code hash", i)
		}
	}
	return nil
}

func (p *Params) Validate() error {
	if p.MaxBytecodeSize == 0 {
		return fmt.Errorf("max_bytecode_size must be positive")
	}
	if p.MaxStateEntries == 0 {
		return fmt.Errorf("max_state_entries must be positive")
	}
	if p.MaxGasPerCall == 0 {
		return fmt.Errorf("max_gas_per_call must be positive")
	}
	return nil
}
