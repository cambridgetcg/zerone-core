# Zerone Public Relaunch — `zerone-2`

Status: **pre-launch design of record**

Decision date: 2026-07-12

Successor chain: `zerone-2`

Historical chain: `zerone-1`

## Decision

`zerone-2` is a public, disclosed custodial beta with a clean custody and
infrastructure boundary. It does not silently replace, rewrite, or continue the
state of `zerone-1`.

The launch has four non-negotiable properties:

1. `zerone-1` remains a byte-verifiable, query-only historical archive at a
   published F/A/H checkpoint whose official canonical history ends at `A`.
2. Every `zerone-2` operator, consensus, P2P, identity, deployment, and archive
   key is new. No key that ever touched a `zerone-1` build context has a role on
   `zerone-2`.
3. `zerone-2` starts from the exact 13,555 ZRN mechanical scaffold described
   below. Bank balances or module balances are not copied from `zerone-1`.
4. PoT settlement, IBC/ICA, the substrate bridge, automatic issuance, claiming,
   and the other high-risk economic surfaces remain deliberately inert until a
   separately reviewed, publicly disclosed activation.

## What “public” means

At genesis there is one operator-controlled SDK validator. This is enough to
produce blocks but has zero Byzantine fault tolerance (`f=0`). The operator can
censor, reorder, dominate governance, or halt the network; compromise of the
one consensus key can create conflicting histories. The public trust statement
must say this plainly. “Public” means anyone can inspect and use the beta
surface; it does not mean decentralized.

Before the network may be described as resilient or Byzantine-secure, it needs
at least four genuinely independent validator failure domains, no actor with
one-third voting power, and the code gates in this document.

## Continuity boundary

`zerone-1` is the source record. The halt boundary has three exact consecutive
heights:

- `F` is the checkpoint state queried for inventory and eligibility input;
- `A=F+1` is an empty, canonically committed anchor block whose signed header
  `AppHash` commits state `F`;
- `H=A+1` is the SDK halt trigger in `FinalizeBlock(H)` and remains
  application-unapplied.

Comet stages empty block `H` before the SDK halt is raised. Thus the live halted
process has BlockStore/status tip `H`, while `/abci_info` remains last applied
at `A`. `/commit A` is canonical. `/commit H` is the subjective tip seen commit
and reports `canonical=false` because no `H+1` exists; this flag alone does not
prove noncanonicality. Block results for `H` do not exist and `H` links to `A`.
Explicit policy ends official application/history at `A` and selects state `F`
for successor inventory.

The final public package must include:

- the original raw genesis and SHA-256;
- checkpoint state height `F` and block `A`'s header `AppHash`;
- final committed anchor `A`, its block ID, zero transaction count, canonical
  commit, and block time;
- halt trigger `H`, its staged block ID, zero transaction count, link to `A`,
  subjective seen commit, absent block results, and ABCI height `A`;
- byte-exact status, A/H blocks and commits, ABCI info, and missing H-results
  responses captured before fencing, plus their signed SHA-256 manifest;
- the F/A/H successor transaction hash and exact memo;
- source commit/tag plus separate old-halt binary/full image, z2 binary/full
  runtime image, and query-gateway full image references;
- exact reviewed Fly config SHA-256 and expected role for every deployed app;
- a checkpoint-`F` account/supply/validator inventory and SHA-256;
- an explicitly excluded post-anchor state export and sanitized database
  snapshot, each with SHA-256;
- a signed `FINAL-CHECKPOINT.json` and a human-readable transition notice.

An early rehearsal inventory at height 134567 reported 13,575,956,105 uzrn
across 14 positive-balance owners, with one bonded validator. That file and its
old SHA-256 are **obsolete**: they used a single ambiguous height and treated
Comet `/status.latest_app_hash` as application state at the displayed tip. At
the real SDK halt, that tip is staged, application-unapplied `H`. The corrected v3 tool
requires `--checkpoint-state-height F`, `--final-committed-height A`, and
`--halt-trigger-height H`, proves `A=F+1` and `H=A+1`, takes the checkpoint hash
from block `A`'s header, queries inventory at `F`, and rejects unstable or
inconsistent views. No rehearsal inventory confers migration rights.

