package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	sdkstakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	customstakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

type censusFixture struct {
	store string
	key   []byte
	value []byte
}

func TestCensusBalancedStatePassesAndLinksSDKByAddressBytes(t *testing.T) {
	leaves, operator, sdkOperator := balancedCensusFixture(t)
	result := runCensusFixture(t, leaves)

	require.True(t, result.Passed)
	require.True(t, result.ClaimantRootComplete)
	require.Empty(t, result.Findings)
	require.Equal(t, mustModuleAddress(), result.ModuleAddress)
	require.Equal(t, fmt.Sprintf("%x", customModuleAddress), result.ModuleAddressHex)
	require.Equal(t, "35", result.BalanceUzrn)
	require.Equal(t, "30", result.DelegationsUzrn)
	require.Equal(t, "5", result.PendingUnbondingsUzrn)
	require.Equal(t, "35", result.LiabilitiesUzrn)
	require.Equal(t, "0", result.DeltaUzrn)
	require.Equal(t, uint64(3), result.ClaimCount)
	require.Len(t, result.Keyspace, 10)
	require.Equal(t, "0x01", result.Keyspace[0].Prefix)
	require.Equal(t, "0x09", result.Keyspace[8].Prefix)
	require.Equal(t, "0x5f6961766c5f696e6974", result.Keyspace[9].Prefix)
	require.Equal(t, "app_iavl_init_sentinel", result.Keyspace[9].Name)
	require.Equal(t, uint64(1), result.Keyspace[9].LeafCount)
	require.NotEmpty(t, result.ClaimantRoot)

	require.Len(t, result.Validators, 1)
	validator := result.Validators[0]
	require.Equal(t, operator, validator.Operator)
	require.Equal(t, "10", validator.ComputedSelf)
	require.Equal(t, "20", validator.ComputedDelegated)
	require.Equal(t, "30", validator.ComputedTotal)
	require.True(t, validator.AggregatesMatch)
	require.Equal(t, "linked", validator.SDKLink)
	require.Equal(t, sdkOperator, validator.SDKOperator)
	require.Equal(t, "legacy-untrusted", validator.LegacyConsensusPubkey)
	require.False(t, validator.LegacyConsensusPubkeyTrusted)

	require.Len(t, result.Balances, 1)
	require.Equal(t, denominationBalance{Denom: "uzrn", Amount: "35"}, result.Balances[0])
	require.Len(t, result.DIDIndexes, 1)
	require.Len(t, result.ReverseIndexes, 2)
	require.Len(t, result.Cooldowns, 1)
	require.Len(t, result.SDKValidators, 1)
	for _, tier := range result.TierConfigs {
		require.True(t, tier.Matches)
	}
}

func TestCensusOutputIsIndependentOfLeafIngestionOrder(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	forward := runCensusFixture(t, leaves)

	reversedLeaves := slices.Clone(leaves)
	slices.Reverse(reversedLeaves)
	reversed := runCensusFixture(t, reversedLeaves)

	shuffledLeaves := slices.Clone(leaves)
	rand.New(rand.NewSource(111)).Shuffle(len(shuffledLeaves), func(i, j int) {
		shuffledLeaves[i], shuffledLeaves[j] = shuffledLeaves[j], shuffledLeaves[i]
	})
	shuffled := runCensusFixture(t, shuffledLeaves)

	forwardJSON, err := json.Marshal(forward)
	require.NoError(t, err)
	reversedJSON, err := json.Marshal(reversed)
	require.NoError(t, err)
	shuffledJSON, err := json.Marshal(shuffled)
	require.NoError(t, err)
	require.Equal(t, string(forwardJSON), string(reversedJSON))
	require.Equal(t, string(forwardJSON), string(shuffledJSON))
}

func TestCensusDetectsSelfUndelegationAggregateFaultEvenWhenCustodyBalances(t *testing.T) {
	leaves, operator, _ := balancedCensusFixture(t)

	// The known legacy self-undelegation path reduces DelegatedStake instead of
	// SelfDelegation. Custody can still satisfy B=D+U while the validator's
	// redundant aggregates disagree with the individual self claim.
	replaceCustomValidator(t, leaves, &customstakingtypes.Validator{
		OperatorAddress: operator,
		ConsensusPubkey: "legacy-untrusted",
		Did:             "did:zrn:test-validator",
		Moniker:         "census fixture",
		Tier:            customstakingtypes.TierApprentice,
		SelfDelegation:  "10",
		DelegatedStake:  "15",
		TotalStake:      "25",
		ReputationScore: 500_000,
		IsActive:        true,
	})
	result := runCensusFixture(t, leaves)

	require.False(t, result.Passed)
	require.Equal(t, "0", result.DeltaUzrn, "custody equality alone must not pass")
	require.False(t, result.Validators[0].AggregatesMatch)
	requireFindingCode(t, result, "validator_delegated_aggregate_mismatch")
	requireFindingCode(t, result, "validator_total_aggregate_mismatch")
}

