package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"

	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/types"
	"github.com/cosmos/iavl/fastnode"
)

const (
	rootLatestVersionKey = "s/latest"
	rootCommitInfoPrefix = "s/"

	ibcPhysicalStorePrefix       = "s/k:ibc/"
	iavlFastNodePrefix           = "f"
	iavlStorageVersionKey        = "mstorage_version"
	requiredFastStorageVersion   = "1.1.0"
	channelUpgradesLogicalPrefix = "channelUpgrades/"
	pruningSequenceLogicalPrefix = "pruningSequenceStart/"

	maxLatestVersionBytes  = 16
	maxCommitInfoBytes     = 16 << 20
	maxCommitInfoStores    = 4096
	maxStorageVersionBytes = 64
	maxPhysicalKeyBytes    = 64 << 10
	maxFastNodeValueBytes  = 64 << 20
	maxScannedInputBytes   = 1 << 30
)

var (
	ibcFastPhysicalPrefix = []byte(ibcPhysicalStorePrefix + iavlFastNodePrefix)
	ibcStorageVersionKey  = []byte(ibcPhysicalStorePrefix + iavlStorageVersionKey)
)

type physicalDB interface {
	Get(key []byte) ([]byte, error)
	Iterator(start, end []byte) (dbm.Iterator, error)
}

type expectedEvidence struct {
	Height  int64
	AppHash []byte
}

type rootSnapshot struct {
	height            int64
	appHash           []byte
	latestVersionRaw  []byte
	commitInfoKey     []byte
	commitInfoRaw     []byte
	storageVersionRaw []byte
}

