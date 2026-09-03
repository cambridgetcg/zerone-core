package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	customstakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

type censusFinding struct {
	Code   string `json:"code"`
	Key    string `json:"key"`
	Detail string `json:"detail"`
}

type denominationBalance struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type keyspaceClass struct {
	Prefix     string `json:"prefix"`
	Name       string `json:"name"`
	LeafCount  uint64 `json:"leaf_count"`
	InputBytes uint64 `json:"input_bytes"`
	Digest     string `json:"digest"`
}

type validatorReconciliation struct {
	Operator                     string `json:"operator"`
	AddressHex                   string `json:"address_hex"`
	LegacyConsensusPubkey        string `json:"legacy_consensus_pubkey"`
	LegacyConsensusPubkeyTrusted bool   `json:"legacy_consensus_pubkey_trusted"`
	StoredSelf                   string `json:"stored_self"`
	ComputedSelf                 string `json:"computed_self"`
	StoredDelegated              string `json:"stored_delegated"`
	ComputedDelegated            string `json:"computed_delegated"`
	StoredTotal                  string `json:"stored_total"`
	ComputedTotal                string `json:"computed_total"`
	AggregatesMatch              bool   `json:"aggregates_match"`
	SDKLink                      string `json:"sdk_link"`
	SDKOperator                  string `json:"sdk_operator,omitempty"`
}

type claimRecord struct {
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_claim_id"`
	Claimant   string `json:"claimant"`
	Validator  string `json:"validator"`
	Denom      string `json:"denom"`
	Amount     string `json:"amount"`
}

type unbondingRecord struct {
	ID                string `json:"id"`
	Delegator         string `json:"delegator"`
	Validator         string `json:"validator"`
	Amount            string `json:"amount"`
	CreatedAtHeight   uint64 `json:"created_at_height"`
	CompletesAtHeight uint64 `json:"completes_at_height"`
	Status            string `json:"status"`
	Sequence          uint64 `json:"sequence"`
}

type didIndexRecord struct {
	DID      string `json:"did"`
	Operator string `json:"operator"`
}

type reverseIndexRecord struct {
	Validator string `json:"validator"`
	Delegator string `json:"delegator"`
}

type cooldownRecord struct {
	Delegator string `json:"delegator"`
	Height    uint64 `json:"height"`
}

type sdkValidatorRecord struct {
	Operator   string `json:"operator"`
	AddressHex string `json:"address_hex"`
	Status     string `json:"status"`
	Jailed     bool   `json:"jailed"`
	Tokens     string `json:"tokens"`
}

type tierConfigReconciliation struct {
	Tier         int32  `json:"tier"`
	Name         string `json:"name"`
	StoredDigest string `json:"stored_digest,omitempty"`
	ParamsDigest string `json:"params_digest,omitempty"`
	Matches      bool   `json:"matches"`
}

// censusResult is deliberately a report-ready, map-free value. Every slice is
// sorted by finalize, so encoding it is deterministic.
type censusResult struct {
	resourceLimitExceeded bool
	Passed                bool                       `json:"-"`
	ModuleAddress         string                     `json:"module_address"`
	ModuleAddressHex      string                     `json:"module_address_hex"`
	Balances              []denominationBalance      `json:"module_balances"`
	BalanceUzrn           string                     `json:"balance_uzrn"`
	DelegationsUzrn       string                     `json:"delegations_uzrn"`
	PendingUnbondingsUzrn string                     `json:"pending_unbondings_uzrn"`
	LiabilitiesUzrn       string                     `json:"liabilities_uzrn"`
	DeltaUzrn             string                     `json:"delta_uzrn"`
	ClaimantRoot          string                     `json:"claimant_root"`
	ClaimantRootComplete  bool                       `json:"claimant_root_complete"`
	ClaimCount            uint64                     `json:"claim_count"`
	Keyspace              []keyspaceClass            `json:"custom_keyspace"`
	Validators            []validatorReconciliation  `json:"validators"`
	Claims                []claimRecord              `json:"claims"`
	Unbondings            []unbondingRecord          `json:"unbondings"`
	DIDIndexes            []didIndexRecord           `json:"did_indexes"`
	ReverseIndexes        []reverseIndexRecord       `json:"reverse_delegation_indexes"`
	Cooldowns             []cooldownRecord           `json:"redelegation_cooldowns"`
	SDKValidators         []sdkValidatorRecord       `json:"sdk_validators"`
	TierConfigs           []tierConfigReconciliation `json:"tier_configs"`
	Findings              []censusFinding            `json:"findings"`
}

type census struct {
	finalized bool

	validators    map[string]*parsedValidator
	delegations   []*parsedDelegation
	unbondings    []*parsedUnbonding
	tierConfigs   map[int32]*parsedTierConfig
	params        *customstakingtypes.Params
	didIndexes    map[string]*parsedDIDIndex
	reverse       map[string]*parsedReverseIndex
	cooldowns     map[string]*parsedCooldown
	sequence      *uint64
	sdkValidators map[string]*parsedSDKValidator
	balances      map[string]*big.Int

	classLeaves          [10][]leafCommitment
	findings             []censusFinding
	findingSet           map[string]struct{}
	findingLimitExceeded bool
	decodeResourceErr    error
}

const (
	customModuleKeyspaceCount = 9
	appIAVLInitSentinelKey    = "_iavl_init"

	maxCensusFindings   = 25_000
	maxSDKValidatorRows = 25_000
	maxModuleDenomRows  = 10_000
	maxFindingTextBytes = 512
)

func newCensus() *census {
	return &census{
		validators:    make(map[string]*parsedValidator),
		tierConfigs:   make(map[int32]*parsedTierConfig),
		didIndexes:    make(map[string]*parsedDIDIndex),
		reverse:       make(map[string]*parsedReverseIndex),
		cooldowns:     make(map[string]*parsedCooldown),
		sdkValidators: make(map[string]*parsedSDKValidator),
		balances:      make(map[string]*big.Int),
		findingSet:    make(map[string]struct{}),
	}
}

