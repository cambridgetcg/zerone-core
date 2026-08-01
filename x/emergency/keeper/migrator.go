package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

const (
	operationsSafetyTargetVersion = byte(2)
	operationsSafetyMarkerSize    = 1 + 8 + sha256.Size
)

// Migrator performs version-map-authenticated emergency store migrations.
type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 may run only after the named coordinated handler has verified
// the complete H-1 snapshot and prepared the state. This prevents an unrelated
// upgrade handler from silently advancing the module version around the
// operations-safety reconciliation.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	store := m.keeper.storeService.OpenKVStore(ctx)
	if activated, err := store.Get(types.OperationsSafetyActivatedKey); err != nil {
		return fmt.Errorf("read operations-safety activation marker: %w", err)
	} else if activated != nil {
		return fmt.Errorf(
			"operations-safety v2 activation marker already exists",
		)
	}
	prepared, err := store.Get(types.OperationsSafetyPreparedKey)
	if err != nil {
		return fmt.Errorf("read operations-safety preparation marker: %w", err)
	}
	height, _, err := decodeOperationsSafetyMarker(prepared)
	if err != nil {
		return fmt.Errorf(
			"emergency v1 to v2 migration requires named-handler preparation: %w",
			err,
		)
	}
	if ctx.BlockHeight() <= 0 || uint64(ctx.BlockHeight()) != height {
		return fmt.Errorf(
			"operations-safety preparation height %d does not match migration height %d",
			height,
			ctx.BlockHeight(),
		)
	}
	if err := store.Set(
		types.OperationsSafetyActivatedKey,
		bytes.Clone(prepared),
	); err != nil {
		return fmt.Errorf("write operations-safety activation marker: %w", err)
	}
	if err := store.Delete(types.OperationsSafetyPreparedKey); err != nil {
		return fmt.Errorf("clear operations-safety preparation marker: %w", err)
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.emergency.operations_safety_activated",
		sdk.NewAttribute("consensus_version", "2"),
		sdk.NewAttribute("activation_height", fmt.Sprintf("%d", height)),
		sdk.NewAttribute(
			"h_minus_one_snapshot_sha256",
			fmt.Sprintf("%x", prepared[9:]),
		),
	))
	return nil
}

// PrepareOperationsSafetyV2FromSnapshot performs the named handler's complete
// reconciliation and writes a height- and snapshot-bound one-block handoff for
// the registered 1->2 module migration.
func (k Keeper) PrepareOperationsSafetyV2FromSnapshot(
	ctx context.Context,
	snapshot OperationsSafetySnapshot,
) (uint64, error) {
	normalized, err := NewOperationsSafetySnapshot(snapshot.Records)
	if err != nil {
		return 0, fmt.Errorf("validate operations-safety snapshot: %w", err)
	}
	for _, record := range normalized.Records {
		if bytes.Equal(record.Key, types.OperationsSafetyPreparedKey) ||
			bytes.Equal(record.Key, types.OperationsSafetyActivatedKey) ||
			bytes.Equal(record.Key, types.QuarantineReleaseBlockKey) ||
			bytes.Equal(record.Key, types.RecoveryAuthorizationKey) {
			return 0, fmt.Errorf(
				"irreparable emergency migration lineage: reserved marker key %x already exists",
				record.Key,
			)
		}
	}

	terminalized, err := k.MigrateOperationsSafetyV1FromSnapshot(
		ctx,
		normalized,
	)
	if err != nil {
		return 0, err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() <= 0 {
		return 0, fmt.Errorf(
			"operations-safety preparation requires a positive activation height",
		)
	}
	digest := operationsSafetySnapshotDigest(normalized)
	marker := make([]byte, operationsSafetyMarkerSize)
	marker[0] = operationsSafetyTargetVersion
	binary.BigEndian.PutUint64(marker[1:9], uint64(sdkCtx.BlockHeight()))
	copy(marker[9:], digest[:])
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.OperationsSafetyPreparedKey, marker); err != nil {
		return 0, fmt.Errorf("write operations-safety preparation marker: %w", err)
	}
	return terminalized, nil
}

func operationsSafetySnapshotDigest(
	snapshot OperationsSafetySnapshot,
) [sha256.Size]byte {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("zerone.emergency.operations-safety-h-minus-one/v1"))
	var scalar [8]byte
	for _, record := range snapshot.Records {
		binary.BigEndian.PutUint64(scalar[:], uint64(len(record.Key)))
		_, _ = hasher.Write(scalar[:])
		_, _ = hasher.Write(record.Key)
		binary.BigEndian.PutUint64(scalar[:], uint64(len(record.Value)))
		_, _ = hasher.Write(scalar[:])
		_, _ = hasher.Write(record.Value)
	}
	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return digest
}

func decodeOperationsSafetyMarker(
	marker []byte,
) (uint64, [sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	if len(marker) != operationsSafetyMarkerSize {
		return 0, digest, fmt.Errorf(
			"marker has %d bytes, expected %d",
			len(marker),
			operationsSafetyMarkerSize,
		)
	}
	if marker[0] != operationsSafetyTargetVersion {
		return 0, digest, fmt.Errorf(
			"marker target version is %d, expected %d",
			marker[0],
			operationsSafetyTargetVersion,
		)
	}
	height := binary.BigEndian.Uint64(marker[1:9])
	if height == 0 {
		return 0, digest, fmt.Errorf("marker activation height is zero")
	}
	copy(digest[:], marker[9:])
	return height, digest, nil
}

// GetOperationsSafetyV2Activation returns the durable named-handler lineage
// proof written by the registered 1->2 migration.
func (k Keeper) GetOperationsSafetyV2Activation(
	ctx context.Context,
) (uint64, [sha256.Size]byte, bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	marker, err := store.Get(types.OperationsSafetyActivatedKey)
	if err != nil {
		return 0, [sha256.Size]byte{}, false, fmt.Errorf(
			"read operations-safety activation marker: %w",
			err,
		)
	}
	if marker == nil {
		return 0, [sha256.Size]byte{}, false, nil
	}
	height, digest, err := decodeOperationsSafetyMarker(marker)
	if err != nil {
		return 0, [sha256.Size]byte{}, false, fmt.Errorf(
			"decode operations-safety activation marker: %w",
			err,
		)
	}
	return height, digest, true, nil
}
