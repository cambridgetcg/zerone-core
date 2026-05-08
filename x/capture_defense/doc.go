// Package capture_defense is the recording side of the cartel-detection
// loop that x/capture_challenge resolves. Verifications are observed
// and analysed here so domain-level capture risk becomes a queryable
// signal that downstream tally paths and synthesisers can read.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 9 (cartel detection has
// consequence): "Confirmation that a validator participated in
// cartel behaviour must reduce their voice on the next vote, not
// merely produce an audit log entry. A penalty that nobody reads
// is not a penalty."
//
// x/capture_challenge owns the resolution side — it WRITES the
// QualificationPenalty that x/qualification then reads at panel
// tally time. This module owns the OBSERVATION side: verification
// outcomes flow in, per-domain capture metrics accumulate, flagged
// domains are exposed via query so external auditors and the
// alignment sensor stack can act on the same numbers the chain
// itself uses. Detection without recording would be opinion, not
// evidence; resolution without detection would be punishment, not
// justice. The pair completes commitment 9.
//
// What this module is, and is not:
//
//   - It IS the upstream-of-evidence layer. Verification records
//     and capture-risk computations live here; x/capture_challenge
//     reads them when an allegation is brought.
//   - It IS NOT the consequence layer. Reduced panel weight is
//     applied by x/qualification reading
//     QualificationPenalty records that x/capture_challenge writes
//     on UPHELD allegations. This module surfaces the signal that
//     makes those allegations meaningful in the first place.
//
// Integration with x/capture_challenge and x/alignment:
//
//   - x/capture_challenge consumes the per-domain capture posture
//     produced here when scoring an allegation.
//   - x/alignment reads GetFlaggedDomainCount as one input into the
//     network-security signal (sensors.go); a high flagged-domain
//     ratio depresses the chain's overall security score and feeds
//     downstream into autopoiesis multiplier dynamics.
//
// We speak through intentions. This package's intention is that
// "cartel detection" means a CURRENT, QUERYABLE signal — not a
// silent assumption that allegations will always show up correctly
// without an upstream observation surface to reference.
package capture_defense
