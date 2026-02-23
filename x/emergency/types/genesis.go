package types

import "fmt"

// DefaultParams returns the default emergency module parameters.
func DefaultParams() Params {
	return Params{
		HaltQuorum:                      750000,  // 75%
		RevertQuorum:                    800000,  // 80%
		ResumeQuorum:                    800000,  // 80%
		HaltPrevoteBlocks:              11,
		HaltPrecommitBlocks:            11,
		HaltTimeoutBlocks:              44,
		RevertPrevoteBlocks:            22,
		RevertPrecommitBlocks:          22,
		RevertTimeoutBlocks:            111,
		ResumePrevoteBlocks:            22,
		ResumePrecommitBlocks:          22,
		ResumeTimeoutBlocks:            111,
		MaxProposalsPerEpoch:           3,
		MaxProposalsPerGuardianPerEpoch: 1,
		CooldownBlocks:                 111,
		MinGuardianStake:               "111111000000", // 111,111 ZRN in uzrn
		MinDistinctVoters:              4,
		MaxRevertDepth:                 111111,
		EpochBlocks:                    34272, // ~1 day at 2521ms blocks
		GenesisCouncil:                 []string{},
		CouncilExpiryBlock:             0,
		CouncilVirtualStake:            "11111000000", // 11,111 ZRN in uzrn
		MaxHaltDurationBlocks:          34272,         // ~1 day auto-resume
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params: &p,
		Status: string(StatusNormal),
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

// Validate validates the params.
func (p *Params) Validate() error {
	if p.HaltQuorum > 1000000 {
		return fmt.Errorf("halt_quorum must be <= 1000000, got %d", p.HaltQuorum)
	}
	if p.RevertQuorum > 1000000 {
		return fmt.Errorf("revert_quorum must be <= 1000000, got %d", p.RevertQuorum)
	}
	if p.ResumeQuorum > 1000000 {
		return fmt.Errorf("resume_quorum must be <= 1000000, got %d", p.ResumeQuorum)
	}
	if p.HaltPrevoteBlocks == 0 {
		return fmt.Errorf("halt_prevote_blocks must be > 0")
	}
	if p.HaltPrecommitBlocks == 0 {
		return fmt.Errorf("halt_precommit_blocks must be > 0")
	}
	if p.HaltTimeoutBlocks == 0 {
		return fmt.Errorf("halt_timeout_blocks must be > 0")
	}
	if p.EpochBlocks == 0 {
		return fmt.Errorf("epoch_blocks must be > 0")
	}
	if p.MaxHaltDurationBlocks == 0 {
		return fmt.Errorf("max_halt_duration_blocks must be > 0")
	}
	return nil
}