func TestCensusRejectsImpossibleContestedVerificationCount(t *testing.T) {
	leaves, operator, _ := balancedCensusFixture(t)
	replaceCustomValidator(t, leaves, &customstakingtypes.Validator{
		OperatorAddress:    operator,
		ConsensusPubkey:    "legacy-untrusted",
		Did:                "did:zrn:test-validator",
		Moniker:            "census fixture",
		Tier:               customstakingtypes.TierApprentice,
		SelfDelegation:     "10",
		DelegatedStake:     "20",
		TotalStake:         "30",
		ReputationScore:    500_000,
		TotalVerifications: 1,
		ContestedCount:     2,
		IsActive:           true,
	})
	result := runCensusFixture(t, leaves)
	require.False(t, result.Passed)
	requireFindingCode(t, result, "validator_verification_counts_invalid")
}

func TestCensusDetectsSlashUndercollateralizationAndSurplus(t *testing.T) {
	t.Run("slash style deficit", func(t *testing.T) {
		leaves, _, _ := balancedCensusFixture(t)
		replaceBankBalance(t, leaves, "15")
		result := runCensusFixture(t, leaves)

		require.False(t, result.Passed)
		require.Equal(t, "-20", result.DeltaUzrn)
		requireFindingCode(t, result, "balance_liability_mismatch")
	})

	t.Run("unassigned surplus", func(t *testing.T) {
		leaves, _, _ := balancedCensusFixture(t)
		replaceBankBalance(t, leaves, "40")
		result := runCensusFixture(t, leaves)

		require.False(t, result.Passed)
		require.Equal(t, "5", result.DeltaUzrn)
		requireFindingCode(t, result, "balance_liability_mismatch")
	})
}

func TestCensusFailsClosedOnCorruptJSONUnknownPrefixAndUnexpectedDenom(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	for index := range leaves {
		if leaves[index].store == customStakingStore && len(leaves[index].key) > 0 && leaves[index].key[0] == customstakingtypes.ValidatorKeyPrefix[0] {
			leaves[index].value = []byte(`{"operator_address":"unknown-field-test","surprise":true}`)
			break
		}
	}
	leaves = append(leaves,
		censusFixture{store: customStakingStore, key: []byte{0x0a, 0x01}, value: []byte("x")},
		bankBalanceFixture(t, "ufoo", "7"),
	)
	result := runCensusFixture(t, leaves)

	require.False(t, result.Passed)
	require.False(t, result.ClaimantRootComplete)
	requireFindingCode(t, result, "validator_json_invalid")
	requireFindingCode(t, result, "custom_key_unknown_prefix")
	requireFindingCode(t, result, "module_balance_unexpected_denom")
	require.Equal(t, []denominationBalance{{Denom: "ufoo", Amount: "7"}, {Denom: "uzrn", Amount: "35"}}, result.Balances)
}

func TestCensusAcceptsOnlyTheExactOptionalAppIAVLInitSentinel(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	withoutSentinel := slices.DeleteFunc(slices.Clone(leaves), func(leaf censusFixture) bool {
		return leaf.store == customStakingStore && bytes.Equal(leaf.key, []byte(appIAVLInitSentinelKey))
	})
	withoutResult := runCensusFixture(t, withoutSentinel)
	require.True(t, withoutResult.Passed)
	require.Zero(t, withoutResult.Keyspace[9].LeafCount)

	for _, mutation := range []censusFixture{
		{store: customStakingStore, key: []byte(appIAVLInitSentinelKey), value: []byte{0x02}},
		{store: customStakingStore, key: []byte(appIAVLInitSentinelKey + "_extra"), value: []byte{0x01}},
	} {
		mutated := append(slices.Clone(withoutSentinel), mutation)
		result := runCensusFixture(t, mutated)
		require.False(t, result.Passed)
	}
	requireFindingCode(
		t,
		runCensusFixture(t, append(slices.Clone(withoutSentinel), censusFixture{
			store: customStakingStore,
			key:   []byte(appIAVLInitSentinelKey),
			value: []byte{0x02},
		})),
		"app_iavl_init_sentinel_invalid",
	)
	requireFindingCode(
		t,
		runCensusFixture(t, append(slices.Clone(withoutSentinel), censusFixture{
			store: customStakingStore,
			key:   []byte(appIAVLInitSentinelKey + "_extra"),
			value: []byte{0x01},
		})),
		"custom_key_unknown_prefix",
	)
}

