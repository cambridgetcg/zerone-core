# Accounting and authority consolidation v1

Status: **prospective source repair; existing-network activation not established**.

This change preserves existing claimant records while closing two contradictory
state transitions: legacy custom staking can consume pooled coins without
reducing the corresponding claims, and custom governance can publish `passed`
before the intended action succeeds. It is a preparation step toward the
accepted [single-authority design](../AUTHORITATIVE-STATE.md), not H4, a new
genesis selection, or economic-to-governance decoupling.

The [2026-09-09 existing-ledger census](../reports/authenticated-ledger-census-2026-09-09.md)
records a committed `zerone-1` checkpoint with an empty custom-staking claim
ledger. Its observed legacy lineage does not establish this handler's required
post-H3 predecessor; the census does not change the activation status above.

The [knowledge settlement follow-up](../reports/knowledge-settlement-history-2026-09-09.md)
reconciles the recorded review fees, surviving reward schedules and pooled
knowledge/vesting funding against that checkpoint. It distinguishes committed
claims from historical transfer observations and does not assign pooled funds
or authorize a historical payout repair.

## Custody is backed by individual claims

The native denomination obeys:

```text
bank(zerone_staking, uzrn)
    = sum(active delegation claims, including operator self-delegation)
    + sum(pending unbonding claims)
```

For each validator, the self-delegation aggregate equals its own delegation
claim, the delegated aggregate equals all other delegation claims, and their
sum equals the total. Aggregates cannot create another claim. All self and
ordinary delegation, undelegation, redelegation and stake-update paths must
preserve those relationships. A same-validator redelegation is rejected.

Activation checks the actual bank account, primary claims, secondary indexes,
unbonding identities and amounts. Discrepancies fail without changing old
records. The candidate does not mint a deficit, assign a surplus, haircut a
claim, or infer a claimant from an aggregate. Historic disputes still require
the separately approved resolution described by the authoritative-state design.

The two legacy pooled monetary-slash entry points move no coins after this
boundary. Their reported monetary result is zero, so a caller cannot create a
vindication liability for coins it did not receive. This does not add a new
penalty policy. SDK consensus slashing is separate and unchanged; any future
economic work penalty requires the verifier's explicit task or round escrow.

Before a mature-unbonding payout batch, the complete ledger is checked once.
Malformed or unbacked state leaves every pending claim pending. Each successful
payout batch commits all bank transfers and completed statuses together, and
cannot pay a claim twice. Bank failure retains the entire batch for a later retry. Bounded
state admission and record checks must prevent newly admitted data from making
this audit impossible. Redelegation cooldowns and the accounting-profile flag
are preserved by staking genesis export/import. At the key ceiling, admission
can refuse a partial withdrawal that requires a new record, while full exit
reclaims its delegation/index keys before recording the unbonding claim.

## A governance result describes a committed action

With the governance accounting marker enabled, approved legacy LIPs execute
their complete immediate action in an SDK cache context. Only successful
execution commits target writes, target events and `passed` together. A missing
handler, invalid attachment, execution error or panic discards the target cache
and records `failed` with a closed `execution_error` code. No unimplemented
adapter-registration, research or seat dispatch acquires authority through this
repair. The existing empty parameter router remains empty.

A text LIP remains advisory. A phase LIP can approve a separately recorded
pending activation; approval is not completion of that delayed action. Historic
terminal LIPs are preserved, including old passed-but-inert records, and are
never re-executed by activation. Aggregate LIP stake remains unchanged because
the legacy state has no sufficient per-staker ledger from which to invent a
refund policy.

This is a prospective correctness repair of the legacy path. SDK governance
remains the target sole ordinary executor. Custom voting weights, parallel
domain registries, controller/electorate selection and final legacy retirement
remain open work under H4-F/H4/H5.

## Version and release ownership

`accounting-authority-v1` alone owns the proposed `zerone_staking` 1→2 and
`zerone_gov` 2→3 transition. Its target must not ride a historical broad migration
or the H3 SDK/IBC handler. Missing, mixed or unknown version pairs fail closed.
Future H4 retirement consequently uses custom staking v3 and custom governance
v4; these target reservations do not declare that retirement implemented.

