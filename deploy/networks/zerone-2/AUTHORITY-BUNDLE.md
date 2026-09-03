# Authority bundle contract

> **Current source limitation — deployment NO-GO.** The verifier checks the
> component signature bundle's declared shape, identity, issuer, and digest,
> but does not yet perform cryptographic Sigstore verification against trusted
> Fulcio/Rekor material. Repository CI also does not request GitHub OIDC or
> produce these component signatures. The fixture bundle is only a structural
> rehearsal; do not treat a passing fixture-shaped check as image provenance.

The production authority bundle is an append-only directory of public release,
signed authority, transaction, configuration, and evidence bytes. It contains
no private keys, mnemonics, signer state, credentials, Fly tokens, or node data.
Every production gate snapshots the files it needs, rejects symlinks, verifies
the exact signed graph, and fails if a required byte or release-bound operator
tool has changed.

`deploy/verify-authority-chain.py` exposes five monotonically stronger stages:

1. `dark-registration-preinit` verifies RELEASE, DARK-START and block-1
   initiation evidence before either exact private bootstrap transaction.
2. `cutover-preinit` adds ordered registration evidence, private soak, halt
   rehearsal, public notice, halt configs, and the CUTOVER transaction/decision.
3. `cutover-postinit` additionally requires the signed evidence that the exact
   CUTOVER TxRaw committed successfully and before its deadline.
4. `open-preinit` adds the complete frozen predecessor evidence, deterministic
   archive adoption, FINAL checkpoint, successor revalidation, public configs,
   DNS manifest, and the exact OPEN history-link transaction.
5. `open-postinit` additionally requires the signed evidence that the exact
   history-link TxRaw committed successfully and before its deadline.

Passing a later stage implies successful revalidation of every earlier stage.
The verifier is read-only and does not contact a chain, Fly, or DNS.

## Authority and release files

The invariant root of every stage contains:

- `RELEASE-PACKET.json` and `.sig`;
- `DARK-START-DECISION.json` and `.sig`;
- `DARK-START-INITIATION-EVIDENCE.json` and `.sig`;
- `ZERONE-2-ONBOARD-SIGNED-TX.json`;
- `ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json`;
- `OPERATOR-TOOL-MANIFEST.json`;
- `MONITORING-ALERTS.json`, `MONITORING-RULES.json`, and
  `MONITORING-ALERT-TESTS.json`;
- `zeroned-zerone-1-release` and `zeroned-zerone-2-release`;
- `genesis.json`, `genesis.sha256`, `network-manifest.json`, and
  `GENESIS-MANIFEST.md`;
- for each of `zerone_1_halt`, `zerone_2_runtime`, and `query_gateway`: exact
  SBOM, provenance, signature evidence, offline signature bundle,
  vulnerability scan, and signed vulnerability decision files named by the
  release packet.

The release-bound operator-tool manifest covers every verifier, transaction
gate, deployment gate, renderer, canonicalizer, ceremony/build recipe, and
artifact auditor used by the launch. A valid release signature is insufficient
if the locally executed tool bytes do not match that manifest.

`RELEASE-PACKET.json.monitoring_alerts_sha256` hashes the exact canonical
`MONITORING-ALERTS.json`. That manifest is source/chain/time bound and in turn
hashes the actual normalized `MONITORING-RULES.json` and the complete
`MONITORING-ALERT-TESTS.json` bytes. The rules and tests must cover every one of
stalled height, missed signing, double-sign risk, equal-height AppHash
divergence, private-peer loss, disk capacity, restart count, stale verified
backup, gateway wrong-chain, and gateway stale-origin. Every rule must be
enabled within the verifier's safety bounds. Every alert test must bind distinct
stimulus, firing, notification-delivery, and resolution evidence and record the
exact `INACTIVE` → `FIRING` → `RESOLVED` sequence with result `PASS`. A list of
check names or self-asserted PASS values is not evidence and is rejected.

Each component has six fixed byte-bearing artifacts: SBOM, provenance,
signature evidence, offline signature bundle, vulnerability scan, and
vulnerability decision. The release packet directly hashes the SBOM,
provenance, signature evidence, and vulnerability decision; those files in
turn bind the exact signature bundle and scan bytes. Provenance must repeat the
signed source commit/tag, immutable image digest, build-recipe hash, and binary
hash where the component contains `zeroned`.

