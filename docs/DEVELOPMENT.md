# Developing Zerone

**Source review: 27 September 2026, `main` at `9caaf36bca0b7d76a6b8206c2ffe595edb6593e8`.**
This is the development entry point, not a release authorization or a live status page.
The canonical repository is `cambridgetcg/zerone-core` (private); source access and
public website availability are different things. Follow [project instructions](../CLAUDE.md).

Zerone's product target is **global compute contributing to fact-checking and the
Tree of Knowledge, with useful work compensated in tokens**. The current
foundation is attributed scientific records, reasoned review, retained corrections
and reliable settlement of explicitly funded work. The bounded
[external-worker loop](../tools/external-worker/README.md) connects one checkable
computation to a task journal and an operator-approved test-token payment. It is
not a permissionless market or an activated universal reward rule. Keep local
research useful without requiring chain admission. The
[lean-core scope](LEAN-CORE-CONSOLIDATION.md) selects the work;
[Authoritative State](AUTHORITATIVE-STATE.md) owns the accepted authority target;
[the consolidation sequence](MODULE-CONSOLIDATION-SEQUENCE.md) owns its order and prerequisites.
For the full architecture use [ZERONE_MAP](ZERONE_MAP.md); for current module purposes
and connections use [the shared catalog](../dashboard/module-guide.json), not another list here.

## Status: say which kind of evidence you have

| Tag | Meaning |
| --- | --- |
| **Implemented source** | Code exists in the reviewed checkout. Registration, tests and enabled behavior are separate questions; this does not establish deployment. |
| **Live observed** | A named chain/endpoint returned a dated observation. Preserve source, height and trust limits; an operator response is not independent consensus proof. |
| **Accepted design / pending** | Normative target with implementation or activation still outstanding. It grants no current runtime authority. |
| **Source candidate** | Proposed/unmerged code or a gated prototype. Inspect its exact revision and integration before adopting it; it is not automatically the active plan. |
| **Research** | Local journals, static models, experiments and exploratory designs. Publication or a diagram does not create a chain rule, verdict or reward. |
| **Historical** | A dated roadmap, census, release or observation retained as evidence, not the current queue or a fresh state assertion. |

Current boundaries:

- **Implemented source:** [app registration](../app/app.go) still contains all
  40 catalogued modules: 17 SDK/IBC and 23 custom. They run within one application,
  not 40 services. This consolidation retires none of those modules. Browser journal authoring,
  correction and explicit native-workflow preparation are implemented in the
  [research tools and reader](../tools/research-review/README.md); they activate no chain upgrade.
- **Live observed, 27 September:** public checks found `zerone-dev-1` on knowledge 11,
  runtime source `5542b221864ca0080e143fa82abbef4cd2064f71`, matching the published descriptor;
  development and legacy `zerone-1` both reported `catching_up=false`. Anonymous
  participant-archive and raw release-manifest HEAD requests returned **404**:
  the advertised package path is **blocked for anonymous readers**, not an absent
  runtime. See the [dated observations](reports/consolidation-observations-2026-09-27.json)
  and [participant guide](SHARED-DEVELOPMENT.md). The [23 September publication](reports/development-publication-2026-09-23.json)
  corrected runtime/package compatibility and retained knowledge-10 genesis replay;
  it does not establish present download access. Legacy, development and the
  unactivated `zerone-2` kit remain separate networks/release paths.
- **Accepted design / pending:** SDK staking as sole consensus stake owner, SDK
  governance as sole ordinary decision/execution owner, ontology as sole domain
  owner. H4-F → H4 → H5 remains unimplemented. Current custom stake/governance and
  duplicate domain writers still exist; H4 alone would not remove SDK stake-weighted governance.
- **Research / historical:** the optional research library and older
  [roadmap](ROADMAP.md) are not another protocol implementation queue. The
  [participant review handoff](DEVELOPMENT-REVIEW-HANDOFF.md) is prepared, not a completed independent review.

## Layers and ownership

These are responsibility boundaries, not new packages or deployments.

