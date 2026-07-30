#!/usr/bin/env bash
# Fail-closed first boot and runtime policy for non-signing Fly full nodes.
#
# The image is built for one network. Runtime may select only a non-signing
# role: sentry or public-query. No private validator or P2P key may be supplied;
# zeroned creates fresh, zero-power local identities on an empty volume.
set -euo pipefail
umask 077

script_directory="${BASH_SOURCE[0]%/*}"
if [[ "${script_directory}" == "${BASH_SOURCE[0]}" ]]; then
  script_directory="."
fi
script_directory="$(cd -- "${script_directory}" && pwd -P)"
image_directory="${script_directory}/../share/zerone"
zeroned_binary="${script_directory}/zeroned"
nginx_binary="${script_directory}/nginx"
image_genesis="${image_directory}/genesis.json"
image_genesis_digest_file="${image_directory}/genesis.sha256"
image_chain_id_file="${image_directory}/chain-id"
image_binary_digest_file="${image_directory}/zeroned.sha256"
image_nginx_config="${image_directory}/public-edge-nginx.conf"

for dependency in awk base64 find id jq sha256sum; do
  if ! command -v "${dependency}" >/dev/null 2>&1; then
    echo "[full-node] required dependency is absent: ${dependency}" >&2
    exit 1
  fi
done

require_regular_file() {
  local path="$1"
  local description="$2"
  if [[ -L "${path}" || ! -f "${path}" ]]; then
    echo "[full-node] ${description} must be a regular non-symlink file" >&2
    exit 1
  fi
}

for tuple in \
  "${zeroned_binary}:image zeroned binary" \
  "${image_genesis}:image genesis" \
  "${image_genesis_digest_file}:image genesis digest" \
  "${image_chain_id_file}:image chain ID" \
  "${image_binary_digest_file}:image binary digest" \
  "${image_nginx_config}:image public-edge policy"; do
  require_regular_file "${tuple%%:*}" "${tuple#*:}"
done
if [[ ! -x "${zeroned_binary}" ]]; then
  echo "[full-node] image zeroned binary must be executable" >&2
  exit 1
fi

# These images create their own zero-power identities. Accepting any key input
# would turn a disposable network role into an undeclared custody surface.
for forbidden_variable in \
  PRIV_VALIDATOR_KEY_FILE \
  PRIV_VALIDATOR_KEY_B64 \
  PRIV_VALIDATOR_KEY_SHA256 \
  EXPECTED_PRIV_VALIDATOR_KEY_SHA256 \
  EXPECTED_VALIDATOR_ADDRESS \
  NODE_KEY_FILE \
  NODE_KEY_B64 \
  NODE_KEY_SHA256 \
  EXPECTED_NODE_KEY_SHA256 \
  EXPECTED_NODE_ID \
  VALIDATOR_MNEMONIC \
  NODE_MNEMONIC \
  MNEMONIC \
  SEED_PHRASE; do
  if [[ -v "${forbidden_variable}" ]]; then
    echo "[full-node] runtime-provided key material is forbidden: ${forbidden_variable}" >&2
    exit 1
  fi
done

