package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	releaseTransitions = map[State]map[State]bool{
		StateRunning:       {StatePreparing: true},
		StatePreparing:     {StateReleaseFrozen: true, StateCancelled: true},
		StateReleaseFrozen: {StateScheduled: true, StateCancelled: true},
		StateScheduled:     {StateStaged: true, StateCancelled: true},
		StateStaged:        {StateHaltedAtH: true, StateCancelled: true},
		StateHaltedAtH:     {StateMigratingH: true, StateRecoveryFailed: true},
		StateMigratingH:    {StateObserving: true, StateRecoveryFailed: true},
		StateObserving:     {StateAccepted: true, StateRecoveryFailed: true},
	}
	incidentTransitions = map[State]map[State]bool{
		StateRunning:        {StateAssessing: true},
		StateAssessing:      {StateContaining: true, StateForkChoice: true},
		StateContaining:     {StateSafetyStopped: true, StateRecoveryDesign: true, StateForkChoice: true},
		StateSafetyStopped:  {StateRecoveryDesign: true, StateForkChoice: true},
		StateRecoveryDesign: {StateRecoveryReady: true, StateForkChoice: true},
		StateRecoveryReady:  {StateActivating: true, StateForkChoice: true},
		StateActivating:     {StateObserving: true, StateRecoveryFailed: true, StateForkChoice: true},
		StateObserving:      {StateClosed: true, StateRecoveryFailed: true, StateForkChoice: true},
	}
	knownStates = map[State]bool{
		StateRunning: true, StatePreparing: true, StateReleaseFrozen: true,
		StateScheduled: true, StateStaged: true, StateHaltedAtH: true,
		StateMigratingH: true, StateObserving: true, StateAccepted: true,
		StateCancelled: true, StateAssessing: true, StateContaining: true,
		StateSafetyStopped: true, StateRecoveryDesign: true,
		StateRecoveryReady: true, StateActivating: true,
		StateRecoveryFailed: true, StateForkChoice: true,
		StateClosed: true,
	}
)

// decodeTransition rejects unknown fields, duplicate/trailing JSON through the
// canonical comparison, and multiple JSON values. Canonical documents are the
// exact compact bytes emitted by encoding/json for Transition, with no newline.
func decodeTransition(data []byte, requireCanonical bool) (Transition, error) {
	var transition Transition
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&transition); err != nil {
		return Transition{}, fmt.Errorf("decode JSON: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Transition{}, errors.New("decode JSON: multiple values are not allowed")
		}
		return Transition{}, fmt.Errorf("decode trailing JSON: %w", err)
	}
	if requireCanonical {
		canonical, err := json.Marshal(transition)
		if err != nil {
			return Transition{}, fmt.Errorf("canonicalize transition: %w", err)
		}
		if !bytes.Equal(data, canonical) {
			return Transition{}, errors.New("JSON is not canonical: require exact compact field order with no whitespace or trailing newline")
		}
	}
	return transition, nil
}

func canonicalTransition(transition Transition) ([]byte, error) {
	return json.Marshal(transition)
}

