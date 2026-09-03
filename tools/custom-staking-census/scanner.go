package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"unicode/utf8"

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

	maxLatestVersionBytes = 16
	maxCommitInfoBytes    = 16 << 20
	maxCommitInfoStores   = 4096
	maxStoreLeafCount     = 5_000_000
	maxCustomStoreLeaves  = 50_000
	maxLogicalKeyBytes    = 64 << 10
	maxLogicalValueBytes  = 4 << 20
	maxScannedInputBytes  = 1 << 30
	maxCustomInputBytes   = 32 << 20
)

var (
	requiredStoreNames = [...]string{"zerone_staking", "bank", "staking"}
	emptyIAVLRootHash  = sha256.Sum256(nil)
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

type logicalLeaf struct {
	Key   []byte
	Value []byte
}

type committedStore struct {
	version  int64
	rootHash []byte
}

type rootSnapshot struct {
	height           int64
	appHash          []byte
	latestVersionRaw []byte
	commitInfoKey    []byte
	commitInfoRaw    []byte
	stores           map[string]committedStore
}

type storeEvidence struct {
	name       string
	version    int64
	rootHash   []byte
	leafCount  int64
	inputBytes uint64
	leavesHash []byte
}

type scanBudget struct {
	inputBytes uint64
}

// scanApplicationDB verifies the externally expected rootmulti commit, then
// streams every logical leaf of each required IAVL store through its visitor.
// No domain filtering occurs here: complete store traversal is part of the
// evidence boundary. Evidence is returned in requiredStoreNames order.
func scanApplicationDB(
	db physicalDB,
	expected expectedEvidence,
	visitors map[string]func(logicalLeaf) error,
) (snapshot rootSnapshot, evidence []storeEvidence, err error) {
	if db == nil {
		return rootSnapshot{}, nil, errors.New("application database is nil")
	}
	if expected.Height <= 0 {
		return rootSnapshot{}, nil, errors.New("expected height must be positive")
	}
	if len(expected.AppHash) != sha256.Size {
		return rootSnapshot{}, nil, errors.New("expected app hash must be exactly 32 bytes")
	}
	if err := validateVisitors(visitors); err != nil {
		return rootSnapshot{}, nil, err
	}

	before, err := readAndVerifyRootSnapshot(db, expected)
	if err != nil {
		return rootSnapshot{}, nil, err
	}
	defer func() {
		if stabilityErr := verifySnapshotUnchanged(db, before); stabilityErr != nil {
			snapshot = rootSnapshot{}
			evidence = nil
			err = errors.Join(err, stabilityErr)
		}
	}()

	budget := &scanBudget{}
	evidence = make([]storeEvidence, 0, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		committed := before.stores[name]
		storeResult, scanErr := scanCommittedIAVLStore(
			db,
			before.height,
			name,
			committed,
			visitors[name],
			budget,
		)
		if scanErr != nil {
			return rootSnapshot{}, nil, scanErr
		}
		evidence = append(evidence, storeResult)
	}
	return before, evidence, nil
}

func validateVisitors(visitors map[string]func(logicalLeaf) error) error {
	if visitors == nil {
		return errors.New("store visitors are required")
	}
	required := make(map[string]struct{}, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		required[name] = struct{}{}
		visitor, exists := visitors[name]
		if !exists || visitor == nil {
			return fmt.Errorf("visitor for required store %q is missing", name)
		}
	}
	for name := range visitors {
		if _, exists := required[name]; !exists {
			return fmt.Errorf("visitor names unsupported store %q", name)
		}
	}
	return nil
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
	if err := preflightCommitInfoProto(commitInfoRaw); err != nil {
		return rootSnapshot{}, fmt.Errorf("preflight root commit info at height %d: %w", height, err)
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

	required := make(map[string]struct{}, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		required[name] = struct{}{}
	}
	seenNames := make(map[string]struct{}, len(commitInfo.StoreInfos))
	stores := make(map[string]committedStore, len(commitInfo.StoreInfos))
	for _, storeInfo := range commitInfo.StoreInfos {
		if storeInfo.Name == "" || !utf8.ValidString(storeInfo.Name) || len(storeInfo.Name) > 128 {
			return rootSnapshot{}, errors.New("root commit info contains an invalid store name")
		}
		if _, exists := seenNames[storeInfo.Name]; exists {
			return rootSnapshot{}, fmt.Errorf(
				"root commit info contains duplicate store name %q",
				storeInfo.Name,
			)
		}
		seenNames[storeInfo.Name] = struct{}{}
		if len(storeInfo.CommitId.Hash) != sha256.Size {
			return rootSnapshot{}, fmt.Errorf(
				"root commit info store %q hash must be exactly 32 bytes: got %d",
				storeInfo.Name,
				len(storeInfo.CommitId.Hash),
			)
		}
		if _, isRequired := required[storeInfo.Name]; isRequired && storeInfo.CommitId.Version != height {
			return rootSnapshot{}, fmt.Errorf(
				"%s commit version mismatch: root height is %d, store is %d",
				storeInfo.Name,
				height,
				storeInfo.CommitId.Version,
			)
		}
		stores[storeInfo.Name] = committedStore{
			version:  storeInfo.CommitId.Version,
			rootHash: bytes.Clone(storeInfo.CommitId.Hash),
		}
	}
	for _, name := range requiredStoreNames {
		if _, exists := stores[name]; !exists {
			return rootSnapshot{}, fmt.Errorf(
				"root commit info must contain required store %q exactly once",
				name,
			)
		}
	}

	appHash := commitInfo.Hash()
	if len(appHash) != sha256.Size {
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
		appHash:          bytes.Clone(appHash),
		latestVersionRaw: bytes.Clone(latestRaw),
		commitInfoKey:    bytes.Clone(commitInfoKey),
		commitInfoRaw:    bytes.Clone(commitInfoRaw),
		stores:           stores,
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
		return nil, fmt.Errorf("value for key %q exceeds %d bytes", key, maxBytes)
	}
	return bytes.Clone(bz), nil
}

func scanCommittedIAVLStore(
	db physicalDB,
	height int64,
	name string,
	committed committedStore,
	visitor func(logicalLeaf) error,
	budget *scanBudget,
) (evidence storeEvidence, err error) {
	evidence = storeEvidence{
		name:       name,
		version:    committed.version,
		rootHash:   bytes.Clone(committed.rootHash),
		leavesHash: bytes.Clone(emptyIAVLRootHash[:]),
	}
	// The canonical empty IAVL root cryptographically binds the store to zero
	// leaves. An untouched store may have no physical IAVL version at all, while
	// a store emptied at H may retain older nodes, so physical loading is neither
	// necessary nor portable for this case.
	if bytes.Equal(committed.rootHash, emptyIAVLRootHash[:]) {
		return evidence, nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			evidence = storeEvidence{}
			err = errors.Join(
				err,
				fmt.Errorf("panic while reading copied %s IAVL tree: %v", name, recovered),
			)
		}
	}()

	readOnlyRoot := newReadOnlyPhysicalDB(db)
	physicalPrefix := []byte("s/k:" + name + "/")
	storeDB := dbm.NewPrefixDB(readOnlyRoot, physicalPrefix)
	tree := iavl.NewMutableTree(
		iavldb.NewWrapper(storeDB),
		0,
		true, // Never inspect, create, or rebuild IAVL fast storage.
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	defer func() {
		closeErr := tree.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close copied %s IAVL tree: %w", name, closeErr)
		}
		var writeErr error
		if attempts := readOnlyRoot.WriteAttempts(); attempts != 0 {
			writeErr = fmt.Errorf(
				"copied %s IAVL read attempted %d database mutations",
				name,
				attempts,
			)
		}
		err = errors.Join(err, closeErr, writeErr)
	}()

	latestVersion, err := tree.LoadVersion(height)
	if err != nil {
		return storeEvidence{}, fmt.Errorf(
			"load copied %s IAVL version %d without fast storage: %w",
			name,
			height,
			err,
		)
	}
	if latestVersion != height || tree.Version() != height {
		return storeEvidence{}, fmt.Errorf(
			"copied %s IAVL version mismatch: root=%d latest=%d loaded=%d",
			name,
			height,
			latestVersion,
			tree.Version(),
		)
	}
	if !bytes.Equal(tree.Hash(), committed.rootHash) {
		return storeEvidence{}, fmt.Errorf(
			"copied %s IAVL root mismatch at height %d: commit info=%s loaded tree=%s",
			name,
			height,
			hex.EncodeToString(committed.rootHash),
			hex.EncodeToString(tree.Hash()),
		)
	}
	return scanEntireIAVLStore(tree, name, committed, visitor, budget)
}

type verifiableIAVLStore interface {
	Size() int64
	Iterator(start, end []byte, ascending bool) (iavldb.Iterator, error)
	GetMembershipProof(key []byte) (*ics23.CommitmentProof, error)
}

// scanEntireIAVLStore does not trust IAVL v1.2.2 Iterator.Error for
// completeness: that iterator can silently swallow a child-load error. The
// root-bound Size and per-leaf ICS23 proof are independent completeness and
// membership checks for the ordered stream supplied to the domain visitors.
func scanEntireIAVLStore(
	store verifiableIAVLStore,
	name string,
	committed committedStore,
	visitor func(logicalLeaf) error,
	budget *scanBudget,
) (evidence storeEvidence, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			evidence = storeEvidence{}
			err = errors.Join(
				err,
				fmt.Errorf("panic while traversing copied %s IAVL tree: %v", name, recovered),
			)
		}
	}()
	if store == nil {
		return storeEvidence{}, fmt.Errorf("%s IAVL tree is nil", name)
	}
	if name == "" {
		return storeEvidence{}, errors.New("IAVL store name is empty")
	}
	if committed.version <= 0 {
		return storeEvidence{}, fmt.Errorf("%s committed version must be positive", name)
	}
	if len(committed.rootHash) != sha256.Size {
		return storeEvidence{}, fmt.Errorf("committed %s root must be exactly 32 bytes", name)
	}
	if visitor == nil {
		return storeEvidence{}, fmt.Errorf("visitor for %s IAVL tree is nil", name)
	}
	if budget == nil {
		return storeEvidence{}, errors.New("scan budget is nil")
	}

	expectedLeaves := store.Size()
	leafLimit := int64(maxStoreLeafCount)
	if name == customStakingStore {
		leafLimit = maxCustomStoreLeaves
	}
	if expectedLeaves < 0 || expectedLeaves > leafLimit {
		return storeEvidence{}, fmt.Errorf(
			"%s IAVL tree leaf count must be between 0 and %d: got %d",
			name,
			leafLimit,
			expectedLeaves,
		)
	}

	iterator, err := store.Iterator(nil, nil, true)
	if err != nil {
		return storeEvidence{}, fmt.Errorf("open complete %s IAVL iterator: %w", name, err)
	}
	if iterator == nil {
		return storeEvidence{}, fmt.Errorf("open complete %s IAVL iterator: nil iterator", name)
	}
	defer func() {
		iterationErr := iterator.Error()
		if iterationErr != nil {
			iterationErr = fmt.Errorf("iterate complete %s IAVL tree: %w", name, iterationErr)
		}
		closeErr := iterator.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close complete %s IAVL iterator: %w", name, closeErr)
		}
		err = errors.Join(err, iterationErr, closeErr)
	}()

	leavesHasher := sha256.New()
	var (
		previousKey []byte
		leafCount   int64
		inputBytes  uint64
	)
	for ; iterator.Valid(); iterator.Next() {
		if leafCount >= expectedLeaves {
			return storeEvidence{}, fmt.Errorf(
				"%s IAVL traversal emitted more than root-bound size %d",
				name,
				expectedLeaves,
			)
		}
		rawKey := iterator.Key()
		rawValue := iterator.Value()
		if len(rawKey) == 0 {
			return storeEvidence{}, fmt.Errorf("%s IAVL traversal emitted an empty key", name)
		}
		if len(rawKey) > maxLogicalKeyBytes {
			return storeEvidence{}, fmt.Errorf(
				"%s IAVL logical key exceeds %d bytes",
				name,
				maxLogicalKeyBytes,
			)
		}
		if len(rawValue) > maxLogicalValueBytes {
			return storeEvidence{}, fmt.Errorf(
				"%s IAVL value for key %q exceeds %d bytes",
				name,
				rawKey,
				maxLogicalValueBytes,
			)
		}
		leafBytes := uint64(len(rawKey)) + uint64(len(rawValue))
		if err := addScannedInputBytes(budget, leafBytes); err != nil {
			return storeEvidence{}, err
		}
		if inputBytes > maxScannedInputBytes || leafBytes > maxScannedInputBytes-inputBytes {
			return storeEvidence{}, fmt.Errorf(
				"complete %s IAVL scan exceeds %d aggregate input bytes",
				name,
				maxScannedInputBytes,
			)
		}
		inputBytes += leafBytes
		if name == customStakingStore && inputBytes > maxCustomInputBytes {
			return storeEvidence{}, fmt.Errorf(
				"complete %s IAVL scan exceeds %d retained-input bytes",
				name,
				maxCustomInputBytes,
			)
		}

		key := bytes.Clone(rawKey)
		value := bytes.Clone(rawValue)
		if previousKey != nil && bytes.Compare(previousKey, key) >= 0 {
			return storeEvidence{}, fmt.Errorf(
				"%s IAVL traversal is not in strict byte order: %q then %q",
				name,
				previousKey,
				key,
			)
		}
		previousKey = bytes.Clone(key)

		proof, err := store.GetMembershipProof(key)
		if err != nil {
			return storeEvidence{}, fmt.Errorf(
				"create %s IAVL membership proof for key %q: %w",
				name,
				key,
				err,
			)
		}
		if proofErr := verifyIAVLMembership(
			committed.rootHash,
			proof,
			key,
			value,
		); proofErr != nil {
			return storeEvidence{}, fmt.Errorf(
				"%s IAVL membership proof for key %q does not verify against committed root: %w",
				name,
				key,
				proofErr,
			)
		}

		hashLogicalLeaf(leavesHasher, key, value)
		if err := visitor(logicalLeaf{Key: key, Value: value}); err != nil {
			return storeEvidence{}, fmt.Errorf(
				"visit %s IAVL leaf %q: %w",
				name,
				key,
				err,
			)
		}
		leafCount++
	}
	if leafCount != expectedLeaves {
		return storeEvidence{}, fmt.Errorf(
			"%s IAVL traversal was incomplete: root-bound size=%d emitted=%d",
			name,
			expectedLeaves,
			leafCount,
		)
	}

	return storeEvidence{
		name:       name,
		version:    committed.version,
		rootHash:   bytes.Clone(committed.rootHash),
		leafCount:  leafCount,
		inputBytes: inputBytes,
		leavesHash: leavesHasher.Sum(nil),
	}, nil
}

