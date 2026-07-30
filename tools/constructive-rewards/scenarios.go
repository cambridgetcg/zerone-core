package main

import (
	"fmt"
	"math"
)

// AttackReport compares address/artifact-count mechanisms with the proposed
// controller- and preclustered mechanism boundaries.
type AttackReport struct {
	WhaleSingleAddressShare       float64 `json:"whale_single_address_share"`
	WhaleHundredAliasShare        float64 `json:"whale_hundred_alias_share"`
	WhaleAliasGain                float64 `json:"whale_alias_gain"`
	NaiveOneArtifactShare         float64 `json:"naive_one_artifact_share"`
	NaiveHundredArtifactShare     float64 `json:"naive_hundred_artifact_share"`
	PreclusteredOneArtifactFunded float64 `json:"preclustered_one_artifact_funded"`
	PreclusteredHundredFunded     float64 `json:"preclustered_hundred_artifact_funded"`
	PreclusteredOneArtifactDirect float64 `json:"preclustered_one_artifact_direct"`
	PreclusteredHundredDirect     float64 `json:"preclustered_hundred_artifact_direct"`
	PreclusteredOneCommons        float64 `json:"preclustered_one_artifact_commons"`
	PreclusteredHundredCommons    float64 `json:"preclustered_hundred_artifact_commons"`
	MonocultureEffectiveCount     float64 `json:"monoculture_effective_count"`
	HundredAliasEffectiveCount    float64 `json:"hundred_alias_effective_count"`
	MixedPanelSingleAliasCount    float64 `json:"mixed_panel_single_alias_count"`
	MixedPanelHundredAliasCount   float64 `json:"mixed_panel_hundred_alias_count"`
	MixedPanelSingleAliasScore    float64 `json:"mixed_panel_single_alias_score"`
	MixedPanelHundredAliasScore   float64 `json:"mixed_panel_hundred_alias_score"`
	TwentyIndependentCount        float64 `json:"twenty_independent_count"`
	ObservedAddressEffectivePower float64 `json:"observed_address_effective_power"`
	ControllerEffectivePower      float64 `json:"controller_effective_power"`
}

type preclusteredOutcome struct {
	Funded  float64
	Direct  float64
	Commons float64
}

// SweepRow records scarcity and cluster-lifetime controller-cap behavior.
type SweepRow struct {
	Alpha                       float64 `json:"alpha"`
	ControllerCapShare          float64 `json:"controller_cap_share"`
	LargestAggregateDirectShare float64 `json:"largest_aggregate_direct_share"`
	MaxClusterCapUtilization    float64 `json:"max_cluster_cap_utilization"`
	NewcomerDirectShare         float64 `json:"newcomer_direct_share"`
	CommonsShare                float64 `json:"commons_share"`
	UnfundedDemand              float64 `json:"unfunded_demand"`
	BudgetError                 float64 `json:"budget_error"`
	Passed                      bool    `json:"passed"`
}

// GateResult distinguishes model invariants from absent chain integrations.
type GateResult struct {
	Class  string `json:"class"`
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// SimulationReport is stable JSON output from the deterministic tool.
type SimulationReport struct {
	Parameters            Params          `json:"parameters"`
	IllustrativeEpoch     EpochResult     `json:"illustrative_epoch"`
	Attacks               AttackReport    `json:"attacks"`
	PowerSurfaces         []SurfaceMetric `json:"current_power_surfaces"`
	WeakestEffectivePower float64         `json:"current_weakest_effective_power"`
	Sweep                 []SweepRow      `json:"sweep"`
	ReleaseGates          []GateResult    `json:"release_gates"`
	ModelChecksPassed     bool            `json:"model_checks_passed"`
	IntegrationReady      bool            `json:"integration_ready"`
}

func signals(prefix string, count int) []Signal {
	result := make([]Signal, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, Signal{
			Controller: fmt.Sprintf("%s-controller-%03d", prefix, i),
			Family:     fmt.Sprintf("%s-family-%03d", prefix, i),
		})
	}
	return result
}

func uniformCorrelation(count int, rho float64) [][]float64 {
	matrix := make([][]float64, count)
	for i := range matrix {
		matrix[i] = make([]float64, count)
		for j := range matrix[i] {
			if i == j {
				matrix[i][j] = 1
			} else {
				matrix[i][j] = rho
			}
		}
	}
	return matrix
}

func capturedPowerSurfaces() map[string]map[string]float64 {
	result := balancedPowerSurfaces("captured-policy")
	result["epistemic-review"] = map[string]float64{
		"review-a": 1,
		"review-b": 1,
		"review-c": 1,
		"review-d": 1,
	}
	result["stake-governance"] = map[string]float64{
		"civic-a": 0.55,
		"civic-b": 0.20,
		"civic-c": 0.15,
		"civic-d": 0.10,
	}
	result["infrastructure"] = map[string]float64{
		"operator-a": 0.80,
		"operator-b": 0.10,
		"operator-c": 0.10,
	}
	result["reward-flow"] = map[string]float64{
		"originator":     0.28,
		"formalizer":     0.22,
		"replicator":     0.20,
		"challenger":     0.16,
		"bridge-builder": 0.14,
	}
	return result
}

