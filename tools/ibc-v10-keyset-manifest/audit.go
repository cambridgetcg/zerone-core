package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"

	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/types"
	"github.com/cosmos/iavl"
	iavldb "github.com/cosmos/iavl/db"
	ics23 "github.com/cosmos/ics23/go"
)

const (
	rootLatestVersionKey = "s/latest"
	rootCommitInfoPrefix = "s/"

	ibcPhysicalStorePrefix       = "s/k:ibc/"
	channelUpgradesLogicalPrefix = "channelUpgrades/"
	pruningSequenceLogicalPrefix = "pruningSequenceStart/"

	maxLatestVersionBytes = 16
	maxCommitInfoBytes    = 16 << 20
	maxCommitInfoStores   = 4096
	maxIBCStoreLeafCount  = 5_000_000
	maxLogicalKeyBytes    = 64 << 10
	maxLogicalValueBytes  = 64 << 20
	maxScannedInputBytes  = 1 << 30
)

type physicalDB interface {
	Get(key []byte) ([]byte, error)
	Iterator(start, end []byte) (dbm.Iterator, error)
	ReverseIterator(start, end []byte) (dbm.Iterator, error)
}

type expectedEvidence struct {
	Height  int64
	AppHash []byte
}

type rootSnapshot struct {
	height           int64
	latestVersionRaw []byte
	commitInfoKey    []byte
	commitInfoRaw    []byte
	ibcHash          []byte
}

type scanBudget struct {
	inputBytes uint64
}

var emptyIAVLRootHash = sha256.Sum256(nil)

func auditApplicationDB(db physicalDB, expected expectedEvidence) ([]byte, error) {
	if db == nil {
		return nil, errors.New("application database is nil")
	}
	if expected.Height <= 0 {
		return nil, errors.New("expected height must be positive")
	}
	if len(expected.AppHash) != 32 {
		return nil, errors.New("expected app hash must be exactly 32 bytes")
	}

	before, err := readAndVerifyRootSnapshot(db, expected)
	if err != nil {
		return nil, err
	}

	channelKeys, pruningKeys, err := collectRegularIAVLKeysets(db, before)
	if err != nil {
		return nil, err
	}

	if err := verifySnapshotUnchanged(db, before); err != nil {
		return nil, err
	}

	return buildPlanInfo(channelKeys, pruningKeys)
}

