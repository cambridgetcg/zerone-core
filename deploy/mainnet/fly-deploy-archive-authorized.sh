#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'fly-deploy-archive-authorized: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  deploy/mainnet/fly-deploy-archive-authorized.sh [--check] \
    RELEASE.json RELEASE.json.sig CUTOVER.json CUTOVER.json.sig \
    CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
    ARCHIVE-TRANSITION.json ARCHIVE-ADOPTION-AUTHORITY.json \
    ARCHIVE-ADOPTION-AUTHORITY.json.sig CONFIG_KEY \
    EXPECTED_MAIN_FINGERPRINT EXPECTED_TRANSITION_FINGERPRINT \
    AUTHORITY_BUNDLE_DIRECTORY

CONFIG_KEY is zerone_1_archive_candidate or zerone_1_archive. The wrapper
reruns the deterministic renderer, byte-compares its authority with the signed
adoption payload, and deploys the reproduced config only.
USAGE
  exit 2
}

MODE=deploy
if [ "${1:-}" = "--check" ]; then
  MODE=check
  shift
fi
[ "$#" -eq 13 ] || usage

RELEASE=$1
RELEASE_SIG=$2
CUTOVER=$3
CUTOVER_SIG=$4
CUTOVER_INITIATION=$5
CUTOVER_INITIATION_SIG=$6
TRANSITION=$7
ADOPTION=$8
ADOPTION_SIG=$9
CONFIG_KEY=${10}
MAIN_FINGERPRINT=${11}
TRANSITION_FINGERPRINT=${12}
AUTHORITY_BUNDLE=${13}
ADOPTION_SIG_NAME=$(basename -- "${ADOPTION_SIG}")

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)
RENDERER="${SCRIPT_DIR}/render-archive-configs.sh"
PINNED_GATE="${ROOT}/deploy/fly-deploy-pinned.sh"

require_regular() {
  local path=$1 label=$2
  [ -f "${path}" ] || die "${label} is not a regular file"
  [ ! -L "${path}" ] || die "${label} must not be a symlink"
}
for pair in \
  "${RELEASE}|release packet" "${RELEASE_SIG}|release signature" \
  "${CUTOVER}|cutover decision" "${CUTOVER_SIG}|cutover signature" \
  "${CUTOVER_INITIATION}|cutover-initiation evidence" \
  "${CUTOVER_INITIATION_SIG}|cutover-initiation signature" \
  "${TRANSITION}|archive transition manifest" \
  "${ADOPTION}|archive adoption authority" \
  "${ADOPTION_SIG}|archive adoption signature" \
  "${RENDERER}|deterministic renderer" "${PINNED_GATE}|pinned deploy gate"; do
  require_regular "${pair%%|*}" "${pair##*|}"
done
case "${CONFIG_KEY}" in
  zerone_1_archive_candidate|zerone_1_archive) ;;
  *) die "config key is outside deterministic archive authority" ;;
