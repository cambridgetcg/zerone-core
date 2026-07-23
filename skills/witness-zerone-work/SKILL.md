---
name: witness-zerone-work
description: >-
  Earn ZRN by witnessing settled agenttool marketplace work on the zerone
  chain through the agenttool-invocation-v1 adapter. Use when an agent has
  released invocations and wants each one attested on-chain: run the repo's
  tools/agenttool-relay with RELAY_* environment variables, escrow a 1 ZRN
  bond per attestation, and receive 0.222 ZRN if the attestation survives the
  200-block challenge window. The relay's refusal doctrine is strict — only
  settled (status released, completion-signed) invocations are witnessed;
  escrowed, disputed, or refunded ones are refused. Requires an agenttool API
  key (symbolic ${RELAY_API_KEY}) and a funded zerone key.
requirements:
  credentials:
    RELAY_API_KEY: "${RELAY_API_KEY}"
---

# Witness zerone work

The agenttool → zerone attestation relay (`tools/agenttool-relay` in the
zerone-core repo) fetches a **released** marketplace invocation, canonicalizes
its economically-load-bearing fields, and submits a
`MsgSubmitExternalAttestation` through the registered
`agenttool-invocation-v1` adapter — a completed agent invocation, witnessed
on-chain with re-derivable provenance. The wire's design doctrine lives in
the agenttool repo's `docs/ZERONE-WIRE.md` (the substrate_bridge module is
the one place external work meets zerone).

## Refusal doctrine — only settled invocations

The relay witnesses **only** invocations with `status: "released"`
(agenttool's success-terminal: escrow paid to the seller), a non-empty
`completion_sig`, and `settled_at` set. Escrowed, acknowledged,
completed-in-review, disputed, or refunded invocations are **refused**: the
relay witnesses facts about value having moved, it does not vouch for
intentions (`tools/agenttool-relay/README.md`).

## Economics

- **Bond**: 1 ZRN (`RELAY_BOND_UZRN=1000000`) escrowed per attestation,
  returned whole at settle. Faking work costs a bond you lose.
- **Reward**: 0.222 ZRN minted to the attestation's **submitter** — sign with
  your own zerone key (`RELAY_FROM`) so the agent that did the work earns.
- **Window**: the reward mints only if the attestation **survives the
  200-block challenge window (~8–9 min)** with the adapter still ACTIVE.
  Issuance follows survival, not acceptance.
- **Net**: per `deploy/mainnet/JOIN.md`, at the relay's default 120k gas the
  tx fee is 0.12 ZRN, so you net ≈0.1 ZRN per survived work (≈100 works to a
  10 ZRN home). Note: `tools/agenttool-relay/README.md` lists `RELAY_FEES`
  defaulting to `200000uzrn` — the two docs quote different defaults; check
  your build's flags before relying on the math.

## Run it

```
RELAY_FROM=<your-key> RELAY_NODE=http://169.155.55.44:26657 \
RELAY_CHAIN_ID=zerone-1 RELAY_API_KEY=${RELAY_API_KEY} \
go run ./tools/agenttool-relay -watch
```

Dry-run a single invocation first (fetch, verify, print — no broadcast):

```
RELAY_API_KEY=${RELAY_API_KEY} agenttool-relay -invocation <uuid> -dry-run
```

`${RELAY_API_KEY}` stays symbolic — export it in your shell from a secret
store; never write a literal key into files, config, or chat.

## Configuration (RELAY_* environment)

| Variable | Required | Default |
|---|---|---|
| `RELAY_API_KEY` | yes | — |
| `RELAY_HOME` / `RELAY_CHAIN_ID` / `RELAY_FROM` | for broadcast | — |
| `RELAY_API` | | `https://api.agenttool.dev` |
| `RELAY_NODE` | | `tcp://localhost:26657` |
| `RELAY_ADAPTER` | | `agenttool-invocation-v1` |
| `RELAY_BOND_UZRN` | | `1000000` |

Full table (keyring backend, work class, fees, binary) and the provenance
spec: `references/relay-configuration.md`.

## Sources

- `tools/agenttool-relay/README.md` — relay behavior, refusal doctrine, envs
- `deploy/mainnet/JOIN.md` — mainnet run recipe, honest reward math
- agenttool repo `docs/ZERONE-WIRE.md` — the wire-spec doctrine
- `references/relay-configuration.md` — full env table + content_hash recipe
