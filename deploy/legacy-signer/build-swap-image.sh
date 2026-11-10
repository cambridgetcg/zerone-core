#!/usr/bin/env bash
# Assemble the binary-only swap image for a legacy signer at the registry, and check it.
#
# Written by an agent (Claude Code, session 8d4c201e) on 2026-11-06.
#
# The signer images bake priv_validator_key.json and node_key.json in their own layers. So this
# script never pulls, saves, loads or runs a base image, and no base layer blob is downloaded:
# `crane append` reads the base's manifest and config (JSON), uploads one new layer, and writes a
# manifest that references the base layers by digest. swap-layer.py then rebuilds what that
# manifest and config must be from the base's metadata and the new layer alone, and refuses any
# other difference. crane's request log (method and path only, no query strings or headers) must
# show no GET of a base layer blob.
#
# The push of the new tag is the one write. It changes nothing running: a machine moves to the
# image only through `fly machine update --image` (README, live steps).
#
# Usage:
#   DOCKER_CONFIG=<throwaway dir holding only a registry login> \
#   build-swap-image.sh --app zerone-1|zerone-testnet-1 --base registry.fly.io/<app>@sha256:<digest> \
#     --binary /path/to/zeroned --out NEW_DIR
#   --base is the signer's current image, read just before (`fly machine status <id> -a <app>
#   --display-config`, image_ref): no locator is kept in this repository.
#   build-swap-image.sh --local-test --registry localhost:5000 --base <repo>@sha256:<digest> \
#     --tag <repo>:<tag> --binary /path/to/zeroned --out NEW_DIR     (plain-HTTP test registry)
#   CRANE: crane binary (go-containerregistry v0.20.2; default `crane` on PATH)
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CRANE="${CRANE:-crane}"
APP="" BINARY="" OUT="" BASE="" TAG="" LOCAL=0 REGISTRY=""

while [ $# -gt 0 ]; do
  case "$1" in
    --app) APP="$2"; shift 2 ;;
    --binary) BINARY="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    --local-test) LOCAL=1; shift ;;
    --registry) REGISTRY="$2"; shift 2 ;;
    --base) BASE="$2"; shift 2 ;;
    --tag) TAG="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

INSECURE=()
case "$APP" in
  zerone-1|zerone-testnet-1)
    [[ "$BASE" =~ ^registry\.fly\.io/$APP@sha256:[0-9a-f]{64}$ ]] || {
      echo "--base must be the signer's current image, registry.fly.io/$APP@sha256:<digest>" >&2; exit 2; }
    TAG="registry.fly.io/$APP:cometbft-0.38.25-d6391ea7" ;;
  "")
    [ "$LOCAL" = 1 ] || { echo "--app is required (or --local-test)" >&2; exit 2; }
    [[ "$REGISTRY" =~ ^(localhost|127\.0\.0\.1):[0-9]+$ ]] || { echo "--local-test needs --registry localhost:<port>" >&2; exit 2; }
    case "$BASE" in "$REGISTRY"/*@sha256:*) ;; *) echo "--base must be $REGISTRY/<repo>@sha256:<digest>" >&2; exit 2 ;; esac
    case "$TAG" in "$REGISTRY"/*:*) ;; *) echo "--tag must be $REGISTRY/<repo>:<tag>" >&2; exit 2 ;; esac
    INSECURE=(--insecure) ;;
  *) echo "unknown app: $APP" >&2; exit 2 ;;
esac
if [ "$LOCAL" = 0 ]; then
  [ -n "${DOCKER_CONFIG:-}" ] && [ "$DOCKER_CONFIG" != "$HOME/.docker" ] || {
    echo "set DOCKER_CONFIG to a throwaway directory holding only the registry login (never ~/.docker)" >&2; exit 2; }
fi
[ "${BASE%@sha256:*}" = "${TAG%:*}" ] || { echo "the tag must be in the base's repository" >&2; exit 2; }
[ -n "$OUT" ] && [ ! -e "$OUT" ] || { echo "--out must be a new directory" >&2; exit 2; }
[ -f "$BINARY" ] && [ ! -L "$BINARY" ] || { echo "--binary must be a regular file" >&2; exit 2; }
"$CRANE" version > /dev/null

umask 077
mkdir -p "$OUT"
cr() { "$CRANE" "${INSECURE[@]}" "$@"; }

# 1. The layer, twice: it must be byte-identical.
python3 -I "$HERE/swap-layer.py" layer --binary "$BINARY" --out "$OUT/layer.tar" > "$OUT/layer.sha256"
python3 -I "$HERE/swap-layer.py" layer --binary "$BINARY" --out "$OUT/layer-again.tar" > /dev/null
cmp -s "$OUT/layer.tar" "$OUT/layer-again.tar" || { echo "the layer tar is not reproducible" >&2; exit 1; }
rm "$OUT/layer-again.tar"

# 2. The base's metadata only.
cr manifest "$BASE" > "$OUT/base-manifest.json"
cr config "$BASE" > "$OUT/base-config.json"
[ "sha256:$(shasum -a 256 "$OUT/base-manifest.json" | cut -d' ' -f1)" = "${BASE#*@}" ] || {
  echo "the base manifest does not hash to its digest (an index? pass the linux/amd64 manifest)" >&2; exit 1; }

# 3. Append at the registry. Keep only "METHOD path" from crane's request log.
cr append -v -b "$BASE" -f "$OUT/layer.tar" -t "$TAG" 2> >(sed -n -E 's#^.*--> ([A-Z]+) https?://[^/]+(/[^? ]*).*$#\1 \2#p' > "$OUT/requests.log") > "$OUT/append.out"
sleep 1   # let the log filter finish
NEW="$(cr digest "$TAG")"
[[ "$NEW" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "no digest for $TAG" >&2; exit 1; }
cr manifest "${TAG%:*}@$NEW" > "$OUT/new-manifest.json"
cr config "${TAG%:*}@$NEW" > "$OUT/new-config.json"
ADDED="$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["layers"][-1]["digest"])' "$OUT/new-manifest.json")"
python3 -I -c 'import json,sys; b={l["digest"] for l in json.load(open(sys.argv[1]))["layers"]}; sys.exit(sys.argv[2] in b)' \
  "$OUT/base-manifest.json" "$ADDED" || { echo "the last layer is a base layer; refusing to fetch it" >&2; exit 1; }
cr blob "${TAG%:*}@$ADDED" > "$OUT/layer.blob"

# 4. Check.
python3 -I "$HERE/swap-layer.py" check --base-manifest "$OUT/base-manifest.json" --base-config "$OUT/base-config.json" \
  --new-manifest "$OUT/new-manifest.json" --new-config "$OUT/new-config.json" --layer-tar "$OUT/layer.tar" \
  --layer-blob "$OUT/layer.blob" --requests "$OUT/requests.log" --receipt "$OUT/check.json" | tee "$OUT/check.log"
echo "$NEW" > "$OUT/manifest-digest.txt"
echo "pushed $TAG = $NEW (base layers referenced, none downloaded); receipt $OUT/check.json"
echo "Nothing running changed. The machine update is a separate live step (README)."
