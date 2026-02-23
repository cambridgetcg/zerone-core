package app

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeNameTestnet = "v1.0.0-testnet"

// RegisterUpgradeHandlers registers upgrade handlers for each named software upgrade.
// When a governance upgrade proposal passes, the corresponding handler here runs
// the necessary state migrations before the new binary starts producing blocks.
//
// Call this AFTER RegisterServices but BEFORE LoadLatestVersion.
func (app *ZeroneApp) RegisterUpgradeHandlers() {
	// v1.0.0-testnet — initial testnet launch.
	// Runs all module migrations from ConsensusVersion 1 → 2.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameTestnet,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))
			return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
		},
	)
}

// RegisterStoreUpgrades configures store loaders for upgrades that add or remove
// module store keys. Call this BEFORE LoadLatestVersion.
func (app *ZeroneApp) RegisterStoreUpgrades() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		// No pending upgrade — nothing to do.
		return
	}

	switch upgradeInfo.Name {
	case UpgradeNameTestnet:
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				// Add new module store keys here when the upgrade introduces them.
			},
		}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
