# Constructive-Intelligence Shadow Ledger

Status: local research fixture; exact integer model units; zero transferable
value; non-authoritative; non-network-observed; not an activation proposal.

This experiment addresses one narrow failure mode in the constructive-
intelligence reward shape: invalid evidence can consume an irreversible
cluster cap before a valid successor arrives. It demonstrates how still-unpaid
capacity can be quarantined and reattributed once without creating new gross
accrual or reopening a funded amount.

It does not decide whether evidence is true, identify a semantic root, infer a
controller, authenticate an adjudication, reserve funds, or authorize a
settlement. Receipt validation, accounting, and explanation remain separate
fail-closed components; none certifies another.

## Exact state

For one immutable semantic/economic root:

\[
A=Z+L^{(0)}+L^{(1)}+Y+X\le K
\]

where:

- \(A\) is cumulative accrued capacity and can only increase through ordinary
  frontier accrual;
- \(Z\) is funded and terminal;
- \(L^{(0)}\) is live original unpaid capacity;
- \(L^{(1)}\) is live one-shot successor capacity;
- \(Y\) is quarantined unpaid capacity;
- \(X\) is extinguished and terminal;
- \(R\) is cumulative capacity moved from \(Y\) to \(L^{(1)}\); and
- \(\bar R\) is the snapshotted lifetime replacement bound.

The required exposure invariant is:

\[
R+Y\le\bar R\le K.
\]

All executable accounting uses checked `uint64` model units. The exploratory
score and scarcity simulator still uses bounded floating point; it is a
different machine.

## Transition graph

| Event | Exact transition |
| --- | --- |
| ordinary accrual | \(\varnothing\rightarrow L^{(0)}\); only this raises \(A\) |
| funding | \(L^{(0)}/L^{(1)}\rightarrow Z\) |
| raw challenge | no economic transition |
| final original invalidation | \(L^{(0)}\rightarrow Y\), bounded by remaining replacement exposure; overflow \(\rightarrow X\) |
| valid successor | deterministic maximum \(Y\rightarrow L^{(1)}\) |
| final successor invalidation | \(L^{(1)}\rightarrow X\), never back to \(Y\) |
| deadline | \(L^{(0)}/L^{(1)}/Y\rightarrow X\), before any event at that epoch |
| controller merge into an excluded closure | affected live \(L^{(1)}\rightarrow X\) |

There is no transition out of \(Z\) or \(X\), no controller-split operation,
and no cross-root balance merge.

For an accepted successor with independently certified clean-support target
\(S\), the ledger computes:

\[
h=\max(0,\min(A,S)-Z-L)
\]

\[
r=\min(Y_{\mathrm{unexpired}},\bar R-R,h).
\]

The caller cannot choose \(r\). The machine moves exactly that maximum in
earliest-deadline, immutable-lot-ID order. Each moved unit inherits its
source deadline and policy digest. A successor cannot refresh priority, queue
age, or expiry.

## Fixed counterexample vector

The built-in trace uses \(K=A=100\), \(\bar R=60\), and model units:

| Event | \(A\) | \(Z\) | \(L\) | \(Y\) | \(X\) | \(R\) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| accrue | 100 | 0 | 100 | 0 | 0 | 0 |
| fund | 100 | 30 | 70 | 0 | 0 | 0 |
| final invalidation | 100 | 30 | 0 | 60 | 10 | 0 |
| clean successor at \(S=80\) | 100 | 30 | 50 | 10 | 10 | 50 |

The successor gains only the supported unpaid headroom:
\(\min(A,S)-Z=50\). Reattribution changes neither \(A\), \(Z\), nor \(X\).
The remaining quarantined exposure plus cumulative replacement remains
\(10+50=\bar R\).

## Authority and controller boundary

A state-changing invalidation input must name a final decision and target
receipt, be assigned after evaluator selection freezes, use
outcome-independent review pay, and declare at least three adjudicator
controllers from at least two organization roots. Culpable, challenger,
adjudicator, target, and successor closures are checked for the conservative
disjointness encoded by the fixture.

