package deploy_test

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	unifiedUpgradeName = "sdk-0.53-ibc-10"
	unifiedPlanInfo    = `{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}}`
)

func unifiedPlanDocument(t *testing.T) []byte {
	t.Helper()
	document, err := json.Marshal(struct {
		Plan struct {
			Name   string `json:"name"`
			Height string `json:"height"`
			Info   string `json:"info"`
		} `json:"plan"`
	}{
		Plan: struct {
			Name   string `json:"name"`
			Height string `json:"height"`
			Info   string `json:"info"`
		}{
			Name:   unifiedUpgradeName,
			Height: "100",
			Info:   unifiedPlanInfo,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func unifiedUpgradeInfoDocument(t *testing.T) []byte {
	t.Helper()
	document, err := json.Marshal(struct {
		Name   string `json:"name"`
		Height int64  `json:"height"`
		Info   string `json:"info"`
	}{
		Name:   unifiedUpgradeName,
		Height: 100,
		Info:   unifiedPlanInfo,
	})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func deterministicPrivateKey(label string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte(label))
	return ed25519.NewKeyFromSeed(seed[:])
}

func validatorKeyJSON(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) []byte {
	address := sha256.Sum256(publicKey)
	return []byte(fmt.Sprintf(
		`{"address":"%s","pub_key":{"type":"tendermint/PubKeyEd25519","value":"%s"},"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"%s"}}`,
		strings.ToUpper(hex.EncodeToString(address[:20])),
		base64.StdEncoding.EncodeToString(publicKey),
		base64.StdEncoding.EncodeToString(privateKey),
	))
}

func nodeKeyJSON(privateKey ed25519.PrivateKey) []byte {
	return []byte(fmt.Sprintf(
		`{"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"%s"}}`,
		base64.StdEncoding.EncodeToString(privateKey),
	))
}

func expectedIdentityEnvironment(t *testing.T, validatorJSON, nodeJSON []byte) []string {
	t.Helper()
	var validator struct {
		Address string `json:"address"`
	}
	var node struct {
		PrivateKey struct {
			Value string `json:"value"`
		} `json:"priv_key"`
	}
	if err := json.Unmarshal(validatorJSON, &validator); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(nodeJSON, &node); err != nil {
		t.Fatal(err)
	}
	nodePrivate, err := base64.StdEncoding.DecodeString(node.PrivateKey.Value)
	if err != nil {
		t.Fatal(err)
	}
	nodePublic := nodePrivate[len(nodePrivate)-ed25519.PublicKeySize:]
	nodeIDHash := sha256.Sum256(nodePublic)
	return []string{
		"EXPECTED_VALIDATOR_ADDRESS=" + validator.Address,
		"EXPECTED_NODE_ID=" + hex.EncodeToString(nodeIDHash[:20]),
		"EXPECTED_PRIV_VALIDATOR_KEY_SHA256=" + fileDigest(validatorJSON),
		"EXPECTED_NODE_KEY_SHA256=" + fileDigest(nodeJSON),
	}
}

func writeFile(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
}

func fileDigest(contents []byte) string {
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}

func cleanEnvironment(overrides ...string) []string {
	blockedPrefixes := []string{
		"PRIV_VALIDATOR_KEY_",
		"NODE_KEY_",
		"ZERONE_HOME=",
		"ZERONE_TEST_",
		"FLY_",
		"UPGRADE_",
		"LAST_COMMITTED_",
		"ATTEMPTED_UPGRADE_",
		"EXPECTED_",
		"OBSERVER_",
		"PRE_ARM_",
		"ARMED_",
		"OLD_BINARY_",
		"CHAIN_ID=",
		"DAEMON_",
		"UNSAFE_SKIP_BACKUP=",
		"FAKE_",
	}
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		blocked := false
		for _, prefix := range blockedPrefixes {
			if strings.HasPrefix(entry, prefix) {
				blocked = true
				break
			}
		}
		if !blocked {
			environment = append(environment, entry)
		}
	}
	return append(environment, overrides...)
}

func installFakeCommand(t *testing.T, directory, name, body string) {
	t.Helper()
	writeFile(t, filepath.Join(directory, name), []byte(body), 0o755)
}

func flyEntrypoint(network string) string {
	return filepath.Join(network, "entrypoint.sh")
}

func networkGenesis(t *testing.T, network string) []byte {
	t.Helper()
	genesis, err := os.ReadFile(filepath.Join(network, "artifacts", "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	return genesis
}

func runtimeKeyEnvironment(
	t *testing.T,
	home, validatorPath string,
	validatorJSON []byte,
	nodePath string,
	nodeJSON []byte,
) []string {
	return append([]string{
		"ZERONE_HOME=" + home,
		"PRIV_VALIDATOR_KEY_FILE=" + validatorPath,
		"PRIV_VALIDATOR_KEY_SHA256=" + fileDigest(validatorJSON),
		"NODE_KEY_FILE=" + nodePath,
		"NODE_KEY_SHA256=" + fileDigest(nodeJSON),
	}, expectedIdentityEnvironment(t, validatorJSON, nodeJSON)...)
}

func TestFlyEntrypointValidatesCandidateBeforeChainInitialization(t *testing.T) {
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "validator-home")
			fakeBin := filepath.Join(root, "bin")
			invocationLog := filepath.Join(root, "zeroned-invocations")
			if err := os.MkdirAll(fakeBin, 0o700); err != nil {
				t.Fatal(err)
			}
			installFakeCommand(t, fakeBin, "zeroned", fmt.Sprintf(
				"#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n",
				invocationLog,
			))

			privateKey := deterministicPrivateKey("candidate-private")
			differentPublic := deterministicPrivateKey("different-public").Public().(ed25519.PublicKey)
			validatorJSON := validatorKeyJSON(privateKey, differentPublic)
			nodeJSON := nodeKeyJSON(deterministicPrivateKey("candidate-node"))
			validatorPath := filepath.Join(root, "candidate-validator.json")
			nodePath := filepath.Join(root, "candidate-node.json")
			writeFile(t, validatorPath, validatorJSON, 0o600)
			writeFile(t, nodePath, nodeJSON, 0o600)

			command := exec.Command("bash", flyEntrypoint(network))
			command.Env = cleanEnvironment(append(
				[]string{"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH")},
				runtimeKeyEnvironment(t, home, validatorPath, validatorJSON, nodePath, nodeJSON)...,
			)...)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("expected inconsistent key rejection, output:\n%s", output)
			}
			if !strings.Contains(string(output), "public key does not match the private key") {
				t.Fatalf("unexpected rejection:\n%s", output)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("validator home was mutated before candidate validation: %v", err)
			}
			if _, err := os.Stat(invocationLog); !os.IsNotExist(err) {
				t.Fatalf("zeroned ran before candidate validation: %v", err)
			}
		})
	}
}

