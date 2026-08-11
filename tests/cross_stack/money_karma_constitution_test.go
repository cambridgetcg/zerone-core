package cross_stack_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Pointer fields make omission distinguishable from an explicit false, zero,
// or empty string. The contract is closed: DisallowUnknownFields rejects
// additions while these presence checks reject deletions.
type moneyKarmaConstitution struct {
	Schema  *string `json:"schema"`
	Version *uint64 `json:"version"`
	Status  struct {
		Artifact                 *string `json:"artifact"`
		ConsensusRuntimeDeployed *bool   `json:"consensus_runtime_deployed"`
		NetworkActivated         *bool   `json:"network_activated"`
		ClaimsPresentNoControl   *bool   `json:"claims_present_no_control"`
	} `json:"status"`
	Separations struct {
		MoneyDirectlyGrantsVoice                 *bool `json:"money_directly_grants_voice"`
		EconomicToGovernanceDecouplingEnforced   *bool `json:"economic_to_governance_decoupling_enforced"`
		FundedRewardActivationRequiresDecoupling *bool `json:"funded_reward_activation_requires_decoupling"`
		KarmaIsMoney                             *bool `json:"karma_is_money"`
		RecognitionGrantsAuthority               *bool `json:"recognition_grants_authority"`
	} `json:"separations"`
	FounderRenunciation struct {
		ShareBps                                       *uint64 `json:"share_bps"`
		Address                                        *string `json:"address"`
		Scope                                          *string `json:"scope"`
		V2FieldsRestorable                             *bool   `json:"v2_fields_restorable"`
		V2BeneficiarySubstitutionAllowed               *bool   `json:"v2_beneficiary_substitution_allowed"`
		OrdinaryParameterGovernanceMayChange           *bool   `json:"ordinary_parameter_governance_may_change"`
		FutureCoordinatedCodeChangeTechnicallyPossible *bool   `json:"future_coordinated_code_change_technically_possible"`
		CompatibilityFieldsOnly                        *bool   `json:"compatibility_fields_only"`
		MigrationRelease                               *string `json:"migration_release"`
		ExactPrestateRequired                          *bool   `json:"exact_prestate_required"`
		OrdinaryNamedUpgradeHitchhikingAllowed         *bool   `json:"ordinary_named_upgrade_hitchhiking_allowed"`
	} `json:"founder_renunciation"`
	Karma struct {
		Stage                            *string `json:"stage"`
		Representation                   *string `json:"representation"`
		EventType                        *string `json:"event_type"`
		EventRegister                    *string `json:"event_register"`
		Meaning                          *string `json:"meaning"`
		ZeroneMintsOrCreates             *bool   `json:"zerone_mints_or_creates"`
		Assignable                       *bool   `json:"assignable"`
		OperatorAssignable               *bool   `json:"operator_assignable"`
		FounderAssignable                *bool   `json:"founder_assignable"`
		ObservationsFallible             *bool   `json:"observations_fallible"`
		ObservationsChallengeable        *bool   `json:"observations_challengeable"`
		RecordingClaimsRelationOwnership *bool   `json:"recording_claims_relation_ownership"`
		Transferable                     *bool   `json:"transferable"`
		Delegable                        *bool   `json:"delegable"`
		Saleable                         *bool   `json:"saleable"`
		Collateralizable                 *bool   `json:"collateralizable"`
		Inheritable                      *bool   `json:"inheritable"`
		Denom                            *bool   `json:"denom"`
		Balance                          *bool   `json:"balance"`
		Bank                             *bool   `json:"bank"`
		IBC                              *bool   `json:"ibc"`
		AMM                              *bool   `json:"amm"`
		RewardMultiplier                 *bool   `json:"reward_multiplier"`
		Payout                           *bool   `json:"payout"`
		GovernanceConsumer               *bool   `json:"governance_consumer"`
		NumericMagnitude                 *bool   `json:"numeric_magnitude"`
		DedicatedStateStore              *bool   `json:"dedicated_state_store"`
		RawEventsQualify                 *bool   `json:"raw_events_qualify"`
		RawEventCountsQualify            *bool   `json:"raw_event_counts_qualify"`
		FutureRandomizedEligibility      struct {
			RuntimeEnforced                     *bool   `json:"runtime_enforced"`
			SameControllerEdgesExcluded         *bool   `json:"same_controller_edges_excluded"`
			SelfEdgesExcluded                   *bool   `json:"self_edges_excluded"`
			ReciprocalEdgesExcluded             *bool   `json:"reciprocal_edges_excluded"`
			CorrelatedFunderEdgesExcluded       *bool   `json:"correlated_funder_edges_excluded"`
			ControllerMergesOnlyReduceUnits     *bool   `json:"controller_merges_only_reduce_units"`
			MaximumLotteryUnitsPerController    *uint64 `json:"maximum_lottery_units_per_controller"`
			CandidateSetFrozenBeforeRandomness  *bool   `json:"candidate_set_frozen_before_randomness"`
			UnbiasedRandomnessRequired          *bool   `json:"unbiased_randomness_required"`
			OperatorOverrideAllowed             *bool   `json:"operator_override_allowed"`
			CountProportionalProbabilityAllowed *bool   `json:"count_proportional_probability_allowed"`
		} `json:"future_randomized_eligibility"`
	} `json:"karma"`
	FutureIndependenceActivation struct {
		Status                                 *string `json:"status"`
		RuntimeEnforced                        *bool   `json:"runtime_enforced"`
		FounderAssociatedRecusalRequired       *bool   `json:"founder_associated_recusal_required"`
		ControlDerivedGrantsProhibited         *bool   `json:"control_derived_grants_prohibited"`
		EconomicToGovernanceDecouplingRequired *bool   `json:"economic_to_governance_decoupling_required"`
	} `json:"future_independence_activation"`
	ResearchGovernance struct {
		Scope                                     *string `json:"scope"`
		SpendingPolicy                            *string `json:"spending_policy"`
		RuntimeIndependencePredicateEnforced      *bool   `json:"runtime_independence_predicate_enforced"`
		CurrentAuthorityManagedSpendingPathExists *bool   `json:"current_authority_managed_spending_path_exists"`
		CurrentIdentityBound                      *bool   `json:"current_identity_bound"`
		ArtifactAuthorizesSpending                *bool   `json:"artifact_authorizes_spending"`
		ArtifactDisablesSpending                  *bool   `json:"artifact_disables_spending"`
		ArtifactRefusesDisbursement               *bool   `json:"artifact_refuses_disbursement"`
		ArtifactChangesGovernance                 *bool   `json:"artifact_changes_governance"`
		CurrentKarmaConsumer                      *bool   `json:"current_karma_consumer"`
		FutureBoundary                            struct {
			ControllerCollapsed          *bool `json:"controller_collapsed"`
			BoundedEligibility           *bool `json:"bounded_eligibility"`
			BoundedSortition             *bool `json:"bounded_sortition"`
			IndependentCheckChamber      *bool `json:"independent_check_chamber"`
			RotationAndConcentrationCaps *bool `json:"rotation_and_concentration_caps"`
			ConflictDisclosure           *bool `json:"conflict_disclosure"`
			ChallengeAndTimelock         *bool `json:"challenge_and_timelock"`
			ScalarWeighting              *bool `json:"scalar_weighting"`
			UnilateralExecution          *bool `json:"unilateral_execution"`
		} `json:"future_boundary"`
	} `json:"research_governance"`
	OperationalCaveat struct {
		AsOf                                         *string `json:"as_of"`
		BondedFungibleWealthAffectsCurrentVoteWeight *bool   `json:"bonded_fungible_wealth_affects_current_vote_weight"`
		SoleFoundingHouseholdControlsValidator       *bool   `json:"sole_founding_household_controls_validator"`
		SoleFoundingHouseholdControlsOperator        *bool   `json:"sole_founding_household_controls_operator"`
		SoleFoundingHouseholdControlsVote            *bool   `json:"sole_founding_household_controls_effective_vote"`
		ValidatorEconomicsRemain                     *bool   `json:"validator_economics_remain"`
		BootstrapRegistrarDiscretionRemains          *bool   `json:"bootstrap_registrar_discretion_remains"`
		DefaultDisabledGovernanceIssuance            *bool   `json:"default_disabled_governance_issuance_remains"`
		IdentityBoundResearchSpendingRemains         *bool   `json:"identity_bound_research_spending_remains"`
		PresentNoControlAchieved                     *bool   `json:"present_no_control_achieved"`
		IndependentGovernanceProven                  *bool   `json:"independent_governance_proven"`
	} `json:"operational_caveat"`
	ActivationExclusions struct {
		H1ActivatesKarma                   *bool `json:"h1_activates_karma"`
		H1ActivatesLifeSciencesOverlay     *bool `json:"h1_activates_life_sciences_overlay"`
		H1ActivatesConstructiveRewards     *bool `json:"h1_activates_constructive_rewards"`
		H2ActivatesKarma                   *bool `json:"h2_activates_karma"`
		H2ActivatesLifeSciencesOverlay     *bool `json:"h2_activates_life_sciences_overlay"`
		H2ActivatesConstructiveRewards     *bool `json:"h2_activates_constructive_rewards"`
		H3ActivatesKarma                   *bool `json:"h3_activates_karma"`
		H3ActivatesLifeSciencesOverlay     *bool `json:"h3_activates_life_sciences_overlay"`
		H3ActivatesConstructiveRewards     *bool `json:"h3_activates_constructive_rewards"`
		SkillTreePositionActivatesIssuance *bool `json:"skill_tree_position_activates_issuance"`
		BreakthroughLabelActivatesIssuance *bool `json:"breakthrough_label_activates_issuance"`
	} `json:"activation_exclusions"`
	Effects struct {
		Economic   *bool `json:"economic"`
		Governance *bool `json:"governance"`
		Consensus  *bool `json:"consensus"`
		Authority  *bool `json:"authority"`
	} `json:"effects"`
}

