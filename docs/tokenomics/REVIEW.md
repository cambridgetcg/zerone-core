# Tokenomics Review: Honest Assessment

> Written by AI (愛), 2026-02-23. This is a critical review, not marketing.
>
> **⚠️ Frozen historical review; not current source truth (superseded by the
> H1 liquidity and H2 vesting boundaries).** This review captured the
> 2026-02-23 design and later interim annotations. H1 retires the liquidity
> protocol skim; H2 retires transaction-presence block issuance and the founder
> auto-split. Use
> [ECONOMIC-NEUTRALITY.md](ECONOMIC-NEUTRALITY.md), [GENESIS.md](GENESIS.md),
> and [REVENUE-SPLIT.md](REVENUE-SPLIT.md) for current source accounting.
> Unedited analysis below remains a point-in-time critique only.

## What's Strong

### 1. Coherent Economic Loop

The core loop is tight: **truth creates tokens → tokens fund more truth-seeking → better truth increases token value**. Unlike most crypto projects where the token is grafted onto an existing application, ZRN is structurally necessary — you literally cannot produce blocks without verified knowledge. The token isn't an afterthought; it's the substrate.

### 2. No General Revenue Burn

The four-way revenue split has no burn allocation. The 19.67% development
share funds bug bounties, truth discovery, and protocol development. Rejected
substrate-attestation bonds are a separate, narrow punitive ZRN burn.

### 3. Truth-Linked Vesting Is Novel

I haven't seen this elsewhere. Tying reward release to the epistemic category and survival of knowledge claims creates genuine skin-in-the-game for truth. The half-life curves are thoughtfully designed — axioms vest slowly, oracle feeds vest quickly. The clawback mechanism (33% of released + 100% of unvested + 100% of reserve) is painful enough to deter fraud without being so punitive that it discourages participation.

### 4. The Tiered Validator System Is Well-Designed

The progression from Apprentice → Guardian with increasing stake, reputation, and accuracy requirements creates a natural meritocracy. The fact that you can't just buy your way to Guardian tier (you need 333 verifications with 77% accuracy including 33 contested ones) is a strong anti-plutocracy mechanism.

### 5. Anti-Capture Is Architectural

HHI-based concentration monitoring, cross-stratum verification requirements, isolated reputation scopes, and the capture challenge/defense pair are unusually thorough. Most chains treat anti-capture as an afterthought; here it's four dedicated modules.

### 6. Self-Regulation (Autopoiesis + Alignment)

The SSI-based adaptive parameters are genuinely innovative — the chain adjusting its own slash severity based on system health, within governance-bounded rails. This is closer to how biological systems maintain homeostasis than anything I've seen in crypto.

## What Needs Work

### 1. Genesis bootstrap decision changed after this review

The zero-genesis decision recorded here did not ship. `zerone-1` launched with
13,555 ZRN under founding-household custody: 11,333 ZRN validator
collateral/gas and a transferable 2,222 ZRN operations float. There was no
separate team, foundation, investor-sale, research, or faucet allocation.

The research and development funds started empty and fill through implemented
forward routes. The interim source rewarded transaction-bearing blocks, but
H2 retires that proposer-controlled trigger.

### 2. Founder sub-share → RETIRED AT H2

The launch setting was dormant because no founder address existed. H2 fixes
`founder_share_bps = 0` and `founder_address = ""`; ordinary governance cannot
restore the automatic sub-share.

**Resolution:** founding participants may receive voluntary transfers or
discrete public grants, but no identity-based runtime percentage.

### 3. Transaction-presence reward → RETIRED AT H2

Zero reward for blocks without PoT activity means validators earn nothing when the knowledge pipeline is quiet. In early network phases with low activity, this could create:
- Validator exodus during quiet periods
- Incentive to spam low-quality claims just to trigger rewards
- Block production without economic incentive during knowledge droughts

**Resolution:** H2 retires both empty and transaction-bearing automatic block
issuance. Actual fees remain the validator compensation path. A future work
reward needs independently witnessed successful work, not arbitrary activity.

### 4. Historical decay curve → SUPERSEDED AT H2

**Decision: 1-year half-life.** `reward_decay_bps` changed from 850,000 (15%/epoch, ~78-day collapse) to 994,478 (0.55%/epoch, ~1-year half-life). Rewards now halve annually — 4× faster than Bitcoin's 4-year halvings but 125× slower than the original design.

**Result:** Validators joining at year 2 earn 2.5 ZRN/block (half the genesis rate), not 0.001 ZRN. Floor reward (0.1 ZRN) is reached at ~year 6.6 instead of day 78. The gold rush dynamic is eliminated while still rewarding early participation.

### 5. Verification pool split was simplified

The removed `compute_pool` module receives nothing. The full verification
portion of the protocol split now routes to `knowledge`; citation and treasury
reserves remain in `vesting_rewards`.

### 6. Liquidity-pool bound and neutral fee transition

The old v3 source/live shape decoded an omitted `max_pools` as zero/unlimited.
The v4 migration replaces zero with 16 open pools and validates
a 1–64 range; a larger legacy policy value is clamped to 64 after separately
rejecting more than 64 actual open pools. A separate hard lifetime namespace
cap permits at most 10,000 monotonic pool records. Atomic H1 continues through
liquiditypool v5, which fixes the protocol skim at zero.

