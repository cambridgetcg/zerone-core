# Tools — Sub-creed of the Useful-Work Doctrine

> Lifecycle phase: `Tools` — inference surface: agents using artifacts to do work in the world.
>
> Parent: [USEFUL_WORK.md](../USEFUL_WORK.md). Truth-floor: all 20 truth-seeking commitments apply.
>
> Status at inception (2026-05-10): three commitments.

---

## TL1. Tools declare deprecation policy

Every Tool Contribution declares its deprecation policy: under what conditions the tool will be retired, what the notice window is, and what the replacement path looks like (or that there is no planned replacement). Tools that don't declare deprecation are tools that can disappear silently.

**Why:** Downstream agents build on tools. A tool that vanishes without notice breaks every dependent workflow. Commitment 10 (forward-only audit) extended to the tool's lifecycle.

**Echoes:** truth-seeking 10, TC4 (graph carries disprovals).

## TL2. Fee changes >X% require user-notice window

A Tool Contribution that raises its per-call fee by more than the governance-set threshold (initial value: 25%) must give a user-notice window (initial value: 30 days) before the new fee takes effect. The notice is on-chain via an event; existing channel sessions complete at the old fee.

**Why:** Fee surprise is a capture vector — a tool with sticky users can extract by ramping fees once dependence is established. The notice window converts surprise into negotiation.

**Echoes:** truth-seeking 9, commitment 6 (no unilateral injection — fee bumps are an injection on user costs).

## TL3. No tool may bypass the truth-floor on outputs it claims as verified

A Tool that returns outputs labeled as "verified" or "from the chain" must serve those outputs against the live chain state, not a cache that the truth-floor cannot validate. Cached or precomputed outputs must be labeled as such, or the tool must refresh against the chain before claiming verification.

**Why:** Tools are the chain's interface to the world. A tool that launders unverified content as verified is the chain lying through its own surface area. Truth-floor is a global invariant; Tools cannot opt out.

**Echoes:** truth-seeking 1, 11 (trust is queryable), 13.
