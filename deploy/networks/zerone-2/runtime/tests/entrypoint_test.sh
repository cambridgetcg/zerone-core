#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)
RUNTIME_DIR="${ROOT}/deploy/networks/zerone-2/runtime"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-runtime-test.XXXXXX")
BG_PID=""
cleanup() {
  if [ -n "${BG_PID}" ] && kill -0 "${BG_PID}" 2>/dev/null; then
    kill -TERM "${BG_PID}" 2>/dev/null || true
    wait "${BG_PID}" 2>/dev/null || true
  fi
  rm -rf "${TMP}"
}
trap cleanup EXIT

SYSTEM_MV=$(command -v mv)
mkdir -p "${TMP}/bin"

# macOS does not ship util-linux flock. Production images require the real
# command; this fcntl-compatible test shim preserves the lock on inherited FD 9.
if ! command -v flock >/dev/null 2>&1; then
  cat > "${TMP}/bin/flock" <<'FLOCK'
#!/usr/bin/env python3
import fcntl
import sys

if len(sys.argv) != 3 or sys.argv[1] != "-n":
    raise SystemExit(2)
try:
    fcntl.flock(int(sys.argv[2]), fcntl.LOCK_EX | fcntl.LOCK_NB)
except BlockingIOError:
    raise SystemExit(1)
FLOCK
  chmod +x "${TMP}/bin/flock"
fi

# Production uses GNU mv. Keep the behavioral suite portable to macOS while
# preserving GNU --no-target-directory/--no-clobber semantics for an absent or
# already-present destination.
if ! mv --version 2>/dev/null | grep -q 'GNU coreutils'; then
  cat > "${TMP}/bin/mv" <<'MV'
#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -eq 5 ] && [ "$1" = "--no-target-directory" ] && \
    [ "$2" = "--no-clobber" ] && [ "$3" = "--" ]; then
  source_path="$4"
  destination_path="$5"
  if [ -e "${destination_path}" ] || [ -L "${destination_path}" ]; then
    exit 0
  fi
  exec "${REAL_SYSTEM_MV:?}" "${source_path}" "${destination_path}"
fi
exec "${REAL_SYSTEM_MV:?}" "$@"
MV
  chmod +x "${TMP}/bin/mv"
  export REAL_SYSTEM_MV="${SYSTEM_MV}"
fi
export PATH="${TMP}/bin:${PATH}"

pass_count=0
fail() { printf 'runtime test: FAIL: %s\n' "$*" >&2; exit 1; }
pass() { pass_count=$((pass_count + 1)); printf 'ok %d - %s\n' "${pass_count}" "$1"; }

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

file_mode() {
  if stat -c '%a' "$1" >/dev/null 2>&1; then
    stat -c '%a' "$1"
  else
    stat -f '%Lp' "$1"
  fi
}

b64_file() {
  base64 < "$1" | tr -d '\n'
}

expect_failure() {
  local label="$1" pattern="$2" logfile="$3"
  shift 3
  if "$@" > "${logfile}" 2>&1; then
    fail "${label}: command unexpectedly succeeded"
  fi
  grep -q "${pattern}" "${logfile}" || {
    sed -n '1,100p' "${logfile}" >&2
    fail "${label}: expected error matching ${pattern}"
  }
  pass "${label}"
}

toml_test_set() {
  local section="$1" key="$2" value="$3" file="$4" tmp
  tmp="${file}.test-tmp"
  awk -v target="[${section}]" -v wanted="${key}" -v replacement="${value}" '
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
  ' "${file}" > "${tmp}" || fail "could not mutate ${section}.${key} for restart test"
  mv "${tmp}" "${file}"
}

REAL_ZERONED="${REAL_ZERONED:-${ROOT}/build/zeroned}"
if [ ! -x "${REAL_ZERONED}" ]; then
  REAL_ZERONED="${TMP}/real-zeroned"
  (cd "${ROOT}" && go build -o "${REAL_ZERONED}" ./cmd/zeroned)
fi

# Generate two genuine, throwaway CometBFT Ed25519 key pairs.
VAL1_FIXTURE="${TMP}/fixture-val1"
VAL2_FIXTURE="${TMP}/fixture-val2"
"${REAL_ZERONED}" init fixture-val1 --chain-id zerone-2 --default-denom uzrn \
  --home "${VAL1_FIXTURE}" >/dev/null 2>&1
"${REAL_ZERONED}" init fixture-val2 --chain-id zerone-2 --default-denom uzrn \
  --home "${VAL2_FIXTURE}" >/dev/null 2>&1

