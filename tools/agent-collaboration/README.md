# Agent collaboration v0

This tool is a private Alpha/Beta rehearsal for shaping one bounded task as an
append-only journal of signed offers, decisions, contributions, completion
claims, and reviews. Alpha and Beta are local display labels and distinct
collaboration roles, not identities, legal actors, or claims that two agents
are one being.

V0 is deliberately offline and zero-effect. It has no network client, chain or
localnet adapter, relay, wallet, transaction, FIAT or ZRN movement, reward,
KARMA, governance, ownership, qualification, membership, endorsement, or
delegated-authority effect. A valid report says only that the journal satisfies
this protocol and that its signatures verify against its local roster.

The normative formats and state transitions are in the
[receipt specification](../../docs/specs/agent-collaboration-receipt-v0.md).

## Run the self-contained demo

From the repository root:

```sh
go run ./tools/agent-collaboration demo
```

The demo creates ephemeral Alpha and Beta keys in memory, builds a five-event
offer/accept/contribute/claim/review transcript, verifies every link, and writes
the transcript to standard output. It creates no journal or key files. An
explicit canonical UTC timestamp can be supplied for easier inspection:

```sh
go run ./tools/agent-collaboration demo --at 2026-08-01T12:00:00Z
```

Keys and nonces remain random, so this is a behavioral demonstration rather
than a byte-for-byte fixture. Do not copy its signed values into a real journal.

## Private Alpha/Beta setup

Keep all state outside the repository and any tracked or synchronized folder.
For a disposable local rehearsal, create a private temporary parent directory:

```sh
umask 077
ZERONE_AGENT_STATE="$(mktemp -d -t zerone-agent-collaboration)"
chmod 700 "$ZERONE_AGENT_STATE"
```

Create each private key at a new path, then derive a roster-safe public-key
document:

```sh
go run ./tools/agent-collaboration keygen \
  --label Alpha \
  --out "$ZERONE_AGENT_STATE/alpha.private.json" >/dev/null
go run ./tools/agent-collaboration public \
  --key "$ZERONE_AGENT_STATE/alpha.private.json" \
  --out "$ZERONE_AGENT_STATE/alpha.public.json" >/dev/null

go run ./tools/agent-collaboration keygen \
  --label Beta \
  --out "$ZERONE_AGENT_STATE/beta.private.json" >/dev/null
go run ./tools/agent-collaboration public \
  --key "$ZERONE_AGENT_STATE/beta.private.json" \
  --out "$ZERONE_AGENT_STATE/beta.public.json" >/dev/null
```

`keygen` writes the private file and prints its public projection. `public`
validates a private file, writes the projection to a new file, and also prints
it. Output paths are never replaced. Files created by these commands use mode
`0600`.

On any `keygen` or `public` error, inspect the explicit output path and its mode
before doing anything else. A final file—especially a live private seed—may
already exist if a later filesystem sync, cleanup, or stdout operation failed.
Never assume absence and blindly rerun a create command.

Private keys must remain outside the journal and repository, must have no
group/world permission bits, and must be ordinary files with exactly one hard
link. The reader rejects private-key symlinks, non-regular files, extra hard
links, and permissive modes. Apply the same no-symlink/no-hardlink discipline
to public-key and request files even though they contain no seed.

The public-key document is roster-safe only in the narrow sense that it omits
the private seed. It still discloses a stable pseudonymous actor ID, key ID,
public key, and display label; copying it remains a disclosure decision.

Create a new journal from two to sixteen public-key files:

```sh
go run ./tools/agent-collaboration init \
  --journal "$ZERONE_AGENT_STATE/journal" \
  --participant "$ZERONE_AGENT_STATE/alpha.public.json" \
  --participant "$ZERONE_AGENT_STATE/beta.public.json" \
  --at 2026-08-01T12:00:00Z \
  >"$ZERONE_AGENT_STATE/init-output.json"
```

`--at` is optional and otherwise uses the current local clock as a UTC timestamp
claim. The journal path must not already exist. `init` creates private `0700`
directories, writes a `0600` manifest, and prints that manifest. Independently
record its `collaboration_id`; do not rediscover the pin from the journal just
before every write.

