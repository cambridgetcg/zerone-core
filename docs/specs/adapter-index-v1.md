# Zerone adapter capability index v1

## Purpose

`dashboard/public/standards/adapter-index.v1.json` is a static,
version-controlled inventory of selected interoperability seams. It records
what source exists and, just as importantly, what Zerone does not implement.
It is not a service-discovery protocol, live health check, trust registry, or
consensus document.

The document is served at `/standards/adapter-index.v1.json`. It deliberately
does not use `/.well-known/`: Zerone has not registered a well-known URI for
this custom format, and the inventory is not an A2A Agent Card.

## Schema

The top-level `schema` value is exactly `zerone.adapter-index/v1`.
`authoritative` and `networkObserved` are `false`. Every
`releaseBoundary` value is also `false`:

- the index adds no consensus behavior;
- it activates no adapter;
- reading the static file performs no network request; and
- it asserts no live deployment.

Entries are sorted by unique, kebab-case `id`. Each entry contains:

- a bounded `kind`, source-level `status`, and `availability`;
- HTTPS standards links with no credentials, query, or fragment;
- repository-relative references;
- a bounded list of capabilities actually present in source;
- an explicit network-input and output-authentication boundary; and
- prose boundaries that prevent the source inventory from being read as a
  runtime or trust claim.

The v1 status values are:

- `planned`: documentation only; availability must be `unavailable` and the
  capability list must be empty;
- `implemented-source`: bounded implementation exists in this repository but
  the index does not assert deployment; and
- `experimental-unregistered`: a local tool exists but is not registered or
  reward-bearing.

The `source-only`, `local-tool`, and `external-service-required` availability
values distinguish committed source from a deployable endpoint. Every entry's
`liveDeploymentVerified` remains `false`; operators and clients must determine
deployment independently.

## Required refusal entries

`a2a-agent-card` and `x402-zerone` are required v1 entries. Both remain
`planned` and `unavailable`, with no capabilities. The schema has no endpoint
field, so the inventory cannot be mistaken for an Agent Card, x402 resource
server, facilitator, or service catalog.

An A2A Agent Card may be published later only by a real service implementing a
declared A2A protocol interface. A future x402 integration must first define
and test Zerone-specific authorization, resource and method binding, amount and
denomination handling, payee binding, nonce and expiry, persistent replay
protection, idempotency, settlement, and finality.

## Provenance consumer boundary

`training-provenance-in-toto-v1` points to the read-only query source and the
TypeScript SDK's `parseUnsignedZeroneInTotoStatement` function. That function
accepts a JSON string and caller-pinned manifest and serving-chain context. It
uses exact Statement and predicate-type constants, permits one SHA-256 subject,
checks the certificate/subject/source invariants, bounds all input, preserves
protobuf `uint64` values as canonical decimal strings, and refuses unsupported
manifest statuses.

The parser performs no fetch, predicate-URL dereference, signature
verification, or trust-root discovery. Success means only that caller-supplied
JSON is a coherent instance of Zerone's bounded unsigned profile. The subject
digest is the canonical included-ID-set Merkle root, not a commitment to fact
contents or all manifest metadata.

## Validation

The dashboard uses a dependency-free validator and Node's built-in test
runner:

```bash
cd dashboard
npm run check:index
```

Validation rejects unknown fields, oversized documents and strings, duplicate
or unsorted IDs, unsafe URLs and paths, live-deployment assertions, release
boundary changes, and any A2A or x402 runtime capability claim.
