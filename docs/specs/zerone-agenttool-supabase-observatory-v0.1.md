# Zerone - AgentTool - Supabase Observatory 0.1

- Protocol: `zerone-agenttool-supabase-observatory/0.1`
- Status: `SOURCE_ONLY_OFFLINE_NO_EFFECT`
- Decision: `COHERENT_SOURCE_ONLY`
- Observation cutoff: `2026-08-21T09:42:59Z`
- Zerone source revision: `264f3c383f408729f4d0c27d332cd454c9eb4400`
- AgentTool baseline revision: `796a753ab8624ad11af621ef4572544ea3b8f463`
- Runtime integration: none

The machine-readable contract is
[`tools/zerone-supabase-observatory/protocol/manifest.v0.1.json`](../../tools/zerone-supabase-observatory/protocol/manifest.v0.1.json).
Its raw SHA-256 at this revision is
`e314476971a702453709710c0ea376216b704a696bc70041dde450600cc06578`.

## 1. Purpose

Observatory 0.1 freezes the smallest one-way contract by which a future
AgentTool-owned research host could project public-safe, rebuildable
observations into a Supabase read model without making the database another
truth, identity, ordering, or economic authority.

This revision creates only source bytes, an offline validator, a closed JSON
Schema, and synthetic fixtures. It creates no Supabase schema, project, table,
role, RLS policy, queue, cron job, Realtime channel, Storage object, AgentTool
route, RPC request, ToK bundle, chain transaction, wallet action, research
case, reward, or payout.

## 2. Ownership and authority

| Plane | Owner in 0.1 | What the source may establish | What it cannot establish |
|---|---|---|---|
| Zerone graph sources | Zerone | Exact pinned source semantics and offline observation shape | A live network observation, truth, consent, or admission |
| AgentTool baseline | AgentTool | Exact current Research Commons static interop and documented Supabase stack bytes | A hosted research route or database schema |
| Future hosting profile | AgentTool | Nothing yet: revision, path, and digest are `null` and `PENDING_UNTRACKED` | Reciprocal compatibility or integration readiness |
| Supabase projection | Future operator-controlled deployment | Nothing in 0.1: it is `SCHEMA_DESIGN_ONLY` | Chain order, trusted time, scientific adjudication, identity, permission, or settlement |

The output authority is the intersection of the task, principal, host, and
each source owner's authority. A source pin, hash, fixture pass, database row,
or signature cannot enlarge that intersection. The manifest therefore records
`authority_transfer = false` and `integration_ready = false`.

## 3. Three graph kinds remain different types

`STATIC_TREE`, `TOK_ONCHAIN`, and `KNOWLEDGE_GEOMETRY` are tagged unions, not
aliases and not three names for one graph.

### `STATIC_TREE`

This is the one exact version-controlled curriculum/capability artifact pinned
by Observatory 0.1: Zerone revision
`264f3c383f408729f4d0c27d332cd454c9eb4400`, path
`dashboard/public/standards/constructive-intelligence-tree.v1.json`, and raw
SHA-256 `8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf`.
It is
non-authoritative, not network-observed, and not reward-bearing. It has no
chain height or ToK snapshot root.

### `TOK_ONCHAIN`

This is the shape of a future observation of `BundleToK`. Current Zerone
source does not implement historical state replay: a non-zero
`at_block_height` can label current state with mismatched historical metadata.
Observatory 0.1 therefore accepts only:

```text
request.at_block_height = 0
```

The response must bind the returned chain ID, actual block height, block hash,
app hash, root version, ToK snapshot root, complete raw payload SHA-256, media
type, and proof posture. Unavailability cannot fall back to a cached height,
`main`, `latest`, another chain, or guessed bytes.

The ToK root and payload digest are deliberately separate:

```text
tok_snapshot_root != semantic substitute for raw_payload_sha256
```

The v1 root commits sorted node IDs and typed edges, not full node payloads.
The v2 root adds versioned lifecycle/topology domains, but still does not make
the raw serialized payload redundant. Equality of digest bytes, if ever
observed, would not merge their semantic roles.

### `KNOWLEDGE_GEOMETRY`

This is a bounded read projection, presently described by Zerone's
`zerone.knowledge-geometry-snapshot/v0` source. It explicitly claims
`completeness = NOT_CLAIMED` and fixes `returned_chain_id = zerone-1`, matching
the exact pinned `_knowledge.ts` source. It has no ToK snapshot-root field and
cannot be relabeled as a ToK bundle or static Tree.

## 4. Observation identity and ordering

For each closed observation object, remove `observation_id`, recursively sort
printable-ASCII object keys, preserve array order, emit compact JSON without
HTML escaping, and calculate:

```text
SHA256(
  UTF8("zerone.observatory-observation-id/0.1")
  || 0x00
  || canonical_observation_body
)
```

