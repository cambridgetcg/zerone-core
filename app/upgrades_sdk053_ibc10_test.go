package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	pruningtypes "cosmossdk.io/store/pruning/types"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

func TestSDK053IBC10StoreUpgrades(t *testing.T) {
	upgrades := sdk053IBC10StoreUpgrades()

	require.Empty(t, upgrades.Added)
	require.Empty(t, upgrades.Renamed)
	require.Equal(t, []string{"capability", "feeibc"}, upgrades.Deleted)
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