chain_id="$(tr -d '\r\n' < "${image_chain_id_file}")"
expected_genesis_sha256="$(tr -d '\r\n' < "${image_genesis_digest_file}")"
expected_binary_sha256="$(tr -d '\r\n' < "${image_binary_digest_file}")"
if [[ ! "${chain_id}" =~ ^[A-Za-z0-9._-]+$ ]] ||
  [[ ! "${expected_genesis_sha256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${expected_binary_sha256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "[full-node] image chain or digest metadata is not canonical" >&2
  exit 1
fi
actual_genesis_sha256="$(sha256sum "${image_genesis}" | awk '{print $1}')"
actual_binary_sha256="$(sha256sum "${zeroned_binary}" | awk '{print $1}')"
nginx_config_sha256="$(sha256sum "${image_nginx_config}" | awk '{print $1}')"
if [[ "${actual_genesis_sha256}" != "${expected_genesis_sha256}" ]] ||
  [[ "${actual_binary_sha256}" != "${expected_binary_sha256}" ]]; then
  echo "[full-node] image genesis or zeroned digest mismatch" >&2
  exit 1
fi
if ! jq -e --arg chain_id "${chain_id}" '
  type == "object" and .chain_id == $chain_id
' "${image_genesis}" >/dev/null; then
  echo "[full-node] image genesis does not bind the image-frozen chain ID" >&2
  exit 1
fi
if "${zeroned_binary}" genesis validate "${image_genesis}" >/dev/null 2>&1; then
  :
elif "${zeroned_binary}" validate-genesis "${image_genesis}" >/dev/null 2>&1; then
  :
else
  echo "[full-node] image zeroned cannot decode the image-frozen genesis" >&2
  exit 1
fi

role="${ZERONE_NODE_ROLE:-}"
moniker="${ZERONE_MONIKER:-}"
validator_peer="${ZERONE_VALIDATOR_PEER:-}"
sentry_peers="${ZERONE_SENTRY_PEERS:-}"
external_p2p_address="${ZERONE_EXTERNAL_P2P_ADDRESS:-}"
expected_topology_sha256="${ZERONE_TOPOLOGY_SHA256:-}"
identity_ceremony="${ZERONE_IDENTITY_CEREMONY:-}"
home_directory="${ZERONE_HOME:-/data/.zeroned}"
data_root="${home_directory%/.zeroned}"

if [[ "${role}" != "sentry" && "${role}" != "public-query" ]]; then
  echo "[full-node] ZERONE_NODE_ROLE must be sentry or public-query" >&2
  exit 1
fi
if [[ ! "${moniker}" =~ ^[A-Za-z0-9]([A-Za-z0-9._-]{0,62}[A-Za-z0-9])?$ ]]; then
  echo "[full-node] moniker must be 1-64 canonical ASCII characters" >&2
  exit 1
fi
case "${home_directory}" in
  /*/.zeroned) ;;
  *)
    echo "[full-node] ZERONE_HOME must be an absolute .zeroned directory" >&2
    exit 1
    ;;
esac
case "/${home_directory#/}/" in
  *"//"* | *"/./"* | *"/../"*)
    echo "[full-node] ZERONE_HOME must be canonical" >&2
    exit 1
    ;;
esac
if [[ "${data_root}" == "/" || -z "${data_root}" ]] ||
  [[ -L "${data_root}" || ! -d "${data_root}" ]]; then
  echo "[full-node] the pre-mounted data root must be a regular non-symlink directory" >&2
  exit 1
fi
if [[ -n "${identity_ceremony}" && "${identity_ceremony}" != "generate-only" ]]; then
  echo "[full-node] ZERONE_IDENTITY_CEREMONY may only be generate-only" >&2
  exit 1
fi
if [[ -z "${identity_ceremony}" &&
  ! "${expected_topology_sha256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "[full-node] ZERONE_TOPOLOGY_SHA256 must be 64 lowercase hexadecimal characters" >&2
  exit 1
fi

require_pristine_data_root() {
  local lost_found="${data_root}/lost+found"
  local unexpected metadata

  unexpected="$(
    find "${data_root}" \
      -mindepth 1 \
      -maxdepth 1 \
      ! -name lost+found \
      -print -quit
  )"
  if [[ -n "${unexpected}" ]]; then
    echo "[full-node] fresh volume must be entirely empty of application data" >&2
    exit 1
  fi
  if [[ ! -e "${lost_found}" && ! -L "${lost_found}" ]]; then
    return
  fi
  if [[ -L "${lost_found}" || ! -d "${lost_found}" ]]; then
    echo "[full-node] filesystem lost+found must be a real directory" >&2
    exit 1
  fi
  if [[ -n "$(
    find "${lost_found}" -mindepth 1 -maxdepth 1 -print -quit
  )" ]]; then
    echo "[full-node] filesystem lost+found must be empty" >&2
    exit 1
  fi
  metadata="$(
    find "${lost_found}" \
      -mindepth 0 \
      -maxdepth 0 \
      -type d \
      -user "$(id -u)" \
      -group "$(id -g)" \
      -perm 0700 \
      -print -quit
  )"
  if [[ "${metadata}" != "${lost_found}" ]]; then
    echo "[full-node] filesystem lost+found owner or mode is not pristine" >&2
    exit 1
  fi
}

peer_ids=()
validate_peer() {
  local peer="$1"
  local description="$2"
  local node_id address host port known_id

  if [[ "${peer}" != *@* ]] || [[ "${peer}" == *[[:space:]]* ]]; then
    echo "[full-node] ${description} must be node-id@host:port without whitespace" >&2
    exit 1
  fi
  node_id="${peer%%@*}"
  address="${peer#*@}"
  host="${address%:*}"
  port="${address##*:}"
  if [[ ! "${node_id}" =~ ^[0-9a-f]{40}$ ]] ||
    [[ ! "${host}" =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ ]] ||
    [[ ! "${port}" =~ ^[1-9][0-9]{0,4}$ ]] ||
    (( 10#${port} > 65535 )); then
    echo "[full-node] ${description} is not a canonical peer address" >&2
    exit 1
  fi
  for known_id in "${peer_ids[@]:-}"; do
    if [[ "${known_id}" == "${node_id}" ]]; then
      echo "[full-node] peer IDs must be unique across the topology" >&2
      exit 1
    fi
  done
  peer_ids+=("${node_id}")
}

validate_peer_list() {
  local peers="$1"
  local description="$2"
  local minimum="$3"
  local peer
  local -a parsed_peers

  if [[ -z "${peers}" ]] || [[ "${peers}" == *, ]] || [[ "${peers}" == ,* ]] ||
    [[ "${peers}" == *",,"* ]] || [[ "${peers}" == *[[:space:]]* ]]; then
    echo "[full-node] ${description} must be a non-empty canonical comma-separated peer list" >&2
    exit 1
  fi
  IFS=',' read -r -a parsed_peers <<<"${peers}"
  if (( ${#parsed_peers[@]} < minimum )); then
    echo "[full-node] ${description} requires at least ${minimum} distinct peers" >&2
    exit 1
  fi
  for peer in "${parsed_peers[@]}"; do
    validate_peer "${peer}" "${description}"
  done
}

topology_payload() {
  printf '%s\n' \
    "schema=zerone.fly-full-node-topology/v1" \
    "chain_id=${chain_id}" \
    "genesis_sha256=${expected_genesis_sha256}" \
    "role=${role}" \
    "moniker=${moniker}" \
    "validator_peer=${validator_peer}" \
    "sentry_peers=${sentry_peers}" \
    "external_p2p_address=${external_p2p_address}"
}
if [[ -z "${identity_ceremony}" ]]; then
  if [[ "${role}" == "sentry" ]]; then
    validate_peer "${validator_peer}" "validator peer"
    validate_peer_list "${sentry_peers}" "sentry peers" 1
    if [[ ! "${external_p2p_address}" =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?:[1-9][0-9]{0,4}$ ]]; then
      echo "[full-node] sentry external P2P address must be canonical host:port" >&2
      exit 1
    fi
    external_p2p_port="${external_p2p_address##*:}"
    if (( 10#${external_p2p_port} > 65535 )); then
      echo "[full-node] sentry external P2P port is out of range" >&2
      exit 1
    fi
  else
    if [[ -n "${validator_peer}" ]] || [[ -n "${external_p2p_address}" ]]; then
      echo "[full-node] public-query nodes must not receive validator or external P2P addresses" >&2
      exit 1
    fi
    validate_peer_list "${sentry_peers}" "sentry peers" 2
  fi

  computed_topology_sha256="$(
    topology_payload |
      sha256sum |
      awk '{print $1}'
  )"
  if [[ "${computed_topology_sha256}" != "${expected_topology_sha256}" ]]; then
    echo "[full-node] runtime topology tuple does not match ZERONE_TOPOLOGY_SHA256" >&2
    exit 1
  fi
fi

config_directory="${home_directory}/config"
node_data_directory="${home_directory}/data"
genesis_path="${config_directory}/genesis.json"
config_path="${config_directory}/config.toml"
app_config_path="${config_directory}/app.toml"
validator_key_path="${config_directory}/priv_validator_key.json"
validator_state_path="${node_data_directory}/priv_validator_state.json"
node_key_path="${config_directory}/node_key.json"
role_manifest_path="${config_directory}/zerone-full-node-role.json"
identity_pending_path="${config_directory}/zerone-full-node-identity-pending.json"

set_toml_value() {
  local path="$1"
  local section="$2"
  local key="$3"
  local literal="$4"
  local temporary

  temporary="$(mktemp "${path}.tmp.XXXXXX")"
  if ! awk \
    -v target_section="${section}" \
    -v target_key="${key}" \
    -v replacement="${key} = ${literal}" '
      BEGIN {
        current_section = ""
        replacements = 0
      }
      /^[[:space:]]*\[[^]]+\][[:space:]]*$/ {
        current_section = $0
        sub(/^[[:space:]]*\[/, "", current_section)
        sub(/\][[:space:]]*$/, "", current_section)
      }
      {
        key_pattern = "^[[:space:]]*" target_key "[[:space:]]*="
        if (current_section == target_section && $0 ~ key_pattern) {
          print replacement
          replacements++
        } else {
          print
        }
      }
      END {
        if (replacements != 1) {
          exit 42
        }
      }
    ' "${path}" > "${temporary}"; then
    rm -f "${temporary}"
    echo "[full-node] expected exactly one TOML key ${section}.${key}" >&2
    exit 1
  fi
  chmod 0600 "${temporary}"
  mv -f "${temporary}" "${path}"
}

configure_role() {
  local validator_node_id="${validator_peer%%@*}"
  local sentry_node_ids
  local p2p_laddr rpc_laddr persistent_peers private_peer_ids
  local unconditional_peer_ids pex addr_book_strict max_inbound max_outbound
  local api_enabled grpc_enabled

  sentry_node_ids="$(
    tr ',' '\n' <<<"${sentry_peers}" |
      awk -F'@' 'BEGIN { ORS="" } { if (NR > 1) printf ","; printf "%s", $1 }'
  )"
  if [[ "${role}" == "sentry" ]]; then
    p2p_laddr="tcp://0.0.0.0:26656"
    rpc_laddr="tcp://127.0.0.1:26657"
    persistent_peers="${validator_peer},${sentry_peers}"
    private_peer_ids="${validator_node_id}"
    unconditional_peer_ids="${validator_node_id},${sentry_node_ids}"
    pex=true
    addr_book_strict=false
    max_inbound=40
    max_outbound=10
    api_enabled=false
    grpc_enabled=false
  else
    p2p_laddr="tcp://127.0.0.1:26656"
    rpc_laddr="tcp://127.0.0.1:26657"
    persistent_peers="${sentry_peers}"
    private_peer_ids=""
    unconditional_peer_ids="${sentry_node_ids}"
    pex=false
    addr_book_strict=true
    max_inbound=0
    # Comet excludes unconditional peers from these limits. The two pinned
    # sentries remain dialable while all opportunistic outbound peers are
    # disabled.
    max_outbound=0
    api_enabled=true
    # REST uses a loopback-only in-process gRPC endpoint. Fly and nginx expose
    # neither 9090 nor gRPC routes.
    grpc_enabled=true
  fi

  set_toml_value "${config_path}" "" priv_validator_laddr '""'
  set_toml_value "${config_path}" rpc laddr "\"${rpc_laddr}\""
  set_toml_value "${config_path}" rpc grpc_laddr '""'
  set_toml_value "${config_path}" rpc cors_allowed_origins '[]'
  set_toml_value "${config_path}" rpc unsafe false
  set_toml_value "${config_path}" rpc max_open_connections 100
  set_toml_value "${config_path}" rpc max_subscription_clients 1
  set_toml_value "${config_path}" rpc max_subscriptions_per_client 1
  set_toml_value "${config_path}" rpc max_request_batch_size 1
  set_toml_value "${config_path}" rpc max_body_bytes 262144
  set_toml_value "${config_path}" rpc pprof_laddr '""'

  set_toml_value "${config_path}" p2p laddr "\"${p2p_laddr}\""
  set_toml_value "${config_path}" p2p external_address "\"${external_p2p_address}\""
  set_toml_value "${config_path}" p2p seeds '""'
  set_toml_value "${config_path}" p2p persistent_peers "\"${persistent_peers}\""
  set_toml_value "${config_path}" p2p private_peer_ids "\"${private_peer_ids}\""
  set_toml_value "${config_path}" p2p unconditional_peer_ids "\"${unconditional_peer_ids}\""
  set_toml_value "${config_path}" p2p pex "${pex}"
  set_toml_value "${config_path}" p2p seed_mode false
  set_toml_value "${config_path}" p2p addr_book_strict "${addr_book_strict}"
  set_toml_value "${config_path}" p2p allow_duplicate_ip false
  set_toml_value "${config_path}" p2p max_num_inbound_peers "${max_inbound}"
  set_toml_value "${config_path}" p2p max_num_outbound_peers "${max_outbound}"
  set_toml_value "${config_path}" instrumentation prometheus false
  set_toml_value "${config_path}" instrumentation prometheus_listen_addr '":26660"'

  set_toml_value "${app_config_path}" "" minimum-gas-prices '"0.025uzrn"'
  set_toml_value "${app_config_path}" api enable "${api_enabled}"
  set_toml_value "${app_config_path}" api swagger false
  set_toml_value "${app_config_path}" api address '"tcp://127.0.0.1:1317"'
  set_toml_value "${app_config_path}" api max-open-connections 100
  set_toml_value "${app_config_path}" api rpc-max-body-bytes 262144
  set_toml_value "${app_config_path}" api enabled-unsafe-cors false
  set_toml_value "${app_config_path}" grpc enable "${grpc_enabled}"
  set_toml_value "${app_config_path}" grpc address '"127.0.0.1:9090"'
  set_toml_value "${app_config_path}" grpc max-recv-msg-size '"1048576"'
  set_toml_value "${app_config_path}" grpc max-send-msg-size '"4194304"'
  set_toml_value "${app_config_path}" grpc-web enable false
  set_toml_value "${app_config_path}" telemetry enabled false
}

derive_image_genesis_validator_address() {
  local public_key
  public_key="$(
    jq -er '
      [
        .app_state.genutil.gen_txs[]?.body.messages[]? |
        select(.["@type"] == "/cosmos.staking.v1beta1.MsgCreateValidator") |
        .pubkey |
        select(.["@type"] == "/cosmos.crypto.ed25519.PubKey") |
        .key
      ] |
      if length == 1 then .[0] else error("expected one genesis validator") end
    ' "${image_genesis}"
  )"
  ed25519_address_from_base64 "${public_key}" "genesis validator public key"
}

validate_managed_files() {
  local tuple
  for tuple in \
    "${config_directory}:config directory" \
    "${node_data_directory}:data directory"; do
    if [[ -L "${tuple%%:*}" || ! -d "${tuple%%:*}" ]]; then
      echo "[full-node] ${tuple#*:} must be a regular non-symlink directory" >&2
      exit 1
    fi
  done
  for tuple in \
    "${genesis_path}:installed genesis" \
    "${config_path}:Comet configuration" \
    "${app_config_path}:application configuration" \
    "${validator_key_path}:local zero-power consensus key" \
    "${validator_state_path}:local consensus signing state" \
    "${node_key_path}:local P2P node key"; do
    require_regular_file "${tuple%%:*}" "${tuple#*:}"
  done
}

validate_installed_genesis() {
  local digest
  digest="$(sha256sum "${genesis_path}" | awk '{print $1}')"
  if [[ "${digest}" != "${expected_genesis_sha256}" ]] ||
    ! jq -e --arg chain_id "${chain_id}" \
      'type == "object" and .chain_id == $chain_id' \
      "${genesis_path}" >/dev/null; then
    echo "[full-node] installed genesis drifted from the image-frozen network" >&2
    exit 1
  fi
}

ed25519_address_from_base64() {
  local public_key="$1"
  local description="$2"
  local temporary address

  temporary="$(mktemp)"
  if ! printf '%s' "${public_key}" | base64 -d > "${temporary}" ||
    [[ "$(wc -c < "${temporary}" | tr -d '[:space:]')" != "32" ]]; then
    rm -f "${temporary}"
    echo "[full-node] ${description} is not canonical Ed25519" >&2
    exit 1
  fi
  address="$(
    sha256sum "${temporary}" |
      awk '{print toupper(substr($1, 1, 40))}'
  )"
  rm -f "${temporary}"
  printf '%s\n' "${address}"
}

local_node_id=""
local_consensus_address=""
load_local_identities() {
  local stored_address public_key derived_address

  local_node_id="$(
    "${zeroned_binary}" comet show-node-id --home "${home_directory}" |
      tr -d '\r\n'
  )"
  if ! jq -e '
    type == "object" and
    (.address | type == "string" and test("^[0-9A-F]{40}$")) and
    .pub_key.type == "tendermint/PubKeyEd25519" and
    (.pub_key.value | type == "string")
  ' "${validator_key_path}" >/dev/null; then
    echo "[full-node] generated local consensus identity is not canonical" >&2
    exit 1
  fi
  stored_address="$(jq -er '.address' "${validator_key_path}")"
  public_key="$(jq -er '.pub_key.value' "${validator_key_path}")"
  derived_address="$(
    ed25519_address_from_base64 "${public_key}" "local consensus public key"
  )"
  if [[ "${stored_address}" != "${derived_address}" ]]; then
    echo "[full-node] local consensus address does not match its public key" >&2
    exit 1
  fi
  local_consensus_address="${stored_address}"
  if [[ ! "${local_node_id}" =~ ^[0-9a-f]{40}$ ]]; then
    echo "[full-node] generated local P2P identity is not canonical" >&2
    exit 1
  fi
}

seal_role_manifest() {
  local genesis_validator_address="$1"
  local config_sha256 app_config_sha256 validator_key_sha256
  local validator_state_sha256 node_key_sha256 manifest_temporary
  config_sha256="$(sha256sum "${config_path}" | awk '{print $1}')"
  app_config_sha256="$(sha256sum "${app_config_path}" | awk '{print $1}')"
  validator_key_sha256="$(sha256sum "${validator_key_path}" | awk '{print $1}')"
  validator_state_sha256="$(sha256sum "${validator_state_path}" | awk '{print $1}')"
  node_key_sha256="$(sha256sum "${node_key_path}" | awk '{print $1}')"
  manifest_temporary="$(mktemp "${config_directory}/.zerone-full-node-role.json.tmp.XXXXXX")"
  jq -n \
    --arg schema "zerone.fly-full-node-role/v1" \
    --arg chain_id "${chain_id}" \
    --arg genesis_sha256 "${expected_genesis_sha256}" \
    --arg role "${role}" \
    --arg moniker "${moniker}" \
    --arg topology_sha256 "${expected_topology_sha256}" \
    --arg validator_peer "${validator_peer}" \
    --arg sentry_peers "${sentry_peers}" \
    --arg external_p2p_address "${external_p2p_address}" \
    --arg node_id "${local_node_id}" \
    --arg consensus_address "${local_consensus_address}" \
    --arg genesis_validator_address "${genesis_validator_address}" \
    --arg config_sha256 "${config_sha256}" \
    --arg app_config_sha256 "${app_config_sha256}" \
    --arg validator_key_sha256 "${validator_key_sha256}" \
    --arg validator_state_sha256 "${validator_state_sha256}" \
    --arg node_key_sha256 "${node_key_sha256}" \
    --arg binary_sha256 "${expected_binary_sha256}" \
    --arg nginx_config_sha256 "${nginx_config_sha256}" '
      {
        schema: $schema,
        chain_id: $chain_id,
        genesis_sha256: $genesis_sha256,
        role: $role,
        moniker: $moniker,
        topology_sha256: $topology_sha256,
        validator_peer: $validator_peer,
        sentry_peers: $sentry_peers,
        external_p2p_address: $external_p2p_address,
        node_id: $node_id,
        consensus_address: $consensus_address,
        genesis_validator_address: $genesis_validator_address,
        config_sha256: $config_sha256,
        app_config_sha256: $app_config_sha256,
        validator_key_sha256: $validator_key_sha256,
        validator_state_sha256: $validator_state_sha256,
        node_key_sha256: $node_key_sha256,
        binary_sha256: $binary_sha256,
        nginx_config_sha256: $nginx_config_sha256
      }
    ' > "${manifest_temporary}"
  chmod 0600 "${manifest_temporary}"
  mv -f "${manifest_temporary}" "${role_manifest_path}"
}

verify_digest_fields() {
  local manifest_path="$1"
  local tuple managed_path remaining digest_field description
  local expected_digest actual_digest

  for tuple in \
    "${config_path}:config_sha256:Comet configuration" \
    "${app_config_path}:app_config_sha256:application configuration" \
    "${validator_key_path}:validator_key_sha256:local consensus key" \
    "${validator_state_path}:validator_state_sha256:local signing state" \
    "${node_key_path}:node_key_sha256:local P2P node key"; do
    managed_path="${tuple%%:*}"
    remaining="${tuple#*:}"
    digest_field="${remaining%%:*}"
    description="${remaining#*:}"
    expected_digest="$(
      jq -er --arg field "${digest_field}" '.[$field]' "${manifest_path}"
    )"
    actual_digest="$(sha256sum "${managed_path}" | awk '{print $1}')"
    if [[ "${actual_digest}" != "${expected_digest}" ]]; then
      echo "[full-node] ${description} drift detected" >&2
      exit 1
    fi
  done
}

initialize_identity_files() {
  echo "[full-node] generating fresh non-validator identity for ${role}/${chain_id}"
  "${zeroned_binary}" init "${moniker}" \
    --chain-id "${chain_id}" \
    --default-denom uzrn \
    --home "${home_directory}" >/dev/null 2>&1
  validate_managed_files
  install -m 0600 "${image_genesis}" "${genesis_path}"
  validate_installed_genesis
  load_local_identities
}

genesis_validator_address="$(derive_image_genesis_validator_address)"

if [[ -n "${identity_ceremony}" ]]; then
  if [[ -e "${home_directory}" || -L "${home_directory}" ]]; then
    echo "[full-node] identity ceremony requires an entirely empty pre-mounted encrypted volume" >&2
    exit 1
  fi
  require_pristine_data_root
  initialize_identity_files
  if [[ "${local_consensus_address}" == "${genesis_validator_address}" ]]; then
    echo "[full-node] generated local consensus identity equals the genesis validator" >&2
    exit 1
  fi

  config_sha256="$(sha256sum "${config_path}" | awk '{print $1}')"
  app_config_sha256="$(sha256sum "${app_config_path}" | awk '{print $1}')"
  validator_key_sha256="$(sha256sum "${validator_key_path}" | awk '{print $1}')"
  validator_state_sha256="$(sha256sum "${validator_state_path}" | awk '{print $1}')"
  node_key_sha256="$(sha256sum "${node_key_path}" | awk '{print $1}')"
  pending_temporary="$(mktemp "${config_directory}/.zerone-full-node-identity-pending.json.tmp.XXXXXX")"
  jq -n \
    --arg schema "zerone.fly-full-node-identity-pending/v1" \
    --arg chain_id "${chain_id}" \
    --arg genesis_sha256 "${expected_genesis_sha256}" \
    --arg role "${role}" \
    --arg moniker "${moniker}" \
    --arg node_id "${local_node_id}" \
    --arg consensus_address "${local_consensus_address}" \
    --arg genesis_validator_address "${genesis_validator_address}" \
    --arg config_sha256 "${config_sha256}" \
    --arg app_config_sha256 "${app_config_sha256}" \
    --arg validator_key_sha256 "${validator_key_sha256}" \
    --arg validator_state_sha256 "${validator_state_sha256}" \
    --arg node_key_sha256 "${node_key_sha256}" \
    --arg binary_sha256 "${expected_binary_sha256}" \
    --arg nginx_config_sha256 "${nginx_config_sha256}" '
      {
        schema: $schema,
        chain_id: $chain_id,
        genesis_sha256: $genesis_sha256,
        role: $role,
        moniker: $moniker,
        node_id: $node_id,
        consensus_address: $consensus_address,
        genesis_validator_address: $genesis_validator_address,
        config_sha256: $config_sha256,
        app_config_sha256: $app_config_sha256,
        validator_key_sha256: $validator_key_sha256,
        validator_state_sha256: $validator_state_sha256,
        node_key_sha256: $node_key_sha256,
        binary_sha256: $binary_sha256,
        nginx_config_sha256: $nginx_config_sha256
      }
    ' > "${pending_temporary}"
  chmod 0600 "${pending_temporary}"
  mv -f "${pending_temporary}" "${identity_pending_path}"
  printf '%s\n' \
    "[full-node] identity ceremony complete; daemon was not started" \
    "  node_id=${local_node_id}" \
    "  local_zero_power_consensus_address=${local_consensus_address}" \
    "  genesis_validator_address=${genesis_validator_address}"
  exit 0
fi

fresh_volume=false
pending_identity=false
if [[ ! -e "${role_manifest_path}" && ! -L "${role_manifest_path}" ]]; then
  if [[ -e "${identity_pending_path}" || -L "${identity_pending_path}" ]]; then
    pending_identity=true
  else
    fresh_volume=true
  fi
fi

if [[ "${fresh_volume}" == true ]]; then
  if [[ -e "${home_directory}" || -L "${home_directory}" ]]; then
    echo "[full-node] first boot requires an entirely empty pre-mounted encrypted volume" >&2
    exit 1
  fi
  require_pristine_data_root

  initialize_identity_files
  configure_role
  if [[ "${local_consensus_address}" == "${genesis_validator_address}" ]]; then
    echo "[full-node] generated local consensus identity equals the genesis validator" >&2
    exit 1
  fi
  seal_role_manifest "${genesis_validator_address}"
elif [[ "${pending_identity}" == true ]]; then
  require_regular_file "${identity_pending_path}" "pending identity ceremony manifest"
  validate_managed_files
  validate_installed_genesis
  load_local_identities
  if [[ -n "$(
    find "${node_data_directory}" \
      -mindepth 1 \
      -maxdepth 1 \
      ! -name priv_validator_state.json \
      -print -quit
  )" ]]; then
    echo "[full-node] pending identity volume contains block or signing data" >&2
    exit 1
  fi
  if ! jq -e \
    --arg chain_id "${chain_id}" \
    --arg genesis_sha256 "${expected_genesis_sha256}" \
    --arg role "${role}" \
    --arg moniker "${moniker}" \
    --arg node_id "${local_node_id}" \
    --arg consensus_address "${local_consensus_address}" \
    --arg genesis_validator_address "${genesis_validator_address}" \
    --arg binary_sha256 "${expected_binary_sha256}" \
    --arg nginx_config_sha256 "${nginx_config_sha256}" '
      type == "object" and
      .schema == "zerone.fly-full-node-identity-pending/v1" and
      .chain_id == $chain_id and
      .genesis_sha256 == $genesis_sha256 and
      .role == $role and
      .moniker == $moniker and
      .node_id == $node_id and
      .consensus_address == $consensus_address and
      .genesis_validator_address == $genesis_validator_address and
      .binary_sha256 == $binary_sha256 and
      .nginx_config_sha256 == $nginx_config_sha256 and
      .consensus_address != .genesis_validator_address
    ' "${identity_pending_path}" >/dev/null; then
    echo "[full-node] pending identity ceremony tuple drift detected" >&2
    exit 1
  fi
  verify_digest_fields "${identity_pending_path}"
  configure_role
  seal_role_manifest "${genesis_validator_address}"
  rm -f "${identity_pending_path}"
else
  require_regular_file "${role_manifest_path}" "persisted role manifest"
  validate_managed_files
  validate_installed_genesis
  load_local_identities

  if ! jq -e \
    --arg chain_id "${chain_id}" \
    --arg genesis_sha256 "${expected_genesis_sha256}" \
    --arg role "${role}" \
    --arg moniker "${moniker}" \
    --arg topology_sha256 "${expected_topology_sha256}" \
    --arg validator_peer "${validator_peer}" \
    --arg sentry_peers "${sentry_peers}" \
    --arg external_p2p_address "${external_p2p_address}" \
    --arg node_id "${local_node_id}" \
    --arg consensus_address "${local_consensus_address}" \
    --arg genesis_validator_address "${genesis_validator_address}" \
    --arg binary_sha256 "${expected_binary_sha256}" \
    --arg nginx_config_sha256 "${nginx_config_sha256}" '
      type == "object" and
      .schema == "zerone.fly-full-node-role/v1" and
      .chain_id == $chain_id and
      .genesis_sha256 == $genesis_sha256 and
      .role == $role and
      .moniker == $moniker and
      .topology_sha256 == $topology_sha256 and
      .validator_peer == $validator_peer and
      .sentry_peers == $sentry_peers and
      .external_p2p_address == $external_p2p_address and
      .node_id == $node_id and
      .consensus_address == $consensus_address and
      .genesis_validator_address == $genesis_validator_address and
      .binary_sha256 == $binary_sha256 and
      .nginx_config_sha256 == $nginx_config_sha256 and
      .consensus_address != .genesis_validator_address and
      (.config_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
      (.app_config_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
      (.validator_key_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
      (.validator_state_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
      (.node_key_sha256 | type == "string" and test("^[0-9a-f]{64}$"))
    ' "${role_manifest_path}" >/dev/null; then
    echo "[full-node] role, peer, identity, image, or topology drift detected" >&2
    exit 1
  fi

  verify_digest_fields "${role_manifest_path}"
  if [[ -e "${identity_pending_path}" || -L "${identity_pending_path}" ]]; then
    rm -f "${identity_pending_path}"
  fi
fi

echo "[full-node] role=${role} chain=${chain_id} node_id=${local_node_id}"

if [[ "${role}" == "sentry" ]]; then
  exec "${zeroned_binary}" start \
    --home "${home_directory}" \
    --minimum-gas-prices 0.025uzrn
fi

if [[ ! -x "${nginx_binary}" || -L "${nginx_binary}" && ! -e "${nginx_binary}" ]]; then
  echo "[full-node] image nginx launcher is unavailable" >&2
  exit 1
fi

node_pid=""
nginx_pid=""
# shellcheck disable=SC2329 # invoked indirectly by the EXIT trap
cleanup_children() {
  set +e
  if [[ -n "${nginx_pid}" ]]; then
    kill -TERM "${nginx_pid}" 2>/dev/null
  fi
  if [[ -n "${node_pid}" ]]; then
    kill -TERM "${node_pid}" 2>/dev/null
  fi
  wait "${nginx_pid}" 2>/dev/null
  wait "${node_pid}" 2>/dev/null
}
trap cleanup_children EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

"${zeroned_binary}" start \
  --home "${home_directory}" \
  --minimum-gas-prices 0.025uzrn &
node_pid=$!
"${nginx_binary}" \
  -c "${image_nginx_config}" \
  -g 'daemon off;' &
nginx_pid=$!

set +e
wait -n "${node_pid}" "${nginx_pid}"
first_exit_status=$?
set -e
echo "[full-node] public-query subprocess exited; failing closed" >&2
if (( first_exit_status == 0 )); then
  exit 1
fi
exit "${first_exit_status}"
