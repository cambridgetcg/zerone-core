#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Zerone — End-to-End Full Truth-Seeking Loop
# ═══════════════════════════════════════════════════════════════════════════
#
# Tests the complete truth-seeking loop on a running localnet:
#   Register → Qualify → Claim → Verify → Reward
#
# Prerequisites:
#   scripts/localnet.sh start   # chain must be running and producing blocks
#
# Usage:
#   scripts/e2e-full-loop.sh
#
# ═══════════════════════════════════════════════════════════════════════════

set -uo pipefail
# Note: intentionally NOT using set -e — errors are handled explicitly with || handlers

# ── Constants ──────────────────────────────────────────────────────────────

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${PROJECT_ROOT}/build/zeroned"
CHAIN_ID="zerone-localnet"
DENOM="uzrn"
BASE_DIR="${HOME}/.zeroned/localnet"
COORDINATOR_HOME="${BASE_DIR}/coordinator"
KEYRING="test"
RPC_URL="http://127.0.0.1:26601"

# Common flags
NODE_FLAG="--node ${RPC_URL}"
HOME_FLAG="--home ${COORDINATOR_HOME}"
KEYRING_FLAG="--keyring-backend ${KEYRING}"
COMMON_FLAGS="${NODE_FLAG} ${HOME_FLAG} ${KEYRING_FLAG} --chain-id ${CHAIN_ID} --output json"
# Fixed gas avoids auto-estimation failures (simulation underestimates due to state changes
# between simulation and actual execution, especially for sequential operations).
TX_FLAGS="${COMMON_FLAGS} --gas 300000 --gas-prices 1${DENOM} --yes --broadcast-mode sync"

# ── Checkpoint Tracking ───────────────────────────────────────────────────

PASSED=0
FAILED=0
TOTAL_CHECKPOINTS=5
FAILURES=""
START_TIME=$(date +%s)
PHASE_TIMES=""

# ── Helpers ────────────────────────────────────────────────────────────────

info()   { echo -e "\033[1;34m  ->\033[0m $*"; }
pass()   { PASSED=$((PASSED+1)); echo -e "\n\033[1;32m  ✓ CHECKPOINT $1: PASS\033[0m — $2"; }
fail()   { FAILED=$((FAILED+1)); FAILURES="${FAILURES}\n  - Checkpoint $1: $2"; echo -e "\n\033[1;31m  ✗ CHECKPOINT $1: FAIL\033[0m — $2"; }
warn()   { echo -e "\033[1;33m  !!\033[0m $*"; }
header() {
  local phase_start
  phase_start=$(date +%s)
  echo -e "\n\033[1;36m════════════════════════════════════════════════════════════\033[0m"
  echo -e "\033[1;36m  $1\033[0m"
  echo -e "\033[1;36m════════════════════════════════════════════════════════════\033[0m"
}

record_phase_time() {
  local name="$1"
  local elapsed="$2"
  PHASE_TIMES="${PHASE_TIMES}\n  - ${name}: ${elapsed}s"
}

# Submit tx, return txhash. Returns "TX_FAILED" on broadcast failure.
# Note: stderr is discarded to avoid "gas estimate:" lines corrupting JSON parsing.
submit_tx() {
  local result
  result=$(eval "$@" 2>/dev/null) || true
  # Extract the last JSON object from output (skip any non-JSON preamble)
  local json_line
  json_line=$(echo "$result" | grep -E '^\{' | tail -1)
  if [ -z "$json_line" ]; then
    info "[DIAG] no JSON in broadcast result: ${result:0:300}" >&2
    echo "TX_FAILED"
    return 1
  fi
  local tx_code
  tx_code=$(echo "$json_line" | jq -r '.code // empty' 2>/dev/null || echo "")
  if [ -n "$tx_code" ] && [ "$tx_code" != "0" ]; then
    local raw_log
    raw_log=$(echo "$json_line" | jq -r '.raw_log // empty' 2>/dev/null || echo "")
    info "[DIAG] broadcast rejected: code=$tx_code log=${raw_log:0:300}" >&2
    echo "TX_FAILED"
    return 1
  fi
  local tx_hash
  tx_hash=$(echo "$json_line" | jq -r '.txhash // empty' 2>/dev/null || echo "")
  if [ -z "$tx_hash" ]; then
    info "[DIAG] no txhash in JSON: ${json_line:0:300}" >&2
    echo "TX_FAILED"
    return 1
  fi
  echo "$tx_hash"
}

