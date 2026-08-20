package bridge

const (
	SettlementFormat       = "agenttool.research-settlement-bundle/0.1"
	PublicProjectionFormat = "agenttool.research-public-projection/0.1"
	NodeRefFormat          = "agenttool.research-node-ref/0.1"
	InteropProfileFormat   = "agenttool.research-commons-zerone-static-interop/0.1"
	InteropProfileDigest   = "sha256:8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a"
	InteropProfileStatus   = "SHADOW_ONLY_NO_LIVE_INTEGRATION"
	ReceiptSchema          = "zerone.agenttool-research-receipt-shadow/v0"
	AdapterVersion         = "agenttool-research-receipt/v1"

	TreeSchema        = "zerone.constructive-intelligence-tree/v1"
	TreeRawDigest     = "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf"
	TargetNodeID      = "math-proofcraft@1"
	TargetNodeDigest  = "sha256:d8f364772611a214aaf5f671c630a5fa00daa3558330bfaf5e85efe7c5a1d0e2"
	PaymentCondition  = "SIMULATED_DELIVERY_ONLY"
	ResultAuthority   = "NONE"
	SimulatedUnit     = "SIMULATED_NONTRANSFERABLE_CREDIT"
	DisclosureLane    = "PUBLIC_DIGEST_ONLY"
	Assurance         = "UNVERIFIED_SHADOW_PROJECTION"
	StatusCandidate   = "STRUCTURAL_CANDIDATE"
	EffectNone        = "NONE"
	NoEquivalence     = "NO_EQUIVALENCE"
	ConsumptionState  = "NOT_RECORDED"
	ReplayProtection  = "NONE_OFFLINE"
	NodeCanonicalizer = "RECURSIVE_UNICODE_CODE_POINT_KEYS_COMPACT_JSON"
	StaticAnchorKind  = "STATIC_CAPABILITY_REFERENCE"
	ProjectionShadow  = "SHADOW_ONLY"
	LedgerProfileID   = "research-commons.six-ledger-boundary/0.1"
	LedgerProfileHash = "sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2"
)

type settlementEnvelope struct {
	SettlementID string         `json:"settlement_id"`
	Settlement   settlementBody `json:"settlement"`
}

type settlementBody struct {
	Format             string          `json:"_format"`
	CaseID             string          `json:"case_id"`
	CommitmentID       string          `json:"commitment_id"`
	ConsumedReceiptIDs []string        `json:"consumed_receipt_ids"`
	DeclaredResultKind string          `json:"declared_result_kind"`
	Effects            zeroEffects     `json:"effects"`
	MilestoneID        string          `json:"milestone_id"`
	PaymentCondition   string          `json:"payment_condition"`
	ResultAuthority    string          `json:"result_authority"`
	SimulatedCredit    simulatedCredit `json:"simulated_credit"`
}

type simulatedCredit struct {
	Amount int64  `json:"amount"`
	Unit   string `json:"unit"`
}

type projectionEnvelope struct {
	Projection   projectionBody `json:"projection"`
	ProjectionID string         `json:"projection_id"`
}

type projectionBody struct {
	Format                    string               `json:"_format"`
	Boundaries                projectionBoundaries `json:"boundaries"`
	CaseID                    string               `json:"case_id"`
	DisclosureLane            string               `json:"disclosure_lane"`
	Effects                   zeroEffects          `json:"effects"`
	HighestEvidenceLevel      *string              `json:"highest_evidence_level"`
	NodeRef                   nodeRef              `json:"node_ref"`
	PublicArtifactRevisionIDs []string             `json:"public_artifact_revision_ids"`
	PublicEvidenceReceiptIDs  []string             `json:"public_evidence_receipt_ids"`
	ResultAuthority           string               `json:"result_authority"`
	SixLedgerBoundary         sixLedgerBoundary    `json:"six_ledger_boundary"`
	SettlementBundleIDs       []string             `json:"settlement_bundle_ids"`
	Status                    string               `json:"status"`
}

type projectionBoundaries struct {
	Authoritative                   bool `json:"authoritative"`
	PrivateLocatorIncluded          bool `json:"private_locator_included"`
	RawEvidenceIncluded             bool `json:"raw_evidence_included"`
	ScientificCorrectnessDetermined bool `json:"scientific_correctness_determined"`
}

type nodeRef struct {
	Format           string `json:"_format"`
	AnchorKind       string `json:"anchor_kind"`
	Canonicalization string `json:"canonicalization"`
	LiveFact         bool   `json:"live_fact"`
	NetworkObserved  bool   `json:"network_observed"`
	NodeDigest       string `json:"node_digest"`
	NodeID           string `json:"node_id"`
	NodeRefID        string `json:"node_ref_id"`
	ResultAuthority  string `json:"result_authority"`
	RewardBearing    bool   `json:"reward_bearing"`
	TreeRawSHA256    string `json:"tree_raw_sha256"`
	TreeSchema       string `json:"tree_schema"`
}

type sixLedgerBoundary struct {
	ProfileDigest string `json:"profile_digest"`
	ProfileID     string `json:"profile_id"`
}