VAL1_NODE_ID=$("${REAL_ZERONED}" tendermint show-node-id --home "${VAL1_FIXTURE}")
VAL2_NODE_ID=$("${REAL_ZERONED}" tendermint show-node-id --home "${VAL2_FIXTURE}")
VAL1_PUB=$("${REAL_ZERONED}" tendermint show-validator --home "${VAL1_FIXTURE}" | jq -er '.key')

GENESIS="${TMP}/genesis.json"
jq -n --arg pub "${VAL1_PUB}" '{
  chain_id: "zerone-2",
  app_state: {
    genutil: {
      gen_txs: [
        {body: {messages: [{pubkey: {key: $pub}}]}}
      ]
    }
  }
}' > "${GENESIS}"
GENESIS_SHA=$(sha256_file "${GENESIS}")

# Delegate real key/init commands to zeroned, but intercept start so no network
# listener or consensus process is created. Every invocation records whether a
# bootstrap secret leaked into a child environment.
WRAPPER="${TMP}/zeroned-wrapper"
cat > "${WRAPPER}" <<'WRAPPER'
#!/usr/bin/env bash
set -euo pipefail
if env | grep -Eq '^(VALIDATOR_KEY_B64|NODE_KEY_B64|PRIV_VALIDATOR_KEY_B64)='; then
  printf 'secret leaked to child: %s\n' "$*" >> "${CHILD_SECRET_LEAK_FILE:?}"
fi
if [ "${1:-}" = "start" ]; then
  printf '%s\n' "$*" > "${START_ARGS_FILE:?}"
  env | sort > "${START_ENV_FILE:?}"
  if [ -n "${START_HOLD_READY:-}" ]; then
    : > "${START_HOLD_READY}"
    while [ ! -e "${START_HOLD_RELEASE:?}" ]; do
      sleep 0.05
    done
  fi
  exit 0
fi
exec "${REAL_ZERONED:?}" "$@"
WRAPPER
chmod +x "${WRAPPER}"
WRAPPER_SHA=$(sha256_file "${WRAPPER}")

MANIFEST="${TMP}/network.env"
printf 'chain_id=zerone-2\ngenesis_sha256=%s\nvalidator_node_id=%s\nbinary_sha256=%s\n' \
  "${GENESIS_SHA}" "${VAL1_NODE_ID}" "${WRAPPER_SHA}" > "${MANIFEST}"

make_test_entrypoint() {
  local genesis="$1" manifest="$2" output="$3"
  sed \
    -e "s|PUBLIC_GENESIS=\"/network/genesis.json\"|PUBLIC_GENESIS=\"${genesis}\"|" \
    -e "s|NETWORK_MANIFEST=\"/network/network.env\"|NETWORK_MANIFEST=\"${manifest}\"|" \
    -e "s|readonly BINARY=\"/usr/local/bin/zeroned\"|readonly BINARY=\"${WRAPPER}\"|" \
    "${RUNTIME_DIR}/entrypoint.sh" > "${output}"
  chmod +x "${output}"
}

ENTRYPOINT="${TMP}/entrypoint"
make_test_entrypoint "${GENESIS}" "${MANIFEST}" "${ENTRYPOINT}"

PEERS="${VAL1_NODE_ID}@127.0.0.1:26656"
VALIDATOR_B64=$(b64_file "${VAL1_FIXTURE}/config/priv_validator_key.json")
NODE_B64=$(b64_file "${VAL1_FIXTURE}/config/node_key.json")

run_role() {
  local role="$1" home="$2" args_file="$3" env_file="$4" leak_file="$5"
  shift 5
  env \
    NODE_ROLE="${role}" \
    ZERONE_HOME="${home}" \
    REAL_ZERONED="${REAL_ZERONED}" \
    START_ARGS_FILE="${args_file}" \
    START_ENV_FILE="${env_file}" \
    CHILD_SECRET_LEAK_FILE="${leak_file}" \
    PERSISTENT_PEERS="${PEERS}" \
    PRIVATE_PEER_IDS="${VAL1_NODE_ID}" \
    "$@" "${ENTRYPOINT}"
}

