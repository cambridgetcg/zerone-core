# `zerone-1` frozen historical package

This directory is the public immutable end-of-chain record. Populate it only
after the halted `H/A` boundary has been captured from the ingress-fenced live
signer and independent observer, both processes have been manually stopped,
and the old signer has been fenced against restart.

The exact contract is checkpoint state `F`, empty canonically committed anchor
`A=F+1` whose header anchors `F`, and empty SDK trigger `H=A+1`. Comet stages
`H` before `FinalizeBlock(H)` fails, leaving BlockStore/status `H`, ABCI `A`,
and no application results for `H`. `/commit A` is canonical. `/commit H`
returns the subjective tip seen commit with `canonical=false` because `H+1`
does not exist; that flag does not itself prove noncanonicality. Explicit
transition policy selects `A` as the final application/history boundary and
`F` as successor inventory.

Before fencing, preserve byte-exact raw responses for status and genesis; the
RELEASE trusted block/commit/complete validator set; blocks, commits, and
complete validator sets at `A` and `H`; `block_results(A)`; ABCI info; and the
missing `block_results(H)` response. The package contains their hashes and
signed manifest so the release halt binary can recompute block IDs, verify
commit voting power/signatures and trusted-set continuity, and keep staged `H`
independently inspectable after it is removed from the serving copy.

The package contains:

- original `genesis.json` and `GENESIS-MANIFEST.md`;
- `FINAL-CHECKPOINT.json`, completed from the v3 manifest example here;
- the `zerone-relaunch-snapshot-v3` inventory and eligibility output hashes;
- raw terminal RPC evidence plus its SHA-256 manifest;
- exact `CUSTOM-STAKING-CENSUS.json` bytes produced from a disposable copy of
  the stopped observer application database at applied height `A` and the
  excluded post-anchor AppHash;
- the sanitized `A/A` archive snapshot and rollback log hashes;
- `FINAL-CHECKPOINT.sha256`, `FINAL-CHECKPOINT.json.sig`, and final
  `TRANSITION.md`.

`FINAL-CHECKPOINT.json` is the signed package manifest; it is not the v3
inventory itself. Never include node/consensus keys, signer state, mnemonics,
tokens, environment files, or an unsanitized private database.

The custom-staking census is deliberately an `A`-state custody/claimant audit,
not checkpoint-`F` successor inventory. Preserve the tool's compact,
self-sealed output bytes exactly; do not pass them through the FINAL canonical
JSON helper. FINAL hashes the sealed file, while the release-bound frozen
validator independently requires `PASS`, verifies its self-hash, binds its
lowercase AppHash to ABCI/excluded post-anchor state at `A`, binds its source
commit to RELEASE, and checks `B = D + U`, zero delta, empty findings, and the
complete claimant root. The report neither migrates a balance nor makes the
legacy `consensus_pubkey` authoritative.

The current RELEASE component model does not yet carry a dedicated census
executable hash, SBOM, provenance, and signature. A real FINAL remains NO-GO
until those bytes are reproducibly built from the signed RELEASE source and
their provenance is independently verified and bound. The authority fixture
only rehearses report/file semantics; it is not binary provenance.

## Exact signed bytes

There is one template only: `FINAL-CHECKPOINT.example.json` in this directory.
Fill a private working copy and validate every hash and the `F/A/H`
relationships independently. For this transition, canonical JSON means one
UTF-8 compact object with keys recursively sorted by `jq`, plus its single
trailing LF. Use the same tested atomic no-overwrite helper as the release and
decision packets:

```bash
scripts/zerone-canonical-json.sh \
  FINAL-CHECKPOINT.draft.json FINAL-CHECKPOINT.json
```

Record the `jq` version and executable SHA-256 in the ceremony log. Repeat the
canonicalization on the second offline machine and require the exact same file
hash before signing. Sign `FINAL-CHECKPOINT.json` itself, not a pretty-printed
copy, its hash text, or the example template. The later `zerone-2` transaction
that commits this manifest's hash is recorded beside the immutable manifest;
putting that future transaction hash inside the hashed bytes would be circular.

The manifest references the verified RELEASE, DARK-START, CUTOVER, and
deterministically generated ARCHIVE-ADOPTION payload/signature hashes, the
inner transition-manifest hash, candidate readiness, final runtime marker, and
private A/A probe evidence. Sign it with the full transition-attestation
OpenPGP fingerprint authorized by RELEASE/CUTOVER. It must not contain a future
OPEN-BETA hash, link-transaction hash, public URL, DNS change, or public config
authority. Those are bound later by the main-key OPEN-BETA decision.

The restartable private archive origin is made only from a stopped copy. In Comet's
pending-block special case, `zeroned rollback --hard` deletes staged `H` while
preserving state/application `A`; replace all node and validator keys before
startup. The serving archive is therefore `A/A`, and `H` must be absent. A
fresh non-validator key causes Comet v0.38 to report `catching_up=true` with no
peers even while serving the frozen data, so generic sync health is unsuitable.
Readiness must instead require the expected chain ID, status height `A`, ABCI
height and hash `A`, block `H` absent, and query-only ingress.

The v3 snapshot must be captured on the live halted `H/A` source before
fencing; it cannot run against the sanitized `A/A` archive. REST responses are
height-pinned to `F` but are not Merkle-proved, so the signed manifest must name
the trusted source and independent signer/observer comparison.
