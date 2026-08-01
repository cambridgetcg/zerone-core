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

## Cache quarantine and local build boundary

The Fly-integrated Depot cache that previously processed historically
key-bearing source images is quarantined. Until an owner-side
Depot project reset has been completed and its project identity, completion
time, and reviewer are recorded as release evidence, **do not use Fly or Depot
to build this image**. This prohibition includes `fly deploy` with a
Dockerfile, `fly deploy --build-only`, remote builders, Depot CLI/API builds,
and any repository-root upload. A successful final-image scan does not erase a
persistent builder cache.

The supported build wrapper invokes only the selected local Docker context. It
rejects TCP/SSH endpoints and Docker override variables, explicitly selects
the context-dependent `default` builder, requires its in-engine `docker`
driver, and does not use a remotely fetched Dockerfile frontend. It accepts
only a digest-pinned base, derives the source date from the exact commit, and
materializes exactly these two tracked Git blobs into a temporary context:

- `deploy/Dockerfile.rpc-edge`
- `deploy/public-edge-nginx.conf`

Build a local image from an exact reviewed checkout:

```sh
revision="$(git rev-parse HEAD)"
RPC_EDGE_SOURCE_COMMIT="${revision}" \
RPC_EDGE_VERSION=v0.1.0 \
RUNTIME_IMAGE='docker.io/library/debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818' \
  deploy/build-rpc-edge-image.sh zerone-rpc-edge:review
```

Use a fresh local Docker daemon or disposable VM with a Unix-socket context for
the release build. Do not attach it to a shared builder. After recording the
base digest, commit, context policy, test results, image scan, and exported
artifact digest, delete the local image and destroy the daemon/VM together with
its complete cache and configuration storage. Record that destruction as part
of the release evidence. `--no-cache` prevents reuse during the build; it does
not itself destroy cache retained by the daemon.

The wrapper never pushes or deploys. Publishing is a separate reviewed action:
push the scanned artifact to the intended registry, resolve the registry
reference to `registry/...@sha256:<64-lowercase-hex>`, and compare that digest
with the reviewed artifact. Deployment is another separate action and must
consume only that exact digest. The invalid image placeholder in
`rpc-edge.fly.toml` prevents an accidental Fly source build; replace it only in
the reviewed deployment input, never with a tag.

After probing a staging app from the exact digest, deploy the same digest with
a rolling strategy, probe every allow and deny class externally, and confirm
mainnet height keeps advancing. Preserve the previous release digest for
immediate rollback. Cache-reset evidence removes the explicit Depot quarantine;
it does not authorize repository-root contexts or mutable image references.
