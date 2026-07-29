---
name: witness-zerone-work
description: >-
  Inspect or locally rehearse the agenttool-to-Zerone attestation relay.
  Current live-network witnessing is paused until a release-bound operator
  packet revalidates the adapter, fees, bond, reward, chain state, and binary.
requirements:
  credentials:
    RELAY_API_KEY: "${RELAY_API_KEY}"
---

# Witness Zerone work — dry-run/local rehearsal only

The relay in `tools/agenttool-relay` fetches a released agenttool invocation,
canonicalizes its load-bearing fields, and can construct a
`MsgSubmitExternalAttestation` for `agenttool-invocation-v1`.

## Current boundary

Do not broadcast to `zerone-1` or `zerone-testnet-1` from current `main`.
Historical bond, reward, fee, adapter-status, key-custody, and challenge-window
claims have not been revalidated against a signed release packet. Source
publication is not transaction authority.

Safe uses:

1. inspect the relay and canonicalization code;
2. fetch and validate one invocation with `-dry-run`; or
3. rehearse against an explicitly disposable `zerone-localnet` home.

Dry run:

```bash
RELAY_API_KEY=${RELAY_API_KEY} \
  agenttool-relay -invocation <uuid> -dry-run
```

Local rehearsal:

```bash
RELAY_HOME=/tmp/zerone-localnet/val0 \
RELAY_CHAIN_ID=zerone-localnet \
RELAY_FROM=test1 \
RELAY_NODE=tcp://127.0.0.1:26601 \
RELAY_API_KEY=${RELAY_API_KEY} \
  agenttool-relay -invocation <uuid>
```

The local chain, account, and adapter must have been created for that
rehearsal. Never substitute a live chain ID or default node home.

Credentials stay symbolic: load `RELAY_API_KEY` from a secret store and never
write it into files or chat. See `references/relay-configuration.md` and
`tools/agenttool-relay/README.md` for the data contract.
