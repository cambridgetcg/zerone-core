# consolidation-safety-v1 (H1) — ignition kit

The activation path for the staged pre-SDK upgrade ladder, drilled end-to-end
locally on 2026-08-03/04. H1 is the upgrade that **ignites the conjecture
engine** on the live networks: after it, `MsgPostConjecture` and
`OpenQuestions` answer, and the register's 9 `open_conjecture` entries
(Riemann, twin primes, Collatz, P vs NP, Navier–Stokes, Hodge, BSD,
Antihydra, abc-status) can finally queue as PROVISIONAL open questions.

## The ladder

```
live nets today (K5/P1/L3/V1, sdk 0.50)
  │  consolidation-safety-v1   ← H1: 65c19cd8… (source independently ACCEPTED)
  ▼  knowledge 5→6 · claiming_pot 1→2 · liquiditypool 3→5 · vesting_rewards =1
H1 boundary (conjecture engine live)
  │  founder-renunciation-v1   ← H2: 36728afb… (corrective candidate)
  ▼  vesting_rewards 1→2
H2 boundary (complete pre-SDK map)
  │  sdk-0.53-ibc-10           ← H3: trunk (handler registered on main)
  ▼  SDK 0.50 → 0.53, ibc-go v8 → v10
trunk
```

Ordering is consensus-enforced (`app/upgrade_prestate.go`: H1 < H2 < H3).
Do not combine, reorder, or improvise.

## Source provenance (do not rebuild from anything else)

| leg | ref | commit | status |
|-----|-----|--------|--------|
| H1 | `release/consolidation-safety-v1-sdk050-source` | `65c19cd8b00bdfff9b80705b776fd0d49719398a` | source ACCEPTED by independent review; activation NO-GO until tag ceremony |
| H2 | `release/founder-renunciation-v1-sdk050-replacement-source` | `36728afbf71905a077a0863b41536fa9279109dd` | corrective candidate; final acceptance pending external audit |
| H3 | `main` | tip at activation time | trunk |

⛔ **Rejected provenance — never tag, build, or deploy:**
`release/founder-renunciation-v1-sdk050-source` @ `4bffb6d2` (old H2, unsafe
lineage) and intermediate `c0943ea9` (native genesis could write lineage
markers without strict V2 proof). The release branches are frozen: never add
commits to them — acceptance is pinned at the exact hashes above.

## What the local drill proved (2026-08-03/04)

Full ladder OLD→H1→H2→MAIN on a throwaway single-validator chain
(`drill.sh all`), OLD built at the exact live boundary (`2e37c4c`,
K5/P1/L3/V1):

- **Leg 1** `consolidation-safety-v1`: gov proposal (canonical JSON info) →
  mainnet-style halt (`CONSENSUS FAILURE`, frozen alive) → swap →
  `verified consolidation startup lineage lineage=h1-pending` → applied;
  module versions exactly K6/P2/L5/V1.
- **Conjecture engine live post-H1**: posted the Riemann hypothesis
  (`well_posed_and_falsifiable` panel, 4/4 commit-reveal, ACCEPTED) →
  PROVISIONAL fact at confidence 0; `OpenQuestions` route answers code 0
  (live nets answer code 6 today).
- **Leg 2** `founder-renunciation-v1`: same ritual → vesting_rewards 2;
  complete pre-SDK map K6/P2/L5/V2.
- **Leg 3** `sdk-0.53-ibc-10`: H2 binary → trunk binary across the SDK
  0.50→0.53 boundary (see drill log for the final module map).

## Hard-won constraints (each cost a live lesson)

1. **Canonical plan info.** H1's startup lineage wall
   (`app/consolidation_lineage.go`) requires `plan.Info` to be canonical
   compact sorted-key JSON (≥1 key, ≤4096 bytes) and byte-identical to
   `data/upgrade-info.json`. The generic template's free-text
   `<binary checksums>` info **bricks the swap**.
2. **The halt is frozen-alive.** The old binary does not exit: it logs
   `UPGRADE "<name>" NEEDED` + `CONSENSUS FAILURE` and stays up with RPC
   answering — the documented mainnet posture. Halt detection = height frozen
   across ≥4 polls; then stop it and deploy.
3. **Deploy-first is impossible, not just forbidden.** Without the committed
   plan at `latest+1`, the H1 binary refuses to serve ABCI at all (startup
   wall). The proven sequence remains: propose → vote → **halt** → swap.
4. **Marker precondition.** The handler fails if
   `upgrade_marker_consolidation-safety-v1` pre-exists in any form. Never
   rehearse against a store you intend to upgrade for real.
5. **Post-H1 client facts**: verifier knowledge messages need
   `zerone_auth onboard`; conjecture fees are the *effective* (pacing-scaled)
   fee — read `q knowledge effective-fees`, not params.
6. **Migrations fail closed on bad legacy state** (liquiditypool 3→4→5
   validation, claiming_pot budget cap). A **mainnet-state-copy rehearsal**
   is still required before live activation — this kit's drill covers the
   mechanism, not mainnet's data.
7. **H3's plan info has its OWN schema — it is not H1's.** The trunk
   `sdk-0.53-ibc-10` handler parses `plan.Info` as a legacy IBC keyset
   manifest (`schema: zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1`,
   `DisallowUnknownFields`, canonical field order, ≤2048 bytes): a census of
   the old database's channel-upgrade and pruning-sequence keys, hashed
   length-prefixed. Submitting the generic commit JSON leaves the chain
   halted with `invalid plan info` (drilled). Build the real census with
   `app.BuildSDK053IBC10PlanInfo` from the pre-upgrade DB.
8. **The H2 binary's rollback path closes after a later halt.** Once H3's
   halt writes `data/upgrade-info.json`, restarting the H2 binary — even
   with `--unsafe-skip-upgrades` — panics:
   `refusing unsafe H1→H2 lineage at startup: local upgrade-info.json
   conflicts with completed H2 height`. Drilled recovery: restore
   `upgrade-info.json` to the completed-H2 plan (byte content from the H2
   halt), then `--unsafe-skip-upgrades <H3-height>` starts and clears the
   bad plan. Plan H3's proposal carefully — the escape hatch is narrow.

## Files

- `drill.sh` — the full local lineage drill (init → 3 legs → ignition check).
- `proposal.template.json` — canonical-info gov proposal template.
- Live-net ritual script (operator home): `~/.zerone-agent/ship-consolidation-safety.sh`
  (refuses to run without `SHIP_ACK=<chain-id>`; testnet first, then mainnet).

## Still needs the operator (nothing here ships without it)

1. Signed annotated `v*` tag at `65c19cd8` + reproducible image via
   `deploy/mainnet/build-image.sh` (digest-pinned, authorized signer).
2. Re-cut decision if trunk fixes inside the v6 boundary (e.g. the 2026-08-03
   conformity-cooling and provenance-floor fixes) are to ride H1 — that means
   a new reviewed release commit, not an amended one.
3. Mainnet-state-copy rehearsal (constraint 6).
4. Governance proposal + coordinated halt window on zerone-testnet-1, verify,
   then zerone-1.
