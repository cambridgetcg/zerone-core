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

census_mutate() {
  local path=$1 mutation=$2
  python3 - "${path}" "${mutation}" <<'PY'
import hashlib
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
mutation = sys.argv[2]
report = json.loads(path.read_bytes())
if mutation == "self-hash":
    report["report_sha256"] = "0" * 64
elif mutation == "result-fail":
    report["result"] = "FAIL"
elif mutation == "checkpoint-height":
    report["evidence"]["height"] = "1000"
elif mutation == "checkpoint-app-hash":
    report["evidence"]["app_hash"] = "b" * 64
elif mutation == "source-commit":
    report["evidence"]["source_commit"] = "2" * 40
elif mutation == "arithmetic":
    report["census"]["liabilities_uzrn"] = "31"
elif mutation == "findings":
    report["census"]["findings"] = [
        {"code": "fixture", "key": "fixture", "detail": "fixture"}
    ]
elif mutation == "claimant-incomplete":
    report["census"]["claimant_root_complete"] = False
elif mutation == "claimant-root":
    report["census"]["claimant_root"] = "f" * 64
elif mutation == "multistore-root-drift":
    next(row for row in report["multistore"] if row["name"] == "bank")[
        "root_sha256"
    ] = "f" * 64
    next(row for row in report["stores"] if row["name"] == "bank")[
        "root_sha256"
    ] = "f" * 64
elif mutation == "module-identity":
    report["census"]["module_address"] = report["census"]["claims"][0]["claimant"]
    report["census"]["module_address_hex"] = "22" * 20
elif mutation == "extra-module-denom":
    report["census"]["module_balances"].insert(
        0, {"denom": "uother", "amount": "1"}
    )
elif mutation == "legacy-key-trusted":
    report["census"]["validators"][0]["legacy_consensus_pubkey_trusted"] = True
elif mutation == "validator-claim-mismatch":
    validator = report["census"]["validators"][0]
    validator["stored_delegated"] = "19"
    validator["computed_delegated"] = "19"
    validator["stored_total"] = "19"
    validator["computed_total"] = "19"
elif mutation == "sdk-link-absent":
    validator = report["census"]["validators"][0]
    validator["sdk_link"] = "absent"
    del validator["sdk_operator"]
elif mutation == "unbonding-mismatch":
    report["census"]["unbondings"][0]["amount"] = "9"
elif mutation == "reverse-mismatch":
    report["census"]["reverse_delegation_indexes"][0]["delegator"] = report[
        "census"
    ]["module_address"]
elif mutation == "hollow-tier":
    report["census"]["tier_configs"][0]["stored_digest"] = ""
elif mutation == "did-byte-ceiling":
    report["census"]["did_indexes"][0]["did"] = "d" * 129
elif mutation == "custom-leaf-ceiling":
    report["stores"][0]["leaf_count"] = "50001"
elif mutation == "custom-input-ceiling":
    report["stores"][0]["input_bytes"] = str((32 << 20) + 1)
elif mutation == "bank-leaf-ceiling":
    report["stores"][1]["leaf_count"] = "5000001"
elif mutation == "aggregate-input-ceiling":
    report["stores"][1]["input_bytes"] = str(1 << 30)
elif mutation == "sdk-row-ceiling":
    report["census"]["sdk_validators"] *= 25001
elif mutation == "unbound-multistore":
    report["stores"][1]["leaves_sha256"] = "f" * 64
elif mutation == "sentinel-count":
    report["census"]["custom_keyspace"][9]["leaf_count"] = 2
    report["stores"][0]["leaf_count"] = "14"
else:
    raise SystemExit(f"unknown census mutation: {mutation}")

if mutation != "self-hash":
    report["report_sha256"] = ""
    unsealed = json.dumps(
        report, separators=(",", ":"), ensure_ascii=False
    ).encode()
    report["report_sha256"] = hashlib.sha256(unsealed).hexdigest()
path.write_bytes(
    (json.dumps(report, separators=(",", ":"), ensure_ascii=False) + "\n").encode()
)
PY
}

