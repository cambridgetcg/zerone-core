# Research checkpoint v1

A checkpoint commits to an exact exported research journal without creating a
native Tree of Knowledge Fact or starting a scientific review round. The first
deployment uses the existing `zerone-dev-1` bank transaction format; no consensus
or application-state upgrade is needed.

## What is recorded

One signed `/cosmos.bank.v1beta1.MsgSend` sends `1uzrn` from the participant to
itself. Its transaction memo is ASCII, with no whitespace:

```text
zerone:research:v1:<collection_uuid>:<entry_count>:<export_sha256>:<head_sha256>
```

The UUID, count and head come from the fully validated journal export. The two
digests are lowercase hexadecimal SHA-256. The file digest covers the **exact
downloaded bytes**, including whitespace and the final newline. The head is the
last journal entry's hash under the existing journal schema. Empty journals are
refused. With the existing 1,000-entry limit this fits the current 256-byte memo
limit. The SDK CLI calls this transaction field `--note`.

The sender signs the Cosmos transaction, including this memo and the chain ID
through its direct SignDoc. This is that account's checkpoint, not signatures
from every attributed author in the journal. No reward, endorsement, scientific
admission, review window or provenance of an idea follows from the self-send.
The advertised development fee is `2000000uzrn`, with gas limit `2000000`; the
self-transfer itself has no net recipient payment. Check current network terms
before preparing a later transaction.

The companion JSON has exactly these fields:

```json
{
  "schema": "zerone-research-checkpoint/v1",
  "chain_id": "zerone-dev-1",
  "collection_id": "<journal UUID>",
  "entry_count": 81,
  "export_sha256": "<exact file SHA-256>",
  "head_sha256": "<last entry SHA-256>",
  "memo": "<the complete memo above>"
}
```

The chain ID is bound by the signed transaction; it is not duplicated inside
the memo. The JSON is preparation metadata, not an inclusion proof by itself.
Transaction hash, height, block time and proof receipts live separately, avoiding
a circular commitment to the transaction that creates the checkpoint.

## Verification and retention

`tools/research-review/checkpoint.py` validates the full journal and compares its
bytes, collection, count and head against a separately saved memo and metadata.
It detects a rewritten and rehashed journal, a valid shorter prefix, or a
different serialization when compared to that commitment. Replacing the journal
and its untrusted companion together is not detected by this comparison alone.

`tools/research-checkpoint/` checks the signed transaction and its inclusion in a
signed block using the original genesis hash pinned in the verifier as its trust
anchor. Read
that tool's README for its exact supported validator model and execution-result
scope. A proof bundle's self-declared trust anchor is not independent evidence;
obtain or retain the expected genesis hash separately. This version supports
only the original `zerone-dev-1` genesis and its sole consensus key; it refuses
other genesis files and validator sets.

On this single-operator development chain, a valid anchored validator signature
does not establish independent consensus, a trustworthy wall clock, current
canonical-chain membership after a reset, or authenticated source history. A
block's time is its consensus header time. The journal's acquisition times and
reported historical dates remain declared values, not retroactive timestamps.

The chain stores the signed transaction and digest commitment, not the exported
journal or its research files. Keep the exact journal, evidence, manifests and
proof bundle on retrievable storage and replicate them. Neither a hash nor a
transaction inclusion proof guarantees future file or RPC availability. A
sanitized derivative must identify its different hash and unavailable original;
it must never stand in as though it matches the original evidence digest.

Later corrections append new records and can receive a later checkpoint. They
do not mutate the previously committed snapshot. Append the checkpoint receipt
to the continuing journal **after** exporting the anchored prefix, so the journal
does not attempt to hash its own future transaction. Logical support and concern
links remain journal relations, separate from this storage mechanism.

## Serving preserved evidence

The memo directly binds the journal export. It does not automatically bind every
file placed next to that export, or a subsequently assembled archive manifest.
For external evidence, compare the obtained bytes with the digest of the
specific referenced artifact. A packet manifest referenced by a journal record
can extend that chain of checks to its listed files. Other publication manifests
need a separately retained expected hash. An inventory is useful for retrieval,
but must not silently expand the scope of the on-chain commitment.

Preserve the exact frozen files when publishing reading copies. Reformatting an
HTML paper to satisfy a hosting policy changes its file hash, even if the text
looks the same. The first publication instead uses narrowly scoped response
policies for its three original HTML documents, including their extensionless
URLs. Only the known frozen style blocks and style-attribute values are allowed;
scripts, connections and forms are disabled. The application's policy stays
separate. PDF copies and source files remain available in the evidence archive.

Check the actual hosting response as well as the files: multiple matching
Content-Security-Policy headers can enforce several policies at once. The
document rules remove the inherited policy before setting the document policy.
The publication's tests pin the allowed hashes and route scope; browser checks
through the hosting runtime verify the final headers, exact response bytes and
rendered reading copies. A plain static-file preview alone does not exercise
this boundary. None of these presentation checks establishes scientific validity.