func baseEvidence(marginal, use float64, replications []Signal) Evidence {
	return Evidence{
		Provenance:   0.99,
		Validity:     0.99,
		Marginal:     marginal,
		Use:          use,
		Safety:       0.97,
		Replications: replications,
	}
}

func illustrativeClusters() []Cluster {
	return []Cluster{
		{
			ID:          "formal-result-alpha",
			ArtifactIDs: []string{"proof", "exposition", "kernel-build"},
			Evidence:    baseEvidence(0.94, 0.72, signals("alpha", 4)),
			LifetimeCap: 1_200,
			Credits: []ControllerCredit{
				{Controller: "originator", Credit: 0.42, Roles: []string{"originator"}},
				{Controller: "formalizer", Credit: 0.23, Roles: []string{"formalizer"}},
				{Controller: "replicator", Credit: 0.20, Roles: []string{"replicator"}},
				{Controller: "challenger", Credit: 0.15, Roles: []string{"challenger"}},
			},
		},
		{
			ID:          "negative-result-beta",
			ArtifactIDs: []string{"counterexample", "repair"},
			Evidence:    baseEvidence(0.78, 0.61, signals("beta", 3)),
			LifetimeCap: 900,
			Credits: []ControllerCredit{
				{Controller: "challenger", Credit: 0.45, Roles: []string{"counterexample"}},
				{Controller: "repairer", Credit: 0.35, Roles: []string{"repair"}},
				{Controller: "replicator", Credit: 0.20, Roles: []string{"replicator"}},
			},
		},
		{
			ID:          "portable-method-gamma",
			ArtifactIDs: []string{"method", "independent-port"},
			Evidence:    baseEvidence(0.69, 0.82, signals("gamma", 5)),
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "bridge-builder", Credit: 0.50, Roles: []string{"bridge-builder"}},
				{Controller: "outsider-lab", Credit: 0.30, Roles: []string{"independent-port"}},
				{Controller: "steward", Credit: 0.20, Roles: []string{"steward"}},
			},
		},
	}
}

func runRegisteredEpoch(params Params, epochID string, clusters []Cluster) (EpochResult, error) {
	engine, err := NewEngine(params)
	if err != nil {
		return EpochResult{}, err
	}
	proposals := make([]ClusterProposal, 0, len(clusters))
	for _, cluster := range clusters {
		if err := engine.RegisterCluster(cluster.ID, cluster.LifetimeCap, cluster.Credits); err != nil {
			return EpochResult{}, err
		}
		proposals = append(proposals, ClusterProposal{
			EventID:     epochID + "/" + cluster.ID,
			ClusterID:   cluster.ID,
			ArtifactIDs: cluster.ArtifactIDs,
			Evidence:    cluster.Evidence,
		})
	}
	return engine.RunEpoch(epochID, proposals)
}

func preclusteredArtifactScenario(artifactCount int, p Params) (preclusteredOutcome, error) {
	attackArtifacts := make([]string, artifactCount)
	for i := range attackArtifacts {
		attackArtifacts[i] = fmt.Sprintf("attacker-shard-%04d", i)
	}
	local := p
	local.Budget = 2_000
	evidence := baseEvidence(0.82, 0.75, signals("salami", 4))
	result, err := runRegisteredEpoch(local, "preclustered-epoch", []Cluster{
		{
			ID:          "attacker-equivalence-cluster",
			ArtifactIDs: attackArtifacts,
			Evidence:    evidence,
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "attacker-controller", Credit: 1},
			},
		},
		{
			ID:          "honest-independent-result",
			ArtifactIDs: []string{"honest-artifact"},
			Evidence:    evidence,
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "honest-controller", Credit: 1},
			},
		},
	})
	if err != nil {
		return preclusteredOutcome{}, err
	}
	for _, cluster := range result.Clusters {
		if cluster.ID == "attacker-equivalence-cluster" {
			return preclusteredOutcome{
				Funded:  cluster.FundedBudget,
				Direct:  result.ControllerDirect["attacker-controller"],
				Commons: sumAllocationOverflow(cluster.Allocations),
			}, nil
		}
	}
	return preclusteredOutcome{}, fmt.Errorf("attacker equivalence cluster missing from result")
}

