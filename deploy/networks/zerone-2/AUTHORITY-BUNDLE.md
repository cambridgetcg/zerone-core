# Authority bundle contract

> **Current artifact state — deployment NO-GO.** Source now performs offline
> cryptographic Sigstore verification and CI contains a protected manual OIDC
> signing job. The three real image bundles, reviewed frozen trusted root,
> hash-pinned verifier binary, and real signed authority/evidence graph do not
> exist merely because those paths compile. A fixture bundle remains a rehearsal
> and is never production image provenance.

The production authority bundle is an append-only directory of public release,
signed authority, transaction, configuration, and evidence bytes. It contains
no private keys, mnemonics, signer state, credentials, Fly tokens, or node data.
Every production gate snapshots the files it needs, rejects symlinks, verifies
the exact signed graph, and fails if a required byte or release-bound operator
tool has changed.

`deploy/verify-authority-chain.py` exposes seven stages:

1. `dark-preinit` verifies RELEASE, DARK-START, both exact private bootstrap
   transactions, the complete monitoring package, release-bound operator
   tools, and every component's offline Sigstore signature against the frozen
   trusted root before private block 1 may start.
2. `dark-registration-preinit` adds block-1
   initiation evidence before either exact private bootstrap transaction.
3. `notice-prepublish` adds ordered registration evidence, completed private
   soak and halt rehearsal, exact notice bytes, and the signed PRE-NOTICE
   decision. It checks the live publication deadline and needs no later
   publication evidence or CUTOVER artifact.
4. `cutover-preinit` adds byte-matched notice capture and v2 publication evidence,
   halt configs, and the CUTOVER transaction/decision. It verifies that the
   notice was published within PRE-NOTICE authority and before CUTOVER creation.
5. `cutover-postinit` additionally requires the signed evidence that the exact
   CUTOVER TxRaw committed successfully and before its deadline.
6. `open-preinit` adds the complete frozen predecessor evidence, deterministic
   archive adoption, FINAL checkpoint, successor revalidation, public configs,
   DNS manifest, and the exact OPEN history-link transaction.
7. `open-postinit` additionally requires the signed evidence that the exact
   history-link TxRaw committed successfully and before its deadline.

Later stages revalidate the earlier authority and evidence. Completed events
use their signed historical deadlines; they do not require an earlier
publication or broadcast window to remain open. A timely published notice
remains verifiable after its deadline, without granting new publication
authority. The verifier is read-only and does not contact a chain, Fly, DNS,
or the notice URL.

## Authority and release files

The invariant root of every stage contains:

- `RELEASE-PACKET.json` and `.sig`;
- `DARK-START-DECISION.json` and `.sig`;
- `ZERONE-2-ONBOARD-SIGNED-TX.json`;
- `ZERONE-2-CUSTOM-VALIDATOR-SIGNED-TX.json`;
- `OPERATOR-TOOL-MANIFEST.json`;
- `custom-staking-census-linux-amd64`, whose filename and SHA-256 are fixed by
  RELEASE v2;
- `MONITORING-ALERTS.json`, `MONITORING-RULES.json`, and
  `MONITORING-ALERT-TESTS.json`;
- the 40 non-empty raw alert-test evidence files named by the fixed monitoring
  evidence convention below;
- `zerone-component-signature-verifier` and
  `SIGSTORE-TRUSTED-ROOT.json`;
- `zeroned-zerone-1-release` and `zeroned-zerone-2-release`;
- `genesis.json`, `genesis.sha256`, `network-manifest.json`, and
  `GENESIS-MANIFEST.md`;
- for each of `zerone_1_halt`, `zerone_2_runtime`, and `query_gateway`: exact
  SBOM, provenance, signature evidence, offline signature bundle,
  vulnerability scan, and signed vulnerability decision files named by the
  release packet.

Every stage after `dark-preinit` additionally requires
`DARK-START-INITIATION-EVIDENCE.json` and `.sig`. Thus the first deployment
gate can authenticate the full release and supply chain before block 1 exists,
while every later stage preserves and revalidates the signed block-1 evidence.

The release-bound v2 operator-tool manifest covers every verifier, transaction
gate, deployment gate, renderer, the archive-gateway render template, the
custom-staking census evidence runner, canonicalizer, ceremony/build recipe,
and artifact auditor used by the launch. It also hashes the exact executable
component-signature verifier and frozen Sigstore trusted root in the authority
bundle and fixes the accepted issuer, workflow SAN, bundle media type, SCT,
Fulcio source-commit claim, transparency-log, and observer-time thresholds. A
valid release signature is insufficient if any locally executed tool,
workflow, template, policy, or trust-root byte differs.

