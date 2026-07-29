#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
VERIFY=${VERIFY:-"${ROOT}/deploy/verify-authority-chain.py"}
FIXTURE="${ROOT}/deploy/test-fixtures/make-authority-bundle.py"
POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
FINAL_TEMPLATE="${ROOT}/deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json"
OPEN_TEMPLATE="${ROOT}/deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json"
ADOPTION_TEMPLATE="${ROOT}/deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-authority-chain-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

canonical_mutate() {
  local path=$1 filter=$2
  shift 2
  jq -S -c "$@" "${filter}" "${path}" > "${path}.new"
  mv "${path}.new" "${path}"
}

clone_bundle() {
  local name=$1
  local destination="${TMP}/${name}"
  cp -R "${BASE_BUNDLE}" "${destination}"
  printf '%s\n' "${destination}"
}

rebind_frozen_source_chain() {
  local bundle=$1
  local signer_manifest_sha observer_manifest_sha transition_sha adoption_sha
  signer_manifest_sha=$(sha256_file \
    "${bundle}/SIGNER-EVIDENCE-MANIFEST.json")
  observer_manifest_sha=$(sha256_file \
    "${bundle}/OBSERVER-EVIDENCE-MANIFEST.json")
  # shellcheck disable=SC2016 # jq variables are not shell variables.
  canonical_mutate "${bundle}/zerone-1-archive-transition.json" \
    '.source_evidence = {
      signer_manifest_sha256: $signer,
      observer_manifest_sha256: $observer
    }' \
    --arg signer "${signer_manifest_sha}" \
    --arg observer "${observer_manifest_sha}"
  transition_sha=$(sha256_file \
    "${bundle}/zerone-1-archive-transition.json")
  # shellcheck disable=SC2016 # jq variables are not shell variables.
  canonical_mutate "${bundle}/ARCHIVE-ADOPTION-AUTHORITY.json" \
    '.archive_transition_manifest.sha256 = $transition
    | .archive_transition_manifest.source_evidence = {
        signer_manifest_sha256: $signer,
        observer_manifest_sha256: $observer
      }' \
    --arg transition "${transition_sha}" \
    --arg signer "${signer_manifest_sha}" \
    --arg observer "${observer_manifest_sha}"
  adoption_sha=$(sha256_file \
    "${bundle}/ARCHIVE-ADOPTION-AUTHORITY.json")
  # shellcheck disable=SC2016 # jq variables are not shell variables.
  canonical_mutate "${bundle}/FINAL-CHECKPOINT.json" \
    '.authority_chain.archive_transition_manifest_sha256 = $transition
    | .authority_chain.archive_adoption_authority.sha256 = $adoption
    | .terminal_rpc_evidence.sources.official_signer.sha256_manifest_sha256
        = $signer
    | .terminal_rpc_evidence.sources.independent_observer.sha256_manifest_sha256
        = $observer' \
    --arg transition "${transition_sha}" \
    --arg adoption "${adoption_sha}" \
    --arg signer "${signer_manifest_sha}" \
    --arg observer "${observer_manifest_sha}"
}

expect_rejected() {
  local label=$1 expected=$2
  shift 2
  local output
  if output=$("$@" 2>&1); then
    printf 'expected rejection: %s\n%s\n' "${label}" "${output}" >&2
    exit 1
  fi
  if ! grep -Fqi -- "${expected}" <<<"${output}"; then
    printf 'wrong rejection for %s; expected %q in:\n%s\n' \
      "${label}" "${expected}" "${output}" >&2
    exit 1
  fi
}

MAIN_FINGERPRINT=$(printf 'a%.0s' {1..40})
TRANSITION_FINGERPRINT=$(printf 'b%.0s' {1..40})
RUNTIME_IMAGE="registry.example/zerone-2-runtime@sha256:$(printf '1%.0s' {1..64})"
HALT_IMAGE="registry.example/zerone-1-halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query-gateway@sha256:$(printf '3%.0s' {1..64})"
SENDER=zerone1authorityfixture

