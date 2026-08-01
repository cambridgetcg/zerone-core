package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
)

type fixture struct {
	input       []byte
	policy      rewritePolicy
	oldKey      cmted25519.PubKey
	newKey      cmted25519.PubKey
	operator    string
	inputDigest string
	policyBytes []byte
	policyFile  string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	genesisBytes, err := os.ReadFile("../../deploy/mainnet/artifacts/genesis.json")
	if err != nil {
		t.Fatal(err)
	}
	appGenesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(genesisBytes))
	if err != nil {
		t.Fatal(err)
	}
	encodingConfig := zeroneapp.MakeEncodingConfig()
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(appGenesis.AppState, &appState); err != nil {
		t.Fatal(err)
	}
	appState["genutil"] = json.RawMessage(`{"gen_txs":[]}`)

	oldPrivate := cmted25519.GenPrivKeyFromSecret([]byte("fork-genesis-old"))
	newPrivate := cmted25519.GenPrivKeyFromSecret([]byte("fork-genesis-new"))
	oldKey := oldPrivate.PubKey().(cmted25519.PubKey)
	newKey := newPrivate.PubKey().(cmted25519.PubKey)
	operatorBytes := bytes.Repeat([]byte{0x31}, 20)
	operator := sdk.ValAddress(operatorBytes).String()
	delegator := sdk.AccAddress(operatorBytes).String()

	validator, err := stakingtypes.NewValidator(
		operator,
		&sdked25519.PubKey{Key: oldKey.Bytes()},
		stakingtypes.Description{Moniker: "fixture-validator"},
	)
	if err != nil {
		t.Fatal(err)
	}
	tokens := math.NewInt(9_000_000)
	shares := math.LegacyNewDecFromInt(tokens)
	validator.Status = stakingtypes.Bonded
	validator.Tokens = tokens
	validator.DelegatorShares = shares
	power := validator.ConsensusPower(sdk.DefaultPowerReduction)
	var stakingGenesis stakingtypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["staking"], &stakingGenesis); err != nil {
		t.Fatal(err)
	}
	stakingGenesis.LastTotalPower = tokens
	stakingGenesis.LastValidatorPowers = []stakingtypes.LastValidatorPower{{
		Address: operator,
		Power:   power,
	}}
	stakingGenesis.Validators = []stakingtypes.Validator{validator}
	stakingGenesis.Delegations = []stakingtypes.Delegation{{
		DelegatorAddress: delegator,
		ValidatorAddress: operator,
		Shares:           shares,
	}}
	stakingGenesis.Exported = true
	appState["staking"], err = encodingConfig.Codec.MarshalJSON(&stakingGenesis)
	if err != nil {
		t.Fatal(err)
	}
	var bankGenesis banktypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["bank"], &bankGenesis); err != nil {
		t.Fatal(err)
	}
	bondedTokens := sdk.NewCoin("uzrn", tokens)
	bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.NewCoins(bondedTokens),
	})
	bankGenesis.Balances = banktypes.SanitizeGenesisBalances(bankGenesis.Balances)
	bankGenesis.Supply = bankGenesis.Supply.Add(bondedTokens)
	appState["bank"], err = encodingConfig.Codec.MarshalJSON(&bankGenesis)
	if err != nil {
		t.Fatal(err)
	}

	// The checked-in live genesis predates liquiditypool consensus v4, where a
	// zero MaxPools meant unlimited. This compiler test needs a source export
	// that already satisfies the target application's module schemas so the
	// narrow compiler can prove it changes no unrelated application state.
	var liquidityGenesis map[string]any
	liquidityDecoder := json.NewDecoder(bytes.NewReader(appState["liquiditypool"]))
	liquidityDecoder.UseNumber()
	if err := liquidityDecoder.Decode(&liquidityGenesis); err != nil {
		t.Fatal(err)
	}
	liquidityParams, ok := liquidityGenesis["params"].(map[string]any)
	if !ok {
		t.Fatal("liquiditypool fixture params are absent")
	}
	liquidityParams["max_pools"] = json.Number("16")
	appState["liquiditypool"], err = json.Marshal(liquidityGenesis)
	if err != nil {
		t.Fatal(err)
	}

	oldConsensusAddress := sdk.ConsAddress(oldKey.Address()).String()
	var slashingGenesis slashingtypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["slashing"], &slashingGenesis); err != nil {
		t.Fatal(err)
	}
	slashingGenesis.SigningInfos = []slashingtypes.SigningInfo{{
		Address: oldConsensusAddress,
		ValidatorSigningInfo: slashingtypes.ValidatorSigningInfo{
			Address:             oldConsensusAddress,
			StartHeight:         5,
			IndexOffset:         17,
			JailedUntil:         time.Unix(0, 0).UTC(),
			Tombstoned:          false,
			MissedBlocksCounter: 2,
		},
	}}
	slashingGenesis.MissedBlocks = []slashingtypes.ValidatorMissedBlocks{{
		Address: oldConsensusAddress,
		MissedBlocks: []slashingtypes.MissedBlock{{
			Index:  3,
			Missed: true,
		}},
	}}
	appState["slashing"], err = encodingConfig.Codec.MarshalJSON(&slashingGenesis)
	if err != nil {
		t.Fatal(err)
	}

	appStateBytes, err := json.Marshal(appState)
	if err != nil {
		t.Fatal(err)
	}
	appGenesis.AppState = appStateBytes
	appGenesis.AppVersion = "legacy-fixture"
	appGenesis.InitialHeight = 101
	appGenesis.GenesisTime = time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	appGenesis.Consensus.Validators = []cmttypes.GenesisValidator{{
		Address: oldKey.Address(),
		PubKey:  oldKey,
		Power:   power,
		Name:    "fixture-validator",
	}}
	input, err := json.Marshal(appGenesis)
	if err != nil {
		t.Fatal(err)
	}
	executableSHA256, err := currentExecutableDigest()
	if err != nil {
		t.Fatal(err)
	}

	policy := rewritePolicy{
		Schema:                   policySchema,
		Profile:                  compilerProfile,
		IncidentID:               "incident-fork-genesis-test",
		SourceChainID:            "zerone-1",
		TargetChainID:            "zerone-2",
		SourceLastHeight:         100,
		SourceBlockIDSHA256:      strings.Repeat("1", 64),
		SourceAppHashSHA256:      strings.Repeat("2", 64),
		SourceLastBlockTime:      "2026-07-29T23:59:59Z",
		SourceSignedCommitSHA256: strings.Repeat("6", 64),
		SourceValidatorSetSHA256: strings.Repeat("7", 64),
		SourceExportSHA256:       digest(input),
		OldOperatorAddress:       operator,
		OldConsensusPublicKey:    base64.StdEncoding.EncodeToString(oldKey.Bytes()),
		NewConsensusPublicKey:    base64.StdEncoding.EncodeToString(newKey.Bytes()),
		OperatorDisposition:      operatorRetain,
		ProhibitedConsensusPublicKeys: []string{
			base64.StdEncoding.EncodeToString(oldKey.Bytes()),
		},
		ProhibitedPrivilegedIdentities: []string{},
		TargetAppVersion:               "v2.0.0-recovery",
		TargetGenesisTime:              "2026-07-30T00:00:00Z",
		CustodyAssessmentSHA256:        strings.Repeat("3", 64),
		ForkPolicySHA256:               strings.Repeat("4", 64),
		RewriteToolSHA256:              executableSHA256,
		IBCDisposition:                 ibcDisposition,
		EmergencyStartMode:             emergencyStartMode,
		IndependentReproducers: []independentReproducer{
			{
				Identity:      "did:zrn:reproducer-a",
				ControlDomain: "reproducer-domain-a",
				PublicKey: hex.EncodeToString(
					cmted25519.GenPrivKeyFromSecret([]byte("reproducer-a")).PubKey().Bytes(),
				),
			},
			{
				Identity:      "did:zrn:reproducer-b",
				ControlDomain: "reproducer-domain-b",
				PublicKey: hex.EncodeToString(
					cmted25519.GenPrivKeyFromSecret([]byte("reproducer-b")).PubKey().Bytes(),
				),
			},
		},
	}
	unsignedPolicy, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	policy.PolicySHA256 = digest(unsignedPolicy)
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{
		input:       input,
		policy:      policy,
		oldKey:      oldKey,
		newKey:      newKey,
		operator:    operator,
		inputDigest: digest(input),
		policyBytes: policyBytes,
		policyFile:  digest(policyBytes),
	}
}