func TestCensusReconcilesDIDReverseTierAndSequenceIndexesExactly(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	filtered := leaves[:0]
	for _, leaf := range leaves {
		if leaf.store != customStakingStore || len(leaf.key) == 0 {
			filtered = append(filtered, leaf)
			continue
		}
		switch leaf.key[0] {
		case customstakingtypes.ValidatorByDIDPrefix[0]:
			continue
		case customstakingtypes.ValidatorDelegationIndexPrefix[0]:
			// Retain only one of two reverse indexes.
			if bytes.Contains(leaf.key, []byte(canonicalAddress(t, 0x22))) {
				continue
			}
		case customstakingtypes.UnbondingSeqKey[0]:
			leaf.value = make([]byte, 8) // behind the observed entry sequence
		case customstakingtypes.TierConfigKeyPrefix[0]:
			if len(leaf.key) == 2 && leaf.key[1] == byte(customstakingtypes.TierGuardian) {
				continue
			}
		}
		filtered = append(filtered, leaf)
	}
	result := runCensusFixture(t, filtered)

	require.False(t, result.Passed)
	requireFindingCode(t, result, "did_index_missing")
	requireFindingCode(t, result, "reverse_index_missing")
	requireFindingCode(t, result, "unbonding_sequence_zero")
	requireFindingCode(t, result, "unbonding_sequence_behind")
	requireFindingCode(t, result, "tier_config_store_missing")
}

func TestCensusRejectsReusedGlobalUnbondingSequence(t *testing.T) {
	leaves, operator, _ := balancedCensusFixture(t)
	delegator := canonicalAddress(t, 0x22)
	completed := &customstakingtypes.UnbondingEntry{
		Id: delegator + "_" + operator + "_4_1", DelegatorAddress: delegator, ValidatorAddress: operator,
		Amount: "1", CreatedAtHeight: 4, CompletesAtHeight: 11, Status: "completed",
	}
	leaves = append(leaves, censusFixture{
		store: customStakingStore,
		key:   customstakingtypes.UnbondingKey(completed.Id),
		value: mustJSON(t, completed),
	})

	result := runCensusFixture(t, leaves)
	require.False(t, result.Passed)
	requireFindingCode(t, result, "unbonding_sequence_duplicate")
}

func TestCensusCountsParseablePendingLiabilityDespiteMalformedKey(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	for index := range leaves {
		if leaves[index].store == customStakingStore && len(leaves[index].key) > 0 &&
			leaves[index].key[0] == customstakingtypes.UnbondingKeyPrefix[0] {
			leaves[index].key = []byte{customstakingtypes.UnbondingKeyPrefix[0], 0xff}
			break
		}
	}

	result := runCensusFixture(t, leaves)
	require.False(t, result.Passed)
	require.Equal(t, "5", result.PendingUnbondingsUzrn)
	require.Equal(t, "0", result.DeltaUzrn)
	requireFindingCode(t, result, "unbonding_key_invalid")
}

func TestCensusClassifiesAbsentSDKValidatorWithoutTreatingLegacyKeyAsAuthority(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	filtered := leaves[:0]
	for _, leaf := range leaves {
		if leaf.store != sdkStakingStore {
			filtered = append(filtered, leaf)
		}
	}
	result := runCensusFixture(t, filtered)

	require.True(t, result.Passed)
	require.Equal(t, "absent", result.Validators[0].SDKLink)
	require.Empty(t, result.Validators[0].SDKOperator)
}

func TestCensusNilParamsTierConfigBecomesFindingWithoutPanic(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	for index := range leaves {
		if leaves[index].store == customStakingStore && bytes.Equal(leaves[index].key, customstakingtypes.ParamsKey) {
			params := customstakingtypes.DefaultParams()
			params.TierConfigs[2] = nil
			leaves[index].value = mustJSON(t, params)
			break
		}
	}

	require.NotPanics(t, func() {
		result := runCensusFixture(t, leaves)
		require.False(t, result.Passed)
		requireFindingCode(t, result, "params_tier_config_nil")
		requireFindingCode(t, result, "params_tier_config_missing")
	})
}

