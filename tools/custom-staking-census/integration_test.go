package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	customstakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

func TestExecuteCensusEndToEndBalancedRootmulti(t *testing.T) {
	leaves, operator, sdkOperator := integrationBalancedCensusFixture(t)
	db, expected := integrationCommitRootmultiFixture(t, leaves)
	before := integrationPhysicalDBDigest(t, db)
	options := integrationCensusOptions(expected)

	reportBytes, passed, err := executeCensus(db, options)
	require.NoError(t, err)
	require.True(t, passed)
	report := requireCanonicalCensusReport(t, reportBytes)
	require.Equal(t, "PASS", report.Result)
	require.Equal(t, before, integrationPhysicalDBDigest(t, db))

	// A second complete scan of the same committed bytes and external evidence
	// must produce the exact same sealed artifact.
	repeated, repeatedPass, err := executeCensus(db, options)
	require.NoError(t, err)
	require.True(t, repeatedPass)
	require.Equal(t, reportBytes, repeated)
	require.Equal(t, before, integrationPhysicalDBDigest(t, db))

	result := report.Census
	require.Empty(t, result.Findings)
	require.Equal(t, "35", result.BalanceUzrn)
	require.Equal(t, "30", result.DelegationsUzrn)
	require.Equal(t, "5", result.PendingUnbondingsUzrn)
	require.Equal(t, "35", result.LiabilitiesUzrn)
	require.Equal(t, "0", result.DeltaUzrn)
	require.Equal(t, uint64(3), result.ClaimCount)
	require.True(t, result.ClaimantRootComplete)

	require.Len(t, result.Unbondings, 2)
	completedFound := false
	for _, unbonding := range result.Unbondings {
		if unbonding.Status == "completed" {
			completedFound = true
			require.Equal(t, "7", unbonding.Amount)
			for _, claim := range result.Claims {
				require.NotEqual(t, unbonding.ID, claim.SourceID,
					"completed unbonding must not remain in U or the claimant root")
			}
		}
	}
	require.True(t, completedFound)

	require.Len(t, result.Keyspace, 9)
	wantKeyspaceCounts := []uint64{1, 2, 2, 4, 1, 1, 1, 1, 2}
	for index, keyspace := range result.Keyspace {
		require.Equal(t, "0x0"+string(rune('1'+index)), keyspace.Prefix)
		require.Equal(t, wantKeyspaceCounts[index], keyspace.LeafCount)
		require.NotEmpty(t, keyspace.Digest)
	}

	require.Len(t, result.Validators, 1)
	validator := result.Validators[0]
	require.Equal(t, operator, validator.Operator)
	require.Equal(t, strings.Repeat("11", 20), validator.AddressHex)
	require.Equal(t, "linked", validator.SDKLink)
	require.Equal(t, sdkOperator, validator.SDKOperator)
	require.True(t, validator.AggregatesMatch)
	require.Len(t, result.SDKValidators, 1)
	require.Equal(t, validator.AddressHex, result.SDKValidators[0].AddressHex)

	require.Len(t, report.Stores, 3)
	for index, name := range requiredStoreNames {
		store := report.Stores[index]
		require.Equal(t, name, store.Name)
		require.Equal(t, "1", store.Version)
		require.Len(t, store.RootSHA256, 64)
		require.Len(t, store.LeavesSHA256, 64)
	}
	require.Equal(t, "15", report.Stores[0].LeafCount)
	require.Equal(t, "1", report.Stores[1].LeafCount)
	require.Equal(t, "1", report.Stores[2].LeafCount)
}