func TestMoneyKarma_ConstitutionIsStrictAndEffectless(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	contractPath := filepath.Join(repoRoot, "docs", "constitution", "money-karma-v1.json")
	raw, err := os.ReadFile(contractPath)
	require.NoError(t, err)
	require.NoError(t, rejectDuplicateJSONKeys(raw),
		"machine constitution must reject duplicate object keys before typed decoding")

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var contract moneyKarmaConstitution
	require.NoError(t, decoder.Decode(&contract), "machine constitution must strict-parse")
	var trailing any
	require.ErrorIs(t, decoder.Decode(&trailing), io.EOF, "machine constitution must contain one JSON value")

	requireString(t, "schema", contract.Schema, "zerone.money-karma.constitution/v1")
	requireUint64(t, "version", contract.Version, 1)
	requireString(t, "status.artifact", contract.Status.Artifact, "SOURCE_CONSTITUTION")
	requireFalse(t, map[string]*bool{
		"status.consensus_runtime_deployed":        contract.Status.ConsensusRuntimeDeployed,
		"status.network_activated":                 contract.Status.NetworkActivated,
		"status.claims_present_no_control":         contract.Status.ClaimsPresentNoControl,
		"separations.money_directly_grants_voice":  contract.Separations.MoneyDirectlyGrantsVoice,
		"separations.decoupling_enforced":          contract.Separations.EconomicToGovernanceDecouplingEnforced,
		"separations.karma_is_money":               contract.Separations.KarmaIsMoney,
		"separations.recognition_grants_authority": contract.Separations.RecognitionGrantsAuthority,
		"founder.v2_fields_restorable":             contract.FounderRenunciation.V2FieldsRestorable,
		"founder.v2_beneficiary_substitution":      contract.FounderRenunciation.V2BeneficiarySubstitutionAllowed,
		"founder.ordinary_governance_may_change":   contract.FounderRenunciation.OrdinaryParameterGovernanceMayChange,
		"founder.ordinary_upgrade_hitchhiking":     contract.FounderRenunciation.OrdinaryNamedUpgradeHitchhikingAllowed,
		"karma.transferable":                       contract.Karma.Transferable,
		"karma.zerone_mints_or_creates":            contract.Karma.ZeroneMintsOrCreates,
		"karma.assignable":                         contract.Karma.Assignable,
		"karma.operator_assignable":                contract.Karma.OperatorAssignable,
		"karma.founder_assignable":                 contract.Karma.FounderAssignable,
		"karma.recording_claims_ownership":         contract.Karma.RecordingClaimsRelationOwnership,
		"karma.delegable":                          contract.Karma.Delegable,
		"karma.saleable":                           contract.Karma.Saleable,
		"karma.collateralizable":                   contract.Karma.Collateralizable,
		"karma.inheritable":                        contract.Karma.Inheritable,
		"karma.denom":                              contract.Karma.Denom,
		"karma.balance":                            contract.Karma.Balance,
		"karma.bank":                               contract.Karma.Bank,
		"karma.ibc":                                contract.Karma.IBC,
		"karma.amm":                                contract.Karma.AMM,
		"karma.reward_multiplier":                  contract.Karma.RewardMultiplier,
		"karma.payout":                             contract.Karma.Payout,
		"karma.governance_consumer":                contract.Karma.GovernanceConsumer,
		"karma.numeric_magnitude":                  contract.Karma.NumericMagnitude,
		"karma.dedicated_state_store":              contract.Karma.DedicatedStateStore,
		"karma.raw_events_qualify":                 contract.Karma.RawEventsQualify,
		"karma.raw_event_counts_qualify":           contract.Karma.RawEventCountsQualify,
		"karma.future.runtime_enforced":            contract.Karma.FutureRandomizedEligibility.RuntimeEnforced,
		"karma.future.operator_override":           contract.Karma.FutureRandomizedEligibility.OperatorOverrideAllowed,
		"karma.future.count_probability":           contract.Karma.FutureRandomizedEligibility.CountProportionalProbabilityAllowed,
		"future_independence.runtime_enforced":     contract.FutureIndependenceActivation.RuntimeEnforced,
		"research.artifact_authorizes_spending":    contract.ResearchGovernance.ArtifactAuthorizesSpending,
		"research.artifact_disables_spending":      contract.ResearchGovernance.ArtifactDisablesSpending,
		"research.artifact_refuses_disbursement":   contract.ResearchGovernance.ArtifactRefusesDisbursement,
		"research.artifact_changes_governance":     contract.ResearchGovernance.ArtifactChangesGovernance,
		"research.runtime_independence_enforced":   contract.ResearchGovernance.RuntimeIndependencePredicateEnforced,
		"research.current_karma_consumer":          contract.ResearchGovernance.CurrentKarmaConsumer,
		"research.future.scalar_weighting":         contract.ResearchGovernance.FutureBoundary.ScalarWeighting,
		"research.future.unilateral_execution":     contract.ResearchGovernance.FutureBoundary.UnilateralExecution,
		"operations.present_no_control_achieved":   contract.OperationalCaveat.PresentNoControlAchieved,
		"operations.independent_governance_proven": contract.OperationalCaveat.IndependentGovernanceProven,
		"activation.h1_activates_karma":            contract.ActivationExclusions.H1ActivatesKarma,
		"activation.h1_activates_life_overlay":     contract.ActivationExclusions.H1ActivatesLifeSciencesOverlay,
		"activation.h1_activates_rewards":          contract.ActivationExclusions.H1ActivatesConstructiveRewards,
		"activation.h2_activates_karma":            contract.ActivationExclusions.H2ActivatesKarma,
		"activation.h2_activates_life_overlay":     contract.ActivationExclusions.H2ActivatesLifeSciencesOverlay,
		"activation.h2_activates_rewards":          contract.ActivationExclusions.H2ActivatesConstructiveRewards,
		"activation.h3_activates_karma":            contract.ActivationExclusions.H3ActivatesKarma,
		"activation.h3_activates_life_overlay":     contract.ActivationExclusions.H3ActivatesLifeSciencesOverlay,
		"activation.h3_activates_rewards":          contract.ActivationExclusions.H3ActivatesConstructiveRewards,
		"activation.skill_position_issues":         contract.ActivationExclusions.SkillTreePositionActivatesIssuance,
		"activation.breakthrough_label_issues":     contract.ActivationExclusions.BreakthroughLabelActivatesIssuance,
		"effects.economic":                         contract.Effects.Economic,
		"effects.governance":                       contract.Effects.Governance,
		"effects.consensus":                        contract.Effects.Consensus,
		"effects.authority":                        contract.Effects.Authority,
	})
	requireTrue(t, map[string]*bool{
		"separations.funded_rewards_require_decoupling":     contract.Separations.FundedRewardActivationRequiresDecoupling,
		"founder.compatibility_fields_only":                 contract.FounderRenunciation.CompatibilityFieldsOnly,
		"founder.future_code_change_technically_possible":   contract.FounderRenunciation.FutureCoordinatedCodeChangeTechnicallyPossible,
		"founder.exact_prestate_required":                   contract.FounderRenunciation.ExactPrestateRequired,
		"research.current_authority_spending_path_exists":   contract.ResearchGovernance.CurrentAuthorityManagedSpendingPathExists,
		"research.current_identity_bound":                   contract.ResearchGovernance.CurrentIdentityBound,
		"research.future.controller_collapsed":              contract.ResearchGovernance.FutureBoundary.ControllerCollapsed,
		"research.future.bounded_eligibility":               contract.ResearchGovernance.FutureBoundary.BoundedEligibility,
		"research.future.bounded_sortition":                 contract.ResearchGovernance.FutureBoundary.BoundedSortition,
		"research.future.independent_check_chamber":         contract.ResearchGovernance.FutureBoundary.IndependentCheckChamber,
		"research.future.rotation_and_concentration_caps":   contract.ResearchGovernance.FutureBoundary.RotationAndConcentrationCaps,
		"research.future.conflict_disclosure":               contract.ResearchGovernance.FutureBoundary.ConflictDisclosure,
		"research.future.challenge_and_timelock":            contract.ResearchGovernance.FutureBoundary.ChallengeAndTimelock,
		"operations.sole_household_controls_validator":      contract.OperationalCaveat.SoleFoundingHouseholdControlsValidator,
		"operations.bonded_wealth_affects_vote_weight":      contract.OperationalCaveat.BondedFungibleWealthAffectsCurrentVoteWeight,
		"operations.sole_household_controls_operator":       contract.OperationalCaveat.SoleFoundingHouseholdControlsOperator,
		"operations.sole_household_controls_effective_vote": contract.OperationalCaveat.SoleFoundingHouseholdControlsVote,
		"operations.validator_economics_remain":             contract.OperationalCaveat.ValidatorEconomicsRemain,
		"operations.bootstrap_registrar_discretion_remains": contract.OperationalCaveat.BootstrapRegistrarDiscretionRemains,
		"operations.default_disabled_governance_issuance":   contract.OperationalCaveat.DefaultDisabledGovernanceIssuance,
		"operations.identity_bound_research_spending":       contract.OperationalCaveat.IdentityBoundResearchSpendingRemains,
		"karma.observations_fallible":                       contract.Karma.ObservationsFallible,
		"karma.observations_challengeable":                  contract.Karma.ObservationsChallengeable,
		"karma.future.same_controller_excluded":             contract.Karma.FutureRandomizedEligibility.SameControllerEdgesExcluded,
		"karma.future.self_excluded":                        contract.Karma.FutureRandomizedEligibility.SelfEdgesExcluded,
		"karma.future.reciprocal_excluded":                  contract.Karma.FutureRandomizedEligibility.ReciprocalEdgesExcluded,
		"karma.future.correlated_funder_excluded":           contract.Karma.FutureRandomizedEligibility.CorrelatedFunderEdgesExcluded,
		"karma.future.merges_reduce_units":                  contract.Karma.FutureRandomizedEligibility.ControllerMergesOnlyReduceUnits,
		"karma.future.candidates_frozen":                    contract.Karma.FutureRandomizedEligibility.CandidateSetFrozenBeforeRandomness,
		"karma.future.unbiased_randomness":                  contract.Karma.FutureRandomizedEligibility.UnbiasedRandomnessRequired,
		"future_independence.founder_recusal":               contract.FutureIndependenceActivation.FounderAssociatedRecusalRequired,
		"future_independence.control_grants_prohibited":     contract.FutureIndependenceActivation.ControlDerivedGrantsProhibited,
		"future_independence.economic_decoupling_required":  contract.FutureIndependenceActivation.EconomicToGovernanceDecouplingRequired,
	})

	requireUint64(t, "founder.share_bps", contract.FounderRenunciation.ShareBps, 0)
	requireString(t, "founder.address", contract.FounderRenunciation.Address, "")
	requireString(t, "founder.scope", contract.FounderRenunciation.Scope, "VESTING_REWARDS_V2_FOUNDER_COMPATIBILITY_FIELDS")
	requireString(t, "founder.migration_release", contract.FounderRenunciation.MigrationRelease, "founder-renunciation-v1")
	requireString(t, "karma.stage", contract.Karma.Stage, "K_ALPHA")
	requireString(t, "karma.representation", contract.Karma.Representation, "EVENT_ONLY")
	requireString(t, "karma.event_type", contract.Karma.EventType, "zerone.karma.edge")
	requireString(t, "karma.event_register", contract.Karma.EventRegister, "priced-coherence")
	requireString(t, "karma.meaning", contract.Karma.Meaning, "artifact-relation")
	requireUint64(t, "karma.future.maximum_units_per_controller",
		contract.Karma.FutureRandomizedEligibility.MaximumLotteryUnitsPerController, 1)
	requireString(t, "future_independence.status",
		contract.FutureIndependenceActivation.Status, "REQUIRED_NOT_ENFORCED")
	requireString(t, "research.scope", contract.ResearchGovernance.Scope, "OUT_OF_SCOPE_UNCHANGED")
	requireString(t, "research.spending_policy", contract.ResearchGovernance.SpendingPolicy, "NORMATIVE_FAIL_CLOSED_UNTIL_GENUINELY_INDEPENDENT")
	requireString(t, "operational_caveat.as_of", contract.OperationalCaveat.AsOf, "2026-08-01")
}

