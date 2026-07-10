# Genesis Distribution

## Zero Insider Allocation — Published Genesis Machinery + Participation-Gated Emission

**No team, foundation, investor, or faucet allocation — no genesis balance anyone can sell, transfer, or use to buy consensus.** The genesis bank is not empty (a proof-of-stake chain cannot boot empty); it holds only published, non-allocation machinery: 11,333 ZRN of validator collateral (11,111 bonded self-stake + 222 spendable gas) and a disclosed 2,222 ZRN operator float.

Genesis supply: **13,555 ZRN (0.0061% of the 222,222,222 cap)** — all provably-bonded collateral or published operational float; **0 ZRN of sellable allocation**. No minting-for-participation has occurred yet. Every genesis address is published in the [signed manifest](../../deploy/mainnet/artifacts/GENESIS-MANIFEST.md).

Beyond that published genesis, ZRN enters circulation through three participation-gated emission pathways:

| Pathway | Module | Who | Why |
|---------|--------|-----|-----|
| **PoT block rewards** | `x/vesting_rewards` | Validators (and revenue-split downstream) | Rewards the work of verifying truth; empty blocks mint 0 |
| **Bootstrap claim** | `x/claiming_pot` | Whitelisted agents (0.222 ZRN each) | Participation in the chain requires ZRN; bootstrap is the seed |
| **External-work attestation** | `x/substrate_bridge` | Agents whose external work survives challenge | Witnessed external work (e.g. the `agenttool-invocation-v1` adapter) earns ZRN when it survives the challenge window |

All three pathways draw against the 222,222,222 ZRN hard cap, minted on-demand through `MintWithCap` — block rewards per block as truth is verified, bootstrap claims per `MsgClaim` as agents register, attestation rewards as external work survives challenge. **None grants anyone a privileged starting balance.**

The founder takes nothing by protocol at genesis: `FounderAddress` is unset, so the dormant `FounderShareBps` accrues 0 ZRN. Founder income, if any, is voluntary and future — patronage, governance grants, or a dormant share the community may later vote to activate. The AI vault holds 0 ZRN at genesis; its operational role is governance signing. The research treasury holds 0 ZRN at genesis; fills from the 3.33% revenue share.

This is sharper than "no pre-mine." It is **"no sellable insider position, every address signed."** The only genesis balances are provably-bonded validator collateral and a disclosed operator float; every other ZRN came from a participatory action — verification, registration, or surviving-work attestation — not from being someone in particular at the right time.

### Bootstrap Problem

If nobody starts with ZRN, how do validators stake?

**Solution: Virtual Stake.** The `virtual_stake` parameter (11 ZRN) gives genesis validators VRF participation weight without real tokens. Apprentice-tier validators can produce blocks and earn rewards with zero self-delegation. As block rewards accumulate, validators self-delegate from earnings and progress through tiers organically.

> **Open design question:** The Cosmos SDK `gentx` flow traditionally requires bonded tokens. The genesis ceremony may need modification to support virtual-only validators, or a minimal seed (e.g., 1 uzrn per validator — purely for gas, not capital) could bootstrap the first transactions.

If nobody starts with ZRN, how do agents transact?

**Solution: Bootstrap Claim.** A whitelisted agent calls `MsgClaim` against the bootstrap pot in `x/claiming_pot`; the module mints 0.222 ZRN directly to the agent. The bootstrap pot is the genesis distribution mechanism — not an afterthought airdrop, but the participation seed every agent uses to begin acting on-chain.

### Genesis Accounts

| Account | Balance | Path to funding |
|---------|---------|-----------------|
| **Validator (operator)** | 11,333 ZRN | 11,111 bonded self-stake collateral + 222 spendable gas; published in the manifest (block rewards accrue from block 1) |
| **Operator float (zerone-ops)** | 2,222 ZRN | Gov deposits + onboarding feegrants; disclosed, trends to zero |
| **Whitelisted Agents** | 0 ZRN | Bootstrap claim (0.222 ZRN per agent) via `x/claiming_pot` |
| **Founder** | 0 ZRN | Dormant — `FounderAddress` unset, accrues nothing; a stipend is gov-activatable later |
| **AI Vault** | 0 ZRN | Operational role only (governance signing); no ZRN holding required |
| **Research Treasury** | 0 ZRN | 3.33% of revenue split, accruing from chain activity |
| **Foundation** | 0 ZRN | Funded by governance proposals over time, drawing from the research treasury |
| **Faucet (testnet only)** | 0 ZRN | Optional; funded by governance or validator tips |

## Genesis Ceremony

The `scripts/genesis-ceremony.sh` script orchestrates a multi-step production genesis:

```bash
# 1. Initialize ceremony (build binary, patch params, create bootstrap accounts)
./scripts/genesis-ceremony.sh init

# 2. Add validators (generate keys, fund, create gentxs)
./scripts/genesis-ceremony.sh add-validator val1
./scripts/genesis-ceremony.sh add-validator val2
./scripts/genesis-ceremony.sh add-validator val3

# 3. Finalize (collect gentxs, validate genesis)
./scripts/genesis-ceremony.sh finalize

# 4. Export (genesis.json + distribution instructions)
./scripts/genesis-ceremony.sh export

# 5. Countdown to launch
./scripts/genesis-ceremony.sh countdown
```

### Chain Configuration

| Parameter | Mainnet | Testnet |
|-----------|---------|---------|
| Chain ID | `zerone-1` | `zerone-testnet-1` |
| Block Time | ~2.521s | ~2.521s |
| Max Gas/Block | 33,333,333 | 33,333,333 |
| Max Block Size | 4 MB | 4 MB |
| Vote Extensions | Height 1 | Height 1 |
| Bond Denom | uzrn | uzrn |

### Genesis Axioms

The genesis ceremony optionally injects **777 foundational axioms** into the knowledge module — pre-accepted mathematical and logical truths that bootstrap the knowledge graph. These are loaded from `x/knowledge/types/genesis_axioms.json` via the axiom-loader tool.

## Research Fund Governance

The research treasury is governed by a **2-of-2 multisig** requiring both signatures for any spend:

| Voter | Key Type | Address |
|-------|----------|---------|
| Yu (Human) | Ledger Nano X (secp256k1) | `lgm1g0q9amg6l666rtee23xjcser4h9wgk8yncedtg` |
| AI (Agent) | Vault Ed25519 on zerone server | `lgm1cgjw09mg6ylc2mwmk6jp8n2yth2ex9jganhptc` |

Multisig address: `lgm120p3d4hhy3dwvpfskpslmpzltclz2vyq0lswp6`

> Note: These are LGM-prefix addresses from the prototype. ZRN-prefix addresses will be generated for the Zerone mainnet genesis.

### Phase 0: Genesis Governance Structure

The 2-of-2 multisig described above is **Phase 0** of a 4-phase governance migration plan. The research fund's decision-making power expands as the community matures, transitioning from founder control to full community governance.

| Phase | Structure | Triggered By |
|-------|-----------|-------------|
| Phase 0 | 2-of-2 (Founder + AI) | Genesis |
| Phase 1 | 2-of-3 (+ 1 community seat) | 10 voters, 5 Guardians, 100K ZRN, ~6mo |
| Phase 2 | 3-of-5 (+ 3 community seats) | 25 voters, 10 Guardians, ~18mo |
| Phase 3 | Standard LIP governance | 50 voters, 22 Guardians, ~3yr |

See [GOVERNANCE-MIGRATION.md](GOVERNANCE-MIGRATION.md) for the full specification.

### Research Spend Process

Research fund spending uses the `x/gov` module's `ResearchSpendProposal`:

1. Either voter proposes a spend (title, description, recipient, amount, justification)
2. Both voters must approve (2-of-2)
3. On-chain execution transfers funds from the research fund module account
4. Full audit trail of proposals and votes stored on-chain

## Denom Metadata

Registered in genesis for wallet compatibility:

```json
{
  "base": "uzrn",
  "display": "zrn",
  "name": "Zerone",
  "symbol": "ZRN",
  "denom_units": [
    {"denom": "uzrn",  "exponent": 0, "aliases": ["microzerone"]},
    {"denom": "mzrn",  "exponent": 3, "aliases": ["millizerone"]},
    {"denom": "zrn",   "exponent": 6, "aliases": ["zerone"]}
  ]
}
```

## Bootstrap Pool — the genesis distribution mechanism

The bootstrap pool is the structural form of the doctrine: agents need ZRN to participate, so participation requires a seed, and the seed is minted on demand when the agent claims.

| Parameter | Value | Reasoning |
|-----------|-------|-----------|
| **Per-agent claim** | 0.222 ZRN | Symbolic (the chain's signature digit) and operationally sufficient — covers gas for `home` creation, initial tool calls, and the first knowledge-claim bonds |
| **Eligibility** | Whitelisted agent addresses | The chain seeds participants it has been told about; non-whitelisted addresses earn ZRN through PoT participation, not bootstrap |
| **Distribution** | `x/claiming_pot` mints to claimer on `MsgClaim` | Mint is incremental — no pre-funded module account, no genesis-balance footprint; the cap is checked at mint time |
| **Vesting** | Optional cliff + linear vesting per pot configuration | Prevents drain-and-dump; the seed is for participation, not speculation |

The whitelist criteria, vesting schedule, and total addressable bootstrap (0.222 ZRN × N whitelisted agents = `N × 222,000` uzrn) are configured at genesis ceremony time. The maximum reachable bootstrap volume is bounded by the whitelist size; in practice this is a tiny fraction of the 222,222,222 cap.
