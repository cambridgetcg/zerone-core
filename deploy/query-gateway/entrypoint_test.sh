#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
SOURCE="${ROOT}/deploy/query-gateway/entrypoint.sh"
TEMPLATE_SOURCE="${ROOT}/deploy/query-gateway/default.conf.template"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-query-gateway-runtime-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM
mkdir -p "${TMP}/bin" "${TMP}/etc/nginx/conf.d" "${TMP}/etc/gateway"
cp "${TEMPLATE_SOURCE}" "${TMP}/etc/gateway/default.conf.template"

sed \
  -e "s|readonly TEMPLATE=/etc/zerone-query-gateway/default.conf.template|readonly TEMPLATE=${TMP}/etc/gateway/default.conf.template|" \
  -e "s|readonly CONFIG=/etc/nginx/conf.d/default.conf|readonly CONFIG=${TMP}/etc/nginx/conf.d/default.conf|" \
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
  grep -q "http://${EXPECTED_UPSTREAM}:26657/status" "${EXPECTED_CONFIG}"
  grep -q "X-Zerone-Expected-Chain \"${EXPECTED_CHAIN}\"" "${EXPECTED_CONFIG}"
  ! grep -q '\${UPSTREAM_HOST}\|\${EXPECTED_CHAIN_ID}' "${EXPECTED_CONFIG}"
  exit 0
fi
[ -z "${UPSTREAM_HOST+x}" ] || exit 90
printf '%s\n' "$@" > "${NGINX_START_LOG}"
EOF
chmod +x "${TMP}/bin/envsubst" "${TMP}/bin/nginx"

PATH="${TMP}/bin:${PATH}" \
GATEWAY_ROLE=zerone-2-query EXPECTED_CHAIN_ID=zerone-2 \
UPSTREAM_HOST=zerone-2-edge.internal \
EXPECTED_UPSTREAM=zerone-2-edge.internal EXPECTED_CHAIN=zerone-2 \
EXPECTED_CONFIG="${TMP}/etc/nginx/conf.d/default.conf" \
NGINX_START_LOG="${TMP}/nginx-start.log" \
  "${TMP}/entrypoint.sh"
grep -qx -- '-g' "${TMP}/nginx-start.log"
grep -qx -- 'daemon off;' "${TMP}/nginx-start.log"

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
# shellcheck disable=SC2016 # Assert the unexpanded template placeholder.
grep -q 'proxy_pass http://${UPSTREAM_HOST}:26657/status' "${TEMPLATE_SOURCE}"
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

printf 'query gateway runtime tests: PASS\n'
