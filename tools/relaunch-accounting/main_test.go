package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

const fixturePath = "testdata/synthetic-100.snapshot-v3.json"

func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fixtureSnapshot(t *testing.T) snapshot {
	t.Helper()
	var s snapshot
	if err := json.Unmarshal(fixture(t), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func encode(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reportFor(t *testing.T, data []byte) accountingReport {
	t.Helper()
	out, err := accountSnapshot(data, digest(data))
	if err != nil {
		t.Fatal(err)
	}
	var report accountingReport
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func requireDenied(t *testing.T, data []byte, contains string) {
	t.Helper()
	out, err := accountSnapshot(data, digest(data))
	if err == nil || out != nil || !strings.Contains(err.Error(), contains) {
		t.Fatalf("want no report and error containing %q, got %s, %v", contains, out, err)
	}
}

func TestSynthetic100GoldenAndIndependentSelfHash(t *testing.T) {
	data := fixture(t)
	out, err := accountSnapshot(data, digest(data))
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile("testdata/synthetic-100.report.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(out, '\n'), golden) {
		t.Fatal("report differs from the reviewed synthetic golden")
	}
	var r accountingReport
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	if r.Supply != "100" || r.Totals.ModuleCustody != "60" || r.Totals.NonModuleLocation != "35" || r.Totals.UnknownLocation != "5" {
		t.Fatalf("not 100 = 60 + 35 + 5: %+v", r.Totals)
	}
	if r.Totals.OwnerCount != 5 || r.Totals.ModuleCount != 1 || r.Totals.NonModuleCount != 2 || r.Totals.UnknownCount != 2 {
		t.Fatalf("partition counted a bank location more than once: %+v", r.Totals)
	}
	if r.Validators.Tokens != "60" || r.Validators.Treatment != "METADATA_ONLY_NOT_ADDITIONAL_PRINCIPAL" {
		t.Fatal("validator metadata was lost or treated as extra principal")
	}
	if r.EntitlementTotal != nil || r.ReserveRequirement != nil || r.EntitlementStatus != "UNKNOWN" || r.ReserveStatus != "UNKNOWN" ||
		r.Eligibility != "UNDETERMINED" || r.EconomicEffect != "NONE" || r.PayoutAuthorization ||
		r.Trust.ProvenanceAuthenticated || r.Trust.SignaturesVerified || r.Trust.StateProofsVerified {
		t.Fatal("arithmetic report overstates authority or turns unknown into zero")
	}
	for i, row := range r.Owners {
		if row.Eligibility != "UNDETERMINED" || row.Restrictions != "UNKNOWN" || len(row.Gaps) < 3 ||
			(i > 0 && r.Owners[i-1].Address >= row.Address) {
			t.Fatalf("missing gaps, wrong eligibility, or unsorted owner: %+v", row)
		}
	}
	// Independently recompute over emitted bytes, not buildReport, its struct
	// field ordering, or its digest helper. Compact field order is significant.
	hashField := []byte(`"report_sha256":"` + r.ReportSHA256 + `"`)
	if bytes.Count(out, hashField) != 1 {
		t.Fatal("self-hash field not unique")
	}
	unsealed := bytes.Replace(out, hashField, []byte(`"report_sha256":""`), 1)
	sum := sha256.Sum256(unsealed)
	if hex.EncodeToString(sum[:]) != r.ReportSHA256 {
		t.Fatal("independent report self-hash recomputation failed")
	}
	inputSum := sha256.Sum256(data)
	if r.InputSHA256 != hex.EncodeToString(inputSum[:]) || r.Input.Bytes != len(data) || !r.Input.RawSHA256MatchesExpected {
		t.Fatal("raw input binding does not match independently hashed bytes")
	}
}

func TestDeterministicNormalizationRetainsRawBinding(t *testing.T) {
	data := fixture(t)
	first, err := accountSnapshot(data, digest(data))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		next, err := accountSnapshot(data, digest(data))
		if err != nil || !bytes.Equal(first, next) {
			t.Fatal("identical input did not produce identical output")
		}
	}
	var input map[string]any
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	owners := input["owners"].([]any)
	for i, j := 0, len(owners)-1; i < j; i, j = i+1, j-1 {
		owners[i], owners[j] = owners[j], owners[i]
	}
	reordered := encode(t, input) // changes field order and whitespace, too
	left, right := reportFor(t, data), reportFor(t, reordered)
	if left.InputSHA256 == right.InputSHA256 || left.ReportSHA256 == right.ReportSHA256 {
		t.Fatal("reordered raw bytes lost their distinct binding")
	}
	if out, err := accountSnapshot(reordered, digest(data)); err == nil || out != nil {
		t.Fatal("normalized equivalence bypassed raw pin")
	}
	left.Input, right.Input = inputEvidence{}, inputEvidence{}
	left.InputSHA256, right.InputSHA256 = "", ""
	left.ReportSHA256, right.ReportSHA256 = "", ""
	if !reflect.DeepEqual(left, right) {
		t.Fatal("normalized accounting differs for reordered equivalent source")
	}
}

func TestZeroInventoryStillHasUnknownEntitlementAndReserves(t *testing.T) {
	for _, nilSlices := range []bool{true, false} {
		s := fixtureSnapshot(t)
		s.Supply, s.Owners, s.Validators = "0", nil, nil
		if !nilSlices {
			s.Owners, s.Validators = []owner{}, []validator{}
		}
		r := reportFor(t, encode(t, s))
		if r.Supply != "0" || r.Totals.ModuleCustody != "0" || r.Totals.NonModuleLocation != "0" || r.Totals.UnknownLocation != "0" ||
			r.EntitlementTotal != nil || r.ReserveRequirement != nil || r.EntitlementStatus != "UNKNOWN" || r.ReserveStatus != "UNKNOWN" ||
			r.Owners == nil || r.Validators.Rows == nil || r.Totals.OwnerCount != 0 {
			t.Fatalf("zero bank supply confused with known entitlements: %+v", r)
		}
	}
	s := fixtureSnapshot(t)
	s.Owners = s.Owners[:3]
	s.Supply = "95"
	r := reportFor(t, encode(t, s))
	if r.Totals.UnknownLocation != "0" || r.Totals.UnknownCount != 0 || r.EntitlementStatus != "UNKNOWN" {
		t.Fatal("zero unknown locations asserted known beneficial entitlement")
	}
}

func TestAccountEvidenceClassification(t *testing.T) {
	cases := []struct{ accountType, module, class, gap string }{
		{moduleAccount, "gov", moduleCustody, "module_label_not_personal_ownership"},
		{baseAccount, "", nonModuleLocation, "base_account_label_not_unrestricted_personal_entitlement"},
		{"bank_only", "", unknownLocation, "auth_account_metadata_missing"},
		{"/future.ModuleAccount", "", unknownLocation, "account_type_unrecognized"},
		{"/future.VestingAccount", "", unknownLocation, "account_type_unrecognized"},
	}
	for _, kind := range []string{"BaseVestingAccount", "ContinuousVestingAccount", "DelayedVestingAccount", "PeriodicVestingAccount", "PermanentLockedAccount"} {
		cases = append(cases, struct{ accountType, module, class, gap string }{
			"/cosmos.vesting.v1beta1." + kind, "", nonModuleLocation, "vesting_schedule_delegation_and_spendability_unavailable",
		})
	}
	for _, tc := range cases {
		t.Run(tc.accountType, func(t *testing.T) {
			r := classify(owner{Address: "synthetic", AccountType: tc.accountType, ModuleName: tc.module, Amount: "1"})
			if r.Classification != tc.class || r.AccountType != tc.accountType || r.ModuleName != tc.module ||
				!strings.Contains(strings.Join(r.Gaps, ","), tc.gap) || r.Eligibility != "UNDETERMINED" {
				t.Fatalf("wrong evidence classification: %+v", r)
			}
		})
	}
}

func TestRejectInvalidSnapshotEvidence(t *testing.T) {
	cases := []struct {
		name    string
		change  func(*snapshot)
		message string
	}{
		{"schema", func(s *snapshot) { s.Schema = "zerone-relaunch-snapshot-v2" }, "schema"},
		{"denom", func(s *snapshot) { s.Denom = "zrn" }, "denomination"},
		{"chain", func(s *snapshot) { s.Source.ChainID = "" }, "chain"},
		{"F zero", func(s *snapshot) { s.Source.CheckpointStateHeight = 0 }, "F>0"},
		{"F overflow", func(s *snapshot) { s.Source.CheckpointStateHeight = 9223372036854775807 }, "F>0"},
		{"A mismatch", func(s *snapshot) { s.Source.FinalCommittedBlockHeight++ }, "F>0"},
		{"H mismatch", func(s *snapshot) { s.Source.HaltTriggerHeight++ }, "F>0"},
		{"blockstore", func(s *snapshot) { s.Source.RPCBlockstoreHeight-- }, "BlockStore"},
		{"ABCI", func(s *snapshot) { s.Source.ABCILastAppliedHeight-- }, "ABCI"},
		{"A txs", func(s *snapshot) { s.Source.FinalCommittedBlockTxs = 1 }, "empty-block"},
		{"H txs", func(s *snapshot) { s.Source.StagedHaltTriggerBlockTxs = 1 }, "empty-block"},
		{"A canonical", func(s *snapshot) { s.Source.FinalCommittedBlockCanonical = false }, "canonical"},
		{"A results", func(s *snapshot) { s.Source.FinalCommittedBlockHasResults = false }, "results"},
		{"H canonical", func(s *snapshot) { s.Source.StagedHaltTriggerCommitCanonical = true }, "canonical"},
		{"H results", func(s *snapshot) { s.Source.StagedHaltTriggerHasBlockResults = true }, "results"},
		{"checkpoint hash", func(s *snapshot) { s.Source.CheckpointAppHash = "wrong" }, "Comet hashes"},
		{"H link", func(s *snapshot) { s.Source.StagedHaltTriggerPreviousBlockHash = strings.Repeat("a", 64) }, "inconsistent"},
		{"post-anchor hash", func(s *snapshot) { s.Source.ExcludedPostAnchorAppHash = strings.Repeat("a", 64) }, "inconsistent"},
		{"same block", func(s *snapshot) { s.Source.StagedHaltTriggerBlockHash = s.Source.FinalCommittedBlockHash }, "inconsistent"},
		{"A time", func(s *snapshot) { s.Source.FinalCommittedBlockTime = "not-a-time" }, "anchor time"},
		{"H time", func(s *snapshot) { s.Source.StagedHaltTriggerBlockTime = "not-a-time" }, "trigger time"},
		{"H before A", func(s *snapshot) { s.Source.StagedHaltTriggerBlockTime = "2025-01-01T00:00:00Z" }, "later"},
		{"H equals A", func(s *snapshot) { s.Source.StagedHaltTriggerBlockTime = s.Source.FinalCommittedBlockTime }, "later"},
		{"RPC genesis", func(s *snapshot) { s.Source.RPCGenesisCanonicalSHA256 = strings.Repeat("E", 64) }, "genesis digests"},
		{"raw genesis", func(s *snapshot) { s.Source.DeclaredGenesisFileSHA256 = "bad" }, "genesis digests"},
		{"trust", func(s *snapshot) { s.Source.RESTTrustModel = "cryptographically verified" }, "trusted REST"},
		{"url protocol", func(s *snapshot) { s.Source.RPC = "file:///snapshot" }, "endpoints"},
		{"url host", func(s *snapshot) { s.Source.REST = "https:///path" }, "endpoints"},
		{"url credentials", func(s *snapshot) { s.Source.RPC = "https://user:synthetic@host.invalid" }, "endpoints"},
		{"url query", func(s *snapshot) { s.Source.RPC += "?key=synthetic" }, "endpoints"},
		{"url empty query", func(s *snapshot) { s.Source.RPC += "?" }, "endpoints"},
		{"url fragment", func(s *snapshot) { s.Source.REST += "#fragment" }, "endpoints"},
		{"supply mismatch", func(s *snapshot) { s.Supply = "101" }, "balance sum"},
		{"missing owner", func(s *snapshot) { s.Owners = s.Owners[1:] }, "balance sum"},
		{"duplicate owner", func(s *snapshot) { s.Owners[1].Address = s.Owners[0].Address }, "duplicate owner"},
		{"owner address", func(s *snapshot) { s.Owners[0].Address = "" }, "invalid address"},
		{"owner type", func(s *snapshot) { s.Owners[0].AccountType = "" }, "account type"},
		{"label control", func(s *snapshot) { s.Owners[0].Address = "synthetic\naddress" }, "invalid address"},
		{"module name", func(s *snapshot) { s.Owners[0].ModuleName = "" }, "module name"},
		{"wrong module type", func(s *snapshot) { s.Owners[0].AccountType = baseAccount }, "without ModuleAccount"},
		{"duplicate module", func(s *snapshot) {
			s.Owners[1].AccountType = moduleAccount
			s.Owners[1].ModuleName = s.Owners[0].ModuleName
		}, "duplicate module"},
		{"owner zero", func(s *snapshot) { s.Owners[0].Amount = "0" }, "positive"},
		{"duplicate validator", func(s *snapshot) { s.Validators = append(s.Validators, s.Validators[0]) }, "duplicate bonded"},
		{"validator operator", func(s *snapshot) { s.Validators[0].OperatorAddress = "" }, "operator"},
		{"validator status", func(s *snapshot) { s.Validators[0].Status = "BOND_STATUS_UNBONDED" }, "status"},
		{"validator zero", func(s *snapshot) { s.Validators[0].Tokens = "0" }, "positive"},
		{"validator excess", func(s *snapshot) { s.Validators[0].Tokens = "101" }, "exceeds bank supply"},
		{"key type", func(s *snapshot) { s.Validators[0].ConsensusPubKey.Type = "" }, "key type"},
		{"key encoding", func(s *snapshot) { s.Validators[0].ConsensusPubKey.Key = "%%%" }, "key encoding"},
		{"key size", func(s *snapshot) { s.Validators[0].ConsensusPubKey.Key = "AA==" }, "key encoding"},
		{"key newline", func(s *snapshot) { s.Validators[0].ConsensusPubKey.Key += "\n" }, "key encoding"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := fixtureSnapshot(t)
			tc.change(&s)
			requireDenied(t, encode(t, s), tc.message)
		})
	}
}

func TestCanonicalAmountsAtEveryLocation(t *testing.T) {
	for _, value := range []string{"", "00", "01", "-1", "-0", "+1", "1.0", "1e2", " 1", "1 ", "١", "NaN", strings.Repeat("9", maxAmountDigits+1)} {
		for _, field := range []string{"supply", "owner", "validator"} {
			t.Run(field+"/"+value, func(t *testing.T) {
				s := fixtureSnapshot(t)
				switch field {
				case "supply":
					s.Supply = value
				case "owner":
					s.Owners[0].Amount = value
				case "validator":
					s.Validators[0].Tokens = value
				}
				requireDenied(t, encode(t, s), "")
			})
		}
	}
	// Exact arithmetic above float64 and uint64; at the documented digit bound.
	s := fixtureSnapshot(t)
	s.Supply = strings.Repeat("9", maxAmountDigits)
	s.Owners = []owner{{Address: "synthetic-large", AccountType: baseAccount, Amount: s.Supply}}
	s.Validators = nil
	if r := reportFor(t, encode(t, s)); r.Totals.NonModuleLocation != s.Supply {
		t.Fatal("large integer lost precision")
	}
}

func TestStrictJSONSchemaAtEveryObjectLevel(t *testing.T) {
	base := string(encode(t, fixtureSnapshot(t)))
	cases := []struct{ name, old, replacement, message string }{
		{"duplicate root", `"schema":`, `"schema":"ignored","schema":`, "duplicate"},
		{"escaped duplicate", `"denom":`, `"denom":"uzrn","` + string(rune(92)) + `u0064enom":`, "duplicate"},
		{"duplicate source", `"chain_id":`, `"chain_id":"ignored","chain_id":`, "duplicate"},
		{"duplicate owner", `"address":`, `"address":"ignored","address":`, "duplicate"},
		{"duplicate validator", `"jailed":`, `"jailed":true,"jailed":`, "duplicate"},
		{"duplicate pubkey", `"@type":`, `"@type":"ignored","@type":`, "duplicate"},
		{"unknown root", `"schema":`, `"extra":true,"schema":`, "unknown"},
		{"unknown source", `"chain_id":`, `"extra":true,"chain_id":`, "unknown"},
		{"unknown owner", `"address":`, `"extra":true,"address":`, "unknown"},
		{"unknown validator", `"jailed":`, `"extra":true,"jailed":`, "unknown"},
		{"unknown pubkey", `"@type":`, `"extra":true,"@type":`, "unknown"},
		{"case alias", `"denom":`, `"Denom":`, "cased"},
		{"case duplicate", `"denom":`, `"Denom":"uzrn","denom":`, "cased"},
		{"missing false bool", `"staged_halt_trigger_commit_canonical":false,`, ``, "missing required"},
		{"null bool", `"staged_halt_trigger_commit_canonical":false`, `"staged_halt_trigger_commit_canonical":null`, "boolean"},
		{"null string", `"denom":"uzrn"`, `"denom":null`, "string"},
		{"number amount", `"supply_uzrn":"100"`, `"supply_uzrn":100`, "string"},
		{"string height", `"checkpoint_state_height":100`, `"checkpoint_state_height":"100"`, "integer"},
		{"float height", `"checkpoint_state_height":100`, `"checkpoint_state_height":100.0`, "canonical"},
		{"exponent height", `"checkpoint_state_height":100`, `"checkpoint_state_height":1e2`, "canonical"},
		{"negative height", `"checkpoint_state_height":100`, `"checkpoint_state_height":-100`, "canonical"},
		{"int overflow", `"checkpoint_state_height":100`, `"checkpoint_state_height":9223372036854775808`, "canonical"},
		{"negative zero", `"final_committed_block_txs":0`, `"final_committed_block_txs":-0`, "canonical"},
		{"null owner", `"owners":[`, `"owners":[null,`, "expected object"},
		{"surrogate", `"denom":"uzrn"`, `"denom":"\ud800"`, "Unicode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.old) {
				t.Fatal("broken mutation")
			}
			requireDenied(t, []byte(strings.Replace(base, tc.old, tc.replacement, 1)), tc.message)
		})
	}
	for _, data := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("{}"), []byte("{"), []byte(base + "{}"), []byte(base + "garbage"), []byte("\xef\xbb\xbf" + base), []byte(strings.Replace(base, "uzrn", "\xff", 1))} {
		requireDenied(t, data, "")
	}
	// Every required wire field is checked, including zero/false values that
	// ordinary struct decoding would silently default when omitted.
	var root map[string]any
	if err := json.Unmarshal([]byte(base), &root); err != nil {
		t.Fatal(err)
	}
	objects := []map[string]any{root, root["source"].(map[string]any), root["owners"].([]any)[0].(map[string]any), root["bonded_validators"].([]any)[0].(map[string]any)}
	objects = append(objects, objects[3]["consensus_pubkey"].(map[string]any))
	for _, obj := range objects {
		for key, value := range obj {
			if key == "module_name" || key == "declared_genesis_file_sha256" {
				continue
			}
			delete(obj, key)
			requireDenied(t, encode(t, root), "missing required")
			obj[key] = value
		}
	}
}

