// Command zerone2-artifact-audit is the fail-closed release gate for the
// zerone-2 genesis artifact set. It deliberately checks a single launch
// profile rather than accepting policy flags: changing any invariant requires
// a reviewed code change to this auditor.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	txsigning "cosmossdk.io/x/tx/signing"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktextsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	bip39 "github.com/cosmos/go-bip39"
	"golang.org/x/crypto/ripemd160"
	"google.golang.org/protobuf/types/known/anypb"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

const (
	expectedChainID          = "zerone-2"
	denom                    = "uzrn"
	totalSupply              = "13555000000"
	validatorBalance         = "11333000000"
	validatorSelfBond        = "11111000000"
	opsBalance               = "2222000000"
	protocolAdmissionBarrier = "222222222000001"
	customStakeMinimum       = "111000000"
	networkManifestSchema    = "zerone-2-network-manifest-v2"
	drillReleaseTag          = "DRILL-NOT-A-RELEASE"
	drillSignerFingerprint   = "DRILL-NOT-A-SIGNER"
	drillNodeID              = "2222222222222222222222222222222222222222"
)

var requiredArtifactFiles = map[string]struct{}{
	"GENESIS-MANIFEST.md":   {},
	"genesis.json":          {},
	"genesis.sha256":        {},
	"network-manifest.json": {},
}

var signingEncodingConfig = zeroneapp.MakeEncodingConfig()

var prohibitedArtifactNames = []string{
	"priv_validator_key.json",
	"node_key.json",
	"mnemonic",
	"seed-phrase",
	"seed_phrase",
	"recovery-phrase",
	"recovery_phrase",
	"private-key",
	"private_key",
	"privkey",
}

var prohibitedContent = [][]byte{
	[]byte("-----BEGIN PRIVATE KEY-----"),
	[]byte("-----BEGIN EC PRIVATE KEY-----"),
	[]byte("-----BEGIN RSA PRIVATE KEY-----"),
	[]byte("-----BEGIN OPENSSH PRIVATE KEY-----"),
	[]byte("AGE-SECRET-KEY-"),
	[]byte("tendermint/PrivKey"),
	[]byte("cometbft/PrivKey"),
}

type issue struct {
	Path    string
	Message string
}

type result struct {
	Issues                   []issue
	GenesisTime              string
	ValidatorAddress         string
	ValidatorOperatorAddress string
	ValidatorConsensusKey    publicKey
	ValidatorNodeID          string
	OpsAddress               string
	EmbeddedGentxSHA256      string
}

type auditor struct {
	root                     map[string]any
	issues                   []issue
	genesisTime              string
	validatorAddress         string
	validatorOperatorAddress string
	validatorConsensusKey    publicKey
	validatorNodeID          string
	opsAddress               string
	validatorAddrData        []byte
	embeddedGentxSHA256      string
}

type publicKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

type networkManifest struct {
	Schema        string `json:"schema"`
	Mode          string `json:"mode"`
	ChainID       string `json:"chain_id"`
	GenesisTime   string `json:"genesis_time"`
	GenesisSHA256 string `json:"genesis_sha256"`
	Release       struct {
		SourceCommit         string `json:"source_commit"`
		Tag                  string `json:"tag"`
		TagSignerFingerprint string `json:"tag_signer_fingerprint"`
		BinarySHA256         string `json:"binary_sha256"`
		BinaryVersion        string `json:"binary_version"`
		BinaryGOOS           string `json:"binary_goos"`
		BinaryGOARCH         string `json:"binary_goarch"`
	} `json:"release"`
	TrustModel struct {
		GenesisValidators       int    `json:"genesis_validators"`
		ByzantineFaultTolerance int    `json:"byzantine_fault_tolerance"`
		Disclosure              string `json:"disclosure"`
	} `json:"trust_model"`
	SupplyUzrn string `json:"supply_uzrn"`
	Validator  struct {
		AccountAddress  string    `json:"account_address"`
		OperatorAddress string    `json:"operator_address"`
		ConsensusPubkey publicKey `json:"consensus_pubkey"`
		NodeID          string    `json:"node_id"`
		SelfBondUzrn    string    `json:"self_bond_uzrn"`
		GentxSHA256     string    `json:"gentx_sha256"`
	} `json:"validator"`
	Operations struct {
		AccountAddress string `json:"account_address"`
	} `json:"operations"`
	Activations struct {
		VoteExtensions           string `json:"vote_extensions"`
		PoT                      string `json:"pot"`
		IBC                      string `json:"ibc"`
		SubstrateBridge          string `json:"substrate_bridge"`
		Claiming                 string `json:"claiming"`
		MessageScheduleAdmission string `json:"message_schedule_admission"`
	} `json:"activations"`
}

func main() {
	artifactDir := flag.String("artifact-dir", "", "directory containing genesis.json and only public launch artifacts")
	requiredMode := flag.String("required-mode", "", "required ceremony mode: drill or real")
	flag.Parse()

	if *artifactDir == "" || (*requiredMode != "drill" && *requiredMode != "real") || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/zerone2-artifact-audit --artifact-dir PATH --required-mode drill|real")
		os.Exit(2)
	}

	r := auditArtifactDir(*artifactDir, *requiredMode)
	if len(r.Issues) != 0 {
		fmt.Fprintf(os.Stderr, "FAIL zerone-2 artifact audit: %d violation(s)\n", len(r.Issues))
		for _, problem := range r.Issues {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", problem.Path, problem.Message)
		}
		os.Exit(1)
	}

	fmt.Println("PASS zerone-2 artifact audit")
	fmt.Printf("  chain-id:  %s\n", expectedChainID)
	fmt.Printf("  supply:    %s%s\n", totalSupply, denom)
	fmt.Printf("  validator: %s\n", r.ValidatorAddress)
	fmt.Printf("  ops:       %s\n", r.OpsAddress)
	fmt.Println("  profile:   one locked SDK validator; protocol-dark; vote extensions off; public artifacts only")
}

func auditArtifactDir(dir, requiredMode string) result {
	if requiredMode != "drill" && requiredMode != "real" {
		return result{Issues: []issue{{Path: "required-mode", Message: "must be exactly drill or real"}}}
	}
	clean, err := filepath.Abs(dir)
	if err != nil {
		return result{Issues: []issue{{Path: "artifact-dir", Message: err.Error()}}}
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return result{Issues: []issue{{Path: "artifact-dir", Message: err.Error()}}}
	}
	if !info.IsDir() {
		return result{Issues: []issue{{Path: "artifact-dir", Message: "must be a directory"}}}
	}

	var filesystemIssues []issue
	err = filepath.WalkDir(clean, func(path string, entry os.DirEntry, walkErr error) error {
		rel, relErr := filepath.Rel(clean, path)
		if relErr != nil {
			filesystemIssues = append(filesystemIssues, issue{Path: path, Message: relErr.Error()})
			return nil
		}
		if walkErr != nil {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: walkErr.Error()})
			return nil
		}
		if rel == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "symlinks are forbidden in release artifacts"})
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "subdirectories are forbidden; the release artifact set has exactly four files"})
			return filepath.SkipDir
		}
		if !entry.Type().IsRegular() {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "only regular public artifact files are allowed"})
			return nil
		}
		if artifactNameLooksPrivate(entry.Name()) {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "private key/secret filename is forbidden"})
		}
		if _, allowed := requiredArtifactFiles[rel]; !allowed {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "unexpected file; the release artifact set has exactly four allowlisted files"})
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: readErr.Error()})
			return nil
		}
		for _, marker := range prohibitedContent {
			if bytes.Contains(contents, marker) {
				filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: fmt.Sprintf("contains prohibited private-key marker %q", marker)})
			}
		}
		if containsValidMnemonic(contents) {
			filesystemIssues = append(filesystemIssues, issue{Path: rel, Message: "contains a valid BIP-39 mnemonic"})
		}
		return nil
	})
	if err != nil {
		filesystemIssues = append(filesystemIssues, issue{Path: "artifact-dir", Message: err.Error()})
	}

	genesis, genesisErr := os.ReadFile(filepath.Join(clean, "genesis.json"))
	checksum, checksumErr := os.ReadFile(filepath.Join(clean, "genesis.sha256"))
	manifest, manifestErr := os.ReadFile(filepath.Join(clean, "network-manifest.json"))
	_, humanManifestErr := os.ReadFile(filepath.Join(clean, "GENESIS-MANIFEST.md"))
	if genesisErr != nil {
		filesystemIssues = append(filesystemIssues, issue{Path: "genesis.json", Message: "required artifact missing or unreadable: " + genesisErr.Error()})
	}
	if checksumErr != nil {
		filesystemIssues = append(filesystemIssues, issue{Path: "genesis.sha256", Message: "required artifact missing or unreadable: " + checksumErr.Error()})
	}
	if manifestErr != nil {
		filesystemIssues = append(filesystemIssues, issue{Path: "network-manifest.json", Message: "required artifact missing or unreadable: " + manifestErr.Error()})
	}
	if humanManifestErr != nil {
		filesystemIssues = append(filesystemIssues, issue{Path: "GENESIS-MANIFEST.md", Message: "required artifact missing or unreadable: " + humanManifestErr.Error()})
	}

	r := result{}
	if genesisErr == nil {
		r = auditGenesis(genesis)
		digest := sha256.Sum256(genesis)
		genesisHash := hex.EncodeToString(digest[:])
		if checksumErr == nil {
			r.Issues = append(r.Issues, auditGenesisChecksum(checksum, genesisHash)...)
		}
		if manifestErr == nil {
			r.Issues = append(r.Issues, auditNetworkManifest(manifest, r, genesisHash, requiredMode)...)
		}
	} else if manifestErr == nil {
		// Still report strict JSON errors even when the source genesis needed for
		// semantic comparison is absent.
		_, parseIssues := decodeNetworkManifest(manifest)
		r.Issues = append(r.Issues, parseIssues...)
	}
	r.Issues = append(r.Issues, filesystemIssues...)
	sortIssues(r.Issues)
	return r
}

