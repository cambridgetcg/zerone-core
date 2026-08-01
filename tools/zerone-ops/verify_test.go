package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func testSHA(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func testPublicKey(seedByte byte) string {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	return hex.EncodeToString(privateKey.Public().(ed25519.PublicKey))
}

func testPowerQuorum(totalPower string) PowerQuorum {
	_ = totalPower
	return PowerQuorum{
		Role:        validatorOperatorRole,
		Numerator:   2,
		Denominator: 3,
		Strict:      true,
	}
}

func testTrustPolicy(incidentID, releaseID string) TrustPolicy {
	weak := ApprovalPolicy{
		MinimumApprovals:          1,
		MinimumDistinctIdentities: 1,
		RequiredRoles:             []string{"operations-approver"},
		SeparatedRolePairs:        []RolePair{},
		PowerQuorum:               PowerQuorum{},
	}
	activation := ApprovalPolicy{
		MinimumApprovals:          1,
		MinimumDistinctIdentities: 1,
		RequiredRoles:             []string{"validator-operator"},
		SeparatedRolePairs:        []RolePair{},
		PowerQuorum:               testPowerQuorum("10"),
	}
	policy := TrustPolicy{
		Schema:           trustPolicySchema,
		PolicyID:         "zerone-test-policy",
		ChainID:          "zerone-1",
		IncidentID:       incidentID,
		ReleaseID:        releaseID,
		ApprovalPolicies: []EdgeApprovalPolicy{},
		Signers: []TrustedSigner{
			{
				Role:      evidenceCustodianRole,
				Identity:  "did:zrn:evidence-custodian",
				PublicKey: testPublicKey(7),
			},
			{
				Role:      governanceCoordinatorRole,
				Identity:  "did:zrn:governance-coordinator",
				PublicKey: testPublicKey(2),
			},
			{
				Role:      ibcLeadRole,
				Identity:  "did:zrn:ibc-lead",
				PublicKey: testPublicKey(3),
			},
			{
				Role:      incidentCommanderRole,
				Identity:  "did:zrn:incident-commander",
				PublicKey: testPublicKey(4),
			},
			{
				Role:      "operations-approver",
				Identity:  "did:zrn:approver-1",
				PublicKey: testPublicKey(1),
			},
			{
				Role:      policyRotationAuthorityRole,
				Identity:  "did:zrn:policy-rotation-authority",
				PublicKey: testPublicKey(8),
			},
			{
				Role:      releaseAuthorRole,
				Identity:  "did:zrn:release-author",
				PublicKey: testPublicKey(13),
			},
			{
				Role:      releaseVerifierRole,
				Identity:  "did:zrn:release-verifier",
				PublicKey: testPublicKey(5),
			},
			{
				Role:      supplyVerifierRole,
				Identity:  "did:zrn:supply-verifier",
				PublicKey: testPublicKey(6),
			},
			{
				Role:      "validator-operator",
				Identity:  "did:zrn:validator-a",
				PublicKey: testPublicKey(9),
			},
			{
				Role:      "validator-operator",
				Identity:  "did:zrn:validator-b",
				PublicKey: testPublicKey(10),
			},
		},
	}
	appendLane := func(journalLane lane, transitions map[State]map[State]bool) {
		for from, destinations := range transitions {
			for to, allowed := range destinations {
				if !allowed {
					continue
				}
				approvalPolicy := weak
				edge := EdgeApprovalPolicy{
					Lane:           journalLane,
					From:           from,
					To:             to,
					ApprovalPolicy: approvalPolicy,
				}
				if edgeRequiresStrictOperatorSupermajority(edge) {
					edge.ApprovalPolicy = activation
				}
				mandatoryRoles := mandatoryRolesForEdge(edge)
				if len(mandatoryRoles) != 0 {
					edge.ApprovalPolicy.RequiredRoles = mandatoryRoles
					edge.ApprovalPolicy.SeparatedRolePairs =
						mandatorySeparatedRolePairsForEdge(edge)
					edge.ApprovalPolicy.MinimumApprovals =
						uint64(len(mandatoryRoles))
					edge.ApprovalPolicy.MinimumDistinctIdentities =
						uint64(len(mandatoryRoles))
				}
				policy.ApprovalPolicies = append(policy.ApprovalPolicies, edge)
			}
		}
	}
	if incidentID != "" {
		appendLane(laneIncident, incidentTransitions)
	}
	if releaseID != "" {
		appendLane(laneRelease, releaseTransitions)
	}
	sort.Slice(policy.ApprovalPolicies, func(i, j int) bool {
		return edgeApprovalPolicyLess(policy.ApprovalPolicies[i], policy.ApprovalPolicies[j])
	})
	sort.Slice(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	})
	return policy
}

func testPolicyForTransition(transition Transition) TrustPolicy {
	return testTrustPolicy(transition.IncidentID, transition.ReleaseID)
}

func mustTestPolicySHA256(t *testing.T, policy TrustPolicy) string {
	t.Helper()
	digest, err := trustPolicySHA256(policy)
	if err != nil {
		t.Fatalf("hash trust policy: %v", err)
	}
	return digest
}

func testVerifyOptions(t *testing.T, transition Transition) VerifyOptions {
	t.Helper()
	policy := testPolicyForTransition(transition)
	return VerifyOptions{
		TrustPolicy:       &policy,
		TrustPolicySHA256: mustTestPolicySHA256(t, policy),
	}
}

func testVerifyOptionsForTransitions(
	t *testing.T,
	transitions []Transition,
) VerifyOptions {
	t.Helper()
	options := testVerifyOptions(t, transitions[0])
	options.ExpectedPowerSnapshotSHA256 = make(map[uint64]string)
	for _, transition := range transitions {
		if transition.PowerSnapshot.Schema != "" {
			options.ExpectedPowerSnapshotSHA256[transition.Sequence] =
				transition.PowerSnapshot.SnapshotSHA256
		}
	}
	return options
}