func TestFlyEntrypointRejectsIdentityOutsideReviewedManifestBeforeInitialization(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "validator-home")
	fakeBin := filepath.Join(root, "bin")
	invocationLog := filepath.Join(root, "zeroned-invocations")
	if err := os.MkdirAll(fakeBin, 0o700); err != nil {
		t.Fatal(err)
	}
	installFakeCommand(t, fakeBin, "zeroned", fmt.Sprintf(
		"#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n",
		invocationLog,
	))

	validatorPrivate := deterministicPrivateKey("manifest-validator")
	validatorJSON := validatorKeyJSON(
		validatorPrivate,
		validatorPrivate.Public().(ed25519.PublicKey),
	)
	nodeJSON := nodeKeyJSON(deterministicPrivateKey("manifest-node"))
	validatorPath := filepath.Join(root, "candidate-validator.json")
	nodePath := filepath.Join(root, "candidate-node.json")
	writeFile(t, validatorPath, validatorJSON, 0o600)
	writeFile(t, nodePath, nodeJSON, 0o600)
	runtimeEnvironment := runtimeKeyEnvironment(
		t,
		home,
		validatorPath,
		validatorJSON,
		nodePath,
		nodeJSON,
	)
	for index, entry := range runtimeEnvironment {
		if strings.HasPrefix(entry, "EXPECTED_NODE_ID=") {
			runtimeEnvironment[index] = "EXPECTED_NODE_ID=" + strings.Repeat("0", 40)
		}
	}

	command := exec.Command("bash", flyEntrypoint("mainnet"))
	command.Env = cleanEnvironment(append(
		[]string{"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH")},
		runtimeEnvironment...,
	)...)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected reviewed node identity rejection, output:\n%s", output)
	}
	if !strings.Contains(string(output), "derived P2P node ID does not match the reviewed identity manifest") {
		t.Fatalf("unexpected rejection:\n%s", output)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("validator home was mutated before identity-manifest validation: %v", err)
	}
	if _, err := os.Stat(invocationLog); !os.IsNotExist(err) {
		t.Fatalf("zeroned ran before identity-manifest validation: %v", err)
	}
}

