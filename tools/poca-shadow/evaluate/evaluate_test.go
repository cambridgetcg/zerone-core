package evaluate

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestEvaluateUnlocksCrownAndNeverRewards(t *testing.T) {
	certificate, err := Evaluate(validProfile(), validEvidence())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if certificate.CrownStatus != statusDeclaredPass {
		t.Fatalf("crown status: want %q, got %q", statusDeclaredPass, certificate.CrownStatus)
	}
	if certificate.AttainedTier != "E5_REUSED" {
		t.Fatalf("attained tier: want E5_REUSED, got %q", certificate.AttainedTier)
	}
	if certificate.Assurance != CertificateAssurance {
		t.Fatalf("assurance: want %q, got %q", CertificateAssurance, certificate.Assurance)
	}
	if certificate.Reward.EconomicEffect != EconomicEffectNone ||
		certificate.Reward.AmountUzrn != "0" ||
		certificate.Reward.Reason != "shadow-profile-v0" {
		t.Fatalf("v0 emitted an economic effect: %#v", certificate.Reward)
	}
	if err := validateDigest("claim_id", certificate.ClaimID); err != nil {
		t.Fatalf("claim id: %v", err)
	}
}

func TestWrapInTotoBindsOuterAndInnerSubject(t *testing.T) {
	statement, err := EvaluateInToto(validProfile(), validEvidence())
	if err != nil {
		t.Fatalf("WrapInToto: %v", err)
	}
	if statement.Type != InTotoStatementV1 || statement.PredicateType != PredicateTypeV0 {
		t.Fatalf("unexpected statement header: %#v", statement)
	}
	if len(statement.Subject) != 1 {
		t.Fatalf("subject count: want 1, got %d", len(statement.Subject))
	}
	if statement.Subject[0].Name != statement.Predicate.Subject.Name ||
		"sha256:"+statement.Subject[0].Digest["sha256"] != statement.Predicate.Subject.Digest {
		t.Fatalf("outer and inner subjects drifted: outer=%#v inner=%#v", statement.Subject[0], statement.Predicate.Subject)
	}
	if statement.Predicate.Reward.EconomicEffect != EconomicEffectNone ||
		statement.Predicate.Reward.AmountUzrn != "0" {
		t.Fatalf("wrapped predicate emitted reward: %#v", statement.Predicate.Reward)
	}

	profile := validProfile()
	profile.Economics.AmountUzrn = "1"
	if _, err := EvaluateInToto(profile, validEvidence()); err == nil || !strings.Contains(err.Error(), "economics") {
		t.Fatalf("expected reward-bearing profile rejection, got %v", err)
	}
}

func TestEvaluateIsDeterministicAcrossInputOrdering(t *testing.T) {
	profile := validProfile()
	evidence := validEvidence()
	first, err := Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("first Evaluate: %v", err)
	}

	slices.Reverse(profile.Standards)
	slices.Reverse(profile.Requirements)
	slices.Reverse(profile.Nodes)
	for i := range profile.Nodes {
		slices.Reverse(profile.Nodes[i].Prerequisites)
		slices.Reverse(profile.Nodes[i].RequirementIDs)
	}
	slices.Reverse(evidence.Participants)
	slices.Reverse(evidence.Evidence)
	second, err := Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("second Evaluate: %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("output changed with input ordering\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
}

func TestClaimIDIsStableAcrossEvidenceEvolution(t *testing.T) {
	profile := validProfile()
	evidence := validEvidence()
	first, err := Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("first Evaluate: %v", err)
	}

	evidence.Evidence[0].ID = "renamed-problem-contract"
	evidence.Evidence[0].SourceURI = "https://renamed.fixture.invalid/evidence"
	evidence.UnresolvedChallengeDigests = []string{digest("f")}
	second, err := Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("second Evaluate: %v", err)
	}
	if first.ClaimID != second.ClaimID {
		t.Fatalf("evidence evolution changed stable claim id: %s != %s", first.ClaimID, second.ClaimID)
	}
	if first.EvidenceBundleDigest == second.EvidenceBundleDigest {
		t.Fatal("evidence evolution did not change snapshot digest")
	}

	evidence.BaselineDigest = digest("e")
	third, err := Evaluate(profile, evidence)
	if err != nil {
		t.Fatalf("third Evaluate: %v", err)
	}
	if third.ClaimID == second.ClaimID {
		t.Fatal("baseline change did not change claim id")
	}
}