func runAttacks(p Params) (AttackReport, error) {
	singleWhale, err := NaiveSqrtStakeShare(40, 60, 1)
	if err != nil {
		return AttackReport{}, err
	}
	splitWhale, err := NaiveSqrtStakeShare(40, 60, 100)
	if err != nil {
		return AttackReport{}, err
	}
	naiveOne, err := NaiveArtifactShare(1, 1)
	if err != nil {
		return AttackReport{}, err
	}
	naiveHundred, err := NaiveArtifactShare(100, 1)
	if err != nil {
		return AttackReport{}, err
	}
	preclusteredOne, err := preclusteredArtifactScenario(1, p)
	if err != nil {
		return AttackReport{}, err
	}
	preclusteredHundred, err := preclusteredArtifactScenario(100, p)
	if err != nil {
		return AttackReport{}, err
	}

	monoculture, err := EffectiveIndependentCount(
		signals("mono", 100),
		uniformCorrelation(100, 0.20),
		0,
	)
	if err != nil {
		return AttackReport{}, err
	}
	aliases := make([]Signal, 100)
	for i := range aliases {
		aliases[i] = Signal{
			Controller: "one-hidden-controller",
			Family:     "one-hidden-family",
		}
	}
	aliasCount, err := EffectiveIndependentCount(aliases, nil, 0)
	if err != nil {
		return AttackReport{}, err
	}
	singleMixed := []Signal{
		{Controller: "controller-a", Family: "family-a"},
		{Controller: "controller-b", Family: "family-b"},
	}
	singleMixedCount, err := EffectiveIndependentCount(singleMixed, nil, 0)
	if err != nil {
		return AttackReport{}, err
	}
	hundredMixed := append([]Signal(nil), aliases...)
	for i := range hundredMixed {
		hundredMixed[i].Controller = "controller-a"
	}
	hundredMixed = append(hundredMixed, Signal{
		Controller: "controller-b",
		Family:     "independent-b",
	})
	hundredMixedCount, err := EffectiveIndependentCount(hundredMixed, nil, 0)
	if err != nil {
		return AttackReport{}, err
	}
	singleMixedScore, err := ScoreEvidence(
		baseEvidence(0.8, 0.8, singleMixed),
		p,
	)
	if err != nil {
		return AttackReport{}, err
	}
	hundredMixedScore, err := ScoreEvidence(
		baseEvidence(0.8, 0.8, hundredMixed),
		p,
	)
	if err != nil {
		return AttackReport{}, err
	}
	independentCount, err := EffectiveIndependentCount(signals("independent", 20), nil, 0)
	if err != nil {
		return AttackReport{}, err
	}

	addressPowers := make(map[string]float64)
	for i := 0; i < 60; i++ {
		addressPowers[fmt.Sprintf("cartel-alias-%02d", i)] = 1
	}
	for i := 0; i < 4; i++ {
		addressPowers[fmt.Sprintf("honest-controller-%d", i)] = 10
	}
	observed, err := MeasurePower("address-observed", addressPowers, 0.34)
	if err != nil {
		return AttackReport{}, err
	}
	controller, err := MeasurePower("controller-truth", map[string]float64{
		"cartel-controller": 60,
		"honest-a":          10,
		"honest-b":          10,
		"honest-c":          10,
		"honest-d":          10,
	}, 0.34)
	if err != nil {
		return AttackReport{}, err
	}

	return AttackReport{
		WhaleSingleAddressShare:       singleWhale,
		WhaleHundredAliasShare:        splitWhale,
		WhaleAliasGain:                splitWhale / singleWhale,
		NaiveOneArtifactShare:         naiveOne,
		NaiveHundredArtifactShare:     naiveHundred,
		PreclusteredOneArtifactFunded: preclusteredOne.Funded,
		PreclusteredHundredFunded:     preclusteredHundred.Funded,
		PreclusteredOneArtifactDirect: preclusteredOne.Direct,
		PreclusteredHundredDirect:     preclusteredHundred.Direct,
		PreclusteredOneCommons:        preclusteredOne.Commons,
		PreclusteredHundredCommons:    preclusteredHundred.Commons,
		MonocultureEffectiveCount:     monoculture,
		HundredAliasEffectiveCount:    aliasCount,
		MixedPanelSingleAliasCount:    singleMixedCount,
		MixedPanelHundredAliasCount:   hundredMixedCount,
		MixedPanelSingleAliasScore:    singleMixedScore.Total,
		MixedPanelHundredAliasScore:   hundredMixedScore.Total,
		TwentyIndependentCount:        independentCount,
		ObservedAddressEffectivePower: observed.EffectiveCount,
		ControllerEffectivePower:      controller.EffectiveCount,
	}, nil
}

func sweepClusters() []Cluster {
	return []Cluster{
		{
			ID:          "incumbent-a",
			Evidence:    baseEvidence(0.95, 0.90, signals("incumbent-a", 5)),
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "incumbent", Credit: 1},
			},
		},
		{
			ID:          "incumbent-b",
			Evidence:    baseEvidence(0.88, 0.84, signals("incumbent-b", 4)),
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "incumbent", Credit: 1},
			},
		},
		{
			ID:          "newcomer",
			Evidence:    baseEvidence(0.74, 0.68, signals("newcomer", 3)),
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "newcomer", Credit: 1},
			},
		},
		{
			ID:          "outsider",
			Evidence:    baseEvidence(0.62, 0.57, signals("outsider", 2)),
			LifetimeCap: 1_000,
			Credits: []ControllerCredit{
				{Controller: "outsider", Credit: 1},
			},
		},
	}
}

