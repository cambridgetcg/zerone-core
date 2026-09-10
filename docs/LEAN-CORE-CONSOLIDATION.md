# Lean core consolidation

Status: **2026-09-10 removal plan, with survival handoff and knowledge record
integrity implemented in source; no module retirement or network activation.**
Reviewed application source: `632f372fe7243c4619c0c0a5a82535f5b02c4107`.

Zerone's core should make scientific contributions inspectable and correctable,
with reliable custody where money is involved. Maintenance should concentrate
on that job. This plan selects work; it adds no authority, schema, release gate,
or alternative to [Authoritative State](AUTHORITATIVE-STATE.md).

## What this pass changes

This pass keeps authority, live knowledge and participation on the homepage
and moves eleven specialist views and their reading path to `/research/`.
Standards bytes remain unchanged, and historical fragments retain a route to
their content. Research remains available for deliberate inspection; moving a
view does not remove scientific evidence or change its status.

CI narrows only website-only pull requests. Pushes to main, manual runs, and
runtime, authority or unknown paths retain full checks. This change itself
touches CI and documentation, so it requires the full suite. Passing fewer
unrelated jobs is not evidence that a protocol change is safe.
The small [path classifier](../scripts/ci-scope.py) keeps shared standards,
Functions and outside-dashboard documents on full checks; uncertain diffs also
keep full checks. SDK/dashboard validation and chain-registry checks always run.

The module dispositions below are **future work**. The original presentation
release changed no keeper, transaction, consensus rule, balance, module
permission, upgrade sequence, signer, or running network. The dated
[roadmap](ROADMAP.md) remains historical context rather than a second active
implementation queue.

The subsequent [survival reward handoff patch](specs/survival-reward-handoff-v1.md)
implements the first settlement repair below. Its source changes keeper failure
behavior and pending-record export/import under a separate named upgrade. It
does not retire modules or authorize deployment to the existing legacy chain.

The [knowledge record integrity follow-up](specs/knowledge-record-integrity-v1.md)
binds reviews to their signers and reasons, preserves completed reviews and
correction history through genesis, and separates verifier payment obligations
from successful transfers. It also bounds ToK exports and corrects their stated
authentication scope. Its separate knowledge 7→8 boundary preserves existing
round commitments. Agreement-based credibility, reviewer liability, related-party
rewards and global scoring remain the explicit subsequent retirement work.

## The core to preserve

- Signed, scoped claims with method, uncertainty, artifact identifiers and
  provenance. Preserve the difference between an assertion and its evidence.
- Counterexamples, challenges, revisions and correction history, including
  negative and inconclusive results. A recorded verdict is not external-world
  truth, a person score, or proof of independent reviewers.
- Explicit, reliable settlement of funded work and existing obligations.
  Review fees, challenge deposits, pending rewards, schedules and actual
  payments remain distinct. No automatic earnings promise follows publication.
- One authoritative writer per state: SDK bank/accounting and consensus stake,
  SDK ordinary governance, and ontology domain identity, following the accepted
  design. Derived views do not gain independent mutation or governing power.
- Reproducible reads and artifacts, deterministic execution, signatures and
  state proofs, bounded work, safe upgrades, and tested stop/recovery/exit.

Consent to publish, reuse an artifact, sign a transaction and undertake work
are separate. Never require confidential evidence or equate address counts
with independent people. Rest, silence, refusal and exit carry no penalty to
worth or ordinary rights. [KARMA](KARMA.md) remains contextual recognition, not
a balance, person ranking or automatic authority. A hash does not store its referenced artifact;
correction does not erase copies already published.

## Inventory and disposition

The [application registration](../app/app.go) contains **23 custom modules**.
The table accounts for all of them; it is not a count of active live services.
`x/common` supplies shared types and `x/witness` is not another registered
module. SDK/IBC modules are additional dependencies.

