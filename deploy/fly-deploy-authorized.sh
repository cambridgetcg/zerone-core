#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'fly-deploy-authorized: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  deploy/fly-deploy-authorized.sh [--check] \
    RELEASE.json RELEASE.json.sig AUTHORITY.json AUTHORITY.json.sig \
    CONFIG CONFIG_KEY EXPECTED_MAIN_FINGERPRINT
  deploy/fly-deploy-authorized.sh [--check] \
    RELEASE.json RELEASE.json.sig AUTHORITY.json AUTHORITY.json.sig \
    CONFIG CONFIG_KEY EXPECTED_MAIN_FINGERPRINT \
    INITIATION-EVIDENCE.json INITIATION-EVIDENCE.json.sig
  deploy/fly-deploy-authorized.sh [--check] \
    RELEASE.json RELEASE.json.sig OPEN-BETA.json OPEN-BETA.json.sig \
    CONFIG CONFIG_KEY EXPECTED_MAIN_FINGERPRINT \
    INITIATION-EVIDENCE.json INITIATION-EVIDENCE.json.sig \
    FINAL-CHECKPOINT.json FINAL-CHECKPOINT.json.sig \
    EXPECTED_TRANSITION_FINGERPRINT OPEN_AUTHORITY_BUNDLE_DIRECTORY

DARK-START may deploy before its deadline without evidence; after block 1, its
signed initiation evidence permits the scoped private continuation. CUTOVER is
accepted only by mainnet/fly-cutover-authorized.sh so observer-first/signer-last
ordering and fresh dual-node height checks cannot be bypassed. OPEN-BETA
deployments always require their signed on-time initiation evidence.
OPEN-BETA additionally requires the transition-signed FINAL and full transition
fingerprint fixed in RELEASE.
Archive candidate/final deployments use fly-deploy-archive-authorized.sh.
USAGE
  exit 2
}

MODE=deploy
if [ "${1:-}" = "--check" ]; then
  MODE=check
  shift
fi
case "$#" in
  7)
    HAS_EVIDENCE=false
    HAS_FINAL=false
    HAS_CHAIN_BUNDLE=false
    ;;
  9)
    HAS_EVIDENCE=true
    HAS_FINAL=false
    HAS_CHAIN_BUNDLE=false
    EVIDENCE=$8
    EVIDENCE_SIGNATURE=$9
    EVIDENCE_SIGNATURE_NAME=$(basename -- "${EVIDENCE_SIGNATURE}")
    ;;
  10)
    HAS_EVIDENCE=true
    HAS_FINAL=false
    HAS_CHAIN_BUNDLE=true
    EVIDENCE=$8
    EVIDENCE_SIGNATURE=$9
    AUTHORITY_BUNDLE=${10}
    EVIDENCE_SIGNATURE_NAME=$(basename -- "${EVIDENCE_SIGNATURE}")
    ;;
  13)
    HAS_EVIDENCE=true
    HAS_FINAL=true
    HAS_CHAIN_BUNDLE=true
    EVIDENCE=$8
    EVIDENCE_SIGNATURE=$9
    FINAL_CHECKPOINT=${10}
    FINAL_SIGNATURE=${11}
    EXPECTED_TRANSITION_SIGNER=${12}
    AUTHORITY_BUNDLE=${13}
    EVIDENCE_SIGNATURE_NAME=$(basename -- "${EVIDENCE_SIGNATURE}")
    FINAL_SIGNATURE_NAME=$(basename -- "${FINAL_SIGNATURE}")
    ;;
  *)
    usage
    ;;
esac

RELEASE=$1
RELEASE_SIGNATURE=$2
AUTHORITY=$3
SIGNATURE=$4
CONFIG=$5
CONFIG_KEY=$6
EXPECTED_SIGNER=$7
RELEASE_SIGNATURE_NAME=$(basename -- "${RELEASE_SIGNATURE}")
SIGNATURE_NAME=$(basename -- "${SIGNATURE}")
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
PINNED_GATE="${ROOT}/deploy/fly-deploy-pinned.sh"
CONFIG_POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
CHAIN_VERIFIER="${ROOT}/deploy/verify-authority-chain.py"
FINAL_TEMPLATE="${ROOT}/deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json"
OPEN_TEMPLATE="${ROOT}/deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json"
ADOPTION_TEMPLATE="${ROOT}/deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json"
TX_GATE="${ROOT}/scripts/zerone-phase-tx-broadcast.sh"

require_regular() {
  local path=$1 label=$2
  [ -f "${path}" ] || die "${label} is not a regular file"
  [ ! -L "${path}" ] || die "${label} must not be a symlink"
}
require_regular "${AUTHORITY}" "authority payload"
require_regular "${SIGNATURE}" "detached signature"
require_regular "${RELEASE}" "release packet"
require_regular "${RELEASE_SIGNATURE}" "release-packet signature"
require_regular "${CONFIG}" "Fly config"
require_regular "${PINNED_GATE}" "pinned deployment gate"
require_regular "${CONFIG_POLICY}" "Fly phase config policy"
require_regular "${CHAIN_VERIFIER}" "authority-chain verifier"
require_regular "${FINAL_TEMPLATE}" "FINAL checkpoint template"
require_regular "${OPEN_TEMPLATE}" "OPEN-BETA template"
require_regular "${ADOPTION_TEMPLATE}" "archive adoption template"
require_regular "${TX_GATE}" "signed phase transaction gate"
if [ "${HAS_EVIDENCE}" = true ]; then
  require_regular "${EVIDENCE}" "initiation evidence"
  require_regular "${EVIDENCE_SIGNATURE}" "initiation evidence signature"
