# Offline consensus-repairs-v1 census

Read-only GoLevelDB census over a separately acquired stopped-node copy. It
shares the application's bounded census; it does not import genesis, open keys,
contact a node, schedule an upgrade or repair a finding.

    GOFLAGS='-mod=readonly -p=4' GOPROXY=off go build -buildvcs=false -trimpath -o <output>/consensus-repairs-census ./tools/consensus-repairs-census
    <output>/consensus-repairs-census --copied-db --home <absolute-copy-home> --chain-id zerone-dev-1 --expected-height <committed-height> --expected-app-hash <lowercase-64-hex>

The tuple must come from independent evidence: the AppHash committed at that
height (`last_block_app_hash`), not the pre-state hash in that block's header.
Both LastCommitID and WorkingHash must match. WorkingHash binds the loaded
per-store roots to the AppHash; it is not a full Merkle re-verification of the
stored tree or each leaf value read by the census. Chain ID remains
an explicitly labelled operator assertion. Only
`data/application.db` is opened, with `ReadOnly` and `ErrorIfMissing`. Symlink
path components and protected default/private home names are refused. The
caller remains responsible for selecting a disposable copy, not a running or
only database. This command supports GoLevelDB, not Pebble or a JSON export.

The copy must already contain a regular, non-symlink `data/application.db/LOCK`.
GoLevelDB creates a missing LOCK even in ReadOnly mode; the command refuses
before opening in that case. It never creates the missing file for the operator.

Use a committed height at most H-1 for preactivation. Markers on a copy at/after
H cause refusal. The offline census (like startup) evaluates committed height;
the handler receipt evaluates H. JSON is therefore not directly byte-equal:
height differs, and StartBlock, PhaseStartedAtBlock or rollback activation at
the boundary can change findings even when the raw-state digest is identical.

Exit 0 means a complete census without blockers. Exit 2 emits the JSON report
with blockers. Exit 1 means argument, identity, open/load/close/output failure.
A pass is not source-lineage validation or scheduling authority. A blocker is
never silently truncated or automatically repaired.

Each named census has a 100,000-record/64 MiB budget over only its selected
prefixes/singletons. The complete image is hashed as a stream, with record and
byte counts per store and first-byte prefix; `census_reads` also gives each
census's counts per selected prefix/exact key. Only selected rows are retained.
Unrelated alignment observations, SDK staking history and bank accounts do not
consume these budgets. Q1 blocks only zero active stake AND zero mint ledger
AND positive bank supply. Other valid readings are held, not rejected merely
because the registry is empty. Reports include both staking ratios, the domain
status histogram, NUL counts and each pending rollback's current-state forecast.
The permanent receipt stores one fixed-shape record/byte count and digest
summary for every census, plus held-finding and pending-rollback counts. Detailed
findings and the full status histogram stay in this offline JSON; rollback
forecasts appear once, in `pending_rollbacks`, not again in `findings`.
Histories and packet flows retain their count/digest summaries.
Read the full
[carrier contract and unresolved prerequisites](../../docs/consensus-repairs-v1.md).

Tests create only ephemeral synthetic databases and compare every source file
before/after passing and failing scans, including a present-but-empty marker,
working-root mismatch and missing/symlink LOCK. Test roots resolve macOS's `/var`
symlink; production input paths still reject symlink components:

    GOFLAGS='-mod=readonly -p=4' GOPROXY=off go test -p 4 ./tools/consensus-repairs-census -count=1