func TestFlyEntrypointInstallsValidatedFreshIdentityUnit(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "validator-home")
	seedDir := filepath.Join(root, "seed")
	fakeBin := filepath.Join(root, "bin")
	invocationLog := filepath.Join(root, "zeroned-invocations")
	if err := os.MkdirAll(fakeBin, 0o700); err != nil {
		t.Fatal(err)
	}
	installFakeCommand(t, fakeBin, "zeroned", fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
if [ "$1" = "init" ]; then
  shift
  home=
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--home" ]; then
      home="$2"
      shift 2
    else
      shift
    fi
  done
  mkdir -p "$home/config" "$home/data"
  printf 'generated-validator' > "$home/config/priv_validator_key.json"
  printf 'generated-node' > "$home/config/node_key.json"
  printf '%%s\n' '{"height":"0","round":0,"step":0}' > "$home/data/priv_validator_state.json"
  printf 'addr_book_strict = true\nallow_duplicate_ip = false\n' > "$home/config/config.toml"
  printf 'minimum-gas-prices = "0.025uzrn"\n' > "$home/config/app.toml"
fi
`, invocationLog))
	installFakeCommand(t, fakeBin, "sed", "#!/bin/sh\nexit 0\n")

	validatorPrivate := deterministicPrivateKey("fresh-validator")
	validatorJSON := validatorKeyJSON(
		validatorPrivate,
		validatorPrivate.Public().(ed25519.PublicKey),
	)
	nodeJSON := nodeKeyJSON(deterministicPrivateKey("fresh-node"))
	validatorPath := filepath.Join(root, "candidate-validator.json")
	nodePath := filepath.Join(root, "candidate-node.json")
	genesisJSON := []byte(`{"chain_id":"fresh-chain"}`)
	writeFile(t, validatorPath, validatorJSON, 0o600)
	writeFile(t, nodePath, nodeJSON, 0o600)
	writeFile(t, filepath.Join(seedDir, "genesis.json"), genesisJSON, 0o600)

	command := exec.Command("bash", "fly-validator-entrypoint-common.sh")
	command.Env = cleanEnvironment(append(
		[]string{
			"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
			"ZERONE_CHAIN_ID=fresh-chain",
			"ZERONE_SEED=" + seedDir,
			"ZERONE_DEFAULT_MONIKER=fresh-validator",
			"ZERONE_GENESIS_SHA256=" + fileDigest(genesisJSON),
		},
		runtimeKeyEnvironment(t, home, validatorPath, validatorJSON, nodePath, nodeJSON)...,
	)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh bootstrap failed: %v\n%s", err, output)
	}
	for path, expected := range map[string][]byte{
		filepath.Join(home, "config", "priv_validator_key.json"): validatorJSON,
		filepath.Join(home, "config", "node_key.json"):           nodeJSON,
		filepath.Join(home, "config", "genesis.json"):            genesisJSON,
	} {
		actual, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != string(expected) {
			t.Fatalf("%s mismatch\nwant: %s\ngot: %s", path, expected, actual)
		}
	}
	state, err := os.ReadFile(filepath.Join(home, "data", "priv_validator_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(state)) != `{"height":"0","round":0,"step":0}` {
		t.Fatalf("unexpected initial signing state: %s", state)
	}
	invocations, err := os.ReadFile(invocationLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(invocations), "init fresh-validator") ||
		!strings.Contains(string(invocations), "start --home "+home) {
		t.Fatalf("unexpected zeroned lifecycle:\n%s", invocations)
	}
}

func TestFlyEntrypointRequiresPersistedSignerKeyAndStateAsAUnit(t *testing.T) {
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "validator-home")
			fakeBin := filepath.Join(root, "bin")
			invocationLog := filepath.Join(root, "zeroned-invocations")
			if err := os.MkdirAll(fakeBin, 0o700); err != nil {
				t.Fatal(err)
			}
			installFakeCommand(t, fakeBin, "zeroned", fmt.Sprintf(
				"#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n",
				invocationLog,
			))

			validatorPrivate := deterministicPrivateKey("persisted-validator")
			validatorJSON := validatorKeyJSON(
				validatorPrivate,
				validatorPrivate.Public().(ed25519.PublicKey),
			)
			nodeJSON := nodeKeyJSON(deterministicPrivateKey("persisted-node"))
			validatorPath := filepath.Join(home, "config", "priv_validator_key.json")
			nodePath := filepath.Join(home, "config", "node_key.json")
			writeFile(
				t,
				filepath.Join(home, "config", "genesis.json"),
				networkGenesis(t, network),
				0o600,
			)
			writeFile(t, filepath.Join(home, "config", "config.toml"), []byte{}, 0o600)
			writeFile(t, filepath.Join(home, "config", "app.toml"), []byte{}, 0o600)
			writeFile(t, validatorPath, validatorJSON, 0o600)
			writeFile(t, nodePath, nodeJSON, 0o600)
			if err := os.MkdirAll(filepath.Join(home, "data"), 0o700); err != nil {
				t.Fatal(err)
			}

			command := exec.Command("bash", flyEntrypoint(network))
			command.Env = cleanEnvironment(append(
				[]string{"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH")},
				runtimeKeyEnvironment(t, home, validatorPath, validatorJSON, nodePath, nodeJSON)...,
			)...)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("expected incomplete signer unit rejection, output:\n%s", output)
			}
			if !strings.Contains(string(output), "persisted validator signing state must be a regular non-symlink file") {
				t.Fatalf("unexpected rejection:\n%s", output)
			}
			if _, err := os.Stat(invocationLog); !os.IsNotExist(err) {
				t.Fatalf("zeroned ran despite incomplete persisted signer unit: %v", err)
			}
		})
	}
}

func TestFlyEntrypointScrubsRuntimeKeyEnvironmentBeforeDaemonExec(t *testing.T) {
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "validator-home")
			fakeBin := filepath.Join(root, "bin")
			environmentLog := filepath.Join(root, "daemon-environment")
			argumentsLog := filepath.Join(root, "daemon-arguments")
			if err := os.MkdirAll(fakeBin, 0o700); err != nil {
				t.Fatal(err)
			}
			installFakeCommand(t, fakeBin, "zeroned", fmt.Sprintf(
				"#!/bin/sh\nenv > %q\nprintf '%%s\\n' \"$*\" > %q\n",
				environmentLog,
				argumentsLog,
			))
			// The runtime scripts use GNU sed flags inside Debian. A no-op is
			// sufficient here because the config already has the desired value.
			installFakeCommand(t, fakeBin, "sed", "#!/bin/sh\nexit 0\n")

			validatorPrivate := deterministicPrivateKey("running-validator")
			validatorJSON := validatorKeyJSON(
				validatorPrivate,
				validatorPrivate.Public().(ed25519.PublicKey),
			)
			nodeJSON := nodeKeyJSON(deterministicPrivateKey("running-node"))
			validatorPath := filepath.Join(home, "config", "priv_validator_key.json")
			nodePath := filepath.Join(home, "config", "node_key.json")
			writeFile(
				t,
				filepath.Join(home, "config", "genesis.json"),
				networkGenesis(t, network),
				0o600,
			)
			writeFile(t, filepath.Join(home, "config", "config.toml"), []byte{}, 0o600)
			writeFile(
				t,
				filepath.Join(home, "config", "app.toml"),
				[]byte("minimum-gas-prices = \"0.025uzrn\"\n"),
				0o600,
			)
			writeFile(t, validatorPath, validatorJSON, 0o600)
			writeFile(t, nodePath, nodeJSON, 0o600)
			writeFile(
				t,
				filepath.Join(home, "data", "priv_validator_state.json"),
				[]byte(`{"height":"41","round":0,"step":3}`),
				0o600,
			)

			base64ValidatorSecret := base64.StdEncoding.EncodeToString(validatorJSON)
			base64NodeSecret := base64.StdEncoding.EncodeToString(nodeJSON)
			command := exec.Command("bash", flyEntrypoint(network))
			command.Env = cleanEnvironment(append([]string{
				"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
				"ZERONE_TEST_ENV_LOG=" + environmentLog,
				"PRIV_VALIDATOR_KEY_B64=" + base64ValidatorSecret,
				"PRIV_VALIDATOR_KEY_SHA256=" + fileDigest(validatorJSON),
				"NODE_KEY_B64=" + base64NodeSecret,
				"NODE_KEY_SHA256=" + fileDigest(nodeJSON),
				"ZERONE_HOME=" + home,
			}, expectedIdentityEnvironment(t, validatorJSON, nodeJSON)...)...)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("entrypoint failed: %v\n%s", err, output)
			}
			daemonEnvironment, err := os.ReadFile(environmentLog)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{
				"PRIV_VALIDATOR_KEY_",
				"NODE_KEY_",
				"EXPECTED_VALIDATOR_",
				"EXPECTED_NODE_",
				"EXPECTED_PRIV_VALIDATOR_",
				base64ValidatorSecret,
				base64NodeSecret,
			} {
				if strings.Contains(string(daemonEnvironment), forbidden) {
					t.Fatalf("daemon environment retained %q:\n%s", forbidden, daemonEnvironment)
				}
			}
			daemonArguments, err := os.ReadFile(argumentsLog)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(daemonArguments), "start --home "+home) {
				t.Fatalf("unexpected daemon arguments: %s", daemonArguments)
			}
		})
	}
}

func TestFlyExactHeightHandoffBindsPlanAndHMinusOneAppHash(t *testing.T) {
	root := t.TempDir()
	fakeBin := filepath.Join(root, "bin")
	flyLog := filepath.Join(root, "fly-invocations")
	if err := os.MkdirAll(fakeBin, 0o700); err != nil {
		t.Fatal(err)
	}
	installFakeCommand(t, fakeBin, "sleep", "#!/bin/sh\nexit 0\n")
	machineID := "8629d6be267178"
	volumeID := "vol_abc123"
	currentImageDigest := strings.Repeat("d", 64)
	imageDigest := strings.Repeat("c", 64)
	appHash := strings.Repeat("b", 64)
	upgradeAppHash := strings.Repeat("e", 64)
	validatorAddress := strings.Repeat("A", 40)
	nodeID := strings.Repeat("1", 40)
	validatorKeyDigest := strings.Repeat("2", 64)
	nodeKeyDigest := strings.Repeat("3", 64)
	genesisDigest := strings.Repeat("9", 64)
	planEvidence := unifiedPlanDocument(t)
	upgradeInfo := unifiedUpgradeInfoDocument(t)
	installFakeCommand(t, fakeBin, "fly", fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
if [ "$1" = "config" ] && [ "$2" = "show" ]; then
  printf '%%s\n' '{"app":"zerone-validator","env":{"ZERONE_HOME":"/data/.zeroned"},"mounts":[{"source":"zerone_data","destination":"/data"}],"services":[{"protocol":"tcp","internal_port":26656,"ports":[{"port":26656}]}]}'
elif [ "$1" = "config" ] && [ "$2" = "validate" ]; then
  exit 0
elif [ "$1" = "secrets" ] && [ "$2" = "list" ]; then
  if [ "${FAKE_NO_SECRETS:-}" = "1" ]; then
    printf '[]\n'
  else
    printf '%%s\n' '[{"name":"PRIV_VALIDATOR_KEY_B64","digest":"one","status":"Deployed"},{"name":"PRIV_VALIDATOR_KEY_SHA256","digest":"two","status":"Deployed"},{"name":"NODE_KEY_B64","digest":"three","status":"Deployed"},{"name":"NODE_KEY_SHA256","digest":"four","status":"Deployed"}]'
  fi
elif [ "$1" = "machine" ] && [ "$2" = "list" ]; then
  state=stopped
  digest=%s
  env='{}'
  services='[{"protocol":"tcp","internal_port":26657,"ports":[{"port":26657}]},{"protocol":"tcp","internal_port":1317,"ports":[{"port":1317}]},{"protocol":"tcp","internal_port":26656,"ports":[{"port":26656}]},{"protocol":"tcp","internal_port":9090,"ports":[{"port":9090}]}]'
  if [ "${FAKE_CONFIG_DRIFT:-}" = "1" ]; then
    env='{"UNREVIEWED_CONFIG_DRIFT":"true"}'
  fi
  if grep -q '^machine update ' %q; then
    digest=%s
    env='{"EXPECTED_VALIDATOR_ADDRESS":"%s","EXPECTED_NODE_ID":"%s","EXPECTED_PRIV_VALIDATOR_KEY_SHA256":"%s","EXPECTED_NODE_KEY_SHA256":"%s"}'
    services='[{"protocol":"tcp","internal_port":26656,"ports":[{"port":26656}]}]'
  fi
  if grep -q '^machine start ' %q; then
    state=started
  fi
  if [ "${FAKE_BAD_STARTED_CONFIG:-}" = "1" ] && grep -q '^machine start ' %q; then
    services='[{"protocol":"tcp","internal_port":26656,"ports":[{"port":443}]}]'
  fi
  if grep -q '^machine stop ' %q; then
    state=stopped
  fi
  printf '{"id":"%s","state":"%%s","image_ref":{"registry":"registry.fly.io","repository":"zerone","digest":"sha256:%%s"},"config":{"env":%%s,"restart":{"policy":"no"},"mounts":[{"encrypted":true,"path":"/data","volume":"%s"}],"services":%%s}}\n' "$state" "$digest" "$env" "$services" | jq -s .
fi
`, flyLog, currentImageDigest, flyLog, imageDigest, validatorAddress, nodeID, validatorKeyDigest, nodeKeyDigest, flyLog, flyLog, flyLog, machineID, volumeID))
	installFakeCommand(t, fakeBin, "curl", fmt.Sprintf(`#!/bin/sh
for argument do
  url="$argument"
done
case "$url" in
  */status)
    if grep -q '^machine start ' %q; then
      printf '%%s\n' '{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"101","latest_app_hash":"%s"}}}'
    else
      printf '%%s\n' '{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"99","latest_app_hash":"%s"}}}'
    fi
    ;;
  */commit?height=101)
    printf '%%s\n' '{"result":{"signed_header":{"header":{"chain_id":"zerone-1","height":"101","app_hash":"%s"}}}}'
    ;;
  */cosmos/base/tendermint/v1beta1/node_info)
    printf '%%s\n' '{"default_node_info":{"network":"zerone-1"}}'
    ;;
  */cosmos/upgrade/v1beta1/current_plan)
    printf '%%s\n' %q
    ;;
  */cosmos/upgrade/v1beta1/applied_plan/sdk-0.53-ibc-10)
    printf '%%s\n' '{"height":"100"}'
    ;;
  *)
    exit 22
    ;;
esac
`, flyLog, upgradeAppHash, appHash, upgradeAppHash, string(planEvidence)))

	configPath := filepath.Join(root, "fly.toml")
	configContents := []byte("app = \"zerone-validator\"\n")
	writeFile(t, configPath, configContents, 0o600)
	configDigest := fileDigest(configContents)
	planEvidencePath := filepath.Join(root, "current-plan.json")
	writeFile(t, planEvidencePath, planEvidence, 0o600)
	planDigest := fileDigest(planEvidence)
	preflightWithoutDigest := []byte(fmt.Sprintf(
		`{"schema":"zerone.activation-preflight/v3","scope":"scheduled-plan-h-minus-one","activation_ready":true,"chain_id":"zerone-1","genesis_sha256":"%s","upgrade_info_sha256":"%s","plan_name":"sdk-0.53-ibc-10","plan_height":100,"plan_info_sha256":"%s","blocks_until_activation":1,"unsafe_skip_upgrade_heights":[],"unsafe_skip_config_sha256":"%s","source_data_manifest_sha256":"%s","source_data_file_count":2,"source_data_bytes":4096,"completed_checks":["complete_iavl_roots_bound_to_app_hash","exact_safety_source_versions","sdk_governance_emergency_authority_audit","zero_unattributed_custom_upgrade_stake","effective_unsafe_skip_configuration_bound","scheduled_plan_exact_h_minus_one","named_handler_plan_specific_preconditions","scheduled_height_not_unsafe_skipped","exact_upgrade_handler_cache_dry_run","source_database_never_opened","isolated_copy_manifest_exact","source_manifest_unchanged_after_verification","expected_chain_height_app_hash_tuple_matched","genesis_chain_id_and_digest_bound","local_upgrade_info_exactly_matches_committed_plan"],"height":99,"app_hash":"%s","safety_source_versions":{"emergency":1,"gov":5,"zerone_gov":2},"custom_gov_unattributed_stake_uzrn":"0"}`,
		genesisDigest,
		fileDigest(upgradeInfo),
		fileDigest([]byte(unifiedPlanInfo)),
		strings.Repeat("6", 64),
		strings.Repeat("7", 64),
		appHash,
	))
	embeddedPreflightDigest := fileDigest(preflightWithoutDigest)
	preflightReport := []byte(strings.Replace(
		string(preflightWithoutDigest),
		`"upgrade_info_sha256":"`+fileDigest(upgradeInfo)+`",`,
		`"upgrade_info_sha256":"`+fileDigest(upgradeInfo)+`","report_sha256":"`+
			embeddedPreflightDigest+`",`,
		1,
	))
	preflightPath := filepath.Join(root, "activation-preflight.json")
	writeFile(t, preflightPath, preflightReport, 0o600)
	preflightFileDigest := fileDigest(preflightReport)
	armEvidencePath := filepath.Join(root, "arm-evidence.json")
	armedMachineConfig := []byte(fmt.Sprintf(
		`{"env":{},"mounts":[{"encrypted":true,"path":"/data","volume":"%s"}],"services":[{"internal_port":26657,"ports":[{"port":26657}],"protocol":"tcp"},{"internal_port":1317,"ports":[{"port":1317}],"protocol":"tcp"},{"internal_port":26656,"ports":[{"port":26656}],"protocol":"tcp"},{"internal_port":9090,"ports":[{"port":9090}],"protocol":"tcp"}]}`,
		volumeID,
	))
	armEvidence := []byte(fmt.Sprintf(
		`{"schema":"zerone.fly-upgrade-arm-evidence/v1","fly_app":"zerone-validator","machine_id":"%s","volume_id":"%s","chain_id":"zerone-1","current_image_ref":"registry.fly.io/zerone@sha256:%s","upgrade_name":"sdk-0.53-ibc-10","upgrade_height":"100","upgrade_plan_sha256":"%s","pre_arm_height":"98","pre_arm_app_hash":"%s","post_arm_height":"99","post_arm_app_hash":"%s","machine_config_sha256":"%s","restart_policy":"no","validator_address":"%s","node_id":"%s","validator_key_sha256":"%s","node_key_sha256":"%s","genesis_sha256":"%s","armed_by":"independent-operator","armed_at":"2026-07-30T00:00:00Z"}`,
		machineID,
		volumeID,
		currentImageDigest,
		planDigest,
		strings.Repeat("a", 64),
		appHash,
		fileDigest(armedMachineConfig),
		validatorAddress,
		nodeID,
		validatorKeyDigest,
		nodeKeyDigest,
		genesisDigest,
	))
	writeFile(t, armEvidencePath, armEvidence, 0o600)
	armEvidenceDigest := fileDigest(armEvidence)
	exitLogPath := filepath.Join(root, "old-binary.log")
	exitLog := []byte(fmt.Sprintf(
		`UPGRADE "%s" NEEDED at height: 100: %s`,
		unifiedUpgradeName,
		unifiedPlanInfo,
	))
	writeFile(t, exitLogPath, exitLog, 0o600)
	upgradeInfoPath := filepath.Join(root, "upgrade-info.json")
	writeFile(t, upgradeInfoPath, upgradeInfo, 0o600)
	exitEvidencePath := filepath.Join(root, "exit-evidence.json")
	exitEvidence := []byte(fmt.Sprintf(
		`{"schema":"zerone.fly-upgrade-exit-evidence/v1","fly_app":"zerone-validator","machine_id":"%s","volume_id":"%s","chain_id":"zerone-1","current_image_ref":"registry.fly.io/zerone@sha256:%s","upgrade_name":"sdk-0.53-ibc-10","upgrade_height":"100","upgrade_plan_sha256":"%s","activation_preflight_report_sha256":"%s","arm_evidence_sha256":"%s","last_committed_height":"99","last_committed_app_hash":"%s","attempted_upgrade_height":"100","validator_address":"%s","node_id":"%s","validator_key_sha256":"%s","node_key_sha256":"%s","genesis_sha256":"%s","old_binary_exit_count":1,"old_binary_exit_log_sha256":"%s","upgrade_info_sha256":"%s","observer":"independent-operator","observed_at":"2026-07-30T00:00:00Z"}`,
		machineID,
		volumeID,
		currentImageDigest,
		planDigest,
		preflightFileDigest,
		armEvidenceDigest,
		appHash,
		validatorAddress,
		nodeID,
		validatorKeyDigest,
		nodeKeyDigest,
		genesisDigest,
		fileDigest(exitLog),
		fileDigest(upgradeInfo),
	))
	writeFile(t, exitEvidencePath, exitEvidence, 0o600)
	exitEvidenceDigest := fileDigest(exitEvidence)
	confirmation := strings.Join([]string{
		"zerone-validator",
		machineID,
		volumeID,
		"zerone-1",
		currentImageDigest,
		imageDigest,
		configDigest,
		unifiedUpgradeName,
		"100",
		planDigest,
		preflightFileDigest,
		armEvidenceDigest,
		exitEvidenceDigest,
		"99",
		appHash,
		"100",
		upgradeAppHash,
		validatorAddress,
		nodeID,
		validatorKeyDigest,
		nodeKeyDigest,
		genesisDigest,
	}, ":")
	command := exec.Command("bash", "fly-exact-height-handoff.sh")
	command.Env = cleanEnvironment(
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FLY_APP=zerone-validator",
		"FLY_MACHINE_ID="+machineID,
		"FLY_VOLUME_ID="+volumeID,
		"FLY_CURRENT_IMAGE_REF=registry.fly.io/zerone@sha256:"+currentImageDigest,
		"FLY_IMAGE_REF=registry.fly.io/zerone@sha256:"+imageDigest,
		"FLY_CONFIG_PATH="+configPath,
		"FLY_CONFIG_SHA256="+configDigest,
		"CHAIN_ID=zerone-1",
		"UPGRADE_NAME="+unifiedUpgradeName,
		"UPGRADE_HEIGHT=100",
		"UPGRADE_PLAN_EVIDENCE_PATH="+planEvidencePath,
		"UPGRADE_PLAN_SHA256="+planDigest,
		"ACTIVATION_PREFLIGHT_REPORT_PATH="+preflightPath,
		"ACTIVATION_PREFLIGHT_REPORT_SHA256="+preflightFileDigest,
		"UPGRADE_ARM_EVIDENCE_PATH="+armEvidencePath,
		"UPGRADE_ARM_EVIDENCE_SHA256="+armEvidenceDigest,
		"UPGRADE_EXIT_EVIDENCE_PATH="+exitEvidencePath,
		"UPGRADE_EXIT_EVIDENCE_SHA256="+exitEvidenceDigest,
		"OLD_BINARY_EXIT_LOG_PATH="+exitLogPath,
		"UPGRADE_INFO_EVIDENCE_PATH="+upgradeInfoPath,
		"LAST_COMMITTED_HEIGHT=99",
		"LAST_COMMITTED_APP_HASH="+appHash,
		"ATTEMPTED_UPGRADE_HEIGHT=100",
		"EXPECTED_UPGRADE_APP_HASH="+upgradeAppHash,
		"EXPECTED_VALIDATOR_ADDRESS="+validatorAddress,
		"EXPECTED_NODE_ID="+nodeID,
		"EXPECTED_PRIV_VALIDATOR_KEY_SHA256="+validatorKeyDigest,
		"EXPECTED_NODE_KEY_SHA256="+nodeKeyDigest,
		"EXPECTED_GENESIS_SHA256="+genesisDigest,
		"OBSERVER_RPC_URL=https://observer-rpc.example",
		"OBSERVER_API_URL=https://observer-api.example",
		"FLY_HANDOFF_CONFIRMATION="+confirmation,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("handoff failed: %v\n%s", err, output)
	}
	invocations, err := os.ReadFile(flyLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"machine update " + machineID,
		"machine start " + machineID,
		"secrets list",
	} {
		if !strings.Contains(string(invocations), expected) {
			t.Fatalf("missing %q in fly invocations:\n%s", expected, invocations)
		}
	}
	if strings.Contains(string(invocations), "machine stop ") {
		t.Fatalf("verified handoff unexpectedly triggered fail-stop:\n%s", invocations)
	}

	t.Run("stopped machine config drift fails before target mutation", func(t *testing.T) {
		if err := os.Remove(flyLog); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		rejected := exec.Command("bash", "fly-exact-height-handoff.sh")
		rejected.Env = append(command.Env, "FAKE_CONFIG_DRIFT=1")
		rejectedOutput, rejectedErr := rejected.CombinedOutput()
		if rejectedErr == nil {
			t.Fatalf(
				"expected stopped Machine config drift rejection, output:\n%s",
				rejectedOutput,
			)
		}
		if !strings.Contains(
			string(rejectedOutput),
			"config drifted from the exact config sealed by pre-H arm evidence",
		) {
			t.Fatalf(
				"unexpected config drift rejection:\n%s",
				rejectedOutput,
			)
		}
		invocations, err := os.ReadFile(flyLog)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(
			string(invocations),
			"machine list --app zerone-validator --json",
		) {
			t.Fatalf(
				"config drift test did not inspect the stopped Machine:\n%s",
				invocations,
			)
		}
		for _, mutation := range []string{
			"machine update ",
			"machine start ",
			"machine stop ",
		} {
			if strings.Contains(string(invocations), mutation) {
				t.Fatalf(
					"Fly mutation %q occurred after stopped config drift:\n%s",
					mutation,
					invocations,
				)
			}
		}
	})

	t.Run("started tuple mismatch fail-stops signer", func(t *testing.T) {
		if err := os.Remove(flyLog); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		rejected := exec.Command("bash", "fly-exact-height-handoff.sh")
		rejected.Env = append(command.Env, "FAKE_BAD_STARTED_CONFIG=1")
		rejectedOutput, rejectedErr := rejected.CombinedOutput()
		if rejectedErr == nil {
			t.Fatalf("expected started tuple rejection, output:\n%s", rejectedOutput)
		}
		invocations, err := os.ReadFile(flyLog)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(
			string(invocations),
			"machine stop "+machineID+" --app zerone-validator",
		) {
			t.Fatalf("post-start failure did not stop signer:\n%s", invocations)
		}
	})

	for name, mutation := range map[string]func([]string) []string{
		"last committed is H": func(environment []string) []string {
			return append(environment, "LAST_COMMITTED_HEIGHT=100")
		},
		"attempted height is H-1": func(environment []string) []string {
			return append(environment, "ATTEMPTED_UPGRADE_HEIGHT=99")
		},
		"plan evidence is not a digest": func(environment []string) []string {
			return append(environment, "UPGRADE_PLAN_SHA256=unbound")
		},
		"activation preflight is not digest-bound": func(environment []string) []string {
			return append(environment, "ACTIVATION_PREFLIGHT_REPORT_SHA256=unbound")
		},
		"app hash is not a digest": func(environment []string) []string {
			return append(environment, "LAST_COMMITTED_APP_HASH=unbound")
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.Remove(flyLog); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			invalid := mutation(command.Env)
			rejected := exec.Command("bash", "fly-exact-height-handoff.sh")
			rejected.Env = invalid
			rejectedOutput, rejectedErr := rejected.CombinedOutput()
			if rejectedErr == nil {
				t.Fatalf("expected evidence rejection, output:\n%s", rejectedOutput)
			}
			if _, err := os.Stat(flyLog); !os.IsNotExist(err) {
				t.Fatalf("fly was invoked before evidence validation: %v", err)
			}
		})
	}

	t.Run("missing runtime identity secrets fail before target mutation", func(t *testing.T) {
		if err := os.Remove(flyLog); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		rejected := exec.Command("bash", "fly-exact-height-handoff.sh")
		rejected.Env = append(command.Env, "FAKE_NO_SECRETS=1")
		rejectedOutput, rejectedErr := rejected.CombinedOutput()
		if rejectedErr == nil {
			t.Fatalf("expected secret preflight rejection, output:\n%s", rejectedOutput)
		}
		invocations, err := os.ReadFile(flyLog)
		if err != nil {
			t.Fatal(err)
		}
		for _, mutation := range []string{"machine stop ", "machine update ", "machine start "} {
			if strings.Contains(string(invocations), mutation) {
				t.Fatalf("destructive Fly action %q occurred before secret preflight:\n%s", mutation, invocations)
			}
		}
	})
}

