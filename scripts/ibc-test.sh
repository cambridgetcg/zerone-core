#!/bin/bash
#
# Zerone — IBC End-to-End Test
#
# Self-contained script that:
#   1. Builds the binary
#   2. Starts two independent Zerone chains (A and B)
#   3. Sets up a Go relayer between them
#   4. Runs 7 IBC test scenarios including rate limiting
#   5. Reports pass/fail results
#
# Prerequisites:
#   - Go 1.24+ (for building zeroned and relayer)
#   - jq (for JSON parsing)
#   - curl (for REST queries)
#
# Usage:
#   bash scripts/ibc-test.sh
#
# Environment:
#   BINARY     — path to zeroned binary (default: ./zeroned)
#   SKIP_BUILD — set to "1" to skip building the binary
#   RLY_BIN    — path to rly binary (default: auto-detect or install)
#
# Exit codes:
#   0 = all tests passed (or skipped)
#   1 = one or more tests failed

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
BINARY="${BINARY:-./zeroned}"
DENOM="uzrn"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Chain A
CHAIN_A_ID="zerone-ibc-a"
CHAIN_A_HOME=$(mktemp -d)
CHAIN_A_MONIKER="validator-a"
CHAIN_A_P2P=26656
CHAIN_A_RPC=26657
CHAIN_A_REST=1317
CHAIN_A_GRPC=9090
CHAIN_A_PPROF=6060
CHAIN_A_RPC_URL="http://localhost:${CHAIN_A_RPC}"
CHAIN_A_REST_URL="http://localhost:${CHAIN_A_REST}"
CHAIN_A_PID=""

# Chain B (offset ports by +100)
CHAIN_B_ID="zerone-ibc-b"
CHAIN_B_HOME=$(mktemp -d)
CHAIN_B_MONIKER="validator-b"
CHAIN_B_P2P=26756
CHAIN_B_RPC=26757
CHAIN_B_REST=1417
CHAIN_B_GRPC=9190
CHAIN_B_PPROF=6160
CHAIN_B_RPC_URL="http://localhost:${CHAIN_B_RPC}"
CHAIN_B_REST_URL="http://localhost:${CHAIN_B_REST}"
CHAIN_B_PID=""

# Relayer
RLY_BIN="${RLY_BIN:-}"
RLY_HOME=$(mktemp -d)
RLY_PID=""
IBC_PATH="ibc-test-path"

# Test counters
PASS=0
FAIL=0
SKIP=0

