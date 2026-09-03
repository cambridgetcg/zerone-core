# Zerone identity census

`identity-census` is a read-only, offline preflight for the boundary between
deployed `zerone-1` identity records and the current `zerone_auth` source
contract. It does not query or mutate a live node, migrate records, or make
consensus/protobuf changes.

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

- current-source DIDs that are not exactly
  `did:zrn:{full-64-lowercase-hex-identity-key}`;
- historical 32-hex or uppercase DID forms as explicit legacy warnings only
  when the enclosing full document identifies chain ID `zerone-1`;
- distinct stored DIDs or addresses that resolve to one full identity-key DID,
  plus ambiguous legacy 32-hex prefixes;
- malformed identity, operational, and DID-mapping public keys, including
  non-canonical, off-curve, small-order, and mixed-order Ed25519 points;
- DID-to-identity-key derivation failures;
- absent `operational_key_hash` as either a distinct deployed-`zerone-1`
  legacy warning or a current-source error, and every nonempty malformed or
  mismatched SHA-256 commitment as an error;
- duplicate/orphaned/inconsistent account and DID-mapping records;
- invalid Zerone Bech32 addresses;
- Cosmos Ed25519 and secp256k1 BaseAccount public keys that do not derive the
  stored account address, including keys nested in vesting/module wrappers;
- whether `last_key_rotations` is present, its record count, invalid/duplicate/
  orphaned anchors, and agreement between each account's operational-key
  version and latest cooldown anchor.

Unknown Cosmos public-key types are reported as unsupported instead of being
guessed.

The Ed25519 and operational-key-hash checks call the same helpers used by the
current auth module, so the census and consensus entry points accept the same
point set and hash encoding.

## Source and deployed-state boundary

Current source accepts only the full 64-lowercase-hex DID derived from the
exact 32-byte Ed25519 identity public key. A 32-hex prefix or uppercase form is
not valid successor genesis state. The census nevertheless needs to describe
historical state truthfully: when, and only when, a full input has
`"chain_id":"zerone-1"`, those known legacy encodings and a missing
`operational_key_hash` are warnings rather than corruption errors. The report
labels that mode `deployed-zerone-1-legacy-audit`.

An `app_state` object or standalone module has no authenticated chain ID, so it
uses the `source-canonical` profile. Copying a module out of a `zerone-1`
export therefore cannot silently acquire a legacy exception. Prefer the full
export and preserve its chain ID, height, and app hash with the report.

Legacy warnings are observations, not migration approval. Use
`--fail-on warning` whenever the desired artifact must already satisfy the
current source contract.

## Coverage boundary

A full genesis or `app_state` object is the complete input. A standalone
`zerone_auth` module object is accepted for targeted analysis, but produces
`COSMOS_AUTH_STATE_MISSING` because it cannot prove BaseAccount invariants.

Current source exports the latest operational-key cooldown anchor as
`last_key_rotations`. Older snapshots may omit that field. Presence and record
count are reported separately; an absent legacy field is not evidence that no
rotation occurred. Even when present, it contains only the latest cooldown
height per account, not the historical sequence of signatures or keys.

The query API does not provide exhaustive account, DID-mapping, and rotation
iteration suitable for this audit. Point queries therefore cannot produce a
complete census. Use an export from a trusted node and record its height/app
hash alongside the generated report.

Even a clean report proves only internal snapshot consistency. It cannot prove
private-key possession, registration/rotation signature validity, historical
events, or data omitted by the audited snapshot.