func TestMoneyKarma_ConstitutionRejectsDuplicateJSONKeys(t *testing.T) {
	tests := map[string]string{
		"top-level last-value-wins":        `{"schema":"first","schema":"second"}`,
		"escaped equivalent keys":          `{"schema":"first","\u0073chema":"second"}`,
		"nested same-value duplicate":      `{"karma":{"transferable":false,"transferable":false}}`,
		"duplicate hidden in array object": `{"items":[{"kind":"cited","kind":"verify"}]}`,
	}
	for name, fixture := range tests {
		t.Run(name, func(t *testing.T) {
			err := rejectDuplicateJSONKeys([]byte(fixture))
			require.ErrorContains(t, err, "duplicate JSON key")
		})
	}
}

// This is source-level evidence for vesting_rewards v2 compatibility-field
// validation and keeper storage only. It does not prove that either migration
// has executed, that a reviewed binary is running, or that the live chain has
// activated v2.
func TestMoneyKarma_VestingRewardsV2CompatibilityFieldsRejectStorageRestoration(t *testing.T) {
	current := vestingtypes.DefaultParams()
	require.Zero(t, current.FounderShareBps)
	require.Empty(t, current.FounderAddress)
	require.NoError(t, vestingtypes.ValidateParams(current))

	withShare := proto.Clone(current).(*vestingtypes.Params)
	withShare.FounderShareBps = 1
	require.Error(t, vestingtypes.ValidateParams(withShare), "any non-zero founder share must be invalid")
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, withShare), vestingtypes.ErrFounderShareRetired)

	withAddress := proto.Clone(current).(*vestingtypes.Params)
	withAddress.FounderAddress = "zerone1beneficiarycannotrestorefoundertap"
	require.Error(t, vestingtypes.ValidateParams(withAddress), "any founder address must be invalid")
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, withAddress), vestingtypes.ErrFounderShareRetired)

	withBoth := proto.Clone(current).(*vestingtypes.Params)
	withBoth.FounderShareBps = 70_000
	withBoth.FounderAddress = "zerone1renamedbeneficiaryisstillabeneficiary"
	require.Error(t, vestingtypes.ValidateParams(withBoth))
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, withBoth), vestingtypes.ErrFounderShareRetired)

	proposedZero := proto.Clone(current).(*vestingtypes.Params)
	require.NoError(t, vestingtypes.ValidateFounderShareChange(current, proposedZero), "the only admissible proposal preserves zero and empty")

	h := NewTestHarness(t)
	storedBefore := h.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, storedBefore.FounderShareBps)
	require.Empty(t, storedBefore.FounderAddress)

	for name, proposed := range map[string]*vestingtypes.Params{
		"share":   withShare,
		"address": withAddress,
		"both":    withBoth,
	} {
		t.Run("keeper refuses "+name, func(t *testing.T) {
			before := proto.Clone(h.VestingRewardsKeeper.GetParams(h.Ctx)).(*vestingtypes.Params)
			require.Error(t, h.VestingRewardsKeeper.SetParams(h.Ctx, proposed))
			require.True(t, proto.Equal(before, h.VestingRewardsKeeper.GetParams(h.Ctx)),
				"rejected compatibility-field restoration mutated keeper state")
		})
	}

	require.NoError(t, h.VestingRewardsKeeper.SetParams(h.Ctx, proposedZero))
	storedAfter := h.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, storedAfter.FounderShareBps)
	require.Empty(t, storedAfter.FounderAddress)
}

