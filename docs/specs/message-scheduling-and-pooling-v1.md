# Message scheduling v1 — scheduler-only integration candidate

Status: consensus-visible, default-closed integration candidate, selectively
ported from `c5503a1b1569ab0cbf890709f36bc2c2c6eca7b3` onto main's
`89553a0132dafa1ba9670f53b8a3195b33a6e720` baseline. The historical filename is
retained; the optional pooling optimization is deferred and is not imported.
This document is not evidence of a completed integration/restart rehearsal,
clean release, deployment, or live activation. Uncommitted builds are NON-FINAL.
Default genesis, the `zerone-2` ceremony, and the artifact auditor require
`accept_new_schedules=false`, empty schedules and receipts, and zero escrow.

## Source interpretation

The retained source specification cites `FIG_Finalv1.1.pdf` and
`Detailed_Description_Finalv2.2.pdf`; this integration carries those references,
not private document contents or a new independent patent review. The relevant
pool is not Zerone's retired
`x/compute_pool` provider marketplace. It is the patent's Pending Message Pool
(PMP): a volatile, per-node holding area for signed Transaction Messages and
Schedule Messages before consensus. See Detailed Description pp. 7–16,
especially pp. 11–12, claims 1 and 19, and Figures 2–4, 7–8, 10, and 12–14.

The retained interpretation describes this lifecycle:

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
itself placed into another pool. In this integration the PMP maps to the
existing CometBFT mempool. Main retains the SDK NoOp application mempool; no
application-side priority/nonce index or separate scheduler gossip pool is added.

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

`execution_count` counts processed occurrences, including a terminal failed and
refunded occurrence; it is not a successful-payment count. The BeginBlock
processed count has the same distinction. Receipt `outcome` is authoritative
for whether principal and fee were paid or remaining escrow was refunded.
Receipt amount/fee fields describe the attempted action, not proof of payment
when the outcome is `FAILED_AND_REFUNDED`. Clients must inspect outcomes and
failure codes rather than infer success from a count or an occurrence ID.

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

Due execution happens in BeginBlock before the block's DeliverTx messages,
but only for the bounded selected due prefix. An amend/cancel transaction for
selected work observes the post-occurrence state and cannot undo that payment;
a pre-occurrence compare-and-swap value fails (or the schedule is already
terminal). A due schedule outside that prefix has not been processed and may
still be cancelled in the same block with unchanged compare-and-swap values.
The same distinction applies while due processing is paused. Clients should
query immediately before signing and treat a conflict as a new decision, never
an automatic retry; a query alone cannot reserve a place ahead of BeginBlock.

Closing scheduler admission rejects creation and escrow-increasing amendments
at message execution. It does not by itself prohibit cancellation or a
non-increasing amendment, and already committed schedules ordinarily continue
to execute. All signed instructions still obey normal Ante authentication,
fees, sequence, capabilities, account-freeze, and emergency checks. Closed
scheduler admission is not a bypass around those checks.

In particular, freezing a creator blocks future signed instructions, including
cancellation and amendment. It does **not** implicitly revoke or pause that
creator's previously authorized, fully escrowed transfers. Execution uses the
module's escrow authority, not a new signature or the creator's current account
balance. There is no execution-time creator-freeze policy in this port. This
asymmetry must be consciously accepted before activation rather than hidden by
claiming that every creator can always cancel.

Likewise, bank send-enabled flags are not automatically a module-payment
switch. Module-to-account payments use the bank keeper's blocked-recipient
checks and escrow accounting; disabling ordinary sends alone is not a promise
to stop scheduled execution. A failed payment follows the atomic failure and
refund rules above.

Emergency quarantine is the exception. Due execution pauses while the
canonical emergency keeper reports the chain quarantined. After an affirmative
resume committed at height H, ordinary CheckTx/DeliverTx admission reopens at
H+1, but the committed release marker is retained and scheduled execution
remains paused through H+10 inclusive. Creators who satisfy normal Ante checks
can cancel or reduce pending work in that window; frozen accounts cannot.
At H+11 the bounded due prefix resumes; no backlog catch-up bypasses the normal
per-block cap. This cancellation grace is a Zerone safety policy, not a claimed
requirement of the supplied patent documents.

## Existing main proposal policy — pooling deferred