// zeroEffects is deliberately verbose. A new effect does not silently inherit
// false: adding one requires a versioned protocol change in both repositories.
type zeroEffects struct {
	AgentToolAPIWrite      bool `json:"agenttool_api_write"`
	AgentToolDatabaseWrite bool `json:"agenttool_database_write"`
	Authority              bool `json:"authority"`
	Bridge                 bool `json:"bridge"`
	Burn                   bool `json:"burn"`
	ChainWrite             bool `json:"chain_write"`
	Consent                bool `json:"consent"`
	CrossLedgerEquivalence bool `json:"cross_ledger_equivalence"`
	Economic               bool `json:"economic"`
	Escrow                 bool `json:"escrow"`
	ExternalValue          bool `json:"external_value"`
	Governance             bool `json:"governance"`
	Identity               bool `json:"identity"`
	IdentityEquivalence    bool `json:"identity_equivalence"`
	KnowledgeAdmission     bool `json:"knowledge_admission"`
	HostedRoute            bool `json:"hosted_route"`
	Mainnet                bool `json:"mainnet"`
	Mint                   bool `json:"mint"`
	Network                bool `json:"network"`
	Payout                 bool `json:"payout"`
	Qualification          bool `json:"qualification"`
	Reputation             bool `json:"reputation"`
	Reward                 bool `json:"reward"`
	ScientificAdjudication bool `json:"scientific_adjudication"`
	Transfer               bool `json:"transfer"`
	Wallet                 bool `json:"wallet"`
	ZRN                    bool `json:"zrn"`
	ZeroneRead             bool `json:"zerone_read"`
	ZeroneWrite            bool `json:"zerone_write"`
}

// Receipt is a deterministic local compatibility projection. Its simulated
// credit amount is retained only as an input-audit field; it has no exchange
// rate and cannot authorize any value movement.
type Receipt struct {
	Schema              string            `json:"schema"`
	AdapterVersion      string            `json:"adapter_version"`
	Assurance           string            `json:"assurance"`
	Status              string            `json:"status"`
	ReceiptID           string            `json:"receipt_id"`
	Source              SourceBinding     `json:"source"`
	Interop             InteropBinding    `json:"interop"`
	Tree                TreeBinding       `json:"tree"`
	Declaration         ResultDeclaration `json:"declaration"`
	Simulation          SimulationBinding `json:"simulation"`
	LedgerBoundary      LedgerBinding     `json:"ledger_boundary"`
	CrossLedgerRelation string            `json:"cross_ledger_relation"`
	KnowledgeAdmission  string            `json:"knowledge_admission"`
	Qualification       string            `json:"qualification"`
	EconomicEffect      string            `json:"economic_effect"`
	AmountUzrn          string            `json:"amount_uzrn"`
	Effects             ZeroneEffects     `json:"effects"`
	Limitations         []string          `json:"limitations"`
}

type SourceBinding struct {
	SettlementFormat string `json:"settlement_format"`
	SettlementID     string `json:"settlement_id"`
	ProjectionFormat string `json:"projection_format"`
	ProjectionID     string `json:"projection_id"`
	ConsumptionKey   string `json:"consumption_key"`
	ConsumptionState string `json:"consumption_state"`
	ReplayProtection string `json:"replay_protection"`
}

type InteropBinding struct {
	Format            string `json:"format"`
	RawDigest         string `json:"raw_digest"`
	IntegrationStatus string `json:"integration_status"`
	Imported          bool   `json:"imported"`
	Activated         bool   `json:"activated"`
}

type TreeBinding struct {
	Schema                    string `json:"schema"`
	DocumentDigest            string `json:"document_digest"`
	NodeID                    string `json:"node_id"`
	NodeDigest                string `json:"node_digest"`
	NetworkObserved           bool   `json:"network_observed"`
	RewardBearing             bool   `json:"reward_bearing"`
	GrantedAttainmentEvidence string `json:"granted_attainment_evidence"`
}

type ResultDeclaration struct {
	CaseID               string  `json:"case_id"`
	DeclaredResultKind   string  `json:"declared_result_kind"`
	HighestEvidenceLevel *string `json:"highest_evidence_level"`
	ResultAuthority      string  `json:"result_authority"`
}

type SimulationBinding struct {
	PaymentCondition string `json:"payment_condition"`
	CreditAmount     int64  `json:"credit_amount"`
	CreditUnit       string `json:"credit_unit"`
	Convertible      bool   `json:"convertible"`
	Transferable     bool   `json:"transferable"`
	WalletBearing    bool   `json:"wallet_bearing"`
}

type LedgerBinding struct {
	ProfileID             string `json:"profile_id"`
	ProfileDigest         string `json:"profile_digest"`
	SharedUnit            bool   `json:"shared_unit"`
	CrossLedgerArithmetic bool   `json:"cross_ledger_arithmetic"`
	CrossLedgerConversion bool   `json:"cross_ledger_conversion"`
	CrossLedgerInference  bool   `json:"cross_ledger_inference"`
}

type ZeroneEffects struct {
	ReadsChainState     bool `json:"reads_chain_state"`
	WritesChainState    bool `json:"writes_chain_state"`
	SubmitsClaim        bool `json:"submits_claim"`
	InvokesBridge       bool `json:"invokes_bridge"`
	MovesFunds          bool `json:"moves_funds"`
	Mints               bool `json:"mints"`
	Burns               bool `json:"burns"`
	GrantsReward        bool `json:"grants_reward"`
	GrantsQualification bool `json:"grants_qualification"`
	ChangesReputation   bool `json:"changes_reputation"`
	ChangesGovernance   bool `json:"changes_governance"`
	InfersIdentity      bool `json:"infers_identity"`
	InfersController    bool `json:"infers_controller"`
	RecordsConsent      bool `json:"records_consent"`
	AdjudicatesScience  bool `json:"adjudicates_science"`
}

func noZeroneEffects() ZeroneEffects {
	return ZeroneEffects{}
}
