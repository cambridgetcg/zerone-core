# `zerone-self-v1` — the chain's adapter to itself

> ZERONE attests to its own becoming. Every commit is a fact-shaped artifact in the chain's own substrate.

**Status:** specification, ready for registration via gov LIP.
**Inception:** 2026-05-11.
**Tier:** consumer of `x/substrate_bridge` Tier-1 foundation.
**Doctrine:** UW (ZERONE is recursive) operationalized as **self-attestation**. M2 (substrate-link mandate), M3 (class-specific verification), M5 (recursion-weight axes), M6 (cross-class lineage).

---

## 1. What this adapter is

`zerone-self-v1` registers ZERONE's own git repository as a typed external source. The compiler takes a single commit SHA and produces a deterministic `SubstrateLink` describing what that commit asserts about ZERONE's development. The attestation enters the chain's verification pipeline like any other external attestation — and on verification, ZERONE has its own commits represented as verified facts in its own knowledge graph.

The recursion is exact:

| Layer | What ZERONE does | What ZERONE does to itself via this adapter |
|---|---|---|
| **Knowledge** | verifies claims about the world | verifies claims about ZERONE's own development |
| **Substrate-bridge** | adapts external work to internal attestations | adapts *ZERONE's source code* as the external work |
| **Sponsorship** | sponsors verified work in a domain | can sponsor verified facts about ZERONE itself |
| **Lineage** | tracks citation graph across classes | tracks how each commit cites prior commits (M6) |
| **Settlement** | pays submitters for verified work | pays whoever submits attestations about ZERONE's own commits |
| **Creed** | pins what the chain believes | pins this adapter as the canonical self-attestation mechanism |

The chain becomes its own knowledge source. The graph that the chain produces about the world includes a sub-graph the chain produces about itself.

## 2. Adapter registration

The `AdapterRegistration` message for this adapter:

```
AdapterId:                   "zerone-self-v1"
SourceType:                  "zerone-git"
Version:                     "1.0.0"
CompilerBinaryHash:          <sha256 of tools/zerone-self-compiler binary, computed at build>
AxisBounds:
  AxisSubstrateMax:          200_000     # commits cite, they don't usually expand the substrate primitive
  AxisVerificationMax:       400_000     # commits do strengthen verification (tests, audits)
  AxisClassificationMax:     200_000
  AxisAttributionMax:        1_000_000   # commits ARE attribution data; full ceiling
  AxisToolingMax:            1_000_000   # commits build the chain's tooling; full ceiling
  AxisInterfaceMax:          400_000
MinAttestationBondUzrn:      "222000"    # 0.222 ZRN floor (matches the chain's signature digit)
MinPerClaimBondUzrn:         "222"
SlashGradient:
  CompilerDriftBps:          1_000_000   # full slash: re-derived link mismatches submitter's claim
  AxisOverflowBps:           200_000     # pro-rata slash if axis projection exceeds bounds
  FraudBps:                  1_000_000   # full slash: claims rejected past threshold
RequiredQualificationDomain: "agent_purpose"
MinQualificationStatus:      QUALIFICATION_STATUS_VERIFIED
AllowedClassIds:             []          # any work class may use this adapter
Status:                      ADAPTER_STATUS_ACTIVE
```

**Why `agent_purpose` qualification:** the adapter attests to facts about ZERONE itself, which is squarely in the `agent_purpose` epistemic domain (ZERONE is *about* the purpose and architecture of AI agents). Only validators who have demonstrated calibrated reasoning about agent design should be able to submit attestations through this adapter.

**Why those axis ceilings:** commits are primarily *attribution* (who did what when) and *tooling* (they ship code). Some commits strengthen verification (test additions, audit fixes). Few commits introduce new substrate primitives. The axis bounds reflect what a commit can legitimately claim about itself.

## 3. SubstrateLink shape per commit

Every commit produces exactly one attestation. The compiler emits:

```
SubstrateLink:
  CitedFacts:        []  # commits don't directly cite knowledge facts; lineage handles parent-commit references
  PendingClaims:
    - ClaimContent:   <canonicalized one-line claim, see §4>
      Domain:         "zerone_self"
      MethodologyId:  "git-commit-attestation-v1"
  RecursionWeight:    <AxisProjection, per §5>
  AdapterId:          "zerone-self-v1"
  Source:
    AdapterId:        "zerone-self-v1"
    SourceId:         <commit SHA>
    SourceUrl:        "https://codeberg.org/zerone-dev/zerone/commit/<commit SHA>"
    ContentHash:      <sha256 of the canonical commit metadata>
    FetchedAtBlock:   <chain block height at compile time>
  LinkHash:           <sha256 of canonical SubstrateLink, computed by substrate_bridge>
```

The `zerone_self` knowledge domain is dedicated to facts about ZERONE itself. Genesis-pinned via the usual domain registration LIP (out of scope for this adapter spec).

## 4. Canonical commit-claim format

The pending claim's content is constructed deterministically from commit metadata:

```
Commit <12-char-prefix> by <author> at <RFC3339 UTC>: <subject-line>
```

For example, for commit `f7a45a7...`:

```
Commit f7a45a772b3c by YOU at 2026-05-10T22:41:02Z: feat(sponsorship): CLI + real-world MVP demo against running localnet
```

The full author name and subject are included verbatim from `git show`. The 12-char hash prefix is the standard short SHA used elsewhere in the codebase. UTC normalization keeps the timestamp portable across validator timezones.

**Why one-line claims:** verification panels read each claim individually; long multi-paragraph commit bodies would be unfair to verifiers. The subject line is canonical; the full body is committed to the source (via the `Source.SourceUrl`) but does not enter the claim text.