func auditGenesisChecksum(contents []byte, genesisHash string) []issue {
	expected := genesisHash + "  genesis.json\n"
	if !bytes.Equal(contents, []byte(expected)) {
		return []issue{{
			Path:    "genesis.sha256",
			Message: "must contain exactly the lowercase genesis SHA-256, two spaces, genesis.json, and one newline",
		}}
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delim {
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
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("malformed JSON object")
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("malformed JSON array")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}

func decodeNetworkManifest(contents []byte) (networkManifest, []issue) {
	var manifest networkManifest
	if err := rejectDuplicateJSONKeys(contents); err != nil {
		return networkManifest{}, []issue{{Path: "network-manifest.json", Message: "ambiguous JSON: " + err.Error()}}
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return networkManifest{}, []issue{{Path: "network-manifest.json", Message: "invalid strict JSON: " + err.Error()}}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return networkManifest{}, []issue{{Path: "network-manifest.json", Message: "contains more than one JSON value"}}
		}
		return networkManifest{}, []issue{{Path: "network-manifest.json", Message: "invalid trailing data: " + err.Error()}}
	}
	return manifest, nil
}

func auditNetworkManifest(contents []byte, genesis result, genesisHash, requiredMode string) []issue {
	manifest, issues := decodeNetworkManifest(contents)
	if len(issues) != 0 {
		return issues
	}
	addMismatch := func(path string, got, expected any) {
		issues = append(issues, issue{
			Path:    "network-manifest.json." + path,
			Message: fmt.Sprintf("got %v, expected %v", got, expected),
		})
	}

	if manifest.Schema != networkManifestSchema {
		addMismatch("schema", manifest.Schema, networkManifestSchema)
	}
	if manifest.Mode != requiredMode {
		addMismatch("mode", manifest.Mode, requiredMode)
	}
	if manifest.ChainID != expectedChainID {
		addMismatch("chain_id", manifest.ChainID, expectedChainID)
	}
	if manifest.GenesisTime != genesis.GenesisTime {
		addMismatch("genesis_time", manifest.GenesisTime, genesis.GenesisTime)
	}
	if manifest.GenesisSHA256 != genesisHash {
		addMismatch("genesis_sha256", manifest.GenesisSHA256, genesisHash)
	}
	if !isLowerHexLength(manifest.Release.SourceCommit, 40) {
		addMismatch("release.source_commit", manifest.Release.SourceCommit, "40 lowercase hexadecimal characters")
	}
	if manifest.Release.Tag == "" {
		addMismatch("release.tag", manifest.Release.Tag, "a non-empty release tag")
	}
	if requiredMode == "real" {
		if !isFingerprint(manifest.Release.TagSignerFingerprint) {
			addMismatch("release.tag_signer_fingerprint", manifest.Release.TagSignerFingerprint, "40 or 64 lowercase hexadecimal characters")
		}
		if manifest.Release.Tag == drillReleaseTag {
			addMismatch("release.tag", manifest.Release.Tag, "a signed production release tag")
		}
		if manifest.Validator.NodeID == drillNodeID {
			addMismatch("validator.node_id", manifest.Validator.NodeID, "a non-drill production node ID")
		}
		if manifest.Validator.NodeID == strings.Repeat("0", 40) {
			addMismatch("validator.node_id", manifest.Validator.NodeID, "a non-zero production node ID")
		}
	} else {
		if manifest.Release.Tag != drillReleaseTag {
			addMismatch("release.tag", manifest.Release.Tag, drillReleaseTag)
		}
		if manifest.Release.TagSignerFingerprint != drillSignerFingerprint {
			addMismatch("release.tag_signer_fingerprint", manifest.Release.TagSignerFingerprint, drillSignerFingerprint)
		}
	}
	if !isLowerHexLength(manifest.Release.BinarySHA256, 64) {
		addMismatch("release.binary_sha256", manifest.Release.BinarySHA256, "64 lowercase hexadecimal characters")
	}
	if manifest.Release.BinaryVersion == "" {
		addMismatch("release.binary_version", manifest.Release.BinaryVersion, "a non-empty binary version")
	}
	if requiredMode == "real" {
		if manifest.Release.BinaryGOOS != "linux" {
			addMismatch("release.binary_goos", manifest.Release.BinaryGOOS, "linux")
		}
		if manifest.Release.BinaryGOARCH != "amd64" && manifest.Release.BinaryGOARCH != "arm64" {
			addMismatch("release.binary_goarch", manifest.Release.BinaryGOARCH, "amd64 or arm64")
		}
	} else {
		if manifest.Release.BinaryGOOS == "" {
			addMismatch("release.binary_goos", manifest.Release.BinaryGOOS, "a non-empty drill binary GOOS")
		}
		if manifest.Release.BinaryGOARCH == "" {
			addMismatch("release.binary_goarch", manifest.Release.BinaryGOARCH, "a non-empty drill binary GOARCH")
		}
	}
	if manifest.TrustModel.GenesisValidators != 1 {
		addMismatch("trust_model.genesis_validators", manifest.TrustModel.GenesisValidators, 1)
	}
	if manifest.TrustModel.ByzantineFaultTolerance != 0 {
		addMismatch("trust_model.byzantine_fault_tolerance", manifest.TrustModel.ByzantineFaultTolerance, 0)
	}
	if strings.TrimSpace(manifest.TrustModel.Disclosure) == "" {
		addMismatch("trust_model.disclosure", manifest.TrustModel.Disclosure, "a non-empty custody disclosure")
	}
	if manifest.SupplyUzrn != totalSupply {
		addMismatch("supply_uzrn", manifest.SupplyUzrn, totalSupply)
	}
	if manifest.Validator.AccountAddress != genesis.ValidatorAddress {
		addMismatch("validator.account_address", manifest.Validator.AccountAddress, genesis.ValidatorAddress)
	}
	if manifest.Validator.OperatorAddress != genesis.ValidatorOperatorAddress {
		addMismatch("validator.operator_address", manifest.Validator.OperatorAddress, genesis.ValidatorOperatorAddress)
	}
	if manifest.Validator.ConsensusPubkey != genesis.ValidatorConsensusKey {
		addMismatch("validator.consensus_pubkey", manifest.Validator.ConsensusPubkey, genesis.ValidatorConsensusKey)
	}
	if manifest.Validator.NodeID != genesis.ValidatorNodeID {
		addMismatch("validator.node_id", manifest.Validator.NodeID, genesis.ValidatorNodeID)
	}
	if manifest.Validator.SelfBondUzrn != validatorSelfBond {
		addMismatch("validator.self_bond_uzrn", manifest.Validator.SelfBondUzrn, validatorSelfBond)
	}
	if manifest.Validator.GentxSHA256 != genesis.EmbeddedGentxSHA256 {
		addMismatch("validator.gentx_sha256", manifest.Validator.GentxSHA256, genesis.EmbeddedGentxSHA256)
	}
	if manifest.Operations.AccountAddress != genesis.OpsAddress {
		addMismatch("operations.account_address", manifest.Operations.AccountAddress, genesis.OpsAddress)
	}
	for _, declaration := range []struct {
		path     string
		got      string
		expected string
	}{
		{path: "activations.vote_extensions", got: manifest.Activations.VoteExtensions, expected: "disabled"},
		{path: "activations.pot", got: manifest.Activations.PoT, expected: "not live"},
		{path: "activations.ibc", got: manifest.Activations.IBC, expected: "external-disabled; localhost-only"},
		{path: "activations.substrate_bridge", got: manifest.Activations.SubstrateBridge, expected: "disabled"},
		{path: "activations.claiming", got: manifest.Activations.Claiming, expected: "disabled"},
		{path: "activations.message_schedule_admission", got: manifest.Activations.MessageScheduleAdmission, expected: "disabled"},
	} {
		if declaration.got != declaration.expected {
			addMismatch(declaration.path, declaration.got, declaration.expected)
		}
	}
	return issues
}

func isLowerHexLength(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

func isFingerprint(value string) bool {
	return isLowerHexLength(value, 40) || isLowerHexLength(value, 64)
}

func containsValidMnemonic(contents []byte) bool {
	// Treat every non-ASCII letter as a separator. This catches mnemonics hidden
	// in JSON arrays, quoted prose, comma-separated text, or punctuation.
	words := strings.FieldsFunc(strings.ToLower(string(contents)), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	for _, length := range []int{12, 15, 18, 21, 24} {
		for start := 0; start+length <= len(words); start++ {
			candidate := strings.Join(words[start:start+length], " ")
			if bip39.IsMnemonicValid(candidate) {
				return true
			}
		}
	}
	return false
}

func artifactNameLooksPrivate(name string) bool {
	normalized := strings.ToLower(name)
	for _, marker := range prohibitedArtifactNames {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if normalized == "keyring" || strings.HasPrefix(normalized, "keyring-") || strings.HasSuffix(normalized, ".mnemonic") {
		return true
	}
	if strings.HasSuffix(normalized, ".pem") || (strings.HasSuffix(normalized, ".key") && !strings.Contains(normalized, "public")) {
		return true
	}
	return false
}

func auditGenesis(data []byte) result {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return result{Issues: []issue{{Path: "genesis.json", Message: "ambiguous JSON: " + err.Error()}}}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root map[string]any
	if err := dec.Decode(&root); err != nil {
		return result{Issues: []issue{{Path: "genesis.json", Message: "invalid JSON: " + err.Error()}}}
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return result{Issues: []issue{{Path: "genesis.json", Message: "contains more than one JSON value"}}}
		}
		return result{Issues: []issue{{Path: "genesis.json", Message: "invalid trailing data: " + err.Error()}}}
	}

	a := &auditor{root: root}
	a.scanJSONSecrets(root, "$")
	a.auditMetadata()
	a.auditBaseProtocolProfile()
	a.auditAllocations()
	a.auditGentx()
	a.auditProtocolDark()
	sortIssues(a.issues)
	return result{
		Issues:                   a.issues,
		GenesisTime:              a.genesisTime,
		ValidatorAddress:         a.validatorAddress,
		ValidatorOperatorAddress: a.validatorOperatorAddress,
		ValidatorConsensusKey:    a.validatorConsensusKey,
		ValidatorNodeID:          a.validatorNodeID,
		OpsAddress:               a.opsAddress,
		EmbeddedGentxSHA256:      a.embeddedGentxSHA256,
	}
}

func (a *auditor) auditMetadata() {
	a.requireString("chain_id", expectedChainID)
	a.requireInteger("initial_height", "1")
	genesisTime, ok := a.get("genesis_time")
	if !ok {
		a.add("genesis_time", "missing required RFC3339 timestamp")
	} else if value, ok := genesisTime.(string); !ok {
		a.add("genesis_time", "must be an RFC3339 string")
	} else if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		a.add("genesis_time", "invalid RFC3339 timestamp: %v", err)
	} else {
		a.genesisTime = value
	}
	a.requireInteger("consensus.params.block.max_bytes", "4194304")
	a.requireInteger("consensus.params.block.max_gas", "33333333")
	a.requireInteger("consensus.params.evidence.max_age_num_blocks", "100000")
	a.requireInteger("consensus.params.evidence.max_age_duration", "172800000000000")
	a.requireInteger("consensus.params.evidence.max_bytes", "1048576")
	a.requireStringArray("consensus.params.validator.pub_key_types", []string{"ed25519"})
	a.requireInteger("consensus.params.version.app", "0")
	a.requireInteger("consensus.params.abci.vote_extensions_enable_height", "0")
}

func (a *auditor) auditBaseProtocolProfile() {
	for _, requirement := range []struct {
		path     string
		expected string
	}{
		{path: "app_state.staking.params.unbonding_time", expected: "1814400s"},
		{path: "app_state.staking.params.bond_denom", expected: denom},
		{path: "app_state.staking.params.min_commission_rate", expected: "0.050000000000000000"},
		{path: "app_state.gov.params.max_deposit_period", expected: "259200s"},
		{path: "app_state.gov.params.voting_period", expected: "259200s"},
		{path: "app_state.gov.params.expedited_voting_period", expected: "86400s"},
		{path: "app_state.gov.params.min_initial_deposit_ratio", expected: "0.250000000000000000"},
		{path: "app_state.gov.params.quorum", expected: "0.334000000000000000"},
		{path: "app_state.gov.params.threshold", expected: "0.500000000000000000"},
		{path: "app_state.gov.params.veto_threshold", expected: "0.334000000000000000"},
		{path: "app_state.gov.params.proposal_cancel_ratio", expected: "0.500000000000000000"},
		{path: "app_state.gov.params.proposal_cancel_dest", expected: ""},
		{path: "app_state.gov.params.expedited_threshold", expected: "0.667000000000000000"},
		{path: "app_state.gov.params.min_deposit_ratio", expected: "0.010000000000000000"},
	} {
		a.requireString(requirement.path, requirement.expected)
	}
	for _, requirement := range []struct {
		path     string
		expected string
	}{
		{path: "app_state.staking.params.max_validators", expected: "33"},
		{path: "app_state.staking.params.max_entries", expected: "7"},
		{path: "app_state.staking.params.historical_entries", expected: "10000"},
		{path: "app_state.knowledge.params.min_verifiers", expected: "3"},
		{path: "app_state.knowledge.params.min_headcount_agreement", expected: "3"},
	} {
		a.requireInteger(requirement.path, requirement.expected)
	}
	a.requireCoinArray("app_state.gov.params.min_deposit", denom, "100000000")
	a.requireCoinArray("app_state.gov.params.expedited_min_deposit", denom, "300000000")
	a.requireBool("app_state.gov.params.burn_vote_quorum", false)
	a.requireBool("app_state.gov.params.burn_proposal_deposit_prevote", false)
	a.requireBool("app_state.gov.params.burn_vote_veto", true)
}

func (a *auditor) auditAllocations() {
	supply, ok := a.slice("app_state.bank.supply")
	if !ok {
		return
	}
	if len(supply) != 1 {
		a.add("app_state.bank.supply", "must contain exactly one denomination, got %d", len(supply))
	} else {
		a.requireCoin("app_state.bank.supply[0]", supply[0], denom, totalSupply)
	}

	balances, ok := a.slice("app_state.bank.balances")
	if !ok {
		return
	}
	if len(balances) != 2 {
		a.add("app_state.bank.balances", "must contain exactly validator and ops allocations, got %d", len(balances))
	}
	seenAddresses := make(map[string]bool)
	amountOwners := make(map[string]string)
	total := new(big.Int)
	for i, raw := range balances {
		path := fmt.Sprintf("app_state.bank.balances[%d]", i)
		balance, ok := raw.(map[string]any)
		if !ok {
			a.add(path, "must be an object")
			continue
		}
		a.requireExactObjectKeys(path, balance, "address", "coins")
		address, ok := balance["address"].(string)
		if !ok || address == "" {
			a.add(path+".address", "must be a non-empty string")
			continue
		}
		if seenAddresses[address] {
			a.add(path+".address", "duplicate balance owner %s", address)
		}
		seenAddresses[address] = true
		coins, ok := balance["coins"].([]any)
		if !ok || len(coins) != 1 {
			a.add(path+".coins", "must contain exactly one uzrn coin")
			continue
		}
		coin, ok := coins[0].(map[string]any)
		if !ok {
			a.add(path+".coins[0]", "must be an object")
			continue
		}
		coinDenom, _ := coin["denom"].(string)
		amount, amountOK := integerString(coin["amount"])
		if coinDenom != denom {
			a.add(path+".coins[0].denom", "must equal %q", denom)
		}
		if !amountOK {
			a.add(path+".coins[0].amount", "must be a base-10 integer")
			continue
		}
		if prior, duplicate := amountOwners[amount]; duplicate {
			a.add(path+".coins[0].amount", "duplicates allocation amount owned by %s", prior)
		}
		amountOwners[amount] = address
		parsed, _ := new(big.Int).SetString(amount, 10)
		total.Add(total, parsed)
	}
	if total.String() != totalSupply {
		a.add("app_state.bank.balances", "sum is %s%s, expected %s%s", total.String(), denom, totalSupply, denom)
	}
	validatorFromBalance, hasValidator := amountOwners[validatorBalance]
	opsFromBalance, hasOps := amountOwners[opsBalance]
	if !hasValidator {
		a.add("app_state.bank.balances", "missing validator allocation %s%s", validatorBalance, denom)
	}
	if !hasOps {
		a.add("app_state.bank.balances", "missing ops allocation %s%s", opsBalance, denom)
	}

	accounts, ok := a.slice("app_state.auth.accounts")
	if !ok {
		return
	}
	if len(accounts) != 2 {
		a.add("app_state.auth.accounts", "must contain exactly one PermanentLocked validator and one ops BaseAccount, got %d", len(accounts))
	}
	var lockedAddress, baseAddress string
	lockedCount, baseCount := 0, 0
	for i, raw := range accounts {
		path := fmt.Sprintf("app_state.auth.accounts[%d]", i)
		account, ok := raw.(map[string]any)
		if !ok {
			a.add(path, "must be an object")
			continue
		}
		accountType, _ := account["@type"].(string)
		switch accountType {
		case "/cosmos.vesting.v1beta1.PermanentLockedAccount":
			lockedCount++
			a.requireExactObjectKeys(path, account, "@type", "base_vesting_account")
			baseVesting, ok := account["base_vesting_account"].(map[string]any)
			if !ok {
				a.add(path+".base_vesting_account", "must be an object")
				continue
			}
			a.requireExactObjectKeys(path+".base_vesting_account", baseVesting,
				"base_account", "original_vesting", "delegated_free", "delegated_vesting", "end_time")
			base, ok := baseVesting["base_account"].(map[string]any)
			if !ok {
				a.add(path+".base_vesting_account.base_account", "must be an object")
				continue
			}
			a.requireExactObjectKeys(path+".base_vesting_account.base_account", base,
				"address", "pub_key", "account_number", "sequence")
			lockedAddress, _ = base["address"].(string)
			a.requireAccountGenesisState(path+".base_vesting_account.base_account", base, "0")
			if base["pub_key"] != nil {
				a.add(path+".base_vesting_account.base_account.pub_key", "must be explicit null at genesis")
			}
			original, ok := baseVesting["original_vesting"].([]any)
			if !ok || len(original) != 1 {
				a.add(path+".base_vesting_account.original_vesting", "must contain exactly the locked self-bond")
			} else {
				a.requireCoin(path+".base_vesting_account.original_vesting[0]", original[0], denom, validatorSelfBond)
			}
			a.requireIntegerValue(path+".base_vesting_account.end_time", baseVesting["end_time"], "0")
			a.requireEmptyArrayValue(path+".base_vesting_account.delegated_free", baseVesting["delegated_free"], true)
			a.requireEmptyArrayValue(path+".base_vesting_account.delegated_vesting", baseVesting["delegated_vesting"], true)
		case "/cosmos.auth.v1beta1.BaseAccount":
			baseCount++
			a.requireExactObjectKeys(path, account, "@type", "address", "pub_key", "account_number", "sequence")
			baseAddress, _ = account["address"].(string)
			a.requireAccountGenesisState(path, account, "1")
			if account["pub_key"] != nil {
				a.add(path+".pub_key", "must be explicit null at genesis")
			}
		default:
			a.add(path+".@type", "unexpected user account type %q", accountType)
		}
	}
	if lockedCount != 1 {
		a.add("app_state.auth.accounts", "expected exactly one PermanentLockedAccount, got %d", lockedCount)
	}
	if baseCount != 1 {
		a.add("app_state.auth.accounts", "expected exactly one ops BaseAccount, got %d", baseCount)
	}
	if hasValidator && lockedAddress != validatorFromBalance {
		a.add("app_state.auth.accounts", "PermanentLockedAccount %q does not own validator allocation %q", lockedAddress, validatorFromBalance)
	}
	if hasOps && baseAddress != opsFromBalance {
		a.add("app_state.auth.accounts", "ops BaseAccount %q does not own ops allocation %q", baseAddress, opsFromBalance)
	}
	if lockedAddress != "" {
		_, data, err := decodeBech32(lockedAddress, "zrn")
		if err != nil || len(data) != 20 {
			a.add("app_state.auth.accounts", "validator account address is invalid zrn bech32: %v", err)
		} else {
			a.validatorAddrData = data
			a.validatorAddress = lockedAddress
		}
	}
	if baseAddress != "" {
		_, data, err := decodeBech32(baseAddress, "zrn")
		if err != nil || len(data) != 20 {
			a.add("app_state.auth.accounts", "ops account address is invalid zrn bech32: %v", err)
		} else {
			a.opsAddress = baseAddress
		}
	}
	if lockedAddress != "" && lockedAddress == baseAddress {
		a.add("app_state.auth.accounts", "validator and ops must be distinct addresses")
	}
}

func (a *auditor) auditGentx() {
	txs, ok := a.slice("app_state.genutil.gen_txs")
	if !ok {
		return
	}
	if len(txs) != 1 {
		a.add("app_state.genutil.gen_txs", "must contain exactly one SDK gentx, got %d", len(txs))
		return
	}
	tx, ok := txs[0].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0]", "must be an object")
		return
	}
	canonicalGentx, err := json.Marshal(tx)
	if err != nil {
		a.add("app_state.genutil.gen_txs[0]", "cannot canonicalize embedded gentx: %v", err)
	} else {
		digest := sha256.Sum256(canonicalGentx)
		a.embeddedGentxSHA256 = hex.EncodeToString(digest[:])
	}
	a.requireExactObjectKeys("app_state.genutil.gen_txs[0]", tx, "body", "auth_info", "signatures")
	body, ok := tx["body"].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0].body", "must be an object")
		return
	}
	a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body", body,
		"messages", "memo", "timeout_height", "extension_options", "non_critical_extension_options",
		"unordered", "timeout_timestamp")
	memo, ok := body["memo"].(string)
	if !ok || memo == "" {
		a.add("app_state.genutil.gen_txs[0].body.memo", "must declare the validator node ID")
	} else {
		parts := strings.Split(memo, "@")
		if len(parts) != 2 || !isLowerHexLength(parts[0], 40) || parts[1] == "" || strings.ContainsAny(parts[1], " \t\r\n") {
			a.add("app_state.genutil.gen_txs[0].body.memo", "must be NODE_ID@P2P_ENDPOINT with one 40-character lowercase node ID and no whitespace")
		} else {
			a.validatorNodeID = parts[0]
		}
	}
	a.requireIntegerValue("app_state.genutil.gen_txs[0].body.timeout_height", body["timeout_height"], "0")
	a.requireEmptyArrayValue("app_state.genutil.gen_txs[0].body.extension_options", body["extension_options"], true)
	a.requireEmptyArrayValue("app_state.genutil.gen_txs[0].body.non_critical_extension_options", body["non_critical_extension_options"], true)
	if unordered, ok := body["unordered"].(bool); !ok || unordered {
		a.add("app_state.genutil.gen_txs[0].body.unordered", "must be explicit false")
	}
	if timeout, exists := body["timeout_timestamp"]; !exists || timeout != nil {
		a.add("app_state.genutil.gen_txs[0].body.timeout_timestamp", "must be explicit null")
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 1 {
		a.add("app_state.genutil.gen_txs[0].body.messages", "must contain exactly one MsgCreateValidator")
		return
	}
	msg, ok := messages[0].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0].body.messages[0]", "must be an object")
		return
	}
	a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body.messages[0]", msg,
		"@type", "description", "commission", "min_self_delegation", "delegator_address", "validator_address", "pubkey", "value")
	if got, _ := msg["@type"].(string); got != "/cosmos.staking.v1beta1.MsgCreateValidator" {
		a.add("app_state.genutil.gen_txs[0].body.messages[0].@type", "must be the SDK MsgCreateValidator, got %q", got)
	}
	value, ok := msg["value"].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0].body.messages[0].value", "must be an object")
	} else {
		a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body.messages[0].value", value, "denom", "amount")
		a.requireCoin("app_state.genutil.gen_txs[0].body.messages[0].value", value, denom, validatorSelfBond)
	}
	description := a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body.messages[0].description", msg["description"],
		"moniker", "identity", "website", "security_contact", "details")
	if description != nil {
		for key, expected := range map[string]string{
			"moniker": "zerone-2-custodian", "identity": "", "website": "", "security_contact": "", "details": "",
		} {
			if got, _ := description[key].(string); got != expected {
				a.add("app_state.genutil.gen_txs[0].body.messages[0].description."+key, "got %q, expected %q", got, expected)
			}
		}
	}
	commission := a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body.messages[0].commission", msg["commission"],
		"rate", "max_rate", "max_change_rate")
	if commission != nil {
		for key, expected := range map[string]string{
			"rate": "0.050000000000000000", "max_rate": "0.200000000000000000", "max_change_rate": "0.010000000000000000",
		} {
			if got, _ := commission[key].(string); got != expected {
				a.add("app_state.genutil.gen_txs[0].body.messages[0].commission."+key, "got %q, expected %q", got, expected)
			}
		}
	}
	a.requireIntegerValue("app_state.genutil.gen_txs[0].body.messages[0].min_self_delegation", msg["min_self_delegation"], "1")
	valoper, _ := msg["validator_address"].(string)
	_, valoperData, err := decodeBech32(valoper, "zrnvaloper")
	if err != nil || len(valoperData) != 20 {
		a.add("app_state.genutil.gen_txs[0].body.messages[0].validator_address", "invalid zrnvaloper bech32: %v", err)
	} else {
		a.validatorOperatorAddress = valoper
		if len(a.validatorAddrData) == 20 && !bytes.Equal(valoperData, a.validatorAddrData) {
			a.add("app_state.genutil.gen_txs[0].body.messages[0].validator_address", "does not encode the PermanentLocked validator account")
		}
	}
	if delegator, _ := msg["delegator_address"].(string); delegator != "" {
		a.add("app_state.genutil.gen_txs[0].body.messages[0].delegator_address", "must be empty in the exact SDK gentx envelope")
	}
	pubkey, ok := msg["pubkey"].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0].body.messages[0].pubkey", "must be an Ed25519 public key object")
	} else {
		a.requireExactObjectKeys("app_state.genutil.gen_txs[0].body.messages[0].pubkey", pubkey, "@type", "key")
		keyBytes := a.requirePublicKey("app_state.genutil.gen_txs[0].body.messages[0].pubkey", pubkey, "/cosmos.crypto.ed25519.PubKey", 32)
		pubkeyType, typeOK := pubkey["@type"].(string)
		encodedKey, keyOK := pubkey["key"].(string)
		if len(keyBytes) == 32 && typeOK && keyOK && pubkeyType == "/cosmos.crypto.ed25519.PubKey" {
			a.validatorConsensusKey = publicKey{
				Type: pubkeyType,
				Key:  encodedKey,
			}
		}
	}

	authInfo, ok := tx["auth_info"].(map[string]any)
	if !ok {
		a.add("app_state.genutil.gen_txs[0].auth_info", "must be an object")
		return
	}
	a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info", authInfo, "signer_infos", "fee", "tip")
	signers, ok := authInfo["signer_infos"].([]any)
	if !ok || len(signers) != 1 {
		a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos", "must contain exactly one signer")
	} else if signer, ok := signers[0].(map[string]any); !ok {
		a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos[0]", "must be an object")
	} else {
		a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info.signer_infos[0]", signer, "public_key", "mode_info", "sequence")
		a.requireIntegerValue("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].sequence", signer["sequence"], "0")
		modeInfo := a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].mode_info", signer["mode_info"], "single")
		if modeInfo != nil {
			single := a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].mode_info.single", modeInfo["single"], "mode")
			if single != nil {
				if mode, _ := single["mode"].(string); mode != "SIGN_MODE_DIRECT" {
					a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].mode_info.single.mode", "got %q, expected SIGN_MODE_DIRECT", mode)
				}
			}
		}
		if publicKey, ok := signer["public_key"].(map[string]any); !ok {
			a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].public_key", "must be a secp256k1 public key object")
		} else {
			a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].public_key", publicKey, "@type", "key")
			if keyBytes := a.requirePublicKey("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].public_key", publicKey, "/cosmos.crypto.secp256k1.PubKey", 33); len(keyBytes) == 33 {
				if keyBytes[0] != 2 && keyBytes[0] != 3 {
					a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].public_key.key", "must be a compressed secp256k1 public key")
				} else if len(a.validatorAddrData) == 20 {
					hash := sha256.Sum256(keyBytes)
					ripe := ripemd160.New()
					_, _ = ripe.Write(hash[:])
					if !bytes.Equal(ripe.Sum(nil), a.validatorAddrData) {
						a.add("app_state.genutil.gen_txs[0].auth_info.signer_infos[0].public_key.key", "does not derive the PermanentLocked validator address")
					}
				}
			}
		}
	}
	fee := a.requireExactObjectKeys("app_state.genutil.gen_txs[0].auth_info.fee", authInfo["fee"], "amount", "gas_limit", "payer", "granter")
	if fee != nil {
		a.requireEmptyArrayValue("app_state.genutil.gen_txs[0].auth_info.fee.amount", fee["amount"], true)
		a.requireIntegerValue("app_state.genutil.gen_txs[0].auth_info.fee.gas_limit", fee["gas_limit"], "200000")
		if payer, _ := fee["payer"].(string); payer != "" {
			a.add("app_state.genutil.gen_txs[0].auth_info.fee.payer", "must be empty")
		}
		if granter, _ := fee["granter"].(string); granter != "" {
			a.add("app_state.genutil.gen_txs[0].auth_info.fee.granter", "must be empty")
		}
	}
	if tip, exists := authInfo["tip"]; !exists || tip != nil {
		a.add("app_state.genutil.gen_txs[0].auth_info.tip", "must be explicit null")
	}

	signatures, ok := tx["signatures"].([]any)
	if !ok || len(signatures) != 1 {
		a.add("app_state.genutil.gen_txs[0].signatures", "must contain exactly one signature")
	} else {
		signature, ok := signatures[0].(string)
		if !ok {
			a.add("app_state.genutil.gen_txs[0].signatures[0]", "must be base64")
		} else if decoded, err := base64.StdEncoding.DecodeString(signature); err != nil || len(decoded) != 64 || allZero(decoded) {
			a.add("app_state.genutil.gen_txs[0].signatures[0]", "must be a non-zero 64-byte base64 signature")
		}
	}
	if canonicalGentx != nil && a.validatorAddress != "" {
		if err := verifyGentxSignature(canonicalGentx, a.validatorAddress); err != nil {
			a.add("app_state.genutil.gen_txs[0].signatures[0]", "cryptographic verification failed: %v", err)
		}
	}
}

