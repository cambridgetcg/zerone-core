package main

import "encoding/json"

const (
	manifestRelativePath      = "tools/zerone-supabase-observatory/protocol/manifest.v0.1.json"
	manifestFormat            = "zerone-agenttool-supabase-observatory-manifest/0.1"
	protocolID                = "zerone-agenttool-supabase-observatory/0.1"
	journalFormat             = "zerone-agenttool-supabase-observatory-journal/0.1"
	verificationFormat        = "zerone-agenttool-supabase-observatory-verification/0.1"
	observationIDDomain       = "zerone.observatory-observation-id/0.1"
	expectedManifestRawSHA256 = "sha256:e314476971a702453709710c0ea376216b704a696bc70041dde450600cc06578"
)

type manifest struct {
	Format             string             `json:"_format"`
	Protocol           string             `json:"protocol"`
	Status             string             `json:"status"`
	AsOf               string             `json:"as_of"`
	SourcePins         []sourcePin        `json:"source_pins"`
	PendingBindings    []pendingBinding   `json:"pending_bindings"`
	Schema             artifactRef        `json:"schema"`
	Fixtures           []fixtureRef       `json:"fixtures"`
	Semantics          manifestSemantics  `json:"semantics"`
	CandidateEffects   candidateEffects   `json:"candidate_effects"`
	ValidatorExecution validatorExecution `json:"validator_execution"`
}

type sourcePin struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	SourceState  string `json:"source_state"`
	Repository   string `json:"repository"`
	Revision     string `json:"revision"`
	Path         string `json:"path"`
	RawSHA256    string `json:"raw_sha256"`
	PinRole      string `json:"pin_role"`
	Verification string `json:"verification"`
}

type pendingBinding struct {
	ID                string  `json:"id"`
	Owner             string  `json:"owner"`
	State             string  `json:"state"`
	Repository        string  `json:"repository"`
	Revision          *string `json:"revision"`
	Path              *string `json:"path"`
	RawSHA256         *string `json:"raw_sha256"`
	AuthorityTransfer bool    `json:"authority_transfer"`
	IntegrationReady  bool    `json:"integration_ready"`
}

type artifactRef struct {
	Path      string `json:"path"`
	RawSHA256 string `json:"raw_sha256"`
	Dialect   string `json:"dialect"`
}

type fixtureRef struct {
	ID                       string `json:"id"`
	Path                     string `json:"path"`
	RawSHA256                string `json:"raw_sha256"`
	ExpectedDecision         string `json:"expected_decision"`
	ExpectedObservationCount int    `json:"expected_observation_count"`
	ExpectedConflictGroups   int    `json:"expected_conflict_groups"`
}

type manifestSemantics struct {
	GraphKinds                  []string `json:"graph_kinds"`
	ToKHeightMode               string   `json:"tok_height_mode"`
	ToKRequestAtBlockHeight     int      `json:"tok_request_at_block_height"`
	ToKRootScope                string   `json:"tok_root_scope"`
	RawPayloadDigestRequired    bool     `json:"raw_payload_digest_required"`
	SameHeightConflictPolicy    string   `json:"same_height_conflict_policy"`
	UnavailableSourcePolicy     string   `json:"unavailable_source_policy"`
	SupabaseProjectionRelation  string   `json:"supabase_projection_relation"`
	DatabaseOrderingAuthority   string   `json:"database_ordering_authority"`
	DatabaseTimestampAuthority  string   `json:"database_timestamp_authority"`
	ScientificAuthority         string   `json:"scientific_authority"`
	ObservationIDCanonicalizing string   `json:"observation_id_canonicalization"`
}

// candidateEffects is deliberately flat and comparable. Every coordinate is
// false for the source-only protocol. Validator file reads are disclosed in a
// separate validatorExecution object rather than hidden in this vector.
type candidateEffects struct {
	Network                bool `json:"network"`
	Storage                bool `json:"storage"`
	DatabaseRead           bool `json:"database_read"`
	DatabaseWrite          bool `json:"database_write"`
	AgentToolAPIWrite      bool `json:"agenttool_api_write"`
	HostedRoute            bool `json:"hosted_route"`
	Economic               bool `json:"economic"`
	Governance             bool `json:"governance"`
	Consensus              bool `json:"consensus"`
	Identity               bool `json:"identity"`
	Permission             bool `json:"permission"`
	AuthorityTransfer      bool `json:"authority_transfer"`
	KARMA                  bool `json:"karma"`
	NEN                    bool `json:"nen"`
	Score                  bool `json:"score"`
	ChainRead              bool `json:"chain_read"`
	ChainWrite             bool `json:"chain_write"`
	KnowledgeAdmission     bool `json:"knowledge_admission"`
	ScientificAdjudication bool `json:"scientific_adjudication"`
	Wallet                 bool `json:"wallet"`
	Escrow                 bool `json:"escrow"`
	Payout                 bool `json:"payout"`
	Reward                 bool `json:"reward"`
	ZRN                    bool `json:"zrn"`
	IntegrationReady       bool `json:"integration_ready"`
}

type validatorExecution struct {
	Network        bool   `json:"network"`
	LocalFileRead  string `json:"local_file_read"`
	LocalFileWrite bool   `json:"local_file_write"`
	Database       bool   `json:"database"`
	Chain          bool   `json:"chain"`
	Economic       bool   `json:"economic"`
	Governance     bool   `json:"governance"`
	Identity       bool   `json:"identity"`
	Permission     bool   `json:"permission"`
	KARMA          bool   `json:"karma"`
	NEN            bool   `json:"nen"`
	Score          bool   `json:"score"`
}

