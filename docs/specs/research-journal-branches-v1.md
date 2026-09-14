# Continuing and comparing research histories

A reviewer can retain a published journal and continue it in a fresh local
store. Two continuations may disagree while preserving the same earlier
records. This extends the existing off-chain journal tools and reader; the
journal wire format and blockchain messages remain unchanged.

## Start from exact bytes

`journal.py fork source.json new-store --expected-sha256 HASH` reads a bounded
regular export file, checks its exact SHA-256 and validates the complete
journal before creating a new private store. Existing destinations and symlink
paths are refused. The expected hash must come from the version you intend to
retain. Copying a hash from an untrusted file beside the export does not
authenticate either file.

The new store preserves the complete header and every existing entry: record
IDs, attribution labels, acquisition times, sequences and hash links. It retains
the exact input bytes as `fork-source.json`. Its working `journal.jsonl` uses the
existing canonical row encoding. A later export uses canonical JSON, so its
whole-file hash can differ from a differently formatted input even before any
entry is added. Keep `fork-source.json` when verifying the original checkpoint.

Only new appends receive new local acquisition times. Inherited times remain
declared historical data. A source with a future final acquisition time can be
retained, but the existing append clock check will refuse a new earlier time.
Do not rewrite inherited times to make an append succeed.

The preserved collection UUID identifies shared history, not an exclusive
writer or canonical branch. Name and retain each export by its own file hash
and head. The fork operation does not grant ownership of inherited work,
authenticate its attributed authors, or establish an independent reviewer.
No automatic merge chooses which continuation becomes authoritative.

## Add a scoped response

Use the existing record kinds:

- A `review` contribution explains what was checked, assumptions, evidence and
  remaining uncertainty. A `uses-input` relation can name the exact earlier
  contribution it examined without asserting that the review supports it.
- A concern targets an earlier contribution. State the precise discrepancy and
  evidence; a concern is not a misconduct verdict.
- An assessment targets an earlier concern or relation. Its disposition belongs
  to that attributed reviewer. `addressed` does not clear a concern globally,
  and another branch can retain `disputed` or `inconclusive`.

Use fresh record IDs. Review corrections append another record; old records and
their assessments stay attached to the original. Include the exact target entry
hash and relevant artifact hashes in evidence when sharing a scoped response.
Nothing executes, fetches, publishes or signs these evidence references.

## Compare complete histories

`journal.py compare left.json right.json` accepts exports or store directories
and validates both complete histories before comparing them. The browser does
the same for two explicitly selected local exports. Comparison always uses the
complete supplied files, independently of the timeline cutoff.

The result has these fields:

```json
{
  "schema": "zerone-research-comparison/v1",
  "relationship": "diverged",
  "left": {"collection_id": "<UUID>", "entry_count": 28, "head_sha256": "<SHA256>"},
  "right": {"collection_id": "<UUID>", "entry_count": 28, "head_sha256": "<SHA256>"},
  "common_prefix": {"entry_count": 27, "head_sha256": "<SHA256>"}
}
```

| Relationship | Meaning |
| --- | --- |
| `same-history` | The headers and every row match. Whole-file bytes can still differ. |
| `left-prefix` | The complete left history is a strict prefix of the right. |
| `right-prefix` | The complete right history is a strict prefix of the left. |
| `diverged` | Headers match; each history has a different suffix after their common prefix. |
| `different-root` | Headers differ, even if their UUID labels match. `common_prefix` is null. |

An empty history's head is the canonical header hash. Matching headers with no
matching entries have a zero-entry common prefix at that hash. Row comparison
includes acquisition times and all other fields. Repeating an ID or text later
in a different branch does not restore a shared prefix after divergence.

This is a structural comparison, not a semantic comparison of scientific
judgments. Divergence can be an ordinary new branch, reordered acquisitions or
a rewritten history; the tool cannot infer intent or misconduct. A strict
extension does not establish that its new claims are correct. An export can be
internally valid yet omit another continuation that exists elsewhere.

The browser additionally displays the SHA-256 of each exact selected file and
whether their bytes match. It verifies neither external expected pins nor
transaction proofs. Imports stay in tab memory; no records are uploaded,
stored in the browser or automatically fetched from evidence links. A failed
comparison clears the previous comparison and leaves a valid primary journal
available for reading.

## Checkpoints remain separate

First verify the original snapshot using its independently retained expected
commitment and the [saved chain proof](research-checkpoint-v1.md). Forking and
comparison alone do not perform that cryptographic verification.

A participant can later prepare a checkpoint of their own extended export with
the existing `checkpoint.py prepare` command. A signed bank checkpoint
authenticates that account's chosen snapshot under the stated chain trust
model. It neither signs each inherited author label nor opens a native claim
review round. The existing published checkpoint is a bank transaction and
supplies no native Claim ID or Fact ID for commit/reveal or contradiction.
Native scientific participation still needs those actual targets and its own
admission process. This workflow creates no reward or automatic endorsement.

See the [worked fork and comparison commands](../../tools/research-review/README.md#continue-a-published-history)
and the downloadable fictional continuations in the
[reader](https://zerone.ai/research/review/). Fictional branches illustrate
disagreement; they are not scientific reviews or records on the chain.
