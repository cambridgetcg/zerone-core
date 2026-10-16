# CLAUDE.md — Project Instructions

## Git Workflow
- Never push to `main`. Work on a branch named `<agent>/<topic>`, push the branch, and open a pull request.
- Merge a pull request only after its CI has finished green. A run still in progress is not a pass.
- GitHub cannot enforce this here: branch protection is unavailable for this private repository on the
  current plan, so the rule holds only because every agent follows it.
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