type journal struct {
	Format       string            `json:"_format"`
	Mode         string            `json:"mode"`
	Projection   projection        `json:"projection"`
	Observations []json.RawMessage `json:"observations"`
	Effects      candidateEffects  `json:"effects"`
}

type projection struct {
	Provider                  string   `json:"provider"`
	Relation                  string   `json:"relation"`
	Mode                      string   `json:"mode"`
	Authority                 string   `json:"authority"`
	Rebuildable               bool     `json:"rebuildable"`
	DatabaseOrderIsChainOrder bool     `json:"database_order_is_chain_order"`
	DatabaseTimeIsTrustedTime bool     `json:"database_time_is_trusted_time"`
	RowIsScientificTruth      bool     `json:"row_is_scientific_truth"`
	Preserves                 []string `json:"preserves"`
	Loses                     []string `json:"loses"`
}

type observationTag struct {
	GraphKind string `json:"graph_kind"`
}

type tokObservation struct {
	ObservationID string      `json:"observation_id"`
	GraphKind     string      `json:"graph_kind"`
	SourceKind    string      `json:"source_kind"`
	SourceStatus  string      `json:"source_status"`
	ObservedAt    string      `json:"observed_at"`
	Request       tokRequest  `json:"request"`
	Response      tokResponse `json:"response"`
}

type tokRequest struct {
	AtBlockHeight  int    `json:"at_block_height"`
	SelectorSHA256 string `json:"selector_sha256"`
}

type tokResponse struct {
	ReturnedChainID           string `json:"returned_chain_id"`
	ReturnedActualBlockHeight string `json:"returned_actual_block_height"`
	ReturnedBlockHash         string `json:"returned_block_hash"`
	ReturnedAppHash           string `json:"returned_app_hash"`
	ToKSnapshotRoot           string `json:"tok_snapshot_root"`
	ToKRootVersion            string `json:"tok_root_version"`
	RawPayloadSHA256          string `json:"raw_payload_sha256"`
	RawPayloadMediaType       string `json:"raw_payload_media_type"`
	RawPayloadComplete        bool   `json:"raw_payload_complete"`
	ProofPosture              string `json:"proof_posture"`
}

type staticTreeObservation struct {
	ObservationID  string                   `json:"observation_id"`
	GraphKind      string                   `json:"graph_kind"`
	SourceKind     string                   `json:"source_kind"`
	SourceStatus   string                   `json:"source_status"`
	ObservedAt     string                   `json:"observed_at"`
	Source         repositoryBytesSource    `json:"source"`
	Interpretation staticTreeInterpretation `json:"interpretation"`
}

type repositoryBytesSource struct {
	Repository         string `json:"repository"`
	Revision           string `json:"revision"`
	Path               string `json:"path"`
	RawPayloadSHA256   string `json:"raw_payload_sha256"`
	RawPayloadComplete bool   `json:"raw_payload_complete"`
}

type staticTreeInterpretation struct {
	Authoritative   bool `json:"authoritative"`
	NetworkObserved bool `json:"network_observed"`
	RewardBearing   bool `json:"reward_bearing"`
}

type knowledgeGeometryObservation struct {
	ObservationID string                    `json:"observation_id"`
	GraphKind     string                    `json:"graph_kind"`
	SourceKind    string                    `json:"source_kind"`
	SourceStatus  string                    `json:"source_status"`
	ObservedAt    string                    `json:"observed_at"`
	Response      knowledgeGeometryResponse `json:"response"`
}

type knowledgeGeometryResponse struct {
	ReturnedChainID           string `json:"returned_chain_id"`
	ReturnedActualBlockHeight string `json:"returned_actual_block_height"`
	RawPayloadSHA256          string `json:"raw_payload_sha256"`
	RawPayloadComplete        bool   `json:"raw_payload_complete"`
	Completeness              string `json:"completeness"`
	Truncated                 bool   `json:"truncated"`
	ProofPosture              string `json:"proof_posture"`
}

type journalValidation struct {
	ObservationCount int             `json:"observation_count"`
	ConflictGroups   []conflictGroup `json:"conflict_groups"`
}

type conflictGroup struct {
	GraphKind                 string   `json:"graph_kind"`
	ReturnedChainID           string   `json:"returned_chain_id"`
	ReturnedActualBlockHeight string   `json:"returned_actual_block_height"`
	ObservationIDs            []string `json:"observation_ids"`
}

type fixtureVerification struct {
	ID               string `json:"id"`
	ExpectedDecision string `json:"expected_decision"`
	ObservedDecision string `json:"observed_decision"`
	ObservationCount int    `json:"observation_count"`
	ConflictGroups   int    `json:"conflict_groups"`
}

type verificationReport struct {
	Format                   string                `json:"_format"`
	Protocol                 string                `json:"protocol"`
	Decision                 string                `json:"decision"`
	ManifestRawSHA256        string                `json:"manifest_raw_sha256"`
	VerifiedLocalSourcePins  int                   `json:"verified_local_source_pins"`
	PinnedExternalSources    int                   `json:"pinned_external_sources"`
	PendingBindings          int                   `json:"pending_bindings"`
	Fixtures                 []fixtureVerification `json:"fixtures"`
	CandidateEffects         candidateEffects      `json:"candidate_effects"`
	ObservedValidatorEffects validatorExecution    `json:"observed_validator_effects"`
}
