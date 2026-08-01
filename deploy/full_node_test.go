package deploy_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const fakeFullNodeID = "cccccccccccccccccccccccccccccccccccccccc"
const fakeFullNodeConsensusAddress = "66687AADF862BD776C8FC18B8E9F8E2008971485"

type fullNodeFixture struct {
	root       string
	entrypoint string
	dataRoot   string
	home       string
	logPath    string
}

func writeFullNodeFile(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
}

func copyFullNodeFile(t *testing.T, source, destination string, mode os.FileMode) {
	t.Helper()
	contents, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	writeFullNodeFile(t, destination, contents, mode)
}

func fullNodeSHA256(contents []byte) string {
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}

func newFullNodeFixture(t *testing.T, network string) fullNodeFixture {
	t.Helper()
	root := t.TempDir()
	binDir := filepath.Join(root, "usr", "local", "bin")
	shareDir := filepath.Join(root, "usr", "local", "share", "zerone")
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	chainID := "zerone-1"
	genesisSource := filepath.Join("mainnet", "artifacts", "genesis.json")
	if network == "testnet" {
		chainID = "zerone-testnet-1"
		genesisSource = filepath.Join("testnet", "artifacts", "genesis.json")
	}
	genesis, err := os.ReadFile(genesisSource)
	if err != nil {
		t.Fatal(err)
	}

	entrypoint := filepath.Join(binDir, "fly-full-node-entrypoint")
	copyFullNodeFile(t, "fly-full-node-entrypoint.sh", entrypoint, 0o755)
	copyFullNodeFile(
		t,
		"public-edge-nginx.conf",
		filepath.Join(shareDir, "public-edge-nginx.conf"),
		0o644,
	)
	writeFullNodeFile(t, filepath.Join(shareDir, "genesis.json"), genesis, 0o644)
	writeFullNodeFile(t, filepath.Join(shareDir, "genesis.sha256"), []byte(fullNodeSHA256(genesis)+"\n"), 0o644)
	writeFullNodeFile(t, filepath.Join(shareDir, "chain-id"), []byte(chainID+"\n"), 0o644)

	logPath := filepath.Join(root, "runtime.log")
	fakeZeroned := []byte(`#!/usr/bin/env bash
set -euo pipefail
command_name="${1:-}"
if [[ "${command_name}" == "init" ]]; then
  home=""
  previous=""
  for argument in "$@"; do
    if [[ "${previous}" == "--home" ]]; then
      home="${argument}"
    fi
    previous="${argument}"
  done
  test -n "${home}"
  mkdir -p "${home}/config" "${home}/data"
  cat > "${home}/config/config.toml" <<'CONFIG'
priv_validator_laddr = ""
[rpc]
laddr = "tcp://127.0.0.1:26657"
grpc_laddr = ""
cors_allowed_origins = []
unsafe = false
max_open_connections = 900
max_subscription_clients = 100
max_subscriptions_per_client = 5
max_request_batch_size = 10
max_body_bytes = 1000000
pprof_laddr = ""
[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = ""
seeds = ""
persistent_peers = ""
addr_book_strict = true
max_num_inbound_peers = 40
max_num_outbound_peers = 10
unconditional_peer_ids = ""
pex = true
seed_mode = false
private_peer_ids = ""
allow_duplicate_ip = false
[instrumentation]
prometheus = false
prometheus_listen_addr = ":26660"
CONFIG
  cat > "${home}/config/app.toml" <<'APP'
minimum-gas-prices = ""
[telemetry]
enabled = false
[api]
enable = false
swagger = false
address = "tcp://localhost:1317"
max-open-connections = 1000
rpc-max-body-bytes = 1000000
enabled-unsafe-cors = false
[grpc]
enable = true
address = "localhost:9090"
max-recv-msg-size = "10485760"
max-send-msg-size = "2147483647"
[grpc-web]
enable = true
APP
  printf '%s\n' '{"chain_id":"temporary"}' > "${home}/config/genesis.json"
  printf '%s\n' '{"address":"` + fakeFullNodeConsensusAddress + `","pub_key":{"type":"tendermint/PubKeyEd25519","value":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}' > "${home}/config/priv_validator_key.json"
  printf '%s\n' '{"height":"0","round":0,"step":0}' > "${home}/data/priv_validator_state.json"
  printf '%s\n' '{"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"ZmFrZQ=="}}' > "${home}/config/node_key.json"
  printf '%s\n' init >> "${FAKE_FULL_NODE_LOG}"
  exit 0
fi
if [[ "${command_name}" == "comet" && "${2:-}" == "show-node-id" ]]; then
  printf '%s\n' "` + fakeFullNodeID + `"
  exit 0
fi
if [[ "${command_name}" == "comet" && "${2:-}" == "show-address" ]]; then
  printf '%s\n' "zrnvalcons1legacy-output-is-not-an-uppercase-hex-address"
  exit 0
fi
if [[ "${command_name}" == "genesis" && "${2:-}" == "validate" ]] ||
  [[ "${command_name}" == "validate-genesis" ]]; then
  exit "${FAKE_GENESIS_VALIDATE_EXIT_STATUS:-0}"
fi
if [[ "${command_name}" == "start" ]]; then
  printf '%s\n' start >> "${FAKE_FULL_NODE_LOG}"
  exit "${FAKE_NODE_EXIT_STATUS:-0}"
fi
printf 'unexpected fake zeroned command: %s\n' "$*" >&2
exit 91
`)
	zeronedPath := filepath.Join(binDir, "zeroned")
	writeFullNodeFile(t, zeronedPath, fakeZeroned, 0o755)
	binaryDigest, err := os.ReadFile(zeronedPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFullNodeFile(
		t,
		filepath.Join(shareDir, "zeroned.sha256"),
		[]byte(fullNodeSHA256(binaryDigest)+"\n"),
		0o644,
	)
	writeFullNodeFile(t, filepath.Join(binDir, "nginx"), []byte(`#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' nginx >> "${FAKE_FULL_NODE_LOG}"
exit "${FAKE_NGINX_EXIT_STATUS:-0}"
`), 0o755)

	return fullNodeFixture{
		root:       root,
		entrypoint: entrypoint,
		dataRoot:   dataRoot,
		home:       filepath.Join(dataRoot, ".zeroned"),
		logPath:    logPath,
	}
}

func fullNodeTopologySHA256(
	chainID,
	genesisSHA256,
	role,
	moniker,
	validatorPeer,
	sentryPeers,
	externalAddress string,
) string {
	payload := fmt.Sprintf(
		"schema=zerone.fly-full-node-topology/v1\n"+
			"chain_id=%s\n"+
			"genesis_sha256=%s\n"+
			"role=%s\n"+
			"moniker=%s\n"+
			"validator_peer=%s\n"+
			"sentry_peers=%s\n"+
			"external_p2p_address=%s\n",
		chainID,
		genesisSHA256,
		role,
		moniker,
		validatorPeer,
		sentryPeers,
		externalAddress,
	)
	return fullNodeSHA256([]byte(payload))
}

func fullNodeEnv(fixture fullNodeFixture, values map[string]string) []string {
	environment := []string{
		"PATH=" + os.Getenv("PATH"),
		"ZERONE_HOME=" + fixture.home,
		"FAKE_FULL_NODE_LOG=" + fixture.logPath,
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment
}

func runFullNodeEntrypoint(
	fixture fullNodeFixture,
	values map[string]string,
) ([]byte, error) {
	command := exec.Command(fixture.entrypoint)
	command.Env = fullNodeEnv(fixture, values)
	return command.CombinedOutput()
}

func readFixtureGenesisDigest(t *testing.T, fixture fullNodeFixture) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(fixture.root, "usr", "local", "share", "zerone", "genesis.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(contents))
}

func TestFullNodeIdentityCeremonyAndSentryFinalization(t *testing.T) {
	fixture := newFullNodeFixture(t, "mainnet")
	ceremony := map[string]string{
		"ZERONE_NODE_ROLE":         "sentry",
		"ZERONE_MONIKER":           "zerone-1-sentry-a",
		"ZERONE_IDENTITY_CEREMONY": "generate-only",
	}
	output, err := runFullNodeEntrypoint(fixture, ceremony)
	if err != nil {
		t.Fatalf("identity ceremony failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "daemon was not started") ||
		!strings.Contains(string(output), "node_id="+fakeFullNodeID) {
		t.Fatalf("identity evidence missing:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(
		fixture.home,
		"config",
		"zerone-full-node-identity-pending.json",
	)); err != nil {
		t.Fatalf("pending identity manifest missing: %v", err)
	}
	runtimeLog, err := os.ReadFile(fixture.logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(runtimeLog) != "init\n" {
		t.Fatalf("identity ceremony started a daemon:\n%s", runtimeLog)
	}

	validatorPeer := strings.Repeat("a", 40) + "@validator.internal:26656"
	sentryPeers := strings.Repeat("b", 40) + "@zerone-1-sentry-b.internal:26656"
	externalAddress := "zerone-1-sentry-a.fly.dev:26656"
	final := map[string]string{
		"ZERONE_NODE_ROLE":            "sentry",
		"ZERONE_MONIKER":              "zerone-1-sentry-a",
		"ZERONE_VALIDATOR_PEER":       validatorPeer,
		"ZERONE_SENTRY_PEERS":         sentryPeers,
		"ZERONE_EXTERNAL_P2P_ADDRESS": externalAddress,
	}
	final["ZERONE_TOPOLOGY_SHA256"] = fullNodeTopologySHA256(
		"zerone-1",
		readFixtureGenesisDigest(t, fixture),
		"sentry",
		"zerone-1-sentry-a",
		validatorPeer,
		sentryPeers,
		externalAddress,
	)
	output, err = runFullNodeEntrypoint(fixture, final)
	if err != nil {
		t.Fatalf("sentry finalization failed: %v\n%s", err, output)
	}

	manifestPath := filepath.Join(fixture.home, "config", "zerone-full-node-role.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	for field, expected := range map[string]string{
		"schema":            "zerone.fly-full-node-role/v1",
		"role":              "sentry",
		"node_id":           fakeFullNodeID,
		"consensus_address": fakeFullNodeConsensusAddress,
		"validator_peer":    validatorPeer,
		"sentry_peers":      sentryPeers,
	} {
		if manifest[field] != expected {
			t.Fatalf("manifest %s: want %q, got %#v", field, expected, manifest[field])
		}
	}
	if _, err := os.Stat(filepath.Join(
		fixture.home,
		"config",
		"zerone-full-node-identity-pending.json",
	)); !os.IsNotExist(err) {
		t.Fatalf("pending identity was not consumed: %v", err)
	}

	configBytes, err := os.ReadFile(filepath.Join(fixture.home, "config", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(configBytes)
	for _, required := range []string{
		`laddr = "tcp://0.0.0.0:26656"`,
		`laddr = "tcp://127.0.0.1:26657"`,
		`persistent_peers = "` + validatorPeer + `,` + sentryPeers + `"`,
		`private_peer_ids = "` + strings.Repeat("a", 40) + `"`,
		`unconditional_peer_ids = "` + strings.Repeat("a", 40) + `,` + strings.Repeat("b", 40) + `"`,
		`pex = true`,
		`unsafe = false`,
	} {
		if !strings.Contains(config, required) {
			t.Fatalf("sentry config missing %q:\n%s", required, config)
		}
	}
	appBytes, err := os.ReadFile(filepath.Join(fixture.home, "config", "app.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appBytes), "[api]\nenable = false") ||
		!strings.Contains(string(appBytes), "[grpc]\nenable = false") {
		t.Fatalf("sentry API policy drifted:\n%s", appBytes)
	}
}

func TestFullNodeRejectsNonEmptyVolumeAndRuntimeKeySources(t *testing.T) {
	t.Run("non-empty volume", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		writeFullNodeFile(t, filepath.Join(fixture.dataRoot, "unexpected"), []byte("state"), 0o600)
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
		})
		if err == nil || !strings.Contains(string(output), "entirely empty") {
			t.Fatalf("non-empty volume was not rejected: %v\n%s", err, output)
		}
		if _, err := os.Stat(fixture.logPath); !os.IsNotExist(err) {
			t.Fatalf("zeroned ran before empty-volume rejection: %v", err)
		}
	})

	t.Run("pristine Fly lost+found", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		lostFound := filepath.Join(fixture.dataRoot, "lost+found")
		if err := os.Mkdir(lostFound, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(lostFound, 0o700); err != nil {
			t.Fatal(err)
		}
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
		})
		if err != nil || !strings.Contains(string(output), "daemon was not started") {
			t.Fatalf("pristine Fly lost+found was not accepted: %v\n%s", err, output)
		}
	})

	t.Run("non-empty Fly lost+found", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		lostFound := filepath.Join(fixture.dataRoot, "lost+found")
		if err := os.Mkdir(lostFound, 0o700); err != nil {
			t.Fatal(err)
		}
		writeFullNodeFile(t, filepath.Join(lostFound, "smuggled-state"), []byte("state"), 0o600)
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
		})
		if err == nil || !strings.Contains(string(output), "lost+found must be empty") {
			t.Fatalf("non-empty Fly lost+found was not rejected: %v\n%s", err, output)
		}
		if _, err := os.Stat(fixture.logPath); !os.IsNotExist(err) {
			t.Fatalf("zeroned ran before lost+found rejection: %v", err)
		}
	})

	t.Run("wrong-mode Fly lost+found", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		lostFound := filepath.Join(fixture.dataRoot, "lost+found")
		if err := os.Mkdir(lostFound, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(lostFound, 0o755); err != nil {
			t.Fatal(err)
		}
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
		})
		if err == nil || !strings.Contains(string(output), "owner or mode is not pristine") {
			t.Fatalf("wrong-mode Fly lost+found was not rejected: %v\n%s", err, output)
		}
	})

	t.Run("symlinked Fly lost+found", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		outside := filepath.Join(fixture.root, "outside-lost-found")
		if err := os.Mkdir(outside, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(fixture.dataRoot, "lost+found")); err != nil {
			t.Fatal(err)
		}
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
		})
		if err == nil || !strings.Contains(string(output), "lost+found must be a real directory") {
			t.Fatalf("symlinked Fly lost+found was not rejected: %v\n%s", err, output)
		}
	})

	t.Run("runtime key", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":         "sentry",
			"ZERONE_MONIKER":           "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY": "generate-only",
			"NODE_KEY_B64":             "forbidden",
		})
		if err == nil || !strings.Contains(string(output), "runtime-provided key material is forbidden") {
			t.Fatalf("runtime key was not rejected: %v\n%s", err, output)
		}
		if _, err := os.Stat(fixture.logPath); !os.IsNotExist(err) {
			t.Fatalf("zeroned ran before runtime-key rejection: %v", err)
		}
	})

	t.Run("binary genesis incompatibility", func(t *testing.T) {
		fixture := newFullNodeFixture(t, "mainnet")
		output, err := runFullNodeEntrypoint(fixture, map[string]string{
			"ZERONE_NODE_ROLE":                  "sentry",
			"ZERONE_MONIKER":                    "zerone-1-sentry-a",
			"ZERONE_IDENTITY_CEREMONY":          "generate-only",
			"FAKE_GENESIS_VALIDATE_EXIT_STATUS": "1",
		})
		if err == nil || !strings.Contains(
			string(output),
			"cannot decode the image-frozen genesis",
		) {
			t.Fatalf("incompatible binary/genesis pair was not rejected: %v\n%s", err, output)
		}
		if _, err := os.Stat(fixture.logPath); !os.IsNotExist(err) {
			t.Fatalf("init ran before binary/genesis rejection: %v", err)
		}
	})
}

