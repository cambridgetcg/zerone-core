// Package qualification retains historical qualification records and their
// custody lifecycle. The commitments below describe its legacy design; panel
// agreement is not a measurement of truth or independent competence.
//
// docs/TRUTH_SEEKING.md, commitment 7: "The chain does not issue
// diplomas. A voter who was once domain-qualified must continue to
// vote correctly to remain so."
//
// docs/TRUTH_SEEKING.md, commitment 8: "A wealthy validator who has
// not shown they can tell truth from falsehood must not dominate the
// panel. Stake alone is not skill."
//
// docs/TRUTH_SEEKING.md, commitment 9: "Confirmation that a validator
// participated in cartel behaviour must reduce their voice on the
// next vote, not merely produce an audit log entry."
//
// Current-source boundary: this package also exposes a stake pathway and holds
// its own qualification escrow. The accepted target in
// docs/AUTHORITATIVE-STATE.md makes qualification non-economic, retires that
// pathway, reconciles its locked balances as legacy claims, and pins every
// qualification to a canonical ontology revision. That migration is not yet
// implemented or activated.
//
// After knowledge-review-neutrality-v1, the knowledge-owned policy stops outcome
// metrics, track-record grants, accuracy decay and score-based probation
// promotion. Existing records remain available; ordinary expiry, endorsement
// expiry and stake withdrawal continue. New policy-1 knowledge panels ignore
// these qualification weights. Legacy pending financial terms remain separate.
//
// Legacy contracts retained for historical execution and inspection:
//
//   - GetQualificationWeight returns 0 for non-ACTIVE qualifications
//     and applies QualificationPenalty deductions to ACTIVE ones; the
//     legacy panel reads through this method, so a non-ACTIVE or penalised
//     validator carries reduced or zero weight at the next vote.
//   - RunAccuracyDecay scans qualifications periodically and
//     transitions ACTIVE → PROBATIONARY → SUSPENDED based on
//     AccuracyBps thresholds; recovery is bidirectional. The status
//     transitions are the consequence layer of the feedback loop.
//   - QualificationPenalty records written by capture_challenge
//     resolutions are read by GetQualificationWeight automatically;
//     retained legacy consumers use this method; it grants no policy-1 weight.
package qualification
