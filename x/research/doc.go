// Package research is where research submissions, peer review, and
// bounty-funded research production live. Submissions are challenged
// and reviewed adversarially, bounties are escrowed by funders and
// paid to whoever fulfils them, and resolved research lands as
// citation material that knowledge claims can build on.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 13 (the training corpus is not
// for sale): "What enters the corpus enters because it survived;
// what survives must continue to earn its place every block." The
// research module is one of the upstream pathways into the corpus,
// and its bounty mechanic is precisely the kind of channel that
// commitment 13 protects against. A funder who could buy a
// pre-decided conclusion would silently corrupt the corpus through
// the back door even if the front door (x/knowledge) holds.
//
// What survives the back door:
//
//   - Bounties pay for the WORK of producing research that meets a
//     stated brief; they do not pay for a particular conclusion.
//     The fulfilment must still pass peer review (ReviewResearch)
//     and adversarial challenge (ChallengeResearch). A funder
//     cannot pre-pay a verdict — only an attempt that the chain's
//     own review machinery then judges on its own terms.
//   - Resolved research that downstream knowledge claims cite
//     inherits the same Popper-weighted TVW machinery as any other
//     fact. Earmarked funding does not raise the survival bar; the
//     chain still demands corroboration earned by withstanding
//     challenge.
//   - Bounty fulfilment is auditable: who funded, who claimed, who
//     fulfilled, what review verdict — all on chain. Commitment 13
//     is meaningful only when the routes into the corpus are
//     legible; this module makes its route legible by design.
//
// What this module is, and is not:
//
//   - It IS a market for directing research effort toward stated
//     questions. Funders express demand by escrowing bounties;
//     researchers express supply by claiming and fulfilling.
//   - It is NOT a market for buying corpus content. Research
//     submissions still pass through review and (when relevant)
//     adversarial challenge before any downstream training-value
//     attaches. The funding pathway is upstream of corpus
//     admission, not parallel to it.
//
// Integration with x/knowledge:
//
// Resolved research provides citation material that knowledge claims
// reference; nothing in this module bypasses the Popper-weighted
// corroboration counter, the methodology requirement, or the
// is-ought wall. The funder paid for an attempt; the chain still
// decides whether the attempt earns a place.
//
// We speak through intentions. This package's intention is that
// "funded research" means "the chain paid for someone to attempt a
// research question" — never "the chain paid for a conclusion to
// enter the corpus."
package research
