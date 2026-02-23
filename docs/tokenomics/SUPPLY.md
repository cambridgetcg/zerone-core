# Supply Cap & Emission Schedule

## Hard Cap

```
222,222,222 ZRN  =  222,222,222,000,000 uzrn
```

Enforced in code at `x/vesting_rewards/types/keys.go`:

```go
MaxSupplyUzrn = "222222222000000"
```

The cap is checked against **current bank supply** (not cumulative minted). This means burned tokens free headroom — the cap is on circulating + locked supply, not on total-ever-created. This is intentional: it creates a soft recycling effect where the protocol can continue minting block rewards even after significant cumulative issuance, as long as enough ZRN has been burned.

## Why 222,222,222?

The number is symbolic (ZERONE — the collapse of duality into unity) and practically useful: it's large enough to support a global knowledge economy at micro-denomination scale while being small enough to make each ZRN meaningful. At 6 decimal places, the smallest unit (1 uzrn) represents roughly one millionth of a single ZRN.

## Emission Formula

```
reward(block) = max(
    baseReward × decayFactor^epoch,
    floorReward
)
```

Where:
- **baseReward** = 10,000,000 uzrn (10 ZRN)
- **decayFactor** = 0.85 (850,000 on 1M BPS scale)
- **epoch** = floor(blockHeight / blocksPerEpoch)
- **blocksPerEpoch** = 100,000 (~2.9 days at 2.521s blocks)
- **floorReward** = 100,000 uzrn (0.1 ZRN)

### Validator Scaling

The effective reward is further scaled by validator participation:

```
effectiveReward = reward × min(1, activeValidators / targetValidators)
```

Where **targetValidators** = 22. This incentivises validator set growth: with only 11 validators, rewards are halved. At 22+ validators, full rewards apply.

### Empty Block Rule

Blocks with no Proof-of-Truth activity (no verified claims, no knowledge transactions) receive **zero** rewards by default (`empty_block_reward_rate = 0`). This is the critical PoT alignment: the chain only mints when truth is produced.

## Decay Curve

The decay uses exponentiation by squaring in fixed-point arithmetic (no floating point):

| Epoch | Days (~) | Per-Block Reward | Epoch Total | Cumulative |
|-------|----------|-----------------|-------------|------------|
| 0 | 0–2.9 | 10.000 ZRN | 1,000,000 ZRN | 1,000,000 ZRN |
| 1 | 2.9–5.8 | 8.500 ZRN | 850,000 ZRN | 1,850,000 ZRN |
| 2 | 5.8–8.7 | 7.225 ZRN | 722,500 ZRN | 2,572,500 ZRN |
| 3 | 8.7–11.6 | 6.141 ZRN | 614,125 ZRN | 3,186,625 ZRN |
| 5 | 14.5–17.4 | 4.437 ZRN | 443,705 ZRN | 4,305,078 ZRN |
| 10 | 29–31.9 | 1.969 ZRN | 196,874 ZRN | 6,513,216 ZRN |
| 20 | 58–60.9 | 0.388 ZRN | 38,760 ZRN | 8,096,766 ZRN |
| 30 | 87–89.9 | 0.100 ZRN* | 10,000 ZRN | 8,436,xxx ZRN |
| 50 | 145+ | 0.100 ZRN* | 10,000 ZRN | 8,836,xxx ZRN |

*Floor reward kicks in around epoch 27 (day ~78). After that, emission is constant at 0.1 ZRN/block.

## Long-Term Supply Projections

### Phase 0: Genesis (Block 0)
- **0 ZRN in circulation** — no pre-mine, no bootstrap allocations
- Validators participate via virtual stake (11 ZRN virtual weight)
- Research fund, foundation, faucet all start empty

### Phase 1: Rapid Emission (Months 1–3)
- ~8M ZRN minted from block rewards alone
- Rewards decrease rapidly (15%/epoch)
- Research fund fills organically (13% of all rewards = ~1M ZRN)
- Establishes initial knowledge base and economic activity

### Phase 2: Floor Emission (Month 3+)
- 0.1 ZRN/block = ~10,000 ZRN per epoch (~2.9 days)
- ~1.26M ZRN/year at floor rate
- Counterbalanced by 10% burn on all revenue

### Phase 3: Net Deflationary (Long-term)
- As on-chain economic activity (billing, toolbox, disputes, channels) grows, fee-based burns may exceed floor minting
- The cap of 222M ZRN is unlikely to ever be reached; practical circulating supply will be much lower

### Effective Circulating Supply

Not all minted ZRN is immediately liquid:
- **Vesting locks**: Rewards vest over time linked to epistemic category (months to years)
- **Staking locks**: Validators stake ZRN with 7-day unbonding
- **Challenge reserves**: 5–20% of vesting rewards held as permanent reserve
- **Burns**: 10% of every revenue event is permanently destroyed

## Burn Recycling

A critical design choice: `MintWithCap` checks current bank supply, not cumulative minted. This means:

1. Block rewards mint 10 ZRN at block 1
2. 1 ZRN (10%) is burned
3. The supply cap headroom is now slightly higher
4. Future blocks can mint into that headroom

This creates a **sustainable long-term emission** even with a hard cap, as long as economic activity generates enough burning. The system is designed so that the cap binds only if burning stops — which would mean economic activity has stopped, at which point new minting isn't needed anyway.

## Comparison to Other Chains

| Chain | Supply Model | Initial Emission | Decay | Cap |
|-------|-------------|-----------------|-------|-----|
| **Zerone** | PoT mining + burn | 10 ZRN/block | 15%/epoch | 222,222,222 |
| Bitcoin | PoW mining | 50 BTC/block | 50%/4 years | 21,000,000 |
| Ethereum | PoS + EIP-1559 | ~2 ETH/block + burn | None (variable) | No cap |
| Cosmos Hub | PoS inflation | Variable | Dynamic | No cap |
| Osmosis | Thirdening | Variable | 33%/year | 1,000,000,000 |

Zerone's decay is faster than Bitcoin's halvings but reaches a permanent floor rather than zero. Combined with active burning, this creates a deflationary equilibrium that Bitcoin's model doesn't achieve until ~2140.