func verifyGentxSignature(txJSON []byte, expectedAddress string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("SDK signature verifier panicked on malformed gentx: %v", recovered)
		}
	}()
	tx, err := signingEncodingConfig.TxConfig.TxJSONDecoder()(txJSON)
	if err != nil {
		return fmt.Errorf("decode SDK transaction: %w", err)
	}
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return fmt.Errorf("decoded transaction does not implement SigVerifiableTx")
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return fmt.Errorf("derive transaction signers: %w", err)
	}
	if len(signers) != 1 {
		return fmt.Errorf("got %d transaction signers, expected 1", len(signers))
	}
	expected, err := sdk.AccAddressFromBech32(expectedAddress)
	if err != nil {
		return fmt.Errorf("decode expected signer: %w", err)
	}
	if !bytes.Equal(signers[0], expected) {
		return fmt.Errorf("transaction signer does not equal the locked validator account")
	}
	signatures, err := sigTx.GetSignaturesV2()
	if err != nil {
		return fmt.Errorf("decode transaction signatures: %w", err)
	}
	if len(signatures) != 1 {
		return fmt.Errorf("got %d signatures, expected 1", len(signatures))
	}
	signature := signatures[0]
	pubKey, ok := signature.PubKey.(*secp256k1.PubKey)
	if !ok || pubKey == nil {
		return fmt.Errorf("signer public key is not SDK secp256k1")
	}
	if !bytes.Equal(pubKey.Address(), expected) {
		return fmt.Errorf("signer public key does not derive the locked validator account")
	}
	if signature.Sequence != 0 {
		return fmt.Errorf("signature sequence is %d, expected 0", signature.Sequence)
	}
	single, ok := signature.Data.(*sdktextsigning.SingleSignatureData)
	if !ok || single == nil {
		return fmt.Errorf("signature is not a single signature")
	}
	if single.SignMode != sdktextsigning.SignMode_SIGN_MODE_DIRECT {
		return fmt.Errorf("signature mode is %s, expected SIGN_MODE_DIRECT", single.SignMode)
	}
	if len(single.Signature) != 64 {
		return fmt.Errorf("signature is %d bytes, expected 64", len(single.Signature))
	}
	packedPubKey, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		return fmt.Errorf("pack signer public key: %w", err)
	}
	adaptableTx, ok := tx.(authsigning.V2AdaptableTx)
	if !ok {
		return fmt.Errorf("decoded transaction does not implement V2AdaptableTx")
	}
	signerData := txsigning.SignerData{
		Address:       expectedAddress,
		ChainID:       expectedChainID,
		AccountNumber: 0,
		Sequence:      0,
		PubKey: &anypb.Any{
			TypeUrl: packedPubKey.TypeUrl,
			Value:   packedPubKey.Value,
		},
	}
	if err := authsigning.VerifySignature(
		context.Background(), pubKey, signerData, single,
		signingEncodingConfig.TxConfig.SignModeHandler(), adaptableTx.GetSigningTxData(),
	); err != nil {
		return err
	}
	return nil
}

