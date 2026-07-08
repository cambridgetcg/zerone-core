#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# zerone-1 — CUSTODIAL LAUNCH genesis (honest, single operator, for fun)
# ═══════════════════════════════════════════════════════════════════════════
#
# This is the honest custodial launch (see deploy/mainnet/TRUST.md): ONE
# operator-run validator, disclosed plainly, no fake decentralization and no
# false "zero pre-mine" claim. Genesis holds only what an honest custodial
# chain needs:
#   - validator: 11,111 ZRN self-bonded stake (operator collateral) + 222 ZRN
#     spendable gas
#   - zerone-ops: 2,222 ZRN operator float — pays gov deposits and feegrants
#     agents their first gas so they can claim the 0.222 ZRN bootstrap bonus
#     (which MINTS on demand — no faucet pre-mine)
# Total genesis supply: 13,555 ZRN = 0.0061% of the 222,222,222 cap. Every
# address is published in the manifest. Everything else mints on participation
# under MintWithCap. NO team/foundation/investor/faucet allocation.
#
# Custodial-launch-phase honesty: while custodial, the operator may re-genesis
# (TRUST.md says so) — the record seals only when the network earns independence.
#
# Usage: deploy/mainnet/make-genesis.sh [output-dir]
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"
OUT="${1:-${PROJECT_ROOT}/deploy/mainnet/artifacts}"
KEYRING="${KEYRING:-${HOME}/.zeroned/mainnet-ops}"
CHAIN_ID="zerone-1"
DENOM="uzrn"
MONIKER="zerone-1-genesis"
GOV_AUTHORITY="zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"

VAL_BALANCE="11333000000${DENOM}"  # 11,333 ZRN (11,111 bonded + 222 gas)
VAL_STAKE="11111000000${DENOM}"    # 11,111 ZRN self-bonded
OPS_BALANCE="2222000000${DENOM}"   # 2,222 ZRN operator float (deposits + feegrants)

