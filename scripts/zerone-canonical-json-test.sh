#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
HELPER="${ROOT}/scripts/zerone-canonical-json.sh"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-canonical-json-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

printf '{"z":1,"a":"ok"}\n' > "${TMP}/draft.json"
"${HELPER}" "${TMP}/draft.json" "${TMP}/canonical.json" >/dev/null
printf '{"a":"ok","z":1}\n' > "${TMP}/expected.json"
cmp "${TMP}/canonical.json" "${TMP}/expected.json"

printf 'must survive\n' > "${TMP}/sentinel.json"
sentinel_before=$(sha256_file "${TMP}/sentinel.json")
if "${HELPER}" "${TMP}/draft.json" "${TMP}/sentinel.json" >/dev/null 2>&1; then
  printf 'canonical JSON test: preexisting output was overwritten\n' >&2
  exit 1
fi
[ "$(sha256_file "${TMP}/sentinel.json")" = "${sentinel_before}" ]

printf '{"value":"REPLACE_WITH_REAL_VALUE"}\n' > "${TMP}/placeholder.json"
if "${HELPER}" "${TMP}/placeholder.json" "${TMP}/placeholder-out.json" \
  >/dev/null 2>&1; then
  printf 'canonical JSON test: unresolved placeholder was accepted\n' >&2
  exit 1
fi
[ ! -e "${TMP}/placeholder-out.json" ]

ln -s draft.json "${TMP}/draft-link.json"
if "${HELPER}" "${TMP}/draft-link.json" "${TMP}/symlink-input-out.json" \
  >/dev/null 2>&1; then
  printf 'canonical JSON test: symlinked input was accepted\n' >&2
  exit 1
fi

ln -s missing-target "${TMP}/dangling-output.json"
if "${HELPER}" "${TMP}/draft.json" "${TMP}/dangling-output.json" \
  >/dev/null 2>&1; then
  printf 'canonical JSON test: dangling output symlink was followed\n' >&2
  exit 1
fi
[ -L "${TMP}/dangling-output.json" ]

printf 'canonical JSON tests: PASS\n'