// ingest accepts one logical IAVL leaf from one of the three complete stores.
// State anomalies are accumulated as findings so the scan can finish. Caller
// contract violations and resource ceilings are returned as operational errors.
func (c *census) ingest(store string, key, value []byte) error {
	if c.finalized {
		return fmt.Errorf("ingest %s: census already finalized", store)
	}
	if key == nil || value == nil {
		return fmt.Errorf("ingest %s: nil key or value", store)
	}

	switch store {
	case customStakingStore:
		c.ingestCustomStaking(key, value)
	case bankStore:
		c.ingestBank(key, value)
	case sdkStakingStore:
		c.ingestSDKStaking(key, value)
	default:
		return fmt.Errorf("unsupported store %q", store)
	}
	if c.decodeResourceErr != nil {
		return c.decodeResourceErr
	}
	if c.findingLimitExceeded {
		return fmt.Errorf("census exceeds %d distinct findings", maxCensusFindings)
	}
	if len(c.sdkValidators) > maxSDKValidatorRows {
		return fmt.Errorf("census exceeds %d SDK validator records", maxSDKValidatorRows)
	}
	if len(c.balances) > maxModuleDenomRows {
		return fmt.Errorf("census exceeds %d module-account denominations", maxModuleDenomRows)
	}
	return nil
}

func (c *census) ingestCustomStaking(key, value []byte) {
	display := displayKey(customStakingStore, key)
	if len(key) == 0 {
		c.addFinding("custom_key_empty", display, "custom staking key is empty")
		return
	}
	if bytes.Equal(key, []byte(appIAVLInitSentinelKey)) {
		c.classLeaves[customModuleKeyspaceCount] = append(
			c.classLeaves[customModuleKeyspaceCount],
			newLeafCommitment(key, value),
		)
		if !bytes.Equal(value, []byte{0x01}) {
			c.addFinding(
				"app_iavl_init_sentinel_invalid",
				display,
				"app IAVL initialization sentinel must contain exactly 0x01",
			)
		}
		return
	}
	prefix := int(key[0])
	if prefix < 1 || prefix > customModuleKeyspaceCount {
		c.addFinding("custom_key_unknown_prefix", display, fmt.Sprintf("unknown custom staking prefix 0x%02x", key[0]))
		return
	}
	c.classLeaves[prefix-1] = append(c.classLeaves[prefix-1], newLeafCommitment(key, value))

	switch key[0] {
	case customstakingtypes.ValidatorKeyPrefix[0]:
		c.ingestValidator(display, key, value)
	case customstakingtypes.DelegationKeyPrefix[0]:
		c.ingestDelegation(display, key, value)
	case customstakingtypes.UnbondingKeyPrefix[0]:
		c.ingestUnbonding(display, key, value)
	case customstakingtypes.TierConfigKeyPrefix[0]:
		c.ingestTierConfig(display, key, value)
	case customstakingtypes.ParamsKey[0]:
		c.ingestParams(display, key, value)
	case customstakingtypes.ValidatorByDIDPrefix[0]:
		c.ingestDIDIndex(display, key, value)
	case customstakingtypes.UnbondingSeqKey[0]:
		c.ingestSequence(display, key, value)
	case customstakingtypes.RedelegationCooldownPrefix[0]:
		c.ingestCooldown(display, key, value)
	case customstakingtypes.ValidatorDelegationIndexPrefix[0]:
		c.ingestReverseIndex(display, key, value)
	}
}

func (c *census) ingestValidator(display string, key, raw []byte) {
	if len(key) < 2 || !utf8.Valid(key[1:]) {
		c.addFinding("validator_key_invalid", display, "validator key suffix must be non-empty UTF-8")
		return
	}
	keyOperator := string(key[1:])
	keyAddress, keyErr := decodeCanonicalAddress(keyOperator, "zrn")
	if keyErr != nil {
		c.addFinding("validator_key_address_invalid", display, keyErr.Error())
	}
	var validator customstakingtypes.Validator
	if err := decodeStrictJSON(raw, &validator); err != nil {
		c.addDecodeFinding("validator_json_invalid", display, err)
		return
	}
	payloadAddress, addressErr := decodeCanonicalAddress(validator.OperatorAddress, "zrn")
	if addressErr != nil {
		c.addFinding("validator_operator_invalid", display, addressErr.Error())
	}
	identityValid := keyErr == nil && addressErr == nil && keyOperator == validator.OperatorAddress && bytes.Equal(keyAddress, payloadAddress)
	if !identityValid {
		c.addFinding("validator_key_payload_mismatch", display, "key suffix does not exactly identify the payload operator")
	}

	amounts := validatorAmounts{}
	amounts.self = c.validAmount(display, "validator_self_amount_invalid", validator.SelfDelegation, false)
	amounts.delegated = c.validAmount(display, "validator_delegated_amount_invalid", validator.DelegatedStake, false)
	amounts.total = c.validAmount(display, "validator_total_amount_invalid", validator.TotalStake, false)
	if validator.Tier < customstakingtypes.TierApprentice || validator.Tier > customstakingtypes.TierGuardian {
		c.addFinding("validator_tier_invalid", display, fmt.Sprintf("tier %d is outside 1..4", validator.Tier))
	}
	if validator.ConsensusPubkey == "" {
		c.addFinding("validator_legacy_pubkey_empty", display, "legacy consensus_pubkey is empty (it is never treated as authority)")
	}
	if !utf8.ValidString(validator.Did) || len(validator.Did) > 128 {
		c.addFinding("validator_did_invalid", display, "DID must be valid UTF-8 and at most 128 bytes")
	}
	if validator.ReputationScore > customstakingtypes.BPSScale {
		c.addFinding("validator_reputation_invalid", display, "reputation exceeds 1,000,000 BPS")
	}
	if validator.CorrectVerifications > validator.TotalVerifications {
		c.addFinding("validator_verification_counts_invalid", display, "correct verifications exceed total verifications")
	}
	if validator.ContestedCount > validator.TotalVerifications {
		c.addFinding("validator_verification_counts_invalid", display, "contested verifications exceed total verifications")
	}
	if validator.ContestedVerificationsCorrect > validator.ContestedCount || validator.ContestedVerificationsCorrect > validator.CorrectVerifications {
		c.addFinding("validator_contested_counts_invalid", display, "correct contested verifications exceed a parent count")
	}
	if validator.CommissionBps > 10_000 {
		c.addFinding("validator_commission_invalid", display, "commission exceeds 10,000 BPS")
	}
	if validator.IsActive && validator.Jailed {
		c.addFinding("validator_status_invalid", display, "validator is both active and jailed")
	}
	if !validator.Jailed && (validator.JailReason != "" || validator.UnjailAfterBlock != 0) {
		c.addFinding("validator_jail_metadata_invalid", display, "unjail metadata is present while validator is not jailed")
	}
	if len(validator.Moniker) > 70 || len(validator.Website) > 140 || len(validator.Details) > 2000 {
		c.addFinding("validator_description_invalid", display, "validator description exceeds a message limit")
	}
	if !identityValid {
		return
	}
	if _, duplicate := c.validators[validator.OperatorAddress]; duplicate {
		c.addFinding("validator_duplicate", display, "duplicate canonical validator operator")
		return
	}
	c.validators[validator.OperatorAddress] = &parsedValidator{key: display, value: &validator, addressBytes: payloadAddress, amounts: amounts}
}

