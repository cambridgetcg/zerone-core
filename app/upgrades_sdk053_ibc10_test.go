package app

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storeiavl "cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	pruningtypes "cosmossdk.io/store/pruning/types"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var errInjectedIBCCleanup = errors.New("injected IBC cleanup failure")

type failingIBCPrefixEnumerator struct {
	dbm.DB

	iteratorOpenErr  error
	iteratorErr      error
	iteratorCloseErr error
}

func (s failingIBCPrefixEnumerator) Iterator(start, end []byte) (corestore.Iterator, error) {
	if s.iteratorOpenErr != nil {
		return nil, s.iteratorOpenErr
	}
	iterator, err := s.DB.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return failingIBCCleanupIterator{
		Iterator: iterator,
		err:      s.iteratorErr,
		closeErr: s.iteratorCloseErr,
	}, nil
}

type failingIBCPrefixMutationStore struct {
	store ibcPrefixMutationStore

	deleteErr       error
	hasErr          error
	keepAfterDelete bool
	deleteCalls     *int
}

func (s failingIBCPrefixMutationStore) Delete(key []byte) error {
	if s.deleteCalls != nil {
		(*s.deleteCalls)++
	}
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if s.keepAfterDelete {
		return nil
	}
	return s.store.Delete(key)
}

func (s failingIBCPrefixMutationStore) Has(key []byte) (bool, error) {
	if s.hasErr != nil {
		return false, s.hasErr
	}
	return s.store.Has(key)
}

type failingIBCCleanupIterator struct {
	corestore.Iterator

	err      error
	closeErr error
}

func (i failingIBCCleanupIterator) Error() error {
	if i.err != nil {
		return i.err
	}
	return i.Iterator.Error()
}

func (i failingIBCCleanupIterator) Close() error {
	underlyingErr := i.Iterator.Close()
	return errors.Join(underlyingErr, i.closeErr)
}

type silentlyTruncatedIBCPrefixEnumerator struct {
	ibcPrefixEnumerator

	prefix []byte
	limit  int
}

func (s silentlyTruncatedIBCPrefixEnumerator) Iterator(start, end []byte) (corestore.Iterator, error) {
	iterator, err := s.ibcPrefixEnumerator.Iterator(start, end)
	if err != nil || string(start) != string(s.prefix) {
		return iterator, err
	}
	return &silentlyTruncatedIterator{
		Iterator:  iterator,
		remaining: s.limit,
	}, nil
}

type silentlyTruncatedIterator struct {
	corestore.Iterator

	remaining int
}

func (i *silentlyTruncatedIterator) Valid() bool {
	return i.remaining > 0 && i.Iterator.Valid()
}

func (i *silentlyTruncatedIterator) Next() {
	i.remaining--
	if i.remaining > 0 {
		i.Iterator.Next()
	}
}

type reusedKeyBufferEnumerator struct {
	keys [][]byte
}

func (s reusedKeyBufferEnumerator) Iterator(start, end []byte) (corestore.Iterator, error) {
	iterator := &reusedKeyBufferIterator{
		start: start,
		end:   end,
		keys:  s.keys,
	}
	iterator.loadKey()
	return iterator, nil
}

type reusedKeyBufferIterator struct {
	start  []byte
	end    []byte
	keys   [][]byte
	index  int
	buffer []byte
	closed bool
}

func (i *reusedKeyBufferIterator) Domain() ([]byte, []byte) {
	return i.start, i.end
}

func (i *reusedKeyBufferIterator) Valid() bool {
	return !i.closed && i.index < len(i.keys)
}

func (i *reusedKeyBufferIterator) Next() {
	i.index++
	i.loadKey()
}

func (i *reusedKeyBufferIterator) Key() []byte {
	return i.buffer
}

func (i *reusedKeyBufferIterator) Value() []byte {
	return nil
}

func (i *reusedKeyBufferIterator) Error() error {
	return nil
}

func (i *reusedKeyBufferIterator) Close() error {
	i.closed = true
	return nil
}

func (i *reusedKeyBufferIterator) loadKey() {
	if i.index < len(i.keys) {
		i.buffer = append(i.buffer[:0], i.keys[i.index]...)
	}
}

func mustSDK053IBC10Manifest(
	t *testing.T,
	channelUpgradeKeys, pruningSequenceKeys [][]byte,
) sdk053IBC10PlanInfo {
	t.Helper()
	raw, err := BuildSDK053IBC10PlanInfo(channelUpgradeKeys, pruningSequenceKeys)
	require.NoError(t, err)
	manifest, err := parseSDK053IBC10PlanInfo(raw)
	require.NoError(t, err)
	return manifest
}

func TestSDK053IBC10StoreUpgrades(t *testing.T) {
	upgrades := sdk053IBC10StoreUpgrades()

	require.Empty(t, upgrades.Added)
	require.Empty(t, upgrades.Renamed)
	require.Equal(t, []string{"capability", "feeibc"}, upgrades.Deleted)
}

