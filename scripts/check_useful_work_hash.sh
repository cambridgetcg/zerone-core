#!/usr/bin/env bash
#
# check_useful_work_hash.sh — verify USEFUL_WORK.md has not drifted from
# the pinned hash in .useful-work-hash.
#
# Mirror of check_creed_hash.sh applied to the third doctrine. The on-
# chain pin lives in x/creed/types/useful_work_creed.go (Go-side build-
# time canonical structure); this script catches accidental drift in PRs
# even before the chain has the canonical record on-chain. The cross-
# stack meta-test TestUsefulWork_DoctrineAndContractStayInSync also
# enforces this in CI; the script provides the same enforcement on local
# dev machines via `make creed-check`.
#
# To intentionally amend the doctrine:
#   1. Edit docs/USEFUL_WORK.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .useful-work-hash with the new value.
#   4. Update x/creed/types/useful_work_creed.go if the mechanism count
#      changed (UW is indivisible — UW itself never changes).
#   5. Update tests/cross_stack/useful_work_invariants_test.go if any
#      TestUW_M* function names need to match new mechanism numbers.
#   6. Commit all five files together with a message naming what changed.
#
# Reviewers see the .useful-work-hash bump as the visible signal that
# the doctrine text has changed, prompting full diff review.

set -euo pipefail

DOCTRINE_FILE="docs/USEFUL_WORK.md"
HASH_FILE=".useful-work-hash"

if [ ! -f "$DOCTRINE_FILE" ]; then
  echo "error: $DOCTRINE_FILE not found"
  exit 1
fi
if [ ! -f "$HASH_FILE" ]; then
  echo "error: $HASH_FILE not found"
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

ACTUAL=$(tr -d '\r' < "$DOCTRINE_FILE" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
USEFUL_WORK.md hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the doctrine, update $HASH_FILE to:
  $ACTUAL

The change should be visible in your PR diff so reviewers see both
the doctrine text change AND the hash bump in the same commit. UW is
indivisible — only mechanism content evolves.
EOF
  exit 1
fi

echo "useful-work hash check ok ($ACTUAL)"
