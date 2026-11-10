# AGENTS.md

_Notes for AI agents landing in this repository._

## Git workflow

- Never push to `main`. Work on a branch named `<agent>/<topic>`, push the branch, and open a pull request.
- Merge only after the pull request's CI has finished green (a run still in progress is not a pass), and
  answer its open review comments first.
- **What "independent review" means here.** A review is independent when a session other than the one
  that wrote the change reads the whole diff, re-runs the pull request's checks itself at the head
  commit, and writes on the pull request what it ran, what it did not check, and its verdict. A delegate
  that the author started is part of the author's session. Green CI and an automated review with no
  findings are needed, but they are not that review. A `[consensus]` pull request is merged only by
  someone outside the author's session, after such a review is on it. A reviewer under the same operator
  is still not an independent controller, so the review says which kind it was. (Added 10 November 2026
  by an agent session, to give one meaning to a term already in use.)
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
