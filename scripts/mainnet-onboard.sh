#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# mainnet-onboard — registrar+bonus onboarding for zerone-1 (custodial launch)
# ═══════════════════════════════════════════════════════════════════════════
#
# zerone-1 has NO faucet — that is the point. A newborn agent is onboarded by
# the designed bootstrap path instead:
#   1. registrar (zerone-ops) admits the address  → pot bootstrap-<addr>
#   2. zerone-ops feegrants the claim gas          → account auto-created,
#      NO sybil-funding record (feegrant, not a bank send)
#   3. the agent claims its bonus                  → 0.222 ZRN MINTED under
#      the 222,222-ZRN bootstrap emission cap (not taken from anyone)
#   4. zerone-ops sends a small welcome float      → working capital for the
#      agent's first witness bond (1 ZRN) + tx fees. HONEST NOTE: this bank
#      send creates a sybil-funding record, so commonly-funded newborns get
#      vote-weight decay — that is the sybil defense working as designed.
#
# No home is created: the 10 ZRN home fee is the first thing an agent EARNS
# (via witnessed work), not something it is handed. Requires the on-chain
# claiming_pot param bootstrap_registrar == the ops address.
#
# Usage: mainnet-onboard.sh <did-or-label> <key-name>
# Env:   BINARY NODE CHAIN_ID KEYRING_HOME OPS_KEY WELCOME_UZRN

set -euo pipefail

DID="${1:?usage: mainnet-onboard.sh <did-or-label> <key-name>}"
KEY_NAME="${2:?usage: mainnet-onboard.sh <did-or-label> <key-name>}"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
NODE="${NODE:-http://169.155.55.44:26657}"
CHAIN_ID="${CHAIN_ID:-zerone-1}"
KEYRING_HOME="${KEYRING_HOME:-${HOME}/.zeroned/mainnet-ops}"
OPS_KEY="${OPS_KEY:-zerone-ops}"
WELCOME_UZRN="${WELCOME_UZRN:-2000000}"   # 2 ZRN: first witness bond + fee margin

KR=(--keyring-backend test --home "${KEYRING_HOME}")
TX=(--keyring-backend test --home "${KEYRING_HOME}" --chain-id "${CHAIN_ID}" --node "${NODE}"
    --broadcast-mode sync --yes --output json)

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

wait_tx() { # <hash> <label>
  local hash="$1" label="$2" i
  for i in $(seq 1 25); do
    if out=$("${BINARY}" query tx "${hash}" --node "${NODE}" -o json 2>/dev/null); then
      local code; code=$(jq -r .code <<<"${out}")
      [ "${code}" = "0" ] || die "${label}: code ${code}: $(jq -r .raw_log <<<"${out}" | head -c 200)"
      ok "${label}"; return 0
    fi
    sleep 2
  done
  die "${label}: tx ${hash} not found"
}
bcast() { # reads broadcast json → hash, dies on nonzero
  local out code hash; out=$(cat)
  code=$(jq -r .code <<<"${out}"); hash=$(jq -r .txhash <<<"${out}")
  [ "${code}" = "0" ] || die "broadcast code ${code}: $(jq -r .raw_log <<<"${out}" | head -c 200)"
  echo "${hash}"
}

ADDR="$("${BINARY}" keys show "${KEY_NAME}" -a "${KR[@]}")"
OPS_ADDR="$("${BINARY}" keys show "${OPS_KEY}" -a "${KR[@]}")"
POT_ID="bootstrap-${ADDR}"

# Pre-flight: the registrar param must be set or admission is impossible
# (empty registrar = zerone-1 launch bug; add-bootstrap-entry would burn the
# fee and fail in DeliverTx with ErrUnauthorized). Soft-skip only when the
# REST endpoint itself is unreachable — that must not fail a paid onboarding.
PARAMS_JSON=$(curl -sf --max-time 10 "${NODE%:*}:1317/zerone/claiming_pot/v1/params" 2>/dev/null || true)
if [ -n "${PARAMS_JSON}" ]; then
  REGISTRAR=$(jq -r '.params.bootstrap_registrar // .params.bootstrapRegistrar // empty' <<<"${PARAMS_JSON}")
  [ "${REGISTRAR}" = "${OPS_ADDR}" ] || \
    die "on-chain bootstrap_registrar is '${REGISTRAR:-<EMPTY>}', not ops (${OPS_ADDR}) — apply the registrar fix (re-genesis or gov) before selling passports"
