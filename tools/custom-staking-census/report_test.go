package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
)

func TestBuildCensusReportIsCanonicalAndSelfVerifying(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	result := runCensusFixture(t, leaves)
	options, snapshot, stores := reportTestEnvelope(result)

	first, passed, err := buildCensusReport(options, snapshot, stores, result)
	if err != nil {
		t.Fatal(err)
	}
	if !passed {
		t.Fatal("passing census was reported as failed")
	}
	second, _, err := buildCensusReport(options, snapshot, stores, result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("same report inputs produced different bytes")
	}

	var report sealedCensusReport
	if err := json.Unmarshal(first, &report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != reportSchema || report.Result != "PASS" {
		t.Fatalf("unexpected report header: %+v", report)
	}
	if report.Evidence.Height != "42" || report.Evidence.AppHash != hex.EncodeToString(options.AppHash) {
		t.Fatalf("unexpected evidence: %+v", report.Evidence)
	}
	if len(report.Stores) != len(requiredStoreNames) || report.Stores[0].LeafCount != "15" {
		t.Fatalf("unexpected stores: %+v", report.Stores)
	}
	if len(report.Multistore) != 4 {
		t.Fatalf("unexpected multistore commitments: %+v", report.Multistore)
	}
	recomputed := storetypes.CommitInfo{Version: options.Height}
	for _, row := range report.Multistore {
		root, err := hex.DecodeString(row.RootSHA256)
		if err != nil {
			t.Fatal(err)
		}
		recomputed.StoreInfos = append(recomputed.StoreInfos, storetypes.StoreInfo{
			Name: row.Name, CommitId: storetypes.CommitID{Version: options.Height, Hash: root},
		})
	}
	if !bytes.Equal(recomputed.Hash(), options.AppHash) {
		t.Fatal("reported multistore commitments do not recompute the AppHash")
	}

	wantDigest := report.ReportSHA256
	report.ReportSHA256 = ""
	unsealed, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(unsealed)
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("report digest = %s, want %s", wantDigest, got)
	}
}

func TestBuildCensusReportPreservesCompletedFailureEvidence(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	result := runCensusFixture(t, leaves)
	result.Passed = false
	result.ClaimantRootComplete = false
	result.Findings = []censusFinding{{Code: "synthetic_failure", Key: "test", Detail: "preserved"}}
	options, snapshot, stores := reportTestEnvelope(result)
	reportBytes, passed, err := buildCensusReport(options, snapshot, stores, result)
	if err != nil {
		t.Fatal(err)
	}
	if passed {
		t.Fatal("failed census was reported as passing")
	}
	var report sealedCensusReport
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	if report.Result != "FAIL" || len(report.Census.Findings) != 1 {
		t.Fatalf("failure evidence was not preserved: %+v", report)
	}
}

