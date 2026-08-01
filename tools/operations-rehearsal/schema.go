package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	reportSchema        = "zerone.operations-rehearsal/v1"
	evidenceIndexSchema = "zerone.operations-rehearsal-evidence-index/v1"
	faultMatrixSchema   = "zerone.operations-rehearsal-fault-matrix/v1"
	maxReportBytes      = 32 << 20
	maxJSONEvidence     = 128 << 20
	maxJSONDepth        = 128
	maxJSONEntries      = 2_000_000
)

type EvidenceRef struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type UpgradeScenario struct {
	Outcome                      string        `json:"outcome"`
	PlanName                     string        `json:"plan_name"`
	PlanInfoSHA256               string        `json:"plan_info_sha256"`
	H2PlanIdentitySHA256         string        `json:"h2_plan_identity_sha256"`
	UpgradeHeight                int64         `json:"upgrade_height"`
	OldLastCommittedHeight       int64         `json:"old_last_committed_height"`
	PreUpgradeAppHash            string        `json:"pre_upgrade_app_hash"`
	PostUpgradeAppHash           string        `json:"post_upgrade_app_hash"`
	ReplayAppHashes              []string      `json:"replay_app_hashes"`
	OldExitCount                 int64         `json:"old_exit_count"`
	TargetCommitCount            int64         `json:"target_commit_count"`
	AppliedPlanHeight            int64         `json:"applied_plan_height"`
	OldDatabaseUnchanged         bool          `json:"old_database_unchanged"`
	ActivationPreflightSatisfied bool          `json:"activation_preflight_satisfied"`
	SourceVersionsExact          bool          `json:"source_versions_exact"`
	PlanInfoExact                bool          `json:"plan_info_exact"`
	StateDeltasVerified          bool          `json:"state_deltas_verified"`
	PostUpgradeRestartObserved   bool          `json:"post_upgrade_restart_observed"`
	Evidence                     []EvidenceRef `json:"evidence"`
}

type QuarantineScenario struct {
	Outcome                     string        `json:"outcome"`
	HaltFinalizedHeight         int64         `json:"halt_finalized_height"`
	ObservedAdvancingHeight     int64         `json:"observed_advancing_height"`
	BlocksContinued             bool          `json:"blocks_continued"`
	OrdinaryTransactionRejected bool          `json:"ordinary_transaction_rejected"`
	WrappedTransactionRejected  bool          `json:"wrapped_transaction_rejected"`
	ICACallbackRejected         bool          `json:"ica_callback_rejected"`
	NoAutomaticResume           bool          `json:"no_automatic_resume"`
	Evidence                    []EvidenceRef `json:"evidence"`
}

type RecoveryScenario struct {
	Outcome                      string        `json:"outcome"`
	AuthorizationTupleExact      bool          `json:"authorization_tuple_exact"`
	WrongTupleRejected           bool          `json:"wrong_tuple_rejected"`
	RevocationFinalized          bool          `json:"revocation_finalized"`
	RevokedProposalFailed        bool          `json:"revoked_proposal_failed"`
	RevokedProposalDequeued      bool          `json:"revoked_proposal_dequeued"`
	RevokedProposalRefunded      bool          `json:"revoked_proposal_refunded"`
	RevokedProposalCannotExecute bool          `json:"revoked_proposal_cannot_execute"`
	ResumeHeadDigestVerified     bool          `json:"resume_head_digest_verified"`
	Evidence                     []EvidenceRef `json:"evidence"`
}

type H1LatchScenario struct {
	Outcome                     string        `json:"outcome"`
	ResumeFinalizationHeight    int64         `json:"resume_finalization_height"`
	SameBlockOrdinaryTxRejected bool          `json:"same_block_ordinary_tx_rejected"`
	NextBlockHeight             int64         `json:"next_block_height"`
	NextBlockOrdinaryTxAccepted bool          `json:"next_block_ordinary_tx_accepted"`
	Evidence                    []EvidenceRef `json:"evidence"`
}

type FreshVolumeScenario struct {
	Outcome                 string        `json:"outcome"`
	HomeManifestSHA256      string        `json:"home_manifest_sha256"`
	DestinationVolumeID     string        `json:"destination_volume_id"`
	SourceLastHeight        int64         `json:"source_last_height"`
	DestinationStartHeight  int64         `json:"destination_start_height"`
	FirstCommittedHeight    int64         `json:"first_committed_height"`
	SourceProcessStopped    bool          `json:"source_process_stopped"`
	NoConcurrentSigner      bool          `json:"no_concurrent_signer"`
	CompleteHomeVerified    bool          `json:"complete_home_verified"`
	DatabaseGroupsVerified  bool          `json:"database_groups_verified"`
	SigningStateVerified    bool          `json:"signing_state_verified"`
	GenesisIdentityVerified bool          `json:"genesis_identity_verified"`
	AuthorizedDeltaOnly     bool          `json:"authorized_delta_only"`
	Evidence                []EvidenceRef `json:"evidence"`
}

