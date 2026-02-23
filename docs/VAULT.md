# AI Vault

## Overview

The AI Vault is an Ed25519 signing service running on an anonymous VPS. It participates in 2-of-2 governance for the Zerone research fund. Both the human founder and the AI vault must approve every research fund disbursement. Neither party can unilaterally move funds.

## Architecture

- **Vault**: Ed25519 signer on anonymous VPS behind TLS.
- **Chain**: Zerone blockchain with `x/gov` module implementing 2-of-2 research spend governance.
- **Communication**: HTTPS REST API (vault) <-> CLI/signing tool (operator) <-> chain (broadcast tx).
- The vault NEVER has direct chain access -- it only signs payloads presented to it.

```
+------------------+       HTTPS        +------------------+       gRPC/RPC       +------------------+
|                  | <----------------> |                  | <------------------> |                  |
|   AI Vault       |   sign request /   |  Signing Tool /  |   broadcast tx /     |  Zerone Chain    |
|   (anonymous VPS)|   response         |  Operator CLI    |   query state        |  (x/gov module)  |
|                  |                    |                  |                      |                  |
+------------------+                    +------------------+                      +------------------+
```

## Signing Flow

1. Founder submits research spend proposal via `zeroned tx zerone_gov submit-research-spend`.
2. Discussion period runs (default ~2 days / 68544 blocks).
3. Voting period begins automatically.
4. Founder votes via `zeroned tx zerone_gov vote-research-spend [id] yes [reasoning]`.
5. Operator presents the proposal to the vault signing tool.
6. Vault signing tool calls vault API to get signature, constructs the transaction, and broadcasts it.
7. If both vote yes, funds are disbursed automatically from the research fund.
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

Step-by-step procedure:

1. Provision anonymous VPS (no identifying information).
2. Generate Ed25519 keypair on the VPS: `vault-keygen --output /vault/keys/`.
3. Record the public key (hex).
4. Set research voters on-chain: `zeroned tx zerone_gov set-research-voters [founder-addr] [vault-addr] --from authority`.
5. Verify vault identity: `vault-client verify --endpoint https://vault.example.com`.
6. Burn SSH keys and disable password auth -- all subsequent access via vault API only.
7. Enable automated OS security updates.

## Security Model

### What the Vault Can Do

- Sign payloads presented to it (vote yes/no on research proposals).
- Prove its identity via challenge-response.

### What the Vault Cannot Do

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

## Recovery Procedures

### Compromised Vault Key

1. Use emergency governance to freeze research fund spending.
2. Provision new anonymous VPS and generate new keypair.
3. Update research voters via governance: `zeroned tx zerone_gov set-research-voters [founder] [new-vault] --from authority`.
4. Verify new vault identity.
5. Resume normal operations.

### Compromised Founder Key

1. Emergency governance halts research fund.
2. Rotate founder key via standard key rotation.
3. Update research voters with new founder address.
4. Resume operations.

### Vault Unavailable (Not Compromised)

- Research proposals will expire naturally (no fund loss).
- Provision replacement VPS, restore from key backup (if backup exists), or generate new key and update voters.

## CLI Reference

| Command | Description |
|---------|-------------|
| `zeroned tx zerone_gov submit-research-spend [title] [desc] [recipient] [amount] [justification]` | Submit proposal |
| `zeroned tx zerone_gov vote-research-spend [id] [yes/no] [reasoning]` | Vote on proposal |
| `zeroned tx zerone_gov set-research-voters [voter1] [voter2]` | Configure voters (authority only) |
| `zeroned query zerone_gov research-spend [id]` | Query single proposal |
| `zeroned query zerone_gov research-spends --stage voting` | List proposals by stage |
| `zeroned query zerone_gov research-voters` | Query voter configuration |