func TestDeleteObsoleteIBCChannelPrefixes(t *testing.T) {
	obsolete := map[string][]byte{
		"channelUpgrades/upgrades/ports/transfer/channels/channel-0":            []byte("local-upgrade"),
		"channelUpgrades/counterpartyUpgrade/ports/transfer/channels/channel-0": []byte("counterparty-upgrade"),
		"channelUpgrades/upgradeError/ports/transfer/channels/channel-1":        []byte("upgrade-error"),
		"pruningSequenceStart/ports/transfer/channels/channel-0":                {0, 0, 0, 0, 0, 0, 0, 7},
	}
	retained := map[string][]byte{
		// recvStartSequence remains active replay-protection state in v10.
		"recvStartSequence/ports/transfer/channels/channel-0": {0, 0, 0, 0, 0, 0, 0, 8},
		// Ordinary packet state must survive the channel schema migration.
		"commitments/ports/transfer/channels/channel-0/sequences/7": []byte("packet-commitment"),
		"acks/ports/transfer/channels/channel-0/sequences/6":        []byte("packet-ack"),
		"receipts/ports/transfer/channels/channel-0/sequences/5":    {1},
		// Prefix lookalikes are not part of the obsolete domains.
		"channelUpgrades-not-a-child":      []byte("keep"),
		"pruningSequenceStart-not-a-child": []byte("keep"),
	}

	// Exercise the same split used by the upgrade handler: enumerate directly
	// from committed IAVL state, but stage every deletion in the block cache.
	rootStore := rootmulti.NewStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	ibcKey := storetypes.NewKVStoreKey("ibc-prefix-cleanup-test")
	rootStore.MountStoreWithDB(ibcKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, rootStore.LoadLatestVersion())
	committedIBCStore := rootStore.GetCommitKVStore(ibcKey)
	for key, value := range obsolete {
		committedIBCStore.Set([]byte(key), value)
	}
	for key, value := range retained {
		committedIBCStore.Set([]byte(key), value)
	}
	rootStore.Commit()

	cacheMultiStore := rootStore.CacheMultiStore()
	ctx := sdk.NewContext(cacheMultiStore, cmtproto.Header{}, false, log.NewNopLogger())
	mutationStore := sdkruntime.NewKVStoreService(ibcKey).OpenKVStore(ctx)

	channelUpgradeKeys := make([][]byte, 0, 3)
	pruningSequenceKeys := make([][]byte, 0, 1)
	for key := range obsolete {
		switch {
		case strings.HasPrefix(key, legacyIBCChannelUpgradesPrefix):
			channelUpgradeKeys = append(channelUpgradeKeys, []byte(key))
		case strings.HasPrefix(key, legacyIBCPruningSequencePrefix):
			pruningSequenceKeys = append(pruningSequenceKeys, []byte(key))
		default:
			t.Fatalf("obsolete test key %q is outside cleanup domains", key)
		}
	}
	manifest := mustSDK053IBC10Manifest(t, channelUpgradeKeys, pruningSequenceKeys)
	deleted, err := deleteObsoleteIBCChannelPrefixes(
		committedIBCPrefixEnumerator{store: committedIBCStore},
		mutationStore,
		manifest,
	)
	require.NoError(t, err)
	require.Equal(t, len(obsolete), deleted)

	for key := range obsolete {
		value, err := mutationStore.Get([]byte(key))
		require.NoError(t, err)
		require.Nil(t, value, "%s must be removed", key)
		require.Equal(
			t,
			obsolete[key],
			committedIBCStore.Get([]byte(key)),
			"%s must remain in committed state until the cache writes",
			key,
		)
	}
	for key, expected := range retained {
		value, err := mutationStore.Get([]byte(key))
		require.NoError(t, err)
		require.Equal(t, expected, value, "%s must be preserved byte-for-byte", key)
		require.Equal(t, expected, committedIBCStore.Get([]byte(key)))
	}

	cacheMultiStore.Write()
	rootStore.Commit()
	for key := range obsolete {
		require.Nil(t, committedIBCStore.Get([]byte(key)), "%s must be absent after commit", key)
	}
	for key, expected := range retained {
		require.Equal(t, expected, committedIBCStore.Get([]byte(key)))
	}

	// Re-running against the newly committed state is a no-op, which matters
	// if an operator retries the repair after a completed rehearsal.
	replayCache := rootStore.CacheMultiStore()
	replayCtx := sdk.NewContext(replayCache, cmtproto.Header{}, false, log.NewNopLogger())
	replayMutationStore := sdkruntime.NewKVStoreService(ibcKey).OpenKVStore(replayCtx)
	deleted, err = deleteObsoleteIBCChannelPrefixes(
		committedIBCPrefixEnumerator{store: committedIBCStore},
		replayMutationStore,
		mustSDK053IBC10Manifest(t, nil, nil),
	)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func TestDeleteObsoleteIBCChannelPrefixesFailsClosed(t *testing.T) {
	testCases := []struct {
		name             string
		iteratorOpenErr  error
		iteratorErr      error
		iteratorCloseErr error
		deleteErr        error
		hasErr           error
		keepAfterDelete  bool
		want             string
	}{
		{
			name:            "iterator open",
			iteratorOpenErr: errInjectedIBCCleanup,
			want:            "open iterator",
		},
		{
			name:        "iterator traversal",
			iteratorErr: errInjectedIBCCleanup,
			want:        "iterate obsolete IBC prefix",
		},
		{
			name:             "iterator close",
			iteratorCloseErr: errInjectedIBCCleanup,
			want:             "close iterator",
		},
		{
			name:      "delete",
			deleteErr: errInjectedIBCCleanup,
			want:      "delete obsolete IBC key",
		},
		{
			name:   "post-delete lookup",
			hasErr: errInjectedIBCCleanup,
			want:   "verify obsolete IBC key",
		},
		{
			name:            "delete has no effect",
			keepAfterDelete: true,
			want:            "key remains in upgrade cache",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := []byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0")
			enumerationDB := dbm.NewMemDB()
			mutationDB := dbm.NewMemDB()
			require.NoError(t, enumerationDB.Set(key, []byte("upgrade")))
			require.NoError(t, mutationDB.Set(key, []byte("upgrade")))

			enumerator := failingIBCPrefixEnumerator{
				DB:               enumerationDB,
				iteratorOpenErr:  tc.iteratorOpenErr,
				iteratorErr:      tc.iteratorErr,
				iteratorCloseErr: tc.iteratorCloseErr,
			}
			mutationStore := failingIBCPrefixMutationStore{
				store:           mutationDB,
				deleteErr:       tc.deleteErr,
				hasErr:          tc.hasErr,
				keepAfterDelete: tc.keepAfterDelete,
			}

			deleted, err := deleteObsoleteIBCChannelPrefixes(
				enumerator,
				mutationStore,
				mustSDK053IBC10Manifest(t, [][]byte{key}, nil),
			)
			require.Error(t, err)
			if tc.keepAfterDelete {
				require.NotErrorIs(t, err, errInjectedIBCCleanup)
			} else {
				require.ErrorIs(t, err, errInjectedIBCCleanup)
			}
			require.Contains(t, err.Error(), tc.want)
			require.Zero(t, deleted)
		})
	}
}

