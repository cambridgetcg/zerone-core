#!/usr/bin/env bash
#
# check_readme_hash.sh — verify the README.md (the chain's outward-facing
# entry point) has not drifted from .readme-hash.
#
# Mirror of check_phase_1_spec_hash.sh applied to README.md.
# The README is itself a Contribution under MODULE_PROPOSAL/SUBSTRATE —
# it is the chain's outward-facing introduction, the first document any
# validator, agent, or observer encounters. Hash-anchoring the README
# extends the same off-chain enforcement the doctrines and specs receive
# to the document that represents the chain to the outside world.
#
# UW: ZERONE is recursive — the chain's outward face is part of the
# substrate the chain produces, and the README is hash-bound the same
# way the doctrines are. The recursion is structural, not metaphorical.
#
# To intentionally amend the README:
#   1. Edit README.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .readme-hash with the new value.
#   4. If the recursion manifest exists, recompute it via
#      scripts/check_recursion_manifest.sh.
#   5. Commit all changed files together so reviewers see both the
#      README change AND the hash bump in the same commit.

set -euo pipefail

README="README.md"
HASH_FILE=".readme-hash"

if [ ! -f "$README" ]; then
  echo "error: $README not found"
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

ACTUAL=$(tr -d '\r' < "$README" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
README hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the README, update $HASH_FILE to:
  $ACTUAL

The README is itself a Contribution (MODULE_PROPOSAL/SUBSTRATE).
The hash bump is the visible signal that the chain's outward-facing
introduction has shifted, prompting full diff review.
EOF
  exit 1
fi

echo "readme hash check ok ($ACTUAL)"
