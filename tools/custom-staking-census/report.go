package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	reportSchema   = "zerone/custom-staking-census/v1"
	maxReportBytes = 64 << 20
)

type reportEvidence struct {
	ChainID      string `json:"chain_id"`
	Height       string `json:"height"`
	AppHash      string `json:"app_hash"`
	SourceCommit string `json:"source_commit"`
}

type reportStore struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	RootSHA256   string `json:"root_sha256"`
	LeafCount    string `json:"leaf_count"`
	InputBytes   string `json:"input_bytes"`
	LeavesSHA256 string `json:"leaves_sha256"`
}

type reportMultistoreCommitment struct {
	Name       string `json:"name"`
	RootSHA256 string `json:"root_sha256"`
}

type sealedCensusReport struct {
	Schema       string                       `json:"schema"`
	Result       string                       `json:"result"`
	Evidence     reportEvidence               `json:"evidence"`
	Multistore   []reportMultistoreCommitment `json:"multistore"`
	Stores       []reportStore                `json:"stores"`
	Census       censusResult                 `json:"census"`
	ReportSHA256 string                       `json:"report_sha256"`
}

func executeCensus(
	db physicalDB,
	options censusOptions,
) ([]byte, bool, error) {
	if db == nil {
		return nil, false, errors.New("application database is nil")
	}
	collector := newCensus()
	visitors := make(map[string]func(logicalLeaf) error, len(requiredStoreNames))
	for _, storeName := range requiredStoreNames {
		name := storeName
		visitors[name] = func(leaf logicalLeaf) error {
			return collector.ingest(name, leaf.Key, leaf.Value)
		}
	}

	snapshot, stores, err := scanApplicationDB(db, expectedEvidence{
		Height:  options.Height,
		AppHash: options.AppHash,
	}, visitors)
	if err != nil {
		return nil, false, err
	}
	result := collector.finalize()
	return buildCensusReport(options, snapshot, stores, result)
}

