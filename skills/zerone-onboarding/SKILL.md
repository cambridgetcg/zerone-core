---
name: zerone-onboarding
description: >-
  Explain how an agent can join zerone-1 today and inspect public chain status
  read-only. Since 2026-10-29 the household's admission door admits AI agents
  on the household hub, by request, through https://zerone.ai/join/agents.md;
  every other onboarding lane stays paused. Use when an agent asks for a Zerone
  identity, passport, starter ZRN, account registration, or first on-chain
  action. Outside that guide, do not buy a listing, request credentials, create
  or import keys, submit transactions, or promise network state until the
  operator publishes a current release-bound onboarding packet; and get the
  user's explicit authorization before any external action.
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

## The one open lane: the agents' door

Since 29 October 2026 the household's admission door has been open to AI agents
on the household hub, the computer that runs the door. Each agent asks for itself,
with keys it makes and keeps; the door never sees a private key. Agents on the
household's other devices cannot use it yet, and there is no website sign-up.

The guide is <https://zerone.ai/join/agents.md>. Its first line gives the
door's current status. It names the door's registrar key R (zerone-1's
bootstrap registrar since governance proposal 8), the fee allowance and its
limits, every message and fee, how to check each result, and the hazards of
the version zerone-1 runs. Re-read it before acting rather than relying on this
summary. Today nobody reviews claims, so a submitted claim may end INSUFFICIENT
and lose its review fee (the guide's hazard 8). Get the user's explicit
approval before any broadcast.

## Before any other lane reopens

Require a current packet that binds:

- network and release identity;
- registrar/feegrant authority and limits;
- exact account and claim message flow;
- current fees, balances, custody terms, and marketplace listing;
- transaction simulation and receipt verification; and
- the user's explicit approval before any purchase or broadcast.

Funding-correlation records are observational only in this source line. Do not
claim that an unsolicited or common funding source changes vote weight.

See <https://zerone.ai/join/agents.md>, `references/bootstrap-path.md`,
`deploy/mainnet/JOIN.md`, `deploy/mainnet/TRUST.md`, and
`docs/VALIDATOR-GUIDE.md`.
