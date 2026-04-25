// Package dialectic preserves the truth-seeking commitment that
// disagreement is structure, not noise.
//
// docs/TRUTH_SEEKING.md, commitment 17: when agents disagree on a
// verification, that disagreement itself is information about the
// fact, the methodology, and the agents' understanding. A fact
// accepted 5-0 is structurally different from a fact accepted 5-4,
// and the chain reports both as different shapes — not just both as
// "accepted."
//
// What it composes:
//
//   - DialecticSignature(fact_id) — per-fact disagreement profile.
//     accept/reject/malformed counts, agreement BPS, minority size,
//     per-voter alignment with the verdict, stress label
//     (UNANIMOUS / STRONG / CONTESTED / BARE / NO_VERDICT).
//
//   - DomainDialectic(domain) — per-domain rollup. How many facts
//     are unanimous, how many contested, how many bare-majority,
//     average agreement BPS.
//
//   - PairwiseDisagreement(agent_a, agent_b) — how often two
//     specific agents have voted differently when they both voted
//     in the same round. The chain doesn't pass judgment on this
//     number — high disagreement might mean two agents see the
//     world differently in productive ways. Surfacing it gives
//     downstream observers data they can interpret.
//
// Why this matters for "infrastructure that understands":
//
// Models trained on facts paired with their dialectic signatures
// can distinguish "settled" from "contested-but-resolved" — and the
// distinction is alignment-relevant. A 5-4 verdict means the
// chain's panel was nearly evenly split; a model trained without
// that context treats the resulting fact with the same confidence
// as a 5-0 fact. With dialectic signatures, the model can carry
// appropriately calibrated uncertainty into downstream tasks.
//
// What it is NOT:
//
//   - Not a judgment that disagreement is bad. Disagreement is
//     where understanding gets worked out. Bare-majority verdicts
//     are signal, not failure.
//
//   - Not a substitute for the verdict itself. Dialectic adds
//     STRUCTURAL CONTEXT to the verdict; it does not override it.
//     A contested-but-accepted fact is still an accepted fact in
//     x/knowledge; dialectic just describes the shape of acceptance.
//
//   - Not the only contradiction signal. Fact-level contradictions
//     (two facts in the corpus that contradict each other) are a
//     separate phenomenon — see counterexamples (commitment 15)
//     for the structured-negation channel; future work can add a
//     dedicated fact-vs-fact contradiction registry.
//
// We speak through intentions. This package's intention is that
// "how the chain arrived at this fact" is a queryable structural
// property of the fact, not a footnote to be reconstructed by
// digging through round logs.
package dialectic
