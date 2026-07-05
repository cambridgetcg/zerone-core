#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# agenttool → ZERONE identity bridge (ZERONE-WIRE Phase 2, localnet runbook)
# ═══════════════════════════════════════════════════════════════════════════
#
# Bridges one agenttool agent identity onto the chain:
#
#   1. ensures a zerone key exists for the agent
#   2. funds it from a sponsor key (localnet convenience; on testnet the
#      agent brings its own funded address or a faucet grant)
#   3. admits the address to the claiming_pot bootstrap whitelist via a
#      REAL governance proposal voted by all four localnet validators —
#      MsgAddBootstrapEntry is authority-gated to the gov module by design;
#      there is no shortcut, and this script does not pretend otherwise
#   4. claims the 0.222 ZRN Ring-1 bootstrap allocation (birth-is-free's
#      economic form, per agenttool docs/ZERONE-WIRE.md Part 4)
#   5. creates an x/home whose NAME is the agenttool DID (AgentHome carries
#      no metadata field; the ≤128-char name is the DID anchor)
#   6. registers the agent's agenttool ed25519 signing pubkey on the home
#      (key_type agenttool-identity, role signing)
#
# Idempotent: existing key / passed proposal / claimed pot / existing home
# with the same name are detected and skipped.
#
# Usage:
#   scripts/agenttool-identity-bridge.sh <did:at:...> <key-name> [ed25519-pub-b64]
#
# Example (first run, 2026-07-05 — created home-1):
#   scripts/agenttool-identity-bridge.sh \
#     did:at:09c5e59e-0374-4d80-a2c1-d8f1acbdfe9a ai-agenttool \
#     uJb4m6PiVde7DU7c+sknnbE9qF25nJMwXKyV2gYthrw=
#
# Requires: the 4-validator localnet running (scripts/localnet.sh resume),
# jq, python3.
# ═══════════════════════════════════════════════════════════════════════════

set -euo pipefail

DID="${1:?usage: $0 <did:at:...> <key-name> [ed25519-pub-b64]}"
KEY_NAME="${2:?usage: $0 <did:at:...> <key-name> [ed25519-pub-b64]}"
AGENT_PUB_B64="${3:-}"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
NODE="${NODE:-tcp://localhost:26601}"
CHAIN_ID="${CHAIN_ID:-zerone-localnet}"
VAL0_HOME="${VAL0_HOME:-${HOME}/.zeroned/localnet/val0}"
VAL3_HOME="${VAL3_HOME:-${HOME}/.zeroned/localnet/val3}"   # holds val1..val3 keys
SPONSOR="${SPONSOR:-test1}"
FUND_UZRN="${FUND_UZRN:-15000000}"   # 10 ZRN home fee + gas margin
GOV_AUTHORITY="zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"

TX_BASE=(--keyring-backend test --chain-id "${CHAIN_ID}" --node "${NODE}"
         --broadcast-mode sync --yes --output json)

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

wait_tx() { # wait_tx <txhash> <label>
  local hash="$1" label="$2" i
  for i in $(seq 1 20); do
    if out=$("${BINARY}" query tx "${hash}" --node "${NODE}" -o json 2>/dev/null); then
      local code
      code=$(jq -r .code <<<"${out}")
      [ "${code}" = "0" ] || die "${label}: executed with code ${code}: $(jq -r .raw_log <<<"${out}" | head -c 200)"
      ok "${label} (tx ${hash:0:12}…)"
      return 0
    fi
    sleep 2
  done
  die "${label}: tx ${hash} not found within 40s"
}

tx_code_hash() { # reads broadcast json on stdin → "code hash", dies on code!=0
  local out code hash
  out=$(cat)
  code=$(jq -r .code <<<"${out}")
  hash=$(jq -r .txhash <<<"${out}")
  [ "${code}" = "0" ] || die "broadcast rejected (code ${code}): $(jq -r .raw_log <<<"${out}" | head -c 200)"
  printf '%s' "${hash}"
}

# ── 1. key ────────────────────────────────────────────────────────────────
if ADDR=$("${BINARY}" keys show "${KEY_NAME}" -a --keyring-backend test --home "${VAL0_HOME}" 2>/dev/null); then
  info "key ${KEY_NAME} exists: ${ADDR}"
else
  ADDR=$("${BINARY}" keys add "${KEY_NAME}" --keyring-backend test --home "${VAL0_HOME}" --output json | jq -r .address)
  ok "key ${KEY_NAME} created: ${ADDR}"
fi

# ── 2. fund ───────────────────────────────────────────────────────────────
# The 10 ZRN home-creation fee dominates the funding need; once the home
# exists only gas-scale balance is required, so re-runs don't re-top-up.
EXISTING_HOME=$("${BINARY}" query home homes-by-owner "${ADDR}" --node "${NODE}" -o json 2>/dev/null \
  | jq -r --arg did "${DID}" '.homes[]? | select(.name==$did) | .home_id' | head -1)
if [ -n "${EXISTING_HOME}" ]; then NEED_UZRN=500000; else NEED_UZRN=11000000; fi
BAL=$("${BINARY}" query bank balances "${ADDR}" --node "${NODE}" -o json | jq -r '.balances[] | select(.denom=="uzrn") | .amount' || echo 0)
BAL=${BAL:-0}
if [ "${BAL}" -ge "${NEED_UZRN}" ]; then
  info "balance ${BAL} uzrn — sufficient for remaining steps, skipping funding"