func readAndVerifyRootSnapshot(
	db physicalDB,
	expected expectedEvidence,
) (rootSnapshot, error) {
	latestRaw, err := getRequiredBounded(
		db,
		[]byte(rootLatestVersionKey),
		maxLatestVersionBytes,
	)
	if err != nil {
		return rootSnapshot{}, fmt.Errorf("read root latest version: %w", err)
	}

	var height int64
	if err := types.StdInt64Unmarshal(&height, latestRaw); err != nil {
		return rootSnapshot{}, fmt.Errorf("decode root latest version: %w", err)
	}
	canonicalHeight, err := types.StdInt64Marshal(height)
	if err != nil {
		return rootSnapshot{}, fmt.Errorf("canonicalize root latest version: %w", err)
	}
	if !bytes.Equal(latestRaw, canonicalHeight) {
		return rootSnapshot{}, errors.New("root latest version is not canonically encoded")
	}
	if height <= 0 {
		return rootSnapshot{}, fmt.Errorf("root latest version must be positive: got %d", height)
	}
	if height != expected.Height {
		return rootSnapshot{}, fmt.Errorf(
			"root latest height mismatch: expected %d, got %d",
			expected.Height,
			height,
		)
	}

	commitInfoKey := []byte(rootCommitInfoPrefix + strconv.FormatInt(height, 10))
	commitInfoRaw, err := getRequiredBounded(db, commitInfoKey, maxCommitInfoBytes)
	if err != nil {
		return rootSnapshot{}, fmt.Errorf("read root commit info at height %d: %w", height, err)
	}

	var commitInfo storetypes.CommitInfo
	if err := commitInfo.Unmarshal(commitInfoRaw); err != nil {
		return rootSnapshot{}, fmt.Errorf("decode root commit info at height %d: %w", height, err)
	}
	if commitInfo.Version != height {
		return rootSnapshot{}, fmt.Errorf(
			"root commit info version mismatch: latest is %d, commit info says %d",
			height,
			commitInfo.Version,
		)
	}
	if len(commitInfo.StoreInfos) == 0 {
		return rootSnapshot{}, errors.New("root commit info has no mounted stores")
	}
	if len(commitInfo.StoreInfos) > maxCommitInfoStores {
		return rootSnapshot{}, fmt.Errorf(
			"root commit info exceeds %d mounted stores",
			maxCommitInfoStores,
		)
	}

	seenNames := make(map[string]struct{}, len(commitInfo.StoreInfos))
	ibcStores := 0
	var ibcHash []byte
	for _, storeInfo := range commitInfo.StoreInfos {
		if storeInfo.Name == "" {
			return rootSnapshot{}, errors.New("root commit info contains an empty store name")
		}
		if _, exists := seenNames[storeInfo.Name]; exists {
			return rootSnapshot{}, fmt.Errorf(
				"root commit info contains duplicate store name %q",
				storeInfo.Name,
			)
		}
		seenNames[storeInfo.Name] = struct{}{}
		if len(storeInfo.CommitId.Hash) != 32 {
			return rootSnapshot{}, fmt.Errorf(
				"root commit info store %q hash must be exactly 32 bytes: got %d",
				storeInfo.Name,
				len(storeInfo.CommitId.Hash),
			)
		}
		if storeInfo.Name == "ibc" {
			ibcStores++
			ibcHash = bytes.Clone(storeInfo.CommitId.Hash)
			if storeInfo.CommitId.Version != height {
				return rootSnapshot{}, fmt.Errorf(
					"IBC commit version mismatch: root height is %d, IBC store is %d",
					height,
					storeInfo.CommitId.Version,
				)
			}
		}
	}
	if ibcStores != 1 {
		return rootSnapshot{}, fmt.Errorf(
			"root commit info must contain exactly one IBC store: got %d",
			ibcStores,
		)
	}

	appHash := commitInfo.Hash()
	if len(appHash) != 32 {
		return rootSnapshot{}, fmt.Errorf(
			"computed root app hash must be 32 bytes: got %d",
			len(appHash),
		)
	}
	if !bytes.Equal(appHash, expected.AppHash) {
		return rootSnapshot{}, fmt.Errorf(
			"root app hash mismatch: expected %s, got %s",
			hex.EncodeToString(expected.AppHash),
			hex.EncodeToString(appHash),
		)
	}

	return rootSnapshot{
		height:           height,
		latestVersionRaw: bytes.Clone(latestRaw),
		commitInfoKey:    bytes.Clone(commitInfoKey),
		commitInfoRaw:    bytes.Clone(commitInfoRaw),
		ibcHash:          ibcHash,
	}, nil
}

func verifySnapshotUnchanged(db physicalDB, before rootSnapshot) error {
	checks := []struct {
		name string
		key  []byte
		want []byte
		max  int
	}{
		{
			name: "root latest version",
			key:  []byte(rootLatestVersionKey),
			want: before.latestVersionRaw,
			max:  maxLatestVersionBytes,
		},
		{
			name: "root commit info",
			key:  before.commitInfoKey,
			want: before.commitInfoRaw,
			max:  maxCommitInfoBytes,
		},
	}
	for _, check := range checks {
		got, err := getRequiredBounded(db, check.key, check.max)
		if err != nil {
			return fmt.Errorf("re-read %s: %w", check.name, err)
		}
		if !bytes.Equal(got, check.want) {
			return fmt.Errorf("%s changed during the raw database census", check.name)
		}
	}
	return nil
}

func getRequiredBounded(db physicalDB, key []byte, maxBytes int) ([]byte, error) {
	bz, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, fmt.Errorf("required key %q is missing", key)
	}
	if len(bz) > maxBytes {
		return nil, fmt.Errorf(
			"value for key %q exceeds %d bytes",
			key,
			maxBytes,
		)
	}
	return bz, nil
}

