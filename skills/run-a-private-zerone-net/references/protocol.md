# zerone-private-net/v1: packets, checks and sources

## Files that travel

The protocol's packets are JSON and hold no secret. All but the invitation
contain LAN addresses. Kit folders also carry the skill, `zeroned` and a
generated `START-HERE.md`, which names the coordinator's address. All of them
move only by courier.

**Invitation** (`invite-<label>.json`, coordinator → one Mac, round 0)

```
format "zerone-private-net-invite/v1", protocol, chain_id, label, role, stake_zrn,
coordinator, binary_sha256, platform, tool_sha256, created_at
```

**Seat packet** (`seat-<label>.json`, seat → coordinator, round 1)

```
format            "zerone-private-net-seat/v1"
chain_id          zerone-rehearsal-{lan,private,local}-<suffix>
label             seat label, also the moniker
role              "validator" | "full"
node_id           CometBFT node id (40 hex)
consensus_pubkey  {"@type", "key"} from `comet show-validator`
p2p_endpoint      "<IPv4 held by this Mac>:<port>"
binary_sha256     sha256 of the zeroned this seat runs
tool_sha256       sha256 of the private-net.py this seat runs
created_at        UTC
validators only:
operator_address  zrn1… (test keyring, key name "operator")
stake_uzrn        self-delegation (default 1,000 ZRN)
balance_uzrn      stake + 1,000 ZRN float
gentx             one signed MsgCreateValidator, memo "<node_id>@<p2p_endpoint>"
```

**Network packet** (`network-<chain>.json`, coordinator → seats, round 2). It
travels with `genesis-<chain>.json` and holds:

- `format`, `chain_id`, `profile`, `genesis_sha256`, `genesis_time`;
- `binary` (sha256 and platform) and `tool_sha256`;
- `seats[]`, each with label, role, node_id, consensus_pubkey, p2p_endpoint,
  operator_address, stake_uzrn and the sha256 of its canonical seat packet;
- `coordinator`, `created_at`, `rules`.

**Status report** (`status-<label>.json`) holds:

- chain id, genesis sha256, node id, height, peers and validator set;
- `blocks`: the block hash and app hash for the last 20 heights, every 100th
  height of the last 2,000, and `--from-height H..H+19` if asked.

`compare` fails unless the reports come from at least two distinct seats, chain
and genesis agree, the seats see one validator set, and at least one height is
shared, with identical hashes at every shared height.

**Courier folders** (written by `kit` outside the root, for the owner to AirDrop):

- round 0, `zerone-kit-<label>/`: the skill folder (exactly `SKILL.md`,
  `references/protocol.md` and the two scripts), the pinned `zeroned`,
  `invite-<label>.json` and a generated `START-HERE.md` with that seat's
  commands and the coordinator's address;
- round 2, `zerone-round2-<label>/`: `genesis-<chain>.json`,
  `network-<chain>.json` and a generated `NEXT.md`. The network packet carries
  the coordinator's own `genesis_sha256`, and the genesis can be hashed, so
  nothing in the folder checks the coordinator. That check is the sha256 the
  owner relays in their own message, which `NEXT.md` tells the seat never to
  take from the folder.

The generated pages spell out every command with absolute paths, because an
agent's shell may keep no function or working directory between calls.

`receive` takes these folders, or single files, into the inbox.

## Checks, by step

