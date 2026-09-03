# Legacy custom-staking census

`custom-staking-census` is a strictly offline, read-only evidence tool for
Zerone's legacy `zerone_staking` ledger. It opens the application database from
a disposable copy of a halted node, binds the relevant stores to an externally
recorded application height and AppHash, and emits a deterministic accounting
report.

The tool exists to answer a narrow question: is the legacy ledger structurally
complete and exactly backed at this committed state? It does not repair state
or perform a migration. "Read-only" refers to the copied node source; the only
supported filesystem write is an optional report published outside that home
with `--output`.

Cosmos SDK `x/staking` remains Zerone's sole consensus-staking and validator
authority. A legacy custom validator is application history, not a CometBFT
validator. In particular, the custom record's `consensus_pubkey` string is
**untrusted**: it was not used to create or authenticate the CometBFT validator
and is never used by this tool to establish consensus identity.

## Safety boundary

Never run this command against a live validator home, the only copy of a node
database, or a database copied while the daemon was running.

Run in an isolated analysis environment where no other process can rename or
modify the copied home during inspection. The command rechecks the opened
database-directory identity and committed root bytes, but path checks are not a
sandbox against a hostile local process.

The required workflow is:

1. inhibit automatic restart and halt the node cleanly;
2. record the stopped height and post-commit AppHash from independent evidence;
3. make a filesystem-level copy of the complete node home while it remains
   halted;
4. place that copy in an offline, disposable analysis environment; and
5. run the census against the copy, not the source home.

`--copied-db` is an explicit operator attestation that this boundary was
followed. It is not proof that a path is safe. The command additionally refuses
Zerone's default `~/.zeroned` home and its descendants, aliases into that home,
the filesystem root, relative homes, and symlinks at the supplied home, `data`,
or `data/application.db` boundary. A custom live home cannot be recognized
mechanically.

Supported database backends are opened in backend-enforced read-only mode:

- `goleveldb` uses GoLevelDB's `ReadOnly` option; and
- `pebbledb` uses Pebble's `ReadOnly` option.

The IAVL adapter rejects every write and batch-write method. The tool performs
no RPC, REST, gRPC, P2P, keyring, transaction, or consensus operation. Backend
read-only mode does not make a live database safe to inspect; the stable halted
copy remains mandatory.

The database and IAVL libraries must materialize an encoded record before the
tool can apply its post-read byte ceiling. Wire and JSON preflights prevent a
bounded record from amplifying into unbounded generated slices, but they cannot
turn the database library itself into a streaming or memory-limited reader. If
the copied files may be corrupt, run the disposable analysis environment with
an operating-system memory limit as an additional containment boundary.

## Required evidence and usage

Build the reviewed revision, then invoke the resulting binary directly so its
`0`, `1`, and `2` exit statuses remain distinguishable. The seven
evidence/safety flags are required; `--output` is the recommended publication
mode:

```sh
go build -trimpath -o /absolute/path/custom-staking-census ./tools/custom-staking-census

status=0
/absolute/path/custom-staking-census \
  --home /absolute/path/to/disposable-halted-node-copy \
  --backend goleveldb \
  --chain-id zerone-1 \
  --expected-height 123456 \
  --expected-app-hash 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --source-commit 0123456789abcdef0123456789abcdef01234567 \
  --copied-db \
  --output /absolute/path/to/new-custom-staking-census.json || status=$?

case "$status" in
  0) echo "reconciliation PASS" ;;
  2) echo "reconciliation FAIL; retain the evidence report" ;;
  *) echo "operational failure; no complete report is usable" >&2 ;;
esac
```

- `--home` is an absolute node-home path containing
  `data/application.db`.
- `--backend` is exactly `goleveldb` or `pebbledb` and must match the copied
  database.
- `--chain-id` is the independently recorded chain ID. It is 1-128 ASCII
  letters, digits, `.`, `_`, or `-`, beginning with a letter or digit.
- `--expected-height` is a canonical positive `int64` decimal application
  height, with no sign or leading zeroes.
- `--expected-app-hash` is the trusted 32-byte post-commit application root for
  that height, rendered as exactly 64 lowercase hexadecimal characters.
- `--source-commit` is the full 40-character lowercase hexadecimal revision of
  the reviewed Zerone source whose schema the census uses and from which the
  census binary should be built.
- `--copied-db` asserts that the node was halted before this disposable copy
  was made.
- `--output`, when supplied, must be an absolute path that does not exist. The
  tool writes and syncs a private temporary file, then publishes the completed
  report atomically without overwriting earlier evidence. It must resolve
  outside the copied source home. Omitting it writes to stdout for supervised
  pipelines.

The chain ID and source commit are external provenance. `application.db` does
not prove either one, and a Git commit string does not prove that a particular
binary was built from it. Retain the independently captured chain evidence,
binary digest, copy manifest, and command transcript with the report.

## Height and AppHash semantics

