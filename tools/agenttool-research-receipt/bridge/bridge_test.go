package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateCreatesZeroEffectStructuralCandidate(t *testing.T) {
	settlementBytes, projectionBytes := validInputs(t, "NULL")
	receipt, err := Evaluate(settlementBytes, projectionBytes, treeBytes(t))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if receipt.Status != StatusCandidate || receipt.Assurance != Assurance {
		t.Fatalf("unexpected candidate header: %#v", receipt)
	}
	if receipt.Declaration.DeclaredResultKind != "NULL" || receipt.Declaration.ResultAuthority != ResultAuthority {
		t.Fatalf("declared result boundary drift: %#v", receipt.Declaration)
	}
	if receipt.Simulation.CreditAmount != 25 || receipt.Simulation.CreditUnit != SimulatedUnit {
		t.Fatalf("simulation binding drift: %#v", receipt.Simulation)
	}
	if receipt.CrossLedgerRelation != NoEquivalence ||
		receipt.EconomicEffect != EffectNone ||
		receipt.AmountUzrn != "0" ||
		receipt.Effects != (ZeroneEffects{}) {
		t.Fatalf("zero-effect boundary crossed: %#v", receipt)
	}
	if receipt.LedgerBoundary.ProfileDigest != LedgerProfileHash ||
		receipt.LedgerBoundary.SharedUnit ||
		receipt.LedgerBoundary.CrossLedgerArithmetic ||
		receipt.LedgerBoundary.CrossLedgerConversion ||
		receipt.LedgerBoundary.CrossLedgerInference {
		t.Fatalf("six-ledger boundary drift: %#v", receipt.LedgerBoundary)
	}
	if receipt.Interop.Format != InteropProfileFormat ||
		receipt.Interop.RawDigest != InteropProfileDigest ||
		receipt.Interop.IntegrationStatus != InteropProfileStatus ||
		receipt.Interop.Imported || receipt.Interop.Activated {
		t.Fatalf("interop profile boundary drift: %#v", receipt.Interop)
	}
	limitations := strings.Join(receipt.Limitations, " ")
	for _, required := range []string{
		"caller-supplied prior_state transition",
		"not signatures or canonical heads",
		"no provenance, trusted time, global ordering, or prevention of old-state forks",
	} {
		if !strings.Contains(limitations, required) {
			t.Fatalf("missing lifecycle limitation %q: %#v", required, receipt.Limitations)
		}
	}
}

func TestEveryDeclaredResultDirectionHasIdenticalAdapterEconomics(t *testing.T) {
	for _, kind := range []string{"POSITIVE", "NEGATIVE", "NULL", "INCONCLUSIVE", "NOT_APPLICABLE"} {
		t.Run(kind, func(t *testing.T) {
			settlementBytes, projectionBytes := validInputs(t, kind)
			receipt, err := Evaluate(settlementBytes, projectionBytes, treeBytes(t))
			if err != nil {
				t.Fatal(err)
			}
			if receipt.Simulation.CreditAmount != 25 ||
				receipt.Simulation.PaymentCondition != PaymentCondition ||
				receipt.EconomicEffect != EffectNone ||
				receipt.AmountUzrn != "0" {
				t.Fatalf("result direction changed adapter economics: %#v", receipt)
			}
		})
	}
}

func TestOmittedAndTrueBoundaryFieldsFailClosed(t *testing.T) {
	settlementBytes, projectionBytes := validInputs(t, "INCONCLUSIVE")
	var settlement map[string]any
	if err := json.Unmarshal(settlementBytes, &settlement); err != nil {
		t.Fatal(err)
	}
	settlementBody := settlement["settlement"].(map[string]any)
	settlementEffects := settlementBody["effects"].(map[string]any)
	delete(settlementEffects, "mainnet")
	mutated, err := json.Marshal(settlement)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Evaluate(mutated, projectionBytes, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "exactly") {
		t.Fatalf("omitted false effect must fail closed, got %v", err)
	}

	settlementBytes, projectionBytes = validInputs(t, "INCONCLUSIVE")
	var projection map[string]any
	if err := json.Unmarshal(projectionBytes, &projection); err != nil {
		t.Fatal(err)
	}
	projectionBody := projection["projection"].(map[string]any)
	projectionBody["boundaries"].(map[string]any)["scientific_correctness_determined"] = true
	mutated, err = json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Evaluate(settlementBytes, mutated, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("true correctness boundary must fail closed, got %v", err)
	}
}

