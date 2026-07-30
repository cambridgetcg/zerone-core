# Non-signing Fly containment topology

These files define the smallest deployable separation around the existing
custodial signer:

```text
                         public TCP 26656
                    +---- sentry A --------+
validator (no Fly   |                       |   public-query full node
service or public IP)                       +--- nginx HTTPS 443
                    |                       |    (GET/HEAD RPC + REST only)
                    +---- sentry B --------+
                         public TCP 26656
```

The package does **not** deploy, rotate, clone, or restart the signer. The
`signer_boundary` in each `topology.json` requires an empty public-service set
and Machine restart policy `no`. A signer cutover remains a separate,
evidence-bound ceremony.

## Forced properties

- Sentries and query nodes use unique apps and fresh encrypted volumes.
- Every Fly profile consumes a prebuilt `registry/...@sha256:<digest>` image.
  Fly never rebuilds mutable source as part of a containment deployment.
- The runtime image's Zerone-specific payload contains only `zeroned`, one
  public genesis, the role entrypoint, and the read-only nginx policy. It
  contains no supplied `priv_validator_key.json`,
  `priv_validator_state.json`, or `node_key.json`.
- The entrypoint rejects every supported runtime key-source variable. It runs
  `zeroned init` itself only when the mounted data root has no application
  data. Fly's empty ext4 `lost+found` is accepted only when it is a real,
  empty, mode-`0700` directory owned by the runtime user.
- Each generated consensus key is zero-power local identity, never the active
  validator key. The entrypoint derives the genesis validator address and
  refuses equality.
- A role manifest binds the chain, genesis, binary, nginx policy, peer tuple,
  generated identities, key/state files, and both TOML files. Later drift
  fails before the daemon starts.
- Sentries expose only raw P2P TCP 26656. Their Comet RPC and SDK APIs are
  loopback-only/disabled; the validator node ID is both private and
  unconditional so it is not shared through peer exchange.
- Inter-app persistent peers use Fly's private `.internal` DNS names. Only a
  sentry's advertised external P2P address uses its public `.fly.dev` name.
- Public-query nodes expose only nginx on port 8080 through Fly HTTPS. P2P and
  Comet RPC are loopback-only. gRPC is loopback-only and has no nginx or Fly
  route.
- nginx permits only GET/HEAD and an explicit RPC/REST query allowlist.
  Broadcast, unsafe, peer-dial, subscription, search, and
  `/cosmos/feegrant/v1beta1/issued` routes are denied.
- Fly Launch `[[restart]] policy = "never"` produces Machine API restart policy
  `no`. No profile auto-stops or auto-starts.

## Identity-first ceremony

Sentry IDs are peers of one another, so they cannot be truthfully filled in
before their fresh volumes generate identities. Use the two-pass path:

1. Build and push the immutable image. Record its
   `registry/...@sha256:<digest>` reference and labels.
2. Create each app and a **new** Fly volume. Read the Machine/volume JSON and
   archive evidence that `encrypted == true`, the volume ID is new, the
   Machine is stopped, and restart policy is `no`.
3. For each stopped sentry Machine, set only
   `ZERONE_IDENTITY_CEREMONY=generate-only` for one start. The entrypoint
   initializes the otherwise empty volume, prints the new P2P node ID and its
   zero-power consensus address, writes a pending identity manifest, starts no
   daemon, and exits.
4. Independently compare the generated addresses to the active validator
   census. Replace every `REPLACE_*` value in both sentry templates and
   `topology.json`. Do not supply either generated key as a secret or
   environment variable.
5. Compute each topology digest using the exact payload below. Remove
   `ZERONE_IDENTITY_CEREMONY`, install the reviewed peer tuple and digest, and
   start with restart policy still `no`. The pending identity can be finalized
   only if no block database or extra signing data appeared.
6. After both sentry IDs are final, repeat the ceremony for the query node or
   initialize it directly with the two final sentry peers.

The topology digest is SHA-256 over this UTF-8 payload, including the final
newline:

```text
schema=zerone.fly-full-node-topology/v1
chain_id=<image chain ID>
genesis_sha256=<image genesis SHA-256>
role=<sentry|public-query>
moniker=<exact moniker>
validator_peer=<node-id@host:port, empty for public-query>
sentry_peers=<ordered comma-separated peers>
external_p2p_address=<host:port, empty for public-query>
```

One reproducible shell form is:

```sh
{
  printf '%s\n' \
    'schema=zerone.fly-full-node-topology/v1' \
    "chain_id=${CHAIN_ID}" \
    "genesis_sha256=${GENESIS_SHA256}" \
    "role=${ROLE}" \
    "moniker=${MONIKER}" \
    "validator_peer=${VALIDATOR_PEER}" \
    "sentry_peers=${SENTRY_PEERS}" \
    "external_p2p_address=${EXTERNAL_P2P_ADDRESS}"
} | sha256sum
```

Order is significant. Sentry peers and topology digests are public identity
data and belong in the reviewed manifest, not in a secret store.

## Image paths

Standard source build:

