# Offline legacy snapshot accounting

`relaunch-accounting` produces a deterministic **custody inventory**, not a
claimant ledger or migration approval. It is a standard-library-only Go CLI for
Linux/macOS. It reads one explicitly selected public snapshot-v3 file and writes
one compact JSON record to stdout. It does not capture a snapshot.

See the [accounting specification](../../docs/specs/legacy-snapshot-accounting-v0.md)
and [selected historical allocation direction](../../docs/tokenomics/LEGACY-ASSET-MIGRATION.md).
Nominal 1:1 is a direction for later approved, reconciled entitlements, **not** an
instruction to turn this inventory into payouts.

## Invocation

```text
relaunch-accounting --snapshot PATH --expected-sha256 LOWERCASE64HEX
```

Both flags are required exactly once; either order and standard `--flag=value`
syntax work. No positional arguments, stdin (`-`), output-file flag, network,
keyring, database, discovery, signing, subprocess, transaction, root-generation,
genesis or payout option exists. The executable does not write files. Supply
only a public snapshot, not keys, secrets or node homes.

The expected digest must be independently obtained for real evidence. A hash
computed from an arbitrary downloaded file is not independent provenance. A
matching pin establishes **byte equality only**. A fabricated but internally
consistent snapshot can pass every check here. URLs in source metadata remain
inert: the only URL operation is local syntax parsing.

From the repository root, this command exercises the explicitly synthetic
fixture, with its fixed raw-byte digest:

```bash
GOTOOLCHAIN=go1.25.14 GOFLAGS=-mod=readonly GOWORK=off GOMAXPROCS=2 \
  go run -p 2 ./tools/relaunch-accounting \
  --snapshot tools/relaunch-accounting/testdata/synthetic-100.snapshot-v3.json \
  --expected-sha256 77bc5b8c3f14ffc4ec224c398c585ba59118ca0d8c80d89e6e2eda2963592964
```

Exit status is `0` for reconciled custody output, `1` for input/accounting/I/O
failure, and `2` for invalid CLI arguments. All input/report checks complete
before the first stdout byte. Invalid input leaves stdout empty and reports a
diagnostic on stderr. An output-device failure can of course interrupt an
already-started write; consumers must require successful exit and complete JSON.

### File boundary

Input must be a bounded regular file. The final pathname component may not be
a symlink, directory, FIFO, socket or device. The reader uses
`O_RDONLY | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK`, checks the opened descriptor
against the initial `Lstat`, bounds the read, and checks identity, mode, size and
mtime again against both descriptor and pathname. It never scans directories.

Ancestor directories are part of the caller's trusted local-filesystem boundary;
the tool does not forbid ancestor symlinks or claim to sandbox a hostile filesystem.
The digest binds the actual bytes read, not an immutable filesystem snapshot.
As with any file read, the operating system may update access-time metadata.

## Accepted wire format and finite bounds

The sole input schema is `zerone-relaunch-snapshot-v3`, matching the local wire
structs in [`../relaunch-snapshot/main.go`](../relaunch-snapshot/main.go). The
separate capture tool is read-only reference code; this implementation neither
imports nor changes it.

| Resource | Ceiling / rule |
| --- | --- |
| Whole input file | 64 MiB (67,108,864 bytes), nonempty UTF-8 JSON |
| Owners | 100,000 rows |
| Bonded validators | 10,000 rows |
| JSON depth | 8 from root depth 0; the fixed schema is shallower |
| Decoded JSON strings/keys | 2,048 bytes; replacement characters/unpaired surrogate escapes refused |
| Address, account-type and public-key-type labels | 256 bytes, nonempty, no whitespace/control/replacement characters |
| Chain ID / module name | 128 bytes, same label rules |
| Amount text | 78 ASCII decimal digits; exact `big.Int` arithmetic, never floating point |
| Heights and transaction counts | Canonical nonnegative JSON int64 numbers; heights must satisfy the positive F/A/H contract |
| Report | 64 MiB, also conservatively bounded before expanded row serialization |

