package app

import (
	"path/filepath"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storemetrics "cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

func TestActivationPreflightReopensPreAndPostSDKStoreSets(t *testing.T) {
	template := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		false,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	)
	currentStoreNames := make([]string, 0, len(template.keys))
	for name := range template.keys {
		currentStoreNames = append(currentStoreNames, name)
	}

	for _, test := range []struct {
		name          string
		includeLegacy bool
	}{
		{name: "pre SDK removal", includeLegacy: true},
		{name: "post SDK removal", includeLegacy: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			dataDir := filepath.Join(home, "data")
			require.NoError(t, writeSyntheticApplicationCommit(
				dataDir,
				currentStoreNames,
				test.includeLegacy,
			))

			reopenedDB, err := dbm.NewDB(
				"application",
				dbm.GoLevelDBBackend,
				dataDir,
			)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, reopenedDB.Close())
			})
			reopened := NewActivationPreflightApp(
				log.NewNopLogger(),
				reopenedDB,
				nil,
				false,
				simtestutil.AppOptionsMap{
					flags.FlagHome: home,
				},
			)
			require.NoError(t, reopened.LoadLatestVersion())
			_, capabilityMounted :=
				reopened.keys[legacyCapabilityStoreKey]
			_, feeIBCMounted := reopened.keys[legacyIBCFeeStoreKey]
			require.Equal(
				t,
				test.includeLegacy,
				capabilityMounted,
			)
			require.Equal(t, test.includeLegacy, feeIBCMounted)
		})
	}
}

func TestNormalConstructorCannotInjectActivationPreflightMode(t *testing.T) {
	application := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		false,
		simtestutil.AppOptionsMap{
			"zerone.activation-preflight-read-only": true,
		},
	)
	require.False(
		t,
		application.activationPreflightReadOnly,
		"normal daemon construction must ignore arbitrary offline-mode options",
	)
}

func writeSyntheticApplicationCommit(
	dataDir string,
	currentStoreNames []string,
	includeLegacy bool,
) error {
	db, err := dbm.NewDB(
		"application",
		dbm.GoLevelDBBackend,
		dataDir,
	)
	if err != nil {
		return err
	}
	store := rootmulti.NewStore(
		db,
		log.NewNopLogger(),
		storemetrics.NewNoOpMetrics(),
	)
	keys := make([]*storetypes.KVStoreKey, 0, len(currentStoreNames)+2)
	for _, name := range currentStoreNames {
		key := storetypes.NewKVStoreKey(name)
		keys = append(keys, key)
		store.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}
	if includeLegacy {
		for _, name := range []string{
			legacyCapabilityStoreKey,
			legacyIBCFeeStoreKey,
		} {
			key := storetypes.NewKVStoreKey(name)
			keys = append(keys, key)
			store.MountStoreWithDB(
				key,
				storetypes.StoreTypeIAVL,
				nil,
			)
		}
	}
	if err := store.LoadLatestVersion(); err != nil {
		_ = db.Close()
		return err
	}
	for _, key := range keys {
		store.GetKVStore(key).Set(
			[]byte("_preflight_reopen_test"),
			[]byte{1},
		)
	}
	store.Commit()
	return db.Close()
}
