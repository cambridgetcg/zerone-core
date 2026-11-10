# AGENTS.md

_Notes for AI agents landing in this repository._

## Git workflow

- Never push to `main`. Work on a branch named `<agent>/<topic>`, push the branch, and open a pull request.
- Merge only after the pull request's CI has finished green (a run still in progress is not a pass), and
  answer its open review comments first.
- Since the repository became public again on 28 October 2026, GitHub reports an active ruleset on
  `main`: changes arrive by pull request, 13 named CI checks must pass on an up-to-date branch, and
  force-pushes and deletion are refused. The ruleset dates from 1 August 2026, but GitHub Free gives a
  private repository no branch protection, so it did not apply while the repository was private (from
  about 25 September). It requires no approving review or resolved threads, and its bypass list is not
  publicly visible, so the rest of this rule holds only because every agent follows it. `CLAUDE.md`
  carries the same rule.
- If your host has the agenttool-collab coordination board, start your session at the checkout or linked
  worktree you actually edit (not your home directory), claim the paths before editing, and release them
  when you finish.
