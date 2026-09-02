package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/user"
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

func TestCopiedApplicationDBScansRequiredStoresReadOnlyAcrossBackends(t *testing.T) {
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
			root.SetIAVLDisableFastNode(true)

			keys := make(map[string]*storetypes.KVStoreKey, len(requiredStoreNames))
			for _, name := range requiredStoreNames {
				keys[name] = storetypes.NewKVStoreKey(name)
				root.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, nil)
			}
			require.NoError(t, root.LoadLatestVersion())

			fixture := map[string][]logicalLeaf{
				"zerone_staking": {
					{Key: []byte{0x01, 'v'}, Value: []byte("validator")},
					{Key: []byte{0x02, 'd'}, Value: []byte("delegation")},
				},
				"bank": {
					{Key: []byte{0x02, 'b'}, Value: []byte("balance")},
				},
				// Leave SDK staking empty to exercise the committed-empty path.
				"staking": nil,
			}
			for name, leaves := range fixture {
				for _, leaf := range leaves {
					root.GetKVStore(keys[name]).Set(leaf.Key, leaf.Value)
				}
			}
			firstCommitID := root.Commit()
			commitID := root.Commit()
			require.Equal(t, int64(1), firstCommitID.Version)
			require.Equal(t, int64(2), commitID.Version)
			// The latest version is a reference root for the unchanged nonempty
			// stores, while SDK staking remains a physically untouched empty tree.
			require.Len(t, commitID.Hash, sha256.Size)
			require.NoError(t, writer.Close())

			dbPath := filepath.Join(dataDir, applicationDBName+dbm.DBFileSuffix)
			before := backendDirectoryContentDigest(t, dbPath)
			reader, err := openCopiedApplicationDB(home, string(backend))
			require.NoError(t, err)

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
			snapshot, evidence, err := scanApplicationDB(
				reader,
				expectedEvidence{Height: commitID.Version, AppHash: commitID.Hash},
				visitors,
			)
			require.NoError(t, err)
			require.Equal(t, commitID.Version, snapshot.height)
			require.Equal(t, commitID.Hash, snapshot.appHash)
			require.Len(t, evidence, len(requiredStoreNames))
			for index, name := range requiredStoreNames {
				require.Equal(t, name, evidence[index].name)
				require.Equal(t, fixture[name], visited[name])
				require.Equal(t, int64(len(fixture[name])), evidence[index].leafCount)
				require.Len(t, evidence[index].rootHash, sha256.Size)
				require.Len(t, evidence[index].leavesHash, sha256.Size)
			}

			switch typed := reader.(type) {
			case *dbm.GoLevelDB:
				require.Error(t, typed.Set([]byte("must-not-write"), []byte("value")))
			case *pebbleReadDB:
				require.Error(t, typed.db.Set(
					[]byte("must-not-write"),
					[]byte("value"),
					pebble.Sync,
				))
			default:
				t.Fatalf("unexpected read-only backend type %T", reader)
			}
			require.NoError(t, reader.Close())
			after := backendDirectoryContentDigest(t, dbPath)
			require.Equal(t, before, after)
		})
	}
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
	require.ErrorContains(t, err, "symlink")

	homeWithLinkedData := t.TempDir()
	dataTarget := t.TempDir()
	require.NoError(t, os.Symlink(dataTarget, filepath.Join(homeWithLinkedData, "data")))
	_, err = openCopiedApplicationDB(homeWithLinkedData, "goleveldb")
	require.ErrorContains(t, err, "data directory")
	require.ErrorContains(t, err, "symlink")
}

func TestOpenCopiedApplicationDBRejectsIntermediateAliasToLiveHome(t *testing.T) {
	fakeUserHome := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(fakeUserHome, ".zeroned"), 0o700))
	t.Setenv("HOME", fakeUserHome)

	aliasParent := t.TempDir()
	userHomeAlias := filepath.Join(aliasParent, "user-home-alias")
	require.NoError(t, os.Symlink(fakeUserHome, userHomeAlias))
	_, err := openCopiedApplicationDB(
		filepath.Join(userHomeAlias, ".zeroned"),
		"goleveldb",
	)
	require.ErrorContains(t, err, "resolved alias")
	require.ErrorContains(t, err, "default live home")
}

func TestProtectedUserHomesCannotBeSpoofedWithEnvironment(t *testing.T) {
	account, err := user.Current()
	require.NoError(t, err)
	t.Setenv("HOME", t.TempDir())

	homes, err := protectedUserHomes()
	require.NoError(t, err)
	require.Contains(t, homes, filepath.Clean(account.HomeDir))
}

func TestOpenCopiedApplicationDBRejectsCaseAliasToLiveHome(t *testing.T) {
	fakeUserHome := t.TempDir()
	canonical := filepath.Join(fakeUserHome, ".zeroned")
	require.NoError(t, os.Mkdir(canonical, 0o700))
	t.Setenv("HOME", fakeUserHome)

	caseAlias := filepath.Join(fakeUserHome, ".ZERONED")
	canonicalInfo, err := os.Stat(canonical)
	require.NoError(t, err)
	aliasInfo, err := os.Stat(caseAlias)
	if err != nil || !os.SameFile(canonicalInfo, aliasInfo) {
		t.Skip("filesystem is case-sensitive")
	}

	_, err = openCopiedApplicationDB(caseAlias, "goleveldb")
	require.ErrorContains(t, err, "default live home")
}

func backendDirectoryContentDigest(t *testing.T, root string) string {
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
