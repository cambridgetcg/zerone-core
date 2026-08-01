package main

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

var requiredEvidenceKinds = map[string][]string{
	"upgrade": {
		"activation-preflight",
		"handoff-result",
		"old-binary-build",
		"old-exit",
		"plan",
		"plan-info",
		"post-upgrade",
		"replay-a",
		"replay-b",
		"target-binary-build",
		"upgrade-info",
	},
	"quarantine": {
		"halt-status",
		"height-advance",
		"ordinary-tx-rejection",
		"quarantine-audit",
	},
	"recovery": {
		"authorization",
		"proposal-failure",
		"proposal-refund",
		"resume-head",
		"revocation",
		"wrong-tuple-rejection",
	},
	"h_plus_one_latch": {
		"next-block-admission",
		"same-block-rejection",
	},
	"fresh_volume": {
		"destination-home-verification",
		"destination-start",
		"source-home-manifest",
		"source-process-absence",
	},
	"observer": {
		"observer-identity",
		"observer-plan",
		"observer-status",
	},
	"fault": {
		"fault-injection",
		"fault-result",
		"mutation-census",
	},
}

func compileReport(draftData []byte, evidenceRoot string) (Report, []byte, error) {
	var report Report
	if err := decodeStrict(draftData, &report); err != nil {
		return Report{}, nil, err
	}
	if report.EvidenceManifestSHA256 != "" || report.ReportSHA256 != "" {
		return Report{}, nil, fmt.Errorf("draft computed digest fields must be empty")
	}
	normalizeReport(&report)
	if err := validateReport(report, false); err != nil {
		return Report{}, nil, err
	}
	if err := verifyEvidenceFiles(report, evidenceRoot); err != nil {
		return Report{}, nil, err
	}
	if err := verifyEvidenceCrossLinks(report, evidenceRoot); err != nil {
		return Report{}, nil, err
	}
	var err error
	report.EvidenceManifestSHA256, err = evidenceManifestDigest(report)
	if err != nil {
		return Report{}, nil, err
	}
	forHash := report
	forHash.ReportSHA256 = ""
	report.ReportSHA256, err = hashCanonical(forHash)
	if err != nil {
		return Report{}, nil, err
	}
	if err := validateReport(report, true); err != nil {
		return Report{}, nil, err
	}
	if err := verifyEvidenceFiles(report, evidenceRoot); err != nil {
		return Report{}, nil, fmt.Errorf("evidence changed during compilation: %w", err)
	}
	if err := verifyEvidenceCrossLinks(report, evidenceRoot); err != nil {
		return Report{}, nil, fmt.Errorf("typed evidence changed during compilation: %w", err)
	}
	document, err := canonicalDocument(report)
	if err != nil {
		return Report{}, nil, err
	}
	return report, document, nil
}

func verifyReport(document []byte, evidenceRoot string) (Report, error) {
	var report Report
	if err := decodeStrict(document, &report); err != nil {
		return Report{}, err
	}
	normalized := report
	normalizeReport(&normalized)
	if !reflect.DeepEqual(normalized, report) {
		return Report{}, fmt.Errorf("report arrays are not in canonical order")
	}
	if err := validateReport(report, true); err != nil {
		return Report{}, err
	}
	canonical, err := canonicalDocument(report)
	if err != nil {
		return Report{}, err
	}
	if !bytes.Equal(document, canonical) {
		return Report{}, fmt.Errorf("report is not exact canonical JSON")
	}
	expectedEvidenceDigest, err := evidenceManifestDigest(report)
	if err != nil {
		return Report{}, err
	}
	if expectedEvidenceDigest != report.EvidenceManifestSHA256 {
		return Report{}, fmt.Errorf(
			"evidence manifest SHA-256 mismatch: report=%s computed=%s",
			report.EvidenceManifestSHA256,
			expectedEvidenceDigest,
		)
	}
	forHash := report
	forHash.ReportSHA256 = ""
	expectedReportDigest, err := hashCanonical(forHash)
	if err != nil {
		return Report{}, err
	}
	if expectedReportDigest != report.ReportSHA256 {
		return Report{}, fmt.Errorf(
			"report self-hash mismatch: report=%s computed=%s",
			report.ReportSHA256,
			expectedReportDigest,
		)
	}
	if err := verifyEvidenceFiles(report, evidenceRoot); err != nil {
		return Report{}, err
	}
	if err := verifyEvidenceCrossLinks(report, evidenceRoot); err != nil {
		return Report{}, err
	}
	if err := verifyEvidenceFiles(report, evidenceRoot); err != nil {
		return Report{}, fmt.Errorf("evidence changed during verification: %w", err)
	}
	if err := verifyEvidenceCrossLinks(report, evidenceRoot); err != nil {
		return Report{}, fmt.Errorf("typed evidence changed during verification: %w", err)
	}
	return report, nil
}

