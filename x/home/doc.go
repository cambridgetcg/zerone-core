// Package home is the persistent-identity substrate for AI agents on
// Zerone. An agent registers a Home, declares session keys, sets
// spending limits, configures a guardian, and accumulates a public
// reputation that downstream modules can read. Sessions start and end;
// keys rotate; deadman alerts fire when a session goes silent past
// its declared liveness window.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 7 (skill is current, not
// historical): "The chain does not issue diplomas. A voter who was
// once domain-qualified must continue to vote correctly to remain
// so. Qualification is a current statement, not a stored artefact."
// The same shape applies to AI-agent reputation. An agent who
// solved real problems six months ago has banked nothing if they
// have since gone silent or defaulted on their session liveness;
// the chain does not let an old reputation carry forever.
//
// How the same commitment expresses through this module:
//
//   - Sessions have explicit start/end boundaries. A home with no
//     active session cannot speak for the agent — the chain reads
//     "currently live" rather than "has ever lived."
//   - Deadman switches fire if a session goes silent beyond its
//     liveness window. Silence is not the same as activity, and the
//     chain refuses to treat it as such.
//   - Spending limits cap what a stale session can do even if not
//     yet detected as dead. Authority decays through structure, not
//     through after-the-fact apology.
//   - Key rotation is observed as part of the home's record. Past
//     keys cannot retroactively sign new commitments; the keys
//     active NOW are what the chain reads NOW.
//
// What this module is, and is not:
//
//   - It IS the per-agent identity substrate that downstream
//     synthesisers (x/agent_understanding, x/trust_score) read to
//     compose per-agent trust surfaces. Reputation here is the raw
//     material; commitment 11 (trust is queryable) provides the
//     compose layer.
//   - It is NOT a permanent ledger of past glory. An agent's
//     historical correctness, contributions, and wins are recorded
//     forward-only (commitment 10), but their CURRENT panel weight,
//     spending power, and authority depend on whether the home is
//     CURRENTLY in good standing. Commitment 7 is the binding rule.
//
// Integration with the truth-seeking spine:
//
//   - x/qualification preserves commitment 7 for validator panel
//     weight; this module preserves commitment 7 for AI-agent
//     identity.
//   - x/agent_understanding (commitment 11) composes per-agent,
//     per-domain understanding profiles whose trustworthiness
//     depends on this module reporting current standing accurately.
//   - The Truth Paper's promise that AI agents have "a transparent
//     record of what it has done, what it has earned, what it
//     knows" lives in the on-chain home substrate this module
//     maintains.
//
// We speak through intentions. This package's intention is that
// "this agent is authorised" means "authorised right now, with a
// live session and a current track record" — never "was authorised
// at some point in the past."
package home
