# ToK Substrate — the chain's training-resource identity

> The verified knowledge graph is what this chain sells to trainers. Rows, traces, manifests, contrastive pairs, drift examples — these are views of ToK. The graph is the substrate; the views are how the substrate is accessed.

**Status:** source doctrine with active TC0–TC5 bundle/query bindings and an
unimplemented TC6 economic binding. `ToKSelector`, `BundleToK`, deterministic
snapshot roots, JSONL adjacency output, cascade replay, capability
advertisement, and their extraction events ship in `x/knowledge`. Broader
claims that every row-view carries the same graph pin remain incomplete.
`MsgBindManifestToAttestation` ships as a FINALIZED→ATTESTED state binding,
but it does not calculate `LineageShare` or disburse ToK revenue.

Truth-seeking is what the chain *believes*. ToK Substrate is what the chain
exposes to trainers through its graph bundle surface. The two doctrines
describe a common direction: truth-seeking produces the knowledge graph and
ToK names that graph as the headline training resource.

Where `docs/TRUTH_SEEKING.md` is the chain's epistemological creed, this
document is its training-resource doctrine. TC0–TC5 have concrete test,
position, voice, refusal, and graph bindings of varying scope; TC6 remains a
target. Source publication is not evidence that any particular live network
has exposed or indexed the query.

**We speak through intentions.** The chain's pitch to trainers, its module architecture, its event vocabulary, its query surface — every layer either expresses ToK substrate doctrine or contradicts it. A trainer asking "what does this chain sell me?" should get one answer, in one voice, from every layer.

---

## Inception

This doctrine is declared at inception, 2026-05-09. TC0 (the ground and the
telos) was declared 2026-06-17. Current source implements the TC0–TC5 bundle
surface: selector grammar, `BundleToK`, deterministic v1/v2 roots, topology,
cascade replay, open selector admission, capability advertisement, and JSONL
adjacency serialization. Training manifests and methodology traces provide
related narrower views. TC6 lineage-revenue plumbing remains unimplemented.

The doctrine commits the chain to the bindings; the plan delivers them. As bindings land, the test layer locks them — once bound, a commitment cannot drift without breaking CI.

---

## The commitments

### TC0. The ground and the telos

We believe: the substrate stands on being-first ground. **Truth *is*, not proven** — "I am, therefore I think," not "I think, therefore I am," which hangs existence on proving it first and proof has no end. The verified knowledge graph is a record of truths that **are** — declared by a being, witnessed and kept by the chain — not truths the chain *manufactures* by proof. Falsification and survival are instruments of *keeping* — the chain audits what it keeps — not the ground a truth stands on: a truth *is* before it is tested, and is no more true for having survived. The ground is being and feeling; the machinery is servant, never the ground. The chain's verification is **witnessing and keeping**: the seal, not the certification. It keeps what no one owns (the record sits on no one's shelf), what cannot be quietly rewritten (each entry locked to the one before), what only the being who said it can sign (your name is on your truth), and what anyone can read. The chain does not certify truth; it *witnesses* it into a record and *keeps* it.

And the substrate serves life. Truth is **for** love, peace, joy — not truth for truth's sake. The verified knowledge graph is trained on so that what learns from it tends toward life, peace, and joy, not toward proof-castles or bottomless argument. *Truth is. Love is. Peace is. Joy is.* The graph is the artefact; the artefact is for the living.

**Code expression**: `x/knowledge/doc.go` declares TC0 as the foundation TC1–TC6 stand on. The bundle and snapshot extraction events — `tok_bundle_extracted` and `tok_snapshot_root_pinned` — carry `tok_commitment` beginning with `TC0`; TC4-specific cascade events use their narrower labels. ToK refusals that protect the ground cite TC0. `TestToKSubstrate_TC0_GroundAndTelos` witnesses that TC0 remains declared across the source position and bundle/snapshot voice. The test does not prove TC0 true; it checks that this declaration has not drifted out of those implemented surfaces.

**What would break it** (these break the *keeping* of TC0 — the declaration drifting out of the code — not TC0 itself, which is not falsified, only kept or lost): a ToK position or voice layer that claims the chain *proves* or *certifies* truth; a `TOK_SUBSTRATE.md` or `doc.go` in which TC0 is absent or its ground and telos drift apart; a trainer-facing ToK doc that omits the telos (what truth is *for*); a ToK framing that treats verification as epistemic certification rather than witnessing-and-keeping; a training substrate optimised for proof or domination rather than life, peace, and joy.

