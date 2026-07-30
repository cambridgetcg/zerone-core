package main

import "fmt"

const shadowScenarioSchema = "zerone.constructive-capacity-shadow/v1"

// ShadowScenarioStep exposes the exact aggregate partition after one accepted
// event. It is a local explanatory trace, not a settlement instruction.
type ShadowScenarioStep struct {
	Event    string               `json:"event"`
	Snapshot ShadowLedgerSnapshot `json:"snapshot"`
}

// ShadowScenarioChecks keeps the claims made by the fixed counterexample
// machine-readable and individually inspectable.
type ShadowScenarioChecks struct {
	ExactPartition                 bool `json:"exact_partition"`
	AccruedUnchangedByReattribute  bool `json:"accrued_unchanged_by_reattribute"`
	FundedUnchangedByReattribute   bool `json:"funded_unchanged_by_reattribute"`
	ExtinguishedUnchangedByMove    bool `json:"extinguished_unchanged_by_reattribute"`
	ReplacementExposureWithinCap   bool `json:"replacement_exposure_within_cap"`
	ReplacementInheritedDeadline   bool `json:"replacement_inherited_deadline"`
	ReplacementGenerationAtMostOne bool `json:"replacement_generation_at_most_one"`
	Passed                         bool `json:"passed"`
}

// ShadowScenarioReport is deliberately zero-value and non-authoritative. The
// fixture demonstrates accounting behavior only; it does not authenticate
// receipts, observe a network, qualify work, or move funds.
type ShadowScenarioReport struct {
	Schema            string               `json:"schema"`
	ArithmeticVersion string               `json:"arithmetic_version"`
	Unit              string               `json:"unit"`
	Authoritative     bool                 `json:"authoritative"`
	NetworkObserved   bool                 `json:"network_observed"`
	RewardBearing     bool                 `json:"reward_bearing"`
	TransferableValue bool                 `json:"transferable_value"`
	MovesFunds        bool                 `json:"moves_funds"`
	SettlementZRN     uint64               `json:"settlement_zrn"`
	IntegrationReady  bool                 `json:"integration_ready"`
	Trace             []ShadowScenarioStep `json:"trace"`
	Checks            ShadowScenarioChecks `json:"checks"`
}

func snapshotPartitionsExactly(snapshot ShadowLedgerSnapshot) bool {
	return snapshot.Accrued ==
		snapshot.Funded+
			snapshot.Live+
			snapshot.Quarantined+
			snapshot.Extinguished
}

// RunShadowScenario executes the canonical cap-poisoning counterexample:
// A=K=100, fund 30, quarantine at most 60, extinguish 10, then reattribute
// exactly 50 against clean support S=80.
func RunShadowScenario() (ShadowScenarioReport, error) {
	ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                 "fixture:semantic-root-alpha",
		PolicyDigest:           "fixture:shadow-policy-v1",
		LifetimeCap:            100,
		LifetimeReplacementCap: 60,
	})
	if err != nil {
		return ShadowScenarioReport{}, err
	}
	trace := make([]ShadowScenarioStep, 0, 4)
	appendStep := func(event string) {
		trace = append(trace, ShadowScenarioStep{
			Event:    event,
			Snapshot: ledger.Snapshot(),
		})
	}

	if err := ledger.Accrue("fixture-accrue", 1, ShadowAccrual{
		LotID:                  "fixture:lot-alpha",
		ReceiptDigest:          "fixture:receipt-original",
		Amount:                 100,
		Deadline:               10,
		BeneficiaryControllers: []string{"fixture:originator"},
		DependencyControllers:  []string{"fixture:dependency"},
		EvaluatorControllers:   []string{"fixture:evaluator-original"},
	}); err != nil {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture accrue: %w", err)
	}
	appendStep("accrue")

	if funded, err := ledger.Fund("fixture-fund", 2, 30); err != nil {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture fund: %w", err)
	} else if funded != 30 {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture funded %d, want 30", funded)
	}
	appendStep("fund")

	if err := ledger.FinalizeInvalidation(
		"fixture-invalidate",
		3,
		ShadowInvalidation{
			DecisionID:             "fixture:decision-original",
			TargetReceiptDigest:    "fixture:receipt-original",
			CleanSupportTarget:     0,
			CulpableControllers:    []string{"fixture:originator"},
			ChallengerControllers:  []string{"fixture:challenger"},
			AdjudicatorControllers: []string{"fixture:adjudicator-a", "fixture:adjudicator-b", "fixture:adjudicator-c"},
			OrganizationRoots:      []string{"fixture:organization-a", "fixture:organization-b"},
			Final:                  true,
			AssignmentAfterFreeze:  true,
			OutcomeIndependentPay:  true,
		},
	); err != nil {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture invalidation: %w", err)
	}
	appendStep("final-invalidation")

	beforeReplacement := ledger.Snapshot()
	moved, err := ledger.Reattribute(
		"fixture-reattribute",
		4,
		ShadowSuccessor{
			ReceiptDigest:          "fixture:receipt-successor",
			PriorReceiptDigest:     "fixture:receipt-original",
			SupportTarget:          80,
			BeneficiaryControllers: []string{"fixture:successor"},
			DependencyControllers:  []string{"fixture:clean-dependency"},
			EvaluatorControllers:   []string{"fixture:clean-evaluator-a", "fixture:clean-evaluator-b", "fixture:clean-evaluator-c"},
			PolicyPassed:           true,
		},
	)
	if err != nil {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture reattribute: %w", err)
	}
	if moved != 50 {
		return ShadowScenarioReport{}, fmt.Errorf("shadow fixture moved %d, want 50", moved)
	}
	appendStep("reattribute")

	afterReplacement := ledger.Snapshot()
	checks := ShadowScenarioChecks{
		ExactPartition: true,
		AccruedUnchangedByReattribute: beforeReplacement.Accrued ==
			afterReplacement.Accrued,
		FundedUnchangedByReattribute: beforeReplacement.Funded ==
			afterReplacement.Funded,
		ExtinguishedUnchangedByMove: beforeReplacement.Extinguished ==
			afterReplacement.Extinguished,
		ReplacementExposureWithinCap: afterReplacement.ReplacementUsed+
			afterReplacement.Quarantined <=
			afterReplacement.Config.LifetimeReplacementCap,
		ReplacementInheritedDeadline:   afterReplacement.Lots[0].Deadline == 10,
		ReplacementGenerationAtMostOne: afterReplacement.Lots[0].ReplacementGeneration <= 1,
	}
	for _, step := range trace {
		checks.ExactPartition = checks.ExactPartition &&
			snapshotPartitionsExactly(step.Snapshot)
	}
	checks.Passed = checks.ExactPartition &&
		checks.AccruedUnchangedByReattribute &&
		checks.FundedUnchangedByReattribute &&
		checks.ExtinguishedUnchangedByMove &&
		checks.ReplacementExposureWithinCap &&
		checks.ReplacementInheritedDeadline &&
		checks.ReplacementGenerationAtMostOne

	return ShadowScenarioReport{
		Schema:            shadowScenarioSchema,
		ArithmeticVersion: shadowLedgerArithmeticVersion,
		Unit:              "model-unit",
		Authoritative:     false,
		NetworkObserved:   false,
		RewardBearing:     false,
		TransferableValue: false,
		MovesFunds:        false,
		SettlementZRN:     0,
		IntegrationReady:  false,
		Trace:             trace,
		Checks:            checks,
	}, nil
}
