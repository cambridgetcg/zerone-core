#!/usr/bin/env bash
# Build the binary-only swap image for a legacy signer and check it against its base.
#
# Written by an agent (Claude Code, delegate of session 8d4c201e) on 2026-11-06.
# This script builds and verifies locally. It never pushes, never calls fly and
# never reads a key. Pulling the real base needs the operator's registry login
# (`fly auth docker`); that pull and the later push are the operator's acts.
#
# It builds the image twice from scratch (--no-cache) into OCI archives with
# SOURCE_DATE_EPOCH and rewrite-timestamp=true, requires the two manifest
# digests to be equal, loads the first archive under the target tag, and runs
# verify-swap-image.py against the base. The printed manifest digest is the
# one `docker push` must report and `fly machine update --image` must use.
#
# Usage:
#   build-swap-image.sh --app zerone-1|zerone-testnet-1 --binary /path/to/zeroned \
#     --docker-host unix:///path/to/docker.sock --out NEW_DIR
#   build-swap-image.sh --local-test --base localhost/name:tag --seed-dir /mainnet-seed \
#     --tag localhost/name:new --binary ... --docker-host ... --out NEW_DIR
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_SHA256=d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443
BINARY_BYTES=100651816
SOURCE_DATE_EPOCH=1784959094   # committer time of 2e37c4c86c31e67515d6aa07b6fd406083e012ea
APP="" BINARY="" HOST="" OUT="" BASE="" SEED="" TAG="" LOCAL=0

while [ $# -gt 0 ]; do
  case "$1" in
    --app) APP="$2"; shift 2 ;;
    --binary) BINARY="$2"; shift 2 ;;
    --docker-host) HOST="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    --local-test) LOCAL=1; shift ;;
    --base) BASE="$2"; shift 2 ;;
    --seed-dir) SEED="$2"; shift 2 ;;
    --tag) TAG="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

case "$APP" in
  zerone-1)
    BASE=registry.fly.io/zerone-1@sha256:ba73fea59838503c9a099887a4604945b4e959a273b2015fd7eaf1baaf3f861a
    SEED=/mainnet-seed
    TAG=registry.fly.io/zerone-1:cometbft-0.38.25-d6391ea7 ;;
  zerone-testnet-1)
    BASE=registry.fly.io/zerone-testnet-1@sha256:8c5a859572cea6eda34de062cce81ca82d46d36f7afaf64fabd1cb668bac311d
    SEED=/testnet-seed
    TAG=registry.fly.io/zerone-testnet-1:cometbft-0.38.25-d6391ea7 ;;
  "")
    [ "$LOCAL" = 1 ] || { echo "--app is required (or --local-test with --base/--seed-dir/--tag)" >&2; exit 2; }
    [ -n "$BASE" ] && [ -n "$SEED" ] && [ -n "$TAG" ] || { echo "--local-test needs --base, --seed-dir and --tag" >&2; exit 2; }
    case "$TAG" in localhost/*) ;; *) echo "--local-test tags must start with localhost/" >&2; exit 2 ;; esac ;;
  *) echo "unknown app: $APP" >&2; exit 2 ;;
esac

[[ "$HOST" =~ ^unix:///[A-Za-z0-9._/-]+$ ]] || { echo "--docker-host must be a literal unix:/// path" >&2; exit 2; }
[ -n "$OUT" ] && [ ! -e "$OUT" ] || { echo "--out must be a new directory" >&2; exit 2; }
[ -f "$BINARY" ] && [ ! -L "$BINARY" ] || { echo "--binary must be a regular file" >&2; exit 2; }

actual=$(shasum -a 256 "$BINARY" | cut -d' ' -f1)
bytes=$(wc -c < "$BINARY" | tr -d ' ')
[ "$actual" = "$BINARY_SHA256" ] && [ "$bytes" = "$BINARY_BYTES" ] || {
  echo "binary differs from the pinned build: sha256 $actual, $bytes bytes" >&2; exit 1; }

mkdir -p "$OUT/context"
install -m 0755 "$BINARY" "$OUT/context/zeroned"
install -m 0644 "$HERE/Dockerfile.swap" "$OUT/context/Dockerfile"

if [ "$LOCAL" = 0 ]; then
  # The base is the signer's current image, pinned by manifest digest.
  docker --host "$HOST" pull --platform linux/amd64 "$BASE"
fi

digest_of() { tar -xOf "$1" index.json | python3 -I -c 'import sys,json; m=json.load(sys.stdin)["manifests"]; assert len(m)==1, m; print(m[0]["digest"])'; }
for n in 1 2; do
  docker --host "$HOST" buildx build --no-cache --platform linux/amd64 --provenance=false --sbom=false \
    --build-arg "BASE=$BASE" --build-arg "SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH" \
    --output "type=oci,dest=$OUT/image-$n.oci.tar,name=$TAG,rewrite-timestamp=true" \
    "$OUT/context" > "$OUT/build-$n.log" 2>&1 || { tail -30 "$OUT/build-$n.log" >&2; exit 1; }
done
D1=$(digest_of "$OUT/image-1.oci.tar"); D2=$(digest_of "$OUT/image-2.oci.tar")
[ "$D1" = "$D2" ] || { echo "the two builds differ: $D1 vs $D2" >&2; exit 1; }
echo "$D1" > "$OUT/manifest-digest.txt"

docker --host "$HOST" load -i "$OUT/image-1.oci.tar" > "$OUT/load.log"
LOADED=$(docker --host "$HOST" image inspect "$TAG" --format '{{.Id}}')
[ "$LOADED" = "$D1" ] || { echo "loaded image id $LOADED is not the built manifest $D1" >&2; exit 1; }

python3 -I "$HERE/verify-swap-image.py" --docker-host "$HOST" --base "$BASE" --new "$TAG" \
  --expected-binary-sha256 "$BINARY_SHA256" --seed-dir "$SEED" --receipt "$OUT/verify.json" | tee "$OUT/verify.log"

shasum -a 256 "$OUT"/image-*.oci.tar > "$OUT/oci-archives.sha256"
echo "built $TAG, manifest $D1 (two no-cache builds agree); verify receipt $OUT/verify.json"
echo "Not pushed. Pushing is a live act for the operator (see the runbook)."
