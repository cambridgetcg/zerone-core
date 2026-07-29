#!/usr/bin/env bash
# Build the zerone-2 runtime from one audited real artifact set and the exact
# release binary named by that artifact set. No repository source is sent to
# Docker and no binary is rebuilt inside the image context.
set -euo pipefail
export LC_ALL=C
export GIT_NO_REPLACE_OBJECTS=1
umask 077

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)

SOURCE_ARTIFACT_DIR="${1:?usage: build-image.sh <audited-artifact-dir> <audited-release-binary> <image-ref>}"
SOURCE_RELEASE_BINARY="${2:?usage: build-image.sh <audited-artifact-dir> <audited-release-binary> <image-ref>}"
IMAGE_REF="${3:?usage: build-image.sh <audited-artifact-dir> <audited-release-binary> <image-ref>}"
[ "$#" -eq 3 ] || { printf 'usage: build-image.sh <audited-artifact-dir> <audited-release-binary> <image-ref>\n' >&2; exit 2; }

EXPECTED_MANIFEST_SCHEMA="zerone-2-network-manifest-v2"
SNAPSHOT=""
CONTEXT=""

die() { printf 'zerone-2 build-image: ERROR: %s\n' "$*" >&2; exit 1; }
sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
is_fingerprint() { [[ "$1" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]]; }
binary_build_setting() {
  local name="$1"
  go version -m "${RELEASE_BINARY}" 2>/dev/null \
    | awk -v name="${name}" '$1 == "build" && index($2, name "=") == 1 {sub(name "=", "", $2); print $2; exit}'
}
install_git_blob() {
  local relative="$1" destination="$2" requested_mode="$3" listing git_mode
  listing=$(git -C "${ROOT}" ls-tree "${SOURCE_COMMIT}" -- "${relative}") || \
    die "could not inspect ${relative} at source commit"
  [ -n "${listing}" ] || die "source commit omits runtime support file: ${relative}"
  [ "$(printf '%s\n' "${listing}" | wc -l | tr -d ' ')" = "1" ] || \
    die "runtime support path resolved ambiguously: ${relative}"
  git_mode=$(printf '%s\n' "${listing}" | awk '{print $1}')
  case "${git_mode}" in
    100644|100755) ;;
    *) die "runtime support input is not a regular Git file: ${relative}" ;;
  esac
  [ "$(git -C "${ROOT}" cat-file -t "${SOURCE_COMMIT}:${relative}")" = "blob" ] || \
    die "runtime support input is not a Git blob: ${relative}"
  mkdir -p "$(dirname "${destination}")"
  git -C "${ROOT}" show "${SOURCE_COMMIT}:${relative}" > "${destination}" || \
    die "could not materialize runtime support file: ${relative}"
  chmod "${requested_mode}" "${destination}"
}

