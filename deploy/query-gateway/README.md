# Query-only TLS gateway

The zerone-2 edge and sanitized zerone-1 archive are private plaintext origins.
Their Fly profiles never expose RPC, REST, or gRPC directly. This gateway is
the only initial-beta public query path.

It terminates public HTTPS at Fly Proxy, accepts GET/HEAD only, forwards an
explicit low-cost CometBFT RPC allowlist to port 26657, forwards Cosmos REST
queries to port 1317, caps request bodies/connections/timeouts, and applies
per-client request limits using Fly's `Fly-Client-IP` header. Broadcast, JSON-
RPC POST, REST POST, dial/unsafe, WebSocket, event-subscription, and private
peer/consensus-topology routes are not proxied. Both Comet `/abci_query` and
Cosmos REST `/cosmos/base/tendermint/v1beta1/abci_query` are explicitly blocked
because they can route unbounded simulation and arbitrary application queries;
supported application queries use typed GET-only REST. Public gRPC is
deliberately unavailable in the initial beta.

`/gateway-health` proxies the origin's `/status`; it fails when the origin is
unreachable. The release smoke test must additionally inspect `/status` and
require the exact expected chain ID, because an HTTP proxy alone cannot
interpret the returned JSON safely.

## Build and deploy boundary

Build from the signed release commit. The base must be a reviewed nginx image
referenced by full immutable digest:

```bash
NGINX_IMAGE='docker.io/library/nginx@sha256:<64-lowercase-hex>' \
ZERONE_RELEASE_TAG='<signed-annotated-tag>' \
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<40-or-64-hex>' \
  deploy/query-gateway/build-image.sh zerone-query-gateway:release
```

Generate an SBOM, scan and sign the result, push it, and record its full
`registry/repository@sha256:<64-hex>` reference in the signed release packet.
Deploy through `deploy/fly-deploy-authorized.sh`: the private z2 gateway uses
the main-key DARK-START authority (and dark-initiation evidence after expiry),
while either public gateway requires OPEN-BETA plus verified on-time
OPEN-BETA-INITIATION-EVIDENCE and the transition-signed FINAL checkpoint chain.
The lower-level pinned gate is for local checks.

Use `fly.zerone-2.private.example.toml` with the service-free edge
`fly.edge.query-soak.example.toml` before signing CUTOVER. From inside the Fly
private network, require:

- gateway `/status` reports `zerone-2` and matches the validator/edge height
  and app hash;
- a REST bank/supply query succeeds;
- `GET /broadcast_tx_sync`, JSON-RPC POST, and REST transaction POST fail;
- request-size, connection, and rate limits produce the expected rejection;
- the gateway has no public service or IP during the soak.

Only after main-key OPEN-BETA GO and successful exact history-link initiation
evidence, switch the same reviewed z2 gateway app to
`fly.zerone-2.public.example.toml`. Use
`fly.zerone-1-archive.public.example.toml` for the separately validated A/A
archive origin. The full immutable gateway image reference and each reviewed
Fly config SHA-256 are part of OPEN-BETA and must be extracted by the signed
authority gate rather than transcribed.

## Local tests

```bash
bash deploy/query-gateway/entrypoint_test.sh
bash deploy/query-gateway/build-image_test.sh
```

The tests use fake nginx/Docker processes and make no network or Fly changes.
