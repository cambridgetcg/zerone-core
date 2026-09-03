// fork-genesis compiles a narrowly scoped, non-zero-height continuation
// genesis that replaces one suspect consensus key while preserving the
// validator operator, accounts, balances, staking economics, and history.
//
// It intentionally supports only the consensus-key-only profile. Any need to
// replace the operator/governance account, reset live IBC state, rebase
// clocks, or rewrite economics is a hard failure requiring a different,
// independently reviewed compiler.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmttypes "github.com/cometbft/cometbft/types"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	scheduletypes "github.com/zerone-chain/zerone/x/schedule/types"
	zeronestakingtypes "github.com/zerone-chain/zerone/x/staking/types"
	"golang.org/x/sys/unix"
)

const (
	policySchema       = "zerone.fork-genesis.policy/v1"
	reportSchema       = "zerone.fork-genesis.report/v2"
	compilerProfile    = "consensus-key-only"
	operatorRetain     = "RETAIN_PROVEN_SAFE"
	ibcDisposition     = "require-empty"
	emergencyStartMode = "CONSENSUS_QUARANTINE"
	maxPolicyBytes     = 1 << 20
	maxGenesisBytes    = 256 << 20

	retiredScheduleModuleName       = "schedule"
	messageScheduleGenesisMigration = "message-schedule-v1-absent-to-default-closed"
	// absentModuleSHA256 is SHA-256 over the canonical JSON literal `null`.
	// It is a report-only sentinel: the source module key is proven absent and
	// the target module's real digest remains in after_sha256.
	absentModuleSHA256 = "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"
)

var requiredModules = []string{
	"auth",
	"bank",
	"distribution",
	"emergency",
	"evidence",
	"feeibc",
	"genutil",
	"gov",
	"ibc",
	"ibcratelimit",
	"interchainaccounts",
	"slashing",
	"staking",
	"transfer",
	"upgrade",
	"zerone_gov",
	"zerone_staking",
}

type rewritePolicy struct {
	Schema                           string                  `json:"schema"`
	Profile                          string                  `json:"profile"`
	IncidentID                       string                  `json:"incident_id"`
	SourceChainID                    string                  `json:"source_chain_id"`
	TargetChainID                    string                  `json:"target_chain_id"`
	SourceLastHeight                 uint64                  `json:"source_last_height"`
	SourceBlockIDSHA256              string                  `json:"source_block_id_sha256"`
	SourceAppHashSHA256              string                  `json:"source_app_hash_sha256"`
	SourceLastBlockTime              string                  `json:"source_last_block_time"`
	SourceSignedCommitSHA256         string                  `json:"source_signed_commit_sha256"`
	SourceValidatorSetSHA256         string                  `json:"source_validator_set_sha256"`
	SourceExportSHA256               string                  `json:"source_export_sha256"`
	OldOperatorAddress               string                  `json:"old_operator_address"`
	OldConsensusPublicKey            string                  `json:"old_consensus_public_key"`
	NewConsensusPublicKey            string                  `json:"new_consensus_public_key"`
	OperatorDisposition              string                  `json:"operator_disposition"`
	ProhibitedConsensusPublicKeys    []string                `json:"prohibited_consensus_public_keys"`
	ProhibitedPrivilegedIdentities   []string                `json:"prohibited_privileged_identities"`
	ExpectedOldCustomConsensusPubkey string                  `json:"expected_old_custom_consensus_pubkey"`
	NewCustomConsensusPubkey         string                  `json:"new_custom_consensus_pubkey"`
	TargetAppVersion                 string                  `json:"target_app_version"`
	TargetGenesisTime                string                  `json:"target_genesis_time"`
	CustodyAssessmentSHA256          string                  `json:"custody_assessment_sha256"`
	ForkPolicySHA256                 string                  `json:"fork_policy_sha256"`
	RewriteToolSHA256                string                  `json:"rewrite_tool_sha256"`
	IBCDisposition                   string                  `json:"ibc_disposition"`
	EmergencyStartMode               string                  `json:"emergency_start_mode"`
	IndependentReproducers           []independentReproducer `json:"independent_reproducers"`
	PolicySHA256                     string                  `json:"policy_sha256"`
}

type independentReproducer struct {
	Identity      string `json:"identity"`
	ControlDomain string `json:"control_domain"`
	PublicKey     string `json:"public_key"`
}

type moduleDigest struct {
	Module       string `json:"module"`
	BeforeSHA256 string `json:"before_sha256"`
	AfterSHA256  string `json:"after_sha256"`
	Changed      bool   `json:"changed"`
}

