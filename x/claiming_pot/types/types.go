package types

import (
	"fmt"
	"math/big"
	"strconv"
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

// PotCommitmentUnits returns ceil(total_amount / PerAgentBootstrapUzrn).
// Bootstrap and legacy general pots share this fixed-size lifetime budget.
func PotCommitmentUnits(totalAmount string) (uint64, error) {
	total := new(big.Int)
	if _, ok := total.SetString(totalAmount, 10); !ok || total.Sign() <= 0 {
		return 0, fmt.Errorf("total_amount must be a positive integer: %q", totalAmount)
	}
	perUnit, _ := new(big.Int).SetString(PerAgentBootstrapUzrn, 10)
	units := new(big.Int).Add(total, new(big.Int).Sub(perUnit, big.NewInt(1)))
	units.Quo(units, perUnit)
	if !units.IsUint64() {
		return 0, fmt.Errorf("total_amount commits more than uint64 units: %q", totalAmount)
	}
	return units.Uint64(), nil
}

// GeneralPotSequence parses the canonical legacy general-pot ID `pot-N`.
func GeneralPotSequence(id string) (uint64, bool) {
	if !strings.HasPrefix(id, "pot-") {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(id, "pot-"), 10, 64)
	return n, err == nil && n > 0
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params != nil {
		if err := gs.Params.Validate(); err != nil {
			return fmt.Errorf("invalid params: %w", err)
		}
	}
	potByID := make(map[string]*ClaimingPot)
	derivedUnits := new(big.Int)
	var maxPotSequence uint64
	for i, pot := range gs.Pots {
		if pot == nil {
			return fmt.Errorf("nil pot at index %d", i)
		}
		if pot.Id == "" {
			return fmt.Errorf("pot at index %d has empty id", i)
		}
		if _, exists := potByID[pot.Id]; exists {
			return fmt.Errorf("duplicate pot id: %s", pot.Id)
		}
		potByID[pot.Id] = pot
		if sequence, ok := GeneralPotSequence(pot.Id); ok && sequence > maxPotSequence {
			maxPotSequence = sequence
		}
		units, err := PotCommitmentUnits(pot.TotalAmount)
		if err != nil {
			return fmt.Errorf("pot %s: %w", pot.Id, err)
		}
		derivedUnits.Add(derivedUnits, new(big.Int).SetUint64(units))
		claimed := new(big.Int)
		if _, ok := claimed.SetString(pot.ClaimedAmount, 10); !ok || claimed.Sign() < 0 {
			return fmt.Errorf("pot %s claimed_amount must be a non-negative integer: %q", pot.Id, pot.ClaimedAmount)
		}
		total, _ := new(big.Int).SetString(pot.TotalAmount, 10)
		if claimed.Cmp(total) > 0 {
			return fmt.Errorf("pot %s claimed_amount %s exceeds total_amount %s", pot.Id, claimed, total)
		}
		if pot.Schedule == nil {
			return fmt.Errorf("pot %s schedule must not be nil", pot.Id)
		}
		if pot.Schedule.EndBlock <= pot.Schedule.StartBlock {
			return fmt.Errorf("pot %s end_block must be greater than start_block", pot.Id)
		}
	}
	// Every genesis pot consumes the same fixed-size lifetime budget as
	// post-genesis admissions and general-pot creation.
	params := gs.Params
	if params == nil {
		params = DefaultParams()
	}
	perEntry, _ := new(big.Int).SetString(PerAgentBootstrapUzrn, 10)
	committedUnits := new(big.Int).Set(derivedUnits)
	if gs.LifetimeCommittedUnits != 0 {
		exportedUnits := new(big.Int).SetUint64(gs.LifetimeCommittedUnits)
		if exportedUnits.Cmp(derivedUnits) < 0 {
			return fmt.Errorf("lifetime_committed_units %s is below pot-derived minimum %s", exportedUnits, derivedUnits)
		}
		committedUnits = exportedUnits
	}
	committed := new(big.Int).Mul(committedUnits, perEntry)
	if committed.Cmp(params.BootstrapEmissionCap()) > 0 {
		return fmt.Errorf("genesis pots commit %s uzrn > bootstrap_emission_cap_uzrn %s", committed, params.BootstrapEmissionCap())
	}
	if gs.PotCounter != 0 && gs.PotCounter < maxPotSequence {
		return fmt.Errorf("pot_counter %d is below highest general pot sequence %d", gs.PotCounter, maxPotSequence)
	}
	if gs.PotCounter == ^uint64(0) || maxPotSequence == ^uint64(0) {
		return fmt.Errorf("pot_counter must leave room for the next general pot id")
	}
	seenClaims := make(map[string]bool)
	claimedByPot := make(map[string]*big.Int)
	for i, claim := range gs.Claims {
		if claim == nil {
			return fmt.Errorf("nil claim at index %d", i)
		}
		pot, exists := potByID[claim.PotId]
		if !exists {
			return fmt.Errorf("claim at index %d references unknown pot %q", i, claim.PotId)
		}
		if _, err := sdk.AccAddressFromBech32(claim.Claimant); err != nil {
			return fmt.Errorf("claim at index %d has invalid claimant: %w", i, err)
		}
		key := claim.PotId + "\x00" + claim.Claimant
		if seenClaims[key] {
			return fmt.Errorf("duplicate claim for pot %s and claimant %s", claim.PotId, claim.Claimant)
		}
		seenClaims[key] = true
		amount := new(big.Int)
		if _, ok := amount.SetString(claim.Amount, 10); !ok || amount.Sign() <= 0 {
			return fmt.Errorf("claim at index %d amount must be a positive integer: %q", i, claim.Amount)
		}
		if claimedByPot[pot.Id] == nil {
			claimedByPot[pot.Id] = new(big.Int)
		}
		claimedByPot[pot.Id].Add(claimedByPot[pot.Id], amount)
	}
	for _, pot := range gs.Pots {
		recorded := claimedByPot[pot.Id]
		if recorded == nil {
			recorded = new(big.Int)
		}
		claimed, _ := new(big.Int).SetString(pot.ClaimedAmount, 10)
		if recorded.Cmp(claimed) != 0 {
			return fmt.Errorf("pot %s claimed_amount %s does not match exported claims %s", pot.Id, claimed, recorded)
		}
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
