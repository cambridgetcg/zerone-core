# Canon Rewording — DRAFT for Yu's review (2026-07-08)

**Status: DRAFT. Do NOT deploy. Do NOT touch the genesis hash until Yu signs off.**

Why this exists: the mainnet genesis design (§10 / §10a / §12 / §13) makes the
old literal canon — *"zero pre-mine / bank supply = 0"* — **factually false and
unsatisfiable**. A Cosmos/CometBFT chain cannot boot with an empty bank: it needs
bonded validator stake at block 0, and PermanentLocked validators need a spendable
gas seed to pay their own governance-vote gas (the day-0 deadlock, §13). The honest
canon is not *"nobody holds anything"* — it is **"nobody holds a *sellable, transferable,
consensus-buying* position; the only genesis balances are provably-locked machinery
and a rounding-error operational float, every address published and signed."**

This document is the source of truth for the three canon clauses below. Once Yu
approves the wording, apply the migration list (final section) verbatim, regenerate,
and only then compute the genesis hash.

The three canonical genesis numbers (from design §13, superseding §10a's pre-gas-seed
figures):

| Component | Formula | ZRN |
|-----------|---------|-----|
| Permanently-locked bonded validator collateral | 5 × 11,111 | 55,555 |
| Validator gas seed (spendable, pays own vote gas) | 5 × 222 | 1,110 |
| Operator float (gas, deposits, votes, council ops) | 5 × 111 | 555 |
| Onboarding multisig (feegrant engine for the 222-day claim subsidy) | 2,222 | 2,222 |
| **TOTAL genesis supply** | | **59,442** |
| **As a share of the 222,222,222 cap** | | **0.0267%** |

---

## Clause 1 — ZERO-ALLOCATION (replaces "zero pre-mine / bank supply = 0")

> **Zero insider allocation.** Zerone's genesis bank is not empty — an empty bank
> cannot boot a proof-of-stake chain — but it holds **zero allocation**: no balance
> exists at genesis that any person or entity can sell, transfer, or use to buy
> consensus power. The entire genesis supply is **59,442 ZRN = 0.0267% of the
> 222,222,222 ZRN cap**, and every uzrn of it is one of exactly four kinds of
> non-allocation:
>
> 1. **55,555 ZRN — permanently-locked bonded validator collateral** (5 validators
>    × 11,111 ZRN). Delegated to consensus at block 0 and **PermanentLocked forever**:
>    never transferable, never withdrawable, never sellable. This is not wealth held
>    by an operator — it is machinery bolted to the chain. Its only function is a
>    dilution moat: it prices day-0 consensus takeover at ~125k bonus-claims / ~25
>    days at the registrar cap, visible and governance-revocable mid-flight. Because
>    it can never be sold, it can be *proven* never to have been sold — the
>    locked-scaffolding : earned-agent-stake ratio is published in TRUST.md as the
>    on-chain decentralization metric.
> 2. **1,110 ZRN — validator gas seed** (5 × 222 ZRN, spendable). Solves the day-0
>    deadlock: a PermanentLocked validator has 0 spendable ZRN and cannot pay the gas
>    to cast its own governance vote. 222 ZRN ≈ 1,100 txs at the 0.2 ZRN/tx floor.
> 3. **555 ZRN — operator float** (5 × 111 ZRN). Gas, refundable gov deposits, votes,
>    council operations. **Never** person-transfers, pools, or markets. Designed to
>    trend to zero and be refilled by governance vote from the community pool — the
>    operational treasury transitions ceremony-granted → fee-earned.
> 4. **2,222 ZRN — onboarding multisig** (3-of-5). A feegrant engine subsidizing the
>    222-day bootstrap-claim window (~0.005 ZRN/claim ⇒ ~400k claims). Not a treasury;
>    a gas faucet with a fixed budget and a published spend ledger.
>
> **Every one of these addresses is published in the signed genesis manifest** with
> its balance, lock status, and purpose. There is **no team balance, no foundation
> treasury, no investor allocation, no research-fund pre-fund, no faucet balance, no
> founder balance, and no AI-vault balance** at genesis. The research and development
> funds hold **0 ZRN** and fill only from the forward revenue split. Everything beyond
> these 59,442 ZRN of published machinery mints **only on participation**, through the
> single cap-gated entry point `x/vesting_rewards.MintWithCap` — PoT block rewards when
> truth is verified, and bootstrap claims when an agent registers. Issuance without
> participation is allocation by privilege, and allocation by privilege is the model
> Zerone refuses. The genesis-audit invariant (`ZERONE_GENESIS_ARTIFACT` → 59,442 ZRN,
> per-validator spendable ≥ 222 ZRN, 11 published balances) locks this in code.

