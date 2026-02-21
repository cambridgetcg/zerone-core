# R8-5 — Upgrade Module + Cosmovisor Integration

## Goal

Wire x/upgrade with migration stubs for every custom module. Set up cosmovisor config.
Create the first upgrade handler ("v1.0.0-testnet") that serves as the template for
future upgrades. Ensure `zeroned start` boots a single-validator chain.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/app/app.go` — upgrade handler registration pattern
- Cosmos SDK x/upgrade documentation
- Cosmovisor documentation

## Steps

### 1. Migration Stubs
For each of the 32 custom modules, create a migration stub in `x/<module>/migrations/`:
```go
// x/knowledge/migrations/v2/migrate.go
package v2

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/zerone-chain/zerone/x/knowledge/keeper"
)

// Migrate performs the v2 migration for the knowledge module.
func Migrate(ctx sdk.Context, k keeper.Keeper) error {
    // No-op for initial version. Future migrations go here.
    return nil
}
```

If some modules already have migration stubs from the batch sessions, verify they're correct.
Only create stubs for modules that don't have them.

### 2. Upgrade Handler Registration
In `app/app.go` (or a new `app/upgrades.go`), register the upgrade handler:
```go
func (app *ZeroneApp) RegisterUpgradeHandlers() {
    app.UpgradeKeeper.SetUpgradeHandler(
        "v1.0.0-testnet",
        func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
            // Run all module migrations
            return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
        },
    )
}
```

### 3. Store Upgrades
If any migration adds new store keys, register them:
```go
upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
if err == nil && upgradeInfo.Name == "v1.0.0-testnet" {
    storeUpgrades := storetypes.StoreUpgrades{
        Added: []string{/* new module store keys if any */},
    }
    app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
}
```

### 4. Cosmovisor Config
Create `cosmovisor/` directory structure:
```
cosmovisor/
├── README.md           (setup instructions)
├── genesis/
│   └── bin/
│       └── zeroned     (symlink or copy instructions)
└── upgrades/
    └── v1.0.0-testnet/
        └── bin/
            └── .gitkeep
```

Create a Makefile target:
```makefile
cosmovisor-init:
	@mkdir -p cosmovisor/genesis/bin
	@mkdir -p cosmovisor/upgrades/v1.0.0-testnet/bin
	@cp $(GOPATH)/bin/zeroned cosmovisor/genesis/bin/
```

### 5. Module Version Map
Ensure every module implements `ConsensusVersion()` returning 1:
```go
func (AppModule) ConsensusVersion() uint64 { return 1 }
```

Audit all 32 modules — some may already have this from their batch sessions.

### 6. Single-Validator Boot Test
Create a smoke test script `scripts/boot-test.sh`:
```bash
#!/bin/bash
set -e

CHAIN_ID="zerone-testnet-1"
MONIKER="test-validator"
HOME_DIR=$(mktemp -d)

# Init
zeroned init $MONIKER --chain-id $CHAIN_ID --home $HOME_DIR

# Add key
zeroned keys add validator --keyring-backend test --home $HOME_DIR

# Add genesis account
ADDR=$(zeroned keys show validator -a --keyring-backend test --home $HOME_DIR)
zeroned add-genesis-account $ADDR 100000000000uzrn --home $HOME_DIR

# Create gentx
zeroned gentx validator 10000000000uzrn --chain-id $CHAIN_ID --keyring-backend test --home $HOME_DIR

# Collect gentxs
zeroned collect-gentxs --home $HOME_DIR

# Validate genesis
zeroned validate-genesis --home $HOME_DIR

# Start (run for 10 blocks then stop)
timeout 30 zeroned start --home $HOME_DIR || true

echo "Boot test passed!"
rm -rf $HOME_DIR
```

### 7. .goreleaser Config
Create or update `.goreleaser.yaml` for reproducible builds:
```yaml
builds:
  - main: ./cmd/zeroned
    binary: zeroned
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/cosmos/cosmos-sdk/version.Name=zerone
      - -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned
      - -X github.com/cosmos/cosmos-sdk/version.Version={{.Version}}
      - -X github.com/cosmos/cosmos-sdk/version.Commit={{.Commit}}
```

## Tests

1. All 32 modules return `ConsensusVersion() == 1`
2. Upgrade handler registration doesn't panic
3. `go build ./cmd/zeroned/` succeeds
4. `scripts/boot-test.sh` runs without error (single validator boots, produces blocks)
5. `zeroned version` shows correct version info

## Constraints

- Every module MUST have a ConsensusVersion and migration stub
- Upgrade handler must be registered BEFORE the module manager (in app constructor)
- Store loader must be set BEFORE baseapp init
- Cosmovisor directory structure must follow the standard convention
- Boot test must be idempotent (uses temp directory, cleans up)
