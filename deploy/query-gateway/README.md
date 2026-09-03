# Query-only TLS gateway

The zerone-2 edge and sanitized zerone-1 archive are private plaintext origins.
Their Fly profiles never expose RPC, REST, or gRPC directly. This gateway is
the only initial-beta public query path.

It terminates public HTTPS at Fly Proxy, accepts GET/HEAD only, and forwards
deny-by-default allowlists of reviewed CometBFT RPC and typed REST reads. The
REST surface is limited to node sync metadata, native-denom supply and account
point reads, and the bounded liquidity-pool registry. Supply and account
balance queries accept exactly `denom=uzrn`; fixed reads and pool listings
accept no query string, so pagination and `count_total` variants fail closed.
Knowledge lists remain private because some filter combinations scan the fact
store. It caps request bodies/connections/timeouts
and applies per-client request limits using Fly's `Fly-Client-IP` header.
Response-heavy `/block`, `/block_by_hash`, and `/tx` reads use the stricter
2r/s request zone and a two-connection limit because SDK query gas does not
bound them.
All proxied GET/HEAD requests are forwarded without a request body or
`Content-Length`. Credentialless browser reads receive wildcard CORS;
credentials are never allowed.

Broadcast, JSON-RPC POST, REST POST, search/index scans, monolithic or chunked
genesis, training bundles, recursive graph queries, leaderboards, dial/unsafe,
WebSocket, event-subscription, and private peer/consensus-topology routes are
not proxied. Both Comet `/abci_query` and Cosmos REST
`/cosmos/base/tendermint/v1beta1/abci_query` are explicitly blocked because
they can route unbounded simulation and arbitrary application queries. Public
gRPC is deliberately unavailable in the initial beta. The origin must keep a
finite SDK query-gas limit enabled; the gateway limits do not replace it.

A loopback-only Go helper polls both Comet RPC and typed REST every two seconds,
caps every response at 64 KiB, shares a strict two-second deadline across each
probe set, and caches the semantic result for at most five seconds. Both roles
require their exact `EXPECTED_CHAIN_ID`, a canonical positive latest height,
and a structurally valid REST syncing boolean that agrees with Comet status.
The active `zerone-2` role also requires `catching_up=false`, a latest block no
more than 30 seconds old, and a timestamp no more than 10 seconds in the future.

The fresh-key frozen `zerone-1` archive is instead bound to the exact signed
A/A checkpoint through `EXPECTED_ARCHIVE_HEIGHT`, the lowercase 64-hex post-A
`EXPECTED_ARCHIVE_APP_HASH`, and the lowercase 64-hex A block ID in
`EXPECTED_ARCHIVE_BLOCK_HASH`. Its Comet status height and latest block hash
must equal A and report `catching_up=true`; `/abci_info` must report height A
and the expected AppHash (whether Comet encodes it as hex or base64); the
bounded `/block?height=A` response must repeat the expected chain, height, and
block ID; and `/block?height=A+1` must return Comet v0.38's exact bounded
HTTP-500 JSON-RPC unavailable-height error. Its block time must be structurally
valid but may be stale because wall-clock liveness is not an archive invariant.
Any RPC or REST probe error replaces the cached success rather than extending
it.

The POSIX entrypoint supervises both the helper and foreground Nginx. An
unexpected exit by either terminates the other and fails the container; a
container stop terminates the helper and asks Nginx to drain connections
gracefully.

`/gateway-health` returns the helper's minimal JSON with HTTP 200 only while
that result is ready, and HTTP 503 otherwise. Every public RPC or REST origin
location first performs an internal `auth_request`; an unhealthy, stale, or
wrong-chain origin is therefore not routable even if it still accepts TCP.
Before each probe, the helper requires the origin name to resolve to exactly one
unique `fdaa::/16` IPv6 address and pins that full probe set to the validated
address. A successful internal authorization response carries that exact
canonical address to Nginx, which proxies the corresponding public request to
the captured numeric 6PN without performing a second DNS lookup. A DNS change
therefore takes effect with the next successful probe, while a multi-Machine
replacement overlap or failed/expired probe clears the address and fails
closed. The private origin must listen on its Fly 6PN interface as well as
remain inaccessible publicly.

## Build and deploy boundary

Build from the signed release commit. `nginx-image.txt` is the single reviewed,
immutable Nginx base pin used by both the release build and the real config
test; a caller-supplied `NGINX_IMAGE` is accepted only when it exactly matches
that signed pin. The Go 1.25.14 builder is independently pinned by digest.
`render-archive-gateway-config.py` is release-pinned and generates the archive
Fly profile from the already-verified RELEASE plus FINAL A/E/B inputs; the
profile must not be hand-populated. OPEN signs the concrete rendered config
hash, and unresolved placeholders are rejected at startup.

```bash
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
- `/gateway-health` rejects catching-up, stale, and wrong-chain origins;
- a REST bank/supply query succeeds;
- knowledge fact/tree/training/list routes fail closed;
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
GOWORK=off go test ./deploy/query-gateway -count=1
bash deploy/query-gateway/entrypoint_test.sh
bash deploy/query-gateway/build-image_test.sh
bash deploy/query-gateway/nginx-config-test.sh
```

The Go tests exercise semantic parsing, active freshness, single-6PN topology,
archive block/AppHash binding and A+1 absence, RPC/REST response bounds, cache
expiry, and readiness HTTP behavior. The entrypoint and image-context tests use
fake processes and make no network or Fly changes. The final config test
requires Docker: it builds the actual gateway from the committed base pins and
runs that image's own `nginx -t` against the rendered production template.
