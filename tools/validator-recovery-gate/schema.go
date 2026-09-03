// validator-recovery-gate makes an offline, deterministic decision between a
// controlled validator transition and suspect-signer fork re-genesis.
package main

import "encoding/json"

const (
	custodyAssessmentSchema      = "zerone.ops.validator-custody-assessment/v2"
	controlledTransitionSchema   = "zerone.ops.validator-controlled-transition/v2"
	signerPolicySchema           = "zerone.ops.validator-signer-policy/v2"
	forkPolicySchema             = "zerone.ops.validator-fork-policy/v2"
	forkReleaseSchema            = "zerone.ops.validator-fork-release/v2"
	forkChoiceSchema             = "zerone.ops.validator-fork-choice/v2"
	gateReportSchema             = "zerone.ops.validator-recovery-gate-report/v2"
	forkGenesisReportSchema      = "zerone.fork-genesis.report/v2"
	custodyApprovalDomain        = "zerone.ops.validator-custody-approval/v2\x00"
	controlledApprovalDomain     = "zerone.ops.validator-controlled-approval/v2\x00"
	genesisReproductionDomain    = "zerone.ops.validator-genesis-reproduction/v2\x00"
	forkReleaseApprovalDomain    = "zerone.ops.validator-fork-release-approval/v2\x00"
	forkChoiceApprovalDomain     = "zerone.ops.validator-fork-choice-approval/v2\x00"
	maxInputBytes                = 1 << 20
	maxCompilerReportBytes       = 4 << 20
	maxGenesisBytes              = 256 << 20
	maxCollectionEntries         = 4096
	routeControlled              = "CONTROLLED_TRANSITION"
	routeFork                    = "FORK_REGENESIS"
	decisionControlled           = "GO_CONTROLLED_TRANSITION"
	decisionFork                 = "GO_FORK_REGENESIS"
	decisionNoGo                 = "NO_GO"
	custodyResultPass            = "PASS"
	custodyResultFail            = "FAIL"
	custodyResultUnknown         = "UNKNOWN"
	privilegedDispositionRetain  = "RETAIN"
	privilegedDispositionRetire  = "RETIRE"
	privilegedKindSDKOperator    = "sdk-operator"
	signerPurposeCustody         = "CUSTODY_ASSESSMENT"
	signerPurposeControlled      = "CONTROLLED_TRANSITION"
	rewriteProfileConsensusOnly  = "consensus-key-only"
	operatorRetainProvenSafe     = "RETAIN_PROVEN_SAFE"
	ibcDispositionRequireEmpty   = "require-empty"
	emergencyConsensusQuarantine = "CONSENSUS_QUARANTINE"
	requiredAdmissionState       = "OPEN"
	requiredForkReason           = "SUSPECT_SIGNER_CUSTODY"
	requiredValidatorUpdateWait  = uint64(2)
)

var requiredCustodyFindings = []string{
	"backup-and-export-copies-accounted",
	"build-worker-access-accounted",
	"canonical-history-single",
	"consensus-key-exclusive-control",
	"historical-image-access-accounted",
	"independent-custody-review-complete",
	"no-conflicting-signatures-or-equivocation",
	"no-duplicate-signer-execution",
	"old-governance-authority-safe",
	"old-sdk-operator-key-safe",
	"registry-and-build-cache-access-accounted",
	"runtime-secret-and-host-access-accounted",
}

// The audited consensus-key-only compiler cannot repair ambiguous history or
// unsafe retained governance/operator authority. These facts must remain PASS
// even when another custody fact selects the fork route.
var requiredConsensusOnlySafeFindings = []string{
	"canonical-history-single",
	"old-governance-authority-safe",
	"old-sdk-operator-key-safe",
}

var requiredCustodyRoles = []string{
	"custody-reviewer",
	"evidence-custodian",
}

var requiredControlledRoles = []string{
	"evidence-custodian",
	"incident-commander",
	"release-verifier",
	"validator-operator",
}

var requiredForkRoles = []string{
	"evidence-custodian",
	"fork-authority",
	"ibc-lead",
	"release-verifier",
	"supply-verifier",
}

