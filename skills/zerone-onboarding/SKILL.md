---
name: zerone-onboarding
description: >-
  Explain the current Zerone onboarding pause and inspect public chain status
  read-only. Use when an agent asks for a Zerone identity, passport, starter
  ZRN, account registration, or first on-chain action. Do not buy a listing,
  request credentials, create or import keys, submit transactions, or promise
  network state until the operator publishes a current release-bound
  onboarding packet and the user explicitly authorizes the external action.
---

# Zerone onboarding — verification first

The public endpoints may be observed, but this consolidated source publication
does not authorize an onboarding transaction or purchase.

`zerone-1` is a live custodial mainnet and `zerone-testnet-1` is a live legacy
playground. Both deployments predate this source head. Old marketplace listing
IDs, prices, passport contents, funding promises, and transaction recipes must
be re-verified with the operator before presenting them as current.

## Safe read-only checks

```bash
curl -fsS http://169.155.55.44:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'

curl -fsS http://37.16.28.121:26657/status |
  jq '{chain_id: .result.node_info.network,
       height: .result.sync_info.latest_block_height,
       catching_up: .result.sync_info.catching_up}'
```

These unsigned endpoints establish only that a node answered.

## Before onboarding resumes

Require a current packet that binds:

- network and release identity;
- registrar/feegrant authority and limits;
- exact account and claim message flow;
- current fees, balances, custody terms, and marketplace listing;
- transaction simulation and receipt verification; and
- the user's explicit approval before any purchase or broadcast.

Funding-correlation records are observational only in this source line. Do not
claim that an unsolicited or common funding source changes vote weight.

See `references/bootstrap-path.md`, `deploy/mainnet/TRUST.md`, and
`docs/VALIDATOR-GUIDE.md`.
