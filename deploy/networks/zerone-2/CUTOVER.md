# `zerone-1` → `zerone-2` cutover runbook

This runbook is intentionally inert. Replace every `REPLACE_*` value, attach
the final manifests, and obtain the explicit operator go/no-go before running
any transaction, Fly mutation, halt, DNS change, or public post.

## 0. Immutable release packet and three scoped decisions

Record these once in canonical signed `RELEASE-PACKET.json` before private
dark-start:

```text
ZERONE_1_GENESIS_SHA=c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e
ZERONE_2_GENESIS_SHA=REPLACE_64_HEX
RELEASE_COMMIT=REPLACE_40_HEX
RELEASE_TAG=REPLACE_ANNOTATED_SIGNED_TAG
ZERONE_1_HALT_BINARY_SHA=REPLACE_64_HEX
ZERONE_1_HALT_IMAGE_REF=REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE_64_HEX
ZERONE_2_BINARY_SHA=REPLACE_64_HEX
ZERONE_2_RUNTIME_IMAGE_REF=REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE_64_HEX
QUERY_GATEWAY_IMAGE_REF=REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE_64_HEX
MONITORING_ALERTS_SHA256=REPLACE_64_HEX
```

Record the exact app, role, image component/reference, and SHA-256 for every
phase-invariant z2/public Fly config. Also bind the z1 halt/observer/archive
template hashes, deterministic archive-renderer hash, fixed private archive
app/volume/region/topology constraints, and the canonical
`MONITORING-ALERTS.json`. That monitoring manifest must hash the exact
`MONITORING-RULES.json` and `MONITORING-ALERT-TESTS.json` bytes. A bare digest
is insufficient: the signed packet must distinguish the live `zerone-1` halt
image, `zerone-2` runtime image, and query-gateway image, while the monitoring
manifest must identify the actual tested rules and evidence.

The packet's accepted policy must state
`message_schedule_admission_live_at_genesis=false`. The matching network and
human manifests must declare native message-schedule admission disabled; source
presence never authorizes schedule creation.

Never edit or reissue that packet merely to choose checkpoint heights; all
three operator decisions reference its unchanged hash and detached signature.
Only the later canonical `CUTOVER-DECISION.json`, after private-soak evidence
exists, fixes:

```text
ZERONE_1_CHECKPOINT_STATE_HEIGHT=REPLACE_F
ZERONE_1_FINAL_COMMITTED_HEIGHT=REPLACE_A
ZERONE_1_HALT_TRIGGER_HEIGHT=REPLACE_H
PUBLIC_NOTICE_SHA256=REPLACE_64_HEX
PRIVATE_SOAK_EVIDENCE_SHA256=REPLACE_64_HEX
ZERONE_1_HALT_SIGNER_CONFIG_SHA256=REPLACE_64_HEX
ZERONE_1_OBSERVER_CONFIG_SHA256=REPLACE_64_HEX
```

That decision must prove `A=F+1` and `H=A+1`. `F` must leave enough public
notice time and be well above the successor-commitment transaction height. `H`
is an SDK trigger and application-unapplied terminal tip, not the final
application block selected by transition policy.

CUTOVER does not bind evidence-dependent archive config bytes and does not
authorize public services. After the halt, the signed deterministic renderer
derives those private configs and the transition key attests only a `MATCH`.
After archive readiness and the signed final checkpoint, the separate main-key
`OPEN-BETA-DECISION.json` binds the exact history-link transaction, public
configs/coordinates, and DNS manifest.

## 1. Preflight — no state changes

- Confirm the checkout is clean and the release tag points at the commit.
- Run `go mod verify`, build, full tests, entrypoint tests, artifact audit,
  ceremony reproducibility, export/import drill, and image secret scan.
- Verify the candidate genesis hash independently on two machines.
- Verify the private validator and public edge use new apps, IPs, volumes, node
  keys, consensus keys, operator keys, identity keys, and deployment tokens.
- Confirm the validator exposes no public service and can dial only its explicit
  sentry/edge peer.
