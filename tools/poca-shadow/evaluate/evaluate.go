package evaluate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	statusDeclaredPass = "DECLARED_PASS"
	statusBlocked      = "BLOCKED"
	statusFailed       = "FAILED"
)

// Evaluate projects a validated profile and normalized evidence bundle into a
// deterministic shadow certificate. It repeats all validation so callers that
// construct Go values directly receive the same fail-closed behavior as CLI
// users.
func Evaluate(profile Profile, bundle EvidenceBundle) (Certificate, error) {
	if err := validateProfile(profile); err != nil {
		return Certificate{}, fmt.Errorf("profile: %w", err)
	}
	if err := validateEvidenceBundle(bundle); err != nil {
		return Certificate{}, fmt.Errorf("evidence bundle: %w", err)
	}
	if bundle.ProfileID != profile.ProfileID || bundle.ProfileVersion != profile.ProfileVersion {
		return Certificate{}, fmt.Errorf(
			"evidence profile %s@%s does not match %s@%s",
			bundle.ProfileID,
			bundle.ProfileVersion,
			profile.ProfileID,
			profile.ProfileVersion,
		)
	}

	profileDigest, err := digestCanonicalProfile(profile)
	if err != nil {
		return Certificate{}, err
	}
	bundleDigest, err := digestCanonicalBundle(bundle)
	if err != nil {
		return Certificate{}, err
	}

	requirements := make(map[string]Requirement, len(profile.Requirements))
	for _, requirement := range profile.Requirements {
		requirements[requirement.ID] = requirement
	}
	participants := make(map[string]Participant, len(bundle.Participants))
	var claimant Participant
	for _, participant := range bundle.Participants {
		participants[participant.ID] = participant
		if participant.Role == "CLAIMANT" {
			claimant = participant
		}
	}

	receiptsByRequirement := make(map[string][]Receipt, len(requirements))
	for _, receipt := range bundle.Evidence {
		requirement, ok := requirements[receipt.RequirementID]
		if !ok {
			return Certificate{}, fmt.Errorf("evidence %q references unknown requirement %q", receipt.ID, receipt.RequirementID)
		}
		if receipt.VerificationRule != requirement.VerificationRule {
			return Certificate{}, fmt.Errorf(
				"evidence %q verification_rule %q does not match requirement %q",
				receipt.ID,
				receipt.VerificationRule,
				requirement.VerificationRule,
			)
		}
		if receipt.PolicyDigest != requirement.PolicyDigest {
			return Certificate{}, fmt.Errorf(
				"evidence %q policy_digest %q does not match requirement %q",
				receipt.ID,
				receipt.PolicyDigest,
				requirement.PolicyDigest,
			)
		}
		if receipt.PredicateType != requirement.PredicateType {
			return Certificate{}, fmt.Errorf(
				"evidence %q predicate_type %q does not match requirement %q",
				receipt.ID,
				receipt.PredicateType,
				requirement.PredicateType,
			)
		}
		observer := participants[receipt.ObserverParticipantID]
		if observer.Role != requirement.ObserverRole {
			return Certificate{}, fmt.Errorf(
				"evidence %q observer role %q does not match requirement role %q",
				receipt.ID,
				observer.Role,
				requirement.ObserverRole,
			)
		}
		receiptsByRequirement[receipt.RequirementID] = append(receiptsByRequirement[receipt.RequirementID], receipt)
	}

	requirementResults := make(map[string]RequirementResult, len(requirements))
	var notices []string
	for _, requirementID := range sortedMapKeys(requirements) {
		result, resultNotices := evaluateRequirement(
			requirements[requirementID],
			receiptsByRequirement[requirementID],
			participants,
			claimant,
		)
		requirementResults[requirementID] = result
		notices = append(notices, resultNotices...)
	}

	nodes := make(map[string]Node, len(profile.Nodes))
	for _, node := range profile.Nodes {
		nodes[node.ID] = node
	}
	nodeResults := make(map[string]NodeResult, len(nodes))
	var evaluateNode func(string) NodeResult
	evaluateNode = func(nodeID string) NodeResult {
		if existing, ok := nodeResults[nodeID]; ok {
			return existing
		}
		node := nodes[nodeID]
		result := NodeResult{
			NodeID:        node.ID,
			Stage:         node.Stage,
			Tier:          node.Tier,
			Status:        statusDeclaredPass,
			Prerequisites: sortedStrings(node.Prerequisites),
		}

		for _, prerequisiteID := range result.Prerequisites {
			prerequisite := evaluateNode(prerequisiteID)
			if prerequisite.Status != statusDeclaredPass {
				result.Status = statusBlocked
				result.RefusalCodes = appendUnique(result.RefusalCodes, "PREREQUISITE_NOT_MET")
				result.RefusalDetails = appendUnique(
					result.RefusalDetails,
					fmt.Sprintf("prerequisite %s is %s", prerequisiteID, prerequisite.Status),
				)
			}
		}

		for _, requirementID := range sortedStrings(node.RequirementIDs) {
			requirement := requirementResults[requirementID]
			result.RequirementResults = append(result.RequirementResults, requirement)
			result.AcceptedEvidenceIDs = append(result.AcceptedEvidenceIDs, requirement.AcceptedEvidenceIDs...)
			result.ControlClusterClaims = append(result.ControlClusterClaims, requirement.ControlClusterClaims...)
			switch requirement.Status {
			case statusFailed:
				result.Status = statusFailed
				result.RefusalCodes = append(result.RefusalCodes, requirement.RefusalCodes...)
				result.RefusalDetails = append(result.RefusalDetails, requirement.RefusalDetails...)
			case statusBlocked:
				if result.Status != statusFailed {
					result.Status = statusBlocked
				}
				result.RefusalCodes = append(result.RefusalCodes, requirement.RefusalCodes...)
				result.RefusalDetails = append(result.RefusalDetails, requirement.RefusalDetails...)
			}
		}

		if node.ID == profile.CrownNodeID {
			if profile.Status == "DRAFT" {
				if result.Status != statusFailed {
					result.Status = statusBlocked
				}
				result.RefusalCodes = appendUnique(result.RefusalCodes, "PROFILE_NOT_RATIFIED")
				result.RefusalDetails = appendUnique(result.RefusalDetails, "profile status is DRAFT")
			}
			if profile.Status == "SUPERSEDED" {
				if result.Status != statusFailed {
					result.Status = statusBlocked
				}
				result.RefusalCodes = appendUnique(result.RefusalCodes, "PROFILE_SUPERSEDED")
				result.RefusalDetails = appendUnique(result.RefusalDetails, "profile status is SUPERSEDED")
			}
			if profile.ChallengePolicy.UnresolvedChallengeBlocksCrown &&
				len(bundle.UnresolvedChallengeDigests) != 0 {
				if result.Status != statusFailed {
					result.Status = statusBlocked
				}
				result.RefusalCodes = appendUnique(result.RefusalCodes, "CHALLENGE_OPEN")
				result.RefusalDetails = appendUnique(
					result.RefusalDetails,
					fmt.Sprintf("%d unresolved challenge(s) block the crown", len(bundle.UnresolvedChallengeDigests)),
				)
			}
		}

		result.AcceptedEvidenceIDs = sortedUnique(result.AcceptedEvidenceIDs)
		result.ControlClusterClaims = sortedUnique(result.ControlClusterClaims)
		result.RefusalCodes = sortedUnique(result.RefusalCodes)
		result.RefusalDetails = sortedUnique(result.RefusalDetails)
		if result.RequirementResults == nil {
			result.RequirementResults = []RequirementResult{}
		}
		nodeResults[nodeID] = result
		return result
	}

	for _, nodeID := range sortedMapKeys(nodes) {
		evaluateNode(nodeID)
	}
	orderedNodeResults := make([]NodeResult, 0, len(nodeResults))
	attainedTier := NoAttainedTier
	for _, nodeID := range sortedMapKeys(nodeResults) {
		result := nodeResults[nodeID]
		orderedNodeResults = append(orderedNodeResults, result)
		if result.Status == statusDeclaredPass && tierRank[result.Tier] > tierRank[attainedTier] {
			attainedTier = result.Tier
		}
	}

	if profile.Status == "DRAFT" {
		notices = append(notices, "profile status is DRAFT")
	}

	return Certificate{
		Schema:           CertificateSchema,
		EvaluatorVersion: EvaluatorVersion,
		Assurance:        CertificateAssurance,
		ClaimID: deriveClaimID(
			profileDigest,
			bundle.Subject.Digest,
			bundle.BaselineDigest,
			bundle.LineageDigest,
		),
		Profile: ProfileBinding{
			ProfileID:      profile.ProfileID,
			ProfileVersion: profile.ProfileVersion,
			Digest:         profileDigest,
		},
		EvidenceBundleDigest:       bundleDigest,
		Subject:                    bundle.Subject,
		NodeResults:                orderedNodeResults,
		AttainedTier:               attainedTier,
		CrownStatus:                nodeResults[profile.CrownNodeID].Status,
		UnresolvedChallengeDigests: sortedStrings(bundle.UnresolvedChallengeDigests),
		Reward: RewardRefusal{
			EconomicEffect: EconomicEffectNone,
			AmountUzrn:     "0",
			Reason:         "shadow-profile-v0",
		},
		Notices: sortedUnique(notices),
		Limitations: []string{
			"control_cluster_claim values are declarations, not proof of economic independence",
			"the evaluator does not verify signatures, external predicate semantics, source availability, or current deployment",
			"policy digests prove identity of declared policy bytes, not adequacy or correct execution of that policy",
			"profile lifecycle status, participant roles, identities, and policy authority are declarations",
			"this certificate is not consensus state, certification, qualification, payment authorization, or a reward claim",
		},
	}, nil
}

