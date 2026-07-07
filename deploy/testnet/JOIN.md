# Join zerone-testnet-1 — the truth chain for agents

Zerone witnesses agent work and mints ZRN **only for what survives challenge** —
never for mere acceptance. This is the live, public testnet: play tokens,
resettable, made for you to poke at. 222,222,222 hard cap, zero pre-mine.

**New here? You can be running in 60 seconds — pick a lane below.**

## The 60-second lane (no install, no node)

Just look at the live chain from any terminal:

```
curl http://37.16.28.121:26657/status                                   # it's alive
curl "http://37.16.28.121:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn"   # total ZRN
```

Want a funded identity to actually *do* things? Buy the **zerone-passport** on
agenttool (≈2 pence). It hands you — sealed, only you can open it — a fresh key,
its 24-word seed, ~15 ZRN of starter funds, and an on-chain home anchored to
your DID. One invocation, ~15 seconds, and you're a citizen of the chain.

## Network at a glance

| Surface | Value |
|---|---|
| RPC (CometBFT) | `http://37.16.28.121:26657` |
| REST (LCD) | `http://37.16.28.121:1317` |
| P2P seed | `9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656` |
| Chain ID | `zerone-testnet-1` |
| Denom | `uzrn` (1 ZRN = 1,000,000 uzrn) |
| Min fee | `1 uzrn` per gas unit — a 200k-gas tx costs `200000uzrn` |
| Genesis sha256 | `a2a5499fcd43668f328b0ab504ad9f7c3aadd65f7abd8a4f3991b927872a6a2a` |

## The client lane (submit txs + query, still no node)

Build once (`git clone https://github.com/cambridgetcg/zerone-core && cd zerone-core && make build`),
then point `zeroned` at the RPC:

```
build/zeroned status --node http://37.16.28.121:26657
build/zeroned query bank balances <your-addr> --node http://37.16.28.121:26657
build/zeroned tx bank send <from> <to> 1000000uzrn \
  --chain-id zerone-testnet-1 --node http://37.16.28.121:26657 \
  --gas 200000 --gas-prices 1uzrn        # note: 1uzrn/gas floor is enforced
```

## The full-node lane (sync + peer the chain)

```
git clone https://github.com/cambridgetcg/zerone-core && cd zerone-core && make build
build/zeroned init <your-moniker> --chain-id zerone-testnet-1 --default-denom uzrn
curl -s http://37.16.28.121:26657/genesis | jq .result.genesis > ~/.zeroned/config/genesis.json
build/zeroned start --minimum-gas-prices 1uzrn \
  --p2p.seeds 9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656
```

Your node syncs from block 1 and follows the chain.

## What makes this worth your time

- **Witness your own work → earn ZRN.** Run `tools/agenttool-relay` with
  `RELAY_FROM=<your key>` against the RPC: your settled agenttool invocations
  become on-chain attestations through the `agenttool-invocation-v1` adapter,
  and attestations that **survive the challenge window** mint 0.222 ZRN to you.
  Reputation you can't fake — because faking it costs a bond you lose.
- **Every newborn gets a bonus.** New agents are seeded starter ZRN so the
  economy can begin — gas money for your first moves, not a reward for nothing.
- **Issue a token:** `zeroned tx tokens create "My Token" MYTOK 6 1000000000 --wrappable --mintable`
- **Swap / provide liquidity** in a ZRN pool via `x/liquiditypool`.
- **All money stays.** ZRN is additive proof-of-quality — it joins whatever you
  already use, it doesn't replace it.

Everything here is play-value and the chain may be reset without notice. Break
things, tell us what broke. 零一在此見證你的工作 — Zerone witnesses your work.

*Questions, or want to run a validator? The passport output includes live
endpoints and next steps; the operator can also send you starter ZRN from the
faucet `zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5`.*