rebind_census_final() {
  local bundle=$1 census_sha
  census_sha=$(sha256_file "${bundle}/CUSTOM-STAKING-CENSUS.json")
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/FINAL-CHECKPOINT.json" \
    '.artifacts.custom_staking_census_sha256 = $sha' \
    --arg sha "${census_sha}"
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

rebind_monitoring_chain() {
  local bundle=$1 rules_sha tests_sha manifest_sha
  rules_sha=$(sha256_file "${bundle}/MONITORING-RULES.json")
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/MONITORING-ALERT-TESTS.json" \
    '.rules_sha256 = $sha' --arg sha "${rules_sha}"
  tests_sha=$(sha256_file "${bundle}/MONITORING-ALERT-TESTS.json")
  # shellcheck disable=SC2016 # jq variables are not shell variables.
  canonical_mutate "${bundle}/MONITORING-ALERTS.json" \
    '.rules.sha256 = $rules | .alert_tests.sha256 = $tests' \
    --arg rules "${rules_sha}" --arg tests "${tests_sha}"
  manifest_sha=$(sha256_file "${bundle}/MONITORING-ALERTS.json")
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/RELEASE-PACKET.json" \
    '.monitoring_alerts_sha256 = $sha' --arg sha "${manifest_sha}"
}

rebind_archive_gateway_open() {
  local bundle=$1 config_sha dns_sha
  config_sha=$(sha256_file \
    "${bundle}/fly.zerone-1-archive-gateway.public.toml")
  # The fake signatures deliberately permit recomputing every OPEN-owned hash;
  # the verifier must still reject bytes that are not the RELEASE/FINAL render.
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/OPEN-BETA-DECISION.json" \
    '.deployment_configs.zerone_1_archive_gateway.sha256 = $sha' \
    --arg sha "${config_sha}"
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/DNS-CHANGE-MANIFEST.json" \
    '.records["archive.example"].config_sha256 = $sha' \
    --arg sha "${config_sha}"
  dns_sha=$(sha256_file "${bundle}/DNS-CHANGE-MANIFEST.json")
  # shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
  canonical_mutate "${bundle}/OPEN-BETA-DECISION.json" \
    '.public_coordinates.canonical_dns_change_manifest_sha256 = $sha' \
    --arg sha "${dns_sha}"
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
    expected_post_anchor=$(/usr/bin/python3 - \
      "$(flag_value --a-block-results "$@")" <<'PY'
import json
import pathlib
import re
import sys

value = json.loads(pathlib.Path(sys.argv[1]).read_bytes())["result"]["app_hash"]
if not isinstance(value, str) or not re.fullmatch(r"[0-9A-F]{64}", value):
    raise SystemExit(65)
print(value)
PY
    )
    [ "$(flag_value --expected-post-anchor-app-hash "$@")" = \
      "${expected_post_anchor}" ] || exit 65
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
# This expected value was generated independently with
# cosmossdk.io/store/types.CommitInfo.Hash at store v1.1.2.
python3 - "${ROOT}/deploy/frozen_evidence.py" <<'PY'
import importlib.util
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
spec = importlib.util.spec_from_file_location("frozen_evidence_vector", path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
roots = {
    "auth": "44" * 32,
    "bank": "02" * 32,
    "staking": "03" * 32,
    "zerone_staking": "01" * 32,
}
expected = "dc083957779448e737b33ba92366b3d2f56d29989c8cabbb2f5d2eaa803a1c48"
if module._cosmos_commit_info_hash(roots) != expected:
    raise SystemExit("Python CommitInfo.Hash port differs from independent Go vector")
upstream_roots = {
    "key1": b"value1".hex(),
    "key2": b"value2".hex(),
    "key3": b"value3".hex(),
}
upstream_expected = (
    "1dd674ec6782a0d586a903c9c63326a41cbe56b3bba33ed6ff5b527af6efb3dc"
)
if module._cosmos_commit_info_hash(upstream_roots) != upstream_expected:
    raise SystemExit("Python RFC6962 split differs from upstream three-leaf golden")
if module._go_uvarint(128) != b"\x80\x01":
    raise SystemExit("Python Go-uvarint port rejects a 128-byte store name length")
address_vectors = (
    (
        "zrn",
        bytes.fromhex("cdd7bb3a2d970c608e033f3369cfbb75a326583b"),
        "zrn1ehtmkw3djuxxprsr8ueknnamwk3jvkpmlzfepn",
    ),
    (
        "zrn",
        b"\x11" * 20,
        "zrn1zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3wp0vre",
    ),
    (
        "zrnvaloper",
        b"\x11" * 20,
        "zrnvaloper1zyg3zyg3zyg3zyg3zyg3zyg3zyg3zyg3lgwxk9",
    ),
)
for hrp, payload, encoded in address_vectors:
    if module._bech32_encode(hrp, payload) != encoded:
        raise SystemExit("Python Bech32 encoder differs from independent Go vector")
    if module._bech32_address(encoded, hrp, "vector") != payload:
        raise SystemExit("Python Bech32 decoder differs from independent Go vector")
try:
    module._cosmos_commit_info_hash({"\ud800": "00" * 32})
except module.FrozenEvidenceError:
    pass
else:
    raise SystemExit("Python CommitInfo.Hash port did not fail closed on invalid UTF-8")
claim = {
    "source_kind": "pending_unbonding",
    "source_claim_id": "\ud800",
    "claimant": "x",
    "validator": "x",
    "denom": "uzrn",
    "amount": "1",
}
try:
    module._claimant_root([claim])
except module.FrozenEvidenceError:
    pass
else:
    raise SystemExit("claimant root did not fail closed on invalid UTF-8")
PY

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

missing_monitoring=$(clone_bundle missing-monitoring)
mv "${missing_monitoring}/MONITORING-ALERTS.json" \
  "${missing_monitoring}/MONITORING-ALERTS.json.missing"
expect_rejected "missing monitoring artifact" \
  "could not open bundle file MONITORING-ALERTS.json" \
  run_cutover_pre "${missing_monitoring}"

stalled_stimulus_evidence=MONITORING-ALERT-STALLED-HEIGHT-STIMULUS-EVIDENCE.json.raw
missed_stimulus_evidence=MONITORING-ALERT-MISSED-SIGNING-STIMULUS-EVIDENCE.json.raw

missing_monitoring_evidence=$(clone_bundle missing-monitoring-evidence)
mv "${missing_monitoring_evidence}/${stalled_stimulus_evidence}" \
  "${missing_monitoring_evidence}/${stalled_stimulus_evidence}.missing"
expect_rejected "missing monitoring evidence blob" \
  "could not open bundle file ${stalled_stimulus_evidence}" \
  run_cutover_pre "${missing_monitoring_evidence}"

substituted_monitoring_evidence=$(clone_bundle substituted-monitoring-evidence)
cp "${substituted_monitoring_evidence}/${missed_stimulus_evidence}" \
  "${substituted_monitoring_evidence}/${stalled_stimulus_evidence}"
expect_rejected "substituted monitoring evidence bytes" \
  "stalled_height stimulus evidence hash differs from bundled ${stalled_stimulus_evidence}" \
  run_cutover_pre "${substituted_monitoring_evidence}"

substituted_monitoring_reference=$(clone_bundle substituted-monitoring-reference)
missed_stimulus_sha=$(sha256_file \
  "${substituted_monitoring_reference}/${missed_stimulus_evidence}")
# shellcheck disable=SC2016 # jq variables are not shell variables.
canonical_mutate \
  "${substituted_monitoring_reference}/MONITORING-ALERT-TESTS.json" \
  '(.tests[] | select(.check == "stalled_height") | .evidence.stimulus) = {
    filename: $filename,
    sha256: $sha
  }' \
  --arg filename "${missed_stimulus_evidence}" \
  --arg sha "${missed_stimulus_sha}"
rebind_monitoring_chain "${substituted_monitoring_reference}"
expect_rejected "substituted monitoring evidence reference" \
  "stalled_height stimulus evidence must reference exact bundle file ${stalled_stimulus_evidence}" \
  run_cutover_pre "${substituted_monitoring_reference}"

malformed_monitoring_reference=$(clone_bundle malformed-monitoring-reference)
canonical_mutate \
  "${malformed_monitoring_reference}/MONITORING-ALERT-TESTS.json" \
  'del(.tests[] | select(.check == "stalled_height")
    | .evidence.stimulus.filename)'
rebind_monitoring_chain "${malformed_monitoring_reference}"
expect_rejected "malformed monitoring evidence reference" \
  "stalled_height stimulus evidence reference does not have the exact required fields" \
  run_cutover_pre "${malformed_monitoring_reference}"

unbound_monitoring_hash=$(clone_bundle unbound-monitoring-hash)
canonical_mutate \
  "${unbound_monitoring_hash}/MONITORING-ALERT-TESTS.json" \
  '(.tests[] | select(.check == "stalled_height")
    | .evidence.stimulus.sha256) = ("f" * 64)'