func TestEvaluateBlocksInsufficientIndependence(t *testing.T) {
	evidence := validEvidence()
	for i := range evidence.Participants {
		if evidence.Participants[i].ID == "verifier-b" {
			evidence.Participants[i].ControlClusterClaim = "cluster.verifier-a"
		}
	}
	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	rebuild := findNodeResult(t, certificate, "capability.independent-rebuild")
	if rebuild.Status != statusBlocked || !slices.Contains(rebuild.RefusalCodes, "INSUFFICIENT_INDEPENDENCE") {
		t.Fatalf("rebuild should be independence-blocked: %#v", rebuild)
	}
	if certificate.CrownStatus != statusBlocked {
		t.Fatalf("crown status: want BLOCKED, got %q", certificate.CrownStatus)
	}
}

func TestEvaluateReportsNoAttainedTierWithoutPassingNode(t *testing.T) {
	evidence := validEvidence()
	evidence.Evidence = []Receipt{}
	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if certificate.AttainedTier != NoAttainedTier {
		t.Fatalf("attained tier: want %q, got %q", NoAttainedTier, certificate.AttainedTier)
	}
}

func TestEvaluateCollapsesDuplicateEvidencePayloads(t *testing.T) {
	evidence := validEvidence()
	var first Receipt
	for _, receipt := range evidence.Evidence {
		if receipt.ID == "rebuild-a" {
			first = receipt
		}
	}
	for i := range evidence.Evidence {
		if evidence.Evidence[i].ID == "rebuild-b" {
			evidence.Evidence[i].StatementDigest = first.StatementDigest
			evidence.Evidence[i].VerificationReceiptDigest = first.VerificationReceiptDigest
		}
	}

	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	rebuild := findNodeResult(t, certificate, "capability.independent-rebuild")
	if rebuild.Status != statusBlocked || !slices.Contains(rebuild.RefusalCodes, "EVIDENCE_MISSING") {
		t.Fatalf("rebuild should be duplicate-blocked: %#v", rebuild)
	}
	if !containsSubstring(certificate.Notices, "collapsed duplicate evidence rebuild-b") {
		t.Fatalf("missing duplicate-collapse notice: %#v", certificate.Notices)
	}
}

func TestDuplicateCollapseDoesNotDependOnCallerIDs(t *testing.T) {
	assertResult := func(t *testing.T, internalID string) {
		t.Helper()
		evidence := validEvidence()
		evidence.Participants = append(evidence.Participants, Participant{
			ID:                  "claimant-verifier",
			Role:                "VERIFIER",
			Identity:            "cosmos:zerone-2:zrn1claimant",
			ControlClusterClaim: "cluster.claimant",
		})
		var duplicate Receipt
		for _, receipt := range evidence.Evidence {
			if receipt.ID == "rebuild-a" {
				duplicate = receipt
				break
			}
		}
		duplicate.ID = internalID
		duplicate.ObserverParticipantID = "claimant-verifier"
		evidence.Evidence = append(evidence.Evidence, duplicate)

		certificate, err := Evaluate(validProfile(), evidence)
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		rebuild := findNodeResult(t, certificate, "capability.independent-rebuild")
		if rebuild.Status != statusDeclaredPass ||
			!slices.Contains(rebuild.AcceptedEvidenceIDs, "rebuild-a") ||
			slices.Contains(rebuild.AcceptedEvidenceIDs, internalID) {
			t.Fatalf("caller-controlled duplicate affected result: %#v", rebuild)
		}
	}
	t.Run("internal sorts first", func(t *testing.T) {
		assertResult(t, "a-internal")
	})
	t.Run("internal sorts last", func(t *testing.T) {
		assertResult(t, "z-internal")
	})
}

func TestEvaluateFailsHardGuardrailAndBlocksDescendants(t *testing.T) {
	evidence := validEvidence()
	for i := range evidence.Evidence {
		if evidence.Evidence[i].ID == "problem-contract" {
			evidence.Evidence[i].Result = "FAIL"
		}
	}

	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	ground := findNodeResult(t, certificate, "ground.problem-contract")
	if ground.Status != statusFailed || !slices.Contains(ground.RefusalCodes, "HARD_GUARDRAIL_FAILED") {
		t.Fatalf("ground node should fail closed: %#v", ground)
	}
	crown := findNodeResult(t, certificate, "crown.constructive-adaptation")
	if crown.Status != statusBlocked || !slices.Contains(crown.RefusalCodes, "PREREQUISITE_NOT_MET") {
		t.Fatalf("crown should be prerequisite-blocked: %#v", crown)
	}
}