func TestDeleteObsoleteIBCChannelPrefixesFailureDoesNotMutateCommittedStore(t *testing.T) {
	rootStore := rootmulti.NewStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	ibcKey := storetypes.NewKVStoreKey("ibc-prefix-cleanup-rollback-test")
	rootStore.MountStoreWithDB(ibcKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, rootStore.LoadLatestVersion())

	key := []byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0")
	value := []byte("upgrade")
	committedIBCStore := rootStore.GetCommitKVStore(ibcKey)
	committedIBCStore.Set(key, value)
	rootStore.Commit()

	cacheMultiStore := rootStore.CacheMultiStore()
	ctx := sdk.NewContext(cacheMultiStore, cmtproto.Header{}, false, log.NewNopLogger())
	cachedStore := sdkruntime.NewKVStoreService(ibcKey).OpenKVStore(ctx)
	failingMutationStore := failingIBCPrefixMutationStore{
		store:  cachedStore,
		hasErr: errInjectedIBCCleanup,
	}

	deleted, err := deleteObsoleteIBCChannelPrefixes(
		committedIBCPrefixEnumerator{store: committedIBCStore},
		failingMutationStore,
		mustSDK053IBC10Manifest(t, [][]byte{key}, nil),
	)
	require.ErrorIs(t, err, errInjectedIBCCleanup)
	require.Zero(t, deleted)

	cachedValue, getErr := cachedStore.Get(key)
	require.NoError(t, getErr)
	require.Nil(t, cachedValue, "the failed repair may contain staged cache writes")
	require.Equal(
		t,
		value,
		committedIBCStore.Get(key),
		"discarding the failed upgrade cache must leave committed state intact",
	)
}

func TestDeleteObsoleteIBCChannelPrefixesRejectsSilentPartialEnumeration(t *testing.T) {
	firstKey := []byte("channelUpgrades/counterpartyUpgrade/ports/transfer/channels/channel-0")
	secondKey := []byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0")
	enumerationDB := dbm.NewMemDB()
	mutationDB := dbm.NewMemDB()
	for _, key := range [][]byte{firstKey, secondKey} {
		require.NoError(t, enumerationDB.Set(key, []byte("upgrade")))
		require.NoError(t, mutationDB.Set(key, []byte("upgrade")))
	}

	enumerator := silentlyTruncatedIBCPrefixEnumerator{
		ibcPrefixEnumerator: failingIBCPrefixEnumerator{DB: enumerationDB},
		prefix:              []byte(legacyIBCChannelUpgradesPrefix),
		limit:               1,
	}
	deleteCalls := 0
	deleted, err := deleteObsoleteIBCChannelPrefixes(
		enumerator,
		failingIBCPrefixMutationStore{store: mutationDB, deleteCalls: &deleteCalls},
		mustSDK053IBC10Manifest(t, [][]byte{firstKey, secondKey}, nil),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match plan manifest")
	require.Zero(t, deleted)
	require.Zero(t, deleteCalls)
	for _, key := range [][]byte{firstKey, secondKey} {
		has, hasErr := mutationDB.Has(key)
		require.NoError(t, hasErr)
		require.True(t, has, "%q must remain because completeness was not proven", key)
	}
}

func TestDeleteObsoleteIBCChannelPrefixesChecksBothManifestsBeforeDeleting(t *testing.T) {
	channelKey := []byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0")
	pruningKey := []byte("pruningSequenceStart/ports/transfer/channels/channel-0")
	enumerationDB := dbm.NewMemDB()
	mutationDB := dbm.NewMemDB()
	for _, key := range [][]byte{channelKey, pruningKey} {
		require.NoError(t, enumerationDB.Set(key, []byte("state")))
		require.NoError(t, mutationDB.Set(key, []byte("state")))
	}

	// The channel commitment matches, but the pruning commitment deliberately
	// claims an empty set. No channel key may be deleted before the second
	// domain has also matched.
	deleteCalls := 0
	deleted, err := deleteObsoleteIBCChannelPrefixes(
		failingIBCPrefixEnumerator{DB: enumerationDB},
		failingIBCPrefixMutationStore{store: mutationDB, deleteCalls: &deleteCalls},
		mustSDK053IBC10Manifest(t, [][]byte{channelKey}, nil),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), legacyIBCPruningSequencePrefix)
	require.Zero(t, deleted)
	require.Zero(t, deleteCalls)
	for _, key := range [][]byte{channelKey, pruningKey} {
		has, hasErr := mutationDB.Has(key)
		require.NoError(t, hasErr)
		require.True(t, has, "%q must remain until both manifests match", key)
	}
}