rebind_monitoring_chain "${unbound_monitoring_hash}"
expect_rejected "unbound monitoring evidence hash" \
  "stalled_height stimulus evidence hash differs from bundled ${stalled_stimulus_evidence}" \
  run_cutover_pre "${unbound_monitoring_hash}"

empty_monitoring_evidence=$(clone_bundle empty-monitoring-evidence)
python3 - "${empty_monitoring_evidence}/${stalled_stimulus_evidence}" <<'PY'
import pathlib
import sys

with pathlib.Path(sys.argv[1]).open("wb"):
    pass
PY
empty_monitoring_sha=$(sha256_file \
  "${empty_monitoring_evidence}/${stalled_stimulus_evidence}")
# shellcheck disable=SC2016 # jq variables are not shell variables.
canonical_mutate \
  "${empty_monitoring_evidence}/MONITORING-ALERT-TESTS.json" \
  '(.tests[] | select(.check == "stalled_height")
    | .evidence.stimulus.sha256) = $sha' \
  --arg sha "${empty_monitoring_sha}"
rebind_monitoring_chain "${empty_monitoring_evidence}"
expect_rejected "empty monitoring evidence blob" \
  "stalled_height stimulus evidence file is empty" \
  run_cutover_pre "${empty_monitoring_evidence}"

