# Public RPC edge

`zerone-rpc.fly.dev` is a stateless compatibility endpoint for mainnet reads.
It preserves the existing `/rpc/*` and `/rest/*` paths, accepts only
GET/HEAD queries (plus browser OPTIONS preflight), and sends admitted requests
to the existing mainnet RPC/REST endpoints. A staging probe proved that a newly
created app could resolve but not reach `zerone-1.internal`, so this immediate
edge deliberately preserves the current public upstream path instead of
claiming an unproven private boundary.

The policy is deny-by-default. Transaction and evidence broadcast, unsafe RPC,
peer dialing, subscriptions, searches, the REST transaction collection, and
the unbounded feegrant-issued route never reach the upstream.

This edge is an immediate interface hardening measure, not validator
containment. The signer remains directly reachable until the separately gated
sentry/query cutover closes its public services without risking a sole-signer
restart.

Build and stage by immutable revision:

```sh
revision="$(git rev-parse HEAD)"
source_date_epoch="$(git show -s --format=%ct "${revision}")"
fly deploy . \
  --app zerone-rpc-hardened-lab \
  --config deploy/rpc-edge.fly.toml \
  --build-only --push \
  --image-label "edge-${revision}" \
  --build-arg VERSION=v0.1.0 \
  --build-arg "COMMIT=${revision}" \
  --build-arg "SOURCE_DATE_EPOCH=${source_date_epoch}"
```

After probing a staging app, deploy the exact digest with a rolling strategy,
probe every allow and deny class externally, and confirm mainnet height keeps
advancing. Preserve the previous release digest for immediate rollback.
