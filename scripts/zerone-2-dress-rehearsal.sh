#!/usr/bin/env bash
# Full local zerone-2 rehearsal using public fixtures only.
#
# Boots audited genesis and the separately funded operator bootstrap, then a
# fresh non-validator joins through a non-validator edge. Exercises a capped
# OPERATIONS-funded user pilot, same-home restart and the existing continuation
# export/import diagnostic. NOT a production restore proof or OPEN authority.
# Public fixture keys only; all sockets are isolated loopback. Safe receipts
# survive outside Git; temporary homes, keys and raw transactions do not.
# shellcheck disable=SC2016 # Single-quoted jq programs intentionally use $names.

set -euo pipefail
export LC_ALL=C
export GOPROXY=off
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${ROOT}/build/zeroned}"
CHAIN_ID="zerone-2"
# No endpoint overrides: this harness must never reach a real chain.
BASE_PORT=$((40000 + (RANDOM % 1800) * 8))
RPC="tcp://127.0.0.1:$((BASE_PORT + 1))"
RPC_HTTP="http://127.0.0.1:$((BASE_PORT + 1))"
REST="http://127.0.0.1:$((BASE_PORT + 2))"
PILOT_CAP=10000000
PILOT_FUND=2000000
TX_FEE=200000
SEND_AMOUNT=12345
TOTAL_SUPPLY="13555000000"
CUSTOM_ESCROW="111000000"
CUSTOM_STAKING_MODULE="zrn1ehtmkw3djuxxprsr8ueknnamwk3jvkpmlzfepn"
DRILL_MNEMONIC="now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-dress.XXXXXX")"
RECEIPTS="$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-dress-receipts.XXXXXX")"
NODE_PID=("" "" "")
NODE_ID=("" "" "")
FAILURE="unexpected command failure"
SCRIPT_SHA="$(shasum -a 256 "${BASH_SOURCE[0]}" | cut -d ' ' -f 1)"

info() { printf '  -> %s\n' "$*"; }
ok() { printf '  OK %s\n' "$*"; }
fail() {
  FAILURE="$*"
  jq -n --arg failure "${FAILURE}" '{failure:$failure}' > "${RECEIPTS}/failure.json"
  printf 'zerone-2 dress rehearsal: FAIL: %s\n' "$*" >&2
  exit 1
}

owned_pid() {
  local pid="${NODE_PID[$1]:-}"
  [ -n "${pid}" ] && kill -0 "${pid}" 2>/dev/null &&
    [ "$(ps -p "${pid}" -o ppid= | tr -d ' ')" = "$$" ]
}

stop_node() {
  local index="${1:-0}"
  if owned_pid "${index}"; then
    kill -TERM "${NODE_PID[$index]}" 2>/dev/null || true
    for _ in $(seq 1 40); do
      owned_pid "${index}" || break
      sleep 0.25
    done
    if owned_pid "${index}"; then
      kill -KILL "${NODE_PID[$index]}" 2>/dev/null || true
    fi
  fi
  if [ -n "${NODE_PID[$index]:-}" ]; then
    wait "${NODE_PID[$index]}" 2>/dev/null || true
  fi
  NODE_PID[index]=""
}

cleanup() {
  local status=$?
  trap - EXIT
  stop_node 2
  stop_node 1
  stop_node 0
  if [ -f "${RECEIPTS}/failure.json" ]; then
    FAILURE="$(jq -r '.failure' "${RECEIPTS}/failure.json")"
  fi
  jq -n --argjson exit_code "${status}" --arg failure "${FAILURE}" \
    --arg source "$(git -C "${ROOT}" rev-parse HEAD)" \
    --arg script_sha256 "${SCRIPT_SHA}" \
    '{scope:"LOCAL FIXTURE ONLY; uncommitted candidate; not release/OPEN or production restore proof",
      exit_code:$exit_code, failure:(if $exit_code == 0 then null else $failure end),
      source_commit:$source, script_sha256:$script_sha256}' > "${RECEIPTS}/result.json"
  # Only this invocation's mktemp home; no caller-selected state or foreign PIDs.
  if [ -d "${TMP}" ] && [ ! -L "${TMP}" ]; then
    chmod -R u+w "${TMP}"
    rm -rf "${TMP}"
  fi
  printf 'Public fixture receipts: %s\n' "${RECEIPTS}"
  exit "${status}"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