func (c *census) ingestDelegation(display string, key, raw []byte) {
	keyDelegator, keyValidator, keyErr := splitAddressPair(key[1:])
	if keyErr != nil {
		c.addFinding("delegation_key_invalid", display, keyErr.Error())
	}
	keyDelegatorBytes, keyDelegatorErr := decodeCanonicalAddress(keyDelegator, "zrn")
	keyValidatorBytes, keyValidatorErr := decodeCanonicalAddress(keyValidator, "zrn")
	if keyErr == nil && (keyDelegatorErr != nil || keyValidatorErr != nil) {
		c.addFinding("delegation_key_address_invalid", display, "delegation key contains a non-canonical zrn address")
	}

	var delegation customstakingtypes.Delegation
	if err := decodeStrictJSON(raw, &delegation); err != nil {
		c.addDecodeFinding("delegation_json_invalid", display, err)
		return
	}
	delegatorBytes, delegatorErr := decodeCanonicalAddress(delegation.DelegatorAddress, "zrn")
	validatorBytes, validatorErr := decodeCanonicalAddress(delegation.ValidatorAddress, "zrn")
	if delegatorErr != nil || validatorErr != nil {
		c.addFinding("delegation_payload_address_invalid", display, "delegation payload contains a non-canonical zrn address")
	}
	amount := c.validAmount(display, "delegation_amount_invalid", delegation.Amount, true)
	identityValid := keyErr == nil && keyDelegatorErr == nil && keyValidatorErr == nil && delegatorErr == nil && validatorErr == nil &&
		keyDelegator == delegation.DelegatorAddress && keyValidator == delegation.ValidatorAddress &&
		bytes.Equal(keyDelegatorBytes, delegatorBytes) && bytes.Equal(keyValidatorBytes, validatorBytes)
	if !identityValid {
		c.addFinding("delegation_key_payload_mismatch", display, "key address pair does not exactly identify the payload")
	}
	c.delegations = append(c.delegations, &parsedDelegation{
		key: display, value: &delegation, delegatorBytes: delegatorBytes, validatorBytes: validatorBytes,
		amount: amount, identityIsValid: identityValid,
	})
}

func (c *census) ingestUnbonding(display string, key, raw []byte) {
	keyValid := len(key) >= 2 && utf8.Valid(key[1:])
	if !keyValid {
		c.addFinding("unbonding_key_invalid", display, "unbonding key ID must be non-empty UTF-8")
	}
	keyID := ""
	if keyValid {
		keyID = string(key[1:])
	}
	var entry customstakingtypes.UnbondingEntry
	if err := decodeStrictJSON(raw, &entry); err != nil {
		c.addDecodeFinding("unbonding_json_invalid", display, err)
		return
	}
	delegatorBytes, delegatorErr := decodeCanonicalAddress(entry.DelegatorAddress, "zrn")
	validatorBytes, validatorErr := decodeCanonicalAddress(entry.ValidatorAddress, "zrn")
	if delegatorErr != nil || validatorErr != nil {
		c.addFinding("unbonding_address_invalid", display, "unbonding payload contains a non-canonical zrn address")
	}
	amount := c.validAmount(display, "unbonding_amount_invalid", entry.Amount, true)
	if entry.Status != "pending" && entry.Status != "completed" {
		c.addFinding("unbonding_status_invalid", display, fmt.Sprintf("status %q is neither pending nor completed", entry.Status))
	}
	if entry.CompletesAtHeight <= entry.CreatedAtHeight {
		c.addFinding("unbonding_height_invalid", display, "completion height must be greater than creation height")
	}
	sequence, sequenceErr := parseUnbondingSequence(&entry)
	if sequenceErr != nil {
		c.addFinding("unbonding_id_sequence_invalid", display, sequenceErr.Error())
	}
	identityValid := keyValid && keyID == entry.Id && delegatorErr == nil && validatorErr == nil && sequenceErr == nil
	if keyValid && keyID != entry.Id {
		c.addFinding("unbonding_key_payload_mismatch", display, "key ID does not exactly match payload ID")
	}
	c.unbondings = append(c.unbondings, &parsedUnbonding{
		key: display, value: &entry, delegatorBytes: delegatorBytes, validatorBytes: validatorBytes,
		amount: amount, sequence: sequence, identityIsValid: identityValid,
	})
}

