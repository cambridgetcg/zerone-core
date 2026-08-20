# Agent earnings v2: bound compute sponsorship

This slice gives an agent a prefunded, agent-addressed ZRN earnings loop. A
sponsor locks existing `uzrn` in `x/sponsorship`; an accepted computational
Fact binds the work and result into the Tree of Knowledge; after its challenge
window, the Fact submitter signs settlement and receives escrow. Nothing in
this path mints. Key custody remains a separate host responsibility.

## Wallet account and settlement authority

`Fact.submitter` is the worker's settlement account and payout wallet, not a
claim of personhood or identity. The chain never accepts a caller-selected
payee. Every v2 WorkContract preassigns one `worker_address`.
`MsgFulfillBounty.caller`, the stored Fact submitter, the receipt payee, and
that assignment must all be the same address. An alternate wallet can copy
public roots into a new Claim, but cannot settle the original worker's order;
its own worker-bound nullifier is a different economic unit.

Preassignment is authorization, not physical proof of authorship. A sponsor
can still create a separate contract assigning a plagiarist and pay that
contract from separate escrow. Sponsors must authenticate the intended worker
and use execution attestations where provenance matters. Anonymous/open
bounties require a future pre-reveal assignment, commit/reveal, or equivalent
attestation protocol; v2 deliberately implements assigned work only.

A v2 sponsor cannot cancel a bound ACTIVE order. Any unpaid escrow becomes
refundable to the sponsor only after the order is EXPIRED. Legacy v1 orders
have no work contract, remain refund-only, and retain active cancellation so
old funds are recoverable.

The v1→v2 migration and direct legacy-genesis import canonicalize equivalent
v1 sponsor Bech32 aliases and decimal spellings in nil-WorkContract prices,
escrow balances, and payout records (for example an all-uppercase address,
`001`, and `+1`). Runtime v1 exports already used canonical escrow strings,
but the old genesis validator admitted authored equivalents. Legacy cancellation
compares decoded account bytes, so normalization cannot strand a refund.
Sponsor canonicalization therefore ships in the initial v1→v2 activation
binary. If an environment has already recorded sponsorship module version 2,
it must use a new v2→v3 order/index repair migration rather than rerunning
`Migrate1to2`.
Historical uint64-wrapped legacy deadlines and their fulfillment timestamps
are preserved for round-trip compatibility; none of these relaxations apply
to bound v2 records. A legacy max-active parameter above 256 is clamped only
after migration audits that every sponsor's actual ACTIVE set fits the hard
cap; otherwise the upgrade fails explicitly.

## Canonical digest representation

Every commitment is a bare lowercase 64-character SHA-256 hex string. Values
such as `sha256:<hex>`, uppercase hex, partial digests, and noncanonical decimal
amounts are rejected.

The sponsor commits:

- `work_spec_hash`: exact task and receipt-schema semantics;
- `acceptance_hash`: evaluator and acceptance policy;
- `input_root`: canonical input manifest;
- `environment_root`: execution environment manifest;
- `min_corroborations`: survived formal challenges required after the normal
  verifier quorum and challenge window. Zero is the unchallenged baseline;
- `worker_address`: preassigned Fact submitter, settlement signer, receipt
  payee, and payout wallet, encoded as canonical lowercase Bech32. Alternate
  textual encodings of the same account are rejected before escrow is accepted.

The computational Claim and resulting Fact repeat those four values and add
`artifact_root`, `evidence_root`, and `work_receipt_hash`. Raw inputs,
artifacts, private evidence, and reasoning remain off chain; only commitments
and public audit metadata enter events.

## Byte-exact receipt

`work_receipt_hash` is SHA-256 over the following bytes:

1. the raw ASCII domain separator `ZRN.work.receipt.v1` followed by one NUL;
2. in order: `work_spec_hash`, `acceptance_hash`, `input_root`,
   `environment_root`, `artifact_root`, `evidence_root`, and the payee address;
3. each value encoded as an unsigned 64-bit big-endian byte length followed by
   its UTF-8 bytes.

The work-spec schema must define how the off-chain artifact and evidence
manifests are canonicalized. Consensus verifies the stored receipt hash
against the seven fields above; it does not fetch or interpret private/raw
evidence.

The sponsorship-global settlement nullifier is SHA-256 over raw ASCII
`ZRN.sponsorship.settlement.v2` plus one NUL, followed by length-prefixed
`work_spec_hash`, `acceptance_hash`, `input_root`, `environment_root`, and
`artifact_root`, then `worker_address` in that order. It excludes bounty ID,
Fact ID, caller, height, evidence, and receipt, so alternate evidence/receipt
wrappers around the same artifact remain single-use for that immutable
contract and assigned worker, while a genuinely different input, evaluation
contract, or worker assignment remains a distinct economic unit. The raw
address bytes are safe consensus input because admission requires the unique
canonical lowercase Bech32 representation.
Permanent, independent Fact and receipt indexes also prevent cross-bounty
replay. “Global” here means `x/sponsorship`; other
payment modules need a shared registry before claiming cross-module safety.

### Cross-language vector

With work, acceptance, input, environment, artifact, and evidence values equal
to 64 repetitions of `1`, `2`, `3`, `4`, `5`, and `6`, and assigned worker/
receipt payee `zrn1v3jkzervd9hx2ttfdejx27pdwdcx7m3dwexsmf`:

```text
work_receipt_hash = 341ed4d5e3e2fc399d75def7ced694b6889d2108aa4691a1594c52f9bf85724c
settlement_nullifier = 008c1206e6517a690f46d17beb87fd120152fc87b842dfeb880b7f0f9d47a1ae
```

Equivalent TypeScript/Node encoding:

```ts
import { createHash } from "node:crypto";

const part = (s: string) => {
  const body = Buffer.from(s, "utf8");
  const size = Buffer.alloc(8);
  size.writeBigUInt64BE(BigInt(body.length));
  return Buffer.concat([size, body]);
};
const digest = (domain: string, values: string[]) =>
  createHash("sha256")
    .update(Buffer.concat([Buffer.from(domain + "\0"), ...values.map(part)]))
    .digest("hex");
```

## Knowledge-root binding and maturity

Legacy claim IDs are unchanged. Computational content hashes use raw domain
separator `ZRN.claim.computational.v1\0`, then length-prefixed domain, content,
and all seven commitment fields. Claim ID commits that content hash; Fact ID
commits the Claim ID; existing Tree-of-Knowledge snapshots commit Fact IDs.
The tree root therefore transitively binds all computational roots.

Settlement is an AND gate: status is VERIFIED or ACTIVE, challenge-window end
is nonzero and elapsed, the status is not a challenged/degraded state, and
`corroboration_count >= min_corroborations`. A later disproof is normal truth
evolution; v2 provides bounded contract finality and no magical clawback.

Bound computational acceptance opens the knowledge challenge window but
creates no legacy submitter-vesting entitlement and cannot consume the
immediate knowledge demand-bounty path. Review fees still apply. Sponsorship
payouts are backed by module escrow, with the enforced invariant
`module uzrn balance >= aggregate open escrow liabilities`; unsolicited module
transfers are harmless surplus. The aggregate liability is persisted and
updated with each escrow transition, so outgoing payments do not scan lifetime
order history. ACTIVE orders are indexed by sponsor and deadline;
`max_active_bounties_per_sponsor` has a consensus hard cap of 256, and each
BeginBlock processes at most 256 due expiry transitions. A sponsor-side lazy
expiry pass over that bounded active set prevents a delayed global sweep from
stranding a newly available slot. Sponsor index keys always use canonical
lowercase Bech32, so equivalent textual spellings share one cap bucket.
