# Open-source crypto SDK and standards review

- Date: 2026-07-28
- Checkout reviewed: Cosmos SDK `v0.50.15`, IBC-Go `v8.8.0`, Go `1.24`

## Outcome

The first integration is an additive interoperability projection:

- official in-toto Attestation `v1.2.0` Go SDK;
- in-toto Statement v1 export from `x/training_provenance`;
- a Cosmos-profile CAIP-2 source label inside the predicate;
- native CLI commands for the certificate and standard Statement; and
- the previously missing module-basic registration for the read-only
  provenance and trust-score synthesizers.

This slice changes no persistent state, transaction validity, economics,
governance, identity lifecycle, or IBC behavior. It can be removed without a
store migration. It does not create a signed in-toto Envelope.

## Fit review

| Candidate | Maturity / license | Zerone fit | Decision |
|---|---|---|---|
| [in-toto Attestation v1.2](https://github.com/in-toto/attestation) | released; Apache-2.0; official Go binding supports Go 1.23+ | official Go types for a Statement v1 carrying a Zerone-defined training-provenance predicate | integrated now |
| [CAIP-2](https://standards.chainagnostic.org/CAIPs/caip-2) | Final core; Cosmos profile Draft; CC0 | explicit source-network label for relying-party policy | integrated in the predicate |
| [CAIP-10](https://standards.chainagnostic.org/CAIPs/caip-10) | Final core; Cosmos profile Draft | wallet/indexer identifier for `zrn1...` accounts, distinct from `did:zrn` | next read-only identity projection |
| [CID / multiformats](https://github.com/multiformats/cid) | established; `go-cid` MIT | real validation and canonical comparison for agent-home memory references | next as client preflight; consensus enforcement needs an upgrade and legacy audit |
| [DID Core 1.0](https://www.w3.org/TR/did-1.0/) | W3C Recommendation | strongest semantic match for sovereign, non-transferable agent identity | defer until identity proof and lifecycle defects below are fixed |
| [VC Data Model 2.0](https://www.w3.org/TR/vc-data-model-2.0/) + [Data Integrity](https://www.w3.org/TR/vc-data-integrity/) | W3C Recommendations | signed exports of qualifications and provenance, without storing personal credentials on chain | after the DID assurance work |
| Cosmos SDK [`x/authz`](https://github.com/cosmos/cosmos-sdk/blob/v0.50.15/x/authz/README.md) | Apache-2.0; available in the current but EOL SDK line | narrow consent grants could fit agent actions | do not reintroduce delegated authority casually; reconsider after the SDK upgrade and slim-cut doctrine review |
| Cosmos SDK `x/circuit` | Apache-2.0 | standard breaker semantics | only as a migration that consolidates existing emergency/circuit controls, never as a second pause plane |
| Cosmos SDK [`x/epochs`](https://docs.cosmos.network/sdk/latest/modules/epochs/README) | Apache-2.0; active in SDK 0.53+ | could consolidate custom decay and epoch clocks | evaluate after the SDK upgrade |
| [UCAN](https://github.com/ucan-wg/spec) | specification says 1.0; latest tagged release remains RC; code/examples MIT | excellent attenuated-capability semantics for sovereign agents | prototype in agenttool, not chain consensus |
| [SLSA 1.2](https://slsa.dev/spec/v1.2/) | approved specification | build and training-pipeline artifact provenance | use in CI/adapters; do not label Zerone's current custom predicate as SLSA |
| [C2PA 2.4](https://spec.c2pa.org/specifications/specifications/2.4/specs/C2PA_Specification.html) | published; CC BY 4.0 | media/document provenance | validate off chain and anchor results only |
| [ERC-8004](https://eips.ethereum.org/EIPS/eip-8004) | Draft; CC0; EVM/ERC-721-specific | emerging agent registry vocabulary | monitor; transferable agent identity conflicts with Zerone's distinct-being model |
| CosmWasm / wasmd | Apache-2.0; large runtime surface | flexible contracts | defer until a concrete workload cannot be served by native modules |

## Why in-toto first

`x/training_provenance` already composes knowledge manifests, qualification
coverage, privileged actions, incidents, and capture resolutions. in-toto adds
a recognized Statement layer without pretending that Zerone implements SLSA,
without introducing a new truth oracle, and without copying data into another
store.

The subject digest is the manifest commitment. The predicate is a live
certificate snapshot. CAIP-2 labels the network claimed by the queried node and
lets consumers reject an unexpected source; it does not cryptographically bind
the digest to a chain. Origin assurance requires a trusted query or state proof
plus, when authentication is required, a signed Envelope and signer policy.

The output remains unsigned. Calling an unsigned Statement an authenticated
attestation would overstate its assurance. DSSE construction and signer-policy
evaluation belong at the off-chain producer/registry boundary.

## Integration boundaries

- The predicate TypeURI is versioned, but its `zerone.money` human-readable
  route currently returns 404. in-toto permits this, but recommends resolution.
  Publish the checked-in v1 definition there before public envelope issuance.
- No provenance/trust-score REST route is mounted in this change. Certificate
  synthesis walks manifest facts and global audit streams without query gas,
  pagination, or an index. Public HTTP exposure needs scale tests, indexing,
  operator rate limits, and abuse budgets first.
- A finalized manifest root is normally fixed, but Zerone has an incident-bound
  correction path. A corrected root is a distinct subject digest; consumers
  must retain block height and correction history.
- CAIP-2 and an unsigned Statement are metadata, not proof of RPC honesty,
  canonical-chain membership, or signer authorization.
- `protojson` output is not promised as canonical across versions. DSSE signs
  its pre-authentication encoding over the Envelope's exact payload type and
  payload, so verifiers must not reserialize JSON for signature checks.

The custom predicate uses narrower signal names than the native certificate:

- included-fact privileged actions only match an action whose target is an
  included fact ID;
- the incident count is knowledge-module-wide, not manifest-specific; and
- cartel resolutions are counted across covered domains.

Those scopes are part of v1 and prevent a generic field name from implying
more precise correlation than the synthesizer currently computes.

## Existing blockers discovered

These are reasons not to claim broader standards compliance yet:

1. `did:zrn` registration checks that the identifier matches a supplied
   Ed25519 key, but does not prove possession of that identity private key.
2. Key rotation requires a non-empty `authorization_signature` but does not
   cryptographically verify it.
3. Both 32- and 64-hex DID forms are accepted while helper and onboarding paths
   derive different lengths, and no deactivation method is specified.
4. Home sessions are not bound to the registered key cited by the transaction,
   and home permissions/spending limits are not consumed by the ante path.
5. The transfer IBC route does not wrap the already-created ICS-29 fee
   middleware, so packet-fee negotiation is not actually active.
6. CID fields currently receive only non-empty/length checks.

A DID document or Verifiable Credential export before items 1–3 are fixed
would imply controller/authentication assurance that the chain does not yet
establish. Enforcing CID parsing in a message handler would change transaction
validity and therefore requires a coordinated upgrade plus an audit of legacy
values.

## Dependency support boundary

The current Cosmos release-family page lists SDK `0.50` as EOL. IBC-Go `v8` is
absent from the currently maintained IBC release lines, which list v10 and v11:

- [Cosmos SDK security policy](https://docs.cosmos.network/sdk/v0.53/security/security-policy)
- [Cosmos SDK release families](https://docs.cosmos.network/sdk/latest/release-family)
- [IBC-Go supported releases](https://github.com/cosmos/ibc-go)

Therefore, importing more consensus modules on the current family is not a
maintained long-term integration even if the code compiles. The next
consensus-facing standards work should begin with a coordinated SDK/IBC
migration. That migration must account for capability removal and changed
ICS-29 wiring; it is not a dependency-only bump.

## Recommended sequence

1. Publish the predicate TypeURI, keep the in-toto/CAIP-2 projection query-only,
   and obtain external consumer feedback.
2. Upgrade the core SDK/IBC family with state, packet, and relayer migration
   tests.
3. Repair identity proof-of-possession, canonical identifier migration,
   rotation authorization, and deactivation.
4. Add read-only CAIP-10 and DID document resolution, then signed VC exports.
5. Add `go-cid` client preflight; adopt a codec/multihash policy before any
   consensus enforcement.
6. If on-chain delegated consent is deliberately restored, use narrow custom
   `x/authz` authorizations with resource, action, use, spend, and expiry
   limits. Never use unrestricted `GenericAuthorization` as agent consent.
7. Complete or migrate interchain fee/rate-limit middleware only on a supported
   IBC family, while external channels remain protocol-gated.

## Verification added

- exact in-toto Statement v1 type and basic structure checks;
- lowercase SHA-256 hex encoding and length checks, not root recomputation;
- Cosmos-profile CAIP-2 direct and hashed official vectors;
- normalized domain ordering and duplicate-domain rejection;
- exact `uint64` preservation as decimal strings;
- exportable status and trust-grade validation;
- cross-stack native-certificate to in-toto projection; and
- default-genesis coverage for both read-only synthesizer modules.
