// Package creed provides optional on-chain pins for the chain's source creed.
// `docs/TRUTH_SEEKING.md` is the human-facing creed. When pins have been
// adopted, this module records the current version and forward-only history.
// Published zerone-1 genesis contains no pin, so the module's presence is not
// evidence that a live chain has adopted the repository hash.
//
// Truth-seeking position:
//
// docs/TRUTH_SEEKING.md, commitment 19 (the creed is governance-
// gated): this module supplies append-only creed pins, an optional
// direct-anchor lockdown, amendment payload storage, and a council
// registry. Those are structural prerequisites, not proof that the
// stronger governance posture is active. Published zerone-1 leaves
// direct_anchor_enabled=true, does not configure the creed-amendment
// LIP category, and does not enforce separate human/AI quorum.
//
// docs/TRUTH_SEEKING.md, commitment 6 (no individual can unilaterally
// inject truth): the original commitment binds AT THE FACT LAYER.
// This module extends the same shape ONE LAYER UP — the chain's
// stated beliefs should not be silently amended. AnchorPin is gated
// by the SDK gov module authority and records append-only history.
// When direct anchoring is disabled, that handler seals completely
// and the separate gov-to-creed dispatch must be used. Two-pool
// human/AI tally is future work; council membership is currently a
// registry/query hook only.
//
// docs/TRUTH_SEEKING.md, commitment 10 (forward-only audit): pins
// are append-only by monotonic version. A new version archives the
// previous one — both remain queryable. A creed amendment cannot
// rewrite history to make a previously-pinned version look
// different now; it can only land a new version that supersedes.
// The chain's record of which creed it has stood on is part of
// its permanent audit trail.
//
// What this module is, and is not:
//
//   - It IS the append-only structural record for creed amendments.
//     The canonical TRUTH_SEEKING.md hash can be pinned on chain,
//     and CI checks the source file against the repository's
//     .creed-hash. That source hash is not evidence that a running
//     network has adopted the same pin.
//   - It IS a per-commitment registry when populated. CommitmentEntry can bind
//     each numbered commitment to amendment provenance; direct authority pins
//     may carry empty LIP IDs while direct anchoring remains enabled.
//   - It is NOT a replacement for the markdown creed itself. The
//     human-facing text remains in `docs/TRUTH_SEEKING.md`. This
//     module records WHICH version of that file the chain pins
//     to, not the text itself. The two are linked by the canonical
//     hash, which the off-chain `scripts/check_creed_hash.sh`
//     verifies pre-merge and the chain's own CI verifies pre-build.
//
// Integration with the truth-seeking spine:
//
//   - AnchorPin is authority-gated to the gov module account, and
//     direct_anchor_enabled defaults to true. The published
//     zerone-1 genesis also leaves it true.
//   - The CategoryCreedAmendment attachment and post-pass dispatch
//     exist, but the live gov category config does not let a newly
//     submitted amendment advance beyond draft. Ordinary LIP tally
//     also has no separate human/AI quorum.
//   - A future release may configure that category and set
//     direct_anchor_enabled=false. The source must not claim the
//     stronger posture until network state and tally code prove it.
//   - Off-chain indexers can compose creed-drift from the public
//     current/history queries; there is no on-chain system-level
//     governance synthesiser in the slim source.
//
// We speak through intentions. This package's intention is that
// "make every creed change explicit and forward-only, then activate
// governance constraints without overstating them" — the same audit
// discipline as commitment 6 applied one layer above the corpus.
//
// USEFUL_WORK doctrine (docs/USEFUL_WORK.md) — the third in the trio.
// One commitment (UW: ZERONE is recursive) + seven mechanisms (M1-M7)
// + six recursive axes (substrate / verification / classification /
// attribution / tooling / interface). Canonical Go-side registration
// in x/creed/types/useful_work_creed.go; cross-stack invariant harness
// in tests/cross_stack/useful_work_invariants_test.go.
//
// Phase 0 (this commit's vintage) ships zero behavioral bindings.
// Phase 1 introduces the x/work module that binds M1-M4, M5 shape, M7;
// Phase 2+ adds per-class registrations (knowledge migration,
// counterexamples, training-run attestation, eval-suite execution,
// dataset curation, alignment artifacts, RL traces, synthetic data,
// kernel optimization). M6 (recursion-amplified lineage) extends
// TC6 (Plan 4 of ToK series) cross-class.
//
// STRANGE_LOOP doctrine (docs/STRANGE_LOOP.md) — the fourth in the quartet.
// One commitment (SL: ZERONE is a strange loop) + six mechanisms (SL-M1
// through SL-M6). SL takes UW to its operational limit by nesting ZERONE
// into itself: doctrines, modules, governance, rewards, validators all
// produced/verified/rewarded through the chain's own machinery.
//
// Canonical Go-side registration in x/creed/types/strange_loop_creed.go
// + cross-doctrine echoes in x/creed/types/doctrine_echoes.go;
// cross-stack invariant harness in tests/cross_stack/strange_loop_
// invariants_test.go; genesis loader in x/knowledge/keeper/doctrine_
// genesis.go.
//
// Phase SL-α (this commit's vintage) binds SL-M1 (doctrine import):
// every commitment in every doctrine becomes a verified Fact in
// x/knowledge with domain=doctrine_*. Phases SL-β through SL-ζ bind
// the remaining five mechanisms (protocol as substrate, governance
// lift, author lineage, self-verification, origin attestation).
package creed
