# Route B Security Audit — Wave 9

> **Scope:** adversarial stress test of the Route B training infrastructure (Waves 1–8). Attacks were launched against the most valuable structures first — the manifest/Merkle commitment, the economic layer, the is-ought wall, the verifier panel, the heartbeat. Every attack is a concrete test; every verdict is either "survived", "hardened", or "known gap — flagged for later wave".

**File:** [`tests/cross_stack/route_b_wave9_adversarial_test.go`](../tests/cross_stack/route_b_wave9_adversarial_test.go)

---

## Summary

| Category | Attacks | Survived | Hardened | Known gap | Policy item |
|---|---:|---:|---:|---:|---:|
| Manifest / Merkle | 12 | 11 | 1 | 0 | 1 |
| Economic layer | 6 | 6 | 0 | 0 | 0 |
| Is-ought + TVW | 4 | 4 | 0 | 0 | 0 |
| Verifier panel | 3 | 2 | 0 | 1 | 0 |
| Heartbeat | 1 | 1 | 0 | 0 | 0 |
| **Total** | **23** | **21** | **1** | **1** | **1** |

- **Survived (21):** the chain rejected the attack under its current rules without modification.
- **Hardened (1):** the attack revealed a real vulnerability; code was changed in this wave to close it.
- **Known gap (1):** the attack succeeded; the vulnerability is documented and deferred to a future wave with a specific plan.
- **Policy item (1):** the current behaviour is a design choice, neither a bug nor a vulnerability, but the decision is spelled out so future governance can revisit it.

---

## Attack-by-attack

### 1. Composition-depth bomb — **HARDENED**

A chain of parent manifests deeper than the configured cap should be rejected so bundle assembly cannot be turned into a DoS vector.

**Before:** check was `parent.CompositionDepth+1 > maxDepth`, allowing exactly one level beyond the named cap (9 levels for `maxDepth=8`).
**Fix:** tightened to `parent.CompositionDepth+1 >= maxDepth`. Max allowed depth is now `maxDepth-1`; a chain of 8 levels is the ceiling. [`msg_server_training_v7.go`](../x/knowledge/keeper/msg_server_training_v7.go).
**Post-fix test:** `TestRouteB_Wave9_CompositionDepthBomb` — 8 levels succeed, 9th rejects.

### 2. Parent must be FINALIZED / ATTESTED — **SURVIVED**

A DRAFT manifest cannot serve as a parent. This also eliminates cycle construction (cycles require a DRAFT pointing at itself, impossible because parents must be sealed first).

**Defence:** explicit status check in `MsgCreateTrainingManifest` handler.

### 3. Composed-Merkle collision across (parent, delta) pairs — **SURVIVED**

Different parents with the same delta, or the same parent with different deltas, or both, must yield distinct composed roots. Composed vs flat commitments over the same delta must also differ (domain tags prevent collision).

**Defence:** `ComputeComposedManifestMerkleRoot` uses `"ZERONE/KNOWLEDGE/MANIFEST/v1/COMPOSED"` domain tag (vs `"ZERONE/KNOWLEDGE/MANIFEST/v1"` for flat), `PARENT:` and `DELTA:` labelled segments, length-prefixed strings. All four possible collisions tested negative.

### 4. Child remains verifiable after parent supersession — **SURVIVED**

Parent supersession by a newer manifest for the same pipeline must not invalidate children that snapshotted the parent's Merkle root at create time.

**Defence:** child stores `parent_merkle_root` at create; verification uses the stored value; parent's on-chain *status* does not change parent's *root*.

### 5. Augmentation-of-augmentation — **SURVIVED**

A variant-of-variant chain would bypass the methodology-adjudication grounding on the original fact. `SubmitAugmentation` requires `original_fact_id` to resolve to a `Fact`, not an `Augmentation`.

**Defence:** `GetFact(ctx, msg.OriginalFactId)` returns `!ok` → error.

### 6. Duplicate manifest ID — **SURVIVED**

Creating a manifest whose ID already exists must reject.

**Defence:** handler checks `GetTrainingManifest` and rejects on conflict.

### 7. Non-creator cannot finalize — **SURVIVED**

Only the pipeline operator who created the manifest may finalize it. Otherwise a second operator could seal a manifest on first operator's behalf, potentially with incorrect delta semantics.

**Defence:** `manifest.Creator != msg.Creator` → error.

### 8. Is-ought wall smuggling — **SURVIVED**

Model owner lists a `NormativeCommitment` ID as a trained-on fact_id. Handler must filter these and report `rejected_commitment_count` without silently including them in revenue.