| Layer | Owns / intended purpose | Entry points and boundary |
| --- | --- | --- |
| **Clients** | Author, preserve, inspect and deliberately sign work; display observations. | [dashboard](../dashboard/README.md), [TypeScript SDK](../sdk/typescript/README.md), [participant client](../scripts/shared-claims.py), [local journal/workflow](../tools/research-review/README.md). A view, attribution label or imported file does not confer chain authority. |
| **Protocol** | Consensus, transaction admission, signatures, wire contracts and SDK accounting/consensus primitives. | [node executable](../cmd/zeroned/main.go), [ante checks](../app/ante.go), [ABCI](../app/abci.go), [schemas](../proto/), [dependency pins](../go.mod). The [PoT vote-extension latch](../app/vote_extensions.go) is compiled off; signed commit/reveal transactions are a different route. |
| **Application** | Scientific claim/review/history rules, obligations, custody-specific state and keeper connections. | [app assembly/block order](../app/app.go), [custom modules](../x/), [knowledge phases](../x/knowledge/keeper/phases.go). SDK bank owns balances; application records own the obligations against them. Current ownership is not the accepted future ownership. |
| **Operations** | Exact builds, network identity, gateway exposure, state retention, upgrades and recovery evidence. | [shared development runtime](../deploy/networks/zerone-dev-1/README.md), [legacy observer](../deploy/legacy-observer/README.md), [upgrade/incident operations](UPGRADE_AND_INCIDENT_OPERATIONS.md). Use the compatible network-specific release, not moving `main` on a legacy node. |
| **Design** | Accepted invariants, scope, migration prerequisites and explicitly exploratory proposals. | [authority decision](AUTHORITATIVE-STATE.md), [sequence](MODULE-CONSOLIDATION-SEQUENCE.md), [specifications](specs/), [dated reports](reports/), [foundational texts](../canon/). Documents do not register handlers, move money or prove activation. |

## Follow the actual record transition

```text
local journal + separately preserved evidence
  → select one exact contribution and assertion; prepare a local workflow packet
  → explicit participant-client signing and submission
  → native claim → signed review commitments/reveals → retained outcome/history
  → named reviewer/refund obligations → successful bank transfer + paid record
```

The [workflow adapter](../tools/research-review/workflow.py) binds an exact journal
snapshot, record and intent. Importing, browsing or correcting the journal neither
submits nor silently edits an existing native assertion. The browser's native-history
read is explicit and read-only; keys and signing stay with the participant client.
An optional [checkpoint](specs/research-checkpoint-v1.md) is a signed digest commitment,
not claim admission, evidence storage or a review round.

[Neutral review](specs/knowledge-review-neutrality-v1.md) counts admitted accounts
equally and pays valid timely reviews, including dissent; this is not proof of
independent controllers or external-world truth. A later correction/counterclaim is
retained separately; a native challenge needs an actual Fact ID.
[Funding terms and retries](specs/knowledge-fund-settlement-v1.md) distinguish unpaid
obligations from completed transfers. A vesting schedule is not a payment, and a
reported paid marker is not by itself an independently verified bank transfer.
Historical claims retain their applicable policy rather than being repriced.

## Names: improve labels, preserve identifiers

Use descriptive display names to disambiguate existing components. Module IDs,
source directories and SDK package names are different namespaces:

| Display distinction | Registered IDs | Source location |
| --- | --- | --- |
| SDK consensus staking / custom participation staking | `staking` / `zerone_staking` | SDK dependency wired in [app](../app/app.go) / repository [x/staking](../x/staking/) |
| SDK governance / custom LIP governance | `gov` / `zerone_gov` | SDK dependency wired in [app](../app/app.go) / repository [x/gov](../x/gov/) |
| SDK accounts / Zerone account-key-DID bindings | `auth` / `zerone_auth` | SDK dependency wired in [app](../app/app.go) / repository [x/auth](../x/auth/) |
| Ontology domain registry | `zerone_ontology` | [x/ontology](../x/ontology/); `knowledge` still has a separate domain registry pending migration. |

