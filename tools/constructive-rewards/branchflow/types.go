package branchflow

const (
	Schema           = "zerone.branch-flow-shadow/v0"
	AlgorithmVersion = "branch-flow-integer/v0"
	Assurance        = "SHADOW_ONLY"
	EconomicEffect   = "NONE"
	ConservationOK   = "EXACT_BALANCED"
	// ReferencePolicyPreimage is the complete canonical policy value hashed by
	// ReferencePolicyDigest. PolicyDigest itself is deliberately excluded.
	ReferencePolicyPreimage = `{"base_commons_ppm":0,"direct_ppm":600000,"domain_id":"general","domain_revision":1,"downstream_continuation_ppm":500000,"downstream_max_depth":5,"downstream_ppm":300000,"envelope_controller_cap_ppm":1000000,"min_projected_payout_uzrn":"0","program_window_cap_uzrn":"","upstream_continuation_ppm":500000,"upstream_max_depth":5,"upstream_ppm":100000}`
	ReferencePolicyDigest   = "sha256:cc8601fdc0efd8b0260a5a979fa456f43e45be7a3ebd821ca330b389ebb26684"

	PPM uint64 = 1_000_000

	MaxIDBytes             = 128
	MaxAmountBits          = 256
	MaxAmountDecimalDigits = 78
	MaxNodes               = 1_024
	MaxEdges               = 4_096
	MaxParentsPerNode      = 16
	MaxCreditsPerNode      = 32
	MaxControllers         = 256
	MaxDescendantImpacts   = 2_048
	MaxPriorReceiptUses    = 8_192
	MaxDepth               = 20
	MaxTraversalOperations = 2_000_000
	MaxAllocationLines     = 32_768
)

// NodeMode separates payment eligibility from dependency propagation.
type NodeMode string

const (
	NodeModePayAndPropagate NodeMode = "PAY_AND_PROPAGATE"
	NodeModePassThrough     NodeMode = "PASS_THROUGH"
	NodeModeBlocked         NodeMode = "BLOCKED"
)

// Leg identifies the source tranche for one projected allocation.
type Leg string

const (
	LegDirect     Leg = "DIRECT"
	LegUpstream   Leg = "UPSTREAM"
	LegDownstream Leg = "DOWNSTREAM"
	LegCommons    Leg = "COMMONS"
)

const (
	MilestoneE2 = "E2"
	MilestoneE3 = "E3"
	MilestoneE4 = "E4"
	MilestoneE5 = "E5"
	MilestoneE6 = "E6"
)

// ImpactDisposition distinguishes a payable admitted consequence receipt from
// an admitted terminal tombstone whose cohort capacity must remain reserved.
type ImpactDisposition string

const (
	ImpactDispositionPayable  ImpactDisposition = "PAYABLE"
	ImpactDispositionTerminal ImpactDisposition = "TERMINAL"
)

// Policy is snapshotted before an envelope is evaluated. Share fields use the
// 1,000,000 PPM scale and must sum exactly to PPM.
type Policy struct {
	DirectPPM      uint32 `json:"direct_ppm"`
	UpstreamPPM    uint32 `json:"upstream_ppm"`
	DownstreamPPM  uint32 `json:"downstream_ppm"`
	BaseCommonsPPM uint32 `json:"base_commons_ppm"`

	UpstreamContinuationPPM   uint32 `json:"upstream_continuation_ppm"`
	DownstreamContinuationPPM uint32 `json:"downstream_continuation_ppm"`
	UpstreamMaxDepth          uint32 `json:"upstream_max_depth"`
	DownstreamMaxDepth        uint32 `json:"downstream_max_depth"`

	EnvelopeControllerCapPPM uint32 `json:"envelope_controller_cap_ppm"`
	// Empty disables the optional shadow program-window cap. A production
	// design still needs an authoritative, merge-safe program exposure ledger.
	ProgramWindowCapUzrn   string `json:"program_window_cap_uzrn"`
	MinProjectedPayoutUzrn string `json:"min_projected_payout_uzrn"`

	PolicyDigest   string `json:"policy_digest"`
	DomainID       string `json:"domain_id"`
	DomainRevision uint64 `json:"domain_revision"`
}

// DefaultPolicy is an illustrative shadow profile, not a genesis or network
// parameter proposal. It matches the reviewed reference partition: 60% direct,
// 10% upstream, and 30% downstream, with absolute half-per-hop decay to depth
// five. Its controller cap is deliberately non-binding; constrained caps need
// a prospectively authorized policy. The digest commits to the exported
// canonical ReferencePolicyPreimage.
func DefaultPolicy() Policy {
	return Policy{
		DirectPPM:                 600_000,
		UpstreamPPM:               100_000,
		DownstreamPPM:             300_000,
		BaseCommonsPPM:            0,
		UpstreamContinuationPPM:   500_000,
		DownstreamContinuationPPM: 500_000,
		UpstreamMaxDepth:          5,
		DownstreamMaxDepth:        5,
		EnvelopeControllerCapPPM:  1_000_000,
		MinProjectedPayoutUzrn:    "0",
		PolicyDigest:              ReferencePolicyDigest,
		DomainID:                  "general",
		DomainRevision:            1,
	}
}