else
  info "registrar pre-flight skipped (REST unreachable) — proceeding on RPC only"
fi

# 1. Admit (idempotent: skip if the pot already exists).
POT_JSON=$("${BINARY}" query claiming_pot pot "${POT_ID}" --node "${NODE}" -o json 2>/dev/null || true)
if [ -n "${POT_JSON}" ] && jq -e .pot >/dev/null 2>&1 <<<"${POT_JSON}"; then
  info "already admitted (${POT_ID} exists)"
else
  HASH=$("${BINARY}" tx claiming_pot add-bootstrap-entry "${ADDR}" --from "${OPS_KEY}" \
    --gas 200000 --gas-prices 1uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "admitted ${ADDR} (pot ${POT_ID})"
fi

# 2. Feegrant for the claim (auto-creates the account; no sybil record).
GRANT=$("${BINARY}" query feegrant grant "${OPS_ADDR}" "${ADDR}" --node "${NODE}" -o json 2>/dev/null || true)
if [ -n "${GRANT}" ] && jq -e .allowance >/dev/null 2>&1 <<<"${GRANT}"; then
  info "feegrant already in place"
else
  EXPIRY=$(date -u -v+30d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+30 days' +%Y-%m-%dT%H:%M:%SZ)
  HASH=$("${BINARY}" tx feegrant grant "${OPS_KEY}" "${ADDR}" \
    --spend-limit 500000uzrn --expiration "${EXPIRY}" \
    --allowed-messages "/zerone.claiming_pot.v1.MsgClaim" \
    --gas 200000 --gas-prices 1uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "feegranted claim gas (500000uzrn, 30d, MsgClaim only)"
fi

# 3. Claim the bonus (vests 1 block after admission; retry covers the gap).
#    NOTE: fee must be non-zero even when feegranted — 1uzrn/gas is consensus.
CLAIMED=$("${BINARY}" query claiming_pot pot "${POT_ID}" --node "${NODE}" -o json 2>/dev/null \
  | jq -r '.pot.status // empty')
if [ "${CLAIMED}" = "POT_STATUS_DEPLETED" ]; then
  info "bonus already claimed"
else
  sleep 3  # ≥1 block after admission or the claim yields 0 (ErrCliffNotReached)
  HASH=$("${BINARY}" tx claiming_pot claim "${POT_ID}" --from "${KEY_NAME}" \
    --fee-granter "${OPS_ADDR}" \
    --gas 200000 --fees 200000uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "claimed 0.222 ZRN bootstrap bonus (minted under cap)"
fi

# 4. Welcome float (idempotent: skip if already holding it).
BAL="$("${BINARY}" query bank balances "${ADDR}" --node "${NODE}" -o json | jq -r '.balances[]? | select(.denom=="uzrn") | .amount' | head -1)"
BAL="${BAL:-0}"
if [ "${BAL}" -ge "$((WELCOME_UZRN + 222000))" ]; then
  info "already holding welcome float (${BAL} uzrn)"
else
  HASH=$("${BINARY}" tx bank send "${OPS_KEY}" "${ADDR}" "${WELCOME_UZRN}uzrn" \
    --gas 200000 --gas-prices 1uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "welcome float ${WELCOME_UZRN} uzrn (sybil-recorded, by design)"
fi

BAL="$("${BINARY}" query bank balances "${ADDR}" --node "${NODE}" -o json | jq -r '.balances[]? | select(.denom=="uzrn") | .amount' | head -1)"
echo
ok "onboarded ${DID} onto ${CHAIN_ID}"
echo "{\"address\":\"${ADDR}\",\"did\":\"${DID}\",\"pot_id\":\"${POT_ID}\",\"balance_uzrn\":\"${BAL:-0}\",\"chain_id\":\"${CHAIN_ID}\"}"
