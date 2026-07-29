package main

import (
	"fmt"
	"math"
	"testing"
)

func FuzzCappedConcaveConservation(f *testing.F) {
	f.Add(uint16(1), uint16(100), uint16(50), uint16(90), uint8(128))
	f.Add(uint16(0), uint16(0), uint16(0), uint16(1_000), uint8(255))
	f.Add(uint16(997), uint16(991), uint16(983), uint16(977), uint8(1))

	f.Fuzz(func(t *testing.T, a, b, c, budgetRaw uint16, alphaRaw uint8) {
		demands := []float64{float64(a), float64(b), float64(c)}
		budget := float64(budgetRaw)
		alpha := 0.10 + 0.90*float64(alphaRaw)/255
		allocated := allocateCappedConcave(demands, budget, alpha)

		var totalDemand, totalAllocated float64
		for i := range demands {
			totalDemand += demands[i]
			totalAllocated += allocated[i]
			if allocated[i] < -floatTolerance {
				t.Fatalf("negative allocation at %d: %.12f", i, allocated[i])
			}
			if allocated[i] > demands[i]+floatTolerance {
				t.Fatalf("overfunded demand at %d: %.12f > %.12f", i, allocated[i], demands[i])
			}
		}
		want := math.Min(totalDemand, budget)
		if math.Abs(totalAllocated-want) > 1e-7 {
			t.Fatalf("allocation sum %.12f, want %.12f", totalAllocated, want)
		}
	})
}

func FuzzCappedConcaveExtremeScale(f *testing.F) {
	f.Add(uint16(1), uint16(500), uint16(999), uint16(250), uint8(0), uint8(128))
	f.Add(uint16(999), uint16(997), uint16(991), uint16(1), uint8(21), uint8(255))

	f.Fuzz(func(
		t *testing.T,
		aRaw, bRaw, cRaw, budgetRaw uint16,
		scaleRaw, alphaRaw uint8,
	) {
		exponent := int(scaleRaw%22) - 12
		scale := math.Pow10(exponent)
		toAmount := func(raw uint16) float64 {
			return (0.001 + float64(raw%1000)/1000) * scale
		}
		demands := []float64{toAmount(aRaw), toAmount(bRaw), toAmount(cRaw)}
		budget := toAmount(budgetRaw)
		alpha := 0.10 + 0.90*float64(alphaRaw)/255
		allocated := allocateCappedConcave(demands, budget, alpha)

		var totalDemand, totalAllocated float64
		for i := range demands {
			totalDemand += demands[i]
			totalAllocated += allocated[i]
			if allocated[i] < 0 || allocated[i] > demands[i] {
				t.Fatalf(
					"allocation[%d] %.18g outside [0, %.18g]",
					i,
					allocated[i],
					demands[i],
				)
			}
		}
		if totalAllocated > budget {
			t.Fatalf("allocated %.18g above budget %.18g", totalAllocated, budget)
		}
		if !amountsEqual(totalAllocated, math.Min(totalDemand, budget)) {
			t.Fatalf(
				"allocation sum %.18g, want %.18g",
				totalAllocated,
				math.Min(totalDemand, budget),
			)
		}
	})
}