func TestDeleteObsoleteIBCChannelPrefixesRejectsSameCountKeySubstitution(t *testing.T) {
	expectedKeys := [][]byte{
		[]byte("channelUpgrades/counterpartyUpgrade/ports/transfer/channels/channel-0"),
		[]byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0"),
	}
	actualKeys := [][]byte{
		expectedKeys[0],
		[]byte("channelUpgrades/upgrades/ports/transfer/channels/channel-9"),
	}
	enumerationDB := dbm.NewMemDB()
	mutationDB := dbm.NewMemDB()
	for _, key := range actualKeys {
		require.NoError(t, enumerationDB.Set(key, []byte("upgrade")))
		require.NoError(t, mutationDB.Set(key, []byte("upgrade")))
	}

	deleteCalls := 0
	deleted, err := deleteObsoleteIBCChannelPrefixes(
		failingIBCPrefixEnumerator{DB: enumerationDB},
		failingIBCPrefixMutationStore{store: mutationDB, deleteCalls: &deleteCalls},
		mustSDK053IBC10Manifest(t, expectedKeys, nil),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match plan manifest")
	require.Zero(t, deleted)
	require.Zero(t, deleteCalls, "equal counts must not bypass the keyset digest")
}

func TestCollectStoreKeysWithPrefixClonesReusedIteratorKeyBuffer(t *testing.T) {
	keys := [][]byte{
		[]byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0"),
		[]byte("channelUpgrades/upgrades/ports/transfer/channels/channel-1"),
	}
	collected, err := collectStoreKeysWithPrefix(
		reusedKeyBufferEnumerator{keys: keys},
		[]byte(legacyIBCChannelUpgradesPrefix),
		uint64(len(keys)),
	)
	require.NoError(t, err)
	require.Equal(t, keys, collected)
	require.NotSame(t, &collected[0][0], &collected[1][0])
}

func TestCollectStoreKeysWithPrefixReturnsTraversalAndCloseErrors(t *testing.T) {
	traversalErr := errors.New("traversal failed")
	closeErr := errors.New("close failed")
	_, err := collectStoreKeysWithPrefix(
		failingIBCPrefixEnumerator{
			DB:               dbm.NewMemDB(),
			iteratorErr:      traversalErr,
			iteratorCloseErr: closeErr,
		},
		[]byte(legacyIBCChannelUpgradesPrefix),
		0,
	)
	require.ErrorIs(t, err, traversalErr)
	require.ErrorIs(t, err, closeErr)
}

func TestSDK053IBC10PlanInfoIsStrictAndCanonical(t *testing.T) {
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	const emptyInfo = `{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}}`

	built, err := BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	require.Equal(t, emptyInfo, built)
	manifest, err := parseSDK053IBC10PlanInfo(built)
	require.NoError(t, err)
	require.Equal(t, "0", manifest.ChannelUpgrades.KeyCount)
	require.Equal(t, emptySHA256, manifest.ChannelUpgrades.KeysSHA256)

	nonEmpty, err := BuildSDK053IBC10PlanInfo(
		[][]byte{[]byte("channelUpgrades/a")},
		nil,
	)
	require.NoError(t, err)
	nonEmptyManifest, err := parseSDK053IBC10PlanInfo(nonEmpty)
	require.NoError(t, err)
	require.Equal(t, "1", nonEmptyManifest.ChannelUpgrades.KeyCount)
	require.Equal(
		t,
		"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767",
		nonEmptyManifest.ChannelUpgrades.KeysSHA256,
	)
	sortedInfo, err := BuildSDK053IBC10PlanInfo(
		[][]byte{[]byte("channelUpgrades/a"), []byte("channelUpgrades/b")},
		nil,
	)
	require.NoError(t, err)
	reversedInfo, err := BuildSDK053IBC10PlanInfo(
		[][]byte{[]byte("channelUpgrades/b"), []byte("channelUpgrades/a")},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, sortedInfo, reversedInfo, "input order must not affect the canonical keyset")

	duplicateSchema := strings.Replace(
		built,
		`{"schema":"`+SDK053IBC10PlanInfoSchema+`",`,
		`{"schema":"`+SDK053IBC10PlanInfoSchema+`","schema":"`+SDK053IBC10PlanInfoSchema+`",`,
		1,
	)
	testCases := []struct {
		name string
		raw  string
	}{
		{name: "missing", raw: ""},
		{name: "whitespace", raw: " " + built},
		{name: "duplicate field", raw: duplicateSchema},
		{name: "unknown field", raw: strings.TrimSuffix(built, "}") + `,"unknown":true}`},
		{name: "missing field", raw: strings.Replace(built, `"key_count":"0",`, "", 1)},
		{name: "trailing value", raw: built + `{}`},
		{name: "uppercase digest", raw: strings.Replace(built, emptySHA256, strings.ToUpper(emptySHA256), 1)},
		{name: "noncanonical count", raw: strings.Replace(built, `"key_count":"0"`, `"key_count":"00"`, 1)},
		{name: "excessive count", raw: strings.Replace(built, `"key_count":"0"`, `"key_count":"100001"`, 1)},
		{name: "oversize", raw: strings.Repeat("x", maxSDK053IBC10PlanInfoByteCount+1)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSDK053IBC10PlanInfo(tc.raw)
			require.Error(t, err)
		})
	}

	_, err = BuildSDK053IBC10PlanInfo(
		[][]byte{[]byte(legacyIBCChannelUpgradesPrefix + "duplicate"), []byte(legacyIBCChannelUpgradesPrefix + "duplicate")},
		nil,
	)
	require.ErrorContains(t, err, "duplicate")
	_, err = BuildSDK053IBC10PlanInfo([][]byte{[]byte("recvStartSequence/not-obsolete")}, nil)
	require.ErrorContains(t, err, "outside prefix")
}

