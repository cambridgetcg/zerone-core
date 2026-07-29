# AI Vault

> **Status — unactivated design/runbook (2026-07-29):** `x/gov` contains
> 2-of-2 research-spend machinery, but the published `zerone-1` genesis leaves
> `research_fund_voters` unset. This repository does not establish that an AI
> vault signer is running, funded, or authorized on the live network. The
> procedures below describe a possible future activation and must not be used
> against a shared network without a release-bound governance packet and
> verified on-chain voter configuration.

## Overview

The AI Vault design uses an Ed25519 signing service on separated
infrastructure. If a human-side address and a vault-controlled address are
configured as `research_fund_voters`, both must approve a Phase 0 research
fund disbursement. With the published voter pair unset, research-spend
submission is refused rather than controlled by a working founder+AI pair.

## Architecture

- **Proposed vault**: Ed25519 signer on separated infrastructure behind TLS.
- **Chain source**: Zerone `x/gov` implements configurable 2-of-2 research
  spend governance.
- **Communication**: HTTPS REST API (vault) <-> CLI/signing tool (operator) <-> chain (broadcast tx).
- **Intended boundary**: the vault has no direct chain access; it signs
  payloads presented to it.

```
+------------------+       HTTPS        +------------------+       gRPC/RPC       +------------------+
|                  | <----------------> |                  | <------------------> |                  |
|   AI Vault       |   sign request /   |  Signing Tool /  |   broadcast tx /     |  Zerone Chain    |
|   (anonymous VPS)|   response         |  Operator CLI    |   query state        |  (x/gov module)  |
|                  |                    |                  |                      |                  |
+------------------+                    +------------------+                      +------------------+
```

## Signing Flow After Activation

1. Founder submits research spend proposal via `zeroned tx zerone_gov submit-research-spend`.
2. Discussion period runs (default ~2 days / 68544 blocks).
3. Voting period begins automatically.
4. Founder votes via `zeroned tx zerone_gov vote-research-spend [id] yes [reasoning]`.
5. Operator presents the proposal to the vault signing tool.
6. Vault signing tool calls vault API to get signature, constructs the transaction, and broadcasts it.
7. If the configured pair both vote yes, funds are disbursed automatically
   from the research fund.
8. If either votes no, the proposal is rejected immediately.
9. If the voting period expires without both votes, the proposal expires.

## Vault API Specification

### GET /v1/public-key

Returns the vault's Ed25519 public key.

**Response:**

```json
{"public_key": "<hex-encoded 32 bytes>"}
```

### POST /v1/sign

Sign a payload. Used by the signing tool to create research spend vote transactions.

**Request:**

```json
{"payload": "<base64-encoded bytes>"}
```

**Response:**

```json
{"signature": "<hex-encoded 64 bytes>"}
```

The vault SHOULD log all sign requests for audit trail.

### POST /v1/challenge

Identity verification via challenge-response.

**Request:**

```json
{"nonce": "<hex-encoded 32 bytes>"}
```

**Response:**

```json
{"signature": "<hex-encoded 64 bytes>"}
```

Caller verifies the signature over the nonce using the vault's public key.

## Key Ceremony

Proposed activation procedure. It is incomplete until a release packet names
the network, exact binary, authority, voter addresses, and verification
evidence:

1. Provision anonymous VPS (no identifying information).
2. Generate Ed25519 keypair on the VPS: `vault-keygen --output /vault/keys/`.
3. Record the public key (hex).
4. Set research voters on-chain: `zeroned tx zerone_gov set-research-voters [founder-addr] [vault-addr] --from authority`.
5. Verify vault identity: `vault-client verify --endpoint https://vault.example.com`.
6. Burn SSH keys and disable password auth -- all subsequent access via vault API only.
7. Enable automated OS security updates.

## Intended Security Model

### What an Activated Vault Can Do

- Sign payloads presented to it (vote yes/no on research proposals).
- Prove its identity via challenge-response.

### What an Activated Vault Cannot Do

- Submit proposals (only designated voters can, but vault needs a tx broadcast path).
- Unilaterally spend funds (requires founder's yes vote too).
- Access chain state directly.
- Export its private key (no API endpoint for this).

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Vault compromise | Attacker can only vote yes/no -- still needs founder approval. Emergency governance override can replace vault voter. |
| Founder compromise | Attacker can only vote yes/no -- still needs vault approval. |
| Both compromised | Full control of research fund. Mitigated by separation of infrastructure and identity. |
| Network interception | TLS required for all vault communication. |
| Vault unavailability | Proposals expire naturally after voting period. No funds at risk. |

## Proposed Recovery Requirements

These are design requirements, not verified live procedures. In particular,
current source does not expose a dedicated emergency-freeze transaction for
the research fund; an activation packet must identify the exact available
governance/authority path before presenting recovery as operational.

### Compromised Vault Key

1. Stop submitting or voting on research spends and publish an incident
   notice through the authorized operational channel.
2. Provision new anonymous VPS and generate new keypair.
3. Update research voters via governance: `zeroned tx zerone_gov set-research-voters [founder] [new-vault] --from authority`.
4. Verify new vault identity.
5. Resume normal operations.

### Compromised Human-Side Key

1. Stop submitting or voting on research spends.
2. Rotate the affected key using the network's release-bound authority path.
3. Update and independently verify the on-chain research voters.
4. Resume only after the replacement configuration is auditable.

### Vault Unavailable (Not Compromised)

- Research proposals will expire naturally (no fund loss).
- Provision replacement VPS, restore from key backup (if backup exists), or generate new key and update voters.

## CLI Reference

| Command | Description |
|---------|-------------|
| `zeroned tx zerone_gov submit-research-spend [title] [desc] [recipient] [amount] [justification]` | Submit proposal |
| `zeroned tx zerone_gov vote-research-spend [id] [yes/no] [reasoning]` | Vote on proposal |
| `zeroned tx zerone_gov set-research-voters [voter1] [voter2]` | Configure voters (authority only; not live authorization) |
| `zeroned query zerone_gov research-spend [id]` | Query single proposal |
| `zeroned query zerone_gov research-spends --stage voting` | List proposals by stage |
| `zeroned query zerone_gov research-voters` | Query voter configuration |