- Restore every new key once in an isolated offline drill; fence the recovery
  signer afterward.
- Sync a fresh-key, non-validator `zerone-1` observer to `catching_up=false` and
  verify its app hash equals the official node at the same applied height.
- Verify the signer and observer deployment inputs carry the same exact F/A/H
  plan, and rehearse arming both before `F` with public transaction ingress,
  relay, the Comet mempool, and the application mempool closed.
- Run the actual release binary through the halt rehearsal. Require empty `A`
  and `H`, status/BlockStore tip `H`, a canonical commit at `A`, the subjective
  tip seen commit at `H` with `canonical=false`, ABCI last applied height `A`,
  no block results for `H`, and the staged `H` header linked to `A`. Record that
  the tip canonical flag is not by itself proof of noncanonicality.
- Prove the runbook does not treat a live daemon or green RPC/health endpoint as
  proof of progress or shutdown. Rehearse bounded stability checks followed by
  manual SIGTERM and signer fencing before copying the stopped database.
- Test every RELEASE-bound rule for stalled height, missed signing,
  double-sign risk without starting a second live signer, equal-height AppHash
  divergence, private-peer loss, disk capacity, restart count, stale verified
  backup, gateway wrong-chain, and gateway stale-origin. Preserve distinct
  stimulus, firing, notification-delivery, and resolution evidence for each;
  every test must observe `INACTIVE` → `FIRING` → `RESOLVED` and end in `PASS`.

Any failed item is a no-go.

## 2. Separate private-dark-start authorization

Do not make the irreversible old-chain decision against an unbooted successor.
First complete the [`GO-NO-GO.md`](GO-NO-GO.md) checklist, canonicalize the
release packet, and sign `DARK-START-DECISION.json` as specified in
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md). That
limited GO authorizes only the exact production `zerone-2` validator,
private edge, monitor/alerts, private query gateway, and the two private
registration transactions below. It does not authorize a public notice,
`zerone-1` transaction or halt, public Fly service, DNS change, or post.
Block 1 is the signed initiation event and must commit before that decision's
initiation deadline. If it does not, the authority lapses. Once it does, only
the exact private registration/soak/evidence path or candidate abandonment
continues; public and old-chain actions remain forbidden.

As soon as private block 1 commits, capture validator and independent-edge raw
evidence, complete canonical `DARK-START-INITIATION-EVIDENCE.json`, sign it with
the main key, and verify its block time is no later than the signed deadline.
Before the deadline, the DARK decision itself can start the profiles needed to
produce block 1. After the deadline, any remaining private profile switch must
also pass that signed evidence pair to the deployment gate.

Deploy every reviewed config through `deploy/fly-deploy-authorized.sh`, using
the canonical RELEASE and DARK-START payload/signature pairs, config key, and
out-of-band main fingerprint. The wrapper derives app, full immutable image
reference, role, and exact config SHA-256 from the signed release/decision
chain. Start the validator, then the service-free edge profile. Keep
RPC/REST loopback-only, P2P private, and all public Fly services absent. Verify:

- chain ID, genesis hash, source tag, runtime binary, and image reference;
- one bonded SDK validator with the manifest consensus key;
- exact 13,555,000,000 uzrn supply and the two expected genesis owners;
- vote extensions disabled and every protocol-dark latch unchanged;
- only internal `09-localhost` IBC support; no external client, transfer, ICA,
  bridge, claiming, knowledge admission/reward, issuance, or native
  message-schedule admission path;
- advancing height, expected cadence, stable private peer, equal validator/edge
  app hashes at equal heights, alerts, backups, and no long-lived custody env.

Any need to change genesis or keys after block 1 requires a new unpublished
candidate chain ID such as `zerone-2-r1`; never reset or rewrite the candidate.

## 3. Register and soak the private successor

Before signing DARK-START, generate and back up the separate identity key inside
the protected validator custody boundary. Using only the release binary and the
account number/sequence derived from the frozen genesis, construct, inspect, and
offline-sign these two transactions in order:

1. one `/zerone.auth.v1.MsgRegisterAccount` for the operator, account type
   `human`, the exact `did:zrn:<identity-public-key>`, fee `200000uzrn`, gas
   `200000`, signer sequence `0`, signed timeout height, and fixed onboarding memo;
2. one `/zerone.staking.v1.MsgRegisterValidator` for the same operator/DID and
   genesis consensus public key, exactly `111000000` uzrn self-delegation,
   commission `500`, the disclosed moniker/details, fee/gas, signer sequence
   `1`, later timeout height, and fixed registration memo.

Put the exact signed JSON envelopes in the authority bundle as
`ZERONE-2-ONBOARD-SIGNED-TX.json` and
`ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json`. DARK-START binds both encoded TxRaw
SHA-256 values, uppercase Comet hashes, decoded fields, timeouts, and broadcast
order. Do not use a direct production `zeroned tx ... --yes` path.

After signed block-1 initiation evidence exists, broadcast only through the
byte gate:

```bash
scripts/zerone-2-bootstrap-tx-broadcast.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  DARK-START-DECISION.json DARK-START-DECISION.json.sig \
  DARK-START-INITIATION-EVIDENCE.json \
  DARK-START-INITIATION-EVIDENCE.json.sig \
  operator-onboarding /secure/zeroned-zerone-2-release \
  http://REPLACE_PRIVATE_ZERONE_2_VALIDATOR_RPC:26657 \
  '<main-full-fingerprint>' /secure/authority-bundle

scripts/zerone-2-bootstrap-tx-broadcast.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  DARK-START-DECISION.json DARK-START-DECISION.json.sig \
  DARK-START-INITIATION-EVIDENCE.json \
  DARK-START-INITIATION-EVIDENCE.json.sig \
  custom-validator-registration /secure/zeroned-zerone-2-release \
  http://REPLACE_PRIVATE_ZERONE_2_VALIDATOR_RPC:26657 \
  '<main-full-fingerprint>' /secure/authority-bundle
```

The second invocation independently proves the exact onboarding TxRaw already
committed successfully. Capture both receipts and post-state queries, complete
canonical `DARK-REGISTRATION-EVIDENCE.json`, sign it with the main key, and
reproduce it on a second machine. Prove the SDK/custom validator records,
operator, consensus key, active status, self-delegation, 111,000,000 uzrn
module backing, balances, and unchanged total supply.

Switch only the private edge to `fly.edge.query-soak.example.toml` and start the
service-free `fly.zerone-2.private.example.toml` query gateway. Run the gateway
smoke contract in `deploy/query-gateway/README.md`: chain-aware status and REST
queries succeed; broadcast, JSON-RPC POST, REST POST, oversized, and rate-limit
tests fail as designed. No public service or P2P address may exist.

Observe the exact production validator, edge, gateway path, alerts, and recovery
for at least 1,000 blocks and 60 minutes. Hash the observations, app/hash
comparisons, registration receipts, reviewed Fly configs, and gateway smoke
output. Prepare the exact public notice, but do not sign CUTOVER until the
notice has actually been published and its byte-bound publication evidence has
been captured in section 4. Until that second immutable decision, `zerone-1`
remains canonical and untouched.

Before signing CUTOVER, construct and offline-sign the exact z1 successor
self-send transaction described in section 5. Record its signed TxRaw byte
SHA-256 and expected transaction hash inside CUTOVER. Do not broadcast it yet.
The transaction must not contain the CUTOVER payload hash, which would be
circular.

## 4. Public pre-announcement, then old-chain CUTOVER GO

Publish the completed pre-cutover public notice before signing CUTOVER. The
notice must include:

- both chain IDs and the relaunch reason;
- `zerone-1` checkpoint state `F`, final committed anchor `A=F+1`, halt trigger
  `H=A+1`, and estimated time;
- `zerone-2` genesis time/hash, private first-block time, observed private-soak
  height, source tag, binary checksum, runtime/gateway image references, and
  validator roster;
