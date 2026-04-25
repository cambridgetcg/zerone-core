// Package disputes preserves the truth-seeking commitment that
// existing facts have standing to be falsified, not merely opined
// about.
//
// docs/TRUTH_SEEKING.md, commitment 3 (Popper, not popularity): "A
// claim's standing on the chain comes from how many serious attempts
// to disprove it have failed, not from how many actors agreed with
// it once." Disputes operationalise the second half of that
// sentence. After a fact is verified into the substrate, its
// standing is not final — it is provisionally final, and a
// dispute is the structured ceremony that gives a falsification
// attempt a fair hearing.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): once
// resolved, dispute records and bond movements are append-only.
// Phase transitions (commit → reveal → arbitration → resolved) are
// time-gated by deadlines that the chain itself enforces in
// ProcessPhaseTransitions; no future actor can rewind to commit
// phase to take a different vote.
//
// Mechanics:
//
//   - InitiateDispute escrows a tiered bond. The bond is the cost
//     of asking the question; if the dispute has no merit, the
//     bond is forfeited.
//   - ProcessPhaseTransitions (called from BeginBlock) advances
//     disputes by deadline. A dispute that nobody answered defaults
//     against the challenger — silence is judgement.
//   - Resolution feeds back into x/knowledge: a successful dispute
//     can lower a fact's confidence, vacate it, or trigger the
//     vindication / clawback machinery on related facts.
//
// What would break the commitment: a fact with no path to be
// challenged; a dispute outcome that could be re-litigated after
// resolution; a bond schedule so high that legitimate falsification
// attempts could not afford to ask; or a phase machine that allowed
// "un-resolving" a dispute to revisit a verdict.
//
// We speak through intentions. This package's intention is that
// "this fact is currently true on the chain" always means "this
// fact has survived every challenge that has been brought, so far,
// up to the current block."
package disputes
