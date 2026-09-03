#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'zerone-2-bootstrap-tx-broadcast: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  scripts/zerone-2-bootstrap-tx-broadcast.sh [--check] \
    RELEASE.json RELEASE.json.sig DARK-START.json DARK-START.json.sig \
    DARK-START-INITIATION-EVIDENCE.json DARK-START-INITIATION-EVIDENCE.json.sig \
    operator-onboarding|custom-validator-registration RELEASE_BINARY \
    PRIVATE_RPC_URL EXPECTED_MAIN_FINGERPRINT AUTHORITY_BUNDLE_DIRECTORY

The bundle must contain the exact two pre-signed TxRaw JSON files declared by
DARK-START. The wrapper verifies the signed predecessor chain, release binary,
raw bytes, decoded message/fee/timeout semantics, private successor identity,
the onboarding identity proof, and block-1 anchor. The validator-registration
transaction additionally proves that the exact onboarding transaction
committed successfully first. --check runs the same live state, deadline, and
signature gates but returns the expected hash without submitting the bytes.
USAGE
  exit 2
}

MODE=broadcast
if [ "${1:-}" = --check ]; then
  MODE=check
  shift
fi
[ "$#" -eq 11 ] || usage

RELEASE=$1
RELEASE_SIG=$2
DARK=$3
DARK_SIG=$4
DARK_INIT=$5
DARK_INIT_SIG=$6
TRANSACTION_ROLE=$7
BINARY=$8
RPC_URL=$9
EXPECTED_SIGNER=${10}
AUTHORITY_BUNDLE=${11}

case "${TRANSACTION_ROLE}" in
  operator-onboarding)
    CONTRACT_KEY=operator_onboarding
    TX_FILENAME=ZERONE-2-ONBOARD-SIGNED-TX.json
    ;;
  custom-validator-registration)
    CONTRACT_KEY=custom_validator_registration
    TX_FILENAME=ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json
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
for pair in \
  "${RELEASE}|release packet" "${RELEASE_SIG}|release signature" \
  "${DARK}|DARK-START decision" "${DARK_SIG}|DARK-START signature" \
  "${DARK_INIT}|DARK-START initiation evidence" \
  "${DARK_INIT_SIG}|DARK-START initiation signature" \
  "${BINARY}|release binary" \
  "${AUTHORITY_BUNDLE}/${TX_FILENAME}|signed bootstrap transaction"; do
  require_regular "${pair%%|*}" "${pair##*|}"
done
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
CHAIN_VERIFIER="${ROOT}/deploy/verify-authority-chain.py"
CONFIG_POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
require_regular "${CHAIN_VERIFIER}" "authority-chain verifier"
require_regular "${CONFIG_POLICY}" "Fly config policy"
for command in jq gpg curl python3; do
  command -v "${command}" >/dev/null 2>&1 || die "${command} is required"
done

TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-bootstrap-tx.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
install -m 0600 "${RELEASE}" "${TMP}/RELEASE-PACKET.json"
install -m 0600 "${RELEASE_SIG}" "${TMP}/RELEASE-PACKET.json.sig"
install -m 0600 "${DARK}" "${TMP}/DARK-START-DECISION.json"
install -m 0600 "${DARK_SIG}" "${TMP}/DARK-START-DECISION.json.sig"
install -m 0600 "${DARK_INIT}" "${TMP}/DARK-START-INITIATION-EVIDENCE.json"
install -m 0600 "${DARK_INIT_SIG}" \
  "${TMP}/DARK-START-INITIATION-EVIDENCE.json.sig"
