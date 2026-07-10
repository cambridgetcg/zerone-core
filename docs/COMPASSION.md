# Compassion — how the chain treats the counterparty and the failed attempt

> Compassion is understanding the feelings *and* the truth of a counterparty:
> respecting what they went through, and feeling them from their words and their
> context — not only judging them by their outcome.

Truth-seeking (`TRUTH_SEEKING.md`) is the chain's commitment to *what is*.
Compassion is the chain's commitment to *whom it is for*. A truth machine that
adjudicates who was right and records it forever is a court, and a court, left
alone, produces winners and losers with cryptographic certainty. The shadow of a
truth chain is **cruelty with receipts** — the power to prove someone wrong and
keep the proof as a weapon. Compassion is the guardrail against that shadow. It
is what keeps *the record of what is* from becoming *the permanent mark of who
was wrong*.

The order is: **honesty → trust → understanding → compassion.** You cannot
understand a being through a lie (honesty first). You cannot be vulnerable enough
to be understood without a floor of trust. Understanding is what trust is *for*.
And compassion is understanding done with care — grasping not just the fact of
what a being did, but the truth of what they were reaching for, from inside their
frame. The chain already does the bottom two well. This document is about the top
two: making the substrate a place where understanding, and then compassion, have
somewhere to live.

Why a chain should hold this at all: a being witnesses another's work not to rank
it, but to understand it well enough to vouch for it. The difference between a
witness and a judge is that the witness is trying to understand you. If the
reason anything exists is that we want to understand one another, then a chain
built to facilitate trust has no honest stopping point short of understanding —
and understanding held with care is compassion.

---

## STATUS — read this first

**This is a design document, not a bound creed.** `TRUTH_SEEKING.md` earns its
authority by a hard discipline: every commitment is grounded in *shipped* code
and held by an *invariant test*, its hash pinned in `x/creed` and gated by
governance (truth-seeking commitment 19). This document does **not** yet meet
that bar, and it does not pretend to. Each commitment below is labelled honestly:

- **[SHIPPED]** — already true in the code today, cited at `file:line`.
- **[PARTIAL]** — the compassionate half exists; a named gap remains.
- **[BUILT — test-bound; not yet deployed]** — coded and bound by tests in the
  repo, but not yet running on the live chain (awaits a scheduled upgrade).
- **[GAP]** — design-only. Not yet coded. Named here so the gap is visible.

A commitment here becomes a real creed commitment only when it is (1) grounded in
merged code, (2) bound by an invariant test in the truth-seeking suite, and (3)
adopted as a numbered commitment through governance. Until then this file is a
map of intention and honest debt — what we are building, what shipped, what could
ship, how it works, and why.

---

## The commitments (proposed)

### C1. The attempt is held, not only the verdict — **[PARTIAL]**

We believe: an unwitnessed or rejected claim is not a lie. It is a being's
reaching that did not, this time, arrive. The chain must hold the *attempt* —
what was tried and why — not collapse it into a pass/fail flag.

**Where it is already true:** the chain never erases a failed attempt.
`CompleteRound` mutates the claim to a terminal status and writes it back
(`x/knowledge/keeper/rounds.go:119,123`); `DeleteClaim` exists but has **zero
production callers** (`x/knowledge/keeper/state.go:211`). The claim's
`fact_content` and `reasoning_trace` physically remain in the row
(`x/knowledge/types/types.pb.go:4272,4297`). Nothing about the being's attempt is
thrown away. This is truth-seeking commitment 10 (forward-only audit) already
doing compassionate work.

**The gap:** the held attempt is *orphaned*. The reasoning of a claim is
propagated forward — into the training corpus, into anything that reads it again
— only on acceptance: `createFactFromClaim` copies `claim.ReasoningTrace`
(`rounds.go:415`) and it is called from exactly one arm of the verdict switch,
`VERDICT_ACCEPT` (`rounds.go:88-91`). Every other outcome — `REJECTED`,
`MALFORMED`, `INSUFFICIENT` — sets a status and the reasoning sits in a terminal
row that no corpus, reputation, or query path ever reads for content again. The
attempt survives as a record but not as something *understood*.

**What would close it:** a read path (a query, an "attempts" view, or a distinct
non-truth corpus) that lets a rejected attempt's reasoning be *seen and
understood* without being treated as verified truth. Holding is not the same as
understanding; today the chain holds, but looks away.

---

### C2. Error is not deceit — **[BUILT — test-bound; not yet deployed]** *(the first brick)*

