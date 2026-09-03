#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'fly-cutover-authorized: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  deploy/mainnet/fly-cutover-authorized.sh [--check] observer \
    RELEASE.json RELEASE.json.sig CUTOVER.json CUTOVER.json.sig \
    fly.halt-signer.toml fly.observer.toml EXPECTED_MAIN_FINGERPRINT \
    AUTHORITY_BUNDLE_DIRECTORY SIGNER_PRIVATE_RPC OBSERVER_PRIVATE_RPC

  deploy/mainnet/fly-cutover-authorized.sh [--check] signer \
    RELEASE.json RELEASE.json.sig CUTOVER.json CUTOVER.json.sig \
    CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
    fly.halt-signer.toml fly.observer.toml EXPECTED_MAIN_FINGERPRINT \
    AUTHORITY_BUNDLE_DIRECTORY SIGNER_PRIVATE_RPC OBSERVER_PRIVATE_RPC

The observer stage is the only CUTOVER path allowed before the successor-link
transaction. It verifies both halt configs first, deploys only the observer,
then proves signer/observer identity, trusted-block agreement, live-chain
agreement, and the signed lead before F. After the exact transaction commits
and signed initiation evidence exists, the signer stage repeats those checks,
deploys the signer last, and rechecks both nodes. Generic per-config CUTOVER
deployment is intentionally rejected.
USAGE
  exit 2
}

MODE=deploy
if [ "${1:-}" = --check ]; then
  MODE=check
  shift
fi
STAGE=${1:-}
[ -n "${STAGE}" ] || usage
shift
case "${STAGE}|$#" in
  observer\|10)
    RELEASE=$1
    RELEASE_SIG=$2
    CUTOVER=$3
    CUTOVER_SIG=$4
    SIGNER_CONFIG=$5
    OBSERVER_CONFIG=$6
    EXPECTED_SIGNER=$7
    AUTHORITY_BUNDLE=$8
    SIGNER_RPC=$9
    OBSERVER_RPC=${10}
    ;;
  signer\|12)
    RELEASE=$1
    RELEASE_SIG=$2
    CUTOVER=$3
    CUTOVER_SIG=$4
    CUTOVER_INIT=$5
    CUTOVER_INIT_SIG=$6
    SIGNER_CONFIG=$7
    OBSERVER_CONFIG=$8
    EXPECTED_SIGNER=$9
    AUTHORITY_BUNDLE=${10}
    SIGNER_RPC=${11}
    OBSERVER_RPC=${12}
    ;;
  *) usage ;;
esac

