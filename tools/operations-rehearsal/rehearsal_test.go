package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCompileAndVerifyQuickReport(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	draft, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	compiled, document, err := compileReport(draft, root)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if compiled.ReportSHA256 == "" || compiled.EvidenceManifestSHA256 == "" {
		t.Fatalf("compiler did not populate digest fields")
	}
	if len(document) == 0 || document[len(document)-1] != '\n' ||
		bytes.Contains(document, []byte("\n  ")) {
		t.Fatalf("compiled document is not compact canonical JSON")
	}
	verified, err := verifyReport(document, root)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.ReportSHA256 != compiled.ReportSHA256 {
		t.Fatalf("report digest changed after verification")
	}
}

func TestCompileRejectsMissingAndUnhashedEvidence(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	report.Upgrade.Evidence = removeEvidenceKind(report.Upgrade.Evidence, "plan-info")
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), `required evidence kind "plan-info" is missing`) {
		t.Fatalf("missing kind error = %v", err)
	}

	root, report = makeTestReport(t, "quick")
	report.Upgrade.Evidence[0].SHA256 = ""
	draft, err = json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "evidence SHA-256") {
		t.Fatalf("unhashed evidence error = %v", err)
	}
}

func TestCompileRejectsCrossLinkDrift(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	report.Upgrade.PlanInfoSHA256 = strings.Repeat("0", 64)
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "indexed plan") {
		t.Fatalf("Plan.Info cross-link error = %v", err)
	}

	root, report = makeTestReport(t, "quick")
	report.FreshVolume.DestinationVolumeID = "vol_drifted"
	draft, err = json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "fresh-volume index") {
		t.Fatalf("home-manifest cross-link error = %v", err)
	}
}

func TestHeightSuccessorChecksRejectInt64Overflow(t *testing.T) {
	latch := H1LatchScenario{
		Outcome:                     "observed",
		ResumeFinalizationHeight:    math.MaxInt64,
		NextBlockHeight:             math.MinInt64,
		SameBlockOrdinaryTxRejected: true,
		NextBlockOrdinaryTxAccepted: true,
	}
	if err := validateH1Latch(latch); err == nil {
		t.Fatal("H+1 latch accepted an overflowing successor")
	}

	freshVolume := FreshVolumeScenario{
		Outcome:                 "observed",
		HomeManifestSHA256:      strings.Repeat("a", 64),
		DestinationVolumeID:     "vol_test_001",
		SourceLastHeight:        math.MaxInt64,
		DestinationStartHeight:  math.MaxInt64,
		FirstCommittedHeight:    math.MinInt64,
		SourceProcessStopped:    true,
		NoConcurrentSigner:      true,
		CompleteHomeVerified:    true,
		DatabaseGroupsVerified:  true,
		SigningStateVerified:    true,
		GenesisIdentityVerified: true,
		AuthorizedDeltaOnly:     true,
	}
	if err := validateFreshVolume(freshVolume); err == nil {
		t.Fatal("fresh-volume continuation accepted an overflowing successor")
	}

	start := DestinationStartObservation{
		StartHeight:          math.MaxInt64,
		FirstCommittedHeight: math.MinInt64,
		NoConcurrentSigner:   true,
	}
	if err := validateObservation(
		"destination-start",
		start,
	); err == nil {
		t.Fatal("destination-start evidence accepted an overflowing successor")
	}
}

