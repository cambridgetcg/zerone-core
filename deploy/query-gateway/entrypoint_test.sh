#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
SOURCE="${ROOT}/deploy/query-gateway/entrypoint.sh"
TEMPLATE_SOURCE="${ROOT}/deploy/query-gateway/default.conf.template"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-query-gateway-runtime-test.XXXXXX")
GATEWAY_PID=""
# shellcheck disable=SC2329 # Invoked by trap.
cleanup() {
  if [ -n "${GATEWAY_PID}" ] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill -TERM "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi
  rm -rf "${TMP}"
}
trap cleanup EXIT HUP INT TERM
mkdir -p "${TMP}/bin" "${TMP}/etc/nginx/conf.d" "${TMP}/etc/gateway"
cp "${TEMPLATE_SOURCE}" "${TMP}/etc/gateway/default.conf.template"

wait_for_file() {
  local file="$1"
  for _ in {1..1000}; do
    [ ! -f "${file}" ] || return 0
    sleep 0.01
  done
  printf 'query gateway runtime test: timed out waiting for %s\n' "${file}" >&2
  return 1
}

sed \
  -e "s|readonly TEMPLATE=/etc/zerone-query-gateway/default.conf.template|readonly TEMPLATE=${TMP}/etc/gateway/default.conf.template|" \
  -e "s|readonly CONFIG=/etc/nginx/conf.d/default.conf|readonly CONFIG=${TMP}/etc/nginx/conf.d/default.conf|" \
  -e "s|readonly READINESS_BIN=/usr/local/bin/zerone-query-readiness|readonly READINESS_BIN=${TMP}/bin/zerone-query-readiness|" \
  -e "s|mktemp /etc/nginx/conf.d/|mktemp ${TMP}/etc/nginx/conf.d/|" \
  "${SOURCE}" > "${TMP}/entrypoint.sh"
chmod +x "${TMP}/entrypoint.sh"

cat > "${TMP}/bin/envsubst" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
sed -e "s|\${UPSTREAM_HOST}|${UPSTREAM_HOST}|g" \
    -e "s|\${EXPECTED_CHAIN_ID}|${EXPECTED_CHAIN_ID}|g"
EOF
cat > "${TMP}/bin/nginx" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "$1" = "-t" ]; then
  grep -Fq 'auth_request_set $zerone_validated_6pn $upstream_http_x_zerone_validated_6pn;' "${EXPECTED_CONFIG}"
  grep -Fq 'proxy_pass http://[$zerone_validated_6pn]:26657;' "${EXPECTED_CONFIG}"
  grep -Fq 'proxy_pass http://[$zerone_validated_6pn]:1317;' "${EXPECTED_CONFIG}"
  grep -q 'proxy_pass http://127.0.0.1:18081/health;' "${EXPECTED_CONFIG}"
  grep -q "X-Zerone-Expected-Chain \"${EXPECTED_CHAIN}\"" "${EXPECTED_CONFIG}"
  ! grep -q '\${EXPECTED_CHAIN_ID}' "${EXPECTED_CONFIG}"
  ! grep -q 'zerone_rpc_origin\|zerone_rest_origin\|resolver \[fdaa::3\]' "${EXPECTED_CONFIG}"
  exit 0
fi
if [ "$1" = "-s" ] && [ "${2:-}" = quit ]; then
  printf 'QUIT\n' > "${NGINX_STOP_LOG}"
  if [ -f "${NGINX_START_LOG}.pid" ]; then
    kill -TERM "$(cat "${NGINX_START_LOG}.pid")" 2>/dev/null || true
  fi
  exit 0
fi
[ -z "${UPSTREAM_HOST+x}" ] || exit 90
trap 'exit 0' TERM INT HUP QUIT
printf '%s\n' "$$" > "${NGINX_START_LOG}.pid"
printf '%s\n' "$@" > "${NGINX_START_LOG}"
if [ "${NGINX_EXIT_IMMEDIATELY:-}" = 1 ]; then
  exit 23
fi
while :; do sleep 1; done
EOF
cat > "${TMP}/bin/zerone-query-readiness" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = -check-config ]; then
  printf 'checked\n' > "${READINESS_CHECK_LOG}"
  exit 0
