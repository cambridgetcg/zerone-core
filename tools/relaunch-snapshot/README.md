# Zerone relaunch snapshot

This read-only tool captures the public supply-bearing inventory at checkpoint
state height `F`. It does not broadcast, export private node state, decide who
is eligible for migration, or sign its output.

The halt contract uses three consecutive heights:

- `F` is the REST inventory and eligibility-input state;
- `A=F+1` is an empty, canonically committed anchor whose signed-header
  `AppHash` commits state `F`;
- `H=A+1` is the SDK halt trigger. Comet stages empty block `H` before
  `FinalizeBlock(H)` fails, so BlockStore/status is at `H` while ABCI remains
  last applied at `A` and no block results exist for `H`.

`/commit?height=A` must report `canonical=true`. At the current BlockStore tip,
`/commit?height=H` returns its subjective seen commit with `canonical=false`
because no `H+1` exists. That flag alone does not prove `H` noncanonical.
Zerone's explicit transition policy makes `A` the final application/history
boundary and `F` the successor-inventory state; staged, application-unapplied
`H` is retained as terminal halt evidence.

Run the v3 tool only against the stable, still-running halted `H/A` source after
transaction, P2P, and public ingress have been fenced, and before manually
stopping the signer/observer:

```bash
go run ./tools/relaunch-snapshot \
  --rpc https://REPLACE_HALTED_RPC \
  --rest https://REPLACE_HALTED_REST \
  --expected-chain-id zerone-1 \
  --checkpoint-state-height REPLACE_F \
  --final-committed-height REPLACE_A \
  --halt-trigger-height REPLACE_H \
  --declared-genesis-sha256 REPLACE_RAW_FILE_SHA256 \
  --out zerone-1-final-inventory-v3.json
```

The tool requires `A=F+1` and `H=A+1`, both `A` and `H` empty, `H` linked to
`A`, canonical commit evidence for `A`, the subjective `canonical=false` seen
commit for terminal tip `H`, ABCI at `A`, results at `A`, no results at `H`, and
stable repeat reads. It pins every REST query to `F`, validates pagination,
owner totals, supply, and bonded validators, and fails if the source changes.

Before fencing and deleting staged `H` from any serving copy, also preserve the
exact raw responses for `/status`, `/block A`, `/commit A`, `/block H`,
`/commit H`, `/abci_info`, and the missing `/block_results H` response. Hash
those files in an immutable manifest and sign that manifest with the v3
inventory. The compact v3 JSON records boundary facts but is not a substitute
for the signed headers and commit signatures in the raw evidence bundle.

After capture, manually stop/fence the signer. On a stopped fresh-key copy,
`zeroned rollback --hard` takes Comet's pending-block special case: it removes
staged `H` while preserving state/application `A`. The restartable public
archive is therefore `A/A` (BlockStore/status `A`, ABCI `A`, `H` absent). This
v3 tool intentionally rejects that sanitized archive and must not be rerun
against it. Verify the archive with the separate `A/A` contract in the cutover
runbook.

Cosmos REST height headers are not Merkle proofs. The output records that trust
model; final publication requires a trusted source, independent signer/observer
comparison, deterministic hashes, and detached signatures. The RPC genesis
hash is a semantic JSON hash, not a replacement for the raw-file SHA-256.

Schemas v1 and v2 are obsolete: they did not encode the observed Comet
BlockStore/application split correctly. The output is inventory, not
entitlement; eligibility remains a separate reviewed and hashed policy output.