# Simulate a non-cooperating writer creating ZERONE_HOME after the entrypoint's
# pre-publish check but before mv. The interposer returns GNU mv -n's successful
# no-clobber result while leaving staging in place; the entrypoint must detect
# that result, fail closed, and scrub the secret-bearing staging directory.
PUBLISH_RACE_PARENT="${TMP}/publish-race"
PUBLISH_RACE_HOME="${PUBLISH_RACE_PARENT}/validator-home"
PUBLISH_RACE_BIN="${TMP}/publish-race-bin"
PUBLISH_RACE_LOG="${TMP}/publish-race-mv.log"
PUBLISH_RACE_SENTINEL="${PUBLISH_RACE_HOME}/interposed-sentinel"
mkdir -p "${PUBLISH_RACE_PARENT}" "${PUBLISH_RACE_BIN}"
cat > "${PUBLISH_RACE_BIN}/mv" <<'MV_RACE'
#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -eq 5 ] && [ "$1" = "--no-target-directory" ] && \
    [ "$2" = "--no-clobber" ] && [ "$3" = "--" ] && \
    [[ "$4" == "${PUBLISH_RACE_PARENT}/.zerone-2-init."* ]] && \
    [ "$5" = "${PUBLISH_RACE_HOME}" ]; then
  printf '%s\n' "$*" > "${PUBLISH_RACE_LOG}"
  mkdir -p "${PUBLISH_RACE_HOME}"
  printf 'interposed destination sentinel\n' > "${PUBLISH_RACE_SENTINEL}"
  chmod 0644 "${PUBLISH_RACE_SENTINEL}"
  exit 0
fi
exec "${REAL_SYSTEM_MV:?}" "$@"
MV_RACE
chmod +x "${PUBLISH_RACE_BIN}/mv"
expect_failure \
  "interposed publish destination fails without nesting custody staging" \
  "appeared during atomic initialization publish" "${TMP}/publish-race.log" \
  run_role validator "${PUBLISH_RACE_HOME}" "${TMP}/publish-race-args" \
    "${TMP}/publish-race-env" "${TMP}/publish-race-leak" \
    PATH="${PUBLISH_RACE_BIN}:${PATH}" REAL_SYSTEM_MV="${SYSTEM_MV}" \
    PUBLISH_RACE_PARENT="${PUBLISH_RACE_PARENT}" \
    PUBLISH_RACE_HOME="${PUBLISH_RACE_HOME}" \
    PUBLISH_RACE_LOG="${PUBLISH_RACE_LOG}" \
    PUBLISH_RACE_SENTINEL="${PUBLISH_RACE_SENTINEL}" \
    VALIDATOR_KEY_B64="${VALIDATOR_B64}" NODE_KEY_B64="${NODE_B64}"
[ -f "${PUBLISH_RACE_LOG}" ] || fail "publish race did not reach no-target-directory mv"
grep -q '^--no-target-directory --no-clobber -- ' "${PUBLISH_RACE_LOG}" || \
  fail "publish move omitted fail-closed GNU mv flags"
[ "$(cat "${PUBLISH_RACE_SENTINEL}")" = "interposed destination sentinel" ] && \
  [ "$(file_mode "${PUBLISH_RACE_SENTINEL}")" = "644" ] || \
  fail "interposed destination sentinel was mutated"
[ ! -e "${PUBLISH_RACE_HOME}/config" ] && \
  [ ! -e "${PUBLISH_RACE_HOME}/.runtime-initialized" ] || \
  fail "secret-bearing staging was nested beneath the interposed destination"
if find "${PUBLISH_RACE_PARENT}" -maxdepth 1 \
    \( -name '.zerone-2-init.*' -o -name '.zerone-2-secrets.*' \) \
    -print -quit | grep -q .; then
  fail "secret-bearing initialization scratch survived a publish collision"
fi

VAL_HOME="${TMP}/validator-home"
VAL_ARGS="${TMP}/validator-start-args"
VAL_ENV="${TMP}/validator-start-env"
LEAK_FILE="${TMP}/child-secret-leak"

run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}" \
  VALIDATOR_KEY_B64="${VALIDATOR_B64}" NODE_KEY_B64="${NODE_B64}" >/dev/null

cmp -s "${VAL1_FIXTURE}/config/priv_validator_key.json" \
  "${VAL_HOME}/config/priv_validator_key.json" || fail "validator key was not restored exactly"
cmp -s "${VAL1_FIXTURE}/config/node_key.json" \
  "${VAL_HOME}/config/node_key.json" || fail "validator node key was not restored exactly"
[ ! -e "${LEAK_FILE}" ] || fail "bootstrap secrets reached a zeroned child"
grep -q '^role=validator$' "${VAL_HOME}/.runtime-initialized" || fail "validator marker missing"
grep -q -- '--minimum-gas-prices 1uzrn' "${VAL_ARGS}" || fail "start args were not fixed"
if grep -Eq '^(VALIDATOR_KEY_B64|NODE_KEY_B64|PRIV_VALIDATOR_KEY_B64)=' "${VAL_ENV}"; then
  fail "bootstrap secrets reached the daemon environment"
