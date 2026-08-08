// Package gov currently implements Zerone's legacy Living Improvement Proposal
// records, custom-stake ballots, research-fund phases and seats, and related
// compatibility queries.
//
// It is separate from Cosmos SDK x/gov. Standard SDK governance is already the
// sole executable software-upgrade authority and the application authority for
// typed privileged messages. The custom parameter router is not a reliable
// execution boundary: it is created without registered target handlers, and
// legacy LIP resolution can record passage independently of target mutation.
//
// The accepted target architecture in docs/AUTHORITATIVE-STATE.md keeps SDK
// x/gov as the sole ordinary proposal, tally, decision, and atomic execution
// kernel. This package's records remain queryable as history, while its
// submission, stake, ballot, seat, phase, treasury, parameter, and execution
// paths are to be retired by a named migration. Until that migration is
// implemented and activated, this package still describes current consensus-
// committed application state and must not be presented as already read-only.
//
// Governance power in the accepted target is a non-transferable,
// proposal-snapshotted controller unit. Money, SDK stake, custom stake, KARMA,
// verifier rewards, tier, and reputation do not scale policy power. Source
// publication alone creates no electorate and changes no running chain.
package gov