// Request is a complete closed-window shadow allocation request. Inputs may be
// supplied in any order; Allocate normalizes copies and never mutates them.
type Request struct {
	Schema                 string             `json:"schema"`
	EnvelopeID             string             `json:"envelope_id"`
	FundedClusterID        string             `json:"funded_cluster_id"`
	FundedMilestone        string             `json:"funded_milestone"`
	EnvelopeUzrn           string             `json:"envelope_uzrn"`
	DescendantWindowClosed bool               `json:"descendant_window_closed"`
	Policy                 Policy             `json:"policy"`
	Nodes                  []Node             `json:"nodes"`
	Edges                  []Edge             `json:"edges"`
	DescendantImpacts      []Impact           `json:"descendant_impacts"`
	PriorReceiptUses       []ReceiptUse       `json:"prior_receipt_uses,omitempty"`
	PriorControllerPaid    []ControllerAmount `json:"prior_controller_paid,omitempty"`
}

// Node is one already-resolved semantic cluster. The allocator does not infer
// semantic equivalence or effective control.
type Node struct {
	ClusterID string   `json:"cluster_id"`
	Mode      NodeMode `json:"mode"`
	Credits   []Credit `json:"credits,omitempty"`
}

// Credit is one controller-aggregated role share within a node.
type Credit struct {
	ControllerID string `json:"controller_id"`
	RoleID       string `json:"role_id"`
	WeightPPM    uint32 `json:"weight_ppm"`
}

// Edge points from a child finding to one dependency ancestor. RawDependencyPPM
// is class-scored evidence input; it is normalized locally with every sibling
// edge before flow begins.
type Edge struct {
	ChildClusterID   string `json:"child_cluster_id"`
	ParentClusterID  string `json:"parent_cluster_id"`
	RawDependencyPPM uint32 `json:"raw_dependency_ppm"`
}

// Impact makes one admitted E5/E6 consequence receipt economically relevant
// to a descendant finding. PAYABLE may produce claimant lines; TERMINAL keeps
// its admitted cohort capacity reserved without paying. Receipt keys are
// exclusive economic-use keys in v0.
type Impact struct {
	DescendantClusterID string            `json:"descendant_cluster_id"`
	Milestone           string            `json:"milestone"`
	ReceiptKey          string            `json:"receipt_key"`
	EconomicSlotID      string            `json:"economic_slot_id"`
	Disposition         ImpactDisposition `json:"disposition"`
	ImpactPPM           uint32            `json:"impact_ppm"`
}

// ReceiptUse is an already-consumed economic receipt slot supplied by the
// caller's external replay snapshot.
type ReceiptUse struct {
	ReceiptKey     string `json:"receipt_key"`
	EconomicSlotID string `json:"economic_slot_id"`
}

// ControllerAmount supplies projected value already attributed within the
// external program-window exposure snapshot.
type ControllerAmount struct {
	ControllerID string `json:"controller_id"`
	AmountUzrn   string `json:"amount_uzrn"`
}

// Allocation is one projected claimant line after envelope and program caps.
type Allocation struct {
	Leg           Leg    `json:"leg"`
	Depth         uint32 `json:"depth,omitempty"`
	ClusterID     string `json:"cluster_id"`
	ControllerID  string `json:"controller_id"`
	RoleID        string `json:"role_id"`
	ReceiptKey    string `json:"receipt_key,omitempty"`
	Milestone     string `json:"milestone,omitempty"`
	ProjectedUzrn string `json:"projected_uzrn"`
}

// CommonsAllocation explains value that is not projected to a claimant.
type CommonsAllocation struct {
	Reason        string `json:"reason"`
	SourceLeg     Leg    `json:"source_leg"`
	Depth         uint32 `json:"depth,omitempty"`
	ProjectedUzrn string `json:"projected_uzrn"`
}

// NormalizedEdge exposes the exact PPM flow share used by the allocator.
type NormalizedEdge struct {
	ChildClusterID  string `json:"child_cluster_id"`
	ParentClusterID string `json:"parent_cluster_id"`
	SharePPM        uint32 `json:"share_ppm"`
}

// DepthBucket records an absolute direction bucket. Tail is always projected
// to commons; a non-tail bucket is never enlarged because other depths are
// empty.
type DepthBucket struct {
	Direction     Leg    `json:"direction"`
	Depth         uint32 `json:"depth,omitempty"`
	Tail          bool   `json:"tail"`
	ProjectedUzrn string `json:"projected_uzrn"`
}

// Result is deterministic shadow output. It neither grants an entitlement nor
// authorizes a transfer.
type Result struct {
	Schema               string              `json:"schema"`
	AlgorithmVersion     string              `json:"algorithm_version"`
	Assurance            string              `json:"assurance"`
	EconomicEffect       string              `json:"economic_effect"`
	MovesFunds           bool                `json:"moves_funds"`
	IntegrationReady     bool                `json:"integration_ready"`
	EnvelopeID           string              `json:"envelope_id"`
	FundedClusterID      string              `json:"funded_cluster_id"`
	FundedMilestone      string              `json:"funded_milestone"`
	EnvelopeUzrn         string              `json:"envelope_uzrn"`
	Policy               Policy              `json:"policy"`
	NormalizedEdges      []NormalizedEdge    `json:"normalized_edges"`
	UpstreamBuckets      []DepthBucket       `json:"upstream_buckets"`
	DownstreamBuckets    []DepthBucket       `json:"downstream_buckets"`
	Allocations          []Allocation        `json:"allocations"`
	Commons              []CommonsAllocation `json:"commons"`
	NewReceiptUses       []ReceiptUse        `json:"new_receipt_uses"`
	ProjectedPaidUzrn    string              `json:"projected_paid_uzrn"`
	ProjectedCommonsUzrn string              `json:"projected_commons_uzrn"`
	ConservationCheck    string              `json:"conservation_check"`
}
