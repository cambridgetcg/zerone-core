package types

import "fmt"

// DefaultParams returns the default parameters.
func DefaultParams() *Params {
	return &Params{
		// 85% — a 5-1 vote (83.3%) is contested; a 6-1 (85.7%) is
		// strong-but-not-unanimous.
		ContestedThresholdBps: 850_000,

		// 55% — bare-majority territory. Below this the verdict is
		// contested-but-resolved with very narrow agreement.
		BareMajorityThresholdBps: 550_000,

		// Bound DomainDialectic walking cost.
		MaxFactsPerDomainQuery: 1000,
	}
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{Params: DefaultParams()}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params required")
	}
	return gs.Params.Validate()
}

func (p *Params) Validate() error {
	if p.ContestedThresholdBps == 0 || p.ContestedThresholdBps > 1_000_000 {
		return fmt.Errorf("contested_threshold_bps must be in (0, 1M]")
	}
	if p.BareMajorityThresholdBps == 0 || p.BareMajorityThresholdBps >= p.ContestedThresholdBps {
		return fmt.Errorf("bare_majority_threshold_bps must be in (0, contested_threshold_bps)")
	}
	if p.MaxFactsPerDomainQuery == 0 {
		return fmt.Errorf("max_facts_per_domain_query must be > 0")
	}
	return nil
}
