#!/usr/bin/env bash
# Build the observer image for linux/amd64, locally. Checks every input, stages a context holding
# only the image's files, and builds. Pushes nothing.
#   BIN_DIR      binaries named by sha256 (default: ./bin beside this script)
#   LANE_SHA     if set, the last ladder row must be this sha256
#   DOCKER_CONTEXT_NAME  docker context (default: colima)
set -euo pipefail
src="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${BIN_DIR:-$src/bin}"
D=(docker --context "${DOCKER_CONTEXT_NAME:-colima}")
die() { echo "REFUSED: $*" >&2; exit 2; }
sha() { shasum -a 256 "$1" | cut -d' ' -f1; }
genesis="$src/genesis.json"; [ -f "$genesis" ] || genesis="$src/../mainnet/artifacts/genesis.json"
[ "$(sha "$genesis")" = "c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e" ] || die "genesis is not zerone-1's"
want=()
while IFS=$'\t' read -r name _ _ _ s _; do
  case "$name" in ''|'#'*) continue ;; esac
  [[ "$s" =~ ^[0-9a-f]{64}$ ]] || die "ladder row $name has no sha256 ($s): extract that rung first"
  [ -f "$BIN_DIR/$s" ] && [ ! -L "$BIN_DIR/$s" ] || die "missing binary $s for $name"
  [ "$(sha "$BIN_DIR/$s")" = "$s" ] || die "binary $s does not hash to its name"
  [ "$(head -c 4 "$BIN_DIR/$s" | od -An -tx1 | tr -d ' \n')" = "7f454c46" ] || die "binary $s is not an ELF file"
  want+=("$s")
done < "$src/ladder.tsv"
[ "${#want[@]}" -ge 1 ] || die "empty ladder"
if [ -n "${LANE_SHA:-}" ]; then [ "${want[${#want[@]}-1]}" = "$LANE_SHA" ] || die "last rung is not LANE_SHA"; fi
stage="$(mktemp -d "${TMPDIR:-/tmp}/zerone-1-replay-build.XXXXXX")"; trap 'rm -rf "$stage"' EXIT
mkdir "$stage/bin"
cp "$src/Dockerfile" "$src/.dockerignore" "$src/observer-start.sh" "$src/ladder.tsv" "$stage/"
cp "$genesis" "$stage/genesis.json"
for s in "${want[@]}"; do [ -e "$stage/bin/$s" ] || cp "$BIN_DIR/$s" "$stage/bin/$s"; done
chmod 0444 "$stage/ladder.tsv" "$stage/genesis.json"; chmod 0555 "$stage/observer-start.sh" "$stage"/bin/*
if find "$stage" \( -name 'priv_validator_key.json' -o -name 'node_key.json' -o -name '*.mnemonic' -o -name 'keyring-*' \) | grep -q .; then die "key-shaped file in the build context"; fi
ladder_sha="$(sha "$stage/ladder.tsv")"
tag="zerone-1-replay:ladder-${ladder_sha:0:12}"
"${D[@]}" build --platform linux/amd64 --provenance=false --sbom=false \
  --label "ai.zerone.observer.ladder_sha256=$ladder_sha" -t "$tag" "$stage"
"${D[@]}" run --rm --platform linux/amd64 --network none --entrypoint /bin/sh "$tag" -c 'ls -l /opt/observer /opt/observer/bin'
echo "built $tag  (ladder sha256 $ladder_sha)"
echo "image id: $("${D[@]}" image inspect -f '{{.Id}}' "$tag")"