type scanBudget struct {
	inputBytes uint64
}

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

	budget := &scanBudget{}
	channelKeys, err := scanFastNodeDomain(
		db,
		channelUpgradesLogicalPrefix,
		before.height,
		budget,
	)
	if err != nil {
		return nil, err
	}
	pruningKeys, err := scanFastNodeDomain(
		db,
		pruningSequenceLogicalPrefix,
		before.height,
		budget,
	)
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

	storageVersionRaw, err := getRequiredBounded(
		db,
		ibcStorageVersionKey,
		maxStorageVersionBytes,
	)
	if err != nil {
		return rootSnapshot{}, fmt.Errorf("read IBC IAVL fast-storage metadata: %w", err)
	}
	expectedStorageVersion := requiredFastStorageVersion + "-" + strconv.FormatInt(height, 10)
	if string(storageVersionRaw) != expectedStorageVersion {
		return rootSnapshot{}, fmt.Errorf(
			"IBC IAVL fast storage is not synchronized to root height %d: expected %q, got %q",
			height,
			expectedStorageVersion,
			storageVersionRaw,
		)
	}

	return rootSnapshot{
		height:            height,
		appHash:           bytes.Clone(appHash),
		latestVersionRaw:  bytes.Clone(latestRaw),
		commitInfoKey:     bytes.Clone(commitInfoKey),
		commitInfoRaw:     bytes.Clone(commitInfoRaw),
		storageVersionRaw: bytes.Clone(storageVersionRaw),
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
		{
			name: "IBC IAVL fast-storage metadata",
			key:  ibcStorageVersionKey,
			want: before.storageVersionRaw,
			max:  maxStorageVersionBytes,
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

func scanFastNodeDomain(
	db physicalDB,
	logicalPrefix string,
	rootHeight int64,
	budget *scanBudget,
) (keys [][]byte, err error) {
	if logicalPrefix == "" || logicalPrefix[len(logicalPrefix)-1] != '/' {
		return nil, fmt.Errorf("logical IBC domain prefix %q must end with slash", logicalPrefix)
	}
	if budget == nil {
		return nil, errors.New("scan budget is nil")
	}

	physicalPrefix := append(bytes.Clone(ibcFastPhysicalPrefix), logicalPrefix...)
	end := prefixEnd(physicalPrefix)
	if end == nil {
		return nil, fmt.Errorf("cannot calculate exclusive end for physical prefix %q", physicalPrefix)
	}

	iterator, err := db.Iterator(physicalPrefix, end)
	if err != nil {
		return nil, fmt.Errorf("open raw iterator for %q: %w", physicalPrefix, err)
	}
	if iterator == nil {
		return nil, fmt.Errorf("open raw iterator for %q: nil iterator", physicalPrefix)
	}
	defer func() {
		iterationErr := iterator.Error()
		if iterationErr != nil {
			iterationErr = fmt.Errorf(
				"iterate raw IBC fast-node domain %q: %w",
				physicalPrefix,
				iterationErr,
			)
		}
		closeErr := iterator.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf(
				"close raw IBC fast-node iterator %q: %w",
				physicalPrefix,
				closeErr,
			)
		}
		err = errors.Join(err, iterationErr, closeErr)
	}()

	logicalPrefixBytes := []byte(logicalPrefix)
	var previousPhysical []byte
	logicalBytes := 0
	for ; iterator.Valid(); iterator.Next() {
		if len(keys) >= maxKeysPerDomain {
			return nil, fmt.Errorf(
				"raw IBC fast-node domain %q exceeds %d keys",
				physicalPrefix,
				maxKeysPerDomain,
			)
		}

		physicalKey := iterator.Key()
		if len(physicalKey) > maxPhysicalKeyBytes {
			return nil, fmt.Errorf(
				"raw IBC fast-node key exceeds %d bytes",
				maxPhysicalKeyBytes,
			)
		}
		if !bytes.HasPrefix(physicalKey, physicalPrefix) {
			return nil, fmt.Errorf(
				"raw iterator for %q returned out-of-domain key %q",
				physicalPrefix,
				physicalKey,
			)
		}
		if previousPhysical != nil && bytes.Compare(previousPhysical, physicalKey) >= 0 {
			return nil, fmt.Errorf(
				"raw iterator for %q returned keys out of strict byte order",
				physicalPrefix,
			)
		}
		previousPhysical = bytes.Clone(physicalKey)

		logicalKey := bytes.Clone(physicalKey[len(ibcFastPhysicalPrefix):])
		if !bytes.HasPrefix(logicalKey, logicalPrefixBytes) {
			return nil, fmt.Errorf(
				"stripped IBC key %q is outside logical prefix %q",
				logicalKey,
				logicalPrefixBytes,
			)
		}
		if len(logicalKey) > maxLogicalBytesPerDomain-logicalBytes {
			return nil, fmt.Errorf(
				"raw IBC fast-node domain %q exceeds %d aggregate logical key bytes",
				physicalPrefix,
				maxLogicalBytesPerDomain,
			)
		}
		logicalBytes += len(logicalKey)

		value := iterator.Value()
		if len(value) > maxFastNodeValueBytes {
			return nil, fmt.Errorf(
				"raw IBC fast-node value for logical key %q exceeds %d bytes",
				logicalKey,
				maxFastNodeValueBytes,
			)
		}
		if err := addScannedInputBytes(
			budget,
			uint64(len(physicalKey))+uint64(len(value)),
		); err != nil {
			return nil, err
		}
		node, err := fastnode.DeserializeNode(logicalKey, value)
		if err != nil {
			return nil, fmt.Errorf(
				"decode raw IBC fast node for logical key %q: %w",
				logicalKey,
				err,
			)
		}
		if node.GetVersionLastUpdatedAt() <= 0 ||
			node.GetVersionLastUpdatedAt() > rootHeight {
			return nil, fmt.Errorf(
				"raw IBC fast node for logical key %q has invalid last-update version %d at root height %d",
				logicalKey,
				node.GetVersionLastUpdatedAt(),
				rootHeight,
			)
		}
		var canonical bytes.Buffer
		if err := node.WriteBytes(&canonical); err != nil {
			return nil, fmt.Errorf(
				"canonicalize raw IBC fast node for logical key %q: %w",
				logicalKey,
				err,
			)
		}
		if !bytes.Equal(value, canonical.Bytes()) {
			return nil, fmt.Errorf(
				"raw IBC fast node for logical key %q is not canonically encoded",
				logicalKey,
			)
		}

		keys = append(keys, logicalKey)
	}

	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i], keys[j]) < 0
	})
	for i := 1; i < len(keys); i++ {
		if bytes.Equal(keys[i-1], keys[i]) {
			return nil, fmt.Errorf(
				"raw IBC fast-node domain %q contains duplicate logical key %q",
				physicalPrefix,
				keys[i],
			)
		}
	}
	return keys, nil
}

func addScannedInputBytes(budget *scanBudget, amount uint64) error {
	if amount > maxScannedInputBytes-budget.inputBytes {
		return fmt.Errorf(
			"raw IBC fast-node scan exceeds %d aggregate input bytes",
			maxScannedInputBytes,
		)
	}
	budget.inputBytes += amount
	return nil
}

func prefixEnd(prefix []byte) []byte {
	end := bytes.Clone(prefix)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] != 0xff {
			end[i]++
			return end[:i+1]
		}
	}
	return nil
}
