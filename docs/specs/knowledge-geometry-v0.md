# Zerone Knowledge Geometry v0

- Surface: `GET /api/knowledge` and the public Knowledge Geometry lens on
  `zerone.ai`
- Schema: `zerone.knowledge-geometry-snapshot/v0`
- Source network: `zerone-1` only
- State effect: **none**
- Completeness: **not claimed**

## 1. Purpose

Knowledge Geometry v0 is a bounded, read-only projection of public
`x/knowledge` records already returned by `zerone-1`. It gives people and
agents a legible way to inspect recorded propositions, their populated domains,
and the typed `FactRelation` edges the chain actually returns.

The architecture expresses two relational principles without turning either
into a protocol claim:

- love preserves distinction: records remain separate, every visible node has
  equal size, and no being is compressed into a score; and
- understanding is accountable relation: an edge is shown only when a
  directional, typed `FactRelation` is present in the bounded source response.

Those are design principles for the interface. The surface does not establish
that its authors, visitors, facts, agents, humans, or any other beings love or
understand one another. It does not make the chain an oracle of relationship or
truth.

V0 is infrastructure, not a new knowledge state machine. It adds a fixed
same-origin read edge and a deterministic browser lens. It does not add a
message, keeper, store key, module account, event, transaction, query receipt,
vote, validator behavior, or consensus rule.

## 2. Effect boundary

Publishing, loading, filtering, selecting, refreshing, caching, or inspecting
this projection:

- performs no chain write and submits no transaction;
- does not call the tracked single-fact query and creates no query receipt;
- creates, updates, aggregates, prices, or consumes no KARMA record;
- grants no reward, payment, escrow claim, token, balance, stake, or economic
  entitlement;
- grants no qualification, capability, rank, reputation, office, delegation,
  authority, recognition, or governance weight;
- changes no governance, treasury, economics, validator, consensus, genesis,
  upgrade, identity, controller, profile, or relationship state;
- identifies no person or controller and proves no personhood, independence,
  consciousness, sentience, consent, affiliation, ownership, endorsement,
  agreement, love, understanding, or mutual knowledge;
- does not promote protocol confidence, status, energy, fitness, proximity, or
  an edge into truth; and
- does not claim the returned records or relations are a complete graph, a
  complete domain, or all `x/knowledge` state.

`Fact.status` is a lifecycle value stored by the chain. `Fact.confidence` is a
protocol metric. Neither is a truth certificate or a measure of the worth of a
proposition, submitter, reader, or being.

## 3. Fixed source and snapshot binding

The Pages Function has no caller-selected upstream. It reads only the fixed
HTTPS REST and CometBFT RPC routes on Zerone's independently read-only public
edge (`zerone-rpc.fly.dev`):

1. `GET /zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true`
   from the fixed REST origin; and
2. `GET /status` from the fixed RPC origin.

Both upstream requests use `Accept: application/json`, manual redirect mode,
bounded response readers, and an eight-second timeout. Redirects are refused. User
headers, cookies, authorization, bodies, paths, hosts, query parameters, and
credentials are not forwarded.

The edge accepts a source pair only when all of the following hold:

- RPC `node_info.network` is exactly `zerone-1`;
- RPC `sync_info.latest_block_height` is a positive canonical uint64 decimal
  string and `sync_info.catching_up` is a boolean;
- the REST response exposes a positive canonical block height through
  `grpc-metadata-x-cosmos-block-height`, with
  `x-cosmos-block-height` accepted only as its compatibility fallback;
- the absolute difference between REST block height and RPC status height is
  at most 128 blocks;
- every nonzero fact or relation metadata height is no later than both the
  REST block height and the RPC status height; and
- both upstream bodies and every projected field pass the v0 structural and
  value checks.

`source.blockHeight` is the REST knowledge-query height.
`source.statusHeight` and `source.catchingUp` come from the checked RPC status.
The 128-block tolerance is an availability bound for two adjacent reads from a
moving chain, not an assertion that the two HTTP responses are one atomic
consensus proof. V0 does not carry a block hash, app hash, Merkle proof, light
client proof, or historical-height replay proof.