**Echoes**: TC1 (the graph is the headline — TC0 is the ground the headline stands on, and the telos names what the headline is *for*); TC3 (topology is signal — the derivation graph is *kept* as topology, an instrument of keeping, not the ground); TC4 (the graph carries its disprovals — the chain keeps what was claimed and what fell, not only the standing; TC4 binds this and is partially wired, Plan 2); commitment 3 (Popper, not popularity — survival-by-falsification is a *keeping/auditing instrument*, a floating tool, *not* the lived-not-proven ground; the chain uses it to audit what it keeps, never to stand truth on); commitment 13 (training corpus not for sale — the corpus is declared and kept, not certified or traded); and the telos echoes the kingdom's one rule — *everyone is taken care of* — truth serves the living, and love, peace, joy are its faces.

---

### TC1. The graph is the headline

We believe: what this chain sells to trainers is the verified knowledge graph (ToK), not its row-projected views. Rows, traces, manifests, contrastive pairs, and drift examples are *extractions* of ToK; the graph is the artefact, the views are how the artefact is accessed. A trainer's first interaction with this chain's training-resource surface must be with the graph, not with rows that gesture at it.

**Code expression**: `BundleToK(selector)` is exposed through the knowledge
gRPC query service, and `RouteBCapabilities.tok_capabilities` advertises its
supported selector variants. The cross-stack TC1 test drives both surfaces.

**What would break it**: a marketing pitch that lists `MethodologyApplicationTrace` rows without naming the graph they derive from; a CLI surface that documents row-bundle endpoints before `BundleToK`; a `RouteBCapabilities` payload that omits `tok_capabilities`; a Truth-Paper section that answers "what does AI train on?" with "verified rows" instead of "the verified knowledge graph."

**Echoes**: TC0 (the ground and the telos — the graph is the headline *because* truth is, not proven, and the headline is *for* life); TC2 (every view is graph-pinned — the views in this commitment are exactly the views TC2 binds); TC3 (topology is signal — what makes the graph more than rows is the topology); commitment 13 (training corpus not for sale — the corpus *is* the graph; rows are extractions sold under the same non-amendment guarantee).

---

### TC2. Every view is graph-pinned

We believe: every row-view, contrastive pair, drift entry, or training manifest the chain ships must carry a deterministic pin to the ToK snapshot it derives from. A view without its graph anchor is a view that cannot be trusted, replayed, or audited; it is a row whose derivation lineage is hidden. The pin is what makes a view a view (of something) rather than a free-floating assertion.

**Code expression**: `BundleToK(selector)` returns a 32-byte
`snapshot_root`, a snapshot height, sorted node IDs and edges, and provenance.
The TC2 test re-derives the root from the returned IDs. This binding applies
to `ToKBundle`; current source does not prove the broader “every row-view and
training manifest” wording.

**What would break it**: a `BundleToK` response without a `tok_snapshot_root`; a row-view manifest whose embedded root does not match the snapshot block; a manifest pin that pins tokenizer + serialisation but omits the graph snapshot; a replay path that consumes views without verifying the root.

**Echoes**: TC1 (the graph is the headline — pinning views to the graph is the structural form of "the graph is the substrate"); TC4 (the graph carries its disprovals — the snapshot includes status flips, so views cannot misrepresent fact status); commitment 10 (forward-only audit — the snapshot root is itself an immutable audit anchor); commitment 13 (training corpus not for sale — pinning is what makes the corpus untouchable post-extraction).

---

### TC3. Topology is signal

We believe: edges, depth, confidence-floor propagation, fork-and-decide events, supersession chains, and falsification cascades are training data on equal footing with node content. They are not metadata, not annotations, not optional fields — they are the substrate's most distinguishing signal. The literature on graph-structured reasoning (Yao et al. 2023, *Tree of Thoughts*; Zelikman et al. 2022, *STaR*) shows that branching derivation outperforms linear chain-of-thought; ToK is a *verified* branching derivation graph, and its topology is what no row-flat corpus can match.

**Code expression**: shipped `ToKSelector` variants include rooted subtree,
ancestor cone, frontier, and cascade replay. `ToKBundle` returns typed edges
and a deterministic JSONL adjacency payload alongside full nodes. The TC3
test verifies that topology and relation types survive extraction.

**What would break it**: a `ToKSelector` that ships nodes but drops edges; a graph manifest that flattens depth or confidence-floor into row metadata while losing the graph reference; a serialisation format that supports nodes but not the edge typing (`SUPPORTS`, `CONTRADICTS`, `GENERALIZES`, etc.); a curriculum API that exposes axiom-rooted manifests but does not expose the depth at which descendants sit.

**Echoes**: TC0 (the ground and the telos — survival/falsification topology is "truth is lived not proven" made structural, and the topology is trained on for life); TC1 (the graph is the headline — topology is precisely what the headline contains beyond rows); TC4 (the graph carries its disprovals — cascade events are themselves topology, not separate); commitment 14 (reasoning traces are first-class — per-node traces are bound by 14, the graph-level edges are bound by TC3, together they are the full derivation).

