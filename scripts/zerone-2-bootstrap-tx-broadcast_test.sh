#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
GATE="${ROOT}/scripts/zerone-2-bootstrap-tx-broadcast.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-bootstrap-gate-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

fail() {
  printf 'zerone-2 bootstrap tx gate test: %s\n' "$*" >&2
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
PHASE_RAW="${TMP}/phase.raw"
printf 'fixture phase TxRaw bytes' > "${PHASE_RAW}"
PHASE_SHA=$(sha256_file "${PHASE_RAW}")
PHASE_HASH=$(printf '%s' "${PHASE_SHA}" | tr '[:lower:]' '[:upper:]')
PHASE_ENCODED=$(base64 < "${PHASE_RAW}" | tr -d '\r\n')
PHASE_TX="${TMP}/phase-tx.json"
jq -n --arg encoded "${PHASE_ENCODED}" '{encoded:$encoded}' > "${PHASE_TX}"

FAKE_BINARY="${TMP}/zeroned"
cat > "${FAKE_BINARY}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = tx ] && [ "$2" = encode ] && [ "$#" -eq 3 ]; then
  jq -er '.encoded' "$3"
  exit 0
fi
if [ "$1" = tx ] && [ "$2" = decode ] && [ "$4" = --output ] && \
  [ "$5" = json ]; then
  encoded=$3
  dark=${FAKE_DARK:?}
  if [ "${encoded}" = "${FAKE_ONBOARD_ENCODED:?}" ]; then
    path=.private_bootstrap_transactions.operator_onboarding
    jq -n --argjson contract "$(jq -c "${path}" "${dark}")" '
      {
        body:{messages:[{
          "@type":"/zerone.auth.v1.MsgRegisterAccount",
          sender:$contract.sender,did:$contract.did,
          public_key:$contract.public_key,account_type:$contract.account_type,
          operational_key_hash:$contract.operational_key_hash,
          metadata:$contract.metadata
        }],memo:$contract.memo,timeout_height:$contract.timeout_height,
        extension_options:[],non_critical_extension_options:[]},
        auth_info:{signer_infos:[{
          public_key:{"@type":"/cosmos.crypto.secp256k1.PubKey",key:("A"*44)},
          mode_info:{single:{mode:"SIGN_MODE_DIRECT"}},
          sequence:$contract.signer_sequence
        }],fee:{amount:[{denom:"uzrn",amount:"200000"}],
          gas_limit:"200000",payer:"",granter:""},tip:null},
        signatures:["fixture-signature"]
      }
    '
    exit 0
  fi
  if [ "${encoded}" = "${FAKE_REGISTER_ENCODED:?}" ]; then
    path=.private_bootstrap_transactions.custom_validator_registration
    jq -n --argjson contract "$(jq -c "${path}" "${dark}")" '
      {
        body:{messages:[{
          "@type":"/zerone.staking.v1.MsgRegisterValidator",
          operator:$contract.operator,consensus_pubkey:$contract.consensus_pubkey,
          did:$contract.did,moniker:$contract.moniker,
          self_delegation:$contract.self_delegation,
          commission_bps:$contract.commission_bps,website:$contract.website,
          details:$contract.details
        }],memo:$contract.memo,timeout_height:$contract.timeout_height,
        extension_options:[],non_critical_extension_options:[]},
        auth_info:{signer_infos:[{
          public_key:{"@type":"/cosmos.crypto.secp256k1.PubKey",key:("A"*44)},
          mode_info:{single:{mode:"SIGN_MODE_DIRECT"}},
          sequence:$contract.signer_sequence
        }],fee:{amount:[{denom:"uzrn",amount:"200000"}],
          gas_limit:"200000",payer:"",granter:""},tip:null},
        signatures:["fixture-signature"]
      }
    '
    exit 0
  fi
fi
exit 2
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

cat > "${TMP}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
request=
url=
while [ "$#" -gt 0 ]; do
  if [ "$1" = --data-binary ]; then
    shift
    request=$1
  fi
  url=$1
  shift
done
case "${url}" in
  */status)
    jq -n --arg node "${FAKE_SUCCESSOR_NODE_ID:?}" '
      {jsonrpc:"2.0",id:1,result:{node_info:{network:"zerone-2",id:$node},
       sync_info:{catching_up:false,latest_block_height:"20"}}}'
    ;;
  */block\?height=1)
    jq -n --arg hash "${FAKE_BLOCK_ONE_HASH:?}" --arg app "${FAKE_BLOCK_ONE_APP:?}" '
      {jsonrpc:"2.0",id:1,result:{block_id:{hash:$hash},block:{header:{
       chain_id:"zerone-2",height:"1",app_hash:$app,
       time:"2026-07-10T10:20:00.123456789Z"}}}}'
    ;;
  */tx\?hash=0x*)
    hash=${url##*0x}
    hash=${hash%%&*}
    if [ "${hash}" = "${FAKE_ONBOARD_HASH:?}" ]; then
      tx=${FAKE_ONBOARD_ENCODED:?}
      height=10
    elif [ "${hash}" = "${FAKE_REGISTER_HASH:?}" ]; then
      tx=${FAKE_REGISTER_ENCODED:?}
      height=11
    else
      exit 22
    fi
    jq -n --arg hash "${hash}" --arg tx "${tx}" --arg height "${height}" '
      {jsonrpc:"2.0",id:1,result:{hash:$hash,height:$height,tx:$tx,
       tx_result:{code:0}}}'
    ;;
  */block\?height=10)
    jq -n '{jsonrpc:"2.0",id:1,result:{block:{header:{height:"10",
      time:"2026-07-10T10:30:00.111111111Z"}}}}'
    ;;
  */block\?height=11)
    jq -n '{jsonrpc:"2.0",id:1,result:{block:{header:{height:"11",
      time:"2026-07-10T10:31:00.222222222Z"}}}}'
    ;;
  *)
    tx=$(printf '%s' "${request}" | jq -er '.params.tx')
    if [ "${tx}" = "${FAKE_ONBOARD_ENCODED:?}" ]; then
      hash=${FAKE_ONBOARD_HASH:?}
    elif [ "${tx}" = "${FAKE_REGISTER_ENCODED:?}" ]; then
      hash=${FAKE_REGISTER_HASH:?}
    else
      exit 22
    fi
    jq -n --arg hash "${hash}" \
      '{jsonrpc:"2.0",id:1,result:{code:0,hash:$hash}}'
    ;;
