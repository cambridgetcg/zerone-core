package cross_stack_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

const repairsMarkersChainID = "zerone-consensus-repairs-v1-markers"

// Synthetic pre-carrier state, not an accepted fresh-genesis selection. The
// owner module tests separately pin false DefaultGenesis and legacy parity.
func seedAbsentRepairsMarkers(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context) {
	t.Helper()
	for store, key := range map[string]byte{"zerone_gov": 0x18, "knowledge": 0x90, "capture_defense": 0x08, "qualification": 0x40, "capture_challenge": 0x05, "vesting_rewards": 0x0c, "zerone_ontology": 0x06, "emergency": 0x14, "substrate_bridge": 0x90, "zerone_ibcrl": 0x04} {
		ctx.KVStore(app.GetStoreKeyForTests(store)).Delete([]byte{key})
	}
}

func repairsMarkersApp(t *testing.T) *zeroneapp.ZeroneApp {
	t.Helper()
	app := zeroneapp.NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID(repairsMarkersChainID), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	return app
}

// repairsMarkerField reads one marker field from module genesis JSON. Modules
// whose encoder omits false fields leave it out, which reads as off.
func repairsMarkerField(t *testing.T, genesis zeroneapp.GenesisState, m zeroneapp.ActivationMarker) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(genesis[m.Module], &fields), m.Module)
	value, ok := fields[m.Name]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on), m.Module)
	return on
}

// stripRepairsMarkerFields removes every marker field from genesis, as in a
// file written before the fields existed.
func stripRepairsMarkerFields(t *testing.T, app *zeroneapp.ZeroneApp, genesis zeroneapp.GenesisState) {
	t.Helper()
	for _, m := range app.ConsensusRepairsV1Markers() {
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(genesis[m.Module], &fields))
		delete(fields, m.Name)
		stripped, err := json.Marshal(fields)
		require.NoError(t, err)
		genesis[m.Module] = stripped
	}
}

func requireRepairsMarkers(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context, want bool) {
	t.Helper()
	for _, m := range app.ConsensusRepairsV1Markers() {
		enabled, err := m.Enabled(ctx)
		require.NoError(t, err, m.Module)
		require.Equal(t, want, enabled, m.Module)
	}
}

func initRepairsMarkersChain(t *testing.T, app *zeroneapp.ZeroneApp, genesis zeroneapp.GenesisState) {
	t.Helper()
	bz, err := json.Marshal(genesis)
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: 1,
		AppStateBytes: bz, ConsensusParams: simtestutil.DefaultConsensusParams})
	require.NoError(t, err)
}

func commitRepairsMarkersBlock(t *testing.T, app *zeroneapp.ZeroneApp) {
	t.Helper()
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
}

// requireExportedRepairsMarkers exports app and imports the export into a
// fresh app, requiring every marker to follow at both ends.
func requireExportedRepairsMarkers(t *testing.T, app *zeroneapp.ZeroneApp, want bool) {
	t.Helper()
	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	var exportedGenesis zeroneapp.GenesisState
	require.NoError(t, json.Unmarshal(exported.AppState, &exportedGenesis))
	for _, m := range app.ConsensusRepairsV1Markers() {
		require.Equal(t, want, repairsMarkerField(t, exportedGenesis, m), "%s export", m.Module)
	}

	imported := repairsMarkersApp(t)
	_, err = imported.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: exported.Height,
		AppStateBytes: exported.AppState, ConsensusParams: &exported.ConsensusParams})
	require.NoError(t, err)
	requireRepairsMarkers(t, imported, imported.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: exported.Height}), want)
}

