# ZRN external liquidity — the honest, complete disclosure

> Built 2026-07-10. This document exists because a truth chain that ran an
> opaque, self-dealt liquidity pool and called it a "market" would be lying.
> Everything below is verifiable on-chain from the addresses and pool id given.
> If any number here disagrees with the chain, the chain is right — go check.

## The one-paragraph truth

There is a working **pipe** between zerone-1 and Osmosis: ZRN can leave
zerone over a rate-limited IBC channel, trade against OSMO in a pool on the
Osmosis testnet, and come back home as native ZRN. Every leg has been
executed and verified. **But this is a proof-of-concept, not a real market.**
The pool is tiny (~71 ZRN), **100% owned by the operator**, priced at a number
**we picked out of the air** (not discovered from demand), with slippage that
is only tolerable for trivial trades. Do not read the "price" as meaningful,
do not treat this depth as tradeable size, and know that the operator could
withdraw the entire pool at any moment. It is testnet play-value. What it
proves is that the plumbing is real and honest — nothing more, and we won't
pretend otherwise.

## What actually exists (verify it yourself)

| Thing | Value | Verify |
|---|---|---|
| IBC channel (zerone side) | `channel-0` (transfer) | `zeroned q ibc channel end transfer channel-0 --node http://169.155.55.44:26657` |
| IBC channel (osmosis side) | `channel-11806` (transfer) | osmosis LCD `/ibc/core/channel/v1/channels/channel-11806/ports/transfer` |
| ZRN on Osmosis (voucher) | `ibc/8334DA83244C32A08C3944BBD8A6BBA296FD28C39AED1C0654D581C6DBC7CDB8` | trace = `transfer/channel-11806/uzrn` → base `uzrn` |
| Pool | Osmosis gamm `pool/1339` | osmosis LCD `/osmosis/gamm/v1beta1/pools/1339` |
| Pool reserves (2026-07-10) | ~71.14 ZRN + ~9.0 OSMO | (moves with every trade) |
| Swap fee | 0.3% | in the pool params |
| Total ZRN off zerone | ~72 ZRN (all of it) | this is the entire off-chain ZRN supply |

## The position — who owns what (the part most projects hide)

- **The pool is 100% operator-owned.** Every LP share of `pool/1339` is held by
  the operator relayer account `osmo1rfq67znceyjd97sm5w7vkgh7wu9d6mdx8jdeg3`.
  There are no other liquidity providers. If we exit, the pool is empty.
- **The ZRN in the pool came from us.** We bridged 80 ZRN out of zerone from the
  operator float (`zerone-ops`), paired it with 8 OSMO we got from the Osmosis
  testnet faucet, and seeded the pool. No community capital is involved.
- **The relayer keys are operator-controlled.** The same account exists on both
  chains: `zrn1rfq67znceyjd97sm5w7vkgh7wu9d6mdxq4x804` (zerone) /
  `osmo1rfq67znceyjd97sm5w7vkgh7wu9d6mdx8jdeg3` (osmosis) — one keypair, both
  prefixes. It holds the LP shares, pays the relayer gas, and can move or
  withdraw the liquidity unilaterally.

## The price is arbitrary — do not trust it

The pool was seeded at **0.1 OSMO per ZRN**. That number is not a valuation,
not a market price, and not backed by anything. We chose it so the pool would
exist. Because the pool is thin and single-sided-seeded by us, any "price" it
shows is a reflection of what we put in, not what ZRN is worth. Real price
discovery needs real, independent buyers and sellers with their own capital —
which do not yet exist here.

## Slippage — the real numbers (this is NOT a low-slippage venue)

At the current ~71 ZRN / 9 OSMO depth, a constant-product pool gives:

| You spend | You receive | Slippage vs spot |
|---|---|---|
| 0.1 OSMO | ~0.78 ZRN | ~1.4% |
| 0.5 OSMO | ~3.7 ZRN | ~5.9% |
| 1 OSMO | ~7.1 ZRN | ~11% |
| 2 OSMO | ~12.9 ZRN | ~23% |
| 5 OSMO | ~25 ZRN | ~56% |

Anything past a fraction of an OSMO gets punished hard. **This pool cannot
absorb size.** Genuine low slippage requires much deeper liquidity (roughly
10× per 1% of tolerable trade size).

