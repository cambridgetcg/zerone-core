#!/usr/bin/env bash
# Role-separated zerone-1 signer/checkpoint/archive runtime.
# A fresh signer restores its published identities from one-time secrets; a
# fresh observer generates non-validator identities; archive is a one-way,
# attested adoption of an allowlisted, rolled-back serving copy with new keys.
# Keys are never baked in the image.
set -euo pipefail
export LC_ALL=C
umask 077

HOME_DIR="${ZERONE_HOME:-/data/.zeroned}"
SEED="${MAINNET_SEED_DIR:-/mainnet-seed}"
readonly BINARY="/usr/local/bin/zeroned"
EXPECTED_GENESIS_SHA256="c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"
EXPECTED_NODE_ID="ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e"
NODE_ROLE="${NODE_ROLE:-signer}"
RUNTIME_MARKER_VERSION="2"
ARCHIVE_TRANSITION_SCHEMA="zerone-1-archive-transition-v1"
ARCHIVE_READINESS_SCHEMA="zerone-1-archive-readiness-v2"
ARCHIVE_TRANSITION_FILE="${HOME_DIR}/.zerone-1-archive-transition.json"

# Bootstrap custody arrives through exported platform secrets. Copy it into
# ordinary shell variables, then remove the exported names before spawning even
# a hashing, JSON, base64, locking, or zeroned helper process.
BOOTSTRAP_VALIDATOR_KEY_B64="${PRIV_VALIDATOR_KEY_B64:-}"
BOOTSTRAP_NODE_KEY_B64="${NODE_KEY_B64:-}"
unset PRIV_VALIDATOR_KEY_B64 NODE_KEY_B64

