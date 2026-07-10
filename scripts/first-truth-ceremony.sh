#!/usr/bin/env bash
# first-truth-ceremony.sh — drive one knowledge claim through the FULL lifecycle:
#   register-account (submitter + 4 verifiers) → submit-claim → 4x commit →
#   4x reveal (unanimous accept) → ACCEPT → survival escrow → (window) → release.
#
# The panel is OPEN self-selection (rounds.go): 4 reveals + >=77% stake share +
# >=3 accept heads are required for a domain claim (effectiveMin = min_verifiers+1).
# Every verifier needs >=100 ZRN SPENDABLE at commit time (hardcoded gate) — fund
# to >=100.5 ZRN because ante fees deduct before the gate is checked.
#
# Env (defaults = localnet drill):
#   BINARY, NODE, CHAIN_ID, HOME_DIR (keyring home), KEYRING
#   FUNDER (key name), SUBMITTER (key name), VERIFIERS (space-sep 4 key names)
#   DOMAIN, CATEGORY, CONTENT, FEE_UZRN (200000 = safe max under critical pacing)
#   STATE_DIR (ed25519 identity keys + ceremony log land here)
#   SKIP_FUND=1 to skip funding, SKIP_REGISTER=1 to skip zerone_auth registration
set -euo pipefail

BINARY="${BINARY:-$HOME/Desktop/zerone/build/zeroned}"
NODE="${NODE:-tcp://localhost:26601}"
CHAIN_ID="${CHAIN_ID:-zerone-localnet}"
HOME_DIR="${HOME_DIR:-$HOME/.zeroned/localnet/coordinator}"
KEYRING="${KEYRING:-test}"
FUNDER="${FUNDER:-faucet}"
SUBMITTER="${SUBMITTER:-drill-submitter}"
VERIFIERS="${VERIFIERS:-drill-v1 drill-v2 drill-v3 drill-v4}"
DOMAIN="${DOMAIN:-computer_science}"
CATEGORY="${CATEGORY:-computational}"
CONTENT="${CONTENT:-The SHA-256 hash of the ASCII string 'zerone' is $(printf 'zerone' | shasum -a 256 | cut -d' ' -f1).}"
FEE_UZRN="${FEE_UZRN:-200000}"
STATE_DIR="${STATE_DIR:-$HOME/.zerone-agent/first-truth/$CHAIN_ID}"
VERIFIER_FUND_UZRN="${VERIFIER_FUND_UZRN:-101000000}"   # 101 ZRN: 100 gate + fees headroom
SUBMITTER_FUND_UZRN="${SUBMITTER_FUND_UZRN:-2000000}"   # 2 ZRN: fee + gas

# Gas (ante enforces a per-msg MINIMUM gas limit; fees = gas x 1uzrn, charged in full)
GAS_SEND="${GAS_SEND:-200000}"
GAS_MULTISEND="${GAS_MULTISEND:-400000}"
GAS_REGISTER="${GAS_REGISTER:-300000}"
GAS_CLAIM="${GAS_CLAIM:-500000}"
GAS_COMMIT="${GAS_COMMIT:-200000}"
GAS_REVEAL="${GAS_REVEAL:-200000}"

TXFLAGS=(--chain-id "$CHAIN_ID" --node "$NODE" --keyring-backend "$KEYRING" --home "$HOME_DIR" --gas-prices 1uzrn -y -o json --broadcast-mode sync)
QFLAGS=(--node "$NODE" -o json)

mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/ceremony-$(date +%Y%m%d-%H%M%S).log"
say()  { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG" >&2; }
fail() { say "FATAL: $*"; exit 1; }

height() { curl -sm 5 "${NODE/tcp:\/\//http://}/status" | jq -r '.result.sync_info.latest_block_height'; }

wait_height() { # wait until chain height >= $1
  local target="$1" h
  while :; do h=$(height); [ "$h" -ge "$target" ] && break; sleep 2; done
}

# send_tx <label> <gas> <args...> — broadcast, wait for inclusion, dump code+gas, echo txhash
send_tx() {
  local label="$1" gas="$2"; shift 2
  local out hash code raw
  out=$("$BINARY" tx "$@" --gas "$gas" "${TXFLAGS[@]}" 2>&1) || fail "$label broadcast failed: $out"
  hash=$(echo "$out" | jq -r '.txhash' 2>/dev/null) || fail "$label: unparseable broadcast output: $out"
  code=$(echo "$out" | jq -r '.code')
  [ "$code" = "0" ] || fail "$label CheckTx rejected (code $code): $(echo "$out" | jq -r '.raw_log')"
  # poll for inclusion
  for _ in $(seq 1 30); do
    raw=$("$BINARY" q tx "$hash" "${QFLAGS[@]}" 2>/dev/null) && break || sleep 2
  done
  [ -n "${raw:-}" ] || fail "$label: tx $hash never included"
  code=$(echo "$raw" | jq -r '.code')
  local gas_used; gas_used=$(echo "$raw" | jq -r '.gas_used')
  if [ "$code" != "0" ]; then
    echo "$raw" | jq '.raw_log' >> "$LOG"
    fail "$label DeliverTx failed (code $code, gas_used $gas_used) tx $hash"
  fi
  say "  $label OK  tx=$hash gas_used=$gas_used"
  echo "$raw" > "$STATE_DIR/tx-$label.json"
  echo "$hash"
}

addr_of() { "$BINARY" keys show "$1" -a --keyring-backend "$KEYRING" --home "$HOME_DIR"; }

ensure_key() {
  "$BINARY" keys show "$1" -a --keyring-backend "$KEYRING" --home "$HOME_DIR" >/dev/null 2>&1 \
    || "$BINARY" keys add "$1" --keyring-backend "$KEYRING" --home "$HOME_DIR" >/dev/null
}

ed25519_identity() { # $1=actor name → prints pub hex; persists priv hex
  local f="$STATE_DIR/$1.ed25519"
  if [ ! -f "$f" ]; then
    python3 - "$f" <<'PY'
import sys
from cryptography.hazmat.primitives.asymmetric import ed25519
from cryptography.hazmat.primitives import serialization
k = ed25519.Ed25519PrivateKey.generate()
priv = k.private_bytes(serialization.Encoding.Raw, serialization.PrivateFormat.Raw, serialization.NoEncryption()).hex()
pub  = k.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw).hex()
open(sys.argv[1], "w").write(f"{priv}\n{pub}\n")
PY
    chmod 600 "$f"
  fi
  sed -n 2p "$f"
}

is_registered() { # $1=address
  local rest="${NODE/tcp:\/\//http://}"; rest="${rest/26657/1317}"; rest="${rest/26601/26611}" # localnet REST heuristic unused; use CLI
  "$BINARY" q zerone_auth account "$1" "${QFLAGS[@]}" >/dev/null 2>&1
}

