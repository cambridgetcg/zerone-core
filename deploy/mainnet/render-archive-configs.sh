#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'render-archive-configs: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  deploy/mainnet/render-archive-configs.sh \
    RELEASE-PACKET.json RELEASE-PACKET.json.sig \
    CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
    CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
    zerone-1-archive-transition.json OUTPUT_DIRECTORY \
    AUTHORITY_BUNDLE_DIRECTORY

The output directory must not exist. The renderer verifies the signed-phase
hash bindings, then emits the only two archive Fly configs authorized by the
release templates plus a canonical ARCHIVE-ADOPTION-AUTHORITY.json for the
narrow transition-attestation key to sign.
USAGE
  exit 2
}

[ "$#" -eq 9 ] || usage
RELEASE_PACKET=$1
RELEASE_SIGNATURE=$2
CUTOVER_DECISION=$3
CUTOVER_SIGNATURE=$4
CUTOVER_INITIATION_EVIDENCE=$5
CUTOVER_INITIATION_SIGNATURE=$6
TRANSITION_MANIFEST=$7
OUTPUT_DIR=$8
AUTHORITY_BUNDLE=$9
RELEASE_SIGNATURE_NAME=$(basename -- "${RELEASE_SIGNATURE}")
CUTOVER_SIGNATURE_NAME=$(basename -- "${CUTOVER_SIGNATURE}")
CUTOVER_INITIATION_SIGNATURE_NAME=$(basename -- \
  "${CUTOVER_INITIATION_SIGNATURE}")

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd -- "${SCRIPT_DIR}/../.." && pwd)
RENDERER="${SCRIPT_DIR}/render-archive-configs.sh"
CANDIDATE_TEMPLATE="${SCRIPT_DIR}/fly.archive-candidate.example.toml"
ARCHIVE_TEMPLATE="${SCRIPT_DIR}/fly.archive.example.toml"
CANONICAL_HELPER="${ROOT}/scripts/zerone-canonical-json.sh"
CHAIN_VERIFIER="${ROOT}/deploy/verify-authority-chain.py"
CONFIG_POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
TX_GATE="${ROOT}/scripts/zerone-phase-tx-broadcast.sh"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

require_regular_input() {
  local path=$1 label=$2
  [ -f "${path}" ] || die "${label} is not a regular file: ${path}"
  [ ! -L "${path}" ] || die "${label} must not be a symlink: ${path}"
}

require_regular_input "${RELEASE_PACKET}" "release packet"
require_regular_input "${RELEASE_SIGNATURE}" "release-packet signature"
require_regular_input "${CUTOVER_DECISION}" "cutover decision"
require_regular_input "${CUTOVER_SIGNATURE}" "cutover-decision signature"
require_regular_input "${CUTOVER_INITIATION_EVIDENCE}" \
  "cutover-initiation evidence"
require_regular_input "${CUTOVER_INITIATION_SIGNATURE}" \
  "cutover-initiation evidence signature"
