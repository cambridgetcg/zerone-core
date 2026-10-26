# IBC rate-limit refund identity

## Activation contract (PR #127)

This is a prospective consensus-changing repair, selected only by
`Keeper.RefundIdentityEnabled(ctx)` from `keeper/refund_identity_activation.go`.
The foundation (#130) owns that reader, its enable helper and the single marker
at `0x04` in the retained `zerone_ibcrl` store. There is no second selector;
`Params.Enabled` remains the ordinary limiter switch, not an activation gate.

Each message, packet interception/terminal callback and BeginBlock reads the
marker once before writes and passes its decision into the mutable helpers.
The shared reader uses a fresh infinite gas meter. An absent marker selects
frozen predecessor bodies in `*refund_identity_legacy.go` (source
`77521a0daf14420836483ae9d8ea3fbcc1ae0d40`), preserving their state writes,
events, errors and gas, including ignored storage errors and naive refunds.
A marker fault wraps `activation.ErrMarkerFault` and propagates as an error.
`OnRecvPacket` has no Go error result, so that interface must panic on a marker
fault rather than send an error acknowledgement; genesis/export also panic.
Ordinary receive-quota errors still produce the old error acknowledgement.
Helpers never re-read a marker. Deprecated raw keeper APIs are for frozen
legacy dispatch/tests only, not a way to update activated rules or refund packets.

Native `DefaultGenesis` enables the repair. Old genesis without the field, or
pre-carrier test fixtures setting `RefundIdentityEnabled = false`, stay off.
The named `consensus-repairs-v1` carrier must enable it at height H, own the
ibcratelimit consensus version 1 -> 2 bump/migration, exact source/target version
map checks, registry/lineage parity and version-list updates. This lane does not
register that carrier or change `ConsensusVersion`. Merging this repair is
hard-gated on the carrier lining up and independent review, not merely #130.

Use an in-place, coordinated named x/upgrade, never a rolling update. The binary
uses ibc-go v10 and must follow H3 (`sdk-0.53/ibc-10`), not jump directly from
live-source `2e37c4c`. Do not roll back the old binary onto new-format state: its
JSON reader ignores extra fields and restores naive reversal.

## Retained state and conservative legacy policy

- A debit receipt binds channel, sequence, denom, amount, window start and rule
  generation. Disabled/unconfigured sends create no receipt. Only the original
  generation and still-unexpired window may reduce quota.
- An old rule with no generation key reads as generation 0. New charges on it
  explicitly persist 0. Remove/re-add advances to 1, even at the same height
  with identical policy. Retain generation tombstones while receipts may exist.
- Old packet flows lack both identity fields. Never backfill them from current
  counters. Terminal callbacks clean them up and still forward to the transfer
  app, but never reduce current quota after activation. A genuinely charged old
  packet therefore temporarily holds capacity on its tuple until window reset;
  token refunds are not withheld. Malformed generation metadata likewise skips
  quota reversal, while charge/install paths still fail closed.
- No clawback, guessed counter reconstruction, historical transfer reversal or
  automatic redistribution. Previously erased counters and historical over-cap
  transfers are not reconstructible from these receipts.
- Export already omits packet flows and also omits generation keys. On a fresh
  import of an enabled export, rules start at generation 1 and no receipts
  exist. Marker-off imports create no generation. Export/reimport is not the
  activation path and does not promise complete outstanding IBC continuity.

The carrier must take and surface a preactivation census: retained rules,
window starts/lengths and counters; outstanding packet-flow counts/shapes per
tuple (including missing/corrupt identity); any unexpected occupancy under
`0x03`; and corrupt generations. Do not equate old recorded amounts with actual
charges. Flag overflow-prone window bounds and hold activation for disposition,
rather than normalizing policy or redistributing usage automatically. Rehearse a
copy with outstanding legacy flows, verify token refunds through the real
transfer app, preserve usage, and verify new generation-0 receipts. Record Yu's
explicit acknowledgement of temporary capacity retention and unreconstructed
counters before H. The invitation supplies the delegated conservative policy;
it is not a live-state census or deployment approval. No live chain was queried.

## Explicit persisted-schema exception for this PR

The review's documented-exception option is used for the pre-existing Go-only
storage addition in PR #127. The protobuf `PacketFlow` remains its original four
fields; the internal JSON charge record extends its stored representation.
Generated files are not hand-edited. Offline `make proto-gen` cannot run its
remote buf generators here, and this lane does not widen the wire schema.
This exception is deliberately recorded here and in the PR, not hidden in a Go
struct; independent review must accept it. A future schema-alignment change
would declare proto3 optional uint64 fields 5 (`window_start`) and 6
(`rule_generation`) and regenerate, preserving zero-vs-absence and these exact
stored bytes. The generation counter below is a raw primitive, not a serialized
protobuf message; converting it to protobuf would require its own migration.

Stored key/value contract:

| Key | Value |
| --- | --- |
| `0x00` | Existing Params JSON (unchanged) |
| `0x01 \| channel \| 0x00 \| denom` | Existing RateLimit JSON (unchanged) |
| `0x02 \| channel \| 0x00 \| sequence(uint64 BE)` | Original flow JSON off; charge-record JSON on |
| `0x03 \| channel \| 0x00 \| denom` | Exactly 8 bytes, uint64 big-endian generation; absent = 0 |
| `0x04` | Foundation activation marker: absent = off, exactly `{1}` = on, anything else = fault |

Activated charge-record JSON fields: `channel_id` (string), `sequence` (uint64),
`denom` (string), `amount` (base-10 positive integer string), `window_start`
(explicit uint64, including 0), `rule_generation` (explicit uint64, including 0).
Missing or null identity fields never authorize a reversal. Channel/sequence
must match the storage key. No other receipt format or fallback is inferred.

`SetRateLimitForActivation` advances generation on installation or byte-for-byte
change of MaxSend, MaxRecv or WindowBlocks, not on counter/window updates.
Normalizing numeric spelling (e.g. `100` -> `0100`) counts as replacement and
invalidates old receipts. A future MsgUpdateRateLimit must preserve/review this
rule. Exhaustion/corrupt generation cannot silently wrap.

## Review limits and gas

The independent review reported keeper-only charge gas 8,887 -> 11,400 (+2,513).
The activated path reads the rule for policy comparison and the generation,
and writes a larger receipt; review the whole MsgTransfer path in the carrier.
`TestRefundIdentityMiddlewareChargeGas` records a separate reproducible
middleware measurement using synthetic transport and a retained generation-0
rule; it is not a whole-transaction/live-node gas benchmark. Marker-off parity
tests compare gas, bytes, write traces, events, results and errors with the frozen
predecessor, not with that activated measurement.

Immediate send-error reversal is a keeper-contract fix only: a production SDK
message error already discards its transaction cache. The redundant fixed-path
middleware reversal wrapper is removed. Raw reversal/receipt-discarding APIs
are deprecated. Tests pin corruption of channel binding with a charged second
channel, so removing the channel check no longer survives the suite.

The pre-existing uint64 addition overflow in send/receive/BeginBlock window
reset remains a separate follow-up; it is intentionally not bundled into this
activation. ReverseSendCharge uses subtraction, but that does not cure the old
reset bug. No new event is introduced: hold-and-surface here means retaining
quota conservatively and surfacing the legacy-state census/capacity trade-off
for the carrier decision, not silently rewriting or redistributing counters.
