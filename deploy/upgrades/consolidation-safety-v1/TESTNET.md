# H1 testnet preparation (local only)

This is the lighter **testnet** bar in `zc-h1-testnet-bar`, not activation
permission and not a replacement for the mainnet release bar. Source remains
`65c19cd8b00bdfff9b80705b776fd0d49719398a`; do not re-cut H1 or build trunk.
The preserve branches are immutable review references. A PR based on
`preserve/h1-ignition-kit` is for review only, not a merge into preserve.

## Gates and evidence classes

| Gate | Testnet | Mainnet (unchanged) |
| --- | --- | --- |
| Source/image | Frozen H1 source, reproducible build, recorded source/binary SHA-256, manifest-digest image; no signing ceremony | Signed annotated `v*` tag, authorized signer, reproducible digest-pinned image **and attestation** |
| State rehearsal | Pass on a **provided current key-scrubbed testnet copy**; retain input identity and results | In-place mainnet-state rehearsal retaining chain ID **`zerone-1`**; no export/import or relabelled synthetic chain substitute |
| Before H | Build, verify and make the exact image available **before proposing/arming H**, not during the halt | Full release packet and coordinated readiness |
| Authority | One explicit yes from Yu for the testnet halt; may be the same yes as the testnet mint seal | Separate mainnet authorization; testnet success grants none |

Fetching a state copy needs a separate yes from Yu. Nothing here fetches it.
A local dry run is not a built image. A synthetic fixture is not current testnet
state. A checksum, a provider's copy manifest and a local success receipt are
not independent authentication or authorization.

The original `drill.sh` remains the historical throwaway full-ladder exercise.
The new `state-copy-drill.py` is a keyless **application migration drill**, not
a CometBFT consensus or frozen-RPC rehearsal. Whether that evidence is sufficient
for the state-copy gate remains an explicit reviewer/operator decision. It must
not be reported as an in-place mainnet rehearsal or image-runtime test.

## Reproducible image recipe

`provenance.py` adapts trunk's `deploy/legacy-observer/provenance.py` at
`46fa54369353b85438d649b0985299159601cd4b` (file SHA-256
`2480cb874eb7510e9712ec2d42652b13f7ef21b3c65d5e12298fd8a46df7677e`):
immutable Git blobs, closed compile-resource allowlist, normalized tar metadata,
per-file hashes, read-only source, isolated build outputs and independent caches.
Exporting the old source reproduces the original recipe's exact tar SHA-256
`62054eed39981c367c9c6d5bdf7bb19e54766afe73d336c2ea7f4a45cbfb400f`.

H1's normalized 674-file closure is pinned to:

- Commit: `65c19cd8b00bdfff9b80705b776fd0d49719398a`
- Git tree: `8e6a2df14088522e7103da7f76dd8b6f3b5cda29`
- Source tar SHA-256: `2ab7a6b96ad6f2294ad7bbb7ddf5140c47763b8fca3ad9ae864e23927e49e722`

The explicit differences from the forensic observer recipe are intentional:

- Frozen **H1**, not the old live executable or the observer dependency patch.
  No dependency/source patch is applied.
- The pinned Go 1.25.14 Linux amd64 builder from trunk's observer build notes:
  `registry-1.docker.io/library/golang@sha256:c268a04d59aea0b180ed9946a658cfab9e7b3391dc90eed6e4969ccff98c851f`.
- `CGO_ENABLED=0`, matching H1's existing static cross-build target, instead of
  the old forensic build's `CGO_ENABLED=1`. This is a new testnet build profile,
  **not** a claim of byte identity to a previously released executable.
- `-buildvcs=false -trimpath`, explicit H1 version/commit linker fields,
  `GOAMD64=v1`, `GOTOOLCHAIN=local`, `GOWORK=off`, `-mod=readonly`, `-p 4`.
- No image pulls, no module downloads, `--network=none`; an already-present
  builder and explicitly provided public module cache are prerequisites.
- `go mod verify` rehashes the cached dependencies before each build. Inherited
  Go configuration/private-proxy bypasses are disabled (`GOENV=off`,
  `GONOPROXY=none`, empty `GOPRIVATE`, `GOVCS=*:off`), including helper builds.
- Two fresh compilation caches must produce byte-identical binaries before
  packaging. This is reproducibility within the pinned local build environment,
  not independent third-party reproduction or vulnerability clearance.

Requirements: Python 3.11+, local Git objects, Docker with a local Unix socket,
the pinned builder already available, and a prepopulated public Go module cache.
A missing prerequisite fails closed; these tools never install or fetch it.
All outputs must be NEW directories under an explicitly selected scratch parent.