func TestMoneyKarma_KarmaProducerSurfaceIsExact(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	sources, err := parseProductionGoSources(repoRoot)
	require.NoError(t, err)
	requireExactKarmaProducerSurface(t, sources)
}

// This repository-wide production-Go AST tripwire is deliberately described as
// a static review boundary, not as formal whole-program proof. It scans every
// non-test Go package and excludes only the exact constructor, helper-call, and
// support-constant nodes admitted by requireExactKarmaProducerSurface.
func TestMoneyKarma_NoProductionGoKarmaConsumerOutsideExactEmitterSurface(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	sources, err := parseProductionGoSources(repoRoot)
	require.NoError(t, err)
	audit := requireExactKarmaProducerSurface(t, sources)
	supportRanges, err := exactKarmaSupportConstantRanges(sources)
	require.NoError(t, err)
	audit.approvedRanges = append(audit.approvedRanges, supportRanges...)
	policyRanges, err := exactAuthorityGraphKarmaPolicyRanges(sources)
	require.NoError(t, err)
	audit.approvedRanges = append(audit.approvedRanges, policyRanges...)

	findings := unapprovedKarmaReferences(sources, audit.approvedRanges)
	require.Empty(t, findings,
		"production Go contains KARMA references outside the exact reviewed emitter/helper surface:\n%s",
		strings.Join(findings, "\n"))
}

// Authority Geometry is an offline source observatory, not a runtime KARMA
// consumer. Admit only its exact reviewed policy vocabulary so this
// repository-wide tripwire still fails on any new identifier, import, or
// string reference in the checker (and everywhere else in production Go).
func exactAuthorityGraphKarmaPolicyRanges(sources []productionGoSource) ([]approvedSourceRange, error) {
	const authorityGraphManifest = "tools/authority-graph/manifest.go"
	expected := map[string]int{
		"identifier:CreatesRewardOrKarma":      2,
		"string:createsRewardOrKarma":          1,
		"string:json:\"createsRewardOrKarma\"": 1,
		"string:karma":                         2,
		"string:karma-to-authority":            1,
	}
	actual := map[string]int{}
	var ranges []approvedSourceRange
	found := false
	for _, source := range sources {
		if source.rel != authorityGraphManifest {
			continue
		}
		found = true
		ast.Inspect(source.file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			var kind, value string
			switch typed := node.(type) {
			case *ast.Ident:
				kind, value = "identifier", typed.Name
			case *ast.ImportSpec:
				kind, value = "import", unquoteGoString(typed.Path.Value)
			case *ast.BasicLit:
				if typed.Kind == token.STRING {
					kind, value = "string", unquoteGoString(typed.Value)
				}
			}
			if kind == "" || !strings.Contains(strings.ToLower(value), "karma") {
				return true
			}
			key := kind + ":" + value
			actual[key]++
			ranges = append(ranges, approvedSourceRange{
				rel: source.rel, start: node.Pos(), end: node.End(),
			})
			return true
		})
	}
	if !found {
		return nil, fmt.Errorf("missing %s", authorityGraphManifest)
	}
	if len(actual) != len(expected) {
		return nil, fmt.Errorf("Authority Geometry KARMA policy vocabulary drifted: got %v", actual)
	}
	for key, count := range expected {
		if actual[key] != count {
			return nil, fmt.Errorf("Authority Geometry KARMA policy vocabulary %q count = %d, want %d", key, actual[key], count)
		}
	}
	return ranges, nil
}

func TestMoneyKarma_KarmaAuditAdversarialFixtures(t *testing.T) {
	t.Run("constant event type plus transparent wrapper chain is transitive", func(t *testing.T) {
		sources := parseGoFixture(t, "app/wrapper_chain.go", `package fixture
			const edgeType = "zerone.karma.edge"
			func wrappedType() string { return edgeType }
			func literal() { sdk.NewEvent("zerone.karma.edge") }
			func constant() { sdk.NewEvent(edgeType) }
			func construct() { sdk.NewEvent(wrappedType()) }
			func wrapper() { construct() }
			func caller() { wrapper() }
			func noisyWrapper() { observe(); construct() }
			func noisyCaller() { noisyWrapper() }
			func unrelated() { sdk.NewEvent("zerone.other") }
		`)
		audit, err := analyzeKarmaProducerSurface(sources)
		require.NoError(t, err)
		require.True(t, audit.emitterNames["construct"])
		require.True(t, audit.emitterNames["literal"])
		require.True(t, audit.emitterNames["constant"])
		require.True(t, audit.emitterNames["wrapper"])
		require.True(t, audit.emitterNames["caller"])
		require.False(t, audit.emitterNames["noisyWrapper"])
		require.False(t, audit.emitterNames["noisyCaller"])
		require.False(t, audit.emitterNames["unrelated"])

		var callers []string
		for _, callsite := range audit.callsites {
			callers = append(callers, callsite.source.rel+":"+callsite.fn.Name.Name+"->"+callsite.callee)
			require.ErrorContains(
				t,
				validateKnownKarmaCallsiteShape(callsite, audit.stringValues, audit.stringFunctions),
				"unreviewed transitive KARMA helper",
				"a newly introduced wrapper/caller must fail the production allowlist",
			)
		}
		sort.Strings(callers)
		require.Equal(t, []string{
			"app/wrapper_chain.go:caller->wrapper",
			"app/wrapper_chain.go:noisyWrapper->construct",
			"app/wrapper_chain.go:wrapper->construct",
		}, callers)
	})

	t.Run("new kind fails the closed ordinal policy", func(t *testing.T) {
		sources := parseGoFixture(t, "x/knowledge/keeper/new_kind.go", `package keeper
			func emitKarmaEdgeState(ctx any, kind, state, beneficiary, counterparty, refID, domain string, extra ...any) {
				sdk.NewEvent("zerone.karma.edge")
			}
			func emitKarmaEdge(ctx any, kind, beneficiary, counterparty, refID, domain string, extra ...any) {
				emitKarmaEdgeState(ctx, kind, "RECOGNIZED", beneficiary, counterparty, refID, domain, extra...)
			}
			func newKindCaller() { emitKarmaEdge(ctx, "ranked", beneficiary, counterparty, refID, domain) }
		`)
		audit, err := analyzeKarmaProducerSurface(sources)
		require.NoError(t, err)
		var rejected error
		for _, callsite := range audit.callsites {
			if callsite.fn.Name.Name == "newKindCaller" {
				_, shapeErr := karmaCallsiteFingerprint(callsite, audit.stringValues, audit.stringFunctions)
				require.NoError(t, shapeErr)
				rejected = validateKnownKarmaCallsiteShape(callsite, audit.stringValues, audit.stringFunctions)
			}
		}
		require.ErrorContains(t, rejected, `kind "ranked"`)
	})

	t.Run("consumer outside the historical producer directories is found", func(t *testing.T) {
		sources := parseGoFixture(t, "app/hidden_consumer.go", `package app
			func consume() string { return "zerone.karma.edge" }
		`)
		findings := unapprovedKarmaReferences(sources, nil)
		require.Len(t, findings, 1)
		require.Contains(t, findings[0], "app/hidden_consumer.go")
	})
}

func requireFalse(t *testing.T, fields map[string]*bool) {
	t.Helper()
	for name, value := range fields {
		require.NotNil(t, value, "%s must be explicitly present", name)
		require.False(t, *value, "%s must remain false", name)
	}
}

func requireTrue(t *testing.T, fields map[string]*bool) {
	t.Helper()
	for name, value := range fields {
		require.NotNil(t, value, "%s must be explicitly present", name)
		require.True(t, *value, "%s must remain true", name)
	}
}

func requireString(t *testing.T, name string, actual *string, expected string) {
	t.Helper()
	require.NotNil(t, actual, "%s must be explicitly present", name)
	require.Equal(t, expected, *actual, "%s drifted", name)
}

func requireUint64(t *testing.T, name string, actual *uint64, expected uint64) {
	t.Helper()
	require.NotNil(t, actual, "%s must be explicitly present", name)
	require.Equal(t, expected, *actual, "%s drifted", name)
}