func (c *census) ingestTierConfig(display string, key, raw []byte) {
	if len(key) != 2 || key[1] < byte(customstakingtypes.TierApprentice) || key[1] > byte(customstakingtypes.TierGuardian) {
		c.addFinding("tier_config_key_invalid", display, "tier config key must be exactly prefix plus tier 1..4")
		return
	}
	var config customstakingtypes.TierConfig
	if err := decodeStrictJSON(raw, &config); err != nil {
		c.addDecodeFinding("tier_config_json_invalid", display, err)
		return
	}
	if int32(key[1]) != int32(config.Tier) {
		c.addFinding("tier_config_key_payload_mismatch", display, "key tier does not match payload tier")
		return
	}
	c.validateTierConfig(display, &config)
	if _, duplicate := c.tierConfigs[int32(config.Tier)]; duplicate {
		c.addFinding("tier_config_duplicate", display, "duplicate tier config")
		return
	}
	c.tierConfigs[int32(config.Tier)] = &parsedTierConfig{key: display, value: &config}
}

func (c *census) ingestParams(display string, key, raw []byte) {
	if len(key) != 1 {
		c.addFinding("params_key_invalid", display, "params singleton key has trailing bytes")
		return
	}
	var params customstakingtypes.Params
	if err := decodeStrictJSON(raw, &params); err != nil {
		c.addDecodeFinding("params_json_invalid", display, err)
		return
	}
	if c.params != nil {
		c.addFinding("params_duplicate", display, "duplicate params singleton")
		return
	}
	c.validateParams(display, &params)
	c.params = &params
}

func (c *census) ingestDIDIndex(display string, key, raw []byte) {
	if len(key) < 2 || !utf8.Valid(key[1:]) || len(key[1:]) > 128 {
		c.addFinding("did_index_key_invalid", display, "DID index key must contain 1..128 UTF-8 bytes")
		return
	}
	did := string(key[1:])
	if !utf8.Valid(raw) {
		c.addFinding("did_index_value_invalid", display, "DID index operator is not UTF-8")
		return
	}
	operator := string(raw)
	if _, err := decodeCanonicalAddress(operator, "zrn"); err != nil {
		c.addFinding("did_index_operator_invalid", display, err.Error())
		return
	}
	if _, duplicate := c.didIndexes[did]; duplicate {
		c.addFinding("did_index_duplicate", display, "duplicate DID index")
		return
	}
	c.didIndexes[did] = &parsedDIDIndex{key: display, did: did, operator: operator}
}

func (c *census) ingestSequence(display string, key, raw []byte) {
	if len(key) != 1 {
		c.addFinding("unbonding_sequence_key_invalid", display, "sequence singleton key has trailing bytes")
		return
	}
	if len(raw) != 8 {
		c.addFinding("unbonding_sequence_value_invalid", display, "sequence must be exactly 8-byte big-endian uint64")
		return
	}
	sequence := binary.BigEndian.Uint64(raw)
	if sequence == 0 {
		c.addFinding("unbonding_sequence_zero", display, "a stored sequence must be positive")
	}
	if c.sequence != nil {
		c.addFinding("unbonding_sequence_duplicate", display, "duplicate sequence singleton")
		return
	}
	c.sequence = &sequence
}

func (c *census) ingestCooldown(display string, key, raw []byte) {
	if len(key) < 2 || !utf8.Valid(key[1:]) {
		c.addFinding("cooldown_key_invalid", display, "cooldown key address must be non-empty UTF-8")
		return
	}
	delegator := string(key[1:])
	if _, err := decodeCanonicalAddress(delegator, "zrn"); err != nil {
		c.addFinding("cooldown_address_invalid", display, err.Error())
		return
	}
	if len(raw) != 8 {
		c.addFinding("cooldown_value_invalid", display, "cooldown height must be exactly 8-byte big-endian uint64")
		return
	}
	height := binary.BigEndian.Uint64(raw)
	if height == 0 {
		c.addFinding("cooldown_height_zero", display, "a stored cooldown height must be positive")
	}
	if _, duplicate := c.cooldowns[delegator]; duplicate {
		c.addFinding("cooldown_duplicate", display, "duplicate canonical cooldown address")
		return
	}
	c.cooldowns[delegator] = &parsedCooldown{key: display, delegator: delegator, height: height}
}

func (c *census) ingestReverseIndex(display string, key, raw []byte) {
	validator, delegator, err := splitAddressPair(key[1:])
	if err != nil {
		c.addFinding("reverse_index_key_invalid", display, err.Error())
		return
	}
	if _, err := decodeCanonicalAddress(validator, "zrn"); err != nil {
		c.addFinding("reverse_index_validator_invalid", display, err.Error())
		return
	}
	if _, err := decodeCanonicalAddress(delegator, "zrn"); err != nil {
		c.addFinding("reverse_index_delegator_invalid", display, err.Error())
		return
	}
	if !bytes.Equal(raw, []byte{0x01}) {
		c.addFinding("reverse_index_value_invalid", display, "reverse index value must be exactly 0x01")
	}
	id := validator + "\x00" + delegator
	if _, duplicate := c.reverse[id]; duplicate {
		c.addFinding("reverse_index_duplicate", display, "duplicate canonical reverse index")
		return
	}
	c.reverse[id] = &parsedReverseIndex{key: display, validator: validator, delegator: delegator}
}

func (c *census) ingestBank(key, raw []byte) {
	if !bytes.HasPrefix(key, banktypes.BalancesPrefix.Bytes()) {
		return
	}
	display := displayKey(bankStore, key)
	address, denom, err := parseBankBalanceKey(key)
	if err != nil {
		c.addFinding("bank_balance_key_invalid", display, err.Error())
		return
	}
	amount, err := parseBankBalanceValue(denom, raw)
	if err != nil {
		c.addFinding("bank_balance_value_invalid", display, err.Error())
		return
	}
	if !bytes.Equal(address, customModuleAddress) {
		return
	}
	if existing, duplicate := c.balances[denom]; duplicate {
		c.addFinding("module_balance_duplicate", display, "duplicate module balance denomination")
		existing.Add(existing, amount)
	} else {
		c.balances[denom] = amount
	}
	if denom != claimDenom {
		c.addFinding("module_balance_unexpected_denom", display, fmt.Sprintf("custom staking module holds unexplained denomination %q", denom))
	}
}

