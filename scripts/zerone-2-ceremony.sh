#!/usr/bin/env bash
# shellcheck disable=SC2016 # jq programs use $name variables inside single quotes.
# Keyless, fail-closed genesis ceremony for the public zerone-2 relaunch.
#
# Real mode accepts public metadata and one offline-signed gentx. It never
# accepts custody material. Drill mode uses an explicitly public fixture key
# and must never be deployed.

set -euo pipefail
export LC_ALL=C
export GIT_NO_REPLACE_OBJECTS=1
umask 077

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="${BINARY:-${PROJECT_ROOT}/build/zeroned}"

CHAIN_ID="zerone-2"
INPUT_SCHEMA="zerone-2-public-ceremony-input-v2"
MANIFEST_SCHEMA="zerone-2-network-manifest-v2"
VALIDATOR_BALANCE_UZRN="11333000000"
SELF_BOND_UZRN="11111000000"
OPS_BALANCE_UZRN="2222000000"
TOTAL_SUPPLY_UZRN="13555000000"
HARD_CAP_PLUS_ONE_UZRN="222222222000001"
CUSTOM_STAKE_UZRN="111000000"
GOV_MIN_DEPOSIT_UZRN="100000000"
GOV_EXPEDITED_DEPOSIT_UZRN="300000000"

# Public test fixture. Funds derived from this phrase are never money.
DRILL_MNEMONIC="now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct"
DRILL_GENESIS_TIME="2026-01-01T00:00:00Z"
DRILL_NODE_ID="2222222222222222222222222222222222222222"

info() { printf '  -> %s\n' "$*"; }
ok() { printf '  OK %s\n' "$*"; }
die() { printf 'FAIL %s\n' "$*" >&2; exit 1; }
require_regular_file() {
  local file="$1" label="$2"
  [ -f "${file}" ] && [ ! -L "${file}" ] || \
    die "${label} must be a regular, non-symlink file: ${file}"
}

usage() {
  cat >&2 <<'EOF'
usage:
  scripts/zerone-2-ceremony.sh drill OUTPUT_DIR
  scripts/zerone-2-ceremony.sh real PUBLIC_INPUT_JSON SIGNED_GENTX_JSON OUTPUT_DIR

OUTPUT_DIR must not already exist. Real inputs must contain public data only.
EOF
  exit 2
}

[ -f "${BINARY}" ] && [ ! -L "${BINARY}" ] && [ -x "${BINARY}" ] || \
  die "zeroned binary must be an executable regular non-symlink file at ${BINARY}"
command -v jq >/dev/null || die "jq is required"
(command -v sha256sum >/dev/null || command -v shasum >/dev/null) || \
  die "sha256sum or shasum is required"
command -v go >/dev/null || die "go is required to inspect binary build metadata"
command -v git >/dev/null || die "git is required"

MODE="${1:-}"
case "${MODE}" in
  drill)
    [ "$#" -eq 2 ] || usage
    OUT_DIR="$2"
    INPUT_FILE=""
    GENTX_INPUT=""
    ;;
  real)
    [ "$#" -eq 4 ] || usage
    INPUT_FILE="$2"
    GENTX_INPUT="$3"
    OUT_DIR="$4"
    ;;
  *) usage ;;
esac

[ ! -e "${OUT_DIR}" ] || die "output already exists; refusing overwrite: ${OUT_DIR}"
OUTPUT_PARENT="$(cd "$(dirname "${OUT_DIR}")" && pwd)" \
  || die "output parent must already exist: $(dirname "${OUT_DIR}")"

SCRATCH="$(mktemp -d "${OUTPUT_PARENT}/.zerone-2-ceremony.XXXXXX")"
cleanup() {
  chmod -R u+w "${SCRATCH}" 2>/dev/null || true
  rm -rf "${SCRATCH}"
}
trap cleanup EXIT INT TERM

# `mv source destination` is unsafe here: if destination appears after the
# early existence check, mv treats it as a directory and nests source inside
# it. Build a tiny publisher around each supported OS's atomic no-replace
# rename primitive. There is no placeholder to leak or clean up: the complete
# directory appears under the final name in one syscall, or nothing changes.
PUBLISH_HELPER_SOURCE="${SCRATCH}/publish-directory.go"
PUBLISH_HELPER_PLATFORM="${SCRATCH}/publish-directory-platform.go"
PUBLISH_HELPER="${SCRATCH}/publish-directory"
cat > "${PUBLISH_HELPER_SOURCE}" <<'EOF'
package main

import (
	"fmt"
	"os"
)

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		fail("usage: publish-directory SOURCE DESTINATION")
	}

	source, destination := os.Args[1], os.Args[2]
	if err := renameNoReplace(source, destination); err != nil {
		fail("could not publish %q to %q: %v", source, destination, err)
	}
}
EOF
GO_HOST_OS=$(GOENV=off GOWORK=off GOFLAGS='' GOTOOLCHAIN=local \
  GOOS='' GOARCH='' go env GOHOSTOS) || die "could not resolve the local Go host OS"
case "${GO_HOST_OS}" in
  darwin)
    cat > "${PUBLISH_HELPER_PLATFORM}" <<'EOF'
package main

import "golang.org/x/sys/unix"

func renameNoReplace(source, destination string) error {
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
}
EOF
    ;;
  linux)
    cat > "${PUBLISH_HELPER_PLATFORM}" <<'EOF'
package main

import "golang.org/x/sys/unix"

func renameNoReplace(source, destination string) error {
	return unix.Renameat2(
		unix.AT_FDCWD,
		source,
		unix.AT_FDCWD,
		destination,
		unix.RENAME_NOREPLACE,
	)
}
EOF
    ;;
  *) die "atomic directory publication is supported only on macOS and Linux" ;;
