#!/usr/bin/env bash
# zerone-1 non-voting observer: replays the whole chain from genesis by blocksync from the official
# validator over Fly's private 6PN network, switching binaries along ladder.tsv, then follows the
# tip with the lane binary (2e37c4c + CometBFT 0.38.25).
#
# What it never does: import or hold a validator key, publish a listener, accept an inbound peer,
# relay or submit a transaction, delete or reset node data. Every refusal stops the machine
# (exit 0 under restart policy on-failure) and leaves the volume for inspection.
#
# Progress and verdicts are single lines on stdout beginning with "OBSERVER ", so `fly logs`
# shows them without ssh. CometBFT checks every block's AppHash and LastResultsHash against the
# next signed header while applying it; this script stops the node and marks the volume if that
# check ever fails, instead of looping.
set -Eeuo pipefail
umask 077

# ---- fixed facts about zerone-1 (public, from genesis and the live chain) -------------------
CHAIN_ID="zerone-1"
GENESIS_SHA256="c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"
VALIDATOR_NODE_ID="ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e"
VALIDATOR_ADDRESS="93DBC1A1D9A07E67423B193395C9EB6F1E41B844"
VALIDATOR_PUBKEY="LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80="

# ---- layout (env overrides exist for the offline tests only) --------------------------------
OPT="${OBSERVER_OPT:-/opt/observer}"
DATA="${OBSERVER_DATA:-/data}"
REQUIRE_MOUNT="${OBSERVER_REQUIRE_MOUNT:-1}"
HOME_DIR="${DATA}/zeroned"
OBS="${DATA}/observer"
LADDER="${OPT}/ladder.tsv"
PEER_HOST="${OBSERVER_PEER_HOST:-zerone-1.internal}"
PEER="${VALIDATOR_NODE_ID}@${PEER_HOST}:26656"
UPSTREAM="${OBSERVER_UPSTREAM:-${PEER_HOST}:26657}"   # "off" disables the spot check
LOCAL_RPC_HOST="127.0.0.1"
LOCAL_RPC_PORT="${OBSERVER_LOCAL_RPC_PORT:-26657}"
TICK="${OBSERVER_TICK:-15}"
REPORT_EVERY="${OBSERVER_REPORT_EVERY:-60}"
SPOT_EVERY="${OBSERVER_SPOT_EVERY:-600}"
SPOT_WAIT="${OBSERVER_SPOT_WAIT:-300}"   # how long a spot waits for the validator's next header
PEERLESS_RESTART="${OBSERVER_PEERLESS_RESTART:-1800}"
STALL_RESTART="${OBSERVER_STALL_RESTART:-3600}"
MIN_FREE_KB="${OBSERVER_MIN_FREE_KB:-3145728}"          # 3 GiB
STOP_GRACE="${OBSERVER_STOP_GRACE:-60}"
EXPECT_FINAL_SHA="${OBSERVER_EXPECT_FINAL_SHA:-}"
PAUSE_UNIT="${OBSERVER_PAUSE_UNIT:-10}"
# While catching up, this node is load on the only signer. If the signer's newest block is older
# than VALIDATOR_LAG seconds, stop asking it for blocks for VALIDATOR_PAUSE seconds.
VALIDATOR_LAG="${OBSERVER_VALIDATOR_LAG:-150}"
VALIDATOR_PAUSE="${OBSERVER_VALIDATOR_PAUSE:-600}"
RUN_UID="${OBSERVER_UID:-10001}"

# Lines CometBFT, the SDK and x/upgrade print for the outcomes this script must act on.
EVENT_PATTERN='wrong Block\.Header\.(AppHash|LastResultsHash)|does not match AppHash after replay|committed an invalid block|CONSENSUS FAILURE|BINARY UPDATED BEFORE TRIGGER|wrong app version|NEEDED at|halt per configuration height|Failed to process committed block|error during handshake|error on replay'
MISMATCH_PATTERN='wrong Block\.Header\.(AppHash|LastResultsHash)|does not match AppHash after replay'
# A mismatch said by one of these is final at once (no blocksync retry follows).
FINAL_PATTERN='committed an invalid block|CONSENSUS FAILURE|Failed to process committed block|error on replay|after replay'

ZPID=""

ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }
jstr() { local s="${1//\\/\\\\}"; s="${s//\"/\\\"}"; printf '"%s"' "${s//$'\n'/ }"; }
say() {  # say <event> [key value]... ; values that look numeric or boolean stay unquoted
  local line event="$1"; shift
  line="{\"ts\":\"$(ts)\",\"event\":$(jstr "$event")"
  while [ "$#" -ge 2 ]; do
    if [[ "$2" =~ ^(-?[0-9]+|true|false|null)$ ]]; then line+=",\"$1\":$2"; else line+=",\"$1\":$(jstr "$2")"; fi
    shift 2
  done
  printf 'OBSERVER %s}\n' "$line"
  if [ -d "$OBS" ]; then printf '%s}\n' "$line" >> "$OBS/journal.jsonl" 2>/dev/null || true; fi
}