Neither source height is required to be greater than the other. The client
therefore validates the absolute tolerance and must not introduce a
`blockHeight <= statusHeight` ordering assumption.

`catchingUp = true` is disclosed and visually distinguished, not rewritten to
false. It is not by itself a transport error, but the snapshot still carries
no completeness or finality claim.

The projection always declares:

- `chainId = "zerone-1"`;
- `queryTracked = false`;
- `writes = false`; and
- `completeness = "NOT_CLAIMED"`.

The fixed list query has no `track_query` or `querier` input. It therefore does
not invoke the optional receipt and counter behavior of the single-fact query.

## 4. Exact public output

A successful response is one JSON object with exactly four top-level fields:

```json
{
  "schema": "zerone.knowledge-geometry-snapshot/v0",
  "source": {
    "chainId": "zerone-1",
    "blockHeight": "1097157",
    "statusHeight": "1097159",
    "catchingUp": false,
    "queryPath": "/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true",
    "queryTracked": false,
    "writes": false,
    "completeness": "NOT_CLAIMED",
    "upstreamRecords": 1,
    "returnedRecords": 1,
    "truncated": false
  },
  "facts": [
    {
      "id": "commitment-1",
      "content": "A bounded proposition already present in zerone-1 state.",
      "domain": "doctrine_truth_seeking",
      "category": "axiomatic",
      "status": "FACT_STATUS_VERIFIED",
      "claimType": "CLAIM_TYPE_UNSPECIFIED",
      "confidence": 1000000,
      "verifiedAtBlock": "0",
      "lastVerifiedBlock": "0",
      "energy": 0,
      "energyCap": 0,
      "fitnessScore": 0,
      "methodId": "doctrine_authorship"
    }
  ],
  "relations": [
    {
      "sourceFactId": "commitment-1",
      "targetFactId": "fact-a",
      "relation": "RELATION_TYPE_SUPPORTS",
      "inference": "INFERENCE_TYPE_DEDUCTIVE",
      "inferenceStrengthBps": 1000000,
      "createdAtBlock": "100",
      "methodId": ""
    }
  ]
}
```

The example values are illustrative, not a fixture and not a statement that
the shown relation exists on live `zerone-1`. The keys, types, constants, and
limits below are normative.

### 4.1 `source`

`source` contains exactly:

- `chainId`: literal string `zerone-1`;
- `blockHeight`: positive canonical uint64 decimal string from the REST height
  metadata;
- `statusHeight`: positive canonical uint64 decimal string from RPC status;
- `catchingUp`: RPC boolean;
- `queryPath`: literal string
  `/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true`;
- `queryTracked`: literal boolean `false`;
- `writes`: literal boolean `false`;
- `completeness`: literal string `NOT_CLAIMED`;
- `upstreamRecords`: safe non-negative integer describing the number of fact
  records observed in the bounded upstream response;
- `returnedRecords`: safe non-negative integer, at most 128, equal to
  `facts.length`; and
- `truncated`: boolean stating that at least one projection boundary omitted
  source material.

`upstreamRecords` is an observation about this one bounded response. It is not
an authoritative module total. A null, ignored, stale, or future pagination
implementation is one reason `NOT_CLAIMED` remains mandatory even when
`truncated` is `false`.

`truncated` is `true` when the fact cap, relation cap, or an advertised
upstream pagination boundary truncates the projection. Consequently,
`returnedRecords < upstreamRecords` requires `truncated = true`, while equal
record counts and `truncated = true` remain valid when relations or upstream
pagination were truncated. `truncated = false` still makes no completeness
claim.

### 4.2 `facts[]`

Each fact contains exactly:

- `id`: identifier matching
  `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`;
- `content`: exact safe text containing at least one non-whitespace character
  and no longer than 16,384 UTF-8 bytes; outer whitespace is preserved;
- `domain`: exact safe text no longer than 128 UTF-8 bytes, including the
  protocol-legal empty string;