oversized_monitoring_evidence=$(clone_bundle oversized-monitoring-evidence)
python3 - "${oversized_monitoring_evidence}/${stalled_stimulus_evidence}" <<'PY'
import pathlib
import sys

with pathlib.Path(sys.argv[1]).open("wb") as evidence:
    evidence.truncate(16 * 1024 * 1024 + 1)
PY
expect_rejected "oversized monitoring evidence blob" \
  "${stalled_stimulus_evidence} exceeds its pre-authentication size limit" \
  run_cutover_pre "${oversized_monitoring_evidence}"

tampered_monitoring=$(clone_bundle tampered-monitoring)
canonical_mutate "${tampered_monitoring}/MONITORING-ALERTS.json" \
  '.result = "FAIL"'
expect_rejected "tampered monitoring artifact" \
  "RELEASE monitoring-alert hash differs" \
  run_cutover_pre "${tampered_monitoring}"

tampered_monitoring_rules=$(clone_bundle tampered-monitoring-rules)
canonical_mutate "${tampered_monitoring_rules}/MONITORING-RULES.json" \
  '.rules[0].parameters.maximum_stall_seconds = 121'
expect_rejected "tampered monitoring rules" \
  "monitoring rules hash differs" \
  run_cutover_pre "${tampered_monitoring_rules}"

disabled_monitoring_rule=$(clone_bundle disabled-monitoring-rule)
canonical_mutate "${disabled_monitoring_rule}/MONITORING-RULES.json" \
  '(.rules[] | select(.check == "stalled_height") | .enabled) = false'
rebind_monitoring_chain "${disabled_monitoring_rule}"
expect_rejected "disabled required monitoring rule" \
  "is disabled or semantically incomplete" \
  run_cutover_pre "${disabled_monitoring_rule}"

self_asserted_monitoring=$(clone_bundle self-asserted-monitoring)
canonical_mutate \
  "${self_asserted_monitoring}/MONITORING-ALERT-TESTS.json" \
  '.tests |= map({check, result})'
rebind_monitoring_chain "${self_asserted_monitoring}"
expect_rejected "self-asserted monitoring PASS without evidence" \
  "does not have the exact required fields" \
  run_cutover_pre "${self_asserted_monitoring}"

missing_monitoring_check=$(clone_bundle missing-monitoring-check)
canonical_mutate \
  "${missing_monitoring_check}/MONITORING-ALERT-TESTS.json" \
  'del(.tests[-1])'
rebind_monitoring_chain "${missing_monitoring_check}"
expect_rejected "missing required monitoring check" \
  "monitoring alert-test evidence is incomplete" \
  run_cutover_pre "${missing_monitoring_check}"

failed_monitoring_check=$(clone_bundle failed-monitoring-check)
canonical_mutate \
  "${failed_monitoring_check}/MONITORING-ALERT-TESTS.json" \
  '.tests[0].result = "FAIL"'
rebind_monitoring_chain "${failed_monitoring_check}"
expect_rejected "failed required monitoring check" \
  "did not prove firing and recovery" \
  run_cutover_pre "${failed_monitoring_check}"

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