func TestBounds(t *testing.T) {
	s := fixtureSnapshot(t)
	s.Owners[0].Address = strings.Repeat("x", maxStringBytes+1)
	requireDenied(t, encode(t, s), "bounded string")
	s = fixtureSnapshot(t)
	s.Owners[0].Address = strings.Repeat("x", 257)
	requireDenied(t, encode(t, s), "invalid address")
	data := bytes.Repeat([]byte(" "), maxInputBytes+1)
	out, err := accountSnapshot(data, digest(data))
	if err == nil || out != nil {
		t.Fatal("accepted oversized input")
	}
	// The array limit fires during token parsing before supply/duplicate checks.
	for _, tc := range []struct {
		key   string
		limit int
		value any
	}{
		{"owners", maxOwners, fixtureSnapshot(t).Owners[1]},
		{"bonded_validators", maxValidators, fixtureSnapshot(t).Validators[0]},
	} {
		row := encode(t, tc.value)
		var root map[string]json.RawMessage
		if err := json.Unmarshal(fixture(t), &root); err != nil {
			t.Fatal(err)
		}
		root[tc.key] = append(append([]byte("["), bytes.Repeat(append(row, ','), tc.limit)...), row...)
		root[tc.key] = append(root[tc.key], ']')
		requireDenied(t, encode(t, root), "row limit")
	}
	d := json.NewDecoder(strings.NewReader(`{}`))
	if err := walkSnapshotJSON(d, reflect.TypeOf(snapshot{}), maxDepth+1); err == nil {
		t.Fatal("depth ceiling not enforced")
	}
	deep := []byte(strings.Repeat("[", maxDepth+2) + "0" + strings.Repeat("]", maxDepth+2))
	requireDenied(t, deep, "expected object")
	// Conservative report size guard runs before expanded rows are constructed.
	s = fixtureSnapshot(t)
	s.Owners = make([]owner, maxOwners)
	for i := range s.Owners {
		s.Owners[i] = owner{Address: strings.Repeat("x", 256), AccountType: baseAccount, Amount: "1"}
	}
	if out, err := buildReport(s, "", 0); err == nil || out != nil || !strings.Contains(err.Error(), "serialization") {
		t.Fatal("report allocation guard missing")
	}
}

