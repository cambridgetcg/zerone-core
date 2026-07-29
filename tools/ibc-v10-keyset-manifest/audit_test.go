package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/types"
	"github.com/cosmos/iavl/fastnode"
	"github.com/stretchr/testify/require"
)

const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type physicalFixtureWriter interface {
	Set(key, value []byte) error
}

func TestAuditApplicationDBEmpty(t *testing.T) {
	db := dbm.NewMemDB()
	emptyRoot := sha256.Sum256(nil)
	expected := seedPhysicalFixture(t, db, 42, nil, nil, emptyRoot[:])

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

func TestAuditApplicationDBRejectsNilStoreHash(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, nil)

	_, err := auditApplicationDB(db, expected)
	require.ErrorContains(t, err, "hash must be exactly 32 bytes")
}

func TestAuditApplicationDBMatchesUpgradeGoldenAndIgnoresOtherDomains(t *testing.T) {
	db := dbm.NewMemDB()
	channelKey := []byte("channelUpgrades/a")
	expected := seedPhysicalFixture(
		t,
		db,
		77,
		[][]byte{channelKey},
		nil,
		hashBytes(3),
	)
	setFastNode(t, db, 77, []byte("recvStartSequence/ports/transfer/channels/channel-0"))
	setFastNode(t, db, 77, []byte("channelUpgrades"))

	got, err := auditApplicationDB(db, expected)
	require.NoError(t, err)
	require.Equal(
		t,
		`{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"1","keys_sha256":"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"`+
			emptySHA256+
			`"}}`,
		string(got),
	)
	require.Len(t, channelKey, 17)
}

func TestAuditApplicationDBRejectsStaleFastStorageMetadata(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, hashBytes(2))
	require.NoError(t, db.Set(ibcStorageVersionKey, []byte("1.1.0-41")))

	_, err := auditApplicationDB(db, expected)
	require.ErrorContains(t, err, "not synchronized")
}

func TestAuditApplicationDBBindsExpectedHeightAndAppHash(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, hashBytes(2))

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
			name: "invalid hash length",
			mutate: func(info *storetypes.CommitInfo) {
				info.StoreInfos[1].CommitId.Hash = []byte("short")
			},
			match: "exactly 32 bytes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			const height = int64(42)
			latestRaw, err := types.StdInt64Marshal(height)
			require.NoError(t, err)
			require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))
			require.NoError(
				t,
				db.Set(ibcStorageVersionKey, []byte(requiredFastStorageVersion+"-42")),
			)

			info := fixtureCommitInfo(height, hashBytes(2))
			test.mutate(&info)
			raw, err := info.Marshal()
			require.NoError(t, err)
			require.NoError(t, db.Set([]byte("s/42"), raw))

			_, err = auditApplicationDB(db, expectedEvidence{
				Height:  height,
				AppHash: info.Hash(),
			})
			require.ErrorContains(t, err, test.match)
		})
	}
}

func TestAuditApplicationDBRejectsMalformedFastNodes(t *testing.T) {
	tests := []struct {
		name  string
		value func() []byte
		match string
	}{
		{
			name: "undecodable",
			value: func() []byte {
				return []byte{0xff}
			},
			match: "decode raw IBC fast node",
		},
		{
			name: "future version",
			value: func() []byte {
				return encodeFastNode(t, 43, []byte("state"))
			},
			match: "invalid last-update version",
		},
		{
			name: "noncanonical trailing bytes",
			value: func() []byte {
				return append(encodeFastNode(t, 42, []byte("state")), 0)
			},
			match: "not canonically encoded",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			key := []byte(channelUpgradesLogicalPrefix + "a")
			expected := seedPhysicalFixture(
				t,
				db,
				42,
				[][]byte{key},
				nil,
				hashBytes(2),
			)
			require.NoError(
				t,
				db.Set(append(bytes.Clone(ibcFastPhysicalPrefix), key...), test.value()),
			)

			_, err := auditApplicationDB(db, expected)
			require.ErrorContains(t, err, test.match)
		})
	}
}

func TestAuditApplicationDBPropagatesIteratorAndCloseErrors(t *testing.T) {
	mem := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, mem, 42, nil, nil, hashBytes(2))
	iterationFailure := errors.New("injected iterator failure")
	closeFailure := errors.New("injected close failure")
	db := &faultPhysicalDB{
		physicalDB: mem,
		targetStart: []byte(
			ibcPhysicalStorePrefix + iavlFastNodePrefix + channelUpgradesLogicalPrefix,
		),
		iterationErr: iterationFailure,
		closeErr:     closeFailure,
	}

	_, err := auditApplicationDB(db, expected)
	require.Error(t, err)
	require.ErrorIs(t, err, iterationFailure)
	require.ErrorIs(t, err, closeFailure)
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
	require.Equal(
		t,
		"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767",
		hashLengthPrefixedKeys([][]byte{key}),
	)
	require.Equal(t, emptySHA256, hashLengthPrefixedKeys(nil))
}

func seedPhysicalFixture(
	t *testing.T,
	db physicalFixtureWriter,
	height int64,
	channelKeys [][]byte,
	pruningKeys [][]byte,
	ibcHash []byte,
) expectedEvidence {
	t.Helper()
	latestRaw, err := types.StdInt64Marshal(height)
	require.NoError(t, err)
	require.NoError(t, db.Set([]byte(rootLatestVersionKey), latestRaw))

	info := fixtureCommitInfo(height, ibcHash)
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
	require.NoError(
		t,
		db.Set(
			ibcStorageVersionKey,
			[]byte(requiredFastStorageVersion+"-"+fmt.Sprint(height)),
		),
	)
	for _, key := range append(append([][]byte{}, channelKeys...), pruningKeys...) {
		setFastNode(t, db, height, key)
	}

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

func setFastNode(
	t *testing.T,
	db physicalFixtureWriter,
	height int64,
	logicalKey []byte,
) {
	t.Helper()
	physicalKey := append(bytes.Clone(ibcFastPhysicalPrefix), logicalKey...)
	require.NoError(t, db.Set(physicalKey, encodeFastNode(t, height, []byte("state"))))
}

func encodeFastNode(t *testing.T, version int64, value []byte) []byte {
	t.Helper()
	node := fastnode.NewNode([]byte("unused"), value, version)
	var encoded bytes.Buffer
	require.NoError(t, node.WriteBytes(&encoded))
	return encoded.Bytes()
}

func hashBytes(seed byte) []byte {
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = seed + byte(i)
	}
	return hash
}

type faultPhysicalDB struct {
	physicalDB
	targetStart  []byte
	iterationErr error
	closeErr     error
}

func (db *faultPhysicalDB) Iterator(start, end []byte) (dbm.Iterator, error) {
	iterator, err := db.physicalDB.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(start, db.targetStart) {
		return &faultIterator{
			Iterator:     iterator,
			iterationErr: db.iterationErr,
			closeErr:     db.closeErr,
		}, nil
	}
	return iterator, nil
}

type faultIterator struct {
	dbm.Iterator
	iterationErr error
	closeErr     error
}

func (iterator *faultIterator) Error() error {
	return errors.Join(iterator.Iterator.Error(), iterator.iterationErr)
}

func (iterator *faultIterator) Close() error {
	return errors.Join(iterator.Iterator.Close(), iterator.closeErr)
}

func TestExpectedEvidenceFormattingIsStable(t *testing.T) {
	db := dbm.NewMemDB()
	expected := seedPhysicalFixture(t, db, 42, nil, nil, hashBytes(2))
	require.Len(t, expected.AppHash, 32)
	require.Len(t, hex.EncodeToString(expected.AppHash), 64)
}
