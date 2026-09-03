package types

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxActiveBountiesPerSponsorHardCap bounds sponsor-index work on every
// CreateBountyOrder, even if governance raises the configurable limit.
const MaxActiveBountiesPerSponsorHardCap uint32 = 256

// ---- Params ----

func DefaultParams() *Params {
	return &Params{
		MinTargetCount:              1,
		MinDurationBlocks:           100,
		MaxActiveBountiesPerSponsor: 16,
	}
}

func (p *Params) Validate() error {
	if p.MinTargetCount == 0 {
		return fmt.Errorf("min_target_count must be positive")
	}
	if p.MinDurationBlocks == 0 {
		return fmt.Errorf("min_duration_blocks must be positive")
	}
	if p.MaxActiveBountiesPerSponsor == 0 {
		return fmt.Errorf("max_active_bounties_per_sponsor must be positive")
	}
	if p.MaxActiveBountiesPerSponsor > MaxActiveBountiesPerSponsorHardCap {
		return fmt.Errorf("max_active_bounties_per_sponsor must be <= %d", MaxActiveBountiesPerSponsorHardCap)
	}
	return nil
}

// ---- Genesis ----

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Orders:       []*BountyOrder{},
		Fulfillments: []*BountyFulfillment{},
		NextBountyId: 1,
	}
}

