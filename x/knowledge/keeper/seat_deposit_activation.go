package keeper

import (
	"context"

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

// EnableSeatDeposit selects the seat deposit. It is idempotent and rewrites no
// param, round, commitment or balance. A round that already exists keeps the
// terms it was created with, none, for its whole life. The predecessor markers
// (reviewer-balance parameter, review neutrality, record integrity) are
// required by genesis validation and when a round would record terms, not
// here, so this stays a plain activation.Enable like every other v3 marker.
func (k Keeper) EnableSeatDeposit(ctx context.Context) error {
	return activation.Enable(ctx, k.storeService, types.SeatDepositEnabledKey, "seat-deposit")
}
