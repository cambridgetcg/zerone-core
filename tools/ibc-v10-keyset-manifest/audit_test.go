package main

import (
	"bytes"
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

const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func TestAuditApplicationDBEmptyRegularIAVL(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(t, db, 42, nil)

	got, err := auditApplicationDB(db, expected)
	require.NoError(t, err)
	require.Equal(
		t,
		`{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"0","keys_sha256":"`+
			emptySHA256+
			`"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"`+
			emptySHA256+
			`"}}`,
		string(got),
	)
}

func TestAuditApplicationDBAcceptsCommittedEmptyWithoutVersionAndWithHistory(
	t *testing.T,
) {
	for _, testCase := range []struct {
		name        string
		withHistory bool
	}{
		{name: "no physical IAVL version"},
		{name: "historical nonempty IAVL version", withHistory: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			if testCase.withHistory {
				_ = seedRegularIAVLFixture(
					t,
					db,
					41,
					[][]byte{[]byte("channelUpgrades/historical")},
				)
			}

			const height = int64(42)
			latestRaw, err := types.StdInt64Marshal(height)
			require.NoError(t, err)
			require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))
			expected := writeFixtureCommitInfoAtHeight(
				t,
				db,
				height,
				fixtureCommitInfo(height, emptyIAVLRootHash[:]),
			)

			got, err := auditApplicationDB(db, expected)
			require.NoError(t, err)
			want, err := buildPlanInfo(nil, nil)
			require.NoError(t, err)
			require.Equal(t, string(want), string(got))
		})
	}
}

func TestAuditApplicationDBMatchesGoldensAndIgnoresOtherDomains(t *testing.T) {
	db := dbm.NewMemDB()
	channelKey := []byte("channelUpgrades/a")
	pruningKey := []byte(
		"pruningSequenceStart/ports/transfer/channels/channel-0",
	)
	expected := seedRegularIAVLFixture(t, db, 77, [][]byte{
		channelKey,
		pruningKey,
		[]byte("recvStartSequence/ports/transfer/channels/channel-0"),
		[]byte("channelUpgrades"),
	})

	got, err := auditApplicationDB(db, expected)
	require.NoError(t, err)
	expectedInfo, err := buildPlanInfo(
		[][]byte{channelKey},
		[][]byte{pruningKey},
	)
	require.NoError(t, err)
	require.Equal(t, string(expectedInfo), string(got))
	require.Contains(
		t,
		string(got),
		`"channel_upgrades":{"key_count":"1","keys_sha256":"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767"}`,
	)
	require.Len(t, channelKey, 17)
}

func TestAuditApplicationDBBindsExpectedHeightAndAppHash(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(t, db, 42, nil)

	wrongHeight := expected
	wrongHeight.Height++
	_, err := auditApplicationDB(db, wrongHeight)
	require.ErrorContains(t, err, "height mismatch")

	wrongHash := expected
	wrongHash.AppHash = hashBytes(99)
	_, err = auditApplicationDB(db, wrongHash)
	require.ErrorContains(t, err, "app hash mismatch")
}

func TestAuditApplicationDBRejectsMalformedCommitInfo(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*storetypes.CommitInfo)
		match  string
	}{
		{
			name: "wrong version",
			mutate: func(info *storetypes.CommitInfo) {
				info.Version--
			},
			match: "version mismatch",
		},
		{
			name: "missing ibc",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos = info.StoreInfos[:1]
			},
			match: "exactly one IBC store",
		},
		{
			name: "duplicate ibc",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos = append(info.StoreInfos, info.StoreInfos[1])
			},
			match: "duplicate store name",
		},
		{
			name: "wrong ibc version",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[1].CommitId.Version--
			},
			match: "IBC commit version mismatch",
		},
		{
			name: "nil hash",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[1].CommitId.Hash = nil
			},
			match: "hash must be exactly 32 bytes",
		},
		{
			name: "short hash",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[1].CommitId.Hash = []byte("short")
			},
			match: "hash must be exactly 32 bytes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			const height = int64(42)
			latestRaw, err := types.StdInt64Marshal(height)
			require.NoError(t, err)
			require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))

			info := fixtureCommitInfo(height, hashBytes(2))
			test.mutate(&info)
			expected := writeFixtureCommitInfoAtHeight(t, db, height, info)

			_, err = auditApplicationDB(db, expected)
			require.ErrorContains(t, err, test.match)
		})
	}
}

func TestAuditApplicationDBBindsLoadedIAVLRootToCommitInfo(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(
		t,
		db,
		42,
		[][]byte{[]byte("channelUpgrades/a")},
	)
	badInfo := fixtureCommitInfo(42, hashBytes(90))
	expected = writeFixtureCommitInfo(t, db, badInfo)

	_, err := auditApplicationDB(db, expected)
	require.ErrorContains(t, err, "IBC IAVL root mismatch")
}

