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
records those exact boundaries and freezes them. The emitted Predicate TypeURI
is pinned to the exact repository revision carrying that specification rather
than a mutable branch; a semantic change requires a new predicate version and
URI.

This is an unsigned, current-state projection with no store, transaction,
reward, or consensus-version change. A producer can sign the returned JSON
off-chain without making validators parse signatures or remote documents.
Missing manifests return `NotFound`; draft or composed manifests return
`FailedPrecondition`; malformed sealed state returns `DataLoss`; and server
configuration failures return `Internal`. Public REST operators should
rate-limit this live synthesis and may cache by `(chain ID, block height,
manifest ID)`, because certificate construction scans global audit records.

The TypeScript SDK also exposes
`parseUnsignedZeroneInTotoStatement` as a strict offline consumer for the
exact v1 profile. Callers must pin the expected manifest and serving chain.
The parser bounds the JSON and every variable collection, preserves protobuf
`uint64` fields as decimal strings, checks the subject/certificate/source
invariants, and returns an explicit unsigned/unverified assurance marker. It
does not fetch the query, dereference the predicate URI, discover trust roots,
verify signatures, or authenticate the predicate.

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
- a strict offline consumer for Zerone's unsigned in-toto provenance profile;
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

Generated codecs are serialization tools, not authority policy. No Zerone
custom-message control was enabled, and the SDK does not claim legacy Amino
support for Zerone messages. The dashboard's only additional write controls
use the already-wired standard Cosmos SDK feegrant module.

## Implemented: static adapter capability index

The dashboard publishes the custom
[`zerone.adapter-index/v1`](../specs/adapter-index-v1.md) inventory at
`/standards/adapter-index.v1.json`. It is version-controlled refusal and
source-status metadata, not a live registry or discovery protocol. Validation
keeps every release-level consensus, network, activation, and live-deployment
assertion false.

The required A2A and x402 entries are `planned` and `unavailable`, advertise no
capabilities, and contain no endpoint field. Existing in-toto, Sigstore, and
AgentTool seams are labelled according to their narrower source, local-tool,
or external-service boundaries. Serving the file performs no upstream request
and does not make the dashboard proxy authoritative.

## Implemented reads; activation-gated Cosmos SDK fee sponsorship

The dashboard exposes the indexed grantee allowance query, exact-pair lookup,
and exact known-grantee revoke. It deliberately does not expose the SDK's
granter-wide `issued` query: that query filters by scanning the full feegrant
store. Pagination limits returned matches, not scan work, so arbitrary public
queries become a validator resource risk as the store grows. The deployed
validator currently exposes that SDK route directly on its REST listener;
activation therefore also requires restricting or rate-limiting direct REST
access, or replacing the scan with an indexed query.

Bounded grant creation and sponsored-bank-send code is prepared but
`FEEGRANT_SPONSORSHIP_ENABLED` remains `false` until a hardened validator
binary is deployed and the direct REST exposure above is addressed. Prepared
grant creation is restricted to an
`AllowedMsgAllowance` wrapping a `BasicAllowance`, with:

- a maximum 100 ZRN aggregate fee cap;
- a dashboard expiry from one through 30 days (the constructor's hard lower
  bound is one minute);
- an explicit allowlist limited to bank sends and claiming-pot claims; and
- a different, checksum-valid `zrn` grantee.

When activated, the dashboard refuses unknown allowance types for spending,
re-queries the exact granter/grantee pair before every sponsored send, and
requires the grantee's own Keplr signature. Sponsor selection is never
automatic. Grant and revoke transactions pay nonzero fees from the connected
wallet. The edge exposes only exact-pair and explicitly limited, indexed
grantee queries.

Review found that the existing signer-only freeze check did not cover a fee
granter, because the granter need not sign the sponsored transaction. The app
now includes a pre-deduction ante guard: if a registered Zerone fee granter is
frozen, the transaction fails before the allowance or balance changes. A full
app test proves freeze, failed use, preserved balance/allowance, unfreeze, and
restored use. This guard adds no store or migration, but it changes transaction
validation and therefore requires an explicit validator binary rollout before
the dashboard flag may be enabled.

CosmJS's standard registry supplies the feegrant protobuf codecs, and SDK tests
prevent Zerone registry composition from dropping them. Grant/revoke also
require a protobuf-direct signer because CosmJS 0.39 has no feegrant Amino
converter.

## Prepared: Cosmos Chain Registry metadata