func TestMatchingFailureIsNotDiscardedForLackOfIndependence(t *testing.T) {
	evidence := validEvidence()
	for i := range evidence.Participants {
		if evidence.Participants[i].ID == "sponsor" {
			evidence.Participants[i].ControlClusterClaim = "cluster.claimant"
		}
	}
	for i := range evidence.Evidence {
		if evidence.Evidence[i].ID == "problem-contract" {
			evidence.Evidence[i].Result = "FAIL"
		}
	}
	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	ground := findNodeResult(t, certificate, "ground.problem-contract")
	if ground.Status != statusFailed || !slices.Contains(ground.RefusalCodes, "HARD_GUARDRAIL_FAILED") {
		t.Fatalf("matching internal failure should fail closed: %#v", ground)
	}
}

func TestEvaluateOpenChallengeBlocksOnlyCrown(t *testing.T) {
	evidence := validEvidence()
	evidence.UnresolvedChallengeDigests = []string{digest("f")}
	certificate, err := Evaluate(validProfile(), evidence)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	industrial := findNodeResult(t, certificate, "industrial.independent-adoption")
	if industrial.Status != statusDeclaredPass {
		t.Fatalf("industrial node unexpectedly blocked: %#v", industrial)
	}
	crown := findNodeResult(t, certificate, "crown.constructive-adaptation")
	if crown.Status != statusBlocked || !slices.Contains(crown.RefusalCodes, "CHALLENGE_OPEN") {
		t.Fatalf("crown should be challenge-blocked: %#v", crown)
	}
}

func TestEvaluateDraftProfileCannotUnlockCrown(t *testing.T) {
	profile := validProfile()
	profile.Status = "DRAFT"
	certificate, err := Evaluate(profile, validEvidence())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	crown := findNodeResult(t, certificate, "crown.constructive-adaptation")
	if crown.Status != statusBlocked || !slices.Contains(crown.RefusalCodes, "PROFILE_NOT_RATIFIED") {
		t.Fatalf("draft profile should block crown: %#v", crown)
	}
	if findNodeResult(t, certificate, "industrial.independent-adoption").Status != statusDeclaredPass {
		t.Fatal("draft profile should not erase lower-node shadow results")
	}
}

func TestEvaluateRejectsCrossDocumentDrift(t *testing.T) {
	t.Run("profile version", func(t *testing.T) {
		evidence := validEvidence()
		evidence.ProfileVersion = "0.2.0"
		_, err := Evaluate(validProfile(), evidence)
		if err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("expected profile mismatch, got %v", err)
		}
	})

	t.Run("verification rule", func(t *testing.T) {
		evidence := validEvidence()
		evidence.Evidence[0].VerificationRule = "ALL_TEST_CASES_PASS"
		_, err := Evaluate(validProfile(), evidence)
		if err == nil || !strings.Contains(err.Error(), "verification_rule") {
			t.Fatalf("expected verification-rule mismatch, got %v", err)
		}
	})

	t.Run("policy digest", func(t *testing.T) {
		evidence := validEvidence()
		evidence.Evidence[0].PolicyDigest = digest("f")
		_, err := Evaluate(validProfile(), evidence)
		if err == nil || !strings.Contains(err.Error(), "policy_digest") {
			t.Fatalf("expected policy-digest mismatch, got %v", err)
		}
	})
}

func TestEvaluateRejectsConflictingControlClustersForSameIdentity(t *testing.T) {
	evidence := validEvidence()
	evidence.Participants[2].Identity = evidence.Participants[1].Identity

	_, err := Evaluate(validProfile(), evidence)
	if err == nil || !strings.Contains(err.Error(), "conflicting control clusters") {
		t.Fatalf("expected identity-to-cluster conflict rejection, got %v", err)
	}
}

