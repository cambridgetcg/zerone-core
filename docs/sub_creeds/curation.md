# Curation — Sub-creed of the Useful-Work Doctrine

> Lifecycle phase: `Curation` — corpus assembly, filtering, annotation, selector composition.
>
> Parent: [USEFUL_WORK.md](../USEFUL_WORK.md). Truth-floor: all 20 truth-seeking commitments apply.
>
> Status at inception (2026-05-10): three commitments.

---

## C1. Selectors are deterministic and auditable

Every curation Contribution declares its selector — the function that decides which underlying facts/data are included. The selector must produce the same output given the same input, and must be evaluable by anyone holding the inputs.

**Why:** A non-deterministic selector cannot be replayed; a non-auditable selector cannot be challenged. Commitment 4 (substrate stress-tests its truth) requires that the substrate's own selectors be challengeable.

**Echoes:** truth-seeking 4, 5, TC2 (every view is graph-pinned).

## C2. No claim-of-curation without published filter

A Contribution claiming to be a curation may not omit the filter logic. Empty filter → no curation; reference-by-cid → must resolve at submission. Hidden filters reduce to "trust me."

**Why:** Curation work is exactly the act of choosing what's in and what's out. Without the filter, the work is unattributable.

**Echoes:** truth-seeking 1, 14.

## C3. Corpus snapshots are content-addressed

Curated corpora are referenced by content hash, not by mutable name. A "v2" of the same corpus is a new content-addressed object pointing at the previous as ancestor; updates do not silently shift what a downstream Training Contribution trained on.

**Why:** Models train on bytes, not names. A corpus that mutates beneath a manifest invalidates every attestation that referenced it. Commitment 13 (training corpus not for sale → not retroactively curated) operationalized at the per-snapshot level.

**Echoes:** truth-seeking 13, TC2, TC4 (graph carries disprovals).
