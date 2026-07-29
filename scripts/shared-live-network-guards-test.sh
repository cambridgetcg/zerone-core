#!/usr/bin/env bash
# Focused regression tests for source-level local-rehearsal chain guards.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IDENTITY_BRIDGE="${ROOT}/scripts/agenttool-identity-bridge.sh"
TRUTH_CEREMONY="${ROOT}/scripts/first-truth-ceremony.sh"
TMP="$(mktemp -d)"

cleanup() {
  rm -rf -- "${TMP}"
}
trap cleanup EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

FAKE_BINARY="${TMP}/zeroned"
# The following arguments are the literal source of the sentinel shell script.
# shellcheck disable=SC2016
printf '%s\n' \
  '#!/bin/sh' \
  ': "${GUARD_MARKER:?}"' \
  'printf "invoked\n" > "${GUARD_MARKER}"' \
  'exit 97' > "${FAKE_BINARY}"
chmod 700 "${FAKE_BINARY}"

run_script() {
  local script="$1"
  local chain_id="$2"
  local marker="$3"
  local state_dir="$4"
  local log_file="$5"
  shift 5

  env \
    BINARY="${FAKE_BINARY}" \
    CHAIN_ID="${chain_id}" \
    CONTENT="A deliberately local rehearsal claim used only by the guard test." \
    GUARD_MARKER="${marker}" \
    HOME="${TMP}/home" \
    NODE="tcp://127.0.0.1:1" \
    STATE_DIR="${state_dir}" \
    bash "${script}" "$@" > "${log_file}" 2>&1
}

expect_refusal_before_action() {
  local script="$1"
  local chain_id="$2"
  local label="$3"
  shift 3
  local marker="${TMP}/${label}-${chain_id}.invoked"
  local state_dir="${TMP}/${label}-${chain_id}.state"
  local log_file="${TMP}/${label}-${chain_id}.log"

  if run_script "${script}" "${chain_id}" "${marker}" "${state_dir}" "${log_file}" "$@"; then
    fail "${label} accepted unsafe chain ${chain_id}"
  fi
  grep -q "refusing shared/live chain ID" "${log_file}" ||
    fail "${label} did not explain unsafe-chain refusal for ${chain_id}"
  [ ! -e "${marker}" ] ||
    fail "${label} invoked zeroned before refusing ${chain_id}"
  [ ! -e "${state_dir}" ] ||
    fail "${label} created state before refusing ${chain_id}"
}

expect_allowed_reaches_binary() {
  local script="$1"
  local chain_id="$2"
  local label="$3"
  shift 3
  local marker="${TMP}/${label}-${chain_id}.allowed-invoked"
  local state_dir="${TMP}/${label}-${chain_id}.allowed-state"
  local log_file="${TMP}/${label}-${chain_id}.allowed-log"

  if run_script "${script}" "${chain_id}" "${marker}" "${state_dir}" "${log_file}" "$@"; then
    fail "${label} unexpectedly completed with the sentinel binary"
  fi
  [ -e "${marker}" ] ||
    fail "${label} rejected allowed drill chain ${chain_id}"
}

for chain_id in zerone-1 zerone-testnet-1 zerone-rehearsal; do
  expect_refusal_before_action \
    "${IDENTITY_BRIDGE}" "${chain_id}" identity \
    "did:at:guard-test" "guard-key"
  expect_refusal_before_action \
    "${TRUTH_CEREMONY}" "${chain_id}" truth
done

for chain_id in zerone-localnet zerone-rehearsal-security-drill; do
  expect_allowed_reaches_binary \
    "${IDENTITY_BRIDGE}" "${chain_id}" identity \
    "did:at:guard-test" "guard-key"
  expect_allowed_reaches_binary \
    "${TRUTH_CEREMONY}" "${chain_id}" truth
done

printf 'shared/live network guard tests: PASS\n'
