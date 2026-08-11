package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestReportSucceedsWithDeclaredConflictsVisible(t *testing.T) {
	root := makeFixture(t)
	var output bytes.Buffer
	if code := run([]string{"report", "--root", root}, &output); code != exitOK {
		t.Fatalf("report exit = %d, want %d\n%s", code, exitOK, output.String())
	}
	var got report
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if got.Status != "CURRENT_SOURCE_COMPLETELY_CLASSIFIED" {
		t.Fatalf("status = %q", got.Status)
	}
	if got.ManifestSHA256 != canonicalManifestSHA256 || got.SourceAnchorsVerified != 17 {
		t.Fatalf("manifest/source seal not reported: %#v", got)
	}
	wantBlockers := expectedBlockerIDs()
	if !reflect.DeepEqual(got.BlockerIDs, wantBlockers) {
		t.Fatalf("blockers = %v, want %v", got.BlockerIDs, wantBlockers)
	}
	wantConsumers := []string{"alignment", "claiming-pot", "emergency", "gov", "knowledge", "qualification"}
	if !reflect.DeepEqual(got.DiscoveredCustomStakeConsumers, wantConsumers) {
		t.Fatalf("custom staking consumers = %v, want %v", got.DiscoveredCustomStakeConsumers, wantConsumers)
	}
	wantConstructors := []string{"custom-gov", "custom-staking", "knowledge", "ontology", "sdk-gov", "sdk-staking"}
	if !reflect.DeepEqual(got.DetectedAuthorityConstructors, wantConstructors) {
		t.Fatalf("constructors = %v, want %v", got.DetectedAuthorityConstructors, wantConstructors)
	}
	if strings.Contains(output.String(), root) {
		t.Fatal("deterministic report disclosed the caller-supplied root")
	}
}

func TestUndeclaredCustomStakingAdapterFailsClosed(t *testing.T) {
	root := makeFixture(t)
	appPath := filepath.Join(root, filepath.FromSlash("app/app.go"))
	appBytes := mustRead(t, appPath)
	appBytes = append(appBytes, []byte("\nfunc authorityGraphUndeclaredAdapter() {\n\t_ = zeronestakingkeeper.NewUndeclaredAdapter(app.ZeroneStakingKeeper)\n}\n")...)
	mustWrite(t, appPath, appBytes)

	m := fixtureManifest(t, root)
	for i := range m.SourceAnchors {
		if m.SourceAnchors[i].ID == "app-wiring" {
			m.SourceAnchors[i].SHA256 = sha256Hex(appBytes)
		}
	}
	writeFixtureManifest(t, root, m)

	output, code := runFixtureMutation(root, "report")
	if code != exitCheckFailed {
		t.Fatalf("mutated report exit = %d, want %d\n%s", code, exitCheckFailed, output)
	}
	assertIssue(t, output, "SOURCE_CUSTOM_STAKING_ADAPTER_SET_MISMATCH")
}

func TestChangedSourceAnchorFailsClosed(t *testing.T) {
	root := makeFixture(t)
	anchorPath := filepath.Join(root, filepath.FromSlash("x/gov/keeper/abci.go"))
	data := append(mustRead(t, anchorPath), '\n')
	mustWrite(t, anchorPath, data)

	var output bytes.Buffer
	if code := run([]string{"report", "--root", root}, &output); code != exitCheckFailed {
		t.Fatalf("changed anchor exit = %d, want %d\n%s", code, exitCheckFailed, output.String())
	}
	assertIssue(t, output.String(), "SOURCE_ANCHOR_SHA256_MISMATCH")
}

func TestTargetAndLiveRelabelsFailClosed(t *testing.T) {
	t.Run("live scope", func(t *testing.T) {
		root := makeFixture(t)
		m := fixtureManifest(t, root)
		m.Status = "LIVE_NETWORK"
		writeFixtureManifest(t, root, m)
		output, code := runFixtureMutation(root, "report")
		if code != exitCheckFailed {
			t.Fatalf("live relabel exit = %d, want %d\n%s", code, exitCheckFailed, output)
		}
		assertIssue(t, output, "MANIFEST_STATUS_INVALID")
	})

	t.Run("target module presented as current", func(t *testing.T) {
		root := makeFixture(t)
		m := fixtureManifest(t, root)
		for i := range m.Nodes {
			if m.Nodes[i].ID == "controller" {
				m.Nodes[i].Implementation = "PRESENT_AND_TARGET"
			}
		}
		writeFixtureManifest(t, root, m)
		output, code := runFixtureMutation(root, "report")
		if code != exitCheckFailed {
			t.Fatalf("target relabel exit = %d, want %d\n%s", code, exitCheckFailed, output)
		}
		assertIssue(t, output, "AUTHORITY_NODE_CLASSIFICATION_INVALID")
	})
}