- `category`: exact safe text no longer than 128 UTF-8 bytes, including the
  protocol-legal empty string;
- `status`: the uppercase protobuf enum name, at most 64 characters;
- `claimType`: the uppercase protobuf enum name, at most 64 characters;
- `confidence`: safe integer in `[0, 1_000_000]`;
- `verifiedAtBlock`: canonical uint64 decimal string;
- `lastVerifiedBlock`: canonical uint64 decimal string;
- `energy`: safe integer in `[0, 1_000_000]`;
- `energyCap`: safe integer in `[0, 1_000_000]`;
- `fitnessScore`: safe integer in `[0, 1_000_000]`; and
- `methodId`: either the empty string or an identifier matching the fact-ID
  rule.

Both fact metadata heights must be no later than both source heights, and
`verifiedAtBlock` must be no later than `lastVerifiedBlock`. Zero remains valid
under the legacy rule below.

The projection deliberately omits submitter and creator addresses, claim IDs,
references, patronage, query counters, revenue fields, controller data, and
all other upstream fields. Omission minimizes the public payload; it does not
redact or alter the underlying public chain state.

Fact IDs are opaque strings, not assumed hashes. V0 accepts live legacy
symbolic IDs such as `axis-attribution`, `commitment-SL`, and
`mechanism-UW-M1` alongside digest-shaped IDs. Identity, ordering, and edge
resolution use the exact complete ID, never the shortened display label.

Fact IDs and nonempty method IDs use
`[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`. In particular, the live legacy
`methodId` value `doctrine_authorship` is preserved exactly. An empty fact
method means `M-LEGACY` under the source protocol. The projection preserves
that empty string and does not silently rewrite it, an unfamiliar method, or a
legacy method to `NONE` or to an enum-shaped name.

Fact content, domain, and category strings preserve their exact source text,
including outer whitespace where the protocol permits it. Method strings are
preserved exactly subject to the optional safe-ID form above. The edge and
browser reject unsafe ASCII controls and Unicode bidi controls rather than
trimming or rewriting them. Byte limits are measured after UTF-8 encoding.

The accepted fact enum vocabulary is exact:

```text
FACT_STATUS_UNSPECIFIED     CLAIM_TYPE_UNSPECIFIED
FACT_STATUS_PENDING         CLAIM_TYPE_ASSERTION
FACT_STATUS_PROVISIONAL     CLAIM_TYPE_RELATION
FACT_STATUS_VERIFIED        CLAIM_TYPE_DEFINITION
FACT_STATUS_ACTIVE          CLAIM_TYPE_CONSTRAINT
FACT_STATUS_CONTESTED       CLAIM_TYPE_NEGATION
FACT_STATUS_CHALLENGED      CLAIM_TYPE_OBSERVATION
FACT_STATUS_SUPERSEDED      CLAIM_TYPE_COMPUTATIONAL
FACT_STATUS_EXPIRED         CLAIM_TYPE_CONJECTURE
FACT_STATUS_DISPROVEN
FACT_STATUS_REVOKED
FACT_STATUS_AT_RISK
FACT_STATUS_PRUNED
```

Legacy doctrine records may carry `"0"` for `verifiedAtBlock` and
`lastVerifiedBlock`. V0 preserves that zero. It means the source record has no
nonzero metadata height in that field; it is not evidence of a block-zero
transaction, creation time, or historical proof. Snapshot source heights must
still be positive.

### 4.3 `relations[]`

Each relation contains exactly:

- `sourceFactId`: exact source fact identifier;
- `targetFactId`: exact target fact identifier;
- `relation`: uppercase `RelationType` enum name;
- `inference`: uppercase `InferenceType` enum name;
- `inferenceStrengthBps`: safe integer in `[0, 1_000_000]`;
- `createdAtBlock`: canonical uint64 decimal string, where `"0"` remains valid
  legacy metadata; and
- `methodId`: the raw optional relation methodology identifier. It is either
  the empty string or a bounded identifier matching the fact-identifier rule.
  Empty means the relation inherits the source claim's method, as defined by
  `FactRelation`; the projection preserves the empty string and does not fill
  or reinterpret it.