type compilerReport struct {
	Schema                   string         `json:"schema"`
	Profile                  string         `json:"profile"`
	ReproducerID             string         `json:"reproducer_id"`
	ReproducerControlDomain  string         `json:"reproducer_control_domain"`
	ReproducerPublicKey      string         `json:"reproducer_public_key"`
	IncidentID               string         `json:"incident_id"`
	SourceGenesisSHA256      string         `json:"source_genesis_sha256"`
	PolicyFileSHA256         string         `json:"policy_file_sha256"`
	PolicySHA256             string         `json:"policy_sha256"`
	SourceChainID            string         `json:"source_chain_id"`
	TargetChainID            string         `json:"target_chain_id"`
	InitialHeight            uint64         `json:"initial_height"`
	SourceBlockIDSHA256      string         `json:"source_block_id_sha256"`
	SourceAppHashSHA256      string         `json:"source_app_hash_sha256"`
	SourceLastBlockTime      string         `json:"source_last_block_time"`
	SourceSignedCommitSHA256 string         `json:"source_signed_commit_sha256"`
	SourceValidatorSetSHA256 string         `json:"source_validator_set_sha256"`
	OldConsensusAddress      string         `json:"old_consensus_address"`
	NewConsensusAddress      string         `json:"new_consensus_address"`
	OldConsensusPublicKey    string         `json:"old_consensus_public_key"`
	NewConsensusPublicKey    string         `json:"new_consensus_public_key"`
	OperatorDisposition      string         `json:"operator_disposition"`
	CustodyAssessmentSHA256  string         `json:"custody_assessment_sha256"`
	ForkPolicySHA256         string         `json:"fork_policy_sha256"`
	RewriteToolSHA256        string         `json:"rewrite_tool_sha256"`
	IBCDisposition           string         `json:"ibc_disposition"`
	EmergencyStartMode       string         `json:"emergency_start_mode"`
	SchemaMigrations         []string       `json:"schema_migrations"`
	ModuleDigests            []moduleDigest `json:"module_digests"`
	OutputGenesisSHA256      string         `json:"output_genesis_sha256"`
	ReportSHA256             string         `json:"report_sha256"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("fork-genesis", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputPath := flags.String("input", "", "non-zero-height continuation genesis export")
	inputSHA256 := flags.String("input-sha256", "", "expected SHA-256 of the exact input bytes")
	policyPath := flags.String("policy", "", "canonical rewrite policy")
	policyFileSHA256 := flags.String("policy-sha256", "", "expected SHA-256 of the exact policy bytes")
	reproducerID := flags.String("reproducer-id", "", "independent reproducer identity selected by the policy")
	outputPath := flags.String("output", "", "new output path for the canonical target genesis")
	reportPath := flags.String("report", "", "new output path for the canonical compiler report")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: fork-genesis --input <export.json> --input-sha256 <hex> --policy <policy.json> --policy-sha256 <hex> --reproducer-id <id> --output <genesis.json> --report <report.json>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "fork-genesis: positional arguments are not accepted")
		return 2
	}
	for name, value := range map[string]string{
		"--input":         *inputPath,
		"--input-sha256":  *inputSHA256,
		"--policy":        *policyPath,
		"--policy-sha256": *policyFileSHA256,
		"--reproducer-id": *reproducerID,
		"--output":        *outputPath,
		"--report":        *reportPath,
	} {
		if value == "" {
			fmt.Fprintf(stderr, "fork-genesis: %s is required\n", name)
			return 2
		}
	}
	if *outputPath == *reportPath {
		fmt.Fprintln(stderr, "fork-genesis: output and report paths must differ")
		return 2
	}
	if err := validateSHA256("input SHA-256", *inputSHA256); err != nil {
		fmt.Fprintf(stderr, "fork-genesis: %v\n", err)
		return 2
	}
	if err := validateSHA256("policy file SHA-256", *policyFileSHA256); err != nil {
		fmt.Fprintf(stderr, "fork-genesis: %v\n", err)
		return 2
	}

	policyBytes, err := readRegularBounded(*policyPath, maxPolicyBytes)
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: read policy: %v\n", err)
		return 2
	}
	if digest(policyBytes) != *policyFileSHA256 {
		fmt.Fprintln(stderr, "fork-genesis: policy file SHA-256 mismatch")
		return 1
	}
	policy, err := decodePolicy(policyBytes)
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: invalid policy: %v\n", err)
		return 1
	}
	if err := validatePolicy(policy, *reproducerID); err != nil {
		fmt.Fprintf(stderr, "fork-genesis: invalid policy: %v\n", err)
		return 1
	}
	if policy.SourceExportSHA256 != *inputSHA256 {
		fmt.Fprintln(stderr, "fork-genesis: policy source export SHA-256 does not match --input-sha256")
		return 1
	}
	executableSHA256, err := currentExecutableDigest()
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: hash compiler executable: %v\n", err)
		return 2
	}
	if executableSHA256 != policy.RewriteToolSHA256 {
		fmt.Fprintln(stderr, "fork-genesis: compiler executable SHA-256 does not match the rewrite policy")
		return 1
	}

	inputBytes, err := readRegularBounded(*inputPath, maxGenesisBytes)
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: read input genesis: %v\n", err)
		return 2
	}
	if digest(inputBytes) != *inputSHA256 {
		fmt.Fprintln(stderr, "fork-genesis: input genesis SHA-256 mismatch")
		return 1
	}
	output, report, err := compileGenesis(
		inputBytes,
		*inputSHA256,
		policy,
		*policyFileSHA256,
		*reproducerID,
	)
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: NO-GO: %v\n", err)
		return 1
	}
	reportBytes, err := sealReport(report)
	if err != nil {
		fmt.Fprintf(stderr, "fork-genesis: seal report: %v\n", err)
		return 1
	}
	if err := writeNewAtomic(*outputPath, output); err != nil {
		fmt.Fprintf(stderr, "fork-genesis: write output genesis: %v\n", err)
		return 2
	}
	if err := writeNewAtomic(*reportPath, reportBytes); err != nil {
		fmt.Fprintf(stderr, "fork-genesis: write report: %v\n", err)
		return 2
	}
	fmt.Fprintf(
		stdout,
		"GO_FORK_REGENESIS profile=%s source_chain_id=%s target_chain_id=%s initial_height=%d output_genesis_sha256=%s report_self_sha256=%s report_file_sha256=%s\n",
		report.Profile,
		report.SourceChainID,
		report.TargetChainID,
		report.InitialHeight,
		report.OutputGenesisSHA256,
		report.ReportSHA256,
		digest(reportBytes),
	)
	return 0
}

func decodePolicy(data []byte) (rewritePolicy, error) {
	var policy rewritePolicy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return rewritePolicy{}, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return rewritePolicy{}, errors.New("multiple JSON values are not allowed")
		}
		return rewritePolicy{}, err
	}
	canonical, err := json.Marshal(policy)
	if err != nil {
		return rewritePolicy{}, err
	}
	if !bytes.Equal(data, canonical) {
		return rewritePolicy{}, errors.New("policy must be exact compact canonical JSON with no trailing newline")
	}
	return policy, nil
}

func validatePolicy(policy rewritePolicy, reproducerID string) error {
	if policy.Schema != policySchema {
		return fmt.Errorf("schema must be %q", policySchema)
	}
	if policy.Profile != compilerProfile {
		return fmt.Errorf("profile must be %q", compilerProfile)
	}
	if err := validateLabel("incident_id", policy.IncidentID, 256); err != nil {
		return err
	}
	if err := validateChainID("source_chain_id", policy.SourceChainID); err != nil {
		return err
	}
	if err := validateChainID("target_chain_id", policy.TargetChainID); err != nil {
		return err
	}
	if policy.SourceChainID == policy.TargetChainID {
		return errors.New("target_chain_id must differ from source_chain_id")
	}
	sourceRevision, err := chainRevision(policy.SourceChainID)
	if err != nil {
		return fmt.Errorf("source_chain_id: %w", err)
	}
	targetRevision, err := chainRevision(policy.TargetChainID)
	if err != nil {
		return fmt.Errorf("target_chain_id: %w", err)
	}
	if targetRevision <= sourceRevision {
		return fmt.Errorf("target chain revision %d must exceed source revision %d", targetRevision, sourceRevision)
	}
	if policy.SourceLastHeight == 0 || policy.SourceLastHeight == ^uint64(0) {
		return errors.New("source_last_height must permit a non-zero H+1 continuation")
	}
	for name, value := range map[string]string{
		"source_block_id_sha256":      policy.SourceBlockIDSHA256,
		"source_app_hash_sha256":      policy.SourceAppHashSHA256,
		"source_signed_commit_sha256": policy.SourceSignedCommitSHA256,
		"source_validator_set_sha256": policy.SourceValidatorSetSHA256,
		"source_export_sha256":        policy.SourceExportSHA256,
		"custody_assessment_sha256":   policy.CustodyAssessmentSHA256,
		"fork_policy_sha256":          policy.ForkPolicySHA256,
		"rewrite_tool_sha256":         policy.RewriteToolSHA256,
	} {
		if err := validateSHA256(name, value); err != nil {
			return err
		}
	}
	if _, err := sdk.ValAddressFromBech32(policy.OldOperatorAddress); err != nil {
		return fmt.Errorf("old_operator_address: %w", err)
	}
	oldKey, err := decodeEd25519PublicKey("old_consensus_public_key", policy.OldConsensusPublicKey)
	if err != nil {
		return err
	}
	newKey, err := decodeEd25519PublicKey("new_consensus_public_key", policy.NewConsensusPublicKey)
	if err != nil {
		return err
	}
	if bytes.Equal(oldKey, newKey) {
		return errors.New("new consensus public key must differ from the suspect old key")
	}
	if policy.OperatorDisposition != operatorRetain {
		return fmt.Errorf("operator_disposition must be %q for this profile", operatorRetain)
	}
	if len(policy.ProhibitedConsensusPublicKeys) != 1 ||
		policy.ProhibitedConsensusPublicKeys[0] != policy.OldConsensusPublicKey {
		return errors.New("prohibited_consensus_public_keys must contain exactly the suspect old consensus key")
	}
	if policy.ProhibitedPrivilegedIdentities == nil ||
		len(policy.ProhibitedPrivilegedIdentities) != 0 {
		return errors.New("consensus-key-only requires prohibited_privileged_identities to be [] after a PASS/RETAIN operator assessment")
	}
	if (policy.ExpectedOldCustomConsensusPubkey == "") != (policy.NewCustomConsensusPubkey == "") {
		return errors.New("custom consensus metadata values must be both empty or both populated")
	}
	if policy.ExpectedOldCustomConsensusPubkey != "" &&
		policy.ExpectedOldCustomConsensusPubkey == policy.NewCustomConsensusPubkey {
		return errors.New("new custom consensus metadata must differ from the old value")
	}
	for name, value := range map[string]string{
		"expected_old_custom_consensus_pubkey": policy.ExpectedOldCustomConsensusPubkey,
		"new_custom_consensus_pubkey":          policy.NewCustomConsensusPubkey,
		"target_app_version":                   policy.TargetAppVersion,
	} {
		if value != "" {
			if err := validateLabel(name, value, 512); err != nil {
				return err
			}
		}
	}
	if policy.TargetAppVersion == "" {
		return errors.New("target_app_version is required")
	}
	sourceLastBlockTime, err := parseCanonicalUTCTime(
		"source_last_block_time",
		policy.SourceLastBlockTime,
	)
	if err != nil {
		return err
	}
	targetGenesisTime, err := parseCanonicalUTCTime(
		"target_genesis_time",
		policy.TargetGenesisTime,
	)
	if err != nil {
		return err
	}
	if !targetGenesisTime.After(sourceLastBlockTime) {
		return errors.New("target_genesis_time must be strictly later than source_last_block_time")
	}
	if policy.IBCDisposition != ibcDisposition {
		return fmt.Errorf("ibc_disposition must be %q", ibcDisposition)
	}
	if policy.EmergencyStartMode != emergencyStartMode {
		return fmt.Errorf("emergency_start_mode must be %q", emergencyStartMode)
	}
	if len(policy.IndependentReproducers) < 2 {
		return errors.New("at least two independent reproducer identities are required")
	}
	foundReproducer := false
	identities := make(map[string]bool, len(policy.IndependentReproducers))
	controlDomains := make(map[string]bool, len(policy.IndependentReproducers))
	publicKeys := make(map[string]bool, len(policy.IndependentReproducers))
	for i, reproducer := range policy.IndependentReproducers {
		if err := validateLabel("independent reproducer identity", reproducer.Identity, 256); err != nil {
			return err
		}
		if err := validateLabel("independent reproducer control domain", reproducer.ControlDomain, 256); err != nil {
			return err
		}
		if err := validateEd25519HexPublicKey("independent reproducer public key", reproducer.PublicKey); err != nil {
			return err
		}
		if i > 0 && !reproducerLess(policy.IndependentReproducers[i-1], reproducer) {
			return errors.New("independent_reproducers must be unique and strictly sorted")
		}
		if identities[reproducer.Identity] ||
			controlDomains[reproducer.ControlDomain] ||
			publicKeys[reproducer.PublicKey] {
			return errors.New("independent_reproducers must use distinct identities, control domains, and public keys")
		}
		identities[reproducer.Identity] = true
		controlDomains[reproducer.ControlDomain] = true
		publicKeys[reproducer.PublicKey] = true
		if reproducer.Identity == reproducerID {
			foundReproducer = true
		}
	}
	if !foundReproducer {
		return fmt.Errorf("reproducer_id %q is not authorized by the policy", reproducerID)
	}
	if err := validateSHA256("policy_sha256", policy.PolicySHA256); err != nil {
		return err
	}
	unsigned := policy
	unsigned.PolicySHA256 = ""
	canonical, err := json.Marshal(unsigned)
	if err != nil {
		return err
	}
	if digest(canonical) != policy.PolicySHA256 {
		return errors.New("policy_sha256 does not match canonical policy with policy_sha256 empty")
	}
	return nil
}

func compileGenesis(
	input []byte,
	inputSHA256 string,
	policy rewritePolicy,
	policyFileSHA256 string,
	reproducerID string,
) ([]byte, compilerReport, error) {
	if digest(input) != inputSHA256 {
		return nil, compilerReport{}, errors.New("input bytes do not match the supplied source export digest")
	}
	if err := rejectDuplicateJSONKeys(input); err != nil {
		return nil, compilerReport{}, fmt.Errorf("source export JSON is ambiguous: %w", err)
	}
	if policy.SourceExportSHA256 != inputSHA256 {
		return nil, compilerReport{}, errors.New("rewrite policy does not bind the supplied source export digest")
	}
	reproducer, found := findReproducer(policy.IndependentReproducers, reproducerID)
	if !found {
		return nil, compilerReport{}, errors.New("rewrite policy does not authorize the selected reproducer")
	}
	appGenesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(input))
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode app genesis: %w", err)
	}
	canonicalInput, err := json.Marshal(appGenesis)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("canonicalize SDK source export: %w", err)
	}
	if !bytes.Equal(input, canonicalInput) {
		return nil, compilerReport{}, errors.New(
			"source export must be exact compact canonical SDK export JSON; capture zeroned export stdout without reformatting",
		)
	}
	if appGenesis.AppName != "zeroned" {
		return nil, compilerReport{}, fmt.Errorf("app_name must be zeroned, got %q", appGenesis.AppName)
	}
	if appGenesis.ChainID != policy.SourceChainID {
		return nil, compilerReport{}, fmt.Errorf("input chain ID %q does not match policy %q", appGenesis.ChainID, policy.SourceChainID)
	}
	if appGenesis.InitialHeight <= 1 || uint64(appGenesis.InitialHeight) != policy.SourceLastHeight+1 {
		return nil, compilerReport{}, fmt.Errorf(
			"input initial height %d must equal non-zero source H+1 (%d)",
			appGenesis.InitialHeight,
			policy.SourceLastHeight+1,
		)
	}
	sourceLastBlockTime, err := parseCanonicalUTCTime(
		"source_last_block_time",
		policy.SourceLastBlockTime,
	)
	if err != nil {
		return nil, compilerReport{}, err
	}
	targetGenesisTime, err := parseCanonicalUTCTime(
		"target_genesis_time",
		policy.TargetGenesisTime,
	)
	if err != nil {
		return nil, compilerReport{}, err
	}
	if sourceLastBlockTime.Before(appGenesis.GenesisTime) {
		return nil, compilerReport{}, errors.New("source last block time cannot precede the export's original genesis time")
	}
	if !targetGenesisTime.After(sourceLastBlockTime) {
		return nil, compilerReport{}, errors.New("target genesis time must be strictly later than the signed source checkpoint time")
	}
	if appGenesis.Consensus == nil || len(appGenesis.Consensus.Validators) != 1 {
		return nil, compilerReport{}, errors.New("consensus-key-only profile requires exactly one exported consensus validator")
	}

	var appState map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(appGenesis.AppState))
	if err := decoder.Decode(&appState); err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode app_state: %w", err)
	}
	if err := requireModules(appState); err != nil {
		return nil, compilerReport{}, err
	}
	encodingConfig := zeroneapp.MakeEncodingConfig()
	if err := requireNoSchedulerBankBalances(
		encodingConfig.Codec,
		appState[banktypes.ModuleName],
	); err != nil {
		return nil, compilerReport{}, err
	}
	if err := requireEmptyIBC(appState); err != nil {
		return nil, compilerReport{}, err
	}
	if err := requireNoPendingGovernance(appState); err != nil {
		return nil, compilerReport{}, err
	}
	if err := requireNoGenesisTransactions(appState); err != nil {
		return nil, compilerReport{}, err
	}
	if err := requireNoPendingEvidence(appState); err != nil {
		return nil, compilerReport{}, err
	}

	// Freeze the source inventory before adding any target-only module. An
	// absent source module is materially different from an unchanged module and
	// must remain visible in the compiler report.
	beforeDigests, err := digestModules(appState)
	if err != nil {
		return nil, compilerReport{}, err
	}
	if err := initializeFreshScheduleGenesis(appState); err != nil {
		return nil, compilerReport{}, err
	}
	schemaMigrations, err := normalizeTargetModuleSchemas(appState)
	if err != nil {
		return nil, compilerReport{}, err
	}
	schemaMigrations = append(schemaMigrations, messageScheduleGenesisMigration)

	var stakingGenesis stakingtypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["staking"], &stakingGenesis); err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode staking genesis: %w", err)
	}
	oldKeyBytes, _ := decodeEd25519PublicKey("old consensus public key", policy.OldConsensusPublicKey)
	newKeyBytes, _ := decodeEd25519PublicKey("new consensus public key", policy.NewConsensusPublicKey)
	oldCmtKey := cmted25519.PubKey(oldKeyBytes)
	newCmtKey := cmted25519.PubKey(newKeyBytes)
	oldConsensusAddress := sdk.ConsAddress(oldCmtKey.Address()).String()
	newConsensusAddress := sdk.ConsAddress(newCmtKey.Address()).String()
	if oldConsensusAddress == newConsensusAddress {
		return nil, compilerReport{}, errors.New("old and new consensus addresses unexpectedly match")
	}

	stakingIndex := -1
	bondedValidators := 0
	for i := range stakingGenesis.Validators {
		validator := stakingGenesis.Validators[i]
		if validator.Status == stakingtypes.Bonded {
			bondedValidators++
		}
		if validator.OperatorAddress != policy.OldOperatorAddress {
			continue
		}
		if stakingIndex != -1 {
			return nil, compilerReport{}, errors.New("duplicate target staking validator")
		}
		stakingIndex = i
	}
	if stakingIndex == -1 {
		return nil, compilerReport{}, errors.New("target staking validator is absent")
	}
	if bondedValidators != 1 {
		return nil, compilerReport{}, fmt.Errorf(
			"consensus-key-only profile requires exactly one bonded staking validator, found %d",
			bondedValidators,
		)
	}
	targetValidator := stakingGenesis.Validators[stakingIndex]
	oldStoredAddress, err := targetValidator.GetConsAddr()
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode stored consensus key: %w", err)
	}
	if !bytes.Equal(oldStoredAddress, oldCmtKey.Address()) {
		return nil, compilerReport{}, errors.New("stored staking consensus key does not match policy old key")
	}
	if targetValidator.Jailed || targetValidator.Status != stakingtypes.Bonded {
		return nil, compilerReport{}, errors.New("consensus-key-only profile requires the one target validator to be bonded and unjailed")
	}
	newSDKKey := &sdked25519.PubKey{Key: append([]byte(nil), newKeyBytes...)}
	newAny, err := codectypes.NewAnyWithValue(newSDKKey)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode replacement consensus key: %w", err)
	}
	targetValidator.ConsensusPubkey = newAny
	stakingGenesis.Validators[stakingIndex] = targetValidator

	exportedValidator := appGenesis.Consensus.Validators[0]
	if !bytes.Equal(exportedValidator.PubKey.Bytes(), oldKeyBytes) {
		return nil, compilerReport{}, errors.New("exported consensus validator does not match policy old key")
	}
	expectedPower := targetValidator.ConsensusPower(sdk.DefaultPowerReduction)
	if expectedPower <= 0 || exportedValidator.Power != expectedPower {
		return nil, compilerReport{}, fmt.Errorf(
			"exported validator power %d does not match staking power %d",
			exportedValidator.Power,
			expectedPower,
		)
	}
	if !stakingGenesis.Exported ||
		len(stakingGenesis.LastValidatorPowers) != 1 ||
		stakingGenesis.LastValidatorPowers[0].Address != policy.OldOperatorAddress ||
		stakingGenesis.LastValidatorPowers[0].Power != expectedPower ||
		!stakingGenesis.LastTotalPower.Equal(targetValidator.Tokens) {
		return nil, compilerReport{}, errors.New(
			"exported staking last-power indexes do not exactly match the sole bonded validator",
		)
	}
	exportedValidator.PubKey = newCmtKey
	exportedValidator.Address = newCmtKey.Address()
	appGenesis.Consensus.Validators[0] = exportedValidator

	var slashingGenesis slashingtypes.GenesisState
	if err := encodingConfig.Codec.UnmarshalJSON(appState["slashing"], &slashingGenesis); err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode slashing genesis: %w", err)
	}
	signingInfoMatches := 0
	for i := range slashingGenesis.SigningInfos {
		if slashingGenesis.SigningInfos[i].Address != oldConsensusAddress {
			continue
		}
		if slashingGenesis.SigningInfos[i].ValidatorSigningInfo.Tombstoned {
			return nil, compilerReport{}, errors.New("consensus-key-only profile refuses a tombstoned old validator")
		}
		signingInfoMatches++
		slashingGenesis.SigningInfos[i].Address = newConsensusAddress
		slashingGenesis.SigningInfos[i].ValidatorSigningInfo.Address = newConsensusAddress
	}
	if signingInfoMatches != 1 {
		return nil, compilerReport{}, fmt.Errorf("expected one old slashing signing info, found %d", signingInfoMatches)
	}
	missedBlockMatches := 0
	for i := range slashingGenesis.MissedBlocks {
		if slashingGenesis.MissedBlocks[i].Address == oldConsensusAddress {
			missedBlockMatches++
			slashingGenesis.MissedBlocks[i].Address = newConsensusAddress
		}
	}
	if missedBlockMatches > 1 {
		return nil, compilerReport{}, errors.New("duplicate old validator missed-block records")
	}

	var customStaking zeronestakingtypes.GenesisState
	if err := json.Unmarshal(appState["zerone_staking"], &customStaking); err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode zerone_staking genesis: %w", err)
	}
	valAddress, _ := sdk.ValAddressFromBech32(policy.OldOperatorAddress)
	accountAddress := sdk.AccAddress(valAddress).String()
	customMatches := 0
	for _, validator := range customStaking.Validators {
		if validator == nil || validator.OperatorAddress != accountAddress {
			continue
		}
		customMatches++
		if policy.ExpectedOldCustomConsensusPubkey == "" || policy.NewCustomConsensusPubkey == "" {
			return nil, compilerReport{}, errors.New("custom validator exists but policy does not bind its metadata rewrite")
		}
		if validator.ConsensusPubkey != policy.ExpectedOldCustomConsensusPubkey {
			return nil, compilerReport{}, errors.New("custom validator consensus metadata does not match policy")
		}
		validator.ConsensusPubkey = policy.NewCustomConsensusPubkey
	}
	if customMatches > 1 {
		return nil, compilerReport{}, errors.New("duplicate custom validator records for the SDK operator")
	}
	if customMatches == 0 &&
		(policy.ExpectedOldCustomConsensusPubkey != "" || policy.NewCustomConsensusPubkey != "") {
		return nil, compilerReport{}, errors.New("policy binds custom validator metadata but no matching record exists")
	}

	var emergencyGenesis emergencytypes.GenesisState
	if err := json.Unmarshal(appState["emergency"], &emergencyGenesis); err != nil {
		return nil, compilerReport{}, fmt.Errorf("decode emergency genesis: %w", err)
	}
	if emergencyGenesis.Status != string(emergencytypes.StatusNormal) ||
		emergencyGenesis.ActiveHaltCeremonyId != "" ||
		emergencyGenesis.HaltStartBlock != 0 ||
		emergencyGenesis.RecoveryAuthorization != nil {
		return nil, compilerReport{}, errors.New("source emergency state must be normal with no active quarantine or recovery authorization")
	}
	for _, ceremony := range emergencyGenesis.Ceremonies {
		if ceremony != nil &&
			ceremony.Phase != string(emergencytypes.PhaseFinalized) &&
			ceremony.Phase != string(emergencytypes.PhaseFailed) {
			return nil, compilerReport{}, errors.New("source emergency state contains a non-terminal ceremony")
		}
	}
	emergencyGenesis.Status = string(emergencytypes.StatusHalted)
	emergencyGenesis.ActiveHaltCeremonyId = "legacy-genesis-quarantine"
	emergencyGenesis.HaltStartBlock = uint64(appGenesis.InitialHeight)
	emergencyGenesis.LastHaltEscalationBlock = 0
	emergencyGenesis.QuarantineReleaseBlock = 0
	emergencyGenesis.RecoveryAuthorization = nil

	appState["staking"], err = encodingConfig.Codec.MarshalJSON(&stakingGenesis)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode staking genesis: %w", err)
	}
	appState["slashing"], err = encodingConfig.Codec.MarshalJSON(&slashingGenesis)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode slashing genesis: %w", err)
	}
	appState["zerone_staking"], err = json.Marshal(&customStaking)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode zerone_staking genesis: %w", err)
	}
	appState["emergency"], err = json.Marshal(&emergencyGenesis)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode emergency genesis: %w", err)
	}
	if err := zeroneapp.ModuleBasics.ValidateGenesis(
		encodingConfig.Codec,
		encodingConfig.TxConfig,
		appState,
	); err != nil {
		return nil, compilerReport{}, fmt.Errorf("target module genesis validation failed: %w", err)
	}

	afterDigests, err := digestModules(appState)
	if err != nil {
		return nil, compilerReport{}, err
	}
	transferChanged := beforeDigests["transfer"] != afterDigests["transfer"]
	if transferChanged != containsString(schemaMigrations, "ibc-transfer-v8-empty-denom-traces-to-v10-empty-denoms") {
		return nil, compilerReport{}, errors.New("transfer genesis changed without the exact declared empty-state schema migration")
	}
	ibcChanged := beforeDigests["ibc"] != afterDigests["ibc"]
	if ibcChanged != containsString(schemaMigrations, "ibc-core-v8-empty-to-v10-empty-v2-state") {
		return nil, compilerReport{}, errors.New("IBC core genesis changed without the exact declared empty-state schema migration")
	}
	for _, module := range []string{"emergency", "slashing", "staking"} {
		if beforeDigests[module] == afterDigests[module] {
			return nil, compilerReport{}, fmt.Errorf("required %s recovery rewrite made no semantic change", module)
		}
	}
	customChanged := beforeDigests["zerone_staking"] != afterDigests["zerone_staking"]
	if customChanged != (customMatches == 1) {
		return nil, compilerReport{}, errors.New("zerone_staking changed without exactly one policy-bound metadata rewrite")
	}
	if _, found := beforeDigests[scheduletypes.ModuleName]; found {
		return nil, compilerReport{}, fmt.Errorf("source digest inventory unexpectedly contains %s", scheduletypes.ModuleName)
	}
	if _, found := afterDigests[scheduletypes.ModuleName]; !found {
		return nil, compilerReport{}, fmt.Errorf("target digest inventory omits added %s", scheduletypes.ModuleName)
	}
	if !containsString(schemaMigrations, messageScheduleGenesisMigration) {
		return nil, compilerReport{}, errors.New("message_schedule was added without its exact declared schema migration")
	}
	allowedChanges := map[string]bool{
		"emergency":      true,
		"ibc":            ibcChanged,
		"slashing":       true,
		"staking":        true,
		"transfer":       transferChanged,
		"zerone_staking": customChanged,
	}
	for module, before := range beforeDigests {
		after, found := afterDigests[module]
		if !found {
			return nil, compilerReport{}, fmt.Errorf("target genesis removed module %q", module)
		}
		if before != after && !allowedChanges[module] {
			return nil, compilerReport{}, fmt.Errorf("consensus-key-only profile unexpectedly changed %s genesis", module)
		}
	}
	if len(afterDigests) != len(beforeDigests)+1 {
		return nil, compilerReport{}, errors.New("target genesis module key set must add only message_schedule")
	}
	for module := range afterDigests {
		if _, found := beforeDigests[module]; !found && module != scheduletypes.ModuleName {
			return nil, compilerReport{}, fmt.Errorf("target genesis unexpectedly added module %q", module)
		}
	}
	moduleNames := make([]string, 0, len(afterDigests))
	for module := range afterDigests {
		moduleNames = append(moduleNames, module)
	}
	sort.Strings(moduleNames)
	moduleDigests := make([]moduleDigest, 0, len(moduleNames))
	for _, module := range moduleNames {
		before := beforeDigests[module]
		if module == scheduletypes.ModuleName {
			before = absentModuleSHA256
		}
		moduleDigests = append(moduleDigests, moduleDigest{
			Module:       module,
			BeforeSHA256: before,
			AfterSHA256:  afterDigests[module],
			Changed:      before != afterDigests[module],
		})
	}

	appStateBytes, err := json.Marshal(appState)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode app_state: %w", err)
	}
	appGenesis.AppState = appStateBytes
	appGenesis.ChainID = policy.TargetChainID
	appGenesis.AppVersion = policy.TargetAppVersion
	appGenesis.GenesisTime = targetGenesisTime
	appGenesis.AppHash = nil
	if err := appGenesis.ValidateAndComplete(); err != nil {
		return nil, compilerReport{}, fmt.Errorf("target app genesis validation failed: %w", err)
	}
	output, err := json.Marshal(appGenesis)
	if err != nil {
		return nil, compilerReport{}, fmt.Errorf("encode target genesis: %w", err)
	}
	prohibitedEncodings := []string{
		policy.OldConsensusPublicKey,
		hex.EncodeToString(oldKeyBytes),
		strings.ToUpper(hex.EncodeToString(oldKeyBytes)),
		oldConsensusAddress,
		base64.StdEncoding.EncodeToString(oldCmtKey.Address()),
		hex.EncodeToString(oldCmtKey.Address()),
		strings.ToUpper(hex.EncodeToString(oldCmtKey.Address())),
	}
	for _, encoding := range prohibitedEncodings {
		if bytes.Contains(output, []byte(encoding)) {
			return nil, compilerReport{}, fmt.Errorf(
				"suspect old consensus key or address encoding %q remains in target genesis",
				encoding,
			)
		}
	}

	report := compilerReport{
		Schema:                   reportSchema,
		Profile:                  compilerProfile,
		ReproducerID:             reproducerID,
		ReproducerControlDomain:  reproducer.ControlDomain,
		ReproducerPublicKey:      reproducer.PublicKey,
		IncidentID:               policy.IncidentID,
		SourceGenesisSHA256:      inputSHA256,
		PolicyFileSHA256:         policyFileSHA256,
		PolicySHA256:             policy.PolicySHA256,
		SourceChainID:            policy.SourceChainID,
		TargetChainID:            policy.TargetChainID,
		InitialHeight:            uint64(appGenesis.InitialHeight),
		SourceBlockIDSHA256:      policy.SourceBlockIDSHA256,
		SourceAppHashSHA256:      policy.SourceAppHashSHA256,
		SourceLastBlockTime:      policy.SourceLastBlockTime,
		SourceSignedCommitSHA256: policy.SourceSignedCommitSHA256,
		SourceValidatorSetSHA256: policy.SourceValidatorSetSHA256,
		OldConsensusAddress:      oldConsensusAddress,
		NewConsensusAddress:      newConsensusAddress,
		OldConsensusPublicKey:    policy.OldConsensusPublicKey,
		NewConsensusPublicKey:    policy.NewConsensusPublicKey,
		OperatorDisposition:      policy.OperatorDisposition,
		CustodyAssessmentSHA256:  policy.CustodyAssessmentSHA256,
		ForkPolicySHA256:         policy.ForkPolicySHA256,
		RewriteToolSHA256:        policy.RewriteToolSHA256,
		IBCDisposition:           policy.IBCDisposition,
		EmergencyStartMode:       policy.EmergencyStartMode,
		SchemaMigrations:         schemaMigrations,
		ModuleDigests:            moduleDigests,
		OutputGenesisSHA256:      digest(output),
	}
	return output, report, nil
}

func requireModules(appState map[string]json.RawMessage) error {
	for _, module := range requiredModules {
		if _, found := appState[module]; !found {
			return fmt.Errorf("required module genesis %q is absent", module)
		}
	}
	return nil
}

func requireNoSchedulerBankBalances(
	cdc sdkcodec.JSONCodec,
	rawBankGenesis json.RawMessage,
) error {
	var bankGenesis banktypes.GenesisState
	if err := cdc.UnmarshalJSON(rawBankGenesis, &bankGenesis); err != nil {
		return fmt.Errorf("decode bank genesis for scheduler balance preflight: %w", err)
	}
	for _, namespace := range []struct {
		moduleName string
		fresh      bool
	}{
		{moduleName: retiredScheduleModuleName},
		{moduleName: scheduletypes.ModuleName, fresh: true},
	} {
		address := authtypes.NewModuleAddress(namespace.moduleName).String()
		for _, balance := range bankGenesis.Balances {
			if balance.Address != address || balance.Coins.IsZero() {
				continue
			}
			if namespace.fresh {
				return fmt.Errorf(
					"fork genesis requires fresh scheduler module account %q to have zero balance before default initialization; found %s",
					address,
					balance.Coins,
				)
			}
			return fmt.Errorf(
				"fork genesis refuses retired scheduler module account %q holding %s; reconcile old-format liabilities first",
				address,
				balance.Coins,
			)
		}
	}
	return nil
}

func initializeFreshScheduleGenesis(appState map[string]json.RawMessage) error {
	if _, found := appState[retiredScheduleModuleName]; found {
		return fmt.Errorf(
			"fork genesis refuses retired scheduler namespace %q; reconcile old-format liabilities instead of silently discarding state",
			retiredScheduleModuleName,
		)
	}
	if _, found := appState[scheduletypes.ModuleName]; found {
		return fmt.Errorf(
			"fork genesis requires source module %q to be absent; existing schedules and chain-bound occurrence IDs cannot be rewritten for the target chain",
			scheduletypes.ModuleName,
		)
	}

	genesis := scheduletypes.DefaultGenesis()
	if err := genesis.Validate(); err != nil {
		return fmt.Errorf("validate fresh %s default genesis: %w", scheduletypes.ModuleName, err)
	}
	if genesis.Params == nil || genesis.Params.AcceptNewSchedules ||
		len(genesis.Schedules) != 0 || len(genesis.Receipts) != 0 ||
		genesis.NextScheduleId != 1 || genesis.TotalEscrowUzrn != "0" {
		return fmt.Errorf("fresh %s default genesis is not empty and admission-closed", scheduletypes.ModuleName)
	}
	raw, err := json.Marshal(genesis)
	if err != nil {
		return fmt.Errorf("encode fresh %s default genesis: %w", scheduletypes.ModuleName, err)
	}
	appState[scheduletypes.ModuleName] = raw
	return nil
}

func requireNoGenesisTransactions(appState map[string]json.RawMessage) error {
	var genutil map[string]any
	if err := json.Unmarshal(appState["genutil"], &genutil); err != nil {
		return fmt.Errorf("decode genutil genesis: %w", err)
	}
	rawTransactions, found := genutil["gen_txs"]
	if !found {
		return errors.New("genutil.gen_txs is absent; input is not a proven stopped-height export")
	}
	transactions, ok := rawTransactions.([]any)
	if !ok || len(transactions) != 0 {
		return errors.New("genutil.gen_txs must be an empty array; initial genesis and pending genesis transactions are refused")
	}
	return nil
}

func requireNoPendingEvidence(appState map[string]json.RawMessage) error {
	var evidence map[string]any
	if err := json.Unmarshal(appState["evidence"], &evidence); err != nil {
		return fmt.Errorf("decode evidence genesis: %w", err)
	}
	rawEvidence, found := evidence["evidence"]
	if !found {
		return errors.New("evidence.evidence is absent; cannot prove quiescent evidence state")
	}
	items, ok := rawEvidence.([]any)
	if !ok || len(items) != 0 {
		return errors.New("evidence.evidence must be an empty array for consensus-key-only profile")
	}
	return nil
}

func requireEmptyIBC(appState map[string]json.RawMessage) error {
	checks := map[string][]string{
		"ibc": {
			"client_genesis.clients",
			"client_genesis.clients_consensus",
			"client_genesis.clients_metadata",
			"connection_genesis.connections",
			"connection_genesis.client_connection_paths",
			"channel_genesis.channels",
			"channel_genesis.acknowledgements",
			"channel_genesis.commitments",
			"channel_genesis.receipts",
			"channel_genesis.send_sequences",
			"channel_genesis.recv_sequences",
			"channel_genesis.ack_sequences",
		},
		"transfer": {
			"total_escrowed",
		},
		"interchainaccounts": {
			"controller_genesis_state.active_channels",
			"controller_genesis_state.interchain_accounts",
			"controller_genesis_state.ports",
			"host_genesis_state.active_channels",
			"host_genesis_state.interchain_accounts",
		},
		"feeibc": {
			"fee_enabled_channels",
			"forward_relayers",
			"identified_fees",
			"registered_counterparty_payees",
			"registered_payees",
		},
	}
	for module, paths := range checks {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(appState[module]))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return fmt.Errorf("decode %s for IBC emptiness: %w", module, err)
		}
		for _, path := range paths {
			resolved, found := resolveJSONPath(value, path)
			if !found {
				return fmt.Errorf("%s.%s is absent; cannot prove empty IBC state", module, path)
			}
			list, ok := resolved.([]any)
			if !ok || len(list) != 0 {
				return fmt.Errorf("%s.%s must be an empty array for fork profile", module, path)
			}
		}
		if module == "transfer" {
			legacy, hasLegacy := resolveJSONPath(value, "denom_traces")
			current, hasCurrent := resolveJSONPath(value, "denoms")
			if hasLegacy == hasCurrent {
				return errors.New("transfer genesis must contain exactly one of denom_traces or denoms")
			}
			registry := current
			path := "denoms"
			if hasLegacy {
				registry = legacy
				path = "denom_traces"
			}
			list, ok := registry.([]any)
			if !ok || len(list) != 0 {
				return fmt.Errorf("transfer.%s must be an empty array for fork profile", path)
			}
		}
		if module == "ibc" {
			clientV2, hasClientV2 := resolveJSONPath(value, "client_v2_genesis")
			channelV2, hasChannelV2 := resolveJSONPath(value, "channel_v2_genesis")
			if hasClientV2 != hasChannelV2 {
				return errors.New("IBC core genesis has a partial v2 state")
			}
			if hasClientV2 {
				v2Checks := []struct {
					fullPath     string
					root         any
					relativePath string
				}{
					{"client_v2_genesis.counterparty_infos", clientV2, "counterparty_infos"},
					{"channel_v2_genesis.acknowledgements", channelV2, "acknowledgements"},
					{"channel_v2_genesis.commitments", channelV2, "commitments"},
					{"channel_v2_genesis.receipts", channelV2, "receipts"},
					{"channel_v2_genesis.async_packets", channelV2, "async_packets"},
					{"channel_v2_genesis.send_sequences", channelV2, "send_sequences"},
				}
				for _, check := range v2Checks {
					resolved, found := resolveJSONPath(check.root, check.relativePath)
					if !found {
						return fmt.Errorf("ibc.%s is absent; cannot prove empty IBC v2 state", check.fullPath)
					}
					list, ok := resolved.([]any)
					if !ok || len(list) != 0 {
						return fmt.Errorf("ibc.%s must be an empty array for fork profile", check.fullPath)
					}
				}
			}
		}
	}
	var rateLimit map[string]any
	if err := json.Unmarshal(appState["ibcratelimit"], &rateLimit); err != nil {
		return fmt.Errorf("decode ibcratelimit genesis: %w", err)
	}
	if raw, found := rateLimit["rate_limits"]; found {
		limits, ok := raw.([]any)
		if !ok || len(limits) != 0 {
			return errors.New("ibcratelimit.rate_limits must be absent or an empty array for fork profile")
		}
	}
	return nil
}

func normalizeTargetModuleSchemas(appState map[string]json.RawMessage) ([]string, error) {
	migrations := make([]string, 0, 2)

	var transfer map[string]json.RawMessage
	if err := json.Unmarshal(appState["transfer"], &transfer); err != nil {
		return nil, fmt.Errorf("decode transfer genesis for target schema normalization: %w", err)
	}
	legacy, hasLegacy := transfer["denom_traces"]
	_, hasCurrent := transfer["denoms"]
	if hasLegacy == hasCurrent {
		return nil, errors.New("transfer genesis must contain exactly one of denom_traces or denoms")
	}
	if !hasCurrent {
		var traces []json.RawMessage
		if err := json.Unmarshal(legacy, &traces); err != nil {
			return nil, fmt.Errorf("decode transfer.denom_traces: %w", err)
		}
		if len(traces) != 0 {
			return nil, errors.New("non-empty legacy transfer denom_traces cannot be migrated by the require-empty profile")
		}
		delete(transfer, "denom_traces")
		transfer["denoms"] = json.RawMessage("[]")
		normalized, err := json.Marshal(transfer)
		if err != nil {
			return nil, fmt.Errorf("encode normalized transfer genesis: %w", err)
		}
		appState["transfer"] = normalized
		migrations = append(migrations, "ibc-transfer-v8-empty-denom-traces-to-v10-empty-denoms")
	}

	var ibc map[string]json.RawMessage
	if err := json.Unmarshal(appState["ibc"], &ibc); err != nil {
		return nil, fmt.Errorf("decode IBC core genesis for target schema normalization: %w", err)
	}
	_, hasClientV2 := ibc["client_v2_genesis"]
	_, hasChannelV2 := ibc["channel_v2_genesis"]
	var channel map[string]json.RawMessage
	if err := json.Unmarshal(ibc["channel_genesis"], &channel); err != nil {
		return nil, fmt.Errorf("decode IBC channel genesis for target schema normalization: %w", err)
	}
	legacyParams, hasLegacyParams := channel["params"]
	if hasClientV2 != hasChannelV2 {
		return nil, errors.New("IBC core genesis has a mixed v8/v10 shape")
	}
	if hasClientV2 && hasLegacyParams {
		return nil, errors.New("IBC v10 genesis must not retain the removed channel params")
	}
	if !hasClientV2 {
		if !hasLegacyParams {
			return nil, errors.New("IBC core genesis is neither the supported v8 shape nor the supported v10 shape")
		}
		expectedParams := []byte(`{"upgrade_timeout":{"height":{"revision_number":"0","revision_height":"0"},"timestamp":"600000000000"}}`)
		canonicalParams, err := canonicalJSON(legacyParams)
		if err != nil {
			return nil, fmt.Errorf("canonicalize legacy IBC channel params: %w", err)
		}
		canonicalExpected, _ := canonicalJSON(expectedParams)
		if !bytes.Equal(canonicalParams, canonicalExpected) {
			return nil, errors.New("legacy IBC channel params differ from the one reviewed empty-state v8 profile")
		}
		delete(channel, "params")
		ibc["client_v2_genesis"] = json.RawMessage(`{"counterparty_infos":[]}`)
		ibc["channel_v2_genesis"] = json.RawMessage(`{"acknowledgements":[],"commitments":[],"receipts":[],"async_packets":[],"send_sequences":[]}`)
		normalizedChannel, err := json.Marshal(channel)
		if err != nil {
			return nil, fmt.Errorf("encode normalized IBC channel genesis: %w", err)
		}
		ibc["channel_genesis"] = normalizedChannel
		normalizedIBC, err := json.Marshal(ibc)
		if err != nil {
			return nil, fmt.Errorf("encode normalized IBC core genesis: %w", err)
		}
		appState["ibc"] = normalizedIBC
		migrations = append(migrations, "ibc-core-v8-empty-to-v10-empty-v2-state")
	}
	return migrations, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func reproducerLess(left, right independentReproducer) bool {
	if left.Identity != right.Identity {
		return left.Identity < right.Identity
	}
	if left.ControlDomain != right.ControlDomain {
		return left.ControlDomain < right.ControlDomain
	}
	return left.PublicKey < right.PublicKey
}

func findReproducer(
	reproducers []independentReproducer,
	identity string,
) (independentReproducer, bool) {
	for _, reproducer := range reproducers {
		if reproducer.Identity == identity {
			return reproducer, true
		}
	}
	return independentReproducer{}, false
}

func requireNoPendingGovernance(appState map[string]json.RawMessage) error {
	var sdkGovernance any
	if err := json.Unmarshal(appState["gov"], &sdkGovernance); err != nil {
		return fmt.Errorf("decode gov governance state: %w", err)
	}
	for _, path := range []string{"proposals", "deposits", "votes"} {
		resolved, found := resolveJSONPath(sdkGovernance, path)
		if !found {
			return fmt.Errorf("gov.%s is absent; cannot prove that SDK governance is quiescent", path)
		}
		list, ok := resolved.([]any)
		if !ok || len(list) != 0 {
			return fmt.Errorf("gov.%s must be an empty array for narrow fork profile", path)
		}
	}

	var customGovernance zeronegovtypes.GenesisState
	if err := json.Unmarshal(appState["zerone_gov"], &customGovernance); err != nil {
		return fmt.Errorf("decode zerone_gov governance state: %w", err)
	}
	if customGovernance.Params == nil ||
		customGovernance.NextLipNumber == 0 ||
		customGovernance.NextSeatElectionNumber == 0 ||
		customGovernance.ResearchFundGovernance == nil {
		return errors.New("zerone_gov lacks required exported-state anchors; cannot prove reviewed schema")
	}
	if len(customGovernance.Lips) != 0 ||
		len(customGovernance.Votes) != 0 ||
		len(customGovernance.UpgradePlans) != 0 ||
		len(customGovernance.SeatElections) != 0 ||
		len(customGovernance.SeatElectionVotes) != 0 ||
		len(customGovernance.CreedAmendmentPins) != 0 ||
		customGovernance.EmergencyTransitionHold != nil {
		return errors.New("zerone_gov must contain no pending proposal, vote, upgrade, election, creed amendment, or emergency transition")
	}
	var upgrade map[string]any
	if err := json.Unmarshal(appState["upgrade"], &upgrade); err != nil {
		return fmt.Errorf("decode upgrade genesis: %w", err)
	}
	if len(upgrade) != 0 {
		return errors.New("upgrade genesis must be empty; pending/applied plan rewrite is outside this profile")
	}
	return nil
}

func resolveJSONPath(value any, path string) (any, bool) {
	current := value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func digestModules(appState map[string]json.RawMessage) (map[string]string, error) {
	result := make(map[string]string, len(appState))
	for module, raw := range appState {
		canonical, err := canonicalJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("canonicalize %s genesis: %w", module, err)
		}
		result[module] = digest(canonical)
	}
	return result, nil
}

func canonicalJSON(data []byte) ([]byte, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("multiple JSON values are not allowed")
	}
	return json.Marshal(value)
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanUniqueJSONValue(decoder, 0); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values are not allowed; found trailing token %v", token)
	}
	return nil
}

func scanUniqueJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 256 {
		return errors.New("JSON nesting exceeds 256 levels")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("JSON object did not end with }")
		}
	case '[':
		for decoder.More() {
			if err := scanUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("JSON array did not end with ]")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func sealReport(report compilerReport) ([]byte, error) {
	report.ReportSHA256 = ""
	unsigned, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	report.ReportSHA256 = digest(unsigned)
	return json.Marshal(report)
}

func readRegularBounded(path string, limit int64) ([]byte, error) {
	directoryFD, basename, err := openParentDirectoryNoSymlinks(path)
	if err != nil {
		return nil, err
	}
	defer unix.Close(directoryFD)
	fd, err := unix.Openat(
		directoryFD,
		basename,
		unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("input must be a regular non-symlink file: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("open input file descriptor")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("input must be a regular non-symlink file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("input exceeds %d-byte limit", limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("input exceeds %d-byte limit", limit)
	}
	return data, nil
}

func writeNewAtomic(path string, data []byte) error {
	directoryFD, basename, err := openParentDirectoryNoSymlinks(path)
	if err != nil {
		return err
	}
	defer unix.Close(directoryFD)

	var temporaryName string
	var temporaryFD int
	for attempts := 0; attempts < 32; attempts++ {
		randomBytes := make([]byte, 16)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("generate temporary output name: %w", err)
		}
		temporaryName = ".fork-genesis-" + hex.EncodeToString(randomBytes)
		temporaryFD, err = unix.Openat(
			directoryFD,
			temporaryName,
			unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW,
			0o600,
		)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("create temporary output: %w", err)
		}
		temporaryName = ""
	}
	if temporaryName == "" {
		return errors.New("could not allocate a unique temporary output name")
	}
	defer func() {
		if temporaryName != "" {
			_ = unix.Unlinkat(directoryFD, temporaryName, 0)
		}
	}()
	temporary := os.NewFile(uintptr(temporaryFD), temporaryName)
	if temporary == nil {
		_ = unix.Close(temporaryFD)
		return errors.New("open temporary output file descriptor")
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := unix.Linkat(directoryFD, temporaryName, directoryFD, basename, 0); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return errors.New("output path already exists")
		}
		return fmt.Errorf("install output without replacement: %w", err)
	}
	if err := unix.Unlinkat(directoryFD, temporaryName, 0); err != nil {
		_ = unix.Unlinkat(directoryFD, basename, 0)
		return fmt.Errorf("remove temporary output link: %w", err)
	}
	temporaryName = ""
	if err := unix.Fsync(directoryFD); err != nil {
		return fmt.Errorf("sync output directory: %w", err)
	}
	return nil
}

func openParentDirectoryNoSymlinks(path string) (int, string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return -1, "", errors.New("path must be absolute")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return -1, "", errors.New("path must be absolute and canonical")
	}
	basename := filepath.Base(clean)
	if basename == "." || basename == string(filepath.Separator) || basename == "" {
		return -1, "", errors.New("path must identify a file beneath an absolute directory")
	}
	directory := filepath.Dir(clean)
	relative := strings.TrimPrefix(directory, string(filepath.Separator))
	directoryFD, err := unix.Open(
		string(filepath.Separator),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return -1, "", fmt.Errorf("open filesystem root: %w", err)
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" {
			continue
		}
		nextFD, err := unix.Openat(
			directoryFD,
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
			0,
		)
		if err != nil {
			_ = unix.Close(directoryFD)
			return -1, "", fmt.Errorf(
				"path parent component %q must be a real non-symlink directory: %w",
				component,
				err,
			)
		}
		_ = unix.Close(directoryFD)
		directoryFD = nextFD
	}
	return directoryFD, basename, nil
}

func decodeEd25519PublicKey(name, value string) ([]byte, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return nil, fmt.Errorf("%s must be non-empty and trimmed", name)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be canonical base64: %w", name, err)
	}
	if len(decoded) != cmted25519.PubKeySize {
		return nil, fmt.Errorf("%s must decode to exactly %d bytes", name, cmted25519.PubKeySize)
	}
	if base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("%s must use canonical base64", name)
	}
	return decoded, nil
}

func validateEd25519HexPublicKey(name, value string) error {
	if len(value) != cmted25519.PubKeySize*2 {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be a lowercase hexadecimal Ed25519 public key", name)
	}
	return nil
}

func validateSHA256(name, value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", name)
	}
	return nil
}

func validateChainID(name, value string) error {
	if err := validateLabel(name, value, cmttypes.MaxChainIDLen); err != nil {
		return err
	}
	for _, character := range value {
		if !((character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == '-') {
			return fmt.Errorf("%s contains a non-canonical character", name)
		}
	}
	return nil
}

func chainRevision(chainID string) (uint64, error) {
	index := strings.LastIndexByte(chainID, '-')
	if index <= 0 || index == len(chainID)-1 {
		return 0, errors.New("chain ID must end in -<positive-revision>")
	}
	revisionText := chainID[index+1:]
	if revisionText[0] == '0' {
		return 0, errors.New("chain revision must be a canonical positive integer")
	}
	revision, err := strconv.ParseUint(revisionText, 10, 64)
	if err != nil || revision == 0 {
		return 0, errors.New("chain revision must be a canonical positive integer")
	}
	return revision, nil
}

func validateLabel(name, value string, maxLength int) error {
	if value == "" || len(value) > maxLength || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must be non-empty, trimmed, and at most %d bytes", name, maxLength)
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("%s contains a control character", name)
		}
	}
	return nil
}

func parseCanonicalUTCTime(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil ||
		parsed.Location() != time.UTC ||
		parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, fmt.Errorf("%s must be canonical UTC RFC3339 with Z", name)
	}
	return parsed, nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func currentExecutableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("compiler executable is not a regular file")
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
