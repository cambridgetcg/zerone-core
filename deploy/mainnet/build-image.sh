#!/usr/bin/env bash
# Build the zerone-1 halt/archive runtime from a custody-free explicit context.
set -euo pipefail
umask 077
export GIT_NO_REPLACE_OBJECTS=1

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
MAINNET_DIR="${ROOT}/deploy/mainnet"
IMAGE_REF="${1:?usage: build-image.sh <image-ref>}"
PROFILE="${MAINNET_BUILD_PROFILE:-release}"
GENESIS="${MAINNET_DIR}/artifacts/genesis.json"
EXPECTED_GENESIS_SHA256="c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"
TARGET_PLATFORM="linux/amd64"

die() { printf 'mainnet build-image: ERROR: %s\n' "$*" >&2; exit 1; }
sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

materialize_git_file() {
  local commit="$1" relative="$2" destination="$3" listing mode
  listing=$(git -C "${ROOT}" ls-tree "${commit}" -- "${relative}") || \
    die "could not inspect ${relative} at ${commit}"
  [ -n "${listing}" ] || die "release commit omits required file: ${relative}"
  [ "$(printf '%s\n' "${listing}" | wc -l | tr -d ' ')" = "1" ] || \
    die "release commit resolves ${relative} ambiguously"
  mode=$(printf '%s\n' "${listing}" | awk '{print $1}')
  case "${mode}" in
    100644|100755) ;;
    *) die "release input must be a regular Git blob: ${relative}" ;;
  esac
  mkdir -p "$(dirname "${destination}")"
  git -C "${ROOT}" show "${commit}:${relative}" > "${destination}" || \
    die "could not materialize ${relative} from ${commit}"
  if [ "${mode}" = "100755" ]; then
    chmod 0755 "${destination}"
  else
    chmod 0644 "${destination}"
  fi
}

verify_clean_release_tree() {
  [ "$(git -C "${ROOT}" rev-parse HEAD)" = "${HEAD_COMMIT}" ] || \
    die "HEAD changed during the release build"
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

verify_release_tag() {
  local tag="${ZERONE_RELEASE_TAG:-}"
  local authorized="${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT:-}"
  local tag_ref verify_output valid_count actual_fingerprint

  [[ "${tag}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] || \
    die "release build requires a safe ZERONE_RELEASE_TAG"
  authorized=$(printf '%s' "${authorized}" | tr '[:lower:]' '[:upper:]')
  [[ "${authorized}" =~ ^([0-9A-F]{40}|[0-9A-F]{64})$ ]] || \
    die "release build requires an exact authorized OpenPGP signer fingerprint"
  tag_ref="refs/tags/${tag}"
  git -C "${ROOT}" show-ref --verify --quiet "${tag_ref}" || \
    die "release tag does not exist exactly: ${tag_ref}"
  [ "$(git -C "${ROOT}" cat-file -t "${tag_ref}")" = "tag" ] || \
    die "release tag must be an annotated tag, not a lightweight tag"
  [ "$(git -C "${ROOT}" rev-parse "${tag_ref}^{commit}")" = "${HEAD_COMMIT}" ] || \
    die "release tag does not point to HEAD"
  if ! verify_output=$(git -C "${ROOT}" \
      -c gpg.format=openpgp -c gpg.program=gpg -c gpg.openpgp.program=gpg \
      verify-tag --raw "${tag_ref}" 2>&1); then
    die "release tag signature verification failed"
  fi
  valid_count=$(printf '%s\n' "${verify_output}" | \
    awk '/^\[GNUPG:\] VALIDSIG / { count++ } END { print count + 0 }')
  [ "${valid_count}" = "1" ] || \
    die "release tag must produce exactly one valid signature"
  actual_fingerprint=$(printf '%s\n' "${verify_output}" | \
    awk '/^\[GNUPG:\] VALIDSIG / { print toupper($3) }')
  [ "${actual_fingerprint}" = "${authorized}" ] || \
    die "release tag was not signed by the authorized fingerprint"
  RELEASE_TAG="${tag}"
  RELEASE_SIGNER_FINGERPRINT="${actual_fingerprint}"
}

case "${PROFILE}" in
  release|development) ;;
  *) die "MAINNET_BUILD_PROFILE must be exactly release or development" ;;
esac
command -v docker >/dev/null 2>&1 || die "docker is required"
command -v git >/dev/null 2>&1 || die "git is required"
command -v jq >/dev/null 2>&1 || die "jq is required"

HEAD_COMMIT=$(git -C "${ROOT}" rev-parse HEAD)
[[ "${HEAD_COMMIT}" =~ ^[0-9a-f]{40}$ ]] || die "HEAD must be a 40-hex commit"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git -C "${ROOT}" show -s --format=%ct "${HEAD_COMMIT}")}"
[[ "${SOURCE_DATE_EPOCH}" =~ ^[1-9][0-9]*$ ]] || \
  die "SOURCE_DATE_EPOCH must be a positive integer"