From the repository root (set `WORK` to an authorized scratch directory):

```sh
KIT=deploy/upgrades/consolidation-safety-v1
python3 "$KIT/provenance.py" --repository "$PWD" \
  --module-cache /path/to/public-go-module-cache --output "$WORK/image-dry-run" --dry-run
```

This exports real source and writes `source-closure.json`, `source.tar`, `src/`
and a command plan. `binary_sha256` and `image_digest` remain **null**;
`reproduced` is false. It does not call Docker.

On a separately authorized build host with prerequisites already available:

```sh
python3 "$KIT/provenance.py" --repository "$PWD" \
  --module-cache /path/to/public-go-module-cache --output "$WORK/image-build"
```

Success produces `provenance.json` (`status: built-local`, `reproduced: true`),
`build-1/zeroned`, `build-2/zeroned`, and `image.oci.tar`. The receipt records
source and executable SHA-256, exact builder/environment/commands, OCI archive
SHA-256 and the **OCI image manifest digest** (not the Docker config/image ID).
Retain logs and failure receipts; do not reuse an interrupted output directory.

The OCI image is a static `scratch` runtime containing only `/usr/local/bin/zeroned`.
Its default argv is `start --home /data/.zeroned`, matching the legacy mount
layout. It includes no genesis, keys, copied state, shell, CA bundle, Fly config,
initialization logic, signing ceremony or remote services. Existing home
configuration/volume compatibility and starting the packaged image remain
separate rehearsal checks. H1's startup lineage wall is unchanged.

Registry publication is a **separate authorized action**, not implemented here.
Use a transfer that preserves the OCI manifest digest, then read the registry
manifest back and confirm it matches the receipt. Do not substitute a mutable
tag or a config ID. Verify platform and image availability before H.

`deploy/testnet/Dockerfile` and `deploy/testnet/fly.toml` are deliberate tombstones;
this kit does not revive them. No new live deployment config is authorized here.

## Provided-copy contract and drill

The provider must first obtain authorization to fetch/copy and scrub state.
Supply a **stopped consistent GoLevelDB application copy**, not a whole node home,
archive of a disk with deleted keys, export genesis, or live database:

```text
provided-copy/
  copy.json
  application.db/
    CURRENT
    MANIFEST-...
    ... LevelDB files only ...
```

No node configuration, Comet DBs, addresses, keyrings, node/private-validator
keys, symlinks, hardlinks, special files or extra directories are accepted.
The wrapper inspects names/types before reading any contents. It hashes the
allowed inputs, clones them into a NEW disposable home, checks the cloned bytes,
and verifies the provided copy is unchanged afterward. It never opens the
original DB through Go. Provider scrubbing is an attestation: filename checks
cannot prove that arbitrary DB/log bytes contain no private material.

`copy.json` is supplied by the provider, with these fields (example placeholders
are deliberately not usable hashes/heights):

```json
{
  "schema": "zerone-h1-scrubbed-copy/v1",
  "chain_id": "zerone-testnet-1",
  "height": 123,
  "app_hash": "<64 lowercase hex from the copied application commit>",
  "block_time": "<RFC3339 time of that committed block>",
  "backend": "goleveldb",
  "key_scrubbed": true,
  "stopped_consistent_copy": true,
  "evidence_class": "provided-state-copy",
  "provenance": "<provider, source observation and copy/scrub procedure reference>"
}
```

The script verifies height/app hash against the application DB, but chain ID,
block time, currency of the copy and its relationship to the live chain depend
on the provider's independently checked provenance. It cannot authenticate a
chain's identity from a bare application DB. Mainnet is refused; do not relabel
mainnet metadata as testnet. Other database backends need a separately reviewed
adapter, not automatic conversion.

Build the keyless helpers with locally installed Go and cached modules:

```sh
python3 "$KIT/state-copy-drill.py" build-helpers --repository "$PWD" \
  --output "$WORK/helpers"
python3 "$KIT/state-copy-drill.py" drill --provided-copy /path/to/provided-copy \
  --helpers "$WORK/helpers" --output "$WORK/state-drill"
```

The build uses the exact production closure from old commit
`2e37c4c86c31e67515d6aa07b6fd406083e012ea` and H1 plus the separately hashed
`rehearsal.go` driver. Helpers are NOT the shipping node binary. `helpers.json`
records both source identities and helper executable hashes.

The driver preserves the copied application's records and chain ID. It:

1. Checks the source height/app hash; refuses an existing plan or completed H1.
2. Stages a clearly labelled **local-only** H1 plan with the old keeper at L+2;
   finalizes/commits an empty synthetic L+1 block. This is not a governance vote.
