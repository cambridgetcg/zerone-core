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
- `zerone-component-signature-verifier` and
  `SIGSTORE-TRUSTED-ROOT.json`;
- `zeroned-zerone-1-release` and `zeroned-zerone-2-release`;
- `genesis.json`, `genesis.sha256`, `network-manifest.json`, and
  `GENESIS-MANIFEST.md`;
- for each of `zerone_1_halt`, `zerone_2_runtime`, and `query_gateway`: exact
  SBOM, provenance, signature evidence, offline signature bundle,
  vulnerability scan, and signed vulnerability decision files named by the
  release packet.

The release-bound v2 operator-tool manifest covers every verifier, transaction
gate, deployment gate, renderer, canonicalizer, ceremony/build recipe, and
artifact auditor used by the launch. It also hashes the exact executable
component-signature verifier and frozen Sigstore trusted root in the authority
bundle and fixes the accepted issuer, workflow SAN, bundle media type, SCT,
Fulcio source-commit claim, transparency-log, and observer-time thresholds. A
valid release signature is insufficient if any locally executed tool,
workflow, template, policy, or trust-root byte differs.

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
export GOTOOLCHAIN=go1.25.12 CGO_ENABLED=0 GOOS=linux GOARCH=amd64
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
successor inventory.

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
