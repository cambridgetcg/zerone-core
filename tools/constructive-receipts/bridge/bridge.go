package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	poca "github.com/zerone-chain/zerone/tools/poca-shadow/evaluate"
)

var requiredPoCAStandards = []poca.StandardRef{
	{
		URI:     "https://github.com/in-toto/attestation/blob/v1.2.0/spec/v1/statement.md",
		Version: "1.2.0",
		Status:  "STABLE",
		Target:  "Statement layer",
	},
	{
		URI:     "https://github.com/sigstore/protobuf-specs/blob/3001afe9102b15b04ca1b91efccd613976bdf514/protos/sigstore_bundle.proto",
		Version: "0.3",
		Status:  "STABLE",
		Target:  "Bundle v0.3 signature, identity, trust-root, time, and inclusion policy",
	},
	{
		URI:     "https://slsa.dev/spec/v1.2/",
		Version: "1.2",
		Status:  "APPROVED",
		Target:  "Build L2",
	},
}

// Evaluate re-evaluates the PoCA profile and evidence locally and projects a
// deterministic constructive receipt. It never accepts a caller-built PoCA
// certificate.
func Evaluate(
	request Request,
	treeBytes []byte,
	profile poca.Profile,
	evidence poca.EvidenceBundle,
) (Receipt, error) {
	if err := validateRequest(request); err != nil {
		return Receipt{}, fmt.Errorf("request: %w", err)
	}
	tree, err := parseTree(treeBytes)
	if err != nil {
		return Receipt{}, err
	}
	if err := matchTreePins(request.Target, tree); err != nil {
		return Receipt{}, err
	}

	certificate, err := poca.Evaluate(profile, evidence)
	if err != nil {
		return Receipt{}, fmt.Errorf("re-evaluate PoCA: %w", err)
	}
	if certificate.Assurance != AssuranceUnverified {
		return Receipt{}, fmt.Errorf("PoCA assurance must be %q", AssuranceUnverified)
	}
	if certificate.Reward.EconomicEffect != EffectNone || certificate.Reward.AmountUzrn != "0" {
		return Receipt{}, fmt.Errorf("PoCA certificate violates zero-economics invariant")
	}
	if err := matchPoCAPins(request.PoCA, certificate); err != nil {
		return Receipt{}, err
	}

	consumptionKey := deriveDigest(
		"zerone.constructive-source-consumption/v0",
		request.Source.SourceSystem,
		request.Source.RecordID,
		request.Source.Revision,
	)
	receiptID := deriveDigest(
		"zerone.constructive-receipt/v0",
		BridgeVersion,
		tree.DocumentDigest,
		tree.PolicyDigest,
		tree.NodeDigest,
		TargetNodeID,
		certificate.Profile.Digest,
		certificate.EvidenceBundleDigest,
		certificate.Subject.Digest,
		consumptionKey,
	)

	refusals := candidateRefusals(profile, certificate)
	status := StatusCandidate
	if len(refusals) != 0 {
		status = StatusRefused
	}

	receipt := Receipt{
		Schema:        ReceiptSchema,
		BridgeVersion: BridgeVersion,
		Assurance:     AssuranceUnverified,
		Status:        status,
		ReceiptID:     receiptID,
		Source: SourceBinding{
			SourceSystem:     request.Source.SourceSystem,
			RecordID:         request.Source.RecordID,
			Revision:         request.Source.Revision,
			ConsumptionKey:   consumptionKey,
			ConsumptionState: ConsumptionUnrecorded,
			ReplayProtection: ReplayProtectionNone,
		},
		Tree: TreeBinding{
			EvidenceNamespace:          TreeEvidenceNamespace,
			Schema:                     TreeSchema,
			PolicyVersion:              TreePolicyVersion,
			PolicyDigest:               tree.PolicyDigest,
			DocumentDigest:             tree.DocumentDigest,
			NodeID:                     TargetNodeID,
			NodeDigest:                 tree.NodeDigest,
			RequiredAttainmentEvidence: TargetNodeEvidence,
			GrantedAttainmentEvidence:  EffectNone,
		},
		PoCA: PoCABinding{
			EvidenceNamespace:    PoCAEvidenceNamespace,
			ProfileID:            certificate.Profile.ProfileID,
			ProfileVersion:       certificate.Profile.ProfileVersion,
			ProfileStatus:        profile.Status,
			ProfileDigest:        certificate.Profile.Digest,
			EvidenceBundleDigest: certificate.EvidenceBundleDigest,
			ClaimID:              certificate.ClaimID,
			SubjectDigest:        certificate.Subject.Digest,
			AttainedTier:         certificate.AttainedTier,
			CrownStatus:          certificate.CrownStatus,
			Assurance:            certificate.Assurance,
		},
		NamespaceRelation: NamespaceRelation,
		Qualification:     EffectNone,
		EconomicEffect:    EffectNone,
		AmountUzrn:        "0",
		Escrow:            zeroEscrow(),
		RefusalReasons:    refusals,
		Limitations: []string{
			"CANDIDATE is not tree attainment, qualification, certification, payment authorization, or entitlement",
			"PoCA tier labels and constructive-tree evidence levels are separate namespaces with no ordinal or semantic equivalence",
			"profile lifecycle, participant roles, control clusters, policy authority, and PoCA results remain declarations",
			"the bridge does not verify Sigstore signatures, identities, trust roots, transparency evidence, or external predicate semantics",
			"the source tuple is caller-declared and the deterministic consumption key is not a replay ledger or proof that a source event is unconsumed",
			"the output is offline, unsigned, unanchored, and retains UNVERIFIED_SHADOW_PROJECTION assurance",
		},
	}
	if err := validateOutput(receipt); err != nil {
		return Receipt{}, fmt.Errorf("internal receipt invariant: %w", err)
	}
	return receipt, nil
}

