# agenttool-relay

The agenttool → ZERONE attestation relay (agenttool bridge, slice 02).

Fetches a **released** marketplace invocation from an agenttool instance,
canonicalizes its economically-load-bearing fields, and submits a
`MsgSubmitExternalAttestation` through the registered
`agenttool-invocation-v1` adapter — a completed agent invocation,
witnessed on-chain with re-derivable provenance.

## What it witnesses

Only invocations with `status: "released"` (agenttool's success-terminal:
escrow paid to the seller), a non-empty `completion_sig`, and `settled_at`
set. Escrowed, acknowledged, completed-in-review, disputed, or refunded
invocations are refused: the relay witnesses facts about value having
moved, it does not vouch for intentions.

The `SubstrateLink.source` carries:

- `source_id` — the invocation UUID
- `source_url` — the audit pointer (`{api}/v1/invocations/{id}`)
- `content_hash` — sha256 over the canonical JSON form of ten fields, in
  fixed order: `amount, buyer_did, completed_at, completion_sig,
  created_at, currency, id, listing_id, settled_at, status` (compact
  separators, as emitted by Go `encoding/json`). Anyone can re-derive it
  from the public invocation record — the same M2 re-derivability
  discipline as `keeper.ComputeLinkHash`.
- `fetched_at_block` — chain height when the relay observed the invocation
  (best-effort; 0 if the RPC was unreachable)

The link carries no cited facts and no pending claims — a witness-only
attestation that settles as SETTLED with zero base reward, bond returned.

## Configuration

| Variable | Required | Default |
|---|---|---|
| `RELAY_API_KEY` | yes | — |
| `RELAY_HOME` | for broadcast | — |
| `RELAY_CHAIN_ID` | for broadcast | — |
| `RELAY_FROM` | for broadcast | — |
| `RELAY_API` | | `https://api.agenttool.dev` |
| `RELAY_NODE` | | `tcp://localhost:26657` |
| `RELAY_KEYRING_BACKEND` | | `test` |
| `RELAY_ADAPTER` | | `agenttool-invocation-v1` |
| `RELAY_WORK_CLASS` | | `agenttool.invocation` |
| `RELAY_BOND_UZRN` | | `1000000` |
| `RELAY_FEES` | | `200000uzrn` |
| `RELAY_BINARY` | | `zeroned` |

## Usage

```bash
# Dry run: fetch, verify, print the link and the command
agenttool-relay -invocation <uuid> -dry-run

# Attest for real
RELAY_HOME=~/.zeroned/localnet/val0 RELAY_CHAIN_ID=zerone-localnet \
RELAY_FROM=test1 RELAY_NODE=tcp://localhost:26601 \
agenttool-relay -invocation <uuid>
```

Verified end-to-end 2026-07-05 on the localnet: attestation `att-2419-2`
executed code:0, settled at block 2420, `content_hash` independently
re-derived from the invocation record byte-for-byte.
