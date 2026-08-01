package main

import (
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

func TestRegularIAVLReaderWithFastNodesDisabledBackends(t *testing.T) {
	testCases := []struct {
		name            string
		keys            [][]byte
		initialOnlyKeys [][]byte
		referenceHeight bool
		pruneHistory    bool
	}{
		{
			name: "empty",
		},
		{
			name: "previously_nonempty_then_empty",
			initialOnlyKeys: [][]byte{
				[]byte("channelUpgrades/removed-before-snapshot"),
			},
		},
		{
			name: "nonempty_reference_height",
			keys: [][]byte{
				[]byte("channelUpgrades/a"),
				[]byte(
					"pruningSequenceStart/ports/transfer/channels/channel-0",
				),
				[]byte("recvStartSequence/ports/transfer/channels/channel-0"),
			},
			referenceHeight: true,
		},
		{
			name: "pruned_latest_root",
			keys: [][]byte{
				[]byte("channelUpgrades/a"),
				[]byte(
					"pruningSequenceStart/ports/transfer/channels/channel-0",
				),
				[]byte("recvStartSequence/ports/transfer/channels/channel-0"),
			},
			pruneHistory: true,
		},
	}

	for _, backend := range []dbm.BackendType{
		dbm.GoLevelDBBackend,
		dbm.PebbleDBBackend,
	} {
		for _, testCase := range testCases {
			t.Run(string(backend)+"/"+testCase.name, func(t *testing.T) {
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
				ibcKey := storetypes.NewKVStoreKey("ibc")
				root.MountStoreWithDB(ibcKey, storetypes.StoreTypeIAVL, nil)
				require.NoError(t, root.LoadLatestVersion())

				for _, key := range testCase.keys {
					root.GetKVStore(ibcKey).Set(
						key,
						append([]byte("value:"), key...),
					)
				}
				for _, key := range testCase.initialOnlyKeys {
					root.GetKVStore(ibcKey).Set(
						key,
						append([]byte("value:"), key...),
					)
				}
				commitID := root.Commit()
				switch {
				case len(testCase.initialOnlyKeys) != 0:
					for _, key := range testCase.initialOnlyKeys {
						root.GetKVStore(ibcKey).Delete(key)
					}
					commitID = root.Commit()

				case testCase.referenceHeight:
					firstIBCCommit := root.GetCommitKVStore(ibcKey).LastCommitID()
					commitID = root.Commit()
					secondIBCCommit := root.GetCommitKVStore(ibcKey).LastCommitID()
					require.Equal(t, int64(2), commitID.Version)
					require.Equal(t, firstIBCCommit.Hash, secondIBCCommit.Hash)

				case testCase.pruneHistory:
					historyKey := []byte("transient-history/key")
					root.GetKVStore(ibcKey).Set(historyKey, []byte("version-two"))
					_ = root.Commit()
					root.GetKVStore(ibcKey).Delete(historyKey)
					commitID = root.Commit()
					require.Equal(t, int64(3), commitID.Version)
					require.NoError(t, root.PruneStores(2))

					versioned := root.GetCommitKVStore(ibcKey).(interface {
						VersionExists(int64) bool
						GetAllVersions() []int
					})
					require.False(t, versioned.VersionExists(1))
					require.False(t, versioned.VersionExists(2))
					require.True(t, versioned.VersionExists(3))
					require.Equal(t, []int{3}, versioned.GetAllVersions())
				}
				require.Len(t, commitID.Hash, 32)
				require.NoError(t, writer.Close())

				dbPath := filepath.Join(dataDir, applicationDBName+dbm.DBFileSuffix)
				before := directoryContentDigest(t, dbPath)
				reader, err := openCopiedApplicationDB(home, string(backend))
				require.NoError(t, err)

				storageVersion, err := reader.Get(
					[]byte(ibcPhysicalStorePrefix + "mstorage_version"),
				)
				require.NoError(t, err)
				require.Nil(t, storageVersion)
				fastNode, err := reader.Get(
					[]byte(ibcPhysicalStorePrefix + "fchannelUpgrades/a"),
				)
				require.NoError(t, err)
				require.Nil(t, fastNode)

				expected := expectedEvidence{
					Height:  commitID.Version,
					AppHash: commitID.Hash,
				}
				got, err := auditApplicationDB(reader, expected)
				require.NoError(t, err)

				var channelKeys, pruningKeys [][]byte
				for _, key := range testCase.keys {
					switch {
					case len(key) >= len(channelUpgradesLogicalPrefix) &&
						string(key[:len(channelUpgradesLogicalPrefix)]) ==
							channelUpgradesLogicalPrefix:
						channelKeys = append(channelKeys, key)
					case len(key) >= len(pruningSequenceLogicalPrefix) &&
						string(key[:len(pruningSequenceLogicalPrefix)]) ==
							pruningSequenceLogicalPrefix:
						pruningKeys = append(pruningKeys, key)
					}
				}
				want, err := buildPlanInfo(channelKeys, pruningKeys)
				require.NoError(t, err)
				require.Equal(t, string(want), string(got))

				switch typed := reader.(type) {
				case *dbm.GoLevelDB:
					require.Error(
						t,
						typed.Set([]byte("must-not-write"), []byte("value")),
					)
				case *pebbleReadDB:
					require.Error(
						t,
						typed.db.Set(
							[]byte("must-not-write"),
							[]byte("value"),
							pebble.Sync,
						),
					)
				default:
					t.Fatalf("unexpected read-only backend type %T", reader)
				}
				require.NoError(t, reader.Close())
				after := directoryContentDigest(t, dbPath)
				require.Equal(t, before, after)
			})
		}
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
	require.ErrorContains(t, err, "home")
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
	require.NoError(
		t,
		os.Mkdir(filepath.Join(fakeUserHome, ".zeroned"), 0o700),
	)
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
