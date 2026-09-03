#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
GATE="${ROOT}/scripts/zerone-phase-tx-broadcast.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-phase-tx-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

FINGERPRINT=$(printf 'a%.0s' {1..40})
GENESIS_SHA=$(printf 'b%.0s' {1..64})
FINAL_SHA=$(printf 'c%.0s' {1..64})
SENDER=zerone1testsender
printf 'exact signed TxRaw bytes' > "${TMP}/raw"
RAW_SHA=$(sha256_file "${TMP}/raw")
TX_HASH=$(printf '%s' "${RAW_SHA}" | tr '[:lower:]' '[:upper:]')
ENCODED=$(base64 < "${TMP}/raw" | tr -d '\r\n')
TX_FILE="${TMP}/signed-tx.json"
jq -n --arg encoded "${ENCODED}" '{encoded:$encoded}' > "${TX_FILE}"

FAKE_BINARY="${TMP}/zeroned"
cat > "${FAKE_BINARY}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = verify-frozen-terminal ]; then
  printf 'frozen-terminal-crypto: MATCH\n'
elif [ "$1" = tx ] && [ "$2" = encode ] && [ "$#" -eq 3 ]; then
  jq -er '.encoded' "$3"
elif [ "$1" = tx ] && [ "$2" = decode ] && [ "$4" = --output ] && \
  [ "$5" = json ]; then
  cat "${FAKE_DECODED_TX:?}"
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
  *CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json.sig*)
    fingerprint=${FAKE_GPG_TRANSITION_FINGERPRINT:?}
    timestamp=1783692840
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
    jq -n --arg chain "${EXPECTED_FAKE_CHAIN_ID:?}" \
      --arg node "${EXPECTED_FAKE_NODE_ID:?}" \
      '{jsonrpc:"2.0",id:1,result:{
        node_info:{network:$chain,id:$node},
        sync_info:{catching_up:false,latest_block_height:"80"}
      }}'
    exit 0
    ;;
  */tx\?hash=*)
    jq -n --arg hash "${EXPECTED_FAKE_TX_HASH:?}" \
      --arg tx "${EXPECTED_FAKE_TX_BASE64:?}" '
      {jsonrpc:"2.0",id:1,result:{
        hash:$hash,height:"81",tx:$tx,tx_result:{code:0}
      }}'
    exit 0
    ;;
  */block\?height=81)
    jq -n --arg chain "${EXPECTED_FAKE_CHAIN_ID:?}" \
      '{jsonrpc:"2.0",id:1,result:{block:{header:{
        chain_id:$chain,height:"81",time:"2026-07-10T16:10:00.123456789Z"
      }}}}'
    exit 0
    ;;
  */block\?height=*)
    height=${url##*=}
    [ "${height}" = "${EXPECTED_FAKE_TRUSTED_HEIGHT:?}" ]
    jq -n --arg chain "${EXPECTED_FAKE_CHAIN_ID:?}" \
      --arg height "${height}" --arg hash "${EXPECTED_FAKE_TRUSTED_BLOCK_HASH:?}" \
      --arg app "${EXPECTED_FAKE_TRUSTED_APP_HASH:?}" '
      {jsonrpc:"2.0",id:1,result:{block_id:{hash:$hash},block:{header:{
        chain_id:$chain,height:$height,app_hash:$app,time:"2026-07-10T10:00:00Z"
      }}}}'
    exit 0
    ;;
esac
[ "$(printf '%s' "${request}" | jq -er '.method')" = broadcast_tx_sync ]
[ "$(printf '%s' "${request}" | jq -er '.params.tx')" = \
  "${EXPECTED_FAKE_TX_BASE64:?}" ]
jq -n --arg hash "${EXPECTED_FAKE_TX_HASH:?}" \
  '{jsonrpc:"2.0",id:1,result:{code:0,hash:$hash}}'
EOF
chmod +x "${TMP}/bin/gpg" "${TMP}/bin/curl"

RELEASE="${TMP}/RELEASE-PACKET.json"
RELEASE_SIG="${TMP}/RELEASE-PACKET.json.sig"
jq -n -S -c \
  --arg fingerprint "${FINGERPRINT}" --arg binary "${BINARY_SHA}" \
  --arg genesis "${GENESIS_SHA}" '
  {
    schema: "zerone-2-release-packet-v2",
    chain_id: "zerone-2",
    signature_authority: {
      algorithm: "openpgp",
      authorized_signer_fingerprint: $fingerprint,
      detached_signature_filename: "RELEASE-PACKET.json.sig"
    },
    components: {
      zerone_1_halt: {binary_sha256:$binary},
      zerone_2_runtime: {binary_sha256:$binary}
    },
    genesis: {sha256:$genesis}
  }
' > "${RELEASE}"
printf 'release signature\n' > "${RELEASE_SIG}"
RELEASE_SHA=$(sha256_file "${RELEASE}")
RELEASE_SIG_SHA=$(sha256_file "${RELEASE_SIG}")

TRANSITION_FINGERPRINT=$(printf 'b%.0s' {1..40})
RUNTIME_IMAGE="registry.example/runtime@sha256:$(printf '1%.0s' {1..64})"
HALT_IMAGE="registry.example/halt@sha256:$(printf '2%.0s' {1..64})"
QUERY_IMAGE="registry.example/query@sha256:$(printf '3%.0s' {1..64})"
AUTHORITY_BUNDLE="${TMP}/authority-bundle"
"${ROOT}/deploy/test-fixtures/make-authority-bundle.py" \
  --output "${AUTHORITY_BUNDLE}" \
  --main "${FINGERPRINT}" --transition "${TRANSITION_FINGERPRINT}" \
  --runtime-binary-sha "${BINARY_SHA}" --halt-binary-sha "${BINARY_SHA}" \
  --runtime-image "${RUNTIME_IMAGE}" --halt-image "${HALT_IMAGE}" \
  --query-image "${QUERY_IMAGE}" --genesis-sha "${GENESIS_SHA}" \
  --tx-raw-sha "${RAW_SHA}" --tx-hash "${TX_HASH}" --sender "${SENDER}" \
  --release-binary-file "${FAKE_BINARY}" --signed-tx-file "${TX_FILE}"
RELEASE="${AUTHORITY_BUNDLE}/RELEASE-PACKET.json"
RELEASE_SIG="${AUTHORITY_BUNDLE}/RELEASE-PACKET.json.sig"
RELEASE_SHA=$(sha256_file "${RELEASE}")
RELEASE_SIG_SHA=$(sha256_file "${RELEASE_SIG}")
GENESIS_SHA=$(jq -er '.genesis.sha256' "${RELEASE}")
FINAL_SHA=$(sha256_file "${AUTHORITY_BUNDLE}/FINAL-CHECKPOINT.json")

make_decision() {
  local phase=$1 deadline=$2 output=$3
  if [ "${phase}" = cutover ]; then
    jq -n -S -c \
      --arg fingerprint "${FINGERPRINT}" --arg release "${RELEASE_SHA}" \
      --arg release_sig "${RELEASE_SIG_SHA}" --arg deadline "${deadline}" \
      --arg raw "${RAW_SHA}" --arg hash "${TX_HASH}" \
      --arg sender "${SENDER}" --arg genesis "${GENESIS_SHA}" '
      {
        schema: "zerone-2-cutover-decision-v1", decision: "GO",
        signature_authority: {
          algorithm:"openpgp", authorized_signer_fingerprint:$fingerprint,
          detached_signature_filename:"CUTOVER-DECISION.json.sig"
        },
        release_packet_sha256:$release,
        release_packet_detached_signature_sha256:$release_sig,
        authorization_semantics:{initiation_deadline:$deadline},
        checkpoint_plan:{
          checkpoint_state_height:"100",
          final_committed_anchor_height:"101",
          halt_trigger_height:"102"
        },
        successor_commitment_transaction:{
          chain_id:"zerone-1", signed_tx_bytes_sha256:$raw,
          expected_transaction_hash:$hash,
          sender:$sender,
          message_type:"/cosmos.bank.v1beta1.MsgSend",
          recipient_equals_sender:true,
          amount:"1uzrn",fee:"200000uzrn",gas_limit:"200000",
          memo:("successor_chain_id=zerone-2;successor_genesis_sha256=" +
            $genesis +
            ";checkpoint_state_height=100;final_committed_height=101;halt_trigger_height=102"),
          must_not_reference_cutover_payload:true
        }
      }
    ' > "${output}"
  else
    jq -n -S -c \
      --arg fingerprint "${FINGERPRINT}" --arg release "${RELEASE_SHA}" \
      --arg release_sig "${RELEASE_SIG_SHA}" --arg deadline "${deadline}" \
      --arg raw "${RAW_SHA}" --arg hash "${TX_HASH}" \
      --arg sender "${SENDER}" --arg final "${FINAL_SHA}" '
      {
        schema: "zerone-2-open-beta-decision-v1", decision: "GO",
        signature_authority: {
          algorithm:"openpgp", authorized_signer_fingerprint:$fingerprint,
          detached_signature_filename:"OPEN-BETA-DECISION.json.sig"
        },
        release_packet:{sha256:$release,detached_signature_sha256:$release_sig},
        authorization_semantics:{initiation_deadline:$deadline},
        final_checkpoint:{sha256:$final,detached_signature_sha256:("d"*64)},
        history_link_transaction:{
          chain_id:"zerone-2", signed_tx_bytes_sha256:$raw,
          expected_transaction_hash:$hash,
          sender:$sender,
          message_type:"/cosmos.bank.v1beta1.MsgSend",
          recipient_equals_sender:true,
          amount:"1uzrn",fee:"200000uzrn",gas_limit:"200000",
          memo:("zerone_1_final_checkpoint_sha256=" + $final),
          must_not_reference_open_beta_payload:true
        }
      }
    ' > "${output}"
  fi
}

make_decoded() {
  local memo=$1 recipient=$2 fee=$3 gas=$4 timeout=$5 output=$6
  jq -n \
    --arg sender "${SENDER}" --arg recipient "${recipient}" \
    --arg memo "${memo}" --arg fee "${fee}" --arg gas "${gas}" \
    --arg timeout "${timeout}" '
    {
      body: {
        messages: [{
          "@type":"/cosmos.bank.v1beta1.MsgSend",
          from_address:$sender,
          to_address:$recipient,
          amount:[{denom:"uzrn",amount:"1"}]
        }],
        memo:$memo,
        timeout_height:$timeout,
        extension_options:[],
        non_critical_extension_options:[]
      },
      auth_info: {
        signer_infos:[{}],
        fee:{amount:[{denom:"uzrn",amount:$fee}],gas_limit:$gas,payer:"",granter:""},
        tip:null
      },
      signatures:["test-signature"]
    }
  ' > "${output}"
}

CUTOVER_MEMO="successor_chain_id=zerone-2;successor_genesis_sha256=${GENESIS_SHA};checkpoint_state_height=1000;final_committed_height=1001;halt_trigger_height=1002"
OPEN_MEMO="zerone_1_final_checkpoint_sha256=${FINAL_SHA}"
CUTOVER_DECODED="${TMP}/decoded-cutover.json"
OPEN_DECODED="${TMP}/decoded-open.json"
make_decoded "${CUTOVER_MEMO}" "${SENDER}" 200000 200000 900 \
  "${CUTOVER_DECODED}"
make_decoded "${OPEN_MEMO}" "${SENDER}" 200000 200000 1300 "${OPEN_DECODED}"

run_gate() {
  local phase=$4 decoded=${6:-} chain bundle=${8:-${AUTHORITY_BUNDLE}}
  local node trusted_height trusted_block trusted_app
  local -a gate_args
  if [ -z "${decoded}" ]; then
    case "${phase}" in
      cutover) decoded=${CUTOVER_DECODED}; chain=zerone-1 ;;
      open-beta) decoded=${OPEN_DECODED}; chain=zerone-2 ;;
    esac
  else
    case "${phase}" in
      cutover) chain=zerone-1 ;;
      open-beta) chain=zerone-2 ;;
    esac
  fi
  [ -z "${7:-}" ] || chain=$7
  case "${phase}" in
    cutover)
      node=$(jq -er '.predecessor.trusted_rpc_node_id' "${RELEASE}")
      trusted_height=$(jq -er '.predecessor.trusted_block.height' "${RELEASE}")
      trusted_block=$(jq -er '.predecessor.trusted_block.block_id_hash' "${RELEASE}")
      trusted_app=$(jq -er '.predecessor.trusted_block.app_hash' "${RELEASE}")
      ;;
    open-beta)
      node=$(jq -er '.public_identities.validator_node_id' "${RELEASE}")
      trusted_height=1
      trusted_block=$(jq -er '.first_committed_block.block_id_hash' \
        "${bundle}/DARK-START-INITIATION-EVIDENCE.json")
      trusted_app=$(jq -er '.first_committed_block.app_hash' \
        "${bundle}/DARK-START-INITIATION-EVIDENCE.json")
      ;;
  esac
  gate_args=(
    "${RELEASE}" "${RELEASE_SIG}" "$1" "$2" "$3" "$4" "$5"
    http://private-rpc.internal:26657 "${FINGERPRINT}" "${bundle}"
  )
  if [ "${phase}" = open-beta ]; then
    gate_args+=("${TRANSITION_FINGERPRINT}")
  fi
  PATH="${TMP}/bin:${PATH}" FAKE_GPG_FINGERPRINT="${FINGERPRINT}" \
  FAKE_GPG_TRANSITION_FINGERPRINT="${TRANSITION_FINGERPRINT}" \
  FAKE_DECODED_TX="${decoded}" EXPECTED_FAKE_CHAIN_ID="${chain}" \
  EXPECTED_FAKE_NODE_ID="${node}" \
  EXPECTED_FAKE_TRUSTED_HEIGHT="${trusted_height}" \
  EXPECTED_FAKE_TRUSTED_BLOCK_HASH="${trusted_block}" \
  EXPECTED_FAKE_TRUSTED_APP_HASH="${trusted_app}" \
  EXPECTED_FAKE_TX_BASE64="${ENCODED}" EXPECTED_FAKE_TX_HASH="${TX_HASH}" \
    "${GATE}" "${gate_args[@]}"
}