- the one-validator `f=0` custody disclosure;
- the no-direct-balance-migration policy, checkpoint-`F` inventory plan, and any
  separate eligibility policy;
- explicit statement that PoT, IBC/ICA, bridge, claiming, and issuance are not
  live at genesis;
- the query-only RPC/REST gateway, explicit initial-beta gRPC unavailability,
  public P2P address, and permanent archive gateway;
- warning to use a distinct home directory and never reuse `zerone-1` keys or
  data.

Capture canonical `PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json` for those exact
notice bytes. Only then complete and sign `CUTOVER-DECISION.json` with the
release, DARK decision/initiation/registration evidence pairs, private-soak and
halt-rehearsal evidence, exact notice/publication hashes and time, three
distinct full image references, exact signer/observer config mappings,
deterministic private archive continuation, exact F/A/H plan, transaction
cutoff/timeout/lead, and accepted launch policies. The CUTOVER creation and
signature times must follow the recorded publication time.

Do not describe a proposed hash, endpoint, or height as final. Do not expose
the private query gateway or public P2P merely because the notice was posted;
those remain forbidden until the later OPEN-BETA decision and exact history-link
transaction commit.

Both the exact notice and exact successor transaction must complete before the
signed CUTOVER initiation deadline. The committed successor transaction is the
initiation event. If the deadline passes first, do not arm/halt `zerone-1` and
publish a signed correction for any notice already posted. Once the transaction
commits on time, expiry does not strand the one-way transition: only the exact
signed F/A/H evidence, signer fencing, deterministically rendered private
archive adoption, and final-checkpoint preparation remain authorized while
their preconditions are satisfiable. Public exposure is still forbidden.

## 5. Commit the successor on `zerone-1`

The exact self-send was constructed, inspected, and pre-signed before CUTOVER
was signed. Its final memo is exact and machine-readable:

```text
successor_chain_id=zerone-2;successor_genesis_sha256=REPLACE_64_HEX;checkpoint_state_height=REPLACE_F;final_committed_height=REPLACE_A;halt_trigger_height=REPLACE_H
```

Its signed bytes, expected transaction hash, sender, self-recipient, 1 uzrn
amount, 200000 uzrn fee, gas limit, and memo must equal CUTOVER. After the exact
notice is published, broadcast only those pre-signed bytes:

```bash
deploy/mainnet/fly-cutover-authorized.sh observer \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  /secure/fly.halt-signer.toml /secure/fly.observer.toml \
  '<main-full-fingerprint>' /secure/authority-bundle \
  http://REPLACE_PRIVATE_ZERONE_1_SIGNER_RPC:26657 \
  http://REPLACE_PRIVATE_ZERONE_1_OBSERVER_RPC:26657

scripts/zerone-phase-tx-broadcast.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  /secure/authority-bundle/CUTOVER-SIGNED-TX.json cutover \
  /secure/zeroned-zerone-1-release \
  http://REPLACE_PRIVATE_ZERONE_1_RPC:26657 \
  '<main-full-fingerprint>' /secure/authority-bundle
```

The observer-first gate prechecks both halt configs, deploys only the pinned
observer image, and proves the signed signer/observer node identities, trusted
block, common live block/AppHash, and minimum lead before `F`. The transaction
gate snapshots every direct input, verifies the complete authority bundle,
release binary, exact trusted RPC node/block, cutoff, timeout and live lead,
encodes the snapshotted signed tx, requires the exact signed TxRaw
SHA-256/Comet hash, and submits those exact raw bytes. Require its CheckTx code
zero/hash, then require deliver code zero at
commit. Record committed height/time and raw query evidence, then query it
independently from the observer. Never use `test` as the production keyring
backend.

This independent transaction evidence comes from the already-synced,
service-free read-only observer established in preflight. It remains running and
query-capable before its later CUTOVER halt-profile redeploy, so the evidence
gate does not depend on the post-init deployment it unlocks.