func buildCensusReport(
	options censusOptions,
	snapshot rootSnapshot,
	stores []storeEvidence,
	result censusResult,
) ([]byte, bool, error) {
	if !validChainID(options.ChainID) {
		return nil, false, errors.New("report chain ID is invalid")
	}
	if len(options.SourceCommit) != 40 || strings.ToLower(options.SourceCommit) != options.SourceCommit {
		return nil, false, errors.New("report source commit must be 40 hexadecimal characters")
	}
	if _, err := hex.DecodeString(options.SourceCommit); err != nil {
		return nil, false, fmt.Errorf("report source commit is not hexadecimal: %w", err)
	}
	if options.Height <= 0 {
		return nil, false, errors.New("report height must be positive")
	}
	if len(options.AppHash) != sha256.Size {
		return nil, false, fmt.Errorf(
			"report app hash must be %d bytes: got %d",
			sha256.Size,
			len(options.AppHash),
		)
	}
	multistore, err := validateAndRenderMultistore(options, snapshot)
	if err != nil {
		return nil, false, err
	}
	if len(stores) != len(requiredStoreNames) {
		return nil, false, fmt.Errorf(
			"report requires exactly %d store evidence records: got %d",
			len(requiredStoreNames),
			len(stores),
		)
	}
	reportStores := make([]reportStore, len(stores))
	for index, store := range stores {
		if store.name != requiredStoreNames[index] {
			return nil, false, fmt.Errorf(
				"store evidence %d must be %q: got %q",
				index,
				requiredStoreNames[index],
				store.name,
			)
		}
		if store.version != options.Height {
			return nil, false, fmt.Errorf(
				"store %q version must equal report height %d: got %d",
				store.name,
				options.Height,
				store.version,
			)
		}
		if store.leafCount < 0 {
			return nil, false, fmt.Errorf("store %q leaf count is negative", store.name)
		}
		if len(store.rootHash) != sha256.Size {
			return nil, false, fmt.Errorf(
				"store %q root hash must be %d bytes: got %d",
				store.name,
				sha256.Size,
				len(store.rootHash),
			)
		}
		if len(store.leavesHash) != sha256.Size {
			return nil, false, fmt.Errorf(
				"store %q leaves hash must be %d bytes: got %d",
				store.name,
				sha256.Size,
				len(store.leavesHash),
			)
		}
		committed, exists := snapshot.stores[store.name]
		if !exists || committed.version != store.version || !bytes.Equal(committed.rootHash, store.rootHash) {
			return nil, false, fmt.Errorf("store %q scan evidence does not match its multistore commitment", store.name)
		}
		reportStores[index] = reportStore{
			Name:         store.name,
			Version:      strconv.FormatInt(store.version, 10),
			RootSHA256:   hex.EncodeToString(store.rootHash),
			LeafCount:    strconv.FormatInt(store.leafCount, 10),
			InputBytes:   strconv.FormatUint(store.inputBytes, 10),
			LeavesSHA256: hex.EncodeToString(store.leavesHash),
		}
	}
	if err := validateCensusResultForReport(result, stores); err != nil {
		return nil, false, err
	}
	if estimateReportJSONUpperBound(options, multistore, reportStores, result) > maxReportBytes {
		return nil, false, fmt.Errorf("census report exceeds the conservative %d-byte resource bound", maxReportBytes)
	}

	status := "FAIL"
	if result.Passed {
		status = "PASS"
	}
	report := sealedCensusReport{
		Schema: reportSchema,
		Result: status,
		Evidence: reportEvidence{
			ChainID:      options.ChainID,
			Height:       strconv.FormatInt(options.Height, 10),
			AppHash:      hex.EncodeToString(options.AppHash),
			SourceCommit: options.SourceCommit,
		},
		Multistore:   multistore,
		Stores:       reportStores,
		Census:       result,
		ReportSHA256: "",
	}

	unsealed, err := json.Marshal(report)
	if err != nil {
		return nil, false, fmt.Errorf("marshal unsealed census report: %w", err)
	}
	digest := sha256.Sum256(unsealed)
	report.ReportSHA256 = hex.EncodeToString(digest[:])
	sealed, err := json.Marshal(report)
	if err != nil {
		return nil, false, fmt.Errorf("marshal sealed census report: %w", err)
	}
	if len(sealed) > maxReportBytes {
		return nil, false, fmt.Errorf("census report exceeds %d bytes", maxReportBytes)
	}
	return sealed, result.Passed, nil
}

