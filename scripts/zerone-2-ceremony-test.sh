#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C
export GOPROXY=off

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${ROOT}/build/zeroned}"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-ceremony-test.XXXXXX")"
trap 'rm -rf "${TMP}"' EXIT INT TERM

fail() {
  printf 'zerone-2 ceremony test: FAIL: %s\n' "$*" >&2
  exit 1
}

run_drill() {
  local output="$1" log="$2"
  if ! BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" \
    drill "${output}" > "${log}" 2>&1; then
    sed -n '1,160p' "${log}" >&2
    fail "drill failed"
  fi
  grep -q 'PASS zerone-2 artifact audit' "${log}" || \
    fail "mandatory artifact audit was not executed"
}

[ -x "${BINARY}" ] || fail "binary missing at ${BINARY}"

# Real-mode public inputs are frozen into private scratch before any parsing.
# Symlinks must be rejected before platform, tag, or signature gates can run.
printf '{}\n' > "${TMP}/public-input-target.json"
printf '{}\n' > "${TMP}/gentx-input.json"
ln -s public-input-target.json "${TMP}/public-input-link.json"
if BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" real \
  "${TMP}/public-input-link.json" "${TMP}/gentx-input.json" \
  "${TMP}/symlink-input-output" > "${TMP}/symlink-input.log" 2>&1; then
  fail "ceremony accepted a symlinked public input"
fi
grep -q 'public ceremony input must be a regular, non-symlink file' \
  "${TMP}/symlink-input.log" || fail "public-input symlink rejection was not explicit"

ln -s gentx-input.json "${TMP}/gentx-input-link.json"
if BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" real \
  "${TMP}/public-input-target.json" "${TMP}/gentx-input-link.json" \
  "${TMP}/symlink-gentx-output" > "${TMP}/symlink-gentx.log" 2>&1; then
  fail "ceremony accepted a symlinked gentx input"
fi
grep -q 'signed gentx input must be a regular, non-symlink file' \
  "${TMP}/symlink-gentx.log" || fail "gentx symlink rejection was not explicit"

jq -n '{
  schema: "zerone-2-public-ceremony-input-v2",
  chain_id: "zerone-2",
  genesis_time: "2026-01-01T00:00:00Z",
  validator: {
    account_address: "zerone1placeholder",
    moniker: "zerone-2-custodian",
    custody_disclosure: "test"
  },
  operations: {account_address: "zerone1operations"},
  release: {
    source_commit: "0000000000000000000000000000000000000000",
    release_tag: "v0-test",
    tag_signer_fingerprint: "0000000000000000000000000000000000000000",
    binary_sha256: "0000000000000000000000000000000000000000000000000000000000000000",
    binary_goos: "linux",
    binary_goarch: "amd64"
  },
  unexpected_typo: true
}' > "${TMP}/unknown-field-input.json"
if BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" real \
  "${TMP}/unknown-field-input.json" "${TMP}/gentx-input.json" \
  "${TMP}/unknown-field-output" > "${TMP}/unknown-field.log" 2>&1; then
  fail "ceremony accepted an unknown public-input field"
fi
grep -q 'public input does not match zerone-2-public-ceremony-input-v2' \
  "${TMP}/unknown-field.log" || fail "unknown-field rejection was not explicit"

run_drill "${TMP}/a" "${TMP}/a.log"
run_drill "${TMP}/b" "${TMP}/b.log"

for artifact in genesis.json genesis.sha256 network-manifest.json GENESIS-MANIFEST.md; do
  cmp "${TMP}/a/${artifact}" "${TMP}/b/${artifact}" >/dev/null || \
    fail "${artifact} is not reproducible"
done

"${BINARY}" genesis validate "${TMP}/a/genesis.json" >/dev/null 2>&1 || \
  fail "zeroned rejected drill genesis"

# A completed directory is immutable input: ceremony must never overwrite it.
if BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" \
  drill "${TMP}/a" > "${TMP}/overwrite.log" 2>&1; then
  fail "ceremony overwrote an existing artifact directory"
fi
grep -q 'refusing overwrite' "${TMP}/overwrite.log" || \
  fail "overwrite rejection was not explicit"

# Interpose a destination after the ceremony's early absence check but before
# final publication. The publisher must fail, preserve the interloper exactly,
# and never nest its completed public-artifacts directory inside it.
RACE_OUT="${TMP}/late-destination"
RACE_LOG="${TMP}/late-destination.log"
BINARY="${BINARY}" "${ROOT}/scripts/zerone-2-ceremony.sh" \
  drill "${RACE_OUT}" > "${RACE_LOG}" 2>&1 &
RACE_PID=$!
RACE_READY=false
for ((attempt = 0; attempt < 1000; attempt++)); do
  if grep -q 'initializing keyless coordinator state' "${RACE_LOG}" 2>/dev/null; then
    RACE_READY=true
    break
  fi
  if ! kill -0 "${RACE_PID}" 2>/dev/null; then
    break
  fi
  sleep 0.01
done
[ "${RACE_READY}" = true ] || {
  wait "${RACE_PID}" 2>/dev/null || true
  sed -n '1,160p' "${RACE_LOG}" >&2
  fail "could not reach the late-destination race window"
}
mkdir "${RACE_OUT}"
printf 'must survive\n' > "${RACE_OUT}/sentinel"
if wait "${RACE_PID}"; then
  fail "ceremony published into a destination created during the run"
fi
grep -q 'could not atomically publish artifacts' "${RACE_LOG}" || \
  fail "late-destination rejection was not explicit"
grep -qx 'must survive' "${RACE_OUT}/sentinel" || \
  fail "late destination was modified or removed"
[ ! -e "${RACE_OUT}/public-artifacts" ] || \
  fail "ceremony nested artifacts inside the late destination"

# Prove the independent gate rejects policy drift even when Cosmos JSON remains
# syntactically valid.
cp -R "${TMP}/a" "${TMP}/drift"
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' \
  "${TMP}/drift/genesis.json" > "${TMP}/drifted-genesis.json"
mv "${TMP}/drifted-genesis.json" "${TMP}/drift/genesis.json"
if go run "${ROOT}/tools/zerone2-artifact-audit" \
  --artifact-dir "${TMP}/drift" --required-mode drill > "${TMP}/drift.log" 2>&1; then
  fail "artifact auditor accepted vote-extension drift"
fi
grep -q 'vote_extensions_enable_height' "${TMP}/drift.log" || \
  fail "vote-extension drift rejection was not explicit"

# A renamed valid mnemonic must also fail the release directory scan.
cp -R "${TMP}/a" "${TMP}/secret"
printf '%s\n' \
  'now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct' \
  > "${TMP}/secret/innocent-notes.txt"
if go run "${ROOT}/tools/zerone2-artifact-audit" \
  --artifact-dir "${TMP}/secret" --required-mode drill > "${TMP}/secret.log" 2>&1; then
  fail "artifact auditor accepted a renamed mnemonic"
fi
grep -q 'valid BIP-39 mnemonic' "${TMP}/secret.log" || \
  fail "renamed mnemonic rejection was not explicit"

printf 'zerone-2 ceremony test: PASS\n'
