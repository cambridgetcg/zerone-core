#!/bin/bash
# deploy/upgrades/consolidation-safety-v1/drill.sh
#
# Local lineage drill for the staged upgrade ladder:
#
#   OLD (live boundary K5/P1/L3/V1)
#     --consolidation-safety-v1-->  H1  (K6/P2/L5/V1, conjecture engine ignites)
#     --founder-renunciation-v1-->  H2  (V1->2)
#     --sdk-0.53-ibc-10---------->  MAIN (trunk, SDK 0.53)
#
# Every leg exercises the proven ritual: gov proposal -> vote -> MANDATORY halt ->
# binary swap -> resume -> verify. First proven end-to-end 2026-08-03.
#
# Constraints this drill encodes (each one cost a live lesson):
#   * H1's startup lineage wall requires the plan Info to be CANONICAL compact
#     sorted-key JSON (>=1 key, <=4096 bytes). Free-text info bricks the swap.
#   * The old binary halts FROZEN-ALIVE (CONSENSUS FAILURE, RPC answering), the
#     same posture as mainnet — it does not panic-exit. Halt detection is
#     height-frozen-N-polls, then the operator stops it.
#   * Verifier knowledge messages require zerone_auth onboarding; panel keys
#     need >100.5 ZRN spendable at commit; conjecture fee is the EFFECTIVE fee
#     (pacing-scaled), not the params minimum.
#   * Throwaway --home and non-default ports only. Never ~/.zeroned, never
#     scripts/localnet.sh (a long-lived localnet may hold 26600-26631,
#     9090-9093, 1317-1320).
#
# Usage:
#   DRILL_ROOT=/tmp/zerone-h1-drill \
#   OLD_BIN=... H1_BIN=... H2_BIN=... MAIN_BIN=... deploy/upgrades/consolidation-safety-v1/drill.sh all
#
# Build the four binaries first (scratch clone, never the shared tree):
#   OLD  = last pre-H1 live commit        (K5/P1/L3/V1; 2026-08 boundary: 2e37c4c)
#   H1   = 65c19cd8b00bdfff9b80705b776fd0d49719398a  (accepted H1 source)
#   H2   = 36728afbf71905a077a0863b41536fa9279109dd  (H2 corrective candidate)
#   MAIN = current trunk tip
set -euo pipefail

DRILL_ROOT=${DRILL_ROOT:?set DRILL_ROOT to a scratch directory}
OLD_BIN=${OLD_BIN:?} ; H1_BIN=${H1_BIN:?} ; H2_BIN=${H2_BIN:?} ; MAIN_BIN=${MAIN_BIN:?}
H1_COMMIT=${H1_COMMIT:-65c19cd8b00bdfff9b80705b776fd0d49719398a}
H2_COMMIT=${H2_COMMIT:-36728afbf71905a077a0863b41536fa9279109dd}
MAIN_COMMIT=${MAIN_COMMIT:-$(git rev-parse HEAD 2>/dev/null || echo trunk)}

HOME_DIR=$DRILL_ROOT/home
CHAIN_ID=zerone-drill-1
P2P=26900 RPCP=26901 GRPCP=9290 APIP=1417
RPC=http://127.0.0.1:$RPCP
LOG=$DRILL_ROOT/logs
mkdir -p "$LOG"

TXFLAGS=(--home "$HOME_DIR" --keyring-backend test --chain-id $CHAIN_ID
         --node tcp://127.0.0.1:$RPCP --gas 500000 --gas-prices 1uzrn --yes
         --broadcast-mode sync -o json)

say() { echo "[drill $(date +%H:%M:%S)] $*"; }
die() { echo "[drill FATAL] $*" >&2; exit 1; }
height() { curl -s --max-time 2 $RPC/status | jq -r '.result.sync_info.latest_block_height // empty'; }
node_pid() { cat $DRILL_ROOT/node.pid 2>/dev/null; }

start_node() { # <binary> <tag>
  "$1" start --home "$HOME_DIR" --minimum-gas-prices 1uzrn > "$LOG/$2.log" 2>&1 &
  echo $! > $DRILL_ROOT/node.pid
  say "started $(basename "$1") pid $(node_pid)"
}

wait_halt() { # <target> — exit OR frozen-alive at boundary
  local target=$1 pid same=0 last="" h
  pid=$(node_pid)
  say "waiting for MANDATORY halt at $target…"
  while true; do
    kill -0 "$pid" 2>/dev/null || { say "node exited — halt"; return 0; }
    h=$(height || true)
    if [ -n "$h" ] && [ "$h" = "$last" ] && [ "$h" -ge $((target - 1)) ]; then
      same=$((same + 1))
      [ "$same" -ge 4 ] && { say "frozen at $h — halt; stopping old binary"; kill "$pid"; sleep 2; kill -9 "$pid" 2>/dev/null || true; return 0; }
    else same=0; fi
    last=$h; sleep 3
  done
}

