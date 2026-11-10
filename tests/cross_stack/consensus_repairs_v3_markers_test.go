package cross_stack_test

import (
	"encoding/json"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

// repairsV3MarkerLocations is the store and key of each
// ConsensusRepairsV3Markers entry, in registry order.
var repairsV3MarkerLocations = []struct {
	store string
	key   byte
}{{"knowledge", 0x94}, {"substrate_bridge", 0x91}}

// Synthetic pre-carrier state: an existing chain that has not run a v3
// carrier. The owner module tests separately pin true DefaultGenesis and the
// file without the field importing as off.
func seedAbsentRepairsV3Markers(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context) {
	t.Helper()
	for _, loc := range repairsV3MarkerLocations {
		ctx.KVStore(app.GetStoreKeyForTests(loc.store)).Delete([]byte{loc.key})
	}
}

func repairsV3Markers(app *zeroneapp.ZeroneApp) []zeroneapp.ActivationMarker {
	return app.ConsensusRepairsV3Markers()
}

// Through the real application (module manager, each module's own genesis
// codec, InitChain, commit, export and a second InitChain), a native genesis
// selects every consensus-repairs-v3 marker and an export carries it. No v3
// carrier exists yet, so InitChain does not require the selection: a genesis
// without the v3 fields starts, commits, exports and re-imports with every v3
// marker off and every v1 and v2 marker on. The v3 carrier will add its own
// genesis selection check, as v2's did, and this case will then be refused.
func TestConsensusRepairsV3MarkersFollowGenesisThroughTheApp(t *testing.T) {
	for _, tc := range []struct {
		name  string
		strip bool
		want  bool
	}{
		{"native genesis", false, true},
		{"genesis without the fields", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := repairsMarkersApp(t)
			genesis := genesisStateWithValSet(t, source, source.DefaultGenesis())
			for _, m := range source.ConsensusRepairsV3Markers() {
				require.True(t, repairsMarkerField(t, genesis, m), "%s default genesis selects %s", m.Module, m.Name)
				if tc.strip {
					var fields map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(genesis[m.Module], &fields))
					delete(fields, m.Name)
					stripped, err := json.Marshal(fields)
					require.NoError(t, err)
					genesis[m.Module] = stripped
				}
			}
			initRepairsMarkersChain(t, source, genesis)
			ctx := source.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
			requireMarkers(t, ctx, source.ConsensusRepairsV3Markers(), tc.want)
			requireMarkers(t, ctx, source.ConsensusRepairsV1Markers(), true)
			requireMarkers(t, ctx, source.ConsensusRepairsV2Markers(), true)

			commitRepairsMarkersBlock(t, source)
			requireMarkers(t, source.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), source.ConsensusRepairsV3Markers(), tc.want)
			requireExportedMarkers(t, source, repairsV3Markers, tc.want)
		})
	}
}

// The registry a v3 carrier will loop over lists exactly the two
// consensus-repairs-v3 markers, each named by the genesis field that carries
// it, in the order of ConsensusRepairsV3GenesisSelections. Enabling every entry
// on a chain without them selects every repair, idempotently, and an export
// then carries all of them.
func TestConsensusRepairsV3RegistryEnablesEveryMarker(t *testing.T) {
	app := repairsMarkersApp(t)
	var listed []string
	for i, m := range app.ConsensusRepairsV3Markers() {
		listed = append(listed, m.Module+"/"+m.Name)
		require.Equal(t, zeroneapp.ActivationSelection{Module: m.Module, Name: m.Name}, zeroneapp.ConsensusRepairsV3GenesisSelections()[i])
	}
	require.Equal(t, []string{
		"knowledge/doctrine_identity_enabled",
		"substrate_bridge/cited_reward_escrow_enabled",
	}, listed)
	require.Len(t, zeroneapp.ConsensusRepairsV3GenesisSelections(), len(listed))
	require.Len(t, repairsV3MarkerLocations, len(listed))

	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsV3Markers(t, app, ctx)
	requireMarkers(t, ctx, app.ConsensusRepairsV3Markers(), false)
	for range 2 {
		for _, m := range app.ConsensusRepairsV3Markers() {
			require.NoError(t, m.Enable(ctx), "%s/%s", m.Module, m.Name)
		}
		requireMarkers(t, ctx, app.ConsensusRepairsV3Markers(), true)
	}

	commitRepairsMarkersBlock(t, app)
	requireMarkers(t, app.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), app.ConsensusRepairsV3Markers(), true)
	requireExportedMarkers(t, app, repairsV3Markers, true)
}

