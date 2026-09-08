# Legacy snapshot accounting v0

**Scope: deterministic offline native-custody inventory, not entitlement or
claims implementation.** The [allocation direction](../tokenomics/LEGACY-ASSET-MIGRATION.md)
is opt-in historical nominal 1:1, but this report approves no owner or payment.
The CLI lives in [`tools/relaunch-accounting`](../../tools/relaunch-accounting/README.md).
It must not change the source snapshot producer, genesis, or runtime latches.

## Local interface and source schema

```text
relaunch-accounting --snapshot PATH --expected-sha256 LOWERCASE_HEX
```

Both arguments are required. `PATH` explicitly selects one local snapshot file;
`LOWERCASE_HEX` is an independently supplied, exact 64-character lowercase
SHA-256 of its **raw bytes**. Success emits one deterministic JSON report on
stdout. Invalid input returns nonzero with a diagnostic on stderr, not a partial
report. There is no network, endpoint discovery, keyring, signing, subprocess,
state-database, transaction, genesis-generation, or payout interface. Endpoint
strings inside the snapshot are inert source metadata, never fetch instructions.

The sole input schema is `zerone-relaunch-snapshot-v3`, as defined by `snapshot`,
`sourceCheckpoint`, `owner`, `validator`, and `consensusPubKey` in
[`tools/relaunch-snapshot/main.go`](../../tools/relaunch-snapshot/main.go).
There is no alternate simplified balance-list, export, or census input format.

| Existing snapshot field | Meaning and handling |
| --- | --- |
| `schema` | Exact v3 identifier; v1/v2 and unrelated reports are rejected. |
| `source` | Complete existing F/A/H boundary metadata, genesis digests, and trusted-REST disclosure; retained as source assertions, not authenticated by the accounting tool. |
| `denom`, `supply_uzrn` | Native `uzrn` and exact canonical decimal supply. |
| `owners` | Positive balances with `address`, `account_type`, optional `module_name`, and `amount_uzrn`. Each address appears once. |
| `bonded_validators` | Existing `operator_address`, `consensus_pubkey` (`@type`, `key`), `jailed`, `status`, and `tokens` evidence. Token amounts are not added to bank supply. |

The producer's optional `declared_genesis_file_sha256` is distinct from
`rpc_genesis_canonical_sha256`: raw-file and semantic JSON digests are not
interchangeable. The former, when present, is still only a declared value here.
Snapshot account metadata preserves a type label and optional module name, not
vesting schedules, delegation/unbonding ledgers, LP ownership, or IBC traces.

## Bounded, unambiguous validation

Reject malformed or trailing JSON, duplicate object keys at every level,
unsupported or misspelled fields, missing required fields, wrong field types,
and duplicate owner addresses. Validate bonded-validator records without
treating them as a second supply ledger. Empty collections must be handled as
the existing v3 producer serializes them; they cannot conceal a supply mismatch.

Amounts are bounded base-10 **strings** using `0` or `[1-9][0-9]*`, never floating
point, signs, exponents, whitespace, or leading zeroes. Owner balances and
bonded-validator tokens are strictly positive; total supply can be zero only
when it reconciles with an empty owner inventory. Use exact integer arithmetic.
The input denomination must be `uzrn`, and owner balances must sum exactly to
`supply_uzrn`.

The implementation must bound file bytes, JSON nesting/tokens, string sizes,
owner/validator counts, amount length, and report bytes before accepting an
input or emitting a report. The accounting tool's checked-in constants, README,
and limit tests define the precise finite numeric ceilings; this specification
does not borrow the snapshot producer's per-HTTP-response ceiling as a whole-file
limit or invent a second set of bounds. A structurally valid but oversized
snapshot is refused, not truncated, sampled, or reported as complete. Final
integration must check source/README/spec and fixtures for parity.

Validate the existing boundary relationships as assertions about the supplied
snapshot:

- `F = checkpoint_state_height > 0`, `A = final_committed_block_height = F+1`,
  `H = halt_trigger_height = A+1`, with overflow refused;
- `rpc_blockstore_height = H` and `abci_last_applied_height = A`;
- `final_committed_block_txs = staged_halt_trigger_block_txs = 0`;
- anchor canonical/results flags are true; staged-H canonical/results flags
  are false;
- `staged_halt_trigger_previous_block_hash = final_committed_block_hash`, and
  `staged_halt_trigger_header_app_hash = excluded_post_anchor_app_hash`;
- hashes, times, genesis metadata, and `rest_trust_model` meet the v3 contract.