func TestParseRejectsAmbiguousOrOversizedJSON(t *testing.T) {
	valid, err := json.Marshal(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	duplicate := strings.Replace(string(valid), `"schema":`, `"schema":"zerone.standard-profile/v0","schema":`, 1)
	if _, err := ParseProfile([]byte(duplicate)); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("expected duplicate-key rejection, got %v", err)
	}

	unknown := strings.Replace(string(valid), `"title":`, `"unknown":true,"title":`, 1)
	if _, err := ParseProfile([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}

	oversized := make([]byte, maxDocumentBytes+1)
	if _, err := ParseProfile(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestParseRequiresPolicyBearingFields(t *testing.T) {
	profileBytes, err := json.Marshal(validProfile())
	if err != nil {
		t.Fatal(err)
	}
	evidenceBytes, err := json.Marshal(validEvidence())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("challenge policy boolean", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		delete(document["challenge_policy"].(map[string]any), "unresolved_challenge_blocks_crown")
		assertMissingFieldRejected(t, document, ParseProfile, "unresolved_challenge_blocks_crown")
	})

	t.Run("independence boolean", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		requirements := document["requirements"].([]any)
		delete(requirements[0].(map[string]any), "require_observer_independent_from_claimant")
		assertMissingFieldRejected(t, document, ParseProfile, "require_observer_independent_from_claimant")
	})

	t.Run("hard guardrail boolean", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		requirements := document["requirements"].([]any)
		delete(requirements[0].(map[string]any), "hard_guardrail")
		assertMissingFieldRejected(t, document, ParseProfile, "hard_guardrail")
	})

	t.Run("node prerequisites", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		nodes := document["nodes"].([]any)
		delete(nodes[0].(map[string]any), "prerequisites")
		assertMissingFieldRejected(t, document, ParseProfile, "prerequisites")
	})

	t.Run("challenge list", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(evidenceBytes, &document); err != nil {
			t.Fatal(err)
		}
		delete(document, "unresolved_challenge_digests")
		assertMissingFieldRejected(t, document, ParseEvidence, "unresolved_challenge_digests")
	})

	t.Run("null boolean", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		document["challenge_policy"].(map[string]any)["unresolved_challenge_blocks_crown"] = nil
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseProfile(encoded); err == nil || !strings.Contains(err.Error(), "null") {
			t.Fatalf("expected null rejection, got %v", err)
		}
	})

	t.Run("empty optional predicate", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(profileBytes, &document); err != nil {
			t.Fatal(err)
		}
		requirements := document["requirements"].([]any)
		requirements[0].(map[string]any)["predicate_type"] = ""
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseProfile(encoded); err == nil || !strings.Contains(err.Error(), "omitted rather than empty") {
			t.Fatalf("expected empty predicate rejection, got %v", err)
		}
	})

	t.Run("empty optional source", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(evidenceBytes, &document); err != nil {
			t.Fatal(err)
		}
		document["subject"].(map[string]any)["source_uri"] = ""
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseEvidence(encoded); err == nil || !strings.Contains(err.Error(), "omitted rather than empty") {
			t.Fatalf("expected empty source rejection, got %v", err)
		}
	})
}

func TestProfileRejectsRewardBearingEconomicsAndCycles(t *testing.T) {
	t.Run("reward", func(t *testing.T) {
		profile := validProfile()
		profile.Economics.AmountUzrn = "1"
		if _, err := Evaluate(profile, validEvidence()); err == nil || !strings.Contains(err.Error(), "economics") {
			t.Fatalf("expected economics rejection, got %v", err)
		}
	})

	t.Run("cycle", func(t *testing.T) {
		profile := validProfile()
		for i := range profile.Nodes {
			switch profile.Nodes[i].ID {
			case "capability.independent-rebuild":
				profile.Nodes[i].Prerequisites = []string{"capability.second"}
			}
		}
		profile.Nodes = append(profile.Nodes, Node{
			ID:             "capability.second",
			Stage:          "CAPABILITY",
			Tier:           "E2_CONFORMANT",
			Title:          "Second capability",
			Prerequisites:  []string{"capability.independent-rebuild"},
			RequirementIDs: []string{"capability.independent-rebuild"},
		})
		if _, err := Evaluate(profile, validEvidence()); err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("expected cycle rejection, got %v", err)
		}
	})

	t.Run("unreferenced requirement", func(t *testing.T) {
		profile := validProfile()
		profile.Requirements = append(profile.Requirements, Requirement{
			ID: "unused.guardrail", Kind: "SECURITY_ASSESSMENT",
			VerificationRule: "ALL_TEST_CASES_PASS", PolicyDigest: digest("f"),
			ObserverRole: "VERIFIER", MinCount: 1, MinIndependentControlClusters: 1,
			RequireObserverIndependentFromClaimant: true, HardGuardrail: true,
		})
		if _, err := Evaluate(profile, validEvidence()); err == nil || !strings.Contains(err.Error(), "not referenced") {
			t.Fatalf("expected unreferenced requirement rejection, got %v", err)
		}
	})

	t.Run("disconnected node", func(t *testing.T) {
		profile := validProfile()
		profile.Nodes = append(profile.Nodes, Node{
			ID: "industrial.disconnected", Stage: "INDUSTRIAL", Tier: "E4_TRANSFERRED",
			Title:          "Disconnected industrial branch",
			RequirementIDs: []string{"industrial.independent-adoption"},
		})
		if _, err := Evaluate(profile, validEvidence()); err == nil || !strings.Contains(err.Error(), "not in crown ancestry") {
			t.Fatalf("expected disconnected node rejection, got %v", err)
		}
	})
}

