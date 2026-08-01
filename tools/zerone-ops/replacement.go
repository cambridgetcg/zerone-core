package main

import (
	"errors"
	"fmt"
	"time"
)

// verifyReplacement composes the three independently signed artifacts used
// for a trust-root replacement. Verifying only one of these artifacts is not
// sufficient: exact old/new head pins prevent prefix acceptance, while the
// first new transition's signed evidence binds it to this sidecar and old
// journal head.
func verifyReplacement(
	oldDocuments [][]byte,
	supersessionDocument []byte,
	newDocuments [][]byte,
	oldOptions VerifyOptions,
	newOptions VerifyOptions,
) (ReplacementVerificationResult, error) {
	if err := validateSHA256(
		"externally pinned old journal head SHA-256",
		oldOptions.ExpectedHeadSHA256,
		false,
	); err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"an externally obtained old journal head is required: %w",
			err,
		)
	}
	if err := validateSHA256(
		"externally pinned replacement journal head SHA-256",
		newOptions.ExpectedHeadSHA256,
		false,
	); err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"an externally obtained replacement journal head is required: %w",
			err,
		)
	}

	oldResult, err := verifyDocuments(oldDocuments, oldOptions)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf("old journal: %w", err)
	}
	if oldOptions.TrustPolicy == nil || newOptions.TrustPolicy == nil {
		return ReplacementVerificationResult{}, errors.New(
			"old and replacement trust policies are required",
		)
	}
	supersessionResult, err := verifySupersession(
		supersessionDocument,
		*oldOptions.TrustPolicy,
		oldOptions.TrustPolicySHA256,
		*newOptions.TrustPolicy,
		newOptions.TrustPolicySHA256,
		oldOptions.ExpectedHeadSHA256,
	)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"supersession sidecar: %w",
			err,
		)
	}
	supersession, err := decodeSupersession(supersessionDocument, true)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"decode verified supersession sidecar: %w",
			err,
		)
	}

	effectiveNewOptions := newOptions
	if effectiveNewOptions.ExpectedChainID != "" &&
		effectiveNewOptions.ExpectedChainID != supersession.ChainID {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"replacement expected chain_id %q conflicts with supersession chain_id %q",
			effectiveNewOptions.ExpectedChainID,
			supersession.ChainID,
		)
	}
	effectiveNewOptions.ExpectedChainID = supersession.ChainID
	switch {
	case supersession.ReplacementIncidentID != "":
		if effectiveNewOptions.ExpectedReleaseID != "" {
			return ReplacementVerificationResult{}, errors.New(
				"incident replacement cannot set expected release_id",
			)
		}
		if effectiveNewOptions.ExpectedIncidentID != "" &&
			effectiveNewOptions.ExpectedIncidentID != supersession.ReplacementIncidentID {
			return ReplacementVerificationResult{}, fmt.Errorf(
				"replacement expected incident_id %q conflicts with supersession incident_id %q",
				effectiveNewOptions.ExpectedIncidentID,
				supersession.ReplacementIncidentID,
			)
		}
		effectiveNewOptions.ExpectedIncidentID = supersession.ReplacementIncidentID
	case supersession.ReplacementReleaseID != "":
		if effectiveNewOptions.ExpectedIncidentID != "" {
			return ReplacementVerificationResult{}, errors.New(
				"release replacement cannot set expected incident_id",
			)
		}
		if effectiveNewOptions.ExpectedReleaseID != "" &&
			effectiveNewOptions.ExpectedReleaseID != supersession.ReplacementReleaseID {
			return ReplacementVerificationResult{}, fmt.Errorf(
				"replacement expected release_id %q conflicts with supersession release_id %q",
				effectiveNewOptions.ExpectedReleaseID,
				supersession.ReplacementReleaseID,
			)
		}
		effectiveNewOptions.ExpectedReleaseID = supersession.ReplacementReleaseID
	default:
		return ReplacementVerificationResult{}, errors.New(
			"supersession has no replacement lane",
		)
	}

	newResult, err := verifyDocuments(newDocuments, effectiveNewOptions)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"replacement journal: %w",
			err,
		)
	}

	oldLast, err := decodeTransition(oldDocuments[len(oldDocuments)-1], true)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"decode verified old journal head: %w",
			err,
		)
	}
	newFirst, err := decodeTransition(newDocuments[0], true)
	if err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"decode verified first replacement transition: %w",
			err,
		)
	}
	if err := validateReplacementChronology(oldLast, supersession, newFirst); err != nil {
		return ReplacementVerificationResult{}, err
	}
	if err := validateCheckpointContinuity(
		oldLast.Checkpoint,
		newFirst.Checkpoint,
	); err != nil {
		return ReplacementVerificationResult{}, fmt.Errorf(
			"replacement checkpoint continuity: %w",
			err,
		)
	}
	if oldResult.Lane == newResult.Lane {
		sameIncident := oldResult.Lane == laneIncident &&
			oldResult.IncidentID == newResult.IncidentID
		sameRelease := oldResult.Lane == laneRelease &&
			oldResult.ReleaseID == newResult.ReleaseID
		if sameIncident || sameRelease {
			return ReplacementVerificationResult{}, fmt.Errorf(
				"replacement %s journal must use a new lane identifier",
				oldResult.Lane,
			)
		}
	}
	if err := validateReplacementEvidence(newFirst, supersession); err != nil {
		return ReplacementVerificationResult{}, err
	}

	return ReplacementVerificationResult{
		OldJournal:   oldResult,
		Supersession: supersessionResult,
		NewJournal:   newResult,
	}, nil
}

func validateReplacementChronology(
	oldLast Transition,
	supersession Supersession,
	newFirst Transition,
) error {
	oldTime, err := time.Parse(time.RFC3339Nano, oldLast.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse verified old head occurred_at: %w", err)
	}
	supersessionTime, err := time.Parse(time.RFC3339Nano, supersession.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse verified supersession occurred_at: %w", err)
	}
	newTime, err := time.Parse(time.RFC3339Nano, newFirst.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse verified replacement occurred_at: %w", err)
	}
	if supersessionTime.Before(oldTime) {
		return fmt.Errorf(
			"supersession occurred_at %q precedes old journal head %q",
			supersession.OccurredAt,
			oldLast.OccurredAt,
		)
	}
	if newTime.Before(supersessionTime) {
		return fmt.Errorf(
			"first replacement transition occurred_at %q precedes supersession %q",
			newFirst.OccurredAt,
			supersession.OccurredAt,
		)
	}
	return nil
}

func validateReplacementEvidence(
	first Transition,
	supersession Supersession,
) error {
	required := map[string]string{
		supersessionSidecarEvidence: supersession.SupersessionSHA256,
		supersededHeadEvidence:      supersession.OldJournalHeadSHA256,
	}
	counts := make(map[string]int, len(required))
	for _, item := range first.Evidence {
		expected, relevant := required[item.Type]
		if !relevant {
			continue
		}
		counts[item.Type]++
		if item.SHA256 != expected {
			return fmt.Errorf(
				"first replacement transition evidence %q has SHA-256 %s, expected %s",
				item.Type,
				item.SHA256,
				expected,
			)
		}
	}
	for evidenceType := range required {
		if counts[evidenceType] != 1 {
			return fmt.Errorf(
				"first replacement transition requires exactly one %q evidence item",
				evidenceType,
			)
		}
	}
	return nil
}
