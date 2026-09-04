# Four operator decisions — `zerone-2`

This Markdown file is a working checklist and is never a signed authority.
Canonicalize and sign the immutable JSON payloads in
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md). The release packet is signed
first; the four later decision files reference its exact hash and have
deliberately different scopes.

## Immutable release packet

- [ ] Source commit / signed annotated tag / signer fingerprint: `REPLACE`
- [ ] Out-of-band authorized OpenPGP decision signer fingerprint equals the
      release-tag signer, all five canonical operator JSON authority fields,
      and all four main-key initiation/registration evidence fields:
      `REPLACE`
- [ ] `zerone-2` genesis time and two-machine SHA-256: `REPLACE`
- [ ] Independent artifact audits confirm the exact pre-genesis slashing
      policy: 34,272-block window, 95% minimum signing, one-hour downtime jail,
      5% double-sign slash, 0.01% downtime slash, and empty signing/missed-block
      state
- [ ] Decision signers explicitly accept that downtime jail/slash is dormant
      in the one-validator set; monitoring and tested operator recovery are the
      only launch availability controls
- [ ] Comet evidence admission is exactly 719,714 blocks and 21 days, matching
      the SDK staking unbonding period; block/validator history capacity covers
      that full window; runtime evidence confirms `timeout_commit=2521ms`,
      `skip_timeout_commit=false`, no `ZERONED_*` overrides, and
      `--min-retain-blocks 0`
- [ ] `zerone-1` halt binary SHA-256 and full immutable image reference:
      `REPLACE` / `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] `zerone-2` binary SHA-256 and full immutable runtime image reference:
      `REPLACE` / `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] Query-gateway full immutable image reference:
      `REPLACE_REGISTRY/REPLACE_REPOSITORY@sha256:REPLACE`
- [ ] SBOM, provenance, vulnerability decision, signature, and secret scan for
      all three deployables: `REPLACE_EVIDENCE_HASHES`
- [ ] GitHub API evidence confirms `zerone-production-signing` exists before
      dispatch, restricts deployment to `main`, has required reviewers, and its
      environment-level `ZERONE_PRODUCTION_SIGNING_POLICY` is exactly
      `required-reviewers-v1`; the API also confirms its three component-specific
      `ZERONE_PRODUCTION_APPROVED_*_IMAGE` variables equal the exact candidate
      digest refs and are not repository- or organization-scoped fallbacks
- [ ] Protected `zerone-production-signing` dispatch produced three real v0.3
      keyless bundles for the exact OCI manifest digests; the exact
      `ci.yml@refs/heads/main` SAN, GitHub OIDC issuer, Fulcio source commit,
      SCT, per-entry Rekor inclusion proof, and RFC 3161 countersignature
      verify offline; these bundles are approval evidence, not standalone build
      provenance or deploy authority
- [ ] The same protected run signed its same-run deterministic Frontier macOS
      archive; the source-bound manifest, exact register/bundle/helper hashes,
      archive checksum, workflow identity, Rekor proof, and RFC 3161 timestamp
      verify before the onboarding tool is extracted or executed
- [ ] TUF-authenticated production Sigstore trusted-root bytes and the
      reproducibly built Linux component verifier are independently reviewed,
      hash-pinned by the v2 operator-tool manifest, and byte-identical in the
      authority bundle
- [ ] Every component evidence `signed_at` equals an authenticated observer
      time returned by the release-bound offline verifier: Rekor v1
      `integratedTime` or an RFC 3161 TSA time
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
      `MONITORING-RULES.json`, v2 `MONITORING-ALERT-TESTS.json`, and all 40
      fixed-name raw evidence files: `REPLACE`
- [ ] All ten required rules are enabled and their stalled-height,
      missed-signing, double-sign-risk, AppHash-divergence, peer-loss, disk,
      restart-count, stale-backup, gateway-wrong-chain, and
      gateway-stale-origin alert tests each prove firing, notification delivery,
      resolution, and `PASS`; each test's exact stimulus, firing, notification,
      and resolution `{filename, sha256}` reference matches the non-empty
      bundled bytes: `REPLACE_EVIDENCE_REVIEW`
- [ ] Full tests, artifact audit, signed-source two-run ceremony comparison,
      restart and complete stopped-data-directory restore, image-context,
      deploy-gate, SBOM, and secret gates pass. Genesis export/import remains a
      diagnostic only: the authoritative-state inventory documents known
      custom-module round-trip omissions and it is not an authorized recovery
      path.
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

## Decision B — transition PRE-NOTICE

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
      service-free constraints match the planned CUTOVER delegation: `REPLACE`
- [ ] Signer and independent fresh-key observer are synced/app-hash matched and
      can be armed with the exact F/A/H plan before `F`
- [ ] Both transaction ingress and mempools freeze before `F`, forcing empty
      `A` and staged `H`
- [ ] Actual release-binary halt rehearsal proves BlockStore/status `H`,
      canonical empty `A`, empty linked staged `H`, subjective tip commit H
      `canonical=false`, ABCI `A`, and no H results
- [ ] Rehearsal captures matching v3 inventory and byte-exact signer/observer
      evidence before SIGTERM/fencing; green daemon health is not halt proof