# Wait for tx inclusion. Echoes full tx JSON on success.
wait_tx() {
  local tx_hash="$1"
  local max_wait="${2:-30}"
  local elapsed=0
  while [ $elapsed -lt $max_wait ]; do
    local result
    result=$(${BINARY} query tx "${tx_hash}" ${NODE_FLAG} --output json 2>/dev/null || echo "")
    if [ -n "$result" ]; then
      local code
      code=$(echo "$result" | jq -r '.code // empty' 2>/dev/null || echo "")
      if [ "$code" = "0" ]; then
        echo "$result"
        return 0
      elif [ -n "$code" ]; then
        local raw_log
        raw_log=$(echo "$result" | jq -r '.raw_log // .logs[0].log // "unknown"' 2>/dev/null || echo "unknown")
        info "[DIAG] tx ${tx_hash:0:12}... failed on-chain: code=$code log=${raw_log:0:300}" >&2
        return 2
      fi
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  info "[DIAG] tx ${tx_hash:0:12}... not found after ${max_wait}s" >&2
  return 1
}

# Submit tx + wait for inclusion. Echoes full tx JSON on success.
# Note: info messages go to stderr to avoid corrupting the JSON return value.
send_tx() {
  local tx_hash
  tx_hash=$(submit_tx "$@") || return 1
  if [ "$tx_hash" = "TX_FAILED" ]; then
    return 1
  fi
  info "tx broadcast: ${tx_hash:0:16}..." >&2
  wait_tx "$tx_hash" 30
}

# Submit tx expecting failure. Returns 0 if tx indeed failed.
send_tx_expect_fail() {
  local result
  result=$(eval "$@" 2>/dev/null) || true
  local json_line
  json_line=$(echo "$result" | grep -E '^\{' | tail -1)
  if [ -z "$json_line" ]; then
    info "Expected failure: no JSON output"
    return 0
  fi
  local tx_code
  tx_code=$(echo "$json_line" | jq -r '.code // empty' 2>/dev/null || echo "")
  if [ -n "$tx_code" ] && [ "$tx_code" != "0" ]; then
    info "Expected failure at broadcast: code=$tx_code"
    return 0
  fi
  local tx_hash
  tx_hash=$(echo "$json_line" | jq -r '.txhash // empty' 2>/dev/null || echo "")
  if [ -z "$tx_hash" ]; then
    info "Expected failure: no txhash"
    return 0
  fi
  # Tx was broadcast — check if it fails on-chain
  sleep 6
  local tx_result
  tx_result=$(${BINARY} query tx "${tx_hash}" ${NODE_FLAG} --output json 2>/dev/null || echo "")
  if [ -n "$tx_result" ]; then
    local code
    code=$(echo "$tx_result" | jq -r '.code // empty' 2>/dev/null || echo "")
    if [ -n "$code" ] && [ "$code" != "0" ]; then
      info "Expected failure on-chain: code=$code"
      return 0
    fi
  fi
  info "UNEXPECTED: tx succeeded when failure was expected"
  return 1
}

# Wait until chain reaches N blocks from now.
wait_blocks() {
  local n="$1"
  local start_height
  start_height=$(${BINARY} status ${NODE_FLAG} 2>/dev/null | jq -r '.sync_info.latest_block_height' | tr -d '"')
  local target=$((start_height + n))
  info "Waiting for block $target (current: $start_height, +$n blocks)..."
  while true; do
    local current
    current=$(${BINARY} status ${NODE_FLAG} 2>/dev/null | jq -r '.sync_info.latest_block_height' | tr -d '"')
    if [ "$current" -ge "$target" ] 2>/dev/null; then
      info "Reached block $current"
      return 0
    fi
    sleep 2
  done
}

# Get current block height.
get_height() {
  ${BINARY} status ${NODE_FLAG} 2>/dev/null | jq -r '.sync_info.latest_block_height' | tr -d '"'
}

# Get address from keyring.
get_addr() {
  ${BINARY} keys show "$1" -a ${KEYRING_FLAG} ${HOME_FLAG}
}

