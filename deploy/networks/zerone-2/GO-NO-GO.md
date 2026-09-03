# Three operator decisions — `zerone-2`

This Markdown file is a working checklist and is never a signed authority.
Canonicalize and sign the immutable JSON payloads in
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md). The release packet is signed
first; the three later decision files reference its exact hash and have
deliberately different scopes.

## Immutable release packet

- [ ] Source commit / signed annotated tag / signer fingerprint: `REPLACE`
- [ ] Out-of-band authorized OpenPGP decision signer fingerprint equals the
      release-tag signer, all four canonical operator JSON authority fields,
      and all four main-key initiation/registration evidence fields:
      `REPLACE`
- [ ] `zerone-2` genesis time and two-machine SHA-256: `REPLACE`
- [ ] `zerone-1` halt binary SHA-256 and full immutable image reference:
      `REPLACE` / `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] `zerone-2` binary SHA-256 and full immutable runtime image reference:
      `REPLACE` / `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] Query-gateway full immutable image reference:
      `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] SBOM, provenance, vulnerability decision, signature, and secret scan for
      all three deployables: `REPLACE_EVIDENCE_HASHES`
- [ ] One-validator `f=0`, no-direct-balance-migration, and protocol-dark launch
      policies accepted

Every Fly config below is identified by exact SHA-256, full image reference,
expected role, app, volume (if any), peer topology, and service exposure. The
deployment wrapper must receive those signed values and match its private
snapshot byte-for-byte.

## Decision A — private DARK-START

### Dark-start inputs and gates

- [ ] New validator, edge, operations, identity, transition-attestation, and
      deployment credentials created; offline recovery drill passed
- [ ] Validator app/volume config SHA-256 and role `validator`: `REPLACE`
- [ ] Service-free edge app/volume config SHA-256 and role `edge`: `REPLACE`
- [ ] Service-free edge query-soak config SHA-256 and role `edge`: `REPLACE`
- [ ] Service-free private gateway config SHA-256 and role `zerone-2-query`:
      `REPLACE`
- [ ] `MONITORING-ALERTS.json` SHA-256 plus its byte-matched
      `MONITORING-RULES.json` and `MONITORING-ALERT-TESTS.json` hashes:
      `REPLACE`
- [ ] All ten required rules are enabled and their stalled-height,
      missed-signing, double-sign-risk, AppHash-divergence, peer-loss, disk,
      restart-count, stale-backup, gateway-wrong-chain, and
      gateway-stale-origin alert tests each prove firing, notification delivery,
      resolution, and `PASS`: `REPLACE_EVIDENCE_HASHES`
- [ ] Full tests, artifact audit, signed-source two-run ceremony comparison,
      restart/export/import, image-context, deploy-gate, SBOM, and secret gates
      pass
- [ ] All private profiles have no public Fly service or public P2P address
- [ ] Genesis supply, two owners, locked self-bond, single validator, and every
      protocol-dark invariant independently audited
- [ ] Abandonment rule accepted: a failed booted candidate is never reset; any
      genesis/key change uses a new unpublished chain ID such as `zerone-2-r1`
- [ ] DARK-START initiation deadline leaves adequate deployment margin; block
      1 is the signed initiation event, after which only private completion or
      abandonment remains authorized
- [ ] Main-key `DARK-START-INITIATION-EVIDENCE` procedure is ready to bind block
      1 and unlock any private continuation after the deadline

### Dark-start decision checklist

- [ ] **DARK-START GO** — authorize only provisioning/starting the exact private
      `zerone-2` production validator, edge, monitor, private query gateway, and
      the operator-onboarding/custom-validator registration transactions in
      `CUTOVER.md` sections 2–3.
- [ ] **DARK-START NO-GO** — do not start the genesis or broadcast either private
      `zerone-2` transaction.

This decision does **not** authorize any `zerone-1` transaction or halt, public
notice/post, public Fly service or IP, endpoint publication, or DNS change.

The actual authority is canonical `DARK-START-DECISION.json`, created from
`DARK-START-DECISION.example.json`. Its detached `.sig` is outside the payload.

## Decision B — irreversible old-chain CUTOVER

### Private-soak evidence

- [ ] Exact production chain observed for at least 1,000 blocks and 60 minutes;
      first-block time / evidence height / evidence app hash: `REPLACE`
- [ ] Validator and edge match at equal heights; every RELEASE-bound monitoring
      rule and alert-test artifact remains byte-identical and `PASS`:
      `REPLACE_HASH`
- [ ] Exact supply/two balances/one SDK validator/protocol-dark profile still
      match genesis: `REPLACE_HASH`
- [ ] Operator onboarding and 111,000,000 uzrn custom-staking registration txs
      succeeded without supply drift: `REPLACE_TX_HASHES`
- [ ] Private gateway `/status` reports `zerone-2`, REST query succeeds, and
      broadcast/POST/oversize/rate-limit negative tests pass: `REPLACE_HASH`
- [ ] Recovery evidence and long-lived environment custody scan pass:
      `REPLACE_HASH`

### Old-chain and publication gates

- [ ] `zerone-1` checkpoint state `F`, empty anchor `A=F+1`, and SDK trigger
      `H=A+1`, with estimated UTC time: `REPLACE`
- [ ] Reviewed exact signer and observer config app/role/image/hash mappings:
      `REPLACE`
- [ ] Release-bound archive renderer, candidate/final template hashes, fixed
      app/volume/region/halt-image roles, immediate strategy, empty peers, and
      service-free constraints equal the CUTOVER delegation: `REPLACE`
- [ ] Signer and independent fresh-key observer are synced/app-hash matched and
      can be armed with the exact F/A/H plan before `F`
- [ ] Both transaction ingress and mempools freeze before `F`, forcing empty
      `A` and staged `H`
- [ ] Actual release-binary halt rehearsal proves BlockStore/status `H`,
      canonical empty `A`, empty linked staged `H`, subjective tip commit H
      `canonical=false`, ABCI `A`, and no H results
- [ ] Rehearsal captures matching v3 inventory and byte-exact signer/observer
      evidence before SIGTERM/fencing; green daemon health is not halt proof
- [ ] Raw genesis plus complete RELEASE-trusted/A/H validator-set evidence is
      byte-identical across signer/observer, and the exact release halt binary
      recomputes block IDs, verifies >2/3 commit signatures/power, proves 1/3
      trusted-set continuity to A, and verifies `block_results(A)` AppHash
- [ ] OPEN verification/deployment runs on the dedicated `linux/amd64` release
      workstation; no macOS rebuild, emulator wrapper, or test double replaces
      the exact RELEASE-bound halt binary
- [ ] Stopped-copy hard rollback and deterministic archive-render/adoption drill
      removes H while preserving A; the fresh-key candidate proves A/A, H
      absent, commit-A tip `canonical=false`, `catching_up=true`, isolated peers,
      and private query-only origin
- [ ] Single authoritative final-checkpoint v3 canonicalization,
      second-machine reproduction, transition-key signature, and private archive
      monitoring rehearsed
- [ ] Checkpoint inventory's height-pinned trusted-source/non-Merkle limitation
      accepted
- [ ] Pre-cutover public notice contains no placeholder, discloses private dark-start,
      `f=0`, migration/dark policies, query-only RPC/REST, no initial public
      gRPC, and the exact rollback boundary
- [ ] CUTOVER initiation deadline leaves sufficient notice/F margin: notice and
      exact successor transaction must complete before it; once committed,
      only exact forward completion remains authorized while preconditions hold
- [ ] Exact successor transaction bytes/hash are pre-signed in CUTOVER, and the
      main-key `CUTOVER-INITIATION-EVIDENCE` procedure independently proves its
      on-time successful commit before halt profile deployment

### Cutover decision checklist

- [ ] **CUTOVER GO** — authorize the exact public pre-announcement, `zerone-1`
      successor transaction and F/A/H halt, signer fencing, evidence capture,
      deterministic private archive adoption, and final-checkpoint preparation
      in `CUTOVER.md` sections 4–7.
- [ ] **CUTOVER NO-GO** — keep all new services private; do not post, mutate or
      halt `zerone-1`, expose endpoints, or change DNS.

The actual authority is canonical `CUTOVER-DECISION.json`, created from
`CUTOVER-DECISION.example.json`. It references the unchanged release packet and
the prior dark-start payload/signature/initiation-evidence hashes and exact
pre-signed successor transaction; its own `.sig` remains outside.
It does **not** authorize the history-link transaction, a public Fly service,
endpoint publication, or DNS.

## Decision C — public OPEN-BETA

### Post-halt evidence and authority gates

- [ ] Inner archive-transition manifest binds exact F/A/H, signer/observer
      evidence, source/fresh identities, expected A/app hashes, and
      pre-transition construction hashes without any future-artifact cycle
- [ ] Deterministic renderer reproduces identical candidate config, final
      archive config, and canonical `ARCHIVE-ADOPTION-AUTHORITY.json` on two
      machines from the verified RELEASE/CUTOVER inputs
- [ ] Adoption attestation result is `MATCH`; its transition-key signature and
      full fingerprint verify; candidate/final deployment used only the signed
      authority extractor
- [ ] Candidate readiness, final runtime marker, and private A/A probe evidence
      are hashed and stable; old signer remains stopped and fenced
- [ ] Exact self-sealed `CUSTOM-STAKING-CENSUS.json` passes at application
      height `A` against the lowercase excluded post-anchor/ABCI AppHash,
      binds the RELEASE commit, reconciles `B = D + U`, has no findings, and
      has independently verified release-binary provenance; it grants no
      migration or validator authority
- [ ] `FINAL-CHECKPOINT.json` contains the full release/dark/cutover/adoption
      authority chain, private archive evidence, no public URL/config authority,
      and a valid transition-key detached signature
- [ ] Exact pre-signed `zerone-2` history-link transaction bytes/hash, sender,
      self-send message, fee/gas, and memo commit only the final-checkpoint hash
      and do not reference OPEN-BETA
- [ ] Private `zerone-2` health, validator/edge app-hash equality, supply,
      one-validator roster, and all protocol-dark latches were freshly
      revalidated: `REPLACE_HASH`
- [ ] Public z2 edge, z2 query gateway, and archive query gateway mappings are
      byte-equal to the immutable release packet; direct origins remain private
- [ ] Exact P2P/RPC/REST/archive coordinates and reviewed canonical DNS-change
      manifest hash are final and consistent with the earlier notice
- [ ] OPEN-BETA initiation deadline leaves adequate margin; successful commit
      of the exact history-link transaction is the initiation event
- [ ] Main-key `OPEN-BETA-INITIATION-EVIDENCE` procedure binds expected tx
      bytes/hash, successful committed hash/height/time, raw query evidence, and
      deadline before any public profile deployment

### Open-beta decision checklist

- [ ] **OPEN-BETA GO** — authorize broadcasting only the exact pre-signed
      history-link transaction. Only after it commits, authorize the exact three
      public profile deployments, coordinates, DNS manifest, evidence release,
      and bounded monitoring in `CUTOVER.md` section 8.
- [ ] **OPEN-BETA NO-GO** — do not broadcast the link transaction; keep public
      Fly services absent and do not publish endpoints or change DNS. The
      private successor and frozen private archive remain intact.

The actual authority is canonical `OPEN-BETA-DECISION.json`, created from
`OPEN-BETA-DECISION.example.json`. It is signed by the main operator key and
references the unchanged RELEASE, DARK-START, CUTOVER, adoption, final
checkpoint, both earlier initiation evidence pairs, readiness, exact
transaction, public configs, coordinates, and DNS
manifest. Its detached `.sig` remains outside the payload.
