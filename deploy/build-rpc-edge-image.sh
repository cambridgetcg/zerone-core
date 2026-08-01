#!/usr/bin/env bash
# Build the stateless RPC edge from two immutable Git blobs on local Docker.
set -euo pipefail
export LC_ALL=C
export GIT_NO_REPLACE_OBJECTS=1
umask 077

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
IMAGE_REF="${1:?usage: build-rpc-edge-image.sh <local-image-reference>}"
TARGET_PLATFORM="linux/amd64"
CONTEXT=""

die() { printf 'rpc-edge build: ERROR: %s\n' "$*" >&2; exit 1; }

cleanup() {
  [ -z "${CONTEXT}" ] || rm -rf -- "${CONTEXT}"
}
trap cleanup EXIT

reject_git_history_overrides() {
  local graft_path
  [ -z "$(git -C "${ROOT}" for-each-ref --format='%(refname)' refs/replace/)" ] || \
    die "replacement Git refs are forbidden"
  graft_path=$(git -C "${ROOT}" rev-parse --path-format=absolute \
    --git-path info/grafts) || die "could not resolve the Git graft path"
  if [ -e "${graft_path}" ] || [ -L "${graft_path}" ]; then
    die "legacy Git grafts are forbidden"
  fi
}

verify_source_identity() {
  local expected_script_blob actual_script_blob
  [ "$(git -C "${ROOT}" rev-parse HEAD)" = "${SOURCE_COMMIT}" ] || \
    die "RPC_EDGE_SOURCE_COMMIT must equal the checked-out HEAD"
  expected_script_blob=$(git -C "${ROOT}" rev-parse \
    "${SOURCE_COMMIT}:deploy/build-rpc-edge-image.sh") || \
    die "source commit omits the build wrapper"
  actual_script_blob=$(git -C "${ROOT}" hash-object \
    "${ROOT}/deploy/build-rpc-edge-image.sh") || \
    die "could not hash the executing build wrapper"
  [ "${actual_script_blob}" = "${expected_script_blob}" ] || \
    die "executing build wrapper differs from the exact source commit"
  reject_git_history_overrides
}

materialize_git_blob() {
  local relative="$1" destination="$2" listing mode object_type
  listing=$(git -C "${ROOT}" ls-tree "${SOURCE_COMMIT}" -- "${relative}") || \
    die "could not inspect ${relative} at ${SOURCE_COMMIT}"
  [ -n "${listing}" ] || die "source commit omits ${relative}"
  [ "$(printf '%s\n' "${listing}" | wc -l | tr -d ' ')" = "1" ] || \
    die "source path resolves ambiguously: ${relative}"
  mode=$(printf '%s\n' "${listing}" | awk '{print $1}')
  object_type=$(printf '%s\n' "${listing}" | awk '{print $2}')
  case "${mode}:${object_type}" in
    100644:blob|100755:blob) ;;
    *) die "source input is not a regular Git blob: ${relative}" ;;
  esac
  mkdir -p "$(dirname "${destination}")"
  git -C "${ROOT}" show "${SOURCE_COMMIT}:${relative}" > "${destination}" || \
    die "could not materialize ${relative}"
  chmod 0644 "${destination}"
}

command -v docker >/dev/null 2>&1 || die "docker is required"
command -v git >/dev/null 2>&1 || die "git is required"
[[ "${IMAGE_REF}" =~ ^[A-Za-z0-9][A-Za-z0-9._/:@-]{0,255}$ ]] || \
  die "local image reference contains unsafe characters"

SOURCE_COMMIT="${RPC_EDGE_SOURCE_COMMIT:-}"
VERSION="${RPC_EDGE_VERSION:-}"
RUNTIME_IMAGE="${RUNTIME_IMAGE:-}"
[[ "${SOURCE_COMMIT}" =~ ^[0-9a-f]{40}$ ]] || \
  die "RPC_EDGE_SOURCE_COMMIT must be exactly 40 lowercase hex characters"
