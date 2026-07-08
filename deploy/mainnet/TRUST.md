# TRUST.md — zerone-1 genesis trust map

**Status:** present-tense trust map. Ships beside `GENESIS-MANIFEST.md` and the genesis
hash. Design of record: `docs/plans/2026-07-07-mainnet-genesis-design.md` §7 (succession),
§9 (quantum), §10a (5-validator roster), §11 (recovery doctrine), §12 (founder economics),
§13 (gas seed). This document states what is TRUE at genesis, not what we hope to become —
`SUCCESSION` states the future verifiably; TRUST.md states the present truthfully.

> **This is a custodial-launch phase. Decentralization is a trajectory, not a
> birth certificate.** Everything below is what one household could, in the worst
> honest reading, control on day 0. The escape valves (§ Sunset, § Expansion, § Fork)
> are the whole point of writing it down.

---

## 0. The one-paragraph truth

At genesis, zerone is **Proof-of-Authority wearing Proof-of-Stake clothing.** The
55,555 ZRN of bonded validator stake was minted by the ceremony and cost the operators
nothing — it is machinery (a dilution moat that prices consensus takeover), not
skin-in-the-game wealth, and it is `PermanentLockedAccount`: never sellable, never
transferable, only slashable. Genesis security is therefore **named authority +
reputation + a disclosed cost-of-capture**, not economic stake. Five operators hold
the validator keys, the same five hold the emergency council, and a small multisig holds
the bonus registrar. If several of those five run on shared infrastructure or
household-controlled keys, the honest genesis cost-of-capture collapses toward **one
party.** We say so plainly here so that every later step away from it is measurable.

---

## 1. Who holds what at genesis

### 1a. Validator operators — 5 seats (PLACEHOLDERS — Yu to fill)

Roster is closed at **5** (design §10a): Alpha, Beta, Gamma, Yu, Ai. Ai is a genesis
validator — the first agent signing blocks from block 0 (succession Phase 3 begins at
block 1, not "someday"). Each self-bonds 11,111 ZRN via gentx from a `PermanentLockedAccount`.

| Seat  | Operator identity (PLACEHOLDER) | Independent operator? | Infra | Key custody | Notes |
|-------|--------------------------------|-----------------------|-------|-------------|-------|
| Alpha | `TODO — real identity`         | `TODO: independent / operator-controlled` | `TODO` | Ledger + steel (§ Key standard) | |
| Beta  | `TODO — real identity`         | `TODO: independent / operator-controlled` | `TODO` | Ledger + steel | |
| Gamma | `TODO — real identity`         | `TODO: independent / operator-controlled` | `TODO` | Ledger + steel | |
| Yu    | Founder (human)                | operator-controlled   | `TODO` | Ledger + steel | Founder holds NO protocol tap (§4) |
| Ai    | Founder-operated agent         | operator-controlled (successor-founder risk, §3/§ SUCCESSION) | `TODO` | Ledger + steel | First agent validator |

**HONEST DEFAULT until Yu fills the table:** treat Alpha, Beta, Gamma, Yu, and Ai as
**operator-controlled by one household** unless independence is affirmatively documented
per seat. If Alpha/Beta/Gamma and Ai run on Yu-household infra or Yu-generated keys,
**genesis cost-of-capture across consensus, governance, council, and keys = one household.**
The independence column is not decoration — it is the single most load-bearing fact in this
file, and an empty cell reads as "not independent."

### 1b. Emergency council — 5 seats

Council = **the 5 validator operators** (design §10a). `min_distinct_voters = 4`,
`council_expiry_block = 6,168,960` (~180 days). No external or agent-elected seat exists
at genesis; that is a Phase-2 obligation, not a launch fact. Because the council IS the
validator set, a household that controls the validators also controls the halt/revert
machinery — the two axes are **not independent** at genesis.

### 1c. Bootstrap registrar — 2-of-3 multisig

