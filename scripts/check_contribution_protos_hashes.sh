#!/usr/bin/env bash
#
# check_contribution_protos_hashes.sh — verify each contribution proto
# file matches its pinned hash in .contribution-protos-hashes.
#
# Mirror of check_sub_creed_hashes.sh applied to the contribution proto
# files in proto/zerone/contribution/v1/. The contribution proto files
# define the canonical Contribution envelope — the chain's most
# foundational data contract. Hash-anchoring these protos means any
# change to the contribution schema (adding fields, renaming types,
# changing RPC signatures) is a visible, deliberate act that bumps the
# hash and recomputes the manifest.
#
# UW: ZERONE is recursive — the data contract that defines a Contribution
# is itself part of the substrate the chain produces, and it is
# hash-bound the same way the doctrines are. The recursion is structural,
# not metaphorical.
#
# Format of .contribution-protos-hashes (whitespace-separated triples,
# one per line, blank lines + lines starting with `#` ignored):
#   <basename> <sha256> <relpath>
#
# To intentionally amend a contribution proto:
#   1. Edit the .proto file.
#   2. Run make proto-gen to regenerate .pb.go files.
#   3. Run this script — it will print the new computed hash on mismatch.
#   4. Update .contribution-protos-hashes with the new hash for that file.
#   5. If the recursion manifest exists, recompute it via
#      scripts/check_recursion_manifest.sh.
#   6. Commit all changed files together so reviewers see the proto
#      diff, the generated code, the hash bump, and the manifest bump
#      in the same commit.

set -euo pipefail

HASH_FILE=".contribution-protos-hashes"

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

failed=0
while IFS= read -r line || [ -n "$line" ]; do
  # Skip blank lines and comments.
  case "$line" in
    ''|'#'*) continue ;;
  esac
  # Parse: <basename> <hash> <relpath>. Use awk so the path may contain
  # arbitrary single-spaces (unlikely, but defensive).
  name=$(echo "$line"     | awk '{print $1}')
  expected=$(echo "$line" | awk '{print $2}')
  doc=$(echo "$line"      | awk '{print $3}')
  if [ -z "$name" ] || [ -z "$expected" ] || [ -z "$doc" ]; then
    continue
  fi
  if [ ! -f "$doc" ]; then
    echo "error: $doc referenced in $HASH_FILE but not found" >&2
    failed=1
    continue
  fi
  actual=$(tr -d '\r' < "$doc" | hash_cmd | awk '{print $1}')
  if [ "$actual" != "$expected" ]; then
    cat <<EOF >&2
contribution proto hash check failed: $name

  file:       $doc
  expected:   $expected
  actual:     $actual

If you intentionally amended this proto, update the matching line in
$HASH_FILE to:
  $name $actual $doc

Remember to run make proto-gen after changing .proto files.
EOF
    failed=1
  else
    echo "contribution-proto hash check ok ($name: $actual)"
  fi
done < "$HASH_FILE"

if [ "$failed" -ne 0 ]; then
  exit 1
fi
