# CometBFT 0.38.25 for the legacy signers (zerone-1, zerone-testnet-1)

Written by an agent: a Claude Code delegate of session 8d4c201e, on
2026-11-06. It follows Yu's words of 2026-11-05, "yes I want CometBFT patched on
zerone-1 ahead of H3" and "yes to observer". This directory is a recipe and a
runbook. Nothing in it has been pushed to a registry or run against a live
machine. Every live step below still needs the operator, and Yu's word for that
step, before it runs.

The two signers run the legacy application `2e37c4c`, built with Go 1.24.13 and
CometBFT v0.38.20 (which reports itself as `0.38.19`). The swap replaces only
the `zeroned` executable inside each signer's current image. Application
source, Cosmos SDK v0.50.15, IBC v8.8.0, the entrypoint, the baked seed
directory, the volume and the machine configuration stay as they are.

## The executable

| | |
|---|---|
| SHA-256 | `d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443` |
| Size | 100,651,816 bytes |
| Source | `2e37c4c86c31e67515d6aa07b6fd406083e012ea` plus `deploy/legacy-observer/observer-dependencies.patch` (SHA-256 `3eb79d1f…`) |
| Toolchain | Go 1.25.14 in `golang@sha256:c268a04d…` (linux/amd64), `CGO_ENABLED=1`, `GOAMD64=v1`, `-trimpath`, `VERSION=dev COMMIT=unknown` |
| Release | the same bytes as `zeroned` in the signed pre-release `zerone-1-observer-20260909.2` |

It is the observer executable, not a new build profile. On 2026-11-06 three
fresh-cache builds on a Colima/Rosetta host, running the unchanged
`observer-build.py` recipe through the wrapper below (the third through this
committed `build-emulated.py`), produced these exact bytes, and every build log
was byte-identical to the release's first build log.
On a native Linux amd64 host, use `observer-build.py` directly, as
`deploy/legacy-observer/README-build.md` describes:

```sh
python3 deploy/legacy-observer/observer-build.py --source-archive "$WORK/legacy-source.tar.gz" \
  --dependency-patch deploy/legacy-observer/observer-dependencies.patch \
  --expected-sha256 d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443 \
  --output "$WORK/signer-binary"
```

On a Mac with Colima (Rosetta), use the wrapper. It changes only the socket and
the native-host check and offers a tmpfs `/tmp`, and it requires the expected
digest:

```sh
DOCKER_CONFIG="$EMPTY_DOCKER_CONFIG" python3 deploy/legacy-signer/build-emulated.py \
  --repository "$PWD" --source-archive "$WORK/legacy-source.tar.gz" \
  --dependency-patch deploy/legacy-observer/observer-dependencies.patch \
  --docker-host "unix://$HOME/.colima/default/docker.sock" --tmpfs-tmp \
  --expected-sha256 d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443 \
  --output "$WORK/signer-binary-emulated"
```

`go version -m` differs from the live binary `94d76a0a…` in the toolchain line,
a `DefaultGODEBUG` line that keeps Go 1.24 behaviour for five settings
(`containermaxprocs=0,decoratemappings=0,tlssha1=1,updatemaxprocs=0,x509sha256skid=0`),
and three linked modules: `github.com/cometbft/cometbft` v0.38.20 → v0.38.25,
`github.com/lib/pq` v1.10.9 → v1.12.0 and `github.com/minio/highwayhash`
v1.0.3 → v1.0.4. `github.com/petermattis/goid` and `github.com/sasha-s/go-deadlock`
move in `go.mod` but are not linked. The other 175 linked modules, including the
app module, the SDK, IBC, IAVL, store and `golang.org/x/text` v0.34.0, are
unchanged. The proposal-coherence gate (`scripts/proposal-coherence-gate.py`)
reports `NOT_AFFECTED` for `2e37c4c`, and for a local commit of `2e37c4c` plus
the patch.

