# Changelog

Notable source changes are recorded here. Network activation and package
publication are separate events and are stated explicitly when they occur.

## Unreleased — 2026-07-29 consolidation

### Added

- An accepted, hash-pinned authoritative-state design that makes SDK staking
  the sole long-lived stake ledger, SDK governance the sole ordinary executor,
  and ontology the sole canonical domain registry. It is source design only;
  no handler, electorate implementation or selection, migration, or network
  activation is included.
- Provisional knowledge conjectures with claim-type-safe lifecycle handling.
- K-alpha recognition events and bounded aggregation/settlement guards.
- A cursor-bounded knowledge probe heartbeat.
- CAIP account-identifier projections, unsigned in-toto training provenance,
  and an isolated Sigstore-to-substrate evidence compiler.
- A repository TypeScript SDK covering 169 request message types across 20
  Zerone `Msg` services.
- A fail-closed `zerone-2` ceremony, authority, runtime, query-gateway, and
  cutover kit.

### Changed

- The TypeScript SDK bounds account-address decoding to BIP-173's 90-character
  limit, preserving packed-consumer compatibility with CosmJS dependencies.
- Starved knowledge challenges settle without leaving facts locked and restore
  conjectures to their type-appropriate status.
- A manual, approval-environment SDK release workflow now requires an
  annotated `sdk-v<version>` tag at current `main`, rebuilds and verifies the
  package, and publishes the one packed tarball with provenance. The workflow
  does not itself grant npm scope authority or publish unless explicitly run.
- Governance vote weights remain based on bonded stake; permissionless funding
  correlations remain observational and do not reduce another wallet's vote.
- Substrate axis projections have a protocol-wide ceiling, and settlement
  clamps legacy over-ceiling values.
- The misleading `lineage_royalty_paid` event is replaced by
  `lineage_royalty_accrued`; lineage propagation records accounting only and
  does not transfer coins to upstream authors.
- Falsification clawback requires an adjudicated verdict.
- Genesis/state validation and protobuf ownership handling are hardened.
- Manual API inventories now defer to the generated Swagger document, which
  contains 217 paths and 446 definitions across the current application.

### Consensus activation

The knowledge, vesting, and substrate changes are
consensus-visible. Existing networks require the coordinated
`consolidation-safety-v1` upgrade with an agreed height and matching validator
binaries. Source publication alone does not activate this behavior.

### Deferred

- The incomplete same-model training-fund replay change.
- The older incomplete and unsafe capture-recovery behavior.
- Funding-correlation vote decay until relationships cannot be dust-poisoned
  by an untrusted sender.
- Cosmos SDK/IBC-Go family migration, general authz, reward-bearing Sigstore
  registration, and rich remote-document parsing.
- Production component signing: current CI neither requests GitHub OIDC nor
  emits a Sigstore bundle, and the authority verifier currently validates the
  declared bundle structure rather than performing Fulcio/Rekor-backed
  cryptographic verification. Validator/image deployment remains NO-GO until
  both production signing and trusted-root verification exist.

### Publication status

This consolidation is intended for the canonical GitHub source repository
only. It does **not** publish `@zerone-chain/sdk` to npm, create a release tag
or binary release, deploy a validator, or synchronize the intentionally
divergent Codeberg history.

The `zerone-2` release remains **NO-GO** until the signed release packet,
ceremony evidence, immutable artifact references, phase decisions, initiation
evidence, and complete authority chain satisfy the release kit.
