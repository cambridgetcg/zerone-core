package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"slices"
	"strings"
	"time"

	cmtcrypto "github.com/cometbft/cometbft/crypto"
)

var requiredForkGenesisModules = []string{
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
	"message_schedule",
	"slashing",
	"staking",
	"transfer",
	"upgrade",
	"zerone_gov",
	"zerone_staking",
}

var requiredForkSchemaMigrations = []string{
	"ibc-transfer-v8-empty-denom-traces-to-v10-empty-denoms",
	"ibc-core-v8-empty-to-v10-empty-v2-state",
	"message-schedule-v1-absent-to-default-closed",
}

const (
	messageScheduleGenesisMigration = "message-schedule-v1-absent-to-default-closed"
	// absentModuleSHA256 is SHA-256 over the canonical JSON literal `null`.
	// It can appear only as before_sha256 for an explicitly declared module
	// addition; it never represents an application-state value.
	absentModuleSHA256 = "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"
)

var freshMessageScheduleGenesis = json.RawMessage(
	`{"params":{"min_schedule_delay_blocks":2,"min_interval_blocks":10,"max_executions_per_schedule":365,"max_active_schedules_per_creator":32,"max_due_records_per_block":64,"max_query_limit":100,"execution_fee_uzrn":"100000","max_transfer_per_execution_uzrn":"1000000000000"},"next_schedule_id":1,"total_escrow_uzrn":"0"}`,
)

func decodeForkGenesis(data []byte) (ForkGenesis, error) {
	var zero ForkGenesis
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return zero, errors.New("fork genesis contains ambiguous duplicate JSON keys")
	}
	var genesis ForkGenesis
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&genesis); err != nil {
		return zero, errors.New("fork genesis does not match the exact AppGenesis envelope")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return zero, errors.New("fork genesis JSON must contain exactly one value")
	}
	canonical, err := json.Marshal(genesis)
	if err != nil || !bytes.Equal(data, canonical) {
		return zero, errors.New("fork genesis must be exact compact canonical AppGenesis JSON")
	}
	return genesis, nil
}

func validateForkArtifacts(
	genesis ForkGenesis,
	genesisFileSHA256 string,
	reports []ForkGenesisReport,
	reportFileSHA256s []string,
	release ForkRelease,
	policy ForkPolicy,
	policyFileSHA256 string,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
) error {
	if genesisFileSHA256 != release.GenesisSHA256 {
		return errors.New("exact fork genesis file digest does not match release")
	}
	exactGenesisSHA256, err := canonicalDigest(genesis)
	if err != nil || exactGenesisSHA256 != genesisFileSHA256 {
		return errors.New("fork genesis digest is not the exact canonical artifact digest")
	}
	if err := validateForkGenesis(genesis, release, assessment); err != nil {
		return err
	}
	moduleAfterSHA256s, err := forkGenesisModuleDigests(genesis.AppState)
	if err != nil {
		return err
	}
	if len(reports) != 2 || len(reportFileSHA256s) != 2 {
		return errors.New("exactly two compiler report files are required")
	}
	if reportFileSHA256s[0] == reportFileSHA256s[1] {
		return errors.New("compiler report files must have distinct exact digests")
	}

	reproductionByReport := make(map[string]GenesisReproduction, 2)
	for _, reproduction := range release.GenesisReproductions {
		reproductionByReport[reproduction.CompilerReportFileSHA256] = reproduction
	}
	for index, report := range reports {
		fileDigest := reportFileSHA256s[index]
		exactReportSHA256, err := canonicalDigest(report)
		if err != nil || exactReportSHA256 != fileDigest {
			return errors.New("compiler report digest is not the exact canonical artifact digest")
		}
		reproduction, found := reproductionByReport[fileDigest]
		if !found {
			return errors.New("compiler report file digest is absent from signed reproductions")
		}
		if err := validateForkGenesisReport(
			report,
			fileDigest,
			reproduction,
			release,
			policy,
			policyFileSHA256,
			assessment,
			assessmentFileSHA256,
			moduleAfterSHA256s,
		); err != nil {
			return err
		}
	}
	if err := requireEquivalentCompilerReports(reports[0], reports[1]); err != nil {
		return err
	}
	return nil
}

