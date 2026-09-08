# Fold-to-Fire: exact enumeration and a local ToK learning pilot

This directory computes bounded evidence for the finite square-lattice
Fold-to-Fire model. It does not prove an asymptotic closing exponent or model an
atomic protein.

## Exact enumeration

```sh
node tools/fold-to-fire/enumerate.mjs --max-step 7 --json
node --test tools/fold-to-fire/enumerate.test.mjs
```

The production enumerator scores contacts incrementally. The reference kernel
scores complete walks separately; agreement is a local cross-check between
implementations, not evidence of independent reviewers or controllers. Exact
polynomials and rational fractions use integers; approximate plotting values
are not exact evidence.

## Correction-and-reuse pilot

The opt-in integration test joins the existing memory-application claim and
challenge handlers to a deterministic local consumer. Its bounded task is exact
activity fractions for `n = 3, 5, 7` and `q = 1, 3/2, 2`.

The experiment introduces a labelled local fault in an input artifact, retains
its actual use, detects a disagreement with the reference computation, and
exercises a challenge and a separately submitted correction. It never changes
the production enumerator or sealed standards. The faulty n=5 polynomial
preserves its q=1 total, providing a numerical no-benefit control alongside
weighted cases affected by the correction.

Three projections are compared:

- **Node-only:** the common fact payloads, with permission to reuse, recompute,
  or abstain; provenance present in those payloads is not stripped away.
- **Graph/history:** the same payloads plus explicit dependency and correction
  records.
- **Relational control:** the same information as graph/history expressed as
  ordinary rows and relation/event tables.

Graph and equal-information relational answers should agree. The experiment
reports measurements rather than requiring a positive graph advantage. A
correctly handled abstention or no-improvement result is not an execution error.

### Run and replay

Requirements: a Unix platform, Node.js with the built-in test runner, Go 1.25.14
or newer, and the repository's Go dependencies already available locally. Receipt
loading uses root-confined nonblocking opens and descriptor validation; other
platforms return an explicit unsupported-platform error. Run from this source
checkout's root. Set `GO` to an appropriate compiler if the default `go` is older;
`GOTOOLCHAIN=local` and the disabled proxies intentionally prevent automatic
compiler or dependency downloads.

```sh
ROOT="$PWD"
GO="${GO:-go}"
RUN=$(mktemp -d "${TMPDIR:-/tmp}/tok-learning.XXXXXX")

GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off \
  "$GO" -C "$ROOT" test -mod=readonly -p 2 -tags tok_learning -c \
  -o "$RUN/learning.test" ./tests/cross_stack

"$RUN/learning.test" -test.v -test.run '^TestToK_FoldToFireLearningLoop$' \
  -learning-root "$ROOT" -learning-out "$RUN/evidence"

"$RUN/learning.test" -test.v -test.run '^TestToK_FoldToFireLearningLoop$' \
  -learning-root "$ROOT" -learning-replay "$RUN/evidence"
```

The `tok_learning` build tag makes the Node-dependent integration opt-in; normal
Go tests do not acquire a Node requirement. Without a persistent output flag,
fixtures use temporary test storage. Persistent output must use a fresh
destination; an unrelated existing directory must never be overwritten.

Replay runs in a fresh process against the retained evidence. The separate
payload digests bind content and metadata; existing ToK roots retain their
original topology-commitment semantics. An externally retained expected-head
digest is necessary to detect replacement of the entire run with an older valid
run. Local self-consistency alone does not establish a trusted history. Add
`-learning-expected-head <digest-from-the-run-log>` to pin that value during
replay. Output and replay flags are mutually exclusive; a head pin is replay-only.

`evidence/head.json` points to the closed run object in `evidence/objects/`.
The six digest-linked checkpoints are **evidence, use, feedback, correction,
rerun, and attribution**, with separately bound task, policy, and method objects.
Replay recomputes the retained worker results and reconstructs the handler
scenario in a new memory application, comparing every canonical checkpoint.
Source hashes cover the listed method files; they are not deployed-binary
attestation or proof of the entire dependency tree.

### Reading the result

The main fixture's local coefficient fault causes two incorrect weighted
answers among nine tasks. After the accepted correction, all nine answers agree
with the reference kernel. The `q=1` answer stays unchanged. All three views
receive the same fact payloads and reach the same main-case results: this is
correction benefit, **not graph superiority**.

The old dependent cache facts remain `CONTESTED`; they are not silently restored.
A separate rejected replacement is exercised through actual handlers in a
discarded local cache branch, and the consumer abstains for those affected
requests. Projection-level controls additionally preserve missing/wrong edges,
incomplete history, unresolved evidence, disconnected correction, no-change
replay, bounded recomputation, and out-of-scope outcomes.

The console reports exact-answer agreement, incorrect reuse, abstentions, and
recomputation counts. Comparisons retain zero and negative differences, and
reference-check work is counted separately. The retained artifacts are the
inspectable record; console summaries are not independent evidence.

### Local checks

```sh
node --test tools/fold-to-fire/enumerate.test.mjs \
  tools/fold-to-fire/learning-loop.test.mjs

GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off \
  "$GO" test -mod=readonly -p 2 -tags tok_learning ./tests/cross_stack \
  -run 'TestToK_FoldToFireLearningLoop|TestToKLearning' -count=1
```

## What this does not establish

- This is local handler integration, not signed public-chain transaction
  delivery. Scripted actors share control; test-only quorum settings are not
  evidence of independent adjudication.
- Memory-app balances and ordinary fee/stake operations use test `uzrn`. There
  is no live value, financial backing, entitlement, payout, qualification,
  reward activation, or public publication.
- Local usage and feedback receipts **do not repair consensus query counters or
  `RateFact`**. The production query/cache boundary remains unchanged.
- Submitted challenge evidence is retained locally; the production
  `ChallengeFact` evidence-ID storage limitation remains. No reference field is
  repurposed to silently change provenance semantics.
- Descriptive attribution identifies artifacts that supplied evidence, were
  consumed, detected a discrepancy, or supplied a correction. Monetary
  allocation is `NOT_EVALUATED`; no E-level attainment, controller independence,
  person score, or payout share is inferred.
- The experiment does not train a model, discover a theorem, demonstrate
  universal learning, or prove that graphs outperform equal-information
  relational storage.
- Existing keeper lifecycle, epoch, genesis-history, dashboard-projection, and
  historical-query issues are not declared fixed by this pilot.
