#!/bin/sh
# Query-only origin gateway for the zerone-2 edge and frozen zerone-1 archive.
set -eu
export LC_ALL=C
umask 077

readonly TEMPLATE=/etc/zerone-query-gateway/default.conf.template
readonly CONFIG=/etc/nginx/conf.d/default.conf
readonly READINESS_BIN=/usr/local/bin/zerone-query-readiness

die() {
  printf 'zerone-query-gateway: ERROR: %s\n' "$*" >&2
  exit 1
}

require_regular_file() {
  file="$1"
  label="$2"
  [ -f "${file}" ] && [ ! -L "${file}" ] || \
    die "${label} must be a regular non-symlink file: ${file}"
}

EXPECTED_CHAIN_ID="${EXPECTED_CHAIN_ID:-}"
GATEWAY_ROLE="${GATEWAY_ROLE:-}"
UPSTREAM_HOST="${UPSTREAM_HOST:-}"
EXPECTED_ARCHIVE_HEIGHT="${EXPECTED_ARCHIVE_HEIGHT:-}"
EXPECTED_ARCHIVE_APP_HASH="${EXPECTED_ARCHIVE_APP_HASH:-}"
EXPECTED_ARCHIVE_BLOCK_HASH="${EXPECTED_ARCHIVE_BLOCK_HASH:-}"

case "${GATEWAY_ROLE}:${EXPECTED_CHAIN_ID}" in
  zerone-2-query:zerone-2)
    [ -z "${EXPECTED_ARCHIVE_HEIGHT}" ] && \
      [ -z "${EXPECTED_ARCHIVE_APP_HASH}" ] && \
      [ -z "${EXPECTED_ARCHIVE_BLOCK_HASH}" ] || \
      die "active gateway forbids archive checkpoint expectations"
    ;;
  zerone-1-archive-query:zerone-1)
    case "${EXPECTED_ARCHIVE_HEIGHT}" in
      ''|0|0*|*[!0-9]*) die "EXPECTED_ARCHIVE_HEIGHT must be a canonical positive integer" ;;
    esac
    printf '%s\n' "${EXPECTED_ARCHIVE_APP_HASH}" | grep -Eq '^[0-9a-f]{64}$' || \
      die "EXPECTED_ARCHIVE_APP_HASH must be exactly 64 lowercase hex characters"
    printf '%s\n' "${EXPECTED_ARCHIVE_BLOCK_HASH}" | grep -Eq '^[0-9a-f]{64}$' || \
      die "EXPECTED_ARCHIVE_BLOCK_HASH must be exactly 64 lowercase hex characters"
    ;;
  *) die "GATEWAY_ROLE and EXPECTED_CHAIN_ID are not an approved pair" ;;
esac
printf '%s\n' "${UPSTREAM_HOST}" | grep -Eq \
  '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.internal$' || \
  die "UPSTREAM_HOST must be a lowercase Fly .internal DNS name"

command -v envsubst >/dev/null 2>&1 || die "envsubst is required"
command -v nginx >/dev/null 2>&1 || die "nginx is required"
require_regular_file "${TEMPLATE}" "gateway configuration template"
require_regular_file "${READINESS_BIN}" "semantic readiness helper"
[ -x "${READINESS_BIN}" ] || die "semantic readiness helper must be executable"
"${READINESS_BIN}" -check-config || die "semantic readiness configuration is invalid"
[ ! -L "${CONFIG}" ] || die "nginx output configuration must not be a symlink"

tmp=$(mktemp /etc/nginx/conf.d/.zerone-query-gateway.XXXXXX)
# shellcheck disable=SC2329 # Invoked by trap.
cleanup_render() { rm -f "${tmp}"; }
trap cleanup_render EXIT
trap 'exit 1' HUP INT TERM
# shellcheck disable=SC2016 # envsubst receives the literal allowlist.
envsubst '${EXPECTED_CHAIN_ID}' < "${TEMPLATE}" > "${tmp}" || \
  die "could not render gateway configuration"
chmod 0644 "${tmp}"
# The image-owned destination directory is not shared with callers. Install
# the candidate, then make nginx parse the exact file it will serve.
mv "${tmp}" "${CONFIG}"
tmp=""
nginx -t -q -c /etc/nginx/nginx.conf || die "rendered gateway configuration is invalid"
trap - EXIT HUP INT TERM

READINESS_PID=""
NGINX_PID=""

process_running() {
  [ -n "$1" ] && kill -0 "$1" 2>/dev/null
}

stop_children() {
  # `nginx -s quit` asks the master to drain active connections. The readiness
  # helper receives TERM so no new request can pass its gate while nginx drains.
  if process_running "${NGINX_PID}"; then
    nginx -s quit >/dev/null 2>&1 || kill -TERM "${NGINX_PID}" 2>/dev/null || true
  fi
  if process_running "${READINESS_PID}"; then
    kill -TERM "${READINESS_PID}" 2>/dev/null || true
  fi

  supervisor_attempt=0
  while { process_running "${NGINX_PID}" || process_running "${READINESS_PID}"; } && \
    [ "${supervisor_attempt}" -lt 10 ]; do
    sleep 1
    supervisor_attempt=$((supervisor_attempt + 1))
  done
  if process_running "${NGINX_PID}"; then
    kill -KILL "${NGINX_PID}" 2>/dev/null || true
  fi
  if process_running "${READINESS_PID}"; then
    kill -KILL "${READINESS_PID}" 2>/dev/null || true
  fi
  [ -z "${NGINX_PID}" ] || wait "${NGINX_PID}" 2>/dev/null || true
  [ -z "${READINESS_PID}" ] || wait "${READINESS_PID}" 2>/dev/null || true
}

# shellcheck disable=SC2329 # Invoked by trap.
shutdown() {
  trap - HUP INT TERM
  stop_children
  exit 0
}

trap shutdown HUP INT TERM
"${READINESS_BIN}" &
READINESS_PID=$!
unset UPSTREAM_HOST EXPECTED_ARCHIVE_HEIGHT EXPECTED_ARCHIVE_APP_HASH EXPECTED_ARCHIVE_BLOCK_HASH
nginx -g 'daemon off;' &
NGINX_PID=$!

while process_running "${READINESS_PID}" && process_running "${NGINX_PID}"; do
  sleep 1
done
trap - HUP INT TERM
stop_children
die "semantic readiness helper or nginx exited unexpectedly"
