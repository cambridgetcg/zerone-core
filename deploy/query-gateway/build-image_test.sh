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
[ -f "${context}/nginx-image.txt" ]
[ -f "${context}/readiness.go" ]
[ -f "${context}/.sanitized-query-gateway-context-v1" ]
[ "$(find "${context}" -mindepth 1 -maxdepth 1 -type f | wc -l | tr -d ' ')" = 6 ]
grep -q '^profile=development$' "${context}/.sanitized-query-gateway-context-v1"
grep -Fqx 'docker.io/library/nginx@sha256:0d3b80406a13a767339fbe2f41406d6c7da727ab89cf8fae399e81f780f814d1' \
  "${context}/nginx-image.txt"
printf '%s\n' "$@" > "${FAKE_DOCKER_LOG}"
EOF
chmod +x "${TMP}/bin/docker"

PATH="${TMP}/bin:${PATH}" FAKE_DOCKER_LOG="${TMP}/docker.log" \
GATEWAY_BUILD_PROFILE=development \
  "${ROOT}/deploy/query-gateway/build-image.sh" zerone-query-gateway:test \
  > "${TMP}/build.log"
grep -q '^zerone-query-gateway:test$' <(awk '/^--tag$/ {getline; print}' "${TMP}/docker.log")
grep -Fqx 'NGINX_IMAGE=docker.io/library/nginx@sha256:0d3b80406a13a767339fbe2f41406d6c7da727ab89cf8fae399e81f780f814d1' \
  "${TMP}/docker.log"
grep -Fqx 'GO_IMAGE=golang:1.25.14-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437' \
  "${TMP}/docker.log"

if PATH="${TMP}/bin:${PATH}" GATEWAY_BUILD_PROFILE=development \
  NGINX_IMAGE='docker.io/library/nginx@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' \
  "${ROOT}/deploy/query-gateway/build-image.sh" bad:test \
  >/dev/null 2>&1; then
  printf 'query gateway build test: base digest differing from signed pin was accepted\n' >&2
  exit 1
fi

printf 'query gateway build tests: PASS\n'
