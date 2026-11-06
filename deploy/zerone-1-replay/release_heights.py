#!/usr/bin/env python3
"""Map image-change times to zerone-1 heights from public block times (GET reads only).

stdin: lines "<label> <UTC time>", e.g. "v5 2026-07-10T06:52:08Z" (from `fly releases --json`).
stdout: one JSON line per label with the last block at or before that time and the largest gaps
in block time nearby. After an x/upgrade halt at plan height H the gap appears at H+2; after a
plain restart whose first height after the gap is G, block G-1 was executed by the new binary
(G-2 by either). Usage: python3 -I release_heights.py < releases.txt   (RPC env overrides the URL)
"""
import datetime as dt
import json
import os
import sys
import time
import urllib.request

RPC = os.environ.get("RPC", "http://zerone-1.fly.dev:26657")
def get(path):
    time.sleep(0.15)
    with urllib.request.urlopen(RPC + path, timeout=20) as r:
        return json.load(r)["result"]
def ts(s):
    s = s.replace("Z", "")
    if "." in s:
        a, b = s.split(".")
        s = a + "." + b[:6]
    return dt.datetime.fromisoformat(s).replace(tzinfo=dt.timezone.utc)
cache = {}
def btime(h):
    if h not in cache:
        cache[h] = ts(get(f"/header?height={h}")["header"]["time"])
    return cache[h]
def metas(a, b):
    r = get(f"/blockchain?minHeight={a}&maxHeight={b}")
    out = {}
    for m in r["block_metas"]:
        out[int(m["header"]["height"])] = (ts(m["header"]["time"]), m["num_txs"])
    return out
tip = int(get("/status")["sync_info"]["latest_block_height"])
releases = [l.split() for l in sys.stdin if l.strip()]
for ver, when in releases:
    t = ts(when)
    lo, hi = 1, tip
    if btime(1) > t:
        print(json.dumps({"release": ver, "at": when, "note": "before block 1"})); continue
    while hi - lo > 1:
        mid = (lo + hi) // 2
        if btime(mid) <= t: lo = mid
        else: hi = mid
    # lo = last block at or before release time; look for the largest gap nearby
    a = max(1, lo - 30); b = min(tip, lo + 30)
    m = {}
    for s in range(a, b + 1, 20):
        m.update(metas(s, min(b, s + 19)))
    hs = sorted(m)
    gaps = sorted(((m[h][0] - m[h-1][0]).total_seconds(), h) for h in hs if h-1 in m)[-3:]
    print(json.dumps({"release": ver, "at": when, "last_block_before": lo, "time_lo": str(btime(lo)),
                      "next": hi, "time_next": str(btime(hi)),
                      "largest_gaps_s_first_height_after": [[round(g,1), h] for g, h in reversed(gaps)]}))
    sys.stdout.flush()
