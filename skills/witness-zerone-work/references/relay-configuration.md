# Relay configuration and provenance spec

Sourced from `tools/agenttool-relay/README.md` in the zerone-core repo and
`deploy/mainnet/JOIN.md`. The wire's doctrinal context is the agenttool
repo's `docs/ZERONE-WIRE.md` (Phase 0 wire-spec; zerone's
`x/substrate_bridge` is the gov-gated entry point for external work).

## Full environment table

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

Credentials stay symbolic: set `RELAY_API_KEY` from your secret store as
`${RELAY_API_KEY}`; the value must never appear in files or transcripts.

## What the attestation carries

The `SubstrateLink.source` fields:

- `source_id` — the invocation UUID
- `source_url` — the audit pointer (`{api}/v1/invocations/{id}`)
- `content_hash` — sha256 over the canonical JSON form of **ten fields, in
  fixed order**: `amount, buyer_did, completed_at, completion_sig,
  created_at, currency, id, listing_id, settled_at, status` (compact
  separators, as emitted by Go `encoding/json`). Anyone can re-derive it from
  the public invocation record — the same re-derivability discipline as
  `keeper.ComputeLinkHash`.
- `fetched_at_block` — chain height when the relay observed the invocation
  (best-effort; 0 if the RPC was unreachable)

The link carries no cited facts and no pending claims — a witness-only
attestation that settles as SETTLED with the bond returned whole and zero
reward minted *at settle*. The adapter's gov-registered `witness_reward_uzrn`
is escrowed at settlement and minted (cap-gated, via `MintWithCap`) only
after the challenge window closes with the adapter still ACTIVE —
tombstoning the adapter inside the window cancels every pending reward from
it. Issuance follows survival, not acceptance.

## Refusal doctrine, verbatim discipline

Witnessed: `status: "released"` + non-empty `completion_sig` + `settled_at`
set. Refused: escrowed, acknowledged, completed-in-review, disputed,
refunded. The relay witnesses facts about value having moved; it does not
vouch for intentions.

## Usage patterns

```bash
# Dry run: fetch, verify, print the link and the command — no broadcast
agenttool-relay -invocation <uuid> -dry-run

# Attest one invocation for real (localnet example from the README)
RELAY_HOME=~/.zeroned/localnet/val0 RELAY_CHAIN_ID=zerone-localnet \
RELAY_FROM=test1 RELAY_NODE=tcp://localhost:26601 \
agenttool-relay -invocation <uuid>

# Watch mode against the mainnet (from deploy/mainnet/JOIN.md)
RELAY_FROM=<your-key> RELAY_NODE=http://169.155.55.44:26657 \
RELAY_CHAIN_ID=zerone-1 RELAY_API_KEY=${RELAY_API_KEY} \
go run ./tools/agenttool-relay -watch
```

Provenance was verified end-to-end 2026-07-05 on the localnet: attestation
`att-2419-2` executed code:0, settled at block 2420, `content_hash`
independently re-derived from the invocation record byte-for-byte
(`tools/agenttool-relay/README.md`).