rpc_url() { printf 'http://127.0.0.1:%s' "$((BASE_PORT + $1 * 4 + 1))"; }
rpc_get() { curl --noproxy '*' -fsS --max-time 3 "$(rpc_url "$1")$2"; }
port_is_free() { ! lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1; }

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
  sed_in_place 's|^allow_duplicate_ip = .*|allow_duplicate_ip = true|' "${config}"
  sed_in_place 's|^addr_book_strict = .*|addr_book_strict = false|' "${config}"
  sed_in_place 's|^pex = .*|pex = false|' "${config}"
  sed_in_place 's|^seeds = .*|seeds = ""|' "${config}"
  sed_in_place 's|^timeout_commit = .*|timeout_commit = "500ms"|' "${config}"
  sed_in_place 's|^prometheus = .*|prometheus = false|' "${config}"
  sed_in_place 's|^pprof_laddr = .*|pprof_laddr = ""|' "${config}"
  sed_in_place 's|^minimum-gas-prices = .*|minimum-gas-prices = "1uzrn"|' "${app}"
  sed_in_place 's|^pruning = .*|pruning = "nothing"|' "${app}"
  sed_in_place 's|^max-txs = -1|max-txs = 5000|' "${app}"
  sed_in_place 's|^iavl-disable-fastnode = false|iavl-disable-fastnode = true|' "${app}"
}

start_node() {
  local home="$1" log="$2" index="${3:-0}" peers="${4:-}" port api=false
  [ -z "${NODE_PID[$index]:-}" ] || fail "node ${index} already recorded"
  for port in $(seq "$((BASE_PORT + index * 4))" "$((BASE_PORT + index * 4 + 3))"); do
    port_is_free "${port}" || fail "port ${port} is already owned; refusing to start"
  done
  [ "${index}" -ne 0 ] || api=true
  "${BINARY}" start --home "${home}" --minimum-gas-prices 1uzrn \
    --rpc.laddr "tcp://127.0.0.1:$((BASE_PORT + index * 4 + 1))" \
    --rpc.pprof_laddr "" --rpc.unsafe=false \
    --p2p.laddr "tcp://127.0.0.1:$((BASE_PORT + index * 4))" \
    --p2p.external-address "127.0.0.1:$((BASE_PORT + index * 4))" \
    --p2p.persistent_peers "${peers}" --p2p.pex=false \
    --api.enable="${api}" --api.address "tcp://127.0.0.1:$((BASE_PORT + index * 4 + 2))" \
    --grpc.enable="${api}" --grpc.address "127.0.0.1:$((BASE_PORT + index * 4 + 3))" \
    --grpc-web.enable=false --log_level error \
    >> "${log}" 2>&1 &
  NODE_PID[index]=$!
  sleep 0.25
  owned_pid "${index}" || fail "node ${index} exited at startup"
}

assert_listeners() {
  local index="$1" listener port seen=0
  while IFS= read -r listener; do
    case "${listener}" in
      n127.0.0.1:*)
        port="${listener##*:}"
        [ "${port}" -ge "$((BASE_PORT + index * 4))" ] &&
          [ "${port}" -le "$((BASE_PORT + index * 4 + 3))" ] || fail "unexpected node port"
        seen=$((seen + 1)) ;;
      n*) fail "non-loopback node listener" ;;
    esac
  done < <(lsof -nP -a -p "${NODE_PID[$index]}" -iTCP -sTCP:LISTEN -Fn)
  [ "${seen}" -ge 2 ] || fail "node ${index} does not own its listeners"
}