func TestFlyArmExactHeightHandoffDisablesRestartBeforeHWithoutConfigDrift(t *testing.T) {
	root := t.TempDir()
	fakeBin := filepath.Join(root, "bin")
	flyLog := filepath.Join(root, "fly-invocations")
	if err := os.MkdirAll(fakeBin, 0o700); err != nil {
		t.Fatal(err)
	}
	machineID := "8629d6be267178"
	volumeID := "vol_abc123"
	imageDigest := strings.Repeat("d", 64)
	appHash := strings.Repeat("b", 64)
	validatorAddress := strings.Repeat("A", 40)
	nodeID := strings.Repeat("1", 40)
	validatorKeyDigest := strings.Repeat("2", 64)
	nodeKeyDigest := strings.Repeat("3", 64)
	genesisDigest := strings.Repeat("9", 64)
	plan := unifiedPlanDocument(t)
	installFakeCommand(t, fakeBin, "fly", fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
if [ "$1" = "machine" ] && [ "$2" = "list" ]; then
  policy=on-failure
  if grep -q '^machine update ' %q; then
    policy=no
  fi
  printf '{"id":"%s","state":"started","image_ref":{"registry":"registry.fly.io","repository":"zerone","digest":"sha256:%s"},"config":{"env":{"ZERONE_HOME":"/data/.zeroned"},"restart":{"policy":"%%s"},"mounts":[{"encrypted":true,"path":"/data","volume":"%s"}],"services":[{"protocol":"tcp","internal_port":26656}]}}\n' "$policy" | jq -s .
fi
`, flyLog, flyLog, machineID, imageDigest, volumeID))
	installFakeCommand(t, fakeBin, "curl", fmt.Sprintf(`#!/bin/sh
for argument do
  url="$argument"
done
case "$url" in
  */status)
    printf '%%s\n' '{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"90","latest_app_hash":"%s"}}}'
    ;;
  */cosmos/base/tendermint/v1beta1/node_info)
    printf '%%s\n' '{"default_node_info":{"network":"zerone-1"}}'
    ;;
  */cosmos/upgrade/v1beta1/current_plan)
    printf '%%s\n' %q
    ;;
  *)
    exit 22
    ;;
