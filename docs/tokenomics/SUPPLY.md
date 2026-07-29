# Supply Cap & Emission Schedule

## Hard Cap

```
222,222,222 ZRN  =  222,222,222,000,000 uzrn
```

Enforced in code at `x/vesting_rewards/types/keys.go`:

```go
MaxSupplyUzrn = "222222222000000"
```

The cap is checked against **current bank supply** (not cumulative minted). The cap is on circulating + locked supply, not on total-ever-created. Zerone has no discretionary burn; the one exception is slashed attestation bonds in `x/substrate_bridge`, which burn at rejection so that slashed dishonesty becomes future emission headroom instead of dead weight in a module escrow. Otherwise the supply monotonically increases toward the hard cap.

## Why 222,222,222?

The number is symbolic (ZERONE — the collapse of duality into unity) and practically useful: it's large enough to support a global knowledge economy at micro-denomination scale while being small enough to make each ZRN meaningful. At 6 decimal places, the smallest unit (1 uzrn) represents roughly one millionth of a single ZRN.

## Emission Pathways

After the disclosed 13,555 ZRN custodial genesis allocation, all wired
post-genesis native issuance calls draw against the 222,222,222 hard cap
through `MintWithCap`.

| Pathway | Module | Trigger | Recipient |
|---------|--------|---------|-----------|
| **Transaction-bearing block rewards** | `x/vesting_rewards` | Any non-injection user tx; scaled by validator count, decay, and survived-challenge rate | Block proposer / configured withdraw address, protocol, development, and research destinations |
| **Claiming-pot claims** | `x/claiming_pot` | A claimant calls `MsgClaim` against a bootstrap or legacy governance-created general pot | The eligible claimant |
| **External-work attestation rewards** | `x/substrate_bridge` | A bonded attestation settles through an active genesis-seeded or gov-authority-registered adapter | The submitter directly. Substrate-linked attestations pay the M4 formula at settle; witness-only attestations pay the adapter's configured `witness_reward_uzrn` only after surviving the challenge window (escrowed at settle, cancelled if the adapter is tombstoned — issuance follows survival, not acceptance) |
| **Probe-bounty pool (disabled by default/published params)** | `x/knowledge` | Governance sets a positive per-block rate | Probe-bounty module pool |
| **Emission periods (disabled by default/published params)** | `x/tokens` | Governance enables the latch and schedules a period | Authority-selected module/account recipient |

The block-reward stream is the primary modeled emission. Eligibility currently
means “the block carried at least one ordinary user transaction,” not “a new
fact was verified.” Claiming pots include the one-time bootstrap seed and a
legacy governance-created general-pot surface; both charge the same fixed-size
lifetime budget. The published genesis seeds `agenttool-invocation-v1` as
ACTIVE, but current live adapter query state was not reverified. Source permits
direct gov-authority adapter registration; CategoryAdapterRegistration LIP
payload dispatch is not wired. Witness rewards are survival-gated.

The final two controls in the table are source-capable but latched off in
default and published params. Training-fund disbursement and
contribution-challenge bonus minting are release-sealed. `InitChain` rejects
bank supply above the cap, and every wired post-genesis native mint call uses
`MintWithCap`, which checks current bank supply. When the cap binds, those
calls stop minting; burns may reopen headroom.

## Block Reward Emission Formula

```
reward(block) = max(
    baseReward × decayFactor^epoch,
    floorReward
)
```

Where:
- **baseReward** = 10,000,000 uzrn (10 ZRN)
- **decayFactor** = 0.994478 (994,478 on 1M BPS scale — ~1-year half-life)
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

Blocks with no non-injection user transaction receive **zero** rewards under
the published/default `empty_block_reward_rate = 0`. Any ordinary user
transaction, including a transfer, makes the block eligible; this rule does
not prove that truth was produced.

## Decay Curve

The projections below are **fully active, full-target upper-bound scenarios**:
they assume every block qualifies for a PoT reward and at least 22 active
validators. Actual issuance is activity-dependent, participation-scaled, and
can be materially lower.

The decay uses exponentiation by squaring in fixed-point arithmetic (no floating point):

With a 1-year half-life, rewards halve approximately every year (~125 epochs):

