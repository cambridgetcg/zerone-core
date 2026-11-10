# CometBFT 0.38.25 for the legacy signers (zerone-1, zerone-testnet-1)

Written by an agent: a Claude Code delegate of session 8d4c201e, on
2026-11-06. It follows Yu's words of 2026-11-05, "yes I want CometBFT patched on
zerone-1 ahead of H3" and "yes to observer". This directory is a recipe and a
runbook. Its live steps are run by the host session under those words, testnet
first, and each is recorded in the dated report the runbook ends with.

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

**The toolchain is a choice, stated here so it is not mistaken for part of the
patch.** CometBFT 0.38.25 and the module bumps it forces (lib/pq, highwayhash;
goid and go-deadlock in `go.mod` only) are the patch. Go 1.25.14 instead of the
live binary's Go 1.24.13 is a second change. It was chosen, by the host agent
(session 8d4c201e), for three reasons. It is the exact signed and reviewed
observer executable, and that executable is the one the zerone-1-replay
observer is proving against real history. Go 1.24 has had no security releases
since Go 1.26 shipped. And the equivalence evidence below found no difference
on the consensus path. The alternative, a Go 1.24.13 build of `2e37c4c` plus the
same patch, would change only CometBFT, but it would be a new executable with no
signature, no review and no replay proof, so each of those would have to be done
again for it. Yu can choose that alternative; nothing below depends on the
toolchain except the pinned digest.

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

The signer images bake the validator's `priv_validator_key.json` and
`node_key.json` in their own layers (`/mainnet-seed`, `/testnet-seed`). So the
swap image is assembled at the registry, and nothing here pulls, saves, loads
or runs a signer image, or downloads any of its layers.

`build-swap-image.sh` runs `crane append` (go-containerregistry v0.20.2): it
reads the base's manifest and config (JSON), uploads one new layer holding
`/usr/local/bin/zeroned` (root, 0755; `swap-layer.py layer` writes it
byte-for-byte reproducibly), and pushes a manifest that references the base
layers by digest. It then fetches back the new manifest, the new config and the
new layer (its own), and `swap-layer.py check` passes only when:

- the base layer descriptors are kept, in order and unchanged, and exactly one is added;
- the added descriptor's blob decompresses to the layer tar, which holds only
  `usr/local/bin/zeroned` (the pinned digest) and its parent directories;
- the config equals the base's apart from one added `diff_id` and one added
  history entry (entrypoint, command, environment, ports, labels, everything else
  unchanged), and the manifest's other fields are unchanged;
- crane's request log (method and path only) shows no GET of a base layer blob:
  only existence checks (HEAD), the base config, and the two uploads.

Because every base layer is kept by digest, every other file in the image,
the entrypoint and the seed directory included, is unchanged without being
read. The entrypoint `exec`s `zeroned start` by `PATH`, and the new layer puts
the patched binary at the same `/usr/local/bin/zeroned`.

| App | Seed dir |
|---|---|
| `zerone-testnet-1` | `/testnet-seed` |
| `zerone-1` | `/mainnet-seed` |

The machine, its volume and its current image are not written here: read them
at the time of the swap with `fly machine list -a $APP --json` (`id`,
`config.mounts[0].volume`, `image_ref`), and pass the image to
`build-swap-image.sh --base`. The operator's own record keeps them.

The registry login is a credential that can pull every layer, the key layers
included. Keep it in a throwaway directory, never `~/.docker`:
`export DOCKER_CONFIG=$(mktemp -d)`, `fly auth docker`, run the script, then
remove the directory. `fly auth docker` mints a token for the apps that exist
when it runs; refresh it if the registry answers `NAME_UNKNOWN`.

Rehearsed on 2026-11-06 against a local registry holding a stand-in signer
image with the real shape (OCI manifest, zstd layers, a throwaway seed): two
runs gave the same manifest digest, every check passed, the request log showed
no base layer GET, nine tampered variants (environment, entrypoint, an extra,
swapped or reordered layer, rewritten history, a wrong blob, a fetched base
layer, a dropped diff_id) were each refused, and the assembled image resolved
`zeroned` to `d6391ea7…` (root, 0755) with the entrypoint unchanged.

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
5. Yu's word covers the swap ("yes I want CometBFT patched on zerone-1 ahead of
   H3", 2026-11-05). Testnet goes first, and zerone-1 waits for a clean testnet
   soak of at least 24 hours.
6. The testnet soak does not overlap the testnet H1 halt, and the H1 image
   carries CometBFT 0.38.25 too (review item cometbft-3), so the halt cannot
   bring the old engine back.

Vulnerability scope: govulncheck with the 2026-11-05 database still reports 61
findings on this executable, every one of which is also on the live binary (89).
The September review covered the observer's way of running. A signer also
publishes RPC (26657), REST (1317) and, on zerone-1, gRPC (9090); moving those
reads off the signer is separate work.

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

