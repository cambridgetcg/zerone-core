#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'zerone-phase-tx-broadcast: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  scripts/zerone-phase-tx-broadcast.sh [--check] \
    RELEASE.json RELEASE.json.sig DECISION.json DECISION.json.sig \
    SIGNED_TX.json cutover|open-beta RELEASE_BINARY PRIVATE_RPC_URL \
    EXPECTED_MAIN_FINGERPRINT AUTHORITY_BUNDLE_DIRECTORY \
    [EXPECTED_TRANSITION_FINGERPRINT_FOR_OPEN]

The wrapper snapshots every direct input and verifies the transitive authority
bundle. CUTOVER requires the exact signed DARK chain and the byte-matching soak,
rehearsal, and notice artifacts. OPEN requires the transition-signed full FINAL
chain and exact adoption/readiness/revalidation artifacts. It then verifies the
deadline and exact transaction semantics before broadcast_tx_sync.
USAGE
  exit 2
}

MODE=broadcast
if [ "${1:-}" = --check ]; then
  MODE=check
  shift
fi
[ "$#" -ge 10 ] || usage
RELEASE=$1
RELEASE_SIG=$2
DECISION=$3
DECISION_SIG=$4
TX_FILE=$5
PHASE=$6
BINARY=$7
RPC_URL=$8
EXPECTED_SIGNER=$9
AUTHORITY_BUNDLE=${10:-}
EXPECTED_TRANSITION_SIGNER=${11:-}
RELEASE_SIG_NAME=$(basename -- "${RELEASE_SIG}")
DECISION_SIG_NAME=$(basename -- "${DECISION_SIG}")

