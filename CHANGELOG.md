# Changelog

Notable source changes are recorded here. Network activation and package
publication are separate events and are stated explicitly when they occur.

## Draft source — consensus-repairs-v2 (not scheduled)

- The successor to consensus-repairs-v1: Q1 not-observed neutrality, Q2
  inconclusive-round accounting, Q5 calibration smoothing and PR37's zero
  confidence floor, plus FX7/FX8 genesis export repairs, in one all-or-nothing
  carrier with bounded censuses and a receipt. See
  [docs/consensus-repairs-v2.md](docs/consensus-repairs-v2.md).
- It now also carries #122's alignment bank-supply denominator (review finding
  F1, 2026-11-06), moved from v1 so that it switches only at Q1's height:
  vesting_rewards 3→4, five markers, and native genesis selects it with Q1.
- Its `q1_health` census states the pacing shift the height causes (review
  finding F5): category, creation and analysis pacing, claim cooldown, review
  fee, risk-analysis interval and the expedited voting period, before and
  after. On zerone-dev-1's measured state that is degraded to `not_observed`,
  pacing 750,000/1,500,000 to 1,000,000/1,000,000, base fees again, and no more
  halved parameter-LIP voting.
- Carries main after #167; cut binaries only from a source the
  proposal-coherence gate reports `COHERENT`, never from `7253100b`.

## Draft source — consensus-repairs-v1 (not scheduled)

- Adds the all-or-nothing successor to claim-integrity: ten guarded version
  steps, nine explicit markers, bounded read-only censuses, exact store-diff
  verification and a marker-readback receipt. No legacy balance is redistributed.
- Carries deterministic seat-runoff ordering, reformulation holds, integer
  capture variance, single-refund qualification, challenge backing, alignment
  active-domain selection, scoped halt evidence, qualification floors and IBC
  refund identity. `home` remains version 1; endorsement gating is not included.
- #122's bank-supply denominator is no longer carried by v1 (review finding F1,
  2026-11-06): on dev-1's measured state it turns neutral readings into zero
  without Q1, so the earlier carrier blocked itself there and v2 could never
  run. It moves to consensus-repairs-v2, which switches it together with Q1.
  `vesting_rewards` stays at ConsensusVersion 3, native genesis must leave
  `alignment_bank_supply_enabled` absent or false, and v1 now passes on
  dev-1's census inputs.
- Fresh genesis must explicitly select all nine repairs. An offline copied-DB
  census supports re-checks before a height; dev-1 tooling recognizes knowledge
  12/13 without scheduling an upgrade.
- This source carries main after #167 (review finding F2): a carrier binary must
  come from a source the proposal-coherence gate reports `COHERENT`, never from
  the original head `a72ca85e`. Not scheduled. See
  [limits and rehearsal](docs/consensus-repairs-v1.md).

## Source — 2026-11-05 Proposal-coherence release gate

- `scripts/proposal-coherence-gate.py` decides, for one release source
  commit, whether `PrepareProposal` is coherent with `ProcessProposal`. It reads
  the commit's Git objects (not a worktree) and reports `NOT_AFFECTED` when no
  non-test Go file replays the ante in `ProcessProposal` and the commit descends
  from neither anchor, `COHERENT` when `PrepareProposal` replays its candidates
  and the eleven named coherence tests, with all their subtests, pass in a
  `git archive` of the commit, and `REFUSED` otherwise, including a descendant
  of `b8cff9c1` it no longer recognises and a descendant of `cd6cd145` that has
  lost the fix. It exits 2 without a verdict in a shallow clone, without an
  anchor commit, on Python older than 3.12, or when Go cannot build or run the
  tests at all.
- `docs/UPGRADES.md` and the `zerone-2` go/no-go checklist require its report,
  from a full clone of `main`, before a candidate source is frozen. CI's Test
  job, now on a full-history checkout, runs its offline self-tests and the gate
  itself on the commit under test.
- On 2026-11-05 it reported `NOT_AFFECTED` for zerone-1's source `2e37c4c`, the
  H1 and H2 pins and the 2026-08-01 H3 candidate `335bb94f`; `REFUSED` for
  `b8cff9c1` and zerone-dev-1's base `5542b221`; and `COHERENT` for `cd6cd145`
  and zerone-dev-1's patch source `bd8cfafd`.

## Source — 2026-11-02 PrepareProposal and ProcessProposal coherence