func validProfile() Profile {
	return Profile{
		Schema:         ProfileSchema,
		ProfileID:      "zerone.release.slsa-build-l2",
		ProfileVersion: "0.1.0",
		Title:          "Zerone release adaptation",
		Status:         "DECLARED_RATIFIED",
		AssuranceMode:  AssuranceModeShadowOnly,
		Standards: []StandardRef{
			{URI: "https://slsa.dev/spec/v1.2/", Version: "1.2", Status: "APPROVED", Target: "Build L2"},
			{URI: "https://in-toto.io/Statement/v1", Version: "1", Status: "STABLE", Target: "Statement envelope"},
		},
		Requirements: []Requirement{
			{
				ID: "ground.problem-contract", Kind: "PROBLEM_CONTRACT",
				VerificationRule: "FROZEN_BASELINE_AND_GUARDRAILS", ObserverRole: "SPONSOR",
				PolicyDigest: digest("1"),
				MinCount:     1, MinIndependentControlClusters: 1,
				RequireObserverIndependentFromClaimant: true, HardGuardrail: true,
			},
			{
				ID: "capability.independent-rebuild", Kind: "INDEPENDENT_REBUILD",
				VerificationRule: "BYTE_IDENTICAL_SUBJECT", ObserverRole: "VERIFIER",
				PolicyDigest: digest("2"),
				MinCount:     2, MinIndependentControlClusters: 2,
				RequireObserverIndependentFromClaimant: true, HardGuardrail: false,
			},
			{
				ID: "industrial.independent-adoption", Kind: "INDEPENDENT_ADOPTION",
				VerificationRule: "INDEPENDENT_ADOPTION_RECEIPT", ObserverRole: "ADOPTER",
				PolicyDigest: digest("3"),
				MinCount:     2, MinIndependentControlClusters: 2,
				RequireObserverIndependentFromClaimant: true, HardGuardrail: false,
			},
			{
				ID: "recursive.useful-work", Kind: "ZERONE_RECURSIVE_VALUE",
				VerificationRule: "ZERONE_RECURSION_LINK", ObserverRole: "VERIFIER",
				PolicyDigest: digest("4"),
				MinCount:     2, MinIndependentControlClusters: 2,
				RequireObserverIndependentFromClaimant: true, HardGuardrail: false,
			},
		},
		Nodes: []Node{
			{
				ID: "ground.problem-contract", Stage: "GROUND", Tier: "E1_REPRODUCED",
				Title: "Frozen problem contract", Prerequisites: []string{},
				RequirementIDs: []string{"ground.problem-contract"},
			},
			{
				ID: "capability.independent-rebuild", Stage: "CAPABILITY", Tier: "E2_CONFORMANT",
				Title: "Independent rebuild", Prerequisites: []string{"ground.problem-contract"},
				RequirementIDs: []string{"capability.independent-rebuild"},
			},
			{
				ID: "industrial.independent-adoption", Stage: "INDUSTRIAL", Tier: "E4_TRANSFERRED",
				Title: "Independent adoption", Prerequisites: []string{"capability.independent-rebuild"},
				RequirementIDs: []string{"industrial.independent-adoption"},
			},
			{
				ID: "recursive.useful-work", Stage: "RECURSIVE", Tier: "E5_REUSED",
				Title: "Recursive useful work", Prerequisites: []string{"capability.independent-rebuild"},
				RequirementIDs: []string{"recursive.useful-work"},
			},
			{
				ID: "crown.constructive-adaptation", Stage: "CROWN", Tier: "E5_REUSED",
				Title:          "Industrial constructive adaptation",
				Prerequisites:  []string{"industrial.independent-adoption", "recursive.useful-work"},
				RequirementIDs: []string{},
			},
		},
		CrownNodeID: "crown.constructive-adaptation",
		ChallengePolicy: ChallengePolicy{
			UnresolvedChallengeBlocksCrown: true,
		},
		Economics: EconomicsPolicy{Mode: EconomicEffectNone, AmountUzrn: "0"},
	}
}

