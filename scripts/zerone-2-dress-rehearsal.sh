#!/usr/bin/env bash
# Full local zerone-2 rehearsal using public fixtures only.
#
# Boots the audited genesis, onboards the operator identity, registers the
# custom validator with real in-chain escrow, restarts, creates a continuation
# export at the stopped height, imports it into a fresh home, and proves
# supply/backing survive. Nothing here is production custody material and no
# remote endpoint is contacted.

set -euo pipefail
export LC_ALL=C
export GOPROXY=off
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${ROOT}/build/zeroned}"
CHAIN_ID="zerone-2"
RPC="tcp://127.0.0.1:37657"
RPC_HTTP="http://127.0.0.1:37657"
REST="http://127.0.0.1:31317"
TOTAL_SUPPLY="13555000000"
CUSTOM_ESCROW="111000000"
CUSTOM_STAKING_MODULE="zrn1ehtmkw3djuxxprsr8ueknnamwk3jvkpmlzfepn"
DRILL_MNEMONIC="now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-dress.XXXXXX")"
NODE_PID=""

info() { printf '  -> %s\n' "$*"; }
ok() { printf '  OK %s\n' "$*"; }
fail() {
  printf 'zerone-2 dress rehearsal: FAIL: %s\n' "$*" >&2
  if [ -f "${TMP}/node.log" ]; then
    sed -n '1,40p' "${TMP}/node.log" >&2 || true
    tail -80 "${TMP}/node.log" >&2 || true
  fi
  exit 1
}

stop_node() {
  if [ -n "${NODE_PID}" ] && kill -0 "${NODE_PID}" 2>/dev/null; then
    kill -TERM "${NODE_PID}" 2>/dev/null || true
    for _ in $(seq 1 40); do
      kill -0 "${NODE_PID}" 2>/dev/null || break
      sleep 0.25
    done
    kill -KILL "${NODE_PID}" 2>/dev/null || true
    wait "${NODE_PID}" 2>/dev/null || true
  fi
  NODE_PID=""
}

cleanup() {
  stop_node
  chmod -R u+w "${TMP}" 2>/dev/null || true
  rm -rf "${TMP}"
}
trap cleanup EXIT INT TERM

sed_in_place() {
  local expression="$1" file="$2"
  if sed --version >/dev/null 2>&1; then
    sed -i "${expression}" "${file}"
  else
    sed -i '' "${expression}" "${file}"
  fi
}

configure_home() {
  local home="$1"
  local config="${home}/config/config.toml"
  local app="${home}/config/app.toml"
  sed_in_place 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://127.0.0.1:37657"|' "${config}"
  sed_in_place 's|^laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://127.0.0.1:37656"|' "${config}"
  sed_in_place 's|^timeout_commit = .*|timeout_commit = "500ms"|' "${config}"
  sed_in_place 's|^prometheus = .*|prometheus = false|' "${config}"
  sed_in_place 's|^pprof_laddr = .*|pprof_laddr = ""|' "${config}"
  sed_in_place 's|^minimum-gas-prices = .*|minimum-gas-prices = "1uzrn"|' "${app}"
  sed_in_place 's|^pruning = .*|pruning = "nothing"|' "${app}"
  sed_in_place 's|^address = "tcp://localhost:1317"|address = "tcp://127.0.0.1:31317"|' "${app}"
  sed_in_place 's|^address = "localhost:9090"|address = "127.0.0.1:39090"|' "${app}"
  sed_in_place 's|^max-txs = -1|max-txs = 5000|' "${app}"
  sed_in_place 's|^iavl-disable-fastnode = false|iavl-disable-fastnode = true|' "${app}"
}

start_node() {
  local home="$1" log="$2"
  "${BINARY}" start --home "${home}" --minimum-gas-prices 1uzrn \
    --rpc.laddr tcp://127.0.0.1:37657 \
    --p2p.laddr tcp://127.0.0.1:37656 \
    --api.address tcp://127.0.0.1:31317 \
    --grpc.address 127.0.0.1:39090 \
    > "${log}" 2>&1 &
  NODE_PID=$!
}