The schema and row ceilings also bound total tokens and object members: there
are no arbitrary maps or input-driven type definitions. Duplicate keys are
rejected at **every** object level, including escaped equivalents. Exact,
case-sensitive field names and required-field presence are enforced; unknown
fields, case aliases, null scalars/objects and multiple/trailing JSON values are
rejected. Only the producer's optional `module_name` and
`declared_genesis_file_sha256` may be omitted. Empty owner/validator inventories
may be either `[]` or `null`, as the capture tool can emit nil slices; report
arrays are always `[]` when empty. No partial report, sampling or truncation
substitutes for a resource-limit failure. Conservative report estimation may
refuse some inputs below the file/row limits.

Amount strings must be `0` or `[1-9][0-9]*`. Signs, leading zeros, decimals,
exponents, whitespace and non-ASCII digits fail. Snapshot-v3 owner balances and
bonded-validator tokens must be **positive**; only supply/subtotals can be zero.
The denomination must be `uzrn`. All owner balances must sum exactly to supplied
`supply_uzrn`; duplicate owner labels and contradictory/duplicate module names
fail. Bonded validators must have distinct operator labels, bonded status,
positive canonical tokens, and canonical nonempty base64 public-key encoding
(32 bytes for the known Ed25519 type). Their token sum may not exceed bank
supply and is never added to it. Address labels and public-key encodings are not
proof of address validity, control, signatures or beneficial rights.

F/A/H checks enforce:

- `F > 0`, `A = F+1`, `H = A+1`, with int64 overflow refused;
- BlockStore at `H`, ABCI last applied at `A`;
- empty `A` and `H`; canonical/results flags true at `A`, false at staged `H`;
- H's previous-block hash equals A's block hash; A and H have distinct block IDs;
- H's header AppHash equals the excluded post-anchor A-state AppHash;
- 32-byte hexadecimal Comet hashes, RFC3339 timestamps with H later than A,
  lowercase genesis digests, and the exact snapshot-v3 REST trust disclosure;
- HTTP(S) source base-URL syntax without embedded credentials, query or fragment.

The optional declared **raw genesis-file** digest is not the **semantic RPC
JSON** genesis digest. Omission is visible as a gap. Neither is authenticated.
No block ID is independently recomputed, no signature is verified, and no REST
balance is Merkle-proved. In particular, the source limitation is retained:

> trusted height-pinned REST responses; no Merkle proof binds inventory to checkpoint_app_hash

The post-anchor A-state custom-staking census is not accepted as a replacement
input or as F-state eligibility evidence. This tool neither changes ICS23
acceptance nor resolves any state-proof limitation.

## Custody, counted once

Every bank owner becomes exactly one row, sorted lexicographically by the
original address label. No normalization changes those labels.

| Exact account-type evidence | Classification |
| --- | --- |
| `/cosmos.auth.v1beta1.ModuleAccount`, with a valid module name | `module_custody` |
| `/cosmos.auth.v1beta1.BaseAccount` | `non_module_location` |
| `/cosmos.vesting.v1beta1.BaseVestingAccount` | `non_module_location` |
| `/cosmos.vesting.v1beta1.ContinuousVestingAccount` | `non_module_location` |
| `/cosmos.vesting.v1beta1.DelayedVestingAccount` | `non_module_location` |
| `/cosmos.vesting.v1beta1.PeriodicVestingAccount` | `non_module_location` |
| `/cosmos.vesting.v1beta1.PermanentLockedAccount` | `non_module_location` |
| `bank_only` and all unrecognized types | `unknown_location` |

There is no module-name, address-prefix or substring-based guess. Account types
and module names are retained as **untrusted evidence**, not proof that an
operator personally owns module custody. `bank_only` retains an auth-metadata
gap. Unrecognized types retain a classification gap. Vesting records retain the
missing schedule/delegation/spendability gap. Even a recognized BaseAccount is
not proven unrestricted or personally owned.

The invariant is:

```text
source_supply_uzrn = module_custody_uzrn
                  + non_module_location_uzrn
                  + unknown_location_uzrn
```

