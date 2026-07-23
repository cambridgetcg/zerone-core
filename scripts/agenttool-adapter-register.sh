#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Register (or re-register) the agenttool-invocation-v1 adapter via governance
# ═══════════════════════════════════════════════════════════════════════════
#
# ZERONE-WIRE Phase 1 as a repeatable runbook: a gov v1 proposal carrying
# MsgRegisterAdapter (authority-gated), voted through the 4 localnet
# validators. WriteAdapter permits in-place update of a non-tombstoned
# adapter, so re-running with a new witness reward is an UPDATE.
#
# Usage:
#   scripts/agenttool-adapter-register.sh [witness-reward-uzrn]
#
#   witness-reward-uzrn  uzrn minted (cap-gated, survival-escrowed) per
#                        witness-only attestation settled through the
#                        adapter. Default 222000 (0.222 ZRN — one bootstrap
#                        unit per witnessed invocation). "0" disables.
#
# min_attestation_bond_uzrn — HISTORY, read before re-running:
#   An earlier draft of this runbook hardcoded 22200000 (22.2 ZRN). That
#   value never reached a live net: BOTH zerone-1 (mainnet, via
#   deploy/mainnet/make-genesis.sh) and zerone-testnet-1 (via
#   scripts/testnet-adapter-register.sh) registered the adapter with
#   1000000 (1 ZRN), and the relay defaults RELAY_BOND_UZRN=1000000 to
#   match. WriteAdapter re-registration is an in-place UPDATE, so re-running
#   this script with 22200000 would silently raise the floor above every
#   relay submission's bond and break all subsequent submits. The default
#   below matches live; raise it only together with a coordinated
#   relay-side RELAY_BOND_UZRN change.
#
# Env overrides: BINARY NODE CHAIN_ID VAL0_HOME VAL3_HOME SPONSOR
#                MIN_ATTESTATION_BOND_UZRN (default 1000000 — live value)

set -euo pipefail

WITNESS_REWARD_UZRN="${1:-222000}"
MIN_ATTESTATION_BOND_UZRN="${MIN_ATTESTATION_BOND_UZRN:-1000000}"

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
NODE="${NODE:-tcp://localhost:26601}"
CHAIN_ID="${CHAIN_ID:-zerone-localnet}"
VAL0_HOME="${VAL0_HOME:-${HOME}/.zeroned/localnet/val0}"
VAL3_HOME="${VAL3_HOME:-${HOME}/.zeroned/localnet/val3}"   # holds val1..val3 keys
SPONSOR="${SPONSOR:-test1}"
GOV_AUTHORITY="zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"

ADAPTER_ID="agenttool-invocation-v1"

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

tx_code_hash() { # reads broadcast json on stdin → hash, dies on code!=0
  local out code hash
  out=$(cat)
  code=$(jq -r .code <<<"${out}")
  hash=$(jq -r .txhash <<<"${out}")
  [ "${code}" = "0" ] || die "broadcast rejected (code ${code}): $(jq -r .raw_log <<<"${out}" | head -c 200)"
  echo "${hash}"
}

info "registering ${ADAPTER_ID} with witness_reward_uzrn=${WITNESS_REWARD_UZRN} min_attestation_bond_uzrn=${MIN_ATTESTATION_BOND_UZRN}"

PROP_FILE=$(mktemp)
trap 'rm -f "${PROP_FILE}"' EXIT
cat > "${PROP_FILE}" <<PROP
{
  "messages": [{
    "@type": "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
    "authority": "${GOV_AUTHORITY}",
    "adapter": {
      "adapter_id": "${ADAPTER_ID}",
      "source_type": "agenttool",
      "version": "1.1.0",
      "min_attestation_bond_uzrn": "${MIN_ATTESTATION_BOND_UZRN}",
      "min_per_claim_bond_uzrn": "100000",
      "allowed_class_ids": [],
      "status": 1,
      "witness_reward_uzrn": "${WITNESS_REWARD_UZRN}"
    }
  }],
  "metadata": "agenttool bridge: register ${ADAPTER_ID} (witness reward ${WITNESS_REWARD_UZRN} uzrn, survival-escrowed)",
  "deposit": "10000000uzrn",
  "title": "Register adapter ${ADAPTER_ID}",
  "summary": "Gov-register the agenttool marketplace invocation adapter (ZERONE-WIRE Phase 1). Witness-only attestations through it escrow ${WITNESS_REWARD_UZRN} uzrn under the challenge window and mint via MintWithCap only on survival — issuance follows survival, not acceptance."
}
PROP

HASH=$("${BINARY}" tx gov submit-proposal "${PROP_FILE}" --from "${SPONSOR}" \
  --home "${VAL0_HOME}" --fees 300000uzrn --gas 300000 "${TX_BASE[@]}" | tx_code_hash)
wait_tx "${HASH}" "proposal submitted"
PROP_ID=$("${BINARY}" query gov proposals --node "${NODE}" -o json | jq -r '.proposals[-1].id')
info "proposal ${PROP_ID} — voting from all 4 validators"
HASH=$("${BINARY}" tx gov vote "${PROP_ID}" yes --from val0 --home "${VAL0_HOME}" \
  --fees 300000uzrn --gas 300000 "${TX_BASE[@]}" | tx_code_hash)
wait_tx "${HASH}" "val0 voted yes"
for V in val1 val2 val3; do
  HASH=$("${BINARY}" tx gov vote "${PROP_ID}" yes --from "${V}" --home "${VAL3_HOME}" \
    --fees 300000uzrn --gas 300000 "${TX_BASE[@]}" | tx_code_hash)
  wait_tx "${HASH}" "${V} voted yes"
done
info "waiting for voting period to end…"
STATUS=""
for i in $(seq 1 40); do
  STATUS=$("${BINARY}" query gov proposal "${PROP_ID}" --node "${NODE}" -o json | jq -r .proposal.status)
  [ "${STATUS}" != "PROPOSAL_STATUS_VOTING_PERIOD" ] && break
  sleep 3
done
[ "${STATUS}" = "PROPOSAL_STATUS_PASSED" ] || die "proposal ${PROP_ID} ended ${STATUS}"
ok "proposal ${PROP_ID} PASSED"

"${BINARY}" query substrate_bridge adapter "${ADAPTER_ID}" --node "${NODE}" -o json \
  | jq '{adapter_id: .adapter.adapter_id, status: .adapter.status, min_attestation_bond_uzrn: .adapter.min_attestation_bond_uzrn, witness_reward_uzrn: .adapter.witness_reward_uzrn, registered_at_block: .adapter.registered_at_block}'
ok "${ADAPTER_ID} live with witness reward ${WITNESS_REWARD_UZRN} uzrn, min bond ${MIN_ATTESTATION_BOND_UZRN} uzrn"