stop_machine() {  # stop_machine <reason class or "-"> <reason> : a deliberate, inspectable stop
  # Only a volume that diverged (APPHASH) or was altered (IDENTITY, CONFIG) is marked, so this
  # script never resumes it. Image-side problems (LADDER, INIT, CRASHLOOP) leave no mark: a
  # corrected image resumes the same volume, since x/upgrade refuses before any state is written.
  local marker="$1" reason="$2" marked=false
  stop_zeroned
  case "$marker" in
    APPHASH|IDENTITY|CONFIG)
      if [ -d "$OBS" ]; then printf '%s %s\n' "$(ts)" "$reason" > "$OBS/HALTED-$marker"; marked=true; fi ;;
  esac
  say stopped marker "$marker" reason "$reason" volume_marked "$marked" volume_preserved true
  exit 0
}

refuse() { stop_machine "${2:-REFUSED}" "$1"; }

stop_zeroned() {
  [ -n "$ZPID" ] || return 0
  if kill -0 "$ZPID" 2>/dev/null; then
    kill -TERM "$ZPID" 2>/dev/null || true
    local waited=0
    while kill -0 "$ZPID" 2>/dev/null && [ "$waited" -lt "$STOP_GRACE" ]; do sleep 1; waited=$((waited + 1)); done
    if kill -0 "$ZPID" 2>/dev/null; then say kill_after_grace seconds "$STOP_GRACE"; kill -KILL "$ZPID" 2>/dev/null || true; fi
  fi
  wait "$ZPID" 2>/dev/null || true
  ZPID=""
}

on_signal() { say signal action "stopping node cleanly"; stop_zeroned; say stopped marker "-" reason "signal" volume_preserved true; exit 0; }
trap on_signal TERM INT
trap 'say script_error line "$LINENO" command "$BASH_COMMAND"; stop_zeroned; exit 1' ERR

sha256_of() { sha256sum "$1" | cut -d' ' -f1; }

# ---- the ladder ------------------------------------------------------------------------------
declare -a R_NAME R_FIRST R_LAST R_END R_SHA
load_ladder() {
  [ -f "$LADDER" ] || refuse "ladder file missing: $LADDER" LADDER
  local name first last end sha rest n=0
  while IFS=$'\t' read -r name first last end sha rest; do
    case "$name" in ''|'#'*) continue ;; esac
    R_NAME[n]="$name"; R_FIRST[n]="$first"; R_LAST[n]="$last"; R_END[n]="$end"; R_SHA[n]="$sha"
    n=$((n + 1))
  done < "$LADDER"
  [ "$n" -ge 1 ] || refuse "ladder has no rows" LADDER
  local i
  for ((i = 0; i < n; i++)); do
    [[ "${R_SHA[i]}" =~ ^[0-9a-f]{64}$ ]] || refuse "rung ${R_NAME[i]} has no sha256 (${R_SHA[i]})" LADDER
    [[ "${R_FIRST[i]}" =~ ^[1-9][0-9]*$ ]] || refuse "rung ${R_NAME[i]} first height invalid" LADDER
    if [ "$i" -eq 0 ]; then
      [ "${R_FIRST[i]}" = "1" ] || refuse "first rung must start at height 1 (genesis replay)" LADDER
    else
      [ "${R_FIRST[i]}" -eq $((R_LAST[i - 1] + 1)) ] || refuse "rung ${R_NAME[i]} does not start at ${R_NAME[i - 1]} last+1" LADDER
    fi
    if [ "$i" -eq $((n - 1)) ]; then
      [ "${R_END[i]}" = "tip" ] && [ "${R_LAST[i]}" = "-" ] || refuse "last rung must be 'tip' with last '-'" LADDER
    else
      [[ "${R_LAST[i]}" =~ ^[1-9][0-9]*$ ]] && [ "${R_LAST[i]}" -ge "${R_FIRST[i]}" ] || refuse "rung ${R_NAME[i]} last height invalid" LADDER
      [[ "${R_END[i]}" =~ ^(halt|upgrade:[A-Za-z0-9._-]+)$ ]] || refuse "rung ${R_NAME[i]} end must be halt or upgrade:<name>" LADDER
    fi
    local bin="$OPT/bin/${R_SHA[i]}"
    [ -f "$bin" ] && [ -x "$bin" ] && [ ! -L "$bin" ] || refuse "binary for ${R_NAME[i]} missing: ${R_SHA[i]}" LADDER
    [ "$(sha256_of "$bin")" = "${R_SHA[i]}" ] || refuse "binary for ${R_NAME[i]} does not hash to its name" LADDER
  done
  RUNGS="$n"
  FINAL_SHA="${R_SHA[n - 1]}"
  if [ -n "$EXPECT_FINAL_SHA" ] && [ "$EXPECT_FINAL_SHA" != "$FINAL_SHA" ]; then
    refuse "final rung sha $FINAL_SHA is not the expected lane binary $EXPECT_FINAL_SHA" LADDER
  fi
  say ladder_verified rungs "$RUNGS" final_sha "$FINAL_SHA" genesis_sha "$GENESIS_SHA256"
}