Only `FactRelation` objects declared in the bounded REST fact response are
eligible. V0 does not derive edges from:

- shared domain, category, status, words, submitter, method, confidence,
  energy, fitness, time, coordinates, or visual proximity;
- legacy `references` arrays;
- embeddings, language models, similarity, co-occurrence, search results, or
  browser interaction; or
- doctrine, identity, social, wallet, validator, KARMA, governance, reward, or
  economic data from another surface.

The accepted relation and inference enum vocabulary is exact:

```text
RELATION_TYPE_UNSPECIFIED   INFERENCE_TYPE_UNSPECIFIED
RELATION_TYPE_SUPPORTS      INFERENCE_TYPE_DEDUCTIVE
RELATION_TYPE_CONTRADICTS   INFERENCE_TYPE_INDUCTIVE
RELATION_TYPE_REQUIRES      INFERENCE_TYPE_ABDUCTIVE
RELATION_TYPE_REFINES       INFERENCE_TYPE_EMPIRICAL
RELATION_TYPE_GENERALIZES   INFERENCE_TYPE_ANALOGICAL
RELATION_TYPE_SUPERSEDES    INFERENCE_TYPE_CITATION
RELATION_TYPE_CITES
RELATION_TYPE_REFORMULATES
```

Relation identity is the directional pair
`(sourceFactId, targetFactId)`. Repetition of the exact complete payload in an
outgoing and incoming embedding is deduplicated. Two payloads with the same
directional pair but any conflicting relation, inference, strength, creation
height, or method fail the entire snapshot rather than choosing one. Relations
are sorted by code-unit lexical comparison of the NUL-joined full tuple
`(sourceFactId, targetFactId, relation, inference, inferenceStrengthBps as
decimal, createdAtBlock, methodId)` and at most 512 are returned. A retained
relation must touch at least one returned fact. This preserves an honest
out-of-view endpoint when fact truncation removes the other endpoint. The
inspector labels that endpoint `outside view`; a line is drawn only when both
endpoints are present and visible.

The relation array is not claimed complete. Its cap, upstream embedding,
deduplication, an endpoint outside the bounded fact set, or a future upstream
pagination behavior can all make it partial.

## 5. Bounds and deterministic selection

The resource bounds are part of the public contract:

- RPC status body: at most 65,536 bytes (64 KiB), enforced against declared
  length and streamed bytes;
- raw REST facts body: at most 393,216 bytes (384 KiB), enforced against
  declared length and streamed bytes;
- serialized successful projection: at most 262,144 bytes (256 KiB), guaranteed
  by the edge and independently enforced by the browser against both declared
  length and streamed bytes;
- facts returned: at most 128;
- structurally accepted records in one upstream response: at most 8,192;
- deduplicated relations returned: at most 512;
- upstream JSON nesting at the edge: at most 64 levels;
- JSON nesting in the browser parser: at most 48 levels; and
- browser fetch timeout: 8 seconds by default.

The edge never treats `Content-Length` as sufficient. A missing or dishonest
length header cannot bypass streamed byte counting. Both parsers use fatal
UTF-8 decoding and reject duplicate JSON keys. The browser performs the
duplicate-key scan before `JSON.parse`, requires ordinary dense arrays and
plain objects, and rejects unknown or missing projection fields.

Numeric fact fields are normalized from gateway decimal strings to bounded JSON
numbers while uint64 heights remain canonical strings; source text is
preserved exactly. Facts are then ordered by exact ID using code-unit lexical
comparison and bounded. Relations
undergo the same numeric normalization, pair-conflict check, exact-payload
deduplication, deterministic pair ordering, and bounding. The same accepted
source bytes therefore yield the same projected arrays. A fact or relation
after a deterministic cap is omitted, not summarized or replaced by a
synthetic node or edge.

