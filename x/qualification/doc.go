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
// SingleRefundEnabled selects the prospective custody repair. Native default
// genesis enables it; old genesis without single_refund_enabled imports as off,
// and export/import preserves the selector. Existing chains require the named
// consensus-repairs-v1 carrier, its census and coordinated activation height;
// merging this source does not activate an existing chain. The carrier owns the
// module version bump and must enable the marker before qualification BeginBlock.
// Marker reads use a fresh infinite gas meter, preserving pre-activation gas.
//
// With the marker on, successful stake expiry atomically refunds and leaves an
// EXPIRED record with an empty StakedAmount. A locked or failed refund instead
// leaves SUSPENDED with the amount and expiry/lock deadline intact, and clears
// ProbationUntil. SUSPENDED also retains its older probation/accuracy meaning:
// status alone is not refund provenance. The retained ExpiresAt and the paired
// stake_refund_pending then qualification_expired events surface unpaid expiry
// (see docs/EVENTS.md). BeginBlock does not retry; the owner can withdraw after
// the lock ends and the bank can pay. A missing bank is not a successful refund.
//
// Legacy EXPIRED stake records with a nonempty amount are ambiguous: expiry may
// already have paid, failed, or skipped a locked stake. After activation they
// remain held, refusing withdrawal and same-domain requalification, with no
// timeout, automatic refund, deletion, redistribution or clawback. A fixed-height
// census against module backing and a separately authorized reconciliation owner
// and path are required before activation. These holds do not replenish escrow;
// a historical deficit surfaces as transfer_failed for later expiries. ACTIVE,
// PROBATIONARY and SUSPENDED legacy records retain ordinary withdrawal. While
// the marker is off, legacy effects, including the double-refund defect, remain.
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