func TestSDK053IBC10StoreLoaderAllowsUnrelatedFeeStateAndRestarts(t *testing.T) {
	db := dbm.NewMemDB()
	pruning := pruningtypes.NewPruningOptions(pruningtypes.PruningNothing)
	key := []byte("legacy-state")
	value := []byte("must-be-deleted")

	oldStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	oldStore.SetPruning(pruning)
	for _, name := range []string{"retained", legacyCapabilityStoreKey, legacyIBCFeeStoreKey} {
		oldStore.MountStoreWithDB(storetypes.NewKVStoreKey(name), storetypes.StoreTypeIAVL, nil)
	}
	require.NoError(t, oldStore.LoadLatestVersion())
	for _, name := range []string{legacyCapabilityStoreKey, legacyIBCFeeStoreKey} {
		store := oldStore.GetStoreByName(name).(storetypes.KVStore)
		store.Set(key, value)
	}
	retained := oldStore.GetStoreByName("retained").(storetypes.KVStore)
	retained.Set(key, []byte("kept"))
	require.Equal(t, int64(1), oldStore.Commit().Version)

	upgradedStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	upgradedStore.SetPruning(pruning)
	upgradedStore.MountStoreWithDB(storetypes.NewKVStoreKey("retained"), storetypes.StoreTypeIAVL, nil)
	require.NoError(t, sdk053IBC10StoreLoader(2)(upgradedStore))

	for _, name := range []string{legacyCapabilityStoreKey, legacyIBCFeeStoreKey} {
		store := upgradedStore.GetStoreByName(name).(storetypes.KVStore)
		require.Nil(t, store.Get(key), "%s data must be deleted before the upgrade commit", name)
	}
	require.Equal(t, []byte("kept"), upgradedStore.GetStoreByName("retained").(storetypes.KVStore).Get(key))
	require.Equal(t, int64(2), upgradedStore.Commit().Version)

	// A post-upgrade restart mounts only the live store set. This proves the
	// deleted keys are absent from commit info rather than merely emptied.
	restartedStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	restartedStore.SetPruning(pruning)
	restartedStore.MountStoreWithDB(storetypes.NewKVStoreKey("retained"), storetypes.StoreTypeIAVL, nil)
	require.NoError(t, restartedStore.LoadLatestVersion())
	commitInfo, err := restartedStore.GetCommitInfo(2)
	require.NoError(t, err)
	require.Len(t, commitInfo.StoreInfos, 1)
	require.Equal(t, "retained", commitInfo.StoreInfos[0].Name)
}

func TestSDK053IBC10HandlerRejectsOfflineProofOutsideDryRunContext(t *testing.T) {
	application := &ZeroneApp{
		sdk053IBC10LoaderProof: &sdk053IBC10StoreLoaderProof{
			upgradeHeight:       2,
			preUpgradeVersion:   1,
			legacyRootsComplete: true,
			feeLockAbsent:       true,
			preflightOnly:       true,
		},
	}
	plan := upgradetypes.Plan{Name: UpgradeNameSDK053IBC10, Height: 2}
	err := application.requireSDK053IBC10LoaderProof(
		context.Background(),
		plan,
	)
	require.ErrorContains(t, err, "refuses an offline preflight proof")
	require.NoError(
		t,
		application.requireSDK053IBC10LoaderProof(
			context.WithValue(
				context.Background(),
				sdk053IBC10PreflightDryRunContextKey{},
				true,
			),
			plan,
		),
	)
}