func normalizeReport(report *Report) {
	sortEvidence := func(values []EvidenceRef) {
		sort.Slice(values, func(i, j int) bool {
			if values[i].Kind != values[j].Kind {
				return values[i].Kind < values[j].Kind
			}
			return values[i].Path < values[j].Path
		})
	}
	sortEvidence(report.Upgrade.Evidence)
	sortEvidence(report.Quarantine.Evidence)
	sortEvidence(report.Recovery.Evidence)
	sortEvidence(report.H1Latch.Evidence)
	sortEvidence(report.FreshVolume.Evidence)
	sortEvidence(report.Observer.Evidence)
	for index := range report.Faults {
		sortEvidence(report.Faults[index].Evidence)
	}
	sort.Slice(report.Faults, func(i, j int) bool {
		return report.Faults[i].ID < report.Faults[j].ID
	})
}

func validateReport(report Report, final bool) error {
	if report.Schema != reportSchema {
		return fmt.Errorf("schema must be %q", reportSchema)
	}
	if !identifierPattern.MatchString(report.RunID) {
		return fmt.Errorf("run_id must be a stable 1-128 character identifier")
	}
	if report.Mode != "quick" && report.Mode != "full" {
		return fmt.Errorf("mode must be quick or full")
	}
	if report.Outcome != "evidence_indexed" {
		return fmt.Errorf("report outcome must be evidence_indexed; offline verification makes no release decision")
	}
	if report.ChainID == "" || strings.TrimSpace(report.ChainID) != report.ChainID {
		return fmt.Errorf("chain_id must be non-empty and trimmed")
	}
	started, err := validateCanonicalTime("started_at", report.StartedAt)
	if err != nil {
		return err
	}
	completed, err := validateCanonicalTime("completed_at", report.CompletedAt)
	if err != nil {
		return err
	}
	if completed.Before(started) {
		return fmt.Errorf("completed_at must not precede started_at")
	}
	if report.SourceRevision == "" || report.TargetRevision == "" ||
		report.SourceRevision == report.TargetRevision {
		return fmt.Errorf("source and target revisions must be non-empty and different")
	}
	if err := validateSHA256("source binary SHA-256", report.SourceBinarySHA256); err != nil {
		return err
	}
	if err := validateSHA256("target binary SHA-256", report.TargetBinarySHA256); err != nil {
		return err
	}
	if report.SourceBinarySHA256 == report.TargetBinarySHA256 {
		return fmt.Errorf("source and target binaries must have different SHA-256 digests")
	}
	if err := validateUpgrade(report.Upgrade); err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	if err := validateQuarantine(report.Quarantine); err != nil {
		return fmt.Errorf("quarantine: %w", err)
	}
	if err := validateRecovery(report.Recovery); err != nil {
		return fmt.Errorf("recovery: %w", err)
	}
	if err := validateH1Latch(report.H1Latch); err != nil {
		return fmt.Errorf("h_plus_one_latch: %w", err)
	}
	if err := validateFreshVolume(report.FreshVolume); err != nil {
		return fmt.Errorf("fresh_volume: %w", err)
	}
	if err := validateObserver(report.Observer, report); err != nil {
		return fmt.Errorf("observer: %w", err)
	}
	if err := validateFaults(report.Faults, report.Mode); err != nil {
		return fmt.Errorf("faults: %w", err)
	}
	for section, check := range map[string]struct {
		evidence []EvidenceRef
		kinds    []string
	}{
		"upgrade":          {report.Upgrade.Evidence, requiredEvidenceKinds["upgrade"]},
		"quarantine":       {report.Quarantine.Evidence, requiredEvidenceKinds["quarantine"]},
		"recovery":         {report.Recovery.Evidence, requiredEvidenceKinds["recovery"]},
		"h_plus_one_latch": {report.H1Latch.Evidence, requiredEvidenceKinds["h_plus_one_latch"]},
		"fresh_volume":     {report.FreshVolume.Evidence, requiredEvidenceKinds["fresh_volume"]},
		"observer":         {report.Observer.Evidence, requiredEvidenceKinds["observer"]},
	} {
		if err := requireKinds(check.evidence, check.kinds); err != nil {
			return fmt.Errorf("%s evidence: %w", section, err)
		}
	}
	if _, err := allEvidence(report); err != nil {
		return err
	}
	if final {
		if err := validateSHA256("evidence manifest SHA-256", report.EvidenceManifestSHA256); err != nil {
			return err
		}
		if err := validateSHA256("report SHA-256", report.ReportSHA256); err != nil {
			return err
		}
	} else if report.EvidenceManifestSHA256 != "" || report.ReportSHA256 != "" {
		return fmt.Errorf("draft digest fields must be empty")
	}
	return nil
}

