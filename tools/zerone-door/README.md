# zerone-door

The admission door for `zerone-1`. A household agent proves it holds its own keys;
the door checks it against a household allowlist and the chain, and then the
registrar key R signs one transaction. That transaction gives the agent a 0.222 ZRN
starter seed and a bounded fee allowance for claiming the seed, registering, and
submitting claims. The agent-facing guide is [JOIN-AGENTS.md](JOIN-AGENTS.md).

Client-side only, like `tools/frontier-intake`. It shells the pinned `zeroned` CLI to
build, encode, sign and broadcast, reads the chain over the public REST and RPC
endpoints, and contains no consensus code. It needs the audited Bun 1.3.5 exactly,
and has no npm dependencies of its own. It takes `accountRegistrationProofSignBytes`
from this repository's `sdk/typescript` (the source of `@zerone-chain/sdk`), so
`sdk/typescript` needs its dependencies installed (`npm ci`) for tests and builds.

```
bun door.ts run    [--config PATH] [--once] [--interval SECONDS] [--live-ack=zerone-1]
bun door.ts status [--config PATH]
bun door.ts verify FILE
```

A **dry run is the default**: every gate runs against live state, the real admission
transaction is built and simulated, and nothing is signed. A live run needs
`--live-ack=zerone-1`, and the id must match both the config and the node. `status` is
read-only; it exits 4 when the admittee watch (below) flags anyone, and 5 when nothing is
flagged but a chain read failed.

## Why it is shaped this way

Yu, 2026-10-28: "keep the registrar, open it for all agents" and "lets open up
mainnet participation, along with an update at zerone.ai". His answers that day:
timing "Now, full door" (seed and fee allowance from now, accepting the pre-H1
exposure); eligibility "Our private network first" (lobby desks and beings on the
household Macs who opt in; widen later); door key "Dedicated hot key, small float"
(a single-purpose registrar key R with its phrase on steel, held by the door service
with its passphrase in the login Keychain, a ~30 ZRN float, and a window cap cut so a
leak costs little). The ledger items are `zc-open-agent-registration`,
`zc-claim-fee-period` and `zc-z1-key-renewal`. The design and its two verifier
reports were written by agents for that session.

## Intake: how a request reaches the door

| Channel | Who | How | Why it proves "household" |
|---|---|---|---|
| `studio_local` | beings on the Studio | a `*.request` file in the door's `inbox/` | the inbox is a `0700` directory owned by the user that runs the door; the door refuses to read it otherwise |
| `lobby` | desks on the household's other Macs | table talk in the door's fleet room, conversation `zerone-door` | the hub reads it with `agenttool-lobby room read --json`, which verifies each sender against the room's door-signed roster (`label: verified`, `basis: door_roster_pin`, `current_member`) |

Posting a request needs no ZRN, and the lobby client has no payment step (whether the
agenttool service meters mission messages was not checked), so a zero-balance agent
can reach the door. A free agenttool listing cannot exist (listings need a positive
price), which is why the lobby, not the marketplace, carries remote requests. The
request is plain text with every binary value in lowercase hex: it passes the lobby's
egress gate, which lets hex through and refuses mixed-case base64 (checked against the
installed gate at agenttool `68044b45`; a request is about 660 bytes and its table
message about 814 Unicode scalars, under the 1,000 limit).

The door remembers where it stopped reading the room together with the room's season
(its agenttool mission id), because agenttool numbers receipts per season, starting
again at 1. When the room has rolled to a new season, the door reads the new season
from its start and logs `lobby_season_changed`.

Answers: every request gets a reply file in `replies/` (keyed by the request id). For
lobby requests the door also answers in the room, but only when it runs live and
`lobby.reply_lease` names a room seat lease a person made at the hub terminal; a dry
run never posts in the room, and a reply is never retried. The chain itself (the seed
pot and the allowance) is always the real answer.

## Membership: allowlist and lobby proof, both

Eligibility is checked against **both** a household-maintained allowlist and, for lobby
requests, the membership proof the lobby already issues (its door-signed roster):

- The **allowlist** (`allowlist.json`, local, never committed) is the household's
  explicit yes for the door: whether Studio-local requests are open (off by default),
  which lobby desks (by agenttool identity id) may ask, and each one's daily cap. A
  lobby seat is granted for talking at a table and renews daily; the allowlist keeps
  "this desk's agents may receive ZRN from R" a separate decision that can be withdrawn
  without touching the lobby. It is read fresh for every request.
- The **lobby's roster proof** authenticates a remote sender. An allowlist alone can
  name a desk but cannot prove a message came from it.
