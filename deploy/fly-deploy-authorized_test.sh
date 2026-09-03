#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
GATE="${ROOT}/deploy/fly-deploy-authorized.sh"
FIXTURE="${ROOT}/deploy/test-fixtures/make-authority-bundle.py"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-authorized-deploy-test.XXXXXX")
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

expect_rejected() {
  local label=$1 expected=$2
  shift 2
  local output
  if output=$("$@" 2>&1); then
    printf 'authorized deploy test: %s was accepted\n%s\n' \
      "${label}" "${output}" >&2
    exit 1
  fi
  if ! grep -Fq -- "${expected}" <<<"${output}"; then
    printf 'authorized deploy test: %s failed for the wrong reason:\n%s\n' \
      "${label}" "${output}" >&2
    exit 1
  fi
}

MAIN_FINGERPRINT=$(printf 'a%.0s' {1..40})
OTHER_FINGERPRINT=$(printf 'b%.0s' {1..40})
TRANSITION_FINGERPRINT=$(printf 'c%.0s' {1..40})
RUNTIME_IMAGE="registry.example/zerone-2-runtime@sha256:$(printf '1%.0s' {1..64})"
HALT_IMAGE="registry.example/zerone-1-halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query-gateway@sha256:$(printf '3%.0s' {1..64})"
SENDER=zerone1authorizedfixture

cat > "${TMP}/zeroned" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -gt 1 ] && [ "$1" = verify-frozen-terminal ]; then
  printf 'frozen-terminal-crypto: MATCH\n'
elif [ "$#" -eq 3 ] && [ "$1" = tx ] && [ "$2" = encode ]; then
  jq -er '.encoded' "$3"
elif [ "$#" -eq 5 ] && [ "$1" = tx ] && [ "$2" = decode ] && \
  [ "$4" = --output ] && [ "$5" = json ]; then
  cat "${FAKE_DECODED_TX:?}"
elif [ "$#" -ge 3 ] && [ "$1" = tx ] && [ "$2" = validate-signatures ]; then
  for required in '--account-number 0' '--sequence 7' '--offline'; do
    case " $* " in
      *" ${required} "*) ;;
      *) exit 2 ;;
    esac
  done
  case " $* " in
    *' --node '*) exit 2 ;;
  esac
  printf 'Signatures: OK\n'
elif [ "$#" -eq 8 ] && [ "$1" = query ] && [ "$2" = auth ] && \
  [ "$3" = account ] && [ "$4" = "${FAKE_ACCOUNT_ADDRESS:?}" ] && \
  [ "$5" = --node ] && [ "$7" = --output ] && [ "$8" = json ]; then
  exit 99
else
  exit 2
fi
EOF
chmod +x "${TMP}/zeroned"
BINARY_SHA=$(sha256_file "${TMP}/zeroned")

printf 'fixture exact phase TxRaw bytes' > "${TMP}/txraw"
TX_RAW_SHA=$(sha256_file "${TMP}/txraw")
TX_HASH=$(printf '%s' "${TX_RAW_SHA}" | tr '[:lower:]' '[:upper:]')
TX_BASE64=$(base64 < "${TMP}/txraw" | tr -d '\r\n')
jq -n -S -c --arg encoded "${TX_BASE64}" '{encoded:$encoded}' \
  > "${TMP}/signed-tx.json"

PRIVATE_CONFIG="${TMP}/fly.edge.private.toml"
cat > "${PRIVATE_CONFIG}" <<EOF
app = "zerone-2-edge"
primary_region = "lhr"
kill_signal = "SIGTERM"
kill_timeout = "30s"

[build]
  image = "${RUNTIME_IMAGE}"

[deploy]
  strategy = "immediate"