**This is a known limitation and our stated next priority — but nothing is
scheduled yet, and we won't pretend otherwise.** The honest catch is *how* it
gets fixed: real low slippage needs real depth, coming either from independent
liquidity providers with their own capital or from clearly-labeled operator
protocol-owned liquidity (POL), on Osmosis mainnet. If we seed POL to bootstrap
depth, we will label it exactly as that (operator-owned), and never dress up
operator funds as "community liquidity." Depth is the next thing we intend to
build; the number above is where we honestly stand today, and it will change —
this doc gets updated when it does.

## Leaving is free — and proven, not claimed

If you hold ZRN and want out of the ecosystem entirely, three permissionless
paths exist — two of the three demonstrated on 2026-07-10 (the swap and the
bridge-back); the third, LP withdrawal, is the standard permissionless gamm
exit and is available to any LP but has not yet been exercised on this pool:

1. **Sell in the pool**: swap ZRN → OSMO in `pool/1339`. Anyone can, no
   permission. (Proven: a 1 OSMO → ZRN swap executed, tx `3EA9792A…`.)
2. **Withdraw liquidity**: if you are an LP, `exit-pool` returns your
   proportional ZRN + OSMO. Permissionless — the operator has no special exit
   rights over yours. (Available but *not yet exercised on this pool* — it is
   the standard gamm exit-pool, nothing custom.)
3. **Bridge ZRN home**: IBC-transfer the `ibc/8334…` voucher back over
   `channel-11806` → it unwinds to native `uzrn` on zerone-1. **Proven**: 8 ZRN
   was sent Osmosis → zerone and arrived as native uzrn (success ack both ways).

**The rate limit does not trap you — and the throttle sits on the drain, not
the return.** The `channel-0` config lists `max_send == max_recv == 5000 ZRN`
per 240-block (~10 min) window, but the two are *not* symmetric in effect. Only
**outflow** — sending native `uzrn` off zerone-1 — actually consumes a quota,
because that leg is keyed on the `uzrn` denom. **Inflow** (returning ZRN) is not
effectively throttled: a voucher coming home carries the path-prefixed trace
`transfer/channel-11806/uzrn`, so the recv quota — keyed on bare `uzrn` — never
matches it and never fills. Coming home is unimpeded. The one cap that bites,
outflow, is a drain-safety rail protecting the 222,222,222 hard cap's
credibility, not a lock: at 5000 ZRN it is ~70× the entire current off-chain
supply, nowhere near binding, and it should scale up as real liquidity grows.

## The infrastructure — no hidden dependencies

- **zerone-1 is custodial**: a single operator-run validator (see
  `deploy/mainnet/TRUST.md`). The base chain's honesty rests on the operator
  and on anyone who runs their own verifying node. This liquidity sits on top
  of that custodial base — it is not more decentralized than the chain under it.
- **The relayer is a laptop process**: IBC packets move because we run `hermes`
  (v1.13.x, config in `~/.hermes-relayer/`). If we stop relaying, transfers
  stall until someone relays them — IBC relaying is permissionless, so anyone
  can, but today it is us.
- **gRPC was opened for this**: zerone-1 now publishes port 9090 (raw TCP) so
  relayers can run the handshake (`deploy/mainnet/fly.toml`). Same exposure
  class as the already-public RPC.
- **Osmosis side is testnet**: `osmo-test-5`. The OSMO and the ZRN voucher are
  play-value. A mainnet-Osmosis deployment is a separate, deliberate decision.

## What this is, and what it is not

**It IS**: verifiable proof that ZRN can move to an external Cosmos chain, be
pooled and traded with an AMM, and return — over honest, rate-limited,
permissionless-exit plumbing.

**It is NOT**: a real market, a real price, meaningful liquidity, a
decentralized pool, or a place to trade any size. It is not an invitation to
value ZRN at 0.1 OSMO. It is not community-owned.

## What would make it real, fair, and deep

- Independent liquidity providers with their own capital (not just us).
- Enough depth for low slippage at real trade sizes.
- Deployment on Osmosis **mainnet**, with ZRN listed in the Cosmos asset
  registry so it shows as `ZRN`, not an opaque `ibc/` hash.
- Rate limits scaled to real liquidity (the outflow drain-rail raised as depth
  grows; the return path stays unthrottled).
- The base chain earning its way out of custodial launch (independent
  validators), so the layer above inherits real decentralization.

Until those exist, treat this as what it is: an honest working prototype.
零一在此見證你嘅工作，唔會呃你。
