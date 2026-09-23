# Research workflow v1

Status: **client integration, 23 September 2026**. A local workflow packet binds
one journal contribution and an explicitly chosen assertion to the existing
development participant client. It brings the resulting native review,
correction and funding observations back into Research Review. It changes no
consensus message, keeper, funding policy, participant package or journal schema.

Use the [walkthrough](../../tools/research-review/README.md#carry-a-record-through-native-review)
and [participant guide](../SHARED-DEVELOPMENT.md) for commands. The Python adapter
is `tools/research-review/workflow.py`; the browser validator is
`dashboard/src/research-workflow.ts`. Both validate the same packet.

## One record, one deliberate admission

1. Export the local journal. Select an existing `contribution` and write the
   bounded assertion, reasoning/evidence references, domain, category and
   optional method you intend to submit. Original attribution stays intact.
2. Prepare and inspect a packet locally. Preparation creates no keys, native
   claim, review window or payment. The complete journal is embedded for later
   inspection, including entries beyond a browser timeline cutoff.
3. Submit explicitly through an existing participant home and its original
   verified client. The adapter checks the home's public binding and all three
   pinned client files before invoking that client. It does not read signing
   keys, replace bindings or implement a second signer/broadcaster.
4. The reviewer uses their own client to inspect, commit and reveal their own
   judgment. Refresh the packet to retain observed native history; import it
   into the browser to see the same record alongside rounds and settlements.
5. A journal correction appends a new contribution with `supersedes`. Its packet
   preserves the original record, but a new native assertion requires a new
   explicit admission and fees. A native challenge separately requires a real
   derived Fact ID and scoped counterevidence. Neither path erases an old claim.

Only the selected claim fields and a hash binding are submitted. The full
journal and referenced evidence are not uploaded by the adapter or reader.
Preserve and share appropriate evidence separately. Sharing a packet shares
its entire embedded journal; hashes do not make its contents private.

## Packet and hash binding

The exact top-level fields of `zerone-research-workflow/v1` are:

| Field | Meaning |
| --- | --- |
| `schema` | Literal `zerone-research-workflow/v1`. |
| `journal` | `export_utf8`, `export_sha256`, `collection_id`, `head_sha256`, `record_id`, `entry_sha256`. The exact UTF-8 export and its complete journal are validated before selecting a contribution. |
| `intent` | Exact strings `content`, `reasoning`, `domain`, `category`, `method`; no inferred native verdict or inherited authorship. |
| `intent_sha256` | SHA-256 of canonical JSON containing `export_sha256`, `record_id`, `entry_sha256`, and `intent`. |
| `participant` | Initially null; later `chain_id`, public `address`, `descriptor_sha256`, raw-file `genesis_sha256`, local `binary_sha256`, `client_files`, and boolean `local_test`. |
| `submission` | Initially null; later `{state: "started", receipt: null}` or `{state: "observed", receipt: PUBLIC_RECEIPT}`. |
| `observations` | Ordered `{observed_at, history_utf8, history_sha256}` records. Each retained history matches the successful receipt and exact submitted assertion. |

Canonical JSON uses the journal's sorted-key, compact UTF-8 serialization,
without Unicode normalization. SHA-256 strings use 64 lowercase hexadecimal
characters. `client_files` contains exactly `shared-claims.py`,
`claim-workflow.py` and `local-node.py`, each with its pinned hash.

The native reasoning is the authored reasoning followed by two newline
characters, `Zerone research binding: `, and canonical JSON of:

```json
{
  "schema": "zerone-research-binding/v1",
  "intent_sha256": "<prepared intent digest>",
  "export_sha256": "<exact journal file digest>",
  "record_id": "<selected contribution ID>",
  "entry_sha256": "<selected journal entry digest>"
}
```

The displayed formatting above is illustrative; `inspect` emits the exact
prepared payload. Content is 20–1,000 UTF-8 bytes. Reasoning including the
binding is at most 8,192 bytes. Domain/category are nonempty and at most 128
bytes; method can be empty and is at most 128 bytes. Text must be valid UTF-8
without NUL. Values are refused, never silently shortened.

The public receipt allows only `id`, `action`, `actor`, `status`, `txhash`,
`height`, `code`, `gas_wanted`, `gas_used`, `claim_id`, `round_id`, `tx_bytes`,
`review_fee_uzrn`, `collateral_uzrn`. The first five are required. The action is
`submit`, actor is `participant`, ID is 32 lowercase hex characters, transaction
hash is 64 uppercase hex characters, and amounts are exact unsigned decimal
strings. Status distinguishes `prepared`, `broadcasting`, `pending`, `rejected`,
`committed`, `committed-unresolved` and `failed`. Only `committed` with positive
height, code zero and a claim ID permits history attachment.

## Recovery and retained observations

Commands lock one packet and write complete validated replacements with private
permissions and filesystem synchronization. Submission first checks the
client's tracked public snapshot for an existing exact match, including for a
fresh copy of the same draft. It persists `started` before invoking submission.
An uncertain child result leaves that marker intact. A second ordinary
`submit` refuses; `refresh` observes without signing or replaying a transaction.

Explicit `resume` is available only for `started` with a null receipt. It first
tries snapshot recovery, then delegates the unchanged bound payload to the
same verified client if no successful match exists. This can sign when no
attempt was ever launched; the existing client refuses a new admission while
its own attempt remains pending and reuses a committed exact context. It does
not automatically replay or reconstruct signed bytes. Known observed receipt
states use `refresh` and the client's explicit transaction `retry` as needed.

A matching snapshot must have the same participant, descriptor, chain and test
mode. Its committed receipt and history must agree on claim ID, submitter,
content, domain, category, method, complete bound reasoning, submission height
and any recorded initial round ID. More than one match is refused. Matching is
limited to the participant client's tracked inventory; it is not a network
search. A missing result stays unresolved instead of fabricating a receipt.

The packet retains up to 32 distinct history observations. Equal history bytes
are a no-op. Observation times use real UTC calendar timestamps with milliseconds
(`YYYY-MM-DDTHH:mm:ss.sssZ`); times and observed block heights cannot regress.
Funding allocations and settlement records receive the same structural checks
as the development history reader, with no future settlement heights. The CLI
retains canonical client-decoded history bytes; a browser refresh retains the
received history JSON. Their exact-byte digests need not match despite equivalent
decoded state. Receipt and snapshot validation do not prove transaction inclusion.

The packet is limited to 16 MiB, each embedded journal/history to 8 MiB, and
the journal retains its existing 1,000-entry bound. Reaching a limit refuses a
new observation without discarding old ones. Archive the packet and continue
with the participant client's `history`/`snapshot` commands; this version has no
automatic packet rotation or truncation.

## Browser and authority boundaries

Imports and preparation run locally without a wallet, browser persistence or
network request. Explicit current-history reads first verify the published
development descriptor, then request only the chosen claim's history. The
read URL and pins come from the website build, not from imported files. Public
reads require the published chain/raw genesis and current or preserved original
descriptor, with `local_test: false`. A participant's macOS binary hash is not
compared to the Linux node's binary hash. Test packets remain inspectable locally.

Both pages reuse the same bounded history fetcher and renderer. Saved state is
labelled with its observation height/time; failed reads clear the current result
while preserving earlier downloadable observations. Editing an intent creates
a new draft without attaching the old receipt to the changed statement.

The journal chain checks supplied bytes, not authorship or completeness. Packet
bindings associate an assertion with those bytes; they do not authenticate a
participant merely because a file names an address. Gateway histories are
endpoint observations, not independent inclusion or bank proofs. A settlement
with a positive `paid_at_block` records an application transfer; a zero marker
records a pending obligation. Account counts, funding, and acceptance do not
establish independent controllers, expertise or scientific truth.

The current runtime still includes Fact ecology and legacy authority paths.
This adapter neither retires those paths nor changes their effects. See the
[module sequence](../MODULE-CONSOLIDATION-SEQUENCE.md) for the separate protocol
work. Automated integration uses synthetic accounts controlled by one test,
on isolated loopback chains; it does not complete the human review handoff.