for phase in cutover open-beta; do
  case "${phase}" in
    cutover)
      DECISION="${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json"
      DECISION_SIG="${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig"
      ;;
    open-beta)
      DECISION="${AUTHORITY_BUNDLE}/OPEN-BETA-DECISION.json"
      DECISION_SIG="${AUTHORITY_BUNDLE}/OPEN-BETA-DECISION.json.sig"
      ;;
  esac
  [ "$(run_gate "${DECISION}" "${DECISION_SIG}" "${TX_FILE}" \
    "${phase}" "${FAKE_BINARY}")" = "${TX_HASH}" ]
done

for mutation in recipient memo fee gas; do
  decoded="${TMP}/decoded-wrong-${mutation}.json"
  recipient=${SENDER}
  memo=${CUTOVER_MEMO}
  fee=200000
  gas=200000
  case "${mutation}" in
    recipient) recipient=zerone1wrongrecipient ;;
    memo) memo='wrong checkpoint memo' ;;
    fee) fee=199999 ;;
    gas) gas=199999 ;;
  esac
  make_decoded "${memo}" "${recipient}" "${fee}" "${gas}" 900 "${decoded}"
  if run_gate "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" \
    "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig" "${TX_FILE}" cutover \
    "${FAKE_BINARY}" "${decoded}" >/dev/null 2>&1; then
    printf 'phase tx gate test: wrong decoded %s was broadcast\n' "${mutation}" >&2
    exit 1
  fi
