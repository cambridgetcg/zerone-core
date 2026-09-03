# Message scheduling and pooling v1

Status: source-complete launch candidate; consensus-visible and default-closed.
No network activation is asserted by this document. Default genesis, the
`zerone-2` ceremony, and the artifact auditor all require
`accept_new_schedules=false`.

## Source interpretation

This design was checked against `FIG_Finalv1.1.pdf` and
`Detailed_Description_Finalv2.2.pdf`. The relevant pool is not Zerone's retired
`x/compute_pool` provider marketplace. It is the patent's Pending Message Pool
(PMP): a volatile, per-node holding area for signed Transaction Messages and
Schedule Messages before consensus. See Detailed Description pp. 7–16,
especially pp. 11–12, claims 1 and 19, and Figures 2–4, 7–8, 10, and 12–14.

The source documents describe this lifecycle:

```text
signed schedule-bearing message
  -> node-local pending message pool and gossip
  -> proposal and consensus inclusion
  -> durable schedule process state
  -> committed condition becomes due
  -> derived occurrence
  -> state transition and durable outcome
```

Zerone maps a patent Transaction Message to the complete signed Cosmos SDK
transaction envelope. That envelope can contain one or more SDK messages,
including a schedule message; a standalone decoded `MsgCreateSchedule` is not
itself placed into another pool. The PMP maps to the CometBFT flood mempool plus
a bounded Cosmos application-side priority/nonce index. There is no third
scheduler gossip pool.

Once a create, update, or cancel transaction commits, its bytes leave the PMP.
Only the `message_schedule` consensus store is authoritative. The transaction
hash identifies the signed ingress envelope; the monotonically allocated
schedule ID identifies the durable process created by it. They are not
interchangeable.

## Consensus model

Version 1 intentionally implements one narrow action: a finite sequence of
prefunded native `uzrn` transfers. It cannot execute arbitrary `Any` messages,
impersonate an account, call a VM, evaluate unbounded logic, or use a
validator's local wall clock.

A schedule contains:

- a canonical, monotonically allocated schedule ID;
- creator and recipient account addresses;
- revision and monotonically increasing processed-occurrence count;
- next committed block height, fixed-delay recurrence interval, and finite
  remaining occurrence count;
- amount per occurrence and a fixed module execution fee per occurrence; and
- separately visible principal and fee liabilities.

The module fee is prefunded module policy, not the patent's dynamically bought
network gas. Ordinary transaction gas and fees still apply to the signed
create, update, and cancel envelopes.

Creation escrows every future principal payment and module execution fee. The
module has no mint or burn permission. Its tracked aggregate liability must
equal its `uzrn` module-account balance after genesis and after every
liability-changing message or occurrence. Existing schedules retain the terms
and per-occurrence fee committed on them; later parameter changes are
prospective. An amendment adopts current parameters and fees for its remaining
work. Lowering a limit never retroactively invalidates already committed state.

The due index is ordered by `(height, schedule_id)`. BeginBlock consumes at
most the governance limit and that limit is itself below an immutable source
ceiling. If a record is processed late, recurrence uses
`executed_height + interval_blocks`; it never performs an unbounded catch-up
burst. Terminal schedules and receipts remain queryable; only their executable
due/active indexes are removed.

Each occurrence uses a cached multistore context. Principal transfer, fee
routing, schedule mutation, liability reduction, and immutable receipt either
commit together or do not commit. If the action cannot transfer, the cached
action is discarded and the complete remaining escrow is atomically refunded;
the schedule becomes terminal `FAILED` with a `FAILED_AND_REFUNDED` receipt.
If even that refund cannot commit, BeginBlock returns an invariant error rather
than recording false success.

The occurrence ID is SHA-256 over a length-delimited, domain-separated encoding
of:

```text
chain_id, schedule_id, revision, execution_sequence, due_height
```

The action digest separately commits recipient, amount, and module fee. These
digests are idempotency and evidence identifiers, not authorization.
Authorization comes from the signed SDK transaction that created or amended
the fully prefunded state.

## Amendment, cancellation, and emergency rules

Update and cancel messages require both `expected_revision` and
`expected_execution_count`. They are compare-and-swap operations, preventing a
client from silently replacing terms after another amendment or occurrence.

Due execution happens in BeginBlock before the block's DeliverTx messages.
Therefore, an amend/cancel transaction included at the due height observes the
post-occurrence count and fails if signed against the pre-occurrence count.
Clients should query immediately before signing and treat a conflict as a new
decision, never an automatic retry.

Closing admission does not strand users. Cancel remains available, and an
existing schedule may be amended while admission is closed only when the
amendment does not increase escrow. Existing schedules ordinarily continue to
execute while admission is closed.

Emergency quarantine is the exception. Due execution pauses while the
canonical emergency keeper reports the chain quarantined. After an affirmative
resume, ordinary CheckTx/DeliverTx admission reopens at the next height, but
the committed release marker is retained and scheduled execution remains
paused for ten complete blocks. Creators can cancel or reduce pending work in
that window. On the following block the bounded due prefix resumes; no backlog
catch-up bypasses the normal per-block cap. The supplied patent documents do
not specify this incident behavior; it is a Zerone safety policy.

