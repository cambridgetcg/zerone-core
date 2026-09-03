package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

const (
	manifestPath            = "dashboard/public/standards/authority-geometry.v1.json"
	manifestSchema          = "zerone.authority-geometry/v1"
	checkerSchema           = "zerone.authority-graph-check/v1"
	canonicalManifestSHA256 = "b5f7c6bc8e897b2431688229141409224a13ab39cd57b3e408ba8ad008dbd7d6"
	canonicalDesignSHA256   = "22d523ee25060957e2c93aba441542e35d767f28f0f0e5e86c800f5fd7ea82e9"
)

type manifest struct {
	Schema              string               `json:"schema"`
	Revision            string               `json:"revision"`
	SnapshotDate        string               `json:"snapshotDate"`
	Status              string               `json:"status"`
	Title               string               `json:"title"`
	Summary             string               `json:"summary"`
	SourceDesign        sourceDesign         `json:"sourceDesign"`
	ObservationScopes   []observationScope   `json:"observationScopes"`
	CurrentTruth        currentTruth         `json:"currentTruth"`
	ReleaseBoundary     releaseBoundary      `json:"releaseBoundary"`
	Principles          []principle          `json:"principles"`
	Capabilities        []capability         `json:"capabilities"`
	Nodes               []authorityNode      `json:"nodes"`
	Edges               []authorityEdge      `json:"edges"`
	ForbiddenInfluence  []forbiddenInfluence `json:"forbiddenInfluence"`
	SourceAnchors       []sourceAnchor       `json:"sourceAnchors"`
	CurrentFindings     []currentFinding     `json:"currentFindings"`
	StaticAuthorityGate staticAuthorityGate  `json:"staticAuthorityGate"`
	ActivationGates     activationGates      `json:"activationGates"`
	ReleaseAssessment   releaseAssessment    `json:"releaseAssessment"`
}

type sourceDesign struct {
	RepositoryPath string `json:"repositoryPath"`
	SHA256         string `json:"sha256"`
	DecisionDate   string `json:"decisionDate"`
	Status         string `json:"status"`
}

type observationScope struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type currentTruth struct {
	Scope                                             string `json:"scope"`
	LiveEvidence                                      string `json:"liveEvidence"`
	NetworkNamedForContext                            string `json:"networkNamedForContext"`
	SourceHasDualAuthoritySurfaces                    *bool  `json:"sourceHasDualAuthoritySurfaces"`
	SourceUsesBondedWealthForSDKGovernance            *bool  `json:"sourceUsesBondedWealthForSdkGovernance"`
	DisclosedFoundingHouseholdRetainsEffectiveControl *bool  `json:"disclosedFoundingHouseholdRetainsEffectiveControl"`
	SourceTargetModulesComplete                       *bool  `json:"sourceTargetModulesComplete"`
	SourceAuthorityUnified                            *bool  `json:"sourceAuthorityUnified"`
	SourceRegistersH4FreezeHandler                    *bool  `json:"sourceRegistersH4FreezeHandler"`
	SourceRegistersH4UnificationHandler               *bool  `json:"sourceRegistersH4UnificationHandler"`
	SourceRegistersH5NonEconomicGovernanceHandler     *bool  `json:"sourceRegistersH5NonEconomicGovernanceHandler"`
}

type releaseBoundary struct {
	AddsConsensusBehavior       *bool `json:"addsConsensusBehavior"`
	RegistersUpgradeHandler     *bool `json:"registersUpgradeHandler"`
	SchedulesUpgrade            *bool `json:"schedulesUpgrade"`
	ChangesValidatorState       *bool `json:"changesValidatorState"`
	ChangesGovernanceState      *bool `json:"changesGovernanceState"`
	ChangesDomainState          *bool `json:"changesDomainState"`
	MovesFunds                  *bool `json:"movesFunds"`
	GrantsQualification         *bool `json:"grantsQualification"`
	CreatesRewardOrKarma        *bool `json:"createsRewardOrKarma"`
	SubmitsTransaction          *bool `json:"submitsTransaction"`
	AssertsDecentralization     *bool `json:"assertsDecentralization"`
	EstablishesLiveNetworkState *bool `json:"establishesLiveNetworkState"`
}

type principle struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Rule  string `json:"rule"`
}

type capability struct {
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	TargetWriter         string   `json:"targetWriter"`
	CurrentStateSurfaces []string `json:"currentStateSurfaces"`
}

type authorityNode struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	Module             string   `json:"module"`
	Role               string   `json:"role"`
	Implementation     string   `json:"implementation"`
	TargetCapabilities []string `json:"targetCapabilities"`
}

