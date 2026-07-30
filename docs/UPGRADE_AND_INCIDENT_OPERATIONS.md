# Zerone Upgrade and Incident Operations

This is the canonical operator runbook for planned design changes, software
upgrades, hostile events, recovery, and restart. If another Zerone document
uses the words *halt*, *rollback*, *resume*, or *quarantine* differently, this
runbook governs the operational meaning.

The mechanism-oriented references remain useful:

- [UPGRADES.md](UPGRADES.md) describes the existing `x/upgrade` flow.
- [UPGRADE_PROTOCOL.md](UPGRADE_PROTOCOL.md) describes how handlers and
  migrations are written.
- [INCIDENT_RESPONSE.md](INCIDENT_RESPONSE.md) describes the on-chain incident
  record.
- [OPEN_CRYPTO_SDK.md](standards/OPEN_CRYPTO_SDK.md) describes the strict
  SDK 0.53 / IBC-Go 10 migration evidence.

They do not override the safety boundaries below.

## 1. Reading this runbook

`OBSERVATION` means behavior verified in the current source or a
height-qualified fact collected from a running chain. `DECISION` means an
operational rule adopted here. Never present a decision as something already
enforced by consensus, and never present live state without a query height,
timestamp, RPC source, and result hash.

The dependency baseline is Cosmos SDK 0.53.8, CometBFT 0.38.25, and IBC-Go
10.7.0 in [go.mod](../go.mod).

The words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are normative.

## 2. Four independent state axes

Never compress system state into one word such as “halted.” Every operations
manifest and status update MUST report all four axes.

### 2.1 Consensus axis

| State | Meaning |
|---|---|
| `PRODUCING` | A single canonical committed history is advancing. Record height, block hash, app hash, and commit voting power. |
| `STALLED` | No new block has obtained more than two-thirds voting power. A stall alone does not prove corruption or a fork. |
| `DIVERGENCE` | Nodes disagree at the same height about application state, binary behavior, or the expected app hash. Stop affected signers and preserve evidence before repair. |
| `FORK` | More than one history is being treated as canonical, whether adversarially or by an explicit recovery decision. A fork is not a routine restart. |