require_regular_input "${TRANSITION_MANIFEST}" "archive transition manifest"
require_regular_input "${CANDIDATE_TEMPLATE}" "archive-candidate template"
require_regular_input "${ARCHIVE_TEMPLATE}" "archive template"
require_regular_input "${RENDERER}" "archive renderer"
require_regular_input "${CANONICAL_HELPER}" "canonical JSON helper"
require_regular_input "${CHAIN_VERIFIER}" "authority-chain verifier"
require_regular_input "${CONFIG_POLICY}" "Fly phase config policy"
require_regular_input "${TX_GATE}" "signed phase transaction gate"
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v gpg >/dev/null 2>&1 || die "gpg is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"
: "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT:?set the out-of-band authorized release signer fingerprint}"
[[ "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "out-of-band release signer fingerprint must be 40 or 64 hexadecimal characters"

[ ! -e "${OUTPUT_DIR}" ] && [ ! -L "${OUTPUT_DIR}" ] || \
  die "output directory already exists"
mkdir -m 0700 -- "${OUTPUT_DIR}" || die "could not create output directory"
COMPLETE=false
cleanup() {
  if [ "${COMPLETE}" != true ]; then
    rm -rf -- "${OUTPUT_DIR}"
  fi
}
trap cleanup EXIT HUP INT TERM

# Freeze every input before hashing or parsing it. All later reads, rendering,
# and signature checks use these private snapshots, never the caller paths.
SNAPSHOT_DIR="${OUTPUT_DIR}/.inputs"
mkdir -m 0700 -- "${SNAPSHOT_DIR}"
install -m 0600 "${RELEASE_PACKET}" "${SNAPSHOT_DIR}/RELEASE-PACKET.json"
install -m 0600 "${RELEASE_SIGNATURE}" \
  "${SNAPSHOT_DIR}/RELEASE-PACKET.json.sig"
install -m 0600 "${CUTOVER_DECISION}" "${SNAPSHOT_DIR}/CUTOVER-DECISION.json"
install -m 0600 "${CUTOVER_SIGNATURE}" \
  "${SNAPSHOT_DIR}/CUTOVER-DECISION.json.sig"
install -m 0600 "${CUTOVER_INITIATION_EVIDENCE}" \
  "${SNAPSHOT_DIR}/CUTOVER-INITIATION-EVIDENCE.json"
install -m 0600 "${CUTOVER_INITIATION_SIGNATURE}" \
  "${SNAPSHOT_DIR}/CUTOVER-INITIATION-EVIDENCE.json.sig"
install -m 0600 "${TRANSITION_MANIFEST}" \
  "${SNAPSHOT_DIR}/zerone-1-archive-transition.json"
install -m 0600 "${CANDIDATE_TEMPLATE}" \
  "${SNAPSHOT_DIR}/fly.archive-candidate.example.toml"
install -m 0600 "${ARCHIVE_TEMPLATE}" \
  "${SNAPSHOT_DIR}/fly.archive.example.toml"
install -m 0700 "${RENDERER}" "${SNAPSHOT_DIR}/render-archive-configs.sh"
install -m 0700 "${CANONICAL_HELPER}" \
  "${SNAPSHOT_DIR}/zerone-canonical-json.sh"
install -m 0700 "${CHAIN_VERIFIER}" \
  "${SNAPSHOT_DIR}/verify-authority-chain.py"
install -m 0700 "${CONFIG_POLICY}" \
  "${SNAPSHOT_DIR}/validate-fly-phase-config.py"

RELEASE_PACKET="${SNAPSHOT_DIR}/RELEASE-PACKET.json"
RELEASE_SIGNATURE="${SNAPSHOT_DIR}/RELEASE-PACKET.json.sig"
CUTOVER_DECISION="${SNAPSHOT_DIR}/CUTOVER-DECISION.json"
CUTOVER_SIGNATURE="${SNAPSHOT_DIR}/CUTOVER-DECISION.json.sig"
CUTOVER_INITIATION_EVIDENCE="${SNAPSHOT_DIR}/CUTOVER-INITIATION-EVIDENCE.json"
CUTOVER_INITIATION_SIGNATURE="${SNAPSHOT_DIR}/CUTOVER-INITIATION-EVIDENCE.json.sig"
TRANSITION_MANIFEST="${SNAPSHOT_DIR}/zerone-1-archive-transition.json"
CANDIDATE_TEMPLATE="${SNAPSHOT_DIR}/fly.archive-candidate.example.toml"
ARCHIVE_TEMPLATE="${SNAPSHOT_DIR}/fly.archive.example.toml"
RENDERER="${SNAPSHOT_DIR}/render-archive-configs.sh"
CANONICAL_HELPER="${SNAPSHOT_DIR}/zerone-canonical-json.sh"
CHAIN_VERIFIER="${SNAPSHOT_DIR}/verify-authority-chain.py"
CONFIG_POLICY="${SNAPSHOT_DIR}/validate-fly-phase-config.py"

RELEASE_SHA256=$(sha256_file "${RELEASE_PACKET}")
RELEASE_SIGNATURE_SHA256=$(sha256_file "${RELEASE_SIGNATURE}")
CUTOVER_SHA256=$(sha256_file "${CUTOVER_DECISION}")
CUTOVER_SIGNATURE_SHA256=$(sha256_file "${CUTOVER_SIGNATURE}")
CUTOVER_INITIATION_EVIDENCE_SHA256=$(sha256_file \
  "${CUTOVER_INITIATION_EVIDENCE}")
CUTOVER_INITIATION_SIGNATURE_SHA256=$(sha256_file \
  "${CUTOVER_INITIATION_SIGNATURE}")
TRANSITION_SHA256=$(sha256_file "${TRANSITION_MANIFEST}")
RENDERER_SHA256=$(sha256_file "${RENDERER}")
CANDIDATE_TEMPLATE_SHA256=$(sha256_file "${CANDIDATE_TEMPLATE}")
ARCHIVE_TEMPLATE_SHA256=$(sha256_file "${ARCHIVE_TEMPLATE}")

for payload in "${RELEASE_PACKET}" "${CUTOVER_DECISION}" \
  "${CUTOVER_INITIATION_EVIDENCE}"; do
  canonical="${SNAPSHOT_DIR}/$(basename -- "${payload}").canonical"
  jq -S -c . "${payload}" > "${canonical}" || \
    die "signed phase input is not valid JSON"
  cmp -s "${payload}" "${canonical}" || \
    die "signed phase input is not canonical JSON: $(basename -- "${payload}")"
  if grep -Eq 'REPLACE_|replace-' "${payload}"; then
    die "signed phase input retains a placeholder: $(basename -- "${payload}")"
  fi
done

jq -e '
  .schema == "zerone-2-release-packet-v2" and
  .chain_id == "zerone-2" and
  .signature_authority.algorithm == "openpgp" and
  .public_identities.transition_attestation.algorithm == "openpgp" and
  (.public_identities.transition_attestation.authorized_signer_fingerprint |
    test("^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$")) and
  (.components.zerone_1_halt.image_ref | type == "string") and
  (.predecessor.genesis_file_sha256 | test("^[0-9a-f]{64}$"))
' "${RELEASE_PACKET}" >/dev/null || die "release packet schema or archive inputs are invalid"
[ "${RELEASE_SIGNATURE_NAME}" = \
  "$(jq -er '.signature_authority.detached_signature_filename' "${RELEASE_PACKET}")" ] || \
  die "release-packet signature filename differs from its signed declaration"

jq -e \
  --arg release_sha "${RELEASE_SHA256}" \
  --arg renderer_sha "${RENDERER_SHA256}" \
  --arg candidate_template_sha "${CANDIDATE_TEMPLATE_SHA256}" \
  --arg archive_template_sha "${ARCHIVE_TEMPLATE_SHA256}" \
  --arg release_signature_sha "${RELEASE_SIGNATURE_SHA256}" '
    .schema == "zerone-2-cutover-decision-v1" and
    .decision == "GO" and
    .signature_authority.algorithm == "openpgp" and
    .release_packet_sha256 == $release_sha and
    .release_packet_detached_signature_sha256 == $release_signature_sha and
    .deterministic_private_continuation.allowed_adoption_authority_schema ==
      "zerone-1-archive-adoption-authority-v1" and
    .deterministic_private_continuation.allowed_transition_manifest_schema ==
      "zerone-1-archive-transition-v1" and
    .deterministic_private_continuation.attestation_algorithm == "openpgp" and
    .deterministic_private_continuation.required_attestation_result == "MATCH" and
    .deterministic_private_continuation.render_contract.renderer_sha256 ==
      $renderer_sha and
    .deterministic_private_continuation.render_contract.archive_candidate_template_sha256 ==
      $candidate_template_sha and
    .deterministic_private_continuation.render_contract.archive_template_sha256 ==
      $archive_template_sha
  ' "${CUTOVER_DECISION}" >/dev/null || \
  die "cutover decision does not authorize this exact deterministic renderer"
[ "${CUTOVER_SIGNATURE_NAME}" = \
  "$(jq -er '.signature_authority.detached_signature_filename' "${CUTOVER_DECISION}")" ] || \
  die "cutover-decision signature filename differs from its signed declaration"

TRANSITION_FINGERPRINT=$(jq -er \
  '.public_identities.transition_attestation.authorized_signer_fingerprint' \
  "${RELEASE_PACKET}")
CUTOVER_TRANSITION_FINGERPRINT=$(jq -er \
  '.deterministic_private_continuation.authorized_transition_signer_fingerprint' \
  "${CUTOVER_DECISION}")
[ "${TRANSITION_FINGERPRINT}" = "${CUTOVER_TRANSITION_FINGERPRINT}" ] || \
  die "transition signer fingerprint differs between release and cutover authority"
MAIN_FINGERPRINT=$(jq -er '.signature_authority.authorized_signer_fingerprint' \
  "${RELEASE_PACKET}")
normalize_fingerprint() {
  printf '%s' "$1" | tr '[:lower:]' '[:upper:]'
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
    parsed = dt.datetime.strptime(match.group(1), "%Y-%m-%dT%H:%M:%S").replace(
        tzinfo=dt.timezone.utc
    )
except ValueError:
    raise SystemExit(1)
if parsed.strftime("%Y-%m-%dT%H:%M:%S") != match.group(1):
    raise SystemExit(1)
raw_fraction = match.group(2) or ""
if raw_fraction.endswith("0"):
    raise SystemExit(1)
fraction = raw_fraction.ljust(9, "0")
print(int(parsed.timestamp()) * 1_000_000_000 + int(fraction or "0"))
PY
}
[ "${MAIN_FINGERPRINT}" = "$(jq -er \
  '.signature_authority.authorized_signer_fingerprint' "${CUTOVER_DECISION}")" ] || \
  die "main signer fingerprint differs between release and cutover authority"
[ "$(normalize_fingerprint "${MAIN_FINGERPRINT}")" != \
  "$(normalize_fingerprint "${TRANSITION_FINGERPRINT}")" ] || \
  die "transition signer must differ from the operator/release signer"

verify_detached_signature() {
  local payload=$1 signature=$2 label=$3 expected=$4 status count actual
  if ! status=$(gpg --batch --status-fd=1 --verify "${signature}" \
    "${payload}" 2>/dev/null); then
    die "${label} detached signature verification failed"
  fi
  count=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { count++ } END { print count + 0 }')
  [ "${count}" -eq 1 ] || die "${label} must have exactly one VALIDSIG record"
  actual=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { print $3 }')
  [ "$(normalize_fingerprint "${actual}")" = \
    "$(normalize_fingerprint "${expected}")" ] || \
    die "${label} signer differs from the out-of-band authorized fingerprint"
}

[ "$(normalize_fingerprint "${MAIN_FINGERPRINT}")" = \
  "$(normalize_fingerprint "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}")" ] || \
  die "release packet repeats the wrong out-of-band signer fingerprint"
verify_detached_signature "${RELEASE_PACKET}" "${RELEASE_SIGNATURE}" \
  "release packet" "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}"
verify_detached_signature "${CUTOVER_DECISION}" "${CUTOVER_SIGNATURE}" \
  "cutover decision" "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}"
verify_detached_signature "${CUTOVER_INITIATION_EVIDENCE}" \
  "${CUTOVER_INITIATION_SIGNATURE}" "cutover initiation evidence" \
  "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}"

python3 "${CHAIN_VERIFIER}" cutover-postinit "${AUTHORITY_BUNDLE}" \
  "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}" \
  --release "${RELEASE_PACKET}" --release-sig "${RELEASE_SIGNATURE}" \
  --decision "${CUTOVER_DECISION}" --decision-sig "${CUTOVER_SIGNATURE}" \
  --initiation "${CUTOVER_INITIATION_EVIDENCE}" \
  --initiation-sig "${CUTOVER_INITIATION_SIGNATURE}" \
  --config-policy "${CONFIG_POLICY}" --tool-root "${ROOT}" >/dev/null || \
  die "CUTOVER transitive authority chain did not verify"
"${TX_GATE}" --offline-artifact-check \
  "${RELEASE_PACKET}" "${RELEASE_SIGNATURE}" \
  "${CUTOVER_DECISION}" "${CUTOVER_SIGNATURE}" \
  "${AUTHORITY_BUNDLE}/CUTOVER-SIGNED-TX.json" cutover \
  "${AUTHORITY_BUNDLE}/zeroned-zerone-1-release" http://localhost \
  "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}" \
  "${AUTHORITY_BUNDLE}" >/dev/null || \
  die "CUTOVER signed transaction differs from the archived initiation event"

[ "${CUTOVER_INITIATION_SIGNATURE_NAME}" = "$(jq -er \
  '.signature_authority.detached_signature_filename' \
  "${CUTOVER_INITIATION_EVIDENCE}")" ] || \
  die "cutover-initiation signature filename differs from its declaration"
[ "$(normalize_fingerprint "$(jq -er \
  '.signature_authority.authorized_signer_fingerprint' \
  "${CUTOVER_INITIATION_EVIDENCE}")")" = \
  "$(normalize_fingerprint "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT}")" ] || \
  die "cutover-initiation evidence repeats the wrong main fingerprint"

