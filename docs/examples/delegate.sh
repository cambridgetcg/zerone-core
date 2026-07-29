#!/usr/bin/env bash
# delegate.sh — Delegate ZRN in a local rehearsal only.
#
# Usage:
#   ZERONE_FROM=<local-key> ZERONE_CHAIN_ID=zerone-localnet \
#   ZERONE_NODE=http://127.0.0.1:26657 \
#     ./delegate.sh <validator-address> <amount>
#
# Example:
#   ./delegate.sh zeronevaloper1abc...xyz 1000000uzrn

set -euo pipefail

VALIDATOR="${1:?Usage: delegate.sh <validator-address> <amount>}"
AMOUNT="${2:?Missing amount (e.g. 1000000uzrn)}"
FROM="${ZERONE_FROM:?Set ZERONE_FROM to a local rehearsal key}"
CHAIN_ID="${ZERONE_CHAIN_ID:?Set ZERONE_CHAIN_ID to a local rehearsal chain}"
NODE="${ZERONE_NODE:?Set ZERONE_NODE to the local rehearsal RPC}"

case "${CHAIN_ID}" in
  zerone-localnet|zerone-rehearsal-*) ;;
  *)
    echo "Refusing to broadcast: this example is local-rehearsal-only." >&2
    echo "Live networks require a release-bound operator packet." >&2
    exit 1
    ;;
esac

echo "==> Delegating $AMOUNT to $VALIDATOR"
echo "    From:     $FROM"
echo "    Chain ID: $CHAIN_ID"
echo ""

zeroned tx staking delegate "$VALIDATOR" "$AMOUNT" \
  --from "$FROM" \
  --chain-id "$CHAIN_ID" \
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

echo ""
echo "==> Updated delegations"
DELEGATOR=$(zeroned keys show "$FROM" -a 2>/dev/null || echo "unknown")
if [ "$DELEGATOR" != "unknown" ] && [ -n "${ZERONE_REST:-}" ]; then
  curl -s "${ZERONE_REST%/}/cosmos/staking/v1beta1/delegations/$DELEGATOR" | jq .
fi