func TestRawDigestFailurePrecedesParsing(t *testing.T) {
	for _, pin := range []string{"", strings.Repeat("A", 64), strings.Repeat("g", 64), strings.Repeat("0", 63)} {
		if out, err := accountSnapshot(fixture(t), pin); err == nil || out != nil || !strings.Contains(err.Error(), "lowercase") {
			t.Fatalf("invalid digest syntax accepted: %q", pin)
		}
	}
	if out, err := accountSnapshot([]byte("not JSON"), strings.Repeat("0", 64)); err == nil || out != nil || !strings.Contains(err.Error(), "raw SHA-256") {
		t.Fatal("raw digest verification not independent of parsing")
	}
}

func TestOptionalGenesisAndValidatorOrdering(t *testing.T) {
	s := fixtureSnapshot(t)
	s.Source.DeclaredGenesisFileSHA256 = ""
	s.Validators[0].Tokens = "30"
	other := s.Validators[0]
	other.OperatorAddress = "aaa-synthetic-validator"
	s.Validators = append(s.Validators, other)
	r := reportFor(t, encode(t, s))
	if !strings.Contains(strings.Join(r.Gaps, ","), "declared_raw_genesis_file_digest_not_supplied") ||
		r.Validators.Rows[0].OperatorAddress != other.OperatorAddress || r.Validators.Tokens != "60" {
		t.Fatal("optional genesis gap or validator sorting missing")
	}
}

func TestCLIAndReadOnlyFilesystemBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "synthetic.snapshot.json")
	data := fixture(t)
	if err := os.WriteFile(path, data, 0o400); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	entriesBefore, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"--snapshot", path, "--expected-sha256", digest(data)}
	if code := run(args, &stdout, &stderr); code != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) || !bytes.HasSuffix(stdout.Bytes(), []byte("\n")) {
		t.Fatalf("CLI failed: code %d, stderr %s", code, &stderr)
	}
	second := new(bytes.Buffer)
	if code := run([]string{"--expected-sha256=" + digest(data), "--snapshot=" + path}, second, &stderr); code != 0 || !bytes.Equal(stdout.Bytes(), second.Bytes()) {
		t.Fatal("flag ordering or input path changed deterministic report")
	}
	after, err := os.Stat(path)
	if err != nil || !sameFile(before, after) {
		t.Fatal("input metadata changed")
	}
	afterBytes, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, afterBytes) {
		t.Fatal("input bytes changed")
	}
	entriesAfter, err := os.ReadDir(dir)
	if err != nil || len(entriesAfter) != len(entriesBefore) {
		t.Fatal("CLI created side files")
	}

	for _, bad := range [][]string{
		nil, {"--snapshot", path}, {"--expected-sha256", digest(data)},
		{"--snapshot", "-", "--expected-sha256", digest(data)},
		{"--snapshot", path, "--expected-sha256", strings.Repeat("A", 64)},
		{"--snapshot", path, "--expected-sha256", strings.Repeat("0", 64)},
		{"--snapshot", path, "--snapshot", path, "--expected-sha256", digest(data)},
		{"--snapshot", path, "--expected-sha256", digest(data), "--expected-sha256", digest(data)},
		{"--snapshot", path, "--expected-sha256", digest(data), "extra"},
		{"--snapshot", path, "--expected-sha256", digest(data), "--out", "forbidden"},
		{"--snapshot", path, "--expected-sha256", digest(data), "--rpc", "https://synthetic.invalid"},
		{"--snapshot", filepath.Join(dir, "absent"), "--expected-sha256", digest(data)},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := run(bad, &stdout, &stderr); code == 0 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("failed CLI emitted report or lost diagnostic: args %v, code %d", bad, code)
		}
	}
	badPath := filepath.Join(dir, "invalid.json")
	for _, invalid := range [][]byte{nil, []byte("{}"), []byte("malformed"), []byte(strings.Replace(string(data), `"supply_uzrn": "100"`, `"supply_uzrn": "101"`, 1))} {
		if err := os.WriteFile(badPath, invalid, 0o600); err != nil {
			t.Fatal(err)
		}
		stdout.Reset()
		stderr.Reset()
		if code := run([]string{"--snapshot", badPath, "--expected-sha256", digest(invalid)}, &stdout, &stderr); code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("input failure emitted stdout: code %d", code)
		}
	}
	stderr.Reset()
	if code := run(args, failedWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "write complete") {
		t.Fatal("output error lost")
	}
	stderr.Reset()
	if code := run(args, shortWriter{}, &stderr); code != 1 {
		t.Fatal("short write accepted")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic write failure") }

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func TestRegularFileBoundary(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, fixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	large := filepath.Join(dir, "large")
	f, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxInputBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, link, fifo, large, filepath.Join(dir, "missing"), "/dev/null"} {
		if data, err := readRegularFile(path); err == nil || data != nil {
			t.Fatalf("accepted nonregular/unbounded input %s", path)
		}
		var stdout, stderr bytes.Buffer
		if code := run([]string{"--snapshot", path, "--expected-sha256", digest(fixture(t))}, &stdout, &stderr); code != 1 || stdout.Len() != 0 {
			t.Fatalf("CLI did not reject file boundary %s", path)
		}
	}
}

