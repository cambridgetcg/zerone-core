#!/usr/bin/env bash
# Host step: copy ONE historical zeroned out of a zerone-1 release image by fetching only its
# binary layer. The other layers of these images hold the validator's key files (/mainnet-seed):
# never `docker pull`, `fly machine run` or unpack those images whole.
#
# Needs crane (go-containerregistry) and read access to registry.fly.io/zerone-1. Keep that
# credential out of ~/.docker: `DOCKER_CONFIG=$(mktemp -d) fly auth docker`, run this with the same
# DOCKER_CONFIG, then remove that directory. It can pull every layer, the key layers included.
#   usage: extract-rung-binary.sh registry.fly.io/zerone-1:deployment-<ULID>
#   BIN_DIR   where the binary lands, named by its sha256 (default: ./bin beside this script)
#   EVIDENCE  tsv log of ref, layer digest, sha256 (default: ./rung-binaries.tsv beside this script)
set -euo pipefail
src="$(cd "$(dirname "$0")" && pwd)"
CRANE="${CRANE:-crane}"; BIN_DIR="${BIN_DIR:-$src/bin}"; EVIDENCE="${EVIDENCE:-$src/rung-binaries.tsv}"
ref="${1:?image reference}"
case "$ref" in registry.fly.io/zerone-1:deployment-*) ;; *) echo "REFUSED: expected registry.fly.io/zerone-1:deployment-<ULID>" >&2; exit 2 ;; esac
repo="${ref%:*}"
work="$(mktemp -d "${TMPDIR:-/tmp}/zerone-rung.XXXXXX")"; trap 'rm -rf "$work"' EXIT
"$CRANE" manifest --platform linux/amd64 "$ref" > "$work/manifest.json"
"$CRANE" config --platform linux/amd64 "$ref" > "$work/config.json"
read -r digest _ < <(python3 -I "$src/rung_layer.py" pick "$work/config.json" "$work/manifest.json")
[[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "REFUSED: bad layer digest" >&2; exit 2; }
"$CRANE" blob "$repo@$digest" > "$work/layer.tgz"
[ "sha256:$(shasum -a 256 "$work/layer.tgz" | cut -d' ' -f1)" = "$digest" ] || { echo "REFUSED: blob does not match its digest" >&2; exit 2; }
sha="$(python3 -I "$src/rung_layer.py" extract "$work/layer.tgz" "$BIN_DIR")"
printf '%s\t%s\t%s\n' "$ref" "$digest" "$sha" | tee -a "$EVIDENCE"
echo "put $sha in the matching ladder.tsv row"