case "${PHASE}" in
  cutover)
    [ "$#" -eq 10 ] || usage
    [ -z "${EXPECTED_TRANSITION_SIGNER}" ] || usage
    ;;
  open-beta)
    [ "$#" -eq 11 ] || usage
    [[ "${EXPECTED_TRANSITION_SIGNER}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
      die "OPEN transition signer must be a full 40- or 64-hex fingerprint"
    ;;
  *) usage ;;
esac
[[ "${EXPECTED_SIGNER}" =~ ^[0-9A-Fa-f]{40}([0-9A-Fa-f]{24})?$ ]] || \
  die "expected signer must be a full 40- or 64-hex fingerprint"
PRIVATE_RPC_RE='^https?://(localhost|127\.0\.0\.1|[a-z0-9]([a-z0-9.-]*[a-z0-9])?\.internal|10\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}|192\.168\.[0-9]{1,3}\.[0-9]{1,3}|172\.(1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3})(:[0-9]{1,5})?/?$'
[[ "${RPC_URL}" =~ ${PRIVATE_RPC_RE} ]] || \
  die "RPC URL must be a loopback, RFC1918, or .internal HTTP(S) origin"

require_regular() {
  local path=$1 label=$2
  [ -f "${path}" ] || die "${label} is not a regular file"
  [ ! -L "${path}" ] || die "${label} must not be a symlink"
}
require_regular "${RELEASE}" "release packet"
require_regular "${RELEASE_SIG}" "release signature"
require_regular "${DECISION}" "phase decision"
require_regular "${DECISION_SIG}" "phase decision signature"
require_regular "${TX_FILE}" "signed transaction"
require_regular "${BINARY}" "release binary"
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
CHAIN_VERIFIER="${ROOT}/deploy/verify-authority-chain.py"
CONFIG_POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
FINAL_TEMPLATE="${ROOT}/deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json"
OPEN_TEMPLATE="${ROOT}/deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json"
ADOPTION_TEMPLATE="${ROOT}/deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json"
require_regular "${CHAIN_VERIFIER}" "authority-chain verifier"
require_regular "${CONFIG_POLICY}" "Fly phase config policy"
require_regular "${FINAL_TEMPLATE}" "FINAL checkpoint template"
require_regular "${OPEN_TEMPLATE}" "OPEN-BETA template"
require_regular "${ADOPTION_TEMPLATE}" "archive adoption template"
require_regular "${AUTHORITY_BUNDLE}/DARK-START-INITIATION-EVIDENCE.json" \
  "bundled DARK-START initiation evidence"
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v gpg >/dev/null 2>&1 || die "gpg is required"
command -v curl >/dev/null 2>&1 || die "curl is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-phase-tx.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
install -m 0600 "${RELEASE}" "${TMP}/release.json"
install -m 0600 "${RELEASE_SIG}" "${TMP}/release.json.sig"
install -m 0600 "${DECISION}" "${TMP}/decision.json"
install -m 0600 "${DECISION_SIG}" "${TMP}/decision.json.sig"
install -m 0600 "${TX_FILE}" "${TMP}/signed-tx.json"
install -m 0700 "${BINARY}" "${TMP}/zeroned"
install -m 0700 "${CHAIN_VERIFIER}" "${TMP}/verify-authority-chain.py"
install -m 0700 "${CONFIG_POLICY}" "${TMP}/validate-fly-phase-config.py"
install -m 0600 "${FINAL_TEMPLATE}" "${TMP}/FINAL-CHECKPOINT.example.json"
install -m 0600 "${OPEN_TEMPLATE}" "${TMP}/OPEN-BETA-DECISION.example.json"
install -m 0600 "${ADOPTION_TEMPLATE}" \
  "${TMP}/ARCHIVE-ADOPTION-AUTHORITY.example.json"
install -m 0600 "${AUTHORITY_BUNDLE}/DARK-START-INITIATION-EVIDENCE.json" \
  "${TMP}/DARK-START-INITIATION-EVIDENCE.json"
RELEASE="${TMP}/release.json"
RELEASE_SIG="${TMP}/release.json.sig"
DECISION="${TMP}/decision.json"
DECISION_SIG="${TMP}/decision.json.sig"
TX_FILE="${TMP}/signed-tx.json"
BINARY="${TMP}/zeroned"
CHAIN_VERIFIER="${TMP}/verify-authority-chain.py"
CONFIG_POLICY="${TMP}/validate-fly-phase-config.py"
FINAL_TEMPLATE="${TMP}/FINAL-CHECKPOINT.example.json"
OPEN_TEMPLATE="${TMP}/OPEN-BETA-DECISION.example.json"
ADOPTION_TEMPLATE="${TMP}/ARCHIVE-ADOPTION-AUTHORITY.example.json"
DARK_INIT_SNAPSHOT="${TMP}/DARK-START-INITIATION-EVIDENCE.json"

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
require_canonical() {
  local payload=$1 label=$2 output
  output="${TMP}/$(basename -- "${payload}").canonical"
  jq -S -c . "${payload}" > "${output}" || die "${label} is not valid JSON"
  cmp -s "${payload}" "${output}" || die "${label} is not canonical JSON"
  grep -Eq 'REPLACE_|replace-' "${payload}" && die "${label} retains a placeholder"
  return 0
}
verify_signature() {
  local payload=$1 signature=$2 signature_name=$3 label=$4
  local signer declared algorithm status count valid
  signer=$(jq -er '.signature_authority.authorized_signer_fingerprint' \
    "${payload}")
  declared=$(jq -er '.signature_authority.detached_signature_filename' \
    "${payload}")
  algorithm=$(jq -er '.signature_authority.algorithm' "${payload}")
  [ "${algorithm}" = "openpgp" ] || die "${label} signature is not OpenPGP"
  [ "${declared}" = "${signature_name}" ] || \
    die "${label} signature filename differs from its declaration"
  [ "$(normalize_fingerprint "${signer}")" = \
    "$(normalize_fingerprint "${EXPECTED_SIGNER}")" ] || \
    die "${label} repeats the wrong main fingerprint"
  if ! status=$(gpg --batch --status-fd=1 --verify "${signature}" \
    "${payload}" 2>/dev/null); then
    die "${label} signature verification failed"
  fi
  count=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { count++ } END { print count + 0 }')
  [ "${count}" -eq 1 ] || die "${label} must produce one VALIDSIG"
  valid=$(printf '%s\n' "${status}" | awk \
    '$1 == "[GNUPG:]" && $2 == "VALIDSIG" { print $3 }')
  [ "$(normalize_fingerprint "${valid}")" = \
    "$(normalize_fingerprint "${EXPECTED_SIGNER}")" ] || \
    die "${label} was signed by a different key"
}

require_canonical "${RELEASE}" "release packet"
require_canonical "${DECISION}" "phase decision"
verify_signature "${RELEASE}" "${RELEASE_SIG}" "${RELEASE_SIG_NAME}" \
  "release packet"
verify_signature "${DECISION}" "${DECISION_SIG}" "${DECISION_SIG_NAME}" \
  "phase decision"

case "${PHASE}" in
  cutover)
    python3 "${CHAIN_VERIFIER}" cutover-preinit "${AUTHORITY_BUNDLE}" \
      "${EXPECTED_SIGNER}" \
      --release "${RELEASE}" --release-sig "${RELEASE_SIG}" \
      --decision "${DECISION}" --decision-sig "${DECISION_SIG}" \
      --config-policy "${CONFIG_POLICY}" \
      --tool-root "${ROOT}" \
      >/dev/null || die "CUTOVER predecessor authority bundle did not verify"
    ;;
  open-beta)
    python3 "${CHAIN_VERIFIER}" open-preinit "${AUTHORITY_BUNDLE}" \
      "${EXPECTED_SIGNER}" "${EXPECTED_TRANSITION_SIGNER}" \
      --release "${RELEASE}" --release-sig "${RELEASE_SIG}" \
      --decision "${DECISION}" --decision-sig "${DECISION_SIG}" \
      --final "${AUTHORITY_BUNDLE}/FINAL-CHECKPOINT.json" \
      --final-sig "${AUTHORITY_BUNDLE}/FINAL-CHECKPOINT.json.sig" \
      --config-policy "${CONFIG_POLICY}" \
      --tool-root "${ROOT}" \
      --final-template "${FINAL_TEMPLATE}" --open-template "${OPEN_TEMPLATE}" \
      --adoption-template "${ADOPTION_TEMPLATE}" \
      >/dev/null || die "OPEN predecessor authority bundle did not verify"
    ;;