func TestCrossPinsAndPilotCeilingFailClosed(t *testing.T) {
	settlementBytes, projectionBytes := validInputs(t, "NEGATIVE")

	var projection map[string]any
	if err := json.Unmarshal(projectionBytes, &projection); err != nil {
		t.Fatal(err)
	}
	projectionBody := projection["projection"].(map[string]any)
	projectionBody["six_ledger_boundary"].(map[string]any)["profile_digest"] = repeatedDigest("f")
	mutated := readdressProjection(t, projectionBody)
	if _, err := Evaluate(settlementBytes, mutated, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "six_ledger_boundary") {
		t.Fatalf("ledger drift must fail closed, got %v", err)
	}

	if err := json.Unmarshal(projectionBytes, &projection); err != nil {
		t.Fatal(err)
	}
	projectionBody = projection["projection"].(map[string]any)
	projectionBody["highest_evidence_level"] = "E3"
	mutated = readdressProjection(t, projectionBody)
	if _, err := Evaluate(settlementBytes, mutated, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "E2 pilot ceiling") {
		t.Fatalf("E3 projection must fail closed, got %v", err)
	}

	changedTree := append(append([]byte(nil), treeBytes(t)...), '\n')
	if _, err := Evaluate(settlementBytes, projectionBytes, changedTree); err == nil || !strings.Contains(err.Error(), "tree raw digest mismatch") {
		t.Fatalf("Tree byte drift must fail closed, got %v", err)
	}

	if err := json.Unmarshal(projectionBytes, &projection); err != nil {
		t.Fatal(err)
	}
	projectionBody = projection["projection"].(map[string]any)
	projectionBody["public_evidence_receipt_ids"] = append(
		projectionBody["public_evidence_receipt_ids"].([]any),
		repeatedDigest("f"),
	)
	mutated = readdressProjection(t, projectionBody)
	if _, err := Evaluate(settlementBytes, mutated, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "exactly equal") {
		t.Fatalf("projection-wide evidence must not inflate one settlement's level, got %v", err)
	}
}