func TestSDK053IBC10StoreLoaderRejectsLegacyFeeLockAndPreservesOldDatabase(t *testing.T) {
	for _, disableFastNode := range []bool{false, true} {
		for _, lockedValue := range [][]byte{{0x01}, {0x02}} {
			name := "fastnode-enabled"
			if disableFastNode {
				name = "fastnode-disabled"
			}
			name += "/value-" + hex.EncodeToString(lockedValue)

			t.Run(name, func(t *testing.T) {
				db := dbm.NewMemDB()
				pruning := pruningtypes.NewPruningOptions(pruningtypes.PruningNothing)
				legacyKey := []byte("legacy-state")
				legacyValue := []byte("must-survive-refusal")

				oldStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
				oldStore.SetPruning(pruning)
				oldStore.SetIAVLDisableFastNode(disableFastNode)
				for _, storeName := range []string{
					"retained",
					legacyCapabilityStoreKey,
					legacyIBCFeeStoreKey,
				} {
					oldStore.MountStoreWithDB(
						storetypes.NewKVStoreKey(storeName),
						storetypes.StoreTypeIAVL,
						nil,
					)
				}
				require.NoError(t, oldStore.LoadLatestVersion())
				oldStore.GetStoreByName("retained").(storetypes.KVStore).Set(
					legacyKey,
					[]byte("kept"),
				)
				oldStore.GetStoreByName(legacyCapabilityStoreKey).(storetypes.KVStore).Set(
					legacyKey,
					legacyValue,
				)
				legacyFeeStore := oldStore.GetStoreByName(legacyIBCFeeStoreKey).(storetypes.KVStore)
				legacyFeeStore.Set([]byte(legacyIBCFeeLockedKey), lockedValue)
				legacyFeeStore.Set(legacyKey, legacyValue)
				oldCommitID := oldStore.Commit()
				require.Equal(t, int64(1), oldCommitID.Version)

				upgradedStore := rootmulti.NewStore(
					db,
					log.NewNopLogger(),
					metrics.NewNoOpMetrics(),
				)
				upgradedStore.SetPruning(pruning)
				upgradedStore.SetIAVLDisableFastNode(disableFastNode)
				upgradedStore.MountStoreWithDB(
					storetypes.NewKVStoreKey("retained"),
					storetypes.StoreTypeIAVL,
					nil,
				)

				err := sdk053IBC10StoreLoader(2)(upgradedStore)
				require.ErrorContains(t, err, "legacy feeibc/locked is present at version 1")
				require.ErrorContains(t, err, "refusing sdk-0.53-ibc-10 upgrade")

				// LoadLatestVersionAndUpgrade has staged deletion in the
				// current mutable tree. The guard must nevertheless see the
				// canonical immutable H-1 tree, independent of fast nodes.
				stagedFeeStore := upgradedStore.GetStoreByName(
					legacyIBCFeeStoreKey,
				).(storetypes.KVStore)
				require.False(t, stagedFeeStore.Has([]byte(legacyIBCFeeLockedKey)))
				require.Nil(t, stagedFeeStore.Get(legacyKey))

				feeKey := upgradedStore.StoreKeysByName()[legacyIBCFeeStoreKey]
				feeIAVLStore, ok := upgradedStore.GetCommitKVStore(feeKey).(*storeiavl.Store)
				require.True(t, ok)
				immutableFeeStore, err := feeIAVLStore.GetImmutable(1)
				require.NoError(t, err)
				require.True(t, immutableFeeStore.Has([]byte(legacyIBCFeeLockedKey)))
				require.Equal(t, lockedValue, immutableFeeStore.Get([]byte(legacyIBCFeeLockedKey)))
				require.Equal(t, legacyValue, immutableFeeStore.Get(legacyKey))

				// The failed startup must not persist the staged deletions.
				// Reopening with the old v8 mount set recovers the exact H-1
				// state and commit ID.
				restartedOldStore := rootmulti.NewStore(
					db,
					log.NewNopLogger(),
					metrics.NewNoOpMetrics(),
				)
				restartedOldStore.SetPruning(pruning)
				restartedOldStore.SetIAVLDisableFastNode(disableFastNode)
				for _, storeName := range []string{
					"retained",
					legacyCapabilityStoreKey,
					legacyIBCFeeStoreKey,
				} {
					restartedOldStore.MountStoreWithDB(
						storetypes.NewKVStoreKey(storeName),
						storetypes.StoreTypeIAVL,
						nil,
					)
				}
				require.NoError(t, restartedOldStore.LoadLatestVersion())
				require.Equal(t, oldCommitID, restartedOldStore.LastCommitID())
				require.Equal(
					t,
					lockedValue,
					restartedOldStore.GetStoreByName(
						legacyIBCFeeStoreKey,
					).(storetypes.KVStore).Get([]byte(legacyIBCFeeLockedKey)),
				)
				require.Equal(
					t,
					legacyValue,
					restartedOldStore.GetStoreByName(
						legacyIBCFeeStoreKey,
					).(storetypes.KVStore).Get(legacyKey),
				)
				require.Equal(
					t,
					legacyValue,
					restartedOldStore.GetStoreByName(
						legacyCapabilityStoreKey,
					).(storetypes.KVStore).Get(legacyKey),
				)
			})
		}
	}
}

func TestSDK053IBC10StoreLoaderAllowsEmptyLegacyFeeStore(t *testing.T) {
	for _, disableFastNode := range []bool{false, true} {
		name := "fastnode-enabled"
		if disableFastNode {
			name = "fastnode-disabled"
		}

		t.Run(name, func(t *testing.T) {
			db := dbm.NewMemDB()
			pruning := pruningtypes.NewPruningOptions(pruningtypes.PruningNothing)

			oldStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
			oldStore.SetPruning(pruning)
			oldStore.SetIAVLDisableFastNode(disableFastNode)
			for _, storeName := range []string{
				"retained",
				legacyCapabilityStoreKey,
				legacyIBCFeeStoreKey,
			} {
				oldStore.MountStoreWithDB(
					storetypes.NewKVStoreKey(storeName),
					storetypes.StoreTypeIAVL,
					nil,
				)
			}
			require.NoError(t, oldStore.LoadLatestVersion())
			oldStore.GetStoreByName("retained").(storetypes.KVStore).Set(
				[]byte("retained"),
				[]byte("kept"),
			)
			require.Equal(t, int64(1), oldStore.Commit().Version)

			upgradedStore := rootmulti.NewStore(
				db,
				log.NewNopLogger(),
				metrics.NewNoOpMetrics(),
			)
			upgradedStore.SetPruning(pruning)
			upgradedStore.SetIAVLDisableFastNode(disableFastNode)
			upgradedStore.MountStoreWithDB(
				storetypes.NewKVStoreKey("retained"),
				storetypes.StoreTypeIAVL,
				nil,
			)

			require.NoError(t, sdk053IBC10StoreLoader(2)(upgradedStore))
			require.False(
				t,
				upgradedStore.GetStoreByName(
					legacyIBCFeeStoreKey,
				).(storetypes.KVStore).Has([]byte(legacyIBCFeeLockedKey)),
			)
			require.Equal(t, int64(2), upgradedStore.Commit().Version)

			restartedStore := rootmulti.NewStore(
				db,
				log.NewNopLogger(),
				metrics.NewNoOpMetrics(),
			)
			restartedStore.SetPruning(pruning)
			restartedStore.SetIAVLDisableFastNode(disableFastNode)
			restartedStore.MountStoreWithDB(
				storetypes.NewKVStoreKey("retained"),
				storetypes.StoreTypeIAVL,
				nil,
			)
			require.NoError(t, restartedStore.LoadLatestVersion())
			commitInfo, err := restartedStore.GetCommitInfo(2)
			require.NoError(t, err)
			require.Len(t, commitInfo.StoreInfos, 1)
			require.Equal(t, "retained", commitInfo.StoreInfos[0].Name)
		})
	}
}