func testEvidence(sequence uint64, journalLane lane, to State) []Evidence {
	types := requiredEvidenceTypes(journalLane, to)
	evidence := make([]Evidence, 0, len(types))
	for _, evidenceType := range types {
		digest := testSHA(
			"evidence-" + strconv.FormatUint(sequence, 10) + "-" + evidenceType,
		)
		if evidenceType == "validator-power-snapshot" {
			digest = testSHA("validator-set-snapshot")
		}
		evidence = append(evidence, Evidence{
			Type:   evidenceType,
			SHA256: digest,
			URI:    "vault://zerone/test",
		})
	}
	sort.Slice(evidence, func(i, j int) bool {
		return evidenceLess(evidence[i], evidence[j])
	})
	return evidence
}

func addTestPowerSnapshotEvidence(transition *Transition) {
	setTestPowerSnapshot(transition, "7", "3")
}

func setTestPowerSnapshot(
	transition *Transition,
	validatorAPower, validatorBPower string,
) {
	transition.PowerSnapshot = testPowerSnapshot(
		*transition,
		validatorAPower,
		validatorBPower,
	)
	found := false
	for i := range transition.Evidence {
		if transition.Evidence[i].Type == "validator-power-snapshot" {
			transition.Evidence[i].SHA256 = transition.PowerSnapshot.SnapshotSHA256
			found = true
		}
	}
	if !found {
		transition.Evidence = append(transition.Evidence, Evidence{
			Type:   "validator-power-snapshot",
			SHA256: transition.PowerSnapshot.SnapshotSHA256,
			URI:    "vault://zerone/test",
		})
	}
	sort.Slice(transition.Evidence, func(i, j int) bool {
		return evidenceLess(transition.Evidence[i], transition.Evidence[j])
	})
}

func testPowerSnapshot(
	transition Transition,
	validatorAPower, validatorBPower string,
) PowerSnapshot {
	totalA, _ := new(big.Int).SetString(validatorAPower, 10)
	totalB, _ := new(big.Int).SetString(validatorBPower, 10)
	total := new(big.Int).Add(totalA, totalB)
	snapshot := PowerSnapshot{
		Schema:        powerSnapshotSchema,
		ChainID:       transition.ChainID,
		Height:        transition.Checkpoint.Height,
		BlockIDSHA256: transition.Checkpoint.BlockIDSHA256,
		AppHashSHA256: transition.Checkpoint.AppHashSHA256,
		Role:          validatorOperatorRole,
		TotalPower:    total.String(),
		CapturedAt:    "2026-07-29T21:00:00Z",
		ValidUntil:    "2026-07-29T23:00:00Z",
		Members: []PowerSnapshotMember{
			{
				Identity:  "did:zrn:validator-a",
				PublicKey: testPublicKey(9),
				Power:     validatorAPower,
			},
			{
				Identity:  "did:zrn:validator-b",
				PublicKey: testPublicKey(10),
				Power:     validatorBPower,
			},
		},
	}
	snapshot.SnapshotSHA256, _ = powerSnapshotHash(snapshot)
	return snapshot
}

func releaseTransition(sequence uint64, from, to State, previous string) Transition {
	policy := testTrustPolicy("", "release-2026-07")
	policySHA256, _ := trustPolicySHA256(policy)
	return Transition{
		Schema:        transitionSchema,
		Lane:          laneRelease,
		Sequence:      sequence,
		From:          from,
		To:            to,
		Event:         "release-transition",
		OccurredAt:    "2026-07-29T22:00:00Z",
		ChainID:       "zerone-1",
		IncidentID:    "",
		ReleaseID:     "release-2026-07",
		ActorRole:     "release-coordinator",
		ActorIdentity: "did:zrn:coordinator",
		Checkpoint: Checkpoint{
			Height:        90,
			BlockIDSHA256: testSHA("block-90"),
			AppHashSHA256: testSHA("app-90"),
		},
		Release:                  ReleaseBinding{},
		PowerSnapshot:            emptyPowerSnapshot(),
		Evidence:                 testEvidence(sequence, laneRelease, to),
		Approvals:                []Approval{},
		TrustPolicySHA256:        policySHA256,
		PreviousTransitionSHA256: previous,
		TransitionSHA256:         "",
	}
}

func incidentTransition(sequence uint64, from, to State, previous string) Transition {
	policy := testTrustPolicy("ZR-2026-0001", "")
	policySHA256, _ := trustPolicySHA256(policy)
	return Transition{
		Schema:        transitionSchema,
		Lane:          laneIncident,
		Sequence:      sequence,
		From:          from,
		To:            to,
		Event:         "incident-transition",
		OccurredAt:    "2026-07-29T22:00:00Z",
		ChainID:       "zerone-1",
		IncidentID:    "ZR-2026-0001",
		ReleaseID:     "",
		ActorRole:     "incident-commander",
		ActorIdentity: "did:zrn:commander",
		Checkpoint: Checkpoint{
			Height:        90,
			BlockIDSHA256: testSHA("block-90"),
			AppHashSHA256: testSHA("app-90"),
		},
		Release:                  ReleaseBinding{},
		PowerSnapshot:            emptyPowerSnapshot(),
		Evidence:                 testEvidence(sequence, laneIncident, to),
		Approvals:                []Approval{},
		TrustPolicySHA256:        policySHA256,
		PreviousTransitionSHA256: previous,
		TransitionSHA256:         "",
	}
}

