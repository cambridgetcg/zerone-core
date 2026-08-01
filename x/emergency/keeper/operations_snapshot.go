package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

const (
	maxOperationsSafetyRecordCount = 100_000
	maxOperationsSafetyRecordBytes = 64 << 20
)

// OperationsSafetyRecord is one immutable H-1 emergency-store key/value pair
// supplied to the coordinated operations-safety migration. The application
// reconstructs the complete committed emergency IAVL root and binds that
// subroot through root CommitInfo to the H-1 app hash before constructing it.
type OperationsSafetyRecord struct {
	Key   []byte
	Value []byte
}

// OperationsSafetySnapshot contains every ceremony and resume-attempt index
// plus each singleton that can affect legacy emergency-state reconciliation.
// Records are strictly byte-sorted and unique; absent singleton keys are
// represented by absence.
type OperationsSafetySnapshot struct {
	Records []OperationsSafetyRecord
}

// NewOperationsSafetySnapshot clones, sorts, and validates raw records.
func NewOperationsSafetySnapshot(
	records []OperationsSafetyRecord,
) (OperationsSafetySnapshot, error) {
	if len(records) > maxOperationsSafetyRecordCount {
		return OperationsSafetySnapshot{}, fmt.Errorf(
			"operations-safety snapshot exceeds %d records",
			maxOperationsSafetyRecordCount,
		)
	}

	normalized := make([]OperationsSafetyRecord, len(records))
	totalBytes := 0
	for i, record := range records {
		if !IsOperationsSafetySnapshotKey(record.Key) {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"operations-safety snapshot contains out-of-domain key %x",
				record.Key,
			)
		}
		if len(record.Key) > maxOperationsSafetyRecordBytes-totalBytes {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"operations-safety snapshot exceeds %d aggregate key/value bytes",
				maxOperationsSafetyRecordBytes,
			)
		}
		totalBytes += len(record.Key)
		if len(record.Value) > maxOperationsSafetyRecordBytes-totalBytes {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"operations-safety snapshot exceeds %d aggregate key/value bytes",
				maxOperationsSafetyRecordBytes,
			)
		}
		totalBytes += len(record.Value)
		normalized[i] = OperationsSafetyRecord{
			Key:   bytes.Clone(record.Key),
			Value: bytes.Clone(record.Value),
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		return bytes.Compare(normalized[i].Key, normalized[j].Key) < 0
	})
	for i := 1; i < len(normalized); i++ {
		if bytes.Equal(normalized[i-1].Key, normalized[i].Key) {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"operations-safety snapshot contains duplicate key %x",
				normalized[i].Key,
			)
		}
	}
	return OperationsSafetySnapshot{Records: normalized}, nil
}

// IsOperationsSafetySnapshotKey reports whether a raw emergency-store key can
// affect the one-time reconciliation. The full ceremony and resume-attempt
// domains are included; all other inputs are exact singleton keys and therefore
// do not depend on an iterator for completeness.
func IsOperationsSafetySnapshotKey(key []byte) bool {
	if bytes.HasPrefix(key, types.CeremonyKeyPrefix) ||
		bytes.HasPrefix(key, types.ResumeAttemptKeyPrefix) {
		return true
	}
	for _, singleton := range operationsSafetySingletonKeys() {
		if bytes.Equal(key, singleton) {
			return true
		}
	}
	return false
}

// ReadOperationsSafetySnapshot reads the current context store. It is kept for
// unit tests and local migration calls. Coordinated upgrade handlers instead
// authenticate a complete committed IAVL export through root CommitInfo before
// calling MigrateOperationsSafetyV1FromSnapshot.
func (k Keeper) ReadOperationsSafetySnapshot(
	ctx context.Context,
) (OperationsSafetySnapshot, error) {
	store := k.storeService.OpenKVStore(ctx)
	records := make([]OperationsSafetyRecord, 0)

	for _, prefix := range [][]byte{
		types.CeremonyKeyPrefix,
		types.ResumeAttemptKeyPrefix,
	} {
		iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
		if err != nil {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"open operations-safety prefix %x iterator: %w",
				prefix,
				err,
			)
		}
		for ; iter.Valid(); iter.Next() {
			records = append(records, OperationsSafetyRecord{
				Key:   bytes.Clone(iter.Key()),
				Value: bytes.Clone(iter.Value()),
			})
		}
		iterationErr := terminalIteratorError(iter)
		closeErr := iter.Close()
		if err := errors.Join(iterationErr, closeErr); err != nil {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"read operations-safety prefix %x: %w",
				prefix,
				err,
			)
		}
	}

	for _, key := range operationsSafetySingletonKeys() {
		value, err := store.Get(key)
		if err != nil {
			return OperationsSafetySnapshot{}, fmt.Errorf(
				"read operations-safety singleton %x: %w",
				key,
				err,
			)
		}
		if value != nil {
			records = append(records, OperationsSafetyRecord{
				Key:   bytes.Clone(key),
				Value: bytes.Clone(value),
			})
		}
	}
	return NewOperationsSafetySnapshot(records)
}

func operationsSafetySingletonKeys() [][]byte {
	return [][]byte{
		types.ParamsKey,
		types.HaltStatusKey,
		types.ActiveHaltCeremonyIdKey,
		types.HaltStartBlockKey,
		types.RevertTargetHeightKey,
		types.RevertTargetHashKey,
		types.RevertCeremonyIdKey,
		types.LastHaltEscalationBlockKey,
		types.ActiveCeremonyIdKey,
		types.OperationsSafetyPreparedKey,
		types.OperationsSafetyActivatedKey,
		types.QuarantineReleaseBlockKey,
		types.RecoveryAuthorizationKey,
	}
}

func operationsSafetySnapshotMap(
	snapshot OperationsSafetySnapshot,
) (map[string][]byte, error) {
	normalized, err := NewOperationsSafetySnapshot(snapshot.Records)
	if err != nil {
		return nil, err
	}
	records := make(map[string][]byte, len(normalized.Records))
	for _, record := range normalized.Records {
		records[string(record.Key)] = bytes.Clone(record.Value)
	}
	return records, nil
}
