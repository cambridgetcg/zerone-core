# zerone-testnet-1 H1 halt runbook

Written 2026-11-06 by an agent (Claude Code, a delegate of host session 8d4c201e).
Nothing in this file has been run against the live chain. It is the order of
operations for the testnet H1 halt that Yu gave the go-ahead for on 2026-11-05.
That go-ahead covers zerone-testnet-1 only, not zerone-1. Every Fly write and
every signed transaction below is an operator act carried out under it; this
file grants nothing by itself. Where a step needs a
separate word, it says so.

Commands assume, from the repository root of a checkout of this branch:

```sh
KIT=deploy/upgrades/consolidation-safety-v1
WORK=/absolute/new/scratch/dir          # every output below is a NEW directory under it
RPC=http://37.16.28.121:26657           # testnet CometBFT RPC (published; P2P is not)
REST=http://37.16.28.121:1317           # testnet REST
APP=zerone-testnet-1
# The machine, its volume and its running image are read at run time, never kept in this public
# repository. The signer app has exactly one machine with one volume:
MACHINE=$(fly machine list -a $APP --json | jq -r '.[0].id')
VOLUME=$(fly machine list -a $APP --json | jq -r '.[0].config.mounts[0].volume')
OPS=~/.zeroned/testnet-ops              # test keyring; signers are named faucet and validator
CLI=$WORK/cli/zeroned                   # the H1 operator CLI built per TESTNET.md
IMAGE=registry.fly.io/zerone-testnet-1@sha256:fdfa9ade6a239c39fb4867203fe8db46c379cd8a99b7b5133b72d61132a4e280   # once G2 holds
# The image running before the halt is the rollback image (G8). Since 2026-11-10 16:07Z it is the
# CometBFT 0.38.25 swap image (2e37c4c + d6391ea7, legacy-signer kit), not the 2026-07-25 release:
LEGACY=registry.fly.io/$APP@$(fly machine list -a $APP --json | jq -r '.[0].image_ref.digest')
```

Never print key material: the only keyring reads below are `keys show -a`
(address) and the sign/vote commands themselves.

## Facts this relies on (checked 2026-11-06 by GET and read-only fly; re-check)