Cosmos/Comet height semantics are easy to shift by one:

- root record `s/<H>` is application state **after commit `H`**;
- an ABCI `/abci_info` response with `last_block_height = H` supplies the
  corresponding `last_block_app_hash`;
- a light-client-verified block header at `H+1` also commits application state
  `H`; and
- the `app_hash` in a block header at `H` commits state `H-1`, so it is not the
  value for `--expected-app-hash H`.

Comet JSON commonly renders AppHash bytes as base64. Decode the bytes first,
then render the 32-byte value as lowercase hexadecimal. Do not derive the
"trusted" height and hash only from the same unreviewed database copy that the
tool will inspect.

Before interpreting state, the census requires `s/latest` to equal
`--expected-height`, decodes `s/<H>` as the matching multistore `CommitInfo`,
and recomputes its root against `--expected-app-hash`. The report includes all
mounted store name/root commitments in canonical name order, allowing an
independent reader to recompute that AppHash, as well as exact scan evidence
for each relevant store:

| Root-bound store | Census use |
| --- | --- |
| `bank` | The `zerone_staking` module-account balance `B` |
| `staking` | Canonical SDK validator operator identity |
| `zerone_staking` | Legacy validators, claims, unbondings, and indexes |

These are three selected stores in the complete application multistore, not a
claim that the application has only three stores. The additional multistore
rows establish inclusion but are not themselves traversed by this tool.

## Complete regular-IAVL scan

For each of the three stores, the tool opens the regular IAVL tree below its
exact physical `s/k:<store>/` prefix at height `H` and requires its version and
hash to equal the root `CommitInfo`. It does not consult, create, upgrade, or
rebuild fast-node storage.

The scan covers the entire logical tree, not just expected prefixes. For every
nonempty tree it:

1. streams leaves in strict, unique byte order;
2. requires the emitted leaf count to equal the root-bound IAVL tree size;
3. creates and directly verifies an ICS23 IAVL membership proof for every
   key/value pair against that store's committed root (for IAVL's valid
   zero-byte values, which the pinned ICS23 helper rejects before hashing, the
   tool first enforces the exact IAVL proof spec and then performs the same
   SHA-256 leaf/path calculation without that non-empty-value policy); and
4. fails on iterator, proof, decode, close, resource-limit, or concurrent-root
   stability errors.

The root latest-version and commit records must remain byte-identical across
the scan. Canonical empty-tree commitments are handled explicitly. A partial
iteration is never treated as absence, and no report is emitted after a root
or proof failure.

Resource ceilings are part of the fail-closed contract:

- root latest-version and `CommitInfo` records are limited to 16 bytes and
  16 MiB respectively, with at most 4,096 mounted stores; store names and
  commitment hashes are limited to 128 and 32 bytes;
- at most 5,000,000 leaves may be traversed in any selected store, logical keys
  and values are limited to 64 KiB and 4 MiB, and all selected stores together
  are limited to 1 GiB of key/value input;
- the retained custom store is limited to 50,000 leaves and 32 MiB of input;
- before typed custom-state decoding, streaming JSON preflight permits at most
  65,536 tokens, 4,096 elements in any array, nesting depth 32, 64 KiB per
  decoded string, and 128 bytes per number;
- an SDK validator permits at most 64 KiB in any singular length-delimited
  field and 65,536 unbonding IDs;
- retained SDK-validator rows and distinct findings are each capped at 25,000,
  module-account denominations at 10,000, and diagnostic strings at 512 bytes;
  long keys or diagnostics use a bounded prefix, byte count, and SHA-256
  digest; and
- before report JSON serialization, a conservative expansion estimate for the
  complete envelope must fit the exact 64 MiB report ceiling.

Crossing a resource ceiling is an operational failure, never a partial PASS.

## Nine legacy custom keyspaces and one app sentinel

Every recognized key in `zerone_staking` is classified into exactly one of the
nine persisted module keyspaces. The app-wide exact `_iavl_init = 0x01`
infrastructure sentinel is classified separately when present; its absence is
valid for a pre-sentinel lineage, but a malformed value or lookalike key fails.

| Prefix | State |
| --- | --- |
| `0x01` | validator records |
| `0x02` | primary delegation claims |
| `0x03` | unbonding entries |
| `0x04` | tier configurations |
| `0x05` | singleton module parameters |
| `0x06` | DID-to-validator secondary index |
| `0x07` | singleton unbonding sequence |
| `0x08` | redelegation cooldown heights |
| `0x09` | validator-to-delegator reverse index |
| `0x5f6961766c5f696e6974` | exact app `_iavl_init` infrastructure sentinel |

Keys outside those prefixes are still bound by the complete store-evidence
digest and reported as failures. Malformed keys, malformed JSON/scalar/index
values, duplicate canonical identities, impossible amounts or statuses, and
key/value identity mismatches are also failures. The full database scan is
important because the current custom staking genesis export omits `0x08`
redelegation-cooldown state; an export-only census could silently miss it.

