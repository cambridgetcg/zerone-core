#!/usr/bin/env bash
# Stand-in for zeroned in the ladder tests. Its behaviour comes from the "role:" line below,
# which build-mocks rewrites per rung (so every mock hashes differently).
# role: ROLE
set -euo pipefail
role="$(sed -n 's/^# role: //p' "$0" | head -n1)"
home=""; halt=""; args=("$@")
for ((k = 0; k < ${#args[@]}; k++)); do
  case "${args[k]}" in --home) home="${args[k + 1]}" ;; --halt-height) halt="${args[k + 1]}" ;; esac
done
log() { printf '{"level":"%s","module":"%s","message":"%s"}\n' "$1" "$2" "$3"; }
case "${1:-}" in
  init)
    mkdir -p "$home/config" "$home/data"
    cp /fixtures/config.toml /fixtures/app.toml "$home/config/"
    printf 'chain-id = ""\n' > "$home/config/client.toml"
    head -c 32 /dev/urandom > "$home/.pub"; pub="$(base64 -w0 < "$home/.pub")"
    addr="$(sha256sum < "$home/.pub" | cut -c1-40 | tr a-f A-F)"; rm -f "$home/.pub"
    if [ "$role" = "badkey" ]; then pub="LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80="; addr="93DBC1A1D9A07E67423B193395C9EB6F1E41B844"; fi
    printf '{\n  "address": "%s",\n  "pub_key": {\n    "type": "tendermint/PubKeyEd25519",\n    "value": "%s"\n  },\n  "priv_key": {\n    "type": "tendermint/PrivKeyEd25519",\n    "value": "TESTONLY"\n  }\n}' "$addr" "$pub" > "$home/config/priv_validator_key.json"
    printf '{"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"TESTONLY"}}' > "$home/config/node_key.json"
    printf '{\n  "height": "0",\n  "round": 0,\n  "step": 0\n}' > "$home/data/priv_validator_state.json"
    printf '{"genesis":"placeholder"}' > "$home/config/genesis.json"
    ;;
  genesis) exit 0 ;;
  comet) echo "0123456789abcdef0123456789abcdef01234567" ;;
  start)
    trap 'log info server "caught signal, shutting down"; exit 0' TERM INT
    log info consensus "ABCI Handshake App Info"
    sleep 2
    case "$role" in
      upgrade:*)
        IFS=: read -r _ name h <<< "$role"
        printf '{"name":"%s","time":"0001-01-01T00:00:00Z","height":%s}' "$name" "$h" > "$home/data/upgrade-info.json"
        log error x/upgrade "UPGRADE \\\"$name\\\" NEEDED at height: $h: "
        echo "panic: Failed to process committed block ($h:ABCDEF): UPGRADE \"$name\" NEEDED at height: $h: "
        exit 2 ;;
      halt)
        log error state "error in proxyAppConn.FinalizeBlock err=halt per configuration height $halt time 0"
        echo "panic: Failed to process committed block ($halt:ABCDEF): halt per configuration height $halt time 0"
        exit 2 ;;
      consensus-upgrade:*)
        IFS=: read -r _ name h <<< "$role"
        printf '{"name":"%s","time":"0001-01-01T00:00:00Z","height":%s}' "$name" "$h" > "$home/data/upgrade-info.json"
        log error consensus "CONSENSUS FAILURE!!! err=failed to apply block; error UPGRADE \\\"$name\\\" NEEDED at height: $h: "
        while true; do sleep 1; done ;;
      mismatch)
        for _ in 1 2 3; do log error blocksync "Error in validation height=1234 err=wrong Block.Header.AppHash.  Expected AAAA, got BBBB"; sleep 1; done
        while true; do sleep 1; done ;;
      early)
        log error x/upgrade "BINARY UPDATED BEFORE TRIGGER! UPGRADE \\\"x\\\" - in binary but not executed on chain. Downgrade your binary"
        while true; do sleep 1; done ;;
      crash) exit 1 ;;
      *) while true; do log info state "executed block"; sleep 2; done ;;
    esac ;;
  *) exit 0 ;;
esac