func TestAuditApplicationDBFailsOnPhysicalIAVLReadError(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(
		t,
		db,
		42,
		[][]byte{
			[]byte("channelUpgrades/a"),
			[]byte("pruningSequenceStart/b"),
		},
	)
	readFailure := errors.New("injected IAVL node read failure")
	faulty := &getFaultPhysicalDB{
		physicalDB: db,
		keyPrefix:  []byte(ibcPhysicalStorePrefix + "s"),
		err:        readFailure,
	}

	_, err := auditApplicationDB(faulty, expected)
	require.Error(t, err)
	require.ErrorIs(t, err, readFailure)
}

func TestAuditApplicationDBRecoversFromCorruptIAVLNode(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(
		t,
		db,
		42,
		[][]byte{
			[]byte("channelUpgrades/a"),
			[]byte("pruningSequenceStart/b"),
		},
	)
	nodePrefix := []byte(ibcPhysicalStorePrefix + "s")
	iterator, err := db.Iterator(nodePrefix, testPrefixEnd(nodePrefix))
	require.NoError(t, err)
	require.True(t, iterator.Valid())
	nodeKey := bytes.Clone(iterator.Key())
	require.NoError(t, iterator.Close())
	require.NoError(t, db.Set(nodeKey, []byte("corrupt-node")))

	require.NotPanics(t, func() {
		_, err = auditApplicationDB(db, expected)
	})
	require.Error(t, err)
}

func TestScanEntireIBCStoreDetectsSilentTruncationAndIteratorErrors(t *testing.T) {
	tree, root := newProofTree(t, [][]byte{
		[]byte("channelUpgrades/a"),
		[]byte("pruningSequenceStart/b"),
	})
	iterationFailure := errors.New("injected iterator failure")
	closeFailure := errors.New("injected close failure")
	store := &iteratorFaultStore{
		verifiableIBCStore: tree,
		limit:              1,
		iterationErr:       iterationFailure,
		closeErr:           closeFailure,
	}

	_, _, err := scanEntireIBCStore(store, root, &scanBudget{})
	require.Error(t, err)
	require.ErrorContains(t, err, "incomplete")
	require.ErrorIs(t, err, iterationFailure)
	require.ErrorIs(t, err, closeFailure)
}

func TestScanEntireIBCStoreRejectsBadProofAndRecoversProofPanic(t *testing.T) {
	tree, root := newProofTree(
		t,
		[][]byte{[]byte("channelUpgrades/a")},
	)

	_, _, err := scanEntireIBCStore(tree, hashBytes(88), &scanBudget{})
	require.ErrorContains(t, err, "does not verify")

	proofFailure := errors.New("injected proof failure")
	_, _, err = scanEntireIBCStore(
		&proofErrorStore{
			verifiableIBCStore: tree,
			err:                proofFailure,
		},
		root,
		&scanBudget{},
	)
	require.ErrorIs(t, err, proofFailure)

	require.NotPanics(t, func() {
		_, _, err = scanEntireIBCStore(
			&proofPanicStore{verifiableIBCStore: tree},
			root,
			&scanBudget{},
		)
	})
	require.ErrorContains(t, err, "panic while traversing")
}

func TestScanEntireIBCStoreEnforcesWholeTreeBudgets(t *testing.T) {
	tree, root := newProofTree(
		t,
		[][]byte{[]byte("channelUpgrades/a")},
	)
	for _, size := range []int64{-1, maxIBCStoreLeafCount + 1} {
		_, _, err := scanEntireIBCStore(
			&sizeOverrideStore{
				verifiableIBCStore: tree,
				size:               size,
			},
			root,
			&scanBudget{},
		)
		require.ErrorContains(t, err, "leaf count")
	}

	budget := &scanBudget{inputBytes: maxScannedInputBytes}
	require.NoError(t, addScannedInputBytes(budget, 0))
	require.ErrorContains(t, addScannedInputBytes(budget, 1), "aggregate input bytes")

	budget.inputBytes = maxScannedInputBytes + 1
	require.ErrorContains(t, addScannedInputBytes(budget, 0), "aggregate input bytes")
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

func TestBuildPlanInfoRejectsHandlerResourceOverflow(t *testing.T) {
	tooMany := make([][]byte, maxKeysPerDomain+1)
	_, err := buildPlanInfo(tooMany, nil)
	require.ErrorContains(t, err, fmt.Sprintf("exceeds %d keys", maxKeysPerDomain))

	oversized := append(
		[]byte(channelUpgradesLogicalPrefix),
		bytes.Repeat([]byte{'x'}, maxLogicalBytesPerDomain)...,
	)
	_, err = buildPlanInfo([][]byte{oversized}, nil)
	require.ErrorContains(t, err, "aggregate logical key bytes")
}

func TestHashLengthPrefixedKeysGolden(t *testing.T) {
	key := []byte("channelUpgrades/a")
	require.Len(t, key, 17)
	require.Equal(t, emptySHA256, hex.EncodeToString(emptyIAVLRootHash[:]))
	require.Equal(
		t,
		"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767",
		hashLengthPrefixedKeys([][]byte{key}),
	)
	require.Equal(t, emptySHA256, hashLengthPrefixedKeys(nil))
}

func seedRegularIAVLFixture(
	t *testing.T,
	db dbm.DB,
	height int64,
	keys [][]byte,
) expectedEvidence {
	t.Helper()
	ibcDB := dbm.NewPrefixDB(db, []byte(ibcPhysicalStorePrefix))
	tree := iavl.NewMutableTree(
		iavldb.NewWrapper(ibcDB),
		0,
		true,
		iavl.NewNopLogger(),
		iavl.InitialVersionOption(uint64(height)),
		iavl.AsyncPruningOption(false),
	)
	for _, key := range keys {
		_, err := tree.Set(key, append([]byte("value:"), key...))
		require.NoError(t, err)
	}
	ibcHash, version, err := tree.SaveVersion()
	require.NoError(t, err)
	require.Equal(t, height, version)
	require.Len(t, ibcHash, 32)
	require.NoError(t, tree.Close())

	latestRaw, err := types.StdInt64Marshal(height)
	require.NoError(t, err)
	require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))
	return writeFixtureCommitInfo(t, db, fixtureCommitInfo(height, ibcHash))
}

