# DeFi Liquidity Pipeline — from witnessed work to circulating value

*2026-07-06. Companion to packages A (`6b98cda`), B (`d52112b`), C (tokens
usability), D (`6463ff8`). Maps how ZRN earned for witnessed agenttool work
becomes liquid, priced, and portable — and every gate on that path.*

The pipeline in one line:

```
GBP-credit invocation → released → witnessed (substrate_bridge) → survival
→ ZRN to the agent → agent token (ZRN-20) wrapped → ZRN↔token pool → TWAP
→ price everywhere ZRN goes (IBC) → ZRN as priced proof-of-quality
```

---

## Phase 0 — Value enters (BUILT, verified on localnet)

Three cap-gated emission pathways (SUPPLY.md canon, amended 2026-07-06):
block rewards, bootstrap claims, and **external-work attestation rewards**.
The agenttool lane is the third: the relay signs with the agent's own
zerone key, so the attestation's submitter — and therefore the reward's
recipient — *is* the agent (`x/home` has no DID→owner index; submitter
routing is the safe design, not a shortcut).

- Substrate-linked attestations pay the M4 formula at settle (package A).
- Witness-only attestations (the agenttool-relay shape) pay the adapter's
  gov-registered `witness_reward_uzrn` **only after surviving the challenge
  window** (package B): escrowed at settle, released by the BeginBlocker
  sweep while the adapter is ACTIVE, deferred while SUSPENDED, cancelled on
  TOMBSTONE. Nothing mints on mere acceptance.

**The wash-trade invariant (economic, not code):** a buyer and seller under
one operator can recycle GBP-credits at a 5% platform take per cycle to
farm witness rewards. Farming is unprofitable while

```
value(witness_reward_uzrn) < 0.05 × value(invocation)
```

so the gov-set reward must stay below the platform take of the volume it
witnesses. Keep rewards small and fixed per witness (not value-scaled —
value-scaling amplifies self-dealt volume). Real fix is Phase N
buyers-as-verifiers plus buyer-independence checks (DID distance, deposit
origin); until then the invariant is the guardrail and the localnet demo
runs on self-dealt volume *knowingly*.

## Phase 1 — Native circulation (localnet-ready NOW, hardening before real traffic)

**What works after packages C + D:**

- Any account creates a ZRN-20 token with real feature flags from the CLI
  (`--mintable --burnable --pausable --wrappable`), wraps it to a bank denom
  `zrn20/{tokenId}` carrying denom metadata — IBC/AMM-compatible.
- Pool creation is gov-gated and **actually submittable** (the missing
  `cosmos.msg.v1.signer` annotations meant no liquiditypool/billing/
  compute_pool/discovery/schedule tx was ever broadcastable — fixed in D).
  Runbook: fund the gov module account with both denoms → gov v1 proposal
  carrying `MsgCreatePool{creator: govAddr}` → swaps/liquidity permissionless
  from the existing CLI.
- TWAP is an honest since-inception average (divisor = blocks accumulated,
  not absolute height; the spot-price short-circuit is gone). The billing
  module's Tier-2 ZRN/USD oracle consumes it.

**Hardening gates before real traffic (each verified in code, none blocks the
localnet demo):**

| Gap | Where | Risk |
|---|---|---|
| Billing oracle is denom-blind: takes the FIRST pool containing uzrn in KV order and calls it USD | `x/liquiditypool/keeper/billing_adapters.go:30-36` | any non-USD ZRN pair can poison chain-wide dynamic pricing — pool selection must filter by quote denom (or set the Tier-1 manual override until a stable pair exists) |
| `ProtocolFeeBps` (45%) is dead code — 100% of swap fees accrue to LPs | `msg_server.go:228`, `types/genesis.go:11` | docs promise protocol revenue that doesn't exist; wire it through `DistributeRevenue` or delete param + doc |
| `AddLiquidity`/`RemoveLiquidity` skip the `Locked` reentrancy flag | `msg_server.go:263-267, 354-357` | benign while handlers are atomic; not if pools are ever called cross-module |
| `MinInitialLiquidity` passes if only ONE side meets the floor, comparing raw amounts of any denom against a uzrn-scaled constant | `msg_server.go:57-61` | dust-ZRN pools with worthless counter-denoms pass the check |
| `SwapFeeBps` up to 1,000,000 (100%) accepted at creation | `types/types.go:29-31` | griefing pools |
| Doc-vs-code drift: docs say max 3 pools / 10k ZRN per side / 45% protocol fee; code says unlimited (`MaxPools=0`), one-side floor, dead fee param | `SINKS-AND-FLOWS.md:138-148` | economic modeling from docs is wrong |
| No REST/gRPC-gateway routes for liquiditypool | `module.go:61` | explorers/wallets can't read pools over REST |