func TestProductionImportAndSideEffectBoundaries(t *testing.T) {
	// Auditable allowlist: no network client, signing package, database, SDK,
	// keyring, dependency, unsafe or process-execution import can slip in.
	allowed := strings.Fields("bytes crypto/sha256 encoding/base64 encoding/hex encoding/json errors flag fmt io math math/big net/url os reflect sort strconv strings syscall time unicode unicode/utf8")
	imports := make(map[string]bool)
	for _, path := range allowed {
		imports[path] = true
	}
	osAllowed := strings.Fields("Args Stdout Stderr Exit Lstat NewFile SameFile FileInfo")
	osNames := make(map[string]bool)
	for _, name := range osAllowed {
		osNames[name] = true
	}
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		aliases := make(map[string]string)
		for _, imp := range file.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !imports[name] {
				t.Fatalf("forbidden runtime import in %s: %s", path, imp.Path.Value)
			}
			if imp.Name != nil {
				t.Fatalf("import aliases require explicit boundary review: %s", path)
			}
			aliases[filepath.Base(name)] = name
		}
		for _, comments := range file.Comments {
			for _, c := range comments.List {
				if strings.HasPrefix(c.Text, "//go:") {
					t.Fatal("compiler directive requires boundary review")
				}
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok && fn.Name.Name == "init" {
				t.Fatal("runtime init side effect forbidden")
			}
			sel, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			switch aliases[pkg.Name] {
			case "os":
				if !osNames[sel.Sel.Name] {
					t.Fatalf("unreviewed runtime os operation: %s", sel.Sel.Name)
				}
			case "syscall":
				switch sel.Sel.Name {
				case "Open", "Close", "O_RDONLY", "O_CLOEXEC", "O_NOFOLLOW", "O_NONBLOCK":
				default:
					t.Fatalf("forbidden runtime syscall: %s", sel.Sel.Name)
				}
			case "net/url":
				if sel.Sel.Name != "Parse" {
					t.Fatal("URL package is for syntax validation only")
				}
			}
			return true
		})
	}
}

func Example_run() {
	// Synthetic fixture only. A real invocation requires a separately obtained
	// expected digest. Computing one here is not provenance authentication.
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		panic(err)
	}
	var stdout bytes.Buffer
	code := run([]string{"--snapshot", fixturePath, "--expected-sha256", digest(data)}, &stdout, io.Discard)
	var r accountingReport
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		panic(err)
	}
	fmt.Printf("exit=%d: %s = %s + %s + %s; eligibility=%s; effect=%s; payout=%t\n", code, r.Supply, r.Totals.ModuleCustody, r.Totals.NonModuleLocation, r.Totals.UnknownLocation, r.Eligibility, r.EconomicEffect, r.PayoutAuthorization)
	// Output: exit=0: 100 = 60 + 35 + 5; eligibility=UNDETERMINED; effect=NONE; payout=false
}