func validateForkGenesis(
	genesis ForkGenesis,
	release ForkRelease,
	assessment CustodyAssessment,
) error {
	if genesis.AppName != "zeroned" {
		return errors.New("fork genesis app_name must be zeroned")
	}
	if err := validateLabel("fork genesis app_version", genesis.AppVersion, 512); err != nil {
		return err
	}
	if genesis.ChainID != release.NewChainID ||
		genesis.InitialHeight <= 1 ||
		uint64(genesis.InitialHeight) != release.InitialHeight {
		return errors.New("fork genesis chain or initial height does not match release")
	}
	genesisTime, err := parseCanonicalUTCTime("fork genesis time", genesis.GenesisTime)
	if err != nil {
		return err
	}
	checkpointTime, err := parseCanonicalUTCTime(
		"checkpoint block time",
		release.Checkpoint.BlockTime,
	)
	if err != nil {
		return err
	}
	if !genesisTime.After(checkpointTime) {
		return errors.New("fork genesis time must be strictly after the signed checkpoint time")
	}
	if len(genesis.AppState) == 0 || bytes.Equal(genesis.AppState, []byte("null")) ||
		len(genesis.Consensus) == 0 || bytes.Equal(genesis.Consensus, []byte("null")) {
		return errors.New("fork genesis app_state and consensus are required")
	}
	if err := rejectDuplicateJSONKeys(genesis.AppState); err != nil {
		return errors.New("fork genesis app_state contains ambiguous duplicate JSON keys")
	}
	if err := rejectDuplicateJSONKeys(genesis.Consensus); err != nil {
		return errors.New("fork genesis consensus contains ambiguous duplicate JSON keys")
	}
	if !bytes.Equal(genesis.AppHash, []byte("null")) {
		return errors.New("fork genesis app_hash must be null for a non-zero-height continuation")
	}
	if len(release.NewValidators) != 1 {
		return errors.New("consensus-key-only genesis requires exactly one release validator")
	}
	validator := release.NewValidators[0]
	consensusPower, err := validateGenesisConsensus(genesis.Consensus, validator)
	if err != nil {
		return err
	}
	if err := validateGenesisStaking(
		genesis.AppState,
		validator,
		assessment,
		consensusPower,
	); err != nil {
		return err
	}
	if err := validateForkGenesisQuiescence(
		genesis.AppState,
		release.InitialHeight,
	); err != nil {
		return err
	}
	if err := rejectOldConsensusIdentity(genesis, assessment); err != nil {
		return err
	}
	return nil
}