func TestForbiddenAcceptedTargetWealthPathFailsClosed(t *testing.T) {
	root := makeFixture(t)
	m := fixtureManifest(t, root)
	mutated := false
	for i := range m.Edges {
		if m.Edges[i].ID == "target-auth-to-controller" {
			m.Edges[i].From = "sdk-staking"
			m.Edges[i].To = "sdk-gov"
			mutated = true
		}
	}
	if !mutated {
		t.Fatal("target edge fixture not found")
	}
	writeFixtureManifest(t, root, m)

	output, code := runFixtureMutation(root, "report")
	if code != exitCheckFailed {
		t.Fatalf("forbidden target path exit = %d, want %d\n%s", code, exitCheckFailed, output)
	}
	assertIssue(t, output, "FORBIDDEN_INFLUENCE_PATH_COUNT_MISMATCH")
}

func TestKarmaForbiddenTargetsCannotOmitMoneyPayoutOrRank(t *testing.T) {
	for _, omitted := range []string{"bank-balance", "vesting-rewards", "verifier-profile"} {
		t.Run(omitted, func(t *testing.T) {
			root := makeFixture(t)
			m := fixtureManifest(t, root)
			for i := range m.ForbiddenInfluence {
				if m.ForbiddenInfluence[i].ID == "karma-to-authority" {
					m.ForbiddenInfluence[i].Targets = removeString(m.ForbiddenInfluence[i].Targets, omitted)
				}
			}
			writeFixtureManifest(t, root, m)

			output, code := runFixtureMutation(root, "report")
			if code != exitCheckFailed {
				t.Fatalf("omitting %s exit = %d, want %d\n%s", omitted, code, exitCheckFailed, output)
			}
			assertIssue(t, output, "FORBIDDEN_INFLUENCE_CLASSIFICATION_INVALID")
		})
	}
}

func TestKnowledgeSlashAuthorityDirectionCannotBeCollapsed(t *testing.T) {
	root := makeFixture(t)
	m := fixtureManifest(t, root)
	for i := range m.Edges {
		if m.Edges[i].ID == "current-knowledge-to-custom-stake-slash" {
			m.Edges[i].From = "custom-staking"
			m.Edges[i].To = "knowledge"
		}
	}
	writeFixtureManifest(t, root, m)

	output, code := runFixtureMutation(root, "report")
	if code != exitCheckFailed {
		t.Fatalf("collapsed slash direction exit = %d, want %d\n%s", code, exitCheckFailed, output)
	}
	assertIssue(t, output, "CURRENT_AUTHORITY_EDGE_CLASSIFICATION_INVALID")
}

func TestEffectfulTargetEdgeCannotBeRelabelledAsReference(t *testing.T) {
	root := makeFixture(t)
	m := fixtureManifest(t, root)
	for i := range m.Edges {
		if m.Edges[i].ID == "target-controller-to-electorate" {
			m.Edges[i].Effect = "REFERENCE_RELATION"
		}
	}
	writeFixtureManifest(t, root, m)

	output, code := runFixtureMutation(root, "report")
	if code != exitCheckFailed {
		t.Fatalf("target effect relabel exit = %d, want %d\n%s", code, exitCheckFailed, output)
	}
	assertIssue(t, output, "TARGET_AUTHORITY_EDGE_CLASSIFICATION_INVALID")
}

func TestTargetReachabilityExcludesNonInfluenceRelations(t *testing.T) {
	for _, effect := range []string{"REFERENCE_RELATION", "EVIDENCE_RELATION", "RETIREMENT_RELATION"} {
		t.Run(effect, func(t *testing.T) {
			edges := []authorityEdge{{
				From:   "wealth",
				To:     "policy",
				Scope:  "ACCEPTED_TARGET",
				Effect: effect,
			}}
			if hasDirectedInfluencePath(acceptedTargetInfluenceAdjacency(edges), "wealth", "policy") {
				t.Fatalf("%s must not carry target influence", effect)
			}
		})
	}
	effectful := []authorityEdge{{
		From:   "wealth",
		To:     "policy",
		Scope:  "ACCEPTED_TARGET",
		Effect: "CONTROL_RELATION",
	}}
	if !hasDirectedInfluencePath(acceptedTargetInfluenceAdjacency(effectful), "wealth", "policy") {
		t.Fatal("effectful accepted-target edge must carry target influence")
	}
}

