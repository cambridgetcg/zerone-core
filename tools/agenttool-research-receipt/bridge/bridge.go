package bridge

import "fmt"

// Evaluate independently validates one AgentTool settlement, its minimized
// public projection, and the exact reviewed static Tree bytes. It performs no
// I/O itself and grants no live status or effect.
func Evaluate(settlementBytes, projectionBytes, treeBytes []byte) (Receipt, error) {
	settlement, err := parseSettlement(settlementBytes)
	if err != nil {
		return Receipt{}, err
	}
	projection, err := parseProjection(projectionBytes)
	if err != nil {
		return Receipt{}, err
	}
	tree, err := parseTree(treeBytes)
	if err != nil {
		return Receipt{}, err
	}
	if err := crossCheck(settlement, projection, tree); err != nil {
		return Receipt{}, err
	}

	consumptionKey := tupleID(
		"zerone.agenttool-research-consumption/v0",
		settlement.SettlementID,
	)
	receiptID := tupleID(
		ReceiptSchema,
		AdapterVersion,
		InteropProfileDigest,
		settlement.SettlementID,
		projection.ProjectionID,
		tree.DocumentDigest,
		tree.NodeDigest,
	)
	receipt := Receipt{
		Schema:         ReceiptSchema,
		AdapterVersion: AdapterVersion,
		Assurance:      Assurance,
		Status:         StatusCandidate,
		ReceiptID:      receiptID,
		Source: SourceBinding{
			SettlementFormat: SettlementFormat,
			SettlementID:     settlement.SettlementID,
			ProjectionFormat: PublicProjectionFormat,
			ProjectionID:     projection.ProjectionID,
			ConsumptionKey:   consumptionKey,
			ConsumptionState: ConsumptionState,
			ReplayProtection: ReplayProtection,
		},
		Interop: InteropBinding{
			Format:            InteropProfileFormat,
			RawDigest:         InteropProfileDigest,
			IntegrationStatus: InteropProfileStatus,
			Imported:          false,
			Activated:         false,
		},
		Tree: TreeBinding{
			Schema:                    TreeSchema,
			DocumentDigest:            tree.DocumentDigest,
			NodeID:                    TargetNodeID,
			NodeDigest:                tree.NodeDigest,
			NetworkObserved:           false,
			RewardBearing:             false,
			GrantedAttainmentEvidence: EffectNone,
		},
		Declaration: ResultDeclaration{
			CaseID:               settlement.Settlement.CaseID,
			DeclaredResultKind:   settlement.Settlement.DeclaredResultKind,
			HighestEvidenceLevel: projection.Projection.HighestEvidenceLevel,
			ResultAuthority:      ResultAuthority,
		},
		Simulation: SimulationBinding{
			PaymentCondition: PaymentCondition,
			CreditAmount:     settlement.Settlement.SimulatedCredit.Amount,
			CreditUnit:       SimulatedUnit,
			Convertible:      false,
			Transferable:     false,
			WalletBearing:    false,
		},
		LedgerBoundary: LedgerBinding{
			ProfileID:             LedgerProfileID,
			ProfileDigest:         LedgerProfileHash,
			SharedUnit:            false,
			CrossLedgerArithmetic: false,
			CrossLedgerConversion: false,
			CrossLedgerInference:  false,
		},
		CrossLedgerRelation: NoEquivalence,
		KnowledgeAdmission:  EffectNone,
		Qualification:       EffectNone,
		EconomicEffect:      EffectNone,
		AmountUzrn:          "0",
		Effects:             noZeroneEffects(),
		Limitations: []string{
			"STRUCTURAL_CANDIDATE means only that exact local bytes passed the fixed offline predicate",
			"the declared result kind is not a scientific verdict and is carried separately from the declared simulated credit amount",
			"this compiler does not verify schedule precommitment, compliant delivery, reviewer neutrality, prefunding, conservation, AgentTool provenance, or result-independent amount selection",
			"AgentTool append-only challenge and work retention is checked only relative to one caller-supplied prior_state transition",
			"content-addressed state IDs are not signatures or canonical heads and prove no provenance, trusted time, global ordering, or prevention of old-state forks",
			"simulated credits are non-transferable, non-wallet-bearing, unbacked by this adapter, and have no ZRN or external-value equivalence",
			"the Tree node is a static non-person coordinate and receives no account, balance, agency, attainment, qualification, or reward",
			"content addressing authenticates neither agent identity nor effective-controller independence, consent, authorship, or truth",
			"the deterministic consumption key is not replay protection and no shared receipt-consumption ledger was consulted",
			"the pinned interop profile binds static vocabulary bytes only and activates or imports nothing",
			"the compiler performs no AgentTool call, ToK query, RPC, chain read, chain write, bridge invocation, or economic action",
		},
	}
	if err := validateOutput(receipt); err != nil {
		return Receipt{}, fmt.Errorf("internal receipt invariant: %w", err)
	}
	return receipt, nil
}