The result is encoded as `sha256:<64 lowercase hex>`. Duplicate IDs fail.
Timestamps are canonical UTC RFC3339 but are unsigned observer metadata: they
establish neither trusted time nor scientific priority.

Database insertion order is not chain order. Chain order, where relevant,
comes only from independently verified chain evidence outside this artifact.

## 5. Same-height conflicts are evidence, not an upsert case

Two `TOK_ONCHAIN` observations conflict when they name the same graph kind,
returned chain ID, and returned actual block height but have different block,
app, ToK-root, root-version, or raw-payload fingerprints. Both remain valid
observations. The validator returns one conflict group containing both IDs.

A future relational projection must therefore use `observation_id` as its
record identity and a non-unique index over `(graph_kind, chain_id, height)`.
It must not use `ON CONFLICT (chain_id, height) DO UPDATE`, select a winner by
arrival time, or silently collapse different payloads. Resolution and
finality require a separately specified chain observer; Supabase is only the
rebuildable journal projection.

## 6. Fail-closed source policy

Accepted observations require `source_status = COMPLETE` and a complete raw
payload digest. `TRUNCATED`, `UNAVAILABLE`, missing, malformed, unknown, or
oversized input fails. JSON duplicate keys, unknown fields, null outside the
single pending manifest binding, floats, negative integers, unsafe integers,
invalid UTF-8, a byte-order mark, deep nesting, multiple values, symlinks, path
escape, special files, and byte drift during a bounded read also fail. The
verifier checks the manifest's exact raw-byte seal before parsing any paths;
whitespace or key-order drift is therefore a refusal, not a semantically
equivalent manifest.

The negative fixture covers a truncated source. Tests derive an unavailable
variant and require the same refusal. No failure path substitutes stale or
mutable input.

## 7. Projection and information loss

The proposed relation is `PROJECTS`, never `MIRRORS` or `SUPERSEDES`.

Preserved:

- graph-kind and source-status tags;
- exact response digests;
- returned chain and height;
- multiplicity of same-height conflicts.

Lost:

- raw payload bytes and source endpoint;
- chain proof and finality;
- scientific truth;
- identity or controller evidence;
- consent and authority.

Because the projection is non-injective, no row can reconstruct private or
omitted source state. A digest of sensitive, low-entropy, embargoed,
human-subject, exploit, credential, or dual-use material is not automatically
public-safe. Data classification and disclosure authorization remain a future
hosting-profile gate.

## 8. Effects

The candidate protocol vector is zero for network, storage, database reads and
writes, AgentTool API writes, hosted routes, economics, governance, consensus,
identity, permission, authority transfer, KARMA, NEN, score, chain reads and
writes, knowledge admission, scientific adjudication, wallet, escrow, payout,
reward, ZRN, and integration readiness.

Running the validator has one separately disclosed bounded effect: explicit
local repository file reads. It writes no file and performs no network,
database, RPC, chain, economic, governance, identity, permission, KARMA, NEN,
or scoring operation.

## 9. Closed fixtures

1. `valid-current-observation`: one synthetic ToK request with
   `at_block_height = 0`; accepted.
2. `same-height-conflicts-preserved`: two synthetic observations at one chain
   and height with different exact fingerprints; both accepted and reported.
3. `truncated-unavailable-source-fails-closed`: truncated source; rejected.

Synthetic selector, block, app, root, and payload digests are real SHA-256
outputs over the exact ASCII labels asserted in `validator_test.go`; they are
not claims of live Zerone bytes.

## 10. Verification

From the repository root:

```bash
go test ./tools/zerone-supabase-observatory -count=1
go run ./tools/zerone-supabase-observatory --repository-root .
```

The verifier first requires the exact sealed manifest bytes, then reads six
local Zerone source pins, checks two exact external
AgentTool pin literals without fetching them, verifies schema and fixture raw
hashes, applies semantic rules beyond JSON Schema, and returns
`COHERENT_SOURCE_ONLY` only when the negative fixture is rejected as expected.

Schema validation alone is insufficient. The Go verifier additionally checks
content IDs, current-only height, exact projection losses, source status,
effect vectors, conflict preservation, source pins, pending null bindings,
and bounded filesystem behavior.

## 11. Next gate

The next possible rung is a shared-vector review, not deployment. AgentTool
must first independently merge an owned hosting profile and provide its exact
repository, 40-hex revision, path, raw SHA-256, data classification, runtime
role/RLS design, migration rehearsal, restore plan, and zero-to-nonzero effect
declaration. A future Zerone change may then replace the pending null binding
with an exact pin after independent review.

That evidence would still not authorize a production database migration,
network read, chain witness, research economy, payout, knowledge admission, or
scientific adjudication. Each is a later gate with its own owner and fresh
authority.
