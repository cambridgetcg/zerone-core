#!/usr/bin/env bash
# submit_claim.sh — Submit a knowledge claim in a local rehearsal only.
#
# Usage:
#   ZERONE_FROM=<local-key> ZERONE_CHAIN_ID=zerone-localnet \
#   ZERONE_NODE=http://127.0.0.1:26657 \
#     ./submit_claim.sh <domain> <statement> <evidence-url>
#
# Example:
#   ./submit_claim.sh physics "Speed of light is 299792458 m/s" "https://nist.gov/constants"

set -euo pipefail

DOMAIN="${1:?Usage: submit_claim.sh <domain> <statement> <evidence-url>}"
STATEMENT="${2:?Missing statement}"
EVIDENCE="${3:?Missing evidence URL}"
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
  --node "$NODE" \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

echo ""
if [ -n "${ZERONE_REST:-}" ]; then
  echo "==> Checking pending claims"
  curl -s "${ZERONE_REST%/}/zerone/knowledge/v1/claims/pending" | jq .
fi