fi
[ "$#" -eq 0 ]
on_term() { printf 'TERM\n' > "${READINESS_STOP_LOG}"; exit 0; }
trap on_term TERM INT HUP
printf 'role=%s\nchain=%s\nupstream=%s\narchive_height=%s\narchive_app_hash=%s\narchive_block_hash=%s\n' \
  "${GATEWAY_ROLE}" "${EXPECTED_CHAIN_ID}" "${UPSTREAM_HOST}" \
  "${EXPECTED_ARCHIVE_HEIGHT:-}" "${EXPECTED_ARCHIVE_APP_HASH:-}" \
  "${EXPECTED_ARCHIVE_BLOCK_HASH:-}" \
  > "${READINESS_START_LOG}"
if [ "${READINESS_EXIT_IMMEDIATELY:-}" = 1 ]; then
  exit 24
fi
while :; do sleep 1; done
EOF
chmod +x "${TMP}/bin/envsubst" "${TMP}/bin/nginx" \
  "${TMP}/bin/zerone-query-readiness"

PATH="${TMP}/bin:${PATH}" \
GATEWAY_ROLE=zerone-2-query EXPECTED_CHAIN_ID=zerone-2 \
UPSTREAM_HOST=zerone-2-edge.internal \
EXPECTED_ARCHIVE_HEIGHT='' EXPECTED_ARCHIVE_APP_HASH='' EXPECTED_ARCHIVE_BLOCK_HASH='' \
EXPECTED_UPSTREAM=zerone-2-edge.internal EXPECTED_CHAIN=zerone-2 \
EXPECTED_CONFIG="${TMP}/etc/nginx/conf.d/default.conf" \
NGINX_START_LOG="${TMP}/nginx-start.log" \
NGINX_STOP_LOG="${TMP}/nginx-stop.log" \
READINESS_CHECK_LOG="${TMP}/readiness-check.log" \
READINESS_START_LOG="${TMP}/readiness-start.log" \
READINESS_STOP_LOG="${TMP}/readiness-stop.log" \
  "${TMP}/entrypoint.sh" &
GATEWAY_PID=$!
wait_for_file "${TMP}/nginx-start.log"
wait_for_file "${TMP}/readiness-start.log"
grep -qx -- '-g' "${TMP}/nginx-start.log"
grep -qx -- 'daemon off;' "${TMP}/nginx-start.log"
grep -qx 'role=zerone-2-query' "${TMP}/readiness-start.log"
grep -qx 'chain=zerone-2' "${TMP}/readiness-start.log"
grep -qx 'upstream=zerone-2-edge.internal' "${TMP}/readiness-start.log"
grep -qx 'archive_height=' "${TMP}/readiness-start.log"
grep -qx 'archive_app_hash=' "${TMP}/readiness-start.log"
grep -qx 'archive_block_hash=' "${TMP}/readiness-start.log"
grep -qx 'checked' "${TMP}/readiness-check.log"
kill -TERM "${GATEWAY_PID}"
wait "${GATEWAY_PID}"
GATEWAY_PID=""
grep -qx 'QUIT' "${TMP}/nginx-stop.log"
grep -qx 'TERM' "${TMP}/readiness-stop.log"

ARCHIVE_HASH=$(printf 'ab%.0s' {1..32})
ARCHIVE_BLOCK_HASH=$(printf 'cd%.0s' {1..32})
rm -f "${TMP}/nginx-start.log" "${TMP}/nginx-stop.log" \
  "${TMP}/readiness-start.log" "${TMP}/readiness-stop.log" \
  "${TMP}/readiness-check.log"
PATH="${TMP}/bin:${PATH}" \
GATEWAY_ROLE=zerone-1-archive-query EXPECTED_CHAIN_ID=zerone-1 \
UPSTREAM_HOST=zerone-1-archive.internal \
EXPECTED_ARCHIVE_HEIGHT=42 EXPECTED_ARCHIVE_APP_HASH="${ARCHIVE_HASH}" \
EXPECTED_ARCHIVE_BLOCK_HASH="${ARCHIVE_BLOCK_HASH}" \
EXPECTED_UPSTREAM=zerone-1-archive.internal EXPECTED_CHAIN=zerone-1 \
EXPECTED_CONFIG="${TMP}/etc/nginx/conf.d/default.conf" \
NGINX_START_LOG="${TMP}/nginx-start.log" \
NGINX_STOP_LOG="${TMP}/nginx-stop.log" \
READINESS_CHECK_LOG="${TMP}/readiness-check.log" \
READINESS_START_LOG="${TMP}/readiness-start.log" \
READINESS_STOP_LOG="${TMP}/readiness-stop.log" \
  "${TMP}/entrypoint.sh" &
