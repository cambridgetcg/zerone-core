package app

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/zerone-chain/zerone/internal/fundsettlementmigration"
)

const UpgradeNameKnowledgeFundSettlementV1 = fundsettlementmigration.UpgradeName

func fundSettlementSourceVersionMap() module.VersionMap {
	return claimRecordsTargetVersionMap()
}

func fundSettlementTargetVersionMap() module.VersionMap {
	vm := fundSettlementSourceVersionMap()
	vm["knowledge"] = 11
	return vm
}

func requireFundSettlementTransitionOwner(name string, fromVM, targetVM module.VersionMap) error {
	from, fromOK := fromVM["knowledge"]
	target, targetOK := targetVM["knowledge"]
	if !fromOK || !targetOK || from < 6 || from > 11 || target < 6 || target > 11 {
		return fmt.Errorf("fund settlement requires complete known knowledge versions")
	}
	if from == target || (from < 11 && target < 11) {
		return nil
	}
	if name != UpgradeNameKnowledgeFundSettlementV1 || from != 10 || target != 11 {
		return fmt.Errorf("upgrade %q cannot carry knowledge 10->11; sole owner is %q", name, UpgradeNameKnowledgeFundSettlementV1)
	}
	return nil
}

func (app *ZeroneApp) validateFundSettlementSource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameKnowledgeFundSettlementV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fmt.Errorf("fund settlement requires its named height plan with empty info and no deprecated fields")
	}
	if !reflect.DeepEqual(fromVM, fundSettlementSourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), fundSettlementTargetVersionMap()) {
		return fmt.Errorf("fund settlement requires exact completed claim-records source and compiled target version maps")
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, fromVM) {
		return fmt.Errorf("fund settlement stored source version map mismatch: %v", err)
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireSurvivalHandoffTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireFundSettlementTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fmt.Errorf("fund settlement H3 predecessor: %w", err)
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return fmt.Errorf("fund settlement requires completed accounting lineage: %v", err)
	}
	if err := app.validateClaimRecordsCompleted(ctx, latest); err != nil {
		return err
	}
	enabled, err := app.KnowledgeKeeper.FundSettlementEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("fund settlement predecessor already enabled or unreadable: %v", err)
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fmt.Errorf("fund settlement predecessor already applied or unreadable: height=%d error=%v", done, err)
	}
	_, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v11_complete")
	if err != nil || marked {
		return fmt.Errorf("fund settlement predecessor already marked or unreadable: %v", err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fmt.Errorf("fund settlement cannot be unsafe-skipped")
	}
	return app.KnowledgeKeeper.ValidateFundSettlementActivation(ctx)
}

func (app *ZeroneApp) registerFundSettlementUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameKnowledgeFundSettlementV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("fund settlement requires exact committed H-1/H boundary")
		}
		if err := app.validateFundSettlementSource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := fundsettlementmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(toVM, fundSettlementTargetVersionMap()) {
			return nil, fmt.Errorf("fund settlement returned unexpected target version map")
		}
		write()
		return toVM, nil
	})
}

func (app *ZeroneApp) validateFundSettlementStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, fundSettlementTargetVersionMap()) {
		enabled, err := app.KnowledgeKeeper.FundSettlementEnabled(ctx)
		if err != nil || !enabled {
			return fmt.Errorf("fund settlement target requires explicit enabled marker: %v", err)
		}
		if err := app.validateClaimRecordsCompleted(ctx, latest); err != nil {
			return err
		}
		done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameKnowledgeFundSettlementV1)
		if err != nil {
			return err
		}
		marker, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v11_complete")
		if err != nil {
			return err
		}
		if done == 0 && !marked {
			return nil // Native/imported genesis selects semantics, not applied history.
		}
		if done <= 0 || done > latest || !marked || marker != "true" {
			return fmt.Errorf("fund settlement target has inconsistent migration marker and done height")
		}
		return nil
	}
	if !reflect.DeepEqual(vm, fundSettlementSourceVersionMap()) {
		return fmt.Errorf("fund settlement startup requires exact complete source or target version map")
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
		return fmt.Errorf("fund settlement source startup requires exact on-chain and local H-1 plan")
	}
	return app.validateFundSettlementSource(ctx, plan, vm)
}

func validateFundSettlementGenesisSelection(genesis GenesisState) error {
	var selection struct {
		Enabled *bool `json:"fund_settlement_enabled"`
	}
	if err := json.Unmarshal(genesis["knowledge"], &selection); err != nil {
		return fmt.Errorf("decode knowledge genesis fund-settlement selection: %w", err)
	}
	if selection.Enabled == nil || !*selection.Enabled {
		return fmt.Errorf("native knowledge genesis requires explicit fund_settlement_enabled=true")
	}
	return nil
}
