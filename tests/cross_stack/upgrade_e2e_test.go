package cross_stack_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// ─── Wave 10: end-to-end upgrade pipeline tests ─────────────────────────

// TestUpgrade_ChainVersionReportWellFormed — the introspection surface
// returns a sorted, complete module list plus the registered upgrade
// lineage. First guard: drift between the registered handlers and the
// self-described lineage would mask a missing upgrade, so we assert
// parity.
func TestUpgrade_ChainVersionReportWellFormed(t *testing.T) {
	h := NewTestHarness(t)

	report := h.App.BuildChainVersionReport()
	require.NotEmpty(t, report.Modules, "chain has registered modules")
	require.NotEmpty(t, report.KnownUpgrades, "at least one upgrade lineage entry")

	// Modules sorted by name deterministically.
	for i := 1; i < len(report.Modules); i++ {
		require.LessOrEqual(t, report.Modules[i-1].ModuleName, report.Modules[i].ModuleName,
			"module list must be name-sorted for deterministic consumption")
	}

	// Every advertised upgrade must have a handler registered. No drift.
	names := h.App.KnownUpgradeNames()
	for _, n := range names {
		require.True(t, h.App.UpgradeKeeper.HasHandler(n),
			"handler for %q must be registered to match the lineage entry", n)
	}

	// The report binds the current consensus releases for modules whose named
	// activation boundaries are pending.
	var sawKnowledge, sawLiquidityPool bool
	for _, m := range report.Modules {
		switch m.ModuleName {
		case "knowledge":
			sawKnowledge = true
			require.Equal(t, uint64(6), m.ConsensusVersion,
				"knowledge module advertises its current ConsensusVersion")
		case liquiditypooltypes.ModuleName:
			sawLiquidityPool = true
			require.Equal(t, uint64(4), m.ConsensusVersion,
				"liquiditypool module advertises the safety-v2 ConsensusVersion")
		}
	}
	require.True(t, sawKnowledge, "knowledge module appears in report")
	require.True(t, sawLiquidityPool, "liquiditypool module appears in report")
}

// TestUpgrade_V1ToV2MigrationPipeline — exercise the v1.0.1-testnet upgrade
// against the knowledge module downshifted from its current ConsensusVersion
// to v1. Exercises the full pipeline Migrate1to2 → Migrate2to3 → Migrate3to4
// in sequence. Other modules stay at their current version (no migration
// runs for them — we're not testing SDK-module migrations here, which have
// their own Cosmos SDK test coverage and require test fixtures this harness
// doesn't set up).
func TestUpgrade_V1ToV2MigrationPipeline(t *testing.T) {
	h := NewTestHarness(t)

	// Build fromVM: all modules at current, knowledge downshifted to v1.
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, ver := range current {
		fromVM[name] = ver
	}
	fromVM["knowledge"] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameTestnetV2, fromVM, h.Height())
	require.NoError(t, err, "v1.0.1-testnet handler completes without error")
	require.Equal(t, uint64(6), toVM["knowledge"],
		"knowledge module advances to its current ConsensusVersion (6) via full migration chain")

	// All migrations ran in sequence — each wrote its marker.
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v2_complete"),
		"v1→v2 migration marker proves Migrate1to2 ran")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v4_complete"),
		"v3→v4 migration marker proves Migrate3to4 ran mid-chain")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v5_complete"),
		"v4→v5 migration marker proves Migrate4to5 ran")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"),
		"v5→v6 migration marker proves Migrate5to6 ran at the end of the chain")

	// The v1.0.1 handler-level marker was written by the upgrade handler itself.
	handlerMarker := h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_v1.0.1")
	require.Equal(t, "migrated", handlerMarker,
		"handler-level marker proves the named upgrade handler executed")
}

// TestUpgrade_V3ToV4KnowledgeMigrationPipeline — exercise the
// v1.0.2-testnet upgrade, which in turn fires the knowledge v3→v4
// migration (TraceSchema backfill + v4 marker).
func TestUpgrade_V3ToV4KnowledgeMigrationPipeline(t *testing.T) {
	h := NewTestHarness(t)

	// Synthetic fromVM: knowledge at v3, everything else at current.
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, ver := range current {
		fromVM[name] = ver
	}
	fromVM["knowledge"] = 3 // downshift so v3→v4 migration fires

	// NO pre-seeded TraceSchema — the v4 migration must backfill it.
	_, seeded := h.KnowledgeKeeper.GetTraceSchema(h.Ctx)
	if seeded {
		t.Log("trace schema already present pre-upgrade; v4 migration will be a no-op on schema")
	}

	toVM, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameTestnetV3, fromVM, h.Height())
	require.NoError(t, err)
	require.Equal(t, uint64(6), toVM["knowledge"],
		"knowledge module is now at ConsensusVersion 6 post-migration (chain now extends 3→4→5→6)")

	// v4 + v5 migration markers prove Migrate3to4 and Migrate4to5 both ran.
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v4_complete"),
		"v4 migration marker present")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v5_complete"),
		"v5 migration marker present")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"),
		"v6 migration marker present")

	// TraceSchema is present post-upgrade.
	schema, ok := h.KnowledgeKeeper.GetTraceSchema(h.Ctx)
	require.True(t, ok, "TraceSchema is backfilled by v4 migration")
	require.Equal(t, uint64(1), schema.Version)

	// v1.0.2 handler-level marker.
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_v1.0.2"))
}