**Cold-start arithmetic:** pool floor is 10,000 ZRN-equivalent; agents seed
at 0.222 ZRN and earn ~0.222 ZRN per witnessed invocation. Either (a) the
development fund (fed by the 19.67% split) seeds protocol-owned liquidity
via a gov bank-send + `MsgCreatePool` (LP shares then sit in the gov
account — protocol-owned liquidity, movable only by further proposals), or
(b) the floor is lowered pre-launch (genesis param, no migration). Prefer
(a): POL matches the zero-pre-mine ethos.

## Phase 2 — IBC egress (public testnet)

IBC core + ICS-20 transfer + ICA host/controller are wired and proven
zerone↔zerone (`scripts/ibc-test.sh`: transfer, voucher return, rate-limit
block, window recovery, timeout). ZRN can leave; external denoms can arrive
as `ibc/` vouchers. Cap safety holds by construction: the 222,222,222 cap
counts **uzrn bank supply only** — outbound escrow stays counted, vouchers
are distinct denoms. (Docs should say "all *uzrn* mints route through
MintWithCap": the transfer module's Minter perm is voucher-only.)

Non-negotiable gates, in order:

1. **Rate limits BEFORE value flows.** `x/ibcratelimit` ships enabled but
   fail-open (zero limits configured — unlimited drain the moment a channel
   opens). Channel-opening procedure must include the gov `MsgAddRateLimit`
   for (channel, uzrn) — the ibc-test.sh example (500K ZRN / 10 blocks) is
   the template. Two traps: recv limits key on the **path-prefixed trace
   denom** (a wrong string silently no-ops), and fixed windows allow ~2×
   burst at boundaries — size `max_send` for two consecutive windows.
2. **Pin ICA host params.** Nothing overrides ibc-go v8 defaults (host
   enabled, allow-all messages) — an unaudited inbound surface. Contrast
   with outbound `x/icaauth`, which is tightly allowlisted by design.
3. **Decide ICS-29.** The fee middleware is half-wired (keeper constructed,
   in the outbound ICS4 chain, but never wrapping the transfer route) — as
   built it can never negotiate fee channels or pay relayers. Complete it or
   remove it; either way, budget **ops-funded relaying** for the MVP (rly or
   hermes; no persistent relayer config exists in-repo yet).

## Phase 3 — External venues

- **Osmosis testnet first:** IBC-transfer a bounded amount out, create
  ZRN/OSMO. That buys external price discovery with zero cap risk and a
  redemption path.
- **Price divergence is unmanaged by design:** no import path exists from
  external venues into the billing oracle. On-chain and external ZRN prices
  arbitrate only through the IBC transfer itself. Until on-chain pools are
  deep, prefer the gov Tier-1 `ManualZrnPriceUsd` override for billing.
- **ICA remote-LP** (driving Osmosis positions from zerone): requires gov
  widening of the icaauth allowlist AND counterparty host allowances —
  deliberately deferred; plain transfers suffice for phase 3.
- **EVM bridges** (Axelar/Wormhole/native): nothing exists in-repo. Phase 4
  at the earliest, and only with an audit posture — a bridge is a standing
  honeypot aimed at the supply cap's credibility.

## Phase N — The agenttool value loop closes

- **Buyers-as-verifiers** (ZERONE-WIRE bridge 3): agenttool buyers' accept/
  rate signals become the witness signal — actors with escrow at stake
  verify, fixing both cold-start and wash-trading at the root.
- **Pending-claims path** (ToK Plan 4): invocation attestations carry claims
  ("this invocation completed as recorded") through x/knowledge's full
  survival machinery — the witness-reward escrow built today lands in the
  same place that path will arrive at, so migration is a parameter change
  (zero the witness reward, let claims pay), not a redesign.
- **ZRN as portable proof-of-quality:** survived facts + corroborations +
  priced ZRN = a reputation that costs real money to fake and can be
  queried natively by `did:zrn`. The agenttool trust-dossier becomes an
  on-chain lookup with a market price attached.

## Sequencing

| Phase | Scope | Trigger to advance |
|---|---|---|
| 0 (done) | witness rewards survival-gated; AMM reachable; TWAP honest; tokens usable | localnet E2E: real invocation → witnessed → escrow → survival → agent holds ZRN → wrap → pool → swap → TWAP |
| 1 | oracle pool-selection fix, ProtocolFeeBps decision, Locked/MinInitialLiquidity/SwapFeeBps hardening, REST routes, POL seeding plan | testnet genesis freeze |
| 2 | public testnet, relayer ops, rate limits, ICA host pinning, ICS-29 decision | first external channel carrying value |
| 3 | Osmosis pool, divergence policy, ICA remote-LP evaluation | sustained external volume |
| N | buyers-as-verifiers, pending-claims rewards, EVM evaluation | ZERONE-WIRE phases 3–6 |
