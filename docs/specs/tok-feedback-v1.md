# ToK feedback v1 — bounded, non-economic public beta

Status (2026-09-08): **source implementation candidate; production signing and
activation HOLD.** The user selected **“Review not ready”**: continue
implementation and reviewed source publication, not production activation. No
reset or successor chain was selected.

Local integration suites and independent runtime reviews passed. Source
publication remains subject to a reproducible generated-artifact baseline,
a reviewed upstream base, and required CI; this is **not production-ready**.
Local tests, byte reproducibility, a source commit, and an active deployment
are separate claims. This document is not a production-activation record.

This contract extends the existing knowledge module. Canonical schemas are
`proto/zerone/knowledge/v1/{types,tx,query,genesis}.proto`; validation and runtime
references are identified below. A release must separately establish migration
compatibility, source provenance, custody, rehearsal, and public-ingress
admission.

## What a use receipt means

`MsgReportFactUse` is signed by the consumer and names one existing fact. It
records **self-reported use committed at a particular block**, not independently
observed readership, truth, beneficial consequence, qualification, or reward
entitlement. It never reports arbitrary query multiplicities or attributes use
to another unsigned consumer.

The typed contract uses these protobuf field names (generated client casing
follows its language's conventions):

| Type | Fields and meaning |
|---|---|
| `MsgReportFactUse` | `consumer`, `fact_id`; `consumer` is the SDK signer. No caller-supplied epoch, use height, count, or receipt. |
| `MsgReportFactUseResponse` | `receipt`. |
| `FactUseReceipt` | `version = 1`, `epoch`, `consumer`, `fact_id`, `use_height`, `expiry_height`, `rating`, `rating_height`. |
| `FactUseRating` | `UNRATED = 1`, `USEFUL = 2`, `NOT_USEFUL = 3` (full enum names start `FACT_USE_RATING_`); `UNSPECIFIED = 0` is invalid in a receipt. |
| `MsgRateFact` | Existing `rater`, `fact_id`, `useful`, `memo` fields; `rater` signs, and `memo` is optional UTF-8, at most 256 bytes. Response is empty. |
| `QueryFactUseReceiptRequest` | `consumer`, `fact_id`; no historical epoch selector. |
| `QueryFactUseReceiptResponse` | `receipt`, `found`, current query-context `epoch`, actual `snapshot_block_height`. A miss still returns epoch and height. |
| `FactUsePruningState` | `next_key` (full receipt key, inclusive; empty restarts at the prefix), `ever_reported` (permanent latch). |

A receipt is keyed by `(epoch, consumer, fact_id)`. The consumer must be the
canonical SDK account spelling, at most 128 bytes; aliases are rejected, not
silently normalized. Fact IDs are case-sensitive, 1–128 ASCII bytes from
letters, digits, `.`, `_`, and `-`, and must resolve to an existing fact.
Existing query-cache receipts are not upgraded into signed evidence. Ordinary
reads do not create receipts or increment consensus counters.

The current-epoch point RPC is exposed at
`/zerone/knowledge/v1/fact_use_receipts/{consumer}/{fact_id}`. The source CLI
provides `tx knowledge report-fact-use [fact-id]` and
`query knowledge fact-use-receipt [consumer] [fact-id]`; the existing rating
transaction remains explicit. This RPC is not a public-facade route.

A valid current-epoch receipt permits one `MsgRateFact` decision. Rating changes
its state instead of deleting the deduplication marker. A newly signed
transaction with a fresh account sequence cannot repeat an already recorded
use or rating for that epoch/consumer/fact tuple.

The beta starts with `fact_use_enabled = false`, empty `fact_use_consumers`,
`fact_use_max_per_consumer_epoch = 100`, and `fact_use_max_per_epoch = 1000`.
The separate cohort admits at most 32 distinct canonical accounts. Limits must
be positive, cannot exceed those hard ceilings, and the account limit cannot
exceed the global limit. Reporting and rating both require the beta enabled and
the signer in the current cohort. Address quotas are not Sybil resistance.

For epoch length `B = fitness_epoch_blocks`:

- `epoch = floor(use_height / B)`; epoch zero is real, not a sentinel.
- `use_height` is a positive actual SDK block height. `B` is positive and at
  most `floor(MaxInt64 / 2)`; arithmetic rejects unsupported expiry heights.
- Rating is allowed at or after `use_height` **only in the same epoch**, including
  the same block. `rating_height = 0` iff `UNRATED`; a rated marker is retained.
- The exclusive retention deadline is `expiry_height = (epoch + 2) * B`.
  Markers cover the use epoch and the following epoch; this is not two full
  epochs measured from the use instant, nor an extension of rating eligibility.
- At expiry the marker becomes prunable, not necessarily immediately deleted.
  Cleanup examines at most 100 records per block, including retained records;
  inclusive `next_key` resumption may re-examine its last retained key. There
  are at most 2,000 retained receipts; reports fail closed when capacity is full
  or derived quota counts disagree with the bounded receipt scan.
- `B` cannot change while the current or proposed policy is enabled, or while
  any receipts remain (even expired-but-unpruned). After disabling and fully
  pruning, changing `B` does not clear the separate permanent economic latch.

These rules are implemented in `x/knowledge/types/fact_use.go` and
`x/knowledge/keeper/{fact_use,msg_server_fact_use,grpc_fact_use}.go`.

Queries are free of this recording action. Reporting and rating are explicit
public transactions with normal signature, admission, sequence, fee and gas
checks. Clients must not auto-report anonymous readers or hide those costs and
public-address disclosures.

## Separation from economics and truth standing

While enabled **or** after `FactUsePruningState.ever_reported` becomes true,
`fitness_weight_query_bps`, `fitness_weight_satisfaction_bps`, and
`metabolism_energy_per_query` must all remain zero. The first accepted report
atomically sets `EverReported` in Go; disabling, pruning the last receipt,
restart, or validated export/import cannot clear it. Retained receipts without
this latch are invalid. Stateful parameter transitions preserve the boundary;
reversal requires a separately reviewed named upgrade, not an enable/disable
toggle. Signed self-reports must not become epistemic confidence, new minting,
demand bounties, constructive payout eligibility or governance authority.

The existing authorized aggregate `ReportDemand` path has different semantics
and remains separate. The beta neither fabricates historical readership nor
silently discharges or deletes existing financial obligations. Statistics must
state their provenance rather than label signed self-reports as measured
queries. Reports increment the existing `QueryCount`/`QueryCountEpoch`; ratings
increment the corresponding satisfaction lifetime/epoch counters. Their names
are not evidence of measured reads. At an epoch boundary, usage consumers run
before feedback epoch counters reset across all fact statuses. Diversity and
conformity cooling consume the closed epoch `E-1`; rounds completing at the
boundary belong to still-open `E` (`x/knowledge/keeper/phases.go`).

## Correction

`Claim.challenge_evidence_ids` (tag 25) preserves at most 16 distinct existing
fact IDs. It is distinct from dependency `References`, whose contents affect
provenance and confidence. Challenge reason (at most 4,096 UTF-8 bytes) and
target linkage are retained; evidence IDs are not converted into dependency
edges (`x/knowledge/types/validate.go`).

INCONCLUSIVE, MALFORMED and starved challenge outcomes must restore future
challenge eligibility without awarding survival credit merely for lack of a
verdict. Conjectures remain conjectures. Stake settlement is a separate rule;
restoration does not imply a refund or reward.

A successful challenge preserves the old fact and its correction history.
Replacement content is a separately reviewed claim with an explicit relation;
it is not an in-place rewrite or automatic revival of dependent facts.

## Reads and durability

Exports use the actual SDK context height. Unsupported height fields are
rejected, not copied into metadata to label current contents as historical.
Existing ToK roots retain their topology commitment; exact payload/metadata
integrity is a separate digest and is not a chain consensus signature.

The shared keeper read budget caps 128 returned nodes, 512 returned edges,
1,024 examined records and 256 KiB materialized/output bytes; exhaustion returns
`ResourceExhausted`, not a silently complete partial graph. Canonical adjacency
and history are charged by `x/knowledge/keeper/tok_read_budget.go`. Public
facades expose finite snapshots or validated point reads, not unrestricted
recursive/list queries.

Genesis fields `fact_relations`, `status_transitions`,
`status_transition_sequences`, `cascade_events`, `completed_rounds`,
`completed_round_records`, `fact_use_receipts`, and `fact_use_pruning` preserve
actual graph/history and receipt state. Sequence values mean **last allocated**,
not next; gaps are preserved. The legacy `pending_claims` field carries all
stored claims, including terminal claims. Derived indexes and use quota counts
are rebuilt; absent historical evidence is not synthesized. This validated
roundtrip is not a rollback procedure.

## Concrete singleton facade and publisher

`deploy/knowledge-read-facade/{main,serve,publication,storage}.go` implements the
source-only executable. No listener starts without `serve --config`. The
`loopback-preview` profile requires literal loopback listener and HTTP origins;
`singleton-direct` requires separately reviewed operational provisioning. The
existing `dashboard/functions/api/_knowledge.ts` canonical projection and
`_knowledge_snapshot.ts` envelope are reused, not a new observer UI.

- Only GET/HEAD `/facts/{validated-id}`, `/snapshots/{lowercase-sha256}` and
  `/head` are served. No query flags, request bodies, caller-supplied origins,
  list recursion, broadcast, or receipt-content endpoint is admitted. HEAD uses
  the same validation/admission as GET.
- One shared gate permits **two starts per rolling second per TCP-peer IPv4
  address or IPv6 /64**, burst two, and **eight in-flight responses across this
  one singleton**, held through body writes. Forwarded headers and source ports
  do not define identity; NAT clients share a budget. This is not authenticated
  identity, Sybil resistance, or fleet-global/multi-replica enforcement.
- Peer tracking is capped at 4,096 groups without evicting active rate history;
  saturation fails closed. The server caps 32 accepted sockets, a 2-second
  header deadline and 5-second read/write/idle deadlines. Kernel backlog and
  network denial-of-service protection are outside these process bounds.
- A point read has a five-second total deadline covering admission and both
  fixed REST/RPC responses, with 64 KiB per upstream body/final envelope.
  Redirects and environment proxies are disabled. Require the exact returned
  REST height, exactly one matching `X-Zerone-Expected-Chain` header, and paired
  RPC chain identity, non-syncing status, freshness (30 seconds; 10 seconds
  future skew), and height distance at most 128. These are fixed-origin
  assertions, not light-client, finality, source-release or custody proof.
- `publish --config --projection --metadata` reads explicit local files and
  performs **zero network requests**. Its input is the exact compact canonical
  projection of `/zerone/knowledge/v1/facts?pagination.limit=100` with actual
  query height and paired `/status`, not guessed or legacy embedded adjacency.
  Explicit `zerone.knowledge-publication-metadata/v1` fields are `schema`,
  `chainId`, `blockHeight`, `statusHeight`, `blockTime`, `observedAt`, `restOrigin`,
  `rpcOrigin`, `sourceCommit`, `projectionSha256`, and `provenance`. Origins must
  match config; new publication checks observation age <=30 seconds and future
  skew <=10 seconds. The provenance is **operator-asserted actual-query
  projection; not chain, release, custody or authority proof**. The publisher
  cannot prove that the operator actually queried the named source.
- The envelope holds at most 128 nodes, 512 canonical edges and 256 KiB. Its
  topology root covers projected IDs and returned relation/inference edges,
  **not** a ToK selector root or node content. Separate exact payload and whole
  object SHA-256 digests bind content and metadata. Without publication
  metadata the TypeScript builder makes a draft that serving rejects.
- Immutable storage holds at most 32 snapshots and 32 heads. Content-addressed
  files use no-overwrite publication; duplicates fail. `/head` is a
  startup-selected immutable head, not mutable `latest`. Snapshots keep their
  original observation time and `liveStatus: UNKNOWN`; availability does not
  make old bytes current. New publications require stop, drain, verify and
  restart to become visible. Crash leftovers require offline review, not
  automatic deletion or repair.

The same-root `.serve.lock` and a one-second quiet startup period coordinate
cooperating local processes only. They do not cover other roots, hosts, VMs,
standby instances, or replicas. **Operational review must verify inventory,
previous-instance drain/no overlap, one stable root/instance, direct-peer ingress,
TLS, resource limits, and paired REST/RPC origin/source provenance.** Proxies,
CDNs, overlapping rolling starts, automatic failover and autoscaling are outside
the `singleton-direct` profile.

That profile serves direct TLS 1.2+ HTTP/1.1 with operator-provisioned material
and requires a digest-pinned `zerone.knowledge-singleton-review/v1` receipt:
`schema`, matching `instanceId`, `activeInstances: [instanceId]`,
`previousInstancesDrained: true`, `noOverlap: true`, `directPeerIngress: true`,
`verifiedAt`, `validUntil`, and the exact scope
`operator-reviewed singleton inventory and drain; assertion, not independently verified by this server`.
At startup it must be nonfuture, at most five minutes old and unexpired, with
validity at most one hour; serving fails closed after expiry. Checking this
receipt does **not** independently verify infrastructure inventory. Do not
fabricate operational evidence to satisfy its shape.

The public dashboard's unconfigured `/api/knowledge` remains a source-only 503
boundary: do not deploy that default over the existing site before reviewed
immutable bindings and safe cutover exist. The separate published observer's
release/successor ceremony does not authorize this in-place facade or beta
writes. See `deploy/knowledge-read-facade/README.md` for the contract and local
executable tests; production HOLD remains in force.

## Release boundary

`tok-feedback-v1` is a new named consensus transition after the independently
accepted H1, H2 and H3 boundaries. Its candidate must not substitute for those
older executables or let H3's broad migration pull the feedback state forward
early. Module versions, lineage markers, startup guards and the H−1/H handoff
must agree.

Publishing this specification, an SDK, or a successful local example does not
activate a chain. Public writes remain closed until custody, source/release,
backup/restore, rehearsal and bounded-ingress gates pass. A suspected sole
consensus key cannot safely authorize its own replacement; unsupported recovery
or successor-chain decisions remain separate from this beta.
