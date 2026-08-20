package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/agenttool-research-receipt/bridge"
)

const fixtureManifestSHA256 = "cf367bb39553567e86c43c0db48501802832396b2a3f681410aaac7c5e2221e8"

type crossLanguageFixtureManifest struct {
	Format     string `json:"_format"`
	Status     string `json:"status"`
	Source     fixtureSource
	Adapter    fixtureAdapter
	SharedPins fixtureSharedPins `json:"shared_pins"`
	Fixtures   []fixturePair
	Boundary   fixtureBoundary
}

type fixtureSource struct {
	Repository        string `json:"repository"`
	SourceRevision    string `json:"source_revision"`
	MainMergeRevision string `json:"main_merge_revision"`
	PullRequest       string `json:"pull_request"`
	SourceDirectory   string `json:"source_directory"`
	LicenseSPDX       string `json:"license_spdx"`
	LicenseSourcePath string `json:"license_source_path"`
	LicenseSourceSHA  string `json:"license_source_raw_sha256"`
	NoticeSourcePath  string `json:"notice_source_path"`
	NoticeSourceSHA   string `json:"notice_source_raw_sha256"`
	NoticeCopiedPath  string `json:"notice_copied_path"`
	NoticeCopiedSHA   string `json:"notice_copied_raw_sha256"`
	CopiedByteForByte bool   `json:"copied_byte_for_byte"`
}

type fixtureAdapter struct {
	Version        string `json:"version"`
	ReceiptSchema  string `json:"receipt_schema"`
	OutputEncoding string `json:"output_encoding"`
	Assurance      string `json:"assurance"`
	Status         string `json:"status"`
	EconomicEffect string `json:"economic_effect"`
	AmountUzrn     string `json:"amount_uzrn"`
}

type fixtureSharedPins struct {
	SettlementBundleFormat string `json:"settlement_bundle_format"`
	PublicProjectionFormat string `json:"public_projection_format"`
	StaticInteropProfileID string `json:"static_interop_profile_id"`
	StaticInteropRawSHA256 string `json:"static_interop_raw_sha256"`
	SixLedgerProfileID     string `json:"six_ledger_profile_id"`
	SixLedgerProfileDigest string `json:"six_ledger_profile_digest"`
	TreeSchema             string `json:"tree_schema"`
	TreeRawSHA256          string `json:"tree_raw_sha256"`
	TreeNodeID             string `json:"tree_node_id"`
	TreeNodeDigest         string `json:"tree_node_digest"`
}

type fixturePair struct {
	ID              string `json:"id"`
	Settlement      fixtureInput
	Projection      fixtureInput
	ExpectedReceipt fixtureReceipt `json:"expected_receipt"`
}

type fixtureInput struct {
	SourcePath      string `json:"source_path"`
	SourceRawSHA256 string `json:"source_raw_sha256"`
	CopiedPath      string `json:"copied_path"`
	CopiedRawSHA256 string `json:"copied_raw_sha256"`
	EnvelopeID      string `json:"envelope_id"`
}

type fixtureReceipt struct {
	ReceiptID             string `json:"receipt_id"`
	RawSHA256             string `json:"raw_sha256"`
	DeclaredResultKind    string `json:"declared_result_kind"`
	HighestEvidenceLevel  string `json:"highest_evidence_level"`
	SimulatedCreditAmount int64  `json:"simulated_credit_amount"`
}

type fixtureBoundary struct {
	ReciprocalCrossPinPresent bool `json:"reciprocal_cross_pin_present"`
	IntegrationReady          bool `json:"integration_ready"`
	ImportsAgentToolState     bool `json:"imports_agenttool_state"`
	CallsAgentTool            bool `json:"calls_agenttool"`
	MovesValue                bool `json:"moves_value"`
	AdjudicatesScience        bool `json:"adjudicates_science"`
}