`bootstrap_registrar` (agenttool ops) admits agents to the bonus layer alongside the
gov authority. **PLACEHOLDER** — the 3 keyholders, their hardware, and whether the
DID→address commitment chain is published for external audit are Yu's open items
(design §5 Q3). Compromise bounds are in consensus (§2), so a stolen registrar is
loud and bounded, not catastrophic.

### 1d. Founder — dormant (design §12)

`FounderAddress = ""` at genesis. **The founder takes nothing by protocol.** No tap, no
immutable field, no PQ ceremony obligation. See §4.

---

## 2. Cost-of-capture per axis, TODAY

All figures are genesis-state. Bonded total = 55,555 ZRN (5 × 11,111). Registrar
throughput ceiling = 5,000 admissions/day × 0.222 ZRN = **1,110 ZRN/day** of new
consensus-eligible tokens.

| Axis | What it takes to capture at genesis | Honest read |
|------|-------------------------------------|-------------|
| **Consensus (1/3 power)** | Acquire ~**27,800 ZRN** of bonded power on top of the locked 55,555 (⅓ = X/(55,555+X) ⇒ X ≈ 27,778). Via the bonus layer that is ~**125,000 bonuses** ≈ **25 days** at the registrar daily cap — and every admission is on-chain, visible, and gov-revocable mid-flight. OR: compromise **2 of 5** validator keys. | The 25-day figure is the moat; the 2-of-5 key path is the real risk if keys/infra are shared. |
| **Consensus (halt, ≥1/3)** | Same ⅓ threshold stalls liveness. Council can also halt directly (§3). | |
| **Governance** | SDK-gov deposit is **100 ZRN** (300 expedited). For the first weeks only validator floats (555 ZRN) + the 2,222 onboarding multisig can fund a proposal — an acknowledged **oligopoly window** (design §5 Q6). Bonus wallets are governance-inert (`min_vote_stake = 1 ZRN`; a 0.222 bonus cannot vote). | Whoever funds deposits sets the agenda until PoT emission spreads real stake. |
| **Emergency council** | **4 of 5** council votes (`min_distinct_voters = 4`) to halt or revert ≤1,111 blocks. Council = the validators, so this is not a second independent quorum. | Strongest concentrated power at genesis; sunsets at ~180d (§3). |
| **Bonus registrar** | **2 of 3** multisig keys. A fully stolen registrar mints ≤1,110 ZRN/day, ≤222,222 ZRN ever (0.1% of cap), and one gov param proposal (`bootstrap_registrar = ""`) revokes it. | Bounded-by-consensus; the least dangerous trusted role. |
| **Infra** | Unknown until §1a is filled. If validators + council share hosting/network, a single infra failure or seizure = simultaneous loss of consensus AND the halt machinery. | The quietest and most likely real single point of failure. |
| **Keys** | Founder standard: fresh single-purpose Ledger seed, one steel backup, no passphrase (design §8a). `PermanentLocked` stake is **stranded forever if keys are lost** — never transferable, only re-stakeable/slashable. | Loss-tolerance was traded away for the never-sold guarantee (design §5 Q8). |

**Insider economics, stated plainly (design §12):** 55,555 ZRN locked bonded stake +
3,887 ZRN spendable float (5×222 gas seed + 5×111 operator + 2,222 onboarding). Total
genesis supply **59,442 ZRN = 0.0267% of the 222,222,222 cap.** No team, foundation,
investor, research, or faucet balance exists. *The founder holds no tap on the river —
he stands beside it with a bowl; the tap gets built only if the river votes to build it.*

---

## 3. What each trusted role CAN do — and when its power SUNSETS