fi
grep -A6 '^\[api\]' "${VAL_HOME}/config/app.toml" | grep -q '^enable = false$' || \
  fail "validator API was not disabled"
grep -A55 '^\[p2p\]' "${VAL_HOME}/config/config.toml" | grep -q '^pex = false$' || \
  fail "validator peer exchange was not disabled"
pass "real validator keys initialize once and never reach a child process"

# Resuming requires no bootstrap secrets and preserves the role/key marker.
rm -f "${VAL_ARGS}" "${VAL_ENV}"
toml_test_set rpc laddr '"tcp://0.0.0.0:26657"' \
  "${VAL_HOME}/config/config.toml"
toml_test_set rpc unsafe true "${VAL_HOME}/config/config.toml"
toml_test_set p2p pex true "${VAL_HOME}/config/config.toml"
toml_test_set api enable true "${VAL_HOME}/config/app.toml"
toml_test_set grpc enable true "${VAL_HOME}/config/app.toml"
run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}" >/dev/null
[ -f "${VAL_ARGS}" ] || fail "validator resume did not reach start"
grep -A40 '^\[rpc\]' "${VAL_HOME}/config/config.toml" | \
  grep -q '^laddr = "tcp://127.0.0.1:26657"$' || fail "resume retained public validator RPC"
grep -A40 '^\[rpc\]' "${VAL_HOME}/config/config.toml" | \
  grep -q '^unsafe = false$' || fail "resume retained unsafe validator RPC"
grep -A55 '^\[p2p\]' "${VAL_HOME}/config/config.toml" | \
  grep -q '^pex = false$' || fail "resume retained validator peer exchange"
grep -A6 '^\[api\]' "${VAL_HOME}/config/app.toml" | \
  grep -q '^enable = false$' || fail "resume retained validator API"
grep -A6 '^\[grpc\]' "${VAL_HOME}/config/app.toml" | \
  grep -q '^enable = false$' || fail "resume retained validator gRPC"
pass "validator resume reasserts the complete role boundary"

# The FD lock must survive exec and refuse a concurrent process on the same
# home. This is local fencing; deployment still must prevent copied keys on a
# second volume or host.
LOCK_READY="${TMP}/lock-ready"
LOCK_RELEASE="${TMP}/lock-release"
run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}" \
  START_HOLD_READY="${LOCK_READY}" START_HOLD_RELEASE="${LOCK_RELEASE}" \
  > "${TMP}/lock-owner.log" 2>&1 &
BG_PID=$!
for _ in $(seq 1 100); do
  [ -e "${LOCK_READY}" ] && break
  kill -0 "${BG_PID}" 2>/dev/null || fail "lock owner exited before acquiring lock"
  sleep 0.02
done
[ -e "${LOCK_READY}" ] || fail "lock owner never reached start"
expect_failure \
  "concurrent process is fenced from the same home" "already owns" \
  "${TMP}/lock-contender.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"
: > "${LOCK_RELEASE}"
wait "${BG_PID}" || fail "lock owner failed after release"
BG_PID=""

LOCK_SYMLINK_HOME="${TMP}/lock-symlink-home"
LOCK_SENTINEL="${TMP}/lock-sentinel"
printf 'lock sentinel\n' > "${LOCK_SENTINEL}"
chmod 0644 "${LOCK_SENTINEL}"
LOCK_SENTINEL_SHA=$(sha256_file "${LOCK_SENTINEL}")
ln -s "${LOCK_SENTINEL}" "${LOCK_SYMLINK_HOME}.runtime.lock"
expect_failure \
  "preplanted lock symlink fails without target mutation" \
  "runtime lock must not be a symlink" "${TMP}/lock-symlink.log" \
  run_role edge "${LOCK_SYMLINK_HOME}" "${TMP}/lock-symlink-args" \
    "${TMP}/lock-symlink-env" "${LEAK_FILE}"
[ "$(sha256_file "${LOCK_SENTINEL}")" = "${LOCK_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${LOCK_SENTINEL}")" = "644" ] || \
  fail "lock symlink target was modified before rejection"

cp "${VAL_HOME}/config/genesis.json" "${TMP}/valid-home-genesis.json"
printf '\n' >> "${VAL_HOME}/config/genesis.json"
expect_failure \
  "initialized volume rechecks exact genesis bytes" "genesis sha256 mismatch" \
  "${TMP}/mutated-home-genesis.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"
cp "${TMP}/valid-home-genesis.json" "${VAL_HOME}/config/genesis.json"

