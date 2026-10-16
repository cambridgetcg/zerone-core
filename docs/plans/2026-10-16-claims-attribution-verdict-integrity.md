# Claims: attribution, verdict and reversal integrity (knowledge v12)

Status: implementation plan, not started. Nothing in this file has been built or tested.
Base: upstream `main` at `a4272bfa`. Branch: `claims/attribution-verdict-integrity`
(worktree `/Users/yu/.claude/worktrees/zerone-core-claims-integrity`). The branch is meant to
fast-forward into `main` when done, as Yu approved. Commit after each task, following the repo
CLAUDE.md.

Sources:
- Probe artifacts from the 2026-10-16 sandbox on a4272bf:
  `/tmp/zerone-sandbox-20261016/probes/index.json`, with `raw/` and `wf/` beside it.
- Three candidate designs and three judge reviews (workflow inputs).
- Code read again for this synthesis.

How certain each claim is:
- **Verified by probe**: P1, P1ext, P2, P3, P5, P6 on a4272bf.
- **Verified by reading code (this synthesis)**: every `file:line` cited in §3–§6.
- **Inferred, not probed**:
  - A starved contradiction can launder a cascade-CONTESTED fact.
  - FactByCanonical misses accepted structured claims. `rounds.go:299` re-saves the claim after
    `rounds.go:799` pointed 0x34 at the fact, and that re-save points it back at the claim ID.
  - Genesis import order decides which duplicate the 0x09 index points to.
  - An attribution niche can be pruned once it holds more than 10 facts.

Each inferred item has either a test that settles it or a note that it is out of scope.

---

## 1. Goals, mapped to Yu's decisions (2026-10-16)

| Yu's decision | What the chain must guarantee after this branch | Defects that currently break it |
|---|---|---|
| **(1) Who said what is recorded on chain** | (a) An attribution record (work, voice, act, proposition, locator) can be submitted, reviewed and found. It refers to Hermes records by hash only and never stores verbatim text. (b) Every status change of a fact records who or what caused it. | P2 (distinct statements collide, canonical forms can be squatted). P6 (wrong causes). P1 (a starved attribution locks its text). |
| **(2) The verdict on a proposition is a separate decision by chain participants** | (a) A fact's status changes only through a panel decision on that fact, or through an accepted finding that conflicts with it or that it depends on. No single submitter can change it. (b) A submitter, or the defender of the target, cannot sit on the panel. (c) Accepting an attribution never changes a proposition's verdict. | P3 (a 1-uzrn admission flips status; a starved round restores it). P5 (self-review). P3: CONTESTED facts cannot be challenged. |
| **(3) Verdicts can be reversed by the latest findings** | (a) A CONTESTED fact has a decision path: ChallengeFact accepts it. (b) When a counter-fact is disproven, its target is re-evaluated. (c) If a fact survives a direct challenge, that clears doubts raised by dependencies refuted earlier. (d) A rejected, starved or fallen proposition can be re-proposed as a new record, with the old one kept. | P1 (permanent lock). P3 (no exit from CONTESTED; no re-evaluation after a counter-fact falls). |

Non-goals for this slice: zerone-1, metabolism economics, new tx fields, on-chain quote detection.
The full list is in §8.

---

## 2. Design summary

The backbone is design D1: minimal, consensus-safe and upgrade-first. All three judges ranked
it highest. It is changed in these ways:

- **D2's idea: contradictions take effect when accepted, not when admitted.** Neither
  `MsgSubmitContradiction` nor `MsgSubmitClaim`+CONTRADICTS changes the target's status at
  admission. When the counter-claim is **accepted**, the target becomes CONTESTED. This removes
  the cheap grief window (a refundable deposit or a 0.1 ZRN fee buys a 450-block CONTESTED
  window). It also takes the unbounded "is any claim still live" scan
  (`rounds.go:972-997`) off every new BeginBlock path. Yu may reverse this choice (§9, D1).
- **Whether a fact stays CONTESTED is computed from its relation edges, not from events.**
  - R1: an accepted contradicting fact that is still standing.
  - R2: a fact it depends on was refuted after the fact's last survived challenge.
  This avoids D2's `CascadeEventByDescendant` gap: the cascade skips descendants that are
  already CONTESTED or CHALLENGED (`vindication.go:619-627`).
- **D2's survival rule, made order-aware.** Surviving a direct challenge (verdict REJECT) clears
  R2 only for dependencies refuted at or before that survival. An inconclusive or starved
  challenge clears nothing, which closes judge 2's laundering path.
- **The scan never errors toward VERIFIED.** If the contest-reason scan errors or hits its bound,
  the fact stays CONTESTED and an event is emitted. A BeginBlock path never receives an error and
  never restores status.
- **D3's index rule, which needs no extra reads.** With the marker on, `SetClaim` writes the 0x09
  and 0x34 index entries only on the claim's first save. It already reads `previousBytes`
  (`state.go:189-195`). Admission of a new claim therefore overwrites the index; re-saves never
  do.
