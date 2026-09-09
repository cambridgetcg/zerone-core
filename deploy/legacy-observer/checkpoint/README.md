# Offline legacy checkpoint verifier

This isolated Go module pins patched CometBFT 0.38.25 and Go 1.25.14. It reads one public checkpoint file, verifies it
offline, and emits a JSON receipt. It never opens a node home or connects to a
network. Build with a pinned Go toolchain and record the binary digest in the
release manifest.

```sh
go test -mod=readonly ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly -trimpath -buildvcs=false -o verify-checkpoint .
./verify-checkpoint --checkpoint /bundle/checkpoint.json
```

`--allow-expired` retains all cryptographic checks while allowing an expired
checkpoint to be examined during existing-state resume. Its receipt explicitly
records this choice. It must not authorize an initial state-sync bootstrap.

The flat checkpoint schema is `zerone-1-observer-checkpoint/v1`:

- `chain_id`: exactly `zerone-1`.
- `height`: positive JSON integer.
- `block_hash`: uppercase 64-character header hash.
- `header_time`: exact UTC RFC3339Nano header timestamp.
- `trust_period_seconds`: exactly 604800 (seven days).
- `signed_header`: the standard Comet `/commit` result's `signed_header` object.
- `validator_set`: the complete standard Comet `/validators` result object,
  containing `block_height`, `validators`, `count`, and `total`.

The verifier checks bounded JSON, duplicate and unknown fields, exact chain and
height, canonical header hashing, header/commit agreement, complete validator
identity and power, validator-set hashing and commit signatures. It also checks
the signature's declared validator address against the fixed validator pin:
that address is omitted from canonical vote sign bytes, and CometBFT versions
through 0.38.20 failed to bind it correctly (CSA-2026-001 / Tachyon). It pins the
observed sole validator's address and public key explicitly. A future change of
validator set requires a reviewed checkpoint contract update; it is not accepted
from a new RPC response alone.

This authenticates consistency with that operator-observed key. Same-upstream
RPC aliases provide no independent witness, and signature verification does not
repair signer custody. The fixture is a public capture at height 1262000 on
2026-09-09, used with a fixed test clock; it is not an evergreen bootstrap anchor.

Source derivation: the canonical Comet checks adapt the earlier private census
`verify-snapshot-anchor.go` (SHA256
`c12485e90bfd5d0596521404ab0271dcc602cd24010592e5355dc506139723e6`).
The release capture additionally verifies adjacent H/H+1/H+2 blocks and their
part-set commitments. This small bootstrap verifier does not claim to validate
block bodies, historical ancestry, genesis replay, or state-census completeness.