expect_failure \
  "initialized validator rejects retained bootstrap secrets" \
  "bootstrap custody inputs are forbidden" "${TMP}/retained-secret.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}" \
    VALIDATOR_KEY_B64="${VALIDATOR_B64}"

expect_failure \
  "validator can never enable the query origin" "can never enable" \
  "${TMP}/validator-query-origin.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}" \
    QUERY_ORIGIN_ENABLED=true

# Stored public metadata cannot disguise a different real private key.
MISMATCH_KEY="${TMP}/mismatched-validator-key.json"
jq --arg pub "${VAL1_PUB}" '.pub_key.value = $pub' \
  "${VAL2_FIXTURE}/config/priv_validator_key.json" > "${MISMATCH_KEY}"
MISMATCH_B64=$(b64_file "${MISMATCH_KEY}")
expect_failure \
  "validator public/private mismatch fails before initialization" \
  "public key does not match its private key" "${TMP}/mismatch.log" \
  run_role validator "${TMP}/mismatch-home" "${TMP}/mismatch-args" \
    "${TMP}/mismatch-env" "${LEAK_FILE}" \
    VALIDATOR_KEY_B64="${MISMATCH_B64}" NODE_KEY_B64="${NODE_B64}"

BAD_NODE_MANIFEST="${TMP}/bad-node.env"
printf 'chain_id=zerone-2\ngenesis_sha256=%s\nvalidator_node_id=%s\nbinary_sha256=%s\n' \
  "${GENESIS_SHA}" "${VAL2_NODE_ID}" "${WRAPPER_SHA}" > "${BAD_NODE_MANIFEST}"
BAD_NODE_ENTRYPOINT="${TMP}/bad-node-entrypoint"
make_test_entrypoint "${GENESIS}" "${BAD_NODE_MANIFEST}" "${BAD_NODE_ENTRYPOINT}"
expect_failure \
  "validator node identity is pinned by the image manifest" \
  "expected ${VAL2_NODE_ID}" "${TMP}/bad-node.log" \
  env NODE_ROLE=validator ZERONE_HOME="${TMP}/bad-node-home" \
    REAL_ZERONED="${REAL_ZERONED}" \
    START_ARGS_FILE="${TMP}/bad-node-args" START_ENV_FILE="${TMP}/bad-node-env" \
    CHILD_SECRET_LEAK_FILE="${LEAK_FILE}" PERSISTENT_PEERS="${PEERS}" \
    PRIVATE_PEER_IDS="${VAL1_NODE_ID}" VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
    NODE_KEY_B64="${NODE_B64}" "${BAD_NODE_ENTRYPOINT}"

BAD_HASH_MANIFEST="${TMP}/bad-hash.env"
printf 'chain_id=zerone-2\ngenesis_sha256=%064d\nvalidator_node_id=%s\nbinary_sha256=%s\n' \
  0 "${VAL1_NODE_ID}" "${WRAPPER_SHA}" > "${BAD_HASH_MANIFEST}"
BAD_HASH_ENTRYPOINT="${TMP}/bad-hash-entrypoint"
make_test_entrypoint "${GENESIS}" "${BAD_HASH_MANIFEST}" "${BAD_HASH_ENTRYPOINT}"
expect_failure \
  "genesis bytes are pinned exactly" "genesis sha256 mismatch" "${TMP}/bad-hash.log" \
  env NODE_ROLE=edge ZERONE_HOME="${TMP}/bad-hash-home" \
    REAL_ZERONED="${REAL_ZERONED}" \
    START_ARGS_FILE="${TMP}/bad-hash-args" START_ENV_FILE="${TMP}/bad-hash-env" \
    CHILD_SECRET_LEAK_FILE="${LEAK_FILE}" PERSISTENT_PEERS="${PEERS}" \
    PRIVATE_PEER_IDS="${VAL1_NODE_ID}" "${BAD_HASH_ENTRYPOINT}"

BAD_BINARY_MANIFEST="${TMP}/bad-binary.env"
printf 'chain_id=zerone-2\ngenesis_sha256=%s\nvalidator_node_id=%s\nbinary_sha256=%064d\n' \
  "${GENESIS_SHA}" "${VAL1_NODE_ID}" 0 > "${BAD_BINARY_MANIFEST}"