RELEASE_RENDERER_SHA=$(jq -er '.archive_render_contract.renderer_sha256' \
  "${RELEASE_PACKET}")
RELEASE_CANDIDATE_TEMPLATE_SHA=$(jq -er \
  '.phase_dependent_config_template_sha256.zerone_1_archive_candidate' \
  "${RELEASE_PACKET}")
RELEASE_ARCHIVE_TEMPLATE_SHA=$(jq -er \
  '.phase_dependent_config_template_sha256.zerone_1_archive' \
  "${RELEASE_PACKET}")
[ "${RELEASE_RENDERER_SHA}" = "${RENDERER_SHA256}" ] || \
  die "renderer differs from the release-packet hash"
[ "${RELEASE_CANDIDATE_TEMPLATE_SHA}" = "${CANDIDATE_TEMPLATE_SHA256}" ] || \
  die "archive-candidate template differs from the release-packet hash"
[ "${RELEASE_ARCHIVE_TEMPLATE_SHA}" = "${ARCHIVE_TEMPLATE_SHA256}" ] || \
  die "archive template differs from the release-packet hash"
jq -e '
  .archive_render_contract.schema == "zerone-1-archive-render-contract-v1" and
  .archive_render_contract.renderer_path ==
    "deploy/mainnet/render-archive-configs.sh" and
  .archive_render_contract.static_constraints == {
    app: "zerone-1-archive",
    volume: "zerone_archive_data",
    region: "lhr",
    image_component: "zerone_1_halt",
    archive_candidate_role: "archive-candidate",
    archive_role: "archive",
    deployment_strategy: "immediate",
    public_fly_service: false,
    public_ip: false,
    persistent_peers: []
  }
