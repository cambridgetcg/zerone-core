#!/usr/bin/env bash
# zerone-2 role-separated runtime entrypoint.
#
# The image carries only the zeroned binary, this script, a public genesis, and
# a public network manifest. Validator custody enters an empty volume exactly
# once and is never accepted by the edge role.
set -euo pipefail
umask 077

readonly RUNTIME_VERSION="1"
readonly REQUIRED_CHAIN_ID="zerone-2"
readonly QUERY_GAS_LIMIT="5000000"
readonly API_READ_TIMEOUT_SECONDS="10"
readonly API_WRITE_TIMEOUT_SECONDS="15"
readonly RPC_MAX_BODY_BYTES="65536"
readonly RPC_MAX_HEADER_BYTES="16384"

HOME_DIR="${ZERONE_HOME:-/data/.zeroned}"
# These paths are intentionally not runtime-overridable. The public genesis and
# its hash manifest are one immutable pair inside the release image.
PUBLIC_GENESIS="/network/genesis.json"
NETWORK_MANIFEST="/network/network.env"
readonly BINARY="/usr/local/bin/zeroned"
NODE_ROLE="${NODE_ROLE:-}"

# Fly bootstrap secrets arrive as exported environment variables. Copy them into
# non-exported shell variables and remove the originals before spawning even a
# helper process (jq, awk, sha256sum, flock, or zeroned).
BOOTSTRAP_VALIDATOR_KEY_B64="${VALIDATOR_KEY_B64:-}"
BOOTSTRAP_NODE_KEY_B64="${NODE_KEY_B64:-}"
BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64="${PRIV_VALIDATOR_KEY_B64:-}"
unset VALIDATOR_KEY_B64 NODE_KEY_B64 PRIV_VALIDATOR_KEY_B64

die() {
  printf '[zerone-2-runtime] ERROR: %s\n' "$*" >&2
  exit 1
}

reject_daemon_env_overrides() {
  local name
  while IFS= read -r name; do
    case "${name}" in
      ZERONED_*) die "daemon configuration environment override is forbidden: ${name}" ;;
    esac
  done < <(compgen -e)
}

info() {
  printf '[zerone-2-runtime] %s\n' "$*"
}

require_regular_file() {
  local file="$1" label="$2"
  [ -f "${file}" ] && [ ! -L "${file}" ] || \
    die "${label} must be a regular, non-symlink file: ${file}"
}

require_directory() {
  local directory="$1" label="$2"
  [ -d "${directory}" ] && [ ! -L "${directory}" ] || \
    die "${label} must be a non-symlink directory: ${directory}"
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "${file}" | awk '{print $NF}'
  else
    shasum -a 256 "${file}" | awk '{print $1}'
  fi
}

decode_base64() {
  if base64 --help 2>&1 | grep -q -- '--decode'; then
    base64 --decode
  else
    base64 -D
  fi
}

kv_value() {
  local file="$1" key="$2"
  awk -v wanted="${key}" '
    index($0, "=") > 0 {
      found_key = substr($0, 1, index($0, "=") - 1)
      if (found_key == wanted) {
        count++
        print substr($0, index($0, "=") + 1)
      }
    }
    END { if (count != 1) exit 1 }
  ' "${file}"
}

toml_quote() {
  jq -Rn --arg value "$1" '$value'
}

toml_set() {
  local section="$1" key="$2" value="$3" file="$4" tmp
  require_regular_file "${file}" "TOML input"
  tmp=$(mktemp "${file}.runtime-tmp.XXXXXX") || \
    die "could not create temporary TOML file beside ${file}"
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
    die "could not set ${section}.${key} in ${file}"
  fi
  mv "${tmp}" "${file}"
}

toml_set_root() {
  local key="$1" value="$2" file="$3" tmp
  require_regular_file "${file}" "TOML input"
  tmp=$(mktemp "${file}.runtime-tmp.XXXXXX") || \
    die "could not create temporary TOML file beside ${file}"
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
    die "could not set root ${key} in ${file}"
  fi
  mv "${tmp}" "${file}"
}

validate_safe_toml_string() {
  local name="$1" value="$2"
  if grep -q '["\\]' <<< "${value}" || [[ "${value}" == *$'\n'* ]]; then
    die "${name} contains unsafe TOML characters"
  fi
}