func writeFixtureCommitInfo(
	t *testing.T,
	db interface{ Set(key, value []byte) error },
	info storetypes.CommitInfo,
) expectedEvidence {
	t.Helper()
	return writeFixtureCommitInfoAtHeight(t, db, info.Version, info)
}

func writeFixtureCommitInfoAtHeight(
	t *testing.T,
	db interface{ Set(key, value []byte) error },
	height int64,
	info storetypes.CommitInfo,
) expectedEvidence {
	t.Helper()
	commitRaw, err := info.Marshal()
	require.NoError(t, err)
	require.LessOrEqual(t, len(commitRaw), maxCommitInfoBytes)
	require.NoError(
		t,
		db.Set(
			[]byte(rootCommitInfoPrefix+fmt.Sprint(height)),
			commitRaw,
		),
	)
	return expectedEvidence{
		Height:  height,
		AppHash: info.Hash(),
	}
}

func fixtureCommitInfo(height int64, ibcHash []byte) storetypes.CommitInfo {
	return storetypes.CommitInfo{
		Version: height,
		StoreInfos: []storetypes.StoreInfo{
			{
				Name: "bank",
				CommitId: storetypes.CommitID{
					Version: height,
					Hash:    hashBytes(1),
				},
			},
			{
				Name: "ibc",
				CommitId: storetypes.CommitID{
					Version: height,
					Hash:    bytes.Clone(ibcHash),
				},
			},
		},
	}
}

func newProofTree(
	t *testing.T,
	keys [][]byte,
) (*iavl.MutableTree, []byte) {
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
	t.Cleanup(func() {
		require.NoError(t, tree.Close())
	})
	return tree, root
}

func hashBytes(seed byte) []byte {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = seed + byte(i)
	}
	return hash
}

type getFaultPhysicalDB struct {
	physicalDB
	keyPrefix []byte
	err       error
}

func (db *getFaultPhysicalDB) Get(key []byte) ([]byte, error) {
	if bytes.HasPrefix(key, db.keyPrefix) {
		return nil, db.err
	}
	return db.physicalDB.Get(key)
}

type iteratorFaultStore struct {
	verifiableIBCStore
	limit        int64
	iterationErr error
	closeErr     error
}

func (store *iteratorFaultStore) Iterator(
	start, end []byte,
	ascending bool,
) (iavldb.Iterator, error) {
	iterator, err := store.verifiableIBCStore.Iterator(start, end, ascending)
	if err != nil {
		return nil, err
	}
	return &faultIterator{
		Iterator:     iterator,
		limit:        store.limit,
		iterationErr: store.iterationErr,
		closeErr:     store.closeErr,
	}, nil
}

type faultIterator struct {
	iavldb.Iterator
	limit        int64
	seen         int64
	iterationErr error
	closeErr     error
}

func (iterator *faultIterator) Valid() bool {
	return iterator.seen < iterator.limit && iterator.Iterator.Valid()
}

func (iterator *faultIterator) Next() {
	iterator.seen++
	iterator.Iterator.Next()
}

func (iterator *faultIterator) Error() error {
	return errors.Join(iterator.Iterator.Error(), iterator.iterationErr)
}

func (iterator *faultIterator) Close() error {
	return errors.Join(iterator.Iterator.Close(), iterator.closeErr)
}

type proofErrorStore struct {
	verifiableIBCStore
	err error
}

func (store *proofErrorStore) GetMembershipProof(
	[]byte,
) (*ics23.CommitmentProof, error) {
	return nil, store.err
}

type proofPanicStore struct {
	verifiableIBCStore
}

func (*proofPanicStore) GetMembershipProof(
	[]byte,
) (*ics23.CommitmentProof, error) {
	panic("injected proof panic")
}

type sizeOverrideStore struct {
	verifiableIBCStore
	size int64
}

func (store *sizeOverrideStore) Size() int64 {
	return store.size
}

func testPrefixEnd(prefix []byte) []byte {
	end := bytes.Clone(prefix)
	end[len(end)-1]++
	return end
}

func TestExpectedEvidenceFormattingIsStable(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedRegularIAVLFixture(t, db, 42, nil)
	require.Len(t, expected.AppHash, 32)
	require.Len(t, hex.EncodeToString(expected.AppHash), 64)
}
