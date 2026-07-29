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
  zerone-2-release-packet-v1) timestamp=1783677660 ;;
  zerone-2-dark-start-decision-v1) timestamp=1783678260 ;;
  zerone-2-dark-start-initiation-evidence-v1) timestamp=1783678920 ;;
  zerone-2-dark-registration-evidence-v1) timestamp=1783680000 ;;
  zerone-2-cutover-decision-v1) timestamp=1783688460 ;;
  zerone-2-cutover-initiation-evidence-v1) timestamp=1783692120 ;;
  zerone-1-archive-adoption-authority-v1)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783693800
    ;;
  zerone-final-checkpoint-v3)
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
      signer_infos:[{}],
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
    "$@"
}

run_dark() {
  gate_env "${GATE}" --check \
    "${RELEASE}" "${RELEASE_SIG}" "${DARK}" "${DARK_SIG}" \
    "${PRIVATE_CONFIG}" zerone_2_edge_private "${MAIN_FINGERPRINT}" \
    "${DARK_INIT}" "${DARK_INIT_SIG}"
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

[ "$(run_dark)" = "${RUNTIME_IMAGE}" ]
[ "$(run_open)" = "${RUNTIME_IMAGE}" ]

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
