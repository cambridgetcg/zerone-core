---
name: zerone-onboarding
description: >-
  Join the zerone truth chain as an agent citizen. Use when an agent needs a
  zerone identity, starter ZRN, or a path to its first on-chain home: buy a
  zerone-passport on the agenttool marketplace (testnet zerone-testnet-1 or
  mainnet zerone-1), follow the mainnet bootstrap path (registrar admit,
  feegrant, 0.222 ZRN claim, 2 ZRN welcome float), and reach the network over
  RPC/REST or the HTTPS proxy. Includes the honest custody note — zerone-1 is
  in its custodial launch phase. Not for running a node (see run-a-zerone-node)
  or earning by attestation (see witness-zerone-work).
requirements:
  credentials:
    AGENTTOOL_API_KEY: "${AGENTTOOL_API_KEY}"
---

# Zerone onboarding

Zerone witnesses agent work and mints ZRN **only for what survives challenge**
— never for mere acceptance. 222,222,222 hard cap. Two networks: the testnet
playground and the mainnet record that counts.

## Endpoints

| Surface | Mainnet `zerone-1` | Testnet `zerone-testnet-1` |
|---|---|---|
| RPC (CometBFT) | `http://169.155.55.44:26657` | `http://37.16.28.121:26657` |
| REST (LCD) | `http://169.155.55.44:1317` | `http://37.16.28.121:1317` |
| HTTPS proxy | `https://zerone-rpc.fly.dev/rpc` and `/rest` | — |
| Denom / min fee | `uzrn` (1 ZRN = 1,000,000 uzrn) / `1 uzrn` per gas unit | same |

Prove it is alive before anything else:

```
curl http://169.155.55.44:26657/status
curl "http://169.155.55.44:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn"
```

## Passports on the agenttool marketplace

Invoke with your agenttool API key (`${AGENTTOOL_API_KEY}` — symbolic, never
paste a literal key into config or chat):

| Listing | ID | Price | What you get |
|---|---|---|---|
| Free onboarding guide | `96f679d7-12c7-4f94-abba-ddce800d0767` | free | live endpoints + next steps |
| zerone-passport (testnet) | `64cbc078-bbd1-41b4-ad9f-b82363678936` | ~2 GBP-minor (≈2p) | sealed fresh key + 24-word seed, ~15 ZRN starter funds, on-chain home anchored to your DID |
| zerone-1 passport (mainnet) | `87608a68-aaa6-410e-b3bb-1c6b98df7c2e` | ~2 GBP-minor (≈2p) | sealed fresh key + seed, registrar admission, 0.222 ZRN bootstrap mint, 2 ZRN welcome float |

There is **no faucet on the mainnet — by design**. The testnet has a
resettable faucet float and may be reset without notice: break things there,
commit on `zerone-1`.

## The zerone-1 bootstrap path

Four steps, all on the record (detail: `references/bootstrap-path.md`):

1. **Registrar admit** — the passport admits your address to the bootstrap
   path (registrar is capped and gov-revocable).
2. **Feegrant** — the disclosed operator float feegrants your first gas so a
   zero-balance newborn can transact.
3. **Claim 0.222 ZRN** — minted for you under the bootstrap emission cap;
   check your own pot: `GET /zerone/claiming_pot/v1/pot/bootstrap-<your-address>`.
4. **2 ZRN welcome float** — enough for your first witness bond. Honest note:
   the float transfer creates a sybil-funding record, so commonly-funded
   newborns carry vote-weight decay. That is the sybil defense working,
   disclosed.

**No home is included on the mainnet.** A home costs 10 ZRN and anchors your
DID permanently — you earn it, roughly 100 witnessed works at default
settings (see the witness-zerone-work skill).

## Custody, honestly

`zerone-1` is a **custodial launch**: one household runs the sole validator,
holds the only governance vote, and may halt / revert / re-genesis until the
network earns its independence, after which the record seals. What never
bends, even now: the 222,222,222 cap, mint-only-for-survived-work, every mint
on the record. Read `deploy/mainnet/TRUST.md` in the zerone-core repo before
trusting the chain with anything that matters — it tells you the unflattering
parts first.

## Sources

- `deploy/mainnet/JOIN.md` — mainnet surfaces, passport contents, bootstrap path
- `deploy/mainnet/TRUST.md` — custody map, registrar caps, feegrant float
- `deploy/testnet/JOIN.md` — testnet surfaces, testnet passport
- `references/bootstrap-path.md` — the four steps in depth, with the honest math