func frozenArtifacts() ReleaseBinding {
	return ReleaseBinding{
		PlanName:            "upgrade-test-v1",
		UpgradeHeight:       100,
		ActivationMode:      activationModeCosmovisor,
		PlanInfoSHA256:      testSHA("plan-info"),
		BinarySHA256:        testSHA("binary-a"),
		ImageSHA256:         "",
		ProvenanceSHA256:    testSHA("provenance"),
		SBOMSHA256:          testSHA("sbom"),
		StateManifestSHA256: "",
	}
}

func mustSeal(t *testing.T, transition Transition) Transition {
	t.Helper()
	policy := testPolicyForTransition(transition)
	policySHA256 := mustTestPolicySHA256(t, policy)
	if transition.TrustPolicySHA256 == "" {
		transition.TrustPolicySHA256 = policySHA256
	}
	if len(transition.Approvals) == 0 {
		edgePolicy, found := approvalPolicyForEdge(
			policy,
			transition.Lane,
			transition.From,
			transition.To,
		)
		if !found {
			t.Fatalf("test policy has no edge %s/%s->%s", transition.Lane, transition.From, transition.To)
		}
		if edgePolicy.PowerQuorum.Role == "validator-operator" {
			addTestPowerSnapshotEvidence(&transition)
		}
		transition.Approvals = make([]Approval, 0, len(edgePolicy.RequiredRoles))
		for _, role := range edgePolicy.RequiredRoles {
			identity, seed, power := testApprovalSigner(role)
			transition.Approvals = append(
				transition.Approvals,
				signApproval(t, transition, role, identity, seed, power),
			)
		}
		if len(transition.Approvals) == 0 {
			transition.Approvals = []Approval{
				signApproval(t, transition, "operations-approver", "did:zrn:approver-1", 1, "0"),
			}
		}
		sort.Slice(transition.Approvals, func(i, j int) bool {
			return approvalLess(transition.Approvals[i], transition.Approvals[j])
		})
	}
	if err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	); err != nil {
		t.Fatalf("validate transition trust: %v", err)
	}
	sealed, err := sealTransition(transition)
	if err != nil {
		t.Fatalf("seal transition: %v", err)
	}
	return sealed
}

func testApprovalSigner(role string) (identity string, seed byte, power string) {
	switch role {
	case governanceCoordinatorRole:
		return "did:zrn:governance-coordinator", 2, "0"
	case ibcLeadRole:
		return "did:zrn:ibc-lead", 3, "0"
	case incidentCommanderRole:
		return "did:zrn:incident-commander", 4, "0"
	case releaseVerifierRole:
		return "did:zrn:release-verifier", 5, "0"
	case supplyVerifierRole:
		return "did:zrn:supply-verifier", 6, "0"
	case evidenceCustodianRole:
		return "did:zrn:evidence-custodian", 7, "0"
	case policyRotationAuthorityRole:
		return "did:zrn:policy-rotation-authority", 8, "0"
	case releaseAuthorRole:
		return "did:zrn:release-author", 13, "0"
	case validatorOperatorRole:
		return "did:zrn:validator-a", 9, "7"
	default:
		return "did:zrn:approver-1", 1, "0"
	}
}

func sealWithoutEdgeValidation(t *testing.T, transition Transition) Transition {
	t.Helper()
	transition.TransitionSHA256 = ""
	transition.Approvals = []Approval{}
	transition.Approvals = []Approval{
		signApproval(t, transition, "operations-approver", "did:zrn:approver-1", 1, "0"),
	}
	hash, err := transitionHash(transition)
	if err != nil {
		t.Fatalf("hash transition: %v", err)
	}
	transition.TransitionSHA256 = hash
	return transition
}

func canonicalDocument(t *testing.T, transition Transition) []byte {
	t.Helper()
	document, err := canonicalTransition(transition)
	if err != nil {
		t.Fatalf("canonical transition: %v", err)
	}
	return document
}

func validReleaseJournal(t *testing.T) ([]Transition, [][]byte) {
	t.Helper()
	first := mustSeal(t, releaseTransition(1, StateRunning, StatePreparing, ""))
	secondDraft := releaseTransition(2, StatePreparing, StateReleaseFrozen, first.TransitionSHA256)
	secondDraft.Release = frozenArtifacts()
	second := mustSeal(t, secondDraft)
	thirdDraft := releaseTransition(3, StateReleaseFrozen, StateScheduled, second.TransitionSHA256)
	thirdDraft.Release = frozenArtifacts()
	third := mustSeal(t, thirdDraft)
	transitions := []Transition{first, second, third}
	return transitions, [][]byte{
		canonicalDocument(t, first),
		canonicalDocument(t, second),
		canonicalDocument(t, third),
	}
}

func validIncidentJournal(t *testing.T) ([]Transition, [][]byte) {
	t.Helper()
	first := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	second := mustSeal(t, incidentTransition(2, StateAssessing, StateContaining, first.TransitionSHA256))
	transitions := []Transition{first, second}
	return transitions, [][]byte{canonicalDocument(t, first), canonicalDocument(t, second)}
}