- `PrepareProposal` now replays each candidate transaction through the full
  ante handler at the proposal's height and time, in proposal order, against
  the disposable proposal state, and offers only the candidates that pass.
  Since the signature-hardening merge (`b8cff9c1`), `ProcessProposal` replays
  the same checks and refuses the whole block when one transaction fails, and
  CometBFT requires a block prepared by a correct proposer to be accepted. The
  mempool is admitted at the committed height, so before this change a
  candidate whose validity depends on the height, the block time or an
  earlier candidate could be offered and then refused.
- A candidate whose replay fails, or panics, is left out and selection
  continues. Each omission is logged at debug level, with one info line per
  proposal giving the count. The handler no longer returns an error: if the
  vote-extension injection (still behind its compiled-off latch) cannot be
  encoded, it is dropped and the proposal is built without it.
- Byte and gas selection, `ProcessProposal`, `FinalizeBlock`, ante rules,
  module versions and genesis are unchanged. A transaction whose wire framing
  is not canonical is proposed in its canonical encoding, the bytes that were
  replayed. A block from a patched proposer is accepted by patched and
  unpatched validators alike.
- This is a binary change with no upgrade handler. Every release binary built
  from a commit that contains `b8cff9c1` needs it. Source publication does not
  deploy it to any network.

## Package and source — 2026-11-02 `@zerone-chain/sdk` 0.1.1 on npm

- Package publication: the package owner published `@zerone-chain/sdk`
  `0.1.1` to npm by hand on 2 November 2026, with npm 2FA and without an npm
  provenance attestation, from a tarball built from the annotated `sdk-v0.1.1`
  tag on `main` commit `38bb765f`. The registry serves it as `latest`; its
  tarball's SHA-256 is
  `3df235181617eb1544f7590abe6aee76c702dee48367d4fa5436e407ae7879a2`, and
  packing the tag reproduces it byte for byte.
  [OPEN_CRYPTO_SDK.md](docs/standards/OPEN_CRYPTO_SDK.md) records its npm
  shasum and integrity. npm's staged publishing also created a placeholder
  version, `0.0.0-stage`, which holds no code: install `0.1.1` exactly and
  never `0.0.0-stage`.
- The agents' guide at `/join/agents.md` now installs `0.1.1` from npm, gives
  its hashes and how to check them, and keeps a rebuild from the tag only as an
  optional comparison. `llms.txt`, `README.md`, the SDK README and
  `OPEN_CRYPTO_SDK.md` say the package is on npm. The SDK README is packed into
  every release tarball, so its new wording ships from the next version.
- `publish-sdk.yml` no longer passes `--provenance=false`. Its stage step now
  passes `--provenance`, so a stage that cannot attest provenance fails instead
  of staging a version without it. npm's documentation says provenance is
  generated for trusted publishes from a public repository; no staged release
  has shown it yet, so confirm `dist.attestations` on the first one. The
  workflow now also requires Node 22.14.0 or later, as trusted publishing does.
  Because it tags each stage `latest` explicitly, which turns off npm's own
  prerelease and version-order checks, it refuses a prerelease version and any
  version not greater than both the registry's `latest` and every plain release
  it lists. Its refusal on a 404 says the package has existed since `0.1.1`.
  Later versions are staged through it and approved with npm 2FA; the
  trusted publisher is configured only when the next version is ready to stage
  within 48 hours.
- The FL-0 receipt profile at
  `/standards/frontier-evaluation-receipt-profile.v0.json` still lists the
  limit `sdk-unpublished`. Its dogfood receipt, dated 2026-08-01, binds those
  exact bytes by SHA-256, so correcting it needs a separately reviewed
  revision.

## Source — 2026-10-28 TypeScript SDK 0.1.1 and public-repository wording

- `@zerone-chain/sdk` is now version `0.1.1`, the first version to publish on
  npm. `0.1.0` was tagged as `sdk-v0.1.0` but never published: the README
  packed from that tag says the source repository "is currently private",
  which stopped being true on 28 October 2026, and GitHub's ruleset for
  `sdk-v*` tags refuses updating or deleting it. This change touches only the
  package's README and version number; the package code is that of
  `sdk-v0.1.0`.
- `README.md` and `docs/standards/OPEN_CRYPTO_SDK.md` state the repository's
  visibility history: first published on 5 July 2026, access-restricted from
  about 25 September, and public again since 28 October 2026. zerone.ai says
  the source has been public since 28 October 2026.
