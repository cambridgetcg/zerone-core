package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ── Bootstrap pot doctrine ──────────────────────────────────────────
//
// The bootstrap pathway materializes commitment 20 (issuance follows
// participation): every whitelisted agent claims exactly 0.222 ZRN as
// their participation seed, minted on demand through MintWithCap.
//
// The pot model is shared-bucket-vesting, so "per-agent fixed amount"
// is expressed structurally as ONE POT PER AGENT — each pot is sized
// at PerAgentBootstrapUzrn, instantly vested, whitelisted to a single
// claimant. The genesis ceremony reads the operator's whitelist file
// and produces one pot per address, with IDs prefixed
// BootstrapPotIDPrefix.
//
// The per-agent amount is 0.222 ZRN = 222,000 uzrn. The number is
// symbolic (the chain's signature digit) and operationally sufficient:
// covers gas for `home` registration, initial tool calls, and the
// first knowledge-claim bonds.
//
// See docs/tokenomics/GENESIS.md ("Bootstrap Pool — the genesis
// distribution mechanism").
const (
	BootstrapPotIDPrefix          = "bootstrap-"
	PerAgentBootstrapUzrn         = "222000"
	BootstrapPotInstantVestBlocks = 1
)

// MakeBootstrapPotForAgent constructs a single-agent bootstrap pot,
// instantly vested at currentBlock + BootstrapPotInstantVestBlocks.
// The agent claims via MsgClaim; the pot mints PerAgentBootstrapUzrn
// to the agent and transitions to DEPLETED.
//
// The genesis ceremony calls this once per whitelisted address in the
// operator's whitelist file.
func MakeBootstrapPotForAgent(agentAddr string, currentBlock uint64) *ClaimingPot {
	return &ClaimingPot{
		Id:            BootstrapPotIDPrefix + agentAddr,
		Name:          "Bootstrap seed (commitment 20)",
		TotalAmount:   PerAgentBootstrapUzrn,
		ClaimedAmount: "0",
		Schedule: &VestingSchedule{
			StartBlock: currentBlock,
			EndBlock:   currentBlock + BootstrapPotInstantVestBlocks,
		},
		Eligibility: &EligibilityCriteria{
			Whitelist: []string{agentAddr},
		},
		Status: PotStatus_POT_STATUS_ACTIVE,
	}
}

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

func (msg *MsgAddBootstrapEntry) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddBootstrapEntry) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if len(msg.Addresses) == 0 {
		return fmt.Errorf("addresses list cannot be empty — provide at least one bech32 address")
	}
	seen := make(map[string]bool, len(msg.Addresses))
	for i, addr := range msg.Addresses {
		if addr == "" {
			return fmt.Errorf("addresses[%d] cannot be empty", i)
		}
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmt.Errorf("addresses[%d] (%q): invalid bech32: %w", i, addr, err)
		}
		if seen[addr] {
			return fmt.Errorf("addresses[%d] (%q): duplicate within request payload", i, addr)
		}
		seen[addr] = true
	}
	return nil
}