func TestCensusEmptyCollectionsMarshalAsArrays(t *testing.T) {
	result := newCensus().finalize()
	require.NotNil(t, result.Balances)
	require.NotNil(t, result.Validators)
	require.NotNil(t, result.Claims)
	require.NotNil(t, result.Unbondings)
	require.NotNil(t, result.DIDIndexes)
	require.NotNil(t, result.ReverseIndexes)
	require.NotNil(t, result.Cooldowns)
	require.NotNil(t, result.SDKValidators)
	require.NotNil(t, result.TierConfigs)
	require.NotNil(t, result.Findings)

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `:null`)
}

func TestCensusBoundsAdversarialFindingEvidence(t *testing.T) {
	collector := newCensus()
	largeKey := bytes.Repeat([]byte{0xff}, maxLogicalKeyBytes)
	largeDetail := strings.Repeat("x", maxFindingTextBytes*4)

	reference := displayKey(bankStore, largeKey)
	require.Less(t, len(reference), maxFindingTextBytes)
	require.Contains(t, reference, "sha256:")
	collector.addFinding("test", reference, largeDetail)
	require.Len(t, collector.findings, 1)
	require.LessOrEqual(t, len(collector.findings[0].Key), maxFindingTextBytes)
	require.LessOrEqual(t, len(collector.findings[0].Detail), maxFindingTextBytes)
	require.Contains(t, collector.findings[0].Detail, "sha256:")
}

func TestCensusBoundsFindingsAddedDuringFinalize(t *testing.T) {
	collector := newCensus()
	for index := 0; index <= maxCensusFindings; index++ {
		collector.addFinding("test", fmt.Sprintf("key-%d", index), "detail")
	}
	require.True(t, collector.findingLimitExceeded)
	require.Len(t, collector.findings, maxCensusFindings)

	result := collector.finalize()
	require.True(t, result.resourceLimitExceeded)
	require.False(t, result.Passed)
	require.Len(t, result.Findings, maxCensusFindings)
}

func balancedCensusFixture(t *testing.T) ([]censusFixture, string, string) {
	t.Helper()
	operator := canonicalAddress(t, 0x11)
	delegator := canonicalAddress(t, 0x22)
	sdkOperator := canonicalValoper(t, 0x11)
	did := "did:zrn:test-validator"

	validator := &customstakingtypes.Validator{
		OperatorAddress: operator,
		ConsensusPubkey: "legacy-untrusted",
		Did:             did,
		Moniker:         "census fixture",
		Tier:            customstakingtypes.TierApprentice,
		SelfDelegation:  "10",
		DelegatedStake:  "20",
		TotalStake:      "30",
		ReputationScore: 500_000,
		IsActive:        true,
	}
	selfDelegation := &customstakingtypes.Delegation{DelegatorAddress: operator, ValidatorAddress: operator, Amount: "10", CreatedAtBlock: 1}
	otherDelegation := &customstakingtypes.Delegation{DelegatorAddress: delegator, ValidatorAddress: operator, Amount: "20", CreatedAtBlock: 2}
	unbonding := &customstakingtypes.UnbondingEntry{
		Id: delegator + "_" + operator + "_3_1", DelegatorAddress: delegator, ValidatorAddress: operator,
		Amount: "5", CreatedAtHeight: 3, CompletesAtHeight: 10, Status: "pending",
	}
	params := customstakingtypes.DefaultParams()

	leaves := []censusFixture{
		{store: customStakingStore, key: []byte(appIAVLInitSentinelKey), value: []byte{0x01}},
		{store: customStakingStore, key: customstakingtypes.ValidatorKey(operator), value: mustJSON(t, validator)},
		{store: customStakingStore, key: customstakingtypes.DelegationKey(operator, operator), value: mustJSON(t, selfDelegation)},
		{store: customStakingStore, key: customstakingtypes.DelegationKey(delegator, operator), value: mustJSON(t, otherDelegation)},
		{store: customStakingStore, key: customstakingtypes.UnbondingKey(unbonding.Id), value: mustJSON(t, unbonding)},
		{store: customStakingStore, key: customstakingtypes.ParamsKey, value: mustJSON(t, params)},
		{store: customStakingStore, key: customstakingtypes.ValidatorByDIDKey(did), value: []byte(operator)},
		{store: customStakingStore, key: customstakingtypes.UnbondingSeqKey, value: uint64Bytes(1)},
		{store: customStakingStore, key: customstakingtypes.RedelegationCooldownKey(delegator), value: uint64Bytes(2)},
		{store: customStakingStore, key: customstakingtypes.ValidatorDelegationIndexKey(operator, operator), value: []byte{0x01}},
		{store: customStakingStore, key: customstakingtypes.ValidatorDelegationIndexKey(operator, delegator), value: []byte{0x01}},
		bankBalanceFixture(t, "uzrn", "35"),
		sdkValidatorFixture(t, sdkOperator, 100),
	}
	for _, config := range params.TierConfigs {
		leaves = append(leaves, censusFixture{
			store: customStakingStore, key: customstakingtypes.TierConfigKey(config.Tier), value: mustJSON(t, config),
		})
	}
	return leaves, operator, sdkOperator
}

