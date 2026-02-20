# R1-5 — Genesis Framework + Integration Test

## Goal

Verify that auth + staking modules work together. `zeroned init` produces
a valid genesis. InitGenesis/ExportGenesis round-trips. A basic transaction
flow works in tests.

## Dependencies

- R1-3 (auth) and R1-4 (staking) must both be complete

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/tests/cross_stack/genesis_validation_test.go` — draft genesis tests
- `/Users/yuai/Desktop/legible_money/scripts/genesis-full.sh` — draft genesis script
- `/Users/yuai/Desktop/legible_money/reports/batch-21/B21-1-genesis-rerun.md` — genesis dry-run report

## Deliverables

### 1. Genesis validation tests

`tests/cross_stack/genesis_test.go`:

```go
func TestDefaultGenesis_AllModules(t *testing.T) {
    // For every registered module:
    // 1. DefaultGenesis()
    // 2. Validate()
    // 3. Assert no error
}

func TestGenesisRoundTrip(t *testing.T) {
    // 1. Create app with DefaultGenesis
    // 2. InitChain
    // 3. Run 5 blocks
    // 4. ExportGenesis
    // 5. Validate exported genesis
    // 6. Create new app, InitChain with exported genesis
    // 7. Run 5 more blocks
    // 8. No panics
}

func TestGenesisJSON_Deterministic(t *testing.T) {
    // Export genesis twice from same state
    // Verify JSON is byte-identical (proto marshal determinism)
}
```

### 2. Auth + Staking integration test

`tests/cross_stack/auth_staking_test.go`:

```go
func TestAuthStaking_RegisterAndStake(t *testing.T) {
    // 1. Register account via x/auth
    // 2. Verify account exists with DID
    // 3. Fund account via bank
    // 4. Register as validator via x/staking
    // 5. Verify validator at Apprentice tier
    // 6. Delegate more stake
    // 7. Verify total delegation increased
    // 8. Check tier advancement prerequisites
}

func TestAuthStaking_FrozenAccountCannotStake(t *testing.T) {
    // 1. Register and fund account
    // 2. Freeze account via x/auth
    // 3. Attempt to register as validator → should fail
    // 4. Unfreeze
    // 5. Register as validator → should succeed
}

func TestAuthStaking_SessionKeyCanDelegate(t *testing.T) {
    // 1. Register account
    // 2. Create session key with delegation permission
    // 3. Use session key to delegate
    // 4. Verify delegation succeeded
    // 5. Revoke session key
    // 6. Attempt delegation with revoked key → fail
}
```

### 3. Multi-keeper test harness

`tests/cross_stack/harness_test.go`:

Create a test harness that wires real keepers (not mocks) for integration tests:

```go
type TestHarness struct {
    App        *app.ZeroneApp
    Ctx        sdk.Context
    AuthKeeper authkeeper.Keeper
    StakingKeeper stakingkeeper.Keeper
    BankKeeper bankkeeper.Keeper
    // ... add more as modules are ported
}

func NewTestHarness(t *testing.T) *TestHarness {
    // Create in-memory app with default genesis
    // Return harness with all keepers accessible
}

func (h *TestHarness) FundAccount(addr sdk.AccAddress, amount sdk.Coins) error
func (h *TestHarness) AdvanceBlocks(n int)
func (h *TestHarness) GetBalance(addr sdk.AccAddress) sdk.Coins
```

This harness grows as we port more modules — each batch adds its keeper.

### 4. Init script

`scripts/init-testnet.sh`:
```bash
#!/bin/bash
# Quick single-validator testnet initialization
BINARY=${1:-"./build/zeroned"}
CHAIN_ID=${2:-"zerone-testnet-1"}
HOME_DIR=$(mktemp -d)

$BINARY init validator-1 --chain-id $CHAIN_ID --home $HOME_DIR
$BINARY keys add validator --keyring-backend test --home $HOME_DIR
ADDR=$($BINARY keys show validator -a --keyring-backend test --home $HOME_DIR)

# Fund validator
$BINARY genesis add-genesis-account $ADDR 1000000000000uzrn --home $HOME_DIR

# Create gentx
$BINARY genesis gentx validator 100000000000uzrn \
    --chain-id $CHAIN_ID \
    --keyring-backend test \
    --home $HOME_DIR

$BINARY genesis collect-gentxs --home $HOME_DIR

echo "Genesis initialized at $HOME_DIR"
echo "Start with: $BINARY start --home $HOME_DIR"
```

### 5. Smoke test

`scripts/smoke-test.sh`:
```bash
#!/bin/bash
# Minimal smoke test: init, start, wait for blocks, stop
set -euo pipefail

BINARY="./build/zeroned"
HOME_DIR=$(mktemp -d)
CHAIN_ID="zerone-smoke"

cleanup() { kill $PID 2>/dev/null || true; rm -rf "$HOME_DIR"; }
trap cleanup EXIT

# Init
$BINARY init smoke-node --chain-id $CHAIN_ID --home $HOME_DIR 2>/dev/null
$BINARY keys add val --keyring-backend test --home $HOME_DIR 2>/dev/null
ADDR=$($BINARY keys show val -a --keyring-backend test --home $HOME_DIR)
$BINARY genesis add-genesis-account $ADDR 1000000000000uzrn --home $HOME_DIR 2>/dev/null
$BINARY genesis gentx val 100000000000uzrn --chain-id $CHAIN_ID --keyring-backend test --home $HOME_DIR 2>/dev/null
$BINARY genesis collect-gentxs --home $HOME_DIR 2>/dev/null

# Start
$BINARY start --home $HOME_DIR --minimum-gas-prices 0uzrn --log_level error &
PID=$!

# Wait for blocks
echo "Waiting for block production..."
for i in $(seq 1 30); do
    HEIGHT=$($BINARY status --home $HOME_DIR 2>/dev/null | jq -r '.sync_info.latest_block_height // "0"' 2>/dev/null || echo "0")
    if [ "$HEIGHT" -ge 5 ] 2>/dev/null; then
        echo "✅ Block $HEIGHT reached. Smoke test passed!"
        exit 0
    fi
    sleep 1
done

echo "❌ Failed to reach block 5 within 30 seconds"
exit 1
```

## Verification

```bash
go build ./...
go test ./... -count=1 -v
make build
bash scripts/smoke-test.sh
```

## Commit

```
feat: genesis framework, auth+staking integration tests, smoke test

- Genesis DefaultGenesis + round-trip tests for all modules
- Auth + staking cross-module integration (register, stake, freeze)
- Multi-keeper test harness for future integration tests
- Smoke test script: init, start, reach block 5
```

## Do NOT

- Use mocks for integration tests — use real keepers
- Skip the genesis round-trip test (this catches serialization bugs)
- Skip the smoke test (must prove the binary actually runs)
- Add modules beyond auth + staking (they come in later batches)
