#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# zerone-testnet-1 — REGENERATE genesis on the new binary, REUSING identities
# ═══════════════════════════════════════════════════════════════════════════
#
# Unlike make-genesis.sh (which mints fresh validator + faucet keys), this
# rebuilds ONLY the genesis app_state with the current binary while REUSING
# the existing node key, validator signing key, and validator + faucet
# mnemonics from deploy/testnet/artifacts/. Result: a fresh chain (height 0,
# new modules/params, the fixed PoT-reward code) that keeps the SAME node-id,
# validator, and faucet address — so the public endpoints, seed string, relay,
# and passport all keep working unchanged after a volume reset.
#
# Usage: deploy/testnet/regen-genesis.sh
# Requires: build/zeroned freshly built from HEAD; jq; the artifacts/ identity files.

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
ART="${PROJECT_ROOT}/deploy/testnet/artifacts"
CHAIN_ID="zerone-testnet-1"
DENOM="uzrn"
MONIKER="zerone-testnet-seed-0"

VAL_BALANCE="200000000000${DENOM}"     # 200,000 ZRN
VAL_STAKE="100000000000${DENOM}"       # 100,000 ZRN self-delegated
FAUCET_BALANCE="1000000000000${DENOM}" # 1,000,000 ZRN faucet float

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

for f in validator.mnemonic faucet.mnemonic node_key.json priv_validator_key.json; do
  [ -f "${ART}/${f}" ] || die "missing identity artifact: ${ART}/${f}"
done
[ -x "${BINARY}" ] || die "binary not found: ${BINARY} (run 'make build')"

CEREMONY="$(mktemp -d)"
trap 'rm -rf "${CEREMONY}"' EXIT
KR=(--keyring-backend test --home "${CEREMONY}")

info "init ${CHAIN_ID} (reusing identities)"
"${BINARY}" init "${MONIKER}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" --home "${CEREMONY}" >/dev/null 2>&1

# Reuse the P2P node key (stable node-id) and the validator consensus key
# (so the baked priv_validator_key still signs) BEFORE gentx reads them.
cp "${ART}/node_key.json"          "${CEREMONY}/config/node_key.json"
cp "${ART}/priv_validator_key.json" "${CEREMONY}/config/priv_validator_key.json"

info "recover validator + faucet keys from artifacts"
"${BINARY}" keys add validator --recover "${KR[@]}" >/dev/null 2>&1 < "${ART}/validator.mnemonic"
"${BINARY}" keys add faucet    --recover "${KR[@]}" >/dev/null 2>&1 < "${ART}/faucet.mnemonic"
VAL_ADDR="$("${BINARY}" keys show validator -a "${KR[@]}")"
FAUCET_ADDR="$("${BINARY}" keys show faucet -a "${KR[@]}")"
[ "${VAL_ADDR}" = "$(cat "${ART}/validator-address.txt")" ] || die "validator addr drift"
[ "${FAUCET_ADDR}" = "$(cat "${ART}/faucet-address.txt")" ] || die "faucet addr drift"

info "genesis accounts (zero pre-mine: validator stake + faucet float)"
"${BINARY}" add-genesis-account "${VAL_ADDR}"    "${VAL_BALANCE}"    --home "${CEREMONY}" >/dev/null 2>&1
"${BINARY}" add-genesis-account "${FAUCET_ADDR}" "${FAUCET_BALANCE}" --home "${CEREMONY}" >/dev/null 2>&1

info "gentx (self-delegate ${VAL_STAKE}, reusing consensus key)"
"${BINARY}" genesis gentx validator "${VAL_STAKE}" --chain-id "${CHAIN_ID}" "${KR[@]}" >/dev/null 2>&1
"${BINARY}" genesis collect-gentxs --home "${CEREMONY}" >/dev/null 2>&1

info "patch testnet params (short windows, real fee floor, witness gate, registrar=faucet)"
GEN="${CEREMONY}/config/genesis.json"
jq --arg faucet "${FAUCET_ADDR}" '
  .app_state.gov.params.voting_period = "120s" |
  .app_state.gov.params.expedited_voting_period = "60s" |
  .app_state.gov.params.max_deposit_period = "300s" |
  (.app_state.substrate_bridge.params.witness_reward_challenge_window_blocks) = "200" |
  (if .app_state.claiming_pot.params.bootstrap_registrar != null
     then .app_state.claiming_pot.params.bootstrap_registrar = $faucet else . end)
' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

info "validate"
"${BINARY}" genesis validate --home "${CEREMONY}" >/dev/null 2>&1 || "${BINARY}" validate-genesis "${GEN}" >/dev/null 2>&1

cp "${GEN}" "${ART}/genesis.json"
NEW_NODE_ID="$("${BINARY}" tendermint show-node-id --home "${CEREMONY}")"
[ "${NEW_NODE_ID}" = "$(cat "${ART}/node-id.txt")" ] || die "node-id drift: ${NEW_NODE_ID}"

ok "regenerated ${ART}/genesis.json on $("${BINARY}" version 2>/dev/null || echo current) binary"
echo "     chain-id:  ${CHAIN_ID}"
echo "     node-id:   ${NEW_NODE_ID} (stable)"
echo "     validator: ${VAL_ADDR} (stable)"
echo "     faucet:    ${FAUCET_ADDR} (stable, 1,000,000 ZRN)"
echo "     registrar: ${FAUCET_ADDR} (claiming_pot bootstrap admission, if param present)"