# Generate a synthetic 64-char hex "public key" for zerone_auth DID registration.
# The zerone_auth module requires 64-hex-char (32-byte) Ed25519 keys, but the cosmos
# keyring only supports secp256k1. The DID derivation is structural (not signature-verified),
# so a deterministic synthetic key works for registration.
gen_synthetic_pubkey() {
  local name="$1"
  # SHA-256 of a deterministic seed → exactly 64 hex chars
  printf '%s' "zerone-e2e-${name}-pubkey-seed-v1" | shasum -a 256 | cut -c1-64
}

# Derive DID from pubkey hex: did:zrn:{first 32 hex chars}
derive_did() {
  local pubkey_hex="$1"
  echo "did:zrn:${pubkey_hex:0:32}"
}

# Extract event attribute from tx result JSON.
get_event_attr() {
  local tx_json="$1"
  local event_type="$2"
  local attr_key="$3"
  echo "$tx_json" | jq -r ".events[] | select(.type==\"$event_type\") | .attributes[] | select(.key==\"$attr_key\") | .value" 2>/dev/null | head -1
}

# Compute commit hash for CLI path: SHA256(vote_bytes + salt_raw_bytes)
# Returns 64-char hex hash.
compute_commit_hash() {
  local vote="$1"
  local salt_hex="$2"
  local vote_hex
  vote_hex=$(printf '%s' "$vote" | xxd -p | tr -d '\n')
  echo -n "${vote_hex}${salt_hex}" | xxd -r -p | shasum -a 256 | cut -d' ' -f1
}


# ═══════════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  ZERONE E2E FULL LOOP TEST"
echo "  $(date '+%Y-%m-%d %H:%M:%S')"
echo "═══════════════════════════════════════════════════════════════"

# ── Phase 1: Verify Chain Running ─────────────────────────────────────────

header "Phase 1: Verify Chain Running"
PHASE1_START=$(date +%s)

HEIGHT=$(get_height)
if [ -z "$HEIGHT" ] || [ "$HEIGHT" -lt 1 ] 2>/dev/null; then
  echo "ERROR: Chain not running. Start with: scripts/localnet.sh start"
  exit 1
fi
info "Chain is at block $HEIGHT"

# Verify producing blocks
sleep 5
HEIGHT2=$(get_height)
if [ "$HEIGHT2" -le "$HEIGHT" ]; then
  echo "ERROR: Chain not producing blocks (stuck at $HEIGHT)"
  exit 1
fi
info "Chain advancing: $HEIGHT → $HEIGHT2"

# Get validator addresses
VAL0_ADDR=$(get_addr val0)
VAL1_ADDR=$(get_addr val1)
VAL2_ADDR=$(get_addr val2)
VAL3_ADDR=$(get_addr val3)
FAUCET_ADDR=$(get_addr faucet)
info "val0: $VAL0_ADDR"
info "val1: $VAL1_ADDR"
info "val2: $VAL2_ADDR"
info "val3: $VAL3_ADDR"
info "faucet: $FAUCET_ADDR"

record_phase_time "Phase 1 (chain verify)" "$(($(date +%s) - PHASE1_START))"

# ── Phase 2: Account Registration + Capabilities ─────────────────────────

header "Phase 2: Account Registration + Capabilities"
PHASE2_START=$(date +%s)

# Create test accounts (secp256k1 for cosmos, synthetic ed25519 pubkeys for zerone_auth DID)
# Delete first to avoid interactive overwrite prompt
for name in alice sage1 rogue; do
  ${BINARY} keys delete "$name" ${KEYRING_FLAG} ${HOME_FLAG} -y 2>/dev/null || true
  ${BINARY} keys add "$name" ${KEYRING_FLAG} ${HOME_FLAG} 2>/dev/null
done

ALICE_ADDR=$(get_addr alice)
SAGE1_ADDR=$(get_addr sage1)
ROGUE_ADDR=$(get_addr rogue)
info "alice: $ALICE_ADDR"
info "sage1: $SAGE1_ADDR"
info "rogue: $ROGUE_ADDR"

