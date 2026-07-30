// Package bridge implements the offline, zero-value constructive receipt
// projection described by the constructive receipt shadow v0 specification.
//
// It deliberately performs no network access, signature verification, replay
// ledger mutation, qualification update, consensus call, or economic action.
package bridge

const (
	RequestSchema = "zerone.constructive-receipt-request/v0"
	ReceiptSchema = "zerone.constructive-receipt/v0"
	EscrowSchema  = "zerone.zero-escrow-compartments/v0"
	BridgeVersion = "constructive-receipts/v0"

	TreeSchema               = "zerone.constructive-intelligence-tree/v1"
	TreePolicyVersion        = "1.0.0"
	TreeEvidenceNamespace    = "zerone.constructive-tree-evidence/v1"
	PoCAEvidenceNamespace    = "zerone.poca-shadow-evidence/v0"
	NamespaceRelation        = "NO_EQUIVALENCE"
	TargetNodeID             = "protocol-software-supply-chain@2026q3"
	TargetNodeEvidence       = "E3"
	TargetNodeStage          = "protocol"
	TargetRewardEligibility  = "qualification-only"
	ReviewedTreeDigest       = "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf"
	ReviewedTreePolicyDigest = "sha256:36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7"
	ReviewedTargetNodeDigest = "sha256:862f9f2d70b51dd03e6fc761eb91487e38c83064414b77d11548ead110356104"

	AssuranceUnverified   = "UNVERIFIED_SHADOW_PROJECTION"
	StatusCandidate       = "CANDIDATE"
	StatusRefused         = "REFUSED"
	EffectNone            = "NONE"
	ConsumptionUnrecorded = "NOT_RECORDED"
	ReplayProtectionNone  = "NONE_OFFLINE"
)

// Request pins every document and source tuple accepted by the v0 bridge.
// Pins are caller intent, not authentication.
type Request struct {
	Schema string              `json:"schema"`
	Target TargetRequest       `json:"target"`
	PoCA   PoCARequest         `json:"poca"`
	Source SourceRecordRequest `json:"source"`
}

type TargetRequest struct {
	TreeSchema         string `json:"tree_schema"`
	TreePolicyVersion  string `json:"tree_policy_version"`
	TreePolicyDigest   string `json:"tree_policy_digest"`
	TreeDocumentDigest string `json:"tree_document_digest"`
	NodeID             string `json:"node_id"`
	NodeDigest         string `json:"node_digest"`
}

type PoCARequest struct {
	ProfileID            string `json:"profile_id"`
	ProfileVersion       string `json:"profile_version"`
	ProfileDigest        string `json:"profile_digest"`
	EvidenceBundleDigest string `json:"evidence_bundle_digest"`
	SubjectDigest        string `json:"subject_digest"`
}

type SourceRecordRequest struct {
	SourceSystem string `json:"source_system"`
	RecordID     string `json:"record_id"`
	Revision     string `json:"revision"`
}

// Receipt is a deterministic shadow output. CANDIDATE means only that the
// fixed local bridge predicate was satisfied; it never means tree attainment,
// qualification, replay consumption, payment authority, or entitlement.
type Receipt struct {
	Schema            string             `json:"schema"`
	BridgeVersion     string             `json:"bridge_version"`
	Assurance         string             `json:"assurance"`
	Status            string             `json:"status"`
	ReceiptID         string             `json:"receipt_id"`
	Source            SourceBinding      `json:"source"`
	Tree              TreeBinding        `json:"tree"`
	PoCA              PoCABinding        `json:"poca"`
	NamespaceRelation string             `json:"namespace_relation"`
	Qualification     string             `json:"qualification"`
	EconomicEffect    string             `json:"economic_effect"`
	AmountUzrn        string             `json:"amount_uzrn"`
	Escrow            EscrowCompartments `json:"escrow"`
	RefusalReasons    []RefusalReason    `json:"refusal_reasons"`
	Limitations       []string           `json:"limitations"`
}

type SourceBinding struct {
	SourceSystem     string `json:"source_system"`
	RecordID         string `json:"record_id"`
	Revision         string `json:"revision"`
	ConsumptionKey   string `json:"consumption_key"`
	ConsumptionState string `json:"consumption_state"`
	ReplayProtection string `json:"replay_protection"`
}

type TreeBinding struct {
	EvidenceNamespace          string `json:"evidence_namespace"`
	Schema                     string `json:"schema"`
	PolicyVersion              string `json:"policy_version"`
	PolicyDigest               string `json:"policy_digest"`
	DocumentDigest             string `json:"document_digest"`
	NodeID                     string `json:"node_id"`
	NodeDigest                 string `json:"node_digest"`
	RequiredAttainmentEvidence string `json:"required_attainment_evidence"`
	GrantedAttainmentEvidence  string `json:"granted_attainment_evidence"`
}

type PoCABinding struct {
	EvidenceNamespace    string `json:"evidence_namespace"`
	ProfileID            string `json:"profile_id"`
	ProfileVersion       string `json:"profile_version"`
	ProfileStatus        string `json:"profile_status"`
	ProfileDigest        string `json:"profile_digest"`
	EvidenceBundleDigest string `json:"evidence_bundle_digest"`
	ClaimID              string `json:"claim_id"`
	SubjectDigest        string `json:"subject_digest"`
	AttainedTier         string `json:"attained_tier"`
	CrownStatus          string `json:"crown_status"`
	Assurance            string `json:"assurance"`
}

type RefusalReason struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// EscrowCompartments is closed to the exact all-zero v0 fixture.
type EscrowCompartments struct {
	Schema                             string `json:"schema"`
	Denom                              string `json:"denom"`
	FundedEscrowUzrn                   string `json:"funded_escrow_uzrn"`
	VerifiedCostBudgetUzrn             string `json:"verified_cost_budget_uzrn"`
	ClaimantMilestoneTranchesUzrn      string `json:"claimant_milestone_tranches_uzrn"`
	ChallengeAndRemediationReserveUzrn string `json:"challenge_and_remediation_reserve_uzrn"`
	ReviewerBudgetUzrn                 string `json:"reviewer_budget_uzrn"`
	AdministrationAndFeeBudgetUzrn     string `json:"administration_and_fee_budget_uzrn"`
	RefundableBalanceUzrn              string `json:"refundable_balance_uzrn"`
	ConservationCheck                  string `json:"conservation_check"`
}

func zeroEscrow() EscrowCompartments {
	return EscrowCompartments{
		Schema:                             EscrowSchema,
		Denom:                              "uzrn",
		FundedEscrowUzrn:                   "0",
		VerifiedCostBudgetUzrn:             "0",
		ClaimantMilestoneTranchesUzrn:      "0",
		ChallengeAndRemediationReserveUzrn: "0",
		ReviewerBudgetUzrn:                 "0",
		AdministrationAndFeeBudgetUzrn:     "0",
		RefundableBalanceUzrn:              "0",
		ConservationCheck:                  "ZERO_BALANCED",
	}
}