esac
(cd "${PROJECT_ROOT}" && \
  GOENV=off GOWORK=off GOFLAGS='' GOTOOLCHAIN=local GOPROXY=off \
  CGO_ENABLED=0 GOOS='' GOARCH='' go build -mod=readonly -trimpath \
    -o "${PUBLISH_HELPER}" \
    "${PUBLISH_HELPER_SOURCE}" "${PUBLISH_HELPER_PLATFORM}") || \
  die "could not build the atomic directory publisher"

HOME_DIR="${SCRATCH}/coordinator-home"
GENESIS="${HOME_DIR}/config/genesis.json"
GENTX_DIR="${SCRATCH}/gentxs"
ARTIFACT_DIR="${SCRATCH}/public-artifacts"
SOURCE_BINARY="${BINARY}"
IMMUTABLE_BINARY_DIR="${SCRATCH}/release-binary"
mkdir -p "${GENTX_DIR}" "${IMMUTABLE_BINARY_DIR}"
mkdir -m 0755 "${ARTIFACT_DIR}"

# Freeze both caller-controlled public inputs before parsing either one. `cp -P`
# preserves a raced symlink as a symlink so the destination check fails closed;
# all later reads use only these private scratch copies.
if [ "${MODE}" = "real" ]; then
  IMMUTABLE_INPUT_DIR="${SCRATCH}/public-inputs"
  mkdir -m 0700 "${IMMUTABLE_INPUT_DIR}"
  require_regular_file "${INPUT_FILE}" "public ceremony input"
  cp -P "${INPUT_FILE}" "${IMMUTABLE_INPUT_DIR}/public-input.json" || \
    die "could not freeze public ceremony input"
  require_regular_file "${IMMUTABLE_INPUT_DIR}/public-input.json" \
    "frozen public ceremony input"
  require_regular_file "${GENTX_INPUT}" "signed gentx input"
  cp -P "${GENTX_INPUT}" "${IMMUTABLE_INPUT_DIR}/signed-gentx.json" || \
    die "could not freeze signed gentx input"
  require_regular_file "${IMMUTABLE_INPUT_DIR}/signed-gentx.json" \
    "frozen signed gentx input"
  chmod 0400 "${IMMUTABLE_INPUT_DIR}/public-input.json" \
    "${IMMUTABLE_INPUT_DIR}/signed-gentx.json"
  INPUT_FILE="${IMMUTABLE_INPUT_DIR}/public-input.json"
  GENTX_INPUT="${IMMUTABLE_INPUT_DIR}/signed-gentx.json"
fi

# Every ceremony operation uses one early private copy. The caller's path can
# no longer be swapped or rebuilt between the declared hash, gentx collection,
# validation, and final audit.
install -m 0555 "${SOURCE_BINARY}" "${IMMUTABLE_BINARY_DIR}/zeroned"
BINARY="${IMMUTABLE_BINARY_DIR}/zeroned"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
is_sha256() { [[ "$1" =~ ^[0-9a-f]{64}$ ]]; }
is_commit() { [[ "$1" =~ ^[0-9a-f]{40}$ ]]; }
is_fingerprint() { [[ "$1" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]]; }

binary_build_setting() {
  local name="$1"
  go version -m "${BINARY}" 2>/dev/null \
    | awk -v name="${name}" '$1 == "build" && index($2, name "=") == 1 {sub(name "=", "", $2); print $2; exit}'
}

reject_git_history_overrides() {
  local git_dir
  [ -z "$(git -C "${PROJECT_ROOT}" for-each-ref --format='%(refname)' refs/replace/)" ] || \
    die "replacement Git refs are forbidden for a real ceremony"
  git_dir=$(git -C "${PROJECT_ROOT}" rev-parse --absolute-git-dir) || \
    die "could not resolve repository Git directory"
  if [ -e "${git_dir}/info/grafts" ] || [ -L "${git_dir}/info/grafts" ]; then
    die "legacy Git grafts are forbidden for a real ceremony"
  fi
}

verify_clean_release_checkout() {
  [ "$(git -C "${PROJECT_ROOT}" rev-parse HEAD)" = "${RELEASE_COMMIT}" ] || \
    die "checked-out HEAD changed during the real ceremony"
  [ -z "$(git -C "${PROJECT_ROOT}" status --porcelain --untracked-files=all)" ] || \
    die "real ceremony requires a clean checkout"
  reject_git_history_overrides
}

materialize_release_file() {
  local relative="$1" destination="$2" listing mode
  listing=$(git -C "${PROJECT_ROOT}" ls-tree "${RELEASE_COMMIT}" -- "${relative}") || \
    die "could not inspect signed auditor input ${relative}"
  [ -n "${listing}" ] || die "signed release omits auditor input ${relative}"
  [ "$(printf '%s\n' "${listing}" | wc -l | tr -d ' ')" = "1" ] || \
    die "signed auditor input resolves ambiguously: ${relative}"
  mode=$(printf '%s\n' "${listing}" | awk '{print $1}')
  case "${mode}" in
    100644|100755) ;;
    *) die "signed auditor input is not a regular Git blob: ${relative}" ;;
  esac
  mkdir -p "$(dirname "${destination}")"
  git -C "${PROJECT_ROOT}" show "${RELEASE_COMMIT}:${relative}" > "${destination}" || \
    die "could not materialize signed auditor input ${relative}"
  chmod 0644 "${destination}"
}