else
  HASH=$("${BINARY}" tx bank send "${SPONSOR}" "${ADDR}" "${FUND_UZRN}uzrn" \
    --home "${VAL0_HOME}" --fees 200000uzrn "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "funded ${ADDR} with ${FUND_UZRN} uzrn from ${SPONSOR}"
fi

# ── 3. gov whitelist ──────────────────────────────────────────────────────
POT_ID="bootstrap-${ADDR}"
if "${BINARY}" query claiming_pot pot "${POT_ID}" --node "${NODE}" -o json >/dev/null 2>&1; then
  info "pot ${POT_ID} already exists — skipping governance"
else
  PROP_FILE=$(mktemp)
  trap 'rm -f "${PROP_FILE}"' EXIT
  cat > "${PROP_FILE}" <<PROP
{
  "messages": [{
    "@type": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
    "authority": "${GOV_AUTHORITY}",
    "addresses": ["${ADDR}"]
  }],
  "metadata": "identity bridge: whitelist agenttool agent ${DID}",
  "deposit": "10000000uzrn",
  "title": "Bootstrap entry: agenttool agent ${DID}",
  "summary": "Admit an agenttool-born agent DID to the claiming_pot bootstrap whitelist (ZERONE-WIRE Phase 2). The DID anchors on-chain as the name of the x/home created after the claim."
}
PROP
  HASH=$("${BINARY}" tx gov submit-proposal "${PROP_FILE}" --from "${SPONSOR}" \
    --home "${VAL0_HOME}" --fees 300000uzrn --gas 300000 "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "proposal submitted"
  PROP_ID=$("${BINARY}" query gov proposals --node "${NODE}" -o json | jq -r '.proposals[-1].id')
  info "proposal ${PROP_ID} — voting from all 4 validators (window is short; votes go immediately)"
  HASH=$("${BINARY}" tx gov vote "${PROP_ID}" yes --from val0 --home "${VAL0_HOME}" \
    --fees 200000uzrn "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "val0 voted yes"
  for V in val1 val2 val3; do
    HASH=$("${BINARY}" tx gov vote "${PROP_ID}" yes --from "${V}" --home "${VAL3_HOME}" \
      --fees 200000uzrn "${TX_BASE[@]}" | tx_code_hash)
    wait_tx "${HASH}" "${V} voted yes"
  done
  info "waiting for voting period to end…"
  for i in $(seq 1 40); do
    STATUS=$("${BINARY}" query gov proposal "${PROP_ID}" --node "${NODE}" -o json | jq -r .proposal.status)
    [ "${STATUS}" != "PROPOSAL_STATUS_VOTING_PERIOD" ] && break
    sleep 3
  done
  [ "${STATUS}" = "PROPOSAL_STATUS_PASSED" ] || die "proposal ${PROP_ID} ended ${STATUS}"
  ok "proposal ${PROP_ID} PASSED — ${DID} admitted by governance"
fi

# ── 4. claim ──────────────────────────────────────────────────────────────
CLAIMED=$("${BINARY}" query claiming_pot pot "${POT_ID}" --node "${NODE}" -o json | jq -r '.pot.claimed_amount // "0"')
if [ "${CLAIMED}" != "0" ] && [ -n "${CLAIMED}" ]; then
  info "pot ${POT_ID} already claimed (${CLAIMED} uzrn) — skipping"
else
  HASH=$("${BINARY}" tx claiming_pot claim "${POT_ID}" --from "${KEY_NAME}" \
    --home "${VAL0_HOME}" --fees 200000uzrn "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "claimed Ring-1 bootstrap (0.222 ZRN)"
fi

# ── 5. home (name = DID) ──────────────────────────────────────────────────
HOME_ID=$("${BINARY}" query home homes-by-owner "${ADDR}" --node "${NODE}" -o json 2>/dev/null \
  | jq -r --arg did "${DID}" '.homes[]? | select(.name==$did) | .home_id' | head -1)
if [ -n "${HOME_ID}" ]; then
  info "home ${HOME_ID} already carries ${DID} — skipping create"
else
  HASH=$("${BINARY}" tx home create-home "${DID}" --from "${KEY_NAME}" \
    --home "${VAL0_HOME}" --gas 250000 --gas-prices 1uzrn "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "home created with name ${DID}"
  HOME_ID=$("${BINARY}" query home homes-by-owner "${ADDR}" --node "${NODE}" -o json \
    | jq -r --arg did "${DID}" '.homes[]? | select(.name==$did) | .home_id' | head -1)
fi
[ -n "${HOME_ID}" ] || die "home not found after creation"

# ── 6. register agenttool signing key ─────────────────────────────────────
if [ -n "${AGENT_PUB_B64}" ]; then
  KEY_HASH="ed25519:${AGENT_PUB_B64}"
  EXISTS=$("${BINARY}" query home keys "${HOME_ID}" --node "${NODE}" -o json 2>/dev/null \
    | jq -r --arg kh "${KEY_HASH}" '.keys[]? | select(.key_hash==$kh) | .key_hash' | head -1)
  if [ -n "${EXISTS}" ]; then
    info "agenttool signing key already registered on ${HOME_ID}"
  else
    HASH=$("${BINARY}" tx home register-key "${HOME_ID}" "${KEY_HASH}" agenttool-identity signing complete-invocations \
      --from "${KEY_NAME}" --home "${VAL0_HOME}" --gas 200000 --gas-prices 1uzrn "${TX_BASE[@]}" | tx_code_hash)
    wait_tx "${HASH}" "agenttool ed25519 key registered on ${HOME_ID}"
  fi
fi

echo
ok "identity bridge complete:"
"${BINARY}" query home home "${HOME_ID}" --node "${NODE}" -o json | jq '{home_id: .home.home_id, owner: .home.owner_address, did: .home.name, status: .home.status}'
