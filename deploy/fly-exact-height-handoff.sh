#!/usr/bin/env bash
set -euo pipefail

: "${FLY_APP:?FLY_APP must name the validator app}"
: "${FLY_MACHINE_ID:?FLY_MACHINE_ID must name the one validator Machine}"
: "${FLY_VOLUME_ID:?FLY_VOLUME_ID must name the attached validator volume}"
: "${FLY_CURRENT_IMAGE_REF:?FLY_CURRENT_IMAGE_REF must bind the currently configured immutable image}"
: "${FLY_IMAGE_REF:?FLY_IMAGE_REF must be an immutable registry.fly.io image digest reference}"
: "${FLY_CONFIG_PATH:?FLY_CONFIG_PATH must name the reviewed Fly configuration}"
: "${FLY_CONFIG_SHA256:?FLY_CONFIG_SHA256 must bind the reviewed Fly configuration}"
: "${CHAIN_ID:?CHAIN_ID must match the observer network exactly}"
: "${UPGRADE_NAME:?UPGRADE_NAME must match the on-chain plan exactly}"
: "${UPGRADE_HEIGHT:?UPGRADE_HEIGHT must match the on-chain plan exactly}"
: "${UPGRADE_PLAN_EVIDENCE_PATH:?UPGRADE_PLAN_EVIDENCE_PATH must contain the captured observer current-plan response}"
: "${UPGRADE_PLAN_SHA256:?UPGRADE_PLAN_SHA256 must bind the independently captured canonical on-chain plan evidence}"
: "${ACTIVATION_PREFLIGHT_REPORT_PATH:?ACTIVATION_PREFLIGHT_REPORT_PATH must contain the exact H-1 activation-preflight report}"
: "${ACTIVATION_PREFLIGHT_REPORT_SHA256:?ACTIVATION_PREFLIGHT_REPORT_SHA256 must bind the exact H-1 activation-preflight report}"
: "${UPGRADE_ARM_EVIDENCE_PATH:?UPGRADE_ARM_EVIDENCE_PATH must contain the reviewed pre-H no-restart evidence}"
: "${UPGRADE_ARM_EVIDENCE_SHA256:?UPGRADE_ARM_EVIDENCE_SHA256 must bind the reviewed pre-H no-restart evidence}"
: "${UPGRADE_EXIT_EVIDENCE_PATH:?UPGRADE_EXIT_EVIDENCE_PATH must contain the reviewed old-binary exit observation}"
: "${UPGRADE_EXIT_EVIDENCE_SHA256:?UPGRADE_EXIT_EVIDENCE_SHA256 must bind the reviewed old-binary exit observation}"
: "${OLD_BINARY_EXIT_LOG_PATH:?OLD_BINARY_EXIT_LOG_PATH must contain the captured old-binary exit log}"
: "${UPGRADE_INFO_EVIDENCE_PATH:?UPGRADE_INFO_EVIDENCE_PATH must contain the captured upgrade-info.json}"
: "${LAST_COMMITTED_HEIGHT:?LAST_COMMITTED_HEIGHT must come from an independent observer after the old binary exits}"
: "${LAST_COMMITTED_APP_HASH:?LAST_COMMITTED_APP_HASH must bind the independently observed H-1 application state}"
: "${ATTEMPTED_UPGRADE_HEIGHT:?ATTEMPTED_UPGRADE_HEIGHT must identify the uncommitted height attempted by the old binary}"
: "${EXPECTED_UPGRADE_APP_HASH:?EXPECTED_UPGRADE_APP_HASH must bind the rehearsed application state after H commits}"
: "${EXPECTED_VALIDATOR_ADDRESS:?EXPECTED_VALIDATOR_ADDRESS must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_ID:?EXPECTED_NODE_ID must come from the reviewed identity manifest}"
: "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256:?EXPECTED_PRIV_VALIDATOR_KEY_SHA256 must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_KEY_SHA256:?EXPECTED_NODE_KEY_SHA256 must come from the reviewed identity manifest}"
: "${EXPECTED_GENESIS_SHA256:?EXPECTED_GENESIS_SHA256 must come from the reviewed network manifest}"
: "${OBSERVER_RPC_URL:?OBSERVER_RPC_URL must name an independent CometBFT RPC endpoint}"
: "${OBSERVER_API_URL:?OBSERVER_API_URL must name an independent Cosmos REST endpoint}"
: "${FLY_HANDOFF_CONFIRMATION:?FLY_HANDOFF_CONFIRMATION must acknowledge the exact handoff tuple}"

for dependency in curl fly jq sha256sum; do
  if ! command -v "${dependency}" >/dev/null 2>&1; then
    echo "fly-exact-height-handoff: ${dependency} is required" >&2
    exit 1
  fi
done

