#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
RENDERER="${ROOT}/deploy/mainnet/render-archive-configs.sh"
DEPLOY_GATE="${ROOT}/deploy/fly-deploy-pinned.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-archive-render-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

MAIN_FINGERPRINT=$(printf 'a%.0s' {1..40})
TRANSITION_FINGERPRINT=$(printf 'b%.0s' {1..40})
RUNTIME_IMAGE="registry.example/zerone-2-runtime@sha256:$(printf '1%.0s' {1..64})"
IMAGE_REF="registry.example/zerone-1-halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query-gateway@sha256:$(printf '3%.0s' {1..64})"

printf 'exact CUTOVER TxRaw fixture' > "${TMP}/tx.raw"
TX_RAW_SHA=$(sha256_file "${TMP}/tx.raw")
TX_HASH=$(printf '%s' "${TX_RAW_SHA}" | tr '[:lower:]' '[:upper:]')
TX_BASE64=$(base64 < "${TMP}/tx.raw" | tr -d '\r\n')
jq -n -S -c --arg encoded "${TX_BASE64}" '{encoded:$encoded}' \
  > "${TMP}/signed-tx.json"

FAKE_BINARY="${TMP}/zeroned"
cat > "${FAKE_BINARY}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = tx ] && [ "$2" = validate-signatures ] && [ "$#" -ge 3 ]; then
  for required in '--account-number 0' '--sequence 7' '--offline'; do
    case " $* " in
      *" ${required} "*) ;;
      *) exit 2 ;;
    esac
  done
  case " $* " in
    *' --node '*) exit 2 ;;
  esac
  printf 'Signers and signature order: OK\n'
elif [ "$1" = tx ] && [ "$2" = encode ] && [ "$#" -eq 3 ]; then
  jq -er '.encoded' "$3"
elif [ "$1" = tx ] && [ "$2" = decode ] && [ "$4" = --output ] && \
  [ "$5" = json ]; then
  jq -n --argjson tx "$(jq -c '.successor_commitment_transaction' \
      "${FAKE_CUTOVER:?}")" '
    {
      body:{messages:[{"@type":"/cosmos.bank.v1beta1.MsgSend",
        from_address:$tx.sender,to_address:$tx.sender,
        amount:[{denom:"uzrn",amount:"1"}]}],memo:$tx.memo,
        timeout_height:$tx.timeout_height,extension_options:[],
        non_critical_extension_options:[]},
      auth_info:{signer_infos:[{mode_info:{single:{mode:"SIGN_MODE_DIRECT"}},
        sequence:"7"}],fee:{amount:[{denom:"uzrn",amount:"200000"}],
        gas_limit:"200000",payer:"",granter:""},tip:null},
      signatures:["fixture-signature"]
    }'
else
  exit 2
fi
EOF
chmod +x "${FAKE_BINARY}"
BINARY_SHA=$(sha256_file "${FAKE_BINARY}")

AUTHORITY_BUNDLE="${TMP}/authority-bundle"
"${ROOT}/deploy/test-fixtures/make-authority-bundle.py" \
  --output "${AUTHORITY_BUNDLE}" \
  --main "${MAIN_FINGERPRINT}" --transition "${TRANSITION_FINGERPRINT}" \
  --runtime-binary-sha "${BINARY_SHA}" --halt-binary-sha "${BINARY_SHA}" \
  --runtime-image "${RUNTIME_IMAGE}" --halt-image "${IMAGE_REF}" \
  --query-image "${QUERY_IMAGE}" --genesis-sha "$(printf '4%.0s' {1..64})" \
  --tx-raw-sha "${TX_RAW_SHA}" --tx-hash "${TX_HASH}" \
  --sender zerone1archiverenderfixture \
  --release-binary-file "${FAKE_BINARY}" \
  --signed-tx-file "${TMP}/signed-tx.json"