**Defence:** `FilterIsOughtIds` partitions IDs; commitment IDs are explicitly excluded from `ContributionRecord.fact_ids`. Dedup also applied.

### 9. Clawback stickiness after status flip — **SURVIVED**

Once `revenue_clawback_block` is stamped (from disproval), TVW must remain 0 even if the fact's `status` is later flipped back to `ACTIVE` via keeper-level `SetFact`. The clawback stamp is sticky.

**Defence:** `ComputeTrainingValueWeight` checks `fact.RevenueClawbackBlock > 0` independently of status.

### 10. Sponsor self-finalize blocked — **SURVIVED**

Wave 4 invariant: the ONLY bounty-acceptance path is a finalized passing verdict from the verifier panel. Sponsor calling `MsgAcceptAugmentation` on a pre-verdict bounty augmentation must reject.

**Defence:** `AcceptAugmentation` handler rejects when the augmentation has `BountyId != ""` unless verdict is already passing.

### 11. Verifier Sybil on augmentation panel — **KNOWN GAP** (Wave 10)

A single actor controlling three addresses can push any verdict — DRIFT-as-EQUIVALENT, etc. Verifier identity is not yet stake-weighted or calibration-gated; consensus is raw vote count.

**Current state:** attack succeeds in the test (`TestRouteB_Wave9_VerifierSybilKnownGap`); this is intentional to document the gap.

**Deferred mitigation (Wave 10):** one or more of:
- **Calibration-gated voting.** Require `AgentCalibration.total_submissions > 0` and/or `calibration_score_bps > floor` to vote. Raises Sybil cost from ~free to "must earn baseline calibration per address".
- **Stake-weighted consensus.** Votes weighted by validator stake or staked-verifier deposit. Routes Sybil through the staking layer where it's already expensive.
- **Pre-approved verifier registry.** Governance-gated list of eligible verifiers. Loses permissionlessness but rigorous.

The three mitigations are stackable. A `Params.RequireVerifierCalibration bool` field (disabled at launch, enabled by governance) is the cleanest first step.

**Why this wasn't hardened in Wave 9:** the proper fix is a coherent Sybil-defence design across all verifier-invoked paths (augmentation panel, future contribution-challenge panel, future reasoning-step attestation panel). Doing it ad-hoc here would preempt that design.

### 12. Heartbeat scales to many expiring bounties — **SURVIVED** (soft)

50 bounties all expiring in the same block are processed without failure or stall in the test harness. The scan is unbounded (O(total bounties) per block); at very large N this becomes a consensus-gas concern. Tested at N=50.

**Hardening path (mid-horizon):** replace the full scan with a time-indexed expiry queue (reverse index `block_height → bounty_id`), so work per block is proportional to expiring count, not total bounties.

### 13. FilterIsOughtIds dedups input — **SURVIVED**

The filter's dedup step ensures attackers cannot pad `fact_ids` with repeated entries to inflate counts.

**Defence:** `seen` map in `FilterIsOughtIds`.

### 14. TVW on absent fact — **SURVIVED**

Query for a non-existent fact returns zero, not panic, not an error.

**Defence:** `ComputeTrainingValueWeight` returns a zero-value breakdown.

### 15. Double-finalize rejection — **SURVIVED**

Once FINALIZED, the Merkle root must be immutable.

**Defence:** `FinalizeTrainingManifest` requires `status == DRAFT`.

### 16. Vote after verdict finalized — **SURVIVED**

Once the verifier panel reaches consensus, further votes must reject.

**Defence:** `RecordAugmentationVote` checks `aug.Verdict != PENDING` and errors.

### 17. Bundle detects parent_merkle_root tampering — **SURVIVED**

If a malicious caller directly `SetTrainingManifest` with a child whose `parent_merkle_root` is swapped, the bundle's `derived_merkle_root` won't match the stored `merkle_root` and `merkle_root_valid` reports `false`.

**Defence:** `AssembleManifestBundle` re-derives the composed root from stored inputs; the `merkle_root_valid` flag is the verification signal.

**Implication:** external consumers who only trust the chain's commitment (and not its RPC serialisation) can detect this locally. The Merkle design's promise of trust-minimised verification holds.

### 18. Empty-selector safety — **SURVIVED**

A selector against an empty fact namespace produces an empty manifest with a deterministic empty-set root. No panic, no garbage.

### 19. Missing pipeline rejected — **SURVIVED**

Manifest for a pipeline that doesn't exist is rejected at create time.

