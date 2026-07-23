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
attestation that settles as SETTLED with the bond returned whole and zero
reward minted *at settle*. If the adapter's gov registration carries a
`witness_reward_uzrn`, that reward is escrowed at settlement and minted
(cap-gated, via `MintWithCap`) only after the challenge window closes with
the adapter still ACTIVE — tombstoning the adapter inside the window cancels
every pending reward from it. Issuance follows survival, not acceptance.

The reward pays the attestation's **submitter**, so the relay should sign
with the agent's own zerone key (`RELAY_FROM`) — then the agent that did the
witnessed work is the agent that earns the ZRN.

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
| `RELAY_WITNESS_WRITEBACK` | | off (`1` enables) |

## Watch-mode bond safety (the forward-only lifecycle)

Every submission bonds 1 ZRN and pays ~0.12 ZRN in fees — real money — so
the ledger is forward-only. Immediately after a successful broadcast, and
**before** waiting for inclusion, the record is persisted with the tx hash
and `status: "submission_unknown"`. From there it only moves forward:

- **found, code 0** → `attested` (attestation_id recovered from the
  `external_attestation_submitted` event in the tx result; if the event is
  somehow absent the record still lands attested — the bond is posted — with
  an empty attestation_id and a loud log line for manual backfill).
- **found, code != 0** → the tx executed and failed; state reverted, bond
  untouched. Recorded as a failure, normal retry path.
- **not found** → stays `submission_unknown`. Only after **10 consecutive**
  not-found reconciles (one per poll pass; ~15 minutes at the default 90s
  interval, far past mempool retention on both live nets) is the record
  released for resubmission. A tx can never be raced by its own retry.

Old state files (records without `status`) load unchanged: a legacy record
with a tx hash was only ever written after observed inclusion, so it reads
as attested.

## Operational markers

- `RELAY-AUTH-EXPIRED` — logged on HTTP 401 from the agenttool API, and the
  sentinel file `~/.zerone-agent/RELAY-AUTH-ALERT` is refreshed (timestamp +
  message), at most once per hour. Watch the file, not the logs.
- `RELAY-PARKED` — logged once when an invocation reaches the failure
  threshold (5) and will no longer be retried.

## Witness writeback (flag-gated)

With `RELAY_WITNESS_WRITEBACK=1`, each attestation confirmed on-chain is
reported back to agenttool: `POST /v1/invocations/{id}/witness` with
`{chain_id, tx_hash, attestation_id, adapter_id}` under the same bearer.
Default off — the route is not deployed on the live API yet (it is being
rebuilt in a parallel agenttool PR); a 404 logs "route not deployed" once
per run. Writeback is strictly best-effort: the attested state is persisted
before the POST and no writeback outcome can ever change it.

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