Unicode tables: `unicode.Version` is `15.0.0` in both Go 1.24.13 and Go 1.25.14,
and x/text v0.34.0 compiles `tables15.0.0.go` (norm `15.0.0`) under both. The
NFC, NFKC, case-range and `strconv` printability tables found byte for byte in
both executables are the Unicode 15 ones; the Unicode 17 tables (Go 1.27) are in
neither. Over every code point, `strings.ToLower`, `strings.ToUpper`,
`strconv.Quote`, `fmt %q`, `strconv.IsPrint` and the `unicode.Is*` predicates
give the same results under both toolchains. Go 1.25 adds the `Cn` and `LC`
categories and adds unassigned code points to `unicode.C`; nothing linked into
`zeroned` reads `unicode.C`, `unicode.Other`, `unicode.Categories` or a `\pC`
regexp class, apart from `regexp/syntax` itself and a test-data generator.

## The image

`Dockerfile.swap` starts `FROM` the signer's current image, pinned by digest,
and adds one layer holding `/usr/local/bin/zeroned` (root, 0755). The base
images were deployed on 2026-07-25 a few minutes after `2e37c4c` was committed,
and their binary `94d76a0a…` reproduces from it, so they are taken to come from
`2e37c4c`'s `deploy/mainnet/Dockerfile` and `deploy/testnet/Dockerfile`:
`ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]`, and the entrypoint `exec`s
`zeroned start --home /data/.zeroned --minimum-gas-prices 0.025uzrn` by `PATH`.
The real bases have not been pulled here; `verify-swap-image.py` checks their
binary, entrypoint and seed directory when the image is built.

On an existing volume the entrypoint rewrites only `external_address`,
`minimum-gas-prices` and (zerone-1 only) the gRPC bind address, with the same
lines in both images. It writes the full configuration only when
`config/genesis.json` is missing.

| App | Machine | Volume | Base image (current) | Seed dir |
|---|---|---|---|---|
| `zerone-testnet-1` | `d8d3736f455508` | `vol_vp25peqkw9dw66k4` | `registry.fly.io/zerone-testnet-1@sha256:8c5a859572cea6eda34de062cce81ca82d46d36f7afaf64fabd1cb668bac311d` | `/testnet-seed` |
| `zerone-1` | `8629d6be267178` | `vol_4ojnz571g2zwmmpr` | `registry.fly.io/zerone-1@sha256:ba73fea59838503c9a099887a4604945b4e959a273b2015fd7eaf1baaf3f861a` | `/mainnet-seed` |

These values were read on 2026-11-06 with `fly machines list --json`. Read them
again before using them.

`build-swap-image.sh --app <app> --binary <zeroned> --docker-host unix://<socket> --out <new dir>`
checks the executable's digest and pulls the base by digest. It then builds the
image twice with `--no-cache`, `SOURCE_DATE_EPOCH=1784959094` (the `2e37c4c`
commit time) and `rewrite-timestamp=true` into OCI archives, and requires both
manifest digests to be equal. It loads the first archive under
`registry.fly.io/<app>:cometbft-0.38.25-d6391ea7` and runs `verify-swap-image.py`.
The verifier compares the new image with its base and passes only when:

- the image configuration (entrypoint, command, environment, working directory,
  ports, user, volumes, labels, stop signal) is unchanged;
- the base layers are kept in order and exactly one layer is added, holding only
  `usr/local/bin/zeroned` and its parent directories, with the pinned digest,
  root ownership and mode 0755;
- every other file is the same in name, size, mode, owner and content, including
  the entrypoint and every file in the seed directory (compared, never printed);
- `zeroned` resolves on `PATH` only to `/usr/local/bin/zeroned`;
- `zeroned version --long` reports go1.25.14, CometBFT v0.38.25, SDK v0.50.15
  and ibc-go v8.8.0.

Pulling the base needs a registry login (`fly auth docker`, which stores a
registry credential in `~/.docker/config.json`). The script never pushes. The
manifest digest it prints is the one the push must report. The load check needs
Docker's containerd image store, where an image ID is its manifest digest.

## Configuration under 0.38.25

The swap writes no configuration. `config.toml` and `app.toml` on each volume
were written once by `zeroned init` with the v0.38.20 template (every binary
from February to July 2026 pinned v0.38.20), with zerone's `timeout_commit`
default of 2.521 s and the entrypoint's first-boot edits. zerone-1's blocks come
exactly 30 s apart, so its `config.toml` was edited after that. Read the live
files before the swap rather than assume them.

- `double_sign_check_height` must be `0`. v0.38.25 fixed an off-by-one: with `1`,
  v0.38.20 checks nothing, but v0.38.25 checks the previous block, finds the
  signer's own signature and refuses to start. Values of 2 or more already
  refuse under v0.38.20, so the restarts of 2026-11-06 show the live value is 0
  or 1. Only a read of the file tells which.
