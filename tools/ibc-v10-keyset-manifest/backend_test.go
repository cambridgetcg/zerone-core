package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	"github.com/cockroachdb/pebble"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

func TestRawReaderMatchesRootmultiIAVLPhysicalFormat(t *testing.T) {
	for _, backend := range []dbm.BackendType{
		dbm.GoLevelDBBackend,
		dbm.PebbleDBBackend,
	} {
		t.Run(string(backend), func(t *testing.T) {
			home := t.TempDir()
			dataDir := filepath.Join(home, "data")
			require.NoError(t, os.Mkdir(dataDir, 0o700))

			writer, err := dbm.NewDB(applicationDBName, backend, dataDir)
			require.NoError(t, err)
			root := rootmulti.NewStore(
				writer,
				log.NewNopLogger(),
				metrics.NewNoOpMetrics(),
			)
			root.SetIAVLSyncPruning(true)
			root.SetIAVLDisableFastNode(false)
			ibcKey := storetypes.NewKVStoreKey("ibc")
			root.MountStoreWithDB(ibcKey, storetypes.StoreTypeIAVL, nil)
			require.NoError(t, root.LoadLatestVersion())

			logicalKey := []byte("channelUpgrades/a")
			root.GetKVStore(ibcKey).Set(logicalKey, []byte("upgrade-state"))
			commitID := root.Commit()
			require.Equal(t, int64(1), commitID.Version)
			require.Len(t, commitID.Hash, 32)
			require.NoError(t, writer.Close())

			reader, err := openCopiedApplicationDB(home, string(backend))
			require.NoError(t, err)
			rawFastNode, err := reader.Get(
				append(bytes.Clone(ibcFastPhysicalPrefix), logicalKey...),
			)
			require.NoError(t, err)
			require.NotEmpty(t, rawFastNode)
			require.Equal(
				t,
				[]byte("1.1.0-1"),
				mustGetPhysical(t, reader, ibcStorageVersionKey),
			)

			got, err := auditApplicationDB(reader, expectedEvidence{
				Height:  commitID.Version,
				AppHash: commitID.Hash,
			})
			require.NoError(t, err)
			require.Contains(
				t,
				string(got),
				`"channel_upgrades":{"key_count":"1","keys_sha256":"a7009f386a4870c5e9825050e952bcf489e916eed29269771ff7713444d57767"}`,
			)
			require.NoError(t, reader.Close())
		})
	}
}

func TestOpenCopiedApplicationDBReadOnlyBackends(t *testing.T) {
	for _, backend := range []dbm.BackendType{
		dbm.GoLevelDBBackend,
		dbm.PebbleDBBackend,
	} {
		t.Run(string(backend), func(t *testing.T) {
			home := t.TempDir()
			dataDir := filepath.Join(home, "data")
			require.NoError(t, os.Mkdir(dataDir, 0o700))

			writer, err := dbm.NewDB(applicationDBName, backend, dataDir)
			require.NoError(t, err)
			emptyRoot := sha256.Sum256(nil)
			expected := seedPhysicalFixture(t, writer, 42, nil, nil, emptyRoot[:])
			require.NoError(t, writer.Close())
			before := directoryContentDigest(t, filepath.Join(dataDir, "application.db"))

			reader, err := openCopiedApplicationDB(home, string(backend))
			require.NoError(t, err)
			got, err := auditApplicationDB(reader, expected)
			require.NoError(t, err)
			require.Contains(t, string(got), `"schema":"`+planInfoSchema+`"`)

			switch typed := reader.(type) {
			case *dbm.GoLevelDB:
				require.Error(t, typed.Set([]byte("must-not-write"), []byte("value")))
			case *pebbleReadDB:
				require.Error(
					t,
					typed.db.Set([]byte("must-not-write"), []byte("value"), pebble.Sync),
				)
			default:
				t.Fatalf("unexpected read-only backend type %T", reader)
			}
			require.NoError(t, reader.Close())

			after := directoryContentDigest(t, filepath.Join(dataDir, "application.db"))
			require.Equal(t, before, after)
		})
	}
}

func mustGetPhysical(t *testing.T, db physicalDB, key []byte) []byte {
	t.Helper()
	value, err := db.Get(key)
	require.NoError(t, err)
	require.NotNil(t, value)
	return value
}

func TestOpenCopiedApplicationDBRejectsUnsafePaths(t *testing.T) {
	userHome, err := os.UserHomeDir()
	require.NoError(t, err)
	_, err = openCopiedApplicationDB(filepath.Join(userHome, ".zeroned"), "goleveldb")
	require.ErrorContains(t, err, "default live home")
	_, err = openCopiedApplicationDB(
		filepath.Join(userHome, ".zeroned", "localnet", "val0"),
		"goleveldb",
	)
	require.ErrorContains(t, err, "default live home")

	_, err = openCopiedApplicationDB("relative/copy", "goleveldb")
	require.ErrorContains(t, err, "absolute path")

	realHome := t.TempDir()
	linkHome := filepath.Join(t.TempDir(), "linked-home")
	require.NoError(t, os.Symlink(realHome, linkHome))
	_, err = openCopiedApplicationDB(linkHome, "goleveldb")
	require.ErrorContains(t, err, "home")
	require.ErrorContains(t, err, "symlink")

	homeWithLinkedData := t.TempDir()
	dataTarget := t.TempDir()
	require.NoError(t, os.Symlink(dataTarget, filepath.Join(homeWithLinkedData, "data")))
	_, err = openCopiedApplicationDB(homeWithLinkedData, "goleveldb")
	require.ErrorContains(t, err, "data directory")
	require.ErrorContains(t, err, "symlink")
}

func directoryContentDigest(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}))
	sort.Strings(paths)

	hasher := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		require.NoError(t, err)
		_, _ = hasher.Write([]byte(relative))
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		_, _ = hasher.Write(contents)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