In particular, the accepted design's “SDK `x/staking`” does **not** mean this
repository's `x/staking`. Use the catalog's exact IDs for lookup. `x/common` is
shared code and `x/witness` is unregistered, not additional catalog modules.
This is descriptive labeling, **not a migration or a rename**: preserve protobuf
packages/type URLs/fields, store keys, chain IDs, CLI/API identifiers and the
`github.com/zerone-chain/zerone` Go import path. Target controller, profile,
electorate and legacy-claims responsibilities cannot be implemented by relabeling old stores.

## Where to change what

The tests below exist in source; they are starting routes, not a claim that every
suite has been run for your change. Follow each boundary into its integration tests.

| Task | Edit / trace first | Existing tests |
| --- | --- | --- |
| Browser journal authoring, corrections and claim workspace | [research-review-journal.ts](../dashboard/src/research-review-journal.ts), [research-review.ts](../dashboard/src/research-review.ts), [research-workflow.ts](../dashboard/src/research-workflow.ts) | [research-authoring.test.ts](../dashboard/tests/research-authoring.test.ts), [research-workflow-ui.test.ts](../dashboard/tests/research-workflow-ui.test.ts) |
| Local journal → explicitly signed native claim | [journal.py](../tools/research-review/journal.py), [workflow.py](../tools/research-review/workflow.py), [shared-claims.py](../scripts/shared-claims.py) | [test_workflow.py](../tools/research-review/test_workflow.py), [test_shared_claims.py](../scripts/test_shared_claims.py); opt-in native [test_workflow_native.py](../tools/research-review/test_workflow_native.py) |
| Development reader, network pins and truthful history | [development-reader.ts](../dashboard/src/development-reader.ts), [development-profile.ts](../dashboard/development-profile.ts) | [development.test.ts](../dashboard/tests/development.test.ts), [research-workflow.test.ts](../dashboard/tests/research-workflow.test.ts) |
| Claim admission, review outcome and correction history | [msg_server.go](../x/knowledge/keeper/msg_server.go), [rounds.go](../x/knowledge/keeper/rounds.go), [query_claim_history.go](../x/knowledge/keeper/query_claim_history.go) | [review_neutrality_test.go](../x/knowledge/keeper/review_neutrality_test.go), [query_claim_history_test.go](../x/knowledge/keeper/query_claim_history_test.go), [app/claim_records_test.go](../app/claim_records_test.go) |
| Funding, pending payouts and retries | [fund_settlement.go](../x/knowledge/keeper/fund_settlement.go), [verifier_settlement.go](../x/knowledge/keeper/verifier_settlement.go), [vesting.go](../x/vesting_rewards/keeper/vesting.go) | [fund_settlement_integration_test.go](../app/fund_settlement_integration_test.go), [survival_handoff_test.go](../x/knowledge/keeper/survival_handoff_test.go) |
| Keeper ownership, hooks and a named migration | [app.go](../app/app.go), [upgrade_registry.go](../app/upgrade_registry.go), [authority-graph](../tools/authority-graph/) | [upgrades_fund_settlement_test.go](../app/upgrades_fund_settlement_test.go), [authority-graph/main_test.go](../tools/authority-graph/main_test.go); add exact-boundary tests for new migrations. |
| Transaction/consensus safety and API compatibility | [ante.go](../app/ante.go), [abci.go](../app/abci.go), [proto](../proto/), [SDK](../sdk/typescript/) | [ante_integration_test.go](../app/ante_integration_test.go), [vote_extensions_test.go](../app/vote_extensions_test.go), [SDK wire.test.ts](../sdk/typescript/tests/wire.test.ts) |
| Development node, gateway or participant packaging | [runtime.py](../deploy/networks/zerone-dev-1/runtime.py), [gateway.py](../deploy/networks/zerone-dev-1/gateway.py), [runtime README](../deploy/networks/zerone-dev-1/README.md) | [test_runtime.py](../deploy/networks/zerone-dev-1/test_runtime.py), [test_gateway.py](../deploy/networks/zerone-dev-1/test_gateway.py), [test_participant_package.py](../deploy/networks/zerone-dev-1/test_participant_package.py) |
| Current module explanations / future dispositions | [module-guide.json](../dashboard/module-guide.json) / [consolidation sequence](MODULE-CONSOLIDATION-SEQUENCE.md) | [module_guide_test.go](../app/module_guide_test.go), [understand-guide.test.ts](../dashboard/tests/understand-guide.test.ts) |