cat > "${TMP}/zeroned" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case ${1:-} in
  verify-frozen-terminal)
    shift
    [ "$#" -eq 46 ] || exit 65
    flag_value() {
      local wanted=$1
      shift
      while [ "$#" -gt 1 ]; do
        if [ "$1" = "${wanted}" ]; then
          printf '%s' "$2"
          return 0
        fi
        shift 2
      done
      return 1
    }
    for flag in \
      --genesis --trusted-block --trusted-commit --trusted-validators \
      --a-block --a-commit --a-validators --a-block-results \
      --h-block --h-commit --h-validators; do
      [ -f "$(flag_value "${flag}" "$@")" ] || exit 65
    done
    [ "$(flag_value --expected-chain-id "$@")" = zerone-1 ] || exit 65
    [ "$(flag_value --trusted-height "$@")" = 42 ] || exit 65
    [ "$(flag_value --checkpoint-state-height "$@")" = 1000 ] || exit 65
    [ "$(flag_value --final-committed-height "$@")" = 1001 ] || exit 65
    [ "$(flag_value --halt-trigger-height "$@")" = 1002 ] || exit 65
    [ "$(flag_value --expected-trusted-block-hash "$@")" = \
      "$(printf 'C%.0s' {1..64})" ] || exit 65
    [ "$(flag_value --expected-trusted-app-hash "$@")" = \
      "$(printf 'D%.0s' {1..64})" ] || exit 65
    [ "$(flag_value --expected-checkpoint-app-hash "$@")" = \
      "$(printf 'B%.0s' {1..64})" ] || exit 65
    [ "$(flag_value --expected-anchor-block-hash "$@")" = \
      "$(printf 'A%.0s' {1..64})" ] || exit 65
    [ "$(flag_value --expected-halt-trigger-block-hash "$@")" = \
      "$(printf 'D%.0s' {1..64})" ] || exit 65
    [ "$(flag_value --expected-post-anchor-app-hash "$@")" = \
      "$(printf 'E%.0s' {1..64})" ] || exit 65
    [[ "$(flag_value --expected-rpc-genesis-sha256 "$@")" =~ \
      ^[0-9a-f]{64}$ ]] || exit 65
    printf 'frozen-terminal-crypto: MATCH\n'
    ;;
  *)
    printf 'unsupported fixture zeroned command: %s\n' "${1:-}" >&2
    exit 64
    ;;
esac
EOF
chmod +x "${TMP}/zeroned"
BINARY_SHA=$(sha256_file "${TMP}/zeroned")
printf 'fixture phase TxRaw bytes' > "${TMP}/txraw"
TX_RAW_SHA=$(sha256_file "${TMP}/txraw")
TX_HASH=$(printf '%s' "${TX_RAW_SHA}" | tr '[:lower:]' '[:upper:]')
TX_BASE64=$(base64 < "${TMP}/txraw" | tr -d '\r\n')
jq -n -S -c --arg encoded "${TX_BASE64}" '{encoded:$encoded}' \
  > "${TMP}/signed-tx.json"

# This test double checks the expected fixture signature filename/content and
# reports a stage-specific historical timestamp. Semantic mutation cases can
# therefore exercise the verifier beyond signature parsing, while a changed
# signature still fails exactly as a real bad detached signature would.
cat > "${TMP}/bin/gpg" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
signature=${@: -2:1}
base=${signature##*/}
expected="fixture signature ${base}"
[ "$(cat "${signature}")" = "${expected}" ] || exit 1

fingerprint=${FAKE_GPG_MAIN_FINGERPRINT:?}
timestamp=1783677660
case "${base}" in
  RELEASE-PACKET.json.sig) timestamp=1783677660 ;;
  DARK-START-DECISION.json.sig) timestamp=1783678260 ;;
  DARK-START-INITIATION-EVIDENCE.json.sig) timestamp=1783678920 ;;
  DARK-REGISTRATION-EVIDENCE.json.sig) timestamp=1783680000 ;;
  CUTOVER-DECISION.json.sig) timestamp=1783688460 ;;
  CUTOVER-INITIATION-EVIDENCE.json.sig) timestamp=1783692120 ;;
  ARCHIVE-ADOPTION-AUTHORITY.json.sig)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783693800
    ;;
  FINAL-CHECKPOINT.json.sig)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783695660
    ;;
  OPEN-BETA-DECISION.json.sig) timestamp=1783699260 ;;
  OPEN-BETA-INITIATION-EVIDENCE.json.sig) timestamp=1783699920 ;;
  *) exit 2 ;;
