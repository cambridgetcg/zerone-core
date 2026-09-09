package types

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns the default module parameters.
func DefaultParams() *Params {
	return &Params{
		UnbondingPeriod:            268_560,    // ~7 days at 2521ms blocks
		VirtualStake:               "11000000", // 11 ZRN (uzrn)
		MaxValidators:              100,
		MinSelfDelegation:          "111000", // 0.111 ZRN (uzrn)
		MaxSlashesPerEpoch:         2,
		SlashDecayPeriodBlocks:     34_272, // ~1 day
		MaxSlashCountDeactivate:    3,
		MinStakeForVerification:    "111000", // 0.111 ZRN
		SlashEscalationBps:         100_000,  // 10%
		ReputationCorrectDelta:     100,      // +0.01%
		ReputationIncorrectDelta:   200,      // -0.02%
		ReputationSlashDelta:       10_000,   // -1%
		RedelegationCooldownBlocks: 1_111,    // ~46 minutes
		TierConfigs:                DefaultTierConfigs(),
	}
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                  DefaultParams(),
		Validators:              nil,
		Delegations:             nil,
		UnbondingEntries:        nil,
		UnbondingSeq:            0,
		AccountingSafetyEnabled: true,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if gs.Params.UnbondingPeriod == 0 {
		return fmt.Errorf("unbonding period must be positive")
	}
	for _, tc := range gs.Params.TierConfigs {
		if tc == nil {
			return fmt.Errorf("nil tier configuration")
		}
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate tier configs
	if len(gs.Params.TierConfigs) != 4 {
		return fmt.Errorf("params must have exactly 4 tier configs, got %d", len(gs.Params.TierConfigs))
	}

	// Check for duplicate operator addresses
	seen := make(map[string]bool)
	for _, v := range gs.Validators {
		if v == nil {
			return fmt.Errorf("nil validator")
		}
		if seen[v.OperatorAddress] {
			return fmt.Errorf("duplicate validator: %s", v.OperatorAddress)
		}
		seen[v.OperatorAddress] = true
	}
	// Catch duplicate source records before InitGenesis could overwrite them.
	// Exact custody, primary/index equality, and bank backing are independently
	// checked after all module genesis state is available.
	if gs.AccountingSafetyEnabled {
		pairs, ids, dids := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, v := range gs.Validators {
			if v.Did != "" {
				if dids[v.Did] {
					return fmt.Errorf("duplicate validator DID")
				}
				dids[v.Did] = true
			}
		}
		for _, d := range gs.Delegations {
			if d == nil {
				return fmt.Errorf("nil delegation")
			}
			key := d.DelegatorAddress + "\x00" + d.ValidatorAddress
			if pairs[key] {
				return fmt.Errorf("duplicate delegation claim")
			}
			pairs[key] = true
		}
		for _, u := range gs.UnbondingEntries {
			if u == nil || ids[u.Id] {
				return fmt.Errorf("nil or duplicate unbonding claim")
			}
			ids[u.Id] = true
		}
	}
	cooldowns := map[string]bool{}
	for _, entry := range gs.RedelegationCooldowns {
		if entry == nil || cooldowns[entry.DelegatorAddress] {
			return fmt.Errorf("nil or duplicate redelegation cooldown")
		}
		addr, err := sdk.AccAddressFromBech32(entry.DelegatorAddress)
		if err != nil || addr.String() != entry.DelegatorAddress || entry.Height == 0 || entry.Height > math.MaxInt64 {
			return fmt.Errorf("invalid redelegation cooldown")
		}
		cooldowns[entry.DelegatorAddress] = true
	}

	return nil
}
