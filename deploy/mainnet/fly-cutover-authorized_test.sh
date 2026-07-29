#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
GATE="${ROOT}/deploy/mainnet/fly-cutover-authorized.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/fly-cutover-authorized-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

fail() {
  printf 'specialized CUTOVER Fly gate test: %s\n' "$*" >&2
  exit 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

MAIN=$(printf 'a%.0s' {1..40})
TRANSITION=$(printf 'b%.0s' {1..40})
RAW="${TMP}/phase.raw"
printf 'exact CUTOVER TxRaw fixture' > "${RAW}"
RAW_SHA=$(sha256_file "${RAW}")
TX_HASH=$(printf '%s' "${RAW_SHA}" | tr '[:lower:]' '[:upper:]')
ENCODED=$(base64 < "${RAW}" | tr -d '\r\n')
TX_FILE="${TMP}/phase-tx.json"
jq -n --arg encoded "${ENCODED}" '{encoded:$encoded}' > "${TX_FILE}"

FAKE_BINARY="${TMP}/zeroned"
cat > "${FAKE_BINARY}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = tx ] && [ "$2" = encode ] && [ "$#" -eq 3 ]; then
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
      auth_info:{signer_infos:[{}],fee:{amount:[{denom:"uzrn",amount:"200000"}],
        gas_limit:"200000",payer:"",granter:""},tip:null},
      signatures:["fixture-signature"]
    }'
else
  exit 2
fi
EOF
chmod +x "${FAKE_BINARY}"
BINARY_SHA=$(sha256_file "${FAKE_BINARY}")

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
  *ARCHIVE-ADOPTION-AUTHORITY.json.sig*)
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

cat > "${TMP}/bin/flyctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[ "$1" = deploy ]
app=
while [ "$#" -gt 0 ]; do
  [ "$1" = --app ] && { shift; app=$1; }
  shift
done
[ -n "${app}" ]
printf '%s\n' "${app}" >> "${FAKE_FLY_LOG:?}"
EOF

cat > "${TMP}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
url=${!#}
case "${url}" in
  *signer.internal*/status) node=${FAKE_SIGNER_NODE:?} ;;
  *observer.internal*/status) node=${FAKE_OBSERVER_NODE:?} ;;
  */status) exit 22 ;;
  *) node= ;;
esac
if [ -n "${node}" ]; then
  jq -n --arg node "${node}" --arg height "${FAKE_LIVE_HEIGHT:-80}" '
    {jsonrpc:"2.0",id:1,result:{node_info:{network:"zerone-1",id:$node},
      sync_info:{catching_up:false,latest_block_height:$height}}}'
  exit 0
fi
case "${url}" in
  */block\?height=*)
    height=${url##*=}
    if [ "${height}" = "${FAKE_TRUSTED_HEIGHT:?}" ]; then
      hash=${FAKE_TRUSTED_BLOCK:?}
      app=${FAKE_TRUSTED_APP:?}
    else
      [ "${height}" = "${FAKE_LIVE_HEIGHT:-80}" ]
      hash=$(printf 'E%.0s' {1..64})
      app=$(printf 'F%.0s' {1..64})
    fi
    jq -n --arg height "${height}" --arg hash "${hash}" --arg app "${app}" '
      {jsonrpc:"2.0",id:1,result:{block_id:{hash:$hash},block:{header:{
        chain_id:"zerone-1",height:$height,app_hash:$app}}}}'
    ;;
  *) exit 22 ;;
esac
EOF
chmod +x "${TMP}/bin/gpg" "${TMP}/bin/flyctl" "${TMP}/bin/curl"

BUNDLE="${TMP}/bundle"
RUNTIME_IMAGE="registry.example/runtime@sha256:$(printf '1%.0s' {1..64})"
HALT_IMAGE="registry.example/halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query@sha256:$(printf '3%.0s' {1..64})"
"${ROOT}/deploy/test-fixtures/make-authority-bundle.py" \
  --output "${BUNDLE}" --main "${MAIN}" --transition "${TRANSITION}" \
  --runtime-binary-sha "${BINARY_SHA}" --halt-binary-sha "${BINARY_SHA}" \
  --runtime-image "${RUNTIME_IMAGE}" --halt-image "${HALT_IMAGE}" \
  --query-image "${QUERY_IMAGE}" --genesis-sha "$(printf '4%.0s' {1..64})" \
  --tx-raw-sha "${RAW_SHA}" --tx-hash "${TX_HASH}" \
  --sender zrn1fixturephaseaddress \
  --release-binary-file "${FAKE_BINARY}" --signed-tx-file "${TX_FILE}"

RELEASE="${BUNDLE}/RELEASE-PACKET.json"
CUTOVER="${BUNDLE}/CUTOVER-DECISION.json"
FLY_LOG="${TMP}/fly.log"
: > "${FLY_LOG}"

run_gate() {
  local stage=$1 mode=${2:-deploy} height=${3:-80}
  local -a args
  args=("${stage}" "${RELEASE}" "${BUNDLE}/RELEASE-PACKET.json.sig"
    "${CUTOVER}" "${BUNDLE}/CUTOVER-DECISION.json.sig")
  if [ "${stage}" = signer ]; then
    args+=("${BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json"
      "${BUNDLE}/CUTOVER-INITIATION-EVIDENCE.json.sig")
  fi
  args+=("${BUNDLE}/fly.halt-signer.toml" "${BUNDLE}/fly.observer.toml"
    "${MAIN}" "${BUNDLE}" http://signer.internal:26657
    http://observer.internal:26657)
  [ "${mode}" = check ] && args=(--check "${args[@]}")
  PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION}" FAKE_CUTOVER="${CUTOVER}" \
    FAKE_FLY_LOG="${FLY_LOG}" \
    FAKE_SIGNER_NODE="$(jq -er '.predecessor.trusted_rpc_node_id' "${RELEASE}")" \
    FAKE_OBSERVER_NODE="$(jq -er '.predecessor.trusted_observer_node_id' "${RELEASE}")" \
    FAKE_TRUSTED_HEIGHT="$(jq -er '.predecessor.trusted_block.height' "${RELEASE}")" \
    FAKE_TRUSTED_BLOCK="$(jq -er '.predecessor.trusted_block.block_id_hash' "${RELEASE}")" \
    FAKE_TRUSTED_APP="$(jq -er '.predecessor.trusted_block.app_hash' "${RELEASE}")" \
    FAKE_LIVE_HEIGHT="${height}" "${GATE}" "${args[@]}"
}

run_gate observer check >/dev/null
run_gate signer check >/dev/null
[ ! -s "${FLY_LOG}" ] || fail "check mode contacted Fly"
run_gate observer >/dev/null
run_gate signer >/dev/null
APP_COUNT=$(wc -l < "${FLY_LOG}" | tr -d ' ')
[ "${APP_COUNT}" -eq 2 ] || fail "expected exactly two ordered deployments"
[ "$(sed -n '1p' "${FLY_LOG}")" = zerone-1-observer ] || \
  fail "observer was not deployed first"
[ "$(sed -n '2p' "${FLY_LOG}")" = zerone-1 ] || \
  fail "signer was not deployed last"

if run_gate signer deploy 950 >/dev/null 2>&1; then
  fail "signer deployment accepted a live height without the signed lead before F"
fi

printf 'specialized observer-first/signer-last CUTOVER tests: PASS\n'
