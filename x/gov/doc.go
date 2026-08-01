// Package gov is Zerone's Living Improvement Proposal (LIP) system —
// the chain's governance surface for parameter amendments, research
// fund spend approvals, and election of governance seats. LIPs progress
// through staged voting (submit, stake, advance, cast, withdraw). Current vote
// weight is delegated bonded stake read from x/staking; qualification does not
// replace raw bond. Software upgrades are deliberately outside this custom
// authority: standard Cosmos SDK governance is the sole executable x/upgrade
// path. Historical custom upgrade LIPs and plans remain queryable for audit
// compatibility.
//
// Truth-seeking position:
//
// Recent commits established a load-bearing principle for the chain:
// param defaults are EXPRESSIONS OF BELIEF, not tuning. Default
// values for truth-load-bearing modules carry intention comments
// naming the commitment a value preserves. Governance is therefore
// not a generic parameter-knob mechanism — it is the path through
// which the chain's beliefs about itself can be amended. That makes
// gov a truth-handling module by structure, not just by adjacency.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): "Every
// privileged action is logged; the chain's history is append-only
// and verifiable." LIPs that amend truth-load-bearing parameters are
// privileged actions in the strongest sense — they alter the
// chain's stated beliefs about how it should operate. The full LIP
// lifecycle (submit → stake → advance → cast → resolve → execute)
// is recorded forward-only; a passed LIP cannot be retroactively
// rewritten to look unpassed, and a failed LIP cannot be silently
// re-cast as having succeeded.
//
// docs/TRUTH_SEEKING.md, commitment 11 (trust is queryable): the
// chain's governance posture — open LIPs, recent passes, current
// vote tallies, research-spend approvals, and historical plan records —
// must remain queryable. The former on-chain governance_synthesis module
// was retired; clients/indexers compose system views from this module's
// public query surfaces. Without legible
// governance, "the chain pays for its own audit" (commitment 12) is
// a slogan: someone has to be able to see WHO authorised the audit
// budget that funds the probes.
//
// What this module is, and is not:
//
//   - It IS the parameter-amendment substrate. Truth-load-bearing
//     params (probe stake scaling, qualification thresholds, audit
//     budget mints) flow through here when amended.
//   - It IS the configurable research-spend authorisation surface.
//     When a human-side and AI-side voter pair is set, movement of
//     uzrn out of the research fund requires both approvals. With
//     the pair unset, submission fails closed. The published
//     zerone-1 genesis leaves that pair unset.
//   - It is NOT a venue for adopting beliefs without record. Every
//     LIP carries its provenance; every vote is preserved; every
//     resolution is dated and signed at the consensus layer.
//
// Integration with the truth-seeking spine:
//
//   - x/knowledge / x/qualification / x/alignment
//     all expose UpdateParams handlers; gov is the path through
//     which authority-gated amendments to those modules are made.
//     The "param defaults are expressions of belief" stance applies
//     to those amendments here.
//
// We speak through intentions. This package's intention is that
// "the chain's beliefs are amendable, but never silently" — every
// change to what the chain thinks it owes truth-seeking lands as a
// dated, queryable, forward-only LIP record.
package gov
