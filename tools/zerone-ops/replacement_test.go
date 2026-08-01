package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type replacementFixture struct {
	oldPolicy       TrustPolicy
	newPolicy       TrustPolicy
	oldTransition   Transition
	supersession    Supersession
	oldDocuments    [][]byte
	newDocuments    [][]byte
	sidecarDocument []byte
	oldOptions      VerifyOptions
	newOptions      VerifyOptions
}

func testReplacementFixture(t *testing.T) replacementFixture {
	t.Helper()
	return testReplacementFixtureAt(t, "2026-07-29T23:00:00Z")
}

func testReplacementFixtureAt(
	t *testing.T,
	supersessionOccurredAt string,
) replacementFixture {
	t.Helper()
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "")
	oldTransition := mustSeal(
		t,
		incidentTransition(1, StateRunning, StateAssessing, ""),
	)
	supersession := signTestSupersession(
		t,
		func() Supersession {
			draft := testSupersession(
				t,
				oldPolicy,
				newPolicy,
				oldTransition.TransitionSHA256,
			)
			draft.OccurredAt = supersessionOccurredAt
			return draft
		}(),
	)
	sealedSupersession, err := sealSupersession(
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
	sidecarDocument, err := json.Marshal(sealedSupersession)
	if err != nil {
		t.Fatalf("marshal supersession: %v", err)
	}
	newTransition := testReplacementTransition(
		t,
		newPolicy,
		sealedSupersession,
		"2026-07-29T23:01:00Z",
		sealedSupersession.SupersessionSHA256,
		sealedSupersession.OldJournalHeadSHA256,
	)
	return replacementFixture{
		oldPolicy:       oldPolicy,
		newPolicy:       newPolicy,
		oldTransition:   oldTransition,
		supersession:    sealedSupersession,
		oldDocuments:    [][]byte{canonicalDocument(t, oldTransition)},
		newDocuments:    [][]byte{canonicalDocument(t, newTransition)},
		sidecarDocument: sidecarDocument,
		oldOptions: VerifyOptions{
			ExpectedHeadSHA256: oldTransition.TransitionSHA256,
			TrustPolicy:        &oldPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, oldPolicy),
		},
		newOptions: VerifyOptions{
			ExpectedHeadSHA256: newTransition.TransitionSHA256,
			TrustPolicy:        &newPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, newPolicy),
		},
	}
}

func testReplacementTransition(
	t *testing.T,
	newPolicy TrustPolicy,
	supersession Supersession,
	occurredAt string,
	sidecarSHA256 string,
	oldHeadSHA256 string,
	mutators ...func(*Transition),
) Transition {
	t.Helper()
	transition := incidentTransition(1, StateRunning, StateAssessing, "")
	transition.IncidentID = newPolicy.IncidentID
	transition.ReleaseID = ""
	transition.OccurredAt = occurredAt
	transition.TrustPolicySHA256 = mustTestPolicySHA256(t, newPolicy)
	if sidecarSHA256 != "" {
		transition.Evidence = append(transition.Evidence, Evidence{
			Type:   supersessionSidecarEvidence,
			SHA256: sidecarSHA256,
			URI:    "vault://zerone/supersession-sidecar",
		})
	}
	if oldHeadSHA256 != "" {
		transition.Evidence = append(transition.Evidence, Evidence{
			Type:   supersededHeadEvidence,
			SHA256: oldHeadSHA256,
			URI:    "vault://zerone/superseded-journal-head",
		})
	}
	for _, mutate := range mutators {
		mutate(&transition)
	}
	sort.Slice(transition.Evidence, func(i, j int) bool {
		return evidenceLess(transition.Evidence[i], transition.Evidence[j])
	})
	return mustSeal(t, transition)
}

