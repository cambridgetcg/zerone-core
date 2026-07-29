#!/usr/bin/env bash
# Emit one immutable canonical JSON payload without overwriting any prior file.
set -euo pipefail
export LC_ALL=C
umask 077

die() {
  printf 'zerone canonical JSON: ERROR: %s\n' "$*" >&2
  exit 1
}

[ "$#" -eq 2 ] || die "usage: zerone-canonical-json.sh INPUT_DRAFT OUTPUT_JSON"
INPUT=$1
REQUESTED_OUTPUT=$2
[ -f "${INPUT}" ] && [ ! -L "${INPUT}" ] || \
  die "input must be a regular non-symlink file"
command -v jq >/dev/null 2>&1 || die "jq is required"
command -v ln >/dev/null 2>&1 || die "ln is required for atomic no-replace publication"

OUTPUT_PARENT=$(cd "$(dirname "${REQUESTED_OUTPUT}")" && pwd) || \
  die "output parent must already exist"
[ -d "${OUTPUT_PARENT}" ] && [ ! -L "${OUTPUT_PARENT}" ] || \
  die "output parent must be a non-symlink directory"
OUTPUT_BASENAME=$(basename "${REQUESTED_OUTPUT}")
case "${OUTPUT_BASENAME}" in ''|.|..) die "output filename is invalid" ;; esac
OUTPUT="${OUTPUT_PARENT}/${OUTPUT_BASENAME}"
[ ! -e "${OUTPUT}" ] && [ ! -L "${OUTPUT}" ] || \
  die "output already exists; refusing overwrite"

INPUT_SNAPSHOT=$(mktemp "${OUTPUT_PARENT}/.zerone-canonical-input.XXXXXX")
TMP=""
cleanup() {
  [ -z "${TMP}" ] || rm -f "${TMP}"
  [ -z "${INPUT_SNAPSHOT}" ] || rm -f "${INPUT_SNAPSHOT}"
}
trap cleanup EXIT HUP INT TERM
cp -P "${INPUT}" "${INPUT_SNAPSHOT}" || die "could not freeze input draft"
[ -f "${INPUT_SNAPSHOT}" ] && [ ! -L "${INPUT_SNAPSHOT}" ] || \
  die "frozen input draft is not a regular file"
chmod 0400 "${INPUT_SNAPSHOT}"

if jq -e '.. | strings | select(contains("REPLACE"))' "${INPUT_SNAPSHOT}" >/dev/null; then
  die "input contains an unresolved REPLACE placeholder"
fi

TMP=$(mktemp "${OUTPUT_PARENT}/.zerone-canonical-json.XXXXXX")
jq --sort-keys --compact-output . "${INPUT_SNAPSHOT}" > "${TMP}" || \
  die "input is not valid JSON"
chmod 0600 "${TMP}"
cmp "${TMP}" <(jq --sort-keys --compact-output . "${INPUT_SNAPSHOT}") || \
  die "canonical JSON reproduction changed during publication"

# Hard-link creation is atomic and fails if the final path appeared after the
# early check. The temporary file lives in the same directory/filesystem.
ln "${TMP}" "${OUTPUT}" || die "could not atomically publish without overwrite"
rm -f "${TMP}"
rm -f "${INPUT_SNAPSHOT}"
INPUT_SNAPSHOT=""
TMP=""
trap - EXIT HUP INT TERM

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${OUTPUT}"
else
  shasum -a 256 "${OUTPUT}"
fi
