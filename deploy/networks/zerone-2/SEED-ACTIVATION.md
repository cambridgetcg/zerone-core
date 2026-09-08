# Separate seed gate — authenticated inputs, not independent chain truth

**Local implementation candidate; no activation or release acceptance is recorded here.**
The example is deliberately incomplete. Its `production` label, acknowledgement
and constants are not approval, a public phase announcement, funding, or proof a
release bundle exists. Missing inputs mean **not provided**, not **nonexistent**.

This gate comes **after** the existing dark public-beta ceremony. It neither
invokes nor changes `verify-authority-chain.py`, launch phases, genesis, CI or
existing gate files. Previously incomplete independent reviews remain incomplete.
A seed result cannot substitute for any release, custody, financial or beta gate.

## Exact scope

- One cohort of 1–32 uniquely sorted recipients; one Wallet intent per recipient.
  Native seed commitment is **222000 uzrn per address**, not new economics.
- Source/genesis/chain-bound Seed profile `0.1`, Cosmos SDK **v0.53.8**, exact
  runtime/helper/CLI/Wallet artifact digests, independently selected node trust.
  No network retargeting or historical Send/witness profile changes.
- Exact native governance admission, or the exact installed registrar. Both need
  a source-bound demonstrated executable route and independent custody attestation.
  A governance module address is **not a private key or a usable signer**. If the
  release has no functioning proposal execution/parameter-update route, stop that
  effect. This verifier does not create a route or certify one from source code.
- Complete five-field claiming-pot parameters are signed before and after. The
  only permitted v1 delta is blank → exact registrar installation; all other
  fields remain identical. Installation requires a demonstrated native governance
  route and confirmed parameter-update receipt. No rotations or unrelated delta.
- Preserve the original allocation, blank genesis registrar, empty genesis pots
  and `claims_live_at_genesis=false`. Post-beta admission does not rewrite genesis.
- Grants are finite `AllowedMsgAllowance(BasicAllowance)`, exactly `MsgClaim`,
  distinct sponsor/grantee, positive limit and explicit expiry. Grant creation
  must positively confirm the recipient Cosmos account exists; no dust funding.
- **Admitted pots do not expire.** Grant expiry/revocation or registrar removal
  stops sponsorship/new admissions, **not already committed issuance**. The signed
  `nonexpiring_pot_exposure_accepted` acknowledgement is mandatory. Revocation is
  not cancellation or budget recovery. No payout is promised; supply clipping is
  still the Claim runtime's responsibility. `QueryClaimable` is not eligibility.

## Independent public trust, not self-approval

An operator supplies a canonical `zerone-seed-trust/v1` document **outside the
bundle's authority**, its `sha256:` pin, and an exported **public-only** OpenPGP
keyring. The trust document has exactly these fields:

```
schema, environment, activation_id, chain_id, genesis_hash, profile_id,
node_trust_id, roles, keyring_sha256, beta_verification_sha256,
verifier_sha256, bundle_manifest_sha256
```

`roles` has exactly `release`, `activation`, `reviewer`, `observer`, `custodian`:
five distinct, full uppercase 40/64-hex **signing-key** fingerprints (a signing
subkey needs its own fingerprint, not just its primary key's). These role names
and distinct keys do not prove distinct humans, organizational independence or
review quality; establishing that is an out-of-band acceptance responsibility.
Here `verifier_sha256` pins the **prior authority verifier**, while the activation
packet's `artifacts.verifier_sha256` pins **this seed verifier**.

The operator also independently selects the exact activation file hash and fresh
stage-evidence hash. Do not mechanically derive these trust pins from an
untrusted bundle. The keyring hash is checked before use. Verification uses fixed
`gpgv` arguments, an isolated temporary public-only home, SHA-2 signatures and
exactly one matching `VALIDSIG` per signature. There is no key discovery, agent,
network, packet-provided command, signing, broadcast or persistent gate state.
`--gpgv` must name the operator-approved absolute executable, not a bundle script.

Custody values are public opaque binding IDs, **never keys, mnemonics, private
paths, provider credentials or signed transaction bytes**. Native route evidence
is attested by custodian and reviewer; node evidence by the observer. This is
accountable testimony under explicit trust, **not portable state/Merkle proof or
independent observation by this verifier**.

## Public input files

All files below live in an explicitly supplied local `--bundle` directory.
No file location or command is selected by packet content.

| File | Required signature / meaning |
|---|---|
| `SEED-ACTIVATION.json` | `.sig` activation + `.review.sig` independent reviewer |
| `RELEASE-PACKET.json` | `.sig` release; actual accepted public document |
| `OPEN-BETA-DECISION.json` | `.sig` release; actual GO document linked to RELEASE |
| `OPEN-BETA-INITIATION-EVIDENCE.json` | `.sig` release; exact successful history-link confirmation |
| `RELEASE-VERIFY.stdout` | actual successful `dark-preinit` stdout |
| `OPEN-BETA-VERIFY.stdout` | actual successful `open-postinit` stdout |
| `BETA-VERIFICATION.json` | `.sig` reviewer; independently pinned execution receipt |
| `SEED-CUSTODY.json` | `.sig` custodian; exact packet, IDs, host and available route |
| `NATIVE-ROUTE-EVIDENCE.json` | hash-bound by packet, custody and reviewer |
| `NATIVE-ROUTE-DEMONSTRATION.json` | actual public demonstration report bytes, hash-bound by route |
| `SEED-PREACTIVATION.json` | `.sig` observer |
| `SEED-POSTACTIVATION.json` | `.sig` observer; required for postactivation/operation |
| `SEED-OPERATION.json` | `.sig` observer + `.custody.sig` custodian; operation only |
| `SEED-PRESIGN.json` | exact host `pre-sign` candidate, canonical without LF; its bundle/observation/coordinates are committed by the signed operation |
| `evidence/<64-lowercase-digest>.json` | exact decoded native confirmation receipts referenced by stages |

The existing authority verifier emits only `authority-chain: MATCH` plus LF.
That text alone authenticates **nothing**. `BETA-VERIFICATION.json` is a separate
reviewer-signed execution attestation with exactly:

```
schema="zerone-seed-beta-verification/v1", environment, chain_id, genesis_hash,
verifier_sha256, bundle_manifest_sha256, verified_at, documents, runs,
dark_genesis, unrelated_policy_hash
```

`documents` maps the six RELEASE/OPEN document-and-signature filenames to their
raw byte hashes. `runs` contains exactly `{stage, exit_code, stdout_sha256}` for
successful `dark-preinit` and `open-postinit`, in that order. `dark_genesis` is
`{claims_live_at_genesis:false, bootstrap_registrar:"", pots:[]}`. The unrelated
policy hash is the semantic hash of RELEASE's complete `accepted_policy` object.
The prior verifier source and actual public bundle manifest are separately
required, hash-checked inputs; this consumer **does not rerun the prior verifier**
or independently prove its execution. An honest reviewer must have witnessed the
actual successful runs over that exact bundle before signing. Do not wrap fixture
stdout, templates or a failed/incomplete review as production execution evidence.

The route receipt fields are `schema="zerone-seed-native-route/v1", environment,
profile_id, mechanism, authority, execution_path, source_manifest_sha256,
demonstration_sha256, parameter_update_supported`. Execution paths are closed to
native `MsgAddBootstrapEntry`, directly for registrar or via
`cosmos.gov.v1.MsgSubmitProposal` for governance. No executable argv is accepted.
A demonstration hash alone does not establish route availability: custodian and
reviewer attest its release-matched interpretation. Absence or uncertainty blocks.

`--artifact-root` must contain exact `zeroned`, `zerone-seed-io`, `wallet.tgz`,
`zerone-seed-cli`, and `source-manifest.json` bytes. Artifact hashes authenticate
selected bytes, not reviewed correspondence, release provenance, running custody,
or all interpreter/dependency behavior. Establish those at the external release
and host gates; do not supply a wrapper while claiming to pin an entire runtime.

## Three checks; the host consumes the third before signing

1. **preactivation**: after verified beta, before finite admission deadline.
   Fresh coherent configured-node observations cover all recipients, absent pots
   and grants, complete parameters, available native authority, lifetime/window
   admission headroom, dedicated funded sponsor and zero prior setup effects.
2. **postactivation**: validate the historical signed preactivation snapshot, then
   fresh exact pots/vesting, positive account creation and finite grants. Require
   distinct successful admission/grant transactions, exact decoded effect receipts,
   inclusion block/time/hash and canonical successor, confirmed fees, full parameter
   result and exact lifetime commitment increment. No inference from broadcast ACKs.
3. **operation**: validate historical pre/post snapshots and a fresh observation
   for **one exact cohort recipient**. Bind its policy, plan/commitment, Wallet
   descriptor/capability/intent/simulation records, original prepared time, source,
   exact bundle and observation hashes, inert request ID and SignDoc hash, key ID,
   account/sequence, independent fee/gas ceilings, exact narrowed commitment expiry,
   host/ledger binding, real journal prestate and currentness/budget snapshot hashes.
   The **latest observed height**, not merely the anchor, must be below timeout.
   Observer and custodian sign the same operation evidence. Its
   `operation_id` is this gate's hash of the operation object excluding that ID;
   it is an additional gate commitment, not a replacement for the runtime plan ID.

Public evidence uses the exact `zerone-seed-evidence/v1` field sets in the verifier
and test builder. Snapshot anchors carry height/hash/time; catching-up,
incoherent, stale/future, unexpected commitments or native-window drift refuse.
Postactivation setup must follow the preactivation height. Every occurrence of a
height across all stage anchors, parameter/admission/grant inclusions and successor
receipts must have the same block hash (and time wherever supplied). A signed
`coherent:true` cannot hide contradictory bindings. These joins do not prove
unprovided headers or independently observe a chain. Previous evidence hashes
bind history without cyclic signing. Decoded native receipts bind
`schema="zerone-seed-native-confirmation/v1", environment, profile_id,
node_trust_id, confirmation, effect`; arbitrary extra fields and signed TxRaw are
not allowed. Receipt contents remain configured-observer attestations.

Budgets are canonical integer **strings**, not floats. Reserve the **full sum of
finite grant limits**, plus separately approved setup limits; never claimant
balance, incoming seed or estimated Claim fees. Enforce both approved lifetime
commitments and the native cap. Confirmed setup fees must explain exact spending.
Both gate and host conservatively retain **full grant plus full setup lifetime
exposure**, including confirmed setup spending. With grant=200000, setup=200000
and spent=100000, balance=300000 refuses; balance=400000 can pass only if no other
exposure remains. `sponsor_other_exposure_uzrn` additionally covers outside-journal
liabilities and journal operations outside this exact cohort. The host reconciles
that amount with its signed budget and actual journal, counts the cohort only
once, and rechecks the entire sum before submission. No used grant or unused setup
headroom is reclaimed, refilled or funded automatically.

The operation status is **`unreserved`**: this is an inert proposal, not a journal
state. The host has zero previous intent use and an available, exact prestate;
`required_grant_exposure_uzrn` and `required_setup_exposure_uzrn` name amounts it
must reserve, not effects already performed. `signing_unknown` is never renamed
unsigned. There is no reusable signing slot and no request before the atomic
boundary. The first operation event remains `sign_boundary`, which consumes the
one use, attempt, sequence and exposure and enters `signing_unknown` together.

### Enforced host interface

1. Configure the independently pinned activation root, initialize the private
   journal explicitly, then run `prepare` and `pre-sign --plan <prepared-file>`.
   `pre-sign` observes/simulates and seals the public Wallet simulation record;
   it opens **no chain signer** and makes **no journal write**. Preserve its entire
   candidate as `SEED-PRESIGN.json`, without LF, and retain original `prepared_at`.
2. The observer/custodian bind that candidate's `operation` in `SEED-OPERATION.json`.
   The gate checks candidate hashes and exact joins between its native observation
   and signed stage evidence. It authenticates attestations, not host/chain truth.
3. Independently select the operation-evidence hash in the host configuration.
   `reserve-sign --plan <candidate-file>` invokes the pinned external verifier
   itself. A caller-provided saved JSON success result is not accepted. The host
   compares the executing compiled CLI's actual digest with the packet CLI pin.
4. Inside one immediate SQLite transaction, reconstruct/verify the original Wallet
   records and exact plan/bytes/simulation; re-read and authenticate the named
   currentness/budget files; compare the exact journal head/status, ledger binding,
   bundle, observation and request coordinates; recheck fresh real native state,
   wallet-wide currentness timestamp/nonce highwaters and aggregate funding. Only
   then generate the one-use signing request and append `signing_unknown`.
5. All signer failures after commit remain sticky, even missing keys. Verification
   may recover the same signed file only. Submission reauthorizes and conserves the
   full cohort exposure; positive failed inclusion need not advance sequence, so
   its fence remains when advance is not actually observed. No gate retry refunds use.

The closed host `activation_gate` configuration is either the explicit test-only
`{mode:"disposable-local"}`, or the following pinned external interface (no packet
chooses a path, program, argument vector or code):

```
mode: "production" | "synthetic"
python: {path, sha256}
verifier: {path, sha256}
codec_sha256                  # adjacent frozen_evidence.py, explicitly loaded
gpgv: {path, sha256}          # no ambient keyring/config
bundle, trust_file, trust_sha256, packet_sha256
evidence_sha256              # null only while producing pre-sign; signing refuses
public_keyring, artifact_root, beta_bundle_manifest, authority_verifier
```

All paths are absolute local operator inputs. Python runs with `-I -B` and fixed
bootstrap code that loads only the pinned codec and verifier, without adding a
bundle or source directory to `sys.path`. Executable/codec drift refuses. The
journal pins the activation root; only its per-operation evidence hash can change.
Missing gate configuration fails closed. Production requires `zerone-2` and
`disposable_test=false`; synthetic and omission modes require explicit disposable
local chains (`seed-local-*`; standalone synthetic gate tests also accept
`seed-synthetic-*`). There is no production omission flag or arbitrary verifier
argv field. Test output remains `SYNTHETIC_MATCH`.

This is a cooperative, one-host boundary, not a sandbox against a privileged
same-user process, an independently invoked signer or replacement of all local
state/configuration. Byte pins do not attest the Python stdlib, shared libraries,
OS or reviewer independence. Gate rereads are not global chain truth, distributed
locks, automatic release acceptance or a production activation receipt.

## Encoding and invocation

Use `scripts/zerone-canonical-json.sh` for public **ceremony files**: sorted compact
UTF-8 JSON with exactly one LF, then detached OpenPGP signatures over those exact
bytes. The verifier additionally rejects duplicate keys, unknown fields, unsafe
numbers, control/non-ASCII strings, placeholders and resource excess. JSON is
bounded to 256 KiB, depth 32 and 4096 values; files are regular, single-link,
no-symlink paths opened component-by-component. Input mutation refuses.

**Different hash domains:** Seed profile/policy semantic hashes use Wallet
canonical bytes **without LF**, excluding only `profile_id`/`policy_hash`.
`source-manifest.json` and runtime `SEED-PRESIGN.json` also have no LF. Ceremony
file hashes include their actual LF. The pre-sign bundle may contain Wallet's
valid Unicode record strings; its public operation coordinates and ceremony
files remain printable ASCII. The host, not Python, verifies Wallet signatures. `sha256:` IDs are lowercase; transaction hashes uppercase. See AgentTool
`docs/specs/ZERONE-SEED-IO-0.1.md`; do not change its policy contract to fit this gate.

```
python3 -B deploy/verify-seed-activation.py --help
python3 -B deploy/verify-seed-activation.py preactivation \
  --bundle /ABSOLUTE/APPROVED/PUBLIC-SEED-BUNDLE \
  --trust /ABSOLUTE/INDEPENDENT/PUBLIC-TRUST.json \
  --trust-sha256 sha256:INDEPENDENT_TRUST_DIGEST \
  --packet-sha256 sha256:INDEPENDENT_PACKET_DIGEST \
  --evidence-sha256 sha256:INDEPENDENT_FRESH_EVIDENCE_DIGEST \
  --public-keyring /ABSOLUTE/INDEPENDENT/PUBLIC-ONLY.gpg \
  --gpgv /ABSOLUTE/APPROVED/gpgv \
  --artifact-root /ABSOLUTE/EXACT-RELEASE-ARTIFACTS \
  --beta-bundle-manifest /ABSOLUTE/VERIFIED-BETA-MANIFEST.json \
  --authority-verifier /ABSOLUTE/RELEASE-MATCHED/verify-authority-chain.py
```

`postactivation` uses the same inputs plus its signed snapshot; `operation` also
requires its signed snapshot and `--operation-id sha256:EXACT_OPERATION_DIGEST`.
Production uses the real UTC clock; `--now` is rejected. Success prints
`AUTHENTICATED_INPUTS_MATCH`, `chain_truth=not_independently_verified`, and
`effects=none`. Failure exits 1 with `REFUSED`; neither result activates anything.

Tests require Python 3.10+, **test-only** `cryptography`, and real `gpgv`:

```
python3 -B deploy/test-seed-activation.py
```

Tests generate disposable in-memory RSA OpenPGP keys and independently verify
signatures with gpgv. They contact no node and cannot select production: synthetic
trust, receipt, packet and `seed-synthetic-*` chain must agree; output is always
`SYNTHETIC_MATCH`. Fixture artifacts are not usable releases. Tests establish local
acceptance/rejection behavior, not real beta acceptance or native route execution.

## Still-required operational inputs

Actual accepted public release bundle/manifest and verifier, public trust anchors
and independent successful-run receipt; reviewed exact artifacts and source
manifest; recipients, sponsor, custody/host bindings; funded finite gas/setup
limits; demonstrated native authority; deadlines, parameter delta if any,
independent signatures and fresh positive chain/host evidence. No production
numeric budget, signer identity or funding source is supplied by this template.
