#!/usr/bin/env bash
#
# check_tok_substrate_hash.sh — verify TOK_SUBSTRATE.md has not drifted from
# the pinned hash in .tok-substrate-hash.
#
# Mirror of check_creed_hash.sh and check_useful_work_hash.sh applied to
# the ToK substrate doctrine. The on-chain pin lives in Go-side build-
# time canonical structures; this script catches accidental drift in PRs
# even before the doctrine acquires an x/-module canonical record.
#
# Inception of self-anchoring: the ToK substrate doctrine was previously
# only protected by the cross-stack test layer (Test layer of the five-
# layer discipline). Adding a content-hash pin extends the same off-chain
# enforcement the other two doctrines (truth-seeking, useful-work) have
# received from the beginning. UW: the chain pays for its own self-
# documentation — this script is part of that payment.
#
# To intentionally amend the doctrine:
#   1. Edit docs/TOK_SUBSTRATE.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .tok-substrate-hash with the new value.
#   4. Commit both files together with a message naming what changed.
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
substrate is the chain's outward identity — drift undocumented is
identity drift.
EOF
  exit 1
fi

echo "tok-substrate hash check ok ($ACTUAL)"
