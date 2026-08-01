# Operations rehearsal evidence index

`operations-rehearsal` is a standard-library-only compiler and verifier for
Zerone upgrade, halt, recovery, and hostile-event rehearsal evidence. It does
not run a chain, observe a deployment, attest that an event really happened, or
make a release decision. Its result is a canonical, hash-bound evidence index.

Schema `zerone.operations-rehearsal/v1` indexes observations for:

- a legacy binary stopping at H−1 and one target-binary commit at H;
- activation preflight, exact `Plan.Info`, deterministic replay, and restart;
- transaction quarantine while block production continues;
- recovery authorization, wrong-tuple rejection, revocation, proposal
  failure/dequeue/refund, and resume-head verification;
- closed admission in the resume block and admission at H+1;
- a complete validator-home move to a fresh, distinct volume;
- an independent observer of the H−1 state and upgrade plan; and
- hostile fault injections selected from a stable reason-code matrix.

`quick` mode requires every matrix entry marked `core`. `full` mode requires
every reason code. The report outcome is always `evidence_indexed`; scenario
and fault outcomes are always `observed`.

## Typed evidence

Every critical evidence kind has its own observation schema. A generic JSON
claim is not accepted. Each evidence file is a canonical envelope with:

- schema `zerone.operations-evidence/<kind>/v1`;
- collector name/version and collector-binary digest;
- invocation, runner, PID, command digest, timestamps, and exit status;
- the exact rehearsal run ID and chain ID;
- kind-specific observations;
- at least one separately hash-bound raw execution artifact; and
- an envelope self-hash.

The compiler cross-links those observations to report fields. This includes
binary and plan digests, H−1/H replay state, quarantine results, proposal IDs,
resume heights, observer state, fault outcomes, and the complete canonical
`zerone.validator-home-manifest/v1` shape and self-hash.

An envelope self-hash proves internal byte consistency, not provenance.
Collector names, runner identities, and observations remain self-attested
unless an external policy separately signs the envelope and pins the signing
key, collector binary, and authorized runner. Keep a detached signature over
the sealed envelope alongside the index, or anchor its digest in a separate
transparency record; do not create a circular envelope/signature reference.

## Artifact handling

All paths are relative to one evidence root. The tool refuses path escapes,
symlinks, special files, duplicate paths, empty artifacts, size or digest
drift, malformed or duplicate-key JSON, and invalid UTF-8 text. JSON and text
are parsed from the same no-follow-opened bytes that are hashed.

Create a reference to an immutable raw artifact:

```sh
go run ./tools/operations-rehearsal digest \
  --evidence-root /secure/rehearsal/run-001 \
  --path raw/upgrade-info.json \
  --kind raw-command-output \
  --media-type application/json
```

Create an envelope draft with an empty `envelope_sha256`, then seal it:

```sh
go run ./tools/operations-rehearsal seal-evidence \
  --draft /secure/rehearsal/run-001/drafts/upgrade-info.json \
  --evidence-root /secure/rehearsal/run-001 \
  --out /secure/rehearsal/run-001/upgrade/upgrade-info.json
```

The output path must not exist. Its parent must already exist and contain no
symlinked path components. Installation is no-replace and the directory is
synced before success is reported.

List the canonical fault matrix:

```sh
go run ./tools/operations-rehearsal fault-matrix
```

## Compile and verify

Build a report draft with `evidence_manifest_sha256` and `report_sha256` set to
empty strings. Evidence arrays may be unsorted; compilation normalizes them
and sorts faults by ID.

```sh
go run ./tools/operations-rehearsal compile \
  --draft /secure/rehearsal/run-001/draft.json \
  --evidence-root /secure/rehearsal/run-001 \
  --out /secure/rehearsal/run-001/report.json

go run ./tools/operations-rehearsal verify \
  --report /secure/rehearsal/run-001/report.json \
  --evidence-root /secure/rehearsal/run-001
```

The output summary starts with `VERIFIED_EVIDENCE_INDEX`. It means that the
schemas, canonical encoding, self-hashes, attachments, and report cross-links
were verified at that moment. It does not mean the underlying observations
are true or that production should proceed. The summary explicitly records
`provenance=self_attested`, `external_controls=unverified`, and
`release_decision=none`.

Canonical documents are compact JSON with one trailing newline. The report
self-hash covers the report with `report_sha256` empty. The evidence-manifest
hash covers the complete sorted references, including kind, path, media type,
size, and digest.
