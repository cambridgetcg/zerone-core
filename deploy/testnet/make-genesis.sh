#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# zerone-testnet-1 — canon genesis (single genesis validator, zero pre-mine)
# ═══════════════════════════════════════════════════════════════════════════
#
# Honors the 222,222,222 / zero-pre-mine canon: NO foundation or research
# genesis allocation. Only what a live testnet physically needs to start —
# a genesis validator's self-delegation stake and a faucet for testers —
# both drawn as play tokens on a disposable chain.
#
# Produces, under $OUT:
#   genesis.json          — the public genesis (goes in the join-kit + image)
#   node_key.json         — P2P identity (determines the seed node id)
#   priv_validator_key.json — the validator signing key (fly SECRET, not public)
#   faucet.mnemonic       — the faucet key mnemonic (operator-held)
#   node-id.txt, faucet-address.txt, seed-hint.txt — join-kit inputs
#
# Usage: deploy/testnet/make-genesis.sh [output-dir]

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
OUT="${1:-${PROJECT_ROOT}/deploy/testnet/artifacts}"
CHAIN_ID="zerone-testnet-1"
DENOM="uzrn"
MONIKER="zerone-testnet-seed-0"

# Play-token amounts (disposable chain).
VAL_BALANCE="200000000000${DENOM}"   # 200,000 ZRN to the validator account
VAL_STAKE="100000000000${DENOM}"     # 100,000 ZRN self-delegated
FAUCET_BALANCE="1000000000000${DENOM}" # 1,000,000 ZRN faucet float

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }

CEREMONY="$(mktemp -d)"
trap 'rm -rf "${CEREMONY}"' EXIT
KR=(--keyring-backend test --home "${CEREMONY}")

info "init ${CHAIN_ID}"
"${BINARY}" init "${MONIKER}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" --home "${CEREMONY}" >/dev/null 2>&1

info "keys: validator + faucet (mnemonics exported — validator key = gov control)"
mkdir -p "${OUT}"
"${BINARY}" keys add validator "${KR[@]}" --output json 2>/dev/null | jq -r .mnemonic > "${OUT}.validator.tmp" || true
"${BINARY}" keys add faucet "${KR[@]}" --output json 2>/dev/null | jq -r .mnemonic > "${OUT}.faucet.tmp" || true
VAL_ADDR="$("${BINARY}" keys show validator -a "${KR[@]}")"
FAUCET_ADDR="$("${BINARY}" keys show faucet -a "${KR[@]}")"

# Reuse an existing node key so the seed node-id stays stable across
# regenerations (join-kit + passport + listing reference it).
if [ -f "${OUT}/node_key.json" ]; then
  cp "${OUT}/node_key.json" "${CEREMONY}/config/node_key.json"
  info "reused existing node_key (stable node-id)"
fi

info "genesis accounts (zero pre-mine: only validator stake + faucet float)"
"${BINARY}" add-genesis-account "${VAL_ADDR}" "${VAL_BALANCE}" --home "${CEREMONY}" >/dev/null 2>&1
"${BINARY}" add-genesis-account "${FAUCET_ADDR}" "${FAUCET_BALANCE}" --home "${CEREMONY}" >/dev/null 2>&1

info "gentx (self-delegate ${VAL_STAKE})"
"${BINARY}" genesis gentx validator "${VAL_STAKE}" --chain-id "${CHAIN_ID}" "${KR[@]}" >/dev/null 2>&1
"${BINARY}" genesis collect-gentxs --home "${CEREMONY}" >/dev/null 2>&1

info "patch testnet params (short windows, real fee floor, tuned witness gate)"
GEN="${CEREMONY}/config/genesis.json"
jq '
  .app_state.gov.params.voting_period = "120s" |
  .app_state.gov.params.expedited_voting_period = "60s" |
  .app_state.gov.params.max_deposit_period = "300s" |
  (.app_state.substrate_bridge.params.witness_reward_challenge_window_blocks) = "200"
' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

info "validate"
"${BINARY}" genesis validate --home "${CEREMONY}" >/dev/null 2>&1 || "${BINARY}" validate-genesis "${GEN}" >/dev/null 2>&1

# ── Export artifacts ─────────────────────────────────────────────────────
mkdir -p "${OUT}"
cp "${GEN}" "${OUT}/genesis.json"
cp "${CEREMONY}/config/node_key.json" "${OUT}/node_key.json"
cp "${CEREMONY}/config/priv_validator_key.json" "${OUT}/priv_validator_key.json"
[ -f "${OUT}.faucet.tmp" ] && mv "${OUT}.faucet.tmp" "${OUT}/faucet.mnemonic"
[ -f "${OUT}.validator.tmp" ] && mv "${OUT}.validator.tmp" "${OUT}/validator.mnemonic"

NODE_ID="$("${BINARY}" tendermint show-node-id --home "${CEREMONY}")"
echo "${NODE_ID}" > "${OUT}/node-id.txt"
echo "${FAUCET_ADDR}" > "${OUT}/faucet-address.txt"
echo "${VAL_ADDR}" > "${OUT}/validator-address.txt"
echo "${NODE_ID}@<PUBLIC_HOST>:26656" > "${OUT}/seed-hint.txt"

ok "artifacts in ${OUT}"
echo "     chain-id:  ${CHAIN_ID}"
echo "     node-id:   ${NODE_ID}"
echo "     validator: ${VAL_ADDR}"
echo "     faucet:    ${FAUCET_ADDR} (1,000,000 ZRN float)"
echo "     supply at genesis: 1,200,000 ZRN (100K bonded + 100K val + 1M faucet); 221,022,222 ZRN of cap headroom for emission"