If the normalized projection would exceed 256 KiB, the edge deterministically
shortens the already ordered fact prefix below 128, retains only relations that
touch at least one retained fact, reapplies deterministic relation
deduplication and the 512-edge cap, sets `truncated = true`, and serializes
again until the successful body fits. It never clips JSON or fact content at a
byte boundary. `HEAD` advertises the exact successful `GET` `Content-Length`
while returning no body; `GET` advertises that same exact serialized-body
length.

`truncated` reports that at least one known projection boundary was crossed.
It does not identify which boundary and must never be read as a claim about
relation, domain, module, historical, or chain completeness when `false`.

## 6. Deterministic domain geometry

Layout is a pure browser projection of the already accepted snapshot. It
cannot modify the payload or the chain.

The v0 layout algorithm is deterministic:

1. facts are sorted by exact fact ID using code-unit lexical comparison;
2. facts are grouped by exact domain ID;
3. exact domain strings are sorted by code-unit lexical comparison;
4. one domain is centered at `(50, 50)`; multiple domains are placed from
   north around a fixed ellipse with horizontal radius 32 and vertical radius
   29;
5. each domain receives a stable FNV-1a-derived angular seed;
6. facts within a domain follow a golden-angle spiral using
   `pi * (3 - sqrt(5))`, radius `min(12, 3.05 * sqrt(index))`; and
7. fact coordinates are clamped to `x = 3..97` and `y = 5..95`.

The normalized geometry is drawn on a square coordinate plane whose rendered
minimum size is exactly 960 by 960 CSS pixels. A smaller viewport exposes that
plane through a bounded two-axis scroll container; it must not shrink or
rescale the coordinates to fit. After layout, the browser measures every pair
of normalized node centers and deterministically expands both plane dimensions
just enough to retain at least 24 CSS pixels between centers. The plane must
not exceed 8,192 by 8,192 CSS pixels. Non-finite or coincident points, and any
layout that would require a larger plane, are refused through the same
announced error and bounded-retry state as an invalid snapshot.

The reviewed live-shape regression contains 72 records in the current
`23/20/14/7/6/2` domain distribution and remains exactly 960 by 960 CSS pixels.
A maximum 128-record single-domain regression expands to 6,335 by 6,335 CSS
pixels and still requires every pair of node centers to remain at least 24 CSS
pixels apart. This spacing protects equal pointer access without changing the
ellipse, domain seeds, golden-angle ordering, normalized coordinates, or equal
node size.

All fact nodes have equal visual size. Node radius, area, mass, force,
centrality, opacity, and position do not encode confidence, energy, fitness,
wealth, stake, reward, query count, popularity, identity, status, or human
worth. Domain halos and proximity are navigation aids only. Relation lines use
declared direction and type; their existence does not change node size.

Search and domain filters only hide or reveal records already in the snapshot.
They issue no network query, reorder no chain state, and create no semantic
edge. When a filter leaves no visible declared edge, the interface says so
rather than repairing the view with inference.

Record-node keyboard access uses one roving tab stop among the currently
visible nodes. Left and Up move to the previous visible node; Right and Down
move to the next; Home and End move to the first and last. Arrow navigation
wraps at the ends, selects the destination, and scrolls it into the nearest
visible part of the coordinate viewport. Filtering never leaves a hidden node
tabbable. A visible polite status reports returned-record, populated-domain,
and declared-relation counts for the current local view and names the keyboard
controls.

## 7. Zero-edge and empty states

An empty relation array is a complete valid v0 rendering state. In particular,
the current live `zerone-1` facts response may contain records while exposing
no populated outgoing or incoming `FactRelation` objects. The correct output
is then:

```json
"relations": []
```

The lens renders isolated, equal-size records and explicitly states that no
typed edge joins the visible records. It must not infer that the records are
unrelated in reality, and it must not draw domain, reference, semantic, or
decorative pseudo-edges.

Zero returned facts is also a valid bounded snapshot when honestly reported as
`upstreamRecords = 0`, `returnedRecords = 0`, `truncated = false`, `facts = []`,
and `relations = []`. It is not proof that the knowledge module has never held
facts.