Complete canonical `CUTOVER-INITIATION-EVIDENCE.json` with the CUTOVER
payload/signature hashes, exact notice/publication evidence, signed transaction
bytes/expected hash, matching committed hash, height/time, and both query
evidence hashes. Require commit time no later than CUTOVER's canonical UTC
deadline, sign the evidence with the main key, and reproduce/verify its bytes
and signature on a second machine. Without this `MATCH` evidence, do not deploy
the halt profiles or proceed toward `F`.

## 6. Arm and observe the old-chain halt

Before `F`, close transaction ingress and relay and reassert both the Comet and
application mempool freeze. The observer was deployed and proven before the
successor transaction. After the exact signed CUTOVER initiation evidence
exists, deploy the already-tested `zerone-1` image to the official signer last,
with no generic extra start flags:

```bash
deploy/mainnet/fly-cutover-authorized.sh signer \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
  /secure/fly.halt-signer.toml /secure/fly.observer.toml \
  '<main-full-fingerprint>' /secure/authority-bundle \
  http://REPLACE_PRIVATE_ZERONE_1_SIGNER_RPC:26657 \
  http://REPLACE_PRIVATE_ZERONE_1_OBSERVER_RPC:26657
```

The specialized gate refuses missing, mismatched, late, near-`F`, wrong-node,
or divergent evidence, deploys the signer last, and immediately rechecks both
nodes. Generic `fly-deploy-authorized.sh` deliberately rejects CUTOVER.

```text
ZERONE_CHECKPOINT_STATE_HEIGHT=REPLACE_F
ZERONE_FINAL_COMMITTED_HEIGHT=REPLACE_A
ZERONE_HALT_TRIGGER_HEIGHT=REPLACE_H
```

The entrypoint rejects partial, malformed, inconsistent, legacy single-height,
or unsafe-skip configurations. Confirm both nodes report `zerone-1`, the
expected distinct node IDs, equal applied state, and advancing height before
`F`. Do not reopen ingress after this check.

The expected halt is not a daemon exit. The SDK raises the configured halt from
`FinalizeBlock(H)` after Comet has staged block `H`; RPC and ordinary health
checks may remain alive and green while consensus can no longer advance. On the
signer and observer, require all of the following exact evidence:

1. `/status` is stable at BlockStore tip `H` and reports `catching_up=false`.
2. `/block?height=A` exists, has zero transactions, and its signed header
   `AppHash` is the checkpoint hash for state `F`.
3. `/commit?height=A` reports `canonical=true`, with matching block ID and
   commit on the signer and observer.
4. `/block?height=H` exists and has zero transactions; its block ID and header
   match on the signer and observer, and its header links to block `A`.
5. `/commit?height=H` reports the subjective seen commit and
   `canonical=false` because no `H+1` exists. Do not interpret this tip flag as
   independent proof that `H` is noncanonical.
6. `/abci_info` reports last applied height `A`, and
   `/block_results?height=H` is unavailable.
7. Repeated bounded checks show no change in the A/H block IDs, commits, ABCI
   height, or halt evidence.

If any invariant differs, do not declare a checkpoint. Preserve logs and stop
for investigation. In particular, never infer success from process liveness,
Fly health, or `/status` height alone.

While both ingress-fenced `H/A` RPCs are still alive and stable, run
`relaunch-snapshot` against each trusted source and compare the deterministic
v3 output. Before deleting `H` from any copy, save byte-exact responses for
`/status` and `/genesis`; RELEASE's trusted `/block`, `/commit`, and complete
`/validators` page; `/block`, `/commit`, and complete `/validators` pages at
both `A` and `H`; `/block_results A`; `/abci_info`; and the missing
`/block_results H` response. Require every non-status byte to match across both
sources and reject a paginated validator set. Hash those files into the two
immutable terminal manifests. The release halt binary must recompute the three
block hashes, cryptographically verify more than two-thirds commit power, and
prove at least one-third trusted-set continuity from RELEASE's signed anchor
to `A`. This raw bundle preserves H's signed header and seen-commit signatures;
the compact v3 inventory alone is not a substitute.