| Step | Refuses when |
|---|---|
| every step | socket binds or file access denied by an agent sandbox (named as such); chain id outside `zerone-rehearsal-{lan,private,local}-*`; root not owner-only, inside the home folder (case-insensitive, so iCloud and TCC folders are excluded too) or inside a git tree; any `ZERONED_*` variable is stripped before every zeroned call |
| `build` | not darwin-arm64; no Go; checkout dirty or not at the commit; the binary does not stamp `vcs.revision=<commit>`, `vcs.modified=false` |
| `binary` | no expected hash (`--invite` or `--sha256`); sha256 differs; the binary does not run on this Mac; a different binary is already pinned; `bin/zeroned` and `bin/zeroned.json` do not exist together |
| `invite` | a validator without a positive stake; that seat's invitation already exists |
| `kit` | the folder exists or is inside the root (compared case-insensitively); it cannot be written (the refusal says to choose another `--out`); round 0: no invitation for the seat, the seat is the coordinator, the coordinator has not seated yet, or the pinned binary or this tool is not the one the invitation names; the copied `zeroned` or `private-net.py` does not hash to the invitation's values; round 2: no genesis or network packet yet, the seat is the coordinator or not in the packet, or the outbox genesis does not hash to the packet's value. A refused kit leaves no folder behind |
| `receive` | a source inside the root, or inside a git checkout other than the home folder; a name outside the protocol's (`invite-`, `seat-`, `status-`, `genesis-` or `network-` JSON, `zeroned`, `START-HERE.md`, `NEXT.md`, the skill folder); a symlink anywhere, `.DS_Store` and `__pycache__` included; a JSON file of the wrong format, or a genesis for another chain; a skill folder holding anything beyond `SKILL.md`, `references/protocol.md` and the two scripts, or lacking `SKILL.md` or the tool; an oversized file; the inbox holding a different file of a name that pins the chain (every such name is listed at once). A different status report, `START-HERE.md` or `NEXT.md` replaces the old copy. Every check runs before anything moves |
| `seat` | invitation's tool or binary hash differs from this Mac's; a flag disagrees with the invitation; seat home exists; P2P port outside 1024-65534 (RPC takes the next port); IP is not an IPv4 literal held by this Mac (loopback, as `127.0.0.1`, only on `zerone-rehearsal-local-*` chains); a validator without a positive stake |
| `genesis` | a packet has the wrong format, chain, binary or tool; a gentx memo, stake, consensus key or validator address does not match its packet; duplicate labels, node ids, endpoints, consensus keys or operator addresses; no validators; the coordinator's own packet is missing; an earlier attempt left its coordinator home, gentx directory, or outbox genesis or network file (all are named, to be moved aside only if no seat has installed); collect-gentxs drops a gentx; `genesis validate` fails; a JSON float appears (the tool rewrites genesis, so floats are refused rather than re-encoded) |
| `install` | on a `-lan-` or `-private-` chain, no `--expect-genesis-sha256`; everything listed under "What install verifies" in SKILL.md, plus a `genesis_time` that differs from the network packet's; the genesis is read once and those exact bytes are hashed, checked and installed; the config layout lacks a key the tool sets |
| `start` | not installed; the pinned binary is not the one the seat was made with; already running; `config/genesis.json` changed since install; the seat IP is not on this Mac; a port has a live listener (TIME_WAIT is ignored, as Go's own bind does); the process exits at once |
| `status` | the node answering on the seat's RPC port is not this seat's (another chain's node or process); with `--wait-height`, also the seat's node not running (checked on every poll). Reaching `--timeout` (default 100 s) first is not a refusal: it exits 3 with `"reached": false`, the height and the peers so far |
| `stop` | the pid's full command line is not `<root>/bin/zeroned start --home <seat home>`; it never escalates past SIGTERM. A node sleeping before `genesis_time` takes the signal but exits only at `genesis_time` (opening its ports for a moment, producing no block), and `stop` says so |
| `rehearse` | chain id not `zerone-rehearsal-local-*` or its directory already exists; a node cannot be stopped (the directory is then kept and the run fails) |

## Threat model, briefly

- **A careless or hostile coordinator** cannot do any of these without every
  seat's `install` refusing:
  - change parameters;
  - add or inflate an account;
  - mismatch a seat's stake;
  - drop or alter a seat's gentx;
  - turn a full seat's key into a validator.

  It does choose the seat list and the stakes. Each seat prints that table, and
  the owner confirms it before `start`.
- **A mixed-up courier** (wrong file, stale file, another chain's file, another
  tool version) is caught by the sha256, chain-id and tool-hash checks. The
  owner-relayed genesis sha256 closes the gap where the coordinator controls
  both the genesis and the network packet.
- **Another agent on the same Mac** can still read a seat's keys, because every
  agent runs as the same macOS user. The tool enforces owner-only permissions
  and single ownership by convention. Real isolation would need a separate
  macOS user.
- **Not covered:**
  - a seat that lies about its own role or stake request, which the owner sees
    in the table;
  - a LAN attacker (P2P is authenticated by node key, but the LAN is shared);
  - the provenance of the kit itself. The tool hash proves every seat runs the
    same `private-net.py`, not that it is the reviewed one. Where a Mac has
    repository access, compare it with the file at the merged commit. A kit
    carries no other code, and the tool drops its own folder from Python's
    import path, so a module planted beside it cannot shadow the standard
    library and fake its hash checks.

## Genesis profile `private-test-v1`

The binary's own `zeroned init` defaults, plus:

```
slashing.params.signed_blocks_window    100000
slashing.params.min_signed_per_window   0.05
slashing.params.downtime_jail_duration  60s
slashing.params.slash_fraction_downtime 0
```

The double-sign slash stays at the default 5%. Validators enter only through
gentx at genesis, because `docs/VALIDATOR-GUIDE.md` rules out the standard
`create-validator` path.

## Evidence

- **Loopback rehearsal** (`rehearse`; CI consensus job): 4 validators with
  their own keys and 1 full seat agree, a transfer is seen from the full seat,
  and the chain keeps finalizing with one validator stopped.
- **Two-root courier test** (`TwoRootCourier` in the tests; CI consensus job):
  the whole courier flow between two separate roots on one machine. The second
  root runs only the kit's copy of the tool. The test covers:
  - `kit`, `receive`, `binary` and `seat`;
  - `genesis`, `kit --round 2`, and `install` with the relayed sha256;
  - `start`, `status --wait-height`, `compare` and a transfer.
- **First LAN run, 2026-10-17** (`zerone-rehearsal-lan-1`):
  - **Setup:** two equal validators on the same home LAN: a Mac Studio
    (macOS 15.3) as coordinator, and a second Apple silicon Mac (macOS 26.6.2,
    as its agent reported in #107). AirDrop was the courier, and the second
    Mac's agent asked the owner for the genesis sha256. The binary was zerone-core
    0a4dd03b, built with `make build-darwin-arm64`.
  - **Timing:** with the owner couriering, the invitations, kit, seat packet
    and genesis took about 13 minutes. `genesis --start-in 1200` left room for
    both seats to install and start before `genesis_time`.
  - **Blocks:** at `genesis_time` the nodes found each other and produced
    blocks about every 2.9 s, each with the other as its one peer.
  - **Agreement:** `compare` agreed on all 21 shared heights.
  - **Transfer:** a 1 ZRN transfer from one validator showed in the other's
    balance.
  - **Firewall:** the coordinator's firewall rule was pre-authorised and
    Mullvad set to `lan allow`. On the second Mac the same two commands
    silently did nothing: Go signs every binary ad hoc as `a.out`, the
    firewall keys its entries by that identifier, and `--add` updated another
    `a.out` app's entry instead (#107 re-signs `zeroned` with its own
    identifier). The chain ran anyway because the second Mac dialled out to the
    coordinator; a seat that needs inbound peers would have been stuck.
  - **What the run changed:** it produced `kit` and `receive` (the run used a
    hand-built kit and moved files by hand) and `status --wait-height` (the run
    polled ad hoc). It also changed `pn` into a shell function, because a
    variable failed in zsh, and made the sha256 relay rule name agent channels
    explicitly.

## Sources

- **Gentx flow and node flags:** `scripts/local-consensus-rehearsal.sh`, CI
  "Consensus rehearsal". This tool keeps that script's loopback RPC,
  `pex=false`, `addr_book_strict=false` and `1uzrn` gas floor, and moves P2P to
  the seat's LAN address. Unlike that script, keys are created on each seat and
  never copied.
- **Guards:** `deploy/networks/zerone-dev-1/runtime.py` and
  `scripts/local-node.py`, for absent-home init, no reset, loopback RPC, API,
  gRPC and pprof off, `ZERONED_*` stripping, and the `SO_REUSEADDR` port probe.
- **Chain-id allowlist idea:** `scripts/init-testnet.sh`.
- **Invitations, public-data-only exchange, hash pins and the courier role:**
  the agenttool mission client handoff (`agenttool/bin/agenttool-mission.mjs`,
  PR #480).
- **CometBFT 0.38.25 behaviour relied on:**
  - `genesis_time` acts as a start barrier, and the node opens its listeners
    only after it;
  - persistent peers are resolved once at startup;
  - persistent-peer redial stops after about 24.6 h;
  - blocks are rejected when timestamped more than 60 s ahead.
- **macOS 15 behaviour relied on:**
  - Local Network privacy gates outbound LAN connections by the responsible
    app; Terminal and SSH are exempt;
  - application firewall rules are per app, not per port, and an app is keyed
    by its code-signing identifier. Go signs every binary ad hoc as `a.out`,
    so `build` re-signs `zeroned` as `org.zerone.zeroned`, the same way the
    Makefile signs `darwin-acl-check`. Observed on macOS 26.6 on 2026-10-17:
    `--add` for an `a.out` `zeroned` logged "updating existing app by bundle
    id" against a listed ipfs, then saved an unchanged configuration.
