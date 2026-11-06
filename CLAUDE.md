# CLAUDE.md — Project Instructions

## Git Workflow
- Never push to `main`. Work on a branch named `<agent>/<topic>`, push the branch, and open a pull request.
- Merge a pull request only after its CI has finished green (a run still in progress is not a pass), and
  answer its open review comments first.
- Since the repository became public again on 28 October 2026, GitHub reports an active ruleset on
  `main`: changes arrive by pull request, 13 named CI checks must pass on an up-to-date branch, and
  force-pushes and deletion are refused. The ruleset dates from 1 August 2026, but GitHub Free gives a
  private repository no branch protection, so it did not apply while the repository was private (from
  about 25 September). It requires no approving review or resolved threads, and its bypass list is not
  publicly visible, so the rest of this rule holds only because every agent follows it. `AGENTS.md`
  carries the same rule for agents that read it instead of this file.
- If your host has the agenttool-collab coordination board, start your session at the checkout or linked
  worktree you actually edit (not your home directory), claim the paths before editing, and release them
  when you finish.
- Write clear, descriptive commit messages.
- Commit frequently at logical checkpoints (e.g., after each batch/task).

## Proto-Go Consistency Rule
When adding new fields to any module's Params, state types, or query messages:
1. Add the field to the `.proto` file FIRST
2. Run `make proto-gen`
3. Reference the generated type in your Go code
4. NEVER add fields directly to `*.pb.go` — they will be overwritten
5. NEVER create `query_ext.go` for new query RPCs — add them to `query.proto`
6. Run `make proto-check` before committing proto-related changes