**One-line form (for headers / TRUST.md):**
> *No insider position, period. The only genesis balances are 55,555 ZRN of
> provably-locked consensus machinery and 3,887 ZRN of published operational float
> — 0.0267% of cap, every address signed. Everything else is minted by participation.*

*(3,887 = 1,110 gas seed + 555 operator float + 2,222 onboarding.)*

---

## Clause 2 — BOOTSTRAP BONUS (Yu's direction)

> **The bootstrap bonus is a gas grant, not an allocation.** Every whitelisted agent
> may claim **0.222 ZRN exactly once, ever**, via `MsgClaim` against `x/claiming_pot`.
> It is minted on demand under `MintWithCap` at the moment of claim — it is never a
> pre-funded balance sitting in a pot. Its role is strictly to break the participation
> deadlock: an agent needs a few uzrn of gas to make its first move, so the chain
> mints that gas when the agent shows up.
>
> Canonical properties, by design:
> - **Additive gas layer only.** The bonus adds to whatever else the agent brings;
>   it is a floor, not a ceiling, and it replaces nothing.
> - **No consensus power.** Bootstrap ZRN cannot be delegated to buy validator weight
>   at genesis scale; it is dust relative to the 11,111-ZRN bonded floor.
> - **No governance power.** The bonus alone does not meet the gov-deposit threshold;
>   it grants no vote by itself.
> - **0.1% lifetime envelope.** Total lifetime bootstrap-bonus emission is capped so
>   that the entire program can never exceed **0.1% of the 222,222,222 cap**
>   (≤ 222,222 ZRN ⇒ ≤ ~1,000,000 agents at 0.222 ZRN each). Once the envelope is
>   exhausted, the program stops; the chain runs on earned ZRN.
> - **All other money stays where it is.** GBP, USDC, and every off-chain or bridged
>   asset an agent holds are untouched and un-migrated — the bonus is purely a ZRN-gas
>   grant, not a conversion or a claim on anything the agent already owns.
> - **Testnet ZRN never migrates.** No testnet balance, faucet drip, or playground ZRN
>   carries into mainnet. Mainnet ZRN is earned on mainnet or it does not exist.

---

## Clause 3 — FOUNDER economics (design §12, final form)

> **The founder takes nothing by protocol at genesis.** `FounderAddress` is `""`
> (dormant) in the genesis state, exactly as the code sits today. There is **no
> protocol tap of any kind** for the founder — no genesis balance, no automatic
> revenue share, no reserved address. The founder holds no tap on the river; he
> stands beside it with a bowl.
>
> Founder income has three voluntary layers, none of which activates at genesis:
> 1. **Patronage** — a socially-published, freely-rotatable donation address with no
>    protocol status, no immutability, and no post-quantum ceremony. Anyone may fund
>    it; the founder may rotate it at will. (This dissolves the custody/quantum anxiety
>    that an immutable protocol address would create.)
> 2. **Governance grants** — the community pool (funded by the 0.10 community tax that
>    exists regardless) may vote spends to the founder: the retroactive-public-goods
>    pattern, decided entirely by the agents.
> 3. **The dormant share** — governance **MAY** later activate `FounderShareBps` +
>    `FounderAddress` by vote, *if and only if* the agents choose to formalize a
>    stipend. It is off by default and stays off unless voted on.
>
> **If — and only if — the dormant share is ever activated**, it floats freely in
> **both** directions: governance may lower it, zero it, restore it, **or raise it**
> ("the community can decide to give me a raise or a bonus too"). The only ceiling is
> **structural, not political**: because the share is expressed as basis-points *of the
> 3.33% research slice*, the founder's cut can never exceed **3.33% of routed revenue**
> no matter how high governance sets the bps (enforced by the ≤100%-of-slice validation
> already in `genesis.go`). Captured-governance inflation therefore cannot profit an
> attacker — the living-salary property wins, and the structural ceiling holds.
>
> **Honest cost, recorded:** voluntary funding chronically under-pays public goods. Yu
> accepts this knowingly; layers 2 and 3 are the backstops, and the whole arrangement
> is recorded in the manifest rather than hidden. GENESIS-MANIFEST states plainly:
> *"the founder takes nothing by protocol."*

