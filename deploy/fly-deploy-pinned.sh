#!/usr/bin/env bash
set -euo pipefail

die() {
  printf 'fly-deploy-pinned: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage:
  deploy/fly-deploy-pinned.sh --check CONFIG EXPECTED_APP EXPECTED_IMAGE EXPECTED_ROLE EXPECTED_CONFIG_SHA256
  deploy/fly-deploy-pinned.sh CONFIG EXPECTED_APP EXPECTED_IMAGE EXPECTED_ROLE EXPECTED_CONFIG_SHA256

The deploy form accepts no flyctl overrides. It snapshots the config, requires
one exact [build].image registry digest and one exact role, binds both to the
signed phase-authority expectations, and passes that same digest to flyctl.
USAGE
  exit 2
}

MODE=deploy
case "$#" in
  5)
    CONFIG=$1
    EXPECTED_APP=$2
    EXPECTED_IMAGE=$3
    EXPECTED_ROLE=$4
    EXPECTED_CONFIG_SHA256=$5
    ;;
  6)
    [ "$1" = "--check" ] || usage
    MODE=check
    CONFIG=$2
    EXPECTED_APP=$3
    EXPECTED_IMAGE=$4
    EXPECTED_ROLE=$5
    EXPECTED_CONFIG_SHA256=$6
    ;;
  *)
    usage
    ;;
esac

[[ "${EXPECTED_APP}" =~ ^[a-z0-9][a-z0-9-]{0,62}$ ]] || \
  die "expected app must be a lowercase Fly app name"
[[ "${EXPECTED_CONFIG_SHA256}" =~ ^[0-9a-f]{64}$ ]] || \
  die "expected config SHA-256 must be 64 lowercase hexadecimal characters"

case "${EXPECTED_ROLE}" in
  signer|observer|archive-candidate|archive|validator|edge|zerone-2-query|zerone-1-archive-query) ;;
  *) die "expected role is not an approved deployment role" ;;
esac

[ -f "${CONFIG}" ] || die "config is not a regular file: ${CONFIG}"
[ ! -L "${CONFIG}" ] || die "config symlinks are forbidden: ${CONFIG}"

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-fly-deploy.XXXXXX")
trap 'rm -rf "${TMP_ROOT}"' EXIT HUP INT TERM
SNAPSHOT="${TMP_ROOT}/fly.toml"
install -m 0600 "${CONFIG}" "${SNAPSHOT}"
if command -v sha256sum >/dev/null 2>&1; then
  CONFIG_SHA256=$(sha256sum "${SNAPSHOT}" | awk '{print $1}')
else
  CONFIG_SHA256=$(shasum -a 256 "${SNAPSHOT}" | awk '{print $1}')
fi
[ "${CONFIG_SHA256}" = "${EXPECTED_CONFIG_SHA256}" ] || \
  die "snapshotted config does not match the signed phase-authority SHA-256"

if ! APP_LINE=$(awk '
  BEGIN { before_section = 1 }
  /^[[:space:]]*\[/ { before_section = 0 }
  /^[[:space:]]*app[[:space:]]*=/ {
    if (before_section) {
      app_count++
      print
    } else {
      outside_count++
    }
  }
  END { if (app_count != 1 || outside_count != 0) exit 42 }
' "${SNAPSHOT}"); then
  die "config must contain exactly one root app assignment"
fi
APP=$(printf '%s\n' "${APP_LINE}" | sed -nE \
  's/^[[:space:]]*app[[:space:]]*=[[:space:]]*"([a-z0-9][a-z0-9-]{0,62})"[[:space:]]*(#.*)?$/\1/p')
[ -n "${APP}" ] || die "root app must be one plain lowercase Fly app name"
[ "${APP}" = "${EXPECTED_APP}" ] || \
  die "config app does not match the signed phase-authority app"

if ! IMAGE_LINE=$(awk '
  function normalized_section(line) {
    sub(/[[:space:]]*#.*/, "", line)
    gsub(/[[:space:]]/, "", line)
    return line
  }
  /^[[:space:]]*\[.*\][[:space:]]*(#.*)?$/ {
    in_build = (normalized_section($0) == "[build]")
    next
  }
  /^[[:space:]]*image[[:space:]]*=/ {
    if (in_build) {
      build_count++
      print
    } else {
      outside_count++
    }
  }
  END {
    if (build_count != 1 || outside_count != 0) exit 42
  }
' "${SNAPSHOT}"); then
  die "config must contain exactly one image assignment, inside [build]"
fi

IMAGE=$(printf '%s\n' "${IMAGE_LINE}" | sed -nE \
  's/^[[:space:]]*image[[:space:]]*=[[:space:]]*"([^"]+)"[[:space:]]*(#.*)?$/\1/p')
[ -n "${IMAGE}" ] || die "[build].image must be one plain double-quoted reference"

case "${IMAGE}" in
  *@sha256:*) ;;
  *) die "[build].image must be registry/repository@sha256:<64-lowercase-hex>" ;;
esac

REPOSITORY=${IMAGE%@sha256:*}
DIGEST=${IMAGE##*@sha256:}
[ "${IMAGE}" = "${REPOSITORY}@sha256:${DIGEST}" ] || \
  die "[build].image contains an ambiguous digest separator"
case "${REPOSITORY}" in
  *'@'*) die "repository contains an unexpected @ separator" ;;
esac

[[ "${DIGEST}" =~ ^[0-9a-f]{64}$ ]] || \
  die "image digest must be exactly 64 lowercase hexadecimal characters"
[[ "${REPOSITORY}" =~ ^[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:[0-9]{1,5})?/[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$ ]] || \
  die "image repository must be a lowercase registry/repository path without a tag"
[ "${IMAGE}" = "${EXPECTED_IMAGE}" ] || \
  die "config image does not match the signed phase-authority image"

if ! ROLE_LINE=$(awk '
  function normalized_section(line) {
    sub(/[[:space:]]*#.*/, "", line)
    gsub(/[[:space:]]/, "", line)
    return line
  }
  /^[[:space:]]*\[.*\][[:space:]]*(#.*)?$/ {
    in_env = (normalized_section($0) == "[env]")
    next
  }
  /^[[:space:]]*(NODE_ROLE|GATEWAY_ROLE)[[:space:]]*=/ {
    if (in_env) {
      role_count++
      print
    } else {
      outside_count++
    }
  }
  END { if (role_count != 1 || outside_count != 0) exit 42 }
' "${SNAPSHOT}"); then
  die "config must contain exactly one NODE_ROLE or GATEWAY_ROLE inside [env]"
fi
ROLE=$(printf '%s\n' "${ROLE_LINE}" | sed -nE \
  's/^[[:space:]]*(NODE_ROLE|GATEWAY_ROLE)[[:space:]]*=[[:space:]]*"([^" ]+)"[[:space:]]*(#.*)?$/\2/p')
[ -n "${ROLE}" ] || die "deployment role must be one plain double-quoted value"
[ "${ROLE}" = "${EXPECTED_ROLE}" ] || \
  die "config role does not match the signed phase-authority role"

if [ "${MODE}" = "check" ]; then
  printf '%s\n' "${IMAGE}"
  exit 0
fi

FLYCTL=$(command -v flyctl) || die "flyctl is not installed"
printf 'fly-deploy-pinned: deploying immutable image %s\n' "${IMAGE}" >&2
unset FLY_APP FLY_APP_NAME FLY_CONFIG
"${FLYCTL}" deploy --app "${APP}" --config "${SNAPSHOT}" --image "${IMAGE}"