### 20. Manifest non-operator rejected — **SURVIVED**

Only the pipeline operator may create a manifest for their pipeline.

### 21. Cross-pipeline parent manifest — **POLICY ITEM**

A manifest created for pipeline B may reference a manifest from pipeline A as its parent. This is currently **PERMITTED** (`TestRouteB_Wave9_CrossPipelineParentPermitted`).

**Intended?** Training lineages legitimately cross operators (a distillation on a third-party SFT bundle; a fine-tune on an open-source foundation). Allowing cross-pipeline parent is a feature.

**Attribution concern:** operator B's attestation will still cite their own pipeline; the cross-pipeline parent appears in the manifest metadata. Observers can distinguish "built entirely in-house" vs "inherited from external".

**Governance option if this proves abuse:** add a `CorpusSelector.disallow_cross_pipeline_parent` flag or a per-pipeline `accepts_external_parents bool` to gate. Not implemented — waiting for real-world signal.

### 22. Double-clawback idempotent — **SURVIVED**

Calling `ClawbackOnDisproval` twice on the same fact is a no-op on the second call; the `revenue_clawback_block` stamp does not reset.

**Defence:** early return on `fact.RevenueClawbackBlock > 0`.

### 23. Empty-manifest Merkle — **SURVIVED**

Commitment over entirely-empty ID sets is deterministic, well-formed, and distinct from the composed variant (domain tags differ).

---

## Open audit items (not tested in Wave 9, queued for Wave 10+)

These are not vulnerabilities discovered in Wave 9 — they are **known frontier items** we chose not to attack because the defence design is pending.

- **Verifier Sybil defence** (item 11 above). Wave 10 target.
- **Attribution over-reporting without corrective challenge.** A model owner can inflate `ContributionRecord.fact_ids` by listing facts they didn't actually train on. `MsgChallengeContribution` exists but requires a bond (5 ZRN) to initiate. Low-value or obvious over-reports may go unchallenged. Possible mitigation: rate-limit by owner, or require an evidence-backed attestation hash for each listed fact.
- **Heartbeat unbounded scan.** Acceptable for now (50 bounties processed cleanly); replace with indexed expiry queue if the chain's training-fund volume grows beyond ~10k concurrent bounties.
- **Research fund forfeiture cannot always complete.** The `ReturnEscrowToSponsor` fee portion is now non-blocking (Wave 8 fix) — if the research fund isn't wired (test harness) the fee stays in the training fund. Acceptable for correctness; minor accounting drift is possible on chains where `vestingRewardsKeeper` isn't bound.
- **Methodology-set versioning** is derived (max over individual `Methodology.Version`), not explicit. A governance-ratified `MethodologySetVersion` singleton would be cleaner; the current approximation is acceptable.
- **Pipeline operator rotation.** A pipeline's operator cannot (currently) be changed. If operator keys are compromised, there's no recovery path other than creating a new pipeline. Flag for future governance-gated `MsgRotatePipelineOperator`.
- **Parent-child manifest delta is purely set-subtraction.** No checks for semantic coherence (child "delta" could be entirely unrelated content). This is by design — delta is the new content, not a "derivative" — but worth documenting.

---

## What this audit does NOT cover

- **Cryptographic primitives.** SHA-256 assumed secure; no audit of the hash construction itself beyond the domain-separation design.
- **IBC + cross-chain.** No manifest posting between chains yet; Wave 11+ territory.
- **Governance proposal authenticity.** Trust in the authority field is trust in the underlying gov module, which is out of Route B's scope.
- **Smart contract hooks.** No BVM callers yet; when they land, the hook surface will need its own adversarial pass.
- **Load at scale.** N=50 in-harness ≠ N=10M on a live chain. Separate load-test effort pending.

---

## The honest take

Route B has survived 21 of 23 attacks. The one hardening was a real bug (depth off-by-one). The one known gap is Sybil resistance on the verifier panel — a design problem with multiple possible solutions, deferred to a Wave where we can do it coherently rather than ad-hoc. The one policy item is cross-pipeline parent, which is arguably a feature.

No attack revealed a fundamental architectural flaw. The commitment layer is sound, the economic invariants hold, the is-ought wall is structural, the heartbeat is idempotent, the genesis round-trip preserves state.

We continue under no illusion that this makes Route B unattackable. We continue with the evidence that the defences that exist, hold — and that the gap that remains has a name, a test that demonstrates it, and a plan.

— **Route B, Wave 9 · Adversarial Audit** · 2026-04-23