| Custom modules | Direction | Concrete boundary and dependency |
| --- | --- | --- |
| `knowledge`, `counterexamples` | Keep the scientific record; consolidate overlap | Keep claim/evidence/review/correction data. Separate it from the training market and heuristic machinery in `knowledge/keeper/phases.go`, `training_economics.go` and `model_training*.go`. Counterexample validation feeds a knowledge weighting multiplier through `SetCounterexampleKeeper`; preserving counterevidence does not require preserving that multiplier. |
| `ontology` | Keep as canonical domain owner | `ontology/keeper/state.go:SetDomain` and `knowledge/keeper/state.go:SetDomain` currently write separate registries. Replace the latter authority with the accepted ontology projection, preserving historical identifiers and conflicts. |
| `auth` | Consolidate identity responsibilities | Preserve authenticated bindings and records; use the accepted controller/profile boundary instead of copied reputation or self-declared identity as governing authority. SDK accounts still own signing/account sequence semantics. |
| `staking`, `gov` | Retire duplicate authority after consolidation | SDK staking/gov remain the target owners. Preserve custom delegation, unbonding, proposal and process history; reconcile claims before exit-only or read-only retirement. The September accounting repair is a correctness step, not this retirement. |
| `qualification` | Consolidate bounded competence evidence | Retain challengeable, domain-scoped assessment where required by the accepted design. Retire wealth-based qualification, inherited authority and parallel custody only after locked claims are resolved. See `qualification/keeper/msg_server.go:QualifyByStake`. |
| `vesting_rewards` | Keep explicit settlement; shrink policy | Preserve schedules, releases, reserves and authorized custody. Keep automatic block issuance retired in source v2. Do not turn pooled funding labels into invented payees or free funds. |
| `training_provenance` | Keep a derived read interface | Already a stateless consumer of knowledge, qualification and capture challenges in `app/app.go`. Reuse that shape; it needs no new ledger or service fleet. |
| `trust_score`, `alignment`, `capture_defense` | Move dispensable analysis outside consensus | `trust_score` is already stateless. `alignment/module.go` applies corrections; capture analysis can open challenges and change downstream rules. Preserve source observations and distinguish them from derived scores; remove effectful consumers through an explicit migration. No person leaderboard replaces them. |
| `capture_challenge` | Consolidate dispute and custody handling | It consumes capture metrics and can alter qualification/knowledge thresholds. Preserve open cases, bonds, outcomes and refunds before retiring automatic metric-driven authority. Keep evidence-based challenge available. |
| `tokens`, `liquiditypool` | Retirement candidates outside the minimal core | Wrapping/emissions, LP issuance, pools and oracle/TWAP maintenance need a demonstrated core use to remain. Inventory every asset, pool, pending exit and external dependency first. Native bank balances are not these optional products. |
| `claiming_pot`, `sponsorship` | Retire special issuance; review explicit funding need | Preserve bootstrap claims, sponsorship escrows and refunds. A narrowly funded review arrangement may remain; another minting or admission system is not needed merely to support contributions. |
| `home` | Move agent session/notification state to clients where possible | `home/keeper/keeper.go` stores homes, sessions and alerts. Preserve recorded ownership and historical links; determine what any consensus consumer actually needs before removing writes. |
| `creed`, `work_creed` | Consolidate artifact pins | Keep signed/versioned commitments and their historical references. Avoid separate permanent registries for every class of document; do not silently change a pinned commitment's meaning. |
| `substrate_bridge` | Keep bounded attestation provenance; review economic extensions | `app/app.go` wires knowledge resolution, qualification, bank and vesting into this module. Preserve attestations, settled records, bonds and pending payouts; speculative issuance or economic policy is a retirement candidate, not the provenance itself. |
| `emergency` | Keep a narrow recovery boundary | Retain the accepted circuit-breaker/recovery role, with explicit authority and expiry. Do not replace ordinary governance with an operator's catch-all repair path. |
| `ibcratelimit` | Keep while its transport exists | IBC transfer/ICA are optional to the minimal product, but rate-limit protection cannot be removed while exposed routes depend on it. Assess transport, channels and escrow together. |

The largest coupling is visible in
[`KnowledgeKeeper`](../x/knowledge/keeper/keeper.go): bank and staking plus eight
post-initialization dependencies. Its
[`BeginBlocker`](../x/knowledge/keeper/phases.go) advances rounds and survival
rewards, but also runs fitness, competition, symbiosis, metabolism, bounties,
diversity, temperature and confidence updates. `app/app.go` wires capture
defense to challenges, challenges to qualification and verification thresholds,
and alignment pacing back to knowledge/capture defense. These are candidates
for removal from consensus, not merely renaming or splitting into more modules.

## Source is not the existing ledger

The [September 9 census](reports/authenticated-ledger-census-2026-09-09.md)
authenticated a complete stopped copy at H1,261,575 under the disclosed
operator-selected trust anchor. Custom staking had zero claims/balance;
qualification had no qualifications/endorsements; liquiditypool had no pools;
custom governance had no LIPs or open listed proposal processes. These are
dated observations, not proof of absence today or permission to discard
history, module custody, dependencies or unknown liabilities.