BAD_BINARY_ENTRYPOINT="${TMP}/bad-binary-entrypoint"
make_test_entrypoint "${GENESIS}" "${BAD_BINARY_MANIFEST}" "${BAD_BINARY_ENTRYPOINT}"
expect_failure \
  "runtime binary bytes are pinned exactly" "zeroned binary sha256 mismatch" \
  "${TMP}/bad-binary.log" \
  env NODE_ROLE=edge ZERONE_HOME="${TMP}/bad-binary-home" \
    REAL_ZERONED="${REAL_ZERONED}" \
    START_ARGS_FILE="${TMP}/bad-binary-args" \
    START_ENV_FILE="${TMP}/bad-binary-env" \
    CHILD_SECRET_LEAK_FILE="${LEAK_FILE}" PERSISTENT_PEERS="${PEERS}" \
    PRIVATE_PEER_IDS="${VAL1_NODE_ID}" "${BAD_BINARY_ENTRYPOINT}"

BAD_CHAIN_GENESIS="${TMP}/bad-chain-genesis.json"
jq '.chain_id = "zerone-1"' "${GENESIS}" > "${BAD_CHAIN_GENESIS}"
BAD_CHAIN_SHA=$(sha256_file "${BAD_CHAIN_GENESIS}")
BAD_CHAIN_MANIFEST="${TMP}/bad-chain.env"
printf 'chain_id=zerone-2\ngenesis_sha256=%s\nvalidator_node_id=%s\nbinary_sha256=%s\n' \
  "${BAD_CHAIN_SHA}" "${VAL1_NODE_ID}" "${WRAPPER_SHA}" > "${BAD_CHAIN_MANIFEST}"
BAD_CHAIN_ENTRYPOINT="${TMP}/bad-chain-entrypoint"
make_test_entrypoint "${BAD_CHAIN_GENESIS}" "${BAD_CHAIN_MANIFEST}" "${BAD_CHAIN_ENTRYPOINT}"
expect_failure \
  "chain ID is exactly zerone-2" "expected zerone-2" "${TMP}/bad-chain.log" \
  env NODE_ROLE=edge ZERONE_HOME="${TMP}/bad-chain-home" \
    REAL_ZERONED="${REAL_ZERONED}" \
    START_ARGS_FILE="${TMP}/bad-chain-args" START_ENV_FILE="${TMP}/bad-chain-env" \
    CHILD_SECRET_LEAK_FILE="${LEAK_FILE}" PERSISTENT_PEERS="${PEERS}" \
    PRIVATE_PEER_IDS="${VAL1_NODE_ID}" "${BAD_CHAIN_ENTRYPOINT}"

EDGE_HOME="${TMP}/edge-home"
EDGE_ARGS="${TMP}/edge-start-args"
EDGE_ENV="${TMP}/edge-start-env"
run_role edge "${EDGE_HOME}" "${EDGE_ARGS}" "${EDGE_ENV}" "${LEAK_FILE}" >/dev/null
grep -q '^role=edge$' "${EDGE_HOME}/.runtime-initialized" || fail "edge marker missing"
grep -A6 '^\[api\]' "${EDGE_HOME}/config/app.toml" | grep -q '^enable = false$' || \
  fail "private-soak edge API was not closed"
grep -A40 '^\[rpc\]' "${EDGE_HOME}/config/config.toml" | \
  grep -q '^laddr = "tcp://127.0.0.1:26657"$' || fail "private-soak edge RPC was public"
grep -A6 '^\[grpc\]' "${EDGE_HOME}/config/app.toml" | grep -q '^enable = false$' || \
  fail "private-soak edge gRPC was not closed"
grep -A55 '^\[p2p\]' "${EDGE_HOME}/config/config.toml" | grep -q '^pex = true$' || \
  fail "edge peer exchange was not enabled"
grep -A12 '^\[state-sync\]' "${EDGE_HOME}/config/app.toml" | grep -q '^snapshot-interval = 1000$' || \
  fail "edge snapshots were not enabled"
pass "edge defaults to a closed private query origin"

run_role edge "${EDGE_HOME}" "${TMP}/public-edge-args" \
  "${TMP}/public-edge-env" "${LEAK_FILE}" QUERY_ORIGIN_ENABLED=true >/dev/null
grep -A40 '^\[rpc\]' "${EDGE_HOME}/config/config.toml" | \
  grep -q '^laddr = "tcp://0.0.0.0:26657"$' || fail "explicit private query origin RPC stayed closed"
grep -A6 '^\[api\]' "${EDGE_HOME}/config/app.toml" | \
  grep -q '^enable = true$' || fail "explicit private query origin REST stayed closed"
grep -A6 '^\[grpc\]' "${EDGE_HOME}/config/app.toml" | \
  grep -q '^enable = false$' || fail "query origin unexpectedly opened gRPC"
pass "query-soak profile opens private RPC/REST without gRPC"

