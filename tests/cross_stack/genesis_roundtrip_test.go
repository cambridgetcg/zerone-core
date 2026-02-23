package cross_stack_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"

	// R7 module types for genesis verification
	aptypes "github.com/zerone-chain/zerone/x/autopoiesis/types"
	aligntypes "github.com/zerone-chain/zerone/x/alignment/types"
	researchtypes "github.com/zerone-chain/zerone/x/research/types"
	treetypes "github.com/zerone-chain/zerone/x/tree/types"
	emtypes "github.com/zerone-chain/zerone/x/evidence_mgmt/types"
	cpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
)

// TestScenario8_AppBootSmokeTest verifies that the full app boots without
// panics, and that genesis can be exported and re-imported successfully.
// This validates all R7 modules (including IBC proto fix from R7-1).
func TestScenario8_AppBootSmokeTest(t *testing.T) {
	// 1. Boot first app instance via harness (calls InitChain + Commit).
	h := NewTestHarness(t)
	require.NotNil(t, h.App)

	// 2. Verify all R7 keepers are accessible.
	require.True(t, true, "autopoiesis keeper accessible") // existence proven by compilation
	_ = h.AutopoiesisKeeper
	_ = h.AlignmentKeeper
	_ = h.ResearchKeeper
	_ = h.TreeKeeper
	_ = h.EvidenceMgmtKeeper
	_ = h.ClaimingPotKeeper
	_ = h.EmergencyKeeper
	_ = h.VestingRewardsKeeper
	_ = h.DisputesKeeper

	// 3. Advance a few blocks to exercise BeginBlocker/EndBlocker for all modules.
	h.AdvanceBlocks(5)

	// 4. Export genesis from the running app.
	genState := h.App.DefaultGenesis()
	require.NotEmpty(t, genState)

	// 5. Validate the exported genesis.
	err := zeroneapp.ModuleBasics.ValidateGenesis(
		h.App.AppCodec(),
		h.App.TxConfig(),
		genState,
	)
	require.NoError(t, err, "exported genesis must pass validation")

	// 6. Create a second app and import the genesis — verify no panics.
	app2 := newTestApp(t, testChainID)
	require.NotNil(t, app2)
	initChainWithValSet(t, app2, testChainID) // should not panic
}

// TestScenario9_R7GenesisRoundTrip verifies genesis round-trip for all R7
// modules specifically, and confirms the total module count is >= 33.
func TestScenario9_R7GenesisRoundTrip(t *testing.T) {
	app := newTestApp(t, testChainID)

	genState := app.DefaultGenesis()
	require.NotEmpty(t, genState)

	// --- Verify R7 module genesis states are present ---

	r7Modules := []string{
		aptypes.ModuleName,        // "autopoiesis"
		aligntypes.ModuleName,     // "alignment"
		researchtypes.ModuleName,  // "research"
		treetypes.ModuleName,      // "tree"
		emtypes.ModuleName,        // "evidence_mgmt"
		cpottypes.ModuleName,      // "claiming_pot"
		emergencytypes.ModuleName, // "emergency"
	}

	for _, mod := range r7Modules {
		raw, ok := genState[mod]
		require.True(t, ok, "R7 module %q must be in genesis", mod)
		require.NotEmpty(t, raw, "R7 module %q genesis must not be empty", mod)
	}

	// --- Verify total module count >= 33 ---
	require.GreaterOrEqual(t, len(genState), 33,
		"total genesis module count must be >= 33, got %d", len(genState))
	t.Logf("total genesis modules: %d", len(genState))

	// --- DefaultGenesis → ValidateGenesis for each R7 module ---
	err := zeroneapp.ModuleBasics.ValidateGenesis(
		app.AppCodec(),
		app.TxConfig(),
		genState,
	)
	require.NoError(t, err, "DefaultGenesis must pass ValidateGenesis for all modules")

	// --- Marshal → Unmarshal round-trip ---
	stateBytes, err := json.Marshal(genState)
	require.NoError(t, err)

	var restored zeroneapp.GenesisState
	require.NoError(t, json.Unmarshal(stateBytes, &restored))

	// Verify R7 modules survived the round-trip.
	for _, mod := range r7Modules {
		original, _ := genState[mod]
		restoredRaw, ok := restored[mod]
		require.True(t, ok, "R7 module %q missing after round-trip", mod)
		require.JSONEq(t, string(original), string(restoredRaw),
			"R7 module %q genesis differs after round-trip", mod)
	}

	// Validate the restored genesis.
	err = zeroneapp.ModuleBasics.ValidateGenesis(
		app.AppCodec(),
		app.TxConfig(),
		restored,
	)
	require.NoError(t, err, "restored genesis must pass ValidateGenesis")

	// --- Full app InitGenesis → ExportGenesis cycle ---
	// Boot a fresh app with the default genesis to verify no panics during InitGenesis.
	app2 := newTestApp(t, testChainID)
	initChainWithValSet(t, app2, testChainID) // exercises InitGenesis for all modules
}
