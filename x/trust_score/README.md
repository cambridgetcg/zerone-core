# Trust-score calculation and chain adapter

This is the existing read-only score rubric, not identity verification, scientific
acceptance or permission to act. Its policy and query wire format are unchanged.

## Two responsibilities

```text
knowledge / qualification / capture_challenge
    → keeper.BuildScore: select observations through existing keeper contracts
    → scoring.Calculate(Input): deterministic arithmetic, no SDK or I/O
    → existing TrustScore query response
```

`scoring` is an ordinary Go package with standard-library dependencies only.
`Input` supplies submission accuracy, effective-weighted verification observations,
active-penalty count and cartel-strike count. `Result` returns the two accuracies,
composite, band and explanation. It has no address, clock, database, credentials,
callbacks, globals to configure or background work. Inputs are not mutated or
retained. See the executable `ExampleCalculate` in
[scoring/score_test.go](scoring/score_test.go).

The [keeper](keeper/score.go) remains the adapter: it selects the address's records,
asks qualification for active effective weights and penalties, asks the installed
challenge adapter for strikes, sorts the domain breakdown, and attaches the
address and observed block height. Producer state stays with those modules.
Existing optional-keeper defaults and query errors are preserved. No registration,
protobuf schema, store, writer, authority or consensus rule moves.

## Calculation contract

Despite the `Bps` suffix, accuracy uses the existing **1,000,000 scale**. Zero
submission accuracy and an empty/zero-weight verification set default to 500,000.
Verification accuracy is the integer weighted mean of positive-effective-weight
observations. The composite starts at the integer mean of the two accuracies,
then deducts 250,000 per strike and 50,000 per active penalty, with a floor at zero.
Strikes override the band; penalties prevent A. Existing explanation strings,
integer truncation, unsigned overflow behavior and absence of an upper clamp are
retained. This extraction adds no validation or normalization; adapters own the
meaning, provenance and validity of supplied observations.

The calculation costs linear time in supplied observations and constant extra
space. Calls share no mutable state and can be arranged by a caller without a
service or scheduler. The chain adapter still scans qualifications and challenges;
this change does not add indexing, pagination, parallel chain queries or horizontal
node scaling. Removing that adapter from an application would be a separate
registration decision, not a consequence of making the calculation portable.

## Exercise separately, then together

Use the repository's pinned Go toolchain. From repository root:

```sh
go test -mod=readonly ./x/trust_score/scoring -count=1
go test -mod=readonly ./x/trust_score/keeper -count=1
go test -mod=readonly ./app -run '^TestTrustScoreComposition$' -count=1
```

The calculation can also be tested with `GO111MODULE=off` from its own directory,
without the repository module graph. The application fixture uses actual stored
synthetic records and the installed adapters/query method. It checks baseline
query-byte hashes and read gas, retained producer-store bytes, sorting, penalty
expiry and challenge filtering. It does not start a node or prove deployment.