- `block_time_tolerance` (new in v0.38.22) is absent from the old template, so
  the default of 60 s applies, and a value of 0 or less is refused. A block whose
  time is 60 s or more ahead of the local clock is rejected, and a committed one
  stops the node. A sole signer stamps blocks from its own clock, so only a
  backward clock step of 60 s or more would trip it. Check NTP and that chain
  time is behind wall-clock time.
- `max_snapshot_chunks` (`[statesync]`, new in v0.38.21) defaults to 100000 and is
  checked only when `statesync.enable` is true. `event_bus_buffer_capacity` (new
  in v0.38.25) defaults to 0, the old unbuffered behaviour.
- No key was removed or renamed and no existing default changed between v0.38.20
  and v0.38.25. `app.toml` belongs to the unchanged SDK.

## Gates before the swap

1. The observer proof has passed (see below), on the same executable digest.
2. A read of each signer's live `config.toml` shows `double_sign_check_height = 0`,
   no non-positive `block_time_tolerance` and `statesync.enable = false`. Record
   the SHA-256 of both files; they must be unchanged after the swap.
3. The signer's clock is NTP-synced, and its latest block time is behind wall-clock
   time.
4. The old image still resolves by digest in the registry, so a rollback is possible.
5. Yu's word for each live step: push, snapshot, update. Testnet goes first, and
   zerone-1 waits for a clean testnet soak.

The observer proof must show, on the same executable, all of the following:

- **Real history, replayed from state.** Re-execute every block from
  `agenttool-seam-v1` (height 527603, the last applied upgrade and the first
  block the `2e37c4c` code ran) up to a recent height. Start from application
  state at 527602 and let CometBFT check `AppHash` and `LastResultsHash` on every
  block. That span holds 278 of zerone-1's 359 transactions (2026-11-06) and
  about 90 fitness epochs. Earlier heights ran under older binaries, so
  replaying them takes those binaries (or a state copy taken at 527602 or earlier).
  A state-synced observer that follows a quiet tip does not meet this gate.
- **The upgrade handler.** If the replay starts below 527603, the
  `agenttool-seam-v1` handler runs under the new toolchain and must give the
  recorded app hash.
- **Following the signer.** After the replay, follow the live signer as a
  zero-power node with identical app hashes for at least 24 hours, through at
  least one fitness-epoch boundary (a height divisible by 10,000).
- **Testnet too.** The same proof, or the swap rehearsal itself, on
  zerone-testnet-1.

## Runbook (testnet first, then zerone-1)

Set `APP`, `ID` (machine), `VOL`, `OLD` (current image by digest) and `RPC`
(`http://<app>.fly.dev:26657`) from the table above. Each step lists what it
must show. Stop at the first step that does not.

1. **Record the before state.**
   `fly machine status $ID -a $APP --display-config > before.json`;
   `curl -s $RPC/status`; `curl -s $RPC/abci_info`.
   Expect: state `started`, `image_ref.digest` equal to `OLD`, services as
   recorded (zerone-1: 26657, 1317, 9090; testnet: 26657, 1317; no 26656),
   `node_info.version` `0.38.19`, `voting_power` 11111 (zerone-1) or 100000
   (testnet), and a height that advances between two reads.
2. **Build and verify the image** (local, no live effect):
   `deploy/legacy-signer/build-swap-image.sh --app $APP --binary <verified zeroned> --docker-host unix://<socket> --out <new dir>`.
   Expect: every check `PASS`, `RESULT PASS`, and one manifest digest `NEW`.
3. **Push** (a live act):
   `docker push registry.fly.io/$APP:cometbft-0.38.25-d6391ea7`.
   Expect: the final line reports `digest: NEW`, and
   `docker buildx imagetools inspect registry.fly.io/$APP@NEW` shows linux/amd64.
   If the digest differs, stop: do not update with anything else.
4. **Snapshot the volume** (a live act):
   `fly volumes snapshots create $VOL -a $APP`, then
   `fly volumes snapshots list $VOL -a $APP`. Expect: a new snapshot id in state `created`.
