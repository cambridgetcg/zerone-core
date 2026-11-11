#!/usr/bin/env bash
# spot_check tests: the real function and helpers from observer-start.sh, with http_get stubbed.
# At the tip the validator's header H+1 appears only after H is read, so a captured spot must wait
# for it rather than report "upstream unavailable" at once. No container, no network.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
src="$here/observer-start.sh"
fns() { for f in "$@"; do awk -v f="$f" '$0 ~ "^"f"\\(\\) \\{" {p=1} p {print} p && /^}/ {exit}' "$src"; done; }
eval "$(fns ts jstr say jget b64hex spot_check)"
grep -q '^SPOT_WAIT=' "$src" || { echo "FAIL SPOT_WAIT is not defined in observer-start.sh"; exit 1; }
OBS=/nonexistent; UPSTREAM="validator:26657"; LOCAL_RPC_HOST=127.0.0.1; LOCAL_RPC_PORT=26657; SPOT_WAIT=300
HASH_B64="q83vAQ=="; HASH_HEX="ABCDEF01"   # base64 of bytes ab cd ef 01
pass=0; fail=0
check() { if printf '%s' "$2" | tr '\n' ' ' | grep -qE "$3"; then pass=$((pass + 1)); else fail=$((fail + 1)); echo "FAIL $1: $2"; fi; }

# The stub: the local node is at 100; the validator's header 101 exists once UP_READY is 1.
http_get() {
  case "$3" in
    /abci_info) printf '{"response":{"last_block_height":"100","last_block_app_hash":"%s"}}' "$HASH_B64" ;;
    "/header?height=101") [ "$UP_READY" = 1 ] && printf '{"header":{"height":"101","app_hash":"%s"}}' "$UP_HASH" ;;
    *) return 1 ;;
  esac
}

# 1. Header out at once: MATCH on the first call (the catch-up case).
SPOT_H=""; UP_READY=1; UP_HASH="$HASH_HEX"
check match-now "$(spot_check)" '"result":"MATCH".*"validator_header":101'

# 2. Header not out yet: the first call says nothing and keeps H; a later call matches it.
SPOT_H=""; UP_READY=0
out="$(spot_check; echo "kept=$SPOT_H")"; spot_check >/dev/null
check wait-silent "$out" '^kept=100 ?$'
UP_READY=1; check wait-then-match "$(spot_check; echo "kept=${SPOT_H:-none}")" '"result":"MATCH".*kept=none'

# 3. A different hash is a DIFFER, never a MATCH.
SPOT_H=""; UP_READY=1; UP_HASH="DEADBEEF"
check differ "$(spot_check)" '"result":"DIFFER".*"validator_header_app_hash":"DEADBEEF"'

# 4. Header never comes: "upstream unavailable" only after SPOT_WAIT, then H is released.
SPOT_H=""; UP_READY=0; spot_check >/dev/null; SPOT_T=$(( $(date +%s) - 301 ))
check gives-up "$(spot_check; echo "kept=${SPOT_H:-none}")" '"result":"upstream unavailable".*"waited_s":30[0-9].*kept=none'

# 5. UPSTREAM=off disables it.
SPOT_H=""; UPSTREAM=off; UP_READY=1
check off "$(spot_check; echo "kept=${SPOT_H:-none}")" '^kept=none ?$'

echo "spot: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