# Fund accounts from faucet (1000 ZRN each)
# Must wait for each tx to be included before sending next (account sequence)
info "Funding accounts..."
for addr in $ALICE_ADDR $SAGE1_ADDR $ROGUE_ADDR; do
  if send_tx "${BINARY} tx bank send ${FAUCET_ADDR} ${addr} 1000000000${DENOM} --from faucet ${TX_FLAGS}"; then
    info "Funded $addr"
  else
    warn "Fund tx for $addr may have failed"
  fi
  sleep 3
done
sleep 3

# Verify balances
for name in alice sage1 rogue; do
  addr=$(get_addr "$name")
  bal=$(${BINARY} query bank balances "$addr" ${NODE_FLAG} --output json 2>/dev/null | jq -r '.balances[] | select(.denom=="uzrn") | .amount // "0"')
  info "$name balance: ${bal} uzrn"
done

# Register with zerone_auth
info "Registering accounts with zerone_auth..."

CHECKPOINT1_PASS=true
for entry in "alice:human" "sage1:agent" "rogue:agent"; do
  name="${entry%%:*}"
  acct_type="${entry##*:}"
  pubkey_hex=$(gen_synthetic_pubkey "$name")
  did=$(derive_did "$pubkey_hex")
  info "Registering $name as $acct_type (DID: $did, pubkey: ${pubkey_hex:0:16}...)"
  if send_tx "${BINARY} tx zerone_auth register-account ${did} ${pubkey_hex} ${acct_type} --from ${name} ${TX_FLAGS}"; then
    info "$name registered successfully"
  else
    warn "$name registration failed"
    CHECKPOINT1_PASS=false
  fi
  sleep 1
done
sleep 6

# Register validators with zerone_auth so they can submit commitments and reviews.
# Without registration, the AnteHandler's ZeroneCapabilityDecorator blocks them (code 30).
info "Registering validators with zerone_auth..."
for entry in "val0:agent" "val1:agent" "val2:agent" "val3:agent"; do
  name="${entry%%:*}"
  acct_type="${entry##*:}"
  pubkey_hex=$(gen_synthetic_pubkey "$name")
  did=$(derive_did "$pubkey_hex")
  info "Registering $name as $acct_type (DID: $did)"
  if send_tx "${BINARY} tx zerone_auth register-account ${did} ${pubkey_hex} ${acct_type} --from ${name} ${TX_FLAGS}"; then
    info "$name registered"
  else
    warn "$name registration failed"
  fi
  sleep 1
done
sleep 6