# ---------------------------------------------------------------------------
# Cleanup (trap EXIT)
# ---------------------------------------------------------------------------
cleanup() {
    echo ""
    echo "Cleaning up..."
    if [ -n "$RLY_PID" ]; then
        kill "$RLY_PID" 2>/dev/null || true
        wait "$RLY_PID" 2>/dev/null || true
    fi
    if [ -n "$CHAIN_A_PID" ]; then
        kill "$CHAIN_A_PID" 2>/dev/null || true
        wait "$CHAIN_A_PID" 2>/dev/null || true
    fi
    if [ -n "$CHAIN_B_PID" ]; then
        kill "$CHAIN_B_PID" 2>/dev/null || true
        wait "$CHAIN_B_PID" 2>/dev/null || true
    fi
    rm -rf "$CHAIN_A_HOME" "$CHAIN_B_HOME" "$RLY_HOME"
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
zeroned_a() {
    "$BINARY" "$@" --home "$CHAIN_A_HOME"
}

zeroned_b() {
    "$BINARY" "$@" --home "$CHAIN_B_HOME"
}

get_height() {
    local rpc_url="$1"
    curl -s "${rpc_url}/status" 2>/dev/null | \
        jq -r '.result.sync_info.latest_block_height // "0"' 2>/dev/null || echo "0"
}

get_balance() {
    local rest_url="$1"
    local addr="$2"
    local denom="$3"
    curl -s "${rest_url}/cosmos/bank/v1beta1/balances/${addr}" 2>/dev/null | \
        jq -r ".balances[] | select(.denom == \"${denom}\") | .amount // \"0\"" 2>/dev/null || echo "0"
}

get_ibc_denom() {
    local rest_url="$1"
    local addr="$2"
    curl -s "${rest_url}/cosmos/bank/v1beta1/balances/${addr}" 2>/dev/null | \
        jq -r '.balances[] | select(.denom | startswith("ibc/")) | .denom' 2>/dev/null | head -1 || echo ""
}

get_ibc_balance() {
    local rest_url="$1"
    local addr="$2"
    curl -s "${rest_url}/cosmos/bank/v1beta1/balances/${addr}" 2>/dev/null | \
        jq -r '.balances[] | select(.denom | startswith("ibc/")) | .amount' 2>/dev/null | head -1 || echo "0"
}

wait_for_height() {
    local rpc_url="$1"
    local target="$2"
    local timeout="${3:-120}"
    for i in $(seq 1 "$timeout"); do
        local h
        h=$(get_height "$rpc_url")
        if [ "$h" -ge "$target" ] 2>/dev/null; then
            return 0
        fi
        sleep 1
    done
    return 1
}

settle() {
    sleep 3
}

run_test_eval() {
    local name="$1"
    local cmd="$2"
    if eval "$cmd" >/dev/null 2>&1; then
        echo "  PASS  $name"
        PASS=$((PASS + 1))
    else
        echo "  FAIL  $name"
        FAIL=$((FAIL + 1))
    fi
}

skip_test() {
    local name="$1"
    local reason="$2"
    echo "  SKIP  $name ($reason)"
    SKIP=$((SKIP + 1))
}

echo "=== IBC End-to-End Test (Zerone) ==="
echo ""

# ---------------------------------------------------------------------------
# [1/9] Prerequisites
# ---------------------------------------------------------------------------
echo "[1/9] Prerequisites"

if ! command -v jq &>/dev/null; then
    echo "  ERROR: jq is required but not installed"
    exit 1
fi
echo "  jq: $(jq --version 2>/dev/null || echo 'found')"

if [ "${SKIP_BUILD:-}" = "1" ] && [ -f "$BINARY" ]; then
    echo "  Binary: skipping build (SKIP_BUILD=1)"
elif [ -f "$BINARY" ]; then
    echo "  Binary: exists at $BINARY, skipping build"
else
    echo "  Building zeroned..."
    go build -o "$BINARY" ./cmd/zeroned
    echo "  Binary: built at $BINARY"
fi

if ! "$BINARY" version >/dev/null 2>&1; then
    echo "  ERROR: Binary at $BINARY does not run"
    exit 1
fi

# Find or install Go relayer
if [ -n "$RLY_BIN" ] && [ -f "$RLY_BIN" ]; then
    echo "  Relayer: using $RLY_BIN"
elif command -v rly &>/dev/null; then
    RLY_BIN="$(command -v rly)"
    echo "  Relayer: found rly at $RLY_BIN"
else
    echo "  Relayer: rly not found, installing..."
    go install github.com/cosmos/relayer/v2/cmd/rly@latest 2>/dev/null || true
    if command -v rly &>/dev/null; then
        RLY_BIN="$(command -v rly)"
        echo "  Relayer: installed rly at $RLY_BIN"
    else
        GOPATH_BIN="${GOPATH:-$HOME/go}/bin"
        if [ -f "$GOPATH_BIN/rly" ]; then
            RLY_BIN="$GOPATH_BIN/rly"
            echo "  Relayer: installed rly at $RLY_BIN"
        else
            echo "  ERROR: Could not install Go relayer (rly)."
            echo "  Install manually: go install github.com/cosmos/relayer/v2/cmd/rly@latest"
            exit 1
        fi
    fi
fi

# ---------------------------------------------------------------------------
# [2/9] Init Chain A
# ---------------------------------------------------------------------------
echo "[2/9] Initialize Chain A ($CHAIN_A_ID)"

zeroned_a init "$CHAIN_A_MONIKER" --chain-id "$CHAIN_A_ID" --default-denom "$DENOM" >/dev/null 2>&1
echo "  Home: $CHAIN_A_HOME"

zeroned_a keys add validator --keyring-backend test >/dev/null 2>&1
CHAIN_A_VALIDATOR=$(zeroned_a keys show validator -a --keyring-backend test)
echo "  Validator: $CHAIN_A_VALIDATOR"

CHAIN_A_USER_OUTPUT=$(zeroned_a keys add user --keyring-backend test --output json 2>&1)
CHAIN_A_USER=$(zeroned_a keys show user -a --keyring-backend test)
echo "  User: $CHAIN_A_USER"

zeroned_a add-genesis-account "$CHAIN_A_VALIDATOR" "1000000000000${DENOM}"
zeroned_a add-genesis-account "$CHAIN_A_USER" "500000000000${DENOM}"
echo "  Genesis accounts funded"

zeroned_a gentx validator "100000000000${DENOM}" \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --moniker "$CHAIN_A_MONIKER" >/dev/null 2>&1
zeroned_a collect-gentxs >/dev/null 2>&1

if [ -f "$SCRIPT_DIR/apply-testnet-config.sh" ]; then
    bash "$SCRIPT_DIR/apply-testnet-config.sh" "$CHAIN_A_HOME"
fi

# Chain A: Add per-(channel,denom) rate limits via genesis
GENESIS_A="$CHAIN_A_HOME/config/genesis.json"
jq '
    .app_state.ibcratelimit.params.enabled = true |
    .app_state.ibcratelimit.rate_limits = [
        {
            "channel_id": "channel-0",
            "denom": "uzrn",
            "max_send": "500000000000",
            "max_recv": "500000000000",
            "window_blocks": 10,
            "current_send": "0",
            "current_recv": "0",
            "window_start": 0
        }
    ]
' "$GENESIS_A" > "${GENESIS_A}.tmp" && mv "${GENESIS_A}.tmp" "$GENESIS_A"
echo "  Rate limits: 500K ZRN per window (10 blocks) on channel-0"

zeroned_a validate "$GENESIS_A" >/dev/null 2>&1
echo "  Genesis validated"

# ---------------------------------------------------------------------------
# [3/9] Init Chain B
# ---------------------------------------------------------------------------
echo "[3/9] Initialize Chain B ($CHAIN_B_ID)"

zeroned_b init "$CHAIN_B_MONIKER" --chain-id "$CHAIN_B_ID" --default-denom "$DENOM" >/dev/null 2>&1
echo "  Home: $CHAIN_B_HOME"

zeroned_b keys add validator --keyring-backend test >/dev/null 2>&1
CHAIN_B_VALIDATOR=$(zeroned_b keys show validator -a --keyring-backend test)
echo "  Validator: $CHAIN_B_VALIDATOR"

CHAIN_B_USER_OUTPUT=$(zeroned_b keys add user --keyring-backend test --output json 2>&1)
CHAIN_B_USER=$(zeroned_b keys show user -a --keyring-backend test)
echo "  User: $CHAIN_B_USER"

zeroned_b add-genesis-account "$CHAIN_B_VALIDATOR" "1000000000000${DENOM}"
zeroned_b add-genesis-account "$CHAIN_B_USER" "500000000000${DENOM}"
echo "  Genesis accounts funded"

zeroned_b gentx validator "100000000000${DENOM}" \
    --chain-id "$CHAIN_B_ID" \
    --keyring-backend test \
    --moniker "$CHAIN_B_MONIKER" >/dev/null 2>&1
zeroned_b collect-gentxs >/dev/null 2>&1

if [ -f "$SCRIPT_DIR/apply-testnet-config.sh" ]; then
    bash "$SCRIPT_DIR/apply-testnet-config.sh" "$CHAIN_B_HOME"
fi

GENESIS_B="$CHAIN_B_HOME/config/genesis.json"
zeroned_b validate "$GENESIS_B" >/dev/null 2>&1
echo "  Genesis validated"

# Configure Chain B ports (offset by +100)
CONFIG_B="$CHAIN_B_HOME/config/config.toml"
APP_B="$CHAIN_B_HOME/config/app.toml"

sed -i.bak "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:${CHAIN_B_P2P}\"|g" "$CONFIG_B"
sed -i.bak "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:${CHAIN_B_RPC}\"|g" "$CONFIG_B"
sed -i.bak "s|pprof_laddr = \"localhost:6060\"|pprof_laddr = \"localhost:${CHAIN_B_PPROF}\"|g" "$CONFIG_B"
sed -i.bak "s|address = \"tcp://localhost:1317\"|address = \"tcp://localhost:${CHAIN_B_REST}\"|g" "$APP_B"
sed -i.bak "s|address = \"localhost:9090\"|address = \"localhost:${CHAIN_B_GRPC}\"|g" "$APP_B"
sed -i.bak "s|address = \"localhost:9091\"|address = \"localhost:9191\"|g" "$APP_B"

rm -f "${CONFIG_B}.bak" "${APP_B}.bak"
echo "  Ports: P2P=$CHAIN_B_P2P RPC=$CHAIN_B_RPC REST=$CHAIN_B_REST gRPC=$CHAIN_B_GRPC"

# ---------------------------------------------------------------------------
# [4/9] Start Chains
# ---------------------------------------------------------------------------
echo "[4/9] Start chains"

zeroned_a start \
    --minimum-gas-prices "0${DENOM}" \
    --log_level error &
CHAIN_A_PID=$!
echo "  Chain A started (PID=$CHAIN_A_PID)"

zeroned_b start \
    --minimum-gas-prices "0${DENOM}" \
    --log_level error &
CHAIN_B_PID=$!
echo "  Chain B started (PID=$CHAIN_B_PID)"

echo "  Waiting for block 5 on Chain A..."
if ! wait_for_height "$CHAIN_A_RPC_URL" 5 90; then
    echo "  ERROR: Chain A failed to reach block 5"
    exit 1
fi
echo "  Chain A: block $(get_height "$CHAIN_A_RPC_URL")"

echo "  Waiting for block 5 on Chain B..."
if ! wait_for_height "$CHAIN_B_RPC_URL" 5 90; then
    echo "  ERROR: Chain B failed to reach block 5"
    exit 1
fi
echo "  Chain B: block $(get_height "$CHAIN_B_RPC_URL")"

# ---------------------------------------------------------------------------
# [5/9] Relayer Setup
# ---------------------------------------------------------------------------
echo "[5/9] Relayer setup"

"$RLY_BIN" config init --home "$RLY_HOME" 2>/dev/null

CHAIN_A_CONFIG=$(mktemp)
cat > "$CHAIN_A_CONFIG" <<EOF
{
    "type": "cosmos",
    "value": {
        "key-directory": "$RLY_HOME/keys/$CHAIN_A_ID",
        "key": "relayer",
        "chain-id": "$CHAIN_A_ID",
        "rpc-addr": "http://localhost:${CHAIN_A_RPC}",
        "account-prefix": "zrn",
        "keyring-backend": "test",
        "gas-adjustment": 1.5,
        "gas-prices": "0${DENOM}",
        "min-gas-amount": 0,
        "max-gas-amount": 0,
        "debug": false,
        "timeout": "20s",
        "block-timeout": "",
        "output-format": "json",
        "sign-mode": "direct",
        "extra-codecs": []
    }
}
EOF

CHAIN_B_CONFIG=$(mktemp)
cat > "$CHAIN_B_CONFIG" <<EOF
{
    "type": "cosmos",
    "value": {
        "key-directory": "$RLY_HOME/keys/$CHAIN_B_ID",
        "key": "relayer",
        "chain-id": "$CHAIN_B_ID",
        "rpc-addr": "http://localhost:${CHAIN_B_RPC}",
        "account-prefix": "zrn",
        "keyring-backend": "test",
        "gas-adjustment": 1.5,
        "gas-prices": "0${DENOM}",
        "min-gas-amount": 0,
        "max-gas-amount": 0,
        "debug": false,
        "timeout": "20s",
        "block-timeout": "",
        "output-format": "json",
        "sign-mode": "direct",
        "extra-codecs": []
    }
}
EOF

"$RLY_BIN" chains add --file "$CHAIN_A_CONFIG" "$CHAIN_A_ID" --home "$RLY_HOME" 2>/dev/null
"$RLY_BIN" chains add --file "$CHAIN_B_CONFIG" "$CHAIN_B_ID" --home "$RLY_HOME" 2>/dev/null
rm -f "$CHAIN_A_CONFIG" "$CHAIN_B_CONFIG"
echo "  Chains added to relayer"

"$RLY_BIN" keys add "$CHAIN_A_ID" relayer --home "$RLY_HOME" 2>/dev/null
"$RLY_BIN" keys add "$CHAIN_B_ID" relayer --home "$RLY_HOME" 2>/dev/null

RLY_ADDR_A=$("$RLY_BIN" keys show "$CHAIN_A_ID" relayer --home "$RLY_HOME" 2>/dev/null)
RLY_ADDR_B=$("$RLY_BIN" keys show "$CHAIN_B_ID" relayer --home "$RLY_HOME" 2>/dev/null)
echo "  Relayer address on A: $RLY_ADDR_A"
echo "  Relayer address on B: $RLY_ADDR_B"

"$RLY_BIN" paths new "$CHAIN_A_ID" "$CHAIN_B_ID" "$IBC_PATH" --home "$RLY_HOME" 2>/dev/null
echo "  Path created: $IBC_PATH"

# ---------------------------------------------------------------------------
# [6/9] Fund Relayer
# ---------------------------------------------------------------------------
echo "[6/9] Fund relayer accounts"

FUND_TX_A=$(zeroned_a tx bank send "$CHAIN_A_VALIDATOR" "$RLY_ADDR_A" "10000000000${DENOM}" \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_A_RPC}" 2>/dev/null | jq -r '.txhash // empty')
echo "  Funded relayer on A: tx=$FUND_TX_A"

settle

FUND_TX_B=$(zeroned_b tx bank send "$CHAIN_B_VALIDATOR" "$RLY_ADDR_B" "10000000000${DENOM}" \
    --chain-id "$CHAIN_B_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_B_RPC}" 2>/dev/null | jq -r '.txhash // empty')
echo "  Funded relayer on B: tx=$FUND_TX_B"

settle

RLY_BAL_A=$(get_balance "$CHAIN_A_REST_URL" "$RLY_ADDR_A" "$DENOM")
RLY_BAL_B=$(get_balance "$CHAIN_B_REST_URL" "$RLY_ADDR_B" "$DENOM")
echo "  Relayer balance on A: ${RLY_BAL_A} ${DENOM}"
echo "  Relayer balance on B: ${RLY_BAL_B} ${DENOM}"

if [ "$RLY_BAL_A" = "0" ] || [ -z "$RLY_BAL_A" ]; then
    echo "  ERROR: Relayer on A has no funds"
    exit 1
fi

echo "  Linking chains (client + connection + channel)..."
if ! "$RLY_BIN" transact link "$IBC_PATH" --home "$RLY_HOME" 2>/dev/null; then
    echo "  ERROR: Failed to link chains via relayer"
    echo "  Trying with debug output:"
    "$RLY_BIN" transact link "$IBC_PATH" --home "$RLY_HOME" --debug 2>&1 | tail -20
    exit 1
fi
echo "  Chains linked"

CHANNEL_A=$(curl -s "${CHAIN_A_REST_URL}/ibc/core/channel/v1/channels" 2>/dev/null | \
    jq -r '.channels[0].channel_id // empty' 2>/dev/null || echo "")
CHANNEL_B=$(curl -s "${CHAIN_B_REST_URL}/ibc/core/channel/v1/channels" 2>/dev/null | \
    jq -r '.channels[0].channel_id // empty' 2>/dev/null || echo "")

if [ -z "$CHANNEL_A" ] || [ -z "$CHANNEL_B" ]; then
    echo "  ERROR: Could not discover IBC channel IDs"
    exit 1
fi
echo "  Channel A: $CHANNEL_A"
echo "  Channel B: $CHANNEL_B"

"$RLY_BIN" start "$IBC_PATH" --home "$RLY_HOME" 2>/dev/null &
RLY_PID=$!
echo "  Relayer started (PID=$RLY_PID)"

sleep 5

# ---------------------------------------------------------------------------
# [7/9] IBC Tests
# ---------------------------------------------------------------------------
echo "[7/9] Running IBC tests..."
echo ""

# --- Test 1: Basic A->B Transfer ---
echo "  Test 1: Basic IBC transfer A->B (200K ZRN)"

BEFORE_BAL_A=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")

TX1_HASH=$(zeroned_a tx ibc-transfer transfer transfer "$CHANNEL_A" "$CHAIN_B_USER" "200000000000${DENOM}" \
    --from user \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_A_RPC}" 2>/dev/null | jq -r '.txhash // empty')
echo "    Sent tx: $TX1_HASH"

sleep 15

AFTER_BAL_A=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")
IBC_BAL_B=$(get_ibc_balance "$CHAIN_B_REST_URL" "$CHAIN_B_USER")
IBC_DENOM_B=$(get_ibc_denom "$CHAIN_B_REST_URL" "$CHAIN_B_USER")

echo "    User balance on A after: ${AFTER_BAL_A}"
echo "    User IBC balance on B: ${IBC_BAL_B} (denom: ${IBC_DENOM_B})"

run_test_eval "Basic A->B: sender balance decreased" \
    "[ \"$AFTER_BAL_A\" -lt \"$BEFORE_BAL_A\" ]"

run_test_eval "Basic A->B: receiver got IBC tokens" \
    "[ -n \"$IBC_DENOM_B\" ] && [ \"$IBC_BAL_B\" = \"200000000000\" ]"

settle

# --- Test 2: Return B->A ---
echo ""
echo "  Test 2: Return IBC transfer B->A (100K IBC-ZRN)"

if [ -z "$IBC_DENOM_B" ]; then
    skip_test "Return B->A" "no IBC denom found on B"
else
    BEFORE_BAL_A2=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")

    TX2_HASH=$(zeroned_b tx ibc-transfer transfer transfer "$CHANNEL_B" "$CHAIN_A_USER" "100000000000${IBC_DENOM_B}" \
        --from user \
        --chain-id "$CHAIN_B_ID" \
        --keyring-backend test \
        --fees "0${DENOM}" \
        --broadcast-mode sync \
        --yes \
        --output json \
        --node "tcp://localhost:${CHAIN_B_RPC}" 2>/dev/null | jq -r '.txhash // empty')
    echo "    Sent tx: $TX2_HASH"

    sleep 15

    AFTER_BAL_A2=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")
    REMAINING_IBC_B=$(get_ibc_balance "$CHAIN_B_REST_URL" "$CHAIN_B_USER")

    echo "    User balance on A after return: ${AFTER_BAL_A2} (was ${BEFORE_BAL_A2})"
    echo "    Remaining IBC balance on B: ${REMAINING_IBC_B}"

    run_test_eval "Return B->A: original denom restored on A" \
        "[ \"$AFTER_BAL_A2\" -gt \"$BEFORE_BAL_A2\" ]"

    run_test_eval "Return B->A: IBC balance decreased on B" \
        "[ \"$REMAINING_IBC_B\" = \"100000000000\" ]"

    settle
fi

# --- Test 3: Rate Limit Hit ---
echo ""
echo "  Test 3: Rate limit enforcement (3x 200K > 500K limit)"

TX3A_HASH=$(zeroned_a tx ibc-transfer transfer transfer "$CHANNEL_A" "$CHAIN_B_USER" "200000000000${DENOM}" \
    --from user \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_A_RPC}" 2>/dev/null | jq -r '.txhash // empty')
echo "    Sent 2nd 200K: tx=$TX3A_HASH"

sleep 10

TX3B_OUTPUT=$(zeroned_a tx ibc-transfer transfer transfer "$CHANNEL_A" "$CHAIN_B_USER" "200000000000${DENOM}" \
    --from user \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_A_RPC}" 2>&1 || true)
TX3B_CODE=$(echo "$TX3B_OUTPUT" | jq -r '.code // "0"' 2>/dev/null || echo "unknown")

echo "    3rd 200K result: code=$TX3B_CODE"

if [ "$TX3B_CODE" != "0" ] && [ "$TX3B_CODE" != "unknown" ]; then
    echo "  PASS  Rate limit: 3rd transfer blocked at send (code=$TX3B_CODE)"
    PASS=$((PASS + 1))
else
    sleep 15
    IBC_BAL_B_AFTER_RATE=$(get_ibc_balance "$CHAIN_B_REST_URL" "$CHAIN_B_USER")
    echo "    IBC balance on B after rate test: $IBC_BAL_B_AFTER_RATE"
    if [ "$IBC_BAL_B_AFTER_RATE" -le "400000000000" ] 2>/dev/null; then
        echo "  PASS  Rate limit: outflow capped (B has ${IBC_BAL_B_AFTER_RATE})"
        PASS=$((PASS + 1))
    else
        echo "  FAIL  Rate limit: outflow exceeded (B has ${IBC_BAL_B_AFTER_RATE})"
        FAIL=$((FAIL + 1))
    fi
fi

settle

# --- Test 4: Rate Limit Recovery ---
echo ""
echo "  Test 4: Rate limit recovery after window reset"

echo "    Waiting for 2+ windows (25+ blocks) for rate limit reset..."
CURRENT_HEIGHT=$(get_height "$CHAIN_A_RPC_URL")
TARGET_HEIGHT=$((CURRENT_HEIGHT + 25))

if ! wait_for_height "$CHAIN_A_RPC_URL" "$TARGET_HEIGHT" 120; then
    echo "  FAIL  Rate limit recovery: timed out waiting for blocks"
    FAIL=$((FAIL + 1))
else
    BEFORE_RECOVERY=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")
    TX4_HASH=$(zeroned_a tx ibc-transfer transfer transfer "$CHANNEL_A" "$CHAIN_B_USER" "10000000000${DENOM}" \
        --from user \
        --chain-id "$CHAIN_A_ID" \
        --keyring-backend test \
        --fees "0${DENOM}" \
        --broadcast-mode sync \
        --yes \
        --output json \
        --node "tcp://localhost:${CHAIN_A_RPC}" 2>/dev/null | jq -r '.txhash // empty')
    echo "    Recovery transfer tx: $TX4_HASH"

    if [ -n "$TX4_HASH" ]; then
        sleep 10
        AFTER_RECOVERY=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")
        if [ "$AFTER_RECOVERY" -lt "$BEFORE_RECOVERY" ] 2>/dev/null; then
            echo "  PASS  Rate limit recovery: transfer succeeded after window reset"
            PASS=$((PASS + 1))
        else
            echo "  FAIL  Rate limit recovery: balance unchanged"
            FAIL=$((FAIL + 1))
        fi
    else
        echo "  FAIL  Rate limit recovery: could not broadcast transfer"
        FAIL=$((FAIL + 1))
    fi
fi

settle

# --- Test 5: Multi-Denom (SKIP) ---
echo ""
skip_test "Multi-denom rate limiting" "only uzrn at genesis; per-(channel,denom) model"

# --- Test 6: Timeout ---
echo ""
echo "  Test 6: Packet timeout and refund"

BEFORE_TIMEOUT=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")

TX6_HASH=$(zeroned_a tx ibc-transfer transfer transfer "$CHANNEL_A" "$CHAIN_B_USER" "10000000000${DENOM}" \
    --from user \
    --chain-id "$CHAIN_A_ID" \
    --keyring-backend test \
    --fees "0${DENOM}" \
    --packet-timeout-timestamp 1 \
    --broadcast-mode sync \
    --yes \
    --output json \
    --node "tcp://localhost:${CHAIN_A_RPC}" 2>/dev/null | jq -r '.txhash // empty')
echo "    Timeout transfer tx: $TX6_HASH"

sleep 20

AFTER_TIMEOUT=$(get_balance "$CHAIN_A_REST_URL" "$CHAIN_A_USER" "$DENOM")
echo "    Balance after timeout: $AFTER_TIMEOUT"

TIMEOUT_DIFF=$((BEFORE_TIMEOUT - AFTER_TIMEOUT))
if [ "$TIMEOUT_DIFF" -ge 0 ] && [ "$TIMEOUT_DIFF" -le 1000000 ] 2>/dev/null; then
    echo "  PASS  Timeout: tokens refunded (diff=${TIMEOUT_DIFF})"
    PASS=$((PASS + 1))
elif [ "$AFTER_TIMEOUT" -ge "$BEFORE_TIMEOUT" ] 2>/dev/null; then
    echo "  PASS  Timeout: tokens refunded (balance equal or higher)"
    PASS=$((PASS + 1))
else
    echo "  FAIL  Timeout: tokens not refunded (lost ${TIMEOUT_DIFF})"
    FAIL=$((FAIL + 1))
fi

settle

# --- Test 7: Channel State ---
echo ""
echo "  Test 7: Packet relay and channel state verification"

CHANNEL_STATE_A=$(curl -s "${CHAIN_A_REST_URL}/ibc/core/channel/v1/channels/${CHANNEL_A}/ports/transfer" 2>/dev/null | \
    jq -r '.channel.state // "unknown"' 2>/dev/null || echo "unknown")
CHANNEL_STATE_B=$(curl -s "${CHAIN_B_REST_URL}/ibc/core/channel/v1/channels/${CHANNEL_B}/ports/transfer" 2>/dev/null | \
    jq -r '.channel.state // "unknown"' 2>/dev/null || echo "unknown")

echo "    Channel A state: $CHANNEL_STATE_A"
echo "    Channel B state: $CHANNEL_STATE_B"

run_test_eval "Channel A is OPEN" \
    "echo '$CHANNEL_STATE_A' | grep -iqE 'open'"
run_test_eval "Channel B is OPEN" \
    "echo '$CHANNEL_STATE_B' | grep -iqE 'open'"

COMMITMENTS_A=$(curl -s "${CHAIN_A_REST_URL}/ibc/core/channel/v1/channels/${CHANNEL_A}/ports/transfer/packet_commitments" 2>/dev/null | \
    jq -r '.commitments | length // 0' 2>/dev/null || echo "unknown")

echo "    Unrelayed packet commitments on A: $COMMITMENTS_A"

run_test_eval "All packets relayed (no pending commitments)" \
    "[ \"$COMMITMENTS_A\" = \"0\" ] || [ \"$COMMITMENTS_A\" = \"null\" ]"

# ---------------------------------------------------------------------------
# [8/9] Results
# ---------------------------------------------------------------------------
FINAL_HEIGHT_A=$(get_height "$CHAIN_A_RPC_URL")
FINAL_HEIGHT_B=$(get_height "$CHAIN_B_RPC_URL")

echo ""
echo "[8/9] Results"
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "  Skipped: $SKIP"
echo "  Chain A final height: $FINAL_HEIGHT_A"
echo "  Chain B final height: $FINAL_HEIGHT_B"

# ---------------------------------------------------------------------------
# [9/9] Cleanup (handled by trap)
# ---------------------------------------------------------------------------
echo ""
echo "[9/9] Cleanup (automatic via trap)"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "  FAILED: $FAIL IBC test(s) failed"
    exit 1
fi

echo ""
echo "  All IBC tests passed! ($PASS passed, $SKIP skipped)"
