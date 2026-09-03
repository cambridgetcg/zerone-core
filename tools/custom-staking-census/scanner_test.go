package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/types"
	"github.com/cosmos/iavl"
	iavldb "github.com/cosmos/iavl/db"
	ics23 "github.com/cosmos/ics23/go"
	"github.com/stretchr/testify/require"
)

func TestScanApplicationDBStreamsEveryLeafAndCommitsInventory(t *testing.T) {
	db := dbm.NewMemDB()
	fixture := map[string][]logicalLeaf{
		"zerone_staking": {
			{Key: []byte("b"), Value: []byte("value-b")},
			{Key: []byte("a"), Value: []byte("value-a")},
		},
		"bank": {
			{Key: []byte{0x02, 0x01}, Value: []byte("bank-balance")},
		},
		"staking": nil,
	}
	expected, _ := seedScannerFixture(t, db, 42, fixture)

	visited := make(map[string][]logicalLeaf, len(requiredStoreNames))
	visitors := make(map[string]func(logicalLeaf) error, len(requiredStoreNames))
	for _, storeName := range requiredStoreNames {
		name := storeName
		visitors[name] = func(leaf logicalLeaf) error {
			visited[name] = append(visited[name], logicalLeaf{
				Key:   bytes.Clone(leaf.Key),
				Value: bytes.Clone(leaf.Value),
			})
			return nil
		}
	}
	snapshot, evidence, err := scanApplicationDB(db, expected, visitors)
	require.NoError(t, err)
	require.Equal(t, expected.Height, snapshot.height)
	require.Equal(t, expected.AppHash, snapshot.appHash)
	require.Len(t, evidence, len(requiredStoreNames))

	for index, name := range requiredStoreNames {
		sortedLeaves := scannerSortedLeaves(fixture[name])
		require.Equal(t, len(sortedLeaves), len(visited[name]))
		if len(sortedLeaves) == 0 {
			require.Empty(t, visited[name])
		} else {
			require.Equal(t, sortedLeaves, visited[name])
		}
		require.Equal(t, name, evidence[index].name)
		require.Equal(t, expected.Height, evidence[index].version)
		require.Equal(t, int64(len(sortedLeaves)), evidence[index].leafCount)
		require.Equal(t, scannerInputBytes(sortedLeaves), evidence[index].inputBytes)
		require.Equal(t, scannerLeavesDigest(sortedLeaves), evidence[index].leavesHash)
		require.Equal(t, snapshot.stores[name].rootHash, evidence[index].rootHash)
	}
	require.Equal(t, emptyIAVLRootHash[:], evidence[2].rootHash)
	require.Equal(t, emptyIAVLRootHash[:], evidence[2].leavesHash)
}

func TestScanApplicationDBBindsExpectedHeightAndAppHash(t *testing.T) {
	db := dbm.NewMemDB()
	expected, _ := seedScannerFixture(t, db, 42, scannerNonemptyFixture())

	wrongHeight := expected
	wrongHeight.Height++
	_, _, err := scanApplicationDB(db, wrongHeight, scannerNoopVisitors())
	require.ErrorContains(t, err, "height mismatch")

	wrongHash := expected
	wrongHash.AppHash = scannerHashBytes(99)
	_, _, err = scanApplicationDB(db, wrongHash, scannerNoopVisitors())
	require.ErrorContains(t, err, "app hash mismatch")
}

func TestScanApplicationDBRequiresExactVisitors(t *testing.T) {
	db := dbm.NewMemDB()
	expected, _ := seedScannerFixture(t, db, 42, nil)

	visitors := scannerNoopVisitors()
	delete(visitors, "bank")
	_, _, err := scanApplicationDB(db, expected, visitors)
	require.ErrorContains(t, err, `visitor for required store "bank" is missing`)

	visitors = scannerNoopVisitors()
	visitors["ibc"] = func(logicalLeaf) error { return nil }
	_, _, err = scanApplicationDB(db, expected, visitors)
	require.ErrorContains(t, err, `unsupported store "ibc"`)
}

