package app_test

import (
	"encoding/json"
	"sort"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

// newTestApp creates a ZeroneApp wired to an in-memory database.
func newTestApp(t *testing.T) *zeroneapp.ZeroneApp {
	t.Helper()
	db := dbm.NewMemDB()
	app := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		db,
		nil,  // traceStore
		true, // loadLatest
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	)
	return app
}

// TestNewZeroneApp verifies the application can be constructed without panicking
// and that all module keepers are initialized.
func TestNewZeroneApp(t *testing.T) {
	app := newTestApp(t)
	require.NotNil(t, app)
	require.NotNil(t, app.AccountKeeper)
	require.NotNil(t, app.BankKeeper)
	require.NotNil(t, app.StakingKeeper)
	require.NotNil(t, app.DistrKeeper)
	require.NotNil(t, app.GovKeeper)
	require.NotNil(t, app.IBCKeeper)
	require.NotNil(t, app.UpgradeKeeper)
}

// TestDefaultGenesis verifies the default genesis state is valid JSON and
// contains the expected module keys.
func TestDefaultGenesis(t *testing.T) {
	app := newTestApp(t)

	genState := app.DefaultGenesis()
	require.NotEmpty(t, genState)

	bz, err := json.Marshal(genState)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	for _, moduleName := range []string{
		"auth",
		"bank",
		"staking",
		"distribution",
		"gov",
		"upgrade",
		"training_provenance",
		"trust_score",
	} {
		_, ok := genState[moduleName]
		require.True(t, ok, "expected module %q in default genesis", moduleName)
	}
}

// TestDefaultGenesisModulesHaveExporters prevents a module from contributing
// default genesis JSON without satisfying one of the module manager's stateful
// genesis interfaces. Such a module boots successfully but disappears from a
// later `zeroned export`, making the exported genesis fail validation.
func TestDefaultGenesisModulesHaveExporters(t *testing.T) {
	app := newTestApp(t)
	genState := app.DefaultGenesis()

	moduleNames := make([]string, 0, len(genState))
	for moduleName := range genState {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	for _, moduleName := range moduleNames {
		t.Run(moduleName, func(t *testing.T) {
			appModule, ok := app.ModuleManager.Modules[moduleName]
			require.True(t, ok, "default-genesis module is not registered")

			_, hasCoreGenesis := appModule.(appmodule.HasGenesis)
			_, hasLegacyGenesis := appModule.(module.HasGenesis)
			_, hasABCIGenesis := appModule.(module.HasABCIGenesis)
			require.True(t, hasCoreGenesis || hasLegacyGenesis || hasABCIGenesis,
				"module contributes default genesis but has no compatible exporter")
		})
	}
}

func TestReadOnlyModulesExportGenesis(t *testing.T) {
	app := newTestApp(t)
	moduleNames := []string{"training_provenance", "trust_score"}

	exported, err := app.ModuleManager.ExportGenesisForModules(
		app.NewContext(true),
		app.AppCodec(),
		moduleNames,
	)
	require.NoError(t, err)
	require.Len(t, exported, len(moduleNames))

	defaults := app.DefaultGenesis()
	for _, moduleName := range moduleNames {
		t.Run(moduleName, func(t *testing.T) {
			raw, ok := exported[moduleName]
			require.True(t, ok, "module manager omitted genesis export")
			require.True(t, json.Valid(raw), "module manager returned invalid genesis JSON")
			require.JSONEq(t, string(defaults[moduleName]), string(raw))
		})
	}
}

// TestExportGenesis verifies the default genesis JSON round-trips correctly
// through marshal/unmarshal without data loss.
func TestExportGenesis(t *testing.T) {
	app := newTestApp(t)

	genState := app.DefaultGenesis()
	require.NotEmpty(t, genState)

	// Marshal → Unmarshal round-trip.
	bz, err := json.Marshal(genState)
	require.NoError(t, err)

	var restored map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(bz, &restored))

	// Same number of modules present.
	require.Equal(t, len(genState), len(restored))

	// Core modules are present.
	for _, mod := range []string{"auth", "bank", "staking", "distribution", "gov", "upgrade", "ibc"} {
		_, ok := restored[mod]
		require.True(t, ok, "expected module %q after round-trip", mod)
	}
}

// TestZRNDenomMetadata verifies that the ZRN denomination metadata structure
// is correct and that the bank genesis state can be updated with it.
func TestZRNDenomMetadata(t *testing.T) {
	app := newTestApp(t)

	// Parse the default bank genesis.
	genState := app.DefaultGenesis()
	bankRaw, ok := genState["bank"]
	require.True(t, ok, "bank module must be in default genesis")

	var bankState banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(bankRaw, &bankState)

	// The default bank genesis does not have uzrn metadata yet
	// (it is injected by InitChainer at chain boot).
	// Verify we can add it programmatically, which is what InitChainer does.
	uzrnMeta := banktypes.Metadata{
		Description: "The native staking and governance token of Zerone",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: "uzrn", Exponent: 0, Aliases: []string{"microzrn"}},
			{Denom: "mzrn", Exponent: 3, Aliases: []string{"millizrn"}},
			{Denom: "zrn", Exponent: 6, Aliases: nil},
		},
		Base:    "uzrn",
		Display: "zrn",
		Name:    "Zerone",
		Symbol:  "ZRN",
	}

	require.NoError(t, uzrnMeta.Validate())
	require.Equal(t, "uzrn", uzrnMeta.Base)
	require.Equal(t, "ZRN", uzrnMeta.Symbol)
	require.Len(t, uzrnMeta.DenomUnits, 3)

	// Coin creation with the registered denom.
	coin := sdk.NewInt64Coin("uzrn", 1_000_000)
	require.Equal(t, "uzrn", coin.Denom)
}