// Through the real application (module manager, each module's own genesis
// codec, InitChain, commit, export and a second InitChain), a native genesis
// selects every consensus-repairs-v1 marker and an export carries it. A legacy
// genesis is now refused rather than silently opting into the new semantics.
func TestConsensusRepairsV1MarkersFollowGenesisThroughTheApp(t *testing.T) {
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
			for _, m := range source.ConsensusRepairsV1Markers() {
				require.True(t, repairsMarkerField(t, genesis, m), "%s default genesis selects %s", m.Module, m.Name)
			}
			if tc.strip {
				stripRepairsMarkerFields(t, source, genesis)
				bz, err := json.Marshal(genesis)
				require.NoError(t, err)
				_, err = source.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: 1, AppStateBytes: bz, ConsensusParams: simtestutil.DefaultConsensusParams})
				require.ErrorContains(t, err, "requires explicit")
				return
			}
			initRepairsMarkersChain(t, source, genesis)
			requireRepairsMarkers(t, source, source.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), tc.want)

			commitRepairsMarkersBlock(t, source)
			requireRepairsMarkers(t, source, source.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), tc.want)
			requireExportedRepairsMarkers(t, source, tc.want)
		})
	}
}

// The registry the carrier loops over lists exactly the ten
// consensus-repairs-v1 markers, each named by the genesis field that carries
// it. Enabling every entry on a chain that started without the fields selects
// every repair, idempotently, and an export then carries all of them.
func TestConsensusRepairsV1RegistryEnablesEveryMarker(t *testing.T) {
	app := repairsMarkersApp(t)
	var listed []string
	for _, m := range app.ConsensusRepairsV1Markers() {
		listed = append(listed, m.Module+"/"+m.Name)
	}
	require.Equal(t, []string{
		"zerone_gov/seat_runoff_ordering_enabled",
		"knowledge/reformulation_vindication_ordering_enabled",
		"capture_defense/integer_variance_enabled",
		"qualification/single_refund_enabled",
		"capture_challenge/challenge_accounting_enabled",
		"vesting_rewards/alignment_bank_supply_enabled",
		"zerone_ontology/active_domain_count_enabled",
		"emergency/halt_reason_scoping_enabled",
		"substrate_bridge/qualification_semantics_enabled",
		"ibcratelimit/refund_identity_enabled",
	}, listed)

	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsMarkers(t, app, ctx)
	requireRepairsMarkers(t, app, ctx, false)
	for range 2 {
		for _, m := range app.ConsensusRepairsV1Markers() {
			require.NoError(t, m.Enable(ctx), m.Module)
		}
		requireRepairsMarkers(t, app, ctx, true)
	}

	commitRepairsMarkersBlock(t, app)
	requireRepairsMarkers(t, app, app.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), true)
	requireExportedRepairsMarkers(t, app, true)
}

// Each registry entry is bound to its own marker and genesis field: from a
// chain with every repair off, enabling one entry turns on that entry's reader
// and that module's genesis field and nothing else, so a swapped Enable, reader
// or Name cannot pass. Every reader is gas-free whether its marker is absent or
// on, so a read added to an existing path cannot change gas before activation.
func TestConsensusRepairsV1RegistryEntriesAreIndependentAndGasFree(t *testing.T) {
	app := repairsMarkersApp(t)
	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	initRepairsMarkersChain(t, app, genesis)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1})
	seedAbsentRepairsMarkers(t, app, ctx)
	markers := app.ConsensusRepairsV1Markers()

	requireGasFree := func(ctx sdk.Context, m zeroneapp.ActivationMarker, want bool) {
		t.Helper()
		meter := storetypes.NewGasMeter(1_000_000)
		enabled, err := m.Enabled(ctx.WithGasMeter(meter))
		require.NoError(t, err, m.Module)
		require.Equal(t, want, enabled, m.Module)
		require.Zero(t, meter.GasConsumed(), "%s reader charged gas", m.Module)
	}

	for i, target := range markers {
		cached, _ := ctx.CacheContext()
		requireGasFree(cached, target, false)
		require.NoError(t, target.Enable(cached), target.Module)
		exported, err := app.ModuleManager.ExportGenesisForModules(cached, app.AppCodec(), nil)
		require.NoError(t, err)
		for j, m := range markers {
			requireGasFree(cached, m, i == j)
			require.Equal(t, i == j, repairsMarkerField(t, exported, m),
				"enabling %s/%s: %s/%s genesis field", target.Module, target.Name, m.Module, m.Name)
		}
	}
}