func (a *auditor) auditProtocolDark() {
	// IBC-go v10 creates the 09-localhost client during InitGenesis even when
	// create_localhost is false. Whitelist that internal client only so the
	// first export remains valid without enabling any external client type.
	// ICS-20 transfer and both ICA roles remain explicitly disabled.
	a.requireStringArray("app_state.ibc.client_genesis.params.allowed_clients", []string{"09-localhost"})
	a.requireBool("app_state.ibc.client_genesis.create_localhost", false)
	transfer, _ := a.get("app_state.transfer")
	transferObject := a.requireExactObjectKeys(
		"app_state.transfer",
		transfer,
		"port_id",
		"denoms",
		"params",
		"total_escrowed",
	)
	if transferObject != nil {
		a.requireExactObjectKeys(
			"app_state.transfer.params",
			transferObject["params"],
			"send_enabled",
			"receive_enabled",
		)
	}
	a.requireString("app_state.transfer.port_id", "transfer")
	for _, path := range []string{
		"app_state.ibc.client_genesis.clients",
		"app_state.ibc.client_genesis.clients_consensus",
		"app_state.ibc.client_genesis.clients_metadata",
		"app_state.ibc.connection_genesis.connections",
		"app_state.ibc.connection_genesis.client_connection_paths",
		"app_state.ibc.channel_genesis.channels",
		"app_state.ibc.channel_genesis.acknowledgements",
		"app_state.ibc.channel_genesis.commitments",
		"app_state.ibc.channel_genesis.receipts",
		"app_state.transfer.denoms",
		"app_state.transfer.total_escrowed",
	} {
		a.requireEmptyArray(path)
	}
	a.requireBool("app_state.transfer.params.send_enabled", false)
	a.requireBool("app_state.transfer.params.receive_enabled", false)
	a.requireBool("app_state.interchainaccounts.controller_genesis_state.params.controller_enabled", false)
	a.requireBool("app_state.interchainaccounts.host_genesis_state.params.host_enabled", false)
	a.requireEmptyArray("app_state.interchainaccounts.host_genesis_state.params.allow_messages")
	a.requireBool("app_state.ibcratelimit.params.enabled", true)
	a.requireEmptyArray("app_state.ibcratelimit.rate_limits")
	for _, path := range []string{
		"app_state.interchainaccounts.controller_genesis_state.active_channels",
		"app_state.interchainaccounts.controller_genesis_state.interchain_accounts",
		"app_state.interchainaccounts.controller_genesis_state.ports",
		"app_state.interchainaccounts.host_genesis_state.active_channels",
		"app_state.interchainaccounts.host_genesis_state.interchain_accounts",
	} {
		a.requireEmptyArray(path)
	}

	// No bridge adapter means there is no accepted external attestation format.
	a.requireEmptyArray("app_state.substrate_bridge.adapters")
	a.requireInteger("app_state.substrate_bridge.params.max_pending_claims_per_attestation", "1000")
	a.requireInteger("app_state.substrate_bridge.params.per_pending_claim_bond_uzrn", "22200")
	a.requireInteger("app_state.substrate_bridge.params.attestation_min_bond_uzrn", "22200000")
	a.requireInteger("app_state.substrate_bridge.params.pending_claim_rejection_threshold_bps", "1000")
	a.requireInteger("app_state.substrate_bridge.params.min_verified_ratio_for_settle_bps", "6667")
	a.requireInteger("app_state.substrate_bridge.params.witness_reward_challenge_window_blocks", "274176")

	// Knowledge remains queryable but all admission routes are economically
	// unreachable at genesis and every autonomous mint/reward path is off.
	a.requireInteger("app_state.knowledge.params.min_review_fee", protocolAdmissionBarrier)
	a.requireInteger("app_state.knowledge.params.min_challenge_stake", protocolAdmissionBarrier)
	a.requireBool("app_state.knowledge.params.bootstrap_fund_enabled", false)
	a.requireBool("app_state.knowledge.params.demand_tracking_enabled", false)
	a.requireBool("app_state.knowledge.params.vindication_refund_enabled", false)
	a.requireInteger("app_state.knowledge.params.contribution_challenge_bond", protocolAdmissionBarrier)
	a.requireInteger("app_state.knowledge.params.contribution_challenge_reward_multiplier_bps", "1000000")
	a.requireEmptyArray("app_state.knowledge.params.guardian_addresses")
	for _, path := range []string{
		"app_state.knowledge.params.verification_reward",
		"app_state.knowledge.params.demand_bounty_base_reward",
		"app_state.knowledge.params.demand_bounty_per_query_bonus",
		"app_state.knowledge.params.training_fund_base_reward",
		"app_state.knowledge.params.probe_bounty_mint_per_block",
		"app_state.knowledge.params.invitation_bonus_amount",
		"app_state.knowledge.bootstrap_fund_allocation",
		"app_state.knowledge.training_fund_allocation",
	} {
		a.requireInteger(path, "0")
	}
	for _, path := range []string{
		"app_state.knowledge.facts",
		"app_state.knowledge.pending_claims",
		"app_state.knowledge.active_rounds",
		"app_state.knowledge.common_knowledge",
		"app_state.knowledge.training_fund_disbursements",
		"app_state.knowledge.augmentation_bounties",
		"app_state.knowledge.augmentations",
		"app_state.knowledge.contribution_challenges",
	} {
		a.requireEmptyArray(path)
	}

	// Automatic issuance is retired. The protocol-dark profile must preserve
	// that invariant even before any validators begin producing blocks.
	a.requireInteger("app_state.vesting_rewards.params.block_reward", "0")
	a.requireInteger("app_state.vesting_rewards.params.floor_reward", "0")
	a.requireInteger("app_state.vesting_rewards.params.empty_block_reward_rate", "0")
	a.requireInteger("app_state.vesting_rewards.params.min_validators_for_full_reward", "22")
	a.requireInteger("app_state.vesting_rewards.params.initial_fund_balance", "0")
	a.requireInteger("app_state.vesting_rewards.params.founder_share_bps", "0")
	a.requireString("app_state.vesting_rewards.params.founder_address", "")
	a.requireBool("app_state.vesting_rewards.params.vesting_enabled", false)
	a.requireInteger("app_state.vesting_rewards.params.knowledge_coupling_target_bps", "0")
	a.requireInteger("app_state.vesting_rewards.params.knowledge_coupling_floor_bps", "0")

	// No bootstrap registrar and no seed pots: gov cannot be mistaken for an
	// operator admission key during the public beta.
	a.requireString("app_state.claiming_pot.params.bootstrap_registrar", "")
	a.requireEmptyArray("app_state.claiming_pot.pots")
	a.requireEmptyArray("app_state.claiming_pot.claims")

	// Source presence is not activation authority. The native transfer
	// scheduler begins empty and admission-closed; a later governance action
	// needs its own reviewed operational decision and load rehearsal. The
	// retired generic scheduler namespace is forbidden, not silently ignored.
	a.requireAbsent("app_state.schedule")
	messageSchedule, _ := a.get("app_state.message_schedule")
	messageScheduleObject := a.requireExactObjectKeys(
		"app_state.message_schedule",
		messageSchedule,
		"params",
		"schedules",
		"receipts",
		"next_schedule_id",
		"total_escrow_uzrn",
	)
	if messageScheduleObject != nil {
		a.requireExactObjectKeys(
			"app_state.message_schedule.params",
			messageScheduleObject["params"],
			"accept_new_schedules",
			"min_schedule_delay_blocks",
			"min_interval_blocks",
			"max_executions_per_schedule",
			"max_active_schedules_per_creator",
			"max_due_records_per_block",
			"max_query_limit",
			"execution_fee_uzrn",
			"max_transfer_per_execution_uzrn",
		)
	}
	a.requireBool("app_state.message_schedule.params.accept_new_schedules", false)
	for path, expected := range map[string]string{
		"min_schedule_delay_blocks":        "2",
		"min_interval_blocks":              "10",
		"max_executions_per_schedule":      "365",
		"max_active_schedules_per_creator": "32",
		"max_due_records_per_block":        "64",
		"max_query_limit":                  "100",
		"execution_fee_uzrn":               "100000",
		"max_transfer_per_execution_uzrn":  "1000000000000",
	} {
		a.requireInteger("app_state.message_schedule.params."+path, expected)
	}
	a.requireEmptyArray("app_state.message_schedule.schedules")
	a.requireEmptyArray("app_state.message_schedule.receipts")
	a.requireInteger("app_state.message_schedule.next_schedule_id", "1")
	a.requireInteger("app_state.message_schedule.total_escrow_uzrn", "0")

	// An empty genesis council and a four-address floor keep the one-validator
	// launch from presenting unilateral operator control as plural emergency
	// governance. The guardian stake floor also exceeds all genesis supply.
	a.requireEmptyArray("app_state.emergency.params.genesis_council")
	a.requireInteger("app_state.emergency.params.council_expiry_block", "0")
	a.requireInteger("app_state.emergency.params.min_distinct_voters", "4")
	a.requireInteger("app_state.emergency.params.min_guardian_stake", protocolAdmissionBarrier)
	a.requireInteger("app_state.emergency.params.max_revert_depth", "1111")
	a.requireString("app_state.emergency.status", "normal")

	// Custom staking is intentionally empty until the separately funded,
	// post-genesis registration transaction has succeeded.
	a.requireInteger("app_state.zerone_staking.params.min_self_delegation", customStakeMinimum)
	a.requireInteger("app_state.zerone_staking.params.min_stake_for_verification", customStakeMinimum)
	a.requireInteger("app_state.zerone_staking.params.max_validators", "33")
	for _, path := range []string{
		"app_state.zerone_staking.validators",
		"app_state.zerone_staking.delegations",
		"app_state.zerone_staking.unbonding_entries",
	} {
		a.requireEmptyArray(path)
	}
	a.requireInteger("app_state.zerone_staking.unbonding_seq", "0")

	// Adjacent autonomous surfaces are frozen too. They are not required for
	// block production and must remain dark until a separately reviewed
	// activation artifact changes this profile.
	a.requireBool("app_state.alignment.params.enabled", false)
	a.requireBool("app_state.alignment.state.enabled", false)
	for _, path := range []string{
		"app_state.alignment.observations",
		"app_state.alignment.scores",
		"app_state.alignment.health_indices",
		"app_state.alignment.corrections",
	} {
		a.requireEmptyArray(path)
	}
	a.requireBool("app_state.counterexamples.params.proposals_enabled", false)
	a.requireEmptyArray("app_state.counterexamples.counterexamples")
	a.requireEmptyArray("app_state.counterexamples.validations")
	a.requireInteger("app_state.liquiditypool.params.max_pools", "3")
	a.requireInteger("app_state.liquiditypool.params.min_initial_liquidity", protocolAdmissionBarrier)
	a.requireInteger("app_state.liquiditypool.params.protocol_fee_bps", "0")
	a.requireEmptyArray("app_state.liquiditypool.params.billing_quote_denoms")
	a.requireEmptyArray("app_state.liquiditypool.pools")
	a.requireEmptyArray("app_state.liquiditypool.twap_accumulators")
}