check_tx() { # <hash>
  local i res code
  for i in $(seq 1 30); do
    res=$(curl -s "$RPC/tx?hash=0x$1" 2>/dev/null)
    code=$(echo "$res" | jq -r '.result.tx_result.code // empty')
    [ -n "$code" ] && { [ "$code" = "0" ] || die "tx $1 code=$code: $(echo "$res" | jq -r .result.tx_result.log)"; return 0; }
    sleep 1
  done
  die "tx $1 never included"
}

init_chain() {
  rm -rf "$HOME_DIR"
  "$OLD_BIN" init drill --chain-id $CHAIN_ID --default-denom uzrn --home "$HOME_DIR" >/dev/null 2>&1
  local G=$HOME_DIR/config/genesis.json
  jq '.consensus.params.block.max_gas="33333333"
    | .consensus.params.abci.vote_extensions_enable_height="1"
    | .app_state.staking.params.bond_denom="uzrn"
    | .app_state.gov.params.voting_period="60s"
    | .app_state.gov.params.expedited_voting_period="30s"
    | .app_state.gov.params.min_deposit[0].denom="uzrn"
    | .app_state.mint.params.mint_denom="uzrn"' "$G" > "$G.tmp" && mv "$G.tmp" "$G"
  for k in val panel1 panel2 panel3 panel4; do
    "$OLD_BIN" keys add $k --keyring-backend test --home "$HOME_DIR" >/dev/null 2>&1
  done
  "$OLD_BIN" add-genesis-account "$("$OLD_BIN" keys show val -a --keyring-backend test --home "$HOME_DIR")" 1000000000000uzrn --home "$HOME_DIR"
  for k in panel1 panel2 panel3 panel4; do
    "$OLD_BIN" add-genesis-account "$("$OLD_BIN" keys show $k -a --keyring-backend test --home "$HOME_DIR")" 200000000uzrn --home "$HOME_DIR"
  done
  "$OLD_BIN" genesis gentx val 100000000000uzrn --chain-id $CHAIN_ID --keyring-backend test --home "$HOME_DIR" --moniker drill-val >/dev/null 2>&1
  "$OLD_BIN" genesis collect-gentxs --home "$HOME_DIR" >/dev/null 2>&1
  "$OLD_BIN" genesis validate --home "$HOME_DIR" >/dev/null 2>&1
  sed -i.bak \
    -e "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://127.0.0.1:$P2P\"|" \
    -e "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:$RPCP\"|" \
    -e 's/^timeout_commit = .*/timeout_commit = "1s"/' \
    -e 's/^timeout_propose = .*/timeout_propose = "1s"/' \
    "$HOME_DIR/config/config.toml" && rm "$HOME_DIR/config/config.toml.bak"
  sed -i.bak \
    -e "s|address = \"localhost:9090\"|address = \"localhost:$GRPCP\"|" \
    -e "s|address = \"tcp://localhost:1317\"|address = \"tcp://localhost:$APIP\"|" \
    -e 's/^minimum-gas-prices = .*/minimum-gas-prices = "1uzrn"/' \
    -e 's/^max-txs = -1/max-txs = 5000/' \
    "$HOME_DIR/config/app.toml" && rm "$HOME_DIR/config/app.toml.bak"
  say "chain initialized (val 1M ZRN, 4 panel keys @ 200 ZRN, 1s blocks)"
}

