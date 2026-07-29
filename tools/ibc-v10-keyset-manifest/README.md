# IBC v10 legacy keyset manifest

This offline tool reads a **disposable copy of a halted node's application
database** and emits the canonical `Plan.Info` commitment required by Zerone's
`sdk-0.53-ibc-10` upgrade handler.

It exists because IBC-Go v10.7.0 deletes the bare
`channelUpgrades` and `pruningSequenceStart` keys during migration, while the
IBC-Go v8 records are children below slash-terminated prefixes. Zerone's
upgrade handler removes the children only after its committed IBC view matches
this independently collected manifest.

## Safety boundary

Never run this command against a validator's live or only database. Halt the
old binary, make a filesystem-level copy of the whole node home while it is
halted, then restart the old binary only after the copy is complete.

The command requires `--copied-db` as an explicit operator attestation and
opens supported backends in read-only mode:

- GoLevelDB uses goleveldb's `ReadOnly` option.
- Pebble uses Pebble's `ReadOnly` option, which disables writes, WAL flushes,
  and background compactions.

The tool never calls a database write API. The copy remains mandatory: it
provides a stable point-in-time source, prevents races with a running process,
and contains the operational consequences of backend/version incompatibility.
The command rejects Zerone's default `~/.zeroned` home, every descendant of
that home (including localnet validator homes), and symlinks at the supplied
home, `data`, or `data/application.db` boundary. A custom live home cannot be
identified mechanically; the `--copied-db` attestation is a hard operational
boundary, not a claim that the tool can detect a dishonest path.

## Evidence required

The operator must supply:

- the copied node home and its exact `goleveldb` or `pebbledb` backend;
- the trusted positive application height; and
- the trusted 32-byte **post-commit root** for that application height.

Cosmos/Comet height semantics matter here. Root record `s/<H>` is application
state after commit `H`. Use an independently recorded ABCI `/abci_info`
response whose `last_block_height` is `H` and whose `last_block_app_hash` is
the supplied hash. A light-client-verified block header at `H+1` also commits
state `H` when such a header exists. Do **not** use the `app_hash` carried by a
block header or status record at height `H`; that field commits state `H-1`.

The tool verifies all of the following before emitting anything:

1. root `s/latest` is canonical and equals `--expected-height`;
2. root `s/<height>` decodes as matching `CommitInfo`;
3. the computed multistore root equals `--expected-app-hash`;
4. the commit contains exactly one `ibc` store committed at that height;
5. `s/k:ibc/mstorage_version` is exactly `1.1.0-<height>`, proving the IBC
   IAVL fast index claims synchronization with the copied root; and
6. those three root/metadata records are byte-identical after the scan.

Obtain the ABCI height/hash evidence through the validator ceremony's
independently recorded halted-height checkpoint. Do not derive the "trusted"
arguments from the same unreviewed copy alone.

## Usage

```sh
go run ./tools/ibc-v10-keyset-manifest \
  --home /absolute/path/to/halted-node-copy \
  --backend goleveldb \
  --expected-height 123456 \
  --expected-app-hash 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --copied-db
```

On success, stdout contains one compact JSON line and nothing else:

```json
{"schema":"zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1","channel_upgrades":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"pruning_sequence_start":{"key_count":"0","keys_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}}
```

Copy that exact line into the governance upgrade plan's `Info` field. Height
and app hash are deliberately verified as external evidence and are not added
to the consensus schema.

## Exact raw scan

The census reads only these physical fast-node domains:

- `s/k:ibc/fchannelUpgrades/`
- `s/k:ibc/fpruningSequenceStart/`

It strips only the physical `s/k:ibc/f` prefix, leaving each complete logical
key (including `channelUpgrades/` or `pruningSequenceStart/`). Each fast-node
value must have canonical IAVL encoding and a last-update version from `1`
through the copied root height.

For each domain, logical keys are byte-lexicographically sorted and committed
as:

```text
SHA256(concat(uint64-BE(len(key)) || key))
```

The empty-vector digest is
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

The tool and handler share the same consensus resource limits: at most 100,000
keys and 32 MiB of aggregate logical key bytes per domain, and at most 2,048
bytes of canonical `Plan.Info`. The raw reader additionally caps individual
physical keys at 64 KiB, fast-node values at 64 MiB, total scanned key/value
input at 1 GiB, root commit metadata at 16 MiB, and mounted stores at 4,096.
Iterator traversal and close errors both fail the command.

## What this does not prove

The IAVL storage-version marker is the library's synchronization claim; this
tool does not rebuild the full IAVL tree from historical nodes. Run the normal
IBC export census, bilateral packet-clearance procedure, and full old-database
upgrade/restart rehearsal as separate gates. A successful manifest is not
authorization to deploy.
