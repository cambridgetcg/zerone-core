#!/usr/bin/env bash
# Deterministic local proof of the zerone-1 terminal checkpoint convention.
#
# Public fixture keys produce state F=3, an empty final anchor block A=4 whose
# header commits state F, and an SDK consensus halt before H=5. The SDK halt
# leaves the daemon and RPC server alive. Comet stages H in its block store
# before FinalizeBlock fails: status therefore reports H, /commit A is canonical,
# /commit H is the subjective tip seen-commit (canonical=false because no H+1),
# and ABCI remains at A. The rehearsal proves the split on independent live
# homes, captures raw evidence, fences both, then builds a fresh-key archive
# from an explicit stopped-observer database allowlist.
set -euo pipefail
export LC_ALL=C
export GOPROXY=off
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BINARY="${BINARY:-${ROOT}/build/zeroned}"
CHAIN_ID="zerone-halt-checkpoint-rehearsal-1"
CHECKPOINT_STATE_HEIGHT=3
FINAL_COMMITTED_HEIGHT=4
HALT_TRIGGER_HEIGHT=5
PUBLIC_MNEMONIC="now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct"

TMP="$(mktemp -d "${TMPDIR:-/tmp}/zerone-halt-checkpoint.XXXXXX")"
SIGNER_HOME="${TMP}/signer"
LIVE_OBSERVER_HOME="${TMP}/live-observer"
ARCHIVE_KEY_HOME="${TMP}/archive-keys"
ARCHIVE_HOME="${TMP}/archive"
SIGNER_SNAPSHOT="${TMP}/signer-checkpoint.json"
OBSERVER_SNAPSHOT="${TMP}/observer-checkpoint.json"
NODE_PID=""
SIGNER_PID=""
LIVE_OBSERVER_PID=""
POST_ANCHOR_APP_HASH=""

# Keep parallel CI jobs away from the conventional Cosmos ports.
PORT_BASE=$((43000 + ($$ % 1000) * 10))
SIGNER_RPC_PORT="${PORT_BASE}"
SIGNER_P2P_PORT=$((PORT_BASE + 1))
SIGNER_REST_PORT=$((PORT_BASE + 2))
SIGNER_GRPC_PORT=$((PORT_BASE + 3))
OBSERVER_RPC_PORT=$((PORT_BASE + 4))
OBSERVER_P2P_PORT=$((PORT_BASE + 5))
OBSERVER_REST_PORT=$((PORT_BASE + 6))
OBSERVER_GRPC_PORT=$((PORT_BASE + 7))
SIGNER_RPC="http://127.0.0.1:${SIGNER_RPC_PORT}"
SIGNER_REST="http://127.0.0.1:${SIGNER_REST_PORT}"
OBSERVER_RPC="http://127.0.0.1:${OBSERVER_RPC_PORT}"
OBSERVER_REST="http://127.0.0.1:${OBSERVER_REST_PORT}"
RPC="${SIGNER_RPC}"
REST="${SIGNER_REST}"

fail() {
  printf 'halt checkpoint rehearsal: FAIL: %s\n' "$*" >&2
  for log in "${TMP}/signer.log" "${TMP}/live-observer.log" "${TMP}/archive.log" \
    "${TMP}/archive-rollback.log"; do
    if [ -f "${log}" ]; then
      printf '%s\n' "--- ${log} ---" >&2
      tail -80 "${log}" >&2 || true
    fi
  done
  exit 1
}

stop_process() {
  local pid="$1"
  if [ -n "${pid}" ] && kill -0 "${pid}" 2>/dev/null; then
    kill -TERM "${pid}" 2>/dev/null || true
    for _ in $(seq 1 40); do
      kill -0 "${pid}" 2>/dev/null || break
      sleep 0.25
    done
    if kill -0 "${pid}" 2>/dev/null; then
      kill -KILL "${pid}" 2>/dev/null || true
    fi
    wait "${pid}" 2>/dev/null || true
  fi
}

stop_node() {
  stop_process "${NODE_PID}"
  NODE_PID=""
}

cleanup() {
  stop_node
  stop_process "${SIGNER_PID}"
  stop_process "${LIVE_OBSERVER_PID}"
  chmod -R u+w "${TMP}" 2>/dev/null || true
  rm -rf "${TMP}"
}
trap cleanup EXIT INT TERM

