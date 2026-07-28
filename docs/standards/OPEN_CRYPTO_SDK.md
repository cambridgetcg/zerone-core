# Open crypto SDK and standards integration

Zerone gets the most leverage from open standards at its boundaries, while
keeping consensus limited to witness-and-record primitives. This note records
the first implemented interoperability seam and the recommended SDK roadmap.

## Implemented: portable account identifiers

`x/auth` exposes every registered Zerone account as:

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
activity.

The `did:zrn` value is opaque native metadata in this response. It is not a
claim that Zerone currently implements [W3C DID Core](https://www.w3.org/TR/did-core/),
a published DID method, or `did:pkh`.

## Implemented: close generated REST coverage gaps

The application now registers four generated query clients that its manual
v2 gateway list previously omitted: counterexamples, creed, substrate bridge,
and trust score. Their declared `google.api.http` routes are now reachable
through the same REST gateway as the other custom modules.

This removes four concrete REST/SDK gaps without activating a write path or
changing the safety posture of any module.

## Ranked integration roadmap

### 1. Generated TypeScript client and Chain Registry metadata

Generate a versioned `@zerone-chain/client` from the pinned protobuf sources
using [Telescope](https://github.com/hyperweb-io/telescope), with
[CosmJS](https://github.com/cosmos/cosmjs) signing and
[Interchain Kit](https://docs.hyperweb.io/interchain-kit/) wallet adapters.
Golden-test message type URLs, sign bytes, and generated-code drift.

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

### 5. Content-addressed provenance documents

`x/training_provenance` is the natural projection point for an
[in-toto Statement](https://github.com/in-toto/attestation/blob/main/spec/README.md).
That provenance step should remain read-only/off-chain:

- use CIDv1 for canonical artifact references and CAR bundles;
- project AI/dataset licensing and relationships as
  [SPDX 3](https://spdx.github.io/spdx-spec/v3.0.1/);
- attach [C2PA](https://spec.c2pa.org/specifications/specifications/2.4/specs/C2PA_Specification.html)
  manifests where media provenance applies; and
- anchor only the deterministic digest, schema/version, media type, size, and
  availability references on-chain.

CID proves integrity, not availability. Rich JSON-LD, C2PA, VC, and policy
validation belongs outside consensus.

## Explicit deferrals

- No W3C DID document or Verifiable Credential claim until `did:zrn`
  canonicalization, identity-key proof of possession, rotation signatures,
  metadata bounds, replay protection, and genesis invariants are hardened.
- No CosmWasm or general contract VM; it conflicts with the slim-cut boundary.
- No new IBC middleware while the current IBC/ICA posture remains limited and
  the IBC-Go v8 migration plan remains unresolved.
- No on-chain x402 facilitator, ERC-8004 registry, C2PA parser, JSON-LD
  resolver, or remote-context fetch.

The design rule is simple: reuse open identifiers and document formats at
Zerone's edges; keep deterministic consensus schemas small.