GATEWAY_PID=$!
wait_for_file "${TMP}/nginx-start.log"
wait_for_file "${TMP}/readiness-start.log"
grep -qx 'role=zerone-1-archive-query' "${TMP}/readiness-start.log"
grep -qx 'chain=zerone-1' "${TMP}/readiness-start.log"
grep -qx 'archive_height=42' "${TMP}/readiness-start.log"
grep -qx "archive_app_hash=${ARCHIVE_HASH}" "${TMP}/readiness-start.log"
grep -qx "archive_block_hash=${ARCHIVE_BLOCK_HASH}" "${TMP}/readiness-start.log"
kill -TERM "${GATEWAY_PID}"
wait "${GATEWAY_PID}"
GATEWAY_PID=""
grep -qx 'QUIT' "${TMP}/nginx-stop.log"
grep -qx 'TERM' "${TMP}/readiness-stop.log"

rm -f "${TMP}/nginx-start.log" "${TMP}/nginx-stop.log" \
  "${TMP}/readiness-start.log" "${TMP}/readiness-stop.log" \
  "${TMP}/readiness-check.log"
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-2-query \
  EXPECTED_CHAIN_ID=zerone-2 UPSTREAM_HOST=zerone-2-edge.internal \
  EXPECTED_ARCHIVE_HEIGHT='' EXPECTED_ARCHIVE_APP_HASH='' EXPECTED_ARCHIVE_BLOCK_HASH='' \
  EXPECTED_UPSTREAM=zerone-2-edge.internal EXPECTED_CHAIN=zerone-2 \
  EXPECTED_CONFIG="${TMP}/etc/nginx/conf.d/default.conf" \
  NGINX_START_LOG="${TMP}/nginx-start.log" NGINX_STOP_LOG="${TMP}/nginx-stop.log" \
  READINESS_CHECK_LOG="${TMP}/readiness-check.log" \
  READINESS_START_LOG="${TMP}/readiness-start.log" \
  READINESS_STOP_LOG="${TMP}/readiness-stop.log" READINESS_EXIT_IMMEDIATELY=1 \
  "${TMP}/entrypoint.sh" >"${TMP}/expected-helper-exit.log" 2>&1; then
  printf 'query gateway runtime test: helper exit did not fail the container\n' >&2
  exit 1
fi
grep -qx 'QUIT' "${TMP}/nginx-stop.log"

rm -f "${TMP}/nginx-start.log" "${TMP}/nginx-stop.log" \
  "${TMP}/readiness-start.log" "${TMP}/readiness-stop.log" \
  "${TMP}/readiness-check.log"
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-2-query \
  EXPECTED_CHAIN_ID=zerone-2 UPSTREAM_HOST=zerone-2-edge.internal \
  EXPECTED_ARCHIVE_HEIGHT='' EXPECTED_ARCHIVE_APP_HASH='' EXPECTED_ARCHIVE_BLOCK_HASH='' \
  EXPECTED_UPSTREAM=zerone-2-edge.internal EXPECTED_CHAIN=zerone-2 \
  EXPECTED_CONFIG="${TMP}/etc/nginx/conf.d/default.conf" \
  NGINX_START_LOG="${TMP}/nginx-start.log" NGINX_STOP_LOG="${TMP}/nginx-stop.log" \
  READINESS_CHECK_LOG="${TMP}/readiness-check.log" \
  READINESS_START_LOG="${TMP}/readiness-start.log" \
  READINESS_STOP_LOG="${TMP}/readiness-stop.log" NGINX_EXIT_IMMEDIATELY=1 \
  "${TMP}/entrypoint.sh" >"${TMP}/expected-nginx-exit.log" 2>&1; then
  printf 'query gateway runtime test: nginx exit did not fail the container\n' >&2
  exit 1
fi
grep -qx 'TERM' "${TMP}/readiness-stop.log"

