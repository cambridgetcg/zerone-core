#!/usr/bin/env bash
# Arm a Fly validator before H so x/upgrade's intentional old-binary exit
# leaves the Machine stopped instead of entering Fly's restart loop.
set -euo pipefail
umask 077

: "${FLY_APP:?FLY_APP must name the validator app}"
: "${FLY_MACHINE_ID:?FLY_MACHINE_ID must name the one validator Machine}"
: "${FLY_VOLUME_ID:?FLY_VOLUME_ID must name the attached validator volume}"
: "${FLY_CURRENT_IMAGE_REF:?FLY_CURRENT_IMAGE_REF must bind the running immutable image}"
: "${CHAIN_ID:?CHAIN_ID must match the observer network exactly}"
: "${UPGRADE_NAME:?UPGRADE_NAME must match the on-chain plan exactly}"
: "${UPGRADE_HEIGHT:?UPGRADE_HEIGHT must match the on-chain plan exactly}"
: "${UPGRADE_PLAN_EVIDENCE_PATH:?UPGRADE_PLAN_EVIDENCE_PATH must contain the reviewed current-plan response}"
: "${UPGRADE_PLAN_SHA256:?UPGRADE_PLAN_SHA256 must bind the reviewed current-plan response}"
: "${PRE_ARM_HEIGHT:?PRE_ARM_HEIGHT must come from the independent pre-H observation}"
: "${PRE_ARM_APP_HASH:?PRE_ARM_APP_HASH must bind the independent pre-H state}"
: "${EXPECTED_VALIDATOR_ADDRESS:?EXPECTED_VALIDATOR_ADDRESS must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_ID:?EXPECTED_NODE_ID must come from the reviewed identity manifest}"
: "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256:?EXPECTED_PRIV_VALIDATOR_KEY_SHA256 must come from the reviewed identity manifest}"
: "${EXPECTED_NODE_KEY_SHA256:?EXPECTED_NODE_KEY_SHA256 must come from the reviewed identity manifest}"
: "${EXPECTED_GENESIS_SHA256:?EXPECTED_GENESIS_SHA256 must come from the reviewed network manifest}"
: "${OBSERVER_RPC_URL:?OBSERVER_RPC_URL must name an independent CometBFT RPC endpoint}"
: "${OBSERVER_API_URL:?OBSERVER_API_URL must name an independent Cosmos REST endpoint}"
: "${ARMED_BY:?ARMED_BY must identify the independent operator}"
: "${ARMED_AT:?ARMED_AT must be a reviewed RFC3339 timestamp}"
: "${FLY_ARM_EVIDENCE_OUTPUT:?FLY_ARM_EVIDENCE_OUTPUT must name a new evidence file}"
: "${FLY_ARM_CONFIRMATION:?FLY_ARM_CONFIRMATION must acknowledge the exact arming tuple}"

for dependency in curl fly jq sha256sum; do
  if ! command -v "${dependency}" >/dev/null 2>&1; then
    echo "fly-arm-exact-height-handoff: ${dependency} is required" >&2
    exit 1
  fi
done