done

if run_gate "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" \
  "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig" "${TX_FILE}" cutover \
  "${FAKE_BINARY}" "${CUTOVER_DECODED}" zerone-2 >/dev/null 2>&1; then
  printf 'phase tx gate test: wrong RPC chain was accepted\n' >&2
  exit 1
fi

EXPIRED="${TMP}/expired-cutover.json"
EXPIRED_BUNDLE="${TMP}/expired-bundle"
cp -R "${AUTHORITY_BUNDLE}" "${EXPIRED_BUNDLE}"
jq -S -c '.authorization_semantics.initiation_deadline = "2000-01-01T00:00:00Z"' \
  "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" > "${EXPIRED}"
cp "${EXPIRED}" "${EXPIRED_BUNDLE}/CUTOVER-DECISION.json"
if run_gate "${EXPIRED}" "${EXPIRED_BUNDLE}/CUTOVER-DECISION.json.sig" \
  "${TX_FILE}" cutover "${FAKE_BINARY}" "" "" "${EXPIRED_BUNDLE}" \
  >/dev/null 2>&1; then
  printf 'phase tx gate test: expired decision was broadcast\n' >&2
  exit 1
fi

INVALID_DEADLINE="${TMP}/invalid-deadline-cutover.json"
INVALID_BUNDLE="${TMP}/invalid-bundle"
cp -R "${AUTHORITY_BUNDLE}" "${INVALID_BUNDLE}"
jq -S -c \
  '.authorization_semantics.initiation_deadline = "9999-99-99T99:99:99Z"' \
  "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" > "${INVALID_DEADLINE}"