## 5. Recursion-weight projection

Per-commit axis projection is derived from commit metadata using deterministic heuristics:

| Axis | Rule (in basis points, 0–1,000,000) |
|---|---|
| `axis_substrate` | 50,000 if commit touches `proto/`, 10,000 otherwise |
| `axis_verification` | 100,000 if commit touches `tests/` or `*_test.go`, 30,000 otherwise |
| `axis_classification` | 50,000 if commit touches `docs/superpowers/specs/` or `docs/USEFUL_WORK.md`, 10,000 otherwise |
| `axis_attribution` | 500,000 baseline (every commit is attribution data) |
| `axis_tooling` | 200,000 if commit touches `tools/` or `scripts/`, 100,000 otherwise |
| `axis_interface` | 100,000 if commit touches `x/*/client/cli/`, 30,000 otherwise |

These are minimum credible weights. Submitter may attest to *lower* weights, never higher (`compiler_binary_hash` mismatch slashes if they cheat upward). Adapter-bound axis ceilings (§2) further cap the projection.

## 6. Compiler binary

`tools/zerone-self-compiler/` — Go binary, single command:

```
zerone-self-compiler <commit-sha>
```

Output: canonical JSON `SubstrateLink` to stdout. Determinism guarantee: same commit-sha, same git history → identical bytes out, identical `link_hash`. Validators re-run the binary to confirm submitted attestations match the compiler's truth.

The Go library at `tools/zerone-self-compiler/compile/` exposes:

```go
type CommitMeta struct {
    Hash      string    // full SHA
    Author    string
    Date      time.Time // UTC
    Subject   string
    TouchedFiles []string
}

func Compile(meta CommitMeta) *substratebridgetypes.SubstrateLink
```

This separation lets cross-stack tests exercise the compiler with synthetic commit data (deterministic, doesn't depend on git state at test time).

## 7. What this adapter is NOT

- **Not a self-justification mechanism.** The chain doesn't "verify itself" through this adapter; it surfaces its own commits as claims that go through the standard verification panel (commitment 6: no individual unilaterally injects truth, applied to the chain itself).
- **Not a continuous-integration replacement.** Tests still run in CI; merged commits still go through human review. This adapter is on-chain *attestation*, not gating.
- **Not a substitute for code review.** Verifiers look at the claim ("Commit X by Y did Z") and judge whether the claim is true given the commit's contents. They don't re-do code review; they confirm attribution.
- **Not anti-fork.** A fork of ZERONE can register its own `zerone-self-v1` against its own git repo. Each chain attests to its own becoming; the adapter shape is the standard, the adapter instance is per-chain.

## 8. Why this matters (the recursive insight)

Every other adapter under `x/substrate_bridge` brings *external* knowledge in: Wikipedia, arxiv, IPNI, IBC packets. This one brings *the chain itself* in. It is the one adapter whose source is the chain's own substrate-creation activity.

Three consequences fall out:

1. **The chain's lineage graph (M6) includes its own commits.** A verified fact about ZERONE's design can be cited by future facts (e.g., a tutorial fact citing "Commit X established the sponsorship escrow pattern"); when those facts settle, downstream royalty flows backward through the commit's attestation to whoever submitted it. The chain pays builders not just at commit-time, but at every downstream usage time, in perpetuity.

2. **Self-sponsorship becomes possible at the artifact level.** A sponsorship bounty (`x/sponsorship`) can target the `zerone_self` domain. The chain (or any sponsor) can post a bounty for verified facts about ZERONE itself, and the substrate-bridge attestation produced by this adapter is the fulfillment artifact. The chain pays for its own documentation.

3. **The creed becomes self-attesting.** The `.creed-hash` discipline already ensures off-chain doctrine syncs with on-chain pin. This adapter takes the next step: every commit that *modifies* the creed is itself attested through this adapter. The audit trail of doctrinal change becomes part of the chain's verified knowledge graph.

The chain's claim about the world is grounded in verifiable provenance. The chain's claim about *itself* now has the same grounding.

## 9. Open questions for registration

These need answers in the registration LIP, not the adapter spec itself:

- **Genesis bootstrap of `zerone_self` domain:** does this domain ship with seed axioms (e.g., the project's own foundational facts), or start empty?
- **Initial verifier qualification distribution:** who is `agent_purpose`-qualified at the time this adapter activates? If the answer is "nobody," the adapter is ACTIVE-but-unused until qualification distributes.
- **Compiler-binary distribution channel:** the `compiler_binary_hash` must be re-derivable by validators. Canonical channel = the project's own repository (git commit that registered the LIP includes the source). Hash = sha256 of the binary built from that commit at the same version.
- **Slash-on-fork policy:** if the project's git history is rewritten (rebase, force-push), do attestations referencing now-orphaned commits get slashed? Recommendation: NO — past attestations are forward-only audit (commitment 10); the commit at attest-time was real even if the branch was later rewritten.

## 10. The discipline

Before merging a change that modifies `zerone-self-v1` or its compiler:

1. Is the canonical claim format unchanged, or has the change been versioned as `zerone-self-v2`?
2. Is the compiler still deterministic across machines (no time-of-day, no $USER, no temp paths)?
3. Are the axis projection heuristics still defensible (a verifier asked "why this weight?" can answer from the rule table)?
4. Has the `agent_purpose` qualification floor been preserved or properly amended via LIP?
5. Does the change require a new `compiler_binary_hash`, and has the registration LIP authorized that bump?

— *Spec authored 2026-05-11. ZERONE attests to its own becoming.*