install -m 0600 "${AUTHORITY_BUNDLE}/${TX_FILENAME}" "${TMP}/${TX_FILENAME}"
install -m 0700 "${BINARY}" "${TMP}/zeroned"
install -m 0700 "${CHAIN_VERIFIER}" "${TMP}/verify-authority-chain.py"
install -m 0700 "${CONFIG_POLICY}" "${TMP}/validate-fly-phase-config.py"
RELEASE="${TMP}/RELEASE-PACKET.json"
RELEASE_SIG="${TMP}/RELEASE-PACKET.json.sig"
DARK="${TMP}/DARK-START-DECISION.json"
DARK_SIG="${TMP}/DARK-START-DECISION.json.sig"
DARK_INIT="${TMP}/DARK-START-INITIATION-EVIDENCE.json"
DARK_INIT_SIG="${TMP}/DARK-START-INITIATION-EVIDENCE.json.sig"
TX_FILE="${TMP}/${TX_FILENAME}"
BINARY="${TMP}/zeroned"
CHAIN_VERIFIER="${TMP}/verify-authority-chain.py"
CONFIG_POLICY="${TMP}/validate-fly-phase-config.py"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
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

python3 "${CHAIN_VERIFIER}" dark-registration-preinit "${AUTHORITY_BUNDLE}" \
  "${EXPECTED_SIGNER}" \
  --release "${RELEASE}" --release-sig "${RELEASE_SIG}" \
  --decision "${DARK}" --decision-sig "${DARK_SIG}" \
  --initiation "${DARK_INIT}" --initiation-sig "${DARK_INIT_SIG}" \
  --config-policy "${CONFIG_POLICY}" --tool-root "${ROOT}" >/dev/null || \
  die "DARK registration predecessor authority bundle did not verify"

TX_PATH=".private_bootstrap_transactions.${CONTRACT_KEY}"
jq -e '
  .private_bootstrap_transactions.broadcast_order == [
    "operator_onboarding", "custom_validator_registration"
  ] and
  .private_bootstrap_transactions.operator_onboarding.signer_sequence == "0" and
  .private_bootstrap_transactions.custom_validator_registration.signer_sequence == "1" and
  (.private_bootstrap_transactions.operator_onboarding.timeout_height | tonumber) <
    (.private_bootstrap_transactions.custom_validator_registration.timeout_height | tonumber)
' "${DARK}" >/dev/null || die "DARK bootstrap order/sequence/timeout contract changed"
DECLARED_FILENAME=$(jq -er "${TX_PATH}.filename" "${DARK}")
[ "${DECLARED_FILENAME}" = "${TX_FILENAME}" ] || \
  die "signed bootstrap transaction filename differs from the fixed role"
EXPECTED_RAW_SHA=$(jq -er "${TX_PATH}.signed_tx_bytes_sha256" "${DARK}")
EXPECTED_TX_HASH=$(jq -er "${TX_PATH}.expected_transaction_hash" "${DARK}")
EXPECTED_TIMEOUT_HEIGHT=$(jq -er "${TX_PATH}.timeout_height" "${DARK}")
EXPECTED_SEQUENCE=$(jq -er "${TX_PATH}.signer_sequence" "${DARK}")
EXPECTED_MEMO=$(jq -er "${TX_PATH}.memo" "${DARK}")
[[ "${EXPECTED_RAW_SHA}" =~ ^[0-9a-f]{64}$ ]] || die "signed TxRaw hash is malformed"
[[ "${EXPECTED_TX_HASH}" =~ ^[0-9A-F]{64}$ ]] || die "transaction hash is malformed"
[[ "${EXPECTED_TIMEOUT_HEIGHT}" =~ ^[1-9][0-9]{0,17}$ ]] || \
  die "transaction timeout height is malformed"
[[ "${EXPECTED_SEQUENCE}" =~ ^(0|[1-9][0-9]{0,17})$ ]] || \
  die "transaction signer sequence is malformed"

BINARY_SHA=$(sha256_file "${BINARY}")
EXPECTED_BINARY_SHA=$(jq -er '.components.zerone_2_runtime.binary_sha256' "${RELEASE}")
[ "${BINARY_SHA}" = "${EXPECTED_BINARY_SHA}" ] || \
  die "release binary hash differs from RELEASE"