func TestScanApplicationDBRejectsMalformedCommitInfo(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*storetypes.CommitInfo)
		match  string
	}{
		{
			name: "wrong root version",
			mutate: func(info *storetypes.CommitInfo) {
				info.Version--
			},
			match: "version mismatch",
		},
		{
			name: "missing required store",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos = info.StoreInfos[1:]
			},
			match: `required store "zerone_staking" exactly once`,
		},
		{
			name: "duplicate store",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos = append(info.StoreInfos, info.StoreInfos[0])
			},
			match: "duplicate store name",
		},
		{
			name: "wrong required store version",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[1].CommitId.Version--
			},
			match: "commit version mismatch",
		},
		{
			name: "short hash",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[2].CommitId.Hash = []byte("short")
			},
			match: "hash must be exactly 32 bytes",
		},
		{
			name: "empty store name",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[0].Name = ""
			},
			match: "invalid store name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			_, info := seedScannerFixture(t, db, 42, nil)
			test.mutate(&info)
			var expected expectedEvidence
			if test.name == "empty store name" {
				raw, marshalErr := info.Marshal()
				require.NoError(t, marshalErr)
				require.NoError(t, db.Set([]byte(rootCommitInfoPrefix+"42"), raw))
				expected = expectedEvidence{Height: 42, AppHash: scannerHashBytes(90)}
			} else {
				expected = writeScannerCommitInfo(t, db, 42, info)
			}
			_, _, err := scanApplicationDB(db, expected, scannerNoopVisitors())
			require.ErrorContains(t, err, test.match)
		})
	}
}

func TestScanApplicationDBBindsLoadedRootsToCommitInfo(t *testing.T) {
	db := dbm.NewMemDB()
	_, info := seedScannerFixture(t, db, 42, scannerNonemptyFixture())
	info.StoreInfos[0].CommitId.Hash = scannerHashBytes(88)
	expected := writeScannerCommitInfo(t, db, 42, info)

	_, _, err := scanApplicationDB(db, expected, scannerNoopVisitors())
	require.ErrorContains(t, err, "zerone_staking IAVL root mismatch")
}

func TestScanApplicationDBDetectsRootBytesChangingDuringScan(t *testing.T) {
	db := dbm.NewMemDB()
	expected, _ := seedScannerFixture(t, db, 42, nil)
	faulty := &scannerChangingRootDB{physicalDB: db, replacementHeight: 43}

	_, _, err := scanApplicationDB(faulty, expected, scannerNoopVisitors())
	require.ErrorContains(t, err, "root latest version changed")
}

func TestScanApplicationDBChecksRootBytesAfterVisitorFailure(t *testing.T) {
	db := dbm.NewMemDB()
	expected, _ := seedScannerFixture(t, db, 42, scannerNonemptyFixture())
	faulty := &scannerChangingRootDB{physicalDB: db, replacementHeight: 43}
	visitorFailure := errors.New("injected visitor failure")
	visitors := scannerNoopVisitors()
	visitors["zerone_staking"] = func(logicalLeaf) error { return visitorFailure }

	_, _, err := scanApplicationDB(faulty, expected, visitors)
	require.ErrorIs(t, err, visitorFailure)
	require.ErrorContains(t, err, "root latest version changed")
}

func TestScanApplicationDBFailsOnPhysicalIAVLReadError(t *testing.T) {
	db := dbm.NewMemDB()
	expected, _ := seedScannerFixture(t, db, 42, scannerNonemptyFixture())
	readFailure := errors.New("injected IAVL node read failure")
	faulty := &scannerGetFaultDB{
		physicalDB: db,
		keyPrefix:  []byte("s/k:zerone_staking/s"),
		err:        readFailure,
	}

	_, _, err := scanApplicationDB(faulty, expected, scannerNoopVisitors())
	require.Error(t, err)
	require.ErrorIs(t, err, readFailure)
}