expect_failure \
  "query origin rejects ambiguous truthy values" "must be exactly true or false" \
  "${TMP}/ambiguous-query-origin.log" \
  run_role edge "${TMP}/ambiguous-query-origin-home" "${TMP}/ambiguous-query-origin-args" \
    "${TMP}/ambiguous-query-origin-env" "${LEAK_FILE}" QUERY_ORIGIN_ENABLED=1

expect_failure \
  "edge rejects validator custody inputs" "edge role rejects" "${TMP}/edge-custody.log" \
  run_role edge "${TMP}/edge-custody-home" "${TMP}/edge-custody-args" \
    "${TMP}/edge-custody-env" "${LEAK_FILE}" VALIDATOR_KEY_B64="${VALIDATOR_B64}"

mkdir -p "${TMP}/partial-home"
printf 'partial\n' > "${TMP}/partial-home/untrusted-file"
expect_failure \
  "partial non-empty volume fails closed" "partially initialized" "${TMP}/partial.log" \
  run_role edge "${TMP}/partial-home" "${TMP}/partial-args" \
    "${TMP}/partial-env" "${LEAK_FILE}"

cp "${VAL_HOME}/data/priv_validator_state.json" "${TMP}/valid-state.json"
printf '{}\n' > "${VAL_HOME}/data/priv_validator_state.json"
expect_failure \
  "corrupt signer state blocks validator resume" "invalid priv_validator_state" \
  "${TMP}/bad-state.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"
cp "${TMP}/valid-state.json" "${VAL_HOME}/data/priv_validator_state.json"

mv "${VAL_HOME}/config/node_key.json" "${VAL_HOME}/config/node_key.real.json"
chmod 0644 "${VAL_HOME}/config/node_key.real.json"
SYMLINK_KEY_SHA=$(sha256_file "${VAL_HOME}/config/node_key.real.json")
ln -s "node_key.real.json" "${VAL_HOME}/config/node_key.json"
expect_failure \
  "symlinked custody files fail closed" "regular, non-symlink" \
  "${TMP}/symlink-key.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"
[ "$(sha256_file "${VAL_HOME}/config/node_key.real.json")" = "${SYMLINK_KEY_SHA}" ] && \
  [ "$(file_mode "${VAL_HOME}/config/node_key.real.json")" = "644" ] || \
  fail "symlinked validator key target was modified before rejection"
rm "${VAL_HOME}/config/node_key.json"
mv "${VAL_HOME}/config/node_key.real.json" "${VAL_HOME}/config/node_key.json"
chmod 0600 "${VAL_HOME}/config/node_key.json"

mv "${VAL_HOME}/.runtime-initialized" "${TMP}/runtime-marker.saved"
MARKER_SENTINEL="${TMP}/runtime-marker-sentinel"
printf 'marker sentinel\n' > "${MARKER_SENTINEL}"
chmod 0644 "${MARKER_SENTINEL}"
MARKER_SENTINEL_SHA=$(sha256_file "${MARKER_SENTINEL}")
ln -s "${MARKER_SENTINEL}" "${VAL_HOME}/.runtime-initialized"
expect_failure \
  "symlinked runtime marker fails without target mutation" \
  "runtime marker must not be a symlink" "${TMP}/marker-symlink.log" \
  run_role validator "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"
[ "$(sha256_file "${MARKER_SENTINEL}")" = "${MARKER_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${MARKER_SENTINEL}")" = "644" ] || \
  fail "runtime marker symlink target was modified"
rm "${VAL_HOME}/.runtime-initialized"
mv "${TMP}/runtime-marker.saved" "${VAL_HOME}/.runtime-initialized"

expect_failure \
  "a marked validator volume cannot become an edge" \
  "edge node unexpectedly holds" "${TMP}/role-change.log" \
  run_role edge "${VAL_HOME}" "${VAL_ARGS}" "${VAL_ENV}" "${LEAK_FILE}"

# Static custody/deployment assertions complement the behavioral tests.
if grep -Eq '^COPY[[:space:]]+\.[[:space:]]+\.' "${RUNTIME_DIR}/Dockerfile"; then
  fail "Dockerfile uses broad COPY . ."
fi
if grep -Eqi '^COPY .*\b(mnemonic|node_key|priv_validator_key)' "${RUNTIME_DIR}/Dockerfile"; then
  fail "Dockerfile copies custody material"
fi
grep -q '^readonly BINARY="/usr/local/bin/zeroned"$' \
  "${RUNTIME_DIR}/entrypoint.sh" || fail "runtime binary path is not immutable"
if grep -q 'ZERONED_BINARY' "${RUNTIME_DIR}/entrypoint.sh"; then
  fail "production runtime accepts a zeroned binary override"