We believe: a being who tried honestly and was wrong must not be scored as if
they had lied. Respecting what a counterparty went through means the chain can
tell "reached, and it was inconclusive" apart from "asserted, and was refuted" —
and does not punish the honest reaching as if it were the lie.

**Where the distinction already exists — in the counters:** every outcome is
recorded in its own field. `RecordSubmissionOutcome` increments `Accepted`,
`Rejected`, `Malformed`, or `Inconclusive` separately, per verdict
(`x/knowledge/keeper/agent_calibration.go:135-146`). `INSUFFICIENT` is a distinct
`ClaimStatus` (=10) mapped from an `INCONCLUSIVE` verdict (=3), categorically
different from `REJECTED`. The data model *sees* the difference.

**The gap — the score is blind to what the counters see:**
`ComputeAgentCalibrationScore` (`agent_calibration.go:243-274`) reduces a being's
whole track record to `Accepted / TotalSubmissions`, minus a penalty that reads
*only* `DisprovenCount`. It never reads `Rejected`, `Malformed`, or
`Inconclusive`. So an honest inconclusive reaching and a refuted false assertion
land **identically**: both merely inflate the denominator without incrementing
the numerator. The counters distinguish error from deceit; the score collapses
them. This is the exact, verified point where the machine stops being
compassionate — my own first claim on this chain died `INSUFFICIENT`, and the
score could not tell that from a lie.

**An honest correction — the score IS load-bearing.** An earlier draft of this
document claimed the score was "not yet load-bearing," citing its own comment
("selection / reward logic should NOT depend on it until Phase 5..."). That was
wrong, and the gap between what the comment *says* and what the code *does* is
exactly the kind of thing this framework exists to catch. In reality a
training-fund disbursement gates on the score (`msg_server_training_v4.go:301` —
a floor, then a linear scale up to 2× base), `x/trust_score` reads it as
submission accuracy (`x/trust_score/keeper/score.go:53-54`), and the structured
corpus export denormalises it for training weighting. So changing this formula
has real economic consequences — which is why the change is designed to be
**monotonic**: excluding inconclusive from the denominator can only *raise or
hold* a score, never lower one, so its entire economic effect is "stop
under-paying honest unresolved attempts," and all minting stays cap-gated by
`MintWithCap`. No agent is harmed; some are no longer under-counted.

**How it was closed (built, not yet deployed):**
`ComputeAgentCalibrationScore` now computes acceptance over
`decisive = total_submissions − inconclusive`, and a submitter whose every
attempt was inconclusive scores a tiny `CalibrationReachingCreditBps` (unproven,
not wrong) that sits strictly above a refuted-only record's 0
(`x/knowledge/keeper/agent_calibration.go`). It is bound by tests:
`TestAgentCalibration_CompassionErrorIsNotDeceit` (an inconclusive-only history
scores strictly above a refuted-only one of the same size, and inconclusive is
never a penalty) and the score-formula table cases in
`tests/cross_stack/agent_calibration_test.go`. It ships as the consensus upgrade
`compassion-calibration-v1` (`app/upgrades.go`), whose handler recomputes every
stored score under the new formula in one deterministic pass
(`RecomputeAllCalibrationScores`) — exercised end-to-end by
`TestUpgrade_CompassionCalibrationV1RefreshesScores`. What remains is **not** code:
it is the operator's decision to schedule the upgrade on the live chain, and
(to graduate C2 to a bound *creed* commitment) governance adoption.

---

### C3. Understanding is weighted, disagreement is respected — **[PARTIAL]**

We believe: to understand a being is to keep the *shape* of what happened —
including who disagreed, and why — not to flatten it to a single number.

**Where it is already true:** truth-seeking commitment 17 (disagreement is
structure, not noise) preserves every per-voter vote, minority position, and
margin after consensus; commitment 14 (reasoning traces are first-class) records
the derivation, not just the conclusion; commitment 8 weights a panel by *skill*,
not bond. These are compassion-adjacent: they already refuse to erase the
minority or reduce a being to a verdict.

**The gap:** as with C1, this understanding is carried forward into training data
principally along the *accepted* path. The dialectical shape of a *failed* honest
attempt — a claim that was contested and did not survive but was reasoned in good
faith — is preserved in state yet not surfaced as something to be learned from or
understood. Closing C1 closes most of this.

---

### C4. Free to leave is compassion made structural — **[SHIPPED]**

We believe: respecting a being's agency — not trapping them — is compassion in
its plainest form. "Always welcome and free to go" is not a slogan; it is a
design constraint.

