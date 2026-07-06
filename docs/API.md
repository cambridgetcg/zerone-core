# Zerone API Reference

Zerone exposes three interfaces for interacting with the blockchain:

| Interface | Port | Protocol | Description |
|-----------|------|----------|-------------|
| REST (LCD) | 1317 | HTTP/JSON | gRPC-Gateway — auto-generated REST from proto |
| gRPC | 9090 | gRPC | Native Protobuf RPC |
| CometBFT RPC | 26657 | HTTP/JSON | Block, tx, consensus queries |

## Configuration

In `~/.zeroned/config/app.toml`:

```toml
[api]
enable = true
swagger = true     # serves /swagger/ UI
address = "tcp://0.0.0.0:1317"

[grpc]
enable = true
address = "0.0.0.0:9090"
```

## Swagger UI

When `api.swagger = true`, visit:

```
http://localhost:1317/swagger/
```

The interactive UI lists all endpoints with request/response schemas.

---

## Module Endpoints

### Knowledge Layer

#### knowledge (12 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/knowledge/v1/params` | Module parameters |
| GET | `/zerone/knowledge/v1/facts/{id}` | Single fact by ID |
| GET | `/zerone/knowledge/v1/facts` | List facts (filter by domain/status/category) |
| GET | `/zerone/knowledge/v1/facts/domain/{domain}` | Facts in a domain |
| GET | `/zerone/knowledge/v1/facts/submitter/{submitter}` | Facts by submitter |
| GET | `/zerone/knowledge/v1/claims/{id}` | Single claim by ID |
| GET | `/zerone/knowledge/v1/claims/pending` | Pending claims |
| GET | `/zerone/knowledge/v1/rounds/{id}` | Verification round |
| GET | `/zerone/knowledge/v1/domains/{name}` | Domain by name |
| GET | `/zerone/knowledge/v1/domains` | All domains |
| GET | `/zerone/knowledge/v1/facts/{id}/confidence` | Fact confidence score |
| GET | `/zerone/knowledge/v1/facts/{id}/citations` | Fact citation count |

#### ontology (10 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/ontology/v1/params` | Module parameters |
| GET | `/zerone/ontology/v1/strata/{id}` | Stratum by ID |
| GET | `/zerone/ontology/v1/strata` | All strata |
| GET | `/zerone/ontology/v1/domains/{name}` | Domain by name |
| GET | `/zerone/ontology/v1/domains/stratum/{stratum_id}` | Domains by stratum |
| GET | `/zerone/ontology/v1/domains` | All domains |
| GET | `/zerone/ontology/v1/proposals/{id}` | Ontology proposal |
| GET | `/zerone/ontology/v1/confidence/{domain}` | Confidence ceiling |
| GET | `/zerone/ontology/v1/logic_zones/{name}` | Logic zone |
| GET | `/zerone/ontology/v1/logic_zones` | All logic zones |

#### research (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/research/v1/research/{id}` | Research by ID |
| GET | `/zerone/research/v1/submissions` | All submissions |
| GET | `/zerone/research/v1/bounties/{id}` | Research bounty |
| GET | `/zerone/research/v1/params` | Module parameters |

#### evidence_mgmt (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/evidence_mgmt/v1/evidence/{id}` | Evidence by ID |
| GET | `/zerone/evidence_mgmt/v1/evidence/submitter/{submitter}` | Evidence by submitter |
| GET | `/zerone/evidence_mgmt/v1/evidence/{id}/custody` | Custody chain |
| GET | `/zerone/evidence_mgmt/v1/evidence/{id}/verifications` | Verifications |
| GET | `/zerone/evidence_mgmt/v1/params` | Module parameters |

### Economic Layer

#### billing (6 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/billing/v1/providers/{address}` | Billing provider |
| GET | `/zerone/billing/v1/providers` | All providers |
| GET | `/zerone/billing/v1/quotes/{fact_id}` | Price quote for a fact |
| GET | `/zerone/billing/v1/quotes/batch` | Batch price quote |
| GET | `/zerone/billing/v1/params` | Module parameters |
| GET | `/zerone/billing/v1/price` | Current ZRN/USD price |

