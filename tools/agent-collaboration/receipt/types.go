// Package receipt implements Zerone's offline agent-collaboration receipt
// profile. It has no network, chain, wallet, reward, or governance surface.
package receipt

import "encoding/json"

const (
	ManifestSchema     = "zerone.agent-collaboration-manifest/v0"
	PrivateKeySchema   = "zerone.agent-collaboration-private-key/v0"
	PublicKeySchema    = "zerone.agent-collaboration-public-key/v0"
	EventRequestSchema = "zerone.agent-collaboration-event-request/v0"
	EventSchema        = "zerone.agent-collaboration-event/v0"
	ReceiptSchema      = "zerone.agent-collaboration-receipt/v0"
	ReportSchema       = "zerone.agent-collaboration-verification/v0"

	ModeInternalLocal = "INTERNAL_LOCAL_ONLY"
	AlgorithmEd25519  = "ED25519"
	None              = "NONE"

	EventTaskProposed      = "TASK_PROPOSED"
	EventTaskDecision      = "TASK_DECISION"
	EventContribution      = "CONTRIBUTION_SUBMITTED"
	EventCompletionClaimed = "COMPLETION_CLAIMED"
	EventCompletionReview  = "COMPLETION_REVIEWED"
	EventHandoffOffered    = "HANDOFF_OFFERED"
	EventControl           = "CONTROL_DECLARED"

	DecisionAccept   = "ACCEPT"
	DecisionRefuse   = "REFUSE"
	ReviewAccept     = "ACCEPT"
	ReviewDispute    = "DISPUTE"
	ControlPause     = "PAUSE"
	ControlResume    = "RESUME"
	ControlStop      = "STOP"
	ControlExit      = "EXIT"
	DisclosureLocal  = "LOCAL_ONLY"
	CreditAppendOnly = "ARTIFACT_AND_ROLE_APPEND_ONLY"

	AssuranceNoSignedEvents     = "NO_SIGNED_EVENTS"
	AssuranceEventKeyPossession = "KEY_POSSESSION_VERIFIED_FOR_EACH_SIGNED_EVENT"
	StatusUnanswered            = "UNANSWERED"
	StatusActive                = "ACCEPTED_SCOPE"
	StatusRefused               = "REFUSED"
	StatusClaimed               = "DECLARED_COMPLETE_BY_SIGNER"
	StatusAccepted              = "PROTOCOL_ACCEPTED_BY_PROPOSER"
	StatusContested             = "CONTESTED"

	ParticipationUnbound       = "UNBOUND"
	ParticipationActive        = "ACTIVE"
	ParticipationPauseDeclared = "PAUSE_DECLARED"
	ParticipationEnded         = "ENDED"
)

// Effects is deliberately closed to these exact non-effects. A receipt can
// witness a declaration; it cannot move value or confer power.
type Effects struct {
	Network       string `json:"network"`
	Chain         string `json:"chain"`
	Economic      string `json:"economic"`
	Fiat          string `json:"fiat"`
	ZRN           string `json:"zrn"`
	Reward        string `json:"reward"`
	Karma         string `json:"karma"`
	Governance    string `json:"governance"`
	Ownership     string `json:"ownership"`
	Qualification string `json:"qualification"`
	Membership    string `json:"membership"`
	Endorsement   string `json:"endorsement"`
	Authority     string `json:"authority"`
	Attribution   string `json:"attribution"`
}

// ZeroEffects returns the only effects block accepted by v0.
func ZeroEffects() Effects {
	return Effects{
		Network:       None,
		Chain:         None,
		Economic:      None,
		Fiat:          None,
		ZRN:           None,
		Reward:        None,
		Karma:         None,
		Governance:    None,
		Ownership:     None,
		Qualification: None,
		Membership:    None,
		Endorsement:   None,
		Authority:     None,
		Attribution:   None,
	}
}

// Participant is a self-certifying local signing identity. Label is display
// text only; neither it nor the signature proves a legal or organizational ID.
type Participant struct {
	ActorID   string `json:"actor_id"`
	Label     string `json:"label"`
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
}

// PrivateKeyFile is local key material. It must never enter a journal.
type PrivateKeyFile struct {
	Schema     string `json:"schema"`
	ActorID    string `json:"actor_id"`
	Label      string `json:"label"`
	KeyID      string `json:"key_id"`
	Algorithm  string `json:"algorithm"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// PublicKeyFile contains only the roster-safe half of an internal key.
type PublicKeyFile struct {
	Schema      string      `json:"schema"`
	Participant Participant `json:"participant"`
}

// Manifest freezes the local collaboration identity, roster, and boundary.
type Manifest struct {
	Schema          string        `json:"schema"`
	Mode            string        `json:"mode"`
	CollaborationID string        `json:"collaboration_id"`
	CreatedAt       string        `json:"created_at"`
	Nonce           string        `json:"nonce"`
	Participants    []Participant `json:"participants"`
	Effects         Effects       `json:"effects"`
}

// EventRequest is the unsigned, local request supplied to append. The journal
// fills the collaboration ID, sequence, head, nonce, and signature.
type EventRequest struct {
	Schema     string          `json:"schema"`
	Kind       string          `json:"kind"`
	ActorID    string          `json:"actor_id"`
	OccurredAt string          `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// Event is the exact typed content committed by an event ID and signature.
type Event struct {
	Schema                string          `json:"schema"`
	CollaborationID       string          `json:"collaboration_id"`
	Sequence              string          `json:"sequence"`
	PreviousReceiptSHA256 string          `json:"previous_receipt_sha256"`
	Kind                  string          `json:"kind"`
	ActorID               string          `json:"actor_id"`
	ActorKeyID            string          `json:"actor_key_id"`
	OccurredAt            string          `json:"occurred_at"`
	Nonce                 string          `json:"nonce"`
	Payload               json.RawMessage `json:"payload"`
	Effects               Effects         `json:"effects"`
}

type Signature struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Value     string `json:"value"`
}

