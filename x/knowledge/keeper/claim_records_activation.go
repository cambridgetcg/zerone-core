package keeper

import (
	"bytes"
	"context"
	"fmt"
)

// ClaimRecordsEnabledStoreKey selects future contradiction input retention.
// It is independent of migration completion so native/imported genesis does
// not pretend that an upgrade ran. Existing records are never rewritten.
const ClaimRecordsEnabledStoreKey = "\x85"

func (k Keeper) ClaimRecordsEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(ClaimRecordsEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read claim records marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed claim records marker")
	}
	return true, nil
}

func (k Keeper) EnableClaimRecords(ctx context.Context) error {
	enabled, err := k.ClaimRecordsEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	records, err := k.RecordIntegrityEnabled(ctx)
	if err != nil || !records {
		return fmt.Errorf("claim records requires record integrity: %v", err)
	}
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil || !neutral {
		return fmt.Errorf("claim records requires review neutrality: %v", err)
	}
	return k.storeService.OpenKVStore(ctx).Set([]byte(ClaimRecordsEnabledStoreKey), []byte{1})
}