func crossCheck(settlement settlementEnvelope, projection projectionEnvelope, tree parsedTree) error {
	if settlement.Settlement.CaseID != projection.Projection.CaseID {
		return fmt.Errorf("settlement and public projection case_id differ")
	}
	if len(projection.Projection.SettlementBundleIDs) != 1 ||
		projection.Projection.SettlementBundleIDs[0] != settlement.SettlementID {
		return fmt.Errorf("public projection must reference exactly the supplied settlement_id")
	}
	if projection.Projection.NodeRef.TreeRawSHA256 != tree.DocumentDigest ||
		projection.Projection.NodeRef.NodeDigest != tree.NodeDigest {
		return fmt.Errorf("public projection Tree pins do not match supplied Tree bytes")
	}
	if len(projection.Projection.PublicEvidenceReceiptIDs) != len(settlement.Settlement.ConsumedReceiptIDs) {
		return fmt.Errorf("public evidence receipt ids must exactly equal the supplied settlement's consumed receipt ids")
	}
	for index, id := range settlement.Settlement.ConsumedReceiptIDs {
		if projection.Projection.PublicEvidenceReceiptIDs[index] != id {
			return fmt.Errorf("public evidence receipt ids must exactly equal the supplied settlement's consumed receipt ids")
		}
	}
	if projection.Projection.HighestEvidenceLevel == nil {
		return fmt.Errorf("settled pilot projection must declare a highest evidence level")
	}
	return nil
}

func validateOutput(receipt Receipt) error {
	if receipt.Schema != ReceiptSchema ||
		receipt.AdapterVersion != AdapterVersion ||
		receipt.Assurance != Assurance ||
		receipt.Status != StatusCandidate {
		return fmt.Errorf("schema, adapter, assurance, or status drift")
	}
	if receipt.CrossLedgerRelation != NoEquivalence ||
		receipt.KnowledgeAdmission != EffectNone ||
		receipt.Qualification != EffectNone ||
		receipt.EconomicEffect != EffectNone ||
		receipt.AmountUzrn != "0" ||
		receipt.Tree.GrantedAttainmentEvidence != EffectNone {
		return fmt.Errorf("zero-value or no-equivalence boundary crossed")
	}
	if receipt.Interop.Format != InteropProfileFormat ||
		receipt.Interop.RawDigest != InteropProfileDigest ||
		receipt.Interop.IntegrationStatus != InteropProfileStatus ||
		receipt.Interop.Imported ||
		receipt.Interop.Activated {
		return fmt.Errorf("static interop profile boundary drift")
	}
	if receipt.LedgerBoundary.ProfileID != LedgerProfileID ||
		receipt.LedgerBoundary.ProfileDigest != LedgerProfileHash ||
		receipt.LedgerBoundary.SharedUnit ||
		receipt.LedgerBoundary.CrossLedgerArithmetic ||
		receipt.LedgerBoundary.CrossLedgerConversion ||
		receipt.LedgerBoundary.CrossLedgerInference {
		return fmt.Errorf("six-ledger boundary drift")
	}
	if receipt.Source.ConsumptionState != ConsumptionState ||
		receipt.Source.ReplayProtection != ReplayProtection {
		return fmt.Errorf("offline output claims shared consumption or replay state")
	}
	if receipt.Effects != (ZeroneEffects{}) {
		return fmt.Errorf("one or more Zerone effects became true")
	}
	if receipt.Simulation.Convertible || receipt.Simulation.Transferable || receipt.Simulation.WalletBearing {
		return fmt.Errorf("simulated credit boundary crossed")
	}
	if len(receipt.Limitations) == 0 {
		return fmt.Errorf("limitations must remain explicit")
	}
	return nil
}