// transitionHash is SHA-256 over canonical JSON with transition_sha256 empty.
func transitionHash(transition Transition) (string, error) {
	transition.TransitionSHA256 = ""
	canonical, err := canonicalTransition(transition)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

type approvalClaim struct {
	Role      string `json:"role"`
	Identity  string `json:"identity"`
	PublicKey string `json:"public_key"`
	Power     string `json:"power"`
}

type approvalStatement struct {
	Transition Transition    `json:"transition"`
	Approval   approvalClaim `json:"approval"`
}

// approvalStatementDigest binds an approval to the transition and that
// approver's role, identity, public key, and declared power. It excludes the
// approval set, signatures, and self-hash to avoid circular dependencies.
func approvalStatementDigest(transition Transition, approval Approval) ([sha256.Size]byte, error) {
	transition.Approvals = []Approval{}
	transition.TransitionSHA256 = ""
	statement := approvalStatement{
		Transition: transition,
		Approval: approvalClaim{
			Role:      approval.Role,
			Identity:  approval.Identity,
			PublicKey: approval.PublicKey,
			Power:     approval.Power,
		},
	}
	canonical, err := json.Marshal(statement)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	message := make([]byte, 0, len(approvalDomain)+len(canonical))
	message = append(message, approvalDomain...)
	message = append(message, canonical...)
	return sha256.Sum256(message), nil
}

func sealTransition(transition Transition) (Transition, error) {
	if transition.TransitionSHA256 != "" {
		return Transition{}, errors.New("transition_sha256 must be empty before sealing")
	}
	if err := validateTransitionBody(transition); err != nil {
		return Transition{}, err
	}
	if err := validateStandaloneEdge(transition); err != nil {
		return Transition{}, err
	}
	hash, err := transitionHash(transition)
	if err != nil {
		return Transition{}, fmt.Errorf("compute transition hash: %w", err)
	}
	transition.TransitionSHA256 = hash
	return transition, nil
}

func verifyDocuments(documents [][]byte, options VerifyOptions) (VerificationResult, error) {
	if len(documents) == 0 {
		return VerificationResult{}, errors.New("at least one transition is required")
	}
	if err := validateExpectedOptions(options); err != nil {
		return VerificationResult{}, err
	}

	transitions := make([]Transition, len(documents))
	seenSequences := make(map[uint64]bool, len(documents))
	for i, document := range documents {
		transition, err := decodeTransition(document, true)
		if err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", i+1, err)
		}
		if err := validateSealedTransition(transition); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", i+1, err)
		}
		if err := validateTransitionAgainstTrustPolicy(
			transition,
			*options.TrustPolicy,
			options.TrustPolicySHA256,
			options.ExpectedPowerSnapshotSHA256[transition.Sequence],
		); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", i+1, err)
		}
		transitions[i] = transition
		seenSequences[transition.Sequence] = true
	}
	for sequence := range options.ExpectedPowerSnapshotSHA256 {
		if !seenSequences[sequence] {
			return VerificationResult{}, fmt.Errorf(
				"externally pinned power snapshot sequence %d is not present in the journal",
				sequence,
			)
		}
	}

	first := transitions[0]
	if first.Sequence != 1 {
		return VerificationResult{}, fmt.Errorf("transition 1: first sequence must be 1, got %d", first.Sequence)
	}
	if first.PreviousTransitionSHA256 != "" {
		return VerificationResult{}, errors.New("transition 1: previous_transition_sha256 must be empty")
	}
	if first.From != StateRunning {
		return VerificationResult{}, fmt.Errorf("transition 1: journal must begin at %s, got %s", StateRunning, first.From)
	}
	journalLane, err := laneForFirstTransition(first)
	if err != nil {
		return VerificationResult{}, fmt.Errorf("transition 1: %w", err)
	}
	if err := validateLaneIdentity(first, journalLane); err != nil {
		return VerificationResult{}, fmt.Errorf("transition 1: %w", err)
	}
	if err := validateLaneTransition(first, journalLane); err != nil {
		return VerificationResult{}, fmt.Errorf("transition 1: %w", err)
	}
	if err := validateLaneGates(first, journalLane); err != nil {
		return VerificationResult{}, fmt.Errorf("transition 1: %w", err)
	}

	for i := 1; i < len(transitions); i++ {
		previous := transitions[i-1]
		current := transitions[i]
		position := i + 1

		if previous.Sequence == ^uint64(0) || current.Sequence != previous.Sequence+1 {
			return VerificationResult{}, fmt.Errorf(
				"transition %d: sequence must be %d, got %d",
				position,
				previous.Sequence+1,
				current.Sequence,
			)
		}
		if current.PreviousTransitionSHA256 != previous.TransitionSHA256 {
			return VerificationResult{}, fmt.Errorf(
				"transition %d: previous_transition_sha256 %q does not match transition %d hash %q",
				position,
				current.PreviousTransitionSHA256,
				position-1,
				previous.TransitionSHA256,
			)
		}
		if current.TrustPolicySHA256 != previous.TrustPolicySHA256 {
			return VerificationResult{}, fmt.Errorf(
				"transition %d: trust_policy_sha256 changed from %q to %q; v1 journals require an immutable externally authorized policy",
				position,
				previous.TrustPolicySHA256,
				current.TrustPolicySHA256,
			)
		}
		if current.From != previous.To {
			return VerificationResult{}, fmt.Errorf(
				"transition %d: from state %s does not match previous to state %s",
				position,
				current.From,
				previous.To,
			)
		}
		if err := validateIdentityContinuity(previous, current); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
		if err := validateReleaseContinuity(previous.Release, current.Release); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
		if err := validateCheckpointContinuity(previous.Checkpoint, current.Checkpoint); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
		previousTime, _ := time.Parse(time.RFC3339Nano, previous.OccurredAt)
		currentTime, _ := time.Parse(time.RFC3339Nano, current.OccurredAt)
		if currentTime.Before(previousTime) {
			return VerificationResult{}, fmt.Errorf(
				"transition %d: occurred_at %q precedes previous value %q",
				position,
				current.OccurredAt,
				previous.OccurredAt,
			)
		}
		if err := validateLaneIdentity(current, journalLane); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
		if err := validateLaneTransition(current, journalLane); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
		if err := validateLaneGates(current, journalLane); err != nil {
			return VerificationResult{}, fmt.Errorf("transition %d: %w", position, err)
		}
	}

	last := transitions[len(transitions)-1]
	if options.ExpectedChainID != "" && first.ChainID != options.ExpectedChainID {
		return VerificationResult{}, fmt.Errorf("chain_id mismatch: journal has %q, expected %q", first.ChainID, options.ExpectedChainID)
	}
	if options.ExpectedIncidentID != "" && first.IncidentID != options.ExpectedIncidentID {
		return VerificationResult{}, fmt.Errorf("incident_id mismatch: journal has %q, expected %q", first.IncidentID, options.ExpectedIncidentID)
	}
	if options.ExpectedReleaseID != "" && first.ReleaseID != options.ExpectedReleaseID {
		return VerificationResult{}, fmt.Errorf("release_id mismatch: journal has %q, expected %q", first.ReleaseID, options.ExpectedReleaseID)
	}
	if options.ExpectedBinarySHA256 != "" && last.Release.BinarySHA256 != options.ExpectedBinarySHA256 {
		return VerificationResult{}, fmt.Errorf(
			"binary_sha256 mismatch: journal has %q, expected %q",
			last.Release.BinarySHA256,
			options.ExpectedBinarySHA256,
		)
	}
	if options.ExpectedHeadSHA256 != "" && last.TransitionSHA256 != options.ExpectedHeadSHA256 {
		return VerificationResult{}, fmt.Errorf(
			"head SHA-256 mismatch: journal has %q, expected %q",
			last.TransitionSHA256,
			options.ExpectedHeadSHA256,
		)
	}

	return VerificationResult{
		Transitions: len(transitions),
		Lane:        journalLane,
		ChainID:     first.ChainID,
		IncidentID:  first.IncidentID,
		ReleaseID:   first.ReleaseID,
		State:       last.To,
		HeadSHA256:  last.TransitionSHA256,
	}, nil
}

