# `zerone-testnet-1`

`zerone-testnet-1` is a reserved testnet identifier and planning directory. It
is not currently a published validator-joining packet.

## Stable identity

| Field | Value |
|---|---|
| Chain ID | `zerone-testnet-1` |
| Base denomination | `uzrn` |
| Display denomination | ZRN (`1 ZRN = 1,000,000 uzrn`) |

## Not yet published

This directory does not currently provide a release-bound:

- signed genesis file and SHA-256;
- validator or seed identity;
- public RPC, REST, or gRPC endpoint;
- faucet;
- validator allocation or final parameter set; or
- binary/image digest and provenance packet.

Old IP addresses, GitHub clone URLs, balances, and parameter tables have been
removed because they were not a current signed network contract. Do not use a
placeholder or historical value to initialize a shared validator.

## Source and joining

Canonical source is
[`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
The Go module path remains `github.com/zerone-chain/zerone` pending a separate
module-path migration.

Prepare with the
[`Validator Guide`](../../docs/VALIDATOR-GUIDE.md), but wait for an explicit
network packet containing the exact release commit, genesis bytes, hashes,
peer identities, parameters, and authorization before joining.

The separate `zerone-2` relaunch process is documented in
[`deploy/networks/zerone-2/README.md`](../../deploy/networks/zerone-2/README.md)
and remains NO-GO until its signed ceremony and authority gates are complete.
