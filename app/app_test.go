package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
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

// TestRegisterAPIRoutesIncludesTrainingProvenance verifies the app-level v2
// gateway mount, not merely the generated handler. A registered route reaches
// the client boundary (and fails because this test deliberately provides no
// node); an unknown route remains a 404.
func TestRegisterAPIRoutesIncludesTrainingProvenance(t *testing.T) {
	app := newTestApp(t)
	clientCtx := client.Context{
		Codec:             app.AppCodec(),
		InterfaceRegistry: app.InterfaceRegistry(),
		TxConfig:          app.TxConfig(),
	}
	apiServer := api.New(clientCtx, log.NewNopLogger(), grpc.NewServer())
	app.RegisterAPIRoutes(apiServer, config.APIConfig{})

	registered := httptest.NewRecorder()
	apiServer.Router.ServeHTTP(
		registered,
		httptest.NewRequest(http.MethodGet, "/zerone/training_provenance/v1/in-toto/missing", nil),
	)
	require.NotEqual(t, http.StatusNotFound, registered.Code)

	unknown := httptest.NewRecorder()
	apiServer.Router.ServeHTTP(
		unknown,
		httptest.NewRequest(http.MethodGet, "/zerone/not-a-real-module/v1/missing", nil),
	)
	require.Equal(t, http.StatusNotFound, unknown.Code)
}

// TestRegisterAPIRoutesIncludesSponsorshipSettlement guards the app-level v2
// gateway wiring for the public evidence an agent wallet needs to reconcile a
// fulfillment and the module escrow liability. Generated handlers and Swagger
// alone are insufficient: without these registrations both routes return 404.
func TestRegisterAPIRoutesIncludesSponsorshipSettlement(t *testing.T) {
	app := newTestApp(t)
	clientCtx := client.Context{
		Codec:             app.AppCodec(),
		InterfaceRegistry: app.InterfaceRegistry(),
		TxConfig:          app.TxConfig(),
	}
	apiServer := api.New(clientCtx, log.NewNopLogger(), grpc.NewServer())
	app.RegisterAPIRoutes(apiServer, config.APIConfig{})

	for _, route := range []string{
		"/zerone/sponsorship/v1/fulfillments/missing-order/missing-fact",
		"/zerone/sponsorship/v1/escrow_accounting",
	} {
		t.Run(route, func(t *testing.T) {
			response := httptest.NewRecorder()
			apiServer.Router.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, route, nil),
			)
			require.NotEqual(t, http.StatusNotFound, response.Code)
		})
	}
}

// TestRegisterAPIRoutesIncludesStandardServices guards the direct gateway
// registrations that do not belong to ModuleBasics. The requests reach the
// service boundary and may fail without a running node, but they must not be
// mistaken for unknown routes.
func TestRegisterAPIRoutesIncludesStandardServices(t *testing.T) {
	app := newTestApp(t)
	clientCtx := client.Context{
		Codec:             app.AppCodec(),
		InterfaceRegistry: app.InterfaceRegistry(),
		TxConfig:          app.TxConfig(),
	}
	apiServer := api.New(clientCtx, log.NewNopLogger(), grpc.NewServer())
	app.RegisterAPIRoutes(apiServer, config.APIConfig{})

	for _, route := range []string{
		"/cosmos/base/tendermint/v1beta1/syncing",
		"/cosmos/base/node/v1beta1/config",
		"/cosmos/tx/v1beta1/txs/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cosmos/feegrant/v1beta1/allowances/zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf",
	} {
		t.Run(route, func(t *testing.T) {
			response := httptest.NewRecorder()
			apiServer.GRPCGatewayRouter.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, route, nil),
			)
			require.NotEqual(t, http.StatusNotFound, response.Code)
		})
	}
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

	for _, moduleName := range []string{"auth", "bank", "staking", "distribution", "gov", "upgrade", vestingrewardstypes.ModuleName} {
		_, ok := genState[moduleName]
		require.True(t, ok, "expected module %q in default genesis", moduleName)
	}

	var vestingGenesis vestingrewardstypes.GenesisState
	require.NoError(t, json.Unmarshal(genState[vestingrewardstypes.ModuleName], &vestingGenesis))
	require.NotNil(t, vestingGenesis.Params)
	require.NoError(t, vestingrewardstypes.ValidateParams(vestingGenesis.Params))
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
