package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// RefundIdentityEnabled reports whether the refund-identity repair of
// zerone-core #127 is selected. The marker at types.RefundIdentityEnabledKey is
// the only selector. Native genesis writes it and the named carrier upgrade
// consensus-repairs-v1 writes it at its height. No message or parameter toggles
// it, and nothing clears it.
//
// The read uses its own gas meter, so a chain without the marker spends exactly
// the gas it spent before this code existed. An unreadable or malformed marker
// is an error, never legacy state; callers propagate it before choosing.
func (k Keeper) RefundIdentityEnabled(ctx context.Context) (bool, error) {
	ungassed := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	bz, err := k.storeService.OpenKVStore(ungassed).Get(types.RefundIdentityEnabledKey)
	if err != nil {
		return false, fmt.Errorf("read refund identity marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed refund identity marker")
	}
	return true, nil
}

// EnableRefundIdentity selects the repair. It is idempotent and rewrites no
// rate limit or packet flow. Native genesis and the consensus-repairs-v1
// carrier call it; the carrier also bumps this module's ConsensusVersion.
func (k Keeper) EnableRefundIdentity(ctx context.Context) error {
	enabled, err := k.RefundIdentityEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.RefundIdentityEnabledKey, []byte{1}); err != nil {
		return fmt.Errorf("write refund identity marker: %w", err)
	}
	return nil
}
