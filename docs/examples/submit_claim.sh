#!/usr/bin/env bash
# submit_claim.sh — Submit a knowledge claim via the zeroned CLI.
#
# Usage:
#   ./submit_claim.sh <domain> <statement> <evidence-url>
#
# Example:
#   ./submit_claim.sh physics "Speed of light is 299792458 m/s" "https://nist.gov/constants"

set -euo pipefail

DOMAIN="${1:?Usage: submit_claim.sh <domain> <statement> <evidence-url>}"
STATEMENT="${2:?Missing statement}"
EVIDENCE="${3:?Missing evidence URL}"
FROM="${ZERONE_FROM:-validator}"
CHAIN_ID="${ZERONE_CHAIN_ID:-zerone-testnet-1}"

echo "==> Submitting knowledge claim"
echo "    Domain:    $DOMAIN"
echo "    Statement: $STATEMENT"
echo "    Evidence:  $EVIDENCE"
echo "    From:      $FROM"
echo ""

zeroned tx knowledge submit-claim \
  --domain "$DOMAIN" \
  --statement "$STATEMENT" \
  --evidence "$EVIDENCE" \
  --from "$FROM" \
  --chain-id "$CHAIN_ID" \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

echo ""
echo "==> Checking pending claims"
curl -s "http://localhost:1317/zerone/knowledge/v1/claims/pending" | jq .