func TestVerifyValidReleaseJournal(t *testing.T) {
	transitions, documents := validReleaseJournal(t)
	options := testVerifyOptions(t, transitions[0])
	options.ExpectedChainID = "zerone-1"
	options.ExpectedReleaseID = "release-2026-07"
	options.ExpectedBinarySHA256 = frozenArtifacts().BinarySHA256
	options.ExpectedHeadSHA256 = transitions[len(transitions)-1].TransitionSHA256
	result, err := verifyDocuments(documents, options)
	if err != nil {
		t.Fatalf("verify valid journal: %v", err)
	}
	if result.Transitions != 3 || result.Lane != laneRelease || result.State != StateScheduled {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRejectBackwardStateTransition(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	backward := incidentTransition(3, StateContaining, StateAssessing, transitions[1].TransitionSHA256)
	backward = sealWithoutEdgeValidation(t, backward)
	documents = append(documents, canonicalDocument(t, backward))

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || (!strings.Contains(err.Error(), "is not allowed") &&
		!strings.Contains(err.Error(), "no approval policy")) {
		t.Fatalf("expected backward-state rejection, got %v", err)
	}
}

func TestRejectSkippedSequence(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	skipped := incidentTransition(4, StateContaining, StateSafetyStopped, transitions[1].TransitionSHA256)
	skipped = mustSeal(t, skipped)
	documents = append(documents, canonicalDocument(t, skipped))

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "sequence must be 3") {
		t.Fatalf("expected skipped-sequence rejection, got %v", err)
	}
}

func TestRejectWrongPreviousHash(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	wrong := incidentTransition(3, StateContaining, StateSafetyStopped, testSHA("wrong-previous"))
	wrong = mustSeal(t, wrong)
	documents = append(documents, canonicalDocument(t, wrong))

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "does not match transition 2 hash") {
		t.Fatalf("expected previous-hash rejection, got %v (head %s)", err, transitions[1].TransitionSHA256)
	}
}

func TestRejectUnknownField(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	document := documents[0]
	document = append(append([]byte{}, document[:len(document)-1]...), []byte(`,"unknown_field":true}`)...)

	_, err := verifyDocuments([][]byte{document}, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
}

func TestRejectNoncanonicalJSON(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	var indented bytes.Buffer
	if err := json.Indent(&indented, documents[0], "", "  "); err != nil {
		t.Fatal(err)
	}

	_, err := verifyDocuments([][]byte{indented.Bytes()}, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("expected noncanonical JSON rejection, got %v", err)
	}
}

func TestRejectChainIdentityMismatch(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	changed := transitions[1]
	changed.ChainID = "hostile-fork-1"
	changed = sealWithoutEdgeValidation(t, changed)
	documents[1] = canonicalDocument(t, changed)

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "trust policy chain_id") {
		t.Fatalf("expected chain identity rejection, got %v", err)
	}
}

func TestRejectJournalIdentityChanges(t *testing.T) {
	t.Run("incident ID", func(t *testing.T) {
		transitions, documents := validIncidentJournal(t)
		changed := transitions[1]
		changed.IncidentID = "ZR-2026-HOSTILE"
		changed = sealWithoutEdgeValidation(t, changed)
		documents[1] = canonicalDocument(t, changed)

		_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
		if err == nil || !strings.Contains(err.Error(), "trust policy incident_id") {
			t.Fatalf("expected incident identity rejection, got %v", err)
		}
	})

	t.Run("release ID", func(t *testing.T) {
		transitions, documents := validReleaseJournal(t)
		changed := transitions[1]
		changed.ReleaseID = "release-hostile-rewrite"
		changed = sealWithoutEdgeValidation(t, changed)
		documents[1] = canonicalDocument(t, changed)

		_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
		if err == nil || !strings.Contains(err.Error(), "trust policy release_id") {
			t.Fatalf("expected release identity rejection, got %v", err)
		}
	})
}

func TestRejectArtifactMismatch(t *testing.T) {
	transitions, documents := validReleaseJournal(t)
	changed := releaseTransition(4, StateScheduled, StateStaged, transitions[2].TransitionSHA256)
	changed.Release = frozenArtifacts()
	changed.Release.BinarySHA256 = testSHA("binary-b")
	changed = mustSeal(t, changed)
	documents = append(documents, canonicalDocument(t, changed))

	options := testVerifyOptionsForTransitions(t, append(transitions, changed))
	_, err := verifyDocuments(documents, options)
	if err == nil || !strings.Contains(err.Error(), "binary_sha256 changed after binding") {
		t.Fatalf("expected artifact-binding rejection, got %v", err)
	}

	_, documents = validReleaseJournal(t)
	options = testVerifyOptions(t, transitions[0])
	options.ExpectedBinarySHA256 = testSHA("external-binary")
	_, err = verifyDocuments(documents, options)
	if err == nil || !strings.Contains(err.Error(), "binary_sha256 mismatch") {
		t.Fatalf("expected externally anchored artifact rejection, got %v", err)
	}
}

func TestRejectCancellationAfterUpgradeHeightCommitted(t *testing.T) {
	transitions, documents := validReleaseJournal(t)
	cancelled := releaseTransition(4, StateScheduled, StateCancelled, transitions[2].TransitionSHA256)
	cancelled.Release = frozenArtifacts()
	cancelled.Checkpoint.Height = cancelled.Release.UpgradeHeight
	cancelled.Checkpoint.BlockIDSHA256 = testSHA("block-h")
	cancelled.Checkpoint.AppHashSHA256 = testSHA("app-h")
	cancelled.Approvals = []Approval{
		signApproval(
			t,
			cancelled,
			governanceCoordinatorRole,
			"did:zrn:governance-coordinator",
			2,
			"0",
		),
		signApproval(
			t,
			cancelled,
			releaseVerifierRole,
			"did:zrn:release-verifier",
			5,
			"0",
		),
	}
	sort.Slice(cancelled.Approvals, func(i, j int) bool {
		return approvalLess(cancelled.Approvals[i], cancelled.Approvals[j])
	})
	cancelled = sealWithoutEdgeValidationWithApprovals(t, cancelled)
	documents = append(documents, canonicalDocument(t, cancelled))

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "checkpoint height below upgrade height") {
		t.Fatalf("expected post-H cancellation rejection, got %v", err)
	}
}

