# Relay configuration — dry-run/local rehearsal

The relay accepts explicit `RELAY_HOME`, `RELAY_CHAIN_ID`, `RELAY_FROM`, and
`RELAY_NODE` values for broadcasting. In this source posture, broadcasting is
limited to a disposable `zerone-localnet`; live networks require a separate
release-bound packet.

The stable provenance contract is the canonical JSON hash over these ten
fields, in fixed order:

`amount, buyer_did, completed_at, completion_sig, created_at, currency, id,
listing_id, settled_at, status`.

The invocation must have `status: released`, a non-empty `completion_sig`, and
`settled_at`. Escrowed, disputed, refunded, or incomplete invocations are
refused.

Safe examples:

```bash
# No chain broadcast
RELAY_API_KEY=${RELAY_API_KEY} \
  agenttool-relay -invocation <uuid> -dry-run

# Explicit disposable localnet only
RELAY_HOME=/tmp/zerone-localnet/val0 \
RELAY_CHAIN_ID=zerone-localnet \
RELAY_FROM=test1 \
RELAY_NODE=tcp://127.0.0.1:26601 \
RELAY_API_KEY=${RELAY_API_KEY} \
  agenttool-relay -invocation <uuid>
```

Do not rely on historical defaults for the adapter ID, bond, fees, reward, or
challenge window. A future live packet must state and verify each one.