[env]
  NODE_ROLE = "edge"
  ZERONE_HOME = "/data/.zeroned"
  MONIKER = "fixture-edge"
  PERSISTENT_PEERS = "0000000000000000000000000000000000000000@zerone-2-validator.internal:26656"
  PRIVATE_PEER_IDS = "0000000000000000000000000000000000000000"
  CORS_ALLOWED_ORIGINS_JSON = "[]"
  QUERY_ORIGIN_ENABLED = "false"

[mounts]
  source = "fixture_edge_data"
  destination = "/data"

[[vm]]
  size = "shared-cpu-2x"
  memory = "4gb"
EOF

# Resolve the signer and historical signature timestamp from the payload schema
# so this also works after the deployment gate snapshots inputs under local
# filenames. The authority-chain verifier consumes the timestamp from VALIDSIG.
cat > "${TMP}/bin/gpg" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
payload=${@: -1}
schema=$(jq -er '.schema' "${payload}")
fingerprint=${FAKE_GPG_MAIN_FINGERPRINT:?}
case "${schema}" in
  zerone-2-release-packet-v2) timestamp=1783677660 ;;
  zerone-2-dark-start-decision-v1) timestamp=1783678260 ;;
  zerone-2-dark-start-initiation-evidence-v1) timestamp=1783678920 ;;
  zerone-2-dark-registration-evidence-v1) timestamp=1783680000 ;;
  zerone-2-cutover-decision-v1) timestamp=1783688460 ;;
  zerone-2-cutover-initiation-evidence-v1) timestamp=1783692120 ;;
  zerone-1-archive-adoption-authority-v1)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783693800
    ;;
  zerone-custom-staking-census-execution-evidence-v1)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783692840
    ;;
  zerone-final-checkpoint-v4)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783695660
    ;;
  zerone-2-open-beta-decision-v1) timestamp=1783699260 ;;
  zerone-2-open-beta-initiation-evidence-v1) timestamp=1783699920 ;;
  *) exit 2 ;;
esac
printf '[GNUPG:] VALIDSIG %s 2026-07-10 %s 0 4 0 1 10 00 %s\n' \
  "${fingerprint}" "${timestamp}" "${fingerprint}"
EOF
chmod +x "${TMP}/bin/gpg"

cat > "${TMP}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url=${!#}
case "${url}" in
  */status)
    jq -n --arg node "${FAKE_PHASE_NODE:?}" '
      {jsonrpc:"2.0",id:1,result:{node_info:{network:"zerone-2",id:$node},
       sync_info:{catching_up:false,latest_block_height:"80"}}}'
    ;;
  */block\?height=1)
    jq -n --arg hash "${FAKE_PHASE_BLOCK_HASH:?}" \
      --arg app "${FAKE_PHASE_APP_HASH:?}" '
      {jsonrpc:"2.0",id:1,result:{block_id:{hash:$hash},block:{header:{
       chain_id:"zerone-2",height:"1",app_hash:$app}}}}'
    ;;
  *) exit 22 ;;
esac
EOF
chmod +x "${TMP}/bin/curl"

BUNDLE="${TMP}/authority-bundle"
"${FIXTURE}" \
  --output "${BUNDLE}" \
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
  --edge-private-config-file "${PRIVATE_CONFIG}" \
  --release-binary-file "${TMP}/zeroned" \
  --signed-tx-file "${TMP}/signed-tx.json"

make_dark_pre_bundle() {
  local name=$1 destination
  destination="${TMP}/${name}"
  cp -R "${BUNDLE}" "${destination}"
  canonical_mutate "${destination}/DARK-START-DECISION.json" \
    '.authorization_semantics.initiation_deadline = "2099-07-10T11:00:00Z"'
  printf '%s\n' "${destination}"
}

rebind_dark_release() {
  local bundle=$1 release_sha release_signature_sha
  release_sha=$(sha256_file "${bundle}/RELEASE-PACKET.json")
  release_signature_sha=$(sha256_file "${bundle}/RELEASE-PACKET.json.sig")
  canonical_mutate "${bundle}/DARK-START-DECISION.json" \
    '.release_packet_sha256 = $release
    | .release_packet_detached_signature_sha256 = $signature' \
    --arg release "${release_sha}" --arg signature "${release_signature_sha}"
}

