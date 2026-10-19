package keeper

import (
	"bytes"
	"context"
	"fmt"
)

// SingleRefundEnabledStoreKey selects prospective single-refund expiry and
// withdrawal semantics in the qualification store. Absence preserves the legacy
// rules. Only a separately reviewed upgrade carrier may enable it; no message,
// parameter, constructor or default genesis does so, and there is no disable seam.
// The current logical genesis schema does not carry this marker: an authorized
// export/import carrier must restore it explicitly before processing blocks or
// withdrawals. Enabling it does not classify or rewrite historical liabilities.
const SingleRefundEnabledStoreKey = "\x40"

// SingleRefundEnabled never interprets unreadable or malformed marker bytes as
// legacy state. Callers must propagate errors before any expiry/withdraw effects.
func (k Keeper) SingleRefundEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(SingleRefundEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read qualification single-refund marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed qualification single-refund marker")
	}
	return true, nil
}

// EnableSingleRefund is the idempotent keeper-only setter. The upgrade owner
// must validate the predecessor, census legacy claims and authorize their
// resolution separately; this setter changes only the execution selector.
func (k Keeper) EnableSingleRefund(ctx context.Context) error {
	enabled, err := k.SingleRefundEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(SingleRefundEnabledStoreKey), []byte{1}); err != nil {
		return fmt.Errorf("write qualification single-refund marker: %w", err)
	}
	return nil
}