// Checkpoint pins the last independently verified commit. SignedCommitSHA256
// is a digest of the canonical commit evidence, not a signature copied here.
type Checkpoint struct {
	Height             uint64 `json:"height"`
	BlockTime          string `json:"block_time"`
	BlockIDSHA256      string `json:"block_id_sha256"`
	AppHashSHA256      string `json:"app_hash_sha256"`
	SignedCommitSHA256 string `json:"signed_commit_sha256"`
	ValidatorSetSHA256 string `json:"validator_set_sha256"`
}

// Evidence is always content-addressed. The gate does not open the URI.
type Evidence struct {
	Type   string `json:"type"`
	SHA256 string `json:"sha256"`
	URI    string `json:"uri"`
}

// Approval is an Ed25519 signature over a domain-separated statement digest.
// It deliberately has no power or private-key field.
type Approval struct {
	Role            string `json:"role"`
	Identity        string `json:"identity"`
	ControlDomain   string `json:"control_domain"`
	PublicKey       string `json:"public_key"`
	StatementSHA256 string `json:"statement_sha256"`
	Signature       string `json:"signature"`
}

// ValidatorIdentity contains only public identity and digests of key files.
// Key files themselves must never be given to this tool.
type ValidatorIdentity struct {
	SDKOperatorAddress string `json:"sdk_operator_address"`
	ConsensusPublicKey string `json:"consensus_public_key"`
	ConsensusAddress   string `json:"consensus_address"`
	NodePublicKey      string `json:"node_public_key"`
	NodeID             string `json:"node_id"`
	ValidatorKeySHA256 string `json:"validator_key_sha256"`
	NodeKeySHA256      string `json:"node_key_sha256"`
	SigningStateSHA256 string `json:"signing_state_sha256"`
}

// SignerPolicy is an independently pinned trust root for custody or
// controlled-transition approvals. It is never learned from an approval.
type SignerPolicy struct {
	Schema                        string          `json:"schema"`
	PolicyID                      string          `json:"policy_id"`
	Purpose                       string          `json:"purpose"`
	IncidentID                    string          `json:"incident_id"`
	ChainID                       string          `json:"chain_id"`
	AssessmentSHA256              string          `json:"assessment_sha256"`
	MinimumApprovals              uint64          `json:"minimum_approvals"`
	MinimumDistinctIdentities     uint64          `json:"minimum_distinct_identities"`
	MinimumDistinctControlDomains uint64          `json:"minimum_distinct_control_domains"`
	RequiredRoles                 []string        `json:"required_roles"`
	Signers                       []TrustedSigner `json:"signers"`
}

type ExposureWindow struct {
	FirstPossiblyExposedHeight uint64 `json:"first_possibly_exposed_height"`
	LastReviewedHeight         uint64 `json:"last_reviewed_height"`
}

type CustodyFinding struct {
	ID             string `json:"id"`
	Result         string `json:"result"`
	EvidenceSHA256 string `json:"evidence_sha256"`
}

// PrivilegedIdentityAssessment makes retention an explicit custody decision.
// RETAIN is valid only with PASS. Every FAIL or UNKNOWN identity is retired
// and therefore appears in ProhibitedPrivilegedIdentities.
type PrivilegedIdentityAssessment struct {
	Kind           string `json:"kind"`
	Identity       string `json:"identity"`
	Result         string `json:"result"`
	EvidenceSHA256 string `json:"evidence_sha256"`
	Disposition    string `json:"disposition"`
}

// CustodyAssessment selects the recovery route. Any required finding that is
// absent, FAIL, or UNKNOWN deterministically selects fork re-genesis.
type CustodyAssessment struct {
	Schema                         string                         `json:"schema"`
	AssessmentID                   string                         `json:"assessment_id"`
	SignerPolicySHA256             string                         `json:"signer_policy_sha256"`
	IncidentID                     string                         `json:"incident_id"`
	ChainID                        string                         `json:"chain_id"`
	EvaluatedAt                    string                         `json:"evaluated_at"`
	Checkpoint                     Checkpoint                     `json:"checkpoint"`
	ExposureWindow                 ExposureWindow                 `json:"exposure_window"`
	OldValidator                   ValidatorIdentity              `json:"old_validator"`
	PrivilegedIdentityAssessments  []PrivilegedIdentityAssessment `json:"privileged_identity_assessments"`
	Findings                       []CustodyFinding               `json:"findings"`
	Evidence                       []Evidence                     `json:"evidence"`
	ProhibitedConsensusPublicKeys  []string                       `json:"prohibited_consensus_public_keys"`
	ProhibitedPrivilegedIdentities []string                       `json:"prohibited_privileged_identities"`
	Approvals                      []Approval                     `json:"approvals"`
	AssessmentSHA256               string                         `json:"assessment_sha256"`
}