3. Reopens the old app and requires the specific upgrade-needed error at H,
   with the real `data/upgrade-info.json`. Never commits the failed block.
4. Starts H1 with `loadLatest=true`, exercising its actual startup lineage wall.
5. Runs H1 through `FinalizeBlock → PreBlock → ApplyUpgrade`, then commits.
   No direct handler shortcut, genesis import or VersionMap patch is used.
6. Checks the entire target VersionMap, K6/P2/L5/V1, H1 marker and done height,
   no pending plan, completed-lineage restart, one further block, and another restart.

`drill.json` records input hashes, provenance, phases, starting/final app hashes,
explicit block times, and limitations. Blocks have empty transactions, votes,
evidence and proposer, with one-second intervals after the provided block time.
Those synthetic inputs affect ordinary block hooks/economics and are **not live
replay**. The drill tests migration compatibility of that copy, not validator
signing, quorum, networking, governance, exact production app hashes or packaged
image startup. Never return its mutated working home to a real validator.

## Ship-script fixes and authorization

The repository-owned `ship-consolidation-safety.sh` wrapper replaces the old
operator-home script **only after separate operator adoption**. This change
never overwrites the installed copy. It fixes the four defects:

1. Requires `--zeroned /explicit/current/verified/CLI`; no deleted
   `$HOME/go/bin/zeroned` fallback. The installed CLI location is operator input,
   not guessed by this offline preparation run.
2. Uses the same explicit bonded `--operator` for proposing and voting, rejects
   `faucet`, and checks the operator is unjailed/bonded with voting power.
3. Emits `fly deploy --config <reviewed-testnet-config> --app zerone-testnet-1
   --image <registry/name@sha256:...>`. It never invokes Fly or builds at the halt.
4. Requires **four consecutive successful equal-height pairs**: Comet status at
   H−1 or H, and ABCI committed height exactly H−1. Comet can stage H before ABCI
   refuses it; a status value H alone does not prove whether H committed.
   Darkness, invalid responses, movement and early stalls reset the streak;
   ABCI reaching H (or Comet passing H) refuses deployment. Waiting is bounded.
   A dark RPC is never a halt.

All paths, endpoints and proposal height are explicit. Default invocation prints
an offline JSON plan and accesses no CLI, home, config, keys or network:

```sh
bash "$KIT/ship-consolidation-safety.sh" --chain-id zerone-testnet-1 \
  --zeroned /path/to/current/verified/zeroned --ops-home /operator/supplied/home \
  --operator validator --node tcp://operator-supplied-endpoint:26657 \
  --image registry/name@sha256:<actual-manifest-digest> \
  --fly-config /path/to/reviewed-testnet.toml --height <chosen-H> \
  --proposal /new/local/proposal.json
```

Do not execute the placeholder command verbatim. `--execute` additionally
requires `SHIP_ACK=zerone-testnet-1` after Yu's explicit testnet halt yes,
`--image-receipt`, `--drill-receipt` and `--image-available`. The last flag is the
operator's verification that the prebuilt digest is available before proposing;
it is not a registry check by the script. The real provided-state drill and
image must identify the same frozen source. Dry runs/fixtures cannot qualify.
Review the voting period, block cadence and chosen H; the simple height-headroom
check is not a timing guarantee. Failed/ambiguous transactions require exact
reconciliation, never blind resubmission. Proposal ID comes from the included
submit transaction, not the latest voting proposal. Fees/gas retain the old
script's 300000 values; the deposit is read from governance params.

The script stops after confirmed halt and prints the pinned deployment command.
Deployment is a separate operator action. After deployment, verify applied H,
K6/P2/L5/V1, resumed height, OpenQuestions and supply continuity; retain evidence.
Mainnet is not accepted by this lighter-profile executable.

## Offline checks and remaining release work

```sh
python3 "$KIT/test-offline.py" --repository "$PWD" --output "$WORK/offline-tests"
```

This builds both historical helpers, creates a synthetic old-state fixture from
public data (no private key), tests the actual migration/restarts and negative
startup/hash paths, and tests the ship adapter with mocks plus its real offline
wrapper. It also performs two real H1 source exports and checks the old export
against trunk's known hash. Outputs are retained, not a success-shaped transcript.

A passing fixture suite does not finish the testnet bar. Outstanding until
separately performed and reviewed: pinned Docker build and executable hashes,
image-runtime exercise/registry availability, current scrubbed-copy delivery and
passing drill, operator adoption/current CLI and deployment configuration,
independent review, and Yu's testnet halt yes. Mainnet retains its full signed-tag,
attestation and in-place `zerone-1` rehearsal requirements without exception.