build_signed_artifact_auditor() {
  local source_root="${SCRATCH}/signed-auditor-source" relative count=0
  SIGNED_AUDITOR="${SCRATCH}/signed-zerone2-artifact-audit"
  mkdir -m 0700 "${source_root}"

  # The auditor imports app.MakeEncodingConfig, so its complete local compile
  # closure is the signed go.mod/go.sum plus app, x, and the embedded Swagger
  # package. Deployment artifacts and caller worktree bytes are excluded.
  while IFS= read -r -d '' relative; do
    materialize_release_file "${relative}" "${source_root}/${relative}"
    count=$((count + 1))
  done < <(git -C "${PROJECT_ROOT}" ls-tree -r -z --name-only \
    "${RELEASE_COMMIT}" -- go.mod go.sum app x docs/swagger-ui \
    tools/zerone2-artifact-audit/main.go)
  [ "${count}" -gt 5 ] || die "signed auditor source allowlist was unexpectedly empty"
  [ -f "${source_root}/tools/zerone2-artifact-audit/main.go" ] || \
    die "signed release omits the artifact auditor"

  (cd "${source_root}" && \
    GOENV=off GOWORK=off GOFLAGS='' GOTOOLCHAIN=local GOPROXY=off \
    CGO_ENABLED=0 GOOS='' GOARCH='' go mod verify && \
    GOENV=off GOWORK=off GOFLAGS='' GOTOOLCHAIN=local GOPROXY=off \
    CGO_ENABLED=0 GOOS='' GOARCH='' go build -mod=readonly -trimpath \
      -o "${SIGNED_AUDITOR}" ./tools/zerone2-artifact-audit) || \
    die "could not build artifact auditor from signed Git blobs"
  [ -f "${SIGNED_AUDITOR}" ] && [ ! -L "${SIGNED_AUDITOR}" ] && \
    [ -x "${SIGNED_AUDITOR}" ] || die "signed artifact auditor build is invalid"
}

patch_genesis() {
  local filter="$1"
  shift
  local tmp="${SCRATCH}/genesis.patch.json"
  jq "$@" "${filter}" "${GENESIS}" > "${tmp}" || die "genesis patch failed"
  mv "${tmp}" "${GENESIS}"
}

CURRENT_COMMIT="$(git -C "${PROJECT_ROOT}" rev-parse HEAD)"
BINARY_SHA="$(sha256_file "${BINARY}")"
if ! BINARY_VERSION_OUTPUT=$("${BINARY}" version 2>&1); then
  die "copied release binary failed to report its version"
fi
BINARY_VERSION=$(printf '%s\n' "${BINARY_VERSION_OUTPUT}" | sed -n '1p')
BINARY_GOOS="$(binary_build_setting GOOS)"
BINARY_GOARCH="$(binary_build_setting GOARCH)"
BINARY_VCS_REVISION="$(binary_build_setting vcs.revision)"
BINARY_VCS_MODIFIED="$(binary_build_setting vcs.modified)"
[[ "${BINARY_VERSION}" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$ ]] || \
  die "copied release binary reported an unsafe version"
[ -n "${BINARY_GOOS}" ] || die "copied release binary has no GOOS build metadata"
[ -n "${BINARY_GOARCH}" ] || die "copied release binary has no GOARCH build metadata"

