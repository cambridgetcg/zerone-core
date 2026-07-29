#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-query-gateway-build-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin"

cat > "${TMP}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
context="${@: -1}"
[ -f "${context}/Dockerfile" ]
[ -f "${context}/entrypoint.sh" ]
[ -f "${context}/default.conf.template" ]
[ -f "${context}/.sanitized-query-gateway-context-v1" ]
[ "$(find "${context}" -mindepth 1 -maxdepth 1 -type f | wc -l | tr -d ' ')" = 4 ]
grep -q '^profile=development$' "${context}/.sanitized-query-gateway-context-v1"
printf '%s\n' "$@" > "${FAKE_DOCKER_LOG}"
EOF
chmod +x "${TMP}/bin/docker"

DIGEST=$(printf 'a%.0s' {1..64})
PATH="${TMP}/bin:${PATH}" FAKE_DOCKER_LOG="${TMP}/docker.log" \
GATEWAY_BUILD_PROFILE=development \
NGINX_IMAGE="docker.io/library/nginx@sha256:${DIGEST}" \
  "${ROOT}/deploy/query-gateway/build-image.sh" zerone-query-gateway:test \
  > "${TMP}/build.log"
grep -q '^zerone-query-gateway:test$' <(awk '/^--tag$/ {getline; print}' "${TMP}/docker.log")
grep -q '^docker.io/library/nginx@sha256:' <(sed -n 's/^NGINX_IMAGE=//p' "${TMP}/docker.log")

if PATH="${TMP}/bin:${PATH}" GATEWAY_BUILD_PROFILE=development \
  NGINX_IMAGE=nginx:latest "${ROOT}/deploy/query-gateway/build-image.sh" bad:test \
  >/dev/null 2>&1; then
  printf 'query gateway build test: mutable base tag was accepted\n' >&2
  exit 1
fi

printf 'query gateway build tests: PASS\n'
