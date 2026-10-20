package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// SeatRunoffOrderingEnabled reports whether the seat-runoff ordering repair of
// zerone-core #117 is selected: due seats are tallied in ascending seat index,
// so the shared runoff ID counter is allocated in one canonical order. The
// marker at types.SeatRunoffOrderingEnabledKey is the only selector. Native
// genesis writes it and the named carrier upgrade consensus-repairs-v1 writes
// it at its height. No message or parameter toggles it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) SeatRunoffOrderingEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz := ungassed.KVStore(k.storeKey).Get(types.SeatRunoffOrderingEnabledKey)
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed seat-runoff ordering marker")
	}
	return true, nil
}

// EnableSeatRunoffOrdering selects the repair. It is idempotent and rewrites
// no seat election, vote or counter. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableSeatRunoffOrdering(ctx context.Context) error {
	enabled, err := k.SeatRunoffOrderingEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	sdk.UnwrapSDKContext(ctx).KVStore(k.storeKey).Set(types.SeatRunoffOrderingEnabledKey, []byte{1})
	return nil
}