func TestVerifyReplacementComposesPinnedHistory(t *testing.T) {
	fixture := testReplacementFixture(t)
	result, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err != nil {
		t.Fatalf("verify composed replacement: %v", err)
	}
	if result.OldJournal.HeadSHA256 != fixture.oldTransition.TransitionSHA256 ||
		result.Supersession.SupersessionSHA256 != fixture.supersession.SupersessionSHA256 ||
		result.NewJournal.IncidentID != "ZR-2026-0002" {
		t.Fatalf("unexpected replacement result: %+v", result)
	}
}

func TestVerifyReplacementRejectsTruncatedOldJournalPin(t *testing.T) {
	fixture := testReplacementFixture(t)
	fixture.oldOptions.ExpectedHeadSHA256 = testSHA("attacker-prefix")
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil || !strings.Contains(err.Error(), "head SHA-256 mismatch") {
		t.Fatalf("wrong old head pin passed composed verification: %v", err)
	}
}

func TestVerifyReplacementRejectsRealOldJournalPrefix(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0002", "")
	first := mustSeal(
		t,
		incidentTransition(1, StateRunning, StateAssessing, ""),
	)
	secondDraft := incidentTransition(
		2,
		StateAssessing,
		StateContaining,
		first.TransitionSHA256,
	)
	secondDraft.OccurredAt = "2026-07-29T22:10:00Z"
	second := mustSeal(t, secondDraft)
	supersessionDraft := testSupersession(
		t,
		oldPolicy,
		newPolicy,
		second.TransitionSHA256,
	)
	sealedSupersession, err := sealSupersession(
		signTestSupersession(t, supersessionDraft),
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		second.TransitionSHA256,
	)
	if err != nil {
		t.Fatalf("seal supersession: %v", err)
	}
	sidecarDocument, err := json.Marshal(sealedSupersession)
	if err != nil {
		t.Fatal(err)
	}
	newTransition := testReplacementTransition(
		t,
		newPolicy,
		sealedSupersession,
		"2026-07-29T23:01:00Z",
		sealedSupersession.SupersessionSHA256,
		second.TransitionSHA256,
	)

	_, err = verifyReplacement(
		[][]byte{canonicalDocument(t, first)},
		sidecarDocument,
		[][]byte{canonicalDocument(t, newTransition)},
		VerifyOptions{
			ExpectedHeadSHA256: second.TransitionSHA256,
			TrustPolicy:        &oldPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, oldPolicy),
		},
		VerifyOptions{
			ExpectedHeadSHA256: newTransition.TransitionSHA256,
			TrustPolicy:        &newPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, newPolicy),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "head SHA-256 mismatch") {
		t.Fatalf("real old-journal prefix passed verification: %v", err)
	}
}

func TestVerifyReplacementRejectsUnpinnedNewHead(t *testing.T) {
	fixture := testReplacementFixture(t)
	fixture.newOptions.ExpectedHeadSHA256 = ""
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "replacement journal head is required") {
		t.Fatalf("missing new head pin passed composed verification: %v", err)
	}
}

func TestVerifyReplacementRequiresBindingsOnFirstNewTransition(t *testing.T) {
	fixture := testReplacementFixture(t)
	newTransition := testReplacementTransition(
		t,
		fixture.newPolicy,
		fixture.supersession,
		"2026-07-29T23:01:00Z",
		"",
		"",
	)
	fixture.newDocuments = [][]byte{canonicalDocument(t, newTransition)}
	fixture.newOptions.ExpectedHeadSHA256 = newTransition.TransitionSHA256
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"first replacement transition requires exactly one",
	) {
		t.Fatalf("unbound replacement journal passed verification: %v", err)
	}
}

func TestVerifyReplacementRejectsWrongSidecarEvidenceDigest(t *testing.T) {
	fixture := testReplacementFixture(t)
	newTransition := testReplacementTransition(
		t,
		fixture.newPolicy,
		fixture.supersession,
		"2026-07-29T23:01:00Z",
		testSHA("different-sidecar"),
		fixture.supersession.OldJournalHeadSHA256,
	)
	fixture.newDocuments = [][]byte{canonicalDocument(t, newTransition)}
	fixture.newOptions.ExpectedHeadSHA256 = newTransition.TransitionSHA256
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil || !strings.Contains(err.Error(), "expected") {
		t.Fatalf("wrong sidecar evidence digest passed verification: %v", err)
	}
}

