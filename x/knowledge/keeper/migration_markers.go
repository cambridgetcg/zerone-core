package keeper

import (
	"context"
	"fmt"
)

// ─── Wave 10: migration marker side-channel ──────────────────────────────
//
// A dedicated sub-namespace (prefix 0x7F) for migration "I ran" markers.
// Kept outside any Route B or core namespace so markers can never conflict
// with domain state. Readable by tests + operators to prove that a named
// migration executed.
//
// Pattern: every migration writes a marker (key, value) at the end of its
// successful run. Post-upgrade tests read the marker to assert execution.
// Markers are append-only — no migration may overwrite another's marker
// with a conflicting value. If a marker is already written, subsequent
// writes with the same value are no-ops; different values return an error.

var migrationMarkerPrefix = []byte{0x7F, 0x01}

// WriteMigrationMarker records a marker announcing a migration executed.
// Idempotent on (key, same-value); errors on (key, different-value).
func (k Keeper) WriteMigrationMarker(ctx context.Context, key, value string) error {
	if key == "" {
		return fmt.Errorf("migration marker key cannot be empty")
	}
	if value == "" {
		return fmt.Errorf("migration marker %q value cannot be empty", key)
	}
	store := k.storeService.OpenKVStore(ctx)
	full := append(append([]byte{}, migrationMarkerPrefix...), []byte(key)...)

	existing, err := store.Get(full)
	if err != nil {
		return fmt.Errorf("read migration marker %q before write: %w", key, err)
	}
	if existing != nil {
		if string(existing) == value {
			return nil // idempotent
		}
		// Divergent marker — flag rather than silently overwrite.
		k.Logger(ctx).Warn("migration marker collision",
			"key", key, "existing", string(existing), "incoming", value)
		// Preserve the first writer; do not overwrite.
		return fmt.Errorf(
			"migration marker %q conflicts: existing value %q, incoming value %q",
			key,
			string(existing),
			value,
		)
	}
	if err := store.Set(full, []byte(value)); err != nil {
		return fmt.Errorf("write migration marker %q: %w", key, err)
	}
	return nil
}

// ReadMigrationMarker returns the value for a marker key, or "" if absent.
func (k Keeper) ReadMigrationMarker(ctx context.Context, key string) string {
	value, err := k.ReadMigrationMarkerChecked(ctx, key)
	if err != nil {
		panic(err)
	}
	return value
}

// ReadMigrationMarkerChecked is the fail-closed form used by upgrade
// preconditions. It distinguishes an absent marker from an unreadable store.
func (k Keeper) ReadMigrationMarkerChecked(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("migration marker key cannot be empty")
	}
	store := k.storeService.OpenKVStore(ctx)
	full := append(append([]byte{}, migrationMarkerPrefix...), []byte(key)...)
	bz, err := store.Get(full)
	if err != nil {
		return "", fmt.Errorf("read migration marker %q: %w", key, err)
	}
	if bz == nil {
		return "", nil
	}
	return string(bz), nil
}