[[ "${VERSION}" =~ ^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] || \
  die "RPC_EDGE_VERSION must be a semantic version"
[[ "${RUNTIME_IMAGE}" =~ ^[a-z0-9][a-z0-9._/:+-]*@sha256:[0-9a-f]{64}$ ]] || \
  die "RUNTIME_IMAGE must be a lowercase digest-pinned image reference"
[[ "${RUNTIME_IMAGE}" != *".."* && "${RUNTIME_IMAGE}" != *"//"* ]] || \
  die "RUNTIME_IMAGE contains an ambiguous path"
git -C "${ROOT}" cat-file -e "${SOURCE_COMMIT}^{commit}" 2>/dev/null || \
  die "RPC_EDGE_SOURCE_COMMIT is not a local commit"
verify_source_identity

SOURCE_DATE_EPOCH=$(git -C "${ROOT}" show -s --format=%ct "${SOURCE_COMMIT}")
[[ "${SOURCE_DATE_EPOCH}" =~ ^[1-9][0-9]*$ ]] || \
  die "source commit timestamp must be a positive Unix epoch"

for forbidden_environment in DOCKER_HOST DOCKER_CONTEXT BUILDX_BUILDER; do
  [ -z "${!forbidden_environment:-}" ] || \
    die "${forbidden_environment} is forbidden; select a reviewed local Docker context"
done
DOCKER_CONTEXT_NAME=$(docker context show) || die "could not resolve Docker context"
[[ "${DOCKER_CONTEXT_NAME}" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || \
  die "Docker context name is unsafe"
DOCKER_ENDPOINT=$(docker context inspect --format \
  '{{ (index .Endpoints "docker").Host }}' "${DOCKER_CONTEXT_NAME}") || \
  die "could not inspect Docker context"
[[ "${DOCKER_ENDPOINT}" == unix:///* ]] || \
  die "Docker context must use a local Unix socket, got ${DOCKER_ENDPOINT}"
BUILDER_INFO=$(docker --context "${DOCKER_CONTEXT_NAME}" \
  buildx --builder "${DOCKER_CONTEXT_NAME}" inspect) || \
  die "could not inspect the selected context's builder"
BUILDER_DRIVER=$(printf '%s\n' "${BUILDER_INFO}" | \
  awk '/^Driver:[[:space:]]*/ { sub(/^Driver:[[:space:]]*/, ""); print }')
[ "${BUILDER_DRIVER}" = "docker" ] || \
  die "selected context's builder must use the docker driver"

CONTEXT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-rpc-edge-context.XXXXXX")
materialize_git_blob deploy/Dockerfile.rpc-edge \
  "${CONTEXT}/deploy/Dockerfile.rpc-edge"
materialize_git_blob deploy/public-edge-nginx.conf \
  "${CONTEXT}/deploy/public-edge-nginx.conf"

expected_files=$'./deploy/Dockerfile.rpc-edge\n./deploy/public-edge-nginx.conf'
actual_files=$(cd "${CONTEXT}" && find . -type f -print | LC_ALL=C sort)
[ "${actual_files}" = "${expected_files}" ] || \
  die "temporary Docker context escaped its two-file allowlist"
if find "${CONTEXT}" -mindepth 1 ! -type d ! -type f -print -quit | grep -q .; then
  die "temporary Docker context contains a non-regular object"
fi

verify_source_identity
docker --context "${DOCKER_CONTEXT_NAME}" build \
  --builder "${DOCKER_CONTEXT_NAME}" \
  --pull \
  --no-cache \
  --load \
  --platform "${TARGET_PLATFORM}" \
  --build-arg "RUNTIME_IMAGE=${RUNTIME_IMAGE}" \
  --build-arg "VERSION=${VERSION}" \
  --build-arg "COMMIT=${SOURCE_COMMIT}" \
  --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
  --label "org.opencontainers.image.base.name=${RUNTIME_IMAGE}" \
  --label "org.opencontainers.image.revision=${SOURCE_COMMIT}" \
  --label "org.opencontainers.image.version=${VERSION}" \
  --label "io.zerone.source-date-epoch=${SOURCE_DATE_EPOCH}" \
  --label "io.zerone.build-context=tracked-rpc-edge-v1" \
  --file "${CONTEXT}/deploy/Dockerfile.rpc-edge" \
  --tag "${IMAGE_REF}" \
  "${CONTEXT}"

printf 'local_image=%s\ncommit=%s\nversion=%s\nsource_date_epoch=%s\nplatform=%s\nbase=%s\ndocker_context=%s\ndocker_endpoint=%s\nbuilder=%s\nbuilder_driver=%s\npush_or_deploy=not-performed\n' \
  "${IMAGE_REF}" "${SOURCE_COMMIT}" "${VERSION}" "${SOURCE_DATE_EPOCH}" \
  "${TARGET_PLATFORM}" "${RUNTIME_IMAGE}" "${DOCKER_CONTEXT_NAME}" \
  "${DOCKER_ENDPOINT}" "${DOCKER_CONTEXT_NAME}" "${BUILDER_DRIVER}"
