# Zerone identity census

`identity-census` is a read-only, offline preflight for an eventual
`zerone_auth` identity migration. It does not query or mutate a live node, and
it makes no consensus or protobuf changes.

Use a fresh export so the Cosmos `auth` and Zerone `zerone_auth` modules are
audited from the same height:

```sh
zeroned export --height <height> > export.json
go run ./tools/identity-census --input export.json
```

For automation, JSON output and strict warning handling are available:

```sh
go run ./tools/identity-census \
  --input export.json \
  --format json \
  --fail-on warning > identity-census.json
```

`--fail-on error` is the default. Exit status is `0` when the selected
threshold is clear, `1` when findings reach it, and `2` for invalid input or
tool usage. The tool never writes to chain state.

## Checks

The census reports:

- uppercase and 64-hex `did:zrn` forms that alias the proposed lowercase
  32-hex migration identifier;
- distinct DIDs or addresses that collapse to one normalized DID;
- malformed identity, operational, and DID-mapping public keys;
- DID-to-identity-key derivation failures;
- duplicate/orphaned/inconsistent account and DID-mapping records;
- invalid Zerone Bech32 addresses;
- Cosmos Ed25519 and secp256k1 BaseAccount public keys that do not derive the
  stored account address, including keys nested in vesting/module wrappers;
- rotated account evidence whose last-rotation height/history is missing from
  the current genesis export schema.

Unknown Cosmos public-key types are reported as unsupported instead of being
guessed.

## Coverage boundary

A full genesis or `app_state` object is the complete input. A standalone
`zerone_auth` module object is accepted for targeted analysis, but produces
`COSMOS_AUTH_STATE_MISSING` because it cannot prove BaseAccount invariants.

The current Zerone query API only supports lookup by a known address/DID plus a
frozen-account listing. It has no exhaustive account or DID-mapping query, and
it does not expose the last-rotation store. Point queries therefore cannot
produce a complete migration census. Use an export from a trusted node and
record its height/app hash alongside the generated report.

Even a clean report proves only internal snapshot consistency. It cannot prove
private-key possession, authorization-signature validity, historical events,
or data that the current export schema omits.