// verifyIAVLMembership retains the upstream ICS23 verifier for ordinary IAVL
// leaves. IAVL also permits a non-nil, zero-byte value (used by Cosmos SDK
// index/sentinel keys), while github.com/cosmos/ics23/go v0.11.0 rejects every
// zero-byte value before hashing it. For that representable IAVL case only, we
// validate the generated proof against the exact IAVL spec and calculate the
// same SHA-256 path without the helper's non-empty-value policy check.
func verifyIAVLMembership(
	rootHash []byte,
	proof *ics23.CommitmentProof,
	key, value []byte,
) error {
	if len(value) != 0 {
		if !ics23.VerifyMembership(ics23.IavlSpec, rootHash, proof, key, value) {
			return errors.New("ICS23 verification failed")
		}
		return nil
	}

	existence := proof.GetExist()
	if existence == nil {
		return errors.New("empty-value proof is not a direct existence proof")
	}
	if err := existence.CheckAgainstSpec(ics23.IavlSpec); err != nil {
		return fmt.Errorf("empty-value proof violates the IAVL ICS23 spec: %w", err)
	}
	if !bytes.Equal(existence.Key, key) {
		return errors.New("empty-value proof key does not match the traversed key")
	}
	if len(existence.Value) != 0 {
		return errors.New("empty-value proof contains a non-empty value")
	}

	calculated := calculateEmptyValueIAVLRoot(existence)
	if !bytes.Equal(calculated, rootHash) {
		return errors.New("empty-value proof calculated a different root")
	}
	return nil
}

