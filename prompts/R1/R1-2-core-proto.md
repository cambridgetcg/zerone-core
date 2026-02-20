# R1-2 — Core Proto: Shared Types + x/upgrade

## Goal

Define the shared protobuf types used across all Zerone modules, generate
the Go code, and wire x/upgrade into the app skeleton from R1-1.

## Dependencies

- R1-1 must be complete (repo scaffold, buf config)

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

Draft proto at `/Users/yuai/Desktop/legible_money/proto/legible/`:
- Common types, module options, shared enums

## Deliverables

### 1. Shared proto types

`proto/zerone/common/v1/common.proto`:
```protobuf
syntax = "proto3";
package zerone.common.v1;

option go_package = "github.com/zerone-chain/zerone/x/common/types";

// BasisPoints represents a value on a 1,000,000 scale (100% = 1,000,000).
// ALL modules use this scale. No exceptions.
message BasisPoints {
  uint64 value = 1;
}

// Revenue split configuration — governance-adjustable.
message RevenueSplit {
  uint64 contributor_bps = 1;  // default 550,000 (55%)
  uint64 protocol_bps = 2;    // default 220,000 (22%)
  uint64 research_bps = 3;    // default 130,000 (13%)
  uint64 burn_bps = 4;        // default 100,000 (10%)
  // Must sum to 1,000,000
}

// ProtocolSubSplit — how the protocol share is divided.
message ProtocolSubSplit {
  uint64 citation_bps = 1;      // default 500,000 (50% of protocol)
  uint64 verification_bps = 2;  // default 300,000 (30% of protocol)
  uint64 treasury_bps = 3;      // default 200,000 (20% of protocol)
  // Must sum to 1,000,000
}
```

### 2. Module options proto

`proto/zerone/module/v1/module.proto`:
```protobuf
syntax = "proto3";
package zerone.module.v1;

option go_package = "github.com/zerone-chain/zerone/x/module/types";

import "cosmos/app/v1alpha1/module.proto";

// Each Zerone module registers itself with:
message Module {
  option (cosmos.app.v1alpha1.module) = {
    go_import: "github.com/zerone-chain/zerone/x/<module>"
  };
  string authority = 1;  // governance authority address
}
```

### 3. x/upgrade integration

Wire `x/upgrade` into `app/app.go`:
- Add upgrade keeper and store key
- Register in ModuleManager
- Add to BeginBlocker (first position)
- Create `app/upgrades.go` with empty upgrade handler registration:

```go
func (app *ZeroneApp) RegisterUpgradeHandlers() {
    // Each version upgrade gets a handler here.
    // Example (not active yet):
    // app.UpgradeKeeper.SetUpgradeHandler("v1.1.0", func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
    //     return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
    // })
}
```

### 4. Cosmovisor compatibility

Create `app/export.go`:
```go
// ExportAppStateAndValidators exports the state of the application for a genesis file.
func (app *ZeroneApp) ExportAppStateAndValidators(...) (...)
```

### 5. Token denomination registration

In `app/app.go` or genesis setup, register the ZRN denomination metadata:

```go
bankGenesis.DenomMetadata = []banktypes.Metadata{
    {
        Description: "The native staking and governance token of Zerone",
        DenomUnits: []*banktypes.DenomUnit{
            {Denom: "uzrn", Exponent: 0, Aliases: []string{"microzrn"}},
            {Denom: "mzrn", Exponent: 3, Aliases: []string{"millizrn"}},
            {Denom: "zrn", Exponent: 6, Aliases: nil},
        },
        Base:    "uzrn",
        Display: "zrn",
        Name:    "Zerone",
        Symbol:  "ZRN",
    },
}
```

### 6. Generate proto code

Run `cd proto && buf generate` (or `make proto-gen`).
Verify generated `.pb.go` files appear in the correct Go packages.

## Tests

Create `app/app_test.go`:

```go
func TestNewZeroneApp(t *testing.T) {
    // Verify app can be constructed without panic
}

func TestExportGenesis(t *testing.T) {
    // Init + Export round-trip
}
```

## Verification

```bash
make proto-gen              # generates Go code from proto
go build ./...              # compiles with generated code
go vet ./...
go test ./... -count=1
make build
build/zeroned init test --chain-id zerone-test
build/zeroned version
```

## Commit

```
feat: add shared proto types, x/upgrade integration, ZRN denomination
```

## Do NOT

- Define module-specific protos (auth, staking — those are R1-3 and R1-4)
- Add custom BeginBlocker/EndBlocker logic yet
- Skip the BasisPoints standardization — this is the foundation for consistency