upgrade_leg() { # <plan-name> <info-json> <old-bin> <new-bin> <tag>
  local name=$1 info=$2 oldbin=$3 newbin=$4 tag=$5
  say "=== LEG $tag: $name ==="
  local gov h target txh pidnum status applied
  gov=$("$oldbin" query auth module-account gov --node tcp://127.0.0.1:$RPCP -o json \
    | jq -r '.account.base_account.address // .account.value.address // empty')
  [ -n "$gov" ] || die "cannot resolve gov module account"
  h=$(height); target=$((h + 120))
  jq -cn --arg auth "$gov" --arg name "$name" --arg height "$target" --arg info "$info" '
    {messages:[{"@type":"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
                authority:$auth, plan:{name:$name, height:$height, info:$info}}],
     metadata:"", deposit:"10000000uzrn",
     title:("drill: "+$name), summary:("local lineage drill leg "+$name)}' \
    > $DRILL_ROOT/proposal-$tag.json
  txh=$("$oldbin" tx gov submit-proposal $DRILL_ROOT/proposal-$tag.json --from val "${TXFLAGS[@]}" | jq -r .txhash)
  check_tx "$txh"; sleep 3
  pidnum=$("$oldbin" query gov proposals --node tcp://127.0.0.1:$RPCP -o json \
    | jq -r '[.proposals[] | select(.status=="PROPOSAL_STATUS_VOTING_PERIOD")][-1].id')
  [ -n "$pidnum" ] && [ "$pidnum" != "null" ] || die "no proposal in voting period"
  txh=$("$oldbin" tx gov vote "$pidnum" yes --from val "${TXFLAGS[@]}" | jq -r .txhash)
  check_tx "$txh"
  say "proposal #$pidnum voted; waiting…"
  local i
  for i in $(seq 1 30); do
    status=$("$oldbin" query gov proposal "$pidnum" --node tcp://127.0.0.1:$RPCP -o json | jq -r '.proposal.status')
    [ "$status" = "PROPOSAL_STATUS_PASSED" ] && break
    { [ "$status" = "PROPOSAL_STATUS_REJECTED" ] || [ "$status" = "PROPOSAL_STATUS_FAILED" ]; } && die "proposal: $status"
    sleep 5
  done
  [ "$status" = "PROPOSAL_STATUS_PASSED" ] || die "proposal never passed"
  wait_halt "$target"
  start_node "$newbin" "$tag-post"
  local deadline=$((SECONDS + 240))
  until h=$(height) && [ -n "$h" ] && [ "$h" -gt "$target" ]; do
    [ $SECONDS -lt $deadline ] || { tail -30 "$LOG/$tag-post.log"; die "no resume past $target"; }
    kill -0 "$(node_pid)" 2>/dev/null || { tail -30 "$LOG/$tag-post.log"; die "new binary exited"; }
    sleep 2
  done
  applied=$("$newbin" q upgrade applied "$name" --node tcp://127.0.0.1:$RPCP 2>/dev/null | grep -o '[0-9]*' | head -1)
  [ "$applied" = "$target" ] || die "applied $name = $applied, expected $target"
  say "LEG $tag COMPLETE: $name applied at $target, chain at $(height)"
}

versions() {
  "$1" q upgrade module_versions --node tcp://127.0.0.1:$RPCP -o json \
    | jq -r '.module_versions[] | select(.name=="knowledge" or .name=="claiming_pot" or .name=="liquiditypool" or .name=="vesting_rewards") | "\(.name)=\(.version)"' | sort | tr '\n' ' '
}

case "${1:-all}" in
  all)
    init_chain
    start_node "$OLD_BIN" old
    until h=$(height) && [ -n "$h" ] && [ "$h" -ge 2 ]; do sleep 1; done
    say "OLD producing; versions: $(versions "$OLD_BIN")"
    # H1/H2 info: canonical compact sorted-key JSON, any keys — H1's startup wall
    # verifies it byte-for-byte against data/upgrade-info.json.
    upgrade_leg consolidation-safety-v1 "{\"commit\":\"$H1_COMMIT\",\"context\":\"local-lineage-drill\"}" "$OLD_BIN" "$H1_BIN" leg1
    say "post-H1 versions: $(versions "$H1_BIN")   (expect knowledge=6 claiming_pot=2 liquiditypool=5 vesting_rewards=1)"
    upgrade_leg founder-renunciation-v1 "{\"commit\":\"$H2_COMMIT\",\"context\":\"local-lineage-drill\"}" "$H1_BIN" "$H2_BIN" leg2
    say "post-H2 versions: $(versions "$H2_BIN")   (expect vesting_rewards=2)"
    # H3 info is NOT the generic commit JSON: the trunk handler parses plan.Info as
    # a legacy IBC keyset manifest (schema-checked, DisallowUnknownFields, canonical
    # field order) — a census of the OLD database's channel-upgrade and
    # pruning-sequence keys. A fresh drill chain has an empty census; for live nets
    # build it from the real old DB via app.BuildSDK053IBC10PlanInfo.
    EMPTY_SHA=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
    H3_INFO="{\"schema\":\"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1\",\"channel_upgrades\":{\"key_count\":\"0\",\"keys_sha256\":\"$EMPTY_SHA\"},\"pruning_sequence_start\":{\"key_count\":\"0\",\"keys_sha256\":\"$EMPTY_SHA\"}}"
    upgrade_leg sdk-0.53-ibc-10 "$H3_INFO" "$H2_BIN" "$MAIN_BIN" leg3
    say "post-H3 versions: $(versions "$MAIN_BIN")"
    say "ignition check (OpenQuestions must answer code 0):"
    curl -s "$RPC/abci_query?path=%22/zerone.knowledge.v1.Query/OpenQuestions%22&data=0x" | jq -c '.result.response | {code, log}'
    say "FULL LADDER COMPLETE"
    ;;
  teardown)
    kill "$(node_pid)" 2>/dev/null || true
    say "node stopped; state left at $DRILL_ROOT for inspection (rm -rf to discard)"
    ;;
  *) die "usage: drill.sh [all|teardown]" ;;
esac