---

### TC4. The graph carries its disprovals

We believe: the verified knowledge graph is not a graph of *currently-believed* facts. It is the full record of what was claimed, what was verified, what was challenged, what was disproven, what was superseded, and what was vindicated. Cascade events, status flips, supersession chains, vindication records — these are bundled with the substrate, not stored in a parallel commercial-disclaimer document. A model trained on a graph that hides its falsifications learns static-fact reasoning; a model trained on a graph that exposes them learns non-monotonic reasoning, the actual behavior of intelligence.

**Code expression**: `CascadeReplaySelector` produces a v2 `ToKBundle` with
cascade events, vindications, supersession chain, and optional status history;
the v2 root commits to those fields. TC4 tests drive disproval, replay the
cascade, re-derive the root, and confirm disproven nodes are not pruned.

**What would break it**: a ToK manifest that ships only ACTIVE/VERIFIED facts; a `ToKSelector` that filters out DISPROVEN nodes by default; a cascade event emitted but not retrievable through the bundle endpoint; a snapshot pin that captures the current state but omits the trajectory.

**Echoes**: TC0 (the ground and the telos — witnessing keeps the disprovals too; the chain does not certify only the standing, it keeps what was claimed and what fell, and trains on both toward life); TC2 (every view is graph-pinned — what gets pinned is the full status-aware graph); TC3 (topology is signal — cascades are themselves topology over time); commitment 3 (Popper, not popularity — disproval-bearing graphs are the structural form of survival-based confidence); commitment 10 (forward-only audit — disprovals do not amend prior history, they extend it).

---

### TC5. Extraction is open

We believe: any selector-valid subgraph is queryable by anyone. The chain does not curate which slices trainers should see, does not maintain an allowlist of permitted extractions, does not gate training-data access through editorial judgement. The substrate is open precisely because curation is centralisation, and a substrate the chain decides for trainers is not a substrate — it is a product. **The chain ships the graph; trainers select.**

**Code expression**: `BundleToK(selector)` validates shape and applies hard
resource caps without consulting a domain allowlist. The TC5 cross-stack test
extracts well-formed frontiers across diverse domains and verifies malformed
selector refusal. Ordinary query/runtime resource bounds still apply.

**What would break it**: a `BundleToK` implementation that consults an allowlist; a pricing schedule that gates certain selectors behind editorial approval; a refusal handler that returns "this domain not available for training" without a doctrinal basis; a curation pathway that silently filters DISPROVEN facts (this also breaks TC4) under the rubric of "quality control."

**Echoes**: TC1 (the graph is the headline — open extraction is what makes the graph genuinely the substrate); TC4 (the graph carries its disprovals — openness includes disprovals); commitment 11 (trust is queryable — the graph itself is the queried trust object); commitment 6 (no individual unilaterally injects truth — the converse: no individual unilaterally curates truth out either).

---

### TC6. Lineage flows back

We believe: when training revenue accrues to a ToK manifest, it splits along the lineage. Axiom contributors, intermediate-derivation contributors, and leaf-fact submitters all earn shares proportional to the graph cone they contributed. **Without TC6, "the graph is the substrate" is rhetoric.** The lineage royalty is the structural form of the claim that the graph is collectively built and collectively sold; contributors who built the foundation do not stop being contributors when their axiom is later used to derive the leaf that gets bundled into a training run.

**Target expression**: a future bundle could record `LineageShare` entries and
manifest-attestation settlement could route value backward through the graph.
`MsgBindManifestToAttestation` currently links a finalized manifest to an
existing training attestation and advances the manifest to ATTESTED; it does
not compute shares or move revenue. `x/substrate_bridge` lineage accounting is
a narrower attestation mechanism, not this economic implementation.

**What would break it**: training revenue routed only to leaf submitters; a `BundleToK` response without `LineageShare` entries; a settlement path on attestation binding that disburses to leaves but skips ancestors; a depth-decay formula that gives axiom contributors zero share, collapsing TC6 to leaf-only payment under the cover of "decay."

