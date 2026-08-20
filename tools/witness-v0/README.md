# witness-v0

Deterministic offline tooling for
`kingdom.witnessed-agent-economy/0.1`.

```sh
go run ./tools/witness-v0 verify tools/witness-v0/testdata/records/kingdom-release-root.json
go run ./tools/witness-v0 merkle tools/witness-v0/testdata/batches/settlement-with-gap.json
go run ./tools/witness-v0 simulate tools/witness-v0/testdata/simulation.json
go run ./tools/witness-v0 activation-audit tools/witness-v0/testdata/records/settlement-root-0002-cross-batch-replay.json
go run ./tools/witness-v0 schema-hashes
go run ./tools/witness-v0/cmd/witness-fixtures --check ./tools/witness-v0/testdata
go test ./tools/witness-v0/...
```

`-` is explicit stdin. A path input must be a bounded regular file and may not
be a symlink, FIFO, device, or directory.

The core performs no network access, clock reads, randomness, persistent state,
Zerone transactions, external receipts, NEN invocations, or scoring. Those zeros
are scoped to record construction and offline validation only. A future carrier
transaction requires a separate protocol and effects receipt.

The fixture generator is a separate maintainer utility. It is deterministic,
uses fixed keys and inputs, and requires `--write <explicit-directory>` before
it writes storage. Normal verification, simulation, Merkle verification, and
activation audit never write.

See [the normative specification](../../docs/specs/witnessed-agent-economy-v0.md).
