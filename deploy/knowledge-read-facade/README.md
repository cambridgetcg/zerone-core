# Finite knowledge read service

**Source implementation; production activation remains HOLD.** No listener starts
without `serve --config`. Nothing here authorizes a chain upgrade, signer,
receipt write, reset, successor, or deployment. Do not deploy the dashboard's
unconfigured `/api/knowledge` 503 over its existing public API.

This package adds a small executable data service, **not another observer UI**.
It reuses the canonical `dashboard/functions/api/_knowledge.ts` projection and
`_knowledge_snapshot.ts` topology/digest contract. The published PR64 observer's
`network-profile.ts`, `src/observer-api.ts` and gateway remain separate; its
release/successor OPEN ceremony is not authority for this in-place read product.
Its active freshness window (30 seconds, 10 seconds future skew) is retained.

## Commands and local verification

From the repository root, with the supported Go toolchain selected:

```sh
GOPROXY=off GOSUMDB=off go build -mod=readonly -p 2 -o /tmp/knowledge-read ./deploy/knowledge-read-facade
GOPROXY=off GOSUMDB=off go test -race -mod=readonly -p 2 ./deploy/knowledge-read-facade -count=1 -v
```

`TestCLIProductPublishServeLimitsAndShutdown` builds the **actual executable with
`-race`**, runs `publish`, starts `serve` on a real loopback socket, reads point,
snapshot and head responses, checks conflicting lowercase identity, prevents a
second local instance, drives eight held requests at 510 ms intervals from **one
real TCP peer**, verifies a ninth point AND snapshot get 429, and cancels an
active read on SIGINT. It uses synthetic local upstreams; no production access.
The test's temporary files/listeners/processes are cleaned up. No runtime
credentials, wallet, private validator keys, network installer or provider are
needed. Go tests use only the standard library, apart from the enclosing module.

For deliberate local use, copy `config.preview.example.json` outside source and
set the absolute root and the two **literal loopback** HTTP origins to the local
REST/RPC pair. Unknown/missing/case-aliased fields, paths, credentials, queries,
fragments, non-loopback preview origins/listeners and invented review inputs fail
closed. Port zero is permitted for preview and prints the selected address.

1. Obtain the existing canonical **actual Facts-query projection**, not a raw
   REST body or guessed embedded relation arrays. The source query is exactly
   `/zerone/knowledge/v1/facts?pagination.limit=100`, with actual response height
   and paired `/status` context. `knowledgeRequest` plus its canonical wire test
   is the existing adapter; this publisher intentionally does not add a scanner
   or observer. Save its exact compact JSON bytes (no trailing newline).
2. Supply explicit `metadata.example.json` fields from that observation, including
   both approved origins, observed/block times, actual query/status heights,
   source commit reference and SHA-256 of the projection bytes. **The examples
   contain synthetic placeholders and are not admissible fresh observations.**
3. Invoke the publisher. The config's zero `headId` is only a pre-publication
   placeholder; it will not resolve during serving.

```sh
/tmp/knowledge-read publish --config /absolute/local-config.json \
  --projection /absolute/query-projection.json --metadata /absolute/observation.json
```

4. Record the returned `snapshotId` and `headId`. Select that exact `headId` in
   the operator config, then run and interact with the service:

```sh
/tmp/knowledge-read serve --config /absolute/local-config.json
# In another terminal, using the printed address and returned digest:
curl --fail http://127.0.0.1:8088/facts/FACT_ID
curl --fail http://127.0.0.1:8088/snapshots/SNAPSHOT_SHA256
# Wait one second before a third request from the same client.
curl --fail http://127.0.0.1:8088/head
# Ctrl-C (SIGINT) or SIGTERM drains/cancels and closes the listener.
```

The publisher performs **zero network requests**. It validates explicit metadata,
canonical projection structure, sorted IDs/relations, node/edge/byte bounds,
source heights and freshness, and recomputes the topology root and payload/body
SHA-256. All metadata is bound into the immutable envelope. The provenance string
says **operator-asserted**, not chain/release/custody/authority proof. Validation
cannot establish that an operator actually queried the claimed source. Fresh
publication requires observation age <=30 seconds and future skew <=10 seconds;
the claimed block must meet the same bounds relative to observation time.

The dashboard builder without explicit publication metadata still produces an
inspectable **draft**; serving rejects it. `generate-fixture.ts` reproduces the
shared TypeScript/Go byte vector using the dashboard's pinned `tsx`:

```sh
dashboard/node_modules/.bin/tsx deploy/knowledge-read-facade/generate-fixture.ts
```

Both language suites independently assert the exact fixture bytes and digest.
No TypeScript toolchain is required at service runtime.

## Routes and actual limits

Only GET/HEAD `/facts/{validated-id}`, `/snapshots/{lowercase-sha256}`, and `/head`
are available. No list, pagination, URL, commands, query flags, request bodies,
broadcast, write, generic proxy, arbitrary file or receipt-content URL is used.
HEAD incurs the same validation and admission as GET. Error responses are small
and non-cacheable. Upstream failures are not relabelled absent, zero or healthy.

- **One shared admission gate for all routes:** at most two starts per rolling
  second per TCP-peer IPv4 address or IPv6 /64, burst two; at most eight in-flight
  responses held through their body write. Points retain that same lease through
  both fixed REST and RPC requests. Forwarded/client headers and source ports
  never define identity. This is an IP-budget policy, **not Sybil resistance or
  an authenticated account limit**; NAT clients share a budget.