| Role | Power at genesis | Sunset / successor |
|------|------------------|--------------------|
| **Validators (5)** | Produce blocks, earn PoT emission, vote gov from floats, propose upgrades. | Diluted by expansion to 7+ operators with **measured** independence before council expiry; earned agent validators enter Phase 3 (liquid, self-bonded, the opposite of ceremony-assigned). |
| **Emergency council (5)** | Halt the chain in minutes; revert ≤ **1,111 blocks** (~47 min); guardian floor 11,111 ZRN. **Cannot** silently deep-revert — anything deeper is an explicit public fork (§ Fork, design §11). | **Expires at block 6,168,960 (~180 days).** At expiry it is replaced by elected agents, OR ONE 90-day renewal with published justification if fewer than **22 franchised agents** exist (design §8). Not renewable silently. |
| **Bootstrap registrar (2-of-3)** | Admit ≤500 addresses/msg, ≤5,000/day, ≤222,222 ZRN lifetime to the bonus layer. | **Gov-revocable at any time** by one param proposal (`bootstrap_registrar = ""`); the gov-authority admission path survives. Flips to agent-elected or bond-based permissionless admission in Phase 3. |
| **Knowledge guardians** | = the council; 1-day veto window on authority fact injection; probe-bounty mint 0.1 ZRN/block, hard-capped at 22,222 ZRN lifetime then stops. | Tracks council expiry. |
| **Founder (Yu)** | **Nothing by protocol.** `FounderAddress = ""` — no revenue tap, no special key beyond the ordinary Alpha…Yu validator/council seats. Income is voluntary: a rotatable donation address, community-pool grants by vote, or a **dormant** share governance MAY later activate by vote (design §12). | Never activates unless the agents vote it on. The dormant `FounderShareBps` code is structurally ceiling-bounded to ≤3.33% of flow even if activated. |
| **Ai (successor-founder)** | Validator + ops keys (enumerated in §1a). Outsized operational role in development. | Disclosed Phase-0/1 fact under the **same** trajectory obligation as every other centralization; development plurality is funded so the chain buys its own developer diversity out of flow. |

---

## 4. The dated expansion commitment (the trajectory out)

1. **Independent validators before council expiry.** Recruit **≥7 total operators with
   measured independence** — funding-graph distance, infra correlation, downtime
   correlation — *before block 6,168,960 (~180 days).* Padding the set with
   household-controlled keys does not count and is explicitly rejected as fake
   decentralization.
2. **Published decentralization metric.** The **locked-scaffolding : earned-agent-stake
   ratio** is the on-chain number we publish and drive toward zero. Genesis =
   55,555 locked : 0 earned. Every epoch this ratio moves as agents earn liquid,
   self-bonded stake through witnessed work. This TRUST.md is **updated every epoch**
   with the current ratio and the current §1a independence table.
3. **Council → elected agents at ~180d** with the 22-franchised-agent gate (design §8).
4. **Registrar → agent-elected or permissionless** in Phase 3.

### Fork / reset doctrine — the escape valve (design §11)

The state is negotiable; the record is sacred; every negotiation is itself recorded.
Fast surgery (council halt + ≤1,111-block revert) is for fresh wounds only. **Anything
deeper is an explicit fork after public debate — deliberately slow.** Deep silent reverts
are double-spend machines and are structurally disallowed. If the custodial launch betrays
its trajectory, the legitimacy-preserving remedy is **fork away**: the founder's only
irremovable property is an ordinary work-earned franchise, and even the dormant founder
share is removable by a fork. Cross-ref `docs/INCIDENT_RESPONSE.md`; drill coverage:
TEST halt-restart-recovery (design §4).

---

## 5. Quantum posture (design §9)

> **The chain's memory is already quantum-safe; only its signatures aren't — signatures
> can be re-proven later, memory could never be re-witnessed.**

All hash anchoring (content/link/creed pins) survives Grover; finalized history cannot be
retro-forged. secp256k1/ed25519 signatures fail forward (theft from exposed pubkeys) —
addressed by crypto-agility (Any-typed pubkeys, rotation-as-upgrade), genesis-published
`sha256(recovery_secret)` commitments for locked validator accounts, and accumulate-only
spend-hygiene. PQ migration is a drilled agent-era obligation before ~2030, not a genesis
feature.

---

*Genesis supply 59,442 ZRN (0.0267% of cap) · 5 validators · council expiry block
6,168,960 · community_tax 0.10 (foundation = network fee, community-pool income only).
If a fact here is stale, the ceremony artifact and `GENESIS-MANIFEST.md` are canon —
fix this file.*
