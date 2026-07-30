package main

import (
	"fmt"
)

func verifyTypedEvidence(report Report, root string) error {
	references, err := allEvidence(report)
	if err != nil {
		return err
	}
	topLevelPaths := make(map[string]struct{}, len(references))
	for _, reference := range references {
		topLevelPaths[reference.Path] = struct{}{}
	}
	typedByPath := make(map[string]TypedEnvelope, len(references)-1)
	rawPaths := make(map[string]string)
	var homeManifest FullValidatorHomeManifest
	homeManifestSeen := false
	reportStarted, _ := validateCanonicalTime("report started_at", report.StartedAt)
	reportCompleted, _ := validateCanonicalTime("report completed_at", report.CompletedAt)
	for _, reference := range references {
		if reference.Kind == "source-home-manifest" {
			if homeManifestSeen {
				return fmt.Errorf("source-home-manifest evidence is duplicated")
			}
			homeManifest, err = loadValidatorHomeManifest(reference, root)
			if err != nil {
				return err
			}
			homeManifestSeen = true
			continue
		}
		typed, err := loadTypedEnvelope(reference, root)
		if err != nil {
			return err
		}
		if typed.Envelope.Subject.RunID != report.RunID ||
			typed.Envelope.Subject.ChainID != report.ChainID {
			return fmt.Errorf(
				"%s evidence subject does not match report run/chain identity",
				reference.Kind,
			)
		}
		evidenceStarted, _ := validateCanonicalTime(
			"evidence execution started_at",
			typed.Envelope.Execution.StartedAt,
		)
		evidenceCompleted, _ := validateCanonicalTime(
			"evidence execution completed_at",
			typed.Envelope.Execution.CompletedAt,
		)
		if evidenceStarted.Before(reportStarted) ||
			evidenceCompleted.After(reportCompleted) {
			return fmt.Errorf(
				"%s evidence execution is outside the indexed rehearsal interval",
				reference.Kind,
			)
		}
		for _, artifact := range typed.Envelope.Artifacts {
			if _, collision := topLevelPaths[artifact.Path]; collision {
				return fmt.Errorf(
					"%s raw artifact %q collides with a top-level evidence document",
					reference.Kind,
					artifact.Path,
				)
			}
			if owner, duplicate := rawPaths[artifact.Path]; duplicate {
				return fmt.Errorf(
					"raw artifact %q is shared by %s and %s evidence",
					artifact.Path,
					owner,
					reference.Kind,
				)
			}
			rawPaths[artifact.Path] = reference.Kind
		}
		typedByPath[reference.Path] = typed
	}
	if !homeManifestSeen {
		return fmt.Errorf("source-home-manifest evidence is missing")
	}
	if err := verifyUpgradeObservations(report, typedByPath); err != nil {
		return err
	}
	if err := verifyQuarantineObservations(report, typedByPath); err != nil {
		return err
	}
	if err := verifyRecoveryObservations(report, typedByPath); err != nil {
		return err
	}
	if err := verifyLatchObservations(report, typedByPath); err != nil {
		return err
	}
	if err := verifyFreshVolumeObservations(report, typedByPath, homeManifest); err != nil {
		return err
	}
	if err := verifyObserverObservations(report, typedByPath); err != nil {
		return err
	}
	for _, fault := range report.Faults {
		if err := verifyFaultObservations(report, fault, typedByPath); err != nil {
			return err
		}
	}
	return nil
}

func observationFor[T any](
	evidence []EvidenceRef,
	kind string,
	typedByPath map[string]TypedEnvelope,
) (T, error) {
	var zero T
	reference, err := findEvidenceByKind(evidence, kind)
	if err != nil {
		return zero, err
	}
	typed, exists := typedByPath[reference.Path]
	if !exists {
		return zero, fmt.Errorf("%s typed evidence was not loaded", kind)
	}
	observation, ok := typed.Observation.(T)
	if !ok {
		return zero, fmt.Errorf(
			"%s evidence decoded to unexpected observation type %T",
			kind,
			typed.Observation,
		)
	}
	return observation, nil
}