wait_for_height() {
  local target="$1"
  for _ in $(seq 1 160); do
    if ! kill -0 "${NODE_PID}" 2>/dev/null; then
      fail "node exited before height ${target}"
    fi
    local status height chain catching
    status="$(curl --noproxy '*' -fsS --max-time 1 "${RPC_HTTP}/status" 2>/dev/null || true)"
    if [ -n "${status}" ]; then
      height="$(jq -r '.result.sync_info.latest_block_height // "0"' <<<"${status}")"
      chain="$(jq -r '.result.node_info.network // ""' <<<"${status}")"
      catching="$(jq -r 'if .result.sync_info.catching_up == false then "false" else "true" end' <<<"${status}")"
      [ "${chain}" = "${CHAIN_ID}" ] || fail "node reported chain ${chain}"
      if [ "${catching}" = "false" ] && [ "${height}" -ge "${target}" ]; then
        return 0
      fi
    fi
    sleep 0.25
  done
  fail "timed out waiting for height ${target}"
}

rest_json() {
  curl --noproxy '*' -fsS --max-time 3 "${REST}$1"
}

assert_supply() {
  local expected="$1" amount
  amount="$(rest_json '/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn' \
    | jq -r '.amount.amount')"
  [ "${amount}" = "${expected}" ] || \
    fail "supply ${amount} != ${expected}"
}

wait_for_tx() {
  local hash="$1"
  for _ in $(seq 1 120); do
    local result
    result="$(${BINARY} query tx "${hash}" --node "${RPC}" -o json 2>/dev/null || true)"
    if [ -n "${result}" ] && jq -e '.code == 0' <<<"${result}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  fail "transaction ${hash} did not commit successfully"
}

broadcast() {
  local output code hash
  output="$(${BINARY} "$@" --node "${RPC}" --chain-id "${CHAIN_ID}" \
    --keyring-backend test --broadcast-mode sync -o json --yes \
    2>> "${TMP}/tx.log")" || \
    fail "transaction command failed"
  code="$(jq -r '.code // 0' <<<"${output}")"
  hash="$(jq -r '.txhash // ""' <<<"${output}")"
  [ "${code}" = "0" ] || fail "CheckTx rejected transaction: ${output}"
  [[ "${hash}" =~ ^[0-9A-F]{64}$ ]] || fail "transaction returned invalid hash"
  wait_for_tx "${hash}"
  printf '%s\n' "${hash}"
}

[ -x "${BINARY}" ] || fail "binary missing at ${BINARY}"
command -v curl >/dev/null || fail "curl is required"
command -v jq >/dev/null || fail "jq is required"
command -v xxd >/dev/null || fail "xxd is required"

ARTIFACTS="${TMP}/artifacts"
HOME_A="${TMP}/home-a"
HOME_B="${TMP}/home-b"

info "running reproducible ceremony with mandatory artifact audit"
"${ROOT}/scripts/zerone-2-ceremony.sh" drill "${ARTIFACTS}" \
  > "${TMP}/ceremony.log" 2>&1 || {
    sed -n '1,180p' "${TMP}/ceremony.log" >&2
    fail "ceremony failed"
  }
grep -q 'PASS zerone-2 artifact audit' "${TMP}/ceremony.log" || \
  fail "ceremony did not run mandatory artifact audit"

info "building the matching public drill validator home"
"${BINARY}" init zerone-2-dress --chain-id "${CHAIN_ID}" --home "${HOME_A}" >/dev/null 2>&1
cp "${ARTIFACTS}/genesis.json" "${HOME_A}/config/genesis.json"
go build -o "${TMP}/ceremony-inject" "${ROOT}/tools/ceremony-inject"
"${TMP}/ceremony-inject" drill-consensus-key zerone-2-public-drill \
  "${HOME_A}/config/priv_validator_key.json" >/dev/null 2>&1