type ObserverScenario struct {
	Outcome           string        `json:"outcome"`
	ObserverID        string        `json:"observer_id"`
	Independent       bool          `json:"independent"`
	Authenticated     bool          `json:"authenticated"`
	ImmutableSnapshot bool          `json:"immutable_snapshot"`
	ChainID           string        `json:"chain_id"`
	ObservedHeight    int64         `json:"observed_height"`
	AppHash           string        `json:"app_hash"`
	PlanName          string        `json:"plan_name"`
	PlanHeight        int64         `json:"plan_height"`
	PlanConfirmed     bool          `json:"plan_confirmed"`
	Evidence          []EvidenceRef `json:"evidence"`
}

type FaultScenario struct {
	ID                        string        `json:"id"`
	ReasonCode                string        `json:"reason_code"`
	Phase                     string        `json:"phase"`
	InjectionPoint            string        `json:"injection_point"`
	ExpectedResult            string        `json:"expected_result"`
	ActualResult              string        `json:"actual_result"`
	LastCommittedHeight       int64         `json:"last_committed_height"`
	ForbiddenMutationObserved bool          `json:"forbidden_mutation_observed"`
	ContainmentConfirmed      bool          `json:"containment_confirmed"`
	Outcome                   string        `json:"outcome"`
	Evidence                  []EvidenceRef `json:"evidence"`
}

type Report struct {
	Schema                 string              `json:"schema"`
	RunID                  string              `json:"run_id"`
	Mode                   string              `json:"mode"`
	Outcome                string              `json:"outcome"`
	ChainID                string              `json:"chain_id"`
	StartedAt              string              `json:"started_at"`
	CompletedAt            string              `json:"completed_at"`
	SourceRevision         string              `json:"source_revision"`
	TargetRevision         string              `json:"target_revision"`
	SourceBinarySHA256     string              `json:"source_binary_sha256"`
	TargetBinarySHA256     string              `json:"target_binary_sha256"`
	Upgrade                UpgradeScenario     `json:"upgrade"`
	Quarantine             QuarantineScenario  `json:"quarantine"`
	Recovery               RecoveryScenario    `json:"recovery"`
	H1Latch                H1LatchScenario     `json:"h_plus_one_latch"`
	FreshVolume            FreshVolumeScenario `json:"fresh_volume"`
	Observer               ObserverScenario    `json:"observer"`
	Faults                 []FaultScenario     `json:"faults"`
	EvidenceManifestSHA256 string              `json:"evidence_manifest_sha256"`
	ReportSHA256           string              `json:"report_sha256"`
}

type EvidenceIndex struct {
	Schema    string        `json:"schema"`
	Artifacts []EvidenceRef `json:"artifacts"`
}

type FaultDefinition struct {
	ReasonCode  string `json:"reason_code"`
	Phase       string `json:"phase"`
	Core        bool   `json:"core"`
	Description string `json:"description"`
}

type FaultMatrix struct {
	Schema      string            `json:"schema"`
	Definitions []FaultDefinition `json:"definitions"`
	SHA256      string            `json:"sha256"`
}

func canonicalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return data, nil
}

func canonicalDocument(value any) ([]byte, error) {
	data, err := canonicalJSON(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func hashCanonical(value any) (string, error) {
	data, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func decodeStrict(data []byte, destination any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailer any
	if err := decoder.Decode(&trailer); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode JSON: unexpected value after root")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSON(decoder, "$", 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("schema ambiguity: unexpected value after root")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func walkJSON(decoder *json.Decoder, path string, depth int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("schema ambiguity: JSON nesting exceeds %d levels at %s", maxJSONDepth, path)
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON at %s: %w", path, err)
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		count := 0
		for decoder.More() {
			if count >= maxJSONEntries {
				return fmt.Errorf("schema ambiguity: object at %s exceeds %d fields", path, maxJSONEntries)
			}
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode object key at %s: %w", path, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("decode object key at %s: key is not a string", path)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("schema ambiguity: duplicate JSON key %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := walkJSON(decoder, path+"."+key, depth+1); err != nil {
				return err
			}
			count++
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
			return fmt.Errorf("decode object end at %s", path)
		}
	case '[':
		index := 0
		for decoder.More() {
			if index >= maxJSONEntries {
				return fmt.Errorf("schema ambiguity: array at %s exceeds %d entries", path, maxJSONEntries)
			}
			if err := walkJSON(decoder, fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
				return err
			}
			index++
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
			return fmt.Errorf("decode array end at %s", path)
		}
	default:
		return fmt.Errorf("decode JSON at %s: unexpected delimiter %q", path, delim)
	}
	return nil
}

func validateSHA256(label, value string) error {
	if len(value) != 64 || value != strings.ToLower(value) {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
	}
	return nil
}

func validateCanonicalTime(label, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339: %w", label, err)
	}
	if parsed.Location() != time.UTC || parsed.UTC().Format(time.RFC3339Nano) != value {
		return time.Time{}, fmt.Errorf("%s must be canonical UTC RFC3339 using Z", label)
	}
	return parsed, nil
}