if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-1-archive-query \
  EXPECTED_CHAIN_ID=zerone-1 UPSTREAM_HOST=zerone-1-archive.internal \
  EXPECTED_ARCHIVE_HEIGHT='' EXPECTED_ARCHIVE_APP_HASH='' EXPECTED_ARCHIVE_BLOCK_HASH='' \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: archive checkpoint pins were optional\n' >&2
  exit 1
fi
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-1-archive-query \
  EXPECTED_CHAIN_ID=zerone-1 UPSTREAM_HOST=zerone-1-archive.internal \
  EXPECTED_ARCHIVE_HEIGHT=42 EXPECTED_ARCHIVE_APP_HASH="${ARCHIVE_HASH}" \
  EXPECTED_ARCHIVE_BLOCK_HASH='' \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: archive block hash was optional\n' >&2
  exit 1
fi
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-1-archive-query \
  EXPECTED_CHAIN_ID=zerone-1 UPSTREAM_HOST=zerone-1-archive.internal \
  EXPECTED_ARCHIVE_HEIGHT=42 EXPECTED_ARCHIVE_APP_HASH="${ARCHIVE_HASH}" \
  EXPECTED_ARCHIVE_BLOCK_HASH="$(printf 'CD%.0s' {1..32})" \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: uppercase archive block hash was accepted\n' >&2
  exit 1
fi
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-2-query \
  EXPECTED_CHAIN_ID=zerone-2 UPSTREAM_HOST=zerone-2-edge.internal \
  EXPECTED_ARCHIVE_HEIGHT=42 EXPECTED_ARCHIVE_APP_HASH="${ARCHIVE_HASH}" \
  EXPECTED_ARCHIVE_BLOCK_HASH="${ARCHIVE_BLOCK_HASH}" \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: active gateway accepted archive pins\n' >&2
  exit 1
fi

if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-2-query \
  EXPECTED_CHAIN_ID=zerone-1 UPSTREAM_HOST=zerone-2-edge.internal \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: mismatched role/chain was accepted\n' >&2
  exit 1
fi
if PATH="${TMP}/bin:${PATH}" GATEWAY_ROLE=zerone-2-query \
  EXPECTED_CHAIN_ID=zerone-2 UPSTREAM_HOST='edge.internal;include bad' \
  "${TMP}/entrypoint.sh" >/dev/null 2>&1; then
  printf 'query gateway runtime test: unsafe upstream host was accepted\n' >&2
  exit 1
fi

grep -q 'limit_except GET HEAD' "${TEMPLATE_SOURCE}"
grep -q 'zerone_search_rate:10m rate=2r/s' "${TEMPLATE_SOURCE}"
grep -q 'Access-Control-Allow-Origin "\*" always' "${TEMPLATE_SOURCE}"
grep -q 'Access-Control-Expose-Headers "X-Cosmos-Block-Height, Grpc-Metadata-X-Cosmos-Block-Height" always' \
  "${TEMPLATE_SOURCE}"
# The semantic proof and served request must share one exact helper-provided
# address. Nginx may not independently resolve or use an origin upstream name.
# shellcheck disable=SC2016 # Assert nginx's literal variables.
grep -Fq 'auth_request_set $zerone_validated_6pn $upstream_http_x_zerone_validated_6pn;' \
  "${TEMPLATE_SOURCE}"
# shellcheck disable=SC2016 # Assert nginx's literal variables.
grep -Fq 'proxy_pass http://[$zerone_validated_6pn]:26657;' "${TEMPLATE_SOURCE}"
# shellcheck disable=SC2016 # Assert nginx's literal variables.
grep -Fq 'proxy_pass http://[$zerone_validated_6pn]:1317;' "${TEMPLATE_SOURCE}"
if grep -Eq 'resolver |zerone_rpc_origin|zerone_rest_origin|proxy_pass http://\$\{UPSTREAM_HOST\}' \
  "${TEMPLATE_SOURCE}"; then
  printf 'query gateway runtime test: nginx retains an independent origin resolution path\n' >&2
  exit 1