type productionGoSource struct {
	rel  string
	fset *token.FileSet
	file *ast.File
}

type productionGoFunction struct {
	source productionGoSource
	fn     *ast.FuncDecl
}

func (fn productionGoFunction) id() string {
	return fn.source.rel + ":" + fn.fn.Name.Name
}

type karmaHelperCallsite struct {
	source productionGoSource
	fn     *ast.FuncDecl
	call   *ast.CallExpr
	callee string
}

type approvedSourceRange struct {
	rel        string
	start, end token.Pos
}

type karmaProducerAudit struct {
	directConstructors map[string]productionGoFunction
	emitterFunctions   map[string]productionGoFunction
	emitterNames       map[string]bool
	callsites          []karmaHelperCallsite
	stringValues       map[string]string
	stringFunctions    map[string]string
	approvedRanges     []approvedSourceRange
}

func parseProductionGoSources(repoRoot string) ([]productionGoSource, error) {
	var sources []productionGoSource
	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".worktrees", "node_modules", "vendor", "dist", "build", "tests", "testdata":
				if path != repoRoot {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		sources = append(sources, productionGoSource{
			rel:  filepath.ToSlash(rel),
			fset: fset,
			file: file,
		})
		return nil
	})
	return sources, err
}

func parseGoFixture(t *testing.T, rel, source string) []productionGoSource {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, source, 0)
	require.NoError(t, err)
	return []productionGoSource{{rel: rel, fset: fset, file: file}}
}

func allProductionGoFunctions(sources []productionGoSource) []productionGoFunction {
	var functions []productionGoFunction
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Body != nil {
				functions = append(functions, productionGoFunction{source: source, fn: fn})
			}
		}
	}
	return functions
}

// analyzeKarmaProducerSurface recognizes direct constructors through literals,
// constants, local aliases, and zero-argument string wrappers. It then follows
// transparent emitter wrappers to a fixed point. Names are treated
// conservatively across packages: a collision expands the review surface.
func analyzeKarmaProducerSurface(sources []productionGoSource) (karmaProducerAudit, error) {
	values, stringFunctions := staticStringSymbols(sources)
	functions := allProductionGoFunctions(sources)
	audit := karmaProducerAudit{
		directConstructors: map[string]productionGoFunction{},
		emitterFunctions:   map[string]productionGoFunction{},
		emitterNames:       map[string]bool{},
		stringValues:       values,
		stringFunctions:    stringFunctions,
	}

	for _, fn := range functions {
		if len(karmaEventConstructorCalls(fn, values, stringFunctions)) == 0 {
			continue
		}
		audit.directConstructors[fn.id()] = fn
		audit.emitterFunctions[fn.id()] = fn
		audit.emitterNames[fn.fn.Name.Name] = true
	}

	for changed := true; changed; {
		changed = false
		for _, fn := range functions {
			if audit.emitterNames[fn.fn.Name.Name] || !isTransparentEmitterWrapper(fn.fn, audit.emitterNames) {
				continue
			}
			audit.emitterNames[fn.fn.Name.Name] = true
			audit.emitterFunctions[fn.id()] = fn
			changed = true
		}
	}

	for _, fn := range functions {
		ast.Inspect(fn.fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee := directCallName(call.Fun)
			if audit.emitterNames[callee] {
				audit.callsites = append(audit.callsites, karmaHelperCallsite{
					source: fn.source,
					fn:     fn.fn,
					call:   call,
					callee: callee,
				})
			}
			return true
		})
	}
	return audit, nil
}

func staticStringSymbols(sources []productionGoSource) (map[string]string, map[string]string) {
	values := map[string]string{}
	functions := map[string]string{}
	for iteration := 0; iteration < 128; iteration++ {
		valueCandidates := map[string]map[string]bool{}
		functionCandidates := map[string]map[string]bool{}
		unresolvedValues := map[string]bool{}
		unresolvedFunctions := map[string]bool{}
		for _, source := range sources {
			for _, decl := range source.file.Decls {
				switch typed := decl.(type) {
				case *ast.GenDecl:
					if typed.Tok != token.CONST {
						continue
					}
					for _, spec := range typed.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok || len(valueSpec.Values) == 0 {
							continue
						}
						for index, name := range valueSpec.Names {
							expr := valueSpec.Values[minInt(index, len(valueSpec.Values)-1)]
							value, ok := staticStringValue(expr, values, functions)
							if !ok {
								unresolvedValues[name.Name] = true
								continue
							}
							addStringCandidate(valueCandidates, name.Name, value)
						}
					}
				case *ast.FuncDecl:
					if typed.Recv != nil || typed.Body == nil || functionParameterCount(typed) != 0 || len(typed.Body.List) != 1 {
						continue
					}
					ret, ok := typed.Body.List[0].(*ast.ReturnStmt)
					if !ok || len(ret.Results) != 1 {
						continue
					}
					value, ok := staticStringValue(ret.Results[0], values, functions)
					if !ok {
						unresolvedFunctions[typed.Name.Name] = true
						continue
					}
					addStringCandidate(functionCandidates, typed.Name.Name, value)
				}
			}
		}
		nextValues := uniqueStringCandidates(valueCandidates, unresolvedValues)
		nextFunctions := uniqueStringCandidates(functionCandidates, unresolvedFunctions)
		if stringMapsEqual(values, nextValues) && stringMapsEqual(functions, nextFunctions) {
			return values, functions
		}
		values, functions = nextValues, nextFunctions
	}
	return map[string]string{}, map[string]string{}
}

func localStaticStringValues(fn *ast.FuncDecl, globals, functions map[string]string) map[string]string {
	localNames := map[string]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				localNames[name.Name] = true
			}
		case *ast.AssignStmt:
			for _, lhs := range typed.Lhs {
				if name, ok := lhs.(*ast.Ident); ok {
					localNames[name.Name] = true
				}
			}
		}
		return true
	})

	base := make(map[string]string, len(globals))
	for name, value := range globals {
		if !localNames[name] {
			base[name] = value
		}
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				delete(base, name.Name)
			}
		}
	}

	values := make(map[string]string, len(base))
	for name, value := range base {
		values[name] = value
	}
	for iteration := 0; iteration < 128; iteration++ {
		candidates := map[string]map[string]bool{}
		unresolved := map[string]bool{}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.ValueSpec:
				for index, name := range typed.Names {
					if len(typed.Values) == 0 {
						unresolved[name.Name] = true
						continue
					}
					expr := typed.Values[minInt(index, len(typed.Values)-1)]
					value, ok := staticStringValue(expr, values, functions)
					if !ok {
						unresolved[name.Name] = true
						continue
					}
					addStringCandidate(candidates, name.Name, value)
				}
			case *ast.AssignStmt:
				for index, lhs := range typed.Lhs {
					name, ok := lhs.(*ast.Ident)
					if !ok || len(typed.Rhs) == 0 {
						continue
					}
					expr := typed.Rhs[minInt(index, len(typed.Rhs)-1)]
					value, ok := staticStringValue(expr, values, functions)
					if !ok {
						unresolved[name.Name] = true
						continue
					}
					addStringCandidate(candidates, name.Name, value)
				}
			}
			return true
		})
		next := make(map[string]string, len(base)+len(candidates))
		for name, value := range base {
			next[name] = value
		}
		for name, value := range uniqueStringCandidates(candidates, unresolved) {
			next[name] = value
		}
		if stringMapsEqual(values, next) {
			return values
		}
		values = next
	}
	return base
}

func addStringCandidate(candidates map[string]map[string]bool, name, value string) {
	if candidates[name] == nil {
		candidates[name] = map[string]bool{}
	}
	candidates[name][value] = true
}

func uniqueStringCandidates(candidates map[string]map[string]bool, unresolved map[string]bool) map[string]string {
	result := map[string]string{}
	for name, values := range candidates {
		if unresolved[name] || len(values) != 1 {
			continue
		}
		for value := range values {
			result[name] = value
		}
	}
	return result
}

func stringMapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		if right[name] != value {
			return false
		}
	}
	return true
}

func staticStringValue(expr ast.Expr, values, functions map[string]string) (string, bool) {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return "", false
		}
		return unquoteGoString(typed.Value), true
	case *ast.Ident:
		value, ok := values[typed.Name]
		return value, ok
	case *ast.SelectorExpr:
		value, ok := values[typed.Sel.Name]
		return value, ok
	case *ast.CallExpr:
		ident, ok := typed.Fun.(*ast.Ident)
		if !ok || len(typed.Args) != 0 {
			return "", false
		}
		value, ok := functions[ident.Name]
		return value, ok
	case *ast.ParenExpr:
		return staticStringValue(typed.X, values, functions)
	default:
		return "", false
	}
}

