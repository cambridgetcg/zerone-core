package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	corestore "cosmossdk.io/core/store"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
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

type failingIBCCleanupStore struct {
	dbm.DB

	iteratorOpenErr  error
	iteratorErr      error
	iteratorCloseErr error
	deleteErr        error
}

func (s failingIBCCleanupStore) Iterator(start, end []byte) (corestore.Iterator, error) {
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

func (s failingIBCCleanupStore) Delete(key []byte) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.DB.Delete(key)
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

	// Exercise the same SDK cache-merge iterator and error-returning runtime
	// adapter used by the upgrade handler, with legacy keys in committed state.
	rootStore := rootmulti.NewStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	ibcKey := storetypes.NewKVStoreKey("ibc-prefix-cleanup-test")
	rootStore.MountStoreWithDB(ibcKey, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, rootStore.LoadLatestVersion())
	committedIBCStore := rootStore.GetKVStore(ibcKey)
	for key, value := range obsolete {
		committedIBCStore.Set([]byte(key), value)
	}
	for key, value := range retained {
		committedIBCStore.Set([]byte(key), value)
	}
	rootStore.Commit()

	ctx := sdk.NewContext(rootStore.CacheMultiStore(), cmtproto.Header{}, false, log.NewNopLogger())
	store := sdkruntime.NewKVStoreService(ibcKey).OpenKVStore(ctx)

	deleted, err := deleteObsoleteIBCChannelPrefixes(store)
	require.NoError(t, err)
	require.Equal(t, len(obsolete), deleted)

	for key := range obsolete {
		value, err := store.Get([]byte(key))
		require.NoError(t, err)
		require.Nil(t, value, "%s must be removed", key)
	}
	for key, expected := range retained {
		value, err := store.Get([]byte(key))
		require.NoError(t, err)
		require.Equal(t, expected, value, "%s must be preserved byte-for-byte", key)
	}

	// Re-running the repair is a no-op, which matters if an upgrade block is
	// replayed after a rollback.
	deleted, err = deleteObsoleteIBCChannelPrefixes(store)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func TestDeleteObsoleteIBCChannelPrefixesFailsClosed(t *testing.T) {
	testCases := []struct {
		name  string
		store failingIBCCleanupStore
		want  string
	}{
		{
			name: "iterator open",
			store: failingIBCCleanupStore{
				DB:              dbm.NewMemDB(),
				iteratorOpenErr: errInjectedIBCCleanup,
			},
			want: "open iterator",
		},
		{
			name: "iterator traversal",
			store: failingIBCCleanupStore{
				DB:          dbm.NewMemDB(),
				iteratorErr: errInjectedIBCCleanup,
			},
			want: "iterate obsolete IBC prefix",
		},
		{
			name: "iterator close",
			store: failingIBCCleanupStore{
				DB:               dbm.NewMemDB(),
				iteratorCloseErr: errInjectedIBCCleanup,
			},
			want: "close iterator",
		},
		{
			name: "delete",
			store: failingIBCCleanupStore{
				DB:        dbm.NewMemDB(),
				deleteErr: errInjectedIBCCleanup,
			},
			want: "delete obsolete IBC key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.store.deleteErr != nil {
				require.NoError(t, tc.store.DB.Set(
					[]byte("channelUpgrades/upgrades/ports/transfer/channels/channel-0"),
					[]byte("upgrade"),
				))
			}

			deleted, err := deleteObsoleteIBCChannelPrefixes(tc.store)
			require.Error(t, err)
			require.ErrorIs(t, err, errInjectedIBCCleanup)
			require.Contains(t, err.Error(), tc.want)
			require.Zero(t, deleted)
		})
	}
}

func TestSDK053IBC10StoreLoaderDeletesLegacyStoresAndRestarts(t *testing.T) {
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