func TestVerifyRejectsArtifactMutation(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	_, document, err := compileReport(draft, root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(report.Upgrade.Evidence[0].Path))
	if err := os.WriteFile(target, []byte(`{"mutated":true}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyReport(document, root); err == nil ||
		(!strings.Contains(err.Error(), "size mismatch") &&
			!strings.Contains(err.Error(), "SHA-256 mismatch")) {
		t.Fatalf("artifact mutation error = %v", err)
	}
}

func TestCompileRejectsMalformedJSONWithMatchingDigest(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(report.Upgrade.Evidence, "plan")
	if index < 0 {
		t.Fatal("plan evidence missing from test report")
	}
	path := filepath.Join(root, filepath.FromSlash(report.Upgrade.Evidence[index].Path))
	malformed, err := os.ReadFile("testdata/malformed.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := makeEvidenceRefWithoutContentValidation(
		root,
		report.Upgrade.Evidence[index].Path,
		"plan",
		"application/json",
	)
	if err != nil {
		t.Fatal(err)
	}
	report.Upgrade.Evidence[index] = reference
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "malformed JSON") {
		t.Fatalf("malformed JSON error = %v", err)
	}
}

func TestCompileRejectsGenericPassAssertionBypass(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(report.Quarantine.Evidence, "ordinary-tx-rejection")
	if index < 0 {
		t.Fatal("ordinary transaction evidence missing")
	}
	path := filepath.Join(
		root,
		filepath.FromSlash(report.Quarantine.Evidence[index].Path),
	)
	if err := os.WriteFile(path, []byte(`{"result":"pass"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := makeEvidenceRef(
		root,
		report.Quarantine.Evidence[index].Path,
		"ordinary-tx-rejection",
		"application/json",
	)
	if err != nil {
		t.Fatal(err)
	}
	report.Quarantine.Evidence[index] = reference
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "ordinary-tx-rejection evidence") {
		t.Fatalf("generic pass assertion bypass error = %v", err)
	}
}

func TestCompileRejectsFourFieldHomeManifestBypass(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(report.FreshVolume.Evidence, "source-home-manifest")
	if index < 0 {
		t.Fatal("source home manifest evidence missing")
	}
	path := filepath.Join(root, filepath.FromSlash(report.FreshVolume.Evidence[index].Path))
	fake := []byte(
		`{"schema":"zerone.validator-home-manifest/v1",` +
			`"destination_volume_id":"vol_test_001",` +
			`"last_height":120,` +
			`"manifest_sha256":"` + report.FreshVolume.HomeManifestSHA256 + `"}` + "\n",
	)
	if err := os.WriteFile(path, fake, 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := makeEvidenceRef(
		root,
		report.FreshVolume.Evidence[index].Path,
		"source-home-manifest",
		"application/json",
	)
	if err != nil {
		t.Fatal(err)
	}
	report.FreshVolume.Evidence[index] = reference
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "source home manifest") {
		t.Fatalf("four-field home manifest bypass error = %v", err)
	}
}

func TestFullHomeManifestRejectsRehashedStoppedEvidenceDrift(t *testing.T) {
	root := t.TempDir()
	manifest, _ := makeTestHomeManifestEvidence(t, root)
	manifest.StoppedEvidence.Method = "forged method"
	forHash := manifest
	forHash.ManifestSHA256 = ""
	var err error
	manifest.ManifestSHA256, err = hashCanonical(forHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFullValidatorHomeManifest(manifest); err == nil ||
		!strings.Contains(err.Error(), "stopped evidence self-hash mismatch") {
		t.Fatalf("stopped-evidence drift error = %v", err)
	}
}

func TestCompileRejectsRawExecutionArtifactMutation(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	reference, err := findEvidenceByKind(report.Upgrade.Evidence, "plan")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(reference.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	typed, err := decodeEvidenceEnvelope(data, "plan", true)
	if err != nil {
		t.Fatal(err)
	}
	rawPath := filepath.Join(
		root,
		filepath.FromSlash(typed.Envelope.Artifacts[0].Path),
	)
	if err := os.WriteFile(rawPath, []byte(`{"mutated":true}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		(!strings.Contains(err.Error(), "size mismatch") &&
			!strings.Contains(err.Error(), "SHA-256 mismatch")) {
		t.Fatalf("raw execution artifact mutation error = %v", err)
	}
}

func TestCompileRejectsEvidenceOutsideRunInterval(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(report.Upgrade.Evidence, "plan")
	report.Upgrade.Evidence[index] = rewriteTestEnvelope(
		t,
		root,
		report.Upgrade.Evidence[index],
		func(envelope *EvidenceEnvelope) {
			envelope.Execution.StartedAt = "2026-07-30T11:59:58Z"
			envelope.Execution.CompletedAt = "2026-07-30T11:59:59Z"
			envelope.CollectedAt = envelope.Execution.CompletedAt
		},
	)
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "outside the indexed rehearsal interval") {
		t.Fatalf("out-of-interval evidence error = %v", err)
	}
}

func TestCompileRejectsSourceProcessIdentityDrift(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(
		report.FreshVolume.Evidence,
		"source-process-absence",
	)
	report.FreshVolume.Evidence[index] = rewriteTestEnvelope(
		t,
		root,
		report.FreshVolume.Evidence[index],
		func(envelope *EvidenceEnvelope) {
			var observation SourceProcessObservation
			if err := decodeStrict(envelope.Observations, &observation); err != nil {
				t.Fatal(err)
			}
			observation.ProcessIdentitySHA256 = strings.Repeat("f", 64)
			encoded, err := canonicalJSON(observation)
			if err != nil {
				t.Fatal(err)
			}
			envelope.Observations = encoded
		},
	)
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "stopped manifest evidence") {
		t.Fatalf("source process identity drift error = %v", err)
	}
}

func TestOfflineIndexNeverAcceptsPassOutcome(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	report.Outcome = "pass"
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "makes no release decision") {
		t.Fatalf("global pass outcome error = %v", err)
	}

	root, report = makeTestReport(t, "quick")
	report.Quarantine.Outcome = "pass"
	draft, err = json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "quarantine") {
		t.Fatalf("scenario pass outcome error = %v", err)
	}
}

func TestCompileRejectsSymlinkEvidence(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	index := evidenceKindIndex(report.Observer.Evidence, "observer-status")
	targetPath := filepath.Join(root, filepath.FromSlash(report.Observer.Evidence[index].Path))
	replacement := filepath.Join(root, "replacement.json")
	if err := os.WriteFile(replacement, []byte(`{"replacement":true}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(targetPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, targetPath); err != nil {
		t.Fatal(err)
	}
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := compileReport(draft, root); err == nil ||
		!strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink evidence error = %v", err)
	}
}

func TestQuickFaultSetDoesNotSatisfyFullMode(t *testing.T) {
	_, report := makeTestReport(t, "quick")
	if err := validateFaults(report.Faults, "full"); err == nil ||
		!strings.Contains(err.Error(), "full-mode fault reason code") {
		t.Fatalf("full coverage error = %v", err)
	}
}

func TestFullReportCoversEntireMatrix(t *testing.T) {
	root, report := makeTestReport(t, "full")
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	compiled, document, err := compileReport(draft, root)
	if err != nil {
		t.Fatalf("compile full report: %v", err)
	}
	if len(compiled.Faults) != len(faultDefinitions) {
		t.Fatalf("compiled faults = %d, want %d", len(compiled.Faults), len(faultDefinitions))
	}
	if _, err := verifyReport(document, root); err != nil {
		t.Fatalf("verify full report: %v", err)
	}
}

func TestFaultMatrixIsCanonicalAndSelfHashed(t *testing.T) {
	matrix, err := buildFaultMatrix()
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Definitions) < 40 {
		t.Fatalf("fault matrix only has %d definitions", len(matrix.Definitions))
	}
	for index := 1; index < len(matrix.Definitions); index++ {
		if matrix.Definitions[index-1].ReasonCode >= matrix.Definitions[index].ReasonCode {
			t.Fatalf("fault matrix is not strictly sorted")
		}
	}
	forHash := matrix
	forHash.SHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		t.Fatal(err)
	}
	if digest != matrix.SHA256 {
		t.Fatalf("fault matrix self-hash mismatch")
	}
}

func TestVerifyRejectsNonCanonicalAndTamperedReport(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	compiled, document, err := compileReport(draft, root)
	if err != nil {
		t.Fatal(err)
	}
	nonCanonical := append([]byte(" "), document...)
	if _, err := verifyReport(nonCanonical, root); err == nil ||
		!strings.Contains(err.Error(), "not exact canonical") {
		t.Fatalf("non-canonical report error = %v", err)
	}
	compiled.RunID = "tampered-run"
	tampered, err := canonicalDocument(compiled)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyReport(tampered, root); err == nil ||
		!strings.Contains(err.Error(), "self-hash mismatch") {
		t.Fatalf("tampered report error = %v", err)
	}
}

func TestDigestAndMatrixCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "artifact.json"), []byte(`{"ok":true}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{
		"digest",
		"--evidence-root", root,
		"--path", "artifact.json",
		"--kind", "test",
		"--media-type", "application/json",
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("digest exit %d: %s", code, stderr.String())
	}
	var reference EvidenceRef
	if err := decodeStrict(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode digest output: %v", err)
	}
	if reference.Kind != "test" || reference.SizeBytes == 0 {
		t.Fatalf("unexpected reference: %+v", reference)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"fault-matrix"}, &stdout, &stderr); code != 0 {
		t.Fatalf("fault-matrix exit %d: %s", code, stderr.String())
	}
	var matrix FaultMatrix
	if err := decodeStrict(stdout.Bytes(), &matrix); err != nil {
		t.Fatalf("decode matrix: %v", err)
	}
	if matrix.Schema != faultMatrixSchema {
		t.Fatalf("matrix schema = %q", matrix.Schema)
	}
}

func TestRunCompileAndVerifyUsesEvidenceIndexSemantics(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	draftPath := filepath.Join(t.TempDir(), "draft.json")
	draft, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftPath, draft, 0o600); err != nil {
		t.Fatal(err)
	}
	outputDirectory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(outputDirectory, "report.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{
		"compile",
		"--draft", draftPath,
		"--evidence-root", root,
		"--out", reportPath,
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("compile exit %d: %s", code, stderr.String())
	}
	if !strings.HasPrefix(
		stdout.String(),
		"VERIFIED_EVIDENCE_INDEX provenance=self_attested ",
	) || !strings.Contains(stdout.String(), "release_decision=none") {
		t.Fatalf("unexpected compile summary: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{
		"verify",
		"--report", reportPath,
		"--evidence-root", root,
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("verify exit %d: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "VERIFIED_EVIDENCE_INDEX ") {
		t.Fatalf("unexpected verify summary: %s", stdout.String())
	}
}

func TestOutputCannotReplaceReferencedEvidence(t *testing.T) {
	root, report := makeTestReport(t, "quick")
	output := filepath.Join(root, filepath.FromSlash(report.Upgrade.Evidence[0].Path))
	if err := ensureOutputDoesNotReplaceEvidence(output, root, report); err == nil ||
		!strings.Contains(err.Error(), "would replace referenced evidence") {
		t.Fatalf("output/evidence collision error = %v", err)
	}
	if err := ensureOutputDoesNotReplaceEvidence(filepath.Join(root, "report.json"), root, report); err != nil {
		t.Fatalf("unreferenced output rejected: %v", err)
	}
}

func TestWriteAtomicRefusesReplacementAndSymlinkedParent(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(directory, "existing.json")
	if err := os.WriteFile(existing, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(existing, []byte("replacement"), &bytes.Buffer{}); err == nil ||
		!strings.Contains(err.Error(), "replacement is refused") {
		t.Fatalf("replacement refusal error = %v", err)
	}
	data, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("existing output was changed: %q", data)
	}

	realParent := filepath.Join(directory, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkParent := filepath.Join(directory, "linked-parent")
	if err := os.Symlink(realParent, symlinkParent); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(
		filepath.Join(symlinkParent, "report.json"),
		[]byte("{}\n"),
		&bytes.Buffer{},
	); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlink-parent refusal error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(realParent, "report.json")); !os.IsNotExist(err) {
		t.Fatalf("symlinked parent received output: %v", err)
	}
}

func makeTestReport(t *testing.T, mode string) (string, Report) {
	t.Helper()
	root := t.TempDir()
	homeManifest, homeReference := makeTestHomeManifestEvidence(t, root)
	artifactCounter := 0
	makeEvidence := func(section, kind string, observation any) EvidenceRef {
		t.Helper()
		artifactCounter++
		return makeTestEnvelopeEvidence(
			t,
			root,
			section,
			kind,
			observation,
			artifactCounter,
		)
	}
	makeSection := func(section string) []EvidenceRef {
		result := make([]EvidenceRef, 0, len(requiredEvidenceKinds[section]))
		for _, kind := range requiredEvidenceKinds[section] {
			if kind == "source-home-manifest" {
				result = append(result, homeReference)
				continue
			}
			result = append(result, makeEvidence(
				section,
				kind,
				testObservation(kind, homeManifest),
			))
		}
		return result
	}

	requiredCodes, err := requiredFaultCodes(mode)
	if err != nil {
		t.Fatal(err)
	}
	definitions := faultDefinitionMap()
	faults := make([]FaultScenario, 0, len(requiredCodes))
	for code := range requiredCodes {
		definition := definitions[code]
		id := "fault-" + strings.ToLower(strings.ReplaceAll(code, "_", "-"))
		fault := FaultScenario{
			ID:                        id,
			ReasonCode:                code,
			Phase:                     definition.Phase,
			InjectionPoint:            "deterministic test injection",
			ExpectedResult:            "operation refused and state contained",
			ActualResult:              "operation refused and state contained",
			LastCommittedHeight:       99,
			ForbiddenMutationObserved: false,
			ContainmentConfirmed:      true,
			Outcome:                   "observed",
		}
		evidence := make([]EvidenceRef, 0, len(requiredEvidenceKinds["fault"]))
		for _, kind := range requiredEvidenceKinds["fault"] {
			evidence = append(evidence, makeEvidence(
				"faults/"+id,
				kind,
				testFaultObservation(kind, fault),
			))
		}
		fault.Evidence = evidence
		faults = append(faults, fault)
	}
	upgradeEvidence := makeSection("upgrade")
	quarantineEvidence := makeSection("quarantine")
	recoveryEvidence := makeSection("recovery")
	latchEvidence := makeSection("h_plus_one_latch")
	freshVolumeEvidence := makeSection("fresh_volume")
	observerEvidence := makeSection("observer")

	return root, Report{
		Schema:             reportSchema,
		RunID:              "test-run-001",
		Mode:               mode,
		Outcome:            "evidence_indexed",
		ChainID:            "zerone-rehearsal-1",
		StartedAt:          "2026-07-30T12:00:00Z",
		CompletedAt:        "2026-07-30T12:30:00Z",
		SourceRevision:     "62727a995563434967b5bab1a22a0199a2d683ae",
		TargetRevision:     "7bded01f",
		SourceBinarySHA256: strings.Repeat("a", 64),
		TargetBinarySHA256: strings.Repeat("b", 64),
		Upgrade: UpgradeScenario{
			Outcome:                      "observed",
			PlanName:                     "sdk-0.53-ibc-10",
			PlanInfoSHA256:               strings.Repeat("c", 64),
			UpgradeHeight:                100,
			OldLastCommittedHeight:       99,
			PreUpgradeAppHash:            strings.Repeat("d", 64),
			PostUpgradeAppHash:           strings.Repeat("e", 64),
			ReplayAppHashes:              []string{strings.Repeat("e", 64), strings.Repeat("e", 64)},
			OldExitCount:                 1,
			TargetCommitCount:            1,
			AppliedPlanHeight:            100,
			OldDatabaseUnchanged:         true,
			ActivationPreflightSatisfied: true,
			SourceVersionsExact:          true,
			PlanInfoExact:                true,
			StateDeltasVerified:          true,
			PostUpgradeRestartObserved:   true,
			Evidence:                     upgradeEvidence,
		},
		Quarantine: QuarantineScenario{
			Outcome:                     "observed",
			HaltFinalizedHeight:         110,
			ObservedAdvancingHeight:     115,
			BlocksContinued:             true,
			OrdinaryTransactionRejected: true,
			WrappedTransactionRejected:  true,
			ICACallbackRejected:         true,
			NoAutomaticResume:           true,
			Evidence:                    quarantineEvidence,
		},
		Recovery: RecoveryScenario{
			Outcome:                      "observed",
			AuthorizationTupleExact:      true,
			WrongTupleRejected:           true,
			RevocationFinalized:          true,
			RevokedProposalFailed:        true,
			RevokedProposalDequeued:      true,
			RevokedProposalRefunded:      true,
			RevokedProposalCannotExecute: true,
			ResumeHeadDigestVerified:     true,
			Evidence:                     recoveryEvidence,
		},
		H1Latch: H1LatchScenario{
			Outcome:                     "observed",
			ResumeFinalizationHeight:    116,
			SameBlockOrdinaryTxRejected: true,
			NextBlockHeight:             117,
			NextBlockOrdinaryTxAccepted: true,
			Evidence:                    latchEvidence,
		},
		FreshVolume: FreshVolumeScenario{
			Outcome:                 "observed",
			HomeManifestSHA256:      homeManifest.ManifestSHA256,
			DestinationVolumeID:     "vol_test_001",
			SourceLastHeight:        120,
			DestinationStartHeight:  120,
			FirstCommittedHeight:    121,
			SourceProcessStopped:    true,
			NoConcurrentSigner:      true,
			CompleteHomeVerified:    true,
			DatabaseGroupsVerified:  true,
			SigningStateVerified:    true,
			GenesisIdentityVerified: true,
			AuthorizedDeltaOnly:     true,
			Evidence:                freshVolumeEvidence,
		},
		Observer: ObserverScenario{
			Outcome:           "observed",
			ObserverID:        "observer-a",
			Independent:       true,
			Authenticated:     true,
			ImmutableSnapshot: true,
			ChainID:           "zerone-rehearsal-1",
			ObservedHeight:    99,
			AppHash:           strings.Repeat("d", 64),
			PlanName:          "sdk-0.53-ibc-10",
			PlanHeight:        100,
			PlanConfirmed:     true,
			Evidence:          observerEvidence,
		},
		Faults: faults,
	}
}

func makeTestEnvelopeEvidence(
	t *testing.T,
	root, section, kind string,
	observation any,
	counter int,
) EvidenceRef {
	t.Helper()
	base := strings.ReplaceAll(kind, "_", "-") + "-" + decimal(counter)
	rawRelative := filepath.ToSlash(filepath.Join("raw", section, base+".json"))
	rawPath := filepath.Join(root, filepath.FromSlash(rawRelative))
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o700); err != nil {
		t.Fatal(err)
	}
	rawDocument := []byte(`{"event":` + quoted(kind) + `,"sequence":` + decimal(counter) + `}` + "\n")
	if err := os.WriteFile(rawPath, rawDocument, 0o600); err != nil {
		t.Fatal(err)
	}
	rawReference, err := makeEvidenceRef(root, rawRelative, "raw-output", "application/json")
	if err != nil {
		t.Fatal(err)
	}
	observationJSON, err := canonicalJSON(observation)
	if err != nil {
		t.Fatal(err)
	}
	envelope := EvidenceEnvelope{
		Schema:      evidenceEnvelopeSchema(kind),
		Kind:        kind,
		Outcome:     "observed",
		CollectedAt: "2026-07-30T12:10:00Z",
		Collector: EvidenceCollector{
			Name:         "zerone-test-collector",
			Version:      "v1",
			BinarySHA256: strings.Repeat("7", 64),
		},
		Execution: EvidenceExecution{
			InvocationID:  "invocation-" + decimal(counter),
			RunnerID:      "runner-test",
			ProcessID:     4242,
			CommandSHA256: strings.Repeat("6", 64),
			StartedAt:     "2026-07-30T12:09:59Z",
			CompletedAt:   "2026-07-30T12:10:00Z",
			ExitCode:      0,
			TimedOut:      false,
			Observed:      true,
		},
		Subject: EvidenceSubject{
			RunID:   "test-run-001",
			ChainID: "zerone-rehearsal-1",
		},
		Observations: observationJSON,
		Artifacts: []EnvelopeArtifact{{
			Role:      "command-output",
			Path:      rawReference.Path,
			MediaType: rawReference.MediaType,
			SizeBytes: rawReference.SizeBytes,
			SHA256:    rawReference.SHA256,
		}},
	}
	draft, err := canonicalDocument(envelope)
	if err != nil {
		t.Fatal(err)
	}
	sealed, document, err := sealEvidenceEnvelope(draft, root)
	if err != nil {
		t.Fatalf("seal %s evidence: %v", kind, err)
	}
	envelopeRelative := filepath.ToSlash(filepath.Join(section, base+"-envelope.json"))
	envelopePath := filepath.Join(root, filepath.FromSlash(envelopeRelative))
	if err := os.MkdirAll(filepath.Dir(envelopePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envelopePath, document, 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := makeEvidenceRef(root, envelopeRelative, kind, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if sealed.EnvelopeSHA256 == "" {
		t.Fatal("sealed envelope has no self-hash")
	}
	return reference
}

func testObservation(kind string, homeManifest FullValidatorHomeManifest) any {
	switch kind {
	case "old-binary-build":
		return BinaryBuildObservation{"source", "62727a995563434967b5bab1a22a0199a2d683ae", strings.Repeat("a", 64), "legacy-v1"}
	case "target-binary-build":
		return BinaryBuildObservation{"target", "7bded01f", strings.Repeat("b", 64), "target-v2"}
	case "plan":
		return PlanObservation{"sdk-0.53-ibc-10", 100, strings.Repeat("c", 64), "scheduled"}
	case "plan-info":
		return PlanInfoObservation{"sdk-0.53-ibc-10", 100, strings.Repeat("c", 64), "zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1"}
	case "activation-preflight":
		return ActivationPreflightObservation{
			"zerone.activation-preflight/v3", 99, strings.Repeat("d", 64),
			strings.Repeat("9", 64), true, true, true, true,
		}
	case "old-exit":
		return OldExitObservation{100, 99, true, 1, true}
	case "upgrade-info":
		return UpgradeInfoObservation{"sdk-0.53-ibc-10", 100, strings.Repeat("c", 64)}
	case "replay-a":
		return ReplayObservation{"clone-a", 100, strings.Repeat("e", 64), 100, 1}
	case "replay-b":
		return ReplayObservation{"clone-b", 100, strings.Repeat("e", 64), 100, 1}
	case "post-upgrade":
		return PostUpgradeObservation{100, strings.Repeat("e", 64), 100, true, true}
	case "handoff-result":
		return HandoffObservation{100, true, true, 1, true}
	case "halt-status":
		return HaltStatusObservation{110, true}
	case "height-advance":
		return HeightAdvanceObservation{110, 115, true}
	case "ordinary-tx-rejection":
		return TransactionRejectionObservation{112, true, true, true}
	case "quarantine-audit":
		return QuarantineAuditObservation{110, 115, true, strings.Repeat("8", 64)}
	case "authorization":
		return AuthorizationObservation{
			77, "guardian-test", strings.Repeat("1", 64), strings.Repeat("2", 64),
			strings.Repeat("3", 64), true,
		}
	case "wrong-tuple-rejection":
		return WrongTupleObservation{78, true, "unauthorized_recovery_tuple"}
	case "revocation":
		return RevocationObservation{77, 1, true}
	case "proposal-failure":
		return ProposalFailureObservation{77, true, true, true}
	case "proposal-refund":
		return ProposalRefundObservation{77, true, "100", "uzero"}
	case "resume-head":
		return ResumeHeadObservation{strings.Repeat("4", 64), true}
	case "same-block-rejection":
		return LatchObservation{116, true, false}
	case "next-block-admission":
		return LatchObservation{117, false, true}
	case "source-process-absence":
		return SourceProcessObservation{
			ProcessID:                        999999,
			ProcessStartTime:                 homeManifest.StoppedEvidence.ProcessStartTime,
			ProcessIdentitySHA256:            homeManifest.StoppedEvidence.ProcessIdentitySHA256,
			RestartInhibitEvidenceSHA256:     homeManifest.StoppedEvidence.RestartInhibitEvidenceSHA256,
			Height:                           120,
			AppHash:                          homeManifest.AppHash,
			LocalPIDAbsent:                   true,
			ExternalRestartAssertionVerified: false,
		}
	case "destination-home-verification":
		return DestinationVerificationObservation{
			HomeManifestSHA256:                homeManifest.ManifestSHA256,
			DestinationVolumeID:               "vol_test_001",
			DestinationDeviceID:               homeManifest.DestinationFilesystem.DeviceID,
			DestinationRootInode:              homeManifest.DestinationFilesystem.RootInode,
			VolumeEvidenceSHA256:              homeManifest.Destination.VolumeEvidenceSHA256,
			SnapshotEvidenceSHA256:            homeManifest.Snapshot.EvidenceSHA256,
			Height:                            120,
			AppHash:                           homeManifest.AppHash,
			LocalManifestVerified:             true,
			ExternalControlAssertionsVerified: false,
			DatabaseGroupsVerified:            true,
			SigningStateVerified:              true,
			GenesisIdentityVerified:           true,
			AuthorizedDeltaOnly:               true,
		}
	case "destination-start":
		return DestinationStartObservation{120, 121, true}
	case "observer-identity":
		return ObserverIdentityObservation{"observer-a", true, true}
	case "observer-plan":
		return ObserverPlanObservation{"sdk-0.53-ibc-10", 100, true}
	case "observer-status":
		return ObserverStatusObservation{"zerone-rehearsal-1", 99, strings.Repeat("d", 64), true}
	default:
		panic("missing test observation for " + kind)
	}
}

func testFaultObservation(kind string, fault FaultScenario) any {
	switch kind {
	case "fault-injection":
		return FaultInjectionObservation{
			fault.ID, fault.ReasonCode, fault.Phase, fault.InjectionPoint, true,
		}
	case "fault-result":
		return FaultResultObservation{
			fault.ID, fault.ExpectedResult, fault.ActualResult, true, "contained",
		}
	case "mutation-census":
		return MutationCensusObservation{
			fault.ID, fault.LastCommittedHeight, false, true,
		}
	default:
		panic("missing fault observation for " + kind)
	}
}

func makeTestHomeManifestEvidence(
	t *testing.T,
	root string,
) (FullValidatorHomeManifest, EvidenceRef) {
	t.Helper()
	files := []ValidatorFileRecord{
		{"config/genesis.json", 10, "0644", strings.Repeat("1", 64)},
		{"config/node_key.json", 10, "0600", strings.Repeat("2", 64)},
		{"config/priv_validator_key.json", 10, "0600", strings.Repeat("3", 64)},
		{"data/application.db/000001.log", 10, "0600", strings.Repeat("4", 64)},
		{"data/blockstore.db/000001.log", 10, "0600", strings.Repeat("5", 64)},
		{"data/priv_validator_state.json", 10, "0600", strings.Repeat("6", 64)},
		{"data/state.db/000001.log", 10, "0600", strings.Repeat("7", 64)},
	}
	content := ValidatorContentManifest{
		Directories: []ValidatorDirectoryRecord{
			{"config", "0700"},
			{"data", "0700"},
			{"data/application.db", "0700"},
			{"data/blockstore.db", "0700"},
			{"data/state.db", "0700"},
		},
		Files: files,
	}
	contentForHash := content
	contentForHash.SHA256 = ""
	var err error
	content.SHA256, err = hashCanonical(contentForHash)
	if err != nil {
		t.Fatal(err)
	}
	databases := []ValidatorDatabaseManifest{
		{Name: "application", Root: "data/application.db", Files: []ValidatorFileRecord{files[3]}},
		{Name: "blockstore", Root: "data/blockstore.db", Files: []ValidatorFileRecord{files[4]}},
		{Name: "comet_state", Root: "data/state.db", Files: []ValidatorFileRecord{files[6]}},
	}
	for index := range databases {
		forHash := databases[index]
		forHash.SHA256 = ""
		databases[index].SHA256, err = hashCanonical(forHash)
		if err != nil {
			t.Fatal(err)
		}
	}
	stopped := ValidatorStoppedEvidence{
		CapturedAt:                   "2026-07-30T12:00:00Z",
		Method:                       "unit-test stopped process",
		Observer:                     "observer-a",
		ProcessID:                    999999,
		ProcessStartTime:             "2026-07-30T11:00:00Z",
		ProcessIdentitySHA256:        strings.Repeat("3", 64),
		ProcessAbsent:                true,
		RestartInhibitEvidenceSHA256: strings.Repeat("4", 64),
	}
	stoppedDocument := ValidatorStoppedEvidenceDocument{
		Schema:                       validatorStoppedEvidenceSchema,
		CapturedAt:                   stopped.CapturedAt,
		Method:                       stopped.Method,
		Observer:                     stopped.Observer,
		SourceHome:                   "/srv/zerone-source",
		ProcessID:                    stopped.ProcessID,
		ProcessStartTime:             stopped.ProcessStartTime,
		ProcessIdentitySHA256:        stopped.ProcessIdentitySHA256,
		ProcessAbsent:                stopped.ProcessAbsent,
		RestartInhibitEvidenceSHA256: stopped.RestartInhibitEvidenceSHA256,
		LastHeight:                   120,
		AppHash:                      strings.Repeat("9", 64),
	}
	stopped.EvidenceSHA256, err = hashCanonical(stoppedDocument)
	if err != nil {
		t.Fatal(err)
	}
	manifest := FullValidatorHomeManifest{
		Schema:     validatorHomeManifestSchema,
		SourceHome: "/srv/zerone-source",
		SourceFilesystem: ValidatorFilesystemIdentity{
			CanonicalPath: "/srv/zerone-source",
			DeviceID:      101,
			RootInode:     1001,
			ReadOnly:      true,
		},
		DestinationFilesystem: ValidatorFilesystemIdentity{
			CanonicalPath: "/srv/zerone-destination",
			DeviceID:      202,
			RootInode:     2002,
			ReadOnly:      false,
		},
		Destination: ValidatorDestinationBinding{
			VolumeID:                "vol_test_001",
			VolumeEvidenceSHA256:    strings.Repeat("e", 64),
			ObservedEmptyBeforeCopy: true,
		},
		Snapshot: ValidatorSnapshotBinding{
			SnapshotID:     "snapshot-test-001",
			EvidenceSHA256: strings.Repeat("f", 64),
		},
		Chain: ValidatorChainIdentity{
			"zerone-rehearsal-1", 1, strings.Repeat("8", 64),
		},
		LastHeight: 120,
		AppHash:    strings.Repeat("9", 64),
		Consensus: ValidatorConsensusIdentity{
			strings.Repeat("A", 40),
			"tendermint/PubKeyEd25519",
			strings.Repeat("a", 64),
			strings.Repeat("b", 40),
			"tendermint/PrivKeyEd25519",
			strings.Repeat("b", 64),
		},
		SigningState: ValidatorSigningState{
			Height:           120,
			Round:            0,
			Step:             3,
			SignaturePresent: true,
			SignatureSHA256:  strings.Repeat("1", 64),
			SignBytesPresent: true,
			SignBytesSHA256:  strings.Repeat("2", 64),
			StateFileSHA256:  strings.Repeat("c", 64),
		},
		StoppedEvidence: stopped,
		Databases:       databases,
		Content:         content,
	}
	forHash := manifest
	forHash.ManifestSHA256 = ""
	manifest.ManifestSHA256, err = hashCanonical(forHash)
	if err != nil {
		t.Fatal(err)
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	relative := "fresh-volume/source-home-manifest.json"
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := makeEvidenceRef(root, relative, "source-home-manifest", "application/json")
	if err != nil {
		t.Fatal(err)
	}
	return manifest, reference
}

func removeEvidenceKind(values []EvidenceRef, kind string) []EvidenceRef {
	result := make([]EvidenceRef, 0, len(values))
	for _, value := range values {
		if value.Kind != kind {
			result = append(result, value)
		}
	}
	return result
}

func evidenceKindIndex(values []EvidenceRef, kind string) int {
	for index, value := range values {
		if value.Kind == kind {
			return index
		}
	}
	return -1
}

func makeEvidenceRefWithoutContentValidation(
	root, relative, kind, mediaType string,
) (EvidenceRef, error) {
	path, err := resolveEvidencePath(root, relative)
	if err != nil {
		return EvidenceRef{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return EvidenceRef{}, err
	}
	sum := sha256.Sum256(data)
	return EvidenceRef{
		Kind:      kind,
		Path:      relative,
		MediaType: mediaType,
		SizeBytes: int64(len(data)),
		SHA256:    hex.EncodeToString(sum[:]),
	}, nil
}

func rewriteTestEnvelope(
	t *testing.T,
	root string,
	reference EvidenceRef,
	mutate func(*EvidenceEnvelope),
) EvidenceRef {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(reference.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	typed, err := decodeEvidenceEnvelope(data, reference.Kind, true)
	if err != nil {
		t.Fatal(err)
	}
	envelope := typed.Envelope
	mutate(&envelope)
	envelope.EnvelopeSHA256 = ""
	draft, err := canonicalDocument(envelope)
	if err != nil {
		t.Fatal(err)
	}
	_, document, err := sealEvidenceEnvelope(draft, root)
	if err != nil {
		t.Fatalf("reseal %s: %v", reference.Kind, err)
	}
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
	updated, err := makeEvidenceRef(
		root,
		reference.Path,
		reference.Kind,
		reference.MediaType,
	)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func decimal(value int) string {
	return strconv.Itoa(value)
}

func quoted(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
