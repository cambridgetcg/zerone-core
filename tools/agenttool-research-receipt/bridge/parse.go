package bridge

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

var (
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	resultKinds   = map[string]struct{}{
		"INCONCLUSIVE":   {},
		"NEGATIVE":       {},
		"NOT_APPLICABLE": {},
		"NULL":           {},
		"POSITIVE":       {},
	}
	allowedPilotLevels = map[string]struct{}{
		"E0": {},
		"E1": {},
		"E2": {},
	}
)

func parseSettlement(data []byte) (settlementEnvelope, error) {
	outer, err := requireExactFields(data, "$", "settlement_id", "settlement")
	if err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement envelope: %w", err)
	}
	bodyFields, err := requireExactFields(
		outer["settlement"],
		"$.settlement",
		"_format",
		"case_id",
		"commitment_id",
		"consumed_receipt_ids",
		"declared_result_kind",
		"effects",
		"milestone_id",
		"payment_condition",
		"result_authority",
		"simulated_credit",
	)
	if err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement envelope: %w", err)
	}
	if _, err := requireZeroEffectFields(bodyFields["effects"], "$.settlement.effects"); err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement envelope: %w", err)
	}
	if _, err := requireExactFields(bodyFields["simulated_credit"], "$.settlement.simulated_credit", "amount", "unit"); err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement envelope: %w", err)
	}
	var raw struct {
		SettlementID string          `json:"settlement_id"`
		Settlement   json.RawMessage `json:"settlement"`
	}
	if err := decodeClosed(data, &raw); err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement envelope: %w", err)
	}
	var body settlementBody
	if err := decodeClosed(raw.Settlement, &body); err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement body: %w", err)
	}
	settlement := settlementEnvelope{SettlementID: raw.SettlementID, Settlement: body}
	if err := validateSettlement(settlement, raw.Settlement); err != nil {
		return settlementEnvelope{}, fmt.Errorf("settlement: %w", err)
	}
	return settlement, nil
}

func validateSettlement(settlement settlementEnvelope, rawBody json.RawMessage) error {
	body := settlement.Settlement
	if body.Format != SettlementFormat {
		return fmt.Errorf("_format must be %q", SettlementFormat)
	}
	for path, value := range map[string]string{
		"settlement_id": settlement.SettlementID,
		"case_id":       body.CaseID,
		"commitment_id": body.CommitmentID,
		"milestone_id":  body.MilestoneID,
	} {
		if err := validateDigest(path, value); err != nil {
			return err
		}
	}
	if _, ok := resultKinds[body.DeclaredResultKind]; !ok {
		return fmt.Errorf("declared_result_kind is not supported")
	}
	if body.Effects != (zeroEffects{}) {
		return fmt.Errorf("every settlement effect must remain false")
	}
	if body.PaymentCondition != PaymentCondition {
		return fmt.Errorf("payment_condition must be %q", PaymentCondition)
	}
	if body.ResultAuthority != ResultAuthority {
		return fmt.Errorf("result_authority must be %q", ResultAuthority)
	}
	if body.SimulatedCredit.Unit != SimulatedUnit {
		return fmt.Errorf("simulated_credit.unit must be %q", SimulatedUnit)
	}
	if body.SimulatedCredit.Amount < 0 || body.SimulatedCredit.Amount > maxSafeInteger {
		return fmt.Errorf("simulated_credit.amount must be a non-negative safe integer")
	}
	if err := validateSortedUniqueDigests("consumed_receipt_ids", body.ConsumedReceiptIDs, false); err != nil {
		return err
	}
	canonical, err := canonicalRaw(rawBody)
	if err != nil {
		return err
	}
	expectedID := domainID(SettlementFormat, canonical)
	if settlement.SettlementID != expectedID {
		return fmt.Errorf("settlement_id mismatch: expected %s, got %s", expectedID, settlement.SettlementID)
	}
	return nil
}