5. **Update the image only** (a live act):
   `fly machine update $ID -a $APP --image registry.fly.io/$APP@NEW --yes`.
   No other flag. Expect the machine to restart once. On 2026-11-06 a restart
   of zerone-1 paused blocks for 71 s.
6. **Check.**
   - `fly machine status $ID -a $APP --display-config > after.json`. Expect
     `image_ref.digest` = `NEW`, state `started`, and no other difference from
     `before.json` apart from image, instance and timestamp fields: the services
     (still no 26656), mounts, guest, restart policy, environment names and
     `init` must be unchanged.
   - `fly logs -a $APP` from the restart. Expect `[entrypoint] existing volume — resuming`,
     `Version info` with `tendermint_version=0.38.25 block=11 p2p=8`,
     `ABCI Handshake App Info` / `ABCI Replay Blocks` at the stopped height,
     `Completed ABCI Handshake - CometBFT and App are synced`, then
     `finalized block` / `committed state` lines. Any of these means roll back at
     once: `fresh volume — seeding` (the volume did not mount), `found signature from the same key`,
     `panic`, `CONSENSUS FAILURE`, `wrong Block.Header` or `does not match`.
   - `curl -s $RPC/status`. Expect `node_info.version` `0.38.25`, the same
     network, node id and validator address as before, the same voting power,
     `catching_up` false, and the height advancing past the pre-swap height.
     `abci_info.version` stays `dev`, so it cannot tell the binaries apart.
   - `fly ssh console -a $APP -C "sha256sum /usr/local/bin/zeroned"`. Expect `d6391ea7…`.
   - If the observer is running, it keeps following with identical app hashes
     across the swap height.
7. **Soak.** For 60 minutes, check about every minute: version, advancing height,
   and the logs. On testnet, wait at least 24 hours (and Yu's word) before zerone-1.
8. **Record.** Write a dated record under `docs/reports/` with the before and
   after configs, digests, snapshot id, heights, times and checks, as the dev-1
   binary patch did on 2026-11-03.

## Rollback

`fly machine update $ID -a $APP --image $OLD --yes`, with `OLD` the before
image by digest, followed by the same checks with `0.38.19` expected. The local
rehearsal showed the v0.38.20 binary resuming on a home written by v0.38.25.
Do not use `fly releases rollback` or `fly deploy`: either one replaces the
machine configuration with an older release's or a `fly.toml`'s, which can bring
back the public P2P service or change other settings.
Do not restore the volume snapshot onto a signer that has signed past it: an
older `priv_validator_state.json` lets it sign heights it already signed. A
restore is a last resort, used only for a damaged home and only when the chain
has not moved past the snapshot.

## Local rehearsal, 2026-11-06

The real base images cannot be pulled without the registry login, so the
rehearsal used a local stand-in. It has the original `94d76a0a` at
`/usr/local/bin/zeroned`, the `2e37c4c` mainnet entrypoint, a `/mainnet-seed`
directory and the same `ENV`/`EXPOSE`/`ENTRYPOINT`, on the local Go builder's
Debian bookworm instead of `debian:bookworm-slim`. The seed is a throwaway chain
`zerone-swaptest-1`. The swap image came from `build-swap-image.sh --local-test`
(all checks passed; the two no-cache builds gave one manifest digest). Every
container ran with `--network none`, on one volume:

- heights 1–30 on the old image (fresh volume; the entrypoint seeded it);
- heights 31–60 on the swap image ("existing volume — resuming", `0.38.25`);
- heights 61–75 on the old image again (the rollback path);
- heights 76–85 on the swap image again;
- 13 transactions (bank sends and delegations, code 0) included in heights 7–67;
- the final home, with `application.db` removed, replayed from height 1 to the
  same app hash (`F1D8CED0…` at height 85) under both binaries;
- the swap image on a fresh volume seeded and produced blocks;
- the old image then ran on to height 224 with four accepted knowledge claims
  (heights 154–215) on code points that Go 1.27's Unicode 17 tables treat
  differently: U+A7CB (which lowercases to U+0264 there, so it would collide with
  the second claim, U+0264), U+10D50, and U+105D2 U+0307. Replaying
  heights 1–224 from an empty `application.db` gave the original app hash at 224
  (`694EA9AD…`) under both binaries, failed transactions included.

That is a check of the image mechanics and of these code paths, not of
zerone-1's history. The observer gates above cover that.