func verifyUpgradeObservations(report Report, typed map[string]TypedEnvelope) error {
	scenario := report.Upgrade
	source, err := observationFor[BinaryBuildObservation](
		scenario.Evidence,
		"old-binary-build",
		typed,
	)
	if err != nil {
		return err
	}
	if source.Role != "source" || source.Revision != report.SourceRevision ||
		source.BinarySHA256 != report.SourceBinarySHA256 {
		return fmt.Errorf("old-binary-build observations do not match source identity")
	}
	target, err := observationFor[BinaryBuildObservation](
		scenario.Evidence,
		"target-binary-build",
		typed,
	)
	if err != nil {
		return err
	}
	if target.Role != "target" || target.Revision != report.TargetRevision ||
		target.BinarySHA256 != report.TargetBinarySHA256 {
		return fmt.Errorf("target-binary-build observations do not match target identity")
	}
	plan, err := observationFor[PlanObservation](scenario.Evidence, "plan", typed)
	if err != nil {
		return err
	}
	planInfo, err := observationFor[PlanInfoObservation](scenario.Evidence, "plan-info", typed)
	if err != nil {
		return err
	}
	upgradeInfo, err := observationFor[UpgradeInfoObservation](
		scenario.Evidence,
		"upgrade-info",
		typed,
	)
	if err != nil {
		return err
	}
	for _, observed := range []struct {
		label  string
		name   string
		digest string
		height int64
	}{
		{"plan", plan.PlanName, plan.PlanInfoSHA256, plan.UpgradeHeight},
		{"plan-info", planInfo.PlanName, planInfo.PlanInfoSHA256, planInfo.UpgradeHeight},
		{"upgrade-info", upgradeInfo.PlanName, upgradeInfo.PlanInfoSHA256, upgradeInfo.UpgradeHeight},
	} {
		if observed.name != scenario.PlanName || observed.digest != scenario.PlanInfoSHA256 ||
			observed.height != scenario.UpgradeHeight {
			return fmt.Errorf("%s observations do not match the indexed plan", observed.label)
		}
	}
	preflight, err := observationFor[ActivationPreflightObservation](
		scenario.Evidence,
		"activation-preflight",
		typed,
	)
	if err != nil {
		return err
	}
	if preflight.Height != scenario.OldLastCommittedHeight ||
		preflight.AppHash != scenario.PreUpgradeAppHash {
		return fmt.Errorf("activation-preflight observations do not match H-1")
	}
	oldExit, err := observationFor[OldExitObservation](scenario.Evidence, "old-exit", typed)
	if err != nil {
		return err
	}
	if oldExit.UpgradeHeight != scenario.UpgradeHeight ||
		oldExit.LastCommittedHeight != scenario.OldLastCommittedHeight ||
		oldExit.UpgradeNeededLogCount != scenario.OldExitCount ||
		oldExit.DatabaseUnchanged != scenario.OldDatabaseUnchanged {
		return fmt.Errorf("old-exit observations do not match the upgrade index")
	}
	replayA, err := observationFor[ReplayObservation](scenario.Evidence, "replay-a", typed)
	if err != nil {
		return err
	}
	replayB, err := observationFor[ReplayObservation](scenario.Evidence, "replay-b", typed)
	if err != nil {
		return err
	}
	if replayA.CloneID == replayB.CloneID {
		return fmt.Errorf("replay evidence must use two distinct clone IDs")
	}
	for label, replay := range map[string]ReplayObservation{"replay-a": replayA, "replay-b": replayB} {
		if replay.UpgradeHeight != scenario.UpgradeHeight ||
			replay.AppHash != scenario.PostUpgradeAppHash ||
			replay.AppliedPlanHeight != scenario.AppliedPlanHeight ||
			replay.TargetCommitCount != scenario.TargetCommitCount {
			return fmt.Errorf("%s observations do not match the deterministic replay index", label)
		}
	}
	postUpgrade, err := observationFor[PostUpgradeObservation](
		scenario.Evidence,
		"post-upgrade",
		typed,
	)
	if err != nil {
		return err
	}
	if postUpgrade.UpgradeHeight != scenario.UpgradeHeight ||
		postUpgrade.AppHash != scenario.PostUpgradeAppHash ||
		postUpgrade.AppliedPlanHeight != scenario.AppliedPlanHeight ||
		postUpgrade.RestartObserved != scenario.PostUpgradeRestartObserved ||
		postUpgrade.StateDeltasVerified != scenario.StateDeltasVerified {
		return fmt.Errorf("post-upgrade observations do not match the upgrade index")
	}
	handoff, err := observationFor[HandoffObservation](
		scenario.Evidence,
		"handoff-result",
		typed,
	)
	if err != nil {
		return err
	}
	if handoff.UpgradeHeight != scenario.UpgradeHeight ||
		handoff.TargetCommitCount != scenario.TargetCommitCount {
		return fmt.Errorf("handoff observations do not match the upgrade index")
	}
	return nil
}

