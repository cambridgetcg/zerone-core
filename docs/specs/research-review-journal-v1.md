# Research Review journal v1

Status: local off-chain format and browser implementation. No consensus, module,
token or transaction changes. [Usage](../../tools/research-review/README.md).

## Time, attribution and identity

`recorded_at` is the local store's generated UTC append time; `sequence` defines
collection order. `occurred_on` is a declared source/event date, or null. Neither
clock is an independently authenticated historical timestamp. `attributed_to`
is a declared label, not a verified signer. Bibliographic authors are not treated
as submitters, endorsers or people consenting to a new claim.

Replay selects a prefix through an inclusive sequence. It answers what this
supplied collection contained then, using records retained now. It does not
reconstruct missing blockchain transactions or claim what the entire scientific
community knew. Full export validation happens before cutoff selection; later
corrupt bytes are not hidden by requesting an earlier snapshot.

## Envelope and canonical encoding

`journal.jsonl` begins with:

```json
{"schema":"zerone-research-journal/v1","collection_id":"<UUID>","created_at":"<UTC timestamp>"}
```

Subsequent rows have exactly `sequence`, `recorded_at`, `previous_sha256`,
`record`, `sha256`. Sequence starts at 1 and is contiguous. The first previous
hash is SHA-256 of the canonical header; later rows refer to the preceding row
hash. A row's hash covers its canonical encoding with `sha256` omitted.
Timestamps use `YYYY-MM-DDTHH:MM:SS.ffffffZ` and are nondecreasing.

Canonical JSON recursively sorts object keys, uses compact separators and
literal Unicode encoded as UTF-8. No Unicode normalization is applied; invalid
surrogates, duplicate keys, floats and non-finite numbers are refused. All
payload keys are the known ASCII schema keys. Unknown fields are rejected.

Export format has exactly:

```json
{"schema":"zerone-research-export/v1","header":{},"entries":[]}
```

`header` is the complete header above and `entries` the complete row array.
Hashes establish internal consistency, not authenticity, truth, completeness or
resistance to rewriting and rehashing by a local controller. Browser imports do
not claim to compare an external trusted head. Selected historical prefixes
carry no proof that later entries do not exist.

## Records

All records contain the following required fields:

| Field | Value |
| --- | --- |
| `id` | ASCII `[a-z0-9][a-z0-9:._-]{0,127}`, unique for the collection |
| `kind` | `contribution`, `relation`, `concern`, `assessment` |
| `title` | 1–300 characters |
| `summary` | 1–8192 UTF-8 bytes |
| `attributed_to` | 1–200 characters; declared attribution |
| `occurred_on` | Valid `YYYY-MM-DD` or null; partial dates remain null |
| `evidence` | 0–16 evidence objects |

Each evidence object has exactly `label` (1–200 characters), `url` (null or
HTTPS URL without credentials, at most 2048 characters) and `sha256` (null or 64
lowercase hex digits). At least URL or hash is required. A URL is a locator; a
hash is a byte commitment. Neither promises availability or scientific validity.

Every kind optionally accepts `supersedes`, referring to an earlier record of
the same kind. Replacement does not delete the original, inherit its assessments
or establish that the replacement is correct.

Additional fields by kind:

| Kind | Required additional fields |
| --- | --- |
| `contribution` | `contribution_type`: `source`, `question`, `claim`, `method`, `experiment`, `finding`, `review`; `doi`: normalized lowercase DOI or null |
| `relation` | `from_id`, `to_id`: distinct earlier contributions; `relation_type` from the table below |
| `concern` | `target_id`: earlier contribution; `category`: `publisher-notice`, `reported-concern`, `metadata-conflict`, `artifact-discrepancy`; `notice_type`: original provider label (1–100 characters) or null; `provider`: 1–200 characters |
| `assessment` | `target_id`: earlier concern or relation; `disposition`: `needs-review`, `notice-checked`, `disputed`, `addressed`, `inconclusive` |

All references resolve to earlier records, including within a batch. A concern
is an attributed lead, never a verdict of misconduct. Assessments remain plural;
there is no last-writer global resolution. `notice-checked` means a local reviewer
says they checked a notice, not that a result or allegation has been proved.

## Relationship directions and review propagation

| Relation | Stored `from_id → to_id` meaning | Reassessment traversal |
| --- | --- | --- |
| `supports` | evidence → supported claim | same direction |
| `requires` | dependent → prerequisite | reverse direction |
| `uses-input` | activity → input | reverse direction, labeled input |
| `inspired-by` | contribution → inspiration | none |
| `refines` | newer contribution → earlier contribution | none |
| `cites` | citing contribution → referenced source | none |

Relations are attributed assertions. Multiple assertions may coexist and have
their own assessments. Traversal follows visible recorded edges, is cycle-safe
and bounded, and returns the contribution IDs and relation IDs for each path.
Neither an assessment nor native chain status silently edits this graph.

Path reachability identifies possible reassessment. It does not establish
logical dependence, falsity or the scope of a publisher's notice. Independently
supported results may survive a withdrawn premise. Grouped premises, alternative
support and sufficiency are not encoded in this first version.

## Storage and limits

New store directories are explicitly created, never adopted over existing data.
`journal.jsonl` and `.lock` must be regular files. Symlink paths, nonregular files
and overwritten export destinations are refused. Appends take an exclusive
process lock, validate existing history, then replace the journal atomically
while retaining prior row bytes. Hash-addressed metadata/lookup artifacts live
separately under `artifacts/`; exports have no private filesystem locators.

Limits: 1,000 entries, 8 MiB journal/export, 64 KiB individual input record, and
the field bounds above. Invalid or partial journals fail closed. The fictional
site fixture is generated with the same format and is checked by both the
Python and TypeScript implementations. No automatic pruning or evidence deletion
is part of this version.

The [fork and comparison workflow](research-journal-branches-v1.md) retains an
exact source export in a fresh store and allows another local continuation.
It preserves this wire format and the complete inherited header and rows.
Matching collection UUIDs do not establish a single canonical writer or branch.
