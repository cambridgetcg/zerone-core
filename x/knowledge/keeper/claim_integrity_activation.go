package keeper

import (
	"bytes"
	"context"
	"fmt"
)

// ClaimIntegrityEnabledStoreKey selects knowledge-v12 claim semantics:
// status-aware duplicate checks, canonical form v2, accepted-only contest,
// decision-path reversal, recorded transition causes and conflicted-reviewer
// exclusion. It is independent of the migration-done marker so native or
// imported genesis does not pretend that an upgrade ran. No message or param
// toggles it, and nothing deactivates it. Existing records are never rewritten.
const ClaimIntegrityEnabledStoreKey = "\x87"

// ClaimIntegrityEnabled never treats an unreadable or malformed marker as
// legacy state. Callers must propagate the error before choosing semantics.
func (k Keeper) ClaimIntegrityEnabled(ctx context.Context) (bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get([]byte(ClaimIntegrityEnabledStoreKey))
	if err != nil {
		return false, fmt.Errorf("read claim integrity marker: %w", err)
	}
	if bz == nil {
		return false, nil
	}
	if !bytes.Equal(bz, []byte{1}) {
		return false, fmt.Errorf("malformed claim integrity marker")
	}
	return true, nil
}

// requireClaimIntegrityPredecessors proves every earlier selection is on.
func (k Keeper) requireClaimIntegrityPredecessors(ctx context.Context) error {
	for _, predecessor := range []struct {
		name    string
		enabled func(context.Context) (bool, error)
	}{
		{"record integrity", k.RecordIntegrityEnabled},
		{"review neutrality", k.ReviewNeutralityEnabled},
		{"claim records", k.ClaimRecordsEnabled},
		{"fund settlement", k.FundSettlementEnabled},
	} {
		enabled, err := predecessor.enabled(ctx)
		if err != nil || !enabled {
			return fmt.Errorf("claim integrity requires %s: %v", predecessor.name, err)
		}
	}
	return nil
}

// EnableClaimIntegrity selects v12 semantics without altering any claim, fact,
// round, index, payment or historical record. The owning migrator validates
// and writes its separate completion marker in the same outer cache.
func (k Keeper) EnableClaimIntegrity(ctx context.Context) error {
	enabled, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil || enabled {
		return err
	}
	if err := k.requireClaimIntegrityPredecessors(ctx); err != nil {
		return err
	}
	if err := k.storeService.OpenKVStore(ctx).Set([]byte(ClaimIntegrityEnabledStoreKey), []byte{1}); err != nil {
		return fmt.Errorf("write claim integrity marker: %w", err)
	}
	return nil
}

// ValidateClaimIntegrityActivation proves the exact knowledge-11 source: the
// marker is absent, every predecessor is on, and no v12 receipt exists. It
// inventories nothing else; activation rewrites no record and rebuilds no index.
func (k Keeper) ValidateClaimIntegrityActivation(ctx context.Context) error {
	enabled, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("claim integrity requires unenabled predecessor: %v", err)
	}
	if err := k.requireClaimIntegrityPredecessors(ctx); err != nil {
		return err
	}
	_, marked, err := k.ReadMigrationMarkerPresenceChecked(ctx, "migration_v12_complete")
	if err != nil || marked {
		return fmt.Errorf("claim integrity predecessor already marked or unreadable: %v", err)
	}
	return nil
}