func TestTargetGateRefusesDeterministically(t *testing.T) {
	root := makeFixture(t)
	var first bytes.Buffer
	if code := run([]string{"target-gate", "--root", root}, &first); code != exitTargetRefused {
		t.Fatalf("target-gate exit = %d, want %d\n%s", code, exitTargetRefused, first.String())
	}
	var second bytes.Buffer
	if code := run([]string{"target-gate", "--root", root}, &second); code != exitTargetRefused {
		t.Fatalf("second target-gate exit = %d, want %d\n%s", code, exitTargetRefused, second.String())
	}
	if first.String() != second.String() {
		t.Fatal("target-gate output is not deterministic")
	}
	var got report
	if err := json.Unmarshal(first.Bytes(), &got); err != nil {
		t.Fatalf("decode target-gate report: %v", err)
	}
	if got.Status != "TARGET_GATE_REFUSED" || !got.TargetGateMustExitNonZero {
		t.Fatalf("unexpected target-gate result: status=%q mustRefuse=%t", got.Status, got.TargetGateMustExitNonZero)
	}
	if !reflect.DeepEqual(got.BlockerIDs, expectedBlockerIDs()) {
		t.Fatalf("target-gate blockers = %v", got.BlockerIDs)
	}
	if len(got.UnevidencedActivationGateIDs) != 38 {
		t.Fatalf("unevidenced gates = %d, want 38", len(got.UnevidencedActivationGateIDs))
	}
}

func TestPresentTargetOnlyDirectoryFailsClosed(t *testing.T) {
	root := makeFixture(t)
	if err := os.MkdirAll(filepath.Join(root, "x", "controller"), 0o755); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := run([]string{"report", "--root", root}, &output); code != exitCheckFailed {
		t.Fatalf("present target module exit = %d, want %d\n%s", code, exitCheckFailed, output.String())
	}
	assertIssue(t, output.String(), "TARGET_ONLY_MODULE_PRESENT")
}

func makeFixture(t *testing.T) string {
	t.Helper()
	sourceRoot := repositoryRoot(t)
	root := t.TempDir()
	manifestBytes := mustRead(t, filepath.Join(sourceRoot, filepath.FromSlash(manifestPath)))
	mustWriteRelative(t, root, manifestPath, manifestBytes)
	var m manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		t.Fatalf("decode fixture manifest: %v", err)
	}
	for _, anchor := range m.SourceAnchors {
		mustWriteRelative(t, root, anchor.Path, mustRead(t, filepath.Join(sourceRoot, filepath.FromSlash(anchor.Path))))
	}
	return root
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func fixtureManifest(t *testing.T, root string) manifest {
	t.Helper()
	var m manifest
	if err := json.Unmarshal(mustRead(t, filepath.Join(root, filepath.FromSlash(manifestPath))), &m); err != nil {
		t.Fatalf("decode fixture manifest: %v", err)
	}
	return m
}

func writeFixtureManifest(t *testing.T, root string, m manifest) {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("encode fixture manifest: %v", err)
	}
	data = append(data, '\n')
	mustWrite(t, filepath.Join(root, filepath.FromSlash(manifestPath)), data)
}

func runFixtureMutation(root, mode string) (string, int) {
	var output bytes.Buffer
	code := runWithOptions([]string{mode, "--root", root}, &output, checkOptions{enforceCanonicalManifest: false})
	return output.String(), code
}

func expectedBlockerIDs() []string {
	return []string{
		"alternate-quarantine-surfaces",
		"custom-staking-runtime-consumers",
		"direct-fact-adoption",
		"dual-domain-registries",
		"dual-governance-systems",
		"dual-staking-ledgers",
		"legacy-research-disbursement",
	}
}

func removeString(values []string, omitted string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != omitted {
			result = append(result, value)
		}
	}
	return result
}

func assertIssue(t *testing.T, output, wanted string) {
	t.Helper()
	var failure failureReport
	if err := json.Unmarshal([]byte(output), &failure); err != nil {
		t.Fatalf("decode failure report: %v\n%s", err, output)
	}
	for _, issue := range failure.Issues {
		if issue.ID == wanted {
			return
		}
	}
	t.Fatalf("issue %s not found in %#v", wanted, failure.Issues)
}

func mustWriteRelative(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	mustWrite(t, path, data)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture file: %v", err)
	}
	return data
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}