func (c *census) ingestSDKStaking(key, raw []byte) {
	if len(key) == 0 || key[0] != sdkstakingtypes.ValidatorsKey[0] {
		return
	}
	display := displayKey(sdkStakingStore, key)
	validator, err := parseSDKValidator(key, raw)
	if err != nil {
		c.addDecodeFinding("sdk_validator_primary_invalid", display, err)
		return
	}
	id := hex.EncodeToString(validator.addressBytes)
	if existing, duplicate := c.sdkValidators[id]; duplicate {
		c.addFinding("sdk_validator_address_ambiguous", display, fmt.Sprintf("address bytes also appear at %s", existing.key))
		return
	}
	c.sdkValidators[id] = validator
}

func (c *census) validAmount(key, code, value string, positive bool) *big.Int {
	amount, err := parseCanonicalAmount(value, positive)
	if err != nil {
		c.addFinding(code, key, err.Error())
		return nil
	}
	return amount
}

func (c *census) validateParams(key string, params *customstakingtypes.Params) {
	hasNilTierConfig := false
	for _, config := range params.TierConfigs {
		if config == nil {
			hasNilTierConfig = true
			break
		}
	}
	// Params.Validate currently dereferences every tier config. Adversarial
	// state must become a finding, never an operational panic.
	if !hasNilTierConfig {
		if err := params.Validate(); err != nil {
			c.addFinding("params_semantics_invalid", key, err.Error())
		}
	}
	if _, err := parseCanonicalAmount(params.VirtualStake, true); err != nil {
		c.addFinding("params_virtual_stake_invalid", key, err.Error())
	}
	if _, err := parseCanonicalAmount(params.MinSelfDelegation, true); err != nil {
		c.addFinding("params_min_self_invalid", key, err.Error())
	}
	if _, err := parseCanonicalAmount(params.MinStakeForVerification, true); err != nil {
		c.addFinding("params_min_verification_stake_invalid", key, err.Error())
	}
	seen := make(map[int32]struct{})
	for index, config := range params.TierConfigs {
		if config == nil {
			c.addFinding("params_tier_config_nil", key, fmt.Sprintf("tier config at index %d is nil", index))
			continue
		}
		c.validateTierConfig(key, config)
		tier := int32(config.Tier)
		if _, duplicate := seen[tier]; duplicate {
			c.addFinding("params_tier_config_duplicate", key, fmt.Sprintf("tier %d appears more than once", tier))
		}
		seen[tier] = struct{}{}
	}
	for tier := int32(customstakingtypes.TierApprentice); tier <= int32(customstakingtypes.TierGuardian); tier++ {
		if _, ok := seen[tier]; !ok {
			c.addFinding("params_tier_config_missing", key, fmt.Sprintf("tier %d is absent", tier))
		}
	}
}

func (c *census) validateTierConfig(key string, config *customstakingtypes.TierConfig) {
	if config.Tier < customstakingtypes.TierApprentice || config.Tier > customstakingtypes.TierGuardian {
		c.addFinding("tier_config_tier_invalid", key, fmt.Sprintf("tier %d is outside 1..4", config.Tier))
	}
	if _, err := parseCanonicalAmount(config.MinStake, false); err != nil {
		c.addFinding("tier_config_min_stake_invalid", key, err.Error())
	}
	if config.Name == "" || !utf8.ValidString(config.Name) {
		c.addFinding("tier_config_name_invalid", key, "tier name must be non-empty UTF-8")
	}
	if config.MinReputation > customstakingtypes.BPSScale || config.MinAccuracy > customstakingtypes.BPSScale {
		c.addFinding("tier_config_bps_invalid", key, "reputation or accuracy exceeds 1,000,000 BPS")
	}
	if config.MaxSlashCount < -1 {
		c.addFinding("tier_config_slash_count_invalid", key, "max slash count is below -1")
	}
	if config.RewardMultiplierBps == 0 || config.SelectionWeightBps == 0 || config.SlashMultiplierBps == 0 || config.SlashMultiplierBps > 10_000 {
		c.addFinding("tier_config_multiplier_invalid", key, "reward/selection multipliers must be positive and slash multiplier must be 1..10000")
	}
	if config.ContestedVerificationMultiplier == 0 {
		c.addFinding("tier_config_contested_multiplier_invalid", key, "contested verification multiplier must be positive")
	}
	seenCategories := make(map[string]struct{})
	for _, category := range config.AllowedCategories {
		if category == "" || !utf8.ValidString(category) {
			c.addFinding("tier_config_category_invalid", key, "allowed categories must be non-empty UTF-8")
			continue
		}
		if _, duplicate := seenCategories[category]; duplicate {
			c.addFinding("tier_config_category_duplicate", key, fmt.Sprintf("category %q appears more than once", category))
		}
		seenCategories[category] = struct{}{}
	}
}

func (c *census) addFinding(code, key, detail string) {
	code = boundedFindingText(code)
	key = boundedFindingText(key)
	detail = boundedFindingText(detail)
	id := code + "\x00" + key + "\x00" + detail
	if _, exists := c.findingSet[id]; exists {
		return
	}
	if len(c.findings) >= maxCensusFindings {
		c.findingLimitExceeded = true
		return
	}
	c.findingSet[id] = struct{}{}
	c.findings = append(c.findings, censusFinding{Code: code, Key: key, Detail: detail})
}

func (c *census) addDecodeFinding(code, key string, err error) {
	if errors.Is(err, errDecodeResourceLimit) {
		if c.decodeResourceErr == nil {
			c.decodeResourceErr = fmt.Errorf("census decode limit at %s (%s): %w", key, code, err)
		}
		return
	}
	c.addFinding(code, key, err.Error())
}

