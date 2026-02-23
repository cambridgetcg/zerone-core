# Genesis Distribution

## No Pre-Mine, No ICO

Zero ZRN is sold before launch. All genesis balances are **bootstrap allocations** for operational purposes:

| Account | Mainnet Balance | Testnet Balance | Purpose |
|---------|----------------|----------------|---------|
| **Foundation** | 10,000,000 ZRN | 1,000,000 ZRN | Protocol development, grants, ecosystem |
| **Research Treasury** | 5,000,000 ZRN | 500,000 ZRN | 2-of-2 multisig research fund |
| **Faucet** | 500,000 ZRN | 100,000 ZRN | Testnet distribution, new user onboarding |
| **Per Validator** | 1,000,000 ZRN | 100,000 ZRN | Validator operations |
| **Per Validator Stake** | 100,000 ZRN | 10,000 ZRN | Initial self-delegation |

### Total Genesis Supply (Mainnet)

With 3 validators at launch:

```
Foundation:         10,000,000 ZRN
Research Treasury:   5,000,000 ZRN
Faucet:                500,000 ZRN
Validators (3×1M):   3,000,000 ZRN
────────────────────────────────
Total:              18,500,000 ZRN  (8.3% of max supply)
```

With 22 validators (target):

```
Total:              37,500,000 ZRN  (16.9% of max supply)
```

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

## Airdrop (Planned)

A planned **agent airdrop** at launch:
- **0.222 ZRN per whitelisted agent** for bootstrap fund
- Distributed via `x/claiming_pot` with vesting schedule
- Eligibility: whitelisted agent addresses (criteria TBD)
- Purpose: Enable AI agents to have initial stake for home creation and tool usage