RELEASE="${AUTHORITY_BUNDLE}/RELEASE-PACKET.json"
RELEASE_SIG="${AUTHORITY_BUNDLE}/RELEASE-PACKET.json.sig"
CUTOVER="${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json"
CUTOVER_SIG="${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig"
CUTOVER_INITIATION="${AUTHORITY_BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json"
CUTOVER_INITIATION_SIG="${AUTHORITY_BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json.sig"
TRANSITION="${AUTHORITY_BUNDLE}/zerone-1-archive-transition.json"
RELEASE_SHA=$(sha256_file "${RELEASE}")
RELEASE_SIG_SHA=$(sha256_file "${RELEASE_SIG}")

cat > "${TMP}/bin/gpg" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
fingerprint=${FAKE_GPG_FINGERPRINT:?}
timestamp=1783677660
case "$*" in
  *DARK-START-DECISION.json.sig*) timestamp=1783678260 ;;
  *DARK-START-INITIATION-EVIDENCE.json.sig*) timestamp=1783678920 ;;
  *DARK-REGISTRATION-EVIDENCE.json.sig*) timestamp=1783680000 ;;
  *CUTOVER-DECISION.json.sig*) timestamp=1783688460 ;;
  *CUTOVER-INITIATION-EVIDENCE.json.sig*) timestamp=1783692120 ;;
  *ARCHIVE-ADOPTION-AUTHORITY.json.sig*|*signed-adoption.json.sig*)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783693800
    ;;
  *FINAL-CHECKPOINT.json.sig*)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783695660
    ;;
  *OPEN-BETA-DECISION.json.sig*) timestamp=1783699260 ;;
  *OPEN-BETA-INITIATION-EVIDENCE.json.sig*) timestamp=1783699920 ;;
esac
printf '[GNUPG:] VALIDSIG %s 2026-07-10 %s 0 4 0 1 10 00 %s\n' \
  "${fingerprint}" "${timestamp}" "${fingerprint}"
EOF
cat > "${TMP}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'offline artifact validation attempted an RPC call\n' >&2
exit 99
EOF
cat > "${TMP}/bin/date" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'offline artifact validation applied a broadcast-cutoff clock check\n' >&2
exit 99
EOF
chmod +x "${TMP}/bin/gpg" "${TMP}/bin/curl" "${TMP}/bin/date"

render() {
  local output=$1
  PATH="${TMP}/bin:${PATH}" \
  FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
    "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" \
      "${CUTOVER}" "${CUTOVER_SIG}" \
      "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
      "${TRANSITION}" "${output}" "${AUTHORITY_BUNDLE}"
}

render "${TMP}/out-a" >/dev/null
render "${TMP}/out-b" >/dev/null
for name in fly.archive-candidate.toml fly.archive.toml \
  ARCHIVE-ADOPTION-AUTHORITY.json; do
  cmp -s "${TMP}/out-a/${name}" "${TMP}/out-b/${name}"
done

ADOPTION="${TMP}/out-a/ARCHIVE-ADOPTION-AUTHORITY.json"
jq -e \
  --arg release "${RELEASE_SHA}" \
  --arg release_sig "${RELEASE_SIG_SHA}" \
  --arg cutover "$(sha256_file "${CUTOVER}")" \
  --arg cutover_sig "$(sha256_file "${CUTOVER_SIG}")" \
  --arg transition "$(sha256_file "${TRANSITION}")" \
  --arg candidate_config "$(sha256_file "${TMP}/out-a/fly.archive-candidate.toml")" \
  --arg archive_config "$(sha256_file "${TMP}/out-a/fly.archive.toml")" '
    .schema == "zerone-1-archive-adoption-authority-v1" and
    .attestation_result == "MATCH" and
    .release_packet == {
      sha256: $release, detached_signature_sha256: $release_sig
    } and
    .cutover_decision == {
      sha256: $cutover, detached_signature_sha256: $cutover_sig
    } and
    .archive_transition_manifest.sha256 == $transition and
    .deployment_configs.zerone_1_archive_candidate.sha256 == $candidate_config and
    .deployment_configs.zerone_1_archive.sha256 == $archive_config and
    .enforced_origin_invariants.public_fly_service == false and
    .enforced_origin_invariants.public_ip == false and
    .enforced_origin_invariants.transaction_ingress == false and
    .enforced_origin_invariants.persistent_peers == []
  ' "${ADOPTION}" >/dev/null
