# zerone-1 replay observer (full replay from genesis)

Fly app and volume: `zerone-1-replay` and `zerone_replay_data`. The names `zerone-1-observer` and
`zerone_observer_data` stay reserved for the checkpoint observer in `deploy/mainnet`
(`fly.observer.example.toml`, `fly-cutover-authorized.sh`), so a signed cutover never deploys onto
this node.

A zero-power node for the existing zerone-1 chain. It replays the whole history from genesis by
blocksync from the official validator over Fly's private 6PN network, and then follows the tip. Its
last binary is the CometBFT-patched legacy build, 2e37c4c plus
`deploy/legacy-observer/observer-dependencies.patch` (CometBFT 0.38.25, Go 1.25.14). That makes
the replay a proof that this binary recomputes every app hash and results hash the validator
committed from height 527603 onward. CometBFT checks both hashes against the next signed header
for every block it applies.

This differs from `deploy/legacy-observer`, which starts from a signed checkpoint by state sync and
checks only the blocks after it.

## Why a ladder of binaries

zerone-1's current lineage (genesis sha256 `c30a523b…170e`, 2026-07-08) went through five
software upgrades and five other image changes. No single binary can replay it. 2e37c4c already
holds the handlers for all five plans, so x/upgrade refuses it with `BINARY UPDATED BEFORE
TRIGGER` as soon as the first plan is stored (x/upgrade v0.1.4 abci.go:115-123). Its InitChainer
also records module versions newer than the genesis binary's (liquiditypool 3 against 2).

`ladder.tsv` lists, in order, the binary that executed each height range and how that range ends:

- `upgrade:<plan>`: the old binary stops by itself at the plan height and writes
  `data/upgrade-info.json`.
- `halt`: the rung starts with `--halt-height`; SDK 0.50 refuses blocks at or above it.
- `tip`: the last rung, which follows the live chain.

The plan heights are the chain's applied plans. Each non-upgrade boundary is the first block after
a restart gap in block times; it is exact to within one block (RUNBOOK section on boundaries).
Historical binaries are identified only by sha256. They are copied from the binary layer of the
release image that ran them (`extract-rung-binary.sh`, gzip or zstd layers). The picker refuses a
binary layer under 1 MiB before fetching it, and a layer holding anything but one
`usr/local/bin/zeroned`. On 2026-11-06 the current release's layer gave the live validator's
binary as the known answer before any rung was taken. The other layers of those legacy images
bake the validator's key files, so those images are never pulled, run or unpacked whole.

## What the node is and is not

- Its consensus key is fresh, made by `zeroned init` on the volume. The start script refuses to
  run if that key is the live validator's (address `93DBC1A1…B844`, pubkey `LA944mpk…l80=`) or
  appears in genesis. It also refuses if `priv_validator_state.json` ever leaves height 0, or if a
  pinned config file changes. `double_sign_check_height = 0`.
- P2P listens on 127.0.0.1 only. It has `max_num_inbound_peers = 0`, `pex = false`, and one
  persistent and unconditional peer: `ed8c8d49…6e@zerone-1.internal:26656`, the validator over the
  org's default 6PN.
- RPC is on 127.0.0.1. REST, gRPC, pprof and Prometheus are off. The mempool is `nop`, so it
  relays nothing. The Fly machine has no services and no IPs.
- `block_time_tolerance = "60s"`, CometBFT 0.38.22+'s wall-clock check. It rejects only
  `block.Time >= now + tolerance` (state/validation.go:124), so replaying old blocks never trips
  it. The handshake replay path does not apply it at all. At the tip it would refuse a block more
  than 60 s ahead of this machine's clock; the script then waits and restarts the node.
- While it catches up it is load on the only signer, so it backs off: if the validator's newest
  block is more than 150 s old, it stops asking for blocks for 10 minutes (`validator_lag_pause`).
- Every deliberate stop exits 0 under restart policy on-failure. The machine stays stopped and the
  volume keeps the evidence. A failed proof also writes `/data/observer/HALTED-<reason>`, and the
  script will not resume a marked volume.

## Reading it

The start script writes one JSON line per event to stdout, each prefixed `OBSERVER `, so
`fly logs` is enough:

- `progress`: height, rate, peers, catching_up, header app hash.
- `apphash_spot`: the applied app hash at H against the validator's header H+1.
- `rung_done` and `rung_advance`: one rung ended and the next began.
- `APPHASH_MISMATCH` or `LADDER_WRONG`: the run stopped itself.
- `UPGRADE_NEEDED`: a new plan has no rung yet; the node stays up.

`watch.sh` reads those lines, plus the validator's block cadence and the machine state, and
changes nothing.

The mismatch lines it acts on:

- blocksync: `"message":"Error in validation"` with `err` `wrong Block.Header.AppHash.  Expected
  <this node>, got <header>` (or `LastResultsHash`), repeated per retry;
- consensus: `CONSENSUS FAILURE!!!` with `+2/3 committed an invalid block: wrong
  Block.Header.AppHash…`;
- restart: `error on replay: …` or `state.AppHash does not match AppHash after replay`.

## Files

| file | what |
|---|---|
| `observer-start.sh` | entrypoint: init, checks, ladder supervisor, progress and spot checks, watchdogs |
| `ladder.tsv` | the rungs; rows marked `UNSET` make the build refuse |
| `Dockerfile` | COPY-only on the digest-pinned debian base the legacy observer uses |
| `build-image.sh` | checks every input, stages a minimal context, builds linux/amd64 locally |
| `extract-rung-binary.sh`, `rung_layer.py` | copy one historical zeroned out of its image layer |
| `machine-config.json`, `machine-config.following.json` | Fly machine settings for the replay (performance-2x, 4 GB) and for following (shared-cpu-2x, 2 GB) |
| `release_heights.py` | reproduce the boundary heights from public RPC block times |
| `watch.sh` | read-only progress, cadence and cost watch |
| `test/test-ladder.sh` | supervisor logic with stand-in binaries (no network) |
| `test/test-spot.sh` | the app-hash spot check, including the wait for the validator's next header at the tip (no network) |
| `test/smoke-real.sh` | init, config, InitChain and clean stop with the real lane binary (no network) |
| `test/test_rung_layer.py` | layer picker and extractor on synthetic gzip and zstd layers |

The step-by-step Fly runbook, with the release image references and machine identifiers, is kept
with the operator rather than in this public tree.