func runSweep(base Params) ([]SweepRow, error) {
	alphas := []float64{0.50, 0.65, 0.80, 1.00}
	caps := []float64{0.15, 0.20, 0.25, 0.33}
	rows := make([]SweepRow, 0, len(alphas)*len(caps))
	for _, alpha := range alphas {
		for _, capShare := range caps {
			p := base
			p.Alpha = alpha
			p.ControllerCapShare = capShare
			result, err := runRegisteredEpoch(p, "sweep-epoch", sweepClusters())
			if err != nil {
				return nil, err
			}
			var largest, maxUtilization float64
			for _, direct := range result.ControllerDirect {
				largest = math.Max(largest, direct/p.Budget)
			}
			for _, cluster := range result.Clusters {
				lifetimeCap := result.States[cluster.ID].LifetimeCap
				controllerCap := p.ControllerCapShare * lifetimeCap
				for _, allocation := range cluster.Allocations {
					if controllerCap > 0 {
						maxUtilization = math.Max(
							maxUtilization,
							allocation.CumulativeDirectTarget/controllerCap,
						)
					}
				}
			}
			accounted := result.DirectTotal + result.CommonsTotal + result.Unallocated
			errorAmount := math.Abs(accounted - p.Budget)
			rows = append(rows, SweepRow{
				Alpha:                       alpha,
				ControllerCapShare:          capShare,
				LargestAggregateDirectShare: largest,
				MaxClusterCapUtilization:    maxUtilization,
				NewcomerDirectShare:         result.ControllerDirect["newcomer"] / p.Budget,
				CommonsShare:                result.CommonsTotal / p.Budget,
				UnfundedDemand:              result.UnfundedDemand,
				BudgetError:                 errorAmount,
				Passed: maxUtilization <= 1+floatTolerance &&
					amountsEqual(accounted, p.Budget) &&
					result.ControllerDirect["newcomer"] > 0,
			})
		}
	}
	return rows, nil
}

type temporalComparison struct {
	Passed         bool
	JumpGross      float64
	SteppedGross   float64
	JumpDirect     float64
	SteppedDirect  float64
	JumpCommons    float64
	SteppedCommons float64
}

func temporalPathComparison(p Params, budget float64) (temporalComparison, error) {
	local := p
	local.Budget = budget
	credits := []ControllerCredit{{Controller: "temporal-controller", Credit: 1}}

	jumpEngine, err := NewEngine(local)
	if err != nil {
		return temporalComparison{}, err
	}
	if err := jumpEngine.RegisterCluster("temporal-cluster", 1_000, credits); err != nil {
		return temporalComparison{}, err
	}
	var jumpGross, jumpDirect, jumpCommons float64
	for i := 0; i < 3; i++ {
		jump, err := jumpEngine.RunEpoch(
			fmt.Sprintf("jump-%d", i),
			[]ClusterProposal{{
				EventID:   fmt.Sprintf("jump-event-%d", i),
				ClusterID: "temporal-cluster",
				Evidence:  baseEvidence(0.90, 0.70, signals("temporal", 3)),
			}},
		)
		if err != nil {
			return temporalComparison{}, err
		}
		jumpGross += jump.Clusters[0].GrossEntitlement
		jumpDirect += jump.DirectTotal
		jumpCommons += jump.CommonsTotal
	}

	stepEngine, err := NewEngine(local)
	if err != nil {
		return temporalComparison{}, err
	}
	if err := stepEngine.RegisterCluster("temporal-cluster", 1_000, credits); err != nil {
		return temporalComparison{}, err
	}
	var steppedGross, steppedDirect, steppedCommons float64
	for i, marginal := range []float64{0.30, 0.60, 0.90} {
		step, err := stepEngine.RunEpoch(
			fmt.Sprintf("step-%d", i),
			[]ClusterProposal{{
				EventID:   fmt.Sprintf("step-event-%d", i),
				ClusterID: "temporal-cluster",
				Evidence:  baseEvidence(marginal, 0.70, signals("temporal", 3)),
			}},
		)
		if err != nil {
			return temporalComparison{}, err
		}
		steppedGross += step.Clusters[0].GrossEntitlement
		steppedDirect += step.DirectTotal
		steppedCommons += step.CommonsTotal
	}
	comparison := temporalComparison{
		JumpGross:      jumpGross,
		SteppedGross:   steppedGross,
		JumpDirect:     jumpDirect,
		SteppedDirect:  steppedDirect,
		JumpCommons:    jumpCommons,
		SteppedCommons: steppedCommons,
	}
	comparison.Passed =
		math.Abs(comparison.JumpGross-comparison.SteppedGross) <= floatTolerance &&
			amountsEqual(comparison.JumpDirect, comparison.SteppedDirect) &&
			amountsEqual(comparison.JumpCommons, comparison.SteppedCommons)
	return comparison, nil
}