is_canonical_height() {
  local value="$1"
  [[ "${value}" =~ ^[1-9][0-9]*$ ]] || return 1
  (( ${#value} < 19 )) && return 0
  # shellcheck disable=SC2071
  [[ ${#value} -eq 19 && "${value}" < "9223372036854775807" ]]
}
if ! is_canonical_height "${UPGRADE_HEIGHT}" ||
  ! is_canonical_height "${PRE_ARM_HEIGHT}" ||
  (( PRE_ARM_HEIGHT >= UPGRADE_HEIGHT )); then
  echo "fly-arm-exact-height-handoff: the independently observed arming height must be canonical and strictly before H" >&2
  exit 1
fi
if [[ ! "${UPGRADE_NAME}" =~ ^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$ ]] ||
  [[ "${UPGRADE_NAME}" == "." || "${UPGRADE_NAME}" == ".." ]] ||
  [[ ! "${FLY_APP}" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]] ||
  [[ ! "${FLY_MACHINE_ID}" =~ ^[0-9a-f]{14}$ ]] ||
  [[ ! "${FLY_VOLUME_ID}" =~ ^vol_[a-z0-9]+$ ]] ||
  [[ ! "${CHAIN_ID}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "fly-arm-exact-height-handoff: app, Machine, volume, chain, or upgrade identifier is not canonical" >&2
  exit 1
fi
if [[ ! "${FLY_CURRENT_IMAGE_REF}" =~ ^registry\.fly\.io/[A-Za-z0-9._/-]+@sha256:[0-9a-f]{64}$ ]] ||
  [[ ! "${UPGRADE_PLAN_SHA256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${PRE_ARM_APP_HASH}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${EXPECTED_VALIDATOR_ADDRESS}" =~ ^[0-9A-F]{40}$ ]] ||
  [[ ! "${EXPECTED_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] ||
  [[ ! "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${EXPECTED_NODE_KEY_SHA256}" =~ ^[0-9a-f]{64}$ ]] ||
  [[ ! "${EXPECTED_GENESIS_SHA256}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "fly-arm-exact-height-handoff: image, plan, AppHash, or reviewed identity value is not canonical" >&2
  exit 1
fi
if [[ -L "${UPGRADE_PLAN_EVIDENCE_PATH}" || ! -f "${UPGRADE_PLAN_EVIDENCE_PATH}" ]] ||
  [[ -L "${FLY_ARM_EVIDENCE_OUTPUT}" || -e "${FLY_ARM_EVIDENCE_OUTPUT}" ]] ||
  [[ ! -d "$(dirname "${FLY_ARM_EVIDENCE_OUTPUT}")" ]] ||
  [[ -L "$(dirname "${FLY_ARM_EVIDENCE_OUTPUT}")" ]]; then
  echo "fly-arm-exact-height-handoff: plan input must be a regular file and evidence output must be a new file in a regular directory" >&2
  exit 1
fi
if [[ "$(sha256sum "${UPGRADE_PLAN_EVIDENCE_PATH}" | awk '{print $1}')" != "${UPGRADE_PLAN_SHA256}" ]]; then
  echo "fly-arm-exact-height-handoff: reviewed plan evidence SHA-256 mismatch" >&2
  exit 1
fi
for observer_url in "${OBSERVER_RPC_URL}" "${OBSERVER_API_URL}"; do
  if [[ "${observer_url}" != https://* ]] ||
    [[ "${observer_url}" == *[[:space:]]* ]]; then
    echo "fly-arm-exact-height-handoff: independent observer URLs must use HTTPS and contain no whitespace" >&2
    exit 1
  fi
done
if [[ -z "${ARMED_BY}" || "${ARMED_BY}" == *$'\n'* ]] ||
  [[ ! "${ARMED_AT}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
  echo "fly-arm-exact-height-handoff: operator identity or reviewed UTC timestamp is invalid" >&2
  exit 1
fi

current_repository="${FLY_CURRENT_IMAGE_REF%@sha256:*}"
current_image_digest="${FLY_CURRENT_IMAGE_REF##*@}"
current_digest="${current_image_digest#sha256:}"
expected_confirmation="arm-no-restart:${FLY_APP}:${FLY_MACHINE_ID}:${FLY_VOLUME_ID}:${CHAIN_ID}:${current_digest}:${UPGRADE_NAME}:${UPGRADE_HEIGHT}:${UPGRADE_PLAN_SHA256}:${PRE_ARM_HEIGHT}:${PRE_ARM_APP_HASH}:${EXPECTED_VALIDATOR_ADDRESS}:${EXPECTED_NODE_ID}:${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}:${EXPECTED_NODE_KEY_SHA256}:${EXPECTED_GENESIS_SHA256}"
if [[ "${FLY_ARM_CONFIRMATION}" != "${expected_confirmation}" ]]; then
  echo "fly-arm-exact-height-handoff: confirmation mismatch; expected ${expected_confirmation}" >&2
  exit 1
fi

machine_before_json="$(fly machine list --app "${FLY_APP}" --json)"
if ! jq -e \
  --arg id "${FLY_MACHINE_ID}" \
  --arg volume "${FLY_VOLUME_ID}" \
  --arg repository "${current_repository}" \
  --arg digest "${current_image_digest}" '
    type == "array" and length == 1 and
    .[0].id == $id and .[0].state == "started" and
    ((.[0].image_ref.registry + "/" + .[0].image_ref.repository) == $repository) and
    .[0].image_ref.digest == $digest and
    (.[0].config.mounts | type == "array" and length == 1) and
    .[0].config.mounts[0].volume == $volume and
    .[0].config.mounts[0].path == "/data" and
    .[0].config.mounts[0].encrypted == true
  ' <<<"${machine_before_json}" >/dev/null; then
  echo "fly-arm-exact-height-handoff: running Machine does not match the reviewed old-image/encrypted-volume tuple" >&2
  exit 1
fi
before_config="$(
  jq -ceS '.[0].config | del(.restart)' <<<"${machine_before_json}"
)"
before_config_sha256="$(
  printf '%s' "${before_config}" | sha256sum | awk '{print $1}'
)"

observer_rpc_url="${OBSERVER_RPC_URL%/}"
observer_api_url="${OBSERVER_API_URL%/}"
pre_status="$(
  curl --fail --silent --show-error --max-time 10 "${observer_rpc_url}/status"
)"
observed_pre_chain="$(jq -r '.result.node_info.network // empty' <<<"${pre_status}")"
observed_pre_height="$(jq -r '.result.sync_info.latest_block_height // empty' <<<"${pre_status}")"
observed_pre_apphash="$(
  jq -r '.result.sync_info.latest_app_hash // empty' <<<"${pre_status}" |
    tr '[:upper:]' '[:lower:]'
)"
rest_node_info="$(
  curl --fail --silent --show-error --max-time 10 \
    "${observer_api_url}/cosmos/base/tendermint/v1beta1/node_info"
)"
observed_rest_chain="$(jq -r '.default_node_info.network // empty' <<<"${rest_node_info}")"
current_plan="$(
  curl --fail --silent --show-error --max-time 10 \
    "${observer_api_url}/cosmos/upgrade/v1beta1/current_plan"
)"
evidence_plan="$(
  jq -ceS \
    --arg name "${UPGRADE_NAME}" \
    --arg height "${UPGRADE_HEIGHT}" '
      .plan | select(.name == $name and (.height | tostring) == $height)
    ' "${UPGRADE_PLAN_EVIDENCE_PATH}"
)"
observed_plan="$(
  jq -ceS \
    --arg name "${UPGRADE_NAME}" \
    --arg height "${UPGRADE_HEIGHT}" '
      .plan | select(.name == $name and (.height | tostring) == $height)
    ' <<<"${current_plan}"
)"
if [[ "${observed_pre_chain}" != "${CHAIN_ID}" ]] ||
  [[ "${observed_rest_chain}" != "${CHAIN_ID}" ]] ||
  [[ "${observed_pre_height}" != "${PRE_ARM_HEIGHT}" ]] ||
  [[ "${observed_pre_apphash}" != "${PRE_ARM_APP_HASH}" ]] ||
  [[ "${evidence_plan}" != "${observed_plan}" ]]; then
  echo "fly-arm-exact-height-handoff: independent pre-H chain/plan/state observation does not match the reviewed tuple" >&2
  exit 1
fi

# This is the only mutation in the arming step. It must happen before H. No
# image, volume, service, environment, or other Machine config is supplied.
fly machine update "${FLY_MACHINE_ID}" \
  --app "${FLY_APP}" \
  --restart no \
  --yes

machine_after_json="$(fly machine list --app "${FLY_APP}" --json)"
if ! jq -e \
  --arg id "${FLY_MACHINE_ID}" \
  --arg volume "${FLY_VOLUME_ID}" \
  --arg repository "${current_repository}" \
  --arg digest "${current_image_digest}" '
    type == "array" and length == 1 and
    .[0].id == $id and .[0].state == "started" and
    ((.[0].image_ref.registry + "/" + .[0].image_ref.repository) == $repository) and
    .[0].image_ref.digest == $digest and
    .[0].config.restart.policy == "no" and
    .[0].config.mounts[0].volume == $volume and
    .[0].config.mounts[0].path == "/data" and
    .[0].config.mounts[0].encrypted == true
  ' <<<"${machine_after_json}" >/dev/null; then
  echo "fly-arm-exact-height-handoff: restart arming did not preserve the exact running old-image/encrypted-volume tuple" >&2
  exit 1
fi
after_config="$(
  jq -ceS '.[0].config | del(.restart)' <<<"${machine_after_json}"
)"
after_config_sha256="$(
  printf '%s' "${after_config}" | sha256sum | awk '{print $1}'
)"
if [[ "${after_config}" != "${before_config}" ]]; then
  echo "fly-arm-exact-height-handoff: arming changed Machine config beyond restart policy" >&2
  exit 1
fi

post_status="$(
  curl --fail --silent --show-error --max-time 10 "${observer_rpc_url}/status"
)"
observed_post_chain="$(jq -r '.result.node_info.network // empty' <<<"${post_status}")"
observed_post_height="$(jq -r '.result.sync_info.latest_block_height // empty' <<<"${post_status}")"
observed_post_apphash="$(
  jq -r '.result.sync_info.latest_app_hash // empty' <<<"${post_status}" |
    tr '[:upper:]' '[:lower:]'
)"
if [[ "${observed_post_chain}" != "${CHAIN_ID}" ]] ||
  ! is_canonical_height "${observed_post_height}" ||
  (( observed_post_height >= UPGRADE_HEIGHT )) ||
  [[ ! "${observed_post_apphash}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "fly-arm-exact-height-handoff: post-arm observer no longer proves a canonical state strictly before H" >&2
  exit 1
fi

output_dir="$(dirname "${FLY_ARM_EVIDENCE_OUTPUT}")"
temporary="$(mktemp "${output_dir}/.fly-arm-evidence.XXXXXX")"
cleanup() {
  rm -f "${temporary}"
}
trap cleanup EXIT
jq -nS \
  --arg schema "zerone.fly-upgrade-arm-evidence/v1" \
  --arg fly_app "${FLY_APP}" \
  --arg machine_id "${FLY_MACHINE_ID}" \
  --arg volume_id "${FLY_VOLUME_ID}" \
  --arg chain_id "${CHAIN_ID}" \
  --arg current_image_ref "${FLY_CURRENT_IMAGE_REF}" \
  --arg upgrade_name "${UPGRADE_NAME}" \
  --arg upgrade_height "${UPGRADE_HEIGHT}" \
  --arg upgrade_plan_sha256 "${UPGRADE_PLAN_SHA256}" \
  --arg pre_arm_height "${PRE_ARM_HEIGHT}" \
  --arg pre_arm_app_hash "${PRE_ARM_APP_HASH}" \
  --arg post_arm_height "${observed_post_height}" \
  --arg post_arm_app_hash "${observed_post_apphash}" \
  --arg config_sha256 "${before_config_sha256}" \
  --arg validator_address "${EXPECTED_VALIDATOR_ADDRESS}" \
  --arg node_id "${EXPECTED_NODE_ID}" \
  --arg validator_key_sha256 "${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}" \
  --arg node_key_sha256 "${EXPECTED_NODE_KEY_SHA256}" \
  --arg genesis_sha256 "${EXPECTED_GENESIS_SHA256}" \
  --arg armed_by "${ARMED_BY}" \
  --arg armed_at "${ARMED_AT}" '
    {
      schema: $schema,
      fly_app: $fly_app,
      machine_id: $machine_id,
      volume_id: $volume_id,
      chain_id: $chain_id,
      current_image_ref: $current_image_ref,
      upgrade_name: $upgrade_name,
      upgrade_height: $upgrade_height,
      upgrade_plan_sha256: $upgrade_plan_sha256,
      pre_arm_height: $pre_arm_height,
      pre_arm_app_hash: $pre_arm_app_hash,
      post_arm_height: $post_arm_height,
      post_arm_app_hash: $post_arm_app_hash,
      machine_config_sha256: $config_sha256,
      restart_policy: "no",
      validator_address: $validator_address,
      node_id: $node_id,
      validator_key_sha256: $validator_key_sha256,
      node_key_sha256: $node_key_sha256,
      genesis_sha256: $genesis_sha256,
      armed_by: $armed_by,
      armed_at: $armed_at
    }
  ' > "${temporary}"
chmod 0600 "${temporary}"
mv "${temporary}" "${FLY_ARM_EVIDENCE_OUTPUT}"
trap - EXIT
evidence_sha256="$(sha256sum "${FLY_ARM_EVIDENCE_OUTPUT}" | awk '{print $1}')"

printf '%s\n' \
  "fly-arm-exact-height-handoff: restart policy is no; exact old signer tuple remained at pre-H" \
  "  evidence=${FLY_ARM_EVIDENCE_OUTPUT}" \
  "  evidence_sha256=${evidence_sha256}" \
  "  config_sha256_before=${before_config_sha256}" \
  "  config_sha256_after=${after_config_sha256}"