cp "${INVALID_DEADLINE}" "${INVALID_BUNDLE}/CUTOVER-DECISION.json"
if run_gate "${INVALID_DEADLINE}" "${INVALID_BUNDLE}/CUTOVER-DECISION.json.sig" \
  "${TX_FILE}" cutover "${FAKE_BINARY}" "" "" "${INVALID_BUNDLE}" \
  >/dev/null 2>&1; then
  printf 'phase tx gate test: impossible deadline was accepted\n' >&2
  exit 1
fi

WRONG_TX="${TMP}/wrong-tx.json"
WRONG_ENCODED=$(printf 'different raw bytes' | base64 | tr -d '\r\n')
jq -n --arg encoded "${WRONG_ENCODED}" '{encoded:$encoded}' > "${WRONG_TX}"
if run_gate "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" \
  "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig" "${WRONG_TX}" cutover \
  "${FAKE_BINARY}" >/dev/null 2>&1; then
  printf 'phase tx gate test: wrong signed transaction bytes were broadcast\n' >&2
  exit 1
fi

ALTERED_BINARY="${TMP}/altered-zeroned"
cp "${FAKE_BINARY}" "${ALTERED_BINARY}"
printf '# altered\n' >> "${ALTERED_BINARY}"
chmod +x "${ALTERED_BINARY}"
if run_gate "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json" \
  "${AUTHORITY_BUNDLE}/CUTOVER-DECISION.json.sig" "${TX_FILE}" cutover \
  "${ALTERED_BINARY}" >/dev/null 2>&1; then
  printf 'phase tx gate test: wrong release binary was used\n' >&2
  exit 1
fi

printf 'signed phase transaction broadcast tests: PASS\n'