type competingTemporalComparison struct {
	Passed       bool
	JumpGross    float64
	PacedGross   float64
	JumpFunded   float64
	PacedFunded  float64
	JumpBacklog  float64
	PacedBacklog float64
}

type temporalPathOutcome struct {
	Gross   float64
	Funded  float64
	Backlog float64
}

func runCompetingTemporalPath(
	p Params,
	prefix string,
	attackerMarginal []float64,
	competitorMarginal []float64,
) (temporalPathOutcome, error) {
	if len(attackerMarginal) == 0 || len(attackerMarginal) != len(competitorMarginal) {
		return temporalPathOutcome{}, fmt.Errorf("temporal paths must have the same positive length")
	}
	engine, err := NewEngine(p)
	if err != nil {
		return temporalPathOutcome{}, err
	}
	if err := engine.RegisterCluster(
		"attacker",
		1_000,
		[]ControllerCredit{{Controller: prefix + "-controller", Credit: 1}},
	); err != nil {
		return temporalPathOutcome{}, err
	}
	if err := engine.RegisterCluster(
		"competitor",
		1_000,
		[]ControllerCredit{{Controller: "competitor-controller", Credit: 1}},
	); err != nil {
		return temporalPathOutcome{}, err
	}
	outcome := temporalPathOutcome{}
	for epoch, point := range attackerMarginal {
		result, err := engine.RunEpoch(
			fmt.Sprintf("%s-epoch-%d", prefix, epoch),
			[]ClusterProposal{
				{
					EventID:   fmt.Sprintf("%s-attacker-%d", prefix, epoch),
					ClusterID: "attacker",
					Evidence:  baseEvidence(point, 0.70, signals(prefix+"-attacker", 3)),
				},
				{
					EventID:   fmt.Sprintf("%s-competitor-%d", prefix, epoch),
					ClusterID: "competitor",
					Evidence: baseEvidence(
						competitorMarginal[epoch],
						0.75,
						signals(prefix+"-competitor", 3),
					),
				},
			},
		)
		if err != nil {
			return temporalPathOutcome{}, err
		}
		for _, cluster := range result.Clusters {
			if cluster.ID == "attacker" {
				outcome.Gross += cluster.GrossEntitlement
				outcome.Funded += cluster.FundedBudget
			}
		}
	}
	state := engine.Snapshot().Clusters["attacker"].State
	outcome.Backlog = state.LifetimeCap*state.HighWater - state.FundedToDate
	return outcome, nil
}

func competingTemporalNoPacingAdvantage(p Params) (competingTemporalComparison, error) {
	local := p
	local.Budget = 100
	local.ControllerCapShare = 1
	competitor := []float64{0.85, 0.85}
	jump, err := runCompetingTemporalPath(
		local,
		"jump-competition",
		[]float64{0.90, 0.90},
		competitor,
	)
	if err != nil {
		return competingTemporalComparison{}, err
	}
	paced, err := runCompetingTemporalPath(
		local,
		"paced-competition",
		[]float64{0.30, 0.90},
		competitor,
	)
	if err != nil {
		return competingTemporalComparison{}, err
	}
	comparison := competingTemporalComparison{
		JumpGross:    jump.Gross,
		PacedGross:   paced.Gross,
		JumpFunded:   jump.Funded,
		PacedFunded:  paced.Funded,
		JumpBacklog:  jump.Backlog,
		PacedBacklog: paced.Backlog,
	}
	comparison.Passed =
		amountsEqual(comparison.JumpGross, comparison.PacedGross) &&
			comparison.PacedFunded <=
				comparison.JumpFunded+
					amountTolerance(comparison.PacedFunded, comparison.JumpFunded) &&
			comparison.PacedBacklog+
				amountTolerance(comparison.PacedBacklog, comparison.JumpBacklog) >=
				comparison.JumpBacklog &&
			!amountsEqual(comparison.JumpBacklog, comparison.PacedBacklog)
	return comparison, nil
}

func properPaymentCheck() (bool, error) {
	for _, truth := range []float64{0.20, 0.50, 0.90} {
		truthful := expectedReviewerPayment(truth, truth)
		for i := 0; i <= 100; i++ {
			report := float64(i) / 100
			candidate := expectedReviewerPayment(truth, report)
			if candidate > truthful+floatTolerance {
				return false, nil
			}
			if math.Abs(report-truth) > floatTolerance &&
				candidate >= truthful-floatTolerance {
				return false, nil
			}
		}
	}
	return true, nil
}

func expectedReviewerPayment(trueProbability, report float64) float64 {
	payOne, _ := ReviewerPayment(report, 1, 1, 2)
	payZero, _ := ReviewerPayment(report, 0, 1, 2)
	return trueProbability*payOne + (1-trueProbability)*payZero
}