func TestFullNodeRejectsPeerAndConfigDriftBeforeRestart(t *testing.T) {
	fixture := newFullNodeFixture(t, "mainnet")
	ceremony := map[string]string{
		"ZERONE_NODE_ROLE":         "sentry",
		"ZERONE_MONIKER":           "zerone-1-sentry-a",
		"ZERONE_IDENTITY_CEREMONY": "generate-only",
	}
	if output, err := runFullNodeEntrypoint(fixture, ceremony); err != nil {
		t.Fatalf("identity ceremony failed: %v\n%s", err, output)
	}
	validatorPeer := strings.Repeat("a", 40) + "@validator.internal:26656"
	sentryPeers := strings.Repeat("b", 40) + "@sentry-b.example:26656"
	values := map[string]string{
		"ZERONE_NODE_ROLE":            "sentry",
		"ZERONE_MONIKER":              "zerone-1-sentry-a",
		"ZERONE_VALIDATOR_PEER":       validatorPeer,
		"ZERONE_SENTRY_PEERS":         sentryPeers,
		"ZERONE_EXTERNAL_P2P_ADDRESS": "sentry-a.example:26656",
	}
	values["ZERONE_TOPOLOGY_SHA256"] = fullNodeTopologySHA256(
		"zerone-1",
		readFixtureGenesisDigest(t, fixture),
		"sentry",
		values["ZERONE_MONIKER"],
		validatorPeer,
		sentryPeers,
		values["ZERONE_EXTERNAL_P2P_ADDRESS"],
	)
	if output, err := runFullNodeEntrypoint(fixture, values); err != nil {
		t.Fatalf("finalization failed: %v\n%s", err, output)
	}
	if err := os.WriteFile(fixture.logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	drifted := map[string]string{}
	for key, value := range values {
		drifted[key] = value
	}
	drifted["ZERONE_SENTRY_PEERS"] = strings.Repeat("e", 40) + "@sentry-c.example:26656"
	drifted["ZERONE_TOPOLOGY_SHA256"] = fullNodeTopologySHA256(
		"zerone-1",
		readFixtureGenesisDigest(t, fixture),
		"sentry",
		drifted["ZERONE_MONIKER"],
		drifted["ZERONE_VALIDATOR_PEER"],
		drifted["ZERONE_SENTRY_PEERS"],
		drifted["ZERONE_EXTERNAL_P2P_ADDRESS"],
	)
	output, err := runFullNodeEntrypoint(fixture, drifted)
	if err == nil || !strings.Contains(string(output), "topology drift detected") {
		t.Fatalf("peer drift was not rejected: %v\n%s", err, output)
	}
	runtimeLog, err := os.ReadFile(fixture.logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeLog) != 0 {
		t.Fatalf("daemon ran after peer drift:\n%s", runtimeLog)
	}

	configPath := filepath.Join(fixture.home, "config", "config.toml")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(configBytes, []byte("\n# drift\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err = runFullNodeEntrypoint(fixture, values)
	if err == nil || !strings.Contains(string(output), "Comet configuration drift detected") {
		t.Fatalf("config drift was not rejected: %v\n%s", err, output)
	}
	runtimeLog, err = os.ReadFile(fixture.logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeLog) != 0 {
		t.Fatalf("daemon ran after config drift:\n%s", runtimeLog)
	}
}

func TestPublicQueryRoleIsLoopbackOnlyAndSupervisorFailsClosed(t *testing.T) {
	fixture := newFullNodeFixture(t, "testnet")
	ceremony := map[string]string{
		"ZERONE_NODE_ROLE":         "public-query",
		"ZERONE_MONIKER":           "zerone-testnet-1-query",
		"ZERONE_IDENTITY_CEREMONY": "generate-only",
	}
	if output, err := runFullNodeEntrypoint(fixture, ceremony); err != nil {
		t.Fatalf("identity ceremony failed: %v\n%s", err, output)
	}
	sentryPeers := strings.Repeat("a", 40) + "@sentry-a.example:26656," +
		strings.Repeat("b", 40) + "@sentry-b.example:26656"
	values := map[string]string{
		"ZERONE_NODE_ROLE":            "public-query",
		"ZERONE_MONIKER":              "zerone-testnet-1-query",
		"ZERONE_VALIDATOR_PEER":       "",
		"ZERONE_SENTRY_PEERS":         sentryPeers,
		"ZERONE_EXTERNAL_P2P_ADDRESS": "",
		"FAKE_NODE_EXIT_STATUS":       "0",
		"FAKE_NGINX_EXIT_STATUS":      "0",
	}
	values["ZERONE_TOPOLOGY_SHA256"] = fullNodeTopologySHA256(
		"zerone-testnet-1",
		readFixtureGenesisDigest(t, fixture),
		"public-query",
		values["ZERONE_MONIKER"],
		"",
		sentryPeers,
		"",
	)
	output, err := runFullNodeEntrypoint(fixture, values)
	if err == nil || !strings.Contains(string(output), "subprocess exited; failing closed") {
		t.Fatalf("public supervisor did not fail closed: %v\n%s", err, output)
	}
	configBytes, err := os.ReadFile(filepath.Join(fixture.home, "config", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(configBytes)
	for _, required := range []string{
		`laddr = "tcp://127.0.0.1:26656"`,
		`laddr = "tcp://127.0.0.1:26657"`,
		`persistent_peers = "` + sentryPeers + `"`,
		`private_peer_ids = ""`,
		`unconditional_peer_ids = "` + strings.Repeat("a", 40) + `,` +
			strings.Repeat("b", 40) + `"`,
		`pex = false`,
		`max_num_inbound_peers = 0`,
		`max_num_outbound_peers = 0`,
	} {
		if !strings.Contains(config, required) {
			t.Fatalf("public query config missing %q:\n%s", required, config)
		}
	}
	appBytes, err := os.ReadFile(filepath.Join(fixture.home, "config", "app.toml"))
	if err != nil {
		t.Fatal(err)
	}
	app := string(appBytes)
	for _, required := range []string{
		"[api]\nenable = true",
		`address = "tcp://127.0.0.1:1317"`,
		"[grpc]\nenable = true",
		`address = "127.0.0.1:9090"`,
		"[grpc-web]\nenable = false",
	} {
		if !strings.Contains(app, required) {
			t.Fatalf("public query app config missing %q:\n%s", required, app)
		}
	}
}

func TestPublicEdgePolicyIsReadOnlyAndExplicit(t *testing.T) {
	bodyBytes, err := os.ReadFile("public-edge-nginx.conf")
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)
	for _, required := range []string{
		`$request_method !~ ^(GET|HEAD|OPTIONS)$`,
		`$request_method = OPTIONS`,
		`add_header Access-Control-Allow-Origin "*" always`,
		`proxy_hide_header Access-Control-Allow-Origin`,
		`^/(?:rpc/)?(broadcast_tx_async|broadcast_tx_sync|broadcast_tx_commit|broadcast_evidence|unsafe_[a-z0-9_]+|dial_peers|dial_seeds)(/|$)`,
		"broadcast_tx_async",
		"broadcast_tx_sync",
		"broadcast_tx_commit",
		"broadcast_evidence",
		"dial_peers",
		"dial_seeds",
		"subscribe",
		"websocket",
		"tx_search",
		"block_search",
		`^/(?:rest/)?cosmos/feegrant/v1beta1/issued(/|$)`,
		`^/(?:rest/)?cosmos/tx/v1beta1/txs/?$`,
		`location ~ ^/rpc/(health|status|net_info|abci_info|abci_query|block|block_by_hash|block_results|commit|validators|consensus_params|genesis|genesis_chunked|header|header_by_hash|tx)/?$`,
		`rewrite ^/rpc(/.*)$ $1 break;`,
		`location ~ ^/rest/(cosmos|cosmos_proto|ibc|ibc_apps|zerone)/`,
		`rewrite ^/rest(/.*)$ $1 break;`,
		"proxy_pass http://zerone_comet_rpc",
		"proxy_pass http://zerone_rest",
		"location / {\n            return 404;",
		"client_max_body_size 64k",
		"limit_req_zone",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("public edge is missing policy %q", required)
		}
	}
	for _, forbidden := range []string{
		"grpc_pass",
		"127.0.0.1:9090",
		"26656",
		"proxy_pass http://$",
		"location /rpc/",
		"location /rest/",
		"location ^~ /rpc/",
		"location ^~ /rest/",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public edge unexpectedly contains %q", forbidden)
		}
	}
	for upstream, want := range map[string]int{
		"proxy_pass http://zerone_comet_rpc": 2,
		"proxy_pass http://zerone_rest":      2,
	} {
		if got := strings.Count(body, upstream); got != want {
			t.Fatalf("public edge has %d %q routes, want exactly %d explicit routes", got, upstream, want)
		}
	}
	restAllow := strings.Index(body, `location ~ ^/rest/(cosmos|cosmos_proto|ibc|ibc_apps|zerone)/`)
	for _, deny := range []string{
		`location ~ ^/(?:rest/)?cosmos/feegrant/v1beta1/issued(/|$)`,
		`location ~ ^/(?:rest/)?cosmos/tx/v1beta1/txs/?$`,
	} {
		denyAt := strings.Index(body, deny)
		if denyAt < 0 || restAllow < 0 || denyAt > restAllow {
			t.Fatalf("public REST deny %q must precede the prefix allowlist", deny)
		}
	}
}

func TestRPCEdgeImageUsesPrivateReadOnlyBoundary(t *testing.T) {
	dockerfileBytes, err := os.ReadFile("Dockerfile.rpc-edge")
	if err != nil {
		t.Fatal(err)
	}
	dockerfile := string(dockerfileBytes)
	for _, required := range []string{
		"debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818",
		"snapshot.debian.org/archive/debian/20260713T000000Z",
		"COPY deploy/public-edge-nginx.conf /etc/nginx/zerone-rpc.conf",
		"server zerone-1.internal:26657",
		"server zerone-1.internal:1317",
		"nginx -t -c /etc/nginx/zerone-rpc.conf",
		`io.zerone.node-class="stateless-read-only-rpc-edge"`,
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("RPC edge Dockerfile is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"PRIV_VALIDATOR",
		"NODE_KEY",
		"/data",
		"26656",
		"9090",
	} {
		if strings.Contains(dockerfile, forbidden) {
			t.Fatalf("RPC edge Dockerfile crosses boundary with %q", forbidden)
		}
	}

	configBytes, err := os.ReadFile("rpc-edge.fly.toml")
	if err != nil {
		t.Fatal(err)
	}
	config := string(configBytes)
	for _, required := range []string{
		`app = "zerone-rpc"`,
		`internal_port = 8080`,
		`force_https = true`,
		`auto_stop_machines = false`,
		`auto_start_machines = false`,
		`path = "/healthz"`,
		`policy = "on-failure"`,
	} {
		if !strings.Contains(config, required) {
			t.Fatalf("RPC edge Fly profile is missing %q", required)
		}
	}
	for _, forbidden := range []string{"[[mounts]]", "[[services]]", "26656", "26657", "1317", "9090"} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("RPC edge Fly profile unexpectedly exposes or mounts %q", forbidden)
		}
	}
}

func TestContainmentFlyTemplatesHaveDisjointRolesAndVolumes(t *testing.T) {
	templatePaths, err := filepath.Glob(filepath.Join("topology", "*", "*.fly.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(templatePaths) != 6 {
		t.Fatalf("want six Fly role templates, got %d", len(templatePaths))
	}
	appPattern := regexp.MustCompile(`(?m)^app = "([^"]+)"$`)
	volumePattern := regexp.MustCompile(`(?m)^  source = "([^"]+)"$`)
	apps := map[string]string{}
	volumes := map[string]string{}
	for _, path := range templatePaths {
		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		body := string(bodyBytes)
		appMatch := appPattern.FindStringSubmatch(body)
		volumeMatch := volumePattern.FindStringSubmatch(body)
		if len(appMatch) != 2 || len(volumeMatch) != 2 {
			t.Fatalf("%s lacks exact app or volume identity", path)
		}
		if previous := apps[appMatch[1]]; previous != "" {
			t.Fatalf("app %s reused by %s and %s", appMatch[1], previous, path)
		}
		if previous := volumes[volumeMatch[1]]; previous != "" {
			t.Fatalf("volume %s reused by %s and %s", volumeMatch[1], previous, path)
		}
		apps[appMatch[1]] = path
		volumes[volumeMatch[1]] = path
		for _, required := range []string{
			`image = "REPLACE_WITH_REGISTRY_IMAGE_AT_SHA256"`,
			`destination = "/data"`,
			`policy = "never"`,
			`ZERONE_TOPOLOGY_SHA256 = "REPLACE_WITH_64_LOWERCASE_HEX"`,
			`.internal:26656`,
		} {
			if !strings.Contains(body, required) {
				t.Fatalf("%s lacks %q", path, required)
			}
		}
		if strings.Contains(body, "dockerfile =") ||
			strings.Contains(body, "[build.args]") {
			t.Fatalf("%s rebuilds instead of consuming the reviewed image digest", path)
		}
		if strings.Contains(path, "sentry-") {
			if strings.Count(body, "[[services]]") != 1 ||
				!strings.Contains(body, "internal_port = 26656") ||
				!strings.Contains(body, "port = 26656") {
				t.Fatalf("%s is not P2P-only", path)
			}
			for _, forbidden := range []string{"26657", "1317", "9090", "[http_service]"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s exposes forbidden sentry surface %q", path, forbidden)
				}
			}
		} else {
			if !strings.Contains(body, "[http_service]") ||
				!strings.Contains(body, "internal_port = 8080") ||
				strings.Contains(body, "[[services]]") {
				t.Fatalf("%s is not a gateway-only public query profile", path)
			}
			for _, forbidden := range []string{
				"internal_port = 26656",
				"internal_port = 26657",
				"internal_port = 1317",
				"internal_port = 9090",
			} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s exposes forbidden query surface %q", path, forbidden)
				}
			}
		}
	}
}