func karmaEventConstructorCalls(fn productionGoFunction, globals, functions map[string]string) []*ast.CallExpr {
	localValues := localStaticStringValues(fn.fn, globals, functions)
	var calls []*ast.CallExpr
	ast.Inspect(fn.fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || directCallName(call.Fun) != "NewEvent" || len(call.Args) == 0 {
			return true
		}
		value, ok := staticStringValue(call.Args[0], localValues, functions)
		if ok && value == "zerone.karma.edge" {
			calls = append(calls, call)
		}
		return true
	})
	return calls
}

func isTransparentEmitterWrapper(fn *ast.FuncDecl, emitterNames map[string]bool) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	var call *ast.CallExpr
	switch stmt := fn.Body.List[0].(type) {
	case *ast.ExprStmt:
		call, _ = stmt.X.(*ast.CallExpr)
	case *ast.ReturnStmt:
		if len(stmt.Results) == 1 {
			call, _ = stmt.Results[0].(*ast.CallExpr)
		}
	}
	return call != nil && emitterNames[directCallName(call.Fun)]
}

func requireExactKarmaProducerSurface(t *testing.T, sources []productionGoSource) karmaProducerAudit {
	t.Helper()
	audit, err := analyzeKarmaProducerSurface(sources)
	require.NoError(t, err)

	var constructors []string
	for _, fn := range audit.directConstructors {
		fingerprint, fingerprintErr := karmaConstructorFingerprint(fn, audit.stringValues, audit.stringFunctions)
		require.NoError(t, fingerprintErr)
		constructors = append(constructors, fingerprint)
		for _, finding := range forbiddenKarmaEmitterOperations(fn) {
			require.Fail(t, "KARMA constructor performs a forbidden economic/state/governance operation", finding)
		}
	}
	sort.Strings(constructors)
	expectedConstructors := []string{
		"x/knowledge/keeper/rounds.go:emitKarmaEdgeState|attrs=beneficiary=$param:beneficiary,kind=$param:kind,state=$param:state,counterparty=$param:counterparty,ref_id=$param:refID,domain=$param:domain,register=priced-coherence|appends=self=true,spread:extra|emit=event",
		"x/substrate_bridge/keeper/settlement.go:emitExternalCiteKarma|attrs=beneficiary=fact.Submitter,kind=external,state=ORDINAL,counterparty=att.Submitter,ref_id=c.FactId,domain=fact.Domain,register=priced-coherence|appends=self=true|emit=event",
	}
	sort.Strings(expectedConstructors)
	require.Equal(t, expectedConstructors, constructors,
		"direct KARMA constructors and exact event attribute shapes require explicit constitutional review")

	var callsites []string
	for _, callsite := range audit.callsites {
		require.NoError(t, validateKnownKarmaCallsiteShape(callsite, audit.stringValues, audit.stringFunctions))
		fingerprint, fingerprintErr := karmaCallsiteFingerprint(callsite, audit.stringValues, audit.stringFunctions)
		require.NoError(t, fingerprintErr)
		callsites = append(callsites, fingerprint)
	}
	sort.Strings(callsites)
	expectedCallsites := []string{
		"x/knowledge/keeper/msg_server.go:ChallengeFact->emitKarmaEdgeState|kind=pending_open|state=ORDINAL|attrs=none",
		"x/knowledge/keeper/msg_server.go:ChallengeProvisionalFact->emitKarmaEdgeState|kind=pending_open|state=ORDINAL|attrs=none",
		"x/knowledge/keeper/msg_server.go:SubmitClaim->emitKarmaEdgeState|kind=pending_open|state=ORDINAL|attrs=none",
		"x/knowledge/keeper/msg_server.go:SubmitContradiction->emitKarmaEdgeState|kind=pending_open|state=ORDINAL|attrs=none",
		"x/knowledge/keeper/msg_server_conjecture.go:PostConjecture->emitKarmaEdgeState|kind=pending_open|state=ORDINAL|attrs=none",
		"x/knowledge/keeper/phases.go:AdvanceRoundPhases->emitKarmaEdgeState|kind=pending_settle|state=ORDINAL|attrs=verdict=types.Verdict_VERDICT_INCONCLUSIVE.String()",
		"x/knowledge/keeper/rounds.go:CompleteRound->emitKarmaEdgeState|kind=pending_settle|state=ORDINAL|attrs=verdict=result.Verdict.String()",
		"x/knowledge/keeper/rounds.go:createFactFromClaim->emitKarmaEdge|kind=cited|state=RECOGNIZED|attrs=none",
		"x/knowledge/keeper/rounds.go:distributeVerifierRewardsFromPool->emitKarmaEdge|kind=verify|state=RECOGNIZED|attrs=local:extra[correct=true]",
		"x/knowledge/keeper/rounds.go:emitKarmaEdge->emitKarmaEdgeState|kind=$param:kind|state=RECOGNIZED|attrs=forward-param:extra",
		"x/knowledge/keeper/rounds.go:handleChallengeSurvival->emitKarmaEdge|kind=corroborate|state=RECOGNIZED|attrs=none",
		"x/knowledge/keeper/rounds.go:handleChallengeSurvival->emitKarmaEdge|kind=corroborated|state=RECOGNIZED|attrs=none",
		"x/substrate_bridge/keeper/settlement.go:finalizeSettle->emitExternalCiteKarma|kind=external|state=ORDINAL|attrs=constructor-closed",
	}
	sort.Strings(expectedCallsites)
	require.Equal(t, expectedCallsites, callsites,
		"every KARMA helper callsite is closed by file:function, kind, state, and attribute shape")

	for _, fn := range audit.emitterFunctions {
		audit.approvedRanges = append(audit.approvedRanges, approvedSourceRange{
			rel: fn.source.rel, start: fn.fn.Name.Pos(), end: fn.fn.Name.End(),
		})
	}
	for _, fn := range audit.directConstructors {
		for _, call := range karmaEventConstructorCalls(fn, audit.stringValues, audit.stringFunctions) {
			audit.approvedRanges = append(audit.approvedRanges, approvedSourceRange{
				rel: fn.source.rel, start: call.Pos(), end: call.End(),
			})
		}
	}
	for _, callsite := range audit.callsites {
		audit.approvedRanges = append(audit.approvedRanges, approvedSourceRange{
			rel: callsite.source.rel, start: callsite.call.Pos(), end: callsite.call.End(),
		})
	}
	return audit
}

func karmaConstructorFingerprint(fn productionGoFunction, globals, functions map[string]string) (string, error) {
	calls := karmaEventConstructorCalls(fn, globals, functions)
	if len(calls) != 1 {
		return "", fmt.Errorf("%s must contain exactly one KARMA NewEvent constructor, got %d", fn.id(), len(calls))
	}
	localValues := localStaticStringValues(fn.fn, globals, functions)
	constructor := calls[0]
	var attrs []string
	for _, arg := range constructor.Args[1:] {
		attr, err := describeKarmaAttribute(fn, arg, localValues, functions)
		if err != nil {
			return "", fmt.Errorf("%s constructor: %w", fn.id(), err)
		}
		attrs = append(attrs, attr)
	}

	var appends, emits []string
	ast.Inspect(fn.fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch directCallName(call.Fun) {
		case "AppendAttributes":
			for index, arg := range call.Args {
				if call.Ellipsis.IsValid() && index == len(call.Args)-1 {
					appends = append(appends, "spread:"+renderExpr(fn.source.fset, arg))
					continue
				}
				attr, err := describeKarmaAttribute(fn, arg, localValues, functions)
				if err != nil {
					appends = append(appends, "ERROR:"+err.Error())
				} else {
					appends = append(appends, attr)
				}
			}
		case "EmitEvent":
			for _, arg := range call.Args {
				emits = append(emits, renderExpr(fn.source.fset, arg))
			}
		}
		return true
	})
	return fmt.Sprintf("%s|attrs=%s|appends=%s|emit=%s",
		fn.id(), strings.Join(attrs, ","), strings.Join(appends, ","), strings.Join(emits, ",")), nil
}

