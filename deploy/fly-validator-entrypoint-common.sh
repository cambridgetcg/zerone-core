#!/usr/bin/env bash
# First-boot seeder shared by the immutable Fly validator profiles.
# Network wrappers provide only public, image-frozen chain parameters.
set -euo pipefail
umask 077

: "${ZERONE_CHAIN_ID:?ZERONE_CHAIN_ID must be fixed by the network image}"
: "${ZERONE_SEED:?ZERONE_SEED must be fixed by the network image}"
: "${ZERONE_DEFAULT_MONIKER:?ZERONE_DEFAULT_MONIKER must be fixed by the network image}"
: "${ZERONE_GENESIS_SHA256:?ZERONE_GENESIS_SHA256 must be fixed by the network image}"
: "${EXPECTED_VALIDATOR_ADDRESS:?EXPECTED_VALIDATOR_ADDRESS must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_ID:?EXPECTED_NODE_ID must come from the reviewed identity manifest}"
: "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256:?EXPECTED_PRIV_VALIDATOR_KEY_SHA256 must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_KEY_SHA256:?EXPECTED_NODE_KEY_SHA256 must come from the reviewed identity manifest}"

HOME_DIR="${ZERONE_HOME:-/data/.zeroned}"
SEED="${ZERONE_SEED}"
expected_genesis_digest="${ZERONE_GENESIS_SHA256}"
unset ZERONE_GENESIS_SHA256
GENESIS_PATH="${HOME_DIR}/config/genesis.json"
VALIDATOR_KEY_PATH="${HOME_DIR}/config/priv_validator_key.json"
VALIDATOR_STATE_PATH="${HOME_DIR}/data/priv_validator_state.json"
NODE_KEY_PATH="${HOME_DIR}/config/node_key.json"
fresh_volume=false
case "${HOME_DIR}" in
  /*) ;;
  *)
    echo "[entrypoint] validator home must be an absolute canonical non-root path" >&2
    exit 1
    ;;
esac
case "/${HOME_DIR#/}/" in
  *"//"* | *"/./"* | *"/../"*)
    echo "[entrypoint] validator home must be an absolute canonical non-root path" >&2
    exit 1
    ;;
esac
if [[ ! -f "${GENESIS_PATH}" ]]; then
  fresh_volume=true
fi

# Capture runtime credentials into non-exported shell variables, then remove
# every key-source variable from the environment before invoking any child
# process. In particular, base64 private keys must never reach zeroned.
validator_key_file="${PRIV_VALIDATOR_KEY_FILE:-}"
validator_key_b64="${PRIV_VALIDATOR_KEY_B64:-}"
validator_key_digest="${PRIV_VALIDATOR_KEY_SHA256:-}"
node_key_file="${NODE_KEY_FILE:-}"
node_key_b64="${NODE_KEY_B64:-}"
node_key_digest="${NODE_KEY_SHA256:-}"
external_address="${EXTERNAL_ADDRESS:-}"
expected_validator_address="${EXPECTED_VALIDATOR_ADDRESS}"
expected_node_id="${EXPECTED_NODE_ID}"
expected_validator_key_digest="${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}"
expected_node_key_digest="${EXPECTED_NODE_KEY_SHA256}"
unset PRIV_VALIDATOR_KEY_FILE PRIV_VALIDATOR_KEY_B64 PRIV_VALIDATOR_KEY_SHA256
unset NODE_KEY_FILE NODE_KEY_B64 NODE_KEY_SHA256
unset EXTERNAL_ADDRESS
unset EXPECTED_VALIDATOR_ADDRESS EXPECTED_NODE_ID
unset EXPECTED_PRIV_VALIDATOR_KEY_SHA256 EXPECTED_NODE_KEY_SHA256

if [[ ! "${expected_validator_address}" =~ ^[0-9A-F]{40}$ ]] ||
  [[ ! "${expected_node_id}" =~ ^[0-9a-f]{40}$ ]] ||
  [[ ! "${expected_validator_key_digest}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${expected_node_key_digest}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "[entrypoint] reviewed validator address, node ID, and key digests are not canonical" >&2
  exit 1
fi
if [[ "${validator_key_digest}" != "${expected_validator_key_digest}" ]] ||
  [[ "${node_key_digest}" != "${expected_node_key_digest}" ]]; then
  echo "[entrypoint] runtime key digests do not match the independently reviewed identity manifest" >&2
  exit 1
fi
if [[ -n "${external_address}" ]]; then
  if [[ ! "${external_address}" =~ ^[A-Za-z0-9.-]+:([1-9][0-9]{0,4})$ ]] ||
    (( 10#${BASH_REMATCH[1]:-0} > 65535 )); then
    echo "[entrypoint] external P2P address must be a canonical DNS/IPv4 host and TCP port" >&2
    exit 1
  fi
fi

if [[ -z "${validator_key_file}" && -z "${validator_key_b64}" ]]; then
  echo "[entrypoint] validator signing key is required via PRIV_VALIDATOR_KEY_FILE or PRIV_VALIDATOR_KEY_B64" >&2
  exit 1
fi
if [[ -z "${node_key_file}" && -z "${node_key_b64}" ]]; then
  echo "[entrypoint] P2P node key is required via NODE_KEY_FILE or NODE_KEY_B64" >&2
  exit 1
fi
for dependency in base64 cmp dd jq openssl sha256sum; do
  if ! command -v "${dependency}" >/dev/null 2>&1; then
    echo "[entrypoint] key validation requires ${dependency}" >&2
    exit 1
  fi
done

validate_genesis() {
  local genesis_path="$1"
  local description="$2"
  local actual_digest
  if [[ -L "${genesis_path}" || ! -f "${genesis_path}" ]]; then
    echo "[entrypoint] ${description} must be a regular non-symlink file" >&2
    exit 1
  fi
  actual_digest="$(sha256sum "${genesis_path}" | awk '{print $1}')"
  if [[ "${actual_digest}" != "${expected_genesis_digest}" ]]; then
    echo "[entrypoint] ${description} does not match the image-frozen genesis SHA-256" >&2
    exit 1
  fi
  if ! jq -e --arg chain_id "${ZERONE_CHAIN_ID}" '
    type == "object" and .chain_id == $chain_id
  ' "${genesis_path}" >/dev/null; then
    echo "[entrypoint] ${description} chain ID does not match the image-frozen network" >&2
    exit 1
  fi
}
if [[ ! "${expected_genesis_digest}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "[entrypoint] image-frozen genesis digest must be 64 lowercase SHA-256 hex characters" >&2
  exit 1
fi
validate_genesis "${SEED}/genesis.json" "image genesis"

candidate_dir="$(mktemp -d /tmp/zerone-key-material.XXXXXX)"
chmod 0700 "${candidate_dir}"
cleanup_candidate_material() {
  find "${candidate_dir}" -mindepth 1 -maxdepth 1 -type f -exec rm -f -- {} +
  rmdir "${candidate_dir}" 2>/dev/null || true
}
trap cleanup_candidate_material EXIT

validate_ed25519_private() {
  local private_bytes="$1"
  local public_bytes="${2:-}"
  local description="$3"
  local derived_output="${4:-}"
  local private_der derived_der derived_public embedded_public

  if [[ "$(wc -c < "${private_bytes}" | tr -d '[:space:]')" != "64" ]]; then
    echo "[entrypoint] ${description}: Ed25519 private key must decode to exactly 64 bytes" >&2
    exit 1
  fi

  private_der="$(mktemp "${candidate_dir}/ed25519-private.XXXXXX")"
  derived_der="$(mktemp "${candidate_dir}/ed25519-public-der.XXXXXX")"
  derived_public="$(mktemp "${candidate_dir}/ed25519-public.XXXXXX")"
  embedded_public="$(mktemp "${candidate_dir}/ed25519-embedded.XXXXXX")"
  # PKCS#8 DER prefix for a raw 32-byte Ed25519 seed.
  {
    printf '\060\056\002\001\000\060\005\006\003\053\145\160\004\042\004\040'
    dd if="${private_bytes}" bs=32 count=1 2>/dev/null
  } > "${private_der}"
  if ! openssl pkey -inform DER -in "${private_der}" -pubout -outform DER \
    -out "${derived_der}" 2>/dev/null; then
    echo "[entrypoint] ${description}: Ed25519 private seed is invalid" >&2
    exit 1
  fi
  tail -c 32 "${derived_der}" > "${derived_public}"
  tail -c 32 "${private_bytes}" > "${embedded_public}"
  if ! cmp -s "${derived_public}" "${embedded_public}"; then
    echo "[entrypoint] ${description}: Ed25519 private key public suffix does not match its seed" >&2
    exit 1
  fi
  if [[ -n "${public_bytes}" ]] && ! cmp -s "${derived_public}" "${public_bytes}"; then
    echo "[entrypoint] ${description}: Ed25519 public key does not match the private key" >&2
    exit 1
  fi
  if [[ -n "${derived_output}" ]]; then
    install -m 0600 "${derived_public}" "${derived_output}"
  fi
}

validate_validator_key() {
  local key_path="$1"
  local description="$2"
  local public_bytes private_bytes address derived_address

  if ! jq -e '
    type == "object" and
    (.address | type == "string" and test("^[0-9A-F]{40}$")) and
    (.pub_key.type == "tendermint/PubKeyEd25519") and
    (.pub_key.value | type == "string" and length > 0) and
    (.priv_key.type == "tendermint/PrivKeyEd25519") and
    (.priv_key.value | type == "string" and length > 0)
  ' "${key_path}" >/dev/null; then
    echo "[entrypoint] ${description}: validator JSON key schema is invalid" >&2
    exit 1
  fi

  public_bytes="$(mktemp "${candidate_dir}/validator-public.XXXXXX")"
  private_bytes="$(mktemp "${candidate_dir}/validator-private.XXXXXX")"
  if ! jq -er '.pub_key.value' "${key_path}" | base64 -d > "${public_bytes}" ||
    [[ "$(wc -c < "${public_bytes}" | tr -d '[:space:]')" != "32" ]]; then
    echo "[entrypoint] ${description}: Ed25519 public key must be valid base64 for exactly 32 bytes" >&2
    exit 1
  fi
  if ! jq -er '.priv_key.value' "${key_path}" | base64 -d > "${private_bytes}"; then
    echo "[entrypoint] ${description}: Ed25519 private key is not valid base64" >&2
    exit 1
  fi
  validate_ed25519_private "${private_bytes}" "${public_bytes}" "${description}"

  address="$(jq -er '.address' "${key_path}")"
  derived_address="$(
    sha256sum "${public_bytes}" |
      awk '{ print toupper(substr($1, 1, 40)) }'
  )"
  if [[ "${address}" != "${derived_address}" ]]; then
    echo "[entrypoint] ${description}: validator address does not match the Ed25519 public key" >&2
    exit 1
  fi
  if [[ "${derived_address}" != "${expected_validator_address}" ]]; then
    echo "[entrypoint] ${description}: derived validator address does not match the reviewed identity manifest" >&2
    exit 1
  fi
}

validate_node_key() {
  local key_path="$1"
  local description="$2"
  local private_bytes public_bytes derived_node_id

  if ! jq -e '
    type == "object" and
    (.priv_key.type == "tendermint/PrivKeyEd25519") and
    (.priv_key.value | type == "string" and length > 0)
  ' "${key_path}" >/dev/null; then
    echo "[entrypoint] ${description}: node JSON key schema is invalid" >&2
    exit 1
  fi
  private_bytes="$(mktemp "${candidate_dir}/node-private.XXXXXX")"
  if ! jq -er '.priv_key.value' "${key_path}" | base64 -d > "${private_bytes}"; then
    echo "[entrypoint] ${description}: Ed25519 private key is not valid base64" >&2
    exit 1
  fi
  public_bytes="$(mktemp "${candidate_dir}/node-public.XXXXXX")"
  validate_ed25519_private "${private_bytes}" "" "${description}" "${public_bytes}"
  derived_node_id="$(
    sha256sum "${public_bytes}" |
      awk '{ print tolower(substr($1, 1, 40)) }'
  )"
  if [[ "${derived_node_id}" != "${expected_node_id}" ]]; then
    echo "[entrypoint] ${description}: derived P2P node ID does not match the reviewed identity manifest" >&2
    exit 1
  fi
}

prepare_runtime_key() {
  local file_path="$1"
  local encoded="$2"
  local expected_digest="$3"
  local description="$4"
  local digest_variable="$5"
  local key_kind="$6"
  local temporary="$7"
  local actual_digest

  if [[ -n "${file_path}" && -n "${encoded}" ]]; then
    echo "[entrypoint] ${description}: configure a mounted file or base64 secret, not both" >&2
    exit 1
  fi
  if [[ ! "${expected_digest}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "[entrypoint] ${description}: ${digest_variable} must pin 64 lowercase SHA-256 hex characters" >&2
    exit 1
  fi
  if [[ -n "${file_path}" ]]; then
    if [[ "${file_path}" != /* ]]; then
      echo "[entrypoint] ${description}: mounted key path must be absolute" >&2
      exit 1
    fi
    if [[ ! -f "${file_path}" || -L "${file_path}" ]]; then
      echo "[entrypoint] ${description}: mounted key must be a regular non-symlink file" >&2
      exit 1
    fi
    if ! install -m 0600 "${file_path}" "${temporary}"; then
      echo "[entrypoint] ${description}: failed to copy mounted key" >&2
      exit 1
    fi
  else
    if ! printf '%s' "${encoded}" | base64 -d > "${temporary}"; then
      echo "[entrypoint] ${description}: invalid base64 key secret" >&2
      exit 1
    fi
    chmod 0600 "${temporary}"
  fi
  if [[ "${key_kind}" == "validator" ]]; then
    validate_validator_key "${temporary}" "${description}"
  else
    validate_node_key "${temporary}" "${description}"
  fi
  actual_digest="$(sha256sum "${temporary}" | awk '{print $1}')"
  if [[ "${actual_digest}" != "${expected_digest}" ]]; then
    echo "[entrypoint] ${description}: SHA-256 mismatch; refusing to replace persisted identity" >&2
    exit 1
  fi
}

validate_signing_state() {
  local state_path="$1"
  if ! jq -e '
    type == "object" and
    (.height | type == "string" and test("^(0|[1-9][0-9]*)$")) and
    (.round | type == "number" and floor == . and . >= 0) and
    (.step | type == "number" and floor == . and . >= 0 and . <= 3) and
    ((has("signature") | not) or .signature == null or (.signature | type == "string")) and
    ((has("signbytes") | not) or .signbytes == null or (.signbytes | type == "string"))
  ' "${state_path}" >/dev/null; then
    echo "[entrypoint] persisted validator signing state is invalid" >&2
    exit 1
  fi
}

require_regular_file() {
  local path="$1"
  local description="$2"
  if [[ -L "${path}" || ! -f "${path}" ]]; then
    echo "[entrypoint] ${description} must be a regular non-symlink file" >&2
    exit 1
  fi
}

verify_persisted_identity_unit() {
  local validator_digest node_digest
  require_regular_file "${VALIDATOR_KEY_PATH}" "persisted validator key"
  require_regular_file "${VALIDATOR_STATE_PATH}" "persisted validator signing state"
  require_regular_file "${NODE_KEY_PATH}" "persisted P2P node key"
  validate_validator_key "${VALIDATOR_KEY_PATH}" "persisted validator signing key"
  validate_signing_state "${VALIDATOR_STATE_PATH}"
  validate_node_key "${NODE_KEY_PATH}" "persisted P2P node key"
  validator_digest="$(sha256sum "${VALIDATOR_KEY_PATH}" | awk '{print $1}')"
  node_digest="$(sha256sum "${NODE_KEY_PATH}" | awk '{print $1}')"
  if [[ "${validator_digest}" != "${validator_key_digest}" ]]; then
    echo "[entrypoint] validator signing key: refusing identity drift while persisted signing state exists; use the reviewed key-rotation procedure" >&2
    exit 1
  fi
  if [[ "${node_digest}" != "${node_key_digest}" ]]; then
    echo "[entrypoint] P2P node key: refusing identity drift on a persisted volume; use the reviewed key-rotation procedure" >&2
    exit 1
  fi
}

install_prepared_key() {
  local prepared="$1"
  local destination="$2"
  local expected_digest="$3"
  local description="$4"
  local allow_generated_replacement="$5"
  local destination_dir temporary installed_digest

  if [[ -L "${destination}" ]]; then
    echo "[entrypoint] ${description}: refusing persisted identity symlink" >&2
    exit 1
  fi
  if [[ -e "${destination}" ]]; then
    if [[ ! -f "${destination}" ]]; then
      echo "[entrypoint] ${description}: persisted identity must be a regular file" >&2
      exit 1
    fi
    installed_digest="$(sha256sum "${destination}" | awk '{print $1}')"
    if [[ "${installed_digest}" == "${expected_digest}" ]]; then
      chmod 0600 "${destination}"
      return
    fi
    if [[ "${allow_generated_replacement}" != true ]]; then
      echo "[entrypoint] ${description}: refusing identity drift on a persisted volume; use the reviewed key-rotation procedure" >&2
      exit 1
    fi
  fi

  destination_dir="$(dirname "${destination}")"
  if [[ -L "${destination_dir}" || ! -d "${destination_dir}" ]]; then
    echo "[entrypoint] ${description}: destination directory must be a regular directory" >&2
    exit 1
  fi
  temporary="$(mktemp "${destination_dir}/.$(basename "${destination}").tmp.XXXXXX")"
  if ! install -m 0600 "${prepared}" "${temporary}"; then
    rm -f "${temporary}"
    echo "[entrypoint] ${description}: failed to stage validated key" >&2
    exit 1
  fi
  installed_digest="$(sha256sum "${temporary}" | awk '{print $1}')"
  if [[ "${installed_digest}" != "${expected_digest}" ]]; then
    rm -f "${temporary}"
    echo "[entrypoint] ${description}: staged key digest mismatch" >&2
    exit 1
  fi
  mv -f "${temporary}" "${destination}"
  chmod 0600 "${destination}"
  installed_digest="$(sha256sum "${destination}" | awk '{print $1}')"
  if [[ "${installed_digest}" != "${expected_digest}" ]]; then
    echo "[entrypoint] ${description}: installed key failed post-install digest verification" >&2
    exit 1
  fi
}

prepared_validator_key="${candidate_dir}/priv_validator_key.json"
prepared_node_key="${candidate_dir}/node_key.json"
prepare_runtime_key \
  "${validator_key_file}" \
  "${validator_key_b64}" \
  "${validator_key_digest}" \
  "validator signing key" \
  "PRIV_VALIDATOR_KEY_SHA256" \
  validator \
  "${prepared_validator_key}"
prepare_runtime_key \
  "${node_key_file}" \
  "${node_key_b64}" \
  "${node_key_digest}" \
  "P2P node key" \
  "NODE_KEY_SHA256" \
  node \
  "${prepared_node_key}"

# Do not retain private key text in shell variables after the validated
# candidate files exist. The files are removed before the final exec.
validator_key_file=""
validator_key_b64=""
node_key_file=""
node_key_b64=""

if [[ -L "${HOME_DIR}" || ( -e "${HOME_DIR}" && ! -d "${HOME_DIR}" ) ]]; then
  echo "[entrypoint] validator home must be a regular directory" >&2
  exit 1
fi
if [[ "${fresh_volume}" == true ]]; then
  if [[ -d "${HOME_DIR}" ]] &&
    [[ -n "$(find "${HOME_DIR}" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
    echo "[entrypoint] genesis is absent but the validator home is not empty; refusing partial-state initialization" >&2
    exit 1
  fi
  echo "[entrypoint] fresh volume — seeding ${HOME_DIR}"
  zeroned init "${MONIKER:-${ZERONE_DEFAULT_MONIKER}}" --chain-id "${ZERONE_CHAIN_ID}" \
    --default-denom uzrn --home "${HOME_DIR}" >/dev/null 2>&1
  validate_signing_state "${VALIDATOR_STATE_PATH}"
  install_prepared_key \
    "${prepared_validator_key}" \
    "${VALIDATOR_KEY_PATH}" \
    "${validator_key_digest}" \
    "validator signing key" \
    true
  install_prepared_key \
    "${prepared_node_key}" \
    "${NODE_KEY_PATH}" \
    "${node_key_digest}" \
    "P2P node key" \
    true
  cp "${SEED}/genesis.json" "${GENESIS_PATH}"
  require_regular_file "${GENESIS_PATH}" "validator genesis"
  validate_genesis "${GENESIS_PATH}" "installed validator genesis"
  require_regular_file "${HOME_DIR}/config/config.toml" "Comet configuration"
  require_regular_file "${HOME_DIR}/config/app.toml" "application configuration"

  CFG="${HOME_DIR}/config/config.toml"
  # P2P remains reachable for the transitional single-validator launch. RPC,
  # REST, and gRPC stay loopback-only; this signer is not a public API node.
  sed -i 's|^addr_book_strict = true|addr_book_strict = false|' "${CFG}"
  sed -i 's|^allow_duplicate_ip = false|allow_duplicate_ip = true|' "${CFG}"
else
  echo "[entrypoint] existing volume — validating persisted signer unit"
  for managed_directory in "${HOME_DIR}/config" "${HOME_DIR}/data"; do
    if [[ -L "${managed_directory}" || ! -d "${managed_directory}" ]]; then
      echo "[entrypoint] persisted validator directories must not be symlinks" >&2
      exit 1
    fi
  done
  require_regular_file "${GENESIS_PATH}" "persisted validator genesis"
  validate_genesis "${GENESIS_PATH}" "persisted validator genesis"
  require_regular_file "${HOME_DIR}/config/config.toml" "persisted Comet configuration"
  require_regular_file "${HOME_DIR}/config/app.toml" "persisted application configuration"
  verify_persisted_identity_unit
  install_prepared_key \
    "${prepared_validator_key}" \
    "${VALIDATOR_KEY_PATH}" \
    "${validator_key_digest}" \
    "validator signing key" \
    false
  install_prepared_key \
    "${prepared_node_key}" \
    "${NODE_KEY_PATH}" \
    "${node_key_digest}" \
    "P2P node key" \
    false
fi

cleanup_candidate_material
trap - EXIT

# Advertise the public P2P address on EVERY boot (the dedicated IP is known
# only after allocation, and peers must dial a routable address to sync).
if [[ -n "${external_address}" ]]; then
  sed -i "s|^external_address = .*|external_address = \"${external_address}\"|" "${HOME_DIR}/config/config.toml"
  echo "[entrypoint] external_address = ${external_address}"
fi

# The node validates app.toml's minimum-gas-prices BEFORE the --minimum-gas-prices
# flag override, so it must be non-empty in app.toml. Set it on EVERY boot
# (handles a resumed volume too); replace an existing line or append if missing.
APP="${HOME_DIR}/config/app.toml"
if grep -q '^minimum-gas-prices' "${APP}"; then
  sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uzrn"|' "${APP}"
else
  printf '\nminimum-gas-prices = "0.025uzrn"\n' >> "${APP}"
fi
echo "[entrypoint] $(grep '^minimum-gas-prices' "${APP}")"

# Run the node directly because this immutable-image deployment receives a new
# binary only through a reviewed image release. Direct execution does not keep
# RPC available at a scheduled upgrade halt: when the old binary reaches an
# unknown upgrade it exits, and the RPC/API processes exit with it. Pre-build,
# checksum, and stage the exact replacement image before the activation height,
# and treat endpoint availability during the binary handoff as untrusted.
#
# Runtime arguments are frozen into the reviewed image. In particular,
# --unsafe-skip-upgrades cannot be injected through mutable environment state.
exec zeroned start --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn
