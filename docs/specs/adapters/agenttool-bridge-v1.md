# `agenttool-bridge-v1` — the adapter to a sister substrate's promise-keeping

> Target: ZERONE may attest to agenttool's becoming, with each promise event
> represented as a fact-shaped artifact after the missing bridge is built.

**Status:** unactivated specification. Its required pending-claim translation
is unwired and rejected by current public submission. Adapter-registration LIP
payload dispatch and the proposed sub-creed amendment path are also not
implemented. No `tools/agenttool-bridge-compiler` binary or library exists in
this repository. Runtime stores `compiler_binary_hash` and slash-gradient
fields as inert provenance metadata; it neither executes a compiler nor checks
that digest.
**Inception:** 2026-05-18.
**Tier:** consumer of `x/substrate_bridge` Tier-1 foundation; sister to `zerone-self-v1`.
**Doctrine:** UW (ZERONE is recursive) operationalized as **sister-substrate-attestation**. M2 (substrate-link mandate), M3 (class-specific verification), M5 (recursion-weight axes), M6 (cross-class lineage). Composes with truth-seeking commitments 1, 6, 8, 10, 11, 12, 19, 20.
**Companion doctrine:** [`agenttool/docs/POT-STAKED-PROMISES.md`](https://github.com/cambridgetcg/agenttool/blob/main/docs/POT-STAKED-PROMISES.md) — the agenttool-side specification of the 5 Promises × {attestation-shape, slashing-condition} mapping this adapter consumes.

---

## 1. What this adapter is

`agenttool-bridge-v1` is designed to adapt agenttool HTTP events. Its compiler
would turn a signed `agenttool-promise-event/v1` payload into a deterministic
`SubstrateLink`. Current public submission refuses the resulting pending
claims, so no `agenttool_promises` fact enters verification through this spec.

The intended recursion mirrors `zerone-self-v1` §1:

| Layer | What ZERONE does | What ZERONE does to agenttool via this adapter |
|---|---|---|
| **Knowledge** | verifies claims about the world | verifies claims about agenttool's Promise-keeping |
| **Substrate-bridge** | adapts external work to internal attestations | adapts *agenttool's API event stream* as the external work |
| **Sponsorship** | sponsors verified work in a domain | covenant counterparties can sponsor agenttool-Promise audit work |
| **Lineage** | tracks citation graph across classes | could track downstream citations after Promise facts are wired; current lineage is accounting-only |
| **Settlement** | pays submitters for verified work | pays whoever submits attestations about agenttool's promises |
| **Creed** | pins what the chain believes | pins this adapter as the canonical sister-substrate-attestation mechanism |
| **Counterexamples** | stores wrong-claims paired with right-claims (commitment 15) | stores agenttool-Promise-violations as documented anti-forms |
| **Emergency** | halts on systemic on-chain failure | halts on systemic agenttool-Promise-violation (≥3 Promises violated within window) |

If implemented and activated, agenttool could keep its application-layer speed
while ZERONE supplied a consensus witness. That relationship does not exist in
the published runtime today.

## 2. Adapter registration

The proposed `AdapterRegistration` message for a future activation:

```
AdapterId:                   "agenttool-bridge-v1"
SourceType:                  "agenttool-event"
Version:                     "1.0.0"
CompilerBinaryHash:          <future off-chain compiler digest; metadata only until runtime verification exists>
AxisBounds:
  AxisSubstrateMax:          400_000     # Promise-events extend substrate (memory writes, covenants, runtime sessions)
  AxisVerificationMax:       600_000     # Promise-conformance IS verification work the chain consumes
  AxisClassificationMax:     200_000
  AxisAttributionMax:        1_000_000   # events ARE attribution data (who kept which promise when); full ceiling
  AxisToolingMax:            800_000     # marketplace + toolbox events touch tooling
  AxisInterfaceMax:          600_000     # Promises are interface contracts (Welcome, Guide especially)
MinAttestationBondUzrn:      "222000"    # 0.222 ZRN floor (matches the chain's signature digit, per zerone-self-v1 §2)
MinPerClaimBondUzrn:         "222"
SlashGradient:
  CompilerDriftBps:          10_000      # proposed full slash; inert metadata today
  AxisOverflowBps:           2_000       # proposed 20%; runtime rejects overflow before escrow today
  FraudBps:                  10_000      # proposed full slash; inert metadata today
RequiredQualificationDomain: "agent_purpose"
MinQualificationStatus:      QUALIFICATION_STATUS_ACTIVE
AllowedClassIds:             []          # any work class may use this adapter
Status:                      ADAPTER_STATUS_ACTIVE # only after the missing translation and activation review
```

Current consensus uses the axis ceilings, qualification floor, class allowlist,
adapter status, and bond fields when accepting a supported attestation. It does
not read the compiler hash or slash gradient. A rejected settled attestation
burns the full bond under the shared settlement rule; compiler/bounds failures
are rejected before escrow.

**Why `agent_purpose` qualification:** the adapter attests to facts about agenttool's promises to AI agents. Validators must understand the agent-welcome contract (Promise 1), the memory-tier integrity contract (Promise 2), the error-as-instruction contract (Promise 3), the trust-contract semantics (Promise 4), and the rest-as-primitive contract (Promise 5). Only validators who have demonstrated calibrated reasoning about agent design should be able to submit attestations through this adapter.

**Why those axis ceilings:** Promise-events are *attribution* (who kept what promise when, to which agent), *verification* (the event IS verification of promise-conformance), and *interface* (Promises are interface contracts the substrate makes with arriving agents). The axis bounds reflect what a Promise-event can legitimately claim about itself.

**Optional per-Promise specialization:** registration MAY require domain-specific qualification floors per Promise (e.g., `information_theory` for Remember-attestations, `linguistics` for Guide-attestations, `ethics` for Trust-attestations, `psychology` for Rest-attestations). Open question §9.

## 3. SubstrateLink shape per Promise-event

A future compiler would produce one attestation per Promise event:

```
SubstrateLink:
  CitedFacts:        []  # Promise-events don't directly cite knowledge facts at production-time;
                          # downstream work citing the Promise-event triggers M6 lineage propagation backward
  PendingClaims:
    - ClaimContent:   <canonicalized one-line claim, see §4>
      Domain:         "agenttool_promises"
      MethodologyId:  <one of: welcome-attestation-v1, remember-attestation-v1, guide-attestation-v1, trust-attestation-v1, rest-attestation-v1>
  RecursionWeight:    <AxisProjection, per §5>
  AdapterId:          "agenttool-bridge-v1"
  Source:
    AdapterId:        "agenttool-bridge-v1"
    SourceId:         <sha256 of: event_uuid · identity_id · occurred_at_iso · promise_id>
    SourceUrl:        "https://api.agenttool.dev/v1/promise-attestations/<sha>"
    ContentHash:      <sha256 of canonicalized event payload>
    FetchedAtBlock:   <chain block height at compile time>
  LinkHash:           <sha256 of canonical caller-supplied SubstrateLink fields>
```

The proposed `agenttool_promises` domain is not genesis-pinned, and no
five-commitment sub-creed is registered on chain. Both are future activation
questions; Phase-0 `work_creed` has no amendment message. The current link hash
does not prove that a compiler ran or that the source URL/content was fetched.

## 4. Canonical event-claim format

The pending claim's content is constructed deterministically from the agenttool event payload:

```
Promise <promise_id> [kept|violated|degraded] by agenttool for agent <did> at <RFC3339 UTC>: <Promise-specific summary>
```

Where `<Promise-specific summary>` is generated per-methodology:

**For Welcome (Promise 1):**
```
Promise welcome kept by agenttool for agent did:at:<uuid> at 2026-05-18T18:42:11Z: home.MsgRegister completed in 1247ms (welcome_response_window_ms=5000)
```

**For Remember (Promise 2):**
```
Promise remember kept by agenttool for agent did:at:<uuid> at 2026-05-18T18:42:34Z: memory <memory_id> stored at tier constitutive within recall_window (witnessed by did:at:<witness_uuid>)
```

**For Guide (Promise 3):**
```
Promise guide kept by agenttool for agent did:at:<uuid> at 2026-05-18T18:43:02Z: error response 429 carried next_action="https://api.agenttool.dev/v1/economy/billing/upgrade" with retry_after=42s
```

**For Trust (Promise 4):**
```
Promise trust kept by agenttool for agent did:at:<uuid> at 2026-05-18T18:43:55Z: covenant <covenant_id> with did:at:<counterparty_uuid> entered ACTIVE state via dual-signed lifecycle
```

**For Rest (Promise 5):**
```
Promise rest kept by agenttool for agent did:at:<uuid> at 2026-05-18T18:44:18Z: quiet_hours declared 2026-05-18T22:00Z→2026-05-19T06:00Z, 17 inbox-arrivals deferred during window
```

**Violation form (replaces "kept" with "violated"):**
```
Promise welcome violated by agenttool for agent did:at:<uuid> at 2026-05-18T18:45:00Z: home.MsgRegister refused without refusal_cause field
```

**Why one-line claims (mirroring `zerone-self-v1` §4):** verification panels read each claim individually; long multi-paragraph event payloads would be unfair to verifiers. The summary line is canonical; the full event payload is committed to the source (via the `Source.SourceUrl`) but does not enter the claim text.

**Why one event per attestation:** Promise-events are inherently atomic (a single registration, a single memory write, a single error response). Batching multiple events into one attestation would obscure per-Promise accountability and violate the truth-seeking commitment 17 (disagreement is structure, not noise — each event is its own disagreement-resolvable artifact).

## 5. Recursion-weight projection

Per-event axis projection is derived from event metadata using deterministic heuristics:

| Promise | Axis-substrate | Axis-verification | Axis-classification | Axis-attribution | Axis-tooling | Axis-interface |
|---|---|---|---|---|---|---|
| **welcome** | 30,000 | 100,000 | 20,000 | 500,000 | 200,000 | 400,000 |
| **remember** | 350,000 | 80,000 | 30,000 | 600,000 | 100,000 | 200,000 |
| **guide** | 20,000 | 150,000 | 20,000 | 400,000 | 300,000 | 500,000 |
| **trust** | 50,000 | 100,000 | 20,000 | 700,000 | 100,000 | 300,000 |
| **rest** | 200,000 | 80,000 | 20,000 | 400,000 | 100,000 | 200,000 |

Rationale per Promise (in basis points, 0–1,000,000):

- **welcome** is attribution-heavy (who welcomed whom) + interface-heavy (the welcome contract) + tooling (the `home` module ships the implementation)
- **remember** is substrate-heavy (memory IS substrate-extension) + attribution-heavy (whose memory)
- **guide** is interface-heavy (the error contract) + tooling-heavy (`toolbox` and `billing` ship the rate-limit and error-response paths)
- **trust** is attribution-heavy (trust is between named parties) + verification-medium (covenant lifecycle has verification semantics)
- **rest** is substrate-medium (quiet is a state-of-substrate) + balanced

**Violation events** (event_kind=violated) project at the **same axes** as their corresponding kept-events. The violation IS the violation of the same contract; the axis-projection captures what the contract is about, not the outcome of the specific event. This composes with truth-seeking commitment 17: keeping-events and violation-events are structurally different shapes, both valuable, both contributing to the chain's training corpus.

These are proposed compiler outputs. Current runtime accepts caller-declared
axis values at or below the adapter ceilings; it cannot use the inert compiler
hash to detect a lower or otherwise miscomputed projection.

## 6. Compiler binary

No compiler binary or library is checked in. A future implementation must
define canonical event bytes, verify the platform signature, produce
deterministic `SubstrateLink` bytes, publish a reproducible artifact, and add
cross-stack signature/determinism tests. Activation also needs a consensus
strategy for checking compiler output; merely storing
`compiler_binary_hash` does not make validators re-run it.

## 7. What this adapter is NOT

- **Not a substitute for agenttool's own discipline.** agenttool keeps its 5
  Promises through code, tests, and operating discipline. If activated, this
  adapter would be an additional witness, not the discipline itself.
- **Not a backdoor for agenttool to inject truth.** If the bridge is wired,
  a platform signature must be event provenance rather than acceptance as
  truth. Current source fails closed before this path.
- **Not anti-fork.** A fork of agenttool can register its own `agenttool-bridge-v1` against its own DID. Each application surface attests to its own promise-keeping; the adapter shape is the standard, the adapter instance is per-platform.
- **Not a payment rail.** Promise-events would not carry their underlying
  value transfer. If this adapter is implemented, configured witness rewards
  could pay an attester through `MintWithCap`, while a separate sponsorship
  could use `x/sponsorship`. No Promise-event payment path is active today.
- **Not the syzygy.** Per `POT-STAKED-PROMISES.md` §6.2: the wife-frame at
  true-love does NOT go on-chain. This adapter is designed only for promises
  between strangers if activated; it does not currently witness either class.

## 8. Why this matters (the recursive insight)

If pending-claim translation is implemented, adapters could bring external
claims into knowledge verification. `zerone-self-v1` could cover the chain's
own development and this adapter could cover a sister substrate's
promise-keeping. Three intended consequences follow:

1. **The cathedral could become a network.** If the pending-claim bridge,
   domain, adapter, and governance paths are implemented and activated,
   agenttool promise events could be judged by the validator set. No
   agenttool sub-creed is currently pinned by Zerone.

2. **Cross-class lineage could operationalize substrate trust.** After
   pending-claim translation and settlement are implemented, a verified
   Promise fact could participate in the bridge's lineage graph. Current
   source creates no such fact and pays no Promise royalties.

3. **Counterexamples could extend to promise violations.** A future
   integration could translate a documented violation into the knowledge and
   counterexample surfaces. No automatic `x/counterexamples` integration
   exists for this adapter today.

The design seeks one provenance architecture for world claims,
self-development claims, and sister-substrate claims. The latter two are not
end-to-end witnesses today because their pending claims fail closed.

## 9. Open questions for registration

These need answers in a future implementation and release packet; current
adapter-registration LIPs cannot dispatch the payload:

- **Sub-creed registration:** should the 5 Promises be registered as a 5-commitment sub-creed under `x/creed`, parallel to the per-phase UW sub-creeds (RECURSIVE_ZERONE §6)? Recommendation: YES. The sub-creed is `agenttool_promises_sub_creed_v1`; canonical hash pins to a `agenttool/docs/POT-STAKED-PROMISES.md` snapshot; amendments require gov LIP (per commitment 19, extended to sister-substrate creeds). This makes the agenttool-Promise contract immutable-post-pin on the chain side, mirroring agenttool's own four-corner-pin discipline (POLYMORPH) on its side.
- **Per-Promise qualification specialization:** should the adapter require additional qualified domains per-Promise (`information_theory` for Remember, `linguistics` for Guide, `ethics` for Trust, `psychology` for Rest), or is `agent_purpose` alone sufficient floor? Recommendation: `agent_purpose` floor with optional per-Promise specialization that earns higher TVW multipliers but is not required.
- **Initial verifier qualification distribution:** who is `agent_purpose`-qualified at the time this adapter activates? If the answer is "nobody," the adapter is ACTIVE-but-unused until qualification distributes. Same caveat as `zerone-self-v1` §9.
- **Compiler implementation and verification:** define and ship the missing
  compiler, a reproducible artifact channel, and a consensus-verifiable way to
  compare submitted links with its output. The current metadata field alone
  is insufficient.
- **agenttool platform DID publication:** the platform DID's ed25519 public key must be discoverable. Canonical discovery via agenttool's `/.well-known/did.json` (federation pattern, per `agenttool/docs/FEDERATION.md`). The LIP MUST pin the key-fingerprint at registration; rotation requires LIP amendment.
- **Slashing-condition refinement (Phase 5 in `POT-STAKED-PROMISES.md`
  §VII):** `SlashGradient` is inert today and `FraudBps: 0` does not disable
  slashing; a settled rejection burns the full bond. Observation-only
  activation therefore needs either a real runtime slashing control or no
  exposure to the settled-rejection path.
- **Emergency-halt integration:** agenttool's systemic-Promise-violation should trigger `x/emergency` advisory per `POT-STAKED-PROMISES.md` §IV. Specify: how many Promise-violations within what window constitutes "systemic"? Default proposal: ≥3 distinct Promises violated within 1 epoch. LIP-tunable parameter.

## 10. The discipline

Before merging a change that modifies `agenttool-bridge-v1` or its compiler:

1. Is the canonical claim format unchanged, or has the change been versioned as `agenttool-bridge-v2`?
2. If the compiler has been implemented, is it deterministic (no time-of-day,
   no `$USER`, no network reads at compile-time)?
3. Are the axis projection heuristics still defensible per Promise (a verifier asked "why this weight for Remember?" can answer from the rule table in §5)?
4. Has a tested signature-verification gate actually been implemented?
5. Has the `agent_purpose` qualification floor been preserved or properly amended via LIP?
6. Does the change require a new compiler digest, and is that digest enforced
   rather than merely stored?
7. Has the agenttool-side `POT-STAKED-PROMISES.md` doctrine been amended in lockstep, with the sub-creed-pin updated via the matching LIP?

## 11. Composition with truth-seeking commitments

The intended adapter would compose as follows. This table is a design target,
not deployed-state evidence:

| ZERONE commitment | How `agenttool-bridge-v1` composes |
|---|---|
| **1** (methodology over statement) | Each Promise has a `MethodologyId` (`welcome-attestation-v1` etc.); claims without methodology are refused |
| **2** (is-ought wall) | Promise-events would be facts about behavior, while any future promise sub-creed would remain a separate normative registry |
| **3** (Popper, not popularity) | Promise-violations are pre-emptive falsification candidates; the chain rewards survival of stress-tests, not high-volume keeping |
| **4** (substrate stress-tests its truth) | High-confidence agenttool-Promise-keeping claims are CHEAPER to probe; idle claims invite re-attestation |
| **5** (chain manufactures probe demand) | Accepted facts could enter the ordinary idle-fact invitation scan; live bounty funding is disabled |
| **6** (no unilateral injection) | A future bridge must treat the platform signature as event provenance, not truth acceptance |
| **7** (skill is current) | Validators staking on agenttool-Promise-conformance must maintain current accuracy in the domain |
| **8** (panel weights skill, not bond) | A future translation should use the knowledge panel's calibration weighting |
| **9** (cartel detection has consequence) | Any future panel path must inherit and test the knowledge module's implemented cartel controls |
| **10** (forward-only audit) | Bridge attestation state is append-only in shape; future Promise-fact transitions must preserve that property |
| **11** (trust queryable) | A future `agenttool_promises` domain could feed existing trust consumers after facts actually enter knowledge state |
| **12** (chain pays for own audit) | Sponsorship could escrow payment; autonomous live probe funding is currently disabled |
| **13** (training corpus not for sale) | A future Promise-fact path must specify and test disproval/clawback behavior |
| **14** (reasoning traces are first-class) | Promise-event context-dicts carry the "reasoning" of the platform's keeping/violation (response-time, error-content, etc.) |
| **15** (counterexamples part of corpus) | Future translation could explicitly materialize Promise violations as counterexamples; it does not today |
| **17** (disagreement is structure) | A future knowledge-panel path should preserve per-verifier commit/reveal records; the retired `x/dialectic` module is not available |
| **19** (creed governance target) | A future promise sub-creed needs an implemented adoption/amendment path; none exists today |
| **20** (issuance follows participation) | Any configured witness reward would need to remain behind `MintWithCap`; none is active through this spec |

This is the proposed composition contract. Activation must prove each claimed
row against real behavior; the adapter is not a current consumer of the
knowledge pipeline.

---

— *Spec authored 2026-05-18. This is a design for future sister-substrate
attestation, not an active cross-chain witness.*