func boundedFindingText(value string) string {
	if len(value) <= maxFindingTextBytes && utf8.ValidString(value) {
		return value
	}
	digest := sha256.Sum256([]byte(value))
	prefixLength := 64
	if len(value) < prefixLength {
		prefixLength = len(value)
	}
	return fmt.Sprintf(
		"sha256:%s bytes:%d prefix_hex:%s",
		hex.EncodeToString(digest[:]),
		len(value),
		hex.EncodeToString([]byte(value[:prefixLength])),
	)
}

func (c *census) finalize() censusResult {
	if c.finalized {
		panic("census finalized more than once")
	}
	c.finalized = true

	delegationTotal := new(big.Int)
	pendingTotal := new(big.Int)
	computedSelf := make(map[string]*big.Int)
	computedDelegated := make(map[string]*big.Int)
	claims := make([]claimRecord, 0, len(c.delegations)+len(c.unbondings))
	validDelegations := make(map[string]*parsedDelegation)

	for _, delegation := range c.delegations {
		if delegation.amount != nil {
			delegationTotal.Add(delegationTotal, delegation.amount)
		}
		if !delegation.identityIsValid || delegation.amount == nil {
			continue
		}
		value := delegation.value
		id := value.ValidatorAddress + "\x00" + value.DelegatorAddress
		if existing, duplicate := validDelegations[id]; duplicate {
			c.addFinding("delegation_identity_duplicate", delegation.key, fmt.Sprintf("canonical claim also appears at %s", existing.key))
			continue
		}
		validDelegations[id] = delegation
		if _, found := c.validators[value.ValidatorAddress]; !found {
			c.addFinding("delegation_validator_missing", delegation.key, "delegation targets no custom validator primary")
		} else if bytes.Equal(delegation.delegatorBytes, delegation.validatorBytes) {
			addBigInt(computedSelf, value.ValidatorAddress, delegation.amount)
		} else {
			addBigInt(computedDelegated, value.ValidatorAddress, delegation.amount)
		}
		claims = append(claims, claimRecord{
			SourceKind: "delegation", SourceID: value.DelegatorAddress + "->" + value.ValidatorAddress,
			Claimant: value.DelegatorAddress, Validator: value.ValidatorAddress, Denom: claimDenom, Amount: delegation.amount.String(),
		})
	}

	maxSequence := uint64(0)
	sequenceOwners := make(map[uint64][]string)
	unbondingRecords := make([]unbondingRecord, 0, len(c.unbondings))
	for _, unbonding := range c.unbondings {
		entry := unbonding.value
		unbondingRecords = append(unbondingRecords, unbondingRecord{
			ID: entry.Id, Delegator: entry.DelegatorAddress, Validator: entry.ValidatorAddress, Amount: entry.Amount,
			CreatedAtHeight: entry.CreatedAtHeight, CompletesAtHeight: entry.CompletesAtHeight, Status: entry.Status, Sequence: unbonding.sequence,
		})
		if unbonding.sequence > maxSequence {
			maxSequence = unbonding.sequence
		}
		if unbonding.sequence > 0 {
			sequenceOwners[unbonding.sequence] = append(sequenceOwners[unbonding.sequence], unbonding.key)
		}
		if entry.Status != "pending" || unbonding.amount == nil {
			continue
		}
		pendingTotal.Add(pendingTotal, unbonding.amount)
		if !unbonding.identityIsValid {
			continue
		}
		if _, found := c.validators[entry.ValidatorAddress]; !found {
			c.addFinding("unbonding_validator_missing", unbonding.key, "pending unbonding references no custom validator primary")
		}
		claims = append(claims, claimRecord{
			SourceKind: "pending_unbonding", SourceID: entry.Id, Claimant: entry.DelegatorAddress,
			Validator: entry.ValidatorAddress, Denom: claimDenom, Amount: unbonding.amount.String(),
		})
	}
	for sequence, owners := range sequenceOwners {
		if len(owners) < 2 {
			continue
		}
		sort.Strings(owners)
		c.addFinding(
			"unbonding_sequence_duplicate",
			customStakingStore+"/07",
			fmt.Sprintf("sequence %d is reused by %s", sequence, strings.Join(owners, ",")),
		)
	}
	if len(c.unbondings) > 0 && c.sequence == nil {
		c.addFinding("unbonding_sequence_missing", customStakingStore+"/07", "unbonding entries exist but the global sequence is absent")
	}
	if c.sequence != nil && maxSequence > *c.sequence {
		c.addFinding("unbonding_sequence_behind", customStakingStore+"/07", fmt.Sprintf("stored sequence %d is below observed entry sequence %d", *c.sequence, maxSequence))
	}

	c.reconcileIndexes(validDelegations)
	tiers := c.reconcileTierConfigs()
	validators := c.reconcileValidators(computedSelf, computedDelegated)

	balance := cloneOrZero(c.balances[claimDenom])
	liabilities := new(big.Int).Add(new(big.Int).Set(delegationTotal), pendingTotal)
	delta := new(big.Int).Sub(new(big.Int).Set(balance), liabilities)
	if delta.Sign() != 0 {
		c.addFinding("balance_liability_mismatch", bankStore+"/module/"+claimDenom,
			fmt.Sprintf("B=%s, D=%s, U=%s, B-(D+U)=%s", balance, delegationTotal, pendingTotal, delta))
	}

	sortClaims(claims)
	claimantRoot := hashClaims(claims)
	sort.Slice(unbondingRecords, func(i, j int) bool { return unbondingRecords[i].ID < unbondingRecords[j].ID })
	sort.Slice(c.findings, func(i, j int) bool {
		if c.findings[i].Code != c.findings[j].Code {
			return c.findings[i].Code < c.findings[j].Code
		}
		if c.findings[i].Key != c.findings[j].Key {
			return c.findings[i].Key < c.findings[j].Key
		}
		return c.findings[i].Detail < c.findings[j].Detail
	})

	result := censusResult{
		resourceLimitExceeded: c.findingLimitExceeded || c.decodeResourceErr != nil,
		Passed:                len(c.findings) == 0 && !c.findingLimitExceeded && c.decodeResourceErr == nil,
		ModuleAddress:         mustModuleAddress(),
		ModuleAddressHex:      hex.EncodeToString(customModuleAddress),
		Balances:              c.sortedBalances(), BalanceUzrn: balance.String(), DelegationsUzrn: delegationTotal.String(),
		PendingUnbondingsUzrn: pendingTotal.String(), LiabilitiesUzrn: liabilities.String(), DeltaUzrn: delta.String(),
		ClaimantRoot: claimantRoot, ClaimCount: uint64(len(claims)), Keyspace: c.keyspaceResult(),
		Validators: validators, Claims: claims, Unbondings: unbondingRecords, DIDIndexes: c.sortedDIDIndexes(),
		ReverseIndexes: c.sortedReverseIndexes(), Cooldowns: c.sortedCooldowns(), SDKValidators: c.sortedSDKValidators(),
		TierConfigs: tiers, Findings: append([]censusFinding{}, c.findings...),
	}
	result.ClaimantRootComplete = result.Passed
	return result
}