printf '%s\n' "${DRILL_MNEMONIC}" \
  | "${BINARY}" keys add validator --recover --account 201 \
      --keyring-backend test --home "${HOME_A}" >/dev/null 2>&1
VALIDATOR_ADDRESS="$(${BINARY} keys show validator -a --keyring-backend test --home "${HOME_A}")"
configure_home "${HOME_A}"

info "booting audited genesis"
start_node "${HOME_A}" "${TMP}/node.log"
wait_for_height 3
assert_supply "${TOTAL_SUPPLY}"

VOTE_EXT_HEIGHT="$(rest_json '/cosmos/consensus/v1/params' \
  | jq -r '.params.abci.vote_extensions_enable_height // "0"')"
[ "${VOTE_EXT_HEIGHT}" = "0" ] || fail "vote extensions unexpectedly enabled"

info "onboarding the public-fixture operator identity"
IDENTITY_FILE="${TMP}/validator-identity.ed25519.json"
ONBOARD_HASH="$(broadcast tx zerone_auth onboard human \
  --from validator --identity-out "${IDENTITY_FILE}" \
  --home "${HOME_A}" --gas 200000 --fees 200000uzrn)"
[ -s "${IDENTITY_FILE}" ] || fail "onboarding did not create the identity file"
ACCOUNT_QUERY="$(${BINARY} query zerone_auth account "${VALIDATOR_ADDRESS}" \
  --node "${RPC}" -o json)"
DID="$(jq -r '.account.did // .did // ""' <<<"${ACCOUNT_QUERY}")"
[[ "${DID}" =~ ^did:zrn:[0-9a-f]{64}$ ]] || fail "on-chain identity DID is invalid"

CONSENSUS_PUBKEY_HEX="$(jq -r '.pub_key.value' "${HOME_A}/config/priv_validator_key.json" \
  | base64 --decode | xxd -p -c 64)"
[[ "${CONSENSUS_PUBKEY_HEX}" =~ ^[0-9a-f]{64}$ ]] || \
  fail "consensus public key did not decode to 32 bytes"

info "registering custom validator with 111 ZRN real escrow"
REGISTER_HASH="$(broadcast tx zerone_staking register-validator \
  "${CONSENSUS_PUBKEY_HEX}" "${CUSTOM_ESCROW}" \
  --from validator --moniker zerone-2-custodian --identity "${DID}" \
  --commission 500 --details 'One publicly disclosed custodial validator' \
  --home "${HOME_A}" --gas 200000 --fees 200000uzrn)"

CUSTOM_QUERY="$(${BINARY} query zerone_staking validator "${VALIDATOR_ADDRESS}" \
  --node "${RPC}" -o json)"
