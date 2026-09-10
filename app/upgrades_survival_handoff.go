package app

import (
	"context"
	"fmt"
	"reflect"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/zerone-chain/zerone/internal/survivalmigration"
)

// UpgradeNameSurvivalRewardHandoffV1 owns knowledge 6->7 and vesting_rewards
// 2->3. Its predecessor is the completed accounting release, never live legacy
// zerone-1 or an unfinished H1/H2/H3 transition.
const UpgradeNameSurvivalRewardHandoffV1 = survivalmigration.UpgradeName

func survivalHandoffSourceVersionMap() module.VersionMap {
	return accountingAuthorityTargetVersionMap()
}

func survivalHandoffTargetVersionMap() module.VersionMap {
	vm := survivalHandoffSourceVersionMap()
	vm["knowledge"], vm["vesting_rewards"] = 7, 3
	return vm
}

// Guard every broad migration call as well as the new handler. Frozen earlier
// binaries may retain their original targets; this candidate cannot append its
// handoff semantics to a historical plan.
func requireSurvivalHandoffTransitionOwner(name string, fromVM, targetVM module.VersionMap) error {
	read := func(vm module.VersionMap) ([2]uint64, error) {
		knowledge, kOK := vm["knowledge"]
		vesting, vOK := vm["vesting_rewards"]
		pair := [2]uint64{knowledge, vesting}
		if !kOK || !vOK || (pair != [2]uint64{6, 2} && pair != [2]uint64{7, 3} && pair != [2]uint64{8, 3}) {
			return pair, fmt.Errorf("survival handoff requires complete known knowledge/vesting_rewards version pair, got %v", pair)
		}
		return pair, nil
	}
	from, err := read(fromVM)
	if err != nil {
		return err
	}
	target, err := read(targetVM)
	if err != nil {
		return err
	}
	if from == target {
		return nil
	}
	if from == [2]uint64{7, 3} && target == [2]uint64{8, 3} {
		return requireRecordIntegrityTransitionOwner(name, fromVM, targetVM)
	}
	if name != UpgradeNameSurvivalRewardHandoffV1 || from != [2]uint64{6, 2} || target != [2]uint64{7, 3} {
		return fmt.Errorf("upgrade %q cannot carry survival handoff knowledge 6->7 and vesting_rewards 2->3; sole owner is %q", name, UpgradeNameSurvivalRewardHandoffV1)
	}
	return nil
}

func (app *ZeroneApp) validateSurvivalHandoffSource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameSurvivalRewardHandoffV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fmt.Errorf("survival handoff requires its named height plan with empty info and no deprecated fields")
	}
	if !reflect.DeepEqual(fromVM, survivalHandoffSourceVersionMap()) {
		return fmt.Errorf("survival handoff requires exact completed accounting source version map")
	}
	if !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), survivalHandoffTargetVersionMap()) {
		return fmt.Errorf("survival handoff requires exact compiled target version map")
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireSurvivalHandoffTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, fromVM) {
		return fmt.Errorf("survival handoff stored source version map mismatch: %v", err)
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fmt.Errorf("survival handoff H3 predecessor: %w", err)
	}
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, app.CommitMultiStore().LastCommitID().Version)
	if err != nil {
		return fmt.Errorf("survival handoff accounting predecessor: %w", err)
	}
	if lineage.Origin == "legacy" {
		return fmt.Errorf("survival handoff requires completed accounting lineage")
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fmt.Errorf("survival handoff source already applied or unreadable: height=%d error=%v", done, err)
	}
	_, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v7_complete")
	if err != nil || marked {
		return fmt.Errorf("survival handoff source already marked or unreadable: present=%t error=%v", marked, err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fmt.Errorf("survival handoff cannot be unsafe-skipped")
	}
	return nil
}

func (app *ZeroneApp) registerSurvivalHandoffUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameSurvivalRewardHandoffV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("survival handoff requires exact committed H-1/H boundary")
		}
		if err := app.validateSurvivalHandoffSource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := survivalmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(toVM, survivalHandoffTargetVersionMap()) {
			return nil, fmt.Errorf("survival handoff returned unexpected target version map")
		}
		write()
		return toVM, nil
	})
}

// validateSurvivalHandoffStartupVersions preserves historical accounting tests
// and binaries at their frozen map. The current compiled target admits its own
// native/restarted state, or the exact predecessor only at scheduled H-1. The
// surrounding accounting startup check still validates all lineage markers.
func (app *ZeroneApp) validateSurvivalHandoffStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	compiled := app.ModuleManager.GetVersionMap()
	if reflect.DeepEqual(compiled, recordIntegrityTargetVersionMap()) {
		return app.validateRecordIntegrityStartupVersions(ctx, vm, latest)
	}
	if reflect.DeepEqual(compiled, accountingAuthorityTargetVersionMap()) {
		return requireAccountingExactVersionMap(vm, accountingAuthorityTargetVersionMap())
	}
	if !reflect.DeepEqual(compiled, survivalHandoffTargetVersionMap()) {
		return fmt.Errorf("unknown compiled survival handoff target version map")
	}
	if reflect.DeepEqual(vm, survivalHandoffTargetVersionMap()) {
		return app.validateSurvivalHandoffCompleted(ctx, latest)
	}
	if !reflect.DeepEqual(vm, survivalHandoffSourceVersionMap()) {
		return fmt.Errorf("survival handoff startup requires an exact complete source or target version map")
	}
	local, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}
	plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	if err != nil {
		return err
	}
	if latest == int64(^uint64(0)>>1) || plan.Height != latest+1 || local.Name != plan.Name || local.Height != plan.Height || local.Info != plan.Info {
		return fmt.Errorf("survival handoff source startup requires the exact on-chain and local H-1 plan")
	}
	return app.validateSurvivalHandoffSource(ctx, plan, vm)
}

// Native and explicitly imported genesis have no fabricated applied-upgrade
// record. Applied state must retain both the original marker and done height.
func (app *ZeroneApp) validateSurvivalHandoffCompleted(ctx sdk.Context, latest int64) error {
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameSurvivalRewardHandoffV1)
	if err != nil {
		return err
	}
	marker, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v7_complete")
	if err != nil {
		return err
	}
	if done == 0 && !marked {
		return nil
	}
	if done <= 0 || done > latest || !marked || marker != "true" {
		return fmt.Errorf("survival handoff target has inconsistent migration marker and done height")
	}
	return nil
}