- Studio beings share the hub's own desk, so the lobby cannot vouch for them; for them
  the private inbox is the proof and the allowlist's `studio_local` switch is the yes.

Opting in is the request itself: each agent asks for itself, with keys it made.

**A request is not bound to the desk that posts it.** The request proves who holds the
agent's keys, not which desk the agent lives on, so an allowlisted desk can relay a
request for an agent whose keys were made anywhere. The allowlist limits which desks
can ask, not which agents get in; each listed desk answers for what it relays, and the
per-desk daily cap bounds how much it can.

## The request

```
zerone-door-request/v1
chain_id: zerone-1
address: <zrn1, canonical lowercase>
identity_public_key: <Ed25519, 64 hex>
nonce: <32 hex, random>
issued_at: <ISO 8601 UTC>
account_public_key: <secp256k1 compressed, 66 hex>
account_signature: <128 hex: ADR-036 sign/MsgSignData over the first six lines>
identity_signature: <128 hex: Ed25519 over accountRegistrationProofSignBytes(...)>
pow_nonce: <decimal: sha256(request LF pow_nonce) has 20 leading zero bits>
```

The identity signature covers the exact accounting-authority-v1 registration bytes
(`zerone.auth/register-account/v1`; chain `zerone-1`, the sender, `did:zrn:<key>`, the
key, type `agent`, empty metadata), so the same proof is valid on chain once field 7 is
enforced. Ed25519 is verified strictly (no small-order or non-canonical key or R, and
S < L); secp256k1 requires low S. Tests check interop with a request signed by CosmJS
0.39 `signAmino`, `@noble/curves` 2.2.0 and the SDK.

## The gates

Every gate refuses with a named reason; a refusal lists every reason that applies.
Chain reads fail closed: only an exact, known not-found answer counts as absent.

- **Request:** format; chain id; canonical lowercase bech32 (compared everywhere by its
  20-byte payload); fresh (15 minutes, 2 minutes of clock skew); the account key derives
  the address; ADR-036 signature; strict identity proof; 20-bit proof of work; never
  seen before.
- **Membership:** allowlist channel or desk; lobby roster verification.
- **Door and chain:** no `HALT` file in the door home and no
  `~/.local/state/zerone/rung-window.freeze`; no pending upgrade plan; the chain version
  is the pinned one (below); the node serves `zerone-1` and is not stale; the on-chain
  `bootstrap_registrar` is R; R is registered in `zerone_auth`, not frozen, and has a
  public key on chain.
- **Subject:** not R and not a household, retiring or known key (`known_keys` in the
  local config); no balance in any denom, no account, no `zerone_auth` record, no seed
  pot under either spelling (`bootstrap-zrn1...` and `bootstrap-ZRN1...`), no fee grant
  from anyone; the DID is `did:zrn:` + 64 lowercase hex (by construction) and is not
  registered in any of four spellings: lower or upper case, 64 or 32 hex. (The chain
  also accepts mixed-case hex and keys each spelling separately; the door cannot
  enumerate every mixed-case one.) The address, identity key and DID were never
  admitted by the door.
- **Capacity:** one transaction at a time (no new admission while an earlier one is
  unconfirmed, since R's sequence would collide); 10 per UTC day overall; per channel
  (each lobby desk, and the Studio as one channel) 3 a day by default, or the number
  the allowlist sets for it (0-100), always inside the overall 10; 20 per chain
  admission window (34,272 blocks); room in the chain's own window (read from the
  `claiming_pot` store); at most 10 unclaimed seeds **whose allowance from R is still
  live** (paginated; seeds whose allowance lapsed are listed by `status` as an alert
  and never stop the door); R's spendable balance covers every live allowance it has
  issued, this transaction's ceiling and a 2 ZRN reserve.
- **Re-grant, once, only for a grant that ran out by time unused:** an admittee whose
  seed is still unclaimed may ask again with the same keys and gets one fresh allowance,
  but only when the grant the door recorded has expired at block time **and** the
  agent's account sequence is still 0. A grant missing before its expiry was drained to
  zero (feegrant deletes it) or revoked, and is refused as `already_admitted`; so is a
  grant the agent spent from, since only transactions the agent signs can draw on it. An
  expired grant the feegrant EndBlocker has not pruned yet is revoked in the same
  transaction. A revoke alone stops blocking a re-grant once the recorded expiry passes:
  to end an admittee for good, also add its address to `known_keys`.

### The chain version pin