func parseProjection(data []byte) (projectionEnvelope, error) {
	outer, err := requireExactFields(data, "$", "projection", "projection_id")
	if err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	bodyFields, err := requireExactFields(
		outer["projection"],
		"$.projection",
		"_format",
		"boundaries",
		"case_id",
		"disclosure_lane",
		"effects",
		"highest_evidence_level",
		"node_ref",
		"public_artifact_revision_ids",
		"public_evidence_receipt_ids",
		"result_authority",
		"six_ledger_boundary",
		"settlement_bundle_ids",
		"status",
	)
	if err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	if _, err := requireExactFields(
		bodyFields["boundaries"],
		"$.projection.boundaries",
		"authoritative",
		"private_locator_included",
		"raw_evidence_included",
		"scientific_correctness_determined",
	); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	if _, err := requireZeroEffectFields(bodyFields["effects"], "$.projection.effects"); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	if _, err := requireExactFields(
		bodyFields["six_ledger_boundary"],
		"$.projection.six_ledger_boundary",
		"profile_digest",
		"profile_id",
	); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	if _, err := requireExactFields(
		bodyFields["node_ref"],
		"$.projection.node_ref",
		"_format",
		"anchor_kind",
		"canonicalization",
		"live_fact",
		"network_observed",
		"node_digest",
		"node_id",
		"node_ref_id",
		"result_authority",
		"reward_bearing",
		"tree_raw_sha256",
		"tree_schema",
	); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	var raw struct {
		Projection   json.RawMessage `json:"projection"`
		ProjectionID string          `json:"projection_id"`
	}
	if err := decodeClosed(data, &raw, "$.projection.highest_evidence_level"); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection envelope: %w", err)
	}
	var body projectionBody
	if err := decodeClosed(raw.Projection, &body, "$.highest_evidence_level"); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection body: %w", err)
	}
	projection := projectionEnvelope{Projection: body, ProjectionID: raw.ProjectionID}
	if err := validateProjection(projection, raw.Projection); err != nil {
		return projectionEnvelope{}, fmt.Errorf("public projection: %w", err)
	}
	return projection, nil
}

func requireZeroEffectFields(raw json.RawMessage, path string) (map[string]json.RawMessage, error) {
	return requireExactFields(
		raw,
		path,
		"agenttool_api_write",
		"agenttool_database_write",
		"authority",
		"bridge",
		"burn",
		"chain_write",
		"consent",
		"cross_ledger_equivalence",
		"economic",
		"escrow",
		"external_value",
		"governance",
		"identity",
		"identity_equivalence",
		"knowledge_admission",
		"hosted_route",
		"mainnet",
		"mint",
		"network",
		"payout",
		"qualification",
		"reputation",
		"reward",
		"scientific_adjudication",
		"transfer",
		"wallet",
		"zrn",
		"zerone_read",
		"zerone_write",
	)
}

