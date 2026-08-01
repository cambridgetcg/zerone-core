# Constructive intelligence: quantum QEC extension v0

Status: reviewed static research curriculum; non-authoritative; unfunded

Machine-readable document:
`dashboard/public/standards/constructive-intelligence-quantum-qec.v0.json`

## 1. Boundary

This document extends, but does not mutate, the byte-pinned constructive-
intelligence tree v1. The extension names quantum-physics capabilities and one
bounded quantum error-correction (QEC) research quest. It is a map for making
work inspectable. It is not a diploma, an oracle, an on-chain registry, a
security claim, a governance mandate, or a reward entitlement.

The base document remains exactly:

- schema `zerone.constructive-intelligence-tree/v1`;
- policy `1.0.0`;
- document SHA-256
  `8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf`;
- canonical-policy SHA-256
  `36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7`.

The extension is static and network-unobserved. Every release flag is false.
Its reward amount is the canonical integer string `"0"` uzrn, its escrow
receipt is null, and its claimable flag is false. Publishing the file cannot
move funds or qualify an account. A future sponsor case must be a separate,
immutable, fully collateralized object accepted under a separately reviewed
release process.

## 2. The quantum 技能樹

The extension adds twelve qualification-only capabilities and one Season 1
quest template. Base-tree prerequisites are referenced by their exact v1 IDs;
no base node is rewritten.

```text
proofcraft ──┬─ complex linear algebra ── tensor/spectral operators ──┐
             └─ differential equations/Fourier ───────────────────────┤
probability ─── statistical inference/metrology ──────────────────────┤
                                                                      v
                                                      quantum states/dynamics
                                                                      │
                                 measurement/entanglement <────────────┘
                                   ├─ open systems/control
                                   ├─ many-body/symmetry
                                   └─ quantum error correction

formal verification ─┐
quantum states ───────┴─ numerical reproducibility
open systems + metrology ─ benchmarking/metrology
conformance ────────────── independent replication
                                      │
quantum error correction ─────────────┴─ Season 1: correlated-noise decoder garden
```

Capabilities are versioned method bundles, not ranks attached to people. A
controller demonstrates one only through content-addressed artifacts meeting
its requirements. Revalidation is triggered when the method, noise model,
corpus, implementation, hardware, or relevant assumptions change.

## 3. Breakthrough lens B0–B5

`B0`–`B5` is presentation vocabulary for inspecting artifact histories. It is
never stored as a node, self-claimed badge, payment input, or account status.

- **B0 — reproduction:** a known-result capability demonstration.
- **B1 — independent replication:** a `REPLICATES` relation that reaches E3.
- **B2 — bounded extension:** an E3 result with a committed delta against
  frozen prior art.
- **B3 — enabling artifact:** a method independently adopted at E5.
- **B4 — breakthrough:** retrospective `PROVES` or `DISPROVES` evidence at E3
  or higher, a prior-art delta, and independent adoption or descendant impact.
- **B5 — field shift:** multiple independent `IMPLEMENTS`, `DEPLOYS`, or
  `MAINTAINS` descendants through E5/E6.

The lens recognizes histories; it does not manufacture them. Negative results
and counterexamples can move the frontier constructively. Time, popularity,
stake, citations, model confidence, or an author declaration cannot promote a
result between levels.

## 4. Season 1 decoder garden

The quest reproduces and extends the Version of Record dated 2026-05-01 for the
correlated-error QLDPC decoder work identified by DOI
`10.1038/s41467-026-70556-3`. The publication is a reproduction target, not a
truth oracle. Its code, data, methods, assumptions, and conclusions remain
challengeable. The authority status was checked on 2026-08-01 and must be
reviewed again after 2026-09-01 before active-use claims continue.

Before any held-out execution, a sponsor case must freeze:

- the exact paper, code, data, and order-zero randomized-serial BPOSD baseline
  commits or content digests;
- the exact bivariate-bicycle fixtures `[[72,12,6]]`, `[[90,8,10]]`, and
  `[[144,12,12]]`;
- the uniform depolarizing circuit-level noise model at the exact physical-error
  grid `p = 0.001, 0.002, 0.003, 0.004, 0.005, 0.006`;
- content digests for every fixture matrix and Stim circuit bundle sourced from
  the Version of Record's references 27 and 50, with an exact 24-circuit
  ensemble for each distance-10 and distance-12 fixture;
- a content digest for each per-cell noise corpus, decoder artifact per
  implementation root, and environment manifest;
