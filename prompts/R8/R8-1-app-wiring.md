# R8-1 — Complete App Wiring: Home, Partnerships, Toolbox

## Goal

Wire the 3 remaining unwired module keepers into app.go so ALL 32 custom modules are fully
instantiated, registered in the module manager, and included in Begin/EndBlocker ordering.
After this session, `go test ./app/...` must pass with every module booting.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Current State

These 3 modules have code in `x/` but are NOT wired into app.go:
- **x/home** — has `AppModuleBasic` registered and maccPerm entry, but NO keeper field, NO `NewKeeper`, NO `NewAppModule`
- **x/partnerships** — completely unwired (only comments)
- **x/toolbox** — completely unwired (only comments)

All other 29 custom modules ARE fully wired.

## Reference

- `/Users/yuai/Desktop/legible_money/app/app.go` — draft wiring for all modules
- Check how existing modules are wired (e.g., `DisputesKeeper`, `ResearchKeeper`) for the pattern

## Steps

### 1. Add Keeper Struct Fields
In `ZeroneApp` struct, add:
```go
HomeKeeper          zeronehomekeeper.Keeper
PartnershipsKeeper  zeronepartnershipskeeper.Keeper
ToolboxKeeper       zeronetoolboxkeeper.Keeper
```

### 2. Instantiate Keepers
Wire `NewKeeper` for each, following the dependency order:
- **Home** depends on: auth, staking, bank
- **Partnerships** depends on: home, staking, bank (circular with home — break with interface setter like draft)
- **Toolbox** depends on: billing, knowledge, tree, bvm, vesting_rewards, discovery, home

Check each module's `NewKeeper` signature for required keeper interfaces.
Wire cross-module interfaces with setter methods (e.g., `SetHomeKeeper`, `SetPartnershipsKeeper`)
the same way autopoiesis/alignment use setters.

### 3. Register in Module Manager
Add `NewAppModule` for all 3 to the module manager list.

### 4. BeginBlocker / EndBlocker
Add all 3 to the ordering lists. Check the draft for correct positions:
- Home: after auth, before partnerships
- Partnerships: after home, after staking
- Toolbox: after billing, after tree

### 5. Genesis Order
Add all 3 to the genesis init/export ordering.

### 6. Module Account Permissions
Home already has a maccPerm entry. Add partnerships and toolbox if they need module accounts.
Check draft `maccPerms` for these modules.

## Verification

```bash
# Must pass — all modules boot without panic
go test ./app/...

# Must pass — cross-stack tests use full app harness
go test ./tests/cross_stack/...

# Full suite — all 35 test packages green
go test ./...
```

## Constraints

- Do NOT change any module logic — this is wiring only
- Follow the exact same wiring pattern as existing modules
- Resolve any circular dependencies with interface setters (not direct imports)
- Ensure all `RegisterInterfaces` and `RegisterLegacyAminoCodec` are called
- All keeper fields must be concrete types (not interfaces) in the app struct