func TestCheckedAgentToolFixturesRemainByteExactAndCrossLanguageCompatible(t *testing.T) {
	repositoryRoot := fixtureRepositoryRoot(t)
	manifestPath := filepath.Join(
		repositoryRoot,
		"docs",
		"examples",
		"agenttool-research-receipt",
		"fixture-manifest.v0.json",
	)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if actual := rawSHA256(manifestBytes); actual != fixtureManifestSHA256 {
		t.Fatalf("fixture manifest raw SHA-256 = %s, want %s", actual, fixtureManifestSHA256)
	}

	var manifest crossLanguageFixtureManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode fixture manifest: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("fixture manifest has trailing JSON: %v", err)
	}
	assertFixtureManifestPins(t, manifest)

	treePath := filepath.Join(
		repositoryRoot,
		"dashboard",
		"public",
		"standards",
		"constructive-intelligence-tree.v1.json",
	)
	for _, fixture := range manifest.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			settlementPath := verifyCopiedFixture(t, repositoryRoot, fixture.Settlement)
			projectionPath := verifyCopiedFixture(t, repositoryRoot, fixture.Projection)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if err := run([]string{
				"--settlement", settlementPath,
				"--projection", projectionPath,
				"--tree", treePath,
			}, &stdout, &stderr); err != nil {
				t.Fatalf("compile fixture: %v; stderr=%q", err, stderr.String())
			}
			if actual := rawSHA256(stdout.Bytes()); actual != fixture.ExpectedReceipt.RawSHA256 {
				t.Fatalf("receipt raw SHA-256 = %s, want %s", actual, fixture.ExpectedReceipt.RawSHA256)
			}

			var receipt bridge.Receipt
			if err := json.Unmarshal(stdout.Bytes(), &receipt); err != nil {
				t.Fatalf("decode receipt: %v", err)
			}
			if receipt.ReceiptID != fixture.ExpectedReceipt.ReceiptID ||
				receipt.Source.SettlementID != fixture.Settlement.EnvelopeID ||
				receipt.Source.ProjectionID != fixture.Projection.EnvelopeID ||
				receipt.Declaration.DeclaredResultKind != fixture.ExpectedReceipt.DeclaredResultKind ||
				receipt.Declaration.HighestEvidenceLevel == nil ||
				*receipt.Declaration.HighestEvidenceLevel != fixture.ExpectedReceipt.HighestEvidenceLevel ||
				receipt.Simulation.CreditAmount != fixture.ExpectedReceipt.SimulatedCreditAmount {
				t.Fatalf("compiled receipt drift: %#v", receipt)
			}
			if receipt.Status != bridge.StatusCandidate ||
				receipt.EconomicEffect != bridge.EffectNone ||
				receipt.AmountUzrn != "0" ||
				receipt.Effects != (bridge.ZeroneEffects{}) {
				t.Fatalf("fixture crossed the zero-effect boundary: %#v", receipt)
			}
		})
	}
}

func TestDocumentationScopesToolchainEffectsOutsideCompiledRuntime(t *testing.T) {
	repositoryRoot := fixtureRepositoryRoot(t)
	for _, relativePath := range []string{
		"tools/agenttool-research-receipt/README.md",
		"docs/specs/adapters/agenttool-research-receipt-v1.md",
	} {
		data, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatal(err)
		}
		copy := strings.Join(strings.Fields(string(data)), " ")
		for _, required := range []string{
			"already-compiled adapter process",
			"module metadata",
			"build or module caches",
			"module proxy or version-control remote",
			"separately controlled build",
		} {
			if !strings.Contains(copy, required) {
				t.Fatalf("%s does not disclose the toolchain/runtime boundary %q", relativePath, required)
			}
		}
	}
}

