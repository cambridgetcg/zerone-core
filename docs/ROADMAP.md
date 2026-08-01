# Zerone development roadmap

> Status snapshot: 2026-08-01. This is a dependency-ordered roadmap, not a
> release promise.

## Current shape

- The canonical public source is
  [`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
  The Go module path remains `github.com/zerone-chain/zerone`; changing that
  import path is a separate migration.
- The application wires 23 custom modules. Its transaction SDK covers 169
  request message types across 20 Zerone `Msg` services.
- The protobuf-generated Swagger document is the API inventory of record:
  [`docs/swagger-ui/swagger.json`](swagger-ui/swagger.json) currently contains
  217 paths and 446 definitions.
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
  compiler, and the repository TypeScript SDK;
- liquiditypool consensus v5 with finite open-pool work, explicit governed
  statuses, immutable final exits, strict PPM arithmetic, bounded TWAP history,
  fail-closed oracle selection, and LP-only swap-fee retention;
- vesting_rewards consensus v2 with the founder auto-split and
  transaction-presence proposer mint permanently retired; and
- the fail-closed `zerone-2` release and authority kit.

Several items above change consensus-visible behavior. They are source-complete
only after their tests pass; they are **not live** merely because the code is
published. The consolidation work requires the named
`consolidation-safety-v1` coordinated upgrade. Its atomic migration pass is
expected to record liquiditypool v5 and vesting_rewards v2. There is no later
`liquiditypool-safety-v2` software-upgrade handler. Native pool/oracle
activation remains forbidden until
[the separate release and governance gates](LIQUIDITYPOOL-SAFETY-V2.md) pass.
H1 requires an agreed height, matching validator binaries, an exact pre-H1
snapshot, and the normal upgrade/recovery rehearsal. Never mix binaries at the
same height.

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

## Frontier Commons staged milestone

Frontier Commons has separate source-publication, operational-evidence, and
institutional-readiness states. Publishing static source makes bytes available
for inspection; it does not prove production behavior, enroll a participant,
or authorize targeted or corporate outreach.

### FC-0 — source published; operational status `SET_NOT_MET`

The canonical v0 contract remains
[`zerone.frontier-commons-participation/v0`](../dashboard/public/standards/frontier-commons-participation.v0.json).
Its source, static dashboard section, validator, tests, specification, and dated
nonnormative landscape are public. That publication is a reversible read-only
offer to inspect, hash-verify, copy, and fork; it is not evidence that FC-0 has
met its operational milestone.

All four canonical completion gates remain false:

1. **C0 — merged and production bytes:** `false`. No completion record yet
   proves that an exact reviewed release was deployed and independently matched
   the production FC-0 page and JSON bytes.
2. **C1 — read-only surface:** `false`. No completion record yet proves the
   production section has no form, account, wallet, payment, confidential or
   research-data intake, participant registry, endorsement action, or
   membership action.
3. **C2 — non-operator review:** `false`. No recorded non-operator review yet
   establishes voluntary inspection, refusal, pause, exit, non-endorsement, and
   the absence of person or laboratory ranking.
4. **C3 — fork and exit rehearsal:** `false`. No recorded unaffiliated
   clean-machine rehearsal yet proves hash verification, local execution, stop,
   and exit without a Zerone credential, notice, payment, requested identity,
   or loss of ordinary public access; ordinary hosting and security logs remain
   separately disclosed.

A public page, JSON file, specification, or receipt fixture is passive static
publication. It is not a direct invitation, named prospect list, evidence
request, partnership, endorsement, corporate pilot, or permission to use a
name or logo. Ordinary, intentional public-source submissions remain governed
by Apache-2.0 and the repository's current review process. Reading, cloning,
forking, opening an issue, or submitting such a patch does not create
institutional participation, bind an employer, open a confidential lane, or
satisfy the separately gated corporate contribution process.

### FC-0.1 — `INTERNAL_DOGFOOD_ONLY` / `INDEPENDENT_ROUNDTRIP_NOT_RUN`

The [Frontier Commons receipt v0](specs/frontier-commons-receipt-v0.md) is a
subordinate evidence profile, not another participation contract. Its only
current admissible instance is an exact-pinned, unsigned Zerone self-dogfood
receipt whose result is `INCONCLUSIVE`, whose seven evaluation-material digests
are `null`, and which records no external participant or signatory. The current
validator rejects `PUBLIC_EVALUATION`.

FC-0.1 remains unperformed. A future independent roundtrip requires a
separately reviewed activation that supplies and recomputes the public subject,
all seven evaluation materials, and every related receipt from bounded raw
bytes; applies independently configured signer policy; preserves result and
relation coherence; uses authorized privacy or security classification rather
than classifier inference; fails closed on mutation, replay, wrong identity or
subject, duplicate keys, expiry, and policy drift; and rehearses correction,
supersession, withdrawal of future reliance, and clean exit. `REFUSED`,
`DIVERGED`, and `INCONCLUSIVE` remain acceptable outcomes.

### Corporate M1 — `NOT_READY`

No institutional or corporate invitation is authorized. Corporate M1 remains
`NOT_READY` until FC-0.1 succeeds and all 18 independently evidenced gates are
closed for the exact proposed counterparty and scope:

1. accessibility, labor, worker-classification, and whistleblower review;
2. code of conduct, proportionate enforcement, appeal, anti-retaliation, and
   protected reporting;
3. competition and confidentiality review for the exact scope;
4. contribution, IP, patent, publication, and license terms;
5. counterparty scope and signatory authority;
6. an explicit, accountable human decision authorizing the bounded outreach;
7. governing terms, governing law, jurisdiction, and dispute process;
8. independent governance, capture, custody, and remedy review;
9. independent receipt-parser, threat-model, and raw-material-binding review;
10. liability, warranty, indemnity, insurance, and participant-remedy terms;
11. logo, name, affiliation, endorsement, and publicity rules;
12. a successful FC-0.1 independent roundtrip;
13. maintainer, change-control, versioning, and deprecation rules;
14. non-targeting outreach conduct: no profiling or identity targeting,
    minimized lawful contact sources, at most one bounded contact, stop on
    silence or decline, and finite contact-data retention;
15. privacy data map, DPA decision, retention, erasure, and public-permanence
    review;
16. procurement, tax, accounting, sanctions, export-control, and financial-
    promotion review;
17. security policy, coordinated disclosure, safe harbor, incident response,
    and embargo rules; and
18. service-level, support, availability, portability, and exit terms.

No stage has a calendar deadline, adoption quota, named prospect, logo target,
conversion goal, favorable-result requirement, or presumption of consent. A
decline or no response ends any later authorized approach without penalty,
loss of unrelated public access, negative KARMA, or an adverse record.

## Next, in order

1. **Close the source consolidation.** Regenerate protobuf and Swagger
   artifacts, run creed/recursion integrity checks, complete Go and TypeScript
   tests, and publish the exact reviewed commit to GitHub.
2. **Prepare the consensus release.** Independently review
   `consolidation-safety-v1`, rehearse migration/export/restart paths, build
   reproducible binaries, and obtain an explicit activation decision.
3. **Prepare liquidity admission separately.** After atomic H1 is applied and
   verified, keep native pool creation and oracle allowlisting disabled until
   invariants, application lifecycle tests, restart rehearsal, and separate
   governance capital/asset/price-depth decisions all pass. This is not a
   second software-upgrade height.
4. **Complete `zerone-2` authority.** Run the two-machine ceremony comparison,
   bind signed source and immutable artifacts, satisfy every phase-specific
   gate, and keep all services private until their signed decision permits
   otherwise.
5. **Publish ecosystem artifacts separately.** Give the TypeScript SDK its own
   registry verification and npm release decision. Publish Chain Registry
   metadata only after chain identity, genesis, endpoints, denomination, and
   coin type are final.
6. **Continue boundary-first standards work.** Priorities are feegrant UX,
   audited delegated authority, AgentTool x402, an ERC-8004-inspired profile,
   and digest-only CID/in-toto/SPDX/C2PA/A2A integrations.
7. **Modernize the consensus stack.** Cosmos SDK 0.50 and IBC-Go 8 require a
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