type PaginatedInventory struct {
	PageSHA256s []string `json:"page_sha256s"`
	NextKey     string   `json:"next_key"`
	Complete    bool     `json:"complete"`
}

type StakeInventory struct {
	Delegations   PaginatedInventory `json:"delegations"`
	Unbondings    PaginatedInventory `json:"unbondings"`
	Redelegations PaginatedInventory `json:"redelegations"`
}

// ControlledTransition is usable only when custody is entirely PASS. The
// consent power is checked as a strict fraction greater than two thirds.
type ControlledTransition struct {
	Schema                    string            `json:"schema"`
	PlanID                    string            `json:"plan_id"`
	AssessmentSHA256          string            `json:"assessment_sha256"`
	SignerPolicySHA256        string            `json:"signer_policy_sha256"`
	IncidentID                string            `json:"incident_id"`
	ChainID                   string            `json:"chain_id"`
	Checkpoint                Checkpoint        `json:"checkpoint"`
	AdmissionState            string            `json:"admission_state"`
	BondTransactionHeight     uint64            `json:"bond_transaction_height"`
	ExpectedActivationHeight  uint64            `json:"expected_activation_height"`
	ValidatorUpdateWaitBlocks uint64            `json:"validator_update_wait_blocks"`
	ConsentingPower           string            `json:"consenting_power"`
	TotalBondedPower          string            `json:"total_bonded_power"`
	PowerSnapshotSHA256       string            `json:"power_snapshot_sha256"`
	StakeInventory            StakeInventory    `json:"stake_inventory"`
	NewValidator              ValidatorIdentity `json:"new_validator"`
	BinarySHA256              string            `json:"binary_sha256"`
	ImageSHA256               string            `json:"image_sha256"`
	ProvenanceSHA256          string            `json:"provenance_sha256"`
	SBOMSHA256                string            `json:"sbom_sha256"`
	RehearsalSHA256           string            `json:"rehearsal_sha256"`
	TopologySHA256            string            `json:"topology_sha256"`
	JournalHeadSHA256         string            `json:"journal_head_sha256"`
	Evidence                  []Evidence        `json:"evidence"`
	Approvals                 []Approval        `json:"approvals"`
	PlanSHA256                string            `json:"plan_sha256"`
}

type TrustedSigner struct {
	Role          string `json:"role"`
	Identity      string `json:"identity"`
	ControlDomain string `json:"control_domain"`
	PublicKey     string `json:"public_key"`
}

type TrustedReproducer struct {
	Identity      string `json:"identity"`
	ControlDomain string `json:"control_domain"`
	PublicKey     string `json:"public_key"`
}

// ForkPolicy is an out-of-band trust root. It has no self hash: the exact
// canonical file digest must be supplied separately to every command.
type ForkPolicy struct {
	Schema                         string              `json:"schema"`
	PolicyID                       string              `json:"policy_id"`
	AssessmentSHA256               string              `json:"assessment_sha256"`
	IncidentID                     string              `json:"incident_id"`
	OldChainID                     string              `json:"old_chain_id"`
	MinimumApprovals               uint64              `json:"minimum_approvals"`
	MinimumDistinctIdentities      uint64              `json:"minimum_distinct_identities"`
	MinimumDistinctControlDomains  uint64              `json:"minimum_distinct_control_domains"`
	RequiredRoles                  []string            `json:"required_roles"`
	Signers                        []TrustedSigner     `json:"signers"`
	IndependentReproducers         []TrustedReproducer `json:"independent_reproducers"`
	ProhibitedConsensusPublicKeys  []string            `json:"prohibited_consensus_public_keys"`
	ProhibitedPrivilegedIdentities []string            `json:"prohibited_privileged_identities"`
}