esac
[[ "${MAIN_FINGERPRINT}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "main signer fingerprint must be full-length hexadecimal"
[[ "${TRANSITION_FINGERPRINT}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "transition signer fingerprint must be full-length hexadecimal"
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v gpg >/dev/null 2>&1 || die "gpg is required"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-archive-authorized-deploy.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
install -m 0600 "${ADOPTION}" "${TMP}/signed-adoption.json"
install -m 0600 "${ADOPTION_SIG}" "${TMP}/signed-adoption.json.sig"
install -m 0700 "${PINNED_GATE}" "${TMP}/fly-deploy-pinned.sh"
ADOPTION="${TMP}/signed-adoption.json"
ADOPTION_SIG="${TMP}/signed-adoption.json.sig"
PINNED_GATE="${TMP}/fly-deploy-pinned.sh"

PATH_TO_OUTPUT="${TMP}/reproduced"
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" \
    "${CUTOVER}" "${CUTOVER_SIG}" \
    "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${TRANSITION}" "${PATH_TO_OUTPUT}" "${AUTHORITY_BUNDLE}" >/dev/null
REPRODUCED_ADOPTION="${PATH_TO_OUTPUT}/ARCHIVE-ADOPTION-AUTHORITY.json"
cmp -s "${ADOPTION}" "${REPRODUCED_ADOPTION}" || \
  die "signed adoption authority differs from deterministic renderer output"

normalize_fingerprint() {
  printf '%s' "$1" | tr '[:lower:]' '[:upper:]'
}
DECLARED_FINGERPRINT=$(jq -er \
  '.signature_authority.authorized_signer_fingerprint' "${ADOPTION}")
DECLARED_ALGORITHM=$(jq -er '.signature_authority.algorithm' "${ADOPTION}")
DECLARED_SIG_NAME=$(jq -er \
  '.signature_authority.detached_signature_filename' "${ADOPTION}")
[ "${DECLARED_ALGORITHM}" = "openpgp" ] || die "adoption signature is not OpenPGP"
[ "${ADOPTION_SIG_NAME}" = "${DECLARED_SIG_NAME}" ] || \
  die "adoption signature filename differs from its signed declaration"
[ "$(normalize_fingerprint "${DECLARED_FINGERPRINT}")" = \
  "$(normalize_fingerprint "${TRANSITION_FINGERPRINT}")" ] || \
  die "adoption authority repeats the wrong transition fingerprint"
[ "$(normalize_fingerprint "${MAIN_FINGERPRINT}")" != \
  "$(normalize_fingerprint "${TRANSITION_FINGERPRINT}")" ] || \
  die "main and transition signer fingerprints must differ"
if ! STATUS=$(gpg --batch --status-fd=1 --verify "${ADOPTION_SIG}" \
  "${ADOPTION}" 2>/dev/null); then
  die "archive adoption detached signature verification failed"
fi
COUNT=$(printf '%s\n' "${STATUS}" | awk \
  '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { count++ } END { print count + 0 }')
[ "${COUNT}" -eq 1 ] || die "adoption signature must produce one VALIDSIG"
VALID=$(printf '%s\n' "${STATUS}" | awk \
  '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { print $3 }')
[ "$(normalize_fingerprint "${VALID}")" = \
  "$(normalize_fingerprint "${TRANSITION_FINGERPRINT}")" ] || \
  die "archive adoption signature was made by a different key"

case "${CONFIG_KEY}" in
  zerone_1_archive_candidate)
    CONFIG="${PATH_TO_OUTPUT}/fly.archive-candidate.toml"
    EXPECTED_ROLE=archive-candidate
    EXPECTED_TEMPLATE=$(jq -er \
      '.render_contract.archive_candidate_template_sha256' "${ADOPTION}")
    ;;
  zerone_1_archive)
    CONFIG="${PATH_TO_OUTPUT}/fly.archive.toml"
    EXPECTED_ROLE=archive
    EXPECTED_TEMPLATE=$(jq -er '.render_contract.archive_template_sha256' \
      "${ADOPTION}")
    ;;
esac
ENTRY=$(jq -cer --arg key "${CONFIG_KEY}" '.deployment_configs[$key]' \
  "${ADOPTION}")
jq -e \
  --arg role "${EXPECTED_ROLE}" --arg template "${EXPECTED_TEMPLATE}" '
    .app == "zerone-1-archive" and
    .region == "lhr" and
    .deployment_strategy == "immediate" and
    .role == $role and
    .image_component == "zerone_1_halt" and
    .volume == "zerone_archive_data" and
    .template_sha256 == $template
  ' <<<"${ENTRY}" >/dev/null || die "reproduced archive mapping changed constraints"
APP=$(jq -er '.app' <<<"${ENTRY}")
IMAGE=$(jq -er '.image_ref' <<<"${ENTRY}")
ROLE=$(jq -er '.role' <<<"${ENTRY}")
CONFIG_SHA=$(jq -er '.sha256' <<<"${ENTRY}")

if [ "${MODE}" = check ]; then
  "${PINNED_GATE}" --check "${CONFIG}" "${APP}" "${IMAGE}" "${ROLE}" \
    "${CONFIG_SHA}"
else
  "${PINNED_GATE}" "${CONFIG}" "${APP}" "${IMAGE}" "${ROLE}" \
    "${CONFIG_SHA}"
fi
