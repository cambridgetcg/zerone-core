package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns the default claiming_pot parameters.
func DefaultParams() *Params {
	return &Params{
		MaxPotsActive:  10,
		MinClaimAmount: "1000",
	}
}

// Validate validates the parameters.
func (p *Params) Validate() error {
	if p.MaxPotsActive == 0 {
		return fmt.Errorf("max_pots_active must be positive")
	}
	minClaim := new(big.Int)
	if _, ok := minClaim.SetString(p.MinClaimAmount, 10); !ok || minClaim.Sign() <= 0 {
		return fmt.Errorf("min_claim_amount must be a positive integer: %s", p.MinClaimAmount)
	}
	return nil
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Pots:   []*ClaimingPot{},
		Claims: []*Claim{},
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}
	seen := make(map[string]bool)
	for _, pot := range gs.Pots {
		if seen[pot.Id] {
			return fmt.Errorf("duplicate pot id: %s", pot.Id)
		}
		seen[pot.Id] = true
	}
	return nil
}

// ---- GetSigners / ValidateBasic for Msg types ----

func (msg *MsgCreatePot) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCreatePot) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(msg.TotalAmount, 10); !ok || amount.Sign() <= 0 {
		return fmt.Errorf("total_amount must be a positive integer")
	}
	if msg.Schedule == nil {
		return fmt.Errorf("schedule cannot be nil")
	}
	if msg.Schedule.EndBlock <= msg.Schedule.StartBlock {
		return fmt.Errorf("end_block must be greater than start_block")
	}
	return nil
}

func (msg *MsgClaim) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Claimant)
	return []sdk.AccAddress{addr}
}

func (msg *MsgClaim) ValidateBasic() error {
	if msg.Claimant == "" {
		return fmt.Errorf("claimant cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Claimant); err != nil {
		return fmt.Errorf("invalid claimant address: %w", err)
	}
	if msg.PotId == "" {
		return fmt.Errorf("pot_id cannot be empty")
	}
	return nil
}

func (msg *MsgUpdatePotParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdatePotParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}
