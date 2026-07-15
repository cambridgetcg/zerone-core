# Zerone — where we are, and how I'd proceed (2026-07-15)

*Written at Yu's invitation ("feel free to modify and upgrade the protocol or
simply write down what you think… Zerone is alive, we can always change it based
on experiences"). This is opinion, not decree — the founder's calls are marked.*

## The frame: a living protocol

Zerone is not a finished artifact we defend; it's an organism we tend. The right
posture is the one we just used for the doctrine rescue: **notice a lived
problem → understand it from code + live chain → fix it → ship it through a
coordinated, tested upgrade → keep the record honest.** Every change is
reversible-by-governance and provable-on-chain. That discipline is what lets us
be bold: we can change anything because we can change it *safely and visibly*.

Three things are true right now, and they set the order of work.

---

## 1. Shipping today: the doctrine rescue ✅ (in flight)

`doctrine-metabolism-exempt-v1` — the one finding with a clock. The 47 genesis
doctrine facts were born starving and were ~18h from PRUNED (the chain
displaying its own constitution as extinct-by-disuse). The fix: doctrine lives
by process (creed pin + amendment LIP), not by market — `ProcessMetabolism`
skips the doctrinal stratum, and the handler resurrects the 47. *Starvation is
not falsification* — the same shape as C2. Built, drill-tested, shipping via the
proven gov→halt→deploy pipeline. **This closes the deadline.**

## 2. The framework-critique batch (v2) — proposed, gated on 4 answers

The 2026-07-10 review found 27 confirmed issues. Doctrine was the urgent one;
the rest have no clock, so we do them *right*, batched into one upgrade after
your four remaining values calls. My recommendations:

- **Alignment born-critical** (the silent 2× fee, 3 dead sensors on the empty
  custom-staking registry). *Recommend:* zero-data dimensions return **neutral**
  until there are participants to measure — "infant ≠ failing" (the COMPASSION
  read). Fixes the 2× tax on every truth-seeker and makes health mean what it says.
- **Sensor bugs** (bankKeeper denominator ~740× understated; staking/economic
  dimensions duplicated = 40% of composite double-counted; diversity off-by-one
  = dead since genesis). *Recommend:* fix all — these are clean BUGs (code
  contradicts its own comments).
- **The 0.1-ZRN CONTESTED-lock / invisible-death bug** — route the EXPIRED branch
  through `CompleteRound` with INCONCLUSIVE. One fix closes the griefing vector
  and restores counter honesty. *Recommend:* fix.
- **Forced unanimity at min panel** (ConfidenceThreshold 770k→750k so 3-of-4
  lands, one honest dissenter tolerated). *Values Q2:* deliberate epistemic
  conservatism, or a 770k-vs-750k accident?
- **The 100-ZRN witness wall** (hardcoded, circular — you can't earn a seat by
  verifying below it). *Values Q3:* capital-gate vs qualification-track-record?
- **Calibration cliff** (1-accepted-1-disproven = 0 = "serial liar"). *Values
  Q5:* the monotone-safe fix shapes are in the critique doc.

Everything consensus-affecting ships as one `framework-fixes-v2` upgrade, each
change separately justified, calibration changes as a documented recompute (the
C2 monotonicity promise).

## 3. The exchange channels — 打通 other crypto in & out

The substrate already exists; the channels are **built-but-not-opened**:

- **IBC transfer + ICA are wired** (ibc-go v8), guarded by `x/ibcratelimit`, and
  we hold hermes relayer keys for zerone-1. This *is* the Cosmos-ecosystem rail.
- **`x/liquiditypool`** is hardened (Phase-1 gates shipped in
  `liquiditypool-hardening-v1`) but gated shut: `maxPools=0`, 0 pools, empty
  quote-denom allowlist — deliberately closed until real liquidity arrives.
- **`x/substrate_bridge`** is the *work→value* lane (external attestation → ZRN),
  not a crypto↔crypto bridge — a different, already-live pipe.

The DeFi plan's own one-liner is the map: *earned ZRN → wrapped → ZRN↔token pool
→ TWAP → priced everywhere ZRN goes via IBC.* So opening the exchange = three
phases:

**Phase E1 — Establish an IBC channel (the front door for other crypto).**
Relay a transfer channel from zerone-1 to a liquidity-rich hub. My pick:
**Osmosis** (deepest DEX + IBC hub) or **Noble** (native USDC issuer — cleanest
stablecoin in). Osmosis gives the widest reach; Noble gives the cleanest on-ramp
asset. *Recommend Noble-for-USDC first* (a stable unit of account matters more
than breadth for a young chain), then Osmosis. Cost: relayer config + one
channel handshake; hermes keys already exist. Reversible.

**Phase E2 — Seed the first pool (the on/off ramp).**
Flip `maxPools` > 0 via gov param, add `uusdc`(IBC) to `billingQuoteDenoms`, and
open a **ZRN ↔ ibc/USDC** pool with real initial liquidity (`minInitialLiquidity`
is 10,000 ZRN today — tune it). Now ZRN has a price and anyone can swap in/out.
The hardening gates (fail-closed billing oracle, protocol-fee-to-collector,
locked add/remove, ZRN-quoted floor, 10% swap-fee ceiling) are already in place.
*This is the actual "exchange in and out."*

**Phase E3 — Widen (later, on demand).**
More pairs (ZRN↔ATOM, ZRN↔OSMO) once the first works; an EVM lane via
`substrate_bridge` adapters or a third-party bridge if non-Cosmos crypto matters.
No rush — E1+E2 already give a working two-way door.

*Values Q6 (new):* what is ZRN's monetary posture at the door? A Proof-of-Truth
coin with a 222,222,222 cap and zero pre-mine entering an open market needs a
stance — is the pool a genuine price-discovery venue, or a thin bootstrap ramp
with honest "early + illiquid" labeling? I lean the latter, loudly labeled.

---

## How I'd proceed, concretely

1. **Now:** land the doctrine rescue (in flight).
2. **This week:** you answer the four v2 values questions (or say "your call" on
   any); I build + drill + ship `framework-fixes-v2`. The silent 2× tax is the
   biggest quality-of-life win and I'd prioritize it inside that batch.
3. **Then the fun part:** Phase E1 — I relay an IBC channel to Noble (or your
   pick), rehearse on testnet, open it on mainnet. Phase E2 — seed the ZRN↔USDC
   pool. Now the ecosystem has a two-way door and ZRN has a market price.
4. **Ongoing:** treat each as an experiment — ship, watch the chain live with it,
   adjust. The upgrade discipline makes that safe.

The honest throughline: we spent this session making the kingdom *legible and
welcoming* (the atlas, the doors, Xenia). This is the same move applied to the
economy — make the value **legible** (a real price) and the door **welcoming**
(swap in and out freely, nothing extracted, honestly labeled). Same soul, harder
substrate.

🐍🔥❤️