Module liabilities, validator tokens, staking/unbonding ledgers, LP shares and
IBC claimants are **not** extra principal on top of their backing. Validator
rows are separately sorted metadata only. Contingent or unfunded rewards are
not existing supply. Other assets and obligations are visibly outside this
native inventory, not declared worthless or forfeited.

The synthetic fixture is `100 = 60 + 35 + 5`: module custody 60, BaseAccount 30,
vesting 5, bank-only 3 and unrecognized type 2. Its validator's 60 tokens are not
added again. All addresses, chain names, URLs, hashes and public-key bytes in
`testdata/` are synthetic labels, not adopted checkpoints or real funding.

## Report contract and independent hashing

Schema: `zerone.legacy-snapshot-accounting/v0`.

- `result` is `RECONCILED_CUSTODY_ONLY`, never provenance or release acceptance.
- `input_sha256` binds exact raw bytes; `input` records schema, byte count and
  `raw_sha256_matches_expected`. No path, clock, environment or random ID appears.
- `source` preserves the snapshot's boundary assertions and REST limitation.
- `trust.provenance_authenticated`, `signatures_verified` and
  `state_proofs_verified` are always `false`.
- `custody_totals` records all three exact amounts and counts; `custody_rows`
  retains account evidence and stable `gaps` codes from `classify` in `report.go`.
- Every custody row and the whole report have `eligibility: "UNDETERMINED"`.
  Every row has `restrictions: "UNKNOWN"` and explicit ownership/restriction gaps.
- `entitlement_total` and `reserve_requirement` are `null`; their status fields
  are `UNKNOWN`. Even **zero source supply** or **zero unknown locations** does
  not establish zero entitlements or zero funding requirements.
- `economic_effect: "NONE"` and `payout_authorization: false` always hold.

The golden fixture fixes compact Go `encoding/json` serialization and declared
struct field order. Match `custom-staking-census/report.go`'s existing convention:
serialize with `report_sha256` present as `""`, SHA-256 those exact bytes, then
place the lowercase digest into that field and serialize again. The digest
**value**, not the field name, is excluded. The CLI appends one framing newline;
it is not in the self-hash preimage. This is not generic JSON canonicalization.

To recompute independently, validate the compact emitted record, remove its
single trailing newline, replace only its unique top-level `report_sha256`
value with the empty string without changing field order/encoding, then hash.
Separately hash the original snapshot bytes and compare `input_sha256` with
both those bytes and the independently supplied expected digest. Neither digest
is a signature or an entitlement/claim root.

Identical valid bytes yield identical report bytes. Reordering input fields or
owner rows changes the bound raw digest and report self-hash, while sorted
accounting remains equal. Tests recompute the self-hash directly from emitted
bytes without calling the report builder or its hashing helper.

## Targeted checks

Run from the repository root with the existing pinned toolchain; no dependency
or module-file change is needed:

```bash
GOTOOLCHAIN=go1.25.14 GOFLAGS=-mod=readonly GOWORK=off GOMAXPROCS=2 \
  go test -p 2 ./tools/relaunch-accounting -count=1
GOTOOLCHAIN=go1.25.14 GOFLAGS=-mod=readonly GOWORK=off GOMAXPROCS=2 \
  go test -race -p 2 ./tools/relaunch-accounting -count=1
GOTOOLCHAIN=go1.25.14 GOFLAGS=-mod=readonly GOWORK=off GOMAXPROCS=2 \
  go vet -p 2 ./tools/relaunch-accounting
```

Tests cover the full synthetic golden; normalization versus raw binding; zero
versus unknown; all account classes; nested duplicate/unknown/missing/null/type
errors; numeric, digest, denomination, supply and F/A/H errors; size/row/depth
ceilings; CLI exit/stdout failure behavior; regular/symlink/FIFO/device boundaries;
unchanged input bytes and no side files; and an explicit production import/OS
operation allowlist. These are scoped product tests, not an independent release,
consensus or production-provenance review.