say "═══ first-truth ceremony ═══"
say "chain=$CHAIN_ID node=$NODE domain=$DOMAIN category=$CATEGORY fee=${FEE_UZRN}uzrn"
say "content: $CONTENT"
[ ${#CONTENT} -ge 20 ] && [ ${#CONTENT} -le 1000 ] || fail "content length ${#CONTENT} outside 20..1000"

# ── Phase 0: params + actors ────────────────────────────────────────────
PARAMS=$("$BINARY" q knowledge params "${QFLAGS[@]}" | jq .params)
COMMIT_BLOCKS=$(echo "$PARAMS" | jq -r .commit_phase_blocks)
REVEAL_BLOCKS=$(echo "$PARAMS" | jq -r .reveal_phase_blocks)
AGG_BLOCKS=$(echo "$PARAMS" | jq -r .aggregation_phase_blocks)
WINDOW=$(echo "$PARAMS" | jq -r .challenge_duration_blocks)
say "phases: commit=$COMMIT_BLOCKS reveal=$REVEAL_BLOCKS agg=$AGG_BLOCKS survival_window=$WINDOW"

for k in $SUBMITTER $VERIFIERS; do ensure_key "$k"; done
SUB_ADDR=$(addr_of "$SUBMITTER")
say "submitter: $SUBMITTER=$SUB_ADDR"
for v in $VERIFIERS; do say "verifier:  $v=$(addr_of "$v")"; done

# ── Phase 1: funding ────────────────────────────────────────────────────
if [ "${SKIP_FUND:-0}" != "1" ]; then
  say "── funding from $FUNDER"
  VADDRS=""; for v in $VERIFIERS; do VADDRS="$VADDRS $(addr_of "$v")"; done
  send_tx fund-verifiers "$GAS_MULTISEND" bank multi-send "$FUNDER" $VADDRS "${VERIFIER_FUND_UZRN}uzrn" --from "$FUNDER" >/dev/null
  send_tx fund-submitter "$GAS_SEND" bank send "$FUNDER" "$SUB_ADDR" "${SUBMITTER_FUND_UZRN}uzrn" --from "$FUNDER" >/dev/null
fi
for v in $VERIFIERS; do
  bal=$("$BINARY" q bank balance "$(addr_of "$v")" uzrn "${QFLAGS[@]}" | jq -r .balance.amount)
  say "  $v balance: ${bal}uzrn"
  [ "$bal" -ge 100000000 ] || fail "$v below the 100 ZRN verifier gate"
done

# ── Phase 2: zerone_auth registration (ed25519 identity per actor) ─────
if [ "${SKIP_REGISTER:-0}" != "1" ]; then
  say "── registering accounts (did:zrn ed25519 identities)"
  for k in $SUBMITTER $VERIFIERS; do
    a=$(addr_of "$k")
    if is_registered "$a"; then say "  $k already registered"; continue; fi
    PUB=$(ed25519_identity "$k")
    send_tx "register-$k" "$GAS_REGISTER" zerone_auth register-account "did:zrn:$PUB" "$PUB" agent --from "$k" >/dev/null
  done
fi

# ── Phase 3: submit the claim ───────────────────────────────────────────
say "── submitting claim"
H0=$(height)
TXH=$(send_tx submit-claim "$GAS_CLAIM" knowledge submit-claim "$CONTENT" "$DOMAIN" "$CATEGORY" "$FEE_UZRN" --claim-type assertion --from "$SUBMITTER")
TXJ="$STATE_DIR/tx-submit-claim.json"
CLAIM_ID=$(jq -r '[.events[] | select(.type|test("submit_claim")) | .attributes[] | select(.key|test("claim_id"))][0].value // empty' "$TXJ")
ROUND_ID=$(jq -r '[.events[] | select(.type|test("verification_round|round_created")) | .attributes[] | select(.key|test("round_id"))][0].value // empty' "$TXJ")
if [ -z "$CLAIM_ID" ] || [ -z "$ROUND_ID" ]; then
  say "event-parse fallback; dumping event types to log"
  jq -r '.events[].type' "$TXJ" | sort -u >> "$LOG"
  CLAIM_ID=${CLAIM_ID:-$(jq -r '[.events[].attributes[] | select(.key=="claim_id")][0].value // empty' "$TXJ")}
  ROUND_ID=${ROUND_ID:-$(jq -r '[.events[].attributes[] | select(.key=="round_id")][0].value // empty' "$TXJ")}
fi
[ -n "$CLAIM_ID" ] && [ -n "$ROUND_ID" ] || fail "could not extract claim_id/round_id — see $LOG and $TXJ"
SUBMIT_H=$(jq -r '.height' "$TXJ")
say "claim_id=$CLAIM_ID round_id=$ROUND_ID submit_height=$SUBMIT_H (H0=$H0)"

# ── Phase 4: commits (all 4, immediately — COMMIT phase is open now) ───
say "── commit phase (deadline: height $((SUBMIT_H + COMMIT_BLOCKS)))"
for v in $VERIFIERS; do
  SALT=$(openssl rand -hex 16)
  echo "$SALT" > "$STATE_DIR/salt-$v"
  HASH=$(printf 'ZRN.commit.v1:%s:accept:0:%s' "$ROUND_ID" "$SALT" | shasum -a 256 | cut -d' ' -f1)
  send_tx "commit-$v" "$GAS_COMMIT" knowledge submit-commitment "$ROUND_ID" "$HASH" --from "$v" >/dev/null
done

# ── Phase 5: reveals (after commit deadline) ────────────────────────────
say "── waiting for REVEAL phase (height > $((SUBMIT_H + COMMIT_BLOCKS)))"
wait_height $((SUBMIT_H + COMMIT_BLOCKS + 1))
for v in $VERIFIERS; do
  SALT=$(cat "$STATE_DIR/salt-$v")
  send_tx "reveal-$v" "$GAS_REVEAL" knowledge submit-reveal "$ROUND_ID" accept "$SALT" --from "$v" >/dev/null
done

# ── Phase 6: aggregation → ACCEPT ───────────────────────────────────────
say "── waiting for aggregation (height > $((SUBMIT_H + COMMIT_BLOCKS + REVEAL_BLOCKS)))"
wait_height $((SUBMIT_H + COMMIT_BLOCKS + REVEAL_BLOCKS + 2))
CLAIM_JSON=$("$BINARY" q knowledge claim "$CLAIM_ID" "${QFLAGS[@]}" 2>/dev/null) || fail "claim query failed post-aggregation"
echo "$CLAIM_JSON" > "$STATE_DIR/claim-final.json"
STATUS=$(echo "$CLAIM_JSON" | jq -r '.. | .status? // empty' | head -1)
say "claim status: $STATUS"
echo "$CLAIM_JSON" | jq . >> "$LOG"
FACT_ID=$(echo "$CLAIM_JSON" | jq -r '.. | .fact_id? // .factId? // empty' | grep -v '^$' | head -1 || true)
if [ -z "$FACT_ID" ]; then
  # try round query / facts listing
  FACT_ID=$("$BINARY" q knowledge facts "${QFLAGS[@]}" 2>/dev/null | jq -r --arg c "$CLAIM_ID" '[.facts[]? | select((.claim_id? // .claimId? // "")==$c)][0] | (.id // .fact_id // .factId) // empty' || true)
fi
say "fact_id: ${FACT_ID:-NOT-FOUND}"
case "$STATUS" in
  *ACCEPT*|*accepted*|6) say "VERDICT: ACCEPTED ✓";;   # CLAIM_STATUS_ACCEPTED = 6 (types.proto:31)
  *) fail "claim did not reach ACCEPTED (status=$STATUS) — see $STATE_DIR/claim-final.json";;