func stateReplayCheck(p Params) (bool, error) {
	engine, err := NewEngine(p)
	if err != nil {
		return false, err
	}
	if err := engine.RegisterCluster(
		"replay-cluster",
		1_000,
		[]ControllerCredit{{Controller: "replay-controller", Credit: 1}},
	); err != nil {
		return false, err
	}
	proposal := ClusterProposal{
		EventID:   "replay-event",
		ClusterID: "replay-cluster",
		Evidence:  baseEvidence(0.8, 0.8, signals("replay", 3)),
	}
	if _, err := engine.RunEpoch("replay-epoch", []ClusterProposal{proposal}); err != nil {
		return false, err
	}
	restored, err := RestoreEngine(engine.Snapshot())
	if err != nil {
		return false, err
	}
	if _, err := engine.RunEpoch("different-epoch", []ClusterProposal{proposal}); err == nil {
		return false, nil
	}
	if _, err := restored.RunEpoch("restored-epoch", []ClusterProposal{proposal}); err == nil {
		return false, nil
	}
	if err := engine.RegisterCluster(
		"replay-cluster",
		9_999,
		[]ControllerCredit{{Controller: "attacker", Credit: 1}},
	); err == nil {
		return false, nil
	}
	return true, nil
}

func quorumAndPowerChecks(p Params) (bool, bool, error) {
	noReplication := baseEvidence(1, 1, nil)
	noReplicationScore, err := ScoreEvidence(noReplication, p)
	if err != nil {
		return false, false, err
	}
	captured := baseEvidence(1, 1, signals("captured", 4))
	capturedParams := cloneParams(p)
	capturedParams.PowerSurfaces = capturedPowerSurfaces()
	capturedScore, err := ScoreEvidence(captured, capturedParams)
	if err != nil {
		return false, false, err
	}
	missingSurfaceParams := cloneParams(p)
	delete(missingSurfaceParams.PowerSurfaces, requiredPowerSurfaceNames[0])
	missingSurfaceFails := missingSurfaceParams.Validate() != nil
	return !noReplicationScore.GatePassed && noReplicationScore.Total == 0,
		!capturedScore.GatePassed && capturedScore.Total == 0 && missingSurfaceFails,
		nil
}

