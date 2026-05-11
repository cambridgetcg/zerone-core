#!/usr/bin/env bash
#
# check_recursion_manifest.sh — verify the manifest-of-hashes.
#
# The hash files at the repo root are themselves not hash-protected:
# someone could edit .creed-hash directly and the chain would carry on.
# This script plugs that hole. .recursion-manifest-hash holds the
# sha256 of the deterministic concatenation of all .*-hash and .*-hashes
# files at the repo root (excluding itself), sorted by filename. Drift
# in any single hash file fails the per-file check above; drift in
# .recursion-manifest-hash fails THIS check. The recursion is closed:
# the index of indices is itself an index entry.
#
# This script must run AFTER all the per-doc hash checks because it
# depends on their outputs being canonical. Run it last in
# `make creed-check`.
#
# To intentionally update the manifest (after a deliberate hash bump):
#   1. Update the doc and bump its individual .NAME-hash file.
#   2. Run this script — it will print the new computed manifest hash.
#   3. Update .recursion-manifest-hash with the new value.
#   4. Commit all changed hash files together with the new manifest.
#
# The parallel session may add new hash files between when one author
# updates the manifest and when CI runs. That fails this check by
# design — it forces the parallel session to update the manifest when
# they add hashes. The friction is the feature.

set -euo pipefail

MANIFEST_FILE=".recursion-manifest-hash"

if [ ! -f "$MANIFEST_FILE" ]; then
  echo "error: $MANIFEST_FILE not found" >&2
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

# Build the deterministic input: every .*-hash and .*-hashes file at
# the repo root (excluding the manifest itself), sorted by filename,
# concatenated with newline boundaries between entries.
#
# We use ls + grep so the glob is portable across shells (zsh's null_glob
# behavior differs from bash, and we run this from both make + dev shells).
files=$(ls -1 .*-hash .*-hashes 2>/dev/null | grep -v "^$MANIFEST_FILE$" | sort)

if [ -z "$files" ]; then
  echo "error: no hash files found at repo root" >&2
  exit 1
fi

# Concatenate: filename + content + newline. Including the filename in
# the input means renaming a hash file (an act that changes the manifest's
# semantic content) is detected even if the file's bytes are unchanged.
# Iterate via `while read` (line-oriented) rather than `for $files`
# (whitespace-split) so the input is robust against any future filename
# containing whitespace — defensive even though current filenames don't.
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
while IFS= read -r f; do
  printf '%s\n' "$f" >> "$TMP"
  cat "$f" >> "$TMP"
  printf '\n' >> "$TMP"
done <<< "$files"

ACTUAL=$(hash_cmd < "$TMP" | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$MANIFEST_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
recursion-manifest hash check failed.

Expected (from $MANIFEST_FILE): $EXPECTED
Actual (computed):              $ACTUAL

Files included in the manifest input (sorted):
$(echo "$files" | sed 's/^/  /')

If a hash file changed intentionally (and you bumped it), update
$MANIFEST_FILE to:
  $ACTUAL

If a NEW hash file appeared (e.g., the parallel session added one),
update $MANIFEST_FILE the same way — the manifest is the index of
indices, and is itself an index entry. The friction is the feature.
EOF
  exit 1
fi

echo "recursion-manifest hash check ok ($ACTUAL)"
