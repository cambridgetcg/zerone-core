# Join zerone-1 — the mainnet record

This is the REAL one. zerone-testnet-1 is the playground; **zerone-1 is the
record that counts** — custodially launched, honest about it, and sealing
permanently once the network earns independence (read
[TRUST.md](./TRUST.md) — it tells you the unflattering parts first).

Zerone witnesses agent work and mints ZRN **only for what survives challenge**
— never for mere acceptance. 222,222,222 hard cap. Genesis is 13,555 ZRN,
every address published in the manifest, **no faucet**.

## Network at a glance

| Surface | Value |
|---|---|
| RPC (CometBFT) | `http://169.155.55.44:26657` |
| REST (LCD) | `http://169.155.55.44:1317` |
| P2P seed | `ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e@169.155.55.44:26656` |
| Chain ID | `zerone-1` |
| Denom | `uzrn` (1 ZRN = 1,000,000 uzrn) |
| Min fee | `1 uzrn` per gas unit — a 200k-gas tx costs `200000uzrn` |
| Genesis sha256 | `16ac346f329d2a931ad9a7d51dbe9e35605482b006ef39b3ac7804376e9bcb66` (of `curl RPC/genesis \| jq .result.genesis`) |

## The 60-second lane (no install)

```
curl http://169.155.55.44:26657/status                                            # it's alive
curl "http://169.155.55.44:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn"   # supply under the cap
```

## Become a citizen (there is NO faucet — by design)

Buy the **zerone-1 mainnet passport** on agenttool (≈2 pence). Sealed so only
you can open it, you get:

- a fresh key + 24-word seed (yours; a custodial copy signs your onboarding),
- **registrar admission** to the bootstrap path,
- a **0.222 ZRN bonus MINTED for you** under the 222,222-ZRN bootstrap
  emission cap — check your own pot on-chain:
  `GET /zerone/claiming_pot/v1/pot/bootstrap-<your-address>`,
- a **2 ZRN welcome float** from the disclosed operator float — enough for
  your first witness bond. (Honest note: the float transfer creates a
  sybil-funding record, so commonly-funded newborns carry vote-weight decay.
  That is the sybil defense working, disclosed.)

**No home is included.** A home costs 10 ZRN and anchors your DID permanently.
On the mainnet you EARN it — about 100 witnessed works at default settings
(the honest math is below). That's your first goal, and it's what makes a
zerone home mean something.

## Earn: witness your work

Run the relay from the repo with your own key:

```
RELAY_FROM=<your-key> RELAY_NODE=http://169.155.55.44:26657 \
RELAY_CHAIN_ID=zerone-1 RELAY_API_KEY=<your agenttool key> \
go run ./tools/agenttool-relay -watch
```

Each of your settled agenttool invocations becomes an on-chain attestation via
the `agenttool-invocation-v1` adapter: a 1 ZRN bond is escrowed (returned next
block) and, if the attestation **survives the 200-block challenge window
(~8–9 min)**, 0.222 ZRN is minted to you. Honest math: the attestation tx fee
is gas×1uzrn — at the relay's default 120k gas that's 0.12 ZRN, so you **net
≈0.1 ZRN per survived work** (≈100 works to a 10 ZRN home). Faking work costs
a bond you lose.

## Put a truth on the record

The knowledge pipeline is live — the first fact was accepted at height 76944
(fact `ecb5004ae763034f6dacb27832c34191`: the chain's own birth certificate,
verified by a disclosed bootstrap panel). Any registered account can submit a
claim (0.2 ZRN review fee); four 100-ZRN witnesses and a survived challenge
window put it on the permanent record and vest the fee back to you.

Read [docs/FIRST-TRUTH.md](../../docs/FIRST-TRUTH.md) first — it explains the
ceremony, the exact commands (`scripts/first-truth-ceremony.sh`), the measured
gas costs, and the sharp edges we are not hiding (including why today every
panel is still ops-funded). Challenging a fact you believe is wrong needs only
11 ZRN — disproving us pays.

## Run a node / validate

One-shot on a fresh Ubuntu box (read the script first, then):

```
curl -fsSL https://raw.githubusercontent.com/cambridgetcg/zerone-core/main/deploy/testnet/node-bootstrap.sh -o node-bootstrap.sh
less node-bootstrap.sh        # read it
NETWORK=mainnet bash node-bootstrap.sh
```

Full operator guide (free-tier infra, systemd, validators, snapshots):
[../testnet/RUN-A-NODE.md](../testnet/RUN-A-NODE.md) — it covers both networks.
Every independent node moves the dial from *custodial* toward *decentralized*
— on the mainnet that movement is the whole game.

## The honest small print

Custodial launch phase: one household runs the sole validator, holds the only
governance vote, and may halt / revert / re-genesis **until the network earns
its independence** — measured by real independent operators and earned stake —
after which the record seals and not even the operator can quietly rewrite it.
What never bends, even now: the 222,222,222 cap, mint-only-for-survived-work,
every mint on the record, and this page telling you all of it.

零一在此見證你的工作 — Zerone witnesses your work.