toml_set() {
  local section="$1" key="$2" value="$3" file="$4" tmp
  tmp="${file}.rehearsal-tmp.$$"
  if ! awk -v target="[${section}]" -v wanted="${key}" -v replacement="${value}" '
    BEGIN { active = 0; changed = 0 }
    /^\[/ { active = ($0 == target) }
    {
      if (active && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=") {
        print wanted " = " replacement
        changed++
        next
      }
      print
    }
    END { if (changed != 1) exit 42 }
  ' "${file}" > "${tmp}"; then
    rm -f "${tmp}"
    fail "could not set ${section}.${key} in ${file}"
  fi
  mv "${tmp}" "${file}"
}

toml_set_root() {
  local key="$1" value="$2" file="$3" tmp
  tmp="${file}.rehearsal-tmp.$$"
  if ! awk -v wanted="${key}" -v replacement="${value}" '
    BEGIN { before_section = 1; changed = 0 }
    /^\[/ { before_section = 0 }
    {
      if (before_section && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=") {
        print wanted " = " replacement
        changed++
        next
      }
      print
    }
    END { if (changed != 1) exit 42 }
  ' "${file}" > "${tmp}"; then
    rm -f "${tmp}"
    fail "could not set root ${key} in ${file}"
  fi
  mv "${tmp}" "${file}"
}

configure_home() {
  local home="$1" rpc_port="$2" p2p_port="$3" rest_port="$4" grpc_port="$5" peers="$6"
  local config="${home}/config/config.toml" app="${home}/config/app.toml"
  toml_set rpc laddr "\"tcp://127.0.0.1:${rpc_port}\"" "${config}"
  toml_set rpc unsafe false "${config}"
  toml_set p2p laddr "\"tcp://127.0.0.1:${p2p_port}\"" "${config}"
  toml_set p2p pex false "${config}"
  toml_set p2p persistent_peers "\"${peers}\"" "${config}"
  toml_set p2p addr_book_strict false "${config}"
  toml_set p2p allow_duplicate_ip true "${config}"
  toml_set consensus timeout_propose '"250ms"' "${config}"
  toml_set consensus timeout_commit '"250ms"' "${config}"
  toml_set instrumentation prometheus false "${config}"
  toml_set instrumentation prometheus_listen_addr '""' "${config}"
  toml_set mempool broadcast false "${config}"
  toml_set mempool wal_dir '""' "${config}"
  toml_set mempool size 0 "${config}"
  toml_set mempool max_txs_bytes 0 "${config}"

  toml_set_root minimum-gas-prices '"0.025uzrn"' "${app}"
  toml_set_root pruning '"nothing"' "${app}"
  toml_set api enable true "${app}"
  toml_set api address "\"tcp://127.0.0.1:${rest_port}\"" "${app}"
  toml_set api enabled-unsafe-cors false "${app}"
  toml_set grpc enable true "${app}"
  toml_set grpc address "\"127.0.0.1:${grpc_port}\"" "${app}"
  toml_set grpc-web enable false "${app}"
  toml_set mempool max-txs -1 "${app}"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

normalize_app_hash() {
  local raw="$1" decoded
  if [[ "${raw}" =~ ^[0-9A-Fa-f]{64}$ ]]; then
    printf '%s' "${raw}" | tr '[:lower:]' '[:upper:]'
    return
  fi
  if ! decoded="$(printf '%s' "${raw}" | base64 --decode 2>/dev/null | \
    od -An -tx1 | tr -d '[:space:]' | tr '[:lower:]' '[:upper:]')"; then
    fail "ABCI returned an invalid base64 app hash"
  fi
  [[ "${decoded}" =~ ^[0-9A-F]{64}$ ]] || fail "ABCI returned an app hash with the wrong decoded length"
  printf '%s' "${decoded}"
}

wait_for_observer() {
  local status height abci app_hash trigger_block anchor_commit
  for _ in $(seq 1 120); do
    if ! kill -0 "${NODE_PID}" 2>/dev/null; then
      fail "observer exited before serving the checkpoint"
    fi
    status="$(curl --noproxy '*' -fsS --max-time 1 "${RPC}/status" 2>/dev/null || true)"
    if [ -n "${status}" ]; then
      height="$(jq -r '.result.sync_info.latest_block_height // "0"' <<<"${status}")"
      if [ "${height}" = "${FINAL_COMMITTED_HEIGHT}" ]; then
        jq -e --arg chain "${CHAIN_ID}" '
          .result.node_info.network == $chain and
          .result.sync_info.catching_up == true
        ' <<<"${status}" >/dev/null || \
          fail "fresh-key archive did not expose the expected block-sync status"
        abci="$(curl --noproxy '*' -fsS --max-time 1 "${RPC}/abci_info" 2>/dev/null || true)"
        jq -e --arg height "${FINAL_COMMITTED_HEIGHT}" \
          '.result.response.last_block_height == $height' <<<"${abci}" >/dev/null || \
          fail "observer application is not at final applied height ${FINAL_COMMITTED_HEIGHT}"
        app_hash="$(normalize_app_hash "$(jq -r '.result.response.last_block_app_hash // ""' <<<"${abci}")")"
        [ "${app_hash}" = "${POST_ANCHOR_APP_HASH}" ] || \
          fail "sanitized observer app hash ${app_hash} != captured post-anchor hash ${POST_ANCHOR_APP_HASH}"
        trigger_block="$(curl --noproxy '*' -sS --max-time 1 \
          "${RPC}/block?height=${HALT_TRIGGER_HEIGHT}" 2>/dev/null || true)"
        if jq -e --arg height "${HALT_TRIGGER_HEIGHT}" \
          '.result.block.header.height == $height' <<<"${trigger_block}" >/dev/null 2>&1; then
          fail "sanitized observer still exposes staged halt-trigger block ${HALT_TRIGGER_HEIGHT}"
        fi
        anchor_commit="$(curl --noproxy '*' -fsS --max-time 1 \
          "${RPC}/commit?height=${FINAL_COMMITTED_HEIGHT}" 2>/dev/null || true)"
        jq -e --arg height "${FINAL_COMMITTED_HEIGHT}" '
          .result.canonical == false and
          .result.signed_header.header.height == $height and
          .result.signed_header.commit.height == $height
        ' <<<"${anchor_commit}" >/dev/null || \
          fail "sanitized A/A archive did not expose A as its canonical=false current tip"
        return
      fi
    fi
    sleep 0.25
  done
  fail "observer did not serve final committed block ${FINAL_COMMITTED_HEIGHT}"
}

wait_for_source_halt() {
  local role="$1" log="$2" status height="unavailable" halt_seen="no"
  for _ in $(seq 1 240); do
    kill -0 "${NODE_PID}" 2>/dev/null || \
      fail "${role} exited before exposing the halted checkpoint"
    status="$(curl --noproxy '*' -fsS --max-time 1 "${RPC}/status" 2>/dev/null || true)"
    if [ -n "${status}" ]; then
      height="$(jq -r '.result.sync_info.latest_block_height // "0"' <<<"${status}")"
      if grep -q "halt per configuration height ${HALT_TRIGGER_HEIGHT}" "${log}"; then
        halt_seen="yes"
      fi
      if [ "${height}" = "${HALT_TRIGGER_HEIGHT}" ] && [ "${halt_seen}" = "yes" ]; then
        jq -e --arg chain "${CHAIN_ID}" '
          .result.node_info.network == $chain and
          .result.sync_info.catching_up == false
        ' <<<"${status}" >/dev/null || fail "${role} status identity is wrong"
        return
      fi
    fi
    sleep 0.25
  done
  fail "${role} did not expose staged trigger ${HALT_TRIGGER_HEIGHT} and its SDK halt (last RPC height=${height}, halt log=${halt_seen})"
}

assert_halt_boundary() {
  local status height anchor trigger anchor_hash trigger_hash anchor_app_hash trigger_app_hash
  local anchor_commit trigger_commit abci applied_height applied_app_hash trigger_results

  kill -0 "${NODE_PID}" 2>/dev/null || fail "signer daemon exited during halted-boundary proof"
  status="$(curl --noproxy '*' -fsS --max-time 1 "${RPC}/status" 2>/dev/null || true)"
  [ -n "${status}" ] || fail "signer RPC disappeared during halted-boundary proof"
  height="$(jq -r '.result.sync_info.latest_block_height // "0"' <<<"${status}")"
  [ "${height}" = "${HALT_TRIGGER_HEIGHT}" ] || \
    fail "block-store tip changed during halt: latest height ${height}"

  anchor="$(curl --noproxy '*' -fsS --max-time 1 \
    "${RPC}/block?height=${FINAL_COMMITTED_HEIGHT}" 2>/dev/null || true)"
  trigger="$(curl --noproxy '*' -fsS --max-time 1 \
    "${RPC}/block?height=${HALT_TRIGGER_HEIGHT}" 2>/dev/null || true)"
  jq -e --arg height "${FINAL_COMMITTED_HEIGHT}" \
    '.result.block.header.height == $height and ((.result.block.data.txs // []) | length == 0)' \
    <<<"${anchor}" >/dev/null || fail "final anchor block is missing or non-empty"
  jq -e --arg height "${HALT_TRIGGER_HEIGHT}" \
    '.result.block.header.height == $height and ((.result.block.data.txs // []) | length == 0)' \
    <<<"${trigger}" >/dev/null || fail "staged halt-trigger block is missing or non-empty"

  anchor_hash="$(jq -r '.result.block_id.hash' <<<"${anchor}")"
  trigger_hash="$(jq -r '.result.block_id.hash' <<<"${trigger}")"
  anchor_app_hash="$(jq -r '.result.block.header.app_hash' <<<"${anchor}")"
  trigger_app_hash="$(jq -r '.result.block.header.app_hash' <<<"${trigger}")"
  [[ "${anchor_hash}" =~ ^[0-9A-Fa-f]{64}$ ]] || fail "anchor block hash is invalid"
  [[ "${trigger_hash}" =~ ^[0-9A-Fa-f]{64}$ ]] || fail "trigger block hash is invalid"
  [[ "${anchor_app_hash}" =~ ^[0-9A-Fa-f]{64}$ ]] || fail "anchor app hash is invalid"
  [[ "${trigger_app_hash}" =~ ^[0-9A-Fa-f]{64}$ ]] || fail "trigger app hash is invalid"
  [ "$(jq -r '.result.block.header.last_block_id.hash' <<<"${trigger}")" = "${anchor_hash}" ] || \
    fail "staged trigger header does not link the final anchor block"
  jq -e --arg hash "${trigger_hash}" --arg app_hash "${trigger_app_hash}" '
    .result.sync_info.latest_block_hash == $hash and
    .result.sync_info.latest_app_hash == $app_hash
  ' <<<"${status}" >/dev/null || fail "status does not identify the staged trigger tip"

  anchor_commit="$(curl --noproxy '*' -fsS --max-time 1 \
    "${RPC}/commit?height=${FINAL_COMMITTED_HEIGHT}" 2>/dev/null || true)"
  trigger_commit="$(curl --noproxy '*' -fsS --max-time 1 \
    "${RPC}/commit?height=${HALT_TRIGGER_HEIGHT}" 2>/dev/null || true)"
  jq -e --arg height "${FINAL_COMMITTED_HEIGHT}" --arg hash "${anchor_hash}" '
    .result.canonical == true and
    .result.signed_header.header.height == $height and
    .result.signed_header.commit.height == $height and
    .result.signed_header.commit.block_id.hash == $hash
  ' <<<"${anchor_commit}" >/dev/null || fail "anchor block lacks a canonical commit"
  jq -e --arg height "${HALT_TRIGGER_HEIGHT}" --arg hash "${trigger_hash}" '
    .result.canonical == false and
    .result.signed_header.header.height == $height and
    .result.signed_header.commit.height == $height and
    .result.signed_header.commit.block_id.hash == $hash
  ' <<<"${trigger_commit}" >/dev/null || fail "halt-trigger tip did not expose the expected canonical=false seen commit"

  abci="$(curl --noproxy '*' -fsS --max-time 1 "${RPC}/abci_info" 2>/dev/null || true)"
  applied_height="$(jq -r '.result.response.last_block_height // "0"' <<<"${abci}")"
  applied_app_hash="$(normalize_app_hash "$(jq -r '.result.response.last_block_app_hash // ""' <<<"${abci}")")"
  trigger_app_hash="$(normalize_app_hash "${trigger_app_hash}")"
  [ "${applied_height}" = "${FINAL_COMMITTED_HEIGHT}" ] || \
    fail "ABCI advanced or regressed during halt: applied height ${applied_height}"
  [ "${applied_app_hash}" = "${trigger_app_hash}" ] || \
    fail "staged trigger header app hash ${trigger_app_hash} != ABCI post-anchor hash ${applied_app_hash}"
  POST_ANCHOR_APP_HASH="${applied_app_hash}"

  trigger_results="$(curl --noproxy '*' -sS --max-time 1 \
    "${RPC}/block_results?height=${HALT_TRIGGER_HEIGHT}" 2>/dev/null || true)"
  if jq -e --arg height "${HALT_TRIGGER_HEIGHT}" \
    '(.result.height | tostring) == $height' <<<"${trigger_results}" >/dev/null 2>&1; then
    fail "halt-trigger height unexpectedly has application block results"
  fi

  ANCHOR_JSON="${anchor}"
  ANCHOR_APP_HASH="${anchor_app_hash}"
}

prove_stable_halt() {

  # --halt-height stops application execution, not the daemon or block staging.
  # Recheck the exact block-store/application split before fencing the signer.
  for _ in $(seq 1 16); do
    assert_halt_boundary
    sleep 0.25
  done
}

capture_raw_evidence() {
  local label="$1" name
  local dir="${TMP}/evidence-${label}"
  mkdir -p "${dir}"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/status.json" "${RPC}/status"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/block-A.json" \
    "${RPC}/block?height=${FINAL_COMMITTED_HEIGHT}"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/commit-A.json" \
    "${RPC}/commit?height=${FINAL_COMMITTED_HEIGHT}"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/block-H.json" \
    "${RPC}/block?height=${HALT_TRIGGER_HEIGHT}"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/commit-H.json" \
    "${RPC}/commit?height=${HALT_TRIGGER_HEIGHT}"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/abci-info.json" "${RPC}/abci_info"
  curl --noproxy '*' -sS --max-time 3 -o "${dir}/block-results-H-missing.json" \
    "${RPC}/block_results?height=${HALT_TRIGGER_HEIGHT}"
  grep -q "could not find results for height #${HALT_TRIGGER_HEIGHT}" \
    "${dir}/block-results-H-missing.json" || fail "raw evidence lost missing H-results proof"
  : > "${dir}/SHA256SUMS"
  for name in status.json block-A.json commit-A.json block-H.json commit-H.json \
    abci-info.json block-results-H-missing.json; do
    printf '%s  %s\n' "$(sha256_file "${dir}/${name}")" "${name}" >> "${dir}/SHA256SUMS"
  done
}

for command in jq curl go base64 od tr; do
  command -v "${command}" >/dev/null 2>&1 || fail "${command} is required"
done
[ -x "${BINARY}" ] || fail "binary missing at ${BINARY}"

"${BINARY}" init halt-checkpoint-signer --chain-id "${CHAIN_ID}" \
  --default-denom uzrn --home "${SIGNER_HOME}" >/dev/null 2>&1
printf '%s\n' "${PUBLIC_MNEMONIC}" | "${BINARY}" keys add validator --recover \
  --account 303 --keyring-backend test --home "${SIGNER_HOME}" >/dev/null 2>&1
VALIDATOR_ADDRESS="$("${BINARY}" keys show validator -a --keyring-backend test --home "${SIGNER_HOME}")"
"${BINARY}" add-genesis-account "${VALIDATOR_ADDRESS}" 20000000000uzrn \
  --home "${SIGNER_HOME}" >/dev/null
go build -o "${TMP}/ceremony-inject" "${ROOT}/tools/ceremony-inject"
if ! "${TMP}/ceremony-inject" drill-consensus-key zerone-halt-checkpoint-drill-public \
  "${SIGNER_HOME}/config/priv_validator_key.json" >"${TMP}/key-inject.log" 2>&1; then
  sed -n '1,80p' "${TMP}/key-inject.log" >&2
  fail "could not install the public deterministic consensus fixture"
fi
NODE_ID="$("${BINARY}" tendermint show-node-id --home "${SIGNER_HOME}")"
"${BINARY}" genesis gentx validator 1000000000uzrn \
  --chain-id "${CHAIN_ID}" --home "${SIGNER_HOME}" --keyring-backend test \
  --moniker halt-checkpoint-signer --ip 127.0.0.1 --node-id "${NODE_ID}" \
  --commission-rate 0.05 --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 --min-self-delegation 1 >/dev/null 2>&1
"${BINARY}" genesis collect-gentxs --home "${SIGNER_HOME}" >/dev/null 2>&1
"${BINARY}" genesis validate "${SIGNER_HOME}/config/genesis.json" >/dev/null 2>&1

# Start a genuinely independent, fresh-key observer before the signer advances.
# It receives the same public genesis and halt plan, but never receives an
# account keyring or either of the signer's private node identities.
"${BINARY}" init halt-checkpoint-live-observer --chain-id "${CHAIN_ID}" \
  --default-denom uzrn --home "${LIVE_OBSERVER_HOME}" >/dev/null 2>&1
cp "${SIGNER_HOME}/config/genesis.json" "${LIVE_OBSERVER_HOME}/config/genesis.json"
LIVE_OBSERVER_NODE_ID="$("${BINARY}" tendermint show-node-id --home "${LIVE_OBSERVER_HOME}")"
LIVE_OBSERVER_VALIDATOR_PUB="$("${BINARY}" tendermint show-validator \
  --home "${LIVE_OBSERVER_HOME}" | jq -r '.key')"
SIGNER_VALIDATOR_PUB="$("${BINARY}" tendermint show-validator --home "${SIGNER_HOME}" | jq -r '.key')"
[ "${LIVE_OBSERVER_VALIDATOR_PUB}" != "${SIGNER_VALIDATOR_PUB}" ] || \
  fail "live observer unexpectedly shares the signer consensus identity"
[ ! -e "${LIVE_OBSERVER_HOME}/keyring-test" ] || fail "live observer contains an account keyring"

configure_home "${SIGNER_HOME}" "${SIGNER_RPC_PORT}" "${SIGNER_P2P_PORT}" \
  "${SIGNER_REST_PORT}" "${SIGNER_GRPC_PORT}" ""
configure_home "${LIVE_OBSERVER_HOME}" "${OBSERVER_RPC_PORT}" "${OBSERVER_P2P_PORT}" \
  "${OBSERVER_REST_PORT}" "${OBSERVER_GRPC_PORT}" \
  "${NODE_ID}@127.0.0.1:${SIGNER_P2P_PORT}"

"${BINARY}" start --home "${LIVE_OBSERVER_HOME}" \
  --halt-height "${HALT_TRIGGER_HEIGHT}" --minimum-gas-prices 0.025uzrn \
  >"${TMP}/live-observer.log" 2>&1 &
LIVE_OBSERVER_PID=$!

"${BINARY}" start --home "${SIGNER_HOME}" \
  --halt-height "${HALT_TRIGGER_HEIGHT}" --minimum-gas-prices 0.025uzrn \
  >"${TMP}/signer.log" 2>&1 &
SIGNER_PID=$!
NODE_PID="${SIGNER_PID}"
RPC="${SIGNER_RPC}"
REST="${SIGNER_REST}"
wait_for_source_halt signer "${TMP}/signer.log"
prove_stable_halt

GENESIS_SHA="$(sha256_file "${SIGNER_HOME}/config/genesis.json")"
(cd "${ROOT}" && go run ./tools/relaunch-snapshot \
  --rpc "${RPC}" --rest "${REST}" --expected-chain-id "${CHAIN_ID}" \
  --checkpoint-state-height "${CHECKPOINT_STATE_HEIGHT}" \
  --final-committed-height "${FINAL_COMMITTED_HEIGHT}" \
  --halt-trigger-height "${HALT_TRIGGER_HEIGHT}" \
  --declared-genesis-sha256 "${GENESIS_SHA}" --out "${SIGNER_SNAPSHOT}")
capture_raw_evidence signer

NODE_PID="${LIVE_OBSERVER_PID}"
RPC="${OBSERVER_RPC}"
REST="${OBSERVER_REST}"
wait_for_source_halt observer "${TMP}/live-observer.log"
prove_stable_halt
(cd "${ROOT}" && go run ./tools/relaunch-snapshot \
  --rpc "${RPC}" --rest "${REST}" --expected-chain-id "${CHAIN_ID}" \
  --checkpoint-state-height "${CHECKPOINT_STATE_HEIGHT}" \
  --final-committed-height "${FINAL_COMMITTED_HEIGHT}" \
  --halt-trigger-height "${HALT_TRIGGER_HEIGHT}" \
  --declared-genesis-sha256 "${GENESIS_SHA}" --out "${OBSERVER_SNAPSHOT}")
capture_raw_evidence observer

for name in block-A.json commit-A.json block-H.json commit-H.json abci-info.json \
  block-results-H-missing.json; do
  cmp "${TMP}/evidence-signer/${name}" "${TMP}/evidence-observer/${name}" >/dev/null || \
    fail "signer/observer raw evidence differs for ${name}"
done
jq -S 'del(.source.rpc, .source.rest)' "${SIGNER_SNAPSHOT}" > "${TMP}/signer-normalized.json"
jq -S 'del(.source.rpc, .source.rest)' "${OBSERVER_SNAPSHOT}" > "${TMP}/observer-normalized.json"
cmp "${TMP}/signer-normalized.json" "${TMP}/observer-normalized.json" >/dev/null || \
  fail "signer and independent observer v3 snapshots differ"

# Fence both live H/A processes only after v3 and raw evidence capture.
NODE_PID="${SIGNER_PID}"
stop_node
SIGNER_PID=""
NODE_PID="${LIVE_OBSERVER_PID}"
stop_node
LIVE_OBSERVER_PID=""

# Build the serving archive from an explicit public-config/database allowlist.
# No signer or observer home is cloned wholesale, so account keyrings, WAL,
# signer state, and incidental custody files cannot cross the boundary.
"${BINARY}" init halt-checkpoint-archive-keys --chain-id "${CHAIN_ID}" \
  --default-denom uzrn --home "${ARCHIVE_KEY_HOME}" >/dev/null 2>&1
mkdir -p "${ARCHIVE_HOME}/config" "${ARCHIVE_HOME}/data"
for name in genesis.json config.toml app.toml client.toml; do
  cp "${LIVE_OBSERVER_HOME}/config/${name}" "${ARCHIVE_HOME}/config/${name}"
done
# The serving archive is deliberately isolated. Do not inherit the live
# observer's dial path back to the old signer, even though that signer is
# separately fenced.
toml_set p2p persistent_peers '""' "${ARCHIVE_HOME}/config/config.toml"
toml_set p2p seeds '""' "${ARCHIVE_HOME}/config/config.toml"
cp "${ARCHIVE_KEY_HOME}/config/node_key.json" "${ARCHIVE_HOME}/config/node_key.json"
cp "${ARCHIVE_KEY_HOME}/config/priv_validator_key.json" \
  "${ARCHIVE_HOME}/config/priv_validator_key.json"
cp "${ARCHIVE_KEY_HOME}/data/priv_validator_state.json" \
  "${ARCHIVE_HOME}/data/priv_validator_state.json"
for name in application.db blockstore.db state.db evidence.db tx_index.db; do
  if [ -e "${LIVE_OBSERVER_HOME}/data/${name}" ]; then
    cp -R "${LIVE_OBSERVER_HOME}/data/${name}" "${ARCHIVE_HOME}/data/${name}"
  fi
done
for required in application.db blockstore.db state.db; do
  [ -e "${ARCHIVE_HOME}/data/${required}" ] || fail "archive allowlist missed ${required}"
done

ARCHIVE_NODE_ID="$("${BINARY}" tendermint show-node-id --home "${ARCHIVE_HOME}")"
ARCHIVE_VALIDATOR_PUB="$("${BINARY}" tendermint show-validator --home "${ARCHIVE_HOME}" | jq -r '.key')"
[ "${ARCHIVE_NODE_ID}" != "${LIVE_OBSERVER_NODE_ID}" ] || fail "archive reused live observer node identity"
[ "${ARCHIVE_VALIDATOR_PUB}" != "${LIVE_OBSERVER_VALIDATOR_PUB}" ] || \
  fail "archive reused live observer consensus identity"
[ "${ARCHIVE_VALIDATOR_PUB}" != "${SIGNER_VALIDATOR_PUB}" ] || \
  fail "archive reused signer consensus identity"
[ ! -e "${ARCHIVE_HOME}/keyring-test" ] || fail "archive contains account keyring"
if find "${ARCHIVE_HOME}" -type f \( -name '*.mnemonic' -o -name '.env' -o \
  -name '*identity*' -o -name '*secret*' \) -print -quit | grep -q .; then
  fail "archive contains a forbidden custody-shaped file"
fi

# The offline live-observer database retains staged H and its seen commit.
# In Comet's pending-block special case (block store H, state/app A), hard
# rollback on the serving copy removes only H and keeps state/app at A. The archive
# therefore starts from an aligned A/A database without replaying eligibility.
if ! "${BINARY}" rollback --hard --home "${ARCHIVE_HOME}" \
  >"${TMP}/archive-rollback.log" 2>&1; then
  sed -n '1,120p' "${TMP}/archive-rollback.log" >&2
  fail "could not sanitize the copied observer database"
fi
grep -q "Rolled back state to height ${FINAL_COMMITTED_HEIGHT} and hash ${POST_ANCHOR_APP_HASH}" \
  "${TMP}/archive-rollback.log" || \
  fail "copied-database rollback did not preserve application height/hash A"
printf '%s\n' '{"height":"0","round":0,"step":0}' \
  >"${ARCHIVE_HOME}/data/priv_validator_state.json"
chmod 600 "${ARCHIVE_HOME}/config/node_key.json" \
  "${ARCHIVE_HOME}/config/priv_validator_key.json" \
  "${ARCHIVE_HOME}/data/priv_validator_state.json"

"${BINARY}" start --home "${ARCHIVE_HOME}" \
  --halt-height "${HALT_TRIGGER_HEIGHT}" --minimum-gas-prices 0.025uzrn \
  >"${TMP}/archive.log" 2>&1 &
NODE_PID=$!
RPC="${OBSERVER_RPC}"
REST="${OBSERVER_REST}"
wait_for_observer

ANCHOR_JSON="$(curl --noproxy '*' -fsS --max-time 3 \
  "${RPC}/block?height=${FINAL_COMMITTED_HEIGHT}")"
ANCHOR_TXS="$(jq -r '(.result.block.data.txs // []) | length' <<<"${ANCHOR_JSON}")"
[ "${ANCHOR_TXS}" = "0" ] || fail "anchor block contains ${ANCHOR_TXS} transaction(s)"
ANCHOR_APP_HASH="$(jq -r '.result.block.header.app_hash' <<<"${ANCHOR_JSON}")"
[[ "${ANCHOR_APP_HASH}" =~ ^[0-9A-Fa-f]{64}$ ]] || fail "anchor app hash is invalid"
TRIGGER_JSON="$(curl --noproxy '*' -sS --max-time 3 \
  "${RPC}/block?height=${HALT_TRIGGER_HEIGHT}" 2>/dev/null || true)"
if jq -e --arg height "${HALT_TRIGGER_HEIGHT}" \
  '.result.block.header.height == $height' <<<"${TRIGGER_JSON}" >/dev/null 2>&1; then
  fail "observer exposed application-unapplied halt-trigger block ${HALT_TRIGGER_HEIGHT}"
fi

jq -e \
  --argjson checkpoint "${CHECKPOINT_STATE_HEIGHT}" \
  --argjson anchor "${FINAL_COMMITTED_HEIGHT}" \
  --argjson halt "${HALT_TRIGGER_HEIGHT}" \
  --arg app_hash "${ANCHOR_APP_HASH}" '
    .schema == "zerone-relaunch-snapshot-v3" and
    .source.checkpoint_state_height == $checkpoint and
    .source.final_committed_block_height == $anchor and
    .source.halt_trigger_height == $halt and
    .source.final_committed_block_txs == 0 and
    .source.final_committed_block_canonical == true and
    .source.rpc_blockstore_height == $halt and
    .source.staged_halt_trigger_block_txs == 0 and
    .source.staged_halt_trigger_commit_canonical == false and
    .source.staged_halt_trigger_has_block_results == false and
    .source.checkpoint_app_hash == ($app_hash | ascii_upcase) and
    (.source.rest_trust_model | contains("no Merkle proof"))
  ' "${OBSERVER_SNAPSHOT}" >/dev/null || fail "snapshot did not bind F/A/H and the anchor app hash"

printf 'halt checkpoint rehearsal: PASS (state F=%d, canonical empty anchor A=%d, staged application-unapplied halt trigger H=%d; independent observer matched; allowlisted archive sanitized)\n' \
  "${CHECKPOINT_STATE_HEIGHT}" "${FINAL_COMMITTED_HEIGHT}" "${HALT_TRIGGER_HEIGHT}"