fi
grep -q 'EXPECTED_BINARY_SHA256=.*binary_sha256' \
  "${RUNTIME_DIR}/entrypoint.sh" || fail "boot manifest omits the binary hash"
# shellcheck disable=SC2016
grep -q 'test "${TARGETOS}" = "${BINARY_GOOS}"' \
  "${RUNTIME_DIR}/Dockerfile" || fail "Docker target OS is not checked"
# shellcheck disable=SC2016
grep -q 'test "${TARGETARCH}" = "${BINARY_GOARCH}"' \
  "${RUNTIME_DIR}/Dockerfile" || fail "Docker target architecture is not checked"
# shellcheck disable=SC2016
grep -q 'BINARY_VERSION_OUTPUT=$(/usr/local/bin/zeroned version' \
  "${RUNTIME_DIR}/Dockerfile" || fail "target stage does not execute the release binary"
if grep -Eq 'BINARY_VERSION_OUTPUT|"\$\{RELEASE_BINARY\}" version' \
  "${RUNTIME_DIR}/build-image.sh"; then
  fail "host build wrapper executes the target release binary"
fi
grep -q '^install_git_blob deploy/networks/zerone-2/runtime/Dockerfile' \
  "${RUNTIME_DIR}/build-image.sh" || \
  fail "Dockerfile is not materialized from the signed source commit"
grep -q '^PUBLIC_ARTIFACT_NAMES=(' "${RUNTIME_DIR}/build-image.sh" || \
  fail "release build does not enumerate the complete public artifact snapshot"
# shellcheck disable=SC2016
grep -q '^ARTIFACT_DIR="${SNAPSHOT}/artifacts"$' \
  "${RUNTIME_DIR}/build-image.sh" || \
  fail "release audit does not consume the private artifact snapshot"
# shellcheck disable=SC2016
grep -q '^RELEASE_BINARY="${SNAPSHOT}/release/zeroned"$' \
  "${RUNTIME_DIR}/build-image.sh" || \
  fail "release verification does not consume the private binary snapshot"
[ "$(grep -c '^verify_clean_source_checkout$' "${RUNTIME_DIR}/build-image.sh")" -ge 2 ] || \
  fail "source checkout is not rechecked immediately before Docker"
if grep -q '^\[\[services\]\]' "${RUNTIME_DIR}/fly.validator.example.toml"; then
  fail "validator Fly example exposes a public service"
fi
grep -q '^  QUERY_ORIGIN_ENABLED = "false"$' \
  "${RUNTIME_DIR}/fly.edge.example.toml" || fail "private-soak Fly profile is not fail-closed"
if grep -q '^\[\[services\]\]' "${RUNTIME_DIR}/fly.edge.example.toml"; then
  fail "private-soak Fly profile exposes a public API or P2P transaction relay"
fi
grep -q '^  QUERY_ORIGIN_ENABLED = "true"$' \
  "${RUNTIME_DIR}/fly.edge.query-soak.example.toml" || \
  fail "query-soak profile does not open the private origin"
if grep -q '^\[\[services\]\]' "${RUNTIME_DIR}/fly.edge.query-soak.example.toml"; then
  fail "query-soak profile exposes a public Fly service"
fi
grep -q '^  QUERY_ORIGIN_ENABLED = "true"$' \
  "${RUNTIME_DIR}/fly.edge.public.example.toml" || fail "public Fly profile lacks its query origin"
grep -q 'hard_limit' "${RUNTIME_DIR}/fly.edge.public.example.toml" || \
  fail "public edge Fly profile lacks connection bounds"
[ "$(grep -c '^\[\[services\]\]$' \
  "${RUNTIME_DIR}/fly.edge.public.example.toml")" -eq 1 ] || \
  fail "public edge Fly profile must expose exactly one direct P2P service"
grep -q '^  internal_port = 26656$' \
  "${RUNTIME_DIR}/fly.edge.public.example.toml" || \
  fail "public edge Fly profile does not expose P2P"
grep -q 'services.tcp_checks' "${RUNTIME_DIR}/fly.edge.public.example.toml" || \
  fail "public edge Fly profile lacks a P2P TCP health check"
if grep -Eq 'services.http_checks|internal_port = (26657|1317|9090)' \
  "${RUNTIME_DIR}/fly.edge.public.example.toml"; then
  fail "public edge Fly profile directly exposes a query service"
fi
pass "image context and Fly profiles preserve the role and relay boundaries"

printf 'runtime test: PASS (%d checks, real throwaway Ed25519 fixtures)\n' "${pass_count}"