if [ "${PROFILE}" = "release" ]; then
  reject_git_history_overrides
  verify_clean_release_tree
  verify_release_tag
  if [ -n "${VERSION:-}" ] && [ "${VERSION}" != "${RELEASE_TAG}" ]; then
    die "release VERSION must exactly equal ZERONE_RELEASE_TAG"
  fi
  VERSION="${RELEASE_TAG}"
  [[ "${GO_IMAGE:-}" =~ ^[^[:space:]]+@sha256:[0-9a-f]{64}$ ]] || \
    die "release build requires GO_IMAGE pinned by sha256 digest"
  [[ "${RUNTIME_IMAGE:-}" =~ ^[^[:space:]]+@sha256:[0-9a-f]{64}$ ]] || \
    die "release build requires RUNTIME_IMAGE pinned by sha256 digest"
else
  VERSION="${VERSION:-zerone-1-halt-${HEAD_COMMIT:0:12}}"
  RELEASE_TAG="development"
  RELEASE_SIGNER_FINGERPRINT="none"
fi
[[ "${VERSION}" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$ ]] || \
  die "VERSION contains unsafe characters"

CONTEXT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-1-image-context.XXXXXX")
cleanup() { rm -rf "${CONTEXT}"; }
trap cleanup EXIT
mkdir -p "${CONTEXT}/runtime" "${CONTEXT}/public"

# Copy only the explicit source allowlist. Release bytes come from the signed
# commit's immutable Git objects; development mode intentionally uses the worktree.
tracked_count=0
while IFS= read -r -d '' relative; do
  if [ "${PROFILE}" = "release" ]; then
    materialize_git_file "${HEAD_COMMIT}" "${relative}" "${CONTEXT}/${relative}"
  else
    source_file="${ROOT}/${relative}"
    [ -f "${source_file}" ] && [ ! -L "${source_file}" ] || \
      die "tracked build input must be a regular non-symlink file: ${relative}"
    mkdir -p "${CONTEXT}/$(dirname "${relative}")"
    cp "${source_file}" "${CONTEXT}/${relative}"
  fi
  tracked_count=$((tracked_count + 1))
done < <(
  if [ "${PROFILE}" = "release" ]; then
    git -C "${ROOT}" ls-tree -r -z --name-only "${HEAD_COMMIT}" -- \
      go.mod go.sum app cmd x docs/swagger-ui
  else
    git -C "${ROOT}" ls-files -z -- go.mod go.sum app cmd x docs/swagger-ui
  fi
)
[ "${tracked_count}" -gt 2 ] || die "tracked source allowlist was unexpectedly empty"

if [ "${PROFILE}" = "release" ]; then
  materialize_git_file "${HEAD_COMMIT}" deploy/mainnet/Dockerfile "${CONTEXT}/Dockerfile"
  materialize_git_file "${HEAD_COMMIT}" deploy/mainnet/entrypoint.sh \
    "${CONTEXT}/runtime/entrypoint.sh"
  materialize_git_file "${HEAD_COMMIT}" deploy/mainnet/artifacts/genesis.json \
    "${CONTEXT}/public/genesis.json"
