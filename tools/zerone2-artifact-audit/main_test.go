package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ripemd160"
)

func TestAuditGenesisAcceptsLockedProtocolDarkProfile(t *testing.T) {
	r := auditGenesis(marshalFixture(t, validGenesisFixture(t)))
	if len(r.Issues) != 0 {
		t.Fatalf("valid fixture failed audit:\n%s", formatIssues(r.Issues))
	}
	if r.ValidatorAddress == "" || r.OpsAddress == "" || r.ValidatorAddress == r.OpsAddress {
		t.Fatalf("unexpected audited identities: validator=%q ops=%q", r.ValidatorAddress, r.OpsAddress)
	}
}

func TestAuditGenesisRejectsInvariantDrift(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		value     any
		issuePath string
	}{
		{name: "chain id", path: "chain_id", value: "zerone-1", issuePath: "chain_id"},
		{name: "initial height", path: "initial_height", value: "2", issuePath: "initial_height"},
		{name: "noncanonical initial height", path: "initial_height", value: "+1", issuePath: "initial_height"},
		{name: "block bytes", path: "consensus.params.block.max_bytes", value: "4194305", issuePath: "max_bytes"},
		{name: "vote extensions", path: "consensus.params.abci.vote_extensions_enable_height", value: "1", issuePath: "vote_extensions_enable_height"},
		{name: "SDK validator count", path: "app_state.staking.params.max_validators", value: 34, issuePath: "staking.params.max_validators"},
		{name: "SDK min commission", path: "app_state.staking.params.min_commission_rate", value: "0.040000000000000000", issuePath: "min_commission_rate"},
		{name: "governance deposit", path: "app_state.gov.params.min_deposit", value: []any{coin(denom, "99999999")}, issuePath: "gov.params.min_deposit"},
		{name: "governance veto burn", path: "app_state.gov.params.burn_vote_veto", value: false, issuePath: "burn_vote_veto"},
		{name: "supply", path: "app_state.bank.supply", value: []any{coin("uzrn", "13555000001")}, issuePath: "bank.supply"},
		{name: "extra allocation", path: "app_state.bank.balances", value: []any{}, issuePath: "bank.balances"},
		{name: "gentx count", path: "app_state.genutil.gen_txs", value: []any{}, issuePath: "gen_txs"},
		{name: "knowledge verifier floor", path: "app_state.knowledge.params.min_verifiers", value: "2", issuePath: "min_verifiers"},
		{name: "IBC wildcard client", path: "app_state.ibc.client_genesis.params.allowed_clients", value: []any{"*"}, issuePath: "allowed_clients"},
		{name: "IBC missing localhost", path: "app_state.ibc.client_genesis.params.allowed_clients", value: []any{}, issuePath: "allowed_clients"},
		{name: "IBC external client", path: "app_state.ibc.client_genesis.params.allowed_clients", value: []any{"09-localhost", "07-tendermint"}, issuePath: "allowed_clients"},
		{name: "transfer send", path: "app_state.transfer.params.send_enabled", value: true, issuePath: "send_enabled"},
		{name: "ICA controller", path: "app_state.interchainaccounts.controller_genesis_state.params.controller_enabled", value: true, issuePath: "controller_enabled"},
		{name: "bridge adapter", path: "app_state.substrate_bridge.adapters", value: []any{map[string]any{"adapter_id": "unexpected"}}, issuePath: "substrate_bridge.adapters"},
		{name: "bridge settlement ratio", path: "app_state.substrate_bridge.params.min_verified_ratio_for_settle_bps", value: 1000, issuePath: "min_verified_ratio_for_settle_bps"},
		{name: "review gate", path: "app_state.knowledge.params.min_review_fee", value: "100000", issuePath: "min_review_fee"},
		{name: "demand enabled", path: "app_state.knowledge.params.demand_tracking_enabled", value: true, issuePath: "demand_tracking_enabled"},
		{name: "training rewards", path: "app_state.knowledge.params.training_fund_base_reward", value: "1", issuePath: "training_fund_base_reward"},
		{name: "block reward", path: "app_state.vesting_rewards.params.block_reward", value: "1", issuePath: "block_reward"},
		{name: "founder", path: "app_state.vesting_rewards.params.founder_share_bps", value: 1, issuePath: "founder_share_bps"},
		{name: "vesting", path: "app_state.vesting_rewards.params.vesting_enabled", value: true, issuePath: "vesting_enabled"},
		{name: "registrar", path: "app_state.claiming_pot.params.bootstrap_registrar", value: "zrn1operator", issuePath: "bootstrap_registrar"},
		{name: "emergency council", path: "app_state.emergency.params.genesis_council", value: []any{"zrn1operator"}, issuePath: "genesis_council"},
		{name: "custom staking minimum", path: "app_state.zerone_staking.params.min_self_delegation", value: "111000", issuePath: "min_self_delegation"},
		{name: "custom validator preloaded", path: "app_state.zerone_staking.validators", value: []any{map[string]any{"operator_address": "zrn1operator"}}, issuePath: "zerone_staking.validators"},
		{name: "alignment enabled", path: "app_state.alignment.params.enabled", value: true, issuePath: "alignment.params.enabled"},
		{name: "counterexamples enabled", path: "app_state.counterexamples.params.proposals_enabled", value: true, issuePath: "proposals_enabled"},
		{name: "liquidity reachable", path: "app_state.liquiditypool.params.min_initial_liquidity", value: totalSupply, issuePath: "min_initial_liquidity"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := validGenesisFixture(t)
			setFixturePath(t, fixture, tc.path, tc.value)
			r := auditGenesis(marshalFixture(t, fixture))
			if !hasIssuePath(r.Issues, tc.issuePath) {
				t.Fatalf("expected issue containing %q; got:\n%s", tc.issuePath, formatIssues(r.Issues))
			}
		})
	}
}

