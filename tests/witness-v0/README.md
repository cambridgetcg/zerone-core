# WITNESS v0 conformance

The executable conformance suite lives with the isolated tool:

```sh
go test ./tools/witness-v0/...
```

It covers the closed 10-kind/18-action matrix, activation blocking for every
action, exact-key deletion and addition matrices, strict prime-subgroup
Ed25519, hash domains, hostile JSON, the full Unicode/string profile, exact
wire bytes, uint64 boundaries, RFC 6962 batches, duplicate-receipt refusal,
the explicit cross-batch settlement limitation, stable asset-bound capability
nullifiers, permanent replay refusal, cumulative spend, lifecycle monotonicity,
complete collaboration journal prefixes, key rotation/revocation, subject
controller/kind pinning, and safe CLI paths.

Fixtures in `tools/witness-v0/testdata` are known-answer protocol artifacts, not
live chain inputs. Their byte and expectation index is reproducible with:

```sh
go run ./tools/witness-v0/cmd/witness-fixtures --check ./tools/witness-v0/testdata
```