- **D2/D3 defender recusal, limited to a single target.** The rule never reads a claim's
  attacker-chosen relation list (judge 2's recusal attack).
- **D3's non-consensus pieces:** ClaimHistory links, an attribution encoder, a quote guard, and
  the `local-node.py --home` fix.

What is left out and why is in §10.

**Proto changes.** There is exactly one: the genesis-only `bool claim_integrity_enabled = 71`
(§3.2). There are no tx-message or state-type proto changes.

---

## 3. Activation and upgrade strategy

### 3.1 Marker
- New file `x/knowledge/keeper/claim_integrity_activation.go`. It follows the shape of
  `fund_settlement.go:13-49`.
- Key `ClaimIntegrityEnabledStoreKey = "\x87"`. 0x87 is free:
  - prefixes in `types/keys.go` stop at 0x80;
  - the existing markers are 0x81, 0x84, 0x85 and 0x86;
  - 0x82 and 0x83 are used by verifier settlement.
- Value is exactly `{1}`. Reads behave as follows:
  - nil → false;
  - `{1}` → true;
  - anything else → an error, which every caller returns. An error is never treated as false.
- `EnableClaimIntegrity` requires markers 0x81, 0x84, 0x85 and 0x86.
- `ValidateClaimIntegrityActivation` checks that the marker is absent, the predecessors are on,
  and `migration_v12_complete` is absent.
- No message or param can toggle the marker, and there is no way to deactivate it.

### 3.2 Genesis flag (the only proto change, and why it is needed)
`proto/zerone/knowledge/v1/genesis.proto`: add `bool claim_integrity_enabled = 71;` after
`fund_settlement_enabled = 70`. Then run `make proto-gen` and `make proto-check`, and never
hand-edit `*.pb.go`.

Why it is needed:
- A fresh chain on the v12 binary has no 11→12 migration, so native genesis must state the
  selection explicitly.
- Deriving the flag from `fund_settlement_enabled` would silently switch semantics when a
  pre-v12 export is imported. That would break the rule in `genesis.proto:375-379`: "import does
  not invent upgrade receipts".

Wiring:
- `types/genesis.go:452`: `DefaultGenesis` sets the flag to true.
- `types/record_integrity_genesis.go`: if the flag is on, require `FundSettlementEnabled`.
- `keeper/genesis.go`:
  - call `EnableClaimIntegrity` right after the fund-settlement block (`:270-274`);
  - export the flag (`:379-399`, `:527`) and pass it into `ValidateGenesisRounds`.
- `app/app.go`: next to `:1489-1493`, add
  `knowledge >= 12 → validateClaimIntegrityGenesisSelection`, which requires an explicit
  `claim_integrity_enabled=true`. Copy the shape of `upgrades_fund_settlement.go:159-170`.

### 3.3 Module migration 11 → 12
- New package `internal/claimintegritymigration`, a copy of `internal/fundsettlementmigration`,
  with `UpgradeName = "knowledge-claim-integrity-v1"`, `WithOwner` and `Require`.
- New `Migrator.Migrate11to12`, following the shape of `migrator.go:146-167`:
  1. `Require` the owner;
  2. open a `CacheContext`;
  3. `ValidateClaimIntegrityActivation`;
  4. `EnableClaimIntegrity`;
  5. `WriteMigrationMarker("migration_v12_complete","true")`;
  6. `write()`.
- `module.go`: `RegisterMigration(knowledge, 11, Migrate11to12)`, and `ConsensusVersion() = 12`.
- No record is rewritten, there is no index rebuild and no inventory scan.

### 3.4 App upgrade handler
New file `app/upgrades_claim_integrity.go`, mirroring `app/upgrades_fund_settlement.go`.

- **Version maps.** `claimIntegritySourceVersionMap() = fundSettlementTargetVersionMap()`. The
  target is the same map with `knowledge = 12`.
- **`requireClaimIntegrityTransitionOwner`.** Knowledge versions must be within 6..12.
  - Return nil if `from == target`, or if both are below 12.
  - Otherwise require the name `knowledge-claim-integrity-v1`, `from == 11` and `target == 12`.
- **`validateClaimIntegritySource`.** The same checks as `validateFundSettlementSource`, plus:
  - exact plan: the name, `height > 0`, empty `Info`, zero `Time`, no client state;
  - exact source and target maps, and the stored map equals `fromVM`;
  - the owner, accounting and survival checks;
  - `validateSDK053IBC10CompletedOrNativeLineage`;
  - fund settlement completed, using a `validateFundSettlementCompleted` helper extracted from
    the target branch of `upgrades_fund_settlement.go:118-140`;
  - 0x87 absent, upgrade done-height 0, marker absent, and the height is not a skip height.
- **Handler.** It has the same shape as `upgrades_fund_settlement.go:91-115`:
  - require `BlockHeight == plan.Height` and `LastCommitID().Version == plan.Height-1`;
  - validate the source;
  - in a cache: `WithOwner`, `RunMigrations`, then require `DeepEqual(target)`;
  - `write()`.
- **Startup validation.** This becomes the first branch of
  `validateSurvivalHandoffStartupVersions` (`upgrades_survival_handoff.go:136-145`).
  - At the target map: require marker `{1}` and completed fund settlement. Then accept either
    native genesis (`done == 0 && !marked`) or an applied upgrade
    (`0 < done ≤ latest && marker == "true"`).
  - At the source map: accept only at exact H−1, with matching on-chain and local plans. Then
    run the source validation.
- **Owner chain.**
  - Raise the hard upper bound from 11 to 12 and add the delegation `from == 11 && target == 12`
    in:
    - `requireRecordIntegrityTransitionOwner` (`upgrades_record_integrity.go:33`);
    - `requireReviewNeutralityTransitionOwner` (`upgrades_review_neutrality.go:33`);
    - `requireClaimRecordsTransitionOwner` (`upgrades_claim_records.go:30`);
    - `requireFundSettlementTransitionOwner` (`upgrades_fund_settlement.go:30`).
  - The fund-settlement owner delegates 11→12 to the claim-integrity owner.
  - Survival handoff (`upgrades_survival_handoff.go:38`, `:53`): add the pair `{12,3}` and
    delegate `{11,3}→{12,3}`.
- **Registration.** Register in `app/upgrades.go:147-153` and add a lineage entry in
  `app/upgrade_registry.go:68`. There is no new store key, so no store loader is needed.

### 3.5 Why replay stays deterministic
1. Every new behaviour is guarded by `ClaimIntegrityEnabled(ctx)`.
2. 0x87 is written only by `InitGenesis` or by `Migrate11to12`, inside the upgrade cache at H.
   No pre-H state contains it.
3. The v12 binary never executes pre-H blocks. The startup guard admits v11 state only at exact
   H−1, and native genesis requires the explicit flag.
4. With the marker off, code paths are byte-identical. Task 2's golden knowledge-store digest
   enforces this for the whole slice.
5. No new code iterates over a map in a consensus path. Relation and history scans iterate KV
   prefixes in key order. Context values carry only strings.

### 3.6 Existing state at activation: nothing is rewritten
| State at activation | What happens |
|---|---|
| Dedup entries (0x09, 0x34) pointing at terminal claims or fallen facts | They stop blocking: the lookup now checks status (§5.4). |
| Legacy v1 canonical hashes | They do not interact with v2: v2 admission never computes v1 hashes, so a reworded twin of a pre-v12 structured claim can be admitted once. Exact text is still caught by the content hash. |
| Facts CONTESTED by a pre-activation admission flip | Released by the same contest-reason predicate when that contradiction's own round ends. If it is accepted, R1 keeps the fact CONTESTED. |
| Facts that were laundered earlier (VERIFIED while a standing counter-fact contradicts them) | Unchanged. §5.6 tells consumers how to read them. Either side can be challenged. |
| Pre-activation self-commits | They still reveal and settle. The P5 guard applies to commits only. |
| Historical status transitions | Never rewritten. |

### 3.7 Networks
- **Fresh local sandboxes.** `zeroned init` uses `DefaultGenesis`, so the new semantics apply
  from block 1.
- **zerone-dev-1.** It runs 5542b22, knowledge v11, with all four flags on and no reviewers.
  - The upgrade would need a signed gov plan `knowledge-claim-integrity-v1` at height H2 with
    empty info, plus the stage-upgrade/run-upgrade flow in
    `deploy/networks/zerone-dev-1/README.md`.
  - The v12 binary also activates everything in `5542b22..a4272bfa`. That range is 64 commits
    and touches 64 files under `x/` and `app/` (+2817/−950), counted in this session.
  - Deploy tooling must also accept v12. The gateway allowlist and the runtime checks both need
    updating.
  - **This branch does not touch dev-1.** Scheduling it is a decision for Yu (§9, D4).
- **zerone-1** (v5 lineage, 30 s blocks): no action. This binary refuses that state. v12 is
  simply appended to the lineage zerone-1 would have to traverse.

---

## 4. On-chain attribution convention (who said what)

### 4.1 Two records, two separate verdicts
1. **Attribution claim.** It says: "work W, in voice V, performs act A on proposition P at
   locator L." Its panel judges **fidelity only**, never truth.
2. **Proposition claim.** An ordinary claim stating P in the extractor's own words. Its panel
   judges truth or support.

The two are joined by a hash. There is no Reference or Relation edge between them:
- `computeProvenance` would clamp the proposition's confidence to the attribution's
  (`rounds.go:1043-1080`).
- A CITES edge would make refuting an attribution cascade into the proposition.

An attribution can stay VERIFIED while its proposition is DISPROVEN. It remains true that Smith
said it.

### 4.2 Field mapping (an ordinary `MsgSubmitClaim`; no new message, field or store key)
| Field | Value | Notes |
|---|---|---|
| `claim_type` | `CLAIM_TYPE_RELATION` | v1 RELATION already keeps object and scope (`canonical.go:55-66`). v2 also keeps temporal_scope and case. |
| `domain` | the proposition's domain | Attribution and proposition share the content-hash namespace. |
| `category` | `empirical` | `historical` is not an accepted category (`types/axiom.go:54-60`). |
| `method_id` | `M-HISTORICAL` | Primary-source documentation with provenance (`types/methodologies.go:273`). Check that it is registered on the target chain. See §9 D7. |
| `structure.subject` | work ID, lowercase: `doi:<doi>`, `arxiv:<id>v<n>`, `isbn:<13>`, or `url-sha256:<64hex>` | The source document anchors the record. |
| `structure.predicate` | `<act>:<voice>`. act ∈ {`states`, `reports`, `hedges`, `denies`}. voice ∈ {`author`, `quoted:<speaker-ref>`} | `speaker-ref` is `orcid:…`, `wikidata:q…`, or `hermes:<64hex>` (opaque). Private persons appear **only** as opaque `hermes:` refs. |
| `structure.object` | `zrn-content:<64hex>` | The proposition claim's on-chain content hash: `sha256("ZRN.claim.v1:" + domain + ":" + text)` (`state.go:571-578`). It can be computed before the proposition exists. |
| `structure.scope` | `ver=sha256:<64hex>;loc=<locator>` | The digest of the retrieved source bytes (Hermes SourceVersion), plus the Hermes locator in a lowercase grammar such as `p4.para2`, `sec3.2.para4`, `fig2`, `t=00:12:31`. It contains no text. |
| `structure.temporal_scope` | publication date of the pinned version (`YYYY-MM-DD`) | Informational. |
| `structure.tags` (≤10, each ≤50 chars) | `attr-v1`, `stmt:<first 40 hex of the StatementOccurrence hash>`, `zc:<first 40 hex of object hash>` | `FactsByTag` joins accepted attributions. Tags are indexed only when a fact is created. |
| `fact_content` (20–1000 B) | fixed template: `Attribution zerone.attribution/v1: <subject> <predicate> <object> at <scope>` | No free text, so no excerpt can get in. The content hash identifies the tuple. |
| `reasoning_trace` (≤8192 B) | RFC 8785 JCS JSON, hashes only. Keys: `schema`, `source_version_record`, `statement_occurrence`, `excerpt_sha256`, `proposition_record`, `extractor` | Full hashes live here. Requires record integrity (already on wherever v12 runs). |
| `references`, `relations` | empty | References must be fact IDs (probe `p4-refs-doi-r1` refused a DOI). |
| `canonical_form` | omitted | Derived from the structure (v2 after activation). |

### 4.3 Example (synthetic)
Proposition claim `P1`:
- domain `chemistry`, category `empirical`;
- type ASSERTION, with structure:
  - subject `boiling point of compound Z`
  - predicate `equals degrees C at 1 atm`
  - object `78.4`
  - scope `pure sample`
- text: `Compound Z boils at 78.4 °C at 1 atm for a pure sample.`
- Content hash (computed):
  `f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026`.
- v2 canonical form:
  `zrn.cf.v2/CLAIM_TYPE_ASSERTION(domain="chemistry",subject="boiling point of compound Z",predicate="equals degrees C at 1 atm",object="78.4",scope="pure sample",temporal_scope="")`

Attribution claim `A1`: Smith 2019 states P1 as author, on p. 4, paragraph 2.
```
claim_type  RELATION   domain chemistry   category empirical   method_id M-HISTORICAL
subject     doi:10.1000/p6-smith-2019
predicate   states:author
object      zrn-content:f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026
scope       ver=sha256:0580a051899e6a386f8213633badf896d07f21a370a8ba915f2bf93ba054bce3;loc=p4.para2
temporal    2019-03-01
tags        attr-v1, stmt:e25fa6b012331b0f8681c04b88d4ee7eacaa089d, zc:f789edd890d61613208c09ebc74248b09c2f170e
fact_content (243 B)
  Attribution zerone.attribution/v1: doi:10.1000/p6-smith-2019 states:author zrn-content:f789…1026 at ver=sha256:0580…bce3;loc=p4.para2
reasoning_trace (JCS)
  {"excerpt_sha256":"sha256:<64hex>","extractor":"did:zrn:<…>","proposition_record":"sha256:<64hex>",
   "schema":"zerone.attribution/v1","source_version_record":"sha256:<64hex>","statement_occurrence":"sha256:e25f…a685"}
```
In the real fact_content, every `…` above is the full 64-hex value.

Further examples:
- **Jones 2021 reports a different value (81.0).** Submit a separate proposition `P2` with
  object `81.0`. After v12 it no longer collides with P1. Then submit attribution `A2` with
  `subject doi:10.1000/p6-jones-2021`, `predicate reports:author` and
  `object zrn-content:<hash of P2>`.
  - If Jones's finding should contest P1, submit P2 as `MsgSubmitClaim` with a CONTRADICTS
    relation to P1's fact, or as `MsgSubmitContradiction`. P1 becomes CONTESTED only if P2 is
    accepted (§5).
- **A quoted speaker.** Use `predicate states:quoted:orcid:0000-0002-1825-0097` (a public
  person), or `states:quoted:hermes:<64hex>` (a private person).

### 4.4 Roles
| Role | Where it lives on chain |
|---|---|
| author / quoted speaker | `subject` plus the voice in `predicate` |
| extractor | `reasoning_trace.extractor`, bound by the StatementOccurrence hash |
| submitter | `Claim.submitter`, signed. The zerone_auth DID is already shown in claim-history. |
| reviewers | Round commits and reveals with `ReviewAttestation{method_id, reason, evidence_ids, scope}` (`types.proto:796-801`). `scope` is `attribution-fidelity` or `proposition-truth`. |

### 4.5 How Hermes records bind to the chain
- Every Hermes record is referenced as `sha256:<64 lowercase hex>` over Hermes's canonical bytes.
  JCS (RFC 8785) is proposed; Yu confirms with the Hermes builder (§9 D5).
- SourceVersion → the `ver=` digest in `scope`, plus `source_version_record`.
- StatementOccurrence → `statement_occurrence`, `excerpt_sha256` and the `stmt:` tag. The excerpt
  and the full role list stay offline.
- Proposition → the proposition claim's text and `proposition_record`. It is joined on chain by
  `zrn-content:<content hash>`.
- EvidenceAssessment and RelationReview → reviewer `evidence_ids` (≤16 × ≤256 B), written as
  `hermes:ea:sha256:<hex>` or `hermes:rr:sha256:<hex>`.
  - They can also go in `MsgSubmitContradiction.evidence_ids`, which is retained since
    claim-records v1.
- After inclusion, Hermes stores a receipt `{chain_id, height, txhash, claim_id, content_hash}`.
- If Hermes later redacts an offline record, the on-chain hash is left pointing at nothing,
  which is harmless.

### 4.6 What attribution reviewers judge (spec rubric)
- **ACCEPT** when all of the following hold:
  - (a) the source version matches `ver=`;
  - (b) at the locator there is an exact excerpt whose NFC sha256 equals `excerpt_sha256`;
  - (c) the excerpt is uttered in the stated voice;
  - (d) the act is faithful: a hedge stays `hedges`, and reporting someone else's result is
    `reports`;
  - (e) the proposition text renders the excerpt fairly, neither stronger nor narrower.
- **REJECT** for misattribution, a wrong locator or distortion.
- **MALFORMED** when hashes cannot be checked, a field contains free text, or any verbatim
  excerpt appears on chain.

Reviewers never vote on whether the proposition is true.

### 4.7 Quote and licensing guard
The chain has no excerpt field and cannot detect quotation. Anything written is permanent. So the
guard runs **before signing**, in `claim-workflow` (Task 4). The Hermes intake should apply the
same algorithm.

- Attribution structure fields must match `^[a-z0-9][a-z0-9:._/;=+-]*$`.
- The excerpt is read from a local file and never leaves it.
- Any on-chain text (fact_content, reasoning_trace, structure fields, contradiction text,
  challenge or review reason) is refused if, after normalisation, it contains:
  - any window of **8 consecutive excerpt tokens**; or
  - the whole excerpt, when the excerpt has fewer than 8 tokens.
  Normalisation is NFKC, casefold, every non-letter/non-digit replaced by a space, then split on
  whitespace.
- Receipts record only the guard version and the excerpt's sha256.

Reviewers back this up on chain by voting MALFORMED. A third-party client can skip the guard;
that limit is disclosed.

### 4.8 Known limits
- Attribution facts are still subject to metabolism (AT_RISK, EXPIRED, PRUNED). The durable
  record is the claim, its verdict and its history, not the fact's metabolic status.
- Facts in the same niche — same domain, subject, claim type and scope (`state.go:1023-1041`) —
  are pruned above `CompetitionMaxNicheSize`. Scope includes the version digest and the locator,
  so in practice a niche is one locator in one version.
- A tag is a 160-bit prefix of a hash. Full hashes are in `object` and `reasoning_trace`.

---

## 5. Verdict and reversal semantics (after activation)

### 5.1 Claim outcomes (unchanged)
| Outcome | Meaning |
|---|---|
| PENDING / IN_VERIFICATION | No verdict yet. |
| ACCEPTED | A fact is created as VERIFIED (PROVISIONAL for a conjecture). |
| REJECTED | Not accepted. |
| MALFORMED | Not truth-apt as stated. |
| INSUFFICIENT (and legacy EXPIRED) | No quorum, so no decision. |

### 5.2 Fact state machine (each row records a cause; P6)
| From | To | Trigger | `cause_event_type / cause_id` |
|---|---|---|---|
| — | VERIFIED (PROVISIONAL for a conjecture) | claim ACCEPTED | `verification / <claim id>` (unchanged) |
| VERIFIED, ACTIVE | CONTESTED | a claim carrying CONTRADICTS→F is **ACCEPTED** | `contradiction_accepted / <counter-claim id>` |
| VERIFIED, ACTIVE, AT_RISK | CONTESTED | a fact F depends on is DISPROVEN (existing cascade) | `cascade / <disproven fact id>` (unchanged) |
| CONTESTED | VERIFIED | a contradiction ends non-accepted (REJECT, MALFORMED, INCONCLUSIVE, starved) **and** no contest reason remains | `contradiction_reversed / <claim id>` |
| CONTESTED | VERIFIED | counter-fact K becomes DISPROVEN and no contest reason remains | `counterfact_disproven / <K>` |
| VERIFIED, ACTIVE, **CONTESTED** | CHALLENGED | `MsgChallengeFact` | `challenge_opened / <challenge claim id>` |
| CHALLENGED | DISPROVEN | challenge ACCEPT | `challenge_disproven / <challenge claim id>` |
| CHALLENGED | ACTIVE or CONTESTED | challenge REJECT (survived). CONTESTED iff R1 holds, or R2 holds for a dependency refuted after this survival (not possible at that instant). | `challenge_survived / <challenge claim id>` |
| CHALLENGED | ACTIVE or CONTESTED | challenge INCONCLUSIVE, MALFORMED or starved. CONTESTED iff R1 or R2 holds. Nothing is cleared. | `challenge_inconclusive / <challenge claim id>` |
| metabolic statuses | — | unchanged | inferred, as today |

Rules that do **not** change any fact:
- Admitting a contradiction.
- Accepting an attribution.
- Accepting a counter-claim, for the counter-claim's own target's *verdict*. The target only
  becomes CONTESTED; it is never made DISPROVEN.

DISPROVEN can only come from a direct challenge on that fact.

### 5.3 Contest reasons (`contestReasonRemains(ctx, F, survivedAt) bool`; it never returns an error)
- **R1 — standing conflict.** Some incoming relation to F has type `CONTRADICTS`, and its source
  fact's status is one of VERIFIED, ACTIVE, CONTESTED, CHALLENGED or AT_RISK.
- **R2 — refuted dependency.** Some outgoing relation from F has type SUPPORTS, REQUIRES, REFINES,
  GENERALIZES or CITES (the cascade set). Its target D is DISPROVEN, and
  `survivedAt == 0 || disprovenAt(D) > survivedAt`, where:
  - `disprovenAt(D)` is the block height of D's latest transition to DISPROVEN, or 0 if none is
    recorded;
  - `survivedAt` is passed in, or else is the height of F's latest `challenge_survived`
    transition (0 if none).
- **Failure handling.** A read error, a malformed record, or exceeding the 100k-record / 64 MiB
  bound (the same bound as `hasOtherActiveChallenge`) all mean the reason remains (`true`). The
  predicate also emits `zerone.knowledge.contest_check_incomplete{fact_id, reason}`.
- **Pending contradictions are not reasons.**

### 5.4 Duplicate checks and resubmission (P1, P2)
An index entry (0x09 content hash, or 0x34 canonical hash) blocks admission only while the record
it points at is live or standing.

| Record the entry points at | Blocks? |
|---|---|
| A claim with status PENDING, PENDING_EVALUATION, EVALUATED, PROVISIONAL, IN_VERIFICATION, CHALLENGED, CONTESTED or UNSPECIFIED | yes |
| A claim with status REJECTED, MALFORMED, INSUFFICIENT or EXPIRED | no |
| A claim with status ACCEPTED | The fact is resolved as `GenerateFactID(claim.Id, round.VerdictBlock)`, which is the same height as `createFactFromClaim` (`rounds.go:205-206`, `:612`). It blocks if that fact is standing. A missing round or fact fails closed: it blocks. |
| A fact with status PENDING, PROVISIONAL, VERIFIED, ACTIVE, CONTESTED, CHALLENGED, AT_RISK or UNSPECIFIED | yes |
| A fact with status SUPERSEDED, EXPIRED, DISPROVEN, REVOKED or PRUNED | no |
| An entry pointing at nothing | no |

Further rules:
- When a resubmission is admitted, its first save overwrites the index, so the index always names
  the latest admitted record.
- The usual fee and cooldown apply.
- The error text keeps its current prefix and adds the status of the record that blocks.
- `claim-workflow.py --resubmit-after <txhash> --max-review-fee-uzrn N` then works on chain with
  no client change.
- Reversing a DISPROVEN proposition is done by re-asserting it: a new claim and a new fact are
  created. The old fact keeps its history.

### 5.5 What consumers read
- **Current verdict of a fact.** Read `fact.status`, the status transitions with their causes,
  and its incoming CONTRADICTS edges together with each source fact's status.
- **Evidence behind a verdict.** Use `ClaimHistory(claim_id)`, which gives:
  - rounds, commits, reveals and attestations: who reviewed, the vote, the method, the reason and
    evidence hashes;
  - the related claims, with link fields `provisional_fact_id`, `challenged_claim_id`,
    `relations.contradicts`, and the new `content_hash`, `canonical_hash` and
    `structure.object` (Task 3).
- **Who said P.** Read `ClaimHistory(P's claim id)` and use the related claims linked by
  `structure.object`. Each one is an attribution claim with its own fidelity verdict. For
  accepted attributions only, `FactsByTag("zc:<40hex>")` also works.
- **A proposition by its structure.** Use `FactByCanonical` with the v2 form. Forms with the
  `zrn.cf.v2/` prefix are hashed verbatim after NFC and trim.

How to interpret statuses:

| Status | Meaning |
|---|---|
| VERIFIED / ACTIVE | currently accepted |
| CONTESTED | Accepted, but a standing accepted finding conflicts with it, or a fact it depends on was refuted. The transition cause says which. It does **not** mean false. |
| CHALLENGED | under adjudication |
| DISPROVEN | refuted by a direct challenge |
| AT_RISK / EXPIRED / PRUNED | metabolic, never epistemic |

Pending contradictions appear only in ClaimHistory; they are allegations, not verdicts. A panel
verdict does not prove truth or reviewer independence (`docs/specs/knowledge-claim-records-v1.md:93-99`).

### 5.6 Reading legacy records (never rewritten)
- Before H, a transition recorded as `cascade` with an empty `cause_id` was a contradiction flip
  at admission.
- Before H, a transition recorded as `verification` right after CONTESTED, with the fact's own
  claim ID, was a contradiction reversal.
- Two VERIFIED facts joined by CONTRADICTS are legacy laundering, as in probe P3. Treat them as an
  unresolved dispute. Either side can now be challenged.

---

## 6. Ordered TDD tasks

Conventions that apply to every task:
- Write the failing test first. Run `go test -count=1` on the touched packages, then on
  `./x/knowledge/... ./app/...` before committing.
- Task 2's characterization suite and golden digest must stay green through every later task.
- New behaviour is tested twice: with the marker on (new fixture `setupClaimIntegrityTest`, which
  is `DefaultGenesis` plus bootstrap funding) and with it off (a marker-off twin).
- Existing tests that rely on legacy semantics while using `DefaultGenesis` must pin
  `ClaimIntegrityEnabled=false` with a comment. Their assertions must not be rewritten.
  - Expected to need this: `app/claim_records_test.go` and `app/fund_settlement_integration_test.go`.
  - Also update any test whose small stake or self-review assumption now fails.
- No builds or writes into tracked paths other than the files listed. Binaries go to `/tmp`.

### Phase A — non-consensus

**Task 1 — Isolate the `local-node.py` version probe** · consensus: no
- Files: `scripts/local-node.py` (`initialize`, `:195-196`), `scripts/test_local_node.py`.
- Failing test: `test_source_version_probe_is_isolated`.
  - A fake executable records its argv and `$HOME`, and prints `cosmos_sdk_version: v0.53.0`.
  - The test calls a new helper `source_version(source)`.
  - It asserts that argv contains `--home <tmp>`, that `HOME == <tmp>`, that `<tmp>` is gone
    afterwards, and that the real HOME (a patched temp dir) has no `.zeroned`.
  - On a4272bf it fails, because there is no helper and no `--home`.
- Change: run `version --long --home <tmp>` inside `tempfile.TemporaryDirectory()`, with
  `env={**os.environ, "HOME": tmp}`. Keep the SDK-version regex.
- Acceptance: `python3 -m unittest discover -s scripts -p 'test_local_node.py' -v` is green.
  Existing init tests still pass.

**Task 2 — Characterize v11 semantics and record a golden digest** · consensus: no (tests only)
- Files: new `x/knowledge/keeper/claim_integrity_legacy_test.go`, and a fixture
  `setupV11SemanticsTest`: `DefaultGenesis` with the four current flags on, as on dev-1.
- Tests. They pass on a4272bf and document each defect:
  - `TestV11_StarvedClaimLocksExactText`
  - `TestV11_ContradictionOverwritesContentIndex`
  - `TestV11_CanonicalDropsObject`
  - `TestV11_UnstructuredCanonicalSquat`
  - `TestV11_StarvedContradictionRestoresUpheldTarget` (A accepted, then 1-uzrn B starves, and F
    goes back to VERIFIED)
  - `TestV11_SubmitterMayReviewOwnClaim`
  - `TestV11_TransitionCausesInferred`
  - `TestV11_GoldenKnowledgeStoreDigest`: run all of the above in one fixture, then compute
    sha256 over `uvarint(len k)‖k‖uvarint(len v)‖v` for the knowledge KV store in key order.
- Failing test: none. This is characterization; it must pass on a4272bf.
- Change: tests only. Record the digest constant with a comment naming `a4272bfa`.
- Acceptance: green on a4272bf. Task 5 adds `ClaimIntegrityEnabled=false` to the fixture; nothing
  else in this file may change afterwards.

**Task 3 — ClaimHistory discovery links** · consensus: no (query only)
- Files: `x/knowledge/keeper/query_claim_history.go` (`claimHistoryLinks` `:198-226` and its
  call site `:58`), `query_claim_history_test.go`, and the link list in
  `docs/specs/knowledge-claim-records-v1.md`.
- Failing tests. A related claim gets:
  - link `content_hash` when it has the root's `ContentHash` (a resubmission);
  - link `canonical_hash` when it has the root's non-empty `CanonicalHash`;
  - link `structure.object` when its `Structure.Object == "zrn-content:" + root.ContentHash`
    (an attribution).
  `target_id` is the root claim ID. Today these claims are absent from `related_claims`.
- Change: pass the root claim into `claimHistoryLinks`. Use the existing single scan and budgets
  and keep the existing sort. Do not expand neighbours.
- Acceptance: the new tests pass. The existing 65,536-read and 8 MiB refusal tests are unchanged.
  The proto comment on `ClaimHistoryLink.field` is updated in Task 5, which already regenerates.

**Task 4 — Attribution encoder and quote guard in `claim-workflow`** · consensus: no (client)
- Files: `scripts/claim-workflow.py`, which gets:
  - an `attribute` subcommand, a dry-run by default, with `--broadcast` going through the
    existing submit path plus structure flags;
  - an `--excerpt-file` guard on `submit`, `challenge` and `commit`.
  New test file `scripts/test_claim_workflow_attribution.py`.
- Failing tests:
  - a Hermes-style fixture record produces the exact §4.2 fields;
  - the fact_content template is byte-exact;
  - `reasoning_trace` is JCS;
  - the `zrn-content` hash matches a Go vector (`ComputeClaimContentHash`);
  - uppercase or free text in structure fields is refused;
  - the guard refuses an 8-token overlap and a whole short excerpt, and accepts a paraphrase;
  - the excerpt never appears in argv, receipts or output.
- Change: implement §4.2, §4.5 and §4.7. The default method is `M-HISTORICAL` and the default
  category `empirical`. Never send `--canonical`.
- Acceptance: the new and existing `scripts/test_claim_workflow*.py` are green. Tests use no
  network.

### Phase B — activation (consensus)

**Task 5 — Marker 0x87, genesis flag 71 and `Migrate11to12` (not yet registered)** · consensus: yes
- Files:
  - `proto/zerone/knowledge/v1/genesis.proto`. Also update the comment on `ClaimHistoryLink` in
    `claim_history.proto`; that is a comment-only change.
  - Files generated by `make proto-gen`.
  - New `x/knowledge/keeper/claim_integrity_activation.go`.
  - New `internal/claimintegritymigration/context.go`.
  - `x/knowledge/keeper/migrator.go`.
  - `x/knowledge/types/genesis.go`, `types/record_integrity_genesis.go`,
    `x/knowledge/keeper/genesis.go`.
  - Fixtures: `helpers_test.go` (3 sites), `record_genesis_test.go`,
    `relation_contract_test.go`, `types/fund_settlement_test.go`, plus the Task 2 fixture. Add
    `ClaimIntegrityEnabled=false` wherever `FundSettlementEnabled=false`.
  - Done in Task 5 (the same rule, found by running the suites): every app or cross-stack
    fixture that deletes marker 0x86 to model a pre-v11 store also deletes 0x87, since no
    pre-v12 store has it. Otherwise `ExportGenesis` panics with "claim integrity requires fund
    settlement". Sites: `app/upgrades_{record_integrity,review_neutrality,claim_records,fund_settlement,survival_handoff}_test.go`
    fixtures, `app/claim_records_test.go`, `app/review_neutrality_settlement_test.go`,
    `app/verifier_settlement_integration_test.go`, and
    `tests/cross_stack/legacy_review_fixture_test.go`. The `ClaimHistoryLink` comment also
    regenerates `docs/swagger-ui/swagger.json`, which `make proto-check` requires to be committed.
- Failing tests (`claim_integrity_activation_test.go`):
  - the marker reads absent, `{1}`, and malformed values `{0}`, `{1,1}`, `{2}` correctly;
  - Enable refuses each missing predecessor;
  - `ValidateGenesis` refuses integrity without fund settlement;
  - `InitGenesis(DefaultGenesis)` writes `0x87={1}`;
  - export→import round-trips the flag as both true and false;
  - `Migrate11to12` refuses without the owner context, with the marker already set, and with
    `migration_v12_complete` present;
  - with the owner context it changes exactly two keys (full store diff).
- Change: as §3.1–§3.3, except that module registration and the ConsensusVersion bump wait for
  Task 6. This keeps app startup green between the two commits.
- Acceptance: `make proto-gen`, `make proto-check`, and `go test` on knowledge and app are green.
  The Task 2 digest is unchanged.

**Task 6 — knowledge v12 and the named upgrade `knowledge-claim-integrity-v1`** · consensus: yes
- Files:
  - `x/knowledge/module.go`
  - new `app/upgrades_claim_integrity.go` and `app/upgrades_claim_integrity_test.go`
  - `app/upgrades.go`, `app/upgrade_registry.go`, `app/app.go`
  - `app/upgrades_survival_handoff.go`, `app/upgrades_fund_settlement.go`,
    `app/upgrades_claim_records.go`, `app/upgrades_review_neutrality.go`,
    `app/upgrades_record_integrity.go`
  - any existing `app/upgrades_*_test.go` that asserts the compiled target is v11 (for example
    `upgrades_fund_settlement_test.go:36,49`)
- Failing tests, mirroring `upgrades_fund_settlement_test.go`:
  - the handler owns exactly the 11→12 boundary, and every older plan name is refused for 11→12;
  - the migration preserves every byte except 0x87 and the marker (full store diff);
  - wrong maps, non-empty info, a skip height, premature execution, and an incoherent
    done-height/marker are all refused;
  - native genesis without the flag is refused;
  - restart at the target is accepted;
  - `BuildChainVersionReport` lists the upgrade;
  - the module refuses an unowned migration.
- Change: as §3.3 (registration and v12) and §3.4.
- Acceptance: `go test -count=1 ./app/... ./x/knowledge/...` is green. The Task 2 digest is
  unchanged.

### Phase C — behaviour (consensus; each change is gated by the marker)

**Task 7 — Record transition causes (P6)** · consensus: yes
- Files:
  - new `x/knowledge/keeper/transition_cause.go`;
  - `history_integrity.go:118-127`;
  - `msg_server.go`: `challengeFact` `:806-810` and `ChallengeProvisionalFact`;
  - `rounds.go`: `handleChallengeSurvival` `:1212` and `restoreChallengedFactOnInconclusive`
    `:1284`;
  - `vindication.go`: the DISPROVEN write in `handleChallengeDisproven`, `:229-232`;
  - new `claim_integrity_history_test.go`.
- Failing tests, with the marker on:
  - challenge open, REJECT, starve and ACCEPT record `challenge_opened`, `challenge_survived`,
    `challenge_inconclusive` and `challenge_disproven`, each with the challenge claim ID;
  - cascade descendants keep `cascade/<root>`, so there is no leak;
  - export→import keeps the transitions;
  - with the marker off, the inferred strings are unchanged.
- Change:
  - Define `transitionCause{factID, eventType, causeID}` under an unexported context key, with
    `withTransitionCause(ctx, factID, type, id)` built on
    `sdkCtx.WithContext(context.WithValue(...))`. `CacheContext` keeps context values.
  - `setFactWithHistory` uses the cause only when `c.factID == fact.Id`; otherwise it infers as
    it does today.
  - Callers attach a cause only when the marker is on.
  - Compute the challenge claim ID before the CHALLENGED write.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 8 — Duplicate checks that look at status (P1)** · consensus: yes
- Files:
  - new `x/knowledge/keeper/claim_integrity_dedup.go`;
  - `state.go`: `SetClaim` `:216-223`;
  - `rounds.go:797-801`;
  - `msg_server.go:168-183`;
  - `msg_server_conjecture.go:106-108`;
  - new `claim_integrity_dedup_test.go`.
- Failing tests:
  - a starved claim allows resubmission, and the index then names the new claim;
  - live and ACCEPTED-with-standing-fact claims block, and the error names the ID and status;
  - REJECTED, MALFORMED and EXPIRED do not block;
  - a fact that has become DISPROVEN, EXPIRED or PRUNED allows re-assertion;
  - re-saving an old terminal claim does not move the index;
  - after acceptance 0x34 holds the fact ID, and `FactByCanonical` returns the fact (this checks
    the inferred defect);
  - a dangling entry allows admission;
  - a corrupt entry is refused (fails closed);
  - conjecture parity;
  - with the marker off, the permanent lock still reproduces.
- Change:
  - Add `dedupEntryBlocks(ctx, id) (bool, error)`, implementing §5.4. It resolves a fact first,
    then a claim, and returns errors to the tx.
  - Add `blockingDuplicate(ctx, key)`.
  - When the marker is on:
    - `SetClaim` writes 0x09/0x34 only when `previousBytes == nil`;
    - `SetCanonicalHash` at acceptance runs only if the entry is absent or equals `claim.Id`.
  - Leave the legacy path unchanged.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 9 — Canonical form v2 (P2)** · consensus: yes; the query part is not
- Files:
  - `x/knowledge/types/canonical.go`, adding `BuildCanonicalFormV2` and
    `NormalizeCanonicalFormForLookup`;
  - new `types/canonical_v2_test.go`;
  - `keeper/msg_server.go:162-177`;
  - `keeper/record_input.go:110-112` (made aware of the marker);
  - `keeper/grpc_query.go:477-483`;
  - `x/knowledge/client/cli/tx.go:205` (help text: "derived from structure; omit");
  - new `keeper/claim_integrity_canonical_test.go`.
- The v2 form:
  - `"zrn.cf.v2/" + type.String() + "(domain=" + q(domain) + ",subject=" + q(subject) + ",predicate=" + q(predicate) + ",object=" + q(object) + ",scope=" + q(scope) + ",temporal_scope=" + q(temporal_scope) + ")"`
  - `q(v) = strconv.Quote(collapse(TrimSpace(NFC(v))))`, where `collapse` turns each run of
    `unicode.IsSpace` into a single U+0020.
  - Each field is quoted on its own, with no post-processing, no lowercasing, and no tags or
    `negatable`.
  - The result is `""` when the structure is nil or the trimmed subject is empty.
  - The type is defaulted (UNSPECIFIED becomes ASSERTION) before the form is built.
- Failing tests:
  - Types:
    - object, scope and temporal_scope each change the hash for every type;
    - `"5 mW"` and `"5 MW"` differ;
    - object-only and scope-only forms differ;
    - COMPUTATIONAL and ASSERTION differ, and UNSPECIFIED equals ASSERTION;
    - quotes, commas, backslashes and control characters never collide;
    - normalisation is idempotent;
    - the v1 golden forms are unchanged.
  - Keeper:
    - 78.4 and 81.0 are both admitted;
    - an identical structure is refused;
    - a canonical form without a structure is refused (`canonical form requires claim structure`);
    - a mismatched canonical form is refused;
    - the squat sequence fails at S1;
    - with the marker off, the v1 collision still occurs.
  - Query: `FactByCanonical` with a v2 form.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 10 — Contradiction admission (P3a)** · consensus: yes
- Files: `msg_server.go`, `submitContradiction` `:1040-1170`; new
  `claim_integrity_contradiction_test.go`.
- Failing tests, with the marker on:
  - 1 uzrn is refused, the message states the minimum, and there is no bank movement;
  - a stake at the minimum is admitted;
  - DISPROVEN and REVOKED targets are refused;
  - counter-claim text equal to a live claim's text is refused, and 0x09 is unchanged;
  - once that claim is INSUFFICIENT, the same contradiction is admitted and 0x09 names it;
  - with the marker off, 1 uzrn is still admitted.
- Change. When the marker is on, before any collateral moves:
  1. the target must be standing (VERIFIED, ACTIVE, CONTESTED, CHALLENGED or AT_RISK);
     conjectures are already refused at `:1063-1066`;
  2. `stake ≥ EffectiveMinChallengeStake(params, target.Confidence)`, the same rule as
     `ChallengeFact` (`:786-792`);
  3. `blockingDuplicate(ContentHashKey(contentHash))`.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 11 — Ante gate for dispute messages** · consensus: yes (DeliverTx ante)
- Files: `app/ante_zerone.go` (`ZeroneCapabilityDecorator` gets `WithClaimIntegrityGate(fn)`),
  `app/ante.go:69`, the ante tests (`app/ante_integration_test.go`).
- Failing tests. With the gate on, for `MsgSubmitContradiction` and `MsgChallengeProvisionalFact`:
  - unregistered accounts are refused with the same onboard hint;
  - a registered account without `CanChallenge` is refused;
  - human and agent accounts are allowed.
  Also:
  - a gate read error rejects the tx;
  - with the gate off, behaviour is the legacy default-allow;
  - other message types never read the gate (checked with a spy).
- Change:
  - Read the gate lazily, at most once per tx.
  - When it is on, treat the two message types exactly like `isChallengeMsg`, both in the
    unregistered branch (`:995-1001`) and in the registered branch (`:1019-1024`).
  - Wire it as `app.KnowledgeKeeper.ClaimIntegrityEnabled`.
  - `MsgPostConjecture` is deliberately not gated (§8).
- Acceptance: `go test ./app/...` is green.

**Task 12 — Contradictions contest only when accepted; contest-reason predicate (P3b)** · consensus: yes
- Files:
  - new `x/knowledge/keeper/claim_integrity_contest.go`, containing `contestReasonRemains`,
    `lastSurvivedAt`, `disprovenAt`, a bounded relation scan, `contestAcceptedTargets` and
    `settleContradictionTargets`;
  - `msg_server.go:281-291` and `:1150-1158`: the admission flips run only when the marker is off;
  - `rounds.go`: the ACCEPT arm after `:243`, and the call sites `:258`, `:265` and `:271`;
  - `phases.go:236`;
  - new `claim_integrity_contest_test.go`.
- Failing tests:
  - Probe replica: A submitted leaves F VERIFIED. A accepted makes F CONTESTED
    (`contradiction_accepted/A`). B (minimum stake) starves and F stays CONTESTED, with no
    `contradiction_reversed` event.
  - A pre-activation flip is released: with F CONTESTED and legacy contradiction C1 pending in
    imported state, C1 starves and F becomes VERIFIED (`contradiction_reversed/C1`).
  - A cascade-contested F plus a starving contradiction leaves F CONTESTED.
  - A `SubmitClaim` with CONTRADICTS does not flip at admission, and contests on ACCEPT.
  - A conjecture-shaped target is restored to PROVISIONAL.
  - A fault-injected relation read or bound exhaustion leaves F CONTESTED and emits
    `contest_check_incomplete`.
  - With the marker off, `TestSubmitContradiction_StarvedDoesNotLockFact` and
    `TestSubmitClaim_ContradictionAutoContests` are unchanged.
- Change:
  - Implement §5.2 rows 2 and 4 and §5.3.
  - ACCEPT arm: for each CONTRADICTS target with status VERIFIED or ACTIVE, set CONTESTED and
    emit `zerone.knowledge.contradiction_accepted{fact_id, counter_claim_id, counter_fact_id}`.
  - Non-accepted arms: for each CONTESTED target, if
    `!contestReasonRemains(target, lastSurvivedAt)`, set it to VERIFIED (PROVISIONAL for a
    conjecture) and emit the existing `contradiction_reversed`.
  - `hasOtherLiveContradiction` is not used when the marker is on.
  - `SetFact` errors are returned where `completeRound` already returns errors. They are logged
    in the starved branch for legacy-policy rounds.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 13 — Participants decide the target; reversal follows the latest finding (P3c)** · consensus: yes
- Files: `msg_server.go:770-773`, `rounds.go:1212` and `:1284`, `vindication.go`
  (`handleChallengeDisproven`, after the DISPROVEN write and the cascade), new
  `claim_integrity_verdict_test.go`.
- Failing tests:
  - `ChallengeFact` accepts a CONTESTED target; with the marker off it refuses.
  - With F CONTESTED while K stands:
    - REJECT leaves F CONTESTED;
    - ACCEPT makes F DISPROVEN;
    - a starved challenge leaves F CONTESTED, so there is no laundering.
  - Cascade-contested F (dependency refuted at h1):
    - REJECT at h2 > h1 makes F ACTIVE;
    - a starved challenge leaves F CONTESTED;
    - after a REJECT, a later starved challenge does not re-contest F on account of the h1
      dependency.
  - Challenging K with ACCEPT makes K DISPROVEN and F VERIFIED (`counterfact_disproven/K`, with
    the event attribute `reverted_by_fact`).
  - With two standing counter-facts, disproving one leaves F CONTESTED.
  - The `conjecture_standing_test.go` suite stays green.
  - The marker-off twin is unchanged.
- Change, when the marker is on:
  1. The status gate also admits CONTESTED. The minimum stake still uses the target's confidence.
  2. On survival, the status is `RestoredStatusFor(F)` unless
     `!IsConjecture(F) && contestReasonRemains(F, survivedAt = current height)`, in which case it
     is CONTESTED.
  3. On an inconclusive, malformed or starved challenge, the same rule applies with
     `survivedAt = lastSurvivedAt(F)`.
  4. After K is disproven, settle each outgoing CONTRADICTS target of K, inside the same cache.
  - The survival reward is still released on REJECT, as today. No economic change.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

**Task 14 — Exclude conflicted reviewers (P5)** · consensus: yes
- Files: `state.go`, `StoreCommitmentInRound` `:634-694`; new
  `keeper/claim_integrity_review.go` with `conflictedParties` and `sameActor`; new
  `claim_integrity_review_test.go`; an app test for the vote-extension path (`app/abci.go:516-538`).
- Failing tests:
  - the submitter's commit is refused on both the tx path and the direct
    `StoreCommitmentInRound` path;
  - the valoper encoding of the submitter's key is refused;
  - the defender (the submitter of the target fact) is refused on a `ChallengeFact` round and on
    a `MsgSubmitContradiction` round;
  - a `SubmitClaim` carrying 16 CONTRADICTS relations excludes **only** its submitter;
  - a self-commit made before activation still reveals and settles after activation;
  - the vote-extension path skips the commit and logs it;
  - with the marker off, self-review is still allowed.
- Change. When the marker is on, after the phase and scheme checks:
  - load the claim with `getClaimRecordChecked`;
  - the parties are the claim's submitter, plus:
    - the submitter of the `ProvisionalFactId` fact, when one is set;
    - otherwise, the submitter of the target fact, when the claim is
      `FundingTerms.Kind == CHALLENGE_DEPOSIT` and carries exactly one relation, of type
      CONTRADICTS;
  - `sameActor` means string equality, or equal bytes after `bech32.DecodeAndConvert`, whatever
    the HRP;
  - refuse with `ErrUnqualifiedVerifier.Wrapf("reviewer %s is a party to claim %s (%s)")`;
  - reveals are not guarded.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.

### Phase D — documentation and verification

**Task 15 — Specification** · consensus: no
- Files: new `docs/specs/knowledge-claim-integrity-v1.md`, covering §3–§5, the cause vocabulary,
  the events, the reviewer rubric, the quote guard algorithm, and what the chain cannot prevent
  (§11). Also add a pointer from `docs/specs/knowledge-claim-records-v1.md`: after v12,
  admission no longer flips the target.
- Failing test: none (documentation).
- Acceptance: every behaviour claim in the spec cites a test name from Tasks 3–14.

**Task 16 — End-to-end probe re-run and upgrade rehearsal** · consensus: no (verification)
- Files: nothing tracked. Artifacts go under `/tmp/zerone-sandbox-<date>-v12/`.
- Acceptance: §7 in full. Record observed versus expected for each line, and report any line that
  cannot run as BLOCKED with the reason. A missing check is not a pass.

---

## 7. End-to-end acceptance checklist (fresh sandbox built from the branch)

Setup:
- Build with `go build -o /tmp/zerone-sandbox-<date>-v12/bin/zeroned ./cmd/zeroned` from the
  worktree.
- Run `python3 scripts/local-node.py init --home /tmp/zerone-sandbox-<date>-v12/home --binary … --chain-id zerone-local-claims-v12 --knowledge-profile --fast-review`.
- Run every CLI call through a wrapper that sets `HOME=<fake home>` and an explicit `--home`, as
  in the 2026-10-16 `tools/z`.
- Onboard `reviewer4`, so that enough reviewers remain once the parties are excluded.
- Read `~/.zeroned` and `~/.zerone-nodes` only with `stat`, before and after; they must be
  unchanged. Never touch a live network.

| # | Probe (2026-10-16 artifact) | Observed on a4272bf | Expected on the branch |
|---|---|---|---|
| 0 | Genesis | — | `knowledge.claim_integrity_enabled=true` in genesis.json. An ABCI store query for key `0x87` returns `01`. |
| 1 | P1: X starves, then the same text and domain are resubmitted (`p1-X-resubmit-same-domain-r2`) | code 1, `duplicate claim … 1a68b4b7…` | code 0, with a new claim ID. `claim-workflow --resubmit-after <hash> --max-review-fee-uzrn N` commits. A second resubmission while the first is live gets code 1, naming the live ID and its status. |
| 2 | P1ext: a contradiction carrying X's text (`p1ext-X-via-contradiction-r2`) | code 0, and 0x09 is overwritten | While X is live: code 1 `duplicate claim`, and 0x09 still names X. After X is INSUFFICIENT: code 0 (stake ≥ minimum), and 0x09 names the contradiction. |
| 3 | P2 B1: object 78.4 vs 81.0 (`p2-B1-assert-dup-r2`) | code 1 `canonical duplicate` | code 0. An identical structure gets code 1. `5 mW` and `5 MW` are both admitted. |
| 4 | P2 B3 / S1: canonical form without a structure (`p2-B3…`, `p2-S1…`) | code 0 (unchecked, a squat) | code 1 `canonical form requires claim structure`. The structured S2 claim gets code 0. |
| 5 | P2: `FactByCanonical` with the v2 form of an accepted claim | (v1: not checked) | Returns the fact. |
| 6 | P3: A submitted against F (`p3-F-fact-after-A-submit`) | F status 5 (CONTESTED) | F stays 3 (VERIFIED). |
| 7 | P3: A accepted (`p3-F-fact-after-A-accepted`) | F 5; no causal record | F 5. The transition reads `contradiction_accepted` with cause_id A. Counter-fact K is VERIFIED. |
| 8 | P3: B with 1 uzrn (`p3-B-contradiction-1uzrn-r4`) | code 0 | Refused, with "below effective minimum". At the minimum stake: code 0, F unchanged. |
| 9 | P3: B starves (`p3-F-fact-after-B-expiry`, blocks 829-831) | F 3 (VERIFIED); `contradiction_reversed` | F stays 5. No `contradiction_reversed` event. |
| 10 | P3: an unregistered account submits a contradiction | admitted | Ante refusal: "not registered with zerone_auth". |
| 11 | P3: `ChallengeFact` on F while F is CONTESTED | refused (status gate) | Admitted, F goes to CHALLENGED (`challenge_opened`). If it starves, F goes back to CONTESTED (`challenge_inconclusive`). |
| 12 | P3: challenge K; three non-party reviewers accept | — | K DISPROVEN (`challenge_disproven`). F VERIFIED (`counterfact_disproven/K`), with an event carrying `reverted_by_fact=K`. |
| 13 | P5: reviewer1 submits T and commits on it (`p5-T-round`) | commit accepted and paid | reviewer1's commit is refused with "is a party to claim". reviewers 2–4 complete the round, and only non-parties are paid. On the challenge-of-F round, F's submitter's commit is refused. |
| 14 | P6: F history (`p3-F-history-after-B-expiry`) | `cascade` with an empty cause_id; `verification` carrying F's own claim ID | Each transition has the exact cause and ID from §5.2. No empty cause_id on any contradiction or challenge transition. |
| 15 | P4: a DOI in `references` (`p4-refs-doi-r1`) | refused | Unchanged (refused). A DOI in structure or tags is still accepted. |
| 16 | Attribution round trip | — | `claim-workflow attribute --dry-run` prints the §4.2 fields. Broadcast proposition P and attribution A1. `ClaimHistory(P)` lists A1 under `structure.object`. A variant carrying 8 excerpt words is refused before signing, and the excerpt does not appear in any artifact. |
| 17 | Export and import | — | `zeroned export` on the stopped sandbox shows the flag as true. Import into a fresh home starts, and has no `migration_v12_complete` (no invented receipt). |
| 18 | Upgrade rehearsal (required before any dev-1 scheduling; recommended for this branch) | — | Start a v11 sandbox from an a4272bf binary. Pass a gov plan `knowledge-claim-integrity-v1` at H with empty info. The chain halts at H. Swap in the branch binary. At H: `0x87=01` and `migration_v12_complete=true`, and the export diff against H−1 shows only those keys plus the version map and heights. The branch binary refuses a v11 home that is not at H−1. Repeat from a 5542b22 sandbox if local-node supports it; otherwise record BLOCKED. |
| 19 | Isolation | `version` wrote `$HOME/.zeroned` | `stat` of the real `~/.zeroned` and `~/.zerone-nodes` is unchanged. The fake HOME has no `.zeroned` from the version probe. |

---

## 8. Out of scope (this branch)

- **Query energy never reaches metabolism.** `fitness.go:134` resets the counter before
  `metabolism.go:167` reads it. This is an economic consensus change and needs its own marker.
- **`method_id` on `MsgSubmitContradiction`** (`tx.proto:296-307`). It is a tx proto change, so
  slice 2.
- **Gating `MsgPostConjecture` in the ante.** That changes who may ask questions.
- **Further identity limits.** Reviewer independence beyond same-address and single-target
  defender exclusion: sybil accounts and a DID controller link (needs an auth proto field).
- **Dispute economics:**
  - full refund of a starved deposit (`fund_settlement.go:137-153`);
  - a per-target challenge cooldown;
  - a contradiction cooldown (unnecessary once admission buys nothing).
- **Metabolic re-checks.** Re-deriving CONTESTED when an AT_RISK target is revived, or when a
  counter-fact decays through metabolism instead of being disproven. The exit is a direct
  challenge.
- **Exempting attribution facts** from niche competition or metabolism.
- **Making AT_RISK, EXPIRED or PRUNED facts challengeable.**
- **Canonical form housekeeping:** cross-dedup between v1 and v2 forms, and repairing legacy
  index pointers.
- **On-chain quote detection.** No excerpt is available on chain.
- **A standing fold on the client side** (`zerone.standing/v1`), and a `PropositionStanding`
  query or index (a proto change).
- **The zerone-1 upgrade path.** Also any dev-1 deploy tooling: gateway allowlist, runtime
  checks, `network-knowledge-12.json`.
- **Rewriting historical transitions or records.**

---

## 9. Decisions that need Yu

Each has a default the plan implements. Changing one alters only the named tasks.

- **D1. When a contradiction takes effect.** Default: **on acceptance** (Task 12).
  - The alternative keeps the flag at admission for staked `MsgSubmitContradiction`. That
    restores an immediate "disputed" signal, but also a refundable-capital grief window,
    especially on reviewer-less chains. It would need the live-claim scan, bounded through
    `ActiveRoundIndex`.
  - Either way, `SubmitClaim`+CONTRADICTS no longer flips at admission.
- **D2. What an accepted contrary finding does.**
  - Default: **asymmetric**. The target becomes CONTESTED; the counter-fact stays VERIFIED.
  - Default: a target that survives a direct challenge **stays CONTESTED** while the counter-fact
    stands. The counter-fact must itself be refuted.
  - Alternatives: symmetric CONTESTED, which freezes the counter-fact out of fitness; or letting
    survival clear R1.
- **D3. What releases text for resubmission.**
  - Default: no-verdict, negative-verdict and fallen states all release it (§5.4).
  - Minimal alternative: only INSUFFICIENT and EXPIRED.
- **D4. zerone-dev-1.**
  - Whether to upgrade it in place, which bundles the 64-commit `5542b22..a4272bfa` delta and
    needs that delta audited first, or relaunch it from a fresh genesis.
  - The height H2, the proposer and the voting path.
  - Who reviews. dev-1 has never had reviewers, and after P5 a submitter cannot review their own
    claim, so every claim will starve (now harmlessly) until there are `MinVerifiers` independent
    reviewers.
- **D5. Hermes contract** (to agree with the Hermes builder):
  - the attribution `object` key: the on-chain content hash (default) or the Hermes Proposition
    hash;
  - JCS plus sha256 with a `sha256:` prefix;
  - the `reasoning_trace` keys;
  - the locator grammar.
- **D6. Licensing and privacy policy.**
  - Default: zero verbatim text on chain, with an 8-token guard threshold.
  - Default: private persons appear only as opaque `hermes:` IDs.
- **D7. Attribution methodology.** Default: `M-HISTORICAL`. The alternative is a
  governance-registered `M-ATTRIBUTION`.

---

## 10. Rejected alternatives (from the designs and judges)

- **D2's 8-byte activation height in 0x87, gas-free marker reads, and the "vintage" rule.**
  - The height breaks the sibling `{1}` convention and is lost on export.
  - Vintage handling is unnecessary: pre-activation flips settle through the same predicate, and
    any accepted counter-claim contests again.
- **D2's re-proposal surcharge.** A one-character change bypasses it, it taxes reversal
  (decision 3), and it is a griefing lever.
- **D2's keeper-level identity gate on five messages, with a new `ZeroneAuthKeeper` method.** Its
  authz rationale does not apply: no authz keeper or message server is wired (`app.go:391,565`).
- **The cascade reason from `CascadeEventByDescendant`.** The cascade skips descendants that are
  already CONTESTED or CHALLENGED, so the reason would be missing. Structural edges are used
  instead.
- **Clearing the cascade reason on any non-refuting challenge outcome.** That reopens laundering
  through a starved self-challenge. Only a REJECT survival clears it, ordered by height.
- **D2/D3 defender exclusion derived from a multi-target CONTRADICTS list.** It enables a recusal
  attack.
- **D3's `RebuildClaimDedupIndexes` migration.** It risks halting at H, and it would repoint 0x34
  fact IDs back to claim IDs.
- **D3's AGGREGATION retry when the scan bound is hit.** That is a liveness hazard. The plan fails
  toward CONTESTED instead.
- **D3's retagging of all 22 status-write sites, with "unattributed".** Only the sites this slice
  touches are tagged; the rest keep inference.
- **D3's AT_RISK challengeability, PostConjecture gating, and a client standing fold.** All scope
  creep.
- **D1's `setDedupIndex` with extra reads.** The first-save rule gives the same guarantees with
  zero extra reads.
- **D1's admission-time flip with `hasOtherLiveContradiction`.** Replaced by D1 (§9). It also
  kept an unbounded full-claim scan on BeginBlock.
- **Deleting dedup index entries.** Replay safety prefers reads that check status.
- **Line-oriented `key=value` reasoning traces.** JCS JSON matches the Hermes side.

---

## 11. What the chain still cannot prevent (to disclose in the spec)

- **One controller holding many DIDs.** Registration is permissionless and there is no controller
  field. For `CHALLENGE_DEPOSIT` disputes, sybil review costs nothing net: 45% is refunded
  whatever the verdict, and the 55% pool pays the sybils. For `REVIEW_FEE` claims, sybil
  self-review loses 45% of the fee.
- **Deliberately starved disputes.** A starved challenge or contradiction is refunded in full, so
  the cost is locked capital and gas. After this branch a starved dispute changes no status.
- **Mempool copying of claim text.**
- **Verbatim quotes from third-party clients.** Only reviewers marking MALFORMED push back on chain.
- **Reviewer independence and the truth of anything a panel accepts.**