Secondary state is checked against primary state. Every DID index must resolve
to its exact validator record, and every reverse delegation-index entry must
correspond one-for-one with a primary delegation claim. Missing, stale,
duplicate, and orphan index entries fail. Tier, parameter, cooldown, and
sequence records are classified and committed as evidence; none is promoted
to consensus authority.

## Liability and aggregate reconciliation

For `uzrn`, the census calculates:

```text
B = bank balance of the deterministic zerone_staking module account
D = sum(every primary delegation KV claim, including operator self-delegation)
U = sum(every unbonding entry whose status is exactly pending)

B = D + U
```

A completed unbonding is retained as history but is no longer included in
`U`. A still-pending entry remains a liability even if its completion height
has passed. Unknown statuses fail rather than being guessed. Amount arithmetic
is exact and overflow-safe, and unexplained balances in another denomination
are reported as failures.

The committed height binds the database snapshot; it is not assumed to share
an uninterrupted height era with every stored historical field because state
may have crossed an export/re-genesis boundary. The census checks each
unbonding's internal creation/completion ordering and global sequence identity,
but does not certify temporal provenance for legacy creation or cooldown
heights.

The census also reconstructs, for every custom validator:

- self-delegation from the primary claim whose delegator and validator address
  bytes are equal;
- third-party delegated stake from all other primary claims targeting that
  validator; and
- total stake as the sum of those two values.

Each computed value must equal the validator's stored `self_delegation`,
`delegated_stake`, and `total_stake`. Aggregate fields never add a liability,
replace an individual claimant record, or repair a discrepancy. Delegations
and pending unbondings must carry valid, canonical claimant and validator
addresses; each must target a present custom validator record.

SDK linkage is checked by decoded address bytes, not Bech32 text. A custom
account address and an SDK `valoper` address may use different human-readable
prefixes while carrying the same canonical bytes. The SDK staking record and
its key must agree before the report classifies an exact match. An absent match
stays absent, and an ambiguous or malformed match fails; the tool never invents
a validator link. SDK operator identity, bonded status, jailed state, and token
amount come only from the SDK `staking` store. Consensus-public-key validation
is outside this census and must use standard SDK/CometBFT tooling. The legacy
custom `consensus_pubkey` is recorded only as untrusted legacy evidence and
never participates in the match.

## Deterministic report and exit behavior

After every root, proof, structural, and arithmetic check completes, the
selected destination receives one compact JSON report followed by one newline
(`--output` leaves stdout empty). The schema fixes field order;
collections and findings are sorted deterministically; monetary amounts and
evidence versions/counts use canonical decimal strings; typed census counts and
heights use unsigned JSON numbers; and hashes and raw bytes use lowercase
hexadecimal.
Machine-local paths, timestamps, and iteration order do not affect the report.

The report contains its own `report_sha256`, calculated over the same compact
JSON structure with `report_sha256` set to the empty string. Independent runs
over byte-identical committed state with the same evidence flags must therefore
produce byte-identical report bytes and hash.

Exit codes distinguish a proven reconciliation result from an operational
failure:

- `0` — `PASS`: all root, proof, classification, index, identity, aggregate,
  and `B = D + U` checks passed;
- `2` — `FAIL`: the scan completed and emitted a deterministic evidence report,
  but the legacy state is not automatically migratable; and
- `1` — operational or usage failure: invalid arguments, unsafe path, database
  error, incomplete scan, failed proof, report serialization/output failure, or
  close error. In atomic `--output` mode no partial report is published, though
  a complete file can exist if a post-publication directory-sync or cleanup
  step failed; quarantine it because durability was not confirmed. In stdout
  mode, if the destination fails after accepting a prefix, discard that partial
  JSON.

Do not replace the exit-code check with a search for the word `PASS`, and do
not discard a `FAIL` report: the discrepancy evidence is the main output.

## What this does not authorize

A `PASS` means only that this tool's evidence and reconciliation contract
passed for the supplied committed state. It does **not**:

- prove that the externally supplied chain ID, height, AppHash, source commit,
  binary, or copy ceremony was honest;
- authorize H4, an upgrade, a deployment, a chain launch, or a signer restart;
- move, mint, burn, haircut, delegate, unbond, refund, or assign any coins;
- infer a missing claimant, treat validator aggregates as claimant records, or
  allocate a surplus;
- convert a custom delegation into an SDK delegation;
- register, activate, jail, unjail, retire, or replace a validator;
- create or link a verifier profile, grant qualification, enable vote
  extensions, select a panel, or grant governance or truth power; or
- establish decentralization, independent control, or live-network safety.

Any discrepancy requires a separately reviewed and published resolution
manifest or successor-genesis decision. Any migration requires its own
deterministic implementation, rehearsal, authority, and release gates. Legacy
state remains read-only evidence, while SDK `x/staking` remains the sole source
of consensus-validator and stake authority.