`checkpoint_app_hash` is the anchor A header's commitment to state F; the
excluded post-anchor hash is distinct evidence, not an alternative inventory
root. These consistency checks do not reverify block signatures, independently
recompute block IDs, authenticate a selected checkpoint, or Merkle-prove any
bank balance. The snapshot's disclosure remains explicit:

> trusted height-pinned REST responses; no Merkle proof binds inventory to checkpoint_app_hash

A matching independently supplied raw digest identifies the expected bytes; it
cannot turn trusted REST inventory into cryptographically proven state. A
fabricated but internally consistent snapshot can still satisfy this offline
contract. Authentic source selection and the existing signed transition package
are separate responsibilities. The post-anchor A-state custom-staking census
is neither this input format nor F-state eligibility evidence.

## Custody partition, not claimant allocation

Create one row per owner, sorted by address, preserving `account_type`, optional
`module_name`, and `amount_uzrn`. Classify each exactly once:

| Partition | Evidence boundary |
| --- | --- |
| `module_custody` | Exact `/cosmos.auth.v1beta1.ModuleAccount` type with consistent module metadata; beneficial owners and liabilities unresolved. |
| `non_module_location` | Explicitly recognized non-module account types, including recognized vesting types; not a claim that the balance is unrestricted, individually owned, or eligible. |
| `unknown_location` | `bank_only` or an unrecognized account type; retain the original evidence and a visible classification/ownership gap instead of dropping the balance. |

The exact recognized-type allowlist and stable restriction/gap labels belong
with the accounting implementation and tests. Do not classify by address prefix,
module-name guess, operator identity, or an assumption that every non-module
type is understood. Reject contradictory module metadata. Known vesting types
must retain a visible restriction gap: v3 does not contain their schedules or
spendability. `bank_only` means the producer found no corresponding auth account,
not that the coins have no owner.

```text
source supply = module_custody + non_module_location + unknown_location
```

These are exact native custody subtotals, each counted once. Module liabilities,
validator tokens, delegated stakes, unbonding records, LP shares, or remote IBC
vouchers cannot be added on top of their bank backing. This report accepts none
of those claimant ledgers and emits no claim root. It does not decide whether
non-native assets or unresolved liabilities are eligible, worthless, forfeited,
or administrator-owned.

## Report and deterministic binding

| Required report content | Meaning |
| --- | --- |
| `schema: "zerone.legacy-snapshot-accounting/v0"` | This offline report, not snapshot-v3 or a claims schema. |
| `input_sha256` | SHA-256 of the exact supplied snapshot bytes, including whitespace/order. |
| Source boundary and trust limitation | Preserve the supplied F/A/H identity and its non-Merkle-proved provenance; do not claim source authentication. |
| Sorted custody rows, subtotals, and gaps | Exactly reconciled bank locations, with original account-type evidence and unresolved interests. |
| `eligibility: "UNDETERMINED"` on every row | Even a recognized BaseAccount or a perfectly reconciled module has no eligibility finding. |
| `entitlement_total: null`, `reserve_requirement: null` | Unknown, not zero, not inferred from source supply or reconciled custody. |
| `economic_effect: "NONE"`, `payout_authorization: false` | No signing, reservation, minting, transfer, activation, or payment permission. |
| `report_sha256` | Deterministic self-hash, not a signature, entitlement root, or activation authority. |

Follow the compact `encoding/json` self-hash pattern in
[`tools/custom-staking-census/report.go`](../../tools/custom-staking-census/report.go):
serialize the ordered report with `report_sha256` present as the empty string,
SHA-256 those bytes, then put the lowercase digest in that field and serialize
again. The stdout record may end with one newline; that framing newline is not
part of the self-hash preimage. The complete field order and serialization are
fixed by the accounting report type and golden tests, not arbitrary JSON
reserialization or a claim of generic JSON canonicalization.

No local path, clock, random identifier, or host state contributes to the
report. Identical valid input bytes produce identical output bytes. Reordering
otherwise equivalent raw input changes `input_sha256` and therefore the bound
report/self-hash, while sorted custody rows and totals remain equivalent. A
consumer must independently recompute both bindings; neither proves source
truth, entitlement correctness, reserve funding, or approval.

## Acceptance boundary

Synthetic fixtures must demonstrate `100 = 60 + 35 + 5` with **no** approved
payout and null entitlement/reserve totals. Include vesting, bank-only, and
unknown types without erasing their limitations. Negative tests must cover
digest/schema/denom mismatch, duplicate keys/owners, malformed or ambiguous
JSON, noncanonical amounts, inconsistent F/A/H, supply mismatch, and resource
limits. Repeated runs and independent self-hash recomputation must check byte
determinism. Fixtures are explicitly synthetic, not an adopted checkpoint,
authenticated production evidence, funding proof, or a release-gate exception.