is_canonical_height() {
  local value="$1"
  [[ "${value}" =~ ^[1-9][0-9]*$ ]] || return 1
  (( ${#value} < 19 )) && return 0
  # Equal-length decimal strings sort in numeric order; avoid overflowing
  # Bash signed arithmetic before computing H+1 below.
  # shellcheck disable=SC2071
  (( ${#value} == 19 )) &&
    [[ "${value}" < "9223372036854775807" ]]
}
if ! is_canonical_height "${UPGRADE_HEIGHT}" ||
  ! is_canonical_height "${LAST_COMMITTED_HEIGHT}" ||
  ! is_canonical_height "${ATTEMPTED_UPGRADE_HEIGHT}" ||
  (( UPGRADE_HEIGHT < 2 )); then
  echo "fly-exact-height-handoff: upgrade and evidence heights must be canonical positive decimal integers with H >= 2" >&2
  exit 1
fi
expected_last_committed_height="$((UPGRADE_HEIGHT - 1))"
if [[ "${LAST_COMMITTED_HEIGHT}" != "${expected_last_committed_height}" ]] ||
  [[ "${ATTEMPTED_UPGRADE_HEIGHT}" != "${UPGRADE_HEIGHT}" ]]; then
  echo "fly-exact-height-handoff: evidence must report LAST_COMMITTED_HEIGHT=H-1 and ATTEMPTED_UPGRADE_HEIGHT=H" >&2
  exit 1
fi
if [[ ! "${UPGRADE_NAME}" =~ ^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$ ]] ||
  [[ "${UPGRADE_NAME}" == "." || "${UPGRADE_NAME}" == ".." ]]; then
  echo "fly-exact-height-handoff: upgrade name must be canonical lowercase ASCII" >&2
  exit 1
fi
if [[ ! "${FLY_APP}" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]] ||
  [[ ! "${FLY_MACHINE_ID}" =~ ^[0-9a-f]{14}$ ]] ||
  [[ ! "${FLY_VOLUME_ID}" =~ ^vol_[a-z0-9]+$ ]] ||
  [[ ! "${CHAIN_ID}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "fly-exact-height-handoff: app, Machine, volume, or chain identifier is not canonical" >&2
  exit 1
fi
for digest in \
  "${UPGRADE_PLAN_SHA256}" \
  "${ACTIVATION_PREFLIGHT_REPORT_SHA256}" \
  "${UPGRADE_ARM_EVIDENCE_SHA256}" \
  "${LAST_COMMITTED_APP_HASH}" \
  "${EXPECTED_UPGRADE_APP_HASH}" \
  "${FLY_CONFIG_SHA256}" \
  "${UPGRADE_EXIT_EVIDENCE_SHA256}"; do
  if [[ ! "${digest}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "fly-exact-height-handoff: evidence, AppHash, and config digests must be 64 lowercase hexadecimal characters" >&2
    exit 1
  fi
done
if [[ ! "${EXPECTED_VALIDATOR_ADDRESS}" =~ ^[0-9A-F]{40}$ ]] ||
  [[ ! "${EXPECTED_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] ||
  [[ ! "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${EXPECTED_NODE_KEY_SHA256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${EXPECTED_GENESIS_SHA256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "fly-exact-height-handoff: reviewed validator address, node ID, key digests, or genesis digest are not canonical" >&2
  exit 1
fi
for evidence_path in \
  "${UPGRADE_PLAN_EVIDENCE_PATH}" \
  "${ACTIVATION_PREFLIGHT_REPORT_PATH}" \
  "${UPGRADE_ARM_EVIDENCE_PATH}" \
  "${UPGRADE_EXIT_EVIDENCE_PATH}" \
  "${OLD_BINARY_EXIT_LOG_PATH}" \
  "${UPGRADE_INFO_EVIDENCE_PATH}"; do
  if [[ -L "${evidence_path}" || ! -f "${evidence_path}" ]]; then
    echo "fly-exact-height-handoff: evidence artifacts must be regular non-symlink files" >&2
    exit 1
  fi
done
actual_plan_digest="$(sha256sum "${UPGRADE_PLAN_EVIDENCE_PATH}" | awk '{ print $1 }')"
actual_preflight_digest="$(sha256sum "${ACTIVATION_PREFLIGHT_REPORT_PATH}" | awk '{ print $1 }')"
actual_arm_digest="$(sha256sum "${UPGRADE_ARM_EVIDENCE_PATH}" | awk '{ print $1 }')"
actual_exit_digest="$(sha256sum "${UPGRADE_EXIT_EVIDENCE_PATH}" | awk '{ print $1 }')"
actual_exit_log_digest="$(sha256sum "${OLD_BINARY_EXIT_LOG_PATH}" | awk '{ print $1 }')"
actual_upgrade_info_digest="$(sha256sum "${UPGRADE_INFO_EVIDENCE_PATH}" | awk '{ print $1 }')"
if [[ "${actual_plan_digest}" != "${UPGRADE_PLAN_SHA256}" ]] ||
  [[ "${actual_preflight_digest}" != "${ACTIVATION_PREFLIGHT_REPORT_SHA256}" ]] ||
  [[ "${actual_arm_digest}" != "${UPGRADE_ARM_EVIDENCE_SHA256}" ]] ||
  [[ "${actual_exit_digest}" != "${UPGRADE_EXIT_EVIDENCE_SHA256}" ]]; then
  echo "fly-exact-height-handoff: plan, activation-preflight, arm, or exit evidence SHA-256 mismatch" >&2
  exit 1
fi
armed_machine_config_sha256="$(
  jq -er '
    .machine_config_sha256 |
    select(type == "string" and test("^[0-9a-f]{64}$"))
  ' "${UPGRADE_ARM_EVIDENCE_PATH}"
)"
if [[ -L "${FLY_CONFIG_PATH}" || ! -f "${FLY_CONFIG_PATH}" ]]; then
  echo "fly-exact-height-handoff: FLY_CONFIG_PATH must be a regular non-symlink file" >&2
  exit 1
fi
actual_config_digest="$(sha256sum "${FLY_CONFIG_PATH}" | awk '{ print $1 }')"
if [[ "${actual_config_digest}" != "${FLY_CONFIG_SHA256}" ]]; then
  echo "fly-exact-height-handoff: reviewed Fly configuration SHA-256 mismatch" >&2
  exit 1
fi
fly config validate \
  --app "${FLY_APP}" \
  --config "${FLY_CONFIG_PATH}" \
  --strict >/dev/null
target_config_json="$(
  fly config show --local --config "${FLY_CONFIG_PATH}"
)"
if ! jq -e --arg app "${FLY_APP}" '
  type == "object" and .app == $app and
  (.env.ZERONE_HOME == "/data/.zeroned") and
  (.mounts | type == "array" and length == 1) and
  .mounts[0].destination == "/data" and
  ((.restart? == null) or .restart.policy == "no") and
  (.services | type == "array" and length == 1) and
  .services[0].protocol == "tcp" and
  .services[0].internal_port == 26656 and
  (.services[0].ports | type == "array" and length == 1) and
  .services[0].ports[0].port == 26656
' <<<"${target_config_json}" >/dev/null; then
  echo "fly-exact-height-handoff: reviewed Fly config must preserve the /data home and expose only P2P TCP 26656" >&2
  exit 1
fi
for image_ref in "${FLY_CURRENT_IMAGE_REF}" "${FLY_IMAGE_REF}"; do
  if [[ ! "${image_ref}" =~ ^registry\.fly\.io/[A-Za-z0-9._/-]+@sha256:[0-9a-f]{64}$ ]]; then
    echo "fly-exact-height-handoff: image references must be canonical immutable registry.fly.io/...@sha256:<64 lowercase hex> references without tags" >&2
    exit 1
  fi
done
for observer_url in "${OBSERVER_RPC_URL}" "${OBSERVER_API_URL}"; do
  if [[ "${observer_url}" != https://* ]] ||
    [[ "${observer_url}" == *[[:space:]]* ]]; then
    echo "fly-exact-height-handoff: independent observer URLs must use HTTPS and contain no whitespace" >&2
    exit 1
  fi
done
observer_rpc_url="${OBSERVER_RPC_URL%/}"
observer_api_url="${OBSERVER_API_URL%/}"

current_image_repository="${FLY_CURRENT_IMAGE_REF%@sha256:*}"
current_image_digest="${FLY_CURRENT_IMAGE_REF##*@}"
target_image_repository="${FLY_IMAGE_REF%@sha256:*}"
target_image_digest="${FLY_IMAGE_REF##*@}"
image_digest="${target_image_digest#sha256:}"
current_digest="${current_image_digest#sha256:}"
expected_confirmation="${FLY_APP}:${FLY_MACHINE_ID}:${FLY_VOLUME_ID}:${CHAIN_ID}:${current_digest}:${image_digest}:${FLY_CONFIG_SHA256}:${UPGRADE_NAME}:${UPGRADE_HEIGHT}:${UPGRADE_PLAN_SHA256}:${ACTIVATION_PREFLIGHT_REPORT_SHA256}:${UPGRADE_ARM_EVIDENCE_SHA256}:${UPGRADE_EXIT_EVIDENCE_SHA256}:${LAST_COMMITTED_HEIGHT}:${LAST_COMMITTED_APP_HASH}:${ATTEMPTED_UPGRADE_HEIGHT}:${EXPECTED_UPGRADE_APP_HASH}:${EXPECTED_VALIDATOR_ADDRESS}:${EXPECTED_NODE_ID}:${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}:${EXPECTED_NODE_KEY_SHA256}:${EXPECTED_GENESIS_SHA256}"
if [[ "${FLY_HANDOFF_CONFIRMATION}" != "${expected_confirmation}" ]]; then
  echo "fly-exact-height-handoff: confirmation mismatch; expected ${expected_confirmation}" >&2
  exit 1
fi

machine_before_json="$(fly machine list --app "${FLY_APP}" --json)"
if ! jq -e \
  --arg id "${FLY_MACHINE_ID}" \
  --arg volume "${FLY_VOLUME_ID}" \
  --arg repository "${current_image_repository}" \
  --arg digest "${current_image_digest}" '
    type == "array" and length == 1 and
    .[0].id == $id and .[0].state == "stopped" and
    ((.[0].image_ref.registry + "/" + .[0].image_ref.repository) == $repository) and
    .[0].image_ref.digest == $digest and
    .[0].config.restart.policy == "no" and
    (.[0].config.mounts | type == "array" and length == 1) and
    .[0].config.mounts[0].volume == $volume and
    .[0].config.mounts[0].path == "/data" and
    .[0].config.mounts[0].encrypted == true
  ' <<<"${machine_before_json}" >/dev/null; then
  echo "fly-exact-height-handoff: Machine must already be stopped after the old-binary exit, retain restart policy no, and match the confirmed image/encrypted-volume prestate" >&2
  exit 1
fi
machine_before_config="$(
  jq -ceS '.[0].config | del(.restart)' <<<"${machine_before_json}"
)"
machine_before_config_sha256="$(
  printf '%s' "${machine_before_config}" |
    sha256sum |
    awk '{ print $1 }'
)"
if [[ "${machine_before_config_sha256}" != "${armed_machine_config_sha256}" ]]; then
  echo "fly-exact-height-handoff: stopped Machine config drifted from the exact config sealed by pre-H arm evidence" >&2
  exit 1
fi

pre_h_status_json="$(
  curl --fail --silent --show-error --max-time 10 \
    "${observer_rpc_url}/status"
)"
observed_pre_h_chain="$(jq -r '.result.node_info.network // empty' <<<"${pre_h_status_json}")"
observed_pre_h_height="$(jq -r '.result.sync_info.latest_block_height // empty' <<<"${pre_h_status_json}")"
observed_pre_h_apphash="$(
  jq -r '.result.sync_info.latest_app_hash // empty' <<<"${pre_h_status_json}" |
    tr '[:upper:]' '[:lower:]'
)"
if [[ "${observed_pre_h_chain}" != "${CHAIN_ID}" ]] ||
  [[ "${observed_pre_h_height}" != "${LAST_COMMITTED_HEIGHT}" ]] ||
  [[ "${observed_pre_h_apphash}" != "${LAST_COMMITTED_APP_HASH}" ]]; then
  echo "fly-exact-height-handoff: independent observer does not confirm the chain/H-1/AppHash prestate" >&2
  exit 1
fi

observer_rest_node_info="$(
  curl --fail --silent --show-error --max-time 10 \
    "${observer_api_url}/cosmos/base/tendermint/v1beta1/node_info"
)"
observed_rest_chain="$(
  jq -r '.default_node_info.network // empty' <<<"${observer_rest_node_info}"
)"
observer_current_plan="$(
  curl --fail --silent --show-error --max-time 10 \
    "${observer_api_url}/cosmos/upgrade/v1beta1/current_plan"
)"
evidence_plan="$(
  jq -ceS \
    --arg name "${UPGRADE_NAME}" \
    --arg height "${UPGRADE_HEIGHT}" '
      .plan |
      select(.name == $name and (.height | tostring) == $height)
    ' "${UPGRADE_PLAN_EVIDENCE_PATH}"
)"
observed_plan="$(
  jq -ceS \
    --arg name "${UPGRADE_NAME}" \
    --arg height "${UPGRADE_HEIGHT}" '
      .plan |
      select(.name == $name and (.height | tostring) == $height)
    ' <<<"${observer_current_plan}"
)"
if [[ "${observed_rest_chain}" != "${CHAIN_ID}" ]] ||
  [[ "${evidence_plan}" != "${observed_plan}" ]]; then
  echo "fly-exact-height-handoff: independent REST chain/current-plan response does not match the digest-bound plan evidence" >&2
  exit 1
fi

plan_info_sha256="$(
  jq -erj '.info | select(type == "string")' <<<"${evidence_plan}" |
    sha256sum |
    awk '{print $1}'
)"
canonical_preflight_without_digest="$(
  jq -c 'del(.report_sha256)' "${ACTIVATION_PREFLIGHT_REPORT_PATH}"
)"
computed_preflight_report_sha256="$(
  printf '%s' "${canonical_preflight_without_digest}" |
    sha256sum |
    awk '{print $1}'
)"
if ! jq -e \
  --arg chain_id "${CHAIN_ID}" \
  --arg upgrade_name "${UPGRADE_NAME}" \
  --arg upgrade_height "${UPGRADE_HEIGHT}" \
  --arg last_height "${LAST_COMMITTED_HEIGHT}" \
  --arg last_apphash "${LAST_COMMITTED_APP_HASH}" \
  --arg plan_info_sha256 "${plan_info_sha256}" \
  --arg genesis_sha256 "${EXPECTED_GENESIS_SHA256}" \
  --arg upgrade_info_sha256 "${actual_upgrade_info_digest}" \
  --arg report_sha256 "${computed_preflight_report_sha256}" '
    type == "object" and
    .schema == "zerone.activation-preflight/v3" and
    .scope == "scheduled-plan-h-minus-one" and
    .activation_ready == true and
    .chain_id == $chain_id and
    .genesis_sha256 == $genesis_sha256 and
    .upgrade_info_sha256 == $upgrade_info_sha256 and
    .report_sha256 == $report_sha256 and
    .plan_name == $upgrade_name and
    (.plan_height | tostring) == $upgrade_height and
    .plan_info_sha256 == $plan_info_sha256 and
    .blocks_until_activation == 1 and
    (.height | tostring) == $last_height and
    .app_hash == $last_apphash and
    (.unsafe_skip_upgrade_heights | type == "array" and length == 0) and
    (.unsafe_skip_config_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
    (.source_data_manifest_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
    (.source_data_file_count | type == "number" and floor == . and . > 0) and
    (.source_data_bytes | type == "number" and floor == . and . > 0) and
    (.safety_source_versions | type == "object") and
    .safety_source_versions.emergency == 1 and
    .safety_source_versions.gov == 5 and
    .safety_source_versions.zerone_gov == 2 and
    .custom_gov_unattributed_stake_uzrn == "0" and
    (.completed_checks | type == "array") and
    (.completed_checks as $checks |
      ([
        "complete_iavl_roots_bound_to_app_hash",
        "exact_safety_source_versions",
        "sdk_governance_emergency_authority_audit",
        "zero_unattributed_custom_upgrade_stake",
        "effective_unsafe_skip_configuration_bound",
        "scheduled_plan_exact_h_minus_one",
        "named_handler_plan_specific_preconditions",
        "scheduled_height_not_unsafe_skipped",
        "exact_upgrade_handler_cache_dry_run",
        "source_database_never_opened",
        "isolated_copy_manifest_exact",
        "source_manifest_unchanged_after_verification",
        "expected_chain_height_app_hash_tuple_matched",
        "genesis_chain_id_and_digest_bound",
        "local_upgrade_info_exactly_matches_committed_plan"
      ] - $checks | length == 0))
  ' "${ACTIVATION_PREFLIGHT_REPORT_PATH}" >/dev/null; then
  echo "fly-exact-height-handoff: activation-preflight report does not bind the exact v3 chain/genesis/H-1/AppHash/plan/readiness tuple" >&2
  exit 1
fi

upgrade_exit_count="$(
  grep -F -c -- \
    "UPGRADE \"${UPGRADE_NAME}\" NEEDED at height: ${UPGRADE_HEIGHT}:" \
    "${OLD_BINARY_EXIT_LOG_PATH}" || true
)"
if [[ "${upgrade_exit_count}" != "1" ]]; then
  echo "fly-exact-height-handoff: old-binary log must contain exactly one matching upgrade-needed exit" >&2
  exit 1
fi
if ! jq -e \
  --arg name "${UPGRADE_NAME}" \
  --arg height "${UPGRADE_HEIGHT}" \
  --argjson plan "${evidence_plan}" '
    type == "object" and
    .name == $name and
    (.height | tostring) == $height and
    .info == $plan.info
  ' "${UPGRADE_INFO_EVIDENCE_PATH}" >/dev/null; then
  echo "fly-exact-height-handoff: upgrade-info evidence does not match the observed current plan" >&2
  exit 1
fi

if ! jq -e \
  --arg app "${FLY_APP}" \
  --arg machine_id "${FLY_MACHINE_ID}" \
  --arg volume_id "${FLY_VOLUME_ID}" \
  --arg chain_id "${CHAIN_ID}" \
  --arg current_image_ref "${FLY_CURRENT_IMAGE_REF}" \
  --arg upgrade_name "${UPGRADE_NAME}" \
  --arg upgrade_height "${UPGRADE_HEIGHT}" \
  --arg plan_sha256 "${UPGRADE_PLAN_SHA256}" \
  --arg validator_address "${EXPECTED_VALIDATOR_ADDRESS}" \
  --arg node_id "${EXPECTED_NODE_ID}" \
  --arg validator_key_sha256 "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  --arg node_key_sha256 "${EXPECTED_NODE_KEY_SHA256}" \
  --arg genesis_sha256 "${EXPECTED_GENESIS_SHA256}" '
    type == "object" and
    .schema == "zerone.fly-upgrade-arm-evidence/v1" and
    .fly_app == $app and
    .machine_id == $machine_id and
    .volume_id == $volume_id and
    .chain_id == $chain_id and
    .current_image_ref == $current_image_ref and
    .upgrade_name == $upgrade_name and
    (.upgrade_height | tostring) == $upgrade_height and
    .upgrade_plan_sha256 == $plan_sha256 and
    .restart_policy == "no" and
    .validator_address == $validator_address and
    .node_id == $node_id and
    .validator_key_sha256 == $validator_key_sha256 and
    .node_key_sha256 == $node_key_sha256 and
    .genesis_sha256 == $genesis_sha256 and
    (.machine_config_sha256 | type == "string" and test("^[0-9a-f]{64}$")) and
    (.pre_arm_app_hash | type == "string" and test("^[0-9a-f]{64}$")) and
    (.post_arm_app_hash | type == "string" and test("^[0-9a-f]{64}$")) and
    (.armed_by | type == "string" and length > 0) and
    (.armed_at | type == "string" and length > 0)
  ' "${UPGRADE_ARM_EVIDENCE_PATH}" >/dev/null; then
  echo "fly-exact-height-handoff: reviewed arm evidence does not bind the exact no-restart/image/identity tuple" >&2
  exit 1
fi
arm_pre_height="$(jq -r '.pre_arm_height | tostring' "${UPGRADE_ARM_EVIDENCE_PATH}")"
arm_post_height="$(jq -r '.post_arm_height | tostring' "${UPGRADE_ARM_EVIDENCE_PATH}")"
if ! is_canonical_height "${arm_pre_height}" ||
  ! is_canonical_height "${arm_post_height}" ||
  (( arm_pre_height >= UPGRADE_HEIGHT )) ||
  (( arm_post_height >= UPGRADE_HEIGHT )); then
  echo "fly-exact-height-handoff: arm evidence observations must both be strictly before H" >&2
  exit 1
fi

if ! jq -e \
  --arg app "${FLY_APP}" \
  --arg machine_id "${FLY_MACHINE_ID}" \
  --arg volume_id "${FLY_VOLUME_ID}" \
  --arg chain_id "${CHAIN_ID}" \
  --arg current_image_ref "${FLY_CURRENT_IMAGE_REF}" \
  --arg upgrade_name "${UPGRADE_NAME}" \
  --arg upgrade_height "${UPGRADE_HEIGHT}" \
  --arg plan_sha256 "${UPGRADE_PLAN_SHA256}" \
  --arg activation_preflight_sha256 "${ACTIVATION_PREFLIGHT_REPORT_SHA256}" \
  --arg arm_evidence_sha256 "${UPGRADE_ARM_EVIDENCE_SHA256}" \
  --arg last_height "${LAST_COMMITTED_HEIGHT}" \
  --arg last_apphash "${LAST_COMMITTED_APP_HASH}" \
  --arg attempted_height "${ATTEMPTED_UPGRADE_HEIGHT}" \
  --arg validator_address "${EXPECTED_VALIDATOR_ADDRESS}" \
  --arg node_id "${EXPECTED_NODE_ID}" \
  --arg validator_key_sha256 "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  --arg node_key_sha256 "${EXPECTED_NODE_KEY_SHA256}" \
  --arg genesis_sha256 "${EXPECTED_GENESIS_SHA256}" \
  --arg exit_log_sha256 "${actual_exit_log_digest}" \
  --arg upgrade_info_sha256 "${actual_upgrade_info_digest}" '
    type == "object" and
    .schema == "zerone.fly-upgrade-exit-evidence/v1" and
    .fly_app == $app and
    .machine_id == $machine_id and
    .volume_id == $volume_id and
    .chain_id == $chain_id and
    .current_image_ref == $current_image_ref and
    .upgrade_name == $upgrade_name and
    (.upgrade_height | tostring) == $upgrade_height and
    .upgrade_plan_sha256 == $plan_sha256 and
    .activation_preflight_report_sha256 == $activation_preflight_sha256 and
    .arm_evidence_sha256 == $arm_evidence_sha256 and
    (.last_committed_height | tostring) == $last_height and
    .last_committed_app_hash == $last_apphash and
    (.attempted_upgrade_height | tostring) == $attempted_height and
    .validator_address == $validator_address and
    .node_id == $node_id and
    .validator_key_sha256 == $validator_key_sha256 and
    .node_key_sha256 == $node_key_sha256 and
    .genesis_sha256 == $genesis_sha256 and
    .old_binary_exit_count == 1 and
    .old_binary_exit_log_sha256 == $exit_log_sha256 and
    .upgrade_info_sha256 == $upgrade_info_sha256 and
    (.observer | type == "string" and length > 0) and
    (.observed_at | type == "string" and length > 0)
  ' "${UPGRADE_EXIT_EVIDENCE_PATH}" >/dev/null; then
  echo "fly-exact-height-handoff: reviewed exit evidence does not bind the exact old-binary attempt tuple and artifacts" >&2
  exit 1
fi

secrets_json="$(fly secrets list --app "${FLY_APP}" --json)"
if ! jq -e 'type == "array"' <<<"${secrets_json}" >/dev/null; then
  echo "fly-exact-height-handoff: Fly secret census did not return a JSON array" >&2
  exit 1
fi

deployed_secret_present() {
  local name="$1"
  jq -e --arg name "${name}" '
    any(.[]; .name == $name and .status == "Deployed")
  ' <<<"${secrets_json}" >/dev/null
}
config_env_present() {
  local name="$1"
  jq -e --arg name "${name}" '
    .env[$name] | type == "string" and length > 0
  ' <<<"${target_config_json}" >/dev/null
}
runtime_setting_present() {
  local name="$1"
  deployed_secret_present "${name}" || config_env_present "${name}"
}

# Base64 key values must be deployed secrets, never plaintext Machine config.
for name in PRIV_VALIDATOR_KEY_B64 NODE_KEY_B64; do
  if config_env_present "${name}"; then
    echo "fly-exact-height-handoff: ${name} must not be plaintext Machine config" >&2
    exit 1
  fi
done

# The public identity manifest values are operator-reviewed handoff inputs, not
# secrets from the same Fly control plane as the private keys. If a reviewed
# config already carries them it must agree exactly; they are also supplied
# explicitly on the target Machine update and verified from Machine state.
for tuple in \
  "EXPECTED_VALIDATOR_ADDRESS:${EXPECTED_VALIDATOR_ADDRESS}" \
  "EXPECTED_NODE_ID:${EXPECTED_NODE_ID}" \
  "EXPECTED_PRIV_VALIDATOR_KEY_SHA256:${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  "EXPECTED_NODE_KEY_SHA256:${EXPECTED_NODE_KEY_SHA256}"; do
  identity_name="${tuple%%:*}"
  identity_value="${tuple#*:}"
  if deployed_secret_present "${identity_name}"; then
    echo "fly-exact-height-handoff: ${identity_name} must come from the independently reviewed handoff, not a Fly secret" >&2
    exit 1
  fi
  if config_env_present "${identity_name}" &&
    [[ "$(jq -r --arg name "${identity_name}" '.env[$name]' <<<"${target_config_json}")" != "${identity_value}" ]]; then
    echo "fly-exact-height-handoff: reviewed Fly config conflicts with ${identity_name}" >&2
    exit 1
  fi
done

validator_sources=0
node_sources=0
runtime_setting_present PRIV_VALIDATOR_KEY_FILE && ((validator_sources += 1))
runtime_setting_present PRIV_VALIDATOR_KEY_B64 && ((validator_sources += 1))
runtime_setting_present NODE_KEY_FILE && ((node_sources += 1))
runtime_setting_present NODE_KEY_B64 && ((node_sources += 1))
if (( validator_sources != 1 || node_sources != 1 )) ||
  ! runtime_setting_present PRIV_VALIDATOR_KEY_SHA256 ||
  ! runtime_setting_present NODE_KEY_SHA256; then
  echo "fly-exact-height-handoff: exactly one deployed key source and one digest are required for both validator and node identities" >&2
  exit 1
fi

# Visible mounted-file settings can be validated before the target update.
for tuple in \
  "PRIV_VALIDATOR_KEY_FILE:PRIV_VALIDATOR_KEY_SHA256" \
  "NODE_KEY_FILE:NODE_KEY_SHA256"; do
  file_name="${tuple%%:*}"
  digest_name="${tuple##*:}"
  if config_env_present "${file_name}"; then
    if ! jq -e --arg file_name "${file_name}" --arg digest_name "${digest_name}" '
      .env as $env |
      ($env[$file_name] | startswith("/data/")) and
      ($env[$digest_name] | type == "string" and test("^[0-9a-f]{64}$"))
    ' <<<"${target_config_json}" >/dev/null; then
      echo "fly-exact-height-handoff: visible mounted-file key settings must be under /data and include canonical digests" >&2
      exit 1
    fi
  fi
done

printf '%s\n' \
  "fly-exact-height-handoff: preflight ${FLY_APP}/${FLY_MACHINE_ID}" \
  "  volume=${FLY_VOLUME_ID}" \
  "  current_image=${FLY_CURRENT_IMAGE_REF}" \
  "  target_image=${FLY_IMAGE_REF}" \
  "  plan=${UPGRADE_NAME}@${UPGRADE_HEIGHT} plan_sha256=${UPGRADE_PLAN_SHA256}" \
  "  activation_preflight_sha256=${ACTIVATION_PREFLIGHT_REPORT_SHA256}" \
  "  arm_evidence_sha256=${UPGRADE_ARM_EVIDENCE_SHA256}" \
  "  exit_evidence_sha256=${UPGRADE_EXIT_EVIDENCE_SHA256}" \
  "  validator_address=${EXPECTED_VALIDATOR_ADDRESS} node_id=${EXPECTED_NODE_ID}" \
  "  last_commit=${LAST_COMMITTED_HEIGHT}:${LAST_COMMITTED_APP_HASH}" \
  "  attempted_height=${ATTEMPTED_UPGRADE_HEIGHT}" \
  "  expected_h_apphash=${EXPECTED_UPGRADE_APP_HASH}"

# --skip-start prevents an update failure or a human verification pause from
# accidentally launching a non-verified image. The persistent validator volume
# remains attached to this same Machine.
fly machine update "${FLY_MACHINE_ID}" \
  --app "${FLY_APP}" \
  --config "${FLY_CONFIG_PATH}" \
  --image "${FLY_IMAGE_REF}" \
  --restart no \
  --env "EXPECTED_VALIDATOR_ADDRESS=${EXPECTED_VALIDATOR_ADDRESS}" \
  --env "EXPECTED_NODE_ID=${EXPECTED_NODE_ID}" \
  --env "EXPECTED_PRIV_VALIDATOR_KEY_SHA256=${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  --env "EXPECTED_NODE_KEY_SHA256=${EXPECTED_NODE_KEY_SHA256}" \
  --skip-start \
  --yes

machine_staged_json="$(fly machine list --app "${FLY_APP}" --json)"
if ! jq -e \
  --arg id "${FLY_MACHINE_ID}" \
  --arg volume "${FLY_VOLUME_ID}" \
  --arg repository "${target_image_repository}" \
  --arg digest "${target_image_digest}" \
  --arg validator_address "${EXPECTED_VALIDATOR_ADDRESS}" \
  --arg node_id "${EXPECTED_NODE_ID}" \
  --arg validator_key_sha256 "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  --arg node_key_sha256 "${EXPECTED_NODE_KEY_SHA256}" '
    type == "array" and length == 1 and
    .[0].id == $id and .[0].state == "stopped" and
    ((.[0].image_ref.registry + "/" + .[0].image_ref.repository) == $repository) and
    .[0].image_ref.digest == $digest and
    .[0].config.restart.policy == "no" and
    .[0].config.env.EXPECTED_VALIDATOR_ADDRESS == $validator_address and
    .[0].config.env.EXPECTED_NODE_ID == $node_id and
    .[0].config.env.EXPECTED_PRIV_VALIDATOR_KEY_SHA256 == $validator_key_sha256 and
    .[0].config.env.EXPECTED_NODE_KEY_SHA256 == $node_key_sha256 and
    (.[0].config.mounts | type == "array" and length == 1) and
    .[0].config.mounts[0].volume == $volume and
    .[0].config.mounts[0].path == "/data" and
    .[0].config.mounts[0].encrypted == true and
    (.[0].config.services | type == "array" and length == 1) and
    .[0].config.services[0].protocol == "tcp" and
    .[0].config.services[0].internal_port == 26656 and
    (.[0].config.services[0].ports | type == "array" and length == 1) and
    .[0].config.services[0].ports[0].port == 26656
  ' <<<"${machine_staged_json}" >/dev/null; then
  echo "fly-exact-height-handoff: staged Machine does not exactly match the target image, identity manifest, no-restart policy, encrypted volume, stopped state, and P2P-only service policy; leaving it stopped" >&2
  exit 1
fi

fail_stop_after_start() {
  local original_status="$?"
  local stopped_json
  trap - EXIT INT TERM
  set +e
  echo "fly-exact-height-handoff: post-start gate failed; stopping ${FLY_MACHINE_ID} with restart policy no" >&2
  fly machine stop "${FLY_MACHINE_ID}" \
    --app "${FLY_APP}" \
    --signal SIGINT \
    --timeout 30 \
    --wait-timeout 1m >&2
  stopped_json="$(fly machine list --app "${FLY_APP}" --json 2>/dev/null)"
  if ! jq -e \
    --arg id "${FLY_MACHINE_ID}" '
      type == "array" and length == 1 and
      .[0].id == $id and .[0].state == "stopped" and
      .[0].config.restart.policy == "no"
    ' <<<"${stopped_json}" >/dev/null 2>&1; then
    echo "fly-exact-height-handoff: graceful fail-stop was not confirmed; requesting SIGKILL" >&2
    fly machine stop "${FLY_MACHINE_ID}" \
      --app "${FLY_APP}" \
      --signal SIGKILL \
      --timeout 5 \
      --wait-timeout 1m >&2
    stopped_json="$(fly machine list --app "${FLY_APP}" --json 2>/dev/null)"
  fi
  if ! jq -e \
    --arg id "${FLY_MACHINE_ID}" '
      type == "array" and length == 1 and
      .[0].id == $id and .[0].state == "stopped" and
      .[0].config.restart.policy == "no"
    ' <<<"${stopped_json}" >/dev/null 2>&1; then
    echo "fly-exact-height-handoff: CRITICAL: control plane did not confirm the signer stopped; invoke incident isolation immediately" >&2
  fi
  if (( original_status == 0 )); then
    original_status=1
  fi
  exit "${original_status}"
}
trap fail_stop_after_start EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fly machine start "${FLY_MACHINE_ID}" --app "${FLY_APP}"

machine_started=false
for ((attempt = 1; attempt <= 30; attempt++)); do
  machine_started_json="$(fly machine list --app "${FLY_APP}" --json)"
  if jq -e \
    --arg id "${FLY_MACHINE_ID}" \
    --arg volume "${FLY_VOLUME_ID}" \
    --arg repository "${target_image_repository}" \
    --arg digest "${target_image_digest}" \
    --arg validator_address "${EXPECTED_VALIDATOR_ADDRESS}" \
    --arg node_id "${EXPECTED_NODE_ID}" \
    --arg validator_key_sha256 "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
    --arg node_key_sha256 "${EXPECTED_NODE_KEY_SHA256}" '
      type == "array" and length == 1 and
      .[0].id == $id and .[0].state == "started" and
      ((.[0].image_ref.registry + "/" + .[0].image_ref.repository) == $repository) and
      .[0].image_ref.digest == $digest and
      .[0].config.restart.policy == "no" and
      .[0].config.env.EXPECTED_VALIDATOR_ADDRESS == $validator_address and
      .[0].config.env.EXPECTED_NODE_ID == $node_id and
      .[0].config.env.EXPECTED_PRIV_VALIDATOR_KEY_SHA256 == $validator_key_sha256 and
      .[0].config.env.EXPECTED_NODE_KEY_SHA256 == $node_key_sha256 and
      (.[0].config.mounts | type == "array" and length == 1) and
      .[0].config.mounts[0].volume == $volume and
      .[0].config.mounts[0].path == "/data" and
      .[0].config.mounts[0].encrypted == true and
      (.[0].config.services | type == "array" and length == 1) and
      .[0].config.services[0].protocol == "tcp" and
      .[0].config.services[0].internal_port == 26656 and
      (.[0].config.services[0].ports | type == "array" and length == 1) and
      .[0].config.services[0].ports[0].port == 26656
    ' <<<"${machine_started_json}" >/dev/null; then
    machine_started=true
    break
  fi
  sleep 2
done
if [[ "${machine_started}" != true ]]; then
  echo "fly-exact-height-handoff: Machine did not reach the exact confirmed started state" >&2
  exit 1
fi

# Header H+1 commits the application hash produced by H. Query it and the
# immutable x/upgrade done marker through endpoints independent of the signer.
verification_height="$((UPGRADE_HEIGHT + 1))"
chain_verified=false
for ((attempt = 1; attempt <= 60; attempt++)); do
  if status_json="$(
    curl --fail --silent --max-time 10 "${observer_rpc_url}/status" 2>/dev/null
  )" &&
    commit_json="$(
      curl --fail --silent --max-time 10 \
        "${observer_rpc_url}/commit?height=${verification_height}" 2>/dev/null
    )" &&
    applied_json="$(
      curl --fail --silent --max-time 10 \
        "${observer_api_url}/cosmos/upgrade/v1beta1/applied_plan/${UPGRADE_NAME}" 2>/dev/null
    )" &&
    post_rest_node_info="$(
      curl --fail --silent --max-time 10 \
        "${observer_api_url}/cosmos/base/tendermint/v1beta1/node_info" 2>/dev/null
    )"; then
    observed_chain_id="$(jq -r '.result.node_info.network // empty' <<<"${status_json}")"
    observed_commit_chain_id="$(
      jq -r '.result.signed_header.header.chain_id // empty' <<<"${commit_json}"
    )"
    observed_commit_height="$(
      jq -r '.result.signed_header.header.height // empty' <<<"${commit_json}"
    )"
    observed_h_apphash="$(
      jq -r '.result.signed_header.header.app_hash // empty' <<<"${commit_json}" |
        tr '[:upper:]' '[:lower:]'
    )"
    observed_applied_height="$(jq -r '.height // empty' <<<"${applied_json}")"
    observed_post_rest_chain="$(
      jq -r '.default_node_info.network // empty' <<<"${post_rest_node_info}"
    )"
    if [[ "${observed_chain_id}" == "${CHAIN_ID}" ]] &&
      [[ "${observed_commit_chain_id}" == "${CHAIN_ID}" ]] &&
      [[ "${observed_commit_height}" == "${verification_height}" ]] &&
      [[ "${observed_h_apphash}" == "${EXPECTED_UPGRADE_APP_HASH}" ]] &&
      [[ "${observed_post_rest_chain}" == "${CHAIN_ID}" ]] &&
      [[ "${observed_applied_height}" == "${UPGRADE_HEIGHT}" ]]; then
      chain_verified=true
      break
    fi
  fi
  sleep 2
done
if [[ "${chain_verified}" != true ]]; then
  echo "fly-exact-height-handoff: Machine started, but independent H AppHash/chain/upgrade-marker verification failed; keep admission closed and enter incident review" >&2
  exit 1
fi

trap - EXIT INT TERM
echo "fly-exact-height-handoff: independently verified ${UPGRADE_NAME} at H=${UPGRADE_HEIGHT}, AppHash=${EXPECTED_UPGRADE_APP_HASH}, Machine=${FLY_MACHINE_ID}, image=${FLY_IMAGE_REF}, volume=${FLY_VOLUME_ID}"