if [ "${MODE}" = "real" ]; then
  reject_git_history_overrides
  jq -e --arg schema "${INPUT_SCHEMA}" --arg chain "${CHAIN_ID}" '
    type == "object"
    and (keys | sort) == (["schema","chain_id","genesis_time","validator","operations","release"] | sort)
    and .schema == $schema
    and .chain_id == $chain
    and (.genesis_time | type == "string" and (fromdateiso8601 | type == "number"))
    and (.validator | type == "object")
    and (.validator | keys | sort) == (["account_address","moniker","custody_disclosure"] | sort)
    and (.validator.account_address | type == "string")
    and .validator.moniker == "zerone-2-custodian"
    and (.validator.custody_disclosure | type == "string" and length > 0)
    and (.operations | type == "object")
    and (.operations | keys) == ["account_address"]
    and (.operations.account_address | type == "string")
    and (.release | type == "object")
    and (.release | keys | sort) == (["source_commit","release_tag","tag_signer_fingerprint","binary_sha256","binary_goos","binary_goarch"] | sort)
    and (.release.source_commit | type == "string")
    and (.release.release_tag | type == "string" and length > 0)
    and (.release.tag_signer_fingerprint | type == "string")
    and (.release.binary_sha256 | type == "string")
    and (.release.binary_goos | type == "string")
    and (.release.binary_goarch | type == "string")
  ' "${INPUT_FILE}" >/dev/null || die "public input does not match ${INPUT_SCHEMA}"

  # Reject secret-shaped input fields even if future schema changes add them.
  jq -e '
    [paths(scalars) as $p
      | ($p | map(tostring) | join(".") | ascii_downcase)
      | select(test("mnemonic|private|priv_key|password|passphrase|secret|seed"))]
    | length == 0
  ' "${INPUT_FILE}" >/dev/null || die "public input contains a custody-shaped field"

  GENESIS_TIME="$(jq -r '.genesis_time' "${INPUT_FILE}")"
  VALIDATOR_ADDRESS="$(jq -r '.validator.account_address' "${INPUT_FILE}")"
  VALIDATOR_MONIKER="$(jq -r '.validator.moniker' "${INPUT_FILE}")"
  CUSTODY_DISCLOSURE="$(jq -r '.validator.custody_disclosure' "${INPUT_FILE}")"
  OPS_ADDRESS="$(jq -r '.operations.account_address' "${INPUT_FILE}")"
  RELEASE_COMMIT="$(jq -r '.release.source_commit' "${INPUT_FILE}")"
  RELEASE_TAG="$(jq -r '.release.release_tag' "${INPUT_FILE}")"
  DECLARED_TAG_SIGNER_FINGERPRINT="$(jq -r '.release.tag_signer_fingerprint' "${INPUT_FILE}")"
  DECLARED_BINARY_SHA="$(jq -r '.release.binary_sha256' "${INPUT_FILE}")"
  DECLARED_BINARY_GOOS="$(jq -r '.release.binary_goos' "${INPUT_FILE}")"
  DECLARED_BINARY_GOARCH="$(jq -r '.release.binary_goarch' "${INPUT_FILE}")"
  AUTHORIZED_TAG_SIGNER_FINGERPRINT="${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT:-}"

  is_commit "${RELEASE_COMMIT}" || die "source_commit must be 40 lowercase hex"
  is_sha256 "${DECLARED_BINARY_SHA}" || die "binary_sha256 must be 64 lowercase hex"
  is_fingerprint "${DECLARED_TAG_SIGNER_FINGERPRINT}" || \
    die "tag_signer_fingerprint must be 40 or 64 lowercase hex"
  is_fingerprint "${AUTHORIZED_TAG_SIGNER_FINGERPRINT}" || \
    die "ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT must be 40 or 64 lowercase hex"
  [ "${DECLARED_TAG_SIGNER_FINGERPRINT}" = "${AUTHORIZED_TAG_SIGNER_FINGERPRINT}" ] || \
    die "public input tag signer is not the independently authorized fingerprint"
  [[ "${RELEASE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$ ]] || \
    die "release_tag contains unsafe characters"
  git -C "${PROJECT_ROOT}" check-ref-format "refs/tags/${RELEASE_TAG}" >/dev/null 2>&1 || \
    die "release_tag is not a valid Git tag name"
  [ "${RELEASE_COMMIT}" = "${CURRENT_COMMIT}" ] || die "source_commit does not match checkout"
  [ "${DECLARED_BINARY_SHA}" = "${BINARY_SHA}" ] || \
    die "binary_sha256 does not match the immutable ceremony copy of ${SOURCE_BINARY}"
  [ "${DECLARED_BINARY_GOOS}" = "${BINARY_GOOS}" ] || \
    die "binary_goos does not match the immutable release binary"
  [ "${DECLARED_BINARY_GOARCH}" = "${BINARY_GOARCH}" ] || \
    die "binary_goarch does not match the immutable release binary"
  [ "${BINARY_GOOS}" = "linux" ] || die "real ceremony requires a Linux release binary"
  case "${BINARY_GOARCH}" in amd64|arm64) ;; *) die "real ceremony requires an amd64 or arm64 binary" ;; esac
  [ "$(go env GOHOSTOS)" = "${BINARY_GOOS}" ] && [ "$(go env GOHOSTARCH)" = "${BINARY_GOARCH}" ] || \
    die "real ceremony must run on the same Linux GOOS/GOARCH as the release binary"
  is_commit "${BINARY_VCS_REVISION}" || die "release binary must embed a 40-hex vcs.revision"
  [ "${BINARY_VCS_REVISION}" = "${RELEASE_COMMIT}" ] || \
    die "release binary vcs.revision does not match source_commit"
  [ "${BINARY_VCS_MODIFIED}" = "false" ] || die "release binary was built from a modified worktree"
  verify_clean_release_checkout
  RELEASE_TAG_REF="refs/tags/${RELEASE_TAG}"
  [ "$(git -C "${PROJECT_ROOT}" cat-file -t "${RELEASE_TAG_REF}" 2>/dev/null || true)" = "tag" ] \
    || die "release_tag must exist as an annotated tag"
  [ "$(git -C "${PROJECT_ROOT}" rev-list -n 1 "${RELEASE_TAG_REF}")" = "${CURRENT_COMMIT}" ] \
    || die "release_tag does not point to source_commit"
  TAG_VERIFY_OUTPUT="$(git -C "${PROJECT_ROOT}" \
    -c gpg.format=openpgp -c gpg.program=gpg -c gpg.openpgp.program=gpg \
    verify-tag --raw "${RELEASE_TAG_REF}" 2>&1)" || \
    die "release_tag signature verification failed"
  TAG_VALID_SIG_COUNT="$(printf '%s\n' "${TAG_VERIFY_OUTPUT}" | awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {count++} END {print count + 0}')"
  [ "${TAG_VALID_SIG_COUNT}" = "1" ] || die "release_tag must produce exactly one OpenPGP VALIDSIG fingerprint"
  TAG_SIGNER_FINGERPRINT="$(printf '%s\n' "${TAG_VERIFY_OUTPUT}" \
    | awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {print tolower($3); exit}')"
  is_fingerprint "${TAG_SIGNER_FINGERPRINT}" || die "release_tag VALIDSIG fingerprint is malformed"
  [ "${TAG_SIGNER_FINGERPRINT}" = "${AUTHORIZED_TAG_SIGNER_FINGERPRINT}" ] || \
    die "release_tag was not signed by the independently authorized fingerprint"
  build_signed_artifact_auditor
  verify_clean_release_checkout
else
  GENESIS_TIME="${DRILL_GENESIS_TIME}"
  VALIDATOR_MONIKER="zerone-2-custodian"
  CUSTODY_DISCLOSURE="DRILL ONLY: public fixture keys; never deploy"
  RELEASE_COMMIT="${CURRENT_COMMIT}"
  RELEASE_TAG="DRILL-NOT-A-RELEASE"
  TAG_SIGNER_FINGERPRINT="DRILL-NOT-A-SIGNER"
  DECLARED_BINARY_SHA="${BINARY_SHA}"