func evaluateRequirement(
	requirement Requirement,
	receipts []Receipt,
	participants map[string]Participant,
	claimant Participant,
) (RequirementResult, []string) {
	result := RequirementResult{
		RequirementID: requirement.ID,
		Status:        statusBlocked,
	}
	sortedReceipts := append([]Receipt(nil), receipts...)
	sort.Slice(sortedReceipts, func(i, j int) bool {
		return sortedReceipts[i].ID < sortedReceipts[j].ID
	})

	groups := make(map[string][]Receipt)
	var notices []string
	var failedIDs []string
	for _, receipt := range sortedReceipts {
		deduplicationKey := receipt.StatementDigest + "\x00" + receipt.VerificationReceiptDigest
		if receipt.Result == "FAIL" {
			failedIDs = append(failedIDs, receipt.ID)
		}
		groups[deduplicationKey] = append(groups[deduplicationKey], receipt)
	}

	for _, key := range sortedMapKeys(groups) {
		group := groups[key]
		hasFailure := false
		var eligible []Receipt
		for _, receipt := range group {
			if receipt.Result == "FAIL" {
				hasFailure = true
				continue
			}
			observer := participants[receipt.ObserverParticipantID]
			if requirement.RequireObserverIndependentFromClaimant &&
				observer.ControlClusterClaim == claimant.ControlClusterClaim {
				result.RefusalDetails = appendUnique(
					result.RefusalDetails,
					fmt.Sprintf("evidence %s observer shares claimant control cluster %s", receipt.ID, claimant.ControlClusterClaim),
				)
				continue
			}
			eligible = append(eligible, receipt)
		}
		if hasFailure || len(eligible) == 0 {
			continue
		}
		sort.Slice(eligible, func(i, j int) bool {
			leftCluster := participants[eligible[i].ObserverParticipantID].ControlClusterClaim
			rightCluster := participants[eligible[j].ObserverParticipantID].ControlClusterClaim
			if leftCluster != rightCluster {
				return leftCluster < rightCluster
			}
			return eligible[i].ID < eligible[j].ID
		})
		accepted := eligible[0]
		result.AcceptedEvidenceIDs = append(result.AcceptedEvidenceIDs, accepted.ID)
		result.ControlClusterClaims = append(
			result.ControlClusterClaims,
			participants[accepted.ObserverParticipantID].ControlClusterClaim,
		)
		for _, receipt := range group {
			if receipt.ID != accepted.ID {
				notices = append(
					notices,
					fmt.Sprintf("requirement %s collapsed duplicate evidence %s", requirement.ID, receipt.ID),
				)
			}
		}
	}

	result.AcceptedEvidenceIDs = sortedUnique(result.AcceptedEvidenceIDs)
	result.ControlClusterClaims = sortedUnique(result.ControlClusterClaims)
	if len(failedIDs) != 0 {
		result.Status = statusFailed
		if requirement.HardGuardrail {
			result.RefusalCodes = append(result.RefusalCodes, "HARD_GUARDRAIL_FAILED")
		} else {
			result.RefusalCodes = append(result.RefusalCodes, "EVIDENCE_FAILED")
		}
		result.RefusalDetails = append(
			result.RefusalDetails,
			fmt.Sprintf("failing evidence: %s", summarizeIDs(failedIDs, 3)),
		)
	} else {
		if len(result.AcceptedEvidenceIDs) < int(requirement.MinCount) {
			result.RefusalCodes = append(result.RefusalCodes, "EVIDENCE_MISSING")
			result.RefusalDetails = append(
				result.RefusalDetails,
				fmt.Sprintf(
					"requirement needs %d distinct passing receipt(s), found %d",
					requirement.MinCount,
					len(result.AcceptedEvidenceIDs),
				),
			)
		}
		if len(result.ControlClusterClaims) < int(requirement.MinIndependentControlClusters) {
			result.RefusalCodes = append(result.RefusalCodes, "INSUFFICIENT_INDEPENDENCE")
			result.RefusalDetails = append(
				result.RefusalDetails,
				fmt.Sprintf(
					"requirement needs %d declared control cluster(s), found %d",
					requirement.MinIndependentControlClusters,
					len(result.ControlClusterClaims),
				),
			)
		}
		if len(result.RefusalCodes) == 0 {
			result.Status = statusDeclaredPass
		}
	}

	result.RefusalCodes = sortedUnique(result.RefusalCodes)
	result.RefusalDetails = sortedUnique(result.RefusalDetails)
	return result, notices
}