The named handler accepts only the exact complete predecessor and target
module-version maps and verified H3-completed or native H3 lineage. Its canonical
`Plan.Info` binds chain identity, the predecessor version map, both complete
custom stores and their actual bank custody. Committed IAVL/root proofs establish
those bytes; any drift at H refuses activation. The handler audits claim backing,
requires every custom LIP, research-spend, seat-election and phase-transition
process to be terminal, and atomically adds the two markers and an exact
plan-bound migration receipt. It changes no historic proposal or claimant amount.

An old database can start with the candidate only at the exact scheduled H−1/H
handoff, with matching on-chain and local upgrade information. Missing, mixed,
skipped or forged lineage refuses. Fresh disposable native genesis explicitly
selects both repaired profiles, validates bank backing and records native
lineage. Export preserves the distinction between native and migrated lineage;
restart checks committed target versions and exact lineage evidence.
The accepted H1/H2/H3 source and executable identities remain distinct.

Prepare the canonical info using an independently authenticated stopped
predecessor and its committed identity:

```sh
zeroned accounting-plan-info --home /absolute/stopped-predecessor \
  --expected-chain-id CHAIN --expected-height HEIGHT \
  --expected-app-hash LOWERCASE_COMMITTED_APP_HASH > preparation.json
jq -r .plan_info preparation.json
```

The command opens only a private database clone and refuses output if the source
database or genesis changes during verification. The report prepares a proposal;
it neither schedules the plan nor grants authority to freeze activity. This repair
does not implement H4-F's global freeze. If any selected state or custody changes
before H, a fresh commitment and valid new schedule are required. Do not overwrite
claims or disable checks to make a stale plan pass.

The offline census retains `legacy-v1` as its default and keeps the historical
`zerone/custom-staking-census/v1` schema. The explicit `--source-profile
accounting-v2` reader additionally requires the exact new staking marker and
emits `zerone/custom-staking-census/accounting-v2`. It retains the same root
proofs, individual-claim accounting and refusal rules. A v2 report does not
satisfy any legacy production gate. The native local consensus rehearsal uses
this explicit profile; it does not rewrite a copied database to resemble v1.

## Evidence required before an existing-network release

- An authenticated, complete stopped-state census and explicit resolution of
  any inconsistent custody or claimant history.
- Independent review of the exact predecessor and target version maps, keeper
  migrations, lineage checks and coordinated handler.
- An old-binary to new-binary handoff at the chosen height, with failure,
  rollback-boundary, restart and complete-directory recovery evidence.
- Source-matched release artifacts, custody and monitoring evidence, and the
  existing bounded operational gates.

The repaired registration path also needs a newly reviewed transaction budget.
The native dress rehearsal measured 235,561 gas for its one registration, above
the frozen production bootstrap profile's 200,000 allowance. The local regression
fixture uses 500,000 gas and fee; that sample is not a production upper bound.
A target release must simulate its actual state and publish the reviewed budget
in its signed transaction profile. The existing production bootstrap policy
remains bound to its original release; this repair does not silently reprice it.

Source tests and a new native localnet establish neither the old network's
solvency nor permission to restart it with different consensus rules.

## Reproduce the local in-place handoff

Build the reviewed predecessor and candidate separately, then run:

```sh
python3 scripts/accounting-authority-rehearsal.py \
  --before /absolute/predecessor/zeroned --after /absolute/candidate/zeroned
```

The harness creates fresh private test homes, binds only loopback listeners,
and uses signed transactions to fund self and ordinary delegation claims. SDK
governance schedules the exact prepared plan; the real predecessor stops at
H−1 and the candidate applies at H. It checks unchanged existing claims and
custody, consistent self withdrawal, maturity payout, restart, premature-start
refusal and refusal after a valid transaction changes the committed source
state. Reports explicitly label this local development evidence. It cannot
accept an existing node home or target a deployed chain.