func TestSealSharedIncidentRecoveryFailedEdge(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	edges := [][2]State{
		{StateContaining, StateRecoveryDesign},
		{StateRecoveryDesign, StateRecoveryReady},
		{StateRecoveryReady, StateActivating},
		{StateActivating, StateObserving},
		{StateObserving, StateRecoveryFailed},
	}
	previous := transitions[len(transitions)-1]
	for _, edge := range edges {
		next := incidentTransition(previous.Sequence+1, edge[0], edge[1], previous.TransitionSHA256)
		next = mustSeal(t, next)
		transitions = append(transitions, next)
		documents = append(documents, canonicalDocument(t, next))
		previous = next
	}

	result, err := verifyDocuments(
		documents,
		testVerifyOptionsForTransitions(t, transitions),
	)
	if err != nil {
		t.Fatalf("shared incident OBSERVING -> RECOVERY_FAILED edge must seal and verify: %v", err)
	}
	if result.State != StateRecoveryFailed {
		t.Fatalf("unexpected final state %s", result.State)
	}
}

func TestRejectNonExactSHA256(t *testing.T) {
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition.Evidence[0].SHA256 = strings.ToUpper(transition.Evidence[0].SHA256)

	err := validateTransitionBody(transition)
	if err == nil || !strings.Contains(err.Error(), "lowercase hexadecimal") {
		t.Fatalf("expected exact lowercase SHA-256 rejection, got %v", err)
	}
}

func TestRejectTrustPolicyIdentityAcrossSeparatedRoles(t *testing.T) {
	policy := testTrustPolicy("ZR-2026-0001", "")
	separatedPolicy := ApprovalPolicy{
		MinimumApprovals:          2,
		MinimumDistinctIdentities: 1,
		RequiredRoles:             []string{"release-author", "release-reviewer"},
		SeparatedRolePairs: []RolePair{{
			RoleA: "release-author",
			RoleB: "release-reviewer",
		}},
		PowerQuorum: PowerQuorum{},
	}
	for i := range policy.ApprovalPolicies {
		if policy.ApprovalPolicies[i].From == StateRunning &&
			policy.ApprovalPolicies[i].To == StateAssessing {
			policy.ApprovalPolicies[i].ApprovalPolicy = separatedPolicy
		}
	}
	policy.Signers = append(policy.Signers,
		TrustedSigner{
			Role:      "release-author",
			Identity:  "did:zrn:same-person",
			PublicKey: testPublicKey(12),
		},
		TrustedSigner{
			Role:      "release-reviewer",
			Identity:  "did:zrn:same-person",
			PublicKey: testPublicKey(12),
		},
	)
	sort.Slice(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	})

	err := validateTrustPolicy(policy)
	if err == nil || !strings.Contains(err.Error(), "fills separated roles") {
		t.Fatalf("expected trust-policy separated-role rejection, got %v", err)
	}
}

func TestExternallyPinnedPowerQuorum(t *testing.T) {
	policy := testTrustPolicy("ZR-2026-0001", "")
	firstEdgePolicy := ApprovalPolicy{
		MinimumApprovals:          1,
		MinimumDistinctIdentities: 1,
		RequiredRoles:             []string{"validator-operator"},
		SeparatedRolePairs:        []RolePair{},
		PowerQuorum:               testPowerQuorum("10"),
	}
	for i := range policy.ApprovalPolicies {
		if policy.ApprovalPolicies[i].From == StateRunning &&
			policy.ApprovalPolicies[i].To == StateAssessing {
			policy.ApprovalPolicies[i].ApprovalPolicy = firstEdgePolicy
		}
	}
	policySHA256 := mustTestPolicySHA256(t, policy)
	if err := validateTrustPolicy(policy); err != nil {
		t.Fatalf("validate test trust policy: %v", err)
	}
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition.TrustPolicySHA256 = policySHA256
	setTestPowerSnapshot(&transition, "6", "4")
	transition.Approvals = []Approval{
		signApproval(t, transition, "validator-operator", "did:zrn:validator-a", 9, "6"),
	}
	if err := validateTransitionBody(transition); err != nil {
		t.Fatalf("validate signed transition body: %v", err)
	}
	if err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	); err == nil || !strings.Contains(err.Error(), "power quorum not met") {
		t.Fatalf("expected insufficient-power rejection, got %v", err)
	}
	transition.Approvals = append(
		transition.Approvals,
		signApproval(t, transition, "validator-operator", "did:zrn:validator-b", 10, "4"),
	)
	sort.Slice(transition.Approvals, func(i, j int) bool {
		return approvalLess(transition.Approvals[i], transition.Approvals[j])
	})
	if err := validateTransitionBody(transition); err != nil {
		t.Fatalf("validate quorum transition body: %v", err)
	}
	if err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	); err != nil {
		t.Fatalf("expected 10/10 to meet strict 2/3 externally pinned quorum: %v", err)
	}
}

func TestWeakEarlyAuthorityCannotAuthorizeStagingOrActivation(t *testing.T) {
	tests := []struct {
		name       string
		transition Transition
	}{
		{
			name:       "release staging",
			transition: releaseTransition(4, StateScheduled, StateStaged, testSHA("previous")),
		},
		{
			name:       "incident activation",
			transition: incidentTransition(6, StateRecoveryReady, StateActivating, testSHA("previous")),
		},
	}
	tests[0].transition.Release = frozenArtifacts()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := testPolicyForTransition(test.transition)
			policySHA256 := mustTestPolicySHA256(t, policy)
			test.transition.TrustPolicySHA256 = policySHA256
			addTestPowerSnapshotEvidence(&test.transition)
			test.transition.Approvals = []Approval{
				signApproval(
					t,
					test.transition,
					"operations-approver",
					"did:zrn:approver-1",
					1,
					"0",
				),
			}
			err := validateTransitionAgainstTrustPolicy(
				test.transition,
				policy,
				policySHA256,
				test.transition.PowerSnapshot.SnapshotSHA256,
			)
			if err == nil {
				t.Fatalf("weak early authority authorized high-risk edge: %v", err)
			}
		})
	}
}