jq -e --arg operator "${VALIDATOR_ADDRESS}" --arg pubkey "${CONSENSUS_PUBKEY_HEX}" \
  --arg escrow "${CUSTOM_ESCROW}" '
  (.validator.operator_address // .operator_address) == $operator
  and (.validator.consensus_pubkey // .consensus_pubkey) == $pubkey
  and ((.validator.self_delegation // .self_delegation) | tostring) == $escrow
' <<<"${CUSTOM_QUERY}" >/dev/null || fail "custom validator record does not match custody/escrow"

MODULE_BALANCE="$(rest_json "/cosmos/bank/v1beta1/balances/${CUSTOM_STAKING_MODULE}/by_denom?denom=uzrn" \
  | jq -r '.balance.amount // "0"')"
[ "${MODULE_BALANCE}" = "${CUSTOM_ESCROW}" ] || \
  fail "custom staking module backing ${MODULE_BALANCE} != ${CUSTOM_ESCROW}"
assert_supply "${TOTAL_SUPPLY}"
ok "private bootstrap txs committed: ${ONBOARD_HASH}, ${REGISTER_HASH}"

info "restarting from the same database"
HEIGHT_BEFORE="$(curl --noproxy '*' -fsS --max-time 3 "${RPC_HTTP}/status" \
  | jq -r '.result.sync_info.latest_block_height')"
stop_node
start_node "${HOME_A}" "${TMP}/node.log"
wait_for_height "$((HEIGHT_BEFORE + 2))"
assert_supply "${TOTAL_SUPPLY}"

info "exporting all modules as a stopped-height continuation genesis"
stop_node
EXPORT="${TMP}/export.json"
if ! "${BINARY}" export --home "${HOME_A}" \
  --output-document "${EXPORT}" > "${TMP}/export.log" 2>&1; then
  sed -n '1,160p' "${TMP}/export.log" >&2
  fail "state export failed"
fi
if ! "${BINARY}" genesis validate "${EXPORT}" \
  > "${TMP}/export-validate.log" 2>&1; then
  sed -n '1,160p' "${TMP}/export-validate.log" >&2
  fail "exported genesis failed validation"
fi
EXPORTED_INITIAL_HEIGHT="$(jq -er '.initial_height | tostring' "${EXPORT}")" || \
  fail "exported continuation genesis lacks initial_height"
[[ "${EXPORTED_INITIAL_HEIGHT}" =~ ^[1-9][0-9]*$ ]] && \
  [ "${EXPORTED_INITIAL_HEIGHT}" -gt 1 ] || \
  fail "continuation export must preserve a non-zero stopped height"
EXPORTED_SUPPLY="$(jq -r '[.app_state.bank.supply[] | select(.denom=="uzrn") | .amount][0]' "${EXPORT}")"
[ "${EXPORTED_SUPPLY}" = "${TOTAL_SUPPLY}" ] || \
  fail "exported supply ${EXPORTED_SUPPLY} != ${TOTAL_SUPPLY}"

info "importing export into an isolated fresh home"
"${BINARY}" init zerone-2-import --chain-id "${CHAIN_ID}" --home "${HOME_B}" >/dev/null 2>&1
cp "${EXPORT}" "${HOME_B}/config/genesis.json"
cp "${HOME_A}/config/priv_validator_key.json" "${HOME_B}/config/priv_validator_key.json"
printf '%s\n' '{"height":"0","round":0,"step":0}' \
  > "${HOME_B}/data/priv_validator_state.json"
chmod 0600 "${HOME_B}/config/priv_validator_key.json" \
  "${HOME_B}/data/priv_validator_state.json"
configure_home "${HOME_B}"
start_node "${HOME_B}" "${TMP}/node.log"
IMPORTED_TARGET_HEIGHT="$((EXPORTED_INITIAL_HEIGHT + 9))"
wait_for_height "${IMPORTED_TARGET_HEIGHT}"
assert_supply "${TOTAL_SUPPLY}"

IMPORTED_CUSTOM="$(${BINARY} query zerone_staking validator "${VALIDATOR_ADDRESS}" \
  --node "${RPC}" -o json)"
jq -e --arg escrow "${CUSTOM_ESCROW}" '
  ((.validator.self_delegation // .self_delegation) | tostring) == $escrow
' <<<"${IMPORTED_CUSTOM}" >/dev/null || fail "custom staking record did not survive export/import"
IMPORTED_MODULE_BALANCE="$(rest_json "/cosmos/bank/v1beta1/balances/${CUSTOM_STAKING_MODULE}/by_denom?denom=uzrn" \
  | jq -r '.balance.amount // "0"')"
[ "${IMPORTED_MODULE_BALANCE}" = "${CUSTOM_ESCROW}" ] || \
  fail "custom staking backing did not survive export/import"

EXPORT_SHA="$(shasum -a 256 "${EXPORT}" | awk '{print $1}')"
ok "restart and ten-block stopped-height export/import rehearsal passed"
ok "supply stayed exactly ${TOTAL_SUPPLY}uzrn; export sha256 ${EXPORT_SHA}"
printf 'zerone-2 dress rehearsal: PASS\n'
