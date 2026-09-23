# Module consolidation sequence

Status: **implementation sequence, 23 September 2026**. App/authority baseline:
`8f40f59c96dd01c0d59b454a7254fda8bc632c89`; the local workflow implementation is
linked below. The app still registers all 23 custom modules. The journal/client
integration changes no keeper wiring, custody or admission rule and adopts no
controller/consent policy. H4-F, H4 and H5 remain pending.

[Authoritative State](AUTHORITATIVE-STATE.md) remains the accepted authority
design. [Lean core consolidation](LEAN-CORE-CONSOLIDATION.md) remains the product
direction: attributed scientific records, reproducible inspection, correction
and explicit funded settlement. A sole writer for a kind of state is a
consistency boundary; it does not establish independent operator control.

## Choose the record before admission

| Need | Existing path | Consequence |
| --- | --- | --- |
| Keep literature, a question, method, concern, assessment or divergent continuation | Local [Research Review journal](specs/research-review-journal-v1.md), export and browser reader | Attributed local records and plural branches; no consensus mutation, native verdict or payment entitlement. |
| Commit one exact research export to a signed chain transaction | Optional [research checkpoint](specs/research-checkpoint-v1.md) | Account-signed digest commitment; no native Claim/Fact or review round. Exact files require separate preservation. |
| Ask for a bounded, paid review of an assertion | Explicit signed native claim through [shared development](SHARED-DEVELOPMENT.md); the [research workflow adapter](specs/research-workflow-v1.md) binds a selected journal contribution to that intent | Public assertion, review deadlines, recorded funding terms, retained reviews and possible Fact. Evidence must be retrievable separately. |
| Present counterevidence to an existing native Fact | Explicit native contradiction/challenge with the actual Fact ID | Separate counterclaim, fees/deposit, review and retained relation; no automatic erasure or final falsification. |

Use the journal when no native admission is needed. A journal entry, DOI
metadata response, checkpoint, static research diagram or agent receipt must
not silently become a native claim or reward request. When deliberately moving
work into a claim, the submitter selects the exact assertion and evidence,
accepts the published costs, and signs that admission. Keep journal attribution,
account control, independent review and scientific validity as separate claims.

The implemented [local adapter](../tools/research-review/README.md) prepares an
exact journal/record/intent binding, delegates explicit submission to the
participant's pinned client, and retains its receipt and dated history in a
separate workflow file. The browser can prepare/import that file and explicitly
read matching native history using the published development pins. Importing
does not submit or fetch; the complete journal stays local. Saved observations
remain labelled observations, and a journal correction needs its own explicit
native action if the participant wants one.

The current neutral-review policy counts admitted accounts equally and pays
timely recorded reviews, including dissent. Its reusable 100 ZRN balance check
does not establish independent controllers. The accepted target instead names
controller-deduplicated, domain-qualified panels. Implementing that target needs
its own reviewed policy and migration; this plan does not quietly change the
current panel or impose identity collection on research readers.

## Sequence and completion evidence

1. **Use the supported development workflow.** Development publication is
   complete: separate knowledge-11 participant packages match the applied
   runtime, while version-10 genesis replay packages, immutable upgrade artifacts
   and existing homes remain available. See the
   [dated runtime observations](reports/development-publication-2026-09-23.json)
   and [participant guide](SHARED-DEVELOPMENT.md). The local journal → explicit
   native submission → shared history reader connection is implemented without
   changing the chain. Run the [participant review handoff](DEVELOPMENT-REVIEW-HANDOFF.md)
   with a willing reviewer's own key and judgment, retaining any relevant
   operator relationship.
   Review by a separate effective controller remains its own
   milestone. Completion records publication consistency, retained
   review/correction and payment observations
   separately; extra operator accounts do not satisfy independence.
