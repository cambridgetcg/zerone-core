#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-rpc-edge-build-test.XXXXXX")
trap 'rm -rf -- "${TEST_ROOT}"' EXIT
REPOSITORY="${TEST_ROOT}/repository"
mkdir -p "${REPOSITORY}/deploy" "${TEST_ROOT}/bin" "${TEST_ROOT}/contexts"

fail() { printf 'rpc-edge build test: FAIL: %s\n' "$*" >&2; exit 1; }

cp "${ROOT}/deploy/build-rpc-edge-image.sh" "${REPOSITORY}/deploy/"
cp "${ROOT}/deploy/Dockerfile.rpc-edge" "${REPOSITORY}/deploy/"
cp "${ROOT}/deploy/public-edge-nginx.conf" "${REPOSITORY}/deploy/"
chmod 0755 "${REPOSITORY}/deploy/build-rpc-edge-image.sh"
git -C "${REPOSITORY}" init -q
git -C "${REPOSITORY}" config user.name "RPC Edge Test"
git -C "${REPOSITORY}" config user.email "rpc-edge-test@example.invalid"
git -C "${REPOSITORY}" add deploy
GIT_AUTHOR_DATE='2026-08-01T00:00:00Z' \
GIT_COMMITTER_DATE='2026-08-01T00:00:00Z' \
  git -C "${REPOSITORY}" commit -q -m fixture
SOURCE_COMMIT=$(git -C "${REPOSITORY}" rev-parse HEAD)
SOURCE_DATE_EPOCH=$(git -C "${REPOSITORY}" show -s --format=%ct "${SOURCE_COMMIT}")
git -C "${REPOSITORY}" show \
  "${SOURCE_COMMIT}:deploy/Dockerfile.rpc-edge" > "${TEST_ROOT}/expected-Dockerfile"
git -C "${REPOSITORY}" show \
  "${SOURCE_COMMIT}:deploy/public-edge-nginx.conf" > "${TEST_ROOT}/expected-nginx.conf"

# Dirty and custody-shaped worktree bytes must never reach the Git-object context.
printf '\nSECRET_SENTINEL_FROM_DIRTY_WORKTREE\n' >> \
  "${REPOSITORY}/deploy/Dockerfile.rpc-edge"
printf '\nSECRET_SENTINEL_FROM_DIRTY_WORKTREE\n' >> \
  "${REPOSITORY}/deploy/public-edge-nginx.conf"
printf '%s\n' 'not-a-real-key' > "${REPOSITORY}/priv_validator_key.json"

cat > "${TEST_ROOT}/bin/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-} ${2:-}" in
  "context show")
    printf '%s\n' 'rpc-edge-isolated-test'
    ;;
  "context inspect")
    [ "${3:-}" = '--format' ]
    [ "${4:-}" = '{{ (index .Endpoints "docker").Host }}' ]
    [ "${5:-}" = 'rpc-edge-isolated-test' ]
    printf '%s\n' "${FAKE_DOCKER_ENDPOINT:-unix:///private/tmp/rpc-edge-test.sock}"
    ;;
  "--context rpc-edge-isolated-test")
    [ "${3:-}" = build ]
    context="${*: -1}"
    [ -d "${context}" ]
    (cd "${context}" && find . -type f -print | LC_ALL=C sort) > \
      "${FAKE_DOCKER_CONTEXT_FILES:?}"
    cmp "${context}/deploy/Dockerfile.rpc-edge" "${EXPECTED_DOCKERFILE:?}"
    cmp "${context}/deploy/public-edge-nginx.conf" "${EXPECTED_NGINX:?}"
    ! grep -R -F 'SECRET_SENTINEL_FROM_DIRTY_WORKTREE' "${context}"
    printf '%s\n' "$@" > "${FAKE_DOCKER_ARGS:?}"
    printf '%s\n' "${context}" > "${FAKE_DOCKER_CONTEXT_PATH:?}"
    ;;
  *)
    printf 'unexpected fake Docker invocation: %s\n' "$*" >&2
    exit 1
    ;;
esac
FAKE_DOCKER
chmod 0755 "${TEST_ROOT}/bin/docker"

