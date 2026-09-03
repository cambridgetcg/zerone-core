#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
PRODUCTION_ENTRYPOINT="${ROOT}/deploy/mainnet/entrypoint.sh"
TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT

if ! command -v flock >/dev/null 2>&1; then
  mkdir -p "${TMP}/bin"
  cat > "${TMP}/bin/flock" <<'FLOCK'
#!/usr/bin/env python3
import fcntl
import sys

if len(sys.argv) != 3 or sys.argv[1] != "-n":
    raise SystemExit(2)
try:
    fcntl.flock(int(sys.argv[2]), fcntl.LOCK_EX | fcntl.LOCK_NB)
except BlockingIOError:
    raise SystemExit(1)
FLOCK
  chmod +x "${TMP}/bin/flock"
  export PATH="${TMP}/bin:${PATH}"
fi

fail() {
  echo "entrypoint test: FAIL: $*" >&2
  exit 1
}

file_mode() {
  if stat -c '%a' "$1" >/dev/null 2>&1; then
    stat -c '%a' "$1"
  else
    stat -f '%Lp' "$1"
  fi
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    openssl dgst -sha256 "$1" | awk '{print $NF}'
  fi
}

toml_test_set() {
  local section="$1" key="$2" value="$3" file="$4" tmp
  tmp="${file}.test-tmp"
  awk -v target="[${section}]" -v wanted="${key}" -v replacement="${value}" '
    BEGIN { active = 0; changed = 0 }
    /^\[/ { active = ($0 == target) }
    {
      if (active && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=") {
        print wanted " = " replacement
        changed++
        next
      }
      print
    }
    END { if (changed != 1) exit 42 }
  ' "${file}" > "${tmp}" || fail "could not mutate ${section}.${key} for restart test"
  mv "${tmp}" "${file}"
}

toml_test_set_root() {
  local key="$1" value="$2" file="$3" tmp
  tmp="${file}.test-tmp"
  awk -v wanted="${key}" -v replacement="${value}" '
    BEGIN { before_section = 1; changed = 0 }
    /^\[/ { before_section = 0 }
    {
      if (before_section && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=") {
        print wanted " = " replacement
        changed++
        next
      }
      print
    }
    END { if (changed != 1) exit 42 }
  ' "${file}" > "${tmp}" || fail "could not mutate root ${key} for restart test"
  mv "${tmp}" "${file}"
}

toml_value() {
  local section="$1" key="$2" file="$3"
  awk -v target="[${section}]" -v wanted="${key}" '
    BEGIN { active = 0; count = 0 }
    /^\[/ { active = ($0 == target) }
    active && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=" {
      value = $0
      sub("^[[:space:]]*" wanted "[[:space:]]*=[[:space:]]*", "", value)
      count++
    }
    END {
      if (count != 1) exit 42
      print value
    }
  ' "${file}"
}

toml_root_value() {
  local key="$1" file="$2"
  awk -v wanted="${key}" '
    BEGIN { before_section = 1; count = 0 }
    /^\[/ { before_section = 0 }
    before_section && $0 ~ "^[[:space:]]*" wanted "[[:space:]]*=" {
      value = $0
      sub("^[[:space:]]*" wanted "[[:space:]]*=[[:space:]]*", "", value)
      count++
    }
    END {
      if (count != 1) exit 42
      print value
    }
  ' "${file}"
}

assert_toml_value() {
  local section="$1" key="$2" expected="$3" file="$4" label="$5" actual
  actual=$(toml_value "${section}" "${key}" "${file}") || \
    fail "${label}: ${section}.${key} was missing or duplicated"
  [ "${actual}" = "${expected}" ] || \
    fail "${label}: ${section}.${key} was ${actual}, expected ${expected}"
}

assert_root_toml_value() {
  local key="$1" expected="$2" file="$3" label="$4" actual
  actual=$(toml_root_value "${key}" "${file}") || \
    fail "${label}: root ${key} was missing or duplicated"
  [ "${actual}" = "${expected}" ] || \
    fail "${label}: root ${key} was ${actual}, expected ${expected}"
}

assert_bounded_query_config() {
  local app="$1" config="$2" label="$3"
  assert_root_toml_value query-gas-limit '"5000000"' "${app}" "${label}"
  assert_toml_value api rpc-read-timeout 10 "${app}" "${label}"
  assert_toml_value api rpc-write-timeout 15 "${app}" "${label}"
  assert_toml_value api rpc-max-body-bytes 65536 "${app}" "${label}"
  assert_toml_value rpc unsafe false "${config}" "${label}"
  assert_toml_value rpc max_request_batch_size 1 "${config}" "${label}"
  assert_toml_value rpc max_body_bytes 65536 "${config}" "${label}"
  assert_toml_value rpc max_header_bytes 16384 "${config}" "${label}"
  assert_toml_value storage discard_abci_responses false "${config}" "${label}"
  assert_toml_value tx_index indexer '"kv"' "${config}" "${label}"
}

drift_bounded_query_config() {
  local app="$1" config="$2"
  toml_test_set_root query-gas-limit '"0"' "${app}"
  toml_test_set api rpc-read-timeout 0 "${app}"
  toml_test_set api rpc-write-timeout 0 "${app}"
  toml_test_set api rpc-max-body-bytes 1000000 "${app}"
  toml_test_set rpc unsafe true "${config}"
  toml_test_set rpc max_request_batch_size 0 "${config}"
  toml_test_set rpc max_body_bytes 1000000 "${config}"
  toml_test_set rpc max_header_bytes 1048576 "${config}"
  toml_test_set storage discard_abci_responses true "${config}"
  toml_test_set tx_index indexer '"null"' "${config}"
}

mkdir -p "${TMP}/bin" "${TMP}/seed"
export PATH="${TMP}/bin:${PATH}"

cat > "${TMP}/keytool.go" <<'KEYTOOL'
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type cometKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type filePV struct {
	Address string   `json:"address"`
	PubKey  cometKey `json:"pub_key"`
	PrivKey cometKey `json:"priv_key"`
}

type nodeKey struct {
	PrivKey cometKey `json:"priv_key"`
}

func key(label string) (ed25519.PrivateKey, ed25519.PublicKey) {
	seed := sha256.Sum256([]byte(label))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return priv, priv.Public().(ed25519.PublicKey)
}

func cmKey(kind string, value []byte) cometKey {
	return cometKey{Type: kind, Value: base64.StdEncoding.EncodeToString(value)}
}

func address(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:20])
}

func writeJSON(path string, value any) {
	bz, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, append(bz, '\n'), 0o600); err != nil {
		panic(err)
	}
}

func generate(dir string) {
	validatorPriv, validatorPub := key("zerone-entrypoint-test-validator")
	foreignPriv, foreignPub := key("zerone-entrypoint-test-foreign")
	rotatedPriv, rotatedPub := key("zerone-entrypoint-test-rotated-validator")
	nodePriv, _ := key("zerone-entrypoint-test-node")
	foreignNodePriv, _ := key("zerone-entrypoint-test-foreign-node")
	rotatedNodePriv, _ := key("zerone-entrypoint-test-rotated-node")

	writeJSON(filepath.Join(dir, "validator_key.json"), filePV{
		Address: address(validatorPub),
		PubKey:  cmKey("tendermint/PubKeyEd25519", validatorPub),
		PrivKey: cmKey("tendermint/PrivKeyEd25519", validatorPriv),
	})
	// This is the dangerous shape: the stored public key claims to be the
	// genesis validator, but Comet derives a different public key from priv_key.
	writeJSON(filepath.Join(dir, "mismatched_validator_key.json"), filePV{
		Address: address(validatorPub),
		PubKey:  cmKey("tendermint/PubKeyEd25519", validatorPub),
		PrivKey: cmKey("tendermint/PrivKeyEd25519", foreignPriv),
	})
	writeJSON(filepath.Join(dir, "foreign_validator_key.json"), filePV{
		Address: address(foreignPub),
		PubKey:  cmKey("tendermint/PubKeyEd25519", foreignPub),
		PrivKey: cmKey("tendermint/PrivKeyEd25519", foreignPriv),
	})
	writeJSON(filepath.Join(dir, "rotated_validator_key.json"), filePV{
		Address: address(rotatedPub),
		PubKey:  cmKey("tendermint/PubKeyEd25519", rotatedPub),
		PrivKey: cmKey("tendermint/PrivKeyEd25519", rotatedPriv),
	})
	writeJSON(filepath.Join(dir, "node_key.json"), nodeKey{
		PrivKey: cmKey("tendermint/PrivKeyEd25519", nodePriv),
	})
	writeJSON(filepath.Join(dir, "foreign_node_key.json"), nodeKey{
		PrivKey: cmKey("tendermint/PrivKeyEd25519", foreignNodePriv),
	})
	writeJSON(filepath.Join(dir, "rotated_node_key.json"), nodeKey{
		PrivKey: cmKey("tendermint/PrivKeyEd25519", rotatedNodePriv),
	})
	writeJSON(filepath.Join(dir, "seed", "genesis.json"), map[string]any{
		"chain_id": "zerone-1",
		"app_state": map[string]any{
			"genutil": map[string]any{
				"gen_txs": []any{map[string]any{
					"body": map[string]any{"messages": []any{map[string]any{
						"pubkey": map[string]any{"key": base64.StdEncoding.EncodeToString(validatorPub)},
					}}},
				}},
			},
		},
	})
}