rung_index() {  # name -> index
  local i
  for ((i = 0; i < RUNGS; i++)); do [ "${R_NAME[i]}" = "$1" ] && { echo "$i"; return 0; }; done
  return 1
}

# ---- config editing: one exact line per key, inside one section, or refuse -------------------
toml_set() {  # toml_set <file> <section or ""> <key> <toml literal>
  local file="$1" section="$2" key="$3" value="$4"
  awk -v sec="$section" -v key="$key" -v val="$value" '
    BEGIN { cur = ""; n = 0 }
    /^\[[^]]+\][ \t]*(#.*)?$/ { s = $0; sub(/^\[/, "", s); sub(/\].*$/, "", s); cur = s; print; next }
    {
      if (cur == sec) {
        line = $0
        sub(/^[ \t]+/, "", line)
        if (index(line, key) == 1) {
          rest = substr(line, length(key) + 1)
          if (rest ~ /^[ \t]*=/) { print key " = " val; n++; next }
        }
      }
      print
    }
    END { if (n != 1) exit 3 }' "$file" > "$file.edit" || { rm -f "$file.edit"; refuse "config key ${section:-top}.${key} not found exactly once in ${file##*/}" CONFIG; }
  mv "$file.edit" "$file"
}

configure_home() {
  local h="$1" c="$1/config/config.toml" a="$1/config/app.toml"
  toml_set "$c" "" moniker '"zerone-1-replay"'
  toml_set "$c" "" log_level '"state:error,*:info"'
  toml_set "$c" "" log_format '"json"'
  toml_set "$c" "" priv_validator_laddr '""'
  toml_set "$c" "" filter_peers false
  toml_set "$c" rpc laddr "\"tcp://${LOCAL_RPC_HOST}:${LOCAL_RPC_PORT}\""
  toml_set "$c" rpc unsafe false
  toml_set "$c" rpc cors_allowed_origins '[]'
  toml_set "$c" rpc pprof_laddr '""'
  toml_set "$c" p2p laddr '"tcp://127.0.0.1:26656"'
  toml_set "$c" p2p external_address '""'
  toml_set "$c" p2p seeds '""'
  toml_set "$c" p2p persistent_peers "\"${PEER}\""
  toml_set "$c" p2p unconditional_peer_ids "\"${VALIDATOR_NODE_ID}\""
  toml_set "$c" p2p private_peer_ids "\"${VALIDATOR_NODE_ID}\""
  toml_set "$c" p2p pex false
  toml_set "$c" p2p seed_mode false
  toml_set "$c" p2p addr_book_strict false
  toml_set "$c" p2p allow_duplicate_ip false
  toml_set "$c" p2p max_num_inbound_peers 0
  toml_set "$c" p2p max_num_outbound_peers 1
  toml_set "$c" mempool type '"nop"'
  toml_set "$c" mempool broadcast false
  toml_set "$c" statesync enable false
  toml_set "$c" blocksync version '"v0"'
  toml_set "$c" consensus double_sign_check_height 0
  toml_set "$c" consensus block_time_tolerance '"60s"'
  toml_set "$c" storage discard_abci_responses false
  toml_set "$c" tx_index indexer '"kv"'
  toml_set "$c" instrumentation prometheus false
  toml_set "$a" "" minimum-gas-prices '"0.025uzrn"'
  toml_set "$a" "" pruning '"default"'
  toml_set "$a" "" min-retain-blocks 0
  toml_set "$a" "" halt-height 0
  toml_set "$a" "" halt-time 0
  toml_set "$a" api enable false
  toml_set "$a" grpc enable false
  toml_set "$a" grpc-web enable false
  toml_set "$a" state-sync snapshot-interval 0
  toml_set "$a" telemetry enabled false
}