esac
`, appHash, string(plan)))

	planPath := filepath.Join(root, "current-plan.json")
	writeFile(t, planPath, plan, 0o600)
	planDigest := fileDigest(plan)
	evidencePath := filepath.Join(root, "arm-evidence.json")
	confirmation := strings.Join([]string{
		"arm-no-restart",
		"zerone-validator",
		machineID,
		volumeID,
		"zerone-1",
		imageDigest,
		unifiedUpgradeName,
		"100",
		planDigest,
		"90",
		appHash,
		validatorAddress,
		nodeID,
		validatorKeyDigest,
		nodeKeyDigest,
		genesisDigest,
	}, ":")
	command := exec.Command("bash", "fly-arm-exact-height-handoff.sh")
	command.Env = cleanEnvironment(
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FLY_APP=zerone-validator",
		"FLY_MACHINE_ID="+machineID,
		"FLY_VOLUME_ID="+volumeID,
		"FLY_CURRENT_IMAGE_REF=registry.fly.io/zerone@sha256:"+imageDigest,
		"CHAIN_ID=zerone-1",
		"UPGRADE_NAME="+unifiedUpgradeName,
		"UPGRADE_HEIGHT=100",
		"UPGRADE_PLAN_EVIDENCE_PATH="+planPath,
		"UPGRADE_PLAN_SHA256="+planDigest,
		"PRE_ARM_HEIGHT=90",
		"PRE_ARM_APP_HASH="+appHash,
		"EXPECTED_VALIDATOR_ADDRESS="+validatorAddress,
		"EXPECTED_NODE_ID="+nodeID,
		"EXPECTED_PRIV_VALIDATOR_KEY_SHA256="+validatorKeyDigest,
		"EXPECTED_NODE_KEY_SHA256="+nodeKeyDigest,
		"EXPECTED_GENESIS_SHA256="+genesisDigest,
		"OBSERVER_RPC_URL=https://observer-rpc.example",
		"OBSERVER_API_URL=https://observer-api.example",
		"ARMED_BY=independent-operator",
		"ARMED_AT=2026-07-30T00:00:00Z",
		"FLY_ARM_EVIDENCE_OUTPUT="+evidencePath,
		"FLY_ARM_CONFIRMATION="+confirmation,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("arming failed: %v\n%s", err, output)
	}
	invocations, err := os.ReadFile(flyLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(invocations), "machine update "+machineID+" --app zerone-validator --restart no --yes") {
		t.Fatalf("missing exact no-restart update:\n%s", invocations)
	}
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	var observed map[string]any
	if err := json.Unmarshal(evidence, &observed); err != nil {
		t.Fatal(err)
	}
	for field, expected := range map[string]string{
		"schema":               "zerone.fly-upgrade-arm-evidence/v1",
		"restart_policy":       "no",
		"current_image_ref":    "registry.fly.io/zerone@sha256:" + imageDigest,
		"validator_address":    validatorAddress,
		"node_id":              nodeID,
		"validator_key_sha256": validatorKeyDigest,
		"node_key_sha256":      nodeKeyDigest,
		"genesis_sha256":       genesisDigest,
	} {
		if observed[field] != expected {
			t.Fatalf("evidence field %s: want %q, got %#v", field, expected, observed[field])
		}
	}
}

func TestCosmovisorEntrypointRejectsMutableSafetyPolicy(t *testing.T) {
	command := exec.Command("sh", "validator-cosmovisor-entrypoint.sh")
	command.Env = cleanEnvironment(
		"DAEMON_HOME="+filepath.Join(t.TempDir(), ".zeroned"),
		"DAEMON_NAME=zeroned",
		"DAEMON_BINARY_SHA256="+strings.Repeat("a", 64),
		"DAEMON_GENESIS_BINARY_SHA256="+strings.Repeat("a", 64),
		"DAEMON_CURRENT_BINARY_SHA256="+strings.Repeat("a", 64),
		"DAEMON_ALLOW_DOWNLOAD_BINARIES=true",
		"DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true",
		"DAEMON_RESTART_AFTER_UPGRADE=true",
		"UNSAFE_SKIP_BACKUP=false",
	)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("expected mutable policy rejection, output:\n%s", output)
	}
	if !strings.Contains(string(output), "DAEMON_ALLOW_DOWNLOAD_BINARIES must be image-frozen to false") {
		t.Fatalf("unexpected rejection:\n%s", output)
	}
}

func TestValidatorDockerfilesPinBuildAndPackageProvenance(t *testing.T) {
	for _, dockerfile := range []string{
		"../Dockerfile.validator",
		"mainnet/Dockerfile",
		"testnet/Dockerfile",
	} {
		t.Run(dockerfile, func(t *testing.T) {
			bodyBytes, err := os.ReadFile(dockerfile)
			if err != nil {
				t.Fatal(err)
			}
			body := string(bodyBytes)
			for _, required := range []string{
				"golang:1.25.12-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58",
				"debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818",
				"ARG VERSION",
				"ARG COMMIT",
				"ARG SOURCE_DATE_EPOCH",
				`make build VERSION="${VERSION}" COMMIT="${COMMIT}"`,
				`grep -Eq '^[0-9a-f]{40}$'`,
				"org.opencontainers.image.revision",
				"io.zerone.source-date-epoch",
				"snapshot.debian.org/archive/debian/20260713T000000Z",
				"snapshot.debian.org/archive/debian-security/20260713T000000Z",
			} {
				if !strings.Contains(body, required) {
					t.Fatalf("%s does not contain provenance control %q", dockerfile, required)
				}
			}
		})
	}
}
