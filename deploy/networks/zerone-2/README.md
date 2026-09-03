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

The detached-signature payload starts from
[`zerone-1/frozen/FINAL-CHECKPOINT.example.json`](../zerone-1/frozen/FINAL-CHECKPOINT.example.json),
the single authoritative v3 template. It is a manifest of hashed evidence, not
a replacement for the v3 inventory or raw RPC bundle. Do not create a second
copy under `zerone-2`; signing two independently editable templates would make
the transition payload ambiguous.

The signed phase graph is acyclic: RELEASE → DARK-START/initiation → private
registration evidence → CUTOVER/initiation → generated ARCHIVE-ADOPTION →
FINAL-CHECKPOINT → OPEN-BETA/initiation → public profiles. CUTOVER may finish
only the private historical package after its old-chain initiation event.
Public P2P, query gateways, the history-link transaction, endpoints, and DNS
require the later main-key OPEN-BETA decision. The transition key signs factual
archive and checkpoint attestations only; it never grants operator GO
authority.

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

RELEASE also pins two distinct trusted predecessor node IDs and one exact
trusted `zerone-1` block height/block ID/AppHash. It hashes the ceremony output,
the operator-tool manifest, and each component's SBOM, provenance, signature,
and vulnerability-decision evidence. It also hashes
`MONITORING-ALERTS.json`, whose exact transitive inputs are the normalized
production rules and complete alert-test evidence. Those public bytes belong in
the append-only authority bundle; a source tag, image digest, genesis hash, or
unresolved monitoring digest on its own is not sufficient release provenance.

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
