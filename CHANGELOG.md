# Changelog

Notable source changes are recorded here. Network activation and package
publication are separate events and are stated explicitly when they occur.

## Unreleased — 2026-08-11 branch-flow shadow architecture

### Added

- An accepted
  [constructive-intelligence branch-flow source architecture](docs/specs/constructive-intelligence-branch-flow-v1.md)
  for one conserved, already funded role envelope: 60% direct work, 10%
  load-bearing ancestors, 30% independently evidenced descendants, and 0%
  base commons, with absolute `q=0.5` geometric depth through depth five and a
  terminal tail. E5 and E6 remain inside their existing outcome-pool tranches,
  `breakthrough` remains retrospective and creates no separate prize, and
  outcome and TC6 realized-training-revenue flows require separate adapters,
  ledgers, and receipt keys.
- A standard-library-only
  [exact-integer shadow kernel](tools/constructive-rewards/branchflow/)
  for deterministic allocation,
  bounded artifact-graph validation, replay/cap checks, and fail-closed result
  projection. The package is non-custodial and exposes no bank, store,
  governance, signer, clock, network, or vesting capability. Golden JSON
  fixtures, unit tests, and conservation/permutation fuzz targets lock
  conservation, absolute-depth, replay, cap, bound, ordering, and
  malformed-input behavior. A
  `-mode branch-flow` CLI renders the golden-locked E5 reference projection
  under the digest-pinned policy while preserving the same no-effect fences.
- A strictly validated, machine-readable Branch Flow v1 profile and a
  digest-pinned, read-only dashboard view publish the reference split, depth
  gradient, authority walls, and closed release gates without exposing a
  transaction or activation path.

### Changed

- The constructive-intelligence reward design now names Cosmos SDK `x/gov` as
  the sole target ordinary binding proposal, tally, authorization, and
  execution path. Current source still has dual custom/SDK authority surfaces;
  this design does not claim that migration or activation has occurred.
  Epistemic review supplies typed evidence to the target path; it is not a
  second binding chamber and owns no veto, treasury key, or executor.

### Consensus activation

The branch-flow specification and exact reference kernel are shadow source
artifacts with `assurance = SHADOW_ONLY` and `economic_effect = NONE`. They do
not change chain state, move or reserve funds, create a payout entitlement,
install a second authority, satisfy the independent-implementation gate, or
activate any network behavior. All value-bearing release gates remain closed.

### Publication status

The source architecture, reference kernel, static standard, and dashboard view
are source/web inspection artifacts only. Publishing those exact static bytes
does not publish a package registry release, validator binary, chain upgrade,
escrow, entitlement, or economic deployment, and activates no network reward
behavior.

## Unreleased — 2026-07-29 consolidation

### Added

- A digest-pinned Relational Topology v0 standard and accessible dashboard map
  for typed authority, economic, emergency, evidence, identity, recognition,
  and reference flows. It names six known source conflicts and enforces five
  same-flow non-domination guards while remaining a non-authoritative static
  projection with no consensus, state, identity, qualification, reward, or
  migration effect.
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
- The dashboard lockfile advances `nanoid` to 3.3.18 to remove the
  zero-size custom-generator denial-of-service advisory.
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