' "${RELEASE_PACKET}" >/dev/null || die "release archive constraints changed"
jq -e '
  .deterministic_private_continuation.render_contract.schema ==
    "zerone-1-archive-render-contract-v1" and
  .deterministic_private_continuation.render_contract.renderer_path ==
    "deploy/mainnet/render-archive-configs.sh" and
  .deterministic_private_continuation.render_contract.static_constraints == {
    app: "zerone-1-archive",
    volume: "zerone_archive_data",
    region: "lhr",
    image_component: "zerone_1_halt",
    archive_candidate_role: "archive-candidate",
    archive_role: "archive",
    deployment_strategy: "immediate",
    public_fly_service: false,
    public_ip: false,
    persistent_peers: []
  }
' "${CUTOVER_DECISION}" >/dev/null || die "cutover archive constraints changed"

F=$(jq -er '.checkpoint_plan.checkpoint_state_height' "${CUTOVER_DECISION}")
A=$(jq -er '.checkpoint_plan.final_committed_anchor_height' "${CUTOVER_DECISION}")
H=$(jq -er '.checkpoint_plan.halt_trigger_height' "${CUTOVER_DECISION}")
for value in "${F}" "${A}" "${H}"; do
  [[ "${value}" =~ ^[1-9][0-9]{0,17}$ ]] || \
    die "F/A/H must be positive decimal strings within the renderer range"
done
[ "$((10#${F} + 1))" -eq "$((10#${A}))" ] && \
  [ "$((10#${A} + 1))" -eq "$((10#${H}))" ] || \
  die "cutover decision must satisfy A=F+1 and H=A+1"

PREDECESSOR_GENESIS_SHA=$(jq -er '.predecessor.genesis_file_sha256' \
  "${RELEASE_PACKET}")