func TestBuildCensusReportRejectsInconsistentEnvelope(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	pass := runCensusFixture(t, leaves)
	options, snapshot, stores := reportTestEnvelope(pass)

	tests := []struct {
		name   string
		mutate func(*censusOptions, *rootSnapshot, *[]storeEvidence, *censusResult)
	}{
		{name: "missing store", mutate: func(_ *censusOptions, _ *rootSnapshot, stores *[]storeEvidence, _ *censusResult) {
			*stores = (*stores)[:2]
		}},
		{name: "wrong store order", mutate: func(_ *censusOptions, _ *rootSnapshot, stores *[]storeEvidence, _ *censusResult) {
			(*stores)[0], (*stores)[1] = (*stores)[1], (*stores)[0]
		}},
		{name: "wrong store version", mutate: func(_ *censusOptions, _ *rootSnapshot, stores *[]storeEvidence, _ *censusResult) {
			(*stores)[0].version++
		}},
		{name: "short root", mutate: func(_ *censusOptions, _ *rootSnapshot, stores *[]storeEvidence, _ *censusResult) {
			(*stores)[0].rootHash = []byte("short")
		}},
		{name: "pass with findings", mutate: func(_ *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, result *censusResult) {
			result.Findings = []censusFinding{{Code: "x"}}
		}},
		{name: "incomplete passing root", mutate: func(_ *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, result *censusResult) {
			result.ClaimantRootComplete = false
		}},
		{name: "invalid source", mutate: func(options *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, _ *censusResult) {
			options.SourceCommit = "NOT-A-COMMIT"
		}},
		{name: "multistore root changed", mutate: func(_ *censusOptions, snapshot *rootSnapshot, _ *[]storeEvidence, _ *censusResult) {
			store := snapshot.stores["auth"]
			store.rootHash = bytes.Repeat([]byte{0xee}, sha256.Size)
			snapshot.stores["auth"] = store
		}},
		{name: "hollow pass", mutate: func(_ *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, result *censusResult) {
			*result = censusResult{Passed: true, ClaimantRootComplete: true, Findings: []censusFinding{}}
		}},
		{name: "false arithmetic", mutate: func(_ *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, result *censusResult) {
			result.LiabilitiesUzrn = "999"
		}},
		{name: "internally consistent nonzero pass delta", mutate: func(_ *censusOptions, _ *rootSnapshot, _ *[]storeEvidence, result *censusResult) {
			result.PendingUnbondingsUzrn = "4"
			result.LiabilitiesUzrn = "34"
			result.DeltaUzrn = "1"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caseOptions := options
			caseSnapshot := cloneReportTestSnapshot(snapshot)
			caseStores := append([]storeEvidence(nil), stores...)
			caseResult := pass
			test.mutate(&caseOptions, &caseSnapshot, &caseStores, &caseResult)
			if _, _, err := buildCensusReport(caseOptions, caseSnapshot, caseStores, caseResult); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildCensusReportRejectsOversizedExpansionBeforeMarshal(t *testing.T) {
	leaves, _, _ := balancedCensusFixture(t)
	result := runCensusFixture(t, leaves)
	result.Passed = false
	result.ClaimantRootComplete = false
	result.Findings = []censusFinding{{
		Code:   "oversized",
		Key:    "test",
		Detail: string(bytes.Repeat([]byte{'x'}, maxReportBytes/6+1)),
	}}
	options, snapshot, stores := reportTestEnvelope(result)
	_, _, err := buildCensusReport(options, snapshot, stores, result)
	if err == nil || !strings.Contains(err.Error(), "resource bound") {
		t.Fatalf("expected pre-marshal resource-bound error, got %v", err)
	}
}

func reportTestEnvelope(result censusResult) (censusOptions, rootSnapshot, []storeEvidence) {
	const height = int64(42)
	stores := reportTestStores(height, result)
	snapshot := rootSnapshot{height: height, stores: make(map[string]committedStore, 4)}
	for _, store := range stores {
		snapshot.stores[store.name] = committedStore{version: store.version, rootHash: bytes.Clone(store.rootHash)}
	}
	snapshot.stores["auth"] = committedStore{version: height, rootHash: bytes.Repeat([]byte{0x44}, sha256.Size)}
	commitInfo := storetypes.CommitInfo{Version: height}
	for name, store := range snapshot.stores {
		commitInfo.StoreInfos = append(commitInfo.StoreInfos, storetypes.StoreInfo{
			Name: name, CommitId: storetypes.CommitID{Version: store.version, Hash: bytes.Clone(store.rootHash)},
		})
	}
	snapshot.appHash = commitInfo.Hash()
	options := censusOptions{
		ChainID:      "zerone-2",
		SourceCommit: "1111111111111111111111111111111111111111",
		Height:       height,
		AppHash:      bytes.Clone(snapshot.appHash),
	}
	return options, snapshot, stores
}

func reportTestStores(height int64, result censusResult) []storeEvidence {
	stores := make([]storeEvidence, len(requiredStoreNames))
	for index, name := range requiredStoreNames {
		stores[index] = storeEvidence{
			name:       name,
			version:    height,
			rootHash:   bytes.Repeat([]byte{byte(index + 1)}, sha256.Size),
			leafCount:  int64(index + 1),
			inputBytes: uint64((index + 1) * 10),
			leavesHash: bytes.Repeat([]byte{byte(index + 11)}, sha256.Size),
		}
	}
	var customLeaves, customBytes uint64
	for _, row := range result.Keyspace {
		customLeaves += row.LeafCount
		customBytes += row.InputBytes
	}
	stores[0].leafCount = int64(customLeaves)
	stores[0].inputBytes = customBytes
	return stores
}

func cloneReportTestSnapshot(snapshot rootSnapshot) rootSnapshot {
	clone := snapshot
	clone.appHash = bytes.Clone(snapshot.appHash)
	clone.stores = make(map[string]committedStore, len(snapshot.stores))
	for name, store := range snapshot.stores {
		clone.stores[name] = committedStore{version: store.version, rootHash: bytes.Clone(store.rootHash)}
	}
	return clone
}