fi

info "initializing keyless coordinator state"
"${BINARY}" init zerone-2-coordinator --chain-id "${CHAIN_ID}" --home "${HOME_DIR}" >/dev/null 2>&1
patch_genesis '.genesis_time = $time | .initial_height = "1"' --arg time "${GENESIS_TIME}"

if [ "${MODE}" = "drill" ]; then
  info "deriving explicit public drill fixtures"
  KR=(--keyring-backend test --home "${HOME_DIR}")
  printf '%s\n' "${DRILL_MNEMONIC}" \
    | "${BINARY}" keys add validator --recover --account 201 "${KR[@]}" >/dev/null 2>&1
  printf '%s\n' "${DRILL_MNEMONIC}" \
    | "${BINARY}" keys add operations --recover --account 202 "${KR[@]}" >/dev/null 2>&1
  VALIDATOR_ADDRESS="$(${BINARY} keys show validator -a "${KR[@]}")"
  OPS_ADDRESS="$(${BINARY} keys show operations -a "${KR[@]}")"
  go build -o "${SCRATCH}/ceremony-inject" "${PROJECT_ROOT}/tools/ceremony-inject"
  "${SCRATCH}/ceremony-inject" drill-consensus-key zerone-2-public-drill \
    "${HOME_DIR}/config/priv_validator_key.json" >/dev/null 2>&1
fi

[ "${VALIDATOR_ADDRESS}" != "${OPS_ADDRESS}" ] || die "validator and operations addresses must differ"

info "adding the only two genesis balances"
"${BINARY}" add-genesis-account "${VALIDATOR_ADDRESS}" "${VALIDATOR_BALANCE_UZRN}uzrn" \
  --home "${HOME_DIR}" >/dev/null
"${BINARY}" add-genesis-account "${OPS_ADDRESS}" "${OPS_BALANCE_UZRN}uzrn" \
  --home "${HOME_DIR}" >/dev/null

patch_genesis '
  .app_state.auth.accounts = [ .app_state.auth.accounts[] |
    if (. ["@type"] == "/cosmos.auth.v1beta1.BaseAccount" and .address == $validator) then
      {
        "@type": "/cosmos.vesting.v1beta1.PermanentLockedAccount",
        "base_vesting_account": {
          "base_account": {
            "address": $validator,
            "pub_key": null,
            "account_number": .account_number,
            "sequence": .sequence
          },
          "original_vesting": [{"denom":"uzrn","amount":$bond}],
          "delegated_free": [],
          "delegated_vesting": [],
          "end_time": "0"
        }
      }
    else . end
  ]
' --arg validator "${VALIDATOR_ADDRESS}" --arg bond "${SELF_BOND_UZRN}"