[[ "${EXPECTED_SIGNER}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "main signer must be a full 40- or 64-hex fingerprint"
PRIVATE_RPC_RE='^https?://(localhost|127\.0\.0\.1|[a-z0-9]([a-z0-9.-]*[a-z0-9])?\.internal|10\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}|192\.168\.[0-9]{1,3}\.[0-9]{1,3}|172\.(1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3})(:[0-9]{1,5})?/?$'
for rpc in "${SIGNER_RPC}" "${OBSERVER_RPC}"; do
  [[ "${rpc}" =~ ${PRIVATE_RPC_RE} ]] || \
    die "both RPC origins must be loopback, RFC1918, or .internal HTTP(S)"
done
[ "${SIGNER_RPC%/}" != "${OBSERVER_RPC%/}" ] || \
  die "signer and observer RPC origins must differ"

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)
VERIFIER="${ROOT}/deploy/verify-authority-chain.py"
POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
PINNED_GATE="${ROOT}/deploy/fly-deploy-pinned.sh"
TX_GATE="${ROOT}/scripts/zerone-phase-tx-broadcast.sh"

require_regular() {
  local path=$1 label=$2
  [ -f "${path}" ] || die "${label} is not a regular file"
  [ ! -L "${path}" ] || die "${label} must not be a symlink"
}
for pair in \
  "${RELEASE}|release packet" "${RELEASE_SIG}|release signature" \
  "${CUTOVER}|CUTOVER decision" "${CUTOVER_SIG}|CUTOVER signature" \
  "${SIGNER_CONFIG}|halt-signer config" "${OBSERVER_CONFIG}|observer config" \
  "${VERIFIER}|authority verifier" "${POLICY}|config policy" \
  "${PINNED_GATE}|pinned Fly gate" "${TX_GATE}|signed transaction gate" \
  "${AUTHORITY_BUNDLE}/zeroned-zerone-1-release|zerone-1 release binary" \
  "${AUTHORITY_BUNDLE}/CUTOVER-SIGNED-TX.json|CUTOVER signed transaction"; do
  require_regular "${pair%%|*}" "${pair##*|}"
done
if [ "${STAGE}" = signer ]; then
  require_regular "${CUTOVER_INIT}" "CUTOVER initiation evidence"
  require_regular "${CUTOVER_INIT_SIG}" "CUTOVER initiation signature"
fi
for command in python3 jq gpg curl; do
  command -v "${command}" >/dev/null 2>&1 || die "${command} is required"
done

cmp -s "${SIGNER_CONFIG}" "${AUTHORITY_BUNDLE}/fly.halt-signer.toml" || \
  die "explicit halt-signer config differs from the authority bundle"
cmp -s "${OBSERVER_CONFIG}" "${AUTHORITY_BUNDLE}/fly.observer.toml" || \
  die "explicit observer config differs from the authority bundle"

if [ "${STAGE}" = signer ]; then
  VERIFY_STAGE=cutover-postinit
else
  VERIFY_STAGE=cutover-preinit
fi
VERIFY_COMMAND=(
  python3 "${VERIFIER}" "${VERIFY_STAGE}" "${AUTHORITY_BUNDLE}"
  "${EXPECTED_SIGNER}"
  --release "${RELEASE}" --release-sig "${RELEASE_SIG}"
  --decision "${CUTOVER}" --decision-sig "${CUTOVER_SIG}"
  --config-policy "${POLICY}" --tool-root "${ROOT}"
)
if [ "${STAGE}" = signer ]; then
  VERIFY_COMMAND+=(
    --initiation "${CUTOVER_INIT}" --initiation-sig "${CUTOVER_INIT_SIG}"
  )
fi
"${VERIFY_COMMAND[@]}" >/dev/null || die "CUTOVER authority chain did not verify"

F=$(jq -er '.checkpoint_plan.checkpoint_state_height' "${CUTOVER}")
A=$(jq -er '.checkpoint_plan.final_committed_anchor_height' "${CUTOVER}")
H=$(jq -er '.checkpoint_plan.halt_trigger_height' "${CUTOVER}")
MINIMUM_LEAD=$(jq -er '.authorization_semantics.minimum_halt_lead_blocks' \
  "${CUTOVER}")
for value in "${F}" "${A}" "${H}" "${MINIMUM_LEAD}"; do
  [[ "${value}" =~ ^[1-9][0-9]{0,17}$ ]] || die "signed F/A/H/lead is malformed"
done
[ "$((10#${F} + 1))" -eq "$((10#${A}))" ] && \
  [ "$((10#${A} + 1))" -eq "$((10#${H}))" ] || \
  die "CUTOVER must satisfy A=F+1 and H=A+1"

python3 "${POLICY}" "${SIGNER_CONFIG}" zerone-2-cutover-decision-v1 \
  zerone_1_halt_signer - "${F}" "${A}" "${H}" - - >/dev/null || \
  die "halt-signer config violates structural policy"
python3 "${POLICY}" "${OBSERVER_CONFIG}" zerone-2-cutover-decision-v1 \
  zerone_1_observer - "${F}" "${A}" "${H}" - - >/dev/null || \
  die "observer config violates structural policy"

"${TX_GATE}" --check "${RELEASE}" "${RELEASE_SIG}" \
  "${CUTOVER}" "${CUTOVER_SIG}" \
  "${AUTHORITY_BUNDLE}/CUTOVER-SIGNED-TX.json" cutover \
  "${AUTHORITY_BUNDLE}/zeroned-zerone-1-release" http://localhost \
  "${EXPECTED_SIGNER}" "${AUTHORITY_BUNDLE}" >/dev/null || \
  die "CUTOVER TxRaw no longer matches the release/authority"

mapping_value() {
  jq -er --arg key "$1" --arg field "$2" \
    '.deployment_configs[$key][$field]' "${CUTOVER}"
}
SIGNER_APP=$(mapping_value zerone_1_halt_signer app)
SIGNER_IMAGE=$(mapping_value zerone_1_halt_signer image_ref)
SIGNER_ROLE=$(mapping_value zerone_1_halt_signer role)
SIGNER_CONFIG_SHA=$(mapping_value zerone_1_halt_signer sha256)
OBSERVER_APP=$(mapping_value zerone_1_observer app)
OBSERVER_IMAGE=$(mapping_value zerone_1_observer image_ref)
OBSERVER_ROLE=$(mapping_value zerone_1_observer role)
OBSERVER_CONFIG_SHA=$(mapping_value zerone_1_observer sha256)

"${PINNED_GATE}" --check "${SIGNER_CONFIG}" "${SIGNER_APP}" \
  "${SIGNER_IMAGE}" "${SIGNER_ROLE}" "${SIGNER_CONFIG_SHA}" >/dev/null || \
  die "halt-signer pinned config check failed"
"${PINNED_GATE}" --check "${OBSERVER_CONFIG}" "${OBSERVER_APP}" \
  "${OBSERVER_IMAGE}" "${OBSERVER_ROLE}" "${OBSERVER_CONFIG_SHA}" >/dev/null || \
  die "observer pinned config check failed"

if [ "${MODE}" = check ]; then
  printf 'fly-cutover-authorized: %s checks MATCH\n' "${STAGE}"
  exit 0
fi

EXPECTED_SIGNER_NODE=$(jq -er '.predecessor.trusted_rpc_node_id' "${RELEASE}")
EXPECTED_OBSERVER_NODE=$(jq -er '.predecessor.trusted_observer_node_id' "${RELEASE}")
TRUSTED_HEIGHT=$(jq -er '.predecessor.trusted_block.height' "${RELEASE}")
TRUSTED_BLOCK_HASH=$(jq -er '.predecessor.trusted_block.block_id_hash' "${RELEASE}")
TRUSTED_APP_HASH=$(jq -er '.predecessor.trusted_block.app_hash' "${RELEASE}")

node_height() {
  local rpc=$1 expected_node=$2 label=$3 response
  response=$(curl -fsS --max-time 5 "${rpc%/}/status") || \
    die "${label} status query failed"
  jq -er --arg node "${expected_node}" '
    select(.result.node_info.network == "zerone-1") |
    select((.result.node_info.id | ascii_downcase) == ($node | ascii_downcase)) |
    select(.result.sync_info.catching_up == false) |
    .result.sync_info.latest_block_height | select(test("^[1-9][0-9]*$"))
  ' <<<"${response}" || die "${label} identity/chain/readiness differs from RELEASE"
}

verify_anchor() {
  local rpc=$1 label=$2 block
  block=$(curl -fsS --max-time 5 "${rpc%/}/block?height=${TRUSTED_HEIGHT}") || \
    die "${label} trusted-block query failed"
  jq -e --arg height "${TRUSTED_HEIGHT}" --arg block "${TRUSTED_BLOCK_HASH}" \
    --arg app "${TRUSTED_APP_HASH}" '
      .result.block_id.hash == $block and
      .result.block.header.chain_id == "zerone-1" and
      .result.block.header.height == $height and
      .result.block.header.app_hash == $app
    ' <<<"${block}" >/dev/null || die "${label} differs from RELEASE trusted block"
}

verify_pair() {
  local signer_height observer_height common signer_block observer_block
  signer_height=$(node_height "${SIGNER_RPC}" "${EXPECTED_SIGNER_NODE}" signer)
  observer_height=$(node_height "${OBSERVER_RPC}" "${EXPECTED_OBSERVER_NODE}" observer)
  [ "$((10#${signer_height} + MINIMUM_LEAD))" -le "$((10#${F}))" ] || \
    die "live signer height no longer preserves the signed lead before F"
  [ "$((10#${observer_height} + MINIMUM_LEAD))" -le "$((10#${F}))" ] || \
    die "live observer height no longer preserves the signed lead before F"
  verify_anchor "${SIGNER_RPC}" signer
  verify_anchor "${OBSERVER_RPC}" observer
  if [ "$((10#${signer_height}))" -le "$((10#${observer_height}))" ]; then
    common=${signer_height}
  else
    common=${observer_height}
  fi
  signer_block=$(curl -fsS --max-time 5 \
    "${SIGNER_RPC%/}/block?height=${common}") || die "signer common-block query failed"
  observer_block=$(curl -fsS --max-time 5 \
    "${OBSERVER_RPC%/}/block?height=${common}") || die "observer common-block query failed"
  jq -ecn --argjson signer "${signer_block}" --argjson observer "${observer_block}" \
    --arg height "${common}" '
      $signer.result.block.header.height == $height and
      $observer.result.block.header.height == $height and
      $signer.result.block.header.chain_id == "zerone-1" and
      $observer.result.block.header.chain_id == "zerone-1" and
      $signer.result.block_id.hash == $observer.result.block_id.hash and
      $signer.result.block.header.app_hash == $observer.result.block.header.app_hash
    ' >/dev/null || die "signer and observer disagree at their common live height"
}

case "${STAGE}" in
  observer)
    "${PINNED_GATE}" "${OBSERVER_CONFIG}" "${OBSERVER_APP}" \
      "${OBSERVER_IMAGE}" "${OBSERVER_ROLE}" "${OBSERVER_CONFIG_SHA}"
    ;;
  signer)
    verify_pair
    "${PINNED_GATE}" "${SIGNER_CONFIG}" "${SIGNER_APP}" \
      "${SIGNER_IMAGE}" "${SIGNER_ROLE}" "${SIGNER_CONFIG_SHA}"
    ;;
esac

verify_pair
printf 'fly-cutover-authorized: %s deployed with signer/observer MATCH\n' "${STAGE}"
