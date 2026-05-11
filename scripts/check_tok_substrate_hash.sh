#!/usr/bin/env bash
#
# check_tok_substrate_hash.sh — verify TOK_SUBSTRATE.md has not drifted from
# the pinned hash in .tok-substrate-hash.
#
# Mirror of check_strange_loop_hash.sh applied to the training-resource
# doctrine. The on-chain pin lives in x/knowledge/types/tok_substrate_creed.go
# (Go-side build-time canonical structure); this script catches accidental
# drift in PRs even before the chain has the canonical record on-chain. The
# cross-stack meta-test TestToKSubstrate_DoctrineAndContractStayInSync also
# enforces this in CI; the script provides the same enforcement on local
# dev machines via `make creed-check`.
#
# To intentionally amend the doctrine:
#   1. Edit docs/TOK_SUBSTRATE.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .tok-substrate-hash with the new value.
#   4. Update x/knowledge/types/tok_substrate_creed.go if commitment count
#      or structure changed (ToK substrate is built on binding, not on
#      number alone — amendment is rare but possible).
#   5. Update tests/cross_stack/tok_substrate_invariants_test.go if any
#      TestTC* function names need to match new commitment numbers.
#   6. Commit all files together with a message naming what changed.
#
# Reviewers see the .tok-substrate-hash bump as the visible signal that
# the doctrine text has changed, prompting full diff review.

set -euo pipefail

DOCTRINE_FILE="docs/TOK_SUBSTRATE.md"
HASH_FILE=".tok-substrate-hash"

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
TOK_SUBSTRATE.md hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the doctrine, update $HASH_FILE to:
  $ACTUAL

The change should be visible in your PR diff so reviewers see both
the doctrine text change AND the hash bump in the same commit. ToK
substrate is built on binding, not number alone.
EOF
  exit 1
fi

echo "tok-substrate hash check ok ($ACTUAL)"
