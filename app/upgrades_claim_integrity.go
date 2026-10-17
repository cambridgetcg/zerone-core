package app

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/zerone-chain/zerone/internal/claimintegritymigration"
)

// UpgradeNameKnowledgeClaimIntegrityV1 owns knowledge 11->12. Its predecessor
// is completed fund settlement; it writes marker 0x87 and its receipt, and
// restores the fact edges that equal-ID augmentations replaced before
// activation (keeper.RestoreAugmentationReplacedEdges), nothing else.
const UpgradeNameKnowledgeClaimIntegrityV1 = claimintegritymigration.UpgradeName

func claimIntegritySourceVersionMap() module.VersionMap {
	return fundSettlementTargetVersionMap()
}

func claimIntegrityTargetVersionMap() module.VersionMap {
	vm := claimIntegritySourceVersionMap()
	vm["knowledge"] = 12
	return vm
}

// Earlier named releases retain their original target. No broad migration may
// append knowledge 11->12, skip its predecessor, or downgrade current records.
func requireClaimIntegrityTransitionOwner(name string, fromVM, targetVM module.VersionMap) error {
	from, fromOK := fromVM["knowledge"]
	target, targetOK := targetVM["knowledge"]
	if !fromOK || !targetOK || from < 6 || from > 12 || target < 6 || target > 12 {
		return fmt.Errorf("claim integrity requires complete known knowledge versions")
	}
	if from == target || (from < 12 && target < 12) {
		return nil
	}
	if name != UpgradeNameKnowledgeClaimIntegrityV1 || from != 11 || target != 12 {
		return fmt.Errorf("upgrade %q cannot carry knowledge 11->12; sole owner is %q", name, UpgradeNameKnowledgeClaimIntegrityV1)
	}
	return nil
}

func (app *ZeroneApp) validateClaimIntegritySource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameKnowledgeClaimIntegrityV1 || plan.Height <= 0 || plan.Info != "" || !plan.Time.IsZero() || plan.UpgradedClientState != nil {
		return fmt.Errorf("claim integrity requires its named height plan with empty info and no deprecated fields")
	}
	if !reflect.DeepEqual(fromVM, claimIntegritySourceVersionMap()) || !reflect.DeepEqual(app.ModuleManager.GetVersionMap(), claimIntegrityTargetVersionMap()) {
		return fmt.Errorf("claim integrity requires exact completed fund-settlement source and compiled target version maps")
	}
	stored, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil || !reflect.DeepEqual(stored, fromVM) {
		return fmt.Errorf("claim integrity stored source version map mismatch: %v", err)
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireSurvivalHandoffTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := requireClaimIntegrityTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return fmt.Errorf("claim integrity H3 predecessor: %w", err)
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	lineage, err := app.accountingAuthorityGenesisMetadata(ctx, latest)
	if err != nil || lineage.Origin == "legacy" {
		return fmt.Errorf("claim integrity requires completed accounting lineage: %v", err)
	}
	if err := app.validateFundSettlementCompleted(ctx, latest); err != nil {
		return err
	}
	enabled, err := app.KnowledgeKeeper.ClaimIntegrityEnabled(ctx)
	if err != nil || enabled {
		return fmt.Errorf("claim integrity predecessor already enabled or unreadable: %v", err)
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	if err != nil || done != 0 {
		return fmt.Errorf("claim integrity predecessor already applied or unreadable: height=%d error=%v", done, err)
	}
	_, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v12_complete")
	if err != nil || marked {
		return fmt.Errorf("claim integrity predecessor already marked or unreadable: %v", err)
	}
	if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
		return fmt.Errorf("claim integrity cannot be unsafe-skipped")
	}
	return app.KnowledgeKeeper.ValidateClaimIntegrityActivation(ctx)
}

func (app *ZeroneApp) registerClaimIntegrityUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameKnowledgeClaimIntegrityV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("claim integrity requires exact committed H-1/H boundary")
		}
		if err := app.validateClaimIntegritySource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := claimintegritymigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(toVM, claimIntegrityTargetVersionMap()) {
			return nil, fmt.Errorf("claim integrity returned unexpected target version map")
		}
		write()
		return toVM, nil
	})
}

func (app *ZeroneApp) validateClaimIntegrityStartupVersions(ctx sdk.Context, vm module.VersionMap, latest int64) error {
	if reflect.DeepEqual(vm, claimIntegrityTargetVersionMap()) {
		return app.validateClaimIntegrityCompleted(ctx, latest)
	}
	if !reflect.DeepEqual(vm, claimIntegritySourceVersionMap()) {
		return fmt.Errorf("claim integrity startup requires exact complete source or target version map")
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
		return fmt.Errorf("claim integrity source startup requires exact on-chain and local H-1 plan")
	}
	return app.validateClaimIntegritySource(ctx, plan, vm)
}

// validateClaimIntegrityGenesisSelection runs before InitGenesis changes any
// store. A v12 binary never infers the selection from fund settlement, so a
// pre-v12 export cannot silently switch semantics on import.
func validateClaimIntegrityGenesisSelection(genesis GenesisState) error {
	var selection struct {
		Enabled *bool `json:"claim_integrity_enabled"`
	}
	if err := json.Unmarshal(genesis["knowledge"], &selection); err != nil {
		return fmt.Errorf("decode knowledge genesis claim-integrity selection: %w", err)
	}
	if selection.Enabled == nil || !*selection.Enabled {
		return fmt.Errorf("native knowledge genesis requires explicit claim_integrity_enabled=true")
	}
	return nil
}

// validateClaimIntegrityCompleted also validates every earlier exact
// predecessor. Native or imported genesis has no receipt and no done height;
// applied state must retain both.
func (app *ZeroneApp) validateClaimIntegrityCompleted(ctx sdk.Context, latest int64) error {
	enabled, err := app.KnowledgeKeeper.ClaimIntegrityEnabled(ctx)
	if err != nil || !enabled {
		return fmt.Errorf("claim integrity target requires explicit enabled marker: %v", err)
	}
	if err := app.validateFundSettlementCompleted(ctx, latest); err != nil {
		return err
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameKnowledgeClaimIntegrityV1)
	if err != nil {
		return err
	}
	marker, marked, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, "migration_v12_complete")
	if err != nil {
		return err
	}
	if done == 0 && !marked {
		return nil // Native/imported genesis selects semantics, not applied history.
	}
	if done <= 0 || done > latest || !marked || marker != "true" {
		return fmt.Errorf("claim integrity target has inconsistent migration marker and done height")
	}
	return nil
}