#### liquiditypool (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/liquiditypool/v1/pools/{pool_id}` | Pool by ID |
| GET | `/zerone/liquiditypool/v1/pools` | All pools |
| GET | `/zerone/liquiditypool/v1/twap/{pool_id}` | TWAP price |
| GET | `/zerone/liquiditypool/v1/simulate/{pool_id}` | Simulate swap |
| GET | `/zerone/liquiditypool/v1/params` | Module parameters |

#### tokens (12 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/tokens/v1/params` | Module parameters |
| GET | `/zerone/tokens/v1/config` | Token configuration |
| GET | `/zerone/tokens/v1/tokens` | All tokens |
| GET | `/zerone/tokens/v1/tokens/symbol/{symbol}` | Token by symbol |
| GET | `/zerone/tokens/v1/balances/{address}/{denom}` | Token balance |
| GET | `/zerone/tokens/v1/supply/{denom}` | Total supply |
| GET | `/zerone/tokens/v1/allowances/{owner}/{spender}/{denom}` | Allowance |
| GET | `/zerone/tokens/v1/delegated_power/{address}` | Delegated power |
| GET | `/zerone/tokens/v1/wrapped/{denom}` | Wrapped token |
| GET | `/zerone/tokens/v1/wrapped` | All wrapped tokens |
| GET | `/zerone/tokens/v1/emissions/{period_id}` | Emission period |
| GET | `/zerone/tokens/v1/emissions` | All emission periods |

#### vesting_rewards (7 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/vesting_rewards/v1/schedules/{id}` | Vesting schedule by ID |
| GET | `/zerone/vesting_rewards/v1/schedules/recipient/{recipient}` | Schedules by recipient |
| GET | `/zerone/vesting_rewards/v1/schedules/active` | Active schedules |
| GET | `/zerone/vesting_rewards/v1/rewards/distribution` | Block reward distribution |
| GET | `/zerone/vesting_rewards/v1/params` | Module parameters |
| GET | `/zerone/vesting_rewards/v1/research_fund` | Research fund balance |
| GET | `/zerone/vesting_rewards/v1/founder_share` | Founder share status |

#### claiming_pot (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/claiming_pot/v1/pots/{pot_id}` | Pot by ID |
| GET | `/zerone/claiming_pot/v1/pots` | All pots |
| GET | `/zerone/claiming_pot/v1/claimable/{address}` | Claimable amounts |
| GET | `/zerone/claiming_pot/v1/claims/{address}` | Claim history |
| GET | `/zerone/claiming_pot/v1/params` | Module parameters |

### Identity & Governance

#### auth (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/auth/v1/accounts/{address}` | Account by address |
| GET | `/zerone/auth/v1/accounts/did/{did}` | Account by DID |
| GET | `/zerone/auth/v1/params` | Module parameters |
| GET | `/zerone/auth/v1/frozen` | Frozen accounts |

#### staking (7 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/staking/v1/validators/{address}` | Validator by address |
| GET | `/zerone/staking/v1/validators` | All validators |
| GET | `/zerone/staking/v1/delegations/{delegator}/{validator}` | Delegation |
| GET | `/zerone/staking/v1/delegations/{delegator}` | Delegator's delegations |
| GET | `/zerone/staking/v1/validators/{address}/delegations` | Validator's delegations |
| GET | `/zerone/staking/v1/params` | Module parameters |
| GET | `/zerone/staking/v1/tiers` | Tier configuration |

#### gov (6 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/gov/v1/lips/{id}` | LIP (proposal) by ID |
| GET | `/zerone/gov/v1/lips` | All LIPs |
| GET | `/zerone/gov/v1/votes/{lip_id}/{voter}` | Vote on a LIP |
| GET | `/zerone/gov/v1/votes/{lip_id}` | All votes on a LIP |
| GET | `/zerone/gov/v1/tally/{lip_id}` | Tally result |
| GET | `/zerone/gov/v1/params` | Module parameters |

#### qualification (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/qualification/v1/qualifications/{validator}` | Qualification record |
| GET | `/zerone/qualification/v1/qualifications/domain/{domain}` | By domain |
| GET | `/zerone/qualification/v1/qualifications/validator/{validator}` | By validator |
| GET | `/zerone/qualification/v1/endorsements/{validator}` | Endorsements |
| GET | `/zerone/qualification/v1/params` | Module parameters |

### Infrastructure

