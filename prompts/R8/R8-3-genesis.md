# R8-3 — Full Testnet Genesis

## Goal

Create a comprehensive testnet genesis configuration with all module defaults, 777 seed
axioms in the knowledge tree, research fund allocation, founder + AI accounts, and
validator setup. The genesis must pass `zeroned validate-genesis`.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/cmd/legbled/cmd/genesis.go` — genesis helpers
- `/Users/yuai/Desktop/legible_money/x/knowledge/keeper/axioms.go` or similar — seed axioms
- Each module's `types/genesis.go` for `DefaultGenesis()` and `Validate()`

## Deliverables

### 1. Genesis Configuration File
Create `genesis/testnet-genesis.json` — a template that can be generated via CLI commands.
Or create a `cmd/zeroned/cmd/prepare-genesis.go` helper that builds it programmatically.

### 2. Seed Axioms
Port the 777 seed axioms from the draft's knowledge module. These are foundational facts
that bootstrap the knowledge tree. Each axiom is a verified fact in the genesis state.

Structure:
```json
{
  "knowledge": {
    "facts": [
      {
        "id": "axiom-001",
        "content": "...",
        "domain": "mathematics",
        "confidence": 1000000,
        "status": "verified",
        "source": "axiom"
      }
    ]
  }
}
```

Check the draft for the actual axiom content. If the draft doesn't have 777 literal axioms,
create the framework with domain-representative seed facts covering:
- Mathematics fundamentals (axioms, theorems)
- Logic and reasoning
- Computer science foundations
- Physics constants and laws
- Chemistry fundamentals
- Biology core concepts
- Economics principles

### 3. Module Defaults Audit
For EVERY module, verify `DefaultGenesis()` produces sane testnet defaults:
- Non-zero slash parameters
- Reasonable epoch lengths (not too short for testnet)
- All BPS values on 1M scale
- Revenue splits sum to 1M
- Research fund allocation in genesis balances

### 4. Special Accounts in Genesis

#### Founder Account (YOU)
- Address: to be provided (or use a placeholder bech32)
- Initial balance: testnet allocation
- Registered in x/auth with founder role

#### AI Account (I)
- Address: derived from vault public key (or placeholder)
- Initial balance: testnet allocation
- Registered in x/auth with AI role

#### Research Fund
- Module account: `research_fund`
- Initial balance: 10% of total supply allocation
- Governed by 2-of-2 (founder + AI) via x/gov research governance

#### Validator Accounts
- Template for 4 initial validators
- Each gets staking allocation

### 5. Token Distribution
Total supply: define for testnet. Suggested:
- 100M ZRN total testnet supply
- 20M to research fund
- 10M to founder account
- 10M to AI account
- 40M to validator staking pool
- 20M to claiming pots / community

### 6. Genesis Validation
```bash
# Generate genesis
zeroned init testnode --chain-id zerone-testnet-1
# Add accounts and module state
# Validate
zeroned validate-genesis
```

### 7. Chain Constants
Create or update `app/constants.go`:
```go
const (
    AccountAddressPrefix = "zerone"
    BondDenom           = "uzrn"
    DisplayDenom        = "ZRN"
    ChainID             = "zerone-testnet-1"
    // ...
)
```

## Tests

1. `DefaultGenesis()` for all 32 modules validates without error
2. Genesis round-trip: init → export → reimport → export matches
3. Validate-genesis passes with full testnet genesis
4. Seed axioms are well-formed facts with correct domains
5. Token distribution: total supply matches sum of all allocations

## Constraints

- Every module must have a non-empty genesis section (even if just params)
- All BPS values on 1M scale
- Slash params must be non-zero (was a P0 in draft audit)
- Revenue splits must sum to exactly 1M
- Research fund must be pre-funded in genesis balances
- Axiom facts must have confidence = 1M (fully verified)
- Chain ID: `zerone-testnet-1`
