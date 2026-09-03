package keeper

import (
	"context"
	"fmt"

	"github.com/zerone-chain/zerone/x/knowledge/types"
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
	value, _, err := k.ReadMigrationMarkerPresenceChecked(ctx, key)
	return value, err
}

// ReadMigrationMarkerPresenceChecked preserves marker-key presence separately
// from its value. Historical code permitted an empty value, so callers that
// must prove a marker is absent cannot safely infer absence from value == "".
func (k Keeper) ReadMigrationMarkerPresenceChecked(
	ctx context.Context,
	key string,
) (value string, found bool, err error) {
	if key == "" {
		return "", false, fmt.Errorf("migration marker key cannot be empty")
	}
	store := k.storeService.OpenKVStore(ctx)
	full := append(append([]byte{}, migrationMarkerPrefix...), []byte(key)...)
	bz, err := store.Get(full)
	if err != nil {
		return "", false, fmt.Errorf("read migration marker %q: %w", key, err)
	}
	if bz == nil {
		return "", false, nil
	}
	return string(bz), true, nil
}

// AgentEconomyActivated reports whether exactly one reviewed activation
// lineage is present. Merely running module migrations is insufficient: this
// keeps a later binary restart or an unrelated broad RunMigrations handler
// from opening computational claims and escrow settlement.
func (k Keeper) AgentEconomyActivated(ctx context.Context) (bool, error) {
	active, _, _, _, err := k.AgentEconomyActivationStatus(ctx)
	return active, err
}

// AgentEconomyActivationStatus returns the exact marker evidence behind the
// binary gate. It is shared by message admission and the public read-only
// query so wallets do not have to infer activation from module versions.
func (k Keeper) AgentEconomyActivationStatus(
	ctx context.Context,
) (active bool, lineage string, marker string, value string, err error) {
	type markerState struct {
		key   string
		kind  string
		value string
		found bool
	}
	states := make([]markerState, 0, 2)
	for _, candidate := range []struct {
		key  string
		kind string
	}{
		{types.AgentEconomyUpgradeMarker, types.AgentEconomyLineageUpgrade},
		{types.AgentEconomyNativeMarker, types.AgentEconomyLineageNative},
	} {
		value, found, err := k.ReadMigrationMarkerPresenceChecked(ctx, candidate.key)
		if err != nil {
			return false, "", "", "", err
		}
		states = append(states, markerState{
			key: candidate.key, kind: candidate.kind, value: value, found: found,
		})
	}
	if states[0].found && states[1].found {
		return false, "", "", "", fmt.Errorf(
			"conflicting agent-economy lineages: both %q and %q are present",
			states[0].key,
			states[1].key,
		)
	}
	for _, state := range states {
		if !state.found {
			continue
		}
		if state.value != types.AgentEconomyActivationValue {
			return false, "", "", "", fmt.Errorf(
				"agent-economy marker %q has value %q, require %q",
				state.key,
				state.value,
				types.AgentEconomyActivationValue,
			)
		}
		return true, state.kind, state.key, state.value, nil
	}
	return false, types.AgentEconomyLineageNone, "", "", nil
}

func (k Keeper) requireAgentEconomyActivated(ctx context.Context) error {
	active, err := k.AgentEconomyActivated(ctx)
	if err != nil {
		return types.ErrAgentEconomyDisabled.Wrap(err.Error())
	}
	if !active {
		return types.ErrAgentEconomyDisabled
	}
	return nil
}
