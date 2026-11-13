package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
)

// A native chain starts at the consensus-repairs target versions with no
// carrier receipt, so the startup check that all nine markers are on is the
// only place that confirms them. Removing it, or any one marker, must fail.
func TestConsensusRepairsStartupRequiresEveryMarker(t *testing.T) {
	app := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID("zerone-dev-1"))
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, app))
	require.NoError(t, err)
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-dev-1"})
	_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: genesis})
	require.NoError(t, err)
	latest := app.CommitMultiStore().Commit().Version
	target := consensusRepairsTargetVersionMap()
	require.NoError(t, app.validateConsensusRepairsStartupVersions(ctx, target, latest), "native genesis selects all nine")

	locations := consensusRepairsMarkerLocations()
	require.Len(t, locations, len(app.ConsensusRepairsV1Markers()))
	for _, loc := range locations {
		store := ctx.KVStore(app.keys[loc.store])
		saved := store.Get(loc.key)
		require.NotNil(t, saved, "%s marker %x not written by native genesis", loc.store, loc.key)
		store.Delete(loc.key)
		require.ErrorContains(t, app.validateConsensusRepairsStartupVersions(ctx, target, latest),
			"consensus repairs target requires", "%s marker %x removed", loc.store, loc.key)
		store.Set(loc.key, saved)
	}
	require.NoError(t, app.validateConsensusRepairsStartupVersions(ctx, target, latest))
}

// The dev-1 runtime checks an applied carrier against its own version map.
// Read that map from runtime.py and hold it to the compiled target, so the two
// cannot drift apart unnoticed (the Python tests pin it only literally).
func TestDev1RuntimeVersionsMatchCarrierTarget(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "deploy", "networks", "zerone-dev-1", "runtime.py"))
	require.NoError(t, err)
	block := regexp.MustCompile(`(?s)\nCONSENSUS_REPAIRS_VERSIONS = \{(.*?)\n\}`).FindSubmatch(source)
	require.NotNil(t, block, "CONSENSUS_REPAIRS_VERSIONS not found in runtime.py")
	runtime := map[string]uint64{}
	for _, pair := range regexp.MustCompile(`"([a-z_]+)": (\d+)`).FindAllSubmatch(block[1], -1) {
		version, err := strconv.ParseUint(string(pair[2]), 10, 64)
		require.NoError(t, err)
		runtime[string(pair[1])] = version
	}
	require.Len(t, runtime, 13, "runtime.py's map: the ten carried modules plus home, zerone_staking and vesting_rewards")

	target := consensusRepairsTargetVersionMap()
	for name := range consensusRepairsVersions() {
		require.Contains(t, runtime, name, "runtime.py does not check %s", name)
	}
	for name, version := range runtime {
		require.Contains(t, target, name, "runtime.py checks %s, which the app does not have", name)
		require.Equal(t, target[name], version, "runtime.py's %s version", name)
	}
}
