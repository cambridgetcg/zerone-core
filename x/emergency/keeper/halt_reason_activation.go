package keeper

import (
	"bytes"
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// HaltReasonFilterEnabled selects reason-specific, start-height-scoped halt
// counts. Absence preserves legacy all-reasons, all-history semantics; corrupt
// or unreadable markers must not silently choose either branch.
func (k Keeper) HaltReasonFilterEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.HaltReasonFilterEnabledKey)
	if err != nil {
		return false, fmt.Errorf("read halt reason filter marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed halt reason filter marker")
	}
	return true, nil
}

// EnableHaltReasonFilter is an idempotent, keeper-only activation seam for a
// separately reviewed scheduled upgrade. No message, parameter, constructor or
// existing migration calls it. It does not rewrite ceremonies or governance
// history, revalidate pending rollbacks, or release an emergency transition hold.
//
// The carrier must inspect historical halt payloads and pending transitions
// before enabling the marker in its upgrade cache. The current protobuf genesis
// schema cannot carry this marker: an export/import continuation needs separate
// explicit restoration by its carrier, not an assumption that export preserves it.
func (k Keeper) EnableHaltReasonFilter(ctx context.Context) error {
	enabled, err := k.HaltReasonFilterEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.HaltReasonFilterEnabledKey, []byte{1}); err != nil {
		return fmt.Errorf("write halt reason filter marker: %w", err)
	}
	return nil
}
