// Package autopoiesis preserves the truth-seeking commitment that the
// chain pays for its own audit out of its own block-by-block budget,
// not by waiting for a victim to fund the investigation.
//
// docs/TRUTH_SEEKING.md, commitment 12 (chain pays for its own audit):
// "Audit costs are paid out of the chain's own emission, not by the
// person who got harmed." The probe bounty pool exists in x/knowledge
// and is funded by per-block mints; this module is the regulator that
// decides how aggressive that minting should be in light of current
// system state.
//
// docs/TRUTH_SEEKING.md, commitment 5 (chain manufactures probe
// demand): probes do not appear because the chain hopes for skeptics
// — the chain pays for skepticism. Autopoiesis adjusts the multipliers
// that gate the audit-budget machinery so that issuance scales with
// SSI (Smoothed System Index): when participation drops or
// verification stalls, audit funding rises to compensate.
//
// Mechanics:
//
//   - CollectAndAdapt runs at each EpochBoundaryBlocks tick. It
//     gathers staking participation, verification rate, and other
//     module-supplied metrics; runs them through an EWMA damper; and
//     produces an SSI score with oscillation detection.
//   - Multiplier targets are adjusted toward the SSI signal, bounded
//     by per-step magnitude caps. The chain is allowed to react, not
//     to whipsaw.
//   - Emergency halts (consumed via the alignment_adapters wiring)
//     pause adaptation. We do not adapt while broken; we adapt as the
//     remediation we believe in.
//
// What would break the commitment: an epoch tick that ran but mutated
// no rates; a multiplier change that ignored the SSI signal; or a
// schedule that issued audit funding even while the chain itself was
// halted (which would be paying for audits of state we have explicitly
// declared we don't trust).
//
// We speak through intentions. This package's intention is that the
// chain's metabolism — its rate of self-correction — is set by the
// chain itself, not configured by an external operator.
package autopoiesis
