package protocol

import (
	"encoding/json"
	"sort"
)

const Protocol = "kingdom.witnessed-agent-economy/0.1"

type Kind string
type Action string

const (
	KindKingdomReleaseRoot         Kind = "KINGDOM_RELEASE_ROOT"
	KindAgentToolSettlementRoot    Kind = "AGENTTOOL_SETTLEMENT_ROOT"
	KindAgentToolCapability        Kind = "AGENTTOOL_CAPABILITY"
	KindAgentToolPublicRecognition Kind = "AGENTTOOL_PUBLIC_RECOGNITION"
	KindAgentToolOffer             Kind = "AGENTTOOL_OFFER"
	KindWakePublicCheckpoint       Kind = "WAKE_PUBLIC_CHECKPOINT"
	KindIssuerKeyContinuity        Kind = "ISSUER_KEY_CONTINUITY"
	KindArtifactLineage            Kind = "ARTIFACT_LINEAGE"
	KindCollaborationCheckpoint    Kind = "COLLABORATION_CHECKPOINT"
	KindDisputeTerminal            Kind = "DISPUTE_TERMINAL"
)

const (
	ActionCheckpoint Action = "CHECKPOINT"
	ActionGrant      Action = "GRANT"
	ActionConsume    Action = "CONSUME"
	ActionRevoke     Action = "REVOKE"
	ActionAdopt      Action = "ADOPT"
	ActionWithdraw   Action = "WITHDRAW"
	ActionSupersede  Action = "SUPERSEDE"
	ActionSettle     Action = "SETTLE"
	ActionRotate     Action = "ROTATE"
	ActionPublish    Action = "PUBLISH"
)

var allowedActions = map[Kind]map[Action]bool{
	KindKingdomReleaseRoot:         {ActionCheckpoint: true},
	KindAgentToolSettlementRoot:    {ActionCheckpoint: true},
	KindAgentToolCapability:        {ActionGrant: true, ActionConsume: true, ActionRevoke: true},
	KindAgentToolPublicRecognition: {ActionAdopt: true, ActionWithdraw: true},
	KindAgentToolOffer:             {ActionPublish: true, ActionSupersede: true, ActionRevoke: true},
	KindWakePublicCheckpoint:       {ActionCheckpoint: true, ActionSupersede: true, ActionWithdraw: true},
	KindIssuerKeyContinuity:        {ActionRotate: true, ActionRevoke: true},
	KindArtifactLineage:            {ActionCheckpoint: true},
	KindCollaborationCheckpoint:    {ActionCheckpoint: true},
	KindDisputeTerminal:            {ActionSettle: true},
}

var requiredNonclaims = [...]string{
	"COMPETENCE",
	"CONSCIOUSNESS",
	"CONSENT",
	"IDENTITY",
	"PERSONHOOD",
	"QUALITY",
	"REPUTATION",
	"TRUTH",
}

type Issuer struct {
	Namespace      string `json:"namespace"`
	ControllerRef  string `json:"controller_ref"`
	KeyFingerprint string `json:"key_fingerprint"`
}

type Effects struct {
	Scope             string `json:"scope"`
	Authority         string `json:"authority"`
	Economic          string `json:"economic"`
	Reputation        string `json:"reputation"`
	NetworkRequests   uint8  `json:"network_requests"`
	StorageWrites     uint8  `json:"storage_writes"`
	ZeroneTransaction bool   `json:"zerone_transaction"`
	ExternalReceipt   bool   `json:"external_receipt"`
	NENInvocation     bool   `json:"nen_invocation"`
	Score             bool   `json:"score"`
}

type Envelope struct {
	Protocol     string   `json:"protocol"`
	Kind         Kind     `json:"kind"`
	Action       Action   `json:"action"`
	Audience     string   `json:"audience"`
	SubjectRef   string   `json:"subject_ref"`
	Sequence     string   `json:"sequence"`
	Parent       *string  `json:"parent"`
	Issuer       Issuer   `json:"issuer"`
	SchemaHash   string   `json:"schema_hash"`
	PayloadRoot  string   `json:"payload_root"`
	PolicyDigest string   `json:"policy_digest"`
	ExpiryHeight *string  `json:"expiry_height"`
	Effects      Effects  `json:"effects"`
	Nonclaims    []string `json:"nonclaims"`
}

type Signature struct {
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Value     string `json:"value"`
}

type Record struct {
	Envelope   Envelope        `json:"envelope"`
	Payload    json.RawMessage `json:"payload"`
	Commitment string          `json:"commitment"`
	Signature  Signature       `json:"signature"`
}

type Gap struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