2. **Reconcile before closing financial paths.** Take a fresh authenticated
   census for each affected chain and exact height: module bank balances,
   claimant records, pending payouts, vesting, deposits, refunds, open cases,
   assets and external dependencies. September's legacy census is historical
   input, not a current zero-liability certificate. Match every liability to
   its owner and exit route; preserve unknown entitlement instead of inferring
   it from a pooled balance. Existing source settlement repairs are prerequisites
   where their exact lineage applies, not permission to skip legacy upgrades.
3. **Unify authority in the accepted order.** Preserve the exact H1/H2/H3
   lineage where required. Implement H4-F drain/freeze and its committed census,
   then H4 reconciliation/migration, then H5 non-economic governance only after
   H4 verification. Retire custom stake/governance/domain writers and all their
   message, hook and block entry points while preserving claims and labelled
   history. H4 alone does not remove stake-weighted SDK governance. The accepted
   target includes controller, verifier-profile, electorate and legacy-claims
   responsibilities; their absence cannot be hidden by renaming old stores.
4. **Remove consensus work that the core no longer needs.** Trace readers,
   writers, automatic callbacks and custody for each candidate below. Move
   analysis and session functions to clients or derived reads, remove economic
   feedback explicitly, and close optional financial products only with usable
   exits. Any step that still depends on legacy authority follows stage 3;
   independent preparation and read-only tooling can proceed earlier.
5. **Verify the reduced application and supported operations.** Rehearse exact
   migration/restart/export/import boundaries and relevant rollback/recovery.
   Prove preserved obligation totals, working exits, no surviving alternate
   writers and bounded worst-case block work. Publish retained-query scope,
   client compatibility and archive availability. Fewer files or passing unit
   tests alone do not establish successful retirement or network activation.

Each protocol change still needs its named source/target, tests, release packet
and actual authorized activation boundary. Do not assign new version numbers or
reuse an earlier handler through this planning table. Independent participation
is useful product evidence, not a substitute for migration or custody proofs.