func (a *auditor) scanJSONSecrets(value any, path string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					return unicode.ToLower(r)
				}
				return -1
			}, key)
			switch normalized {
			case "mnemonic", "privkey", "privatekey", "seedphrase", "recoveryphrase", "secretkey", "secret":
				a.add(path+"."+key, "private/secret key field is forbidden")
			}
			a.scanJSONSecrets(child, path+"."+key)
		}
	case []any:
		for i, child := range typed {
			a.scanJSONSecrets(child, fmt.Sprintf("%s[%d]", path, i))
		}
	case string:
		for _, marker := range prohibitedContent {
			if strings.Contains(typed, string(marker)) {
				a.add(path, "contains prohibited private-key marker %q", marker)
			}
		}
	}
}

func (a *auditor) requireAccountGenesisState(path string, account map[string]any, accountNumber string) {
	a.requireIntegerValue(path+".account_number", account["account_number"], accountNumber)
	a.requireIntegerValue(path+".sequence", account["sequence"], "0")
}

func (a *auditor) requireCoin(path string, raw any, expectedDenom, expectedAmount string) {
	coin := a.requireExactObjectKeys(path, raw, "denom", "amount")
	if coin == nil {
		return
	}
	gotDenom, _ := coin["denom"].(string)
	if gotDenom != expectedDenom {
		a.add(path+".denom", "got %q, expected %q", gotDenom, expectedDenom)
	}
	a.requireIntegerValue(path+".amount", coin["amount"], expectedAmount)
}