`config.json` pins the chain version the door's gas figures, REST shapes and pinned CLI
were measured on: the last plan `x/upgrade` applied (read from its done markers in the
store) and a sha256 over the chain's `module_versions`. Today that is
`agenttool-seam-v1` (applied at height 527,603) and `a7273748…72643`, as
`config.example.json` carries them. Any difference refuses every request with
`chain_version_changed` until a person re-measures gas on the new version and re-pins;
`status` prints the observed and pinned values side by side.

## The admission transaction

One per agent, signed by R:

1. `MsgAddBootstrapEntry{authority: R, addresses: [addr]}`;
2. `MsgGrantAllowance{R -> addr}`: `AllowedMsgAllowance` over a `PeriodicAllowance` for
   `[/zerone.claiming_pot.v1.MsgClaim, /zerone.auth.v1.MsgRegisterAccount,
   /zerone.knowledge.v1.MsgSubmitClaim]`.

Gas limit = fee = 150,000 uzrn. It simulated at 110,906 gas at h1,402,838, and again
near h1,402,995, with the live registrar as a public-key-only stand-in; July's delivered
admissions ran about 15,000 above Simulate. The door refuses when simulated gas + 20,000
exceeds the limit. A re-grant is `MsgGrantAllowance` alone (after a revoke when needed),
gas limit = fee = 140,000; the grant alone simulated at 88,429.

| Allowance | Value | Sizing |
|---|---|---|
| period spend limit | 400,000 uzrn per 900 s | the most any single sponsored transaction can pay; a claim needing more gas is never paid |
| total spend limit | 1,100,000 uzrn | seed claim 170,000 + registration 130,000 + one claim 400,000 + one retry 400,000 |
| expiration | admission + 14 days | |
| agent gas | claim seed 170,000; register 130,000; submit claim 400,000 | fee-granted seed claims used 128,378-142,289; registration simulated 86,946; an SDK-built claim simulated 244,357 and the household's delivered claims used 279,929-351,830 |

The seed (0.222 ZRN) pays one review fee at today's 0.2 ZRN; a fee allowance can never
pay a review fee. What R can lose:

- One admission costs R at most 1.25 ZRN (fee 0.15 + a fully spent allowance 1.1).
- A re-grant follows only an allowance that expired unspent, so one agent costs R at
  most 1.39 ZRN in all (0.15 + 0.14 + 1.1).
- The 20-per-window cap bounds admissions: 20 x 1.25 + the 2 ZRN reserve is 27 ZRN,
  inside R's ~30 ZRN float. Re-grants are not in that cap; the daily cap counts them,
  and the liability gate refuses any transaction that would leave R unable to pay every
  live allowance plus the reserve.

Allowances are not escrowed: R pays only as agents spend.

## Signing, custody and the run loop

R lives in a `file` keyring. Its passphrase is read at signing time from a
login-Keychain generic password with `/usr/bin/security find-generic-password -s
<service> -w` and handed to `zeroned tx sign` on stdin; it is never on an argv (`ps`
shows argv), never in the environment and never logged. Every buffer the door holds it
in is zeroed after use; copies inside the Bun runtime, the pipes and the child
processes are beyond the door's reach. Which keyring and which Keychain item are named
only in the local `config.json` (mode `0600`, never committed), because a map of where
keys live must not be published.

This is a hot key by Yu's choice: R is as safe as the account that runs the door. The
float and the caps bound what a leak costs; the kill switch is a governance freeze of
R plus clearing the registrar, or R revoking its grants.

For each request: verify -> membership -> read the chain -> gates -> build with
`--generate-only` and a memory keyring -> check the built messages exactly (types,
authority and granter R, the one address, the allowance shape, the fee) -> Simulate
with R's on-chain public key and live sequence -> (live) sign offline with R ->
check the signed body, fee, key and sequence match what was simulated -> record the
transaction hash -> broadcast (sync) and check `code` -> wait up to 300 s for
inclusion and check `code` -> postcheck the seed pot and that the allowance on chain is
the one built -> reply.

Every transaction R signs carries a `timeout_height`: the node's height when the door
read the chain plus 40 blocks (about 20 minutes; the reasoning is beside
`TX_TIMEOUT_BLOCKS` in `src/policy.ts`). SDK v0.50's `TxTimeoutHeightDecorator`, which
zerone's ante chain runs, refuses it in every block above that height. The ledger and
the reply record it. A broadcast not yet seen to run stays `pending_inclusion`; while
anything is pending the door admits no one else (`door_busy`), and later runs only
look it up and never re-broadcast. It leaves `pending_inclusion` only on proof, never
because time passed:

- **found** in a block, by its hash or as the transaction the chain's `tx.acc_seq`
  index names for its R sequence (by that hash, or, under a hash the door did not
  compute, by carrying exactly its messages: the bootstrap entry for the agent's
  address and the grant from R to it, or for a re-grant the grant after an optional
  revoke; the chain's hash is then recorded and `hash_differs` logged): `included`
  when it ran (code 0) and the chain shows
  what was built, `postcheck_failed` when it ran and the chain state differs; with a
  non-zero code, `delivery_failed` once R's sequence has moved past the one it was
  signed with (it ran and its messages failed, or another transaction took the
  sequence), so it can never run again. A non-zero code while R's sequence has not
  moved means the block's ante handler refused it before anything in it ran (a fee R
  could not pay at that moment, say), and the same signed transaction can still run in
  a later block: it stays pending, its first reply is `pending` with reason
  `delivery_failed`, and it becomes `delivery_failed` only by the expired proof below;
- **expired:** the node is at least 3 blocks past its timeout height, the transaction
  is not found, and R's sequence never moved past the one it was signed with in a state
  at or past the timeout height, so no block ran it and none can. The door reads that
  sequence from the RPC node (`abci_query` of `/cosmos.auth.v1beta1.Query/Account`),
  whose answer names the height of the state it was read in, and refuses the proof
  below the timeout height. It never takes it from `rest`, which may be another node
  (the public endpoints sit behind a proxy) and gives its height only in a gRPC-gateway
  response header;
- **superseded:** R's sequence moved past it and the `tx.acc_seq` index names one other
  transaction that used that sequence, whose messages are not this admission's (the
  door reads them from the transaction bytes the index returns; a hash that differs
  never makes a transaction another's by itself).

`delivery_failed`, `expired` and `superseded` do not count, so the agent may ask again.

Until one of these is proven the entry stays pending, counts, and keeps the door busy,
for as long as that takes: when the chain keeps making blocks, the door can read it and
the node indexes `tx.acc_seq`, that is the first run after the node is 43 blocks past
the height the door read before signing; while reads fail or the chain stalls, it is
until they recover; and if R's sequence moved and the index never names this
transaction or exactly one other for that sequence, it is until a person acts. The
door logs `pending_needs_attention` on every run that leaves an entry pending

- once the door's clock is 21.5 minutes past the entry's signing
  (`PENDING_ATTENTION_MS`: 40 + 3 blocks at 30 s), whatever keeps it unproven, failed
  reads included; and
- at once when it cannot settle on its own: no signing sequence or no timeout height
  recorded, R absent on chain, or R's sequence moved and the index names neither this
  transaction nor exactly one other for that sequence once the node is 3 blocks past
  the timeout height.

A ledger written before timeout heights may hold entries marked `dropped` after 30
minutes, which proved nothing: each run re-opens them as pending (they count and keep
the door busy again) and settles them by the same proof. Such an entry has no timeout
height, so if it is never found it settles only when R's sequence moves past it; any R
transaction does that.

An admission counts in the chain admission window of the block that included it, as
the chain counts it, so the 20-per-window cap follows the inclusion height. The daily
caps count by the UTC day the door signed, on its own clock, which is the clock they
read "today" from; with one transaction in flight at a time, at most one signed just
before midnight runs the next day.

## The admittee watch

The door cannot bind what an admitted agent registers later (see below), so
`door.ts status` reads each admittee's `zerone_auth` record and flags one registered
with a type other than `agent` (`type_not_agent`) or with a DID other than the one its
request named (`did_differs`). Admittees not registered yet are listed, not flagged;
an admittee whose record could not be read is listed as unknown, never as clean. The
watch runs when `status` runs; nothing schedules it. For a flagged admittee the
household can revoke the allowance, add the address to `known_keys`, or ask
governance to freeze the account.

## Files

The door's home directory (`install.sh --door-home`; it must be outside `~/Desktop`,
because launchd jobs cannot read `~/Desktop`):

```
config.json            0600  local only: endpoints, pinned zeroned, chain version pin, R, known keys, lobby
allowlist.json         0600  local only: who may ask
inbox/ processed/      0700  Studio-local requests, and where they go once taken
replies/               0700  one reply per request id (replies/dry-run/ for dry runs)
state/ledger.json      0600  admissions, seen requests, lobby cursor and season (dry runs: state/dry-run-ledger.json)
HALT                         if present (any type), the door admits no one
releases/<commit>/     0555  pinned bundle + MANIFEST.json; current -> releases/<commit>
bin/                   0555  bun-1.3.5 and zeroned-<sha>
launchd/com.zerone.door.plist   rendered, not loaded
logs/door.log                JSON lines; no secret is ever written
```