func modelGates(p Params, epoch EpochResult, attacks AttackReport, sweep []SweepRow) ([]GateResult, error) {
	accounted := epoch.DirectTotal + epoch.CommonsTotal + epoch.Unallocated
	budgetPass := amountsEqual(accounted, epoch.Budget)
	abundantPath, err := temporalPathComparison(p, 10_000)
	if err != nil {
		return nil, err
	}
	scarcePath, err := temporalPathComparison(p, 100)
	if err != nil {
		return nil, err
	}
	competingPath, err := competingTemporalNoPacingAdvantage(p)
	if err != nil {
		return nil, err
	}
	properPass, err := properPaymentCheck()
	if err != nil {
		return nil, err
	}
	replayPass, err := stateReplayCheck(p)
	if err != nil {
		return nil, err
	}
	quorumPass, decentralizationPass, err := quorumAndPowerChecks(p)
	if err != nil {
		return nil, err
	}
	safetyAtThreshold := baseEvidence(0.8, 0.8, signals("safety-gate", 4))
	safetyAtThreshold.Safety = p.MinSafety
	thresholdSafetyScore, err := ScoreEvidence(safetyAtThreshold, p)
	if err != nil {
		return nil, err
	}
	safetyAboveThreshold := safetyAtThreshold
	safetyAboveThreshold.Safety = 1
	aboveSafetyScore, err := ScoreEvidence(safetyAboveThreshold, p)
	if err != nil {
		return nil, err
	}
	safetyPass := thresholdSafetyScore.GatePassed &&
		aboveSafetyScore.GatePassed &&
		math.Abs(thresholdSafetyScore.Total-aboveSafetyScore.Total) <= floatTolerance
	actorBefore := ActorPower{
		Qualified:       true,
		ConflictFree:    true,
		RewardBalance:   0,
		LiquidWealth:    1,
		CivicCredential: 0.6,
	}
	actorAfter := actorBefore
	actorAfter.RewardBalance = 1_000_000_000
	actorAfter.LiquidWealth = 1_000_000_000
	powerPass := ReviewVoice(actorBefore) == ReviewVoice(actorAfter) &&
		GovernanceVoice(actorBefore) == GovernanceVoice(actorAfter)
	sweepPass := true
	for _, row := range sweep {
		sweepPass = sweepPass && row.Passed
	}
	return []GateResult{
		{
			Class:  "model",
			Name:   "budget-conservation",
			Passed: budgetPass,
			Detail: fmt.Sprintf("direct + commons + unallocated = %.9f of %.9f", accounted, epoch.Budget),
		},
		{
			Class: "model",
			Name:  "preclustered-artifact-count-invariance",
			Passed: math.Abs(
				attacks.PreclusteredOneArtifactDirect-attacks.PreclusteredHundredDirect,
			) <= amountTolerance(
				attacks.PreclusteredOneArtifactDirect,
				attacks.PreclusteredHundredDirect,
			) &&
				amountsEqual(
					attacks.PreclusteredOneArtifactFunded,
					attacks.PreclusteredHundredFunded,
				) &&
				amountsEqual(
					attacks.PreclusteredOneCommons,
					attacks.PreclusteredHundredCommons,
				),
			Detail: fmt.Sprintf(
				"given one equivalence cluster: funded %.9f/%.9f direct %.9f/%.9f commons %.9f/%.9f",
				attacks.PreclusteredOneArtifactFunded,
				attacks.PreclusteredHundredFunded,
				attacks.PreclusteredOneArtifactDirect,
				attacks.PreclusteredHundredDirect,
				attacks.PreclusteredOneCommons,
				attacks.PreclusteredHundredCommons,
			),
		},
		{
			Class: "model",
			Name:  "controller-alias-collapse",
			Passed: math.Abs(attacks.HundredAliasEffectiveCount-1) <= floatTolerance &&
				math.Abs(
					attacks.MixedPanelSingleAliasCount-attacks.MixedPanelHundredAliasCount,
				) <= floatTolerance &&
				math.Abs(
					attacks.MixedPanelSingleAliasScore-attacks.MixedPanelHundredAliasScore,
				) <= floatTolerance,
			Detail: fmt.Sprintf(
				"100 linked aliases alone n_eff %.9f; mixed panel count %.9f/%.9f score %.9f/%.9f",
				attacks.HundredAliasEffectiveCount,
				attacks.MixedPanelSingleAliasCount,
				attacks.MixedPanelHundredAliasCount,
				attacks.MixedPanelSingleAliasScore,
				attacks.MixedPanelHundredAliasScore,
			),
		},
		{
			Class:  "model",
			Name:   "temporal-accrual-no-pacing-advantage",
			Passed: abundantPath.Passed && scarcePath.Passed && competingPath.Passed,
			Detail: fmt.Sprintf(
				"isolated scarce gross %.9f/%.9f direct %.9f/%.9f; competing funded jump/pace %.9f/%.9f backlog %.9f/%.9f",
				scarcePath.JumpGross,
				scarcePath.SteppedGross,
				scarcePath.JumpDirect,
				scarcePath.SteppedDirect,
				competingPath.JumpFunded,
				competingPath.PacedFunded,
				competingPath.JumpBacklog,
				competingPath.PacedBacklog,
			),
		},
		{
			Class:  "model",
			Name:   "correlation-discount",
			Passed: math.Abs(attacks.MonocultureEffectiveCount-(100.0/20.8)) <= floatTolerance,
			Detail: fmt.Sprintf("100 controller signals at rho=.2 yield n_eff %.9f", attacks.MonocultureEffectiveCount),
		},
		{
			Class:  "model",
			Name:   "replication-quorum",
			Passed: quorumPass,
			Detail: "zero submitted/controller/effective replication signals fail the reward gate",
		},
		{
			Class:  "model",
			Name:   "multi-surface-decentralization-gate",
			Passed: decentralizationPass,
			Detail: "all 12 policy-owned surfaces are required; 80% captured infrastructure fails the gate",
		},
		{
			Class:  "model",
			Name:   "safety-non-compensation",
			Passed: safetyPass,
			Detail: "safety is a hard threshold; stronger passing safety evidence does not scale artifact score",
		},
		{
			Class:  "model",
			Name:   "strictly-proper-reviewer-payment",
			Passed: properPass,
			Detail: "positive affine Brier payment is maximized by truthful probabilities on the tested grid",
		},
		{
			Class:  "model",
			Name:   "state-owned-replay-protection",
			Passed: replayPass,
			Detail: "behavior-complete snapshots restore replay IDs, parameters, credits, and economic state",
		},
		{
			Class:  "model",
			Name:   "power-non-conversion",
			Passed: powerPass,
			Detail: "changing liquid wealth and reward balance changes neither review nor civic voice",
		},
		{
			Class:  "model",
			Name:   "parameter-sweep-invariants",
			Passed: sweepPass,
			Detail: "all alpha/cap cells preserve budget, cluster-lifetime cap, and newcomer reachability",
		},
	}, nil
}