# ---- identity checks: a fresh throwaway key that is not the validator's ---------------------
flat() { tr -d '\n\r' < "$1"; }
check_identity() {
  local h="$1" key="$1/config/priv_validator_key.json" raw address pubtype pubkey derived
  [ -f "$key" ] && [ ! -L "$key" ] || refuse "no throwaway consensus key in the home" IDENTITY
  raw="$(flat "$key")"
  address="$(printf '%s' "$raw" | sed -n 's/^.*"address"[ \t]*:[ \t]*"\([0-9A-Fa-f]*\)".*$/\1/p')"
  pubtype="$(printf '%s' "$raw" | sed -n 's/^.*"pub_key"[ \t]*:[ \t]*{[^}]*"type"[ \t]*:[ \t]*"\([^"]*\)".*$/\1/p')"
  pubkey="$(printf '%s' "$raw" | sed -n 's/^.*"pub_key"[ \t]*:[ \t]*{[^}]*"value"[ \t]*:[ \t]*"\([^"]*\)".*$/\1/p')"
  raw=""
  [[ "$address" =~ ^[0-9A-F]{40}$ ]] || refuse "throwaway key address malformed" IDENTITY
  [ "$pubtype" = "tendermint/PubKeyEd25519" ] || refuse "throwaway key type is not ed25519" IDENTITY
  derived="$({ printf '%s' "$pubkey" | base64 -d 2>/dev/null || true; } | sha256sum | cut -c1-40 | tr 'a-f' 'A-F')"
  [ "$derived" = "$address" ] || refuse "throwaway key address does not match its public key" IDENTITY
  [ "$pubkey" != "$VALIDATOR_PUBKEY" ] || refuse "home holds the LIVE validator public key" IDENTITY
  [ "$address" != "$VALIDATOR_ADDRESS" ] || refuse "home holds the LIVE validator address" IDENTITY
  if grep -qF "$pubkey" "$h/config/genesis.json"; then refuse "throwaway key appears in genesis" IDENTITY; fi
  OBS_VALIDATOR_ADDRESS="$address"; OBS_VALIDATOR_PUBKEY="$pubkey"
}

check_signing_state() {
  local s="$1/data/priv_validator_state.json" raw
  [ -f "$s" ] || refuse "priv_validator_state.json missing" IDENTITY
  raw="$(flat "$s")"
  [[ "$raw" =~ \"height\"[[:space:]]*:[[:space:]]*\"0\" ]] && [[ "$raw" =~ \"round\"[[:space:]]*:[[:space:]]*0[^0-9] ]] && [[ "$raw" =~ \"step\"[[:space:]]*:[[:space:]]*0[^0-9] ]] \
    || refuse "observer signing state is not untouched (height/round/step not 0): it may have signed" IDENTITY
  if [[ "$raw" =~ \"signature\"[[:space:]]*:[[:space:]]*\"[^\"]+\" ]]; then refuse "observer signing state holds a signature" IDENTITY; fi
}

CONTROLS=(config/genesis.json config/config.toml config/app.toml config/node_key.json config/priv_validator_key.json)
record_controls() { (cd "$1" && sha256sum "${CONTROLS[@]}") > "$OBS/controls.sha256"; }
verify_controls() {
  (cd "$HOME_DIR" && sha256sum -c --quiet --status "$OBS/controls.sha256") || refuse "a pinned home control file changed since init" CONFIG
}

# ---- first boot ------------------------------------------------------------------------------
init_home() {
  local final_bin="$OPT/bin/$FINAL_SHA" tmp="$DATA/zeroned.init" node_id
  [ ! -e "$HOME_DIR" ] || refuse "a node home exists without an observer state file; nothing touched" INIT
  [ "$(sha256_of "$OPT/genesis.json")" = "$GENESIS_SHA256" ] || refuse "baked genesis is not zerone-1's" INIT
  rm -rf "$tmp"   # only ever our own half-made init directory
  mkdir -p "$tmp"
  "$final_bin" init zerone-1-replay --chain-id "$CHAIN_ID" --home "$tmp" > /dev/null 2>&1 || refuse "zeroned init failed" INIT
  cp "$OPT/genesis.json" "$tmp/config/genesis.json"
  [ "$(sha256_of "$tmp/config/genesis.json")" = "$GENESIS_SHA256" ] || refuse "genesis copy mismatch" INIT
  configure_home "$tmp"
  "$final_bin" genesis validate "$tmp/config/genesis.json" --home "$tmp" > /dev/null 2>&1 || refuse "genesis validate failed with the final binary" INIT
  check_identity "$tmp"
  check_signing_state "$tmp"
  node_id="$("$final_bin" comet show-node-id --home "$tmp" 2>/dev/null)" || refuse "cannot read the fresh node id" INIT
  [[ "$node_id" =~ ^[0-9a-f]{40}$ ]] && [ "$node_id" != "$VALIDATOR_NODE_ID" ] || refuse "fresh node id invalid or equal to the validator's" INIT
  mv "$tmp" "$HOME_DIR"
  record_controls "$HOME_DIR"
  printf '%s\n' "$node_id" > "$OBS/node_id"
  printf 'rung=%s\n' "${R_NAME[0]}" > "$OBS/state.tmp" && mv "$OBS/state.tmp" "$OBS/state"
  say initialized node_id "$node_id" consensus_address "$OBS_VALIDATOR_ADDRESS" consensus_pubkey "$OBS_VALIDATOR_PUBKEY" \
    validator_power 0 peer "$PEER" genesis_sha "$GENESIS_SHA256"
}

# ---- tiny HTTP GET over bash /dev/tcp (no curl in the image) --------------------------------
http_get() {  # http_get <host> <port> <path>; prints the body
  # shellcheck disable=SC2016  # the single-quoted body is expanded by the inner bash
  timeout 10 bash -c 'exec 3<>"/dev/tcp/$0/$1" || exit 1; printf "GET %s HTTP/1.0\r\nHost: %s\r\nConnection: close\r\n\r\n" "$2" "$0" >&3; cat <&3' \
    "$1" "$2" "$3" 2>/dev/null | tr -d '\r' | sed '1,/^$/d'
}
# Field getters for the node's own small JSON replies. Each key read here occurs once per reply.
jget() { tr -d '\n' | sed -n "s/^.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*\$/\1/p"; }
jgetraw() { tr -d '\n' | sed -n "s/^.*\"$1\"[[:space:]]*:[[:space:]]*\([^,}[:space:]]*\).*\$/\1/p"; }
b64hex() { { base64 -d 2>/dev/null || true; } | od -An -v -tx1 | tr -d ' \n' | tr 'a-f' 'A-F'; }

wait_for_peer_dns() {  # CometBFT resolves persistent_peers once, at start, and exits if it cannot
  local waited=0
  until getent ahosts "$PEER_HOST" > /dev/null 2>&1; do
    if [ $((waited % 300)) -eq 0 ]; then say waiting_for_peer_dns host "$PEER_HOST" seconds "$waited"; fi
    sleep 15 & wait "$!" || true
    waited=$((waited + 15))
  done
  say peer_dns host "$PEER_HOST" addresses "$(getent ahosts "$PEER_HOST" | awk '{print $1}' | sort -u | tr '\n' ' ')"
}

# ---- running one rung ------------------------------------------------------------------------
upgrade_info() {  # prints "name height" from data/upgrade-info.json, or nothing
  local f="$HOME_DIR/data/upgrade-info.json" raw name height
  [ -f "$f" ] || return 0
  raw="$(flat "$f")"
  name="$(printf '%s' "$raw" | sed -n 's/^.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*$/\1/p')"
  height="$(printf '%s' "$raw" | sed -n 's/^.*"height"[[:space:]]*:[[:space:]]*"\{0,1\}\([0-9]*\).*$/\1/p')"
  printf '%s %s\n' "$name" "$height"
}

LAST_REPORT=0; LAST_SPOT=0; PREV_H=""; PREV_T=""; PEERLESS_SINCE=""; STALL_H=""; STALL_SINCE=""
SPOT_H=""; SPOT_HEX=""; SPOT_T=0
report() {  # report <rung index>; sets REPORT_H, REPORT_PEERS, REPORT_CATCHING
  local i="$1" st ni now h bt ah cu peers rate=""
  st="$(http_get "$LOCAL_RPC_HOST" "$LOCAL_RPC_PORT" /status || true)"
  REPORT_H=""; REPORT_PEERS=""; REPORT_CATCHING=""
  if [ -z "$st" ]; then say progress rung "${R_NAME[i]}" rpc "not answering yet"; return 0; fi
  h="$(printf '%s' "$st" | jget latest_block_height)"; bt="$(printf '%s' "$st" | jget latest_block_time)"
  ah="$(printf '%s' "$st" | jget latest_app_hash)"; cu="$(printf '%s' "$st" | jgetraw catching_up)"
  ni="$(http_get "$LOCAL_RPC_HOST" "$LOCAL_RPC_PORT" /net_info || true)"
  peers="$(printf '%s' "$ni" | jget n_peers)"
  now="$(date +%s)"
  if [ -n "$PREV_H" ] && [ -n "$h" ] && [ "$now" -gt "$PREV_T" ]; then rate=$(( (h - PREV_H) / (now - PREV_T) )); fi
  PREV_H="$h"; PREV_T="$now"
  REPORT_H="$h"; REPORT_PEERS="${peers:-0}"; REPORT_CATCHING="$cu"
  say progress rung "${R_NAME[i]}" rung_first "${R_FIRST[i]}" rung_last "${R_LAST[i]}" height "${h:-0}" \
    blocks_per_s "${rate:-0}" catching_up "${cu:-null}" peers "${peers:-0}" header_time "$bt" header_app_hash "$ah"
}

spot_check() {  # compare this node's applied app hash at H with the validator's header H+1
  # Following at the tip, the validator's H+1 header usually is not out yet when H is read, so the
  # captured H waits for it (rechecked every tick, up to SPOT_WAIT seconds). Without the wait,
  # 435 of 437 following spots on 2026-11-07..10 read "upstream unavailable".
  local ai h lh up uph now
  [ "$UPSTREAM" != "off" ] || return 0
  now="$(date +%s)"
  if [ -z "$SPOT_H" ]; then
    ai="$(http_get "$LOCAL_RPC_HOST" "$LOCAL_RPC_PORT" /abci_info || true)"
    h="$(printf '%s' "$ai" | jget last_block_height)"; lh="$(printf '%s' "$ai" | jget last_block_app_hash)"
    [[ "$h" =~ ^[1-9][0-9]*$ ]] && [ -n "$lh" ] || return 0
    SPOT_H="$h"; SPOT_HEX="$(printf '%s' "$lh" | b64hex || true)"; SPOT_T="$now"
  fi
  up="$(http_get "${UPSTREAM%:*}" "${UPSTREAM##*:}" "/header?height=$((SPOT_H + 1))" || true)"
  uph="$(printf '%s' "$up" | jget app_hash)"
  if [ -z "$uph" ]; then
    [ $((now - SPOT_T)) -ge "$SPOT_WAIT" ] || return 0
    say apphash_spot height "$SPOT_H" result "upstream unavailable" waited_s "$((now - SPOT_T))"
  elif [ "$uph" = "$SPOT_HEX" ]; then
    say apphash_spot height "$SPOT_H" result MATCH applied_app_hash "$SPOT_HEX" validator_header "$((SPOT_H + 1))"
  else say apphash_spot height "$SPOT_H" result DIFFER applied_app_hash "$SPOT_HEX" validator_header_app_hash "$uph"; fi
  SPOT_H=""
}

validator_lagging() {  # true when the signer's newest block is older than VALIDATOR_LAG seconds
  local st bt bts now
  [ "$UPSTREAM" != "off" ] || return 1
  st="$(http_get "${UPSTREAM%:*}" "${UPSTREAM##*:}" /status || true)"
  bt="$(printf '%s' "$st" | jget latest_block_time)"
  [ -n "$bt" ] || return 1                       # unreadable: the peerless and stall watchdogs cover it
  bts="$(date -u -d "$bt" +%s 2>/dev/null)" || return 1
  now="$(date -u +%s)"
  VALIDATOR_AGE=$((now - bts))
  [ "$VALIDATOR_AGE" -gt "$VALIDATOR_LAG" ]
}

disk_guard() {
  local free
  free="$(df -Pk "$DATA" | awk 'NR == 2 { print $4 }')"
  if [[ "$free" =~ ^[0-9]+$ ]] && [ "$free" -lt "$MIN_FREE_KB" ]; then
    stop_machine - "free space on the volume below $((MIN_FREE_KB / 1024)) MiB ($((free / 1024)) MiB left): extend the volume, then start the machine"
  fi
}

has() { grep -qE "$1" <<< "$2"; }          # has <regex> <text>   (here-string: no SIGPIPE)
count() { grep -cE "$1" <<< "$2" || true; } # count <regex> <text>
lastof() { grep -E "$1" <<< "$2" | tail -n1 | cut -c1-900 || true; }

run_rung() {  # run_rung <index>; sets OUTCOME to advanced | restart | crashed
  local i="$1" bin="$OPT/bin/${R_SHA[$1]}" end="${R_END[$1]}" events="$OBS/events-${R_NAME[$1]}.log" started now fresh
  local -a args=(start --home "$HOME_DIR")
  local halt_at="" want_upgrade="" announced_upgrade="" mark_lines
  if [ "$end" = "halt" ]; then halt_at=$((R_LAST[i] + 1)); args+=(--halt-height "$halt_at"); fi
  case "$end" in upgrade:*) want_upgrade="${end#upgrade:}" ;; esac
  [ "$(sha256_of "$bin")" = "${R_SHA[i]}" ] || refuse "binary for ${R_NAME[i]} changed on disk" LADDER
  printf '\n--- %s start %s\n' "$(ts)" "${R_NAME[i]}" >> "$events"
  mark_lines="$(wc -l < "$events")"
  say rung_start rung "${R_NAME[i]}" sha256 "${R_SHA[i]}" first "${R_FIRST[i]}" last "${R_LAST[i]}" end "$end" halt_height "${halt_at:-null}"
  "$bin" "${args[@]}" > >(tee >(grep --line-buffered -E "$EVENT_PATTERN" >> "$events" || true)) 2>&1 &
  ZPID=$!
  started="$(date +%s)"; LAST_REPORT=0; LAST_SPOT=0; PEERLESS_SINCE=""; STALL_H=""; STALL_SINCE=""; PREV_H=""
  OUTCOME=""
  while kill -0 "$ZPID" 2>/dev/null; do
    sleep "$TICK" & wait "$!" || true
    now="$(date +%s)"
    fresh="$(tail -n +"$((mark_lines + 1))" "$events" 2>/dev/null || true)"
    # 1. The proof failing. Blocksync logs "Error in validation" for every retry of the same block,
    #    so three lines settle it; a consensus-mode or replay failure is said once and is final.
    if [ "$(count "$MISMATCH_PATTERN" "$fresh")" -ge 3 ] || has "($MISMATCH_PATTERN).*($FINAL_PATTERN)|($FINAL_PATTERN).*($MISMATCH_PATTERN)" "$fresh"; then
      say APPHASH_MISMATCH rung "${R_NAME[i]}" evidence "$(lastof "$MISMATCH_PATTERN" "$fresh")"
      stop_machine APPHASH "app hash or results hash mismatch in ${R_NAME[i]}; see ${events}"
    fi
    # 2. The ladder disagrees with the chain's upgrade history.
    if has 'BINARY UPDATED BEFORE TRIGGER|wrong app version' "$fresh"; then
      say LADDER_WRONG rung "${R_NAME[i]}" evidence "$(lastof 'BINARY UPDATED BEFORE TRIGGER|wrong app version' "$fresh")"
      stop_machine LADDER "binary ${R_NAME[i]} disagrees with the chain's upgrade history"
    fi
    # 3. A committed block too far ahead of this machine's clock (block_time_tolerance).
    if has 'too far in the future' "$fresh"; then
      say clock_tolerance rung "${R_NAME[i]}" local_time "$(ts)" evidence "$(lastof 'too far in the future' "$fresh")"
      stop_zeroned; sleep 120 & wait "$!" || true; OUTCOME=restart; return 0
    fi
    # 4. x/upgrade halts. In blocksync the process exits by itself (handled after the loop); while
    #    following in consensus mode CometBFT keeps the process up, so stop it here.
    if has 'NEEDED at' "$fresh"; then
      if [ -n "$want_upgrade" ] && [ "$(upgrade_info)" = "$want_upgrade $((R_LAST[i] + 1))" ]; then
        say upgrade_halt rung "${R_NAME[i]}" plan "$want_upgrade" height "$((R_LAST[i] + 1))"; stop_zeroned; break
      fi
      if [ "$end" = "tip" ] && [ -z "$announced_upgrade" ]; then
        announced_upgrade="$(upgrade_info)"
        say UPGRADE_NEEDED rung "${R_NAME[i]}" plan_and_height "$announced_upgrade" \
          action "add a rung with the patched binary for this plan to ladder.tsv and redeploy; the node stays up meanwhile"
      fi
    elif has 'CONSENSUS FAILURE' "$fresh"; then
      say consensus_failure rung "${R_NAME[i]}" evidence "$(lastof 'CONSENSUS FAILURE' "$fresh")" action "restarting the node"
      stop_zeroned; OUTCOME=restart; return 0
    fi
    # 5. Progress, spot check and watchdogs.
    if [ $((now - LAST_REPORT)) -ge "$REPORT_EVERY" ]; then
      LAST_REPORT="$now"; report "$i"; disk_guard
      if [ "${REPORT_CATCHING:-}" = "true" ] && validator_lagging; then
        say validator_lag_pause rung "${R_NAME[i]}" validator_block_age_s "$VALIDATOR_AGE" pause_seconds "$VALIDATOR_PAUSE" \
          note "the signer is behind; not loading it while it recovers"
        stop_zeroned; sleep "$VALIDATOR_PAUSE" & wait "$!" || true; OUTCOME=restart; return 0
      fi
      if [ -n "$REPORT_H" ] && [ "${REPORT_PEERS:-0}" = "0" ]; then
        [ -n "$PEERLESS_SINCE" ] || PEERLESS_SINCE="$now"
        if [ $((now - PEERLESS_SINCE)) -ge "$PEERLESS_RESTART" ]; then
          say peerless_restart rung "${R_NAME[i]}" seconds "$((now - PEERLESS_SINCE))" note "re-resolving ${PEER_HOST} and redialling"
          stop_zeroned; OUTCOME=restart; return 0
        fi
      else PEERLESS_SINCE=""; fi
      if [ "${REPORT_CATCHING:-}" = "true" ] && [ -n "$REPORT_H" ]; then
        if [ "$REPORT_H" != "$STALL_H" ]; then STALL_H="$REPORT_H"; STALL_SINCE="$now"
        elif [ $((now - STALL_SINCE)) -ge "$STALL_RESTART" ]; then
          say stall_restart rung "${R_NAME[i]}" height "$REPORT_H" seconds "$((now - STALL_SINCE))"
          stop_zeroned; OUTCOME=restart; return 0
        fi
      fi
    fi
    if [ -n "$SPOT_H" ]; then spot_check
    elif [ $((now - LAST_SPOT)) -ge "$SPOT_EVERY" ]; then LAST_SPOT="$now"; spot_check; fi
  done
  local rc=0 ran
  if [ -n "$ZPID" ]; then wait "$ZPID" || rc=$?; ZPID=""; fi
  sleep 2   # let tee and grep flush the last lines
  ran=$(( $(date +%s) - started ))
  fresh="$(tail -n +"$((mark_lines + 1))" "$events" 2>/dev/null || true)"
  if [ "$(count "$MISMATCH_PATTERN" "$fresh")" -ge 1 ]; then
    say APPHASH_MISMATCH rung "${R_NAME[i]}" evidence "$(lastof "$MISMATCH_PATTERN" "$fresh")"
    stop_machine APPHASH "app hash or results hash mismatch at exit of ${R_NAME[i]}; see ${events}"
  fi
  if has 'BINARY UPDATED BEFORE TRIGGER|wrong app version' "$fresh"; then
    say LADDER_WRONG rung "${R_NAME[i]}" evidence "$(lastof 'BINARY UPDATED BEFORE TRIGGER|wrong app version' "$fresh")"
    stop_machine LADDER "binary ${R_NAME[i]} disagrees with the chain's upgrade history"
  fi
  if [ -n "$want_upgrade" ] && [ "$(upgrade_info)" = "$want_upgrade $((R_LAST[i] + 1))" ]; then
    say rung_done rung "${R_NAME[i]}" by "upgrade $want_upgrade" next_height "$((R_LAST[i] + 1))" exit_code "$rc" seconds "$ran"
    OUTCOME=advanced; return 0
  fi
  if [ -n "$halt_at" ] && has "halt per configuration height ${halt_at} " "$fresh"; then
    say rung_done rung "${R_NAME[i]}" by "halt-height $halt_at" next_height "$halt_at" exit_code "$rc" seconds "$ran"
    OUTCOME=advanced; return 0
  fi
  if [ -n "$want_upgrade" ] && has 'NEEDED at' "$fresh"; then
    say LADDER_WRONG rung "${R_NAME[i]}" expected "$want_upgrade $((R_LAST[i] + 1))" found "$(upgrade_info)"
    stop_machine LADDER "rung ${R_NAME[i]} stopped for a different upgrade than the ladder says"
  fi
  say node_exit rung "${R_NAME[i]}" exit_code "$rc" seconds "$ran" last_event "$(lastof '.' "$fresh")"
  RAN_SECONDS="$ran"; OUTCOME=crashed
}

# ---- main ------------------------------------------------------------------------------------
main() {
  if [ "$(id -u)" = "0" ]; then
    # Root only prepares the volume, then the node runs as an unprivileged uid with no capabilities.
    if [ "$REQUIRE_MOUNT" = "1" ]; then mountpoint -q "$DATA" || { say stopped reason "$DATA is not a mounted volume; refusing to sync into the root filesystem"; exit 0; }; fi
    chown "$RUN_UID:$RUN_UID" "$DATA"
    exec setpriv --reuid="$RUN_UID" --regid="$RUN_UID" --clear-groups --inh-caps=-all --bounding-set=-all --no-new-privs -- "$0" "$@"
  fi
  if [ "$REQUIRE_MOUNT" = "1" ]; then mountpoint -q "$DATA" || { say stopped reason "$DATA is not a mounted volume; refusing to sync into the root filesystem"; exit 0; }; fi
  mkdir -p "$OBS"
  local m
  for m in "$OBS"/HALTED-*; do
    [ -e "$m" ] || continue
    say stopped reason "volume is marked ${m##*/}: $(cat "$m")" action "inspect; this script will not resume a halted volume"
    exit 0
  done
  load_ladder
  if [ ! -f "$OBS/state" ]; then init_home; fi
  verify_controls
  check_identity "$HOME_DIR"
  check_signing_state "$HOME_DIR"
  say boot node_id "$(cat "$OBS/node_id" 2>/dev/null || echo unknown)" consensus_address "$OBS_VALIDATOR_ADDRESS" peer "$PEER" local_time "$(ts)"
  local fast_fails=0 i name
  while true; do
    name="$(sed -n 's/^rung=\(.*\)$/\1/p' "$OBS/state")"
    i="$(rung_index "$name")" || refuse "state names unknown rung '$name'" LADDER
    check_signing_state "$HOME_DIR"
    wait_for_peer_dns
    RAN_SECONDS=0
    run_rung "$i"
    case "$OUTCOME" in
      advanced)
        fast_fails=0
        [ "$i" -lt $((RUNGS - 1)) ] || refuse "the tip rung cannot advance" LADDER
        printf 'rung=%s\n' "${R_NAME[i + 1]}" > "$OBS/state.tmp" && mv "$OBS/state.tmp" "$OBS/state"
        say rung_advance from "${R_NAME[i]}" to "${R_NAME[i + 1]}" ;;
      restart) fast_fails=0 ;;
      crashed)
        if [ "$RAN_SECONDS" -lt 120 ]; then fast_fails=$((fast_fails + 1)); else fast_fails=0; fi
        if [ "$fast_fails" -ge 8 ]; then stop_machine CRASHLOOP "zeroned exited 8 times within 120 s each in ${R_NAME[i]}"; fi
        local pause=$((PAUSE_UNIT * (1 << (fast_fails > 5 ? 5 : fast_fails))))
        say restart_after_exit rung "${R_NAME[i]}" pause_seconds "$pause" consecutive_fast_exits "$fast_fails"
        sleep "$pause" & wait "$!" || true ;;
    esac
  done
}

main "$@"