esac

RELEASE_SHA=$(sha256_file "${RELEASE}")
RELEASE_SIG_SHA=$(sha256_file "${RELEASE_SIG}")
DECISION_SHA=$(sha256_file "${DECISION}")
jq -e '.schema == "zerone-2-release-packet-v2" and .chain_id == "zerone-2"' \
  "${RELEASE}" >/dev/null || die "release root has the wrong schema or chain ID"
DEADLINE=$(jq -er '.authorization_semantics.initiation_deadline' "${DECISION}")
DEADLINE_EPOCH=$(canonical_utc_epoch "${DEADLINE}") || \
  die "decision deadline is not a real canonical UTC second"
DEADLINE_NANOSECONDS=$((10#${DEADLINE_EPOCH} * 1000000000))
BROADCAST_NOT_AFTER=$(jq -er \
  '.authorization_semantics.broadcast_not_after' "${DECISION}")
BROADCAST_NOT_AFTER_EPOCH=$(canonical_utc_epoch "${BROADCAST_NOT_AFTER}") || \
  die "decision broadcast-not-after is not a real canonical UTC second"
INCLUSION_MARGIN=$(jq -er \
  '.authorization_semantics.minimum_inclusion_margin_seconds' "${DECISION}")
[[ "${INCLUSION_MARGIN}" =~ ^[1-9][0-9]*$ ]] && \
  [ "$((DEADLINE_EPOCH - BROADCAST_NOT_AFTER_EPOCH))" -ge \
    "$((10#${INCLUSION_MARGIN}))" ] || \
  die "decision broadcast cutoff lacks its signed inclusion margin"
NOW_EPOCH=$(date -u '+%s')
[ "${NOW_EPOCH}" -gt "${BROADCAST_NOT_AFTER_EPOCH}" ] && \
  die "signed broadcast cutoff passed before submission"

case "${PHASE}" in
  cutover)
    jq -e \
      --arg release "${RELEASE_SHA}" --arg release_sig "${RELEASE_SIG_SHA}" '
        .schema == "zerone-2-cutover-decision-v1" and
        .decision == "GO" and
        .release_packet_sha256 == $release and
        .release_packet_detached_signature_sha256 == $release_sig and
        .successor_commitment_transaction.chain_id == "zerone-1" and
        .successor_commitment_transaction.message_type ==
          "/cosmos.bank.v1beta1.MsgSend" and
        .successor_commitment_transaction.recipient_equals_sender == true and
        .successor_commitment_transaction.amount == "1uzrn" and
        .successor_commitment_transaction.fee == "200000uzrn" and
        .successor_commitment_transaction.gas_limit == "200000" and
        .successor_commitment_transaction.must_not_reference_cutover_payload == true
      ' "${DECISION}" >/dev/null || die "CUTOVER decision does not match release/scope"
    TX_PATH=.successor_commitment_transaction
    COMPONENT=zerone_1_halt
    EXPECTED_CHAIN_ID=zerone-1
    EXPECTED_RPC_NODE_ID=$(jq -er '.predecessor.trusted_rpc_node_id' "${RELEASE}")
    TRUSTED_BLOCK_HEIGHT=$(jq -er '.predecessor.trusted_block.height' "${RELEASE}")
    TRUSTED_BLOCK_HASH=$(jq -er '.predecessor.trusted_block.block_id_hash' "${RELEASE}")
    TRUSTED_BLOCK_APP_HASH=$(jq -er '.predecessor.trusted_block.app_hash' "${RELEASE}")
    F=$(jq -er '.checkpoint_plan.checkpoint_state_height' "${DECISION}")
    A=$(jq -er '.checkpoint_plan.final_committed_anchor_height' "${DECISION}")
    H=$(jq -er '.checkpoint_plan.halt_trigger_height' "${DECISION}")
    for height in "${F}" "${A}" "${H}"; do
      [[ "${height}" =~ ^[1-9][0-9]{0,17}$ ]] || \
        die "CUTOVER F/A/H must be positive decimal strings in range"
    done
    [ "$((10#${F} + 1))" -eq "$((10#${A}))" ] && \
      [ "$((10#${A} + 1))" -eq "$((10#${H}))" ] || \
      die "CUTOVER must satisfy A=F+1 and H=A+1"
    SUCCESSOR_CHAIN=$(jq -er '.chain_id' "${RELEASE}")
    SUCCESSOR_GENESIS=$(jq -er '.genesis.sha256' "${RELEASE}")
    [[ "${SUCCESSOR_GENESIS}" =~ ^[0-9a-f]{64}$ ]] || \
      die "release successor genesis hash is malformed"
    EXPECTED_MEMO="successor_chain_id=${SUCCESSOR_CHAIN};successor_genesis_sha256=${SUCCESSOR_GENESIS};checkpoint_state_height=${F};final_committed_height=${A};halt_trigger_height=${H}"
    MINIMUM_HALT_LEAD=$(jq -er \
      '.authorization_semantics.minimum_halt_lead_blocks' "${DECISION}")
    ;;
  open-beta)
    jq -e \
      --arg release "${RELEASE_SHA}" --arg release_sig "${RELEASE_SIG_SHA}" '
        .schema == "zerone-2-open-beta-decision-v1" and
        .decision == "GO" and
        .release_packet.sha256 == $release and
        .release_packet.detached_signature_sha256 == $release_sig and
        .history_link_transaction.chain_id == "zerone-2" and
        .history_link_transaction.message_type ==
          "/cosmos.bank.v1beta1.MsgSend" and
        .history_link_transaction.recipient_equals_sender == true and
        .history_link_transaction.amount == "1uzrn" and
        .history_link_transaction.fee == "200000uzrn" and
        .history_link_transaction.gas_limit == "200000" and
        .history_link_transaction.must_not_reference_open_beta_payload == true
      ' "${DECISION}" >/dev/null || die "OPEN-BETA decision does not match release/scope"
    TX_PATH=.history_link_transaction
    COMPONENT=zerone_2_runtime
    EXPECTED_CHAIN_ID=zerone-2
    EXPECTED_RPC_NODE_ID=$(jq -er '.public_identities.validator_node_id' "${RELEASE}")
    TRUSTED_BLOCK_HEIGHT=1
    TRUSTED_BLOCK_HASH=$(jq -er \
      '.first_committed_block.block_id_hash' \
      "${DARK_INIT_SNAPSHOT}")
    TRUSTED_BLOCK_APP_HASH=$(jq -er \
      '.first_committed_block.app_hash' \
      "${DARK_INIT_SNAPSHOT}")
    [ "$(sha256_file "${DARK_INIT_SNAPSHOT}")" = \
      "$(jq -er '.dark_start_initiation_evidence.sha256' "${DECISION}")" ] || \
      die "snapshotted DARK initiation evidence differs from OPEN authority"
    FINAL_CHECKPOINT_SHA=$(jq -er '.final_checkpoint.sha256' "${DECISION}")
    [[ "${FINAL_CHECKPOINT_SHA}" =~ ^[0-9a-f]{64}$ ]] || \
      die "OPEN-BETA final-checkpoint hash is malformed"
    EXPECTED_MEMO="zerone_1_final_checkpoint_sha256=${FINAL_CHECKPOINT_SHA}"
    ;;