esac
if [ "${FAKE_GPG_FUTURE_SIGNATURE:-}" = "${base}" ]; then
  timestamp=4102444800
fi
printf '[GNUPG:] VALIDSIG %s 2026-07-10 %s 0 4 0 1 10 00 %s\n' \
  "${fingerprint}" "${timestamp}" "${fingerprint}"
EOF
chmod +x "${TMP}/bin/gpg"

BASE_BUNDLE="${TMP}/authority-bundle"
"${FIXTURE}" \
  --output "${BASE_BUNDLE}" \
  --main "${MAIN_FINGERPRINT}" \
  --transition "${TRANSITION_FINGERPRINT}" \
  --runtime-binary-sha "${BINARY_SHA}" \
  --halt-binary-sha "${BINARY_SHA}" \
  --runtime-image "${RUNTIME_IMAGE}" \
  --halt-image "${HALT_IMAGE}" \
  --query-image "${QUERY_IMAGE}" \
  --genesis-sha "$(printf '4%.0s' {1..64})" \
  --tx-raw-sha "${TX_RAW_SHA}" \
  --tx-hash "${TX_HASH}" \
  --sender "${SENDER}" \
  --release-binary-file "${TMP}/zeroned" \
  --signed-tx-file "${TMP}/signed-tx.json"

run_cutover_pre() {
  local bundle=$1 tool_root=${2:-${ROOT}}
  PATH="${TMP}/bin:${PATH}" \
    FAKE_GPG_MAIN_FINGERPRINT="${MAIN_FINGERPRINT}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
    FAKE_GPG_FUTURE_SIGNATURE="${FAKE_GPG_FUTURE_SIGNATURE:-}" \
    "${VERIFY}" cutover-preinit "${bundle}" "${MAIN_FINGERPRINT}" \
      --release "${bundle}/RELEASE-PACKET.json" \
      --release-sig "${bundle}/RELEASE-PACKET.json.sig" \
      --decision "${bundle}/CUTOVER-DECISION.json" \
      --decision-sig "${bundle}/CUTOVER-DECISION.json.sig" \
      --config-policy "${POLICY}" --tool-root "${tool_root}"
}

run_dark_registration() {
  local bundle=$1
  PATH="${TMP}/bin:${PATH}" \
    FAKE_GPG_MAIN_FINGERPRINT="${MAIN_FINGERPRINT}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
    FAKE_GPG_FUTURE_SIGNATURE="${FAKE_GPG_FUTURE_SIGNATURE:-}" \
    "${VERIFY}" dark-registration-preinit "${bundle}" "${MAIN_FINGERPRINT}" \
      --release "${bundle}/RELEASE-PACKET.json" \
      --release-sig "${bundle}/RELEASE-PACKET.json.sig" \
      --decision "${bundle}/DARK-START-DECISION.json" \
      --decision-sig "${bundle}/DARK-START-DECISION.json.sig" \
      --initiation "${bundle}/DARK-START-INITIATION-EVIDENCE.json" \
      --initiation-sig \
        "${bundle}/DARK-START-INITIATION-EVIDENCE.json.sig" \
      --config-policy "${POLICY}" --tool-root "${ROOT}"
}

run_cutover_post() {
  local bundle=$1
  PATH="${TMP}/bin:${PATH}" \
    FAKE_GPG_MAIN_FINGERPRINT="${MAIN_FINGERPRINT}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
    FAKE_GPG_FUTURE_SIGNATURE="${FAKE_GPG_FUTURE_SIGNATURE:-}" \
    "${VERIFY}" cutover-postinit "${bundle}" "${MAIN_FINGERPRINT}" \
      --release "${bundle}/RELEASE-PACKET.json" \
      --release-sig "${bundle}/RELEASE-PACKET.json.sig" \
      --decision "${bundle}/CUTOVER-DECISION.json" \
      --decision-sig "${bundle}/CUTOVER-DECISION.json.sig" \
      --initiation "${bundle}/CUTOVER-INITIATION-EVIDENCE.json" \
      --initiation-sig "${bundle}/CUTOVER-INITIATION-EVIDENCE.json.sig" \
      --config-policy "${POLICY}" --tool-root "${ROOT}"
}