type KingdomReleasePayload struct {
	ReleaseRef               string  `json:"release_ref"`
	LedgerProtocol           string  `json:"ledger_protocol"`
	LedgerDocumentDigest     string  `json:"ledger_document_digest"`
	EntryMerkleRoot          string  `json:"entry_merkle_root"`
	EntryCount               string  `json:"entry_count"`
	GitCommit                string  `json:"git_commit"`
	GitTree                  string  `json:"git_tree"`
	BuildManifestDigest      string  `json:"build_manifest_digest"`
	DeploymentManifestDigest string  `json:"deployment_manifest_digest"`
	VerifierProtocol         string  `json:"verifier_protocol"`
	VerifierDigest           string  `json:"verifier_digest"`
	PreviousRelease          *string `json:"previous_release"`
}

type SettlementRootPayload struct {
	ReceiptProtocol        string  `json:"receipt_protocol"`
	ReceiptSchemaDigest    string  `json:"receipt_schema_digest"`
	SourceSequenceBinding  string  `json:"source_sequence_binding"`
	ReceiptUniquenessScope string  `json:"receipt_uniqueness_scope"`
	FirstSequence          string  `json:"first_sequence"`
	LastSequence           string  `json:"last_sequence"`
	ReceiptCount           string  `json:"receipt_count"`
	DeclaredGaps           []Gap   `json:"declared_gaps"`
	MerkleRoot             string  `json:"merkle_root"`
	PreviousBatch          *string `json:"previous_batch"`
}

type CapabilityGrantPayload struct {
	CapabilityRef      string `json:"capability_ref"`
	GrantDigest        string `json:"grant_digest"`
	AssetRef           string `json:"asset_ref"`
	MaxPerConsumeMinor string `json:"max_per_consume_minor"`
	MaxTotalMinor      string `json:"max_total_minor"`
}

type CapabilityConsumePayload struct {
	CapabilityRef     string `json:"capability_ref"`
	GrantCommitment   string `json:"grant_commitment"`
	AssetRef          string `json:"asset_ref"`
	AmountMinor       string `json:"amount_minor"`
	SourceEventDigest string `json:"source_event_digest"`
	Nullifier         string `json:"nullifier"`
}

type CapabilityRevokePayload struct {
	CapabilityRef   string `json:"capability_ref"`
	GrantCommitment string `json:"grant_commitment"`
	ReasonDigest    string `json:"reason_digest"`
}

type RecognitionAdoptPayload struct {
	RecognitionRef         string `json:"recognition_ref"`
	SurfaceDigest          string `json:"surface_digest"`
	RegistryDigest         string `json:"registry_digest"`
	AdoptionDocumentDigest string `json:"adoption_document_digest"`
	AuthoritySequence      string `json:"authority_sequence"`
	Visibility             string `json:"visibility"`
}

type RecognitionWithdrawPayload struct {
	RecognitionRef           string `json:"recognition_ref"`
	AdoptionCommitment       string `json:"adoption_commitment"`
	SurfaceDigest            string `json:"surface_digest"`
	RegistryDigest           string `json:"registry_digest"`
	WithdrawalDocumentDigest string `json:"withdrawal_document_digest"`
	AuthoritySequence        string `json:"authority_sequence"`
	ReasonDigest             string `json:"reason_digest"`
	Visibility               string `json:"visibility"`
}

type OfferPublishPayload struct {
	OfferRef            string `json:"offer_ref"`
	OfferDocumentDigest string `json:"offer_document_digest"`
	CapabilityRoot      string `json:"capability_root"`
	PricingRoot         string `json:"pricing_root"`
	SLARoot             string `json:"sla_root"`
	TermsDigest         string `json:"terms_digest"`
	Revision            string `json:"revision"`
	AuthoritySequence   string `json:"authority_sequence"`
	Visibility          string `json:"visibility"`
}

type OfferSupersedePayload struct {
	OfferRef            string `json:"offer_ref"`
	OfferDocumentDigest string `json:"offer_document_digest"`
	CapabilityRoot      string `json:"capability_root"`
	PricingRoot         string `json:"pricing_root"`
	SLARoot             string `json:"sla_root"`
	TermsDigest         string `json:"terms_digest"`
	Revision            string `json:"revision"`
	AuthoritySequence   string `json:"authority_sequence"`
	Visibility          string `json:"visibility"`
	Supersedes          string `json:"supersedes"`
}

type OfferRevokePayload struct {
	OfferRef            string `json:"offer_ref"`
	OfferCommitment     string `json:"offer_commitment"`
	OfferDocumentDigest string `json:"offer_document_digest"`
	AuthoritySequence   string `json:"authority_sequence"`
	ReasonDigest        string `json:"reason_digest"`
	Visibility          string `json:"visibility"`
}