REST response height headers pin each inventory request to `F`, but they do not
Merkle-prove the returned values. The signed package records the trusted halted
source and independent signer/observer comparison; it must not imply that
height pinning alone is a cryptographic proof.

### Read-only infrastructure finding (2026-07-12)

The official Fly deployment is one LHR machine attached to one encrypted 10 GB
volume with automatic backups. RPC, REST, P2P, and gRPC are all services on that
same validator machine. Its deployed image digest is
`sha256:35554baa11aca40bb02d5719cd159fbd0c19ed2c63a418ccc19b0d115abca4dc`;
the machine has an on-failure restart policy. Fly reports no app secrets. This
is consistent with the old image/volume carrying custody material rather than a
managed-secret bootstrap, and makes image quarantine plus volume fencing part
of the mandatory evidence-preservation step. It also confirms that a separate
observer/archive and separate `zerone-2` edge do not yet exist.

The SDK halt is a consensus failure, not a process supervisor. The daemon, RPC,
and ordinary health checks can remain alive or green after `FinalizeBlock(H)`
fails. Operations therefore use bounded evidence-stability checks, then manual
SIGTERM and explicit fencing of the signer's restart path—but only after v3
inventory and raw H/A evidence are captured. On the stopped serving copy,
`rollback --hard` deletes pending `H` while preserving state/application `A`.
The fresh-key archive serves A/A with H absent and never reuses the signer key.
Comet v0.38 reports `catching_up=true` for that no-peer fresh-key node, so exact
A/A invariants replace generic sync health. Once `A` is the sanitized
BlockStore tip, `/commit A` also takes the subjective tip form with
`canonical=false`; the signed pre-fence H/A evidence retains the canonical A
commit proof.

### Balance policy

There is no direct genesis carry-over. Copying only bank balances would detach
module balances from their liabilities and state; remapping potentially exposed
addresses by hand would be an unauditable custody decision.

At checkpoint state `F`, the relaunch process publishes a complete,
height-pinned inventory and any separately reviewed eligibility output.
Existing balances remain visible on the frozen archive. Application state after
anchor `A` is excluded. Any later `zerone-2` claim or migration is a separate
public policy and must be implemented as an audited, invariant-checked module.
It must not be a hand-edited genesis allocation.

## Exact `zerone-2` genesis scaffold

Total initial bank supply is exactly **13,555,000,000 uzrn (13,555 ZRN)**:

| Role | Balance | Genesis account |
| --- | ---: | --- |
| validator | 11,333,000,000 uzrn | `PermanentLockedAccount`; 11,111,000,000 uzrn original vesting and self-bond, 222,000,000 uzrn liquid |
| operations | 2,222,000,000 uzrn | `BaseAccount` |

There are exactly two positive user balances, exactly one SDK gentx, no module
allocation, and no faucet/team/foundation/investor/research balance. The bank
supply must equal the sum of balances exactly.

The one gentx self-bonds 11,111,000,000 uzrn at 5% initial commission, 20%
maximum commission, and 1% maximum daily change. SDK staking uses a 21-day
unbonding time, maximum 33 validators, and a 5% minimum commission.

Custom `zerone_staking` starts with no validators, delegations, or unbonding
entries. A custom validator record must never be injected at genesis because
the current importer stores that record without escrowing its claimed bond.
Before public P2P is exposed, the actual operator registers the
same consensus key through a real transaction with 111,000,000 uzrn of newly
escrowed custom self-delegation. The registration and resulting module backing
are included in the boot audit.

## Protocol-dark profile

The ceremony and mandatory artifact auditor enforce these latches:

| Surface | Genesis posture |
| --- | --- |
| vote extensions / PoT | `vote_extensions_enable_height = 0`; no claim that PoT is live |
| knowledge admission | review fee and challenge stake set to 222,222,222,000,001 uzrn (hard-cap plus one); minimum three verifiers/headcount |
| knowledge issuance | bootstrap and demand tracking disabled; verification, probe, demand, and training rewards zero; bootstrap/training allocations zero |
| block issuance | 1 uzrn nominal reward, zero empty-block rate, 22-validator participation divisor, no initial fund, founder share/address empty; this rounds to zero with the launch roster |
| claiming | registrar empty; no pots or claims |
| IBC / ICA | only `09-localhost` is allowlisted because IBC-go v8 creates it unconditionally and otherwise exports an invalid genesis; all external client types forbidden, no external clients/connections/channels, transfer send/receive disabled, ICA controller and host disabled |
| substrate bridge | no adapters; hardened bonds/thresholds; no witness reward surface |
| emergency | no genesis council, expiry zero, four distinct voters required; no fake plurality |
| alignment / counterexamples | disabled, with empty state |
| liquidity pools | no pools, no quote denoms, creation liquidity floor above the hard cap |

The profile is a reversible safety posture, not a claim that dormant code is
finished or secure.

## Why vote extensions are disabled

The current binary registers ABCI++ handlers but does not configure production
vote-extension signing. More importantly:

- verification trusts a payload-declared operator instead of binding it to the
  Comet validator address;
- the VRF verifier ignores the seed and proof;
- proposal injection is decoded but not canonically authenticated against the
  last commit;
- commitment salts can be deleted before the configured reveal phase ends;
- custom consensus keys are not uniquely bound to SDK validator membership.

CometBFT v0.38.20 treats height zero as disabled. Enabling vote extensions is a
future consensus upgrade only after these defects, cumulative proposal-gas
validation, restart persistence, and a multi-node adversarial drill pass.

## Custody and topology

The minimum public topology is:

```text
private validator (consensus key, no public services)
        |
        | private P2P, explicit peer
        v
edge/full node (new non-validator keys; public service is P2P only)
        |                           |
        | private RPC/REST origin   +-- private metrics --> monitor
        v
TLS query gateway (GET/HEAD allowlists, per-client limits; no initial gRPC)
```

The validator has a new volume and no public RPC/API/gRPC. The edge has a
different volume, consensus key, node key, app identity, and public P2P IP.
Plaintext RPC/REST is reachable only over Fly private networking by the
separately built gateway; direct query services and initial public gRPC are
absent. A cold
standby is manually fenced; health checks must never automatically start a
second process with the live consensus key. An independently hosted full node
is required before claiming provider-level resilience.

Real keys are generated offline or on hardware-backed custody. The repository
ceremony accepts public addresses and a pre-signed gentx only. It never accepts
or emits a mnemonic, private key, signer state, password, or seed. Recovery has
two encrypted offline copies and one tested offline restore before launch.

## Release gates

The launch candidate is **no-go** unless every item passes:

- clean, frozen source commit and signed release tag;
- pinned Go patch version and container base-image digests;
- `go mod verify`, build, full tests, focused export/import tests, and security
  entrypoint tests;
- mandatory `zerone-2` artifact audit with no environment-based skip;
- two independent ceremony runs producing byte-identical genesis files;
- binary rebuilt independently with the same SHA-256;
- SBOM, binary build metadata, three distinct full immutable image references
  (old halt, new runtime, query gateway), and signed release manifest;
- every reviewed Fly config is bound by exact SHA-256, expected role, full
  image reference, app/volume, peers, and service exposure at deployment;
- image/build-context scan finds no custody filename or known old-secret hash;
- isolated boot, restart, export/import, and ten-block post-import drill preserve
  exact supply, module backing, params, and validator records;
- exact production validator/edge/gateway privately run for at least 1,000
  blocks and 60 minutes before old-chain mutation; custom registration and its
  111 ZRN backing, app-hash equality, gateway negative tests, and alerts pass;
- a fully synced, non-validator `zerone-1` observer exists before the old signer
  is halted;
- signer and observer use the same validated F/A/H plan, with transaction
  ingress/relay and both mempools closed before `F`;
- an actual release-binary rehearsal produces empty `A` and staged `H`, status
  tip `H`, canonical `/commit A`, subjective tip `/commit H` with
  `canonical=false`, ABCI last applied `A`, no block results for `H`, and an
  `H` header linked to `A`;