func TestExecuteCensusEndToEndCommittedStateMutations(t *testing.T) {
	tests := []struct {
		name          string
		mutateLeaves  func(*testing.T, []censusFixture) []censusFixture
		mutateOptions func(*censusOptions)
		finding       string
		delta         string
		operational   string
	}{
		{
			name: "one unit deficit",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				replaceBankBalance(t, leaves, "34")
				return leaves
			},
			finding: "balance_liability_mismatch",
			delta:   "-1",
		},
		{
			name: "one unit surplus",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				replaceBankBalance(t, leaves, "36")
				return leaves
			},
			finding: "balance_liability_mismatch",
			delta:   "1",
		},
		{
			name: "validator aggregate mismatch",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				for index := range leaves {
					if leaves[index].store != customStakingStore ||
						len(leaves[index].key) == 0 ||
						leaves[index].key[0] != customstakingtypes.ValidatorKeyPrefix[0] {
						continue
					}
					var validator customstakingtypes.Validator
					require.NoError(t, json.Unmarshal(leaves[index].value, &validator))
					validator.DelegatedStake = "21"
					leaves[index].value = mustJSON(t, &validator)
					return leaves
				}
				t.Fatal("custom validator fixture not found")
				return nil
			},
			finding: "validator_delegated_aggregate_mismatch",
			delta:   "0",
		},
		{
			name: "unknown custom key prefix",
			mutateLeaves: func(_ *testing.T, leaves []censusFixture) []censusFixture {
				return append(leaves, censusFixture{
					store: customStakingStore,
					key:   []byte{0x0a, 0x01},
					value: []byte("unknown"),
				})
			},
			finding: "custom_key_unknown_prefix",
			delta:   "0",
		},
		{
			name: "missing reverse index",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				operator := canonicalAddress(t, 0x11)
				delegator := canonicalAddress(t, 0x22)
				remove := customstakingtypes.ValidatorDelegationIndexKey(operator, delegator)
				return integrationWithoutLeaf(leaves, customStakingStore, remove)
			},
			finding: "reverse_index_missing",
			delta:   "0",
		},
		{
			name: "stale reverse index",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				operator := canonicalAddress(t, 0x11)
				delegator := canonicalAddress(t, 0x22)
				original := customstakingtypes.ValidatorDelegationIndexKey(operator, delegator)
				for index := range leaves {
					if leaves[index].store == customStakingStore && bytes.Equal(leaves[index].key, original) {
						leaves[index].key = customstakingtypes.ValidatorDelegationIndexKey(
							operator,
							canonicalAddress(t, 0x33),
						)
						return leaves
					}
				}
				t.Fatal("reverse index fixture not found")
				return nil
			},
			finding: "reverse_index_orphan",
			delta:   "0",
		},
		{
			name: "malformed custom JSON",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				for index := range leaves {
					if leaves[index].store == customStakingStore &&
						len(leaves[index].key) > 0 &&
						leaves[index].key[0] == customstakingtypes.ValidatorKeyPrefix[0] {
						leaves[index].value = []byte(`{"operator_address":`)
						return leaves
					}
				}
				t.Fatal("custom validator fixture not found")
				return nil
			},
			finding: "validator_json_invalid",
			delta:   "0",
		},
		{
			name: "unexplained module denomination",
			mutateLeaves: func(t *testing.T, leaves []censusFixture) []censusFixture {
				return append(leaves, bankBalanceFixture(t, "ufoo", "1"))
			},
			finding: "module_balance_unexpected_denom",
			delta:   "0",
		},
		{
			name: "wrong external checkpoint",
			mutateOptions: func(options *censusOptions) {
				options.AppHash = bytes.Clone(options.AppHash)
				options.AppHash[0] ^= 0xff
			},
			operational: "root app hash mismatch",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			leaves, _, _ := integrationBalancedCensusFixture(t)
			if test.mutateLeaves != nil {
				leaves = test.mutateLeaves(t, leaves)
			}
			db, expected := integrationCommitRootmultiFixture(t, leaves)
			before := integrationPhysicalDBDigest(t, db)
			options := integrationCensusOptions(expected)
			if test.mutateOptions != nil {
				test.mutateOptions(&options)
			}

			reportBytes, passed, err := executeCensus(db, options)
			if test.operational != "" {
				require.ErrorContains(t, err, test.operational)
				require.False(t, passed)
				require.Nil(t, reportBytes)
				require.Equal(t, before, integrationPhysicalDBDigest(t, db))
				return
			}

			require.NoError(t, err)
			require.False(t, passed)
			report := requireCanonicalCensusReport(t, reportBytes)
			require.Equal(t, "FAIL", report.Result)
			require.False(t, report.Census.ClaimantRootComplete)
			require.Equal(t, test.delta, report.Census.DeltaUzrn)
			requireFindingCode(t, report.Census, test.finding)
			require.Equal(t, before, integrationPhysicalDBDigest(t, db))
		})
	}
}