func TestVerifyReplacementRejectsWrongOldHeadEvidenceDigest(t *testing.T) {
	fixture := testReplacementFixture(t)
	newTransition := testReplacementTransition(
		t,
		fixture.newPolicy,
		fixture.supersession,
		"2026-07-29T23:01:00Z",
		fixture.supersession.SupersessionSHA256,
		testSHA("wrong-old-head-evidence"),
	)
	fixture.newDocuments = [][]byte{canonicalDocument(t, newTransition)}
	fixture.newOptions.ExpectedHeadSHA256 = newTransition.TransitionSHA256
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil ||
		!strings.Contains(err.Error(), supersededHeadEvidence) {
		t.Fatalf("wrong old-head evidence digest passed verification: %v", err)
	}
}

func TestVerifyReplacementRejectsCheckpointRollback(t *testing.T) {
	fixture := testReplacementFixture(t)
	newTransition := testReplacementTransition(
		t,
		fixture.newPolicy,
		fixture.supersession,
		"2026-07-29T23:01:00Z",
		fixture.supersession.SupersessionSHA256,
		fixture.supersession.OldJournalHeadSHA256,
		func(transition *Transition) {
			transition.Checkpoint = Checkpoint{}
		},
	)
	fixture.newDocuments = [][]byte{canonicalDocument(t, newTransition)}
	fixture.newOptions.ExpectedHeadSHA256 = newTransition.TransitionSHA256
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "checkpoint height moved backward") {
		t.Fatalf("checkpoint rollback passed replacement verification: %v", err)
	}
}

func TestVerifyReplacementRejectsChronologyInversion(t *testing.T) {
	fixture := testReplacementFixture(t)
	newTransition := testReplacementTransition(
		t,
		fixture.newPolicy,
		fixture.supersession,
		"2026-07-29T22:30:00Z",
		fixture.supersession.SupersessionSHA256,
		fixture.supersession.OldJournalHeadSHA256,
	)
	fixture.newDocuments = [][]byte{canonicalDocument(t, newTransition)}
	fixture.newOptions.ExpectedHeadSHA256 = newTransition.TransitionSHA256
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil || !strings.Contains(err.Error(), "precedes supersession") {
		t.Fatalf("chronology inversion passed replacement verification: %v", err)
	}
}

func TestVerifyReplacementRejectsSupersessionBeforeOldHead(t *testing.T) {
	fixture := testReplacementFixtureAt(t, "2026-07-29T21:59:59Z")
	_, err := verifyReplacement(
		fixture.oldDocuments,
		fixture.sidecarDocument,
		fixture.newDocuments,
		fixture.oldOptions,
		fixture.newOptions,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "precedes old journal head") {
		t.Fatalf("sidecar-before-old chronology passed verification: %v", err)
	}
}

