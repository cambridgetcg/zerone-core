#!/usr/bin/env bash
# Isolated four-validator consensus and MsgSend rehearsal.
#
# Every binary, home, key, log, and Go build cache lives below a fresh mktemp
# directory. All listeners and peers are loopback-only. The directory is
# removed only after a successful run unless --keep/KEEP_REHEARSAL=1 is set;
# failed runs are deliberately retained for diagnosis.

set -euo pipefail
export LC_ALL=C
export LANG=C
export GOPROXY=off
export GOFLAGS=-mod=readonly
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHAIN_ID="zerone-consensus-rehearsal-1"
DENOM="uzrn"
VALIDATOR_BALANCE="2000000000"
VALIDATOR_STAKE="1000000000"
SENDER_BALANCE="1000000000"
RECEIVER_BALANCE="1000000"
SEND_AMOUNT="12345"
TX_FEE="250000"
TX_GAS="250000"
KEEP="${KEEP_REHEARSAL:-0}"
ALLOW_DIRTY=0
SUCCESS=0
RUN_ROOT=""
BINARY=""
VERIFY_BINARY=""
CENSUS_BINARY=""
SOURCE_HEAD=""
SOURCE_FULL_HEAD=""
SOURCE_LABEL=""
BASE_PORT=""

declare -a NODE_HOME NODE_LOG NODE_PID NODE_ID P2P_PORT RPC_PORT

usage() {
  cat <<'USAGE'
Usage: scripts/local-consensus-rehearsal.sh [--keep] [--allow-dirty]

Builds the current checkout into fresh temporary state and proves:
  * four equal-power validators agree on block ID and app hash;
  * three validators continue finality after one validator stops;
  * two validators cannot finalize after a second validator stops;
  * the stopped validators recover and converge;
  * one signed MsgSend commits, verifies with a Merkle proof on all nodes,
    increments the sender sequence exactly once, and is rejected on replay; and
  * two stopped, independently copied application databases produce the same
    AppHash-bound, passing legacy custom-staking census.

--keep  Retain successful state and logs (failures are always retained).
--allow-dirty
        Development-only escape hatch. A dirty build is labeled NON-FINAL;
        the definitive rehearsal refuses any tracked or untracked changes.
USAGE
}

info() { printf '  -> %s\n' "$*"; }
ok() { printf '  OK %s\n' "$*"; }

die() {
  printf 'local consensus rehearsal: FAIL: %s\n' "$*" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

normalize_app_hash() {
  python3 - "$1" <<'PY'
import base64
import binascii
import re
import sys

value = sys.argv[1]
if re.fullmatch(r"[0-9A-Fa-f]{64}", value):
    print(value.lower())
    raise SystemExit(0)
try:
    decoded = base64.b64decode(value, validate=True)
except (binascii.Error, ValueError) as error:
    raise SystemExit(f"invalid AppHash encoding: {error}")
if len(decoded) != 32:
    raise SystemExit(f"decoded AppHash is {len(decoded)} bytes, expected 32")
print(decoded.hex())
PY
}

is_owned_pid() {
  local index="$1" pid command_line
  pid="${NODE_PID[$index]:-}"
  [ -n "${pid}" ] || return 1
  kill -0 "${pid}" 2>/dev/null || return 1
  command_line="$(ps -p "${pid}" -o command= 2>/dev/null || true)"
  case "${command_line}" in
    *"${BINARY}"*"start"*"--home ${NODE_HOME[$index]}"*) return 0 ;;
    *) return 1 ;;
  esac
}

stop_node() {
  local index="$1" pid
  pid="${NODE_PID[$index]:-}"
  [ -n "${pid}" ] || return 0

  if kill -0 "${pid}" 2>/dev/null; then
    is_owned_pid "${index}" || die "refusing to signal unrecognized PID ${pid} for node ${index}"
    kill -TERM "${pid}" 2>/dev/null || true
    for _ in $(seq 1 40); do
      kill -0 "${pid}" 2>/dev/null || break
      sleep 0.25
    done
    if kill -0 "${pid}" 2>/dev/null; then
      is_owned_pid "${index}" || die "PID ${pid} changed identity while stopping node ${index}"
      kill -KILL "${pid}" 2>/dev/null || true
    fi
    wait "${pid}" 2>/dev/null || true
  fi
  NODE_PID[index]=""
}

stop_all() {
  local index
  for index in 0 1 2 3; do
    stop_node "${index}" || true
  done
}