#### compute_pool (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/compute_pool/v1/params` | Module parameters |
| GET | `/zerone/compute_pool/v1/providers/{address}` | Compute provider |
| GET | `/zerone/compute_pool/v1/providers` | All providers |
| GET | `/zerone/compute_pool/v1/credits/{validator_addr}` | Compute credit |

#### discovery (3 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/discovery/v1/params` | Module parameters |
| GET | `/zerone/discovery/v1/profiles/{address}` | Agent profile |
| GET | `/zerone/discovery/v1/search` | Search agents (query params: domain, capability_type, min_reputation) |

#### bvm (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/bvm/v1/contracts/{address}` | Contract by address |
| GET | `/zerone/bvm/v1/contracts/creator/{creator}` | Contracts by creator |
| GET | `/zerone/bvm/v1/contracts/{address}/state` | Contract state |
| GET | `/zerone/bvm/v1/params` | Module parameters |

#### channels (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/channels/v1/channels/{id}` | Channel by ID |
| GET | `/zerone/channels/v1/channels/payer/{address}` | Channels by payer |
| GET | `/zerone/channels/v1/channels/receiver/{address}` | Channels by receiver |
| GET | `/zerone/channels/v1/disputes/{channel_id}` | Channel dispute |
| GET | `/zerone/channels/v1/params` | Module parameters |

#### schedule (3 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/schedule/v1/params` | Module parameters |
| GET | `/zerone/schedule/v1/processes/{process_id}` | Scheduled process |
| GET | `/zerone/schedule/v1/processes/creator/{creator}` | Processes by creator |

#### home (7 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/home/v1/homes/{address}` | Home by address |
| GET | `/zerone/home/v1/homes/owner/{owner}` | Homes by owner |
| GET | `/zerone/home/v1/homes/{address}/keys` | Home keys |
| GET | `/zerone/home/v1/homes/{address}/sessions` | Home sessions |
| GET | `/zerone/home/v1/homes/{address}/alerts` | Home alerts |
| GET | `/zerone/home/v1/homes/{address}/spending_limits` | Spending limits |
| GET | `/zerone/home/v1/params` | Module parameters |

#### partnerships (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/partnerships/v1/partnerships/{id}` | Partnership by ID |
| GET | `/zerone/partnerships/v1/partnerships/address/{address}` | Partnerships by address |
| GET | `/zerone/partnerships/v1/partnerships/{id}/ops` | Pending operations |
| GET | `/zerone/partnerships/v1/formation_pool` | Formation pool |
| GET | `/zerone/partnerships/v1/params` | Module parameters |

#### toolbox (8 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/toolbox/v1/tools/{id}` | Tool by ID |
| GET | `/zerone/toolbox/v1/tools/deployer/{deployer}` | Tools by deployer |
| GET | `/zerone/toolbox/v1/tools/category/{category}` | Tools by category |
| GET | `/zerone/toolbox/v1/tools/{id}/trust` | Tool trust score |
| GET | `/zerone/toolbox/v1/tools/{id}/dependencies` | Dependency tree |
| GET | `/zerone/toolbox/v1/tools/{id}/free_allowance` | Free allowance |
| GET | `/zerone/toolbox/v1/params` | Module parameters |
| GET | `/zerone/toolbox/v1/tools/{id}/price` | Tool price |

#### tree (7 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/tree/v1/projects/{id}` | Project by ID |
| GET | `/zerone/tree/v1/projects/founder/{founder}` | Projects by founder |
| GET | `/zerone/tree/v1/tasks/{id}` | Task by ID |
| GET | `/zerone/tree/v1/tasks/project/{project_id}` | Tasks by project |
| GET | `/zerone/tree/v1/services/{id}` | Service by ID |
| GET | `/zerone/tree/v1/seeds/{id}` | Seed by ID |
| GET | `/zerone/tree/v1/params` | Module parameters |

### Security & Safety

#### alignment (6 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/alignment/v1/params` | Module parameters |
| GET | `/zerone/alignment/v1/state` | Current alignment state |
| GET | `/zerone/alignment/v1/observations/{height}` | Observation at height |
| GET | `/zerone/alignment/v1/scores/{height}` | Dimension scores at height |
| GET | `/zerone/alignment/v1/health/{height}` | Health index at height |
| GET | `/zerone/alignment/v1/corrections` | Correction history |