fi
if [ "${HAS_FINAL}" = true ]; then
  require_regular "${FINAL_CHECKPOINT}" "final checkpoint"
  require_regular "${FINAL_SIGNATURE}" "final-checkpoint signature"
  [[ "${EXPECTED_TRANSITION_SIGNER}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
    die "transition signer must be a full 40- or 64-hex fingerprint"
fi
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v gpg >/dev/null 2>&1 || die "gpg is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"
[[ "${CONFIG_KEY}" =~ ^[a-z0-9_]+$ ]] || die "config key is malformed"
[[ "${EXPECTED_SIGNER}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "expected signer must be a full 40- or 64-hex fingerprint"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-authorized-deploy.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
install -m 0600 "${AUTHORITY}" "${TMP}/authority.json"
install -m 0600 "${SIGNATURE}" "${TMP}/${SIGNATURE_NAME}"
install -m 0600 "${RELEASE}" "${TMP}/release.json"
install -m 0600 "${RELEASE_SIGNATURE}" "${TMP}/${RELEASE_SIGNATURE_NAME}"
install -m 0600 "${CONFIG}" "${TMP}/fly.toml"
install -m 0700 "${PINNED_GATE}" "${TMP}/fly-deploy-pinned.sh"
install -m 0700 "${CONFIG_POLICY}" "${TMP}/validate-fly-phase-config.py"
install -m 0700 "${CHAIN_VERIFIER}" "${TMP}/verify-authority-chain.py"
install -m 0600 "${FINAL_TEMPLATE}" "${TMP}/FINAL-CHECKPOINT.example.json"
install -m 0600 "${OPEN_TEMPLATE}" "${TMP}/OPEN-BETA-DECISION.example.json"
install -m 0600 "${ADOPTION_TEMPLATE}" \
  "${TMP}/ARCHIVE-ADOPTION-AUTHORITY.example.json"
if [ "${HAS_EVIDENCE}" = true ]; then
  install -m 0600 "${EVIDENCE}" "${TMP}/initiation-evidence.json"
  install -m 0600 "${EVIDENCE_SIGNATURE}" "${TMP}/${EVIDENCE_SIGNATURE_NAME}"
  EVIDENCE="${TMP}/initiation-evidence.json"
  EVIDENCE_SIGNATURE="${TMP}/${EVIDENCE_SIGNATURE_NAME}"
fi
if [ "${HAS_FINAL}" = true ]; then
  install -m 0600 "${FINAL_CHECKPOINT}" "${TMP}/final-checkpoint.json"
  install -m 0600 "${FINAL_SIGNATURE}" "${TMP}/${FINAL_SIGNATURE_NAME}"
  FINAL_CHECKPOINT="${TMP}/final-checkpoint.json"
  FINAL_SIGNATURE="${TMP}/${FINAL_SIGNATURE_NAME}"
fi
AUTHORITY="${TMP}/authority.json"
SIGNATURE="${TMP}/${SIGNATURE_NAME}"
RELEASE="${TMP}/release.json"
RELEASE_SIGNATURE="${TMP}/${RELEASE_SIGNATURE_NAME}"
CONFIG="${TMP}/fly.toml"
PINNED_GATE="${TMP}/fly-deploy-pinned.sh"
CONFIG_POLICY="${TMP}/validate-fly-phase-config.py"
CHAIN_VERIFIER="${TMP}/verify-authority-chain.py"
FINAL_TEMPLATE="${TMP}/FINAL-CHECKPOINT.example.json"
OPEN_TEMPLATE="${TMP}/OPEN-BETA-DECISION.example.json"
ADOPTION_TEMPLATE="${TMP}/ARCHIVE-ADOPTION-AUTHORITY.example.json"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

normalize_fingerprint() {
  printf '%s' "$1" | tr '[:lower:]' '[:upper:]'
}

canonical_utc_epoch() {
  jq -ern --arg timestamp "$1" '
    ($timestamp | fromdateiso8601) as $epoch |
    select(($epoch | todateiso8601) == $timestamp) |
    $epoch
  '
}

canonical_utc_nanoseconds() {
  python3 - "$1" <<'PY'
import datetime as dt
import re
import sys

value = sys.argv[1]
match = re.fullmatch(
    r"([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})"
    r"(?:\.([0-9]{1,9}))?Z",
    value,
)
if match is None:
    raise SystemExit(1)
try:
    whole = dt.datetime.strptime(match.group(1), "%Y-%m-%dT%H:%M:%S").replace(
        tzinfo=dt.timezone.utc
    )
except ValueError:
    raise SystemExit(1)
fraction = (match.group(2) or "").ljust(9, "0")
print(int(whole.timestamp()) * 1_000_000_000 + int(fraction or "0"))
PY
}

require_canonical_json() {
  local payload=$1 label=$2 output
  output="${TMP}/$(basename -- "${payload}").canonical"
  jq -S -c . "${payload}" > "${output}" || die "${label} is not valid JSON"
  cmp -s "${payload}" "${output}" || die "${label} is not canonical JSON"
  if grep -Eq 'REPLACE_|replace-' "${payload}"; then
    die "${label} retains a placeholder"
  fi
}

verify_signed_payload() {
  local payload=$1 signature=$2 signature_name=$3 label=$4
  local declared algorithm signer status count valid
  algorithm=$(jq -er '.signature_authority.algorithm' "${payload}") || \
    die "${label} signature algorithm is missing"
  signer=$(jq -er '.signature_authority.authorized_signer_fingerprint' \
    "${payload}") || die "${label} signer fingerprint is missing"
  declared=$(jq -er '.signature_authority.detached_signature_filename' \
    "${payload}") || die "${label} signature filename is missing"
  [ "${algorithm}" = "openpgp" ] || die "${label} is not OpenPGP"
  [ "${signature_name}" = "${declared}" ] || \
    die "${label} signature filename differs from its signed declaration"
  [ "$(normalize_fingerprint "${signer}")" = \
    "$(normalize_fingerprint "${EXPECTED_SIGNER}")" ] || \
    die "${label} repeats the wrong out-of-band signer fingerprint"
  if ! status=$(gpg --batch --status-fd=1 --verify "${signature}" \
    "${payload}" 2>/dev/null); then
    die "${label} detached signature verification failed"
  fi
  count=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { count++ } END { print count + 0 }')
  [ "${count}" -eq 1 ] || die "${label} must produce exactly one VALIDSIG"
  valid=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { print $3 }')
  [ "$(normalize_fingerprint "${valid}")" = \
    "$(normalize_fingerprint "${EXPECTED_SIGNER}")" ] || \
    die "${label} signature was made by a different key"
}

require_canonical_json "${RELEASE}" "release packet"
verify_signed_payload "${RELEASE}" "${RELEASE_SIGNATURE}" \
  "${RELEASE_SIGNATURE_NAME}" "release packet"
jq -e '.schema == "zerone-2-release-packet-v1" and .chain_id == "zerone-2"' \
  "${RELEASE}" >/dev/null || die "release root has the wrong schema or chain ID"
jq -e '
  (.deployment_configs | keys | sort) == ([
    "zerone_2_validator", "zerone_2_edge_private",
    "zerone_2_edge_query_soak", "zerone_2_gateway_private",
    "zerone_2_edge_public", "zerone_2_gateway_public",
    "zerone_1_archive_gateway"
  ] | sort) and
  (.deployment_configs.zerone_1_archive_gateway | keys | sort) == ([
    "app", "role", "image_component", "image_ref"
  ] | sort) and
  (.archive_gateway_render_contract | keys | sort) == ([
    "schema", "renderer_path", "renderer_sha256", "template_path", "bindings"
  ] | sort) and
  .archive_gateway_render_contract.schema ==
    "zerone-1-archive-gateway-render-contract-v1" and
  .archive_gateway_render_contract.renderer_path ==
    "deploy/query-gateway/render-archive-gateway-config.py" and
  .archive_gateway_render_contract.template_path ==
    "deploy/query-gateway/fly.zerone-1-archive.public.example.toml" and
  ([.deployment_configs[].app] | all(.[];
    test("^[a-z0-9][a-z0-9-]{0,62}$"))) and
  .deployment_configs.zerone_2_edge_private.app ==
    .deployment_configs.zerone_2_edge_query_soak.app and
  .deployment_configs.zerone_2_edge_private.app ==
    .deployment_configs.zerone_2_edge_public.app and
  .deployment_configs.zerone_2_gateway_private.app ==
    .deployment_configs.zerone_2_gateway_public.app and
  ([
    .deployment_configs.zerone_2_validator.app,
    .deployment_configs.zerone_2_edge_private.app,
    .deployment_configs.zerone_2_gateway_private.app,
    .deployment_configs.zerone_1_archive_gateway.app
  ] | unique | length) == 4 and
  ([.deployment_configs[].app] |
    all(.[]; . != "zerone-1" and . != "zerone-1-observer" and
      . != "zerone-1-archive")) and
  (. as $release | all(.deployment_configs | to_entries[];
    .value.image_ref == $release.components[.value.image_component].image_ref)) and
  .components.zerone_1_halt.image_ref != .components.zerone_2_runtime.image_ref and
  .components.zerone_1_halt.image_ref != .components.query_gateway.image_ref and
  .components.zerone_2_runtime.image_ref != .components.query_gateway.image_ref
' "${RELEASE}" >/dev/null || die "release deployment topology is unsafe or inconsistent"
require_canonical_json "${AUTHORITY}" "authority"
verify_signed_payload "${AUTHORITY}" "${SIGNATURE}" "${SIGNATURE_NAME}" \
  "authority"
if sed -E 's/[[:space:]]*#.*$//' "${CONFIG}" | grep -Eq 'REPLACE_|replace-'; then
  die "Fly config retains an active placeholder"
fi

AUTHORITY_SHA=$(sha256_file "${AUTHORITY}")
AUTHORITY_SIGNATURE_SHA=$(sha256_file "${SIGNATURE}")
RELEASE_SHA=$(sha256_file "${RELEASE}")
RELEASE_SIGNATURE_SHA=$(sha256_file "${RELEASE_SIGNATURE}")
SCHEMA=$(jq -er '.schema' "${AUTHORITY}") || die "authority schema is missing"
DEADLINE=$(jq -er '.authorization_semantics.initiation_deadline' \
  "${AUTHORITY}") || die "authority initiation deadline is missing"
DEADLINE_EPOCH=$(canonical_utc_epoch "${DEADLINE}") || \
  die "authority initiation deadline is not a real canonical UTC second"

EXPECTED_EVIDENCE_SCHEMA=
POLICY_UPSTREAM=-
POLICY_F=-
POLICY_A=-
POLICY_H=-
POLICY_ARCHIVE_APP_HASH=-
POLICY_ARCHIVE_BLOCK_HASH=-
case "${SCHEMA}" in
  zerone-2-dark-start-decision-v1)
    jq -e \
      --arg release "${RELEASE_SHA}" \
      --arg release_sig "${RELEASE_SIGNATURE_SHA}" '
      .decision == "GO" and
      .release_packet_sha256 == $release and
      .release_packet_detached_signature_sha256 == $release_sig and
      (.deployment_configs | keys | sort) == ([
        "zerone_2_validator", "zerone_2_edge_private",
        "zerone_2_edge_query_soak", "zerone_2_gateway_private"
      ] | sort)
    ' "${AUTHORITY}" >/dev/null || die "dark-start authority is not an exact GO"
    case "${CONFIG_KEY}" in
      zerone_2_validator|zerone_2_edge_private|zerone_2_edge_query_soak|zerone_2_gateway_private) ;;
      *) die "config key is outside dark-start authority" ;;
    esac
    EXPECTED_EVIDENCE_SCHEMA=zerone-2-dark-start-initiation-evidence-v1
    if [ "${HAS_EVIDENCE}" = false ]; then
      NOW_EPOCH=$(date -u '+%s')
      [ "${NOW_EPOCH}" -gt "${DEADLINE_EPOCH}" ] && \
        die "dark-start authority expired before this deployment"
    fi
    ;;
  zerone-2-cutover-decision-v1)
    die "CUTOVER must use mainnet/fly-cutover-authorized.sh (observer first, signer last)"
    ;;
  zerone-2-open-beta-decision-v1)
    [ "${HAS_EVIDENCE}" = true ] || \
      die "OPEN-BETA deployment requires signed initiation evidence"
    [ "${HAS_FINAL}" = true ] || \
      die "OPEN-BETA deployment requires the transition-signed final checkpoint"
    jq -e \
      --arg release "${RELEASE_SHA}" \
      --arg release_sig "${RELEASE_SIGNATURE_SHA}" '
      .decision == "GO" and
      .release_packet.sha256 == $release and
      .release_packet.detached_signature_sha256 == $release_sig and
      (.deployment_configs | keys | sort) == ([
        "zerone_2_edge_public", "zerone_2_gateway_public",
        "zerone_1_archive_gateway"
      ] | sort)
    ' "${AUTHORITY}" >/dev/null || die "open-beta authority is not an exact GO"
    case "${CONFIG_KEY}" in
      zerone_2_edge_public|zerone_2_gateway_public|zerone_1_archive_gateway) ;;
      *) die "config key is outside open-beta authority" ;;
    esac
    EXPECTED_EVIDENCE_SCHEMA=zerone-2-open-beta-initiation-evidence-v1
    ;;
  zerone-1-archive-adoption-authority-v1)
    die "archive deployment requires deterministic reproduction through fly-deploy-archive-authorized.sh"
    ;;
  *)
    die "payload schema cannot authorize a Fly deployment"
    ;;
