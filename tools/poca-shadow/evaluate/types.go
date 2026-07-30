// Package evaluate implements the bounded, deterministic shadow projection
// described by the Proof of Constructive Adaptation v0 specification.
//
// It deliberately performs no network access, signature verification,
// consensus read, transaction, qualification update, or reward calculation.
package evaluate

const (
	ProfileSchema     = "zerone.standard-profile/v0"
	EvidenceSchema    = "zerone.evidence-bundle/v0"
	CertificateSchema = "zerone.breakthrough-certificate/v0"
	EvaluatorVersion  = "poca-shadow/v0"
	InTotoStatementV1 = "https://in-toto.io/Statement/v1"
	PredicateTypeV0   = "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/attestations/proof-constructive-adaptation-v0.md"

	AssuranceModeShadowOnly = "SHADOW_ONLY"
	CertificateAssurance    = "UNVERIFIED_SHADOW_PROJECTION"
	EconomicEffectNone      = "NONE"
	NoAttainedTier          = "NONE"
)

// Profile defines a versioned industrial-standard capability graph.
type Profile struct {
	Schema          string          `json:"schema"`
	ProfileID       string          `json:"profile_id"`
	ProfileVersion  string          `json:"profile_version"`
	Title           string          `json:"title"`
	Status          string          `json:"status"`
	AssuranceMode   string          `json:"assurance_mode"`
	Standards       []StandardRef   `json:"standards"`
	Requirements    []Requirement   `json:"requirements"`
	Nodes           []Node          `json:"nodes"`
	CrownNodeID     string          `json:"crown_node_id"`
	ChallengePolicy ChallengePolicy `json:"challenge_policy"`
	Economics       EconomicsPolicy `json:"economics"`
}