run_open() {
  local stage=$1 bundle=$2 tool_root=${3:-${ROOT}}
  local args=(
    "${VERIFY}" "${stage}" "${bundle}"
    "${MAIN_FINGERPRINT}" "${TRANSITION_FINGERPRINT}"
    --release "${bundle}/RELEASE-PACKET.json"
    --release-sig "${bundle}/RELEASE-PACKET.json.sig"
    --decision "${bundle}/OPEN-BETA-DECISION.json"
    --decision-sig "${bundle}/OPEN-BETA-DECISION.json.sig"
    --final "${bundle}/FINAL-CHECKPOINT.json"
    --final-sig "${bundle}/FINAL-CHECKPOINT.json.sig"
    --final-template "${FINAL_TEMPLATE}"
    --open-template "${OPEN_TEMPLATE}"
    --adoption-template "${ADOPTION_TEMPLATE}"
    --config-policy "${POLICY}"
    --tool-root "${tool_root}"
  )
  if [ "${stage}" = open-postinit ]; then
    args+=(
      --initiation "${bundle}/OPEN-BETA-INITIATION-EVIDENCE.json"
      --initiation-sig "${bundle}/OPEN-BETA-INITIATION-EVIDENCE.json.sig"
    )
  fi
  PATH="${TMP}/bin:${PATH}" \
    FAKE_GPG_MAIN_FINGERPRINT="${MAIN_FINGERPRINT}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
    FAKE_GPG_FUTURE_SIGNATURE="${FAKE_GPG_FUTURE_SIGNATURE:-}" \
    "${args[@]}"
}

run_open_pre() {
  run_open open-preinit "$1"
}

run_open_post() {
  run_open open-postinit "$1"
}

run_dark_registration "${BASE_BUNDLE}" >/dev/null
run_cutover_pre "${BASE_BUNDLE}" >/dev/null
run_cutover_post "${BASE_BUNDLE}" >/dev/null
run_open_pre "${BASE_BUNDLE}" >/dev/null
run_open_post "${BASE_BUNDLE}" >/dev/null

missing=$(clone_bundle missing-authority)
mv "${missing}/DARK-START-DECISION.json" \
  "${missing}/DARK-START-DECISION.json.missing"
expect_rejected "missing predecessor authority" \
  "could not open bundle file DARK-START-DECISION.json" \
  run_cutover_pre "${missing}"

bad_signature=$(clone_bundle bad-signature)
printf 'tampered\n' >> "${bad_signature}/CUTOVER-DECISION.json.sig"
expect_rejected "bad detached signature" "detached signature verification failed" \
  run_cutover_pre "${bad_signature}"

unauthenticated_helper=$(clone_bundle unauthenticated-helper)
unauthenticated_tool_root="${TMP}/unauthenticated-tool-root"
while IFS= read -r relative; do
  mkdir -p "${unauthenticated_tool_root}/$(dirname -- "${relative}")"
  cp "${ROOT}/${relative}" "${unauthenticated_tool_root}/${relative}"
done < <(jq -r '.files | keys[]' \
  "${unauthenticated_helper}/OPERATOR-TOOL-MANIFEST.json")
printf '%s\n' \
  'import os' \
  'from pathlib import Path' \
  'Path(os.environ["FROZEN_HELPER_EXECUTION_MARKER"]).write_text("executed")' \
  >> "${unauthenticated_tool_root}/deploy/frozen_evidence.py"
unauthenticated_helper_sha=$(sha256_file \
  "${unauthenticated_tool_root}/deploy/frozen_evidence.py")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${unauthenticated_helper}/OPERATOR-TOOL-MANIFEST.json" \
  '.files["deploy/frozen_evidence.py"] = $sha' \
  --arg sha "${unauthenticated_helper_sha}"
