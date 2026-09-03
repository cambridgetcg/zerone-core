package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryArtifactIsCoherentSourceOnly(t *testing.T) {
	report, err := verifyRepository(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Format != verificationFormat || report.Protocol != protocolID || report.Decision != "COHERENT_SOURCE_ONLY" ||
		report.ManifestRawSHA256 != expectedManifestRawSHA256 || report.VerifiedLocalSourcePins != 6 ||
		report.PinnedExternalSources != 2 || report.PendingBindings != 1 || len(report.Fixtures) != 3 ||
		report.CandidateEffects != (candidateEffects{}) ||
		report.ObservedValidatorEffects != (validatorExecution{LocalFileRead: "BOUNDED_EXPLICIT_REPOSITORY_FILES"}) {
		t.Fatalf("artifact report drifted: %#v", report)
	}
	if report.Fixtures[0].ObservedDecision != "ACCEPT" || report.Fixtures[1].ObservedDecision != "ACCEPT" || report.Fixtures[1].ConflictGroups != 1 || report.Fixtures[2].ObservedDecision != "REJECT" {
		t.Fatalf("fixture decisions drifted: %#v", report.Fixtures)
	}
}

func TestCommandEmitsClosedVerificationReport(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--repository-root", repositoryRoot(t)}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v; stderr=%q", err, stderr.String())
	}
	var report verificationReport
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Decision != "COHERENT_SOURCE_ONLY" || report.ManifestRawSHA256 != expectedManifestRawSHA256 {
		t.Fatalf("command report drifted: %#v", report)
	}
}

