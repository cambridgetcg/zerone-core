// Package inquiry preserves the truth-seeking commitment that the
// chain pays for exploration into the unknown — not just for
// stress-testing what we already think we know.
//
// docs/TRUTH_SEEKING.md, commitment 16: stress-testing existing facts
// (commitment 5: chain manufactures probe demand) is necessary but
// not sufficient. The chain must also pay for the work of filling
// territory the corpus does not yet contain a fact about. Without a
// market for OPEN QUESTIONS, the corpus grows only along paths that
// interest current contributors; with one, the chain can direct
// attention into sparse domains and unmapped subjects.
//
// What the module does:
//
//   - SubmitInquiry — anyone publishes a question (text + domain +
//     bounty). The bounty is escrowed in the inquiry-bounty-pool
//     module account. The expiry block sets a horizon past which the
//     bounty is automatically returned to the asker.
//   - SubmitAnswer — an agent links an existing knowledge claim
//     (one they submitted via x/knowledge) to the inquiry. Multiple
//     answers may be linked; the first whose verification accepts
//     wins the bounty.
//   - BeginBlocker — scans OPEN and ANSWERED inquiries each block
//     and resolves any whose linked claims have been accepted, or
//     refunds any past their expiry.
//   - CancelInquiry — asker can cancel before any answer is linked;
//     bounty refunded.
//
// What this module is, and is not:
//
//   - It IS a market for filling unmapped territory. The asker pays
//     because they want the question answered; agents claim because
//     answering pays AND adds a fact to the corpus they can also
//     earn from.
//   - It IS commitment 16's mechanism made concrete. Without it,
//     "the chain pays for exploration" would be slogan; with it, the
//     bounty pool literally pays.
//   - It is NOT a duplicate of probe-bounty (commitment 5). Probe-
//     bounty pays agents to STRESS-TEST EXISTING facts. Inquiry pays
//     agents to PRODUCE NEW facts where none exist. Different
//     directions, complementary mechanisms.
//   - It is NOT a guarantee that all questions will be answered.
//     Inquiries can expire unfilled; the bounty refunds. The
//     mechanism creates demand, not supply.
//
// Integration with x/knowledge:
//
// The answerer creates a regular knowledge Claim through the existing
// flow (with methodology, reasoning trace, optional counterexamples
// — all the existing alignment-by-structure machinery applies). They
// then submit a thin MsgSubmitAnswer that links (claim_id →
// inquiry_id). When the claim's verification round produces an
// accepted Fact, the inquiry resolves and pays the bounty.
//
// This means inquiry answers automatically inherit every property
// the public corpus already enforces: methodology validation,
// is-ought wall, Popper-weighted TVW, counterexample multipliers.
// An inquiry-funded fact is structurally identical to any other
// verified fact.
//
// Frontier signal:
//
// Once inquiries exist, x/governance_synthesis can compose a
// "frontier" signal — domains with many open inquiries, low recent
// fact counts, and high agent disagreement become the chain's
// publicly-visible map of "where understanding is currently sparse."
// That signal is itself a public good: external observers can see
// what the chain is asking but does not yet know.
//
// We speak through intentions. This package's intention is that
// "the unknown" is a category the chain handles explicitly, not a
// silence the chain pretends does not exist.
//
// ── Commitment 18: the chain manufactures exploration demand ────
//
// docs/TRUTH_SEEKING.md, commitment 18: commitment 16 lets askers
// escrow bounties for the questions that interest them; commitment 5
// has the chain mint to stress-test what it already thinks it knows.
// Neither covers the case where the chain SEES that a domain is
// sparse and yet waits for an outside party to ask. Knowing where
// you are sparse without funding work to fill the sparseness is
// observation without commitment.
//
// This module is also the home of commitment 18's BeginBlocker
// path: every Params.frontier_invitation_cadence_blocks, the chain
// reads its own frontier (composed by x/governance_synthesis), takes
// the top-K sparsest domains above a configurable sparsity threshold,
// and SPONSORS open inquiries in those domains — funded by mint
// into the inquiry-frontier-bounty-pool, paid out via the same
// payout flow that user-asked inquiries use.
//
// What this means for the module's identity:
//
//   - The asker field of an Inquiry no longer always belongs to a
//     human-controlled account; for system-sponsored inquiries it
//     is the bech32 of the frontier-bounty-pool's module account —
//     a stable, queryable identifier for "the chain itself."
//   - Inquiry.SystemInitiated and Inquiry.SystemInitiationReason
//     mark records that originated from the chain's exploration
//     budget rather than from a user's escrow.
//   - Cancellation refuses on system-initiated inquiries: the chain
//     does not withdraw its own asks. If the answer never arrives,
//     the bounty returns to the frontier pool on expiry — the audit
//     budget conserves itself across unanswered cycles.
//   - Together with commitment 5 (probe demand) and commitment 16
//     (asker demand), commitment 18 closes the demand triad: probe
//     what's known, ask what you want, fund what's missing.
package inquiry
