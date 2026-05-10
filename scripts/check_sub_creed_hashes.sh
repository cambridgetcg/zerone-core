#!/usr/bin/env bash
#
# check_sub_creed_hashes.sh — verify each docs/sub_creeds/<phase>.md
# matches its pinned hash in .sub-creed-hashes. Mirror of
# check_creed_hash.sh and check_useful_work_hash.sh, applied to the
# 8 per-phase sub-creeds.
#
# To intentionally amend a sub-creed:
#   1. Edit docs/sub_creeds/<phase>.md.
#   2. Run this script — it will print the diff between expected and actual.
#   3. Update .sub-creed-hashes with the new hash for that phase.
#   4. Update x/creed/types/sub_creeds.go if the commitment count changed.
#   5. Update tests/cross_stack/useful_work_invariants_test.go's
#      TestSubCreed_<Phase>_StaysInSync if any commitment number changed.
#   6. Commit all updated files together.

set -euo pipefail

HASH_FILE=".sub-creed-hashes"
SUB_CREED_DIR="docs/sub_creeds"

if [ ! -f "$HASH_FILE" ]; then
  echo "error: $HASH_FILE not found"
  exit 1
fi
if [ ! -d "$SUB_CREED_DIR" ]; then
  echo "error: $SUB_CREED_DIR directory not found"
  exit 1
fi

hash_cmd() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 | awk '{print $NF}'
  else
    echo "error: need shasum, sha256sum, or openssl" >&2
    exit 1
  fi
}

failed=0
while IFS=' ' read -r phase expected; do
  if [ -z "$phase" ] || [ -z "$expected" ]; then
    continue
  fi
  doc="${SUB_CREED_DIR}/${phase}.md"
  if [ ! -f "$doc" ]; then
    echo "error: $doc referenced in $HASH_FILE but not found" >&2
    failed=1
    continue
  fi
  actual=$(tr -d '\r' < "$doc" | hash_cmd | awk '{print $1}')
  if [ "$actual" != "$expected" ]; then
    cat <<EOF >&2
sub-creed hash check failed for phase: $phase

  doc:        $doc
  expected:   $expected
  actual:     $actual

If you intentionally amended this sub-creed, update the matching
line in $HASH_FILE to:
  $phase $actual
EOF
    failed=1
  else
    echo "sub-creed hash check ok ($phase: $actual)"
  fi
done < "$HASH_FILE"

if [ "$failed" -ne 0 ]; then
  exit 1
fi