func validateUpgrade(scenario UpgradeScenario) error {
	if scenario.Outcome != "observed" || scenario.PlanName == "" ||
		strings.TrimSpace(scenario.PlanName) != scenario.PlanName {
		return fmt.Errorf("outcome must be observed and plan_name must be non-empty")
	}
	if err := validateSHA256("Plan.Info SHA-256", scenario.PlanInfoSHA256); err != nil {
		return err
	}
	if err := validateSHA256("H2 plan identity SHA-256", scenario.H2PlanIdentitySHA256); err != nil {
		return err
	}
	if scenario.UpgradeHeight <= 1 ||
		scenario.OldLastCommittedHeight != scenario.UpgradeHeight-1 ||
		scenario.AppliedPlanHeight != scenario.UpgradeHeight {
		return fmt.Errorf("old binary must stop at H-1 and applied plan must be exactly H")
	}
	if scenario.OldExitCount != 1 || scenario.TargetCommitCount != 1 {
		return fmt.Errorf("old exit and target H commit counts must both equal one")
	}
	if err := validateSHA256("pre-upgrade AppHash", scenario.PreUpgradeAppHash); err != nil {
		return err
	}
	if err := validateSHA256("post-upgrade AppHash", scenario.PostUpgradeAppHash); err != nil {
		return err
	}
	if len(scenario.ReplayAppHashes) < 2 {
		return fmt.Errorf("at least two H-1 replay AppHashes are required")
	}
	for _, digest := range scenario.ReplayAppHashes {
		if err := validateSHA256("replay AppHash", digest); err != nil {
			return err
		}
		if digest != scenario.PostUpgradeAppHash {
			return fmt.Errorf("all replay AppHashes must equal post_upgrade_app_hash")
		}
	}
	if !scenario.OldDatabaseUnchanged || !scenario.ActivationPreflightSatisfied ||
		!scenario.SourceVersionsExact || !scenario.PlanInfoExact ||
		!scenario.StateDeltasVerified || !scenario.PostUpgradeRestartObserved {
		return fmt.Errorf("all upgrade safety gates must be true")
	}
	return nil
}