// Each registry entry is bound to its own marker, store key and genesis field:
// from a chain with every v3 repair off, enabling one entry writes exactly its
// own key and turns on that entry's reader and genesis field and nothing else,
// so a swapped Enable, reader or Name cannot pass. No v1 or v2 marker moves.
// Every reader is gas-free whether its marker is absent or on, so a read added
// to an existing path cannot change gas before activation.
func TestConsensusRepairsV3RegistryEntriesAreIndependentAndGasFree(t *testing.T) {
	app := repairsMarkersApp(t)
	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsV3Markers(t, app, ctx)
	markers := app.ConsensusRepairsV3Markers()

	requireGasFree := func(ctx sdk.Context, m zeroneapp.ActivationMarker, want bool) {
		t.Helper()
		meter := storetypes.NewGasMeter(1_000_000)
		enabled, err := m.Enabled(ctx.WithGasMeter(meter))
		require.NoError(t, err, "%s/%s", m.Module, m.Name)
		require.Equal(t, want, enabled, "%s/%s", m.Module, m.Name)
		require.Zero(t, meter.GasConsumed(), "%s/%s reader charged gas", m.Module, m.Name)
	}

	for i, target := range markers {
		cached, _ := ctx.CacheContext()
		requireGasFree(cached, target, false)
		store := app.GetStoreKeyForTests(repairsV3MarkerLocations[i].store)
		before := storeSnapshot(t, cached, store)
		require.NoError(t, target.Enable(cached), "%s/%s", target.Module, target.Name)
		before[string([]byte{repairsV3MarkerLocations[i].key})] = "\x01"
		require.Equal(t, before, storeSnapshot(t, cached, store), "enabling %s/%s writes exactly its own key", target.Module, target.Name)

		exported, err := app.ModuleManager.ExportGenesisForModules(cached, app.AppCodec(), nil)
		require.NoError(t, err)
		for j, m := range markers {
			requireGasFree(cached, m, i == j)
			require.Equal(t, i == j, repairsMarkerField(t, exported, m),
				"enabling %s/%s: %s/%s genesis field", target.Module, target.Name, m.Module, m.Name)
		}
		requireMarkers(t, cached, app.ConsensusRepairsV1Markers(), true)
		requireMarkers(t, cached, app.ConsensusRepairsV2Markers(), true)
	}
}

// The v3 registry is disjoint from v1 and v2: no module/field pair is listed
// twice across the three, and from a chain with every v1, v2 and v3 marker
// absent, enabling every v1 and v2 entry leaves every v3 marker off, and
// enabling every v3 entry leaves every v1 and v2 marker off.
func TestConsensusRepairsV3RegistryIsDisjointFromV1AndV2(t *testing.T) {
	app := repairsMarkersApp(t)
	earlier := append(app.ConsensusRepairsV1Markers(), app.ConsensusRepairsV2Markers()...)
	v3 := app.ConsensusRepairsV3Markers()
	names := map[string]bool{}
	for _, m := range append(append([]zeroneapp.ActivationMarker{}, earlier...), v3...) {
		require.False(t, names[m.Module+"/"+m.Name], "%s/%s listed twice", m.Module, m.Name)
		names[m.Module+"/"+m.Name] = true
	}

	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsMarkers(t, app, ctx)
	seedAbsentRepairsV2Markers(t, app, ctx)
	seedAbsentRepairsV3Markers(t, app, ctx)
	for _, tc := range []struct {
		name          string
		enable, other []zeroneapp.ActivationMarker
	}{
		{"v1 and v2 leave v3", earlier, v3},
		{"v3 leaves v1 and v2", v3, earlier},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cached, _ := ctx.CacheContext()
			requireMarkers(t, cached, tc.enable, false)
			requireMarkers(t, cached, tc.other, false)
			for _, m := range tc.enable {
				require.NoError(t, m.Enable(cached), "%s/%s", m.Module, m.Name)
			}
			requireMarkers(t, cached, tc.enable, true)
			requireMarkers(t, cached, tc.other, false)
		})
	}
}