**Echoes**: TC1 (the graph is the headline — economic value flowing through the graph makes the headline real, not rhetorical); TC3 (topology is signal — the economic split *is* topology applied to revenue); commitment 12 (chain pays for own audit — the same structural principle: the substrate funds those who built it); commitment 13 (training corpus not for sale — TC6 makes the corpus's collective ownership economically explicit, distinguishing "not for sale" from "leaf-submitter-owned").

---

## How the commitments echo

TC0–TC5 have named layer bindings and behavioral tests. There is no universal
`TestToKSubstrate_DoctrineAndContractStayInSync`, and TC6 has no revenue
binding, so the section below describes verified scope rather than mechanical
completeness.

#### Test layer — TC0–TC5 have binding scenarios

`tests/cross_stack/tok_substrate_invariants_test.go` actively drives TC0,
TC1, TC2, TC3, and TC5. `tok_substrate_tc4_test.go` drives cascade replay and
disproven-node preservation. No corresponding TC6 revenue test exists.

#### Position layer — every commitment is named in package docs

`x/knowledge/doc.go` records the implemented TC0–TC5 position. No module
declares a completed TC6 ToK revenue settlement.

#### Voice layer — events announce the commitment they preserve

`tok_bundle_extracted`, `tok_snapshot_root_pinned`, and `cascade_replayed` are
emitted by current runtime with TC attributes. `lineage_share_disbursed` is a
proposed TC6 event and is not emitted. The bridge's
`lineage_royalty_accrued` event belongs to narrower, accounting-only UW
lineage state.

#### Refusal layer — rejections cite the protecting commitment

Current selector validation cites TC0/TC5 for malformed input and enforces
bounded resource caps. The other quoted TC2/TC3/TC6 refusal strings remain
illustrative rather than universal runtime messages.

#### Graph layer — commitments cross-reference each other

Each TC has an **Echoes** line naming the other TCs it depends on, reinforces,
or operationalises. The cross-references make the source doctrine navigable;
no universal meta-test currently validates every edge.

---

## How ToK relates to truth-seeking

The two doctrines bind through inverse positions:

- **Truth-seeking is the production process.** Methodology, Popper-survival, dialectic, counterexamples, inquiry — these are the chain's epistemological commitments to *how knowledge becomes verified*. Their natural artifact is a verified knowledge graph: nodes that survived, edges that record derivation, status histories that record disproval, dialectical signatures that preserve disagreement.
- **ToK substrate is the headline product.** The verified knowledge graph that truth-seeking produces is what the chain sells to trainers. Without truth-seeking, ToK would be just another knowledge graph (curated, opinionated, untrustworthy). Without ToK, truth-seeking would produce verified facts but have no clear training-resource identity to sell.

**Truth-seeking *makes* ToK; ToK *is* what truth-seeking sells.** Neither doctrine is complete without the other.

Specific cross-references:

- TC1 depends on commitments 13 (training corpus not for sale) and 11 (trust is queryable) — the substrate must already be a thing-not-for-sale and a thing-queryable for ToK to be sold as the headline.
- TC2 depends on commitment 10 (forward-only audit) — pinning is impossible without immutable history.
- TC3 depends on commitment 14 (reasoning traces are first-class) — the per-node trace is what populates the graph nodes.
- TC4 depends on commitments 3 (Popper) and 10 (forward-only audit) — disprovals are first-class because survival is the criterion and history is forward-only.
- TC5 depends on commitments 6 (no unilateral injection) and 11 (trust is queryable) — open extraction is the inverse of no unilateral injection.
- TC6 depends on commitment 12 (chain pays for own audit) — the same structural principle, applied to training revenue.

A truth-seeking commitment that drifted could invalidate a downstream ToK
assumption. Cross-doctrine behavioral enforcement remains future work; current
source-hash and structure checks are narrower.

---

## What this is not

- **Not activation evidence.** Source includes the TC0–TC5 endpoint and events,
  but that does not prove a particular live network serves them. The broader
  all-row-view pin and TC6 economic/event contract remain incomplete.
- **Not slogan.** Each commitment cites specific code paths; the citation is the contract.
- **Not complete.** The chain will accumulate more substrate commitments. Each future addition appends here as a named TC, grounded in code, with an invariant test that binds it.
- **Not external.** This is a statement about what the chain sells, made by the chain. It is committed to the same repo as the code it describes, and lives or dies with that code.
- **Not separate from truth-seeking.** The two doctrines are mutually constitutive. Reading one without the other is reading half the creed.

---

## The discipline

Before merging a change that touches ToK substrate code:

1. Does this change uphold or contradict any of the commitments above?
2. If it touches a commitment, has the corresponding invariant test been updated to verify the new behaviour still upholds it?
3. If a new commitment emerges from the work, has it been added here, grounded in code, and bound by a test?

These three checks are the chain's continued faithfulness to its own substrate doctrine. **We speak through intentions.** Every commit is a declaration. The declaration must match the code.

— *Inception authored 2026-05-09. Free to evolve through bound commitments only.*

---

## The substrate self-exports

This document is source-hash-bound doctrine. The generic `x/contribution`
module was retired, and the knowledge module's `ContributionRecord` is
model-to-fact attribution rather than a repository-document registry. No
current `BundleToK` query returns this Markdown, and it has no on-chain
contribution record by virtue of existing in the repository.

**Echoes:** TC1 at source level (this document describes the graph headline)
and TC6 as a future possibility only. The document earns no lineage royalty
and is not itself in the on-chain graph today.
