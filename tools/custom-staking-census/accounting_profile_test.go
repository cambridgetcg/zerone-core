package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func accountingFixtureResult(t *testing.T, leaves []censusFixture, enabled bool) censusResult {
	t.Helper()
	c := newCensus()
	c.accountingV2 = enabled
	for _, leaf := range leaves {
		require.NoError(t, c.ingest(leaf.store, leaf.key, leaf.value))
	}
	return c.finalize()
}

func TestAccountingProfileRequiresExplicitSelectionAndExactMarker(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	marker := censusFixture{store: customStakingStore, key: []byte{0x0a}, value: []byte{1}}
	withMarker := append(append([]censusFixture(nil), leaves...), marker)
	legacy := accountingFixtureResult(t, withMarker, false)
	require.False(t, legacy.Passed)
	require.False(t, legacy.ClaimantRootComplete)
	missing := accountingFixtureResult(t, leaves, true)
	require.False(t, missing.Passed)
	current := accountingFixtureResult(t, withMarker, true)
	require.True(t, current.Passed, "%+v", current.Findings)
	require.Equal(t, legacy.ClaimantRoot, current.ClaimantRoot)
	require.Len(t, current.Keyspace, 11)
	require.Equal(t, "accounting_safety_marker", current.Keyspace[10].Name)
	require.Equal(t, "0x0a", current.Keyspace[10].Prefix)
	require.Equal(t, uint64(1), current.Keyspace[10].LeafCount)
	for _, bad := range []censusFixture{
		{store: customStakingStore, key: []byte{0x0a}, value: []byte{0}},
		{store: customStakingStore, key: []byte{0x0a}, value: []byte{1, 0}},
		{store: customStakingStore, key: []byte{0x0a, 0}, value: []byte{1}},
	} {
		result := accountingFixtureResult(t, append(append([]censusFixture(nil), leaves...), bad), true)
		require.False(t, result.Passed)
		require.False(t, result.ClaimantRootComplete)
	}
}

func TestAccountingReportCannotMasqueradeAsLegacyEvidence(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	leaves = append(leaves, censusFixture{store: customStakingStore, key: []byte{0x0a}, value: []byte{1}})
	result := accountingFixtureResult(t, leaves, true)
	options, snapshot, stores := reportTestEnvelope(result)
	_, _, err := buildCensusReport(options, snapshot, stores, result)
	require.ErrorContains(t, err, "source profile")
	options.SourceProfile = accountingSourceProfile
	data, passed, err := buildCensusReport(options, snapshot, stores, result)
	require.NoError(t, err)
	require.True(t, passed)
	var report sealedCensusReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.Equal(t, accountingReportSchema, report.Schema)
	require.NotEqual(t, reportSchema, report.Schema)
	second, _, err := buildCensusReport(options, snapshot, stores, result)
	require.NoError(t, err)
	require.Equal(t, data, second)
	result.BalanceUzrn = "0"
	_, _, err = buildCensusReport(options, snapshot, stores, result)
	require.Error(t, err)
}

func TestCensusUnknownSourceProfileRefusesBeforeOpeningDatabase(t *testing.T) {
	for _, profile := range []string{"future", ""} {
		t.Run("profile="+profile, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append(validCommandArgs(), "--source-profile", profile)
			code := run(args, &stdout, &stderr, func(_, _ string) (openedPhysicalDB, error) {
				t.Fatal("invalid profile opened database")
				return nil, nil
			}, func(physicalDB, censusOptions) ([]byte, bool, error) {
				t.Fatal("invalid profile scanned database")
				return nil, false, nil
			})
			require.Equal(t, exitOperational, code)
			require.Empty(t, stdout.String())
		})
	}
}

func TestAccountingReportRequiresExactMarkerCommitment(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	leaves = append(leaves, censusFixture{store: customStakingStore, key: []byte{0x0a}, value: []byte{1}})
	for _, mutation := range []string{"digest", "bytes"} {
		t.Run(mutation, func(t *testing.T) {
			result := accountingFixtureResult(t, leaves, true)
			options, snapshot, stores := reportTestEnvelope(result)
			options.SourceProfile = accountingSourceProfile
			marker := &result.Keyspace[customModuleKeyspaceCount+1]
			if mutation == "digest" {
				marker.Digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			} else {
				marker.InputBytes = 1
			}
			_, _, err := buildCensusReport(options, snapshot, stores, result)
			require.ErrorContains(t, err, "exact singleton")
		})
	}
}
