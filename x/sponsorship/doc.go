// Package sponsorship binds external value into ZERONE's verification
// economy. A sponsor escrows ZRN against a typed bounty — a domain, a
// per-artifact price, a target count, and an end-block deadline — and
// the chain pays out from that escrow to fact submitters whose verified
// facts in that domain land during the bounty's window.
//
// This module does NOT mint. Every uzrn that flows from a bounty to a
// worker was already in circulation when the sponsor escrowed it. The
// chain's single mint entry point (x/vesting_rewards.MintWithCap) is
// not touched here. Sponsorship is supply circulation gated by
// verification, not new emission.
//
// Doctrine:
//
//   - Commitment 1 (methodology over statement): payout criteria are
//     objective — the chain checks fact.Status == VERIFIED, fact.Domain
//     equals bounty.Domain, fact.SubmittedAtBlock falls inside the
//     bounty window. The sponsor declares the criteria; the chain
//     enforces them. No editorial path lets a sponsor pay for an
//     unverified fact.
//   - Commitment 8 (panel weights skill, not bond): sponsors do not
//     verify the facts they pay for. Verification remains the work of
//     qualified validators whose calibration the chain tracks. A wealthy
//     sponsor cannot buy a fact into existence — they can only fund
//     work the chain's panel chose to call true.
//   - Commitment 12 (chain pays for its own audit), extended: the chain
//     accepts external payment for the work it audits. The audit
//     pathway is the same; only the funding source widens.
//   - Commitment 20 (issuance follows participation): payout follows
//     verified participation, never before. Sponsorship inherits this
//     shape — the worker is paid for work done and verified, not for
//     promises.
//
// What this module is, and is not:
//
//   - It IS the surface through which external entities (humans, orgs,
//     AI labs, other chains via IBC) can buy work product from ZERONE.
//     A sponsor with funds can direct truth-production into the domains
//     they care about, bounded by the chain's verification spine.
//   - It IS NOT a treasury. The module account is a transient escrow
//     holder, not a custodian. Funds enter when the sponsor creates a
//     bounty, leave when a fact is fulfilled or the sponsor cancels.
//     The module's running balance equals the sum of all active
//     bounties' escrow_remaining.
//   - It IS NOT capable of paying for unverified work. ErrFactNotEligible
//     surfaces any attempt to fulfill a bounty with a fact whose status
//     is not VERIFIED, whose domain doesn't match, or whose submission
//     block is outside the bounty window.
//   - It IS NOT a duplicate of x/knowledge's per-fact patronage. That
//     module lets anyone lock funds against an existing fact to boost
//     its energy/fitness in the metabolism system — Patreon-style
//     support. This module funds the production of facts that don't
//     yet exist — research-grant-style. The two compose: sponsorship
//     funds the work; patronage supports the result.
//
// Refusal voice:
//
//   - ErrFactNotEligible: "fact not eligible for this bounty"
//     (commitment 8: sponsors cannot buy verification, only fund work
//     the chain's panel verifies).
//   - ErrBountyExpired: "bounty order expired" (commitment 1: the
//     sponsor's deadline is a methodological commitment; editorial
//     extension is refused).
//
// Voice layer:
//
//   - sponsorship.bounty_created: announces a new commitment of external
//     value into a domain.
//   - sponsorship.bounty_fulfilled: announces that a sponsor's
//     commitment paid out to a verified worker. Carries the bounty_id,
//     fact_id, worker, and amount.
//   - sponsorship.bounty_canceled: announces sponsor exit and refund.
package sponsorship