DIGEST=$(printf 'a%.0s' {1..64})
RUNTIME_IMAGE="docker.io/library/debian:bookworm-slim@sha256:${DIGEST}"
FAKE_DOCKER_ARGS="${TEST_ROOT}/docker-args" \
FAKE_DOCKER_CONTEXT_FILES="${TEST_ROOT}/context-files" \
FAKE_DOCKER_CONTEXT_PATH="${TEST_ROOT}/context-path" \
EXPECTED_DOCKERFILE="${TEST_ROOT}/expected-Dockerfile" \
EXPECTED_NGINX="${TEST_ROOT}/expected-nginx.conf" \
PATH="${TEST_ROOT}/bin:${PATH}" \
TMPDIR="${TEST_ROOT}/contexts" \
RPC_EDGE_SOURCE_COMMIT="${SOURCE_COMMIT}" \
RPC_EDGE_VERSION='v0.1.0' \
RUNTIME_IMAGE="${RUNTIME_IMAGE}" \
  "${REPOSITORY}/deploy/build-rpc-edge-image.sh" zerone-rpc-edge:test \
  > "${TEST_ROOT}/build-output"

printf '%s\n' \
  './deploy/Dockerfile.rpc-edge' \
  './deploy/public-edge-nginx.conf' > "${TEST_ROOT}/expected-context-files"
cmp "${TEST_ROOT}/context-files" "${TEST_ROOT}/expected-context-files"
CONTEXT_PATH=$(sed -n '1p' "${TEST_ROOT}/context-path")
[ ! -e "${CONTEXT_PATH}" ] || fail "temporary context survived the build"
for exact_argument in \
  '--pull' \
  '--no-cache' \
  'linux/amd64' \
  "RUNTIME_IMAGE=${RUNTIME_IMAGE}" \
  'VERSION=v0.1.0' \
  "COMMIT=${SOURCE_COMMIT}" \
  "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
  "org.opencontainers.image.base.name=${RUNTIME_IMAGE}" \
  "org.opencontainers.image.revision=${SOURCE_COMMIT}" \
  'io.zerone.build-context=tracked-rpc-edge-v1' \
  'zerone-rpc-edge:test'; do
  grep -Fqx -- "${exact_argument}" "${TEST_ROOT}/docker-args" || \
    fail "Docker arguments omitted ${exact_argument}"
done
if grep -Eiq '(^|[-])(push|deploy|buildx)($|[-])|fly|depot' \
    "${TEST_ROOT}/docker-args"; then
  fail "build wrapper crossed into push, deploy, Fly, Depot, or buildx"
fi
grep -q '^push_or_deploy=not-performed$' "${TEST_ROOT}/build-output" || \
  fail "build output did not preserve the publication boundary"

if PATH="${TEST_ROOT}/bin:${PATH}" \
    RPC_EDGE_SOURCE_COMMIT="${SOURCE_COMMIT}" \
    RPC_EDGE_VERSION='v0.1.0' \
    RUNTIME_IMAGE='debian:bookworm-slim' \
    "${REPOSITORY}/deploy/build-rpc-edge-image.sh" bad:test \
    >/dev/null 2>&1; then
  fail "mutable base image was accepted"
fi

: > "${TEST_ROOT}/docker-args"
if FAKE_DOCKER_ENDPOINT='tcp://remote-builder.invalid:2376' \
    FAKE_DOCKER_ARGS="${TEST_ROOT}/docker-args" \
    FAKE_DOCKER_CONTEXT_FILES="${TEST_ROOT}/context-files" \
    FAKE_DOCKER_CONTEXT_PATH="${TEST_ROOT}/context-path" \
    EXPECTED_DOCKERFILE="${TEST_ROOT}/expected-Dockerfile" \
    EXPECTED_NGINX="${TEST_ROOT}/expected-nginx.conf" \
    PATH="${TEST_ROOT}/bin:${PATH}" \
    RPC_EDGE_SOURCE_COMMIT="${SOURCE_COMMIT}" \
    RPC_EDGE_VERSION='v0.1.0' \
    RUNTIME_IMAGE="${RUNTIME_IMAGE}" \
    "${REPOSITORY}/deploy/build-rpc-edge-image.sh" bad:test \
    >/dev/null 2>&1; then
  fail "remote Docker endpoint was accepted"
fi
[ ! -s "${TEST_ROOT}/docker-args" ] || \
  fail "remote Docker endpoint reached the build command"

printf 'rpc-edge build tests: PASS (two committed blobs; fake local Docker)\n'
