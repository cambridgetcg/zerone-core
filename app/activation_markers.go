package app

import (
	"context"

	zeronealignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
	zeronecctypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
	zeronecdtypes "github.com/zerone-chain/zerone/x/capture_defense/types"
	zeroneemergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	zeroneibcrltypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
	zeroneknowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	zeroneontologytypes "github.com/zerone-chain/zerone/x/ontology/types"
	zeronequalificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ActivationMarker is one default-off activation marker, bound to this app's
// keeper. Name is the field of the module's genesis JSON that carries it.
// Enabled and Enable are the keeper's own reader and enable function, with
// internal/activation's contract.
type ActivationMarker struct {
	Module  string
	Name    string
	Enabled func(context.Context) (bool, error)
	Enable  func(context.Context) error
}

// ActivationSelection is the native/imported genesis contract, independent of
// keeper construction. Tests pin this static list to the production registry.
type ActivationSelection struct {
	Module string
	Name   string
}

// ConsensusRepairsDeferredGenesisSelections are markers whose code is in this
// source but which no carrier here may switch on: native genesis must leave
// each one absent or false. None remain: #122's bank-supply denominator, which
// consensus-repairs-v1 deferred, is a consensus-repairs-v2 selection.
func ConsensusRepairsDeferredGenesisSelections() []ActivationSelection {
	return nil
}

func ConsensusRepairsV1GenesisSelections() []ActivationSelection {
	return []ActivationSelection{
		{zeronegovtypes.ModuleName, "seat_runoff_ordering_enabled"},
		{zeroneknowledgetypes.ModuleName, "reformulation_vindication_ordering_enabled"},
		{zeronecdtypes.ModuleName, "integer_variance_enabled"},
		{zeronequalificationtypes.ModuleName, "single_refund_enabled"},
		{zeronecctypes.ModuleName, "challenge_accounting_enabled"},
		{zeroneontologytypes.ModuleName, "active_domain_count_enabled"},
		{zeroneemergencytypes.ModuleName, "halt_reason_scoping_enabled"},
		{substratebridgetypes.ModuleName, "qualification_semantics_enabled"},
		{zeroneibcrltypes.ModuleName, "refund_identity_enabled"},
	}
}

// ConsensusRepairsV1Markers lists every consensus-repairs-v1 marker in the
// order of the repairs' zerone-core pull requests (#117-#127). Native genesis
// selects each one through its module's genesis field; the carrier upgrade
// enables each one by looping over this list. It lives outside app.go because
// the authority-graph manifest pins app.go's SHA-256.
//
// #122's alignment bank-supply marker is deliberately absent. On an empty
// staking registry with a zero mint ledger and positive bank supply (zerone-dev-1
// today) it turns the neutral economic and staking readings into zero, so it
// may only switch at the same height as Q1's not-observed neutrality:
// ConsensusRepairsV2Markers carries it, right after Q1.
func (app *ZeroneApp) ConsensusRepairsV1Markers() []ActivationMarker {
	return []ActivationMarker{
		{zeronegovtypes.ModuleName, "seat_runoff_ordering_enabled",
			app.ZeroneGovKeeper.SeatRunoffOrderingEnabled, app.ZeroneGovKeeper.EnableSeatRunoffOrdering},
		{zeroneknowledgetypes.ModuleName, "reformulation_vindication_ordering_enabled",
			app.KnowledgeKeeper.ReformulationVindicationOrderingEnabled, app.KnowledgeKeeper.EnableReformulationVindicationOrdering},
		{zeronecdtypes.ModuleName, "integer_variance_enabled",
			app.CaptureDefenseKeeper.IntegerVarianceEnabled, app.CaptureDefenseKeeper.EnableIntegerVariance},
		{zeronequalificationtypes.ModuleName, "single_refund_enabled",
			app.QualificationKeeper.SingleRefundEnabled, app.QualificationKeeper.EnableSingleRefund},
		{zeronecctypes.ModuleName, "challenge_accounting_enabled",
			app.CaptureChallengeKeeper.ChallengeAccountingEnabled, app.CaptureChallengeKeeper.EnableChallengeAccounting},
		{zeroneontologytypes.ModuleName, "active_domain_count_enabled",
			app.ZeroneOntologyKeeper.ActiveDomainCountEnabled, app.ZeroneOntologyKeeper.EnableActiveDomainCount},
		{zeroneemergencytypes.ModuleName, "halt_reason_scoping_enabled",
			app.EmergencyKeeper.HaltReasonScopingEnabled, app.EmergencyKeeper.EnableHaltReasonScoping},
		{substratebridgetypes.ModuleName, "qualification_semantics_enabled",
			app.SubstrateBridgeKeeper.QualificationSemanticsEnabled, app.SubstrateBridgeKeeper.EnableQualificationSemantics},
		{zeroneibcrltypes.ModuleName, "refund_identity_enabled",
			app.IBCRateLimitKeeper.RefundIdentityEnabled, app.IBCRateLimitKeeper.EnableRefundIdentity},
	}
}

// ConsensusRepairsV2Markers lists every consensus-repairs-v2 marker in the
// order of their decisions in the decision ledger: zc-health-unknown-q1,
// zc-panel-unanimity-q2, zc-calibration-cliff-q5 and zc-pr37-frontier-fixes.
// #122's alignment bank-supply denominator follows Q1 directly: it may switch
// only at Q1's height (review finding F1), so it moved here from v1.
// Native genesis selects each one through its module's genesis field; the
// consensus-repairs-v2 carrier enables each one by looping over this list.
// It shares no module/field pair or store key with ConsensusRepairsV1Markers.
// The genesis-only FX7 and FX8 schemas change export and import, never a live
// state transition, so they have no marker here.
func (app *ZeroneApp) ConsensusRepairsV2Markers() []ActivationMarker {
	return []ActivationMarker{
		{zeronealignmenttypes.ModuleName, "unobserved_health_enabled",
			app.AlignmentKeeper.UnobservedHealthEnabled, app.AlignmentKeeper.EnableUnobservedHealth},
		{vestingrewardstypes.ModuleName, "alignment_bank_supply_enabled",
			app.VestingRewardsKeeper.AlignmentBankSupplyEnabled, app.VestingRewardsKeeper.EnableAlignmentBankSupply},
		{zeroneknowledgetypes.ModuleName, "inconclusive_round_accounting_enabled",
			app.KnowledgeKeeper.InconclusiveRoundAccountingEnabled, app.KnowledgeKeeper.EnableInconclusiveRoundAccounting},
		{zeroneknowledgetypes.ModuleName, "calibration_smoothing_enabled",
			app.KnowledgeKeeper.CalibrationSmoothingEnabled, app.KnowledgeKeeper.EnableCalibrationSmoothing},
		{zeroneknowledgetypes.ModuleName, "zero_confidence_floor_enabled",
			app.KnowledgeKeeper.ZeroConfidenceFloorEnabled, app.KnowledgeKeeper.EnableZeroConfidenceFloor},
	}
}

func ConsensusRepairsV2GenesisSelections() []ActivationSelection {
	return []ActivationSelection{
		{zeronealignmenttypes.ModuleName, "unobserved_health_enabled"},
		{vestingrewardstypes.ModuleName, "alignment_bank_supply_enabled"},
		{zeroneknowledgetypes.ModuleName, "inconclusive_round_accounting_enabled"},
		{zeroneknowledgetypes.ModuleName, "calibration_smoothing_enabled"},
		{zeroneknowledgetypes.ModuleName, "zero_confidence_floor_enabled"},
	}
}

// ConsensusRepairsV3Markers lists the consensus-repairs-v3 markers:
//
//   - knowledge/doctrine_identity_enabled (knowledge 0x94): the metabolism
//     exemption for doctrine follows provenance instead of the category label.
//   - substrate_bridge/cited_reward_escrow_enabled (substrate_bridge 0x91): the
//     reward for an attestation that cites existing facts waits in the existing
//     witness escrow window.
//   - knowledge/dormant_fact_contest_enabled (knowledge 0x95): a fact that lost
//     standing through disuse stays open to challenge and contradiction, and a
//     pruned conjecture stays an open question.
//   - knowledge/reviewer_balance_param_enabled (knowledge 0x96): the balance a
//     reviewer must hold to commit is read from a knowledge parameter instead
//     of a fixed constant.
//   - knowledge/seat_deposit_enabled (knowledge 0x97): a review round records
//     seat terms, and each commitment in it locks the round's deposit, which
//     is returned in full. Its Enable requires the reviewer-balance parameter
//     (listed just before it), review neutrality and record integrity, and
//     the seat-deposit custody (EnableSeatDeposit), so a carrier looping over
//     this list in order turns it on last.
//
// Native genesis selects each one through its module's genesis field, and a
// genesis without the field imports with the marker off. No carrier upgrade,
// census or version step exists for v3 yet, so nothing turns these markers on
// for an existing chain until a carrier is written. That carrier will loop over
// this list, and will add InitChain's requirement that native genesis select
// ConsensusRepairsV3GenesisSelections explicitly, as v2's carrier did for v2.
// It shares no module/field pair or store key with ConsensusRepairsV1Markers
// or ConsensusRepairsV2Markers.
func (app *ZeroneApp) ConsensusRepairsV3Markers() []ActivationMarker {
	return []ActivationMarker{
		{zeroneknowledgetypes.ModuleName, "doctrine_identity_enabled",
			app.KnowledgeKeeper.DoctrineIdentityEnabled, app.KnowledgeKeeper.EnableDoctrineIdentity},
		{substratebridgetypes.ModuleName, "cited_reward_escrow_enabled",
			app.SubstrateBridgeKeeper.CitedRewardEscrowEnabled, app.SubstrateBridgeKeeper.EnableCitedRewardEscrow},
		{zeroneknowledgetypes.ModuleName, "dormant_fact_contest_enabled",
			app.KnowledgeKeeper.DormantFactContestEnabled, app.KnowledgeKeeper.EnableDormantFactContest},
		{zeroneknowledgetypes.ModuleName, "reviewer_balance_param_enabled",
			app.KnowledgeKeeper.ReviewerBalanceParamEnabled, app.KnowledgeKeeper.EnableReviewerBalanceParam},
		{zeroneknowledgetypes.ModuleName, "seat_deposit_enabled",
			app.KnowledgeKeeper.SeatDepositEnabled, app.KnowledgeKeeper.EnableSeatDeposit},
	}
}

// ConsensusRepairsV3GenesisSelections is the native genesis selection of the
// v3 markers, in ConsensusRepairsV3Markers order. DefaultGenesis sets each
// field true. ValidateConsensusRepairsGenesisSelection does not require them
// yet: the v3 carrier adds that.
func ConsensusRepairsV3GenesisSelections() []ActivationSelection {
	return []ActivationSelection{
		{zeroneknowledgetypes.ModuleName, "doctrine_identity_enabled"},
		{substratebridgetypes.ModuleName, "cited_reward_escrow_enabled"},
		{zeroneknowledgetypes.ModuleName, "dormant_fact_contest_enabled"},
		{zeroneknowledgetypes.ModuleName, "reviewer_balance_param_enabled"},
		{zeroneknowledgetypes.ModuleName, "seat_deposit_enabled"},
	}
}
