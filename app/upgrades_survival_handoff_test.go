package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/survivalmigration"
	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Keep this earlier release's target frozen at knowledge7. Current production
// startup correctly refuses this state except under the next exact H-1 plan.
type archivedSurvivalKnowledgeModule struct{ knowledgemodule.AppModule }

func (archivedSurvivalKnowledgeModule) ConsensusVersion() uint64 { return 7 }

func freezeSurvivalCompiledTarget(app *ZeroneApp) {
	app.ModuleManager.Modules["knowledge"] = archivedSurvivalKnowledgeModule{app.ModuleManager.Modules["knowledge"].(knowledgemodule.AppModule)}
}

func newFrozenSurvivalFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	app, ctx, db := newAccountingAuthorityFixture(t)
	freezeSurvivalCompiledTarget(app)
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
	ctx.KVStore(app.keys["knowledge"]).Delete([]byte(knowledgekeeper.RecordIntegrityEnabledStoreKey))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, survivalHandoffTargetVersionMap()))
	return app, ctx, db
}

func survivalHandoffSourceFixture(t *testing.T) (*ZeroneApp, sdk.Context) {
	t.Helper()
	app, ctx, _ := newFrozenSurvivalFixture(t)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, survivalHandoffSourceVersionMap()))
	return app, ctx
}

func survivalHandoffPendingFixture(t *testing.T, app *ZeroneApp, ctx sdk.Context) ([]byte, []byte, []byte) {
	t.Helper()
	pr := knowledgekeeper.SurvivalPendingReward{
		ClaimId: "handoff-claim", FactId: "handoff-fact", Recipient: sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String(),
		Amount: "123", Category: "FORMAL", Deadline: 100,
	}
	require.NoError(t, app.KnowledgeKeeper.SetSurvivalPendingReward(ctx, pr))
	key := append(bytes.Clone(knowledgetypes.SurvivalPendingRewardPrefix), []byte(pr.FactId)...)
	deadline := append(bytes.Clone(knowledgetypes.SurvivalDeadlineIndexPrefix), make([]byte, 8)...)
	binary.BigEndian.PutUint64(deadline[len(knowledgetypes.SurvivalDeadlineIndexPrefix):], pr.Deadline)
	deadline = append(deadline, []byte(pr.FactId)...)
	store := ctx.KVStore(app.keys["knowledge"])
	before := bytes.Clone(store.Get(key))
	store.Delete(deadline) // An old CHALLENGED sweep could leave this shape.
	return key, before, deadline
}

func TestSurvivalHandoffOwnsOnlyItsExactVersionPair(t *testing.T) {
	app, _, _ := newFrozenSurvivalFixture(t)
	source, target := survivalHandoffSourceVersionMap(), survivalHandoffTargetVersionMap()
	require.Equal(t, target, app.CurrentModuleVersionMap())
	require.Equal(t, uint64(6), accountingAuthorityTargetVersionMap()["knowledge"])
	require.Equal(t, uint64(2), accountingAuthorityTargetVersionMap()["vesting_rewards"])
	require.Equal(t, uint64(6), accountingAuthoritySourceVersionMap()["knowledge"])
	require.Contains(t, app.KnownUpgradeNames(), UpgradeNameSurvivalRewardHandoffV1)
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameSurvivalRewardHandoffV1))
	for _, name := range append(app.KnownUpgradeNames(), "unrelated") {
		err := requireSurvivalHandoffTransitionOwner(name, source, target)
		if name == UpgradeNameSurvivalRewardHandoffV1 {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "sole owner")
		}
	}
	for _, pair := range [][2]uint64{{6, 3}, {7, 2}, {8, 3}, {0, 0}} {
		invalid := module.VersionMap{"knowledge": pair[0], "vesting_rewards": pair[1]}
		require.Error(t, requireSurvivalHandoffTransitionOwner(UpgradeNameSurvivalRewardHandoffV1, invalid, target))
	}
	require.Error(t, requireSurvivalHandoffTransitionOwner(UpgradeNameSurvivalRewardHandoffV1, module.VersionMap{"knowledge": 6}, target))
	require.Error(t, requireSurvivalHandoffTransitionOwner(UpgradeNameSurvivalRewardHandoffV1, target, source))
}

func TestSurvivalHandoffModulesRefuseUnownedBroadMigration(t *testing.T) {
	app, ctx := survivalHandoffSourceFixture(t)
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, survivalHandoffSourceVersionMap())
	require.ErrorContains(t, err, "explicit")
	require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v7_complete"))
	_, err = survivalmigration.WithOwner(ctx, UpgradeNameAccountingAuthorityV1)
	require.Error(t, err)
}