func validateGenesisConsensus(
	raw json.RawMessage,
	validator ValidatorIdentity,
) (*big.Int, error) {
	type consensusPublicKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type consensusValidator struct {
		Address string             `json:"address"`
		PubKey  consensusPublicKey `json:"pub_key"`
		Power   string             `json:"power"`
		Name    string             `json:"name"`
	}
	var consensus struct {
		Validators []consensusValidator `json:"validators"`
		Params     json.RawMessage      `json:"params"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&consensus); err != nil {
		return nil, errors.New("fork genesis consensus envelope is invalid")
	}
	if len(consensus.Validators) != 1 {
		return nil, errors.New("fork genesis must contain exactly one consensus validator")
	}
	actual := consensus.Validators[0]
	if actual.Address != validator.ConsensusAddress ||
		actual.PubKey.Type != "tendermint/PubKeyEd25519" {
		return nil, errors.New("fork genesis consensus validator identity does not match release")
	}
	keyBytes, err := decodeCanonicalBase64Ed25519(
		"fork genesis consensus public key",
		actual.PubKey.Value,
	)
	if err != nil || hex.EncodeToString(keyBytes) != validator.ConsensusPublicKey {
		return nil, errors.New("fork genesis consensus public key does not match release")
	}
	power, err := parseCanonicalPositiveInteger(
		"fork genesis consensus power",
		actual.Power,
	)
	if err != nil {
		return nil, err
	}
	if len(consensus.Params) == 0 || bytes.Equal(consensus.Params, []byte("null")) {
		return nil, errors.New("fork genesis consensus params are required")
	}
	return power, nil
}

func validateGenesisStaking(
	appStateRaw json.RawMessage,
	validator ValidatorIdentity,
	assessment CustodyAssessment,
	consensusPower *big.Int,
) error {
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(appStateRaw, &appState); err != nil {
		return errors.New("fork genesis app_state is invalid")
	}
	stakingRaw, found := appState["staking"]
	if !found {
		return errors.New("fork genesis staking module is absent")
	}
	var staking struct {
		Validators []struct {
			ConsensusPubkey struct {
				Type string `json:"@type"`
				Key  string `json:"key"`
			} `json:"consensus_pubkey"`
			OperatorAddress string `json:"operator_address"`
			Status          string `json:"status"`
			Tokens          string `json:"tokens"`
			Jailed          bool   `json:"jailed"`
		} `json:"validators"`
	}
	if err := json.Unmarshal(stakingRaw, &staking); err != nil {
		return errors.New("fork genesis staking module is invalid")
	}
	bonded := 0
	matched := 0
	for _, stakingValidator := range staking.Validators {
		if stakingValidator.Status == "BOND_STATUS_BONDED" {
			bonded++
		}
		if stakingValidator.OperatorAddress != validator.SDKOperatorAddress {
			continue
		}
		matched++
		keyBytes, err := decodeCanonicalBase64Ed25519(
			"staking consensus public key",
			stakingValidator.ConsensusPubkey.Key,
		)
		if err != nil ||
			stakingValidator.ConsensusPubkey.Type !=
				"/cosmos.crypto.ed25519.PubKey" ||
			hex.EncodeToString(keyBytes) != validator.ConsensusPublicKey ||
			stakingValidator.Status != "BOND_STATUS_BONDED" ||
			stakingValidator.Jailed {
			return errors.New("fork genesis staking validator does not match the release identity")
		}
		tokens, err := parseCanonicalPositiveInteger(
			"fork genesis staking validator tokens",
			stakingValidator.Tokens,
		)
		if err != nil {
			return err
		}
		expectedPower := new(big.Int).Quo(
			tokens,
			big.NewInt(1_000_000),
		)
		if expectedPower.Sign() <= 0 ||
			expectedPower.Cmp(consensusPower) != 0 {
			return errors.New("fork genesis consensus power does not reconcile to staking tokens")
		}
	}
	if bonded != 1 || matched != 1 ||
		validator.SDKOperatorAddress != assessment.OldValidator.SDKOperatorAddress {
		return errors.New("consensus-key-only genesis must retain exactly one bonded proven-safe SDK operator")
	}
	return nil
}

func validateForkGenesisReport(
	report ForkGenesisReport,
	reportFileSHA256 string,
	reproduction GenesisReproduction,
	release ForkRelease,
	policy ForkPolicy,
	policyFileSHA256 string,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
	moduleAfterSHA256s map[string]string,
) error {
	if report.Schema != forkGenesisReportSchema ||
		report.Profile != rewriteProfileConsensusOnly {
		return errors.New("compiler report schema or profile is unsupported")
	}
	if report.ReproducerID != reproduction.Identity ||
		report.ReproducerControlDomain != reproduction.ControlDomain ||
		report.ReproducerPublicKey != reproduction.PublicKey ||
		reportFileSHA256 != reproduction.CompilerReportFileSHA256 {
		return errors.New("compiler report reproducer binding does not match signed reproduction")
	}
	if report.IncidentID != release.IncidentID ||
		report.SourceGenesisSHA256 != release.SourceExportSHA256 ||
		report.PolicyFileSHA256 != release.RewritePolicyFileSHA256 ||
		report.PolicySHA256 != release.RewritePolicySelfSHA256 ||
		report.SourceChainID != release.OldChainID ||
		report.TargetChainID != release.NewChainID ||
		report.InitialHeight != release.InitialHeight ||
		report.SourceBlockIDSHA256 != release.Checkpoint.BlockIDSHA256 ||
		report.SourceAppHashSHA256 != release.Checkpoint.AppHashSHA256 ||
		report.SourceLastBlockTime != release.Checkpoint.BlockTime ||
		report.SourceSignedCommitSHA256 != release.Checkpoint.SignedCommitSHA256 ||
		report.SourceValidatorSetSHA256 != release.Checkpoint.ValidatorSetSHA256 ||
		report.CustodyAssessmentSHA256 != assessmentFileSHA256 ||
		report.ForkPolicySHA256 != policyFileSHA256 ||
		report.RewriteToolSHA256 != release.RewriteToolSHA256 ||
		report.OutputGenesisSHA256 != release.GenesisSHA256 {
		return errors.New("compiler report does not match assessment, policy, checkpoint, release, or genesis")
	}
	if report.InitialHeight != assessment.Checkpoint.Height+1 {
		return errors.New("compiler report initial height is not checkpoint H+1")
	}
	if report.OperatorDisposition != operatorRetainProvenSafe ||
		report.IBCDisposition != ibcDispositionRequireEmpty ||
		report.EmergencyStartMode != emergencyConsensusQuarantine {
		return errors.New("compiler report recovery disposition is unsupported")
	}
	oldKey, err := decodeCanonicalBase64Ed25519(
		"compiler report old consensus key",
		report.OldConsensusPublicKey,
	)
	if err != nil ||
		hex.EncodeToString(oldKey) != assessment.OldValidator.ConsensusPublicKey {
		return errors.New("compiler report old consensus key does not match assessment")
	}
	newKey, err := decodeCanonicalBase64Ed25519(
		"compiler report new consensus key",
		report.NewConsensusPublicKey,
	)
	if err != nil ||
		len(release.NewValidators) != 1 ||
		hex.EncodeToString(newKey) != release.NewValidators[0].ConsensusPublicKey {
		return errors.New("compiler report new consensus key does not match release")
	}
	oldAddress, err := decodeCanonicalBech32Address(
		"compiler report old consensus address",
		report.OldConsensusAddress,
		"zrnvalcons",
	)
	if err != nil ||
		stringsUpperHex(oldAddress) != assessment.OldValidator.ConsensusAddress {
		return errors.New("compiler report old consensus address does not match assessment")
	}
	newAddress, err := decodeCanonicalBech32Address(
		"compiler report new consensus address",
		report.NewConsensusAddress,
		"zrnvalcons",
	)
	if err != nil ||
		stringsUpperHex(newAddress) != release.NewValidators[0].ConsensusAddress {
		return errors.New("compiler report new consensus address does not match release")
	}
	if err := validateForkSchemaMigrations(report.SchemaMigrations); err != nil {
		return err
	}
	if err := validateForkGenesisModuleDigests(
		report.ModuleDigests,
		moduleAfterSHA256s,
		report.SchemaMigrations,
	); err != nil {
		return err
	}
	actualSelfHash := report.ReportSHA256
	report.ReportSHA256 = ""
	digest, err := canonicalDigest(report)
	if err != nil || actualSelfHash != digest {
		return errors.New("compiler report self hash mismatch")
	}
	if err := validateSHA256("compiler report exact file digest", reportFileSHA256, false); err != nil {
		return err
	}
	return nil
}

func validateForkGenesisModuleDigests(
	digests []ForkGenesisModuleDigest,
	moduleAfterSHA256s map[string]string,
	schemaMigrations []string,
) error {
	if len(digests) != len(moduleAfterSHA256s) ||
		len(digests) < len(requiredForkGenesisModules) ||
		len(digests) > maxCollectionEntries {
		return errors.New("compiler report module digest set is incomplete")
	}
	if !slices.IsSortedFunc(digests, func(left, right ForkGenesisModuleDigest) int {
		return strings.Compare(left.Module, right.Module)
	}) {
		return errors.New("compiler report module digests must be sorted by module")
	}
	seen := make(map[string]bool, len(digests))
	for _, digest := range digests {
		if err := validateLabel("compiler report module", digest.Module, 128); err != nil {
			return err
		}
		if seen[digest.Module] {
			return errors.New("compiler report module digests must be unique")
		}
		seen[digest.Module] = true
		expectedAfter, found := moduleAfterSHA256s[digest.Module]
		if !found || digest.AfterSHA256 != expectedAfter {
			return errors.New("compiler report module digest does not match the exact fork genesis app_state")
		}
		if err := validateSHA256("module before digest", digest.BeforeSHA256, false); err != nil {
			return err
		}
		if err := validateSHA256("module after digest", digest.AfterSHA256, false); err != nil {
			return err
		}
		if digest.Changed != (digest.BeforeSHA256 != digest.AfterSHA256) {
			return errors.New("compiler report module changed flag does not match its digests")
		}
		switch digest.Module {
		case "emergency", "slashing", "staking":
			if !digest.Changed {
				return errors.New("compiler report omits a mandatory audited module rewrite")
			}
		case "ibc":
			expected := slices.Contains(
				schemaMigrations,
				"ibc-core-v8-empty-to-v10-empty-v2-state",
			)
			if digest.Changed != expected {
				return errors.New("compiler report IBC change does not match its schema migration")
			}
		case "transfer":
			expected := slices.Contains(
				schemaMigrations,
				"ibc-transfer-v8-empty-denom-traces-to-v10-empty-denoms",
			)
			if digest.Changed != expected {
				return errors.New("compiler report transfer change does not match its schema migration")
			}
		case "message_schedule":
			if !slices.Contains(schemaMigrations, messageScheduleGenesisMigration) ||
				digest.BeforeSHA256 != absentModuleSHA256 || !digest.Changed {
				return errors.New("compiler report message_schedule addition is not the exact audited absent-to-default migration")
			}
		case "zerone_staking":
			// The custom validator metadata exists only on some source
			// exports, so this audited rewrite is optional.
		default:
			if digest.Changed {
				return errors.New("compiler report changed a module outside the audited profile")
			}
		}
	}
	for _, required := range requiredForkGenesisModules {
		if !seen[required] {
			return errors.New("compiler report module digest set is incomplete")
		}
	}
	return nil
}

func validateForkSchemaMigrations(migrations []string) error {
	if migrations == nil || len(migrations) > len(requiredForkSchemaMigrations) {
		return errors.New("compiler report schema migrations must be a bounded array")
	}
	allowedIndex := 0
	for _, migration := range migrations {
		for allowedIndex < len(requiredForkSchemaMigrations) &&
			requiredForkSchemaMigrations[allowedIndex] != migration {
			allowedIndex++
		}
		if allowedIndex == len(requiredForkSchemaMigrations) {
			return errors.New("compiler report contains an unsupported or non-canonical schema migration")
		}
		allowedIndex++
	}
	return nil
}

func requireEquivalentCompilerReports(
	left,
	right ForkGenesisReport,
) error {
	left.ReproducerID = ""
	left.ReproducerControlDomain = ""
	left.ReproducerPublicKey = ""
	left.ReportSHA256 = ""
	right.ReproducerID = ""
	right.ReproducerControlDomain = ""
	right.ReproducerPublicKey = ""
	right.ReportSHA256 = ""
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if !bytes.Equal(leftJSON, rightJSON) {
		return errors.New("independent compiler reports disagree on recovery output")
	}
	return nil
}

func forkGenesisModuleDigests(
	appStateRaw json.RawMessage,
) (map[string]string, error) {
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(appStateRaw, &appState); err != nil {
		return nil, errors.New("fork genesis app_state is invalid")
	}
	if len(appState) > maxCollectionEntries {
		return nil, errors.New("fork genesis app_state contains too many modules")
	}
	result := make(map[string]string, len(appState))
	for module, raw := range appState {
		if err := validateLabel("fork genesis module", module, 128); err != nil {
			return nil, err
		}
		canonical, err := canonicalJSONValue(raw)
		if err != nil {
			return nil, fmt.Errorf("fork genesis module %s is not canonical JSON", module)
		}
		result[module] = digestBytes(canonical)
	}
	return result, nil
}

func canonicalJSONValue(data []byte) ([]byte, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
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
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if seen[key] {
				return errors.New("duplicate JSON object key")
			}
			seen[key] = true
			if err := scanUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("JSON object did not end with }")
		}
	case '[':
		for decoder.More() {
			if err := scanUniqueJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("JSON array did not end with ]")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func decodeJSONAny(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
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

func requireEmptyArrayPaths(
	module string,
	value any,
	paths []string,
) error {
	for _, path := range paths {
		resolved, found := resolveJSONPath(value, path)
		array, ok := resolved.([]any)
		if !found || !ok || len(array) != 0 {
			return fmt.Errorf(
				"fork genesis %s.%s must be a present empty array",
				module,
				path,
			)
		}
	}
	return nil
}

func absentOrEmptyArray(value any, path string) bool {
	resolved, found := resolveJSONPath(value, path)
	if !found {
		return true
	}
	array, ok := resolved.([]any)
	return ok && len(array) == 0
}

func absentNullOrEmptyArray(value any, path string) bool {
	resolved, found := resolveJSONPath(value, path)
	if !found || resolved == nil {
		return true
	}
	array, ok := resolved.([]any)
	return ok && len(array) == 0
}

func canonicalPositiveJSONInteger(value any) bool {
	var text string
	switch typed := value.(type) {
	case json.Number:
		text = typed.String()
	case string:
		text = typed
	default:
		return false
	}
	_, err := parseCanonicalPositiveInteger("JSON integer", text)
	return err == nil
}

func validateForkGenesisQuiescence(
	appStateRaw json.RawMessage,
	initialHeight uint64,
) error {
	var modules map[string]json.RawMessage
	if err := json.Unmarshal(appStateRaw, &modules); err != nil {
		return errors.New("fork genesis app_state is invalid")
	}
	for _, module := range requiredForkGenesisModules {
		if _, found := modules[module]; !found {
			return fmt.Errorf("fork genesis required module %s is absent", module)
		}
	}
	if _, found := modules["schedule"]; found {
		return errors.New("fork genesis retains the retired schedule module namespace")
	}
	actualSchedule, err := canonicalJSONValue(modules["message_schedule"])
	if err != nil {
		return errors.New("fork genesis message_schedule module is invalid")
	}
	expectedSchedule, err := canonicalJSONValue(freshMessageScheduleGenesis)
	if err != nil {
		panic("invalid embedded fresh message_schedule genesis: " + err.Error())
	}
	if !bytes.Equal(actualSchedule, expectedSchedule) {
		return errors.New("fork genesis message_schedule module is not the exact empty admission-closed default")
	}
	if err := validateForkSchedulerBankBalances(modules["bank"]); err != nil {
		return err
	}

	emergencyValue, err := decodeJSONAny(modules["emergency"])
	if err != nil {
		return errors.New("fork genesis emergency module is invalid")
	}
	emergency, ok := emergencyValue.(map[string]any)
	if !ok ||
		emergency["status"] != "halted" ||
		emergency["active_halt_ceremony_id"] != "legacy-genesis-quarantine" {
		return errors.New("fork genesis emergency module is not in consensus quarantine")
	}
	haltStart, ok := emergency["halt_start_block"]
	if !ok || !canonicalJSONUint64Equals(haltStart, initialHeight) {
		return errors.New("fork genesis emergency halt start is not H+1")
	}
	for _, field := range []string{
		"last_halt_escalation_block",
		"quarantine_release_block",
	} {
		if value, found := emergency[field]; found &&
			!canonicalJSONUint64Equals(value, 0) {
			return errors.New("fork genesis emergency quarantine clock is not reset")
		}
	}
	if authorization, found := emergency["recovery_authorization"]; found &&
		authorization != nil {
		return errors.New("fork genesis must not preload emergency recovery authorization")
	}

	for module, paths := range map[string][]string{
		"genutil":  {"gen_txs"},
		"evidence": {"evidence"},
		"gov":      {"proposals", "deposits", "votes"},
	} {
		value, err := decodeJSONAny(modules[module])
		if err != nil || requireEmptyArrayPaths(module, value, paths) != nil {
			return fmt.Errorf("fork genesis %s module is not quiescent", module)
		}
	}
	upgradeValue, err := decodeJSONAny(modules["upgrade"])
	upgrade, ok := upgradeValue.(map[string]any)
	if err != nil || !ok || len(upgrade) != 0 {
		return errors.New("fork genesis upgrade module must be an empty object")
	}

	customValue, err := decodeJSONAny(modules["zerone_gov"])
	custom, ok := customValue.(map[string]any)
	if err != nil || !ok || custom["params"] == nil ||
		custom["research_fund_governance"] == nil ||
		!canonicalPositiveJSONInteger(custom["next_lip_number"]) ||
		!canonicalPositiveJSONInteger(custom["next_seat_election_number"]) {
		return errors.New("fork genesis zerone_gov exported-state anchors are missing")
	}
	for _, path := range []string{
		"lips",
		"votes",
		"upgrade_plans",
		"seat_elections",
		"seat_election_votes",
		"creed_amendment_pins",
	} {
		if !absentNullOrEmptyArray(customValue, path) {
			return errors.New("fork genesis zerone_gov contains pending governance")
		}
	}
	if hold, found := custom["emergency_transition_hold"]; found && hold != nil {
		return errors.New("fork genesis zerone_gov contains an emergency transition hold")
	}

	if err := validateEmptyForkIBC(modules); err != nil {
		return err
	}
	return nil
}

func validateForkSchedulerBankBalances(raw json.RawMessage) error {
	value, err := decodeJSONAny(raw)
	if err != nil {
		return errors.New("fork genesis bank module is invalid")
	}
	bank, ok := value.(map[string]any)
	if !ok {
		return errors.New("fork genesis bank module must be an object")
	}
	rawBalances, found := bank["balances"]
	balances, ok := rawBalances.([]any)
	if !found || !ok {
		return errors.New("fork genesis bank.balances must be an array")
	}
	targets := []struct {
		address []byte
		label   string
	}{
		{address: cmtcrypto.AddressHash([]byte("schedule")), label: "retired schedule"},
		{address: cmtcrypto.AddressHash([]byte("message_schedule")), label: "fresh message_schedule"},
	}
	for index, rawBalance := range balances {
		balance, ok := rawBalance.(map[string]any)
		if !ok {
			return fmt.Errorf("fork genesis bank.balances[%d] must be an object", index)
		}
		address, ok := balance["address"].(string)
		if !ok || address == "" {
			return fmt.Errorf("fork genesis bank.balances[%d].address is invalid", index)
		}
		addressBytes, err := decodeCanonicalBech32Address(
			fmt.Sprintf("fork genesis bank.balances[%d].address", index),
			address,
			"zrn",
		)
		if err != nil {
			return err
		}
		label := ""
		for _, target := range targets {
			if bytes.Equal(addressBytes, target.address) {
				label = target.label
				break
			}
		}
		if label == "" {
			continue
		}
		coins, ok := balance["coins"].([]any)
		if !ok {
			return fmt.Errorf("fork genesis %s module account balance is not a coin array", label)
		}
		if len(coins) != 0 {
			return fmt.Errorf("fork genesis %s module account must have zero all-denom balance", label)
		}
	}
	return nil
}

func schedulerModuleAccountAddress(moduleName string) string {
	encoded, err := encodeBech32("zrn", cmtcrypto.AddressHash([]byte(moduleName)))
	if err != nil {
		panic("encode scheduler module account address: " + err.Error())
	}
	return encoded
}

func canonicalJSONUint64Equals(value any, expected uint64) bool {
	number, ok := value.(json.Number)
	if !ok || number.String() == "" ||
		(len(number.String()) > 1 && number.String()[0] == '0') {
		return false
	}
	parsed := new(big.Int)
	if _, ok := parsed.SetString(number.String(), 10); !ok ||
		parsed.Sign() < 0 || !parsed.IsUint64() {
		return false
	}
	return parsed.Uint64() == expected
}

func validateEmptyForkIBC(
	modules map[string]json.RawMessage,
) error {
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
			"client_v2_genesis.counterparty_infos",
			"channel_v2_genesis.acknowledgements",
			"channel_v2_genesis.commitments",
			"channel_v2_genesis.receipts",
			"channel_v2_genesis.async_packets",
			"channel_v2_genesis.send_sequences",
		},
		"transfer": {
			"total_escrowed",
			"denoms",
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
		value, err := decodeJSONAny(modules[module])
		if err != nil {
			return fmt.Errorf("fork genesis %s module is invalid", module)
		}
		if err := requireEmptyArrayPaths(module, value, paths); err != nil {
			return err
		}
		if module == "transfer" {
			if _, hasLegacy := resolveJSONPath(value, "denom_traces"); hasLegacy {
				return errors.New("fork genesis transfer module retains legacy denom_traces")
			}
		}
		if module == "ibc" {
			if _, hasLegacy := resolveJSONPath(value, "channel_genesis.params"); hasLegacy {
				return errors.New("fork genesis IBC module retains legacy channel params")
			}
		}
	}
	rateLimitValue, err := decodeJSONAny(modules["ibcratelimit"])
	if err != nil ||
		!absentOrEmptyArray(rateLimitValue, "rate_limits") {
		return errors.New("fork genesis IBC rate-limit state is not empty")
	}
	return nil
}

func rejectOldConsensusIdentity(
	genesis ForkGenesis,
	assessment CustodyAssessment,
) error {
	canonical, err := json.Marshal(genesis)
	if err != nil {
		return errors.New("fork genesis cannot be encoded for old-key scan")
	}
	oldKey, _ := hex.DecodeString(assessment.OldValidator.ConsensusPublicKey)
	oldAddress, _ := hex.DecodeString(
		strings.ToLower(assessment.OldValidator.ConsensusAddress),
	)
	oldBech32, err := encodeBech32("zrnvalcons", oldAddress)
	if err != nil {
		return errors.New("old consensus address cannot be derived for eradication scan")
	}
	for _, prohibited := range [][]byte{
		[]byte(base64.StdEncoding.EncodeToString(oldKey)),
		[]byte(hex.EncodeToString(oldKey)),
		[]byte(strings.ToUpper(hex.EncodeToString(oldKey))),
		[]byte(oldBech32),
		[]byte(base64.StdEncoding.EncodeToString(oldAddress)),
		[]byte(hex.EncodeToString(oldAddress)),
		[]byte(assessment.OldValidator.ConsensusAddress),
	} {
		if bytes.Contains(canonical, prohibited) {
			return errors.New("old consensus identity remains in the exact fork genesis")
		}
	}
	return nil
}

func decodeCanonicalBase64Ed25519(
	name,
	value string,
) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != 32 ||
		base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("%s must be canonical base64 Ed25519 bytes", name)
	}
	return decoded, nil
}

func parseCanonicalUTCTime(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Location() != time.UTC ||
		parsed.UTC().Format(time.RFC3339Nano) != value {
		return time.Time{}, fmt.Errorf("%s must be canonical UTC RFC3339Nano using Z", name)
	}
	return parsed, nil
}

func stringsUpperHex(value []byte) string {
	return strings.ToUpper(hex.EncodeToString(value))
}

func decodeCanonicalBech32Address(
	name,
	value,
	expectedHRP string,
) ([]byte, error) {
	hrp, payload, err := decodeBech32(value)
	if err != nil || hrp != expectedHRP || len(payload) != cometAddressBytes {
		return nil, fmt.Errorf("%s must be canonical %s Bech32 for 20 bytes", name, expectedHRP)
	}
	canonical, err := encodeBech32(expectedHRP, payload)
	if err != nil || canonical != value {
		return nil, fmt.Errorf("%s must be canonical %s Bech32 for 20 bytes", name, expectedHRP)
	}
	return payload, nil
}