esac

# ── Phase 7: survival window → release ──────────────────────────────────
ACCEPT_H=$(height)
if [ "${NO_WAIT_SURVIVAL:-0}" = "1" ]; then
  say "── NO_WAIT_SURVIVAL: escrow pending; release lands ~$WINDOW blocks after accept (height ~$((ACCEPT_H + WINDOW))). Verify later with: $BINARY q vesting_rewards schedules-by-recipient $SUB_ADDR --node $NODE"
  [ -n "$FACT_ID" ] && "$BINARY" q knowledge fact "$FACT_ID" "${QFLAGS[@]}" > "$STATE_DIR/fact-at-accept.json" 2>/dev/null || true
  say "═══ ceremony reached ACCEPT + escrow — artifacts in $STATE_DIR ═══"
  exit 0
fi
say "── survival window: waiting $WINDOW blocks (until ~height $((ACCEPT_H + WINDOW + 2)))"
wait_height $((ACCEPT_H + WINDOW + 2))
if [ -n "$FACT_ID" ]; then
  "$BINARY" q knowledge fact "$FACT_ID" "${QFLAGS[@]}" > "$STATE_DIR/fact-final.json" 2>/dev/null || say "WARN: fact query failed"
fi
SCHED=$("$BINARY" q vesting_rewards schedules-by-recipient "$SUB_ADDR" "${QFLAGS[@]}" 2>/dev/null || echo '{}')
echo "$SCHED" > "$STATE_DIR/vesting-schedules.json"
NSCHED=$(echo "$SCHED" | jq -r '[.schedules[]?] | length' 2>/dev/null || echo 0)
say "vesting schedules for submitter: $NSCHED"
[ "${NSCHED:-0}" -ge 1 ] || say "WARN: no vesting schedule found — check query name/shape (release may still have happened; see events)"

# ── Phase 8: calibration score (REST-only: /zerone/knowledge/v1/agent/leaderboard) ──
REST="${REST_URL:-http://127.0.0.1:1317}"
CALIB=$(curl -sm 5 "$REST/zerone/knowledge/v1/agent/leaderboard" 2>/dev/null || echo '')
if [ -n "$CALIB" ] && [ "$CALIB" != "" ]; then
  say "leaderboard: $(echo "$CALIB" | jq -c . 2>/dev/null | head -c 800)"
  echo "$CALIB" > "$STATE_DIR/calibration.json"
else say "WARN: leaderboard REST query failed ($REST)"; fi

say "═══ ceremony complete — artifacts in $STATE_DIR ═══"