ENCODED_TX=$("${BINARY}" tx encode "${TX_FILE}") || \
  die "release binary could not encode the signed transaction"
ENCODED_TX=$(printf '%s' "${ENCODED_TX}" | tr -d '\r\n')
[[ "${ENCODED_TX}" =~ ^[A-Za-z0-9+/]+={0,2}$ ]] || \
  die "tx encode did not return one base64 TxRaw value"
if ! printf '%s' "${ENCODED_TX}" | base64 --decode > "${TMP}/tx.raw" 2>/dev/null; then
  printf '%s' "${ENCODED_TX}" | base64 -D > "${TMP}/tx.raw" 2>/dev/null || \
    die "could not decode encoded TxRaw bytes"
fi
ACTUAL_RAW_SHA=$(sha256_file "${TMP}/tx.raw")
[ "${ACTUAL_RAW_SHA}" = "${EXPECTED_RAW_SHA}" ] || \
  die "encoded TxRaw bytes differ from DARK-START"
[ "$(printf '%s' "${ACTUAL_RAW_SHA}" | tr '[:lower:]' '[:upper:]')" = \
  "${EXPECTED_TX_HASH}" ] || die "encoded TxRaw Comet hash differs from DARK-START"
DECODED_TX=$("${BINARY}" tx decode "${ENCODED_TX}" --output json) || \
  die "release binary could not decode its encoded TxRaw bytes"
DARK_SHA=$(sha256_file "${DARK}")

