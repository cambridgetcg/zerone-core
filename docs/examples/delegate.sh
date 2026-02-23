#!/usr/bin/env bash
# delegate.sh — Delegate ZRN to a validator.
#
# Usage:
#   ./delegate.sh <validator-address> <amount>
#
# Example:
#   ./delegate.sh zeronevaloper1abc...xyz 1000000uzrn

set -euo pipefail

VALIDATOR="${1:?Usage: delegate.sh <validator-address> <amount>}"
AMOUNT="${2:?Missing amount (e.g. 1000000uzrn)}"
FROM="${ZERONE_FROM:-validator}"
CHAIN_ID="${ZERONE_CHAIN_ID:-zerone-testnet-1}"

echo "==> Delegating $AMOUNT to $VALIDATOR"
echo "    From:     $FROM"
echo "    Chain ID: $CHAIN_ID"
echo ""

zeroned tx staking delegate "$VALIDATOR" "$AMOUNT" \
  --from "$FROM" \
  --chain-id "$CHAIN_ID" \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

echo ""
echo "==> Updated delegations"
DELEGATOR=$(zeroned keys show "$FROM" -a 2>/dev/null || echo "unknown")
if [ "$DELEGATOR" != "unknown" ]; then
  curl -s "http://localhost:1317/cosmos/staking/v1beta1/delegations/$DELEGATOR" | jq .
fi