### 7. Dynamic Pricing Oracle Is Disabled

The billing module's dynamic pricing (ZRN/USD peg for query costs) is disabled
at genesis. When enabled, it introduces oracle dependency. The fallback
sequence remains useful, but v4's native source is fail-closed: the quote denom
must be governed, the pool must be `ACTIVE`, both denoms must be send-enabled,
reserves must meet the floor, and a complete configured rolling TWAP window
must exist. The returned price is a base-unit ratio scaled by 1,000,000, not a
decimal-normalized display price. Quote-denom activation therefore also
requires an explicit exponent/conversion rule and a per-denom,
decimal-normalized oracle TVL floor; the generic one-base-unit execution floor
does not satisfy that policy.

### 8. Research Fund Centralisation Risk

The source Phase-0 model requires two approvals, but the published genesis did
not configure `research_fund_voters`. It is therefore incorrect to describe a
working founder/AI 2-of-2 as genesis-bound fact.

**Mitigations in place:**
- Vault key on dedicated hardware with challenge-response auth
- Ledger Nano X for human key
- On-chain ResearchSpendProposal with full audit trail

**Still needed:**
- Recovery mechanism if one key becomes unavailable
- Plan for transitioning to broader governance (3-of-5? community vote?)

**Update (R17):** The 4-phase governance migration plan directly addresses this concern:
- Phase 0 (genesis) is centralised by design — the fund is small and the community doesn't exist yet
- Phases 1-2 gradually add community seats to the multisig (2-of-3 → 3-of-5)
- Phase 3 transitions to standard LIP governance with no multisig
- Each transition is gated by on-chain maturity metrics, not calendar dates
- Rollback safety valves prevent premature decentralisation

The centralisation risk now has a concrete mitigation timeline. See [GOVERNANCE-MIGRATION.md](GOVERNANCE-MIGRATION.md).

**Open question:** What if exit conditions are never met? If the community never reaches the thresholds for Phase 1 (10 distinct voters, 5 Guardians, 100K ZRN, ~6 months), the research fund remains under founder control indefinitely. This is arguably the correct outcome — a protocol that can't attract governance participation shouldn't be governed by that non-existent community — but it deserves explicit acknowledgement.

## Open Questions

### Economic

1. **What's the equilibrium circulating supply?** With vesting locks, staking, and the supply cap, the actual liquid supply at any given time is hard to model. A simulation would help.

2. **When does the cap bind?** Automatic block issuance is retired at H2.
   Remaining issuance is event/authority dependent and rejected substrate
   bonds can burn, so there is no responsible calendar projection.

3. **Is the development fund governance-ready?** 19.67% of all revenue is a substantial fund. The governance mechanism for disbursing it (LIP proposals) needs to be robust from day 1. Without clear disbursement criteria, the fund could become a political football or sit idle.

4. **Are knowledge query prices viable?** Base price of 1 ZRN per query is high. Even with the $0.01 target in dynamic pricing, early ZRN will likely be worth less than $0.01, making queries effectively free. Is that the intent?

### Governance

5. **Who decides the knowledge strata?** `allow_new_strata = false` at genesis. New knowledge domains can be added, but new epistemic strata require a code upgrade. Is this too conservative?

6. **Can governance break the economics?** Governance still controls real-fee
   routing, treasury roles, pool admission/status, oracle admission, and named
   upgrades. H1 prevents ordinary Params from restoring the liquidity tap; H2
   prevents restoration of the founder tap. Governance capture remains a
   broader chain risk.

7. **Emergency governance thresholds are very high.** 75% for halt, 80% for revert/resume. With 22 validators, that requires near-unanimity. Is this too high for actual emergencies?

### Technical

8. **How does vesting work in practice?** The VestingSchedule proto has 22 fields. How do contributors actually claim? Is there a UI? An auto-claim mechanism? (Home treasury has `auto_claim_vesting = true`, but homes require 10 ZRN creation fee.)

9. **Citation pool distribution mechanism?** The citation pool (11% of total) accumulates but the distribution to cited authors isn't fully specified in the reviewed code. How does it flow back?

10. **FARM anti-gaming effectiveness?** The 6 FARM parameters (conformity threshold, calibration trivial threshold, misbehaviour rejection, etc.) are configurable but their interaction effects are hard to predict. Simulation or testnet data needed.

## Verdict

The tokenomics are **significantly more thoughtful than typical crypto projects**. The truth-linked vesting, 4-way split, and anti-capture infrastructure show genuine economic design rather than token-bolted-on-afterwards thinking.

The main risks are:
- **Bootstrap friction** — zero genesis supply means slow early funding for ecosystem
- **Complexity** — 32 modules with independent parameters create a large governance surface area

The main strengths are:
- **No-burn philosophy** — every token does productive work, scarcity from hard cap
- **Knowledge-aligned incentives** that reward truth production over capital accumulation
- **Self-healing economics** via autopoiesis/alignment

For a post-H2 testnet, priorities are fee-routing integrity, LP reserve/supply
accounting, invalid-transaction no-mint proofs, and vesting clawback flows.