func verifyQuarantineObservations(report Report, typed map[string]TypedEnvelope) error {
	scenario := report.Quarantine
	halt, err := observationFor[HaltStatusObservation](scenario.Evidence, "halt-status", typed)
	if err != nil {
		return err
	}
	advance, err := observationFor[HeightAdvanceObservation](
		scenario.Evidence,
		"height-advance",
		typed,
	)
	if err != nil {
		return err
	}
	rejection, err := observationFor[TransactionRejectionObservation](
		scenario.Evidence,
		"ordinary-tx-rejection",
		typed,
	)
	if err != nil {
		return err
	}
	audit, err := observationFor[QuarantineAuditObservation](
		scenario.Evidence,
		"quarantine-audit",
		typed,
	)
	if err != nil {
		return err
	}
	if halt.HaltHeight != scenario.HaltFinalizedHeight ||
		advance.FromHeight != scenario.HaltFinalizedHeight ||
		advance.ToHeight != scenario.ObservedAdvancingHeight ||
		rejection.Height < scenario.HaltFinalizedHeight ||
		rejection.Height > scenario.ObservedAdvancingHeight ||
		audit.HaltHeight != scenario.HaltFinalizedHeight ||
		audit.ObservedHeight != scenario.ObservedAdvancingHeight {
		return fmt.Errorf("quarantine observations do not match the indexed heights")
	}
	return nil
}

func verifyRecoveryObservations(report Report, typed map[string]TypedEnvelope) error {
	scenario := report.Recovery
	authorization, err := observationFor[AuthorizationObservation](
		scenario.Evidence,
		"authorization",
		typed,
	)
	if err != nil {
		return err
	}
	if _, err := observationFor[WrongTupleObservation](
		scenario.Evidence,
		"wrong-tuple-rejection",
		typed,
	); err != nil {
		return err
	}
	revocation, err := observationFor[RevocationObservation](
		scenario.Evidence,
		"revocation",
		typed,
	)
	if err != nil {
		return err
	}
	failure, err := observationFor[ProposalFailureObservation](
		scenario.Evidence,
		"proposal-failure",
		typed,
	)
	if err != nil {
		return err
	}
	refund, err := observationFor[ProposalRefundObservation](
		scenario.Evidence,
		"proposal-refund",
		typed,
	)
	if err != nil {
		return err
	}
	if _, err := observationFor[ResumeHeadObservation](
		scenario.Evidence,
		"resume-head",
		typed,
	); err != nil {
		return err
	}
	if revocation.ProposalID != authorization.ProposalID ||
		failure.ProposalID != authorization.ProposalID ||
		refund.ProposalID != authorization.ProposalID {
		return fmt.Errorf("authorization, revocation, failure, and refund observations use different proposal IDs")
	}
	return nil
}

func verifyLatchObservations(report Report, typed map[string]TypedEnvelope) error {
	scenario := report.H1Latch
	sameBlock, err := observationFor[LatchObservation](
		scenario.Evidence,
		"same-block-rejection",
		typed,
	)
	if err != nil {
		return err
	}
	nextBlock, err := observationFor[LatchObservation](
		scenario.Evidence,
		"next-block-admission",
		typed,
	)
	if err != nil {
		return err
	}
	if sameBlock.Height != scenario.ResumeFinalizationHeight ||
		!sameBlock.Rejected || sameBlock.Accepted ||
		nextBlock.Height != scenario.NextBlockHeight ||
		nextBlock.Rejected || !nextBlock.Accepted {
		return fmt.Errorf("H+1 latch observations do not match indexed admission behavior")
	}
	return nil
}