info() { printf '\033[1;34m  ->\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  OK\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

CEREMONY="$(mktemp -d)"; trap 'rm -rf "${CEREMONY}"' EXIT
# gentx uses one --home for BOTH keyring and genesis, so the ceremony keyring
# lives in the ceremony home; the operator keyring (${KEYRING}) is kept in sync
# for later use (adapter registration, relay).
KR=(--keyring-backend test --home "${CEREMONY}")
OPSKR=(--keyring-backend test --home "${KEYRING}")
mkdir -p "${OUT}" "${KEYRING}"

info "init ${CHAIN_ID}"
"${BINARY}" init "${MONIKER}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" --home "${CEREMONY}" >/dev/null 2>&1

# Keys (custodial — operator-held; created once in artifacts/, reused on re-runs).
for k in validator zerone-ops; do
  if [ -f "${OUT}/${k}.mnemonic" ]; then
    "${BINARY}" keys add "$k" --recover "${KR[@]}"    >/dev/null 2>&1 < "${OUT}/${k}.mnemonic"
    "${BINARY}" keys show "$k" "${OPSKR[@]}" >/dev/null 2>&1 || "${BINARY}" keys add "$k" --recover "${OPSKR[@]}" >/dev/null 2>&1 < "${OUT}/${k}.mnemonic"
  else
    "${BINARY}" keys add "$k" "${KR[@]}" --output json 2>/dev/null | jq -r .mnemonic > "${OUT}/${k}.mnemonic"
    "${BINARY}" keys add "$k" --recover "${OPSKR[@]}" >/dev/null 2>&1 < "${OUT}/${k}.mnemonic"
  fi
done
VAL_ADDR="$("${BINARY}" keys show validator -a "${KR[@]}")"
OPS_ADDR="$("${BINARY}" keys show zerone-ops -a "${KR[@]}")"

# Reuse node key so the seed id is stable across re-genesis.
[ -f "${OUT}/node_key.json" ] && cp "${OUT}/node_key.json" "${CEREMONY}/config/node_key.json"

info "genesis accounts (honest: validator stake+gas + operator float; NO faucet pre-mine)"
"${BINARY}" add-genesis-account "${VAL_ADDR}" "${VAL_BALANCE}" --home "${CEREMONY}" >/dev/null 2>&1
"${BINARY}" add-genesis-account "${OPS_ADDR}" "${OPS_BALANCE}" --home "${CEREMONY}" >/dev/null 2>&1

info "gentx (self-bond ${VAL_STAKE})"
"${BINARY}" genesis gentx validator "${VAL_STAKE}" --chain-id "${CHAIN_ID}" "${KR[@]}" >/dev/null 2>&1
"${BINARY}" genesis collect-gentxs --home "${CEREMONY}" >/dev/null 2>&1

info "honest params (uzrn gov deposits, agile 10-min voting, registrar = operator, no knowledge InitGenesis mint)"
GEN="${CEREMONY}/config/genesis.json"
jq --arg ops "${OPS_ADDR}" '
  .app_state.gov.params.min_deposit = [{"denom":"uzrn","amount":"100000000"}] |
  .app_state.gov.params.expedited_min_deposit = [{"denom":"uzrn","amount":"300000000"}] |
  .app_state.gov.params.voting_period = "600s" |
  .app_state.gov.params.expedited_voting_period = "300s" |
  .app_state.gov.params.max_deposit_period = "1200s" |
  (.app_state.substrate_bridge.params.witness_reward_challenge_window_blocks) = "200" |
  (if .app_state.knowledge.bootstrap_fund_allocation != null then .app_state.knowledge.bootstrap_fund_allocation = "0" else . end) |
  (if .app_state.claiming_pot.params.bootstrap_registrar != null then .app_state.claiming_pot.params.bootstrap_registrar = $ops else . end)
' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

# Adapter ACTIVE at genesis so the witness economy works from block 0 (no 12h
# gov wait). ceremony-inject sets a 22.2 ZRN bond; patch it to a fun-cheap 1 ZRN
# so newborn agents (0.222 ZRN bonus) can afford to witness.
info "injecting agenttool-invocation-v1 adapter (ACTIVE at genesis, 1 ZRN witness bond)"
INJECT="$(mktemp -d)/ceremony-inject"
(cd "${PROJECT_ROOT}" && go build -o "${INJECT}" ./tools/ceremony-inject) 2>/dev/null
"${INJECT}" adapter "${GEN}" >/dev/null 2>&1
jq '(.app_state.substrate_bridge.adapters[]? | select(.adapter_id=="agenttool-invocation-v1") | .min_attestation_bond_uzrn) = "1000000"' "${GEN}" > "${GEN}.tmp" && mv "${GEN}.tmp" "${GEN}"

"${BINARY}" genesis validate --home "${CEREMONY}" >/dev/null 2>&1 || "${BINARY}" validate-genesis "${GEN}" >/dev/null 2>&1

cp "${GEN}" "${OUT}/genesis.json"
cp "${CEREMONY}/config/node_key.json" "${OUT}/node_key.json"
cp "${CEREMONY}/config/priv_validator_key.json" "${OUT}/priv_validator_key.json"
NODE_ID="$("${BINARY}" tendermint show-node-id --home "${CEREMONY}")"
SUPPLY=$(jq -r '.app_state.bank.supply[0].amount' "${OUT}/genesis.json")
GENHASH=$(shasum -a 256 "${OUT}/genesis.json" | awk '{print $1}')

cat > "${OUT}/GENESIS-MANIFEST.md" <<MAN
# zerone-1 genesis manifest (custodial launch)

- chain-id: \`${CHAIN_ID}\`
- genesis supply: **${SUPPLY} uzrn** ($(echo "scale=3; ${SUPPLY}/1000000" | bc) ZRN = 0.0061% of the 222,222,222 cap)
- genesis sha256: \`${GENHASH}\`
- node-id: \`${NODE_ID}\`

## Every genesis address (nothing hidden)
- validator (operator, self-bonded stake): \`${VAL_ADDR}\` — 11,333 ZRN (11,111 bonded + 222 gas)
- zerone-ops (operator float; gov deposits + onboarding feegrants): \`${OPS_ADDR}\` — 2,222 ZRN

No team / foundation / investor / faucet allocation. All other ZRN mints on
participation under MintWithCap. Custodial launch — see TRUST.md.
MAN

ok "zerone-1 genesis: ${SUPPLY} uzrn, sha256 ${GENHASH:0:16}…"
echo "     validator: ${VAL_ADDR}"
echo "     ops:       ${OPS_ADDR}"
echo "     node-id:   ${NODE_ID}"
