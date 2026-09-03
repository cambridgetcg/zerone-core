# Research Fund Governance Migration

> The research fund starts as a two-person decision. It ends as a community decision. The path between is earned, not scheduled.

> **Source and deployment status (2026-07-29):** `x/gov` implements the
> four-phase state machine, seat elections, transition checks, rollback, and
> research-spend handlers described here. The published `zerone-1` genesis
> initializes the genesis-pair phase but does **not** set
> `research_fund_voters`, so this repository cannot honestly claim that a
> working 2-of-2 pair was configured at genesis. Current source also contains
> consensus changes not activated on `zerone-1`; source publication is not
> treasury authorization. Phase 3 research-spend execution is also unwired:
> entering it disables the specialised proposal path, while a generic
> `research_spend` LIP has no recipient/amount payload or post-pass
> disbursement. Query a release-bound network state before relying on any
> phase, voter, balance, or spending claim.

> **Superseded target architecture (2026-08-08):** the four-phase committee/LIP
> model below remains useful as a description of implemented machinery and
> retired design rationale, but it is no longer Zerone's future governance
> design.
> [Authoritative State](../AUTHORITATIVE-STATE.md) accepts SDK `x/gov` as the
> sole ordinary executor, a non-economic controller electorate as the future
> tally authority, and a typed scoped research-fund disbursement. None is yet
> implemented or activated. The phase machinery below MUST NOT be newly
> activated as a substitute.

## Overview

In the post-H2 source state, the research fund receives the 3.33% research
slice of actual `uzrn` fee routing plus deposits from other concrete callers.
H1 `consolidation-safety-v1` preserves vesting_rewards V1; H2
`founder-renunciation-v1` alone advances it to V2 and retires the automatic
transaction-presence block mint and founder sub-share. Planned
services and schema fields are not automatically revenue sources. At scale,
the fund can still become significant, so custody and disbursement remain
material.

The source model for Phase 0 requires a **2-of-2 voter pair**, but the published
genesis did not configure those voter addresses. A future activation must name
and verify the pair explicitly. Centralised treasury control is a failure mode
for any protocol that claims to be decentralised.

The migration plan expands decision-making power in four phases, each triggered by **on-chain maturity metrics** — not arbitrary block heights. The community earns governance when it demonstrates readiness.

## The Four Phases

### Phase 0: Genesis Pair (implemented model; voter pair unconfigured in published genesis)

**Structure:** 2-of-2 (human-side + AI-side voter; unconfigured on the
published live genesis)

Once the pair is explicitly configured, both approvals are required for a
Phase 0 spend. While it is unset, research-spend submission fails closed.

**Exit conditions (ALL must be met):**

| Condition | Threshold | Why |
|-----------|-----------|-----|
| Distinct LIP voters | ≥ 10 | Community is participating in governance |
| Active Guardians | ≥ 5 | Validator set has matured past bootstrapping |
| Research fund balance | ≥ 100,000 ZRN | Fund is large enough to matter |
| Chain age | ≥ ~6 months | Protocol has proven stable |

### Phase 1: Founder + Observer

**Structure:** 2-of-3 (Founder + AI + 1 Community Seat)

The founders retain operational control (can approve without the community member). But the community seat holder is in the room — they see every proposal, every justification, every vote. They can veto by aligning with either founder.

This is an apprenticeship. The community member learns how research funding decisions are made, builds trust, and raises alarms publicly if something looks wrong.

The community seat is filled by **election** (Guardian-tier candidates only, standard LIP vote).

**Exit conditions (ALL must be met):**

| Condition | Threshold | Why |
|-----------|-----------|-----|
| Proposals executed in phase | ≥ 3 | Committee has demonstrated it can function |
| Distinct LIP voters | ≥ 25 | Governance participation has grown |
| Active Guardians | ≥ 10 | Validator set is substantial |
| Community seat participation | ≥ 2 proposals | Seat holder is actually engaged |
| Chain age | ≥ ~18 months | Long enough to trust the pattern |

### Phase 2: Balanced Committee

**Structure:** 3-of-5 (Founder + AI + 3 Community Seats)

