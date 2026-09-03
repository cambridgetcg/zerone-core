#!/usr/bin/env bash
# Build the real pinned gateway image and make its nginx parse the rendered
# production template. This complements, but does not replace, the fast fakes.
set -euo pipefail
export LC_ALL=C
umask 077

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
IMAGE_REF="zerone-query-gateway-config-test:${GITHUB_RUN_ID:-local}-$$"
CONTAINER_NAME="zerone-query-gateway-lifecycle-test-${GITHUB_RUN_ID:-local}-$$"
BUILT=false
CONTAINER_STARTED=false

cleanup() {
  if [ "${CONTAINER_STARTED}" = true ]; then
    docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi
  if [ "${BUILT}" = true ]; then
    docker image rm "${IMAGE_REF}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT HUP INT TERM

command -v docker >/dev/null 2>&1 || {
  printf 'query gateway nginx config test: docker is required\n' >&2
  exit 1
}

GATEWAY_BUILD_PROFILE=development \
  "${ROOT}/deploy/query-gateway/build-image.sh" "${IMAGE_REF}"
BUILT=true

docker run --rm --platform linux/amd64 \
  --entrypoint /usr/local/bin/zerone-query-readiness \
  -e GATEWAY_ROLE=zerone-2-query \
  -e EXPECTED_CHAIN_ID=zerone-2 \
  -e UPSTREAM_HOST=zerone-2-edge.internal \
  "${IMAGE_REF}" -check-config

docker run --rm --platform linux/amd64 \
  --entrypoint /usr/local/bin/zerone-query-readiness \
  -e GATEWAY_ROLE=zerone-1-archive-query \
  -e EXPECTED_CHAIN_ID=zerone-1 \
  -e UPSTREAM_HOST=zerone-1-archive.internal \
  -e EXPECTED_ARCHIVE_HEIGHT=42 \
  -e EXPECTED_ARCHIVE_APP_HASH=abababababababababababababababababababababababababababababababab \
  -e EXPECTED_ARCHIVE_BLOCK_HASH=cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd \
  "${IMAGE_REF}" -check-config

docker run --rm --platform linux/amd64 --entrypoint sh \
  -e UPSTREAM_HOST=zerone-2-edge.internal \
  -e EXPECTED_CHAIN_ID=zerone-2 \
  "${IMAGE_REF}" -ec '
    envsubst '\''${EXPECTED_CHAIN_ID}'\'' \
      < /etc/zerone-query-gateway/default.conf.template \
      > /etc/nginx/conf.d/default.conf
    nginx -t -q -c /etc/nginx/nginx.conf
    nginx
    trap "nginx -s quit >/dev/null 2>&1 || true" EXIT HUP INT TERM
    sleep 1
    account=zrn1
    account_character=0
    while [ "${account_character}" -lt 38 ]; do
      account="${account}0"
      account_character=$((account_character + 1))
    done
    for path in \
      "/cosmos/base/tendermint/v1beta1/syncing?unexpected=true" \
      "/cosmos/base/tendermint/v1beta1/node_info?unexpected=true" \
      "/cosmos/bank/v1beta1/denoms_metadata/uzrn?unexpected=true" \
      "/zerone/liquiditypool/v1/params?unexpected=true" \
      "/zerone/liquiditypool/v1/pools?pagination.count_total=true" \
      "/cosmos/auth/v1beta1/accounts/${account}?unexpected=true" \
      "/zerone/auth/v1/account_identifier/${account}?unexpected=true"; do
      probe_output=$(wget -S -O /dev/null -T 2 -t 1 \
        "http://127.0.0.1:8080${path}" 2>&1 || true)
      printf "%s\n" "${probe_output}" | grep -Eq "HTTP/1\\.[01] 404" || {
        printf "query gateway real nginx config test: query variant was not rejected: %s\n" \
          "${path}" >&2
        exit 1
      }
    done
    for path in \
      "/cosmos/auth/v1beta1/accounts/${account}" \
      "/zerone/auth/v1/account_identifier/${account}"; do
      probe_output=$(wget -S -O /dev/null -T 2 -t 1 \
        "http://127.0.0.1:8080${path}" 2>&1 || true)
      printf "%s\n" "${probe_output}" | grep -Eq "HTTP/1\\.[01] 500" || {
        printf "query gateway real nginx config test: account route did not reach readiness gate: %s\n" \
          "${path}" >&2
        exit 1
      }
    done
    nginx -s quit
    trap - EXIT HUP INT TERM
  '

# Route one body-bearing GET through real nginx to a one-shot local origin.
# Only the test target/readiness response are replaced; the inherited
# production proxy body directives remain unchanged.
docker run --rm --platform linux/amd64 --entrypoint sh \
  -e UPSTREAM_HOST=zerone-2-edge.internal \
  -e EXPECTED_CHAIN_ID=zerone-2 \
  "${IMAGE_REF}" -ec '
    envsubst '\''${EXPECTED_CHAIN_ID}'\'' \
      < /etc/zerone-query-gateway/default.conf.template \
      > /etc/nginx/conf.d/default.conf
    printf "%s\n" \
      "server { listen 127.0.0.1:18081; location = /ready { add_header X-Zerone-Validated-6PN \"::1\" always; return 204; } }" \
      "log_format body_probe \"\$request_method|\$content_length|\$http_transfer_encoding|\$request_body\";" \
      "server { listen [::1]:1317; access_log /tmp/origin-request body_probe; location / { return 200 \"rest-ok\"; } }" \
      "server { listen [::1]:26657; location / { return 200 \"rpc-ok\"; } }" \
      >> /etc/nginx/conf.d/default.conf
    nginx -t -q -c /etc/nginx/nginx.conf
    nginx >/tmp/nginx.log 2>&1
    trap "nginx -s quit >/dev/null 2>&1 || true" EXIT HUP INT TERM
    sleep 1
    { printf "GET /cosmos/base/tendermint/v1beta1/syncing HTTP/1.1\r\nHost: gateway\r\nX-Zerone-Validated-6PN: fdaa::dead\r\nConnection: close\r\nContent-Length: 7\r\n\r\nPAYLOAD"; sleep 2; } | \
      nc -w 5 127.0.0.1 8080 >/tmp/client-response || true
    body_probe_attempt=0
    while [ "${body_probe_attempt}" -lt 10 ] && [ ! -s /tmp/origin-request ]; do
      body_probe_attempt=$((body_probe_attempt + 1))
      sleep 1
    done
    grep -Fq "GET|-|-|-" /tmp/origin-request || {
      cat /tmp/nginx.log >&2
      cat /tmp/origin-request >&2 || true
      printf "query gateway real nginx config test: proxied GET was not bodyless\n" >&2
      exit 1
    }
    grep -Fq "rest-ok" /tmp/client-response || {
      printf "query gateway real nginx config test: REST request did not use the helper-validated IPv6 origin\n" >&2
      exit 1
    }
    rpc_response=$(wget -qO- -T 2 -t 1 http://127.0.0.1:8080/status || true)
    [ "${rpc_response}" = rpc-ok ] || {
      printf "query gateway real nginx config test: RPC request did not use the helper-validated IPv6 origin\n" >&2
      exit 1
    }
    nginx -s quit >/dev/null 2>&1
    trap - EXIT HUP INT TERM
  '

# Exercise the production POSIX entrypoint, helper, and nginx supervisor in the
# pinned image. The semantic probe may remain unready without Fly DNS, but both
# long-running children must stay alive and then stop together on SIGTERM.
docker run --rm -d --platform linux/amd64 --name "${CONTAINER_NAME}" \
  -e GATEWAY_ROLE=zerone-2-query \
  -e EXPECTED_CHAIN_ID=zerone-2 \
  -e UPSTREAM_HOST=zerone-2-edge.internal \
  "${IMAGE_REF}" >/dev/null
CONTAINER_STARTED=true
sleep 2
if [ "$(docker inspect --format '{{.State.Running}}' "${CONTAINER_NAME}" 2>/dev/null || true)" != true ]; then
  docker logs "${CONTAINER_NAME}" >&2 || true
  printf 'query gateway real nginx config test: supervised container exited at startup\n' >&2
  exit 1
fi
docker stop --time 15 "${CONTAINER_NAME}" >/dev/null
CONTAINER_STARTED=false

printf 'query gateway real nginx config test: PASS\n'