show_diagnostics() {
  local log
  [ -n "${RUN_ROOT}" ] || return 0
  for log in "${RUN_ROOT}"/logs/*.log; do
    [ -f "${log}" ] || continue
    printf '\n== %s (last 60 lines) ==\n' "${log}" >&2
    tail -60 "${log}" >&2 || true
  done
}

safe_remove_run_root() {
  [ -n "${RUN_ROOT}" ] || return 0
  [ -f "${RUN_ROOT}/.zerone-consensus-rehearsal-owned" ] || \
    die "refusing to remove unmarked path ${RUN_ROOT}"
  case "$(basename "${RUN_ROOT}")" in
    zerone-consensus-rehearsal.*) ;;
    *) die "refusing to remove unexpected path ${RUN_ROOT}" ;;
  esac
  rm -rf -- "${RUN_ROOT}"
}

cleanup() {
  local rc=$?
  trap - EXIT INT TERM
  stop_all
  if [ "${rc}" -eq 0 ] && [ "${SUCCESS}" -eq 1 ] && [ "${KEEP}" != "1" ]; then
    safe_remove_run_root
  else
    if [ "${rc}" -ne 0 ]; then
      show_diagnostics
    fi
    [ -z "${RUN_ROOT}" ] || printf '\nRehearsal state retained at: %s\n' "${RUN_ROOT}" >&2
  fi
  exit "${rc}"
}

trap cleanup EXIT
trap 'exit 130' INT TERM

sed_in_place() {
  local expression="$1" file="$2"
  if sed --version >/dev/null 2>&1; then
    sed -i "${expression}" "${file}"
  else
    sed -i '' "${expression}" "${file}"
  fi
}

port_is_free() {
  ! lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

allocate_ports() {
  local candidate offset available
  for _ in $(seq 1 200); do
    candidate=$((40000 + (RANDOM % 20000)))
    candidate=$((candidate - (candidate % 16)))
    available=1
    for offset in $(seq 0 9); do
      if ! port_is_free "$((candidate + offset))"; then
        available=0
        break
      fi
    done
    if [ "${available}" -eq 1 ]; then
      BASE_PORT="${candidate}"
      for offset in 0 1 2 3; do
        P2P_PORT[offset]=$((candidate + offset * 2))
        RPC_PORT[offset]=$((candidate + offset * 2 + 1))
      done
      return 0
    fi
  done
  die "could not reserve ten free loopback TCP ports"
}

rpc_url() {
  printf 'http://127.0.0.1:%s' "${RPC_PORT[$1]}"
}

rpc_get() {
  local index="$1" path="$2"
  curl --noproxy '*' -fsS --connect-timeout 1 --max-time 3 "$(rpc_url "${index}")${path}"
}

current_height() {
  local response height
  response="$(rpc_get "$1" /status 2>/dev/null || true)"
  height="$(printf '%s' "${response}" | jq -r '.result.sync_info.latest_block_height // "0"' 2>/dev/null || printf '0')"
  case "${height}" in
    ''|*[!0-9]*) printf '0' ;;
    *) printf '%s' "${height}" ;;
  esac
}

wait_for_height() {
  local index="$1" target="$2" label="$3" height chain catching
  for _ in $(seq 1 240); do
    is_owned_pid "${index}" || die "node ${index} exited while waiting for ${label}"
    height="$(current_height "${index}")"
    if [ "${height}" -ge "${target}" ]; then
      chain="$(rpc_get "${index}" /status | jq -r '.result.node_info.network // ""')"
      catching="$(rpc_get "${index}" /status | jq -r '.result.sync_info.catching_up')"
      [ "${chain}" = "${CHAIN_ID}" ] || die "node ${index} reported unexpected chain ${chain}"
      [ "${catching}" = "false" ] || {
        sleep 0.25
        continue
      }
      return 0
    fi
    sleep 0.25
  done
  die "timed out waiting for node ${index} to reach ${label} (target ${target}, height ${height:-0})"
}

wait_for_advance() {
  local blocks="$1" label="$2" first target index
  shift 2
  first="$1"
  target=$(( $(current_height "${first}") + blocks ))
  for index in "$@"; do
    wait_for_height "${index}" "${target}" "${label}"
  done
}

configure_node() {
  local index="$1" config app
  config="${NODE_HOME[$index]}/config/config.toml"
  app="${NODE_HOME[$index]}/config/app.toml"

  sed_in_place 's|^allow_duplicate_ip = .*|allow_duplicate_ip = true|' "${config}"
  sed_in_place 's|^addr_book_strict = .*|addr_book_strict = false|' "${config}"
  sed_in_place 's|^pex = .*|pex = false|' "${config}"
  sed_in_place 's|^seeds = .*|seeds = ""|' "${config}"
  sed_in_place 's|^timeout_propose = .*|timeout_propose = "500ms"|' "${config}"
  sed_in_place 's|^timeout_commit = .*|timeout_commit = "500ms"|' "${config}"
  sed_in_place 's|^prometheus = .*|prometheus = false|' "${config}"
  sed_in_place 's|^pprof_laddr = .*|pprof_laddr = ""|' "${config}"
  sed_in_place 's|^minimum-gas-prices = .*|minimum-gas-prices = "1uzrn"|' "${app}"
}

persistent_peers() {
  local index="$1" peer peers=""
  for peer in 0 1 2 3; do
    [ "${peer}" -eq "${index}" ] && continue
    [ -z "${peers}" ] || peers="${peers},"
    peers="${peers}${NODE_ID[$peer]}@127.0.0.1:${P2P_PORT[$peer]}"
  done
  printf '%s' "${peers}"
}

start_node() {
  local index="$1" peers
  [ -z "${NODE_PID[$index]:-}" ] || die "node ${index} already has a recorded PID"
  port_is_free "${P2P_PORT[$index]}" || die "node ${index} P2P port is no longer free"
  port_is_free "${RPC_PORT[$index]}" || die "node ${index} RPC port is no longer free"
  peers="$(persistent_peers "${index}")"

  "${BINARY}" start \
    --home "${NODE_HOME[$index]}" \
    --minimum-gas-prices "1${DENOM}" \
    --rpc.laddr "tcp://127.0.0.1:${RPC_PORT[$index]}" \
    --rpc.pprof_laddr "" \
    --rpc.unsafe=false \
    --p2p.laddr "tcp://127.0.0.1:${P2P_PORT[$index]}" \
    --p2p.external-address "127.0.0.1:${P2P_PORT[$index]}" \
    --p2p.persistent_peers "${peers}" \
    --p2p.pex=false \
    --api.enable=false \
    --grpc.enable=false \
    --grpc-web.enable=false \
    --consensus.create_empty_blocks=true \
    --log_level error \
    >> "${NODE_LOG[$index]}" 2>&1 &
  NODE_PID[index]=$!
  sleep 0.25
  is_owned_pid "${index}" || die "node ${index} failed immediately (PID ${NODE_PID[$index]})"
}

assert_loopback_listeners() {
  local index="$1" listener seen_p2p=0 seen_rpc=0
  while IFS= read -r listener; do
    [ -n "${listener}" ] || continue
    case "${listener}" in
      "127.0.0.1:${P2P_PORT[$index]}"* ) seen_p2p=1 ;;
      "127.0.0.1:${RPC_PORT[$index]}"* ) seen_rpc=1 ;;
      *) die "node ${index} opened unexpected/non-loopback listener ${listener}" ;;
    esac
  done < <(lsof -nP -a -p "${NODE_PID[$index]}" -iTCP -sTCP:LISTEN 2>/dev/null | awk 'NR > 1 {print $9}')
  [ "${seen_p2p}" -eq 1 ] || die "node ${index} has no loopback P2P listener"
  [ "${seen_rpc}" -eq 1 ] || die "node ${index} has no loopback RPC listener"
}

run_commit_test() {
  local label="$1" index="$2" log
  log="${RUN_ROOT}/logs/commit-test-${label}.log"
  [ "$(git -C "${ROOT}" rev-parse HEAD)" = "${SOURCE_FULL_HEAD}" ] || \
    die "checkout HEAD changed before ${label} commit test"
  ZERONE_TEST_RPC_ADDR="$(rpc_url "${index}")" \
    GOCACHE="${RUN_ROOT}/go-cache" GOTMPDIR="${RUN_ROOT}/go-tmp" \
    go test -tags=integration ./tests/multivalidator \
      -run '^TestBlockSignatures$' -count=1 -v -timeout 30s \
      > "${log}" 2>&1 || die "cryptographic multivalidator test failed in ${label} phase"
}

verify_phase() {
  local label="$1" index endpoints="" report
  shift
  for index in "$@"; do
    [ -z "${endpoints}" ] || endpoints="${endpoints},"
    endpoints="${endpoints}$(rpc_url "${index}")"
  done
  report="${RUN_ROOT}/reports/${label}.json"
  "${VERIFY_BINARY}" \
    -rpcs "${endpoints}" \
    -expect-validators 4 \
    -expect-equal-power \
    > "${report}" || die "cross-node consensus verification failed in ${label} phase"
  run_commit_test "${label}" "$1"
  ok "${label}: canonical commit and cross-node hashes verified"
}

query_balance() {
  "${BINARY}" query bank balance "$1" "${DENOM}" \
    --home "${RUN_ROOT}/coordinator" --node "$(rpc_url "$2")" --output json \
    | jq -er '.balance.amount | select(test("^[0-9]+$"))'
}

query_sequence() {
  "${BINARY}" query auth account "$1" \
    --home "${RUN_ROOT}/coordinator" --node "$(rpc_url "$2")" --output json \
    | jq -er --arg address "$1" '
      .account as $account |
      (
        $account.address //
        $account.value.address //
        $account.base_account.address //
        $account.base_vesting_account.base_account.address //
        $account.value.base_account.address //
        $account.value.base_vesting_account.base_account.address
      ) as $actual_address |
      select($actual_address == $address) |
      (
        $account.sequence //
        $account.value.sequence //
        $account.base_account.sequence //
        $account.base_vesting_account.base_account.sequence //
        $account.value.base_account.sequence //
        $account.value.base_vesting_account.base_account.sequence //
        "0"
      ) | tostring | select(test("^(0|[1-9][0-9]*)$"))'
}

wait_for_tx() {
  local hash="$1" index="$2" response
  for _ in $(seq 1 160); do
    response="$(rpc_get "${index}" "/tx?hash=0x${hash}&prove=true" 2>/dev/null || true)"
    if printf '%s' "${response}" | jq -e --arg hash "${hash}" '
      .error == null and
      ((.result.hash | ascii_upcase) == $hash) and
      ((.result.tx_result.code | tonumber) == 0) and
      (.result.height | test("^[1-9][0-9]*$")) and
      (.result.proof != null)
    ' >/dev/null 2>&1; then
      printf '%s' "${response}"
      return 0
    fi
    sleep 0.25
  done
  die "transaction ${hash} did not commit with a proof"
}

verify_message_flow() {
  local sender receiver before_balance before_sender_balance before_sequence
  local expected_receiver_balance expected_sender_balance after_sequence
  local memo output code hash tx_query encoded decoded report replay_request replay_response replay_sequence
  local index node_receiver_balance node_sender_balance node_sequence tx_height

  : > "${RUN_ROOT}/logs/message.log"
  sender="$("${BINARY}" keys show sender -a --keyring-backend test --home "${RUN_ROOT}/coordinator")" || \
    die "could not load sender key"
  receiver="$("${BINARY}" keys show receiver -a --keyring-backend test --home "${RUN_ROOT}/coordinator")" || \
    die "could not load receiver key"
  before_balance="$(query_balance "${receiver}" 0 2>> "${RUN_ROOT}/logs/message.log")" || \
    die "could not query the receiver's pre-send balance"
  before_sender_balance="$(query_balance "${sender}" 0 2>> "${RUN_ROOT}/logs/message.log")" || \
    die "could not query the sender's pre-send balance"
  before_sequence="$(query_sequence "${sender}" 0 2>> "${RUN_ROOT}/logs/message.log")" || \
    die "could not query the sender's pre-send sequence"
  memo="isolated-consensus-rehearsal-${SOURCE_HEAD}-${BASE_PORT}"

  output="$("${BINARY}" tx bank send "${sender}" "${receiver}" "${SEND_AMOUNT}${DENOM}" \
    --from sender \
    --home "${RUN_ROOT}/coordinator" \
    --keyring-backend test \
    --chain-id "${CHAIN_ID}" \
    --node "$(rpc_url 0)" \
    --fees "${TX_FEE}${DENOM}" \
    --gas "${TX_GAS}" \
    --note "${memo}" \
    --broadcast-mode sync \
    --yes \
    --output json \
    2>> "${RUN_ROOT}/logs/message.log")" || die "MsgSend command failed"
  code="$(printf '%s' "${output}" | jq -er '(.code // 0) | tonumber')"
  hash="$(printf '%s' "${output}" | jq -er '.txhash | ascii_upcase | select(test("^[0-9A-F]{64}$"))')"
  [ "${code}" -eq 0 ] || die "MsgSend CheckTx failed with code ${code}"

  tx_query="$(wait_for_tx "${hash}" 0)"
  encoded="$(printf '%s' "${tx_query}" | jq -er '.result.tx | select(test("^[A-Za-z0-9+/]+={0,2}$"))')"
  decoded="$("${BINARY}" tx decode "${encoded}" --home "${RUN_ROOT}/coordinator" --output json)" || \
    die "could not decode committed transaction bytes"
  printf '%s\n' "${decoded}" > "${RUN_ROOT}/reports/msgsend-decoded.json"
  printf '%s' "${decoded}" | jq -e \
    --arg sender "${sender}" \
    --arg receiver "${receiver}" \
    --arg amount "${SEND_AMOUNT}" \
    --arg fee "${TX_FEE}" \
    --arg gas "${TX_GAS}" \
    --arg sequence "${before_sequence}" \
    --arg memo "${memo}" '
      (.body.messages | length) == 1 and
      .body.messages[0]["@type"] == "/cosmos.bank.v1beta1.MsgSend" and
      .body.messages[0].from_address == $sender and
      .body.messages[0].to_address == $receiver and
      .body.messages[0].amount == [{denom:"uzrn", amount:$amount}] and
      .body.memo == $memo and
      (.auth_info.signer_infos | length) == 1 and
      ((.auth_info.signer_infos[0].sequence // "0") == $sequence) and
      .auth_info.fee.amount == [{denom:"uzrn", amount:$fee}] and
      .auth_info.fee.gas_limit == $gas and
      (.signatures | length) == 1 and
      (.signatures[0] | type == "string" and length > 0 and
        test("^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$"))
    ' >/dev/null || die "decoded committed MsgSend semantics differ from the request"

  for index in 1 2 3; do
    wait_for_tx "${hash}" "${index}" >/dev/null
  done
  tx_height="$(printf '%s' "${tx_query}" | jq -er '.result.height | select(test("^[1-9][0-9]*$"))')"
  for index in 0 1 2 3; do
    wait_for_height "${index}" "$((tx_height + 2))" "canonical post-state height after MsgSend"
  done
  expected_receiver_balance=$((before_balance + SEND_AMOUNT))
  expected_sender_balance=$((before_sender_balance - SEND_AMOUNT - TX_FEE))
  after_sequence=$((before_sequence + 1))
  for index in 0 1 2 3; do
    node_receiver_balance="$(query_balance "${receiver}" "${index}" 2>> "${RUN_ROOT}/logs/message.log")" || \
      die "could not query node ${index}'s receiver balance"
    node_sender_balance="$(query_balance "${sender}" "${index}" 2>> "${RUN_ROOT}/logs/message.log")" || \
      die "could not query node ${index}'s sender balance"
    node_sequence="$(query_sequence "${sender}" "${index}" 2>> "${RUN_ROOT}/logs/message.log")" || \
      die "could not query node ${index}'s sender sequence"
    [ "${node_receiver_balance}" -eq "${expected_receiver_balance}" ] || \
      die "node ${index} receiver balance ${node_receiver_balance}, expected ${expected_receiver_balance}"
    [ "${node_sender_balance}" -eq "${expected_sender_balance}" ] || \
      die "node ${index} sender balance ${node_sender_balance}, expected ${expected_sender_balance}"
    [ "${node_sequence}" -eq "${after_sequence}" ] || \
      die "node ${index} sender sequence ${node_sequence}, expected ${before_sequence} + 1"
  done

  report="${RUN_ROOT}/reports/msgsend-all-nodes.json"
  "${VERIFY_BINARY}" \
    -rpcs "$(rpc_url 0),$(rpc_url 1),$(rpc_url 2),$(rpc_url 3)" \
    -expect-validators 4 \
    -expect-equal-power \
    -tx "${hash}" \
    > "${report}" || die "MsgSend commit/proof differs across nodes"

  replay_request="$(jq -cn --arg tx "${encoded}" \
    '{jsonrpc:"2.0", id:1, method:"broadcast_tx_sync", params:{tx:$tx}}')"
  replay_response="$(curl --noproxy '*' -fsS --connect-timeout 1 --max-time 10 \
    -H 'Content-Type: application/json' --data-binary "${replay_request}" "$(rpc_url 1)")" || \
    die "could not replay exact transaction bytes to node 1"
  printf '%s\n' "${replay_response}" > "${RUN_ROOT}/reports/msgsend-replay.json"
  printf '%s' "${replay_response}" | jq -e --arg hash "${hash}" '
    (
      .error == null and
      (.result != null) and
      ((.result.hash | ascii_upcase) == $hash) and
      ((.result.codespace // "") == "sdk") and
      ((.result.code | tonumber) == 32)
    ) or (
      (.result == null) and
      (.error != null) and
      ([.error.message // "", .error.data // ""] | map(tostring) | join(" ") |
        ascii_downcase | contains("tx already exists in cache"))
    )
  ' >/dev/null || \
    die "exact committed transaction replay was neither nonzero CheckTx nor the specific tx-cache rejection"

  replay_sequence="$(query_sequence "${sender}" 1 2>> "${RUN_ROOT}/logs/message.log")" || \
    die "could not query the sender sequence after replay"
  [ "${replay_sequence}" -eq "${after_sequence}" ] || \
    die "sender sequence advanced again after replay (${after_sequence} -> ${replay_sequence})"

  run_commit_test "after-msgsend" 3
  ok "MsgSend ${hash}: inclusion proof matched all nodes; replay rejected; sequence advanced once"
}

verify_offline_custom_staking_census() {
  local first_height second_height peer_height chain_id
  local abci_0 abci_1 app_height_0 app_height_1 app_hash_0 app_hash_1
  local copy_0 copy_1 report_0 report_1 evidence

  info "halting finality before copying two application databases for the offline census"
  stop_node 3
  stop_node 2
  sleep 3

  first_height="$(current_height 0)"
  peer_height="$(current_height 1)"
  [ "${first_height}" -gt 0 ] || die "node 0 has no committed height before the census copy"
  [ "${peer_height}" -eq "${first_height}" ] || \
    die "census source nodes disagree on halted height (${first_height} vs ${peer_height})"
  sleep 3
  second_height="$(current_height 0)"
  [ "${second_height}" -eq "${first_height}" ] || \
    die "chain advanced after census quorum was removed (${first_height} -> ${second_height})"
  [ "$(current_height 1)" -eq "${first_height}" ] || \
    die "peer chain advanced after census quorum was removed"

  chain_id="$(rpc_get 0 /status | jq -er '.result.node_info.network')"
  [ "${chain_id}" = "${CHAIN_ID}" ] || die "census source reported unexpected chain ${chain_id}"
  [ "$(rpc_get 1 /status | jq -er '.result.node_info.network')" = "${CHAIN_ID}" ] || \
    die "peer census source reported the wrong chain"

  abci_0="$(rpc_get 0 /abci_info)"
  abci_1="$(rpc_get 1 /abci_info)"
  printf '%s\n' "${abci_0}" > "${RUN_ROOT}/reports/census-node0-abci-info.json.raw"
  printf '%s\n' "${abci_1}" > "${RUN_ROOT}/reports/census-node1-abci-info.json.raw"
  app_height_0="$(printf '%s' "${abci_0}" | jq -er '.result.response.last_block_height | select(test("^[1-9][0-9]*$"))')"
  app_height_1="$(printf '%s' "${abci_1}" | jq -er '.result.response.last_block_height | select(test("^[1-9][0-9]*$"))')"
  app_hash_0="$(normalize_app_hash "$(printf '%s' "${abci_0}" | jq -er '.result.response.last_block_app_hash')")" || \
    die "node 0 ABCI AppHash is neither 32-byte hexadecimal nor base64"
  app_hash_1="$(normalize_app_hash "$(printf '%s' "${abci_1}" | jq -er '.result.response.last_block_app_hash')")" || \
    die "node 1 ABCI AppHash is neither 32-byte hexadecimal nor base64"
  [ "${app_height_0}" = "${first_height}" ] || \
    die "node 0 ABCI height ${app_height_0} differs from halted consensus height ${first_height}"
  [ "${app_height_1}" = "${app_height_0}" ] || \
    die "census source ABCI heights differ (${app_height_0} vs ${app_height_1})"
  [ "${app_hash_1}" = "${app_hash_0}" ] || \
    die "census source ABCI AppHashes differ"

  evidence="${RUN_ROOT}/reports/custom-staking-census-source-evidence.json"
  jq -cnS \
    --arg chain_id "${chain_id}" \
    --arg height "${app_height_0}" \
    --arg app_hash "${app_hash_0}" \
    --arg node0_id "${NODE_ID[0]}" \
    --arg node1_id "${NODE_ID[1]}" \
    --arg node0_abci_sha256 "$(sha256_file "${RUN_ROOT}/reports/census-node0-abci-info.json.raw")" \
    --arg node1_abci_sha256 "$(sha256_file "${RUN_ROOT}/reports/census-node1-abci-info.json.raw")" \
    '{schema:"zerone.local-census-source-evidence/v1",chain_id:$chain_id,height:$height,app_hash:$app_hash,sources:[{node_id:$node0_id,abci_info_sha256:$node0_abci_sha256},{node_id:$node1_id,abci_info_sha256:$node1_abci_sha256}]}' \
    > "${evidence}"

  stop_node 0
  stop_node 1
  for index in 0 1 2 3; do
    [ -z "${NODE_PID[$index]:-}" ] || die "node ${index} still has an owned PID before database copy"
  done
  [ -z "$(find "${NODE_HOME[0]}" "${NODE_HOME[1]}" -type l -print -quit)" ] || \
    die "census source home unexpectedly contains a symlink"

  copy_0="${RUN_ROOT}/census-node0-copy"
  copy_1="${RUN_ROOT}/census-node1-copy"
  cp -R "${NODE_HOME[0]}" "${copy_0}"
  cp -R "${NODE_HOME[1]}" "${copy_1}"
  report_0="${RUN_ROOT}/reports/custom-staking-census.json"
  report_1="${RUN_ROOT}/reports/custom-staking-census-node1.json"

  "${CENSUS_BINARY}" \
    --home "${copy_0}" \
    --backend goleveldb \
    --chain-id "${chain_id}" \
    --expected-height "${app_height_0}" \
    --expected-app-hash "${app_hash_0}" \
    --source-commit "${SOURCE_FULL_HEAD}" \
    --copied-db \
    --output "${report_0}" || die "node 0 custom-staking census did not pass"
  "${CENSUS_BINARY}" \
    --home "${copy_1}" \
    --backend goleveldb \
    --chain-id "${chain_id}" \
    --expected-height "${app_height_1}" \
    --expected-app-hash "${app_hash_1}" \
    --source-commit "${SOURCE_FULL_HEAD}" \
    --copied-db \
    --output "${report_1}" || die "node 1 custom-staking census did not pass"
  cmp "${report_0}" "${report_1}" || die "independent node census reports differ"
  jq -e \
    --arg chain_id "${chain_id}" \
    --arg height "${app_height_0}" \
    --arg app_hash "${app_hash_0}" \
    --arg source_commit "${SOURCE_FULL_HEAD}" '
      .schema == "zerone/custom-staking-census/v1" and
      .result == "PASS" and
      .evidence == {
        chain_id: $chain_id,
        height: $height,
        app_hash: $app_hash,
        source_commit: $source_commit
      } and
      .census.claimant_root_complete == true and
      .census.delta_uzrn == "0" and
      .census.findings == [] and
      (.report_sha256 | test("^[0-9a-f]{64}$"))
    ' "${report_0}" >/dev/null || die "custom-staking census report failed its launch-rehearsal contract"
  ok "offline custom-staking census matched two stopped copies at height ${app_height_0}, AppHash ${app_hash_0}"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --keep) KEEP=1 ;;
    --allow-dirty) ALLOW_DIRTY=1 ;;
    -h|--help) usage; SUCCESS=1; exit 0 ;;
    *) usage >&2; die "unknown argument $1" ;;
  esac
  shift
done

case "${KEEP}" in
  0|1) ;;
  *) die "KEEP_REHEARSAL must be 0 or 1" ;;
esac

for dependency in go git jq curl lsof ps awk sed shasum rg python3; do
  command -v "${dependency}" >/dev/null 2>&1 || die "${dependency} is required"
done
[ -f "${ROOT}/go.mod" ] || die "repository root not found at ${ROOT}"
[ -f "${ROOT}/tests/multivalidator/multivalidator_test.go" ] || die "multivalidator test source is missing"
rg -q 'ZERONE_TEST_RPC_ADDR' "${ROOT}/tests/multivalidator/multivalidator_test.go" || \
  die "TestBlockSignatures does not support an isolated RPC endpoint"
rg -q 'validatorSet\.VerifyCommit\(' "${ROOT}/tests/multivalidator/multivalidator_test.go" || \
  die "TestBlockSignatures is not the cryptographic commit verifier"

SOURCE_FULL_HEAD="$(git -C "${ROOT}" rev-parse HEAD)"
SOURCE_HEAD="$(git -C "${ROOT}" rev-parse --short=12 HEAD)"
WORKTREE_STATUS="$(git -C "${ROOT}" status --porcelain=v1 --untracked-files=all --ignore-submodules=none)"
if [ -n "${WORKTREE_STATUS}" ]; then
  if [ "${ALLOW_DIRTY}" -ne 1 ]; then
    printf '%s\n' "${WORKTREE_STATUS}" >&2
    die "worktree is dirty; commit/stash all tracked and untracked changes before a definitive run"
  fi
  SOURCE_LABEL="${SOURCE_HEAD}-dirty-NON-FINAL"
  printf 'WARNING: --allow-dirty selected; this rehearsal is NON-FINAL and cannot establish exact-SHA provenance.\n' >&2
else
  SOURCE_LABEL="${SOURCE_HEAD}"
fi

RUN_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/zerone-consensus-rehearsal.XXXXXX")"
touch "${RUN_ROOT}/.zerone-consensus-rehearsal-owned"
mkdir -p "${RUN_ROOT}/bin" "${RUN_ROOT}/logs" "${RUN_ROOT}/reports" \
  "${RUN_ROOT}/go-cache" "${RUN_ROOT}/go-tmp" "${RUN_ROOT}/cli"
BINARY="${RUN_ROOT}/bin/zeroned"
VERIFY_BINARY="${RUN_ROOT}/bin/local-consensus-verify"
CENSUS_BINARY="${RUN_ROOT}/bin/custom-staking-census"
git -C "${ROOT}" status --short > "${RUN_ROOT}/reports/source-status.txt"
allocate_ports

info "isolated state: ${RUN_ROOT}"
info "building checkout ${SOURCE_HEAD} into temporary binaries"
VERSION="rehearsal-${SOURCE_LABEL}"
LDFLAGS="-s -w -X github.com/cosmos/cosmos-sdk/version.Name=zerone -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned -X github.com/cosmos/cosmos-sdk/version.Version=${VERSION} -X github.com/cosmos/cosmos-sdk/version.Commit=${SOURCE_FULL_HEAD}"
(
  cd "${ROOT}"
  GOCACHE="${RUN_ROOT}/go-cache" GOTMPDIR="${RUN_ROOT}/go-tmp" \
    go build -trimpath -buildvcs=true -ldflags "${LDFLAGS}" -o "${BINARY}" ./cmd/zeroned
  GOCACHE="${RUN_ROOT}/go-cache" GOTMPDIR="${RUN_ROOT}/go-tmp" \
    go build -trimpath -buildvcs=true -o "${VERIFY_BINARY}" ./tools/local-consensus-verify
  GOCACHE="${RUN_ROOT}/go-cache" GOTMPDIR="${RUN_ROOT}/go-tmp" \
    go build -trimpath -buildvcs=true -o "${CENSUS_BINARY}" ./tools/custom-staking-census
) > "${RUN_ROOT}/logs/build.log" 2>&1 || die "fresh binary build failed"

BINARY_SHA="$(sha256_file "${BINARY}")"
EXPECTED_COMET="$(cd "${ROOT}" && go list -m -f '{{.Version}}' github.com/cometbft/cometbft)"
go version -m "${BINARY}" > "${RUN_ROOT}/reports/binary-build-info.txt"
"${BINARY}" version --long --home "${RUN_ROOT}/cli" \
  > "${RUN_ROOT}/reports/zeroned-version.txt" 2>&1 || die "fresh binary version query failed"
VERSION_REPORT_COMMIT="$(sed -n 's/^commit: //p' "${RUN_ROOT}/reports/zeroned-version.txt")"
VERSION_REPORT_VERSION="$(sed -n 's/^version: //p' "${RUN_ROOT}/reports/zeroned-version.txt")"
[ "${VERSION_REPORT_COMMIT}" = "${SOURCE_FULL_HEAD}" ] || \
  die "built binary explicit commit ${VERSION_REPORT_COMMIT:-missing} != checkout ${SOURCE_FULL_HEAD}"
[ "${VERSION_REPORT_VERSION}" = "${VERSION}" ] || \
  die "built binary version ${VERSION_REPORT_VERSION:-missing} != requested ${VERSION}"
VCS_REVISION="$(sed -n 's/.*vcs\.revision=//p' "${RUN_ROOT}/reports/binary-build-info.txt")"
VCS_MODIFIED="$(sed -n 's/.*vcs\.modified=//p' "${RUN_ROOT}/reports/binary-build-info.txt")"
if [ -n "${VCS_REVISION}" ] || [ -n "${VCS_MODIFIED}" ]; then
  [ "${VCS_REVISION}" = "${SOURCE_FULL_HEAD}" ] || \
    die "built binary VCS revision ${VCS_REVISION:-missing} != checkout ${SOURCE_FULL_HEAD}"
  if [ "${ALLOW_DIRTY}" -eq 0 ]; then
    [ "${VCS_MODIFIED}" = "false" ] || \
      die "definitive binary reports vcs.modified=${VCS_MODIFIED:-missing}, expected false"
  else
    [ "${VCS_MODIFIED}" = "true" ] || \
      die "dirty development binary unexpectedly reports vcs.modified=${VCS_MODIFIED:-missing}"
  fi
else
  printf '%s\n' \
    "Go omitted optional VCS settings; exact commit is bound by the clean checkout, build command, and verified SDK commit field." \
    > "${RUN_ROOT}/reports/go-vcs-stamp-note.txt"
fi
grep -F "github.com/cometbft/cometbft" "${RUN_ROOT}/reports/binary-build-info.txt" | \
  grep -F "${EXPECTED_COMET}" >/dev/null || die "built binary does not embed CometBFT ${EXPECTED_COMET}"
ok "fresh zeroned sha256=${BINARY_SHA}, CometBFT=${EXPECTED_COMET}"

COORDINATOR="${RUN_ROOT}/coordinator"
"${BINARY}" init coordinator --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" \
  --home "${COORDINATOR}" > "${RUN_ROOT}/logs/init.log" 2>&1 || die "coordinator init failed"

for account in sender receiver val0 val1 val2 val3; do
  "${BINARY}" keys add "${account}" --keyring-backend test --home "${COORDINATOR}" \
    >/dev/null 2>> "${RUN_ROOT}/logs/init.log" || die "could not create ${account} test key"
done
SENDER_ADDR="$("${BINARY}" keys show sender -a --keyring-backend test --home "${COORDINATOR}")"
RECEIVER_ADDR="$("${BINARY}" keys show receiver -a --keyring-backend test --home "${COORDINATOR}")"
"${BINARY}" add-genesis-account "${SENDER_ADDR}" "${SENDER_BALANCE}${DENOM}" --home "${COORDINATOR}" \
  >> "${RUN_ROOT}/logs/init.log" 2>&1 || die "could not fund sender"
"${BINARY}" add-genesis-account "${RECEIVER_ADDR}" "${RECEIVER_BALANCE}${DENOM}" --home "${COORDINATOR}" \
  >> "${RUN_ROOT}/logs/init.log" 2>&1 || die "could not fund receiver"

for index in 0 1 2 3; do
  VALIDATOR_ADDR="$("${BINARY}" keys show "val${index}" -a --keyring-backend test --home "${COORDINATOR}")"
  "${BINARY}" add-genesis-account "${VALIDATOR_ADDR}" "${VALIDATOR_BALANCE}${DENOM}" --home "${COORDINATOR}" \
    >> "${RUN_ROOT}/logs/init.log" 2>&1 || die "could not fund validator ${index}"
done

mkdir -p "${COORDINATOR}/config/gentx"
for index in 0 1 2 3; do
  NODE_HOME[index]="${RUN_ROOT}/node${index}"
  NODE_LOG[index]="${RUN_ROOT}/logs/node${index}.log"
  NODE_PID[index]=""
  "${BINARY}" init "val${index}" --chain-id "${CHAIN_ID}" --default-denom "${DENOM}" \
    --home "${NODE_HOME[$index]}" > /dev/null 2>> "${RUN_ROOT}/logs/init.log" || die "node ${index} init failed"
  cp "${COORDINATOR}/config/genesis.json" "${NODE_HOME[$index]}/config/genesis.json"
  cp -R "${COORDINATOR}/keyring-test" "${NODE_HOME[$index]}/"
  "${BINARY}" genesis gentx "val${index}" "${VALIDATOR_STAKE}${DENOM}" \
    --chain-id "${CHAIN_ID}" \
    --keyring-backend test \
    --home "${NODE_HOME[$index]}" \
    --moniker "val${index}" \
    --commission-rate 0.10 \
    --commission-max-rate 0.20 \
    --commission-max-change-rate 0.01 \
    --output-document "${COORDINATOR}/config/gentx/gentx-val${index}.json" \
    >/dev/null 2>> "${RUN_ROOT}/logs/init.log" || die "validator ${index} gentx failed"
done

"${BINARY}" genesis collect-gentxs --home "${COORDINATOR}" \
  >> "${RUN_ROOT}/logs/init.log" 2>&1 || die "gentx collection failed"
"${BINARY}" genesis validate --home "${COORDINATOR}" \
  >> "${RUN_ROOT}/logs/init.log" 2>&1 || die "generated genesis is invalid"

for index in 0 1 2 3; do
  cp "${COORDINATOR}/config/genesis.json" "${NODE_HOME[$index]}/config/genesis.json"
  NODE_ID[index]="$("${BINARY}" comet show-node-id --home "${NODE_HOME[$index]}" 2>/dev/null || \
    "${BINARY}" tendermint show-node-id --home "${NODE_HOME[$index]}")"
  [ -n "${NODE_ID[$index]}" ] || die "node ${index} has no CometBFT node ID"
  configure_node "${index}"
done
ok "fresh genesis contains four validators with identical ${VALIDATOR_STAKE}${DENOM} stake"

info "starting four validators on loopback ports ${BASE_PORT}-$((BASE_PORT + 7))"
for index in 0 1 2 3; do
  start_node "${index}"
done
for index in 0 1 2 3; do
  wait_for_height "${index}" 4 "initial height 4"
  assert_loopback_listeners "${index}"
done
verify_phase four-up 0 1 2 3

info "stopping validator 3; 75% voting power must continue finality"
stop_node 3
wait_for_advance 3 one-validator-down 0 1 2
verify_phase one-down 0 1 2

info "stopping validator 2; 50% voting power must halt finality"
stop_node 2
sleep 3
HALT_HEIGHT_0="$(current_height 0)"
HALT_HEIGHT_1="$(current_height 1)"
[ "${HALT_HEIGHT_0}" -gt 0 ] && [ "${HALT_HEIGHT_1}" -gt 0 ] || die "remaining validators became unreachable"
[ "${HALT_HEIGHT_0}" -eq "${HALT_HEIGHT_1}" ] || \
  die "two surviving validators froze at different heights (${HALT_HEIGHT_0} vs ${HALT_HEIGHT_1})"
HALT_BLOCK_0="$(rpc_get 0 "/block?height=${HALT_HEIGHT_0}")"
HALT_BLOCK_1="$(rpc_get 1 "/block?height=${HALT_HEIGHT_1}")"
HALT_BLOCK_ID_0="$(printf '%s' "${HALT_BLOCK_0}" | jq -er '.result.block_id.hash | select(test("^[0-9A-Fa-f]{64}$"))')"
HALT_BLOCK_ID_1="$(printf '%s' "${HALT_BLOCK_1}" | jq -er '.result.block_id.hash | select(test("^[0-9A-Fa-f]{64}$"))')"
HALT_APP_HASH_0="$(printf '%s' "${HALT_BLOCK_0}" | jq -er '.result.block.header.app_hash')"
HALT_APP_HASH_1="$(printf '%s' "${HALT_BLOCK_1}" | jq -er '.result.block.header.app_hash')"
[ "${HALT_BLOCK_ID_0}" = "${HALT_BLOCK_ID_1}" ] || \
  die "surviving validators disagree on frozen block ID at height ${HALT_HEIGHT_0}"
[ "${HALT_APP_HASH_0}" = "${HALT_APP_HASH_1}" ] || \
  die "surviving validators disagree on frozen app hash at height ${HALT_HEIGHT_0}"
[ "${HALT_HEIGHT_0}" -gt 1 ] || die "frozen chain has no preceding canonical commit"
"${VERIFY_BINARY}" \
  -rpcs "$(rpc_url 0),$(rpc_url 1)" \
  -height "$((HALT_HEIGHT_0 - 1))" \
  -expect-validators 4 \
  -expect-equal-power \
  > "${RUN_ROOT}/reports/two-down-frozen.json" || \
  die "surviving validators disagree on the frozen block/app hash"
sleep 4
is_owned_pid 0 || die "node 0 exited during the halt check"
is_owned_pid 1 || die "node 1 exited during the halt check"
[ "$(current_height 0)" -eq "${HALT_HEIGHT_0}" ] || die "node 0 finalized a block with only 50% voting power"
[ "$(current_height 1)" -eq "${HALT_HEIGHT_1}" ] || die "node 1 finalized a block with only 50% voting power"
ok "two-validator partition stayed live but finalized no block at height ${HALT_HEIGHT_0}"

info "restarting validator 2; 75% voting power must recover finality"
start_node 2
wait_for_advance 3 three-up-recovery 0 1 2
verify_phase three-up-recovery 0 1 2

info "restarting validator 3 and waiting for all four nodes to converge"
start_node 3
RECOVERY_TARGET=$(( $(current_height 0) + 3 ))
for index in 0 1 2 3; do
  wait_for_height "${index}" "${RECOVERY_TARGET}" "four-node recovery"
done
verify_phase four-up-recovered 0 1 2 3

info "broadcasting, proving, and replay-testing one signed local MsgSend"
verify_message_flow

verify_offline_custom_staking_census

[ "$(git -C "${ROOT}" rev-parse HEAD)" = "${SOURCE_FULL_HEAD}" ] || \
  die "checkout HEAD changed during the rehearsal"
if [ "${ALLOW_DIRTY}" -eq 0 ] && \
   [ -n "$(git -C "${ROOT}" status --porcelain=v1 --untracked-files=all --ignore-submodules=none)" ]; then
  die "worktree changed during the definitive rehearsal"
fi
SUCCESS=1
printf '\nPASS isolated consensus rehearsal\n'
if [ "${ALLOW_DIRTY}" -eq 1 ]; then
  printf '  status: NON-FINAL dirty development run\n'
fi
printf '  checkout: %s\n' "${SOURCE_FULL_HEAD}"
printf '  zeroned sha256: %s\n' "${BINARY_SHA}"
printf '  CometBFT: %s\n' "${EXPECTED_COMET}"
if [ "${KEEP}" = "1" ]; then
  printf '  retained state: %s\n' "${RUN_ROOT}"
else
  printf '  state: temporary (removed after node shutdown)\n'
fi