`RELEASE-PACKET.json.monitoring_alerts_sha256` hashes the exact canonical
`MONITORING-ALERTS.json`. That manifest is source/chain/time bound and in turn
hashes the actual normalized `MONITORING-RULES.json` and the complete
`MONITORING-ALERT-TESTS.json` bytes. The rules and tests must cover every one of
stalled height, missed signing, double-sign risk, equal-height AppHash
divergence, private-peer loss, disk capacity, restart count, stale verified
backup, gateway wrong-chain, and gateway stale-origin. Every rule must be
enabled within the verifier's safety bounds.

The alert-test document uses
`zerone-production-monitoring-alert-tests-v2`. Each test has one exact
`evidence` object with exactly `stimulus`, `firing`, `notification`, and
`resolution` members. Each member is an exact `{filename, sha256}` reference.
The verifier derives, requires, and hashes the filename itself; a document
cannot choose another path. The fixed form is:

```text
MONITORING-ALERT-<CHECK>-<KIND>-EVIDENCE.json.raw
```

`<CHECK>` is exactly one of `STALLED-HEIGHT`, `MISSED-SIGNING`,
`DOUBLE-SIGN-RISK`, `APP-HASH-DIVERGENCE`, `PEER-LOSS`, `DISK-CAPACITY`,
`RESTART-COUNT`, `STALE-BACKUP`, `GATEWAY-WRONG-CHAIN`, or
`GATEWAY-STALE-ORIGIN`. `<KIND>` is exactly `STIMULUS`, `FIRING`,
`NOTIFICATION`, or `RESOLUTION`. Thus every stage requires exactly 40 raw
proof files. Each must be non-empty, no two references may reuse a digest, and
each file is capped at 16 MiB before authentication and counted toward the
bundle's 1 GiB aggregate cap.

Every alert test must record the exact `INACTIVE` → `FIRING` → `RESOLVED`
sequence with result `PASS` and bind the actual bytes for stimulus, firing,
notification delivery, and resolution. A list of check names, self-asserted
PASS values, unbound digests, missing bytes, or a substituted proof file is
rejected. The raw evidence is intentionally byte-preserved rather than parsed
or normalized by the authority verifier; the signing operator remains
responsible for reviewing what those bytes prove.

Each component has six fixed byte-bearing artifacts: SBOM, provenance,
signature evidence, offline signature bundle, vulnerability scan, and
vulnerability decision. The release packet directly hashes the SBOM,
provenance, signature evidence, and vulnerability decision; those files in
turn bind the exact signature bundle and scan bytes. Provenance must repeat the
signed source commit/tag, immutable image digest, build-recipe hash, and binary
hash where the component contains `zeroned`.

Component signature acceptance is cryptographic, not structural. After the
OpenPGP RELEASE signature authenticates the verifier and trusted-root hashes,
the authority verifier invokes that exact local executable once per component.
It requires a v0.3 Sigstore `messageSignature`, exact GitHub Actions issuer/SAN
and Fulcio source-repository commit, one valid certificate SCT, an inclusion
proof on every supplied Rekor entry, at least one trusted Rekor entry, and one
signed observer timestamp. The evidence file's `signed_at` must equal a
verified Rekor v1 integrated time or RFC 3161 TSA countersignature time. No
network, ambient TUF state, or workstation-current-time fallback participates
in that decision.

The GitHub Actions `CI` workflow exposes `sign_release_components` only through
manual dispatch on current `main`. Its `zerone-production-signing` environment
must exist before the first dispatch, have required reviewers, and define the
environment-level variable `ZERONE_PRODUCTION_SIGNING_POLICY` with the exact
value `required-reviewers-v1`. The same environment must define three exact
full digest references under
`ZERONE_PRODUCTION_APPROVED_ZERONE_1_HALT_IMAGE`,
`ZERONE_PRODUCTION_APPROVED_ZERONE_2_RUNTIME_IMAGE`, and
`ZERONE_PRODUCTION_APPROVED_QUERY_GATEWAY_IMAGE`. Those values are the
authoritative component-to-repository mapping for the signing run; each
dispatch input must match its approved value byte-for-byte.

