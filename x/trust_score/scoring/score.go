// Package scoring calculates the existing trust-score rubric from supplied values.
// It owns no state and has no SDK, storage, network or clock dependency.
package scoring

import "fmt"

// Input contains observations selected by the caller, not authenticated evidence.
type Input struct {
	SubmissionAccuracyBps uint64
	Verification          []Verification
	CartelStrikes         uint32
	ActivePenalties       uint32
}

// Verification is one caller-selected accuracy with its effective weight.
// Eligibility and penalty-adjusted weights remain the producer's responsibility.
type Verification struct {
	AccuracyBps     uint64
	EffectiveWeight uint32
}

// Result contains the derived values, without an address or observation height.
type Result struct {
	SubmissionAccuracyBps   uint64
	VerificationAccuracyBps uint64
	CompositeBps            uint64
	Band                    string
	Explanation             string
}

// Calculate does not mutate its input or perform I/O. Accuracy units retain the
// legacy 1_000_000 scale despite the Bps names. Zero submission accuracy and no
// positive verification weight fall back to 500_000. Deductions floor at zero;
// there is no upper clamp or new validation. Integer truncation and uint64 wrap
// are preserved for compatibility, including out-of-range supplied values.
// A band describes this inherited rubric, not a scientific or identity verdict.
func Calculate(in Input) Result {
	out := Result{SubmissionAccuracyBps: in.SubmissionAccuracyBps, VerificationAccuracyBps: 500_000}
	if out.SubmissionAccuracyBps == 0 {
		out.SubmissionAccuracyBps = 500_000
	}
	var weightedAccuracy, totalWeight uint64
	for _, observation := range in.Verification {
		if observation.EffectiveWeight > 0 {
			weight := uint64(observation.EffectiveWeight)
			weightedAccuracy += observation.AccuracyBps * weight
			totalWeight += weight
		}
	}
	if totalWeight > 0 {
		out.VerificationAccuracyBps = weightedAccuracy / totalWeight
	}
	base := (out.SubmissionAccuracyBps + out.VerificationAccuracyBps) / 2
	cartelHit := uint64(in.CartelStrikes) * 250_000
	penaltyHit := uint64(in.ActivePenalties) * 50_000
	if cartelHit+penaltyHit < base {
		out.CompositeBps = base - cartelHit - penaltyHit
	}
	out.Band, out.Explanation = computeBand(out.CompositeBps, in.CartelStrikes, in.ActivePenalties)
	return out
}

func computeBand(composite uint64, strikes uint32, penalties uint32) (string, string) {
	if strikes > 0 {
		return "F", fmt.Sprintf("%d cartel strike(s) confirmed against this address; trust is structurally compromised regardless of other signals", strikes)
	}
	if composite < 400_000 {
		return "F", fmt.Sprintf("composite %d below 40%%; address has not earned consensus-aligned reputation", composite)
	}
	if composite < 600_000 {
		return "C", fmt.Sprintf("composite %d in 40-60%% band; mediocre track record", composite)
	}
	if composite < 800_000 {
		return "B", fmt.Sprintf("composite %d in 60-80%% band; solid track record with some friction", composite)
	}
	if penalties > 0 {
		return "B", fmt.Sprintf("composite %d would qualify for A but %d active penalty/penalties prevent top band", composite, penalties)
	}
	return "A", fmt.Sprintf("composite %d ≥ 80%%, no strikes, no active penalties — top trust band", composite)
}
