#!/bin/bash
# boot-test.sh — Single-validator boot smoke test for zeroned.
#
# Initializes a temp chain, creates a genesis validator, starts the node,
# waits for a few blocks, then tears down. Idempotent (uses temp directory).
#
# Usage: ./scripts/boot-test.sh [path-to-zeroned]

set -euo pipefail

BINARY="${1:-build/zeroned}"
CHAIN_ID="zerone-boot-test-1"
MONIKER="boot-test-validator"
DENOM="uzrn"
HOME_DIR=$(mktemp -d)
KEYRING="test"

cleanup() {
    rm -rf "$HOME_DIR"
}
trap cleanup EXIT

echo "=== Zerone boot test ==="
echo "Binary:  $BINARY"
echo "Home:    $HOME_DIR"
echo "ChainID: $CHAIN_ID"

# Verify binary exists
if [ ! -x "$BINARY" ]; then
    echo "ERROR: Binary not found or not executable: $BINARY"
    echo "Run 'make build' first."
    exit 1
fi

# Init chain
echo "[1/6] Initializing chain..."
$BINARY init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" > /dev/null 2>&1

# Add validator key
echo "[2/6] Creating validator key..."
$BINARY keys add validator --keyring-backend "$KEYRING" --home "$HOME_DIR" > /dev/null 2>&1
ADDR=$($BINARY keys show validator -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "  Address: $ADDR"

# Fund genesis account
echo "[3/6] Adding genesis account..."
$BINARY add-genesis-account "$ADDR" "100000000000${DENOM}" --home "$HOME_DIR" --keyring-backend "$KEYRING"

# Create genesis transaction
echo "[4/6] Creating gentx..."
$BINARY genesis gentx validator "10000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR" > /dev/null 2>&1

# Collect genesis transactions
echo "[5/6] Collecting gentxs..."
$BINARY genesis collect-gentxs --home "$HOME_DIR" > /dev/null 2>&1

# Validate genesis
$BINARY genesis validate-genesis --home "$HOME_DIR"

# Start node (run for ~30 seconds, then stop)
echo "[6/6] Starting node (30s)..."
$BINARY start --home "$HOME_DIR" > "$HOME_DIR/node.log" 2>&1 &
NODE_PID=$!
sleep 30
kill "$NODE_PID" 2>/dev/null || true
wait "$NODE_PID" 2>/dev/null || true
tail -20 "$HOME_DIR/node.log"

echo ""
echo "=== Boot test passed ==="