func TestSurvivalHandoffCommittedBoundaryRepairsOnlyDerivedIndex(t *testing.T) {
	app, ctx, db := newFrozenSurvivalFixture(t)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, survivalHandoffSourceVersionMap()))
	key, pending, deadline := survivalHandoffPendingFixture(t, app, ctx)
	// Existing handoffs and progress are history; the upgrade must not reset or
	// duplicate them while preserving a different still-pending obligation.
	recipient := sdk.AccAddress(bytes.Repeat([]byte{8}, 20)).String()
	schedule, err := app.VestingRewardsKeeper.CreateVestingSchedule(ctx, "previous-claim", "previous-fact", recipient, "456", vestingtypes.CategoryFormalProof, vestingtypes.SourceVerification)
	require.NoError(t, err)
	schedule.ReleasedAmount, schedule.ClaimableAmount = "10", "20"
	app.VestingRewardsKeeper.SetVestingSchedule(ctx, schedule)
	scheduleKey := append(bytes.Clone(vestingtypes.VestingScheduleKeyPrefix), []byte(schedule.Id)...)
	scheduleBefore := bytes.Clone(ctx.KVStore(app.keys[vestingtypes.StoreKey]).Get(scheduleKey))
	stale := append(bytes.Clone(knowledgetypes.SurvivalDeadlineIndexPrefix), []byte("obsolete-index")...)
	ctx.KVStore(app.keys["knowledge"]).Set(stale, []byte{1})
	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
	beforeMetadata, err := app.accountingAuthorityGenesisMetadata(ctx, 0)
	require.NoError(t, err)
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameSurvivalRewardHandoffV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Error(t, app.ValidateAccountingAuthorityStartup(), "new semantics must not execute an early source block")
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	wrong := plan
	wrong.Info = "unreviewed"
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, wrong))
	require.Error(t, app.ValidateAccountingAuthorityStartup())
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	require.False(t, ctx.KVStore(app.keys["knowledge"]).Has(deadline), "outer block cache must remain atomic")
	write()
	require.Equal(t, pending, ctx.KVStore(app.keys["knowledge"]).Get(key))
	require.Equal(t, []byte{1}, ctx.KVStore(app.keys["knowledge"]).Get(deadline))
	require.False(t, ctx.KVStore(app.keys["knowledge"]).Has(stale))
	require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Len(t, app.VestingRewardsKeeper.GetAllVestingSchedules(ctx), 1)
	require.Equal(t, scheduleBefore, ctx.KVStore(app.keys[vestingtypes.StoreKey]).Get(scheduleKey))
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, survivalHandoffTargetVersionMap(), vm)
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	afterMetadata, err := app.accountingAuthorityGenesisMetadata(ctx, plan.Height)
	require.NoError(t, err)
	require.Equal(t, beforeMetadata, afterMetadata, "the handoff does not relabel accounting lineage")
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan), "the transition cannot replay")

	// The real loadLatest constructor admits the committed target without a
	// local upgrade-info file. Export/import retains both sides of the handoff
	// and explicitly retains accounting origin without inventing done heights.
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID(ctx.ChainID()))
	freezeSurvivalCompiledTarget(restarted)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
	restartCtx := restarted.NewUncachedContext(true, cmtproto.Header{Height: plan.Height, ChainID: ctx.ChainID()})
	require.Equal(t, pending, restartCtx.KVStore(restarted.keys["knowledge"]).Get(key))
	require.Equal(t, scheduleBefore, restartCtx.KVStore(restarted.keys[vestingtypes.StoreKey]).Get(scheduleKey))
	genesis, err := restarted.ModuleManager.ExportGenesis(restartCtx, restarted.appCodec)
	require.NoError(t, err)
	genesis[accountingAuthorityGenesisKey], err = json.Marshal(afterMetadata)
	require.NoError(t, err)
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("imported-handoff"))
	freezeSurvivalCompiledTarget(imported)
	require.NoError(t, imported.LoadLatestVersion())
	importCtx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "imported-handoff"})
	_, err = imported.InitChainer(importCtx, &abci.RequestInitChain{ChainId: "imported-handoff", AppStateBytes: raw})
	require.NoError(t, err)
	require.Equal(t, pending, importCtx.KVStore(imported.keys["knowledge"]).Get(key))
	require.Equal(t, []byte{1}, importCtx.KVStore(imported.keys["knowledge"]).Get(deadline))
	require.Equal(t, scheduleBefore, importCtx.KVStore(imported.keys[vestingtypes.StoreKey]).Get(scheduleKey))
	done, err := imported.UpgradeKeeper.GetDoneHeight(importCtx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done, "genesis import is not a witnessed application of the upgrade")
	imported.CommitMultiStore().Commit()
	require.NoError(t, imported.ValidateAccountingAuthorityStartup())
}