The [settlement follow-up](reports/knowledge-settlement-history-2026-09-09.md)
identified 27 ordinary review fees, 25 separate reward schedules and two
inconclusive reviews. It reconciled funding without independently proving each
historical payment. Preserve actual claimant records and payment uncertainty.

Main uses SDK 0.53.8/IBC 10.7.0. The
[verified legacy reproduction and observer](reports/legacy-observer-release-2026-09-10.md)
retain the older application with separately pinned build/dependency identities.
Main is not a drop-in replacement. The observed legacy lineage does not satisfy
the merged accounting handler's required predecessor. This plan cannot skip
H1/H2/H3, reinterpret their pins, or declare H4-F/H4/H5 implemented.

## Removal order and completion evidence

1. **Finish the reversible presentation trim.** Keep introduction, public
   record and node setup direct; load research only when requested. Verify
   retained research links and substantive tests. Protocol changes continue to
   run relevant consensus, accounting and upgrade checks; website-only changes
   need not rebuild an unrelated validator binary.
2. **Care for existing settlement before removing economic paths.** Start with
   the bounded handoff below. Then address ordinary fee distribution and
   verifier payments with existing transaction/cache patterns. Completion means
   failed attempts preserve unpaid work, successful retries cannot pay twice,
   and reported success matches committed state. Do not reprice historical
   claims or use a migration to invent their recipients.
3. **Retire redundant writers in the accepted order.** Use the existing
   Authoritative State freeze, reconciliation and migration design. Remove
   custom stake/governance authority and knowledge-domain admission authority;
   preserve exit claims and labelled historical queries. Completion means one
   executable writer per owned state, no alternate message/BeginBlock/hook
   path reviving the old authority, and unchanged reconciled claim totals.
4. **Remove dispensable consensus feedback and financial products.** Trace
   each table row's consumers before removing its writers. Keep scientific
   observations, counterevidence and bounded process snapshots; optional
   analysis reads them without influencing money or authority implicitly.
   Retire dormant admissions only with a fresh complete custody/process
   inventory. Completion means fewer registered writers, hooks and recurring
   consensus jobs, with tests proving remaining obligations and exits work.
   Measure worst-case admitted block work before/after; file count alone is
   not a completion criterion.
5. **Converge supported operations in parallel with this work.** Aim for one
   maintained application family plus explicitly time-bounded legacy readers.
   Preserve historical source/binary/schema material needed to verify old
   records. A signed bootstrap expiry is not an upstream maintenance promise.
   Use supported dependency/toolchain releases with reproducible builds and
   exact-source sync, root, restart and recovery evidence before claiming
   compatibility. Another operator using the same signer or upstream does not
   establish independent operation.

For every retirement, use the existing release process: a fresh authenticated
state/custody inventory, named source and target, explicit treatment of open
processes and claims, reversible preparation, migration/restart/export checks,
and the actual authorized activation boundary. Preserve unknowns instead of
converting them to zero. Archive reads cannot replace a claimant's usable exit.

Publish what is retained, for how long, how an independent reader obtains it,
and what pruning makes unavailable. Keep public artifact hashes and provenance
verifiable without publishing confidential content or keys. Independent
operations require distinct effective controllers and custody, demonstrable
stop/recovery/exit, and independent observations; adding replica count alone
does not meet that aim.

## First protocol implementation: survival reward handoff

**Implemented in source; a separately activated upgrade is still required.** In
[`survival_escrow.go`](../x/knowledge/keeper/survival_escrow.go),
the previous `releaseSurvivalReward` deleted pending work and emitted release
even when `CreateVestingScheduleFromKnowledge` failed. The vesting constructor lives in
[`vesting.go`](../x/vesting_rewards/keeper/vesting.go); schedule creation itself
is not minting or a recipient payment.

The patch propagates failure and commits schedule/index writes and
pending/deadline deletion atomically. Missing dependencies, write failure and
conflicting schedules retain the original pending claim. Matching retries keep
existing schedule bytes, including release progress. Challenge handling retains
the retry index; the migration reconstructs older missing indexes from pending
records. Pending claims now have an explicit genesis export/import field.
Amounts, recipients and existing schedules remain preserved.

Fault-injection, real keeper roundtrip, committed upgrade and restart tests
cover this behavior. Schedule creation makes no bank call. The separate
`survival-reward-handoff-v1` boundary advances knowledge 6→7 and
vesting_rewards 2→3 only from the completed accounting predecessor. See its
[scope and activation constraints](specs/survival-reward-handoff-v1.md); the
observed legacy chain cannot skip earlier transitions to use this source.