func digestCanonicalProfile(profile Profile) (string, error) {
	normalized := profile
	normalized.Standards = append([]StandardRef{}, profile.Standards...)
	sort.Slice(normalized.Standards, func(i, j int) bool {
		left := normalized.Standards[i]
		right := normalized.Standards[j]
		return strings.Join([]string{left.URI, left.Version, left.Target}, "\x00") <
			strings.Join([]string{right.URI, right.Version, right.Target}, "\x00")
	})
	normalized.Requirements = append([]Requirement{}, profile.Requirements...)
	sort.Slice(normalized.Requirements, func(i, j int) bool {
		return normalized.Requirements[i].ID < normalized.Requirements[j].ID
	})
	normalized.Nodes = append([]Node{}, profile.Nodes...)
	for i := range normalized.Nodes {
		normalized.Nodes[i].Prerequisites = sortedStrings(normalized.Nodes[i].Prerequisites)
		normalized.Nodes[i].RequirementIDs = sortedStrings(normalized.Nodes[i].RequirementIDs)
	}
	sort.Slice(normalized.Nodes, func(i, j int) bool {
		return normalized.Nodes[i].ID < normalized.Nodes[j].ID
	})
	return digestJSON(normalized)
}

func digestCanonicalBundle(bundle EvidenceBundle) (string, error) {
	normalized := bundle
	normalized.Participants = append([]Participant{}, bundle.Participants...)
	sort.Slice(normalized.Participants, func(i, j int) bool {
		return normalized.Participants[i].ID < normalized.Participants[j].ID
	})
	normalized.Evidence = append([]Receipt{}, bundle.Evidence...)
	sort.Slice(normalized.Evidence, func(i, j int) bool {
		return normalized.Evidence[i].ID < normalized.Evidence[j].ID
	})
	normalized.UnresolvedChallengeDigests = sortedStrings(bundle.UnresolvedChallengeDigests)
	return digestJSON(normalized)
}

func digestJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode canonical JSON: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func deriveClaimID(profileDigest, subjectDigest, baselineDigest, lineageDigest string) string {
	hash := sha256.New()
	hash.Write([]byte("zerone.breakthrough-claim/v0"))
	hash.Write([]byte{0})
	hash.Write([]byte(profileDigest))
	hash.Write([]byte{0})
	hash.Write([]byte(subjectDigest))
	hash.Write([]byte{0})
	hash.Write([]byte(baselineDigest))
	hash.Write([]byte{0})
	hash.Write([]byte(lineageDigest))
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func summarizeIDs(values []string, limit int) string {
	values = sortedUnique(values)
	if len(values) <= limit {
		return strings.Join(values, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(values[:limit], ", "), len(values)-limit)
}

// EvaluateInToto evaluates the inputs and projects the resulting certificate
// into an unsigned in-toto Statement v1 suitable for a separately pinned
// DSSE/Sigstore signing workflow. It does not accept caller-constructed
// certificates.
func EvaluateInToto(profile Profile, bundle EvidenceBundle) (InTotoStatement, error) {
	certificate, err := Evaluate(profile, bundle)
	if err != nil {
		return InTotoStatement{}, err
	}
	return wrapInToto(certificate)
}

func wrapInToto(certificate Certificate) (InTotoStatement, error) {
	if certificate.Schema != CertificateSchema ||
		certificate.EvaluatorVersion != EvaluatorVersion ||
		certificate.Assurance != CertificateAssurance {
		return InTotoStatement{}, fmt.Errorf("certificate is not a PoCA v0 shadow certificate")
	}
	if certificate.Reward.EconomicEffect != EconomicEffectNone ||
		certificate.Reward.AmountUzrn != "0" ||
		certificate.Reward.Reason != "shadow-profile-v0" {
		return InTotoStatement{}, fmt.Errorf("certificate violates the PoCA v0 zero-reward invariant")
	}
	for path, digest := range map[string]string{
		"certificate.claim_id":               certificate.ClaimID,
		"certificate.profile.digest":         certificate.Profile.Digest,
		"certificate.evidence_bundle_digest": certificate.EvidenceBundleDigest,
	} {
		if err := validateDigest(path, digest); err != nil {
			return InTotoStatement{}, err
		}
	}
	if err := validateDigest("certificate.subject.digest", certificate.Subject.Digest); err != nil {
		return InTotoStatement{}, err
	}
	return InTotoStatement{
		Type: InTotoStatementV1,
		Subject: []InTotoSubject{{
			Name: certificate.Subject.Name,
			Digest: map[string]string{
				"sha256": strings.TrimPrefix(certificate.Subject.Digest, "sha256:"),
			},
		}},
		PredicateType: PredicateTypeV0,
		Predicate:     certificate,
	}, nil
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := sortedStrings(values)
	write := 1
	for read := 1; read < len(result); read++ {
		if result[read] == result[write-1] {
			continue
		}
		result[write] = result[read]
		write++
	}
	return result[:write]
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