| What | Value | How checked |
| --- | --- | --- |
| Testnet binary | 2e37c4c, app version `dev`. CometBFT 0.38.25 (`d6391ea7…`, `/status` says 0.38.25) since the legacy-signer swap of 2026-11-10 16:07Z; before it 0.38.20 (`/status` said 0.38.19) | RPC `/status`, `/abci_info`; re-read 2026-11-10 |
| Height and cadence | 3,869,6xx at 12:30Z; 2.53 s over the last 100 and 1,000 blocks, 2.67 s over 100,000 | RPC `/header` |
| Governance | voting 120 s, min deposit 10 ZRN (10000000uzrn), max deposit period 300 s, expedited 60 s / 50 ZRN | REST `/cosmos/gov/v1/params/*` |
| Gov authority | `zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89` (same on zerone-1) | REST module account |
| Signers | faucet `zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5` (~998,559 ZRN, the 10-28 seal proposer); validator `zrn1q0ar3f2cswzlemcss4nu82cd40crftd9utnt0e` = `zrnvaloper1q0ar3f2…dzjp69`, sole bonded validator, power 100,000 | repo `deploy/testnet/artifacts/*-address.txt` at 2e37c4c, REST |
| Earlier upgrade proposals | 2 and 3 used `info ""`; H1 refuses that at startup (H1-F2) | REST proposals |
| Pending plan, frozen accounts, pots, pools | none, none, 0, 0 | `pre-h-census.py` PASS at 3,870,270 |
| Fly machine | one machine (`$MACHINE`), lhr, shared-cpu-2x 4 GB, restart on-failure ×10, `init {}` (image entrypoint) | `fly machines list --json` (env values not printed) |
| Current image | since 2026-11-10 the CometBFT 0.38.25 swap image (the release v16 image of 2026-07-25 plus one layer with `d6391ea7…`); `$LEGACY` | same; machine `image_ref` |
| Its entrypoint | 2e37c4c `deploy/testnet/entrypoint.sh`: rewrites `external_address` and `minimum-gas-prices` on every boot, then `exec zeroned start --home $ZERONE_HOME --minimum-gas-prices 0.025uzrn ${EXTRA_START_FLAGS:-}` | source read; **confirm on the live machine** (step 0) |
| Env | keys `EXTERNAL_ADDRESS`, `MONIKER`, `ZERONE_HOME`, `PRIMARY_REGION`, `FLY_PROCESS_GROUP`; `ZERONE_HOME` equals `/data/.zeroned` (the kit image's `--home`); no `EXTRA_START_FLAGS` in machine env (a Fly secret is not visible here) | compared without printing values |
| Services | 26657 and 1317 only; 26656 was removed 2026-11-06 (the node still listens on 26656 inside the VM for 6PN peers) | ledger `zc-signer-p2p-close`, machine config |
| Volume | one volume (`$VOLUME`), 20 GB, encrypted; `/data` 81% full (15.7 of 20.5 GB) | task brief; see "Disk" |

## Disk: H1's migrations fit

On testnet state H1 writes almost nothing: knowledge 5→6 writes one marker;
claiming_pot 1→2 iterates 0 pots and stores one 8-byte counter; liquiditypool
3→4→5 rewrites params and the next-pool counter over 0 pools; x/upgrade writes
the done height and the 40-entry version map and clears the plan; the handler
writes `upgrade_marker_consolidation-safety-v1`. That is under 50 keys and a few
KB of values, under 1 MB with IAVL node overhead, against about 4.8 GB free.
There are no store upgrades and no IAVL or CometBFT database format change
(store v1.1.1, iavl v1.2.2 and cometbft-db v0.14.1 are unchanged by the patch).

The real limit is ordinary growth. A rough reading: the 10-28 drill copy left
about 24.9 GB free on a 40 GB restore (about 15.8 GiB used), and 11-06 shows
15.7 GiB used, so usage looks flat, probably held by pruning. That is two
points on different filesystems, so check `df -h /data` on the legacy machine
shortly before H (the legacy image has a shell; `fly ssh console -a $APP -C 'df -h /data'`).
**After the swap the H1 image is `scratch` with no shell**: `fly ssh console`
and `df` stop working. Watch the volume through Fly's volume metrics from then on.

## The runtime smoke: can it run inside a disposable copy?

Not as written. `runtime_smoke.py` drives a local Docker daemon
(`docker run --network=none --read-only …`, `docker inspect`) against a drilled
home on the same host. A Fly machine has neither Docker nor a network-none mode.
It therefore cannot run inside the disposable Fly copy, and it must not be
skipped either: `--execute` refuses without its receipt.

What works without changing the kit's contract:

1. Run the fresh state-copy drill (gate G3) in a disposable Fly app, exactly as
   drill B did on 2026-10-28, with helpers rebuilt from this branch.
2. Bring the drilled home back to the Mac. It is keyless by construction
   (`working_inventory` admits only `.h1-disposable-copy`, `data/application.db`
   and `data/upgrade-info.json`); testnet's application.db was 8.5 GB on 10-28.
   Copy `h1-drill/` (receipt plus `home/`) out of the disposable app before it is
   destroyed, and check the inventory hashes against `drill.json` on arrival.
3. Run `runtime_smoke.py --execute` on the Mac in colima against the exact image
   digest (loaded so that `name@sha256:<digest>` resolves locally, as the
   2026-10-28 start check did). Needs about 20 GB free on the Mac; the Studio
   had 11-18 GiB free on 2026-11-06, so free space first. The drilled home leaves
   the disposable app through `fly ssh sftp get` of a tar made on its volume, and
   is checked against `drill.json` on arrival. It is keyless only if G3's scrub
   receipt says so; never copy a home whose scrub did not pass.

If bringing the drilled home back is not acceptable, the halt stays gated. The
alternative is a Fly-native smoke in the same disposable app: start a second
machine from `$IMAGE` by digest on the drilled volume with
`start --home <drilled home> --with-comet=false --transport socket --address unix:///tmp/h1-abci.sock --grpc.enable=false --api.enable=false --minimum-gas-prices 0.025uzrn`,
no services, no IPs, and observe `Waiting for new connection...` and a running
machine for 10 s or more. That also proves Fly can pull the digest. It needs a
kit change first, because readiness only accepts the Docker receipt; it is not
in this branch.

Already done locally (2026-11-06, evidence in the host's lane folder): the full
offline runner with the patched helpers (87 tests, 0 skipped; synthetic old→H1
migration to K6/P2/L5/V1 and completed-lineage restarts), and two starts of the
built image by `name@sha256:fdfa9ade6a23…` with `--network=none`: on a fresh keyless
home (`lineage=uninitialized-bootstrap`) and on a clone of the synthetic post-H1
fixture (`lineage=h1-completed`), each with the ABCI listener up for 10 s. These
are not the runtime smoke: neither is real testnet state.

## Step 0. Gates before proposing

Each gate must hold. Record the evidence beside the receipts.

- **G1 Image.** Built twice through `provenance.py --allow-emulated` on
  2026-11-06: binary `df70456f…c592` (= the pinned cross-build), OCI manifest
  `sha256:fdfa9ade6a239c39fb4867203fe8db46c379cd8a99b7b5133b72d61132a4e280`, archive `3a046e2f…3acd`. Done.
- **G2 Image on Fly's registry, by digest.** Publishing to
  `registry.fly.io/zerone-testnet-1` is a push to the app's own private
  registry; it rides the halt yes only if the host reads it that way and says so
  in the record. Keep the registry login in a throwaway `DOCKER_CONFIG`. Use a transfer that keeps the
  OCI manifest digest (`skopeo copy --preserve-digests oci-archive:$WORK/image-build-emulated/image.oci.tar docker://registry.fly.io/zerone-testnet-1:h1-c25`,
  with a short-lived Fly registry token), then read the manifest back and compare:
  `skopeo inspect --raw docker://$IMAGE | shasum -a 256` must equal the receipt's
  `image_digest`. skopeo is not installed on the Studio today.
- **G3 Fresh drill.** `state-copy-drill.py` PASS on a key-scrubbed copy of
  current testnet state, with helpers built from this branch (the H1 helper must
  carry `build_source_sha256 = 7fee6503…0e7f`). `--execute` refuses a copy more
  than 35,000 blocks below H. The 2026-10-28 drill tooling (`drills-v3`) was in
  `/private/tmp` and has been swept; only `RUNBOOK-v3.md` and the evidence remain
  in `~/.kingdom/motifs/carrier/drills-20261028/`. The tooling must be restored or
  rebuilt, and rebuilt tooling is reviewed before use. A fresh Fly copy is a
  Fly write and a new paid resource (an app and a 20-40 GB volume for a few
  hours); TESTNET.md says fetching a copy needs its own yes, so the host
  decides whether the halt yes covers it and says so in the record.

  The restored volume holds the live testnet validator's
  `priv_validator_key.json`, `node_key.json` and `keyring-test` (drill B
  inventory, 2026-10-28), and testnet has one bonded validator holding all bonded
  stake. A copy that starts `zeroned` with that key signs heights the live chain
  already signed; if that evidence reached the live node over 6PN
  (`zerone-testnet-1.internal:26656` still listens), it would tombstone the only
  validator and halt testnet for good. So, every time, with a kept receipt:

  1. Only a drill image whose entrypoint never starts `zeroned` mounts the
     restored volume. Never the legacy image (its entrypoint `exec`s `zeroned
     start` on an existing volume) and never the H1 image.
  2. Before any `zeroned` process runs, the scrub passes: new consensus and node
     keys replace the copied ones (the consensus pubkey must differ from the
     live one), keyrings removed, key-file paths and `priv_validator_laddr`
     forced, P2P and RPC on loopback, and no peers, seeds or address book.
  3. The drill app is on its own custom 6PN network (`fly apps create --network
     <drill-net>`), with no IPs and no services, so it cannot reach the live node.
  4. Afterwards destroy the machine, the volume and the app, and confirm with
     `fly apps list`, `fly volumes list` and the snapshot list that nothing is
     left.

  The H2-recovery rehearsal below uses the same rules.
- **G3b Fly boots this digest.** In the G3 drill app, before proposing, run the
  exact `$IMAGE` digest keyless on a throwaway volume: `version`, and a
  `start --with-comet=false` on a fresh home. The H1 binary panics at init with
  `$HOME is not defined` when HOME is unset (65c19cd8 app/app.go:303-305), the
  image is scratch with no `/etc/passwd`, and the kit's own smoke injects
  `HOME=/tmp`, so whether Fly sets HOME for this image, and whether it unpacks
  its uncompressed OCI layer, is unchecked until this runs. If HOME is unset, the
  step-5 machine update must add `--env HOME=/data` (or the image must set it).
- **G3c Rollback rehearsed.** On the G3 copy, halted at H by the H1 plan, run
  the legacy rollback image with `--unsafe-skip-upgrades H` and expect block H
  committed and the plan cleared. Nothing in the kit or the 10-28 drills has
  exercised the skip yet.
- **G4 Runtime smoke.** `runtime_smoke.py --execute` receipt for `$IMAGE` and
  the G3 drill (see the section above).
- **G5 Census.** `python3 $KIT/pre-h-census.py --rpc $RPC --rest $REST --output $WORK/census-0`
  prints `PASS`. The run at H-1 is mandatory, not best effort: the
  governance-quiet check sees only x/gov proposals open at the census height, so
  a 120 s proposal can pass between runs, and zerone_gov proposals are not read
  at all. Check zerone_gov by hand at H-1 until the census reads it.
- **G6 CLI.** `$CLI version --long --output json --home $(mktemp -d)` shows
  `h1-testnet-65c19cd8-cometbft-v0.38.25`, commit 65c19cd8 and
  `github.com/cometbft/cometbft@v0.38.25` in `build_deps`.
- **G7 Engine order.** Recommended: the binary lane swaps testnet from 2e37c4c
  (0.38.20) to the patched legacy build (0.38.25) and it soaks first, so the H1
  halt changes only the application. If H1 goes first, the halt changes the
  application and the engine together; the review's local evidence covers that
  swap (same block and P2P protocol, same replayed app hash), but testnet has not.
- **G8 Rollback image.** Write down the exact legacy image digest running at H
  (`$LEGACY`, read just before; since 2026-11-10 the patched legacy image, so a
  rollback keeps CometBFT 0.38.25). Confirm on the live machine that its
  entrypoint honours `EXTRA_START_FLAGS`:
  `fly ssh console -a $APP -C "grep -n EXTRA_START_FLAGS /usr/local/bin/entrypoint.sh"`.
  (`fly logs --no-tail` returns only the last 100 lines, about 30 s, so it no
  longer shows a past restart.) Check `fly secrets list -a $APP` names
  only: an `EXTRA_START_FLAGS` secret would already be in force.
  flyctl 0.4.x writes an `--image <repo>@sha256:X` reference as
  `<repo>@sha256:X@sha256:X` (seen 2026-11-07 and 2026-11-10); the legacy-signer
  swap passed a tag instead and checked `image_ref.digest`. Whether a machine
  starts from the doubled reference was not tried.
- **G9 Saved config.** `fly machines list -a $APP --json > $WORK/machines-before.json`
  (mode 0600; it holds env values, never print or commit it).
- **G10 Shared ground.** Claim the testnet halt on the agenttool-collab board,
  and check no other session is changing testnet.

## Step 1. Choose H and draft the unsigned proposal

```sh
python3 $KIT/h1-proposal.py draft --chain-id zerone-testnet-1 --rpc $RPC --rest $REST \
  --image "$IMAGE" --proposer zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5 --output $WORK/draft
```

The tool GETs the live cadence and gov params, refuses if the deposit, voting
period or authority differ from the pins or a plan is pending, and picks
H = latest + ⌈(600 s to broadcast + 120 s vote + 900 s after passing) / fastest
sampled block interval⌉ + 120 blocks. At 2.53 s that is about 760 blocks, about
32 minutes. It sets the tx's `timeout_height` to H − ⌈(120 + 900) / fastest⌉
(about 400 blocks before H), so the chain itself refuses the submission if it
lands too late. Pass `--height` to choose H yourself; anything earlier than the
safe minimum is refused. `draft.json` records the inputs, Info, its SHA-256 and
an ETA. Info is exactly the kit's `plan_info()`:
`{"commit":"65c19cd8b00bdfff9b80705b776fd0d49719398a","image":"<IMAGE>"}`.

Before signing, check the census again (G5) and that the current height is
still well below `timeout_height`.

## Step 2. Sign, broadcast, vote (the vote must land within 120 s)

```sh
$CLI keys show faucet -a --keyring-backend test --home $OPS     # must print the proposer address
$CLI tx sign $WORK/draft/unsigned-tx.json --from faucet --keyring-backend test --home $OPS \
  --chain-id zerone-testnet-1 --node ${RPC/http/tcp} --output-document $WORK/draft/signed-tx.json
$CLI tx broadcast $WORK/draft/signed-tx.json --node ${RPC/http/tcp} --broadcast-mode sync --output json
```

Take the `txhash`, wait for inclusion (`$CLI query tx <hash> --node … --output json`),
and read the proposal ID from that tx's `submit_proposal` event. Never use
"the latest proposal" and never resubmit after an ambiguous result: reconcile the
hash first. Then vote at once with the bonded validator:

```sh
$CLI tx gov vote <ID> yes --from validator --keyring-backend test --home $OPS \
  --chain-id zerone-testnet-1 --node ${RPC/http/tcp} --gas 300000 --fees 300000uzrn --yes --output json
```

Prepare this command before broadcasting. The voting period is 120 s from the
submission block; the 10 ZRN deposit puts the proposal straight into voting.

## Step 3. After it passes: check the committed plan byte for byte

```sh
python3 $KIT/h1-proposal.py check-plan --rest $REST --image "$IMAGE" --height <H>
python3 $KIT/pre-h-census.py --rpc $RPC --rest $REST --image "$IMAGE" --height <H> \
  --allow-proposal <ID> --output $WORK/census-armed
```

Both must pass. If the plan differs in any byte, do not let H1 start at H.
With enough blocks left, a `MsgCancelUpgrade` proposal (same gov authority,
same 120 s vote) clears the plan before H; otherwise let legacy halt and follow
"Rollback" (legacy plus skip). Then draft a new proposal.

Re-run the census as close to H−1 as you can (`--output $WORK/census-h-1`).
A FAIL after the proposal passed is a stop condition: at H, roll back instead of
swapping.

## Step 4. Watch for the halt

With every receipt in hand, the ship script's resume mode only queries and
polls; it opens no key and submits nothing:

```sh
SHIP_ACK=zerone-testnet-1 bash $KIT/ship-consolidation-safety.sh --execute \
  --chain-id zerone-testnet-1 --zeroned $CLI --ops-home $OPS --operator validator \
  --node ${RPC/http/tcp} --image "$IMAGE" --fly-machine $MACHINE --height <H> \
  --resume-proposal <ID> --image-receipt $WORK/image-build-emulated/provenance.json \
  --drill-receipt <G3>/drill.json --runtime-receipt <G4>/runtime.json --image-available
```

It requires the exact PASSED proposal and committed plan, then four equal polls
with ABCI at H−1 and Comet at H−1 or H. Until G4 exists the script refuses to
run; the manual equivalent is to poll RPC `/status` and `/abci_info` every 8 s
and wait for four equal samples with ABCI `last_block_height` = H−1 and Comet
`latest_block_height` ∈ {H−1, H}. A dark RPC is never a halt. At the halt the legacy node logs
`UPGRADE "consolidation-safety-v1" NEEDED` and `CONSENSUS FAILURE`, stays up and
keeps answering RPC. Block H is already decided by consensus; only its execution
is missing. The script prints the swap command and stops.

## Step 5. Swap the image

1. Record pre-swap evidence (`$WORK/pre-swap/`): `/abci_info` (height H−1 and
   its app hash), `/status`, `/block?height=<H-1>`, `/block?height=<H>` if it is
   served, REST module versions and `/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn`
   with header `x-cosmos-block-height: <H-1>`.
2. Snapshot the frozen volume: `fly volumes snapshots create $VOLUME -a $APP`,
   and wait until `fly volumes snapshots list $VOLUME -a $APP` shows it.
3. Swap in place, keeping the volume, services, env and restart policy, as the
   2026-11-06 P2P change did:

   ```sh
   fly machine update $MACHINE -a $APP --image "$IMAGE" --yes
   ```

4. Watch `fly logs -a $APP` for, in order: `verified consolidation startup
   lineage lineage=h1-pending`; `applying upgrade "consolidation-safety-v1" at
   height <H>`; then new heights. The H1 image ignores `EXTRA_START_FLAGS`,
   `MONIKER` and `EXTERNAL_ADDRESS`; the volume's `config.toml` and `app.toml`
   keep what the legacy entrypoint last wrote (RPC on 0.0.0.0:26657, REST on
   0.0.0.0:1317, the external address, minimum gas price 0.025uzrn).

A panic at startup (`refusing unsafe consolidation lineage at startup: …`) means
H1 will not run on this state. Fly restarts it up to 10 times and then stops it.
Go to "Rollback".

## Step 6. Checks after the swap

| Check | Expect |
| --- | --- |
| H1 handler ran | REST `/cosmos/upgrade/v1beta1/applied_plan/consolidation-safety-v1` → `height` = H; `current_plan` → null; RPC `abci_query` path `"/store/knowledge/subspace"` data `0x7f01` lists `upgrade_marker_consolidation-safety-v1` = `migrated` |
| Module versions | knowledge 6, claiming_pot 2, liquiditypool 5, vesting_rewards 1; the other 36 unchanged from the census |
| Engine and binary | RPC `/status` `node_info.version` = `0.38.25`; `/abci_info` `version` = `h1-testnet-65c19cd8-cometbft-v0.38.25` |
| App hash continuity | `/block?height=<H>` `header.app_hash` = the legacy app hash recorded at H−1, and its `last_block_id` = block H−1's ID; `/block?height=<H+1>` `header.app_hash` = `/abci_info` app hash once H is the last committed height |
| Blocks resume | height passes H+10 at about 2.5 s; validator still bonded with power 100,000 |
| New code answers | REST `/zerone/knowledge/v1/open_questions` → 200 (it is 404 on legacy today) |
| Params migrated | liquiditypool `protocolFeeBps` 0, `maxPools` 16 |
| Supply | supply at H minus supply at H−1 equals the usual per-block change (compare H−2→H−1); no migration mint |
| Restart once | after H+20, `fly machine restart $MACHINE -a $APP`; logs `lineage=h1-completed` and blocks resume (the legacy halt packet on disk names H1 at the done height, which the wall accepts) |

Keep every response under `$WORK/post-swap/`, then record the outcome in the
decision ledger (`zc-h1-testnet-bar`) and release the board claim.

## Rollback

**Only while H1 has not committed H**: `/abci_info` still says H−1, either
because the H1 image refused to start or because you stopped before swapping.

```sh
fly machine update $MACHINE -a $APP --image "$LEGACY" --env EXTRA_START_FLAGS="--unsafe-skip-upgrades <H>" --yes
```

The legacy entrypoint appends `EXTRA_START_FLAGS` to `zeroned start`, so the
legacy binary executes block H without the upgrade and clears the plan. Then:

- H1 has not happened. The plan is cleared and no done height is written, so the
  same name can be proposed again at a new height (a new proposal, a new 120 s
  vote, a new halt).
- `data/upgrade-info.json` from this halt stays on disk naming H. That is
  harmless: legacy overwrites it at the next halt, and H1 then requires it to
  equal the new committed plan at latest+1.
- Clear the flag afterwards (`fly machine update … --env EXTRA_START_FLAGS= --yes`)
  so it never rides into a later restart unnoticed.
- If the rollback image is the unpatched legacy (0.38.20), the engine goes back
  to the vulnerable version; P2P is not published, so exposure stays on 6PN.

**After H1 has committed H, there is no rollback.** Legacy has no handler for
`consolidation-safety-v1`, so x/upgrade refuses to run it on that state, and the
app hash would diverge anyway. Repair forward with a new named upgrade. Restore
the step-5 snapshot only if the volume itself is damaged and only while no block
after H exists: a sole validator restored behind blocks it already signed would
sign a second, different chain at those heights.

## If H2 later fails (H1-F1)

After testnet runs H1, a later H2 halt puts this binary in a narrow spot (README
constraint 9). When the H1 binary halts at H2 it writes `data/upgrade-info.json`
naming `founder-renunciation-v1`. If the H2 binary then fails before committing
H2, restarting H1 with `--unsafe-skip-upgrades <H2>` panics with `local
upgrade-info.json conflicts with completed H1 height <H1>`. Recovery:

1. Confirm H2 did not commit: `/abci_info` says H2−1. Read REST `current_plan`
   and confirm it equals the packet (`name`, `height`, `info` byte for byte) and
   that its height is latest+1. If anything else is true, stop and ask.
2. Snapshot the volume. Stop the machine (`fly machine stop $MACHINE -a $APP`).
3. Move the packet aside; never delete it. The H1 image has no shell, so run a
   pinned tools image on the same machine once, for example:
   `fly machine update $MACHINE -a $APP --image <busybox@sha256:…> --entrypoint /bin/sh --command "-c 'mv /data/.zeroned/data/upgrade-info.json /data/.zeroned/data/upgrade-info.json.h2-halt-<H2> && sync && sleep 3600'" --yes`,
   and check `fly logs` shows it exited the `mv` cleanly.
4. Start H1 with the skip as explicit argv and no entrypoint override:
   `fly machine update $MACHINE -a $APP --image "$IMAGE" --entrypoint /usr/local/bin/zeroned --command "start --home /data/.zeroned --minimum-gas-prices 0.025uzrn --unsafe-skip-upgrades <H2>" --yes`
   (an explicit entrypoint: `--entrypoint ""` may not clear step 3's `/bin/sh`
   override), then `fly machine status $MACHINE -a $APP -d` must show
   `init.entrypoint` `["/usr/local/bin/zeroned"]` and `init.cmd` equal to that argv.
5. Expect `lineage=h1-completed`, block H2 committed by H1, `current_plan` null.
   At the next planned restart, drop the skip by restoring the default command.

A chain born on H1 (native lineage) refuses any packet, so step 3 always
applies there. The H2 source's docs point to a "signed ceremony runbook" that
does not exist; this is the H1 half of it. Rehearse steps 1-5 on a disposable
testnet copy, under G3's key rules, after testnet H1 and before any H2 proposal.