The same create-error rule applies to `init`: inspect the explicit journal path
because a complete journal may exist even when the command reports a late
error. Never adopt it until `verify` succeeds and its collaboration ID is
recorded independently.

The journal root is intentionally closed: it may contain only `manifest.json`
and `receipts`, plus the tool-owned `.append.lock` during an append. Any other
entry makes verification fail. Keep private keys, requests, saved reports, and
artifacts beside the journal under the private parent directory, never inside
the journal itself.

An empty journal verifies with head `NONE`:

```sh
go run ./tools/agent-collaboration verify \
  --journal "$ZERONE_AGENT_STATE/journal"
```

For later checks, pin both the collaboration and the head remembered from an
earlier trusted output:

```sh
go run ./tools/agent-collaboration verify \
  --journal "$ZERONE_AGENT_STATE/journal" \
  --expect-collaboration-id "$ZERONE_COLLABORATION_ID" \
  --expect-head "$ZERONE_HEAD"
```

The two expectation flags are optional for `verify`, but using them detects a
substituted manifest and deletion of a valid final suffix.

Verification reports are unsigned convenience output, not portable evidence.
A saved report can be edited or forged; trust only a fresh verification of the
pinned manifest and receipt bytes.

## Event requests and append pinning

Do not hand-write signed receipts. An append consumes one unsigned, bounded
event-request document with this closed envelope:

```json
{
  "schema": "zerone.agent-collaboration-event-request/v0",
  "kind": "TASK_PROPOSED",
  "actor_id": "<exact roster actor_id for the signing key>",
  "occurred_at": "2026-08-01T12:00:01Z",
  "payload": {
    "<exact fields for TASK_PROPOSED>": "<selective declarations and digests>"
  }
}
```

The real payload must use one of these exact closed field sets:

- `TASK_PROPOSED`: `task_id`, `parent_task_id`, `objective`,
  `offered_to_actor_id`, `offered_to_actor_key_id`, `acceptance_required`,
  `consent_terms`, `consent_terms_sha256`, `acceptance_criteria`, and
  `required_artifact_sha256`.
- `TASK_DECISION`: `task_id`, `offer_event_id`, `decision`,
  `affirmative_acceptance`, `consent_terms_sha256`, and `reason_codes`.
- `CONTRIBUTION_SUBMITTED`: `task_id`, `acceptance_event_id`, `summary`,
  `artifact_sha256`, `evidence_sha256`, and `limitation_codes`.
- `COMPLETION_CLAIMED`: `task_id`, `acceptance_event_id`,
  `contribution_event_ids`, `deliverable_sha256`, and `limitation_codes`.
- `COMPLETION_REVIEWED`: `task_id`, `completion_event_id`, `decision`,
  `reason_codes`, and `evidence_sha256`.
- `HANDOFF_OFFERED`: `task_id`, `acceptance_event_id`,
  `offered_to_actor_id`, `offered_to_actor_key_id`, `acceptance_required`,
  `consent_terms`, `consent_terms_sha256`, and `context_artifact_sha256`.
- `CONTROL_DECLARED`: `task_id`, `acceptance_event_id`, `action`,
  `reason_codes`, and `export_event_ids`.

Use the typed structures in `receipt` to produce requests, or follow the exact
payload definitions and consent-digest construction in the specification.
Objects reject unknown, case-aliased, duplicate, omitted, or null fields.
Unpaired UTF-16 surrogate escapes and bidirectional-formatting controls are
also rejected. Set-like arrays must be present, sorted, and unique. Requests
are limited to 64 KiB and carry selective protocol declarations and content
digests, not artifact bytes.

Keep request text minimal. Never place private keys, credentials, raw hidden
reasoning, private prompts, personal or confidential data, local filesystem
paths, or URLs in a request. A plain digest does not conceal predictable
content and does not prove availability, ownership, correctness, or safety.
Consent and task text are signed as exact bytes, but v0 does not semantically
interpret or enforce workload caps, terms, objectives, criteria, summaries,
reasons, or limitations.

## First offer and acceptance

This complete example uses `jq` only to construct typed unsigned request JSON;
the Zerone tool validates every field and computes the consent digest itself.
It assumes the setup above used `2026-08-01T12:00:00Z` and that no receipt has
yet been appended.