// TestUpgrade_UnknownHandlerRejected — calling an unregistered upgrade
// name must return a clear error, not silently succeed or panic.
func TestUpgrade_UnknownHandlerRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, "not-a-real-upgrade", module.VersionMap{}, h.Height())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no upgrade handler registered")
}

// TestUpgrade_MigrationMarkerIdempotent — writing the same marker twice
// is a no-op (idempotent); writing a DIFFERENT value for the same key is
// rejected without overwriting (first writer wins).
// TestUpgrade_CompassionCalibrationV1RefreshesScores drives the real
// compassion-calibration-v1 handler and asserts it refreshes a stored
// calibration score under the inconclusive-excluding formula (docs/COMPASSION.md
// commitment C2). A record whose honest inconclusive attempts had dragged its
// stored score down is lifted to its true decisive-accuracy score, and the
// migration marker is written.
func TestUpgrade_CompassionCalibrationV1RefreshesScores(t *testing.T) {
	h := NewTestHarness(t)

	addr := "zerone1compassionupgrade00000000000000aa"
	// 3 accepted + 7 inconclusive. Under the OLD formula the stored score was
	// 3/10 = 300_000; seed that stale value so we can prove the handler recomputes.
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address:             addr,
		TotalSubmissions:    10,
		Accepted:            3,
		Inconclusive:        7,
		CalibrationScoreBps: 300_000, // stale, old-formula value
	}))

	// Run the real upgrade handler through the full pipeline.
	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameCompassionCalibrationV1, fromVM, h.Height())
	require.NoError(t, err)

	// The 7 inconclusive attempts leave the denominator: 3/3 decisive = BPS.
	refreshed, found := h.KnowledgeKeeper.GetAgentCalibration(h.Ctx, addr)
	require.True(t, found)
	require.Equal(t, uint64(1_000_000), refreshed.CalibrationScoreBps,
		"handler must recompute the stale score under the inconclusive-excluding formula")

	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_compassion-calibration-v1"),
		"handler must write the compassion-calibration-v1 migration marker")
}

func TestUpgrade_MigrationMarkerIdempotent(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "alpha"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"))

	// Same value again — idempotent.
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "alpha"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"))

	// Different value — preserves original, does not error (warns via log).
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "beta"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"),
		"first writer wins: conflicting value does not overwrite")
}

// TestUpgrade_LineageParityWithHandlers — every entry in the lineage list
// has a registered handler, AND every registered handler appears in the
// lineage list. Parity prevents the chain from drifting into a state
// where a handler exists but is invisible to operators, or vice versa.
func TestUpgrade_LineageParityWithHandlers(t *testing.T) {
	h := NewTestHarness(t)
	lineage := h.App.BuildChainVersionReport().KnownUpgrades

	// Every advertised upgrade must have a handler.
	for _, entry := range lineage {
		require.True(t, h.App.UpgradeKeeper.HasHandler(entry.UpgradeName),
			"lineage entry %q advertises a handler; must be registered", entry.UpgradeName)
	}

	// Inverse — release constants with consensus/operator significance must be
	// visible in the lineage.
	lineageNames := h.App.KnownUpgradeNames()
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnet)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnetV2)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnetV3)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameDoctrineMetabolismExemptV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameSubstrateDedupeV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameConsolidationSafetyV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameLiquiditySafetyV2)

	// Operators must apply the already-pending consolidation boundary first.
	// Keep the advertised lineage in that same order.
	var consolidationIndex, liquidityIndex = -1, -1
	for i, name := range lineageNames {
		switch name {
		case zeroneapp.UpgradeNameConsolidationSafetyV1:
			consolidationIndex = i
		case zeroneapp.UpgradeNameLiquiditySafetyV2:
			liquidityIndex = i
		}
	}
	require.NotEqual(t, -1, consolidationIndex)
	require.NotEqual(t, -1, liquidityIndex)
	require.Less(t, consolidationIndex, liquidityIndex,
		"liquiditypool-safety-v2 must be advertised after consolidation-safety-v1")
}