func TestHighRiskEdgesRequireCanonicalValidatorOperatorRole(t *testing.T) {
	policy := testTrustPolicy("", "release-2026-07")
	policy.Signers = append(policy.Signers, TrustedSigner{
		Role:      "fake-power",
		Identity:  "did:zrn:fake-power",
		PublicKey: testPublicKey(12),
	})
	for i := range policy.ApprovalPolicies {
		edge := &policy.ApprovalPolicies[i]
		if edge.Lane == laneRelease &&
			edge.From == StateScheduled &&
			edge.To == StateStaged {
			edge.ApprovalPolicy.RequiredRoles = []string{
				"fake-power",
				releaseVerifierRole,
			}
			edge.ApprovalPolicy.PowerQuorum.Role = "fake-power"
		}
	}
	sort.Slice(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	})

	err := validateTrustPolicy(policy)
	if err == nil || !strings.Contains(err.Error(), validatorOperatorRole) {
		t.Fatalf("self-chosen power role satisfied high-risk guard: %v", err)
	}
}

func TestPowerSnapshotIsFreshTransitionStateUnderStableTrustRoot(t *testing.T) {
	preparing := releaseTransition(1, StateRunning, StatePreparing, "")
	policy := testPolicyForTransition(preparing)
	policySHA256 := mustTestPolicySHA256(t, policy)
	preparing.TrustPolicySHA256 = policySHA256
	preparing.Approvals = []Approval{
		signApproval(t, preparing, "operations-approver", "did:zrn:approver-1", 1, "0"),
	}
	if err := validateTransitionAgainstTrustPolicy(preparing, policy, policySHA256, ""); err != nil {
		t.Fatalf("preparing under stable trust root: %v", err)
	}

	staged := releaseTransition(4, StateScheduled, StateStaged, testSHA("previous"))
	staged.Release = frozenArtifacts()
	staged.Checkpoint.Height = 99
	staged.Checkpoint.BlockIDSHA256 = testSHA("block-99")
	staged.Checkpoint.AppHashSHA256 = testSHA("app-99")
	staged.TrustPolicySHA256 = policySHA256
	setTestPowerSnapshot(&staged, "7", "3")
	staged.Approvals = []Approval{
		signApproval(t, staged, releaseVerifierRole, "did:zrn:release-verifier", 5, "0"),
		signApproval(t, staged, validatorOperatorRole, "did:zrn:validator-a", 9, "7"),
	}
	sort.Slice(staged.Approvals, func(i, j int) bool {
		return approvalLess(staged.Approvals[i], staged.Approvals[j])
	})
	if err := validateTransitionAgainstTrustPolicy(
		staged,
		policy,
		policySHA256,
		staged.PowerSnapshot.SnapshotSHA256,
	); err != nil {
		t.Fatalf("fresh checkpoint snapshot failed under unchanged trust root: %v", err)
	}
	if preparing.TrustPolicySHA256 != staged.TrustPolicySHA256 {
		t.Fatal("dynamic power snapshot unexpectedly rotated the stable trust root")
	}
}

func TestJournalVerificationRequiresExternallyPinnedPowerSnapshot(t *testing.T) {
	policy := testTrustPolicy("ZR-2026-0001", "")
	for i := range policy.ApprovalPolicies {
		edge := &policy.ApprovalPolicies[i]
		if edge.Lane == laneIncident &&
			edge.From == StateRunning &&
			edge.To == StateAssessing {
			edge.ApprovalPolicy = ApprovalPolicy{
				MinimumApprovals:          1,
				MinimumDistinctIdentities: 1,
				RequiredRoles:             []string{validatorOperatorRole},
				SeparatedRolePairs:        []RolePair{},
				PowerQuorum:               testPowerQuorum("10"),
			}
		}
	}
	policySHA256 := mustTestPolicySHA256(t, policy)
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition.TrustPolicySHA256 = policySHA256
	addTestPowerSnapshotEvidence(&transition)
	transition.Approvals = []Approval{
		signApproval(t, transition, validatorOperatorRole, "did:zrn:validator-a", 9, "7"),
	}
	sealed, err := sealTransition(transition)
	if err != nil {
		t.Fatal(err)
	}
	document := canonicalDocument(t, sealed)
	options := VerifyOptions{
		TrustPolicy:       &policy,
		TrustPolicySHA256: policySHA256,
	}

	if _, err := verifyDocuments([][]byte{document}, options); err == nil ||
		!strings.Contains(err.Error(), "externally obtained snapshot digest") {
		t.Fatalf("self-asserted validator power snapshot passed verification: %v", err)
	}
	options.ExpectedPowerSnapshotSHA256 = map[uint64]string{
		1: transition.PowerSnapshot.SnapshotSHA256,
	}
	if _, err := verifyDocuments([][]byte{document}, options); err != nil {
		t.Fatalf("independently pinned validator snapshot failed verification: %v", err)
	}
}

func TestPowerSnapshotMustBindExactCheckpoint(t *testing.T) {
	transition := releaseTransition(4, StateScheduled, StateStaged, testSHA("previous"))
	transition.Release = frozenArtifacts()
	policy := testPolicyForTransition(transition)
	policySHA256 := mustTestPolicySHA256(t, policy)
	transition.TrustPolicySHA256 = policySHA256
	addTestPowerSnapshotEvidence(&transition)
	transition.PowerSnapshot.Height--
	transition.PowerSnapshot.SnapshotSHA256, _ = powerSnapshotHash(transition.PowerSnapshot)
	for i := range transition.Evidence {
		if transition.Evidence[i].Type == "validator-power-snapshot" {
			transition.Evidence[i].SHA256 = transition.PowerSnapshot.SnapshotSHA256
		}
	}
	transition.Approvals = []Approval{
		signApproval(t, transition, validatorOperatorRole, "did:zrn:validator-a", 9, "7"),
	}

	err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	)
	if err == nil || !strings.Contains(err.Error(), "exact transition checkpoint") {
		t.Fatalf("older power snapshot checkpoint authorized transition: %v", err)
	}
}

