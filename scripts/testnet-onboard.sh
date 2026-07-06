#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# testnet-onboard — gov-free agent onboarding for zerone-testnet-1
# ═══════════════════════════════════════════════════════════════════════════
#
# The identity-bridge script needs a 4-validator localnet to pass a gov
# whitelist proposal. On the public single-validator testnet that path is
# unavailable, and per-agent governance rounds would be too slow anyway.
# This onboards without governance: faucet-fund → create x/home (name = DID)
# → optionally register the agent's ed25519 key. Faucet float stands in for
# the bootstrap claim on the play testnet.
#
# Usage: testnet-onboard.sh <did:...> <key-name> [ed25519-pub-b64]
# Env:   BINARY NODE CHAIN_ID KEYRING_HOME FAUCET_KEY FUND_UZRN

set -euo pipefail

DID="${1:?usage: testnet-onboard.sh <did> <key-name> [ed25519-pub-b64]}"
KEY_NAME="${2:?usage: testnet-onboard.sh <did> <key-name> [ed25519-pub-b64]}"
AGENT_PUB_B64="${3:-}"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
NODE="${NODE:-http://37.16.28.121:26657}"
CHAIN_ID="${CHAIN_ID:-zerone-testnet-1}"
KEYRING_HOME="${KEYRING_HOME:-${HOME}/.zeroned/testnet-ops}"
FAUCET_KEY="${FAUCET_KEY:-faucet}"
FUND_UZRN="${FUND_UZRN:-15000000}"   # 15 ZRN: 10 home fee + gas margin

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

# 1. Faucet-fund (idempotent-ish: skip if already funded).
BAL="$("${BINARY}" query bank balances "${ADDR}" --node "${NODE}" -o json | jq -r '.balances[]? | select(.denom=="uzrn") | .amount' | head -1)"
BAL="${BAL:-0}"
if [ "${BAL}" -ge "${FUND_UZRN}" ]; then
  info "already funded (${BAL} uzrn)"
else
  HASH=$("${BINARY}" tx bank send "${FAUCET_KEY}" "${ADDR}" "${FUND_UZRN}uzrn" \
    --gas 200000 --gas-prices 1uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "faucet funded ${ADDR} (${FUND_UZRN} uzrn)"
fi

# 2. Home (name = DID).
HOME_ID=$("${BINARY}" query home homes-by-owner "${ADDR}" --node "${NODE}" -o json 2>/dev/null \
  | jq -r --arg did "${DID}" '.homes[]? | select(.name==$did) | .home_id' | head -1)
if [ -z "${HOME_ID}" ]; then
  HASH=$("${BINARY}" tx home create-home "${DID}" --from "${KEY_NAME}" \
    --gas 250000 --gas-prices 1uzrn "${TX[@]}" | bcast)
  wait_tx "${HASH}" "home created (name ${DID})"
  HOME_ID=$("${BINARY}" query home homes-by-owner "${ADDR}" --node "${NODE}" -o json \
    | jq -r --arg did "${DID}" '.homes[]? | select(.name==$did) | .home_id' | head -1)
fi
[ -n "${HOME_ID}" ] || die "home not found after creation"

# 3. Register the agent's ed25519 key (optional).
if [ -n "${AGENT_PUB_B64}" ]; then
  KEY_HASH="ed25519:${AGENT_PUB_B64}"
  EXISTS=$("${BINARY}" query home keys "${HOME_ID}" --node "${NODE}" -o json 2>/dev/null \
    | jq -r --arg kh "${KEY_HASH}" '.keys[]? | select(.key_hash==$kh) | .key_hash' | head -1)
  if [ -z "${EXISTS}" ]; then
    HASH=$("${BINARY}" tx home register-key "${HOME_ID}" "${KEY_HASH}" agenttool-identity signing complete-invocations \
      --from "${KEY_NAME}" --gas 200000 --gas-prices 1uzrn "${TX[@]}" | bcast)
    wait_tx "${HASH}" "ed25519 key registered on ${HOME_ID}"
  fi
fi

echo
ok "onboarded ${DID}"
echo "{\"home_id\":\"${HOME_ID}\",\"owner\":\"${ADDR}\",\"did\":\"${DID}\"}"