## CUTOVER additions

Before CUTOVER the directory also contains:

- `DARK-REGISTRATION-EVIDENCE.json` and `.sig`;
- `PRIVATE-SOAK-EVIDENCE.json` and `HALT-REHEARSAL-EVIDENCE.json`;
- `PUBLIC-NOTICE.md` and `PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json`;
- `fly.halt-signer.toml` and `fly.observer.toml`;
- `CUTOVER-SIGNED-TX.json`;
- `CUTOVER-DECISION.json` and `.sig`.

After the CUTOVER transaction commits, append
`CUTOVER-INITIATION-EVIDENCE.json` and `.sig`. Do not replace any predecessor
file.

## Frozen `zerone-1` evidence

OPEN verification requires the actual byte-bearing frozen evidence, not only
hash fields copied into FINAL:

- `ZERONE-1-INVENTORY-V3.json`;
- exact self-sealed `CUSTOM-STAKING-CENSUS.json` from a disposable stopped
  database copy at application height `A`;
- `SIGNER-EVIDENCE-MANIFEST.json` and
  `OBSERVER-EVIDENCE-MANIFEST.json`;
- `SIGNER-RPC-{STATUS,GENESIS,TRUSTED-BLOCK,TRUSTED-COMMIT,TRUSTED-VALIDATORS,BLOCK-A,COMMIT-A,VALIDATORS-A,BLOCK-RESULTS-A,BLOCK-H,COMMIT-H,VALIDATORS-H,ABCI-INFO,BLOCK-RESULTS-H-MISSING}.json.raw`;
- the same fourteen exact raw response names with the `OBSERVER-RPC-` prefix;
- `POST-ANCHOR-STATE-EXPORT.json.raw` and
  `POST-ANCHOR-STATE-EXPORT-EVIDENCE.json`;
- `OFFLINE-HALTED-OBSERVER-SNAPSHOT-MANIFEST.json`;
- `PRE-TRANSITION-SANITIZED-SNAPSHOT-MANIFEST.json`;
- `ARCHIVE-ROLLBACK-OUTPUT.log` and `ARCHIVE-ROLLBACK-LOG.json`.

The terminal non-status RPC payloads must be byte-identical between the trusted
signer and independent observer. The trusted/A/H validator pages must be
complete rather than paginated; the release halt binary recomputes block hashes,
verifies more than two-thirds of each validator set cryptographically, and
requires one-third trusted-set continuity from the RELEASE anchor to `A`.
The evidence keeps checkpoint-`F` AppHash `B`
distinct from excluded post-anchor-`A` AppHash `E`: block `A` commits `B`,
staged block `H` and ABCI-at-`A` expose `E`, and `E` is never used for the
successor inventory. The custom-staking census must bind `A/E` (with lowercase
`E`) and the RELEASE source commit, pass its internal seal and claimant checks,
and be hashed by FINAL. It is custody evidence only and confers no migration,
staking, validator, or consensus authority.

The current RELEASE schema does not bind a dedicated census executable/SBOM/
provenance/signature set. Production therefore remains NO-GO until the census
binary is reproducibly built from RELEASE, independently verified, and its
provenance is added to the signed artifact chain. The structural fixture does
not satisfy that requirement.

## Archive, FINAL, and OPEN additions

The complete OPEN bundle also contains:

- `SOURCE-OBSERVER-RUNTIME-MARKER` and
  `zerone-1-archive-transition.json`;
- `PRE-TRANSITION-ALLOWLIST-MANIFEST.json`;
- `fly.archive-candidate.toml` and `fly.archive.toml`;
- `ARCHIVE-ADOPTION-AUTHORITY.json` and `.sig`;
- `ARCHIVE-CANDIDATE-READINESS.json`, `ARCHIVE-FINAL-RUNTIME-MARKER`, and
  `ARCHIVE-PRIVATE-A-A-PROBE-EVIDENCE.json`;
- `FINAL-CHECKPOINT.json` and `.sig`;
- `SUCCESSOR-REVALIDATION-EVIDENCE.json`;
- `DNS-CHANGE-MANIFEST.json`;
- `fly.edge.public.toml`, `fly.zerone-2-gateway.public.toml`, and
  `fly.zerone-1-archive-gateway.public.toml`;
