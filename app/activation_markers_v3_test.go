package app

import (
	"bytes"
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

const consensusRepairsV3TestChainID = "consensus-repairs-v3-markers"

// consensusRepairsV3TestLocations pairs each ConsensusRepairsV3Markers entry
// with its store key. It lives in this test because no v3 carrier exists yet to
// own a production list beside consensusRepairsV2MarkerLocations.
func consensusRepairsV3TestLocations() []consensusRepairsMarkerLocation {
	return []consensusRepairsMarkerLocation{
		{"knowledge", []byte{0x94}}, {"substrate_bridge", []byte{0x91}},
		{"knowledge", []byte{0x95}}, {"knowledge", []byte{0x96}},
	}
}

// consensusRepairsV3Genesis is native genesis with each v3 selection kept
// (mode ""), removed ("absent") or set false ("false").
func consensusRepairsV3Genesis(t *testing.T, a *ZeroneApp, mode string) GenesisState {
	t.Helper()
	g := sdk053IBC10GenesisWithValidator(t, a)
	for _, selection := range ConsensusRepairsV3GenesisSelections() {
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(g[selection.Module], &fields))
		require.JSONEq(t, "true", string(fields[selection.Name]), "native genesis selects %s.%s", selection.Module, selection.Name)
		switch mode {
		case "absent":
			delete(fields, selection.Name)
		case "false":
			fields[selection.Name] = json.RawMessage("false")
		}
		raw, err := json.Marshal(fields)
		require.NoError(t, err)
		g[selection.Module] = raw
	}
	return g
}

func consensusRepairsV3InitChain(t *testing.T, g GenesisState) (*ZeroneApp, sdk.Context, error) {
	t.Helper()
	a := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(consensusRepairsV3TestChainID))
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	ctx := a.NewUncachedContext(false, cmtproto.Header{ChainID: consensusRepairsV3TestChainID})
	_, err = a.InitChainer(ctx, &abci.RequestInitChain{ChainId: consensusRepairsV3TestChainID, AppStateBytes: raw})
	return a, ctx, err
}

// The registry lists the v3 selections in order. From a chain without them,
// each entry's Enable writes exactly its own key in all of the app's stores,
// and no v3 location is a v1 or v2 marker location.
func TestConsensusRepairsV3RegistryWritesOnlyItsOwnKeys(t *testing.T) {
	builder := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	a, ctx, err := consensusRepairsV3InitChain(t, consensusRepairsV3Genesis(t, builder, ""))
	require.NoError(t, err)
	markers, locations := a.ConsensusRepairsV3Markers(), consensusRepairsV3TestLocations()
	require.Len(t, markers, len(locations))
	require.Len(t, ConsensusRepairsV3GenesisSelections(), len(markers))
	earlier := append(consensusRepairsMarkerLocations(), consensusRepairsV2MarkerLocations()...)
	for _, loc := range locations {
		for _, e := range earlier {
			require.False(t, loc.store == e.store && bytes.Equal(loc.key, e.key), "v3 %s %x is an earlier marker", loc.store, loc.key)
		}
		require.Equal(t, []byte{1}, ctx.KVStore(a.keys[loc.store]).Get(loc.key), "native genesis wrote %s %x", loc.store, loc.key)
		ctx.KVStore(a.keys[loc.store]).Delete(loc.key)
	}
	for i, marker := range markers {
		require.Equal(t, ActivationSelection{marker.Module, marker.Name}, ConsensusRepairsV3GenesisSelections()[i])
		cache, _ := ctx.CacheContext()
		before := claimIntegrityAllStores(t, a, cache)
		require.NoError(t, marker.Enable(cache))
		after := claimIntegrityAllStores(t, a, cache)
		key := locations[i].store + ":" + string(locations[i].key)
		require.Nil(t, before[key])
		require.Equal(t, []byte{1}, after[key])
		delete(after, key)
		require.Equal(t, before, after, "%s.%s writes only its own key", marker.Module, marker.Name)
	}
}

// No v3 carrier exists in this source, so native genesis may leave the v3
// fields out or set them false. ValidateConsensusRepairsGenesisSelection and
// InitChain accept it, every v3 marker starts off, and the v1 and v2 readback
// still passes. The v3 carrier's genesis requirement will change this test.
func TestConsensusRepairsV3SelectionIsNotRequiredBeforeItsCarrier(t *testing.T) {
	builder := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	for _, name := range builder.KnownUpgradeNames() {
		require.NotContains(t, name, "consensus-repairs-v3", "a v3 carrier now exists; move the genesis requirement with it")
	}
	for _, mode := range []string{"absent", "false"} {
		t.Run(mode, func(t *testing.T) {
			g := consensusRepairsV3Genesis(t, builder, mode)
			require.NoError(t, ValidateConsensusRepairsGenesisSelection(g))
			a, ctx, err := consensusRepairsV3InitChain(t, g)
			require.NoError(t, err)
			for _, marker := range a.ConsensusRepairsV3Markers() {
				on, err := marker.Enabled(ctx)
				require.NoError(t, err)
				require.False(t, on, "%s.%s", marker.Module, marker.Name)
			}
			for _, marker := range append(a.ConsensusRepairsV1Markers(), a.ConsensusRepairsV2Markers()...) {
				on, err := marker.Enabled(ctx)
				require.NoError(t, err)
				require.True(t, on, "%s.%s", marker.Module, marker.Name)
			}
		})
	}
}