func TestAuditGenesisRejectsUnlockedOrMisdirectedValidator(t *testing.T) {
	t.Run("unlocked account", func(t *testing.T) {
		fixture := validGenesisFixture(t)
		accounts := fixtureAppState(t, fixture)["auth"].(map[string]any)["accounts"].([]any)
		accounts[0] = map[string]any{
			"@type":          "/cosmos.auth.v1beta1.BaseAccount",
			"address":        validatorAddressForFixture(t),
			"pub_key":        nil,
			"account_number": "0",
			"sequence":       "0",
		}
		r := auditGenesis(marshalFixture(t, fixture))
		if !hasIssuePath(r.Issues, "PermanentLockedAccount") && !hasIssueMessage(r.Issues, "PermanentLockedAccount") {
			t.Fatalf("expected locked-account rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("wrong locked amount", func(t *testing.T) {
		fixture := validGenesisFixture(t)
		setFixturePath(t, fixture, "app_state.auth.accounts", mutateLockedAmount(t, fixture, "11110999999"))
		r := auditGenesis(marshalFixture(t, fixture))
		if !hasIssuePath(r.Issues, "original_vesting") {
			t.Fatalf("expected locked amount rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("gentx valoper mismatch", func(t *testing.T) {
		fixture := validGenesisFixture(t)
		setFixturePath(t, fixture, "app_state.genutil.gen_txs", mutateGentxValoper(t, fixture, bech32Encode(t, "zrnvaloper", bytes20(0x99))))
		r := auditGenesis(marshalFixture(t, fixture))
		if !hasIssuePath(r.Issues, "validator_address") {
			t.Fatalf("expected valoper mismatch rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("gentx signer mismatch", func(t *testing.T) {
		fixture := validGenesisFixture(t)
		txs := fixtureAppState(t, fixture)["genutil"].(map[string]any)["gen_txs"].([]any)
		tx := txs[0].(map[string]any)
		signer := tx["auth_info"].(map[string]any)["signer_infos"].([]any)[0].(map[string]any)
		signer["public_key"].(map[string]any)["key"] = base64.StdEncoding.EncodeToString(append([]byte{2}, bytes32(0x71)...))
		r := auditGenesis(marshalFixture(t, fixture))
		if !hasIssueMessage(r.Issues, "does not derive") {
			t.Fatalf("expected signer identity rejection; got:\n%s", formatIssues(r.Issues))
		}
	})
}

func TestAuditGenesisRejectsPrivateMaterial(t *testing.T) {
	fixture := validGenesisFixture(t)
	fixture["mnemonic"] = "public release artifacts must never contain this"
	r := auditGenesis(marshalFixture(t, fixture))
	if !hasIssueMessage(r.Issues, "private/secret key field") {
		t.Fatalf("expected mnemonic field rejection; got:\n%s", formatIssues(r.Issues))
	}
}

func TestAuditGenesisRejectsMutatedGentxSignature(t *testing.T) {
	fixture := validGenesisFixture(t)
	tx := fixtureAppState(t, fixture)["genutil"].(map[string]any)["gen_txs"].([]any)[0].(map[string]any)
	signature, err := base64.StdEncoding.DecodeString(tx["signatures"].([]any)[0].(string))
	if err != nil {
		t.Fatal(err)
	}
	signature[0] ^= 0x01
	tx["signatures"].([]any)[0] = base64.StdEncoding.EncodeToString(signature)
	r := auditGenesis(marshalFixture(t, fixture))
	if !hasIssueMessage(r.Issues, "cryptographic verification failed") {
		t.Fatalf("expected cryptographic signature rejection; got:\n%s", formatIssues(r.Issues))
	}
}

func TestAuditGenesisRejectsDuplicateJSONKeys(t *testing.T) {
	genesis := string(marshalFixture(t, validGenesisFixture(t)))
	duplicate := strings.Replace(genesis, `"chain_id": "zerone-2",`, `"chain_id": "zerone-2", "chain_id": "zerone-2",`, 1)
	r := auditGenesis([]byte(duplicate))
	if !hasIssueMessage(r.Issues, "duplicate JSON object key") {
		t.Fatalf("expected duplicate-key rejection; got:\n%s", formatIssues(r.Issues))
	}
}

func TestAuditGenesisRejectsTrailingData(t *testing.T) {
	genesis := append(marshalFixture(t, validGenesisFixture(t)), []byte("\n{}\n")...)
	r := auditGenesis(genesis)
	if !hasIssueMessage(r.Issues, "multiple JSON values") {
		t.Fatalf("expected trailing genesis JSON rejection; got:\n%s", formatIssues(r.Issues))
	}
}

func TestAuditArtifactDirIntegration(t *testing.T) {
	dir := t.TempDir()
	writePublicArtifactSet(t, dir, marshalFixture(t, validGenesisFixture(t)))

	if r := auditArtifactDir(dir, "drill"); len(r.Issues) != 0 {
		t.Fatalf("public artifact directory failed:\n%s", formatIssues(r.Issues))
	}

	t.Run("private filename", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		writeFixture(t, filepath.Join(badDir, "priv_validator_key.json"), []byte("{}"))
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "filename is forbidden") {
			t.Fatalf("expected private filename rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("private content", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		writeFixture(t, filepath.Join(badDir, "notes.txt"), []byte("-----BEGIN PRIVATE KEY-----\nredacted\n"))
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "private-key marker") {
			t.Fatalf("expected private content rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("renamed mnemonic", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		writeFixture(t, filepath.Join(badDir, "innocent-looking-notes.txt"), []byte("now aware tomorrow wire robust regular unveil swallow trigger about immune wool humor allow inch runway sock acoustic scare weather outdoor shield attract direct\n"))
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "valid BIP-39 mnemonic") {
			t.Fatalf("expected renamed mnemonic rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("punctuated JSON mnemonic", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		writeFixture(t, filepath.Join(badDir, "public-notes.txt"), []byte(
			`["now","aware","tomorrow","wire","robust","regular","unveil","swallow","trigger","about","immune","wool","humor","allow","inch","runway","sock","acoustic","scare","weather","outdoor","shield","attract","direct"]`,
		))
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "valid BIP-39 mnemonic") {
			t.Fatalf("expected punctuated mnemonic rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("benign extra file", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		writeFixture(t, filepath.Join(badDir, "README.txt"), []byte("public but not allowlisted\n"))
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "exactly four allowlisted files") {
			t.Fatalf("expected exact-allowlist rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("symlink", func(t *testing.T) {
		badDir := t.TempDir()
		writePublicArtifactSet(t, badDir, marshalFixture(t, validGenesisFixture(t)))
		if err := os.Symlink("genesis.json", filepath.Join(badDir, "genesis-link.json")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		r := auditArtifactDir(badDir, "drill")
		if !hasIssueMessage(r.Issues, "symlinks are forbidden") {
			t.Fatalf("expected symlink rejection; got:\n%s", formatIssues(r.Issues))
		}
	})
}

func TestAuditArtifactDirRequiresPublishedMetadata(t *testing.T) {
	genesis := marshalFixture(t, validGenesisFixture(t))

	t.Run("missing checksum", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, filepath.Join(dir, "genesis.json"), genesis)
		writeFixture(t, filepath.Join(dir, "network-manifest.json"), marshalFixture(t, networkManifestFixture(t, genesis)))
		r := auditArtifactDir(dir, "drill")
		if !hasIssuePath(r.Issues, "genesis.sha256") || !hasIssueMessage(r.Issues, "required artifact") {
			t.Fatalf("expected missing checksum rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("missing network manifest", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, filepath.Join(dir, "genesis.json"), genesis)
		writeFixture(t, filepath.Join(dir, "genesis.sha256"), []byte(genesisChecksum(genesis)))
		r := auditArtifactDir(dir, "drill")
		if !hasIssuePath(r.Issues, "network-manifest.json") || !hasIssueMessage(r.Issues, "required artifact") {
			t.Fatalf("expected missing network manifest rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("missing human manifest", func(t *testing.T) {
		dir := t.TempDir()
		writePublicArtifactSet(t, dir, genesis)
		if err := os.Remove(filepath.Join(dir, "GENESIS-MANIFEST.md")); err != nil {
			t.Fatal(err)
		}
		r := auditArtifactDir(dir, "drill")
		if !hasIssuePath(r.Issues, "GENESIS-MANIFEST.md") || !hasIssueMessage(r.Issues, "required artifact") {
			t.Fatalf("expected missing human manifest rejection; got:\n%s", formatIssues(r.Issues))
		}
	})
}

func TestAuditArtifactDirRejectsChecksumMismatchOrSyntaxDrift(t *testing.T) {
	genesis := marshalFixture(t, validGenesisFixture(t))
	tests := []struct {
		name     string
		checksum string
	}{
		{name: "wrong hash", checksum: strings.Repeat("0", 64) + "  genesis.json\n"},
		{name: "single separator space", checksum: strings.TrimSuffix(genesisChecksum(genesis), "\n") + "\n"},
		{name: "trailing line", checksum: genesisChecksum(genesis) + "unexpected\n"},
	}
	// Replace the exact two-space separator in the dedicated syntax case.
	tests[1].checksum = strings.Replace(genesisChecksum(genesis), "  genesis.json", " genesis.json", 1)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writePublicArtifactSet(t, dir, genesis)
			writeFixture(t, filepath.Join(dir, "genesis.sha256"), []byte(tc.checksum))
			r := auditArtifactDir(dir, "drill")
			if !hasIssuePath(r.Issues, "genesis.sha256") || !hasIssueMessage(r.Issues, "must contain exactly") {
				t.Fatalf("expected checksum rejection; got:\n%s", formatIssues(r.Issues))
			}
		})
	}
}

func TestAuditArtifactDirCrossChecksNetworkManifest(t *testing.T) {
	genesis := marshalFixture(t, validGenesisFixture(t))
	tests := []struct {
		name      string
		path      string
		value     any
		issuePath string
	}{
		{name: "chain", path: "chain_id", value: "zerone-1", issuePath: "chain_id"},
		{name: "genesis hash", path: "genesis_sha256", value: strings.Repeat("0", 64), issuePath: "genesis_sha256"},
		{name: "supply", path: "supply_uzrn", value: "13555000001", issuePath: "supply_uzrn"},
		{name: "validator account", path: "validator.account_address", value: "zrn1wrong", issuePath: "validator.account_address"},
		{name: "validator operator", path: "validator.operator_address", value: "zrnvaloper1wrong", issuePath: "validator.operator_address"},
		{name: "ops account", path: "operations.account_address", value: "zrn1wrong", issuePath: "operations.account_address"},
		{name: "gentx hash", path: "validator.gentx_sha256", value: strings.Repeat("c", 64), issuePath: "validator.gentx_sha256"},
		{name: "dark activation", path: "activations.ibc", value: "enabled", issuePath: "activations.ibc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writePublicArtifactSet(t, dir, genesis)
			manifest := networkManifestFixture(t, genesis)
			setFixturePath(t, manifest, tc.path, tc.value)
			writeFixture(t, filepath.Join(dir, "network-manifest.json"), marshalFixture(t, manifest))
			r := auditArtifactDir(dir, "drill")
			if !hasIssuePath(r.Issues, tc.issuePath) {
				t.Fatalf("expected manifest mismatch at %q; got:\n%s", tc.issuePath, formatIssues(r.Issues))
			}
		})
	}
}

func TestAuditArtifactDirRequiresExplicitMode(t *testing.T) {
	dir := t.TempDir()
	writePublicArtifactSet(t, dir, marshalFixture(t, validGenesisFixture(t)))
	r := auditArtifactDir(dir, "real")
	if !hasIssuePath(r.Issues, "network-manifest.json.mode") {
		t.Fatalf("expected real-mode gate to reject drill manifest; got:\n%s", formatIssues(r.Issues))
	}
}

func TestAuditArtifactDirRejectsNonStrictOrTrailingManifestJSON(t *testing.T) {
	genesis := marshalFixture(t, validGenesisFixture(t))

	t.Run("unknown field", func(t *testing.T) {
		dir := t.TempDir()
		writePublicArtifactSet(t, dir, genesis)
		manifest := networkManifestFixture(t, genesis)
		manifest["unexpected"] = true
		writeFixture(t, filepath.Join(dir, "network-manifest.json"), marshalFixture(t, manifest))
		r := auditArtifactDir(dir, "drill")
		if !hasIssueMessage(r.Issues, "unknown field") {
			t.Fatalf("expected unknown-field rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("trailing JSON value", func(t *testing.T) {
		dir := t.TempDir()
		writePublicArtifactSet(t, dir, genesis)
		manifest := append(marshalFixture(t, networkManifestFixture(t, genesis)), []byte("\n{}\n")...)
		writeFixture(t, filepath.Join(dir, "network-manifest.json"), manifest)
		r := auditArtifactDir(dir, "drill")
		if !hasIssueMessage(r.Issues, "multiple JSON values") {
			t.Fatalf("expected trailing manifest rejection; got:\n%s", formatIssues(r.Issues))
		}
	})

	t.Run("duplicate field", func(t *testing.T) {
		dir := t.TempDir()
		writePublicArtifactSet(t, dir, genesis)
		manifest := string(marshalFixture(t, networkManifestFixture(t, genesis)))
		manifest = strings.Replace(manifest, `"mode": "drill",`, `"mode": "drill", "mode": "drill",`, 1)
		writeFixture(t, filepath.Join(dir, "network-manifest.json"), []byte(manifest))
		r := auditArtifactDir(dir, "drill")
		if !hasIssueMessage(r.Issues, "duplicate JSON object key") {
			t.Fatalf("expected duplicate manifest field rejection; got:\n%s", formatIssues(r.Issues))
		}
	})
}

func validGenesisFixture(t *testing.T) map[string]any {
	t.Helper()
	signerKey, err := base64.StdEncoding.DecodeString("A2yEC8F4wjSwtwq0tF5AJTZ/eDy+ioK1RWp80TOroIZW")
	if err != nil {
		t.Fatal(err)
	}
	validatorData := accountID(signerKey)
	validator := bech32Encode(t, "zrn", validatorData)
	valoper := bech32Encode(t, "zrnvaloper", validatorData)
	ops := bech32Encode(t, "zrn", bytes20(0x42))
	consensusKey := "cTCcuxAYZrd34/0M+0FvCUXtooB0tI99lV2zuY5HxKQ="
	signature := "5OoV0V3W7+rXrVw4/qmSfITaK56TNo8JH+wnbaqjwukEcECXqWXqb5PnKUT6MLbYua4i9m/jh1hjBMnZu0x83A=="

	return map[string]any{
		"chain_id":       expectedChainID,
		"genesis_time":   "2026-07-20T12:00:00Z",
		"initial_height": "1",
		"consensus": map[string]any{"params": map[string]any{
			"block": map[string]any{"max_bytes": "4194304", "max_gas": "33333333"},
			"evidence": map[string]any{
				"max_age_num_blocks": "100000", "max_age_duration": "172800000000000", "max_bytes": "1048576",
			},
			"validator": map[string]any{"pub_key_types": []any{"ed25519"}},
			"version":   map[string]any{"app": "0"},
			"abci":      map[string]any{"vote_extensions_enable_height": "0"},
		}},
		"app_state": map[string]any{
			"auth": map[string]any{"accounts": []any{
				map[string]any{
					"@type": "/cosmos.vesting.v1beta1.PermanentLockedAccount",
					"base_vesting_account": map[string]any{
						"base_account": map[string]any{
							"address": validator, "pub_key": nil, "account_number": "0", "sequence": "0",
						},
						"original_vesting":  []any{coin(denom, validatorSelfBond)},
						"delegated_free":    []any{},
						"delegated_vesting": []any{},
						"end_time":          "0",
					},
				},
				map[string]any{
					"@type": "/cosmos.auth.v1beta1.BaseAccount", "address": ops, "pub_key": nil,
					"account_number": "1", "sequence": "0",
				},
			}},
			"bank": map[string]any{
				"balances": []any{
					map[string]any{"address": validator, "coins": []any{coin(denom, validatorBalance)}},
					map[string]any{"address": ops, "coins": []any{coin(denom, opsBalance)}},
				},
				"supply": []any{coin(denom, totalSupply)},
			},
			"staking": map[string]any{"params": map[string]any{
				"unbonding_time": "1814400s", "max_validators": 33, "max_entries": 7,
				"historical_entries": 10000, "bond_denom": denom, "min_commission_rate": "0.050000000000000000",
			}},
			"gov": map[string]any{"params": map[string]any{
				"min_deposit": []any{coin(denom, "100000000")}, "max_deposit_period": "259200s",
				"voting_period": "259200s", "quorum": "0.334000000000000000", "threshold": "0.500000000000000000",
				"veto_threshold": "0.334000000000000000", "min_initial_deposit_ratio": "0.250000000000000000",
				"proposal_cancel_ratio": "0.500000000000000000", "proposal_cancel_dest": "",
				"expedited_voting_period": "86400s", "expedited_threshold": "0.667000000000000000",
				"expedited_min_deposit": []any{coin(denom, "300000000")}, "burn_vote_quorum": false,
				"burn_proposal_deposit_prevote": false, "burn_vote_veto": true,
				"min_deposit_ratio": "0.010000000000000000",
			}},
			"genutil": map[string]any{"gen_txs": []any{map[string]any{
				"body": map[string]any{
					"memo":           strings.Repeat("2", 40) + "@127.0.0.1:26656",
					"timeout_height": "0", "extension_options": []any{}, "non_critical_extension_options": []any{},
					"messages": []any{map[string]any{
						"@type": "/cosmos.staking.v1beta1.MsgCreateValidator",
						"description": map[string]any{
							"moniker": "zerone-2-custodian", "identity": "", "website": "", "security_contact": "", "details": "",
						},
						"commission": map[string]any{
							"rate": "0.050000000000000000", "max_rate": "0.200000000000000000", "max_change_rate": "0.010000000000000000",
						},
						"delegator_address":   "",
						"validator_address":   valoper,
						"min_self_delegation": "1",
						"pubkey":              map[string]any{"@type": "/cosmos.crypto.ed25519.PubKey", "key": consensusKey},
						"value":               coin(denom, validatorSelfBond),
					}},
				},
				"auth_info": map[string]any{
					"signer_infos": []any{map[string]any{
						"public_key": map[string]any{"@type": "/cosmos.crypto.secp256k1.PubKey", "key": base64.StdEncoding.EncodeToString(signerKey)},
						"mode_info":  map[string]any{"single": map[string]any{"mode": "SIGN_MODE_DIRECT"}}, "sequence": "0",
					}},
					"fee": map[string]any{"amount": []any{}, "gas_limit": "200000", "payer": "", "granter": ""}, "tip": nil,
				},
				"signatures": []any{signature},
			}}},
			"ibc": map[string]any{
				"client_genesis": map[string]any{
					"clients": []any{}, "clients_consensus": []any{}, "clients_metadata": []any{},
					"params": map[string]any{"allowed_clients": []any{"09-localhost"}}, "create_localhost": false,
				},
				"connection_genesis": map[string]any{"connections": []any{}, "client_connection_paths": []any{}},
				"channel_genesis": map[string]any{
					"channels": []any{}, "acknowledgements": []any{}, "commitments": []any{}, "receipts": []any{},
				},
			},
			"transfer": map[string]any{
				"denom_traces": []any{}, "total_escrowed": []any{},
				"params": map[string]any{"send_enabled": false, "receive_enabled": false},
			},
			"interchainaccounts": map[string]any{
				"controller_genesis_state": map[string]any{
					"active_channels": []any{}, "interchain_accounts": []any{}, "ports": []any{},
					"params": map[string]any{"controller_enabled": false},
				},
				"host_genesis_state": map[string]any{
					"active_channels": []any{}, "interchain_accounts": []any{},
					"params": map[string]any{"host_enabled": false, "allow_messages": []any{}},
				},
			},
			"ibcratelimit": map[string]any{"params": map[string]any{"enabled": true}, "rate_limits": []any{}},
			"substrate_bridge": map[string]any{
				"adapters": []any{},
				"params": map[string]any{
					"max_pending_claims_per_attestation": 1000,
					"per_pending_claim_bond_uzrn":        "22200", "attestation_min_bond_uzrn": "22200000",
					"pending_claim_rejection_threshold_bps": 1000, "min_verified_ratio_for_settle_bps": 6667,
					"witness_reward_challenge_window_blocks": "274176",
				},
			},
			"knowledge": map[string]any{
				"params": map[string]any{
					"min_verifiers": "3", "min_headcount_agreement": "3",
					"min_review_fee": protocolAdmissionBarrier, "min_challenge_stake": protocolAdmissionBarrier,
					"bootstrap_fund_enabled": false, "demand_tracking_enabled": false, "vindication_refund_enabled": false,
					"verification_reward": "0", "demand_bounty_base_reward": "0", "demand_bounty_per_query_bonus": "0",
					"training_fund_base_reward": "0", "probe_bounty_mint_per_block": "0", "invitation_bonus_amount": "0",
					"contribution_challenge_bond": protocolAdmissionBarrier, "contribution_challenge_reward_multiplier_bps": "1000000",
					"guardian_addresses": []any{},
				},
				"bootstrap_fund_allocation": "0", "training_fund_allocation": "0",
				"facts": []any{}, "pending_claims": []any{}, "active_rounds": []any{}, "common_knowledge": []any{},
				"training_fund_disbursements": []any{}, "augmentation_bounties": []any{}, "augmentations": []any{}, "contribution_challenges": []any{},
			},
			"vesting_rewards": map[string]any{"params": map[string]any{
				"block_reward": "0", "floor_reward": "0", "empty_block_reward_rate": 0,
				"min_validators_for_full_reward": 22, "initial_fund_balance": "0",
				"founder_share_bps": 0, "founder_address": "", "vesting_enabled": false,
				"knowledge_coupling_target_bps": 0, "knowledge_coupling_floor_bps": 0,
			}},
			"claiming_pot": map[string]any{
				"params": map[string]any{"bootstrap_registrar": ""}, "pots": []any{}, "claims": []any{},
			},
			"emergency": map[string]any{
				"params": map[string]any{
					"genesis_council": []any{}, "council_expiry_block": 0, "min_distinct_voters": 4,
					"min_guardian_stake": protocolAdmissionBarrier, "max_revert_depth": 1111,
				},
				"status": "normal",
			},
			"zerone_staking": map[string]any{
				"params":     map[string]any{"min_self_delegation": customStakeMinimum, "min_stake_for_verification": customStakeMinimum, "max_validators": 33},
				"validators": []any{}, "delegations": []any{}, "unbonding_entries": []any{}, "unbonding_seq": 0,
			},
			"alignment": map[string]any{
				"params": map[string]any{"enabled": false}, "state": map[string]any{"enabled": false},
				"observations": []any{}, "scores": []any{}, "health_indices": []any{}, "corrections": []any{},
			},
			"counterexamples": map[string]any{
				"params": map[string]any{"proposals_enabled": false}, "counterexamples": []any{}, "validations": []any{},
			},
			"liquiditypool": map[string]any{
				"params": map[string]any{
					"max_pools": 3, "min_initial_liquidity": protocolAdmissionBarrier, "protocol_fee_bps": 0,
					"billing_quote_denoms": []any{},
				},
				"pools": []any{}, "twap_accumulators": []any{},
			},
		},
	}
}

func accountID(publicKey []byte) []byte {
	hash := sha256.Sum256(publicKey)
	ripe := ripemd160.New()
	_, _ = ripe.Write(hash[:])
	return ripe.Sum(nil)
}

func validatorAddressForFixture(t *testing.T) string {
	t.Helper()
	fixture := validGenesisFixture(t)
	accounts := fixtureAppState(t, fixture)["auth"].(map[string]any)["accounts"].([]any)
	return accounts[0].(map[string]any)["base_vesting_account"].(map[string]any)["base_account"].(map[string]any)["address"].(string)
}

func coin(coinDenom, amount string) map[string]any {
	return map[string]any{"denom": coinDenom, "amount": amount}
}

func fixtureAppState(t *testing.T, fixture map[string]any) map[string]any {
	t.Helper()
	state, ok := fixture["app_state"].(map[string]any)
	if !ok {
		t.Fatal("fixture app_state missing")
	}
	return state
}

func mutateLockedAmount(t *testing.T, fixture map[string]any, amount string) []any {
	t.Helper()
	accounts := fixtureAppState(t, fixture)["auth"].(map[string]any)["accounts"].([]any)
	locked := accounts[0].(map[string]any)["base_vesting_account"].(map[string]any)
	locked["original_vesting"].([]any)[0].(map[string]any)["amount"] = amount
	return accounts
}

func mutateGentxValoper(t *testing.T, fixture map[string]any, valoper string) []any {
	t.Helper()
	txs := fixtureAppState(t, fixture)["genutil"].(map[string]any)["gen_txs"].([]any)
	msg := txs[0].(map[string]any)["body"].(map[string]any)["messages"].([]any)[0].(map[string]any)
	msg["validator_address"] = valoper
	return txs
}

func setFixturePath(t *testing.T, fixture map[string]any, path string, value any) {
	t.Helper()
	parts := strings.Split(path, ".")
	current := fixture
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			t.Fatalf("fixture path %s is not an object", part)
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func marshalFixture(t *testing.T, fixture map[string]any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func networkManifestFixture(t *testing.T, genesis []byte) map[string]any {
	t.Helper()
	audited := auditGenesis(genesis)
	if len(audited.Issues) != 0 {
		t.Fatalf("cannot build manifest for invalid genesis:\n%s", formatIssues(audited.Issues))
	}
	digest := sha256.Sum256(genesis)
	return map[string]any{
		"schema":         networkManifestSchema,
		"mode":           "drill",
		"chain_id":       expectedChainID,
		"genesis_time":   audited.GenesisTime,
		"genesis_sha256": hex.EncodeToString(digest[:]),
		"release": map[string]any{
			"source_commit": strings.Repeat("a", 40), "tag": "DRILL-NOT-A-RELEASE",
			"tag_signer_fingerprint": drillSignerFingerprint,
			"binary_sha256":          strings.Repeat("b", 64), "binary_version": "synthetic-test",
			"binary_goos": "darwin", "binary_goarch": "arm64",
		},
		"trust_model": map[string]any{
			"genesis_validators": 1, "byzantine_fault_tolerance": 0,
			"disclosure": "DRILL ONLY: public fixture keys; never deploy",
		},
		"supply_uzrn": totalSupply,
		"validator": map[string]any{
			"account_address": audited.ValidatorAddress, "operator_address": audited.ValidatorOperatorAddress,
			"consensus_pubkey": map[string]any{"@type": audited.ValidatorConsensusKey.Type, "key": audited.ValidatorConsensusKey.Key},
			"node_id":          audited.ValidatorNodeID, "self_bond_uzrn": validatorSelfBond,
			"gentx_sha256": audited.EmbeddedGentxSHA256,
		},
		"operations": map[string]any{"account_address": audited.OpsAddress},
		"activations": map[string]any{
			"vote_extensions": "disabled", "pot": "not live", "ibc": "external-disabled; localhost-only",
			"substrate_bridge": "disabled", "claiming": "disabled",
		},
	}
}

func genesisChecksum(genesis []byte) string {
	digest := sha256.Sum256(genesis)
	return hex.EncodeToString(digest[:]) + "  genesis.json\n"
}

func writePublicArtifactSet(t *testing.T, dir string, genesis []byte) {
	t.Helper()
	writeFixture(t, filepath.Join(dir, "genesis.json"), genesis)
	writeFixture(t, filepath.Join(dir, "genesis.sha256"), []byte(genesisChecksum(genesis)))
	writeFixture(t, filepath.Join(dir, "network-manifest.json"), marshalFixture(t, networkManifestFixture(t, genesis)))
	writeFixture(t, filepath.Join(dir, "GENESIS-MANIFEST.md"), []byte("# synthetic public fixture\n"))
}

func writeFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasIssuePath(issues []issue, fragment string) bool {
	for _, problem := range issues {
		if strings.Contains(problem.Path, fragment) {
			return true
		}
	}
	return false
}

func hasIssueMessage(issues []issue, fragment string) bool {
	for _, problem := range issues {
		if strings.Contains(problem.Message, fragment) {
			return true
		}
	}
	return false
}

func formatIssues(issues []issue) string {
	var builder strings.Builder
	for _, problem := range issues {
		builder.WriteString(problem.Path)
		builder.WriteString(": ")
		builder.WriteString(problem.Message)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func bytes20(seed byte) []byte { return filledBytes(20, seed) }
func bytes32(seed byte) []byte { return filledBytes(32, seed) }
func filledBytes(length int, seed byte) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = seed + byte(i%31)
	}
	return result
}

func bech32Encode(t *testing.T, hrp string, data []byte) string {
	t.Helper()
	converted, err := convertBits(data, 8, 5, true)
	if err != nil {
		t.Fatal(err)
	}
	values := append(append([]byte{}, converted...), make([]byte, 6)...)
	polymod := bech32Polymod(append(bech32HRPExpand(hrp), values...)) ^ 1
	for i := 0; i < 6; i++ {
		values[len(converted)+i] = byte(polymod >> (5 * (5 - i)) & 31)
	}
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	var builder strings.Builder
	builder.WriteString(hrp)
	builder.WriteByte('1')
	for _, value := range values {
		builder.WriteByte(charset[value])
	}
	return builder.String()
}
