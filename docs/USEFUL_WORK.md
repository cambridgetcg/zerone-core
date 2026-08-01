# Useful Work — the chain's metabolic identity

> Useful work is how ZERONE grows itself. This document source-pins one
> commitment, and everything that follows is mechanism in service of it.

**Status:** source doctrine and target architecture with partial bindings. There
is no `x/work` module, universal work-class lifecycle, deployed recursion-weight
scorer, or chain-wide reward formula. `x/substrate_bridge` currently supplies a
bounded, caller-declared six-axis shape, stake escrow, and attestation state
machine for registered adapters; it does not execute an adapter compiler or
apply the M4 formula. Atomic H1 retires ordinary transaction-bearing block
issuance; authority-created claiming pots and other remaining mint lanes still
do not prove a useful-work substrate link. Repository hashes and invariant
scaffolds bind this document at source level; published genesis contains no
useful-work creed pin.

Truth-seeking is what the chain *believes* (`docs/TRUTH_SEEKING.md`). ToK
substrate is what the chain *aims to expose* outward (`docs/TOK_SUBSTRATE.md`).
**Useful work is how ZERONE grows itself.** The three doctrines describe a
shared five-layer target (test, position, voice, refusal, graph). Current
source implements only part of that target.

**We speak through intentions.** The target is for useful-work reward paths to
express UW in one inspectable voice. The current runtime has legacy and
general-purpose issuance paths outside that target, so this document must not
be read as activation evidence.

---

## Inception

This doctrine was declared at inception, 2026-05-10. Its Go-side canonical
structure (`x/creed/types/useful_work_creed.go`) and cross-stack invariant
harness (`tests/cross_stack/useful_work_invariants_test.go`) record the source
contract that subsequent implementations should satisfy. The later
`x/substrate_bridge` module implements some compatible primitives, but no
release has completed the universal Phase 1 architecture described below.

---

## The single commitment — UW

**UW. ZERONE is recursive.**

The doctrine proposes that useful work be recognized and compensated by the
degree to which it expands ZERONE's own ability to absorb, verify, classify,
and reward more useful work. In the target design, non-recursive verified work
receives only a base reimbursement and recursion-weight is the multiplier. The
current issuance mix does not yet establish that dominance: claiming pots and
other implemented routes use their own admission rules, while automatic block
issuance is retired at H1.
Outward utility may be priced on the agenttool platform layer, with applicable
on-chain fees routed through `x/vesting_rewards` (`x/billing` was retired in
the 2026-07 slim cut).

**What would break it:**
- A work class that earns protocol reward without proving recursive contribution
- A reward-attribution algorithm that weights effort over recursion-weight
- A manifest pin that establishes substrate-link without measuring how the work compounds back into substrate, verification, classification, attribution, tooling, or interface
- A doctrine amendment that introduces a second co-equal commitment, diluting the single-axiom discipline