func TestFixtureObservationIDsAreCanonical(t *testing.T) {
	for _, path := range fixturePaths() {
		data := mustRead(t, path)
		var document journal
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		for index, raw := range document.Observations {
			var envelope struct {
				ObservationID string `json:"observation_id"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatal(err)
			}
			expected, err := observationID(raw)
			if err != nil {
				t.Fatal(err)
			}
			if envelope.ObservationID != expected {
				t.Errorf("%s observation[%d] ID = %s, want %s", filepath.Base(path), index, envelope.ObservationID, expected)
			}
		}
	}
}

func TestValidCurrentObservationAndSameHeightConflict(t *testing.T) {
	current, err := validateJournal(mustRead(t, fixturePaths()[0]))
	if err != nil {
		t.Fatal(err)
	}
	if current.ObservationCount != 1 || len(current.ConflictGroups) != 0 {
		t.Fatalf("current observation summary = %#v", current)
	}
	conflict, err := validateJournal(mustRead(t, fixturePaths()[1]))
	if err != nil {
		t.Fatal(err)
	}
	if conflict.ObservationCount != 2 || len(conflict.ConflictGroups) != 1 || len(conflict.ConflictGroups[0].ObservationIDs) != 2 {
		t.Fatalf("same-height conflict was not preserved: %#v", conflict)
	}
}

func TestTruncatedAndUnavailableSourcesFailClosed(t *testing.T) {
	truncated := mustRead(t, fixturePaths()[2])
	if _, err := validateJournal(truncated); err == nil || !strings.Contains(err.Error(), "source_status") {
		t.Fatalf("truncated source did not fail closed: %v", err)
	}
	unavailable := bytes.Replace(truncated, []byte(`"source_status": "TRUNCATED"`), []byte(`"source_status": "UNAVAILABLE"`), 1)
	if _, err := validateJournal(unavailable); err == nil || !strings.Contains(err.Error(), "source_status") {
		t.Fatalf("unavailable source did not fail closed: %v", err)
	}
}

func TestToKIsCurrentOnlyAndPayloadDigestIsSeparate(t *testing.T) {
	valid := mustRead(t, fixturePaths()[0])
	nonCurrent := bytes.Replace(valid, []byte(`"at_block_height": 0`), []byte(`"at_block_height": 1`), 1)
	if _, err := validateJournal(nonCurrent); err == nil || !strings.Contains(err.Error(), "at_block_height") {
		t.Fatalf("non-current ToK request did not fail: %v", err)
	}
	var document journal
	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatal(err)
	}
	var observation tokObservation
	if err := json.Unmarshal(document.Observations[0], &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Response.ToKSnapshotRoot == "" || observation.Response.RawPayloadSHA256 == "" || strings.TrimPrefix(observation.Response.ToKSnapshotRoot, "hex:") == strings.TrimPrefix(observation.Response.RawPayloadSHA256, "sha256:") {
		t.Fatal("ToK topology root and raw payload SHA-256 must remain separate")
	}
	missingPayload := bytes.Replace(valid, []byte(observation.Response.RawPayloadSHA256), []byte("sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), 1)
	if _, err := validateJournal(missingPayload); err == nil || !strings.Contains(err.Error(), "ID =") {
		t.Fatalf("unbound payload digest replacement did not fail content ID: %v", err)
	}
}

func TestGraphKindsStayTaggedAndStructurallyDistinct(t *testing.T) {
	static := staticTreeObservation{
		ObservationID: zeroDigest(), GraphKind: "STATIC_TREE", SourceKind: "REPOSITORY_BYTES", SourceStatus: "COMPLETE", ObservedAt: "2026-08-21T09:44:00Z",
		Source: repositoryBytesSource{staticTreeRepository, staticTreeRevision, staticTreePath, staticTreeRawPayloadSHA256, true},
	}
	geometry := knowledgeGeometryObservation{
		ObservationID: zeroDigest(), GraphKind: "KNOWLEDGE_GEOMETRY", SourceKind: "ZERONE_BOUNDED_READ_PROJECTION", SourceStatus: "COMPLETE", ObservedAt: "2026-08-21T09:44:01Z",
		Response: knowledgeGeometryResponse{knowledgeGeometryChainID, "424244", "sha256:4dd27e6ce2db5d732c4743694499b199e96cffca877f2b87e206e870498ca666", true, "NOT_CLAIMED", false, "SYNTHETIC_UNVERIFIED_RESPONSE"},
	}
	for _, observation := range []any{static, geometry} {
		raw := observationWithCanonicalID(t, observation)
		document := journalBytes(t, raw)
		if _, err := validateJournal(document); err != nil {
			t.Fatalf("valid tagged graph observation failed: %v", err)
		}
	}
	tok := mustRead(t, fixturePaths()[0])
	relabelled := bytes.Replace(tok, []byte(`"graph_kind": "TOK_ONCHAIN"`), []byte(`"graph_kind": "STATIC_TREE"`), 1)
	if _, err := validateJournal(relabelled); err == nil {
		t.Fatal("ToK fields were accepted after relabelling as STATIC_TREE")
	}
}

func TestKnowledgeGeometryAcceptsOnlyPinnedProjectionChain(t *testing.T) {
	value := knowledgeGeometryObservation{
		ObservationID: zeroDigest(), GraphKind: "KNOWLEDGE_GEOMETRY", SourceKind: "ZERONE_BOUNDED_READ_PROJECTION", SourceStatus: "COMPLETE", ObservedAt: "2026-08-21T09:44:01Z",
		Response: knowledgeGeometryResponse{"evil-1", "424244", "sha256:4dd27e6ce2db5d732c4743694499b199e96cffca877f2b87e206e870498ca666", true, "NOT_CLAIMED", false, "SYNTHETIC_UNVERIFIED_RESPONSE"},
	}
	if _, err := validateJournal(journalBytes(t, observationWithCanonicalID(t, value))); err == nil || !strings.Contains(err.Error(), "Knowledge Geometry response") {
		t.Fatalf("unpinned Knowledge Geometry chain was accepted: %v", err)
	}
}

func TestStaticTreeAcceptsOnlyExactPinnedSource(t *testing.T) {
	base := staticTreeObservation{
		ObservationID: zeroDigest(), GraphKind: "STATIC_TREE", SourceKind: "REPOSITORY_BYTES", SourceStatus: "COMPLETE", ObservedAt: "2026-08-21T09:44:00Z",
		Source: repositoryBytesSource{staticTreeRepository, staticTreeRevision, staticTreePath, staticTreeRawPayloadSHA256, true},
	}
	mutations := []struct {
		name   string
		mutate func(*staticTreeObservation)
	}{
		{"arbitrary repository", func(value *staticTreeObservation) { value.Source.Repository = "x" }},
		{"path outside schema", func(value *staticTreeObservation) { value.Source.Path = "a b" }},
		{"different well-formed revision", func(value *staticTreeObservation) { value.Source.Revision = strings.Repeat("f", 40) }},
		{"different well-formed digest", func(value *staticTreeObservation) {
			value.Source.RawPayloadSHA256 = "sha256:" + strings.Repeat("f", 64)
		}},
	}
	for _, item := range mutations {
		t.Run(item.name, func(t *testing.T) {
			mutated := base
			item.mutate(&mutated)
			if _, err := validateJournal(journalBytes(t, observationWithCanonicalID(t, mutated))); err == nil || !strings.Contains(err.Error(), "exact pinned Zerone Tree") {
				t.Fatalf("schema-hostile static Tree source was accepted: %v", err)
			}
		})
	}
}

func TestRawByteSealsRejectBOMAndWhitespaceDrift(t *testing.T) {
	manifest := mustRead(t, filepath.Join(repositoryRoot(t), manifestRelativePath))
	if err := validateManifestSeal(manifest); err != nil {
		t.Fatal(err)
	}
	for name, drifted := range map[string][]byte{
		"bom":        append([]byte{0xef, 0xbb, 0xbf}, manifest...),
		"whitespace": append(append([]byte(nil), manifest...), '\n'),
	} {
		t.Run("manifest "+name, func(t *testing.T) {
			if err := validateManifestSeal(drifted); err == nil || !strings.Contains(err.Error(), "exact sealed bytes") {
				t.Fatalf("manifest raw-byte drift was accepted: %v", err)
			}
			repository := t.TempDir()
			path := filepath.Join(repository, filepath.FromSlash(manifestRelativePath))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, drifted, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := verifyRepository(repository); err == nil || !strings.Contains(err.Error(), "exact sealed bytes") {
				t.Fatalf("repository verifier did not reject manifest raw-byte drift before path use: %v", err)
			}
		})
	}
	valid := mustRead(t, fixturePaths()[0])
	if _, err := validateJournal(append([]byte{0xef, 0xbb, 0xbf}, valid...)); err == nil {
		t.Fatal("BOM-prefixed journal was accepted")
	}
}

func TestClosedJSONAndZeroEffects(t *testing.T) {
	valid := mustRead(t, fixturePaths()[0])
	unknown := bytes.Replace(valid, []byte(`"mode": "SOURCE_ONLY_SYNTHETIC_FIXTURE",`), []byte(`"mode": "SOURCE_ONLY_SYNTHETIC_FIXTURE", "surprise": false,`), 1)
	if _, err := validateJournal(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field was accepted: %v", err)
	}
	duplicate := bytes.Replace(valid, []byte(`"mode": "SOURCE_ONLY_SYNTHETIC_FIXTURE",`), []byte(`"mode": "SOURCE_ONLY_SYNTHETIC_FIXTURE", "mode": "SOURCE_ONLY_SYNTHETIC_FIXTURE",`), 1)
	if _, err := validateJournal(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate field was accepted: %v", err)
	}
	economic := bytes.Replace(valid, []byte(`"economic": false`), []byte(`"economic": true`), 1)
	if _, err := validateJournal(economic); err == nil || !strings.Contains(err.Error(), "effect vector") {
		t.Fatalf("economic effect was accepted: %v", err)
	}
	missingFalseEffect := bytes.Replace(valid, []byte("    \"economic\": false,\n"), nil, 1)
	if _, err := validateJournal(missingFalseEffect); err == nil || !strings.Contains(err.Error(), "exactly 25 fields") {
		t.Fatalf("missing false effect field was accepted: %v", err)
	}
	missingFalseProjectionBoundary := bytes.Replace(valid, []byte("    \"database_order_is_chain_order\": false,\n"), nil, 1)
	if _, err := validateJournal(missingFalseProjectionBoundary); err == nil || !strings.Contains(err.Error(), "exactly 10 fields") {
		t.Fatalf("missing false projection field was accepted: %v", err)
	}
}

func TestSyntheticDigestPreimagesAreReal(t *testing.T) {
	type check struct {
		fixture, index                      int
		selector, block, app, root, payload string
	}
	checks := []check{
		{0, 0, "selector:valid-current:v0.1", "block:valid-current:v0.1", "app:valid-current:v0.1", "tok-root:valid-current:v1", "payload:valid-current:v0.1"},
		{1, 0, "selector:same-height-conflict:v0.1", "block:same-height-conflict:a:v0.1", "app:same-height-conflict:a:v0.1", "tok-root:same-height-conflict:a:v1", "payload:same-height-conflict:a:v0.1"},
		{1, 1, "selector:same-height-conflict:v0.1", "block:same-height-conflict:b:v0.1", "app:same-height-conflict:b:v0.1", "tok-root:same-height-conflict:b:v1", "payload:same-height-conflict:b:v0.1"},
		{2, 0, "selector:truncated-unavailable:v0.1", "block:truncated-unavailable:v0.1", "app:truncated-unavailable:v0.1", "tok-root:truncated-unavailable:v1", "payload:truncated-unavailable:prefix:v0.1"},
	}
	for _, item := range checks {
		var document journal
		if err := json.Unmarshal(mustRead(t, fixturePaths()[item.fixture]), &document); err != nil {
			t.Fatal(err)
		}
		var observation tokObservation
		if err := json.Unmarshal(document.Observations[item.index], &observation); err != nil {
			t.Fatal(err)
		}
		if observation.Request.SelectorSHA256 != sha256Label("sha256:", item.selector) ||
			observation.Response.ReturnedBlockHash != sha256Label("hex:", item.block) ||
			observation.Response.ReturnedAppHash != sha256Label("hex:", item.app) ||
			observation.Response.ToKSnapshotRoot != sha256Label("hex:", item.root) ||
			observation.Response.RawPayloadSHA256 != sha256Label("sha256:", item.payload) {
			t.Fatalf("synthetic digest preimage drift: %#v", observation)
		}
	}
}

func TestSchemaNamesAllThreeGraphKindsAndIsClosed(t *testing.T) {
	data := mustRead(t, filepath.Join(repositoryRoot(t), "tools/zerone-supabase-observatory/protocol/observation-journal.v0.1.schema.json"))
	if err := validateSchemaSurface(data); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"STATIC_TREE", "TOK_ONCHAIN", "KNOWLEDGE_GEOMETRY", `"additionalProperties": false`,
		`"at_block_height": {"const": 0}`, `"repository": {"const": "` + staticTreeRepository + `"}`,
		`"revision": {"const": "` + staticTreeRevision + `"}`, `"path": {"const": "` + staticTreePath + `"}`,
		`"raw_payload_sha256": {"const": "` + staticTreeRawPayloadSHA256 + `"}`,
		`"returned_chain_id": {"const": "` + knowledgeGeometryChainID + `"}`,
	} {
		if !bytes.Contains(data, []byte(required)) {
			t.Fatalf("schema missing %q", required)
		}
	}
}

func TestToolHasNoRuntimeNetworkDatabaseOrChainImports(t *testing.T) {
	directory := filepath.Join(repositoryRoot(t), "tools/zerone-supabase-observatory")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data := mustRead(t, filepath.Join(directory, entry.Name()))
		for _, forbidden := range []string{`"net/http"`, `"database/sql"`, "supabase-go", "cosmos-sdk", "cometbft/rpc", "grpc.Dial"} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatalf("%s contains runtime integration %q", entry.Name(), forbidden)
			}
		}
	}
}

func TestBoundedRepositoryReaderRejectsSymlinksAndMissingSources(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "source.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("source.json", filepath.Join(directory, "alias.json")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := readRootFile(root, "alias.json", 1024); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink was accepted: %v", err)
	}
	if _, err := readRootFile(root, "missing.json", 1024); err == nil {
		t.Fatal("missing source was accepted")
	}
	if _, err := readRootFile(root, "../source.json", 1024); err == nil {
		t.Fatal("path escape was accepted")
	}
}

func fixturePaths() []string {
	root := repositoryRootNoTest()
	return []string{
		filepath.Join(root, "tools/zerone-supabase-observatory/testdata/valid-current-observation.json"),
		filepath.Join(root, "tools/zerone-supabase-observatory/testdata/same-height-conflicts-preserved.json"),
		filepath.Join(root, "tools/zerone-supabase-observatory/testdata/truncated-unavailable-source-fails-closed.json"),
	}
}

func repositoryRoot(t *testing.T) string { t.Helper(); return repositoryRootNoTest() }
func repositoryRootNoTest() string {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func zeroDigest() string { return "sha256:" + strings.Repeat("0", 64) }
func sha256Label(prefix, label string) string {
	digest := sha256.Sum256([]byte(label))
	return prefix + hex.EncodeToString(digest[:])
}

func observationWithCanonicalID(t *testing.T, observation any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	id, err := observationID(raw)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object["observation_id"] = id
	raw, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func journalBytes(t *testing.T, observations ...json.RawMessage) []byte {
	t.Helper()
	document := journal{Format: journalFormat, Mode: "SOURCE_ONLY_SYNTHETIC_FIXTURE", Projection: projection{
		Provider: "SUPABASE_POSTGRESQL", Relation: "PROJECTS", Mode: "SCHEMA_DESIGN_ONLY", Authority: "NONE", Rebuildable: true,
		Preserves: []string{"GRAPH_KIND", "SOURCE_STATUS", "EXACT_RESPONSE_DIGESTS", "RETURNED_CHAIN_AND_HEIGHT", "SAME_HEIGHT_CONFLICT_MULTIPLICITY"},
		Loses:     []string{"RAW_PAYLOAD_BYTES", "SOURCE_ENDPOINT", "CHAIN_PROOF", "SCIENTIFIC_TRUTH", "IDENTITY_AND_CONTROLLER", "CONSENT_AND_AUTHORITY"},
	}, Observations: observations}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
