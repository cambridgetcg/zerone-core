# `zerone-2` public relaunch artifacts

This directory contains the keyless ceremony contract and public transition
templates for `zerone-2`. It must never contain real mnemonics, private keys,
node keys, validator signer state, passwords, seeds, deployment tokens, or
unencrypted recovery material.

The production artifact directory is created outside the repository first and
is copied here only after the final public hashes are fixed. A production run
refuses to overwrite an existing output directory.

## Historical boundary

The predecessor checkpoint is a three-height contract, never a single “halt
height”:

- checkpoint state `F` is the height-pinned inventory and eligibility input;
- empty final committed block `A=F+1` has a signed header `AppHash` that anchors
  state `F`;
- `H=A+1` triggers the SDK halt in `FinalizeBlock(H)`.

At the stable halt, BlockStore/status contains empty staged block `H`, but ABCI
last applied height is `A`; `/commit A` is canonical and `/commit H` returns the
subjective tip seen commit with `canonical=false` because no `H+1` exists. That
tip flag alone is not canonicality proof. Block results for `H` do not exist.
Explicit policy ends official application/history at `A` and selects `F` for
`zerone-2` inventory and eligibility.

Both the old signer and independent observer must be armed with the same `H`
while transaction ingress and both mempools are closed before `F`. A halted SDK
does not imply a stopped daemon: RPC and ordinary health checks can remain
green. After bounded stability checks, operators manually SIGTERM and fence the
signer—but first capture the v3 inventory and byte-exact H/A RPC evidence. A
hard rollback on the stopped serving copy removes pending H while preserving
state/app A; the fresh-key public archive is A/A and block H is absent. Generic
`catching_up` is not its readiness test. As the new BlockStore tip, archive
`/commit A` reports `canonical=false`; the signed pre-fence H/A bundle retains
the canonical A commit proof. REST inventory responses are
height-pinned to `F` but are trusted-source data, not Merkle proofs; the signed
manifest discloses that limitation and preserves hashes of the raw evidence.

The authoritative operational sequence and evidence gates are in
[`CUTOVER.md`](CUTOVER.md) and [`GO-NO-GO.md`](GO-NO-GO.md). The checklist is
not signed; the immutable release packet, three scoped operator decisions,
four main-key initiation/registration evidence attestations, and two narrow
transition-key attestations follow
[`CANONICAL-SIGNING.md`](CANONICAL-SIGNING.md). The exact append-only file set
and five fail-closed verification stages are documented in
[`AUTHORITY-BUNDLE.md`](AUTHORITY-BUNDLE.md).

Preparation for attention gathering and external explanation follows
[`LAUNCH-COMMUNICATIONS.md`](LAUNCH-COMMUNICATIONS.md). That plan is not
authority to publish: it keeps claims phase-scoped, evidence-linked, and
explicit about the custodial and NO-GO boundaries.

The chain/release signature domains and authoritative-time rules are fixed in
[`docs/SIGNATURE-AND-TIME.md`](../../../docs/SIGNATURE-AND-TIME.md).

The detached-signature payload starts from
[`zerone-1/frozen/FINAL-CHECKPOINT.example.json`](../zerone-1/frozen/FINAL-CHECKPOINT.example.json),
the single authoritative v4 template. It is a manifest of hashed evidence, not
a replacement for the v3 inventory or raw RPC bundle. Do not create a second
copy under `zerone-2`; signing two independently editable templates would make
the transition payload ambiguous.

The signed phase graph is acyclic: RELEASE → DARK-START/initiation → private
registration evidence → CUTOVER/initiation → generated ARCHIVE-ADOPTION plus
generated custom-staking census execution evidence → FINAL-CHECKPOINT →
OPEN-BETA/initiation → public profiles. CUTOVER may finish
only the private historical package after its old-chain initiation event.
Public P2P, query gateways, the history-link transaction, endpoints, and DNS
require the later main-key OPEN-BETA decision. The transition key signs factual
archive and checkpoint attestations only; it never grants operator GO
authority. RELEASE v2 additionally binds the exact census binary and execution
evidence contract. Its transition signature remains a factual operational
attestation, not proof of process execution or binary provenance. RELEASE binds
only the archive gateway's static mapping, renderer, and template; verified
FINAL supplies A/E/B, and OPEN binds the exact deterministically rendered
public config hash.