func calculateEmptyValueIAVLRoot(proof *ics23.ExistenceProof) []byte {
	valueHash := sha256.Sum256(nil)
	preimage := make(
		[]byte,
		0,
		len(proof.Leaf.Prefix)+binary.MaxVarintLen64+len(proof.Key)+1+sha256.Size,
	)
	preimage = append(preimage, proof.Leaf.Prefix...)
	preimage = appendUvarintBytes(preimage, proof.Key)
	preimage = appendUvarintBytes(preimage, valueHash[:])
	currentArray := sha256.Sum256(preimage)
	current := currentArray[:]

	for _, operation := range proof.Path {
		preimage = make(
			[]byte,
			0,
			len(operation.Prefix)+len(current)+len(operation.Suffix),
		)
		preimage = append(preimage, operation.Prefix...)
		preimage = append(preimage, current...)
		preimage = append(preimage, operation.Suffix...)
		next := sha256.Sum256(preimage)
		current = next[:]
	}
	return bytes.Clone(current)
}

func appendUvarintBytes(destination, value []byte) []byte {
	var length [binary.MaxVarintLen64]byte
	written := binary.PutUvarint(length[:], uint64(len(value)))
	destination = append(destination, length[:written]...)
	return append(destination, value...)
}

func hashLogicalLeaf(hasher hash.Hash, key, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(key)))
	_, _ = hasher.Write(length[:])
	_, _ = hasher.Write(key)
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = hasher.Write(length[:])
	_, _ = hasher.Write(value)
}

func addScannedInputBytes(budget *scanBudget, amount uint64) error {
	if budget.inputBytes > maxScannedInputBytes ||
		amount > maxScannedInputBytes-budget.inputBytes {
		return fmt.Errorf(
			"complete required-store IAVL scan exceeds %d aggregate input bytes",
			maxScannedInputBytes,
		)
	}
	budget.inputBytes += amount
	return nil
}
