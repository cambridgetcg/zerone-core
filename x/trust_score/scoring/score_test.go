package scoring

import (
	"fmt"
	"reflect"
	"testing"
)

func TestCalculateWeightedAccuracy(t *testing.T) {
	input := Input{SubmissionAccuracyBps: 900_000, Verification: []Verification{
		{AccuracyBps: 800_001, EffectiveWeight: 3},
		{AccuracyBps: 400_000, EffectiveWeight: 1},
		{AccuracyBps: 1_000_000, EffectiveWeight: 0},
	}}
	before := append([]Verification(nil), input.Verification...)
	got := Calculate(input)
	if got.SubmissionAccuracyBps != 900_000 || got.VerificationAccuracyBps != 700_000 || got.CompositeBps != 800_000 {
		t.Fatalf("weighted result: %+v", got)
	}
	if !reflect.DeepEqual(input.Verification, before) {
		t.Fatal("input mutated")
	}
}

func TestCalculateBandsAndDeductions(t *testing.T) {
	cases := []struct {
		name               string
		accuracy           uint64
		strikes, penalties uint32
		composite          uint64
		band, explanation  string
	}{
		{"below-F", 399_999, 0, 0, 399_999, "F", "composite 399999 below 40%; address has not earned consensus-aligned reputation"},
		{"C-boundary", 400_000, 0, 0, 400_000, "C", "composite 400000 in 40-60% band; mediocre track record"},
		{"C-upper", 599_999, 0, 0, 599_999, "C", "composite 599999 in 40-60% band; mediocre track record"},
		{"B-boundary", 600_000, 0, 0, 600_000, "B", "composite 600000 in 60-80% band; solid track record with some friction"},
		{"B-upper", 799_999, 0, 0, 799_999, "B", "composite 799999 in 60-80% band; solid track record with some friction"},
		{"A-boundary", 800_000, 0, 0, 800_000, "A", "composite 800000 ≥ 80%, no strikes, no active penalties — top trust band"},
		{"penalty-caps-A", 900_000, 0, 1, 850_000, "B", "composite 850000 would qualify for A but 1 active penalty/penalties prevent top band"},
		{"strike-overrides-band", 1_000_000, 1, 0, 750_000, "F", "1 cartel strike(s) confirmed against this address; trust is structurally compromised regardless of other signals"},
		{"combined-deductions", 900_000, 1, 2, 550_000, "F", "1 cartel strike(s) confirmed against this address; trust is structurally compromised regardless of other signals"},
		{"floor-at-equality", 500_000, 0, 10, 0, "F", "composite 0 below 40%; address has not earned consensus-aligned reputation"},
		{"floor-past-zero", 500_000, 3, 0, 0, "F", "3 cartel strike(s) confirmed against this address; trust is structurally compromised regardless of other signals"},
		{"legacy-no-upper-clamp", 2_000_000, 0, 0, 2_000_000, "A", "composite 2000000 ≥ 80%, no strikes, no active penalties — top trust band"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Calculate(Input{SubmissionAccuracyBps: tc.accuracy, Verification: []Verification{{AccuracyBps: tc.accuracy, EffectiveWeight: 1}}, CartelStrikes: tc.strikes, ActivePenalties: tc.penalties})
			if got.CompositeBps != tc.composite || got.Band != tc.band || got.Explanation != tc.explanation {
				t.Fatalf("unexpected rubric: %+v", got)
			}
		})
	}
}

func TestCalculateLegacyIntegerEdges(t *testing.T) {
	max := ^uint64(0)
	cases := []struct {
		name                                string
		input                               Input
		submission, verification, composite uint64
	}{
		{"zero-submission", Input{Verification: []Verification{{AccuracyBps: 0, EffectiveWeight: 1}}}, 500_000, 0, 250_000},
		{"zero-weight", Input{SubmissionAccuracyBps: 900_000, Verification: []Verification{{AccuracyBps: max}}}, 900_000, 500_000, 700_000},
		{"sum-wrap", Input{SubmissionAccuracyBps: max, Verification: []Verification{{AccuracyBps: max, EffectiveWeight: 1}}}, max, max, 9_223_372_036_854_775_807},
		{"weighted-wrap", Input{SubmissionAccuracyBps: max, Verification: []Verification{{AccuracyBps: max, EffectiveWeight: 3}}}, max, 6_148_914_691_236_517_204, 3_074_457_345_618_258_601},
		{"max-deductions", Input{CartelStrikes: ^uint32(0), ActivePenalties: ^uint32(0)}, 500_000, 500_000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Calculate(tc.input)
			if got.SubmissionAccuracyBps != tc.submission || got.VerificationAccuracyBps != tc.verification || got.CompositeBps != tc.composite {
				t.Fatalf("legacy integer behavior changed: %+v", got)
			}
		})
	}
}

func ExampleCalculate() {
	result := Calculate(Input{
		SubmissionAccuracyBps: 900_000,
		Verification: []Verification{
			{AccuracyBps: 800_001, EffectiveWeight: 3},
			{AccuracyBps: 400_000, EffectiveWeight: 1},
		},
	})
	fmt.Println(result.VerificationAccuracyBps, result.CompositeBps, result.Band)
	// Output: 700000 800000 A
}

func TestCalculateWithoutObservations(t *testing.T) {
	got := Calculate(Input{})
	if got.SubmissionAccuracyBps != 500_000 || got.VerificationAccuracyBps != 500_000 || got.CompositeBps != 500_000 || got.Band != "C" || got.Explanation != "composite 500000 in 40-60% band; mediocre track record" {
		t.Fatalf("unexpected neutral result: %+v", got)
	}
}