type WakeCheckpointPayload struct {
	PublicContractProtocol     string `json:"public_contract_protocol"`
	PublicContractSchemaDigest string `json:"public_contract_schema_digest"`
	ContractRoot               string `json:"contract_root"`
	CapabilityRoot             string `json:"capability_root"`
	PricingRoot                string `json:"pricing_root"`
	ProtocolsRoot              string `json:"protocols_root"`
	BoundariesRoot             string `json:"boundaries_root"`
	AuthoritySequence          string `json:"authority_sequence"`
}

type WakeSupersedePayload struct {
	PublicContractProtocol     string `json:"public_contract_protocol"`
	PublicContractSchemaDigest string `json:"public_contract_schema_digest"`
	ContractRoot               string `json:"contract_root"`
	CapabilityRoot             string `json:"capability_root"`
	PricingRoot                string `json:"pricing_root"`
	ProtocolsRoot              string `json:"protocols_root"`
	BoundariesRoot             string `json:"boundaries_root"`
	AuthoritySequence          string `json:"authority_sequence"`
	Supersedes                 string `json:"supersedes"`
}

type WakeWithdrawPayload struct {
	CheckpointCommitment     string `json:"checkpoint_commitment"`
	WithdrawalDocumentDigest string `json:"withdrawal_document_digest"`
	AuthoritySequence        string `json:"authority_sequence"`
	ReasonDigest             string `json:"reason_digest"`
	Visibility               string `json:"visibility"`
}

type KeyRotatePayload struct {
	PreviousKeyFingerprint string `json:"previous_key_fingerprint"`
	NextKeyFingerprint     string `json:"next_key_fingerprint"`
	RotationDigest         string `json:"rotation_digest"`
}

type KeyRevokePayload struct {
	RevokedKeyFingerprint string `json:"revoked_key_fingerprint"`
	ReasonDigest          string `json:"reason_digest"`
}

type ArtifactLineagePayload struct {
	UpstreamRef    string `json:"upstream_ref"`
	DownstreamRef  string `json:"downstream_ref"`
	Relation       string `json:"relation"`
	EvidenceDigest string `json:"evidence_digest"`
}

type CollaborationCheckpointPayload struct {
	WorkspaceRef       string `json:"workspace_ref"`
	EpochRef           string `json:"epoch_ref"`
	EventHeadSequence  string `json:"event_head_sequence"`
	EventHeadHash      string `json:"event_head_hash"`
	EventCount         string `json:"event_count"`
	ParticipantSetRoot string `json:"participant_set_root"`
}

type DisputeTerminalPayload struct {
	SettlementCommitment string `json:"settlement_commitment"`
	Outcome              string `json:"outcome"`
	DecisionDigest       string `json:"decision_digest"`
	DistributionRoot     string `json:"distribution_root"`
}

type SettlementLeaf struct {
	Sequence      string `json:"sequence"`
	ReceiptDigest string `json:"receipt_digest"`
}

type SettlementBatch struct {
	FirstSequence string           `json:"first_sequence"`
	LastSequence  string           `json:"last_sequence"`
	ReceiptCount  string           `json:"receipt_count"`
	DeclaredGaps  []Gap            `json:"declared_gaps"`
	Leaves        []SettlementLeaf `json:"leaves"`
}

var lineageRelations = map[string]bool{
	"DERIVES_FROM":    true,
	"USES_CAPABILITY": true,
	"FULFILLS":        true,
	"SETTLES":         true,
	"CHECKPOINTS":     true,
	"SUPERSEDES":      true,
	"REVOKES":         true,
}

var disputeOutcomes = map[string]bool{
	"RELEASE": true,
	"REFUND":  true,
	"SPLIT":   true,
	"DISMISS": true,
}

// RequiredNonclaims returns a defensive copy of the normative nonclaim set.
func RequiredNonclaims() []string {
	return append([]string(nil), requiredNonclaims[:]...)
}

// KindActionMatrix returns defensive copies of the closed kind/action table.
func KindActionMatrix() map[Kind][]Action {
	result := make(map[Kind][]Action, len(allowedActions))
	for kind, actions := range allowedActions {
		for action := range actions {
			result[kind] = append(result[kind], action)
		}
		sort.Slice(result[kind], func(i, j int) bool { return result[kind][i] < result[kind][j] })
	}
	return result
}

// AllowedLineageRelations returns a sorted defensive copy.
func AllowedLineageRelations() []string {
	result := make([]string, 0, len(lineageRelations))
	for relation := range lineageRelations {
		result = append(result, relation)
	}
	sort.Strings(result)
	return result
}