- Source publication does not publish `@zerone-chain/sdk` to npm. The package
  owner publishes `0.1.1` by hand from the `sdk-v0.1.1` tag, as
  [OPEN_CRYPTO_SDK.md](docs/standards/OPEN_CRYPTO_SDK.md) describes.

## Source — 2026-10-27 TypeScript SDK pre-release corrections

- The generated protobuf writer now encodes 64-bit varints in
  [2^31, 2^32) correctly. Before this change, values such as a post-2038
  `MsgRotateKey.authorization_expires_at_unix` were written as one truncated
  byte plus zero padding. The repair is applied by `scripts/generate.mjs`, so
  regeneration reproduces it, and it fails closed if the generator output
  changes. A shared Go/TypeScript wire vector and a boundary test cover it.
- The SDK README states which live networks accept the registration
  identity proof and dual-signature rotation (neither does on the v0.50.15
  line that `zerone-1` and `zerone-testnet-1` run), and names
  `accounting-authority-v1` as the first `zerone-1` upgrade whose source
  carries them. It warns against sending `MsgRotateKey` to those networks
  before that upgrade, and documents the `system` account freeze, the
  rotation cooldown and the metadata limit.
- Source publication does not publish `@zerone-chain/sdk` to npm.

## Source — 2026-09-13 claim fund settlement

- The separate `knowledge-fund-settlement-v1` 10→11 upgrade records prospective
  funding terms for paid claims, conjectures and all challenge routes. Valid
  reviews retain their 55% pool; refundable deposits return at every verdict,
  and unused review budgets return when nobody supplies an eligible review.
- Reviewer payments and payer refunds share the existing bounded retry queue
  and commit atomically with their paid markers. Frozen old obligations and
  historical absence retain their original terms; no retrospective refund,
  new issuance or scientific authority is inferred.
- An explicit pinned runtime handoff supports the original development genesis
  and predecessor replay before the recorded upgrade height. Source publication
  is separate from network activation and package publication. See the
  [financial terms and exact boundary](docs/specs/knowledge-fund-settlement-v1.md).

## Source — 2026-09-11 claim records consolidation

- The separate `knowledge-claim-records-v1` 9→10 upgrade retains exact new
  contradiction reasons and ordered evidence in existing Claim fields. Existing
  IDs are refused before collateral movement; record and bank changes commit
  atomically. Older records remain unchanged.
- A bounded ClaimHistory query, CLI command and narrow TypeScript protobuf
  reader join retained reviews, derived facts and directly linked challenges.
  Missing records and proof limits remain explicit. No persistent index,
  scientific score or reward mechanism is added.
- Source publication does not activate this behavior on the legacy live chain
  or publish the SDK to npm. See the
  [boundary and query contract](docs/specs/knowledge-claim-records-v1.md).

## Unreleased — 2026-08-15 correspondence geometry

### Added

- Correspondence Geometry v0: a sealed, digest-pinned atlas of seven bounded
  mappings across religion, energy, understanding, and universe. Each record
  exposes its epistemic lanes, background, map, preserved invariants,
  information loss, counterexamples, round-trip status, and prohibited
  transfers instead of treating resemblance as equivalence.
- A strict offline validator, bounded same-origin browser loader, accessible
  dimension-filtered dashboard view, substantive no-JavaScript summary, and
  source-byte checks for the local Zerone materials on which the atlas relies.
- A six-lane energy firewall that keeps physical energy, compute, protocol
  metabolism, economic value, lived experience, and spiritual or poetic
  language non-convertible by default.

### Epistemic and activation boundary

V0 contains zero `DUALITY_CANDIDATE` records and claims zero equivalences.
Its physics citations motivate questions but do not warrant Zerone mappings;
its theology and meditation lane creates no divine, doctrinal, scientific, or
protocol authority. Publication changes no consensus, chain state, fact
status, confidence, identity, person score, KARMA, qualification, reward,
money, governance, consent, rights, or automatic protocol/authority action.

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

- A bounded, read-only Knowledge Geometry projection and dashboard lens for
  public `zerone-1` facts and their explicitly declared typed relations. It
  preserves missing relations and contradiction without inferring affinity,
  understanding, authority, qualification, KARMA, or economic effect.
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
- Authority Geometry v1: a deterministic, source-pinned capability graph,
  independent report and target-gate checks, all 24 H4 and 14 H5 gates held
  closed, and a read-only public observatory. Current source intentionally
  fails the target gate; the artifact changes no chain or economic state.
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
  contains 218 paths and 457 definitions across the current application.

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
