package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SeatDepositEnabled reports whether the reviewer seat deposit is selected: a
// review round created under review policy 1 and commitment scheme 2 records
// seat terms, and each commitment in such a round locks the round's deposit
// until every seat's deposit is returned in full (the reviewer-entry design
// draft, pull request #190, step B). The marker at types.SeatDepositEnabledKey
// is the only selector. Genesis writes it when the file selects it; native
// genesis does not select it yet. No carrier upgrade writes it yet. No message
// or param toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) SeatDepositEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, k.storeService, types.SeatDepositEnabledKey, "seat-deposit")
}

// EnableSeatDeposit selects the seat deposit. It is idempotent: with the
// marker already on it checks nothing and writes nothing. Otherwise it first
// runs ValidateSeatDepositActivation, so a carrier that loops over the v3
// registry cannot turn the deposit on without its predecessors or with an
// address the deposits cannot use. It rewrites no param, round, commitment,
// account or balance. A round that already exists keeps the terms it was
// created with, none, for its whole life.
func (k Keeper) EnableSeatDeposit(ctx context.Context) error {
	enabled, err := k.SeatDepositEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.ValidateSeatDepositActivation(ctx); err != nil {
		return err
	}
	return activation.Enable(ctx, k.storeService, types.SeatDepositEnabledKey, "seat-deposit")
}

// ValidateSeatDepositActivation is what turning the seat deposit on requires
// (the reviewer-entry design, section 7, and D6), read from the store:
//
//   - its predecessor markers are on: the reviewer-balance parameter, review
//     neutrality and record integrity. Genesis validation applies the same
//     rule to a file (types.ValidateSeatDepositGenesis); this applies it to a
//     chain, for a carrier and for import, which writes those markers first;
//   - the seat-deposit address holds nothing, its own module account, or a
//     plain or vesting account that no key has signed for, which the first
//     deposit converts (seatDepositSquatter). Without an account keeper this
//     part is skipped, as at import;
//   - the seat-deposit account holds at least the locked, unreturned deposits
//     every stored round records (ValidateSeatDepositCustody). Without a bank
//     keeper nothing can be locked (submitDepositedCommitment and import both
//     refuse), so this part is skipped.
//
// It writes nothing.
func (k Keeper) ValidateSeatDepositActivation(ctx context.Context) error {
	for _, predecessor := range []struct {
		name string
		read func(context.Context) (bool, error)
	}{
		{"reviewer_balance_param_enabled", k.ReviewerBalanceParamEnabled},
		{"review_neutrality_enabled", k.ReviewNeutralityEnabled},
		{"record_integrity_enabled", k.RecordIntegrityEnabled},
	} {
		on, err := predecessor.read(ctx)
		if err != nil {
			return err
		}
		if !on {
			return fmt.Errorf("seat_deposit_enabled requires %s", predecessor.name)
		}
	}
	if k.accountKeeper != nil {
		if _, err := k.seatDepositSquatter(ctx); err != nil {
			return err
		}
	}
	if k.bankKeeper == nil {
		return nil
	}
	return k.ValidateSeatDepositCustody(ctx)
}
