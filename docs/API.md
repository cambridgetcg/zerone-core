# Zerone API reference

Zerone exposes gRPC, gRPC-Gateway REST, and CometBFT RPC. The generated
protobuf surface is authoritative; this file explains how to discover and use
it without duplicating a manual endpoint table.

## Canonical inventory

The checked-in Swagger 2.0 document is
[`docs/swagger-ui/swagger.json`](swagger-ui/swagger.json). At this source
revision it contains:

- 217 REST paths; and
- 446 schema definitions.

Those counts cover standard Cosmos APIs and the 23 custom Zerone modules.
Transaction generation separately covers 169 request message types across 20
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

## Source identity

The canonical public repository is
[`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
The current Go module/import path remains
`github.com/zerone-chain/zerone` pending a deliberate module-path migration.
Do not derive the source repository URL from that historical import path.
