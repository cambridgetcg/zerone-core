package app

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/zerone-chain/zerone/internal/recordmigration"
)

const UpgradeNameKnowledgeRecordIntegrityV1 = recordmigration.UpgradeName

func recordIntegritySourceVersionMap() module.VersionMap {
	return survivalHandoffTargetVersionMap()
}

func recordIntegrityTargetVersionMap() module.VersionMap {
	vm := recordIntegritySourceVersionMap()
	vm["knowledge"] = 8
	return vm
}

// Earlier named releases retain their original target. No broad migration may
// append knowledge 7->8, skip its predecessor, or downgrade current records.
func requireRecordIntegrityTransitionOwner(name string, fromVM, targetVM module.VersionMap) error {
	from, fromOK := fromVM["knowledge"]
	target, targetOK := targetVM["knowledge"]
	if !fromOK || !targetOK || from < 6 || from > 8 || target < 6 || target > 8 {
		return fmt.Errorf("record integrity requires complete known knowledge versions")
	}
	if from == target || (from < 8 && target < 8) {
		return nil
	}
	if name != UpgradeNameKnowledgeRecordIntegrityV1 || from != 7 || target != 8 {
		return fmt.Errorf("upgrade %q cannot carry knowledge 7->8; sole owner is %q", name, UpgradeNameKnowledgeRecordIntegrityV1)
	}
	return nil
}

func (app *ZeroneApp) validateRecordIntegritySource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameKnowledgeRecordIntegrityV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fmt.Errorf("record integrity requires its named height plan with empty info and no deprecated fields")
	}
	if !reflect.DeepEqual(fromVM, recordIntegritySourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), recordIntegrityTargetVersionMap()) {
		return fmt.Errorf("record integrity requires exact completed survival source and compiled target version maps")
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, fromVM) {
		return fmt.Errorf("record integrity stored source version map mismatch: %v", err)
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireSurvivalHandoffTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireRecordIntegrityTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fmt.Errorf("record integrity H3 predecessor: %w", err)
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return fmt.Errorf("record integrity requires completed accounting lineage: %v", err)
	}
	if err := app.validateSurvivalHandoffCompleted(ctx, latest); err != nil {
		return err
	}
	enabled, err := app.KnowledgeKeeper.RecordIntegrityEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("record integrity predecessor already enabled or unreadable: %v", err)
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fmt.Errorf("record integrity predecessor already applied or unreadable: height=%d error=%v", done, err)
	}
	_, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v8_complete")
	if err != nil || marked {
		return fmt.Errorf("record integrity predecessor already marked or unreadable: %v", err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fmt.Errorf("record integrity cannot be unsafe-skipped")
	}
	return nil
}

func (app *ZeroneApp) registerRecordIntegrityUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameKnowledgeRecordIntegrityV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("record integrity requires exact committed H-1/H boundary")
		}
		if err := app.validateRecordIntegritySource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := recordmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(toVM, recordIntegrityTargetVersionMap()) {
			return nil, fmt.Errorf("record integrity returned unexpected target version map")
		}
		write()
		return toVM, nil
	})
}

func (app *ZeroneApp) validateRecordIntegrityStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, recordIntegrityTargetVersionMap()) {
		enabled, err := app.KnowledgeKeeper.RecordIntegrityEnabled(ctx)
		if err != nil || !enabled {
			return fmt.Errorf("record integrity target requires explicit enabled marker: %v", err)
		}
		if err := app.validateSurvivalHandoffCompleted(ctx, latest); err != nil {
			return err
		}
		done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameKnowledgeRecordIntegrityV1)
		if err != nil {
			return err
		}
		marker, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v8_complete")
		if err != nil {
			return err
		}
		if done == 0 && !marked {
			return nil // Native/imported genesis preserves semantics, not applied history.
		}
		if done <= 0 || done > latest || !marked || marker != "true" {
			return fmt.Errorf("record integrity target has inconsistent migration marker and done height")
		}
		return nil
	}
	if !reflect.DeepEqual(vm, recordIntegritySourceVersionMap()) {
		return fmt.Errorf("record integrity startup requires exact complete source or target version map")
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
		return fmt.Errorf("record integrity source startup requires exact on-chain and local H-1 plan")
	}
	return app.validateRecordIntegritySource(ctx, plan, vm)
}

func validateRecordIntegrityGenesisSelection(genesis GenesisState) error {
	var selection struct {
		Enabled *bool `json:"record_integrity_enabled"`
	}
	if err := json.Unmarshal(genesis["knowledge"], &selection); err != nil {
		return fmt.Errorf("decode knowledge genesis integrity selection: %w", err)
	}
	if selection.Enabled == nil || !*selection.Enabled {
		return fmt.Errorf("native knowledge genesis requires explicit record_integrity_enabled=true")
	}
	return nil
}