Set `APP` and `RPC` (`http://<app>.fly.dev:26657`), and read `ID` (machine), `VOL`
and `OLD` (current image, `registry.fly.io/$APP@sha256:<digest>`) as above. Each step lists what it
must show. Stop at the first step that does not.

1. **Record the before state.**
   `fly machine status $ID -a $APP --display-config > before.json`;
   `curl -s $RPC/status`; `curl -s $RPC/abci_info`.
   Expect: state `started`, `image_ref.digest` equal to `OLD`, services as
   recorded (zerone-1: 26657, 1317, 9090; testnet: 26657, 1317; no 26656),
   `node_info.version` `0.38.19`, `voting_power` 11111 (zerone-1) or 100000
   (testnet), and a height that advances between two reads.
2. **Assemble and check the image at the registry** (a push to the app's own
   registry; nothing running changes):
   `DOCKER_CONFIG=<throwaway> CRANE=<crane v0.20.2> deploy/legacy-signer/build-swap-image.sh --app $APP --base $OLD --binary <verified zeroned> --out <new dir>`.
   Expect: every check `PASS`, `RESULT PASS`, and one manifest digest `NEW` in
   `manifest-digest.txt`. Run it a second time into another new directory: it
   must print the same `NEW`.
3. **Keep the rollback image referenced:** `crane tag $OLD rollback-94d76a0a`
   (with the same `DOCKER_CONFIG`), so the old manifest keeps a tag through the
   soak. `crane digest registry.fly.io/$APP:rollback-94d76a0a` must print `OLD`'s digest.
4. **Snapshot the volume** (a live act; a snapshot is billed at $0.08/GB-month
   past the free 10 GB):
   `fly volumes snapshots create $VOL -a $APP`, then
   `fly volumes snapshots list $VOL -a $APP`. Expect: a new snapshot id in state `created`.
5. **Update the image only** (a live act). First start streaming the logs into
   a file, `fly logs -a $APP > swap-logs.txt &`: `fly logs --no-tail` returns
   only the last 100 lines (about 30 s on testnet), so the startup lines step 6
   needs are gone within a minute. Run the update from an empty directory, so
   no stray `fly.toml` is read (`2e37c4c`'s `deploy/mainnet/fly.toml` still
   declares the public 26656 service), and first without `--yes` (flyctl then
   prints the diff and refuses):
   `cd "$(mktemp -d)" && fly machine update $ID -a $APP --image registry.fly.io/$APP:cometbft-0.38.25-d6391ea7`.
   Pass the tag, not `@sha256:`: flyctl 0.4.x appends the resolved digest, so a
   digest reference comes out doubled (`…@sha256:X@sha256:X`). Read flyctl's
   config diff: only the image may change, to the tag pinned at `NEW`. Then
   confirm with `--yes`. Expect the machine to restart once. On 2026-11-06 a
   restart of zerone-1 paused blocks for 71 s; the testnet swap on 2026-11-10
   paused them for 13.3 s.
6. **Check.**
   - `fly machine status $ID -a $APP --display-config > after.json`. Expect
     `image_ref.digest` = `NEW`, state `started`, and no other difference from
     `before.json` apart from image, instance and timestamp fields: the services
     (still no 26656), mounts, guest, restart policy, environment names and
     `init` must be unchanged.
   - `swap-logs.txt` from the restart. Expect `[entrypoint] existing volume — resuming`,
     a `Version info` line matching `tendermint_version=0.38.25` (the real line
     reads `Version info abci=2.0.0 block=11 commit_hash= module=server p2p=8 tendermint_version=0.38.25`),
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
8. **Record.** Write a dated record under `docs/reports/` with the heights,
   times, versions, the executable and configuration-file digests, the config
   diff and the checks, as the dev-1 binary patch did on 2026-11-03. This
   repository is public, so the machine and volume IDs, the image digests and
   the snapshot id stay in the operator's own record, not in the report.

## Rollback

First `crane digest registry.fly.io/$APP:rollback-94d76a0a` must still print
`OLD`'s digest. Then, from an empty directory,
`fly machine update $ID -a $APP --image registry.fly.io/$APP:rollback-94d76a0a --yes`
(the tag, for the reason in step 5), check that `image_ref.digest` is `OLD`'s,
and run the same checks with `0.38.19` expected. The local
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
`zerone-swaptest-1`. The swap image came from the kit's first design, a local
`FROM <base>` build adding the same `/usr/local/bin/zeroned` (two no-cache builds
gave one manifest digest). That design pulled the base, key layers included, so
it was replaced the same day by the registry-side assembly above; the binary,
its path and the inherited entrypoint are the same in both. Every
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