func collectRegularIAVLKeysets(
	db physicalDB,
	snapshot rootSnapshot,
) (channelKeys, pruningKeys [][]byte, err error) {
	// IAVL's canonical empty root is SHA256(empty). An externally attested
	// rootmulti CommitInfo containing that IBC hash cryptographically binds the
	// logical store at H to zero leaves, so no physical traversal is needed.
	// This also handles valid backend-specific encodings: untouched stores may
	// have no IAVL version, while nonempty-then-emptied stores retain history.
	if bytes.Equal(snapshot.ibcHash, emptyIAVLRootHash[:]) {
		return nil, nil, nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			channelKeys = nil
			pruningKeys = nil
			err = errors.Join(
				err,
				fmt.Errorf("panic while reading copied IBC IAVL tree: %v", recovered),
			)
		}
	}()

	readOnlyRoot := newReadOnlyPhysicalDB(db)
	ibcDB := dbm.NewPrefixDB(readOnlyRoot, []byte(ibcPhysicalStorePrefix))
	tree := iavl.NewMutableTree(
		iavldb.NewWrapper(ibcDB),
		0,
		true, // Never inspect, create, or rebuild IAVL fast storage.
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	defer func() {
		closeErr := tree.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close copied IBC IAVL tree: %w", closeErr)
		}
		var writeErr error
		if attempts := readOnlyRoot.WriteAttempts(); attempts != 0 {
			writeErr = fmt.Errorf(
				"copied IBC IAVL read attempted %d database mutations",
				attempts,
			)
		}
		err = errors.Join(err, closeErr, writeErr)
	}()

	latestVersion, err := tree.LoadVersion(snapshot.height)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"load copied IBC IAVL version %d without fast storage: %w",
			snapshot.height,
			err,
		)
	}
	if latestVersion != snapshot.height || tree.Version() != snapshot.height {
		return nil, nil, fmt.Errorf(
			"copied IBC IAVL version mismatch: root=%d latest=%d loaded=%d",
			snapshot.height,
			latestVersion,
			tree.Version(),
		)
	}
	if !bytes.Equal(tree.Hash(), snapshot.ibcHash) {
		return nil, nil, fmt.Errorf(
			"copied IBC IAVL root mismatch at height %d: commit info=%s loaded tree=%s",
			snapshot.height,
			hex.EncodeToString(snapshot.ibcHash),
			hex.EncodeToString(tree.Hash()),
		)
	}
	return scanEntireIBCStore(tree, snapshot.ibcHash, &scanBudget{})
}

type verifiableIBCStore interface {
	Size() int64
	Iterator(start, end []byte, ascending bool) (iavldb.Iterator, error)
	GetMembershipProof(key []byte) (*ics23.CommitmentProof, error)
}