```sh
docker build \
  --file deploy/Dockerfile.full-node \
  --tag 'registry.fly.io/<artifact-app>:<candidate>' \
  --build-arg NETWORK=mainnet \
  --build-arg VERSION=<reviewed-version> \
  --build-arg COMMIT=<full-40-hex-commit> \
  --build-arg SOURCE_DATE_EPOCH=<commit-time> \
  --target full-node \
  .
docker push 'registry.fly.io/<artifact-app>:<candidate>'
```

Resolve the pushed reference to `registry/...@sha256:<digest>`, put that exact
reference in all three matching Fly profiles and in `topology.json`, and review
the equality before creating a Machine.

If the live binary must be preserved while abandoning a historically
key-bearing image, the optional `legacy-full-node` target accepts only a
digest-pinned parent reference and an exact binary SHA-256. It copies just
`/usr/local/bin/zeroned` into the clean runtime base and uses the public
genesis from this repository:

```sh
docker build \
  --file deploy/Dockerfile.full-node \
  --target legacy-full-node \
  --build-arg NETWORK=mainnet \
  --build-arg LEGACY_SOURCE_IMAGE='registry.fly.io/<app>@sha256:<64-hex>' \
  --build-arg LEGACY_ZERONED_SHA256='<64-hex>' \
  --build-arg VERSION=<containment-bundle-semver> \
  --build-arg COMMIT=<reviewed-source-commit> \
  --build-arg SOURCE_DATE_EPOCH=<positive-unix-seconds> \
  .
```

This is layer separation, not source attestation. The extracted binary still
needs an external signature/provenance decision and exact behavior rehearsal.
For this target, the OCI `version` and `revision` identify the containment
bundle and wrapper source; they do not attribute the legacy executable. Its
separate labels record the source image digest, executable digest, and
`source-revision=unattributed`.

## Upgrade and hostile-event behavior

The persisted role manifest deliberately binds the binary, genesis, gateway
policy, identities, peer tuple, and generated configuration. An image or
configuration change on the old volume therefore fails before `zeroned`
starts. There is no in-place exception switch.

Both image targets and every boot also ask the selected binary to decode and
validate the frozen public genesis. Failure is a release stop: do not delete
unknown fields or regenerate an already-live chain's genesis to make a newer
binary accept it. Preserve an exactly hashed compatible binary through the
clean legacy-extraction target, or perform a separately governed protocol
migration.

Treat a planned full-node upgrade as a replacement:

1. build and rehearse a new digest without changing the running node;
2. create a new encrypted volume and Machine with restart policy `no`;
3. run the identity ceremony, check the fresh address against the live
   validator census, and bind the new peer tuple;
4. let the replacement sync and prove its public/private exposure;
5. update dependent peers one node at a time, then stop the superseded
   Machine; and
6. retain the old stopped Machine and evidence until rollback expiry.

For a consensus-height binary upgrade, non-signing replacements may be staged
before the height, but their activation must follow the independently approved
signer halt/upgrade ceremony. A changed genesis is a new network: use a
network-specific image and wholly new volumes, never a manifest rewrite.

If a sentry or query node is suspected hostile, stop it first; these roles hold
no active validator key and availability is subordinate to containment. Do not
restart its volume or preserve its generated identity. Remove the compromised
node ID from every peer tuple, replace it from a reviewed image digest on a
fresh encrypted volume, recompute all affected topology digests, and externally
probe the signer boundary before restoring traffic.

## Deployment gates

Do not replace placeholders or deploy from this directory until:

- custody review has classified the sole signer key;
- the image digest and public genesis digest have been independently checked;
- both sentry node IDs and every peer address are final;
- the signer has an approved no-public-service transition;
- public DNS points only to the query edge, never the signer;
- external probes prove signer RPC/REST/gRPC/P2P ingress closed, sentries expose
  only 26656, and the query app exposes only HTTPS;
- restart, config drift, key drift, topology drift, and non-empty-volume drills
  all fail before daemon start.

Fly volumes are encrypted by the platform, but a process inside the guest
cannot prove its own control-plane encryption flag. The ceremony must therefore
bind the `fly volumes list --app <app> --json` encryption observation and the
separate Machine attachment into the external journal before first start.

### Current fail-closed status

The source-built SDK 0.53 / IBC-Go 10 binary currently refuses both checked-in
live-network genesis files because their deployed IBC-Go v8 transfer/channel
shapes include fields such as `denom_traces` and `channel_genesis.params` that
are not v10 genesis fields. That is a migration boundary, not a field to delete
in place. Consequently:

- the standard `full-node` target is intentionally not deployable against the
  current live genesis;
- a clean `legacy-full-node` extraction is the only compatible containment
  candidate until a governed migration, but the extracted binary remains
  source-unattributed and must match the independently observed executable
  SHA-256 exactly; and
- no signer cutover is authorized while operator custody, a stopped-height
  export, and the recovery gate remain unresolved.

The Fly templates retain placeholders so an accidental `fly deploy` cannot
silently choose identities, topology, or an image on the operator's behalf.