type authorityEdge struct {
	ID           string          `json:"id"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Relationship string          `json:"relationship"`
	Scope        string          `json:"scope"`
	Effect       string          `json:"effect"`
	Evidence     []string        `json:"evidence"`
	Protections  edgeProtections `json:"protections"`
}

type edgeProtections struct {
	ConsentBoundary string `json:"consentBoundary"`
	ChallengeRoute  string `json:"challengeRoute"`
	RepairRoute     string `json:"repairRoute"`
	ExitRoute       string `json:"exitRoute"`
}

type forbiddenInfluence struct {
	ID                      string   `json:"id"`
	Label                   string   `json:"label"`
	Sources                 []string `json:"sources"`
	Targets                 []string `json:"targets"`
	AcceptedTargetPathCount int      `json:"acceptedTargetPathCount"`
}

type sourceAnchor struct {
	ID                string   `json:"id"`
	Path              string   `json:"path"`
	SHA256            string   `json:"sha256"`
	RequiredSnippets  []string `json:"requiredSnippets"`
	ForbiddenSnippets []string `json:"forbiddenSnippets"`
}

type currentFinding struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	ReleaseSurface string   `json:"releaseSurface"`
	Nodes          []string `json:"nodes"`
	Evidence       []string `json:"evidence"`
}

type staticAuthorityGate struct {
	AuthoritativeStateReleaseGate string         `json:"authoritativeStateReleaseGate"`
	Status                        string         `json:"status"`
	Mode                          string         `json:"mode"`
	SurfaceChecks                 []surfaceCheck `json:"surfaceChecks"`
}

type surfaceCheck struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	FindingIDs []string `json:"findingIds"`
}

type activationGates struct {
	H4 []activationGate `json:"h4"`
	H5 []activationGate `json:"h5"`
}

type activationGate struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type releaseAssessment struct {
	Overall                        string `json:"overall"`
	StaticAuthoritySurfacesPassing int    `json:"staticAuthoritySurfacesPassing"`
	StaticAuthoritySurfacesTotal   int    `json:"staticAuthoritySurfacesTotal"`
	H4GatesEvidenced               int    `json:"h4GatesEvidenced"`
	H4GatesTotal                   int    `json:"h4GatesTotal"`
	H5GatesEvidenced               int    `json:"h5GatesEvidenced"`
	H5GatesTotal                   int    `json:"h5GatesTotal"`
	TargetGateMustExitNonZero      *bool  `json:"targetGateMustExitNonZero"`
}

type issue struct {
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

type issueSet struct {
	items []issue
}

func (s *issueSet) add(id, detail string) {
	s.items = append(s.items, issue{ID: id, Detail: detail})
}

func (s *issueSet) sorted() []issue {
	items := append([]issue(nil), s.items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID == items[j].ID {
			return items[i].Detail < items[j].Detail
		}
		return items[i].ID < items[j].ID
	})
	return items
}

type expectedAnchor struct {
	ID        string
	Path      string
	Required  []string
	Forbidden []string
}

var expectedAnchors = []expectedAnchor{
	{
		ID:   "authoritative-state-design",
		Path: "docs/AUTHORITATIVE-STATE.md",
		Required: []string{
			"Zerone adopts three single-writer authority boundaries:",
			"A static authority-graph check finds no alternate validator-update,",
			"current source has dual custom/SDK authority surfaces",
		},
		Forbidden: []string{},
	},
	{
		ID:   "app-wiring",
		Path: "app/app.go",
		Required: []string{
			"app.StakingKeeper = stakingkeeper.NewKeeper(",
			"app.GovKeeper = govkeeper.NewKeeper(",
			"app.ZeroneStakingKeeper = zeronestakingkeeper.NewKeeper(",
			"govStakingAdapter := zeronestakingkeeper.NewGovStakingKeeperAdapter(app.ZeroneStakingKeeper)",
			"app.ZeroneGovKeeper = zeronegovkeeper.NewKeeper(",
			"app.ZeroneOntologyKeeper = zeroneontologykeeper.NewKeeper(",
			"app.KnowledgeKeeper = zeroneknowledgekeeper.NewKeeper(",
			"stakingAdapter := zeronestakingkeeper.NewStakingKeeperAdapter(app.ZeroneStakingKeeper)",
			"qualificationStakingAdapter := zeronestakingkeeper.NewQualificationStakingKeeperAdapter(app.ZeroneStakingKeeper)",
			"emergencyStakingAdapter := zeronestakingkeeper.NewEmergencyStakingAdapter(app.ZeroneStakingKeeper)",
			"alignmentStakingAdapter := zeronestakingkeeper.NewAlignmentStakingAdapter(app.ZeroneStakingKeeper)",
			"cpotStakingAdapter := zeronestakingkeeper.NewClaimingPotStakingAdapter(app.ZeroneStakingKeeper)",
		},
		Forbidden: []string{"AppendSendRestriction(ResearchFundRestriction)"},
	},
	{
		ID:   "custom-gov-resolution",
		Path: "x/gov/keeper/abci.go",
		Required: []string{
			"lip.Stage = types.StatusPassed",
			"func (k Keeper) executeParamChanges(ctx sdk.Context, lip *types.LIP)",
			"param router not set, skipping param change",
			"param change failed",
		},
		Forbidden: []string{},
	},
	{
		ID:   "custom-research-spend",
		Path: "x/gov/keeper/research_spend.go",
		Required: []string{
			"func (k Keeper) SubmitResearchSpend(ctx sdk.Context, msg *types.MsgSubmitResearchSpend)",
			"func (k Keeper) VoteResearchSpend(ctx sdk.Context, msg *types.MsgVoteResearchSpend)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "knowledge-writers",
		Path: "x/knowledge/keeper/msg_server.go",
		Required: []string{
			"func (m *msgServer) AddFact(ctx context.Context, msg *types.MsgAddFact)",
			"func (m *msgServer) ProposeDomain(ctx context.Context, msg *types.MsgProposeDomain)",
			"func (m *msgServer) EndorseDomainProposal(ctx context.Context, msg *types.MsgEndorseDomainProposal)",
			"Immediate path (no guardian veto configured)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "knowledge-pause-writer",
		Path: "x/knowledge/keeper/resilience.go",
		Required: []string{
			"func (m *msgServer) PauseModule(ctx context.Context, msg *types.MsgPauseModule)",
			"func (m *msgServer) UnpauseModule(ctx context.Context, msg *types.MsgUnpauseModule)",
			"only governance authority may pause modules",
		},
		Forbidden: []string{},
	},
	{
		ID:   "ontology-writer",
		Path: "x/ontology/keeper/msg_server.go",
		Required: []string{
			"func (k msgServer) ProposeDomain(goCtx context.Context, msg *types.MsgProposeDomain)",
			"func (k msgServer) UpdateDomain(goCtx context.Context, msg *types.MsgUpdateDomain)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "research-router-restriction",
		Path: "app/research_fund_restriction.go",
		Required: []string{
			"func ResearchFundRestriction(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins)",
			"must route through DepositToResearchFund",
		},
		Forbidden: []string{},
	},
	{
		ID:   "custom-emergency-electorate",
		Path: "x/emergency/keeper/keeper.go",
		Required: []string{
			"func (k Keeper) IsGuardian(ctx context.Context, operatorAddr string) bool",
			"func (k Keeper) GetGuardianStake(ctx context.Context) *big.Int",
			"k.stakingKeeper.GetGuardianValidators(ctx)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "knowledge-staking-adapter",
		Path: "x/staking/keeper/knowledge_adapters.go",
		Required: []string{
			"func (a *StakingKeeperAdapter) GetEffectiveStake(ctx context.Context, addr string)",
			"func (a *StakingKeeperAdapter) SlashValidator(ctx context.Context, addr string, slashBps uint64) error",
			"func (a *StakingKeeperAdapter) SlashValidatorToModule(ctx context.Context, addr string, slashBps uint64, destModule string)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "knowledge-slash-callsites",
		Path: "x/knowledge/keeper/rounds.go",
		Required: []string{
			"for _, slash := range result.Slashes {",
			"slashedAmt, err := k.stakingKeeper.SlashValidatorToModule(ctx, slash.Verifier, slash.SlashBps, types.VindicationEscrowModuleName)",
			"_ = k.stakingKeeper.SlashValidator(ctx, slash.Verifier, slash.SlashBps)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "alignment-staking-adapter",
		Path: "x/staking/keeper/alignment_adapters.go",
		Required: []string{
			"func NewAlignmentStakingAdapter(k Keeper) *AlignmentStakingAdapter",
			"func (a *AlignmentStakingAdapter) GetTotalStaked(ctx context.Context) *big.Int",
			"func (a *AlignmentStakingAdapter) GetActiveValidatorCount(ctx context.Context) uint64",
		},
		Forbidden: []string{},
	},
	{
		ID:   "alignment-staking-callsites",
		Path: "x/alignment/keeper/sensors.go",
		Required: []string{
			"totalStaked := k.stakingKeeper.GetTotalStaked(ctx)",
			"active := k.stakingKeeper.GetActiveValidatorCount(ctx)",
			"target := k.stakingKeeper.GetTargetValidatorCount(ctx)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "claiming-pot-staking-adapter",
		Path: "x/staking/keeper/claiming_pot_adapters.go",
		Required: []string{
			"func NewClaimingPotStakingAdapter(k Keeper) *ClaimingPotStakingAdapter",
			"func (a *ClaimingPotStakingAdapter) GetValidatorTier(ctx context.Context, addr string) (uint32, error)",
		},
		Forbidden: []string{},
	},
	{
		ID:   "claiming-pot-staking-callsite",
		Path: "x/claiming_pot/keeper/msg_server.go",
		Required: []string{
			"if elig.MinStakingTier > 0 {",
			"tier, err := m.stakingKeeper.GetValidatorTier(ctx, claimant)",
			"if tier < elig.MinStakingTier {",
		},
		Forbidden: []string{},
	},
	{
		ID:   "custom-gov-quarantine-wrapper",
		Path: "app/emergency_custom_gov.go",
		Required: []string{
			"type emergencyAwareCustomGovAppModule struct {",
			"func (am emergencyAwareCustomGovAppModule) BeginBlock(",
			"hold, changed, err := am.keeper.EnsureEmergencyTransitionHold(",
			"emitCustomGovFrozen(ctx, hold, \"application_transaction_quarantine\")",
		},
		Forbidden: []string{},
	},
	{
		ID:   "custom-gov-emergency-hold-writer",
		Path: "x/gov/keeper/emergency_transition_hold.go",
		Required: []string{
			"func (k Keeper) RequireNoEmergencyTransitionHold(ctx sdk.Context) error",
			"func (k Keeper) SetEmergencyTransitionHold(",
			"ctx.KVStore(k.storeKey).Set(types.EmergencyTransitionHoldKey, bz)",
			"func (k Keeper) EnsureEmergencyTransitionHold(",
		},
		Forbidden: []string{},
	},
}

type expectedCapability struct {
	ID              string
	Writer          string
	CurrentSurfaces []string
}

var expectedCapabilities = []expectedCapability{
	{"consensus-stake", "sdk-staking", []string{"sdk-staking", "custom-staking"}},
	{"account-key-binding", "zerone-auth", []string{"zerone-auth"}},
	{"controller-identity", "controller", []string{}},
	{"verifier-profile", "verifier-profile", []string{"custom-staking", "zerone-auth"}},
	{"domain-qualification", "qualification", []string{"qualification"}},
	{"knowledge-evidence", "knowledge", []string{"knowledge"}},
	{"ordinary-governance", "sdk-gov", []string{"sdk-gov", "custom-gov"}},
	{"electorate-policy", "electorate", []string{}},
	{"domain-registry", "ontology", []string{"ontology", "knowledge"}},
	{"legacy-claims", "legacy-claims", []string{}},
	{"research-fund-egress", "vesting-rewards", []string{"custom-gov"}},
	{"emergency-quarantine", "emergency", []string{"emergency", "custom-gov", "knowledge"}},
}

type expectedNode struct {
	ID             string
	Module         string
	Role           string
	Implementation string
	Capabilities   []string
}

var expectedNodes = []expectedNode{
	{"bank-balance", "x/bank", "ECONOMIC_INPUT", "PRESENT_SOURCE", []string{}},
	{"sdk-staking", "cosmos/staking", "CANONICAL_WRITER", "PRESENT_AND_TARGET", []string{"consensus-stake"}},
	{"custom-staking", "x/zerone_staking", "LEGACY_WRITER", "PRESENT_SOURCE", []string{}},
	{"sdk-gov", "cosmos/gov", "CANONICAL_WRITER", "PRESENT_AND_TARGET", []string{"ordinary-governance"}},
	{"custom-gov", "x/gov", "LEGACY_WRITER", "PRESENT_SOURCE", []string{}},
	{"ontology", "x/ontology", "CANONICAL_WRITER", "PRESENT_AND_TARGET", []string{"domain-registry"}},
	{"knowledge", "x/knowledge", "BOUNDED_WRITER", "PRESENT_AND_TARGET", []string{"knowledge-evidence"}},
	{"zerone-auth", "x/auth", "BOUNDED_WRITER", "PRESENT_AND_TARGET", []string{"account-key-binding"}},
	{"controller", "x/controller", "TARGET_MODULE", "TARGET_ONLY", []string{"controller-identity"}},
	{"verifier-profile", "x/verifier_profile", "TARGET_MODULE", "TARGET_ONLY", []string{"verifier-profile"}},
	{"qualification", "x/qualification", "BOUNDED_WRITER", "PRESENT_REQUIRES_REDESIGN", []string{"domain-qualification"}},
	{"electorate", "x/electorate", "TARGET_MODULE", "TARGET_ONLY", []string{"electorate-policy"}},
	{"legacy-claims", "x/legacy_claims", "TARGET_MODULE", "TARGET_ONLY", []string{"legacy-claims"}},
	{"vesting-rewards", "x/vesting_rewards", "BOUNDED_WRITER", "PRESENT_REQUIRES_REDESIGN", []string{"research-fund-egress"}},
	{"emergency", "x/emergency", "BOUNDED_WRITER", "PRESENT_REQUIRES_REDESIGN", []string{"emergency-quarantine"}},
	{"alignment", "x/alignment", "CURRENT_CONSUMER", "PRESENT_SOURCE", []string{}},
	{"claiming-pot", "x/claiming_pot", "CURRENT_CONSUMER", "PRESENT_SOURCE", []string{}},
	{"karma", "priced-coherence register", "EVIDENCE_ONLY", "SOURCE_CONSTITUTION_ONLY", []string{}},
}

type expectedEdge struct {
	ID           string
	From         string
	To           string
	Relationship string
	Effect       string
	Evidence     []string
}

var expectedCurrentEdges = []expectedEdge{
	{"current-liquid-to-sdk-stake", "bank-balance", "sdk-staking", "FUNDS_DELEGATION", "ECONOMIC_INFLUENCE", []string{"app-wiring"}},
	{"current-sdk-stake-to-sdk-policy", "sdk-staking", "sdk-gov", "SUPPLIES_VOTING_POWER", "ECONOMIC_INFLUENCE", []string{"app-wiring"}},
	{"current-liquid-to-custom-stake", "bank-balance", "custom-staking", "FUNDS_LEGACY_DELEGATION", "ECONOMIC_INFLUENCE", []string{"app-wiring"}},
	{"current-custom-stake-to-custom-policy", "custom-staking", "custom-gov", "SUPPLIES_LIP_VOTING_POWER", "ECONOMIC_INFLUENCE", []string{"app-wiring", "custom-gov-resolution"}},
	{"current-custom-stake-to-knowledge", "custom-staking", "knowledge", "SUPPLIES_SELECTION_INPUTS", "EPISTEMIC_INFLUENCE", []string{"app-wiring", "knowledge-staking-adapter"}},
	{"current-knowledge-to-custom-stake-slash", "knowledge", "custom-staking", "EXECUTES_VERIFIER_SLASH_AND_COIN_ROUTE", "VALUE_WRITE", []string{"app-wiring", "knowledge-staking-adapter", "knowledge-slash-callsites"}},
	{"current-custom-stake-to-qualification", "custom-staking", "qualification", "GRANTS_STAKE_PATHWAY", "EPISTEMIC_INFLUENCE", []string{"app-wiring"}},
	{"current-custom-stake-to-emergency", "custom-staking", "emergency", "SUPPLIES_GUARDIAN_POWER", "CONTROL_INFLUENCE", []string{"app-wiring", "custom-emergency-electorate"}},
	{"current-custom-stake-to-alignment", "custom-staking", "alignment", "SUPPLIES_ALIGNMENT_HEALTH_INPUTS", "SYSTEM_SIGNAL_INFLUENCE", []string{"app-wiring", "alignment-staking-adapter", "alignment-staking-callsites"}},
	{"current-custom-stake-to-claiming-pot", "custom-staking", "claiming-pot", "GATES_CLAIM_ELIGIBILITY", "VALUE_INFLUENCE", []string{"app-wiring", "claiming-pot-staking-adapter", "claiming-pot-staking-callsite"}},
	{"current-custom-policy-to-research", "custom-gov", "vesting-rewards", "LEGACY_RESEARCH_DISBURSEMENT", "VALUE_WRITE", []string{"custom-research-spend", "research-router-restriction", "app-wiring"}},
	{"current-sdk-policy-to-knowledge-fact", "sdk-gov", "knowledge", "AUTHENTICATES_DIRECT_FACT_ADOPTION", "KNOWLEDGE_WRITE", []string{"knowledge-writers", "app-wiring"}},
}

var expectedTargetEdges = []expectedEdge{
	{"target-auth-to-controller", "zerone-auth", "controller", "AUTHENTICATES_BINDING", "IDENTITY_RELATION", []string{"authoritative-state-design"}},
	{"target-controller-to-profile", "controller", "verifier-profile", "RESOLVES_PROFILE", "IDENTITY_RELATION", []string{"authoritative-state-design"}},
	{"target-profile-to-qualification", "verifier-profile", "qualification", "SCOPES_COMPETENCE", "ELIGIBILITY_RELATION", []string{"authoritative-state-design"}},
	{"target-ontology-to-qualification", "ontology", "qualification", "REVISION_PINS_DOMAIN", "REFERENCE_RELATION", []string{"authoritative-state-design"}},
	{"target-qualification-to-knowledge", "qualification", "knowledge", "ADMITS_PANEL_ELIGIBILITY", "ELIGIBILITY_RELATION", []string{"authoritative-state-design"}},
	{"target-controller-to-electorate", "controller", "electorate", "DEDUPLICATES_VOICE", "CONTROL_RELATION", []string{"authoritative-state-design"}},
	{"target-electorate-to-sdk-policy", "electorate", "sdk-gov", "SNAPSHOTS_NON_ECONOMIC_BALLOTS", "CONTROL_RELATION", []string{"authoritative-state-design"}},
	{"target-sdk-policy-to-ontology", "sdk-gov", "ontology", "TYPED_ATOMIC_EXECUTION", "DOMAIN_WRITE", []string{"authoritative-state-design"}},
	{"target-sdk-policy-to-research", "sdk-gov", "vesting-rewards", "SCOPED_TREASURY_EXECUTION", "VALUE_WRITE", []string{"authoritative-state-design"}},
	{"target-ontology-to-knowledge", "ontology", "knowledge", "ADMITS_BY_REVISION_REFERENCE", "REFERENCE_RELATION", []string{"authoritative-state-design"}},
	{"target-knowledge-to-profile", "knowledge", "verifier-profile", "EMITS_REPLAY_PROTECTED_RECEIPT", "EVIDENCE_RELATION", []string{"authoritative-state-design"}},
	{"target-emergency-to-sdk-policy", "emergency", "sdk-gov", "BOUNDED_INCIDENT_CAPABILITY", "CONTROL_RELATION", []string{"authoritative-state-design"}},
	{"target-custom-stake-to-legacy-exit", "custom-staking", "legacy-claims", "RECONCILES_TO_CLAIMANT_EXIT", "RETIREMENT_RELATION", []string{"authoritative-state-design"}},
	{"target-knowledge-domain-to-ontology", "knowledge", "ontology", "RECONCILES_LEGACY_DOMAIN", "RETIREMENT_RELATION", []string{"authoritative-state-design"}},
}

type expectedForbiddenRule struct {
	ID      string
	Sources []string
	Targets []string
}

var expectedForbiddenRules = []expectedForbiddenRule{
	{"wealth-to-policy", []string{"bank-balance", "sdk-staking", "custom-staking"}, []string{"sdk-gov", "custom-gov", "electorate"}},
	{"wealth-to-qualification", []string{"bank-balance", "sdk-staking", "custom-staking"}, []string{"qualification"}},
	{"wealth-to-truth-power", []string{"bank-balance", "sdk-staking", "custom-staking"}, []string{"knowledge"}},
	{"karma-to-authority", []string{"karma"}, []string{"bank-balance", "sdk-staking", "sdk-gov", "electorate", "qualification", "knowledge", "vesting-rewards", "verifier-profile"}},
}

type expectedFinding struct {
	ID             string
	ReleaseSurface string
	Nodes          []string
	Evidence       []string
}

var expectedFindings = []expectedFinding{
	{"dual-staking-ledgers", "CUSTOM_STAKING_RUNTIME", []string{"sdk-staking", "custom-staking"}, []string{"app-wiring", "knowledge-staking-adapter"}},
	{"dual-governance-systems", "ORDINARY_GOVERNANCE_EXECUTION", []string{"sdk-gov", "custom-gov"}, []string{"app-wiring", "custom-gov-resolution"}},
	{"dual-domain-registries", "DOMAIN_REGISTRY", []string{"ontology", "knowledge"}, []string{"ontology-writer", "knowledge-writers"}},
	{"direct-fact-adoption", "DIRECT_FACT_ADOPTION", []string{"sdk-gov", "knowledge"}, []string{"knowledge-writers", "app-wiring"}},
	{"legacy-research-disbursement", "RESEARCH_DISBURSEMENT", []string{"custom-gov", "vesting-rewards"}, []string{"custom-research-spend", "research-router-restriction", "app-wiring"}},
	{"alternate-quarantine-surfaces", "QUARANTINE_PAUSE", []string{"emergency", "custom-gov", "knowledge"}, []string{"custom-emergency-electorate", "custom-gov-quarantine-wrapper", "custom-gov-emergency-hold-writer", "knowledge-pause-writer", "app-wiring"}},
	{"custom-staking-runtime-consumers", "CUSTOM_STAKING_RUNTIME", []string{"custom-staking", "custom-gov", "knowledge", "qualification", "emergency", "alignment", "claiming-pot"}, []string{"app-wiring", "knowledge-staking-adapter", "knowledge-slash-callsites", "alignment-staking-adapter", "alignment-staking-callsites", "claiming-pot-staking-adapter", "claiming-pot-staking-callsite"}},
}

var expectedSurfaces = []surfaceCheck{
	{"validator-update-writer", "PASS_STATIC_INSPECTION", []string{}},
	{"ordinary-governance-execution", "FAIL", []string{"dual-governance-systems"}},
	{"domain-registry-writer", "FAIL", []string{"dual-domain-registries"}},
	{"direct-fact-adoption-writer", "FAIL", []string{"direct-fact-adoption"}},
	{"research-disbursement-writer", "FAIL", []string{"legacy-research-disbursement"}},
	{"quarantine-pause-authority", "FAIL", []string{"alternate-quarantine-surfaces"}},
	{"custom-staking-runtime-consumer", "FAIL", []string{"dual-staking-ledgers", "custom-staking-runtime-consumers"}},
}

func decodeManifest(data []byte, issues *issueSet) (manifest, bool) {
	var m manifest
	if err := rejectDuplicateJSONKeys(data); err != nil {
		issues.add("MANIFEST_DUPLICATE_KEY", "manifest JSON contains a duplicate object key")
		return m, false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		issues.add("MANIFEST_JSON_INVALID", "manifest JSON does not match the v1 contract")
		return m, false
	}
	if err := ensureJSONEOF(decoder); err != nil {
		issues.add("MANIFEST_JSON_INVALID", "manifest JSON contains trailing data")
		return m, false
	}
	return m, true
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("extra JSON value")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := readJSONValue(decoder); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func readJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate key")
			}
			seen[key] = struct{}{}
			if err := readJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("unterminated object")
		}
	case '[':
		for decoder.More() {
			if err := readJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("unterminated array")
		}
	default:
		return fmt.Errorf("unexpected delimiter")
	}
	return nil
}

func validateManifest(m manifest, issues *issueSet) {
	if m.Schema != manifestSchema {
		issues.add("MANIFEST_SCHEMA_INVALID", "schema must be zerone.authority-geometry/v1")
	}
	if m.Revision != "1.0.0" {
		issues.add("MANIFEST_REVISION_INVALID", "revision must be 1.0.0")
	}
	if m.SnapshotDate != "2026-08-11" {
		issues.add("MANIFEST_SNAPSHOT_INVALID", "snapshotDate must remain the reviewed v1 date")
	}
	if m.Status != "SOURCE_OBSERVATORY_ONLY" {
		issues.add("MANIFEST_STATUS_INVALID", "status must remain SOURCE_OBSERVATORY_ONLY")
	}
	if strings.TrimSpace(m.Title) == "" || strings.TrimSpace(m.Summary) == "" {
		issues.add("MANIFEST_DESCRIPTION_INVALID", "title and summary must be non-empty")
	}
	validateSourceDesign(m.SourceDesign, issues)
	validateObservationScopes(m.ObservationScopes, issues)
	validateCurrentTruth(m.CurrentTruth, issues)
	validateReleaseBoundary(m.ReleaseBoundary, issues)
	validatePrinciples(m.Principles, issues)
	validateCapabilitiesAndNodes(m.Capabilities, m.Nodes, issues)
	validateEdges(m.Edges, m.Nodes, m.SourceAnchors, issues)
	validateForbiddenInfluence(m.ForbiddenInfluence, m.Nodes, m.Edges, issues)
	validateAnchorDeclarations(m.SourceAnchors, m.Edges, m.CurrentFindings, issues)
	validateFindings(m.CurrentFindings, issues)
	validateStaticGate(m.StaticAuthorityGate, issues)
	validateActivationGates(m.ActivationGates, issues)
	validateReleaseAssessment(m.ReleaseAssessment, issues)
}

func validateSourceDesign(got sourceDesign, issues *issueSet) {
	expected := sourceDesign{
		RepositoryPath: "docs/AUTHORITATIVE-STATE.md",
		SHA256:         canonicalDesignSHA256,
		DecisionDate:   "2026-08-08",
		Status:         "ACCEPTED_SOURCE_DESIGN_ONLY",
	}
	if got != expected {
		issues.add("SOURCE_DESIGN_INVALID", "sourceDesign must remain pinned to the accepted source-only design")
	}
}

func validateObservationScopes(scopes []observationScope, issues *issueSet) {
	expected := []struct{ id, status string }{
		{"LIVE_NETWORK", "NOT_ESTABLISHED_BY_THIS_ARTIFACT"},
		{"CURRENT_SOURCE", "PINNED_STATIC_INSPECTION"},
		{"ACCEPTED_TARGET", "DESIGN_ONLY"},
	}
	if len(scopes) != len(expected) {
		issues.add("OBSERVATION_SCOPES_INVALID", "exactly three ordered observation scopes are required")
		return
	}
	for i, want := range expected {
		if scopes[i].ID != want.id || scopes[i].Status != want.status || strings.TrimSpace(scopes[i].Description) == "" {
			issues.add("OBSERVATION_SCOPE_INVALID", fmt.Sprintf("observation scope %d must be %s/%s", i+1, want.id, want.status))
		}
	}
}

func validateCurrentTruth(truth currentTruth, issues *issueSet) {
	if truth.Scope != "CURRENT_SOURCE_AND_DISCLOSED_CONTEXT" ||
		truth.LiveEvidence != "NOT_ESTABLISHED_BY_THIS_ARTIFACT" ||
		truth.NetworkNamedForContext != "zerone-1" {
		issues.add("CURRENT_TRUTH_SCOPE_INVALID", "currentTruth must distinguish source, disclosed context, and unavailable live evidence")
	}
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceHasDualAuthoritySurfaces", truth.SourceHasDualAuthoritySurfaces, true)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceUsesBondedWealthForSdkGovernance", truth.SourceUsesBondedWealthForSDKGovernance, true)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "disclosedFoundingHouseholdRetainsEffectiveControl", truth.DisclosedFoundingHouseholdRetainsEffectiveControl, true)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceTargetModulesComplete", truth.SourceTargetModulesComplete, false)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceAuthorityUnified", truth.SourceAuthorityUnified, false)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceRegistersH4FreezeHandler", truth.SourceRegistersH4FreezeHandler, false)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceRegistersH4UnificationHandler", truth.SourceRegistersH4UnificationHandler, false)
	requireBool(issues, "CURRENT_TRUTH_INVALID", "sourceRegistersH5NonEconomicGovernanceHandler", truth.SourceRegistersH5NonEconomicGovernanceHandler, false)
}

func validateReleaseBoundary(boundary releaseBoundary, issues *issueSet) {
	fields := []struct {
		name  string
		value *bool
	}{
		{"addsConsensusBehavior", boundary.AddsConsensusBehavior},
		{"registersUpgradeHandler", boundary.RegistersUpgradeHandler},
		{"schedulesUpgrade", boundary.SchedulesUpgrade},
		{"changesValidatorState", boundary.ChangesValidatorState},
		{"changesGovernanceState", boundary.ChangesGovernanceState},
		{"changesDomainState", boundary.ChangesDomainState},
		{"movesFunds", boundary.MovesFunds},
		{"grantsQualification", boundary.GrantsQualification},
		{"createsRewardOrKarma", boundary.CreatesRewardOrKarma},
		{"submitsTransaction", boundary.SubmitsTransaction},
		{"assertsDecentralization", boundary.AssertsDecentralization},
		{"establishesLiveNetworkState", boundary.EstablishesLiveNetworkState},
	}
	for _, field := range fields {
		requireBool(issues, "RELEASE_BOUNDARY_INVALID", field.name, field.value, false)
	}
}

func requireBool(issues *issueSet, id, name string, got *bool, want bool) {
	if got == nil || *got != want {
		issues.add(id, fmt.Sprintf("%s must be %t", name, want))
	}
}

func validatePrinciples(principles []principle, issues *issueSet) {
	expectedIDs := []string{"distinction", "legibility", "non-economic-agency", "challenge-and-repair", "honest-transition"}
	if len(principles) != len(expectedIDs) {
		issues.add("PRINCIPLES_INVALID", "exactly five ordered relationship principles are required")
		return
	}
	for i, id := range expectedIDs {
		if principles[i].ID != id || strings.TrimSpace(principles[i].Label) == "" || strings.TrimSpace(principles[i].Rule) == "" {
			issues.add("PRINCIPLE_INVALID", fmt.Sprintf("principle %d must remain %s with a label and rule", i+1, id))
		}
	}
}

func validateCapabilitiesAndNodes(capabilities []capability, nodes []authorityNode, issues *issueSet) {
	if len(capabilities) != len(expectedCapabilities) {
		issues.add("CAPABILITY_SET_INVALID", "exactly twelve ordered authority capabilities are required")
	}
	for i, want := range expectedCapabilities {
		if i >= len(capabilities) {
			break
		}
		got := capabilities[i]
		if got.ID != want.ID || got.TargetWriter != want.Writer || !reflect.DeepEqual(got.CurrentStateSurfaces, want.CurrentSurfaces) || strings.TrimSpace(got.Label) == "" {
			issues.add("CAPABILITY_CLASSIFICATION_INVALID", fmt.Sprintf("capability %d must preserve the reviewed classification for %s", i+1, want.ID))
		}
	}
	if len(nodes) != len(expectedNodes) {
		issues.add("AUTHORITY_NODE_SET_INVALID", "exactly eighteen ordered authority nodes are required")
	}
	nodeByID := make(map[string]authorityNode)
	for i, node := range nodes {
		if _, duplicate := nodeByID[node.ID]; duplicate {
			issues.add("AUTHORITY_NODE_DUPLICATE", fmt.Sprintf("authority node %s is duplicated", node.ID))
		}
		nodeByID[node.ID] = node
		if i >= len(expectedNodes) {
			continue
		}
		want := expectedNodes[i]
		if node.ID != want.ID || node.Module != want.Module || node.Role != want.Role || node.Implementation != want.Implementation || !reflect.DeepEqual(node.TargetCapabilities, want.Capabilities) || strings.TrimSpace(node.Label) == "" {
			issues.add("AUTHORITY_NODE_CLASSIFICATION_INVALID", fmt.Sprintf("authority node %d must preserve the reviewed classification for %s", i+1, want.ID))
		}
	}
	for _, capability := range capabilities {
		writer, ok := nodeByID[capability.TargetWriter]
		if !ok {
			issues.add("CAPABILITY_WRITER_UNRESOLVED", fmt.Sprintf("capability %s has an unresolved target writer", capability.ID))
			continue
		}
		if !containsString(writer.TargetCapabilities, capability.ID) {
			issues.add("CAPABILITY_WRITER_ASYMMETRIC", fmt.Sprintf("target writer %s does not claim capability %s", writer.ID, capability.ID))
		}
	}
}

func validateEdges(edges []authorityEdge, nodes []authorityNode, anchors []sourceAnchor, issues *issueSet) {
	if len(edges) != 26 {
		issues.add("AUTHORITY_EDGE_SET_INVALID", "exactly twenty-six classified relationships are required")
	}
	nodeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeIDs[node.ID] = struct{}{}
	}
	anchorIDs := make(map[string]struct{}, len(anchors))
	for _, anchor := range anchors {
		anchorIDs[anchor.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(edges))
	currentCount := 0
	targetCount := 0
	for _, edge := range edges {
		if edge.ID == "" {
			issues.add("AUTHORITY_EDGE_INVALID", "an authority edge has no id")
		} else if _, duplicate := seen[edge.ID]; duplicate {
			issues.add("AUTHORITY_EDGE_DUPLICATE", fmt.Sprintf("authority edge %s is duplicated", edge.ID))
		}
		seen[edge.ID] = struct{}{}
		if _, ok := nodeIDs[edge.From]; !ok {
			issues.add("AUTHORITY_EDGE_REFERENCE_INVALID", fmt.Sprintf("edge %s has an unresolved from node", edge.ID))
		}
		if _, ok := nodeIDs[edge.To]; !ok {
			issues.add("AUTHORITY_EDGE_REFERENCE_INVALID", fmt.Sprintf("edge %s has an unresolved to node", edge.ID))
		}
		if edge.Scope != "CURRENT_SOURCE" && edge.Scope != "ACCEPTED_TARGET" {
			issues.add("AUTHORITY_EDGE_SCOPE_INVALID", fmt.Sprintf("edge %s has an unsupported scope", edge.ID))
		}
		if edge.Scope == "CURRENT_SOURCE" {
			currentCount++
		} else if edge.Scope == "ACCEPTED_TARGET" {
			targetCount++
		}
		if strings.TrimSpace(edge.Relationship) == "" || strings.TrimSpace(edge.Effect) == "" || len(edge.Evidence) == 0 {
			issues.add("AUTHORITY_EDGE_CLASSIFICATION_INVALID", fmt.Sprintf("edge %s lacks relationship, effect, or evidence", edge.ID))
		}
		for _, evidenceID := range edge.Evidence {
			if _, ok := anchorIDs[evidenceID]; !ok {
				issues.add("AUTHORITY_EDGE_EVIDENCE_INVALID", fmt.Sprintf("edge %s cites an unresolved source anchor", edge.ID))
			}
		}
		if edge.Scope == "CURRENT_SOURCE" && !hasCodeAnchor(edge.Evidence) {
			issues.add("AUTHORITY_EDGE_EVIDENCE_INVALID", fmt.Sprintf("current-source edge %s must cite current code", edge.ID))
		}
		if edge.Scope == "ACCEPTED_TARGET" && !containsString(edge.Evidence, "authoritative-state-design") {
			issues.add("AUTHORITY_EDGE_EVIDENCE_INVALID", fmt.Sprintf("accepted-target edge %s must cite the accepted design", edge.ID))
		}
		if strings.TrimSpace(edge.Protections.ConsentBoundary) == "" ||
			strings.TrimSpace(edge.Protections.ChallengeRoute) == "" ||
			strings.TrimSpace(edge.Protections.RepairRoute) == "" ||
			strings.TrimSpace(edge.Protections.ExitRoute) == "" {
			issues.add("AUTHORITY_EDGE_PROTECTIONS_INVALID", fmt.Sprintf("edge %s must name consent, challenge, repair, and exit", edge.ID))
		}
	}
	if currentCount != len(expectedCurrentEdges) {
		issues.add("CURRENT_AUTHORITY_EDGE_SET_INVALID", "exactly twelve current-source relationships are required")
	}
	if targetCount != len(expectedTargetEdges) {
		issues.add("TARGET_AUTHORITY_EDGE_SET_INVALID", "exactly fourteen accepted-target relationships are required")
	}
	for i, want := range expectedCurrentEdges {
		if i >= len(edges) {
			break
		}
		got := edges[i]
		if got.ID != want.ID || got.From != want.From || got.To != want.To ||
			got.Relationship != want.Relationship || got.Scope != "CURRENT_SOURCE" ||
			got.Effect != want.Effect || !reflect.DeepEqual(got.Evidence, want.Evidence) {
			issues.add("CURRENT_AUTHORITY_EDGE_CLASSIFICATION_INVALID", fmt.Sprintf("current-source relationship %d must preserve %s", i+1, want.ID))
		}
	}
	for i, want := range expectedTargetEdges {
		edgeIndex := len(expectedCurrentEdges) + i
		if edgeIndex >= len(edges) {
			break
		}
		got := edges[edgeIndex]
		if got.ID != want.ID || got.From != want.From || got.To != want.To ||
			got.Relationship != want.Relationship || got.Scope != "ACCEPTED_TARGET" ||
			got.Effect != want.Effect || !reflect.DeepEqual(got.Evidence, want.Evidence) {
			issues.add("TARGET_AUTHORITY_EDGE_CLASSIFICATION_INVALID", fmt.Sprintf("accepted-target relationship %d must preserve %s", i+1, want.ID))
		}
	}
}

func hasCodeAnchor(evidence []string) bool {
	for _, id := range evidence {
		if id != "authoritative-state-design" {
			return true
		}
	}
	return false
}

func validateForbiddenInfluence(rules []forbiddenInfluence, nodes []authorityNode, edges []authorityEdge, issues *issueSet) {
	if len(rules) != len(expectedForbiddenRules) {
		issues.add("FORBIDDEN_INFLUENCE_SET_INVALID", "exactly four ordered zero-path rules are required")
	}
	nodeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeIDs[node.ID] = struct{}{}
	}
	adjacency := acceptedTargetInfluenceAdjacency(edges)
	for i, rule := range rules {
		if i < len(expectedForbiddenRules) {
			want := expectedForbiddenRules[i]
			if rule.ID != want.ID || !reflect.DeepEqual(rule.Sources, want.Sources) || !reflect.DeepEqual(rule.Targets, want.Targets) {
				issues.add("FORBIDDEN_INFLUENCE_CLASSIFICATION_INVALID", fmt.Sprintf("zero-path rule %d must preserve the exact endpoints for %s", i+1, want.ID))
			}
		}
		if strings.TrimSpace(rule.Label) == "" || len(rule.Sources) == 0 || len(rule.Targets) == 0 || rule.AcceptedTargetPathCount != 0 {
			issues.add("FORBIDDEN_INFLUENCE_INVALID", fmt.Sprintf("zero-path rule %s must declare non-empty endpoints and zero target paths", rule.ID))
		}
		for _, id := range append(append([]string(nil), rule.Sources...), rule.Targets...) {
			if _, ok := nodeIDs[id]; !ok {
				issues.add("FORBIDDEN_INFLUENCE_REFERENCE_INVALID", fmt.Sprintf("zero-path rule %s cites an unresolved node", rule.ID))
			}
		}
		computedPathCount := 0
		for _, source := range rule.Sources {
			for _, target := range rule.Targets {
				if hasDirectedInfluencePath(adjacency, source, target) {
					computedPathCount++
				}
			}
		}
		if computedPathCount != rule.AcceptedTargetPathCount || computedPathCount != 0 {
			issues.add(
				"FORBIDDEN_INFLUENCE_PATH_COUNT_MISMATCH",
				fmt.Sprintf("zero-path rule %s declares %d but the accepted target graph computes %d", rule.ID, rule.AcceptedTargetPathCount, computedPathCount),
			)
		}
	}
}

func acceptedTargetInfluenceAdjacency(edges []authorityEdge) map[string][]string {
	adjacency := make(map[string][]string)
	for _, edge := range edges {
		if edge.Scope != "ACCEPTED_TARGET" || !isEffectfulInfluenceEdge(edge) {
			continue
		}
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}
	for from := range adjacency {
		sort.Strings(adjacency[from])
	}
	return adjacency
}

func isEffectfulInfluenceEdge(edge authorityEdge) bool {
	switch edge.Effect {
	case "REFERENCE_RELATION", "EVIDENCE_RELATION", "RETIREMENT_RELATION":
		return false
	default:
		return true
	}
}

func hasDirectedInfluencePath(adjacency map[string][]string, source, target string) bool {
	pending := []string{source}
	visited := make(map[string]bool)
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		if current == target && current != source {
			return true
		}
		pending = append(pending, adjacency[current]...)
	}
	return false
}

func validateAnchorDeclarations(anchors []sourceAnchor, edges []authorityEdge, findings []currentFinding, issues *issueSet) {
	if len(anchors) != len(expectedAnchors) {
		issues.add("SOURCE_ANCHOR_SET_INVALID", "exactly seventeen ordered source anchors are required")
	}
	used := make(map[string]bool)
	for _, edge := range edges {
		for _, id := range edge.Evidence {
			used[id] = true
		}
	}
	for _, finding := range findings {
		for _, id := range finding.Evidence {
			used[id] = true
		}
	}
	seen := make(map[string]struct{}, len(anchors))
	for i, anchor := range anchors {
		if _, duplicate := seen[anchor.ID]; duplicate {
			issues.add("SOURCE_ANCHOR_DUPLICATE", fmt.Sprintf("source anchor %s is duplicated", anchor.ID))
		}
		seen[anchor.ID] = struct{}{}
		if !used[anchor.ID] {
			issues.add("SOURCE_ANCHOR_UNUSED", fmt.Sprintf("source anchor %s is not referenced by a graph fact", anchor.ID))
		}
		if i >= len(expectedAnchors) {
			continue
		}
		want := expectedAnchors[i]
		if anchor.ID != want.ID || anchor.Path != want.Path || !reflect.DeepEqual(anchor.RequiredSnippets, want.Required) || !reflect.DeepEqual(anchor.ForbiddenSnippets, want.Forbidden) {
			issues.add("SOURCE_ANCHOR_DECLARATION_INVALID", fmt.Sprintf("source anchor %d must preserve the reviewed declaration for %s", i+1, want.ID))
		}
		if !isLowerSHA256(anchor.SHA256) {
			issues.add("SOURCE_ANCHOR_DIGEST_INVALID", fmt.Sprintf("source anchor %s must declare a lowercase SHA-256", anchor.ID))
		}
	}
}

func validateFindings(findings []currentFinding, issues *issueSet) {
	if len(findings) != len(expectedFindings) {
		issues.add("CURRENT_FINDING_SET_INVALID", "all seven ordered current-source conflicts must remain visible")
	}
	for i, want := range expectedFindings {
		if i >= len(findings) {
			break
		}
		got := findings[i]
		if got.ID != want.ID || got.Status != "OPEN" || got.ReleaseSurface != want.ReleaseSurface || !reflect.DeepEqual(got.Nodes, want.Nodes) || !reflect.DeepEqual(got.Evidence, want.Evidence) || strings.TrimSpace(got.Label) == "" {
			issues.add("CURRENT_FINDING_CLASSIFICATION_INVALID", fmt.Sprintf("finding %d must preserve the open reviewed conflict %s", i+1, want.ID))
		}
	}
}

func validateStaticGate(gate staticAuthorityGate, issues *issueSet) {
	if gate.AuthoritativeStateReleaseGate != "H4-02" || gate.Status != "FAIL_CURRENT_SOURCE" || gate.Mode != "REPORT_SUCCEEDS_TARGET_GATE_REFUSES" {
		issues.add("STATIC_AUTHORITY_GATE_INVALID", "the H4-02 source gate must remain FAIL_CURRENT_SOURCE in split report/target mode")
	}
	if !reflect.DeepEqual(gate.SurfaceChecks, expectedSurfaces) {
		issues.add("STATIC_AUTHORITY_SURFACES_INVALID", "the seven ordered static surfaces must preserve one pass and six failures")
	}
}

func validateActivationGates(gates activationGates, issues *issueSet) {
	validateGateSeries("H4", gates.H4, 24, issues)
	validateGateSeries("H5", gates.H5, 14, issues)
}

func validateGateSeries(prefix string, gates []activationGate, count int, issues *issueSet) {
	if len(gates) != count {
		issues.add("ACTIVATION_GATE_SET_INVALID", fmt.Sprintf("%s must contain exactly %d ordered gates", prefix, count))
	}
	seen := make(map[string]struct{}, len(gates))
	for i, gate := range gates {
		expectedID := fmt.Sprintf("%s-%02d", prefix, i+1)
		if gate.ID != expectedID {
			issues.add("ACTIVATION_GATE_ORDER_INVALID", fmt.Sprintf("%s gate %d must be %s", prefix, i+1, expectedID))
		}
		if _, duplicate := seen[gate.ID]; duplicate {
			issues.add("ACTIVATION_GATE_DUPLICATE", fmt.Sprintf("activation gate %s is duplicated", gate.ID))
		}
		seen[gate.ID] = struct{}{}
		if gate.Status != "NOT_EVIDENCED" {
			issues.add("ACTIVATION_GATE_STATUS_INVALID", fmt.Sprintf("activation gate %s must remain NOT_EVIDENCED in v1", gate.ID))
		}
		if strings.TrimSpace(gate.Summary) == "" {
			issues.add("ACTIVATION_GATE_SUMMARY_INVALID", fmt.Sprintf("activation gate %s requires a summary", gate.ID))
		}
	}
}

func validateReleaseAssessment(assessment releaseAssessment, issues *issueSet) {
	if assessment.Overall != "NO_GO" ||
		assessment.StaticAuthoritySurfacesPassing != 1 ||
		assessment.StaticAuthoritySurfacesTotal != 7 ||
		assessment.H4GatesEvidenced != 0 ||
		assessment.H4GatesTotal != 24 ||
		assessment.H5GatesEvidenced != 0 ||
		assessment.H5GatesTotal != 14 {
		issues.add("RELEASE_ASSESSMENT_INVALID", "release assessment must remain NO_GO with static 1/7, H4 0/24, and H5 0/14")
	}
	requireBool(issues, "RELEASE_ASSESSMENT_INVALID", "targetGateMustExitNonZero", assessment.TargetGateMustExitNonZero, true)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func isLowerSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
