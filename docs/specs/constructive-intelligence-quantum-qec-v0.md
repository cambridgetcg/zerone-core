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

The adjacent `constitutionBinding` pins the exact raw bytes of
`docs/constitution/money-karma-v1.json`. The dependency-free validator hashes
that checked-in file, and the browser parser requires the same schema and
literal digest. The quantum profile cannot silently follow an unreviewed
Money–KARMA revision.

Neither source publication nor any H1, H2, or H3 boundary activates this
extension. Those upgrade sources create no qualification, reward, escrow,
KARMA consumer, performance result, or scientific authority.

The extension is static and network-unobserved. Every release flag is false.
Its reward amount is the canonical integer string `"0"` uzrn, its escrow
receipt is null, and its claimable flag is false. Publishing the file cannot
move funds or qualify an account. A future sponsor case must be a separate,
immutable, fully collateralized object accepted under a separately reviewed
release process.

V0 validates measurement completeness only. It cannot emit a performance
pass, and measurement precision by itself establishes no candidate advantage,
non-inferiority, or equivalence. Until a funded case prospectively binds its
entire performance decision rule, every attempted performance conclusion is
`INCONCLUSIVE_NO_PASS`.

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

Capabilities are versioned curriculum method bundles, not ranks attached to
people. The inherited `qualification-only` wire value is displayed as
“curriculum evidence only (no qualification)”; `sponsor-milestones` is
displayed as “future sponsor-case template (unfunded).” Neither value presently
qualifies a controller or creates an entitlement. Revalidation is triggered
when the method, noise model, corpus, implementation, hardware, or relevant
assumptions change.

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

Those reproducibility bindings are still not a performance decision rule. A
future funded case must additionally and prospectively bind, at minimum, the
baseline-comparator digest and its independent-review receipt, comparison
direction, confidence-bound decision rule, minimum effect or equivalence
margin, estimand/null/direction, latency deadline and quantile, a mandatory
multi-metric conflict rule (hard constraint, lexicographic rule, or Pareto
rule) for logical-error rate versus latency, multiple-comparison policy,
negative-result routing, and matched-resource rule. The list must be frozen
before work or funding starts; post-result threshold selection is
inadmissible. A funded case may choose among conflict rules, but it may not
declare the LER-versus-latency tradeoff unused.

Measurement completeness is evaluated separately for every Cartesian-product
cell of fixture, physical-error value, implementation root, and execution
environment. Results cannot be pooled across cells to fill a missing gate. The
baseline-regression and correlated-noise logical-error targets require, per
cell, respectively at least 100,000 and 1,000,000 cases, at least 100 observed
logical failures, a two-sided 99% binomial interval, and relative half-width no
greater than 30%. Meeting these gates says the declared measurements are
complete enough to inspect; it does not say the candidate performs well.

The matched-resource latency target requires at least 100,000 cases per cell, a
precommitted latency quantile and deadline, and either a two-sided 99% quantile
interval or a 99% one-sided upper confidence bound on the deadline-miss rate.
Zero deadline misses may satisfy measurement completeness; they cannot emit a
performance pass, and latency is not forced to manufacture 100 misses. Raw
trials, exclusions, failures, uncertainty intervals, and negative results
remain required for both modes.

If a frozen compute cap is exhausted before the applicable case, event, and
confidence gates are met, the cell is measurement-incomplete and inconclusive.
For a rare logical-error cell, direct sampling may be replaced only by a
separately reviewed unbiased rare-event estimator. Its method, independent
review receipt, variance method, and coverage validation must be content-
digested before the case opens. It replaces only the direct minimum-case and
100-observed-failure gates; every other coverage and case binding remains.
Its own completeness gate is an estimator-specific two-sided 99% interval
with relative half-width no greater than 30%. This alternative does not apply
to the latency target and cannot emit a performance pass. Regardless of
completeness, v0 emits no performance pass.

For historical reproduction context only, the Version of Record reports
observations of `6.70 ± 1.93e-9`, mean latency `273 ns`, and `99.99% < 1 μs`.
These numbers are explicitly non-normative: they are not
thresholds, margins, deadlines, or acceptance rules. Even an arbitrarily
precise estimate—including one from a bad decoder—cannot become a performance
pass under v0.