func TestWeakEarlyAuthorityCannotAuthorizeAnyForkChoiceEdge(t *testing.T) {
	policy := testTrustPolicy("ZR-2026-0001", "")
	forkRoles := append(
		[]string{"fork-authority"},
		mandatoryRolesForEdge(EdgeApprovalPolicy{
			Lane: laneIncident,
			To:   StateForkChoice,
		})...,
	)
	sort.Strings(forkRoles)
	forkPolicy := ApprovalPolicy{
		MinimumApprovals:          uint64(len(forkRoles)),
		MinimumDistinctIdentities: uint64(len(forkRoles)),
		RequiredRoles:             forkRoles,
		SeparatedRolePairs: mandatorySeparatedRolePairsForEdge(
			EdgeApprovalPolicy{Lane: laneIncident, To: StateForkChoice},
		),
		PowerQuorum: testPowerQuorum("10"),
	}
	policy.Signers = append(policy.Signers, TrustedSigner{
		Role:      "fork-authority",
		Identity:  "did:zrn:fork-council",
		PublicKey: testPublicKey(11),
	})
	var forkSources []State
	for i := range policy.ApprovalPolicies {
		edge := &policy.ApprovalPolicies[i]
		if edge.Lane == laneIncident && edge.To == StateForkChoice {
			edge.ApprovalPolicy = forkPolicy
			forkSources = append(forkSources, edge.From)
		}
	}
	sort.Slice(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	})
	if err := validateTrustPolicy(policy); err != nil {
		t.Fatalf("validate fork-choice policy: %v", err)
	}
	policySHA256 := mustTestPolicySHA256(t, policy)

	for _, from := range forkSources {
		transition := incidentTransition(9, from, StateForkChoice, testSHA("previous-"+string(from)))
		transition.TrustPolicySHA256 = policySHA256
		addTestPowerSnapshotEvidence(&transition)
		transition.Approvals = []Approval{
			signApproval(t, transition, "operations-approver", "did:zrn:approver-1", 1, "0"),
		}
		err := validateTransitionAgainstTrustPolicy(
			transition,
			policy,
			policySHA256,
			transition.PowerSnapshot.SnapshotSHA256,
		)
		if err == nil {
			t.Fatalf("weak authority authorized %s -> FORK_CHOICE: %v", from, err)
		}
	}
}

func TestTrustPolicyRequiresExactSortedEdgeCoverage(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		policy := testTrustPolicy("ZR-2026-0001", "")
		policy.ApprovalPolicies = policy.ApprovalPolicies[1:]
		err := validateTrustPolicy(policy)
		if err == nil || !strings.Contains(err.Error(), "missing required edge") {
			t.Fatalf("expected missing-edge rejection, got %v", err)
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		policy := testTrustPolicy("ZR-2026-0001", "")
		policy.ApprovalPolicies = append(
			policy.ApprovalPolicies,
			policy.ApprovalPolicies[len(policy.ApprovalPolicies)-1],
		)
		err := validateTrustPolicy(policy)
		if err == nil || !strings.Contains(err.Error(), "duplicates edge") {
			t.Fatalf("expected duplicate-edge rejection, got %v", err)
		}
	})
	t.Run("unordered", func(t *testing.T) {
		policy := testTrustPolicy("ZR-2026-0001", "")
		policy.ApprovalPolicies[0], policy.ApprovalPolicies[1] =
			policy.ApprovalPolicies[1], policy.ApprovalPolicies[0]
		err := validateTrustPolicy(policy)
		if err == nil || !strings.Contains(err.Error(), "must be sorted") {
			t.Fatalf("expected unordered-edge rejection, got %v", err)
		}
	})
}

func TestTransitionRequiresSemanticEvidenceForState(t *testing.T) {
	transition := releaseTransition(2, StatePreparing, StateReleaseFrozen, testSHA("previous"))
	transition.Release = frozenArtifacts()
	transition.Evidence = []Evidence{{
		Type:   "intent-record",
		SHA256: testSHA("irrelevant-preparation-evidence"),
		URI:    "vault://zerone/test",
	}}
	err := validateTransitionBody(transition)
	if err == nil || !strings.Contains(err.Error(), "release-manifest") {
		t.Fatalf("irrelevant evidence satisfied RELEASE_FROZEN: %v", err)
	}
}

func TestFrozenReleaseEnforcesActivationProfileArtifacts(t *testing.T) {
	validate := func(transition Transition) error {
		if err := validateTransitionBody(transition); err != nil {
			return err
		}
		return validateStandaloneEdge(transition)
	}
	cosmovisor := releaseTransition(2, StatePreparing, StateReleaseFrozen, testSHA("previous"))
	cosmovisor.Release = frozenArtifacts()
	if err := validate(cosmovisor); err != nil {
		t.Fatalf("cosmovisor profile must not require a container image digest: %v", err)
	}

	immutable := cosmovisor
	immutable.Release.ActivationMode = activationModeImmutableImage
	if err := validate(immutable); err == nil ||
		!strings.Contains(err.Error(), "release.image_sha256") {
		t.Fatalf("immutable-image profile accepted without image digest: %v", err)
	}
	immutable.Release.ImageSHA256 = testSHA("immutable-validator-image")
	if err := validate(immutable); err != nil {
		t.Fatalf("immutable-image profile rejected with pinned image digest: %v", err)
	}
}