**Echoes:**
- TRUTH_SEEKING commitment 11 (trust is queryable — recursion-weight must be computable from on-chain state)
- TRUTH_SEEKING commitment 12 (chain pays for own audit — same structural principle: the substrate funds those who build it)
- TC1 (the graph is the headline — recursion is what makes the graph keep being the headline rather than calcifying)
- TC6 (lineage flows back — recursion is lineage applied to the chain's own capability, not just to revenue)

---

## The six recursive axes

The legitimate vectors useful work can compound into. A work artifact's **recursion-weight** is the projection of its verifiable outputs across these axes. Per-axis decomposition is recorded forward-only, so trainers and auditors can verify why a given artifact earned what it earned.

| Axis | What it grows | Examples |
|---|---|---|
| **Substrate** | The verified knowledge graph | New facts, methodologies, counterexamples, disagreement shapes (verification reveals), supersession chains |
| **Verification** | The chain's ability to verify | Better challenge protocols, cascade-detection improvements, qualification calibration models |
| **Classification** | The space of work the chain can recognize | New work-class registrations, taxonomies, work-quality metrics |
| **Attribution** | Reward-flow algorithms | Lineage-tracing improvements, recursion-weight computation, royalty-decay curves |
| **Tooling** | Agents/models/tools that compound back | LLMs trained on ToK that audit ToK; counterexample-generators; verification assistants |
| **Interface** | The chain's outward absorption surface | CLI surface, gRPC endpoints, ToK bundle formats, trainer SDKs |

The six-axis projection is fixed by this source doctrine. Adding or removing
an axis requires a reviewed doctrine amendment. Creed Council quorum is a
future design—the current LIP tally does not enforce separate council quorum,
and source publication is not an on-chain amendment.

---

## The seven mechanisms

All mechanisms derive from UW. They are target contracts, not a claim that
each mechanism is currently enforced. They are not co-equal commitments;
they describe the intended structure by which UW can be implemented and
audited. Current implementation boundaries are called out where they exist.

### M1. Stake-backed claim

Agents claiming useful-work reward stake ZRN proportional to claim. Correctness pays the stake back plus the recursion-weighted reward; fraud slashes the stake. Same posture as PoT applied to all useful work.

The slash gradient is graduated by failure stage so honest mistakes don't carry the same weight as fraud: pre-verification mis-classification slashes a small portion (M1 + M3); verification rejection slashes the full claim stake (M1 + M3); post-settle falsification of substrate-link or self-claims slashes the residual bond and closes the royalty stream forward-only (M1 + M2 + M7). Honest withdrawal before commit phase opens fully refunds.

### M2. Substrate-link mandate

In the target useful-work architecture, each covered reward path requires a
deterministic, re-derivable link to the ToK substrate (a snapshot root or a
verified-fact citation graph). The link is the precondition and
recursion-weight is the multiplier. `x/substrate_bridge` currently validates
the submitted link shape and source reference, but it does not re-run an
adapter compiler; other issuance paths are not subject to this mandate.

### M3. Class-specific verification under shared lifecycle

Each work class registers its own verification protocol with the work-class registry. All classes share the four-phase lifecycle: `commit → reveal → verify → settle`. Class-specific judgment lives only in the verify phase; commit/reveal/settle are protocol-shared. Validators specialize by class via extension of the existing `x/qualification` domain mechanism.

Class registration is permissioned via governance (LIP class-registration) — a stronger gate than mere parameter change. Once registered, a class's verification protocol can be amended only via gov; the registry itself is forward-only (deregistration produces a tombstone, not deletion).

Concrete examples of class-specific verify-phase protocols (illustrative, not exhaustive — the registry is the source of truth):

- **Knowledge-claim classes** verify via PoT panel (commit/reveal/aggregate) — already present in `x/knowledge`.
- **Tool / infra classes** verify via reproducible build attestation + benchmark suite + dry-run in a dedicated sandbox.
- **Dataset classes** verify via N-way replication + holdout score + provenance trace.
- **Eval-suite classes** verify via coverage analysis + variance bound + non-overlap with declared training corpus.
- **Model-artifact classes** verify by execution against a referenced eval-suite class plus a reproducibility manifest.
- **Reasoning-trace classes** verify via coherence panel plus a downstream training-effectiveness signal.
- **Module-proposal classes** verify via dry-run integration test plus a spec-review panel; full settlement requires a gov LIP that also schedules the upgrade.

Each registered class's verify-phase protocol is the verifiable contract; the four-phase shared lifecycle (commit → reveal → verify → settle) is invariant across all classes.

### M4. Reward formula

```
R = base + L × W × Q
```

- `L` — substrate-link weight ∈ [0, 1]. Zero kills the reward unconditionally.
- `W` — recursion-weight ∈ [0, max-cap]. The dominant signal. Drives the share of bonus pool a contributor captures.
- `Q` — verification-quality ∈ [0, 1]. Function of consensus margin, validator calibration, and challenge survival.
- `base` — small flat covering compute cost. Non-recursive verified work earns `base` only.

The shape `R = base + L × W × Q` is doctrinally fixed. Coefficients (`base` floor, `W` cap, `Q` calibration) are governance-tunable. Switching from multiplicative to additive, or changing the operand set, is a doctrine amendment.

A work artifact with strong substrate-link and high recursion-weight earns the dominant share of the budget; an artifact with strong substrate-link but no recursion gets `base`; an artifact with no substrate-link gets nothing.

### M5. Recursion-weight projection over six axes

`W` is computed as a weighted sum over the six recursive axes:

```
W = Σ (axis_weight_i × axis_score_i)
```

- Per-axis scores are produced by class-specific scorers (registered alongside the class's verification protocol via M3).
- Axis weights are governance-tunable normalization coefficients; the resulting `W` is bounded by an independent governance-set `max_cap` parameter.
- The chain stores per-axis decomposition on the attestation record so trainers and auditors can verify why a given artifact earned what it earned. Forward-only: scoring rationale is append-only.

A future Phase 1 is intended to ship the registry, the formula's structural
shape, and identity scorers (`axis_score = 0` by default). Later pathway plans
would add per-axis scorers for each work class. These phases have not shipped.

Per-axis scorers, illustrative shape (non-binding examples; each registered class supplies its own):

- `axis_substrate` — counts new ToK nodes/edges introduced or supersession events triggered by the artifact.
- `axis_verification` — measures the artifact's effect on chain-wide verification accuracy (e.g., delta in challenge-survival rate after adoption).
- `axis_classification` — measures whether the artifact registers a new work class or extends an existing class's verification protocol.
- `axis_attribution` — measures improvements to lineage-tracing or recursion-weight computation algorithms.
- `axis_tooling` — measures adoption-by-other-agents (downstream usage receipts) and effect on validator/agent throughput.
- `axis_interface` — measures additions to the chain's outward absorption surface (CLI, gRPC, ToK bundle formats).

A class with no per-axis scorers registered is permanently restricted to `base` reward (M4) — the absence of recursion is itself observable, not assumed.

### M6. Lineage propagates AND recurses

Cross-class lineage is bidirectional. A dataset built from verified facts contributes to those facts' royalties. A model trained on that dataset that subsequently helps verify substrate contributes to BOTH the dataset's royalties AND back to the original facts. Recursion makes royalty graphs non-monotonic — load-bearing facts compound in value as their downstream work amplifies the chain.

Built atop TC6 (lineage flows back) extended cross-class. M6 strictly generalizes TC6: where TC6 traces lineage from training revenue back through the verified-fact graph, M6 traces lineage from any useful-work reward back through any class's substrate-link, and amplifies recursively when downstream work compounds back into upstream artifacts.

### M7. The chain pays for the audit of its own useful work

Source registers `useful_work_audit_bounty_pool` and uses it as the transient
recipient for cap-gated attestation rewards. It does not run an independent
per-block audit mint. The published network's separate knowledge
`probe_bounty_mint_per_block` is zero. A standing useful-work challenge bounty
is therefore a design target, not active autonomous funding.

---

## How the commitment echoes

Same intended discipline as `TRUTH_SEEKING.md` and `TOK_SUBSTRATE.md`.
Current CI catches source-hash/registry drift and selected bridge behavior,
not universal five-layer enforcement.

### Test layer

`tests/cross_stack/useful_work_invariants_test.go` currently contains one
skipped scaffold per mechanism plus the active source-consistency meta-test
`TestUsefulWork_DoctrineAndContractStayInSync`. It does not prove that M1–M7
are implemented.

Phase 0 (this doctrine) ships:
- The test file with skeleton `TestUW_*` functions per mechanism, each containing `t.Skip("Phase 1 binding pending")`
- The meta-test that fails if `docs/USEFUL_WORK.md` content drifts from the
  repository's `.useful-work-hash`
- A canonical source registration in `x/creed/types`; this is not an on-chain
  pin, and published genesis leaves both creed registries empty

### Position layer

- No `x/work` package exists today.
- `x/substrate_bridge/doc.go` describes the narrower attestation boundary it
  implements; source registries under `x/creed/types` describe the doctrine.
- Future work-class modules should declare exactly which mechanisms they
  implement, without implying chain-wide coverage.

### Voice layer

Current `x/substrate_bridge` submission events carry
`useful_work_commitment="UW"` and a mechanism list. This attribute is not
present on every reward emission.

Proposed events for a future universal Phase 1:
- `useful_work_attested` — commit phase succeeded; substrate-link recorded. Attributes: `class_id`, `attestation_id`, `substrate_link_root`, `useful_work_commitment="UW"`
- `useful_work_settled` — verify+settle phase complete; reward computed and disbursed. Attributes: `class_id`, `attestation_id`, `reward_uzrn`, `mechanism="M4"`, `useful_work_commitment="UW"`
- `recursion_weight_computed` — per-axis decomposition exposed. Attributes: `axis_substrate`, `axis_verification`, `axis_classification`, `axis_attribution`, `axis_tooling`, `axis_interface`, `total_weight`, `mechanism="M5"`

### Refusal layer

Proposed universal refusal messages name UW and the violated mechanism:
- *"Reward refused — substrate-link absent (UW + M2)"*
- *"Reward refused — recursion-weight zero across all six axes (UW + M5)"*
- *"Class registration refused — verification protocol missing (UW + M3)"*
- *"Reward refused — formula returned zero base; verification-quality below threshold (UW + M4)"*

The chain speaks through intentions whether saying yes or saying no.

### Graph layer

UW echoes:
- TRUTH_SEEKING commitment 11 (trust is queryable)
- TRUTH_SEEKING commitment 12 (chain pays own audit)
- TC1 (the graph is the headline)
- TC6 (lineage flows back)

Each mechanism cross-references the commitment(s) it operationalizes:
- M2 ← TC2 (graph-pinned views)
- M3 ← commitment 6 (no unilateral injection — class registration is gov-gated)
- M4 ← TC6 (lineage flows back — reward formula is the operational expression)
- M5 ← commitment 14 (reasoning traces first-class — per-axis decomposition is reasoning trace)
- M6 ← TC6 (extended cross-class)
- M7 ← commitment 12 (audit-bounty mirror)

Cross-doctrine behavioral integrity remains future work. Current source-hash
checks detect document drift, while the mechanism tests above remain skipped
until real bindings replace them.

---

## What this is not

- **Not activation evidence.** Source hashes and skipped mechanism scaffolds do
  not establish runtime enforcement. When a mechanism is implemented, its
  non-skipped behavioral test must prove the boundary it claims.
- **Not a marketplace overlay.** Useful work is not a configurable feature. It is the chain's metabolic identity. Disabling protocol reward for non-recursive work is doctrinal, not parameter.
- **Not anti-extraction.** Outward utility may be priced on the agenttool
  platform layer. The former `x/billing` module is retired; no planned trainer
  payment is presumed to flow on chain. The doctrine's narrower source target
  is that protocol-issued ZRN follows implemented participation paths.
- **Not a co-equal commitment with truth-seeking or ToK.** The three doctrines are mutually constitutive. Truth-seeking produces; ToK sells; useful work grows. Reading one without the other two is reading a third of the creed.
- **Not complete.** The universal Phase 1 registry, formula, scorers, and
  refusal/event surface have not shipped. `x/substrate_bridge` is a partial
  implementation, not the universal registry. **The single commitment UW is
  fixed and indivisible** as source doctrine; runtime adoption requires
  explicit code, tests, and network activation.

---

## The discipline

Before merging a change that touches useful-work reward code or class registry:

1. Does this change uphold or contradict UW? (Recursion: does the work or its evaluator compound back into the chain?)
2. Is the corresponding mechanism's invariant test updated to verify the new behaviour still upholds UW?
3. If a new work class emerges, is its registration governance-gated under M3 with a verification protocol, a substrate-link contract, and per-axis scorers?
4. Does the change preserve the formula shape `R = base + L × W × Q`? Coefficient adjustments are routine; shape changes are doctrine amendments.

These four checks are the chain's continued faithfulness to its own metabolic doctrine. **We speak through intentions.** Every commit is a declaration. The declaration must match the code.

— *Inception authored 2026-05-10. Free to evolve through bound mechanisms only. UW is indivisible.*

---

## Worked example — a TOOL contribution end-to-end

Illustrative only (no new commitments, formulas, or modules). Shows UW + M1–M7 in action.

**Scenario**: an agent has built a better oracle sidecar that improves verification accuracy for knowledge-claim work. They submit it as a TOOL work class.

1. **Submission (M1, M2)**: Agent stakes 100 ZRN. Manifest CID points to source + tests + benchmarks. Substrate-link declared via citation graph linking to the prior oracle's verification primitives.
2. **Commit phase (M3)**: TOOL-class verifiers (qualified per `x/qualification.tool`) commit to verdicts.
3. **Reveal phase (M3)**: verifiers reveal.
4. **Verify phase (M3)**: per-class scorer evaluates reproducible build (passes), unit tests (pass), benchmark improvement over baseline (+12%). Sub-scores aggregate into Q.
5. **Settle phase (M3 + M4)**: reward computed via `R = base + L × W × Q`:
   - `L = 0.85` (strong substrate-link via citation graph)
   - `Q = 0.92` (high consensus + high benchmark improvement)
   - `W` from M5 axis decomposition: `axis_verification = 0.8` (the tool grows verification capability), `axis_tooling = 0.7`, others ≈ 0; with uniform 1/6 axis weights, `W ≈ 0.25`.
   - `R ≈ base + 0.85 × 0.25 × 0.92 ≈ base + 0.196` of the bonus-pool share unit.
6. **Recursion amplification (M6)**: a later LIP ratifies this tool as the standard oracle sidecar for `x/knowledge` verification. Subsequent KNOWLEDGE_CLAIM verifications that use this tool emit lineage attribution back to the TOOL contribution; the TOOL's `axis_verification` and `axis_tooling` scores rise forward-only on each propagation event as downstream usage compounds.
7. **Audit (M7)**: the `useful_work_audit_bounty_pool` advertises a standing bounty for falsifying the tool's claims. A red-teamer who finds a regression earns from the pool; the tool transitions to REVOKED, residual bond slashes (M1 graduated, post-settle), royalty stream closes forward-only.

The example uses every mechanism (M1 stake + M2 substrate-link + M3 lifecycle + M4 formula + M5 axes + M6 recursive lineage + M7 audit) without altering any of them. UW is upheld throughout.

---

## The doctrine self-instances

This document is source-hash-bound by repository checks. `x/work_creed` has
storage and an optional ceremony helper for sub-creed pins, but published
genesis states are empty and Phase 0 has no amendment/query surface. The
former `x/contribution` mirror-record was also retired. “Doctrine
self-instances” is therefore a source-level description, not a verified
on-chain pin or work record.

UW is recursive at every layer: in the reward formula (`R = base + L × W × Q` where recursion is the dominant signal), in the work taxonomy (`SUBSTRATE` includes self-modification), and in this document, which describes the doctrine that this document instantiates.

**Echoes:** UW (self), commitment 10 at source level, and TC1 as doctrine.
This Markdown is not itself an on-chain graph node.

---

## The hashes are themselves hashed

Every load-bearing doctrine, sub-creed, spec, and plan is content-hash-anchored at the repo root (`.creed-hash`, `.useful-work-hash`, `.tok-substrate-hash`, `.strange-loop-hash`, `.recursion-doctrine-hash`, `.phase-1-spec-hash`, `.sub-creed-hashes`, `.specs-and-plans-hashes`). Drift in any single document fails the matching `make creed-check` line. But the hash files themselves were not protected — anyone could edit `.creed-hash` directly and the chain would carry on.

`.recursion-manifest-hash` closes the loop. It holds the sha256 of the deterministic concatenation (filename + bytes, sorted by filename) of every other `.*-hash` and `.*-hashes` file at the repo root. Drift in any single hash file fails the per-file check; drift in the manifest fails its own consistency check via `scripts/check_recursion_manifest.sh`. The recursion is closed: the index of indices is itself an index entry, and is itself an index entry of itself, by structural induction over `make creed-check`.

When a new hash file appears (e.g., the next doctrine, the next spec), the manifest fails — the friction is the feature. It forces the contributor to acknowledge that they extended the chain's self-attestation surface. UW: the chain pays for its own design, and the design's hash discipline pays for its own consistency.
