package bridge

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	poca "github.com/zerone-chain/zerone/tools/poca-shadow/evaluate"
)

func TestPublishedPartialFixtureRefusesWithoutTreeEvidenceOrEconomics(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	receipt, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if receipt.Status != StatusRefused {
		t.Fatalf("status: want %q, got %q", StatusRefused, receipt.Status)
	}
	if !hasRefusal(receipt, "POCA_PROFILE_NOT_DECLARED_RATIFIED") ||
		!hasRefusal(receipt, "POCA_TARGET_STANDARD_BINDING_INCOMPLETE") {
		t.Fatalf("published fixture refusal reasons: %#v", receipt.RefusalReasons)
	}
	assertZeroBoundary(t, receipt)
	if receipt.Tree.RequiredAttainmentEvidence != "E3" ||
		receipt.PoCA.AttainedTier != "E2_CONFORMANT" ||
		receipt.NamespaceRelation != NamespaceRelation {
		t.Fatalf("namespace bindings drifted: tree=%#v poca=%#v relation=%q", receipt.Tree, receipt.PoCA, receipt.NamespaceRelation)
	}
	if receipt.PoCA.Assurance != AssuranceUnverified || receipt.Assurance != AssuranceUnverified {
		t.Fatalf("assurance drift: receipt=%q PoCA=%q", receipt.Assurance, receipt.PoCA.Assurance)
	}

	assertEqual(t, "consumption key", receipt.Source.ConsumptionKey, "sha256:8df1821e9ddded60ea67920eafb2e08cd4e0048bd1fc7b979510818a0c51b5a3")
	assertEqual(t, "receipt id", receipt.ReceiptID, "sha256:54cc3e31a69cba6892095c175d62383c27466b23a2f107b6c448db9eaaccfc8a")
	compact, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "compact receipt JSON sha256", fmt.Sprintf("%x", sha256.Sum256(compact)), "42cfb6549a23b4628fc1af706cd4b1472768d8ed7023e1ee2920f63238459e61")
}

