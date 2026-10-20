package cross_stack_test

import (
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

	zeroneapp "github.com/zerone-chain/zerone/app"
	capturechallengetypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
	capturedefensetypes "github.com/zerone-chain/zerone/x/capture_defense/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	ontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const repairsMarkersChainID = "zerone-consensus-repairs-v1-markers"

// repairsMarker is one consensus-repairs-v1 activation marker: the module's
// genesis key, its GenesisState field, and the keeper reader.
type repairsMarker struct {
	module  string
	field   string
	enabled func(*zeroneapp.ZeroneApp, sdk.Context) (bool, error)
}

func consensusRepairsV1Markers() []repairsMarker {
	return []repairsMarker{
		{govtypes.ModuleName, "seat_runoff_ordering_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.ZeroneGovKeeper.SeatRunoffOrderingEnabled(ctx)
		}},
		{knowledgetypes.ModuleName, "reformulation_vindication_ordering_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.KnowledgeKeeper.ReformulationVindicationOrderingEnabled(ctx)
		}},
		{capturedefensetypes.ModuleName, "integer_variance_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.CaptureDefenseKeeper.IntegerVarianceEnabled(ctx)
		}},
		{qualificationtypes.ModuleName, "single_refund_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.QualificationKeeper.SingleRefundEnabled(ctx)
		}},
		{capturechallengetypes.ModuleName, "challenge_accounting_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.CaptureChallengeKeeper.ChallengeAccountingEnabled(ctx)
		}},
		{vestingrewardstypes.ModuleName, "alignment_bank_supply_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.VestingRewardsKeeper.AlignmentBankSupplyEnabled(ctx)
		}},
		{ontologytypes.ModuleName, "active_domain_count_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.ZeroneOntologyKeeper.ActiveDomainCountEnabled(ctx)
		}},
		{emergencytypes.ModuleName, "halt_reason_scoping_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.EmergencyKeeper.HaltReasonScopingEnabled(ctx)
		}},
		{substratebridgetypes.ModuleName, "qualification_semantics_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.SubstrateBridgeKeeper.QualificationSemanticsEnabled(ctx)
		}},
		{ibcratelimittypes.ModuleName, "refund_identity_enabled", func(app *zeroneapp.ZeroneApp, ctx sdk.Context) (bool, error) {
			return app.IBCRateLimitKeeper.RefundIdentityEnabled(ctx)
		}},
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
func repairsMarkerField(t *testing.T, genesis zeroneapp.GenesisState, m repairsMarker) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(genesis[m.module], &fields), m.module)
	value, ok := fields[m.field]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on), m.module)
	return on
}

func requireRepairsMarkers(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context, want bool) {
	t.Helper()
	for _, m := range consensusRepairsV1Markers() {
		enabled, err := m.enabled(app, ctx)
		require.NoError(t, err, m.module)
		require.Equal(t, want, enabled, m.module)
	}
}

// Through the real application (module manager, each module's own genesis
// codec, InitChain, commit, export and a second InitChain), a native genesis
// selects every consensus-repairs-v1 marker and an export carries it, while a
// genesis written before the fields existed starts and stays with all of them
// off. The binary alone selects nothing.
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
			for _, m := range consensusRepairsV1Markers() {
				require.True(t, repairsMarkerField(t, genesis, m), "%s default genesis selects %s", m.module, m.field)
				if tc.strip {
					var fields map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(genesis[m.module], &fields))
					delete(fields, m.field)
					stripped, err := json.Marshal(fields)
					require.NoError(t, err)
					genesis[m.module] = stripped
				}
			}
			bz, err := json.Marshal(genesis)
			require.NoError(t, err)
			_, err = source.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: 1,
				AppStateBytes: bz, ConsensusParams: simtestutil.DefaultConsensusParams})
			require.NoError(t, err)
			requireRepairsMarkers(t, source, source.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), tc.want)

			_, err = source.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
			require.NoError(t, err)
			_, err = source.Commit()
			require.NoError(t, err)
			requireRepairsMarkers(t, source, source.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1}), tc.want)

			exported, err := source.ExportAppStateAndValidators(false, nil, nil)
			require.NoError(t, err)
			var exportedGenesis zeroneapp.GenesisState
			require.NoError(t, json.Unmarshal(exported.AppState, &exportedGenesis))
			for _, m := range consensusRepairsV1Markers() {
				require.Equal(t, tc.want, repairsMarkerField(t, exportedGenesis, m), "%s export", m.module)
			}

			imported := repairsMarkersApp(t)
			_, err = imported.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID, InitialHeight: exported.Height,
				AppStateBytes: exported.AppState, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			requireRepairsMarkers(t, imported, imported.NewContextLegacy(false, cmtproto.Header{ChainID: repairsMarkersChainID, Height: exported.Height}), tc.want)
		})
	}
}