func (a *auditor) requireCoinArray(path, expectedDenom, expectedAmount string) {
	coins, ok := a.slice(path)
	if !ok {
		return
	}
	if len(coins) != 1 {
		a.add(path, "must contain exactly one %s coin", expectedDenom)
		return
	}
	a.requireCoin(path+"[0]", coins[0], expectedDenom, expectedAmount)
}

func (a *auditor) requirePublicKey(path string, key map[string]any, expectedType string, expectedBytes int) []byte {
	if got, _ := key["@type"].(string); got != expectedType {
		a.add(path+".@type", "got %q, expected %q", got, expectedType)
	}
	encoded, ok := key["key"].(string)
	if !ok {
		a.add(path+".key", "must be base64")
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != expectedBytes || allZero(decoded) {
		a.add(path+".key", "must be a non-zero %d-byte base64 public key", expectedBytes)
		return nil
	}
	return decoded
}

func allZero(b []byte) bool {
	for _, value := range b {
		if value != 0 {
			return false
		}
	}
	return true
}

func (a *auditor) get(path string) (any, bool) {
	var current any = a.root
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func (a *auditor) requireExactObjectKeys(path string, raw any, expected ...string) map[string]any {
	object, ok := raw.(map[string]any)
	if !ok {
		a.add(path, "must be an object")
		return nil
	}
	allowed := make(map[string]struct{}, len(expected))
	for _, key := range expected {
		allowed[key] = struct{}{}
		if _, exists := object[key]; !exists {
			a.add(path+"."+key, "missing required field")
		}
	}
	for key := range object {
		if _, exists := allowed[key]; !exists {
			a.add(path+"."+key, "unexpected field in exact launch profile")
		}
	}
	return object
}

func (a *auditor) slice(path string) ([]any, bool) {
	value, ok := a.get(path)
	if !ok {
		a.add(path, "missing required array")
		return nil, false
	}
	items, ok := value.([]any)
	if !ok {
		a.add(path, "must be an array")
		return nil, false
	}
	return items, true
}

func (a *auditor) requireEmptyArray(path string) {
	items, ok := a.slice(path)
	if ok && len(items) != 0 {
		a.add(path, "must be empty, got %d item(s)", len(items))
	}
}

func (a *auditor) requireAbsent(path string) {
	if _, ok := a.get(path); ok {
		a.add(path, "retired or forbidden field must be absent")
	}
}

func (a *auditor) requireStringArray(path string, expected []string) {
	items, ok := a.slice(path)
	if !ok {
		return
	}
	if len(items) != len(expected) {
		a.add(path, "must equal %v, got %v", expected, items)
		return
	}
	for i, item := range items {
		value, ok := item.(string)
		if !ok || value != expected[i] {
			a.add(path, "must equal %v, got %v", expected, items)
			return
		}
	}
}

func (a *auditor) requireEmptyArrayValue(path string, value any, required bool) {
	if value == nil && !required {
		return
	}
	items, ok := value.([]any)
	if !ok {
		a.add(path, "must be an array")
		return
	}
	if len(items) != 0 {
		a.add(path, "must be empty, got %d item(s)", len(items))
	}
}

func (a *auditor) requireString(path, expected string) {
	value, ok := a.get(path)
	if !ok {
		a.add(path, "missing required string (expected %q)", expected)
		return
	}
	got, ok := value.(string)
	if !ok || got != expected {
		a.add(path, "got %v, expected string %q", value, expected)
	}
}

func (a *auditor) requireBool(path string, expected bool) {
	value, ok := a.get(path)
	if !ok {
		a.add(path, "missing required boolean (expected %t)", expected)
		return
	}
	got, ok := value.(bool)
	if !ok || got != expected {
		a.add(path, "got %v, expected %t", value, expected)
	}
}

func (a *auditor) requireInteger(path, expected string) {
	value, ok := a.get(path)
	if !ok {
		a.add(path, "missing required integer (expected %s)", expected)
		return
	}
	a.requireIntegerValue(path, value, expected)
}

func (a *auditor) requireIntegerValue(path string, value any, expected string) {
	got, ok := integerString(value)
	if !ok || got != expected {
		a.add(path, "got %v, expected integer %s", value, expected)
	}
}

func integerString(value any) (string, bool) {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case json.Number:
		raw = typed.String()
	default:
		return "", false
	}
	parsed, ok := new(big.Int).SetString(raw, 10)
	if !ok || parsed.String() != raw {
		return "", false
	}
	return parsed.String(), true
}

func (a *auditor) add(path, format string, args ...any) {
	a.issues = append(a.issues, issue{Path: path, Message: fmt.Sprintf(format, args...)})
}

func sortIssues(issues []issue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path == issues[j].Path {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Path < issues[j].Path
	})
}

