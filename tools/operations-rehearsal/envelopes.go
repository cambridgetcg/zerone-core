package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
)

const evidenceEnvelopeSchemaPrefix = "zerone.operations-evidence/"

type EvidenceCollector struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	BinarySHA256 string `json:"binary_sha256"`
}

type EvidenceExecution struct {
	InvocationID  string `json:"invocation_id"`
	RunnerID      string `json:"runner_id"`
	ProcessID     int64  `json:"process_id"`
	CommandSHA256 string `json:"command_sha256"`
	StartedAt     string `json:"started_at"`
	CompletedAt   string `json:"completed_at"`
	ExitCode      int64  `json:"exit_code"`
	TimedOut      bool   `json:"timed_out"`
	Observed      bool   `json:"observed"`
}

type EvidenceSubject struct {
	RunID   string `json:"run_id"`
	ChainID string `json:"chain_id"`
}

type EnvelopeArtifact struct {
	Role      string `json:"role"`
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type EvidenceEnvelope struct {
	Schema         string             `json:"schema"`
	Kind           string             `json:"kind"`
	Outcome        string             `json:"outcome"`
	CollectedAt    string             `json:"collected_at"`
	Collector      EvidenceCollector  `json:"collector"`
	Execution      EvidenceExecution  `json:"execution"`
	Subject        EvidenceSubject    `json:"subject"`
	Observations   json.RawMessage    `json:"observations"`
	Artifacts      []EnvelopeArtifact `json:"artifacts"`
	EnvelopeSHA256 string             `json:"envelope_sha256"`
}

type BinaryBuildObservation struct {
	Role         string `json:"role"`
	Revision     string `json:"revision"`
	BinarySHA256 string `json:"binary_sha256"`
	Version      string `json:"version"`
}

type PlanObservation struct {
	PlanName       string `json:"plan_name"`
	UpgradeHeight  int64  `json:"upgrade_height"`
	PlanInfoSHA256 string `json:"plan_info_sha256"`
	Status         string `json:"status"`
}

type PlanInfoObservation struct {
	PlanName       string `json:"plan_name"`
	UpgradeHeight  int64  `json:"upgrade_height"`
	PlanInfoSHA256 string `json:"plan_info_sha256"`
	Schema         string `json:"schema"`
}

type ActivationPreflightObservation struct {
	ReportSchema            string `json:"report_schema"`
	Height                  int64  `json:"height"`
	AppHash                 string `json:"app_hash"`
	ReportSHA256            string `json:"report_sha256"`
	H2PlanIdentitySHA256    string `json:"h2_plan_identity_sha256"`
	GateSatisfied           bool   `json:"gate_satisfied"`
	SourceDatabaseUnchanged bool   `json:"source_database_unchanged"`
	SourceVersionsExact     bool   `json:"source_versions_exact"`
	PlanInfoExact           bool   `json:"plan_info_exact"`
}

type OldExitObservation struct {
	UpgradeHeight         int64 `json:"upgrade_height"`
	LastCommittedHeight   int64 `json:"last_committed_height"`
	ProcessExited         bool  `json:"process_exited"`
	UpgradeNeededLogCount int64 `json:"upgrade_needed_log_count"`
	DatabaseUnchanged     bool  `json:"database_unchanged"`
}

type UpgradeInfoObservation struct {
	PlanName       string `json:"plan_name"`
	UpgradeHeight  int64  `json:"upgrade_height"`
	PlanInfoSHA256 string `json:"plan_info_sha256"`
}

type ReplayObservation struct {
	CloneID           string `json:"clone_id"`
	UpgradeHeight     int64  `json:"upgrade_height"`
	AppHash           string `json:"app_hash"`
	AppliedPlanHeight int64  `json:"applied_plan_height"`
	TargetCommitCount int64  `json:"target_commit_count"`
}

type PostUpgradeObservation struct {
	UpgradeHeight       int64  `json:"upgrade_height"`
	AppHash             string `json:"app_hash"`
	AppliedPlanHeight   int64  `json:"applied_plan_height"`
	RestartObserved     bool   `json:"restart_observed"`
	StateDeltasVerified bool   `json:"state_deltas_verified"`
}

type HandoffObservation struct {
	UpgradeHeight     int64 `json:"upgrade_height"`
	OldStopped        bool  `json:"old_stopped"`
	TargetStarted     bool  `json:"target_started"`
	TargetCommitCount int64 `json:"target_commit_count"`
	FailStopArmed     bool  `json:"fail_stop_armed"`
}

type HaltStatusObservation struct {
	HaltHeight       int64 `json:"halt_height"`
	QuarantineActive bool  `json:"quarantine_active"`
}

type HeightAdvanceObservation struct {
	FromHeight      int64 `json:"from_height"`
	ToHeight        int64 `json:"to_height"`
	BlocksContinued bool  `json:"blocks_continued"`
}

type TransactionRejectionObservation struct {
	Height              int64 `json:"height"`
	OrdinaryRejected    bool  `json:"ordinary_rejected"`
	WrappedRejected     bool  `json:"wrapped_rejected"`
	ICACallbackRejected bool  `json:"ica_callback_rejected"`
}

type QuarantineAuditObservation struct {
	HaltHeight        int64  `json:"halt_height"`
	ObservedHeight    int64  `json:"observed_height"`
	NoAutomaticResume bool   `json:"no_automatic_resume"`
	AuditEntrySHA256  string `json:"audit_entry_sha256"`
}

type AuthorizationObservation struct {
	ProposalID     int64  `json:"proposal_id"`
	Submitter      string `json:"submitter"`
	ActionSHA256   string `json:"action_sha256"`
	PlanSHA256     string `json:"plan_sha256"`
	ManifestSHA256 string `json:"manifest_sha256"`
	Authorized     bool   `json:"authorized"`
}

type WrongTupleObservation struct {
	ProposalID int64  `json:"proposal_id"`
	Rejected   bool   `json:"rejected"`
	ErrorCode  string `json:"error_code"`
}

type RevocationObservation struct {
	ProposalID              int64 `json:"proposal_id"`
	AuthorizationGeneration int64 `json:"authorization_generation"`
	Revoked                 bool  `json:"revoked"`
}

type ProposalFailureObservation struct {
	ProposalID    int64 `json:"proposal_id"`
	Failed        bool  `json:"failed"`
	Dequeued      bool  `json:"dequeued"`
	CannotExecute bool  `json:"cannot_execute"`
}

type ProposalRefundObservation struct {
	ProposalID int64  `json:"proposal_id"`
	Refunded   bool   `json:"refunded"`
	Amount     string `json:"amount"`
	Denom      string `json:"denom"`
}

type ResumeHeadObservation struct {
	HeadSHA256 string `json:"head_sha256"`
	Verified   bool   `json:"verified"`
}

type LatchObservation struct {
	Height   int64 `json:"height"`
	Rejected bool  `json:"rejected"`
	Accepted bool  `json:"accepted"`
}

type SourceProcessObservation struct {
	ProcessID                        int64  `json:"process_id"`
	ProcessStartTime                 string `json:"process_start_time"`
	ProcessIdentitySHA256            string `json:"process_identity_sha256"`
	RestartInhibitEvidenceSHA256     string `json:"restart_inhibit_evidence_sha256"`
	Height                           int64  `json:"height"`
	AppHash                          string `json:"app_hash"`
	LocalPIDAbsent                   bool   `json:"local_pid_absent"`
	ExternalRestartAssertionVerified bool   `json:"external_restart_assertion_verified"`
}

type DestinationVerificationObservation struct {
	HomeManifestSHA256                string `json:"home_manifest_sha256"`
	DestinationVolumeID               string `json:"destination_volume_id"`
	DestinationDeviceID               uint64 `json:"destination_device_id"`
	DestinationRootInode              uint64 `json:"destination_root_inode"`
	VolumeEvidenceSHA256              string `json:"volume_evidence_sha256"`
	SnapshotEvidenceSHA256            string `json:"snapshot_evidence_sha256"`
	Height                            int64  `json:"height"`
	AppHash                           string `json:"app_hash"`
	LocalManifestVerified             bool   `json:"local_manifest_verified"`
	ExternalControlAssertionsVerified bool   `json:"external_control_assertions_verified"`
	DatabaseGroupsVerified            bool   `json:"database_groups_verified"`
	SigningStateVerified              bool   `json:"signing_state_verified"`
	GenesisIdentityVerified           bool   `json:"genesis_identity_verified"`
	AuthorizedDeltaOnly               bool   `json:"authorized_delta_only"`
}

type DestinationStartObservation struct {
	StartHeight          int64 `json:"start_height"`
	FirstCommittedHeight int64 `json:"first_committed_height"`
	NoConcurrentSigner   bool  `json:"no_concurrent_signer"`
}

type ObserverIdentityObservation struct {
	ObserverID    string `json:"observer_id"`
	Independent   bool   `json:"independent"`
	Authenticated bool   `json:"authenticated"`
}

type ObserverPlanObservation struct {
	PlanName      string `json:"plan_name"`
	PlanHeight    int64  `json:"plan_height"`
	PlanConfirmed bool   `json:"plan_confirmed"`
}

type ObserverStatusObservation struct {
	ChainID           string `json:"chain_id"`
	Height            int64  `json:"height"`
	AppHash           string `json:"app_hash"`
	ImmutableSnapshot bool   `json:"immutable_snapshot"`
}

type FaultInjectionObservation struct {
	FaultID        string `json:"fault_id"`
	ReasonCode     string `json:"reason_code"`
	Phase          string `json:"phase"`
	InjectionPoint string `json:"injection_point"`
	Armed          bool   `json:"armed"`
}

type FaultResultObservation struct {
	FaultID              string `json:"fault_id"`
	ExpectedResult       string `json:"expected_result"`
	ActualResult         string `json:"actual_result"`
	ContainmentConfirmed bool   `json:"containment_confirmed"`
	Result               string `json:"result"`
}

type MutationCensusObservation struct {
	FaultID                   string `json:"fault_id"`
	LastCommittedHeight       int64  `json:"last_committed_height"`
	ForbiddenMutationObserved bool   `json:"forbidden_mutation_observed"`
	SourceStateUnchanged      bool   `json:"source_state_unchanged"`
}

type TypedEnvelope struct {
	Envelope    EvidenceEnvelope
	Observation any
}

func evidenceEnvelopeSchema(kind string) string {
	return evidenceEnvelopeSchemaPrefix + kind + "/v1"
}

func decodeEvidenceEnvelope(data []byte, expectedKind string, final bool) (TypedEnvelope, error) {
	var envelope EvidenceEnvelope
	if err := decodeStrict(data, &envelope); err != nil {
		return TypedEnvelope{}, err
	}
	if err := validateEvidenceEnvelope(envelope, expectedKind, final); err != nil {
		return TypedEnvelope{}, err
	}
	observation, err := decodeTypedObservation(expectedKind, envelope.Observations)
	if err != nil {
		return TypedEnvelope{}, err
	}
	if final {
		forHash := envelope
		forHash.EnvelopeSHA256 = ""
		digest, err := hashCanonical(forHash)
		if err != nil {
			return TypedEnvelope{}, err
		}
		if digest != envelope.EnvelopeSHA256 {
			return TypedEnvelope{}, fmt.Errorf("evidence envelope self-hash mismatch")
		}
		canonical, err := canonicalDocument(envelope)
		if err != nil {
			return TypedEnvelope{}, err
		}
		if !bytes.Equal(data, canonical) {
			return TypedEnvelope{}, fmt.Errorf("evidence envelope is not exact canonical JSON")
		}
	}
	return TypedEnvelope{Envelope: envelope, Observation: observation}, nil
}

func validateEvidenceEnvelope(envelope EvidenceEnvelope, expectedKind string, final bool) error {
	if expectedKind == "" || envelope.Kind != expectedKind {
		return fmt.Errorf("evidence envelope kind must be %q", expectedKind)
	}
	if envelope.Schema != evidenceEnvelopeSchema(expectedKind) {
		return fmt.Errorf("evidence envelope schema must be %q", evidenceEnvelopeSchema(expectedKind))
	}
	if envelope.Outcome != "observed" {
		return fmt.Errorf("evidence envelope outcome must be observed")
	}
	if _, err := validateCanonicalTime("evidence collected_at", envelope.CollectedAt); err != nil {
		return err
	}
	if envelope.Collector.Name == "" || strings.TrimSpace(envelope.Collector.Name) != envelope.Collector.Name ||
		envelope.Collector.Version == "" || strings.TrimSpace(envelope.Collector.Version) != envelope.Collector.Version {
		return fmt.Errorf("evidence collector name and version must be non-empty and trimmed")
	}
	if err := validateSHA256("collector binary SHA-256", envelope.Collector.BinarySHA256); err != nil {
		return err
	}
	if !identifierPattern.MatchString(envelope.Execution.InvocationID) ||
		envelope.Execution.RunnerID == "" ||
		strings.TrimSpace(envelope.Execution.RunnerID) != envelope.Execution.RunnerID ||
		envelope.Execution.ProcessID <= 1 || !envelope.Execution.Observed {
		return fmt.Errorf("evidence execution identity and observed process are required")
	}
	if err := validateSHA256("execution command SHA-256", envelope.Execution.CommandSHA256); err != nil {
		return err
	}
	started, err := validateCanonicalTime("evidence execution started_at", envelope.Execution.StartedAt)
	if err != nil {
		return err
	}
	completed, err := validateCanonicalTime("evidence execution completed_at", envelope.Execution.CompletedAt)
	if err != nil {
		return err
	}
	if completed.Before(started) || envelope.CollectedAt != envelope.Execution.CompletedAt {
		return fmt.Errorf("evidence execution timestamps are inconsistent")
	}
	if envelope.Execution.ExitCode != 0 || envelope.Execution.TimedOut {
		return fmt.Errorf("evidence collector execution must exit zero without timing out")
	}
	if !identifierPattern.MatchString(envelope.Subject.RunID) ||
		envelope.Subject.ChainID == "" ||
		strings.TrimSpace(envelope.Subject.ChainID) != envelope.Subject.ChainID {
		return fmt.Errorf("evidence subject run and chain identities are required")
	}
	if len(envelope.Observations) == 0 {
		return fmt.Errorf("typed observations are required")
	}
	if len(envelope.Artifacts) == 0 {
		return fmt.Errorf("at least one raw execution artifact is required")
	}
	previous := ""
	seenRoles := make(map[string]struct{}, len(envelope.Artifacts))
	for _, artifact := range envelope.Artifacts {
		if artifact.Role == "" || strings.TrimSpace(artifact.Role) != artifact.Role {
			return fmt.Errorf("envelope artifact role must be non-empty and trimmed")
		}
		key := artifact.Role + "\x00" + artifact.Path
		if previous != "" && key <= previous {
			return fmt.Errorf("envelope artifacts are not in canonical role/path order")
		}
		previous = key
		if _, duplicate := seenRoles[artifact.Role]; duplicate {
			return fmt.Errorf("envelope artifact role %q is duplicated", artifact.Role)
		}
		seenRoles[artifact.Role] = struct{}{}
		if err := validateEvidenceRefShape(EvidenceRef{
			Kind:      "raw-" + artifact.Role,
			Path:      artifact.Path,
			MediaType: artifact.MediaType,
			SizeBytes: artifact.SizeBytes,
			SHA256:    artifact.SHA256,
		}); err != nil {
			return err
		}
	}
	if final {
		if err := validateSHA256("evidence envelope SHA-256", envelope.EnvelopeSHA256); err != nil {
			return err
		}
	} else if envelope.EnvelopeSHA256 != "" {
		return fmt.Errorf("draft evidence envelope self-hash must be empty")
	}
	return nil
}

func decodeTypedObservation(kind string, raw json.RawMessage) (any, error) {
	switch kind {
	case "old-binary-build", "target-binary-build":
		return decodeObservation[BinaryBuildObservation](kind, raw)
	case "plan":
		return decodeObservation[PlanObservation](kind, raw)
	case "plan-info":
		return decodeObservation[PlanInfoObservation](kind, raw)
	case "activation-preflight":
		return decodeObservation[ActivationPreflightObservation](kind, raw)
	case "old-exit":
		return decodeObservation[OldExitObservation](kind, raw)
	case "upgrade-info":
		return decodeObservation[UpgradeInfoObservation](kind, raw)
	case "replay-a", "replay-b":
		return decodeObservation[ReplayObservation](kind, raw)
	case "post-upgrade":
		return decodeObservation[PostUpgradeObservation](kind, raw)
	case "handoff-result":
		return decodeObservation[HandoffObservation](kind, raw)
	case "halt-status":
		return decodeObservation[HaltStatusObservation](kind, raw)
	case "height-advance":
		return decodeObservation[HeightAdvanceObservation](kind, raw)
	case "ordinary-tx-rejection":
		return decodeObservation[TransactionRejectionObservation](kind, raw)
	case "quarantine-audit":
		return decodeObservation[QuarantineAuditObservation](kind, raw)
	case "authorization":
		return decodeObservation[AuthorizationObservation](kind, raw)
	case "wrong-tuple-rejection":
		return decodeObservation[WrongTupleObservation](kind, raw)
	case "revocation":
		return decodeObservation[RevocationObservation](kind, raw)
	case "proposal-failure":
		return decodeObservation[ProposalFailureObservation](kind, raw)
	case "proposal-refund":
		return decodeObservation[ProposalRefundObservation](kind, raw)
	case "resume-head":
		return decodeObservation[ResumeHeadObservation](kind, raw)
	case "same-block-rejection", "next-block-admission":
		return decodeObservation[LatchObservation](kind, raw)
	case "source-process-absence":
		return decodeObservation[SourceProcessObservation](kind, raw)
	case "destination-home-verification":
		return decodeObservation[DestinationVerificationObservation](kind, raw)
	case "destination-start":
		return decodeObservation[DestinationStartObservation](kind, raw)
	case "observer-identity":
		return decodeObservation[ObserverIdentityObservation](kind, raw)
	case "observer-plan":
		return decodeObservation[ObserverPlanObservation](kind, raw)
	case "observer-status":
		return decodeObservation[ObserverStatusObservation](kind, raw)
	case "fault-injection":
		return decodeObservation[FaultInjectionObservation](kind, raw)
	case "fault-result":
		return decodeObservation[FaultResultObservation](kind, raw)
	case "mutation-census":
		return decodeObservation[MutationCensusObservation](kind, raw)
	default:
		return nil, fmt.Errorf("evidence kind %q has no typed observation schema", kind)
	}
}

func decodeObservation[T any](kind string, raw json.RawMessage) (any, error) {
	var observation T
	if err := decodeStrict(raw, &observation); err != nil {
		return nil, fmt.Errorf("%s observations: %w", kind, err)
	}
	canonical, err := canonicalJSON(observation)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(raw, canonical) {
		return nil, fmt.Errorf("%s observations are not exact canonical JSON", kind)
	}
	if err := validateObservation(kind, any(observation)); err != nil {
		return nil, err
	}
	return observation, nil
}

func validateObservation(kind string, observation any) error {
	positive := func(label string, value int64) error {
		if value <= 0 {
			return fmt.Errorf("%s observations require positive %s", kind, label)
		}
		return nil
	}
	nonEmpty := func(label, value string) error {
		if value == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%s observations require non-empty %s", kind, label)
		}
		return nil
	}
	switch value := observation.(type) {
	case BinaryBuildObservation:
		if value.Role != "source" && value.Role != "target" {
			return fmt.Errorf("%s observations require source or target role", kind)
		}
		if err := nonEmpty("revision", value.Revision); err != nil {
			return err
		}
		if err := nonEmpty("version", value.Version); err != nil {
			return err
		}
		return validateSHA256("observed binary SHA-256", value.BinarySHA256)
	case PlanObservation:
		if err := nonEmpty("plan_name", value.PlanName); err != nil {
			return err
		}
		if err := positive("upgrade_height", value.UpgradeHeight); err != nil {
			return err
		}
		if value.Status != "scheduled" {
			return fmt.Errorf("plan observations require status scheduled")
		}
		return validateSHA256("observed Plan.Info SHA-256", value.PlanInfoSHA256)
	case PlanInfoObservation:
		if err := nonEmpty("plan_name", value.PlanName); err != nil {
			return err
		}
		if err := positive("upgrade_height", value.UpgradeHeight); err != nil {
			return err
		}
		if err := nonEmpty("Plan.Info schema", value.Schema); err != nil {
			return err
		}
		return validateSHA256("observed Plan.Info SHA-256", value.PlanInfoSHA256)
	case ActivationPreflightObservation:
		if value.ReportSchema != "zerone.activation-preflight/v5" || !value.GateSatisfied ||
			!value.SourceDatabaseUnchanged || !value.SourceVersionsExact ||
			!value.PlanInfoExact {
			return fmt.Errorf("activation-preflight observations do not prove the v5 read-only gate")
		}
		if err := positive("height", value.Height); err != nil {
			return err
		}
		if err := validateSHA256("activation AppHash", value.AppHash); err != nil {
			return err
		}
		if err := validateSHA256("H2 plan identity SHA-256", value.H2PlanIdentitySHA256); err != nil {
			return err
		}
		return validateSHA256("activation report SHA-256", value.ReportSHA256)
	case OldExitObservation:
		if value.UpgradeHeight <= 1 || value.LastCommittedHeight != value.UpgradeHeight-1 ||
			!value.ProcessExited || value.UpgradeNeededLogCount != 1 || !value.DatabaseUnchanged {
			return fmt.Errorf("old-exit observations do not establish exact H-1 stop")
		}
	case UpgradeInfoObservation:
		if err := nonEmpty("plan_name", value.PlanName); err != nil {
			return err
		}
		if err := positive("upgrade_height", value.UpgradeHeight); err != nil {
			return err
		}
		return validateSHA256("upgrade-info Plan.Info SHA-256", value.PlanInfoSHA256)
	case ReplayObservation:
		if err := nonEmpty("clone_id", value.CloneID); err != nil {
			return err
		}
		if value.UpgradeHeight <= 1 || value.AppliedPlanHeight != value.UpgradeHeight ||
			value.TargetCommitCount != 1 {
			return fmt.Errorf("%s observations do not establish one H commit", kind)
		}
		return validateSHA256("replay AppHash", value.AppHash)
	case PostUpgradeObservation:
		if value.UpgradeHeight <= 1 || value.AppliedPlanHeight != value.UpgradeHeight ||
			!value.RestartObserved || !value.StateDeltasVerified {
			return fmt.Errorf("post-upgrade observations do not satisfy restart/state gates")
		}
		return validateSHA256("post-upgrade AppHash", value.AppHash)
	case HandoffObservation:
		if value.UpgradeHeight <= 1 || !value.OldStopped || !value.TargetStarted ||
			value.TargetCommitCount != 1 || !value.FailStopArmed {
			return fmt.Errorf("handoff observations do not satisfy exact-height gates")
		}
	case HaltStatusObservation:
		if value.HaltHeight <= 0 || !value.QuarantineActive {
			return fmt.Errorf("halt-status observations do not establish active quarantine")
		}
	case HeightAdvanceObservation:
		if value.FromHeight <= 0 || value.ToHeight <= value.FromHeight || !value.BlocksContinued {
			return fmt.Errorf("height-advance observations do not establish continuing blocks")
		}
	case TransactionRejectionObservation:
		if value.Height <= 0 || !value.OrdinaryRejected || !value.WrappedRejected ||
			!value.ICACallbackRejected {
			return fmt.Errorf("transaction-rejection observations are incomplete")
		}
	case QuarantineAuditObservation:
		if value.HaltHeight <= 0 || value.ObservedHeight <= value.HaltHeight ||
			!value.NoAutomaticResume {
			return fmt.Errorf("quarantine-audit observations do not establish no auto-resume")
		}
		return validateSHA256("quarantine audit entry SHA-256", value.AuditEntrySHA256)
	case AuthorizationObservation:
		if value.ProposalID <= 0 || !value.Authorized {
			return fmt.Errorf("authorization observations do not establish an authorized proposal")
		}
		if err := nonEmpty("submitter", value.Submitter); err != nil {
			return err
		}
		for label, digest := range map[string]string{
			"action": value.ActionSHA256, "plan": value.PlanSHA256, "manifest": value.ManifestSHA256,
		} {
			if err := validateSHA256("authorization "+label+" SHA-256", digest); err != nil {
				return err
			}
		}
	case WrongTupleObservation:
		if value.ProposalID <= 0 || !value.Rejected {
			return fmt.Errorf("wrong-tuple observations do not establish rejection")
		}
		return nonEmpty("error_code", value.ErrorCode)
	case RevocationObservation:
		if value.ProposalID <= 0 || value.AuthorizationGeneration <= 0 || !value.Revoked {
			return fmt.Errorf("revocation observations do not establish final revocation")
		}
	case ProposalFailureObservation:
		if value.ProposalID <= 0 || !value.Failed || !value.Dequeued || !value.CannotExecute {
			return fmt.Errorf("proposal-failure observations are incomplete")
		}
	case ProposalRefundObservation:
		if value.ProposalID <= 0 || !value.Refunded {
			return fmt.Errorf("proposal-refund observations do not establish refund")
		}
		if err := nonEmpty("amount", value.Amount); err != nil {
			return err
		}
		return nonEmpty("denom", value.Denom)
	case ResumeHeadObservation:
		if !value.Verified {
			return fmt.Errorf("resume-head observations do not establish verification")
		}
		return validateSHA256("resume head SHA-256", value.HeadSHA256)
	case LatchObservation:
		if value.Height <= 0 || value.Rejected == value.Accepted {
			return fmt.Errorf("%s observations require exactly one of rejected/accepted", kind)
		}
	case SourceProcessObservation:
		if value.ProcessID <= 1 || value.Height <= 0 || !value.LocalPIDAbsent ||
			value.ExternalRestartAssertionVerified {
			return fmt.Errorf("source-process observations do not establish bounded local absence")
		}
		if _, err := validateCanonicalTime(
			"source process start time",
			value.ProcessStartTime,
		); err != nil {
			return err
		}
		for label, digest := range map[string]string{
			"source process identity": value.ProcessIdentitySHA256,
			"source restart inhibit":  value.RestartInhibitEvidenceSHA256,
			"source AppHash":          value.AppHash,
		} {
			if err := validateSHA256(label+" SHA-256", digest); err != nil {
				return err
			}
		}
	case DestinationVerificationObservation:
		if value.Height <= 0 || value.DestinationDeviceID == 0 ||
			value.DestinationRootInode == 0 || !value.LocalManifestVerified ||
			value.ExternalControlAssertionsVerified ||
			!value.DatabaseGroupsVerified ||
			!value.SigningStateVerified || !value.GenesisIdentityVerified ||
			!value.AuthorizedDeltaOnly {
			return fmt.Errorf("destination verification observations are incomplete")
		}
		if err := nonEmpty("destination_volume_id", value.DestinationVolumeID); err != nil {
			return err
		}
		for label, digest := range map[string]string{
			"destination home manifest":     value.HomeManifestSHA256,
			"destination volume evidence":   value.VolumeEvidenceSHA256,
			"destination snapshot evidence": value.SnapshotEvidenceSHA256,
			"destination AppHash":           value.AppHash,
		} {
			if err := validateSHA256(label+" SHA-256", digest); err != nil {
				return err
			}
		}
	case DestinationStartObservation:
		if value.StartHeight <= 0 || value.StartHeight == math.MaxInt64 ||
			value.FirstCommittedHeight != value.StartHeight+1 ||
			!value.NoConcurrentSigner {
			return fmt.Errorf("destination-start observations do not establish safe continuation")
		}
	case ObserverIdentityObservation:
		if err := nonEmpty("observer_id", value.ObserverID); err != nil {
			return err
		}
		if !value.Independent || !value.Authenticated {
			return fmt.Errorf("observer identity is not independent and authenticated")
		}
	case ObserverPlanObservation:
		if err := nonEmpty("plan_name", value.PlanName); err != nil {
			return err
		}
		if value.PlanHeight <= 0 || !value.PlanConfirmed {
			return fmt.Errorf("observer plan observations are incomplete")
		}
	case ObserverStatusObservation:
		if err := nonEmpty("chain_id", value.ChainID); err != nil {
			return err
		}
		if value.Height <= 0 || !value.ImmutableSnapshot {
			return fmt.Errorf("observer status does not bind an immutable height")
		}
		return validateSHA256("observer AppHash", value.AppHash)
	case FaultInjectionObservation:
		if !identifierPattern.MatchString(value.FaultID) || !value.Armed {
			return fmt.Errorf("fault-injection observations are incomplete")
		}
		if err := nonEmpty("reason_code", value.ReasonCode); err != nil {
			return err
		}
		if err := nonEmpty("phase", value.Phase); err != nil {
			return err
		}
		return nonEmpty("injection_point", value.InjectionPoint)
	case FaultResultObservation:
		if !identifierPattern.MatchString(value.FaultID) || !value.ContainmentConfirmed ||
			value.Result != "contained" || isReleaseDecisionToken(value.ExpectedResult) ||
			isReleaseDecisionToken(value.ActualResult) {
			return fmt.Errorf("fault-result observations do not establish containment")
		}
		if err := nonEmpty("expected_result", value.ExpectedResult); err != nil {
			return err
		}
		return nonEmpty("actual_result", value.ActualResult)
	case MutationCensusObservation:
		if !identifierPattern.MatchString(value.FaultID) || value.LastCommittedHeight < 0 ||
			value.ForbiddenMutationObserved || !value.SourceStateUnchanged {
			return fmt.Errorf("mutation-census observations do not establish unchanged source state")
		}
	default:
		return fmt.Errorf("%s observations have unexpected decoded type %T", kind, observation)
	}
	return nil
}