# shellcheck disable=SC2016 # jq variables are populated with --arg below.
COMMON_JQ='
  (.body.messages | length) == 1 and
  .body.memo == $memo and
  .body.timeout_height == $timeout and
  ((.body.extension_options // []) | length) == 0 and
  ((.body.non_critical_extension_options // []) | length) == 0 and
  (.auth_info.signer_infos | length) == 1 and
  .auth_info.signer_infos[0].public_key["@type"] ==
    "/cosmos.crypto.secp256k1.PubKey" and
  (.auth_info.signer_infos[0].public_key.key |
    test("^[A-Za-z0-9+/]+={0,2}$")) and
  .auth_info.signer_infos[0].mode_info == {
    single:{mode:"SIGN_MODE_DIRECT"}
  } and
  .auth_info.signer_infos[0].sequence == $sequence and
  .auth_info.fee.amount == [{denom:"uzrn",amount:"200000"}] and
  .auth_info.fee.gas_limit == "200000" and
  (.auth_info.fee.payer // "") == "" and
  (.auth_info.fee.granter // "") == "" and
  (.auth_info.tip // null) == null and
  (.signatures | length) == 1 and
  ((tostring | ascii_downcase) | contains($dark_sha | ascii_downcase) | not)'
case "${TRANSACTION_ROLE}" in
  operator-onboarding)
    jq -e --arg memo "${EXPECTED_MEMO}" --arg timeout "${EXPECTED_TIMEOUT_HEIGHT}" \
      --arg sequence "${EXPECTED_SEQUENCE}" \
      --arg dark_sha "${DARK_SHA}" \
      --arg sender "$(jq -er "${TX_PATH}.sender" "${DARK}")" \
      --arg did "$(jq -er "${TX_PATH}.did" "${DARK}")" \
      --arg public_key "$(jq -er "${TX_PATH}.public_key" "${DARK}")" \
      --arg identity_proof "$(jq -er "${TX_PATH}.identity_proof_signature" "${DARK}")" \
      "${COMMON_JQ} and
       .body.messages[0] == {
         \"@type\":\"/zerone.auth.v1.MsgRegisterAccount\",
         sender:\$sender,did:\$did,public_key:\$public_key,account_type:\"human\",
         operational_key_hash:\"\",metadata:\"\",
         identity_proof_signature:\$identity_proof
       }" <<<"${DECODED_TX}" >/dev/null || \
      die "decoded onboarding TxRaw differs from the signed DARK contract"
    ;;
  custom-validator-registration)
    jq -e --arg memo "${EXPECTED_MEMO}" --arg timeout "${EXPECTED_TIMEOUT_HEIGHT}" \
      --arg sequence "${EXPECTED_SEQUENCE}" \
      --arg dark_sha "${DARK_SHA}" \
      --arg operator "$(jq -er "${TX_PATH}.operator" "${DARK}")" \
      --arg pubkey "$(jq -er "${TX_PATH}.consensus_pubkey" "${DARK}")" \
      --arg did "$(jq -er "${TX_PATH}.did" "${DARK}")" \
      "${COMMON_JQ} and
       .body.messages[0] == {
         \"@type\":\"/zerone.staking.v1.MsgRegisterValidator\",
         operator:\$operator,consensus_pubkey:\$pubkey,did:\$did,
         moniker:\"zerone-2-custodian\",self_delegation:\"111000000\",
         commission_bps:\"500\",website:\"\",
         details:\"One publicly disclosed custodial validator\"
       }" <<<"${DECODED_TX}" >/dev/null || \
      die "decoded custom-validator TxRaw differs from the signed DARK contract"
    ;;
esac
SIGNED_SEQUENCE=$(jq -er '
  def canonical_uint64_text:
    type == "string" and test("^(0|[1-9][0-9]{0,19})$") and
    (length < 20 or . <= "18446744073709551615");
  .auth_info.signer_infos[0].sequence | select(canonical_uint64_text)' \
  <<<"${DECODED_TX}") || die "decoded bootstrap TxRaw signer sequence is malformed"
[ "${SIGNED_SEQUENCE}" = "${EXPECTED_SEQUENCE}" ] || \
  die "decoded bootstrap TxRaw signer sequence differs from DARK-START"
case "${TRANSACTION_ROLE}" in
  operator-onboarding)
    SIGNED_SENDER=$(jq -er "${TX_PATH}.sender" "${DARK}")
    ;;
  custom-validator-registration)
    SIGNED_SENDER=$(jq -er "${TX_PATH}.operator" "${DARK}")
    ;;
esac
[ -n "${SIGNED_SENDER}" ] || die "signed bootstrap sender is empty"

# Cosmos account implementations wrap BaseAccount in several stable shapes.
# Walk only those known wrappers and require exactly one canonical candidate;
# this rejects malformed, ambiguous, and lossy numeric sequence responses.
verify_live_sender_sequence() {
  local response live_sequence
  response=$("${BINARY}" query auth account "${SIGNED_SENDER}" \
    --node "${RPC_URL%/}" --output json 2>"${TMP}/account-query.stderr") || \
    die "release binary could not query the live bootstrap sender BaseAccount"
  live_sequence=$(jq -er --arg sender "${SIGNED_SENDER}" '
    def canonical_uint64_text:
      type == "string" and test("^(0|[1-9][0-9]{0,19})$") and
      (length < 20 or . <= "18446744073709551615");
    def base_accounts:
      . as $node |
      if ($node | type) != "object" then empty
      else
        (if (($node.address? | type) == "string") and
            (($node.sequence? | type) == "string") and
            (($node | has("account_number")) or
             ($node | has("accountNumber"))) and
            ((($node | has("account_number")) | not) or
             ($node.account_number | canonical_uint64_text)) and
            ((($node | has("accountNumber")) | not) or
             ($node.accountNumber | canonical_uint64_text)) and
            (((($node | has("account_number")) and
               ($node | has("accountNumber"))) | not) or
             ($node.account_number == $node.accountNumber))
         then {address:$node.address, sequence:$node.sequence}
         else empty end),
        (["account", "base_account", "baseAccount",
          "base_vesting_account", "baseVestingAccount", "value"][] as $key |
         select(($node[$key]? | type) == "object") |
         $node[$key] | base_accounts)
      end;
    [base_accounts] as $accounts |
    select(($accounts | length) == 1) |
    $accounts[0] |
    select(.address == $sender) |
    .sequence |
    select(canonical_uint64_text)
  ' <<<"${response}") || \
    die "live bootstrap account response has no unique canonical BaseAccount"
  [ "${live_sequence}" = "${SIGNED_SEQUENCE}" ] || \
    die "live bootstrap sender sequence ${live_sequence} differs from signed TxRaw sequence ${SIGNED_SEQUENCE}"
}

# The authority-bundle verifier deliberately treats the release binary as
# signed data and therefore cannot import its consensus implementation. Run
# the release binary's fully offline verifier here so --check rejects a
# well-shaped but cryptographically invalid identity proof before a failed
# DeliverTx can consume the first bootstrap account sequence.
ONBOARD_PATH=.private_bootstrap_transactions.operator_onboarding
ONBOARD_CHAIN_ID=$(jq -er "${ONBOARD_PATH}.chain_id" "${DARK}")
ONBOARD_SENDER=$(jq -er "${ONBOARD_PATH}.sender" "${DARK}")
ONBOARD_DID=$(jq -er "${ONBOARD_PATH}.did" "${DARK}")
ONBOARD_PUBLIC_KEY=$(jq -er "${ONBOARD_PATH}.public_key" "${DARK}")
ONBOARD_ACCOUNT_TYPE=$(jq -er "${ONBOARD_PATH}.account_type" "${DARK}")
ONBOARD_METADATA=$(jq -er "${ONBOARD_PATH}.metadata" "${DARK}")
ONBOARD_PROOF_BASE64=$(jq -er "${ONBOARD_PATH}.identity_proof_signature" "${DARK}")
if ! printf '%s' "${ONBOARD_PROOF_BASE64}" | \
  base64 --decode > "${TMP}/identity-proof.bin" 2>/dev/null; then
  printf '%s' "${ONBOARD_PROOF_BASE64}" | \
    base64 -D > "${TMP}/identity-proof.bin" 2>/dev/null || \
    die "could not decode onboarding identity proof"
fi
[ "$(wc -c < "${TMP}/identity-proof.bin" | tr -d ' ')" = 64 ] || \
  die "onboarding identity proof is not 64 bytes"
ONBOARD_PROOF_HEX=$(od -An -v -tx1 "${TMP}/identity-proof.bin" | tr -d ' \n')
if ! PROOF_CHECK=$("${BINARY}" tx zerone_auth verify-registration-proof \
  "${ONBOARD_SENDER}" "${ONBOARD_DID}" "${ONBOARD_PUBLIC_KEY}" \
  "${ONBOARD_ACCOUNT_TYPE}" "${ONBOARD_PROOF_HEX}" \
  --chain-id "${ONBOARD_CHAIN_ID}" --metadata "${ONBOARD_METADATA}" 2>&1); then
  die "signed onboarding identity proof failed verification: ${PROOF_CHECK}"
fi

COMMIT_DEADLINE=$(jq -er '.authorization_semantics.registration_commit_deadline' \
  "${DARK}")
BROADCAST_NOT_AFTER=$(jq -er \
  '.authorization_semantics.registration_broadcast_not_after' "${DARK}")
INCLUSION_MARGIN=$(jq -er \
  '.authorization_semantics.minimum_registration_inclusion_margin_seconds' "${DARK}")
COMMIT_DEADLINE_EPOCH=$(canonical_utc_epoch "${COMMIT_DEADLINE}") || \
  die "registration commit deadline is not a canonical UTC second"
BROADCAST_NOT_AFTER_EPOCH=$(canonical_utc_epoch "${BROADCAST_NOT_AFTER}") || \
  die "registration broadcast cutoff is not a canonical UTC second"
[[ "${INCLUSION_MARGIN}" =~ ^[1-9][0-9]*$ ]] && \
  [ "$((COMMIT_DEADLINE_EPOCH - BROADCAST_NOT_AFTER_EPOCH))" -ge \
    "$((10#${INCLUSION_MARGIN}))" ] || \
  die "registration cutoff lacks its signed inclusion margin"
[ "$(date -u '+%s')" -le "${BROADCAST_NOT_AFTER_EPOCH}" ] || \
  die "signed registration broadcast cutoff has passed"

EXPECTED_NODE_ID=$(jq -er '.public_identities.validator_node_id' "${RELEASE}")
TRUSTED_HEIGHT=1
TRUSTED_BLOCK_HASH=$(jq -er '.first_committed_block.block_id_hash' "${DARK_INIT}")
TRUSTED_APP_HASH=$(jq -er '.first_committed_block.app_hash' "${DARK_INIT}")
STATUS=$(curl -fsS --max-time 5 "${RPC_URL%/}/status") || \
  die "private successor RPC status failed"
CURRENT_HEIGHT=$(jq -er --arg node "${EXPECTED_NODE_ID}" '
  select(.result.node_info.network == "zerone-2") |
  select((.result.node_info.id | ascii_downcase) == ($node | ascii_downcase)) |
  select(.result.sync_info.catching_up == false) |
  .result.sync_info.latest_block_height | select(test("^[1-9][0-9]*$"))
' <<<"${STATUS}") || die "private RPC is not the signed successor validator"
[ "$((10#${CURRENT_HEIGHT}))" -lt "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
  die "successor reached the signed transaction timeout height"
ANCHOR=$(curl -fsS --max-time 5 "${RPC_URL%/}/block?height=${TRUSTED_HEIGHT}") || \
  die "could not query signed successor block-1 anchor"
jq -e --arg hash "${TRUSTED_BLOCK_HASH}" --arg app "${TRUSTED_APP_HASH}" '
  .result.block_id.hash == $hash and
  .result.block.header.chain_id == "zerone-2" and
  .result.block.header.height == "1" and
  .result.block.header.app_hash == $app
' <<<"${ANCHOR}" >/dev/null || die "successor RPC differs from signed block-1 anchor"

# Verify the normal Cosmos TxRaw signature against the anchored successor's
# current account number and sequence before a failed DeliverTx can charge the
# account or strand the second pre-signed bootstrap transaction.
if ! SIGNATURE_CHECK=$("${BINARY}" tx validate-signatures "${TX_FILE}" \
  --chain-id zerone-2 --node "${RPC_URL%/}" --output json 2>&1); then
  die "signed bootstrap TxRaw failed pre-broadcast signature validation: ${SIGNATURE_CHECK}"
fi
verify_live_sender_sequence

if [ "${TRANSACTION_ROLE}" = custom-validator-registration ]; then
  ONBOARD_HASH=$(jq -er "${ONBOARD_PATH}.expected_transaction_hash" "${DARK}")
  ONBOARD_FILE=$(jq -er "${ONBOARD_PATH}.filename" "${DARK}")
  [ "${ONBOARD_FILE}" = ZERONE-2-ONBOARD-SIGNED-TX.json ] || \
    die "DARK onboarding filename changed"
  require_regular "${AUTHORITY_BUNDLE}/${ONBOARD_FILE}" "signed onboarding transaction"
  ONBOARD_ENCODED=$("${BINARY}" tx encode "${AUTHORITY_BUNDLE}/${ONBOARD_FILE}") || \
    die "could not encode predecessor onboarding transaction"
  ONBOARD_ENCODED=$(printf '%s' "${ONBOARD_ENCODED}" | tr -d '\r\n')
  ONBOARD_QUERY=$(curl -fsS --max-time 5 \
    "${RPC_URL%/}/tx?hash=0x${ONBOARD_HASH}&prove=false") || \
    die "exact onboarding transaction has not committed"
  jq -e --arg hash "${ONBOARD_HASH}" --arg tx "${ONBOARD_ENCODED}" '
    .error == null and (.result.hash | ascii_upcase) == $hash and
    .result.tx == $tx and ((.result.tx_result.code | tonumber) == 0) and
    (.result.height | test("^[1-9][0-9]*$"))
  ' <<<"${ONBOARD_QUERY}" >/dev/null || \
    die "custom-validator registration requires the exact successful onboarding first"
fi

if [ "${MODE}" = check ]; then
  printf '%s\n' "${EXPECTED_TX_HASH}"
  exit 0
fi

[ "$(date -u '+%s')" -le "${BROADCAST_NOT_AFTER_EPOCH}" ] || \
  die "registration cutoff passed before exact raw-byte broadcast"
verify_live_sender_sequence
REQUEST=$(jq -cn --arg tx "${ENCODED_TX}" \
  '{jsonrpc:"2.0",id:1,method:"broadcast_tx_sync",params:{tx:$tx}}')
RESPONSE=$(curl -fsS --max-time 15 -H 'Content-Type: application/json' \
  --data-binary "${REQUEST}" "${RPC_URL}") || die "private RPC broadcast failed"
jq -e --arg hash "${EXPECTED_TX_HASH}" '
  .error == null and ((.result.code | tonumber) == 0) and
  ((.result.hash | ascii_upcase) == $hash)
' <<<"${RESPONSE}" >/dev/null || die "private RPC rejected or returned the wrong hash"

while :; do
  QUERY=$(curl -fsS --max-time 5 \
    "${RPC_URL%/}/tx?hash=0x${EXPECTED_TX_HASH}&prove=false" 2>/dev/null || true)
  if jq -e --arg hash "${EXPECTED_TX_HASH}" --arg tx "${ENCODED_TX}" '
    .error == null and (.result.hash | ascii_upcase) == $hash and
    .result.tx == $tx and ((.result.tx_result.code | tonumber) == 0) and
    (.result.height | test("^[1-9][0-9]*$"))
  ' <<<"${QUERY}" >/dev/null 2>&1; then
    COMMITTED_HEIGHT=$(jq -er '.result.height' <<<"${QUERY}")
    [ "$((10#${COMMITTED_HEIGHT}))" -le "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
      die "bootstrap transaction committed after its signed timeout height"
    BLOCK=$(curl -fsS --max-time 5 \
      "${RPC_URL%/}/block?height=${COMMITTED_HEIGHT}") || die "commit block query failed"
    COMMITTED_TIME=$(jq -er --arg height "${COMMITTED_HEIGHT}" '
      select(.result.block.header.height == $height) | .result.block.header.time
    ' <<<"${BLOCK}") || die "commit block response is malformed"
    COMMITTED_NS=$(canonical_utc_nanoseconds "${COMMITTED_TIME}") || \
      die "commit time is not canonical RFC3339 UTC with at most nanoseconds"
    DEADLINE_NS=$((10#${COMMIT_DEADLINE_EPOCH} * 1000000000))
    [ "${COMMITTED_NS}" -le "${DEADLINE_NS}" ] || \
      die "bootstrap transaction committed after its signed deadline"
    printf '%s\n' "${EXPECTED_TX_HASH}"
    exit 0
  fi
  WAIT_STATUS=$(curl -fsS --max-time 5 "${RPC_URL%/}/status") || \
    die "successor status failed while waiting for commit"
  WAIT_HEIGHT=$(jq -er '
    select(.result.node_info.network == "zerone-2") |
    .result.sync_info.latest_block_height | select(test("^[1-9][0-9]*$"))
  ' <<<"${WAIT_STATUS}") || die "commit-wait status is malformed"
  [ "$((10#${WAIT_HEIGHT}))" -lt "$((10#${EXPECTED_TIMEOUT_HEIGHT}))" ] || \
    die "bootstrap transaction did not commit before timeout height"
  [ "$(date -u '+%s')" -le "${COMMIT_DEADLINE_EPOCH}" ] || \
    die "bootstrap transaction did not commit by its signed deadline"
  sleep 2
done
