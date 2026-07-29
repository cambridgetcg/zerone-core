#!/usr/bin/env bash
# Build the query-only gateway from a three-file, custody-free context.
set -euo pipefail
export LC_ALL=C
export GIT_NO_REPLACE_OBJECTS=1
umask 077

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
IMAGE_REF="${1:?usage: build-image.sh <local-image-reference>}"
PROFILE="${GATEWAY_BUILD_PROFILE:-release}"
TARGET_PLATFORM="linux/amd64"
CONTEXT=""

die() { printf 'query-gateway build: ERROR: %s\n' "$*" >&2; exit 1; }

cleanup() {
  [ -z "${CONTEXT}" ] || rm -rf "${CONTEXT}"
}
trap cleanup EXIT HUP INT TERM

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

verify_clean_release_tree() {
  [ "$(git -C "${ROOT}" rev-parse HEAD)" = "${SOURCE_COMMIT}" ] || \
    die "HEAD changed during the gateway release build"
  [ -z "$(git -C "${ROOT}" status --porcelain --untracked-files=all)" ] || \
    die "gateway release build requires a clean worktree"
  reject_git_history_overrides
}

materialize_git_file() {
  local relative="$1" destination="$2" listing mode
  listing=$(git -C "${ROOT}" ls-tree "${SOURCE_COMMIT}" -- "${relative}") || \
    die "could not inspect ${relative} at ${SOURCE_COMMIT}"
  [ -n "${listing}" ] || die "release commit omits ${relative}"
  [ "$(printf '%s\n' "${listing}" | wc -l | tr -d ' ')" = "1" ] || \
    die "release path resolves ambiguously: ${relative}"
  mode=$(printf '%s\n' "${listing}" | awk '{print $1}')
  case "${mode}" in 100644|100755) ;; *) die "release input is not a regular Git blob: ${relative}" ;; esac
  git -C "${ROOT}" show "${SOURCE_COMMIT}:${relative}" > "${destination}" || \
    die "could not materialize ${relative} from signed release"
}

verify_release_tag() {
  local tag_ref output count actual
  RELEASE_TAG="${ZERONE_RELEASE_TAG:-}"
  AUTHORIZED_FINGERPRINT=$(printf '%s' \
    "${ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT:-}" | tr '[:upper:]' '[:lower:]')
  [[ "${RELEASE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$ ]] || \
    die "release build requires a safe ZERONE_RELEASE_TAG"
  [[ "${AUTHORIZED_FINGERPRINT}" =~ ^([0-9a-f]{40}|[0-9a-f]{64})$ ]] || \
    die "release build requires an authorized OpenPGP signer fingerprint"
  tag_ref="refs/tags/${RELEASE_TAG}"
  [ "$(git -C "${ROOT}" cat-file -t "${tag_ref}" 2>/dev/null || true)" = tag ] || \
    die "release tag must be annotated"
  [ "$(git -C "${ROOT}" rev-list -n 1 "${tag_ref}")" = "${SOURCE_COMMIT}" ] || \
    die "release tag does not point to HEAD"
  output=$(git -C "${ROOT}" -c gpg.format=openpgp -c gpg.program=gpg \
    -c gpg.openpgp.program=gpg verify-tag --raw "${tag_ref}" 2>&1) || \
    die "release tag signature verification failed"
  count=$(printf '%s\n' "${output}" | \
    awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {n++} END {print n + 0}')
  [ "${count}" = 1 ] || die "release tag must produce exactly one VALIDSIG"
  actual=$(printf '%s\n' "${output}" | \
    awk '$1 == "[GNUPG:]" && $2 == "VALIDSIG" {print tolower($3); exit}')
  [ "${actual}" = "${AUTHORIZED_FINGERPRINT}" ] || \
    die "release tag was not signed by the authorized fingerprint"
}

case "${PROFILE}" in release|development) ;; *) die "GATEWAY_BUILD_PROFILE must be release or development" ;; esac
command -v docker >/dev/null 2>&1 || die "docker is required"
command -v git >/dev/null 2>&1 || die "git is required"
[[ "${IMAGE_REF}" =~ ^[A-Za-z0-9][A-Za-z0-9._/:@-]{0,255}$ ]] || \
  die "local image reference contains unsafe characters"
[[ "${NGINX_IMAGE:-}" =~ ^[a-z0-9]([a-z0-9._/-]*[a-z0-9])?@sha256:[0-9a-f]{64}$ ]] || \
  die "NGINX_IMAGE must be a lowercase immutable image digest reference"

SOURCE_COMMIT=$(git -C "${ROOT}" rev-parse HEAD)
[[ "${SOURCE_COMMIT}" =~ ^[0-9a-f]{40}$ ]] || die "HEAD must be a 40-hex commit"
if [ "${PROFILE}" = release ]; then
  verify_clean_release_tree
  verify_release_tag
else
  RELEASE_TAG=development
  AUTHORIZED_FINGERPRINT=none
fi

CONTEXT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-query-gateway-context.XXXXXX")
for name in Dockerfile entrypoint.sh default.conf.template; do
  relative="deploy/query-gateway/${name}"
  if [ "${PROFILE}" = release ]; then
    materialize_git_file "${relative}" "${CONTEXT}/${name}"
  else
    source_file="${ROOT}/${relative}"
    [ -f "${source_file}" ] && [ ! -L "${source_file}" ] || \
      die "development input must be a regular non-symlink file: ${relative}"
    cp "${source_file}" "${CONTEXT}/${name}"
  fi
done
chmod 0644 "${CONTEXT}/Dockerfile" "${CONTEXT}/default.conf.template"
chmod 0755 "${CONTEXT}/entrypoint.sh"
printf 'schema=zerone-query-gateway-context-v1\nprofile=%s\nsource_commit=%s\nrelease_tag=%s\nrelease_signer_fingerprint=%s\nplatform=%s\n' \
  "${PROFILE}" "${SOURCE_COMMIT}" "${RELEASE_TAG}" \
  "${AUTHORIZED_FINGERPRINT}" "${TARGET_PLATFORM}" \
  > "${CONTEXT}/.sanitized-query-gateway-context-v1"

[ "$(find "${CONTEXT}" -mindepth 1 -maxdepth 1 -type f | wc -l | tr -d ' ')" = 4 ] || \
  die "gateway context contains an unexpected file"
if find "${CONTEXT}" -type f \( -name '*key*' -o -name '*mnemonic*' -o -name '*.env' \) \
  -print -quit | grep -q .; then
  die "custody-shaped file entered the gateway context"
fi

[ "${PROFILE}" != release ] || verify_clean_release_tree
docker build --platform "${TARGET_PLATFORM}" \
  --build-arg "NGINX_IMAGE=${NGINX_IMAGE}" \
  --label "org.opencontainers.image.revision=${SOURCE_COMMIT}" \
  --label "org.opencontainers.image.version=${RELEASE_TAG}" \
  --label "money.zerone.component=query-gateway" \
  --label "money.zerone.release-signer-fingerprint=${AUTHORIZED_FINGERPRINT}" \
  --tag "${IMAGE_REF}" "${CONTEXT}"

printf 'image=%s\nprofile=%s\ncommit=%s\ntag=%s\nplatform=%s\n' \
  "${IMAGE_REF}" "${PROFILE}" "${SOURCE_COMMIT}" "${RELEASE_TAG}" "${TARGET_PLATFORM}"