DARK_PRE_BUNDLE=$(make_dark_pre_bundle dark-preinit-authority-bundle)

RELEASE="${BUNDLE}/RELEASE-PACKET.json"
RELEASE_SIG="${BUNDLE}/RELEASE-PACKET.json.sig"
DARK="${BUNDLE}/DARK-START-DECISION.json"
DARK_SIG="${BUNDLE}/DARK-START-DECISION.json.sig"
DARK_INIT="${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json"
DARK_INIT_SIG="${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json.sig"
CUTOVER="${BUNDLE}/CUTOVER-DECISION.json"
CUTOVER_SIG="${BUNDLE}/CUTOVER-DECISION.json.sig"
CUTOVER_INIT="${BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json"
CUTOVER_INIT_SIG="${BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json.sig"
OPEN="${BUNDLE}/OPEN-BETA-DECISION.json"
OPEN_SIG="${BUNDLE}/OPEN-BETA-DECISION.json.sig"
OPEN_INIT="${BUNDLE}/OPEN-BETA-INITIATION-EVIDENCE.json"
OPEN_INIT_SIG="${BUNDLE}/OPEN-BETA-INITIATION-EVIDENCE.json.sig"
FINAL="${BUNDLE}/FINAL-CHECKPOINT.json"
FINAL_SIG="${BUNDLE}/FINAL-CHECKPOINT.json.sig"
PUBLIC_CONFIG="${BUNDLE}/fly.edge.public.toml"
ARCHIVE_CONFIG="${BUNDLE}/fly.zerone-1-archive-gateway.public.toml"

jq -n -S -c \
  --arg sender "${SENDER}" \
  --arg final "$(sha256_file "${FINAL}")" '
  {
    body: {
      messages: [{
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        from_address:$sender,
        to_address:$sender,
        amount:[{denom:"uzrn",amount:"1"}]
      }],
      memo:("zerone_1_final_checkpoint_sha256=" + $final),
      timeout_height:"1300",
      extension_options:[],
      non_critical_extension_options:[]
    },
    auth_info: {
      signer_infos:[{
        mode_info:{single:{mode:"SIGN_MODE_DIRECT"}},sequence:"7"
      }],
      fee:{
        amount:[{denom:"uzrn",amount:"200000"}],
        gas_limit:"200000",
        payer:"",
        granter:""
      }
    },
    signatures:["fixture"]
  }
' > "${TMP}/decoded-open-tx.json"

gate_env() {
  PATH="${TMP}/bin:${PATH}" \
    FAKE_GPG_MAIN_FINGERPRINT="${FAKE_GPG_MAIN_FINGERPRINT:-${MAIN_FINGERPRINT}}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
    FAKE_DECODED_TX="${TMP}/decoded-open-tx.json" \
    FAKE_ACCOUNT_ADDRESS="${SENDER}" FAKE_ACCOUNT_SEQUENCE=7 \
    FAKE_PHASE_NODE="$(jq -er '.public_identities.validator_node_id' "${RELEASE}")" \
    FAKE_PHASE_BLOCK_HASH="$(jq -er '.first_committed_block.block_id_hash' "${DARK_INIT}")" \
    FAKE_PHASE_APP_HASH="$(jq -er '.first_committed_block.app_hash' "${DARK_INIT}")" \
    "$@"
}

run_dark() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${DARK}" "${DARK_SIG}" \
    "${PRIVATE_CONFIG}" zerone_2_edge_private "${MAIN_FINGERPRINT}" \
    "${DARK_INIT}" "${DARK_INIT_SIG}" "${BUNDLE}"
}