info "applying the zerone-2 protocol-dark profile"
patch_genesis '
  .consensus.params.block.max_bytes = "4194304"
  | .consensus.params.block.max_gas = "33333333"
  | .consensus.params.evidence.max_age_num_blocks = "100000"
  | .consensus.params.evidence.max_age_duration = "172800000000000"
  | .consensus.params.evidence.max_bytes = "1048576"
  | .consensus.params.validator.pub_key_types = ["ed25519"]
  | .consensus.params.version.app = "0"
  | .consensus.params.abci.vote_extensions_enable_height = "0"

  | .app_state.staking.params.unbonding_time = "1814400s"
  | .app_state.staking.params.max_validators = 33
  | .app_state.staking.params.max_entries = 7
  | .app_state.staking.params.historical_entries = 10000
  | .app_state.staking.params.bond_denom = "uzrn"
  | .app_state.staking.params.min_commission_rate = "0.050000000000000000"

  | .app_state.gov.params.min_deposit = [{"denom":"uzrn","amount":$gov_deposit}]
  | .app_state.gov.params.expedited_min_deposit = [{"denom":"uzrn","amount":$gov_expedited}]
  | .app_state.gov.params.max_deposit_period = "259200s"
  | .app_state.gov.params.voting_period = "259200s"
  | .app_state.gov.params.expedited_voting_period = "86400s"
  | .app_state.gov.params.min_initial_deposit_ratio = "0.250000000000000000"

  | .app_state.zerone_staking.params.max_validators = 33
  | .app_state.zerone_staking.params.min_self_delegation = $custom_stake
  | .app_state.zerone_staking.params.min_stake_for_verification = $custom_stake
  | .app_state.zerone_staking.validators = []
  | .app_state.zerone_staking.delegations = []
  | .app_state.zerone_staking.unbonding_entries = []
  | .app_state.zerone_staking.unbonding_seq = 0

  | .app_state.knowledge.params.min_verifiers = "3"
  | .app_state.knowledge.params.min_headcount_agreement = "3"
  | .app_state.knowledge.params.min_review_fee = $cap_plus_one
  | .app_state.knowledge.params.min_challenge_stake = $cap_plus_one
  | .app_state.knowledge.params.bootstrap_fund_enabled = false
  | .app_state.knowledge.params.demand_tracking_enabled = false
  | .app_state.knowledge.params.vindication_refund_enabled = false
  | .app_state.knowledge.params.verification_reward = "0"
  | .app_state.knowledge.params.demand_bounty_base_reward = "0"
  | .app_state.knowledge.params.demand_bounty_per_query_bonus = "0"
  | .app_state.knowledge.params.probe_bounty_mint_per_block = "0"
  | .app_state.knowledge.params.invitation_bonus_amount = "0"
  | .app_state.knowledge.params.training_fund_base_reward = "0"
  | .app_state.knowledge.params.contribution_challenge_bond = $cap_plus_one
  | .app_state.knowledge.params.contribution_challenge_reward_multiplier_bps = "1000000"
  | .app_state.knowledge.params.guardian_addresses = []
  | .app_state.knowledge.bootstrap_fund_allocation = "0"
  | .app_state.knowledge.training_fund_allocation = "0"

  | .app_state.vesting_rewards.params.block_reward = "0"
  | .app_state.vesting_rewards.params.floor_reward = "0"
  | .app_state.vesting_rewards.params.empty_block_reward_rate = 0
  | .app_state.vesting_rewards.params.min_validators_for_full_reward = 22
  | .app_state.vesting_rewards.params.initial_fund_balance = "0"
  | .app_state.vesting_rewards.params.founder_share_bps = 0
  | .app_state.vesting_rewards.params.founder_address = ""
  | .app_state.vesting_rewards.params.vesting_enabled = false
  | .app_state.vesting_rewards.params.knowledge_coupling_target_bps = 0
  | .app_state.vesting_rewards.params.knowledge_coupling_floor_bps = 0

  | .app_state.claiming_pot.params.bootstrap_registrar = ""
  | .app_state.claiming_pot.pots = []
  | .app_state.claiming_pot.claims = []

  | .app_state.ibc.client_genesis.params.allowed_clients = ["09-localhost"]
  | .app_state.ibc.client_genesis.clients = []
  | .app_state.ibc.client_genesis.clients_consensus = []
  | .app_state.ibc.client_genesis.clients_metadata = []
  | .app_state.ibc.client_genesis.create_localhost = false
  | .app_state.ibc.connection_genesis.connections = []
  | .app_state.ibc.connection_genesis.client_connection_paths = []
  | .app_state.ibc.channel_genesis.channels = []
  | .app_state.transfer.params.send_enabled = false
  | .app_state.transfer.params.receive_enabled = false
  | .app_state.interchainaccounts.controller_genesis_state.params.controller_enabled = false
  | .app_state.interchainaccounts.host_genesis_state.params.host_enabled = false
  | .app_state.interchainaccounts.host_genesis_state.params.allow_messages = []
  | .app_state.ibcratelimit.params.enabled = true
  | .app_state.ibcratelimit.rate_limits = []

  | .app_state.substrate_bridge.adapters = []
  | .app_state.substrate_bridge.params.max_pending_claims_per_attestation = 1000
  | .app_state.substrate_bridge.params.per_pending_claim_bond_uzrn = "22200"
  | .app_state.substrate_bridge.params.attestation_min_bond_uzrn = "22200000"
  | .app_state.substrate_bridge.params.pending_claim_rejection_threshold_bps = 1000
  | .app_state.substrate_bridge.params.min_verified_ratio_for_settle_bps = 6667
  | .app_state.substrate_bridge.params.witness_reward_challenge_window_blocks = "274176"

  | .app_state.emergency.params.genesis_council = []
  | .app_state.emergency.params.council_expiry_block = 0
  | .app_state.emergency.params.min_distinct_voters = 4
  | .app_state.emergency.params.min_guardian_stake = $cap_plus_one
  | .app_state.emergency.params.max_revert_depth = 1111

  | .app_state.alignment.params.enabled = false
  | .app_state.alignment.state.enabled = false
  | .app_state.alignment.observations = []
  | .app_state.alignment.scores = []
  | .app_state.alignment.health_indices = []
  | .app_state.alignment.corrections = []
  | .app_state.counterexamples.params.proposals_enabled = false
  | .app_state.counterexamples.counterexamples = []
  | .app_state.counterexamples.validations = []

  | .app_state.liquiditypool.params.max_pools = 3
  | .app_state.liquiditypool.params.min_initial_liquidity = $cap_plus_one
  | .app_state.liquiditypool.params.protocol_fee_bps = 0
  | .app_state.liquiditypool.params.billing_quote_denoms = []
  | .app_state.liquiditypool.pools = []
  | .app_state.liquiditypool.twap_accumulators = []
' \
  --arg cap_plus_one "${HARD_CAP_PLUS_ONE_UZRN}" \
  --arg custom_stake "${CUSTOM_STAKE_UZRN}" \
  --arg gov_deposit "${GOV_MIN_DEPOSIT_UZRN}" \
  --arg gov_expedited "${GOV_EXPEDITED_DEPOSIT_UZRN}"

if [ "${MODE}" = "drill" ]; then
  info "creating deterministic public-fixture gentx"
  "${BINARY}" genesis gentx validator "${SELF_BOND_UZRN}uzrn" \
    --chain-id "${CHAIN_ID}" --home "${HOME_DIR}" --keyring-backend test \
    --moniker "${VALIDATOR_MONIKER}" --ip 127.0.0.1 --node-id "${DRILL_NODE_ID}" \
    --commission-rate 0.05 --commission-max-rate 0.20 \
    --commission-max-change-rate 0.01 --min-self-delegation 1 \
    --output-document "${GENTX_DIR}/gentx.json" >/dev/null 2>&1
else
  cp "${GENTX_INPUT}" "${GENTX_DIR}/gentx.json"
fi

GENTX="${GENTX_DIR}/gentx.json"