fi
grep -q 'send_timeout 15s;' "${TEMPLATE_SOURCE}"
grep -q 'proxy_pass_request_body off;' "${TEMPLATE_SOURCE}"
grep -q 'proxy_set_header Content-Length "";' "${TEMPLATE_SOURCE}"
grep -A8 'location = /_gateway_ready' "${TEMPLATE_SOURCE}" | grep -q 'internal;' || \
  { printf 'query gateway runtime test: readiness auth endpoint is public\n' >&2; exit 1; }
grep -A8 'location = /_gateway_ready' "${TEMPLATE_SOURCE}" | \
  grep -q 'proxy_pass http://127.0.0.1:18081/ready;' || \
  { printf 'query gateway runtime test: readiness auth endpoint misses helper\n' >&2; exit 1; }
if grep -Eq 'location .*dump_consensus_state|location .*consensus_state|location .*net_info' \
  "${TEMPLATE_SOURCE}"; then
  printf 'query gateway runtime test: private topology route is public\n' >&2
  exit 1
fi
if [ "$(grep -Ec 'location .*abci_query' "${TEMPLATE_SOURCE}")" != 2 ] || \
   ! grep -F -A2 'location ~ ^/abci_query/?$' "${TEMPLATE_SOURCE}" | \
     grep -q 'return 404;' || \
   ! grep -A2 'location .*cosmos/base/tendermint/v1beta1/abci_query' \
     "${TEMPLATE_SOURCE}" | grep -q 'return 404;'; then
  printf 'query gateway runtime test: generic ABCI query/simulate route is public\n' >&2
  exit 1
fi
grep -A6 'location = /gateway-health' "${TEMPLATE_SOURCE}" | \
  grep -q 'limit_except GET HEAD' || \
  { printf 'query gateway runtime test: health route lacks method control\n' >&2; exit 1; }
grep -A6 'location = /gateway-health' "${TEMPLATE_SOURCE}" | \
  grep -q 'limit_req zone=zerone_query_rate' || \
  { printf 'query gateway runtime test: health route bypasses rate limits\n' >&2; exit 1; }
grep -A6 'location = /gateway-health' "${TEMPLATE_SOURCE}" | \
  grep -q 'proxy_pass http://127.0.0.1:18081/health;' || \
  { printf 'query gateway runtime test: health route bypasses semantic helper\n' >&2; exit 1; }

# shellcheck disable=SC2016 # Assert nginx's literal variable.
origin_proxies=$(grep -Fc 'proxy_pass http://[$zerone_validated_6pn]:' "${TEMPLATE_SOURCE}")
auth_gates=$(grep -c 'auth_request /_gateway_ready;' "${TEMPLATE_SOURCE}")
if [ "${origin_proxies}" -eq 0 ] || [ "${origin_proxies}" != "${auth_gates}" ]; then
  printf 'query gateway runtime test: public origin routes are not all readiness-gated\n' >&2
  exit 1
fi
awk '
  /^[[:space:]]*location / { gated = 0 }
  /auth_request \/_gateway_ready;/ { gated = 1 }
  /proxy_pass http:\/\/\[\$zerone_validated_6pn\]:(26657|1317);/ && !gated { exit 1 }
' "${TEMPLATE_SOURCE}" || {
  printf 'query gateway runtime test: origin proxy lacks a local readiness gate\n' >&2
  exit 1
}

# Public RPC/REST is deny-by-default. Expensive search/history routes and
# unreviewed typed queries must not appear in a proxied allowlist.
grep -Fq 'cosmos/bank/v1beta1/balances/zrn1' "${TEMPLATE_SOURCE}"
grep -A2 'location / {' "${TEMPLATE_SOURCE}" | grep -q 'return 404;' || \
  { printf 'query gateway runtime test: REST catch-all is not closed\n' >&2; exit 1; }
if grep -Eq 'location .*block_search|location .*blockchain|location .*tx_search|location .*block_results|location .*genesis_chunked|location .*([^_]genesis[|/)])' \
  "${TEMPLATE_SOURCE}"; then
  printf 'query gateway runtime test: unbounded RPC history/search route is public\n' >&2
  exit 1
fi
for private_route in knowledge/v1/facts training proof_tree descendant_tree leaderboard simulate; do
  if grep -E "location .*${private_route}" "${TEMPLATE_SOURCE}" >/dev/null; then
    printf 'query gateway runtime test: unbounded REST route is public: %s\n' "${private_route}" >&2
    exit 1
  fi
done