func forbiddenKarmaEmitterOperations(fn productionGoFunction) []string {
	var forbidden []string
	ast.Inspect(fn.fn.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.Ident:
			lower := strings.ToLower(typed.Name)
			if lower == "coin" || lower == "coins" || lower == "denom" || lower == "balance" ||
				lower == "magnitude" || lower == "amount" || lower == "bank" || lower == "bankkeeper" {
				forbidden = append(forbidden, fmt.Sprintf("%s:%d identifier %q", fn.id(), fn.source.fset.Position(typed.Pos()).Line, typed.Name))
			}
		case *ast.CallExpr:
			name := directCallName(typed.Fun)
			lower := strings.ToLower(name)
			if expressionContainsIdentifier(typed, "event") && name != "AppendAttributes" && name != "EmitEvent" {
				forbidden = append(forbidden, fmt.Sprintf("%s:%d passes the KARMA event through unreviewed call %q",
					fn.id(), fn.source.fset.Position(typed.Pos()).Line, name))
			}
			if strings.HasPrefix(lower, "set") || strings.HasPrefix(lower, "store") ||
				strings.HasPrefix(lower, "save") || strings.HasPrefix(lower, "delete") ||
				strings.HasPrefix(lower, "insert") || strings.HasPrefix(lower, "update") ||
				strings.Contains(lower, "coin") || strings.Contains(lower, "bank") ||
				strings.Contains(lower, "mint") || strings.Contains(lower, "burn") ||
				strings.Contains(lower, "transfer") || strings.Contains(lower, "delegate") ||
				strings.Contains(lower, "vote") || strings.Contains(lower, "proposal") {
				forbidden = append(forbidden, fmt.Sprintf("%s:%d call %q", fn.id(), fn.source.fset.Position(typed.Pos()).Line, name))
			}
		case *ast.AssignStmt:
			for index, lhs := range typed.Lhs {
				if !expressionContainsIdentifier(lhs, "event") || len(typed.Rhs) == 0 {
					continue
				}
				rhs := typed.Rhs[minInt(index, len(typed.Rhs)-1)]
				call, ok := rhs.(*ast.CallExpr)
				if !ok || (directCallName(call.Fun) != "NewEvent" && directCallName(call.Fun) != "AppendAttributes") {
					forbidden = append(forbidden, fmt.Sprintf("%s:%d mutates the KARMA event through an unreviewed assignment",
						fn.id(), fn.source.fset.Position(typed.Pos()).Line))
				}
			}
		}
		return true
	})
	return forbidden
}

func expressionContainsIdentifier(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(candidate ast.Node) bool {
		ident, ok := candidate.(*ast.Ident)
		if ok && ident.Name == name {
			found = true
			return false
		}
		return !found
	})
	return found
}

type karmaCallsiteShape struct {
	kind, state, attrs string
}

func karmaCallsiteFingerprint(callsite karmaHelperCallsite, globals, functions map[string]string) (string, error) {
	shape, err := describeKarmaCallsite(callsite, globals, functions)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s->%s|kind=%s|state=%s|attrs=%s",
		callsite.source.rel, callsite.fn.Name.Name, callsite.callee, shape.kind, shape.state, shape.attrs), nil
}

func describeKarmaCallsite(callsite karmaHelperCallsite, globals, functions map[string]string) (karmaCallsiteShape, error) {
	localValues := localStaticStringValues(callsite.fn, globals, functions)
	call := callsite.call
	switch callsite.callee {
	case "emitKarmaEdge":
		if len(call.Args) < 6 {
			return karmaCallsiteShape{}, fmt.Errorf("%s:%s emitKarmaEdge has %d args", callsite.source.rel, callsite.fn.Name.Name, len(call.Args))
		}
		attrs, err := describeKarmaExtraAttributes(callsite, 6, localValues, functions)
		return karmaCallsiteShape{
			kind:  describeKarmaValue(callsite.source.fset, callsite.fn, call.Args[1], localValues, functions),
			state: "RECOGNIZED",
			attrs: attrs,
		}, err
	case "emitKarmaEdgeState":
		if len(call.Args) < 7 {
			return karmaCallsiteShape{}, fmt.Errorf("%s:%s emitKarmaEdgeState has %d args", callsite.source.rel, callsite.fn.Name.Name, len(call.Args))
		}
		attrs, err := describeKarmaExtraAttributes(callsite, 7, localValues, functions)
		return karmaCallsiteShape{
			kind:  describeKarmaValue(callsite.source.fset, callsite.fn, call.Args[1], localValues, functions),
			state: describeKarmaValue(callsite.source.fset, callsite.fn, call.Args[2], localValues, functions),
			attrs: attrs,
		}, err
	case "emitExternalCiteKarma":
		if len(call.Args) != 2 {
			return karmaCallsiteShape{}, fmt.Errorf("%s:%s emitExternalCiteKarma must receive context and attestation", callsite.source.rel, callsite.fn.Name.Name)
		}
		return karmaCallsiteShape{kind: "external", state: "ORDINAL", attrs: "constructor-closed"}, nil
	default:
		return karmaCallsiteShape{}, fmt.Errorf("unreviewed transitive KARMA helper %s called by %s:%s",
			callsite.callee, callsite.source.rel, callsite.fn.Name.Name)
	}
}

func validateKnownKarmaCallsiteShape(callsite karmaHelperCallsite, globals, functions map[string]string) error {
	shape, err := describeKarmaCallsite(callsite, globals, functions)
	if err != nil {
		return err
	}
	switch callsite.callee {
	case "emitKarmaEdge":
		if shape.state != "RECOGNIZED" {
			return fmt.Errorf("emitKarmaEdge state %q is not permitted", shape.state)
		}
		switch shape.kind {
		case "verify":
			if shape.attrs != "local:extra[correct=true]" {
				return fmt.Errorf("verify attributes %q are not permitted", shape.attrs)
			}
		case "cited", "corroborate", "corroborated":
			if shape.attrs != "none" {
				return fmt.Errorf("%s attributes %q are not permitted", shape.kind, shape.attrs)
			}
		default:
			return fmt.Errorf("emitKarmaEdge kind %q is not permitted", shape.kind)
		}
	case "emitKarmaEdgeState":
		switch shape.kind {
		case "$param:kind":
			if callsite.source.rel != "x/knowledge/keeper/rounds.go" || callsite.fn.Name.Name != "emitKarmaEdge" ||
				shape.state != "RECOGNIZED" || shape.attrs != "forward-param:extra" {
				return fmt.Errorf("dynamic KARMA kind forwarding is only permitted in the exact recognized wrapper")
			}
		case "pending_open":
			if shape.state != "ORDINAL" || shape.attrs != "none" {
				return fmt.Errorf("pending_open must remain ORDINAL with no extra attributes")
			}
		case "pending_settle":
			if shape.state != "ORDINAL" || (shape.attrs != "verdict=result.Verdict.String()" &&
				shape.attrs != "verdict=types.Verdict_VERDICT_INCONCLUSIVE.String()") {
				return fmt.Errorf("pending_settle must remain ORDINAL with one closed verdict attribute, got %q", shape.attrs)
			}
		default:
			return fmt.Errorf("emitKarmaEdgeState kind %q is not permitted", shape.kind)
		}
	case "emitExternalCiteKarma":
		if shape.kind != "external" || shape.state != "ORDINAL" || shape.attrs != "constructor-closed" {
			return fmt.Errorf("external citation shape drifted: %+v", shape)
		}
	default:
		return fmt.Errorf("unreviewed transitive KARMA helper %q", callsite.callee)
	}
	return nil
}

func describeKarmaExtraAttributes(
	callsite karmaHelperCallsite,
	start int,
	values, functions map[string]string,
) (string, error) {
	if len(callsite.call.Args) == start {
		return "none", nil
	}
	var attrs []string
	for index, arg := range callsite.call.Args[start:] {
		isSpread := callsite.call.Ellipsis.IsValid() && index == len(callsite.call.Args[start:])-1
		if isSpread {
			ident, ok := arg.(*ast.Ident)
			if !ok {
				return "", fmt.Errorf("%s:%s spreads a non-identifier KARMA attribute source", callsite.source.rel, callsite.fn.Name.Name)
			}
			if functionParameterNamed(callsite.fn, ident.Name) {
				attrs = append(attrs, "forward-param:"+ident.Name)
				continue
			}
			local, err := describeLocalKarmaAttributeSlice(callsite, ident.Name, values, functions)
			if err != nil {
				return "", err
			}
			attrs = append(attrs, local)
			continue
		}
		attr, err := describeKarmaAttribute(
			productionGoFunction{source: callsite.source, fn: callsite.fn}, arg, values, functions)
		if err != nil {
			return "", err
		}
		attrs = append(attrs, attr)
	}
	return strings.Join(attrs, ","), nil
}