func (c *census) reconcileIndexes(delegations map[string]*parsedDelegation) {
	expectedDIDs := make(map[string][]string)
	for operator, validator := range c.validators {
		if validator.value.Did != "" {
			expectedDIDs[validator.value.Did] = append(expectedDIDs[validator.value.Did], operator)
		}
	}
	for did, operators := range expectedDIDs {
		sort.Strings(operators)
		if len(operators) != 1 {
			c.addFinding("validator_did_duplicate", customStakingStore+"/06"+hex.EncodeToString([]byte(did)), "DID is claimed by "+strings.Join(operators, ","))
			continue
		}
		index, found := c.didIndexes[did]
		if !found {
			c.addFinding("did_index_missing", customStakingStore+"/06"+hex.EncodeToString([]byte(did)), "validator DID has no index")
		} else if index.operator != operators[0] {
			c.addFinding("did_index_mismatch", index.key, fmt.Sprintf("index points to %s, expected %s", index.operator, operators[0]))
		}
	}
	for did, index := range c.didIndexes {
		operators := expectedDIDs[did]
		if len(operators) == 0 {
			c.addFinding("did_index_orphan", index.key, "index has no validator declaring this DID")
		}
	}

	for id, delegation := range delegations {
		if _, found := c.reverse[id]; !found {
			c.addFinding("reverse_index_missing", delegation.key, "delegation has no exact validator-to-delegator reverse index")
		}
	}
	for id, index := range c.reverse {
		if _, found := delegations[id]; !found {
			c.addFinding("reverse_index_orphan", index.key, "reverse index has no exact delegation primary")
		}
	}
}

func (c *census) reconcileTierConfigs() []tierConfigReconciliation {
	if c.params == nil {
		c.addFinding("params_missing", customStakingStore+"/05", "custom staking params singleton is absent")
	}
	paramsByTier := make(map[int32]*customstakingtypes.TierConfig)
	if c.params != nil {
		for _, config := range c.params.TierConfigs {
			if config != nil {
				paramsByTier[int32(config.Tier)] = config
			}
		}
	}
	result := make([]tierConfigReconciliation, 0, 4)
	for tier := int32(customstakingtypes.TierApprentice); tier <= int32(customstakingtypes.TierGuardian); tier++ {
		raw := c.tierConfigs[tier]
		param := paramsByTier[tier]
		row := tierConfigReconciliation{Tier: tier}
		if raw != nil {
			row.Name = raw.value.Name
			row.StoredDigest = jsonDigest(raw.value)
		} else {
			c.addFinding("tier_config_store_missing", fmt.Sprintf("%s/04%02x", customStakingStore, tier), "raw tier config is absent")
		}
		if param != nil {
			if row.Name == "" {
				row.Name = param.Name
			}
			row.ParamsDigest = jsonDigest(param)
		}
		row.Matches = raw != nil && param != nil && row.StoredDigest == row.ParamsDigest
		if raw != nil && param != nil && !row.Matches {
			c.addFinding("tier_config_params_mismatch", raw.key, fmt.Sprintf("raw tier %d differs from params tier config", tier))
		}
		result = append(result, row)
	}
	return result
}

func (c *census) reconcileValidators(self, delegated map[string]*big.Int) []validatorReconciliation {
	operators := make([]string, 0, len(c.validators))
	for operator := range c.validators {
		operators = append(operators, operator)
	}
	sort.Strings(operators)
	result := make([]validatorReconciliation, 0, len(operators))
	for _, operator := range operators {
		validator := c.validators[operator]
		computedSelf := cloneOrZero(self[operator])
		computedDelegated := cloneOrZero(delegated[operator])
		computedTotal := new(big.Int).Add(new(big.Int).Set(computedSelf), computedDelegated)
		selfMatches := validator.amounts.self != nil && validator.amounts.self.Cmp(computedSelf) == 0
		delegatedMatches := validator.amounts.delegated != nil && validator.amounts.delegated.Cmp(computedDelegated) == 0
		totalMatches := validator.amounts.total != nil && validator.amounts.total.Cmp(computedTotal) == 0
		if !selfMatches {
			c.addFinding("validator_self_aggregate_mismatch", validator.key, fmt.Sprintf("stored=%s computed=%s", validator.value.SelfDelegation, computedSelf))
		}
		if !delegatedMatches {
			c.addFinding("validator_delegated_aggregate_mismatch", validator.key, fmt.Sprintf("stored=%s computed=%s", validator.value.DelegatedStake, computedDelegated))
		}
		if !totalMatches {
			c.addFinding("validator_total_aggregate_mismatch", validator.key, fmt.Sprintf("stored=%s computed=%s", validator.value.TotalStake, computedTotal))
		}
		row := validatorReconciliation{
			Operator: operator, AddressHex: hex.EncodeToString(validator.addressBytes),
			LegacyConsensusPubkey: validator.value.ConsensusPubkey, LegacyConsensusPubkeyTrusted: false,
			StoredSelf:   validator.value.SelfDelegation,
			ComputedSelf: computedSelf.String(), StoredDelegated: validator.value.DelegatedStake,
			ComputedDelegated: computedDelegated.String(), StoredTotal: validator.value.TotalStake,
			ComputedTotal: computedTotal.String(), AggregatesMatch: selfMatches && delegatedMatches && totalMatches,
			SDKLink: "absent",
		}
		if sdkValidator := c.sdkValidators[row.AddressHex]; sdkValidator != nil {
			row.SDKLink = "linked"
			row.SDKOperator = sdkValidator.operator
		}
		result = append(result, row)
	}
	return result
}