// decodeBech32 validates a classic bech32 string and returns the decoded
// 8-bit payload. Zerone account and validator operator addresses both encode
// the same 20-byte account identifier under different HRPs.
func decodeBech32(encoded, expectedHRP string) (string, []byte, error) {
	if encoded == "" || strings.ToLower(encoded) != encoded || strings.ToUpper(encoded) == encoded {
		return "", nil, fmt.Errorf("must be non-empty lowercase bech32")
	}
	separator := strings.LastIndexByte(encoded, '1')
	if separator < 1 || separator+7 > len(encoded) {
		return "", nil, fmt.Errorf("invalid separator/checksum length")
	}
	hrp := encoded[:separator]
	if hrp != expectedHRP {
		return "", nil, fmt.Errorf("HRP %q, expected %q", hrp, expectedHRP)
	}
	charset := "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	values := make([]byte, 0, len(encoded)-separator-1)
	for _, char := range encoded[separator+1:] {
		index := strings.IndexRune(charset, char)
		if index < 0 {
			return "", nil, fmt.Errorf("invalid data character %q", char)
		}
		values = append(values, byte(index))
	}
	if bech32Polymod(append(bech32HRPExpand(hrp), values...)) != 1 {
		return "", nil, fmt.Errorf("checksum mismatch")
	}
	payload, err := convertBits(values[:len(values)-6], 5, 8, false)
	if err != nil {
		return "", nil, err
	}
	return hrp, payload, nil
}