// TestUpgrade_SubstrateDedupeV1SeedsAndArms drives the real substrate-dedupe-v1
// handler end-to-end: it must run RunMigrations + ReconcileModuleAccountPerms +
// SeedSourceRefs + WriteMigrationMarker without error, index a pre-existing
// settled attestation's source, and leave enforcement armed. Guards the whole
// wiring under the exact plan name a governance proposal would carry.
func TestUpgrade_SubstrateDedupeV1SeedsAndArms(t *testing.T) {
	h := NewTestHarness(t)

	// A settled attestation that predates the wall — the seed must index its
	// source so a post-upgrade replay is blocked.
	link := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId: "agenttool-invocation-v1",
			SourceId:  "inv-preupgrade",
		},
	}
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId: "att-1-1",
		AdapterId:     "agenttool-invocation-v1",
		Submitter:     "zerone1preupgradesubmitter000000000000aa",
		Status:        substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		Link:          link,
	}))

	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameSubstrateDedupeV1, fromVM, h.Height())
	require.NoError(t, err)

	holder, taken := h.SubstrateBridgeKeeper.GetSourceRef(h.Ctx, "agenttool-invocation-v1", "inv-preupgrade")
	require.True(t, taken, "handler must seed the pre-existing attestation's source")
	require.Equal(t, "att-1-1", holder)
	require.True(t, h.SubstrateBridgeKeeper.IsDedupeArmed(h.Ctx), "handler must arm enforcement")
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_substrate-dedupe-v1"),
		"handler must write the substrate-dedupe-v1 migration marker")
}

// TestUpgrade_AgenttoolSeamV1DeclaresAxisBounds drives the real agenttool-seam-v1
// handler end-to-end under the exact plan name a governance proposal would carry.
// The adapter is written in the genesis shape of agenttool-invocation-v1 — active,
// AxisBounds nil — which is the state that let recursion weight through unbounded.
// After the handler, the ceiling must exist, and a weighted claim must be refused
// by the same code path a real submission takes.
func TestUpgrade_AgenttoolSeamV1DeclaresAxisBounds(t *testing.T) {
	h := NewTestHarness(t)

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId: "agenttool-invocation-v1",
		Status:    substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		// AxisBounds nil — exactly how genesis seeded it.
	}))

	before, found := h.SubstrateBridgeKeeper.GetAdapter(h.Ctx, "agenttool-invocation-v1")
	require.True(t, found)
	require.Nil(t, before.AxisBounds, "precondition: the drain state must be reproduced")

	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameAgenttoolSeamV1, fromVM, h.Height())
	require.NoError(t, err)

	after, found := h.SubstrateBridgeKeeper.GetAdapter(h.Ctx, "agenttool-invocation-v1")
	require.True(t, found)
	require.NotNil(t, after.AxisBounds, "handler must leave the adapter declaring a ceiling")

	// The drain, attempted through the real validation path.
	drain := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId: "agenttool-invocation-v1",
			SourceId:  "inv-drain",
		},
		RecursionWeight: &substratebridgetypes.AxisProjection{
			AxisSubstrate: ^uint64(0), AxisVerification: ^uint64(0),
			AxisClassification: ^uint64(0), AxisAttribution: ^uint64(0),
			AxisTooling: ^uint64(0), AxisInterface: ^uint64(0),
		},
	}
	substrateParams := substratebridgetypes.DefaultParams()
	// ErrAxisOverflow, not ErrAdapterAxisBoundsUnset: after the migration this
	// adapter *does* declare a ceiling, so the claim is refused for exceeding it.
	// The two errors mark the two halves of the fix — the migration closes the
	// drain for adapters that already exist, the gate closes it for any adapter
	// registered without bounds later. This test exercises the first.
	require.ErrorIs(t, h.SubstrateBridgeKeeper.ValidateLink(h.Ctx, drain, substrateParams),
		substratebridgetypes.ErrAxisOverflow,
		"a weighted claim must not survive the upgrade")

	// The live relay's shape — source only, no weight — must still pass.
	relayShaped := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId:   "agenttool-invocation-v1",
			SourceId:    "inv-normal",
			ContentHash: make([]byte, 32),
		},
	}
	require.NoError(t, h.SubstrateBridgeKeeper.ValidateLink(h.Ctx, relayShaped, substrateParams),
		"the upgrade must not stall the live bridge")

	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_agenttool-seam-v1"),
		"handler must write the agenttool-seam-v1 migration marker")
}