func TestVerifyReplacementRequiresNewIdentifierInSameLane(t *testing.T) {
	oldPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy := testTrustPolicy("ZR-2026-0001", "")
	newPolicy.PolicyID = "zerone-replacement-policy"
	oldTransition := mustSeal(
		t,
		incidentTransition(1, StateRunning, StateAssessing, ""),
	)
	supersessionDraft := testSupersession(
		t,
		oldPolicy,
		newPolicy,
		oldTransition.TransitionSHA256,
	)
	sealedSupersession, err := sealSupersession(
		signTestSupersession(t, supersessionDraft),
		oldPolicy,
		mustTestPolicySHA256(t, oldPolicy),
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		oldTransition.TransitionSHA256,
	)
	if err != nil {
		t.Fatalf("seal supersession: %v", err)
	}
	sidecarDocument, err := json.Marshal(sealedSupersession)
	if err != nil {
		t.Fatal(err)
	}
	newDraft := incidentTransition(1, StateRunning, StateAssessing, "")
	newDraft.OccurredAt = "2026-07-29T23:01:00Z"
	newDraft.TrustPolicySHA256 = mustTestPolicySHA256(t, newPolicy)
	newDraft.Evidence = append(
		newDraft.Evidence,
		Evidence{
			Type:   supersessionSidecarEvidence,
			SHA256: sealedSupersession.SupersessionSHA256,
			URI:    "vault://zerone/supersession-sidecar",
		},
		Evidence{
			Type:   supersededHeadEvidence,
			SHA256: oldTransition.TransitionSHA256,
			URI:    "vault://zerone/superseded-journal-head",
		},
	)
	sort.Slice(newDraft.Evidence, func(i, j int) bool {
		return evidenceLess(newDraft.Evidence[i], newDraft.Evidence[j])
	})
	newDraft.Approvals = []Approval{
		signApproval(
			t,
			newDraft,
			"operations-approver",
			"did:zrn:approver-1",
			1,
			"0",
		),
	}
	if err := validateTransitionAgainstTrustPolicy(
		newDraft,
		newPolicy,
		mustTestPolicySHA256(t, newPolicy),
		"",
	); err != nil {
		t.Fatalf("validate replacement transition: %v", err)
	}
	newTransition, err := sealTransition(newDraft)
	if err != nil {
		t.Fatalf("seal replacement transition: %v", err)
	}

	_, err = verifyReplacement(
		[][]byte{canonicalDocument(t, oldTransition)},
		sidecarDocument,
		[][]byte{canonicalDocument(t, newTransition)},
		VerifyOptions{
			ExpectedHeadSHA256: oldTransition.TransitionSHA256,
			TrustPolicy:        &oldPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, oldPolicy),
		},
		VerifyOptions{
			ExpectedHeadSHA256: newTransition.TransitionSHA256,
			TrustPolicy:        &newPolicy,
			TrustPolicySHA256:  mustTestPolicySHA256(t, newPolicy),
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "must use a new lane identifier") {
		t.Fatalf("same-identity replacement journal passed verification: %v", err)
	}
}

func TestRunVerifyReplacementComposesAllArtifacts(t *testing.T) {
	fixture := testReplacementFixture(t)
	root := t.TempDir()
	write := func(name string, document []byte) string {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, document, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	oldPolicyDocument, err := json.Marshal(fixture.oldPolicy)
	if err != nil {
		t.Fatal(err)
	}
	newPolicyDocument, err := json.Marshal(fixture.newPolicy)
	if err != nil {
		t.Fatal(err)
	}
	oldPolicyPath := write("old-policy.json", oldPolicyDocument)
	newPolicyPath := write("new-policy.json", newPolicyDocument)
	oldJournalPath := write("old-0001.json", fixture.oldDocuments[0])
	sidecarPath := write("supersession.json", fixture.sidecarDocument)
	newJournalPath := write("new-0001.json", fixture.newDocuments[0])
	var stdout, stderr bytes.Buffer
	exitCode := run(
		[]string{
			"verify-replacement",
			"--old-trust-policy", oldPolicyPath,
			"--old-trust-policy-sha256",
			mustTestPolicySHA256(t, fixture.oldPolicy),
			"--new-trust-policy", newPolicyPath,
			"--new-trust-policy-sha256",
			mustTestPolicySHA256(t, fixture.newPolicy),
			"--old-head-sha256", fixture.oldOptions.ExpectedHeadSHA256,
			"--new-head-sha256", fixture.newOptions.ExpectedHeadSHA256,
			"--old-journal", oldJournalPath,
			"--input", sidecarPath,
			"--new-journal", newJournalPath,
		},
		bytes.NewReader(nil),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf(
			"verify-replacement exit=%d stderr=%s",
			exitCode,
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "VALID-REPLACEMENT") {
		t.Fatalf("unexpected verify-replacement output: %s", stdout.String())
	}
}
