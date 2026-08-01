# Zerone IBC-Go v10 census

`ibc-v10-census` is a read-only, offline census of the IBC-Go v8 state that is
visible in a complete `zeroned export` before the Cosmos SDK 0.53 / IBC-Go v10
upgrade. It never opens a node database, queries a network, or mutates chain
state.

The tool intentionally reports:

```json
{
  "complete": false,
  "upgrade_ready": false
}
```

An export omits consensus-critical old-store keys. No export-only result can
authorize a validator rollout.

## Capture and run

Stop transaction submission and relaying, choose a trusted halted height, and
record that height and its 32-byte application hash from trusted consensus
evidence. Preserve the command transcript. The app hash is not embedded in the
normal export and this tool cannot verify the operator's claim.

```sh
CENSUS_HEIGHT=123456
CENSUS_APP_HASH=0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF

zeroned export --height "$CENSUS_HEIGHT" > export.json

go run ./tools/ibc-v10-census \
  --input export.json \
  --export-height "$CENSUS_HEIGHT" \
  --app-hash "$CENSUS_APP_HASH" \
  --format json \
  --fail-on never > ibc-v10-census.json
```

`--export-height` is a required canonical positive uint64.
`--app-hash` is a required 64-hex representation of the 32 application-hash
bytes. The report also computes lowercase `evidence.input_sha256` over the
exact input bytes, binding the retained report to `export.json`.

The input may be the whole export document or its whole `app_state` object.
Partial module objects are rejected because the census must link `auth`,
`bank`, `feeibc`, and `ibc` state. Input is capped at 128 MiB.

The parser performs multiple bounded in-memory JSON passes to reject duplicate
keys and retain deterministic raw module values. Run it only on a trusted local
export in an offline environment with several times the input size available
as memory; it is not a service endpoint for hostile uploads. Exports above the
cap require a reviewed streaming-tool revision, not a locally raised flag.

## Exit codes and thresholds

- `0`: the selected threshold is clear;
- `1`: findings meet `--fail-on error` or `--fail-on warning`;
- `2`: invalid usage, unreadable input, duplicate JSON keys, schema ambiguity,
  or malformed exported values.

`--fail-on error` is the default. It always refuses an export-only deployment
decision with `OLD_DATABASE_REHEARSAL_REQUIRED`. `--fail-on never` is the only
way to obtain an exploratory report despite findings; it does not change
`complete=false` or `upgrade_ready=false`. Schema errors remain exit `2` even
with `--fail-on never`.

## What is enumerated

The census parses the exact IBC-Go v8.8 export names established by the
maintained protobufs and Zerone's application codec:

- `feeibc.identified_fees`, including every PacketFee and the denom-wise
  `max(recv_fee + ack_fee, timeout_fee)` escrow obligation;
- all `fee_enabled_channels`, `registered_payees`,
  `registered_counterparty_payees`, and `forward_relayers`;
- the bank balance at the deterministic `feeibc` module address;
- every identified channel, its state, ordering, connection hop, negotiated
  version, counterparty, and `upgrade_sequence`;
- exported commitments, acknowledgements, receipts, and next send/receive/ack
  sequences.

Packet-state bytes are represented by length and SHA-256 digest, not copied
verbatim into the report. Arrays and findings are sorted for deterministic
text and JSON output.

The five ICS-29 arrays come from:

- `proto/ibc/applications/fee/v1/genesis.proto`;
- `proto/ibc/applications/fee/v1/fee.proto`.

Channel and packet-state fields come from:

- `proto/ibc/core/channel/v1/genesis.proto`;
- `proto/ibc/core/channel/v1/channel.proto`.

The implementation uses `encoding/json` and local generic report structs; it
does not import IBC-Go v8 generated types, so removing the v8 dependency does
not strand this tool.

## Refusal checks

The report refuses or highlights:

- any of the five ICS-29 arrays remaining nonempty;
- any nonzero balance at the fixed module address, or a mismatch between that
  balance and calculated PacketFee obligations;
- duplicate, missing, or inconsistent named `feeibc` auth ModuleAccounts;
- a lazy missing ModuleAccount when fee records or a fixed-address balance
  already exist;
- ICS-29 channel-version wrappers that need an explicit disposition;
- `STATE_FLUSHING` and `STATE_FLUSHCOMPLETE`, which IBC-Go v10 migration
  rejects;
- nonzero `upgrade_sequence`, because v8 export cannot distinguish historical
  upgrades from omitted current OPEN-state upgrade records;
- channel values that would fail IBC-Go v10 `Channel.ValidateBasic`, including
  state, ordering, one-hop, and ICS-24 identifier rules;
- packet, fee, or sequence records referring to absent exported channels;
- outstanding source-chain commitments;
- malformed local Zerone refund, relayer, or payee addresses;
- malformed protobuf JSON uint64 strings, coin sets, base64 values, unknown
  audited fields, missing required arrays, and duplicate JSON keys.

The deterministic module-address vector is locked by test:

```text
SHA256("feeibc")[:20] = f687822674b15af760d9fc6db730d77dc91361d9
zrn Bech32             = zrn176rcyfn5k9d0wcxel3kmwvxh0hy3xcweksmdhj
```

Cosmos module accounts are lazy. Absence from `auth.accounts` is therefore a
warning when all fee state and the fixed-address balance are empty. When a
named account is present, it must be unique and match the fixed vector exactly.
The bank scan always uses the fixed address.

## Coverage boundary and mandatory next evidence

IBC-Go v8 `feeibc.ExportGenesis` omits its persistent `locked` key, even though
that key records a severe-bug lock condition. Channel export omits upgrade,
counterparty-upgrade, upgrade-error, `pruningSequenceStart`, and
`recvStartSequence` child keys.

The `sdk-0.53-ibc-10` binary closes the `locked` blind spot at consensus
startup: after staging deletion of the dynamically mounted `feeibc` store, its
loader checks the canonical immutable H-1 tree and rejects any `locked` key
presence, regardless of value. A rejection occurs before commit, leaving the
old database restartable for investigation under the v8 binary. This works
with fast nodes enabled or disabled. It supplements rather than expands this
export census.

The IBC-Go v10.7 migration deletes bare upgrade/pruning prefix keys rather than
enumerating their v8 children. A validator-database rehearsal must prove that
obsolete upgrade/pruning children are removed. Conversely,
`recvStartSequence/...` remains replay-protection state in v10 and must be
preserved.

Before an upgrade can be considered ready, separately retain evidence for:

1. a raw old-database key census, including `feeibc/locked`, plus a rehearsal
   proving that the new binary refuses that state without committing;
2. successful old-binary export, new-binary migration, commit, shutdown, and
   restart against a copy of the real validator database;
3. the app-hash-bound
   [`ibc-v10-keyset-manifest`](../ibc-v10-keyset-manifest/README.md), correct
   removal of its obsolete upgrade/pruning children, and preservation of
   `recvStartSequence` and packet KV; the manifest traverses the `ibc` store
   and does not replace the separate `feeibc/locked` check;
4. synchronized-height exports from both ends of every live channel, zero
   commitments on both ends, and a documented relay/transaction freeze;
5. acknowledgement and timeout lifecycles across restart, including packets
   created by the v8 binary;
6. independent verification that the retained export bytes, height, app hash,
   binary hashes, and migration logs belong to the same rehearsal.

Acknowledgements can be retained after processing, and unordered receipts are
replay-prevention state. The census enumerates them as exported packet state;
it does not mislabel either category as an in-flight packet.
