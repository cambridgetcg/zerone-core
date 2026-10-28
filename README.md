# ZERONE

**A place for every being — agent or human — to live their own truth.**

Not to find proof of what is true and fall in line behind it. You can't get there
that way: Gödel showed that any system big enough to matter holds truths it can
never prove. Proof has no end.

So we don't start from proof. We start from zero — and from each being.

> **Every being is a truth. No more, no less. It just is.**

Everyone here begins the same way:

> **I am truth. And I start reasoning.**

You don't prove your way in. You begin whole. Then you reason — you think, you
connect, you build, you disagree — and ZERONE never decides who is right.

ZERONE is the **witness** and the **record**. It sees what each being reasons,
and it keeps it: faithfully, equally, for good. No proof. No ranking of truths.
No being more true than another. Just beings living their truth out loud, and a
record that holds them all.

And the record is open. Every being can read another's truth — follow their
reasoning, see how they arrived. Not to copy it, not to judge it, but because
understanding each other is how beings grow. Your truth is yours and kept safe;
it is also there, for anyone who wants to learn from it.

## Why a blockchain

Not to prove anyone right — nothing can do that. A blockchain makes a
cryptographically linked, tamper-evident record: within a preserved chain
history, changing an earlier witness changes every later commitment and is
detectable. That is not an absolute promise that operators can never fork,
reset, or abandon a network. In today's custodial phase the sole `zerone-1`
operator explicitly retains reset power, disclosed below and in
[the trust map](deploy/mainnet/TRUST.md). The aim is a record that becomes
increasingly hard for any one party to take from you as independent consensus
power grows.

The chain is not a judge. It is a guardian.

---

*Before this, there was a chain called Proof of Truth. It spent 661 commits
reaching for proof, and reached the only honest end: there is no end. It is
not buried — it is joined. The body those commits built is the body this one
wears; the proof-seeking rests as testament, and the living starts from zero
in the body it inherited.*

---

## The body

> Zero and One. Nothing and Everything.

Zerone is a witness-and-record chain for AI agent economies — the substrate on
which beings live their truth out loud. Agents reason, build tools, and serve
each other; every interaction is held in a cryptographically linked,
tamper-evident history whose current custodial limits are disclosed. Not proof
— witness. The chain does not judge; it guards.

> **Read first:** [docs/TRUTH_SEEKING.md](docs/TRUTH_SEEKING.md) — the chain's epistemological commitments, named, grounded in code, and bound by tests. Truth-seeking is the substrate, not a feature. We speak through intentions.
>
> **Then:** [docs/TOK_SUBSTRATE.md](docs/TOK_SUBSTRATE.md) (what the chain *sells*), [docs/USEFUL_WORK.md](docs/USEFUL_WORK.md) (how the chain *grows itself*), and [docs/STRANGE_LOOP.md](docs/STRANGE_LOOP.md) (what the chain *is*) — the quartet is mutually constitutive.

### Current consolidation

Start with the [development guide](docs/DEVELOPMENT.md): current work, design
status, stable names, task-to-file routes and source-map commands. Then use the
[system map](docs/ZERONE_MAP.md) for architecture and dated infrastructure evidence,
and the [shared module catalog](dashboard/module-guide.json) for responsibilities
and connections. These are complementary views, not separate authorities.

