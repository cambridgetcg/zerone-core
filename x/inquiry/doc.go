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
package inquiry