func sealEvidenceEnvelope(data []byte, root string) (EvidenceEnvelope, []byte, error) {
	var draft EvidenceEnvelope
	if err := decodeStrict(data, &draft); err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	if draft.EnvelopeSHA256 != "" {
		return EvidenceEnvelope{}, nil, fmt.Errorf("draft envelope_sha256 must be empty")
	}
	normalizeEnvelope(&draft)
	if err := validateEvidenceEnvelope(draft, draft.Kind, false); err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	if _, err := decodeTypedObservation(draft.Kind, draft.Observations); err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	if err := verifyEnvelopeArtifacts(draft, root); err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	forHash := draft
	forHash.EnvelopeSHA256 = ""
	digest, err := hashCanonical(forHash)
	if err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	draft.EnvelopeSHA256 = digest
	document, err := canonicalDocument(draft)
	if err != nil {
		return EvidenceEnvelope{}, nil, err
	}
	return draft, document, nil
}

func loadTypedEnvelope(reference EvidenceRef, root string) (TypedEnvelope, error) {
	if reference.MediaType != "application/json" {
		return TypedEnvelope{}, fmt.Errorf("%s evidence must use application/json", reference.Kind)
	}
	data, err := readVerifiedEvidence(root, reference)
	if err != nil {
		return TypedEnvelope{}, fmt.Errorf("read %s evidence envelope: %w", reference.Kind, err)
	}
	typed, err := decodeEvidenceEnvelope(data, reference.Kind, true)
	if err != nil {
		return TypedEnvelope{}, fmt.Errorf("%s evidence: %w", reference.Kind, err)
	}
	if err := verifyEnvelopeArtifacts(typed.Envelope, root); err != nil {
		return TypedEnvelope{}, fmt.Errorf("%s evidence: %w", reference.Kind, err)
	}
	return typed, nil
}