```bash
go run ./tools/relaunch-snapshot \
  --rpc REPLACE_HALTED_OBSERVER_RPC \
  --rest REPLACE_HALTED_OBSERVER_REST \
  --expected-chain-id zerone-1 \
  --checkpoint-state-height REPLACE_F \
  --final-committed-height REPLACE_A \
  --halt-trigger-height REPLACE_H \
  --declared-genesis-sha256 c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e \
  --out REPLACE_SECURE_DIR/zerone-1-final-inventory-v3.json
```

After capture, manually SIGTERM both processes. Fence the official signer and
automatic restart path, prove the PID is gone, and only then quarantine its
volume and copy the stopped observer database. Do not reset or restart the
signer. Explicit transition policy ends official application/history at `A`,
selects inventory state `F`, and retains staged application-unapplied `H` only
as terminal evidence.

## 7. Capture the final historical package

Verify and sign the v3 inventories and raw-evidence hash manifest captured in
section 6. REST response headers pin the trusted view to `F` but are not Merkle
proofs. Block `A`'s header `AppHash` is the checkpoint hash for `F`; explicit
policy excludes post-`A` application state from successor eligibility.

Make a stopped observer copy and preserve the original `H/A` database offline
as evidence. On the serving copy only, run `zeroned rollback --hard`. Comet's
pending-block special case must report height `A` and the captured post-anchor
app hash: it deletes staged `H` without rolling application/state below `A`.
Abort if the reported height or hash differs.

Build the base serving copy from the exact public-config/database allowlist in
`deploy/mainnet/BUILD.md`; never clone the observer home. Before adding any
transition/future authority file, hash the pre-transition sanitized snapshot,
rollback log, and a manifest that hashes only those base allowlisted files. Its
explicit exclusion set is the inner transition manifest, rendered Fly configs,
adoption authority, readiness, final checkpoint, and OPEN-BETA bytes. This
keeps the authority graph acyclic.

Inject fresh non-validator node/consensus keys and a fresh height-zero validator
state into the private runtime home, retain halt trigger `H`, and keep all
P2P/transaction ingress fenced. No keyring, WAL, mnemonic, identity,
environment, deployment credential, or old/source node/consensus identity may
cross the boundary. The separately published database artifact excludes all
keys and validator state; the private runtime home necessarily contains only
the newly generated non-validator key/state files required to boot.

Create the exact `zerone-1-archive-transition-v1` inner manifest described in
`deploy/mainnet/BUILD.md`. It binds:

- F/A/H, genesis, expected A block hash, and post-A application hash;
- signed CUTOVER-initiation evidence payload/signature hashes plus exact
  notice/transaction hash, committed height/time, and publication evidence;
- signer/observer evidence, source observer and fresh candidate identities;
- pre-transition snapshot/rollback/allowlist hashes and the fixed exclusion
  set; and
- a fresh transition nonce.

The inner manifest contains no rendered config, adoption, readiness, FINAL, or
OPEN hash. Put it at the allowlisted runtime path and reproduce its exact hash.

Run the deterministic renderer on two isolated machines:

```bash
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<main-full-fingerprint>' \
  deploy/mainnet/render-archive-configs.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
  zerone-1-archive-transition.json /secure/archive-render-a \
  /secure/authority-bundle
```

Require byte equality for `fly.archive-candidate.toml`, `fly.archive.toml`, and
canonical `ARCHIVE-ADOPTION-AUTHORITY.json`. The renderer verifies both main
signatures, on-time initiation evidence, exact templates/renderer/static
constraints, and every allowed substitution. Sign only its emitted adoption
JSON with the transition-attestation key; verify the full fingerprint and
detached signature independently. Never hand-author a `MATCH` payload.

Deploy the private candidate using the reproduction gate (use the same command
with config key `zerone_1_archive` only after candidate readiness):

