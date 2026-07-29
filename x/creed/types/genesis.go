package types

// DefaultParams preserves the historical direct-authority anchor posture.
// Production deployments must not infer governance-only amendment from this
// default: a release must first configure a working Creed Amendment category
// and then explicitly set DirectAnchorEnabled false.
func DefaultParams() *Params {
	return &Params{
		Authority:           "",   // compatibility-only; keeper constructor owns runtime authority
		DirectAnchorEnabled: true, // explicit historical default; not governance-only
	}
}

// DefaultGenesis returns a non-pinned starting state. Deployments may
// explicitly supply GenesisPin, but no runtime or repository-CI check proves
// that a running network did so; consumers must query network state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		GenesisPin: nil,
		History:    nil,
	}
}

// Validate checks that the genesis state is internally consistent
// before InitGenesis runs.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return ErrInvalidParams.Wrap("params must not be nil")
	}
	if gs.GenesisPin != nil {
		if err := validatePin(gs.GenesisPin); err != nil {
			return err
		}
	}
	prevVersion := uint32(0)
	for i, p := range gs.History {
		if p == nil {
			return ErrInvalidParams.Wrapf("history[%d] is nil", i)
		}
		if err := validatePin(p); err != nil {
			return err
		}
		if p.Version <= prevVersion {
			return ErrVersionNotMonotonic.Wrapf("history[%d] version %d not strictly greater than previous %d", i, p.Version, prevVersion)
		}
		prevVersion = p.Version
	}
	if gs.GenesisPin != nil && gs.GenesisPin.Version <= prevVersion {
		return ErrVersionNotMonotonic.Wrapf("genesis_pin version %d must be greater than all history entries (last %d)", gs.GenesisPin.Version, prevVersion)
	}
	return nil
}

// validatePin enforces the structural invariants commitment 10
// names: monotonic version, non-empty hash, unique and contiguous
// commitment numbers (modulo archived entries).
func validatePin(p *PinnedCreed) error {
	if p.Version == 0 {
		return ErrVersionNotMonotonic.Wrap("version must be ≥ 1")
	}
	if err := ValidateCanonicalHash(p.CanonicalHash); err != nil {
		return err
	}
	return ValidateCommitmentRegistryAtHeight(p.Commitments, p.PinnedAtHeight)
}
