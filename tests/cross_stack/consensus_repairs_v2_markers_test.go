package cross_stack_test

import (
	"encoding/json"
	"testing"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

// repairsV2MarkerLocations is the store and key of each
// ConsensusRepairsV2Markers entry, in registry order.
var repairsV2MarkerLocations = []struct {
	store string
	key   byte
}{{"alignment", 0x09}, {"knowledge", 0x91}, {"knowledge", 0x92}, {"knowledge", 0x93}}

// Synthetic pre-carrier state, not an accepted fresh-genesis selection. The
// owner module tests separately pin true DefaultGenesis and the file without
// the field importing as off.
func seedAbsentRepairsV2Markers(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context) {
	t.Helper()
	for _, loc := range repairsV2MarkerLocations {
		ctx.KVStore(app.GetStoreKeyForTests(loc.store)).Delete([]byte{loc.key})
	}
}

func requireMarkers(t *testing.T, ctx sdk.Context, markers []zeroneapp.ActivationMarker, want bool) {
	t.Helper()
	for _, m := range markers {
		enabled, err := m.Enabled(ctx)
		require.NoError(t, err, "%s/%s", m.Module, m.Name)
		require.Equal(t, want, enabled, "%s/%s", m.Module, m.Name)
	}
}

func storeSnapshot(t *testing.T, ctx sdk.Context, key storetypes.StoreKey) map[string]string {
	t.Helper()
	iter := ctx.KVStore(key).Iterator(nil, nil)
	defer iter.Close()
	entries := map[string]string{}
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = string(iter.Value())
	}
	return entries
}

// requireExportedMarkers exports app and imports the export into a fresh app,
// requiring every listed marker to follow at both ends.
func requireExportedMarkers(t *testing.T, app *zeroneapp.ZeroneApp, markers func(*zeroneapp.ZeroneApp) []zeroneapp.ActivationMarker, want bool) {
	t.Helper()
	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	var exportedGenesis zeroneapp.GenesisState
	require.NoError(t, json.Unmarshal(exported.AppState, &exportedGenesis))
	for _, m := range markers(app) {
		require.Equal(t, want, repairsMarkerField(t, exportedGenesis, m), "%s/%s export", m.Module, m.Name)
	}

	imported := repairsMarkersApp(t)
	_, err = imported.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: exported.Height,
		AppStateBytes: exported.AppState, ConsensusParams: &exported.ConsensusParams})
	require.NoError(t, err)
	requireMarkers(t, imported.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: exported.Height}), markers(imported), want)
}

func repairsV2Markers(app *zeroneapp.ZeroneApp) []zeroneapp.ActivationMarker {
	return app.ConsensusRepairsV2Markers()
}

// Through the real application (module manager, each module's own genesis
// codec, InitChain, commit, export and a second InitChain), a native genesis
// selects every consensus-repairs-v2 marker and an export carries it. An older
// genesis without those selections is refused rather than silently activated.
func TestConsensusRepairsV2MarkersFollowGenesisThroughTheApp(t *testing.T) {
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
			for _, m := range source.ConsensusRepairsV2Markers() {
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
			if tc.strip {
				raw, err := json.Marshal(genesis)
				require.NoError(t, err)
				_, err = source.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, AppStateBytes: raw})
				require.ErrorContains(t, err, "requires explicit")
				return
			}
			initRepairsMarkersChain(t, source, genesis)
			ctx := source.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
			requireMarkers(t, ctx, source.ConsensusRepairsV2Markers(), tc.want)
			requireMarkers(t, ctx, source.ConsensusRepairsV1Markers(), true)

			commitRepairsMarkersBlock(t, source)
			requireMarkers(t, source.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), source.ConsensusRepairsV2Markers(), tc.want)
			requireExportedMarkers(t, source, repairsV2Markers, tc.want)
		})
	}
}