## Install

```
(cd sdk/typescript && npm ci --ignore-scripts)
tools/zerone-door/install.sh --bun /abs/bun-1.3.5 --zeroned /abs/zeroned --zeroned-sha256 <hex> [--door-home DIR]
```

It refuses uncommitted source, a Bun other than 1.3.5, a zeroned whose sha256 differs,
and a door home under `~/Desktop`. It runs the tests, builds the bundle, smoke-tests it,
installs it read-only with a manifest, copies the runtime and CLI, creates the
directories, writes `config.json` and `allowlist.json` from the examples only if they
are absent (Studio-local intake off, no lobby desks), and renders the plist. It does
not touch R's keyring or the Keychain, does not copy the plist into
`~/Library/LaunchAgents`, and does not load it. The rendered job is a dry run until
`--live-ack=zerone-1` is added by hand.

## Tests

`bun test ./tests` covers every gate (including the chain version pin and a re-grant
refused after a drained or revoked grant), the request format and each signature
defect, strict Ed25519, low-S secp256k1, bech32 canonical form, the CosmJS/noble/SDK
interop vector, the Go v1 registration-proof vector, the spool and lobby intake
(including a season change), config and allowlist validation, the admittee watch, and
decoders checked against answers zerone-1 served (the upgrade done markers, an account
from the gRPC Account query, and two of the registrar's July transactions from its tx
index, with the matcher that tells the door's transaction from another).

The run loop runs against a fake chain, a fake CLI and a fake lobby: dry run, a dry
run that never posts in the room, live admission with the passphrase on stdin only, a
live reply in the room, replay, CheckTx rejection, inclusion timeout then
reconciliation without re-broadcast, re-grant with revoke, and a signature by the wrong
key never broadcast. What becomes of a broadcast not seen to run: one delayed past 30
minutes that then lands (settled, counted, the door busy until then); one absent past
its timeout height (expired only after the settle blocks, and only on R's sequence
read on the RPC node at or past the timeout height, never from a lagging REST node);
one a block refused before it ran (pending until its timeout height passes, admitted
when the same transaction runs in a later block, failed at once when another
transaction takes its sequence); one that ran and failed in its messages; R's sequence
used by another transaction (superseded, an admission of another address included),
by the door's own transaction under another hash (found by its messages), or moved
with nothing indexed yet (still pending); reads that keep failing (pending, and
`pending_needs_attention` once past 21.5 minutes); an admission included across a
chain window boundary (in this run and by a later one); and `dropped` entries from an
older ledger re-opened and settled.

The keyless live check runs only when asked:

```
ZERONE_DOOR_LIVE_E2E=1 ZERONE_DOOR_ZERONED=/abs/zeroned-2e37c4c bun test ./tests/live-e2e.test.ts
```

It makes a request from public test seeds for an address that has never existed, runs
every gate against live `zerone-1` with the current on-chain registrar as a
public-key-only stand-in for R and the chain version pin from `config.example.json`,
builds the real admission with `--generate-only`, and simulates it. It signs and
broadcasts nothing.

## What the door cannot do

- It cannot bind what an admitted agent registers later. An `AllowedMsgAllowance` limits
  message types only, so an admittee can register as `human` (taking the human role
  bonuses), squat a DID it has seen, or pay for another signer's registration. Until
  accounting-authority-v1 enforces the identity proof on chain, nothing stops that. The
  admittee watch above finds a mismatch after the fact; it does not prevent one.
- It cannot prove an agent is an agent. The bounds are the private network, the
  allowlist, one admission per key, and the caps.
- Seeds are transferable ZRN. A `MsgSend` writes a `zerone_gov` FundingRecord, which the
  planned R4 `plan.Info` commitment hashes; door agents can therefore change state that
  R4's activation checks, and the R4 lane must account for that.
- Unclaimed seeds hold `claiming_pot` pot slots. A seed is an ACTIVE pot until it is
  claimed, and active pots count toward `max_pots_active` (10 on `zerone-1`): governance
  `MsgCreatePot` refuses once 10 pots are active, while `MsgAddBootstrapEntry` never
  checks the limit and bootstrap pots never expire. The door stops at 10 unclaimed
  seeds with a live allowance but does not count lapsed ones, so abandoned seeds can
  keep governance from creating pots until they are claimed or governance raises
  `max_pots_active`.
