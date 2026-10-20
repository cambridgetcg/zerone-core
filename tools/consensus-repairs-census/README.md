# Offline consensus-repairs-v1 census

Read-only GoLevelDB census over a separately acquired stopped-node copy. It
shares the application's bounded census; it does not import genesis, open keys,
contact a node, schedule an upgrade or repair a finding.

    GOFLAGS='-mod=readonly -p=4' GOPROXY=off go build -buildvcs=false -trimpath -o <output>/consensus-repairs-census ./tools/consensus-repairs-census
    <output>/consensus-repairs-census --copied-db --home <absolute-copy-home> --chain-id zerone-dev-1 --expected-height <height> --expected-app-hash <lowercase-64-hex>

The tuple must come from independent evidence. The DB verifies height/AppHash;
chain ID remains an explicitly labelled operator assertion. Only
`data/application.db` is opened, with `ReadOnly` and `ErrorIfMissing`. Symlink
path components and protected default/private home names are refused. The
caller remains responsible for selecting a disposable copy, not a running or
only database. This command supports GoLevelDB, not Pebble or a JSON export.

Exit 0 means a complete census without blockers. Exit 2 emits the JSON report
with blockers. Exit 1 means argument, identity, open/load/close/output failure.
A pass is not source-lineage validation or scheduling authority. A blocker is
never silently truncated or automatically repaired.

The global complete-image budget is 100,000 records/64 MiB with activation
receipt headroom. An empty custom-staking registry blocks while Q1 neutral
not-observed pacing is missing. Read the full
[carrier contract and unresolved prerequisites](../../docs/consensus-repairs-v1.md).

Tests create only ephemeral synthetic databases and compare every source file
before/after passing and failing scans, including a present-but-empty marker:

    GOFLAGS='-mod=readonly -p=4' GOPROXY=off go test -p 4 ./tools/consensus-repairs-census -count=1
