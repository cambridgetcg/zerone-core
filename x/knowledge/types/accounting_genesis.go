package types

import (
	"fmt"
	"unicode/utf8"

	sdkmath "cosmossdk.io/math"
)

// ValidateGenesisAccounting validates the complete present inventory. Absence
// retains legacy import; neither absence nor a present inventory proves that
// historical exports preserved all obligations. Bank backing is checked by the
// keeper against the bank genesis already imported by the application.
func ValidateGenesisAccounting(gs *GenesisState) error {
	state := gs.AccountingState
	if state == nil {
		return nil
	}
	allocation, err := genesisAccountingAmount("training_fund_allocation", gs.TrainingFundAllocation)
	if err != nil {
		return err
	}
	nonEscrow, err := genesisAccountingAmount("training_fund_non_escrow_balance", state.TrainingFundNonEscrowBalance)
	if err != nil {
		return err
	}
	if nonEscrow.GT(allocation) {
		return fmt.Errorf("accounting_state non-escrow balance exceeds training allocation")
	}
	// Subtract from the bounded allocation rather than summing hostile input:
	// even a collection of individually valid 256-bit integers cannot overflow.
	unassigned := allocation.Sub(nonEscrow)
	bounties := make(map[string]bool, len(gs.AugmentationBounties))
	for _, bounty := range gs.AugmentationBounties {
		if bounty == nil || bounty.Id == "" || !utf8.ValidString(bounty.Id) {
			return fmt.Errorf("accounting_state requires nonempty UTF-8 bounty identities")
		}
		if bounties[bounty.Id] {
			return fmt.Errorf("accounting_state duplicate bounty %q", bounty.Id)
		}
		bounties[bounty.Id] = true
	}
	escrows := make(map[string]bool, len(state.AugmentationEscrows))
	for _, entry := range state.AugmentationEscrows {
		if entry == nil || !bounties[entry.BountyId] {
			return fmt.Errorf("accounting_state escrow references an absent bounty")
		}
		if escrows[entry.BountyId] {
			return fmt.Errorf("accounting_state duplicate escrow %q", entry.BountyId)
		}
		escrows[entry.BountyId] = true
		remaining, err := genesisAccountingAmount("remaining_uzrn", entry.RemainingUzrn)
		if err != nil {
			return err
		}
		if remaining.GT(unassigned) {
			return fmt.Errorf("accounting_state escrow plus non-escrow balance exceeds training allocation")
		}
		unassigned = unassigned.Sub(remaining)
	}
	if !unassigned.IsZero() {
		return fmt.Errorf("accounting_state escrow plus non-escrow balance differs from training allocation")
	}
	addresses := make(map[string]bool, len(state.BootstrapAddressCounts))
	for _, entry := range state.BootstrapAddressCounts {
		if entry == nil || entry.Address == "" || !utf8.ValidString(entry.Address) {
			return fmt.Errorf("accounting_state requires nonempty UTF-8 bootstrap address keys")
		}
		// Keys historically used the exact supplied address string. Do not
		// normalize case or decode/re-encode it into a different primary key.
		if addresses[entry.Address] {
			return fmt.Errorf("accounting_state duplicate bootstrap address %q", entry.Address)
		}
		addresses[entry.Address] = true
	}
	epochs := make(map[uint64]bool, len(state.BootstrapEpochCounts))
	for _, entry := range state.BootstrapEpochCounts {
		if entry == nil {
			return fmt.Errorf("accounting_state nil bootstrap epoch count")
		}
		if epochs[entry.Epoch] {
			return fmt.Errorf("accounting_state duplicate bootstrap epoch %d", entry.Epoch)
		}
		epochs[entry.Epoch] = true
	}
	return nil
}

func genesisAccountingAmount(field, value string) (sdkmath.Int, error) {
	amount, ok := sdkmath.NewIntFromString(value)
	if !ok || amount.IsNegative() || amount.String() != value {
		return sdkmath.Int{}, fmt.Errorf("accounting_state %s must be a canonical non-negative base-10 integer within 256 bits", field)
	}
	return amount, nil
}