validate_host_port() {
  local name="$1" value="$2" host port
  host="${value%:*}"
  port="${value##*:}"
  [ -n "${host}" ] && [ "${host}" != "${value}" ] || \
    die "${name} must use host:port"
  [[ "${port}" =~ ^[1-9][0-9]{0,4}$ ]] && \
    [ "$((10#${port}))" -le 65535 ] || die "${name} has an invalid TCP port"
}

validate_peer_list() {
  local name="$1" value="$2" entry node_id endpoint seen
  [ -n "${value}" ] || die "${name} is required"
  validate_safe_toml_string "${name}" "${value}"

  PERSISTENT_PEER_NODE_IDS=","
  seen=","
  IFS=',' read -r -a peer_entries <<< "${value}"
  for entry in "${peer_entries[@]}"; do
    node_id="${entry%%@*}"
    endpoint="${entry#*@}"
    [[ "${node_id}" =~ ^[0-9a-f]{40}$ ]] || \
      die "${name} entries must begin with a 40-character lowercase hex node ID"
    [ "${endpoint}" != "${entry}" ] && [ -n "${endpoint}" ] || \
      die "${name} entries must use node-id@host:port"
    if [[ "${endpoint}" == *@* ]] || [[ "${endpoint}" =~ [[:space:],] ]]; then
      die "${name} contains an invalid peer endpoint"
    fi
    validate_host_port "${name}" "${endpoint}"
    [[ "${seen}" != *",${node_id},"* ]] || \
      die "${name} contains duplicate node ID ${node_id}"
    seen="${seen}${node_id},"
    PERSISTENT_PEER_NODE_IDS="${PERSISTENT_PEER_NODE_IDS}${node_id},"
  done
}

validate_peer_ids() {
  local value="$1" id seen
  [ -n "${value}" ] || die "PRIVATE_PEER_IDS is required"
  seen=","
  IFS=',' read -r -a ids <<< "${value}"
  for id in "${ids[@]}"; do
    [[ "${id}" =~ ^[0-9a-f]{40}$ ]] || \
      die "PRIVATE_PEER_IDS must be comma-separated 40-character lowercase hex node IDs"
    [ "${id}" != "0000000000000000000000000000000000000000" ] || \
      die "PRIVATE_PEER_IDS still contains the example placeholder"
    [[ "${seen}" != *",${id},"* ]] || \
      die "PRIVATE_PEER_IDS contains duplicate node ID ${id}"
    [[ "${PERSISTENT_PEER_NODE_IDS}" == *",${id},"* ]] || \
      die "PRIVATE_PEER_IDS node ${id} is not present in PERSISTENT_PEERS"
    seen="${seen}${id},"
  done
}

validate_cors() {
  local value="$1"
  jq -e '
    type == "array" and
    all(.[]; type == "string" and length > 0 and . != "*")
  ' <<< "${value}" >/dev/null || \
    die "CORS_ALLOWED_ORIGINS_JSON must be a JSON string array and may not contain *"
}

validate_query_origin_profile() {
  QUERY_ORIGIN_MODE="${QUERY_ORIGIN_ENABLED:-false}"
  case "${QUERY_ORIGIN_MODE}" in
    true|false) ;;
    *) die "QUERY_ORIGIN_ENABLED must be exactly true or false" ;;
  esac
  if [ "${NODE_ROLE}" = "validator" ] && [ "${QUERY_ORIGIN_MODE}" = "true" ]; then
    die "validator role can never enable the query origin"
  fi
}