esac

[[ "${EXPECTED_RPC_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] || \
  die "signed transaction RPC node ID is malformed"
[[ "${TRUSTED_BLOCK_HEIGHT}" =~ ^[1-9][0-9]{0,17}$ ]] || \
  die "signed trusted-block height is malformed"
[[ "${TRUSTED_BLOCK_HASH}" =~ ^[0-9A-F]{64}$ ]] || \
  die "signed trusted block ID is malformed"
[[ "${TRUSTED_BLOCK_APP_HASH}" =~ ^[0-9A-F]{64}$ ]] || \
  die "signed trusted-block AppHash is malformed"

EXPECTED_TIMEOUT_HEIGHT=$(jq -er "${TX_PATH}.timeout_height" "${DECISION}")
[[ "${EXPECTED_TIMEOUT_HEIGHT}" =~ ^[1-9][0-9]{0,17}$ ]] || \
  die "signed transaction timeout height is malformed"

DECISION_MEMO=$(jq -er "${TX_PATH}.memo" "${DECISION}")
[ "${DECISION_MEMO}" = "${EXPECTED_MEMO}" ] || \
  die "signed transaction memo differs from immutable phase values"
SENDER=$(jq -er "${TX_PATH}.sender" "${DECISION}")
[ -n "${SENDER}" ] || die "signed transaction sender is empty"

BINARY_SHA=$(sha256_file "${BINARY}")
EXPECTED_BINARY_SHA=$(jq -er ".components.${COMPONENT}.binary_sha256" "${RELEASE}")
[ "${BINARY_SHA}" = "${EXPECTED_BINARY_SHA}" ] || \
  die "release binary hash differs from the signed release packet"
EXPECTED_RAW_SHA=$(jq -er "${TX_PATH}.signed_tx_bytes_sha256" "${DECISION}")
EXPECTED_TX_HASH=$(jq -er "${TX_PATH}.expected_transaction_hash" "${DECISION}")
[[ "${EXPECTED_RAW_SHA}" =~ ^[0-9a-f]{64}$ ]] || die "signed TxRaw SHA-256 is malformed"
[[ "${EXPECTED_TX_HASH}" =~ ^[0-9A-F]{64}$ ]] || die "expected tx hash is malformed"

ENCODED_TX=$("${BINARY}" tx encode "${TX_FILE}") || die "release binary could not encode signed transaction"
ENCODED_TX=$(printf '%s' "${ENCODED_TX}" | tr -d '\r\n')
[[ "${ENCODED_TX}" =~ ^[A-Za-z0-9+/]+={0,2}$ ]] || \
  die "tx encode did not return one base64 TxRaw value"
if ! printf '%s' "${ENCODED_TX}" | base64 --decode > "${TMP}/tx.raw" 2>/dev/null; then
  printf '%s' "${ENCODED_TX}" | base64 -D > "${TMP}/tx.raw" 2>/dev/null || \
    die "could not decode encoded TxRaw bytes"
fi
ACTUAL_RAW_SHA=$(sha256_file "${TMP}/tx.raw")
ACTUAL_TX_HASH=$(printf '%s' "${ACTUAL_RAW_SHA}" | tr '[:lower:]' '[:upper:]')
[ "${ACTUAL_RAW_SHA}" = "${EXPECTED_RAW_SHA}" ] || \
  die "encoded TxRaw bytes differ from the signed SHA-256"
[ "${ACTUAL_TX_HASH}" = "${EXPECTED_TX_HASH}" ] || \
  die "encoded TxRaw Comet hash differs from the signed expected hash"

DECODED_TX=$("${BINARY}" tx decode "${ENCODED_TX}" --output json) || \
  die "release binary could not decode its encoded TxRaw bytes"
jq -e \
  --arg sender "${SENDER}" --arg memo "${EXPECTED_MEMO}" \
  --arg timeout_height "${EXPECTED_TIMEOUT_HEIGHT}" \
  --arg decision_sha "${DECISION_SHA}" '
    (.body.messages | length) == 1 and
    .body.messages[0]["@type"] == "/cosmos.bank.v1beta1.MsgSend" and
    .body.messages[0].from_address == $sender and
    .body.messages[0].to_address == $sender and
    .body.messages[0].amount == [{denom:"uzrn",amount:"1"}] and
    .body.memo == $memo and
    .body.timeout_height == $timeout_height and
    ((.body.extension_options // []) | length) == 0 and
    ((.body.non_critical_extension_options // []) | length) == 0 and
    (.auth_info.signer_infos | length) == 1 and
    .auth_info.fee.amount == [{denom:"uzrn",amount:"200000"}] and
    .auth_info.fee.gas_limit == "200000" and
    (.auth_info.fee.payer // "") == "" and
    (.auth_info.fee.granter // "") == "" and
    (.auth_info.tip // null) == null and
    (.signatures | length) == 1 and
    ((tostring | ascii_downcase) | contains($decision_sha | ascii_downcase) | not)
  ' <<<"${DECODED_TX}" >/dev/null || \
  die "decoded TxRaw semantics differ from the signed phase contract"

if [ "${MODE}" = check ]; then
  printf '%s\n' "${EXPECTED_TX_HASH}"
  exit 0
fi

verify_trusted_rpc_block() {
  local response
  response=$(curl -fsS --max-time 5 \
    "${RPC_URL%/}/block?height=${TRUSTED_BLOCK_HEIGHT}") || \
    die "private RPC trusted-block query failed"
  jq -e --arg chain "${EXPECTED_CHAIN_ID}" \
    --arg height "${TRUSTED_BLOCK_HEIGHT}" \
    --arg block "${TRUSTED_BLOCK_HASH}" \
    --arg app "${TRUSTED_BLOCK_APP_HASH}" '
      .result.block_id.hash == $block and
      .result.block.header.chain_id == $chain and
      .result.block.header.height == $height and
      .result.block.header.app_hash == $app
    ' <<<"${response}" >/dev/null || \
    die "private RPC differs from the signed trusted-chain block"
}

STATUS_RESPONSE=$(curl -fsS --max-time 5 "${RPC_URL%/}/status") || \
  die "private RPC status check failed"
CURRENT_HEIGHT=$(jq -er --arg chain "${EXPECTED_CHAIN_ID}" \
  --arg node "${EXPECTED_RPC_NODE_ID}" '
  select(.result.node_info.network == $chain) |
  select((.result.node_info.id | ascii_downcase) == ($node | ascii_downcase)) |
  select(.result.sync_info.catching_up == false) |
  .result.sync_info.latest_block_height |
  select(test("^[1-9][0-9]*$"))
' <<<"${STATUS_RESPONSE}") || \
  die "private RPC has the wrong signed node/chain identity, is catching up, or has no positive height"
verify_trusted_rpc_block
[ "$((10#${CURRENT_HEIGHT}))" -lt "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
  die "private chain has reached the signed transaction timeout height"
if [ "${PHASE}" = cutover ]; then
  [ "$((10#${CURRENT_HEIGHT} + MINIMUM_HALT_LEAD))" -le "$((10#${F}))" ] || \
    die "live zerone-1 height no longer preserves the signed halt lead"
fi
NOW_EPOCH=$(date -u '+%s')
[ "${NOW_EPOCH}" -gt "${BROADCAST_NOT_AFTER_EPOCH}" ] && \
  die "signed broadcast cutoff passed before the exact raw-byte broadcast"

FINAL_STATUS_RESPONSE=$(curl -fsS --max-time 5 "${RPC_URL%/}/status") || \
  die "final private RPC status recheck failed"
FINAL_HEIGHT=$(jq -er --arg chain "${EXPECTED_CHAIN_ID}" \
  --arg node "${EXPECTED_RPC_NODE_ID}" '
  select(.result.node_info.network == $chain) |
  select((.result.node_info.id | ascii_downcase) == ($node | ascii_downcase)) |
  select(.result.sync_info.catching_up == false) |
  .result.sync_info.latest_block_height |
  select(test("^[1-9][0-9]*$"))
' <<<"${FINAL_STATUS_RESPONSE}") || \
  die "final private RPC identity/height recheck failed"
verify_trusted_rpc_block
[ "$((10#${FINAL_HEIGHT}))" -lt "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
  die "private chain reached the timeout height before broadcast"
if [ "${PHASE}" = cutover ]; then
  [ "$((10#${FINAL_HEIGHT} + MINIMUM_HALT_LEAD))" -le "$((10#${F}))" ] || \
    die "final zerone-1 height no longer preserves the signed halt lead"
fi

REQUEST=$(jq -cn --arg tx "${ENCODED_TX}" \
  '{jsonrpc:"2.0",id:1,method:"broadcast_tx_sync",params:{tx:$tx}}')
RESPONSE=$(curl -fsS --max-time 15 -H 'Content-Type: application/json' \
  --data-binary "${REQUEST}" "${RPC_URL}") || die "private RPC broadcast failed"
jq -e --arg hash "${EXPECTED_TX_HASH}" '
  .error == null and
  ((.result.code | tonumber) == 0) and
  ((.result.hash | ascii_upcase) == $hash)
' <<<"${RESPONSE}" >/dev/null || die "private RPC rejected or returned the wrong tx hash"

while :; do
  TX_QUERY=$(curl -fsS --max-time 5 \
    "${RPC_URL%/}/tx?hash=0x${EXPECTED_TX_HASH}&prove=false" 2>/dev/null || true)
  if jq -e --arg hash "${EXPECTED_TX_HASH}" --arg tx "${ENCODED_TX}" '
    .error == null and
    ((.result.hash | ascii_upcase) == $hash) and
    .result.tx == $tx and
    ((.result.tx_result.code | tonumber) == 0) and
    (.result.height | test("^[1-9][0-9]*$"))
  ' <<<"${TX_QUERY}" >/dev/null 2>&1; then
    COMMITTED_HEIGHT=$(jq -er '.result.height' <<<"${TX_QUERY}")
    [ "$((10#${COMMITTED_HEIGHT}))" -le \
      "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
      die "transaction committed after its signed timeout height"
    BLOCK_QUERY=$(curl -fsS --max-time 5 \
      "${RPC_URL%/}/block?height=${COMMITTED_HEIGHT}") || \
      die "could not query the transaction commit block"
    COMMITTED_TIME=$(jq -er --arg height "${COMMITTED_HEIGHT}" '
      select(.result.block.header.height == $height) |
      .result.block.header.time
    ' <<<"${BLOCK_QUERY}") || die "commit block response is malformed"
    COMMITTED_NANOSECONDS=$(canonical_utc_nanoseconds "${COMMITTED_TIME}") || \
      die "commit block time is not canonical RFC3339 UTC with at most nanoseconds"
    [ "${COMMITTED_NANOSECONDS}" -le "${DEADLINE_NANOSECONDS}" ] || \
      die "transaction committed after the signed initiation deadline"
    if [ "${PHASE}" = cutover ]; then
      [ "$((10#${COMMITTED_HEIGHT} + MINIMUM_HALT_LEAD))" -le \
        "$((10#${F}))" ] || \
        die "committed CUTOVER transaction no longer preserves the signed halt lead"
    fi
    printf '%s\n' "${EXPECTED_TX_HASH}"
    exit 0
  fi
  WAIT_STATUS=$(curl -fsS --max-time 5 "${RPC_URL%/}/status") || \
    die "private RPC status failed while waiting for commit"
  WAIT_HEIGHT=$(jq -er --arg chain "${EXPECTED_CHAIN_ID}" \
    --arg node "${EXPECTED_RPC_NODE_ID}" '
    select(.result.node_info.network == $chain) |
    select((.result.node_info.id | ascii_downcase) == ($node | ascii_downcase)) |
    .result.sync_info.latest_block_height |
    select(test("^[1-9][0-9]*$"))
  ' <<<"${WAIT_STATUS}") || die "commit-wait status response is malformed"
  [ "$((10#${WAIT_HEIGHT}))" -lt "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
    die "transaction did not commit before its enforceable timeout height"
  sleep 2
done
