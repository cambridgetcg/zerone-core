# Worktree consolidation inventory — 2026-09-09

Status: **dated source inventory and recommended order; not authorization,
ledger reconciliation, or a readiness/activation decision.**

Observed 2026-09-09, 15:23–15:27 UTC, against GitHub main
[`b0168835d161cff5de209c996d1055648dee4ca7`](https://github.com/cambridgetcg/zerone-core/commit/b0168835d161cff5de209c996d1055648dee4ca7).
The scan read Git metadata, status filenames, ancestry, patch comparisons, and
open-PR metadata. No checkout, index, branch, remote ref, or production state
was changed. Checkout labels below omit private workstation paths.

## Canonical source and misleading local names

The primary repository's `github/main` matched the merged accounting repair at
`b0168835`. Keep that repair as the integration baseline; its
[prospective source status](../specs/accounting-authority-v1.md) does not prove
existing-network adoption or complete the
[accepted authority design](../AUTHORITATIVE-STATE.md).

- `open-standards-main` holds local **`main` at `89553a01`**, 15 commits behind
  GitHub main, with **133 changed/untracked paths**. Its name is not a current
  source reference.
- The primary `zerone` checkout is clean on
  `codex/zerone-2-daemon-readiness` at `d975b22d`, 11 commits ahead and 286 behind
  GitHub main. It belongs to an older relaunch lineage.
- Primary `origin/main` refers to Codeberg at `146cb9b5`, not GitHub main.
  Separate clones also retain older `main` and remote-tracking refs.
- The release checkout preserves exact `b0168835`; the authenticated-ledger
  census branch began from that same commit. Later work needs its own exact pin.

## Inventory counts

| Scope | Existing directories | Dirty | Missing registered paths |
| --- | ---: | ---: | ---: |
| Primary shared repository: 104 registered worktrees | 81 | 7 | 23 |
| Three independent `zerone-core` clones | 3 | 1 | 0 |
| Combined | **84** | **8** | **23** |

The primary repository has 110 local branches. Of the 84 existing directories,
53 are clean ancestors of main, seven more have all individual non-merge
patches represented on main, and 16 other clean directories retain divergent
commits. The remaining eight are dirty. The separate `zerone-truth` site
repository is excluded.

“Clean” covers tracked and nonignored untracked paths only. Ignored runtime
data were not inspected. Missing registrations and patch-equivalent branches
are not deletion recommendations. The metadata inventory alone is not a content
backup; the later bounded source recovery is described below. Active work may
have changed since observation.

## Dirty work requiring preservation

| Checkout label | Changed/untracked paths | Preservation concern |
| --- | ---: | --- |
| `open-standards-main` | 133 | Native/MVP work on stale local `main`. |
| `mvp-node-first-beta-20260908` | 127 | Mixed staged and unstaged native/MVP work. |
| `zerone-release-20260908` | 121 | Separate clone at PR #66's published head; additional local work. |
| `karma-life-sciences` | 72 | Includes 11 unresolved index conflicts. |
| `quantum-karma-season1` | 74 | Includes governance/vesting source and schema changes. |
| `relational-geometry-v0` | 11 | Dashboard, edge, and specification changes. |
| `constructive-intelligence-observatory` | 7 | Dashboard source and tests. |
| `zerone-supabase-observatory-v01` | 6 | Untracked observatory source files. |

Dirty local main and node-first MVP share **122 changed path names**. The
separate PR #66 clone shares 70 with each. Shared names do not establish equal
contents: compare all three snapshots, including their indexes, before
choosing a final delta. A single `git worktree list` misses the separate clone.

## Follow-up content comparison and preservation

At 15:37 UTC, a read-only comparison covered 189 selected source paths,
including corresponding HEAD, index stages, working files, and deletion states.
It found 116 identical working files among the 122 shared main/node-first paths.

- Of dirty local main's 133 changed versions, 130 match published PR #66
  `abdc0fad` in bytes and mode. Its remaining CI workflow and two hash manifests
  match node-first's stage 0. Nine Fold-to-Fire/ToK files also match `b0168835`.
  Reapplying the whole dirty tree would duplicate represented work.
- Node-first retains 11 working versions that differ from its index, plus
  three staged CI/hash variants that differ from both compared commits.
  Preserve its distinct staged and working CI versions separately.
- The PR #66 clone retains 119 working versions that differ from both compared
  commits; two deletion states already match absence on current main. None of
  its 70 shared dirty paths matches either other tree's working bytes. Review
  recovery/source-binding maintenance separately from its scheduler, native
  ingress, and SDK feature changes; rebuild derived artifacts from final source.

A subsequent private recovery capture preserved scoped source in all eight
dirty trees: 736 tree/path records, raw indexes, selected HEAD/index blobs,
working bytes and modes, deletions, binary patches, and all 11 conflicted paths'
index stages. Payload hashes, private permissions, and source/index/HEAD/status
before-after equality passed. One credential-like example was excluded without
reading its contents; ignored/runtime and unselected files remain outside this
bounded backup. This preserves work for review; no source was applied.

## Published candidates and source conflicts

| Draft PR | Head | Ahead / behind main | Changed files | Files also changed by the accounting repair |
| --- | --- | --- | ---: | ---: |
| [#66: native scheduler MVP](https://github.com/cambridgetcg/zerone-core/pull/66) | `abdc0fad` | 1 / 11 | 122 | 19 |
| [#69: signed ToK feedback and correction](https://github.com/cambridgetcg/zerone-core/pull/69) | `58e5bfc3` | 2 / 7 | 170 | 36 |

GitHub reported both PRs as having merge conflicts. Shared paths include
application/upgrade wiring, generated SDK artifacts, and source-bound authority
artifacts. These overlaps require semantic review as well as conflict
resolution; path counts are not defect counts. No #69 head was found among the
bounded local checkout inventory. Its published commits remain on GitHub.

## Older commits already represented on main

| Older commit | Integrated commit | Evidence |
| --- | --- | --- |
| `171e346c` — custom-staking census | `7ee0e442` | `git cherry`: equivalent patch. |
| `30ca0f34` — census evidence binding | `357975e3` | `git range-diff`: surrounding monitoring context differs; the binding change is represented. |
| `2bcde21a` — census verification hardening | `57f14f1a` | `git cherry`: equivalent patch. |
| `609df8ba` — consensus rehearsal | `beabc2b3` | `git range-diff`: equal patch. |
| `4ee8241e` — seed I/O tooling | `557b37f7` | `git cherry`: equivalent patch; the old branch also has a merge commit. |

Do not reapply these commits merely because their original hashes are absent
from main's ancestry. Main contains subsequent census authentication and the
accounting repair. Patch comparison identifies duplicated work; it does not
prove that every later behavior or historical release boundary is equivalent.

## Recommended preservation and integration order

1. **Keep the current repair.** Continue ledger census and discrepancy work
   from an exact current-main baseline. Determine the actual ledger/source
   relationship before selecting a migration or native candidate.
2. **Recover and compare every dirty tree before cleanup.** Preserve staged,
   unstaged, untracked, and conflicted work. Reconcile the three native/MVP
   snapshots as one workstream; do not stack them blindly or overwrite local
   `main` to make its name current.
3. **Separate necessary maintenance from feature expansion.** Extract any
   evidenced accounting, recovery, source-binding, or operational correction
   into bounded changes. Review the remaining #66 and #69 features separately,
   retaining their activation boundaries and the merged repair. Regenerate
   shared SDK/source artifacts from the final reviewed source.
4. **Preserve historical exact pins.** H1 source `65c19cd8`, replacement H2
   source `36728afb`, older founder source `4bffb6d2`, relaunch source refs, and
   deployed release snapshots have lineage purposes. Divergence alone does
   not make them ordinary merge candidates or obsolete evidence.
   Abbreviations here are labels; retain full source and executable identities
   in their release records.
5. **Defer unrelated expansion and organizational cleanup.** Keep the
   [#57 agent-economy draft](https://github.com/cambridgetcg/zerone-core/pull/57)
   separate from custody reconciliation. Record integrated equivalents first;
   consider retiring redundant working directories only after each dirty
   tree, ignored-data dependency, and historical pin has a preserved home.

No merge, pruning, branch movement, activation, or new approval is recorded by
this report. No tests were run: the report changes documentation only.