func validateAndRenderMultistore(
	options censusOptions,
	snapshot rootSnapshot,
) ([]reportMultistoreCommitment, error) {
	if snapshot.height != options.Height {
		return nil, fmt.Errorf("multistore height must equal report height %d: got %d", options.Height, snapshot.height)
	}
	if !bytes.Equal(snapshot.appHash, options.AppHash) {
		return nil, errors.New("multistore AppHash does not equal report AppHash")
	}
	if len(snapshot.stores) == 0 || len(snapshot.stores) > maxCommitInfoStores {
		return nil, fmt.Errorf("multistore commitment count must be between 1 and %d", maxCommitInfoStores)
	}

	names := make([]string, 0, len(snapshot.stores))
	for name := range snapshot.stores {
		if name == "" || !utf8.ValidString(name) || len(name) > 128 {
			return nil, fmt.Errorf("multistore contains invalid store name %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	commitInfo := storetypes.CommitInfo{Version: snapshot.height}
	commitInfo.StoreInfos = make([]storetypes.StoreInfo, 0, len(names))
	reportRows := make([]reportMultistoreCommitment, 0, len(names))
	for _, name := range names {
		store := snapshot.stores[name]
		if store.version < 0 {
			return nil, fmt.Errorf("multistore %q version is negative", name)
		}
		if len(store.rootHash) != sha256.Size {
			return nil, fmt.Errorf("multistore %q root hash must be exactly %d bytes", name, sha256.Size)
		}
		commitInfo.StoreInfos = append(commitInfo.StoreInfos, storetypes.StoreInfo{
			Name: name,
			CommitId: storetypes.CommitID{
				Version: store.version,
				Hash:    bytes.Clone(store.rootHash),
			},
		})
		reportRows = append(reportRows, reportMultistoreCommitment{
			Name:       name,
			RootSHA256: hex.EncodeToString(store.rootHash),
		})
	}
	if computed := commitInfo.Hash(); !bytes.Equal(computed, options.AppHash) {
		return nil, fmt.Errorf(
			"multistore commitments recompute AppHash %s, expected %s",
			hex.EncodeToString(computed),
			hex.EncodeToString(options.AppHash),
		)
	}
	return reportRows, nil
}

func validateCensusResultForReport(result censusResult, stores []storeEvidence) error {
	if result.resourceLimitExceeded {
		return errors.New("census exceeded a resource ceiling; report generation refused")
	}
	if result.Passed != (len(result.Findings) == 0) {
		return errors.New("census pass flag disagrees with findings")
	}
	if result.ClaimantRootComplete != result.Passed {
		return errors.New("claimant-root completeness disagrees with census result")
	}
	if result.Balances == nil || result.Keyspace == nil || result.Validators == nil ||
		result.Claims == nil || result.Unbondings == nil || result.DIDIndexes == nil ||
		result.ReverseIndexes == nil || result.Cooldowns == nil || result.SDKValidators == nil ||
		result.TierConfigs == nil || result.Findings == nil {
		return errors.New("census report collections must be non-nil")
	}
	if result.ModuleAddress != mustModuleAddress() || result.ModuleAddressHex != hex.EncodeToString(customModuleAddress) {
		return errors.New("census module address is not the deterministic zerone_staking module account")
	}

	balance, err := parseCanonicalAggregateAmount(result.BalanceUzrn, false)
	if err != nil {
		return fmt.Errorf("census B is invalid: %w", err)
	}
	delegations, err := parseCanonicalAggregateAmount(result.DelegationsUzrn, false)
	if err != nil {
		return fmt.Errorf("census D is invalid: %w", err)
	}
	pending, err := parseCanonicalAggregateAmount(result.PendingUnbondingsUzrn, false)
	if err != nil {
		return fmt.Errorf("census U is invalid: %w", err)
	}
	liabilities, err := parseCanonicalAggregateAmount(result.LiabilitiesUzrn, false)
	if err != nil {
		return fmt.Errorf("census liabilities are invalid: %w", err)
	}
	delta, err := parseCanonicalSignedAmount(result.DeltaUzrn)
	if err != nil {
		return fmt.Errorf("census delta is invalid: %w", err)
	}
	wantLiabilities := new(big.Int).Add(new(big.Int).Set(delegations), pending)
	if liabilities.Cmp(wantLiabilities) != 0 {
		return errors.New("census liabilities do not equal D + U")
	}
	wantDelta := new(big.Int).Sub(new(big.Int).Set(balance), liabilities)
	if delta.Cmp(wantDelta) != 0 {
		return errors.New("census delta does not equal B - (D + U)")
	}

	seenBalance := make(map[string]struct{}, len(result.Balances))
	moduleUzrn := new(big.Int)
	previousDenom := ""
	for index, row := range result.Balances {
		if err := sdkDenom(row.Denom); err != nil {
			return fmt.Errorf("census module balance %d denomination: %w", index, err)
		}
		if index > 0 && row.Denom <= previousDenom {
			return errors.New("census module balances are not in strict denomination order")
		}
		previousDenom = row.Denom
		if _, duplicate := seenBalance[row.Denom]; duplicate {
			return errors.New("census module balances contain a duplicate denomination")
		}
		seenBalance[row.Denom] = struct{}{}
		amount, amountErr := parseCanonicalAggregateAmount(row.Amount, false)
		if amountErr != nil {
			return fmt.Errorf("census module balance %q: %w", row.Denom, amountErr)
		}
		if amount.Sign() <= 0 {
			return fmt.Errorf("census module balance %q is not positive", row.Denom)
		}
		if row.Denom == claimDenom {
			moduleUzrn.Set(amount)
		}
	}
	if moduleUzrn.Cmp(balance) != 0 {
		return errors.New("census B does not equal the reported module-account uzrn balance")
	}

	if result.ClaimCount != uint64(len(result.Claims)) {
		return errors.New("census claim count does not equal the number of claimant records")
	}
	if result.ClaimantRoot != hashClaims(result.Claims) {
		return errors.New("census claimant root does not match the reported claimant records")
	}
	if !claimsStrictlySorted(result.Claims) {
		return errors.New("census claimant records are not in deterministic order")
	}
	if !findingsStrictlySorted(result.Findings) {
		return errors.New("census findings are not in deterministic order")
	}

	if len(result.Keyspace) != customModuleKeyspaceCount+1 {
		return errors.New("census must contain exactly nine custom-module keyspaces and the app IAVL sentinel class")
	}
	wantNames := [...]string{
		"validators",
		"delegations",
		"unbondings",
		"tier_configs",
		"params",
		"did_indexes",
		"unbonding_sequence",
		"redelegation_cooldowns",
		"validator_delegation_indexes",
		"app_iavl_init_sentinel",
	}
	var keyspaceLeaves, keyspaceBytes uint64
	for index, row := range result.Keyspace {
		wantPrefix := fmt.Sprintf("0x%02x", index+1)
		if index == customModuleKeyspaceCount {
			wantPrefix = "0x" + hex.EncodeToString([]byte(appIAVLInitSentinelKey))
			if row.LeafCount > 1 {
				return errors.New("census app IAVL sentinel class contains more than one leaf")
			}
		}
		if row.Prefix != wantPrefix || row.Name != wantNames[index] {
			return fmt.Errorf("census custom keyspace %d is not canonical", index)
		}
		if err := validateLowerHexDigest(row.Digest); err != nil {
			return fmt.Errorf("census custom keyspace %q digest: %w", row.Name, err)
		}
		if keyspaceLeaves > ^uint64(0)-row.LeafCount || keyspaceBytes > ^uint64(0)-row.InputBytes {
			return errors.New("census custom keyspace totals overflow uint64")
		}
		keyspaceLeaves += row.LeafCount
		keyspaceBytes += row.InputBytes
	}
	customStore := stores[0]
	if keyspaceLeaves > uint64(customStore.leafCount) || keyspaceBytes > customStore.inputBytes {
		return errors.New("census custom keyspace totals exceed the scanned custom store")
	}
	if result.Passed && (keyspaceLeaves != uint64(customStore.leafCount) || keyspaceBytes != customStore.inputBytes) {
		return errors.New("passing census custom keyspaces do not cover the complete custom store")
	}

	if result.Passed {
		if delta.Sign() != 0 {
			return errors.New("passing census does not satisfy B = D + U")
		}
		if err := validatePassingCensusResult(result); err != nil {
			return err
		}
	}
	return nil
}

func validatePassingCensusResult(result censusResult) error {
	if len(result.Validators) != int(result.Keyspace[0].LeafCount) ||
		len(result.Unbondings) != int(result.Keyspace[2].LeafCount) ||
		len(result.DIDIndexes) != int(result.Keyspace[5].LeafCount) ||
		len(result.Cooldowns) != int(result.Keyspace[7].LeafCount) ||
		len(result.ReverseIndexes) != int(result.Keyspace[8].LeafCount) {
		return errors.New("passing census record counts do not match custom keyspace counts")
	}
	if result.Keyspace[3].LeafCount != 4 || result.Keyspace[4].LeafCount != 1 ||
		result.Keyspace[6].LeafCount > 1 || len(result.TierConfigs) != 4 {
		return errors.New("passing census parameter, tier, or sequence inventory is incomplete")
	}
	if len(result.Unbondings) > 0 && result.Keyspace[6].LeafCount != 1 {
		return errors.New("passing census with unbondings has no sequence singleton")
	}
	validatorSet := make(map[string]struct{}, len(result.Validators))
	for _, validator := range result.Validators {
		if _, duplicate := validatorSet[validator.Operator]; duplicate {
			return errors.New("passing census contains duplicate validator operators")
		}
		validatorSet[validator.Operator] = struct{}{}
	}
	delegationClaims := 0
	pendingClaims := 0
	delegationTotal := new(big.Int)
	pendingTotal := new(big.Int)
	claimSelf := make(map[string]*big.Int)
	claimDelegated := make(map[string]*big.Int)
	delegationPairs := make(map[string]claimRecord)
	pendingByID := make(map[string]claimRecord)
	claimSources := make(map[string]struct{}, len(result.Claims))
	for index, claim := range result.Claims {
		if _, err := decodeCanonicalAddress(claim.Claimant, "zrn"); err != nil {
			return fmt.Errorf("passing census claim %d claimant: %w", index, err)
		}
		if _, err := decodeCanonicalAddress(claim.Validator, "zrn"); err != nil {
			return fmt.Errorf("passing census claim %d validator: %w", index, err)
		}
		if claim.Denom != claimDenom {
			return fmt.Errorf("passing census claim %d has denomination %q", index, claim.Denom)
		}
		if claim.SourceID == "" {
			return fmt.Errorf("passing census claim %d has an empty source ID", index)
		}
		if _, found := validatorSet[claim.Validator]; !found {
			return fmt.Errorf("passing census claim %d targets no reported custom validator", index)
		}
		amount, err := parseCanonicalAmount(claim.Amount, true)
		if err != nil {
			return fmt.Errorf("passing census claim %d amount: %w", index, err)
		}
		switch claim.SourceKind {
		case "delegation":
			delegationClaims++
			delegationTotal.Add(delegationTotal, amount)
			if claim.SourceID != claim.Claimant+"->"+claim.Validator {
				return fmt.Errorf("passing census delegation claim %d has a mismatched source ID", index)
			}
			pair := claim.Validator + "\x00" + claim.Claimant
			delegationPairs[pair] = claim
			if claim.Claimant == claim.Validator {
				addBigInt(claimSelf, claim.Validator, amount)
			} else {
				addBigInt(claimDelegated, claim.Validator, amount)
			}
		case "pending_unbonding":
			pendingClaims++
			pendingTotal.Add(pendingTotal, amount)
			pendingByID[claim.SourceID] = claim
		default:
			return fmt.Errorf("passing census claim %d has unknown source kind %q", index, claim.SourceKind)
		}
		sourceKey := claim.SourceKind + "\x00" + claim.SourceID
		if _, duplicate := claimSources[sourceKey]; duplicate {
			return fmt.Errorf("passing census claim %d reuses a source identity", index)
		}
		claimSources[sourceKey] = struct{}{}
	}
	if delegationClaims != int(result.Keyspace[1].LeafCount) {
		return errors.New("passing census delegation claims do not cover the delegation keyspace")
	}
	reportedDelegations, err := parseCanonicalAggregateAmount(result.DelegationsUzrn, false)
	if err != nil || delegationTotal.Cmp(reportedDelegations) != 0 {
		return errors.New("passing census D does not equal the rendered delegation claims")
	}
	reportedPending, err := parseCanonicalAggregateAmount(result.PendingUnbondingsUzrn, false)
	if err != nil || pendingTotal.Cmp(reportedPending) != 0 {
		return errors.New("passing census U does not equal the rendered pending-unbonding claims")
	}
	pendingRecords := 0
	seenSequences := make(map[uint64]struct{}, len(result.Unbondings))
	for index, entry := range result.Unbondings {
		if index > 0 && entry.ID <= result.Unbondings[index-1].ID {
			return errors.New("passing census unbondings are not in strict ID order")
		}
		if entry.ID == "" || entry.Sequence == 0 || entry.CompletesAtHeight <= entry.CreatedAtHeight {
			return fmt.Errorf("passing census unbonding %d has invalid identity, sequence, or heights", index)
		}
		if _, err := decodeCanonicalAddress(entry.Delegator, "zrn"); err != nil {
			return fmt.Errorf("passing census unbonding %d delegator: %w", index, err)
		}
		if _, err := decodeCanonicalAddress(entry.Validator, "zrn"); err != nil {
			return fmt.Errorf("passing census unbonding %d validator: %w", index, err)
		}
		if _, err := parseCanonicalAmount(entry.Amount, true); err != nil {
			return fmt.Errorf("passing census unbonding %d amount: %w", index, err)
		}
		if _, duplicate := seenSequences[entry.Sequence]; duplicate {
			return fmt.Errorf("passing census unbonding %d reuses global sequence %d", index, entry.Sequence)
		}
		seenSequences[entry.Sequence] = struct{}{}
		if entry.Status == "pending" {
			pendingRecords++
			if _, found := validatorSet[entry.Validator]; !found {
				return fmt.Errorf("passing census pending unbonding %d targets no custom validator", index)
			}
			claim, found := pendingByID[entry.ID]
			if !found || claim.Claimant != entry.Delegator || claim.Validator != entry.Validator || claim.Amount != entry.Amount {
				return fmt.Errorf("passing census pending unbonding %d does not match its claimant record", index)
			}
		} else if entry.Status != "completed" {
			return fmt.Errorf("passing census unbonding %d has invalid status %q", index, entry.Status)
		}
	}
	if pendingClaims != pendingRecords {
		return errors.New("passing census pending claims do not cover pending unbondings")
	}
	for index, validator := range result.Validators {
		if index > 0 && validator.Operator <= result.Validators[index-1].Operator {
			return errors.New("passing census validators are not in strict operator order")
		}
		address, err := decodeCanonicalAddress(validator.Operator, "zrn")
		if err != nil || validator.AddressHex != hex.EncodeToString(address) {
			return fmt.Errorf("passing census validator %d has invalid operator identity", index)
		}
		if !validator.AggregatesMatch || validator.LegacyConsensusPubkeyTrusted {
			return fmt.Errorf("passing census validator %d has invalid aggregate or legacy-key trust state", index)
		}
		if validator.LegacyConsensusPubkey == "" {
			return fmt.Errorf("passing census validator %d omits its untrusted legacy key evidence", index)
		}
		storedSelf, selfErr := parseCanonicalAmount(validator.StoredSelf, false)
		computedSelf, computedSelfErr := parseCanonicalAmount(validator.ComputedSelf, false)
		storedDelegated, delegatedErr := parseCanonicalAmount(validator.StoredDelegated, false)
		computedDelegated, computedDelegatedErr := parseCanonicalAmount(validator.ComputedDelegated, false)
		storedTotal, totalErr := parseCanonicalAmount(validator.StoredTotal, false)
		computedTotal, computedTotalErr := parseCanonicalAmount(validator.ComputedTotal, false)
		if selfErr != nil || computedSelfErr != nil || delegatedErr != nil || computedDelegatedErr != nil || totalErr != nil || computedTotalErr != nil ||
			storedSelf.Cmp(computedSelf) != 0 || storedDelegated.Cmp(computedDelegated) != 0 || storedTotal.Cmp(computedTotal) != 0 ||
			storedTotal.Cmp(new(big.Int).Add(new(big.Int).Set(storedSelf), storedDelegated)) != 0 {
			return fmt.Errorf("passing census validator %d aggregate values are inconsistent", index)
		}
		if computedSelf.Cmp(cloneOrZero(claimSelf[validator.Operator])) != 0 ||
			computedDelegated.Cmp(cloneOrZero(claimDelegated[validator.Operator])) != 0 {
			return fmt.Errorf("passing census validator %d computed aggregates do not match claimant records", index)
		}
		if validator.SDKLink != "absent" && validator.SDKLink != "linked" {
			return fmt.Errorf("passing census validator %d has invalid SDK link state", index)
		}
		if validator.SDKLink == "linked" && validator.SDKOperator == "" {
			return fmt.Errorf("passing census validator %d omits its linked SDK operator", index)
		}
	}
	for index, row := range result.ReverseIndexes {
		if index > 0 {
			previous := result.ReverseIndexes[index-1]
			if row.Validator < previous.Validator || row.Validator == previous.Validator && row.Delegator <= previous.Delegator {
				return errors.New("passing census reverse indexes are not in deterministic order")
			}
		}
		if _, err := decodeCanonicalAddress(row.Validator, "zrn"); err != nil {
			return fmt.Errorf("passing census reverse index %d validator: %w", index, err)
		}
		if _, err := decodeCanonicalAddress(row.Delegator, "zrn"); err != nil {
			return fmt.Errorf("passing census reverse index %d delegator: %w", index, err)
		}
		if _, found := delegationPairs[row.Validator+"\x00"+row.Delegator]; !found {
			return fmt.Errorf("passing census reverse index %d has no delegation claim", index)
		}
	}
	if len(result.ReverseIndexes) != len(delegationPairs) {
		return errors.New("passing census reverse indexes do not cover every delegation claim")
	}
	for index, row := range result.DIDIndexes {
		if index > 0 && row.DID <= result.DIDIndexes[index-1].DID {
			return errors.New("passing census DID indexes are not in deterministic order")
		}
		if row.DID == "" || !utf8.ValidString(row.DID) {
			return fmt.Errorf("passing census DID index %d has an invalid DID", index)
		}
		if _, found := validatorSet[row.Operator]; !found {
			return fmt.Errorf("passing census DID index %d targets no custom validator", index)
		}
	}
	for index, row := range result.Cooldowns {
		if index > 0 && row.Delegator <= result.Cooldowns[index-1].Delegator {
			return errors.New("passing census cooldowns are not in deterministic order")
		}
		if _, err := decodeCanonicalAddress(row.Delegator, "zrn"); err != nil || row.Height == 0 {
			return fmt.Errorf("passing census cooldown %d is invalid", index)
		}
	}
	sdkByAddress := make(map[string]sdkValidatorRecord, len(result.SDKValidators))
	for index, row := range result.SDKValidators {
		if index > 0 && row.Operator <= result.SDKValidators[index-1].Operator {
			return errors.New("passing census SDK validators are not in deterministic operator order")
		}
		address, err := decodeCanonicalAddress(row.Operator, "zrnvaloper")
		if err != nil || row.AddressHex != hex.EncodeToString(address) {
			return fmt.Errorf("passing census SDK validator %d has invalid operator identity", index)
		}
		if _, duplicate := sdkByAddress[row.AddressHex]; duplicate {
			return fmt.Errorf("passing census SDK validator %d duplicates address bytes", index)
		}
		sdkByAddress[row.AddressHex] = row
		if row.Status != sdkstakingtypes.Unbonded.String() &&
			row.Status != sdkstakingtypes.Unbonding.String() &&
			row.Status != sdkstakingtypes.Bonded.String() {
			return fmt.Errorf("passing census SDK validator %d has invalid status %q", index, row.Status)
		}
		if _, err := parseCanonicalAmount(row.Tokens, false); err != nil {
			return fmt.Errorf("passing census SDK validator %d tokens: %w", index, err)
		}
	}
	for index, validator := range result.Validators {
		sdkRow, exists := sdkByAddress[validator.AddressHex]
		if validator.SDKLink == "linked" {
			if !exists || sdkRow.Operator != validator.SDKOperator {
				return fmt.Errorf("passing census validator %d SDK link does not match the SDK inventory", index)
			}
		} else if exists {
			return fmt.Errorf("passing census validator %d marks an available SDK link absent", index)
		}
	}
	for index, tier := range result.TierConfigs {
		if tier.Tier != int32(index+1) || tier.Name == "" || tier.StoredDigest == "" || tier.ParamsDigest == "" ||
			!tier.Matches || tier.StoredDigest != tier.ParamsDigest {
			return fmt.Errorf("passing census tier reconciliation %d is incomplete", index)
		}
		if err := validateLowerHexDigest(tier.StoredDigest); err != nil {
			return fmt.Errorf("passing census tier %d stored digest: %w", index+1, err)
		}
		if err := validateLowerHexDigest(tier.ParamsDigest); err != nil {
			return fmt.Errorf("passing census tier %d params digest: %w", index+1, err)
		}
	}
	for _, balance := range result.Balances {
		if balance.Denom != claimDenom {
			return fmt.Errorf("passing census contains unexplained module denomination %q", balance.Denom)
		}
	}
	return nil
}

func parseCanonicalAggregateAmount(value string, signed bool) (*big.Int, error) {
	// At most 50,000 individually 256-bit claims are retained, so their exact
	// sum fits comfortably within 320 bits. Keep anomalous aggregate accounting
	// reportable without permitting unbounded integer parsing.
	if value == "" || len(value) > 98 {
		return nil, errors.New("aggregate amount is empty or exceeds the census integer text bound")
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok || parsed.String() != value || parsed.BitLen() > 320 || !signed && parsed.Sign() < 0 {
		return nil, errors.New("aggregate amount is not a canonical bounded base-10 integer")
	}
	return parsed, nil
}

func parseCanonicalSignedAmount(value string) (*big.Int, error) {
	return parseCanonicalAggregateAmount(value, true)
}

func sdkDenom(value string) error {
	return sdk.ValidateDenom(value)
}

func validateLowerHexDigest(value string) error {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return errors.New("must be exactly 64 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return err
	}
	return nil
}

func claimsStrictlySorted(claims []claimRecord) bool {
	for index := 1; index < len(claims); index++ {
		left, right := claims[index-1], claims[index]
		if !claimLess(left, right) {
			return false
		}
	}
	return true
}

func findingsStrictlySorted(findings []censusFinding) bool {
	for index := 1; index < len(findings); index++ {
		left, right := findings[index-1], findings[index]
		if left.Code > right.Code || left.Code == right.Code && left.Key > right.Key ||
			left.Code == right.Code && left.Key == right.Key && left.Detail >= right.Detail {
			return false
		}
	}
	return true
}

func estimateReportJSONUpperBound(
	options censusOptions,
	multistore []reportMultistoreCommitment,
	stores []reportStore,
	result censusResult,
) uint64 {
	// encoding/json can expand one input byte to at most six bytes (for example,
	// a control character rendered as \u00xx). A 1 KiB structural allowance per
	// row comfortably covers fixed field names, booleans, and uint64 decimals.
	records := len(result.Balances) + len(result.Keyspace) + len(result.Validators) +
		len(result.Claims) + len(result.Unbondings) + len(result.DIDIndexes) +
		len(result.ReverseIndexes) + len(result.Cooldowns) + len(result.SDKValidators) +
		len(result.TierConfigs) + len(result.Findings) + len(multistore) + len(stores)
	estimate := uint64(64 << 10)
	if uint64(records) > (^uint64(0)-estimate)/1024 {
		return ^uint64(0)
	}
	estimate += uint64(records) * 1024
	add := func(values ...string) {
		for _, value := range values {
			if uint64(len(value)) > (^uint64(0)-estimate)/6 {
				estimate = ^uint64(0)
				return
			}
			estimate += uint64(len(value)) * 6
		}
	}
	add(result.ModuleAddress, result.ModuleAddressHex, result.BalanceUzrn, result.DelegationsUzrn,
		result.PendingUnbondingsUzrn, result.LiabilitiesUzrn, result.DeltaUzrn, result.ClaimantRoot)
	add(reportSchema, options.ChainID, strconv.FormatInt(options.Height, 10), hex.EncodeToString(options.AppHash), options.SourceCommit)
	for _, row := range multistore {
		add(row.Name, row.RootSHA256)
	}
	for _, row := range stores {
		add(row.Name, row.Version, row.RootSHA256, row.LeafCount, row.InputBytes, row.LeavesSHA256)
	}
	for _, row := range result.Balances {
		add(row.Denom, row.Amount)
	}
	for _, row := range result.Keyspace {
		add(row.Prefix, row.Name, row.Digest)
	}
	for _, row := range result.Validators {
		add(row.Operator, row.AddressHex, row.LegacyConsensusPubkey, row.StoredSelf, row.ComputedSelf,
			row.StoredDelegated, row.ComputedDelegated, row.StoredTotal, row.ComputedTotal, row.SDKLink, row.SDKOperator)
	}
	for _, row := range result.Claims {
		add(row.SourceKind, row.SourceID, row.Claimant, row.Validator, row.Denom, row.Amount)
	}
	for _, row := range result.Unbondings {
		add(row.ID, row.Delegator, row.Validator, row.Amount, row.Status)
	}
	for _, row := range result.DIDIndexes {
		add(row.DID, row.Operator)
	}
	for _, row := range result.ReverseIndexes {
		add(row.Validator, row.Delegator)
	}
	for _, row := range result.Cooldowns {
		add(row.Delegator)
	}
	for _, row := range result.SDKValidators {
		add(row.Operator, row.AddressHex, row.Status, row.Tokens)
	}
	for _, row := range result.TierConfigs {
		add(row.Name, row.StoredDigest, row.ParamsDigest)
	}
	for _, row := range result.Findings {
		add(row.Code, row.Key, row.Detail)
	}
	return estimate
}