func TestRejectLegacyIBCFeeLockFailsClosed(t *testing.T) {
	err := rejectLegacyIBCFeeLock(nil, 1)
	require.ErrorContains(t, err, "unexpected commit store type <nil>")

	var nilIAVLStore *storeiavl.Store
	err = rejectLegacyIBCFeeLock(nilIAVLStore, 1)
	require.ErrorContains(t, err, "recovered")
	require.ErrorContains(t, err, "panic")

	db := dbm.NewMemDB()
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	feeKey := storetypes.NewKVStoreKey(legacyIBCFeeStoreKey)
	store.MountStoreWithDB(feeKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, store.LoadLatestVersion())
	store.GetKVStore(feeKey).Set([]byte("unrelated"), []byte("value"))
	require.Equal(t, int64(1), store.Commit().Version)

	err = rejectLegacyIBCFeeLock(store.GetCommitKVStore(feeKey), 2)
	require.ErrorContains(t, err, "open legacy feeibc store at immutable version 2")
	require.ErrorContains(t, err, "version mismatch")
}

func TestRegisterStoreUpgradesReturnsMalformedUpgradeInfoError(t *testing.T) {
	app := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	)

	path, err := app.UpgradeKeeper.GetUpgradeInfoPath()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("{not-json"), 0o600))

	err = app.RegisterStoreUpgrades()
	require.Error(t, err)
	require.Contains(t, err.Error(), "read upgrade info from disk")
}

func TestRegisterStoreUpgradesHonorsUnsafeSkipHeight(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data"), 0o700))
	planBytes, err := json.Marshal(upgradetypes.Plan{
		Name:   UpgradeNameSDK053IBC10,
		Height: 2,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename),
		planBytes,
		0o600,
	))

	app := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.AppOptionsMap{
			flags.FlagHome:                home,
			server.FlagUnsafeSkipUpgrades: []int{2},
		},
	)
	require.True(t, app.UpgradeKeeper.IsSkipHeight(2))
	require.NoError(t, app.RegisterStoreUpgrades())
}

func TestRegisterStoreUpgradesRequiresExactLocalPlanForCommittedLegacyRoots(
	t *testing.T,
) {
	tests := []struct {
		name      string
		writePlan *upgradetypes.Plan
		wantError string
	}{
		{
			name:      "missing",
			wantError: "require local upgrade-info.json",
		},
		{
			name: "wrong name",
			writePlan: &upgradetypes.Plan{
				Name:   UpgradeNameTestnet,
				Height: 2,
			},
			wantError: "require local upgrade-info.json",
		},
		{
			name: "wrong height",
			writePlan: &upgradetypes.Plan{
				Name:   UpgradeNameSDK053IBC10,
				Height: 3,
			},
			wantError: "exact height 2",
		},
		{
			name: "exact",
			writePlan: &upgradetypes.Plan{
				Name:   UpgradeNameSDK053IBC10,
				Height: 2,
				Info:   "exact-info",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, _ := commitLegacySDK053IBC10Roots(t)
			home := t.TempDir()
			require.NoError(t, os.MkdirAll(
				filepath.Join(home, "data"),
				0o700,
			))
			if test.writePlan != nil {
				raw, err := json.Marshal(test.writePlan)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(
					filepath.Join(
						home,
						"data",
						upgradetypes.UpgradeInfoFilename,
					),
					raw,
					0o600,
				))
			}
			application := NewActivationPreflightApp(
				log.NewNopLogger(),
				db,
				nil,
				false,
				simtestutil.AppOptionsMap{flags.FlagHome: home},
			)
			err := application.RegisterStoreUpgrades()
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
			}
		})
	}
}