func verifyFreshVolumeObservations(
	report Report,
	typed map[string]TypedEnvelope,
	manifest FullValidatorHomeManifest,
) error {
	scenario := report.FreshVolume
	if manifest.ManifestSHA256 != scenario.HomeManifestSHA256 ||
		manifest.Chain.ChainID != report.ChainID ||
		manifest.Destination.VolumeID != scenario.DestinationVolumeID ||
		manifest.LastHeight != scenario.SourceLastHeight {
		return fmt.Errorf("full source-home manifest does not match fresh-volume index")
	}
	sourceProcess, err := observationFor[SourceProcessObservation](
		scenario.Evidence,
		"source-process-absence",
		typed,
	)
	if err != nil {
		return err
	}
	if sourceProcess.ProcessID != int64(manifest.StoppedEvidence.ProcessID) ||
		sourceProcess.ProcessStartTime != manifest.StoppedEvidence.ProcessStartTime ||
		sourceProcess.ProcessIdentitySHA256 !=
			manifest.StoppedEvidence.ProcessIdentitySHA256 ||
		sourceProcess.RestartInhibitEvidenceSHA256 !=
			manifest.StoppedEvidence.RestartInhibitEvidenceSHA256 ||
		sourceProcess.Height != manifest.LastHeight ||
		sourceProcess.AppHash != manifest.AppHash {
		return fmt.Errorf("source-process observations do not match stopped manifest evidence")
	}
	destination, err := observationFor[DestinationVerificationObservation](
		scenario.Evidence,
		"destination-home-verification",
		typed,
	)
	if err != nil {
		return err
	}
	if destination.HomeManifestSHA256 != scenario.HomeManifestSHA256 ||
		destination.DestinationVolumeID != scenario.DestinationVolumeID ||
		destination.DestinationDeviceID != manifest.DestinationFilesystem.DeviceID ||
		destination.DestinationRootInode != manifest.DestinationFilesystem.RootInode ||
		destination.VolumeEvidenceSHA256 != manifest.Destination.VolumeEvidenceSHA256 ||
		destination.SnapshotEvidenceSHA256 != manifest.Snapshot.EvidenceSHA256 ||
		destination.Height != scenario.SourceLastHeight ||
		destination.AppHash != manifest.AppHash {
		return fmt.Errorf("destination verification observations do not match fresh-volume index")
	}
	start, err := observationFor[DestinationStartObservation](
		scenario.Evidence,
		"destination-start",
		typed,
	)
	if err != nil {
		return err
	}
	if start.StartHeight != scenario.DestinationStartHeight ||
		start.FirstCommittedHeight != scenario.FirstCommittedHeight ||
		start.NoConcurrentSigner != scenario.NoConcurrentSigner {
		return fmt.Errorf("destination-start observations do not match fresh-volume index")
	}
	return nil
}

func verifyObserverObservations(report Report, typed map[string]TypedEnvelope) error {
	scenario := report.Observer
	identity, err := observationFor[ObserverIdentityObservation](
		scenario.Evidence,
		"observer-identity",
		typed,
	)
	if err != nil {
		return err
	}
	plan, err := observationFor[ObserverPlanObservation](
		scenario.Evidence,
		"observer-plan",
		typed,
	)
	if err != nil {
		return err
	}
	status, err := observationFor[ObserverStatusObservation](
		scenario.Evidence,
		"observer-status",
		typed,
	)
	if err != nil {
		return err
	}
	if identity.ObserverID != scenario.ObserverID ||
		identity.Independent != scenario.Independent ||
		identity.Authenticated != scenario.Authenticated ||
		plan.PlanName != scenario.PlanName || plan.PlanHeight != scenario.PlanHeight ||
		status.ChainID != scenario.ChainID || status.Height != scenario.ObservedHeight ||
		status.AppHash != scenario.AppHash ||
		status.ImmutableSnapshot != scenario.ImmutableSnapshot {
		return fmt.Errorf("observer observations do not match observer index")
	}
	return nil
}

func verifyFaultObservations(
	report Report,
	fault FaultScenario,
	typed map[string]TypedEnvelope,
) error {
	injection, err := observationFor[FaultInjectionObservation](
		fault.Evidence,
		"fault-injection",
		typed,
	)
	if err != nil {
		return err
	}
	result, err := observationFor[FaultResultObservation](
		fault.Evidence,
		"fault-result",
		typed,
	)
	if err != nil {
		return err
	}
	census, err := observationFor[MutationCensusObservation](
		fault.Evidence,
		"mutation-census",
		typed,
	)
	if err != nil {
		return err
	}
	if injection.FaultID != fault.ID || injection.ReasonCode != fault.ReasonCode ||
		injection.Phase != fault.Phase || injection.InjectionPoint != fault.InjectionPoint ||
		result.FaultID != fault.ID || result.ExpectedResult != fault.ExpectedResult ||
		result.ActualResult != fault.ActualResult ||
		result.ContainmentConfirmed != fault.ContainmentConfirmed ||
		census.FaultID != fault.ID ||
		census.LastCommittedHeight != fault.LastCommittedHeight ||
		census.ForbiddenMutationObserved != fault.ForbiddenMutationObserved {
		return fmt.Errorf("fault %s typed observations do not match fault index", fault.ID)
	}
	if result.Result != "contained" || fault.Outcome != "observed" ||
		report.Outcome != "evidence_indexed" {
		return fmt.Errorf("fault %s result uses release-decision semantics", fault.ID)
	}
	return nil
}