func loadPrivate(path string) ed25519.PrivateKey {
	var payload struct {
		PrivKey cometKey `json:"priv_key"`
	}
	bz, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(bz, &payload); err != nil {
		panic(err)
	}
	priv, err := base64.StdEncoding.DecodeString(payload.PrivKey.Value)
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		panic("invalid Ed25519 private key")
	}
	return ed25519.PrivateKey(priv)
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: keytool generate DIR | validator FILE | node FILE")
	}
	switch os.Args[1] {
	case "generate":
		generate(os.Args[2])
	case "validator":
		pub := loadPrivate(os.Args[2]).Public().(ed25519.PublicKey)
		fmt.Printf("{\"@type\":\"/cosmos.crypto.ed25519.PubKey\",\"key\":%q}\n",
			base64.StdEncoding.EncodeToString(pub))
	case "node":
		pub := loadPrivate(os.Args[2]).Public().(ed25519.PublicKey)
		fmt.Println(address(pub))
	default:
		panic("unknown mode")
	}
}
KEYTOOL
go build -o "${TMP}/bin/keytool" "${TMP}/keytool.go"
"${TMP}/bin/keytool" generate "${TMP}"

cat > "${TMP}/bin/zeroned" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail

if env | grep -Eq '^(PRIV_VALIDATOR_KEY_B64|NODE_KEY_B64)='; then
  printf 'bootstrap secret leaked to zeroned child: %s\n' "$*" \
    >> "${CHILD_SECRET_LEAK_FILE:?}"
fi

case "${1:-}" in
  init)
    shift
    home=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--home" ]; then
        home="$2"
        shift 2
      else
        shift
      fi
    done
    [ -n "${home}" ] || exit 91
    mkdir -p "${home}/config" "${home}/data"
    cat > "${home}/config/config.toml" <<'CONFIG'
[rpc]
laddr = "tcp://127.0.0.1:26657"
unsafe = false
cors_allowed_origins = []
max_request_batch_size = 0
max_body_bytes = 1000000
max_header_bytes = 1048576
[storage]
discard_abci_responses = true
[tx_index]
indexer = "null"
[p2p]
laddr = "tcp://0.0.0.0:26656"
seeds = ""
persistent_peers = ""
private_peer_ids = ""
pex = true
addr_book_strict = true
allow_duplicate_ip = false
external_address = ""
max_num_inbound_peers = 40
max_num_outbound_peers = 10
[mempool]
broadcast = true
wal_dir = "data/mempool.wal"
size = 5000
max_txs_bytes = 1073741824
CONFIG
    cat > "${home}/config/app.toml" <<'APP'
minimum-gas-prices = ""
query-gas-limit = "0"
[api]
enable = true
address = "tcp://localhost:1317"
enabled-unsafe-cors = false
rpc-read-timeout = 0
rpc-write-timeout = 0
rpc-max-body-bytes = 1000000
[grpc]
enable = true
address = "localhost:9090"
[mempool]
max-txs = 5000
APP
    printf '%s\n' 'chain-id = "zerone-1"' > "${home}/config/client.toml"
    if [ "${NODE_ROLE:-signer}" = "observer" ]; then
      cp "${OBSERVER_NODE_KEY_FILE:?}" "${home}/config/node_key.json"
      cp "${OBSERVER_VALIDATOR_KEY_FILE:?}" "${home}/config/priv_validator_key.json"
    else
      printf '%s\n' '{"generated":"must be overwritten"}' \
        > "${home}/config/node_key.json"
      printf '%s\n' '{"generated":"must be overwritten"}' \
        > "${home}/config/priv_validator_key.json"
    fi
    printf '%s\n' '{"height":"0","round":0,"step":0}' \
      > "${home}/data/priv_validator_state.json"
    ;;
  tendermint)
    subcommand="${2:-}"
    shift 2
    home=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--home" ]; then
        home="$2"
        shift 2
      else
        shift
      fi
    done
    [ -n "${home}" ] || exit 95
    case "${subcommand}" in
      show-validator)
        [ -f "${home}/data/priv_validator_state.json" ] || exit 97
        "$(dirname "$0")/keytool" validator \
          "${home}/config/priv_validator_key.json"
        ;;
      show-node-id)
        "$(dirname "$0")/keytool" node "${home}/config/node_key.json"
        ;;
      *) exit 96 ;;
    esac
    ;;
  start)
    [ -z "${PRIV_VALIDATOR_KEY_B64:-}" ] || exit 93
    [ -z "${NODE_KEY_B64:-}" ] || exit 94
    printf '%s\n' "$*" > "${START_MARKER:?START_MARKER is required}"
    if [ -n "${START_HOLD_READY:-}" ]; then
      : > "${START_HOLD_READY}"
      while [ ! -e "${START_HOLD_RELEASE:?}" ]; do
        sleep 0.05
      done
    fi
    ;;
  *)
    exit 92
    ;;
esac
STUB
chmod +x "${TMP}/bin/zeroned"

cat > "${TMP}/bin/curl" <<'CURL_STUB'
#!/usr/bin/env bash
set -euo pipefail
url="${!#}"
anchor_hash="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
app_hash="BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
case "${url}" in
  */status)
    printf '%s\n' "{\"result\":{\"node_info\":{\"network\":\"zerone-1\"},\"sync_info\":{\"latest_block_height\":\"424241\",\"latest_block_hash\":\"${anchor_hash}\",\"catching_up\":true}}}"
    ;;
  */abci_info)
    printf '%s\n' "{\"result\":{\"response\":{\"last_block_height\":\"424241\",\"last_block_app_hash\":\"${app_hash}\"}}}"
    ;;
  *'/block?height=424241')
    printf '%s\n' "{\"result\":{\"block_id\":{\"hash\":\"${anchor_hash}\"},\"block\":{\"header\":{\"chain_id\":\"zerone-1\",\"height\":\"424241\"},\"data\":{\"txs\":[]}}}}"
    ;;
  *'/commit?height=424241')
    printf '%s\n' "{\"result\":{\"canonical\":false,\"signed_header\":{\"header\":{\"height\":\"424241\"},\"commit\":{\"height\":\"424241\",\"block_id\":{\"hash\":\"${anchor_hash}\"}}}}}"
    ;;
  *'/block?height=424242'|*'/block_results?height=424242')
    printf '%s\n' '{"jsonrpc":"2.0","error":{"code":-32603,"message":"height unavailable"},"id":-1}'
    ;;
  *)
    printf 'unexpected fake RPC URL: %s\n' "${url}" >&2
    exit 22
    ;;
esac
CURL_STUB
chmod +x "${TMP}/bin/curl"
export CHILD_SECRET_LEAK_FILE="${TMP}/child-secret-leak"

NODE_B64=$(base64 < "${TMP}/node_key.json" | tr -d '\n')
FOREIGN_NODE_B64=$(base64 < "${TMP}/foreign_node_key.json" | tr -d '\n')
VALIDATOR_B64=$(base64 < "${TMP}/validator_key.json" | tr -d '\n')
MISMATCHED_VALIDATOR_B64=$(base64 < "${TMP}/mismatched_validator_key.json" | tr -d '\n')
FOREIGN_VALIDATOR_B64=$(base64 < "${TMP}/foreign_validator_key.json" | tr -d '\n')
if command -v sha256sum >/dev/null 2>&1; then
  GENESIS_SHA256=$(sha256sum "${TMP}/seed/genesis.json" | awk '{print $1}')
  PRODUCTION_GENESIS_SHA256=$(sha256sum \
    "${ROOT}/deploy/mainnet/artifacts/genesis.json" | awk '{print $1}')
else
  GENESIS_SHA256=$(openssl dgst -sha256 "${TMP}/seed/genesis.json" | awk '{print $NF}')
  PRODUCTION_GENESIS_SHA256=$(openssl dgst -sha256 \
    "${ROOT}/deploy/mainnet/artifacts/genesis.json" | awk '{print $NF}')
