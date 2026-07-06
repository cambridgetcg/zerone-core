// Package alignment preserves the truth-seeking commitment that the
// chain senses its own state and corrects course before degradation
// becomes catastrophe.
//
// docs/TRUTH_SEEKING.md, commitment 11 (per-system scope): "Trust must
// be queryable, synthesised from many signals." Where x/trust_score
// answers per-address and x/governance_synthesis answers per-chain,
// this module is the upstream observation layer — the sensors whose
// readings those queries depend on.
//
// docs/TRUTH_SEEKING.md, commitment 12 (chain pays for its own audit):
// the alignment correction pipeline records suggested corrections in
// response to network state. When sensing detects that knowledge
// quality is decaying, consuming modules read alignment's pacing
// multipliers (GetGlobalPacingMultiplier) and slow down the flows
// that depend on healthy verification — a feedback loop that pays
// for its own auditing budget by throttling growth when the
// substrate is stressed.
//
// Mechanics:
//
//   - Periodic ObserveAll cycles run sensor.go and produce a composite
//     score across knowledge quality, economic stability, governance
//     participation, network security, and staking ratio.
//   - GenerateCorrections turns the score into immutable correction
//     records. Corrections are append-only; we do not retroactively
//     un-correct.
//   - Pacing multipliers expose alignment's current stance to other
//     modules. Reading the multiplier is the contract; the value
//     itself is the truth-seeking signal.
//
// What would break the commitment: a sensor that fired but produced no
// queryable result; an alignment that observed but never adjusted; or
// adjustments that moved past corrections rather than appending new
// ones. The package's IsHalted gate (alignment/module.go) also enforces
// commitment 4 indirectly — a halted chain does not pretend its
// observations are still informative.
//
// We speak through intentions. This package's intention is that the
// chain's awareness of itself is itself a public artefact, queryable
// in the same way as the facts it verifies.
package alignment
