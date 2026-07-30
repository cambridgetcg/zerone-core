package main

const (
	transitionSchema            = "zerone.ops.transition/v1"
	trustPolicySchema           = "zerone.ops.trust-policy/v1"
	powerSnapshotSchema         = "zerone.ops.validator-power-snapshot/v1"
	supersessionSchema          = "zerone.ops.supersession/v1"
	approvalDomain              = "zerone.ops.approval/v1\x00"
	supersessionApprovalDomain  = "zerone.ops.supersession-approval/v1\x00"
	supersessionSidecarEvidence = "supersession-sidecar"
	supersededHeadEvidence      = "superseded-journal-head"
	validatorOperatorRole       = "validator-operator"
	releaseAuthorRole           = "release-author"
	releaseVerifierRole         = "release-verifier"
	incidentCommanderRole       = "incident-commander"
	ibcLeadRole                 = "ibc-lead"
	supplyVerifierRole          = "supply-verifier"
	governanceCoordinatorRole   = "governance-coordinator"
	evidenceCustodianRole       = "evidence-custodian"
	policyRotationAuthorityRole = "policy-rotation-authority"
)

// State is an append-only operational state. Planned releases and hostile
// incidents have separate transition graphs; both begin at RUNNING.
type State string

const (
	StateRunning        State = "RUNNING"
	StatePreparing      State = "PREPARING"
	StateReleaseFrozen  State = "RELEASE_FROZEN"
	StateScheduled      State = "SCHEDULED"
	StateStaged         State = "STAGED"
	StateHaltedAtH      State = "HALTED_AT_H"
	StateMigratingH     State = "MIGRATING_H"
	StateObserving      State = "OBSERVING"
	StateAccepted       State = "ACCEPTED"
	StateCancelled      State = "CANCELLED"
	StateAssessing      State = "ASSESSING"
	StateContaining     State = "CONTAINING"
	StateSafetyStopped  State = "SAFETY_STOPPED"
	StateRecoveryDesign State = "RECOVERY_DESIGN"
	StateRecoveryReady  State = "RECOVERY_READY"
	StateActivating     State = "ACTIVATING"
	StateRecoveryFailed State = "RECOVERY_FAILED"
	StateForkChoice     State = "FORK_CHOICE"
	StateClosed         State = "CLOSED"
)

type lane string

const (
	laneRelease  lane = "release"
	laneIncident lane = "incident"
)

type activationMode string

const (
	activationModeCosmovisor     activationMode = "cosmovisor"
	activationModeImmutableImage activationMode = "immutable-image"
)

// Transition is one canonical, hash-chained operations decision. Every field is
// deliberately present in canonical JSON; empty slices must be encoded as [].
type Transition struct {
	Schema                   string         `json:"schema"`
	Lane                     lane           `json:"lane"`
	Sequence                 uint64         `json:"sequence"`
	From                     State          `json:"from"`
	To                       State          `json:"to"`
	Event                    string         `json:"event"`
	OccurredAt               string         `json:"occurred_at"`
	ChainID                  string         `json:"chain_id"`
	IncidentID               string         `json:"incident_id"`
	ReleaseID                string         `json:"release_id"`
	ActorRole                string         `json:"actor_role"`
	ActorIdentity            string         `json:"actor_identity"`
	Checkpoint               Checkpoint     `json:"checkpoint"`
	Release                  ReleaseBinding `json:"release"`
	PowerSnapshot            PowerSnapshot  `json:"power_snapshot"`
	Evidence                 []Evidence     `json:"evidence"`
	Approvals                []Approval     `json:"approvals"`
	TrustPolicySHA256        string         `json:"trust_policy_sha256"`
	PreviousTransitionSHA256 string         `json:"previous_transition_sha256"`
	TransitionSHA256         string         `json:"transition_sha256"`
}

// Checkpoint is an observed committed state, not a declaration that consensus
// has halted. A zero height requires both hashes to be empty.
type Checkpoint struct {
	Height        uint64 `json:"height"`
	BlockIDSHA256 string `json:"block_id_sha256"`
	AppHashSHA256 string `json:"app_hash_sha256"`
}

// ReleaseBinding pins the scheduled upgrade and its reviewed artifacts.
// Empty hashes may be introduced later, but a non-empty binding is immutable
// for the remainder of a journal.
type ReleaseBinding struct {
	PlanName            string         `json:"plan_name"`
	UpgradeHeight       uint64         `json:"upgrade_height"`
	ActivationMode      activationMode `json:"activation_mode"`
	PlanInfoSHA256      string         `json:"plan_info_sha256"`
	BinarySHA256        string         `json:"binary_sha256"`
	ImageSHA256         string         `json:"image_sha256"`
	ProvenanceSHA256    string         `json:"provenance_sha256"`
	SBOMSHA256          string         `json:"sbom_sha256"`
	StateManifestSHA256 string         `json:"state_manifest_sha256"`
}

// Evidence references content-addressed material. URI can be a private
// evidence-vault locator; only its digest belongs in a public journal.
type Evidence struct {
	Type   string `json:"type"`
	SHA256 string `json:"sha256"`
	URI    string `json:"uri"`
}

// Approval is an Ed25519 approval over the transition's approval statement.
// PublicKey and Signature are lowercase hexadecimal. Power is a canonical
// non-negative decimal and is counted only for a declared PowerQuorum role.
type Approval struct {
	Role            string `json:"role"`
	Identity        string `json:"identity"`
	PublicKey       string `json:"public_key"`
	StatementSHA256 string `json:"statement_sha256"`
	Signature       string `json:"signature"`
	Power           string `json:"power"`
}