A faster result bought with undisclosed parallelism, energy, hardware, tuning
access, or test-set leakage cannot establish performance.

E3 requires at least three effective control clusters, two organization roots,
two independent implementation roots, and two execution environments. Merely
rerunning the originator's container is useful verification but not independent
reimplementation. E5 additionally requires a maintained fixture or upstream
merge by an independent controller. Unexpected security- or safety-relevant
findings enter private triage before publication.

## 5. Reward shape: funded first, earned in stages

There are two orthogonal accounting axes. Each is a complete 100% shape; they
are non-additive and never form 200%. Both are currently zero-value, unfunded,
and inactive. The evidence ladder describes **when** a future prefunded outcome
pool might release:

- E0 commitment: 0%; precedence only.
- E1 inspectable artifact: 0% outcome release; verified costs only, within a
  separate prefunded cap and outside the outcome percentage.
- E2 class-verified: 15%.
- E3 independently reproduced: 20%.
- E4 adversarially survived or fix-tested: 15%.
- E5 independently adopted: 25%.
- E6 maintained: 10%.
- challenge and remediation reserve: 15%.

The attribution-credit axis says **whose constructive work** a future case may
fund:

- originating artifact: 30%;
- independent replication: 25%;
- independent review: 10% attribution credit, not adjudicator pay;
- downstream adoption: 25%;
- falsification and challenge: 7%;
- safety and maintenance: 3%.

Both axes conserve exactly 10,000 basis points. V0 deliberately leaves the
cross-axis allocation and rounding rule undefined; binds no escrow
compartments; implements no single settlement; and supplies no verified-cost
cap, reviewer-budget cap, or unused challenge-reserve route. Any one of those
missing bindings blocks funding. Role-collapse and deterministic-refund rules
are explicit null machine fields in v0 and must be defined before work starts.
The outcome-independence requirement for any future reviewer budget is an
explicit true machine invariant even though the budget cap remains null. The
static extension therefore creates no liability.

`independentReview` records attribution credit for constructive review; it is
not a reviewer-payment rail. Any future adjudicator budget must be separately
prefunded, capped, and outcome-independent. Self-use, address splitting,
reciprocal citation, or correlated agents do not create independence.

The template directly grants no governance authority. That is not a claim
that money can never influence governance: `uzrn` remains bondable under the
current protocol, and bonded stake carries an indirect stake-weight path. A
funded reward mechanism must not activate until economic-to-governance
decoupling is actually enforced.

## 6. KARMA and bounded future governance

The actual KARMA event type is `zerone.karma.edge`; its event register is
`priced-coherence`. It records fallible, challengeable observations of domain
relations. It does not establish human worth, truth, ownership, or controller
merit. Zerone does not mint or create KARMA, and neither an operator nor a
founder may assign it. Recording a relation owns nothing.

KARMA is not a coin, property right, transferable balance, truth score,
universal rank, vote weight, or payout multiplier. Rewards do not buy KARMA;
KARMA does not buy rewards or votes. Raw events never establish candidate
status, and event counts never establish status or increase selection
probability.

This extension fixes its template founder share and founder-reserved seats at
zero. It creates no creator seat, veto, royalty, restoration option, or
privileged reward route. That narrow template boundary does not retire or
decentralize the protocol's disclosed validator, research-spending,
deployment, upgrade, or other control surfaces.

If KARMA later contributes to governance, its first admissible role is only a
domain-scoped, controller-capped, randomized candidate filter for a diverse
deliberation pool. This filter is not runtime-enforced today. Before any future
use, same-controller, self, reciprocal, and correlated-funder edges must be
excluded; controller merges may only reduce candidate units; each controller
may have at most one lottery unit; the candidate set must freeze before
unbiased randomness; and neither operator override nor count-proportional
selection probability is allowed. Activation also requires all of the
following to be independently evidenced first:

1. controller clustering and pair/collusion caps;
2. independently controlled validator, stake, host, and upgrade-binary
   surfaces;
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