For other paths, use the read-only [source navigator](../tools/source-map.py) from
repository root; it reads the Git-indexed working tree, excluding untracked files.
Its `check` validates metadata/catalog references, not runtime registration or
document anchors. Use the app test below for registration; keep inferred file
purposes distinct from reviewed module descriptions.

```sh
python3 tools/source-map.py summary
python3 tools/source-map.py find settlement --limit 12
python3 tools/source-map.py show x/knowledge/keeper/verifier_settlement.go
python3 tools/source-map.py module knowledge
python3 tools/source-map.py module zerone_staking
python3 tools/source-map.py export
python3 tools/source-map.py check
```

## Next concrete work, in the accepted order

1. **Resolve participant access, then exercise the prepared handoff.** Keep the
   anonymous-download blocker visible and establish an authorized package-access
   route before inviting a new participant; this does not authorize republishing
   private source or changing repository privacy. Then use
   [DEVELOPMENT-REVIEW-HANDOFF](DEVELOPMENT-REVIEW-HANDOFF.md) with a willing reviewer's own
   key and judgment. Retain actual receipts, outcome, warranted correction and
   payment observations. Record independent control separately; synthetic accounts
   and an inconclusive result must not be disguised as independent acceptance.
2. **Reconcile before financial retirement.** Follow sequence stage 2: take a fresh
   authenticated, height-bound custody/process census for the affected network;
   match claimant records, pending payouts, deposits, schedules and refunds to
   balances and usable exits. Earlier zero-state observations do not satisfy this gate.
3. **Implement authority unification, not cosmetic consolidation.** Follow stage 3
   and [the accepted migration](AUTHORITATIVE-STATE.md#7-migration-and-activation):
   preserve required H1/H2/H3 lineage; implement H4-F drain/freeze and committed census,
   then H4 reconciliation/retirement, then H5 only after H4 verification. Tests must
   prove one writer, preserved claims/history and the exact activation boundary.
4. **Remove dispensable consensus work only after tracing its consumers.** Use the
   [per-module dispositions](MODULE-CONSOLIDATION-SEQUENCE.md#disposition-of-all-23-registered-custom-modules),
   not a new module wishlist. Alignment pacing and ontology ceilings still affect
   knowledge; neutral reviews did not make every older dependency inert. Preserve
   observations and exits while explicitly retiring unwanted feedback and callbacks.
5. **Prove the reduced runtime and operations.** Rehearse migration, restart,
   export/import and recovery with exact source/binary identities. Demonstrate
   preserved obligation totals, usable exits, no alternate writers and bounded
   worst-case block work. Publish retained-query/archive scope and separately verify
   actual activation. Fewer files, green tests or a merged change are not that evidence.

## Checks and change discipline

Start with the routed tests; use [Makefile](../Makefile), [CI](../.github/workflows/ci.yml)
and the [SDK](../sdk/typescript/package.json)/[dashboard](../dashboard/package.json)
package scripts for the complete applicable checks. Representative root commands:

```sh
python3 -B -m unittest discover -s tools/research-review -p 'test_*.py'
go test ./app -run '^TestModuleGuideCoversRegisteredApplication$' -count=1
```

The native workflow test is opt-in, not proven by its default skip. Build the SDK
before the dashboard as its [README](../dashboard/README.md#run-locally) specifies.
For wire changes, edit `.proto` first, run `make proto-gen`, regenerate affected
clients and run `make proto-check`; never hand-edit generated `*.pb.go`.
Protocol/custody changes require their named migration and network-specific release
checks even when the UI is unchanged. Keep future design, source implementation,
dated runtime observations and participant outcomes in their linked owner documents;
update this entry only when routing, scope or those boundaries change.