func TestAmbiguousOrUnsafeWireJSONFailsClosed(t *testing.T) {
	settlementBytes, projectionBytes := validInputs(t, "NULL")

	duplicate := []byte(strings.Replace(
		string(settlementBytes),
		"{",
		`{"settlement_id":"`+repeatedDigest("0")+`",`,
		1,
	))
	if _, err := Evaluate(duplicate, projectionBytes, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate decoded key must fail closed, got %v", err)
	}

	malformedUTF8 := append(append([]byte(nil), settlementBytes...), 0xff)
	if _, err := Evaluate(malformedUTF8, projectionBytes, treeBytes(t)); err == nil {
		t.Fatal("malformed UTF-8 must fail closed")
	}

	var envelope map[string]any
	if err := json.Unmarshal(settlementBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	body := envelope["settlement"].(map[string]any)
	body["simulated_credit"].(map[string]any)["amount"] = float64(9_007_199_254_740_992)
	unsafeInteger := readdressSettlement(t, body)
	if _, err := Evaluate(unsafeInteger, projectionBytes, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "safe integer") {
		t.Fatalf("unsafe integer must fail closed, got %v", err)
	}

	if err := json.Unmarshal(settlementBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	body = envelope["settlement"].(map[string]any)
	body["simulated_credit"].(map[string]any)["amount"] = -1
	negative := readdressSettlement(t, body)
	if _, err := Evaluate(negative, projectionBytes, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("negative simulated credit must fail closed, got %v", err)
	}

	if err := json.Unmarshal(settlementBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	body = envelope["settlement"].(map[string]any)
	body["payment_condition"] = "SIMULATED_DELIVERY_ONLY\u007f"
	nonPrintable := readdressSettlement(t, body)
	if _, err := Evaluate(nonPrintable, projectionBytes, treeBytes(t)); err == nil || !strings.Contains(err.Error(), "printable ASCII") {
		t.Fatalf("non-printable string must fail closed, got %v", err)
	}
}

func validInputs(t *testing.T, resultKind string) ([]byte, []byte) {
	t.Helper()
	receiptID := repeatedDigest("1")
	settlementBody := settlementBody{
		Format:             SettlementFormat,
		CaseID:             repeatedDigest("2"),
		CommitmentID:       repeatedDigest("3"),
		ConsumedReceiptIDs: []string{receiptID},
		DeclaredResultKind: resultKind,
		Effects:            zeroEffects{},
		MilestoneID:        repeatedDigest("4"),
		PaymentCondition:   PaymentCondition,
		ResultAuthority:    ResultAuthority,
		SimulatedCredit:    simulatedCredit{Amount: 25, Unit: SimulatedUnit},
	}
	settlementRaw, err := json.Marshal(settlementBody)
	if err != nil {
		t.Fatal(err)
	}
	settlementCanonical, err := canonicalRaw(settlementRaw)
	if err != nil {
		t.Fatal(err)
	}
	settlement := settlementEnvelope{
		SettlementID: domainID(SettlementFormat, settlementCanonical),
		Settlement:   settlementBody,
	}
	settlementBytes, err := json.Marshal(settlement)
	if err != nil {
		t.Fatal(err)
	}

	referenceWithoutID := struct {
		Format           string `json:"_format"`
		AnchorKind       string `json:"anchor_kind"`
		Canonicalization string `json:"canonicalization"`
		LiveFact         bool   `json:"live_fact"`
		NetworkObserved  bool   `json:"network_observed"`
		NodeDigest       string `json:"node_digest"`
		NodeID           string `json:"node_id"`
		ResultAuthority  string `json:"result_authority"`
		RewardBearing    bool   `json:"reward_bearing"`
		TreeRawSHA256    string `json:"tree_raw_sha256"`
		TreeSchema       string `json:"tree_schema"`
	}{
		Format:           NodeRefFormat,
		AnchorKind:       StaticAnchorKind,
		Canonicalization: NodeCanonicalizer,
		LiveFact:         false,
		NetworkObserved:  false,
		NodeDigest:       TargetNodeDigest,
		NodeID:           TargetNodeID,
		ResultAuthority:  ResultAuthority,
		RewardBearing:    false,
		TreeRawSHA256:    TreeRawDigest,
		TreeSchema:       TreeSchema,
	}
	referenceRaw, err := json.Marshal(referenceWithoutID)
	if err != nil {
		t.Fatal(err)
	}
	referenceCanonical, err := canonicalRaw(referenceRaw)
	if err != nil {
		t.Fatal(err)
	}
	reference := nodeRef{
		Format:           referenceWithoutID.Format,
		AnchorKind:       referenceWithoutID.AnchorKind,
		Canonicalization: referenceWithoutID.Canonicalization,
		LiveFact:         referenceWithoutID.LiveFact,
		NetworkObserved:  referenceWithoutID.NetworkObserved,
		NodeDigest:       referenceWithoutID.NodeDigest,
		NodeID:           referenceWithoutID.NodeID,
		NodeRefID:        domainID(NodeRefFormat, referenceCanonical),
		ResultAuthority:  referenceWithoutID.ResultAuthority,
		RewardBearing:    referenceWithoutID.RewardBearing,
		TreeRawSHA256:    referenceWithoutID.TreeRawSHA256,
		TreeSchema:       referenceWithoutID.TreeSchema,
	}
	highest := "E2"
	projectionBody := projectionBody{
		Format:                    PublicProjectionFormat,
		Boundaries:                projectionBoundaries{},
		CaseID:                    settlementBody.CaseID,
		DisclosureLane:            DisclosureLane,
		Effects:                   zeroEffects{},
		HighestEvidenceLevel:      &highest,
		NodeRef:                   reference,
		PublicArtifactRevisionIDs: []string{repeatedDigest("5")},
		PublicEvidenceReceiptIDs:  []string{receiptID},
		ResultAuthority:           ResultAuthority,
		SixLedgerBoundary:         sixLedgerBoundary{ProfileDigest: LedgerProfileHash, ProfileID: LedgerProfileID},
		SettlementBundleIDs:       []string{settlement.SettlementID},
		Status:                    ProjectionShadow,
	}
	projectionRaw, err := json.Marshal(projectionBody)
	if err != nil {
		t.Fatal(err)
	}
	projectionCanonical, err := canonicalRaw(projectionRaw)
	if err != nil {
		t.Fatal(err)
	}
	projection := projectionEnvelope{
		Projection:   projectionBody,
		ProjectionID: domainID(PublicProjectionFormat, projectionCanonical),
	}
	projectionBytes, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	return settlementBytes, projectionBytes
}

func readdressProjection(t *testing.T, body map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	envelope := map[string]any{
		"projection":    body,
		"projection_id": domainID(PublicProjectionFormat, canonical),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func readdressSettlement(t *testing.T, body map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	envelope := map[string]any{
		"settlement":    body,
		"settlement_id": domainID(SettlementFormat, canonical),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func treeBytes(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "dashboard", "public", "standards", "constructive-intelligence-tree.v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repeatedDigest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}