func bech32HRPExpand(hrp string) []byte {
	result := make([]byte, 0, len(hrp)*2+1)
	for _, char := range hrp {
		result = append(result, byte(char>>5))
	}
	result = append(result, 0)
	for _, char := range hrp {
		result = append(result, byte(char&31))
	}
	return result
}

func bech32Polymod(values []byte) uint32 {
	checksum := uint32(1)
	generators := [...]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	for _, value := range values {
		top := checksum >> 25
		checksum = (checksum&0x1ffffff)<<5 ^ uint32(value)
		for i, generator := range generators {
			if (top>>i)&1 == 1 {
				checksum ^= generator
			}
		}
	}
	return checksum
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var accumulator uint32
	var bits uint
	maxValue := uint32(1<<toBits) - 1
	maxAccumulator := uint32(1<<(fromBits+toBits-1)) - 1
	result := make([]byte, 0, len(data)*int(fromBits)/int(toBits))
	for _, value := range data {
		if value>>fromBits != 0 {
			return nil, fmt.Errorf("invalid bech32 value")
		}
		accumulator = (accumulator<<fromBits | uint32(value)) & maxAccumulator
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			result = append(result, byte(accumulator>>bits&maxValue))
		}
	}
	if pad {
		if bits > 0 {
			result = append(result, byte(accumulator<<(toBits-bits)&maxValue))
		}
	} else if bits >= fromBits || accumulator<<(toBits-bits)&maxValue != 0 {
		return nil, fmt.Errorf("invalid bech32 padding")
	}
	return result, nil
}