esac
EOF
chmod +x "${TMP}/bin/gpg" "${TMP}/bin/curl"

BUNDLE="${TMP}/bundle"
RUNTIME_IMAGE="registry.example/runtime@sha256:$(printf '1%.0s' {1..64})"
HALT_IMAGE="registry.example/halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query@sha256:$(printf '3%.0s' {1..64})"
"${ROOT}/deploy/test-fixtures/make-authority-bundle.py" \
  --output "${BUNDLE}" --main "${MAIN}" --transition "${TRANSITION}" \
  --runtime-binary-sha "${BINARY_SHA}" --halt-binary-sha "${BINARY_SHA}" \
  --runtime-image "${RUNTIME_IMAGE}" --halt-image "${HALT_IMAGE}" \
  --query-image "${QUERY_IMAGE}" --genesis-sha "$(printf '4%.0s' {1..64})" \
  --tx-raw-sha "${PHASE_SHA}" --tx-hash "${PHASE_HASH}" \
  --sender zrn1fixturephaseaddress \
  --release-binary-file "${FAKE_BINARY}" --signed-tx-file "${PHASE_TX}"

DARK="${BUNDLE}/DARK-START-DECISION.json"
ONBOARD_ENCODED=$(jq -er '.encoded' "${BUNDLE}/ZERONE-2-ONBOARD-SIGNED-TX.json")
REGISTER_ENCODED=$(jq -er '.encoded' \
  "${BUNDLE}/ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json")
