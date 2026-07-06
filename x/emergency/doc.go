// Package emergency preserves the truth-seeking commitment that the
// chain admits when it is in trouble, and that admission itself is
// recorded forever.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): "Every
// privileged action must be appended to a log that no future actor
// can rewrite." Halts, reverts, and resumes are the most privileged
// actions the chain can take — they pause normal commerce, claw back
// state, or restart it. Each one is a ceremony with a recorded
// supermajority signature, and once resolved, the ceremony is
// immutable.
//
// docs/TRUTH_SEEKING.md, commitment 4 (substrate stress-tests its
// truth): a chain that is halting because it is broken should not
// continue stress-testing the broken state and producing junk audit
// results. IsHalted is the gate: alignment and the challenge engines
// consult it. Stress-testing only runs against state we believe is
// sane enough to test.
//
// Mechanics:
//
//   - CreateHaltCeremony / CreateRevertCeremony / CreateResumeCeremony
//     each open a guardian vote with prevote and precommit phases. A
//     supermajority is required to advance; the votes themselves are
//     part of the immutable record.
//   - MaxPauseDurationBlocks bounds how long a halt can hold; the
//     chain auto-resumes past that to prevent halts from being used
//     as a denial-of-service. (See x/knowledge param of the same
//     name for the partner setting.)
//   - IsHalted is the read-only contract that other modules consume
//     via alignment_adapters / gov_adapters. The contract is one
//     boolean; the discipline is that everyone respects it.
//
// What would break the commitment: a halt that left no record, a
// revert that allowed re-writing of pre-revert facts, an emergency
// power that bypassed the ceremony, or modules that ignored
// IsHalted and kept running their stress-tests against a state the
// chain itself had declared unsafe.
//
// We speak through intentions. This package's intention is that
// "the chain has admitted it is in trouble" is a queryable, dated,
// signed fact — and that everyone who depends on the chain can see
// it the moment it becomes true.
package emergency
