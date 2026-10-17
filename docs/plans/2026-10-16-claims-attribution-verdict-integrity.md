# Claims: attribution, verdict and reversal integrity (knowledge v12)

Status: Tasks 1–16, review fixes G1, G1R and G2–G5, a cleanup pass and the import-index fix
from the final acceptance run (row 10; 2026-10-17) are implemented and committed on the branch,
which is not yet merged. Task 16's end-to-end run (at
`cb4d7dfd`, before the review fixes) passed; it has not been re-run since. Pending Yu's
approval: three re-pins (authority geometry, spec hashes, the SDK's generated codecs), behind
the only checks that still fail (§6, "Deferred checks"). Decisions (a) and (b) were taken by
Claude and are reversible (§9, D1–D3).
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
    Settled in Task 12: true on knowledge 11 (the marker-off twin of
    `TestClaimIntegrity_CascadeContestedTargetStaysContested` restores the fact to VERIFIED).
    With the marker on, R2 keeps it CONTESTED.
  - FactByCanonical misses accepted structured claims. `rounds.go:299` re-saves the claim after
    `rounds.go:799` pointed 0x34 at the fact, and that re-save points it back at the claim ID.
  - Genesis import order decides which duplicate the 0x09 index points to.
    Settled in Task 8 (review): true, and with status-aware checks it let a duplicate in after
    export and import. With the flag on, `InitGenesis` now replays claims in admission order;
    `TestClaimIntegrity_ImportReplaysAdmissionOrder` covers it.
  - An attribution niche can be pruned once it holds more than 10 facts.

Each inferred item has either a test that settles it or a note that it is out of scope.

---

## 1. Goals, mapped to Yu's decisions (2026-10-16)

| Yu's decision | What the chain must guarantee after this branch | Defects that currently break it |
|---|---|---|
| **(1) Who said what is recorded on chain** | (a) An attribution record (work, voice, act, proposition, locator) can be submitted, reviewed and found. It refers to Hermes records by hash only and never stores verbatim text. (b) Every status change of a fact records who or what caused it. | P2 (distinct statements collide, canonical forms can be squatted). P6 (wrong causes). P1 (a starved attribution locks its text). |
| **(2) The verdict on a proposition is a separate decision by chain participants** | (a) A fact's status changes only through a panel decision on that fact, or through an accepted finding that conflicts with it or that it depends on. No single submitter can change it. (b) A submitter, or the defender of the target, cannot sit on the panel. (c) Accepting an attribution never changes a proposition's verdict. | P3 (a 1-uzrn admission flips status; a starved round restores it). P5 (self-review). P3: CONTESTED facts cannot be challenged. |
| **(3) Verdicts can be reversed by the latest findings** | (a) A CONTESTED fact has a decision path: ChallengeFact accepts it. (b) When a counter-fact is disproven, its target is re-evaluated. (c) If a fact survives a direct challenge, that clears doubts raised by dependencies refuted before the challenge opened (review fix SEC-1). (d) A rejected, starved or fallen proposition can be re-proposed as a new record, with the old one kept. | P1 (permanent lock). P3 (no exit from CONTESTED; no re-evaluation after a counter-fact falls). |

Non-goals for this slice: zerone-1, metabolism economics, new tx fields, on-chain quote detection.
The full list is in §8.

---

## 2. Design summary

The backbone is design D1: minimal, consensus-safe and upgrade-first. All three judges ranked
it highest. It is changed in these ways:

- **D2's idea: contradictions take effect when accepted, not when admitted.**
  `MsgSubmitContradiction` does not change the target's status at admission, and after
  activation a `MsgSubmitClaim` may not carry CONTRADICTS at all (decision (a), review G2), so
  every new contradiction goes through the contradiction admission checks (Task 10). When the
  counter-claim is **accepted**, the target becomes CONTESTED. This removes
  the cheap grief window (a refundable deposit or a 0.1 ZRN fee buys a 450-block CONTESTED
  window). It also takes the unbounded "is any claim still live" scan
  (`rounds.go:972-997`) off every new BeginBlock path. Yu may reverse this choice (§9, D1).
- **Whether a fact stays CONTESTED is computed from its relation edges, not from events.**
  - R1: an accepted contradicting fact that is still standing.
  - R2: a fact it depends on was refuted at or after the height the fact's last survived
    challenge opened (review fixes SEC-1, F1).
  This avoids D2's `CascadeEventByDescendant` gap: the cascade skips descendants that are
  already CONTESTED or CHALLENGED (`vindication.go:619-627`).
- **D2's survival rule, made order-aware.** Surviving a direct challenge (verdict REJECT) clears
  R2 only for dependencies refuted before that challenge opened (review fix SEC-1: not before it
  ended, so a refutation while the challenge was open is never cleared by it). An inconclusive
  or starved challenge clears nothing, which closes judge 2's laundering path.
- **The scan never errors toward VERIFIED.** If the contest-reason scan errors or hits its bound,
  no CONTESTED fact is released and an event is emitted. That holds at a challenge's end too:
  an incomplete check leaves the fact CONTESTED (review G1R, which reverts G1's SEC-6
  fallback). R2 is read before R1, a reason either finds is conclusive, and R2 reads a
  claim-born fact's dependencies by point lookups from its claim, so it never reads what others
  write under the fact's prefix. A BeginBlock path never receives an error.
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
- No record is rewritten, there is no index rebuild and no inventory scan. (An import rebuilds
  the subject and tag indexes, §3.6; the in-place upgrade keeps the live entries.)

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
| REFORMULATES edges from pre-activation augmentation verdicts, including IDs that extend (`<F>/x`) or equal a fact's ID | Kept, a closed set. R2 reads a claim-born fact's edges by key from its claim; the prefix scan of a claimless fact skips foreign entries (§5.3). The exception: an equal ID that replaced an existing fact's own edge, which the check then reads (residual in §8). After activation the writer refuses such IDs (review G1R), except the ID of a fact created after the verdict (§8). |
| Subject (0x32) and tag (0x33) index entries | Kept by the in-place upgrade. Only a round writes them (`IndexFactBySubject` at fact creation) and the export carries none, so on knowledge 11 (and on this branch before the fix) an import restored every fact with its tags but no entry, and `FactsByTag`/`FactsBySubject` answered nothing (final acceptance row 10; pre-existing at a4272bf). With `gs.ClaimIntegrityEnabled`, `InitGenesis` now rebuilds both for every imported fact with a structure, with `IndexFactBySubject` itself, in creation order: `verified_at_block` (a round-created fact's creation height, never rewritten), then the round ID its claim names (`AdvanceRoundPhases` completes a block's rounds in round-ID order), then the fact ID. So the single-valued subject entry names the fact created last, as live, and an export's entries come back exactly. Residuals: a structured fact from a genesis file is indexed although its original chain never indexed it; facts of one block whose claims the file does not list are ordered by fact ID (an export lists every claim). With the flag off nothing is rebuilt, byte for byte as at a4272bf. See "Final acceptance fix" in Phase D. |
| Historical status transitions | Never rewritten. |
| Content hashes shared by two records admitted before activation (P1ext: knowledge 11 let a `MsgSubmitContradiction` carry another claim's exact text) | Not rewritten. Knowledge 11 rewrote 0x09 on every save, so the entry names the record **saved last**, and the live chain decides dedup from that record's standing. An import replays first saves in admission order (Task 8), so there the entry names the record **admitted last**. When the two records differ in standing, the live and the imported chain disagree, in either direction: one admits a third copy of a standing fact's text and the other refuses it (review CI-REVIEW-1; both pinned by `TestClaimIntegrity_LegacyContentCollisionLiveVsImport`). After activation every admission checks 0x09 first, so no new pair arises. D4 adds a pre-upgrade check (§9). |

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
| `structure.subject` | work ID, lowercase: `doi:<doi>`, `arxiv:<id>v<n>`, `isbn:<13>`, or `url-sha256:<64hex>` | The source document anchors the record. A DOI's suffix is one or more of `a-z0-9:._/;=+()<>#-`: parentheses for DOIs such as `doi:10.1016/s0140-6736(20)30183-5`, and `<`, `>`, `#` for SICI DOIs such as `doi:10.1002/(sici)1097-4571(199806)49:8<693::aid-asi4>3.0.co;2-0` (review G4, F3). |
| `structure.predicate` | `<act>:<voice>`. act ∈ {`states`, `reports`, `hedges`, `denies`}. voice ∈ {`author`, `quoted:<speaker-ref>`} | `speaker-ref` is `orcid:…`, `wikidata:q…`, or `hermes:<64hex>` (opaque). Private persons appear **only** as opaque `hermes:` refs. |
| `structure.object` | `zrn-content:<64hex>` | The proposition claim's on-chain content hash: `sha256("ZRN.claim.v1:" + domain + ":" + text)` (`state.go:571-578`). It can be computed before the proposition exists. |
| `structure.scope` | `ver=sha256:<64hex>;loc=<locator>` | The digest of the retrieved source bytes (Hermes SourceVersion), plus the Hermes locator in a lowercase grammar such as `p4.para2`, `sec3.2.para4`, `fig2`, `t=00:12:31`. It contains no text. |
| `structure.temporal_scope` | publication date of the pinned version (`YYYY-MM-DD`) | Informational. |
| `structure.tags` (≤10, each ≤50 chars) | `attr-v1`, `stmt:<first 40 hex of the StatementOccurrence hash>`, `zc:<first 40 hex of object hash>` | `FactsByTag` joins accepted attributions. Tags are indexed only when a fact is created. |
| `fact_content` (20–1000 B) | fixed template: `Attribution zerone.attribution/v1: <subject> <predicate> <object> at <scope>` | No free text, so no excerpt can get in. The content hash identifies the tuple. With the marker on, the chain refuses any other record text that opens with `Attribution zerone.attribution/v1:`, and any record domain that contains `Attribution zerone.attribution/v1` (review G2, SEC-4; limits in §4.8). |
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
- **Jones 2021 reports a different value (81.0).** P2 goes on chain in one of two ways, not
  both. Either way, then submit attribution `A2` with `subject doi:10.1000/p6-jones-2021`,
  `predicate reports:author` and `object zrn-content:<hash of P2>`.
  - To record P2 beside P1, submit a separate structured proposition `P2` with object `81.0`.
    After v12 it no longer collides with P1.
  - If Jones's finding should contest P1, submit P2 **instead** as `MsgSubmitContradiction`
    against P1's fact, staking at least the minimum for challenging P1 (Task 10). Its
    counter-claim text is P2's text, so its content hash is the one A2's `object` names. P1
    becomes CONTESTED only if P2 is accepted (§5). After activation a `MsgSubmitClaim` may not
    carry a CONTRADICTS relation (decision (a), review G2).
  - The trade-off (review G2 re-review). `MsgSubmitContradiction` carries no structure,
    `method_id` or `reasoning_trace`, so a contesting P2 has no form v2 and no 0x34 entry; A2
    names only its content hash. The two ways exclude each other: while a structured P2 is live
    or stands, a contradiction with P2's text is refused as a duplicate (or, from the same
    account in the same block, as `contradiction claim identity already exists`), so a
    structured P2 cannot later contest P1 with its own text. Test:
    `TestClaimIntegrity_StructuredPropositionCannotAlsoContest`.
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

- Attribution structure fields must match `^[a-z0-9][a-z0-9:._/;=+()<>#-]*$` (`(`, `)`,
  `<`, `>` and `#` were added for DOIs; review G4, F3).
- The excerpt is read from a local file and never leaves it.
- Any on-chain text (fact_content, reasoning_trace, structure fields, contradiction text,
  challenge or review reason, challenge or review evidence, review scope) is refused if,
  after normalisation, it contains:
  - any window of **8 consecutive excerpt tokens**; or
  - the whole excerpt, when the excerpt has fewer than 8 tokens; or
  - a run of consecutive excerpt **units weighing 16**, or every excerpt unit in order when
    the excerpt weighs less (review G4, TQ-2). Units split each token again: every letter of
    Han, Hiragana, Katakana, Thai, Lao, Khmer or Myanmar (scripts written without spaces
    between words, listed as Unicode blocks and checked after NFKD) is its own unit,
    weighing 1; the rest of a token is one unit, weighing 2. Those scripts put a whole
    clause into one token, so a run-in or trimmed quote never matched the 8-token rule.
    16 characters is about 8 words of Chinese or Japanese (roughly 2 characters a word);
    a Thai, Lao, Khmer or Myanmar word keeps several letters after mark deletion, so there
    16 letters is fewer words and the rule is stricter, erring toward refusal. A word
    weighing 2 keeps 8 spaced words at 16, so text without these letters is judged exactly
    as before, and a spaced quote glued to such a letter (which used to lose its first
    word) is still caught. Other unspaced scripts (Tai Tham, Balinese, Javanese, …) are not
    split.
  Normalisation is applied identically to the excerpt and to each field: NFKD, casefold, NFKD;
  then **delete** every combining mark (M\*), every format character (Cf) and every
  Default_Ignorable_Code_Point (soft hyphen, ZWSP/ZWJ/ZWNJ, word joiner, BOM, bidi controls,
  variation selectors, Hangul fillers, tags); then every remaining character that is not a
  letter (L\*) or decimal digit (Nd) becomes a space; then split on whitespace. Deleting, rather
  than spacing, means a hidden character cannot split a word. Folding to base letters means a
  composed, decomposed or unaccented copy of a quote matches the same window. (Changed after
  the Task 4 review; it was NFKC with every non-letter/non-digit spaced, so a soft hyphen hid
  a quote. See "Follow-up to Task 4".) The first NFKD folds compatibility capitals that
  casefold alone keeps (`ᴬ`, lunate sigma `ϲ`); the second changes no token under Unicode
  14.0–16.0 (checked over every code point) and stays against later casefold mappings.
  Category data comes from the running Python's `unicodedata` (Unicode 14 on 3.11); a code
  point assigned in a later Unicode version can tokenize differently between runtimes.
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
- The template is reserved on chain (review G2, SEC-4): with the marker on, `MsgSubmitClaim`,
  `MsgPostConjecture` and `MsgSubmitContradiction` refuse text that opens with
  `Attribution zerone.attribution/v1:` unless the message is a `CLAIM_TYPE_RELATION` claim whose
  structure renders exactly that text. Two limits remain:
  - The rendering is one-to-one only when subject, predicate and object hold no space, as the
    §4.7 grammar requires. The chain checks only that the structure renders the text. Another
    RELATION claim with the same four fields (a different date or tags, say) is the same tuple
    and is a duplicate under the usual first-admitted rule.
  - The domain is checked too (review G2 re-review). The preimage
    `"ZRN.claim.v1:" + domain + ":" + text` is ambiguous: a domain that swallows the head of an
    attribution's text up to any `:` in it (`chemistry:Attribution zerone.attribution/v1`, or
    `chemistry:Attribution zerone.attribution/v1: doi` with text `10.1000/…`) gives the
    attribution's hash with text the prefix check never sees. So the three messages also refuse
    a domain that contains `Attribution zerone.attribution/v1`; for a contradiction that is the
    domain its hash uses (its own, or its target's when it names none), and it is checked
    although `MsgSubmitContradiction` never checks that its domain exists. Every split lies at
    or after the name's `:` when the attribution's own domain holds no `:`, which the
    encoder's DOMAIN grammar `[a-z0-9][a-z0-9_.-]{0,127}` guarantees, so this closes every
    split of an encoder-made attribution. The chain does not enforce that grammar: a
    third-party attribution in a domain holding `:` can still be split at that `:`.
  - The ambiguity itself remains for every other text: a record in domain `d:x` with text `y`
    has the hash of a record in domain `d` with text `x:y`. Domain names are not validated,
    `MsgProposeDomain` needs no stake, a PROPOSED domain admits claims, and
    `MsgSubmitContradiction` never checks that its domain exists, so any text containing `:`
    can be squatted this way (review G2's probes confirmed it on all three paths). Closing it,
    by refusing `:` in domain names or by a length-prefixed preimage (which changes every hash
    and needs its own design), is a separately scheduled item.

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
| — | VERIFIED (PROVISIONAL for a conjecture) | claim ACCEPTED | `verification / <claim id>` (unchanged for VERIFIED; new for PROVISIONAL, which knowledge 11 recorded as `unknown`) |
| VERIFIED, ACTIVE | CONTESTED | a claim carrying CONTRADICTS→F is **ACCEPTED** and the edge stored from its counter-fact K to F is CONTRADICTS, the edge R1 reads (one edge per pair; the last relation written wins, so `[CONTRADICTS F, CITES F]` stores CITES and contests nothing). After activation only `MsgSubmitContradiction` admits such a claim; a `MsgSubmitClaim` carrying CONTRADICTS is refused (decision (a), review G2), so one reaches this row only if admitted before activation. | `contradiction_accepted / <counter-claim id>` |
| VERIFIED, ACTIVE, AT_RISK | CONTESTED | a fact F depends on is DISPROVEN (existing cascade) | `cascade / <disproven fact id>` (unchanged) |
| CONTESTED | VERIFIED | a contradiction ends non-accepted (REJECT, MALFORMED, INCONCLUSIVE, starved) **and** no contest reason remains | `contradiction_reversed / <claim id>` |
| CONTESTED | VERIFIED | counter-fact K becomes DISPROVEN and no contest reason remains | `counterfact_disproven / <K>` |
| VERIFIED, ACTIVE, **CONTESTED** | CHALLENGED | `MsgChallengeFact` | `challenge_opened / <challenge claim id>` |
| CHALLENGED | DISPROVEN | challenge ACCEPT | `challenge_disproven / <challenge claim id>` |
| CHALLENGED | ACTIVE or CONTESTED | challenge REJECT (survived). CONTESTED iff R1 holds, or R2 holds for a dependency refuted at or after the height this challenge opened (`survivedAt` = its claim's `SubmittedAtBlock`; review fix SEC-1). | `challenge_survived / <challenge claim id>` |
| CHALLENGED | ACTIVE or CONTESTED | challenge INCONCLUSIVE, MALFORMED or starved. CONTESTED iff R1 or R2 holds. Nothing is cleared. | `challenge_inconclusive / <challenge claim id>` |
| metabolic statuses | — | unchanged | inferred, as today |

A check that cannot finish at a challenge's end returns F to CONTESTED, whatever F's standing
before the challenge, and emits `contest_check_incomplete` (review G1R). G1's SEC-6 fallback,
which restored the standing before the challenge, is reverted: with it, any read an attacker
can inflate became a laundering lever (§5.3). A reason found before the check stops is
conclusive (§5.3).

Rules that do **not** change any fact:
- Admitting a contradiction.
- Accepting an attribution.
- Accepting a counter-claim, for the counter-claim's own target's *verdict*. The target only
  becomes CONTESTED; it is never made DISPROVEN.

DISPROVEN can only come from a direct challenge on that fact.

### 5.3 Contest reasons (`contestReasonRemains(ctx, F, survivedAt) bool`; it never returns an error)
- **R1 — standing conflict.** Some incoming relation to F has type `CONTRADICTS`, and its source
  fact's status is one of VERIFIED, ACTIVE, CONTESTED, CHALLENGED, AT_RISK or EXPIRED. EXPIRED
  counts because patronage or metabolism can revive it (decision (b), review fix SEC-5); only
  SUPERSEDED, DISPROVEN, REVOKED and PRUNED have fallen.
- **R2 — refuted dependency.** Some outgoing relation from F has type SUPPORTS, REQUIRES, REFINES,
  GENERALIZES or CITES (the cascade set). Its target D is DISPROVEN, and
  `survivedAt == 0 || disprovenAt(D) >= survivedAt`, where:
  - `disprovenAt(D)` is the block height of D's latest transition to DISPROVEN, or 0 if none is
    recorded;
  - `survivedAt` is the evidence cutoff of F's last survival: the height its challenge opened.
    It is passed in (the surviving challenge's `SubmittedAtBlock`), or else read from the
    challenge claim that F's latest `challenge_survived` transition names (0 if there is none).
    A survival whose claim is missing, unreadable, names another fact or opened after the
    survival makes the check incomplete. Review fix SEC-1: judging from the survival's own
    height cleared a dependency refuted while the challenge was open.
  - A fall at the cutoff height counts (`>=`; review fixes F1, CI-REVIEW-2). Heights cannot
    order two events of one block, so a tie goes to CONTESTED. The survival and every later
    check use the same cutoff and the same comparison, so a survival never clears something
    that a later starved challenge would re-contest, nor the reverse.
- **Order and conclusiveness.** R2 is read first, then R1, within one budget. A reason found by
  either rule is conclusive, whatever else could not be read. Inside each rule a failed item (a
  malformed or unreadable record, an edge naming a missing fact) is recorded and the rule goes
  on, so a reason found after it is still conclusive (review G1R). R1 reads every incoming edge
  of every type, which any later fact can add, so it goes second (review G1, second pass).
- **How R2 reads (review G1R).** For a fact accepted from a claim, R2 reads that claim and, for
  each target the claim names, the stored edge `F→target` by key (`FactRelationKey`). The stored
  edge's type decides, as it decides the cascade, so R2 sees exactly what a scan of F's forward
  prefix would see, restricted to the targets F's claim names; an entry found by key whose payload
  names another pair (keys are `<source>/<target>`, so F→`x/y` and `<F>/x`→y share one) is not F's
  edge. F's claim wrote every outgoing edge of F. The one other writer is an augmentation whose ID
  equals F's: before activation, or after it when the verdict came before F existed (§8). Its
  REFORMULATES edge is outside the cascade set. So the point reads find every dependency a prefix
  scan would find, and never read what others add under `<F>/`. Before activation such an
  augmentation, finalized after F existed, replaced F's own edge to the augmented fact; the
  point read, the scan and the cascade then all read REFORMULATES there, so that target is no
  longer a dependency (residual in §8). R2's other reads are F's claim
  (one record its submitter wrote, at most `MaxRelationsPerClaim` relations since that cap) and
  the status histories of F and of each refuted dependency, which grow only by status changes (a
  challenge, a round, a cascade or metabolism), not by records others can add cheaply. A fact
  whose claim is missing or unreadable makes R2 fail; it never falls back to the scan. Only a fact
  with no claim (genesis, doctrine, an authority injection) has its forward prefix scanned.
- **Entries naming another fact are skipped.** A forward prefix `<F>/` can also hold edges whose
  source is not F: a passing augmentation wrote a REFORMULATES edge whose source is its
  user-chosen ID (`ApplyFinalizedAugmentationVerdict`), so an ID `<F>/x` landed under F's
  prefix. `validateToKStoredRecord` rebuilds the key from the payload, so the payload names the
  edge's owner. R2's prefix scan (claimless facts only) skips an entry whose source is not F,
  and R1 skips an entry whose target is not F, instead of failing the check (review G1, third
  pass). A skipped entry is still read and counted toward the bound. The third pass also
  claimed that such entries could hide a refuted dependency only at about 100k passing
  augmentations. That was false: the 64 MiB byte bound governs, and about 343 augmentation IDs
  of 96 KiB extending F's filled it (review G1R). With the marker on the writer no longer
  writes an edge whose source contains `/`, is longer than 128 bytes, or is a stored fact's ID
  (review G1R, §8), so such entries are limited to those written before activation.
- **Failure handling.** When neither rule finds a reason, a read error, a malformed record, or
  exceeding the 100k-record / 64 MiB bound (the same bound as `hasOtherActiveChallenge`) all
  mean the reason remains (`true`). The predicate also emits
  `zerone.knowledge.contest_check_incomplete{fact_id, reason}`, whose reason names the failed
  reads (up to eight per rule, the rest counted; a stopped scan is always named). Every caller,
  `challengeEndStatus` included, keeps F CONTESTED: an incomplete check never releases (review
  G1R). G1's SEC-6 fallback returned F to its standing before the challenge instead, and the
  final G1 re-review proved it a laundering primitive: any read an attacker can inflate, plus a
  rule that releases on an incomplete check, launders a shielded fact.
- **Disclosed limitation (replaces SEC-6).** A fact whose check cannot finish stays CONTESTED
  once any challenge of it ends, even a starved, refunded one, and no release path frees it;
  only a challenge that refutes it moves it. In practice that is a fact with more than 100k
  incoming relations of any type, or 64 MiB of them (R1 reads them all; a long-ID augmentation
  of F adds one incoming REFORMULATES entry), or a claimless fact whose forward prefix was
  inflated before activation. This is griefing, not laundering, and it is disclosed (§11). The
  remedy, an index of CONTRADICTS edges alone (a new store key) so that R1 is bounded, is out
  of scope (§8).
- **Pending contradictions are not reasons.**

### 5.4 Duplicate checks and resubmission (P1, P2)
An index entry (0x09 content hash, or 0x34 canonical hash) blocks admission only while the record
it points at is live or standing.

| Record the entry points at | Blocks? |
|---|---|
| A claim with status PENDING, PENDING_EVALUATION, EVALUATED, PROVISIONAL, IN_VERIFICATION, CHALLENGED, CONTESTED or UNSPECIFIED | yes |
| A claim with status REJECTED, MALFORMED, INSUFFICIENT or EXPIRED (the legacy no-quorum outcome, §5.1) | no |
| A claim with status ACCEPTED | The fact is resolved as `GenerateFactID(claim.Id, round.VerdictBlock)`, which is the same height as `createFactFromClaim` (`rounds.go:205-206`, `:612`). It blocks if that fact is standing. A missing round or fact fails closed: it blocks. |
| A fact with status PENDING, PROVISIONAL, VERIFIED, ACTIVE, CONTESTED, CHALLENGED, AT_RISK, EXPIRED or UNSPECIFIED | yes |
| A fact with status SUPERSEDED, DISPROVEN, REVOKED or PRUNED | no |
| An entry pointing at nothing | no |

Further rules:
- An EXPIRED fact blocks. Patronage or metabolism can revive it, and a twin admitted meanwhile
  would then stand beside it (decision (b), review G2 SEC-3; the same set R1 counts as standing,
  §5.3). An EXPIRED fact cannot be challenged (§8), so it holds its text until it is revived or
  pruned (by default 20 metabolism epochs after it expires).
- When a resubmission is admitted, its first save overwrites the index, so the index always names
  the latest admitted record.
- The usual fee and cooldown apply.
- The error text keeps its current prefix and adds the status of the record that blocks.
- `claim-workflow.py --resubmit-after <txhash> --max-review-fee-uzrn N` then works on chain with
  no client change.
- Reversing a DISPROVEN proposition is done by re-asserting it: a new claim and a new fact are
  created. The old fact keeps its history.

### 5.5 What consumers read
- **Current verdict of a fact.** Read `fact.status` and the status transitions with their
  causes. A transition's cause records the event that last changed the status, not the reason
  the fact holds that status now. A fact that a challenge returns to CONTESTED records
  `challenge_survived` or `challenge_inconclusive`, although a counter-fact or a refuted
  dependency is what holds it. A reason can also lapse with no transition of F: a counter-fact
  that is pruned, for example. So for a CONTESTED fact, recompute both rules of §5.3 to learn
  the current reason (review G5, F4):
  - **R1.** Take F's incoming CONTRADICTS edges (`FactRelations` with direction `incoming`, or
    `ClaimHistory`'s `incoming_relations`) whose target is F. One holds if its source fact is
    VERIFIED, ACTIVE, CONTESTED, CHALLENGED, AT_RISK or EXPIRED.
  - **R2.** Take F's outgoing SUPPORTS, REQUIRES, REFINES, GENERALIZES and CITES edges
    (`FactRelations` with direction `outgoing`, or `outgoing_relations`) whose source is F. A
    prefix read can also return the edge of an augmentation whose ID extends F's (§8); its
    source is not F. Keep each target D that is DISPROVEN. The cutoff is the
    `submitted_at_block` of the challenge claim that F's latest `challenge_survived` transition
    names, or 0 if there is none. D holds F if the cutoff is 0, or if D's latest transition to
    DISPROVEN is at or after the cutoff.

  When neither rule holds, F stays CONTESTED until it is released: when a contradiction of F
  ends without acceptance, or a challenge of F ends without refuting it (§5.2). The exception
  is a fact with more than 100k incoming edges of any type, or 64 MiB of them. There the
  chain's own check cannot finish, so F stays CONTESTED even when the recomputation finds no
  reason (the disclosed limitation of §5.3).
- **Evidence behind a verdict.** Use `ClaimHistory(claim_id)`, which gives:
  - rounds, commits, reveals and attestations: who reviewed, the vote, the method, the reason and
    evidence hashes;
  - the related claims, with link fields `provisional_fact_id`, `challenged_claim_id`,
    `relations.contradicts`, and the new `content_hash`, `canonical_hash` and
    `structure.object` (Task 3).
- **Who said P.** Read `ClaimHistory(P's claim id)` and take the related claims linked by
  `structure.object`. A link proves only that the claim's `structure.object` names P's content
  hash: any structured claim may set that field. Treat a linked claim as an attribution (with
  its own fidelity verdict) only when it is `CLAIM_TYPE_RELATION`, carries the `attr-v1` tag,
  has a predicate in the §4.2 grammar (`<act>:<voice>`, act in `states`, `reports`, `hedges`,
  `denies`, voice `author` or `quoted:<speaker-ref>`) and a §4.2 work ID as its subject, and its
  text is the template rendered from its structure (review G5, F4). Of these, the chain
  enforces only that template text matches, and only for records admitted with the marker on:
  text that opens with the template must belong to a RELATION claim whose structure renders it
  exactly (§4.8). The tag, the grammar and the use of the template at all are conventions it
  does not enforce. For accepted attributions only, `FactsByTag("zc:<40hex>")` also works, and
  its results need the same checks. It also works on a chain started from an export with the
  flag on, where `InitGenesis` rebuilds the tag index (§3.6); after a knowledge-11 import it
  returns nothing, and `ClaimHistory` is the only path. It lists facts in any status (no entry
  is ever removed), so read each fact's status. `FactsBySubject` returns one fact per domain
  and subject: the one created last.
- **A proposition by its structure.** Use `FactByCanonical` with the v2 form. Forms with the
  `zrn.cf.v2/` prefix are hashed verbatim after NFC and trim.

How to interpret statuses:

| Status | Meaning |
|---|---|
| VERIFIED / ACTIVE | currently accepted |
| CONTESTED | Accepted, but a standing accepted finding conflicts with it (R1), or a fact it depends on was refuted (R2). The latest transition's cause records the event that last changed the status, which need not be the current reason; recompute R1 and R2 as above. It does **not** mean false. |
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
- Before H, a conjecture's creation (UNSPECIFIED→PROVISIONAL) was recorded as `unknown` with an
  empty `cause_id`. Its claim is the fact's own `claim_id`.
- Two VERIFIED facts joined by CONTRADICTS are legacy laundering, as in probe P3. Treat them as an
  unresolved dispute. Either side can now be challenged.

---

## 6. Ordered TDD tasks

Conventions that apply to every task:
- Write the failing test first, run it, and record the red result in the task's note.
- Verification commands, run before every commit (with `GOTOOLCHAIN=auto`):
  - `go test -count=1` on the touched packages;
  - `go test -count=1 ./x/knowledge/... ./app/...`;
  - `python3 -m unittest discover -s scripts -p 'test_*.py'` when `scripts/` is touched;
  - `go test -count=1 ./tools/authority-graph/...` and `scripts/check_specs_and_plans_hashes.sh`
    (the `make test` and `make creed-check` pieces that pin sources and documents). Until the
    re-pin, each may fail only as listed under "Deferred checks" below. Any other failure in
    either is a real failure.
  - when `dashboard/` or `sdk/typescript/` is touched: `npm test`, `npx tsc --noEmit` and
    `npx tsc --project tsconfig.functions.json --noEmit` in `dashboard/`, and the steps of
    `npm run check` in `sdk/typescript/` (`check:generated`, `check:registry`, `typecheck`,
    `test`), run one by one until the re-pin because `npm run check` stops at the first. Added
    in the cleanup pass, after `cdb937e0` broke the second tsc project unnoticed. The same rule
    applies: only the failures listed below are expected.
- Deferred checks (pending Yu's approval to re-pin; review G5, TQ-6; the third item was added
  in the cleanup pass, 2026-10-17, which ran every command below). These are the only failing
  checks known on the branch:
  1. Authority geometry. `go test ./tools/authority-graph/...` fails
     (`TestReportSucceedsWithDeclaredConflictsVisible`,
     `TestTargetGateRefusesDeterministically`) with `SOURCE_ANCHOR_SHA256_MISMATCH` for two
     source anchors in `dashboard/public/standards/authority-geometry.v1.json`:
     - `knowledge-writers` (`x/knowledge/keeper/msg_server.go`), stale since Task 9
       (`dcf3732c`);
     - `knowledge-slash-callsites` (`x/knowledge/keeper/rounds.go`), stale since the Task 7
       follow-up (`3fe9872c`, the conjecture's creation cause). That follow-up's note does not
       record it; Task 12's note is the first to mention this anchor.

     Each pin still matches its file at the commit before the one named (checked in G5 by
     hashing the file at every branch commit). Both files were edited again by later tasks and
     review fixes. The re-pin follows Task 6: the anchors, then the manifest digest in
     `tools/authority-graph/manifest.go`, `dashboard/src/authority-geometry.ts`,
     `.specs-and-plans-hashes` (`authority-geometry-v1-manifest`) and `.recursion-manifest-hash`.

     The same stale pins fail in `dashboard/`: `npm run check:authority-geometry` exits 1, and
     `npm test` fails two tests in `tests/authority-geometry.test.ts` ("verifies every exact
     repository source hash and required or forbidden snippet" and "exits nonzero when the CLI
     is explicitly asked for the target gate"). The TypeScript validator stops at the first
     mismatch, `knowledge-writers`. The re-pin fixes these too.
  2. `scripts/check_specs_and_plans_hashes.sh` (`make creed-check`) fails on
     `knowledge-claim-records-v1`. Its pin in `.specs-and-plans-hashes` has not matched
     `docs/specs/knowledge-claim-records-v1.md` since Task 3 (`52bb3848`), and Task 15 edited
     the file again. In the same file, `knowledge-claim-integrity-v1`
     (`docs/specs/knowledge-claim-integrity-v1.md`) is not registered, although the sibling
     knowledge specs are pinned there. The script cannot fail on a document it does not list,
     so the spec is simply unpinned.
  3. The SDK's generated codecs. `npm run check:generated` in `sdk/typescript/` (the first step
     of its `npm run check`) fails with "Generated codec inputs are stale". Task 5
     (`176a84ed`) edited `proto/zerone/` and is the only branch commit that touches the
     check's inputs; the check passes at `176a84ed~1` and at the base `a4272bfa` (run on
     exported snapshots in the cleanup pass). The Task 3 follow-up note recorded it first.
     The fix, `npm run generate`, rewrites `src/generated/` and re-pins
     `src/generated/{SOURCE,GENERATED}_SHA256`, so it belongs to the pinning step. The SDK's
     `check:registry`, `typecheck` and `test` pass.

  Not deferred, fixed: the dashboard's second type check
  (`npx tsc --project tsconfig.functions.json --noEmit`) failed from `cdb937e0` (TS2322 ×3 and
  TS2352 in `tests/development.test.ts`, in the Task 3 follow-up's test) until the cleanup pass
  fixed that test's typing (`e2e41a59`). Both dashboard tsc projects now pass.

  Neither this plan, the spec nor `docs/EVENTS.md` is pinned, so editing them changes no
  check.

  The import-index fix (final acceptance row 10) adds no deferred check and changes none of
  the three above: it edits `x/knowledge/keeper/genesis.go`, which no authority-geometry
  anchor pins, adds two test files, and edits this plan and the spec, which are unpinned. It
  changes no proto, so the SDK's generated codecs are unaffected.
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
- Follow-up to Task 1 (review finding): the probe was not the only leak. Every zeroned
  command builds a temporary app that creates `DefaultNodeHome/data`, and `DefaultNodeHome` is
  `$HOME/.zeroned` whatever `--home` says (`app/app.go:343-347`, `:717-723`). So `init`'s
  `cli()` calls and `start` still wrote the operator's `~/.zeroned/data`; this was seen with the
  a4272bf binary. `node_environment(home)` now sets `HOME=<home>/.cli-home` (created 0700;
  refused if it is a symlink, foreign-owned or group/world-accessible) for every `cli()` call
  and for `start`. `scripts/claim-workflow.py` (`Workflow.cli`) uses the same environment.
  New red tests: `test_cli_runs_zeroned_with_sandbox_private_home`,
  `test_start_runs_zeroned_with_sandbox_private_home` and
  `test_symlinked_cli_home_refused_before_binary` in `test_local_node.py`, and
  `test_cli_runs_zeroned_with_sandbox_private_home` in `test_claim_workflow.py`. Each fake
  binary creates `$HOME/.zeroned/data` the way the SDK does.

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
- Follow-up to Task 2 (review finding): the golden digest cannot detect a marker-off leak of
  Task 8's first-save-only index write. Task 8 (`e6dc3cee`) already guards it with the marker-off
  twins in `claim_integrity_dedup_test.go`, so no test was added and this file is unchanged.
  Mutants were run on a /tmp copy of `ab151287`:
  - `SetClaim` writes 0x09/0x34 only on a first save, whatever the marker. The digest stays
    green, and three marker-off subtests fail: `ResaveDoesNotMoveIndex` (0x09 and 0x34 after a
    direct re-save), and `StandingFactBlocksAndDisprovenReleases` and
    `AcceptedFactKeepsCanonicalEntry` (0x34 names the claim after the acceptance re-save at
    `rounds.go:299`).
  - Only the 0x34 write made first-save-only: the same three fail. Only the 0x09 write:
    `ResaveDoesNotMoveIndex` fails.
  - `InitGenesis` sorts claims whatever the flag: `ImportReplaysAdmissionOrder` fails
    (flag=false).
  - `acceptedFactMayTakeCanonicalEntry` ignores the marker: nothing fails. With the marker off,
    the re-save at `rounds.go:299` points 0x34 back at the claim, so the end state is the same.
    The only difference is one extra read.

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
- Red observed: post hoc only (review G5, 2026-10-17). The Task 3 commit `52bb3848` recorded no
  red run. In the check, the Task 3 version of `query_claim_history_test.go` ran against the
  parent's code (`c7d60f01`, an exported copy under `/tmp/zerone-v12-g5`).
  `TestClaimHistoryLinksResubmissionsCanonicalTwinsAndAttributions` failed (`"[]" should have
  4 item(s), but has 0`). `TestClaimHistoryHashLinksRequireRetainedNonEmptyRootHashes` passed
  there: it asserts that no hash link appears, which the parent's code also satisfies. To show
  it is not vacuous, four mutants were run on a copy of `52bb3848`, and each failed at least one
  of the two tests: the content-hash, canonical-hash or `structure.object` link without its
  non-empty-root guard, and the `structure.object` link dropped. Both tests pass unmutated at
  `52bb3848`.
- Follow-up to Task 3 (review finding): three repo clients refused any link field outside the
  old closed set, so a history with a resubmission, canonical twin or attribution (the §5.5
  "who said P" read) threw. Their sets now also accept `content_hash`, `canonical_hash` and
  `structure.object`, and still refuse anything else (exact match, no case folding):
  - `sdk/typescript/src/claim-history.ts` (`queryClaimHistory`), with `dist/index.js` rebuilt
    by `npm run build`; only that set changes and the dist digest files stay as they are;
  - `dashboard/src/development-reader.ts` (`validateHistory`, also used by the research
    workflow);
  - `tools/research-review/workflow.py` (`validate_history`).
  New red tests feed each new field and four near-miss names. `scripts/claim-workflow.py`,
  `claim-workflow-view.py`, `shared-claims.py` and the dev-1 gateway pass links through
  without a closed set, so they need no change. Not done here: the SDK's generated codecs
  have been stale since Task 5's proto edits (`npm run check:generated` fails). Fixing that
  needs `npm run generate`, which re-pins `src/generated/{SOURCE,GENERATED}_SHA256`, so it
  is left for the separate reviewed pinning step.

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
- Red observed: post hoc only (review G5, 2026-10-17). The Task 4 commit `5541f48d` recorded no
  red run. In the check, the Task 4 version of `test_claim_workflow_attribution.py` ran against
  the parent's `claim-workflow.py` (`52bb3848`, exported under `/tmp/zerone-v12-g5`), and all 24
  of its tests failed: 71 errors and 7 failures, counting subtests. Most errors are
  `AttributeError`s for the encoder and guard functions that did not exist yet
  (`attribution_claim`, `quote_guard`, `jcs`, …). Of the failures, three are exit code 2
  (argparse usage) where the `attribute` command was expected to run or refuse. The other four
  are guard refusals that the parent never made: three were not raised at all, and one was a
  different refusal (`Contradiction evidence references cannot contain commas…`). The
  follow-ups below recorded their red runs when they were made.
- Follow-up to Task 4 (review finding):
  - The guard turned format characters into spaces, so a verbatim quote with soft hyphens
    (or ZWSP, U+2060, U+FEFF) split its words and passed the 8-token window. §4.7 now
    deletes marks, format characters and default-ignorable code points after NFKD, the same
    way for excerpt and field. The default-ignorable table matches one derived from Go's
    Unicode 15 tables. New red tests cover a soft hyphen in the excerpt and in a field, ZWSP
    and the other default-ignorables, combining marks that NFKC keeps, accent folding,
    whole Devanagari/Hangul words, and idempotence.
  - `QUOTE_GUARD_VERSION` stays `zerone.quote-guard/v1`: v1 has not left this branch, so
    it is redefined before merge rather than bumped.
  - `attribute --dry-run` is accepted as an explicit no-op (a dry run is the default), as §7
    #16 spells it. With `--broadcast` it is a usage error (exit 2).
- Follow-up to Task 4 (review G4):
  - F3: DOIs with `(`, `)` (Elsevier PII) and SICI DOIs with `<…>` and a `#` check
    character were refused; §4.2 and §4.7 now admit those characters. Only a DOI can use
    them: the other work IDs and structure fields keep their narrower grammars.
  - TQ-2: the 16-weight unit rule of §4.7 for scripts written without spaces. Red tests
    cover run-in and trimmed Chinese, Japanese and Thai quotes, the 15/16 boundary, a word
    weighing 2, a short excerpt run into other text, and unchanged spaced-text results.
    `QUOTE_GUARD_VERSION` stays `zerone.quote-guard/v1` for the same reason as above.
  - TQ-3: the refusal test has one row per guarded field (contradiction text, challenge
    reason, each evidence position, review reason and scope, the extractor DID through
    `reasoning_trace`), each in a fresh home, plus clean controls; casefold tests cover
    ß/ss/ẞ and ligatures both ways and the first NFKD. No input needs the second NFKD under
    Unicode 14.0–16.0; a test checks that over every code point instead.
  - TQ-4: `local-node.py`'s private-HOME refusals are tested one property at a time (a
    symlink to a private owned directory, a 0750 directory, a regular file, a foreign
    owner), through `cli` and `start`, before any subprocess call. Removing any one check
    now fails exactly one test; before, removing any one failed none.

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
- Red observed: post hoc only (review G5, 2026-10-17). The Task 5 commit `176a84ed` recorded no
  red run. Against the parent's code (`5541f48d`, exported under `/tmp/zerone-v12-g5`), the
  Task 5 `claim_integrity_activation_test.go` does not build: `internal/claimintegritymigration`
  does not exist yet, and neither do the marker API and genesis field. That red is a build
  failure only. It shows that the tests need the new code, not that each assertion
  discriminates. So eight mutants were also run on a copy of `176a84ed`, and each failed at least
  one of the seven tests:
  - a malformed marker read as off;
  - fund settlement not required to enable;
  - genesis validation without the integrity-requires-fund-settlement rule;
  - `InitGenesis` not enabling the marker;
  - `ExportGenesis` exporting false;
  - `Migrate11to12` writing no receipt;
  - `Migrate11to12` running without its owner;
  - activation ignoring an existing receipt.

  All seven tests pass unmutated at `176a84ed`.

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
  - Done in Task 6 (found by running the suites): `upgrades_fund_settlement_test.go` now freezes
    its compiled target at knowledge 11, as the claim-records test already does at 10, and
    restarts or imports through `LoadLatestVersion` on the frozen app. Three test files outside
    the list also needed one line each:
    - `tests/cross_stack/upgrade_e2e_test.go` asserted knowledge ConsensusVersion 11;
    - `tools/fork-genesis/main_test.go` adds `claim_integrity_enabled=true` next to its other
      explicit native selections;
    - the header comment of `claim_integrity_activation_test.go` said registration was still
      pending.
  - Done in Task 6 (`make test` → `tools/authority-graph`): editing `app/app.go` changes the
    source anchor `app-wiring` in `dashboard/public/standards/authority-geometry.v1.json`. The
    anchor was re-pinned, and so was the manifest digest that four places pin:
    `tools/authority-graph/manifest.go`, `dashboard/src/authority-geometry.ts`,
    `.specs-and-plans-hashes` (`authority-geometry-v1-manifest`) and `.recursion-manifest-hash`.
    The revision and snapshot date were left unchanged, as in `97aa0044`. Tasks 7–14 edit two
    more anchors, `msg_server.go` (`knowledge-writers`) and `rounds.go`
    (`knowledge-slash-callsites`), so each of those tasks must re-pin the same way.
  - Test note: `TestClaimIntegrityRejectsWrongSourceAndPrematureActivation` runs every refusal
    through `UpgradeKeeper.ApplyUpgrade` and checks the exact error text. It does not use
    `RunUpgradeHandlerWithInfoForTests`. That helper has no dedicated branch for this plan, so it
    falls to the generic preflight `requireCompletedPreSDKTransitionVersions`, which requires
    knowledge 6. A throwaway probe showed the helper refusing an unmodified, valid
    fund-settlement source with `require knowledge=6, got 10` before the handler ran. The
    fund-settlement rejection test therefore proves less than its scenario names suggest. That
    test is left unchanged here.
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
- Done in Task 7:
  - Each caller attaches the cause to a derived context used only for its own `SetFact`
    call, through `claimIntegrityTransitionCause`, which reads the marker and returns any
    read error.
  - One extra test file, `transition_cause_test.go` (package `keeper`), checks the
    `factID` scoping directly. The scenario alone cannot check it, because cascade
    descendants are written with an explicit transition and `SetFactSkipTransition`.
  - The MALFORMED arm is covered by a `MsgChallengeProvisionalFact` round on a conjecture.
  - Not changed in Task 7: a conjecture's creation transition (UNSPECIFIED→PROVISIONAL) was
    still inferred as `unknown` with no ID, so §5.2 row 1 did not hold for conjectures. The
    follow-up below fixes this.
  - `msg_server.go` and `rounds.go` anchors re-pinned as described in Task 6.
- Follow-up to Task 7 (review finding): §5.2 row 1 now holds for conjectures.
  - With the marker on, `createFactFromClaim` attaches `verification/<claim id>` to the
    conjecture's creation `SetFact` only, through `claimIntegrityTransitionCause`. A VERIFIED
    creation already gets the same cause by inference.
  - With the marker off, the context is unchanged and `unknown` with no ID is recorded as
    before. The only difference is one read of the 0x87 marker, on the BeginBlocker path.
  - Test: in `claim_integrity_history_test.go`, `conjectureChallengeTransitions` now takes the
    expected creation row. The marker-on test expects `verification` with the conjecture's
    claim ID and the marker-off twin expects `unknown` with no ID.
  - The Task 2 golden digest has no conjecture in it, so it cannot see this path. The
    marker-off twin is the guard: a mutant that attaches the cause whatever the marker fails
    it.

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
  - a fact that has become DISPROVEN, EXPIRED or PRUNED allows re-assertion (superseded for
    EXPIRED by decision (b), review G2: an EXPIRED fact now blocks, §5.4);
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
- Done in Task 8:
  - `dedupEntryBlocks` also returns the status that decided, so the refusal names it
    without extra reads. `blockingDuplicate` returns that record (or nil). The refusal keeps
    its prefix and appends `(status <STATUS>)`. For an ACCEPTED claim it appends
    `(status CLAIM_STATUS_ACCEPTED; fact <id> <FACT_STATUS>)`, or `… fact <id> missing`,
    `… round <id> missing` or `… no round` when it fails closed.
  - One addition: an empty entry value is treated as corrupt and refuses the tx.
  - The acceptance-time guard is `acceptedFactMayTakeCanonicalEntry`.
  - Observation on the inferred defect: with the marker off, the acceptance re-save does
    point 0x34 back at the claim ID, but `FactByCanonical` still returns the fact. It finds
    it through its fallback scan of every fact (`grpc_query.go`), not through the index.
    With the marker on, the index names the fact directly.
  - As the §5.4 table implies, a 0x34 entry naming a deleted fact names nothing and does
    not block. The 0x09 entry of the same accepted claim does block (fact missing).
  - Until Task 10, a contradiction's first save still takes a live claim's 0x09 entry
    (P1ext).
  - `msg_server.go` and `rounds.go` anchors re-pinned as described in Task 6.
  - Review fix, outside the file list: `genesis.go`. A genesis file lists claims in claim-ID
    order, and `InitGenesis` replays each claim's first save, so the claim listed last took
    0x09/0x34 even when it was an old terminal claim. A third copy was then admitted while the
    resubmission was live or its fact stood. With `gs.ClaimIntegrityEnabled`, `InitGenesis`
    now replays a copy of the claims stably sorted by (`SubmittedAtBlock`, ID). With the flag
    off the file order is kept, so the Task 2 digest does not change.
    `TestClaimIntegrity_ImportReplaysAdmissionOrder` imports the file in its exported order and
    reversed, with the flag on and off.
  - After import, 0x34 for an accepted claim names the claim, not its fact. Both decide the
    same way: the claim resolves to the same fact through its round, and `DeleteFact` has no
    caller outside tests.
    `FactByCanonical` still returns the fact through its fallback scan.
  - Residual: two same-hash claims admitted in one block are ordered by ID, not tx order. With
    the marker on this needs the P1ext contradiction path, which Task 10 closes.
  - Review nit: tests now cover an ACCEPTED claim with no round and an unreadable round record.

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
  - `q(v) = strconv.QuoteToASCII(collapse(TrimSpace(NFC(v))))`, where `collapse` turns each
    run of `unicode.IsSpace` into a single U+0020. (Planned as `strconv.Quote`; changed in
    Task 9, see "Done in Task 9".)
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
- Done in Task 9:
  - `q` uses `strconv.QuoteToASCII`, not `strconv.Quote`, so every v2 form is ASCII. With
    `strconv.Quote` the idempotence test failed on a valid field: ZWSP then U+0307 quotes as
    `\u200b` + U+0307, and NFC composes the escape's `b` with the mark, so the lookup
    normaliser changed a form the chain built. Which runes `strconv.Quote` leaves unescaped
    also follows the Go toolchain's Unicode tables, so a newly assigned code point could change
    a form (and a dedup decision) between binaries. The §4.3 example form is ASCII and is
    unchanged.
  - The v2 checks live in `validateClaimRecordInput`, which now takes the marker: a declared
    form must equal `BuildCanonicalFormV2` after `NormalizeCanonicalFormForLookup`, and a
    declared form with no structure is refused. With the marker on, `submitClaim` always
    stores the derived v2 form and never the declared text. The marker read moved to the top
    of `submitClaim` so both use one read.
  - One Task 8 test changed: the flag-off import branch of
    `TestClaimIntegrity_ImportReplaysAdmissionOrder` submitted a v1 structured claim expecting
    it to meet an exported (now v2) canonical entry. It now asserts that the two hashes differ.
    The replay-order assertions are unchanged.
  - Extra test: `TestClaimIntegrity_LegacyV1CanonicalEntriesDoNotInteract` checks the §3.6
    row on legacy v1 hashes after `EnableClaimIntegrity`.
  - Not done: re-pinning the `knowledge-writers` anchor for `msg_server.go` (Task 6 note). The
    session's permission classifier refused the hash edit, so `tools/authority-graph` fails
    until someone with permission re-pins it.

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
- Done in Task 10:
  - The three checks are `claimIntegrityContradictionAdmission` in `msg_server.go`, in plan
    order. `submitContradiction` reads the marker once, after the identity check and before the
    stake lock, and calls it only when the marker is on. With the marker off the only
    difference is that one read of 0x87.
  - Refusals: `fact <id> is not standing (status: <STATUS>); …`,
    `contradiction stake <s> below effective minimum <m> (fact confidence <c>; …)`, and
    `duplicate claim: content hash matches existing claim <id> (status …)`, the same text as
    `submitClaim`'s refusal.
  - Tests beyond the list: every fact status applied in a cache (the five standing ones
    admit), a DISPROVEN target reached through a real challenge panel, one uzrn below the
    minimum, the target's own accepted text (blocked while its fact stands), and a copy of a
    live contradiction's text through either message. No assertion reads the target after an
    admission, so Task 12 does not change these tests.
  - The Task 8 residual is closed. With the marker on, all three admissions that write a 0x09
    entry (`submitClaim`, the conjecture path and `submitContradiction`) now check it first.
    The P1ext test is refused in the same block as the live claim, so two same-hash claims
    can no longer be admitted in one block.
  - Outside the file list, as the §6 conventions expected: `app/claim_records_test.go`
    characterizes claim-records retention on knowledge-11 contradiction admission with a
    1 ZRN stake, below the 11 ZRN minimum. Its fixture now deletes 0x87 with a comment; no
    assertion changed. Every other knowledge, app and cross-stack test passed unchanged.
  - Not done: re-pinning the `knowledge-writers` anchor for `msg_server.go` (Task 6 note). It
    is deferred to the separate reviewed pinning step, so `tools/authority-graph` fails until
    then.

- Review fixes, group G2 — admission and dedup (2026-10-17; every rule behind the marker,
  marker off unchanged, Task 2 digest unchanged; tests in
  `claim_integrity_admission_test.go` unless named):
  - SEC-2 and F5, decision (a) (§9 D1): `submitClaim` refuses a CONTRADICTS relation with
    `claim relation <i> names fact <F> as CONTRADICTS; a claim cannot carry a CONTRADICTS
    relation — dispute the fact with MsgSubmitContradiction (…)`, before any target is read
    (`refuseClaimContradictsRelation`). Test:
    `TestClaimIntegrity_SubmitClaimRefusesContradictsRelation`, with its marker-off twin. Four
    existing tests admitted such a claim with the marker on. Each now admits it before
    activation, the only way one can still reach a knowledge-12 round:
    `TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted` became
    `TestClaimIntegrity_PreActivationRelationContradictionHoldsOnceAccepted` (accepted after
    activation, its edge holds F through R1, while knowledge 11 launders F);
    `TestClaimIntegrity_AcceptedContestFollowsStoredEdge` and
    `TestClaimIntegrity_SubmitClaimRelationsRecuseNoDefender` keep their names;
    `TestClaimIntegrity_UnacceptedContradictionChangesNothing` drops its `MsgSubmitClaim` arm,
    and `TestClaimIntegrity_PreActivationContestSettlesThroughPredicate` gains one (either
    knowledge-11 admission path settles through the predicate). A legacy claim naming F as
    `[CONTRADICTS F, CITES F]` and accepted after activation leaves an already CONTESTED F with
    no contest reason; the next ending of a contradiction or challenge of F releases it.
  - SEC-3, decision (b) (§5.4, §9 D3): `factBlocksDedup` counts EXPIRED as standing. Test:
    `TestClaimIntegrity_ExpiredFactBlocksDedup` (real metabolism to EXPIRED; exact, reworded
    structured and conjecture copies refused; patronage revives P and no twin exists; PRUNED
    releases); `TestClaimIntegrity_FactStatusDecidesWhetherEntryBlocks` updated.
  - SEC-4 (§4.2, §4.8): `refuseReservedAttributionText` in `submitClaim` (the text may open
    with the template only when the claim is `CLAIM_TYPE_RELATION` and its structure renders
    exactly that text), in the conjecture path and first in
    `claimIntegrityContradictionAdmission`. Pure checks, no reads. Test:
    `TestClaimIntegrity_AttributionTemplateIsReserved` (seven `MsgSubmitClaim` shapes, a
    conjecture and a contradiction refused; the real attribution admitted; marker-off twin
    shows the knowledge-11 squat). The ':'-in-domain split was first reported as a residual;
    the re-review below closes it for attributions.
  - CI-REVIEW-1 (§3.6, §9 D4): no rewrite.
    `TestClaimIntegrity_LegacyContentCollisionLiveVsImport` pins the live (last-saved) and
    imported (last-admitted) decisions in both directions.
  - Docs: spec §1.5, §2.2, §2.3, §2.8, §3, §5.1, §6.1, §6.4, §7, §11; `docs/EVENTS.md`
    (`contradiction_accepted`); the `--relations` flag help of `submit-claim`.

- Review fixes, group G2 re-review (2026-10-17; behind the marker, marker off unchanged,
  Task 2 digest unchanged):
  - SEC-4, domain split (§4.8): `refuseReservedAttributionText` takes the domain the content
    hash uses and first refuses one that contains `Attribution zerone.attribution/v1`, with
    `domain "<d>" contains "Attribution zerone.attribution/v1", the reserved attribution
    template's name; …`. `submitClaim` and the conjecture path pass `msg.Domain`;
    `claimIntegrityContradictionAdmission` gets the defaulted domain (`msg.Domain`, else the
    target's), not `msg.Domain`. Still pure, still behind the marker. Tests:
    `TestClaimIntegrity_AttributionHashCannotBeSplitIntoDomain` (every split of A1, at each of
    its five `:`, refused on `MsgSubmitClaim` and `MsgPostConjecture` in a proposed domain and on
    `MsgSubmitContradiction` in a domain nobody proposed; a domain that only contains the name
    is refused on all three; the real attribution admitted; marker-off twin admits every split
    on every path, each blocking the real attribution) and
    `TestClaimIntegrity_ContradictionDefaultDomainIsChecked` (a contradiction naming no domain,
    against a pre-activation fact in a split domain, refused after an in-place upgrade;
    marker-off twin admits it and it blocks the real attribution). Eight mutants were each
    killed: the contradiction checking `msg.Domain` or no domain, the conjecture or claim path
    checking no domain, `HasSuffix` for `Contains`, the prefix with its `:` for the name, the
    domain checked only when the text opens with the prefix, and the conjecture check ungated.
  - Decision (a) cost (§4.3, §9 D1): a contesting P2 goes in as `MsgSubmitContradiction`
    only, with no structure, and a structured P2 cannot later contest with the same text.
    Pinned by `TestClaimIntegrity_StructuredPropositionCannotAlsoContest`.
  - §9 D4: how to read the live 0x09 entry, and why the "any member blocks" clause already
    covers every disagreement.
  - Task 12 and Task 14 failing-test bullets that describe marker-on `MsgSubmitClaim` +
    CONTRADICTS admission now point here.
  - Docs: spec §2.2, §2.3, §2.8, §3, §5.1.

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
- Done in Task 11:
  - `isClaimIntegrityDisputeMsg` names the two messages. They stay out of
    `isZeroneSpecificMsg`, so without the gate they keep the legacy default-allow.
    `WithClaimIntegrityGate` returns a copy of the decorator. A decorator built without it
    (a nil gate) never reads anything and behaves as before; only `NewAnteHandler` wires it.
  - The gate is read whenever a tx carries a dispute message, once per tx whatever the number
    of signers and messages, before the account lookup, even when the signer would be allowed
    either way. A read error refuses the tx as
    `read claim integrity gate for dispute message: <cause>`. Simulation skips the decorator,
    as before, so it never reads the gate.
  - Cost with the marker off: one 0x87 read (gas) in the ante of a tx that carries a dispute
    message. Nothing is written, so the Task 2 digest is unaffected.
  - Tests beyond the list: `TestAnteIntegration_ClaimIntegrityGateWiredToKnowledgeMarker`
    runs the complete `NewAnteHandler` chain with a signed tx on a `DefaultGenesis` app. An
    unregistered signer is refused with the onboard hint, admitted once 0x87 is deleted, and
    refused when 0x87 is `{2}`; `MsgPostConjecture` is admitted with the marker on. With the
    gate on, each refusal has exactly `MsgChallengeFact`'s text. The spy test also checks that
    ten other message types give the legacy decision for unregistered, contract and human
    signers. Mutants (gate ignored for registered accounts, gate read for every message, no
    memo, read error treated as off) each fail at least one test.
  - No existing test changed. The ante files are not authority-geometry anchors, so nothing new
    needs re-pinning.

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
    (After review G2: such a claim is admitted before activation, since decision (a) refuses
    it with the marker on; see the G2 block in Task 10.)
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
- Done in Task 12:
  - `claim_integrity_contest.go` holds the predicate and both arms. `completeRound` reads the
    marker once, at its top and before any write. The ACCEPT arm calls `contestAcceptedTargets`; the
    three non-accepted arms call `endContradiction`, which runs `settleContradictionTargets`
    with the marker on and the unchanged `reverseContradictionsFromClaim` with it off. In
    `phases.go` the legacy-policy starved branch reads the marker and calls the same helper. A
    marker read error or a write error there is logged and leaves every target unchanged; it
    is never read as the legacy path.
  - Predicate: `contestReasonRemains(ctx, F, survivedAt)` and
    `contestReasonRemainsSinceLastSurvival(ctx, F)`. Task 13 needs both forms. The scan
    charges every entry it reads (relation, fact, transition) against one budget: 100k records
    and 64 MiB per check. Records are validated with `validateToKStoredRecord`. An edge naming a
    missing fact counts as malformed, so the reason remains. R1 stops at the first standing
    counter-fact. `lastSurvivedAt` counts only `challenge_survived` (Task 7). A knowledge-11
    survival, recorded as `unknown`, therefore clears nothing. `survivedAt` is read only when
    a DISPROVEN dependency makes it matter.
  - Failure handling choices the plan left open:
    - in `settleContradictionTargets`, a target record that cannot be read is left as it is and
      reported with `contest_check_incomplete` (reason `target: …`), so an ending round still
      completes;
    - `contestAcceptedTargets` returns read and write errors, and the round cache discards them;
    - the settle writes run in their own cache, so an error leaves every target unchanged on
      the logged path too.
  - Conjectures follow the plan text: the predicate applies, and a released conjecture returns
    to PROVISIONAL. With the marker on, a conjecture cannot be contradicted or contested by an
    acceptance, so only legacy records reach this.
  - The legacy-policy starved branch runs only when no aggregation ran between the reveal and
    aggregation deadlines, as in `TestSubmitContradiction_StarvedDoesNotLockFact`. Otherwise
    such a round ends through `completeRound`, like a neutral one.
  - Task 10's standing check now calls `factStands`, the same five statuses. Its behaviour and
    messages are unchanged.
  - Tests beyond the list:
    - all three non-accepted endings, through both admission paths;
    - the pre-activation release, and the P3 state before activation (R1 holds), each upgraded
      in place and imported;
    - the legacy-policy starved branch;
    - a second file, `claim_integrity_contest_predicate_test.go` (package `keeper`). It checks
      R1 for every fact status, R2 for every relation type and every survival order,
      `lastSurvivedAt`, and eleven faults. The faults include an injected read error, both
      bounds and malformed records. Each fault yields exactly one `contest_check_incomplete`
      and no write.
  - Mutants that each fail at least one test:
    - settle ignores the predicate;
    - an incomplete check is read as no reason;
    - a survival is matched by status rather than cause;
    - `>=` in place of `>` (since review fix F1 the predicate uses `>=`; the mutant is now `>`);
    - acceptance does not contest;
    - R1 narrowed to VERIFIED/ACTIVE;
    - `phases.go` keeps the legacy reversal;
    - `submitClaim` still flips at admission.
  - Outside the file list:
    - `docs/EVENTS.md` documents `contradiction_accepted` and `contest_check_incomplete` and
      the v12 meaning of `contradiction_reversed`. `TestEventAudit_DocumentationCompleteness`
      and `TestTruthSeeking_VoiceLayerDocMirror` require this.
    - `tests/cross_stack/correction_admission_test.go` characterizes the knowledge-11 flip at
      admission on its contradiction route. That route now deletes 0x87 with a comment, as the
      §6 conventions prescribe. No assertion changed.
  - Cost with the marker off: one 0x87 read in `completeRound`, and one in the starved branch.
    Nothing is written, and the Task 2 digest is unchanged (`f849e099…`).
  - Not done: re-pinning the `knowledge-writers` (`msg_server.go`) and
    `knowledge-slash-callsites` (`rounds.go`) anchors. That is deferred to the separate
    reviewed pinning step, so `tools/authority-graph` fails until then.
- Task 12 review (must_fix and should_fix, addressed):
  - Laundering through a twice-named target. The relation store keeps one edge per
    (source, target) pair, and the last write wins. A claim with `[CONTRADICTS F, CITES F]`
    therefore stored K→F as CITES, but acceptance still contested F from the claim's relation
    list. R1 found no CONTRADICTS edge, so a later starved contradiction released F while K
    stood. Fix: `contestAcceptedTargets` contests F only when the stored edge K→F is
    CONTRADICTS (`storedContradiction`). It reads the reverse-index entry, the same record R1
    scans. An absent edge contests nothing; an unreadable edge is returned as an error, so the
    round does not complete. This is marker-on code only, and it also covers claims admitted
    before activation. §5.2 row 2 now says this.
    `TestClaimIntegrity_AcceptedContestFollowsStoredEdge` covers both orders and has marker-off
    twins: knowledge 11 flips F at admission and launders it in either order.
  - Not done: refusing a claim that names a CONTRADICTS target twice. With the fix, acceptance
    and R1 agree, so a duplicate cannot launder. Refusing it would be a new admission rule, and
    the review marked it optional.
  - New tests in `claim_integrity_contest_predicate_test.go` (package `keeper`):
    - `TestContestAcceptedTargets`: every fact status, and only VERIFIED and ACTIVE become
      CONTESTED. It also covers the stored-edge cases: CITES, absent, reverse direction,
      unreadable, wrong pair, and an injected read error.
    - `TestSettleContradictionTargetsSinceLastSurvival`: settle uses F's last
      `challenge_survived` height.
    - `TestCompleteRoundRefusesMalformedClaimIntegrityMarker`: a malformed 0x87 is an error
      before any write, and the round is not completed.
  - `TestClaimIntegrity_LegacyPolicyStarvedContradictionSettles` has a new `malformed marker`
    case. The phases.go starved branch leaves F CONTESTED, emits no event, and BeginBlock
    does not halt.
  - Mutants, each of which now fails at least one test:
    - settle with `survivedAt` set to 0;
    - the contest filter narrowed to VERIFIED;
    - the filter widened to every standing status except CONTESTED;
    - the marker read error ignored in `completeRound`;
    - the marker read error ignored in phases.go;
    - the stored edge type ignored;
    - the stored edge read error ignored.
  - An explicit source/target check in `storedContradiction` survived its mutant, because
    `validateToKStoredRecord` already rebuilds the key from the payload. It was removed.
- Review fixes, group G3 — marker-off twins for `completeRound` (2026-10-17; tests only, no
  code or semantic change; the Task 2 digest is unchanged). TQ-1: three marker gates in
  `completeRound` had no marker-off test, so forcing the knowledge-12 branch at any of them
  passed every suite (`./x/knowledge/...`, `./app/...`, `./tests/cross_stack/...`). The gates
  are the `contestAcceptedTargets` call in the ACCEPT arm and the `endContradiction` calls in
  the REJECT and MALFORMED arms.
  - `TestClaimIntegrity_ContradictionAdmittedDuringChallengeContestsOnAcceptance` (new; marker
    on and off). A contradiction A is admitted while F is CHALLENGED and accepted one block
    after F survives, when F is ACTIVE. Marker on: F becomes CONTESTED with
    `contradiction_accepted/A`. Marker off: F stays ACTIVE, with no transition and no
    `contradiction_accepted` event.
  - `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted` now ends B in each of
    `contestEndings` (rejected, malformed, starved) instead of starving it only. Marker on: F
    stays CONTESTED while K stands. Marker off: F returns to VERIFIED with the knowledge-11
    inferred cause `verification/<F's claim>` and `contradiction_reversed{reverted_by_claim: B}`.
  - Mutants on a /tmp copy of `29022b71` (`/tmp/zerone-v12-g3`). Each survived the suites at
    that commit and is killed only by its twin: `if true` at the ACCEPT gate (M5) by the new
    test's `marker_off` case; `endContradiction(ctx, claim, true)` in the REJECT arm (S1) by
    `marker_off/rejected`; the same in the MALFORMED arm (S2) by `marker_off/malformed`. The
    INCONCLUSIVE arm (S3) was already guarded: at that commit its mutant fails the starved
    marker-off twins and `TestV11_GoldenKnowledgeStoreDigest`.

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
    - REJECT at h2 > h1 makes F ACTIVE (after review fix SEC-1: only if the challenge opened
      after h1; one opened at or before h1 leaves F CONTESTED);
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
     is CONTESTED. (Review fix SEC-1: `survivedAt` is the height the challenge opened, its
     claim's `SubmittedAtBlock`, not the current height.)
  3. On an inconclusive, malformed or starved challenge, the same rule applies with
     `survivedAt = lastSurvivedAt(F)`.
  4. After K is disproven, settle each outgoing CONTRADICTS target of K, inside the same cache.
  - The survival reward is still released on REJECT, as today. No economic change.
- Acceptance: the tests above pass, and the Task 2 digest is unchanged.
- Done in Task 13:
  - The helpers live in `claim_integrity_contest.go`, next to the predicate:
    - `challengeEndStatus` implements rules 2 and 3;
    - `settleCounterFactTargets` implements rule 4;
    - `releaseContested` is the release step, now shared with Task 12's
      `settleContradictionTargets`. That settle behaves exactly as before.
  - Gate. `challengeFact` reads the marker only when the target is not VERIFIED or ACTIVE.
    With the marker off the refusal text is unchanged. With it on, other statuses are refused
    with `… (commitment 4: only verified, active or contested facts accept probes; …)`.
  - Survival and nondecisive endings:
    - each reads the marker once, in place of the Task 7 `claimIntegrityTransitionCause`
      call at that site (`handleChallengeDisproven` too), so the read count is unchanged with
      the marker off;
    - a survival is judged with `survivedAt` = the current height. Every other ending is
      judged since `lastSurvivedAt`. (Superseded by review fix SEC-1: a survival is judged
      from the height its challenge opened, and `lastSurvivedAt` returns that height.)
    - a check that cannot finish returns the fact to CONTESTED and emits
      `contest_check_incomplete`. This applies to any challenged fact, including one that
      was VERIFIED before the challenge. (Superseded by review fix SEC-6: only a fact that was
      CONTESTED when the challenge opened returns to CONTESTED; any other returns to ACTIVE.
      Restored by review G1R, which reverts SEC-6: this rule holds as first written.)
  - Consequence of §5.2, now pinned by a test: a fact laundered before activation (VERIFIED
    while K stands, §3.6) returns to CONTESTED when a challenge ends without refuting it, after
    an in-place upgrade or an import. The marker-off twin returns it to ACTIVE.
  - Counter-fact settle:
    - It lists K's stored CONTRADICTS edges (the forward index) within the contest-scan
      budget. Only CONTESTED targets are re-evaluated.
    - A released target becomes VERIFIED (PROVISIONAL for a conjecture) with cause
      `counterfact_disproven/<K>`.
    - The plan named only the event attribute. The event reuses
      `zerone.knowledge.contradiction_reversed`, with `fact_id` and `reverted_by_fact=<K>`
      in place of `reverted_by_claim`.
  - Failure choices the plan left open:
    - if K's relations cannot be listed (unreadable, mismatched or past the bound), no target
      settles and one `contest_check_incomplete` names K;
    - a target that cannot be read is reported and left as it is, and the other targets
      still settle;
    - the disproval stands in both cases;
    - a write error is returned, so the round's cache discards it, like the other errors in
      `handleChallengeDisproven`.
  - Conjectures: a challenge ending always returns one to PROVISIONAL (rules 2 and 3). The
    counter-fact settle applies the predicate as Task 12's settle does, so another standing
    counter-fact keeps a conjecture CONTESTED. `ChallengeFact` accepts a CONTESTED conjecture
    at the unscaled minimum, since its confidence is 0; that is the same minimum
    `ChallengeProvisionalFact` uses. Only legacy records reach any of these paths.
  - Tests beyond the list:
    - `claim_integrity_verdict_test.go`:
      - every fact status at the gate, with the marker on and off;
      - a stake one uzrn below the minimum on a CONTESTED target;
      - MALFORMED as well as REJECT, ACCEPT and starved;
      - a second dependency refuted after a survival, which re-contests F until a new
        survival;
      - a counter-fact falling while a refuted dependency still holds F (the cascade skipped
        F);
      - the laundered legacy fact;
      - conjecture exemptions;
      - five fault cases.
    - `claim_integrity_contest_predicate_test.go` (package `keeper`, outside the list):
      - `TestSettleCounterFactTargets` covers every target status, every edge type, a
        conjecture, three held cases and a missing target;
      - `TestSettleCounterFactTargetsFailsTowardContested` covers six faults and a write
        error.
  - Mutants: 20 were run on a /tmp copy, and each fails at least one test. They were:
    - the gate ignores the marker;
    - the gate refuses CONTESTED;
    - survival judged since the last survival;
    - an inconclusive ending judged at the current height;
    - the conjecture exemption dropped;
    - settle not called;
    - settle run before the DISPROVEN write;
    - release ignores the predicate;
    - every edge type settled;
    - any status released;
    - a listing error returned;
    - targets collected before a listing error settled;
    - a target read error returned;
    - a target read error made silent;
    - reasons ignored at a challenge's end;
    - each of the three causes attached with the marker off;
    - the marker ignored on either ending path.
  - Outside the file list: `docs/EVENTS.md`. It documents `reverted_by_fact`, the challenge of
    a CONTESTED fact, the CONTESTED return in `challenge_inconclusive_restored`, and
    `contest_check_incomplete` naming K.
  - Cost with the marker off: one 0x87 read in `ChallengeFact`, only on its refusal path.
    Nothing is written. The Task 2 digest is unchanged (`f849e099…`), and no existing test
    changed.
  - Not done: re-pinning the `knowledge-writers` (`msg_server.go`) and
    `knowledge-slash-callsites` (`rounds.go`) anchors. That is deferred to the separate reviewed
    pinning step, so `tools/authority-graph` fails until then. `vindication.go` is not an
    anchor.

- Review fixes, group G1 (2026-10-17; marker-gated; marker off unchanged and the Task 2 digest
  unchanged):
  - SEC-1: `challengeEndStatus` takes the challenge claim. A survival is judged with
    `survivedAt` = its `SubmittedAtBlock`. `lastSurvivedAt` returns the `SubmittedAtBlock` of
    the challenge claim that the latest `challenge_survived` transition names, read and
    validated within the scan budget. The claim, not the `challenge_opened` row, is the single
    source, so the survival and every later check read the same height, including for a
    challenge opened before activation (whose opening was recorded as `unknown`).
  - F1, CI-REVIEW-2: R2 counts a dependency refuted at the cutoff height (`>=`). The review
    asked for strict `>` "only where the survival instant itself is judged"; after SEC-1 no
    path judges against the survival's own height, and both paths must use one comparison, so
    `>=` is used throughout.
  - SEC-5: R1 uses `counterFactStands`, `factStands` plus EXPIRED (decision (b)). Contradiction
    admission keeps `factStands`.
  - SEC-6 (reverted by review G1R below; kept here as history): when the check at a
    challenge's end cannot finish, the fact returns to CONTESTED
    only if the latest `challenge_opened` transition naming this challenge has prior status
    CONTESTED, and otherwise to ACTIVE. If that transition cannot be read or is absent, it
    returns to CONTESTED. One `contest_check_incomplete` carries both reasons. Chosen over
    narrowing the bound to CONTRADICTS edges, because the reverse index cannot be filtered by
    type without reading every edge. The cost (narrowed by the second pass below): a fact that
    was not CONTESTED but has an R1 reason (a counter-fact accepted while it was CHALLENGED, or
    a pre-activation laundered fact), and whose check also cannot finish, returns to ACTIVE.
    The residual: a fact over the bound that is once CONTESTED is never returned to VERIFIED or
    ACTIVE, even by a challenge it survives that opened after its refutation, since every
    release check reaches R1 and cannot finish. The longer-term fix, an index of CONTRADICTS
    edges alone so that R1 is bounded, is a follow-up outside G1.
  - Tests: `TestClaimIntegrity_SurvivalClearsNothingRefutedWhileTheChallengeWasOpen`,
    `TestClaimIntegrity_SameBlockRefutationAfterSurvivalIsNotCleared`,
    `TestClaimIntegrity_ExpiredCounterFactStillHoldsTarget`,
    `TestClaimIntegrity_IncompleteCheckAtChallengeEndKeepsPriorStanding` (renamed in G1R to
    `TestClaimIntegrity_IncompleteCheckAtChallengeEndLeavesFactContested`), each with a
    marker-off twin, `TestChallengeEndStatus`, and updated `TestContestReason*` and
    `TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases` (renamed from
    `…FailsTowardContested`: a VERIFIED-before fact now ends ACTIVE). A test-only hook
    (`export_test.go`) lowers the contest-scan record ceiling. Nine mutants, one per rule
    above, each fail at least one test.
  - Second pass (review of `fecf1407`, must-fix): R1 read every incoming edge before R2, and
    both shared one budget, so a heavily cited F hit the bound before R2 saw its refuted
    dependency, and the SEC-6 fallback returned F to ACTIVE: the SEC-1 shield challenge still
    laundered F, even through a starved, refunded challenge. `contestReason` now reads R2
    (`refutedDependency`) before R1 (`standingCounterFact`). A reason found by either is
    conclusive and returned with no error; only when neither finds one does a failed read make
    the check incomplete, and the error then names every failed read. A complete R2 finding is
    therefore never masked by R1's bound, and a complete R1 finding is not masked by an
    unreadable dependency. Tests: `TestClaimIntegrity_ShieldChallengeOfHeavilyCitedFactLaundersNothing`
    (reject, malformed, starved; marker-off twin), `TestContestReasonFindingIsConclusive`, and
    the "refuted dependency past the bound" cases of `TestChallengeEndStatus`. Updated: the
    `TestChallengeEndStatus` incomplete cases drop D; the contested-before
    malformed/starved cases of `TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases`
    and `TestClaimIntegrity_IncompleteCheckAtChallengeEndKeepsPriorStanding` still end
    CONTESTED but now emit no `contest_check_incomplete` (D is found first). Four mutants
    (R1 first; an R2 error stops before R1; only the first error reported; an R2 error
    releases) each fail at least two tests.
  - Third pass (review of `87557ceb`, must-fix): the second pass claimed that F's outgoing edges
    come only from its own claim, so a refuted dependency is never lost to the SEC-6 fallback.
    That was false. A passing augmentation writes a REFORMULATES edge whose source is its
    user-chosen ID, so an augmentation named `<F>/0` puts an edge under F's forward prefix.
    `refutedDependency` treated an entry whose source is not F as an error, the check was
    incomplete before it reached D, and the SEC-6 fallback returned a shielded F to ACTIVE (the
    error predates the second pass; G1's SEC-6 fallback made it laundering). Such an entry is
    now skipped, and still charged. `standingCounterFact` skips an entry whose target is not F
    the same way. Reachable with the marker on through an augmentation admitted before review
    neutrality whose ballot finalizes afterwards, or one that passed before it. Tests:
    `TestClaimIntegrity_ShieldChallengeBehindAugmentationEdgeLaundersNothing` (through
    `ApplyFinalizedAugmentationVerdict`; reject, malformed, starved; marker-off twin) and
    `TestContestReasonSkipsForeignEdgesUnderTheFactsPrefix`. The shield scenario is shared with
    the heavily cited case (`vdShieldChallenge`). An augmentation whose ID equals a fact's ID is
    a separate, pre-existing relation-graph defect, routed to §8.

- Review G1R, the contest-predicate redesign (2026-10-17; marker-gated; marker off unchanged and
  the Task 2 digest unchanged). The final G1 re-review proved the SEC-6 fallback a laundering
  primitive (scratch `/tmp/zerone-v12-g1fr`, probe `zz_final_bytebound_test.go`): about 343
  augmentations admitted before review neutrality, with 96 KiB IDs extending F's ID, filled the
  64 MiB byte bound of R2's prefix scan, so a shielded F returned to ACTIVE although a
  dependency fell while its challenge was open. The third pass's "about 100k augmentations"
  was false: the byte bound governs. Any attacker-inflatable read plus a rule that releases on
  an incomplete check launders. The plan's original rule is restored: an incomplete check never
  releases. Changes:
  - SEC-6 reverted. `challengeEndStatus` returns CONTESTED when the check cannot finish, and
    `contestedWhenChallengeOpened` is gone. SEC-1 (opening-height survival), F1 (`>=`), SEC-5
    (EXPIRED stands for R1) and the foreign-entry skips are kept. SEC-6 becomes the disclosed
    limitation of §5.3 and §8.
  - R2 by point lookups (`dependencies` in `claim_integrity_contest.go`): a claim-born fact
    reads its claim and `FactRelationKey(F, target)` for each named target, and the stored
    edge's type decides. Every named target is looked up, not only the cascade-typed claim
    relations, so the result equals the prefix scan restricted to the claim's targets (a claim
    naming `[CONTRADICTS D]` whose stored edge is CITES still counts, as the cascade would).
    Keys are `<source>/<target>`, so the key of F→`x/y` is also the key of `<F>/x`→y; an entry
    found by key whose payload names another pair is not F's edge and is skipped. The prefix
    scan remains only for facts with no claim. A missing or unreadable claim makes R2 fail; it
    never falls back to the scan.
  - The augmentation edge writer is gated (`claim_integrity_augmentation_edge.go`). With 0x87
    on, `ApplyFinalizedAugmentationVerdict` writes no REFORMULATES edge when the augmentation ID
    contains `/`, is longer than `AugmentationEdgeSourceMaxBytes` (128), or is a stored fact's
    ID, and emits `augmentation_edge_withheld{augmentation_id, original_fact_id, reason}`. The
    verdict, payout and acceptance are unchanged. The marker is read before any write of the
    verdict, so a malformed marker refuses it and moves no coins. 128 bytes: generated IDs are
    32 hex characters, so it leaves four times that for client-chosen IDs, and it keeps an edge
    entry under about 340 bytes, so 100k such entries stay far below 64 MiB. With the marker off
    the edge is written byte for byte as before, the overwrite included. Cost with the marker
    off: one 0x87 read per passing verdict.
  - Conclusive findings inside loops. Both rules record a failed item and go on; an error is
    returned only when no reason is found. The event names up to eight failures per rule and
    counts the rest; a stopped scan (the bound) is always named.
  - Tests, red first (logs `/tmp/zerone-v12-g1r/red-*.txt`):
    - `TestClaimIntegrity_ShieldChallengeBehindLongAugmentationIDsLaundersNothing` (new; byte
      bound lowered to 64 KiB through `SetContestScanMaxBytesForTest`; long-ID augmentations
      through the real writer before activation; reject, malformed, starve; marker-off twin
      ACTIVE). Red at `1484408b`: F ended ACTIVE. Red after the SEC-6 revert alone: CONTESTED
      but with `contest_check_incomplete`. Green with point lookups: CONTESTED, check complete.
    - `TestChallengeEndStatus`, `TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases`
      and `TestClaimIntegrity_IncompleteCheckAtChallengeEndLeavesFactContested` (renamed from
      `…KeepsPriorStanding`) moved to the restored semantics: every incomplete case ends
      CONTESTED; marker-off twins ACTIVE.
    - `TestContestReasonReadsClaimBornDependenciesFromTheClaim` (new; claim-born and genesis F),
      new claim-read faults in `TestContestReasonFailsTowardContested`, nine new cases and two
      failure-report cases in `TestContestReasonFindingIsConclusive`, and claim-born/genesis
      splits of `TestContestReasonRefutedDependencyIsOrderedBySurvival` and
      `TestContestReasonSkipsForeignEdgesUnderTheFactsPrefix`. The predicate fixture's facts
      are now claim-born, with each relation named in the fact's claim.
    - `TestClaimIntegrity_AugmentationEdgeStaysOutOfTheFactIDSpace` (new; six ID shapes, marker
      on and off; the marker-off twin pins the knowledge-11 bytes and the overwrite) and
      `TestClaimIntegrity_AugmentationEdgeGateFailsClosed` (written after the code; a bountied
      verdict with a malformed marker moves no coins).
    - `TestClaimIntegrity_ShieldChallengeBehindAugmentationEdgeLaundersNothing` now applies its
      verdict before activation, since the gated writer no longer writes `<F>/0` after it.
  - Mutants: 25 on a /tmp copy, each killed by at least one test: release on an incomplete
    check (two endings); R2 always scanning; the claim's type deciding; no stored-edge
    confirmation; no pair check on the looked-up edge; reading only cascade-typed claim
    relations; a missing claim read as no
    dependencies; an unreadable claim falling back to the scan; seven per-item stops (R2: missing
    dependency, history, named edge; genesis scan; R1: entry, missing fact, unreadable fact); no
    failure cap; a stopped scan hidden by the cap; the gate without its slash, length or fact-ID
    rule, with the cap off by one, ignoring the marker, swallowing a marker error, or decided
    after the payout; an incomplete check emitting nothing.
  - Residuals, all griefing (a fact stays CONTESTED or a settle names K), none a release:
    the disclosed limitation (§5.3, §8); R1's reverse prefix still holds one incoming
    REFORMULATES entry per passing augmentation of F, so long IDs admitted before review
    neutrality can fill R1's byte budget; `settleCounterFactTargets` still scans K's forward
    prefix, so an inflated prefix stops K's settle (no target released, the event names K);
    and the edges written before activation remain (§8).

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
  - a `SubmitClaim` carrying 16 CONTRADICTS relations excludes **only** its submitter (after
    review G2: admitted before activation; see the G2 block in Task 10);
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
- Done in Task 14:
  - `claim_integrity_review.go` holds `refuseConflictedReviewer`, `conflictedParties`,
    `disputedFactID` and `sameActor`. `StoreCommitmentInRound` reads the marker once, after the
    phase and scheme checks and before the duplicate and equivocation lookup, and runs the check
    only when the marker is on. A party's new commitment after activation is therefore refused
    as a party, not as equivocation.
  - Refusal text: `reviewer <r> is a party to claim <id> (submitter)` or
    `(defender of fact <F>)`, followed by `: verifier not qualified for domain`, the registered
    text of `ErrUnqualifiedVerifier`.
  - `sameActor` treats two empty strings as different actors, and bech32 data of length zero
    as no key.
  - The tx path refuses an operator (valoper) address before this check, at the balance gate.
    A scheme-2 round refuses one at `CanonicalReviewAddress`. Only a legacy-scheme round, the
    kind the vote-extension writer reaches, gets as far as the party check with such an
    address.
  - Failure choices the plan left open:
    - an unreadable claim, an unreadable disputed fact or a malformed marker refuses the
      commitment (`read claim <id> of round <id>: …`, `read disputed fact <F> of claim <id>: …`);
    - an absent record names no party, the same way `getClaimRecordChecked` treats an absent
      historical claim. Three cases:
      - a legacy round may have no claim; the `abci_test.go` `StoreCommitmentInRound` tests
        commit to such a round under `DefaultGenesis` (marker on), and they are unchanged;
      - a neutral round without its claim is already refused when the round is saved;
      - facts are never deleted on chain, so an absent disputed fact means store damage. In
        that case the submitter is still refused.
  - Vote-extension path. The release latch keeps `ProcessVoteExtInjection` closed, so the test
    cannot reach the loop. `app/abci.go`'s per-commitment store-and-log block therefore moved,
    unchanged, into `storeInjectedCommitment`, which the app test drives directly. A party
    refusal is skipped and logged at the existing Debug level (`skipped commitment from vote
    extension`, with the refusal text as `reason`).
  - Tests beyond the list:
    - `claim_integrity_review_test.go`:
      - the challenger is also refused on a `ChallengeFact` round;
      - the defender is refused on a `MsgChallengeProvisionalFact` round (the conjecture's
        proposer);
      - a `SubmitClaim` with a single CONTRADICTS relation (a REVIEW_FEE claim) recuses no
        defender (after review G2: admitted before activation; see the G2 block in Task 10);
      - upper-case and foreign-prefix encodings of the submitter's key are refused;
      - a pre-activation self-commit settles both after an in-place upgrade and after an
        import, and the submitter is paid;
      - three faults, and the absent-fact case.
    - The app test is `app/claim_integrity_review_test.go` (`package app`).
    - `claim_integrity_review_internal_test.go` (package `keeper`, outside the file list):
      `TestSameActor` and `TestDisputedFactID`, written after the implementation. The second
      pins shapes that no message creates.
  - Mutants: 17 were run on a /tmp copy, and each fails at least one test. They were:
    - the marker ignored;
    - the check never run;
    - string-only `sameActor`;
    - no defender;
    - the funding kind ignored;
    - an unreadable fact read as no party;
    - an unreadable claim read as no party;
    - the check moved after the equivocation lookup;
    - the marker read error ignored;
    - empty strings read as the same actor;
    - the empty-data guard dropped;
    - `ProvisionalFactId` ignored;
    - any relation type accepted;
    - the relation count ignored;
    - the nil-relation guard dropped;
    - an absent fact refused;
    - an absent claim refused.
  - Cost with the marker off: one 0x87 read per commitment, on both paths. Nothing is written.
    The Task 2 digest is unchanged (`f849e099…`), and no existing test changed.
  - `state.go` and `app/abci.go` are not authority-geometry anchors, so nothing new needs
    re-pinning. The `knowledge-writers` and `knowledge-slash-callsites` re-pin is still deferred
    to the separate reviewed pinning step.

### Phase D — documentation and verification

**Task 15 — Specification** · consensus: no
- Files: new `docs/specs/knowledge-claim-integrity-v1.md`, covering §3–§5, the cause vocabulary,
  the events, the reviewer rubric, the quote guard algorithm, and what the chain cannot prevent
  (§11). Also add a pointer from `docs/specs/knowledge-claim-records-v1.md`: after v12,
  admission no longer flips the target.
- Failing test: none (documentation).
- Acceptance: every behaviour claim in the spec cites a test name from Tasks 3–14.
- Done in Task 15:
  - `docs/specs/knowledge-claim-integrity-v1.md` covers §3–§5, the cause vocabulary, the
    events, the reviewer rubric, the quote guard and §11. Each behaviour statement ends with
    a `Tests:` line. Statements the tests do not check are labelled *Not tested here*, with
    the reason (read from code, or a convention the chain does not enforce).
  - `docs/specs/knowledge-claim-records-v1.md` has a paragraph after the admission rules: with
    knowledge 12, admission writes no target status, the target becomes CONTESTED only if
    the contradiction is accepted, and the Task 10 admission checks apply.
  - The acceptance check was an uncommitted script under `/tmp`. Before the spec existed it
    failed. It now passes, and it checks three things:
    - every backticked test name in the spec (131) is defined in a tracked test file, and
      the two quoted TypeScript test titles exist;
    - every Go and Python test function in the Task 3–14 test files is cited at least once;
    - the Task 3 and Task 11 tests, which live in shared files, are cited too.
  - The spec also cites Task 2 and pre-existing tests (`TestV11_*`,
    `TestSubmitContradiction_StarvedDoesNotLockFact`, `TestSubmitClaim_ContradictionAutoContests`,
    `TestEventAudit_DocumentationCompleteness`), always next to a Task 3–14 test.
  - Gaps in the tests, labelled in the spec:
    - no test ends one contradiction while another is still pending, so "pending
      contradictions are not reasons" is read from code;
    - `settleContradictionTargets` has no test for an unreadable target or for a write error
      (the counter-fact path has both);
    - the resubmission fee, cooldown and `--resubmit-after` flow are checked only by the §7
      sandbox run.
  - The spec words §11's "a starved dispute changes no status" more precisely: a starved
    contradiction changes no fact, and a starved challenge clears no contest reason. (A
    starved challenge still moves a fact out of CHALLENGED.) §11 here is unchanged.
  - Not done, deferred to the separate reviewed pinning step:
    - `knowledge-claim-records-v1.md` is pinned in `.specs-and-plans-hashes`. Its pin has
      not matched since Task 3 (`52bb3848`), and this edit changes the file again, so
      `scripts/check_specs_and_plans_hashes.sh` (`make creed-check`) fails until the re-pin.
    - The new spec is not added to `.specs-and-plans-hashes`, where the sibling knowledge
      specs are pinned.
- Review fixes, group G5 — documentation (2026-10-17; docs only, no code or test change):
  - F4: the transition cause records the event that last changed the status, not the current
    reason. §5.5 and spec §6.2 and §9 now say so, and the consumer recipe recomputes R2 as well
    as R1. A "who said P" reader checks the `attr-v1` tag, the predicate grammar, the work ID
    and the template before treating a `structure.object` link as an attribution (§5.5, spec
    §2.5 and §9).
  - F6: the status line at the top.
  - TQ-6: one list of the deferred checks, and `check_specs_and_plans_hashes.sh` among the
    verification commands (§6 conventions). Hashing the anchored files at every branch commit
    showed that `knowledge-slash-callsites` went stale at the Task 7 follow-up, not Task 12.
  - TQ-7: post-hoc "Red observed" lines for Tasks 3–5, with mutant checks where the red was
    vacuous (Task 3's second test) or a build failure (Task 5). Scratch:
    `/tmp/zerone-v12-g5` (`logs/`, `tools/mutants.py`).
  - §8: the relation-graph follow-up from the G1 re-review, stated in full with both proposed
    remedies. One more open case was found: the stored-fact check cannot catch a fact created
    after the verdict (also spec §1.5, §6.6, §11, and the "one other writer" sentence of §5.3
    and spec §6.3).
  - Consistency with G1–G4: the survival cutoff is the height the surviving challenge opened in
    §1 and §2, in the spec's introduction and §6.5, and in the `contradiction_reversed` and
    `challenge_inconclusive_restored` entries of `docs/EVENTS.md`. The Task 8 and Task 13 test
    bullets carry notes where G2 or SEC-1 changed the outcome. Task 16's recorded result is
    added.
- Cleanup pass (2026-10-17; no consensus code changed):
  - `e2e41a59` fixes the dashboard type error `cdb937e0` introduced: the v12 history value in
    `dashboard/tests/development.test.ts` is typed with `related_claims: unknown[]` instead of
    inheriting the fixture's `never[]`. Typing only; `npm test` is unchanged (497 of 499 pass,
    the two authority-geometry tests fail as deferred).
  - Docs: the equal-ID residual of pre-activation augmentation edges is stated (spec §1.5,
    §6.3, §6.6, §10, §11; here §3.6, §5.3, §8, §11). The deferred-check list is complete (the
    SDK's generated codecs and the dashboard's authority-geometry checks were missing), and the
    verification commands name the dashboard and SDK checks. The status line names G1R.
- Final acceptance fix: import rebuilds the subject and tag indexes (2026-10-17; consensus:
  yes, `InitGenesis` with the flag on only):
  - Found by the final acceptance run, `/tmp/zerone-v12-e2e3/index.json` row 10:
    `facts-by-tag doi:10.1000/xyz123` named G's fact before export and nothing after import,
    although the fact and its `structure.tags` were imported. Cause: 0x32/0x33 are written
    only by `IndexFactBySubject` from `rounds.go` at fact creation, `InitGenesis` never
    rebuilt them, and the export has no index field. Pre-existing at a4272bf (the a4272bf
    snapshot below imports a structured, tagged genesis with zero 0x32/0x33 entries). It
    broke the §5.5 `FactsByTag("zc:<40hex>")` lookup on any chain started from an export,
    which is D4's relaunch option.
  - Red observed: `TestClaimIntegrity_ImportRebuildsSubjectAndTagIndexes`, flag on, both file
    orders: the imported 0x32/0x33 entries were `nil` against the source's 15. Its flag-off
    twins passed, pinning knowledge 11. `TestClaimIntegrity_ImportIndexesInCreationOrder`
    failed the same way (`FactsByTag("golden-shared")` empty).
  - Change (`keeper/genesis.go`): with `gs.ClaimIntegrityEnabled`, right after the facts are
    set, `InitGenesis` calls `IndexFactBySubject` for every fact with a structure, in the
    order of `factsInCreationOrder` (§3.6). The round ID comes from the claims in the same
    file, read through a lookup map that is never iterated.
  - Which fact wins the subject entry: the fact created last, as live. The live chain
    creates facts only in `BeginBlocker` → `AdvanceRoundPhases`, which completes a block's
    rounds in active-round key (round-ID) order; `VerifiedAtBlock` is set only at creation;
    a claim has one round, created at its admission; and `DeleteFact`/`DeleteClaim` have no
    caller outside tests (all read from code).
  - Tests (new files `claim_integrity_import_index_test.go`,
    `claim_integrity_import_golden_test.go`):
    - `TestClaimIntegrity_ImportRebuildsSubjectAndTagIndexes`: an accepted attribution
      (`attr-v1`, `stmt:`, `zc:` tags), a fact with mixed-case, padded, blank and repeated
      tags, two facts on subject X in different blocks, and two on subject Y accepted in one
      block whose round-ID and fact-ID orders disagree (asserted). Export, then import with the
      flag on and off, in file order and reversed. Flag on: the 0x32/0x33 entries equal the
      source's entry for entry, `FactsByTag` (including `ZC:…` upper case) and
      `FactsBySubject` return the same IDs, and admission warns of the same duplicate subject.
      Flag off: no entry, no answers, no warning.
    - `TestClaimIntegrity_ImportIndexesInCreationOrder`: a hand-written genesis whose last
      subject-S fact in the file is not the one created last; a block-40 tie with no claims
      goes to the greater fact ID; a tag-only structure gets tag entries and no subject entry.
    - `TestV11_GoldenImportStoreDigest`: the same hand-written genesis imported with the flag
      off. Its digest `97c340e3…7528` (318 entries) was recorded by importing the identical
      genesis on an `a4272bf` snapshot (`git archive` into
      `/tmp/zerone-v12-importidx-a4272bf`, the flag line dropped, the fixture and digest
      copied verbatim); the branch gave the same digest before and after the fix.
    - Mutants, each restored: indexing in file order fails both flag-on tests; dropping the
      round-ID tie-break fails the live round-trip test. (The first Y pair happened to order
      both ways alike; the texts were changed to a pair that disagrees.)
  - The Task 2 digest is unchanged (`f849e099…`). `go test -count=1 ./x/knowledge/...
    ./app/...` passes.
  - Not changed: the §3.3 migration (the in-place upgrade keeps the live entries); the export
    format; the deferred-check list (see the note under "Deferred checks").
  - Not done: the sandbox run was not repeated for row 10; the keeper round trip above
    stands in for it until the next end-to-end run.

**Task 16 — End-to-end probe re-run and upgrade rehearsal** · consensus: no (verification)
- Files: nothing tracked. Artifacts go under `/tmp/zerone-sandbox-<date>-v12/`.
- Acceptance: §7 in full. Record observed versus expected for each line, and report any line that
  cannot run as BLOCKED with the reason. A missing check is not a pass.
- Done in Task 16 (recorded in review G5 from `/tmp/zerone-v12-e2e/index.json`, which was read
  for this note; the run itself was not repeated): the run built the branch at `cb4d7dfd`.
  Rows 0–19 passed, row 18 for the `a4272bf` source. Row 18's repeat from a `5542b22` sandbox
  is BLOCKED. No row failed. The run predates review fixes G1–G5, so it does not cover them; it
  has not been re-run since.

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
| 16 | Attribution round trip | — | `claim-workflow attribute --dry-run` (the default; the flag is an explicit no-op) prints the §4.2 fields. Broadcast proposition P and attribution A1. `ClaimHistory(P)` lists A1 under `structure.object`. A variant carrying 8 excerpt words is refused before signing, and the excerpt does not appear in any artifact. |
| 17 | Export and import | — | `zeroned export` on the stopped sandbox shows the flag as true. Import into a fresh home starts, and has no `migration_v12_complete` (no invented receipt). `facts-by-tag` (an accepted attribution's `zc:` tag) and `facts-by-subject` return the same facts after import as before (added after the final acceptance run's row 10). |
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
- **A bounded R1: the disclosed limitation that replaced SEC-6.** A fact whose contest check
  cannot finish (more than 100k incoming relations of any type, or 64 MiB of them) stays
  CONTESTED once any challenge of it ends, even a starved, refunded one, and no release path
  frees it; only a challenge that refutes it moves it (§5.3; review G1R). This is griefing, not
  laundering. The remedy is an index of CONTRADICTS edges alone (a new store key), so that R1
  reads only contradictions and a heavily referenced fact can leave CONTESTED once its reasons
  are gone.
- **Relation-graph integrity for augmentation edges** (pre-existing; found in the G1
  re-review, recorded here in G1 and G5). Augmentation IDs are chosen by the submitter, and
  `SubmitAugmentation` checks an ID only against other augmentations. A passing augmentation
  writes a REFORMULATES edge from that ID to the augmented fact
  (`ApplyFinalizedAugmentationVerdict`) into the same relation index as fact edges, keyed
  `<source>/<target>`, one edge per (source, target) pair. Two defects follow:
  - an augmentation whose ID equals fact F's ID, augmenting D, overwrites F's own edge to D at
    the same key `F/D` (a CITES edge becomes REFORMULATES, which the cascade does not follow);
  - an ID such as `<F>/x` shares F's forward prefix `F/`, so prefix readers such as
    `GetFactRelations` and `FactRelations` return its edges among F's outgoing relations
    (tested).

  Done with the marker on (review G1R): the writer withholds the edge when the ID contains `/`,
  is longer than 128 bytes, or is a stored fact's ID, and emits `augmentation_edge_withheld`.
  Entries under `<F>/` are never read as F's edges: R2 reads a claim-born fact's edges by key
  from its claim, and its prefix scan of a claimless fact skips entries whose source is not F
  (they still count toward the bound of any scan that reads them). G1R and G5 wrote that the contest check does not depend
  on the rest at all. That is false for the equal-ID shape below (corrected in the cleanup
  pass, 2026-10-17).
  Left open:
  - the edges written by augmentations finalized before activation. They form a closed,
    auditable set: each is an augmentation record with a passing verdict and a `verdict_block`
    below the activation height. The contest check depends on one shape of them. An
    augmentation of D whose ID equals the ID of a fact F that existed at the verdict, where F
    had its own edge to D, replaced that edge (the forward key `F/D` and its reverse entry)
    with REFORMULATES. R1, R2 and the cascade all read the stored edge, so for that pair:
    - if F's edge was in the cascade set, R2 no longer counts D as F's dependency, and a later
      refutation of D does not contest F;
    - if it was CONTRADICTS, R1 no longer counts F as a standing counter-fact of D.

    A fact that only the replaced edge held CONTESTED is released at its next check: when a
    contradiction or challenge of it ends without refuting it (a starved one included), or
    another counter-fact of it is disproven. Tested:
    `TestClaimIntegrity_AugmentationEdgeStaysOutOfTheFactIDSpace` (marker off, "ID equal to a
    fact's ID": both entries replaced) and `TestContestReasonReadsClaimBornDependenciesFromTheClaim`
    ("named, stored edge overwritten with another type": no R2 reason, for a claim-born and a
    genesis F). The R1 half is read from code (`standingCounterFact` skips an entry that is not
    CONTRADICTS; `TestContestReasonStandingCounterFact`), not tested end to end; an uncommitted
    test run once in the cleanup pass (the real writer, marker off, ID equal to the
    counter-fact's) confirmed that R1 then finds no reason. With the marker on no fact's own
    edge is replaced, so these pairs cannot grow;
  - an ID equal to a fact created after the verdict. The stored-fact check runs only when the
    verdict is final, and fact IDs are derived from public values (`GenerateFactID(claim ID,
    verdict height)`). If that fact's claim names the same target, the fact's own write
    overwrites the augmentation's edge. Otherwise the stray REFORMULATES edge stays under the
    fact's prefix. Review neutrality refuses new augmentations, so only IDs chosen before it
    can do this.

  Proposed remedies, for Yu to choose:
  - (i) namespace augmentation edge sources: write them under a source no fact ID can take,
    such as a reserved prefix, or in an index of their own. This closes both defects whatever
    the ID;
  - (ii) refuse IDs in the fact-ID space when the verdict is final: the three G1R shapes plus
    every ID of the generated shape (32 lowercase hex characters), so no future fact can share
    one.

  Either needs a reviewed migration (or an explicit decision to leave them) for the
  pre-activation edges, including whether to restore the fact edges an equal ID replaced.
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
  - Decision (a), taken by Claude on 2026-10-17 and reversible (review G2, SEC-2 and F5): with
    the marker on, `MsgSubmitClaim` may not carry a CONTRADICTS relation at all. It is refused
    at admission, before any target is read, with an error that points to
    `MsgSubmitContradiction`. Otherwise one REVIEW_FEE claim whose true but unrelated text a
    panel accepts contested up to 16 facts at once, past the standing, minimum-stake and text
    checks of Task 10. Other relation types are unchanged. With the marker off, knowledge 11
    admits it. The alternative is to apply the Task 10 checks to each CONTRADICTS relation of a
    `MsgSubmitClaim`, which needs per-target stake accounting.
  - Cost of decision (a) (review G2 re-review): a finding that should contest another goes on
    chain only as `MsgSubmitContradiction`, which carries no structure, `method_id` or
    `reasoning_trace`, so it has no form v2 and no 0x34 entry, and its attributions name only
    its content hash. A structured proposition cannot later contest with its own text: while
    it is live or stands, a contradiction carrying that text is refused as a duplicate (§4.3).
    The per-target alternative above would let one structured claim both record and contest.
- **D2. What an accepted contrary finding does.**
  - Default: **asymmetric**. The target becomes CONTESTED; the counter-fact stays VERIFIED.
  - Default: a target that survives a direct challenge **stays CONTESTED** while the counter-fact
    stands. The counter-fact must itself be refuted.
  - Decision (b), taken by Claude on 2026-10-16 and reversible: an EXPIRED counter-fact still
    stands for R1, because patronage or metabolism can revive it. Only SUPERSEDED, DISPROVEN,
    REVOKED and PRUNED counter-facts have fallen (review fix SEC-5). The same decision makes an
    EXPIRED fact block dedup (D3 below), implemented in review G2 (SEC-3).
  - Alternatives: symmetric CONTESTED, which freezes the counter-fact out of fitness; or letting
    survival clear R1.
- **D3. What releases text for resubmission.**
  - Default: no-verdict, negative-verdict and fallen states all release it (§5.4).
  - Decision (b) (above) supersedes this default for an EXPIRED **fact**: it can be revived, so
    it counts as standing and blocks. The plan first listed it as fallen. Only SUPERSEDED,
    DISPROVEN, REVOKED and PRUNED facts release. A claim whose status is EXPIRED (the legacy
    no-quorum outcome) still releases.
  - Minimal alternative: only the INSUFFICIENT and EXPIRED claim statuses.
- **D4. zerone-dev-1.**
  - Whether to upgrade it in place, which bundles the 64-commit `5542b22..a4272bfa` delta and
    needs that delta audited first, or relaunch it from a fresh genesis. A relaunch from an
    export keeps `FactsByTag` and `FactsBySubject` only because `InitGenesis` now rebuilds the
    subject and tag indexes with the flag on (§3.6); before that fix it emptied them.
  - The height H2, the proposer and the voting path.
  - Who reviews. dev-1 has never had reviewers, and after P5 a submitter cannot review their own
    claim, so every claim will starve (now harmlessly) until there are `MinVerifiers` independent
    reviewers.
  - Pre-upgrade rehearsal check (review CI-REVIEW-1). Export the source state at H−1 and group
    its claims by `content_hash`. Every hash held by more than one claim is a knowledge-11
    collision (§3.6). For each group, record the record the live 0x09 entry names (the one saved
    last) and the record an import would name (the one admitted last, by `submitted_at_block`
    then ID), with each one's status and, for an accepted claim, its fact's status. List every
    group where the two disagree about blocking, or where any member stands. Nothing is
    rewritten; the list decides between an in-place upgrade and a relaunch, and tells reviewers
    which texts can be admitted again.
  - How to read the live entry: neither the genesis export nor any query exposes 0x09, and the
    export does not record save order. Read the raw key at H−1 from a node that still holds
    that height, for example CometBFT `abci_query` with path `/store/knowledge/key`, data
    `0x09` followed by the ASCII bytes of the 64-hex content hash, and `height` H−1; the value
    is the claim ID. The live-entry column is informational: two records disagree only when one
    of them blocks, so listing every group in which any member would block (is live or stands)
    already lists every disagreement.
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
  the cost is locked capital and gas. After this branch a starved dispute changes no status,
  with one disclosed exception: a fact whose contest check cannot finish (very heavily
  referenced) returns from any challenge to CONTESTED and stays there (§5.3, §8). Separately,
  a pre-activation augmentation whose ID equalled an existing fact's could replace that fact's
  edge and so remove a reason; any later check, a starved challenge's included, then releases
  the fact it held (§8).
- **Mempool copying of claim text.**
- **Verbatim quotes from third-party clients.** Only reviewers marking MALFORMED push back on chain.
- **Reviewer independence and the truth of anything a panel accepts.**
