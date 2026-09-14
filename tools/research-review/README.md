# Research Review

Record contributions, their dependencies and sourced concerns in a local journal;
replay what the collection contained at an earlier point. This is the first
off-chain implementation of the Tree of Knowledge's contribution history.
The [browser view](https://zerone.ai/research/review/) loads an explicitly fictional
demo or a JSON export selected from your computer. Selected files remain in
browser memory. The view verifies the export before displaying its contents.

Use Python 3.10+ on macOS/Linux. These local Python tools have no dependencies,
accounts, background services, signing keys or blockchain transactions. Work in a real
directory: stores and file paths containing symlinks are refused. On macOS,
`/tmp` and `/var` are system symlinks; use a physical path or a directory under
your home instead. Windows locking is not supported in this version.

## Start a collection

From the repository root, choose a new private directory whose parent exists:

```sh
python3 -B tools/research-review/journal.py init ./my-research
python3 -B tools/research-review/scan.py ./my-research \
  10.1023/A:1010933404324 10.1007/BF00994018
python3 -B tools/research-review/journal.py export ./my-research --output ./review.json
```

The explicit scan sends those DOIs to Crossref. The two example works are routine
controls, not suspected misconduct. Open Research Review and select `review.json`.
Exports never overwrite an existing file; choose a new output after a refresh.
Keep the store's `artifacts/` directory with your collection: the export carries
hashes and source URLs, not embedded source responses. Raw metadata can contain
abstracts with separate rights; this tool does not publish it.

The scanner performs one production Crossref work lookup per explicit DOI, at
most 25 per invocation, sequentially with a one-second interval. Requests have
a 20-second timeout and a 2 MiB response bound. It stops on HTTP 429; retry later
explicitly. It does not crawl citations, fetch notice pages, scrape PubPeer or
execute paper code. A nonzero exit means a failed/incomplete scan or refusal;
inspect the retained coverage observations and JSON output.

Original work `updated-by` entries point to notice DOIs. A notice's `update-to`
entries point to originals. Same-DOI editorial updates are retained. Repeated
original/notice/type assertions from publisher and Retraction Watch are one lead
with both sources, rather than two independent confirmations. Every successful
or failed lookup has its own observation and receipt. Raw responses are stored
by SHA-256; refresh never erases an earlier concern. Unknown labels remain
inspectable. Differing labels/dates for the same notice in one response produce
a metadata concern; changed date/source attribution on a later lookup also
creates a visible metadata concern while retaining the first observation.
Later snapshots do not automatically resolve earlier notices. Concurrent scanners can encounter a
duplicate-ID refusal; retry after the other append completes.

**A returned notice is a sourced lead. No notice returned is a coverage result.**
Neither tells you that a paper is fraudulent, false, correct or free of concerns.
An expression of concern, a correction, a retraction and a reproducibility
failure have different scopes. Read the actual notice, retain any author
response, and identify exactly which claim or artifact is affected. An editorial
concern about peer review does not prove a mathematical theorem false.

## Add and assess a contribution

Create a UTF-8 JSON file containing one record, or an ordered list of records.
Here is a fictional example (`record.json`):

```json
{
  "id": "example:question-1",
  "kind": "contribution",
  "title": "Does the bound require independent samples?",
  "summary": "Fictional question for a future proof review; no result asserted.",
  "attributed_to": "Local reviewer (declared label)",
  "occurred_on": null,
  "evidence": [],
  "contribution_type": "question",
  "doi": null
}
```

```sh
python3 -B tools/research-review/journal.py append ./my-research ./record.json
python3 -B tools/research-review/journal.py inspect ./my-research --through 2
python3 -B tools/research-review/journal.py impact ./my-research example:question-1
```

Use the [record specification](../../docs/specs/research-review-journal-v1.md)
for relationship, concern and assessment fields. Referenced records must already
exist, or appear earlier in the same atomic batch. Corrections get a fresh ID and
may name `supersedes`; old records and their assessments remain attached to the
original. Multiple reviewers can disagree. An `addressed` assessment is that
reviewer's statement, not a global clearance of the concern.

`impact` follows recorded support, prerequisite and input links to identify paths
worth reassessing. It does not propagate through citations, inspiration or
refinement. A path is not a proof that a downstream claim is false. Missing
relations and independent alternative evidence are not inferred. Logical groups
of necessary/sufficient premises are a later design step.

## Integrity and history

The journal generates append times and sequences; `occurred_on` is a separately
declared historical date. Importing a 1998 paper today therefore does not pretend
Zerone witnessed it in 1998. The browser's cutoff selects records by sequence,
then optionally sorts the selected contributions by their reported dates. A
late discovery of an early idea remains a late addition to this collection.

Every row hashes the previous row and its own complete payload. Readers validate
the entire supplied history before selecting a cutoff. This detects inconsistent
bytes **within the supplied export**. A person controlling the store can rewrite
and rehash it or supply a valid shorter prefix. An external saved head hash is
needed to detect such replacement. The journal and browser alone do not provide
independent timestamping, authenticated identities or completeness proofs.
Keep exports or head hashes separately when those comparisons matter. The
optional [checkpoint workflow](../../docs/specs/research-checkpoint-v1.md) binds
an exact export to an existing development-chain transaction.

Stores use a process lock, private permissions and atomic batches. Limits are
1,000 entries and 8 MiB per journal/export; this is a small collection tool.
Reaching a limit refuses further work, rather than silently dropping history.
Interrupted scans can leave unreferenced content-addressed artifacts. No garbage
collector deletes evidence. Browser imports render text as text and never fetch
evidence URLs automatically; opening a source link is an explicit user action.
If a filesystem sync fails after atomic replacement, a complete batch may already
be visible despite the reported error. Inspect the collection before retrying;
immutable IDs prevent a repeated batch from being appended twice.

## Where this fits Zerone

This release records local contributions and observations. The blockchain's
existing claim, review and challenge messages remain a separate admission
process; a DOI import does not create a native Fact or submit a claim. Before a
later chain submission, choose a bounded statement and reproducible evidence,
adopt it under the actual contributor's key, and prepare an explicit mapping to
the native record. There is no automatic upload, reward or scientific verdict.

Useful next investigation: select one documented concern, extract the exact
claim and assumptions, preserve the notice and response, and attempt a bounded
proof or computational check. Publish the result only with its evidence and
scope. Reward design should value careful review, corrected concerns, negative
results and shared artifacts; paying for accusation counts would invite gaming.

## Prepare a storage checkpoint

Freeze an exported file before preparing the memo. These commands are local and
do not sign or submit anything:

```sh
python3 -B tools/research-review/checkpoint.py prepare ./review.json \
  --chain-id zerone-dev-1 --output ./checkpoint.json
python3 -B tools/research-review/checkpoint.py verify ./review.json ./checkpoint.json \
  --chain-id zerone-dev-1 --memo '<exact memo from the transaction>'
```

The verification result means the file matches that memo. Follow the
[offline chain-proof instructions](../research-checkpoint/README.md) to verify
the transaction's signature, inclusion and validator trust anchor. A checkpoint
uses a signed bank self-send and does not open a claim review round. The chain
holds the digest commitment; publish or replicate the full records separately.
See the [published checkpoint](https://zerone.ai/research/checkpoints/) for a
worked example, downloadable history and explicit evidence-availability limits.

## Verify

```sh
python3 -B -m unittest discover -s tools/research-review -p 'test_*.py' -v
```

Dashboard tests verify the same wire format, fictional fixture, historical
cutoffs and invalid imports. See the normal SDK/dashboard build instructions.

Provider references: [Crossref production update metadata](https://community.crossref.org/t/deprecating-retraction-watch-annotations-in-the-labs-api/15884),
[Crossmark update types and in-situ updates](https://www.crossref.org/documentation/crossmark/participating-in-crossmark/),
[Retraction Watch metadata](https://www.crossref.org/documentation/retrieve-metadata/retraction-watch/),
[REST access and limits](https://www.crossref.org/documentation/retrieve-metadata/rest-api/access-and-authentication/).