build_signed_artifact_auditor() {
  local source_root="${SNAPSHOT}/signed-auditor-source" relative count=0
  SIGNED_AUDITOR="${SNAPSHOT}/signed-zerone2-artifact-audit"
  mkdir -m 0700 "${source_root}"
  while IFS= read -r -d '' relative; do
    install_git_blob "${relative}" "${source_root}/${relative}" 0644
    count=$((count + 1))
  done < <(git -C "${ROOT}" ls-tree -r -z --name-only "${SOURCE_COMMIT}" -- \
    go.mod go.sum app x docs/swagger-ui tools/zerone2-artifact-audit/main.go)
  [ "${count}" -gt 5 ] || die "signed auditor source allowlist was unexpectedly empty"
  [ -f "${source_root}/tools/zerone2-artifact-audit/main.go" ] || \
    die "signed source commit omits the artifact auditor"
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

verify_clean_source_checkout() {
  [ "$(git -C "${ROOT}" rev-parse HEAD)" = "${SOURCE_COMMIT}" ] || \
    die "checked-out HEAD changed during the release build"
  [ -z "$(git -C "${ROOT}" status --porcelain --untracked-files=all)" ] || \
    die "release build requires a clean worktree"
}

reject_git_history_overrides() {
  local git_dir
  [ -z "$(git -C "${ROOT}" for-each-ref --format='%(refname)' refs/replace/)" ] || \
    die "replacement Git refs are forbidden for release builds"
  git_dir=$(git -C "${ROOT}" rev-parse --absolute-git-dir) || \
    die "could not resolve repository Git directory"
  if [ -e "${git_dir}/info/grafts" ] || [ -L "${git_dir}/info/grafts" ]; then
    die "legacy Git grafts are forbidden for release builds"
  fi
}

cleanup() {
  local directory
  for directory in "${CONTEXT}" "${SNAPSHOT}"; do
    [ -z "${directory}" ] || chmod -R u+w "${directory}" 2>/dev/null || true
    [ -z "${directory}" ] || rm -rf "${directory}"
  done
}
trap cleanup EXIT INT TERM

command -v docker >/dev/null 2>&1 || die "docker is required"
command -v git >/dev/null 2>&1 || die "git is required"
command -v go >/dev/null 2>&1 || die "go is required to inspect release build metadata"
command -v jq >/dev/null 2>&1 || die "jq is required"
[[ "${IMAGE_REF}" =~ ^[A-Za-z0-9][A-Za-z0-9._/:@-]{0,255}$ ]] || die "image reference contains unsafe characters"
[ -d "${SOURCE_ARTIFACT_DIR}" ] && [ ! -L "${SOURCE_ARTIFACT_DIR}" ] || \
  die "artifact directory must be a non-symlink directory"
[ -f "${SOURCE_RELEASE_BINARY}" ] && [ ! -L "${SOURCE_RELEASE_BINARY}" ] && \
  [ -x "${SOURCE_RELEASE_BINARY}" ] || \
  die "release binary must be an executable regular non-symlink file"

# Freeze the complete four-file public artifact set and release binary before
# parsing or auditing anything. Every later read uses only these private copies,
# so a caller cannot swap a coherently forged set after the policy audit.
PUBLIC_ARTIFACT_NAMES=(
  genesis.json
  genesis.sha256
  network-manifest.json
  GENESIS-MANIFEST.md
)
[ "$(find "${SOURCE_ARTIFACT_DIR}" -mindepth 1 -maxdepth 1 -print | wc -l | tr -d ' ')" = "4" ] || \
  die "artifact directory must contain exactly the four public ceremony files"
SNAPSHOT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-release-snapshot.XXXXXX")
mkdir -m 0700 "${SNAPSHOT}/artifacts" "${SNAPSHOT}/release"
for name in "${PUBLIC_ARTIFACT_NAMES[@]}"; do
  source_file="${SOURCE_ARTIFACT_DIR}/${name}"
  [ -f "${source_file}" ] && [ ! -L "${source_file}" ] || \
    die "public artifact must be a regular non-symlink file: ${name}"
  cp -P "${source_file}" "${SNAPSHOT}/artifacts/${name}" || \
    die "could not freeze public artifact: ${name}"
  [ -f "${SNAPSHOT}/artifacts/${name}" ] && \
    [ ! -L "${SNAPSHOT}/artifacts/${name}" ] || \
    die "frozen public artifact is not a regular file: ${name}"
  chmod 0400 "${SNAPSHOT}/artifacts/${name}"
done
cp -P "${SOURCE_RELEASE_BINARY}" "${SNAPSHOT}/release/zeroned" || \
  die "could not freeze release binary"
[ -f "${SNAPSHOT}/release/zeroned" ] && [ ! -L "${SNAPSHOT}/release/zeroned" ] || \
  die "frozen release binary is not a regular file"
chmod 0500 "${SNAPSHOT}/release/zeroned"

ARTIFACT_DIR="${SNAPSHOT}/artifacts"
RELEASE_BINARY="${SNAPSHOT}/release/zeroned"
GENESIS_FILE="${ARTIFACT_DIR}/genesis.json"
PUBLIC_MANIFEST="${ARTIFACT_DIR}/network-manifest.json"

MANIFEST_SCHEMA=$(jq -er '.schema' "${PUBLIC_MANIFEST}")
MANIFEST_MODE=$(jq -er '.mode' "${PUBLIC_MANIFEST}")
CHAIN_ID=$(jq -er '.chain_id' "${PUBLIC_MANIFEST}")
GENESIS_SHA256=$(jq -er '.genesis_sha256' "${PUBLIC_MANIFEST}")
SOURCE_COMMIT=$(jq -er '.release.source_commit' "${PUBLIC_MANIFEST}")
RELEASE_TAG=$(jq -er '.release.tag' "${PUBLIC_MANIFEST}")
TAG_SIGNER_FINGERPRINT=$(jq -er '.release.tag_signer_fingerprint' "${PUBLIC_MANIFEST}")
EXPECTED_BINARY_SHA256=$(jq -er '.release.binary_sha256' "${PUBLIC_MANIFEST}")
EXPECTED_BINARY_VERSION=$(jq -er '.release.binary_version' "${PUBLIC_MANIFEST}")
BINARY_GOOS=$(jq -er '.release.binary_goos' "${PUBLIC_MANIFEST}")
BINARY_GOARCH=$(jq -er '.release.binary_goarch' "${PUBLIC_MANIFEST}")
VALIDATOR_NODE_ID=$(jq -er '.validator.node_id' "${PUBLIC_MANIFEST}")
NETWORK_MANIFEST_SHA256=$(sha256_file "${PUBLIC_MANIFEST}")

[ "${MANIFEST_SCHEMA}" = "${EXPECTED_MANIFEST_SCHEMA}" ] || die "network manifest schema is not v2"
[ "${MANIFEST_MODE}" = "real" ] || die "runtime images require a real ceremony manifest"
[ "${CHAIN_ID}" = "zerone-2" ] || die "network manifest chain_id must be exactly zerone-2"
[[ "${GENESIS_SHA256}" =~ ^[0-9a-f]{64}$ ]] || die "genesis hash is malformed"
[ "$(sha256_file "${GENESIS_FILE}")" = "${GENESIS_SHA256}" ] || die "genesis hash does not match the manifest"
[[ "${SOURCE_COMMIT}" =~ ^[0-9a-f]{40}$ ]] || die "source commit is malformed"
[[ "${RELEASE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$ ]] || die "release tag contains unsafe characters"
is_fingerprint "${TAG_SIGNER_FINGERPRINT}" || die "manifest tag signer fingerprint is malformed"
[[ "${EXPECTED_BINARY_SHA256}" =~ ^[0-9a-f]{64}$ ]] || die "release binary hash is malformed"
[[ "${EXPECTED_BINARY_VERSION}" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$ ]] || \
  die "release binary version contains unsafe characters"
[ "${BINARY_GOOS}" = "linux" ] || die "release binary GOOS must be linux"
case "${BINARY_GOARCH}" in amd64|arm64) ;; *) die "release binary GOARCH must be amd64 or arm64" ;; esac
[[ "${VALIDATOR_NODE_ID}" =~ ^[0-9a-f]{40}$ ]] || die "validator node ID must be 40 lowercase hex characters"
[ "${VALIDATOR_NODE_ID}" != "0000000000000000000000000000000000000000" ] || die "zero validator node ID is forbidden"
[ "${VALIDATOR_NODE_ID}" != "2222222222222222222222222222222222222222" ] || die "drill validator node ID is forbidden"

AUTHORIZED_FINGERPRINT="${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT:-}"
is_fingerprint "${AUTHORIZED_FINGERPRINT}" || \
  die "ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT must be 40 or 64 lowercase hex"
[ "${AUTHORIZED_FINGERPRINT}" = "${TAG_SIGNER_FINGERPRINT}" ] || \
  die "manifest tag signer is not the independently authorized fingerprint"

HEAD_COMMIT=$(git -C "${ROOT}" rev-parse HEAD)
[ "${HEAD_COMMIT}" = "${SOURCE_COMMIT}" ] || die "manifest source commit does not equal checked-out HEAD"
reject_git_history_overrides
verify_clean_source_checkout
git -C "${ROOT}" check-ref-format "refs/tags/${RELEASE_TAG}" >/dev/null 2>&1 || die "release tag is not a valid tag name"
[ "$(git -C "${ROOT}" cat-file -t "refs/tags/${RELEASE_TAG}" 2>/dev/null || true)" = "tag" ] || \
  die "release tag must be an annotated tag"
[ "$(git -C "${ROOT}" rev-list -n 1 "refs/tags/${RELEASE_TAG}")" = "${SOURCE_COMMIT}" ] || \
  die "release tag does not point to the manifest source commit"
TAG_VERIFY_OUTPUT="$(git -C "${ROOT}" \
  -c gpg.format=openpgp -c gpg.program=gpg -c gpg.openpgp.program=gpg \
  verify-tag --raw "refs/tags/${RELEASE_TAG}" 2>&1)" || \
  die "release tag signature verification failed"
TAG_VALID_SIG_COUNT="$(printf '%s\n' "${TAG_VERIFY_OUTPUT}" | awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {count++} END {print count + 0}')"
[ "${TAG_VALID_SIG_COUNT}" = "1" ] || die "release tag must produce exactly one OpenPGP VALIDSIG fingerprint"
ACTUAL_TAG_FINGERPRINT="$(printf '%s\n' "${TAG_VERIFY_OUTPUT}" \
  | awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {print tolower($3); exit}')"
[ "${ACTUAL_TAG_FINGERPRINT}" = "${AUTHORIZED_FINGERPRINT}" ] || \
  die "release tag was not signed by the independently authorized fingerprint"

# Only the clean, signed source commit's auditor may approve the immutable
# snapshot. Its complete local compile closure is materialized from Git blobs;
# no mutable worktree byte or ambient Go overlay can change the approving code.
build_signed_artifact_auditor
verify_clean_source_checkout
"${SIGNED_AUDITOR}" --artifact-dir "${ARTIFACT_DIR}" --required-mode real || \
  die "mandatory real artifact audit failed"
verify_clean_source_checkout

ACTUAL_BINARY_SHA256=$(sha256_file "${RELEASE_BINARY}")
[ "${ACTUAL_BINARY_SHA256}" = "${EXPECTED_BINARY_SHA256}" ] || die "release binary hash does not match the manifest"
ACTUAL_BINARY_GOOS=$(binary_build_setting GOOS)
ACTUAL_BINARY_GOARCH=$(binary_build_setting GOARCH)
ACTUAL_BINARY_VCS_REVISION=$(binary_build_setting vcs.revision)
ACTUAL_BINARY_VCS_MODIFIED=$(binary_build_setting vcs.modified)
[ "${ACTUAL_BINARY_GOOS}" = "${BINARY_GOOS}" ] || die "release binary GOOS does not match the manifest"
[ "${ACTUAL_BINARY_GOARCH}" = "${BINARY_GOARCH}" ] || die "release binary GOARCH does not match the manifest"
[ "${ACTUAL_BINARY_VCS_REVISION}" = "${SOURCE_COMMIT}" ] || die "release binary vcs.revision does not match source commit"
[ "${ACTUAL_BINARY_VCS_MODIFIED}" = "false" ] || die "release binary was built from a modified worktree"

[[ "${RUNTIME_IMAGE:-}" =~ ^[^[:space:]]+@sha256:[0-9a-f]{64}$ ]] || \
  die "RUNTIME_IMAGE must be pinned by sha256 digest"

CONTEXT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-2-image-context.XXXXXX")
mkdir -p "${CONTEXT}/release" "${CONTEXT}/runtime" "${CONTEXT}/public"

install_git_blob deploy/networks/zerone-2/runtime/Dockerfile \
  "${CONTEXT}/Dockerfile" 0644
install_git_blob deploy/networks/zerone-2/runtime/entrypoint.sh \
  "${CONTEXT}/runtime/entrypoint.sh" 0755
install -m 0555 "${RELEASE_BINARY}" "${CONTEXT}/release/zeroned"
install -m 0644 "${GENESIS_FILE}" "${CONTEXT}/public/genesis.json"
install -m 0644 "${PUBLIC_MANIFEST}" "${CONTEXT}/public/network-manifest.json"
[ "$(sha256_file "${CONTEXT}/release/zeroned")" = "${EXPECTED_BINARY_SHA256}" ] || \
  die "release binary changed while entering the sanitized context"
[ "$(sha256_file "${CONTEXT}/public/genesis.json")" = "${GENESIS_SHA256}" ] || \
  die "genesis changed while entering the sanitized context"
[ "$(sha256_file "${CONTEXT}/public/network-manifest.json")" = \
    "${NETWORK_MANIFEST_SHA256}" ] || \
  die "network manifest changed while entering the sanitized context"

printf 'schema=zerone-2-sanitized-image-context-v2\nchain_id=%s\nsource_commit=%s\nrelease_tag=%s\ntag_signer_fingerprint=%s\ngenesis_sha256=%s\nnetwork_manifest_sha256=%s\nbinary_sha256=%s\nbinary_goos=%s\nbinary_goarch=%s\n' \
  "${CHAIN_ID}" "${SOURCE_COMMIT}" "${RELEASE_TAG}" "${TAG_SIGNER_FINGERPRINT}" \
  "${GENESIS_SHA256}" "${NETWORK_MANIFEST_SHA256}" "${EXPECTED_BINARY_SHA256}" \
  "${BINARY_GOOS}" "${BINARY_GOARCH}" > "${CONTEXT}/.sanitized-zerone-2-context-v2"

if find "${CONTEXT}" -type f \( -name '*.mnemonic' -o -name 'node_key.json' -o \
    -name 'priv_validator_key.json' -o -name 'priv_validator_state.json' \) \
    -print -quit | grep -q .; then
  die "custody-shaped file entered the sanitized context"
fi

docker_args=(build
  --platform "linux/${BINARY_GOARCH}"
  --build-arg "RUNTIME_IMAGE=${RUNTIME_IMAGE}"
  --build-arg "SOURCE_COMMIT=${SOURCE_COMMIT}"
  --build-arg "RELEASE_TAG=${RELEASE_TAG}"
  --build-arg "TAG_SIGNER_FINGERPRINT=${TAG_SIGNER_FINGERPRINT}"
  --build-arg "GENESIS_SHA256=${GENESIS_SHA256}"
  --build-arg "NETWORK_MANIFEST_SHA256=${NETWORK_MANIFEST_SHA256}"
  --build-arg "BINARY_SHA256=${EXPECTED_BINARY_SHA256}"
  --build-arg "BINARY_VERSION=${EXPECTED_BINARY_VERSION}"
  --build-arg "BINARY_GOOS=${BINARY_GOOS}"
  --build-arg "BINARY_GOARCH=${BINARY_GOARCH}"
  --build-arg "VALIDATOR_NODE_ID=${VALIDATOR_NODE_ID}"
  --label "org.opencontainers.image.revision=${SOURCE_COMMIT}"
  --label "org.opencontainers.image.version=${EXPECTED_BINARY_VERSION}"
  --label "org.opencontainers.image.ref.name=${RELEASE_TAG}"
  --label "money.zerone.chain-id=${CHAIN_ID}"
  --label "money.zerone.genesis-sha256=${GENESIS_SHA256}"
  --label "money.zerone.network-manifest-sha256=${NETWORK_MANIFEST_SHA256}"
  --label "money.zerone.release-binary-sha256=${EXPECTED_BINARY_SHA256}"
  --label "money.zerone.release-tag-signer=${TAG_SIGNER_FINGERPRINT}"
  --label "money.zerone.binary-platform=${BINARY_GOOS}/${BINARY_GOARCH}"
  --tag "${IMAGE_REF}")
verify_clean_source_checkout
docker "${docker_args[@]}" "${CONTEXT}"

printf 'image=%s\ncommit=%s\ntag=%s\ntag_signer_fingerprint=%s\ngenesis_sha256=%s\nnetwork_manifest_sha256=%s\nbinary_sha256=%s\nbinary_platform=%s/%s\nvalidator_node_id=%s\n' \
  "${IMAGE_REF}" "${SOURCE_COMMIT}" "${RELEASE_TAG}" "${TAG_SIGNER_FINGERPRINT}" \
  "${GENESIS_SHA256}" "${NETWORK_MANIFEST_SHA256}" "${EXPECTED_BINARY_SHA256}" \
  "${BINARY_GOOS}" "${BINARY_GOARCH}" "${VALIDATOR_NODE_ID}"