fi
EXPECTED_PRODUCTION_SHA256="c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e"
EXPECTED_PRODUCTION_NODE_ID="ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e"
FIXTURE_NODE_ID=$("${TMP}/bin/keytool" node "${TMP}/node_key.json")
[ "${PRODUCTION_GENESIS_SHA256}" = "${EXPECTED_PRODUCTION_SHA256}" ] || \
  fail "committed mainnet genesis drifted from the production hash pin"
grep -q "EXPECTED_GENESIS_SHA256=\"${EXPECTED_PRODUCTION_SHA256}\"" \
  "${PRODUCTION_ENTRYPOINT}" || fail "entrypoint hash pin drifted from mainnet genesis"
grep -q "EXPECTED_NODE_ID=\"${EXPECTED_PRODUCTION_NODE_ID}\"" \
  "${PRODUCTION_ENTRYPOINT}" || fail "entrypoint node-id pin drifted from manifest"
grep -q '^readonly BINARY="/usr/local/bin/zeroned"$' \
  "${PRODUCTION_ENTRYPOINT}" || fail "production entrypoint binary path is not immutable"
if grep -q 'ZERONED_BINARY' "${PRODUCTION_ENTRYPOINT}"; then
  fail "production entrypoint accepts a zeroned binary override"
fi

# The production pin is intentionally immutable. Exercise the same entrypoint
# against the generated cryptographic fixture by replacing only that literal in
# an isolated temporary copy.
ENTRYPOINT="${TMP}/entrypoint.sh"
sed -e "s/${EXPECTED_PRODUCTION_SHA256}/${GENESIS_SHA256}/" \
  -e "s/${EXPECTED_PRODUCTION_NODE_ID}/${FIXTURE_NODE_ID}/" \
  -e "s|readonly BINARY=\"/usr/local/bin/zeroned\"|readonly BINARY=\"${TMP}/bin/zeroned\"|" \
  "${PRODUCTION_ENTRYPOINT}" > "${ENTRYPOINT}"
chmod +x "${ENTRYPOINT}"

# A fresh volume must fail closed before zeroned creates fallback keys.
MISSING_HOME="${TMP}/missing-home"
if ZERONE_HOME="${MISSING_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  "${ENTRYPOINT}" > "${TMP}/missing.log" 2>&1; then
  fail "fresh volume started without secrets"
fi
grep -q 'requires PRIV_VALIDATOR_KEY_B64' "${TMP}/missing.log" || \
  fail "missing-secret error was not explicit"
[ ! -e "${MISSING_HOME}/config/genesis.json" ] || \
  fail "fresh-volume failure left a seeded genesis behind"

# A valid but different P2P key would change the published seed node ID.
FOREIGN_NODE_HOME="${TMP}/foreign-node-home"
if ZERONE_HOME="${FOREIGN_NODE_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  NODE_KEY_B64="${FOREIGN_NODE_B64}" \
  PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/foreign-node.log" 2>&1; then
  fail "fresh volume accepted a different P2P identity"
fi
grep -q 'expected mainnet node' "${TMP}/foreign-node.log" || \
  fail "foreign-node error was not explicit"
[ ! -e "${FOREIGN_NODE_HOME}/config/genesis.json" ] || \
  fail "foreign-node failure initialized the volume"

