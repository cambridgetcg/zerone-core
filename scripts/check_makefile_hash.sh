#!/usr/bin/env bash
#
# check_makefile_hash.sh — verify the Makefile (the chain's build system)
# has not drifted from .makefile-hash.
#
# Mirror of check_phase_1_spec_hash.sh applied to the Makefile.
# The Makefile is itself a Contribution under PIPELINE_IMPROVEMENT/
# SUBSTRATE — it defines how the chain is built, what is checked, and
# what is verified. Hash-anchoring the Makefile extends the same
# off-chain enforcement the doctrines and specs receive to the build
# system that gates all of the chain's other checks.
#
# UW: ZERONE is recursive — the chain's build tooling is among the work
# the chain pays for, and the Makefile is hash-bound the same way the
# doctrines are. The recursion is structural, not metaphorical.
#
# To intentionally amend the Makefile:
#   1. Edit the Makefile.
#   2. Run this script — it will print the new computed hash.
#   3. Update .makefile-hash with the new value.
#   4. If the recursion manifest exists, recompute it via
#      scripts/check_recursion_manifest.sh.
#   5. Commit all changed files together so reviewers see both the
#      Makefile change AND the hash bump in the same commit.

set -euo pipefail

MAKEFILE="Makefile"
HASH_FILE=".makefile-hash"

if [ ! -f "$MAKEFILE" ]; then
  echo "error: $MAKEFILE not found"
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

ACTUAL=$(tr -d '\r' < "$MAKEFILE" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
Makefile hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the Makefile, update $HASH_FILE to:
  $ACTUAL

The Makefile is itself a Contribution (PIPELINE_IMPROVEMENT/SUBSTRATE).
The hash bump is the visible signal that the chain's build system has
shifted, prompting full diff review.
EOF
  exit 1
fi

echo "makefile hash check ok ($ACTUAL)"