func assertFixtureManifestPins(t *testing.T, manifest crossLanguageFixtureManifest) {
	t.Helper()
	if manifest.Format != "zerone.agenttool-research-fixture-set/0.1" ||
		manifest.Status != "PHASE_A_STATIC_FIXTURE_ONLY" ||
		manifest.Source.Repository != "https://github.com/cambridgetcg/agenttool" ||
		manifest.Source.SourceRevision != "6a644b9e858b7d23bdea613d91412bf7310c2338" ||
		manifest.Source.MainMergeRevision != "55342fac97250898c2c4ea884f1a03bec1f8cc8c" ||
		manifest.Source.PullRequest != "https://github.com/cambridgetcg/agenttool/pull/335" ||
		manifest.Source.SourceDirectory != "packages/research-commons/examples/amplitude-bootstrap-garden" ||
		manifest.Source.LicenseSPDX != "Apache-2.0" ||
		manifest.Source.LicenseSourcePath != "packages/research-commons/LICENSE" ||
		manifest.Source.LicenseSourceSHA != "0536b51c54e477f03f1becf00eedeee82f6276f76f08c1b94d3a30632724eb15" ||
		manifest.Source.NoticeSourcePath != "packages/research-commons/NOTICE" ||
		manifest.Source.NoticeSourceSHA != "d03f1590ea4f829d90760ee163304191c0d36a4e283fc7c06da459e717ff3e44" ||
		manifest.Source.NoticeCopiedPath != "docs/examples/agenttool-research-receipt/NOTICE" ||
		manifest.Source.NoticeCopiedSHA != manifest.Source.NoticeSourceSHA ||
		!manifest.Source.CopiedByteForByte {
		t.Fatalf("AgentTool source provenance drift: %#v", manifest.Source)
	}
	noticeBytes, err := os.ReadFile(filepath.Join(fixtureRepositoryRoot(t), filepath.FromSlash(manifest.Source.NoticeCopiedPath)))
	if err != nil {
		t.Fatal(err)
	}
	if actual := rawSHA256(noticeBytes); actual != manifest.Source.NoticeCopiedSHA {
		t.Fatalf("vendored AgentTool NOTICE raw SHA-256 = %s, want %s", actual, manifest.Source.NoticeCopiedSHA)
	}
	if manifest.Adapter.Version != bridge.AdapterVersion ||
		manifest.Adapter.ReceiptSchema != bridge.ReceiptSchema ||
		manifest.Adapter.OutputEncoding != "UTF8_GO_JSON_INDENT_2_TRAILING_LF" ||
		manifest.Adapter.Assurance != bridge.Assurance ||
		manifest.Adapter.Status != bridge.StatusCandidate ||
		manifest.Adapter.EconomicEffect != bridge.EffectNone ||
		manifest.Adapter.AmountUzrn != "0" {
		t.Fatalf("adapter fixture boundary drift: %#v", manifest.Adapter)
	}
	if manifest.SharedPins.SettlementBundleFormat != bridge.SettlementFormat ||
		manifest.SharedPins.PublicProjectionFormat != bridge.PublicProjectionFormat ||
		manifest.SharedPins.StaticInteropProfileID != bridge.InteropProfileFormat ||
		manifest.SharedPins.StaticInteropRawSHA256 != "8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a" ||
		manifest.SharedPins.SixLedgerProfileID != bridge.LedgerProfileID ||
		manifest.SharedPins.SixLedgerProfileDigest != bridge.LedgerProfileHash ||
		manifest.SharedPins.TreeSchema != bridge.TreeSchema ||
		manifest.SharedPins.TreeRawSHA256 != bridge.TreeRawDigest ||
		manifest.SharedPins.TreeNodeID != bridge.TargetNodeID ||
		manifest.SharedPins.TreeNodeDigest != bridge.TargetNodeDigest {
		t.Fatalf("shared fixture pin drift: %#v", manifest.SharedPins)
	}
	if len(manifest.Fixtures) != 2 ||
		manifest.Fixtures[0].ID != "primary-e2-null" ||
		manifest.Fixtures[1].ID != "reviewer-e1-not-applicable" {
		t.Fatalf("fixture order or ids drifted: %#v", manifest.Fixtures)
	}
	if manifest.Boundary != (fixtureBoundary{}) {
		t.Fatalf("Phase A fixture claimed an effect or reciprocal pin: %#v", manifest.Boundary)
	}
}

func verifyCopiedFixture(t *testing.T, repositoryRoot string, fixture fixtureInput) string {
	t.Helper()
	if fixture.SourceRawSHA256 != fixture.CopiedRawSHA256 {
		t.Fatalf("source and copied SHA-256 differ: %#v", fixture)
	}
	path := filepath.Join(repositoryRoot, filepath.FromSlash(fixture.CopiedPath))
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if actual := rawSHA256(bytes); actual != fixture.CopiedRawSHA256 {
		t.Fatalf("copied fixture %s raw SHA-256 = %s, want %s", fixture.CopiedPath, actual, fixture.CopiedRawSHA256)
	}
	return path
}

func fixtureRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve fixture test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func rawSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
