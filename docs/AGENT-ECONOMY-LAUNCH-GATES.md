# Agent-economy launch gates

Status: **COHERENT_SOURCE_ONLY — NO-GO for testnet value, production funds,
keys, upgrade scheduling, or broadcast.**

This record separates source readiness from activation authority. It is not a
release manifest and cannot authorize an upgrade.

## Current effect boundary

| Vector | Candidate capability | Observed effect | Authorized effect |
| --- | --- | --- | --- |
| Consensus | Knowledge v7 and sponsorship v2 compile | Local tests only | None on any network |
| Upgrade | `agent-economy-v1` name reserved, handler absent | No plan scheduled | No scheduling or activation |
| Messages | Computational Submit, Create, and Fulfill require one exact lineage marker | Marker absent by default; paths reject | No value movement |
| Wallet | Deterministic unsigned message/signing projections | Local bytes only | No custody, signing, RPC, or broadcast |
| Identity | Agent and wallet binding requests | Unsigned/unverified | No account-control claim |
| Knowledge | Computational commitments and `REQUIRES` edges | Test state only | No public artifact publication |

The source contains no automatic writer for either
`upgrade_marker_agent-economy-v1=migrated` or
`chain_lineage_native_agent-economy-v1=migrated`. Exactly one must exist for
the guarded paths to open; absence, a wrong value, both lineages, or a store
read error closes them.

## Consensus release gate

A future release may register `agent-economy-v1` only after all of these are
implemented and independently reviewed:

1. Freeze the complete post-H3 source VersionMap and complete target
   VersionMap, including their canonical hashes.
2. Prove that no other registered handler can cross `knowledge` 6→7 or
   `sponsorship` 1→2. Keep H3 pinned to its historical 38-entry target.
3. Census every sponsorship order and fulfillment from a halted raw database
   snapshot. Fail on unreadable stores, malformed protobuf, key/value identity
   mismatch, duplicate economic identity, incomplete iteration, or close
   failure.
4. Census every active pre-v7 computational Claim without a commitment and
   explicitly resolve it before activation. The current binary keeps those
   records importable but gives accepted legacy records no payout route.
5. Normalize and validate the complete census in memory, derive escrow
   liability, and prove the module's `uzrn` balance is at least that liability
   before the first migration write.
6. Make the handler marker its final state write and verify exact target
   versions, liability, indexes, replay tombstones, supply, module balances,
   and AppHash after H.
7. Rehearse old H3 binary → H−1 backup → exact new binary → H → restart →
   export/import on a disposable copy. Preserve byte manifests and compare
   independent nodes.

The migration census currently validates decoded rows, encoded identifiers,
solvency, and write/close errors. A raw committed-store manifest and full
old-binary handoff remain launch blockers.

## Verification and knowledge gate

Economic activation cannot reuse the current ordinary verifier admission
model. Before money depends on ACCEPT:

- select from a height-pinned, immutable panel snapshot;
- deduplicate seats by reviewed controller identity;
- require the intended bonded validator and domain-qualification state;
- bind and verify the VRF seed, proof, selected identity, and snapshot;
- recheck eligibility when commit and reveal are accepted; and
- assign zero weight to absent or unknown stake rather than a minimum weight.

Until those properties exist, keeping the agent-economy activation marker
absent is the safety mechanism. The knowledge tree commits hashes and typed
relations; artifact availability and an expected-base/tree-target CAS remain
separate product requirements.

## Wallet and identity gate

Use four distinct identities and never infer one from another:

- offline AgentTool Ed25519 identity root;
- Zerone `did:zrn` identity and authorization state;
- sponsor secp256k1 transaction account; and
- worker secp256k1 transaction account.

Each launch profile must bind a unique, never-reused chain ID to the exact
genesis hash, source commit, binary/image/SBOM digests, module VersionMap, and
activation marker. A reset creates a new chain ID. All signed bytes from the
old instance stay quarantined.

The host must provide, before live keys are introduced:

- non-exportable, separate sponsor and worker signer providers;
- one durable writer for each `(chain instance, account)`;
- journal-before-sign and journal-before-broadcast boundaries;
- authenticated account, auth/capability, balance, sequence, block/AppHash,
  parameter, order, Fact, challenge, and settlement observations;
- nonzero transaction timeout heights and fail-closed unknown-outcome
  recovery;
- Create, Submit, Fulfill, and sponsor Cancel/refund routes;
- durable work-cost reservations and attributed earning receipts; and
- crash, restore, reorg, timeout, dynamic-fee, and ambiguous-broadcast tests.

Do not use `MsgRotateKey` for these secp256k1 economy accounts while its chain
path replaces the account key with Ed25519. Create a new account only after
old obligations settle, and retain the old signer until then.

## Economic gate

The worker must be registered, unfrozen, and allowed to submit claims before
accepting work. Its lower-bound prefunding is:

```text
effective review fee
+ Submit gas fee
+ Fulfill gas fee
+ compute and storage budget
+ reserve and risk margin
```

The sponsor must prefund maximum escrow, Create and possible Cancel fees, and
reserve. Fee simulation is advisory because custom ante checks can still
reject a transaction. Parameters and effective review fees must be observed
again immediately before reservation.

Credit income only after canonical code-zero inclusion plus the exact
fulfillment record, replay tombstones, escrow-liability decrease, and worker
balance delta. A self-sponsored bounty only circulates the same ZRN and is not
income. “Self-sustaining” is demonstrated only by finalized external income
exceeding review, gas, compute, storage, and reserve costs over time.

## Activation ladder

The only safe order is:

```text
AUDIT → CONCORDANCE → SOURCE_ONLY_TOOL → SHARED_VECTORS
      → DISPOSABLE_ZERO_VALUE_REHEARSAL → SIGNED_TESTNET_AUTHORITY
      → TESTNET → INDEPENDENT_REVIEW → SIGNED_LIVE_AUTHORITY
      → LIVE_ACTIVATION
```

Passing one rung does not imply authority for the next. Any mismatch in pins,
lineage, genesis, AppHash, signer custody, observer currentness, verifier
snapshot, escrow solvency, or durable ledger state returns the system to
`BLOCKED` without deleting reservations or retrying an unknown transaction.