wait_for_height() {
  local target="$1" index="${2:-0}"
  for _ in $(seq 1 240); do
    owned_pid "${index}" || fail "node ${index} exited before height ${target}"
    local status height chain catching
    status="$(rpc_get "${index}" /status 2>/dev/null || true)"
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
  local hash="$1" index="${2:-0}" result
  for _ in $(seq 1 160); do
    result="$(rpc_get "${index}" "/tx?hash=0x${hash}&prove=true" 2>/dev/null || true)"
    if jq -e --arg hash "${hash}" '
      .error == null and .result.hash == $hash and
      .result.tx_result.code == 0 and (.result.height | test("^[1-9][0-9]*$"))
      and .result.proof != null
    ' <<<"${result}" >/dev/null 2>&1; then
      printf '%s\n' "${result}"
      return 0
    fi
    sleep 0.25
  done
  fail "transaction ${hash} did not commit successfully on node ${index}"
}

broadcast() {
  local output code hash
  output="$(${BINARY} "$@" --node "${TX_RPC:-${RPC}}" --chain-id "${CHAIN_ID}" \
    --keyring-backend test --broadcast-mode sync -o json --yes \
    2>> "${TMP}/tx.log")" || fail "transaction command failed"
  code="$(jq -er '.code | numbers' <<<"${output}")"
  hash="$(jq -r '.txhash // ""' <<<"${output}")"
  [ "${code}" = "0" ] || fail "CheckTx rejected transaction with code ${code}"
  [[ "${hash}" =~ ^[0-9A-F]{64}$ ]] || fail "transaction returned invalid hash"
  wait_for_tx "${hash}" >/dev/null
  printf '%s\n' "${hash}"
}

query_balance() {
  "${BINARY}" query bank balance "$1" uzrn --home "${HOME_A}" \
    --node "$(rpc_url "${2:-0}")" -o json \
    | jq -er '.balance.amount | select(test("^[0-9]+$"))' || fail "bank balance query/encoding failed"
}