jq -e --arg moniker "${VALIDATOR_MONIKER}" --arg bond "${SELF_BOND_UZRN}" '
  (.body.messages | length) == 1
  and .body.messages[0]["@type"] == "/cosmos.staking.v1beta1.MsgCreateValidator"
  and .body.messages[0].description.moniker == $moniker
  and .body.messages[0].value == {"denom":"uzrn","amount":$bond}
  and .body.messages[0].commission.rate == "0.050000000000000000"
  and .body.messages[0].commission.max_rate == "0.200000000000000000"
  and .body.messages[0].commission.max_change_rate == "0.010000000000000000"
  and .body.messages[0].min_self_delegation == "1"
  and (.auth_info.signer_infos | length) == 1
  and (.signatures | length) == 1
  and (.signatures[0] | type == "string" and length > 0)
' "${GENTX}" >/dev/null || die "gentx does not match the zerone-2 validator contract"

VALOPER="$(jq -r '.body.messages[0].validator_address' "${GENTX}")"
GENTX_ACCOUNT="$(${BINARY} debug addr "${VALOPER}" | awk '/Bech32 Acc:/ {print $3}')"
[ "${GENTX_ACCOUNT}" = "${VALIDATOR_ADDRESS}" ] \
  || die "gentx validator ${GENTX_ACCOUNT} does not match ${VALIDATOR_ADDRESS}"

info "collecting the one signed gentx and validating genesis"
"${BINARY}" genesis collect-gentxs --gentx-dir "${GENTX_DIR}" --home "${HOME_DIR}" >/dev/null 2>&1
"${BINARY}" genesis validate "${GENESIS}" >/dev/null 2>&1 || die "zeroned rejected genesis"

# Hash the exact transaction object embedded in the final genesis. A hash of
# the input gentx file would not bind collection-time normalization or edits.
EMBEDDED_GENTX_CANONICAL="${SCRATCH}/embedded-gentx.canonical.json"
jq -cS -j '.app_state.genutil.gen_txs[0]' "${GENESIS}" > "${EMBEDDED_GENTX_CANONICAL}" \
  || die "could not canonicalize the embedded gentx"
[ -s "${EMBEDDED_GENTX_CANONICAL}" ] || die "embedded gentx canonical form is empty"
GENTX_SHA="$(sha256_file "${EMBEDDED_GENTX_CANONICAL}")"

jq -e '.chain_id == "zerone-2"' "${GENESIS}" >/dev/null \
  || die "post-collection chain ID invariant failed"
jq -e '(.initial_height | tostring) == "1"' "${GENESIS}" >/dev/null \
  || die "post-collection initial height invariant failed"
jq -e '(.app_state.genutil.gen_txs | length) == 1' "${GENESIS}" >/dev/null \
  || die "post-collection gentx count invariant failed"
jq -e '(.app_state.bank.balances | length) == 2' "${GENESIS}" >/dev/null \
  || die "post-collection balance count invariant failed"
jq -e --arg supply "${TOTAL_SUPPLY_UZRN}" '
  ([.app_state.bank.supply[] | select(.denom == "uzrn") | .amount] == [$supply])
  and ([.app_state.bank.supply[] | select(.denom != "uzrn")] | length == 0)
' "${GENESIS}" >/dev/null || die "post-collection supply invariant failed"
jq -e --arg validator "${VALIDATOR_ADDRESS}" --arg ops "${OPS_ADDRESS}" '
  ([.app_state.bank.balances[].address] | sort) == ([$validator,$ops] | sort)
' "${GENESIS}" >/dev/null || die "post-collection balance owner invariant failed"
jq -e --arg validator "${VALIDATOR_ADDRESS}" --arg locked "${SELF_BOND_UZRN}" '
  [.app_state.auth.accounts[]
    | select(.["@type"] == "/cosmos.vesting.v1beta1.PermanentLockedAccount")
    | select(.base_vesting_account.base_account.address == $validator)
    | .base_vesting_account.original_vesting[0].amount] == [$locked]
' "${GENESIS}" >/dev/null || die "post-collection permanent lock invariant failed"

# Genesis and public manifest are the only JSON artifacts copied out of the
# private scratch directory.
cp "${GENESIS}" "${ARTIFACT_DIR}/genesis.json"
GENESIS_SHA="$(sha256_file "${ARTIFACT_DIR}/genesis.json")"
printf '%s  genesis.json\n' "${GENESIS_SHA}" > "${ARTIFACT_DIR}/genesis.sha256"

CONSENSUS_PUBKEY="$(jq -c '.body.messages[0].pubkey' "${GENTX}")"
NODE_ID="$(jq -r '.body.memo | split("@")[0]' "${GENTX}")"
[[ "${NODE_ID}" =~ ^[0-9a-f]{40}$ ]] || \
  die "gentx memo must begin with the new 40-hex P2P node ID"
[ "$(sha256_file "${BINARY}")" = "${BINARY_SHA}" ] || \
  die "immutable ceremony binary changed before manifest publication"