func TestScanEntireIAVLStoreDetectsTruncationOrderingAndIteratorErrors(t *testing.T) {
	tree, root := newScannerProofTree(t, [][]byte{[]byte("a"), []byte("b")})
	committed := committedStore{version: tree.Version(), rootHash: root}
	iterationFailure := errors.New("injected iterator failure")
	closeFailure := errors.New("injected close failure")
	truncated := &scannerIteratorFaultStore{
		verifiableIAVLStore: tree,
		limit:               1,
		iterationErr:        iterationFailure,
		closeErr:            closeFailure,
	}

	_, err := scanEntireIAVLStore(
		truncated,
		"zerone_staking",
		committed,
		func(logicalLeaf) error { return nil },
		&scanBudget{},
	)
	require.ErrorContains(t, err, "incomplete")
	require.ErrorIs(t, err, iterationFailure)
	require.ErrorIs(t, err, closeFailure)

	outOfOrder := &scannerIteratorOverrideStore{
		verifiableIAVLStore: tree,
		leaves: []logicalLeaf{
			{Key: []byte("b"), Value: []byte("value:b")},
			{Key: []byte("a"), Value: []byte("value:a")},
		},
	}
	_, err = scanEntireIAVLStore(
		outOfOrder,
		"zerone_staking",
		committed,
		func(logicalLeaf) error { return nil },
		&scanBudget{},
	)
	require.ErrorContains(t, err, "not in strict byte order")
}

func TestScanEntireIAVLStoreRejectsBadProofAndVisitorFailure(t *testing.T) {
	tree, root := newScannerProofTree(t, [][]byte{[]byte("a")})
	committed := committedStore{version: tree.Version(), rootHash: root}

	badRoot := committed
	badRoot.rootHash = scannerHashBytes(77)
	_, err := scanEntireIAVLStore(
		tree,
		"bank",
		badRoot,
		func(logicalLeaf) error { return nil },
		&scanBudget{},
	)
	require.ErrorContains(t, err, "does not verify")

	proofFailure := errors.New("injected proof failure")
	_, err = scanEntireIAVLStore(
		&scannerProofErrorStore{verifiableIAVLStore: tree, err: proofFailure},
		"bank",
		committed,
		func(logicalLeaf) error { return nil },
		&scanBudget{},
	)
	require.ErrorIs(t, err, proofFailure)

	visitorFailure := errors.New("injected visitor failure")
	_, err = scanEntireIAVLStore(
		tree,
		"bank",
		committed,
		func(logicalLeaf) error { return visitorFailure },
		&scanBudget{},
	)
	require.ErrorIs(t, err, visitorFailure)

	require.NotPanics(t, func() {
		_, err = scanEntireIAVLStore(
			tree,
			"bank",
			committed,
			func(logicalLeaf) error { panic("injected visitor panic") },
			&scanBudget{},
		)
	})
	require.ErrorContains(t, err, "panic while traversing")
}

func TestVerifyIAVLMembershipSupportsCommittedEmptyValue(t *testing.T) {
	tree := iavl.NewMutableTree(
		iavldb.NewMemDB(),
		0,
		true,
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	require.False(t, mustSetScannerLeaf(t, tree, []byte("empty"), []byte{}))
	require.False(t, mustSetScannerLeaf(t, tree, []byte("ordinary"), []byte("value")))
	root, _, err := tree.SaveVersion()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tree.Close()) })

	emptyProof, err := tree.GetMembershipProof([]byte("empty"))
	require.NoError(t, err)
	require.False(t, ics23.VerifyMembership(
		ics23.IavlSpec,
		root,
		emptyProof,
		[]byte("empty"),
		[]byte{},
	), "the upstream helper is known to reject representable empty IAVL values")
	require.NoError(t, verifyIAVLMembership(root, emptyProof, []byte("empty"), []byte{}))
	require.ErrorContains(
		t,
		verifyIAVLMembership(scannerHashBytes(91), emptyProof, []byte("empty"), []byte{}),
		"different root",
	)
	require.ErrorContains(
		t,
		verifyIAVLMembership(root, emptyProof, []byte("wrong"), []byte{}),
		"key does not match",
	)

	ordinaryProof, err := tree.GetMembershipProof([]byte("ordinary"))
	require.NoError(t, err)
	require.NoError(t, verifyIAVLMembership(
		root,
		ordinaryProof,
		[]byte("ordinary"),
		[]byte("value"),
	))
	require.ErrorContains(t, verifyIAVLMembership(
		root,
		ordinaryProof,
		[]byte("ordinary"),
		[]byte("tampered"),
	), "ICS23 verification failed")
}

