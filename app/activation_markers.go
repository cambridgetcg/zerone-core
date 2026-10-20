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

func ConsensusRepairsV1GenesisSelections() []ActivationSelection {
	return []ActivationSelection{
		{zeronegovtypes.ModuleName, "seat_runoff_ordering_enabled"},
		{zeroneknowledgetypes.ModuleName, "reformulation_vindication_ordering_enabled"},
		{zeronecdtypes.ModuleName, "integer_variance_enabled"},
		{zeronequalificationtypes.ModuleName, "single_refund_enabled"},
		{zeronecctypes.ModuleName, "challenge_accounting_enabled"},
		{vestingrewardstypes.ModuleName, "alignment_bank_supply_enabled"},
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
		{vestingrewardstypes.ModuleName, "alignment_bank_supply_enabled",
			app.VestingRewardsKeeper.AlignmentBankSupplyEnabled, app.VestingRewardsKeeper.EnableAlignmentBankSupply},
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
// Native genesis selects each one through its module's genesis field; the
// consensus-repairs-v2 carrier will enable each one by looping over this list.
// It shares no module/field pair or store key with ConsensusRepairsV1Markers.
// The genesis-only FX7 and FX8 schemas change export and import, never a live
// state transition, so they have no marker here.
func (app *ZeroneApp) ConsensusRepairsV2Markers() []ActivationMarker {
	return []ActivationMarker{
		{zeronealignmenttypes.ModuleName, "unobserved_health_enabled",
			app.AlignmentKeeper.UnobservedHealthEnabled, app.AlignmentKeeper.EnableUnobservedHealth},
		{zeroneknowledgetypes.ModuleName, "inconclusive_round_accounting_enabled",
			app.KnowledgeKeeper.InconclusiveRoundAccountingEnabled, app.KnowledgeKeeper.EnableInconclusiveRoundAccounting},
		{zeroneknowledgetypes.ModuleName, "calibration_smoothing_enabled",
			app.KnowledgeKeeper.CalibrationSmoothingEnabled, app.KnowledgeKeeper.EnableCalibrationSmoothing},
		{zeroneknowledgetypes.ModuleName, "zero_confidence_floor_enabled",
			app.KnowledgeKeeper.ZeroConfidenceFloorEnabled, app.KnowledgeKeeper.EnableZeroConfidenceFloor},
	}
}