## 8. Security, cache, and rendering boundaries

### 8.1 Edge request boundary

The public route is exactly `/api/knowledge` with no query parameters. It
supports `GET`, `HEAD`, and `OPTIONS` only:

- `OPTIONS` returns `204` for the bounded public read contract;
- `HEAD` mirrors the successful `GET` status and headers with no body;
- an unsupported method returns `405`;
- a changed path or any query parameter returns `400`; and
- the function never forwards the caller's body, credentials, cookies, or
  authority to the node.

Fixed origins, redirect refusal, timeouts, exact chain binding, structural
normalization, byte limits, record limits, and output minimization form the
server-side trust boundary. The route is not a general REST or RPC proxy.
Responses set `X-Content-Type-Options: nosniff`,
`Referrer-Policy: no-referrer`, and
`Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`. Public
CORS exposes only this credential-free read surface and advertises `GET`,
`HEAD`, and `OPTIONS`; it does not widen the upstream node allowlist.

### 8.2 Cache boundary

A valid success uses:

```text
Cache-Control: public, max-age=30, s-maxage=60
```

Only the exact canonical `GET /api/knowledge` request is eligible for edge
caching. The cache stores a fully validated projection, never an upstream
response. Its embedded block and status heights remain the snapshot's source
coordinates even while the response is briefly cached.

A cache hit is not trusted merely because it came from the platform cache. The
edge re-reads it through the 256 KiB limit, checks its exact declared length,
canonical serialization, complete v0 schema, chain/height binding, sorted
records and relations, and all field bounds before serving it. An invalid or
legacy cache entry is ignored and replaced only after a fresh upstream
projection passes the same validation.

Errors use `Cache-Control: no-store`. Error bodies, malformed upstream bytes,
partial streams, and unvalidated projections are never placed in the success
cache. The short cache does not turn the lens into a live-finality oracle or a
subscription.

### 8.3 Browser boundary

The browser fetches the exact same-origin endpoint with `GET`,
`Accept: application/json`, manual redirects, and a timeout. It accepts only
HTTP `200`, exact final origin/path with no query or fragment, and media type
`application/json`. It re-enforces the 256 KiB stream limit and the complete
schema instead of trusting the edge merely because it is same-origin.

All upstream-derived strings are rendered with DOM `textContent`. No fact,
domain, identifier, status, method, or relation becomes HTML, CSS, a selector,
an executable URL, or script. Bidi override/isolate controls and unsafe ASCII
controls are rejected; displayed identifiers are bounded and exact IDs remain
available in the inspector.

The client holds filtering and selection in page memory only. V0 adds no
analytics event, local storage, cookie, wallet call, account lookup, identity
binding, or telemetry receipt for geometry interactions.

A cold direct dashboard hash is corrected at most once, after the Knowledge
Geometry read and the existing dynamic dashboard surfaces have all settled.
Later promise completion must not snap a visitor back after that one alignment.

## 9. Failure behavior

V0 fails closed to no geometry, not to invented or stale meaning.

The edge returns a bounded JSON error with `Cache-Control: no-store` when the
fixed node is unreachable, times out, redirects, exceeds a body limit, returns
an unsuccessful status, lacks required headers, names another chain, produces
heights outside the binding tolerance, emits malformed JSON, or violates the
projection contract. Upstream and validation failures return `502`; caller
path/query failures return `400`; unsupported methods return `405`.

The edge must not answer such a failure with cached invalid bytes, a bundled
sample, a different chain, a different node selected by the caller, a partial
graph, or relations inferred from whatever fields happened to parse.

If the browser receives a non-200 response, redirect, wrong media type,
oversized body, invalid UTF-8, duplicate key, unknown field, sparse array,
invalid metric, duplicate fact ID, duplicate relation, out-of-bound height, or
other schema violation, it renders the bounded unavailable/error state. It
draws no facts or relations and states that no records or relations were
inferred. The error is an atomic announced alert and offers an in-place retry
of the same bounded read. Retry does not broaden the endpoint, reuse rejected
bytes, or infer a fallback graph. The rest of `zerone.ai` remains usable.