func mustSetScannerLeaf(
	t *testing.T,
	tree *iavl.MutableTree,
	key, value []byte,
) bool {
	t.Helper()
	updated, err := tree.Set(key, value)
	require.NoError(t, err)
	return updated
}

func TestScanEntireIAVLStoreEnforcesBounds(t *testing.T) {
	tree, root := newScannerProofTree(t, [][]byte{[]byte("a")})
	committed := committedStore{version: tree.Version(), rootHash: root}
	for _, size := range []int64{-1, maxStoreLeafCount + 1} {
		_, err := scanEntireIAVLStore(
			&scannerSizeOverrideStore{verifiableIAVLStore: tree, size: size},
			"staking",
			committed,
			func(logicalLeaf) error { return nil },
			&scanBudget{},
		)
		require.ErrorContains(t, err, "leaf count")
	}
	_, err := scanEntireIAVLStore(
		&scannerSizeOverrideStore{verifiableIAVLStore: tree, size: maxCustomStoreLeaves + 1},
		customStakingStore,
		committed,
		func(logicalLeaf) error { return nil },
		&scanBudget{},
	)
	require.ErrorContains(t, err, "leaf count")

	budget := &scanBudget{inputBytes: maxScannedInputBytes}
	require.NoError(t, addScannedInputBytes(budget, 0))
	require.ErrorContains(t, addScannedInputBytes(budget, 1), "aggregate input bytes")
	budget.inputBytes = maxScannedInputBytes + 1
	require.ErrorContains(t, addScannedInputBytes(budget, 0), "aggregate input bytes")
}

func TestLogicalLeavesHashGolden(t *testing.T) {
	hasher := sha256.New()
	hashLogicalLeaf(hasher, []byte("a"), []byte("value-a"))
	require.Equal(
		t,
		"13771f68818428d5b8b32ec62782ffa064e8ab42cd1ee629656aad0e0478eed6",
		hex.EncodeToString(hasher.Sum(nil)),
	)
	require.Equal(t, emptyIAVLRootHash[:], sha256.New().Sum(nil))
}

func TestReadOnlyIAVLAdapterRejectsEveryMutationSurface(t *testing.T) {
	adapter := newReadOnlyPhysicalDB(dbm.NewMemDB())
	require.ErrorIs(t, adapter.Set([]byte("k"), []byte("v")), errReadOnlyIAVLCensus)
	require.ErrorIs(t, adapter.SetSync([]byte("k"), []byte("v")), errReadOnlyIAVLCensus)
	require.ErrorIs(t, adapter.Delete([]byte("k")), errReadOnlyIAVLCensus)
	require.ErrorIs(t, adapter.DeleteSync([]byte("k")), errReadOnlyIAVLCensus)

	batch := adapter.NewBatchWithSize(10)
	require.ErrorIs(t, batch.Set([]byte("k"), []byte("v")), errReadOnlyIAVLCensus)
	require.ErrorIs(t, batch.Delete([]byte("k")), errReadOnlyIAVLCensus)
	require.ErrorIs(t, batch.Write(), errReadOnlyIAVLCensus)
	require.ErrorIs(t, batch.WriteSync(), errReadOnlyIAVLCensus)
	require.Equal(t, int64(8), adapter.WriteAttempts())
	require.NoError(t, batch.Close())
}