CometBFT needs more than two-thirds of voting power to commit. Stopping one
signer does not necessarily stop a multi-validator chain; stopping at least
one-third of voting power prevents a new commit. See the upstream
[CometBFT consensus specification](https://docs.cometbft.com/v0.38/spec/consensus/consensus).

### 2.2 Signer axis

| State | Meaning |
|---|---|
| `ACTIVE` | The consensus signer is authorized and able to sign the canonical chain. |
| `STOPPED` | Signing is deliberately disabled. The full node may also be stopped, but these are separate facts. |
| `SUSPECT` | Key compromise, duplicate execution, unsafe restoration, or equivocation is possible. A suspect signer MUST be stopped and isolated. |
| `RETIRED` | Its voting power has been removed or its replacement has completed; it must never sign again for the retired identity. |

Any signer operator MAY defensively stop their own signer without waiting for
permission. Restart requires the recovery quorum in section 5.

Consensus keys and account/governance keys are different systems. CometBFT
validator signing uses its consensus key and last-sign state; transactions use
Cosmos SDK account keys. See the upstream
[validator-signing rules](https://docs.cometbft.com/v0.38/spec/consensus/signing)
and [Cosmos SDK account/keyring documentation](https://docs.cosmos.network/sdk/v0.53/learn/beginner/accounts).

### 2.3 Transaction-admission axis

| State | Meaning |
|---|---|
| `OPEN` | Normal transaction admission. |
| `EDGE_QUARANTINE` | Public RPC, REST, gRPC, relayers, or sentries refuse selected traffic. Other peers or validators can still introduce transactions. |
| `CONSENSUS_QUARANTINE` | Application consensus logic rejects non-allowlisted transactions on every honest node. Blocks may still be produced. |

`EDGE_QUARANTINE` is useful against DDoS and opportunistic exploitation, but
it is not a consensus control. `CONSENSUS_QUARANTINE` controls transactions,
but it is not a block-production stop.

### 2.4 Release axis

| State | Required evidence |
|---|---|
| `PREPARING` | Change, threat model, migration scope, and recovery design under review. |
| `RELEASE_FROZEN` | Source commit, upgrade name/height, build inputs, binary, provenance, SBOM, independent review, and rehearsals are fixed and verified. |
| `SCHEDULED` | One canonical on-chain `x/upgrade` plan exists at height `H`. |
| `STAGED` | Exact attested artifact installed in the matching Cosmovisor directory on each reporting node. |
| `HALTED_AT_H` | The old binary committed H−1 and reached the scheduled stop before committing H. |
| `MIGRATING_H` | The attested binary is executing the deterministic height-H migration. |
| `OBSERVING` | H or later committed; app hashes, module versions, supply, IBC, and functional probes are under review. |
| `ACCEPTED` | Post-upgrade checks and the observation window passed. |
| `CANCELLED` | Plan cancelled with a recorded checkpoint below H. Impossible after H commits. |
| `RECOVERY_FAILED` | Activation/observation failed after the forward-only boundary; open an incident recovery journal. |

These are the exact machine states enforced by `zerone-ops` in section 6;
`x/upgrade` stores only its own scheduled/applied state.

## 3. Four mechanisms that must not be confused

### 3.1 Transaction quarantine: current `x/emergency`

`OBSERVATION`: `x/emergency` is a consensus transaction gate. The ante
decorator in [app/ante_zerone.go](../app/ante_zerone.go) rejects normal
messages while the keeper reports halted. It admits the emergency coordination
messages in [x/emergency/types/types.go](../x/emergency/types/types.go).
Height-only propose/vote-revert messages remain wire-compatible but fail
closed in the message server. The ante decorator also admits a recovery lane
for:

- an expedited SDK governance proposal containing exactly one
  `MsgSoftwareUpgrade` or `MsgCancelUpgrade`;
- votes, weighted votes, and deposits only after the target proposal is read
  from consensus state and verified to have that exact shape.

It does **not** stop block production. The app still runs `x/upgrade` PreBlock
and nearly every standard and custom BeginBlock and EndBlock in the order
defined in [app/app.go](../app/app.go). Heights, timeouts, rewards, staking,
slashing, governance tallies, IBC processing, and module timers can continue.

It still blocks general standard governance, custom governance,
incident-record, IBC-containment, account-freeze, and parameter messages.
Even emergency `MsgUpdateParams` is not allowlisted. The narrow recovery lane
does not admit mixed proposals, legacy content, authz wrappers, or unrelated
expedited actions.

`max_halt_duration_blocks` is now an escalation deadline only. Crossing it
emits operator telemetry but never reopens transaction admission. Resume
requires a new guardian ceremony bound to the active quarantine ID and a
lowercase SHA-256 commitment to the reviewed recovery manifest. Concretely,
`recovery_manifest_sha256` is the independently anchored
`transition_sha256` of the verified `RECOVERY_READY` journal head immediately
before `MsgProposeResume`. After the proposal is accepted, append
`RECOVERY_READY -> ACTIVATING` with the transaction and ceremony as evidence;
this avoids a circular hash.

At ceremony creation, consensus state snapshots the exact sorted
Guardian/council electorate, each member's effective power, the quorum, and
the minimum distinct-voter requirement. Council expiry, staking changes, and
parameter amendments cannot shrink an in-flight denominator. A
non-terminal pre-hardening ceremony without this snapshot is terminalized;
malformed legacy resume state and missing quarantine linkage are repaired in
BeginBlock while admission remains quarantined.

`DECISION`:

- Call this state `CONSENSUS_QUARANTINE`, never “consensus halt.”
- Never claim that autonomous module effects are frozen.
- Elapsed time, restart, or supervisor behavior is never resume authorization.
- Treat a malformed or unlinked recovery commitment as a failed resume; keep
  signers or edges stopped as independently required.
- If governance or incident transactions will be needed after containment,
  either anchor them before quarantine when safe, or record the incident in
  the external manifest and anchor it on-chain after recovery.

### 3.2 Signer/consensus stop

`OBSERVATION`: stopping enough consensus signing power stalls CometBFT. This is
an operator action, not an `x/emergency` state transition.

`DECISION`:

- For a suspect validator key, disable the signer before investigating.
- Preserve `priv_validator_state.json`, consensus WAL, logs, binary digest, and
  node configuration. Never delete signer state to make a node start.
- Never run the same consensus key on two machines.
- A chain-wide stop is confirmed only when the consensus axis is `STALLED`
  from multiple independent observations, not when one process exits.
- No on-chain vote can be expected after a real consensus stop. Recovery
  authority must therefore be expressed in the signed operations manifest and
  by the required operator quorum.

### 3.3 Scheduled `x/upgrade` halt

`OBSERVATION`: the application calls module PreBlock first in
[app/abci.go](../app/abci.go), preserving the SDK `x/upgrade` mechanism. A
scheduled plan at height `H` behaves as follows:

1. An old binary without the named handler reaches H, writes
   `data/upgrade-info.json`, and halts its ABCI state transition.
2. H has not committed. H−1 is the last committed state.
3. The new binary with the named handler starts from H−1, executes the handler
   at H, and may commit H.
4. Once H commits, the migration is part of canonical history.

If the running binary already contains the named handler, it may execute the
handler at H without an observable process stop. A test that schedules a
handler already present in the same binary is therefore not evidence of a
real old-to-new binary handoff.

The implementation lives in [app/upgrades.go](../app/upgrades.go);
store-loader registration occurs before state load in
[app/app.go](../app/app.go). The SDK 0.53 / IBC-Go 10 path additionally uses
the strict tests and H−1 guards in
[app/upgrades_sdk053_ibc10_test.go](../app/upgrades_sdk053_ibc10_test.go).

`DECISION`:

- H−1 MUST have a verified, restorable cold snapshot before activation.
- Before H commits, the preferred recovery is a corrected binary for the same
  agreed plan. Cancellation is still possible only if governance can act
  before H.
- `--unsafe-skip-upgrades H` is an exceptional off-chain decision for nodes
  still on H−1. It MUST be authorized by the same chain-wide recovery manifest
  and used consistently. It is not a rollback.
- After H commits, never restart the old binary and never use
  `--unsafe-skip-upgrades H` to pretend H did not happen. Repair forward with a
  new named upgrade, or invoke the fork/re-genesis process.

Upstream behavior is documented in the Cosmos SDK
[application-upgrade guide](https://docs.cosmos.network/sdk/v0.53/build/building-apps/app-upgrade)
and [Cosmovisor guide](https://docs.cosmos.network/sdk/v0.53/build/tooling/cosmovisor).

### 3.4 Fork or export/re-genesis

A fork/re-genesis is a new canonical-history decision. It is appropriate only
when committed state cannot safely be repaired forward, when histories have
diverged, or when a state rewrite cannot be expressed as a deterministic
in-place migration.

It requires:

- a last common committed height, block hash, app hash, and signed commit;
- deterministic export and rewrite tooling;
- a complete supply and module-account reconciliation;
- a new, unique chain identity or revision plan;
- explicit IBC client/channel/packet treatment on both sides;
- a signed genesis hash and validator set;
- retirement or controlled migration of old consensus keys;
- the fork/re-genesis quorum in section 5.

It is never called “rollback.” CometBFT explicitly requires a unique chain ID
for a new chain; see its
[command and genesis guide](https://docs.cometbft.com/v0.38/core/using-cometbft).

## 4. Current operational observations and limitations

These observations describe the audited source. Live-network observations
MUST still be re-queried.

1. Standard SDK governance is the authority for `x/upgrade`, consensus
   parameters, and most privileged messages in [app/app.go](../app/app.go).
   Custom `x/gov` uses the separate `x/zerone_staking` electorate, but its
   software-upgrade execution authority is retired: upgrade LIPs cannot enter
   executable voting, legacy voting LIPs resolve `FAILED`, and app wiring does
   not give it an upgrade keeper. Historical attached plans remain queryable
   for audit/export compatibility. This removes the plan-overwrite/cancel race;
   only standard SDK governance can schedule or cancel canonical `x/upgrade`
   state.
2. `x/emergency` guardian power comes from custom tier-4 staking or the
   temporary genesis council, not CometBFT voting power. Guardian quorum,
   governance quorum, consensus commit power, and release approval are four
   different authorities.
3. Emergency CLI exposes evidence-bound `propose-resume` and `vote-resume`.
   It intentionally does not register `propose-revert`: the legacy height-only
   message always fails with `ErrUnsafeRevertDisabled`.
4. No on-chain instruction claims arbitrary rollback. The actual
   `zeroned rollback` command rewinds local application and Comet state by one
   height only; it is not an arbitrary target-height recovery tool.
5. IBC rate limits are keyed by the exact channel/denomination tuple.
   [x/ibcratelimit/keeper/quota.go](../x/ibcratelimit/keeper/quota.go) explicitly
   allows traffic when the limiter is disabled or no tuple is configured.
   [x/ibcratelimit/ibc_middleware.go](../x/ibcratelimit/ibc_middleware.go)
   passes malformed/non-transfer packet data through to the underlying IBC
   module. Rate limiting is therefore fail-open for unconfigured tuples and is
   not a global IBC circuit breaker.
6. The live-state review reported an expired IBC client. That statement is
   time-sensitive, not a source-code invariant. Every operation MUST record
   the current client IDs, statuses, trusting-period margin, channels, packet
   commitments, and relayer state. An expired client may already prevent new
   proofs while leaving escrow, vouchers, commitments, and counterparty state
   that still require reconciliation. A long stop can also cause a currently
   active counterparty client to expire.
7. `BuildChainVersionReport` in
   [app/upgrade_registry.go](../app/upgrade_registry.go) is a useful internal
   report but is not currently exposed as a production CLI, gRPC, or REST
   endpoint.

## 5. Roles, authority, and quorum

### 5.1 Roles

| Role | Responsibility | Must not be its only counterparty |
|---|---|---|
| Incident commander | Declares incident phase, maintains manifest sequence, coordinates communications. | Evidence custodian |
| Consensus operator | Stops/starts signer and node, reports voting power and signed height. | Release builder |
| Release manager | Freezes source, builds/stages artifacts, records provenance. | Sole release verifier |
| Release verifier | Independently rebuilds or verifies digest, provenance, tests, and migration rehearsal. | Release builder |
| Governance coordinator | Submits/cancels the standard SDK upgrade proposal and records its execution. | Sole consensus operator |
| IBC/relayer lead | Stops owned relayers, inventories clients/channels/packets/escrow, coordinates counterparties. | Sole supply verifier |
| Evidence custodian | Preserves logs, DB snapshots, headers, signatures, and hashes without modifying originals. | Incident commander |
| Supply verifier | Reconciles total supply, module accounts, escrow, bonded pools, and migration deltas. | Code author of the balance-changing migration |

One person may hold multiple roles during the custodial exception, but the
manifest MUST show the collapse explicitly.

### 5.2 Default authorization

| Action | Required authorization |
|---|---|
| Stop one suspect signer | Its operator alone; notify immediately. |
| Planned upgrade freeze | Release manager plus independent release verifier. |
| Schedule or cancel planned upgrade | Standard SDK governance plus an attested release manifest. |
| Seal release `SCHEDULED -> STAGED` | Reports from operators representing more than two-thirds voting power, plus release verifier. |
| Resume after an incident | Operators representing more than two-thirds voting power, incident commander, release verifier, and IBC lead when IBC is affected. |
| Retire/replace a suspect signer | On-chain validator-set authority plus the affected operator and evidence custodian. |
| Fork/re-genesis | Operators representing more than two-thirds voting power, governance/social decision, release verifier, supply verifier, and IBC lead. |
| Reopen IBC | IBC lead, supply verifier, incident commander, and counterparty acknowledgement where their state is affected. |

More than two-thirds is a readiness and restart threshold, not proof that the
release is correct. Independent artifact and state verification remain
mandatory.

### 5.3 Current 1/1 custodial exception

The current custodial topology permits one active consensus signer and one
custodial operating domain to satisfy actions that would otherwise require a
validator-power quorum.

This exception MUST be represented as:

```text
authorization_mode: custodial_exception_1_of_1
consensus_fault_tolerance: none
independent_validator_quorum: unavailable
```

Under this exception:

- Stopping the sole signer stalls the chain; compromising it compromises all
  live consensus authority.
- A `1/1` approval MUST NOT be described as decentralized, Byzantine-fault
  tolerant, independent, or multisignature approval.
- Planned operations still require frozen source, artifact hashes, H−1
  recovery evidence, migration rehearsal, and a separately recorded release
  review. When an independent person cannot authorize, obtain a
  non-authorizing external witness signature and disclose that it is only a
  witness.
- Emergency operations MAY proceed with the sole operator to limit damage.
  Publish the signed manifest as soon as disclosure is safe.
- The exception does not permit unsigned artifacts, silent re-genesis,
  generic rollback, automatic resume, copying a consensus key to parallel
  hosts, or reopening IBC without reconciliation.

The exception retires only after consensus control and privileged transaction
authority are held by independently controlled parties with tested threshold
procedures. Adding node processes under one custodian does not retire it.

## 6. Operations manifest and artifact contract

Every planned upgrade and every P0/P1 incident MUST have a signed,
hash-chained transition journal plus a content-addressed evidence dossier.
This is the control plane that still works when consensus does not. Neither
artifact contains private keys, seed phrases, credentials, or unredacted
exploit material.

### 6.1 Hash-chained body

Use the stdlib-only verifier in
[tools/zerone-ops](../tools/zerone-ops/README.md). Its machine schema is
`zerone.ops.transition/v1` and its two forward-only lanes are:

```text
release:  RUNNING -> PREPARING -> RELEASE_FROZEN -> SCHEDULED -> STAGED
          -> HALTED_AT_H -> MIGRATING_H -> OBSERVING -> ACCEPTED

incident: RUNNING -> ASSESSING -> CONTAINING -> [SAFETY_STOPPED]
          -> RECOVERY_DESIGN -> RECOVERY_READY -> ACTIVATING
          -> OBSERVING -> CLOSED
```

`CANCELLED`, `RECOVERY_FAILED`, and `FORK_CHOICE` are explicit terminal
branches. There is no backward, automatic-resume, generic-rollback, or
automatic fork-choice edge.

The verifier requires the exact compact JSON encoding documented in its
README: fixed schema field order, sorted lists, no insignificant whitespace
or trailing newline, and `[]` rather than `null`. This is intentionally a
Zerone schema encoding, **not** RFC 8785. Do not reserialize a sealed
transition with a generic JSON canonicalizer.

Each transition binds:

- sequence, previous transition hash, `from`/`to`, event, actor, and UTC time;
- chain, incident, and/or release identity;
- committed checkpoint height, block ID hash, and AppHash;
- immutable upgrade name, height, plan-info, binary, image, provenance, SBOM,
  and state-manifest hashes;
- content-addressed evidence;
- Ed25519 approvals and an immutable `trust_policy_sha256`;
- its own SHA-256 with `transition_sha256` empty during hashing.

Provision a canonical `zerone.ops.trust-policy/v1` document outside the
journal. It binds chain/incident/release identity, required roles, role
separation, exact trusted identity/key/power tuples, and the integer
voting-power quorum. Record its SHA-256 through a separately protected channel.
Every command requires both the file and that independently obtained digest;
the policy cannot be replaced inside a v1 journal.

Generate an approval statement, collect its Ed25519 signature, seal, and then
verify the complete ordered journal:

```shell
go run ./tools/zerone-ops approval-statement \
  --trust-policy /protected/zerone-ops-policy.json \
  --trust-policy-sha256 <independently-anchored-policy-sha256> \
  --input transition-draft.json \
  --role release-verifier \
  --identity did:zrn:reviewer-1 \
  --public-key <64-lowercase-hex> \
  --power 0

go run ./tools/zerone-ops seal \
  --trust-policy /protected/zerone-ops-policy.json \
  --trust-policy-sha256 <independently-anchored-policy-sha256> \
  --input approved-transition-draft.json > 0001.json

go run ./tools/zerone-ops verify \
  --trust-policy /protected/zerone-ops-policy.json \
  --trust-policy-sha256 <independently-anchored-policy-sha256> \
  --chain-id zerone-1 \
  --release-id <release-id> \
  --binary-sha256 <64-lowercase-hex> \
  --head-sha256 <independently-anchored-64-lowercase-hex> \
  0001.json 0002.json
```

The evidence dossier records full observations, rejected alternatives,
private disclosure metadata, expiry/abort criteria, raw RPC responses, and
operator notes. Store each dossier object by SHA-256 and reference it from the
journal. RFC 8785 MAY be used for a separate dossier JSON convention, but it
is not the journal encoding.

Corrections append a new transition; they never overwrite an old one. Anchor
the journal head through a separately protected channel and store transitions,
signatures, and evidence in at least two independently controlled locations,
one append-only or write-once. The tool verifies pinned keys and power; it
does not discover the live validator set. Bind the independently captured
validator-set snapshot as evidence and provision the policy from that reviewed
snapshot.

### 6.2 Release artifact set

An upgrade cannot reach `RELEASE_FROZEN` without:

- upgrade name and target H;
- source repository, exact commit, signed tag if used, and clean-tree proof;
- Go version, build flags, dependency lock, `go mod verify` result, and module
  inventory;
- per-OS/architecture binary SHA-256;
- binary `version` output and embedded commit/version metadata;
- SBOM and vulnerability-scan result;
- signed build provenance;
- source and output of every code generator;
- module version map before and after;
- store additions/deletions and migration order;
- migration test and multi-node rehearsal results;
- H−1 snapshot/chunk manifest and app hash;
- pre/post supply and module-account reconciliation;
- IBC clients, channels, commitments, escrow, voucher supply, and rate-limit
  tuple inventory;
- configuration and environment deltas;
- explicit abort and recovery procedure.

Use a pinned Cosmovisor version. Keep
`DAEMON_ALLOW_DOWNLOAD_BINARIES=false` for validators and stage the exact
attested binary at:

```text
<DAEMON_HOME>/cosmovisor/upgrades/<upgrade-name>/bin/zeroned
```

Do not use a mutable symlink or an artifact fetched at H. Automatic download
is not a substitute for prior verification. The upstream Cosmovisor guide
also warns about download-time failure and documents checksum-bearing URLs.

Signed provenance SHOULD use
[Sigstore verification](https://docs.sigstore.dev/cosign/verifying/verify/)
and a [SLSA provenance](https://slsa.dev/spec/v1.0/provenance) predicate.
Where those tools are unavailable, use a documented offline detached-signature
scheme and retain the public verification material.

For `sdk-0.53-ibc-10`, plan `info` is reserved for the canonical legacy-IBC
keyset commitment required by [OPEN_CRYPTO_SDK.md](standards/OPEN_CRYPTO_SDK.md)
and [the manifest tool](../tools/ibc-v10-keyset-manifest/). Keep the general
release-manifest hash in the external operations manifest; do not replace the
strict on-chain payload with generic checksum prose.

`sdk-0.53-ibc-10` is also the single coordinated activation for the
transaction-quarantine, recovery-authorization, immutable-electorate,
signer-policy, and custom-upgrade-retirement changes in this release. The
binary intentionally does not register a standalone
`upgrade-incident-operations-v1` handler: the SDK handler migrates emergency
version 1 to version 2 in the same transaction as the legacy-store removal, so
a later handler requiring emergency version 1 would be impossible. Never
schedule that retired draft name.

## 7. Classifying planned design changes

### 7.1 Change classes

| Class | Examples | Delivery |
|---|---|---|
| Edge-only | Reverse proxy, dashboard, logging, alert routing. | Rolling change with rollback at the edge. |
| Node-local | Pruning, snapshots, sentry peers, RPC exposure. | Canary then rolling change; never change consensus-signing identity casually. |
| On-chain parameter | Consensus params, module params, IBC transfer/rate-limit settings. | Standard governance with before/after invariant checks. |
| Consensus-affecting binary | Ante, proposal processing, vote extensions, Begin/End/PreBlock, state writes, auth, gas, keeper wiring, dependency behavior. | New named `x/upgrade`, even if no store schema changes. |
| State-shape migration | Store key, protobuf persistence, module version, account permission, IBC store migration. | New named upgrade, store loader, deterministic migration, H−1 recovery rehearsal. |
| State rewrite/history change | Irreparable committed corruption or incompatible canonical histories. | Fork/re-genesis, never an in-place “rollback.” |

`DECISION`: additive protobuf or code-only changes are not automatically safe
for rolling deployment. If old and new binaries can produce different
FinalizeBlock results, proposal decisions, vote extensions, authentication,
gas outcomes, or persisted state, use a named coordinated upgrade.

### 7.2 Canonical governance path

Use standard SDK governance with
`/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade`, as shown in
[UPGRADES.md](UPGRADES.md). Use `MsgCancelUpgrade` before H when cancellation
is required.

Custom `x/gov` software-upgrade execution is retired. Historical plan records
remain visible, but upgrade LIPs cannot become executable votes and legacy
voting LIPs fail without calling `ScheduleUpgrade`. Do not represent a custom
LIP as upgrade authorization.

Only one standard SDK `x/upgrade` plan may be canonical. The operations
manifest MUST record `zeroned query upgrade plan`, the native governance
proposal, and any historical custom upgrade-LIP/plan records. Those custom
records are reconciliation evidence only; they are not a second executable
authority.

## 8. Planned upgrade procedure

### Gate 0 — intent and ownership

- Allocate an upgrade name that has never been used.
- Classify the change using section 7.
- Name release manager, verifier, governance coordinator, consensus operators,
  IBC lead, evidence custodian, and supply verifier.
- Open manifest sequence 1 and state the 1/1 exception if applicable.
- Define a target window, but do not schedule H.

### Gate 1 — freeze source and migration

- Freeze source commit and handler name.
- Register the handler and any store loader in
  [app/upgrades.go](../app/upgrades.go).
- Add lineage and test parity in
  [app/upgrade_registry.go](../app/upgrade_registry.go) and
  [tests/cross_stack/upgrade_e2e_test.go](../tests/cross_stack/upgrade_e2e_test.go).
- Run migrations, permission reconciliation, and deterministic postconditions.
- Never edit an old named handler after it has run on any network.

Reject if any migration reads wall-clock time, external services, random or
unsorted iteration, mutable files, or live network state.

### Gate 2 — build and attest

- Build on a clean, isolated worker from the frozen commit.
- Independently rebuild or verify provenance.
- Produce the artifact set in section 6.
- Pin Cosmovisor; never install `@latest` as part of an activation.
- Verify the staged binary hash again after transfer and on every node.
- Keep release state `PREPARING` while any artifact, review, or rehearsal
  remains incomplete.

### Gate 3 — state rehearsal

Use a recent, trusted production snapshot or raw database copy taken after a
clean stop. Never copy a live LevelDB/IAVL directory and call it a backup.

Rehearse:

1. old binary reaches H with H−1 committed;
2. old binary lacks the new handler and cannot commit H;
3. new binary loads the H−1 database;
4. handler and store loader run exactly once at H;
5. independent nodes produce the same H app hash and module version map;
6. restart at H and later heights is idempotent;
7. a deliberately failing handler leaves the recoverable committed state at
   H−1;
8. a corrected binary can resume from that H−1 state;
9. supply, module accounts, IBC state, and packet accounting reconcile.

`RunUpgradeHandlerForTests` and [scripts/upgrade-test.sh](../scripts/upgrade-test.sh)
are useful component tests, but a same-binary run is not this handoff drill.

Seal `PREPARING -> RELEASE_FROZEN` only after every build/attestation item and
every Gate 3 rehearsal passes. The release binding is immutable after this
transition.

### Gate 4 — schedule with lead time

- Submit the standard SDK governance plan only after `RELEASE_FROZEN`.
- H MUST leave enough time for governance completion, independent review,
  staging, H−1 backup, IBC preparation, and cancellation.
- Record proposal ID, plan name, H, plan-info digest, governance close time,
  and query evidence.
- Re-query the plan after every governance transition.
- Cancel if the on-chain fields do not exactly match the attested manifest.
- After the exact native plan is confirmed on chain, seal
  `RELEASE_FROZEN -> SCHEDULED`.

### Gate 5 — IBC and supply preparation

Before H:

- inventory every IBC client and status, connection, channel, exact packet
  denomination, rate-limit tuple, sequence, commitment, acknowledgement,
  timeout, transfer escrow account, and voucher supply;
- calculate time remaining before each active client’s trusting period expires;
- arrange counterparty and relayer coverage for the upgrade window;
- stop new cross-chain exposure using an enforceable on-chain control before
  entering consensus quarantine;
- wait for or explicitly account for in-flight packets;
- record total supply by denomination and all relevant module-account balances.

Stopping Zerone-operated relayers is not sufficient: IBC relaying is
permissionless, and packets already committed remain obligations. The local
rate limiter permits unconfigured channel/denomination tuples.

The IBC-Go 8.1-to-10 upstream migration requirements are documented in the
[IBC-Go v10 migration guide](https://ibc.cosmos.network/v10/migrations/v8_1-to-v10/).

### Gate 6 — stage and declare readiness

Each consensus operator reports:

- node ID and validator consensus address;
- current binary digest and version;
- staged binary digest and version;
- Cosmovisor digest/version/config;
- free disk and snapshot validation result;
- H−1 snapshot destination and restore estimate;
- current height, block hash, app hash, and peer health;
- signer state and last signed height;
- IBC/relayer readiness where applicable.

Seal `SCHEDULED -> STAGED` only after reports representing more than
two-thirds voting power match the manifest. Under the 1/1 exception, record
that the sole report represents 100% custodial power but zero independent
fault tolerance.

#### Fly validator identity gate

The transitional Fly profiles use
`deploy/fly-validator-entrypoint-common.sh`. They require the validator and
P2P node identities from a regular mounted file or a base64 runtime secret,
with an independently provisioned lowercase SHA-256 for each. The entrypoint
copies both candidates into a private `/tmp` directory and completes all of
these checks before `zeroned init` or any validator-home mutation:

- exact Comet Ed25519 JSON types and decoded key sizes;
- derivation of the Ed25519 public key from the private seed;
- equality of the derived public key, private-key public suffix, and explicit
  validator public key;
- equality of the validator address and the SHA-256-truncated public-key
  address;
- byte-exact candidate SHA-256.

The runtime also requires `EXPECTED_VALIDATOR_ADDRESS`,
`EXPECTED_NODE_ID`, `EXPECTED_PRIV_VALIDATOR_KEY_SHA256`, and
`EXPECTED_NODE_KEY_SHA256` from the separately reviewed identity manifest.
The entrypoint derives the uppercase Comet validator address and lowercase P2P
node ID from the private-key public components, compares both identities and
both file digests, and removes all source and expected-identity variables
before the daemon `exec`. A key plus a checksum stored in the same secret
control plane is not independent authenticity evidence.

Each network wrapper also freezes the raw public genesis SHA-256. Before key
installation or initialization, the shared entrypoint requires the image
genesis to be a regular non-symlink file, verifies its digest and chain ID, and
repeats those checks on the persisted genesis on every boot. A changed genesis
therefore cannot silently turn the same validator key into a signer on another
history.

An existing home must contain the validator key, validator signing state, and
P2P node key as regular non-symlink files. Both persisted keys are validated
again and must match the provisioned digests; signing state must have a
canonical non-negative height/round and valid Comet signing step. A missing
member, malformed state, or identity drift fails closed. If genesis is absent
but the home is non-empty, the entrypoint refuses to reinterpret it as a fresh
volume. Never delete `priv_validator_state.json` or the genesis file to bypass
this unit check.

The key source, base64 value, digest, and expected-identity variables are
removed from the environment before any child process runs and the private
candidate directory is removed before the daemon `exec`. Treat a failed
identity gate as an operator incident; key rotation must use the reviewed
rotation procedure and must preserve double-signing safety.

### Gate 7 — activation at H

- Freeze unrelated deployments and configuration changes.
- Keep incident, release, IBC, and consensus channels staffed.
- Observe H−1 from multiple RPC/full-node sources.
- Record the H−1 header, commit, app hash, binary hashes, and snapshot hash.
- Observe `upgrade-info.json` and Cosmovisor without editing them.
- When the old binary has stopped before committing H, seal
  `STAGED -> HALTED_AT_H`. When the frozen new binary starts the deterministic
  height-H migration, seal `HALTED_AT_H -> MIGRATING_H`.
- Do not repeatedly restart a failing binary. A failure at or after the H
  boundary seals the applicable `HALTED_AT_H`/`MIGRATING_H` transition to
  `RECOVERY_FAILED` and opens an incident journal.
- Do not authorize normal traffic merely because a process restarted.

#### Exact-height immutable Fly image handoff

The Fly profiles run the binary directly and therefore cannot use a mutable
Cosmovisor path to pre-stage a replacement. Fly restart policy must first be
armed to `no` while the old binary is healthy and the observer height is still
strictly below H. Otherwise the intentional `x/upgrade` exit can trigger an
old-binary restart loop at H.

Retain a byte-exact canonical plan-evidence artifact containing at least the
on-chain plan name, height, `info`, query height, and source. Set
`UPGRADE_PLAN_SHA256` to that artifact's lowercase SHA-256. Set
the expected validator address, P2P node ID, and both key-file digests from an
independently reviewed and signed identity manifest outside the Fly control
plane. The public values are not Fly secrets.

Build the current and target Fly images with reviewed `VERSION`, full
40-character lowercase `COMMIT`, and positive `SOURCE_DATE_EPOCH` build
arguments. The Dockerfiles reject omitted/malformed provenance, verify the
embedded binary version/commit, use digest-pinned bases and a fixed Debian
snapshot, and emit OCI source/revision/version labels. Record the resulting
immutable image digest and inspected labels in the signed release manifest
before arming.

Well before H, record an independent pre-arm height/AppHash and run:

```shell
export FLY_APP=zerone-validator
export FLY_MACHINE_ID=<exact-validator-machine-id>
export FLY_VOLUME_ID=<exact-encrypted-validator-volume-id>
export FLY_CURRENT_IMAGE_REF=registry.fly.io/zerone-validator@sha256:<current-image-digest>
export CHAIN_ID=zerone-1
export UPGRADE_NAME=sdk-0.53-ibc-10
export UPGRADE_HEIGHT=<H>
export UPGRADE_PLAN_EVIDENCE_PATH=<reviewed-current-plan.json>
export UPGRADE_PLAN_SHA256=<canonical-plan-evidence-sha256>
export PRE_ARM_HEIGHT=<independently-observed-height-below-H>
export PRE_ARM_APP_HASH=<lowercase-pre-arm-apphash>
export EXPECTED_VALIDATOR_ADDRESS=<40-uppercase-hex>
export EXPECTED_NODE_ID=<40-lowercase-hex>
export EXPECTED_PRIV_VALIDATOR_KEY_SHA256=<64-lowercase-hex>
export EXPECTED_NODE_KEY_SHA256=<64-lowercase-hex>
export EXPECTED_GENESIS_SHA256=<reviewed-raw-genesis-json-sha256>
export OBSERVER_RPC_URL=https://<independent-observer-rpc>
export OBSERVER_API_URL=https://<independent-observer-rest>
export ARMED_BY=<independent-operator-identity>
export ARMED_AT=<reviewed-UTC-RFC3339-time>
export FLY_ARM_EVIDENCE_OUTPUT=<new-arm-evidence.json>

CURRENT_IMAGE_DIGEST="${FLY_CURRENT_IMAGE_REF##*@sha256:}"
export FLY_ARM_CONFIRMATION="arm-no-restart:${FLY_APP}:${FLY_MACHINE_ID}:${FLY_VOLUME_ID}:${CHAIN_ID}:${CURRENT_IMAGE_DIGEST}:${UPGRADE_NAME}:${UPGRADE_HEIGHT}:${UPGRADE_PLAN_SHA256}:${PRE_ARM_HEIGHT}:${PRE_ARM_APP_HASH}:${EXPECTED_VALIDATOR_ADDRESS}:${EXPECTED_NODE_ID}:${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}:${EXPECTED_NODE_KEY_SHA256}:${EXPECTED_GENESIS_SHA256}"
./deploy/fly-arm-exact-height-handoff.sh
```

The arming script confirms the exact running old image and encrypted volume,
semantically compares the live plan with the digest-bound evidence, and changes
only `restart.policy` to `no`. It compares the rest of the canonical Machine
config before and after, re-observes a height below H, and writes a new
`zerone.fly-upgrade-arm-evidence/v1` artifact. Review and externally sign that
artifact, then record its SHA-256. Conduct this early enough that a controlled
old-binary process restart cannot collide with H. The exact H−1 activation
report does not exist yet and therefore is deliberately not claimed by this
early arm artifact.

At H, independently capture the exact old-binary upgrade-needed log and
`upgrade-info.json`. The old Machine must exit and remain stopped by itself;
the handoff script never stops it. Against a stopped independent full node or
isolated copy at the same H−1 AppHash, run:

```shell
export LAST_COMMITTED_HEIGHT=<H-minus-1>
export LAST_COMMITTED_APP_HASH=<lowercase-H-minus-1-apphash>
zeroned verify-activation-prestate \
  --home <stopped-or-isolated-observer-home> \
  --expected-chain-id "${CHAIN_ID}" \
  --expected-height "${LAST_COMMITTED_HEIGHT}" \
  --expected-app-hash "${LAST_COMMITTED_APP_HASH}" \
  > activation-preflight.json
```

The required v3 report binds the chain ID, raw genesis digest, exact H−1
height/AppHash, on-chain plan and plan-info digest, empty unsafe-skip set,
complete-IAVL safety checks, isolated `application.db`-only manifest, exact
local `upgrade-info.json` digest/equality, dry-run, and its own canonical report
digest. Record the raw report-file SHA-256 separately.
Seal a reviewed
`zerone.fly-upgrade-exit-evidence/v1` artifact binding the arm-evidence digest,
activation-preflight report digest, plan, current image, identity/genesis
tuple, H−1 height/AppHash, attempted H, and the actual hashes of the captured
log and upgrade-info files. The reviewed log must contain
exactly one matching upgrade-needed exit, and the evidence records
`old_binary_exit_count: 1`. Only after observers agree H−1 is the last commit
and the old binary attempted but did not commit H, run:

```shell
export FLY_IMAGE_REF=registry.fly.io/zerone-validator@sha256:<target-image-digest>
export FLY_CONFIG_PATH=deploy/mainnet/fly.toml
export FLY_CONFIG_SHA256=<reviewed-fly-config-sha256>
export ACTIVATION_PREFLIGHT_REPORT_PATH=<activation-preflight.json>
export ACTIVATION_PREFLIGHT_REPORT_SHA256=<raw-report-file-sha256>
export UPGRADE_ARM_EVIDENCE_PATH=<reviewed-arm-evidence.json>
export UPGRADE_ARM_EVIDENCE_SHA256=<reviewed-arm-evidence-sha256>
export UPGRADE_EXIT_EVIDENCE_PATH=<reviewed-exit-evidence.json>
export UPGRADE_EXIT_EVIDENCE_SHA256=<reviewed-exit-evidence-sha256>
export OLD_BINARY_EXIT_LOG_PATH=<captured-old-binary.log>
export UPGRADE_INFO_EVIDENCE_PATH=<captured-upgrade-info.json>
export LAST_COMMITTED_HEIGHT=<H-minus-1>
export LAST_COMMITTED_APP_HASH=<lowercase-H-minus-1-apphash>
export ATTEMPTED_UPGRADE_HEIGHT=<H>
export EXPECTED_UPGRADE_APP_HASH=<rehearsed-lowercase-H-apphash>

CURRENT_IMAGE_DIGEST="${FLY_CURRENT_IMAGE_REF##*@sha256:}"
IMAGE_DIGEST="${FLY_IMAGE_REF##*@sha256:}"
export FLY_HANDOFF_CONFIRMATION="${FLY_APP}:${FLY_MACHINE_ID}:${FLY_VOLUME_ID}:${CHAIN_ID}:${CURRENT_IMAGE_DIGEST}:${IMAGE_DIGEST}:${FLY_CONFIG_SHA256}:${UPGRADE_NAME}:${UPGRADE_HEIGHT}:${UPGRADE_PLAN_SHA256}:${ACTIVATION_PREFLIGHT_REPORT_SHA256}:${UPGRADE_ARM_EVIDENCE_SHA256}:${UPGRADE_EXIT_EVIDENCE_SHA256}:${LAST_COMMITTED_HEIGHT}:${LAST_COMMITTED_APP_HASH}:${ATTEMPTED_UPGRADE_HEIGHT}:${EXPECTED_UPGRADE_APP_HASH}:${EXPECTED_VALIDATOR_ADDRESS}:${EXPECTED_NODE_ID}:${EXPECTED_PRIV_VALIDATOR_KEY_SHA256}:${EXPECTED_NODE_KEY_SHA256}:${EXPECTED_GENESIS_SHA256}"
./deploy/fly-exact-height-handoff.sh
```

The script requires exactly `LAST_COMMITTED_HEIGHT=H-1` and
`ATTEMPTED_UPGRADE_HEIGHT=H`. It recomputes both the raw preflight-file digest
and the v3 embedded canonical report digest, content-validates its
chain/genesis/plan/H−1/AppHash/readiness/check tuple, validates the arm and exit
evidence, semantically compares the live REST plan to the plan artifact, checks
the exact upgrade-needed log and `upgrade-info.json`, and binds every
evidence/config/image/state/identity digest into the manual confirmation.
Independent RPC and REST observations must both report the expected chain.

The runtime gate requires exactly one key source plus one colocated transport
digest for each validator/node key. Plaintext base64 Machine config and
non-deployed secrets are rejected. Those colocated digests provide corruption
detection only; authenticity comes from the separately reviewed expected key
digests and derived validator address/node ID in the confirmation. The update
places those public expected values into the Machine config, and the entrypoint
derives and compares both identities before initialization or daemon start.

The control-plane checks use Fly JSON and require one exact Machine on the
confirmed current image with the confirmed encrypted `/data` volume, already
stopped, with restart policy `no`. Before any update or start, the script
canonicalizes the current stopped Machine config with only `.restart` removed,
recomputes its SHA-256, and requires exact equality with
`machine_config_sha256` sealed by the pre-H arm evidence. Any intervening
command, init, environment, service, mount, or key-source drift therefore
fails without a target mutation. The update uses the digest-bound Fly config,
the immutable target image, explicit expected identity values, `--restart no`,
and `--skip-start`. While stopped, the script requires the exact target image,
identity tuple, same encrypted volume, no-restart policy, and P2P-only
port-26656 service/port policy before it starts the Machine.

After start, the script checks that exact tuple again and leaves restart policy
`no` through verification. It queries independent RPC header H+1—which commits
the AppHash produced by H—and the immutable `x/upgrade` applied-plan height,
requiring exact RPC/REST/header chain IDs, exact header height, expected H
AppHash, and applied height H. A later restart-policy change is a separate
reviewed action. If verification fails, keep admission closed and enter
incident review. A fail-stop trap is armed before `fly machine start`; every
non-success exit or signal after that point requests a graceful stop, escalates
to `SIGKILL` if the stopped/no-restart state is not confirmed, and is disarmed
only after independent H verification succeeds. These scripts validate hashes
and schemas but do not create observer independence or cryptographic
signatures; archive externally signed evidence and the transition journal with
their output.

### Gate 8 — verification and reopen

After H commits, seal `MIGRATING_H -> OBSERVING`, then verify:

- the H block hash, app hash, and commit agree across independent nodes;
- all validators report the attested binary;
- `zeroned query upgrade applied <name>` reports H;
- module versions and migration markers match the manifest;
- supply and module accounts match the expected delta exactly;
- no unexpected mint, burn, escrow, or delegation movement occurred;
- IBC clients/channels/commitments and voucher/escrow accounting reconcile;
- rate limits exist for every intended channel/denomination tuple;
- transaction, query, vote-extension, restart, and snapshot probes pass;
- no new consensus evidence, missed-signature anomaly, panic loop, or
  privileged-action anomaly is present.

Reopening uses a new signed manifest. IBC and public write APIs MAY reopen
later than ordinary block production. There is no automatic reopen.
Seal `OBSERVING -> ACCEPTED` only after every check and the declared
observation window pass. Any failed check seals `OBSERVING -> RECOVERY_FAILED`.

## 9. Hostile-event decision matrix

| Event | Initial axes | Immediate action | Recovery gate | Never do |
|---|---|---|---|---|
| Public RPC/API DDoS | Consensus often `PRODUCING`; admission `OPEN` | `EDGE_QUARANTINE`, rate-limit/disable broadcast endpoints, move traffic through sentries, preserve query/status paths | Stable consensus, protected RPC capacity, no state exploit | Stop consensus merely because public RPC is unavailable |
| P2P/validator DDoS | `PRODUCING` or `STALLED`; signer may remain `ACTIVE` | Hide validator behind sentries/private peers, isolate public RPC, add clean sentries | Canonical peer view and stable rounds from multiple sources | Expose validator address or blindly restart signers |
| Transaction exploit | `PRODUCING`, admission `OPEN` | Edge quarantine immediately; use consensus quarantine if peer-injected txs remain dangerous; use signer stop if autonomous block processing is unsafe | Fixed attested binary/param, state reconciliation, explicit resume | Claim `x/emergency` stopped blocks or module timers |
| BeginBlock/EndBlock/consensus bug | May be `DIVERGENCE` or unsafe `PRODUCING` | Stop signers before another commit; preserve H−1/H evidence | Deterministic replay from last common commit on corrected binary | Rely on transaction quarantine |
| App-hash divergence | `DIVERGENCE` | Stop affected signers, freeze DBs/logs, identify last common signed commit | Rehearsal proving one deterministic result; fork process if conflicting committed histories exist | Pick a database by node count or newest height |
| Consensus stall without divergence | `STALLED` | Query status and consensus state across nodes; check power, peers, rounds, and binary digest | More than two-thirds healthy signers agree on canonical last commit and restart manifest | Delete WAL or signer state as a first response |
| Consensus-key suspicion/equivocation | Signer `SUSPECT` | Stop and isolate signer; preserve key-state metadata and evidence; coordinate safe power removal only if an independently safe set already controls more than two-thirds | Double-sign-safe validator-set transition when independently safe power can commit it; a possibly copied sole 1/1 key requires the signed fork/re-genesis branch in 11.4.1 | Ask a suspect sole key to authorize its replacement, or copy key/state to a second live host |
| Governance/authority key suspicion | Consensus may remain `PRODUCING` | Alert on privileged actions, cancel malicious pending plan if governance can act, edge-quarantine compromised endpoints; stop consensus if a dangerous activation is imminent | Authority rotation/threshold control, review every action since compromise | Trust an apparently valid privileged message solely because it is authorized |
| Release or build compromise before H | Release `PREPARING`, `RELEASE_FROZEN`, `SCHEDULED`, or `STAGED` | Remove artifact from staging without destroying evidence; seal `CANCELLED` while checkpoint < H and cancel any native plan; publish revocation hash | New release journal/name or explicitly superseding manifest, clean rebuild, repeat all gates | Replace bytes under the same attested digest/path silently |
| Bad new binary before H commits | `HALTED_AT_H` or `MIGRATING_H`, last commit H−1 | Stop restart loop, preserve database and logs, seal `RECOVERY_FAILED`, use a freshly authorized corrected binary or chain-wide skip decision | Multi-node H−1 replay and fresh incident/recovery manifest | Call it a post-H rollback |
| Bad state after H commits | `OBSERVING`, possibly `PRODUCING`/`DIVERGENCE` | Seal `RECOVERY_FAILED`; contain transactions/IBC; stop signers if damage continues | Forward repair by new upgrade, or fork/re-genesis quorum | Restart old binary or skip H |
| Single-node DB corruption | Canonical chain elsewhere `PRODUCING`; signer `SUSPECT`/`STOPPED` | Isolate node, preserve original disk, compare canonical app hash | Verified snapshot/state sync from independent sources | Copy a live DB or make corrupted node canonical |
| Fleet-wide DB corruption | `STALLED` or `DIVERGENCE` | Preserve every independent copy; determine last common commit | Verified snapshot restore or fork/re-genesis | Rewind each node independently |
| IBC exploit or accounting anomaly | Usually `PRODUCING` | Stop owned relayers, quarantine transfer txs, inventory packets/escrow/vouchers/clients, contact counterparties | Supply and packet reconciliation plus explicit IBC reopen manifest | Assume stopped relayers or missing rate-limit tuples are a circuit breaker |
| IBC client expired/near expiry | `PRODUCING`; IBC path impaired | Record client and counterparty state, stop creating obligations on affected path, coordinate client recovery/replacement | Verified client status and counterparty acknowledgement | Treat expired client as absence of escrow or packet liability |
| Unexpected supply change | `PRODUCING` or unsafe | Quarantine mint/transfer exposure; stop signers if autonomous minting continues; snapshot and reconcile | Deterministic correction/upgrade with exact expected delta | Apply an ad hoc compensating mint or burn |
| Deliberate fork/re-genesis | `FORK` by decision | Stop old signers/relayers, publish last common commit and signed genesis manifest | Section 13 procedure and section 14 acceptance | Reuse old chain identity ambiguously or conceal discarded history |

## 10. Incident execution

### 10.1 First five minutes

1. Assign an incident ID and commander.
2. Preserve raw observations before changing state.
3. Report all four axes.
4. If a signer or signing host is suspect, stop it.
5. If public ingress is the problem, edge-quarantine it without disrupting
   private consensus paths.
6. If transactions from any peer can exploit state, use consensus quarantine
   or stop enough signers.
7. Stop owned relayers when IBC may be affected, while explicitly noting that
   this is not a consensus IBC stop.
8. Start operations-manifest sequence 1 and record the authorization mode.
9. Establish a public status channel and a sealed evidence channel. Never put
   secrets or an unpatched exploit recipe in the public/on-chain record.

If consensus quarantine would block the on-chain incident transaction, keep
the external hash-chained record and anchor its hash on-chain after safe
admission resumes.

### 10.2 Evidence package

Collect without modifying originals:

- UTC clock source and host time offset;
- `/status`, `/net_info`, `/consensus_state`, and, when access-controlled,
  `/dump_consensus_state`;
- block, header, commit, block results, app hash, and validator set around the
  event;
- current and staged binary hashes, version output, process arguments, and
  environment/config digest;
- application, CometBFT, proxy, sentry, relayer, Cosmovisor, and system logs;
- consensus WAL and last-sign metadata;
- DB engine, filesystem, mount, snapshot, disk-health, and crash evidence;
- governance proposals, upgrade plan, privileged actions, emergency
  ceremonies, consensus params, and slashing/evidence queries;
- IBC clients, channels, packet commitments/acks/timeouts, escrow, vouchers,
  relayer versions, and exact rate-limit tuples;
- total supply, all balances relevant to the incident, and module-account
  permissions/balances.

CometBFT documents its production databases, WAL, RPC diagnostics, corruption
risks, and DDoS posture in
[Running in production](https://docs.cometbft.com/v0.38/core/running-in-production).

### 10.3 Status messages

Every external update states:

- what was observed and at which height/time;
- what was decided and by whom;
- all four axes;
- user/IBC impact;
- what remains unknown;
- next decision gate and expected update time;
- current manifest hash.

Do not promise a resume time before acceptance gates pass.

## 11. Domain-specific containment and recovery

### 11.1 IBC

Before containment, distinguish:

- disabling Zerone-operated relayers;
- blocking transaction ingress at Zerone edges;
- consensus rejection of transfer/admin transactions;
- on-chain transfer send/receive settings;
- per-channel/per-denomination rate limits;
- client expiry or freezing;
- channel closure.

Only the latter on-chain states are shared consensus controls, and their
governance messages may be blocked by current consensus quarantine.

Required IBC snapshot:

- client ID, type, latest height, status, trusting period, and counterparty;
- connection and channel ends on both chains;
- next send/recv/ack sequence;
- packet commitments, receipts, acknowledgements, and timed-out obligations;
- transfer escrow address and balance per channel/denom;
- native supply and every IBC voucher trace/supply;
- limiter enabled flag and exact configured tuples;
- relayer identities, versions, routes, and last successful relay.

An unconfigured denom or channel is allowed by the current limiter. Packet
denominations may be trace-prefixed and direction-dependent; inventory the
exact strings observed by middleware rather than assuming `uzrn`.

After a long stop, re-query both local and counterparty clients before
restarting relayers. State sync or local chain recovery does not itself repair
an expired counterparty light client. IBC security follows light-client and
packet semantics in the canonical
[Interchain Standards](https://github.com/cosmos/ibc).

### 11.2 Supply and economic state

The supply verifier records, by denomination:

- bank total supply;
- sum of account balances;
- bonded and not-bonded staking pools;
- fee collector and distribution/community pools;
- vesting, research, development, reward, liquidity, bridge, and escrow module
  accounts;
- IBC native escrow and voucher supply;
- migration-specific minted, burned, transferred, or newly inaccessible
  amounts;
- configured issuance caps and their counters.

The expected post-state must be an equation, not “looks reasonable.” Every
non-zero delta names the handler statement or transaction that caused it.
Do not correct a discrepancy manually. Preserve it, reproduce it, and express
the correction as a deterministic authority message, named upgrade, or
fork/re-genesis rewrite with its own manifest.

### 11.3 Release and dependency supply chain

- Freeze dependency versions; never resolve `latest` during an incident.
- Verify source tag/commit signatures and repository ownership.
- Run `go mod verify` and retain its output.
- Compare the built module graph to the attested graph.
- Build on at least two independent workers when possible.
- Verify binary digest after every transport and on the execution host.
- Keep Cosmovisor auto-download disabled on validators.
- Reject unsigned replacement archives, mutable URLs without a checksum, and
  plan metadata that does not match the manifest.
- If a builder or signing key is compromised, revoke the artifact by digest,
  not only by version name.

### 11.4 Consensus and governance keys

- Keep transaction/governance keys separate from consensus signing keys.
- Use OS/hardware/threshold-backed key storage; never use a test keyring in
  production.
- Protect the consensus signer behind private connectivity and sentries.
- Treat the consensus key plus its last-sign state as one migration unit.
- On suspicion, set signer `SUSPECT`, stop it, preserve evidence, and enter
  the applicable validator-identity replacement path. On a 1/1 set, a
  possibly copied consensus key requires signed fork/re-genesis, never an
  in-chain transition.
- Do not “test” a recovered consensus key against a live chain.
- Move privileged transaction authority to threshold control as the custodial
  exception is retired.

#### 11.4.1 One-validator key replacement

Zerone has two independent staking records:

- Cosmos SDK `x/staking` controls the CometBFT validator set and consensus
  power. Its CLI is `zeroned tx staking ...`.
- `x/zerone_staking` records Proof-of-Truth participation, tiers, and Guardian
  eligibility. Its CLI is `zeroned tx zerone_staking ...`; registering there
  does **not** add a CometBFT validator.

There is no in-place `x/staking` transaction that changes an existing
validator's consensus public key. A consensus-key replacement is a new SDK
validator identity followed by an observed validator-set transition. It is
not a file replacement.

Classify the affected key before choosing a path:

| Key condition | Required path |
|---|---|
| P2P `node_key.json` only is exposed; consensus signer and transaction authorities are proven safe | Stop the node, create a fresh P2P identity on a fresh home/volume, update signed node-ID and peer manifests, then move the unchanged consensus key **and its unchanged signing state together** while the old process remains stopped. |
| Consensus key is being replaced routinely and compromise has been ruled out with evidence | The controlled in-chain validator-set transition below may be used. |
| The sole consensus key may have been copied, used by a hostile party, or cannot be proved exclusive | Stop it. The 1/1 chain cannot safely authorize its own replacement: every transition block would still depend on the suspect key. Use the signed fork/re-genesis path from the last agreed safe checkpoint. |
| The SDK validator operator key is unavailable or suspect, so its stake cannot be moved safely | Do not invent an authority override. Use a governed, rehearsed state migration while a safe chain still exists, otherwise fork/re-genesis. |
| A custom Zerone Guardian/operator identity is suspect | Standard `x/staking` replacement is insufficient. Retire or migrate the separate `x/zerone_staking` record and immutable emergency snapshots through an explicit governed migration or fork design. |

The distinction between “copied” and “not copied” is a security conclusion,
not an operator convenience. Unexpected host access, an untrusted image with
key access, conflicting signatures, missing custody evidence, or an
unaccounted backup means **suspect**.

The historical Zerone validator images contained consensus and P2P private
keys. Consequently, the controlled transition below is **not** the assumed
remediation for the currently deployed 1/1 networks. It becomes eligible only
if an independent custody review proves the consensus key never left trusted
control; missing registry/build-host/access evidence selects the suspect-key
fork branch.

For a Fly P2P-only replacement, “fresh volume” means a new volume pre-seeded
offline from a byte-consistent, cleanly stopped **full validator
home/volume**, including application and Comet databases/state/WAL, genesis,
and the Cosmovisor tree—not from a generic application snapshot. Never pair a
current signing-state file with older application or Comet state. If the node
is rebuilt through state sync instead, withhold the consensus signer/key until
catch-up and perform a separately reviewed signing-state handoff at the fixed
height. Install the unchanged `priv_validator_key.json` and byte-exact
`priv_validator_state.json` together with a fresh `node_key.json` before the
replacement can sign. Update the expected node ID and P2P-key digest in the
signed manifest and Machine configuration while leaving the validator address
and validator-key digest unchanged. The old and replacement Machines must
never run concurrently.

##### Controlled in-chain transition

This path is prohibited for a suspected sole consensus-key compromise. It is
available only when the old consensus and operator keys are proven controlled
long enough to commit the transition, no conflicting history exists, and the
new operator has independently authorized liquid or redelegatable stake.
Application transaction admission must also be `OPEN`: Zerone's emergency
quarantine deliberately rejects `MsgCreateValidator` and
`MsgBeginRedelegate`. Do not reopen admission merely to make rotation
convenient; while quarantine must remain active, use the exact recovery
upgrade lane for a reviewed deterministic migration or select fork/re-genesis.

1. Seal an operations transition naming the old validator operator and
   consensus addresses, the last observed signing height, the new operator,
   the exact power-transfer plan, and abort conditions. Record independent
   headers, commits, and app hashes.
2. Build a new signer on a fresh host and fresh home/volume. Generate fresh
   `priv_validator_key.json` and `node_key.json`; never overwrite either file
   on the old home and never reset or copy the old
   `priv_validator_state.json` onto the new consensus identity. Sync the new
   node while its new consensus address is absent from the active validator
   set. Initialize only an empty home; `init --overwrite` is prohibited.
   Replace the generated genesis with the byte-exact reviewed canonical
   genesis and verify its chain ID and signed SHA-256 before any sync.
3. Produce a separately signed identity manifest containing chain ID, genesis
   SHA-256, binary/image digest, SDK operator and validator-operator
   addresses, Comet consensus public key/address, P2P node ID, both key-file
   SHA-256 values, and the fresh signing-state tuple. Under the Fly profile,
   provision the new values before attaching a **new** volume; changing
   runtime secrets on an existing volume is not rotation because the
   persisted-identity gate will correctly reject the mismatch.

   Capture the source values directly from the fresh home with a private
   umask, then reconcile every redundant address/public-key representation
   before signing the manifest:

   ```bash
   umask 077
   zeroned comet show-validator --home "${NEW_HOME}" \
     > validator-pubkey.json
   zeroned comet show-address --home "${NEW_HOME}"
   zeroned comet show-node-id --home "${NEW_HOME}"
   jq -er '.address' \
     "${NEW_HOME}/config/priv_validator_key.json"
   shasum -a 256 \
     "${NEW_HOME}/config/priv_validator_key.json" \
     "${NEW_HOME}/config/node_key.json"
   jq -S . "${NEW_HOME}/data/priv_validator_state.json"
   ```

   The Comet address from the command, key JSON, and independently derived
   public key must agree. Hash the canonical genesis and the binary/image in
   the same evidence package.
4. Rehearse the exact transition on a clone at the same validator powers.
   Calculate a post-transition set in which the replacement alone has
   strictly more than two-thirds of total voting power after integer
   rounding. This SDK build computes consensus power as
   `floor(tokens / 1_000_000)` (`sdk.DefaultPowerReduction`), but the
   authoritative gate uses the returned Comet integer powers and current
   active set after `max_validators` selection:
   `new_power * 3 > total_power * 2`. Do not use floating-point percentages.
   Merely adding a small second validator does not let a 1/1 chain stop the
   old signer. At a fixed rehearsal height, inventory the old delegator's
   existing redelegation and unbonding entries. A transitive redelegation,
   maximum-entry limit, or slash exposure carried through the redelegation
   period can invalidate a seemingly sufficient transfer; never infer
   eligibility from the displayed bonded balance alone.

   Pin the account shape and stake inventory to one retained height/header:

   ```bash
   export H='<retained-fixed-height>'
   export OLD_ACCOUNT='<old-zrn-account-address>'
   export OLD_VALOPER='<old-zrnvaloper-address>'
   export NODE_RPC='<private-trusted-http-rpc>'
   export REST='<private-trusted-rest>'

   curl --fail --silent --show-error \
     "${NODE_RPC}/block?height=${H}" > checkpoint-block.json
   zeroned query auth account "${OLD_ACCOUNT}" \
     --height "${H}" --node "${NODE_RPC}" --output json
   zeroned query staking delegations "${OLD_ACCOUNT}" \
     --height "${H}" --node "${NODE_RPC}" --output json
   zeroned query staking unbonding-delegations "${OLD_ACCOUNT}" \
     --height "${H}" --node "${NODE_RPC}" --output json
   curl --fail --silent --show-error \
     --header "x-cosmos-block-height: ${H}" \
     --dump-header redelegations.headers \
     "${REST}/cosmos/staking/v1beta1/delegators/${OLD_ACCOUNT}/redelegations?pagination.limit=100" \
     > redelegations.json
   ```

   Require the REST response height to equal H and its pagination
   `next_key` to be empty, or retrieve and hash every subsequent page. Query
   each known path again with
   `zeroned query staking redelegation OLD_ACCOUNT OLD_VALOPER DST_VALOPER
   --height H`. Bind all outputs to the checkpoint block/app hash.
5. Create the new SDK validator from the safe transaction key using the
   Cosmos SDK v0.53 JSON-file form:

   ```bash
   export NEW_HOME='<fresh-signer-home>'
   export NEW_OPERATOR_KEY='<safe-new-operator-key>'
   export NEW_SELF_BOND='<reviewed-amount>uzrn'
   export CHAIN_ID='<exact-chain-id>'
   export NODE_RPC='<private-trusted-rpc>'

   zeroned comet show-validator --home "${NEW_HOME}" > validator-pubkey.json
   jq -n \
     --slurpfile pubkey validator-pubkey.json \
     --arg amount "${NEW_SELF_BOND}" \
     '{
       pubkey: $pubkey[0],
       amount: $amount,
       moniker: "replacement-validator",
       identity: "",
       website: "",
       security: "",
       details: "reviewed validator-key replacement",
       "commission-rate": "0.05",
       "commission-max-rate": "0.20",
       "commission-max-change-rate": "0.01",
       "min-self-delegation": "1"
     }' > validator.json

   zeroned tx staking create-validator validator.json \
     --from "${NEW_OPERATOR_KEY}" \
     --chain-id "${CHAIN_ID}" \
     --node "${NODE_RPC}" \
     --gas auto --gas-adjustment 1.4 \
     --fees '<reviewed-nonzero-fee>uzrn'
   ```

   Treat `validator-pubkey.json` and `validator.json` as ceremony artifacts:
   hash them, compare the public key to the identity manifest, and do not use
   shell substitution that obscures the exact bytes reviewed. The new
   operator account must already hold the declared self-bond and fees through
   an authorized supply path.
6. If the replacement needs stake held by the old operator, submit the
   rehearsed SDK redelegation from the still-safe old transaction key:

   ```bash
   zeroned tx staking redelegate \
     '<old-zrnvaloper-address>' \
     '<new-zrnvaloper-address>' \
     '<reviewed-power-transfer>uzrn' \
     --from '<safe-old-operator-key>' \
     --chain-id "${CHAIN_ID}" \
     --node "${NODE_RPC}" \
     --gas auto --gas-adjustment 1.4 \
     --fees '<reviewed-nonzero-fee>uzrn'
   ```

   Do not infer the account type from a repository ceremony design. A
   read-only query on 2026-07-30 showed the deployed 1/1 mainnet validator
   operator as a Cosmos `BaseAccount`; the unreleased five-validator
   `scripts/mainnet-ceremony.sh` design instead models permanent-locked
   genesis stake. SDK redelegation moves staking shares and recalculates the
   intended validator power in the same application transition while its
   redelegation entry retains slashability bookkeeping; it does not wait for
   the unbonding period. Comet still sees the resulting validator update only
   at B+2. Falling below the old validator's minimum self-delegation removes
   that validator's indexed power through the same delayed update. Rehearsal
   must choose explicitly between a partial transfer leaving less than
   one-third residual power and a full/below-minimum removal.

   Permanent-locked stake may be redelegatable while remaining non-spendable,
   but redelegation does not change its owner: the old locked account key can
   later redelegate those shares again. It is therefore not durable operator
   custody rotation. If that transaction key is suspect or lost, use fresh
   independently controlled stake plus explicit retirement, a governed state
   migration while the chain is safe, or fork/re-genesis. Either account
   shape and the exact transition must pass rehearsal.
7. Cosmos SDK v0.53.8 uses `ValidatorUpdateDelay = 1`: a staking update
   returned while committing block B is expected to enter the signing set at
   B+2. The old sole signer must remain controlled and online through B+1,
   while the replacement must be ready before B+2. Do not act on timing alone.
   At fixed heights, query the SDK validator record and Comet validator set
   from independent nodes, calculate voting power from the returned integers,
   and inspect commits:

   ```bash
   zeroned query staking validator '<new-zrnvaloper-address>' \
     --node "${NODE_RPC}" --output json
   curl --fail --silent --show-error \
     "${NODE_RPC}/validators?height=<fixed-height>" > validators.json
   curl --fail --silent --show-error \
     "${NODE_RPC}/commit?height=<fixed-height>" > commit.json
   ```

   The gate passes only when the new consensus address is present with the
   rehearsed power, its signatures appear in consecutive commits, all
   independent nodes agree, and the new validator alone has strictly more
   than two-thirds of total current power. Record the response hashes and
   heights.
8. Stop and isolate the old signer. Observe further commits without it before
   removing residual SDK stake. Preserve its key file, signing state, WAL,
   database, logs, image, and hashes read-only as incident evidence; do not
   start it again on either history.
9. Reconcile the distinct `x/zerone_staking` identity. A new
   `zerone_staking register-validator` record starts with new participation
   history and does not transfer Guardian status. The current module has no
   consensus-key update and no general deregistration transaction. Its
   `consensus_pubkey` field is only required to be non-empty; the module does
   not decode it, prove possession, bind it to the SDK validator, or emit
   Comet validator updates. Treat it as non-authoritative metadata. Do not
   leave a suspect custom Guardian active or claim that the SDK transition
   rotated it. If that role must move, use a named deterministic migration
   with explicit before/after records, immutable-snapshot treatment, quorum
   analysis, and tests—or choose fork/re-genesis.
10. Seal old identity `RETIRED` only after the old signer is offline, the
    replacement has carried the chain through the observation window, SDK
    and custom records are reconciled, peers/sentries use the new node ID,
    monitoring sees no old-key signatures, and the signed evidence package is
    complete.

##### Suspect sole-key fork boundary

For a possible 1/1 consensus-key compromise, stop at the last independently
agreed safe commit and follow section 13. Classify consensus-key custody
separately from SDK operator, governance, and custom Guardian custody:

- If only the consensus key is suspect and an independent assessment proves
  the existing SDK operator `PASS/RETAIN`, the narrow
  [`tools/fork-genesis`](../tools/fork-genesis/) compiler may rotate that one
  consensus key. It requires one bonded/unjailed validator, exact stopped
  `H+1` export and checkpoint bindings, empty IBC, no pending governance,
  upgrade, or genesis transactions, a new chain revision, and a quarantined
  start. It preserves the proven-safe operator and refuses every broader
  rewrite.
- If the SDK operator, governance authority, or custom Guardian is suspect or
  unknown, a full-identity rewrite must remove and replace every affected
  privilege and reconcile staking, distribution, slashing, evidence,
  `zerone_staking`, governance, emergency, and supply state. The narrow
  compiler deliberately cannot do this.

The currently deployed legacy 1/1 images may have exposed operator authority,
and that custody has not been independently proved. Production therefore
starts `SUSPECT/NO_GO`; the existence of the narrow compiler does not authorize
its use until the operator assessment is `PASS/RETAIN`. A full-identity
compiler is not present in this repository, so a failed or unknown operator
assessment remains a release blocker.

There is no safe shortcut consisting of replacing
`priv_validator_key.json`, deleting `priv_validator_state.json`, restoring a
snapshot under the new key, or asking the suspect key to sign “one final”
transition. An unsupported rewrite profile must be implemented, reviewed, and
reproduced; its absence is a blocker, not permission to hand-edit genesis.

### 11.5 Database corruption

For a single corrupt node while a canonical chain exists:

1. Stop its signer and node.
2. Preserve the original disk read-only.
3. Record the canonical trusted height, header, commit, and app hash from
   independent sources.
4. Restore a verified application snapshot or use state sync.
5. Verify the restored app hash against a light block and replay before
   enabling signing.

CometBFT state sync uses light-client verification of the snapshot-height app
hash; see the upstream
[state-sync specification](https://docs.cometbft.com/v0.38/spec/p2p/legacy-docs/messages/state-sync).

For fleet-wide corruption or conflicting committed histories, stop and move
to fork/re-genesis analysis. Never independently rewind validators and hope
their app hashes converge.

### 11.6 DDoS and network attack

- Validators MUST use sentry architecture and MUST NOT expose validator RPC
  publicly.
- Public RPC/API nodes sit behind connection, request-size, concurrency,
  method, and per-origin limits.
- Disable or heavily restrict broadcast, unsafe, dump, and unbounded query
  endpoints on public nodes.
- Preserve authenticated operator access to status and narrow diagnostic
  queries.
- Keep alternate sentries, seeds, private peer paths, and out-of-band operator
  communications tested.
- Scale or replace disposable edge nodes; do not move consensus keys into
  public infrastructure.
- If a validator cannot determine the canonical peer set safely, stop its
  signer rather than risk equivocation.

The deployable reference boundary is
[`deploy/topology`](../deploy/topology/): two disposable P2P-only sentries, a
separate read-only query edge, unique encrypted volumes, digest-pinned images,
fresh zero-power identities, and signer restart policy `no`. Its clean
`legacy-full-node` image target may extract only the exact hashed `zeroned`
binary from a historical image; it never inherits that image's filesystem
layers or keys. Extraction is containment, not source provenance, and requires
an external binary-origin decision.

Move a live validator behind that boundary only after both sentries are
independently synced and probed. Remove public signer services and addresses
in a separate controlled transition; never apply a sentry template to the
signer or couple edge deployment with signer restart. A hostile sentry/query
node is stopped and replaced from a reviewed digest on a new volume—its old
identity and volume are not restarted.

Upstream recommends sentry nodes and warns against public validator RPC:
[CometBFT production guidance](https://docs.cometbft.com/v0.38/core/running-in-production)
and [sentry-node model](https://docs.cometbft.com/v0.38/spec/p2p/legacy-docs/node).

## 12. No automatic resume and no generic rollback

### 12.1 Resume rule

No timer, process restart, Cosmovisor symlink switch, on-chain
`StatusNormal`, block production, or majority node count authorizes resume.

Resume requires a new signed manifest proving:

- root cause is understood enough to bound the risk;
- selected binary/config/state is attested;
- canonical last commit and app hash are agreed;
- signer states are safe;
- recovery was rehearsed from the preserved state;
- supply and IBC acceptance gates passed;
- the required quorum approved;
- monitoring and abort triggers are active.

The on-chain `recovery_manifest_sha256` MUST equal the independently anchored,
successfully verified `RECOVERY_READY` transition head. It is not a free-form
document digest chosen inside the resume transaction.

### 12.2 Local one-height rollback

`zeroned rollback` is only a narrow local repair for a node whose Comet and
application state disagree at the latest height. It rewinds one height; it is
not the emergency module’s target-height revert.

It MAY be considered only when:

- the exact bad local transition is the latest height;
- the canonical network history is known;
- the node signer is stopped;
- original DB and signer metadata are preserved;
- replay of that one height on the intended binary is deterministic;
- the manifest authorizes the exact node, height, and expected app hash.

It MUST NOT be looped to reach an arbitrary height, applied independently
across validators, or used to erase a committed network event.

### 12.3 Forward-only boundary

- Before H commits: H−1 remains the canonical committed state. Correct the
  activation or, under explicit chain-wide authorization, skip/cancel it.
- After H commits: repair with a new forward transition.
- If committed history itself must be abandoned: use a disclosed
  fork/re-genesis with a new signed genesis and IBC plan.

There is no middle category called “deep rollback.”

## 13. Fork/re-genesis procedure

1. Set consensus axis `FORK` by explicit decision and stop old-chain signers.
2. Stop relayers and public transaction ingress on every controlled endpoint.
3. Identify the last common committed height using headers and commits, not
   local database height alone.
4. Prove the validator process and platform restart route are stopped. Freeze
   the source read-only; create a
   [`validator-home-manifest`](../tools/validator-home-manifest/) from that
   stopped evidence and hash candidate databases, exports, logs, binaries, and
   control-plane records.
5. Select the export/rewrite base and state why every later block is retained
   or abandoned. Bind the height, block ID, app hash, last block time, signed
   commit, validator set, and exact compact SDK export bytes.
6. Seal separate custody findings for consensus, P2P, SDK operator, governance,
   custom Guardian, build, registry, and recovery authorities. Missing evidence
   is `UNKNOWN`, never `PASS`.
7. Select an implemented rewrite profile. For
   `consensus-key-only`, the SDK operator must independently pass
   `RETAIN_PROVEN_SAFE`; otherwise stop because the available compiler does not
   support the required identity rewrite.
8. Independently pin the incident, custody assessment, rewrite policy,
   compiler executable digest, fresh key, target chain revision/time, and two
   reproducer tuples with distinct control domains and signing keys.
9. Run [`fork-genesis`](../tools/fork-genesis/) in both domains. Require
   byte-identical target genesis and output digest. Each reproducer signs an
   external attestation over its exact compiler-report file digest; a report
   self-hash alone is not execution proof.
10. Reconcile total supply, module accounts, vesting, staking, governance,
    evidence, incidents, custom modules, and every IBC obligation. The narrow
    profile only accepts empty IBC and its two explicit v8-to-v10 empty-state
    schema migrations.
11. Validate the target with the target application and
    [tools/genesis-check](../tools/genesis-check/). Initialize and commit it
    twice in isolated environments, then compare genesis, initial validator
    set, app hash, quarantine state, module digest inventory, and supply.
12. Run the deterministic
    [`operations-rehearsal`](../tools/operations-rehearsal/) fault suite
    against pinned, signed collector evidence. Exercise stale evidence,
    duplicate execution, wrong binary, wrong height, same-volume restore,
    overlapping signers, process restart, tampered artifacts, and divergent
    reproduction.
13. Seal the release around the matching genesis, both compiler reports,
    rehearsal, stopped-home manifest, fresh-volume proof, topology, journal,
    and signed approvals. Then sign the final fork choice; it is intentionally
    downstream of compilation to avoid a hash cycle.
14. Run [`validator-recovery-gate`](../tools/validator-recovery-gate/) from
    exact pinned files. Any invalid identity derivation, untrusted approval,
    evidence omission, profile mismatch, or reproduction mismatch is `NO_GO`.
15. Obtain the fork/re-genesis authorization from section 5. A tool result
    cannot substitute for the required human/institutional authority.
16. Only after both gates pass, create wholly new stopped volumes and isolated
    Machines with restart policy `no`. Compare genesis, app hash, validator
    identity, signer state, binary digest, and network exposure before
    connecting validators.
17. Keep public writes and IBC closed through the observation window. Mark old
    signer identities `RETIRED` where appropriate, retain old evidence
    read-only, and monitor both histories for accidental signing.

The external manifest chain preserves the incident evidence that an on-chain
rewind or rewritten genesis would otherwise remove. Re-anchor its latest hash
on the recovered chain when safe.

## 14. Drills and acceptance gates

Run release-specific drills before every upgrade and the full incident suite
at least quarterly. A drill fails if evidence is asserted rather than
captured.

### 14.1 Required drills

| Drill | Acceptance |
|---|---|
| Real old/new `x/upgrade` | Old binary commits H−1 but not H; attested new binary executes once; independent nodes agree at H and after restart. |
| Failed migration | New binary fails before H commit; original H−1 database remains restorable; corrected binary succeeds without ad hoc state edits. |
| Artifact tamper | One-bit binary/archive/manifest change is rejected before staging or execution. |
| Native plan conflict/cancel | A pre-existing or wrong native `x/upgrade` plan blocks `SCHEDULED`; historical custom plan records are reported as non-authoritative; cancellation and a checkpoint below H are confirmed. |
| Consensus quarantine | Normal txs fail; emergency coordination and the exact expedited upgrade lane work; Begin/End effects continue; crossing the escalation deadline does not reopen admission. |
| Signer stop/resume | Required power stops; chain state is classified correctly; no signer double-signs; resume requires a new manifest. |
| 1/1 custodial | Sole signer stop stalls chain; all reports carry the custodial exception and do not claim independent quorum. |
| Key compromise | Suspect signer is isolated, evidence preserved, replacement/retirement rehearsed without parallel key use. |
| Stopped-home evidence | Validator PID/executable/home, platform restart policy, source filesystem identity, and stop interval are externally bound; unrelated PID, stale stop proof, hard-linked key, and same-volume destination all fail. |
| DB restore | Clean-stop snapshot restores on a new host; app hash verifies; replay reaches canonical height before signing. |
| Containment topology | Two sentries and a query edge use fresh zero-power identities and distinct volumes; signer has no public service or auto-restart; hostile edge replacement never starts the signer. |
| DDoS | Public writes are quarantined while private consensus and operator diagnostics remain available through sentries. |
| IBC containment | Owned relayers stop, all exact tuples/clients/packets/escrow are inventoried, an unconfigured denom is detected as fail-open, and reopen is separately authorized. |
| Expired client | Operators identify the expired/near-expiry path and exercise counterparty/client recovery without inventing packet or escrow state. |
| Supply invariant | Pre/post equations detect an injected one-unit discrepancy and block verification. |
| Evidence trust | A self-hashed but unsigned collector envelope, an unpinned signer, missing inventory membership, and a replayed stale observation all fail closed. |
| Recovery gate tamper | Invalid Bech32/derived identity, forged GO report, mismatched compiler output, untrusted role approval, wrong rewrite profile, and altered selected file all return `NO_GO`. |
| Fork/re-genesis | Two policy-authorized signed reproductions match exactly; unique chain identity, supply equality, IBC treatment, quarantine, old-key retirement, and manifest re-anchoring pass. |

### 14.2 Release acceptance

A planned upgrade may be scheduled only when:

- all gates 0–3 passed;
- every artifact is content-addressed and signed;
- no unresolved critical/high security finding affects the path;
- the recovery time objective was met in rehearsal;
- the H−1 restore was tested, not merely created;
- supply and IBC baselines are complete;
- authority and the 1/1 exception are represented honestly.

It may reach the H boundary only when:

- release is `STAGED`;
- more than two-thirds power reported the exact staged digest, or the explicit
  1/1 exception is active;
- cancellation remains possible for the declared safety window;
- H−1 evidence and operator staffing are current.

It may seal `OBSERVING -> ACCEPTED` only when every Gate 8 check and the
declared observation window pass. “Blocks are moving” is one observation, not
upgrade acceptance.

## 15. Fast decision sequence

1. **Is a signer or signing host suspect?** Stop it and preserve evidence.
2. **Is one canonical history still committing?** If no, classify `STALLED`,
   `DIVERGENCE`, or `FORK` before touching a DB.
3. **Is harm introduced only through public ingress?** Edge-quarantine.
4. **Can any peer introduce the harmful transaction?** Consensus-quarantine.
5. **Can harm occur without a transaction?** Stop enough signers.
6. **Is this planned H and has H committed?** If no, H−1 is the recovery
   boundary. If yes, repair forward.
7. **Is IBC affected or can the stop outlast a trusting period?** Stop owned
   relayers, inventory both sides, and keep IBC closed.
8. **Is committed history irreparable?** Begin fork/re-genesis; do not loop
   local rollback.
9. **Have recovery gates and quorum passed?** If no, do not resume.

## 16. Upstream references

- [Cosmos SDK 0.53 application upgrades](https://docs.cosmos.network/sdk/v0.53/build/building-apps/app-upgrade)
- [Cosmos SDK 0.53 Cosmovisor](https://docs.cosmos.network/sdk/v0.53/build/tooling/cosmovisor)
- [Cosmos SDK upgrades and store migrations](https://docs.cosmos.network/sdk/latest/guides/upgrades/upgrade)
- [CometBFT 0.38 production operations](https://docs.cometbft.com/v0.38/core/running-in-production)
- [CometBFT 0.38 validator signing](https://docs.cometbft.com/v0.38/spec/consensus/signing)
- [CometBFT 0.38 state sync](https://docs.cometbft.com/v0.38/spec/p2p/legacy-docs/messages/state-sync)
- [CometBFT sentry-node model](https://docs.cometbft.com/v0.38/spec/p2p/legacy-docs/node)
- [IBC-Go v8.1 to v10 migration](https://ibc.cosmos.network/v10/migrations/v8_1-to-v10/)
- [Canonical Interchain Standards](https://github.com/cosmos/ibc)
- [Sigstore artifact verification](https://docs.sigstore.dev/cosign/verifying/verify/)
- [SLSA provenance v1.0](https://slsa.dev/spec/v1.0/provenance)
- [RFC 8785 JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785)
