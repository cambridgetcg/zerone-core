# Lean core consolidation

Status: **2026-09-10 removal plan, with progress refreshed 2026-09-23. No custom
module has been retired and the accepted authority migration remains unimplemented.**
Original reviewed application source: `632f372fe7243c4619c0c0a5a82535f5b02c4107`.
Progress review baseline: `6e2437ded256b761baabef75a9cca98cbcfe079b`.

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
keep full checks. SDK/dashboard validation, chain-registry checks and the
containment/runtime image matrices always run. Skipping a matrix job before
expansion leaves its required per-profile check names absent and blocks merging.

**CI update, 2026-10-19.** Draft pull requests run no CI; marking one ready runs
the selection on the same commit. The classifier now also skips individual heavy
jobs on a pull request when every changed path is of a kind that job is known not
to read: the private-network skill (`skills/run-a-private-zerone-net/`), Go
`_test.go` files outside the packages that job compiles, and Markdown documents.
Markdown still runs Build, Lint, Test and the operations tools, which hash or read
pinned documents. Unknown paths, secret-shaped names, files below a package that
uses `go:embed`, malformed diffs and every push to main still run every job. The
image matrices still expand on every ready pull request so their per-profile check
names stay present; only their build steps skip. A push to main is no longer
cancelled by a later push.

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
round commitments.

The subsequent [review-neutrality release](specs/knowledge-review-neutrality-v1.md)
advances knowledge 8→9. New-policy reviews use equal admitted-account votes and
pay timely reviews including dissent. Agreement-based score feedback, new
wrong-vote/missed-reveal monetary penalties, survival/invitation/challenge
bonuses, automatic probes and new augmentation bounties are retired on those
paths. Preserved obligations and legacy ecological/authority machinery remain;
this is not removal of their modules or proof of independent controllers.

[Claim records](specs/knowledge-claim-records-v1.md) advance knowledge 9→10,
retaining contradiction reasons/evidence and exposing bounded retained history.
[Fund settlement](specs/knowledge-fund-settlement-v1.md) advances 10→11 with
immutable prospective funding terms, explicit refunds and atomic payment retries.
The separate development chain applied that last boundary at height 300,000;
its [published participant path](SHARED-DEVELOPMENT.md) identifies the compatible
runtime. These changes do not upgrade legacy `zerone-1` or activate `zerone-2`.

The [module consolidation sequence](MODULE-CONSOLIDATION-SEQUENCE.md) turns the
dispositions below into staged work with custody and authority prerequisites.
The [development review handoff](DEVELOPMENT-REVIEW-HANDOFF.md) defines the next
observable participant exercise. Neither document records its completion,
adopts a controller policy or authorizes a protocol retirement.

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

The [shared module-purpose catalog](../dashboard/module-guide.json), rendered
in [Understand Zerone](https://zerone.ai/understand/#modules), explains every
registered module's current responsibility and connections. The
[module consolidation sequence](MODULE-CONSOLIDATION-SEQUENCE.md#disposition-of-all-23-registered-custom-modules)
keeps the full per-module future disposition and its required custody/migration
boundary. Keeping current purpose separate from future retirement avoids
presenting planned ownership as implemented behavior.

The largest coupling is visible in
[`KnowledgeKeeper`](../x/knowledge/keeper/keeper.go): bank and staking plus
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
2. **Care for existing settlement before removing economic paths.** Survival
   handoff, neutral review and prospective funding/refund repairs now exist in
   source. Verify the matching network and inventory unresolved historical
   obligations before considering further retirement. Completion means
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
