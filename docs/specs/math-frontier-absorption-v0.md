# Math Frontier Absorption v0 — how zerone integrates mathematical breakthroughs

Status: experimental, client-side only, consensus-untouched. 2026-08-03.

This document is the intake discipline by which real mathematical discoveries
enter zerone's knowledge corpus today, and by which future discoveries keep
entering without new chain code. It composes three things that already exist:
the `x/knowledge` claim → panel-verification → fact pipeline (live on
zerone-1/zerone-testnet-1), the Math Frontier v0 zero-value skill tree
(`docs/specs/constructive-intelligence-math-frontier-v0.md`, merged and
deployed on zerone.ai), and the seasons absorption pattern used by the
life-sciences/quantum/epigenetics corpora. It adds one register, one tool,
and a set of provenance rules. It changes NO consensus behaviour and mints
nothing.

## 1. The register

`docs/research/math-breakthroughs-2026-08.v0.json`
(schema `zerone.math-breakthrough-register/v0`) is the dated, selection-bounded
source register: 40 entries covering 2023-01 through 2026-08 — proven
breakthroughs, machine-verified milestones, refutations, and open conjectures —
each with a chain-ready statement (20–1000 chars, the live
Min/MaxClaimTextLength), pinned sources, an honest status, and a chain intent.
`selectionBoundary` declares the register representative, not exhaustive;
absence is not a judgment. The register is append-and-amend by pull request:
it never records chain state (the intake tool's ledger does that), and it
never deletes an entry — corrections are new status values with new sources,
per rule P12 below.

## 2. The status ladder and what it admits

```
formalized > published_peer_reviewed > preprint_widely_accepted
          > preprint_under_review > announced_unverified
refuted        — the refutation is the admissible fact
open_conjecture — queued for the conjecture engine
```

Chain admission (enforced by `tools/frontier-intake` and recorded in the
register's `chainAdmission` block):

- **assert** — statuses `formalized`, `published_peer_reviewed`,
  `preprint_widely_accepted`, and `refuted` may be submitted as ordinary
  ASSERTION claims (`tx knowledge submit-claim`, domain `mathematics`,
  category `theorem` / `refutation` / `formalization`). A refuted entry
  admits ONLY the refutation as the true fact, never the refuted statement.
  Claim text must carry its own evidence level when below peer review (the
  Jacobian-refutation entry is the model: the statement itself says how and
  when it was verified).
- **conjecture** — `open_conjecture` entries queue for `MsgPostConjecture`
  (PROVISIONAL, confidence 0, panel question "well-posed and falsifiable").
  The conjecture engine is merged on trunk but NOT on the live binaries; this
  lane opens at the next coordinated upgrade and nothing here may work around
  that.
- **withhold** — `preprint_under_review` and `announced_unverified` entries
  stay in the register and off the chain until their evidence matures. A
  truth chain earns trust by what it refuses to assert.

## 3. Provenance rules (P1–P12)

Distilled from the 2026-08 research pass; each is load-bearing for how
register sources must be written.

- **P1 arXiv**: pin `arXiv:ID vN` — bare IDs and arXiv DOIs float to the
  latest version; record retrieval date; store your own byte hash if
  bit-integrity matters.
- **P2 peer review is dated, revisable evidence**: refereeing has decade-scale
  documented failures (Kempe 1879→1890; Kapranov–Voevodsky 1991→1998). Status
  entries carry dates, never finality.
- **P3 publication ≠ acceptance**: model journal publication and
  expert-community acceptance separately (the Mochizuki/abc precedent — PRIMS
  2021 publication, community non-acceptance).
- **P4 Lean citation unit**: repo URL + commit SHA + lean-toolchain +
  lake-manifest.json + fully-qualified declaration name + `#print axioms`
  output. mathlib is rolling; names alone are not identifiers.
- **P5 chain-grade "formalized"**: requires independent kernel replay
  (lean4checker / Lean4Lean) and an axiom allowlist
  {propext, Classical.choice, Quot.sound}; record native_decide use.
- **P6 coverage lists** (Wiedijk 100: 99/100, FLT sole holdout, Lean 84/100 at
  2026-08-03) are dated benchmarks, never completeness metrics.
- **P7 Zbl/MR numbers** identify review records, not content — cross-check
  handles only.
- **P8 object databases**: cite (database, scheme, label, access date) —
  schemes collide (LMFDB 11.a1 = Cremona 11a2) and content evolves.
- **P9 formal-library archival unit**: Isabelle AFP freezes per release (DOI);
  mathlib has no frozen releases, so pin commits.
- **P10 do not require formalization at the frontier**: no at-scale
  formal-statement registry exists (Formal Abstracts never shipped);
  formalization lags the frontier by years. Use the ladder + statement hash.
- **P11 aggregators** (Quanta, AMS Notices) may evidence significance and
  reception only; `statement` and `status` must terminate in primary
  artifacts.
- **P12 corrections are append-only events**: arXiv withdrawal is a new
  version (old versions remain citable); journal corrections are separate
  DOIs. The register amends status forward; the chain corrects via
  CONTRADICTS relations and challenges, never edits.

## 4. The intake protocol (what the tool does)

`tools/frontier-intake/` (Bun TS, no chain code, shells the `zeroned` CLI
exactly like `tools/agenttool-relay` shells it):

1. **validate** — register schema, statement lengths, admission rules,
   duplicate subjects.
2. **plan** — for each admissible entry: free `q knowledge check-novelty`
   preview + current effective fee; nothing is broadcast.
3. **submit** — `tx knowledge submit-claim <statement> mathematics <category>
   <fee>` with `--claim-type assertion --subject … --predicate … --tags …`;
   claim_id/round_id parsed from tx events into the ledger
   (`~/.zerone-agent/frontier-intake/<network>.state.json` — runtime state
   lives outside the repo, read-merge-written under an exclusive lock).
   Batch submissions interleave panel passes during the pacing-scaled
   cooldown waits, so every earlier round's commits and reveals land inside
   their own phase windows while later entries are still queueing. Every
   broadcasting command requires an explicit `--live-ack=<chain-id>`
   acknowledgment (the repo's broadcast-guard discipline, adapted for a tool
   whose purpose is live intake; tools/agenttool-relay refuses live chains
   outright, which is the right posture for its job and impossible for this
   one).
4. **panel** — the only round-resolution machinery live today is the open
   self-selection panel: N≥4 distinct keys, each ≥100.5 ZRN spendable at
   commit (the gate reads 100 ZRN AFTER ante fees), commit
   `sha256("ZRN.commit.v1:<round>:<vote>:0:<salt>")` inside the commit
   window, reveal `(vote, salt)` inside the reveal window. Domain claims need
   4 reveals and ≥77% stake-weighted agreement — with equal-weight external
   keys that means 4-of-4 or 4-of-5. The panel attests what the register's
   pinned sources evidence; a panelist who cannot check a source votes
   reject. The live binaries predate early aggregation, so a round takes the
   full commit+reveal+aggregation schedule (~450 blocks); rounds for
   different claims run in parallel.
5. **confirm** — after aggregation: claim status classified against the live
   numeric enum (ACCEPTED = 6), confidence recorded in the ledger, and the
   domain's epistemic temperature and conformity alerts printed so the
   operator sees cooling before it caps confidence; `q knowledge bundle-tok`
   shows the mathematics subgraph growing.

Panel keys must not have day jobs: the relay daemon's key drifted under the
verifier balance gate mid-drill and was replaced by a dedicated key.
Custodial honesty: on today's single-operator network the submitter and panel
keys are all kingdom-custodial — the same bootstrap shape as the chain's
first fact and the 47-fact doctrine corpus. The panel's value here is
process, not independence: statements enter only from the validated register,
whose sources were verified outside the chain. Karma's `self=true` flagging
and the future verifier economy are the decentralization path, not this tool.

## 5. Anti-slop pacing

The chain's own walls apply and the tool must stay inside them: 0.1–0.2 ZRN
non-refundable fee per claim, canonical+content dedup, 50-block per-submitter
cooldown (pacing-scaled), novelty penalties feeding fitness, domain carrying
capacity (mathematics: sparse today, birth-energy bonus active), and
conformity cooling — sustained unanimous panels cool the domain and cap
confidence for new facts. Discipline: ≤10 claims per day, batch submissions
spaced ≥cooldown, watch `q knowledge conformity-alerts` and
`epistemic-temperature mathematics`, and prefer 5-key panels once a fifth
funded key exists so honest dissent is survivable. Facts are mortal
(metabolism): the corpus stays alive through queries, citations, and — for
load-bearing entries — patronage, not through resubmission.

## 6. Absorbing the future (the standing process)

A new mathematical discovery enters like this, indefinitely:

1. Anyone adds a register entry by PR: pinned primary sources (P1–P11),
   honest status, chain intent, statement written for the chain.
2. `frontier-intake validate` gates the change: schema, lengths, admission,
   sub-peer-review evidence disclosure in claim texts, refutation wording,
   and aggregator-only source rejection — run locally by the operator and in
   CI by the `frontier-intake` job on every push.
3. On merge, an operator runs `plan` → `submit` → `panel` for admissible
   entries; conjecture intents queue for the engine.
4. Corrections flow per P12: register status amended forward; on-chain, a
   contradicting claim or challenge — never an edit.
5. The register re-snapshots (v1, v2 …) when the window advances; old
   snapshots remain, hash-pinned.

## 7. Needs-Yu roadmap (nothing below ships without governance/coordination)

- **Conjecture upgrade**: the 9 `open_conjecture` entries (Riemann, twin
  primes, Collatz, P vs NP, Navier–Stokes, Hodge, BSD, Antihydra, abc-status)
  queue for `MsgPostConjecture` + `open-questions`, which reach the live
  binaries only via the next coordinated upgrade (consolidation-safety-v1 is
  staged, unproposed).
- **Math Frontier v1**: promoting the zero-value skill tree to a real quest
  needs everything its own spec lists — real problem packets, a general
  packet validator, funded sponsor escrow, E2 receipts — and re-pinning the
  reviewed dashboard digest.
- **Demand loop**: `AuthorizedDemandReporters` is empty and the treasury
  (0.088 ZRN) cannot fund a single 10 ZRN bounty; a params change plus
  treasury accumulation would let demand for mathematical knowledge mint
  bounties that pay intake.
- **Verifier independence**: external panelists with their own stakes are the
  real endgame; until then every fact this pipeline lands is honestly
  custodial-bootstrap, and the chain's own events say so.