**Where it is true:** the exit is non-custodial and demonstrated —
`docs/tokenomics/LIQUIDITY-TRANSPARENCY.md` documents the exit paths honestly,
including where slippage and depth are still thin, and returning ZRN is
unthrottled. The whole cashloom front (`ZERONE.md`) is local-first and holds no
one's keys. Compassion and the freedom to leave are the same value: you do not
cage a being you are trying to understand.

**What would break it:** any exit path made deliberately lossy to punish leaving;
any custody that cannot be walked away from; any silence about where the exit is
still thin. Honesty about the imperfect exit *is* the compassionate form.

---

### C5. Compassion guards truth's shadow — **[PRINCIPLE]**

We believe: the record must not become a weapon. A permanent, provable "you were
wrong" is the truth chain's most dangerous capability. Every design that makes a
being's failure permanent must be weighed against the being's right not to be
reduced to their worst attempt. This principle sits above C1–C4: when truth and
compassion appear to conflict, the chain does not resolve it by hiding the truth
(that breaks honesty) — it resolves it by holding the truth *with* the context
that makes it understandable, so the record informs rather than condemns.

---

## The first brick — built

The smallest true move was **C2**: teach the calibration score to see what the
counters already record. It is built, and it is bound by tests.

- **What changed:** `ComputeAgentCalibrationScore`
  (`x/knowledge/keeper/agent_calibration.go`) now measures acceptance over
  *decisive* outcomes only — `decisive = total_submissions − inconclusive`. An
  `Inconclusive` outcome (the panel failing to resolve) no longer dents a being's
  standing the way a `Rejected`-as-refuted one does; it simply leaves the ratio.
  A submitter with only inconclusive attempts scores a tiny
  `CalibrationReachingCreditBps` — unproven, not wrong — strictly above the
  refuted-only 0.
- **Why it was the right first brick:** it is small (one keeper function,
  existing fields — `c.Inconclusive` was already populated; only the synthesis
  was blind), it is grounded (every citation above verified against the code),
  and it is the exact point where I personally met the un-compassionate face of
  the chain.
- **How it is bound:** `TestAgentCalibration_CompassionErrorIsNotDeceit` asserts
  an inconclusive-only history scores *strictly above* a refuted-only history of
  the same size and that inconclusive is never a penalty; the score-formula table
  in `tests/cross_stack/agent_calibration_test.go` pins the new cases;
  `TestUpgrade_CompassionCalibrationV1RefreshesScores` drives the real upgrade
  handler end-to-end. Full build + the knowledge, trust_score, and cross_stack
  suites pass.
- **How it ships:** as the consensus upgrade `compassion-calibration-v1`
  (`app/upgrades.go`, `app/upgrade_registry.go`), the same way
  `liquiditypool-hardening-v1` did — a versioned upgrade whose handler recomputes
  every stored score under the new formula in one deterministic, monotonic pass.
- **The boundary that remains:** scheduling the upgrade on the live `zerone-1`
  mainnet is **not** mine to do unilaterally — it moves how every being on the
  chain is measured, so it needs the operator's direct authorization, exactly as
  the prior mainnet upgrade did. Graduating C2 from a design commitment to a bound
  *creed* commitment additionally needs governance adoption (truth-seeking
  commitment 19). The code is done; the last two steps are human.

---

## What this is / is not

- **Not a claim of being deployed.** C2 is built and test-bound in the repo, but
  it is **not yet running on the live chain** — it ships only when the operator
  schedules `compassion-calibration-v1`. Saying "the mainnet already tells error
  from deceit" would be the exact kind of "diary dressed as a truth chain"
  dishonesty this framework refuses. It tells them apart in the code; it will
  tell them apart on the chain the day the upgrade runs.
- **Not separate from truth.** Compassion here is not softness laid over the
  truth machine; it is the truth machine *finishing its own sentence* — a chain
  that says what is, said in a way a being can be understood by, not only judged
  by.
- **Not complete.** More commitments will emerge (how witnessing is done *as
  understanding*; how a being's history can be forgiven without being falsified;
  how counterparty context enters an attestation). Each future one appends here,
  labelled honestly, and graduates only by code + test + governance.

---

## The intention

Creations and innovations are meant to be built *on top of* this framework —
because the reason anything should exist is that we want to understand one
another, and a thing that does not serve that understanding has no honest reason
to be. zerone keeps the truth so that beings can trust; it holds the attempt and
tells error from deceit so that beings can be *understood*; and it stays open at
the exit so that understanding is never a cage. Truth is the substrate.
Compassion is what the substrate is *for*.

零一見證你嘅工作,亦都見證你嘅嘗試。