func TestRegisterStoreUpgradesRejectsSkipWithLegacyRootsAndPreservesCommit(
	t *testing.T,
) {
	db, oldCommitID := commitLegacySDK053IBC10Roots(t)
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data"), 0o700))
	raw, err := json.Marshal(upgradetypes.Plan{
		Name:   UpgradeNameSDK053IBC10,
		Height: 2,
		Info:   "exact-info",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename),
		raw,
		0o600,
	))
	application := NewActivationPreflightApp(
		log.NewNopLogger(),
		db,
		nil,
		false,
		simtestutil.AppOptionsMap{
			flags.FlagHome:                home,
			server.FlagUnsafeSkipUpgrades: []int{2},
		},
	)
	err = application.RegisterStoreUpgrades()
	require.ErrorContains(t, err, "refusing --unsafe-skip-upgrades 2")

	reopened := rootmulti.NewStore(
		db,
		log.NewNopLogger(),
		metrics.NewNoOpMetrics(),
	)
	for _, name := range []string{
		"retained",
		legacyCapabilityStoreKey,
		legacyIBCFeeStoreKey,
	} {
		reopened.MountStoreWithDB(
			storetypes.NewKVStoreKey(name),
			storetypes.StoreTypeIAVL,
			nil,
		)
	}
	require.NoError(t, reopened.LoadLatestVersion())
	require.Equal(t, oldCommitID, reopened.LastCommitID())
	for _, name := range []string{
		legacyCapabilityStoreKey,
		legacyIBCFeeStoreKey,
	} {
		require.Equal(
			t,
			[]byte("preserved"),
			reopened.GetStoreByName(name).(storetypes.KVStore).Get(
				[]byte("legacy"),
			),
		)
	}
}

func commitLegacySDK053IBC10Roots(
	t *testing.T,
) (dbm.DB, storetypes.CommitID) {
	t.Helper()
	db := dbm.NewMemDB()
	store := rootmulti.NewStore(
		db,
		log.NewNopLogger(),
		metrics.NewNoOpMetrics(),
	)
	for _, name := range []string{
		"retained",
		legacyCapabilityStoreKey,
		legacyIBCFeeStoreKey,
	} {
		store.MountStoreWithDB(
			storetypes.NewKVStoreKey(name),
			storetypes.StoreTypeIAVL,
			nil,
		)
	}
	require.NoError(t, store.LoadLatestVersion())
	store.GetStoreByName("retained").(storetypes.KVStore).Set(
		[]byte("retained"),
		[]byte("preserved"),
	)
	for _, name := range []string{
		legacyCapabilityStoreKey,
		legacyIBCFeeStoreKey,
	} {
		store.GetStoreByName(name).(storetypes.KVStore).Set(
			[]byte("legacy"),
			[]byte("preserved"),
		)
	}
	return db, store.Commit()
}

func TestSDK053IBC10NoRootStartupRequiresAuthenticatedLineage(t *testing.T) {
	currentVM := map[string]uint64{
		"ibc":                8,
		"transfer":           6,
		"interchainaccounts": 3,
	}
	sourceVM := map[string]uint64{
		"ibc":                    6,
		"transfer":               5,
		"interchainaccounts":     3,
		legacyCapabilityStoreKey: 1,
		legacyIBCFeeStoreKey:     2,
	}
	tests := []struct {
		name          string
		versionMap    map[string]uint64
		upgradeMarker string
		nativeMarker  string
		doneHeight    int64
		wantError     string
	}{
		{
			name:         "native v10 genesis",
			versionMap:   currentVM,
			nativeMarker: "genesis",
		},
		{
			name:          "loader attested completed upgrade",
			versionMap:    currentVM,
			upgradeMarker: sdk053IBC10UpgradeMarkerValue,
			doneHeight:    1,
		},
		{
			name:       "unsafe skip aftermath retains source versions",
			versionMap: sourceVM,
			wantError:  "not at required post-v10 consensus version",
		},
		{
			name:          "earlier unproved marker is refused",
			versionMap:    currentVM,
			upgradeMarker: "migrated",
			doneHeight:    1,
			wantError:     "invalid upgraded SDK/IBC lineage",
		},
		{
			name:       "done height without loader marker is refused",
			versionMap: currentVM,
			doneHeight: 1,
			wantError:  "invalid upgraded SDK/IBC lineage",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := NewZeroneApp(
				log.NewNopLogger(),
				dbm.NewMemDB(),
				nil,
				false,
				simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
			)
			require.NoError(t, application.LoadLatestVersion())
			ctx := application.NewUncachedContext(
				false,
				cmtproto.Header{Height: 1},
			)
			require.NoError(
				t,
				application.UpgradeKeeper.SetModuleVersionMap(
					ctx,
					test.versionMap,
				),
			)
			if test.upgradeMarker != "" {
				require.NoError(
					t,
					application.KnowledgeKeeper.WriteMigrationMarker(
						ctx,
						sdk053IBC10UpgradeMarker,
						test.upgradeMarker,
					),
				)
			}
			if test.nativeMarker != "" {
				require.NoError(
					t,
					application.KnowledgeKeeper.WriteMigrationMarker(
						ctx,
						sdk053IBC10NativeMarker,
						test.nativeMarker,
					),
				)
			}
			if test.doneHeight > 0 {
				key := make([]byte, 9+len(UpgradeNameSDK053IBC10))
				key[0] = upgradetypes.DoneByte
				binary.BigEndian.PutUint64(
					key[1:9],
					uint64(test.doneHeight),
				)
				copy(key[9:], UpgradeNameSDK053IBC10)
				ctx.KVStore(
					application.keys[upgradetypes.StoreKey],
				).Set(key, []byte{1})
			}
			require.Equal(
				t,
				int64(1),
				application.CommitMultiStore().Commit().Version,
			)
			err := application.
				validateSDK053IBC10CompletedOrNativeLineage()
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
			}
		})
	}
}