func TestUpgrade_ConsolidationSafetyV1RecordsActivationBoundary(t *testing.T) {
	h := NewTestHarness(t)

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["knowledge"] = 5
	fromVM["claiming_pot"] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(6), toVM["knowledge"])
	require.Equal(t, uint64(2), toVM["claiming_pot"])
	require.Equal(t, "true",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"),
		"knowledge migration marker proves the module activation boundary ran")
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_consolidation-safety-v1"),
		"handler marker proves the named upgrade ran")
}

func TestUpgrade_ConsolidationSafetyV1CannotBeBlockedByAnUnrelatedBallot(t *testing.T) {
	h := NewTestHarness(t)
	h.GovKeeper.SetLIP(h.Ctx, &zeronegovtypes.LIP{
		Id:    "LIP-live-ballot",
		Stage: zeronegovtypes.StatusVoting,
	})

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["knowledge"] = 5
	fromVM["claiming_pot"] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err,
		"a permissionless unrelated ballot must not be able to halt a scheduled upgrade")
	require.Equal(t, uint64(6), toVM["knowledge"])
	require.Equal(t, uint64(2), toVM["claiming_pot"])
	require.Equal(t, "true",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"))
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_consolidation-safety-v1"))
	lip, found := h.GovKeeper.GetLIP(h.Ctx, "LIP-live-ballot")
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusVoting, lip.Stage,
		"the upgrade must not rewrite an unrelated ballot")
}

// TestUpgrade_LiquiditySafetyV2AfterConsolidationIsIdempotent binds the
// production sequencing rule. consolidation-safety-v1 runs migrations first
// and may therefore advance liquiditypool v3→v4. The later, dedicated
// liquiditypool-safety-v2 handler must still succeed, keep the module at v4,
// reconcile permissions, and record its own readiness checkpoint.
func TestUpgrade_LiquiditySafetyV2AfterConsolidationIsIdempotent(t *testing.T) {
	h := NewTestHarness(t)

	current := h.App.CurrentModuleVersionMap()
	require.Equal(t, uint64(4), current[liquiditypooltypes.ModuleName],
		"test requires the approved liquiditypool consensus v4 binary")

	// Reproduce the v3 params shape: zero meant unlimited and neither
	// creation-admission field existed on the wire.
	legacyParams := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	legacyParams.MaxPools = 0
	legacyParams.AllowedPoolDenoms = nil
	legacyParams.PoolCreators = nil
	h.App.LiquidityPoolKeeper.SetParams(h.Ctx, legacyParams)

	beforeConsolidation := make(module.VersionMap, len(current))
	for name, version := range current {
		beforeConsolidation[name] = version
	}
	beforeConsolidation[liquiditypooltypes.ModuleName] = 3

	afterConsolidation, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		beforeConsolidation,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), afterConsolidation[liquiditypooltypes.ModuleName],
		"the earlier RunMigrations may already advance liquiditypool to v4")
	migratedParams := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	require.Equal(t, uint64(16), migratedParams.MaxPools,
		"v4 migration must replace the legacy unlimited pool cap")
	require.Empty(t, migratedParams.AllowedPoolDenoms,
		"v4 migration must leave asset admission fail-closed")
	require.Empty(t, migratedParams.PoolCreators,
		"v4 migration must leave creator admission fail-closed")

	afterLiquidityReadiness, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameLiquiditySafetyV2,
		afterConsolidation,
		h.Height()+1,
	)
	require.NoError(t, err,
		"the named liquidity readiness checkpoint must be safe after v4 already migrated")
	require.Equal(t, uint64(4), afterLiquidityReadiness[liquiditypooltypes.ModuleName])
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_liquiditypool-safety-v2"),
		"the dedicated handler marker proves the later readiness checkpoint ran")
}

// TestUpgrade_LiquiditySafetyV2ReconcilesModulePermissions proves the new
// handler retains the permanent safety step shared by all release handlers.
func TestUpgrade_LiquiditySafetyV2ReconcilesModulePermissions(t *testing.T) {
	h := NewTestHarness(t)

	moduleAccount := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.NotNil(t, moduleAccount)

	// Reproduce an old stored account whose permissions drifted from maccPerms.
	h.AccountKeeper.SetModuleAccount(h.Ctx, authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			moduleAccount.GetAddress(),
			nil,
			moduleAccount.GetAccountNumber(),
			moduleAccount.GetSequence(),
		),
		liquiditypooltypes.ModuleName,
	))

	drifted := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.Empty(t, drifted.GetPermissions(), "precondition: stored permissions drifted")

	fromVM := h.App.CurrentModuleVersionMap()
	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameLiquiditySafetyV2,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(4), toVM[liquiditypooltypes.ModuleName])

	reconciled := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.ElementsMatch(t,
		[]string{authtypes.Minter, authtypes.Burner},
		reconciled.GetPermissions(),
		"handler must restore the permissions declared by app maccPerms")
}