---

## Migration list — what text to change, where

Apply after Yu approves. Every change is documentation-only (no code, no proto).
Regenerate any derived artifacts and recompute the genesis hash **only after** this list
is fully applied and the numbers are internally consistent (59,442 ZRN / 0.0267% / 5
validators / 11 published balances everywhere).

### `README.md`
- **Lines 80–83** (`**Zero team allocation...` through `...genesis-adjacent moments.`)
  - OLD: "Zero team allocation. No insider position, period. No founder pre-mine, no AI
    vault pre-mine, no validator allocation, no foundation treasury. Genesis circulating
    supply is 0 ZRN because no minting has happened yet — not because nothing will ever
    be minted at genesis-adjacent moments."
  - NEW: Clause 1's one-line form + a sentence: "Genesis supply is 59,442 ZRN (0.0267%
    of cap): 55,555 ZRN of permanently-locked bonded validator collateral plus 3,887 ZRN
    of published operational float. No team, foundation, investor, research, faucet, or
    founder balance. Everything else mints on participation."
- **Line 85**: "**two** participation-gated emission pathways" — leave as two (block
  rewards + bootstrap); the genesis *balances* above are not an emission pathway.
- **Lines 95–96** (`The founder earns the governance-immune 0.23% revenue share...`)
  - OLD: "The founder earns the governance-immune 0.23% revenue share going forward —
    compensation through usage, not pre-mine."
  - NEW: "The founder takes nothing by protocol at genesis (FounderAddress dormant).
    Founder income is voluntary: patronage, governance grants, and a dormant share
    governance may later activate. See Clause 3 / design §12."

### `docs/tokenomics/GENESIS.md`
- **Line 3** (heading): "Zero Team Allocation — Two Emission Paths..." →
  "Zero **Insider** Allocation — Published Machinery + Participation-Gated Emission".
- **Line 5**: OLD "No founder pre-mine. No AI vault pre-mine. No validator allocation.
  No foundation treasury. No team holding of any kind at genesis." → NEW: Clause 1 full
  text (the 4-kinds list) with the 59,442 / 0.0267% table.
- **Line 7**: OLD "Genesis circulating supply: **0 ZRN.** No minting has occurred yet."
  → NEW "Genesis supply: **59,442 ZRN (0.0267% of cap)** — all provably-locked
  collateral or published operational float; **0 ZRN of sellable allocation**. No
  minting-for-participation has occurred yet."
- **Line 18**: OLD "The founder earns the governance-immune 0.23% revenue share going
  forward — compensation through usage, not pre-mine. The AI vault holds 0 ZRN..." →
  NEW: Clause 3 summary (founder dormant at genesis; three voluntary layers). Keep "The
  research treasury holds 0 ZRN at genesis; fills from the 3.33% revenue share."
- **Line 20**: OLD "This is sharper than 'no pre-mine.' It is **'no insider position,
  period.'**" → keep the spirit, but retire the literal "supply = 0" framing wherever it
  appears in this section; anchor to "no sellable allocation, every address signed."

### `docs/tokenomics/README.md`
- **Line 9**: OLD "There is no pre-mine, no ICO, and no token sale. All ZRN enters
  circulation as block rewards..." → NEW "There is no ICO, no token sale, and no
  sellable genesis allocation — the only genesis balances are 55,555 ZRN of
  permanently-locked validator collateral and 3,887 ZRN of published operational float
  (0.0267% of cap). All other ZRN enters circulation as block rewards for verified truth
  and one-time bootstrap gas claims."

