# Zerone API reference

Zerone exposes gRPC, gRPC-Gateway REST, and CometBFT RPC. The generated
protobuf surface is authoritative; this file explains how to discover and use
it without duplicating a manual endpoint table.

## Canonical inventory

The checked-in Swagger 2.0 document is
[`docs/swagger-ui/swagger.json`](swagger-ui/swagger.json). At this source
revision it contains:

- 218 REST paths; and
- 449 schema definitions.

Those counts cover standard Cosmos APIs and the 23 custom Zerone modules.
Transaction generation separately covers 170 request message types across 20
Zerone `Msg` services.

Regenerate and verify the document with:

```bash
make proto-swagger-gen
make proto-check

jq '.paths | length' docs/swagger-ui/swagger.json
jq '.definitions | length' docs/swagger-ui/swagger.json
```

If this prose and the generated document differ, the generated document and
its protobuf sources win.

## Interfaces

| Interface | Default port | Purpose |
|---|---:|---|
| REST / gRPC-Gateway | 1317 | JSON queries and enabled transaction routes |
| gRPC | 9090 | Native protobuf services |
| CometBFT RPC | 26657 | Blocks, transactions, status, and consensus data |

An operator may bind these services to different addresses or keep them
private. A route present in Swagger is not a promise that a particular public
node exposes it.

Enable local REST, Swagger UI, and gRPC in
`~/.zeroned/config/app.toml`:

```toml
[api]
enable = true
swagger = true
address = "tcp://127.0.0.1:1317"

[grpc]
enable = true
address = "127.0.0.1:9090"
```

The UI is then available at `http://127.0.0.1:1317/swagger/`.

## Discovery

List gRPC services and methods:

```bash
grpcurl -plaintext 127.0.0.1:9090 list
grpcurl -plaintext 127.0.0.1:9090 list zerone.knowledge.v1.Query
```

List generated REST paths:

```bash
jq -r '.paths | keys[]' docs/swagger-ui/swagger.json
```

Representative read-only queries:

```bash
curl --fail-with-body \
  http://127.0.0.1:1317/zerone/auth/v1/account_identifier/zrn1...

curl --fail-with-body \
  http://127.0.0.1:1317/zerone/training_provenance/v1/in-toto/<manifest-id>

grpcurl -plaintext 127.0.0.1:9090 \
  zerone.knowledge.v1.Query/Params
```

CAIP and in-toto responses are computed projections. Their limitations are
documented in
[`docs/standards/OPEN_CRYPTO_SDK.md`](standards/OPEN_CRYPTO_SDK.md).

## Transactions

Use `zeroned tx <module> <command>` or a protobuf/direct-signing client. A
generic REST broadcast uses the standard Cosmos endpoint:

```bash
curl --fail-with-body -X POST \
  http://127.0.0.1:1317/cosmos/tx/v1beta1/txs \
  -H 'Content-Type: application/json' \
  --data-binary @signed_tx.json
```

Generated codecs serialize messages; they do not grant authority, choose fees,
or prove that a message is safe for a particular network. Follow the active
chain's signed release and governance policy.

## Signed fact use (source candidate)

**Not activated.** `tok-feedback-v1` is source-only here. Production remains
`NO_GO` pending independent custody evidence, exact H1→H2→H3 lineage, the
separate feedback upgrade and release/rehearsal gates. Source presence, a local
commitment test, and broadcast acknowledgement establish none of those gates.

The new `MsgReportFactUse { consumer, fact_id }` requires `consumer`'s SDK
transaction signature. It records exactly one **self-reported** use; it does not
measure readership, establish usefulness, pay a bounty, or mint currency.
Both reports and ratings require registered, unfrozen Zerone accounts and the
separate admitted consumer cohort. Consensus handlers validate canonical account
spelling, bounded IDs/memos and current state, independently of CLI preflight.

Defaults and immutable ceilings:

- disabled, empty cohort (at most 32 accounts);
- at most 100 reports/consumer/epoch and 1,000 globally (governance can lower);
- one report and one rating per `(epoch, consumer, fact_id)`; fresh transaction
  sequences do not bypass semantic deduplication;
- epoch is `floor(height / fitness_epoch_blocks)`; rating is allowed only in
  the report's epoch, whereas the marker expires at `(epoch + 2) * blocks`;
- at most 2,000 retained markers and 100 **examined** records pruned per block;
- epoch length is frozen while enabled or any markers remain. Once any report
  commits, `EverReported` never clears, including on disable/prune: query and
  satisfaction fitness weights and query-derived energy remain zero until a
  separately reviewed future upgrade, not a later parameter toggle.

Existing-wallet commands, for a separately authorized local/test cohort only:

```bash
# Replace every placeholder and simulate gas against the intended release.
zeroned tx knowledge report-fact-use FACT_ID --from EXISTING_KEY \
  --chain-id LOCAL_CHAIN --node LOCAL_RPC --gas auto --gas-adjustment 1.5 \
  --gas-prices 1uzrn
# Verify transaction commitment before querying/rating.
zeroned query knowledge fact-use-receipt CONSUMER FACT_ID --node LOCAL_RPC
zeroned tx knowledge rate-fact FACT_ID true 'public reason' --from EXISTING_KEY \
  --chain-id LOCAL_CHAIN --node LOCAL_RPC --gas auto --gas-adjustment 1.5 \
  --gas-prices 1uzrn
```

The optional rating memo is public UTF-8, at most 256 **bytes**, not characters.
Reports and ratings are public, paid transactions. Declared gas limits and fees
must cover actual state-dependent work: the 22,222 generic minimum and message
admission floors are **not estimates**. The SDK normally retains ante fees and
account-sequence increments when a validly signed message fails; module atomicity
does not mean the sender's entire account balance stays unchanged. An invalid
signature fails ante and is a different case.

`Query/FactUseReceipt` accepts `{ consumer, fact_id }` and returns
`{ receipt, found, epoch, snapshot_block_height }` for the **query context's
current epoch only**. `found=false` is not a claim that no historical use exists.
Version-1 receipts retain use/expiry heights and rating/height. `UNSPECIFIED=0`
is invalid, `UNRATED=1`, `USEFUL=2`, `NOT_USEFUL=3`. Querying creates no receipt.
`Fact.track_query` and `querier` remain wire-compatible but cause no writes.
Historical `query_count`/satisfaction fields retain their prior provenance;
only newly committed self-reports increment the new use flow. No missing
historical readership is backfilled, and `ReportDemand` is unchanged.

The repository SDK's existing `knowledgeMessages.withTypeUrl.reportFactUse`
and `.rateFact` composers use generated codecs in `createZeroneRegistry`.
Use the existing wallet's direct signer and verify the committed result; no
backend user-signer or new wallet is provided. See
[the contract](specs/tok-feedback-v1.md) for the correction and durability lanes.

## Source identity

The canonical public repository is
[`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
The current Go module/import path remains
`github.com/zerone-chain/zerone` pending a deliberate module-path migration.
Do not derive the source repository URL from that historical import path.