- At most 4,096 tracked peer groups, reclaimed only after their rate window
  expires; no eviction of an active client's history. Saturation fails closed.
- At most 32 accepted TCP sockets; 2-second request-header deadline, 5-second
  read/write/idle deadlines, 8-KiB configured header budget. Go's HTTP parser has
  its documented additional header buffering allowance; malformed transport
  requests are refused before the application gate. Kernel backlog and network
  DDoS are outside this process bound.
- Point deadline is five seconds **including admission, both responses and body
  reads**. Each upstream body and final point envelope is <=64 KiB. Connection
  pools are bounded; redirects, environmental HTTP proxies and compression are
  disabled. The fixed RPC chain must match, be non-syncing/fresh and within 128
  blocks of the actual REST response height. The original chain-binding gate is
  preserved: the reviewed REST origin must supply exactly one matching
  `X-Zerone-Expected-Chain` header; plain SDK REST without that binding is refused.
  This header is still only an operator/source assertion, not proof. Missing,
  duplicate/conflicting height or chain assertions fail closed. Exact lowercase `fact`/`id` lookups reject ALL
  case aliases, duplicates, malformed UTF-8 and nesting beyond 64 levels.
- Publication accepts <=128 nodes, <=512 canonical edges and <=256 KiB per
  immutable envelope. The canonical query adapter separately caps 1,024 examined
  records. No whole-graph coverage is claimed. A topology root excludes content
  and metadata; the separate payload and object digests commit those bytes.

## Immutable storage and restart behavior

The fixed root contains `<snapshot-sha256>.json`, `head-<head-sha256>.json` and two
zero-byte advisory OS lock files. No mutable latest file exists. `/head` is the
**startup-selected immutable head**, served no-store; it is not a latest-chain
claim. Snapshots retain original observation time and `liveStatus: UNKNOWN`;
serving historical immutable bytes does not make them fresh current facts.

The root holds at most 32 snapshots plus 32 heads (<=8 MiB snapshot bytes plus
128 KiB head bytes), with bounded directory enumeration. Startup loads immutable
bytes once, verifies every snapshot plus selected head identity/metadata, and
never refreshes from requests. Adding another publication does not mutate a
running service's map/head. Stop, drain, verify and restart to select a new head.

Root-relative operations prevent path escape. Symlinks, nonregular or unexpected
entries fail closed. Writers use a same-host `flock`, write/fsync a temporary file,
then hard-link it into the content-addressed name with no-overwrite semantics and
fsync the directory. Duplicate publication fails rather than replacing bytes. A
crash may leave an orphan snapshot or `.pending`; no automatic repair or deletion
conceals it. Review/recover offline and verify hashes before restart. At capacity,
provision a separately reviewed new root; immutable storage is not silently
pruned. The root is an operator-controlled local filesystem, not hostile shared
storage or a distributed lock service.

`.serve.lock` prevents two cooperating local processes on the **same root**.
Every start also holds a one-second quiet period before opening its listener to
avoid a restart resetting the rolling rate window. Locks do not cover another
root, host, VM or replica. Never claim these in-memory limits are fleet-global.

## Production singleton profile: external gates remain

`singleton-direct` is implemented but **not activated or production-verified**.
It requires fixed operator-approved HTTPS REST/RPC origins, a literal nonzero
listen endpoint, separately provisioned TLS certificate/private-key paths, and
an exact SHA-256-pinned local `deploymentReviewPath` receipt. Direct TLS 1.2+
HTTP/1.1 serving preserves TCP peer identity; no TLS keys are shipped here.
Proxies/CDNs, rolling overlaps, serverless replicas, automatic failover and
multi-instance autoscaling are **unsupported by this profile**. Do not present a
reverse proxy's IP as a verified end-user identity.

The exact review schema (see `deploymentReview` in `serve.go`) requires:
`schema: zerone.knowledge-singleton-review/v1`, the config's `instanceId`,
`activeInstances: [instanceId]`, `previousInstancesDrained: true`,
`noOverlap: true`, `directPeerIngress: true`, `verifiedAt`, `validUntil`, and
`scope: operator-reviewed singleton inventory and drain; assertion, not independently verified by this server`.
At startup the receipt must be at most five minutes old, nonfuture, unexpired,
and valid for at most one hour. After expiry the service fails closed; it does
not grant itself a renewal or discover another endpoint.

**Do not fabricate this receipt.** A separately authorized operator must actually
verify infrastructure inventory (including old/autostart/standby machines),
stop/drain the predecessor, enforce no overlap and one stable root/instance,
verify direct-peer ingress, install TLS and resource limits, and verify the paired
REST/RPC source/release provenance and required REST chain-header binding. The server checks the receipt's bytes and
internal consistency, not the truth of external inventory claims. It performs
no inventory API calls and has no deployment authority. Review the exact source,
load behavior, process/network resource limits, storage backups and recovery,
observability, origin pairing and safe API cutover before considering provision.
Current independent custody review is **not ready**; production activation is
held, no reset/successor was selected, and PR64's observer ceremony does not
clear this gate. Source publication is distinct from deployment acceptance.
