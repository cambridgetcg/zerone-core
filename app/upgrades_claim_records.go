package app

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/zerone-chain/zerone/internal/claimrecordmigration"
)

const UpgradeNameKnowledgeClaimRecordsV1 = claimrecordmigration.UpgradeName

func claimRecordsSourceVersionMap() module.VersionMap {
	return reviewNeutralityTargetVersionMap()
}

func claimRecordsTargetVersionMap() module.VersionMap {
	vm := claimRecordsSourceVersionMap()
	vm["knowledge"] = 10
	return vm
}

func requireClaimRecordsTransitionOwner(name string, fromVM, targetVM module.VersionMap) error {
	from, fromOK := fromVM["knowledge"]
	target, targetOK := targetVM["knowledge"]
	if !fromOK || !targetOK || from < 6 || from > 11 || target < 6 || target > 11 {
		return fmt.Errorf("claim records requires complete known knowledge versions")
	}
	if from == 10 && target == 11 {
		return requireFundSettlementTransitionOwner(name, fromVM, targetVM)
	}
	if from == target || (from < 10 && target < 10) {
		return nil
	}
	if name != UpgradeNameKnowledgeClaimRecordsV1 || from != 9 || target != 10 {
		return fmt.Errorf("upgrade %q cannot carry knowledge 9->10; sole owner is %q", name, UpgradeNameKnowledgeClaimRecordsV1)
	}
	return nil
}

func (app *ZeroneApp) validateClaimRecordsSource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameKnowledgeClaimRecordsV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fmt.Errorf("claim records requires its named height plan with empty info and no deprecated fields")
	}
	if !reflect.DeepEqual(fromVM, claimRecordsSourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), claimRecordsTargetVersionMap()) {
		return fmt.Errorf("claim records requires exact completed review-neutrality source and compiled target version maps")
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, fromVM) {
		return fmt.Errorf("claim records stored source version map mismatch: %v", err)
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireSurvivalHandoffTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireClaimRecordsTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fmt.Errorf("claim records H3 predecessor: %w", err)
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return fmt.Errorf("claim records requires completed accounting lineage: %v", err)
	}
	if err := app.validateReviewNeutralityCompleted(ctx, latest); err != nil {
		return err
	}
	enabled, err := app.KnowledgeKeeper.ClaimRecordsEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("claim records predecessor already enabled or unreadable: %v", err)
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fmt.Errorf("claim records predecessor already applied or unreadable: height=%d error=%v", done, err)
	}
	_, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v10_complete")
	if err != nil || marked {
		return fmt.Errorf("claim records predecessor already marked or unreadable: %v", err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fmt.Errorf("claim records cannot be unsafe-skipped")
	}
	return nil
}

func (app *ZeroneApp) registerClaimRecordsUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameKnowledgeClaimRecordsV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("claim records requires exact committed H-1/H boundary")
		}
		if err := app.validateClaimRecordsSource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := claimrecordmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(toVM, claimRecordsTargetVersionMap()) {
			return nil, fmt.Errorf("claim records returned unexpected target version map")
		}
		write()
		return toVM, nil
	})
}

func (app *ZeroneApp) validateClaimRecordsStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, claimRecordsTargetVersionMap()) {
		return app.validateClaimRecordsCompleted(ctx, latest)
	}
	if !reflect.DeepEqual(vm, claimRecordsSourceVersionMap()) {
		return fmt.Errorf("claim records startup requires exact complete source or target version map")
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
		return fmt.Errorf("claim records source startup requires exact on-chain and local H-1 plan")
	}
	return app.validateClaimRecordsSource(ctx, plan, vm)
}

func validateClaimRecordsGenesisSelection(genesis GenesisState) error {
	var selection struct {
		Enabled *bool `json:"claim_records_enabled"`
	}
	if err := json.Unmarshal(genesis["knowledge"], &selection); err != nil {
		return fmt.Errorf("decode knowledge genesis claim-record selection: %w", err)
	}
	if selection.Enabled == nil || !*selection.Enabled {
		return fmt.Errorf("native knowledge genesis requires explicit claim_records_enabled=true")
	}
	return nil
}

// validateClaimRecordsCompleted also validates later exact predecessors.
func (app *ZeroneApp) validateClaimRecordsCompleted(ctx sdk.Context, latest int64) error {
	enabled, err := app.KnowledgeKeeper.ClaimRecordsEnabled(ctx)
	if err != nil || !enabled {
		return fmt.Errorf("claim records target requires explicit enabled marker: %v", err)
	}
	if err := app.validateReviewNeutralityCompleted(ctx, latest); err != nil {
		return err
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameKnowledgeClaimRecordsV1)
	if err != nil {
		return err
	}
	marker, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v10_complete")
	if err != nil {
		return err
	}
	if done == 0 && !marked {
		return nil // Native/imported genesis selects semantics, not applied history.
	}
	if done <= 0 || done > latest || !marked || marker != "true" {
		return fmt.Errorf("claim records target has inconsistent migration marker and done height")
	}
	return nil
}