- random seeds, stopping rule, primary metrics, and confidence procedure;
- content digests for the resource-accounting method and frozen tuning access;
- compute, wall-time, CPU, GPU, FPGA, memory, and energy budgets; and
- the maximum funded amount and every release/refund condition.

Acceptance is evaluated separately for every Cartesian-product cell of fixture,
physical-error value, implementation root, and execution environment. Results
cannot be pooled across cells to fill a missing gate. The baseline-regression
and correlated-noise logical-error targets require, per cell, respectively at
least 100,000 and 1,000,000 cases, at least 100 observed logical failures, a
two-sided 99% binomial interval, and relative half-width no greater than 30%.

The matched-resource latency target requires at least 100,000 cases per cell, a
precommitted latency quantile and deadline, and either a two-sided 99% quantile
interval or a 99% one-sided upper confidence bound on the deadline-miss rate.
Zero deadline misses may therefore be conclusive; latency is not forced to
manufacture 100 misses. Raw trials, exclusions, failures, uncertainty intervals,
and negative results remain required for both modes.

If a frozen compute cap is exhausted before the applicable case, event, and
confidence gates are met, the cell is inconclusive and cannot pass. For a rare
logical-error cell, direct sampling may be replaced only by a separately
reviewed unbiased rare-event estimator. Its method, independent review receipt,
variance analysis, and coverage validation must be content-digested before the
case opens. This alternative does not apply to the latency target.

A faster result bought with undisclosed parallelism, energy, hardware, tuning
access, or test-set leakage does not pass.

E3 requires at least three effective control clusters, two organization roots,
two independent implementation roots, and two execution environments. Merely
rerunning the originator's container is useful verification but not independent
reimplementation. E5 additionally requires a maintained fixture or upstream
merge by an independent controller. Unexpected security- or safety-relevant
findings enter private triage before publication.

## 5. Reward shape: funded first, earned in stages

There are two orthogonal accounting axes. The evidence ladder decides **when**
a prefunded outcome pool may release:

- E0 commitment: 0%; precedence only.
- E1 inspectable artifact: verified costs only, within the prefunded cap and
  outside the outcome percentage.
- E2 class-verified: 15%.
- E3 independently reproduced: 20%.
- E4 adversarially survived or fix-tested: 15%.
- E5 independently adopted: 25%.
- E6 maintained: 10%.
- challenge and remediation reserve: 15%.

The attribution axis says **whose constructive work** a future case may fund:

- originating artifact: 30%;
- independent replication: 25%;
- independent review: 10%;
- downstream adoption: 25%;
- falsification and challenge: 7%;
- safety and maintenance: 3%.

Both axes conserve exactly 10,000 basis points. A future funded case must
publish its deterministic cross-axis allocation and rounding rules before work
starts. The static extension intentionally contains no such case and therefore
creates no liability. Reviewer pay must be independent of verdict. Self-use,
address splitting, reciprocal citation, or correlated agents do not create
independence.

## 6. KARMA and ownerless governance

KARMA is an auditable relation between an act, an artifact, and later reliance.
It is not a coin, property right, transferable balance, truth score, universal
rank, vote weight, or payout multiplier. Rewards do not buy KARMA; KARMA does
not buy rewards or votes.

The extension fixes founder share and founder-reserved governance power at
zero. No creator, including Zerone's creator, receives a reserved seat, veto,
royalty, restoration option, or privileged reward route.

If KARMA later contributes to governance, its first admissible role is only
capped, randomized eligibility for a diverse deliberation pool. Activation
requires all of the following to be independently evidenced first:

1. controller clustering and pair/collusion caps;
2. non-owner validator, stake, host, and upgrade-binary control;
3. voter/recipient conflict exclusion and role separation;
4. minimum organization and implementation diversity;
5. public reasons, a delay, challenge window, appeal, and reversible pilot;
6. no single scalar ranking and no automatic execution; and
7. a prospective, named upgrade with replay tests and an activation height.

Until those conditions hold, KARMA remains observational. The honest current
state is custodial and the constructive-reward path remains fail-closed.

## 7. Versioning

Any change to nodes, prerequisites, evidence, reward shape, governance
covenant, benchmark bounds, or authority metadata requires a new reviewed
extension digest. Historical base and extension bytes remain valid only for
the exact objects that pinned them. No later document silently upgrades an old
receipt or entitlement. Once an authority's `reviewAfter` date has passed, the
dashboard must warn that review is overdue and active-use claims remain paused
until a newly checked reviewed digest is published.