query_sequence() {
  "${BINARY}" query auth account "$1" --home "${HOME_A}" \
    --node "$(rpc_url "${2:-0}")" -o json | jq -er --arg address "$1" '
      [.account | .. | objects | select(.address? == $address)
        | (.sequence // "0") | tostring] | unique | select(length == 1) | .[0]
        | select(test("^[0-9]+$"))' || fail "account sequence query/encoding failed"
}

assert_dark() {
  local module
  assert_supply "${TOTAL_SUPPLY}"
  for module in knowledge vesting_rewards claiming_pot; do
    rest_json "/zerone/${module}/v1/params" > "${TMP}/${module}-params.json"
    jq --arg module "${module}" --slurpfile genesis "${ARTIFACTS}/genesis.json" '
      # Proto REST lowerCamelCase/int64 strings versus genesis encoding/json.
      def wire: walk(if type == "object" then
        with_entries(.key |= gsub("_(?<letter>[a-z])"; .letter | ascii_upcase))
        elif type == "number" then tostring else . end);
      (.params | wire) as $actual | ($genesis[0].app_state[$module].params | wire |
        # Deprecated uint64 has omitempty in genesis.pb.go, but REST emits zero.
        if $module == "vesting_rewards" then .governanceActivationHeight //= "0" else . end) as $expected |
      [($expected + $actual | keys[]) as $key |
       select($expected[$key] != $actual[$key]) |
       {key:$key, expected:$expected[$key], actual:$actual[$key]}]
    ' "${TMP}/${module}-params.json" > "${RECEIPTS}/${module}-params-diff.json"
    jq -e '. == []' "${RECEIPTS}/${module}-params-diff.json" >/dev/null || fail "${module} dark params changed"
  done
  rest_json /zerone/substrate_bridge/v1/adapters | jq -e '.adapters == []' >/dev/null ||
    fail "substrate adapters are not empty"
  rest_json /zerone/claiming_pot/v1/pots | jq -e '
    .pots == [] and (.pagination.nextKey == null or .pagination.nextKey == "")
  ' >/dev/null || fail "claiming pots are not empty"
  rest_json /zerone/vesting_rewards/v1/schedules/active | jq -e '
    .schedules == [] and (.total | tonumber) == 0
  ' >/dev/null || fail "reward schedules are not empty"
  rest_json /zerone/vesting_rewards/v1/supply_coupling | jq -e --arg supply "${TOTAL_SUPPLY}" '
    .totalMinted == "0" and .currentSupply == $supply and .couplingEnabled == false
  ' >/dev/null || fail "reward mint ledger or supply changed"
}

canonical_genesis() {
  jq -S 'if has("consensus") then
    {genesis_time, chain_id, initial_height, app_hash, app_state,
     consensus_params:.consensus.params, validators:.consensus.validators}
    else {genesis_time, chain_id, initial_height, app_hash, app_state, consensus_params, validators}
    end | .initial_height |= tostring | .app_hash //= "" | .validators //= [] |
    .consensus_params |= walk(if type == "number" then tostring else . end) |
    .validators |= walk(if type == "number" then tostring else . end)' "$1"
}

init_follower() {
  local index="$1" home="$2" upstream="$3"
  "${BINARY}" init "zerone-2-fresh-${index}" --chain-id "${CHAIN_ID}" --home "${home}" >/dev/null 2>&1
  # Fetch only genesis, never a validator data directory, signer key or snapshot.
  rpc_get "${upstream}" /genesis | jq -e '.result.genesis | select(.chain_id == "zerone-2")' \
    > "${home}/config/genesis.json"
  canonical_genesis "${home}/config/genesis.json" > "${TMP}/genesis-${index}.json"
  cmp "${TMP}/genesis-expected.json" "${TMP}/genesis-${index}.json" || fail "fetched genesis differs"
  [ "$(find "${home}/data" -type f ! -name priv_validator_state.json | wc -l | tr -d ' ')" = 0 ] ||
    fail "fresh non-validator already has consensus database files"
  jq -e '.height == "0"' "${home}/data/priv_validator_state.json" >/dev/null || fail "fresh signer state is not zero"
  NODE_ID[index]="$("${BINARY}" comet show-node-id --home "${home}")"
  configure_home "${home}"
}

assert_edge_topology() {
  local index expected actual ready
  for _ in $(seq 1 120); do
    ready=true
    for index in 0 1 2; do
      if [ "${index}" -eq 1 ]; then
        expected="$(jq -cn --arg a "${NODE_ID[0]}" --arg b "${NODE_ID[2]}" '[$a,$b]|sort')"
      else
        expected="$(jq -cn --arg a "${NODE_ID[1]}" '[$a]')"
      fi
      actual="$(rpc_get "${index}" /net_info | jq -c '[.result.peers[].node_info.id]|sort')"
      [ "${actual}" = "${expected}" ] || ready=false
    done
    [ "${ready}" != true ] || return 0
    sleep 0.25
  done
  fail "topology is not validator <-> non-validator edge <-> fresh user node"
}

verify_checkpoint() {
  local label="$1" index home pubkey
  shift
  for index in 0 1 2; do
    assert_listeners "${index}"
    [ "$(rpc_get "${index}" /status | jq -r '.result.node_info.id')" = "${NODE_ID[$index]}" ] ||
      fail "RPC identity differs from owned node"
    rpc_get "${index}" /genesis | jq -e '.result.genesis' > "${TMP}/rpc-genesis.json"
    canonical_genesis "${TMP}/rpc-genesis.json" > "${TMP}/rpc-genesis-canonical.json"
    cmp "${TMP}/genesis-expected.json" "${TMP}/rpc-genesis-canonical.json" || fail "node genesis differs"
  done
  assert_edge_topology
  for home in "${HOME_EDGE}" "${HOME_USER}"; do
    pubkey="$(jq -r '.pub_key.value' "${home}/config/priv_validator_key.json")"
    rpc_get 0 /validators | jq -e --arg key "${pubkey}" '
      .result.total == "1" and all(.result.validators[]; .pub_key.value != $key)
    ' >/dev/null || fail "fresh follower is a validator"
    jq -e '.height == "0"' "${home}/data/priv_validator_state.json" >/dev/null || fail "follower signed consensus"
  done
  # Reuse strict commit/header/AppHash and transaction inclusion verification.
  # No arbitrary bank-absence proof is requested or claimed here.
  "${TMP}/consensus-verify" -rpcs "$(rpc_url 0),$(rpc_url 1),$(rpc_url 2)" \
    -expect-chain-id "${CHAIN_ID}" -expect-validators 1 "$@" \
    > "${RECEIPTS}/${label}.json" 2> "${RECEIPTS}/${label}-verify.log" || fail "${label} consensus/proof verification failed"
}

verify_send() {
  local hash="$1" sender="$2" receiver="$3" amount="$4" sequence="$5" result encoded digest
  result="$(wait_for_tx "${hash}" 2)"
  encoded="$(jq -er '.result.tx' <<<"${result}")"
  digest="$(printf '%s' "${encoded}" | base64 --decode | shasum -a 256 | cut -d ' ' -f 1 | tr '[:lower:]' '[:upper:]')"
  [ "${digest}" = "${hash}" ] || fail "committed tx bytes differ from hash"
  "${BINARY}" tx decode "${encoded}" --home "${HOME_USER}" -o json | jq -e \
    --arg sender "${sender}" --arg receiver "${receiver}" --arg amount "${amount}" \
    --arg sequence "${sequence}" --arg fee "${TX_FEE}" '
    (.body.messages | length) == 1 and
    .body.messages[0]["@type"] == "/cosmos.bank.v1beta1.MsgSend" and
    .body.messages[0].from_address == $sender and .body.messages[0].to_address == $receiver and
    .body.messages[0].amount == [{denom:"uzrn",amount:$amount}] and
    (.auth_info.signer_infos | length) == 1 and
    .auth_info.signer_infos[0].sequence == $sequence and
    .auth_info.fee.amount == [{denom:"uzrn",amount:$fee}] and .auth_info.fee.gas_limit == "200000" and
    (.signatures | length) == 1 and (.signatures[0] | type == "string" and length > 0)
  ' >/dev/null || fail "committed MsgSend semantics differ"
  # Private scratch only: never retain seeds, signatures or raw signed txs in receipts.
  printf '%s' "${encoded}" > "${TMP}/last-send.base64"
}

assert_pilot_balances() {
  local index
  for index in 0 1 2; do
    [ "$(query_balance "${OPS_ADDRESS}" "${index}")" = "$((OPS_BEFORE - PILOT_FUND - TX_FEE))" ] || fail "OPERATIONS debit differs"
    [ "$(query_sequence "${OPS_ADDRESS}" "${index}")" = "$((OPS_SEQUENCE + 1))" ] || fail "OPERATIONS sequence differs"
    [ "$(query_balance "${PILOT_ADDRESS}" "${index}")" = "$((PILOT_FUND - 2 * TX_FEE - SEND_AMOUNT))" ] || fail "pilot balance differs"
    [ "$(query_sequence "${PILOT_ADDRESS}" "${index}")" = 2 ] || fail "pilot sequence differs"
    [ "$(query_balance "${RECEIVER_ADDRESS}" "${index}")" = "${SEND_AMOUNT}" ] || fail "recipient balance differs"
  done
  assert_dark
}

run_pilot() {
  local fund_hash onboard_hash send_hash response hash replay height index did
  info "simulating post-OPEN user pilot with a capped fixture OPERATIONS debit"
  # Ceremony source uses account 202 for OPERATIONS, never validator account 201.
  printf '%s\n' "${DRILL_MNEMONIC}" | "${BINARY}" keys add operations --recover --account 202 \
    --keyring-backend test --home "${HOME_A}" >/dev/null 2>&1
  OPS_ADDRESS="$("${BINARY}" keys show operations -a --keyring-backend test --home "${HOME_A}")"
  [ "${OPS_ADDRESS}" = "$(jq -r '.operations.account_address' "${ARTIFACTS}/network-manifest.json")" ] || fail "wrong OPERATIONS fixture account"
  [ "${OPS_ADDRESS}" != "${VALIDATOR_ADDRESS}" ] || fail "pilot source is validator"
  for name in pilot recipient; do
    "${BINARY}" keys add "${name}" --keyring-backend test --home "${HOME_USER}" >/dev/null 2>&1
  done
  PILOT_ADDRESS="$("${BINARY}" keys show pilot -a --keyring-backend test --home "${HOME_USER}")"
  RECEIVER_ADDRESS="$("${BINARY}" keys show recipient -a --keyring-backend test --home "${HOME_USER}")"
  [ "$(query_balance "${PILOT_ADDRESS}" 2)" = 0 ] && [ "$(query_balance "${RECEIVER_ADDRESS}" 2)" = 0 ] || fail "fresh user already funded"
  OPS_BEFORE="$(query_balance "${OPS_ADDRESS}")"
  OPS_SEQUENCE="$(query_sequence "${OPS_ADDRESS}")"
  [ "${OPS_BEFORE}" = 2222000000 ] || fail "OPERATIONS float was used by bootstrap"
  [ "$((PILOT_FUND + TX_FEE))" -le "${PILOT_CAP}" ] || fail "pilot budget exceeds 10 ZRN"
  TX_RPC="$(rpc_url 2)"
  fund_hash="$(broadcast tx bank send "${OPS_ADDRESS}" "${PILOT_ADDRESS}" "${PILOT_FUND}uzrn" \
    --from operations --home "${HOME_A}" --gas 200000 --fees "${TX_FEE}uzrn")"
  verify_send "${fund_hash}" "${OPS_ADDRESS}" "${PILOT_ADDRESS}" "${PILOT_FUND}" "${OPS_SEQUENCE}"
  [ "$(query_balance "${PILOT_ADDRESS}" 2)" = "${PILOT_FUND}" ] || fail "funding credit differs"
  [ "$(query_sequence "${PILOT_ADDRESS}" 2)" = 0 ] || fail "fresh pilot sequence is not zero"
  onboard_hash="$(broadcast tx zerone_auth onboard human --from pilot \
    --identity-out "${TMP}/pilot-identity.ed25519.json" --home "${HOME_USER}" \
    --gas 200000 --fees "${TX_FEE}uzrn")"
  wait_for_tx "${onboard_hash}" 2 >/dev/null
  did="$("${BINARY}" query zerone_auth account "${PILOT_ADDRESS}" --home "${HOME_USER}" \
    --node "${TX_RPC}" -o json | jq -r '.account.did // .did // ""')"
  [[ "${did}" =~ ^did:zrn:[0-9a-f]{64}$ ]] || fail "pilot DID registration failed"
  [ "$(query_sequence "${PILOT_ADDRESS}" 2)" = 1 ] || fail "onboard sequence differs"
  [ "$(query_balance "${PILOT_ADDRESS}" 2)" = "$((PILOT_FUND - TX_FEE))" ] || fail "onboard fee differs"
  send_hash="$(broadcast tx bank send "${PILOT_ADDRESS}" "${RECEIVER_ADDRESS}" "${SEND_AMOUNT}uzrn" \
    --from pilot --home "${HOME_USER}" --gas 200000 --fees "${TX_FEE}uzrn")"
  for index in 0 1 2; do wait_for_tx "${send_hash}" "${index}" >/dev/null; done
  verify_send "${send_hash}" "${PILOT_ADDRESS}" "${RECEIVER_ADDRESS}" "${SEND_AMOUNT}" 1
  height="$(rpc_get 0 /status | jq -r '.result.sync_info.latest_block_height')"
  for index in 0 1 2; do wait_for_height "$((height + 2))" "${index}"; done
  verify_checkpoint pilot-send -tx "${send_hash}"
  assert_pilot_balances

  replay="$(jq -cn --rawfile tx "${TMP}/last-send.base64" '{jsonrpc:"2.0",id:1,method:"broadcast_tx_sync",params:{tx:$tx}}')"
  response="$(curl --noproxy '*' -fsS --max-time 10 -H 'Content-Type: application/json' \
    --data-binary "${replay}" "${TX_RPC}")"
  jq -e --arg hash "${send_hash}" '
    (.error == null and .result.hash == $hash and .result.codespace == "sdk" and .result.code == 32) or
    (.result == null and .error != null and
      ([.error.message // "", .error.data // ""] | join(" ") | ascii_downcase | contains("tx already exists in cache")))
  ' <<<"${response}" >/dev/null || fail "exact replay was not rejected for sequence/cache"
  jq '{result:(.result | if . == null then null else {hash,code,codespace} end),
       cache_rejection:(.error != null)}' <<<"${response}" > "${RECEIPTS}/replay.json"

  # Still sign and broadcast through the user's own node; no mocked fee assertion.
  response="$("${BINARY}" tx bank send "${PILOT_ADDRESS}" "${RECEIVER_ADDRESS}" 1uzrn \
    --from pilot --home "${HOME_USER}" --keyring-backend test --chain-id "${CHAIN_ID}" \
    --node "${TX_RPC}" --gas 200000 --fees 1uzrn --broadcast-mode sync -o json --yes \
    2>> "${TMP}/tx.log")" || fail "underfee command did not return CheckTx evidence"
  jq -e '.code == 13 and .codespace == "sdk" and (.txhash | test("^[0-9A-F]{64}$"))' \
    <<<"${response}" >/dev/null || fail "underfee transaction was not rejected with sdk/13"
  hash="$(jq -r '.txhash' <<<"${response}")"
  jq '{txhash,code,codespace}' <<<"${response}" > "${RECEIPTS}/insufficient-fee.json"
  height="$(rpc_get 0 /status | jq -r '.result.sync_info.latest_block_height')"
  for index in 0 1 2; do
    wait_for_height "$((height + 2))" "${index}"
    # Comet returns HTTP 500 for the expected JSON-RPC tx-not-found error.
    response="$(curl --noproxy '*' -sS --max-time 3 "$(rpc_url "${index}")/tx?hash=0x${hash}")" ||
      fail "underfee absence query transport failed"
    jq -e --arg hash "${hash}" '
      .result == null and .error.code == -32603 and
      (.error.data | contains("not found") and (ascii_upcase | contains($hash)))
    ' <<<"${response}" >/dev/null || fail "underfee tx-not-found response invalid"
    jq -n --arg hash "${hash}" --argjson node "${index}" \
      '{txhash:$hash,node:$node,query_code:-32603,not_found:true}' \
      > "${RECEIPTS}/underfee-not-found-${index}.json"
  done
  assert_pilot_balances
  jq -n --arg fund_hash "${fund_hash}" --arg onboard_hash "${onboard_hash}" --arg send_hash "${send_hash}" \
    --argjson debit "$((OPS_BEFORE - $(query_balance "${OPS_ADDRESS}")))" \
    --argjson cap "${PILOT_CAP}" --argjson amount "${SEND_AMOUNT}" \
    '{fixture_only:true, funding_tx:$fund_hash, onboard_tx:$onboard_hash, send_tx:$send_hash,
      operations_debit_uzrn:$debit, cap_uzrn:$cap, recipient_credit_uzrn:$amount,
      pilot_sequence:2, total_supply_uzrn:"13555000000", dark_invariants:true}' > "${RECEIPTS}/pilot.json"
  [ "$((OPS_BEFORE - $(query_balance "${OPS_ADDRESS}")))" -le "${PILOT_CAP}" ] || fail "observed pilot debit exceeds cap"
  unset TX_RPC
  ok "capped pilot: exact fee/balance/sequence arithmetic, DID, gossip, replay and underfee rejection"
}

[ -x "${BINARY}" ] || fail "binary missing at ${BINARY}"
command -v curl >/dev/null || fail "curl is required"
command -v jq >/dev/null || fail "jq is required"
command -v xxd >/dev/null || fail "xxd is required"
command -v lsof >/dev/null || fail "lsof is required"
for port in $(seq "${BASE_PORT}" "$((BASE_PORT + 11))"); do
  port_is_free "${port}" || fail "port ${port} is already owned; refusing rehearsal"
done
cd "${ROOT}"
go version -m "${BINARY}" | grep -E 'go1\.|[[:space:]](path|mod)[[:space:]]|vcs\.|GOOS=|GOARCH=' \
  > "${RECEIPTS}/binary-build-info.txt"
shasum -a 256 "${BINARY}" > "${RECEIPTS}/binary.sha256"

ARTIFACTS="${TMP}/artifacts"
HOME_A="${TMP}/home-a"
HOME_B="${TMP}/home-b"
HOME_EDGE="${TMP}/edge"
HOME_USER="${TMP}/user"

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
go build -o "${TMP}/consensus-verify" "${ROOT}/tools/local-consensus-verify"
canonical_genesis "${ARTIFACTS}/genesis.json" > "${TMP}/genesis-expected.json"
shasum -a 256 "${ARTIFACTS}/genesis.json" "${TMP}/genesis-expected.json" > "${RECEIPTS}/genesis.sha256"
"${TMP}/ceremony-inject" drill-consensus-key zerone-2-public-drill \
  "${HOME_A}/config/priv_validator_key.json" >/dev/null 2>&1
printf '%s\n' "${DRILL_MNEMONIC}" \
  | "${BINARY}" keys add validator --recover --account 201 \
      --keyring-backend test --home "${HOME_A}" >/dev/null 2>&1
VALIDATOR_ADDRESS="$(${BINARY} keys show validator -a --keyring-backend test --home "${HOME_A}")"
configure_home "${HOME_A}"
NODE_ID[0]="$("${BINARY}" comet show-node-id --home "${HOME_A}")"

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
ok "separately funded private operator bootstrap txs committed: ${ONBOARD_HASH}, ${REGISTER_HASH}"
assert_dark

info "joining fresh non-validator edge, then fresh user node only through that edge"
init_follower 1 "${HOME_EDGE}" 0
start_node "${HOME_EDGE}" "${TMP}/edge.log" 1 "${NODE_ID[0]}@127.0.0.1:${BASE_PORT}"
wait_for_height 5 1
init_follower 2 "${HOME_USER}" 1
start_node "${HOME_USER}" "${TMP}/user.log" 2 "${NODE_ID[1]}@127.0.0.1:$((BASE_PORT + 4))"
JOIN_HEIGHT="$(rpc_get 0 /status | jq -r '.result.sync_info.latest_block_height')"
for index in 0 1 2; do wait_for_height "$((JOIN_HEIGHT + 2))" "${index}"; done
verify_checkpoint fresh-edge-join

info "restarting the same user home and replaying blocks missed while offline"
USER_NODE_KEY_SHA="$(shasum -a 256 "${HOME_USER}/config/node_key.json" | cut -d ' ' -f 1)"
RESTART_HEIGHT="$(rpc_get 2 /status | jq -r '.result.sync_info.latest_block_height')"
stop_node 2
wait_for_height "$((RESTART_HEIGHT + 4))"
start_node "${HOME_USER}" "${TMP}/user.log" 2 "${NODE_ID[1]}@127.0.0.1:$((BASE_PORT + 4))"
wait_for_height "$((RESTART_HEIGHT + 6))" 2
[ "$(shasum -a 256 "${HOME_USER}/config/node_key.json" | cut -d ' ' -f 1)" = "${USER_NODE_KEY_SHA}" ] || fail "restart changed user node identity"
verify_checkpoint same-home-restart
# The retained common header must still match after restart, not only the new tip.
verify_checkpoint retained-join-height -height "$(jq -r '.height' "${RECEIPTS}/fresh-edge-join.json")"
run_pilot
stop_node 2
stop_node 1

info "restarting the validator from the same database (existing diagnostic)"
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

assert_dark
EXPORT_SHA="$(shasum -a 256 "${EXPORT}" | cut -d ' ' -f 1)"
jq -n --arg sha256 "${EXPORT_SHA}" --arg initial_height "${EXPORTED_INITIAL_HEIGHT}" \
  --arg target_height "${IMPORTED_TARGET_HEIGHT}" \
  '{diagnostic:"continuation export/import only; NOT stopped-directory production restore proof",
    export_sha256:$sha256, initial_height:$initial_height, target_height:$target_height,
    custom_escrow_uzrn:"111000000", total_supply_uzrn:"13555000000", dark_invariants:true}' \
  > "${RECEIPTS}/continuation-import.json"
ok "restart and ten-block stopped-height export/import diagnostic passed (not production restore proof)"
ok "supply stayed exactly ${TOTAL_SUPPLY}uzrn; export sha256 ${EXPORT_SHA}"
printf 'zerone-2 dress rehearsal: PASS\n'