func verifyEnvelopeArtifacts(envelope EvidenceEnvelope, root string) error {
	for _, artifact := range envelope.Artifacts {
		if err := inspectEvidence(root, EvidenceRef{
			Kind:      "raw-" + artifact.Role,
			Path:      artifact.Path,
			MediaType: artifact.MediaType,
			SizeBytes: artifact.SizeBytes,
			SHA256:    artifact.SHA256,
		}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeEnvelope(envelope *EvidenceEnvelope) {
	sort.Slice(envelope.Artifacts, func(i, j int) bool {
		if envelope.Artifacts[i].Role != envelope.Artifacts[j].Role {
			return envelope.Artifacts[i].Role < envelope.Artifacts[j].Role
		}
		return envelope.Artifacts[i].Path < envelope.Artifacts[j].Path
	})
}

func ensureEnvelopeOutputDoesNotReplaceArtifact(
	output string,
	root string,
	envelope EvidenceEnvelope,
) error {
	if output == "-" {
		return nil
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve envelope output: %w", err)
	}
	relative, err := filepath.Rel(root, filepath.Clean(absolute))
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil
	}
	relative = filepath.ToSlash(relative)
	for _, artifact := range envelope.Artifacts {
		if artifact.Path == relative {
			return fmt.Errorf("envelope output would replace raw artifact %q", relative)
		}
	}
	return nil
}