unauthenticated_manifest_sha=$(sha256_file \
  "${unauthenticated_helper}/OPERATOR-TOOL-MANIFEST.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${unauthenticated_helper}/RELEASE-PACKET.json" \
  '.operator_tool_manifest_sha256 = $sha' \
  --arg sha "${unauthenticated_manifest_sha}"
printf 'tampered\n' >> \
  "${unauthenticated_helper}/RELEASE-PACKET.json.sig"
unauthenticated_helper_marker="${TMP}/unauthenticated-helper-executed"
run_bad_release_with_unauthenticated_helper() {
  FROZEN_HELPER_EXECUTION_MARKER="${unauthenticated_helper_marker}" \
    run_open open-preinit "${unauthenticated_helper}" \
      "${unauthenticated_tool_root}"
}
expect_rejected "helper execution before RELEASE authentication" \
  "RELEASE-PACKET.json detached signature verification failed" \
  run_bad_release_with_unauthenticated_helper
if [ -e "${unauthenticated_helper_marker}" ]; then
  printf 'unauthenticated frozen-evidence helper executed before RELEASE verification\n' \
    >&2
  exit 1
fi

run_future_signature() {
  FAKE_GPG_FUTURE_SIGNATURE=DARK-START-DECISION.json.sig \
    run_cutover_pre "${BASE_BUNDLE}"
}
expect_rejected "future signature timestamp" "in the future" run_future_signature

wrong_lead=$(clone_bundle wrong-halt-lead)
canonical_mutate "${wrong_lead}/CUTOVER-DECISION.json" \
  '.successor_commitment_transaction.timeout_height = "901"'
expect_rejected "insufficient halt lead" "does not preserve the signed halt lead" \
  run_cutover_pre "${wrong_lead}"

trusted_anchor=$(clone_bundle trusted-anchor)
canonical_mutate "${trusted_anchor}/RELEASE-PACKET.json" \
  '.predecessor.trusted_block.app_hash = ("a" * 64)'
expect_rejected "malformed trusted predecessor anchor" \
  "predecessor trusted AppHash is not an exact uppercase SHA-256" \
  run_cutover_pre "${trusted_anchor}"

provenance=$(clone_bundle provenance-artifact)
canonical_mutate "${provenance}/ZERONE-2-RUNTIME-PROVENANCE.json" \
  '.subject.image_digest = ("sha256:" + ("9" * 64))'
provenance_sha=$(sha256_file \
  "${provenance}/ZERONE-2-RUNTIME-PROVENANCE.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${provenance}/RELEASE-PACKET.json" \
  '.components.zerone_2_runtime.provenance_sha256 = $sha' \
  --arg sha "${provenance_sha}"
expect_rejected "provenance subject drift" \
  "provenance does not bind source/build/image/binary" \
  run_cutover_pre "${provenance}"

registration=$(clone_bundle registration-evidence)
canonical_mutate "${registration}/DARK-REGISTRATION-EVIDENCE.json" \
  '.custom_validator_registration.deliver_code = 1'
expect_rejected "registration evidence drift" "registration" \
  run_cutover_pre "${registration}"

transition=$(clone_bundle transition-manifest)
canonical_mutate "${transition}/zerone-1-archive-transition.json" \
  '.archive_transition_nonce = ("8" * 64)'
expect_rejected "transition manifest drift" "archive adoption" \
  run_open_pre "${transition}"

adoption=$(clone_bundle adoption-authority)
canonical_mutate "${adoption}/ARCHIVE-ADOPTION-AUTHORITY.json" \
  '.checkpoint_plan.halt_trigger_height = "1003"'
expect_rejected "archive adoption drift" "archive adoption authority" \
  run_open_pre "${adoption}"

final=$(clone_bundle final-checkpoint)
canonical_mutate "${final}/FINAL-CHECKPOINT.json" \
  'del(.authority_chain.archive_adoption_authority)'
expect_rejected "truncated FINAL checkpoint" "FINAL-CHECKPOINT" \
  run_open_pre "${final}"

rpc_byte_drift=$(clone_bundle rpc-byte-drift)
printf ' \n' >> "${rpc_byte_drift}/OBSERVER-RPC-BLOCK-A.json.raw"
rpc_byte_drift_sha=$(sha256_file \
  "${rpc_byte_drift}/OBSERVER-RPC-BLOCK-A.json.raw")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${rpc_byte_drift}/OBSERVER-EVIDENCE-MANIFEST.json" \
  '.payload_sha256.block_a_json = $sha' \
  --arg sha "${rpc_byte_drift_sha}"
rebind_frozen_source_chain "${rpc_byte_drift}"
expect_rejected "non-status RPC raw byte drift" \
  "terminal signer/observer block_a_json raw bytes do not match" \
  run_open_pre "${rpc_byte_drift}"

collapsed_checkpoint=$(clone_bundle collapsed-checkpoint)
canonical_mutate "${collapsed_checkpoint}/ZERONE-1-INVENTORY-V3.json" \
  '.source.checkpoint_app_hash = .source.excluded_post_anchor_app_hash'
expect_rejected "collapsed checkpoint and post-anchor state" \
  "inventory collapses checkpoint-F state and excluded post-anchor-A state" \
  run_open_pre "${collapsed_checkpoint}"

empty_commit=$(clone_bundle empty-commit)
for prefix in SIGNER OBSERVER; do
  canonical_mutate "${empty_commit}/${prefix}-RPC-COMMIT-A.json.raw" \
    '.result.signed_header.commit.signatures = []'
done
empty_commit_sha=$(sha256_file \
  "${empty_commit}/SIGNER-RPC-COMMIT-A.json.raw")
for prefix in SIGNER OBSERVER; do
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${empty_commit}/${prefix}-EVIDENCE-MANIFEST.json" \
    '.payload_sha256.commit_a_json = $sha' \
    --arg sha "${empty_commit_sha}"
done
rebind_frozen_source_chain "${empty_commit}"
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${empty_commit}/FINAL-CHECKPOINT.json" \
  '.terminal_rpc_evidence.matching_payload_sha256.commit_a_json = $sha' \
  --arg sha "${empty_commit_sha}"
expect_rejected "empty terminal commit signature set" \
  "SIGNER commit A is not the expected signed commit" \
  run_open_pre "${empty_commit}"

empty_commit_signature=$(clone_bundle empty-commit-signature)
for prefix in SIGNER OBSERVER; do
  canonical_mutate \
    "${empty_commit_signature}/${prefix}-RPC-COMMIT-A.json.raw" \
    '.result.signed_header.commit.signatures[0].signature = ""'
done
empty_commit_signature_sha=$(sha256_file \
  "${empty_commit_signature}/SIGNER-RPC-COMMIT-A.json.raw")
for prefix in SIGNER OBSERVER; do
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate \
    "${empty_commit_signature}/${prefix}-EVIDENCE-MANIFEST.json" \
    '.payload_sha256.commit_a_json = $sha' \
    --arg sha "${empty_commit_signature_sha}"
done
rebind_frozen_source_chain "${empty_commit_signature}"
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${empty_commit_signature}/FINAL-CHECKPOINT.json" \
  '.terminal_rpc_evidence.matching_payload_sha256.commit_a_json = $sha' \
  --arg sha "${empty_commit_signature_sha}"
expect_rejected "empty structural commit signature" \
  "SIGNER commit A signature[0] bytes is not canonical base64" \
  run_open_pre "${empty_commit_signature}"

observer_voting=$(clone_bundle observer-voting)
canonical_mutate "${observer_voting}/OBSERVER-RPC-STATUS.json.raw" \
  '.result.validator_info.voting_power = "1"'
observer_status_sha=$(sha256_file \
  "${observer_voting}/OBSERVER-RPC-STATUS.json.raw")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${observer_voting}/OBSERVER-EVIDENCE-MANIFEST.json" \
  '.payload_sha256.status_json = $sha' \
  --arg sha "${observer_status_sha}"
rebind_frozen_source_chain "${observer_voting}"
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${observer_voting}/FINAL-CHECKPOINT.json" \
  '.terminal_rpc_evidence.sources.independent_observer.status_json_sha256 = $sha' \
  --arg sha "${observer_status_sha}"
expect_rejected "observer with nonzero voting power" \
  "OBSERVER status does not prove the expected stable H/A source" \
  run_open_pre "${observer_voting}"

observer_bonded=$(clone_bundle observer-bonded)
observer_key=$(jq -r '.validator_pubkey' \
  "${observer_bonded}/OBSERVER-EVIDENCE-MANIFEST.json")
# shellcheck disable=SC2016 # $key is a jq variable, not a shell variable.
canonical_mutate "${observer_bonded}/ZERONE-1-INVENTORY-V3.json" \
  '.bonded_validators += [{
    operator_address: "zeronevaloper1observer",
    consensus_pubkey: {
      "@type": "/cosmos.crypto.ed25519.PubKey",
      key: $key
    },
    jailed: false,
    status: "BOND_STATUS_BONDED",
    tokens: "1"
  }]
  | .bonded_validators |= sort_by(.operator_address)' \
  --arg key "${observer_key}"
