# Zerone validator guide

This guide describes safe preparation. It is not an invitation to join a
network and it is not deployment authority.

## Current release posture

- Canonical source:
  [`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
- The Go module path remains `github.com/zerone-chain/zerone`; that historical
  import path is not the clone URL.
- No tagged binary release is part of the 2026-07-29 consolidation.
- No validator deployment is authorized by source publication.
- `zerone-2` remains **NO-GO** until its signed ceremony and authority
  requirements are complete.
- Current CI contains a protected manual OIDC component-signing job, and the
  checked-in authority verifier cryptographically validates exact v0.3 bundles
  against a pinned local Sigstore root, identity, source commit, transparency
  log, and observer-time policy. Those source paths do not create the three
  real bundles, reviewed trusted root, or signed authority graph; fixture
  checks are not production provenance.

Do not download from old `zerone-chain/zerone` or `nickkpope/zerone` release
URLs. Do not build a production validator from a moving branch.

## Build a review candidate

For development or rehearsal, pin the exact reviewed commit:

```bash
git clone https://github.com/cambridgetcg/zerone-core.git
cd zerone-core
git checkout --detach <reviewed-commit>

make build
./build/zeroned version --long
git rev-parse HEAD
```

Go 1.25.14 exactly, `make`, `git`, `jq`, and a supported C toolchain are
required. A production release must additionally bind the source commit,
binary digest, build platform, SBOM/provenance, vulnerability decision, and
immutable image digest in its signed release packet.

Never infer production provenance from `zeroned version` alone. Verify the
embedded revision and the release-bound hashes.

## Roles are distinct

- **Non-signing full node / observer:** independently replays and checks the
  application and Comet consensus history. It has zero consensus voting power,
  submits no consensus votes, and needs no funded account or account keyring.
  Fresh local Comet P2P and unused consensus identity files are normal; never
  copy a validator's identities or signing state into it.
- **Custom verifier:** an application-level participant that submits the
  release-specific verification messages. Its account registration, fees and
  stake are separate from Comet validator membership. Running a full node does
  not register that participant or authorize transactions.
- **Genesis custodian:** prepares/reviews the genesis and ceremony evidence
  under the selected release policy. Custody of release inputs does not grant
  consensus membership or permission to run a second signer.
- **Consensus validator:** holds an admitted consensus identity and voting power,
  signs proposals/votes, and requires independently reviewed custody, fencing,
  admission and upgrade operations. Do not conflate this role with a custom
  verifier or with public query service.

## Isolated local replay and restart rehearsal

Use the existing disposable harness, not `scripts/localnet.sh` or an existing
user-local chain home:

```bash
# Clean reviewed candidate; builds its own binaries and disposable chain.
bash scripts/local-consensus-rehearsal.sh --keep

# Edited development candidate only: explicitly NON-FINAL provenance.
bash scripts/local-consensus-rehearsal.sh --keep --allow-dirty

# Limited diagnostic, NOT a substitute for the default integrated suite.
bash scripts/local-consensus-rehearsal.sh --observer-only --keep --allow-dirty
```

The **default invocation** retains all four-validator quorum tests, scheduler TERM/KILL and
known-key proof checks, the signed local MsgSend/replay check, and the offline
census. After those live phases recover all four validators, a **fifth fresh
observer** receives only validated public genesis and loopback peer identities.
Its database is initially absent. It replays ordinary P2P history from the
fixture's declared initial height (currently 10), not from a copied database or
state-sync checkpoint. That fixture's imported scheduler history is synthetic,
not evidence of earlier daemon execution.

**`--observer-only` is an explicit targeted diagnostic.** It uses the same fresh
four-validator genesis setup, generated local test keys, builds and node runner.
It waits until all four validators have canonical history above initial height
10, verifies initial quorum agreement, then runs only the observer replay,
clean same-home restart, retained-checkpoint and catch-up phase. It prints
**`PASS targeted observer replay/restart diagnostic`**, never the default suite's
PASS. It also prints **NOT RUN** for scheduler timing/TERM/KILL/bank-zero proofs,
75%-power progress/50%-power halt and validator recovery, MsgSend/replay and the
offline census: **full integrated suite not established**. Imported schedules may
execute in this same genesis, but this mode does not assess them. Selecting this
mode is never an automatic fallback from a failing default test.

The observed SDK bank-zero nonmembership limitation remains separate: deleting
a zero balance can produce a valid tree witness whose neighbor is an empty-valued
`0x03` bank index, rejected by ICS23 Go v0.11.0 with `leaf op needs value`. The
observer diagnostic neither fixes this compatibility issue nor changes strict
absence/value verification. An observer-only success must not be reported as
scheduler or full-suite success. Both modes retain before-build, after-build and
final candidate-source digest comparisons and abort on source drift;
`--allow-dirty` remains explicitly **NON-FINAL**.

RPC/P2P listeners are loopback-only; state sync, remote signing, unsafe RPC,
profiling, public metrics, REST and gRPC services are disabled. Node-config
environment overrides are refused. Each start binds the genesis and binary
checksums. Account creation, keyring copying, gentxs and voting remain limited
to the four validators/coordinator; no observer account is created.

The verifier checks five distinct nodes but exactly four height-pinned genesis
validators, canonical block IDs, validator-set hashes and cryptographic quorum.
The observer's identity must be excluded, voting power and last-sign state must
remain zero, and no account keyring may appear. A sampled ABCI applied checkpoint
at **H** is bound to **header H+1's `pre_state_app_hash`**, not header H's root.
After a graceful observer-only stop and same-home restart without reset/init,
it catches up and proves the retained application version H with an SDK
IAVL/multistore **membership proof for `cosmos.bank.v1beta1.Params` at store
`bank`, key `05`**. SDK bank InitGenesis persists this parameter record; this
fixture must explicitly have `default_send_enabled=true`, ensuring nonempty
initial protobuf bytes. Missing or incompatible genesis parameters fail preflight.
The verifier checks the returned exact canonical parameter bytes against the
already quorum-authenticated H+1 root, with fixed chain/store/key/height bindings.
A missing, empty, malformed or noncanonical retained record fails explicitly;
there is no alternate-key or absence fallback. Reports name the anchored record
and include its exact value bytes (base64 in JSON), SHA-256 and proof. This is not an accounting, whole-state
completeness or arbitrary bank-zero proof claim, nor a measurement of the exact
stop/Commit boundary.

Public evidence is under the retained run's `reports/observer-preflight.json`,
`observer-replay.json`, `observer-restart.json`,
`five-up-observer-recovered.json` and the public `observer-expect-*.json`
expectations. Diagnostic mode adds `observer-only-initial-quorum.json`; only the
default suite produces the scheduler, fault, MsgSend and census verification
reports (fixture expectations alone are not results). CI uploads
an explicit public-report allowlist, not homes, private identities, databases or
wildcard raw logs. A kept/failed local run still contains disposable test custody
and logs: do not publish its whole directory.

This is a same-machine loopback mechanics test. It does **not** establish
clean-machine isolation, public peer reachability, authenticated release
provenance, a successful production join or operational authorization.

## Before joining any shared network

Require all of the following from the network operator:

1. exact chain ID and genesis bytes with independently verified SHA-256;
2. exact release commit, binary/image digest, and signature/provenance policy;
3. seed and persistent-peer identities from a trusted channel;
4. the selected role; account/custom-verifier registration or consensus
   admission commands only when that role actually requires them;
5. role-relevant staking, commission, gas, slashing, and validator-tier parameters;
6. upgrade plan, halt behavior, rollback boundary, and incident contacts; and
7. explicit authorization for the network phase being joined.

Zerone has custom account and validator registration. Do not substitute the
standard Cosmos `create-validator` flow or reuse commands whose parameters
have not been checked against the selected release.

### Preparing a release-bound non-signing public join (not yet acceptance)

Select and independently authenticate one complete joining tuple before any
shared-network execution: **chain ID and non-signing role; exact genesis bytes
and digest; executable checksum, platform and pinned source; immutable image
when relevant; independently trusted release/signature anchors; public P2P peer
IDs/addresses; synchronization method; and checkpoint/rehearsal evidence**. An
inventory of unsigned files is preparation, not release authority. No actual
public release bundle location or independently trusted signer has been supplied
for this readiness change: those inputs are **`not_provided`**, not a finding
that a release does not exist.

For the later, separately authorized clean-machine repetition:

1. Verify that tuple on the release-bound workstation. Follow the existing
   authority-chain phase checks; source publication or inventory output is not
   permission to provision, deploy or publish an endpoint.
2. Use an empty, separately owned home and independently generated local Comet
   identities. Install only the validated public genesis and reviewed node
   configuration/peer identities; never transfer a validator database, signing
   identity, account keyring or custody archive.
3. Select ordinary P2P replay with state sync disabled for this acceptance path.
   Peers must actually serve the release's required history, including any
   separately rehearsed upgrade boundaries. A historical chain cannot be
   assumed replayable with one arbitrary current binary. State sync would need
   its own authenticated trust material and provider acceptance; this drill
   proves neither.
4. Reuse the existing non-signing topology/runtime contracts in
   [`deploy/fly-full-node-entrypoint.sh`](../deploy/fly-full-node-entrypoint.sh)
   and [`deploy/Dockerfile.full-node`](../deploy/Dockerfile.full-node), with their
   release-specific sentry/public-query role and identity-drift guards. These
   are Fly-specific contracts, **not a generic VPS installer**; do not weaken
   their genesis-validator or closed-release policies to fit this local drill.
5. After an authorized start, independently verify chain/genesis/binary
   bindings, observer exclusion, zero signing state, catch-up and quorum-bound
   checkpoint H/H+1. Cleanly restart the same home and recheck identity,
   persisted checkpoint and catch-up. Public reachability and isolation remain
   separate acceptance observations; keep private custody and raw logs out of
   published reports.

The [`query gateway`](../deploy/query-gateway/README.md) is GET-only. It is not
a standard Cosmos JSON-RPC POST/broadcast endpoint and is not advertised here
as a state-sync provider. Keep legacy joining stubs paused. No full-node-only
real-network runner, validator-admission shortcut, endpoint/DNS publication or
live action is introduced by this preparation.

## Consensus upgrade requirement

The consolidated source includes consensus-visible knowledge, vesting, and
substrate hardening. A legacy SDK v0.50 / IBC-Go v8 network must cross two
ordered pre-SDK boundaries before the later SDK/IBC transition. H1
`consolidation-safety-v1` advances knowledge v5→v6, claiming_pot v1→v2, and
liquiditypool v3→v5 while explicitly preserving vesting_rewards v1. It keeps
every swap fee in pool reserves for bearer LP shares. H2
`founder-renunciation-v1` requires the completed H1 state and alone advances
vesting_rewards v1→v2, retiring the founder auto-split and the
proposer-controlled transaction-presence mint. The exact H1 binary must not
register H2; each boundary uses its own accepted source and separately
attested executable. The obsolete `liquiditypool-safety-v2` name is not a
handler. Neither H1 nor H2 authorizes native pool creation or oracle
allowlisting.

Before activation:

- all validators must run the exact approved binary;
- the handler and module migration boundary must pass export/import and restart
  rehearsal;
- the activation height and recovery procedure must be explicit; and
- old and new binaries must never be mixed outside the agreed upgrade
  sequence.

Publishing the source commit is not the upgrade.

Before H1, the release packet must bind a same-height snapshot containing the
chain ID, height, app hash, module-version map, liquidity and vesting_rewards
Params, module-account balances and permissions, every native pool and LP bank
supply, and the billing quote-denom allowlist. Positive legacy pools migrate
`EXIT_ONLY`, so holders may withdraw without silently enabling swaps, deposits,
or oracle use; any existing pool still requires a separately reviewed
transition. After H1, verify liquiditypool v5, vesting_rewards v1, unchanged
vesting Params and custody, zero protocol skim, empty admission lists, the
applied H1 plan and marker, and matching validator app hashes.

Before H2, bind the complete H1 poststate, ordered plan and done-height
evidence, strict persisted vesting_rewards V1 Params, and the vesting
module-account state. After H2, verify vesting_rewards v2, all retired founder
and automatic-reward fields zero or empty, both ordered markers and done
heights, the retained H2 plan-identity commitment, unchanged supply/custody,
and matching app hashes. The current integrated SDK v0.53 source registers
neither pre-SDK handler and must not be substituted for either release. See
[LIQUIDITYPOOL-SAFETY-V2.md](LIQUIDITYPOOL-SAFETY-V2.md) for the full
liquidity invariant, lifecycle, PPM, governance, and external-Osmosis
separation gates, and [UPGRADES.md](UPGRADES.md) for the ordered H1/H2/H3
release contract.

## `zerone-2`

The release-kit entry point is
[`deploy/networks/zerone-2/README.md`](../deploy/networks/zerone-2/README.md).
The operator checklist is
[`deploy/networks/zerone-2/GO-NO-GO.md`](../deploy/networks/zerone-2/GO-NO-GO.md),
and canonical signing rules are in
[`deploy/networks/zerone-2/CANONICAL-SIGNING.md`](../deploy/networks/zerone-2/CANONICAL-SIGNING.md).

A successful drill, build, or test does not authorize DARK-START, CUTOVER, or
OPEN-BETA. Each phase requires its own signed decision and initiation evidence.
Until those requirements are satisfied, do not start a validator, broadcast a
transition transaction, publish endpoints, or change DNS.

## Operations baseline

Once a network is explicitly authorized, monitor at least:

```bash
zeroned status | jq '.sync_info'
curl --fail-with-body http://127.0.0.1:26657/status
curl --fail-with-body http://127.0.0.1:26657/net_info
```

Keep the consensus signer isolated, back up recovery material through the
approved offline procedure, alert on missed blocks and app-hash divergence,
and rehearse restore/fencing without copying live validator state into a
second signer.

Further references:

- [`docs/INCIDENT_RESPONSE.md`](INCIDENT_RESPONSE.md)
- [`docs/RESILIENCE_PHILOSOPHY.md`](RESILIENCE_PHILOSOPHY.md)
- [`docs/PARAMETERS.md`](PARAMETERS.md)
- [`docs/API.md`](API.md)