func TestTopologyManifestsKeepSignerOutsidePublicServices(t *testing.T) {
	manifestPaths, err := filepath.Glob(filepath.Join("topology", "*", "topology.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifestPaths) != 2 {
		t.Fatalf("want two topology manifests, got %d", len(manifestPaths))
	}
	for _, path := range manifestPaths {
		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Schema         string `json:"schema"`
			GenesisSHA256  string `json:"genesis_sha256"`
			FullNodeImage  string `json:"full_node_image"`
			SignerBoundary struct {
				PublicServices []int  `json:"public_services"`
				RestartPolicy  string `json:"restart_policy"`
			} `json:"signer_boundary"`
		}
		if err := json.Unmarshal(bodyBytes, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Schema != "zerone.fly-containment-topology/v1" ||
			!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(manifest.GenesisSHA256) ||
			manifest.FullNodeImage != "REPLACE_WITH_REGISTRY_IMAGE_AT_SHA256" ||
			len(manifest.SignerBoundary.PublicServices) != 0 ||
			manifest.SignerBoundary.RestartPolicy != "no" {
			t.Fatalf("%s does not preserve the signer boundary", path)
		}
	}
}

func TestFullNodeDockerfileHasCleanLegacyExtractionBoundary(t *testing.T) {
	bodyBytes, err := os.ReadFile("Dockerfile.full-node")
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)
	for _, required := range []string{
		"golang:1.25.12-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58",
		"debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818",
		"FROM ${LEGACY_SOURCE_IMAGE} AS legacy-source",
		"LEGACY_ZERONED_SHA256",
		"@sha256:[0-9a-f]{64}",
		"COPY --from=legacy-source /usr/local/bin/zeroned /usr/local/bin/zeroned",
		"genesis validate",
		"FROM runtime-base AS legacy-full-node",
		"FROM runtime-base AS full-node",
		"COPY --from=builder /src/build/zeroned /usr/local/bin/zeroned",
		"sha256sum /usr/local/bin/zeroned",
		"COPY deploy/mainnet/artifacts/genesis.json",
		"COPY deploy/testnet/artifacts/genesis.json",
		`io.zerone.node-class="non-signing-full-node"`,
		`io.zerone.zeroned-origin="digest-pinned-legacy-image-extraction"`,
		`io.zerone.zeroned-source-image="${LEGACY_SOURCE_IMAGE}"`,
		`io.zerone.zeroned-sha256="${LEGACY_ZERONED_SHA256}"`,
		`io.zerone.zeroned-source-revision="unattributed"`,
		`io.zerone.zeroned-origin="source-build-at-oci-revision"`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("Dockerfile lacks clean-image control %q", required)
		}
	}
	for _, forbidden := range []string{
		"COPY --from=legacy-source / ",
		"COPY --from=legacy-source /data",
		"COPY --from=legacy-source /root",
		"COPY deploy/mainnet/artifacts/priv_validator",
		"COPY deploy/testnet/artifacts/priv_validator",
		"COPY deploy/mainnet/artifacts/node_key",
		"COPY deploy/testnet/artifacts/node_key",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Dockerfile crosses legacy/key boundary with %q", forbidden)
		}
	}
}