run_dark_pre() {
  local bundle=${1:-${DARK_PRE_BUNDLE}}
  gate_env "${GATE}" --check \
    "${bundle}/RELEASE-PACKET.json" \
    "${bundle}/RELEASE-PACKET.json.sig" \
    "${bundle}/DARK-START-DECISION.json" \
    "${bundle}/DARK-START-DECISION.json.sig" \
    "${PRIVATE_CONFIG}" zerone_2_edge_private "${MAIN_FINGERPRINT}" \
    "${bundle}"
}

run_open() {
  local config=${1:-${PUBLIC_CONFIG}}
  local transition=${2:-${TRANSITION_FINGERPRINT}}
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${OPEN}" "${OPEN_SIG}" \
    "${config}" zerone_2_edge_public "${MAIN_FINGERPRINT}" \
    "${OPEN_INIT}" "${OPEN_INIT_SIG}" \
    "${FINAL}" "${FINAL_SIG}" "${transition}" "${BUNDLE}"
}

run_open_archive() {
  local config=${1:-${ARCHIVE_CONFIG}}
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${OPEN}" "${OPEN_SIG}" \
    "${config}" zerone_1_archive_gateway "${MAIN_FINGERPRINT}" \
    "${OPEN_INIT}" "${OPEN_INIT_SIG}" \
    "${FINAL}" "${FINAL_SIG}" "${TRANSITION_FINGERPRINT}" "${BUNDLE}"
}

[ "$(run_dark_pre)" = "${RUNTIME_IMAGE}" ]
[ "$(run_dark)" = "${RUNTIME_IMAGE}" ]
[ "$(run_open)" = "${RUNTIME_IMAGE}" ]
[ "$(run_open_archive)" = "${QUERY_IMAGE}" ]

run_dark_without_bundle() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${DARK}" "${DARK_SIG}" \
    "${PRIVATE_CONFIG}" zerone_2_edge_private "${MAIN_FINGERPRINT}" \
    "${DARK_INIT}" "${DARK_INIT_SIG}"
}
expect_rejected "DARK deployment without authority bundle" \
  "DARK-START deployment requires the complete authority bundle" \
  run_dark_without_bundle

missing_dark_sigstore=$(make_dark_pre_bundle dark-preinit-missing-sigstore)
mv "${missing_dark_sigstore}/SIGSTORE-TRUSTED-ROOT.json" \
  "${missing_dark_sigstore}/SIGSTORE-TRUSTED-ROOT.json.missing"
expect_rejected "DARK deployment with missing Sigstore root" \
  "could not open bundle file SIGSTORE-TRUSTED-ROOT.json" \
  run_dark_pre "${missing_dark_sigstore}"

corrupt_dark_sigstore=$(make_dark_pre_bundle dark-preinit-corrupt-sigstore)
bad_component_signature=$(printf 'x%.0s' {1..80} | base64 | tr -d '\r\n')
canonical_mutate \
  "${corrupt_dark_sigstore}/ZERONE-2-RUNTIME-SIGNATURE-BUNDLE.json" \
  '.messageSignature.signature = $signature' \
  --arg signature "${bad_component_signature}"
component_bundle_sha=$(sha256_file \
  "${corrupt_dark_sigstore}/ZERONE-2-RUNTIME-SIGNATURE-BUNDLE.json")
canonical_mutate \
  "${corrupt_dark_sigstore}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json" \
  '.bundle_sha256 = $sha' --arg sha "${component_bundle_sha}"
component_evidence_sha=$(sha256_file \
  "${corrupt_dark_sigstore}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json")
canonical_mutate "${corrupt_dark_sigstore}/RELEASE-PACKET.json" \
  '.components.zerone_2_runtime.signature_sha256 = $sha' \
  --arg sha "${component_evidence_sha}"
rebind_dark_release "${corrupt_dark_sigstore}"
expect_rejected "DARK deployment with corrupt Sigstore signature" \
  "Sigstore cryptographic verification failed" \
  run_dark_pre "${corrupt_dark_sigstore}"

