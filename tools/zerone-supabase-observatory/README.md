# Zerone AgentTool Supabase Observatory 0.1

This directory contains a source-only, offline-verifiable contract for a
future rebuildable Supabase observation plane. It does not connect to
Supabase, AgentTool, an RPC endpoint, or a chain.

Run from the Zerone repository root:

```bash
go test ./tools/zerone-supabase-observatory -count=1
go run ./tools/zerone-supabase-observatory --repository-root .
```

The command reads only explicit bounded repository files and emits one JSON
verification report to stdout. It does not write a report to disk.

## Contents

- `protocol/manifest.v0.1.json`: exact source pins, pending AgentTool binding,
  schema/fixture hashes, semantics, and effect boundaries.
- `protocol/observation-journal.v0.1.schema.json`: closed JSON Schema 2020-12
  tagged union for `STATIC_TREE`, `TOK_ONCHAIN`, and `KNOWLEDGE_GEOMETRY`.
- `testdata/valid-current-observation.json`: accepted current-only ToK shape.
- `testdata/same-height-conflicts-preserved.json`: accepted conflicting
  same-height observations; neither overwrites the other.
- `testdata/truncated-unavailable-source-fails-closed.json`: expected refusal.
- `validator.go` and `strictjson.go`: semantic and hostile-input verifier.

The full normative boundary and next gate are in
[`docs/specs/zerone-agenttool-supabase-observatory-v0.1.md`](../../docs/specs/zerone-agenttool-supabase-observatory-v0.1.md).

The four changed ToK-feedback source pins are exact `CANDIDATE_LOCAL_BYTES`
with revision `UNPUBLISHED`, not bytes attributed to the historical commit.
Their digests bind the inspected candidate source without asserting a verified
publication revision. Publishing this code does not upgrade that source-binding
assurance. No release signature, deployed source identity, or network observation
is implied. The current manifest's exact raw-byte seal is
`sha256:a3e3a8e3448b873bb21a93fcd2881c2376361a1ccf68e929081985ad74235205`.
Unchanged historical and external pins retain their own provenance. The bounded
current-only ToK root still excludes full payload; the projection still does not
claim completeness, and its default public entrypoint is closed without a
publication binding. Passing this offline contract does not activate the facade.

The future AgentTool hosting profile is intentionally `PENDING_UNTRACKED` with
null revision/path/digest. Current AgentTool main is a baseline pin only; this
artifact makes no reciprocal conformance or integration-ready claim.