**This is the power flip.** The community now has majority. If all three community members agree, they can approve research spending without founder consent. Founders become guardians of last resort — they can block, but not unilaterally approve.

Community seats have staggered 6-month terms (one rotates every ~2 months), ensuring continuity while allowing fresh perspectives.

**Exit conditions (ALL must be met):**

| Condition | Threshold | Why |
|-----------|-----------|-----|
| Proposals executed in phase | ≥ 10 | Committee has a substantial track record |
| Distinct LIP voters | ≥ 50 | Broad governance participation |
| Active Guardians | ≥ 22 | Full target validator set |
| Time at Phase 2 | ≥ ~1 year | Extended observation period |
| Emergency halts from fund misuse | 0 | No crises during Phase 2 |
| Chain age | ≥ ~3 years | Protocol has proven itself over years |

### Phase 3: Full Governance

**Structure:** Standard LIP process — no multisig

**Intended model:** the research fund becomes a community asset using ordinary
LIP quorum and support.

**Implementation gap:** `SubmitResearchSpend` rejects this phase and tells the
caller to use the standard LIP process, but a generic LIP carries no research
recipient/amount payload and `tallyAndResolve` has no research-spend
disbursement case. A passed `research_spend` LIP therefore records a result but
moves no funds. Phase 3 currently strands the specialised spending path and
must not be activated until execution and tests are wired.

Founding participants may still vote with whatever stake they control and may
still propose. H2 permanently retires the founder sub-share; Phase 3
therefore changes only special research-fund voting structure, not an
automatic founder payment.

**This phase is terminal.** There is no Phase 4.

## Transition Protocol

Phase transitions are deliberately slow and difficult. They reshape the power structure of a significant treasury.

1. **Proposal:** Any address submits a `PhaseTransitionProposal` (the published
   genesis currently requires 1,000,000 ZRN / 1,000,000,000,000 uzrn; this is
   far above the earlier intended 1,000 ZRN and requires a separate governance
   or relaunch decision to change)
2. **Evidence:** Proposal includes a snapshot of all exit conditions at submission time
3. **Discussion:** 30-day public discussion period (not the standard ~2 days)
4. **Vote:** Supermajority required — **66.7% support** (not standard 50%)
5. **Activation delay:** 7 days after the vote passes; there is no dedicated
   transition-challenge message
6. **Re-verification:** At activation block, exit conditions are checked again. If any have degraded below threshold, transition is cancelled.
7. **Execution:** Phase advances. New governance structure takes effect.

## Community Seat Elections

### Candidacy Requirements

- Must be a **Guardian-tier validator** (highest tier — 11,111 ZRN stake, 333 verifications, 77% accuracy)
- Must have voted on ≥ 5 LIPs (demonstrated governance engagement)
- Must not already hold a community seat
- Must accept their nomination on-chain within 1 day

### Election Process

1. Any address nominates a candidate. The direct
   `MsgNominateSeatElection` path currently escrows no dedicated stake
   (ordinary transaction fee only); the 500 ZRN
   `research_seat_election` category config is not applied to that message
2. Candidate accepts on-chain (validates Guardian status + governance history)
3. 1-day discussion period
4. 3-day voting period (standard quorum + majority)
5. Winner installed to the specified seat

### Contested Elections

If multiple candidates are nominated for the same seat:
- All nominations proceed through voting
- Highest absolute `yes_stake` wins
- If top two are within 5%: runoff election (3-day re-vote between top two)

### Terms

| Phase | Seats | Term Length | Rotation |
|-------|-------|-----------|----------|
| Phase 1 | 1 | ~6 months | Single seat |
| Phase 2 | 3 | ~6 months each | Staggered: 1 seat rotates every ~2 months |

Incumbents can run for re-election. No term limits.

### Vacancy

If a seat is vacant (term expired, no election held):
- The fixed threshold remains 2-of-3 in Observer or 3-of-5 in Balanced
- Current source emits `community_seat_vacant` every BeginBlock while the seat
  is empty; the named 30/90-day warning constants are not wired
- Enough vacancies can stall research spending