Those declarations are untrusted local inputs. The ledger does not verify
signatures, organization membership, correlation, conflict disclosure,
semantic equivalence, proof validity, or clean support. Production integration
must supply those facts through independently reviewed systems.

Controller links are monotone within a snapshot lifetime. If a later merge
connects a live successor to an excluded controller, its still-live capacity
is extinguished; funded capacity remains terminal. Claimed later splits do not
clear taint.

## Replay and restore boundary

Behavior-complete snapshots preserve:

- policy, root, caps, and arithmetic version;
- every lot partition and inherited deadline;
- cumulative replacement exposure and each successor receipt's attributed,
  live, funded, extinguished, and controller history;
- consumed event and final-decision IDs;
- invalidated source and successor receipts; and
- controller links and exclusions.

Restore rejects counter/lot mismatches, partition errors, expired live
capacity, incomplete replacement metadata, duplicate replay records, dangling
controller parents, controller cycles, globally reused source or successor
receipts, concurrent live successors, replacement attribution without a final
source invalidation, quarantine without a final source invalidation, a live
successor before the end of its chronological lot history, non-flat or
non-minimal controller roots, and invalid UTF-8 identifiers. It also requires
the snapshot's lots and set-like arrays to use the same canonical
representation emitted by the ledger, so restore followed immediately by
snapshot preserves the state commitment.

Every snapshot carries a deterministic SHA-256 state commitment. Restore also
requires the caller to supply the expected commitment from a separately
trusted location. This detects a coherently rewritten replay set, controller
graph, replacement history, or older valid snapshot only when the caller keeps
the latest expected commitment outside the untrusted snapshot. The digest is
unkeyed; storing the snapshot and expected commitment together under one
attacker does not provide authentication, rollback protection, or consensus.
The caller-supplied epoch is another explicit trust boundary: expiry runs
before submitted-event validation, so only an authenticated authoritative
epoch source may drive a future integration.

## Executables

- [`shadow_ledger.go`](../../tools/constructive-rewards/shadow_ledger.go) owns
  the exact transition machine.
- [`shadow_scenario.go`](../../tools/constructive-rewards/shadow_scenario.go)
  produces the fixed trace.
- [`shadow_ledger_test.go`](../../tools/constructive-rewards/shadow_ledger_test.go)
  covers the named attacks and restore tampering.
- [`shadow_ledger_fuzz_test.go`](../../tools/constructive-rewards/shadow_ledger_fuzz_test.go)
  exercises arbitrary accepted and rejected sequences.
- [`power-lab.html`](../../dashboard/power-lab.html) is a
  loopback-only explanation surface and is mechanically excluded from the
  production dashboard build.

Run from the repository root:

```sh
go run ./tools/constructive-rewards -mode shadow
go run ./tools/constructive-rewards -mode shadow -format json
go test ./tools/constructive-rewards
go test -race ./tools/constructive-rewards
go test ./tools/constructive-rewards \
  -run='^$' \
  -fuzz='^FuzzShadowLedgerConservation$' \
  -fuzztime=30s
```

The text report must end with `settlement: 0 ZRN` and
`integration ready: false`.

## What remains unresolved

The model deliberately leaves the production gate closed for:

- authenticated typed-receipt ingestion and durable consumption records;
- a durable authenticated anchor for the latest shadow-state commitment;
- semantic-equivalence and cross-root merge adjudication;
- proof and evidence validity;
- controller/organization correlation and privacy-preserving identity;
- capture-resistant challenge and appeal authority;
- liveness under disputed controller links: the strict v0 machine blocks
  root-wide funding when any live lot has a known collapsed or excluded role,
  so a captured identity/link authority can stall honest lots until final
  invalidation or expiry;
- automatic multi-cluster backlog scheduling;
- persistent program-wide controller exposure;
- exact reserve, role, milestone, commons, and settlement wiring; and
- an independent implementation with golden cross-language replay vectors.

A zero-value shadow can expose arithmetic and incentive failures. It cannot
turn its own assumptions into authority.