else
  [ -f "${GENESIS}" ] && [ ! -L "${GENESIS}" ] || \
    die "public zerone-1 genesis is missing"
  install -m 0644 "${MAINNET_DIR}/Dockerfile" "${CONTEXT}/Dockerfile"
  install -m 0755 "${MAINNET_DIR}/entrypoint.sh" "${CONTEXT}/runtime/entrypoint.sh"
  install -m 0644 "${GENESIS}" "${CONTEXT}/public/genesis.json"
fi
[ "$(jq -er '.chain_id' "${CONTEXT}/public/genesis.json")" = "zerone-1" ] || \
  die "genesis chain_id must be exactly zerone-1"
GENESIS_SHA256=$(sha256_file "${CONTEXT}/public/genesis.json")
[ "${GENESIS_SHA256}" = "${EXPECTED_GENESIS_SHA256}" ] || \
  die "zerone-1 genesis hash does not match the production pin"
printf 'profile=%s\ncommit=%s\ngenesis_sha256=%s\nrelease_tag=%s\nrelease_signer_fingerprint=%s\nplatform=%s\n' \
  "${PROFILE}" "${HEAD_COMMIT}" "${GENESIS_SHA256}" "${RELEASE_TAG}" \
  "${RELEASE_SIGNER_FINGERPRINT}" "${TARGET_PLATFORM}" \
  > "${CONTEXT}/.sanitized-mainnet-context-v1"

if find "${CONTEXT}" -type f \( \
    -name '*.mnemonic' -o -name 'node_key.json' -o \
    -name 'priv_validator_key.json' -o -name 'priv_validator_state.json' \
  \) -print -quit | grep -q .; then
  die "custody-shaped file entered the sanitized context"
fi

docker_args=(build
  --platform "${TARGET_PLATFORM}"
  --build-arg "VERSION=${VERSION}"
  --build-arg "COMMIT=${HEAD_COMMIT}"
  --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}"
  --build-arg "GENESIS_SHA256=${GENESIS_SHA256}"
  --build-arg "BUILD_PROFILE=${PROFILE}"
  --build-arg "RELEASE_TAG=${RELEASE_TAG}"
  --build-arg "RELEASE_SIGNER_FINGERPRINT=${RELEASE_SIGNER_FINGERPRINT}"
  --label "org.opencontainers.image.revision=${HEAD_COMMIT}"
  --label "org.opencontainers.image.version=${VERSION}"
  --label "money.zerone.chain-id=zerone-1"
  --label "money.zerone.genesis-sha256=${GENESIS_SHA256}"
  --label "money.zerone.build-profile=${PROFILE}"
  --label "money.zerone.platform=${TARGET_PLATFORM}"
  --label "money.zerone.release-tag=${RELEASE_TAG}"
  --label "money.zerone.release-signer-fingerprint=${RELEASE_SIGNER_FINGERPRINT}"
  --tag "${IMAGE_REF}")
[ -z "${GO_IMAGE:-}" ] || docker_args+=(--build-arg "GO_IMAGE=${GO_IMAGE}")
[ -z "${RUNTIME_IMAGE:-}" ] || docker_args+=(--build-arg "RUNTIME_IMAGE=${RUNTIME_IMAGE}")
if [ "${PROFILE}" = "release" ]; then
  verify_clean_release_tree
fi
docker "${docker_args[@]}" "${CONTEXT}"

printf 'image=%s\nprofile=%s\ncommit=%s\ngenesis_sha256=%s\nplatform=%s\nrelease_tag=%s\nrelease_signer_fingerprint=%s\n' \
  "${IMAGE_REF}" "${PROFILE}" "${HEAD_COMMIT}" "${GENESIS_SHA256}" \
  "${TARGET_PLATFORM}" "${RELEASE_TAG}" "${RELEASE_SIGNER_FINGERPRINT}"