func validateOutput(receipt Receipt) error {
	if receipt.Schema != ReceiptSchema ||
		receipt.BridgeVersion != BridgeVersion ||
		receipt.Assurance != AssuranceUnverified {
		return fmt.Errorf("schema, bridge version, or assurance drift")
	}
	if receipt.Status != StatusCandidate && receipt.Status != StatusRefused {
		return fmt.Errorf("unsupported status %q", receipt.Status)
	}
	if receipt.Status == StatusCandidate && len(receipt.RefusalReasons) != 0 {
		return fmt.Errorf("candidate contains refusal reasons")
	}
	if receipt.Status == StatusRefused && len(receipt.RefusalReasons) == 0 {
		return fmt.Errorf("refusal has no refusal reason")
	}
	if receipt.NamespaceRelation != NamespaceRelation ||
		receipt.Tree.EvidenceNamespace != TreeEvidenceNamespace ||
		receipt.PoCA.EvidenceNamespace != PoCAEvidenceNamespace {
		return fmt.Errorf("evidence namespace drift")
	}
	if receipt.Qualification != EffectNone ||
		receipt.EconomicEffect != EffectNone ||
		receipt.AmountUzrn != "0" ||
		receipt.Tree.GrantedAttainmentEvidence != EffectNone {
		return fmt.Errorf("zero-value boundary crossed")
	}
	if receipt.Source.ConsumptionState != ConsumptionUnrecorded ||
		receipt.Source.ReplayProtection != ReplayProtectionNone {
		return fmt.Errorf("offline receipt claims replay-ledger state")
	}
	if receipt.Escrow != zeroEscrow() {
		return fmt.Errorf("escrow differs from exact all-zero compartments")
	}
	return nil
}

func matchTreePins(request TargetRequest, tree parsedTree) error {
	for _, comparison := range []struct {
		path     string
		expected string
		actual   string
	}{
		{"tree schema", request.TreeSchema, TreeSchema},
		{"tree policy version", request.TreePolicyVersion, TreePolicyVersion},
		{"tree policy digest", request.TreePolicyDigest, tree.PolicyDigest},
		{"tree document digest", request.TreeDocumentDigest, tree.DocumentDigest},
		{"tree node id", request.NodeID, TargetNodeID},
		{"tree node digest", request.NodeDigest, tree.NodeDigest},
	} {
		if comparison.expected != comparison.actual {
			return fmt.Errorf(
				"%s pin mismatch: expected %s, got %s",
				comparison.path,
				comparison.expected,
				comparison.actual,
			)
		}
	}
	return nil
}

