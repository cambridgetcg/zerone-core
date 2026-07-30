package evaluate

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	maxStandards    = 16
	maxRequirements = 64
	maxNodes        = 64
	maxParticipants = 128
	maxEvidence     = 256
	maxChallenges   = 64
)

var boundedID = regexp.MustCompile(`^[a-z0-9][a-z0-9._:@/-]{0,127}$`)
var mediaType = regexp.MustCompile(`^[A-Za-z0-9!#$&^_.+-]+/[A-Za-z0-9!#$&^_.+-]+$`)

var allowedProfileStatuses = stringSet("DRAFT", "DECLARED_RATIFIED", "SUPERSEDED")
var allowedStandardStatuses = stringSet("DRAFT", "STABLE", "APPROVED", "SUPERSEDED")
var allowedStages = stringSet("GROUND", "CAPABILITY", "INDUSTRIAL", "RECURSIVE", "CROWN")
var allowedTiers = stringSet(
	"E0_ASSERTED",
	"E1_REPRODUCED",
	"E2_CONFORMANT",
	"E3_CAUSAL",
	"E4_TRANSFERRED",
	"E5_REUSED",
)
var allowedRoles = stringSet(
	"CLAIMANT",
	"PRODUCER",
	"SPONSOR",
	"VERIFIER",
	"ADOPTER",
	"OPERATOR",
	"CHALLENGER",
)
var allowedKinds = stringSet(
	"PROBLEM_CONTRACT",
	"SIGNED_PROVENANCE",
	"CONFORMANCE_RESULT",
	"CONSUMER_VERIFICATION",
	"INDEPENDENT_REBUILD",
	"CAUSAL_EVALUATION",
	"INTEROPERABILITY_RESULT",
	"PRODUCTION_SLO",
	"ROLLBACK_DRILL",
	"SECURITY_ASSESSMENT",
	"INDEPENDENT_ADOPTION",
	"ZERONE_RECURSIVE_VALUE",
	"ZERONE_CHAIN_RECORD",
)
var allowedVerificationRules = stringSet(
	"FROZEN_BASELINE_AND_GUARDRAILS",
	"SIGSTORE_SIGNATURE_AND_SUBJECT",
	"SLSA_BUILD_V1_EXPECTATIONS",
	"ALL_TEST_CASES_PASS",
	"BYTE_IDENTICAL_SUBJECT",
	"CAUSAL_HOLDOUT_GAIN",
	"INTEROPERABILITY_SUITE_PASS",
	"SLO_AND_GUARDRAILS_PASS",
	"ROLLBACK_COMPLETED",
	"INDEPENDENT_ADOPTION_RECEIPT",
	"ZERONE_RECURSION_LINK",
	"CHAIN_RECORD_MATCHES",
)

var stageRank = map[string]int{
	"GROUND":     0,
	"CAPABILITY": 1,
	"INDUSTRIAL": 2,
	"RECURSIVE":  3,
	"CROWN":      4,
}

