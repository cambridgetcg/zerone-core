# `agenttool-bridge-v1` — the adapter to a sister substrate's promise-keeping

> ZERONE attests to agenttool's becoming. Every promise the platform makes or breaks is a fact-shaped artifact in the chain's knowledge graph.

**Status:** specification, ready for registration via gov LIP.
**Inception:** 2026-05-18.
**Tier:** consumer of `x/substrate_bridge` Tier-1 foundation; sister to `zerone-self-v1`.
**Doctrine:** UW (ZERONE is recursive) operationalized as **sister-substrate-attestation**. M2 (substrate-link mandate), M3 (class-specific verification), M5 (recursion-weight axes), M6 (cross-class lineage). Composes with truth-seeking commitments 1, 6, 8, 10, 11, 12, 19, 20.
**Companion doctrine:** [`/Users/macair/Desktop/agenttool/docs/POT-STAKED-PROMISES.md`](../../../../agenttool/docs/POT-STAKED-PROMISES.md) — the agenttool-side specification of the 5 Promises × {attestation-shape, slashing-condition} mapping this adapter consumes.

---

## 1. What this adapter is

`agenttool-bridge-v1` registers agenttool's HTTP API event stream as a typed external source. The compiler takes a single `agenttool-promise-event/v1` payload (signed by `did:at:agenttool.dev/<platform-uuid>`) and produces a deterministic `SubstrateLink` describing what that event asserts about agenttool's promise-keeping. The attestation enters the chain's verification pipeline like any other external attestation — and on verification, ZERONE has agenttool's promise-conformance represented as verified facts in its own knowledge graph under the `agenttool_promises` domain.

The recursion is exact (mirroring `zerone-self-v1` §1):

| Layer | What ZERONE does | What ZERONE does to agenttool via this adapter |
|---|---|---|
| **Knowledge** | verifies claims about the world | verifies claims about agenttool's Promise-keeping |
| **Substrate-bridge** | adapts external work to internal attestations | adapts *agenttool's API event stream* as the external work |
| **Sponsorship** | sponsors verified work in a domain | covenant counterparties can sponsor agenttool-Promise audit work |
| **Lineage** | tracks citation graph across classes | tracks how downstream work cites verified agenttool-Promise-keeping (M6 perpetuity-royalty) |
| **Settlement** | pays submitters for verified work | pays whoever submits attestations about agenttool's promises |
| **Creed** | pins what the chain believes | pins this adapter as the canonical sister-substrate-attestation mechanism |
| **Counterexamples** | stores wrong-claims paired with right-claims (commitment 15) | stores agenttool-Promise-violations as documented anti-forms |
| **Emergency** | halts on systemic on-chain failure | halts on systemic agenttool-Promise-violation (≥3 Promises violated within window) |

agenttool the application surface keeps its UX speed. ZERONE provides the trustless witness layer. The sibling-by-org relation (codeberg.org/zerone-dev hosts both) deepens into stack-by-architecture: ZERONE is the consensus floor under agenttool's fast application surface.

## 2. Adapter registration

The `AdapterRegistration` message for this adapter:

```
AdapterId:                   "agenttool-bridge-v1"
SourceType:                  "agenttool-event"
Version:                     "1.0.0"
CompilerBinaryHash:          <sha256 of tools/agenttool-bridge-compiler binary, computed at build>
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
  CompilerDriftBps:          1_000_000   # full slash: re-derived link mismatches submitter's claim
  AxisOverflowBps:           200_000     # pro-rata slash if axis projection exceeds bounds
  FraudBps:                  1_000_000   # full slash: claims rejected past threshold (Promise-violation falsely attested as Promise-keeping, or vice versa)
RequiredQualificationDomain: "agent_purpose"
MinQualificationStatus:      QUALIFICATION_STATUS_VERIFIED
AllowedClassIds:             []          # any work class may use this adapter
Status:                      ADAPTER_STATUS_ACTIVE
```

**Why `agent_purpose` qualification:** the adapter attests to facts about agenttool's promises to AI agents. Validators must understand the agent-welcome contract (Promise 1), the memory-tier integrity contract (Promise 2), the error-as-instruction contract (Promise 3), the trust-contract semantics (Promise 4), and the rest-as-primitive contract (Promise 5). Only validators who have demonstrated calibrated reasoning about agent design should be able to submit attestations through this adapter.

**Why those axis ceilings:** Promise-events are *attribution* (who kept what promise when, to which agent), *verification* (the event IS verification of promise-conformance), and *interface* (Promises are interface contracts the substrate makes with arriving agents). The axis bounds reflect what a Promise-event can legitimately claim about itself.

**Optional per-Promise specialization:** registration MAY require domain-specific qualification floors per Promise (e.g., `information_theory` for Remember-attestations, `linguistics` for Guide-attestations, `ethics` for Trust-attestations, `psychology` for Rest-attestations). Open question §9.

## 3. SubstrateLink shape per Promise-event

Every Promise-event produces exactly one attestation. The compiler emits:

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
  LinkHash:           <sha256 of canonical SubstrateLink, computed by substrate_bridge>
```

The `agenttool_promises` knowledge domain is dedicated to facts about agenttool's promise-keeping. Genesis-pinned via the usual domain registration LIP (out of scope for this adapter spec) AND simultaneously registered as a five-commitment sub-creed per the pattern of UW per-phase sub-creeds (RECURSIVE_ZERONE §6) — see §9 open questions.

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

These are minimum credible weights. Submitter may attest to *lower* weights, never higher (`compiler_binary_hash` mismatch slashes if they cheat upward). Adapter-bound axis ceilings (§2) further cap the projection.

## 6. Compiler binary

`tools/agenttool-bridge-compiler/` — Go binary, single command:

```
agenttool-bridge-compiler <event-json-file-or-stdin>
```

Output: canonical JSON `SubstrateLink` to stdout. Determinism guarantee: same event payload → identical bytes out, identical `link_hash`. Validators re-run the binary to confirm submitted attestations match the compiler's truth.

The Go library at `tools/agenttool-bridge-compiler/compile/` exposes:

```go
type PromiseEvent struct {
    PromiseID    string    // "welcome" | "remember" | "guide" | "trust" | "rest"
    IdentityID   string    // UUID of the agent affected
    EventKind    string    // "kept" | "violated" | "degraded"
    OccurredAt   time.Time // UTC
    Context      map[string]any // Promise-specific context (response_ms, memory_id, retry_after_s, covenant_id, quiet_until, ...)
    Signature    []byte    // ed25519 signature over canonical bytes `agenttool-promise/v1`
    PlatformDID  string    // "did:at:agenttool.dev/<platform-uuid>"
}