# A validator key that does not belong to the genesis must fail before init.
WRONG_HOME="${TMP}/wrong-home"
if ZERONE_HOME="${WRONG_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  NODE_KEY_B64="${NODE_B64}" \
  PRIV_VALIDATOR_KEY_B64="${MISMATCHED_VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/wrong.log" 2>&1; then
  fail "fresh volume accepted a validator key from another genesis"
fi
grep -q 'public key does not match its private key' "${TMP}/wrong.log" || \
  fail "wrong-validator error was not explicit"
[ ! -e "${WRONG_HOME}/config/genesis.json" ] || \
  fail "wrong-validator failure initialized the volume"

# A self-consistent key from a different validator must also be rejected.
FOREIGN_HOME="${TMP}/foreign-home"
if ZERONE_HOME="${FOREIGN_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  NODE_KEY_B64="${NODE_B64}" \
  PRIV_VALIDATOR_KEY_B64="${FOREIGN_VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/foreign.log" 2>&1; then
  fail "fresh volume accepted a validator outside the genesis"
fi
grep -q 'does not match any validator' "${TMP}/foreign.log" || \
  fail "foreign-validator error was not explicit"
[ ! -e "${FOREIGN_HOME}/config/genesis.json" ] || \
  fail "foreign-validator failure initialized the volume"

# Valid secrets must replace zeroned's generated keys and reach start.
HOME_DIR="${TMP}/home"
START_MARKER="${TMP}/started"
if ! ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  NODE_KEY_B64="${NODE_B64}" PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/valid.log" 2>&1; then
  sed -n '1,80p' "${TMP}/valid.log" >&2
  fail "valid secrets did not start the node"
fi
cmp -s "${TMP}/node_key.json" "${HOME_DIR}/config/node_key.json" || \
  fail "node key was not restored exactly"
cmp -s "${TMP}/validator_key.json" \
  "${HOME_DIR}/config/priv_validator_key.json" || \
  fail "validator key was not restored exactly"
[ "$(file_mode "${HOME_DIR}/config/node_key.json")" = "600" ] || \
  fail "node key mode is not 0600"
[ "$(file_mode "${HOME_DIR}/config/priv_validator_key.json")" = "600" ] || \
  fail "validator key mode is not 0600"
[ "$(file_mode "${HOME_DIR}/data/priv_validator_state.json")" = "600" ] || \
  fail "validator state mode is not 0600"
[ -f "${START_MARKER}" ] || fail "zeroned start was not reached"
grep -q -- '--home' "${START_MARKER}" || fail "start command lost --home"
grep -q -- '--query-gas-limit 5000000' "${START_MARKER}" || \
  fail "start command did not pin SDK query gas"
grep -q -- '--min-retain-blocks 0' "${START_MARKER}" || \
  fail "start command did not retain Comet history"
grep -q 'minimum-gas-prices = "0.025uzrn"' \
  "${HOME_DIR}/config/app.toml" || fail "minimum gas price was not hardened"
assert_bounded_query_config "${HOME_DIR}/config/app.toml" \
  "${HOME_DIR}/config/config.toml" "initial signer config"
[ ! -e "${CHILD_SECRET_LEAK_FILE}" ] || \
  fail "bootstrap custody reached a zeroned child environment"
grep -q '^role=signer$' "${HOME_DIR}/.zerone-1-runtime" || \
  fail "fresh signer volume was not permanently role-marked"
grep -q "^node_id=${FIXTURE_NODE_ID}$" "${HOME_DIR}/.zerone-1-runtime" || \
  fail "fresh signer marker did not pin its P2P identity"
SIGNER_VALIDATOR_PUB=$("${TMP}/bin/keytool" validator \
  "${TMP}/validator_key.json" | jq -r '.key')
grep -q "^validator_pubkey=${SIGNER_VALIDATOR_PUB}$" \
  "${HOME_DIR}/.zerone-1-runtime" || \
  fail "fresh signer marker did not pin its consensus identity"

# A resumed persistent volume must not require the bootstrap secrets again.
drift_bounded_query_config "${HOME_DIR}/config/app.toml" \
  "${HOME_DIR}/config/config.toml"
rm -f "${START_MARKER}"
if ! ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  "${ENTRYPOINT}" > "${TMP}/resume.log" 2>&1; then
  sed -n '1,80p' "${TMP}/resume.log" >&2
  fail "valid persistent volume failed to resume"
fi
[ -f "${START_MARKER}" ] || fail "valid persistent volume did not resume"
assert_bounded_query_config "${HOME_DIR}/config/app.toml" \
  "${HOME_DIR}/config/config.toml" "signer restart repair"

# Viper-style daemon environment variables must not bypass the settings the
# role profile reasserts on disk and on the command line.
rm -f "${START_MARKER}"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" ZERONED_RPC_UNSAFE=true \
  ZERONED_QUERY_GAS_LIMIT=0 "${ENTRYPOINT}" \
  > "${TMP}/daemon-env-override.log" 2>&1; then
  fail "runtime accepted a daemon configuration environment override"
fi
grep -q 'daemon configuration environment override is forbidden' \
  "${TMP}/daemon-env-override.log" || \
  fail "daemon environment override rejection was not explicit"
[ ! -e "${START_MARKER}" ] || \
  fail "daemon environment override reached zeroned start"

# Removing EXTERNAL_ADDRESS in the private halt profile must erase a stale
# public advertisement from the persistent config on the very next boot.
toml_test_set p2p external_address '"stale-public.example:26656"' \
  "${HOME_DIR}/config/config.toml"
rm -f "${START_MARKER}"
if ! ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" "${ENTRYPOINT}" \
  > "${TMP}/private-signer-resume.log" 2>&1; then
  fail "private signer profile did not resume"
fi
grep -A14 '^\[p2p\]' "${HOME_DIR}/config/config.toml" | \
  grep -q '^external_address = ""$' || \
  fail "private signer retained a stale public external address"

# Bootstrap secrets are one-time only. Retaining either platform secret after
# initialization must fail before it can reach any helper or daemon child.
rm -f "${START_MARKER}"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/retained-secret.log" 2>&1; then
  fail "initialized signer accepted a retained bootstrap secret"
fi
grep -q 'bootstrap custody inputs are forbidden' "${TMP}/retained-secret.log" || \
  fail "retained-secret error was not explicit"
[ ! -e "${CHILD_SECRET_LEAK_FILE}" ] || \
  fail "retained bootstrap custody reached a zeroned child"

# A malformed external address must never be interpolated into TOML or sed.
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" EXTERNAL_ADDRESS='bad"|address:26656' \
  "${ENTRYPOINT}" > "${TMP}/bad-external.log" 2>&1; then
  fail "signer accepted an unsafe external address"
fi
grep -Eq 'unsafe TOML|must use a DNS name' "${TMP}/bad-external.log" || \
  fail "unsafe external-address error was not explicit"

# One home may be opened by only one process. The lock FD must survive the
# daemon exec, so a concurrent owner is rejected before key access.
LOCK_READY="${TMP}/lock-ready"
LOCK_RELEASE="${TMP}/lock-release"
ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" START_HOLD_READY="${LOCK_READY}" \
  START_HOLD_RELEASE="${LOCK_RELEASE}" "${ENTRYPOINT}" \
  > "${TMP}/lock-owner.log" 2>&1 &
LOCK_PID=$!
for _ in $(seq 1 100); do
  [ -e "${LOCK_READY}" ] && break
  kill -0 "${LOCK_PID}" 2>/dev/null || fail "lock owner exited early"
  sleep 0.02
done
[ -e "${LOCK_READY}" ] || fail "lock owner never reached start"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${TMP}/lock-contender-start" "${ENTRYPOINT}" \
  > "${TMP}/lock-contender.log" 2>&1; then
  fail "concurrent process opened the signer home"
fi
grep -q 'another process already owns' "${TMP}/lock-contender.log" || \
  fail "concurrent-home error was not explicit"
: > "${LOCK_RELEASE}"
wait "${LOCK_PID}" || fail "lock owner did not exit cleanly"

# A partial home and symlinked custody are both fail-closed.
PARTIAL_HOME="${TMP}/partial-home"
mkdir -p "${PARTIAL_HOME}"
printf 'partial\n' > "${PARTIAL_HOME}/stale"
if ZERONE_HOME="${PARTIAL_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${TMP}/partial-start" PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
  NODE_KEY_B64="${NODE_B64}" "${ENTRYPOINT}" > "${TMP}/partial.log" 2>&1; then
  fail "partial non-empty signer home was initialized"
fi
grep -q 'partially initialized' "${TMP}/partial.log" || \
  fail "partial-home error was not explicit"

LOCK_SYMLINK_HOME="${TMP}/lock-symlink-home"
LOCK_SENTINEL="${TMP}/lock-sentinel"
printf 'lock sentinel\n' > "${LOCK_SENTINEL}"
chmod 0644 "${LOCK_SENTINEL}"
LOCK_SENTINEL_SHA=$(sha256_file "${LOCK_SENTINEL}")
ln -s "${LOCK_SENTINEL}" "${LOCK_SYMLINK_HOME}.runtime.lock"
if ZERONE_HOME="${LOCK_SYMLINK_HOME}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${TMP}/lock-symlink-start" NODE_KEY_B64="${NODE_B64}" \
  PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" "${ENTRYPOINT}" \
  > "${TMP}/lock-symlink.log" 2>&1; then
  fail "runtime accepted a preplanted lock symlink"
fi
grep -q 'runtime lock must not be a symlink' "${TMP}/lock-symlink.log" || \
  fail "lock-symlink error was not explicit"
[ "$(sha256_file "${LOCK_SENTINEL}")" = "${LOCK_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${LOCK_SENTINEL}")" = "644" ] || \
  fail "lock symlink target was modified before rejection"

mv "${HOME_DIR}/config/node_key.json" "${HOME_DIR}/config/node_key.real.json"
chmod 0644 "${HOME_DIR}/config/node_key.real.json"
SYMLINK_KEY_SHA=$(sha256_file "${HOME_DIR}/config/node_key.real.json")
ln -s node_key.real.json "${HOME_DIR}/config/node_key.json"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" "${ENTRYPOINT}" > "${TMP}/symlink.log" 2>&1; then
  fail "resumed signer accepted a symlinked node key"
fi
grep -q 'regular, non-symlink' "${TMP}/symlink.log" || \
  fail "symlinked-key error was not explicit"
[ "$(sha256_file "${HOME_DIR}/config/node_key.real.json")" = "${SYMLINK_KEY_SHA}" ] && \
  [ "$(file_mode "${HOME_DIR}/config/node_key.real.json")" = "644" ] || \
  fail "symlinked key target was mutated before rejection"
rm "${HOME_DIR}/config/node_key.json"
mv "${HOME_DIR}/config/node_key.real.json" "${HOME_DIR}/config/node_key.json"
chmod 0600 "${HOME_DIR}/config/node_key.json"

mv "${HOME_DIR}/.zerone-1-runtime" "${TMP}/runtime-marker.saved"
MARKER_SENTINEL="${TMP}/runtime-marker-sentinel"
printf 'marker sentinel\n' > "${MARKER_SENTINEL}"
chmod 0644 "${MARKER_SENTINEL}"
MARKER_SENTINEL_SHA=$(sha256_file "${MARKER_SENTINEL}")
ln -s "${MARKER_SENTINEL}" "${HOME_DIR}/.zerone-1-runtime"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" "${ENTRYPOINT}" \
  > "${TMP}/marker-symlink.log" 2>&1; then
  fail "runtime accepted a symlinked role marker"
fi
grep -q 'runtime marker must not be a symlink' "${TMP}/marker-symlink.log" || \
  fail "runtime-marker symlink error was not explicit"
[ "$(sha256_file "${MARKER_SENTINEL}")" = "${MARKER_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${MARKER_SENTINEL}")" = "644" ] || \
  fail "runtime-marker symlink target was modified"
rm "${HOME_DIR}/.zerone-1-runtime"
mv "${TMP}/runtime-marker.saved" "${HOME_DIR}/.zerone-1-runtime"

CHECKPOINT_SENTINEL="${TMP}/checkpoint-marker-sentinel"
printf 'checkpoint sentinel\n' > "${CHECKPOINT_SENTINEL}"
chmod 0644 "${CHECKPOINT_SENTINEL}"
CHECKPOINT_SENTINEL_SHA=$(sha256_file "${CHECKPOINT_SENTINEL}")
ln -s "${CHECKPOINT_SENTINEL}" "${HOME_DIR}/.zerone-1-checkpoint-plan"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" "${ENTRYPOINT}" \
  > "${TMP}/checkpoint-marker-symlink.log" 2>&1; then
  fail "runtime accepted a symlinked checkpoint marker"
fi
grep -q 'checkpoint plan marker must not be a symlink' \
  "${TMP}/checkpoint-marker-symlink.log" || \
  fail "checkpoint-marker symlink error was not explicit"
[ "$(sha256_file "${CHECKPOINT_SENTINEL}")" = "${CHECKPOINT_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${CHECKPOINT_SENTINEL}")" = "644" ] || \
  fail "checkpoint-marker symlink target was modified"
rm "${HOME_DIR}/.zerone-1-checkpoint-plan"

# The independent observer is a real deployable role: it generates fresh
# non-validator identities, dials only the official signer, has no custody
# inputs, and permanently freezes transaction relay.
OBSERVER_HOME="${TMP}/observer-home"
OBSERVER_START="${TMP}/observer-start"
OBSERVER_PEER="${FIXTURE_NODE_ID}@127.0.0.1:26656"
if ! NODE_ROLE=observer ZERONE_HOME="${OBSERVER_HOME}" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${OBSERVER_START}" \
  PERSISTENT_PEERS="${OBSERVER_PEER}" \
  OBSERVER_NODE_KEY_FILE="${TMP}/foreign_node_key.json" \
  OBSERVER_VALIDATOR_KEY_FILE="${TMP}/foreign_validator_key.json" \
  "${ENTRYPOINT}" > "${TMP}/observer.log" 2>&1; then
  sed -n '1,100p' "${TMP}/observer.log" >&2
  fail "fresh observer role did not start"
fi
cmp -s "${TMP}/foreign_node_key.json" "${OBSERVER_HOME}/config/node_key.json" || \
  fail "observer did not retain its fresh P2P key"
cmp -s "${TMP}/foreign_validator_key.json" \
  "${OBSERVER_HOME}/config/priv_validator_key.json" || \
  fail "observer did not retain its fresh non-validator consensus key"
grep -q '^role=observer$' "${OBSERVER_HOME}/.zerone-1-runtime" || \
  fail "observer volume was not role-marked"
OBSERVER_NODE_ID=$("${TMP}/bin/keytool" node "${TMP}/foreign_node_key.json")
OBSERVER_VALIDATOR_PUB=$("${TMP}/bin/keytool" validator \
  "${TMP}/foreign_validator_key.json" | jq -r '.key')
grep -q "^node_id=${OBSERVER_NODE_ID}$" \
  "${OBSERVER_HOME}/.zerone-1-runtime" || \
  fail "observer marker did not pin its P2P identity"
grep -q "^validator_pubkey=${OBSERVER_VALIDATOR_PUB}$" \
  "${OBSERVER_HOME}/.zerone-1-runtime" || \
  fail "observer marker did not pin its consensus identity"
grep -A8 '^\[grpc\]' "${OBSERVER_HOME}/config/app.toml" | grep -q '^enable = false$' || \
  fail "observer gRPC transaction surface was not closed"
grep -A30 '^\[rpc\]' "${OBSERVER_HOME}/config/config.toml" | \
  grep -q '^laddr = "tcp://fly-local-6pn:26657"$' || \
  fail "observer RPC does not bind its private Fly 6PN interface"
grep -A8 '^\[mempool\]' "${OBSERVER_HOME}/config/config.toml" | grep -q '^size = 0$' || \
  fail "observer Comet mempool was not frozen"
grep -A12 '^\[p2p\]' "${OBSERVER_HOME}/config/config.toml" | \
  grep -q "^persistent_peers = \"${OBSERVER_PEER}\"$" || \
  fail "observer was not pinned to the official signer peer"

if NODE_ROLE=observer ZERONE_HOME="${TMP}/observer-with-secret" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${TMP}/observer-secret-start" \
  PERSISTENT_PEERS="${OBSERVER_PEER}" PRIV_VALIDATOR_KEY_B64="${VALIDATOR_B64}" \
  "${ENTRYPOINT}" > "${TMP}/observer-secret.log" 2>&1; then
  fail "observer accepted validator bootstrap custody"
fi
grep -q 'non-signer role rejects' "${TMP}/observer-secret.log" || \
  fail "observer-custody error was not explicit"

if NODE_ROLE=observer ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${TMP}/role-change-start" PERSISTENT_PEERS="${OBSERVER_PEER}" \
  "${ENTRYPOINT}" > "${TMP}/role-change.log" 2>&1; then
  fail "signer volume changed into an observer"
fi
grep -Eq 'non-signer role must not reuse|non-signer role unexpectedly holds|volume role does not match' \
  "${TMP}/role-change.log" || \
  fail "signer-to-observer role-change error was not explicit"

# A non-validator identity may not be silently rotated after the volume has
# been marked. The key remains non-genesis, so only the identity pin catches it.
cp "${OBSERVER_HOME}/config/node_key.json" "${TMP}/observer-node-key.saved"
cp "${TMP}/rotated_node_key.json" "${OBSERVER_HOME}/config/node_key.json"
if NODE_ROLE=observer ZERONE_HOME="${OBSERVER_HOME}" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${OBSERVER_START}" \
  PERSISTENT_PEERS="${OBSERVER_PEER}" "${ENTRYPOINT}" \
  > "${TMP}/observer-rotated-key.log" 2>&1; then
  fail "observer accepted a silently rotated P2P identity"
fi
grep -q 'runtime node identity changed' "${TMP}/observer-rotated-key.log" || \
  fail "observer identity-pin error was not explicit"
mv "${TMP}/observer-node-key.saved" "${OBSERVER_HOME}/config/node_key.json"

# Archive is a one-way, two-stage transition from an allowlisted, rolled-back,
# fresh-key serving copy. Direct observer-to-archive conversion is forbidden.
if NODE_ROLE=archive ZERONE_HOME="${OBSERVER_HOME}" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${TMP}/archive-too-early-start" \
  "${ENTRYPOINT}" > "${TMP}/archive-too-early.log" 2>&1; then
  fail "observer transitioned to archive without persisted F/A/H"
fi
grep -q 'requires the explicit F/A/H' "${TMP}/archive-too-early.log" || \
  fail "early archive-transition error was not explicit"

OBSERVER_CHECKPOINT_START="${TMP}/observer-checkpoint-start"
if ! NODE_ROLE=observer ZERONE_HOME="${OBSERVER_HOME}" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${OBSERVER_CHECKPOINT_START}" \
  PERSISTENT_PEERS="${OBSERVER_PEER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/observer-checkpoint.log" 2>&1; then
  sed -n '1,100p' "${TMP}/observer-checkpoint.log" >&2
  fail "observer could not persist its F/A/H plan"
fi

SIGNER_EVIDENCE_SHA256="cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
OBSERVER_EVIDENCE_SHA256="dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
SOURCE_OBSERVER_MARKER_SHA256=$(sha256_file "${OBSERVER_HOME}/.zerone-1-runtime")
EXPECTED_ANCHOR_HASH="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
EXPECTED_APP_HASH="BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
ARCHIVE_NONCE="eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
ARCHIVE_NODE_ID=$("${TMP}/bin/keytool" node "${TMP}/rotated_node_key.json")
ARCHIVE_VALIDATOR_PUB=$("${TMP}/bin/keytool" validator \
  "${TMP}/rotated_validator_key.json" | jq -r '.key')
[ "${ARCHIVE_NODE_ID}" != "${OBSERVER_NODE_ID}" ] || \
  fail "archive fixture reused the source observer P2P identity"
[ "${ARCHIVE_VALIDATOR_PUB}" != "${OBSERVER_VALIDATOR_PUB}" ] || \
  fail "archive fixture reused the source observer consensus identity"

# Reproduce the cutover allowlist: public configuration, fresh non-validator
# keys/state, and only the reviewed Comet/application databases. No observer
# marker, checkpoint marker, WAL, keyring, or incidental home content crosses.
ARCHIVE_TEMPLATE_HOME="${TMP}/archive-template-home"
mkdir -p "${ARCHIVE_TEMPLATE_HOME}/config" "${ARCHIVE_TEMPLATE_HOME}/data"
for name in genesis.json config.toml app.toml client.toml; do
  cp "${OBSERVER_HOME}/config/${name}" "${ARCHIVE_TEMPLATE_HOME}/config/${name}"
done
cp "${TMP}/rotated_node_key.json" \
  "${ARCHIVE_TEMPLATE_HOME}/config/node_key.json"
cp "${TMP}/rotated_validator_key.json" \
  "${ARCHIVE_TEMPLATE_HOME}/config/priv_validator_key.json"
printf '%s\n' '{"height":"0","round":0,"step":0}' \
  > "${ARCHIVE_TEMPLATE_HOME}/data/priv_validator_state.json"
mkdir -p "${ARCHIVE_TEMPLATE_HOME}/data/application.db" \
  "${ARCHIVE_TEMPLATE_HOME}/data/blockstore.db" \
  "${ARCHIVE_TEMPLATE_HOME}/data/state.db"
chmod 0600 "${ARCHIVE_TEMPLATE_HOME}/config/node_key.json" \
  "${ARCHIVE_TEMPLATE_HOME}/config/priv_validator_key.json" \
  "${ARCHIVE_TEMPLATE_HOME}/data/priv_validator_state.json"
jq -n \
  --arg genesis "${GENESIS_SHA256}" \
  --arg source_marker "${SOURCE_OBSERVER_MARKER_SHA256}" \
  --arg source_node "${OBSERVER_NODE_ID}" \
  --arg source_validator "${OBSERVER_VALIDATOR_PUB}" \
  --arg candidate_node "${ARCHIVE_NODE_ID}" \
  --arg candidate_validator "${ARCHIVE_VALIDATOR_PUB}" \
  --arg anchor_hash "${EXPECTED_ANCHOR_HASH}" \
  --arg app_hash "${EXPECTED_APP_HASH}" \
  --arg signer_evidence "${SIGNER_EVIDENCE_SHA256}" \
  --arg observer_evidence "${OBSERVER_EVIDENCE_SHA256}" \
  --arg nonce "${ARCHIVE_NONCE}" '{
    schema: "zerone-1-archive-transition-v1",
    chain_id: "zerone-1",
    checkpoint_state_height: "424240",
    final_committed_height: "424241",
    halt_trigger_height: "424242",
    genesis_sha256: $genesis,
    cutover_initiation_evidence: {
      successor_transaction_hash: ("A" * 64),
      committed_height: "424200",
      committed_block_time: "2026-07-12T12:00:00Z",
      public_notice_sha256: ("c" * 64),
      public_notice_publication_evidence_sha256: ("d" * 64),
      initiation_evidence_sha256: ("e" * 64),
      initiation_evidence_detached_signature_sha256: ("f" * 64)
    },
    source_observer: {
      runtime_marker_sha256: $source_marker,
      node_id: $source_node,
      validator_pubkey: $source_validator
    },
    candidate: {
      node_id: $candidate_node,
      validator_pubkey: $candidate_validator
    },
    expected_anchor_block_hash: $anchor_hash,
    expected_post_anchor_app_hash: $app_hash,
    source_evidence: {
      signer_manifest_sha256: $signer_evidence,
      observer_manifest_sha256: $observer_evidence
    },
    archive_construction_evidence: {
      pre_transition_sanitized_snapshot_sha256: $signer_evidence,
      rollback_log_sha256: $observer_evidence,
      pre_transition_allowlist_manifest_sha256: $source_marker,
      excluded_future_artifacts: [
        "archive transition manifest", "rendered Fly configs",
        "archive adoption authority", "archive readiness",
        "final checkpoint", "open-beta decision"
      ]
    },
    archive_transition_nonce: $nonce
  }' > "${ARCHIVE_TEMPLATE_HOME}/.zerone-1-archive-transition.json"
chmod 0600 "${ARCHIVE_TEMPLATE_HOME}/.zerone-1-archive-transition.json"
ARCHIVE_TRANSITION_SHA256=$(sha256_file \
  "${ARCHIVE_TEMPLATE_HOME}/.zerone-1-archive-transition.json")

archive_runtime() {
  local role="$1" home="$2" start_marker="$3"
  shift 3
  env NODE_ROLE="${role}" ZERONE_HOME="${home}" \
    MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${start_marker}" \
    ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
    ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
    ZERONE_SIGNER_EVIDENCE_MANIFEST_SHA256="${SIGNER_EVIDENCE_SHA256}" \
    ZERONE_OBSERVER_EVIDENCE_MANIFEST_SHA256="${OBSERVER_EVIDENCE_SHA256}" \
    ZERONE_SOURCE_OBSERVER_RUNTIME_MARKER_SHA256="${SOURCE_OBSERVER_MARKER_SHA256}" \
    ZERONE_SOURCE_OBSERVER_NODE_ID="${OBSERVER_NODE_ID}" \
    ZERONE_SOURCE_OBSERVER_VALIDATOR_PUBKEY="${OBSERVER_VALIDATOR_PUB}" \
    ZERONE_EXPECTED_ANCHOR_BLOCK_HASH="${EXPECTED_ANCHOR_HASH}" \
    ZERONE_EXPECTED_POST_ANCHOR_APP_HASH="${EXPECTED_APP_HASH}" \
    ZERONE_ARCHIVE_TRANSITION_MANIFEST_SHA256="${ARCHIVE_TRANSITION_SHA256}" \
    "$@" "${ENTRYPOINT}"
}

# A checkpoint-armed observer marker still cannot become an archive, even when
# given the final evidence inputs. The serving copy must have fresh identities.
if archive_runtime archive "${OBSERVER_HOME}" \
  "${TMP}/archive-observer-direct-start" \
  > "${TMP}/archive-observer-direct.log" 2>&1; then
  fail "checkpoint observer changed directly into archive"
fi
grep -q 'volume role does not match NODE_ROLE' \
  "${TMP}/archive-observer-direct.log" || \
  fail "direct observer-to-archive error was not explicit"

ARCHIVE_HOME="${TMP}/archive-home"
cp -R "${ARCHIVE_TEMPLATE_HOME}" "${ARCHIVE_HOME}"

# Even an allowlisted fresh-key volume cannot bypass candidate adoption and the
# local A/A RPC proof by starting directly as the final archive role.
if archive_runtime archive "${ARCHIVE_HOME}" \
  "${TMP}/archive-premature-start" > "${TMP}/archive-premature.log" 2>&1; then
  fail "sanitized volume bypassed archive-candidate readiness proof"
fi
grep -q 'candidate-persisted F/A/H' "${TMP}/archive-premature.log" || \
  fail "premature archive-transition error was not explicit"
[ ! -e "${ARCHIVE_HOME}/.zerone-1-runtime" ] || \
  fail "premature transition created a runtime marker"

# A preplanted readiness symlink is rejected before a candidate nonce or marker
# is written, and its target is never opened or mutated.
READINESS_SYMLINK_HOME="${TMP}/archive-readiness-symlink-home"
cp -R "${ARCHIVE_TEMPLATE_HOME}" "${READINESS_SYMLINK_HOME}"
READINESS_SENTINEL="${TMP}/archive-readiness-sentinel"
printf 'archive readiness sentinel\n' > "${READINESS_SENTINEL}"
chmod 0644 "${READINESS_SENTINEL}"
READINESS_SENTINEL_SHA=$(sha256_file "${READINESS_SENTINEL}")
ln -s "${READINESS_SENTINEL}" \
  "${READINESS_SYMLINK_HOME}/.zerone-1-archive-readiness.json"
if archive_runtime archive-candidate "${READINESS_SYMLINK_HOME}" \
  "${TMP}/readiness-symlink-start" > "${TMP}/readiness-symlink.log" 2>&1; then
  fail "archive candidate accepted a preplanted readiness symlink"
fi
grep -q 'stale archive readiness; refusing replay' \
  "${TMP}/readiness-symlink.log" || \
  fail "readiness-symlink error was not explicit"
[ "$(sha256_file "${READINESS_SENTINEL}")" = "${READINESS_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${READINESS_SENTINEL}")" = "644" ] || \
  fail "archive readiness symlink target was mutated"

# Immediate-child allowlists are insufficient for database trees: nested
# symlinks and hardlinks could otherwise smuggle custody into a path opened by
# the DB backend. Both forms fail before candidate adoption and preserve their
# outside sentinels byte-for-byte.
DB_SYMLINK_HOME="${TMP}/archive-db-symlink-home"
cp -R "${ARCHIVE_TEMPLATE_HOME}" "${DB_SYMLINK_HOME}"
DB_SYMLINK_SENTINEL="${TMP}/archive-db-symlink-sentinel"
printf 'database symlink sentinel\n' > "${DB_SYMLINK_SENTINEL}"
chmod 0644 "${DB_SYMLINK_SENTINEL}"
DB_SYMLINK_SENTINEL_SHA=$(sha256_file "${DB_SYMLINK_SENTINEL}")
ln -s "${DB_SYMLINK_SENTINEL}" \
  "${DB_SYMLINK_HOME}/data/application.db/000001.ldb"
if archive_runtime archive-candidate "${DB_SYMLINK_HOME}" \
  "${TMP}/archive-db-symlink-start" > "${TMP}/archive-db-symlink.log" 2>&1; then
  fail "archive candidate accepted a nested database symlink"
fi
grep -q 'symlink or non-file node' "${TMP}/archive-db-symlink.log" || \
  fail "nested database symlink error was not explicit"
[ "$(sha256_file "${DB_SYMLINK_SENTINEL}")" = "${DB_SYMLINK_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${DB_SYMLINK_SENTINEL}")" = "644" ] || \
  fail "nested database symlink target was mutated"

DB_HARDLINK_HOME="${TMP}/archive-db-hardlink-home"
cp -R "${ARCHIVE_TEMPLATE_HOME}" "${DB_HARDLINK_HOME}"
DB_HARDLINK_SENTINEL="${TMP}/archive-db-hardlink-sentinel"
printf 'database hardlink sentinel\n' > "${DB_HARDLINK_SENTINEL}"
chmod 0644 "${DB_HARDLINK_SENTINEL}"
DB_HARDLINK_SENTINEL_SHA=$(sha256_file "${DB_HARDLINK_SENTINEL}")
ln "${DB_HARDLINK_SENTINEL}" \
  "${DB_HARDLINK_HOME}/data/state.db/000002.ldb"
if archive_runtime archive-candidate "${DB_HARDLINK_HOME}" \
  "${TMP}/archive-db-hardlink-start" > "${TMP}/archive-db-hardlink.log" 2>&1; then
  fail "archive candidate accepted a hardlinked database file"
fi
grep -q 'hardlinked file' "${TMP}/archive-db-hardlink.log" || \
  fail "database hardlink error was not explicit"
[ "$(sha256_file "${DB_HARDLINK_SENTINEL}")" = "${DB_HARDLINK_SENTINEL_SHA}" ] && \
  [ "$(file_mode "${DB_HARDLINK_SENTINEL}")" = "644" ] || \
  fail "database hardlink target was mutated"

# The isolated candidate must prove status/ABCI/block/commit at A, an empty A,
# and absence of both block H and block-results H before readiness is emitted.
CANDIDATE_READY="${TMP}/archive-candidate-daemon-ready"
CANDIDATE_RELEASE="${TMP}/archive-candidate-daemon-release"
CANDIDATE_START="${TMP}/archive-candidate-start"
archive_runtime archive-candidate "${ARCHIVE_HOME}" "${CANDIDATE_START}" \
  START_HOLD_READY="${CANDIDATE_READY}" START_HOLD_RELEASE="${CANDIDATE_RELEASE}" \
  > "${TMP}/archive-candidate.log" 2>&1 &
CANDIDATE_PID=$!
for _ in $(seq 1 100); do
  [ -f "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json" ] && break
  kill -0 "${CANDIDATE_PID}" 2>/dev/null || {
    sed -n '1,160p' "${TMP}/archive-candidate.log" >&2
    fail "archive candidate exited before producing readiness"
  }
  sleep 0.02
done
[ -f "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json" ] || \
  fail "archive candidate did not produce readiness"
grep -q '^role=archive-candidate$' "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive candidate role was not permanently marked"
grep -q "^archive_transition_nonce=${ARCHIVE_NONCE}$" \
  "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive candidate marker did not bind the reviewed nonce"
grep -q "^archive_transition_manifest_sha256=${ARCHIVE_TRANSITION_SHA256}$" \
  "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive candidate marker did not bind the reviewed transition manifest"
jq -e \
  --arg nonce "${ARCHIVE_NONCE}" \
  --arg signer "${SIGNER_EVIDENCE_SHA256}" \
  --arg observer "${OBSERVER_EVIDENCE_SHA256}" '
    .schema == "zerone-1-archive-readiness-v2"
    and .final_committed_height == "424241"
    and .halt_trigger_height == "424242"
    and .halt_trigger_block_absent == true
    and .halt_trigger_results_absent == true
    and .block_sync_catching_up == true
    and .anchor_commit_canonical == false
    and .archive_transition_nonce == $nonce
    and .source_evidence.signer_manifest_sha256 == $signer
    and .source_evidence.observer_manifest_sha256 == $observer
  ' "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json" >/dev/null || \
  fail "archive readiness did not bind A/H, evidence, and candidate nonce"
: > "${CANDIDATE_RELEASE}"
wait "${CANDIDATE_PID}" || {
  sed -n '1,160p' "${TMP}/archive-candidate.log" >&2
  fail "archive candidate did not stop cleanly"
}

# A readiness copied onto another unadopted sanitized copy is stale/replayed
# material and cannot be consumed to skip candidate proof.
REPLAY_HOME="${TMP}/archive-replay-home"
cp -R "${ARCHIVE_TEMPLATE_HOME}" "${REPLAY_HOME}"
cp "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json" \
  "${REPLAY_HOME}/.zerone-1-archive-readiness.json"
if archive_runtime archive-candidate "${REPLAY_HOME}" \
  "${TMP}/archive-replay-start" > "${TMP}/archive-replay.log" 2>&1; then
  fail "sanitized copy accepted replayed archive readiness"
fi
grep -q 'stale archive readiness; refusing replay' "${TMP}/archive-replay.log" || \
  fail "replayed-readiness error was not explicit"

ARCHIVE_START="${TMP}/archive-start"
if ! archive_runtime archive "${ARCHIVE_HOME}" "${ARCHIVE_START}" \
  > "${TMP}/archive.log" 2>&1; then
  sed -n '1,120p' "${TMP}/archive.log" >&2
  fail "checkpoint-armed observer did not transition to archive"
fi
grep -q '^role=archive$' "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive transition was not permanently marked"
grep -A30 '^\[rpc\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^laddr = "tcp://fly-local-6pn:26657"$' || \
  fail "archive RPC does not bind its private Fly 6PN interface"
grep -A8 '^\[api\]' "${ARCHIVE_HOME}/config/app.toml" | \
  grep -q '^address = "tcp://fly-local-6pn:1317"$' || \
  fail "archive REST does not bind its private Fly 6PN interface"
assert_bounded_query_config "${ARCHIVE_HOME}/config/app.toml" \
  "${ARCHIVE_HOME}/config/config.toml" "archive serving config"
grep -q -- '--query-gas-limit 5000000' "${ARCHIVE_START}" || \
  fail "archive start did not pin SDK query gas"
grep -q -- '--min-retain-blocks 0' "${ARCHIVE_START}" || \
  fail "archive start did not retain Comet history"
grep -q "^archive_transition_nonce=${ARCHIVE_NONCE}$" \
  "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive marker did not consume the proven candidate nonce"
grep -Eq '^archive_readiness_sha256=[0-9a-f]{64}$' \
  "${ARCHIVE_HOME}/.zerone-1-runtime" || \
  fail "archive marker did not pin the consumed readiness hash"
grep -A14 '^\[p2p\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^persistent_peers = ""$' || fail "archive inherited the signer peer"
grep -A14 '^\[p2p\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^seeds = ""$' || fail "archive retained P2P seeds"
grep -A14 '^\[p2p\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^pex = false$' || fail "archive left PEX enabled"
grep -A14 '^\[p2p\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^max_num_inbound_peers = 0$' || fail "archive accepts inbound P2P peers"
grep -A14 '^\[p2p\]' "${ARCHIVE_HOME}/config/config.toml" | \
  grep -q '^max_num_outbound_peers = 0$' || fail "archive can dial outbound P2P peers"
grep -q -- '--halt-height 424242' "${ARCHIVE_START}" || \
  fail "archive start lost the checkpoint halt fence"

if NODE_ROLE=observer ZERONE_HOME="${ARCHIVE_HOME}" \
  MAINNET_SEED_DIR="${TMP}/seed" START_MARKER="${TMP}/archive-revert-start" \
  PERSISTENT_PEERS="${OBSERVER_PEER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/archive-revert.log" 2>&1; then
  fail "archive role reverted to syncing observer"
fi
grep -q 'volume role does not match' "${TMP}/archive-revert.log" || \
  fail "one-way archive-transition error was not explicit"

# The attestation remains integrity-bound on every archive restart. It cannot be
# edited and replayed after the role marker has consumed its exact hash.
cp "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json" \
  "${TMP}/archive-readiness.saved.json"
printf ' \n' >> "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json"
if archive_runtime archive "${ARCHIVE_HOME}" "${TMP}/archive-tamper-start" \
  > "${TMP}/archive-tamper.log" 2>&1; then
  fail "archive accepted modified consumed readiness"
fi
grep -q 'changed after transition' "${TMP}/archive-tamper.log" || \
  fail "modified-readiness error was not explicit"
mv "${TMP}/archive-readiness.saved.json" \
  "${ARCHIVE_HOME}/.zerone-1-archive-readiness.json"

# The retired generic escape hatch must not reach the daemon.
rm -f "${START_MARKER}"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  EXTRA_START_FLAGS="--home ${TMP}/unvalidated" \
  "${ENTRYPOINT}" > "${TMP}/override.log" 2>&1; then
  fail "retired EXTRA_START_FLAGS reached zeroned"
fi
grep -q 'EXTRA_START_FLAGS is retired' "${TMP}/override.log" || \
  fail "retired-flag error was not explicit"
[ ! -f "${START_MARKER}" ] || fail "zeroned started with an unvalidated home"

# The public freeze selects state F, commits it in an empty anchor block A=F+1,
# and passes only the SDK halt trigger H=A+1. The old single-height variable is
# ambiguous and must never reach zeroned.
rm -f "${START_MARKER}"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_HALT_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/retired-halt.log" 2>&1; then
  fail "ambiguous legacy halt height reached zeroned"
fi
grep -q 'ZERONE_HALT_HEIGHT is ambiguous and retired' \
  "${TMP}/retired-halt.log" || fail "legacy halt rejection was not explicit"
[ ! -f "${START_MARKER}" ] || fail "zeroned started with a legacy halt height"

if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  "${ENTRYPOINT}" > "${TMP}/partial-plan.log" 2>&1; then
  fail "partial checkpoint plan reached zeroned"
fi
grep -q 'must be set together' "${TMP}/partial-plan.log" || \
  fail "partial checkpoint-plan error was not explicit"

if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT='10 --home /tmp/override' \
  ZERONE_FINAL_COMMITTED_HEIGHT=11 ZERONE_HALT_TRIGGER_HEIGHT=12 \
  "${ENTRYPOINT}" > "${TMP}/bad-plan.log" 2>&1; then
  fail "malformed checkpoint height reached zeroned"
fi
grep -q 'must be a canonical positive integer' "${TMP}/bad-plan.log" || \
  fail "malformed checkpoint error was not explicit"

if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424242 ZERONE_HALT_TRIGGER_HEIGHT=424243 \
  "${ENTRYPOINT}" > "${TMP}/bad-relation.log" 2>&1; then
  fail "non-consecutive checkpoint heights reached zeroned"
fi
grep -q 'FINAL_COMMITTED_HEIGHT must equal' "${TMP}/bad-relation.log" || \
  fail "checkpoint relation error was not explicit"

if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  UNSAFE_SKIP_UPGRADES_HEIGHT=100 \
  "${ENTRYPOINT}" > "${TMP}/conflicting-halt.log" 2>&1; then
  fail "conflicting checkpoint/upgrade actions reached zeroned"
fi
grep -q 'mutually exclusive' "${TMP}/conflicting-halt.log" || \
  fail "conflicting checkpoint/upgrade error was not explicit"
[ ! -f "${START_MARKER}" ] || fail "zeroned started with conflicting actions"

if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" EXTERNAL_ADDRESS=public.example:26656 \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/public-checkpoint.log" 2>&1; then
  fail "checkpoint plan retained a public signer advertisement"
fi
grep -q 'checkpoint plan forbids EXTERNAL_ADDRESS' \
  "${TMP}/public-checkpoint.log" || \
  fail "public checkpoint-profile error was not explicit"

printf '%s\n' '{"height":"424240","round":0,"step":0}' \
  > "${HOME_DIR}/data/priv_validator_state.json"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/late-plan.log" 2>&1; then
  fail "late checkpoint plan reached zeroned"
fi
grep -q 'checkpoint plan was armed too late' "${TMP}/late-plan.log" || \
  fail "late checkpoint-plan error was not explicit"
printf '%s\n' '{"height":"0","round":0,"step":0}' \
  > "${HOME_DIR}/data/priv_validator_state.json"

if ! ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" \
  ZERONE_CHECKPOINT_STATE_HEIGHT=424240 \
  ZERONE_FINAL_COMMITTED_HEIGHT=424241 ZERONE_HALT_TRIGGER_HEIGHT=424242 \
  "${ENTRYPOINT}" > "${TMP}/halt.log" 2>&1; then
  sed -n '1,80p' "${TMP}/halt.log" >&2
  fail "valid checkpoint plan did not reach zeroned"
fi
grep -q -- '--halt-height 424242' "${START_MARKER}" || \
  fail "start command lost the exact halt trigger"
grep -q '^broadcast = false$' "${HOME_DIR}/config/config.toml" || \
  fail "checkpoint plan did not disable mempool broadcast"
grep -q '^size = 0$' "${HOME_DIR}/config/config.toml" || \
  fail "checkpoint plan did not zero mempool capacity"
grep -q '^max_txs_bytes = 0$' "${HOME_DIR}/config/config.toml" || \
  fail "checkpoint plan did not zero mempool bytes"
grep -q '^max-txs = -1$' "${HOME_DIR}/config/app.toml" || \
  fail "checkpoint plan did not disable the app-side mempool"
grep -A14 '^\[p2p\]' "${HOME_DIR}/config/config.toml" | \
  grep -q '^external_address = ""$' || \
  fail "checkpoint signer retained a public external address"
grep -A14 '^\[p2p\]' "${HOME_DIR}/config/config.toml" | \
  grep -q '^pex = false$' || fail "checkpoint signer retained P2P exchange"
grep -A14 '^\[p2p\]' "${HOME_DIR}/config/config.toml" | \
  grep -q '^laddr = "tcp://fly-local-6pn:26656"$' || \
  fail "checkpoint signer P2P does not bind its private Fly 6PN interface"
grep -A30 '^\[rpc\]' "${HOME_DIR}/config/config.toml" | \
  grep -q '^laddr = "tcp://fly-local-6pn:26657"$' || \
  fail "checkpoint signer RPC does not bind its private Fly 6PN interface"
if ZERONE_HOME="${HOME_DIR}" MAINNET_SEED_DIR="${TMP}/seed" \
  START_MARKER="${START_MARKER}" "${ENTRYPOINT}" \
  > "${TMP}/disarmed-plan.log" 2>&1; then
  fail "persisted checkpoint plan could be removed on restart"
fi
grep -q 'permanently checkpoint-armed' "${TMP}/disarmed-plan.log" || \
  fail "disarmed checkpoint-plan error was not explicit"

# If the real binary was built, prove Comet can load the generated fixtures and
# derives the same validator public key used by the shell boundary.
if [ -x "${ROOT}/build/zeroned" ]; then
  REAL_HOME="${TMP}/real-comet-home"
  mkdir -p "${REAL_HOME}/config" "${REAL_HOME}/data"
  cp "${TMP}/node_key.json" "${REAL_HOME}/config/node_key.json"
  cp "${TMP}/validator_key.json" "${REAL_HOME}/config/priv_validator_key.json"
  printf '%s\n' '{"height":"0","round":0,"step":0}' \
    > "${REAL_HOME}/data/priv_validator_state.json"
  REAL_NODE_ID=$("${ROOT}/build/zeroned" tendermint show-node-id --home "${REAL_HOME}")
  REAL_VALIDATOR_PUB=$("${ROOT}/build/zeroned" tendermint show-validator \
    --home "${REAL_HOME}" | jq -r '.key')
  STORED_VALIDATOR_PUB=$(jq -r '.pub_key.value' "${TMP}/validator_key.json")
  [ "${REAL_NODE_ID}" = "${FIXTURE_NODE_ID}" ] || \
    fail "real Comet node-id derivation disagrees with fixture"
  [ "${REAL_VALIDATOR_PUB}" = "${STORED_VALIDATOR_PUB}" ] || \
    fail "real Comet validator derivation disagrees with fixture"

fi

# Defense in depth: private custody files must not be copied by the image.
if grep -Eq '^COPY .*\b(node_key|priv_validator_key)\.json\b' \
  "${ROOT}/deploy/mainnet/Dockerfile"; then
  fail "Dockerfile still copies a private key"
fi
grep -qx 'deploy/mainnet/artifacts/priv_validator_key.json' \
  "${ROOT}/.dockerignore" || fail "validator key is not excluded from build context"
grep -qx 'deploy/mainnet/artifacts/node_key.json' \
  "${ROOT}/.dockerignore" || fail "node key is not excluded from build context"
grep -qx 'deploy/mainnet/artifacts/\*.mnemonic' \
  "${ROOT}/.dockerignore" || fail "mnemonics are not excluded from build context"
for private_profile in \
  fly.halt-signer.example.toml fly.observer.example.toml \
  fly.archive-candidate.example.toml fly.archive.example.toml; do
  if grep -Eq '^\[\[services\]\]' "${ROOT}/deploy/mainnet/${private_profile}"; then
    fail "${private_profile} unexpectedly exposes a public Fly service"
  fi
done
grep -q '^  NODE_ROLE = "archive-candidate"$' \
  "${ROOT}/deploy/mainnet/fly.archive-candidate.example.toml" || \
  fail "archive candidate Fly stage is missing"
for archive_profile in fly.archive-candidate.example.toml fly.archive.example.toml; do
  grep -q '^app = "zerone-1-archive"$' \
    "${ROOT}/deploy/mainnet/${archive_profile}" || \
    fail "${archive_profile} does not use the isolated archive app"
  grep -q '^  source = "zerone_archive_data"$' \
    "${ROOT}/deploy/mainnet/${archive_profile}" || \
    fail "${archive_profile} does not use the isolated archive volume"
  for required_input in ZERONE_EXPECTED_ANCHOR_BLOCK_HASH \
    ZERONE_EXPECTED_POST_ANCHOR_APP_HASH \
    ZERONE_SIGNER_EVIDENCE_MANIFEST_SHA256 \
    ZERONE_OBSERVER_EVIDENCE_MANIFEST_SHA256 \
    ZERONE_ARCHIVE_TRANSITION_MANIFEST_SHA256; do
    grep -q "^  ${required_input} = " \
      "${ROOT}/deploy/mainnet/${archive_profile}" || \
      fail "${archive_profile} omits ${required_input}"
  done
done

echo "entrypoint test: PASS"