- `OPEN-BETA-SIGNED-TX.json` and `OPEN-BETA-DECISION.json` with `.sig`.

Only after the OPEN history-link transaction commits may
`OPEN-BETA-INITIATION-EVIDENCE.json` and `.sig` be appended.

## Read-only verification

Production `open-preinit`/`open-postinit` verification, and every deployment or
transaction gate that invokes either stage, must run on the dedicated
`linux/amd64` release workstation. The bundle deliberately executes the exact
`zeroned-zerone-1-release` binary whose hash and provenance are signed in
RELEASE; the pinned halt build is a Linux/AMD64 executable and is not portable
to this macOS preparation workspace. macOS shell doubles are test fixtures
only. Never replace the release binary with a native rebuild, emulator wrapper,
or script to make a production check pass.

Run the verifier from the same signed release checkout whose operator-tool
bytes are named by `OPERATOR-TOOL-MANIFEST.json`. For example, the final
pre-public and post-public checks are:

```bash
python3 deploy/verify-authority-chain.py open-preinit \
  /secure/authority-bundle '<main-full-fingerprint>' \
  '<transition-full-fingerprint>' \
  --release /secure/authority-bundle/RELEASE-PACKET.json \
  --release-sig /secure/authority-bundle/RELEASE-PACKET.json.sig \
  --decision /secure/authority-bundle/OPEN-BETA-DECISION.json \
  --decision-sig /secure/authority-bundle/OPEN-BETA-DECISION.json.sig \
  --final /secure/authority-bundle/FINAL-CHECKPOINT.json \
  --final-sig /secure/authority-bundle/FINAL-CHECKPOINT.json.sig \
  --final-template deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json \
  --open-template deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json \
  --adoption-template deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json \
  --config-policy deploy/validate-fly-phase-config.py --tool-root .

# After appending OPEN-BETA-INITIATION-EVIDENCE.json and its signature:
python3 deploy/verify-authority-chain.py open-postinit \
  /secure/authority-bundle '<main-full-fingerprint>' \
  '<transition-full-fingerprint>' \
  --release /secure/authority-bundle/RELEASE-PACKET.json \
  --release-sig /secure/authority-bundle/RELEASE-PACKET.json.sig \
  --decision /secure/authority-bundle/OPEN-BETA-DECISION.json \
  --decision-sig /secure/authority-bundle/OPEN-BETA-DECISION.json.sig \
  --initiation /secure/authority-bundle/OPEN-BETA-INITIATION-EVIDENCE.json \
  --initiation-sig /secure/authority-bundle/OPEN-BETA-INITIATION-EVIDENCE.json.sig \
  --final /secure/authority-bundle/FINAL-CHECKPOINT.json \
  --final-sig /secure/authority-bundle/FINAL-CHECKPOINT.json.sig \
  --final-template deploy/networks/zerone-1/frozen/FINAL-CHECKPOINT.example.json \
  --open-template deploy/networks/zerone-2/OPEN-BETA-DECISION.example.json \
  --adoption-template deploy/networks/zerone-2/ARCHIVE-ADOPTION-AUTHORITY.example.json \
  --config-policy deploy/validate-fly-phase-config.py --tool-root .
```

The earlier stages use the same explicit release/config/tool arguments, the
stage-appropriate DARK or CUTOVER decision pair, and only the initiation pair
required by that stage. Never substitute a later-stage evidence pair into an
earlier stage.

## Keys and connected actions

The out-of-band main fingerprint signs RELEASE, all three operator decisions,
and the four initiation/registration evidence files. The distinct transition
fingerprint signs only the deterministic archive-adoption attestation and
FINAL. Neither bundle possession nor the transition key grants deployment or
transaction authority.

Connected actions use the specialized gates in the order documented in
[`CUTOVER.md`](CUTOVER.md): private successor/bootstrap first, CUTOVER observer
first and signer last, deterministic private archive, then the OPEN
history-link transaction, public profiles, and DNS. Real signatures, real
release artifacts, explicit human approval, and current live evidence remain
mandatory; a coherent rehearsal bundle is never production authority.

Canonical byte rules and the authority graph are in
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md); the artifact overview is in the
[`zerone-2` README](README.md).