jq -n \
  --arg schema "${MANIFEST_SCHEMA}" \
  --arg mode "${MODE}" \
  --arg chain "${CHAIN_ID}" \
  --arg time "${GENESIS_TIME}" \
  --arg genesis_sha "${GENESIS_SHA}" \
  --arg commit "${RELEASE_COMMIT}" \
  --arg tag "${RELEASE_TAG}" \
  --arg tag_signer_fingerprint "${TAG_SIGNER_FINGERPRINT}" \
  --arg binary_sha "${BINARY_SHA}" \
  --arg binary_version "${BINARY_VERSION}" \
  --arg binary_goos "${BINARY_GOOS}" \
  --arg binary_goarch "${BINARY_GOARCH}" \
  --arg validator "${VALIDATOR_ADDRESS}" \
  --arg valoper "${VALOPER}" \
  --argjson consensus_pubkey "${CONSENSUS_PUBKEY}" \
  --arg node_id "${NODE_ID}" \
  --arg ops "${OPS_ADDRESS}" \
  --arg gentx_sha "${GENTX_SHA}" \
  --arg custody "${CUSTODY_DISCLOSURE}" \
  --arg supply "${TOTAL_SUPPLY_UZRN}" \
  --arg bond "${SELF_BOND_UZRN}" '
  {
    schema: $schema,
    mode: $mode,
    chain_id: $chain,
    genesis_time: $time,
    genesis_sha256: $genesis_sha,
    release: {
      source_commit: $commit,
      tag: $tag,
      tag_signer_fingerprint: $tag_signer_fingerprint,
      binary_sha256: $binary_sha,
      binary_version: $binary_version,
      binary_goos: $binary_goos,
      binary_goarch: $binary_goarch
    },
    trust_model: {
      genesis_validators: 1,
      byzantine_fault_tolerance: 0,
      disclosure: $custody
    },
    supply_uzrn: $supply,
    validator: {
      account_address: $validator,
      operator_address: $valoper,
      consensus_pubkey: $consensus_pubkey,
      node_id: $node_id,
      self_bond_uzrn: $bond,
      gentx_sha256: $gentx_sha
    },
    operations: {account_address: $ops},
    activations: {
      vote_extensions: "disabled",
      pot: "not live",
      ibc: "external-disabled; localhost-only",
      substrate_bridge: "disabled",
      claiming: "disabled"
    }
  }
' > "${ARTIFACT_DIR}/network-manifest.json"

cat > "${ARTIFACT_DIR}/GENESIS-MANIFEST.md" <<EOF
# Zerone Genesis Manifest — zerone-2

- Ceremony mode: **${MODE}**
- Genesis time: ${GENESIS_TIME}
- Genesis SHA-256: ${GENESIS_SHA}
- Source commit: ${RELEASE_COMMIT}
- Release tag: ${RELEASE_TAG}
- Release tag signer fingerprint: ${TAG_SIGNER_FINGERPRINT}
- Binary SHA-256: ${BINARY_SHA}
- Binary version: ${BINARY_VERSION}
- Binary target: ${BINARY_GOOS}/${BINARY_GOARCH}
- Trust model: ${CUSTODY_DISCLOSURE}

## Exact supply

Total: **${TOTAL_SUPPLY_UZRN} uzrn (13,555 ZRN)**.

- Validator ${VALIDATOR_ADDRESS}: ${VALIDATOR_BALANCE_UZRN} uzrn, of which
  ${SELF_BOND_UZRN} uzrn is permanent-locked self-bond and 222000000 uzrn is
  liquid operations gas.
- Operations ${OPS_ADDRESS}: ${OPS_BALANCE_UZRN} uzrn.
- No other positive genesis balance or module allocation.

## Validator

- SDK operator: ${VALOPER}
- P2P node ID declared in gentx: ${NODE_ID}
- Gentx SHA-256: ${GENTX_SHA}
- Genesis validators: 1; Byzantine fault tolerance: 0.

## Dark launch

Vote extensions and PoT settlement are not live. IBC/ICA, transfers, the
substrate bridge, knowledge admission/rewards, claiming, alignment corrections,
counterexamples, and liquidity creation are latched off in genesis and enforced
by the mandatory artifact audit.
EOF

# Assert no custody-shaped JSON field escaped into public artifacts.
jq -e '
  [paths(scalars) as $p
    | ($p | map(tostring) | join(".") | ascii_downcase)
    | select(test("mnemonic|private|priv_key|password|passphrase|secret|seed"))]
  | length == 0
' "${ARTIFACT_DIR}/genesis.json" "${ARTIFACT_DIR}/network-manifest.json" >/dev/null \
  || die "custody-shaped field escaped into public artifacts"

chmod 0644 "${ARTIFACT_DIR}/genesis.json" "${ARTIFACT_DIR}/genesis.sha256" \
  "${ARTIFACT_DIR}/network-manifest.json" "${ARTIFACT_DIR}/GENESIS-MANIFEST.md"

info "running the mandatory zerone-2 artifact audit"
if [ "${MODE}" = "real" ]; then
  verify_clean_release_checkout
  "${SIGNED_AUDITOR}" --artifact-dir "${ARTIFACT_DIR}" --required-mode real || \
    die "mandatory signed-source artifact audit failed"
  verify_clean_release_checkout
else
  (cd "${PROJECT_ROOT}" && \
    GOENV=off GOWORK=off GOFLAGS='' GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 \
    go run -mod=readonly ./tools/zerone2-artifact-audit \
    --artifact-dir "${ARTIFACT_DIR}" --required-mode drill) || \
    die "mandatory artifact audit failed"
fi
[ "$(sha256_file "${BINARY}")" = "${BINARY_SHA}" ] || \
  die "immutable ceremony binary changed after final audit"

# Publish only after every gate passes. Scratch lives beside the destination,
# so the reservation and rename are same-filesystem. The helper atomically
# publishes the complete directory or fails without clobbering/nesting.
[ "${MODE}" != "real" ] || verify_clean_release_checkout
"${PUBLISH_HELPER}" "${ARTIFACT_DIR}" "${OUT_DIR}" || \
  die "could not atomically publish artifacts; destination may already exist: ${OUT_DIR}"

ok "public artifacts written to ${OUT_DIR}"
ok "genesis SHA-256 ${GENESIS_SHA}"
[ "${MODE}" != "drill" ] || printf '  !! DRILL FIXTURES ARE PUBLIC AND MUST NEVER BE DEPLOYED\n'