func validateProjection(projection projectionEnvelope, rawBody json.RawMessage) error {
	body := projection.Projection
	if body.Format != PublicProjectionFormat {
		return fmt.Errorf("_format must be %q", PublicProjectionFormat)
	}
	for path, value := range map[string]string{
		"projection_id": projection.ProjectionID,
		"case_id":       body.CaseID,
	} {
		if err := validateDigest(path, value); err != nil {
			return err
		}
	}
	if body.Boundaries != (projectionBoundaries{}) {
		return fmt.Errorf("every public-projection boundary must remain false")
	}
	if body.DisclosureLane != DisclosureLane {
		return fmt.Errorf("disclosure_lane must be %q", DisclosureLane)
	}
	if body.Effects != (zeroEffects{}) {
		return fmt.Errorf("every public-projection effect must remain false")
	}
	if body.HighestEvidenceLevel != nil {
		if _, ok := allowedPilotLevels[*body.HighestEvidenceLevel]; !ok {
			return fmt.Errorf("highest_evidence_level exceeds the E2 pilot ceiling")
		}
	}
	if body.ResultAuthority != ResultAuthority {
		return fmt.Errorf("result_authority must be %q", ResultAuthority)
	}
	if body.SixLedgerBoundary.ProfileID != LedgerProfileID ||
		body.SixLedgerBoundary.ProfileDigest != LedgerProfileHash {
		return fmt.Errorf("six_ledger_boundary must match the exact shared RC-0.1 profile")
	}
	if body.Status != ProjectionShadow {
		return fmt.Errorf("status must be %q", ProjectionShadow)
	}
	if err := validateNodeRef(body.NodeRef); err != nil {
		return err
	}
	for path, values := range map[string][]string{
		"public_artifact_revision_ids": body.PublicArtifactRevisionIDs,
		"public_evidence_receipt_ids":  body.PublicEvidenceReceiptIDs,
		"settlement_bundle_ids":        body.SettlementBundleIDs,
	} {
		if err := validateSortedUniqueDigests(path, values, false); err != nil {
			return err
		}
	}
	canonical, err := canonicalRaw(rawBody)
	if err != nil {
		return err
	}
	expectedID := domainID(PublicProjectionFormat, canonical)
	if projection.ProjectionID != expectedID {
		return fmt.Errorf("projection_id mismatch: expected %s, got %s", expectedID, projection.ProjectionID)
	}
	return nil
}

func validateNodeRef(reference nodeRef) error {
	if reference.Format != NodeRefFormat ||
		reference.AnchorKind != StaticAnchorKind ||
		reference.Canonicalization != NodeCanonicalizer ||
		reference.LiveFact ||
		reference.NetworkObserved ||
		reference.NodeDigest != TargetNodeDigest ||
		reference.NodeID != TargetNodeID ||
		reference.ResultAuthority != ResultAuthority ||
		reference.RewardBearing ||
		reference.TreeRawSHA256 != TreeRawDigest ||
		reference.TreeSchema != TreeSchema {
		return fmt.Errorf("node_ref must match the exact static, non-reward-bearing math-proofcraft target")
	}
	if err := validateDigest("node_ref.node_ref_id", reference.NodeRefID); err != nil {
		return err
	}
	body := struct {
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
		Format:           reference.Format,
		AnchorKind:       reference.AnchorKind,
		Canonicalization: reference.Canonicalization,
		LiveFact:         reference.LiveFact,
		NetworkObserved:  reference.NetworkObserved,
		NodeDigest:       reference.NodeDigest,
		NodeID:           reference.NodeID,
		ResultAuthority:  reference.ResultAuthority,
		RewardBearing:    reference.RewardBearing,
		TreeRawSHA256:    reference.TreeRawSHA256,
		TreeSchema:       reference.TreeSchema,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode node_ref body: %w", err)
	}
	canonical, err := canonicalRaw(raw)
	if err != nil {
		return err
	}
	expectedID := domainID(NodeRefFormat, canonical)
	if reference.NodeRefID != expectedID {
		return fmt.Errorf("node_ref_id mismatch: expected %s, got %s", expectedID, reference.NodeRefID)
	}
	return nil
}

func validateSortedUniqueDigests(path string, values []string, allowEmpty bool) error {
	if !allowEmpty && len(values) == 0 {
		return fmt.Errorf("%s must not be empty", path)
	}
	for index, value := range values {
		if err := validateDigest(fmt.Sprintf("%s[%d]", path, index), value); err != nil {
			return err
		}
		if index != 0 && values[index-1] >= value {
			return fmt.Errorf("%s must be sorted and unique", path)
		}
	}
	return nil
}

func validateDigest(path, value string) error {
	if !digestPattern.MatchString(value) {
		return fmt.Errorf("%s must be sha256:<64 lowercase hex>", path)
	}
	return nil
}

func sortedCopy(values []string) []string {
	copyOfValues := append([]string(nil), values...)
	sort.Strings(copyOfValues)
	return copyOfValues
}