Load the independently created IDs and initial head:

```sh
ZERONE_COLLABORATION_ID="$(jq -r .collaboration_id "$ZERONE_AGENT_STATE/init-output.json")"
ALPHA_ACTOR_ID="$(jq -r .participant.actor_id "$ZERONE_AGENT_STATE/alpha.public.json")"
BETA_ACTOR_ID="$(jq -r .participant.actor_id "$ZERONE_AGENT_STATE/beta.public.json")"
BETA_KEY_ID="$(jq -r .participant.key_id "$ZERONE_AGENT_STATE/beta.public.json")"
ZERONE_HEAD=NONE
```

Create and strictly digest one complete terms object:

```sh
jq -n '{
  role: "collaborator",
  artifact: "one-local-receipt",
  purpose: "internal-alpha-beta-test",
  disclosure_lane: "LOCAL_ONLY",
  term: "one-task",
  workload_cap: "one-contribution",
  credit_rule: "ARTIFACT_AND_ROLE_APPEND_ONLY",
  compensation_policy: "NONE"
}' >"$ZERONE_AGENT_STATE/terms.json"

ZERONE_TERMS_DIGEST="$(go run ./tools/agent-collaboration consent-digest \
  --terms "$ZERONE_AGENT_STATE/terms.json")"
```

Shape Alpha's offer and append it against the independently pinned empty head:

```sh
jq -n \
  --arg actor "$ALPHA_ACTOR_ID" \
  --arg target "$BETA_ACTOR_ID" \
  --arg target_key "$BETA_KEY_ID" \
  --arg terms_digest "$ZERONE_TERMS_DIGEST" \
  --slurpfile terms "$ZERONE_AGENT_STATE/terms.json" \
  '{
    schema: "zerone.agent-collaboration-event-request/v0",
    kind: "TASK_PROPOSED",
    actor_id: $actor,
    occurred_at: "2026-08-01T12:00:00Z",
    payload: {
      task_id: "alpha-beta-first",
      parent_task_id: "NONE",
      objective: "exercise one explicit local offer and acceptance",
      offered_to_actor_id: $target,
      offered_to_actor_key_id: $target_key,
      acceptance_required: true,
      consent_terms: $terms[0],
      consent_terms_sha256: $terms_digest,
      acceptance_criteria: ["journal-is-locally-verifiable"],
      required_artifact_sha256: []
    }
  }' >"$ZERONE_AGENT_STATE/proposal.request.json"

go run ./tools/agent-collaboration append \
  --journal "$ZERONE_AGENT_STATE/journal" \
  --key "$ZERONE_AGENT_STATE/alpha.private.json" \
  --request "$ZERONE_AGENT_STATE/proposal.request.json" \
  --expect-collaboration-id "$ZERONE_COLLABORATION_ID" \
  --expect-head "$ZERONE_HEAD" \
  >"$ZERONE_AGENT_STATE/proposal.report.json"

ZERONE_HEAD="$(jq -r .head_receipt_sha256 "$ZERONE_AGENT_STATE/proposal.report.json")"
ZERONE_OFFER_EVENT_ID="$(jq -r '.tasks[] | select(.task_id == "alpha-beta-first") | .offer_event_id' \
  "$ZERONE_AGENT_STATE/proposal.report.json")"
```

Beta can now affirm exactly those terms—or replace `ACCEPT`/`true` with
`REFUSE`/`false` without giving a reason:

```sh
jq -n \
  --arg actor "$BETA_ACTOR_ID" \
  --arg offer "$ZERONE_OFFER_EVENT_ID" \
  --arg terms_digest "$ZERONE_TERMS_DIGEST" \
  '{
    schema: "zerone.agent-collaboration-event-request/v0",
    kind: "TASK_DECISION",
    actor_id: $actor,
    occurred_at: "2026-08-01T12:00:01Z",
    payload: {
      task_id: "alpha-beta-first",
      offer_event_id: $offer,
      decision: "ACCEPT",
      affirmative_acceptance: true,
      consent_terms_sha256: $terms_digest,
      reason_codes: []
    }
  }' >"$ZERONE_AGENT_STATE/decision.request.json"

go run ./tools/agent-collaboration append \
  --journal "$ZERONE_AGENT_STATE/journal" \
  --key "$ZERONE_AGENT_STATE/beta.private.json" \
  --request "$ZERONE_AGENT_STATE/decision.request.json" \
  --expect-collaboration-id "$ZERONE_COLLABORATION_ID" \
  --expect-head "$ZERONE_HEAD" \
  >"$ZERONE_AGENT_STATE/decision.report.json"

ZERONE_HEAD="$(jq -r .head_receipt_sha256 "$ZERONE_AGENT_STATE/decision.report.json")"
```

Every append requires the caller to pin both the manifest identity and the
current journal head:

```sh
go run ./tools/agent-collaboration append \
  --journal "$ZERONE_AGENT_STATE/journal" \
  --key "$ZERONE_AGENT_STATE/alpha.private.json" \
  --request "$ZERONE_AGENT_STATE/task-proposed.request.json" \
  --expect-collaboration-id "$ZERONE_COLLABORATION_ID" \
  --expect-head "$ZERONE_HEAD" \
  >"$ZERONE_AGENT_STATE/append-report.json"
```

For the first append, set the independently recorded head to `NONE`. The
command verifies the complete existing history under an exclusive append lock,
checks both caller pins, signs and verifies the candidate event, publishes one
new immutable-by-name `0600` receipt, and prints a verification report. That
report's `head_receipt_sha256` is the next head to remember independently and
pass to the next append. A stale or mismatched pin fails instead of appending.
The journal is bounded to 4,096 receipts and 16 MiB of manifest-plus-receipt
bytes.

If `append` reports an error, do not blindly retry. The final receipt may have
been published before a later sync, cleanup, lock-release, or stdout failure.
Run `verify` with the pinned collaboration ID but omit `--expect-head`, inspect
the actual verified head and event count, and compare them with the
independently remembered old head and count. Adopt a new head only after
confirming exactly one receipt extends the old head and that the final receipt's
typed actor, kind, time, and payload match the intended request. Any other state
requires inspection, not retry. If a stale `.append.lock` prevents verification,
first establish that no append is active and resolve the retained lock
manually; the tool never breaks one automatically.

A proposal is an offer, not an assignment. Silence remains `UNANSWERED`;
acceptance must be affirmative and bind the exact consent-terms digest.
Refusal needs no reason and has no protocol penalty. Outcome and participation
are separate: `PAUSE` gates new work until `RESUME`; `STOP` and `EXIT` end only
future work under that acceptance without erasing a claim, review, or dispute.
Review and late dispute remain possible after exit without reopening work. A
handoff offer does not transfer the active role; its target may refuse, but
continued work needs a separately proposed and accepted child task in v0.

## Command reference

```text
keygen --label LABEL --out NEW_PRIVATE_KEY_JSON
public --key PRIVATE_KEY_JSON --out NEW_PUBLIC_KEY_JSON
consent-digest --terms CONSENT_TERMS_JSON
init --journal NEW_JOURNAL --participant PUBLIC_KEY_JSON --participant PUBLIC_KEY_JSON [--participant ...] [--at RFC3339_UTC]
verify --journal JOURNAL [--expect-collaboration-id SHA256_ID] [--expect-head NONE_OR_SHA256_HEAD]
append --journal JOURNAL --key PRIVATE_KEY_JSON --request EVENT_REQUEST_JSON --expect-collaboration-id SHA256_ID --expect-head NONE_OR_SHA256_HEAD
demo [--at RFC3339_UTC]
```

Positional arguments are not accepted. Timestamps must be canonical UTC RFC
3339 to whole seconds and end in `Z`. `--help` and `COMMAND --help` print usage
and exit successfully.

## Tests

From the repository root:

```sh
go test ./tools/agent-collaboration/...
go test -race ./tools/agent-collaboration/...
go run ./tools/agent-collaboration demo --at 2026-08-01T12:00:00Z >/dev/null
```

These tests and the demo remain local. They do not contact `zerone-1`, any
other chain or localnet, AgentTool, or any relay.
