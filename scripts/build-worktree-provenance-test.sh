#!/usr/bin/env bash
# Exercise the real Makefile against a tiny module in a nested Git worktree.
set -euo pipefail
export LC_ALL=C LANG=C
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
unset GIT_DIR GIT_WORK_TREE GIT_COMMON_DIR GIT_INDEX_FILE
unset GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/zerone-build-provenance.XXXXXX")
trap 'rm -rf "$SCRATCH"' EXIT HUP INT TERM
PARENT="$SCRATCH/parent checkout"
WORKTREE="$PARENT/.worktrees/release candidate"
mkdir -p "$PARENT/cmd/zeroned"
cp "$ROOT/Makefile" "$PARENT/Makefile"
printf 'module example.invalid/zerone-build-fixture\n\ngo 1.25.0\n' > "$PARENT/go.mod"
printf 'package main\n\nfunc main() {}\n' > "$PARENT/cmd/zeroned/main.go"
printf '/build/\n/.worktrees/\n' > "$PARENT/.gitignore"
git -c init.defaultBranch=fixture init --quiet "$PARENT"
git -C "$PARENT" add .
git -C "$PARENT" -c core.hooksPath=/dev/null -c commit.gpgsign=false \
  -c user.name='Build fixture' -c user.email=fixture@example.invalid \
  commit --quiet -m 'Parent fixture'
PARENT_COMMIT=$(git -C "$PARENT" rev-parse HEAD)
git -C "$PARENT" -c core.hooksPath=/dev/null worktree add --quiet --detach "$WORKTREE" HEAD
printf '\n// The release worktree has its own commit.\n' >> "$WORKTREE/cmd/zeroned/main.go"
git -C "$WORKTREE" add cmd/zeroned/main.go
git -C "$WORKTREE" -c core.hooksPath=/dev/null -c commit.gpgsign=false \
  -c user.name='Build fixture' -c user.email=fixture@example.invalid \
  commit --quiet -m 'Release fixture'
EXPECTED_COMMIT=$(git -C "$WORKTREE" rev-parse HEAD)
test "$EXPECTED_COMMIT" != "$PARENT_COMMIT"

check_metadata() {
  local binary=$1 modified=$2 metadata actual_commit actual_modified
  metadata=$(go version -m "$binary")
  actual_commit=$(printf '%s\n' "$metadata" | sed -n 's/.*vcs\.revision=//p')
  actual_modified=$(printf '%s\n' "$metadata" | sed -n 's/.*vcs\.modified=//p')
  if [ "$actual_commit" != "$EXPECTED_COMMIT" ] || [ "$actual_modified" != "$modified" ]; then
    printf 'FAIL: expected worktree %s modified=%s, got %s modified=%s\n' \
      "$EXPECTED_COMMIT" "$modified" "$actual_commit" "$actual_modified" >&2
    exit 1
  fi
}

make -s -C "$WORKTREE" build-linux-amd64
check_metadata "$WORKTREE/build/zeroned-linux-amd64" false
cp "$WORKTREE/build/zeroned-linux-amd64" "$SCRATCH/first-build"
make -s -C "$WORKTREE" build-linux-amd64
cmp "$SCRATCH/first-build" "$WORKTREE/build/zeroned-linux-amd64"
make -s -C "$WORKTREE" build-linux-arm64
check_metadata "$WORKTREE/build/zeroned-linux-arm64" false
printf '\n// Uncommitted release change.\n' >> "$WORKTREE/cmd/zeroned/main.go"
make -s -C "$WORKTREE" build-linux-amd64
check_metadata "$WORKTREE/build/zeroned-linux-amd64" true
test -z "$(git -C "$PARENT" status --porcelain --untracked-files=all)"
printf 'PASS: clean, dirty, repeated, and cross-architecture builds bind the nested worktree.\n'