die() {
  echo "[entrypoint] ERROR: $*" >&2
  exit 1
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

custody_env_present() {
  [ -n "${BOOTSTRAP_VALIDATOR_KEY_B64}" ] || \
    [ -n "${BOOTSTRAP_NODE_KEY_B64}" ]
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

acquire_home_lock() {
  local parent legacy_lock_file
  parent=$(dirname "${HOME_DIR}")
  mkdir -p "${parent}"
  require_directory "${parent}" "ZERONE_HOME parent"
  legacy_lock_file="${HOME_DIR}.runtime.lock"
  [ ! -L "${legacy_lock_file}" ] || \
    die "runtime lock must not be a symlink: ${legacy_lock_file}"
  # Lock the stable parent directory itself. This avoids opening/truncating any
  # attacker-preplanted lock pathname and survives child exec and home renames.
  exec 9<"${parent}" || die "could not open runtime lock directory ${parent}"
  flock -n 9 || die "another process already owns ${HOME_DIR}"
}

validate_positive_height() {
  local name="$1" value="$2" maximum="$3"
  [[ "${value}" =~ ^[1-9][0-9]*$ ]] || \
    die "${name} must be a canonical positive integer"
  if [ "${#value}" -gt "${#maximum}" ] || \
     { [ "${#value}" -eq "${#maximum}" ] && [[ "${value}" > "${maximum}" ]]; }; then
    die "${name} exceeds the supported CometBFT height range"
  fi
}

validate_checkpoint_plan() {
  local checkpoint="${ZERONE_CHECKPOINT_STATE_HEIGHT:-}"
  local final_block="${ZERONE_FINAL_COMMITTED_HEIGHT:-}"
  local halt_trigger="${ZERONE_HALT_TRIGGER_HEIGHT:-}"
  local configured=0

  [ -z "${ZERONE_HALT_HEIGHT:-}" ] || \
    die "ZERONE_HALT_HEIGHT is ambiguous and retired; set checkpoint, final-committed, and halt-trigger heights"
  [ -z "${EXTRA_START_FLAGS:-}" ] || \
    die "EXTRA_START_FLAGS is retired; use the explicit checkpoint plan or UNSAFE_SKIP_UPGRADES_HEIGHT"

  [ -z "${checkpoint}" ] || configured=$((configured + 1))
  [ -z "${final_block}" ] || configured=$((configured + 1))
  [ -z "${halt_trigger}" ] || configured=$((configured + 1))
  if [ "${configured}" -ne 0 ] && [ "${configured}" -ne 3 ]; then
    die "ZERONE_CHECKPOINT_STATE_HEIGHT, ZERONE_FINAL_COMMITTED_HEIGHT, and ZERONE_HALT_TRIGGER_HEIGHT must be set together"
  fi
  if [ "${configured}" -eq 0 ]; then
    CHECKPOINT_PLAN_ARMED=0
    return
  fi

  [ -z "${UNSAFE_SKIP_UPGRADES_HEIGHT:-}" ] || \
    die "checkpoint heights and UNSAFE_SKIP_UPGRADES_HEIGHT are mutually exclusive"
  [ -z "${EXTERNAL_ADDRESS:-}" ] || \
    die "checkpoint plan forbids EXTERNAL_ADDRESS; deploy the private halt signer profile"
  validate_positive_height ZERONE_CHECKPOINT_STATE_HEIGHT "${checkpoint}" 9223372036854775805
  validate_positive_height ZERONE_FINAL_COMMITTED_HEIGHT "${final_block}" 9223372036854775806
  validate_positive_height ZERONE_HALT_TRIGGER_HEIGHT "${halt_trigger}" 9223372036854775807
  [ "$((checkpoint + 1))" -eq "${final_block}" ] || \
    die "ZERONE_FINAL_COMMITTED_HEIGHT must equal ZERONE_CHECKPOINT_STATE_HEIGHT + 1"
  [ "$((final_block + 1))" -eq "${halt_trigger}" ] || \
    die "ZERONE_HALT_TRIGGER_HEIGHT must equal ZERONE_FINAL_COMMITTED_HEIGHT + 1"
  CHECKPOINT_PLAN_ARMED=1
}

validate_persisted_checkpoint_plan() {
  local marker="${HOME_DIR}/.zerone-1-checkpoint-plan"
  CHECKPOINT_PLAN_PERSISTED=0
  [ ! -L "${marker}" ] || die "checkpoint plan marker must not be a symlink"
  if [ ! -e "${marker}" ]; then
    return 0
  fi
  require_regular_file "${marker}" checkpoint-plan-marker
  [ "${CHECKPOINT_PLAN_ARMED}" -eq 1 ] || \
    die "this volume is permanently checkpoint-armed; all three F/A/H settings remain required"
  [ "$(kv_value "${marker}" checkpoint_state_height)" = \
      "${ZERONE_CHECKPOINT_STATE_HEIGHT}" ] || die "checkpoint state height changed after arming"
  [ "$(kv_value "${marker}" final_committed_height)" = \
      "${ZERONE_FINAL_COMMITTED_HEIGHT}" ] || die "final committed height changed after arming"
  [ "$(kv_value "${marker}" halt_trigger_height)" = \
      "${ZERONE_HALT_TRIGGER_HEIGHT}" ] || die "halt trigger height changed after arming"
  CHECKPOINT_PLAN_PERSISTED=1
}

persist_checkpoint_plan() {
  local marker="${HOME_DIR}/.zerone-1-checkpoint-plan" tmp
  [ ! -L "${marker}" ] || die "checkpoint plan marker must not be a symlink"
  tmp=$(mktemp "${HOME_DIR}/.zerone-1-checkpoint-plan.tmp.XXXXXX") || \
    die "could not create checkpoint plan marker"
  {
    printf 'checkpoint_state_height=%s\n' "${ZERONE_CHECKPOINT_STATE_HEIGHT}"
    printf 'final_committed_height=%s\n' "${ZERONE_FINAL_COMMITTED_HEIGHT}"
    printf 'halt_trigger_height=%s\n' "${ZERONE_HALT_TRIGGER_HEIGHT}"
  } > "${tmp}"
  chmod 600 "${tmp}"
  mv "${tmp}" "${marker}"
  CHECKPOINT_PLAN_PERSISTED=1
}

validate_archive_inputs() {
  SIGNER_EVIDENCE_SHA256="${ZERONE_SIGNER_EVIDENCE_MANIFEST_SHA256:-}"
  OBSERVER_EVIDENCE_SHA256="${ZERONE_OBSERVER_EVIDENCE_MANIFEST_SHA256:-}"
  SOURCE_OBSERVER_MARKER_SHA256="${ZERONE_SOURCE_OBSERVER_RUNTIME_MARKER_SHA256:-}"
  SOURCE_OBSERVER_NODE_ID="${ZERONE_SOURCE_OBSERVER_NODE_ID:-}"
  SOURCE_OBSERVER_VALIDATOR_PUBKEY="${ZERONE_SOURCE_OBSERVER_VALIDATOR_PUBKEY:-}"
  EXPECTED_ANCHOR_BLOCK_HASH="${ZERONE_EXPECTED_ANCHOR_BLOCK_HASH:-}"
  EXPECTED_POST_ANCHOR_APP_HASH="${ZERONE_EXPECTED_POST_ANCHOR_APP_HASH:-}"
  ARCHIVE_TRANSITION_MANIFEST_SHA256="${ZERONE_ARCHIVE_TRANSITION_MANIFEST_SHA256:-}"
  [[ "${SIGNER_EVIDENCE_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "archive workflow requires ZERONE_SIGNER_EVIDENCE_MANIFEST_SHA256"
  [[ "${OBSERVER_EVIDENCE_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "archive workflow requires ZERONE_OBSERVER_EVIDENCE_MANIFEST_SHA256"
  [[ "${SOURCE_OBSERVER_MARKER_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "archive workflow requires ZERONE_SOURCE_OBSERVER_RUNTIME_MARKER_SHA256"
  [[ "${SOURCE_OBSERVER_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] || \
    die "archive workflow requires canonical ZERONE_SOURCE_OBSERVER_NODE_ID"
  [ "${SOURCE_OBSERVER_NODE_ID}" != "${EXPECTED_NODE_ID}" ] || \
    die "source observer identity unexpectedly equals the official signer"
  [[ "${SOURCE_OBSERVER_VALIDATOR_PUBKEY}" =~ ^[A-Za-z0-9+/]{43}=$ ]] || \
    die "archive workflow requires canonical ZERONE_SOURCE_OBSERVER_VALIDATOR_PUBKEY"
  [[ "${EXPECTED_ANCHOR_BLOCK_HASH}" =~ ^[0-9A-F]{64}$ ]] || \
    die "archive workflow requires uppercase ZERONE_EXPECTED_ANCHOR_BLOCK_HASH"
  [[ "${EXPECTED_POST_ANCHOR_APP_HASH}" =~ ^[0-9A-F]{64}$ ]] || \
    die "archive workflow requires uppercase ZERONE_EXPECTED_POST_ANCHOR_APP_HASH"
  [[ "${ARCHIVE_TRANSITION_MANIFEST_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
    die "archive workflow requires ZERONE_ARCHIVE_TRANSITION_MANIFEST_SHA256"
}

validate_sanitized_archive_allowlist() {
  local entry name forbidden
  for entry in "${HOME_DIR}/config" "${HOME_DIR}/data"; do
    require_directory "${entry}" "sanitized archive directory"
  done
  while IFS= read -r -d '' entry; do
    name=${entry##*/}
    case "${name}" in
      config|data|.zerone-1-archive-transition.json) ;;
      *) die "sanitized archive root contains non-allowlisted entry: ${name}" ;;
    esac
  done < <(find "${HOME_DIR}" -mindepth 1 -maxdepth 1 -print0)
  while IFS= read -r -d '' entry; do
    name=${entry##*/}
    case "${name}" in
      genesis.json|config.toml|app.toml|client.toml|node_key.json|priv_validator_key.json)
        require_regular_file "${entry}" "sanitized archive config"
        ;;
      *) die "sanitized archive config contains non-allowlisted entry: ${name}" ;;
    esac
  done < <(find "${HOME_DIR}/config" -mindepth 1 -maxdepth 1 -print0)
  for name in genesis.json config.toml app.toml client.toml node_key.json \
    priv_validator_key.json; do
    require_regular_file "${HOME_DIR}/config/${name}" "sanitized archive config"
  done
  while IFS= read -r -d '' entry; do
    name=${entry##*/}
    case "${name}" in
      priv_validator_state.json)
        require_regular_file "${entry}" "sanitized archive state"
        ;;
      application.db|blockstore.db|state.db|evidence.db|tx_index.db)
        require_directory "${entry}" "sanitized archive database"
        ;;
      *) die "sanitized archive data contains non-allowlisted entry: ${name}" ;;
    esac
  done < <(find "${HOME_DIR}/data" -mindepth 1 -maxdepth 1 -print0)
  for name in application.db blockstore.db state.db; do
    require_directory "${HOME_DIR}/data/${name}" "required sanitized archive database"
  done
  require_regular_file "${HOME_DIR}/data/priv_validator_state.json" \
    "sanitized archive state"
  forbidden=$(find "${HOME_DIR}" \
    \( -type l -o \( ! -type f ! -type d \) \) -print -quit) || \
    die "could not recursively inspect sanitized archive nodes"
  [ -z "${forbidden}" ] || \
    die "sanitized archive contains a symlink or non-file node: ${forbidden}"
  forbidden=$(find "${HOME_DIR}" -type f -links +1 -print -quit) || \
    die "could not inspect sanitized archive link counts"
  [ -z "${forbidden}" ] || \
    die "sanitized archive contains a hardlinked file: ${forbidden}"
}

validate_archive_transition_manifest() {
  local actual_hash
  [ ! -L "${ARCHIVE_TRANSITION_FILE}" ] || \
    die "archive transition manifest must not be a symlink"
  require_regular_file "${ARCHIVE_TRANSITION_FILE}" "archive transition manifest"
  actual_hash=$(sha256_file "${ARCHIVE_TRANSITION_FILE}") || \
    die "could not hash archive transition manifest"
  [ "${actual_hash}" = "${ARCHIVE_TRANSITION_MANIFEST_SHA256}" ] || \
    die "archive transition manifest hash does not match reviewed deployment input"
  [ "${DERIVED_NODE_ID}" != "${SOURCE_OBSERVER_NODE_ID}" ] || \
    die "sanitized archive reused the source observer P2P identity"
  [ "${DERIVED_VALIDATOR_PUBKEY}" != "${SOURCE_OBSERVER_VALIDATOR_PUBKEY}" ] || \
    die "sanitized archive reused the source observer consensus identity"
  if jq -e --arg pubkey "${SOURCE_OBSERVER_VALIDATOR_PUBKEY}" '
    [ .app_state.genutil.gen_txs[]?.body.messages[]?.pubkey.key ]
    | index($pubkey) != null
  ' "${HOME_DIR}/config/genesis.json" >/dev/null; then
    die "source observer consensus identity unexpectedly belongs to a genesis validator"
  fi
  jq -e \
    --arg schema "${ARCHIVE_TRANSITION_SCHEMA}" \
    --arg checkpoint "${ZERONE_CHECKPOINT_STATE_HEIGHT}" \
    --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" \
    --arg halt "${ZERONE_HALT_TRIGGER_HEIGHT}" \
    --arg genesis "${EXPECTED_GENESIS_SHA256}" \
    --arg source_marker "${SOURCE_OBSERVER_MARKER_SHA256}" \
    --arg source_node "${SOURCE_OBSERVER_NODE_ID}" \
    --arg source_validator "${SOURCE_OBSERVER_VALIDATOR_PUBKEY}" \
    --arg candidate_node "${DERIVED_NODE_ID}" \
    --arg candidate_validator "${DERIVED_VALIDATOR_PUBKEY}" \
    --arg anchor_hash "${EXPECTED_ANCHOR_BLOCK_HASH}" \
    --arg app_hash "${EXPECTED_POST_ANCHOR_APP_HASH}" \
    --arg signer_evidence "${SIGNER_EVIDENCE_SHA256}" \
    --arg observer_evidence "${OBSERVER_EVIDENCE_SHA256}" '
      type == "object"
      and (keys | sort) == ([
        "schema", "chain_id", "checkpoint_state_height",
        "final_committed_height", "halt_trigger_height", "genesis_sha256",
        "cutover_initiation_evidence",
        "source_observer", "candidate", "expected_anchor_block_hash",
        "expected_post_anchor_app_hash", "source_evidence",
        "archive_construction_evidence", "archive_transition_nonce"
      ] | sort)
      and .schema == $schema
      and .chain_id == "zerone-1"
      and .checkpoint_state_height == $checkpoint
      and .final_committed_height == $anchor
      and .halt_trigger_height == $halt
      and .genesis_sha256 == $genesis
      and (.cutover_initiation_evidence | keys | sort) == ([
        "successor_transaction_hash", "committed_height",
        "committed_block_time", "public_notice_sha256",
        "public_notice_publication_evidence_sha256",
        "initiation_evidence_sha256",
        "initiation_evidence_detached_signature_sha256"
      ] | sort)
      and (.cutover_initiation_evidence.successor_transaction_hash |
        test("^[0-9A-F]{64}$"))
      and (.cutover_initiation_evidence.committed_height |
        test("^[1-9][0-9]*$"))
      and (.cutover_initiation_evidence.committed_block_time |
        test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
      and (.cutover_initiation_evidence.public_notice_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.cutover_initiation_evidence.public_notice_publication_evidence_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.cutover_initiation_evidence.initiation_evidence_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.cutover_initiation_evidence.initiation_evidence_detached_signature_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.source_observer | keys | sort) == ([
        "runtime_marker_sha256", "node_id", "validator_pubkey"
      ] | sort)
      and .source_observer.runtime_marker_sha256 == $source_marker
      and .source_observer.node_id == $source_node
      and .source_observer.validator_pubkey == $source_validator
      and (.candidate | keys | sort) == (["node_id", "validator_pubkey"] | sort)
      and .candidate.node_id == $candidate_node
      and .candidate.validator_pubkey == $candidate_validator
      and .expected_anchor_block_hash == $anchor_hash
      and .expected_post_anchor_app_hash == $app_hash
      and (.source_evidence | keys | sort) == ([
        "signer_manifest_sha256", "observer_manifest_sha256"
      ] | sort)
      and .source_evidence.signer_manifest_sha256 == $signer_evidence
      and .source_evidence.observer_manifest_sha256 == $observer_evidence
      and (.archive_construction_evidence | keys | sort) == ([
        "pre_transition_sanitized_snapshot_sha256", "rollback_log_sha256",
        "pre_transition_allowlist_manifest_sha256", "excluded_future_artifacts"
      ] | sort)
      and (.archive_construction_evidence.pre_transition_sanitized_snapshot_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.archive_construction_evidence.rollback_log_sha256 |
        test("^[0-9a-f]{64}$"))
      and (.archive_construction_evidence.pre_transition_allowlist_manifest_sha256 |
        test("^[0-9a-f]{64}$"))
      and .archive_construction_evidence.excluded_future_artifacts == [
        "archive transition manifest", "rendered Fly configs",
        "archive adoption authority", "archive readiness",
        "final checkpoint", "open-beta decision"
      ]
      and (.archive_transition_nonce | test("^[0-9a-f]{64}$"))
    ' "${ARCHIVE_TRANSITION_FILE}" >/dev/null || \
    die "archive transition manifest does not bind the reviewed source, candidate, and checkpoint"
  ARCHIVE_TRANSITION_NONCE=$(jq -er '.archive_transition_nonce' \
    "${ARCHIVE_TRANSITION_FILE}") || die "archive transition nonce is missing"
}

normalize_app_hash() {
  local raw="$1" decoded
  if [[ "${raw}" =~ ^[0-9A-Fa-f]{64}$ ]]; then
    printf '%s' "${raw}" | tr '[:lower:]' '[:upper:]'
    return
  fi
  if ! decoded=$(printf '%s' "${raw}" | base64 --decode 2>/dev/null | \
    od -An -tx1 | tr -d '[:space:]' | tr '[:lower:]' '[:upper:]'); then
    return 1
  fi
  [[ "${decoded}" =~ ^[0-9A-F]{64}$ ]] || return 1
  printf '%s' "${decoded}"
}

validate_archive_readiness() {
  local readiness="${HOME_DIR}/.zerone-1-archive-readiness.json"
  [ ! -L "${readiness}" ] || die "archive readiness attestation must not be a symlink"
  require_regular_file "${readiness}" "archive readiness attestation"
  jq -e \
    --arg schema "${ARCHIVE_READINESS_SCHEMA}" \
    --arg checkpoint "${ZERONE_CHECKPOINT_STATE_HEIGHT}" \
    --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" \
    --arg halt "${ZERONE_HALT_TRIGGER_HEIGHT}" \
    --arg signer_evidence "${SIGNER_EVIDENCE_SHA256}" \
    --arg observer_evidence "${OBSERVER_EVIDENCE_SHA256}" \
    --arg transition_manifest "${ARCHIVE_TRANSITION_MANIFEST_SHA256}" \
    --arg anchor_hash "${EXPECTED_ANCHOR_BLOCK_HASH}" \
    --arg app_hash "${EXPECTED_POST_ANCHOR_APP_HASH}" \
    --arg transition_nonce "${ARCHIVE_TRANSITION_NONCE}" \
    --arg node_id "${DERIVED_NODE_ID}" \
    --arg validator_pubkey "${DERIVED_VALIDATOR_PUBKEY}" '
      type == "object"
      and (keys | sort) == ([
        "schema", "chain_id", "checkpoint_state_height",
        "final_committed_height", "halt_trigger_height", "anchor_block_hash",
        "post_anchor_app_hash", "halt_trigger_block_absent",
        "halt_trigger_results_absent", "block_sync_catching_up",
        "anchor_commit_canonical", "source_evidence", "node_id",
        "validator_pubkey", "archive_transition_nonce",
        "transition_manifest_sha256"
      ] | sort)
      and .schema == $schema
      and .chain_id == "zerone-1"
      and .checkpoint_state_height == $checkpoint
      and .final_committed_height == $anchor
      and .halt_trigger_height == $halt
      and .anchor_block_hash == $anchor_hash
      and .post_anchor_app_hash == $app_hash
      and .halt_trigger_block_absent == true
      and .halt_trigger_results_absent == true
      and .block_sync_catching_up == true
      and .anchor_commit_canonical == false
      and (.source_evidence | type == "object")
      and (.source_evidence | keys | sort) == ([
        "signer_manifest_sha256", "observer_manifest_sha256"
      ] | sort)
      and .source_evidence.signer_manifest_sha256 == $signer_evidence
      and .source_evidence.observer_manifest_sha256 == $observer_evidence
      and .transition_manifest_sha256 == $transition_manifest
      and .archive_transition_nonce == $transition_nonce
      and .node_id == $node_id
      and .validator_pubkey == $validator_pubkey
    ' "${readiness}" >/dev/null || \
    die "archive readiness attestation does not match this A/A volume and evidence plan"
}

capture_archive_readiness() {
  local rpc="http://127.0.0.1:26657" status abci anchor commit trigger trigger_results
  local anchor_hash status_hash app_hash readiness tmp
  status=$(curl -fsS --max-time 3 "${rpc}/status") || return 1
  abci=$(curl -fsS --max-time 3 "${rpc}/abci_info") || return 1
  anchor=$(curl -fsS --max-time 3 \
    "${rpc}/block?height=${ZERONE_FINAL_COMMITTED_HEIGHT}") || return 1
  commit=$(curl -fsS --max-time 3 \
    "${rpc}/commit?height=${ZERONE_FINAL_COMMITTED_HEIGHT}") || return 1
  trigger=$(curl -sS --max-time 3 \
    "${rpc}/block?height=${ZERONE_HALT_TRIGGER_HEIGHT}") || return 1
  trigger_results=$(curl -sS --max-time 3 \
    "${rpc}/block_results?height=${ZERONE_HALT_TRIGGER_HEIGHT}") || return 1

  jq -e --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" '
    .result.node_info.network == "zerone-1"
    and .result.sync_info.latest_block_height == $anchor
    and .result.sync_info.catching_up == true
    and (.result.sync_info.latest_block_hash | test("^[0-9A-Fa-f]{64}$"))
  ' <<< "${status}" >/dev/null || return 1
  jq -e --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" '
    .result.response.last_block_height == $anchor
    and (.result.response.last_block_app_hash | type == "string" and length > 0)
  ' <<< "${abci}" >/dev/null || return 1
  jq -e --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" '
    .result.block.header.chain_id == "zerone-1"
    and .result.block.header.height == $anchor
    and (.result.block_id.hash | test("^[0-9A-Fa-f]{64}$"))
    and ((.result.block.data.txs // []) | length) == 0
  ' <<< "${anchor}" >/dev/null || return 1
  anchor_hash=$(jq -er '.result.block_id.hash | ascii_upcase' <<< "${anchor}") || return 1
  status_hash=$(jq -er '.result.sync_info.latest_block_hash | ascii_upcase' \
    <<< "${status}") || return 1
  [ "${anchor_hash}" = "${status_hash}" ] || return 1
  [ "${anchor_hash}" = "${EXPECTED_ANCHOR_BLOCK_HASH}" ] || return 1
  jq -e --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" --arg hash "${anchor_hash}" '
    .result.canonical == false
    and .result.signed_header.header.height == $anchor
    and .result.signed_header.commit.height == $anchor
    and (.result.signed_header.commit.block_id.hash | ascii_upcase) == $hash
  ' <<< "${commit}" >/dev/null || return 1
  jq -e '.error != null and (.result == null)' <<< "${trigger}" >/dev/null || return 1
  jq -e '.error != null and (.result == null)' <<< "${trigger_results}" >/dev/null || return 1
  app_hash=$(normalize_app_hash \
    "$(jq -er '.result.response.last_block_app_hash' <<< "${abci}")") || return 1
  [ "${app_hash}" = "${EXPECTED_POST_ANCHOR_APP_HASH}" ] || return 1

  readiness="${HOME_DIR}/.zerone-1-archive-readiness.json"
  [ ! -L "${readiness}" ] || die "archive readiness attestation must not be a symlink"
  tmp=$(mktemp "${HOME_DIR}/.zerone-1-archive-readiness.tmp.XXXXXX") || \
    die "could not create archive readiness attestation"
  jq -S -c -n \
    --arg schema "${ARCHIVE_READINESS_SCHEMA}" \
    --arg checkpoint "${ZERONE_CHECKPOINT_STATE_HEIGHT}" \
    --arg anchor "${ZERONE_FINAL_COMMITTED_HEIGHT}" \
    --arg halt "${ZERONE_HALT_TRIGGER_HEIGHT}" \
    --arg anchor_hash "${anchor_hash}" --arg app_hash "${app_hash}" \
    --arg signer_evidence "${SIGNER_EVIDENCE_SHA256}" \
    --arg observer_evidence "${OBSERVER_EVIDENCE_SHA256}" \
    --arg transition_manifest "${ARCHIVE_TRANSITION_MANIFEST_SHA256}" \
    --arg transition_nonce "${ARCHIVE_TRANSITION_NONCE}" \
    --arg node_id "${DERIVED_NODE_ID}" \
    --arg validator_pubkey "${DERIVED_VALIDATOR_PUBKEY}" '{
      schema: $schema,
      chain_id: "zerone-1",
      checkpoint_state_height: $checkpoint,
      final_committed_height: $anchor,
      halt_trigger_height: $halt,
      anchor_block_hash: $anchor_hash,
      post_anchor_app_hash: $app_hash,
      halt_trigger_block_absent: true,
      halt_trigger_results_absent: true,
      block_sync_catching_up: true,
      anchor_commit_canonical: false,
      source_evidence: {
        signer_manifest_sha256: $signer_evidence,
        observer_manifest_sha256: $observer_evidence
      },
      transition_manifest_sha256: $transition_manifest,
      archive_transition_nonce: $transition_nonce,
      node_id: $node_id,
      validator_pubkey: $validator_pubkey
    }' > "${tmp}" || die "could not encode archive readiness attestation"
  chmod 0600 "${tmp}"
  mv "${tmp}" "${readiness}"
  validate_archive_readiness
}

run_archive_candidate() {
  local daemon_pid ready=0 attempt exit_status
  "${BINARY}" start --halt-height "${ZERONE_HALT_TRIGGER_HEIGHT}" \
    --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn &
  daemon_pid=$!
  trap 'kill -TERM "${daemon_pid}" 2>/dev/null || true' TERM INT
  attempt=1
  while [ "${attempt}" -le 60 ]; do
    kill -0 "${daemon_pid}" 2>/dev/null || break
    if capture_archive_readiness; then
      ready=1
      break
    fi
    sleep 2
    attempt=$((attempt + 1))
  done
  if [ "${ready}" -ne 1 ]; then
    kill -TERM "${daemon_pid}" 2>/dev/null || true
    wait "${daemon_pid}" 2>/dev/null || true
    trap - TERM INT
    die "archive candidate never proved sanitized A/A state with H absent"
  fi
  echo "[entrypoint] archive candidate readiness attestation written; redeploy NODE_ROLE=archive"
  if wait "${daemon_pid}"; then exit_status=0; else exit_status=$?; fi
  trap - TERM INT
  return "${exit_status}"
}

toml_set() {
  local section="$1" key="$2" value="$3" file="$4" tmp
  require_regular_file "${file}" "TOML input"
  tmp=$(mktemp "${file}.checkpoint-tmp.XXXXXX") || \
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
  tmp=$(mktemp "${file}.checkpoint-tmp.XXXXXX") || \
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

toml_quote() {
  jq -Rn --arg value "$1" '$value'
}

validate_safe_toml_string() {
  local name="$1" value="$2"
  if grep -q '["\\]' <<< "${value}" || [[ "${value}" == *$'\n'* ]]; then
    die "${name} contains unsafe TOML characters"
  fi
}

validate_host_port() {
  local name="$1" value="$2" host port
  validate_safe_toml_string "${name}" "${value}"
  [[ "${value}" =~ ^[A-Za-z0-9.-]+:[1-9][0-9]{0,4}$ ]] || \
    die "${name} must use a DNS name or IPv4 address followed by :port"
  host="${value%:*}"
  port="${value##*:}"
  [ -n "${host}" ] || die "${name} host is empty"
  [ "$((10#${port}))" -le 65535 ] || die "${name} has an invalid TCP port"
}

validate_observer_peer() {
  local value="${PERSISTENT_PEERS:-}" endpoint
  [ -n "${value}" ] || die "observer role requires PERSISTENT_PEERS"
  [[ "${value}" != *,* ]] || die "observer PERSISTENT_PEERS must contain exactly the official signer"
  [[ "${value}" == "${EXPECTED_NODE_ID}@"* ]] || \
    die "observer PERSISTENT_PEERS must use official signer node ID ${EXPECTED_NODE_ID}"
  endpoint="${value#*@}"
  [ "${endpoint}" != "${value}" ] || die "observer peer must use node-id@host:port"
  validate_host_port PERSISTENT_PEERS "${endpoint}"
}

arm_checkpoint_mempool_freeze() {
  local config="$1" app="$2"
  # Restarting with a zero-capacity Comet mempool drains volatile pending
  # transactions and prevents RPC/P2P CheckTx submissions from entering the
  # explicitly empty anchor block. Network ingress is fenced separately.
  toml_set mempool broadcast false "${config}"
  toml_set mempool wal_dir '""' "${config}"
  toml_set mempool size 0 "${config}"
  toml_set mempool max_txs_bytes 0 "${config}"
  toml_set mempool max-txs -1 "${app}"
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

validate_genesis() {
  local genesis="$1" actual_hash chain_id
  require_regular_file "${genesis}" genesis
  chain_id=$(jq -er '.chain_id' "${genesis}") || die "genesis chain_id is missing"
  [ "${chain_id}" = "zerone-1" ] || \
    die "refusing genesis for chain ${chain_id}; expected zerone-1"
  actual_hash=$(sha256_file "${genesis}") || die "could not hash genesis"
  [ "${actual_hash}" = "${EXPECTED_GENESIS_SHA256}" ] || \
    die "genesis sha256 mismatch: got ${actual_hash}, expected ${EXPECTED_GENESIS_SHA256}"
}

validate_validator_state() {
  local state_file="$1"
  require_regular_file "${state_file}" priv_validator_state.json
  jq -e '
    (.height | tostring | test("^(0|[1-9][0-9]*)$")) and
    (.round | type == "number" and . >= 0 and floor == .) and
    (.step | type == "number" and . >= 0 and . <= 3 and floor == .)
  ' "${state_file}" >/dev/null || die "invalid priv_validator_state.json"
}

validator_state_height() {
  jq -er '.height | tostring' "$1"
}

validate_role_keys() {
  local genesis="$1" node_key="$2" validator_key="$3" validator_home="$4"
  local stored_pubkey derived_pubkey derived_node_id in_genesis

  require_regular_file "${node_key}" node_key.json
  require_regular_file "${validator_key}" priv_validator_key.json

  jq -e '
    .priv_key.type == "tendermint/PrivKeyEd25519" and
    (.priv_key.value | type == "string" and length > 0)
  ' "${node_key}" >/dev/null || die "invalid node_key.json"

  jq -e '
    .priv_key.type == "tendermint/PrivKeyEd25519" and
    .pub_key.type == "tendermint/PubKeyEd25519" and
    (.priv_key.value | type == "string" and length > 0) and
    (.pub_key.value | type == "string" and length > 0)
  ' "${validator_key}" >/dev/null || die "invalid priv_validator_key.json"

  stored_pubkey=$(jq -er '.pub_key.value' "${validator_key}") || \
    die "validator public key is missing"
  derived_node_id=$("${BINARY}" tendermint show-node-id \
    --home "${validator_home}" 2>/dev/null) || \
    die "node private key cannot be loaded by CometBFT"
  derived_pubkey=$("${BINARY}" tendermint show-validator \
    --home "${validator_home}" 2>/dev/null | jq -er '.key') || \
    die "validator private key cannot be loaded by CometBFT"
  [ "${stored_pubkey}" = "${derived_pubkey}" ] || \
    die "validator public key does not match its private key"
  DERIVED_NODE_ID="${derived_node_id}"
  DERIVED_VALIDATOR_PUBKEY="${derived_pubkey}"

  in_genesis=0
  if jq -e --arg pubkey "${derived_pubkey}" '
    [ .app_state.genutil.gen_txs[]?.body.messages[]?.pubkey.key ]
    | index($pubkey) != null
  ' "${genesis}" >/dev/null; then
    in_genesis=1
  fi

  if [ "${NODE_ROLE}" = "signer" ]; then
    [ "${derived_node_id}" = "${EXPECTED_NODE_ID}" ] || \
      die "signer node key derives ${derived_node_id}, expected mainnet node ${EXPECTED_NODE_ID}"
    [ "${in_genesis}" -eq 1 ] || \
      die "signer key does not match any validator in the baked genesis"
  else
    [ "${derived_node_id}" != "${EXPECTED_NODE_ID}" ] || \
      die "non-signer role must not reuse the official signer P2P key"
    [ "${in_genesis}" -eq 0 ] || \
      die "non-signer role unexpectedly holds the zerone-1 validator consensus key"
  fi
}

write_runtime_marker() {
  local marker="${HOME_DIR}/.zerone-1-runtime" tmp
  [ ! -L "${marker}" ] || die "runtime marker must not be a symlink"
  tmp=$(mktemp "${HOME_DIR}/.zerone-1-runtime.tmp.XXXXXX") || \
    die "could not create runtime marker"
  {
    printf 'runtime_version=%s\n' "${RUNTIME_MARKER_VERSION}"
    printf 'role=%s\n' "${NODE_ROLE}"
    printf 'chain_id=zerone-1\n'
    printf 'genesis_sha256=%s\n' "${EXPECTED_GENESIS_SHA256}"
    printf 'node_id=%s\n' "${DERIVED_NODE_ID}"
    printf 'validator_pubkey=%s\n' "${DERIVED_VALIDATOR_PUBKEY}"
    if [ "${NODE_ROLE}" = "archive-candidate" ] || \
       [ "${NODE_ROLE}" = "archive" ]; then
      printf 'archive_transition_nonce=%s\n' "${ARCHIVE_TRANSITION_NONCE}"
      printf 'archive_transition_manifest_sha256=%s\n' \
        "${ARCHIVE_TRANSITION_MANIFEST_SHA256}"
    fi
    if [ "${NODE_ROLE}" = "archive" ]; then
      printf 'archive_readiness_sha256=%s\n' "${ARCHIVE_READINESS_SHA256}"
    fi
  } > "${tmp}"
  chmod 600 "${tmp}"
  mv "${tmp}" "${marker}"
}

validate_runtime_marker() {
  local marker="${HOME_DIR}/.zerone-1-runtime" marker_version marker_role
  local readiness="${HOME_DIR}/.zerone-1-archive-readiness.json"
  local marker_nonce marker_transition_hash
  [ ! -L "${marker}" ] || die "runtime marker must not be a symlink"
  if [ ! -e "${marker}" ]; then
    if [ "${NODE_ROLE}" = "archive-candidate" ]; then
      if [ -e "${readiness}" ] || [ -L "${readiness}" ]; then
        die "sanitized candidate contains stale archive readiness; refusing replay"
      fi
      validate_sanitized_archive_allowlist
      validate_archive_transition_manifest
      write_runtime_marker
      echo "[entrypoint] adopted reviewed, fresh-key sanitized archive candidate"
      return
    fi
    [ "${NODE_ROLE}" = "signer" ] || \
      die "non-signer volume is missing its role marker; archive requires a reviewed candidate stage"
    # The live zerone-1 signer predates role markers. Accept exactly that one
    # legacy shape only after its genesis, node ID, validator key, and signer
    # state have passed the full checks, then permanently mark it as signer.
    write_runtime_marker
    echo "[entrypoint] adopted validated legacy signer volume"
    return
  fi
  require_regular_file "${marker}" runtime-marker
  marker_version=$(kv_value "${marker}" runtime_version) || \
    die "runtime marker version is missing or ambiguous"
  case "${marker_version}" in
    1|"${RUNTIME_MARKER_VERSION}") ;;
    *) die "runtime marker version mismatch" ;;
  esac
  marker_role=$(kv_value "${marker}" role) || \
    die "runtime marker role is missing or ambiguous"
  [ "$(kv_value "${marker}" chain_id)" = "zerone-1" ] || \
    die "runtime marker chain ID mismatch"
  [ "$(kv_value "${marker}" genesis_sha256)" = "${EXPECTED_GENESIS_SHA256}" ] || \
    die "runtime marker genesis hash mismatch"
  if [ "${marker_version}" = "${RUNTIME_MARKER_VERSION}" ]; then
    [ "$(kv_value "${marker}" node_id)" = "${DERIVED_NODE_ID}" ] || \
      die "runtime node identity changed after initialization"
    [ "$(kv_value "${marker}" validator_pubkey)" = \
        "${DERIVED_VALIDATOR_PUBKEY}" ] || \
      die "runtime consensus identity changed after initialization"
  fi
  if [ "${marker_role}" != "${NODE_ROLE}" ] && \
     { [ "${marker_role}" != "archive-candidate" ] || \
       [ "${NODE_ROLE}" != "archive" ]; }; then
    die "volume role does not match NODE_ROLE"
  fi
  if [ "${marker_role}" = "archive-candidate" ] || \
     [ "${marker_role}" = "archive" ]; then
    marker_nonce=$(kv_value "${marker}" archive_transition_nonce) || \
      die "archive transition nonce is missing or ambiguous"
    [[ "${marker_nonce}" =~ ^[0-9a-f]{64}$ ]] || \
      die "archive transition nonce is malformed"
    marker_transition_hash=$(kv_value "${marker}" \
      archive_transition_manifest_sha256) || \
      die "archive transition manifest hash is missing or ambiguous"
    [ "${marker_transition_hash}" = "${ARCHIVE_TRANSITION_MANIFEST_SHA256}" ] || \
      die "archive transition manifest changed after candidate adoption"
    validate_archive_transition_manifest
    [ "${marker_nonce}" = "${ARCHIVE_TRANSITION_NONCE}" ] || \
      die "archive transition nonce changed after candidate adoption"
  fi
  if [ "${marker_role}" != "${NODE_ROLE}" ]; then
    if [ "${marker_role}" = "archive-candidate" ] && \
       [ "${NODE_ROLE}" = "archive" ]; then
      validate_archive_readiness
      ARCHIVE_READINESS_SHA256=$(sha256_file "${readiness}") || \
        die "could not hash archive readiness attestation"
      [[ "${ARCHIVE_READINESS_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
        die "archive readiness hash is malformed"
      # The role marker is written only after the local RPC attestation has
      # bound A/A, H absence, evidence hashes, identities, and the one-time
      # candidate nonce. The marker is then an irreversible local consume.
      write_runtime_marker
      echo "[entrypoint] permanently consumed readiness and transitioned candidate to archive"
      return
    fi
    die "volume role does not match NODE_ROLE"
  fi
  if [ "${NODE_ROLE}" = "archive" ]; then
    validate_archive_readiness
    ARCHIVE_READINESS_SHA256=$(sha256_file "${readiness}") || \
      die "could not hash archive readiness attestation"
    [ "$(kv_value "${marker}" archive_readiness_sha256)" = \
        "${ARCHIVE_READINESS_SHA256}" ] || \
      die "archive readiness attestation changed after transition"
  fi
  if [ "${marker_version}" = "1" ]; then
    write_runtime_marker
    echo "[entrypoint] upgraded validated runtime marker with pinned identities"
    return
  fi
}

configure_role() {
  local config="${HOME_DIR}/config/config.toml"
  local app="${HOME_DIR}/config/app.toml"
  require_regular_file "${config}" config.toml
  require_regular_file "${app}" app.toml

  toml_set_root minimum-gas-prices '"0.025uzrn"' "${app}"
  toml_set rpc unsafe false "${config}"
  toml_set p2p seeds '""' "${config}"

  if [ "${NODE_ROLE}" = "signer" ]; then
    [ -z "${PERSISTENT_PEERS:-}" ] || \
      die "signer role does not accept PERSISTENT_PEERS; observers dial the signer"
    toml_set rpc laddr '"tcp://0.0.0.0:26657"' "${config}"
    toml_set rpc cors_allowed_origins '["*"]' "${config}"
    toml_set p2p persistent_peers '""' "${config}"
    toml_set p2p private_peer_ids '""' "${config}"
    toml_set p2p pex true "${config}"
    toml_set p2p addr_book_strict false "${config}"
    toml_set p2p allow_duplicate_ip true "${config}"
    toml_set p2p external_address '""' "${config}"
    toml_set api enable true "${app}"
    toml_set api address '"tcp://0.0.0.0:1317"' "${app}"
    toml_set api enabled-unsafe-cors true "${app}"
    toml_set grpc enable true "${app}"
    toml_set grpc address '"0.0.0.0:9090"' "${app}"
  elif [ "${NODE_ROLE}" = "observer" ]; then
    [ -z "${EXTERNAL_ADDRESS:-}" ] || die "observer role forbids EXTERNAL_ADDRESS"
    validate_observer_peer
    toml_set rpc laddr '"tcp://0.0.0.0:26657"' "${config}"
    toml_set rpc cors_allowed_origins '[]' "${config}"
    toml_set p2p persistent_peers "$(toml_quote "${PERSISTENT_PEERS}")" "${config}"
    toml_set p2p private_peer_ids "$(toml_quote "${EXPECTED_NODE_ID}")" "${config}"
    toml_set p2p pex false "${config}"
    toml_set p2p addr_book_strict true "${config}"
    toml_set p2p allow_duplicate_ip false "${config}"
    toml_set p2p external_address '""' "${config}"
    toml_set api enable true "${app}"
    toml_set api address '"tcp://0.0.0.0:1317"' "${app}"
    toml_set api enabled-unsafe-cors false "${app}"
    toml_set grpc enable false "${app}"
    toml_set grpc address '"127.0.0.1:9090"' "${app}"
    # The observer is a block-sync/query source, never a transaction relay.
    arm_checkpoint_mempool_freeze "${config}" "${app}"
  else
    [ -z "${EXTERNAL_ADDRESS:-}" ] || die "${NODE_ROLE} role forbids EXTERNAL_ADDRESS"
    [ -z "${PERSISTENT_PEERS:-}" ] || die "${NODE_ROLE} role forbids PERSISTENT_PEERS"
    toml_set rpc laddr '"tcp://0.0.0.0:26657"' "${config}"
    toml_set rpc cors_allowed_origins '[]' "${config}"
    toml_set p2p laddr '"tcp://127.0.0.1:26656"' "${config}"
    toml_set p2p persistent_peers '""' "${config}"
    toml_set p2p private_peer_ids '""' "${config}"
    toml_set p2p pex false "${config}"
    toml_set p2p addr_book_strict true "${config}"
    toml_set p2p allow_duplicate_ip false "${config}"
    toml_set p2p external_address '""' "${config}"
    toml_set p2p max_num_inbound_peers 0 "${config}"
    toml_set p2p max_num_outbound_peers 0 "${config}"
    toml_set api enable true "${app}"
    toml_set api address '"tcp://0.0.0.0:1317"' "${app}"
    toml_set api enabled-unsafe-cors false "${app}"
    toml_set grpc enable false "${app}"
    toml_set grpc address '"127.0.0.1:9090"' "${app}"
    arm_checkpoint_mempool_freeze "${config}" "${app}"
  fi
}

case "${NODE_ROLE}" in
  signer|observer|archive-candidate|archive) ;;
  *) die "NODE_ROLE must be exactly signer, observer, archive-candidate, or archive" ;;
esac
require_regular_file "${BINARY}" zeroned
[ -x "${BINARY}" ] || die "zeroned is not executable: ${BINARY}"
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v flock >/dev/null 2>&1 || die "flock is required for single-home process fencing"
if [ "${NODE_ROLE}" = "archive-candidate" ]; then
  command -v curl >/dev/null 2>&1 || die "curl is required for archive readiness proof"
  command -v find >/dev/null 2>&1 || die "find is required for archive allowlist validation"
  command -v base64 >/dev/null 2>&1 || die "base64 is required for archive app-hash validation"
  command -v od >/dev/null 2>&1 || die "od is required for archive app-hash validation"
  command -v tr >/dev/null 2>&1 || die "tr is required for archive app-hash validation"
fi
[ ! -L "${HOME_DIR}" ] || die "ZERONE_HOME must not be a symlink"
if [ -e "${HOME_DIR}" ] && [ ! -d "${HOME_DIR}" ]; then
  die "ZERONE_HOME exists but is not a directory"
fi
for subdirectory in config data; do
  [ ! -L "${HOME_DIR}/${subdirectory}" ] || \
    die "ZERONE_HOME ${subdirectory} must not be a symlink"
done

acquire_home_lock
CHECKPOINT_PLAN_ARMED=0
validate_checkpoint_plan
NEW_VOLUME=0

if [ ! -e "${HOME_DIR}/config/genesis.json" ]; then
  NEW_VOLUME=1
  if [ -d "${HOME_DIR}" ] && [ -n "$(ls -A "${HOME_DIR}")" ]; then
    die "refusing partially initialized non-empty volume without genesis"
  fi
  validate_genesis "${SEED}/genesis.json"
  if [ "${NODE_ROLE}" = "archive-candidate" ] || [ "${NODE_ROLE}" = "archive" ]; then
    die "${NODE_ROLE} role requires an initialized, checkpoint-armed observer clone"
  fi

  SECRET_TMP=""
  if [ "${NODE_ROLE}" = "signer" ]; then
    [ -n "${BOOTSTRAP_VALIDATOR_KEY_B64}" ] || \
      die "fresh signer volume requires PRIV_VALIDATOR_KEY_B64"
    [ -n "${BOOTSTRAP_NODE_KEY_B64}" ] || \
      die "fresh signer volume requires NODE_KEY_B64"

    validator_b64="${BOOTSTRAP_VALIDATOR_KEY_B64}"
    node_b64="${BOOTSTRAP_NODE_KEY_B64}"
    unset BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64
    SECRET_TMP=$(mktemp -d)
    trap 'rm -rf "${SECRET_TMP}"' EXIT
    mkdir -p "${SECRET_TMP}/config" "${SECRET_TMP}/data"
    printf '%s\n' '{"height":"0","round":0,"step":0}' \
      > "${SECRET_TMP}/data/priv_validator_state.json"
    printf '%s' "${validator_b64}" | base64 --decode \
      > "${SECRET_TMP}/config/priv_validator_key.json" || \
      die "PRIV_VALIDATOR_KEY_B64 is not valid base64"
    printf '%s' "${node_b64}" | base64 --decode \
      > "${SECRET_TMP}/config/node_key.json" || \
      die "NODE_KEY_B64 is not valid base64"
    unset validator_b64 node_b64
    chmod 600 "${SECRET_TMP}/config/priv_validator_key.json" \
      "${SECRET_TMP}/config/node_key.json" \
      "${SECRET_TMP}/data/priv_validator_state.json"
    validate_role_keys "${SEED}/genesis.json" \
      "${SECRET_TMP}/config/node_key.json" \
      "${SECRET_TMP}/config/priv_validator_key.json" "${SECRET_TMP}"
  elif [ "${NODE_ROLE}" = "observer" ]; then
    custody_env_present && die "non-signer role rejects validator bootstrap custody"
    unset BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64
    validate_observer_peer
  else
    die "${NODE_ROLE} role cannot initialize a fresh volume"
  fi

  echo "[entrypoint] fresh ${NODE_ROLE} volume — seeding ${HOME_DIR}"
  "${BINARY}" init "${MONIKER:-zerone-1-${NODE_ROLE}}" --chain-id zerone-1 \
    --default-denom uzrn --home "${HOME_DIR}" >/dev/null 2>&1
  cp "${SEED}/genesis.json" "${HOME_DIR}/config/genesis.json"

  if [ "${NODE_ROLE}" = "signer" ]; then
    install -m 600 "${SECRET_TMP}/config/node_key.json" \
      "${HOME_DIR}/config/node_key.json"
    install -m 600 "${SECRET_TMP}/config/priv_validator_key.json" \
      "${HOME_DIR}/config/priv_validator_key.json"
    rm -rf "${SECRET_TMP}"
    SECRET_TMP=""
    trap - EXIT
    echo "[entrypoint] signer node and validator keys restored from one-time secrets"
  fi
else
  custody_env_present && \
    die "bootstrap custody inputs are forbidden on an initialized volume; remove them after first boot"
  unset BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64
  echo "[entrypoint] existing ${NODE_ROLE} volume — resuming"
fi

# Refuse missing, malformed, symlinked, or wrong-role custody material before
# any daemon start. Observer/archive keys are ordinary fresh Comet keys and
# must prove that they are not either published signer identity.
require_directory "${HOME_DIR}/config" "config directory"
require_directory "${HOME_DIR}/data" "data directory"
validate_genesis "${HOME_DIR}/config/genesis.json"
validate_validator_state "${HOME_DIR}/data/priv_validator_state.json"
require_regular_file "${HOME_DIR}/config/node_key.json" "node key"
require_regular_file "${HOME_DIR}/config/priv_validator_key.json" "validator key"
chmod 600 "${HOME_DIR}/config/node_key.json" \
  "${HOME_DIR}/config/priv_validator_key.json" \
  "${HOME_DIR}/data/priv_validator_state.json"
validate_role_keys "${HOME_DIR}/config/genesis.json" \
  "${HOME_DIR}/config/node_key.json" \
  "${HOME_DIR}/config/priv_validator_key.json" "${HOME_DIR}"
if [ "${NEW_VOLUME}" -eq 1 ]; then
  write_runtime_marker
fi
validate_persisted_checkpoint_plan
if [ "${NODE_ROLE}" = "archive-candidate" ] || [ "${NODE_ROLE}" = "archive" ]; then
  [ "${CHECKPOINT_PLAN_ARMED}" -eq 1 ] || \
    die "${NODE_ROLE} role requires the explicit F/A/H checkpoint plan"
  if [ "${NODE_ROLE}" = "archive" ]; then
    [ "${CHECKPOINT_PLAN_PERSISTED}" -eq 1 ] || \
      die "archive role requires the candidate-persisted F/A/H checkpoint plan"
  fi
  validate_archive_inputs
fi
validate_runtime_marker
if [ "${NODE_ROLE}" = "archive-candidate" ] && \
   [ "${CHECKPOINT_PLAN_PERSISTED}" -ne 1 ]; then
  persist_checkpoint_plan
fi
configure_role

if [ "${CHECKPOINT_PLAN_ARMED}" -eq 1 ] && [ "${NODE_ROLE}" = "signer" ]; then
  SIGNER_HEIGHT=$(validator_state_height "${HOME_DIR}/data/priv_validator_state.json") || \
    die "could not read signer height"
  # Equal-length canonical decimal strings are intentionally compared
  # lexicographically here to avoid overflowing Bash arithmetic.
  # shellcheck disable=SC2071
  if [ "${#SIGNER_HEIGHT}" -gt 19 ] || \
     { [ "${#SIGNER_HEIGHT}" -eq 19 ] && [ "${SIGNER_HEIGHT}" \> "9223372036854775807" ]; }; then
    die "signer height exceeds the supported CometBFT height range"
  fi
  [ "${SIGNER_HEIGHT}" -lt "${ZERONE_CHECKPOINT_STATE_HEIGHT}" ] || \
    die "checkpoint plan was armed too late: signer height ${SIGNER_HEIGHT} is not below state height ${ZERONE_CHECKPOINT_STATE_HEIGHT}"
  arm_checkpoint_mempool_freeze \
    "${HOME_DIR}/config/config.toml" "${HOME_DIR}/config/app.toml"
  # The service-free halt profile keeps the P2P listener available only to the
  # explicitly configured Fly-private observer. Never advertise old public
  # coordinates or learn/gossip additional peers after F/A/H is armed.
  toml_set p2p external_address '""' "${HOME_DIR}/config/config.toml"
  toml_set p2p pex false "${HOME_DIR}/config/config.toml"
fi
if [ "${NODE_ROLE}" = "archive-candidate" ]; then
  run_archive_candidate
  exit $?
fi
if [ "${CHECKPOINT_PLAN_ARMED}" -eq 1 ]; then
  [ "${CHECKPOINT_PLAN_PERSISTED}" -eq 1 ] || persist_checkpoint_plan
  echo "[entrypoint] checkpoint plan armed: state=${ZERONE_CHECKPOINT_STATE_HEIGHT} anchor=${ZERONE_FINAL_COMMITTED_HEIGHT} halt-trigger=${ZERONE_HALT_TRIGGER_HEIGHT}"
fi

# Advertise the signer address during ordinary public operation, but never
# interpolate unchecked text into TOML. Checkpoint and non-signer roles forbid it.
if [ "${NODE_ROLE}" = "signer" ] && [ -n "${EXTERNAL_ADDRESS:-}" ]; then
  validate_host_port EXTERNAL_ADDRESS "${EXTERNAL_ADDRESS}"
  toml_set p2p external_address "$(toml_quote "${EXTERNAL_ADDRESS}")" \
    "${HOME_DIR}/config/config.toml"
  echo "[entrypoint] external_address = ${EXTERNAL_ADDRESS}"
fi

# Defense in depth: no bootstrap custody variable is inherited by the daemon.
unset PRIV_VALIDATOR_KEY_B64 NODE_KEY_B64 \
  BOOTSTRAP_VALIDATOR_KEY_B64 BOOTSTRAP_NODE_KEY_B64

# The SDK refuses FinalizeBlock at --halt-height. The selected checkpoint state
# is F, carried by empty canonical anchor A=F+1; Comet stages H=A+1 before that
# refusal. Consensus stops, but the daemon and health endpoints may remain live,
# so the cutover runbook must verify the split and explicitly fence the signer.
# Never pass F itself as --halt-height.
if [ "${CHECKPOINT_PLAN_ARMED}" -eq 1 ]; then
  exec "${BINARY}" start \
    --halt-height "${ZERONE_HALT_TRIGGER_HEIGHT}" \
    --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn
fi
# Run the node directly. cosmovisor was trialled here and removed on 2026-07-25:
# in an immutable-image deployment it cannot help with a NEW upgrade (the
# replacement binary only ever arrives by deploying an image), and it actively
# hurt. At the agenttool-seam-v1 halt it exited 1, fly's crash-loop backoff then
# STOPPED the machine, RPC went dark, and the machine still needed a manual
# start after the correct image was deployed.
#
# Without it the same halt is a good one: the node stays up, RPC keeps
# answering, the log says UPGRADE "<name>" NEEDED, and deploying the
# pre-built image resumes the chain in ~90s. Pre-building the image before the
# upgrade height is what actually turned a 28h outage into ~2min — that is the
# practice worth keeping, not the supervisor.
#
# The only incident escape hatch retained by this fail-closed runtime is a
# validated, signer-only upgrade height. Arbitrary EXTRA_START_FLAGS are
# rejected earlier in startup.
if [ -n "${UNSAFE_SKIP_UPGRADES_HEIGHT:-}" ]; then
  [ "${NODE_ROLE}" = "signer" ] || \
    die "non-signer role forbids UNSAFE_SKIP_UPGRADES_HEIGHT"
  validate_positive_height UNSAFE_SKIP_UPGRADES_HEIGHT \
    "${UNSAFE_SKIP_UPGRADES_HEIGHT}" 9223372036854775807
  exec "${BINARY}" start \
    --unsafe-skip-upgrades "${UNSAFE_SKIP_UPGRADES_HEIGHT}" \
    --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn
fi
exec "${BINARY}" start --home "${HOME_DIR}" --minimum-gas-prices 0.025uzrn
