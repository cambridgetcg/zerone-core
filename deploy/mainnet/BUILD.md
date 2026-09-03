# Building the `zerone-1` halt/archive image

Never run `docker build` or `fly deploy` with the repository root as its build
context. Git-ignored validator, testnet, keyring, or custom ceremony files can be
uploaded to a remote builder even when a Dockerfile does not copy them into its
final stage.

The only supported release path is the explicit tracked-source allowlist:

```bash
GO_IMAGE='golang:1.25.14-bookworm@sha256:<64-hex-digest>' \
RUNTIME_IMAGE='debian:bookworm-slim@sha256:<64-hex-digest>' \
ZERONE_RELEASE_TAG='<signed-annotated-tag-at-HEAD>' \
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<full-OpenPGP-fingerprint>' \
deploy/mainnet/build-image.sh <private-local-image-reference>
```

Release mode requires a clean checkout, the exact pinned `zerone-1` genesis,
digest-pinned build/runtime bases, and an annotated tag at `HEAD` with exactly
one valid signature from the configured full fingerprint. Release inputs are
materialized from that commit's Git objects, and the checkout is rechecked
immediately before Docker runs. Replacement refs and legacy grafts are rejected,
replacement-object lookup is disabled, and repository GPG-format/program
overrides cannot redirect annotated-tag verification. The temporary context contains only the Go
build allowlist, public genesis, entrypoint, Dockerfile, and a public context
marker. The audited output platform is always `linux/amd64`; the Dockerfile
also enforces that target before compiling. The context is deleted afterward.

After scanning and publishing that image, replace the deliberately invalid
`[build].image` placeholder in `fly.toml` with its exact registry digest. The
placeholder prevents Fly from silently uploading the repository and building it
remotely. Deploy only through the fail-closed gate in
[`deploy/FLY-DEPLOY.md`](../FLY-DEPLOY.md); it rejects tags and placeholders
before `flyctl deploy` can run and passes the verified digest explicitly.

For a local context-assembly test that never invokes Docker:

```bash
bash deploy/mainnet/build-image_test.sh
```

`MAINNET_BUILD_PROFILE=development` exists only for local images and emits a
visible `money.zerone.build-profile=development` label. Never publish or deploy
that profile.

## Runtime roles

`fly.toml` is the public `NODE_ROLE=signer` profile. Bootstrap validator and P2P
keys are one-time secrets for a fresh signer volume and must be removed after
the first successful boot.

Before arming the production checkpoint, replace that historical public shape
with `fly.halt-signer.example.toml` on the same app and volume. It contains the
exact F/A/H triplet and no Fly services or external address; the observer dials
`zerone-1.internal` privately. Do not arm F/A/H while `fly.toml` is still live.

`fly.observer.example.toml` creates an independent, custody-free
`NODE_ROLE=observer` volume. It privately dials only the official signer and
publishes no Fly service. Supply the exact F/A/H checkpoint triplet while it
syncs so the plan is permanently recorded on the volume.

Do not convert or clone the observer home into the serving archive. After the
signed H/A evidence is captured and both nodes are fenced, preserve the stopped
observer copy offline. On a separate `zerone_archive_data` volume, reproduce the
allowlist used by `halt-checkpoint-rehearsal.sh`: only `genesis.json`,
`config.toml`, `app.toml`, `client.toml`, fresh `node_key.json` and
`priv_validator_key.json`, reset `priv_validator_state.json`, and the copied
`application.db`, `blockstore.db`, `state.db` plus optional `evidence.db` and
`tx_index.db`. Hard-rollback that stopped serving copy so block store, state,
and application all end at `A`. No observer runtime marker, checkpoint marker,
WAL, addrbook, keyring, or incidental home file may cross this boundary.
The candidate recursively rejects symlinks, devices/FIFOs/sockets, and
multi-link regular files anywhere under the serving home; copy database bytes
onto the fresh volume instead of linking them.

Create `/data/.zeroned/.zerone-1-archive-transition.json` with exactly this
shape, using the reviewed values from the signed source evidence and the fresh
candidate keys:

