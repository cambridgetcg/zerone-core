# R30 — Hygiene: Proto-Go Consistency

**Goal:** Eliminate the class of bugs where proto definitions and generated Go code drift apart, causing silent field drops in JSON serialization.

## Background

R28 and R29 introduced ~30 new params and several new query RPCs across 8 modules. Multiple sessions hit the same bug pattern:
- New field added to `.proto` but `make proto-gen` not run → stale rawDesc → `protojson` drops the field
- New field added to Go struct but not to `.proto` → field lost on next proto regen
- New query RPC hand-rolled as `query_ext.go` → app panics on init (`cannot find method descriptor`)

This is a systemic issue. One prompt to fix it and prevent recurrence.

## Sessions (1)

| # | File | Scope |
|---|------|-------|
| R30-1 | R30-1-proto-consistency.md | Full audit, CI enforcement, documentation |

## Run Order

Single session. Run after R29 is complete.

## Known Issues Already Fixed

These were caught during R28/R29 review and fixed manually:
- `capture_defense/types/query_ext.go` — FlaggedDomains (R28-8)
- `capture_challenge/types/query_ext.go` — ActiveChallenges (R28-8)
- `knowledge` MetabolismStatus query — never in proto (R28-4)
- `alignment` correction confidence params — never in proto (R29-4)
- All module rawDesc stale after R29 param additions

R30-1 verifies no more remain and adds automation to prevent recurrence.
