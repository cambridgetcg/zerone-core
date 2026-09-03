# Canonical authority and attestation bytes

One immutable release root and three independently immutable operator
decisions define the launch:

1. `RELEASE-PACKET.json`, made from `RELEASE-PACKET.example.json`, fixes the
   release components, policies, public identities, invariant config mappings,
   phase-dependent template hashes, and both deterministic archive renderers.
2. `DARK-START-DECISION.json` references the release bytes and may authorize
   only the private successor boot, registration, and soak.
3. `CUTOVER-DECISION.json` references the unchanged release packet, dark-start
   decision/signature, completed soak evidence, exact F/A/H, halt configs, and
   a narrowly deterministic private archive continuation.
4. `OPEN-BETA-DECISION.json` is created only after the private archive and final
   checkpoint exist. It references every prior authority/evidence hash and may
   authorize only the exact history-link transaction followed by the public
   profiles, coordinates, DNS manifest, and evidence publication.

The main release/operator key signs all four. The fingerprint is the same
independently pinned fingerprint used to verify the signed release tag. It is
supplied out-of-band before the ceremony and must equal the single `VALIDSIG`
fingerprint for every payload.

Four main-key initiation/evidence files are factual evidence, not new decisions:

- `DARK-START-INITIATION-EVIDENCE.json` proves private block 1 committed no
  later than DARK-START's deadline. It permits private continuation after the
  deadline; before initiation, the deploy gate checks wall-clock expiry.
- `DARK-REGISTRATION-EVIDENCE.json` proves the exact two pre-signed private
  bootstrap TxRaw envelopes committed in order and produced the declared
  identity/custom-validator state before the signed registration deadline.
- `CUTOVER-INITIATION-EVIDENCE.json` proves the exact notice publication and
  pre-signed z1 successor transaction committed successfully and on time. It is
  required before halt signer/observer deployment and before archive rendering.
- `OPEN-BETA-INITIATION-EVIDENCE.json` proves the exact pre-signed z2 history
  link committed successfully and on time. It is required before any public
  profile deployment.

Canonicalize, sign, and independently verify each with the main operator key
after its event. Their result is `MATCH`; they add no scope beyond forward
completion already granted by the corresponding decision.

Two transition-key files are factual attestations, not operator decisions:

- `ARCHIVE-ADOPTION-AUTHORITY.json` is emitted—never hand-authored—by
  `deploy/mainnet/render-archive-configs.sh`. It binds the inner transition
  manifest, verified release/CUTOVER payload and signature hashes, exact
  deterministic templates/renderer, construction evidence, and rendered
  private archive config bytes. Its result is `MATCH`, not `GO`.
- `FINAL-CHECKPOINT.json` attests the frozen historical evidence and private
  A/A archive readiness. It contains no OPEN-BETA reference, history-link
  transaction, public URL, DNS change, or public deployment authority.

Both use the separate transition-attestation OpenPGP fingerprint fixed in the
release packet and CUTOVER delegation. That fingerprint must differ from the
main key. It cannot authorize a public service or operator decision.

The resulting acyclic graph is:

```text
RELEASE -> DARK-START -> dark-initiation -> registration-evidence -> CUTOVER
                                            |
                                            v
                                  cutover-initiation -> ARCHIVE-ADOPTION
                                                            |
                                                            v
                                                    FINAL-CHECKPOINT
                                                            |
                                                            v
                                                       OPEN-BETA
                                                            |
                                                            v
                                          open-beta-initiation -> PUBLIC
```

The inner archive-transition manifest contains only halt/source/candidate and
pre-transition construction evidence. Candidate/final configs contain its
hash. The adoption attestation then hashes those configs. Readiness comes next,
then FINAL, then OPEN-BETA. No earlier file contains a later file's hash.

The exact append-only inventory and verifier stage contract are in
[`AUTHORITY-BUNDLE.md`](AUTHORITY-BUNDLE.md). The connected-action order and
current command lines are in [`CUTOVER.md`](CUTOVER.md).

## Exact canonical bytes

Never edit a canonical payload after signing it. If any release-packet field
changes, issue a new packet and all dependent decisions. Detached signatures
use the filename declared inside each payload and live beside—not inside—the
signed JSON. A payload never contains its own signature hash.

For each private draft, independently validate its schema, evidence,
initiation-deadline semantics, decision (`GO` or `NO-GO`) or attestation result,
and scope. Create the exact bytes with the tested no-overwrite helper:

```bash
scripts/zerone-canonical-json.sh REPLACE_DRAFT_JSON REPLACE_CANONICAL_JSON
```

Canonical bytes are one UTF-8 compact JSON object with recursively sorted keys
and one trailing LF. Record the `jq` version and executable SHA-256. Reproduce
the same output hash on a second offline machine, sign the canonical file, and
verify the detached signature there. Require exactly one valid signature and
require its `VALIDSIG` fingerprint to equal the appropriate out-of-band full
fingerprint. A valid signature from any other key is a no-go.

The helper uses a private same-directory temporary file and atomic hard-link
publication. It refuses an existing file or dangling symlink instead of ever
hashing stale output. Run `bash scripts/zerone-canonical-json-test.sh` as a
release gate.

## Deterministic archive adoption bytes

After the halt evidence and pre-transition sanitized-copy evidence exist,
verify the release and CUTOVER signatures, then run on two isolated machines:

```bash
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<main-full-fingerprint>' \
  deploy/mainnet/render-archive-configs.sh \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  CUTOVER-DECISION.json CUTOVER-DECISION.json.sig \
  CUTOVER-INITIATION-EVIDENCE.json CUTOVER-INITIATION-EVIDENCE.json.sig \
  zerone-1-archive-transition.json REPLACE_NEW_OUTPUT_DIRECTORY \
  REPLACE_AUTHORITY_BUNDLE_DIRECTORY
```

The renderer snapshots every input before reading it, verifies both main-key
signatures, requires the exact signed renderer/template/static constraints,
and emits only:

- `fly.archive-candidate.toml`;
- `fly.archive.toml`;
- canonical `ARCHIVE-ADOPTION-AUTHORITY.json`.

Require byte equality for all three outputs on the second machine. Sign only
the emitted adoption JSON with the authorized transition key. Do not fill the
example by hand. A different app, volume, region, image, role, strategy,
topology, substitution set, or config byte requires a new main-key authority;
the transition key has no discretion to choose one.

## Deterministic public archive-gateway bytes

RELEASE fixes the archive gateway app/role/image, the public template hash, the
renderer hash, and declarative bindings to the private archive app/region. It
cannot fix the concrete config hash because its runtime pins—A, the excluded
post-A AppHash E, and the application block ID B—exist only in verified FINAL.
After FINAL is signed and independently verified, render on two isolated
machines:

```bash
python3 deploy/query-gateway/render-archive-gateway-config.py \
  RELEASE-PACKET.json FINAL-CHECKPOINT.json \
  deploy/query-gateway/fly.zerone-1-archive.public.example.toml \
  fly.zerone-1-archive-gateway.public.toml
```

Require byte equality. OPEN, signed by the main key, contains the resulting
five-field concrete mapping. The authority verifier independently rerenders
from authenticated RELEASE and FINAL and byte-compares it; only the two z2
public mappings remain byte-identical to RELEASE. Never put a template hash in
the RELEASE mapping's `sha256` field or let the transition key authorize public
exposure.

## Signed-authority deployment gate

Production DARK/OPEN Fly deployments use `deploy/fly-deploy-authorized.sh`;
CUTOVER uses only `deploy/mainnet/fly-cutover-authorized.sh`, not manually
transcribed expected values. It snapshots RELEASE, the canonical phase
authority, both signatures, and the config; verifies the exact out-of-band
signer and release-to-phase chain; extracts the app, image, role, component,
and concrete config hash from the signed phase `GO`; then passes those values
to the lower-level immutable-image gate. Example:

```bash
# Valid only before private block 1 and while DARK-START is unexpired:
deploy/fly-deploy-authorized.sh --check \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  DARK-START-DECISION.json DARK-START-DECISION.json.sig \
  /secure/fly.edge.toml zerone_2_edge_private \
  '<main-full-fingerprint>'

# After block 1, every remaining DARK profile switch requires its evidence:
deploy/fly-deploy-authorized.sh --check \
  RELEASE-PACKET.json RELEASE-PACKET.json.sig \
  DARK-START-DECISION.json DARK-START-DECISION.json.sig \
  /secure/fly.edge.toml zerone_2_edge_private \
  '<main-full-fingerprint>' \
  DARK-START-INITIATION-EVIDENCE.json \
  DARK-START-INITIATION-EVIDENCE.json.sig
```

Only DARK-START may deploy private successor profiles; the specialized CUTOVER
gate enforces observer-first/signer-last deployment; ARCHIVE-ADOPTION may
deploy deterministic private candidate/final archive profiles, and OPEN-BETA
may deploy the three public profiles. CUTOVER and OPEN require their verified
initiation-evidence payload/signature pairs. OPEN additionally requires the
transition-signed
`FINAL-CHECKPOINT.json`, its detached signature, and the separately pinned full
transition-key fingerprint. Production OPEN verification runs on the dedicated
`linux/amd64` release workstation because it executes the exact signed
Linux/AMD64 predecessor halt binary for offline Comet cryptography; native
macOS rebuilds and test doubles are forbidden. Archive candidate/final
deployments use
`deploy/mainnet/fly-deploy-archive-authorized.sh`, which reruns the renderer and
byte-compares its output before accepting the transition signature. The release
packet alone authorizes no deployment.

## Exact phase transaction bytes

CUTOVER and OPEN-BETA each bind a pre-signed transaction's encoded TxRaw
SHA-256 and uppercase Comet transaction hash. Broadcast only with
`scripts/zerone-phase-tx-broadcast.sh`. The two DARK bootstrap transactions use
`scripts/zerone-2-bootstrap-tx-broadcast.sh`. Each gate snapshots its direct
inputs, signatures, transaction JSON, and the release binary; verifies the
transitive authority bundle and unexpired broadcast/commit deadlines; checks
the binary hash; decodes the transaction; verifies the private RPC identity and
trusted anchor; then encodes the snapshot and submits those exact raw bytes.
The DARK gate additionally enforces the two fixed filenames, message types,
sequence `0` then `1`, increasing timeout heights, DID/operator/consensus-key
bindings, and proof that onboarding committed before validator registration.
The CUTOVER/OPEN gate enforces the exact one-message self-send, amount, fee,
gas, signer, timeout, memo, and extension-free semantics. A direct `zeroned tx
broadcast` command is not an authorized release path.

After successful commit, create the corresponding canonical main-key
initiation-evidence payload from independent raw query evidence. No halt/public
deployment is unlocked by CheckTx alone.