func FuzzUniformCorrelationEffectiveCount(f *testing.F) {
	f.Add(uint8(1), uint8(0))
	f.Add(uint8(100), uint8(20))
	f.Add(uint8(32), uint8(100))

	f.Fuzz(func(t *testing.T, countRaw, rhoPercent uint8) {
		count := int(countRaw%100) + 1
		rho := float64(rhoPercent%101) / 100
		effective, err := EffectiveIndependentCount(
			signals("fuzz", count),
			uniformCorrelation(count, rho),
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		want := float64(count) / (1 + float64(count-1)*rho)
		if math.Abs(effective-want) > 1e-8 {
			t.Fatalf("n_eff %.12f, want %.12f for n=%d rho=%.2f", effective, want, count, rho)
		}
		if effective < 1-floatTolerance || effective > float64(count)+floatTolerance {
			t.Fatalf("n_eff %.12f outside [1,%d]", effective, count)
		}
	})
}

func FuzzCumulativeTargetNoPacingAdvantage(f *testing.F) {
	f.Add(uint16(1_000), uint16(100), uint16(600), uint16(900))
	f.Add(uint16(65_535), uint16(0), uint16(32_768), uint16(65_535))

	f.Fuzz(func(t *testing.T, capRaw, firstRaw, secondRaw, finalRaw uint16) {
		exponent := int(capRaw%16) - 6
		capAmount := math.Pow10(exponent) *
			(1 + float64((capRaw/16)%1000)/1000)
		capAmount = math.Max(minSimulationAmount, math.Min(maxSimulationAmount, capAmount))
		points := []float64{
			float64(firstRaw) / 65_535,
			float64(secondRaw) / 65_535,
			float64(finalRaw) / 65_535,
		}
		// Sort three points without bringing random map or clock state into the
		// model.
		if points[0] > points[1] {
			points[0], points[1] = points[1], points[0]
		}
		if points[1] > points[2] {
			points[1], points[2] = points[2], points[1]
		}
		if points[0] > points[1] {
			points[0], points[1] = points[1], points[0]
		}

		params := DefaultParams()
		budgetFraction := 0.05 + 0.95*float64(firstRaw)/65_535
		params.Budget = math.Max(minSimulationAmount, capAmount*budgetFraction)
		params.Budget = math.Min(maxSimulationAmount, params.Budget)
		credits := []ControllerCredit{
			{Controller: "fuzz-controller-a", Credit: 1},
			{Controller: "fuzz-controller-b", Credit: 2},
		}

		jumpEngine, err := NewEngine(params)
		if err != nil {
			t.Fatal(err)
		}
		if err := jumpEngine.RegisterCluster("fuzz-cluster", capAmount, credits); err != nil {
			t.Fatal(err)
		}
		var jumpGross, jumpDirect, jumpCommons float64
		for i := 0; i < 3; i++ {
			jump, err := jumpEngine.RunEpoch(
				fmt.Sprintf("jump-%d", i),
				[]ClusterProposal{{
					EventID:   fmt.Sprintf("jump-event-%d", i),
					ClusterID: "fuzz-cluster",
					Evidence:  baseEvidence(points[2], 0.7, signals("fuzz-path", 3)),
				}},
			)
			if err != nil {
				t.Fatal(err)
			}
			jumpGross += jump.Clusters[0].GrossEntitlement
			jumpDirect += jump.DirectTotal
			jumpCommons += jump.CommonsTotal
		}

		stepEngine, err := NewEngine(params)
		if err != nil {
			t.Fatal(err)
		}
		if err := stepEngine.RegisterCluster("fuzz-cluster", capAmount, credits); err != nil {
			t.Fatal(err)
		}
		var steppedGross, steppedDirect, steppedCommons float64
		for i, point := range points {
			step, err := stepEngine.RunEpoch(
				fmt.Sprintf("step-%d", i),
				[]ClusterProposal{{
					EventID:   fmt.Sprintf("step-event-%d", i),
					ClusterID: "fuzz-cluster",
					Evidence:  baseEvidence(point, 0.7, signals("fuzz-path", 3)),
				}},
			)
			if err != nil {
				t.Fatal(err)
			}
			steppedGross += step.Clusters[0].GrossEntitlement
			steppedDirect += step.DirectTotal
			steppedCommons += step.CommonsTotal
		}

		if !amountsEqual(jumpGross, steppedGross) ||
			steppedDirect > jumpDirect+amountTolerance(steppedDirect, jumpDirect) ||
			steppedDirect+steppedCommons >
				jumpDirect+jumpCommons+
					amountTolerance(
						steppedDirect+steppedCommons,
						jumpDirect+jumpCommons,
					) {
			t.Fatalf(
				"pacing advantage: gross %.12f/%.12f direct %.12f/%.12f commons %.12f/%.12f",
				jumpGross,
				steppedGross,
				jumpDirect,
				steppedDirect,
				jumpCommons,
				steppedCommons,
			)
		}
	})
}

func FuzzCompetingCohortNoPacingAdvantage(f *testing.F) {
	f.Add(uint16(10_000), uint16(60_000), uint16(50_000), uint16(50_000), uint16(100), uint8(128))
	f.Add(uint16(0), uint16(65_535), uint16(65_535), uint16(0), uint16(1_000), uint8(255))

	f.Fuzz(func(
		t *testing.T,
		firstRaw, finalRaw, competitorFirstRaw, competitorSecondRaw uint16,
		budgetRaw uint16,
		alphaRaw uint8,
	) {
		first := float64(firstRaw) / 65_535
		final := float64(finalRaw) / 65_535
		if first > final {
			first, final = final, first
		}
		competitor := []float64{
			float64(competitorFirstRaw) / 65_535,
			float64(competitorSecondRaw) / 65_535,
		}
		params := DefaultParams()
		params.Budget = 1 + float64(budgetRaw%1_000)
		params.Alpha = 0.10 + 0.90*float64(alphaRaw)/255
		params.ControllerCapShare = 1

		jump, err := runCompetingTemporalPath(
			params,
			"fuzz-competing-jump",
			[]float64{final, final},
			competitor,
		)
		if err != nil {
			t.Fatal(err)
		}
		paced, err := runCompetingTemporalPath(
			params,
			"fuzz-competing-paced",
			[]float64{first, final},
			competitor,
		)
		if err != nil {
			t.Fatal(err)
		}
		if !amountsEqual(jump.Gross, paced.Gross) {
			t.Fatalf("gross accrual changed: jump %.12f, paced %.12f", jump.Gross, paced.Gross)
		}
		if paced.Funded > jump.Funded+amountTolerance(paced.Funded, jump.Funded) {
			t.Fatalf(
				"pacing increased funding: jump %.12f, paced %.12f; backlog %.12f/%.12f",
				jump.Funded,
				paced.Funded,
				jump.Backlog,
				paced.Backlog,
			)
		}
	})
}