signer_identity=$(clone_bundle component-signer-identity)
canonical_mutate \
  "${signer_identity}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json" \
  '.signer_identity =
    "https://github.com/other/zerone-core/.github/workflows/ci.yml@refs/heads/main"'
signer_identity_sha=$(sha256_file \
  "${signer_identity}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${signer_identity}/RELEASE-PACKET.json" \
  '.components.zerone_2_runtime.signature_sha256 = $sha' \
  --arg sha "${signer_identity_sha}"
expect_rejected "non-canonical component signer identity" \
  "image signature evidence is incomplete or mismatched" \
  run_cutover_pre "${signer_identity}"

certificate_issuer=$(clone_bundle component-certificate-issuer)
canonical_mutate \
  "${certificate_issuer}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json" \
  '.certificate_issuer = "https://issuer.example"'
certificate_issuer_sha=$(sha256_file \
  "${certificate_issuer}/ZERONE-2-RUNTIME-SIGNATURE-EVIDENCE.json")
# shellcheck disable=SC2016 # $sha is a jq variable, not a shell variable.
canonical_mutate "${certificate_issuer}/RELEASE-PACKET.json" \
  '.components.zerone_2_runtime.signature_sha256 = $sha' \
  --arg sha "${certificate_issuer_sha}"
expect_rejected "non-canonical component certificate issuer" \
  "image signature evidence is incomplete or mismatched" \
  run_cutover_pre "${certificate_issuer}"

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

missing_census=$(clone_bundle missing-custom-staking-census)
mv "${missing_census}/CUSTOM-STAKING-CENSUS.json" \
  "${missing_census}/CUSTOM-STAKING-CENSUS.json.missing"
expect_rejected "missing custom-staking census" \
  "could not open bundle file CUSTOM-STAKING-CENSUS.json" \
  run_open_pre "${missing_census}"

census_self_hash=$(clone_bundle custom-staking-census-self-hash)
census_mutate "${census_self_hash}/CUSTOM-STAKING-CENSUS.json" self-hash
rebind_census_final "${census_self_hash}"
expect_rejected "custom-staking census self-hash drift" \
  "census self-hash does not match its unsealed bytes" \
  run_open_pre "${census_self_hash}"

census_result=$(clone_bundle custom-staking-census-result)
census_mutate "${census_result}/CUSTOM-STAKING-CENSUS.json" result-fail
rebind_census_final "${census_result}"
expect_rejected "non-passing custom-staking census" \
  "custom-staking census did not PASS" \
  run_open_pre "${census_result}"

census_height=$(clone_bundle custom-staking-census-height)
census_mutate "${census_height}/CUSTOM-STAKING-CENSUS.json" checkpoint-height
rebind_census_final "${census_height}"
expect_rejected "custom-staking census at checkpoint F" \
  "post-anchor application state A, never checkpoint F" \
  run_open_pre "${census_height}"

census_app_hash=$(clone_bundle custom-staking-census-app-hash)
census_mutate \
  "${census_app_hash}/CUSTOM-STAKING-CENSUS.json" checkpoint-app-hash
rebind_census_final "${census_app_hash}"
expect_rejected "custom-staking census at checkpoint AppHash" \
  "post-anchor application state A, never checkpoint F" \
  run_open_pre "${census_app_hash}"

census_source=$(clone_bundle custom-staking-census-source)
census_mutate "${census_source}/CUSTOM-STAKING-CENSUS.json" source-commit
rebind_census_final "${census_source}"
expect_rejected "custom-staking census source drift" \
  "must bind RELEASE source" \
  run_open_pre "${census_source}"

census_arithmetic=$(clone_bundle custom-staking-census-arithmetic)
census_mutate "${census_arithmetic}/CUSTOM-STAKING-CENSUS.json" arithmetic
rebind_census_final "${census_arithmetic}"
expect_rejected "custom-staking census arithmetic drift" \
  "does not prove B = D + U and delta = 0" \
  run_open_pre "${census_arithmetic}"

census_findings=$(clone_bundle custom-staking-census-findings)
census_mutate "${census_findings}/CUSTOM-STAKING-CENSUS.json" findings
rebind_census_final "${census_findings}"
expect_rejected "custom-staking census findings" \
  "is incomplete or contains findings" \
  run_open_pre "${census_findings}"

census_incomplete=$(clone_bundle custom-staking-census-incomplete)
census_mutate \
  "${census_incomplete}/CUSTOM-STAKING-CENSUS.json" claimant-incomplete
rebind_census_final "${census_incomplete}"
expect_rejected "custom-staking census incomplete claimant root" \
  "is incomplete or contains findings" \
  run_open_pre "${census_incomplete}"

census_claimant_root=$(clone_bundle custom-staking-census-claimant-root)
census_mutate "${census_claimant_root}/CUSTOM-STAKING-CENSUS.json" claimant-root
rebind_census_final "${census_claimant_root}"
expect_rejected "custom-staking census claimant root drift" \
  "claimant root does not match the complete claim list" \
  run_open_pre "${census_claimant_root}"

census_multistore_root=$(clone_bundle custom-staking-census-multistore-root)
census_mutate "${census_multistore_root}/CUSTOM-STAKING-CENSUS.json" \
  multistore-root-drift
rebind_census_final "${census_multistore_root}"
expect_rejected "custom-staking census multistore root drift" \
  "multistore roots do not recompute post-anchor AppHash E" \
  run_open_pre "${census_multistore_root}"

census_module_identity=$(clone_bundle custom-staking-census-module-identity)
census_mutate "${census_module_identity}/CUSTOM-STAKING-CENSUS.json" \
  module-identity
rebind_census_final "${census_module_identity}"
expect_rejected "custom-staking census module identity drift" \
  "module identity is not deterministic" \
  run_open_pre "${census_module_identity}"

census_extra_denom=$(clone_bundle custom-staking-census-extra-denom)
census_mutate "${census_extra_denom}/CUSTOM-STAKING-CENSUS.json" \
  extra-module-denom
rebind_census_final "${census_extra_denom}"
expect_rejected "custom-staking census unexplained module denomination" \
  "module balances contain an unexplained denomination" \
  run_open_pre "${census_extra_denom}"

census_legacy_trust=$(clone_bundle custom-staking-census-legacy-key-trust)
census_mutate "${census_legacy_trust}/CUSTOM-STAKING-CENSUS.json" \
  legacy-key-trusted
rebind_census_final "${census_legacy_trust}"
expect_rejected "custom-staking census trusted legacy consensus key" \
  "invalid aggregate or legacy-key trust state" \
  run_open_pre "${census_legacy_trust}"

census_validator_claim=$(clone_bundle custom-staking-census-validator-claim)
census_mutate "${census_validator_claim}/CUSTOM-STAKING-CENSUS.json" \
  validator-claim-mismatch
rebind_census_final "${census_validator_claim}"
expect_rejected "custom-staking census validator claimant aggregate mismatch" \
  "validator computed aggregates do not match claims" \
  run_open_pre "${census_validator_claim}"

census_sdk_link=$(clone_bundle custom-staking-census-sdk-link)
census_mutate "${census_sdk_link}/CUSTOM-STAKING-CENSUS.json" sdk-link-absent
rebind_census_final "${census_sdk_link}"
expect_rejected "custom-staking census omitted SDK link" \
  "marks an available SDK link absent" \
  run_open_pre "${census_sdk_link}"

census_unbonding=$(clone_bundle custom-staking-census-unbonding-mismatch)
census_mutate "${census_unbonding}/CUSTOM-STAKING-CENSUS.json" \
  unbonding-mismatch
rebind_census_final "${census_unbonding}"
expect_rejected "custom-staking census unbonding claimant mismatch" \
  "does not match its pending claimant record" \
  run_open_pre "${census_unbonding}"

census_reverse=$(clone_bundle custom-staking-census-reverse-mismatch)
census_mutate "${census_reverse}/CUSTOM-STAKING-CENSUS.json" reverse-mismatch
rebind_census_final "${census_reverse}"
expect_rejected "custom-staking census reverse index mismatch" \
  "has no delegation claim" \
  run_open_pre "${census_reverse}"

census_tier=$(clone_bundle custom-staking-census-hollow-tier)
census_mutate "${census_tier}/CUSTOM-STAKING-CENSUS.json" hollow-tier
rebind_census_final "${census_tier}"
expect_rejected "custom-staking census hollow tier reconciliation" \
  "tier reconciliation[0] is incomplete" \
  run_open_pre "${census_tier}"

census_did_ceiling=$(clone_bundle custom-staking-census-did-byte-ceiling)
census_mutate "${census_did_ceiling}/CUSTOM-STAKING-CENSUS.json" \
  did-byte-ceiling
rebind_census_final "${census_did_ceiling}"
expect_rejected "custom-staking census oversized DID index" \
  "DID index[0] exceeds the 128-byte producer ceiling" \
  run_open_pre "${census_did_ceiling}"

census_custom_leaves=$(clone_bundle custom-staking-census-custom-leaf-ceiling)
census_mutate "${census_custom_leaves}/CUSTOM-STAKING-CENSUS.json" \
  custom-leaf-ceiling
rebind_census_final "${census_custom_leaves}"
expect_rejected "custom-staking census custom-store leaf ceiling" \
  "store[0] exceeds its scan leaf ceiling" \
  run_open_pre "${census_custom_leaves}"

census_custom_input=$(clone_bundle custom-staking-census-custom-input-ceiling)
census_mutate "${census_custom_input}/CUSTOM-STAKING-CENSUS.json" \
  custom-input-ceiling
rebind_census_final "${census_custom_input}"
expect_rejected "custom-staking census custom-store input ceiling" \
  "custom-staking store exceeds its retained-input scan ceiling" \
  run_open_pre "${census_custom_input}"

census_bank_leaves=$(clone_bundle custom-staking-census-bank-leaf-ceiling)
census_mutate "${census_bank_leaves}/CUSTOM-STAKING-CENSUS.json" \
  bank-leaf-ceiling
rebind_census_final "${census_bank_leaves}"
expect_rejected "custom-staking census bank-store leaf ceiling" \
  "store[1] exceeds its scan leaf ceiling" \
  run_open_pre "${census_bank_leaves}"

census_input_ceiling=$(clone_bundle custom-staking-census-input-byte-ceiling)
census_mutate "${census_input_ceiling}/CUSTOM-STAKING-CENSUS.json" \
  aggregate-input-ceiling
rebind_census_final "${census_input_ceiling}"
expect_rejected "custom-staking census required-store input ceiling" \
  "required stores exceed the aggregate scan byte ceiling" \
  run_open_pre "${census_input_ceiling}"

census_sdk_ceiling=$(clone_bundle custom-staking-census-sdk-row-ceiling)
census_mutate "${census_sdk_ceiling}/CUSTOM-STAKING-CENSUS.json" \
  sdk-row-ceiling
rebind_census_final "${census_sdk_ceiling}"
expect_rejected "custom-staking census SDK row ceiling" \
  "SDK validator inventory exceeds its row ceiling" \
  run_open_pre "${census_sdk_ceiling}"

census_sentinel=$(clone_bundle custom-staking-census-sentinel)
census_mutate "${census_sentinel}/CUSTOM-STAKING-CENSUS.json" sentinel-count
rebind_census_final "${census_sentinel}"
expect_rejected "custom-staking census duplicate IAVL sentinel" \
  "app IAVL sentinel inventory is invalid" \
  run_open_pre "${census_sentinel}"

census_unbound=$(clone_bundle custom-staking-census-unbound)
census_mutate "${census_unbound}/CUSTOM-STAKING-CENSUS.json" \
  unbound-multistore
expect_rejected "custom-staking census not hashed by FINAL" \
  "FINAL frozen artifact hashes differ from the actual evidence" \
  run_open_pre "${census_unbound}"

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

for archive_pin in height app-hash block-hash; do
  archive_drift=$(clone_bundle "archive-gateway-${archive_pin}-drift")
  archive_config="${archive_drift}/fly.zerone-1-archive-gateway.public.toml"
  case "${archive_pin}" in
    height)
      sed 's/EXPECTED_ARCHIVE_HEIGHT = "1001"/EXPECTED_ARCHIVE_HEIGHT = "1000"/' \
        "${archive_config}" > "${archive_config}.new"
      ;;
    app-hash)
      sed -E 's/(EXPECTED_ARCHIVE_APP_HASH = ")[0-9a-f]{64}(".*)/\1cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\2/' \
        "${archive_config}" > "${archive_config}.new"
      ;;
    block-hash)
      sed -E 's/(EXPECTED_ARCHIVE_BLOCK_HASH = ")[0-9a-f]{64}(".*)/\1dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd\2/' \
        "${archive_config}" > "${archive_config}.new"
      ;;
  esac
  mv "${archive_config}.new" "${archive_config}"
  rebind_archive_gateway_open "${archive_drift}"
  expect_rejected "OPEN-owned archive ${archive_pin} drift" \
    "not the exact RELEASE/FINAL render" run_open_pre "${archive_drift}"
done

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