- [ ] Rehearsal raw genesis plus complete RELEASE-trusted/A/H validator-set evidence is
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
- [ ] Single authoritative final-checkpoint v4 canonicalization,
      second-machine reproduction, transition-key signature, and private archive
      monitoring rehearsed
- [ ] Checkpoint inventory's height-pinned trusted-source/non-Merkle limitation
      accepted
- [ ] Pre-cutover public notice contains no placeholder, discloses private dark-start,
      `f=0`, migration/dark policies, query-only RPC/REST, no initial public
      gRPC, and the exact rollback boundary; it publishes no successor network
      coordinates
- [ ] Canonical main-key `PRE-NOTICE-DECISION.json` binds the exact RELEASE,
      DARK decision/initiation/registration payload/signature pairs, completed
      soak and halt-rehearsal evidence, proposed F/A/H, notice bytes, one HTTPS
      URL, and publication deadline; only exact notice publication is enabled
- [ ] The complete `notice-prepublish` gate passes before posting; it requires
      no future CUTOVER decision or publication evidence

### Pre-notice decision checklist

- [ ] **PRE-NOTICE GO** — publish only the exact signed notice at the exact URL
      before the publication deadline. Capture the observed response body as
      `PUBLIC-NOTICE-CAPTURE.md` and complete v2 publication evidence.
- [ ] **PRE-NOTICE NO-GO** — publish nothing; retain the private successor and
      leave `zerone-1` untouched.

This decision authorizes no transaction, halt, deployment, endpoint, DNS
change, or live-network claim. Its actual authority is canonical
`PRE-NOTICE-DECISION.json`, created from
`PRE-NOTICE-DECISION.example.json`, with a detached main-key `.sig`.

## Decision C — irreversible old-chain CUTOVER

### Publication and initiation gates

- [ ] CUTOVER binds the exact pre-notice payload/signature pair and unchanged
      F/A/H, notice bytes, URL, soak, and halt-rehearsal evidence
- [ ] V2 publication evidence binds the pre-notice pair, exact notice/URL,
      publication time, and non-empty response-body capture whose bytes equal
      the notice; publication occurred after signature and by its deadline
- [ ] CUTOVER creation/signature follows publication. The captured body and
      recorded URL/time are independently reviewed operator evidence, not
      cryptographic proof of network delivery or time
- [ ] CUTOVER initiation deadline leaves sufficient notice/F margin:
      authorized notice publication must already be complete and the exact successor
      transaction must commit before the CUTOVER deadline; once committed,
      only exact forward completion remains authorized while preconditions hold
- [ ] Exact successor transaction bytes/hash are pre-signed in CUTOVER, and the
      main-key `CUTOVER-INITIATION-EVIDENCE` procedure independently proves its
      on-time successful commit before halt profile deployment

### Cutover decision checklist

- [ ] **CUTOVER GO** — authorize the exact `zerone-1` successor transaction
      and F/A/H halt, signer fencing, evidence capture,
      deterministic private archive adoption, and final-checkpoint preparation
      in `CUTOVER.md` sections 4–7.
- [ ] **CUTOVER NO-GO** — keep all new services private; do not post, mutate or
      halt `zerone-1`, expose endpoints, or change DNS.

The actual authority is canonical `CUTOVER-DECISION.json`, created from
`CUTOVER-DECISION.example.json`. It references the unchanged release packet and
the prior dark-start payload/signature/initiation-evidence hashes and exact
pre-notice payload/signature pair and pre-signed successor transaction; its own
`.sig` remains outside.
It does **not** authorize notice publication, the history-link transaction,
a public Fly service, endpoint publication, or DNS.

## Decision D — public OPEN-BETA

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
      binds the RELEASE commit, reconciles `B = D + U`, and has no findings;
      the RELEASE-bound producer ran only after terminal H through the full
      `cutover-postinit` gate and matched the actual stopped-copy file manifest
      before and after execution; transition custody excluded concurrent
      same-UID writers to both the database and private manifest throughout
- [ ] Canonical census execution evidence binds the exact RELEASE and
      CUTOVER-initiation pairs, runner/binary, snapshot hashes, argv, exit-zero
      PASS, stdout-captured atomic report publication, and full-scan/per-leaf-
      proof claims; its detached transition-key signature verifies, and FINAL
      binds report/evidence/signature bytes
- [ ] Separately, the release-built census binary has independently verified
      SBOM/Sigstore/reproducible-build provenance; the transition-signed
      execution attestation does not establish provenance or mechanically
      prove execution
- [ ] `FINAL-CHECKPOINT.json` contains the full release/dark/cutover/adoption
      authority chain, private archive evidence, no public URL/config authority,
      and a valid transition-key detached signature
- [ ] Exact pre-signed `zerone-2` history-link transaction bytes/hash, sender,
      self-send message, fee/gas, and memo commit only the final-checkpoint hash
      and do not reference OPEN-BETA
- [ ] Private `zerone-2` health, validator/edge app-hash equality, supply,
      one-validator roster, and all protocol-dark latches were freshly
      revalidated: `REPLACE_HASH`
- [ ] Public z2 edge and z2 query gateway mappings are byte-equal to RELEASE;
      the archive query gateway static mapping equals RELEASE and its concrete
      config is the deterministic RELEASE-template render of verified FINAL
      A/E/B; direct origins remain private
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