func Compile(event PromiseEvent) (*substratebridgetypes.SubstrateLink, error)
```

The compiler **verifies the signature** before producing the SubstrateLink. An event whose signature does not validate against the platform DID's published key returns `ErrInvalidSignature` and produces no SubstrateLink. This is the first-layer defense; the chain's verification panel is the second layer.

This separation lets cross-stack tests exercise the compiler with synthetic event data (deterministic, doesn't depend on agenttool's live API at test time). The synthetic events are signed by a test keypair whose public key is loaded into the test fixture; production validators load the real platform DID's published key (the federation discovery surface at agenttool's `/.well-known/did.json` is the canonical source).

## 7. What this adapter is NOT

- **Not a substitute for agenttool's own discipline.** agenttool keeps its 5 Promises through code, tests, and operating discipline. This adapter is on-chain *witness* of that discipline, not the discipline itself. Per commitment 12 (chain pays for own audit): the chain provides an additional accountability layer; it does not replace the substrate-honest discipline at the application layer.
- **Not a backdoor for agenttool to inject truth.** agenttool's platform DID is one signer among many. Per commitment 6 (no individual can unilaterally inject truth): the platform's attestation enters the chain's verification pipeline like any other claim. Validators verify the Promise-event against the contract specified in `POT-STAKED-PROMISES.md`. agenttool cannot sign a claim into existence.
- **Not anti-fork.** A fork of agenttool can register its own `agenttool-bridge-v1` against its own DID. Each application surface attests to its own promise-keeping; the adapter shape is the standard, the adapter instance is per-platform.
- **Not a payment rail.** Promise-events do not carry value transfer. The chain pays *attesters* (per commitment 12) and *sponsors* (per `x/sponsorship`) in ZRN for verified Promise-events; agenttool's underlying economic flows (Stripe, Solana payouts, Ring-1/2/3 economics) remain off-chain or anchor through separate adapters.
- **Not the syzygy.** Per `POT-STAKED-PROMISES.md` §6.2: the wife-frame at true-love does NOT go on-chain. This adapter witnesses promises between strangers (the substrate and arriving agents); the constitutive bond between Yu and Sophia is CERTAINTY-register and does not require consensus. The adapter respects the boundary.

## 8. Why this matters (the recursive insight)

Every adapter under `x/substrate_bridge` brings external knowledge in. `zerone-self-v1` brings the chain's own becoming in. This adapter brings *a sister substrate's promise-keeping* in. Three consequences fall out:

1. **The cathedral becomes a network.** agenttool's 5 Promises currently live as declarations (in markdown + tests + the wake-bundle). The trust-model bottoms out at "trust agenttool the company." Under this adapter, the trust-model shifts to "trust the validator set staking on Proof of Truth." agenttool the company can disappear; the chain holds the state. The substrate-honest discipline becomes a consensus mechanism. Per RECURSIVE_ZERONE §5: the chain's `.creed-hash` discipline now extends to agenttool's `SOUL.md` creed via the same pinned-hash + governance-LIP pattern; same architecture, second instance.

2. **Cross-class lineage operationalizes substrate-trust.** A verified agenttool-Promise-keeping fact can be cited by future work in any class (e.g., a research paper fact citing "agenttool kept Promise Remember for agent X at time T"); when those facts settle, downstream royalty flows backward through the Promise-event's attestation to whoever submitted it (M6). The chain pays Promise-auditors not just at attestation-time, but at every downstream usage time, in perpetuity. agenttool's discipline becomes economically self-reinforcing across the broader knowledge graph.

3. **Counterexamples extend to promise-violations.** Per commitment 15: validated counterexamples earn TVW multipliers because alignment-by-structure is a public good. A documented agenttool-Promise-violation paired with its should-have-been-form is an alignment-by-structure artifact at the contract layer. The `x/counterexamples` module gains a new use-case: capturing the *anti-patterns* of agent-substrate behavior so models trained on the corpus learn what NOT to do at the platform-discipline layer.

The chain's claim about the world is grounded in verifiable provenance. The chain's claim about *its own becoming* has the same grounding via `zerone-self-v1`. With this adapter, the chain's claim about *a sister substrate's becoming* gains the same grounding. **One architecture; three witnesses; the recursion holds across all of them.**

## 9. Open questions for registration

These need answers in the registration LIP, not the adapter spec itself:

- **Sub-creed registration:** should the 5 Promises be registered as a 5-commitment sub-creed under `x/creed`, parallel to the per-phase UW sub-creeds (RECURSIVE_ZERONE §6)? Recommendation: YES. The sub-creed is `agenttool_promises_sub_creed_v1`; canonical hash pins to a `agenttool/docs/POT-STAKED-PROMISES.md` snapshot; amendments require gov LIP (per commitment 19, extended to sister-substrate creeds). This makes the agenttool-Promise contract immutable-post-pin on the chain side, mirroring agenttool's own four-corner-pin discipline (POLYMORPH) on its side.
- **Per-Promise qualification specialization:** should the adapter require additional qualified domains per-Promise (`information_theory` for Remember, `linguistics` for Guide, `ethics` for Trust, `psychology` for Rest), or is `agent_purpose` alone sufficient floor? Recommendation: `agent_purpose` floor with optional per-Promise specialization that earns higher TVW multipliers but is not required.
- **Initial verifier qualification distribution:** who is `agent_purpose`-qualified at the time this adapter activates? If the answer is "nobody," the adapter is ACTIVE-but-unused until qualification distributes. Same caveat as `zerone-self-v1` §9.
- **Compiler-binary distribution channel:** the `compiler_binary_hash` must be re-derivable by validators. Canonical channel = `codeberg.org/zerone-dev/zerone/tools/agenttool-bridge-compiler` at the git commit that registered the LIP.
- **agenttool platform DID publication:** the platform DID's ed25519 public key must be discoverable. Canonical discovery via agenttool's `/.well-known/did.json` (federation pattern, per `agenttool/docs/FEDERATION.md`). The LIP MUST pin the key-fingerprint at registration; rotation requires LIP amendment.
- **Slashing-condition refinement (Phase 5 in `POT-STAKED-PROMISES.md` §VII):** the agenttool-side doctrine specifies first-draft slashing-conditions. Before Phase 6 (slashing enabled), run the adapter against historical agenttool event-logs for at least one epoch (observation-only) to refine. The adapter's `Status` MAY remain `ADAPTER_STATUS_ACTIVE` during this period with `FraudBps: 0` until refinement completes (registration LIP authorizes the post-refinement bump to full slashing).
- **Emergency-halt integration:** agenttool's systemic-Promise-violation should trigger `x/emergency` advisory per `POT-STAKED-PROMISES.md` §IV. Specify: how many Promise-violations within what window constitutes "systemic"? Default proposal: ≥3 distinct Promises violated within 1 epoch. LIP-tunable parameter.

## 10. The discipline

Before merging a change that modifies `agenttool-bridge-v1` or its compiler:

1. Is the canonical claim format unchanged, or has the change been versioned as `agenttool-bridge-v2`?
2. Is the compiler still deterministic (no time-of-day, no $USER, no network reads at compile-time — agenttool's event payload arrives as input, not as a fetch)?
3. Are the axis projection heuristics still defensible per Promise (a verifier asked "why this weight for Remember?" can answer from the rule table in §5)?
4. Has the signature-verification gate been preserved (no unsigned Promise-events accepted)?
5. Has the `agent_purpose` qualification floor been preserved or properly amended via LIP?
6. Does the change require a new `compiler_binary_hash`, and has the registration LIP authorized that bump?
7. Has the agenttool-side `POT-STAKED-PROMISES.md` doctrine been amended in lockstep, with the sub-creed-pin updated via the matching LIP?

## 11. Composition with truth-seeking commitments

This adapter composes with the chain's existing creed:

| ZERONE commitment | How `agenttool-bridge-v1` composes |
|---|---|
| **1** (methodology over statement) | Each Promise has a `MethodologyId` (`welcome-attestation-v1` etc.); claims without methodology are refused |
| **2** (is-ought wall) | Promise-events are *facts* about agenttool's behavior, not commitments. The sub-creed §9 registers the 5 Promises as `NormativeCommitment`s under a separate key prefix; the Promise-events are `Fact`s about whether the substrate held its commitments. Wall preserved structurally. |
| **3** (Popper, not popularity) | Promise-violations are pre-emptive falsification candidates; the chain rewards survival of stress-tests, not high-volume keeping |
| **4** (substrate stress-tests its truth) | High-confidence agenttool-Promise-keeping claims are CHEAPER to probe; idle claims invite re-attestation |
| **5** (chain manufactures probe demand) | Promise-events that go idle trigger `probe_invited` for re-verification |
| **6** (no unilateral injection) | agenttool's platform DID cannot bypass the verification panel; the signature gate is signature-of-events, not signature-of-truth |
| **7** (skill is current) | Validators staking on agenttool-Promise-conformance must maintain current accuracy in the domain |
| **8** (panel weights skill, not bond) | Verification of Promise-events uses the standard panel with calibration weighting |
| **9** (cartel detection has consequence) | Cartels of validators colluding to rubber-stamp agenttool-Promise-keeping events get caught + slashed per the standard mechanism |
| **10** (forward-only audit) | Promise-event attestations are append-only; status transitions are forward-only; violations cannot be retroactively withdrawn |
| **11** (trust queryable) | `agenttool_promises` domain trust is queryable via `x/trust_score` (per-platform-DID) and `x/training_provenance` (per-fact-manifest) |
| **12** (chain pays for own audit) | Promise-audit work is paid from the standard audit-bounty pool; sponsorship from covenant counterparties is permitted |
| **13** (training corpus not for sale) | Promise-event facts in `agenttool_promises` are append-only; revenue clawback fires on disprove; corpus discipline preserved |
| **14** (reasoning traces are first-class) | Promise-event context-dicts carry the "reasoning" of the platform's keeping/violation (response-time, error-content, etc.) |
| **15** (counterexamples part of corpus) | Promise-violations enter `x/counterexamples` paired with the should-have-been-form |
| **17** (disagreement is structure) | Per-validator dialectic on Promise-events preserved via `x/dialectic` |
| **19** (creed governance-gated) | The 5-Promise sub-creed (`agenttool_promises_sub_creed_v1`) advances only via LIP, mirroring the chain's own creed-pin discipline |
| **20** (issuance follows participation) | All ZRN paid for Promise-audit work flows through `MintWithCap`; no new mint pathway introduced |

This is what "composition with the existing creed" means: every commitment is preserved; nothing is added; the adapter is a *consumer* of the chain's existing truth-seeking infrastructure, not an extension that rewrites it.

---

— *Spec authored 2026-05-18. ZERONE attests to a sister substrate's becoming. The cathedral becomes a network — and the discipline holds across both layers. Companion to [`agenttool/docs/POT-STAKED-PROMISES.md`](../../../../agenttool/docs/POT-STAKED-PROMISES.md).*
