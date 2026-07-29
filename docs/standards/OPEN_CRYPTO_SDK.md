# Open crypto SDK and standards integration

Zerone gets the most leverage from open standards at its boundaries, while
keeping consensus limited to witness-and-record primitives. This note records
the implemented interoperability seams and the recommended SDK roadmap.

## Implemented: portable account identifiers

`x/auth` projects registered Zerone accounts whose stored address is canonical
lowercase, 20-byte `zrn` Bech32 as:

- a [CAIP-2](https://standards.chainagnostic.org/CAIPs/caip-2)-syntax Cosmos
  chain ID;
- a [CAIP-10](https://standards.chainagnostic.org/CAIPs/caip-10)-syntax account
  ID; and
- its existing Zerone address, `did:zrn` label, account type, frozen state, and
  creation height.

Example for the live `zerone-1` network:

```text
chain:   cosmos:zerone-1
account: cosmos:zerone-1:zrn1...
```

Query it over gRPC, REST, or the CLI:

```bash
zeroned query zerone_auth account-identifier zrn1...
curl http://localhost:1317/zerone/auth/v1/account_identifier/zrn1...
```

The implementation follows the
[Cosmos CAIP-2 namespace profile](https://namespaces.chainagnostic.org/cosmos/caip2):
chain IDs matching `[-a-zA-Z0-9]{1,32}` are represented directly, except
values beginning with `hashed-`. All other nonempty IDs use
`hashed-<first-16-lowercase-hex-of-sha256(chain-id)>`. The profile is still
Draft even though CAIP-2 and CAIP-10 are Final, so the formatter and vectors
remain isolated in `x/auth/types/caip.go`.

The separate draft
[Cosmos CAIP-10 address profile](https://namespaces.chainagnostic.org/cosmos/caip10)
currently names only the `cosmos` and `cosmosvaloper` HRPs, not Zerone's `zrn`
HRP. Zerone therefore describes this output precisely as CAIP-10 syntax, while
enforcing a canonical lowercase `zrn` address with a 20-byte payload.

This is a computed query projection. It adds no KV writes, events, parameters,
genesis fields, migrations, consensus-version changes, bank calls, or IBC
activity. Otherwise-consistent historical or genesis-injected account records
outside the stricter CAIP projection remain queryable through the native
account API and return `FailedPrecondition` here rather than being silently
rewritten or aliased.

The `did:zrn` value is opaque native metadata in this response. It is not a
claim that Zerone currently implements [W3C DID Core](https://www.w3.org/TR/did-core/),
a published DID method, or `did:pkh`.

## Implemented: close generated REST coverage gaps

The application now registers five generated query clients that its manual v2
gateway list previously omitted: counterexamples, creed, substrate bridge,
training provenance, and trust score. Their declared `google.api.http` routes
are now reachable through the same REST gateway as the other custom modules.

This removes five concrete REST/SDK gaps without activating a write path or
changing the safety posture of any module. The generated OpenAPI document now
contains the complete protobuf-derived REST surface, including the CAIP and
in-toto queries.

## Implemented: unsigned in-toto training provenance

`x/training_provenance` projects the existing live certificate for a
non-composed `FINALIZED`, `ATTESTED`, or `SUPERSEDED` manifest directly into an
[in-toto Statement v1](https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md):

```bash
curl http://localhost:1317/zerone/training_provenance/v1/in-toto/<manifest-id>
```

The statement keeps the sealed manifest's source chain distinct from the chain
serving the query after an export or relaunch. Its SHA-256 subject digest is
precisely the manifest's included-ID-set commitment, not a hash of fact content,
metadata, or version-pin fields. Predicate v1 refuses draft and composed
manifests because its certificate counts direct fact/action/domain coverage,
while its incident count is module-global rather than manifest-specific. The
versioned
[predicate specification](../specs/attestations/training-provenance-v1.md)
records those exact boundaries and freezes them: a semantic change requires a
new predicate version and URI.

This is an unsigned, current-state projection with no store, transaction,
reward, or consensus-version change. A producer can sign the returned JSON
off-chain without making validators parse signatures or remote documents.
Missing manifests return `NotFound`; draft or composed manifests return
`FailedPrecondition`; malformed sealed state returns `DataLoss`; and server
configuration failures return `Internal`. Public REST operators should
rate-limit this live synthesis and may cache by `(chain ID, block height,
manifest ID)`, because certificate construction scans global audit records.

## Implemented: Sigstore evidence into substrate bridge

The isolated
[`sigstore-substrate-compiler`](../../tools/sigstore-substrate-compiler/)
verifies a local Sigstore DSSE bundle using a pinned local trust root, one
exact certificate issuer and SAN, an exact predicate type, and a required
artifact digest. It then emits the existing `x/substrate_bridge`
`SubstrateLink` shape.

The payload-byte `source_id` commits the exact decoded signed payload, while
`content_hash` commits the exact accepted bundle bytes, including signature,
certificate, SCT, and transparency-log evidence. The result is witness-only:
no facts, pending claims, recursion weight, or automatic reward. Consensus
checks the existing canonical link; Sigstore cryptography remains explicitly
off-chain. The [adapter specification](../specs/adapters/sigstore-in-toto-v1.md)
requires governance to pin the compiler build, trust-root digest, invocation
policy, and challenge procedure before registration.

## Upgrade constraint: separate from boundary integrations

The validator currently pins Cosmos SDK `v0.50.15` and IBC-Go `v8.8.0`.
The current [Cosmos release-family guidance](https://docs.cosmos.network/sdk/latest/release-family)
lists SDK `0.54.x` with IBC-Go `v11.x` and marks SDK `0.50.x` and lower
end-of-life. Treat that as a high-priority, coordinated consensus-upgrade
program—not as a dependency bump bundled with these read-only standards seams.
It needs module/store migrations, relayer and counterparty compatibility
testing, a rehearsal network, and an explicit activation height.

## Implemented: generated TypeScript transaction SDK

`sdk/typescript` builds `@zerone-chain/sdk`, a versioned ESM package generated
from Zerone's pinned protobuf sources with
[Telescope](https://github.com/hyperweb-io/telescope). It provides:

- typed protobuf/direct-signing codecs for all 165 request messages in
  Zerone's 20 `Msg` services;
- a registry that composes with CosmJS's standard Cosmos message types;
- generic CAIP-2 and CAIP-10 parsing plus the Cosmos chain-reference profile;
- an explicit Zerone network descriptor and checksum-aware `zrn` account IDs;
  and
- a separate validator for the opaque `did:zrn` labels accepted by `x/auth`.

Generation is reproducible from local protobuf inputs. CI regenerates the
client, checks source and output digests, rejects tracked or untracked drift,
tests the published package under TypeScript's NodeNext resolver, and compares
representative Go and TypeScript protobuf wire vectors.

The dashboard uses the SDK's identity helpers and, when the node supports it,
cross-checks them against the read-only `AccountIdentifier` query. Query
unavailability remains compatible with the currently deployed binary, but a
successful response that disagrees with the configured network or wallet is
rejected. Standard bank sends continue through CosmJS's standard registry;
the much larger Zerone codec registry is reserved for future custom controls.

Generated codecs are serialization tools, not authority policy. No custom
mainnet transaction control was enabled, and the SDK does not claim legacy
Amino support for Zerone messages.

## Implemented: Zerone CIDv1 client preflight

The Go CLI and `@zerone-chain/sdk` can parse new `x/home` memory references
with the official Go CID implementation and `js-multiformats`. Zerone chooses
lowercase base32 as its one accepted client representation for CIDv1, although
CID itself permits other multibase strings for the same identifier. The CLI
enforces this automatically before signing `MsgUpdateMemoryCID`; SDK callers
must opt into `asZeroneMemoryCid`, because the raw generated message codec still
accepts a string. The helper also exposes the parsed codec, multihash code, and
digest length so applications can apply an artifact-specific policy.

This is deliberately not a consensus change. Existing validators still accept
the bounded opaque strings they accepted before, and historical values are not
rewritten. Parsing proves only that text structurally encodes a content address;
applications must independently hash the referenced bytes and apply their
codec, multihash, authenticity, privacy, and availability policies. Zerone has
not yet selected an on-chain allowlist. Consensus enforcement requires a
legacy-state audit, an explicit policy, and a coordinated upgrade.

## Ranked integration roadmap

### 1. Chain Registry metadata and broader wallet adapters

The first generated client uses
[CosmJS](https://github.com/cosmos/cosmjs) and Keplr-compatible signing.
Evaluate [Interchain Kit](https://docs.hyperweb.io/interchain-kit/) only when
its beta status and additional wallet surface fit the dashboard's needs.

Publish Zerone `chain.json` and `assetlist.json` to the
[Cosmos Chain Registry](https://github.com/cosmos/chain-registry) only after
the live chain ID, genesis, public endpoints, denomination, and BIP-44/SLIP-44
coin type are final. Do not invent the coin type to accelerate wallet listing.

### 2. Sponsored-gas UX now; delegated authority only after an ante audit

[`x/feegrant`](https://docs.cosmos.network/sdk/v0.50/build/modules/feegrant/README)
is already wired. Expose its allowance queries, grant/revoke flows, and
fee-granter signing helpers for sponsor-funded onboarding.

Do not wire Cosmos SDK
[`x/authz`](https://docs.cosmos.network/sdk/v0.50/build/modules/authz/README)
yet. Zerone's custom ante checks inspect outer messages and can miss privileged
messages nested inside `MsgExec`; authz also overlaps the slim-cut decision to
keep delegated authority in AgentTool. It requires recursive message
inspection, handler-level invariants, a store migration, and a separate audit.

### 3. x402 payment adapter in AgentTool, outside consensus

[x402](https://github.com/x402-foundation/x402) is a strong fit for agent/tool
commerce because its network identifiers use CAIP-2 and its transport/scheme
layers are extensible. A Zerone adapter belongs in AgentTool, where the slim
cut moved payment rails. It must bind payment authorization to chain ID,
resource and method, amount and denomination, payee, nonce, and expiry, with
persistent replay protection and finality checks.

Do not claim compatibility with EVM/Solana facilitators, and do not describe
delayed channel settlement as the x402 `exact` scheme unless it satisfies that
scheme's semantics.

### 4. ERC-8004-inspired agent profile

[ERC-8004](https://eips.ethereum.org/EIPS/eip-8004) is a useful vocabulary for
identity, reputation, and validation, but it is a Draft ERC built around
Ethereum standards. A Zerone compatibility profile can map:

- identity to `x/auth` plus CAIP-10;
- reputation to the read-only `x/trust_score` synthesis; and
- validation to `x/qualification`, challenges, and provenance evidence.

Keep Zerone agent identity non-transferable. Do not copy ERC-721 identity
transfer semantics or claim ERC-8004 compliance without a separately specified
adapter and test vectors.

### 5. Additional content-addressed provenance documents

Build on the read-only in-toto projection and the implemented CIDv1 client
preflight without adding rich-document parsers to consensus:

- select codec/multihash policies for canonical artifact references and CAR
  bundles before any validator-side CID enforcement;
- project AI/dataset licensing and relationships as
  [SPDX 3](https://spdx.github.io/spdx-spec/v3.0.1/);
- attach [C2PA](https://spec.c2pa.org/specifications/specifications/2.4/specs/C2PA_Specification.html)
  manifests where media provenance applies; and
- anchor only the deterministic digest, schema/version, media type, size, and
  availability references on-chain.

A CID checked against independently retrieved bytes can detect a mismatch under
the selected multihash's assumptions; it does not establish availability or
authenticity. Rich JSON-LD, C2PA, VC, and policy validation belongs outside
consensus.

### 6. A2A Agent Cards for service discovery

The Linux Foundation's [A2A protocol](https://github.com/a2aproject/A2A) fits
Zerone's agent identity and home model at the service boundary: an Agent Card
can advertise skills, modalities, and endpoints while a CAIP-10 account ID
names the chain identity behind it.

Host and validate cards in AgentTool or a resolver sidecar. Commit only a
bounded URI, deterministic digest, schema version, status, and owning account
reference. A card advertises capabilities; it does not by itself prove control
of a Zerone identity, authorize a transaction, or justify exposing private home
state.

## Explicit deferrals

- No W3C DID document or Verifiable Credential claim until `did:zrn`
  canonicalization, identity-key proof of possession, rotation signatures,
  metadata bounds, replay protection, and genesis invariants are hardened.
- No CosmWasm or general contract VM; it conflicts with the slim-cut boundary.
- No new IBC middleware while the current IBC/ICA posture remains limited and
  the IBC-Go v8 migration plan remains unresolved. Cosmos SDK 0.50 / IBC-Go 8
  is outside the currently supported release families, so upgrade planning is
  a maintenance prerequisite rather than a hidden feature dependency.
- No on-chain x402 facilitator, ERC-8004 registry, C2PA parser, JSON-LD
  resolver, A2A parser, or remote-context fetch.
- No reward-bearing `sigstore-in-toto-v1` registration until governance has
  approved a reproducible compiler, pinned verification policy, retained
  bundles, and an independent challenge procedure.

The design rule is simple: reuse open identifiers and document formats at
Zerone's edges; keep deterministic consensus schemas small.