observer_bonded_inventory_sha=$(sha256_file \
  "${observer_bonded}/ZERONE-1-INVENTORY-V3.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${observer_bonded}/FINAL-CHECKPOINT.json" \
  '.checkpoint_state.inventory_v3_sha256 = $sha' \
  --arg sha "${observer_bonded_inventory_sha}"
expect_rejected "observer key reused by bonded validator" \
  "terminal observer key is unexpectedly in the bonded-validator inventory" \
  run_open_pre "${observer_bonded}"

float_height=$(clone_bundle float-inventory-height)
canonical_mutate "${float_height}/ZERONE-1-INVENTORY-V3.json" \
  '.source.checkpoint_state_height = 1000.5'
expect_rejected "floating-point inventory height" \
  "inventory source checkpoint_state_height is not an integer >= 1" \
  run_open_pre "${float_height}"

float_count=$(clone_bundle float-inventory-count)
canonical_mutate "${float_count}/ZERONE-1-INVENTORY-V3.json" \
  '.source.final_committed_block_txs = 0.5'
expect_rejected "floating-point inventory transaction count" \
  "inventory source final_committed_block_txs is not an integer >= 0" \
  run_open_pre "${float_count}"

malformed_url=$(clone_bundle malformed-inventory-url)
canonical_mutate "${malformed_url}/ZERONE-1-INVENTORY-V3.json" \
  '.source.rpc = "http://[::1"'
expect_rejected "malformed inventory URL" \
  "inventory RPC URL is not a valid URL" \
  run_open_pre "${malformed_url}"

oversized_decimal=$(clone_bundle oversized-inventory-decimal)
canonical_mutate "${oversized_decimal}/ZERONE-1-INVENTORY-V3.json" \
  '.supply_uzrn = ("1" * 257)'
expect_rejected "oversized inventory decimal" \
  "inventory supply is not canonical decimal" \
  run_open_pre "${oversized_decimal}"

rollback_output_drift=$(clone_bundle rollback-output-drift)
printf 'tampered rollback output\n' >> \
  "${rollback_output_drift}/ARCHIVE-ROLLBACK-OUTPUT.log"
expect_rejected "raw rollback output drift" \
  "archive rollback evidence differs from the H/A -> A/A operation" \
  run_open_pre "${rollback_output_drift}"

tool_root="${TMP}/drifted-tool-root"
while IFS= read -r relative; do
  mkdir -p "${tool_root}/$(dirname -- "${relative}")"
  cp "${ROOT}/${relative}" "${tool_root}/${relative}"
done < <(jq -r '.files | keys[]' \
  "${BASE_BUNDLE}/OPERATOR-TOOL-MANIFEST.json")
printf '\n# fixture drift\n' >> "${tool_root}/deploy/validate-fly-phase-config.py"
expect_rejected "operator tool drift" "operator tool bytes drifted" \
  run_cutover_pre "${BASE_BUNDLE}" "${tool_root}"

upstream=$(clone_bundle upstream-config)
sed 's/zerone-2-edge\.internal/attacker.internal/g' \
  "${upstream}/fly.zerone-2-gateway.public.toml" \
  > "${upstream}/fly.zerone-2-gateway.public.toml.new"
mv "${upstream}/fly.zerone-2-gateway.public.toml.new" \
  "${upstream}/fly.zerone-2-gateway.public.toml"
expect_rejected "unsigned upstream change" "config bytes differ" \
  run_open_pre "${upstream}"

dns=$(clone_bundle dns-manifest)
canonical_mutate "${dns}/DNS-CHANGE-MANIFEST.json" \
  '.records["rpc.example"].app = "wrong-app"'
dns_sha=$(sha256_file "${dns}/DNS-CHANGE-MANIFEST.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${dns}/OPEN-BETA-DECISION.json" \
  '.public_coordinates.canonical_dns_change_manifest_sha256 = $sha' \
  --arg sha "${dns_sha}"
expect_rejected "DNS record/app drift" \
  "DNS records differ from exact apps/config hashes" run_open_pre "${dns}"

printf 'verify-authority-chain tests: PASS\n'