GitHub otherwise creates a referenced missing environment without protection;
the workflow checks all four values before checkout, Cosign installation, or
OIDC use and fails closed when any are absent or malformed. Confirm through the
GitHub API that the environment, main-only policy, reviewers, and all four
variables are environment-scoped as an operator preflight; the in-job checks
cannot prove variable scope. The job waits for every CI job, checks three
pairwise-distinct digest-only image references, rejects multi-platform indexes,
checks the source-revision label against the workflow commit, captures a
default Sigstore signing configuration containing Rekor and TSA services,
hash-checks and signs each exact OCI manifest, verifies the exact workflow
identity and commit plus its RFC 3161 timestamp, and uploads the evidence for
14 days. Its separate non-cancelling concurrency lane prevents an ordinary
push from interrupting a signing run after log publication. Dispatching the
job creates public signature/transparency evidence for environment-approved
bytes; it is not builder provenance, does not create chain authority, and does
not deploy anything. No component-image bundle is accepted for launch outside
the later OpenPGP-signed RELEASE graph, which independently binds component
provenance, recipes, source, images, and binaries.

On the dedicated Linux release workstation, build the verifier twice from the
signed release checkout and compare the outputs before freezing it:

```bash
cd tools/sigstore-substrate-compiler
export GOTOOLCHAIN=go1.25.14 CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go mod verify
go test ./...
go build -trimpath -buildvcs=false -ldflags='-buildid=' \
  -o /secure/build/component-verifier-a \
  ./cmd/zerone-component-signature-verifier
go build -trimpath -buildvcs=false -ldflags='-buildid=' \
  -o /secure/build/component-verifier-b \
  ./cmd/zerone-component-signature-verifier
cmp /secure/build/component-verifier-a /secure/build/component-verifier-b
sha256sum /secure/build/component-verifier-a
```

Retrieve `trusted_root.json` through Sigstore's signed production TUF
repository, review its Fulcio, Rekor, CT-log, and timestamp-authority entries,
then copy those exact bytes as `SIGSTORE-TRUSTED-ROOT.json`. Do not use the
mutable GitHub branch copy
as the trust bootstrap. Copy the compared verifier as
`zerone-component-signature-verifier`, calculate both hashes, and place them in
`OPERATOR-TOOL-MANIFEST.json` before canonicalizing and signing RELEASE.

## PRE-NOTICE additions

Before exact notice publication the directory also contains:

- `DARK-REGISTRATION-EVIDENCE.json` and `.sig`;
- `PRIVATE-SOAK-EVIDENCE.json` and `HALT-REHEARSAL-EVIDENCE.json`;
- `PUBLIC-NOTICE.md`;
- `PRE-NOTICE-DECISION.json` and `.sig`.

The canonical main-key PRE-NOTICE decision binds the exact release/DARK/
initiation/registration payload and signature pairs, completed soak/rehearsal
hashes, notice hash, exact HTTPS URL, proposed F/A/H, creation time, and
publication deadline. Only publication of those exact notice bytes is enabled;
all chain and infrastructure effects remain excluded. No later CUTOVER or
publication artifact is a prerequisite of `notice-prepublish`.

## CUTOVER additions

After authorized notice publication, append:

- `PUBLIC-NOTICE-CAPTURE.md`, the non-empty observed response body, byte-equal
  to `PUBLIC-NOTICE.md`;
- canonical `PUBLIC-NOTICE-PUBLICATION-EVIDENCE.json` using
  `zerone-public-notice-publication-evidence-v2`;
- `fly.halt-signer.toml` and `fly.observer.toml`;
- `CUTOVER-SIGNED-TX.json`;
- `CUTOVER-DECISION.json` and `.sig`.

After the CUTOVER transaction commits, append
`CUTOVER-INITIATION-EVIDENCE.json` and `.sig`. Do not replace any predecessor
file.

CUTOVER binds the pre-notice payload/signature pair. V2 publication evidence
binds that same pair, notice hash, URL, publication time, and capture hash.
Publication must occur after the pre-notice signature and by its deadline, and
before CUTOVER creation/signature. The CUTOVER initiation evidence must use the
exact publication JSON hash, which remains the same through archive and FINAL
bindings. The capture is preserved as observed; no rendering or normalization
may be substituted. These bytes and the downstream signatures are reviewed
operator attestations, not cryptographic proof of internet delivery or time.

## Frozen `zerone-1` evidence