esac

if [ "${SCHEMA}" != "zerone-2-open-beta-decision-v1" ] && \
  [ "${HAS_FINAL}" = true ]; then
  die "final-checkpoint arguments are valid only for OPEN-BETA"
fi

if [ "${HAS_EVIDENCE}" = true ]; then
  require_canonical_json "${EVIDENCE}" "initiation evidence"
  verify_signed_payload "${EVIDENCE}" "${EVIDENCE_SIGNATURE}" \
    "${EVIDENCE_SIGNATURE_NAME}" "initiation evidence"
  EVIDENCE_SCHEMA=$(jq -er '.schema' "${EVIDENCE}")
  [ "${EVIDENCE_SCHEMA}" = "${EXPECTED_EVIDENCE_SCHEMA}" ] || \
    die "wrong initiation-evidence schema for this authority"
  jq -e --arg deadline "${DEADLINE}" '
    .attestation_result == "MATCH" and
    .initiation_deadline == $deadline and
    .deadline_satisfied == true
  ' "${EVIDENCE}" >/dev/null || die "initiation evidence is not an exact MATCH"
  case "${SCHEMA}" in
    zerone-2-dark-start-decision-v1)
      jq -e \
        --arg authority "${AUTHORITY_SHA}" \
        --arg authority_sig "${AUTHORITY_SIGNATURE_SHA}" '
          .dark_start_decision.sha256 == $authority and
          .dark_start_decision.detached_signature_sha256 == $authority_sig and
          .first_committed_block.chain_id == "zerone-2" and
          .first_committed_block.height == "1" and
          (.first_committed_block.block_id_hash | test("^[0-9A-F]{64}$")) and
          (.first_committed_block.app_hash | test("^[0-9A-F]{64}$")) and
          (.first_committed_block.committed_block_time |
            test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$")) and
          (.first_committed_block.validator_evidence_sha256 |
            test("^[0-9a-f]{64}$")) and
          (.first_committed_block.independent_edge_evidence_sha256 |
            test("^[0-9a-f]{64}$"))
        ' "${EVIDENCE}" >/dev/null || die "dark-start initiation does not match"
      COMMIT_TIME=$(jq -er '.first_committed_block.committed_block_time' \
        "${EVIDENCE}")
      ;;
    zerone-2-cutover-decision-v1)
      jq -e \
        --arg authority "${AUTHORITY_SHA}" \
        --arg authority_sig "${AUTHORITY_SIGNATURE_SHA}" \
        --arg notice "$(jq -er '.public_notice_sha256' "${AUTHORITY}")" \
        --arg tx_bytes "$(jq -er \
          '.successor_commitment_transaction.signed_tx_bytes_sha256' \
          "${AUTHORITY}")" \
        --arg tx_hash "$(jq -er \
          '.successor_commitment_transaction.expected_transaction_hash' \
          "${AUTHORITY}")" '
          .cutover_decision.sha256 == $authority and
          .cutover_decision.detached_signature_sha256 == $authority_sig and
          .public_notice.sha256 == $notice and
          (.public_notice.publication_evidence_sha256 | test("^[0-9a-f]{64}$")) and
          .successor_commitment_transaction.signed_tx_bytes_sha256 == $tx_bytes and
          .successor_commitment_transaction.expected_transaction_hash == $tx_hash and
          .successor_commitment_transaction.committed_transaction_hash == $tx_hash and
          .successor_commitment_transaction.deliver_code == 0 and
          (.successor_commitment_transaction.committed_height |
            test("^[1-9][0-9]*$")) and
          (.successor_commitment_transaction.committed_block_time |
            test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$")) and
          (.successor_commitment_transaction.raw_transaction_query_evidence_sha256 |
            test("^[0-9a-f]{64}$")) and
          (.successor_commitment_transaction.independent_observer_evidence_sha256 |
            test("^[0-9a-f]{64}$"))
        ' "${EVIDENCE}" >/dev/null || die "cutover initiation does not match"
      COMMIT_TIME=$(jq -er \
        '.successor_commitment_transaction.committed_block_time' "${EVIDENCE}")
      ;;
    zerone-2-open-beta-decision-v1)
      jq -e \
        --arg authority "${AUTHORITY_SHA}" \
        --arg authority_sig "${AUTHORITY_SIGNATURE_SHA}" \
        --arg tx_bytes "$(jq -er \
          '.history_link_transaction.signed_tx_bytes_sha256' "${AUTHORITY}")" \
        --arg tx_hash "$(jq -er \
          '.history_link_transaction.expected_transaction_hash' "${AUTHORITY}")" '
          .open_beta_decision.sha256 == $authority and
          .open_beta_decision.detached_signature_sha256 == $authority_sig and
          .history_link_transaction.signed_tx_bytes_sha256 == $tx_bytes and
          .history_link_transaction.expected_transaction_hash == $tx_hash and
          .history_link_transaction.committed_transaction_hash == $tx_hash and
          .history_link_transaction.deliver_code == 0 and
          (.history_link_transaction.committed_height | test("^[1-9][0-9]*$")) and
          (.history_link_transaction.committed_block_time |
            test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$")) and
          (.history_link_transaction.raw_transaction_query_evidence_sha256 |
            test("^[0-9a-f]{64}$")) and
          (.history_link_transaction.independent_edge_evidence_sha256 |
            test("^[0-9a-f]{64}$"))
        ' "${EVIDENCE}" >/dev/null || die "open-beta initiation does not match"
      COMMIT_TIME=$(jq -er '.history_link_transaction.committed_block_time' \
        "${EVIDENCE}")
      ;;
  esac
  COMMIT_NANOSECONDS=$(canonical_utc_nanoseconds "${COMMIT_TIME}") || \
    die "initiation commit time is not canonical RFC3339 UTC with at most nanoseconds"
  DEADLINE_NANOSECONDS=$((10#${DEADLINE_EPOCH} * 1000000000))
  [ "${COMMIT_NANOSECONDS}" -gt "${DEADLINE_NANOSECONDS}" ] && \
    die "initiation evidence records a commit after the signed deadline"
  NOW_NANOSECONDS="$(date -u '+%s')000000000"
  [ "${COMMIT_NANOSECONDS}" -gt "${NOW_NANOSECONDS}" ] && \
    die "initiation evidence records a commit in the future"
fi

if [ "${SCHEMA}" = "zerone-2-open-beta-decision-v1" ]; then
  require_canonical_json "${FINAL_CHECKPOINT}" "final checkpoint"
  RELEASE_TRANSITION_SIGNER=$(jq -er \
    '.public_identities.transition_attestation.authorized_signer_fingerprint' \
    "${RELEASE}") || die "release transition fingerprint is missing"
  [ "$(normalize_fingerprint "${RELEASE_TRANSITION_SIGNER}")" = \
    "$(normalize_fingerprint "${EXPECTED_TRANSITION_SIGNER}")" ] || \
    die "final-checkpoint signer differs from the release transition key"
  [ "$(normalize_fingerprint "${EXPECTED_TRANSITION_SIGNER}")" != \
    "$(normalize_fingerprint "${EXPECTED_SIGNER}")" ] || \
    die "main and transition signer fingerprints must differ"
  FINAL_ALGORITHM=$(jq -er '.attestation.algorithm' "${FINAL_CHECKPOINT}")
  FINAL_SIGNER=$(jq -er '.attestation.authorized_signer_fingerprint' \
    "${FINAL_CHECKPOINT}")
  FINAL_DECLARED_SIGNATURE=$(jq -er '.attestation.detached_signature_filename' \
    "${FINAL_CHECKPOINT}")
  [ "${FINAL_ALGORITHM}" = "openpgp" ] || \
    die "final-checkpoint signature is not OpenPGP"
  [ "${FINAL_SIGNATURE_NAME}" = "${FINAL_DECLARED_SIGNATURE}" ] || \
    die "final-checkpoint signature filename differs from its declaration"
  [ "$(normalize_fingerprint "${FINAL_SIGNER}")" = \
    "$(normalize_fingerprint "${EXPECTED_TRANSITION_SIGNER}")" ] || \
    die "final checkpoint repeats the wrong transition fingerprint"
  if ! FINAL_STATUS=$(gpg --batch --status-fd=1 --verify "${FINAL_SIGNATURE}" \
    "${FINAL_CHECKPOINT}" 2>/dev/null); then
    die "final-checkpoint detached signature verification failed"
  fi
  FINAL_VALID_COUNT=$(printf '%s\n' "${FINAL_STATUS}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { count++ } END { print count + 0 }')
  [ "${FINAL_VALID_COUNT}" -eq 1 ] || \
    die "final-checkpoint signature must produce one VALIDSIG"
  FINAL_VALID=$(printf '%s\n' "${FINAL_STATUS}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { print $3 }')
  [ "$(normalize_fingerprint "${FINAL_VALID}")" = \
    "$(normalize_fingerprint "${EXPECTED_TRANSITION_SIGNER}")" ] || \
    die "final checkpoint was signed by a different key"
  FINAL_SHA=$(sha256_file "${FINAL_CHECKPOINT}")
  FINAL_SIGNATURE_SHA=$(sha256_file "${FINAL_SIGNATURE}")
  jq -e \
    --arg release "${RELEASE_SHA}" \
    --arg release_sig "${RELEASE_SIGNATURE_SHA}" \
    --arg final "${FINAL_SHA}" \
    --arg final_sig "${FINAL_SIGNATURE_SHA}" \
    --argjson open "$(jq -c . "${AUTHORITY}")" '
      def hash_pair:
        type == "object" and
        (keys | sort) == (["sha256", "detached_signature_sha256"] | sort) and
        (.sha256 | test("^[0-9a-f]{64}$")) and
        (.detached_signature_sha256 | test("^[0-9a-f]{64}$"));
      def readiness:
        type == "object" and
        (keys | sort) == ([
          "candidate_readiness_sha256", "final_runtime_marker_sha256",
          "private_a_a_probe_evidence_sha256"
        ] | sort) and
        (.candidate_readiness_sha256 | test("^[0-9a-f]{64}$")) and
        (.final_runtime_marker_sha256 | test("^[0-9a-f]{64}$")) and
        (.private_a_a_probe_evidence_sha256 | test("^[0-9a-f]{64}$"));
      .schema == "zerone-final-checkpoint-v3" and
      .status == "frozen" and
      .chain_id == "zerone-1" and
      ($open.dark_start_decision | hash_pair) and
      ($open.dark_start_initiation_evidence | hash_pair) and
      ($open.cutover_decision | hash_pair) and
      ($open.cutover_initiation_evidence | hash_pair) and
      ($open.archive_adoption_authority | hash_pair) and
      ($open.final_checkpoint | hash_pair) and
      ($open.archive_readiness | readiness) and
      (.authority_chain.release_packet | hash_pair) and
      (.authority_chain.dark_start_decision | hash_pair) and
      (.authority_chain.dark_start_initiation_evidence | hash_pair) and
      (.authority_chain.cutover_decision | hash_pair) and
      (.authority_chain.cutover_initiation_evidence | hash_pair) and
      (.authority_chain.archive_adoption_authority | hash_pair) and
      .authority_chain.release_packet == {
        sha256: $release, detached_signature_sha256: $release_sig
      } and
      .authority_chain.dark_start_decision == $open.dark_start_decision and
      .authority_chain.dark_start_initiation_evidence ==
        $open.dark_start_initiation_evidence and
      .authority_chain.cutover_decision == $open.cutover_decision and
      .authority_chain.cutover_initiation_evidence ==
        $open.cutover_initiation_evidence and
      .authority_chain.archive_adoption_authority ==
        $open.archive_adoption_authority and
      (.authority_chain.archive_transition_manifest_sha256 |
        test("^[0-9a-f]{64}$")) and
      $open.final_checkpoint == {
        sha256: $final, detached_signature_sha256: $final_sig
      } and
      .archive.candidate_readiness_sha256 ==
        $open.archive_readiness.candidate_readiness_sha256 and
      .archive.final_runtime_marker_sha256 ==
        $open.archive_readiness.final_runtime_marker_sha256 and
      .archive.private_a_a_probe_evidence_sha256 ==
        $open.archive_readiness.private_a_a_probe_evidence_sha256 and
      ({
        candidate_readiness_sha256: .archive.candidate_readiness_sha256,
        final_runtime_marker_sha256: .archive.final_runtime_marker_sha256,
        private_a_a_probe_evidence_sha256:
          .archive.private_a_a_probe_evidence_sha256
      } | readiness) and
      .archive.public_service_authorized == false
    ' "${FINAL_CHECKPOINT}" >/dev/null || \
    die "final checkpoint does not complete the OPEN-BETA authority chain"
fi

case "${SCHEMA}" in
  zerone-2-dark-start-decision-v1)
    [ "${HAS_CHAIN_BUNDLE}" = false ] || \
      die "DARK-START does not accept a CUTOVER/OPEN authority bundle"
    ;;
  zerone-2-cutover-decision-v1)
    [ "${HAS_CHAIN_BUNDLE}" = true ] || \
      die "CUTOVER deployment requires the complete authority bundle"
    python3 "${CHAIN_VERIFIER}" cutover-postinit "${AUTHORITY_BUNDLE}" \
      "${EXPECTED_SIGNER}" \
      --release "${RELEASE}" --release-sig "${RELEASE_SIGNATURE}" \
      --decision "${AUTHORITY}" --decision-sig "${SIGNATURE}" \
      --initiation "${EVIDENCE}" --initiation-sig "${EVIDENCE_SIGNATURE}" \
      --config-policy "${CONFIG_POLICY}" \
      --tool-root "${ROOT}" \
      >/dev/null || die "CUTOVER transitive authority chain did not verify"
    "${TX_GATE}" --check \
      "${RELEASE}" "${RELEASE_SIGNATURE}" "${AUTHORITY}" "${SIGNATURE}" \
      "${AUTHORITY_BUNDLE}/CUTOVER-SIGNED-TX.json" cutover \
      "${AUTHORITY_BUNDLE}/zeroned-zerone-1-release" http://localhost \
      "${EXPECTED_SIGNER}" "${AUTHORITY_BUNDLE}" >/dev/null || \
      die "CUTOVER signed transaction no longer matches its release/authority"
    ;;
  zerone-2-open-beta-decision-v1)
    [ "${HAS_CHAIN_BUNDLE}" = true ] || \
      die "OPEN-BETA deployment requires the complete authority bundle"
    python3 "${CHAIN_VERIFIER}" open-postinit "${AUTHORITY_BUNDLE}" \
      "${EXPECTED_SIGNER}" "${EXPECTED_TRANSITION_SIGNER}" \
      --release "${RELEASE}" --release-sig "${RELEASE_SIGNATURE}" \
      --decision "${AUTHORITY}" --decision-sig "${SIGNATURE}" \
      --initiation "${EVIDENCE}" --initiation-sig "${EVIDENCE_SIGNATURE}" \
      --final "${FINAL_CHECKPOINT}" --final-sig "${FINAL_SIGNATURE}" \
      --config-policy "${CONFIG_POLICY}" \
      --tool-root "${ROOT}" \
      --final-template "${FINAL_TEMPLATE}" --open-template "${OPEN_TEMPLATE}" \
      --adoption-template "${ADOPTION_TEMPLATE}" \
      >/dev/null || die "OPEN-BETA transitive authority chain did not verify"
    "${TX_GATE}" --check \
      "${RELEASE}" "${RELEASE_SIGNATURE}" "${AUTHORITY}" "${SIGNATURE}" \
      "${AUTHORITY_BUNDLE}/OPEN-BETA-SIGNED-TX.json" open-beta \
      "${AUTHORITY_BUNDLE}/zeroned-zerone-2-release" http://localhost \
      "${EXPECTED_SIGNER}" "${AUTHORITY_BUNDLE}" \
      "${EXPECTED_TRANSITION_SIGNER}" >/dev/null || \
      die "OPEN signed transaction no longer matches its release/authority"
    ;;
esac

ENTRY=$(jq -cer --arg key "${CONFIG_KEY}" '
  .deployment_configs[$key] |
  select(type == "object") |
  select((.app | type) == "string") |
  select((.role | type) == "string") |
  select((.image_component | type) == "string") |
  select((.image_ref | type) == "string") |
  select((.sha256 | type) == "string")
' "${AUTHORITY}") || die "signed config mapping is incomplete"
APP=$(printf '%s' "${ENTRY}" | jq -er '.app')
IMAGE=$(printf '%s' "${ENTRY}" | jq -er '.image_ref')
ROLE=$(printf '%s' "${ENTRY}" | jq -er '.role')
COMPONENT=$(printf '%s' "${ENTRY}" | jq -er '.image_component')
CONFIG_SHA=$(printf '%s' "${ENTRY}" | jq -er '.sha256')
RELEASE_IMAGE=$(jq -er --arg component "${COMPONENT}" \
  '.components[$component].image_ref' "${RELEASE}") || \
  die "signed image component is absent from the release packet"
[ "${IMAGE}" = "${RELEASE_IMAGE}" ] || \
  die "phase decision image differs from its signed release component"
case "${SCHEMA}" in
  zerone-2-dark-start-decision-v1)
    jq -e --arg key "${CONFIG_KEY}" --argjson entry "${ENTRY}" \
      '.deployment_configs[$key] == $entry' "${RELEASE}" >/dev/null || \
      die "phase config mapping differs from the immutable release packet"
    ;;
  zerone-2-open-beta-decision-v1)
    if [ "${CONFIG_KEY}" = zerone_1_archive_gateway ]; then
      jq -e --arg key "${CONFIG_KEY}" --argjson entry "${ENTRY}" '
        (.deployment_configs[$key] | keys | sort) ==
          (["app", "role", "image_component", "image_ref"] | sort) and
        ($entry | del(.sha256)) == .deployment_configs[$key]
      ' "${RELEASE}" >/dev/null || \
        die "archive gateway static mapping differs from the immutable release packet"
    else
      jq -e --arg key "${CONFIG_KEY}" --argjson entry "${ENTRY}" \
        '.deployment_configs[$key] == $entry' "${RELEASE}" >/dev/null || \
        die "phase config mapping differs from the immutable release packet"
    fi
    ;;
esac
case "${CONFIG_KEY}" in
  zerone_2_validator)
    [ "${ROLE}|${COMPONENT}" = "validator|zerone_2_runtime" ] || \
      die "validator mapping has the wrong role or image component"
    ;;
  zerone_2_edge_private|zerone_2_edge_query_soak|zerone_2_edge_public)
    [ "${ROLE}|${COMPONENT}" = "edge|zerone_2_runtime" ] || \
      die "edge mapping has the wrong role or image component"
    ;;
  zerone_2_gateway_private|zerone_2_gateway_public)
    [ "${ROLE}|${COMPONENT}" = "zerone-2-query|query_gateway" ] || \
      die "zerone-2 gateway mapping has the wrong role or image component"
    RELEASE_EDGE_APP=$(jq -er \
      '.deployment_configs.zerone_2_edge_private.app' "${RELEASE}")
    POLICY_UPSTREAM="${RELEASE_EDGE_APP}.internal"
    ;;
  zerone_1_halt_signer)
    [ "${ROLE}|${COMPONENT}" = "signer|zerone_1_halt" ] || \
      die "halt-signer mapping has the wrong role or image component"
    ;;
  zerone_1_observer)
    [ "${ROLE}|${COMPONENT}" = "observer|zerone_1_halt" ] || \
      die "observer mapping has the wrong role or image component"
    ;;
  zerone_1_archive_gateway)
    [ "${ROLE}|${COMPONENT}" = "zerone-1-archive-query|query_gateway" ] || \
      die "archive gateway mapping has the wrong role or image component"
    RELEASE_ARCHIVE_APP=$(jq -er \
      '.archive_render_contract.static_constraints.app' "${RELEASE}") || \
      die "release archive origin app is missing"
    POLICY_UPSTREAM="${RELEASE_ARCHIVE_APP}.internal"
    POLICY_A=$(jq -er '.final_application_block.height' \
      "${FINAL_CHECKPOINT}") || die "FINAL archive height A is missing"
    POLICY_ARCHIVE_APP_HASH=$(jq -er \
      '.excluded_post_anchor_state.app_hash | ascii_downcase' \
      "${FINAL_CHECKPOINT}") || die "FINAL archive app hash E is missing"
    POLICY_ARCHIVE_BLOCK_HASH=$(jq -er \
      '.final_application_block.block_id_hash | ascii_downcase' \
      "${FINAL_CHECKPOINT}") || die "FINAL archive block hash B is missing"
    ;;
esac

command -v python3 >/dev/null 2>&1 || \
  die "python3 with standard-library tomllib is required"
python3 "${CONFIG_POLICY}" "${CONFIG}" "${SCHEMA}" "${CONFIG_KEY}" \
  "${POLICY_UPSTREAM}" "${POLICY_F}" "${POLICY_A}" "${POLICY_H}" \
  "${POLICY_ARCHIVE_APP_HASH}" "${POLICY_ARCHIVE_BLOCK_HASH}" || \
  die "Fly config violates phase service/state policy"

if [ "${MODE}" = check ]; then
  "${PINNED_GATE}" --check "${CONFIG}" "${APP}" "${IMAGE}" "${ROLE}" \
    "${CONFIG_SHA}"
else
  "${PINNED_GATE}" "${CONFIG}" "${APP}" "${IMAGE}" "${ROLE}" \
    "${CONFIG_SHA}"
fi