## Ceremony

```bash
make build

# Deterministic, public-fixture rehearsal only:
scripts/zerone-2-ceremony.sh drill /tmp/zerone-2-drill-a
scripts/zerone-2-ceremony.sh drill /tmp/zerone-2-drill-b
cmp /tmp/zerone-2-drill-a/genesis.json /tmp/zerone-2-drill-b/genesis.json

# Real ceremony; both inputs contain public data only. Run this on the target
# Linux architecture with a binary built from the signed release commit:
ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT='<40-or-64-lowercase-hex>' \
BINARY=/secure/zeroned-linux-amd64 \
  scripts/zerone-2-ceremony.sh real \
  /secure/public-input.json \
  /secure/gentx-zerone-2-custodian.json \
  /secure/zerone-2-artifacts
```

The real input follows [public-input.example.json](public-input.example.json).
The `source_commit`, binary hash, Linux GOOS/GOARCH, signed annotated tag, and
OpenPGP signer fingerprint must all agree. The signer fingerprint is supplied
both in the public input and independently through the environment. The copied
binary's embedded VCS revision must equal the release commit. The gentx must be
pre-signed offline by the new validator operator and contain the exact
self-bond and commission profile.

The ceremony also replaces the SDK's short default slashing window with one
explicit launch policy: `signed_blocks_window=34272`,
`min_signed_per_window=0.950000000000000000`,
`downtime_jail_duration=3600s`,
`slash_fraction_double_sign=0.050000000000000000`, and
`slash_fraction_downtime=0.000100000000000000`. At the runtime's 2.521-second
target commit cadence, the window is approximately one day and the 95% floor
allows approximately 72 minutes of total missed signing. Release-bound
monitoring alerts on the first missed signature. In the disclosed one-validator
set, however, no block can commit while that validator is absent, so the
downtime jail/slash cannot execute. Those parameters remain dormant until the
set can commit without one validator; monitoring and operator recovery are the
only launch availability controls. Double-signing retains the SDK's 5% slash
and permanent tombstone behavior.

The runtime reasserts `timeout_commit=2521ms` and
`skip_timeout_commit=false` on every start, rejects all `ZERONED_*` environment
overrides, and passes `--min-retain-blocks 0` explicitly. Comet evidence
admission is separately pinned to 719,714 blocks and
1,814,400,000,000,000 nanoseconds—approximately 21 days at the target cadence,
matching the SDK staking unbonding period. This keeps delayed equivocation and
light-client evidence admissible while the responsible stake remains at risk.
The runtime retains Comet history (`min-retain-blocks=0`); capacity monitoring
and backup freshness are launch gates.

RELEASE also pins two distinct trusted predecessor node IDs and one exact
trusted `zerone-1` block height/block ID/AppHash. It hashes the ceremony output,
the operator-tool manifest, and each component's SBOM, provenance, signature,
and vulnerability-decision evidence. It also hashes
`MONITORING-ALERTS.json`, whose exact transitive inputs are the normalized
production rules, the v2 alert-test document, and its 40 exact raw stimulus,
firing, notification, and resolution evidence files. Those public bytes belong
in the append-only authority bundle; a source tag, image digest, genesis hash,
or unresolved monitoring digest on its own is not sufficient release
provenance.

The ceremony emits public artifacts only:

- `genesis.json`
- `genesis.sha256`
- `network-manifest.json`
- `GENESIS-MANIFEST.md`

Run the mandatory artifact auditor with `--required-mode real` on the result.
A successful Cosmos genesis validation alone is not enough; it does not
enforce the relaunch policy.

## Never reuse

Do not reuse `scripts/mainnet-ceremony.sh`: it creates five genesis validators
and a different economic/bridge profile. Do not reuse any file in
`deploy/mainnet/artifacts/` except the public `zerone-1` genesis and manifest as
historical evidence.
