package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

func testSupersession(
	t *testing.T,
	oldPolicy TrustPolicy,
	newPolicy TrustPolicy,
	oldHead string,
) Supersession {
	t.Helper()
	oldPolicySHA256 := mustTestPolicySHA256(t, oldPolicy)
	newPolicySHA256 := mustTestPolicySHA256(t, newPolicy)
	evidence := []Evidence{
		{
			Type:   "trust-policy-compromise-assessment",
			SHA256: testSHA("compromise-assessment"),
			URI:    "vault://zerone/compromise-assessment",
		},
		{
			Type:   "replacement-policy-ceremony",
			SHA256: testSHA("replacement-policy-ceremony"),
			URI:    "vault://zerone/replacement-policy-ceremony",
		},
	}
	sort.Slice(evidence, func(i, j int) bool {
		return evidenceLess(evidence[i], evidence[j])
	})
	return Supersession{
		Schema:                supersessionSchema,
		ChainID:               oldPolicy.ChainID,
		OldJournalHeadSHA256:  oldHead,
		OldTrustPolicySHA256:  oldPolicySHA256,
		NewTrustPolicySHA256:  newPolicySHA256,
		ReplacementIncidentID: newPolicy.IncidentID,
		ReplacementReleaseID:  newPolicy.ReleaseID,
		OccurredAt:            "2026-07-29T23:00:00Z",
		Reason:                "offline policy-rotation key material was exposed during an active incident",
		Evidence:              evidence,
		Approvals:             []Approval{},
		SupersessionSHA256:    "",
	}
}

func signSupersessionApproval(
	t *testing.T,
	supersession Supersession,
	role, identity string,
	seedByte byte,
) Approval {
	t.Helper()
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	approval := Approval{
		Role:      role,
		Identity:  identity,
		PublicKey: hex.EncodeToString(privateKey.Public().(ed25519.PublicKey)),
		Power:     "0",
	}
	digest, err := supersessionApprovalStatementDigest(supersession, approval)
	if err != nil {
		t.Fatalf("supersession approval statement: %v", err)
	}
	approval.StatementSHA256 = hex.EncodeToString(digest[:])
	approval.Signature = hex.EncodeToString(ed25519.Sign(privateKey, digest[:]))
	return approval
}

func signTestSupersession(t *testing.T, supersession Supersession) Supersession {
	t.Helper()
	supersession.Approvals = []Approval{
		signSupersessionApproval(
			t,
			supersession,
			evidenceCustodianRole,
			"did:zrn:evidence-custodian",
			7,
		),
		signSupersessionApproval(
			t,
			supersession,
			policyRotationAuthorityRole,
			"did:zrn:policy-rotation-authority",
			8,
		),
	}
	sort.Slice(supersession.Approvals, func(i, j int) bool {
		return approvalLess(supersession.Approvals[i], supersession.Approvals[j])
	})
	return supersession
}

func TestSupersessionSidecarRotatesPolicyBeforeJournalTerminalState(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "")

	// This is the head of a journal still at ASSESSING. A compromised policy
	// must not force operators to fabricate progress to CLOSED.
	oldTransition := mustSeal(
		t,
		incidentTransition(1, StateRunning, StateAssessing, ""),
	)
	supersession := signTestSupersession(
		t,
		testSupersession(t, oldPolicy, newPolicy, oldTransition.TransitionSHA256),
	)
	sealed, err := sealSupersession(
		supersession,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		oldTransition.TransitionSHA256,
	)
	if err != nil {
		t.Fatalf("seal supersession: %v", err)
	}
	document, err := json.Marshal(sealed)
	if err != nil {
		t.Fatalf("marshal sealed supersession: %v", err)
	}
	result, err := verifySupersession(
		document,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		oldTransition.TransitionSHA256,
	)
	if err != nil {
		t.Fatalf("verify supersession: %v", err)
	}
	if result.ReplacementIncidentID != "ZR-2026-0002" {
		t.Fatalf("unexpected replacement incident: %s", result.ReplacementIncidentID)
	}
}

func TestSupersessionRequiresExternalOldHeadPin(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "")
	supersession := signTestSupersession(
		t,
		testSupersession(t, oldPolicy, newPolicy, testSHA("old-head")),
	)
	if _, err := sealSupersession(
		supersession,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		testSHA("attacker-selected-head"),
	); err == nil || !strings.Contains(err.Error(), "old journal head mismatch") {
		t.Fatalf("untrusted old head passed supersession: %v", err)
	}
}

func TestSupersessionRequiresExactlyOneReplacementLane(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "release-2026-08")
	supersession := signTestSupersession(
		t,
		testSupersession(t, oldPolicy, newPolicy, testSHA("old-head")),
	)
	if _, err := sealSupersession(
		supersession,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		supersession.OldJournalHeadSHA256,
	); err == nil ||
		!strings.Contains(err.Error(), "exactly one replacement") {
		t.Fatalf("two replacement lanes passed supersession: %v", err)
	}
}

func TestSupersessionRequiresExactlyOneOfEachCeremonyEvidence(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "")
	supersession := testSupersession(
		t,
		oldPolicy,
		newPolicy,
		testSHA("old-head"),
	)
	supersession.Evidence = append(supersession.Evidence, Evidence{
		Type:   "trust-policy-compromise-assessment",
		SHA256: testSHA("second-assessment"),
		URI:    "vault://zerone/second-compromise-assessment",
	})
	sort.Slice(supersession.Evidence, func(i, j int) bool {
		return evidenceLess(supersession.Evidence[i], supersession.Evidence[j])
	})
	supersession = signTestSupersession(t, supersession)
	if _, err := sealSupersession(
		supersession,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		supersession.OldJournalHeadSHA256,
	); err == nil ||
		!strings.Contains(err.Error(), "exactly one evidence item") {
		t.Fatalf("ambiguous compromise evidence passed supersession: %v", err)
	}
}

func TestSupersessionRequiresPreprovisionedIndependentRoles(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	filtered := oldPolicy.Signers[:0]
	for _, signer := range oldPolicy.Signers {
		if signer.Role != evidenceCustodianRole &&
			signer.Role != policyRotationAuthorityRole {
			filtered = append(filtered, signer)
		}
	}
	oldPolicy.Signers = filtered
	newPolicy := testTrustPolicy("ZR-2026-0002", "")
	supersession := testSupersession(t, oldPolicy, newPolicy, testSHA("old-head"))
	supersession = signTestSupersession(t, supersession)
	if _, err := sealSupersession(
		supersession,
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		supersession.OldJournalHeadSHA256,
	); err == nil || !strings.Contains(err.Error(), "requires preprovisioned offline role") {
		t.Fatalf("policy without independent rotation roles superseded a journal: %v", err)
	}
}

func TestTransitionV1SealedFixtureRemainsByteCompatible(t *testing.T) {
	document, err := os.ReadFile("testdata/transition-v1-sealed.json")
	if err != nil {
		t.Fatalf("read frozen v1 fixture: %v", err)
	}
	document = bytes.TrimSuffix(document, []byte("\n"))
	transition, err := decodeTransition(document, true)
	if err != nil {
		t.Fatalf("decode frozen v1 fixture: %v", err)
	}
	if err := validateSealedTransition(transition); err != nil {
		t.Fatalf("verify frozen v1 fixture: %v", err)
	}
}