grep -A6 'location .*/(block|block_by_hash|tx)' "${TEMPLATE_SOURCE}" | \
  grep -q 'limit_req zone=zerone_search_rate' || \
  { printf 'query gateway runtime test: heavy Comet responses are not throttled\n' >&2; exit 1; }
grep -A6 'location .*/(block|block_by_hash|tx)' "${TEMPLATE_SOURCE}" | \
  grep -q 'limit_conn zerone_query_conn 2;' || \
  { printf 'query gateway runtime test: heavy Comet responses lack low connection limit\n' >&2; exit 1; }

# Fixed REST reads reject all query-string variants. Pools stay in a dedicated
# location so pagination/count_total cannot be introduced through a broad regex.
grep -Fq 'location ~ ^/zerone/liquiditypool/v1/pools/?$' "${TEMPLATE_SOURCE}"
if grep -F 'cosmos/base/tendermint/v1beta1/(syncing|node_info)' "${TEMPLATE_SOURCE}" | \
  grep -q 'liquiditypool/v1/(pools|params)'; then
  printf 'query gateway runtime test: pools route was not split from fixed REST reads\n' >&2
  exit 1
fi
for location_pattern in \
  'location ~ ^/(cosmos/base/tendermint/v1beta1/(syncing|node_info)|cosmos/bank/v1beta1/denoms_metadata/uzrn|zerone/liquiditypool/v1/params)/?$' \
  'location ~ ^/zerone/liquiditypool/v1/pools/?$' \
  'location ~ "^/(cosmos/auth/v1beta1/accounts/zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}|zerone/auth/v1/account_identifier/zrn1[023456789acdefghjklmnpqrstuvwxyz]{38})/?$"'; do
  # shellcheck disable=SC2016 # Assert nginx's literal args variable.
  grep -F -A3 "${location_pattern}" "${TEMPLATE_SOURCE}" | \
    grep -Fq 'if ($args != "") { return 404; }' || {
      printf 'query gateway runtime test: fixed REST query strings are not rejected: %s\n' \
        "${location_pattern}" >&2
      exit 1
    }
done

# Native supply and account balance point reads accept no query variants.
for route in \
  'cosmos/bank/v1beta1/supply/by_denom' \
  'cosmos/bank/v1beta1/balances/zrn1'; do
  # shellcheck disable=SC2016 # Assert nginx's literal args variable.
  grep -A3 "location .*${route}" "${TEMPLATE_SOURCE}" | \
    grep -Fq 'if ($args != "denom=uzrn") { return 404; }' || {
      printf 'query gateway runtime test: native-denom query is not exact: %s\n' "${route}" >&2
      exit 1
    }
done

PRIVATE_PROFILE="${ROOT}/deploy/query-gateway/fly.zerone-2.private.example.toml"
if grep -Eq '^\[\[services\]\]$|^\[http_service\]$' "${PRIVATE_PROFILE}"; then
  printf 'query gateway runtime test: private soak profile exposes a service\n' >&2
  exit 1
fi
for profile in \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  "${ROOT}/deploy/query-gateway/fly.zerone-1-archive.public.example.toml"; do
  [ "$(grep -c '^\[\[services\]\]$' "${profile}")" = 1 ]
  grep -q '^  internal_port = 8080$' "${profile}"
  grep -q '^    handlers = \["tls", "http"\]$' "${profile}"
  if grep -Eq '^  internal_port = (26657|1317|9090)$' "${profile}"; then
    printf 'query gateway runtime test: profile bypasses gateway origin\n' >&2
    exit 1
  fi
done

ARCHIVE_PROFILE="${ROOT}/deploy/query-gateway/fly.zerone-1-archive.public.example.toml"
grep -q '^  EXPECTED_ARCHIVE_HEIGHT = "REPLACE_WITH_A"$' "${ARCHIVE_PROFILE}"
grep -q '^  EXPECTED_ARCHIVE_APP_HASH = "REPLACE_WITH_LOWERCASE_POST_A_APP_HASH"$' \
  "${ARCHIVE_PROFILE}"
grep -q '^  EXPECTED_ARCHIVE_BLOCK_HASH = "REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH"$' \
  "${ARCHIVE_PROFILE}"

printf 'query gateway runtime tests: PASS\n'