func TestSurvivalHandoffValidationFailureRollsBackIndexAndVersions(t *testing.T) {
	app, ctx := survivalHandoffSourceFixture(t)
	key, pending, deadline := survivalHandoffPendingFixture(t, app, ctx)
	// An orphan primary/index cannot be repaired by assigning an inferred owner.
	orphan := append(bytes.Clone(vestingtypes.ClaimRecordKeyPrefix), []byte("orphan-claim")...)
	ctx.KVStore(app.keys[vestingtypes.StoreKey]).Set(orphan, []byte("missing-schedule"))
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameSurvivalRewardHandoffV1, Height: latest + 1}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	beforeEvents := len(ctx.EventManager().Events())
	err := app.UpgradeKeeper.ApplyUpgrade(ctx, plan)
	require.ErrorContains(t, err, "vesting")
	require.Equal(t, pending, ctx.KVStore(app.keys["knowledge"]).Get(key))
	require.False(t, ctx.KVStore(app.keys["knowledge"]).Has(deadline))
	require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v7_complete"))
	require.Equal(t, beforeEvents, len(ctx.EventManager().Events()))
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, survivalHandoffSourceVersionMap(), vm)
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestSurvivalHandoffRefusesUnknownMapAndPlan(t *testing.T) {
	for _, change := range []struct {
		name string
		fn   func(module.VersionMap, *upgradetypes.Plan)
	}{
		{"missing", func(vm module.VersionMap, _ *upgradetypes.Plan) { delete(vm, "tokens") }},
		{"extra", func(vm module.VersionMap, _ *upgradetypes.Plan) { vm["unknown"] = 1 }},
		{"mixed", func(vm module.VersionMap, _ *upgradetypes.Plan) { vm["knowledge"] = 7 }},
		{"wrong name", func(_ module.VersionMap, p *upgradetypes.Plan) { p.Name = UpgradeNameAccountingAuthorityV1 }},
		{"info", func(_ module.VersionMap, p *upgradetypes.Plan) { p.Info = "{}" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			app, ctx := survivalHandoffSourceFixture(t)
			app.CommitMultiStore().Commit()
			vm := survivalHandoffSourceVersionMap()
			plan := upgradetypes.Plan{Name: UpgradeNameSurvivalRewardHandoffV1, Height: 2}
			change.fn(vm, &plan)
			require.Error(t, app.validateSurvivalHandoffSource(ctx, plan, vm))
		})
	}
}

func TestSurvivalHandoffMalformedPendingRollsBack(t *testing.T) {
	app, ctx := survivalHandoffSourceFixture(t)
	key, _, deadline := survivalHandoffPendingFixture(t, app, ctx)
	malformed := []byte(`{"fact_id":"handoff-fact","amount":"not-a-number"}`)
	ctx.KVStore(app.keys["knowledge"]).Set(key, malformed)
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameSurvivalRewardHandoffV1, Height: latest + 1}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	beforeEvents := len(ctx.EventManager().Events())
	require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	require.Equal(t, malformed, ctx.KVStore(app.keys["knowledge"]).Get(key))
	require.False(t, ctx.KVStore(app.keys["knowledge"]).Has(deadline))
	require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, "migration_v7_complete"))
	require.Equal(t, beforeEvents, len(ctx.EventManager().Events()))
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, survivalHandoffSourceVersionMap(), vm)
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	require.NoError(t, err)
	require.Zero(t, done)
}

func TestSurvivalHandoffRefusesLegacyOrAlreadyMarkedSource(t *testing.T) {
	for _, scenario := range []string{"legacy", "marker"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx := survivalHandoffSourceFixture(t)
			if scenario == "legacy" {
				ctx.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 0x01}, []byte(accountingAuthorityNativeMarker)...))
				ctx.KVStore(app.keys["zerone_staking"]).Delete([]byte{0x0a})
				ctx.KVStore(app.keys["zerone_gov"]).Delete([]byte{0x17})
			} else {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v7_complete", "true"))
			}
			latest := app.CommitMultiStore().Commit().Version
			plan := upgradetypes.Plan{Name: UpgradeNameSurvivalRewardHandoffV1, Height: latest + 1}
			require.Error(t, app.validateSurvivalHandoffSource(ctx, plan, survivalHandoffSourceVersionMap()))
		})
	}
}

func TestSurvivalHandoffTargetRequiresConsistentAppliedMarker(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		marker string
		done   int64
	}{
		{"marker only", "true", 0},
		{"done only", "", 1},
		{"malformed marker", "false", 1},
		{"future done", "true", 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			app, ctx, _ := newFrozenSurvivalFixture(t)
			if scenario.marker != "" {
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, "migration_v7_complete", scenario.marker))
			}
			if scenario.done > 0 {
				writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameSurvivalRewardHandoffV1, uint64(scenario.done))
			}
			app.CommitMultiStore().Commit()
			require.ErrorContains(t, app.ValidateAccountingAuthorityStartup(), "inconsistent migration marker")
		})
	}
}