// scanEntireIBCStore does not trust IAVL v1.2.2 Iterator.Error for
// completeness: that iterator can silently swallow a child-load error. It
// therefore scans every leaf, requires the observed count to equal the
// root-hash-bound tree size, and independently verifies an ICS23 membership
// proof for every emitted key/value against the CommitInfo IBC root.
func scanEntireIBCStore(
	store verifiableIBCStore,
	committedRoot []byte,
	budget *scanBudget,
) (channelKeys, pruningKeys [][]byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			channelKeys = nil
			pruningKeys = nil
			err = errors.Join(
				err,
				fmt.Errorf("panic while traversing copied IBC IAVL tree: %v", recovered),
			)
		}
	}()
	if store == nil {
		return nil, nil, errors.New("IBC IAVL tree is nil")
	}
	if len(committedRoot) != 32 {
		return nil, nil, errors.New("committed IBC root must be exactly 32 bytes")
	}
	if budget == nil {
		return nil, nil, errors.New("scan budget is nil")
	}

	expectedLeaves := store.Size()
	if expectedLeaves < 0 || expectedLeaves > maxIBCStoreLeafCount {
		return nil, nil, fmt.Errorf(
			"IBC IAVL tree leaf count must be between 0 and %d: got %d",
			maxIBCStoreLeafCount,
			expectedLeaves,
		)
	}

	iterator, err := store.Iterator(nil, nil, true)
	if err != nil {
		return nil, nil, fmt.Errorf("open complete IBC IAVL iterator: %w", err)
	}
	if iterator == nil {
		return nil, nil, errors.New("open complete IBC IAVL iterator: nil iterator")
	}
	defer func() {
		iterationErr := iterator.Error()
		if iterationErr != nil {
			iterationErr = fmt.Errorf("iterate complete IBC IAVL tree: %w", iterationErr)
		}
		closeErr := iterator.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close complete IBC IAVL iterator: %w", closeErr)
		}
		err = errors.Join(err, iterationErr, closeErr)
	}()

	channelPrefix := []byte(channelUpgradesLogicalPrefix)
	pruningPrefix := []byte(pruningSequenceLogicalPrefix)
	var (
		previousKey  []byte
		leafCount    int64
		channelBytes int
		pruningBytes int
	)
	for ; iterator.Valid(); iterator.Next() {
		if leafCount >= expectedLeaves {
			return nil, nil, fmt.Errorf(
				"IBC IAVL traversal emitted more than root-bound size %d",
				expectedLeaves,
			)
		}
		rawKey := iterator.Key()
		rawValue := iterator.Value()
		if len(rawKey) == 0 {
			return nil, nil, errors.New("IBC IAVL traversal emitted an empty key")
		}
		if len(rawKey) > maxLogicalKeyBytes {
			return nil, nil, fmt.Errorf(
				"IBC IAVL logical key exceeds %d bytes",
				maxLogicalKeyBytes,
			)
		}
		if len(rawValue) > maxLogicalValueBytes {
			return nil, nil, fmt.Errorf(
				"IBC IAVL value for key %q exceeds %d bytes",
				rawKey,
				maxLogicalValueBytes,
			)
		}
		if err := addScannedInputBytes(
			budget,
			uint64(len(rawKey))+uint64(len(rawValue)),
		); err != nil {
			return nil, nil, err
		}

		key := bytes.Clone(rawKey)
		value := bytes.Clone(rawValue)
		if previousKey != nil && bytes.Compare(previousKey, key) >= 0 {
			return nil, nil, fmt.Errorf(
				"IBC IAVL traversal is not in strict byte order: %q then %q",
				previousKey,
				key,
			)
		}
		previousKey = bytes.Clone(key)

		proof, err := store.GetMembershipProof(key)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"create IBC IAVL membership proof for key %q: %w",
				key,
				err,
			)
		}
		if !ics23.VerifyMembership(
			ics23.IavlSpec,
			committedRoot,
			proof,
			key,
			value,
		) {
			return nil, nil, fmt.Errorf(
				"IBC IAVL membership proof for key %q does not verify against committed root",
				key,
			)
		}

		switch {
		case bytes.HasPrefix(key, channelPrefix):
			if len(channelKeys) >= maxKeysPerDomain {
				return nil, nil, fmt.Errorf(
					"IBC domain %q exceeds %d keys",
					channelPrefix,
					maxKeysPerDomain,
				)
			}
			if len(key) > maxLogicalBytesPerDomain-channelBytes {
				return nil, nil, fmt.Errorf(
					"IBC domain %q exceeds %d aggregate logical key bytes",
					channelPrefix,
					maxLogicalBytesPerDomain,
				)
			}
			channelBytes += len(key)
			channelKeys = append(channelKeys, key)

		case bytes.HasPrefix(key, pruningPrefix):
			if len(pruningKeys) >= maxKeysPerDomain {
				return nil, nil, fmt.Errorf(
					"IBC domain %q exceeds %d keys",
					pruningPrefix,
					maxKeysPerDomain,
				)
			}
			if len(key) > maxLogicalBytesPerDomain-pruningBytes {
				return nil, nil, fmt.Errorf(
					"IBC domain %q exceeds %d aggregate logical key bytes",
					pruningPrefix,
					maxLogicalBytesPerDomain,
				)
			}
			pruningBytes += len(key)
			pruningKeys = append(pruningKeys, key)
		}
		leafCount++
	}
	if leafCount != expectedLeaves {
		return nil, nil, fmt.Errorf(
			"IBC IAVL traversal was incomplete: root-bound size=%d emitted=%d",
			expectedLeaves,
			leafCount,
		)
	}

	sort.Slice(channelKeys, func(i, j int) bool {
		return bytes.Compare(channelKeys[i], channelKeys[j]) < 0
	})
	sort.Slice(pruningKeys, func(i, j int) bool {
		return bytes.Compare(pruningKeys[i], pruningKeys[j]) < 0
	})
	return channelKeys, pruningKeys, nil
}

func addScannedInputBytes(budget *scanBudget, amount uint64) error {
	if budget.inputBytes > maxScannedInputBytes ||
		amount > maxScannedInputBytes-budget.inputBytes {
		return fmt.Errorf(
			"complete IBC regular-IAVL scan exceeds %d aggregate input bytes",
			maxScannedInputBytes,
		)
	}
	budget.inputBytes += amount
	return nil
}
