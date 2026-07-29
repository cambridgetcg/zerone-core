# Changelog

Notable source changes are recorded here. Network activation and package
publication are separate events and are stated explicitly when they occur.

## Unreleased — 2026-07-29 consolidation

### Added

- Provisional knowledge conjectures with claim-type-safe lifecycle handling.
- K-alpha recognition events and bounded aggregation/settlement guards.
- A cursor-bounded knowledge probe heartbeat.
- CAIP account-identifier projections, unsigned in-toto training provenance,
  and an isolated Sigstore-to-substrate evidence compiler.
- A repository TypeScript SDK covering 166 request message types across 20
  Zerone `Msg` services.
- A fail-closed `zerone-2` ceremony, authority, runtime, query-gateway, and
  cutover kit.

### Changed

- Starved knowledge challenges settle without leaving facts locked and restore
  conjectures to their type-appropriate status.
- Governance applies funding-correlation decay to effective vote weight.
- Substrate axis projections have a protocol-wide ceiling, and settlement
  clamps legacy over-ceiling values.
- Falsification clawback requires an adjudicated verdict.
- Genesis/state validation and protobuf ownership handling are hardened.
- Manual API inventories now defer to the generated Swagger document, which
  contains 214 paths and 438 definitions across the current application.

### Consensus activation

The knowledge, governance, vesting, and substrate changes are
consensus-visible. Existing networks require the coordinated
`consolidation-safety-v1` upgrade with an agreed height and matching validator
binaries. Source publication alone does not activate this behavior.

### Deferred

- The incomplete same-model training-fund replay change.
- The older incomplete and unsafe capture-recovery behavior.
- Cosmos SDK/IBC-Go family migration, general authz, reward-bearing Sigstore
  registration, and rich remote-document parsing.

### Publication status

This consolidation is intended for the canonical GitHub source repository
only. It does **not** publish `@zerone-chain/sdk` to npm, create a release tag
or binary release, deploy a validator, or synchronize the intentionally
divergent Codeberg history.

The `zerone-2` release remains **NO-GO** until the signed release packet,
ceremony evidence, immutable artifact references, phase decisions, initiation
evidence, and complete authority chain satisfy the release kit.