| Period | Per-Block Reward | Annual Emission | Cumulative | % of Cap |
|--------|-----------------|----------------|------------|----------|
| Year 0 start | 10.0 ZRN | — | — | — |
| Year 1 end | 5.0 ZRN | ~90M ZRN | 90M | 40.7% |
| Year 2 end | 2.5 ZRN | ~45M ZRN | 136M | 61.1% |
| Year 3 end | 1.25 ZRN | ~23M ZRN | 158M | 71.2% |
| Year 4 end | 0.625 ZRN | ~11M ZRN | 169M | 76.1% |
| Year 5 end | 0.31 ZRN | ~6M ZRN | 175M | 78.9% |
| Year 6.6 | 0.1 ZRN (floor) | ~1.26M ZRN | ~180M | ~81% |
| Year 10+ | 0.1 ZRN (floor) | ~1.26M ZRN | ~185M+ | ~83%+ |

Floor reward (0.1 ZRN) kicks in around epoch 832 (~year 6.6). After that, emission is constant at 0.1 ZRN/block.

## Long-Term Supply Projections

### Phase 0: Genesis (Block 0)
- **13,555 ZRN at genesis (0.0061% of cap)** — 11,333 ZRN validator
  collateral (11,111 bonded + 222 spendable gas) + a transferable 2,222 ZRN
  operator float, every address published in the hash-bound manifest.
  **0 ZRN of participation-minted supply** at genesis — that begins only when
  a participation pathway executes
- The validator secures consensus with real bonded self-stake collateral
  (11,111 ZRN, locked as collateral). A block reward is possible only on
  transaction-bearing blocks; height alone does not earn one, but an ordinary
  transfer is sufficient for eligibility
- Research fund, foundation, faucet, and separate founder stipend start empty.
  The founding household nevertheless controls the published validator and
  operations balances above
- Bootstrap pool is configured (whitelist + 0.222 ZRN per agent) but mints nothing until agents claim

### Phase 1: Bootstrap Emission (Year 1, upper-bound scenario)
- Up to ~90M ZRN minted from block rewards (~40.7% of cap) under the
  full-activity/full-target assumptions above
- Rewards halve annually (1-year half-life)
- Research fund fills organically (3.33% of all rewards)
- Development fund receives 19.67% for ecosystem growth
- Establishes validator economy and initial knowledge base

### Phase 2: Gradual Decay (Years 2–6.6, upper-bound scenario)
- Year 2: ~45M ZRN minted (2.5 ZRN/block at year end)
- Year 5: ~6M ZRN minted (0.31 ZRN/block at year end)
- Late joiners earn meaningful rewards: year 2 validators earn half of genesis rate, not 1/100th
- No general revenue burn; rejected substrate-attestation bonds are the narrow
  punitive burn exception

### Phase 3: Floor Emission (Year 6.6+)
- 0.1 ZRN/block = ~1.26M ZRN/year at floor rate
- Floor provides permanent validator incentive layer
- Fee-based economy has ~6.5 years to develop before floor kicks in

### Phase 4: Cap Approach (Long-term)
- At floor rate (~1.26M/year), reaching 222M from ~180M takes ~33 more years
- The economy has decades to mature before the cap binds
- When cap binds, economy transitions to pure fee-based incentives

### Effective Circulating Supply

Not all minted ZRN is immediately liquid:
- **Vesting locks**: Rewards vest over time linked to epistemic category (months to years)
- **Staking locks**: Validators stake ZRN with 7-day unbonding
- **Challenge reserves**: category defaults span 5–40% of vesting rewards held
  as permanent reserve
- **Development fund**: 19.67% of revenue held for governance-directed disbursement

## Supply Cap Mechanics

`MintWithCap` checks **current bank supply** (not cumulative minted). Except for
rejected substrate-attestation bonds, supply increases toward the hard cap. The
cap will bind when total circulating + locked ZRN reaches 222,222,222 ZRN.

At that point, block reward minting stops and the economy runs purely on transaction fees and existing token velocity. This is a deliberate end-state: the knowledge base is mature enough that new minting isn't needed to incentivise participation.

## Comparison to Other Chains

| Chain | Supply Model | Initial Emission | Decay | Cap |
|-------|-------------|-----------------|-------|-----|
| **Zerone** | PoT mining | 10 ZRN/block | 50%/year | 222,222,222 |
| Bitcoin | PoW mining | 50 BTC/block | 50%/4 years | 21,000,000 |
| Ethereum | PoS + EIP-1559 | ~2 ETH/block + burn | None (variable) | No cap |
| Cosmos Hub | PoS inflation | Variable | Dynamic | No cap |
| Osmosis | Thirdening | Variable | 33%/year | 1,000,000,000 |

Zerone's decay is 4× faster than Bitcoin's halvings but reaches a permanent
floor (0.1 ZRN) rather than zero. Zerone has no general revenue burn; rejected
substrate-attestation bonds are the narrow exception. The 1-year half-life
balances early adopter advantage with accessibility for late joiners.
