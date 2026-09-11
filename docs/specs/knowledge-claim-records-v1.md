# Knowledge claim records v1

**Source protocol:** `knowledge-claim-records-v1`, knowledge consensus 9→10.
The sole predecessor is the completed review-neutrality source, including
canonical relation export/import, frozen at
`2fab850a0ae6f2193fb6e8986b303ee77d9a7d91`. Publishing this source or its website
does not upgrade the deployed legacy `zerone-1` or its observer package.

## Preserve the submitted contradiction

After activation, `MsgSubmitContradiction.reason` is retained verbatim in
`Claim.argument_text`, and `evidence_ids` in the existing ordered
`Claim.evidence_ids`. The counterclaim text remains `Claim.fact_content`; its
explicit `CONTRADICTS` input relation names the target Fact. The existing
response field `counter_fact_id` still returns the counterclaim's **Claim ID**.
Clients should query that Claim ID rather than assume a Fact already exists.

The optional reason accepts at most 4,096 UTF-8 bytes. At most 16 evidence IDs
are admitted, each nonempty valid UTF-8 with at most 256 bytes; duplicate IDs
are refused. Input order and whitespace are preserved. Unknown protobuf fields
are refused at this activation boundary. These bounds constrain the stored
record; references do not attest that external evidence is available or valid.

All admitted contradiction writes and collateral movement commit together. A
failed target status write rolls back the new Claim, round and bank movement.
An existing deterministic Claim ID is refused before collateral movement, so
an identical submission by the same account in the same block cannot overwrite
the retained reason, evidence or target and lock stake again. This keeps the
existing identifier algorithm and scientific adjudication rules.

Older contradictions are unchanged. An empty historical `argument_text` or
`evidence_ids` cannot establish that the submitter supplied no reason or
evidence: the predecessor dropped those fields. A transaction archive may
retain the original signed input separately; this release does not reconstruct
or import it as though it had always been canonical Claim state.

## Read the retained claim history

The read-only `zerone.knowledge.v1.Query/ClaimHistory` method and REST route
`GET /zerone/knowledge/v1/claims/{id}/history` return:

- The requested Claim when retained, all primary VerificationRound records
  carrying its ID, and all retained Facts derived from it.
- Complete reviewer commitments, reveals, attestations, policy versions and
  recorded payment plans within those rounds.
- Full current incoming/outgoing canonical relation metadata and recorded
  status transitions for each included Fact.
- One hop of related Claims whose stored `provisional_fact_id` targets a root
  Fact, whose `challenged_claim_id` equals the root Claim ID, or whose
  `CONTRADICTS` input relation targets a root Fact. Each link names the literal
  field that matched. Their retained rounds and derived Facts are included too.

Primary Claim, Round and Fact namespaces are scanned once under a shared
budget. Active and selected-round indexes are not treated as an inventory of
past rounds. A retained `Claim.verification_round_id` whose row is missing is
reported in `missing_round_ids`. Surviving rounds or Facts can be returned with
an absent Claim. A reference that resolves to a different Claim's round is an
inconsistency and fails the query. No synthetic history fills missing rows.

Neighbor relation IDs do not recursively expand into more Fact bodies,
descendants or challenges. Canonical relations show their current stored
metadata, not a complete sequence of edge edits. Historical missing rounds or
transitions with no surviving reference cannot be counted or recovered.

The query uses the existing ToK ceilings: 65,536 total read entries, 16 MiB of
read bytes, and 8 MiB of encoded response. A limit, malformed record or storage
error refuses the entire query; no truncated result is presented as complete.
This deliberately uses existing storage without a new permanent index. Its
full-scan ceiling is a practical limit as the ledger grows; a future inventory
index should be justified by measured usage and preserved historical scope.

The response records the actual SDK `chain_id` and `block_height`. A nonzero
request `at_block_height` must match that context. Selecting a historical
height is the transport's responsibility; pruned or unavailable state cannot
be relabelled as historical evidence.

```sh
# Use a node running this compatible source; standard SDK query flags apply.
zeroned query knowledge claim-history CLAIM_ID --node http://127.0.0.1:26657 -o json
zeroned query knowledge claim-history CLAIM_ID --node http://127.0.0.1:26657 --height 1234 -o json
```

The TypeScript SDK exports `queryClaimHistory` using an explicitly supplied
protobuf RPC transport. It has no default endpoint or signing side effect.
The website proxy does not expose this method; use a compatible local or
operator-provided node. See the SDK README for its consumer contract.

These are endpoint observations of retained state. The response does not carry
signed TxRaw envelopes, transaction inclusion proofs, light-client headers or
a proof of the entire scan. To independently authenticate a statement, retain
the signed transaction and verify its inclusion against an independently
verified chain commitment. A payment plan's existence is distinct from its
paid marker and actual bank settlement. Neither block-signing weight nor a
panel verdict proves scientific expertise, independence or truth.

## Exact execution boundary

The sole named upgrade admits the complete predecessor module map at committed
H−1, with a matching on-chain/local positive-height plan, empty `info` and no
deprecated fields. Earlier handlers retain their frozen targets. Mixed maps,
unsafe skipping, premature execution and inconsistent marker/done metadata
are refused. The migration selects the prospective behavior atomically; no
Claim, round, relation, history or financial obligation is rewritten.

Native current genesis explicitly selects `claim_records_enabled=true` with
record integrity and review neutrality. Export/import preserves this flag and
the existing Claim fields without inventing an applied-upgrade receipt.
Restart checks execution/version/receipt coherence. Query availability belongs
to the source binary; the new write behavior additionally requires activation.

The focused tests and signed two-binary rehearsal exercise preserved old
records, exact new reason/evidence retention, direct history joins, actual
query heights, atomic failure and restart/export preservation. Local accounts
and balances are synthetic fixtures, not independent scientific participants.