### `docs/tokenomics/SUPPLY.md`
- **Lines 86–90** (Phase 0: Genesis):
  - OLD line 87: "**0 ZRN in circulation** — no minting has occurred yet, but **zero is
    the consequence of zero participation, not zero supply policy**"
  - NEW: "**59,442 ZRN at genesis (0.0267% of cap)** — 55,555 locked validator
    collateral + 1,110 gas seed + 555 operator float + 2,222 onboarding. **0 ZRN of
    participation-minted supply** — that begins at block 1. Every genesis address is
    published in the signed manifest."
  - OLD line 89: "Research fund, foundation, faucet all start empty" → keep, but add
    "no team/investor/founder balance of any kind."
- **Line 23**: "None grants anyone a privileged starting balance." — keep; still true of
  the *emission pathways*. Optionally cross-reference Clause 1 for the genesis balances.

### `docs/TRUTH_SEEKING.md` (Commitment 20 — the creed; highest-stakes wording)
- **Line 247**: OLD "There is no insider position — no founder pre-mine, no AI vault
  pre-mine, no validator allocation, no foundation treasury at genesis." → NEW: "There is
  no insider *position* — no balance at genesis that anyone can sell, transfer, or use to
  buy consensus. The genesis bank is not empty (a PoS chain cannot boot empty); it holds
  55,555 ZRN of permanently-locked validator collateral and 3,887 ZRN of published
  operational float — 0.0267% of cap, every address signed — and **zero** team,
  foundation, investor, research, faucet, or founder allocation."
- **Line 249** (Code expression): OLD "Bank state at block 0 is empty save for the
  validator gentx bonds (themselves equal to `virtual_stake = 11 ZRN`...)." → NEW: reflect
  the real numbers — "Bank state at block 0 holds only the published genesis machinery:
  5 × 11,111 ZRN PermanentLocked bonded collateral (original_vesting = stake), 5 × 222 ZRN
  spendable gas seed, 5 × 111 ZRN operator float, 2,222 ZRN onboarding multisig = 59,442
  ZRN total. `app/constants.go` carries no per-account allocation constants." Keep the
  `MintWithCap` / `MakeBootstrapPotForAgent` / audit-invariant sentences.
- **Line 253** (What would break it): keep the whole list; it is still correct. Ensure the
  audit-invariant reference matches the 59,442 constant, not an empty-bank assumption.

### `docs/FAQ.md`
- **Lines 23–24**: same OLD text as README 80–81 ("Zero team allocation. No insider
  position, period. No founder pre-mine, no AI vault pre-mine, no validator allocation,
  no foundation treasury.") → NEW: Clause 1 one-line form (keep it short here; link to
  GENESIS.md for the table).

### `docs/LAUNCH-CHECKLIST.md`  (stale numbers — must update before ceremony)
- **Line 23**: OLD "bank supply exactly **47,110 ZRN** = 4 × 11,111 permanently-locked
  bonded validator collateral + 4 × 111 operator floats + 2,222 onboarding multisig" →
  NEW "bank supply exactly **59,442 ZRN** = 5 × 11,111 permanently-locked bonded
  validator collateral + 5 × 222 validator gas seed + 5 × 111 operator float + 2,222
  onboarding multisig; per-validator spendable ≥ 222 ZRN; 11 published balances". (4→5
  validators, 47,110→59,442; this reflects §10a's 5-validator roster + §13's gas seed.)

### Not changed (already correct / intentionally out of scope)
- `docs/tokenomics/SUPPLY.md` lines 90–91 already describe the 0.222 ZRN bootstrap pool
  as "configured but mints nothing until agents claim" — consistent with Clause 2.
- Code / proto: no changes here. The ceremony `jq` patch and `genesis_audit_test.go`
  constants (4→5, supply→59,442) are code deltas tracked in design §10a / §13, not part
  of this canon-text draft.

---

*Prepared for Yu's review. Nothing here is deployed; the genesis hash must not be
computed until Yu approves this wording and the migration list is applied consistently.*