## Pending-message pool production profile

`PrepareProposal` uses the bounded application priority/nonce index when
enabled and full Ante verification over CometBFT candidates in the NoOp
fallback. Cross-sender proposal choice is local proposer policy, not consensus
authority. Every validator independently runs `ProcessProposal` over the
proposed order, and that order becomes durable consensus state only on commit.

Proposal construction and validation enforce:

- full Ante authentication for signatures, fees, account sequences, timeouts,
  emergency policy, and Zerone message policy;
- exact CometBFT protobuf byte framing under both on-chain and immutable
  four-MiB ceilings;
- overflow-safe aggregate declared gas under both on-chain and immutable
  33,333,333 ceilings;
- a 22,222 minimum declared gas per ordinary transaction, which implies at
  most 1,500 ordinary transactions plus one optional VEX envelope;
- at most 2,048 inspected proposal candidates, including stale candidates; and
- a signer extractor that supports Cosmos's valid stored-key wire shape where
  `SignerInfo.public_key` is omitted, without making FinalizeBlock behavior
  depend on the node's local pool choice.

Capacity-skipped or dependent future-sequence transactions are preserved
rather than misclassified as invalid. Ante-invalid pool entries are evicted
after the pool iterator releases its lock. An unauthenticated but otherwise
well-formed transaction is rejected.

The `zerone-2` runtime reasserts both pool configurations on every boot:

- CometBFT type `flood`, recheck and broadcast enabled;
- 5,000 transactions, 64 MiB aggregate raw bytes, 256 KiB per transaction,
  and 10,000 cache entries;
- invalid transactions are not retained forever because a future-sequence
  transaction can later become valid;
- gossip fanout is capped at ten persistent and ten non-persistent peers; and
- the application priority/nonce index is capped at 5,000 transactions.

The mempool WAL remains disabled. A signed client must reconcile committed
account sequence after reconnect and rebroadcast intentionally. Neither local
pool is durable or authoritative schedule state.

## Safety-motivated adaptation of the supplied documents

The documents sometimes rely on message arrival order, Internet-synchronized
clocks, and a validator locally emitting a derived Transaction Message. Those
are unsafe consensus inputs: arrival order and wall clocks differ by node, and
local re-gossip can duplicate an occurrence.

Zerone therefore orders due work only from committed height and schedule ID.
At a due height, BeginBlock calls the bank keeper and commits a receipt. It does
not construct a second SDK transaction, obtain a second account signature,
buy another transaction gas allowance, or re-enter either mempool. Calling
this a literal implementation of the patent's generated Transaction Message
would be inaccurate; it is a safety-motivated adaptation preserving the
two-stage committed-schedule lifecycle without a validator-local authorization
path.

Time-of-day triggers, state predicates, compound conditions, indefinite
recurrence, arbitrary SDK messages, contract calls, expiring-token actions,
priority fees, retry/backoff, and a general-purpose derived-transaction lane
are out of scope. Each requires a separately bounded state model, authority
analysis, and consensus rehearsal.

## Growth, activation, and migration boundary

Per-creator active work, per-schedule lifetime work, query pages, proposal
work, and due work are bounded. Total historical schedules and receipts are
append-only and therefore are not globally bounded. Broad activation needs an
explicit state-growth, retention/rent, snapshot, and operator-capacity plan;
the per-block caps alone do not solve long-term growth.

Source publication is not activation authority. Before changing
`accept_new_schedules` to true, require at minimum:

1. independent review of escrow conservation, queue bounds, proposal handling,
   genesis/export, namespace isolation, and module-account permissions;
2. same-height backlog, emergency pause/resume, restart, recheck,
   state-sync/export-import, and multi-validator AppHash rehearsals using the
   exact release binary;
3. operational alerts for both pool saturation, proposal rejection, due
   backlog, failed-and-refunded occurrences, and escrow invariant failure;
4. a historical-state growth/retention decision; and
5. an explicit governance and launch decision naming the parameter set and
   activation height.

The fresh module uses store and account namespace `message_schedule`, protobuf
package `zerone.schedule.v2`, and module consensus version 1. It never mounts,
renames, deletes, or interprets the incompatible retired `schedule` store. The
guarded H3 path adds the fresh store admission-closed only if the old store root
and VersionMap entry are absent and both the retired `schedule` and fresh
`message_schedule` module addresses hold zero coins in every denomination.
Any residue fails closed and requires a dedicated reconciliation. Both
addresses are blocked from ordinary transfers. A fork compiler that injects
the same zero-liability default must prove the identical two-address bank
precondition, and its recovery gate must verify it independently.

The same default-closed state applies at a fresh `zerone-2` genesis. Ceremony
input containing `app_state.schedule` is rejected rather than silently
discarded, and the final artifact auditor independently requires its absence.
This binary must never be pointed at `zerone-1` state outside the exact guarded
upgrade lineage merely because it compiles.

PoT vote-extension proposal injection remains separately release-disabled. Its
unsigned payload is not a substitute for schedule occurrence authorization.