// StandardRef pins one external standard, version, status, and exact target.
// A version alone never implies a conformance target.
type StandardRef struct {
	URI     string `json:"uri"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Target  string `json:"target"`
}

// Requirement describes the normalized receipts a node needs. The evaluator
// checks only this bounded shape; it does not verify the external predicate's
// cryptographic or domain semantics.
type Requirement struct {
	ID                                     string `json:"id"`
	Kind                                   string `json:"kind"`
	VerificationRule                       string `json:"verification_rule"`
	PolicyDigest                           string `json:"policy_digest"`
	PredicateType                          string `json:"predicate_type,omitempty"`
	ObserverRole                           string `json:"observer_role"`
	MinCount                               uint32 `json:"min_count"`
	MinIndependentControlClusters          uint32 `json:"min_independent_control_clusters"`
	RequireObserverIndependentFromClaimant bool   `json:"require_observer_independent_from_claimant"`
	HardGuardrail                          bool   `json:"hard_guardrail"`
}

// Node is one contextual, versioned capability. It is not a scalar ranking of
// a person or agent.
type Node struct {
	ID             string   `json:"id"`
	Stage          string   `json:"stage"`
	Tier           string   `json:"tier"`
	Title          string   `json:"title"`
	Prerequisites  []string `json:"prerequisites"`
	RequirementIDs []string `json:"requirement_ids"`
}

type ChallengePolicy struct {
	UnresolvedChallengeBlocksCrown bool `json:"unresolved_challenge_blocks_crown"`
}

// EconomicsPolicy is intentionally closed in v0. Any non-zero or reward
// bearing value is rejected before evaluation.
type EconomicsPolicy struct {
	Mode       string `json:"mode"`
	AmountUzrn string `json:"amount_uzrn"`
}

// EvidenceBundle holds normalized, content-addressed shadow receipts for one
// subject. It has no caller-selected claim or disbursement ID.
type EvidenceBundle struct {
	Schema                     string        `json:"schema"`
	ProfileID                  string        `json:"profile_id"`
	ProfileVersion             string        `json:"profile_version"`
	Subject                    ArtifactRef   `json:"subject"`
	BaselineDigest             string        `json:"baseline_digest"`
	LineageDigest              string        `json:"lineage_digest"`
	Participants               []Participant `json:"participants"`
	Evidence                   []Receipt     `json:"evidence"`
	UnresolvedChallengeDigests []string      `json:"unresolved_challenge_digests"`
}

// ArtifactRef commits to a local or externally retained artifact. A source
// URI is audit metadata only and is never fetched by this package.
type ArtifactRef struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Digest    string `json:"digest"`
	SourceURI string `json:"source_uri,omitempty"`
}

// Participant carries a declared economic-control cluster. v0 records and
// counts the declaration but cannot prove beneficial ownership or control.
type Participant struct {
	ID                  string `json:"id"`
	Role                string `json:"role"`
	Identity            string `json:"identity"`
	ControlClusterClaim string `json:"control_cluster_claim"`
}

// Receipt is a normalized reference to verification performed elsewhere.
// PASS means the bundle declares that the named rule passed. The shadow
// evaluator does not itself verify the referenced signature or predicate.
type Receipt struct {
	ID                        string `json:"id"`
	RequirementID             string `json:"requirement_id"`
	ProducerParticipantID     string `json:"producer_participant_id"`
	ObserverParticipantID     string `json:"observer_participant_id"`
	Result                    string `json:"result"`
	PredicateType             string `json:"predicate_type,omitempty"`
	VerificationRule          string `json:"verification_rule"`
	PolicyDigest              string `json:"policy_digest"`
	SubjectDigest             string `json:"subject_digest"`
	EnvironmentDigest         string `json:"environment_digest"`
	StatementDigest           string `json:"statement_digest"`
	VerificationReceiptDigest string `json:"verification_receipt_digest"`
	SourceURI                 string `json:"source_uri,omitempty"`
}

// Certificate is the deterministic output of a shadow evaluation. Its
// assurance and reward fields are immutable v0 refusals.
type Certificate struct {
	Schema                     string         `json:"schema"`
	EvaluatorVersion           string         `json:"evaluator_version"`
	Assurance                  string         `json:"assurance"`
	ClaimID                    string         `json:"claim_id"`
	Profile                    ProfileBinding `json:"profile"`
	EvidenceBundleDigest       string         `json:"evidence_bundle_digest"`
	Subject                    ArtifactRef    `json:"subject"`
	NodeResults                []NodeResult   `json:"node_results"`
	AttainedTier               string         `json:"attained_tier"`
	CrownStatus                string         `json:"crown_status"`
	UnresolvedChallengeDigests []string       `json:"unresolved_challenge_digests"`
	Reward                     RewardRefusal  `json:"reward"`
	Notices                    []string       `json:"notices"`
	Limitations                []string       `json:"limitations"`
}

type ProfileBinding struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion string `json:"profile_version"`
	Digest         string `json:"digest"`
}

type NodeResult struct {
	NodeID               string              `json:"node_id"`
	Stage                string              `json:"stage"`
	Tier                 string              `json:"tier"`
	Status               string              `json:"status"`
	Prerequisites        []string            `json:"prerequisites"`
	RequirementResults   []RequirementResult `json:"requirement_results"`
	AcceptedEvidenceIDs  []string            `json:"accepted_evidence_ids"`
	ControlClusterClaims []string            `json:"control_cluster_claims"`
	RefusalCodes         []string            `json:"refusal_codes"`
	RefusalDetails       []string            `json:"refusal_details"`
}

type RequirementResult struct {
	RequirementID        string   `json:"requirement_id"`
	Status               string   `json:"status"`
	AcceptedEvidenceIDs  []string `json:"accepted_evidence_ids"`
	ControlClusterClaims []string `json:"control_cluster_claims"`
	RefusalCodes         []string `json:"refusal_codes"`
	RefusalDetails       []string `json:"refusal_details"`
}

type RewardRefusal struct {
	EconomicEffect string `json:"economic_effect"`
	AmountUzrn     string `json:"amount_uzrn"`
	Reason         string `json:"reason"`
}

// InTotoStatement is the unsigned, exact wrapper offered for off-chain DSSE
// signing. The inner and outer subjects are constructed together so they
// cannot drift.
type InTotoStatement struct {
	Type          string          `json:"_type"`
	Subject       []InTotoSubject `json:"subject"`
	PredicateType string          `json:"predicateType"`
	Predicate     Certificate     `json:"predicate"`
}

type InTotoSubject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}