corrupt_dark_verifier=$(make_dark_pre_bundle dark-preinit-corrupt-verifier)
printf '\n# corrupt bundled verifier\n' >> \
  "${corrupt_dark_verifier}/zerone-component-signature-verifier"
expect_rejected "DARK deployment with corrupt component verifier" \
  "bundled component-signature verifier bytes differ" \
  run_dark_pre "${corrupt_dark_verifier}"

run_cutover_generic() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${CUTOVER}" "${CUTOVER_SIG}" \
    "${BUNDLE}/fly.observer.toml" zerone_1_observer "${MAIN_FINGERPRINT}" \
    "${CUTOVER_INIT}" "${CUTOVER_INIT_SIG}" "${BUNDLE}"
}
expect_rejected "generic CUTOVER deployment" \
  "CUTOVER must use mainnet/fly-cutover-authorized.sh" run_cutover_generic

run_open_without_evidence() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${OPEN}" "${OPEN_SIG}" \
    "${PUBLIC_CONFIG}" zerone_2_edge_public "${MAIN_FINGERPRINT}"
}
expect_rejected "OPEN without initiation evidence" \
  "OPEN-BETA deployment requires signed initiation evidence" \
  run_open_without_evidence

run_open_without_final() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${OPEN}" "${OPEN_SIG}" \
    "${PUBLIC_CONFIG}" zerone_2_edge_public "${MAIN_FINGERPRINT}" \
    "${OPEN_INIT}" "${OPEN_INIT_SIG}"
}
expect_rejected "OPEN without FINAL" \
  "OPEN-BETA deployment requires the transition-signed final checkpoint" \
  run_open_without_final

expect_rejected "wrong OPEN transition fingerprint" \
  "final-checkpoint signer differs from the release transition key" \
  run_open "${PUBLIC_CONFIG}" "${OTHER_FINGERPRINT}"

ALTERED_CONFIG="${TMP}/fly.edge.public.altered.toml"
cp "${PUBLIC_CONFIG}" "${ALTERED_CONFIG}"
printf '\n# changed after OPEN was signed\n' >> "${ALTERED_CONFIG}"
expect_rejected "post-signature Fly config drift" \
  "snapshotted config does not match the signed phase-authority SHA-256" \
  run_open "${ALTERED_CONFIG}"

ALTERED_ARCHIVE_CONFIG="${TMP}/fly.archive-gateway.altered.toml"
cp "${ARCHIVE_CONFIG}" "${ALTERED_ARCHIVE_CONFIG}"
sed -i.bak 's/EXPECTED_ARCHIVE_HEIGHT = "1001"/EXPECTED_ARCHIVE_HEIGHT = "1000"/' \
  "${ALTERED_ARCHIVE_CONFIG}"
rm -f "${ALTERED_ARCHIVE_CONFIG}.bak"
expect_rejected "archive gateway FINAL pin drift" \
  "archive gateway height differs from verified FINAL A" \
  run_open_archive "${ALTERED_ARCHIVE_CONFIG}"

run_open_wrong_main_signature() {
  FAKE_GPG_MAIN_FINGERPRINT=${OTHER_FINGERPRINT} run_open
}
expect_rejected "wrong main-key signature" \
  "release packet signature was made by a different key" \
  run_open_wrong_main_signature

run_open_wrong_key() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${OPEN}" "${OPEN_SIG}" \
    "${PUBLIC_CONFIG}" zerone_2_validator "${MAIN_FINGERPRINT}" \
    "${OPEN_INIT}" "${OPEN_INIT_SIG}" \
    "${FINAL}" "${FINAL_SIG}" "${TRANSITION_FINGERPRINT}" "${BUNDLE}"
}
expect_rejected "OPEN config outside signed phase scope" \
  "config key is outside open-beta authority" run_open_wrong_key

printf 'Fly signed-phase deployment gate tests: PASS\n'
