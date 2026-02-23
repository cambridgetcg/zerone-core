package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Challenges:  []*CaptureChallenge{},
		BountyPools: []*DomainBountyPool{},
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}
