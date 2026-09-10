package keeper

import (
	"bytes"
	"context"
	"fmt"
)

// RecordIntegrityEnabledStoreKey is separate from the migration-done markers:
// imported/native genesis preserves execution semantics without inventing an
// applied upgrade. 0x82/0x83 are the independently owned payout index/cursor.
const RecordIntegrityEnabledStoreKey = "\x81"

// RecordIntegrityEnabled never treats an unreadable or malformed marker as
// legacy state. Callers must propagate the error before choosing semantics.
func (k Keeper) RecordIntegrityEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(RecordIntegrityEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read record integrity marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed record integrity marker")
	}
	return true, nil
}

// EnableRecordIntegrity selects current semantics without altering any claim,
// round, payment or historical record. The owning migrator performs validation
// and writes its separate completion marker in the same outer cache.
func (k Keeper) EnableRecordIntegrity(ctx context.Context) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(RecordIntegrityEnabledStoreKey), []byte{1}); err != nil {
		return fmt.Errorf("write record integrity marker: %w", err)
	}
	return nil
}