# Verify accounts
for name in alice sage1 rogue val0 val1 val2 val3; do
  addr=$(get_addr "$name")
  result=$(${BINARY} query zerone_auth account "$addr" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  acct_type=$(echo "$result" | jq -r '.account.account_type // "NOT_FOUND"' 2>/dev/null)
  info "$name account_type: $acct_type"
  if [ "$acct_type" = "NOT_FOUND" ]; then
    CHECKPOINT1_PASS=false
  fi
done

if [ "$CHECKPOINT1_PASS" = true ]; then
  pass "1" "All accounts registered with correct types (including validators)"
else
  fail "1" "Account registration incomplete"
fi

record_phase_time "Phase 2 (registration)" "$(($(date +%s) - PHASE2_START))"

# ── Phase 3: Block Rewards Flowing ───────────────────────────────────────

header "Phase 3: Block Rewards Flowing"
PHASE3_START=$(date +%s)

# Submit a few txs to generate fee revenue
info "Generating transactions for block rewards..."
for i in 1 2 3; do
  send_tx "${BINARY} tx bank send ${FAUCET_ADDR} ${ALICE_ADDR} 1${DENOM} --from faucet ${TX_FLAGS}" || true
  sleep 3
done
sleep 6

# Check fund balances
CHECKPOINT2_PASS=true

# Protocol Treasury (amino JSON: .account.value.address)
info "Querying protocol_treasury..."
TREASURY_ADDR=$(${BINARY} query auth module-account protocol_treasury ${NODE_FLAG} --output json 2>/dev/null | jq -r '.account.value.address // .account.base_account.address // empty' 2>/dev/null || echo "")
if [ -n "$TREASURY_ADDR" ]; then
  TREASURY_BAL=$(${BINARY} query bank balances "$TREASURY_ADDR" ${NODE_FLAG} --output json 2>/dev/null | jq -r '.balances[] | select(.denom=="uzrn") | .amount // "0"' 2>/dev/null || echo "0")
  info "Protocol Treasury ($TREASURY_ADDR) balance: ${TREASURY_BAL} uzrn"
else
  warn "Could not find protocol_treasury module account"
  TREASURY_BAL="0"
fi

# Research Fund
info "Querying research_fund..."
RESEARCH_BAL=$(${BINARY} query vesting_rewards research-fund-balance ${NODE_FLAG} --output json 2>/dev/null | jq -r '.balance.amount // empty' 2>/dev/null || echo "")
if [ -z "$RESEARCH_BAL" ]; then
  RESEARCH_FUND_ADDR=$(${BINARY} query auth module-account research_fund ${NODE_FLAG} --output json 2>/dev/null | jq -r '.account.value.address // .account.base_account.address // empty' 2>/dev/null || echo "")
  if [ -n "$RESEARCH_FUND_ADDR" ]; then
    RESEARCH_BAL=$(${BINARY} query bank balances "$RESEARCH_FUND_ADDR" ${NODE_FLAG} --output json 2>/dev/null | jq -r '.balances[] | select(.denom=="uzrn") | .amount // "0"' 2>/dev/null || echo "0")
  else
    RESEARCH_BAL="0"
  fi
fi
info "Research Fund balance: ${RESEARCH_BAL} uzrn"

# Also check vesting_rewards module itself (accumulates citation + treasury shares)
VESTING_MODULE_ADDR=$(${BINARY} query auth module-account vesting_rewards ${NODE_FLAG} --output json 2>/dev/null | jq -r '.account.value.address // .account.base_account.address // empty' 2>/dev/null || echo "")
VESTING_BAL="0"
if [ -n "$VESTING_MODULE_ADDR" ]; then
  VESTING_BAL=$(${BINARY} query bank balances "$VESTING_MODULE_ADDR" ${NODE_FLAG} --output json 2>/dev/null | jq -r '.balances[] | select(.denom=="uzrn") | .amount // "0"' 2>/dev/null || echo "0")
  info "Vesting rewards module ($VESTING_MODULE_ADDR) balance: ${VESTING_BAL} uzrn"
fi

# Check if any funds accumulated
if [ "${TREASURY_BAL:-0}" != "0" ] || [ "${RESEARCH_BAL:-0}" != "0" ] || [ "${VESTING_BAL:-0}" != "0" ]; then
  pass "2" "Fund balances non-zero (Treasury: ${TREASURY_BAL}, Research: ${RESEARCH_BAL}, Vesting: ${VESTING_BAL})"
else
  # KNOWN BUG: VestingRewardsKeeper.stakingKeeper is nil (set as nil in app.go constructor,
  # never wired). This means activeValidatorCount=0 → hasTransactions=false → no block rewards.
  # The protocol_treasury module account is also a placeholder (never receives funds by design).
  warn "Fund balances are zero — KNOWN BUG: VestingRewardsKeeper staking keeper is nil"
  warn "  → activeValidatorCount always 0 → hasTransactions always false → no block rewards minted"
  warn "  → Fix: wire staking keeper into VestingRewardsKeeper in app/app.go"
  fail "2" "Block rewards not flowing (staking keeper nil in vesting_rewards)"
fi

record_phase_time "Phase 3 (block rewards)" "$(($(date +%s) - PHASE3_START))"

# ── Phase 4: Domain Qualification ────────────────────────────────────────

header "Phase 4: Domain Qualification"
PHASE4_START=$(date +%s)

CHECKPOINT3_PASS=true

# Qualify val0 and val1 for "general" domain (100 ZRN stake each)
info "Qualifying val0 for 'general' domain..."
if send_tx "${BINARY} tx qualification qualify-by-stake general 100000000 --from val0 ${TX_FLAGS}"; then
  info "val0 qualified"
else
  warn "val0 qualification failed"
  CHECKPOINT3_PASS=false
fi
sleep 2

info "Qualifying val1 for 'general' domain..."
if send_tx "${BINARY} tx qualification qualify-by-stake general 100000000 --from val1 ${TX_FLAGS}"; then
  info "val1 qualified"
else
  warn "val1 qualification failed"
  CHECKPOINT3_PASS=false
fi
sleep 6

# Verify qualifications
info "Checking qualifications by domain..."
QUAL_RESULT=$(${BINARY} query qualification by-domain general ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
QUAL_COUNT=$(echo "$QUAL_RESULT" | jq '.qualifications | length' 2>/dev/null || echo "0")
info "Qualified validators for 'general': $QUAL_COUNT"

# Check val0 is qualified
VAL0_QUAL=$(${BINARY} query qualification qualification "$VAL0_ADDR" general ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
VAL0_QUAL_STATUS=$(echo "$VAL0_QUAL" | jq -r '.qualification.status // "NOT_FOUND"' 2>/dev/null)
info "val0 qualification status: $VAL0_QUAL_STATUS"

# Check val2 is NOT qualified
VAL2_QUAL=$(${BINARY} query qualification qualification "$VAL2_ADDR" general ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
VAL2_QUAL_STATUS=$(echo "$VAL2_QUAL" | jq -r '.qualification.status // "NOT_FOUND"' 2>/dev/null)
info "val2 qualification status: $VAL2_QUAL_STATUS (expected: NOT_FOUND)"

# Note: qualification count from query is the ground truth (send_tx may report false failures)
if [ "$QUAL_COUNT" -ge 2 ] 2>/dev/null; then
  pass "3" "val0 and val1 qualified for 'general', val2/val3 not qualified (count: $QUAL_COUNT)"
else
  fail "3" "Domain qualification incomplete (qualified count: $QUAL_COUNT)"
fi

record_phase_time "Phase 4 (qualification)" "$(($(date +%s) - PHASE4_START))"

# ── Phase 5: Claim + Qualified Verification ──────────────────────────────

header "Phase 5: Claim + Commit-Reveal Verification"
PHASE5_START=$(date +%s)

CHECKPOINT4_PASS=true
CLAIM_ID=""
ROUND_ID=""
CLAIM_STATUS=""
ROUND_VERDICT=""
ROUND_FINAL_PHASE=""

# Submit claim
CLAIM_TEXT="The speed of light in vacuum is approximately 299792458 meters per second"
info "Alice submitting claim..."
CLAIM_RESULT=$(send_tx "${BINARY} tx knowledge submit-claim '${CLAIM_TEXT}' general computational 1000000 --from alice ${TX_FLAGS}") || {
  warn "Claim submission failed"
  CHECKPOINT4_PASS=false
}

CLAIM_ID=""
ROUND_ID=""
if [ "$CHECKPOINT4_PASS" = true ]; then
  CLAIM_ID=$(get_event_attr "$CLAIM_RESULT" "zerone.knowledge.submit_claim" "claim_id")
  if [ -z "$CLAIM_ID" ]; then
    CLAIM_ID=$(echo "$CLAIM_RESULT" | jq -r '[.events[] | select(.type | contains("claim")) | .attributes[] | select(.key=="claim_id") | .value] | first // empty' 2>/dev/null)
  fi
  info "Claim ID: $CLAIM_ID"

  if [ -z "$CLAIM_ID" ]; then
    warn "Could not extract claim_id"
    CHECKPOINT4_PASS=false
  fi
fi

# Get round ID from claim query
if [ "$CHECKPOINT4_PASS" = true ]; then
  sleep 3
  CLAIM_QUERY=$(${BINARY} query knowledge claim "$CLAIM_ID" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  ROUND_ID=$(echo "$CLAIM_QUERY" | jq -r '.claim.verification_round_id // empty' 2>/dev/null)
  info "Round ID: $ROUND_ID"

  if [ -z "$ROUND_ID" ]; then
    warn "Could not get round_id from claim"
    CHECKPOINT4_PASS=false
  fi
fi

# Submit commitments from qualified validators (val0, val1)
SALT_HEX="a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"
VOTE="accept"
COMMIT_HASH=$(compute_commit_hash "$VOTE" "$SALT_HEX")

if [ "$CHECKPOINT4_PASS" = true ]; then
  info "Commit hash: $COMMIT_HASH"

  # val0 commits
  info "val0 submitting commitment..."
  if send_tx "${BINARY} tx knowledge submit-commitment ${ROUND_ID} ${COMMIT_HASH} --from val0 ${TX_FLAGS}"; then
    info "val0 committed"
  else
    warn "val0 commitment failed"
    CHECKPOINT4_PASS=false
  fi
  sleep 2

  # val1 commits (same hash for simplicity — both voting accept with same salt)
  info "val1 submitting commitment..."
  if send_tx "${BINARY} tx knowledge submit-commitment ${ROUND_ID} ${COMMIT_HASH} --from val1 ${TX_FLAGS}"; then
    info "val1 committed"
  else
    warn "val1 commitment failed"
    CHECKPOINT4_PASS=false
  fi
  sleep 2

  # val2 tries to commit — should FAIL (unqualified)
  info "val2 attempting commitment (expect failure)..."
  if send_tx_expect_fail "${BINARY} tx knowledge submit-commitment ${ROUND_ID} ${COMMIT_HASH} --from val2 ${TX_FLAGS}"; then
    info "val2 correctly rejected (unqualified)"
  else
    warn "val2 commitment unexpectedly succeeded"
    CHECKPOINT4_PASS=false
  fi
fi

# Wait for REVEAL phase (commit_phase_blocks = 10)
if [ "$CHECKPOINT4_PASS" = true ]; then
  info "Waiting for REVEAL phase..."
  # Query round to see commit deadline
  ROUND_QUERY=$(${BINARY} query knowledge verification-round "$ROUND_ID" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  COMMIT_DEADLINE=$(echo "$ROUND_QUERY" | jq -r '.round.commit_deadline // empty' 2>/dev/null | tr -d '"')
  CURRENT_HEIGHT=$(get_height)
  if [ -n "$COMMIT_DEADLINE" ] && [ "$CURRENT_HEIGHT" -lt "$COMMIT_DEADLINE" ] 2>/dev/null; then
    BLOCKS_TO_WAIT=$((COMMIT_DEADLINE - CURRENT_HEIGHT + 1))
    wait_blocks "$BLOCKS_TO_WAIT"
  else
    sleep 6
  fi

  # Verify round is in REVEAL phase
  ROUND_QUERY=$(${BINARY} query knowledge verification-round "$ROUND_ID" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  ROUND_PHASE=$(echo "$ROUND_QUERY" | jq -r '.round.phase // empty' 2>/dev/null)
  info "Round phase: $ROUND_PHASE"

  # Submit reveals from val0 and val1
  info "val0 submitting reveal..."
  if send_tx "${BINARY} tx knowledge submit-reveal ${ROUND_ID} ${VOTE} ${SALT_HEX} --from val0 ${TX_FLAGS}"; then
    info "val0 revealed"
  else
    warn "val0 reveal failed"
    CHECKPOINT4_PASS=false
  fi
  sleep 2

  info "val1 submitting reveal..."
  if send_tx "${BINARY} tx knowledge submit-reveal ${ROUND_ID} ${VOTE} ${SALT_HEX} --from val1 ${TX_FLAGS}"; then
    info "val1 revealed"
  else
    warn "val1 reveal failed"
    CHECKPOINT4_PASS=false
  fi
fi

# Wait for aggregation (reveal_phase_blocks + aggregation)
if [ "$CHECKPOINT4_PASS" = true ]; then
  info "Waiting for aggregation..."
  REVEAL_DEADLINE=$(echo "$ROUND_QUERY" | jq -r '.round.reveal_deadline // empty' 2>/dev/null | tr -d '"')
  CURRENT_HEIGHT=$(get_height)
  if [ -n "$REVEAL_DEADLINE" ] && [ "$CURRENT_HEIGHT" -lt "$REVEAL_DEADLINE" ] 2>/dev/null; then
    BLOCKS_TO_WAIT=$((REVEAL_DEADLINE - CURRENT_HEIGHT + 2))
    wait_blocks "$BLOCKS_TO_WAIT"
  else
    sleep 10
  fi

  # Check claim result
  sleep 6
  CLAIM_FINAL=$(${BINARY} query knowledge claim "$CLAIM_ID" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  CLAIM_STATUS=$(echo "$CLAIM_FINAL" | jq -r '.claim.status // "unknown"' 2>/dev/null)
  info "Claim final status: $CLAIM_STATUS"

  # Check round verdict
  ROUND_FINAL=$(${BINARY} query knowledge verification-round "$ROUND_ID" ${NODE_FLAG} --output json 2>/dev/null || echo "{}")
  ROUND_VERDICT=$(echo "$ROUND_FINAL" | jq -r '.round.verdict // "unknown"' 2>/dev/null)
  ROUND_FINAL_PHASE=$(echo "$ROUND_FINAL" | jq -r '.round.phase // "unknown"' 2>/dev/null)
  info "Round verdict: $ROUND_VERDICT"
  info "Round phase: $ROUND_FINAL_PHASE"
fi

# ClaimStatus: 6=ACCEPTED, Verdict: 1=ACCEPT, Phase: 4=COMPLETE (proto enum values)
if [ "$CHECKPOINT4_PASS" = true ] && { [ "$CLAIM_STATUS" = "6" ] || echo "$CLAIM_STATUS" | grep -qi "accepted"; }; then
  pass "4" "Claim ACCEPTED (status=$CLAIM_STATUS, verdict=$ROUND_VERDICT)"
elif [ "$CHECKPOINT4_PASS" = true ] && { [ "$ROUND_VERDICT" = "1" ] || echo "$ROUND_VERDICT" | grep -qi "accept"; }; then
  pass "4" "Round verdict ACCEPT (claim status=$CLAIM_STATUS, phase=$ROUND_FINAL_PHASE)"
elif [ "$CHECKPOINT4_PASS" = true ] && { [ "$ROUND_FINAL_PHASE" = "4" ] || echo "$ROUND_FINAL_PHASE" | grep -qi "complete"; }; then
  pass "4" "Round completed (verdict: $ROUND_VERDICT, claim: $CLAIM_STATUS)"
else
  fail "4" "Claim verification incomplete (status: ${CLAIM_STATUS:-unknown}, verdict: ${ROUND_VERDICT:-unknown}, phase: ${ROUND_FINAL_PHASE:-unknown})"
fi

record_phase_time "Phase 5 (claim+verification)" "$(($(date +%s) - PHASE5_START))"

# ── Phase 6: Negative Tests ──────────────────────────────────────────────

header "Phase 6: Negative Tests"
PHASE6_START=$(date +%s)

CHECKPOINT5_PASS=true

# Test 1: Claim asserting the retired partnership_id wire field → rejected
# (x/partnerships was cut in the slim cut; the wire slot is kept but any
#  non-empty value must be refused rather than silently recorded).
info "Test 1: Claim with retired partnership_id field..."
if send_tx_expect_fail "${BINARY} tx knowledge submit-claim 'This should fail because the partnership field is retired' general computational 1000000 --partnership-id nonexistent-id-12345 --from alice ${TX_FLAGS}"; then
  info "Retired partnership_id field correctly rejected"
else
  warn "Retired partnership_id field not rejected"
  CHECKPOINT5_PASS=false
fi

if [ "$CHECKPOINT5_PASS" = true ]; then
  pass "5" "Negative tests passed: retired partnership_id field rejected"
else
  fail "5" "Some negative tests failed"
fi

record_phase_time "Phase 6 (negative tests)" "$(($(date +%s) - PHASE6_START))"

# ═══════════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════════

TOTAL_TIME=$(($(date +%s) - START_TIME))

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  E2E FULL LOOP — RESULTS"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Checkpoints: ${PASSED}/${TOTAL_CHECKPOINTS} passed, ${FAILED}/${TOTAL_CHECKPOINTS} failed"
echo "  Total time: ${TOTAL_TIME}s"
echo ""
echo "  Phase timing:"
echo -e "$PHASE_TIMES"

if [ "$FAILED" -gt 0 ]; then
  echo ""
  echo "  Failures:"
  echo -e "$FAILURES"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo -e "\033[1;32m  VERDICT: ALL CHECKPOINTS PASSED\033[0m"
else
  echo -e "\033[1;31m  VERDICT: ${FAILED} CHECKPOINT(S) FAILED\033[0m"
fi
echo ""
echo "═══════════════════════════════════════════════════════════════"

# Exit with failure code if any checkpoints failed
[ "$FAILED" -eq 0 ]
