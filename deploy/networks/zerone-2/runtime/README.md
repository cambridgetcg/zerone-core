# zerone-2 role-separated runtime kit

This directory is local deployment scaffolding only. It does not create Fly
apps, allocate addresses or volumes, generate keys, build with real custody
material, or deploy anything.

## Invariants

- The image contains the binary, public genesis, public network manifest, and
  runtime support files only.
- `build-image.sh` sends no Go source. It carries the exact audited Linux
  release binary into a minimal context and verifies its hash again in-image
  and on every boot before the first `zeroned` invocation.
- Chain ID is exactly `zerone-2`; the image-pinned genesis SHA is checked at
  build time and every boot.
- A volume is initialized once through a same-filesystem staging directory and
  atomic rename. Non-empty partial or unmarked homes fail closed.
- A validator needs a real genesis consensus key plus its expected P2P node
  key. Both public identities are derived with CometBFT before installation is
  accepted.
- An edge node rejects validator custody environment variables and proves its
  generated consensus key is not in the genesis validator set.
- A volume can never change roles. Key identity and signer state are checked on
  every restart.
- An exclusive OS file lock is held through the daemon `exec`, so a second
  process cannot open the same home. Persistent role, signer, peer, RPC, API,
  gRPC, and consensus settings are reasserted on every restart.
- Bootstrap secrets are removed from the environment before any `zeroned`
  child runs and are forbidden after first initialization.
- Edge RPC/REST origin listeners are closed by default. Opening them requires
  the exact `QUERY_ORIGIN_ENABLED=true` latch. Fly service exposure is a
  separate profile decision, so the private gateway path can be soaked without
  publishing either P2P or plaintext query ports.

## Build

The zerone-2 ceremony must first publish its complete four-file `real` artifact
directory. Keep the exact release binary used by that ceremony. The build
wrapper reruns the auditor in required-real mode, verifies the binary hash,
Linux GOOS/GOARCH, clean embedded VCS revision, signed annotated tag, and
independently authorized OpenPGP fingerprint, then carries those exact bytes
into the image. Host inspection uses only `go version -m`, which reads build
metadata without executing a possibly foreign-architecture binary. The actual
`zeroned version` check runs inside the matching target Docker stage. Runtime
Dockerfile and entrypoint bytes are materialized from the manifest's immutable
`source_commit` Git blobs, and HEAD/clean status are rechecked immediately
before Docker starts. Before any policy audit or parsing, the wrapper also
copies the exact four public ceremony files and release binary into one private
temporary snapshot; every subsequent audit, hash, metadata read, and context
copy uses only that snapshot. From the repository root:

```bash
RUNTIME_IMAGE='debian:bookworm-slim@sha256:<64-hex-digest>' \
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<40-or-64-lowercase-hex>' \
deploy/networks/zerone-2/runtime/build-image.sh \
  /absolute/path/to/audited-zerone-2-artifact-directory \
  /absolute/path/to/the-audited-linux-zeroned-binary \
  <local-or-private-image-reference>
```

The build is production-only: drill manifests, dirty worktrees, unpinned
runtime bases, source/tag mismatches, unauthorized tag signatures, modified
binaries, and architecture drift all fail closed. Docker builds for the
manifest's explicit `linux/amd64` or `linux/arm64` platform. The complete public
network manifest and a flattened provenance file are embedded under
`/network`; neither custody material nor repository source enters the context.

## Roles

### Validator

Use `fly.validator.example.toml` as a template outside Git. It intentionally
has no public service. On the first empty-volume boot only, provide:

- `VALIDATOR_KEY_B64`: base64 of `priv_validator_key.json`
- `NODE_KEY_B64`: base64 of the matching validator `node_key.json`

Use the Fly secret mechanism through stdin rather than command-line arguments
or shell history. After successful initialization, remove both bootstrap
secrets. A restart with either secret still present fails deliberately.

The validator requires private `PERSISTENT_PEERS` and `PRIVATE_PEER_IDS` for
the edge/sentry. Never start a second machine from the same key or volume.

### Edge

Use `fly.edge.example.toml` for the private soak. The edge generates and
persists ordinary non-validator CometBFT keys, rejects all validator custody
inputs, and has no public Fly service. P2P stays on the Fly-private network,
RPC stays on loopback, and REST/gRPC remain disabled, so neither API broadcast
nor public P2P mempool gossip can relay a transaction during the private soak
and validator registration. The signed minimum soak is 1,000 blocks and
60 minutes; a shorter observation is a no-go.

Before signing CUTOVER, use `fly.edge.query-soak.example.toml` to
open RPC and REST only on Fly private networking. It still has no public Fly
service or external P2P address. Pair it with the service-free query gateway
profile and prove `/status` reports `zerone-2`, REST queries work, and broadcast
and non-GET methods are refused. The exact `QUERY_ORIGIN_ENABLED=true` latch is
validated on every boot; values such as `1` or `yes` fail closed. gRPC remains
disabled in every initial-beta edge profile.

Only after main-key OPEN-BETA GO, successful commit of its exact pre-signed
history-link transaction, and verified OPEN-BETA-INITIATION-EVIDENCE, replace
the edge configuration with
`fly.edge.public.example.toml`. That profile keeps the already-soaked private
RPC/REST origin and adds exactly one direct public service: P2P on 26656.

The public Fly profile exposes only direct P2P on port 26656. It deliberately
has no Fly services or health checks for RPC (26657), REST (1317), or gRPC
(9090). A separate, mandatory query gateway must reach those listeners over
Fly private networking, terminate TLS, enforce per-client rate and connection
limits, restrict methods and request sizes, and publish the canonical RPC/REST
URL. Public gRPC is deliberately unavailable in the initial beta. Never expose
the plaintext query listeners directly. Replace every
placeholder outside Git; do not commit real app names, addresses, peer
strings, volume names, or secrets here.

All Fly deployments must pass the repository's signed-authority gate described in
[`deploy/FLY-DEPLOY.md`](../../../FLY-DEPLOY.md). The gate rejects tags and
placeholders, hashes the private config snapshot, binds its expected role, and
gives Fly both those exact signed bytes and the matching
`registry/repository@sha256:<64-lowercase-hex>` image reference.

## Recovery boundary

- Restore the complete marked home, including `data/priv_validator_state.json`,
  or initialize a truly empty volume with the offline backups.
- Do not copy only the validator key into a stale or partial home.
- Never attach a validator volume to two machines.
- The local file lock prevents two processes sharing one home; it cannot detect
  a copied consensus key on another volume or host. Deployment-level fencing
  and the one-machine signer invariant remain mandatory.
- Never restore a validator snapshot below a height already signed by that key.
- An edge volume cannot become a validator volume, and vice versa.

## Tests

```bash
bash deploy/networks/zerone-2/runtime/tests/entrypoint_test.sh
```

The test uses throwaway Ed25519 node and validator keys generated by the real
`zeroned` binary, while a wrapper intercepts only `start`. It proves restart
config repair, default-closed and private query-origin profiles, symlink and
partial-home rejection, and concurrent-process fencing. It never contacts a
network or Fly.