`integrations/chain-registry/zerone/` contains a schema-valid `chain.json`,
`assetlist.json`, and ZRN SVG submitted as
[cosmos/chain-registry#7859](https://github.com/cosmos/chain-registry/pull/7859).
Local checks cross-reference the committed genesis and deployment
configuration; CI also runs the pinned upstream Chain Registry validator.

The candidate advertises only the complete public CometBFT RPC and verified
peer data. It deliberately omits general REST/gRPC, IBC channels, snapshots,
binaries, and a recommended release because those claims are not currently
supported by the deployed evidence. The configured SLIP-0044 value `118` is
described as the shared Cosmos derivation path, not a Zerone-owned
registration. Upstream completion also depends on registering the `zrn`
Bech32 HRP in SLIP-0173.

## Implemented: Zerone CIDv1 client preflight

The Go CLI and `@zerone-chain/sdk` can parse new `x/home` memory references
with the official Go CID implementation and `js-multiformats`, following the
[maintained CID specification](https://specs.ipfs.tech/cid/). Zerone chooses
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

Track the submitted
[Cosmos Chain Registry proposal](https://github.com/cosmos/chain-registry/pull/7859)
and the `zrn`
[SLIP-0173 proposal](https://github.com/satoshilabs/slips/pull/2039) through
review. Keep the metadata evidence-limited as public endpoints and release
posture evolve; do not present shared coin type `118` as a Zerone-specific
allocation.

### 2. Delegated authority only after an ante audit

[`x/feegrant`](https://docs.cosmos.network/sdk/v0.50/build/modules/feegrant/README)
is already wired. Zerone's runtime CLI already exposes allowance queries plus
grant, revoke, and prune transactions. The TypeScript SDK now builds a finite
`BasicAllowance` wrapped by `AllowedMsgAllowance`, a revoke message, and the
fee-granter field used by CosmJS for sponsor-funded onboarding. The builder
requires an explicit positive budget, future expiry, and exact allowed message
type URLs. Its reviewed onboarding policy currently allowlists only
`/zerone.claiming_pot.v1.MsgClaim`; all other messages, including nested
authorization envelopes, fail closed. These are client guardrails, not added
consensus restrictions. The dashboard's separately prepared grant and
sponsored-spend paths must remain disabled until the reviewed fee-granter ante
guard is deployed and verified, and the unindexed direct REST `issued` route is
restricted or rate-limited.

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

## Prepared: fail-closed authentication policy

Zerone's post-auth DID, frozen-account, and capability decorators now derive
authenticated addresses from `SigVerifiableTx.GetSigners()` and propagate
extraction errors. They no longer infer the signer from the optional
`SignerInfo.public_key` field. A real TxRaw regression fixture covers a stored
account key with that wire field omitted.

This transaction-validity change is part of the single guarded
`sdk-0.53-ibc-10` plan and must activate at one coordinated binary height,
not as a mixed-validator rolling deployment. The retired
`auth-ante-hardening-v1` name has no handler, lineage entry, or store loader,
so it cannot bypass the SDK/IBC source-version and legacy-fee-balance guards.
The unified handler writes its auth-hardening marker only after those guards
and module migrations succeed. Cosmos SDK 0.50.15 itself panics in its v2
signing adapter on the omitted-key wire shape; Cosmos SDK 0.53.8 contains the
upstream
[nil-key adapter fix](https://github.com/cosmos/cosmos-sdk/commit/a91a822eb2339d563bbe8c7bc61d71fa6c6c60e2).
Dependency migration and this app-level policy regression both have to pass
before the upgrade is scheduled.

The unified handler also repairs an
[IBC-Go v10.7.0 channel-migration defect](https://github.com/cosmos/ibc-go/blob/v10.7.0/modules/core/04-channel/migrations/v10/store.go#L111-L123).
That upstream migration calls exact-key deletion on `channelUpgrades` and
`pruningSequenceStart`, while the v8 records are child keys below
`channelUpgrades/…` and `pruningSequenceStart/…`. After module migrations
succeed, Zerone enumerates, closes the iterators, and deletes the real child
keys; any iterator or deletion error aborts before the auth-hardening marker.
The repair deliberately preserves `recvStartSequence/…` byte-for-byte because
IBC-Go v10 still uses it for replay protection, and likewise preserves packet
commitments, acknowledgements, and receipts. The old-database rehearsal must
assert both sides of that boundary: obsolete child prefixes gone, live replay
and packet state retained.

This repair does not make the current `did:zrn` lifecycle trustworthy.
Registration still lacks identity-key proof of possession, rotation does not
verify its authorization signature, legacy identifier aliases are ambiguous,
and terminal deactivation is absent. Those remain a separate state/protobuf
migration after a live-state census. The read-only
[`identity-census`](../../tools/identity-census/README.md) tool audits a
same-height application export for aliases, mapping inconsistencies, malformed
keys, and Cosmos account-key/address mismatches. Its report must be retained
with the export height and app hash; a clean report is not proof of private-key
possession.

## Explicit deferrals

- No W3C DID document or Verifiable Credential claim until `did:zrn`
  canonicalization, identity-key proof of possession, rotation signatures,
  metadata bounds, replay protection, and genesis invariants are hardened.
- No CosmWasm or general contract VM; it conflicts with the slim-cut boundary.
- No new IBC middleware while the current IBC/ICA posture remains limited.
  The Cosmos SDK 0.53 / IBC-Go 10 prototype is not deployment-ready until one
  coordinated upgrade plan includes the ante marker and passes a real
  pre-upgrade state census, ICS29/channel safety checks, old-database restart
  rehearsal, and in-flight packet tests. Cosmos SDK 0.50 / IBC-Go 8 is outside
  the currently supported release families, so this maintenance migration is a
  prerequisite rather than a hidden feature dependency.
- No on-chain x402 facilitator, ERC-8004 registry, C2PA parser, JSON-LD
  resolver, A2A parser, or remote-context fetch.
- No reward-bearing `sigstore-in-toto-v1` registration until governance has
  approved a reproducible compiler, pinned verification policy, retained
  bundles, and an independent challenge procedure.

The design rule is simple: reuse open identifiers and document formats at
Zerone's edges; keep deterministic consensus schemas small.