func validateExpectedOptions(options VerifyOptions) error {
	if options.TrustPolicy == nil {
		return errors.New("an externally pinned trust policy is required")
	}
	if err := validateTrustPolicy(*options.TrustPolicy); err != nil {
		return fmt.Errorf("invalid externally pinned trust policy: %w", err)
	}
	if err := validateSHA256("trusted policy SHA-256", options.TrustPolicySHA256, false); err != nil {
		return err
	}
	actualTrustPolicySHA256, err := trustPolicySHA256(*options.TrustPolicy)
	if err != nil {
		return err
	}
	if actualTrustPolicySHA256 != options.TrustPolicySHA256 {
		return fmt.Errorf(
			"externally pinned trust policy SHA-256 mismatch: policy has %s, expected %s",
			actualTrustPolicySHA256,
			options.TrustPolicySHA256,
		)
	}
	hashes := []struct {
		name  string
		value string
	}{
		{name: "expected binary SHA-256", value: options.ExpectedBinarySHA256},
		{name: "expected head SHA-256", value: options.ExpectedHeadSHA256},
	}
	for _, hash := range hashes {
		if hash.value != "" {
			if err := validateSHA256(hash.name, hash.value, false); err != nil {
				return err
			}
		}
	}
	for sequence, digest := range options.ExpectedPowerSnapshotSHA256 {
		if sequence == 0 {
			return errors.New("expected power snapshot sequence must be greater than zero")
		}
		if err := validateSHA256(
			fmt.Sprintf("expected power snapshot SHA-256 at sequence %d", sequence),
			digest,
			false,
		); err != nil {
			return err
		}
	}
	identities := []struct {
		name  string
		value string
	}{
		{name: "expected chain ID", value: options.ExpectedChainID},
		{name: "expected incident ID", value: options.ExpectedIncidentID},
		{name: "expected release ID", value: options.ExpectedReleaseID},
	}
	for _, identity := range identities {
		if identity.value != "" {
			if err := validateLabel(identity.name, identity.value, 128); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSealedTransition(transition Transition) error {
	if err := validateTransitionBody(transition); err != nil {
		return err
	}
	if err := validateSHA256("transition_sha256", transition.TransitionSHA256, false); err != nil {
		return err
	}
	expected, err := transitionHash(transition)
	if err != nil {
		return fmt.Errorf("compute transition hash: %w", err)
	}
	if transition.TransitionSHA256 != expected {
		return fmt.Errorf("transition_sha256 mismatch: got %q, expected %q", transition.TransitionSHA256, expected)
	}
	return nil
}

func validateTransitionBody(transition Transition) error {
	if transition.Schema != transitionSchema {
		return fmt.Errorf("schema must be %q, got %q", transitionSchema, transition.Schema)
	}
	if transition.Lane != laneRelease && transition.Lane != laneIncident {
		return fmt.Errorf("lane must be %q or %q, got %q", laneRelease, laneIncident, transition.Lane)
	}
	if transition.Sequence == 0 {
		return errors.New("sequence must be greater than zero")
	}
	if !knownStates[transition.From] {
		return fmt.Errorf("unknown from state %q", transition.From)
	}
	if !knownStates[transition.To] {
		return fmt.Errorf("unknown to state %q", transition.To)
	}
	if err := validateLabel("event", transition.Event, 128); err != nil {
		return err
	}
	if err := validateCanonicalTime(transition.OccurredAt); err != nil {
		return err
	}
	if err := validateLabel("chain_id", transition.ChainID, 128); err != nil {
		return err
	}
	if transition.IncidentID == "" && transition.ReleaseID == "" {
		return errors.New("at least one of incident_id or release_id must be set")
	}
	if transition.IncidentID != "" {
		if err := validateLabel("incident_id", transition.IncidentID, 128); err != nil {
			return err
		}
	}
	if transition.ReleaseID != "" {
		if err := validateLabel("release_id", transition.ReleaseID, 128); err != nil {
			return err
		}
	}
	if err := validateLabel("actor_role", transition.ActorRole, 128); err != nil {
		return err
	}
	if err := validateLabel("actor_identity", transition.ActorIdentity, 256); err != nil {
		return err
	}
	if err := validateCheckpoint(transition.Checkpoint); err != nil {
		return err
	}
	if err := validateReleaseBinding(transition.ReleaseID, transition.Release); err != nil {
		return err
	}
	if err := validatePowerSnapshotShape(transition.PowerSnapshot); err != nil {
		return err
	}
	if err := validateEvidence(transition.Evidence); err != nil {
		return err
	}
	if err := validateRequiredEvidence(transition); err != nil {
		return err
	}
	if err := validateSHA256("trust_policy_sha256", transition.TrustPolicySHA256, false); err != nil {
		return err
	}
	if err := validateApprovals(transition); err != nil {
		return err
	}
	if transition.Sequence == 1 {
		if transition.PreviousTransitionSHA256 != "" {
			return errors.New("sequence 1 requires empty previous_transition_sha256")
		}
	} else if err := validateSHA256("previous_transition_sha256", transition.PreviousTransitionSHA256, false); err != nil {
		return err
	}
	return nil
}

func validateStandaloneEdge(transition Transition) error {
	if transition.Sequence == 1 {
		if transition.From != StateRunning {
			return fmt.Errorf("sequence 1 must begin at %s", StateRunning)
		}
		journalLane, err := laneForFirstTransition(transition)
		if err != nil {
			return err
		}
		if err := validateLaneIdentity(transition, journalLane); err != nil {
			return err
		}
		if err := validateLaneTransition(transition, journalLane); err != nil {
			return err
		}
		return validateLaneGates(transition, journalLane)
	}
	if err := validateLaneIdentity(transition, transition.Lane); err != nil {
		return err
	}
	if err := validateLaneTransition(transition, transition.Lane); err != nil {
		return err
	}
	return validateLaneGates(transition, transition.Lane)
}

func laneForFirstTransition(transition Transition) (lane, error) {
	expectedTo := StatePreparing
	if transition.Lane == laneIncident {
		expectedTo = StateAssessing
	}
	if transition.Lane != laneRelease && transition.Lane != laneIncident {
		return "", fmt.Errorf("unknown explicit journal lane %q", transition.Lane)
	}
	if transition.To != expectedTo {
		return "", fmt.Errorf(
			"first %s transition must be %s -> %s, got %s -> %s",
			transition.Lane,
			StateRunning,
			expectedTo,
			transition.From,
			transition.To,
		)
	}
	return transition.Lane, nil
}

func validateLaneTransition(transition Transition, journalLane lane) error {
	var allowed bool
	switch journalLane {
	case laneRelease:
		allowed = releaseTransitions[transition.From][transition.To]
	case laneIncident:
		allowed = incidentTransitions[transition.From][transition.To]
	default:
		return fmt.Errorf("unknown journal lane %q", journalLane)
	}
	if !allowed {
		return fmt.Errorf("%s journal transition %s -> %s is not allowed", journalLane, transition.From, transition.To)
	}
	return nil
}

func validateLaneIdentity(transition Transition, journalLane lane) error {
	switch journalLane {
	case laneRelease:
		if transition.ReleaseID == "" {
			return errors.New("release journal requires release_id")
		}
		if transition.IncidentID != "" {
			return errors.New("release journal requires empty incident_id")
		}
	case laneIncident:
		if transition.IncidentID == "" {
			return errors.New("incident journal requires incident_id")
		}
		if transition.ReleaseID != "" {
			return errors.New("incident journal requires empty release_id")
		}
	default:
		return fmt.Errorf("unknown journal lane %q", journalLane)
	}
	return nil
}

func validateLaneGates(transition Transition, journalLane lane) error {
	if journalLane == laneRelease {
		switch transition.To {
		case StatePreparing, StateReleaseFrozen, StateScheduled, StateStaged, StateCancelled:
			if transition.Release.UpgradeHeight != 0 &&
				transition.Checkpoint.Height >= transition.Release.UpgradeHeight {
				return fmt.Errorf(
					"pre-activation state %s requires checkpoint height below upgrade height %d, got %d",
					transition.To,
					transition.Release.UpgradeHeight,
					transition.Checkpoint.Height,
				)
			}
		}
		if releaseStateRequiresFrozenArtifacts(transition.To) {
			if transition.Release.PlanName == "" {
				return fmt.Errorf("release.plan_name is required by state %s", transition.To)
			}
			if transition.Release.UpgradeHeight == 0 {
				return fmt.Errorf("release.upgrade_height is required by state %s", transition.To)
			}
			if transition.Release.ActivationMode != activationModeCosmovisor &&
				transition.Release.ActivationMode != activationModeImmutableImage {
				return fmt.Errorf("release.activation_mode is required by state %s", transition.To)
			}
			required := []struct {
				name  string
				value string
			}{
				{name: "release.plan_info_sha256", value: transition.Release.PlanInfoSHA256},
				{name: "release.binary_sha256", value: transition.Release.BinarySHA256},
				{name: "release.provenance_sha256", value: transition.Release.ProvenanceSHA256},
				{name: "release.sbom_sha256", value: transition.Release.SBOMSHA256},
			}
			for _, artifact := range required {
				if artifact.value == "" {
					return fmt.Errorf("%s is required by state %s", artifact.name, transition.To)
				}
			}
			if transition.Release.ActivationMode == activationModeImmutableImage &&
				transition.Release.ImageSHA256 == "" {
				return fmt.Errorf(
					"release.image_sha256 is required by state %s for immutable-image activation",
					transition.To,
				)
			}
		}
		switch transition.To {
		case StateHaltedAtH, StateMigratingH:
			if transition.Release.UpgradeHeight == 0 || transition.Checkpoint.Height != transition.Release.UpgradeHeight-1 {
				return fmt.Errorf(
					"state %s requires checkpoint height H-1: checkpoint=%d upgrade_height=%d",
					transition.To,
					transition.Checkpoint.Height,
					transition.Release.UpgradeHeight,
				)
			}
		case StateObserving, StateAccepted:
			if transition.Checkpoint.Height < transition.Release.UpgradeHeight {
				return fmt.Errorf(
					"state %s requires checkpoint height at or above upgrade height %d, got %d",
					transition.To,
					transition.Release.UpgradeHeight,
					transition.Checkpoint.Height,
				)
			}
		}
	}
	if journalLane == laneIncident {
		switch transition.To {
		case StateRecoveryDesign, StateRecoveryReady, StateActivating, StateObserving, StateClosed, StateForkChoice:
			if transition.Checkpoint.Height == 0 {
				return fmt.Errorf("state %s requires a last-trusted checkpoint", transition.To)
			}
		}
	}
	return nil
}

func releaseStateRequiresFrozenArtifacts(state State) bool {
	switch state {
	case StateReleaseFrozen, StateScheduled, StateStaged, StateHaltedAtH,
		StateMigratingH, StateObserving, StateAccepted:
		return true
	default:
		return false
	}
}

func validateIdentityContinuity(previous, current Transition) error {
	if current.Lane != previous.Lane {
		return fmt.Errorf("lane changed from %q to %q", previous.Lane, current.Lane)
	}
	if current.ChainID != previous.ChainID {
		return fmt.Errorf("chain_id changed from %q to %q", previous.ChainID, current.ChainID)
	}
	if current.IncidentID != previous.IncidentID {
		return fmt.Errorf("incident_id changed from %q to %q", previous.IncidentID, current.IncidentID)
	}
	if current.ReleaseID != previous.ReleaseID {
		return fmt.Errorf("release_id changed from %q to %q", previous.ReleaseID, current.ReleaseID)
	}
	return nil
}

func validateReleaseContinuity(previous, current ReleaseBinding) error {
	if previous.PlanName != "" && current.PlanName != previous.PlanName {
		return fmt.Errorf("release.plan_name changed from %q to %q", previous.PlanName, current.PlanName)
	}
	if previous.UpgradeHeight != 0 && current.UpgradeHeight != previous.UpgradeHeight {
		return fmt.Errorf("release.upgrade_height changed from %d to %d", previous.UpgradeHeight, current.UpgradeHeight)
	}
	if previous.ActivationMode != "" && current.ActivationMode != previous.ActivationMode {
		return fmt.Errorf(
			"release.activation_mode changed after binding: %q -> %q",
			previous.ActivationMode,
			current.ActivationMode,
		)
	}
	bindings := []struct {
		name     string
		previous string
		current  string
	}{
		{name: "plan_info_sha256", previous: previous.PlanInfoSHA256, current: current.PlanInfoSHA256},
		{name: "binary_sha256", previous: previous.BinarySHA256, current: current.BinarySHA256},
		{name: "image_sha256", previous: previous.ImageSHA256, current: current.ImageSHA256},
		{name: "provenance_sha256", previous: previous.ProvenanceSHA256, current: current.ProvenanceSHA256},
		{name: "sbom_sha256", previous: previous.SBOMSHA256, current: current.SBOMSHA256},
		{name: "state_manifest_sha256", previous: previous.StateManifestSHA256, current: current.StateManifestSHA256},
	}
	for _, binding := range bindings {
		if binding.previous != "" && binding.current != binding.previous {
			return fmt.Errorf(
				"release.%s changed after binding: %q -> %q",
				binding.name,
				binding.previous,
				binding.current,
			)
		}
	}
	return nil
}

func validateCheckpointContinuity(previous, current Checkpoint) error {
	if current.Height < previous.Height {
		return fmt.Errorf("checkpoint height moved backward from %d to %d", previous.Height, current.Height)
	}
	if current.Height == previous.Height &&
		(current.BlockIDSHA256 != previous.BlockIDSHA256 || current.AppHashSHA256 != previous.AppHashSHA256) {
		return fmt.Errorf("checkpoint hashes changed at height %d", current.Height)
	}
	return nil
}

func validateCheckpoint(checkpoint Checkpoint) error {
	if checkpoint.Height == 0 {
		if checkpoint.BlockIDSHA256 != "" || checkpoint.AppHashSHA256 != "" {
			return errors.New("checkpoint height 0 requires empty block_id_sha256 and app_hash_sha256")
		}
		return nil
	}
	if err := validateSHA256("checkpoint.block_id_sha256", checkpoint.BlockIDSHA256, false); err != nil {
		return err
	}
	return validateSHA256("checkpoint.app_hash_sha256", checkpoint.AppHashSHA256, false)
}

func validateReleaseBinding(releaseID string, release ReleaseBinding) error {
	if releaseID == "" {
		if release != (ReleaseBinding{}) {
			return errors.New("release binding must be empty when release_id is empty")
		}
		return nil
	}
	if release.PlanName != "" {
		if err := validateLabel("release.plan_name", release.PlanName, 128); err != nil {
			return err
		}
	}
	if (release.PlanName == "") != (release.UpgradeHeight == 0) {
		return errors.New("release.plan_name and release.upgrade_height must be both set or both empty")
	}
	switch release.ActivationMode {
	case "", activationModeCosmovisor, activationModeImmutableImage:
	default:
		return fmt.Errorf(
			"release.activation_mode must be empty, %q, or %q",
			activationModeCosmovisor,
			activationModeImmutableImage,
		)
	}
	hashes := []struct {
		name  string
		value string
	}{
		{name: "release.plan_info_sha256", value: release.PlanInfoSHA256},
		{name: "release.binary_sha256", value: release.BinarySHA256},
		{name: "release.image_sha256", value: release.ImageSHA256},
		{name: "release.provenance_sha256", value: release.ProvenanceSHA256},
		{name: "release.sbom_sha256", value: release.SBOMSHA256},
		{name: "release.state_manifest_sha256", value: release.StateManifestSHA256},
	}
	for _, hash := range hashes {
		if hash.value != "" {
			if err := validateSHA256(hash.name, hash.value, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func emptyPowerSnapshot() PowerSnapshot {
	return PowerSnapshot{Members: []PowerSnapshotMember{}}
}

func isEmptyPowerSnapshot(snapshot PowerSnapshot) bool {
	return snapshot.Schema == "" &&
		snapshot.ChainID == "" &&
		snapshot.Height == 0 &&
		snapshot.BlockIDSHA256 == "" &&
		snapshot.AppHashSHA256 == "" &&
		snapshot.Role == "" &&
		snapshot.TotalPower == "" &&
		snapshot.CapturedAt == "" &&
		snapshot.ValidUntil == "" &&
		snapshot.Members != nil &&
		len(snapshot.Members) == 0 &&
		snapshot.SnapshotSHA256 == ""
}

func validatePowerSnapshotShape(snapshot PowerSnapshot) error {
	if snapshot.Schema == "" {
		if !isEmptyPowerSnapshot(snapshot) {
			return errors.New("empty power_snapshot requires every scalar field empty/zero and members=[]")
		}
		return nil
	}
	if snapshot.Schema != powerSnapshotSchema {
		return fmt.Errorf("power_snapshot.schema must be %q", powerSnapshotSchema)
	}
	if err := validateLabel("power_snapshot.chain_id", snapshot.ChainID, 128); err != nil {
		return err
	}
	if snapshot.Height == 0 {
		return errors.New("power_snapshot.height must be positive")
	}
	if err := validateSHA256("power_snapshot.block_id_sha256", snapshot.BlockIDSHA256, false); err != nil {
		return err
	}
	if err := validateSHA256("power_snapshot.app_hash_sha256", snapshot.AppHashSHA256, false); err != nil {
		return err
	}
	if err := validateLabel("power_snapshot.role", snapshot.Role, 128); err != nil {
		return err
	}
	total, err := parseCanonicalDecimal("power_snapshot.total_power", snapshot.TotalPower)
	if err != nil {
		return err
	}
	if total.Sign() <= 0 {
		return errors.New("power_snapshot.total_power must be positive")
	}
	if err := validateCanonicalTimeField("power_snapshot.captured_at", snapshot.CapturedAt); err != nil {
		return err
	}
	if err := validateCanonicalTimeField("power_snapshot.valid_until", snapshot.ValidUntil); err != nil {
		return err
	}
	capturedAt, _ := time.Parse(time.RFC3339Nano, snapshot.CapturedAt)
	validUntil, _ := time.Parse(time.RFC3339Nano, snapshot.ValidUntil)
	if !validUntil.After(capturedAt) {
		return errors.New("power_snapshot.valid_until must be after captured_at")
	}
	if snapshot.Members == nil {
		return errors.New("power_snapshot.members must be [] rather than null")
	}
	if len(snapshot.Members) == 0 {
		return errors.New("power_snapshot.members must contain at least one validator operator")
	}
	if !sort.SliceIsSorted(snapshot.Members, func(i, j int) bool {
		return powerSnapshotMemberLess(snapshot.Members[i], snapshot.Members[j])
	}) {
		return errors.New("power_snapshot.members must be sorted by identity, then public_key")
	}
	identityToKey := make(map[string]string, len(snapshot.Members))
	keyToIdentity := make(map[string]string, len(snapshot.Members))
	actualTotal := new(big.Int)
	for i, member := range snapshot.Members {
		prefix := fmt.Sprintf("power_snapshot.members[%d]", i)
		if err := validateLabel(prefix+".identity", member.Identity, 256); err != nil {
			return err
		}
		if _, err := decodeExactHex(prefix+".public_key", member.PublicKey, ed25519.PublicKeySize); err != nil {
			return err
		}
		power, err := parseCanonicalDecimal(prefix+".power", member.Power)
		if err != nil {
			return err
		}
		if power.Sign() <= 0 {
			return fmt.Errorf("%s.power must be positive", prefix)
		}
		if existing, found := identityToKey[member.Identity]; found {
			return fmt.Errorf(
				"power snapshot identity %q appears more than once (keys %s and %s)",
				member.Identity,
				existing,
				member.PublicKey,
			)
		}
		if existing, found := keyToIdentity[member.PublicKey]; found {
			return fmt.Errorf(
				"power snapshot public key %q appears more than once (identities %s and %s)",
				member.PublicKey,
				existing,
				member.Identity,
			)
		}
		identityToKey[member.Identity] = member.PublicKey
		keyToIdentity[member.PublicKey] = member.Identity
		actualTotal.Add(actualTotal, power)
	}
	if actualTotal.Cmp(total) != 0 {
		return fmt.Errorf(
			"power_snapshot member power %s does not equal total_power %s",
			actualTotal,
			total,
		)
	}
	if err := validateSHA256("power_snapshot.snapshot_sha256", snapshot.SnapshotSHA256, false); err != nil {
		return err
	}
	actualDigest, err := powerSnapshotHash(snapshot)
	if err != nil {
		return err
	}
	if snapshot.SnapshotSHA256 != actualDigest {
		return fmt.Errorf(
			"power_snapshot.snapshot_sha256 mismatch: have %s want %s",
			snapshot.SnapshotSHA256,
			actualDigest,
		)
	}
	return nil
}

func powerSnapshotHash(snapshot PowerSnapshot) (string, error) {
	snapshot.SnapshotSHA256 = ""
	canonical, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("canonicalize power snapshot: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func powerSnapshotMemberLess(a, b PowerSnapshotMember) bool {
	if a.Identity != b.Identity {
		return a.Identity < b.Identity
	}
	return a.PublicKey < b.PublicKey
}

func validateEvidence(evidence []Evidence) error {
	if evidence == nil {
		return errors.New("evidence must be [] rather than null")
	}
	if len(evidence) == 0 {
		return errors.New("at least one evidence item is required")
	}
	if !sort.SliceIsSorted(evidence, func(i, j int) bool {
		return evidenceLess(evidence[i], evidence[j])
	}) {
		return errors.New("evidence must be sorted by type, sha256, then uri")
	}
	seen := make(map[string]bool, len(evidence))
	for i, item := range evidence {
		if err := validateLabel(fmt.Sprintf("evidence[%d].type", i), item.Type, 128); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("evidence[%d].sha256", i), item.SHA256, false); err != nil {
			return err
		}
		if item.URI == "" || strings.TrimSpace(item.URI) != item.URI || len(item.URI) > 2048 {
			return fmt.Errorf("evidence[%d].uri must be non-empty, trimmed, and at most 2048 bytes", i)
		}
		key := item.Type + "\x00" + item.SHA256 + "\x00" + item.URI
		if seen[key] {
			return fmt.Errorf("evidence[%d] duplicates an earlier item", i)
		}
		seen[key] = true
	}
	return nil
}

func evidenceLess(a, b Evidence) bool {
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	if a.SHA256 != b.SHA256 {
		return a.SHA256 < b.SHA256
	}
	return a.URI < b.URI
}

func validateRequiredEvidence(transition Transition) error {
	required := requiredEvidenceTypes(transition.Lane, transition.To)
	present := make(map[string]bool, len(transition.Evidence))
	for _, item := range transition.Evidence {
		present[item.Type] = true
	}
	for _, evidenceType := range required {
		if !present[evidenceType] {
			return fmt.Errorf(
				"transition %s/%s->%s requires evidence type %q",
				transition.Lane,
				transition.From,
				transition.To,
				evidenceType,
			)
		}
	}
	return nil
}

func requiredEvidenceTypes(journalLane lane, to State) []string {
	if journalLane == laneRelease {
		switch to {
		case StatePreparing:
			return []string{"intent-record"}
		case StateReleaseFrozen:
			return []string{"release-manifest"}
		case StateScheduled:
			return []string{"on-chain-upgrade-plan"}
		case StateStaged:
			return []string{"activation-profile", "validator-power-snapshot", "validator-readiness-report"}
		case StateHaltedAtH:
			return []string{"h-minus-one-checkpoint"}
		case StateMigratingH:
			return []string{"migration-start-record"}
		case StateObserving:
			return []string{"post-upgrade-observation"}
		case StateAccepted:
			return []string{"acceptance-report"}
		case StateCancelled:
			return []string{"cancellation-record"}
		case StateRecoveryFailed:
			return []string{"failure-record"}
		}
	}
	if journalLane == laneIncident {
		switch to {
		case StateAssessing:
			return []string{"incident-assessment"}
		case StateContaining:
			return []string{"containment-record"}
		case StateSafetyStopped:
			return []string{"signer-stop-record"}
		case StateRecoveryDesign:
			return []string{"recovery-design"}
		case StateRecoveryReady:
			return []string{"recovery-manifest"}
		case StateActivating:
			return []string{"activation-record", "validator-power-snapshot"}
		case StateObserving:
			return []string{"recovery-observation"}
		case StateClosed:
			return []string{"closure-report"}
		case StateRecoveryFailed:
			return []string{"failure-record"}
		case StateForkChoice:
			return []string{"fork-choice-manifest", "validator-power-snapshot"}
		}
	}
	return nil
}

func validateApprovalPolicyShape(policy ApprovalPolicy) error {
	if policy.RequiredRoles == nil {
		return errors.New("approval_policy.required_roles must be [] rather than null")
	}
	if policy.SeparatedRolePairs == nil {
		return errors.New("approval_policy.separated_role_pairs must be [] rather than null")
	}
	if !sort.StringsAreSorted(policy.RequiredRoles) {
		return errors.New("approval_policy.required_roles must be sorted")
	}
	seenRoles := make(map[string]bool, len(policy.RequiredRoles))
	for i, role := range policy.RequiredRoles {
		if err := validateLabel(fmt.Sprintf("approval_policy.required_roles[%d]", i), role, 128); err != nil {
			return err
		}
		if seenRoles[role] {
			return fmt.Errorf("approval_policy.required_roles contains duplicate %q", role)
		}
		seenRoles[role] = true
	}

	if !sort.SliceIsSorted(policy.SeparatedRolePairs, func(i, j int) bool {
		a, b := policy.SeparatedRolePairs[i], policy.SeparatedRolePairs[j]
		if a.RoleA != b.RoleA {
			return a.RoleA < b.RoleA
		}
		return a.RoleB < b.RoleB
	}) {
		return errors.New("approval_policy.separated_role_pairs must be sorted")
	}
	seenPairs := make(map[string]bool, len(policy.SeparatedRolePairs))
	for i, pair := range policy.SeparatedRolePairs {
		if err := validateLabel(fmt.Sprintf("approval_policy.separated_role_pairs[%d].role_a", i), pair.RoleA, 128); err != nil {
			return err
		}
		if err := validateLabel(fmt.Sprintf("approval_policy.separated_role_pairs[%d].role_b", i), pair.RoleB, 128); err != nil {
			return err
		}
		if pair.RoleA >= pair.RoleB {
			return fmt.Errorf("approval_policy.separated_role_pairs[%d] requires role_a < role_b", i)
		}
		key := pair.RoleA + "\x00" + pair.RoleB
		if seenPairs[key] {
			return fmt.Errorf("approval_policy.separated_role_pairs contains duplicate %q/%q", pair.RoleA, pair.RoleB)
		}
		seenPairs[key] = true
	}

	quorum := policy.PowerQuorum
	if quorum.Role == "" {
		if quorum.Numerator != 0 ||
			quorum.Denominator != 0 ||
			quorum.Strict {
			return errors.New("disabled approval_policy.power_quorum must have zero threshold and strict=false")
		}
		return nil
	}
	if err := validateLabel("approval_policy.power_quorum.role", quorum.Role, 128); err != nil {
		return err
	}
	if quorum.Numerator == 0 || quorum.Denominator == 0 || quorum.Numerator > quorum.Denominator {
		return errors.New("approval_policy.power_quorum requires 0 < numerator <= denominator")
	}
	return nil
}

func validateApprovals(transition Transition) error {
	if transition.Approvals == nil {
		return errors.New("approvals must be [] rather than null")
	}
	if !sort.SliceIsSorted(transition.Approvals, func(i, j int) bool {
		return approvalLess(transition.Approvals[i], transition.Approvals[j])
	}) {
		return errors.New("approvals must be sorted by role, identity, then public_key")
	}
	identityToKey := make(map[string]string)
	keyToIdentity := make(map[string]string)
	identityRoles := make(map[string]map[string]bool)
	keyRoles := make(map[string]map[string]bool)
	seenIdentityRole := make(map[string]bool)
	seenKeyRole := make(map[string]bool)
	roleCounts := make(map[string]int)

	for i, approval := range transition.Approvals {
		prefix := fmt.Sprintf("approvals[%d]", i)
		if err := validateLabel(prefix+".role", approval.Role, 128); err != nil {
			return err
		}
		if err := validateLabel(prefix+".identity", approval.Identity, 256); err != nil {
			return err
		}
		publicKey, err := decodeExactHex(prefix+".public_key", approval.PublicKey, ed25519.PublicKeySize)
		if err != nil {
			return err
		}
		signature, err := decodeExactHex(prefix+".signature", approval.Signature, ed25519.SignatureSize)
		if err != nil {
			return err
		}
		statementDigest, err := approvalStatementDigest(transition, approval)
		if err != nil {
			return fmt.Errorf("compute %s approval statement: %w", prefix, err)
		}
		statementHex := hex.EncodeToString(statementDigest[:])
		if err := validateSHA256(prefix+".statement_sha256", approval.StatementSHA256, false); err != nil {
			return err
		}
		if approval.StatementSHA256 != statementHex {
			return fmt.Errorf("%s.statement_sha256 does not bind this transition", prefix)
		}
		if !ed25519.Verify(ed25519.PublicKey(publicKey), statementDigest[:], signature) {
			return fmt.Errorf("%s.signature is not a valid Ed25519 signature for the approval statement", prefix)
		}
		if _, err := parseCanonicalDecimal(prefix+".power", approval.Power); err != nil {
			return err
		}

		if existing, ok := identityToKey[approval.Identity]; ok && existing != approval.PublicKey {
			return fmt.Errorf("identity %q uses multiple public keys", approval.Identity)
		}
		if existing, ok := keyToIdentity[approval.PublicKey]; ok && existing != approval.Identity {
			return fmt.Errorf("public key %q uses multiple identities", approval.PublicKey)
		}
		identityToKey[approval.Identity] = approval.PublicKey
		keyToIdentity[approval.PublicKey] = approval.Identity

		identityRole := approval.Identity + "\x00" + approval.Role
		if seenIdentityRole[identityRole] {
			return fmt.Errorf("identity %q has duplicate approval for role %q", approval.Identity, approval.Role)
		}
		seenIdentityRole[identityRole] = true
		keyRole := approval.PublicKey + "\x00" + approval.Role
		if seenKeyRole[keyRole] {
			return fmt.Errorf("public key %q has duplicate approval for role %q", approval.PublicKey, approval.Role)
		}
		seenKeyRole[keyRole] = true

		if identityRoles[approval.Identity] == nil {
			identityRoles[approval.Identity] = make(map[string]bool)
		}
		identityRoles[approval.Identity][approval.Role] = true
		if keyRoles[approval.PublicKey] == nil {
			keyRoles[approval.PublicKey] = make(map[string]bool)
		}
		keyRoles[approval.PublicKey][approval.Role] = true
		roleCounts[approval.Role]++
	}

	return nil
}

func validateApprovalQuorum(
	approvals []Approval,
	policy ApprovalPolicy,
	snapshot PowerSnapshot,
) error {
	identityToKey := make(map[string]string)
	identityRoles := make(map[string]map[string]bool)
	keyRoles := make(map[string]map[string]bool)
	roleCounts := make(map[string]int)
	for _, approval := range approvals {
		identityToKey[approval.Identity] = approval.PublicKey
		if identityRoles[approval.Identity] == nil {
			identityRoles[approval.Identity] = make(map[string]bool)
		}
		identityRoles[approval.Identity][approval.Role] = true
		if keyRoles[approval.PublicKey] == nil {
			keyRoles[approval.PublicKey] = make(map[string]bool)
		}
		keyRoles[approval.PublicKey][approval.Role] = true
		roleCounts[approval.Role]++
	}

	if uint64(len(approvals)) < policy.MinimumApprovals {
		return fmt.Errorf(
			"approval quorum not met: have %d approvals, require %d",
			len(approvals),
			policy.MinimumApprovals,
		)
	}
	if uint64(len(identityToKey)) < policy.MinimumDistinctIdentities {
		return fmt.Errorf(
			"identity quorum not met: have %d distinct identities, require %d",
			len(identityToKey),
			policy.MinimumDistinctIdentities,
		)
	}
	for _, role := range policy.RequiredRoles {
		if roleCounts[role] == 0 {
			return fmt.Errorf("required approval role %q is missing", role)
		}
	}
	for _, pair := range policy.SeparatedRolePairs {
		for identity, roles := range identityRoles {
			if roles[pair.RoleA] && roles[pair.RoleB] {
				return fmt.Errorf("identity %q fills separated roles %q and %q", identity, pair.RoleA, pair.RoleB)
			}
		}
		for publicKey, roles := range keyRoles {
			if roles[pair.RoleA] && roles[pair.RoleB] {
				return fmt.Errorf("public key %q fills separated roles %q and %q", publicKey, pair.RoleA, pair.RoleB)
			}
		}
	}
	return validatePowerQuorum(approvals, policy.PowerQuorum, snapshot)
}

func validatePowerQuorum(
	approvals []Approval,
	quorum PowerQuorum,
	snapshot PowerSnapshot,
) error {
	if quorum.Role == "" {
		for i, approval := range approvals {
			if approval.Power != "0" {
				return fmt.Errorf("approvals[%d].power must be \"0\" when no power quorum is declared", i)
			}
		}
		return nil
	}
	total, err := parseCanonicalDecimal("power_snapshot.total_power", snapshot.TotalPower)
	if err != nil {
		return err
	}
	memberPower := make(map[string]string, len(snapshot.Members))
	for _, member := range snapshot.Members {
		memberPower[trustedSignerKey(
			snapshot.Role,
			member.Identity,
			member.PublicKey,
		)] = member.Power
	}
	approved := new(big.Int)
	for i, approval := range approvals {
		power, err := parseCanonicalDecimal(fmt.Sprintf("approvals[%d].power", i), approval.Power)
		if err != nil {
			return err
		}
		if approval.Role != quorum.Role {
			if power.Sign() != 0 {
				return fmt.Errorf(
					"approvals[%d].power must be \"0\": role %q is outside power quorum role %q",
					i,
					approval.Role,
					quorum.Role,
				)
			}
			continue
		}
		if power.Sign() <= 0 {
			return fmt.Errorf("approvals[%d].power must be positive for quorum role %q", i, quorum.Role)
		}
		snapshotPower, found := memberPower[trustedSignerKey(
			approval.Role,
			approval.Identity,
			approval.PublicKey,
		)]
		if !found {
			return fmt.Errorf(
				"approvals[%d] is absent from the transition power snapshot",
				i,
			)
		}
		if approval.Power != snapshotPower {
			return fmt.Errorf(
				"approvals[%d].power %s does not match transition snapshot power %s",
				i,
				approval.Power,
				snapshotPower,
			)
		}
		approved.Add(approved, power)
	}
	if approved.Cmp(total) > 0 {
		return fmt.Errorf("approved power %s exceeds declared total power %s", approved, total)
	}
	left := new(big.Int).Mul(new(big.Int).Set(approved), new(big.Int).SetUint64(quorum.Denominator))
	right := new(big.Int).Mul(new(big.Int).Set(total), new(big.Int).SetUint64(quorum.Numerator))
	comparison := left.Cmp(right)
	if comparison < 0 || (quorum.Strict && comparison == 0) {
		comparator := ">="
		if quorum.Strict {
			comparator = ">"
		}
		return fmt.Errorf(
			"power quorum not met: approved=%s total=%s require %s %d/%d",
			approved,
			total,
			comparator,
			quorum.Numerator,
			quorum.Denominator,
		)
	}
	return nil
}

func approvalLess(a, b Approval) bool {
	if a.Role != b.Role {
		return a.Role < b.Role
	}
	if a.Identity != b.Identity {
		return a.Identity < b.Identity
	}
	return a.PublicKey < b.PublicKey
}

func validateSHA256(name, value string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	_, err := decodeExactHex(name, value, sha256.Size)
	return err
}

func decodeExactHex(name, value string, byteLength int) ([]byte, error) {
	if len(value) != byteLength*2 {
		return nil, fmt.Errorf("%s must be exactly %d lowercase hexadecimal characters", name, byteLength*2)
	}
	if value != strings.ToLower(value) {
		return nil, fmt.Errorf("%s must use lowercase hexadecimal", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be lowercase hexadecimal: %w", name, err)
	}
	return decoded, nil
}

func parseCanonicalDecimal(name, value string) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("%s must be a canonical non-negative decimal", name)
	}
	if value != "0" && value[0] == '0' {
		return nil, fmt.Errorf("%s must not contain leading zeroes", name)
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return nil, fmt.Errorf("%s must be a canonical non-negative decimal", name)
		}
	}
	number, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("%s must be a canonical non-negative decimal", name)
	}
	return number, nil
}

func validateLabel(name, value string, maxLength int) error {
	if value == "" || strings.TrimSpace(value) != value || len(value) > maxLength {
		return fmt.Errorf("%s must be non-empty, trimmed, and at most %d bytes", name, maxLength)
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return fmt.Errorf("%s must not contain whitespace or control characters", name)
		}
	}
	return nil
}

func validateCanonicalTime(value string) error {
	return validateCanonicalTimeField("occurred_at", value)
}

func validateCanonicalTimeField(name, value string) error {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return fmt.Errorf("%s must be RFC3339: %w", name, err)
	}
	if parsed.UTC().Format(time.RFC3339Nano) != value {
		return fmt.Errorf("%s must be canonical UTC RFC3339Nano using Z", name)
	}
	return nil
}