jq -e \
  --arg f "${F}" --arg a "${A}" --arg h "${H}" \
  --arg genesis "${PREDECESSOR_GENESIS_SHA}" '
    type == "object" and
    (keys | sort) == ([
      "schema", "chain_id", "checkpoint_state_height",
      "final_committed_height", "halt_trigger_height", "genesis_sha256",
      "cutover_initiation_evidence",
      "source_observer", "candidate", "expected_anchor_block_hash",
      "expected_post_anchor_app_hash", "source_evidence",
      "archive_construction_evidence", "archive_transition_nonce"
    ] | sort) and
    .schema == "zerone-1-archive-transition-v1" and
    .chain_id == "zerone-1" and
    .checkpoint_state_height == $f and
    .final_committed_height == $a and
    .halt_trigger_height == $h and
    .genesis_sha256 == $genesis and
    (.cutover_initiation_evidence | keys | sort) == ([
      "successor_transaction_hash", "committed_height",
      "committed_block_time", "public_notice_sha256",
      "public_notice_publication_evidence_sha256",
      "initiation_evidence_sha256",
      "initiation_evidence_detached_signature_sha256"
    ] | sort) and
    (.cutover_initiation_evidence.successor_transaction_hash |
      test("^[0-9A-F]{64}$")) and
    (.cutover_initiation_evidence.committed_height |
      test("^[1-9][0-9]*$")) and
    (.cutover_initiation_evidence.committed_block_time |
      test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$")) and
    (.cutover_initiation_evidence.public_notice_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.cutover_initiation_evidence.public_notice_publication_evidence_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.cutover_initiation_evidence.initiation_evidence_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.cutover_initiation_evidence.initiation_evidence_detached_signature_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.source_observer | keys | sort) ==
      (["runtime_marker_sha256", "node_id", "validator_pubkey"] | sort) and
    (.candidate | keys | sort) == (["node_id", "validator_pubkey"] | sort) and
    (.source_evidence | keys | sort) ==
      (["signer_manifest_sha256", "observer_manifest_sha256"] | sort) and
    (.source_observer.runtime_marker_sha256 | test("^[0-9a-f]{64}$")) and
    (.source_observer.node_id | test("^[0-9a-f]{40}$")) and
    (.source_observer.validator_pubkey | test("^[A-Za-z0-9+/]{43}=$")) and
    (.candidate.node_id | test("^[0-9a-f]{40}$")) and
    (.candidate.validator_pubkey | test("^[A-Za-z0-9+/]{43}=$")) and
    .candidate.node_id != .source_observer.node_id and
    .candidate.validator_pubkey != .source_observer.validator_pubkey and
    (.expected_anchor_block_hash | test("^[0-9A-F]{64}$")) and
    (.expected_post_anchor_app_hash | test("^[0-9A-F]{64}$")) and
    (.source_evidence.signer_manifest_sha256 | test("^[0-9a-f]{64}$")) and
    (.source_evidence.observer_manifest_sha256 | test("^[0-9a-f]{64}$")) and
    (.archive_construction_evidence | keys | sort) == ([
      "pre_transition_sanitized_snapshot_sha256", "rollback_log_sha256",
      "pre_transition_allowlist_manifest_sha256", "excluded_future_artifacts"
    ] | sort) and
    (.archive_construction_evidence.pre_transition_sanitized_snapshot_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.archive_construction_evidence.rollback_log_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.archive_construction_evidence.pre_transition_allowlist_manifest_sha256 |
      test("^[0-9a-f]{64}$")) and
    .archive_construction_evidence.excluded_future_artifacts == [
      "archive transition manifest", "rendered Fly configs",
      "archive adoption authority", "archive readiness",
      "final checkpoint", "open-beta decision"
    ] and
    (.archive_transition_nonce | test("^[0-9a-f]{64}$"))
  ' "${TRANSITION_MANIFEST}" >/dev/null || \
  die "archive transition manifest is malformed or differs from the cutover plan"

IMAGE_REF=$(jq -er '.components.zerone_1_halt.image_ref' "${RELEASE_PACKET}")
[[ "${IMAGE_REF}" =~ ^[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:[0-9]{1,5})?/[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*@sha256:[0-9a-f]{64}$ ]] || \
  die "zerone-1 halt image must be one full lowercase immutable reference"

SIGNER_EVIDENCE=$(jq -er '.source_evidence.signer_manifest_sha256' \
  "${TRANSITION_MANIFEST}")
OBSERVER_EVIDENCE=$(jq -er '.source_evidence.observer_manifest_sha256' \
  "${TRANSITION_MANIFEST}")
SOURCE_MARKER=$(jq -er '.source_observer.runtime_marker_sha256' \
  "${TRANSITION_MANIFEST}")
SOURCE_NODE=$(jq -er '.source_observer.node_id' "${TRANSITION_MANIFEST}")
SOURCE_VALIDATOR=$(jq -er '.source_observer.validator_pubkey' \
  "${TRANSITION_MANIFEST}")
CANDIDATE_NODE=$(jq -er '.candidate.node_id' "${TRANSITION_MANIFEST}")
CANDIDATE_VALIDATOR=$(jq -er '.candidate.validator_pubkey' \
  "${TRANSITION_MANIFEST}")
ANCHOR_HASH=$(jq -er '.expected_anchor_block_hash' "${TRANSITION_MANIFEST}")
APP_HASH=$(jq -er '.expected_post_anchor_app_hash' "${TRANSITION_MANIFEST}")
SANITIZED_SNAPSHOT_SHA=$(jq -er \
  '.archive_construction_evidence.pre_transition_sanitized_snapshot_sha256' \
  "${TRANSITION_MANIFEST}")
ROLLBACK_LOG_SHA=$(jq -er '.archive_construction_evidence.rollback_log_sha256' \
  "${TRANSITION_MANIFEST}")
ALLOWLIST_MANIFEST_SHA=$(jq -er \
  '.archive_construction_evidence.pre_transition_allowlist_manifest_sha256' \
  "${TRANSITION_MANIFEST}")
TRANSITION_NONCE=$(jq -er '.archive_transition_nonce' "${TRANSITION_MANIFEST}")
CUTOVER_TX_HASH=$(jq -er \
  '.cutover_initiation_evidence.successor_transaction_hash' \
  "${TRANSITION_MANIFEST}")
CUTOVER_TX_HEIGHT=$(jq -er '.cutover_initiation_evidence.committed_height' \
  "${TRANSITION_MANIFEST}")
CUTOVER_TX_TIME=$(jq -er '.cutover_initiation_evidence.committed_block_time' \
  "${TRANSITION_MANIFEST}")
NOTICE_SHA=$(jq -er '.cutover_initiation_evidence.public_notice_sha256' \
  "${TRANSITION_MANIFEST}")
NOTICE_PUBLICATION_EVIDENCE_SHA=$(jq -er \
  '.cutover_initiation_evidence.public_notice_publication_evidence_sha256' \
  "${TRANSITION_MANIFEST}")
CUTOVER_INITIATION_EVIDENCE_SHA=$(jq -er \
  '.cutover_initiation_evidence.initiation_evidence_sha256' \
  "${TRANSITION_MANIFEST}")
CUTOVER_INITIATION_EVIDENCE_SIGNATURE_SHA=$(jq -er \
  '.cutover_initiation_evidence.initiation_evidence_detached_signature_sha256' \
  "${TRANSITION_MANIFEST}")
CUTOVER_NOTICE_SHA=$(jq -er '.public_notice_sha256' "${CUTOVER_DECISION}")
CUTOVER_DEADLINE=$(jq -er '.authorization_semantics.initiation_deadline' \
  "${CUTOVER_DECISION}")
CUTOVER_SIGNED_TX_SHA=$(jq -er \
  '.successor_commitment_transaction.signed_tx_bytes_sha256' \
  "${CUTOVER_DECISION}")
CUTOVER_EXPECTED_TX_HASH=$(jq -er \
  '.successor_commitment_transaction.expected_transaction_hash' \
  "${CUTOVER_DECISION}")
CUTOVER_DEADLINE_NS=$(canonical_utc_nanoseconds "${CUTOVER_DEADLINE}") || \
  die "cutover initiation deadline is not canonical UTC RFC3339Nano"
[ "${NOTICE_SHA}" = "${CUTOVER_NOTICE_SHA}" ] || \
  die "published notice hash differs from the CUTOVER authority"
CUTOVER_TX_NS=$(canonical_utc_nanoseconds "${CUTOVER_TX_TIME}") || \
  die "successor commit time is not canonical UTC RFC3339Nano"
[ "${CUTOVER_TX_NS}" -gt "${CUTOVER_DEADLINE_NS}" ] && \
  die "successor transaction committed after the CUTOVER initiation deadline"

jq -e \
  --arg fingerprint "${MAIN_FINGERPRINT}" \
  --arg cutover_sha "${CUTOVER_SHA256}" \
  --arg cutover_signature_sha "${CUTOVER_SIGNATURE_SHA256}" \
  --arg deadline "${CUTOVER_DEADLINE}" \
  --arg notice_sha "${CUTOVER_NOTICE_SHA}" \
  --arg signed_tx_sha "${CUTOVER_SIGNED_TX_SHA}" \
  --arg expected_tx_hash "${CUTOVER_EXPECTED_TX_HASH}" '
    .schema == "zerone-2-cutover-initiation-evidence-v1" and
    .attestation_result == "MATCH" and
    .signature_authority.algorithm == "openpgp" and
    .signature_authority.authorized_signer_fingerprint == $fingerprint and
    .cutover_decision.sha256 == $cutover_sha and
    .cutover_decision.detached_signature_sha256 == $cutover_signature_sha and
    .initiation_deadline == $deadline and
    .public_notice.sha256 == $notice_sha and
    (.public_notice.publication_evidence_sha256 | test("^[0-9a-f]{64}$")) and
    .successor_commitment_transaction.signed_tx_bytes_sha256 == $signed_tx_sha and
    .successor_commitment_transaction.expected_transaction_hash ==
      $expected_tx_hash and
    .successor_commitment_transaction.committed_transaction_hash ==
      $expected_tx_hash and
    .successor_commitment_transaction.deliver_code == 0 and
    (.successor_commitment_transaction.committed_height |
      test("^[1-9][0-9]*$")) and
    (.successor_commitment_transaction.committed_block_time |
      test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?Z$")) and
    (.successor_commitment_transaction.raw_transaction_query_evidence_sha256 |
      test("^[0-9a-f]{64}$")) and
    (.successor_commitment_transaction.independent_observer_evidence_sha256 |
      test("^[0-9a-f]{64}$")) and
    .deadline_satisfied == true
  ' "${CUTOVER_INITIATION_EVIDENCE}" >/dev/null || \
  die "cutover-initiation evidence does not match the signed CUTOVER event"

EVIDENCE_TX_HASH=$(jq -er \
  '.successor_commitment_transaction.committed_transaction_hash' \
  "${CUTOVER_INITIATION_EVIDENCE}")
EVIDENCE_TX_HEIGHT=$(jq -er \
  '.successor_commitment_transaction.committed_height' \
  "${CUTOVER_INITIATION_EVIDENCE}")
EVIDENCE_TX_TIME=$(jq -er \
  '.successor_commitment_transaction.committed_block_time' \
  "${CUTOVER_INITIATION_EVIDENCE}")
EVIDENCE_NOTICE_PUBLICATION_SHA=$(jq -er \
  '.public_notice.publication_evidence_sha256' \
  "${CUTOVER_INITIATION_EVIDENCE}")
EVIDENCE_TX_NS=$(canonical_utc_nanoseconds "${EVIDENCE_TX_TIME}") || \
  die "cutover-initiation evidence time is not canonical UTC RFC3339Nano"
[ "${EVIDENCE_TX_NS}" -gt "${CUTOVER_DEADLINE_NS}" ] && \
  die "signed cutover-initiation evidence records a late commit"
[ "${CUTOVER_INITIATION_EVIDENCE_SHA}" = \
  "${CUTOVER_INITIATION_EVIDENCE_SHA256}" ] && \
  [ "${CUTOVER_INITIATION_EVIDENCE_SIGNATURE_SHA}" = \
  "${CUTOVER_INITIATION_SIGNATURE_SHA256}" ] && \
  [ "${CUTOVER_TX_HASH}" = "${EVIDENCE_TX_HASH}" ] && \
  [ "${CUTOVER_TX_HEIGHT}" = "${EVIDENCE_TX_HEIGHT}" ] && \
  [ "${CUTOVER_TX_TIME}" = "${EVIDENCE_TX_TIME}" ] && \
  [ "${NOTICE_PUBLICATION_EVIDENCE_SHA}" = \
  "${EVIDENCE_NOTICE_PUBLICATION_SHA}" ] || \
  die "inner transition manifest differs from cutover-initiation evidence"

render_template() {
  local source=$1 destination=$2
  LC_ALL=C sed \
    -e "s|REPLACE_WITH_PINNED_ZERONE_1_HALT_IMAGE_DIGEST|${IMAGE_REF}|g" \
    -e "s|REPLACE_WITH_F|${F}|g" \
    -e "s|REPLACE_WITH_A|${A}|g" \
    -e "s|REPLACE_WITH_H|${H}|g" \
    -e "s|REPLACE_WITH_SIGNER_EVIDENCE_SHA256|${SIGNER_EVIDENCE}|g" \
    -e "s|REPLACE_WITH_OBSERVER_EVIDENCE_SHA256|${OBSERVER_EVIDENCE}|g" \
    -e "s|REPLACE_WITH_SOURCE_OBSERVER_MARKER_SHA256|${SOURCE_MARKER}|g" \
    -e "s|REPLACE_WITH_SOURCE_OBSERVER_NODE_ID|${SOURCE_NODE}|g" \
    -e "s|REPLACE_WITH_SOURCE_OBSERVER_VALIDATOR_PUBKEY_BASE64|${SOURCE_VALIDATOR}|g" \
    -e "s|REPLACE_WITH_UPPERCASE_A_BLOCK_HASH|${ANCHOR_HASH}|g" \
    -e "s|REPLACE_WITH_UPPERCASE_POST_A_APP_HASH|${APP_HASH}|g" \
    -e "s|REPLACE_WITH_ARCHIVE_TRANSITION_MANIFEST_SHA256|${TRANSITION_SHA256}|g" \
    "${source}" > "${destination}"
  chmod 0600 "${destination}"
  if grep -Eq 'REPLACE_|replace-' "${destination}"; then
    die "rendered config retains a placeholder: ${destination}"
  fi
}

CANDIDATE_CONFIG="${OUTPUT_DIR}/fly.archive-candidate.toml"
ARCHIVE_CONFIG="${OUTPUT_DIR}/fly.archive.toml"
render_template "${CANDIDATE_TEMPLATE}" "${CANDIDATE_CONFIG}"
render_template "${ARCHIVE_TEMPLATE}" "${ARCHIVE_CONFIG}"

if grep -Eq '^\[\[services\]\]$|^\[http_service\]$' \
  "${CANDIDATE_CONFIG}" "${ARCHIVE_CONFIG}"; then
  die "archive origin templates unexpectedly expose a Fly service"
fi
grep -q '^  NODE_ROLE = "archive-candidate"$' "${CANDIDATE_CONFIG}" || \
  die "candidate config role changed"
grep -q '^  NODE_ROLE = "archive"$' "${ARCHIVE_CONFIG}" || \
  die "archive config role changed"
grep -q '^  source = "zerone_archive_data"$' "${CANDIDATE_CONFIG}" || \
  die "candidate config volume changed"
grep -q '^  source = "zerone_archive_data"$' "${ARCHIVE_CONFIG}" || \
  die "archive config volume changed"

CANDIDATE_CONFIG_SHA256=$(sha256_file "${CANDIDATE_CONFIG}")
ARCHIVE_CONFIG_SHA256=$(sha256_file "${ARCHIVE_CONFIG}")
ADOPTION_DRAFT="${OUTPUT_DIR}/.ARCHIVE-ADOPTION-AUTHORITY.draft.json"
ADOPTION_JSON="${OUTPUT_DIR}/ARCHIVE-ADOPTION-AUTHORITY.json"

jq -n \
  --arg fingerprint "${TRANSITION_FINGERPRINT}" \
  --arg release_sha "${RELEASE_SHA256}" \
  --arg release_signature_sha "${RELEASE_SIGNATURE_SHA256}" \
  --arg cutover_sha "${CUTOVER_SHA256}" \
  --arg cutover_signature_sha "${CUTOVER_SIGNATURE_SHA256}" \
  --arg transition_sha "${TRANSITION_SHA256}" \
  --arg renderer_sha "${RENDERER_SHA256}" \
  --arg candidate_template_sha "${CANDIDATE_TEMPLATE_SHA256}" \
  --arg archive_template_sha "${ARCHIVE_TEMPLATE_SHA256}" \
  --arg f "${F}" --arg a "${A}" --arg h "${H}" \
  --arg signer_evidence "${SIGNER_EVIDENCE}" \
  --arg observer_evidence "${OBSERVER_EVIDENCE}" \
  --arg source_marker "${SOURCE_MARKER}" \
  --arg source_node "${SOURCE_NODE}" \
  --arg source_validator "${SOURCE_VALIDATOR}" \
  --arg candidate_node "${CANDIDATE_NODE}" \
  --arg candidate_validator "${CANDIDATE_VALIDATOR}" \
  --arg anchor_hash "${ANCHOR_HASH}" \
  --arg app_hash "${APP_HASH}" \
  --arg sanitized_snapshot_sha "${SANITIZED_SNAPSHOT_SHA}" \
  --arg rollback_log_sha "${ROLLBACK_LOG_SHA}" \
  --arg allowlist_manifest_sha "${ALLOWLIST_MANIFEST_SHA}" \
  --arg transition_nonce "${TRANSITION_NONCE}" \
  --arg cutover_tx_hash "${CUTOVER_TX_HASH}" \
  --arg cutover_tx_height "${CUTOVER_TX_HEIGHT}" \
  --arg cutover_tx_time "${CUTOVER_TX_TIME}" \
  --arg notice_sha "${NOTICE_SHA}" \
  --arg notice_publication_evidence_sha "${NOTICE_PUBLICATION_EVIDENCE_SHA}" \
  --arg cutover_initiation_evidence_sha "${CUTOVER_INITIATION_EVIDENCE_SHA}" \
  --arg cutover_initiation_evidence_signature_sha \
    "${CUTOVER_INITIATION_EVIDENCE_SIGNATURE_SHA}" \
  --arg image_ref "${IMAGE_REF}" \
  --arg candidate_config_sha "${CANDIDATE_CONFIG_SHA256}" \
  --arg archive_config_sha "${ARCHIVE_CONFIG_SHA256}" '
  {
    schema: "zerone-1-archive-adoption-authority-v1",
    attestation_result: "MATCH",
    signature_authority: {
      algorithm: "openpgp",
      authorized_signer_fingerprint: $fingerprint,
      detached_signature_filename: "ARCHIVE-ADOPTION-AUTHORITY.json.sig"
    },
    release_packet: {
      sha256: $release_sha,
      detached_signature_sha256: $release_signature_sha
    },
    cutover_decision: {
      sha256: $cutover_sha,
      detached_signature_sha256: $cutover_signature_sha
    },
    checkpoint_plan: {
      checkpoint_state_height: $f,
      final_committed_anchor_height: $a,
      halt_trigger_height: $h
    },
    cutover_initiation_evidence: {
      successor_transaction_hash: $cutover_tx_hash,
      committed_height: $cutover_tx_height,
      committed_block_time: $cutover_tx_time,
      public_notice_sha256: $notice_sha,
      public_notice_publication_evidence_sha256:
        $notice_publication_evidence_sha,
      initiation_evidence_sha256: $cutover_initiation_evidence_sha,
      initiation_evidence_detached_signature_sha256:
        $cutover_initiation_evidence_signature_sha
    },
    archive_transition_manifest: {
      schema: "zerone-1-archive-transition-v1",
      sha256: $transition_sha,
      archive_transition_nonce: $transition_nonce,
      source_evidence: {
        signer_manifest_sha256: $signer_evidence,
        observer_manifest_sha256: $observer_evidence
      },
      source_observer: {
        runtime_marker_sha256: $source_marker,
        node_id: $source_node,
        validator_pubkey: $source_validator
      },
      candidate: {
        node_id: $candidate_node,
        validator_pubkey: $candidate_validator
      },
      expected_anchor_block_hash: $anchor_hash,
      expected_post_anchor_app_hash: $app_hash
    },
    archive_construction_evidence: {
      pre_transition_sanitized_snapshot_sha256: $sanitized_snapshot_sha,
      rollback_log_sha256: $rollback_log_sha,
      pre_transition_allowlist_manifest_sha256: $allowlist_manifest_sha,
      excluded_future_artifacts: [
        "archive transition manifest", "rendered Fly configs",
        "archive adoption authority", "archive readiness",
        "final checkpoint", "open-beta decision"
      ]
    },
    render_contract: {
      schema: "zerone-1-archive-render-contract-v1",
      renderer_path: "deploy/mainnet/render-archive-configs.sh",
      renderer_sha256: $renderer_sha,
      archive_candidate_template_path:
        "deploy/mainnet/fly.archive-candidate.example.toml",
      archive_candidate_template_sha256: $candidate_template_sha,
      archive_template_path: "deploy/mainnet/fly.archive.example.toml",
      archive_template_sha256: $archive_template_sha,
      allowed_substitutions: [
        "zerone_1_halt_image_ref", "F", "A", "H",
        "signer_evidence_manifest_sha256", "observer_evidence_manifest_sha256",
        "source_observer_runtime_marker_sha256", "source_observer_node_id",
        "source_observer_validator_pubkey", "expected_anchor_block_hash",
        "expected_post_anchor_app_hash", "archive_transition_manifest_sha256"
      ]
    },
    deployment_configs: {
      zerone_1_archive_candidate: {
        app: "zerone-1-archive",
        region: "lhr",
        deployment_strategy: "immediate",
        role: "archive-candidate",
        image_component: "zerone_1_halt",
        image_ref: $image_ref,
        volume: "zerone_archive_data",
        template_sha256: $candidate_template_sha,
        sha256: $candidate_config_sha
      },
      zerone_1_archive: {
        app: "zerone-1-archive",
        region: "lhr",
        deployment_strategy: "immediate",
        role: "archive",
        image_component: "zerone_1_halt",
        image_ref: $image_ref,
        volume: "zerone_archive_data",
        template_sha256: $archive_template_sha,
        sha256: $archive_config_sha
      }
    },
    enforced_origin_invariants: {
      public_fly_service: false,
      public_ip: false,
      persistent_peers: [],
      transaction_ingress: false,
      same_app_and_volume_one_way_adoption: true
    },
    authority_limit:
      "deterministically rendered private archive candidate/final adoption only; no operator GO, public service, history-link transaction, endpoint publication, or DNS authority"
  }
' > "${ADOPTION_DRAFT}"
chmod 0600 "${ADOPTION_DRAFT}"
"${CANONICAL_HELPER}" "${ADOPTION_DRAFT}" "${ADOPTION_JSON}"
rm -f -- "${ADOPTION_DRAFT}"
rm -rf -- "${SNAPSHOT_DIR}"

COMPLETE=true
trap - EXIT HUP INT TERM
printf 'archive-candidate-config-sha256=%s\n' "${CANDIDATE_CONFIG_SHA256}"
printf 'archive-config-sha256=%s\n' "${ARCHIVE_CONFIG_SHA256}"
printf 'archive-adoption-authority-sha256=%s\n' "$(sha256_file "${ADOPTION_JSON}")"
