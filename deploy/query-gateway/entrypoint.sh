#!/usr/bin/env bash
# Query-only origin gateway for the zerone-2 edge and frozen zerone-1 archive.
set -euo pipefail
export LC_ALL=C
umask 077

readonly TEMPLATE=/etc/zerone-query-gateway/default.conf.template
readonly CONFIG=/etc/nginx/conf.d/default.conf

die() {
  printf 'zerone-query-gateway: ERROR: %s\n' "$*" >&2
  exit 1
}

require_regular_file() {
  local file="$1" label="$2"
  [ -f "${file}" ] && [ ! -L "${file}" ] || \
    die "${label} must be a regular non-symlink file: ${file}"
}

EXPECTED_CHAIN_ID="${EXPECTED_CHAIN_ID:-}"
GATEWAY_ROLE="${GATEWAY_ROLE:-}"
UPSTREAM_HOST="${UPSTREAM_HOST:-}"

case "${GATEWAY_ROLE}:${EXPECTED_CHAIN_ID}" in
  zerone-2-query:zerone-2|zerone-1-archive-query:zerone-1) ;;
  *) die "GATEWAY_ROLE and EXPECTED_CHAIN_ID are not an approved pair" ;;
esac
[[ "${UPSTREAM_HOST}" =~ ^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.internal$ ]] || \
  die "UPSTREAM_HOST must be a lowercase Fly .internal DNS name"

command -v envsubst >/dev/null 2>&1 || die "envsubst is required"
command -v nginx >/dev/null 2>&1 || die "nginx is required"
require_regular_file "${TEMPLATE}" "gateway configuration template"
[ ! -L "${CONFIG}" ] || die "nginx output configuration must not be a symlink"

tmp=$(mktemp /etc/nginx/conf.d/.zerone-query-gateway.XXXXXX)
cleanup() { rm -f "${tmp}"; }
trap cleanup EXIT HUP INT TERM
# shellcheck disable=SC2016 # envsubst receives the literal allowlist.
envsubst '${UPSTREAM_HOST} ${EXPECTED_CHAIN_ID}' < "${TEMPLATE}" > "${tmp}" || \
  die "could not render gateway configuration"
chmod 0644 "${tmp}"
# The image-owned destination directory is not shared with callers. Install
# the candidate, then make nginx parse the exact file it will serve.
mv "${tmp}" "${CONFIG}"
tmp=""
nginx -t -q -c /etc/nginx/nginx.conf || die "rendered gateway configuration is invalid"
trap - EXIT HUP INT TERM

unset UPSTREAM_HOST
exec nginx -g 'daemon off;'
