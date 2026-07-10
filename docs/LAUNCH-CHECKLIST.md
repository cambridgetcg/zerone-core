# Zerone Testnet Launch Checklist

Chain ID: `zerone-testnet-1`

---

## Pre-Launch

### Code & Build

- [ ] All modules compile: `go build ./...`
- [ ] All unit tests pass: `go test ./...`
- [ ] Cross-stack integration tests pass
- [ ] Multi-validator tests pass (4-node local cluster)
- [ ] IBC relay tests pass (hermes/rly)
- [ ] Simulation tests pass (1000+ blocks)
- [ ] Race condition check: `go test -race ./...`
- [ ] Binary reproducible: same SHA across builds

### Genesis Preparation

- [ ] Run `scripts/mainnet-ceremony.sh` with the ceremony.env inputs (addresses, PQ commitment hashes, gentx dir, whitelist snapshot) — the script runs `zeroned init`, seeds the two genesis balances (validator + operator float), applies every §2 parameter patch, and emits `GENESIS-MANIFEST.md` + the genesis sha256
- [ ] Verify the zero-ALLOCATION invariant: bank supply exactly **13,555 ZRN** = 11,111 bonded validator self-stake collateral + 222 validator gas + 2,222 operator float; 2 published balances, 1 validator; no team/foundation/investor/research/faucet balance; everything else mints on participation (run `ZERONE_GENESIS_ARTIFACT=<genesis.json> go test ./tests/cross_stack/ -run TestGenesisArtifact`)
- [ ] Verify `knowledge.bootstrap_fund_allocation = "0"` and the Genesis Creed + 8 work_creed inception pins are present (no axiom seeding — the axiom tier was deliberately removed; knowledge enters via participation)
- [ ] Verify hard cap `MaxSupplyUzrn` = `222222222000000` (222,222,222 ZRN)
- [ ] Verify bootstrap pool whitelist + per-agent claim amount (`0.222 ZRN`) configured in `x/claiming_pot` genesis state
- [ ] Verify all three mint pathways gate through `MintWithCap`: `x/vesting_rewards` block rewards, `x/claiming_pot` bootstrap claims, and `x/substrate_bridge` external-work attestations
- [ ] Distribute signed genesis.json to all validators
- [ ] SHA256 hash matches across all validators
- [ ] Genesis time set (coordinated UTC timestamp)

### Validator Coordination

- [ ] 4 genesis validators confirmed with addresses
- [ ] Each validator generates `gentx` with self-delegation
- [ ] All `gentx` files collected and added to genesis
- [ ] Persistent peer list compiled (node IDs + IPs)
- [ ] Seed node addresses shared
- [ ] Communication channel established (Discord/Telegram)

### Infrastructure

- [ ] Each validator: 4+ CPU cores, 16 GB RAM, 500 GB SSD
- [ ] Port 26656 (P2P) open on all validators
- [ ] Port 26657 (RPC) open on public-facing nodes
- [ ] Port 1317 (REST API) open on API nodes
- [ ] Port 9090 (gRPC) open on API nodes
- [ ] Cosmovisor installed and configured on all validators
- [ ] Monitoring stack deployed (Prometheus + Grafana)
- [ ] Log aggregation configured
- [ ] Backup strategy documented and tested

### Security

- [ ] Validator keys backed up securely offline
- [ ] No validator runs duplicate signing keys
- [ ] Sentry node architecture for public validators
- [ ] Firewall rules reviewed
- [ ] DDoS protection in place
- [ ] Key management documented

### Documentation

- [ ] README.md updated with testnet info
- [ ] Validator Guide reviewed and tested end-to-end
- [ ] Parameter Reference complete for all 35 parameter-bearing modules (38 total; 3 pure synthesisers carry no params)
- [ ] FAQ covers common validator questions
- [ ] API documentation (Swagger) deployed

---

## Launch Day

### Sequence

1. [ ] All validators confirm genesis.json hash match
2. [ ] Set `--halt-height 0` (no premature halt)
3. [ ] Start all validators within the genesis time window
4. [ ] Verify first block produced
5. [ ] Verify all 4 validators signing blocks
6. [ ] Announce testnet live

### Health Checks (First Hour)

- [ ] Blocks producing at ~2.5s intervals
- [ ] All 4 validators active and signing
- [ ] No missed blocks in first 100 blocks
- [ ] RPC endpoints responding
- [ ] REST API endpoints responding
- [ ] P2P peer count stable (each node sees 3+ peers)

### First Transactions

- [ ] Send a transfer between two accounts
- [ ] Register a new validator (tier: Apprentice)
- [ ] Submit a knowledge claim
- [ ] Complete a verification round (commit + reveal)
- [ ] Query knowledge module state
- [ ] Verify a whitelisted agent's `MsgClaim` succeeds via feegrant from the onboarding account (there is NO gas-free window — the bootstrap gas-free epoch was retired; fees apply from block 0)

---

## Post-Launch

### Bootstrap Period Monitoring (First 14 Days)

- [ ] Bootstrap claims minting correctly (0.222 ZRN per whitelisted agent, feegrant-subsidized; no gas-free window exists)
- [ ] Block rewards distributing correctly (10 ZRN/block)
- [ ] Verification rewards working (3 ZRN per correct verification)
- [ ] Validator tiers computing correctly
- [ ] No unexpected slashing events
- [ ] Alignment health index reporting (observations + queryable corrections)

### Economic Health

- [ ] Token distribution matches genesis allocations
- [ ] Research fund balance correct
- [ ] Claiming pots balance correct
- [ ] Fee distribution working (93% validators, 7% research fund)
- [ ] Vesting schedules active (if applicable)
- [ ] No unexpected minting or burning

### Knowledge System

- [ ] New claims can be submitted and verified
- [ ] Confidence scoring working
- [ ] Citation economics distributing correctly
- [ ] Cross-domain references tracked
- [ ] Adversarial verification system functional

### Incident Response

- [ ] Emergency halt procedure tested (offline, not on live chain)
- [ ] Validator communication channel active 24/7 first week
- [ ] Upgrade procedure documented (via Cosmovisor)
- [ ] State export/import tested
- [ ] Rollback procedure documented

---

## Fees & Governance (from Block 0 — no bootstrap gas-free epoch)

- [ ] Gas fees enforced for all transaction types from block 0
- [ ] Minimum gas price enforced (0.025 uzrn, signed launch-policy commitment on every validator)
- [ ] Fee market functioning
- [ ] Governance proposals can be submitted (100 ZRN deposit fundable from one operator float)
- [ ] Parameter changes can be proposed and voted on