func integrationGates() []GateResult {
	return []GateResult{
		{
			Class:  "integration",
			Name:   "canonical-tree-receipt-binding",
			Passed: false,
			Detail: "a zero-value offline adapter validates tree-v1 typed receipts, but neither calculator nor ledger consumes its decision or binds sponsor escrow and E0-E6 compartments",
		},
		{
			Class:  "integration",
			Name:   "canonical-independence-disclosure",
			Passed: false,
			Detail: "tree-v1 organization, implementation, environment, assignment, authorization, and disclosure-lane floors are not enforced",
		},
		{
			Class:  "integration",
			Name:   "semantic-equivalence-system",
			Passed: false,
			Detail: "the model assumes preclustered artifacts; no production theorem/proof DAG adjudicator is wired",
		},
		{
			Class:  "integration",
			Name:   "node-frontier-revocation-ledger",
			Passed: false,
			Detail: "the model has one irreversible economic scalar, not per-node epistemic frontiers and revocation transitions",
		},
		{
			Class:  "integration",
			Name:   "cap-poisoning-replacement-policy",
			Passed: false,
			Detail: "an exact one-shot local shadow transition exists, but no authenticated production adjudication, root, controller, or receipt binding executes it",
		},
		{
			Class:  "integration",
			Name:   "deterministic-backlog-scheduler",
			Passed: false,
			Detail: "eligible backlog persists when a cluster is evaluated, but no production scheduler automatically carries every backlog into later cohorts",
		},
		{
			Class:  "integration",
			Name:   "backlog-expiry-extinguishment",
			Passed: false,
			Detail: "the exact local shadow ledger has expiring lots and irreversible extinguishment, but no production scheduler or durable authenticated state executes them",
		},
		{
			Class:  "integration",
			Name:   "controller-correlation-attestation",
			Passed: false,
			Detail: "controller/family labels, correlations, appeals, and privacy-preserving attestations are not implemented",
		},
		{
			Class:  "integration",
			Name:   "authoritative-power-and-path-cut",
			Passed: false,
			Detail: "the model requires 12 policy-owned surfaces; authenticated live snapshots and joint path-cut analysis are absent",
		},
		{
			Class:  "integration",
			Name:   "program-wide-controller-exposure",
			Passed: false,
			Detail: "cluster-lifetime caps exist in-model; a persistent cross-cluster program cap is not designed",
		},
		{
			Class:  "integration",
			Name:   "collateralized-reserve",
			Passed: false,
			Detail: "no dedicated pre-funded breakthrough reserve and resolve-or-expire ledger exists",
		},
		{
			Class:  "integration",
			Name:   "atomic-role-commons-settlement",
			Passed: false,
			Detail: "the calculator has controller totals only; no role/tranche bank batch or commons paid-to-date state exists",
		},
		{
			Class:  "integration",
			Name:   "terminal-disposition",
			Passed: false,
			Detail: "expiry, invalidation, missing roles, and unused tranche weight are not executed into named commons or refund accounts",
		},
		{
			Class:  "integration",
			Name:   "durable-replay-ledger",
			Passed: false,
			Detail: "behavior-complete in-memory snapshots exist, but no durable consensus replay store or migration exists",
		},
		{
			Class:  "integration",
			Name:   "governance-power-separation",
			Passed: false,
			Detail: "current custom governance reads bonded stake; reward-to-power conversion remains possible",
		},
		{
			Class:  "integration",
			Name:   "formal-proof-verification",
			Passed: false,
			Detail: "pinned multi-kernel builds and axiom-policy checks are not consensus-enforced",
		},
		{
			Class:  "integration",
			Name:   "reviewer-outcome-and-budget-enforcement",
			Passed: false,
			Detail: "the proper-score primitive lacks assignment uniqueness, exogenous outcome, effort, and isolated-budget plumbing",
		},
		{
			Class:  "integration",
			Name:   "fixed-point-independent-implementation",
			Passed: false,
			Detail: "the exploratory float model has no second exact implementation or golden vectors",
		},
		{
			Class:  "integration",
			Name:   "external-red-team",
			Passed: false,
			Detail: "internal adversarial passes drove corrections; independent external reproduction remains required",
		},
	}
}

// RunSimulation produces one reproducible report and never reads chain state,
// network services, environment variables, clocks, or random sources.
func RunSimulation(p Params) (SimulationReport, error) {
	epoch, err := runRegisteredEpoch(p, "illustrative-epoch", illustrativeClusters())
	if err != nil {
		return SimulationReport{}, err
	}
	attacks, err := runAttacks(p)
	if err != nil {
		return SimulationReport{}, err
	}
	sweep, err := runSweep(p)
	if err != nil {
		return SimulationReport{}, err
	}
	power, weakest, err := AuditRequiredPower(
		capturedPowerSurfaces(),
		p.PowerCoalitionThresholds,
	)
	if err != nil {
		return SimulationReport{}, err
	}
	model, err := modelGates(p, epoch, attacks, sweep)
	if err != nil {
		return SimulationReport{}, err
	}
	integration := integrationGates()
	modelChecksPassed := true
	for _, gate := range model {
		modelChecksPassed = modelChecksPassed && gate.Passed
	}
	integrationReady := true
	for _, gate := range integration {
		integrationReady = integrationReady && gate.Passed
	}
	return SimulationReport{
		Parameters:            p,
		IllustrativeEpoch:     epoch,
		Attacks:               attacks,
		PowerSurfaces:         power,
		WeakestEffectivePower: weakest,
		Sweep:                 sweep,
		ReleaseGates:          append(model, integration...),
		ModelChecksPassed:     modelChecksPassed,
		IntegrationReady:      integrationReady,
	}, nil
}