func (gs *GenesisState) Validate() error {
	if gs == nil {
		return fmt.Errorf("genesis state cannot be nil")
	}
	if err := gs.normalizeLegacyRecords(); err != nil {
		return err
	}
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	seenOrders := make(map[string]*BountyOrder, len(gs.Orders))
	activeBySponsor := make(map[string]uint32)
	var maxBountyID uint64
	for _, o := range gs.Orders {
		if o == nil {
			return fmt.Errorf("nil bounty order")
		}
		if _, exists := seenOrders[o.Id]; exists {
			return fmt.Errorf("duplicate bounty order id: %s", o.Id)
		}
		n, err := parseBountyID(o.Id)
		if err != nil {
			return err
		}
		if n > maxBountyID {
			maxBountyID = n
		}
		canonicalSponsor, err := CanonicalAccountAddress(o.Sponsor)
		if err != nil {
			return fmt.Errorf("bounty %s has invalid sponsor: %w", o.Id, err)
		}
		if o.Sponsor != canonicalSponsor {
			return fmt.Errorf("bounty %s sponsor must use canonical lowercase bech32 encoding", o.Id)
		}
		if o.Domain == "" {
			return fmt.Errorf("bounty %s has empty domain", o.Id)
		}
		if o.WorkContract != nil && o.EndBlock <= o.StartBlock {
			return fmt.Errorf("bounty %s end_block must be greater than start_block", o.Id)
		}
		if o.WorkContract != nil {
			if err := o.WorkContract.Validate(); err != nil {
				return fmt.Errorf("bounty %s invalid work_contract: %w", o.Id, err)
			}
		}
		expected, err := ExpectedEscrowRemaining(o)
		if err != nil {
			return fmt.Errorf("bounty %s: %w", o.Id, err)
		}
		stored, err := ParseNonNegativeAmount(o.EscrowRemaining)
		if err != nil || stored.Cmp(expected) != 0 {
			return fmt.Errorf("bounty %s escrow_remaining %q does not equal derived liability %s", o.Id, o.EscrowRemaining, expected)
		}
		seenOrders[o.Id] = o
		if o.Status == BountyStatus_BOUNTY_STATUS_ACTIVE {
			activeBySponsor[o.Sponsor]++
			if activeBySponsor[o.Sponsor] > gs.Params.MaxActiveBountiesPerSponsor {
				return fmt.Errorf("sponsor %s has %d active bounties, max %d", o.Sponsor, activeBySponsor[o.Sponsor], gs.Params.MaxActiveBountiesPerSponsor)
			}
		}
	}
	if gs.NextBountyId == 0 || maxBountyID == math.MaxUint64 ||
		(gs.NextBountyId != math.MaxUint64 && gs.NextBountyId <= maxBountyID) {
		return fmt.Errorf("next_bounty_id %d must be greater than every existing bounty id (max %d)", gs.NextBountyId, maxBountyID)
	}

	counts := make(map[string]uint32, len(gs.Orders))
	seenPairs := make(map[string]struct{}, len(gs.Fulfillments))
	// v1 allowed the same fact to fill multiple legacy orders. Preserve that
	// historical export/import shape, but never permit a v2-bound fulfillment
	// to share a fact with any other payout.
	seenFacts := make(map[string]bool, len(gs.Fulfillments)) // value: prior occurrence was v2
	seenReceipts := make(map[string]struct{}, len(gs.Fulfillments))
	seenNullifiers := make(map[string]struct{}, len(gs.Fulfillments))
	for _, f := range gs.Fulfillments {
		if f == nil {
			return fmt.Errorf("nil bounty fulfillment")
		}
		order, exists := seenOrders[f.BountyId]
		if !exists {
			return fmt.Errorf("fulfillment references unknown bounty %s", f.BountyId)
		}
		if f.FactId == "" {
			return fmt.Errorf("bounty %s fulfillment has empty fact_id", f.BountyId)
		}
		if _, err := sdk.AccAddressFromBech32(f.Worker); err != nil {
			return fmt.Errorf("bounty %s fact %s has invalid worker: %w", f.BountyId, f.FactId, err)
		}
		amount, err := ParsePositiveAmount(f.AmountPaid)
		if err != nil || amount.String() != order.PricePerArtifact {
			return fmt.Errorf("bounty %s fact %s amount_paid %q != price_per_artifact %q", f.BountyId, f.FactId, f.AmountPaid, order.PricePerArtifact)
		}
		legacyWrappedWindow := order.WorkContract == nil && order.EndBlock <= order.StartBlock
		if !legacyWrappedWindow && (f.FulfilledAtBlock < order.StartBlock || f.FulfilledAtBlock >= order.EndBlock) {
			return fmt.Errorf("bounty %s fact %s fulfillment block outside order window", f.BountyId, f.FactId)
		}
		pair := f.BountyId + "\x00" + f.FactId
		if _, duplicate := seenPairs[pair]; duplicate {
			return fmt.Errorf("duplicate fulfillment pair %s/%s", f.BountyId, f.FactId)
		}
		seenPairs[pair] = struct{}{}
		currentV2 := order.WorkContract != nil
		if priorV2, duplicate := seenFacts[f.FactId]; duplicate && (priorV2 || currentV2) {
			return fmt.Errorf("fact %s fulfilled across multiple bounties with a v2-bound order", f.FactId)
		}
		seenFacts[f.FactId] = currentV2
		if order.WorkContract == nil {
			if f.WorkReceiptHash != "" || f.SettlementNullifier != "" || f.ArtifactRoot != "" {
				return fmt.Errorf("legacy bounty %s fulfillment carries v2 settlement fields", f.BountyId)
			}
		} else {
			if f.Worker != order.WorkContract.WorkerAddress {
				return fmt.Errorf("bounty %s fact %s worker %q != assigned worker %q",
					f.BountyId, f.FactId, f.Worker, order.WorkContract.WorkerAddress)
			}
			if err := ValidateSHA256Hex("work_receipt_hash", f.WorkReceiptHash); err != nil {
				return fmt.Errorf("bounty %s fact %s: %w", f.BountyId, f.FactId, err)
			}
			if err := ValidateSHA256Hex("artifact_root", f.ArtifactRoot); err != nil {
				return fmt.Errorf("bounty %s fact %s: %w", f.BountyId, f.FactId, err)
			}
			wantNullifier := ComputeSettlementNullifier(
				order.WorkContract.WorkSpecHash,
				order.WorkContract.AcceptanceHash,
				order.WorkContract.InputRoot,
				order.WorkContract.EnvironmentRoot,
				f.ArtifactRoot,
				order.WorkContract.WorkerAddress,
			)
			if f.SettlementNullifier != wantNullifier {
				return fmt.Errorf("bounty %s fact %s settlement_nullifier %q != derived %q", f.BountyId, f.FactId, f.SettlementNullifier, wantNullifier)
			}
			if _, duplicate := seenReceipts[f.WorkReceiptHash]; duplicate {
				return fmt.Errorf("work receipt %s fulfilled across multiple bounties", f.WorkReceiptHash)
			}
			seenReceipts[f.WorkReceiptHash] = struct{}{}
			if _, duplicate := seenNullifiers[f.SettlementNullifier]; duplicate {
				return fmt.Errorf("settlement nullifier %s is duplicated", f.SettlementNullifier)
			}
			seenNullifiers[f.SettlementNullifier] = struct{}{}
		}
		counts[f.BountyId]++
	}
	for id, order := range seenOrders {
		if counts[id] != order.FulfilledCount {
			return fmt.Errorf("bounty %s fulfilled_count %d != fulfillment records %d", id, order.FulfilledCount, counts[id])
		}
	}
	return nil
}