#### autopoiesis (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/autopoiesis/v1/params` | Module parameters |
| GET | `/zerone/autopoiesis/v1/multipliers/{path}` | Multiplier by path |
| GET | `/zerone/autopoiesis/v1/multipliers` | All multipliers |
| GET | `/zerone/autopoiesis/v1/epochs/{epoch}` | Epoch snapshot |
| GET | `/zerone/autopoiesis/v1/ssi` | System Sustainability Index |

#### emergency (5 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/emergency/v1/status` | Emergency status |
| GET | `/zerone/emergency/v1/ceremonies/active` | Active halt ceremony |
| GET | `/zerone/emergency/v1/ceremonies/completed` | Completed ceremonies |
| GET | `/zerone/emergency/v1/audit` | Audit log |
| GET | `/zerone/emergency/v1/params` | Module parameters |

#### disputes (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/disputes/v1/disputes/{id}` | Dispute by ID |
| GET | `/zerone/disputes/v1/disputes/target/{target}` | Disputes by target |
| GET | `/zerone/disputes/v1/disputes/active` | Active disputes |
| GET | `/zerone/disputes/v1/params` | Module parameters |

#### capture_defense (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/capture_defense/v1/params` | Module parameters |
| GET | `/zerone/capture_defense/v1/reputation/{address}` | Capture reputation |
| GET | `/zerone/capture_defense/v1/metrics` | Capture metrics |
| GET | `/zerone/capture_defense/v1/cross_stratum` | Cross-stratum requirements |

#### capture_challenge (4 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/capture_challenge/v1/params` | Module parameters |
| GET | `/zerone/capture_challenge/v1/challenges/{id}` | Challenge by ID |
| GET | `/zerone/capture_challenge/v1/bounty_pool` | Bounty pool status |
| GET | `/zerone/capture_challenge/v1/challenges/domain/{domain}` | Challenges by domain |

### IBC & Cross-Chain

#### ibcratelimit (3 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/ibcratelimit/v1/limits/{channel_id}/{denom}` | Rate limit for channel/denom |
| GET | `/zerone/ibcratelimit/v1/limits` | All rate limits |
| GET | `/zerone/ibcratelimit/v1/params` | Module parameters |

#### icaauth (3 RPCs)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/zerone/icaauth/v1/accounts/{owner}/{connection_id}` | ICA account |
| GET | `/zerone/icaauth/v1/accounts/{owner}` | ICA accounts by owner |
| GET | `/zerone/icaauth/v1/params` | Module parameters |

---

## gRPC Usage

List all available services:

```bash
grpcurl -plaintext localhost:9090 list
```

Query a specific service:

```bash
grpcurl -plaintext localhost:9090 list zerone.knowledge.v1.Query
```

Example — query module params:

```bash
grpcurl -plaintext localhost:9090 zerone.knowledge.v1.Query/Params
```

Example — query a fact:

```bash
grpcurl -plaintext -d '{"id": "fact-001"}' \
  localhost:9090 zerone.knowledge.v1.Query/Fact
```

## Transaction Broadcasting

Transactions are submitted via the standard Cosmos SDK `/cosmos/tx/v1beta1/txs` endpoint or via `zeroned tx` CLI commands.

1. Build a transaction with `zeroned tx <module> <msg> --generate-only`
2. Sign it with `zeroned tx sign`
3. Broadcast via REST:

```bash
curl -X POST http://localhost:1317/cosmos/tx/v1beta1/txs \
  -H "Content-Type: application/json" \
  -d @signed_tx.json
```

Or broadcast directly:

```bash
zeroned tx broadcast signed_tx.json --node tcp://localhost:26657
```

## Standard Cosmos Endpoints

In addition to Zerone-specific endpoints, all standard Cosmos SDK REST endpoints are available:

- `/cosmos/auth/v1beta1/accounts/{address}` — Account info
- `/cosmos/bank/v1beta1/balances/{address}` — Token balances
- `/cosmos/staking/v1beta1/validators` — Validator set
- `/cosmos/gov/v1/proposals` — Governance proposals
- `/cosmos/tx/v1beta1/txs/{hash}` — Transaction by hash

See the Swagger UI for the complete list.