read_network_manifest() {
  require_regular_file "${NETWORK_MANIFEST}" "network manifest"
  EXPECTED_CHAIN_ID=$(kv_value "${NETWORK_MANIFEST}" chain_id) || \
    die "network manifest must contain exactly one chain_id"
  EXPECTED_GENESIS_SHA256=$(kv_value "${NETWORK_MANIFEST}" genesis_sha256) || \
    die "network manifest must contain exactly one genesis_sha256"
  EXPECTED_VALIDATOR_NODE_ID=$(kv_value "${NETWORK_MANIFEST}" validator_node_id) || \
    die "network manifest must contain exactly one validator_node_id"
  EXPECTED_BINARY_SHA256=$(kv_value "${NETWORK_MANIFEST}" binary_sha256) || \
    die "network manifest must contain exactly one binary_sha256"

  [ "${EXPECTED_CHAIN_ID}" = "${REQUIRED_CHAIN_ID}" ] || \
    die "image manifest is for ${EXPECTED_CHAIN_ID}, expected ${REQUIRED_CHAIN_ID}"
  [[ "${EXPECTED_GENESIS_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "network manifest genesis_sha256 is invalid"
  [[ "${EXPECTED_VALIDATOR_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] || \
    die "network manifest validator_node_id is invalid"
  [[ "${EXPECTED_BINARY_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "network manifest binary_sha256 is invalid"
  [ "${EXPECTED_VALIDATOR_NODE_ID}" != "0000000000000000000000000000000000000000" ] || \
    die "network manifest still contains the placeholder validator node ID"
}

validate_release_binary() {
  local actual_hash
  require_regular_file "${BINARY}" "zeroned binary"
  [ -x "${BINARY}" ] || die "zeroned binary is not executable: ${BINARY}"
  actual_hash=$(sha256_file "${BINARY}") || die "could not hash zeroned binary"
  [ "${actual_hash}" = "${EXPECTED_BINARY_SHA256}" ] || \
    die "zeroned binary sha256 mismatch: got ${actual_hash}, expected ${EXPECTED_BINARY_SHA256}"
}

validate_genesis() {
  local genesis="$1" chain_id actual_hash
  require_regular_file "${genesis}" genesis
  chain_id=$(jq -er '.chain_id' "${genesis}") || die "genesis chain_id is missing"
  [ "${chain_id}" = "${EXPECTED_CHAIN_ID}" ] || \
    die "refusing genesis for ${chain_id}; expected ${EXPECTED_CHAIN_ID}"
  actual_hash=$(sha256_file "${genesis}") || die "could not hash genesis"
  [ "${actual_hash}" = "${EXPECTED_GENESIS_SHA256}" ] || \
    die "genesis sha256 mismatch: got ${actual_hash}, expected ${EXPECTED_GENESIS_SHA256}"
}

validate_signer_state() {
  local state_file="$1"
  [ -f "${state_file}" ] || die "priv_validator_state.json is missing"
  jq -e '
    (.height | tostring | test("^(0|[1-9][0-9]*)$")) and
    (.round | type == "number" and . >= 0 and floor == .) and
    (.step | type == "number" and . >= 0 and . <= 3 and floor == .)
  ' "${state_file}" >/dev/null || die "invalid priv_validator_state.json"
}

derive_keys() {
  local home="$1" stored_pubkey
  stored_pubkey=$(jq -er '.pub_key.value' "${home}/config/priv_validator_key.json") || \
    die "validator public key is missing"
  DERIVED_NODE_ID=$("${BINARY}" tendermint show-node-id --home "${home}" 2>/dev/null) || \
    die "node private key cannot be loaded by CometBFT"
  DERIVED_VALIDATOR_PUBKEY=$(
    "${BINARY}" tendermint show-validator --home "${home}" 2>/dev/null | jq -er '.key'
  ) || die "validator private key cannot be loaded by CometBFT"
  [ "${stored_pubkey}" = "${DERIVED_VALIDATOR_PUBKEY}" ] || \
    die "validator public key does not match its private key"
}

validator_pubkey_is_in_genesis() {
  local genesis="$1" pubkey="$2"
  jq -e --arg pubkey "${pubkey}" '
    [ .app_state.genutil.gen_txs[]?.body.messages[]?.pubkey.key ]
    | index($pubkey) != null
  ' "${genesis}" >/dev/null
}

validate_role_keys() {
  local home="$1" genesis="$2"
  derive_keys "${home}"
  if [ "${NODE_ROLE}" = "validator" ]; then
    [ "${DERIVED_NODE_ID}" = "${EXPECTED_VALIDATOR_NODE_ID}" ] || \
      die "validator node key derives ${DERIVED_NODE_ID}, expected ${EXPECTED_VALIDATOR_NODE_ID}"
    validator_pubkey_is_in_genesis "${genesis}" "${DERIVED_VALIDATOR_PUBKEY}" || \
      die "validator private key does not belong to the zerone-2 genesis validator set"
  else
    if validator_pubkey_is_in_genesis "${genesis}" "${DERIVED_VALIDATOR_PUBKEY}"; then
      die "edge node unexpectedly holds a zerone-2 genesis validator key"
    fi
  fi
}

configure_home() {
  local home="$1" config app
  local peers private_ids cors
  config="${home}/config/config.toml"
  app="${home}/config/app.toml"

  peers="${PERSISTENT_PEERS:-}"
  private_ids="${PRIVATE_PEER_IDS:-}"
  validate_peer_list PERSISTENT_PEERS "${peers}"
  validate_peer_ids "${private_ids}"
  validate_query_origin_profile
  cors="${CORS_ALLOWED_ORIGINS_JSON:-[]}"
  validate_cors "${cors}"
  cors=$(jq -c . <<< "${cors}")

  require_regular_file "${config}" config.toml
  require_regular_file "${app}" app.toml

  toml_set_root minimum-gas-prices '"1uzrn"' "${app}"
  toml_set_root query-gas-limit "\"${QUERY_GAS_LIMIT}\"" "${app}"
  toml_set_root pruning '"default"' "${app}"
  toml_set_root iavl-disable-fastnode true "${app}"
  toml_set_root priv_validator_laddr '""' "${config}"
  toml_set mempool max-txs 5000 "${app}"
  toml_set telemetry enabled true "${app}"
  toml_set telemetry prometheus-retention-time 60 "${app}"
  toml_set instrumentation prometheus true "${config}"
  toml_set instrumentation prometheus_listen_addr '":26660"' "${config}"
  toml_set rpc unsafe false "${config}"
  toml_set rpc max_request_batch_size 1 "${config}"
  toml_set rpc max_body_bytes "${RPC_MAX_BODY_BYTES}" "${config}"
  toml_set rpc max_header_bytes "${RPC_MAX_HEADER_BYTES}" "${config}"
  toml_set rpc grpc_laddr '""' "${config}"
  toml_set rpc pprof_laddr '""' "${config}"
  toml_set storage discard_abci_responses false "${config}"
  toml_set tx_index indexer '"kv"' "${config}"
  toml_set consensus timeout_propose '"2s"' "${config}"
  toml_set consensus timeout_commit '"2521ms"' "${config}"
  toml_set consensus skip_timeout_commit false "${config}"
  toml_set p2p seeds '""' "${config}"
  toml_set p2p persistent_peers "$(toml_quote "${peers}")" "${config}"
  toml_set p2p private_peer_ids "$(toml_quote "${private_ids}")" "${config}"
  toml_set p2p unconditional_peer_ids "$(toml_quote "${private_ids}")" "${config}"
  toml_set p2p addr_book_strict true "${config}"
  toml_set p2p allow_duplicate_ip false "${config}"
  toml_set p2p max_num_inbound_peers 40 "${config}"
  toml_set p2p max_num_outbound_peers 10 "${config}"

  if [ -n "${P2P_EXTERNAL_ADDRESS:-}" ]; then
    validate_safe_toml_string P2P_EXTERNAL_ADDRESS "${P2P_EXTERNAL_ADDRESS}"
    [[ "${P2P_EXTERNAL_ADDRESS}" =~ ^[^[:space:],@]+:[1-9][0-9]{0,4}$ ]] || \
      die "P2P_EXTERNAL_ADDRESS must use host:port without credentials"
    validate_host_port P2P_EXTERNAL_ADDRESS "${P2P_EXTERNAL_ADDRESS}"
  fi
  toml_set p2p external_address \
    "$(toml_quote "${P2P_EXTERNAL_ADDRESS:-}")" "${config}"

  if [ "${NODE_ROLE}" = "validator" ]; then
    # Fly .internal resolves to the Machine's IPv6 6PN address. The validator
    # has no public service, so bind P2P only to that private interface.
    toml_set p2p laddr '"tcp://fly-local-6pn:26656"' "${config}"
    toml_set rpc laddr '"tcp://127.0.0.1:26657"' "${config}"
    toml_set rpc cors_allowed_origins '[]' "${config}"
    toml_set rpc max_open_connections 50 "${config}"
    toml_set p2p pex false "${config}"
    toml_set p2p max_num_inbound_peers 10 "${config}"
    toml_set api enable false "${app}"
    toml_set api address '"tcp://127.0.0.1:1317"' "${app}"
    toml_set api swagger false "${app}"
    toml_set api enabled-unsafe-cors false "${app}"
    toml_set api rpc-read-timeout "${API_READ_TIMEOUT_SECONDS}" "${app}"
    toml_set api rpc-write-timeout "${API_WRITE_TIMEOUT_SECONDS}" "${app}"
    toml_set api rpc-max-body-bytes "${RPC_MAX_BODY_BYTES}" "${app}"
    toml_set grpc enable false "${app}"
    toml_set grpc address '"127.0.0.1:9090"' "${app}"
    toml_set grpc-web enable false "${app}"
    toml_set state-sync snapshot-interval 0 "${app}"
  else
    # The edge's public P2P service is reached through Fly Proxy over IPv4. RPC
    # and REST are distinct private origins and bind only to the Machine 6PN.
    toml_set p2p laddr '"tcp://0.0.0.0:26656"' "${config}"
    toml_set p2p pex true "${config}"
    toml_set api swagger false "${app}"
    toml_set api enabled-unsafe-cors false "${app}"
    toml_set api rpc-read-timeout "${API_READ_TIMEOUT_SECONDS}" "${app}"
    toml_set api rpc-write-timeout "${API_WRITE_TIMEOUT_SECONDS}" "${app}"
    toml_set api rpc-max-body-bytes "${RPC_MAX_BODY_BYTES}" "${app}"
    toml_set grpc-web enable false "${app}"
    toml_set state-sync snapshot-interval 1000 "${app}"
    toml_set state-sync snapshot-keep-recent 2 "${app}"

    if [ "${QUERY_ORIGIN_MODE}" = "true" ]; then
      toml_set rpc laddr '"tcp://fly-local-6pn:26657"' "${config}"
      toml_set rpc cors_allowed_origins "${cors}" "${config}"
      toml_set rpc max_open_connections 200 "${config}"
      toml_set rpc max_subscription_clients 50 "${config}"
      toml_set rpc max_subscriptions_per_client 5 "${config}"
      toml_set api enable true "${app}"
      toml_set api address '"tcp://fly-local-6pn:1317"' "${app}"
      toml_set api max-open-connections 200 "${app}"
      toml_set grpc enable false "${app}"
      toml_set grpc address '"127.0.0.1:9090"' "${app}"
      info "edge private query origin is explicitly enabled"
    else
      toml_set rpc laddr '"tcp://127.0.0.1:26657"' "${config}"
      toml_set rpc cors_allowed_origins '[]' "${config}"
      toml_set rpc max_open_connections 50 "${config}"
      toml_set rpc max_subscription_clients 10 "${config}"
      toml_set rpc max_subscriptions_per_client 2 "${config}"
      toml_set api enable false "${app}"
      toml_set api address '"tcp://127.0.0.1:1317"' "${app}"
      toml_set api max-open-connections 50 "${app}"
      toml_set grpc enable false "${app}"
      toml_set grpc address '"127.0.0.1:9090"' "${app}"
      info "edge private query origin is closed"
    fi
  fi
}

write_runtime_marker() {
  local home="$1" marker tmp
  marker="${home}/.runtime-initialized"
  [ ! -L "${marker}" ] || die "runtime marker must not be a symlink"
  tmp=$(mktemp "${home}/.runtime-initialized.tmp.XXXXXX") || \
    die "could not create runtime marker"
  {
    printf 'runtime_version=%s\n' "${RUNTIME_VERSION}"
    printf 'role=%s\n' "${NODE_ROLE}"
    printf 'chain_id=%s\n' "${EXPECTED_CHAIN_ID}"
    printf 'genesis_sha256=%s\n' "${EXPECTED_GENESIS_SHA256}"
    printf 'node_id=%s\n' "${DERIVED_NODE_ID}"
    printf 'validator_pubkey=%s\n' "${DERIVED_VALIDATOR_PUBKEY}"
  } > "${tmp}"
  chmod 600 "${tmp}"
  mv "${tmp}" "${marker}"
}

validate_runtime_marker() {
  local home="$1" marker
  marker="${home}/.runtime-initialized"
  [ ! -L "${marker}" ] || die "runtime marker must not be a symlink"
  [ -f "${marker}" ] || die "initialized volume is missing its runtime marker"
  [ "$(kv_value "${marker}" runtime_version)" = "${RUNTIME_VERSION}" ] || \
    die "runtime marker version mismatch"
  [ "$(kv_value "${marker}" role)" = "${NODE_ROLE}" ] || \
    die "volume role does not match NODE_ROLE"
  [ "$(kv_value "${marker}" chain_id)" = "${EXPECTED_CHAIN_ID}" ] || \
    die "volume chain ID does not match the image"
  [ "$(kv_value "${marker}" genesis_sha256)" = "${EXPECTED_GENESIS_SHA256}" ] || \
    die "volume genesis hash does not match the image"
  [ "$(kv_value "${marker}" node_id)" = "${DERIVED_NODE_ID}" ] || \
    die "volume node key changed after initialization"
  [ "$(kv_value "${marker}" validator_pubkey)" = "${DERIVED_VALIDATOR_PUBKEY}" ] || \
    die "volume validator key changed after initialization"
}

custody_env_present() {
  [ -n "${BOOTSTRAP_VALIDATOR_KEY_B64}" ] || \
    [ -n "${BOOTSTRAP_NODE_KEY_B64}" ] || \
    [ -n "${BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64}" ]
}

acquire_home_lock() {
  local parent legacy_lock_file
  parent=$(dirname "${HOME_DIR}")
  mkdir -p "${parent}"
  require_directory "${parent}" "ZERONE_HOME parent"
  legacy_lock_file="${HOME_DIR}.runtime.lock"
  [ ! -L "${legacy_lock_file}" ] || \
    die "runtime lock must not be a symlink: ${legacy_lock_file}"
  exec 9<"${parent}" || die "could not open runtime lock directory ${parent}"
  flock -n 9 || die "another process already owns ${HOME_DIR}"
}

initialize_volume() {
  local parent staging secret_dir validator_b64 node_b64
  parent=$(dirname "${HOME_DIR}")
  mkdir -p "${parent}"

  [ ! -L "${HOME_DIR}" ] || die "ZERONE_HOME must not be a symlink"
  if [ -e "${HOME_DIR}" ]; then
    require_directory "${HOME_DIR}" "ZERONE_HOME"
    # rmdir is both the emptiness check and a no-follow removal. Avoid listing a
    # path that a non-cooperating writer could replace with a symlink.
    rmdir "${HOME_DIR}" 2>/dev/null || \
      die "refusing partially initialized non-empty or replaced volume without runtime marker"
  fi

  staging=$(mktemp -d "${parent}/.zerone-2-init.XXXXXX")
  secret_dir=""
  cleanup_initialization() {
    [ -z "${secret_dir}" ] || rm -rf "${secret_dir}"
    [ -z "${staging}" ] || rm -rf "${staging}"
  }
  trap cleanup_initialization EXIT

  if [ "${NODE_ROLE}" = "validator" ]; then
    [ -n "${BOOTSTRAP_VALIDATOR_KEY_B64}" ] || die "empty validator volume requires VALIDATOR_KEY_B64"
    [ -n "${BOOTSTRAP_NODE_KEY_B64}" ] || die "empty validator volume requires NODE_KEY_B64"
    [ -z "${BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64}" ] || \
      die "legacy PRIV_VALIDATOR_KEY_B64 is forbidden; use VALIDATOR_KEY_B64"

    validator_b64="${BOOTSTRAP_VALIDATOR_KEY_B64}"
    node_b64="${BOOTSTRAP_NODE_KEY_B64}"
    unset BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64 \
      BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64
    secret_dir=$(mktemp -d "${parent}/.zerone-2-secrets.XXXXXX")
    printf '%s' "${validator_b64}" | decode_base64 > "${secret_dir}/priv_validator_key.json" || \
      die "VALIDATOR_KEY_B64 is not valid base64"
    printf '%s' "${node_b64}" | decode_base64 > "${secret_dir}/node_key.json" || \
      die "NODE_KEY_B64 is not valid base64"
    chmod 600 "${secret_dir}/priv_validator_key.json" "${secret_dir}/node_key.json"
    unset validator_b64 node_b64
  else
    custody_env_present && die "edge role rejects all validator custody inputs"
    unset BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64 \
      BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64
  fi

  "${BINARY}" init "${MONIKER:-zerone-2-${NODE_ROLE}}" \
    --chain-id "${EXPECTED_CHAIN_ID}" --default-denom uzrn \
    --home "${staging}" >/dev/null 2>&1 || die "zeroned init failed"
  cp "${PUBLIC_GENESIS}" "${staging}/config/genesis.json"

  if [ "${NODE_ROLE}" = "validator" ]; then
    install -m 600 "${secret_dir}/priv_validator_key.json" \
      "${staging}/config/priv_validator_key.json"
    install -m 600 "${secret_dir}/node_key.json" "${staging}/config/node_key.json"
    rm -rf "${secret_dir}"
    secret_dir=""
  fi

  validate_signer_state "${staging}/data/priv_validator_state.json"
  validate_role_keys "${staging}" "${staging}/config/genesis.json"
  configure_home "${staging}"
  chmod 600 "${staging}/config/node_key.json" \
    "${staging}/config/priv_validator_key.json" \
    "${staging}/data/priv_validator_state.json"
  write_runtime_marker "${staging}"

  require_directory "${staging}" "initialization staging directory"
  if [ -e "${HOME_DIR}" ] || [ -L "${HOME_DIR}" ]; then
    die "ZERONE_HOME appeared during atomic initialization publish"
  fi
  # Production is Linux with GNU coreutils. --no-target-directory prevents an
  # interposed directory from turning this into staging/HOME_DIR, while
  # --no-clobber leaves staging in place on any destination collision. GNU mv
  # reports a no-clobber collision as success, so the source-path check is the
  # authoritative proof that the rename actually happened.
  mv --no-target-directory --no-clobber -- "${staging}" "${HOME_DIR}" || \
    die "could not atomically publish initialized ZERONE_HOME"
  if [ -e "${staging}" ] || [ -L "${staging}" ]; then
    die "ZERONE_HOME appeared during atomic initialization publish"
  fi
  require_directory "${HOME_DIR}" "published ZERONE_HOME"
  require_regular_file "${HOME_DIR}/.runtime-initialized" "published runtime marker"
  staging=""
  trap - EXIT
  info "initialized ${NODE_ROLE} volume for ${EXPECTED_CHAIN_ID}"
}

resume_volume() {
  custody_env_present && \
    die "bootstrap custody inputs are forbidden on an initialized volume; remove them after first boot"
  [ ! -L "${HOME_DIR}/.runtime-initialized" ] || \
    die "runtime marker must not be a symlink"
  [ -f "${HOME_DIR}/.runtime-initialized" ] || \
    die "refusing unmarked or partially restored volume"
  require_directory "${HOME_DIR}/config" "config directory"
  require_directory "${HOME_DIR}/data" "data directory"
  require_regular_file "${HOME_DIR}/.runtime-initialized" "runtime marker"
  require_regular_file "${HOME_DIR}/config/node_key.json" "node key"
  require_regular_file "${HOME_DIR}/config/priv_validator_key.json" "validator key"
  require_regular_file "${HOME_DIR}/data/priv_validator_state.json" "validator signer state"
  validate_genesis "${HOME_DIR}/config/genesis.json"
  validate_signer_state "${HOME_DIR}/data/priv_validator_state.json"
  chmod 600 "${HOME_DIR}/config/node_key.json" \
    "${HOME_DIR}/config/priv_validator_key.json" \
    "${HOME_DIR}/data/priv_validator_state.json"
  validate_role_keys "${HOME_DIR}" "${HOME_DIR}/config/genesis.json"
  validate_runtime_marker "${HOME_DIR}"
  # Config lives on the persistent volume. Reapply every fixed role boundary and
  # the current peer/relay profile so stale or tampered settings cannot survive.
  configure_home "${HOME_DIR}"
  info "validated existing ${NODE_ROLE} volume"
}

case "${NODE_ROLE}" in
  validator|edge) ;;
  *) die "NODE_ROLE must be exactly validator or edge" ;;
esac

reject_daemon_env_overrides
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v flock >/dev/null 2>&1 || die "flock is required for process fencing"
[ ! -L "${HOME_DIR}" ] || die "ZERONE_HOME must not be a symlink"

acquire_home_lock
read_network_manifest
validate_release_binary
validate_genesis "${PUBLIC_GENESIS}"

if [ -f "${HOME_DIR}/.runtime-initialized" ]; then
  resume_volume
else
  initialize_volume
fi

# Defense in depth: never pass bootstrap custody material to the daemon.
unset VALIDATOR_KEY_B64 NODE_KEY_B64 PRIV_VALIDATOR_KEY_B64 \
  BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64 \
  BOOTSTRAP_LEGACY_VALIDATOR_KEY_B64

exec "${BINARY}" start --home "${HOME_DIR}" --minimum-gas-prices 1uzrn \
  --query-gas-limit "${QUERY_GAS_LIMIT}" --min-retain-blocks 0