// SignedReceipt is one immutable link in a collaboration journal.
type SignedReceipt struct {
	Schema        string    `json:"schema"`
	EventID       string    `json:"event_id"`
	Event         Event     `json:"event"`
	Signature     Signature `json:"signature"`
	ReceiptSHA256 string    `json:"receipt_sha256"`
}

// ConsentTerms freezes the complete scope that an addressed participant may
// accept or refuse. Any change requires a new offer and a new decision.
type ConsentTerms struct {
	Role               string `json:"role"`
	Artifact           string `json:"artifact"`
	Purpose            string `json:"purpose"`
	DisclosureLane     string `json:"disclosure_lane"`
	Term               string `json:"term"`
	WorkloadCap        string `json:"workload_cap"`
	CreditRule         string `json:"credit_rule"`
	CompensationPolicy string `json:"compensation_policy"`
}

type TaskProposed struct {
	TaskID                 string       `json:"task_id"`
	ParentTaskID           string       `json:"parent_task_id"`
	Objective              string       `json:"objective"`
	OfferedToActorID       string       `json:"offered_to_actor_id"`
	OfferedToActorKeyID    string       `json:"offered_to_actor_key_id"`
	AcceptanceRequired     bool         `json:"acceptance_required"`
	ConsentTerms           ConsentTerms `json:"consent_terms"`
	ConsentTermsSHA256     string       `json:"consent_terms_sha256"`
	AcceptanceCriteria     []string     `json:"acceptance_criteria"`
	RequiredArtifactSHA256 []string     `json:"required_artifact_sha256"`
}

type TaskDecision struct {
	TaskID             string   `json:"task_id"`
	OfferEventID       string   `json:"offer_event_id"`
	Decision           string   `json:"decision"`
	Affirmative        bool     `json:"affirmative_acceptance"`
	ConsentTermsSHA256 string   `json:"consent_terms_sha256"`
	ReasonCodes        []string `json:"reason_codes"`
}

type ContributionSubmitted struct {
	TaskID            string   `json:"task_id"`
	AcceptanceEventID string   `json:"acceptance_event_id"`
	Summary           string   `json:"summary"`
	ArtifactSHA256    []string `json:"artifact_sha256"`
	EvidenceSHA256    []string `json:"evidence_sha256"`
	LimitationCodes   []string `json:"limitation_codes"`
}

type CompletionClaimed struct {
	TaskID               string   `json:"task_id"`
	AcceptanceEventID    string   `json:"acceptance_event_id"`
	ContributionEventIDs []string `json:"contribution_event_ids"`
	DeliverableSHA256    []string `json:"deliverable_sha256"`
	LimitationCodes      []string `json:"limitation_codes"`
}

type CompletionReviewed struct {
	TaskID            string   `json:"task_id"`
	CompletionEventID string   `json:"completion_event_id"`
	Decision          string   `json:"decision"`
	ReasonCodes       []string `json:"reason_codes"`
	EvidenceSHA256    []string `json:"evidence_sha256"`
}

type HandoffOffered struct {
	TaskID                string       `json:"task_id"`
	AcceptanceEventID     string       `json:"acceptance_event_id"`
	OfferedToActorID      string       `json:"offered_to_actor_id"`
	OfferedToActorKeyID   string       `json:"offered_to_actor_key_id"`
	AcceptanceRequired    bool         `json:"acceptance_required"`
	ConsentTerms          ConsentTerms `json:"consent_terms"`
	ConsentTermsSHA256    string       `json:"consent_terms_sha256"`
	ContextArtifactSHA256 []string     `json:"context_artifact_sha256"`
}

type ControlDeclared struct {
	TaskID            string   `json:"task_id"`
	AcceptanceEventID string   `json:"acceptance_event_id"`
	Action            string   `json:"action"`
	ReasonCodes       []string `json:"reason_codes"`
	ExportEventIDs    []string `json:"export_event_ids"`
}

// TaskSummary is a projection of signed declarations, never a quality or truth
// score. Status names stay deliberately epistemic.
type TaskSummary struct {
	TaskID                string   `json:"task_id"`
	ParentTaskID          string   `json:"parent_task_id"`
	ProposerActorID       string   `json:"proposer_actor_id"`
	OfferedToActorID      string   `json:"offered_to_actor_id"`
	ActiveActorID         string   `json:"active_actor_id"`
	Status                string   `json:"status"`
	Participation         string   `json:"participation"`
	OfferEventID          string   `json:"offer_event_id"`
	AcceptanceEventID     string   `json:"acceptance_event_id"`
	ContributionEventIDs  []string `json:"contribution_event_ids"`
	LatestCompletionEvent string   `json:"latest_completion_event_id"`
}

type VerificationReport struct {
	Schema            string        `json:"schema"`
	Valid             bool          `json:"valid"`
	Mode              string        `json:"mode"`
	Assurance         string        `json:"assurance"`
	CollaborationID   string        `json:"collaboration_id"`
	EventCount        string        `json:"event_count"`
	HeadReceiptSHA256 string        `json:"head_receipt_sha256"`
	Effects           Effects       `json:"effects"`
	Tasks             []TaskSummary `json:"tasks"`
	Limitations       []string      `json:"limitations"`
}