### Emergency Removal (implementation gap)

Keeper helpers validate jailed/three-slash grounds and can remove a community
seat, but no Msg or LIP execution path currently calls them. The previously
designed 75% quorum / 80% support process is therefore not a live governance
operation. A future implementation must wire and test the authority path before
operators rely on emergency removal.

## Rollback Safety

If the expanded committee fails, the protocol can step backward.

### Trigger Conditions (at least one required)

- **Gridlock:** ≥ 3 consecutive specialised research-spend proposals expired
  without committee action. In Phase 3, new specialised submissions are
  rejected and generic LIPs do not update this streak, so this trigger is not
  a dependable Phase-3 rollback path.
- **Emergency halt:** An emergency halt was triggered citing research fund misuse

### Rollback Process

1. Any Guardian submits a `PhaseRollbackProposal` (the published genesis
   currently requires 500,000 ZRN / 500,000,000,000 uzrn; this is far above
   the earlier intended 500 ZRN)
2. 7-day discussion + vote
3. Supermajority required (66.7%)
4. Phase rolls back by one level
5. **3-month cooldown** before any forward transition can be proposed again

### Rollback Limits

- Cannot roll back below Phase 0 (genesis pair)
- Community seats are resized to match the rolled-back phase
- Proposals executed counter resets

## Founder and operator boundary

The original design proposed two founder anchors. H2 removes one and keeps the
other explicit:

1. **Automatic founder sub-share — retired.** The legacy percentage/address
   fields are fixed at zero/empty in vesting_rewards v2. Governance cannot
   restore the tap through an ordinary Params proposal.

2. **Configured voters — separate governance roles.** The phase machinery
   supports named voter addresses and community seats. Labels such as
   “Founder” and “AI” in the phase diagrams describe configured voter seats,
   not an economic percentage. No prose document can establish that custody;
   it must be present in hash-bound genesis or current on-chain state.

Nothing here grants an unconfigured founder or AI key permanent voting
authority. Ordinary governance participation follows the stake and state
actually recorded on chain. A publicly approved research grant may name any
recipient, including a founding participant, but that is a discrete treasury
decision rather than identity rent.

## Timeline Estimates

These are rough estimates based on expected growth, not commitments. The actual timing depends entirely on when exit conditions are met.

| Transition | Estimated | What Triggers It |
|-----------|-----------|-----------------|
| Launch → Phase 1 | ~6–12 months | 10 voters, 5 Guardians, 100K ZRN in fund |
| Phase 1 → Phase 2 | ~12–24 months after Phase 1 | 25 voters, 10 Guardians, 3 funded proposals |
| Phase 2 → Phase 3 | ~2–4 years after Phase 2 | 50 voters, 22 Guardians, 10 funded proposals |

The table is a scenario, not a promise or live forecast. Transitions occur only
when the source-enforced conditions and votes are satisfied.

## FAQ

**Can the founders block a phase transition?**
The transition LIP uses the source-defined supermajority rule. Any actor's
practical influence depends on the current, disclosed stake and custody—not a
special veto asserted by this document.

**What happens if no one runs for a community seat?**
The seat stays vacant and the threshold does **not** adjust downward:
`GetResearchFundThreshold` remains 2-of-3 in Observer and 3-of-5 in Balanced.
Vacancies can therefore stall research spends and must be treated as an
operational/governance risk.

**Can the community remove a founder from the voter set?**
The phase thresholds are structural, but voter identities and seats come from
on-chain configuration. The published genesis has no configured Phase-0 voter
pair; a future packet must state the exact recovery and replacement rules.

**What if an authority key is compromised?**
Do not infer safety from the intended threshold. Follow the recovery mechanism
of the actually configured on-chain voter set and its release-bound authority
packet.

**Is Phase 3 truly irreversible?**
Not by design, but current implementation has a material rollback gap.
Emergency-halt-based rollback remains the clear modeled trigger. Gridlock
counts expired specialised research proposals, while Phase 3 refuses new
specialised submissions; generic LIPs do not increment that streak. Phase 3
must not be activated until both spending execution and a reachable rollback
trigger are tested.