```bash
deploy/mainnet/fly-deploy-archive-authorized.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
  zerone-1-archive-transition.json \
  ARCHIVE-ADOPTION-AUTHORITY.json ARCHIVE-ADOPTION-AUTHORITY.json.sig \
  zerone_1_archive_candidate \
  '<main-full-fingerprint>' '<transition-full-fingerprint>' \
  /secure/authority-bundle
```

The gate reruns the renderer, byte-compares the signed adoption authority, and
deploys only its reproduced private config. The candidate must prove:

Do not run `relaunch-snapshot` v3 against this sanitized copy: it correctly
requires the pre-fence `H/A` view. The separate serving-archive contract is:

1. chain ID is exactly `zerone-1`;
2. BlockStore/status height is `A`;
3. ABCI last applied height and hash equal captured `A`;
4. `/block?height=H` is absent;
5. `/commit?height=A` now reports `canonical=false` because `A` is the current
   BlockStore tip. This does not undo the canonical `A` proof captured before
   fencing, when `H` still carried `A`'s commit;
6. fresh keys do not match the old validator, peer lists are empty, and all
   broadcast paths are off.

With a fresh non-validator key and no peers, CometBFT v0.38 reports
`catching_up=true` even while serving this frozen `A/A` view; there is no
block-sync enable flag in this version. Do not use generic sync health as
archive readiness. Capture the candidate readiness file/hash only after all six
invariants and byte-stable query probes pass. Stop the candidate, then rerun the
same reproduction-gated command with config key `zerone_1_archive`. The final
private archive role must consume the exact readiness marker on the same
`zerone-1-archive` app/`zerone_archive_data` volume. Capture the final runtime
marker and a fresh private A/A probe-evidence hash. A missing or changed
readiness marker fails closed.

Before completing FINAL, build `custom-staking-census` reproducibly from the
exact signed RELEASE source. After every source daemon is stopped, run it only
against a separately captured disposable observer database copy, with chain ID
`zerone-1`, expected height `A`, the lowercase excluded post-anchor/ABCI
AppHash, and the RELEASE commit. Preserve its exact self-sealed
`CUSTOM-STAKING-CENSUS.json` bytes in the authority bundle for FINAL. Separately,
independently verify its release-built binary provenance before OPEN. A report
at checkpoint `F`, or one used as a migration plan, is invalid.