```json
{
  "schema": "zerone-1-archive-transition-v1",
  "chain_id": "zerone-1",
  "checkpoint_state_height": "F",
  "final_committed_height": "A",
  "halt_trigger_height": "H",
  "genesis_sha256": "64-lowercase-hex",
  "cutover_initiation_evidence": {
    "successor_transaction_hash": "64-UPPERCASE-hex",
    "committed_height": "positive-decimal-string",
    "committed_block_time": "YYYY-MM-DDTHH:MM:SSZ",
    "public_notice_sha256": "64-lowercase-hex",
    "public_notice_publication_evidence_sha256": "64-lowercase-hex",
    "initiation_evidence_sha256": "64-lowercase-hex",
    "initiation_evidence_detached_signature_sha256": "64-lowercase-hex"
  },
  "source_observer": {
    "runtime_marker_sha256": "64-lowercase-hex",
    "node_id": "40-lowercase-hex",
    "validator_pubkey": "base64-ed25519-public-key"
  },
  "candidate": {
    "node_id": "fresh-40-lowercase-hex",
    "validator_pubkey": "fresh-base64-ed25519-public-key"
  },
  "expected_anchor_block_hash": "64-UPPERCASE-HEX",
  "expected_post_anchor_app_hash": "64-UPPERCASE-HEX",
  "source_evidence": {
    "signer_manifest_sha256": "64-lowercase-hex",
    "observer_manifest_sha256": "64-lowercase-hex"
  },
  "archive_construction_evidence": {
    "pre_transition_sanitized_snapshot_sha256": "64-lowercase-hex",
    "rollback_log_sha256": "64-lowercase-hex",
    "pre_transition_allowlist_manifest_sha256": "64-lowercase-hex",
    "excluded_future_artifacts": [
      "archive transition manifest",
      "rendered Fly configs",
      "archive adoption authority",
      "archive readiness",
      "final checkpoint",
      "open-beta decision"
    ]
  },
  "archive_transition_nonce": "64-random-lowercase-hex"
}
```

Publish and review the byte-exact SHA-256 of that manifest alongside the signed
source evidence. The cutover initiation object must prove the exact notice was
published and the exact successor transaction committed no later than the
signed CUTOVER initiation deadline. The deterministic renderer enforces that
deadline before emitting any adoption authority. Also bind the pre-transition
sanitized-copy, rollback-log, and exact
pre-transition allowlist-manifest hashes. Those two pre-transition artifacts
must exclude the transition manifest itself, rendered configs, adoption
authority, readiness, final checkpoint, and open-beta decision; otherwise the
hash graph would be circular. Put the same transition-manifest hash, source marker and
identity values, source evidence hashes, and expected A block/application hashes
into both Fly profiles. The runtime verifies the manifest byte hash, exact
object keys, candidate identity binding, source/candidate identity inequality,
and F/A/H values before it writes any candidate role marker.

Generate both configs and the adoption authority with
`render-archive-configs.sh`, reproduce all three outputs, and sign only the
generated authority with the transition key. Deploy the candidate first through
`fly-deploy-archive-authorized.sh`; that gate reruns the renderer and refuses
hand-authored MATCH/config bytes. It is service-free and P2P-isolated. Over
loopback it requires the documented
fresh-key A/A shape: status and ABCI at `A`, `catching_up=true`, the exact
reviewed A block and post-A app hashes, an empty A, subjective-tip `/commit A`
with `canonical=false`, and both block H and block-results H absent. Only then
does it atomically write a nonce/evidence-bound readiness attestation. Stop that
candidate and redeploy the same `zerone-1-archive` app and
`zerone_archive_data` volume with `fly.archive.example.toml` and identical
reviewed inputs. `NODE_ROLE=archive` consumes the exact readiness hash into its
permanent role marker; direct observer-to-archive and candidate bypasses fail
closed.

The final archive remains a private RPC/API origin with no public Fly service.
Public archive queries require a separate reviewed query-only TLS/rate-limit
proxy over private networking; never expose the origin directly. The runtime
marker pins the fresh non-validator identities, transition manifest, and
readiness hash, so changing or replaying any of them fails closed.