func integrationBalancedCensusFixture(
	t *testing.T,
) ([]censusFixture, string, string) {
	t.Helper()
	leaves, operator, sdkOperator := balancedCensusFixture(t)
	delegator := canonicalAddress(t, 0x22)
	completed := &customstakingtypes.UnbondingEntry{
		Id:                delegator + "_" + operator + "_4_2",
		DelegatorAddress:  delegator,
		ValidatorAddress:  operator,
		Amount:            "7",
		CreatedAtHeight:   4,
		CompletesAtHeight: 11,
		Status:            "completed",
	}
	leaves = append(leaves, censusFixture{
		store: customStakingStore,
		key:   customstakingtypes.UnbondingKey(completed.Id),
		value: mustJSON(t, completed),
	})
	for index := range leaves {
		if leaves[index].store == customStakingStore &&
			bytes.Equal(leaves[index].key, customstakingtypes.UnbondingSeqKey) {
			leaves[index].value = uint64Bytes(2)
			return leaves, operator, sdkOperator
		}
	}
	t.Fatal("unbonding sequence fixture not found")
	return nil, "", ""
}

func integrationCommitRootmultiFixture(
	t *testing.T,
	leaves []censusFixture,
) (dbm.DB, expectedEvidence) {
	t.Helper()
	db := dbm.NewMemDB()
	root := rootmulti.NewStore(
		db,
		log.NewNopLogger(),
		metrics.NewNoOpMetrics(),
	)
	root.SetIAVLSyncPruning(true)
	root.SetIAVLDisableFastNode(true)

	storeKeys := make(map[string]*storetypes.KVStoreKey, len(requiredStoreNames))
	for _, name := range requiredStoreNames {
		storeKeys[name] = storetypes.NewKVStoreKey(name)
		root.MountStoreWithDB(storeKeys[name], storetypes.StoreTypeIAVL, nil)
	}
	require.NoError(t, root.LoadLatestVersion())
	for _, leaf := range leaves {
		key := storeKeys[leaf.store]
		if key == nil {
			t.Fatalf("fixture targets unmounted store %q", leaf.store)
		}
		root.GetKVStore(key).Set(bytes.Clone(leaf.key), bytes.Clone(leaf.value))
	}
	commitID := root.Commit()
	require.Equal(t, int64(1), commitID.Version)
	require.Len(t, commitID.Hash, sha256.Size)
	return db, expectedEvidence{Height: commitID.Version, AppHash: bytes.Clone(commitID.Hash)}
}

func integrationCensusOptions(expected expectedEvidence) censusOptions {
	return censusOptions{
		ChainID:      "zerone-integration-1",
		SourceCommit: strings.Repeat("1", 40),
		Height:       expected.Height,
		AppHash:      bytes.Clone(expected.AppHash),
	}
}

func integrationPhysicalDBDigest(t *testing.T, db physicalDB) []byte {
	t.Helper()
	iterator, err := db.Iterator(nil, nil)
	require.NoError(t, err)
	require.NotNil(t, iterator)
	hasher := sha256.New()
	for ; iterator.Valid(); iterator.Next() {
		hashLogicalLeaf(hasher, iterator.Key(), iterator.Value())
	}
	require.NoError(t, iterator.Error())
	require.NoError(t, iterator.Close())
	return hasher.Sum(nil)
}

func requireCanonicalCensusReport(
	t *testing.T,
	reportBytes []byte,
) sealedCensusReport {
	t.Helper()
	require.True(t, json.Valid(reportBytes))
	var report sealedCensusReport
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	remarshaled, err := json.Marshal(report)
	require.NoError(t, err)
	require.Equal(t, reportBytes, remarshaled, "report must use its canonical compact encoding")

	sealedDigest := report.ReportSHA256
	require.Len(t, sealedDigest, 64)
	_, err = hex.DecodeString(sealedDigest)
	require.NoError(t, err)
	report.ReportSHA256 = ""
	unsealed, err := json.Marshal(report)
	require.NoError(t, err)
	wantDigest := sha256.Sum256(unsealed)
	require.Equal(t, hex.EncodeToString(wantDigest[:]), sealedDigest)
	report.ReportSHA256 = sealedDigest
	return report
}

func integrationWithoutLeaf(
	leaves []censusFixture,
	store string,
	key []byte,
) []censusFixture {
	filtered := make([]censusFixture, 0, len(leaves))
	for _, leaf := range leaves {
		if leaf.store == store && bytes.Equal(leaf.key, key) {
			continue
		}
		filtered = append(filtered, leaf)
	}
	return filtered
}