func TestPowerSnapshotMustBeFreshAtPowerGatedEdge(t *testing.T) {
	transition := releaseTransition(4, StateScheduled, StateStaged, testSHA("previous"))
	transition.Release = frozenArtifacts()
	policy := testPolicyForTransition(transition)
	if err := validateTrustPolicy(policy); err != nil {
		t.Fatalf("validate test policy: %v", err)
	}
	policySHA256 := mustTestPolicySHA256(t, policy)
	transition.TrustPolicySHA256 = policySHA256
	addTestPowerSnapshotEvidence(&transition)
	transition.PowerSnapshot.ValidUntil = "2026-07-29T21:30:00Z"
	transition.PowerSnapshot.SnapshotSHA256, _ = powerSnapshotHash(transition.PowerSnapshot)
	for i := range transition.Evidence {
		if transition.Evidence[i].Type == "validator-power-snapshot" {
			transition.Evidence[i].SHA256 = transition.PowerSnapshot.SnapshotSHA256
		}
	}
	transition.Approvals = []Approval{
		signApproval(t, transition, "validator-operator", "did:zrn:validator-a", 9, "7"),
	}
	err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	)
	if err == nil || !strings.Contains(err.Error(), "power snapshot expired") {
		t.Fatalf("stale power snapshot authorized STAGED: %v", err)
	}
}

func TestStrictPowerQuorumRejectsExactTwoThirds(t *testing.T) {
	policy := testTrustPolicy("ZR-2026-0001", "")
	for i := range policy.ApprovalPolicies {
		if policy.ApprovalPolicies[i].From == StateRunning &&
			policy.ApprovalPolicies[i].To == StateAssessing {
			policy.ApprovalPolicies[i].ApprovalPolicy = ApprovalPolicy{
				MinimumApprovals:          1,
				MinimumDistinctIdentities: 1,
				RequiredRoles:             []string{"validator-operator"},
				SeparatedRolePairs:        []RolePair{},
				PowerQuorum:               testPowerQuorum("3"),
			}
		}
	}
	if err := validateTrustPolicy(policy); err != nil {
		t.Fatalf("validate strict policy: %v", err)
	}
	policySHA256 := mustTestPolicySHA256(t, policy)
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition.TrustPolicySHA256 = policySHA256
	setTestPowerSnapshot(&transition, "2", "1")
	transition.Approvals = []Approval{
		signApproval(t, transition, "validator-operator", "did:zrn:validator-a", 9, "2"),
	}
	err := validateTransitionAgainstTrustPolicy(
		transition,
		policy,
		policySHA256,
		transition.PowerSnapshot.SnapshotSHA256,
	)
	if err == nil || !strings.Contains(err.Error(), "power quorum not met") {
		t.Fatalf("exact 2/3 must fail strict readiness quorum, got %v", err)
	}
}

func TestRejectUnsignedSelfHashedTransition(t *testing.T) {
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition = sealWithoutEdgeValidation(t, transition)
	transition.Approvals = []Approval{}
	transition.TransitionSHA256 = ""
	hash, err := transitionHash(transition)
	if err != nil {
		t.Fatal(err)
	}
	transition.TransitionSHA256 = hash
	options := testVerifyOptions(t, transition)

	_, err = verifyDocuments(
		[][]byte{canonicalDocument(t, transition)},
		options,
	)
	if err == nil || !strings.Contains(err.Error(), "approval quorum not met") {
		t.Fatalf("expected unsigned transition rejection, got %v", err)
	}
}

func TestRejectJournalDeclaredPolicyReplacement(t *testing.T) {
	transitions, documents := validIncidentJournal(t)
	hostilePolicy := testTrustPolicy("ZR-2026-0001", "")
	hostilePolicy.PolicyID = "attacker-policy"
	hostilePolicy.Signers[0].PublicKey = testPublicKey(99)
	hostileSHA256 := mustTestPolicySHA256(t, hostilePolicy)

	hostile := incidentTransition(3, StateContaining, StateSafetyStopped, transitions[1].TransitionSHA256)
	hostile.TrustPolicySHA256 = hostileSHA256
	hostile.Approvals = []Approval{
		signApproval(t, hostile, "operations-approver", "did:zrn:approver-1", 99, "0"),
	}
	hostile = sealWithoutEdgeValidationWithApprovals(t, hostile)
	documents = append(documents, canonicalDocument(t, hostile))

	_, err := verifyDocuments(documents, testVerifyOptions(t, transitions[0]))
	if err == nil || !strings.Contains(err.Error(), "externally pinned policy") {
		t.Fatalf("expected hostile policy replacement rejection, got %v", err)
	}
}

func TestSelfHashUsesEmptyTransitionHash(t *testing.T) {
	transition := mustSeal(t, incidentTransition(1, StateRunning, StateAssessing, ""))
	original := transition.TransitionSHA256
	transition.TransitionSHA256 = testSHA("arbitrary-existing-value")
	recomputed, err := transitionHash(transition)
	if err != nil {
		t.Fatal(err)
	}
	if recomputed != original {
		t.Fatalf("self hash depended on existing transition_sha256: got %s want %s", recomputed, original)
	}
}

func sealWithoutEdgeValidationWithApprovals(t *testing.T, transition Transition) Transition {
	t.Helper()
	transition.TransitionSHA256 = ""
	hash, err := transitionHash(transition)
	if err != nil {
		t.Fatalf("hash transition: %v", err)
	}
	transition.TransitionSHA256 = hash
	return transition
}

func signApproval(
	t *testing.T,
	transition Transition,
	role string,
	identity string,
	seedByte byte,
	power string,
) Approval {
	t.Helper()
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	approval := Approval{
		Role:      role,
		Identity:  identity,
		PublicKey: hex.EncodeToString(publicKey),
		Power:     power,
	}
	digest, err := approvalStatementDigest(transition, approval)
	if err != nil {
		t.Fatalf("approval statement: %v", err)
	}
	approval.StatementSHA256 = hex.EncodeToString(digest[:])
	approval.Signature = hex.EncodeToString(ed25519.Sign(privateKey, digest[:]))
	return approval
}
