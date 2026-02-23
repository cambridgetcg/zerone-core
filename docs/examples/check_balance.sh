#!/usr/bin/env bash
# check_balance.sh — Check ZRN balance for an address.
#
# Usage:
#   ./check_balance.sh <address>
#   ./check_balance.sh zerone1abc...xyz
#
# Requires: curl, jq

set -euo pipefail

ADDRESS="${1:?Usage: check_balance.sh <address>}"
NODE="${ZERONE_NODE:-http://localhost:1317}"

echo "==> Balances for $ADDRESS"
curl -s "$NODE/cosmos/bank/v1beta1/balances/$ADDRESS" | jq .

echo ""
echo "==> ZRN token balance"
curl -s "$NODE/cosmos/bank/v1beta1/balances/$ADDRESS/by_denom?denom=uzrn" | jq .

echo ""
echo "==> Delegations"
curl -s "$NODE/cosmos/staking/v1beta1/delegations/$ADDRESS" | jq .