func matchPoCAPins(request PoCARequest, certificate poca.Certificate) error {
	for _, comparison := range []struct {
		path     string
		expected string
		actual   string
	}{
		{"PoCA profile id", request.ProfileID, certificate.Profile.ProfileID},
		{"PoCA profile version", request.ProfileVersion, certificate.Profile.ProfileVersion},
		{"PoCA profile digest", request.ProfileDigest, certificate.Profile.Digest},
		{"PoCA evidence bundle digest", request.EvidenceBundleDigest, certificate.EvidenceBundleDigest},
		{"PoCA subject digest", request.SubjectDigest, certificate.Subject.Digest},
	} {
		if comparison.expected != comparison.actual {
			return fmt.Errorf(
				"%s pin mismatch: expected %s, got %s",
				comparison.path,
				comparison.expected,
				comparison.actual,
			)
		}
	}
	return nil
}

func candidateRefusals(profile poca.Profile, certificate poca.Certificate) []RefusalReason {
	refusals := []RefusalReason{}
	if profile.Status != "DECLARED_RATIFIED" {
		refusals = append(refusals, RefusalReason{
			Code:   "POCA_PROFILE_NOT_DECLARED_RATIFIED",
			Detail: fmt.Sprintf("profile status is %s", profile.Status),
		})
	}
	if len(certificate.UnresolvedChallengeDigests) != 0 {
		refusals = append(refusals, RefusalReason{
			Code:   "POCA_UNRESOLVED_CHALLENGE",
			Detail: fmt.Sprintf("%d unresolved PoCA challenge(s)", len(certificate.UnresolvedChallengeDigests)),
		})
	}
	if missing := missingStandards(profile.Standards, requiredPoCAStandards); len(missing) != 0 {
		refusals = append(refusals, RefusalReason{
			Code:   "POCA_TARGET_STANDARD_BINDING_INCOMPLETE",
			Detail: "missing exact PoCA standard binding(s): " + strings.Join(missing, ", "),
		})
	}
	for _, required := range []struct {
		id   string
		code string
	}{
		{"standard.slsa-build-l2", "POCA_SLSA_NODE_NOT_DECLARED_PASS"},
		{"capability.independent-rebuild", "POCA_INDEPENDENT_REBUILD_NOT_DECLARED_PASS"},
	} {
		status := "MISSING"
		for _, result := range certificate.NodeResults {
			if result.NodeID == required.id {
				status = result.Status
				break
			}
		}
		if status != "DECLARED_PASS" {
			refusals = append(refusals, RefusalReason{
				Code:   required.code,
				Detail: fmt.Sprintf("PoCA node %s is %s", required.id, status),
			})
		}
	}
	sort.Slice(refusals, func(i, j int) bool {
		if refusals[i].Code != refusals[j].Code {
			return refusals[i].Code < refusals[j].Code
		}
		return refusals[i].Detail < refusals[j].Detail
	})
	return refusals
}

func missingStandards(actual, required []poca.StandardRef) []string {
	keys := make(map[string]struct{}, len(actual))
	for _, standard := range actual {
		keys[standardKey(standard)] = struct{}{}
	}
	var missing []string
	for _, standard := range required {
		if _, exists := keys[standardKey(standard)]; !exists {
			missing = append(missing, standard.URI+" @ "+standard.Version+" / "+standard.Target)
		}
	}
	sort.Strings(missing)
	return missing
}

func standardKey(standard poca.StandardRef) string {
	return strings.Join(
		[]string{standard.URI, standard.Version, standard.Status, standard.Target},
		"\x00",
	)
}

func deriveDigest(domain string, values ...string) string {
	hash := sha256.New()
	hash.Write([]byte(domain))
	for _, value := range values {
		hash.Write([]byte{0})
		hash.Write([]byte(value))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