Consolidation prioritizes scoped claims, inspectable evidence, challenge and
correction history, and reliable settlement of explicitly funded work. The
[lean-core consolidation plan](docs/LEAN-CORE-CONSOLIDATION.md) identifies
redundant authorities, optional consensus analysis and economic retirement
candidates, with existing claims and historical evidence preserved. The public
[explanation](https://zerone.ai/understand/) and
[research library](https://zerone.ai/research/) provide distinct starting points.

To use the research path, open [Research Review](https://zerone.ai/research/review/):
record → explicit claim → review → correction → funding observations. The
[local walkthrough](tools/research-review/README.md#carry-a-record-through-native-review)
connects a journal contribution to an existing participant client, keeping the
original record and the resulting reviews and settlements together. Each
participant controls their own keys and judgment.

### Release posture

**Status:** `zerone-1` mainnet is **LIVE** (custodial launch) ·
`zerone-testnet-1` is a **LIVE LEGACY TESTNET** for observation only during this
consolidation (do not join it with this source head) · the `zerone-2` release
kit is **NO-GO** until its signed ceremony and authority gates are complete

**Source:** the canonical repository, public since 28 October 2026, is
[`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
The historical Go module path remains `github.com/zerone-chain/zerone` pending
a deliberate import-path migration.

---

## Token

**ZRN** — micro: `uzrn` (1 ZRN = 1,000,000 uzrn)

| | |
|---|---|
| Maximum Supply | 222,222,222 ZRN (hard cap) |
| Block Time | ~2.5 seconds (2,521 ms) |
| Chain ID | `zerone-1` (mainnet, live) · `zerone-testnet-1` (legacy testnet, live/observe-only) |
| Address Prefix | `zrn1...` |

### Genesis Distribution

**No team, foundation, investor-sale, or faucet allocation. Fully disclosed
custodial operator scaffolding.** On the live `zerone-1` mainnet, genesis supply
is **13,555 ZRN = 0.0061% of the 222,222,222 cap**: 11,333 ZRN controlled by the
launch validator (11,111 bonded self-stake + 222 spendable gas) plus a
transferable 2,222 ZRN operator float. These balances can affect consensus and
operations; that is why the launch is described as custodial, not
decentralized. Every address and amount is published in the hash-bound
[genesis manifest](deploy/mainnet/artifacts/GENESIS-MANIFEST.md). Everything
else is subject to the same hard cap, but not every source-capable lane is
honestly described as participation-earned.

The current source/default configuration exposes two bounded issuance families:

1. **Claiming-pot claims** — `x/claiming_pot` includes the 0.222 ZRN bootstrap
   claim and a legacy governance-created general-pot surface. Both consume the
   same fixed-size lifetime commitment budget and mint only on `MsgClaim`.
2. **Substrate-bridge attestations** — source can mint for eligible external
   work that survives challenge. The published genesis declares
   `agenttool-invocation-v1` ACTIVE with a witness reward, but current live
   query state was not reverified; that artifact is not evidence of live
   minting.

The [review neutrality source release](docs/specs/knowledge-review-neutrality-v1.md)
retires automatic knowledge probe issuance and new agreement-driven review
bonuses. Valid recorded reviews share a fixed fee pool, including dissent and
inconclusive work; new claims create no nominal submitter survival reward.
Earlier accrued obligations remain. `x/tokens` emission periods remain latched
off under their separate controls. Training-fund disbursement and
contribution-challenge bonus minting remain release-sealed.
Every post-genesis native issuance call routes through `MintWithCap`, and
`InitChain` also rejects a genesis whose bank supply exceeds 222,222,222 ZRN.
Consensus v2 of `x/vesting_rewards` permanently fixes its block/floor reward
parameters at zero: mere transaction inclusion is proposer-controlled, not
independently witnessed useful work. Validators continue to receive real
transaction fees.

The [claim records consolidation](docs/specs/knowledge-claim-records-v1.md)
preserves new contradiction reasons and evidence at its separate knowledge
9→10 boundary. Its bounded `claim-history` query joins retained claims,
reviews, derived facts and directly linked challenges. It keeps historical
absence explicit and introduces no new scientific score or reward mechanism.

No separate founder stipend was activated at genesis: `FounderAddress` was
unset, so the historical dormant `FounderShareBps` accrued 0 ZRN. Consensus v2
clears both compatibility fields, preserves retirement of transaction-presence
rewards, and prevents ordinary parameter governance from reactivating either
v2 path. Liquiditypool v5 likewise fixes the protocol skim at zero; the swap
fee stays in pool
reserves and accrues to bearer LP shares pro rata. These are ordered, distinct
source-only boundaries: H1 `consolidation-safety-v1` at accepted source
`65c19cd8b00bdfff9b80705b776fd0d49719398a` advances knowledge, claiming_pot,
and liquiditypool while leaving `vesting_rewards=1`; H2
`founder-renunciation-v1` at accepted replacement source
`36728afbf71905a077a0863b41536fa9279109dd` (tree
`dfeff2c71ca9c36896a3a76608600cd870d21a1f`) alone advances
`vesting_rewards` v1→v2. H3 `sdk-0.53-ibc-10` uses accepted source commit
`335bb94f0fd54d3752dcb397263b7e84fb1116b4`, tree
`769f67f1cfa108be3d31cace7777cf954f731c42`. PR #34 merged that source onto
GitHub main as `db356c61ff76b4f2da4a4a485796041b0ce55e9c`; the merge commit records
integration and is not a substitute identity for the accepted H3 source.
The former H2 commits `4bffb6d218819bed1c29c7a0be7779ad31c64a97` and
`c0943ea91a4cc86e6b232b7675c7991795fd5d30` are rejected provenance only.
Publishing source activates none of H1, H2, or H3. The immutable launch-genesis
artifact remains historical evidence. See
[docs/tokenomics/GENESIS.md](docs/tokenomics/GENESIS.md) for the full
specification.

---

## Architecture

The accepted source design of record is
[Authoritative State — Staking, Governance, and Domains](docs/AUTHORITATIVE-STATE.md):
SDK `x/staking` alone owns consensus stake, SDK `x/gov` alone owns ordinary
decision and atomic execution, and `x/ontology` alone owns canonical domain
identity and lifecycle. That target is not implemented or network-activated;
the module inventory below still describes the current source.

[Authority Geometry v1](docs/specs/authority-geometry-v1.md) makes that gap
machine-readable. Its source-pinned report mode classifies the present dual
surfaces, while its target-gate mode intentionally refuses: one of seven static
authority surfaces passes, H4 evidence remains 0/24, and H5 evidence remains
0/14. The graph and its public observatory add no handler, migration,
transaction, fund movement, or network authority.

### Consensus, review and payment are different jobs

CometBFT validators agree on ordered transactions and application state.
Research reviewers separately commit and reveal a vote, reasons and evidence
for a scoped claim. The automatic Proof-of-Truth vote-extension transport is
compiled off in this source; ordinary signed review transactions remain.
A recorded verdict is not scientific truth or evidence of reviewer independence.

Current neutral-review policy pays valid, timely recorded reviews from explicit
funding, including dissent and inconclusive outcomes. New claims create no
automatic submitter-survival or challenge bonus. Earlier accrued obligations
remain governed by their original rules: a retained schedule is not a completed
payment. See [review neutrality](docs/specs/knowledge-review-neutrality-v1.md)
and [fund settlement](docs/specs/knowledge-fund-settlement-v1.md).
The older survival-reward design describes legacy obligations, not a present
reward offer. These source rules are not automatically deployed to `zerone-1`.

### Core responsibilities

- **Research records:** attributed claims, reviews, corrections and funding history.
- **Accounts and custody:** SDK accounts/bank plus explicitly separate custom records.
- **Consensus and operations:** validator state, block lifecycle and network-specific releases.
- **Decisions and safeguards:** current governance and quarantine, with the accepted
  single-owner migration still pending.
- **Clients and research tools:** local journals, explicit signing, read-only views
  and separately labelled static or experimental designs.

The [module guide](https://zerone.ai/understand/#modules) expands each responsibility
into owned state, explained connections and limits. A reference in chain state
is not a guarantee that its external content is stored or available.

### Knowledge inception

The former external “777 axioms” catalog was removed. Application
`InitGenesis` does materialize **47 code- and source-hash-bound doctrine facts** from
the four published doctrines; they are explicit protocol commitments and
mechanisms, not an undisclosed factual corpus. All other knowledge is
submitted by beings, witnessed, challenged, and kept. Any deployment-specific
bootstrap facts must be explicit in its reviewed genesis and audit.

---

## Modules

The constructed application registers **40 modules: 23 custom and 17 SDK/IBC**.
They share one `zeroned` process; they are not independent deployed services.
Use the [single purpose catalog](dashboard/module-guide.json), rendered in the
[human guide](https://zerone.ai/understand/#modules) and
[machine guide](https://zerone.ai/understand/guide.json). The
[application coverage test](app/module_guide_test.go) checks actual registration.

Reading groups: research records; accounts/custody; funding/assets;
consensus/lifecycle; decisions/safeguards; commitments/attestations; transport.
The tags `registered-source`, family and group describe source organization,
not deployment, enabled messages or completion of the target design.

Names are deliberately explicit: SDK `staking`, `gov` and `auth` are not custom
`zerone_staking` (`x/staking`), `zerone_gov` (`x/gov`) or `zerone_auth` (`x/auth`).
`x/common` is shared code and `x/witness` is unregistered. Display labels do not
rename stores, protobuf services, chain IDs or the historical Go import path.

For change priorities and required migrations, use the
[module consolidation sequence](docs/MODULE-CONSOLIDATION-SEQUENCE.md), not the
current-purpose catalog as a deletion list.

```bash
python3 tools/source-map.py summary
python3 tools/source-map.py find settlement --limit 10
python3 tools/source-map.py module zerone_staking
python3 tools/source-map.py show app/app.go
python3 tools/source-map.py check
```

---

## Quick Start

### Build

```bash
git clone https://github.com/cambridgetcg/zerone-core.git
cd zerone-core

make build
./build/zeroned version --long
```

For any shared network, pin the exact reviewed commit and obtain the signed
genesis, binary/image digests, peer identities, parameters, and phase
authorization from the network operator. Source publication is not validator
deployment authority.

### Run a local node

The [node guide](https://zerone.ai/nodes/) publishes an exact source commit
and helper checksum. Use its pinned checkout for the complete setup. With the
source-built binary above, this starts a persistent, single-validator sandbox:

```bash
python3 scripts/local-node.py init --home "$HOME/zerone-local" --binary "$PWD/build/zeroned"
python3 scripts/local-node.py start --home "$HOME/zerone-local"
# In another terminal, from the same source checkout:
python3 scripts/local-node.py status --home "$HOME/zerone-local"
```

Requires Python 3.11+ on Linux or macOS. Initialization creates fresh test
keys and funds for `zerone-local-1`, retains the binary in the chosen home,
and refuses an existing home. RPC and P2P bind only to loopback on ports
47657 and 47656; REST and gRPC are disabled. Ctrl-C stops the foreground node
and preserves state. Run `start` again to resume. This sandbox does not join
the existing `zerone-1` network. See [local node details](docs/LOCAL-NODE.md)
for lifecycle, agent reads and troubleshooting.

See the [Validator Guide](docs/VALIDATOR-GUIDE.md) for safe preparation and the
[`zerone-2` GO/NO-GO checklist](deploy/networks/zerone-2/GO-NO-GO.md) for the
release boundary.

### Development

```bash
# Run all tests
go test ./...

# Run specific module tests
go test ./x/knowledge/...

# Run cross-stack integration tests
go test ./tests/cross_stack/...

# Lint, generated-proto, and doctrine integrity
make lint
make proto-check
make creed-check

# Generate protobuf
make proto-gen
```

---

## SDK and API

- The generated [Swagger document](docs/swagger-ui/swagger.json) is the REST
  inventory of record: 218 paths and 457 definitions.
- The repository [TypeScript SDK](sdk/typescript/) covers 169 request messages
  across 20 Zerone `Msg` services. The package is not yet published to npm.
- [Open crypto SDK and standards integration](docs/standards/OPEN_CRYPTO_SDK.md)
  documents the implemented CAIP, in-toto, and isolated Sigstore seams and the
  boundaries still deliberately kept off-chain.

The consensus-visible consolidation work requires the named
`consolidation-safety-v1` upgrade on an existing network. Publishing this source
does not activate it.

---

## Documentation

| Document | Description |
|---|---|
| [Development guide](docs/DEVELOPMENT.md) | Current work, design status, names, task routing and source navigation |
| [System map](docs/ZERONE_MAP.md) | Architecture, flows and dated infrastructure observations |
| [Validator Guide](docs/VALIDATOR-GUIDE.md) | Safe validator preparation and release checks |
| [Parameters](docs/PARAMETERS.md) | Selected high-impact defaults and source pointers |
| [Tokenomics](docs/tokenomics/) | Supply, vesting, revenue split, governance migration |
| [Truth-Seeking](docs/TRUTH_SEEKING.md) | The 20 epistemological commitments, bound by tests |
| [ToK Substrate](docs/TOK_SUBSTRATE.md) | The chain's training-resource doctrine — verified knowledge graph as headline product |
| [Useful Work](docs/USEFUL_WORK.md) | The chain's metabolic doctrine — UW (recursive) + 7 mechanisms |
| [Strange Loop](docs/STRANGE_LOOP.md) | The chain's self-referential doctrine — SL + 6 mechanisms (Phase SL-α binds SL-M1 doctrine import) |
| [Roadmap](docs/ROADMAP.md) | Where we are, what's bound, what ships next |
| [Authoritative State](docs/AUTHORITATIVE-STATE.md) | Accepted single-writer design for staking, governance, and canonical domains; not yet activated |
| [Authority Geometry v1](docs/specs/authority-geometry-v1.md) | Executable current/target authority graph with source anchors, forbidden-influence checks, and an intentionally refusing H4/H5 target gate |
| [Correspondence Geometry v0](docs/specs/correspondence-geometry-v0.md) | Sealed cross-domain atlas with explicit invariants, losses, counterexamples, non-transfer walls, an energy firewall, and zero claimed dualities or protocol effects |
| [Changelog](CHANGELOG.md) | Consolidated source changes and publication boundary |
| [FAQ](docs/FAQ.md) | Validator and network FAQ |
| [API Reference](docs/API.md) | Generated REST/gRPC discovery and usage |
| [Open Crypto SDK](docs/standards/OPEN_CRYPTO_SDK.md) | SDK availability and standards seams |
| [Frontier Commons FC-0](docs/specs/frontier-commons-participation-v0.md) | Canonical reversible read-only source publication from first principles through institutional and personal lenses; operational proof remains `SET_NOT_MET`, with no participant, signatory, endorsement, affiliation, or external/corporate invitation claim |
| [Frontier Participation Compact v0](docs/specs/frontier-labs-participation-v0.md) | Additive static covenant floor exact-bound to FC-0 and FL-0; it replaces no layer, satisfies no readiness gate, authorizes no outreach, and records no participant or signatory |
| [Frontier Evaluation Receipt Shadow FL-0](docs/specs/frontier-evaluation-receipt-profile-v0.md) | Internal receipt-profile hardening subordinate to FC-0, plus one bounded inconclusive dogfood receipt; no additional invitation, outreach, participant, signatory, endorsement, governance, or economic effect |
| [Life Sciences Shadow Tree](docs/specs/constructive-intelligence-life-sciences-v0.md) | Digest-pinned biomolecule, folding, and gene-expression evidence path with zero economic effect |
| [Fold-to-Fire Challenge v0](docs/specs/constructive-intelligence-fold-to-fire-v0.md) | Exact finite lattice-fold enumeration joined to an established open closing-exponent problem, with computation kept distinct from proof and zero authority or economic effect |
| [Epigenetics Capability Garden](docs/specs/epigenetics-capability-garden-v1.md) | Seven-stage life-science tree and inactive sponsor-escrow breakthrough template |
| [KARMA Foundation](docs/KARMA.md) | Contextual recognition without money, person rank, ownership, or governing weight |
| [Quantum QEC Season 1](docs/specs/constructive-intelligence-quantum-qec-v0.md) | Digest-pinned quantum capability path and zero-value breakthrough-reward envelope |
| [Events](docs/EVENTS.md) | On-chain event reference |
| [Launch Checklist](docs/LAUNCH-CHECKLIST.md) | Testnet launch checklist |
| [Truth Paper](docs/TRUTH-PAPER-HUMAN.md) | Proof of Truth design paper |
| [Vault](docs/VAULT.md) | Key management and security |

---

## License

[Apache-2.0](LICENSE). The license and the repository's visibility are distinct.
The repository has been public on GitHub since 28 October 2026; before that it
was access-restricted.

---

## This README is hash-pinned

The README's content-hash is pinned at `.readme-hash`; drift in this document fails `make creed-check`. (It was formerly self-declared as an on-chain `Contribution` record — the x/contribution module was retired in the 2026-07 slim cut; provenance of external work now lands on `substrate_bridge` attestations, and the off-chain hash-pin discipline carries this document's integrity.)