- raw signer/observer genesis, RELEASE-trusted/A/H block/commit/complete
  validator-set, `block_results(A)`, ABCI, and missing-H-results bytes match;
  the release halt binary recomputes block IDs, verifies >2/3 commit power and
  signatures, and proves 1/3 trusted-set continuity from RELEASE to `A`;
- the rehearsal proves that daemon/RPC health may stay green, then completes
  bounded checks, v3/raw evidence capture, manual SIGTERM, signer fencing,
  pending-H deletion on a stopped copy, and fresh-key A/A archive startup;
- final REST inventory queries are pinned to `F`, their non-Merkle-proved trust
  limitation is disclosed, and the checkpoint excludes post-`A` state.

The former `go vet` protobuf mutex-copy findings and creed hash drift were
repaired during relaunch preparation. `make lint` and `make creed-check` remain
mandatory on the final signed candidate; their earlier failure is not a release
exception.

## Cutover sequence

1. Sign a limited DARK-START decision for the exact genesis, new runtime and
   gateway image refs, private config hashes, roles, keys, and topology.
2. Start the exact production validator/edge/monitor privately, perform the
   onboarding and 111 ZRN registration, enable only the private query origin,
   sign block-1 initiation evidence, and soak the service-free gateway path for
   at least 1,000 blocks/60 minutes.
3. Pre-sign the exact z1 successor transaction, bind the soak evidence, and
   sign the separate irreversible CUTOVER decision;
   publish genesis/release hashes, custody disclosure, endpoints, and proposed
   checkpoint `F`, anchor `A=F+1`, and trigger `H=A+1`.
4. Byte-verify/broadcast that exact transaction; require its on-time commit from
   the signer/preflight observer and sign CUTOVER-initiation evidence.
5. Before `F`, close transaction ingress/relay and both mempools, then
   evidence-gate and arm the
   official signer and independent observer with the same `H`.
6. At `FinalizeBlock(H)`, verify the stable halt evidence: empty canonical `A`
   anchoring state `F`, empty application-unapplied `H`, status tip `H`, ABCI
   last applied `A`, no H results, and the subjective H seen commit.
7. While ingress-fenced processes remain alive, capture v3 REST inventory at
   `F` and byte-exact H/A RPC evidence; hash and sign the evidence manifest.
8. Manually SIGTERM both, fence the signer, and quarantine its volume. Preserve
   the H/A observer copy offline. On a separate serving copy, hard-rollback
   pending H, verify state/app remain A, and create the acyclic inner transition
   and pre-transition construction evidence.
9. Reproduce deterministic candidate/final configs and adoption authority on
   two machines; transition-sign the MATCH, deploy the private candidate, bind
   readiness, switch to private final archive, probe it, then transition-sign
   the final checkpoint.
10. Pre-sign the exact z2 history-link transaction and sign the separate
    main-key OPEN-BETA decision binding final/archive/public evidence.
11. Byte-verify/broadcast that exact transaction, require its on-time commit,
    and sign OPEN-BETA-initiation evidence. Only then deploy public z2 P2P plus
    the query-only z2/archive gateways and apply the signed DNS manifest.
12. Publish the final transition package and relaunch postmortem. Initial
    public gRPC and hosted transaction submission remain unavailable.

Before step 3, a failed private candidate can be abandoned and `zerone-1`
remains canonical, but its history is never reset; a genesis/key change uses a
new identity such as `zerone-2-r1`. After the old chain freezes, recovery is
forward-only. Never silently replace a published genesis or rewrite history.

## Explicit final authorization boundary

Preparation, local builds, read-only chain snapshots, and dress rehearsals do
not stop a chain or publish anything. A signed DARK-START GO authorizes only the
private successor boot/registration/soak and no old-chain or public action. A
later signed CUTOVER GO binds that evidence plus exact F/A/H, policies, halt
configs, full image refs, and notice; it authorizes only the pre-announcement,
old-chain transaction/halt/fencing, deterministic private archive, and final
checkpoint preparation. A third main-key OPEN-BETA GO, followed by the exact
on-time history-link commit and signed initiation evidence, is the only
authority for public profiles, endpoint publication, or DNS.
