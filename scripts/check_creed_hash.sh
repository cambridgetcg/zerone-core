#!/usr/bin/env bash
#
# check_creed_hash.sh — verify TRUTH_SEEKING.md has not drifted from
# the pinned hash in .creed-hash.
#
# This is the off-chain stage of creed protection (commitment 19,
# pre-x/creed). The on-chain pin lives in x/creed.PinnedCreed once the
# module ships; this script catches accidental drift in PRs even
# before the chain has the canonical record.
#
# To intentionally amend the creed:
#   1. Edit docs/TRUTH_SEEKING.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .creed-hash with the new value.
#   4. Commit both files together with a message naming which
#      commitment(s) you are NEW / AMEND / ARCHIVE.
#
# Reviewers see the .creed-hash bump as the visible signal that
# the creed text has changed, prompting full diff review.

set -euo pipefail

CREED_FILE="docs/TRUTH_SEEKING.md"
HASH_FILE=".creed-hash"

if [ ! -f "$CREED_FILE" ]; then
  echo "error: $CREED_FILE not found"
  exit 1
fi
if [ ! -f "$HASH_FILE" ]; then
  echo "error: $HASH_FILE not found"
  exit 1
fi

# Pick a portable sha256 tool. shasum is on macOS by default;
# sha256sum is on Linux by default; openssl is on both.
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

# Normalize: strip CR (so Windows line endings don't change the hash),
# then sha256.
ACTUAL=$(tr -d '\r' < "$CREED_FILE" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
TRUTH_SEEKING.md hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the creed, update $HASH_FILE to:
  $ACTUAL

The change should be visible in your PR diff so reviewers see both
the creed text change AND the hash bump in the same commit. This is
the off-chain enforcement of commitment 6 extended to the creed
itself: no individual can unilaterally amend the chain's voice
without the change being structurally surfaced.
EOF
  exit 1
fi

echo "creed hash check ok ($ACTUAL)"