func describeLocalKarmaAttributeSlice(
	callsite karmaHelperCallsite,
	name string,
	values, functions map[string]string,
) (string, error) {
	var attrs []string
	var invalid error
	ast.Inspect(callsite.fn.Body, func(node ast.Node) bool {
		if invalid != nil {
			return false
		}
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for index, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok || ident.Name != name || len(assign.Rhs) == 0 {
				continue
			}
			rhs := assign.Rhs[minInt(index, len(assign.Rhs)-1)]
			appendCall, ok := rhs.(*ast.CallExpr)
			if !ok || directCallName(appendCall.Fun) != "append" || len(appendCall.Args) < 2 {
				invalid = fmt.Errorf("%s:%s dynamically assigns KARMA attribute slice %q", callsite.source.rel, callsite.fn.Name.Name, name)
				return false
			}
			base, ok := appendCall.Args[0].(*ast.Ident)
			if !ok || base.Name != name {
				invalid = fmt.Errorf("%s:%s replaces KARMA attribute slice %q", callsite.source.rel, callsite.fn.Name.Name, name)
				return false
			}
			for _, appended := range appendCall.Args[1:] {
				attr, err := describeKarmaAttribute(
					productionGoFunction{source: callsite.source, fn: callsite.fn}, appended, values, functions)
				if err != nil {
					invalid = err
					return false
				}
				attrs = append(attrs, attr)
			}
		}
		return true
	})
	if invalid != nil {
		return "", invalid
	}
	if len(attrs) == 0 {
		return "", fmt.Errorf("%s:%s forwards unproved KARMA attribute slice %q", callsite.source.rel, callsite.fn.Name.Name, name)
	}
	sort.Strings(attrs)
	return "local:" + name + "[" + strings.Join(attrs, ",") + "]", nil
}

func describeKarmaAttribute(
	fn productionGoFunction,
	expr ast.Expr,
	values, functions map[string]string,
) (string, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || directCallName(call.Fun) != "NewAttribute" || len(call.Args) != 2 {
		return "", fmt.Errorf("%s KARMA metadata must be an explicit two-argument NewAttribute", fn.id())
	}
	key, ok := staticStringValue(call.Args[0], values, functions)
	if !ok {
		return "", fmt.Errorf("%s KARMA metadata key is dynamic", fn.id())
	}
	value := describeKarmaValue(fn.source.fset, fn.fn, call.Args[1], values, functions)
	return key + "=" + value, nil
}

func describeKarmaValue(
	fset *token.FileSet,
	fn *ast.FuncDecl,
	expr ast.Expr,
	values, functions map[string]string,
) string {
	if ident, ok := expr.(*ast.Ident); ok && functionParameterNamed(fn, ident.Name) {
		return "$param:" + ident.Name
	}
	if value, ok := staticStringValue(expr, values, functions); ok {
		return value
	}
	return renderExpr(fset, expr)
}

func renderExpr(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return fmt.Sprintf("<unprintable:%v>", err)
	}
	return buf.String()
}

func exactKarmaSupportConstantRanges(sources []productionGoSource) ([]approvedSourceRange, error) {
	expected := map[string]string{
		"x/knowledge/keeper/rounds.go:karmaEdgeStateRecognized":          "RECOGNIZED",
		"x/knowledge/keeper/rounds.go:karmaEdgeStateOrdinal":             "ORDINAL",
		"x/knowledge/keeper/rounds.go:karmaRegister":                     "priced-coherence",
		"x/substrate_bridge/types/karma.go:EventTypeKarmaEdge":           "zerone.karma.edge",
		"x/substrate_bridge/types/karma.go:AttrKarmaBeneficiary":         "beneficiary",
		"x/substrate_bridge/types/karma.go:AttrKarmaKind":                "kind",
		"x/substrate_bridge/types/karma.go:AttrKarmaState":               "state",
		"x/substrate_bridge/types/karma.go:AttrKarmaCounterparty":        "counterparty",
		"x/substrate_bridge/types/karma.go:AttrKarmaRefID":               "ref_id",
		"x/substrate_bridge/types/karma.go:AttrKarmaDomain":              "domain",
		"x/substrate_bridge/types/karma.go:AttrKarmaSelf":                "self",
		"x/substrate_bridge/types/karma.go:AttrKarmaRegister":            "register",
		"x/substrate_bridge/types/karma.go:KarmaKindExternal":            "external",
		"x/substrate_bridge/types/karma.go:KarmaStateOrdinal":            "ORDINAL",
		"x/substrate_bridge/types/karma.go:KarmaRegisterPricedCoherence": "priced-coherence",
	}
	values, functions := staticStringSymbols(sources)
	seen := map[string]bool{}
	var ranges []approvedSourceRange
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || len(valueSpec.Values) == 0 {
					continue
				}
				for index, name := range valueSpec.Names {
					key := source.rel + ":" + name.Name
					expectedValue, approved := expected[key]
					if !approved {
						continue
					}
					expr := valueSpec.Values[minInt(index, len(valueSpec.Values)-1)]
					actualValue, ok := staticStringValue(expr, values, functions)
					if !ok || actualValue != expectedValue {
						return nil, fmt.Errorf("KARMA support constant %s must equal %q, got %q", key, expectedValue, actualValue)
					}
					if seen[key] {
						return nil, fmt.Errorf("duplicate KARMA support constant %s", key)
					}
					seen[key] = true
					ranges = append(ranges,
						approvedSourceRange{rel: source.rel, start: name.Pos(), end: name.End()},
						approvedSourceRange{rel: source.rel, start: expr.Pos(), end: expr.End()},
					)
				}
			}
		}
	}
	var missing []string
	for key := range expected {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing exact KARMA support constants: %s", strings.Join(missing, ", "))
	}
	return ranges, nil
}

func unapprovedKarmaReferences(sources []productionGoSource, approved []approvedSourceRange) []string {
	var findings []string
	for _, source := range sources {
		ast.Inspect(source.file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			var kind, value string
			switch typed := node.(type) {
			case *ast.Ident:
				kind, value = "identifier", typed.Name
			case *ast.ImportSpec:
				kind, value = "import", unquoteGoString(typed.Path.Value)
			case *ast.BasicLit:
				if typed.Kind == token.STRING {
					kind, value = "string", unquoteGoString(typed.Value)
				}
			}
			if kind == "" || !strings.Contains(strings.ToLower(value), "karma") ||
				positionApproved(source.rel, node.Pos(), node.End(), approved) {
				return true
			}
			findings = append(findings, fmt.Sprintf("%s:%d: %s %q",
				source.rel, source.fset.Position(node.Pos()).Line, kind, value))
			return true
		})
	}
	sort.Strings(findings)
	return findings
}

func positionApproved(rel string, start, end token.Pos, approved []approvedSourceRange) bool {
	for _, candidate := range approved {
		if candidate.rel == rel && start >= candidate.start && end <= candidate.end {
			return true
		}
	}
	return false
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := consumeUniqueJSONValue(decoder, "$"); err != nil {
		return err
	}
	if tokenValue, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return fmt.Errorf("trailing JSON token: %w", err)
		}
		return fmt.Errorf("trailing JSON value beginning with %v", tokenValue)
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder, path string) error {
	tokenValue, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelim := tokenValue.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key at %s is not a string", path)
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q at %s", key, path)
			}
			seen[key] = true
			if err := consumeUniqueJSONValue(decoder, fmt.Sprintf("%s[%q]", path, key)); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object at %s closed with %v", path, closing)
		}
	case '[':
		for index := 0; decoder.More(); index++ {
			if err := consumeUniqueJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array at %s closed with %v", path, closing)
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q at %s", delim, path)
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func functionParameterCount(fn *ast.FuncDecl) int {
	if fn.Type.Params == nil {
		return 0
	}
	count := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			count++
		} else {
			count += len(field.Names)
		}
	}
	return count
}

func functionParameterNamed(fn *ast.FuncDecl, name string) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		for _, fieldName := range field.Names {
			if fieldName.Name == name {
				return true
			}
		}
	}
	return false
}

func directCallName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func unquoteGoString(value string) string {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}
	return unquoted
}
