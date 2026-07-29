package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Adapters: nil,
	}
}

func (gs *GenesisState) Validate() error {
	if gs == nil {
		return fmt.Errorf("genesis state must not be nil")
	}
	if gs.Params == nil {
		return fmt.Errorf("params must not be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, a := range gs.Adapters {
		if seen[a.AdapterId] {
			return fmt.Errorf("duplicate adapter_id in genesis: %s", a.AdapterId)
		}
		seen[a.AdapterId] = true
	}
	return nil
}
