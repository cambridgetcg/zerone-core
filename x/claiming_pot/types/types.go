package types

import (
	"fmt"
	"math/big"
	"strings"

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

	// MaxBootstrapAddressesPerMsg caps the repeated addresses field of
	// MsgAddBootstrapEntry (enforced in ValidateBasic and re-checked in the
	// msg server). An unbounded repeated field would let a single tx do
	// unbounded state writes.
	MaxBootstrapAddressesPerMsg = 500

	// BootstrapAdmissionWindowBlocks is the length of one registrar
	// admission window (~1 day at 2.52s blocks). Windows are consecutive
	// fixed (tumbling) spans indexed by height/BootstrapAdmissionWindowBlocks:
	// at most Params.BootstrapDailyAdmissionCap registrar admissions land in
	// each fixed window, so any rolling span of this length sees at most
	// 2x the cap (two adjacent windows) — the consensus compromise bound
	// the design relies on holds per window and within 2x across a boundary.
	BootstrapAdmissionWindowBlocks = 34272

	// DefaultBootstrapEmissionCapUzrn is 222,222 ZRN = 0.1% of the
	// 222,222,222 ZRN max supply: the lifetime bootstrap issuance budget.
	DefaultBootstrapEmissionCapUzrn = "222222000000"

	// DefaultBootstrapDailyAdmissionCap bounds registrar admissions per
	// window: 5,000 admissions x 0.222 ZRN = 1,110 ZRN/day worst case.
	DefaultBootstrapDailyAdmissionCap = 5000
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
		MaxPotsActive:              10,
		MinClaimAmount:             "1000",
		BootstrapRegistrar:         "",
		BootstrapEmissionCapUzrn:   DefaultBootstrapEmissionCapUzrn,
		BootstrapDailyAdmissionCap: DefaultBootstrapDailyAdmissionCap,
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
	if p.BootstrapEmissionCapUzrn != "" {
		emissionCap := new(big.Int)
		if _, ok := emissionCap.SetString(p.BootstrapEmissionCapUzrn, 10); !ok || emissionCap.Sign() <= 0 {
			return fmt.Errorf("bootstrap_emission_cap_uzrn must be a positive integer: %s", p.BootstrapEmissionCapUzrn)
		}
	}
	if p.BootstrapRegistrar != "" {
		if _, err := sdk.AccAddressFromBech32(p.BootstrapRegistrar); err != nil {
			return fmt.Errorf("bootstrap_registrar must be a valid bech32 address: %w", err)
		}
		// A registrar with a zero daily cap is a misconfiguration, not a
		// pause switch — pausing is done by setting the registrar to "".
		if p.BootstrapDailyAdmissionCap == 0 {
			return fmt.Errorf("bootstrap_daily_admission_cap must be positive when bootstrap_registrar is set")
		}
	}
	return nil
}

// BootstrapEmissionCap returns the lifetime bootstrap emission cap in uzrn.
// Params stored before the field existed unmarshal to "" — those fall back
// to the default cap, so the aggregate bound is enforced even on state
// written by older binaries (fail-closed, never fail-open to unlimited).
func (p *Params) BootstrapEmissionCap() *big.Int {
	emissionCap := new(big.Int)
	if _, ok := emissionCap.SetString(p.BootstrapEmissionCapUzrn, 10); !ok || emissionCap.Sign() <= 0 {
		emissionCap.SetString(DefaultBootstrapEmissionCapUzrn, 10)
	}
	return emissionCap
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
	bootstrapEntries := int64(0)
	for _, pot := range gs.Pots {
		if seen[pot.Id] {
			return fmt.Errorf("duplicate pot id: %s", pot.Id)
		}
		seen[pot.Id] = true
		if strings.HasPrefix(pot.Id, BootstrapPotIDPrefix) {
			bootstrapEntries++
		}
	}
	// Genesis bootstrap pots consume the same lifetime emission budget as
	// post-genesis admissions — catch a ceremony that over-injects.
	params := gs.Params
	if params == nil {
		params = DefaultParams()
	}
	perEntry, _ := new(big.Int).SetString(PerAgentBootstrapUzrn, 10)
	committed := new(big.Int).Mul(big.NewInt(bootstrapEntries), perEntry)
	if committed.Cmp(params.BootstrapEmissionCap()) > 0 {
		return fmt.Errorf("genesis bootstrap pots commit %s uzrn > bootstrap_emission_cap_uzrn %s", committed, params.BootstrapEmissionCap())
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
	if len(msg.Addresses) > MaxBootstrapAddressesPerMsg {
		return fmt.Errorf("too many addresses: %d > max %d per message — split into batches", len(msg.Addresses), MaxBootstrapAddressesPerMsg)
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