func TestCompileGenesisConsensusKeyOnly(t *testing.T) {
	f := newFixture(t)
	outputA, reportA, err := compileGenesis(
		f.input,
		f.inputDigest,
		f.policy,
		f.policyFile,
		"did:zrn:reproducer-a",
	)
	if err != nil {
		t.Fatal(err)
	}
	outputB, reportB, err := compileGenesis(
		f.input,
		f.inputDigest,
		f.policy,
		f.policyFile,
		"did:zrn:reproducer-b",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(outputA, outputB) {
		t.Fatal("independent reproductions produced different genesis bytes")
	}
	if reportA.OutputGenesisSHA256 != reportB.OutputGenesisSHA256 ||
		reportA.OutputGenesisSHA256 != digest(outputA) {
		t.Fatal("independent output digests differ")
	}
	if reportA.ReproducerID == reportB.ReproducerID {
		t.Fatal("independent report identities unexpectedly match")
	}
	if reportA.ReproducerControlDomain == reportB.ReproducerControlDomain ||
		reportA.ReproducerPublicKey == reportB.ReproducerPublicKey {
		t.Fatal("independent reports reused a control domain or attestation key")
	}
	if reportA.ReproducerControlDomain != "reproducer-domain-a" ||
		reportA.ReproducerPublicKey != f.policy.IndependentReproducers[0].PublicKey {
		t.Fatal("compiler report did not bind the selected policy-authorized reproducer")
	}
	if bytes.Contains(outputA, []byte(f.policy.OldConsensusPublicKey)) {
		t.Fatal("old consensus public key remains in output")
	}
	if !bytes.Contains(outputA, []byte(f.policy.NewConsensusPublicKey)) {
		t.Fatal("new consensus public key is absent from output")
	}

	target, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(outputA))
	if err != nil {
		t.Fatal(err)
	}
	if target.ChainID != f.policy.TargetChainID ||
		target.InitialHeight != 101 ||
		target.AppVersion != f.policy.TargetAppVersion {
		t.Fatalf("unexpected target identity: %#v", target)
	}
	if len(target.Consensus.Validators) != 1 ||
		!bytes.Equal(target.Consensus.Validators[0].PubKey.Bytes(), f.newKey.Bytes()) {
		t.Fatal("target consensus validator does not use the new key")
	}
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(target.AppState, &appState); err != nil {
		t.Fatal(err)
	}
	encodingConfig := zeroneapp.MakeEncodingConfig()
	var emergency emergencytypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["emergency"], &emergency); err != nil {
		t.Fatal(err)
	}
	if emergency.Status != string(emergencytypes.StatusHalted) ||
		emergency.ActiveHaltCeremonyId != "legacy-genesis-quarantine" ||
		emergency.HaltStartBlock != 101 {
		t.Fatalf(
			"target did not begin quarantined: status=%q ceremony=%q halt_start=%d",
			emergency.Status,
			emergency.ActiveHaltCeremonyId,
			emergency.HaltStartBlock,
		)
	}
	for _, module := range reportA.ModuleDigests {
		switch module.Module {
		case "staking", "slashing", "emergency", "ibc", "transfer":
			if !module.Changed {
				t.Fatalf("expected %s to change", module.Module)
			}
		default:
			if module.Changed {
				t.Fatalf("unexpected module change: %s", module.Module)
			}
		}
	}
	if len(reportA.SchemaMigrations) != 2 ||
		reportA.SchemaMigrations[0] != "ibc-transfer-v8-empty-denom-traces-to-v10-empty-denoms" ||
		reportA.SchemaMigrations[1] != "ibc-core-v8-empty-to-v10-empty-v2-state" {
		t.Fatalf("unexpected schema migrations: %#v", reportA.SchemaMigrations)
	}
	sealed, err := sealReport(reportA)
	if err != nil {
		t.Fatal(err)
	}
	var decoded compilerReport
	if err := json.Unmarshal(sealed, &decoded); err != nil {
		t.Fatal(err)
	}
	unsigned := decoded
	unsigned.ReportSHA256 = ""
	unsignedBytes, _ := json.Marshal(unsigned)
	if decoded.ReportSHA256 != digest(unsignedBytes) {
		t.Fatal("report self-hash is invalid")
	}
}