OPEN verification requires the actual byte-bearing frozen evidence, not only
hash fields copied into FINAL:

- `ZERONE-1-INVENTORY-V3.json`;
- exact self-sealed `CUSTOM-STAKING-CENSUS.json` from a disposable stopped
  database copy at application height `A`;
- canonical `CUSTOM-STAKING-CENSUS-EXECUTION-EVIDENCE.json` and its detached
  transition-key `.sig`;
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
independently recompute `E` from its complete multistore roots, satisfy the
passing row reconciliations, and be hashed by FINAL. RELEASE v2 fixes the exact
census filename/hash and the execution-evidence filenames/transition signer.
The release-bound runner first executes the complete `cutover-postinit`
authority gate, verifies the copied application database against its private
byte-bearing file manifest before and after the run, snapshots the already-
authenticated census binary into a private execution path, invokes only the
fixed stdout-mode argv, captures the report in an already-open unlinked file,
and atomically publishes those exact validated bytes before emitting the
canonical receipt. FINAL binds the report, receipt, and receipt signature
bytes.

The transition custodian must run it on an isolated, stopped copy and exclude
all concurrent same-UID writers to both the copied database and its private
file manifest for the entire scan. The before/after equality checks reject
persistent drift; they cannot mechanically detect a transient write that is
restored byte-for-byte before the second scan.

This closes the previously unauthenticated report-substitution path only at the
transition key's factual-attestation trust boundary. The receipt does not
cryptographically prove that execution occurred, and its signed full-scan and
per-leaf-proof claims are not a retained root witness. It also does not provide
SBOM, Sigstore, or reproducible-build provenance for the census binary.
Production therefore remains NO-GO until those binary-provenance requirements
are independently satisfied. The structural fixture satisfies none of them.

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

The archive-gateway TOML is necessarily post-FINAL. RELEASE binds its exact
renderer/template and four static mapping fields, while OPEN binds the concrete
five-field mapping after deterministic substitution of FINAL A, lowercase E,
and lowercase B. The verifier rerenders and byte-compares it; a hand-authored
config or a template hash masquerading as deployable config bytes is invalid.

## Read-only verification

Every production verification stage, and every deployment or transaction gate
that invokes one, must run on the dedicated `linux/amd64` release workstation.
All stages execute the exact release-bound Linux/AMD64 component-signature
verifier. OPEN verification additionally executes the exact
`zeroned-zerone-1-release` binary whose hash and provenance are signed in
RELEASE. These pinned binaries are not portable to this macOS preparation
workspace; macOS shell doubles are test fixtures only. Never replace either
binary with a native rebuild, emulator wrapper, or script to make a production
check pass.

Run the verifier from the same signed release checkout whose operator-tool
bytes are named by `OPERATOR-TOOL-MANIFEST.json`. Before exact notice publication:

```bash
python3 deploy/verify-authority-chain.py notice-prepublish \
  /secure/authority-bundle '<main-full-fingerprint>' \
  --release /secure/authority-bundle/RELEASE-PACKET.json \
  --release-sig /secure/authority-bundle/RELEASE-PACKET.json.sig \
  --decision /secure/authority-bundle/PRE-NOTICE-DECISION.json \
  --decision-sig /secure/authority-bundle/PRE-NOTICE-DECISION.json.sig \
  --config-policy deploy/validate-fly-phase-config.py --tool-root .
```

The final pre-public and post-public checks are:

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
stage-appropriate DARK, PRE-NOTICE, or CUTOVER decision pair, and only the
initiation pair required by that stage. Never substitute a later-stage evidence
pair into an earlier stage.

## Keys and connected actions

The out-of-band main fingerprint signs RELEASE, all four operator decisions,
and the four initiation/registration evidence files. The distinct transition
fingerprint signs only the deterministic archive-adoption attestation, census
execution evidence, and FINAL. Neither bundle possession nor the transition
key grants deployment or transaction authority.

Connected actions use the specialized gates in the order documented in
[`CUTOVER.md`](CUTOVER.md): private successor/bootstrap first, separately
authorized notice publication and capture, CUTOVER observer
first and signer last, deterministic private archive, then the OPEN
history-link transaction, public profiles, and DNS. Real signatures, real
release artifacts, explicit human approval, and current live evidence remain
mandatory; a coherent rehearsal bundle is never production authority.

Canonical byte rules and the authority graph are in
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md); the artifact overview is in the
[`zerone-2` README](README.md).
