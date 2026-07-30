package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= floatTolerance
}

func TestDefaultParamsValidate(t *testing.T) {
	if err := DefaultParams().Validate(); err != nil {
		t.Fatalf("default parameters should validate: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{"negative budget", func(p *Params) { p.Budget = -1 }},
		{"zero budget", func(p *Params) { p.Budget = 0 }},
		{"subminimum budget", func(p *Params) { p.Budget = minSimulationAmount / 2 }},
		{"zero alpha", func(p *Params) { p.Alpha = 0 }},
		{"alpha above one", func(p *Params) { p.Alpha = 1.01 }},
		{"zero controller cap", func(p *Params) { p.ControllerCapShare = 0 }},
		{"invalid gate", func(p *Params) { p.MinValidity = 1.1 }},
		{"invalid family correlation", func(p *Params) { p.SameFamilyCorrelationFloor = -0.1 }},
		{"zero replication target", func(p *Params) { p.ReplicationTarget = 0 }},
		{"zero submitted quorum", func(p *Params) { p.MinSubmittedSignals = 0 }},
		{"controller quorum differs", func(p *Params) { p.MinControllerSignals++ }},
		{"effective quorum above controllers", func(p *Params) { p.MinEffectiveSignals = float64(p.MinControllerSignals) + 1 }},
		{"zero power target", func(p *Params) { p.PowerEffectiveTarget = 0 }},
		{"zero score epsilon", func(p *Params) { p.ScoreEpsilon = 0 }},
		{"unsafe score epsilon", func(p *Params) { p.ScoreEpsilon = math.Nextafter(1, 0) }},
		{"missing power surface", func(p *Params) {
			delete(p.PowerSurfaces, requiredPowerSurfaceNames[0])
		}},
		{"arbitrary power surface", func(p *Params) {
			p.PowerSurfaces = map[string]map[string]float64{
				"decoy": {"a": 1, "b": 1, "c": 1, "d": 1},
			}
		}},
		{"nan power threshold", func(p *Params) {
			p.PowerCoalitionThresholds[requiredPowerSurfaceNames[0]] = math.NaN()
		}},
		{"weights do not sum", func(p *Params) { p.Weights.Use = 0 }},
		{"negative weight", func(p *Params) {
			p.Weights.Marginal = -0.1
			p.Weights.Use = 0.725
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := DefaultParams()
			test.mutate(&params)
			if err := params.Validate(); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func TestEffectiveIndependentCountEndpoints(t *testing.T) {
	independent := signals("independent", 20)
	count, err := EffectiveIndependentCount(independent, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(count, 20) {
		t.Fatalf("independent count = %.12f, want 20", count)
	}

	fullyCorrelated, err := EffectiveIndependentCount(
		signals("correlated", 20),
		uniformCorrelation(20, 1),
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(fullyCorrelated, 1) {
		t.Fatalf("fully correlated count = %.12f, want 1", fullyCorrelated)
	}

	aliases := make([]Signal, 100)
	for i := range aliases {
		aliases[i] = Signal{Controller: "same-controller"}
	}
	aliasCount, err := EffectiveIndependentCount(aliases, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(aliasCount, 1) {
		t.Fatalf("linked alias count = %.12f, want 1", aliasCount)
	}

	singleMixed := []Signal{
		{Controller: "alias-controller"},
		{Controller: "independent-controller"},
	}
	hundredMixed := append([]Signal(nil), aliases...)
	hundredMixed = append(hundredMixed, Signal{Controller: "independent-controller"})
	singleCount, err := EffectiveIndependentCount(singleMixed, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	hundredCount, err := EffectiveIndependentCount(hundredMixed, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(singleCount, 2) || !almostEqual(singleCount, hundredCount) {
		t.Fatalf("alias collapse changed mixed panel: one %.12f, hundred %.12f", singleCount, hundredCount)
	}
	singleScore, err := ScoreEvidence(
		baseEvidence(0.8, 0.8, singleMixed),
		DefaultParams(),
	)
	if err != nil {
		t.Fatal(err)
	}
	hundredScore, err := ScoreEvidence(
		baseEvidence(0.8, 0.8, hundredMixed),
		DefaultParams(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if singleScore.GatePassed != hundredScore.GatePassed ||
		!almostEqual(singleScore.Total, hundredScore.Total) {
		t.Fatalf(
			"aliases changed reward gate or score: one %+v, hundred %+v",
			singleScore,
			hundredScore,
		)
	}

	correlatedAliases := []Signal{
		{Controller: "controller-a"},
		{Controller: "controller-a"},
		{Controller: "controller-b"},
	}
	correlatedAliasCount, err := EffectiveIndependentCount(
		correlatedAliases,
		[][]float64{{1, 0.4}, {0.4, 1}},
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	correlatedSingleCount, err := EffectiveIndependentCount(
		singleMixed,
		[][]float64{{1, 0.4}, {0.4, 1}},
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(correlatedAliasCount, correlatedSingleCount) {
		t.Fatalf(
			"alias collapse failed conservative cross-controller correlation: aliases %.12f, collapsed %.12f",
			correlatedAliasCount,
			correlatedSingleCount,
		)
	}
}

func TestMonocultureEffectiveCount(t *testing.T) {
	count, err := EffectiveIndependentCount(
		signals("monoculture", 100),
		uniformCorrelation(100, 0.20),
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 100.0 / (1 + 99*0.20)
	if !almostEqual(count, want) {
		t.Fatalf("n_eff = %.12f, want %.12f", count, want)
	}
}

func TestCorrelationMonotonicity(t *testing.T) {
	reviewers := signals("reviewer", 12)
	var previous float64
	for i, rho := range []float64{0, 0.05, 0.20, 0.50, 1} {
		count, err := EffectiveIndependentCount(reviewers, uniformCorrelation(len(reviewers), rho), 0)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && count > previous+floatTolerance {
			t.Fatalf("n_eff increased from %.12f to %.12f when rho increased", previous, count)
		}
		previous = count
	}
}

func TestCorrelationMatrixFailsClosed(t *testing.T) {
	reviewers := signals("matrix", 2)
	tests := []struct {
		name   string
		matrix [][]float64
	}{
		{"non-unit diagonal", [][]float64{{0, 0.2}, {0.2, 1}}},
		{"asymmetric", [][]float64{{1, 0.2}, {0.3, 1}}},
		{"ragged", [][]float64{{1, 0.2}, {}}},
		{"out of range", [][]float64{{1, 1.1}, {1.1, 1}}},
		{"not positive semidefinite", [][]float64{
			{1, 0.9, 0.9},
			{0.9, 1, 0},
			{0.9, 0, 1},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testReviewers := reviewers
			if len(test.matrix) == 3 {
				testReviewers = signals("matrix", 3)
			}
			if _, err := EffectiveIndependentCount(testReviewers, test.matrix, 0); err == nil {
				t.Fatal("expected invalid correlation matrix to fail")
			}
		})
	}
}

func TestControllerCorrelationIsPostCollapseAndFamilyConsistent(t *testing.T) {
	aliases := []Signal{
		{Controller: "same", Family: "family-a"},
		{Controller: "same", Family: "family-a"},
		{Controller: "other", Family: "family-b"},
	}
	if _, err := EffectiveIndependentCount(
		aliases,
		uniformCorrelation(3, 0.2),
		0,
	); err == nil {
		t.Fatal("raw-signal-sized correlation matrix should fail after controller collapse")
	}
	conflicting := append([]Signal(nil), aliases...)
	conflicting[1].Family = "conflicting-family"
	if _, err := EffectiveIndependentCount(conflicting, nil, 0); err == nil {
		t.Fatal("conflicting family labels within one controller should fail")
	}
	sameFamily := []Signal{
		{Controller: "a", Family: "shared"},
		{Controller: "b", Family: "shared"},
	}
	if _, err := EffectiveIndependentCount(
		sameFamily,
		[][]float64{{1, 0.2}, {0.2, 1}},
		0.75,
	); err == nil {
		t.Fatal("controller correlation below the policy family floor should fail")
	}
}

func TestScoreHardGate(t *testing.T) {
	params := DefaultParams()
	evidence := baseEvidence(1, 1, signals("gate", 5))
	evidence.Validity = params.MinValidity - 0.01
	score, err := ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}
	if score.GatePassed || score.Total != 0 {
		t.Fatalf("failed hard gate produced score %+v", score)
	}

	evidence.Validity = params.MinValidity
	score, err = ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}
	if !score.GatePassed || score.Total <= 0 {
		t.Fatalf("passing evidence did not score: %+v", score)
	}

	noReplication := baseEvidence(1, 1, nil)
	score, err = ScoreEvidence(noReplication, params)
	if err != nil {
		t.Fatal(err)
	}
	if score.GatePassed || score.Total != 0 {
		t.Fatalf("zero-replication evidence passed: %+v", score)
	}

	captured := baseEvidence(1, 1, signals("captured-gate", 4))
	capturedParams := cloneParams(params)
	capturedParams.PowerSurfaces = capturedPowerSurfaces()
	score, err = ScoreEvidence(captured, capturedParams)
	if err != nil {
		t.Fatal(err)
	}
	if score.GatePassed || score.Total != 0 {
		t.Fatalf("captured power surface passed: %+v", score)
	}
}

func TestPolicyPowerIsACompleteHardGateNotArtifactReward(t *testing.T) {
	evidence := baseEvidence(0.8, 0.8, signals("power-policy", 4))
	balancedParams := DefaultParams()
	balanced, err := ScoreEvidence(evidence, balancedParams)
	if err != nil {
		t.Fatal(err)
	}
	if !balanced.GatePassed {
		t.Fatal("balanced policy snapshot should pass")
	}

	concentratedParams := cloneParams(balancedParams)
	for _, surface := range requiredPowerSurfaceNames {
		concentratedParams.PowerSurfaces[surface] = map[string]float64{
			"a": 0.30,
			"b": 0.30,
			"c": 0.20,
			"d": 0.20,
		}
	}
	concentrated, err := ScoreEvidence(evidence, concentratedParams)
	if err != nil {
		t.Fatal(err)
	}
	if !concentrated.GatePassed {
		t.Fatal("less balanced but above-floor policy snapshot should pass")
	}
	if !almostEqual(balanced.Total, concentrated.Total) {
		t.Fatalf(
			"global power changed artifact reward score: balanced %.12f, concentrated %.12f",
			balanced.Total,
			concentrated.Total,
		)
	}
	if !(concentrated.Decentralization < balanced.Decentralization) {
		t.Fatal("power diagnostic should still expose weaker concentration")
	}

	missing := cloneParams(balancedParams)
	delete(missing.PowerSurfaces, requiredPowerSurfaceNames[0])
	if _, err := ScoreEvidence(evidence, missing); err == nil {
		t.Fatal("missing required policy surface should fail closed")
	}
}

func TestSafetyIsAHardGateNotACompensableScore(t *testing.T) {
	params := DefaultParams()
	evidence := baseEvidence(0.8, 0.8, signals("safety-gate", 4))
	evidence.Safety = params.MinSafety
	atThreshold, err := ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}
	evidence.Safety = 1
	aboveThreshold, err := ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}
	if !atThreshold.GatePassed || !aboveThreshold.GatePassed {
		t.Fatal("passing safety evidence should pass the hard gate")
	}
	if !almostEqual(atThreshold.Total, aboveThreshold.Total) {
		t.Fatalf(
			"passing safety value scaled reward score: threshold %.12f, maximum %.12f",
			atThreshold.Total,
			aboveThreshold.Total,
		)
	}
	evidence.Safety = math.Nextafter(params.MinSafety, 0)
	belowThreshold, err := ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}
	if belowThreshold.GatePassed || belowThreshold.Total != 0 {
		t.Fatalf("sub-threshold safety passed: %+v", belowThreshold)
	}
}

func oneCluster(evidence Evidence, prior float64) Cluster {
	return Cluster{
		ID:             "one-cluster",
		ArtifactIDs:    []string{"artifact"},
		Evidence:       evidence,
		PriorHighWater: prior,
		LifetimeCap:    1_000,
		Credits: []ControllerCredit{
			{Controller: "one-controller", Credit: 1},
		},
	}
}

func TestHighWaterReplayAndRegressionPayZero(t *testing.T) {
	params := DefaultParams()
	params.ControllerCapShare = 1
	evidence := baseEvidence(0.80, 0.75, signals("highwater", 4))
	engine, err := NewEngine(params)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterCluster(
		"one-cluster",
		1_000,
		[]ControllerCredit{{Controller: "one-controller", Credit: 1}},
	); err != nil {
		t.Fatal(err)
	}
	firstProposal := ClusterProposal{
		EventID:   "first-event",
		ClusterID: "one-cluster",
		Evidence:  evidence,
	}
	first, err := engine.RunEpoch("first-epoch", []ClusterProposal{firstProposal})
	if err != nil {
		t.Fatal(err)
	}
	if first.Clusters[0].GrossEntitlement <= 0 {
		t.Fatal("first evidence should create entitlement")
	}
	if _, err := engine.RunEpoch("replay-epoch", []ClusterProposal{firstProposal}); err == nil {
		t.Fatal("repeated event ID should fail")
	}

	replay, err := engine.RunEpoch("same-score-epoch", []ClusterProposal{{
		EventID:   "same-score-event",
		ClusterID: "one-cluster",
		Evidence:  evidence,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if replay.Clusters[0].GrossEntitlement != 0 || replay.DirectTotal != 0 {
		t.Fatalf("replay paid again: %+v", replay)
	}
	if !almostEqual(replay.Unallocated, params.Budget) {
		t.Fatalf("empty entitlement should leave budget unallocated: %.12f", replay.Unallocated)
	}

	regressed := evidence
	regressed.Marginal = 0.10
	regression, err := engine.RunEpoch("regression-epoch", []ClusterProposal{{
		EventID:   "regression-event",
		ClusterID: "one-cluster",
		Evidence:  regressed,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if regression.Clusters[0].GrossEntitlement != 0 || regression.DirectTotal != 0 {
		t.Fatalf("regression created reward: %+v", regression)
	}
	if !almostEqual(
		regression.HighWater["one-cluster"],
		first.HighWater["one-cluster"],
	) {
		t.Fatalf(
			"regression lowered high water from %.12f to %.12f",
			first.HighWater["one-cluster"],
			regression.HighWater["one-cluster"],
		)
	}

	recovery, err := engine.RunEpoch("recovery-epoch", []ClusterProposal{{
		EventID:   "recovery-event",
		ClusterID: "one-cluster",
		Evidence:  evidence,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if recovery.Clusters[0].GrossEntitlement != 0 ||
		recovery.DirectTotal != 0 ||
		recovery.CommonsTotal != 0 {
		t.Fatalf("recovery reaccrued the already-reached economic target: %+v", recovery)
	}
}

func TestEngineRejectsInvalidBatchAtomically(t *testing.T) {
	engine, err := NewEngine(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	credits := []ControllerCredit{{Controller: "controller", Credit: 1}}
	for _, clusterID := range []string{"atomic-a", "atomic-b"} {
		if err := engine.RegisterCluster(clusterID, 1_000, credits); err != nil {
			t.Fatal(err)
		}
	}
	before := engine.Snapshot()
	invalid := baseEvidence(0.8, 0.8, signals("atomic-invalid", 3))
	invalid.Validity = 2
	proposals := []ClusterProposal{
		{
			EventID:   "atomic-event-a",
			ClusterID: "atomic-a",
			Evidence:  baseEvidence(0.8, 0.8, signals("atomic-a", 3)),
		},
		{
			EventID:   "atomic-event-b",
			ClusterID: "atomic-b",
			Evidence:  invalid,
		},
	}
	if _, err := engine.RunEpoch("atomic-epoch", proposals); err == nil {
		t.Fatal("invalid batch should fail")
	}
	if after := engine.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("invalid batch changed state: before=%+v after=%+v", before, after)
	}

	proposals[1].Evidence = baseEvidence(0.8, 0.8, signals("atomic-b", 3))
	if _, err := engine.RunEpoch("atomic-epoch", proposals); err != nil {
		t.Fatalf("failed batch consumed epoch or event replay IDs: %v", err)
	}
}

func TestEngineOwnsPowerPolicySnapshot(t *testing.T) {
	params := DefaultParams()
	engine, err := NewEngine(params)
	if err != nil {
		t.Fatal(err)
	}
	credits := []ControllerCredit{{
		Controller: "controller",
		Credit:     1,
		Roles:      []string{"originator"},
	}}
	if err := engine.RegisterCluster(
		"policy-owned",
		1_000,
		credits,
	); err != nil {
		t.Fatal(err)
	}
	credits[0].Roles[0] = "caller-mutated"
	for controller := range params.PowerSurfaces["infrastructure"] {
		params.PowerSurfaces["infrastructure"][controller] = 0
	}
	params.PowerCoalitionThresholds["infrastructure"] = math.NaN()
	result, err := engine.RunEpoch("policy-epoch", []ClusterProposal{{
		EventID:   "policy-event",
		ClusterID: "policy-owned",
		Evidence:  baseEvidence(0.8, 0.8, signals("policy-owned", 3)),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Clusters[0].Score.GatePassed {
		t.Fatal("caller mutation changed the engine-owned power policy")
	}
	if got := engine.Snapshot().Clusters["policy-owned"].Credits[0].Roles[0]; got != "originator" {
		t.Fatalf("caller mutation changed immutable credit roles to %q", got)
	}
}

func TestEngineSnapshotIsBehaviorCompleteAndRestorable(t *testing.T) {
	engine, err := NewEngine(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterCluster(
		"snapshot-cluster",
		1_000,
		[]ControllerCredit{{
			Controller: "snapshot-controller",
			Credit:     1,
			Roles:      []string{"originator"},
		}},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RunEpoch("empty-snapshot-epoch", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RunEpoch("snapshot-epoch", []ClusterProposal{{
		EventID:   "snapshot-event",
		ClusterID: "snapshot-cluster",
		Evidence:  baseEvidence(0.5, 0.5, signals("snapshot", 3)),
	}}); err != nil {
		t.Fatal(err)
	}

	snapshot := engine.Snapshot()
	restored, err := RestoreEngine(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, restored.Snapshot()) {
		t.Fatal("restored engine snapshot differs")
	}
	if _, err := restored.RunEpoch("empty-snapshot-epoch", nil); err == nil {
		t.Fatal("restored snapshot forgot an empty epoch replay ID")
	}
	if _, err := restored.RunEpoch("other-epoch", []ClusterProposal{{
		EventID:   "snapshot-event",
		ClusterID: "snapshot-cluster",
		Evidence:  baseEvidence(0.5, 0.5, signals("snapshot", 3)),
	}}); err == nil {
		t.Fatal("restored snapshot forgot an event replay ID")
	}

	nextProposal := []ClusterProposal{{
		EventID:   "snapshot-next-event",
		ClusterID: "snapshot-cluster",
		Evidence:  baseEvidence(0.8, 0.8, signals("snapshot", 3)),
	}}
	originalNext, err := engine.RunEpoch("snapshot-next-epoch", nextProposal)
	if err != nil {
		t.Fatal(err)
	}
	restoredNext, err := restored.RunEpoch("snapshot-next-epoch", nextProposal)
	if err != nil {
		t.Fatal(err)
	}
	originalJSON, _ := json.Marshal(originalNext)
	restoredJSON, _ := json.Marshal(restoredNext)
	if !bytes.Equal(originalJSON, restoredJSON) {
		t.Fatal("restored engine produced different future behavior")
	}
}

func TestEngineSerializesConcurrentReplay(t *testing.T) {
	engine, err := NewEngine(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterCluster(
		"concurrent-cluster",
		1_000,
		[]ControllerCredit{{Controller: "controller", Credit: 1}},
	); err != nil {
		t.Fatal(err)
	}
	proposal := []ClusterProposal{{
		EventID:   "concurrent-event",
		ClusterID: "concurrent-cluster",
		Evidence:  baseEvidence(0.8, 0.8, signals("concurrent", 3)),
	}}
	errorsByCall := make(chan error, 2)
	var wait sync.WaitGroup
	for _, epochID := range []string{"concurrent-epoch-a", "concurrent-epoch-b"} {
		wait.Add(1)
		go func(epochID string) {
			defer wait.Done()
			_, err := engine.RunEpoch(epochID, proposal)
			errorsByCall <- err
		}(epochID)
	}
	wait.Wait()
	close(errorsByCall)
	var successes, failures int
	for err := range errorsByCall {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent replay yielded %d successes and %d failures", successes, failures)
	}
}

func TestTemporalAccrualAndNoPacingAdvantage(t *testing.T) {
	for _, budget := range []float64{100, 10_000} {
		comparison, err := temporalPathComparison(DefaultParams(), budget)
		if err != nil {
			t.Fatal(err)
		}
		if !comparison.Passed {
			t.Fatalf("budget %.0f temporal comparison failed: %+v", budget, comparison)
		}
	}
	competing, err := competingTemporalNoPacingAdvantage(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	if !competing.Passed {
		t.Fatalf("competing-cohort temporal comparison failed: %+v", competing)
	}
}

func TestPreclusteredArtifactCountInvariance(t *testing.T) {
	params := DefaultParams()
	one, err := preclusteredArtifactScenario(1, params)
	if err != nil {
		t.Fatal(err)
	}
	thousand, err := preclusteredArtifactScenario(1_000, params)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(one.Funded, thousand.Funded) ||
		!almostEqual(one.Direct, thousand.Direct) ||
		!almostEqual(one.Commons, thousand.Commons) {
		t.Fatalf("artifact count changed economics: one %+v, thousand %+v", one, thousand)
	}
}

func TestFixedBudgetControllerCapAndNewcomerReachability(t *testing.T) {
	params := DefaultParams()
	params.ControllerCapShare = 0.15
	result, err := EvaluateEpoch(sweepClusters(), params)
	if err != nil {
		t.Fatal(err)
	}
	accounted := result.DirectTotal + result.CommonsTotal + result.Unallocated
	if !almostEqual(accounted, params.Budget) {
		t.Fatalf("budget not conserved: %.12f != %.12f", accounted, params.Budget)
	}
	for clusterID, state := range result.States {
		clusterResult := result.Clusters[0]
		for _, candidate := range result.Clusters {
			if candidate.ID == clusterID {
				clusterResult = candidate
				break
			}
		}
		for _, allocation := range clusterResult.Allocations {
			cap := params.ControllerCapShare * state.LifetimeCap
			if allocation.CumulativeDirectTarget > cap+floatTolerance {
				t.Fatalf(
					"cluster %q controller %q cumulative target %.12f exceeds lifetime cap %.12f",
					clusterID,
					allocation.Controller,
					allocation.CumulativeDirectTarget,
					cap,
				)
			}
		}
	}
	if result.ControllerDirect["newcomer"] <= 0 {
		t.Fatal("newcomer received no direct allocation")
	}
	if result.CommonsTotal <= 0 {
		t.Fatal("controller overflow should fund commons in this scenario")
	}
}

func TestEmptyAndSparseEpochDoNotExhaustBudget(t *testing.T) {
	params := DefaultParams()
	params.ControllerCapShare = 1
	evidence := baseEvidence(0.50, 0.50, signals("sparse", 2))
	score, err := ScoreEvidence(evidence, params)
	if err != nil {
		t.Fatal(err)
	}

	emptyCluster := oneCluster(evidence, score.Total)
	emptyCluster.FundedToDate = emptyCluster.LifetimeCap * score.Total
	emptyCluster.ControllerDirectToDate = map[string]float64{
		"one-controller": emptyCluster.FundedToDate,
	}
	empty, err := EvaluateEpoch([]Cluster{emptyCluster}, params)
	if err != nil {
		t.Fatal(err)
	}
	if empty.DirectTotal != 0 || !almostEqual(empty.Unallocated, params.Budget) {
		t.Fatalf("empty epoch spent budget: %+v", empty)
	}

	sparseCluster := oneCluster(evidence, score.Total-0.01)
	sparseCluster.LifetimeCap = 100
	sparseCluster.FundedToDate = sparseCluster.LifetimeCap * sparseCluster.PriorHighWater
	sparseCluster.ControllerDirectToDate = map[string]float64{
		"one-controller": sparseCluster.FundedToDate,
	}
	sparse, err := EvaluateEpoch([]Cluster{sparseCluster}, params)
	if err != nil {
		t.Fatal(err)
	}
	if sparse.DirectTotal <= 0 || sparse.DirectTotal >= params.Budget {
		t.Fatalf("sparse demand should spend some but not all budget: %+v", sparse)
	}
	if sparse.Unallocated <= 0 {
		t.Fatal("sparse epoch failed to retain unused budget")
	}
}

func TestCappedConcaveAllocationNeverOverfunds(t *testing.T) {
	got := allocateCappedConcave([]float64{1, 100}, 90, 0.5)
	if !almostEqual(got[0], 1) || !almostEqual(got[1], 89) {
		t.Fatalf("capped water-fill = %v, want [1 89]", got)
	}
	if !almostEqual(got[0]+got[1], 90) {
		t.Fatalf("allocation sum = %.12f, want 90", got[0]+got[1])
	}

	full := allocateCappedConcave([]float64{1, 2, 3}, 10, 0.5)
	if !reflect.DeepEqual(full, []float64{1, 2, 3}) {
		t.Fatalf("ample budget should fund exact demand, got %v", full)
	}
}

func TestCappedConcaveTinyBudgetNeverOverfundsDirectionally(t *testing.T) {
	budget := 1e-12
	allocated := allocateCappedConcave([]float64{5e-10}, budget, 0.65)
	if len(allocated) != 1 {
		t.Fatalf("got %d allocations, want 1", len(allocated))
	}
	if allocated[0] > budget {
		t.Fatalf("allocated %.18g above tiny budget %.18g", allocated[0], budget)
	}
	if allocated[0] < 0 || allocated[0] > 5e-10 {
		t.Fatalf("allocation %.18g outside demand bounds", allocated[0])
	}
}

func TestMaximumRangeMultiControllerTransitionsRemainClosed(t *testing.T) {
	params := DefaultParams()
	params.Budget = maxSimulationAmount / 7
	params.ControllerCapShare = 0.33
	engine, err := NewEngine(params)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.RegisterCluster(
		"maximum-range",
		maxSimulationAmount,
		[]ControllerCredit{
			{Controller: "controller-a", Credit: 1},
			{Controller: "controller-b", Credit: 2},
		},
	); err != nil {
		t.Fatal(err)
	}
	for i, marginal := range []float64{0.3, 0.6, 0.9} {
		result, err := engine.RunEpoch(
			fmt.Sprintf("maximum-epoch-%d", i),
			[]ClusterProposal{{
				EventID:   fmt.Sprintf("maximum-event-%d", i),
				ClusterID: "maximum-range",
				Evidence:  baseEvidence(marginal, 0.8, signals("maximum", 4)),
			}},
		)
		if err != nil {
			t.Fatal(err)
		}
		accounted := result.DirectTotal + result.CommonsTotal + result.Unallocated
		if !amountsEqual(accounted, params.Budget) {
			t.Fatalf("epoch %d accounting %.12f != budget %.12f", i, accounted, params.Budget)
		}
		if result.DirectTotal+result.CommonsTotal > params.Budget {
			t.Fatalf("epoch %d funded above budget", i)
		}
		state := result.States["maximum-range"]
		for controller, paid := range state.ControllerDirectToDate {
			capAmount := params.ControllerCapShare * state.LifetimeCap
			if paid > capAmount && !amountsEqual(paid, capAmount) {
				t.Fatalf(
					"epoch %d controller %q paid %.12f above cap %.12f",
					i,
					controller,
					paid,
					capAmount,
				)
			}
		}
	}
}

func TestControllerCreditMustBePreAggregated(t *testing.T) {
	cluster := oneCluster(baseEvidence(0.8, 0.8, signals("credit", 3)), 0)
	cluster.Credits = []ControllerCredit{
		{Controller: "same", Credit: 1},
		{Controller: "same", Credit: 1},
	}
	if _, err := EvaluateEpoch([]Cluster{cluster}, DefaultParams()); err == nil {
		t.Fatal("expected duplicate controller credit to fail closed")
	}
}

func TestBrierSkillIsSignedAndReviewerPaymentIsStrictlyProper(t *testing.T) {
	skill, err := BrierSkill(0.90, 1, 0.50)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(skill, 0.24) {
		t.Fatalf("Brier skill = %.12f, want .24", skill)
	}
	badSkill, err := BrierSkill(0.10, 1, 0.50)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(badSkill, -0.56) {
		t.Fatalf("worse-than-baseline signed skill = %.12f, want -.56", badSkill)
	}
	if _, err := BrierSkill(0.5, 0.5, 0.5); err == nil {
		t.Fatal("fractional resolved outcome should fail")
	}
	if _, err := ReviewerPayment(0.5, 1, 0, 0); err == nil {
		t.Fatal("non-positive bonus scale should fail")
	}
	for _, truth := range []float64{0.20, 0.50, 0.90} {
		truthful := expectedReviewerPayment(truth, truth)
		for i := 0; i <= 100; i++ {
			report := float64(i) / 100
			candidate := expectedReviewerPayment(truth, report)
			if candidate > truthful+floatTolerance {
				t.Fatalf(
					"truth %.2f: report %.2f yields %.12f above truthful %.12f",
					truth,
					report,
					candidate,
					truthful,
				)
			}
			if math.Abs(report-truth) > floatTolerance &&
				candidate >= truthful-floatTolerance {
				t.Fatalf(
					"truth %.2f: nontruthful report %.2f ties truthful expected payment %.12f",
					truth,
					report,
					truthful,
				)
			}
		}
	}
}

func TestRewardAndWealthDoNotConvertToPower(t *testing.T) {
	before := ActorPower{
		Qualified:       true,
		ConflictFree:    true,
		RewardBalance:   0,
		LiquidWealth:    1,
		CivicCredential: 0.75,
	}
	after := before
	after.RewardBalance = math.MaxFloat64
	after.LiquidWealth = math.MaxFloat64
	if ReviewVoice(before) != ReviewVoice(after) {
		t.Fatal("reward or wealth changed review voice")
	}
	if GovernanceVoice(before) != GovernanceVoice(after) {
		t.Fatal("reward or wealth changed governance voice")
	}
	nanCredential := before
	nanCredential.CivicCredential = math.NaN()
	if GovernanceVoice(nanCredential) != 0 {
		t.Fatal("NaN civic credential should fail closed")
	}
}

func TestPowerAuditUsesWeakestSurface(t *testing.T) {
	metrics, weakest, err := AuditPower(map[string]map[string]float64{
		"healthy":  {"a": 1, "b": 1, "c": 1, "d": 1},
		"captured": {"x": 9, "y": 1},
	}, 0.34)
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 2 {
		t.Fatalf("got %d metrics, want 2", len(metrics))
	}
	captured, err := MeasurePower("captured", map[string]float64{"x": 9, "y": 1}, 0.34)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(weakest, captured.EffectiveCount) {
		t.Fatalf("weakest effective count %.12f, want %.12f", weakest, captured.EffectiveCount)
	}
}

func TestPowerAuditFailsClosedOnInvalidOrOverflowingInput(t *testing.T) {
	tests := []struct {
		name      string
		surface   string
		powers    map[string]float64
		threshold float64
	}{
		{"missing surface", "", map[string]float64{"a": 1}, 0.34},
		{"missing controller", "surface", map[string]float64{"": 1}, 0.34},
		{"zero total", "surface", map[string]float64{"a": 0}, 0.34},
		{
			"overflowing aggregate",
			"surface",
			map[string]float64{"a": maxSimulationAmount, "b": 1},
			0.34,
		},
		{"nan threshold", "surface", map[string]float64{"a": 1}, math.NaN()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := MeasurePower(test.surface, test.powers, test.threshold); err == nil {
				t.Fatal("expected invalid power surface to fail")
			}
		})
	}
}

func TestPermutationDeterminism(t *testing.T) {
	params := DefaultParams()
	clusters := illustrativeClusters()
	first, err := EvaluateEpoch(clusters, params)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]Cluster(nil), clusters...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	for clusterIndex := range reversed {
		cluster := &reversed[clusterIndex]
		for left, right := 0, len(cluster.Credits)-1; left < right; left, right = left+1, right-1 {
			cluster.Credits[left], cluster.Credits[right] = cluster.Credits[right], cluster.Credits[left]
		}
		cluster.Evidence.Replications = reverseSignals(cluster.Evidence.Replications)
		cluster.Evidence.Correlation = reverseCorrelation(cluster.Evidence.Correlation)
	}
	second, err := EvaluateEpoch(reversed, params)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("permutation changed output:\n%s\n%s", firstJSON, secondJSON)
	}
}

func reverseSignals(input []Signal) []Signal {
	reversed := append([]Signal(nil), input...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}

func reverseCorrelation(input [][]float64) [][]float64 {
	if input == nil {
		return nil
	}
	size := len(input)
	reversed := make([][]float64, size)
	for row := 0; row < size; row++ {
		reversed[row] = make([]float64, size)
		for column := 0; column < size; column++ {
			reversed[row][column] = input[size-1-row][size-1-column]
		}
	}
	return reversed
}

func TestSimulationReportDeterministicAndFailClosed(t *testing.T) {
	first, err := RunSimulation(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunSimulation(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("same simulation produced different JSON")
	}
	if !first.ModelChecksPassed {
		t.Fatal("model invariants should pass")
	}
	if first.IntegrationReady {
		t.Fatal("integration gate must remain closed")
	}
	for _, row := range first.Sweep {
		if !row.Passed {
			t.Fatalf("sweep row failed: %+v", row)
		}
	}
}

func TestCLIExitCodes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-mode", "model"}, &stdout, &stderr); code != 0 {
		t.Fatalf("model mode exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "budget-conservation") {
		t.Fatalf("model output missing gates: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-mode", "release"}, &stdout, &stderr); code != 1 {
		t.Fatalf("release mode exit = %d, want fail-closed 1", code)
	}
	if !strings.Contains(stdout.String(), "integration ready: false") {
		t.Fatalf("release output missing closed status: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-mode", "shadow"}, &stdout, &stderr); code != 0 {
		t.Fatalf("shadow mode exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "settlement: 0 ZRN") ||
		!strings.Contains(stdout.String(), "integration ready: false") {
		t.Fatalf("shadow output crossed/missed its boundary: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-alpha", "0"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid parameters exit = %d, want usage/error 2", code)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-budget", "0"}, &stdout, &stderr); code != 2 {
		t.Fatalf("zero budget exit = %d, want usage/error 2", code)
	}
}
