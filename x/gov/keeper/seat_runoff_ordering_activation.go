package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// SeatRunoffOrderingEnabled reports whether the seat-runoff ordering repair of
// zerone-core #117 is selected: due seats are tallied in ascending seat index,
// so the shared runoff ID counter is allocated in one canonical order. The
// marker at types.SeatRunoffOrderingEnabledKey is the only selector. Native
// genesis writes it and the named carrier upgrade consensus-repairs-v1 writes
// it at its height. No message or parameter toggles it, and nothing clears it.
//
// activation.Read holds the contract: the read charges no gas, and an
// unreadable or malformed marker is an error wrapping activation.ErrMarkerFault,
// never legacy state. Callers propagate it before choosing.
func (k Keeper) SeatRunoffOrderingEnabled(ctx context.Context) (bool, error) {
	return activation.Read(ctx, activation.StoreKeyService(k.storeKey), types.SeatRunoffOrderingEnabledKey, "seat-runoff ordering")
}

// EnableSeatRunoffOrdering selects the repair. It is idempotent and rewrites
// no seat election, vote or counter. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableSeatRunoffOrdering(ctx context.Context) error {
	return activation.Enable(ctx, activation.StoreKeyService(k.storeKey), types.SeatRunoffOrderingEnabledKey, "seat-runoff ordering")
}