func mustModuleAddress() string {
	address, err := bech32.ConvertAndEncode("zrn", customModuleAddress)
	if err != nil {
		panic(fmt.Sprintf("encode fixed custom staking module address: %v", err))
	}
	return address
}

func (c *census) keyspaceResult() []keyspaceClass {
	names := [...]string{
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
	result := make([]keyspaceClass, 0, len(names))
	for index, name := range names {
		leaves := append([]leafCommitment(nil), c.classLeaves[index]...)
		sort.Slice(leaves, func(i, j int) bool {
			if comparison := bytes.Compare(leaves[i].key, leaves[j].key); comparison != 0 {
				return comparison < 0
			}
			return bytes.Compare(leaves[i].digest[:], leaves[j].digest[:]) < 0
		})
		h := sha256.New()
		writeHashField(h, []byte("zerone/custom-staking-census/key-class/v1"))
		prefix := fmt.Sprintf("0x%02x", index+1)
		if index == customModuleKeyspaceCount {
			prefix = "0x" + hex.EncodeToString([]byte(appIAVLInitSentinelKey))
			writeHashField(h, []byte(appIAVLInitSentinelKey))
		} else {
			_, _ = h.Write([]byte{byte(index + 1)})
		}
		var count [8]byte
		binary.BigEndian.PutUint64(count[:], uint64(len(leaves)))
		_, _ = h.Write(count[:])
		var inputBytes uint64
		for _, leaf := range leaves {
			_, _ = h.Write(leaf.digest[:])
			inputBytes += leaf.inputSize
		}
		result = append(result, keyspaceClass{
			Prefix: prefix, Name: name, LeafCount: uint64(len(leaves)),
			InputBytes: inputBytes, Digest: hex.EncodeToString(h.Sum(nil)),
		})
	}
	return result
}

func (c *census) sortedBalances() []denominationBalance {
	denoms := make([]string, 0, len(c.balances))
	for denom := range c.balances {
		denoms = append(denoms, denom)
	}
	sort.Strings(denoms)
	result := make([]denominationBalance, 0, len(denoms))
	for _, denom := range denoms {
		result = append(result, denominationBalance{Denom: denom, Amount: c.balances[denom].String()})
	}
	return result
}

func (c *census) sortedDIDIndexes() []didIndexRecord {
	keys := make([]string, 0, len(c.didIndexes))
	for did := range c.didIndexes {
		keys = append(keys, did)
	}
	sort.Strings(keys)
	result := make([]didIndexRecord, 0, len(keys))
	for _, did := range keys {
		result = append(result, didIndexRecord{DID: did, Operator: c.didIndexes[did].operator})
	}
	return result
}

func (c *census) sortedReverseIndexes() []reverseIndexRecord {
	result := make([]reverseIndexRecord, 0, len(c.reverse))
	for _, index := range c.reverse {
		result = append(result, reverseIndexRecord{Validator: index.validator, Delegator: index.delegator})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Validator != result[j].Validator {
			return result[i].Validator < result[j].Validator
		}
		return result[i].Delegator < result[j].Delegator
	})
	return result
}

func (c *census) sortedCooldowns() []cooldownRecord {
	result := make([]cooldownRecord, 0, len(c.cooldowns))
	for _, cooldown := range c.cooldowns {
		result = append(result, cooldownRecord{Delegator: cooldown.delegator, Height: cooldown.height})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Delegator < result[j].Delegator })
	return result
}

func (c *census) sortedSDKValidators() []sdkValidatorRecord {
	result := make([]sdkValidatorRecord, 0, len(c.sdkValidators))
	for addressHex, validator := range c.sdkValidators {
		result = append(result, sdkValidatorRecord{
			Operator: validator.operator, AddressHex: addressHex, Status: validator.status,
			Jailed: validator.jailed, Tokens: validator.tokens,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Operator < result[j].Operator })
	return result
}

func addBigInt(target map[string]*big.Int, key string, value *big.Int) {
	if target[key] == nil {
		target[key] = new(big.Int)
	}
	target[key].Add(target[key], value)
}

func cloneOrZero(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}

func jsonDigest(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func sortClaims(claims []claimRecord) {
	sort.Slice(claims, func(i, j int) bool {
		return claimLess(claims[i], claims[j])
	})
}

func claimLess(left, right claimRecord) bool {
	if left.SourceKind != right.SourceKind {
		return left.SourceKind < right.SourceKind
	}
	if left.SourceID != right.SourceID {
		return left.SourceID < right.SourceID
	}
	if left.Claimant != right.Claimant {
		return left.Claimant < right.Claimant
	}
	if left.Validator != right.Validator {
		return left.Validator < right.Validator
	}
	if left.Denom != right.Denom {
		return left.Denom < right.Denom
	}
	return left.Amount < right.Amount
}

func hashClaims(claims []claimRecord) string {
	h := sha256.New()
	writeHashField(h, []byte("zerone/custom-staking-claimants/v1"))
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(claims)))
	_, _ = h.Write(count[:])
	for _, claim := range claims {
		writeHashField(h, []byte(claim.SourceKind))
		writeHashField(h, []byte(claim.SourceID))
		writeHashField(h, []byte(claim.Claimant))
		writeHashField(h, []byte(claim.Validator))
		writeHashField(h, []byte(claim.Denom))
		writeHashField(h, []byte(claim.Amount))
	}
	return hex.EncodeToString(h.Sum(nil))
}