// normalizeLegacyRecords upgrades the wire-compatible v1 genesis shape in
// place before strict v2 validation. Only nil-WorkContract orders qualify.
// Bound records never receive compatibility normalization.
func (gs *GenesisState) normalizeLegacyRecords() error {
	legacyOrders := make(map[string]bool, len(gs.Orders))
	legacyOnly := true
	for _, order := range gs.Orders {
		if order == nil {
			continue
		}
		if order.WorkContract != nil {
			legacyOnly = false
			continue
		}
		canonicalSponsor, err := CanonicalAccountAddress(order.Sponsor)
		if err != nil {
			return fmt.Errorf("legacy bounty %s has invalid sponsor %q: %w", order.Id, order.Sponsor, err)
		}
		price, err := NormalizeLegacyPositiveAmount(order.PricePerArtifact)
		if err != nil {
			return fmt.Errorf("legacy bounty %s has invalid price_per_artifact %q: %w", order.Id, order.PricePerArtifact, err)
		}
		remaining, err := NormalizeLegacyNonNegativeAmount(order.EscrowRemaining)
		if err != nil {
			return fmt.Errorf("legacy bounty %s has invalid escrow_remaining %q: %w", order.Id, order.EscrowRemaining, err)
		}
		order.PricePerArtifact = price
		order.EscrowRemaining = remaining
		order.Sponsor = canonicalSponsor
		legacyOrders[order.Id] = true
	}
	for _, fulfillment := range gs.Fulfillments {
		if fulfillment == nil || !legacyOrders[fulfillment.BountyId] {
			continue
		}
		amount, err := NormalizeLegacyPositiveAmount(fulfillment.AmountPaid)
		if err != nil {
			return fmt.Errorf("legacy fulfillment %s/%s has invalid amount_paid %q: %w",
				fulfillment.BountyId, fulfillment.FactId, fulfillment.AmountPaid, err)
		}
		fulfillment.AmountPaid = amount
	}
	// v1 allowed this governance/genesis parameter above the v2 consensus
	// work bound. An all-legacy import can be clamped deterministically; the
	// strict validation pass below still rejects an actual sponsor ACTIVE set
	// above the cap. Any bound v2 order disables this compatibility path.
	if legacyOnly && gs.Params != nil && gs.Params.MaxActiveBountiesPerSponsor > MaxActiveBountiesPerSponsorHardCap {
		gs.Params.MaxActiveBountiesPerSponsor = MaxActiveBountiesPerSponsorHardCap
	}
	return nil
}

func parseBountyID(id string) (uint64, error) {
	raw, ok := strings.CutPrefix(id, "bounty-")
	if !ok || raw == "" {
		return 0, fmt.Errorf("invalid bounty id %q", id)
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 || fmt.Sprintf("bounty-%d", n) != id {
		return 0, fmt.Errorf("invalid bounty id %q", id)
	}
	return n, nil
}

// ---- MsgCreateBountyOrder ----

func (msg *MsgCreateBountyOrder) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sponsor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCreateBountyOrder) ValidateBasic() error {
	if err := ValidateCanonicalAccountAddress("sponsor", msg.Sponsor); err != nil {
		return fmt.Errorf("invalid sponsor address: %w", err)
	}
	if msg.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if _, err := ParsePositiveAmount(msg.PricePerArtifact); err != nil {
		return fmt.Errorf("price_per_artifact must be a positive integer in uzrn")
	}
	if msg.TargetCount == 0 {
		return fmt.Errorf("target_count must be positive")
	}
	if msg.DurationBlocks == 0 {
		return fmt.Errorf("duration_blocks must be positive")
	}
	if err := msg.WorkContract.Validate(); err != nil {
		return fmt.Errorf("invalid work_contract: %w", err)
	}
	return nil
}

// ---- MsgFulfillBounty ----

func (msg *MsgFulfillBounty) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Caller)
	return []sdk.AccAddress{addr}
}

func (msg *MsgFulfillBounty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if msg.BountyId == "" {
		return fmt.Errorf("bounty_id cannot be empty")
	}
	if msg.FactId == "" {
		return fmt.Errorf("fact_id cannot be empty")
	}
	return nil
}

// ---- MsgCancelBountyOrder ----

func (msg *MsgCancelBountyOrder) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sponsor)
	return []sdk.AccAddress{addr}
}

func (msg *MsgCancelBountyOrder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sponsor); err != nil {
		return fmt.Errorf("invalid sponsor address: %w", err)
	}
	if msg.BountyId == "" {
		return fmt.Errorf("bounty_id cannot be empty")
	}
	return nil
}
