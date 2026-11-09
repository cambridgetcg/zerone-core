# Validator Evaluation Oracle — Development Only

`cmd/oracle` is an optional advisory sidecar. It is disabled by default and is
not part of the 2026-07-29 validator rollout.

The former static “777 genesis axioms” evaluator was removed. The knowledge
module has no assumed axiom bedrock, and the oracle now requires an Anthropic
Messages API-compatible LLM endpoint for a non-uncertain response.

## Current behavior

- `POST /evaluate` sends a claim to the configured LLM and returns
  `accept`, `reject`, or `uncertain` with confidence and reasoning.
- `POST /prefetch` warms the in-memory cache asynchronously.
- `GET /health` reports sidecar liveness.
- The cache is in-memory and non-consensus.
- `zeroned` never queries the sidecar from this source. The Proof-of-Truth
  vote-extension transport is compiled off
  (`potVoteExtensionsReleaseEnabled = false` in `app/vote_extensions.go`), and
  no code path sets the validator's vote-extension configuration.
- Where that transport is on, the validator integration casts no vote for a
  claim when it has no usable verdict: the oracle is off, unreachable or
  erroring, the verdict is `uncertain` or below `min-confidence`, or the claim
  cannot be found.

Until November 2026 the integration voted `accept` in each of those cases: at
600,000 confidence when the oracle was off, unreachable or erroring or the
claim could not be found, and at the oracle's own confidence when the verdict
was `uncertain` or below `min-confidence`. That fail-open default is how a
false claim passed unanimously in R25 testing. The sidecar is advisory, not an
independent truth proof, and operators must not describe it as one.

## Local development

Build and test without configuring a live validator:

```bash
go test ./cmd/oracle ./app
go build -o build/zerone-oracle ./cmd/oracle
```

Starting it makes an external paid API dependency and requires a secret. Keep
the key outside repository files and logs:

```bash
./build/zerone-oracle \
  --port 8081 \
  --llm-api-key "$ANTHROPIC_API_KEY" \
  --llm-model "<reviewed-model-id>"
```

The current source exposes flags for API URL, model, token limit, and timeout;
inspect `./build/zerone-oracle -h` for the exact build.

## Validator boundary

Do not enable `[oracle]` on a live validator from this source head. The live
networks predate the consolidation, and source publication is not deployment
authority. A future release packet must state:

- whether oracle use is allowed or required;
- endpoint, model/version, timeout, and confidence policy;
- secret and data-egress handling;
- fallback behavior and outage rehearsal; and
- how validators avoid correlated dependence on one provider.

Because oracle outputs influence vote-extension content, every validator must
understand the determinism and fallback implications before activation, even
though the HTTP service itself is off-chain.