func seedScannerFixture(
	t *testing.T,
	db dbm.DB,
	height int64,
	fixture map[string][]logicalLeaf,
) (expectedEvidence, storetypes.CommitInfo) {
	t.Helper()
	storeInfos := make([]storetypes.StoreInfo, 0, len(requiredStoreNames)+1)
	for _, name := range requiredStoreNames {
		rootHash := bytes.Clone(emptyIAVLRootHash[:])
		if leaves := fixture[name]; len(leaves) != 0 {
			storeDB := dbm.NewPrefixDB(db, []byte("s/k:"+name+"/"))
			tree := iavl.NewMutableTree(
				iavldb.NewWrapper(storeDB),
				0,
				true,
				iavl.NewNopLogger(),
				iavl.InitialVersionOption(uint64(height)),
				iavl.AsyncPruningOption(false),
			)
			for _, leaf := range leaves {
				_, err := tree.Set(leaf.Key, leaf.Value)
				require.NoError(t, err)
			}
			var version int64
			var err error
			rootHash, version, err = tree.SaveVersion()
			require.NoError(t, err)
			require.Equal(t, height, version)
			require.NoError(t, tree.Close())
		}
		storeInfos = append(storeInfos, storetypes.StoreInfo{
			Name: name,
			CommitId: storetypes.CommitID{
				Version: height,
				Hash:    bytes.Clone(rootHash),
			},
		})
	}
	storeInfos = append(storeInfos, storetypes.StoreInfo{
		Name: "auth",
		CommitId: storetypes.CommitID{
			Version: height,
			Hash:    scannerHashBytes(10),
		},
	})
	info := storetypes.CommitInfo{Version: height, StoreInfos: storeInfos}

	latestRaw, err := types.StdInt64Marshal(height)
	require.NoError(t, err)
	require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))
	return writeScannerCommitInfo(t, db, height, info), info
}

func writeScannerCommitInfo(
	t *testing.T,
	db interface{ Set(key, value []byte) error },
	height int64,
	info storetypes.CommitInfo,
) expectedEvidence {
	t.Helper()
	raw, err := info.Marshal()
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), maxCommitInfoBytes)
	require.NoError(t, db.Set(
		[]byte(rootCommitInfoPrefix+fmt.Sprint(height)),
		raw,
	))
	return expectedEvidence{Height: height, AppHash: info.Hash()}
}

func scannerNonemptyFixture() map[string][]logicalLeaf {
	return map[string][]logicalLeaf{
		"zerone_staking": {{Key: []byte("a"), Value: []byte("value:a")}},
		"bank":           {{Key: []byte("b"), Value: []byte("value:b")}},
		"staking":        {{Key: []byte("c"), Value: []byte("value:c")}},
	}
}

func scannerNoopVisitors() map[string]func(logicalLeaf) error {
	visitors := make(map[string]func(logicalLeaf) error, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		visitors[name] = func(logicalLeaf) error { return nil }
	}
	return visitors
}

func scannerSortedLeaves(input []logicalLeaf) []logicalLeaf {
	result := make([]logicalLeaf, len(input))
	for index, leaf := range input {
		result[index] = logicalLeaf{
			Key:   bytes.Clone(leaf.Key),
			Value: bytes.Clone(leaf.Value),
		}
	}
	// Fixtures are intentionally tiny; insertion sort keeps this helper
	// independent of scanner implementation details.
	for index := 1; index < len(result); index++ {
		for cursor := index; cursor > 0 && bytes.Compare(result[cursor-1].Key, result[cursor].Key) > 0; cursor-- {
			result[cursor-1], result[cursor] = result[cursor], result[cursor-1]
		}
	}
	return result
}

func scannerLeavesDigest(leaves []logicalLeaf) []byte {
	hasher := sha256.New()
	var length [8]byte
	for _, leaf := range leaves {
		binary.BigEndian.PutUint64(length[:], uint64(len(leaf.Key)))
		_, _ = hasher.Write(length[:])
		_, _ = hasher.Write(leaf.Key)
		binary.BigEndian.PutUint64(length[:], uint64(len(leaf.Value)))
		_, _ = hasher.Write(length[:])
		_, _ = hasher.Write(leaf.Value)
	}
	return hasher.Sum(nil)
}

func scannerInputBytes(leaves []logicalLeaf) uint64 {
	var total uint64
	for _, leaf := range leaves {
		total += uint64(len(leaf.Key) + len(leaf.Value))
	}
	return total
}