// The registry the v2 carrier will loop over lists exactly the four
// consensus-repairs-v2 markers, each named by the genesis field that carries
// it. Enabling every entry on a chain without them selects every repair,
// idempotently, and an export then carries all of them.
func TestConsensusRepairsV2RegistryEnablesEveryMarker(t *testing.T) {
	app := repairsMarkersApp(t)
	var listed []string
	for _, m := range app.ConsensusRepairsV2Markers() {
		listed = append(listed, m.Module+"/"+m.Name)
	}
	require.Equal(t, []string{
		"alignment/unobserved_health_enabled",
		"knowledge/inconclusive_round_accounting_enabled",
		"knowledge/calibration_smoothing_enabled",
		"knowledge/zero_confidence_floor_enabled",
	}, listed)
	require.Len(t, repairsV2MarkerLocations, len(listed))

	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsV2Markers(t, app, ctx)
	requireMarkers(t, ctx, app.ConsensusRepairsV2Markers(), false)
	for range 2 {
		for _, m := range app.ConsensusRepairsV2Markers() {
			require.NoError(t, m.Enable(ctx), "%s/%s", m.Module, m.Name)
		}
		requireMarkers(t, ctx, app.ConsensusRepairsV2Markers(), true)
	}

	commitRepairsMarkersBlock(t, app)
	requireMarkers(t, app.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), app.ConsensusRepairsV2Markers(), true)
	requireExportedMarkers(t, app, repairsV2Markers, true)
}

// Each registry entry is bound to its own marker, store key and genesis field:
// from a chain with every v2 repair off, enabling one entry writes exactly its
// own key and turns on that entry's reader and genesis field and nothing else,
// so a swapped Enable, reader or Name cannot pass. No v1 marker moves. Every
// reader is gas-free whether its marker is absent or on, so a read added to an
// existing path cannot change gas before activation.
func TestConsensusRepairsV2RegistryEntriesAreIndependentAndGasFree(t *testing.T) {
	app := repairsMarkersApp(t)
	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsV2Markers(t, app, ctx)
	markers := app.ConsensusRepairsV2Markers()

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
		store := app.GetStoreKeyForTests(repairsV2MarkerLocations[i].store)
		before := storeSnapshot(t, cached, store)
		require.NoError(t, target.Enable(cached), "%s/%s", target.Module, target.Name)
		before[string([]byte{repairsV2MarkerLocations[i].key})] = "\x01"
		require.Equal(t, before, storeSnapshot(t, cached, store), "enabling %s/%s writes exactly its own key", target.Module, target.Name)

		exported, err := app.ModuleManager.ExportGenesisForModules(cached, app.AppCodec(), nil)
		require.NoError(t, err)
		for j, m := range markers {
			requireGasFree(cached, m, i == j)
			require.Equal(t, i == j, repairsMarkerField(t, exported, m),
				"enabling %s/%s: %s/%s genesis field", target.Module, target.Name, m.Module, m.Name)
		}
		requireMarkers(t, cached, app.ConsensusRepairsV1Markers(), true)
	}
}

// The two registries are disjoint: no module/field pair is listed twice, and
// from a chain with every v1 and v2 marker absent, enabling every v1 entry
// leaves every v2 marker off, and enabling every v2 entry leaves every v1
// marker off.
func TestConsensusRepairsRegistriesAreDisjoint(t *testing.T) {
	app := repairsMarkersApp(t)
	names := map[string]bool{}
	for _, m := range append(app.ConsensusRepairsV1Markers(), app.ConsensusRepairsV2Markers()...) {
		require.False(t, names[m.Module+"/"+m.Name], "%s/%s listed twice", m.Module, m.Name)
		names[m.Module+"/"+m.Name] = true
	}

	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsMarkers(t, app, ctx)
	seedAbsentRepairsV2Markers(t, app, ctx)
	for _, tc := range []struct {
		name          string
		enable, other []zeroneapp.ActivationMarker
	}{
		{"v1 leaves v2", app.ConsensusRepairsV1Markers(), app.ConsensusRepairsV2Markers()},
		{"v2 leaves v1", app.ConsensusRepairsV2Markers(), app.ConsensusRepairsV1Markers()},
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