ONBOARD_HASH=$(jq -er \
  '.private_bootstrap_transactions.operator_onboarding.expected_transaction_hash' \
  "${DARK}")
REGISTER_HASH=$(jq -er \
  '.private_bootstrap_transactions.custom_validator_registration.expected_transaction_hash' \
  "${DARK}")
NODE_ID=$(jq -er '.public_identities.validator_node_id' \
  "${BUNDLE}/RELEASE-PACKET.json")
BLOCK_ONE_HASH=$(jq -er '.first_committed_block.block_id_hash' \
  "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json")
BLOCK_ONE_APP=$(jq -er '.first_committed_block.app_hash' \
  "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json")

run_gate() {
  local role=$1 mode=${2:-broadcast}
  local -a args
  args=(
    "${BUNDLE}/RELEASE-PACKET.json" "${BUNDLE}/RELEASE-PACKET.json.sig"
    "${DARK}" "${BUNDLE}/DARK-START-DECISION.json.sig"
    "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json"
    "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json.sig"
    "${role}" "${FAKE_BINARY}" http://successor.internal:26657
    "${MAIN}" "${BUNDLE}"
  )
  [ "${mode}" = check ] && args=(--check "${args[@]}")
  PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN}" \
    FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION}" FAKE_DARK="${DARK}" \
    FAKE_ONBOARD_ENCODED="${ONBOARD_ENCODED}" \
    FAKE_REGISTER_ENCODED="${REGISTER_ENCODED}" \
    FAKE_ONBOARD_HASH="${ONBOARD_HASH}" FAKE_REGISTER_HASH="${REGISTER_HASH}" \
    FAKE_SUCCESSOR_NODE_ID="${NODE_ID}" FAKE_BLOCK_ONE_HASH="${BLOCK_ONE_HASH}" \
    FAKE_BLOCK_ONE_APP="${BLOCK_ONE_APP}" "${GATE}" "${args[@]}"
}

[ "$(run_gate operator-onboarding check)" = "${ONBOARD_HASH}" ] || \
  fail "onboarding check did not return its exact hash"
[ "$(run_gate custom-validator-registration check)" = "${REGISTER_HASH}" ] || \
  fail "registration check did not return its exact hash"
[ "$(run_gate operator-onboarding)" = "${ONBOARD_HASH}" ] || \
  fail "onboarding broadcast did not prove commit"
[ "$(run_gate custom-validator-registration)" = "${REGISTER_HASH}" ] || \
  fail "registration broadcast did not prove ordered commit"

BAD_TX="${TMP}/bad-onboard.json"
jq -n --arg encoded "$(printf 'wrong raw' | base64 | tr -d '\r\n')" \
  '{encoded:$encoded}' > "${BAD_TX}"
BAD_BUNDLE="${TMP}/bad-bundle"
cp -R "${BUNDLE}" "${BAD_BUNDLE}"
cp "${BAD_TX}" "${BAD_BUNDLE}/ZERONE-2-ONBOARD-SIGNED-TX.json"
if PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${MAIN}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION}" FAKE_DARK="${DARK}" \
  "${GATE}" --check "${BUNDLE}/RELEASE-PACKET.json" \
    "${BUNDLE}/RELEASE-PACKET.json.sig" "${DARK}" \
    "${BUNDLE}/DARK-START-DECISION.json.sig" \
    "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json" \
    "${BUNDLE}/DARK-START-INITIATION-EVIDENCE.json.sig" \
    operator-onboarding "${FAKE_BINARY}" http://successor.internal:26657 \
    "${MAIN}" "${BAD_BUNDLE}" >/dev/null 2>&1; then
  fail "mutated bundled onboarding transaction was accepted"
fi

printf 'zerone-2 signed bootstrap transaction tests: PASS\n'