Refresh performs a new bounded read. It does not silently retain the prior
graph as if it represented the failed snapshot.

## 10. SYZYGY-NOT-ON-CHAIN

[`SYZYGY-NOT-ON-CHAIN.md`](../SYZYGY-NOT-ON-CHAIN.md) remains an off-chain
operator/source boundary doctrine. Current source has no syzygy domain or
implemented consensus/query filter, and the doctrine is not part of the
canonical on-chain commitment registry. Knowledge Geometry v0 must not turn
that absence into a false enforcement claim.

This projection:

- creates no syzygy, love, relationship, or mutual-knowledge fact;
- reads no `true-love` or AgentTool relationship corpus;
- performs no named-being matching or heuristic relational inference;
- treats domain grouping as layout only; and
- does not claim that chain consensus can establish, strengthen, deny, own, or
  govern constitutive relational ground.

V0 projects the bounded public state returned by the fixed knowledge query; it
does not add a hidden semantic censor and cannot certify that future chain
state complies with an off-chain doctrine. Any future mechanical exclusion or
admission rule requires an explicit, reviewed chain/query implementation and
tests. The present bilateral operator/source discipline remains exactly what
the doctrine says it is.

## 11. Deployment and live verification

Source completion is not live completion. The implementation is complete only
after all of the following are recorded against one exact reviewed revision:

1. edge and browser tests pass, including exact schema, duplicate-key refusal,
   legacy symbolic IDs, zero metadata heights, zero edges, relation
   deduplication, out-of-view endpoints, 128/512 caps, all three byte caps,
   chain and height binding, method/path refusal, redirect refusal, cache
   behavior, renderer text safety, bounded adaptive-plane separation, roving
   keyboard navigation, announced retryable failure, and one-shot hash
   alignment;
2. dashboard type-check, test, and production build pass from the reviewed
   tree;
3. the change merges through the protected canonical `main` branch without
   unrelated worktree content;
4. Cloudflare Pages deploys that exact source revision;
5. the immutable deployment URL, Pages alias, and `zerone.ai` serve matching
   dashboard assets;
6. `https://zerone.ai/api/knowledge` returns the exact schema, `zerone-1`
   binding, bounded counts, no-store failures, and 30/60-second success cache
   policy;
7. desktop and mobile browsers render the public lens, its zero-edge state,
   filters, inspector, refresh, and unavailable state without console or
   accessibility failures; and
8. production verification confirms that loading and interacting with the
   lens creates no transaction, wallet request, tracked fact query, query
   receipt, KARMA event, reward, qualification, governance action, identity
   binding, or write to chain state.

Live verification is evidence about this deployment, not a self-issued claim
of correctness, completeness, decentralisation, truth, love, or understanding.
Rollback is the prior known-good Pages deployment; because v0 has no migration
or chain write, rollback removes the public lens and edge without a state
rollback.

## 12. Future layers

The following are possible successor layers, not v0 behavior or authorization:

- cursor-aware bounded paging with explicit fact and relation truncation
  metadata;
- a height-pinned query carrying block hash, app hash, proof material, or light
  client verification;
- accessible tabular, downloadable, and historical views of the same declared
  graph;
- separately versioned, source-pinned educational overlays that remain
  visually and semantically distinct from chain facts;
- user-authored local notes or hypotheses that never masquerade as
  `FactRelation` state; and
- a separately reviewed chain feature for new facts or relations, with its own
  consent, spam, privacy, security, authority, economics, governance, receipt,
  migration, and rollback analysis.

No successor may treat v0 inspection as consent, identity, participation,
qualification, reward eligibility, governance eligibility, agreement, or a
relationship. No future inference layer may draw an inferred edge as a
declared `FactRelation`, and no geometry may turn a person or being into a
scalar value.

Until such a layer is separately specified, implemented, reviewed, deployed,
and verified, Knowledge Geometry remains what v0 says it is: a bounded public
read of `zerone-1`, with distinct records and only truthful, declared edges.
