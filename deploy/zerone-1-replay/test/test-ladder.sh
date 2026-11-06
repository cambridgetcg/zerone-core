#!/usr/bin/env bash
# Ladder logic tests for image/observer-start.sh with stand-in binaries, in a native debian
# container (the same digest-pinned base as the image), no network. Each scenario runs the real
# start script as root, so the privilege drop to uid 10001 is exercised too.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"   # the observer directory
D=(docker --context colima)
BASE="docker.io/library/debian@sha256:5ae3c39ebd15e229dcedd5cee596b2497182493d41ff162e824ba13fc1b2b867"
# colima shares only the home directory with its VM, so the work tree lives beside this script.
mkdir -p "$here/test/.work"; work="$(mktemp -d "$here/test/.work/ladder.XXXXXX")"; trap 'rm -rf "$work"' EXIT
genesis="$here/genesis.json"; [ -f "$genesis" ] || genesis="$here/../mainnet/artifacts/genesis.json"
pass=0; fail=0
mkmock() {  # mkmock <dir> <role> -> prints sha
  local f; f="$(mktemp "$work/m.XXXXXX")"
  sed "s|^# role: ROLE|# role: $2|" "$here/test/mock/zeroned-mock.sh" > "$f"
  local sha; sha="$(shasum -a 256 "$f" | cut -d' ' -f1)"; mkdir -p "$1/bin"; mv "$f" "$1/bin/$sha"; chmod 755 "$1/bin/$sha"; echo "$sha"
}
scenario() {  # scenario <name> <seconds> <expect-regex> [not-regex] -- rows: "first last end role"...
  local name="$1" secs="$2" expect="$3" notre="$4"; shift 5
  local opt="$work/$name/opt" data="$work/$name/data"; mkdir -p "$opt" "$data"
  cp "$here/observer-start.sh" "$opt/"; chmod 755 "$opt/observer-start.sh"
  cp "$genesis" "$opt/genesis.json"
  : > "$opt/ladder.tsv"; local n=0 row first last end role sha
  for row in "$@"; do
    read -r first last end role <<< "$row"; sha="$(mkmock "$opt" "$role")"
    printf 'R%02d\t%s\t%s\t%s\t%s\ttest\n' "$n" "$first" "$last" "$end" "$sha" >> "$opt/ladder.tsv"; n=$((n + 1))
  done
  chmod -R a+rX "$opt"
  local out="$work/$name.log"
  timeout "$((secs + 30))" "${D[@]}" run --rm --network none -v "$opt:/opt/observer:ro" -v "$here/test/fixtures:/fixtures:ro" \
    -e OBSERVER_REQUIRE_MOUNT=0 -e OBSERVER_DATA=/data -e OBSERVER_PEER_HOST=localhost -e OBSERVER_UPSTREAM=off \
    -e OBSERVER_TICK=1 -e OBSERVER_REPORT_EVERY=3 -e OBSERVER_PAUSE_UNIT=1 -e OBSERVER_STOP_GRACE=5 \
    --tmpfs /data:rw,mode=0755 --stop-timeout 20 "$BASE" bash -c "timeout --preserve-status -s TERM $secs /opt/observer/observer-start.sh; echo EXIT=\$?; ls /data/observer | tr '\n' ' '" > "$out" 2>&1 || true
  if grep -qE "$expect" "$out" && { [ -z "$notre" ] || ! grep -qE "$notre" "$out"; }; then
    pass=$((pass + 1)); echo "PASS $name"
  else
    fail=$((fail + 1)); echo "FAIL $name"; grep '^OBSERVER\|EXIT=\|HALTED\|state' "$out" | cut -c1-300 | tail -n 25
  fi
  cp "$out" "$here/test/.work/last-$name.log"
}
# 1. full ladder: upgrade -> halt -> upgrade -> tip, then SIGTERM at the tip
scenario happy 40 'rung_advance.*"to":"R03".*|"to":"R03"' 'HALTED|APPHASH|LADDER_WRONG' -- \
  "1 99 upgrade:plan-a upgrade:plan-a:100" "100 149 halt halt" "150 199 upgrade:plan-b upgrade:plan-b:200" "200 - tip tip"
check() { if eval "$2"; then pass=$((pass + 1)); echo "PASS $1"; else fail=$((fail + 1)); echo "FAIL $1"; fi; }
check happy-clean-stop "grep -q '\"event\":\"stopped\".*\"reason\":\"signal\"' '$work/happy.log' && grep -q 'EXIT=0' '$work/happy.log'"
check no-private-key-in-logs "grep -q consensus_address '$work/happy.log' && ! grep -q TESTONLY '$work/happy.log'"
check happy-no-script-error "! grep -q script_error '$work/happy.log'"
# 2. app hash mismatch stops the machine and marks the volume
scenario mismatch 25 'APPHASH_MISMATCH.*wrong Block.Header.AppHash' '' -- "1 - tip mismatch"
check mismatch-marker "grep -q HALTED-APPHASH '$work/mismatch.log'"
# 3. a binary that has a pending plan's handler too early
scenario early 25 'LADDER_WRONG.*BINARY UPDATED BEFORE TRIGGER' '' -- "1 99 upgrade:plan-a early" "100 - tip tip"
# 4. a rung that stops for a different plan than the ladder says
scenario wrongplan 25 'LADDER_WRONG.*plan-a 100' '' -- "1 99 upgrade:plan-a upgrade:plan-z:100" "100 - tip tip"
# 5. a crash loop stops the machine after 8 fast exits
scenario crashloop 260 'HALTED|CRASHLOOP' '' -- "1 - tip crash"
# 6. the live validator's key in the home is refused before any start
scenario badkey 20 'stopped.*(LIVE validator|IDENTITY)' 'rung_start' -- "1 - tip badkey"
# 7. a ladder that does not start at 1 is refused
scenario gap 15 'stopped.*first rung must start at height 1' 'rung_start' -- "5 - tip tip"
# 8. following at the tip: an upgrade the ladder has no binary for is announced and the node stays up
scenario tipupgrade 25 'UPGRADE_NEEDED.*plan-x' 'HALTED|rung_advance' -- "1 - tip consensus-upgrade:plan-x:500"
# 9. consensus-mode upgrade halt for a planned rung: stop it and advance
scenario consensusplanned 30 '"to":"R01"' 'HALTED' -- "1 499 upgrade:plan-x consensus-upgrade:plan-x:500" "500 - tip tip"
echo "passed $pass, failed $fail"
[ "$fail" -eq 0 ]