The current wiring must be traced before stage 4 removes or reconnects it.
Alignment's [corrections are advisory](../x/alignment/keeper/corrections.go#L107),
but its [pacing input](../app/app.go#L1100) still changes claim
[cooldowns and minimum fees](../x/knowledge/keeper/pacing.go).
[Ontology ceilings](../x/knowledge/keeper/confidence.go#L166) still apply to an
accepted neutral review. Neutral review therefore does not make all older
dependencies inert. Keep these effects separate from the retired review-score
bonuses and automatic issuance; changing them needs explicit behavior tests and
its own protocol boundary.

## Disposition of all 23 registered custom modules

“Stage” refers to the sequence above and describes a dependency, not a scheduled
height. No row is an instruction to delete its store today.

| Module | Direction | Required boundary before completion |
| --- | --- | --- |
| `knowledge` | Keep claims, reviews, correction history and funding settlement; remove unrelated training/ecology policy | Stages 2–4: preserve all terms, rounds, commitments, pending payments and fact relations; replace domain authority with ontology projection; trace legacy training and block consumers before removal. |
| `counterexamples` | Consolidate counterevidence into the scientific record | Stages 2–4: retain structured validations and references; retire the knowledge-weight multiplier separately from the evidence. |
| `ontology` | Keep sole domain identity/lifecycle owner | Stage 3: reconcile both registries, aliases, tombstones, statuses, revisions and every custody-bearing reference; no automatic taxonomy union. |
| `auth` | Keep authenticated account/key/DID bindings; separate controller/profile responsibilities | Stage 3: preserve rotations, freezes and capabilities; remove copied reputation authority; a DID never substitutes for controller uniqueness. SDK accounts retain sequence/signature semantics. |
| `staking` | Retire custom stake authority into bounded exits and legacy history | Stages 2–3: reconcile individual delegation/unbonding claims to actual custody and aggregates; preserve claimant choice; remove every custom consensus/eligibility authority consumer. SDK staking remains consensus owner. |
| `gov` | Retire custom ordinary decisions/execution | Stage 3: drain and preserve LIPs, votes, deposits, seats, research spends and phase records; SDK gov becomes sole ordinary executor. Do not replay old passed-but-inert actions. |
| `qualification` | Keep bounded non-economic competence evidence | Stages 2–3: refund or reconcile locked claims; retire stake-based qualification and inherited score authority; target evidence policy remains versioned and challengeable. |
| `vesting_rewards` | Keep custody, explicit schedules/releases, fee routing and mint-cap checks while callers remain | Stages 2–3: preserve accrued schedules/reserves and unpaid work; install the sole scoped research-fund disbursement and bank guard; keep automatic founder/block issuance retired. |
| `training_provenance` | Keep useful stateless derived reads | Stage 4: preserve source attribution and query compatibility as underlying knowledge/qualification/challenge owners change; no new ledger. |
| `trust_score` | Move optional analysis to derived/client reads | Stage 4: remove any authority/economic reliance before retiring the query surface; preserve labelled source observations, not a person ranking. |
| `alignment` | Move analysis/pacing outside consensus where dispensable | Stage 4: enumerate and retire correction/pacing effects on knowledge, capture defense and governance; preserve observations and establish replacement bounds where needed. |
| `capture_defense` | Move concentration analysis outside consensus | Stage 4: retain observations, remove automatic challenge/threshold consumers deliberately, and avoid treating address counts as controller independence. |
| `capture_challenge` | Consolidate bounded disputes and custody | Stages 2–4: settle open cases, bonds, refunds and outcomes; remove automatic metric-driven qualification/knowledge changes before retiring their writer. |
| `tokens` | Retirement candidate outside the core | Stages 2–4: inventory issued/wrapped assets, allowances, backing, emissions and external holders; define usable redemption/exit and preserve native bank balances. |
| `liquiditypool` | Retirement candidate outside the core | Stages 2–4: inventory assets, reserves, LP bearer claims, pending exits and oracle users; retain exits and safety restrictions until custody is discharged. |
| `claiming_pot` | Retire special bootstrap/general-pot issuance if unnecessary | Stages 2–4: identify existing claims and mint-cap callers, close prospective admission explicitly, preserve owed claims without inventing new payees. |
| `sponsorship` | Retain only a demonstrated explicit funded-work need | Stages 2–4: this spends existing funds, not new issuance; preserve orders, escrow, fulfillment, sponsor vetoes and refunds before consolidating with core funding. |
| `home` | Move session/notification state to clients where possible | Stage 4: inventory ownership, sessions, guardian alerts and memory references; preserve historical links and any required consensus consumer before closing writes. |
| `creed` | Consolidate signed/versioned artifact pins | Stages 3–4: preserve historical commitments and council records; migrate authority dispatch without changing the meaning of an existing pin. |
| `work_creed` | Consolidate genesis artifact pins | Stage 4: preserve every referenced sub-creed and consumers; current lack of a transaction API is not proof no history needs retention. |
| `substrate_bridge` | Keep bounded attestation provenance; review optional economics | Stages 2–4: preserve adapter revisions, attestations, challenges, bonds and pending payouts; retire speculative issuance only after knowledge callbacks and economic obligations are reconciled. |
| `emergency` | Keep narrow quarantine/recovery authority | Stage 3: implement accepted scoped, expiring and controller-deduplicated recovery; preserve incident history; no alternate ordinary treasury, electorate or domain authority. |
| `ibcratelimit` | Keep protection while exposed IBC transport exists | Stages 2–4: inventory channels, transfer/ICA routes, escrow and external obligations; remove protection only with the corresponding transport exposure safely closed. |

Registration and actual dependencies live in [app/app.go](../app/app.go).
SDK/IBC modules are additional dependencies; `x/common` and unregistered
`x/witness` do not add registered custom modules. Stage 4 should reduce mutable
writers, hooks and recurring block work even if stage 3 adds explicit target
owners. Measure those changes and preserved liabilities, rather than promising
a smaller module count in advance.