func runCensusFixture(t *testing.T, leaves []censusFixture) censusResult {
	t.Helper()
	collector := newCensus()
	for _, leaf := range leaves {
		require.NoError(t, collector.ingest(leaf.store, leaf.key, leaf.value))
	}
	return collector.finalize()
}

func canonicalAddress(t *testing.T, fill byte) string {
	t.Helper()
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{fill}, 20))
	require.NoError(t, err)
	return address
}

func canonicalValoper(t *testing.T, fill byte) string {
	t.Helper()
	address, err := bech32.ConvertAndEncode("zrnvaloper", bytes.Repeat([]byte{fill}, 20))
	require.NoError(t, err)
	return address
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func uint64Bytes(value uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)
	return encoded
}

func bankBalanceFixture(t *testing.T, denom, amountText string) censusFixture {
	t.Helper()
	pair := collections.Join(sdk.AccAddress(customModuleAddress), denom)
	key := make([]byte, len(banktypes.BalancesPrefix.Bytes())+bankBalanceKeyCodec.Size(pair))
	copy(key, banktypes.BalancesPrefix.Bytes())
	written, err := bankBalanceKeyCodec.Encode(key[len(banktypes.BalancesPrefix.Bytes()):], pair)
	require.NoError(t, err)
	require.Equal(t, len(key)-len(banktypes.BalancesPrefix.Bytes()), written)
	amount, ok := sdkmath.NewIntFromString(amountText)
	require.True(t, ok)
	value, err := sdk.IntValue.Encode(amount)
	require.NoError(t, err)
	return censusFixture{store: bankStore, key: key, value: value}
}

func sdkValidatorFixture(t *testing.T, operator string, tokens int64) censusFixture {
	t.Helper()
	_, address, err := bech32.DecodeAndConvert(operator)
	require.NoError(t, err)
	validator := &sdkstakingtypes.Validator{
		OperatorAddress:   operator,
		Status:            sdkstakingtypes.Bonded,
		Tokens:            sdkmath.NewInt(tokens),
		DelegatorShares:   sdkmath.LegacyNewDec(tokens),
		MinSelfDelegation: sdkmath.OneInt(),
	}
	value, err := validator.Marshal()
	require.NoError(t, err)
	key := append([]byte{sdkstakingtypes.ValidatorsKey[0], byte(len(address))}, address...)
	return censusFixture{store: sdkStakingStore, key: key, value: value}
}

func replaceCustomValidator(t *testing.T, leaves []censusFixture, validator *customstakingtypes.Validator) {
	t.Helper()
	for index := range leaves {
		if leaves[index].store == customStakingStore && len(leaves[index].key) > 0 && leaves[index].key[0] == customstakingtypes.ValidatorKeyPrefix[0] {
			leaves[index].value = mustJSON(t, validator)
			return
		}
	}
	t.Fatal("validator fixture not found")
}

func replaceBankBalance(t *testing.T, leaves []censusFixture, amount string) {
	t.Helper()
	replacement := bankBalanceFixture(t, "uzrn", amount)
	for index := range leaves {
		if leaves[index].store == bankStore && bytes.Equal(leaves[index].key, replacement.key) {
			leaves[index] = replacement
			return
		}
	}
	t.Fatal("bank balance fixture not found")
}

func requireFindingCode(t *testing.T, result censusResult, code string) {
	t.Helper()
	for _, finding := range result.Findings {
		if finding.Code == code {
			return
		}
	}
	available := make([]string, 0, len(result.Findings))
	for _, finding := range result.Findings {
		available = append(available, finding.Code)
	}
	t.Fatalf("finding %q not present; got %s", code, strings.Join(available, ", "))
}