func TestCandidateStillGrantsNothing(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	profile.Status = "DECLARED_RATIFIED"
	profile.Standards = append(profile.Standards, requiredPoCAStandards[1])
	certificate, err := poca.Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("prepare candidate certificate: %v", err)
	}
	request.PoCA.ProfileDigest = certificate.Profile.Digest
	request.PoCA.EvidenceBundleDigest = certificate.EvidenceBundleDigest
	request.PoCA.SubjectDigest = certificate.Subject.Digest

	receipt, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if receipt.Status != StatusCandidate || len(receipt.RefusalReasons) != 0 {
		t.Fatalf("candidate result: %#v", receipt)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"refusal_reasons":[]`) {
		t.Fatalf("candidate refusal_reasons must encode as an empty array: %s", encoded)
	}
	assertZeroBoundary(t, receipt)
	if receipt.PoCA.CrownStatus != "BLOCKED" {
		t.Fatalf("candidate test should not require or imply PoCA crown: %#v", receipt.PoCA)
	}
}

func TestRequestParsingRejectsDuplicateUnknownNullOmissionAndOversize(t *testing.T) {
	valid := mustRead(t, examplePath(t, "zerone-release-partial-v0.request.json"))

	duplicate := strings.Replace(
		string(valid),
		`"schema": "zerone.constructive-receipt-request/v0",`,
		`"schema": "zerone.constructive-receipt-request/v0", "schema": "zerone.constructive-receipt-request/v0",`,
		1,
	)
	if _, err := ParseRequest([]byte(duplicate)); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("expected duplicate-key rejection, got %v", err)
	}

	unknown := strings.Replace(string(valid), `"target": {`, `"unknown": true, "target": {`, 1)
	if _, err := ParseRequest([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatal(err)
	}
	document["source"].(map[string]any)["revision"] = nil
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRequest(encoded); err == nil || !strings.Contains(err.Error(), "null") {
		t.Fatalf("expected null rejection, got %v", err)
	}

	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatal(err)
	}
	delete(document["poca"].(map[string]any), "subject_digest")
	encoded, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRequest(encoded); err == nil || !strings.Contains(err.Error(), "subject_digest") {
		t.Fatalf("expected omission rejection, got %v", err)
	}

	if _, err := ParseRequest(make([]byte, maxRequestBytes+1)); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversize rejection, got %v", err)
	}
}

func TestTreeAndPoCAPinDriftFailClosed(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)

	t.Run("request tree document pin", func(t *testing.T) {
		changed := request
		changed.Target.TreeDocumentDigest = repeatedDigest("f")
		if _, err := Evaluate(changed, tree, profile, evidence); err == nil || !strings.Contains(err.Error(), "tree document digest pin mismatch") {
			t.Fatalf("expected tree document pin rejection, got %v", err)
		}
	})

	t.Run("tree bytes", func(t *testing.T) {
		changed := append([]byte(nil), tree...)
		changed = append(changed, '\n')
		if _, err := Evaluate(request, changed, profile, evidence); err == nil || !strings.Contains(err.Error(), "tree document digest mismatch") {
			t.Fatalf("expected exact tree-byte rejection, got %v", err)
		}
	})

	t.Run("profile", func(t *testing.T) {
		changed := request
		changed.PoCA.ProfileDigest = repeatedDigest("f")
		if _, err := Evaluate(changed, tree, profile, evidence); err == nil || !strings.Contains(err.Error(), "PoCA profile digest pin mismatch") {
			t.Fatalf("expected profile pin rejection, got %v", err)
		}
	})

	t.Run("evidence", func(t *testing.T) {
		changed := request
		changed.PoCA.EvidenceBundleDigest = repeatedDigest("f")
		if _, err := Evaluate(changed, tree, profile, evidence); err == nil || !strings.Contains(err.Error(), "PoCA evidence bundle digest pin mismatch") {
			t.Fatalf("expected evidence pin rejection, got %v", err)
		}
	})

	t.Run("subject", func(t *testing.T) {
		changed := request
		changed.PoCA.SubjectDigest = repeatedDigest("f")
		if _, err := Evaluate(changed, tree, profile, evidence); err == nil || !strings.Contains(err.Error(), "PoCA subject digest pin mismatch") {
			t.Fatalf("expected subject pin rejection, got %v", err)
		}
	})
}

func TestTreeParsingRejectsDuplicateAndOversizedJSON(t *testing.T) {
	tree := mustRead(t, rootPath(t, "dashboard", "public", "standards", "constructive-intelligence-tree.v1.json"))
	duplicate := strings.Replace(
		string(tree),
		`"schema": "zerone.constructive-intelligence-tree/v1",`,
		`"schema": "zerone.constructive-intelligence-tree/v1", "schema": "zerone.constructive-intelligence-tree/v1",`,
		1,
	)
	if _, err := parseTree([]byte(duplicate)); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("expected duplicate tree key rejection, got %v", err)
	}
	if _, err := parseTree(make([]byte, maxTreeBytes+1)); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized tree rejection, got %v", err)
	}
}

func TestNamespaceConfusionFailsClosed(t *testing.T) {
	requestBytes := mustRead(t, examplePath(t, "zerone-release-partial-v0.request.json"))
	confused := strings.Replace(
		string(requestBytes),
		`"tree_schema": "zerone.constructive-intelligence-tree/v1"`,
		`"tree_schema": "zerone.standard-profile/v0"`,
		1,
	)
	if _, err := ParseRequest([]byte(confused)); err == nil || !strings.Contains(err.Error(), "tree_schema") {
		t.Fatalf("expected tree/PoCA namespace confusion rejection, got %v", err)
	}

	injectedTier := strings.Replace(
		string(requestBytes),
		`"profile_id":`,
		`"attained_tier": "E3", "profile_id":`,
		1,
	)
	if _, err := ParseRequest([]byte(injectedTier)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected caller-selected tier rejection, got %v", err)
	}
}

func TestCanonicalPoCAInputReorderingKeepsReceiptStable(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	first, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}

	slices.Reverse(profile.Standards)
	slices.Reverse(profile.Requirements)
	slices.Reverse(profile.Nodes)
	for index := range profile.Nodes {
		slices.Reverse(profile.Nodes[index].Prerequisites)
		slices.Reverse(profile.Nodes[index].RequirementIDs)
	}
	slices.Reverse(evidence.Participants)
	slices.Reverse(evidence.Evidence)
	slices.Reverse(evidence.UnresolvedChallengeDigests)
	second, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatalf("receipt changed after canonical PoCA set reordering\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
}

func TestSourceTupleChangeChangesConsumptionAndReceiptKeysOnly(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	first, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	request.Source.Revision = "fixture-2"
	second, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if first.Source.ConsumptionKey == second.Source.ConsumptionKey {
		t.Fatal("source revision change did not change consumption key")
	}
	if first.ReceiptID == second.ReceiptID {
		t.Fatal("source revision change did not change receipt id")
	}
	if first.Tree != second.Tree || first.PoCA != second.PoCA ||
		first.Status != second.Status || first.EconomicEffect != second.EconomicEffect {
		t.Fatalf("source revision changed unrelated bindings\nfirst: %#v\nsecond: %#v", first, second)
	}
}

func TestSourceConsumptionKeyDependsOnlyOnSourceTuple(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	first, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	evidence.Evidence[0].SourceURI = "https://changed.example.invalid/receipt"
	certificate, err := poca.Evaluate(profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	request.PoCA.EvidenceBundleDigest = certificate.EvidenceBundleDigest
	second, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if first.Source.ConsumptionKey != second.Source.ConsumptionKey {
		t.Fatalf("evidence evolution changed tuple-only consumption key: %s != %s", first.Source.ConsumptionKey, second.Source.ConsumptionKey)
	}
	if first.ReceiptID == second.ReceiptID {
		t.Fatal("evidence evolution did not change receipt id")
	}
}

func TestPublishedReceiptAndZeroEscrowFixturesMatchExecutableOutput(t *testing.T) {
	request, tree, profile, evidence := publishedInputs(t)
	receipt, err := Evaluate(request, tree, profile, evidence)
	if err != nil {
		t.Fatal(err)
	}
	var expected Receipt
	if err := json.Unmarshal(
		mustRead(t, examplePath(t, "zerone-release-partial-v0.receipt.json")),
		&expected,
	); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(receipt, expected) {
		t.Fatalf("published receipt fixture drifted\nwant: %#v\ngot:  %#v", expected, receipt)
	}
	var escrow EscrowCompartments
	if err := json.Unmarshal(
		mustRead(t, examplePath(t, "zero-escrow-compartments-v0.json")),
		&escrow,
	); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(receipt.Escrow, escrow) {
		t.Fatalf("zero escrow fixture drifted\nwant: %#v\ngot:  %#v", escrow, receipt.Escrow)
	}
}

func publishedInputs(t *testing.T) (Request, []byte, poca.Profile, poca.EvidenceBundle) {
	t.Helper()
	request, err := ParseRequest(mustRead(t, examplePath(t, "zerone-release-partial-v0.request.json")))
	if err != nil {
		t.Fatal(err)
	}
	tree := mustRead(t, rootPath(t, "dashboard", "public", "standards", "constructive-intelligence-tree.v1.json"))
	profile, err := poca.ParseProfile(mustRead(t, rootPath(t, "docs", "examples", "poca", "slsa-build-l2-v0.profile.json")))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := poca.ParseEvidence(mustRead(t, rootPath(t, "docs", "examples", "poca", "zerone-release-partial-v0.evidence.json")))
	if err != nil {
		t.Fatal(err)
	}
	return request, tree, profile, evidence
}

func rootPath(t *testing.T, elements ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	return filepath.Join(append([]string{root}, elements...)...)
}

func examplePath(t *testing.T, name string) string {
	t.Helper()
	return rootPath(t, "docs", "examples", "constructive-receipts", name)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertZeroBoundary(t *testing.T, receipt Receipt) {
	t.Helper()
	if receipt.Qualification != EffectNone ||
		receipt.EconomicEffect != EffectNone ||
		receipt.AmountUzrn != "0" ||
		receipt.Tree.GrantedAttainmentEvidence != EffectNone {
		t.Fatalf("receipt crossed zero boundary: %#v", receipt)
	}
	escrow := receipt.Escrow
	if escrow.Schema != EscrowSchema ||
		escrow.Denom != "uzrn" ||
		escrow.FundedEscrowUzrn != "0" ||
		escrow.VerifiedCostBudgetUzrn != "0" ||
		escrow.ClaimantMilestoneTranchesUzrn != "0" ||
		escrow.ChallengeAndRemediationReserveUzrn != "0" ||
		escrow.ReviewerBudgetUzrn != "0" ||
		escrow.AdministrationAndFeeBudgetUzrn != "0" ||
		escrow.RefundableBalanceUzrn != "0" ||
		escrow.ConservationCheck != "ZERO_BALANCED" {
		t.Fatalf("non-zero or malformed escrow: %#v", escrow)
	}
	if receipt.Source.ConsumptionState != ConsumptionUnrecorded ||
		receipt.Source.ReplayProtection != ReplayProtectionNone {
		t.Fatalf("offline output claimed replay state: %#v", receipt.Source)
	}
}

func hasRefusal(receipt Receipt, code string) bool {
	for _, refusal := range receipt.RefusalReasons {
		if refusal.Code == code {
			return true
		}
	}
	return false
}

func repeatedDigest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}

func assertEqual(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s drift: want %s, got %s", name, want, got)
	}
}