func scannerHashBytes(seed byte) []byte {
	hash := make([]byte, sha256.Size)
	for index := range hash {
		hash[index] = seed + byte(index)
	}
	return hash
}

func newScannerProofTree(t *testing.T, keys [][]byte) (*iavl.MutableTree, []byte) {
	t.Helper()
	tree := iavl.NewMutableTree(
		iavldb.NewMemDB(),
		0,
		true,
		iavl.NewNopLogger(),
		iavl.AsyncPruningOption(false),
	)
	for _, key := range keys {
		_, err := tree.Set(key, append([]byte("value:"), key...))
		require.NoError(t, err)
	}
	root, _, err := tree.SaveVersion()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tree.Close()) })
	return tree, root
}

type scannerChangingRootDB struct {
	physicalDB
	latestReads       int
	replacementHeight int64
}

func (db *scannerChangingRootDB) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, []byte(rootLatestVersionKey)) {
		db.latestReads++
		if db.latestReads > 1 {
			return types.StdInt64Marshal(db.replacementHeight)
		}
	}
	return db.physicalDB.Get(key)
}

type scannerGetFaultDB struct {
	physicalDB
	keyPrefix []byte
	err       error
}

func (db *scannerGetFaultDB) Get(key []byte) ([]byte, error) {
	if bytes.HasPrefix(key, db.keyPrefix) {
		return nil, db.err
	}
	return db.physicalDB.Get(key)
}

type scannerIteratorFaultStore struct {
	verifiableIAVLStore
	limit        int64
	iterationErr error
	closeErr     error
}

func (store *scannerIteratorFaultStore) Iterator(
	start, end []byte,
	ascending bool,
) (iavldb.Iterator, error) {
	iterator, err := store.verifiableIAVLStore.Iterator(start, end, ascending)
	if err != nil {
		return nil, err
	}
	return &scannerFaultIterator{
		Iterator:     iterator,
		limit:        store.limit,
		iterationErr: store.iterationErr,
		closeErr:     store.closeErr,
	}, nil
}

type scannerFaultIterator struct {
	iavldb.Iterator
	limit        int64
	seen         int64
	iterationErr error
	closeErr     error
}

func (iterator *scannerFaultIterator) Valid() bool {
	return iterator.seen < iterator.limit && iterator.Iterator.Valid()
}

func (iterator *scannerFaultIterator) Next() {
	iterator.seen++
	iterator.Iterator.Next()
}

func (iterator *scannerFaultIterator) Error() error {
	return errors.Join(iterator.Iterator.Error(), iterator.iterationErr)
}

func (iterator *scannerFaultIterator) Close() error {
	return errors.Join(iterator.Iterator.Close(), iterator.closeErr)
}

type scannerIteratorOverrideStore struct {
	verifiableIAVLStore
	leaves []logicalLeaf
}

func (store *scannerIteratorOverrideStore) Iterator(
	_, _ []byte,
	_ bool,
) (iavldb.Iterator, error) {
	return &scannerSliceIterator{leaves: store.leaves}, nil
}

type scannerSliceIterator struct {
	leaves []logicalLeaf
	index  int
}

func (*scannerSliceIterator) Domain() ([]byte, []byte) { return nil, nil }
func (iterator *scannerSliceIterator) Valid() bool     { return iterator.index < len(iterator.leaves) }
func (iterator *scannerSliceIterator) Next()           { iterator.index++ }
func (iterator *scannerSliceIterator) Key() []byte     { return iterator.leaves[iterator.index].Key }
func (iterator *scannerSliceIterator) Value() []byte   { return iterator.leaves[iterator.index].Value }
func (*scannerSliceIterator) Error() error             { return nil }
func (*scannerSliceIterator) Close() error             { return nil }

type scannerProofErrorStore struct {
	verifiableIAVLStore
	err error
}

func (store *scannerProofErrorStore) GetMembershipProof(
	[]byte,
) (*ics23.CommitmentProof, error) {
	return nil, store.err
}

type scannerSizeOverrideStore struct {
	verifiableIAVLStore
	size int64
}

func (store *scannerSizeOverrideStore) Size() int64 { return store.size }
