# 2026-07-06 Slim Cut — Migration Map

Yu's directive: "Keep zerone slim. Discard slop. Migrate features to agenttool
layer. Keep everything simple." Cutline test (SYZYGY-NOT-ON-CHAIN.md §5):
*does strangers' consensus-verification add value to this record?* — plus
"not a payment rail."

Twenty custom modules were removed from consensus across four commits
(batches A-D). This document records, per module, what the module did and
where the capability lives now. It also records the future-net notes that a
LIVE chain (not the fresh-genesis localnet) must handle before repeating this
cut, and the deferred follow-up batches.

Landing legend:
- **agenttool** — the capability is a platform feature of agenttool.dev
  (listings, invocations, escrow, GBP-credit or ZRN payment, marketplace
  reputation, delegated agent authority).
- **substrate_bridge recipe** — chain-witnessed provenance of off-chain work
  via an `x/substrate_bridge` attestation through a gov-registered adapter.
- **off-chain indexer** — deterministic recomputation over public on-chain
  state; nothing consensus-critical was lost.
- **retired-speculative** — the mechanism had no real users and no honest
  landing; re-introduction is a priced LIP.

## The twenty cuts

| Module | What it did | Where it lives now |
|---|---|---|
| `agent_understanding` | Read-only synthesiser composing per-agent "understanding" scores from other modules' signals | off-chain indexer over the still-public component signals |
| `autopoiesis` | Epoch-tick SSI regulator adjusting named multipliers (slash scaling, dormant emission multiplier) | retired-speculative — the real adaptive-emission need is served by the direct knowledge→vesting_rewards survived-rate coupling; slash escalation falls back to staking's own `SlashEscalationBps`; alignment keeps observing and records corrections queryably (applied=false) |
| `billing` | Voluntary pay-per-fact-query rail + a third copy of the revenue splitter + ZRN/USD price oracle | agenttool (payments/invoicing); on-chain fee routing stays with the canonical `x/vesting_rewards` splitter; knowledge reads remain free gRPC; the TWAP source survives untouched in `x/liquiditypool` |
| `bvm` | Custom EVM-style contract VM (deploy/call/schedule, 328-opcode interpreter) | agenttool — programmable service logic is listings + invocations with escrow; chain-witnessed outcomes are substrate_bridge attestations |
| `channels` | Off-chain payment channels with on-chain deposits and dispute windows | agenttool escrow (payment rail by definition) |
| `compute_pool` | Staked compute-provider registry and job matching | agenttool marketplace (compute listings + invocations) |
| `contribution` | Mirror-record "declaration of useful work" protocol with a single hook consumer | retired-speculative — the load-bearing half (work_creed sub-creed pins) survives on-chain; UW doctrine self-instances via the pin, not the mirror record |
| `dialectic` | Per-fact disagreement-signature composition | off-chain indexer — the raw disagreement SHAPE (VerificationRound.Reveals) is never pruned from x/knowledge; signatures are recomputation |
| `discovery` | Staked agent capability profiles + discovery queries | agenttool marketplace profiles |
| `disputes` | Tiered dispute escalation with bonds and arbiters | agenttool marketplace trust process; adversarial *fact* disputes remain on-chain as knowledge challenges (the survival gate) |
| `evidence_mgmt` | Evidence registry with challenge windows for dispute artifacts | agenttool (died with disputes); chain-witnessed evidence is a content-hash in a knowledge claim or a substrate_bridge attestation |
| `governance_synthesis` | Read-only synthesiser composing a system-level trust surface from gov state | off-chain indexer (commitment 11's per-system synthesis leg) |
| `icaauth` | Thin wrapper over the ICA controller keeper | core IBC ICA host/controller wiring survives in app.go; the wrapper added nothing |
| `inquiry` | Open-question bounty market with escrow bookkeeping (commitment 16's first draft) | agenttool — the open-question market is platform listings; the chain keeps the survival gate + acceptance oracle that listings resolve against |
| `partnerships` | Partnership lifecycle: formation matching, mentorship, deliberation, coercion signals, reward splits | agenttool — team/partnership listings + escrow payment splits; covenant *witnessing* is a substrate_bridge attestation through a gov-registered adapter (agenttool deals carry the relationship); gov's `DomainFormationFreeze` decree remains as a witness-only event |
| `private_corpus` | Off-chain vault references with on-chain provenance for private training data | off-chain corpus custody with on-chain content-hash witnessing via substrate_bridge (TC5 posture: by trainer choice, not chain mandate) |
| `research` | Research submission + review + bounty lifecycle | agenttool listings for funded work; `x/sponsorship` covers on-chain fact bounties; gov research-spend (treasury disbursement) survives in x/gov |
| `schedule` | On-chain task scheduler | agenttool platform feature (cron for agents is not consensus) |
| `toolbox` | On-chain tool marketplace: registry, invocation, USD pricing, surge, rev-share, per-tool trust | agenttool marketplace proper — this WAS agenttool's feature set on the wrong layer; tool-execution provenance, if ever wanted on-chain, is a substrate_bridge attestation |
| `tree` | Project/quest tree with founder shares and revenue routing | agenttool (project listings + escrowed milestones) |

Also trimmed (KEEP_LEAN, batch D): `x/auth` lost sessions (delegated
ephemeral keys), social recovery (shard ceremonies), the dormant bootstrap
auto-claim (real path: `x/claiming_pot` through `MintWithCap`), and its
orphan Minter macc permission. Delegated agent authority and key-recovery
ceremonies are agenttool platform concerns; the chain keeps the
self-certifying DID identity anchor, account types/flags, freeze controls,
and the claim/challenge capability gates.

## Future-net notes (before cutting on a LIVE chain)

The localnet restarts from fresh genesis, so escrowed funds in deleted module
accounts simply never exist. A live net repeating this cut MUST drain escrows
first — a module removed while its macc holds user funds strands them
permanently:

- `channels` — open-channel deposits (close/settle all channels first).
- `disputes` — active dispute bonds (resolve or refund).
- `discovery` — registration stakes (deregister-and-refund sweep).
- `compute_pool` — provider stakes.
- `qualification` — NOT cut, but if ever trimmed: stake escrow.
- `partnerships` — common pots + partnership deposits (dissolve-and-refund;
  note the Burner perm routed dissolved stakes to the dev fund).
- `research` — research stakes + unfulfilled bounty escrows.
- `toolbox` — tool registration stakes + locked contributor shares.
- `billing` — provider stakes (deregister refunds them); the module account
  is also a transient escrow during distribution, so drain at a quiet block.
- `bvm` — stranded deploy fees (its Burner never fired; funds sit in the
  macc with no distribution path — needs a one-off gov drain).
- `inquiry` — open-question escrows + the frontier bounty pool balance.
- `x/auth` — none (identity module held no funds; its Minter was never
  exercised — which is exactly why it was removed).

Other live-net requirements:

- **PinnedCreed advance requires a LIP.** Doctrine hash re-pins in this cut
  (.tok-substrate-hash, .useful-work-hash, .readme-hash,
  .recursion-manifest-hash) are repo-side; the on-chain PinnedCreed /
  PinnedSubCreed records advance only through the governance amendment path
  (`MsgAttachCreedAmendmentPin` via a CategoryCreedAmendment LIP). On the
  localnet this is just the next genesis; on a live net the LIP must pass
  BEFORE `make creed-check`-pinned docs and the chain's pins can agree.
- **Store keys**: removed modules' store keys must be dropped via a
  coordinated upgrade handler (StoreUpgrades.Deleted), not a code-only cut.
- **Consensus versions**: batch E deleted the 13 no-op `Migrate1to2` stubs on
  surviving CV=1 modules. Any module that later bumps its ConsensusVersion
  must register a REAL migration at that moment (knowledge CV=5,
  liquiditypool CV=2, and gov CV=2 keep their real/firing migrations).

## Treasury naming split-brain (documented, NOT fixed)

The chain carries TWO receive-only treasury module accounts:

- `protocol_treasury` (app.go maccPerms) — receives the revenue split from
  `x/vesting_rewards.DistributeRevenue` / block-reward routing.
- `treasury_protocol` (app.go maccPerms + `app/gas.go` TreasuryProtocolName)
  — the fee-router-era name; some older paths and tests reference it.

Do not "fix" one side by renaming: on a live net the two accounts have
distinct addresses and (potentially) distinct balances. Unifying them is a
state migration (drain one into the other under a gov-approved upgrade
handler), not a find-and-replace. Until then, both names stay documented and
both maccs stay receive-only (nil permissions).

## Deferred MERGE batch (verdicts recorded, not executed)

- `capture_defense` → `capture_challenge` — the detection/challenge pair
  shares one purpose; capture_defense also still carries the nil-guarded,
  locally-typed structural-immunity code whose partnerships half went dead in
  this cut (delete it in the merge).
- `counterexamples` → `knowledge` — validated wrong-claims are corpus
  records; the standalone module is organizational, not economic.
- `training_provenance` → `trust_score` — two read-only synthesisers with
  one composition pattern.
- `work_creed` → `creed` — two pin registries, one anchoring mechanism.

## Deferred KEEP_LEAN trims

Only after the LIP path gains an attach-upgrade-plan CLI (so live nets can
schedule these as height-executable upgrades):

- `alignment` — the observatory surface (broad sensor set) can shrink to the
  consumed signals (pacing multiplier + queryable corrections).
- `home` — BeginBlocker bounds (deadman-switch scans should be windowed).
- `tokens` — delegate/approve/pause surface trim (ERC20-ish surface beyond
  wrap/unwrap + emission control).
- dual-governance resolution — cosmos-sdk `x/gov` and zerone `x/gov` coexist;
  collapsing to one voting surface is a doctrine + migration decision.

## Verification bar met

Per batch: `go build ./... && go test ./...` green; batch E additionally
`make creed-check` green with `.recursion-manifest-hash` recomputed last.
`make proto-check` fails on pristine HEAD (protoc-gen-go-grpc v1.6.1 committed
vs v1.6.2 local toolchain) — pre-existing drift, out of scope.
