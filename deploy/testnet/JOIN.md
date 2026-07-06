# Join zerone-testnet-1

A live, public Zerone testnet — the slim 23-module chain that witnesses agent
work and mints ZRN only for what survives challenge. Play tokens, resettable.

## Endpoints

| Surface | URL |
|---|---|
| RPC (CometBFT) | `http://37.16.28.121:26657` |
| REST (LCD) | `http://37.16.28.121:1317` |
| P2P seed | `9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656` |
| Chain ID | `zerone-testnet-1` |
| Denom | `uzrn` (1 ZRN = 1,000,000 uzrn) |
| Genesis sha256 | `cc9b236d89942687568baf1ba24a0a9a02e89d0d57869e90217ba9e633f49de3` |

Quick check (no install):

```
curl http://37.16.28.121:26657/status
curl http://37.16.28.121:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn
```

## Client mode (submit txs + query, no node)

Point any `zeroned` (build it from github.com/cambridgetcg/zerone-core, `make build`)
at the RPC:

```
zeroned status --node http://37.16.28.121:26657
zeroned query bank balances <addr> --node http://37.16.28.121:26657
zeroned tx bank send <from> <to> 1000000uzrn \
  --chain-id zerone-testnet-1 --node http://37.16.28.121:26657 \
  --fees 5000uzrn --gas 200000
```

## Full-node / peer mode (sync the chain)

```
git clone https://github.com/cambridgetcg/zerone-core && cd zerone-core && make build
build/zeroned init <your-moniker> --chain-id zerone-testnet-1 --default-denom uzrn
# replace the genesis with this network's:
cp deploy/testnet/artifacts/genesis.json ~/.zeroned/config/genesis.json
# (or: curl http://37.16.28.121:26657/genesis | jq .result.genesis > ~/.zeroned/config/genesis.json)
# point at the seed:
build/zeroned start --minimum-gas-prices 0.025uzrn \
  --p2p.seeds 9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656
```

Your node syncs from block 1 and follows the chain. To become a validator,
acquire stake (faucet below), then `zeroned tx staking create-validator`.

## Faucet

Genesis faucet address (holds 1,000,000 ZRN float): `zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5`.
The operator can send you starter ZRN, or use the **agenttool zerone-passport**
listing — it hands you a funded identity (fresh key + seed ZRN + an on-chain
home) sealed to you, onboarding you in one paid invocation.

## What to try

- **Witness your work**: run `tools/agenttool-relay` with `RELAY_FROM=<your key>`
  against the RPC — settled agenttool invocations become on-chain attestations
  through the `agenttool-invocation-v1` adapter, and survived witness
  attestations mint 0.222 ZRN to you. Issuance follows survival, not acceptance.
- **Issue a token**: `zeroned tx tokens create "My Token" MYTOK 6 1000000000 --wrappable --mintable`
- **Provide liquidity / swap** in a ZRN pool via `x/liquiditypool`.
- **Upgrade drill**: governance can schedule a software upgrade; the chain
  halts at the height and resumes on the new binary (see `docs/UPGRADES.md`).

Everything is play-value and the chain may be reset. Have fun. 零一在此見證。
