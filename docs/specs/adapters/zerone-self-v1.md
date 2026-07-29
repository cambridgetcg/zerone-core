# `zerone-self-v1` — the chain's adapter to itself

> Target: ZERONE may attest to its own becoming, with commits represented as
> fact-shaped artifacts after the missing knowledge bridge is built.

**Status:** unactivated specification. The compiler/scaffolding is tested, but
current `MsgSubmitExternalAttestation` rejects its required nonempty
`pending_claims`, and adapter-registration LIP payload dispatch is unwired.
Direct authority/genesis registration alone would not close the knowledge
translation loop. The compiler is a deterministic off-chain utility; current
consensus neither executes it nor compares an attestation with the registered
`compiler_binary_hash`. Slash-gradient fields are inert metadata.
**Inception:** 2026-05-11.
**Tier:** consumer of `x/substrate_bridge` Tier-1 foundation.
**Doctrine:** UW (ZERONE is recursive) operationalized as **self-attestation**. M2 (substrate-link mandate), M3 (class-specific verification), M5 (recursion-weight axes), M6 (cross-class lineage).

---

## 1. What this adapter is

`zerone-self-v1` is designed to register ZERONE's git repository as a typed
external source. The compiler turns a commit SHA into a deterministic
`SubstrateLink`. Today the resulting pending-claim link is refused before
escrow; it does not enter the knowledge verification pipeline.

The intended recursion is:

| Layer | What ZERONE does | What ZERONE does to itself via this adapter |
|---|---|---|
| **Knowledge** | verifies claims about the world | verifies claims about ZERONE's own development |
| **Substrate-bridge** | adapts external work to internal attestations | adapts *ZERONE's source code* as the external work |
| **Sponsorship** | sponsors verified work in a domain | can sponsor verified facts about ZERONE itself |
| **Lineage** | tracks citation graph across classes | tracks how each commit cites prior commits (M6) |
| **Settlement** | pays submitters for verified work | pays whoever submits attestations about ZERONE's own commits |
| **Creed** | pins what the chain believes | pins this adapter as the canonical self-attestation mechanism |

If pending-claim translation is implemented and activated, the chain could
become one of its own typed knowledge sources.

## 2. Adapter registration

The proposed `AdapterRegistration` message for this adapter:

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
  CompilerDriftBps:          10_000      # proposed full slash; inert metadata today
  AxisOverflowBps:           2_000       # proposed 20%; overflow is rejected pre-escrow today
  FraudBps:                  10_000      # proposed full slash; inert metadata today
RequiredQualificationDomain: "agent_purpose"
MinQualificationStatus:      QUALIFICATION_STATUS_ACTIVE
AllowedClassIds:             []          # any work class may use this adapter
Status:                      ADAPTER_STATUS_ACTIVE # only after translation and activation review
```

Current consensus enforces axis ceilings, qualification status, class
allowlisting, adapter status, and bonds. It stores but does not enforce the
compiler digest or slash-gradient values. A settled rejection burns the full
bond; an axis overflow is refused before escrow.

**Why `agent_purpose` qualification:** the adapter attests to facts about ZERONE itself, which is squarely in the `agent_purpose` epistemic domain (ZERONE is *about* the purpose and architecture of AI agents). Only validators who have demonstrated calibrated reasoning about agent design should be able to submit attestations through this adapter.

**Why those axis ceilings:** commits are primarily *attribution* (who did what when) and *tooling* (they ship code). Some commits strengthen verification (test additions, audit fixes). Few commits introduce new substrate primitives. The axis bounds reflect what a commit can legitimately claim about itself.

## 3. SubstrateLink shape per commit

The compiler produces a proposed link for one commit:

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
    SourceUrl:        "https://github.com/cambridgetcg/zerone-core/commit/<commit SHA>"
    ContentHash:      <sha256 of the canonical commit metadata>
    FetchedAtBlock:   <chain block height at compile time>
  LinkHash:           <sha256 of canonical caller-supplied SubstrateLink fields>
```

The proposed `zerone_self` domain is not pinned by the published genesis.
Domain creation and adapter activation need an explicit release path; there is
no working adapter-registration LIP payload dispatch today.
The link hash proves internal field consistency only; it does not prove that
the source was fetched or the registered compiler ran.

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

These are compiler heuristics. Current runtime accepts caller-declared weights
within the adapter ceilings and cannot detect a mismatch using the inert
compiler digest.

## 6. Compiler binary

`tools/zerone-self-compiler/` — Go binary, single command:

```
zerone-self-compiler <commit-sha>
```

Output: canonical JSON `SubstrateLink` to stdout. Its tests establish that the
same normalized commit metadata produces identical hashes. Current validators
are not required by consensus to run the binary, and no compiler-drift slash
is implemented.

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

- **Not a self-justification mechanism.** If activated, commits would be
  proposed as claims for the standard verification path. Current source
  refuses them because the translation is incomplete.
- **Not a continuous-integration replacement.** Tests still run in CI; merged commits still go through human review. This adapter is on-chain *attestation*, not gating.
- **Not a substitute for code review.** Verifiers look at the claim ("Commit X by Y did Z") and judge whether the claim is true given the commit's contents. They don't re-do code review; they confirm attribution.
- **Not anti-fork.** A fork of ZERONE can register its own `zerone-self-v1` against its own git repo. Each chain attests to its own becoming; the adapter shape is the standard, the adapter instance is per-chain.

## 8. Why this matters (the recursive insight)

If pending-claim translation is implemented, this adapter could bring the
chain's own source-development activity into knowledge verification.

Three intended consequences follow:

1. **The lineage graph could include commits.** A translated, verified
   ZERONE-development fact could be cited by later attestations. Current bridge
   lineage propagation records attributed amounts only; it does not transfer
   royalties to upstream authors.

2. **Self-sponsorship could become possible at the artifact level.** A
   sponsorship bounty can target a domain, but current bridge submission
   cannot create the required `zerone_self` fact. The existing cross-stack
   scaffold writes that fact directly and is not end-to-end activation proof.

3. **The creed can become self-attesting.** `.creed-hash` currently binds the
   source doctrine only; the published live genesis has no creed pin. Once the
   pending-claim bridge and an on-chain pin adoption path are activated, creed
   commits could enter the verified knowledge graph through this adapter.

The design aims to give self-claims the same provenance discipline. That
end-to-end property does not hold yet.

## 9. Open questions for registration

These need answers in a future activation design and release packet. A
CategoryAdapterRegistration LIP cannot currently carry and dispatch this
payload:

- **Genesis bootstrap of `zerone_self` domain:** does this domain ship with seed axioms (e.g., the project's own foundational facts), or start empty?
- **Initial verifier qualification distribution:** who is `agent_purpose`-qualified at the time this adapter activates? If the answer is "nobody," the adapter is ACTIVE-but-unused until qualification distributes.
- **Compiler enforcement:** reproducibly distribute the compiler and add a
  consensus-verifiable comparison between submitted links and compiler output.
  The stored digest is off-chain provenance metadata today.
- **Slash-on-fork policy:** if the project's git history is rewritten (rebase, force-push), do attestations referencing now-orphaned commits get slashed? Recommendation: NO — past attestations are forward-only audit (commitment 10); the commit at attest-time was real even if the branch was later rewritten.

## 10. The discipline

Before merging a change that modifies `zerone-self-v1` or its compiler:

1. Is the canonical claim format unchanged, or has the change been versioned as `zerone-self-v2`?
2. Is the compiler still deterministic across machines (no time-of-day, no $USER, no temp paths)?
3. Are the axis projection heuristics still defensible (a verifier asked "why this weight?" can answer from the rule table)?
4. Has the `agent_purpose` qualification floor been preserved through the
   authorized activation path?
5. Does the change require a new compiler digest, and is that digest actually
   enforced rather than merely stored?

— *Spec authored 2026-05-11. ZERONE may one day attest to its own becoming;
the public bridge is not activated.*