// TrustPolicy is an out-of-journal trust root. Operators provision its exact
// canonical SHA-256 through a separate trusted channel; a journal may reference
// it but cannot redefine or rotate it.
type TrustPolicy struct {
	Schema           string               `json:"schema"`
	PolicyID         string               `json:"policy_id"`
	ChainID          string               `json:"chain_id"`
	IncidentID       string               `json:"incident_id"`
	ReleaseID        string               `json:"release_id"`
	ApprovalPolicies []EdgeApprovalPolicy `json:"approval_policies"`
	Signers          []TrustedSigner      `json:"signers"`
}

// EdgeApprovalPolicy binds one exact allowed state edge to its own authority
// and quorum. The externally pinned policy must cover every edge in each lane
// it authorizes.
type EdgeApprovalPolicy struct {
	Lane           lane           `json:"lane"`
	From           State          `json:"from"`
	To             State          `json:"to"`
	ApprovalPolicy ApprovalPolicy `json:"approval_policy"`
}

// TrustedSigner pins one stable role/identity/key tuple. Dynamic validator
// power belongs to a checkpoint-bound PowerSnapshot in each gated transition.
type TrustedSigner struct {
	Role      string `json:"role"`
	Identity  string `json:"identity"`
	PublicKey string `json:"public_key"`
}

// ApprovalPolicy declares the quorum enforced from the separately pinned trust
// policy. It is never accepted from a transition journal.
type ApprovalPolicy struct {
	MinimumApprovals          uint64      `json:"minimum_approvals"`
	MinimumDistinctIdentities uint64      `json:"minimum_distinct_identities"`
	RequiredRoles             []string    `json:"required_roles"`
	SeparatedRolePairs        []RolePair  `json:"separated_role_pairs"`
	PowerQuorum               PowerQuorum `json:"power_quorum"`
}

// RolePair prevents one identity or one public key from filling both roles.
// Canonical pairs require RoleA < RoleB.
type RolePair struct {
	RoleA string `json:"role_a"`
	RoleB string `json:"role_b"`
}

// PowerQuorum checks approvals for Role against Numerator/Denominator of the
// transition's fresh PowerSnapshot. The all-zero form disables power checking.
type PowerQuorum struct {
	Role        string `json:"role"`
	Numerator   uint64 `json:"numerator"`
	Denominator uint64 `json:"denominator"`
	Strict      bool   `json:"strict"`
}

// PowerSnapshot is transition-time validator authority. Its exact checkpoint
// and canonical member powers are signed as part of the transition instead of
// being guessed when the long-lived trust root is provisioned.
type PowerSnapshot struct {
	Schema         string                `json:"schema"`
	ChainID        string                `json:"chain_id"`
	Height         uint64                `json:"height"`
	BlockIDSHA256  string                `json:"block_id_sha256"`
	AppHashSHA256  string                `json:"app_hash_sha256"`
	Role           string                `json:"role"`
	TotalPower     string                `json:"total_power"`
	CapturedAt     string                `json:"captured_at"`
	ValidUntil     string                `json:"valid_until"`
	Members        []PowerSnapshotMember `json:"members"`
	SnapshotSHA256 string                `json:"snapshot_sha256"`
}

// PowerSnapshotMember binds one stable trusted operator key to its power at
// the snapshot checkpoint.
type PowerSnapshotMember struct {
	Identity  string `json:"identity"`
	PublicKey string `json:"public_key"`
	Power     string `json:"power"`
}

// VerifyOptions pins a journal to values obtained through a separate trusted
// channel. Empty string options are not enforced.
type VerifyOptions struct {
	ExpectedChainID             string
	ExpectedIncidentID          string
	ExpectedReleaseID           string
	ExpectedBinarySHA256        string
	ExpectedHeadSHA256          string
	ExpectedPowerSnapshotSHA256 map[uint64]string
	TrustPolicy                 *TrustPolicy
	TrustPolicySHA256           string
}

// VerificationResult is the stable summary printed after successful
// verification.
type VerificationResult struct {
	Transitions int
	Lane        lane
	ChainID     string
	IncidentID  string
	ReleaseID   string
	State       State
	HeadSHA256  string
}

// Supersession is a separately versioned, canonical sidecar that links the
// frozen head and trust policy of an interrupted v1 journal to a replacement
// trust policy. It deliberately does not add fields or states to Transition
// v1, so previously sealed journal bytes and approval digests remain valid.
type Supersession struct {
	Schema                string     `json:"schema"`
	ChainID               string     `json:"chain_id"`
	OldJournalHeadSHA256  string     `json:"old_journal_head_sha256"`
	OldTrustPolicySHA256  string     `json:"old_trust_policy_sha256"`
	NewTrustPolicySHA256  string     `json:"new_trust_policy_sha256"`
	ReplacementIncidentID string     `json:"replacement_incident_id"`
	ReplacementReleaseID  string     `json:"replacement_release_id"`
	OccurredAt            string     `json:"occurred_at"`
	Reason                string     `json:"reason"`
	Evidence              []Evidence `json:"evidence"`
	Approvals             []Approval `json:"approvals"`
	SupersessionSHA256    string     `json:"supersession_sha256"`
}

// SupersessionVerificationResult is printed after a sidecar is verified
// against independently pinned old/new policies and the old journal head.
type SupersessionVerificationResult struct {
	ChainID               string
	OldJournalHeadSHA256  string
	OldTrustPolicySHA256  string
	NewTrustPolicySHA256  string
	ReplacementIncidentID string
	ReplacementReleaseID  string
	SupersessionSHA256    string
}

// ReplacementVerificationResult is returned only after the old journal,
// supersession sidecar, and first-to-last replacement journal have all been
// verified as one composed history against independently pinned heads.
type ReplacementVerificationResult struct {
	OldJournal   VerificationResult
	Supersession SupersessionVerificationResult
	NewJournal   VerificationResult
}
