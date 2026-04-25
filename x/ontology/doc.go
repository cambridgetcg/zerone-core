// Package ontology preserves the truth-seeking commitment that the
// chain refuses to confuse what is with what ought to be.
//
// docs/TRUTH_SEEKING.md, commitment 2 (is-ought wall): "Empirical
// claims and normative claims must be representable but not
// interchangeable. A normative commitment ID does not earn training
// weight as if it were an observation." This module enforces the
// wall structurally, not by convention. Domains carry a Stratum
// (empirical / normative / synthetic / etc.); GetStratumPropsForDomain
// returns the confidence ceiling and decay rate that downstream
// verification reads. A normatively-classified domain cannot route
// itself into the training weight pipeline by changing its label.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): domain
// proposals expire if not promoted; lifecycle transitions
// (active → deprecated → archived) are forward-only; archived
// domains cannot be revived. The taxonomy of what we are willing to
// consider knowledge about is itself an append-only history.
//
// Mechanics:
//
//   - GetStratumPropsForDomain (domains.go) is the gate read by
//     x/knowledge when scoring confidence and by x/training_provenance
//     when computing TVW. It returns the stratum's MaxConfidence cap
//     so empirical and normative claims are handled differently.
//   - Domain proposals run through proposals.go with submission,
//     evidence, and resolution phases. Expiry is the silent default
//     when a proposal does not gather support.
//   - Depth constraints (Godel constraint) prevent infinite ontology
//     trees — a child domain cannot exceed configured depth, so the
//     stratum classification near the root cannot be obscured by
//     burying claims under arbitrary sub-domains.
//
// What would break the commitment: a domain that could be reclassified
// from normative to empirical without an audit record; a stratum
// confidence cap that defaulted to "no cap" for unknown strata; or a
// proposal pipeline that allowed retroactive activation of an archived
// domain (which would let old normatively-classified facts re-enter
// the empirical training corpus).
//
// We speak through intentions. This package's intention is that the
// chain's ontology is a public, dated, irrevocable map of which kinds
// of claims are eligible to count as which kinds of evidence.
package ontology