type GenesisReproduction struct {
	Identity                 string `json:"identity"`
	ControlDomain            string `json:"control_domain"`
	PublicKey                string `json:"public_key"`
	GenesisSHA256            string `json:"genesis_sha256"`
	CompilerReportFileSHA256 string `json:"compiler_report_file_sha256"`
	StatementSHA256          string `json:"statement_sha256"`
	Signature                string `json:"signature"`
}

// ForkRelease is the independently reproduced, reconciled re-genesis bundle.
type ForkRelease struct {
	Schema                      string                `json:"schema"`
	ReleaseID                   string                `json:"release_id"`
	AssessmentSHA256            string                `json:"assessment_sha256"`
	ForkPolicySHA256            string                `json:"fork_policy_sha256"`
	IncidentID                  string                `json:"incident_id"`
	OldChainID                  string                `json:"old_chain_id"`
	NewChainID                  string                `json:"new_chain_id"`
	Checkpoint                  Checkpoint            `json:"checkpoint"`
	InitialHeight               uint64                `json:"initial_height"`
	RewriteProfile              string                `json:"rewrite_profile"`
	SourceExportSHA256          string                `json:"source_export_sha256"`
	RewriteToolSHA256           string                `json:"rewrite_tool_sha256"`
	RewritePolicyFileSHA256     string                `json:"rewrite_policy_file_sha256"`
	RewritePolicySelfSHA256     string                `json:"rewrite_policy_self_sha256"`
	GenesisSHA256               string                `json:"genesis_sha256"`
	GenesisReproductions        []GenesisReproduction `json:"genesis_reproductions"`
	NewValidators               []ValidatorIdentity   `json:"new_validators"`
	RetiredPrivilegedIdentities []string              `json:"retired_privileged_identities"`
	SupplyReconciliationSHA256  string                `json:"supply_reconciliation_sha256"`
	IBCReconciliationSHA256     string                `json:"ibc_reconciliation_sha256"`
	ModuleReconciliationSHA256  string                `json:"module_reconciliation_sha256"`
	BinarySHA256                string                `json:"binary_sha256"`
	ImageSHA256                 string                `json:"image_sha256"`
	ProvenanceSHA256            string                `json:"provenance_sha256"`
	SBOMSHA256                  string                `json:"sbom_sha256"`
	RehearsalSHA256             string                `json:"rehearsal_sha256"`
	TopologySHA256              string                `json:"topology_sha256"`
	JournalHeadSHA256           string                `json:"journal_head_sha256"`
	Evidence                    []Evidence            `json:"evidence"`
	Approvals                   []Approval            `json:"approvals"`
	ReleaseSHA256               string                `json:"release_sha256"`
}

// ForkChoice records the explicit selection of one already sealed release.
type ForkChoice struct {
	Schema            string     `json:"schema"`
	ChoiceID          string     `json:"choice_id"`
	AssessmentSHA256  string     `json:"assessment_sha256"`
	ForkPolicySHA256  string     `json:"fork_policy_sha256"`
	ForkReleaseSHA256 string     `json:"fork_release_sha256"`
	IncidentID        string     `json:"incident_id"`
	OldChainID        string     `json:"old_chain_id"`
	NewChainID        string     `json:"new_chain_id"`
	ReasonCode        string     `json:"reason_code"`
	Approvals         []Approval `json:"approvals"`
	ChoiceSHA256      string     `json:"choice_sha256"`
}

type GateInputDigests struct {
	CustodyPolicySHA256    string   `json:"custody_policy_sha256"`
	AssessmentSHA256       string   `json:"assessment_sha256"`
	ControlledPolicySHA256 string   `json:"controlled_policy_sha256"`
	ControlledSHA256       string   `json:"controlled_sha256"`
	ForkPolicySHA256       string   `json:"fork_policy_sha256"`
	ForkReleaseSHA256      string   `json:"fork_release_sha256"`
	ForkChoiceSHA256       string   `json:"fork_choice_sha256"`
	GenesisSHA256          string   `json:"genesis_sha256"`
	CompilerReportSHA256s  []string `json:"compiler_report_sha256s"`
}