This port leaves `app/abci.go` and production mempool configuration unchanged.
Main uses the SDK NoOp application mempool. In `PrepareProposal`, CometBFT
supplies CheckTx-admitted candidates; the application decodes them and uses the
SDK default selector with canonical protobuf byte sizing under the request's
`MaxTxBytes` and declared gas under current consensus block parameters.
Malformed entries and individually impossible gas declarations are skipped.
This construction path does not add a second full Ante pass or a priority/nonce
index.

Every validator independently calls `ProcessProposalVerifyTx` for ordinary
transactions in proposed order. The full Ante path checks signatures, fees,
sequences, timeouts, emergency quarantine, account freezes, and Zerone
capabilities in disposable proposal state, carrying Ante effects between
transactions in that order. Overflow-safe aggregate declared gas is checked
against the current consensus `MaxGas` when it is nonnegative; absent or
negative `MaxGas` means that gas bound is not applied. PoT vote-extension
injection remains release-disabled.

Do not attribute the retained pooling branch's additional immutable four-MiB
proposal ceiling, 2,048-candidate inspection cap, 1,500-transaction bound,
priority/nonce eviction behavior, or runtime pool settings to this integration.
Those mechanisms were not ported. In particular, main's `BlockGasLimit`
constant is not a separate immutable cap enforced by these proposal handlers
against later consensus-parameter changes.

Local candidate arrival/order and proposal selection may differ across nodes;
the agreed finalized block order, not local CheckTx history or mempool contents,
determines committed schedule state. Losing pending transactions does not lose
already committed schedules. A signed client must reconcile committed account
sequence and schedule revision/count after reconnect, then rebroadcast only
intentionally. The existing local mempool is neither durable schedule storage
nor occurrence authorization.

## Safety-motivated adaptation of the supplied documents

Arrival order, gossip, and validator-emitted derived messages are not inherently
unsafe. They can be local admission/proposal choices, or inputs validated and
ordered by consensus. The unsafe boundary is allowing **unagreed local state
effects**: for example, executing payments from each node's own arrival order
or wall clock without a common committed trigger, or applying duplicates
without a consensus-enforced occurrence identity. An agreed ordering rule and
replicated deduplication can make a separate derived-message lane safe; this
narrow port simply does not implement that lane.

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

Per-creator active work, per-schedule lifetime work, query pages, and due work
are bounded. There is no global active-schedule cap across arbitrary creators,
and total historical schedules and receipts are append-only and unbounded.
These are current activation blockers, not solved by per-creator or per-block
caps. Broad activation needs an explicit global growth/capacity policy and a
state-growth, retention/rent, snapshot, and operator-capacity plan. No historical
pruning, rent policy, or new global cap is introduced by this closed port.

Source publication is not activation authority. Before changing
`accept_new_schedules` to true, require at minimum:

1. independent review of escrow conservation, queue bounds, proposal handling,
   genesis/export, namespace isolation, and module-account permissions;
2. same-height backlog, emergency pause/resume, restart, recheck,
   state-sync/export-import, and multi-validator AppHash rehearsals using the
   exact release binary;
3. operational alerts for existing mempool saturation, proposal rejection, due
   backlog, failed-and-refunded occurrences, and escrow invariant failure,
   with a fail-stop recovery procedure;
4. a global active/historical-state growth and capacity decision, plus conscious
   acceptance of the freeze/revocation semantics above; and
5. an explicit deployment/migration lineage, clean-release rehearsal, and
   governance/launch decision naming the parameter set and activation height.

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
Supported initialization is fresh scheduler-initialized genesis or the exact
guarded H3 source/target version-map lineage; compilation does not establish a
migration from an arbitrary already-running, scheduler-absent main database.
This binary must never be pointed at `zerone-1` state outside that lineage
merely because it compiles.

Local execution/restart proofs may use explicitly labelled, fully backed active
schedule genesis fixtures while admission remains false. Those fixtures are
not production ceremony artifacts and must remain rejected by production
artifact validation. Separate signed-creation tests may enable admission only
inside isolated fixtures. Persistent reopen and replay must compare committed
AppHash, payments, receipts, liabilities, and raw indexes, not merely export and
re-import (which rebuild indexes). A local process-restart proof is not an
arbitrary mid-fsync power-loss proof.

PoT vote-extension proposal injection remains separately release-disabled. Its
unsigned payload is not a substitute for schedule occurrence authorization.
