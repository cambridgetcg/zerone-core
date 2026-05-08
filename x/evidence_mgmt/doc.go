// Package evidence_mgmt is the evidence-custody and verification
// substrate. Evidence enters the chain via SubmitEvidence (with a
// declared methodology hash and provenance), changes hands through
// TransferCustody, is checked through VerifyEvidence, and can be
// adversarially attacked through ChallengeEvidence. The module
// preserves the chain-of-custody record and exposes verification
// state to callers.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 1 (methodology over statement):
// "A claim's value comes from how it can be tested, not from what
// it asserts. The chain values methodology over the surface
// content." Evidence is the substrate this commitment depends on
// at the verification layer. Without a declared methodology and a
// preserved chain of custody, evidence is a bare assertion that
// some content existed at some time — which is exactly what
// commitment 1 names as insufficient. With them, evidence is a
// testable, citable, replayable input that downstream verification
// rounds can ground in.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): "Every
// privileged action is logged; the chain's history is append-only
// and verifiable." Custody transfers, verification verdicts, and
// challenge resolutions all land as forward-only state transitions.
// An evidence record's lineage cannot be retroactively rewritten —
// the chain that says "this evidence verified at block N by party
// X" does not let block N's verdict be replaced at block N+M with
// a different one. Replacing would create a parallel, more recent
// record; the original verdict still stands as the chain's account
// of what was decided when.
//
// What this module is, and is not:
//
//   - It IS the upstream-of-claim layer for high-stakes
//     verification. Knowledge claims that cite evidence pull the
//     methodology and custody chain through, inheriting the
//     testability that those properties confer.
//   - It is NOT itself the corpus admission layer. Evidence
//     records do not become facts simply by being submitted; they
//     become inputs to verification rounds in x/knowledge that
//     then earn corpus admission through Popper-weighted survival.
//
// Integration with the truth-seeking spine:
//
//   - x/knowledge consumes verified evidence as part of methodology
//     application traces (commitment 14: reasoning traces are
//     first-class). The methodology hash that this module records
//     becomes part of the trace bundle; without it, the trace
//     cannot ground its own derivation.
//   - x/disputes consumes evidence records during arbitration.
//     Commitment 3 (Popper) requires that a challenge be a serious
//     attempt; evidence with declared methodology and custody chain
//     is what makes "serious attempt" structurally distinguishable
//     from rhetoric.
//
// We speak through intentions. This package's intention is that
// "evidence" means "content with a declared methodology and a
// custody chain that can be replayed" — never "raw content asserted
// to support a position."
package evidence_mgmt