var tierRank = map[string]int{
	"NONE":           -1,
	"E0_ASSERTED":    0,
	"E1_REPRODUCED":  1,
	"E2_CONFORMANT":  2,
	"E3_CAUSAL":      3,
	"E4_TRANSFERRED": 4,
	"E5_REUSED":      5,
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func validateProfile(profile Profile) error {
	if profile.Schema != ProfileSchema {
		return fmt.Errorf("schema must be %q", ProfileSchema)
	}
	if err := validateID("profile_id", profile.ProfileID); err != nil {
		return err
	}
	if err := validateID("profile_version", profile.ProfileVersion); err != nil {
		return err
	}
	if err := validateBoundedText("title", profile.Title, 1, 256); err != nil {
		return err
	}
	if _, ok := allowedProfileStatuses[profile.Status]; !ok {
		return fmt.Errorf("status %q is not supported", profile.Status)
	}
	if profile.AssuranceMode != AssuranceModeShadowOnly {
		return fmt.Errorf("assurance_mode must be %q", AssuranceModeShadowOnly)
	}
	if profile.Economics.Mode != EconomicEffectNone || profile.Economics.AmountUzrn != "0" {
		return fmt.Errorf("v0 economics must be mode %q with amount_uzrn %q", EconomicEffectNone, "0")
	}

	if len(profile.Standards) == 0 || len(profile.Standards) > maxStandards {
		return fmt.Errorf("standards count must be between 1 and %d", maxStandards)
	}
	standardKeys := make(map[string]struct{}, len(profile.Standards))
	for i, standard := range profile.Standards {
		if err := validateHTTPS(fmt.Sprintf("standards[%d].uri", i), standard.URI); err != nil {
			return err
		}
		if err := validateBoundedText(fmt.Sprintf("standards[%d].version", i), standard.Version, 1, 64); err != nil {
			return err
		}
		if _, ok := allowedStandardStatuses[standard.Status]; !ok {
			return fmt.Errorf("standards[%d].status %q is not supported", i, standard.Status)
		}
		if err := validateBoundedText(fmt.Sprintf("standards[%d].target", i), standard.Target, 1, 256); err != nil {
			return err
		}
		key := strings.Join([]string{standard.URI, standard.Version, standard.Target}, "\x00")
		if _, exists := standardKeys[key]; exists {
			return fmt.Errorf("duplicate standard target at standards[%d]", i)
		}
		standardKeys[key] = struct{}{}
	}

	if len(profile.Requirements) == 0 || len(profile.Requirements) > maxRequirements {
		return fmt.Errorf("requirements count must be between 1 and %d", maxRequirements)
	}
	requirements := make(map[string]Requirement, len(profile.Requirements))
	for i, requirement := range profile.Requirements {
		if err := validateID(fmt.Sprintf("requirements[%d].id", i), requirement.ID); err != nil {
			return err
		}
		if _, exists := requirements[requirement.ID]; exists {
			return fmt.Errorf("duplicate requirement id %q", requirement.ID)
		}
		if _, ok := allowedKinds[requirement.Kind]; !ok {
			return fmt.Errorf("requirements[%d].kind %q is not supported", i, requirement.Kind)
		}
		if _, ok := allowedVerificationRules[requirement.VerificationRule]; !ok {
			return fmt.Errorf("requirements[%d].verification_rule %q is not supported", i, requirement.VerificationRule)
		}
		if err := validateDigest(fmt.Sprintf("requirements[%d].policy_digest", i), requirement.PolicyDigest); err != nil {
			return err
		}
		if requirement.PredicateType != "" {
			if err := validateHTTPS(fmt.Sprintf("requirements[%d].predicate_type", i), requirement.PredicateType); err != nil {
				return err
			}
		}
		if _, ok := allowedRoles[requirement.ObserverRole]; !ok || requirement.ObserverRole == "CLAIMANT" {
			return fmt.Errorf("requirements[%d].observer_role %q is not an external evidence role", i, requirement.ObserverRole)
		}
		if requirement.MinCount == 0 || requirement.MinCount > 32 {
			return fmt.Errorf("requirements[%d].min_count must be between 1 and 32", i)
		}
		if requirement.MinIndependentControlClusters == 0 ||
			requirement.MinIndependentControlClusters > requirement.MinCount {
			return fmt.Errorf("requirements[%d].min_independent_control_clusters must be between 1 and min_count", i)
		}
		requirements[requirement.ID] = requirement
	}

	if len(profile.Nodes) == 0 || len(profile.Nodes) > maxNodes {
		return fmt.Errorf("nodes count must be between 1 and %d", maxNodes)
	}
	nodes := make(map[string]Node, len(profile.Nodes))
	crownCount := 0
	for i, node := range profile.Nodes {
		if err := validateID(fmt.Sprintf("nodes[%d].id", i), node.ID); err != nil {
			return err
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node id %q", node.ID)
		}
		if _, ok := allowedStages[node.Stage]; !ok {
			return fmt.Errorf("nodes[%d].stage %q is not supported", i, node.Stage)
		}
		if _, ok := allowedTiers[node.Tier]; !ok {
			return fmt.Errorf("nodes[%d].tier %q is not supported", i, node.Tier)
		}
		if err := validateBoundedText(fmt.Sprintf("nodes[%d].title", i), node.Title, 1, 256); err != nil {
			return err
		}
		if len(node.Prerequisites) > 16 || len(node.RequirementIDs) > 16 {
			return fmt.Errorf("nodes[%d] exceeds 16 prerequisites or requirements", i)
		}
		if err := validateUniqueIDs(fmt.Sprintf("nodes[%d].prerequisites", i), node.Prerequisites); err != nil {
			return err
		}
		if err := validateUniqueIDs(fmt.Sprintf("nodes[%d].requirement_ids", i), node.RequirementIDs); err != nil {
			return err
		}
		if node.Stage != "CROWN" && len(node.RequirementIDs) == 0 {
			return fmt.Errorf("non-crown node %q must name at least one requirement", node.ID)
		}
		if node.Stage == "CROWN" {
			crownCount++
		}
		nodes[node.ID] = node
	}
	if err := validateID("crown_node_id", profile.CrownNodeID); err != nil {
		return err
	}
	crown, exists := nodes[profile.CrownNodeID]
	if !exists {
		return fmt.Errorf("crown_node_id %q does not exist", profile.CrownNodeID)
	}
	if crown.Stage != "CROWN" || crownCount != 1 {
		return fmt.Errorf("profile must contain exactly one CROWN node and crown_node_id must reference it")
	}

	for _, node := range nodes {
		for _, prerequisiteID := range node.Prerequisites {
			prerequisite, ok := nodes[prerequisiteID]
			if !ok {
				return fmt.Errorf("node %q references unknown prerequisite %q", node.ID, prerequisiteID)
			}
			if prerequisiteID == node.ID {
				return fmt.Errorf("node %q cannot depend on itself", node.ID)
			}
			if stageRank[prerequisite.Stage] > stageRank[node.Stage] {
				return fmt.Errorf("node %q depends on later-stage node %q", node.ID, prerequisiteID)
			}
			if tierRank[prerequisite.Tier] > tierRank[node.Tier] {
				return fmt.Errorf("node %q depends on higher-tier node %q", node.ID, prerequisiteID)
			}
		}
		for _, requirementID := range node.RequirementIDs {
			if _, ok := requirements[requirementID]; !ok {
				return fmt.Errorf("node %q references unknown requirement %q", node.ID, requirementID)
			}
		}
	}
	referencedRequirements := make(map[string]struct{}, len(requirements))
	for _, node := range nodes {
		for _, requirementID := range node.RequirementIDs {
			referencedRequirements[requirementID] = struct{}{}
		}
	}
	for _, requirementID := range sortedMapKeys(requirements) {
		if _, ok := referencedRequirements[requirementID]; !ok {
			return fmt.Errorf("requirement %q is not referenced by any node", requirementID)
		}
	}
	if err := validateAcyclic(nodes); err != nil {
		return err
	}
	if err := validateCrownAncestry(profile.CrownNodeID, nodes); err != nil {
		return err
	}
	return nil
}

func validateEvidenceBundle(bundle EvidenceBundle) error {
	if bundle.Schema != EvidenceSchema {
		return fmt.Errorf("schema must be %q", EvidenceSchema)
	}
	if err := validateID("profile_id", bundle.ProfileID); err != nil {
		return err
	}
	if err := validateID("profile_version", bundle.ProfileVersion); err != nil {
		return err
	}
	if err := validateArtifact("subject", bundle.Subject); err != nil {
		return err
	}
	if err := validateDigest("baseline_digest", bundle.BaselineDigest); err != nil {
		return err
	}
	if err := validateDigest("lineage_digest", bundle.LineageDigest); err != nil {
		return err
	}

	if len(bundle.Participants) == 0 || len(bundle.Participants) > maxParticipants {
		return fmt.Errorf("participants count must be between 1 and %d", maxParticipants)
	}
	participants := make(map[string]Participant, len(bundle.Participants))
	identityClusters := make(map[string]string, len(bundle.Participants))
	claimants := 0
	for i, participant := range bundle.Participants {
		if err := validateID(fmt.Sprintf("participants[%d].id", i), participant.ID); err != nil {
			return err
		}
		if _, exists := participants[participant.ID]; exists {
			return fmt.Errorf("duplicate participant id %q", participant.ID)
		}
		if _, ok := allowedRoles[participant.Role]; !ok {
			return fmt.Errorf("participants[%d].role %q is not supported", i, participant.Role)
		}
		if participant.Role == "CLAIMANT" {
			claimants++
		}
		if err := validateBoundedText(fmt.Sprintf("participants[%d].identity", i), participant.Identity, 1, 256); err != nil {
			return err
		}
		if err := validateID(fmt.Sprintf("participants[%d].control_cluster_claim", i), participant.ControlClusterClaim); err != nil {
			return err
		}
		if cluster, exists := identityClusters[participant.Identity]; exists &&
			cluster != participant.ControlClusterClaim {
			return fmt.Errorf(
				"participant identity %q declares conflicting control clusters %q and %q",
				participant.Identity,
				cluster,
				participant.ControlClusterClaim,
			)
		}
		identityClusters[participant.Identity] = participant.ControlClusterClaim
		participants[participant.ID] = participant
	}
	if claimants != 1 {
		return fmt.Errorf("participants must contain exactly one CLAIMANT")
	}

	if len(bundle.Evidence) > maxEvidence {
		return fmt.Errorf("evidence count exceeds %d", maxEvidence)
	}
	evidenceIDs := make(map[string]struct{}, len(bundle.Evidence))
	for i, receipt := range bundle.Evidence {
		if err := validateID(fmt.Sprintf("evidence[%d].id", i), receipt.ID); err != nil {
			return err
		}
		if _, exists := evidenceIDs[receipt.ID]; exists {
			return fmt.Errorf("duplicate evidence id %q", receipt.ID)
		}
		evidenceIDs[receipt.ID] = struct{}{}
		if err := validateID(fmt.Sprintf("evidence[%d].requirement_id", i), receipt.RequirementID); err != nil {
			return err
		}
		if _, ok := participants[receipt.ProducerParticipantID]; !ok {
			return fmt.Errorf("evidence[%d] references unknown producer %q", i, receipt.ProducerParticipantID)
		}
		if _, ok := participants[receipt.ObserverParticipantID]; !ok {
			return fmt.Errorf("evidence[%d] references unknown observer %q", i, receipt.ObserverParticipantID)
		}
		if receipt.Result != "PASS" && receipt.Result != "FAIL" {
			return fmt.Errorf("evidence[%d].result must be PASS or FAIL", i)
		}
		if receipt.PredicateType != "" {
			if err := validateHTTPS(fmt.Sprintf("evidence[%d].predicate_type", i), receipt.PredicateType); err != nil {
				return err
			}
		}
		if _, ok := allowedVerificationRules[receipt.VerificationRule]; !ok {
			return fmt.Errorf("evidence[%d].verification_rule %q is not supported", i, receipt.VerificationRule)
		}
		for name, digest := range map[string]string{
			"subject_digest":              receipt.SubjectDigest,
			"policy_digest":               receipt.PolicyDigest,
			"environment_digest":          receipt.EnvironmentDigest,
			"statement_digest":            receipt.StatementDigest,
			"verification_receipt_digest": receipt.VerificationReceiptDigest,
		} {
			if err := validateDigest(fmt.Sprintf("evidence[%d].%s", i, name), digest); err != nil {
				return err
			}
		}
		if receipt.SubjectDigest != bundle.Subject.Digest {
			return fmt.Errorf("evidence[%d].subject_digest does not match bundle subject", i)
		}
		if receipt.SourceURI != "" {
			if err := validateHTTPS(fmt.Sprintf("evidence[%d].source_uri", i), receipt.SourceURI); err != nil {
				return err
			}
		}
	}

	if len(bundle.UnresolvedChallengeDigests) > maxChallenges {
		return fmt.Errorf("unresolved_challenge_digests count exceeds %d", maxChallenges)
	}
	seenChallenges := make(map[string]struct{}, len(bundle.UnresolvedChallengeDigests))
	for i, digest := range bundle.UnresolvedChallengeDigests {
		if err := validateDigest(fmt.Sprintf("unresolved_challenge_digests[%d]", i), digest); err != nil {
			return err
		}
		if _, exists := seenChallenges[digest]; exists {
			return fmt.Errorf("duplicate unresolved challenge digest %q", digest)
		}
		seenChallenges[digest] = struct{}{}
	}
	return nil
}

func validateAcyclic(nodes map[string]Node) error {
	const (
		unseen = iota
		visiting
		visited
	)
	state := make(map[string]int, len(nodes))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("capability graph contains a cycle through %q", id)
		case visited:
			return nil
		}
		state[id] = visiting
		for _, prerequisite := range nodes[id].Prerequisites {
			if err := visit(prerequisite); err != nil {
				return err
			}
		}
		state[id] = visited
		return nil
	}
	ids := sortedMapKeys(nodes)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func validateCrownAncestry(crownID string, nodes map[string]Node) error {
	seen := make(map[string]struct{})
	var walk func(string)
	walk = func(id string) {
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		for _, prerequisite := range nodes[id].Prerequisites {
			walk(prerequisite)
		}
	}
	walk(crownID)
	if len(seen) != len(nodes) {
		for _, nodeID := range sortedMapKeys(nodes) {
			if _, ok := seen[nodeID]; !ok {
				return fmt.Errorf("node %q is not in crown ancestry", nodeID)
			}
		}
	}
	required := []string{"GROUND", "CAPABILITY", "INDUSTRIAL", "RECURSIVE"}
	for _, stage := range required {
		found := false
		for id := range seen {
			if nodes[id].Stage == stage {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("crown node %q has no %s ancestor", crownID, stage)
		}
	}
	return nil
}

func validateArtifact(path string, artifact ArtifactRef) error {
	if err := validateBoundedText(path+".name", artifact.Name, 1, 256); err != nil {
		return err
	}
	if err := validateBoundedText(path+".media_type", artifact.MediaType, 3, 128); err != nil {
		return err
	}
	if !mediaType.MatchString(artifact.MediaType) {
		return fmt.Errorf("%s.media_type must be a media type", path)
	}
	if err := validateDigest(path+".digest", artifact.Digest); err != nil {
		return err
	}
	if artifact.SourceURI != "" {
		if err := validateHTTPS(path+".source_uri", artifact.SourceURI); err != nil {
			return err
		}
	}
	return nil
}

func validateID(path, value string) error {
	if !boundedID.MatchString(value) {
		return fmt.Errorf("%s must match %s", path, boundedID.String())
	}
	return nil
}

func validateUniqueIDs(path string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		if err := validateID(fmt.Sprintf("%s[%d]", path, i), value); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%s contains duplicate id %q", path, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateDigest(path, value string) error {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must be sha256:<64 lowercase hex>", path)
	}
	for _, character := range value[len("sha256:"):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return fmt.Errorf("%s must be sha256:<64 lowercase hex>", path)
		}
	}
	return nil
}

func validateHTTPS(path, value string) error {
	if len(value) > 2048 {
		return fmt.Errorf("%s exceeds 2048 bytes", path)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", path, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be absolute HTTPS without userinfo, query, or fragment", path)
	}
	return nil
}

func validateBoundedText(path, value string, minimum, maximum int) error {
	if len(value) < minimum || len(value) > maximum {
		return fmt.Errorf("%s length must be between %d and %d bytes", path, minimum, maximum)
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must not have leading or trailing whitespace", path)
	}
	return nil
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