func validEvidence() EvidenceBundle {
	subjectDigest := digest("a")
	participants := []Participant{
		{ID: "claimant", Role: "CLAIMANT", Identity: "cosmos:zerone-2:zrn1claimant", ControlClusterClaim: "cluster.claimant"},
		{ID: "sponsor", Role: "SPONSOR", Identity: "https://sponsor.fixture.invalid", ControlClusterClaim: "cluster.sponsor"},
		{ID: "verifier-a", Role: "VERIFIER", Identity: "https://verifier-a.fixture.invalid", ControlClusterClaim: "cluster.verifier-a"},
		{ID: "verifier-b", Role: "VERIFIER", Identity: "https://verifier-b.fixture.invalid", ControlClusterClaim: "cluster.verifier-b"},
		{ID: "adopter-a", Role: "ADOPTER", Identity: "https://adopter-a.fixture.invalid", ControlClusterClaim: "cluster.adopter-a"},
		{ID: "adopter-b", Role: "ADOPTER", Identity: "https://adopter-b.fixture.invalid", ControlClusterClaim: "cluster.adopter-b"},
	}
	receipt := func(id, requirement, producer, observer, rule, policy, environment, statement, verification string) Receipt {
		return Receipt{
			ID: id, RequirementID: requirement,
			ProducerParticipantID: producer, ObserverParticipantID: observer,
			Result: "PASS", VerificationRule: rule, SubjectDigest: subjectDigest,
			PolicyDigest: digest(policy), EnvironmentDigest: digest(environment),
			StatementDigest: digest(statement), VerificationReceiptDigest: digest(verification),
		}
	}
	return EvidenceBundle{
		Schema:         EvidenceSchema,
		ProfileID:      "zerone.release.slsa-build-l2",
		ProfileVersion: "0.1.0",
		Subject: ArtifactRef{
			Name: "zeroned-linux-amd64", MediaType: "application/octet-stream", Digest: subjectDigest,
		},
		BaselineDigest: digest("b"),
		LineageDigest:  digest("c"),
		Participants:   participants,
		Evidence: []Receipt{
			receipt("problem-contract", "ground.problem-contract", "claimant", "sponsor", "FROZEN_BASELINE_AND_GUARDRAILS", "1", "5", "6", "7"),
			receipt("rebuild-a", "capability.independent-rebuild", "claimant", "verifier-a", "BYTE_IDENTICAL_SUBJECT", "2", "8", "9", "0"),
			receipt("rebuild-b", "capability.independent-rebuild", "claimant", "verifier-b", "BYTE_IDENTICAL_SUBJECT", "2", "9", "a", "b"),
			receipt("adoption-a", "industrial.independent-adoption", "claimant", "adopter-a", "INDEPENDENT_ADOPTION_RECEIPT", "3", "a", "c", "d"),
			receipt("adoption-b", "industrial.independent-adoption", "claimant", "adopter-b", "INDEPENDENT_ADOPTION_RECEIPT", "3", "b", "d", "e"),
			receipt("recursion-a", "recursive.useful-work", "claimant", "verifier-a", "ZERONE_RECURSION_LINK", "4", "c", "e", "f"),
			receipt("recursion-b", "recursive.useful-work", "claimant", "verifier-b", "ZERONE_RECURSION_LINK", "4", "d", "f", "1"),
		},
		UnresolvedChallengeDigests: []string{},
	}
}

func digest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}

func findNodeResult(t *testing.T, certificate Certificate, nodeID string) NodeResult {
	t.Helper()
	for _, result := range certificate.NodeResults {
		if result.NodeID == nodeID {
			return result
		}
	}
	t.Fatalf("node result %q not found", nodeID)
	return NodeResult{}
}

func containsSubstring(values []string, substring string) bool {
	for _, value := range values {
		if strings.Contains(value, substring) {
			return true
		}
	}
	return false
}

func assertMissingFieldRejected[T any](
	t *testing.T,
	document map[string]any,
	parse func([]byte) (T, error),
	field string,
) {
	t.Helper()
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parse(encoded); err == nil || !strings.Contains(err.Error(), field) {
		t.Fatalf("expected missing %s rejection, got %v", field, err)
	}
}