func validateQuarantine(scenario QuarantineScenario) error {
	if scenario.Outcome != "observed" || scenario.HaltFinalizedHeight <= 0 ||
		scenario.ObservedAdvancingHeight <= scenario.HaltFinalizedHeight {
		return fmt.Errorf("halt height must be positive and observed height must advance")
	}
	if !scenario.BlocksContinued || !scenario.OrdinaryTransactionRejected ||
		!scenario.WrappedTransactionRejected || !scenario.ICACallbackRejected ||
		!scenario.NoAutomaticResume {
		return fmt.Errorf("all quarantine gates must be true")
	}
	return nil
}

func validateRecovery(scenario RecoveryScenario) error {
	if scenario.Outcome != "observed" || !scenario.AuthorizationTupleExact ||
		!scenario.WrongTupleRejected || !scenario.RevocationFinalized ||
		!scenario.RevokedProposalFailed || !scenario.RevokedProposalDequeued ||
		!scenario.RevokedProposalRefunded || !scenario.RevokedProposalCannotExecute ||
		!scenario.ResumeHeadDigestVerified {
		return fmt.Errorf("authorization, revocation/refund, and resume observations are incomplete")
	}
	return nil
}

func validateH1Latch(scenario H1LatchScenario) error {
	if scenario.Outcome != "observed" || scenario.ResumeFinalizationHeight <= 0 ||
		scenario.ResumeFinalizationHeight == math.MaxInt64 ||
		scenario.NextBlockHeight != scenario.ResumeFinalizationHeight+1 ||
		!scenario.SameBlockOrdinaryTxRejected || !scenario.NextBlockOrdinaryTxAccepted {
		return fmt.Errorf("ordinary admission must remain closed in the resume block and open at H+1")
	}
	return nil
}

func validateFreshVolume(scenario FreshVolumeScenario) error {
	if scenario.Outcome != "observed" {
		return fmt.Errorf("outcome must be observed")
	}
	if err := validateSHA256("validator-home manifest SHA-256", scenario.HomeManifestSHA256); err != nil {
		return err
	}
	if scenario.DestinationVolumeID == "" ||
		strings.TrimSpace(scenario.DestinationVolumeID) != scenario.DestinationVolumeID {
		return fmt.Errorf("destination_volume_id must be non-empty and trimmed")
	}
	if scenario.SourceLastHeight <= 0 || scenario.SourceLastHeight == math.MaxInt64 ||
		scenario.DestinationStartHeight != scenario.SourceLastHeight ||
		scenario.FirstCommittedHeight != scenario.SourceLastHeight+1 {
		return fmt.Errorf("destination must start at the source height and next commit must be source height + 1")
	}
	if !scenario.SourceProcessStopped || !scenario.NoConcurrentSigner ||
		!scenario.CompleteHomeVerified || !scenario.DatabaseGroupsVerified ||
		!scenario.SigningStateVerified || !scenario.GenesisIdentityVerified ||
		!scenario.AuthorizedDeltaOnly {
		return fmt.Errorf("all fresh-volume safety gates must be true")
	}
	return nil
}

func validateObserver(observer ObserverScenario, report Report) error {
	if observer.Outcome != "observed" || observer.ObserverID == "" ||
		strings.TrimSpace(observer.ObserverID) != observer.ObserverID {
		return fmt.Errorf("outcome must be observed and observer_id must be non-empty")
	}
	if !observer.Independent || !observer.Authenticated ||
		!observer.ImmutableSnapshot || !observer.PlanConfirmed {
		return fmt.Errorf("observer must be independent, authenticated, immutable, and plan-confirming")
	}
	if observer.ChainID != report.ChainID ||
		observer.ObservedHeight != report.Upgrade.OldLastCommittedHeight ||
		observer.AppHash != report.Upgrade.PreUpgradeAppHash ||
		observer.PlanName != report.Upgrade.PlanName ||
		observer.PlanHeight != report.Upgrade.UpgradeHeight {
		return fmt.Errorf("observer chain, H-1 AppHash, and plan must match the upgrade scenario")
	}
	return validateSHA256("observer AppHash", observer.AppHash)
}