Independently hash the inventory, custom-staking census, post-anchor state
export, and sanitized database archive. The post-anchor application state at
`A` is archival/custody evidence only and is explicitly excluded from
successor inventory and eligibility. Only after candidate readiness, final
runtime-marker, and private probe evidence exist, complete
`FINAL-CHECKPOINT.json` from
[`zerone-1/frozen/FINAL-CHECKPOINT.example.json`](../zerone-1/frozen/FINAL-CHECKPOINT.example.json),
the single authoritative v3 template. It must bind RELEASE, DARK-START and its
initiation evidence, CUTOVER and its initiation evidence, ARCHIVE-ADOPTION,
inner transition, archive config, readiness, runtime-marker, and probe hashes.
It contains no public URL/config, history-link transaction, or OPEN reference.
Reject every remaining `REPLACE_` value, canonicalize it using the exact byte procedure in
[`zerone-1/frozen/README.md`](../zerone-1/frozen/README.md#exact-signed-bytes),
sign those bytes with the offline transition-attestation key, and verify both
the independently reproduced file hash and detached signature on a second
machine.

Keep the final A/A origin private and service-free. The later public gateway
must reject `broadcast_tx_*` and non-query REST methods. The signed raw bundle,
not the serving database, preserves `H`. `FINAL-CHECKPOINT.json` is the signed
manifest of the v3 inventory, raw response hashes, custom-staking custody
census, excluded state, rollback, adoption, and private archive evidence; it is
not the v3 inventory itself and does not authorize exposure or migration.

## 8. Link histories and open the beta

Freshly revalidate private z2 health/app hashes, supply/validator roster, and
all protocol-dark latches, including
`message_schedule.params.accept_new_schedules=false`. Finalize the exact public
P2P/RPC/REST/archive coordinates and DNS-change manifest without deploying them.

Construct and offline-sign the exact z2 self-send history-link transaction. Its
memo commits only:

```text
zerone_1_final_checkpoint_sha256=REPLACE_WITH_FINAL_CHECKPOINT_64_HEX
```

Record the signed TxRaw byte SHA-256 and expected transaction hash. The
transaction must not contain the future OPEN-BETA payload hash. Complete
canonical `OPEN-BETA-DECISION.json` with every prior payload/signature hash,
archive readiness, final checkpoint/signature, successor revalidation,
pre-signed transaction, exact three public config mappings, coordinates, DNS
manifest, and canonical UTC initiation deadline. Reproduce the bytes, sign with
the main operator key, and verify the full fingerprint on a second machine.

Before the deadline, broadcast only the exact pre-signed history-link bytes
through the same byte-verifying gate:

```bash
scripts/zerone-phase-tx-broadcast.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  OPEN-BETA-DECISION.json OPEN-BETA-DECISION.json.sig \
  /secure/authority-bundle/OPEN-BETA-SIGNED-TX.json open-beta \
  /secure/zeroned-zerone-2-release \
  http://REPLACE_PRIVATE_ZERONE_2_RPC:26657 \
  '<main-full-fingerprint>' /secure/authority-bundle \
  '<transition-full-fingerprint>'
```

The successful commit of its exact expected transaction hash is the OPEN-BETA
initiation event. If it does not commit on time, the decision lapses: keep all
public services absent, do not change DNS, and issue a new transaction/decision
if still appropriate.

After a successful on-time commit, capture raw transaction and independent-edge
evidence. Complete canonical `OPEN-BETA-INITIATION-EVIDENCE.json` with the OPEN
payload/signature hashes, exact bytes/expected and committed hash, code zero,
height/time, deadline, and evidence hashes. Sign and independently verify it
with the main key. No public deployment is allowed without this `MATCH` file.

Deploy only the three profiles present in OPEN-BETA—the public z2 edge, public
z2 query gateway, and public z1 archive query gateway—through the signed gate:

```bash
deploy/fly-deploy-authorized.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  OPEN-BETA-DECISION.json OPEN-BETA-DECISION.json.sig \
  /secure/fly.edge.public.toml zerone_2_edge_public \
  '<main-full-fingerprint>' \
  OPEN-BETA-INITIATION-EVIDENCE.json OPEN-BETA-INITIATION-EVIDENCE.json.sig \
  FINAL-CHECKPOINT.json FINAL-CHECKPOINT.json.sig \
  '<transition-full-fingerprint>' /secure/authority-bundle
```

Repeat for keys `zerone_2_gateway_public` and
`zerone_1_archive_gateway` with their exact configs. Never redeploy the private
archive origin under OPEN; section 7 already adopted it under the deterministic
archive authority.

Verify external gateway `/status` chain IDs and archive A/A invariants, then
publish query-only RPC/REST, public P2P, and archive endpoints. Public gRPC and
hosted transaction submission remain unavailable. Apply the exact signed DNS
manifest, publish all authority/evidence/checkpoint/registration hashes and the
postmortem, and monitor the first 1,000 public blocks continuously.

If the edge fails, replace the edge while the private validator continues. If
the validator fails, fence it and recover manually; never start two copies of
the consensus key.

## Rollback boundary

- After private `zerone-2` block 1 but before the public notice and old-chain
  successor transaction, `zerone-1` remains canonical. The private candidate
  may be abandoned, but never reset; any genesis/key change uses a new chain ID
  such as `zerone-2-r1`.
- After the public successor transaction but before the old freeze, abort the
  halt only with an immediate signed correction. Never silently repoint the
  already-published `zerone-2` identity.
- After `zerone-1` freezes, forward-only. Recover the already-soaked successor
  with compatible pinned images or publish a new explicit chain ID. Never
  rewrite either history and never restart the fenced old signer.