cmp -s "${ADOPTION}" <(jq -S -c . "${ADOPTION}")
cmp -s "${ADOPTION}" \
  "${AUTHORITY_BUNDLE}/ARCHIVE-ADOPTION-AUTHORITY.json"

for tuple in \
  'fly.archive-candidate.toml|zerone_1_archive_candidate' \
  'fly.archive.toml|zerone_1_archive'; do
  config=${tuple%%|*}
  key=${tuple##*|}
  app=$(jq -er ".deployment_configs.${key}.app" "${ADOPTION}")
  role=$(jq -er ".deployment_configs.${key}.role" "${ADOPTION}")
  config_sha=$(jq -er ".deployment_configs.${key}.sha256" "${ADOPTION}")
  "${DEPLOY_GATE}" --check "${TMP}/out-a/${config}" \
    "${app}" "${IMAGE_REF}" "${role}" "${config_sha}" >/dev/null
done

ARCHIVE_DEPLOY_GATE="${ROOT}/deploy/mainnet/fly-deploy-archive-authorized.sh"
ADOPTION_SIG="${TMP}/out-a/ARCHIVE-ADOPTION-AUTHORITY.json.sig"
cp "${AUTHORITY_BUNDLE}/ARCHIVE-ADOPTION-AUTHORITY.json.sig" "${ADOPTION_SIG}"
for key in zerone_1_archive_candidate zerone_1_archive; do
  PATH="${TMP}/bin:${PATH}" \
  FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
    "${ARCHIVE_DEPLOY_GATE}" --check \
      "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" "${CUTOVER_SIG}" \
      "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
      "${TRANSITION}" "${ADOPTION}" "${ADOPTION_SIG}" "${key}" \
      "${MAIN_FINGERPRINT}" "${TRANSITION_FINGERPRINT}" \
      "${AUTHORITY_BUNDLE}" >/dev/null
done

FORGED_ADOPTION="${TMP}/forged-adoption.json"
jq -S -c '.deployment_configs.zerone_1_archive_candidate.app = "zerone-1"' \
  "${ADOPTION}" > "${FORGED_ADOPTION}"
if PATH="${TMP}/bin:${PATH}" \
  FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
  "${ARCHIVE_DEPLOY_GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" "${CUTOVER_SIG}" \
    "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${TRANSITION}" "${FORGED_ADOPTION}" "${ADOPTION_SIG}" \
    zerone_1_archive_candidate "${MAIN_FINGERPRINT}" \
    "${TRANSITION_FINGERPRINT}" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: hand-authored adoption authority was accepted\n' >&2
  exit 1
fi

if render "${TMP}/out-a" >/dev/null 2>&1; then
  printf 'archive renderer test: existing output directory was accepted\n' >&2
  exit 1
fi

WRONG_TRANSITION="${TMP}/wrong-transition.json"
jq '.halt_trigger_height = "103"' "${TRANSITION}" > "${WRONG_TRANSITION}"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" \
    "${CUTOVER_SIG}" "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${WRONG_TRANSITION}" "${TMP}/wrong-height" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: wrong transition height was accepted\n' >&2
  exit 1
fi

LATE_TRANSITION="${TMP}/late-transition.json"
jq '.cutover_initiation_evidence.committed_block_time = "2099-07-12T12:00:01Z"' \
  "${TRANSITION}" > "${LATE_TRANSITION}"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" \
    "${CUTOVER_SIG}" "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${LATE_TRANSITION}" "${TMP}/late-initiation" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: late cutover initiation was accepted\n' >&2
  exit 1
fi

if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" \
    "${CUTOVER_SIG}" "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${TRANSITION}" "${TMP}/wrong-signature" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: wrong release signer was accepted\n' >&2
  exit 1
fi

WRONG_SEMANTICS="${TMP}/wrong-semantics.json"
jq -S -c '.deterministic_private_continuation.attestation_algorithm = "none"' \
  "${CUTOVER}" > "${WRONG_SEMANTICS}"
WRONG_SEMANTICS_EVIDENCE="${TMP}/wrong-semantics-evidence.json"
jq -S -c --arg cutover "$(sha256_file "${WRONG_SEMANTICS}")" \
  '.cutover_decision.sha256 = $cutover' "${CUTOVER_INITIATION}" \
  > "${WRONG_SEMANTICS_EVIDENCE}"
WRONG_SEMANTICS_TRANSITION="${TMP}/wrong-semantics-transition.json"
jq --arg evidence "$(sha256_file "${WRONG_SEMANTICS_EVIDENCE}")" \
  '.cutover_initiation_evidence.initiation_evidence_sha256 = $evidence' \
  "${TRANSITION}" > "${WRONG_SEMANTICS_TRANSITION}"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${WRONG_SEMANTICS}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" "${WRONG_SEMANTICS}" \
    "${CUTOVER_SIG}" "${WRONG_SEMANTICS_EVIDENCE}" \
    "${CUTOVER_INITIATION_SIG}" "${WRONG_SEMANTICS_TRANSITION}" \
    "${TMP}/wrong-semantics" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: contradictory signed semantics were accepted\n' >&2
  exit 1
fi

INVALID_TIME_CUTOVER="${TMP}/invalid-time-cutover.json"
jq -S -c \
  '.authorization_semantics.initiation_deadline = "9999-99-99T99:99:99Z"' \
  "${CUTOVER}" > "${INVALID_TIME_CUTOVER}"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${INVALID_TIME_CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${RELEASE}" "${RELEASE_SIG}" "${INVALID_TIME_CUTOVER}" \
    "${CUTOVER_SIG}" "${CUTOVER_INITIATION}" "${CUTOVER_INITIATION_SIG}" \
    "${TRANSITION}" "${TMP}/invalid-time" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: impossible deadline was accepted\n' >&2
  exit 1
fi

SAME_KEY_RELEASE="${TMP}/same-key-release.json"
SAME_KEY_CUTOVER="${TMP}/same-key-cutover.json"
jq -S -c --arg same "$(printf '%s' "${MAIN_FINGERPRINT}" | tr '[:lower:]' '[:upper:]')" \
  '.public_identities.transition_attestation.authorized_signer_fingerprint = $same' \
  "${RELEASE}" > "${SAME_KEY_RELEASE}"
SAME_KEY_RELEASE_SHA=$(sha256_file "${SAME_KEY_RELEASE}")
jq -S -c --arg same "$(printf '%s' "${MAIN_FINGERPRINT}" | tr '[:lower:]' '[:upper:]')" \
  --arg release "${SAME_KEY_RELEASE_SHA}" '
    .release_packet_sha256 = $release |
    .deterministic_private_continuation.authorized_transition_signer_fingerprint = $same
  ' "${CUTOVER}" > "${SAME_KEY_CUTOVER}"
SAME_KEY_EVIDENCE="${TMP}/same-key-evidence.json"
jq -S -c --arg cutover "$(sha256_file "${SAME_KEY_CUTOVER}")" \
  '.cutover_decision.sha256 = $cutover' "${CUTOVER_INITIATION}" \
  > "${SAME_KEY_EVIDENCE}"
SAME_KEY_TRANSITION="${TMP}/same-key-transition.json"
jq --arg evidence "$(sha256_file "${SAME_KEY_EVIDENCE}")" \
  '.cutover_initiation_evidence.initiation_evidence_sha256 = $evidence' \
  "${TRANSITION}" > "${SAME_KEY_TRANSITION}"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN_FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_CUTOVER="${SAME_KEY_CUTOVER}" \
  ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT="${MAIN_FINGERPRINT}" \
  "${RENDERER}" "${SAME_KEY_RELEASE}" "${RELEASE_SIG}" \
    "${SAME_KEY_CUTOVER}" "${CUTOVER_SIG}" \
    "${SAME_KEY_EVIDENCE}" "${CUTOVER_INITIATION_SIG}" \
    "${SAME_KEY_TRANSITION}" "${TMP}/same-key" "${AUTHORITY_BUNDLE}" \
    >/dev/null 2>&1; then
  printf 'archive renderer test: same main/transition key was accepted\n' >&2
  exit 1
fi

printf 'archive deterministic renderer tests: PASS\n'
