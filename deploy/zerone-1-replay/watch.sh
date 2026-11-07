#!/usr/bin/env bash
# Read-only watch for the observer run. Changes nothing: reads Fly logs and machine state, and GETs
# the validator's public RPC. Prints the latest OBSERVER verdict lines, the replay rate and ETA, the
# validator's block cadence, and the running cost so far. Exit 3 means "stop the machine" advice.
# usage: tools/watch.sh            (APP defaults to zerone-1-replay)
set -euo pipefail
APP="${APP:-zerone-1-replay}"
RPC="${VALIDATOR_RPC:-http://zerone-1.fly.dev:26657}"
HOURLY_USD="${HOURLY_USD:-0.0952}"      # performance-2x in ams (iad $0.0917/h x 1.038); shared-cpu-2x 4GB ~ 0.0365
advice=0
logs="$(fly logs -a "$APP" --no-tail 2>/dev/null | grep -o 'OBSERVER {.*}' || true)"
echo "== observer verdicts (latest of each)"
for ev in APPHASH_MISMATCH LADDER_WRONG UPGRADE_NEEDED stopped rung_advance rung_start apphash_spot progress; do
  line="$(printf '%s\n' "$logs" | grep "\"event\":\"$ev\"" | tail -n1 || true)"
  [ -n "$line" ] && echo "$line" | cut -c1-400
done
if printf '%s\n' "$logs" | grep -qE '"event":"(APPHASH_MISMATCH|LADDER_WRONG)"'; then echo "!! proof failed: machine has stopped itself; read the evidence line"; advice=3; fi
if printf '%s\n' "$logs" | grep -q '"result":"DIFFER"'; then echo "!! spot check DIFFER seen"; advice=3; fi
echo "== validator cadence (last 20 headers, public GET)"
st="$(curl -fsS --max-time 15 "$RPC/status")"
tip="$(printf '%s' "$st" | python3 -I -c 'import json,sys; print(json.load(sys.stdin)["result"]["sync_info"]["latest_block_height"])')"
curl -fsS --max-time 15 "$RPC/blockchain?minHeight=$((tip - 19))&maxHeight=$tip" | python3 -I -c '
import json, sys, datetime as dt
m = json.load(sys.stdin)["result"]["block_metas"]
t = sorted(dt.datetime.fromisoformat(x["header"]["time"][:26].rstrip("Z")) for x in m)
gap = (t[-1] - t[0]).total_seconds() / (len(t) - 1)
age = (dt.datetime.utcnow() - t[-1]).total_seconds()
print("tip %s  mean interval %.1f s  last block %.0f s ago" % (m[0]["header"]["height"], gap, age))
sys.exit(3 if gap > 45 or age > 120 else 0)' || { echo "!! validator slower than 45 s per block or stalled: stop the observer machine first"; advice=3; }
p="$(printf '%s\n' "$logs" | grep '"event":"progress"' | tail -n1 || true)"
if [ -n "$p" ]; then
  printf '%s' "$p" | python3 -I -c '
import json, sys
d = json.loads(sys.stdin.read().split(" ", 1)[1]); tip = int(sys.argv[1])
h = int(d.get("height") or 0); r = int(d.get("blocks_per_s") or 0)
eta = (tip - h) / r / 3600 if r > 0 else float("nan")
print("observer %s at %d of %d (%.1f%%), %d blocks/s, ETA %.1f h" % (d.get("rung"), h, tip, 100.0 * h / tip, r, eta))' "$tip"
fi
echo "== machine and cost"
fly machines list -a "$APP" --json 2>/dev/null | python3 -I -c '
import json, sys, datetime as dt
rate = float(sys.argv[1])
for m in json.load(sys.stdin):
    g = m.get("config", {}).get("guest", {})
    made = dt.datetime.fromisoformat(m["created_at"].replace("Z", "+00:00"))
    hours = (dt.datetime.now(dt.timezone.utc) - made).total_seconds() / 3600
    print("%s %s %s %s/%dx %d MB  up to %.1f h since create  <= $%.2f compute at $%.4f/h" % (
        m["id"], m["state"], m["region"], g.get("cpu_kind"), g.get("cpus", 0), g.get("memory_mb", 0), hours, hours * rate, rate))' "$HOURLY_USD" || true
fly ips list -a "$APP" 2>/dev/null | sed -n '2,$p' | grep -q . && { echo "!! the observer app has IP addresses: release them"; advice=3; } || echo "no IPs on $APP (expected)"
exit "$advice"
