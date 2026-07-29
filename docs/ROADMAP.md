# Zerone development roadmap

> Status snapshot: 2026-07-29. This is a dependency-ordered roadmap, not a
> release promise.

## Current shape

- The canonical public source is
  [`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
  The Go module path remains `github.com/zerone-chain/zerone`; changing that
  import path is a separate migration.
- The application wires 23 custom modules. Its transaction SDK covers 166
  request message types across 20 Zerone `Msg` services.
- The protobuf-generated Swagger document is the API inventory of record:
  [`docs/swagger-ui/swagger.json`](swagger-ui/swagger.json) currently contains
  214 paths and 438 definitions.
- The truth-seeking creed contains 20 commitments, and the ToK substrate
  doctrine contains TC0–TC6. Their executable bindings remain the authority
  over prose summaries.
- Repository material describes the existing `zerone-1` custodial launch.
  Source consolidation does not update a running validator.

## Consolidated in this source line

The current consolidation brings the compatible hardening work onto one source
line:

- provisional conjectures and type-correct challenge restoration;
- starvation-safe challenge settlement and bounded probe scanning;
- explicit deferral of funding-correlation vote decay after adversarial review
  found that permissionless dust transfers could poison another wallet's
  weight;
- protocol-wide substrate-axis ceilings and settlement clamps;
- adjudicated falsification clawback;
- K-alpha recognition events and bounded accounting guards;
- state/genesis validation and protobuf ownership fixes;
- CAIP account projections, unsigned in-toto provenance, the isolated Sigstore
  compiler, and the repository TypeScript SDK; and
- the fail-closed `zerone-2` release and authority kit.

Several items above change consensus-visible behavior. They are source-complete
only after their tests pass; they are **not live** merely because the code is
published. Activation requires the named `consolidation-safety-v1` coordinated
upgrade, an agreed height, matching validator binaries, and the normal
upgrade/recovery rehearsal. Never mix this binary with an older validator
binary at the same height.

## Publication boundary

This consolidation authorizes a GitHub source update only. It does not
authorize:

- a validator or network deployment;
- an npm publication of `@zerone-chain/sdk`;
- a release tag or binary release; or
- a merge, overwrite, or synchronization of the intentionally divergent
  Codeberg history.

The `zerone-2` kit remains **NO-GO**. A green source tree or successful drill is
not production authority. The signed release packet, ceremony artifacts,
operator decisions, initiation evidence, immutable image references, and the
complete authority chain in
[`deploy/networks/zerone-2/GO-NO-GO.md`](../deploy/networks/zerone-2/GO-NO-GO.md)
must all be satisfied before any phase they authorize.

## Next, in order

1. **Close the source consolidation.** Regenerate protobuf and Swagger
   artifacts, run creed/recursion integrity checks, complete Go and TypeScript
   tests, and publish the exact reviewed commit to GitHub.
2. **Prepare the consensus release.** Independently review
   `consolidation-safety-v1`, rehearse migration/export/restart paths, build
   reproducible binaries, and obtain an explicit activation decision.
3. **Complete `zerone-2` authority.** Run the two-machine ceremony comparison,
   bind signed source and immutable artifacts, satisfy every phase-specific
   gate, and keep all services private until their signed decision permits
   otherwise.
4. **Publish ecosystem artifacts separately.** Give the TypeScript SDK its own
   registry verification and npm release decision. Publish Chain Registry
   metadata only after chain identity, genesis, endpoints, denomination, and
   coin type are final.
5. **Continue boundary-first standards work.** Priorities are feegrant UX,
   audited delegated authority, AgentTool x402, an ERC-8004-inspired profile,
   and digest-only CID/in-toto/SPDX/C2PA/A2A integrations.
6. **Modernize the consensus stack.** Cosmos SDK 0.50 and IBC-Go 8 require a
   planned, rehearsed migration rather than an opportunistic dependency bump.

## Explicit deferrals

- The older same-model training-fund replay change is incomplete and remains
  out until the full debit/replay invariant is specified and tested.
- The older capture-recovery work is incomplete and unsafe to recover as a
  bulk branch merge. Any surviving behavior must be re-derived against current
  state and authority rules.
- Funding-correlation records remain observable, but do not affect vote weight.
  A future design must require evidence an untrusted sender cannot fabricate.
- General authz, a contract VM, reward-bearing Sigstore registration, and rich
  remote-document parsing remain outside the current consensus boundary.

## Historical record

The original round-by-round implementation record remains available in
[`docs/archive/REWRITE-PLAN.md`](archive/REWRITE-PLAN.md), [`prompts/`](../prompts),
and [`docs/plans/`](plans). Current doctrine is defined by
[`docs/TRUTH_SEEKING.md`](TRUTH_SEEKING.md) and
[`docs/TOK_SUBSTRATE.md`](TOK_SUBSTRATE.md).
