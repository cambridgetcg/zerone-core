#!/usr/bin/env bash
# Offline smoke test with the REAL lane-recipe binary (the signed 2026-09-09 build, d6391ea7...):
# a one-rung ladder from genesis, no network at all (--network none), peer host 127.0.0.1.
# Proves on linux/amd64 (emulated on colima) that: init + every config edit + genesis validate +
# identity checks pass with the real binary; the node starts, runs InitChain on zerone-1's genesis
# and answers RPC on loopback; SIGTERM stops it cleanly; the signing state stays untouched.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"   # the observer directory
# SMOKE_BINARY: the lane binary (linux/amd64), e.g. zeroned from release zerone-1-observer-20260909.2
bin_src="${SMOKE_BINARY:?set SMOKE_BINARY to the lane zeroned}"
genesis="$here/genesis.json"; [ -f "$genesis" ] || genesis="$here/../mainnet/artifacts/genesis.json"
log="$here/test/.work/smoke-real.log"; mkdir -p "$here/test/.work"
ctx="$(mktemp -d "${TMPDIR:-/tmp}/obs-smoke.XXXXXX")"
tag="zerone-1-replay-smoke:local"; vol="zerone-1-replay-smoke-data"; name="zerone-1-replay-smoke"
D=(docker --context colima)
cleanup() { "${D[@]}" rm -f "$name" >/dev/null 2>&1 || true; "${D[@]}" volume rm "$vol" >/dev/null 2>&1 || true; "${D[@]}" image rm "$tag" >/dev/null 2>&1 || true; rm -rf "$ctx"; }
trap cleanup EXIT
sha="$(shasum -a 256 "$bin_src" | cut -d' ' -f1)"
mkdir -p "$ctx/bin"
cp "$here/Dockerfile" "$here/.dockerignore" "$here/observer-start.sh" "$ctx/"; cp "$genesis" "$ctx/genesis.json"
cp "$bin_src" "$ctx/bin/$sha"
printf '# smoke ladder\nR00\t1\t-\ttip\t%s\tsmoke\n' "$sha" > "$ctx/ladder.tsv"
chmod 0444 "$ctx/ladder.tsv" "$ctx/genesis.json"; chmod 0555 "$ctx/observer-start.sh" "$ctx/bin/$sha"
"${D[@]}" build -q --platform linux/amd64 --provenance=false -t "$tag" "$ctx" > /dev/null
"${D[@]}" volume create "$vol" > /dev/null
"${D[@]}" run -d --name "$name" --platform linux/amd64 --network none -v "$vol:/data" \
  -e OBSERVER_PEER_HOST=127.0.0.1 -e OBSERVER_UPSTREAM=off -e OBSERVER_TICK=5 -e OBSERVER_REPORT_EVERY=20 \
  -e OBSERVER_EXPECT_FINAL_SHA="$sha" "$tag" > /dev/null
for _ in $(seq 1 60); do
  sleep 5
  if "${D[@]}" logs "$name" 2>&1 | grep -q '"event":"progress".*"height"'; then break; fi
  if ! "${D[@]}" ps -q --filter "name=$name" | grep -q .; then break; fi
done
sleep 25
"${D[@]}" stop -t 90 "$name" > /dev/null || true
"${D[@]}" logs "$name" > "$log" 2>&1 || true
echo "exit code: $("${D[@]}" inspect -f '{{.State.ExitCode}}' "$name")"
grep '^OBSERVER ' "$log" | cut -c1-400
echo "--- node lines (first and last 8, JSON log):"
grep -v '^OBSERVER ' "$log" | head -n 8 | cut -c1-300
grep -v '^OBSERVER ' "$log" | tail -n 8 | cut -c1-300
echo "--- volume after stop (names only; signing state shown, it holds no key):"
"${D[@]}" run --rm --platform linux/amd64 --network none -v "$vol:/data:ro" --entrypoint /bin/sh "$tag" -c \
  'ls -ln /data /data/observer /data/zeroned/config /data/zeroned/data | cut -c1-120; echo; cat /data/zeroned/data/priv_validator_state.json; echo; cat /data/observer/state'