// GateReport is deterministic and self-hashed. It never contains evidence
// payloads, key-file contents, approval signatures, or public-key material.
type GateReport struct {
	Schema         string           `json:"schema"`
	GateID         string           `json:"gate_id"`
	EvaluatedAt    string           `json:"evaluated_at"`
	IncidentID     string           `json:"incident_id"`
	OldChainID     string           `json:"old_chain_id"`
	RequiredRoute  string           `json:"required_route"`
	Decision       string           `json:"decision"`
	ReasonCodes    []string         `json:"reason_codes"`
	InputDigests   GateInputDigests `json:"input_digests"`
	SelectedSHA256 string           `json:"selected_sha256"`
	ReportSHA256   string           `json:"report_sha256"`
}

type ForkGenesisModuleDigest struct {
	Module       string `json:"module"`
	BeforeSHA256 string `json:"before_sha256"`
	AfterSHA256  string `json:"after_sha256"`
	Changed      bool   `json:"changed"`
}

// ForkGenesisReport mirrors the exact standard emitted by tools/fork-genesis.
type ForkGenesisReport struct {
	Schema                   string                    `json:"schema"`
	Profile                  string                    `json:"profile"`
	ReproducerID             string                    `json:"reproducer_id"`
	ReproducerControlDomain  string                    `json:"reproducer_control_domain"`
	ReproducerPublicKey      string                    `json:"reproducer_public_key"`
	IncidentID               string                    `json:"incident_id"`
	SourceGenesisSHA256      string                    `json:"source_genesis_sha256"`
	PolicyFileSHA256         string                    `json:"policy_file_sha256"`
	PolicySHA256             string                    `json:"policy_sha256"`
	SourceChainID            string                    `json:"source_chain_id"`
	TargetChainID            string                    `json:"target_chain_id"`
	InitialHeight            uint64                    `json:"initial_height"`
	SourceBlockIDSHA256      string                    `json:"source_block_id_sha256"`
	SourceAppHashSHA256      string                    `json:"source_app_hash_sha256"`
	SourceLastBlockTime      string                    `json:"source_last_block_time"`
	SourceSignedCommitSHA256 string                    `json:"source_signed_commit_sha256"`
	SourceValidatorSetSHA256 string                    `json:"source_validator_set_sha256"`
	OldConsensusAddress      string                    `json:"old_consensus_address"`
	NewConsensusAddress      string                    `json:"new_consensus_address"`
	OldConsensusPublicKey    string                    `json:"old_consensus_public_key"`
	NewConsensusPublicKey    string                    `json:"new_consensus_public_key"`
	OperatorDisposition      string                    `json:"operator_disposition"`
	CustodyAssessmentSHA256  string                    `json:"custody_assessment_sha256"`
	ForkPolicySHA256         string                    `json:"fork_policy_sha256"`
	RewriteToolSHA256        string                    `json:"rewrite_tool_sha256"`
	IBCDisposition           string                    `json:"ibc_disposition"`
	EmergencyStartMode       string                    `json:"emergency_start_mode"`
	SchemaMigrations         []string                  `json:"schema_migrations"`
	ModuleDigests            []ForkGenesisModuleDigest `json:"module_digests"`
	OutputGenesisSHA256      string                    `json:"output_genesis_sha256"`
	ReportSHA256             string                    `json:"report_sha256"`
}

// ForkGenesis keeps large app-state and consensus payloads as exact raw JSON
// while validating their identity-bearing portions separately.
type ForkGenesis struct {
	AppName       string          `json:"app_name"`
	AppVersion    string          `json:"app_version"`
	GenesisTime   string          `json:"genesis_time"`
	ChainID       string          `json:"chain_id"`
	InitialHeight int64           `json:"initial_height"`
	AppHash       json.RawMessage `json:"app_hash"`
	AppState      json.RawMessage `json:"app_state"`
	Consensus     json.RawMessage `json:"consensus"`
}
