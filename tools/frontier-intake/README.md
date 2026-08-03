# frontier-intake

Client-side intake pipeline for the math breakthrough register: validated
register entries become `x/knowledge` claims, panel-verified into facts on the
live zerone networks. Shells the `zeroned` CLI; contains no consensus code and
no key custody. Normative context: `docs/specs/math-frontier-absorption-v0.md`.

```
bun intake.ts validate                       # register schema + admission rules (no network)
bun intake.ts plan   --network zerone-testnet-1                 # free novelty/fee preview
bun intake.ts setup  --network zerone-testnet-1 --live-ack=zerone-testnet-1
bun intake.ts submit --network zerone-testnet-1 --live-ack=zerone-testnet-1 --batch
bun intake.ts panel  --network zerone-testnet-1 --live-ack=zerone-testnet-1 --watch
bun intake.ts status --network zerone-testnet-1
```

## Release posture

Broadcasting commands (`setup`, `submit`, `panel`) refuse to run without
`--live-ack=<exact-chain-id>` — the explicit live-network authorization this
repo's broadcast-guard discipline requires (cf. `tools/agenttool-relay`,
which refuses live chains entirely; this tool's purpose IS live intake, so
the guard is an explicit per-invocation acknowledgment instead of a refusal).
Live reruns after the 2026-08 drill should revalidate: effective fees and
pacing (`plan`), verifier balance headroom (the gate reads spendable balance
and panel keys drift — never reuse a key with a day job, e.g. the relay's
`relayer`), RPC endpoints, and the phase parameters read live at startup.

## State

Runtime state lives outside the repo at
`~/.zerone-agent/frontier-intake/<network>.state.json` (0600), updated
read-merge-write under an exclusive lock so `submit` and a concurrent
`panel --watch` never drop each other's rounds. `submit` interleaves panel
passes during cooldown waits, so a single process also covers a full batch.

Rounds record honest stages: `submitted`, `pending_inclusion` (broadcast
accepted but inclusion unobserved — backfilled later, never lost),
`resolved_accepted`, `resolved_other`, `missed_window`, `abandoned`, `error`.
Claim statuses are classified numerically per the live enum (ACCEPTED = 6);
in-flight statuses keep waiting and transient RPC failures never produce a
terminal verdict.

## Tests

`bun test` — register validation (admission, disclosure, aggregator-only
sources, refutation wording), commit-hash conformance against the proven
first-truth ceremony recipe, and the pure panel phase logic including
missed-window honesty. Wired into CI as the `frontier-intake` job.