func TestCompiledGenesisInitializesCurrentApplication(t *testing.T) {
	f := newFixture(t)
	output, _, err := compileGenesis(
		f.input,
		f.inputDigest,
		f.policy,
		f.policyFile,
		"did:zrn:reproducer-a",
	)
	if err != nil {
		t.Fatal(err)
	}
	target, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	application := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID(target.ChainID),
	)
	consensusParams := target.Consensus.Params.ToProto()
	if _, err := application.InitChain(&abci.RequestInitChain{
		Time:            target.GenesisTime,
		ChainId:         target.ChainID,
		ConsensusParams: &consensusParams,
		AppStateBytes:   target.AppState,
		InitialHeight:   target.InitialHeight,
	}); err != nil {
		t.Fatalf("compiled genesis did not initialize the current application: %v", err)
	}
	if _, err := application.Commit(); err != nil {
		t.Fatalf("compiled genesis did not commit its initialized state: %v", err)
	}
}

func TestCompileGenesisRejectsUnsafeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, f *fixture)
		wantErr string
	}{
		{
			name: "live IBC client",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					ibc := state["ibc"].(map[string]any)
					clients := ibc["client_genesis"].(map[string]any)
					clients["clients"] = []any{map[string]any{"client_id": "07-tendermint-0"}}
				})
			},
			wantErr: "must be an empty array",
		},
		{
			name: "live IBC rate limit",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					rateLimit := state["ibcratelimit"].(map[string]any)
					rateLimit["rate_limits"] = []any{map[string]any{
						"channel_id": "channel-0",
						"denom":      "uzrn",
					}}
				})
			},
			wantErr: "ibcratelimit.rate_limits",
		},
		{
			name: "pending SDK governance",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					gov := state["gov"].(map[string]any)
					gov["proposals"] = []any{map[string]any{"id": "1"}}
				})
			},
			wantErr: "gov.proposals must be an empty array",
		},
		{
			name: "missing SDK governance inventory",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					gov := state["gov"].(map[string]any)
					delete(gov, "proposals")
				})
			},
			wantErr: "gov.proposals is absent",
		},
		{
			name: "hidden second bonded validator",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					staking := state["staking"].(map[string]any)
					validators := staking["validators"].([]any)
					encoded, err := json.Marshal(validators[0])
					if err != nil {
						t.Fatal(err)
					}
					var duplicate map[string]any
					if err := json.Unmarshal(encoded, &duplicate); err != nil {
						t.Fatal(err)
					}
					duplicate["operator_address"] = sdk.ValAddress(
						bytes.Repeat([]byte{0x5a}, 20),
					).String()
					staking["validators"] = append(validators, duplicate)
				})
			},
			wantErr: "exactly one bonded staking validator",
		},
		{
			name: "genesis transaction replay",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					genutil := state["genutil"].(map[string]any)
					genutil["gen_txs"] = []any{map[string]any{"body": map[string]any{}}}
				})
			},
			wantErr: "genutil.gen_txs must be an empty array",
		},
		{
			name: "pending consensus evidence",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					evidence := state["evidence"].(map[string]any)
					evidence["evidence"] = []any{map[string]any{
						"@type": "/cometbft.types.DuplicateVoteEvidence",
					}}
				})
			},
			wantErr: "evidence.evidence must be an empty array",
		},
		{
			name: "tombstoned old signer",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					slashing := state["slashing"].(map[string]any)
					infos := slashing["signing_infos"].([]any)
					info := infos[0].(map[string]any)
					nested := info["validator_signing_info"].(map[string]any)
					nested["tombstoned"] = true
				})
			},
			wantErr: "refuses a tombstoned old validator",
		},
		{
			name: "custom metadata not pinned",
			mutate: func(t *testing.T, f *fixture) {
				mutateAppState(t, f, func(state map[string]any) {
					custom := state["zerone_staking"].(map[string]any)
					operatorBytes, _ := sdk.ValAddressFromBech32(f.operator)
					custom["validators"] = []any{map[string]any{
						"operator_address": sdk.AccAddress(operatorBytes).String(),
						"consensus_pubkey": "old-metadata",
					}}
				})
			},
			wantErr: "does not bind its metadata rewrite",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			test.mutate(t, &f)
			_, _, err := compileGenesis(
				f.input,
				digest(f.input),
				f.policy,
				f.policyFile,
				"did:zrn:reproducer-a",
			)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestPolicyDefaultsToFailClosed(t *testing.T) {
	f := newFixture(t)
	if err := validatePolicy(f.policy, "did:zrn:reproducer-a"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*rewritePolicy)
		want   string
	}{
		{
			name: "same key",
			mutate: func(policy *rewritePolicy) {
				policy.NewConsensusPublicKey = policy.OldConsensusPublicKey
			},
			want: "must differ",
		},
		{
			name: "reused chain id",
			mutate: func(policy *rewritePolicy) {
				policy.TargetChainID = policy.SourceChainID
			},
			want: "must differ",
		},
		{
			name: "checkpoint clock rewind",
			mutate: func(policy *rewritePolicy) {
				policy.TargetGenesisTime = "2026-07-29T12:00:00Z"
			},
			want: "strictly later than source_last_block_time",
		},
		{
			name: "single reproducer",
			mutate: func(policy *rewritePolicy) {
				policy.IndependentReproducers = policy.IndependentReproducers[:1]
			},
			want: "at least two",
		},
		{
			name: "duplicate reproducer control domain",
			mutate: func(policy *rewritePolicy) {
				policy.IndependentReproducers = append(
					[]independentReproducer(nil),
					policy.IndependentReproducers...,
				)
				policy.IndependentReproducers[1].ControlDomain =
					policy.IndependentReproducers[0].ControlDomain
			},
			want: "distinct identities, control domains, and public keys",
		},
		{
			name: "duplicate reproducer public key",
			mutate: func(policy *rewritePolicy) {
				policy.IndependentReproducers = append(
					[]independentReproducer(nil),
					policy.IndependentReproducers...,
				)
				policy.IndependentReproducers[1].PublicKey =
					policy.IndependentReproducers[0].PublicKey
			},
			want: "distinct identities, control domains, and public keys",
		},
		{
			name: "operator retention not proven",
			mutate: func(policy *rewritePolicy) {
				policy.OperatorDisposition = "UNKNOWN"
			},
			want: "operator_disposition",
		},
		{
			name: "operator prohibited",
			mutate: func(policy *rewritePolicy) {
				policy.ProhibitedPrivilegedIdentities = []string{policy.OldOperatorAddress}
			},
			want: "prohibited_privileged_identities",
		},
		{
			name:   "reproducer absent",
			mutate: func(policy *rewritePolicy) {},
			want:   "is not authorized",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			policy := f.policy
			test.mutate(&policy)
			unsigned := policy
			unsigned.PolicySHA256 = ""
			unsignedBytes, _ := json.Marshal(unsigned)
			policy.PolicySHA256 = digest(unsignedBytes)
			reproducer := "did:zrn:reproducer-a"
			if test.name == "reproducer absent" {
				reproducer = "did:zrn:outsider"
			}
			err := validatePolicy(policy, reproducer)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestPolicyContractHasNoForkChoiceCycle(t *testing.T) {
	f := newFixture(t)
	if bytes.Contains(f.policyBytes, []byte("fork_choice")) {
		t.Fatal("rewrite policy must not bind the final fork choice")
	}
	if !bytes.Contains(f.policyBytes, []byte(`"fork_policy_sha256"`)) ||
		!bytes.Contains(f.policyBytes, []byte(`"rewrite_tool_sha256"`)) {
		t.Fatal("rewrite policy is missing its pre-compilation trust bindings")
	}
}

func TestCLIRefusesSymlinkAndExistingOutputs(t *testing.T) {
	f := newFixture(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(root, "input.json")
	policyPath := filepath.Join(root, "policy.json")
	symlinkPath := filepath.Join(root, "input-link.json")
	if err := os.WriteFile(inputPath, f.input, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, f.policyBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inputPath, symlinkPath); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--input", symlinkPath,
		"--input-sha256", f.inputDigest,
		"--policy", policyPath,
		"--policy-sha256", f.policyFile,
		"--reproducer-id", "did:zrn:reproducer-a",
		"--output", filepath.Join(root, "output.json"),
		"--report", filepath.Join(root, "report.json"),
	}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "regular non-symlink") {
		t.Fatalf("unexpected symlink result code=%d stderr=%q", code, stderr.String())
	}

	outputPath := filepath.Join(root, "existing-output.json")
	if err := os.WriteFile(outputPath, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"--input", inputPath,
		"--input-sha256", f.inputDigest,
		"--policy", policyPath,
		"--policy-sha256", f.policyFile,
		"--reproducer-id", "did:zrn:reproducer-a",
		"--output", outputPath,
		"--report", filepath.Join(root, "report.json"),
	}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("unexpected existing-output result code=%d stderr=%q", code, stderr.String())
	}
	content, _ := os.ReadFile(outputPath)
	if string(content) != "preserve" {
		t.Fatal("existing output was modified")
	}
}

func TestCompileGenesisRejectsAmbiguousOrNonCanonicalExport(t *testing.T) {
	f := newFixture(t)
	duplicate := append([]byte(`{"chain_id":"shadow",`), f.input[1:]...)
	_, _, err := compileGenesis(
		duplicate,
		digest(duplicate),
		f.policy,
		f.policyFile,
		"did:zrn:reproducer-a",
	)
	if err == nil || !strings.Contains(err.Error(), `duplicate JSON object key "chain_id"`) {
		t.Fatalf("duplicate key was not rejected: %v", err)
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, f.input, "", "  "); err != nil {
		t.Fatal(err)
	}
	indentedPolicy := f.policy
	indentedPolicy.SourceExportSHA256 = digest(indented.Bytes())
	_, _, err = compileGenesis(
		indented.Bytes(),
		digest(indented.Bytes()),
		indentedPolicy,
		f.policyFile,
		"did:zrn:reproducer-a",
	)
	if err == nil || !strings.Contains(err.Error(), "exact compact canonical SDK export JSON") {
		t.Fatalf("non-canonical export was not rejected: %v", err)
	}
}

func TestWriteNewAtomicIsNoReplaceUnderContention(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "contended.json")
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, body := range [][]byte{[]byte("first"), []byte("second")} {
		body := append([]byte(nil), body...)
		go func() {
			<-start
			results <- writeNewAtomic(outputPath, body)
		}()
	}
	close(start)
	firstErr := <-results
	secondErr := <-results
	successes := 0
	existing := 0
	for _, result := range []error{firstErr, secondErr} {
		switch {
		case result == nil:
			successes++
		case strings.Contains(result.Error(), "already exists"):
			existing++
		default:
			t.Fatalf("unexpected no-replace result: %v", result)
		}
	}
	if successes != 1 || existing != 1 {
		t.Fatalf("want one install and one refusal, got success=%d existing=%d", successes, existing)
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "first" && string(contents) != "second" {
		t.Fatalf("output was corrupted: %q", contents)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("output mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestWriteNewAtomicRejectsSymlinkedParent(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	realDirectory := filepath.Join(root, "real")
	if err := os.Mkdir(realDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	linkDirectory := filepath.Join(root, "link")
	if err := os.Symlink(realDirectory, linkDirectory); err != nil {
		t.Fatal(err)
	}
	err = writeNewAtomic(filepath.Join(linkDirectory, "output"), []byte("blocked"))
	if err == nil || !strings.Contains(err.Error(), "real non-symlink directory") {
		t.Fatalf("symlinked output parent was not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(realDirectory, "output")); !os.IsNotExist(err) {
		t.Fatalf("output crossed a symlinked parent: %v", err)
	}
}

func mutateAppState(t *testing.T, f *fixture, mutate func(map[string]any)) {
	t.Helper()
	appGenesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(f.input))
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	decoder := json.NewDecoder(bytes.NewReader(appGenesis.AppState))
	decoder.UseNumber()
	if err := decoder.Decode(&state); err != nil {
		t.Fatal(err)
	}
	mutate(state)
	appGenesis.AppState, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	f.input, err = json.Marshal(appGenesis)
	if err != nil {
		t.Fatal(err)
	}
	f.inputDigest = digest(f.input)
	f.policy.SourceExportSHA256 = f.inputDigest
	unsigned := f.policy
	unsigned.PolicySHA256 = ""
	unsignedBytes, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	f.policy.PolicySHA256 = digest(unsignedBytes)
	f.policyBytes, err = json.Marshal(f.policy)
	if err != nil {
		t.Fatal(err)
	}
	f.policyFile = digest(f.policyBytes)
}

func TestDigestKnownAnswer(t *testing.T) {
	sum := sha256.Sum256([]byte("zerone"))
	if digest([]byte("zerone")) != base64ToHex(sum[:]) {
		t.Fatal("digest helper mismatch")
	}
}

func base64ToHex(value []byte) string {
	const alphabet = "0123456789abcdef"
	output := make([]byte, len(value)*2)
	for i, item := range value {
		output[i*2] = alphabet[item>>4]
		output[i*2+1] = alphabet[item&0x0f]
	}
	return string(output)
}