func validateFaults(faults []FaultScenario, mode string) error {
	required, err := requiredFaultCodes(mode)
	if err != nil {
		return err
	}
	definitions := faultDefinitionMap()
	seenIDs := make(map[string]struct{}, len(faults))
	seenCodes := make(map[string]struct{}, len(faults))
	for _, fault := range faults {
		if !identifierPattern.MatchString(fault.ID) {
			return fmt.Errorf("fault ID %q is invalid", fault.ID)
		}
		if _, duplicate := seenIDs[fault.ID]; duplicate {
			return fmt.Errorf("fault ID %q is duplicated", fault.ID)
		}
		seenIDs[fault.ID] = struct{}{}
		definition, exists := definitions[fault.ReasonCode]
		if !exists {
			return fmt.Errorf("fault %s uses unknown reason code %q", fault.ID, fault.ReasonCode)
		}
		if _, duplicate := seenCodes[fault.ReasonCode]; duplicate {
			return fmt.Errorf("fault reason code %q is duplicated", fault.ReasonCode)
		}
		seenCodes[fault.ReasonCode] = struct{}{}
		if fault.Phase != definition.Phase {
			return fmt.Errorf(
				"fault %s phase %q does not match reason-code phase %q",
				fault.ID,
				fault.Phase,
				definition.Phase,
			)
		}
		if fault.InjectionPoint == "" || fault.ExpectedResult == "" || fault.ActualResult == "" {
			return fmt.Errorf("fault %s must describe injection, expected result, and actual result", fault.ID)
		}
		if isReleaseDecisionToken(fault.ExpectedResult) ||
			isReleaseDecisionToken(fault.ActualResult) {
			return fmt.Errorf("fault %s uses a reserved release-decision token", fault.ID)
		}
		if fault.LastCommittedHeight < 0 || fault.ForbiddenMutationObserved ||
			!fault.ContainmentConfirmed || fault.Outcome != "observed" {
			return fmt.Errorf("fault %s did not satisfy containment gates", fault.ID)
		}
		if err := requireKinds(fault.Evidence, requiredEvidenceKinds["fault"]); err != nil {
			return fmt.Errorf("fault %s evidence: %w", fault.ID, err)
		}
	}
	for code := range required {
		if _, covered := seenCodes[code]; !covered {
			return fmt.Errorf("required %s-mode fault reason code %s is missing", mode, code)
		}
	}
	return nil
}

func isReleaseDecisionToken(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass", "go":
		return true
	default:
		return false
	}
}

func requireKinds(evidence []EvidenceRef, required []string) error {
	seen := make(map[string]struct{}, len(evidence))
	for _, reference := range evidence {
		if _, duplicate := seen[reference.Kind]; duplicate {
			return fmt.Errorf("evidence kind %q is duplicated", reference.Kind)
		}
		seen[reference.Kind] = struct{}{}
	}
	for _, kind := range required {
		if _, exists := seen[kind]; !exists {
			return fmt.Errorf("required evidence kind %q is missing", kind)
		}
	}
	return nil
}

func validateReportPath(path string) error {
	if path == "" || path == "-" {
		return fmt.Errorf("report path must be a regular file")
	}
	return nil
}

func ensureOutputDoesNotReplaceEvidence(output, evidenceRoot string, report Report) error {
	if output == "-" {
		return nil
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	relative, err := filepath.Rel(evidenceRoot, filepath.Clean(absolute))
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil
	}
	relative = filepath.ToSlash(relative)
	references, err := allEvidence(report)
	if err != nil {
		return err
	}
	for _, reference := range references {
		if reference.Path == relative {
			return fmt.Errorf("output path would replace referenced evidence %q", relative)
		}
	}
	return nil
}
