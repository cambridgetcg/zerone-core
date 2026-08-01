# Go dependency security baseline

This note records the dependency baseline prepared on 2026-07-29 for the
Cosmos SDK 0.53 / IBC-Go v10 migration branch. It is an audit record, not
approval for a rolling validator deployment.

## Baseline

- The root module, CI, Docker builders, node bootstrap and ceremony scripts,
  active operator documentation, and isolated Sigstore compiler use Go
  1.25.12.
- `govulncheck` is run from source with
  `golang.org/x/vuln/cmd/govulncheck@v1.6.0`.
- The patched graph includes gRPC 1.82.1, `x/text` 0.39.0,
  go-getter 1.8.6, go-ethereum 1.17.0, OpenTelemetry 1.44.0,
  `filippo.io/edwards25519` 1.1.1, `x/net` 0.56.0,
  `x/crypto` 0.53.0, and `klauspost/compress` 1.18.7.
- AWS EventStream 1.7.8 is not sufficient by itself for GO-2026-5764.
  The S3 service module is also pinned at 1.97.3, the first fixed S3
  release listed by the advisory.
- Compared with the migration parent, the root scan drops from 16 symbol
  findings plus 3 package-only findings to the two symbol findings documented
  below and zero package findings. The isolated Sigstore compiler reports one
  symbol finding: the same no-fix `x/crypto/openpgp` advisory documented below.

The large module-graph change is expected. The go-getter 1.8.6 fix replaces
its AWS v1 dependency path with AWS v2, and the go-ethereum 1.17.0 fix adds
its current transitive proof-runtime dependencies. Reconstructing the update
from the pre-hardening module with only the reachable fixed versions produces
the same graph shape; this is not an unbounded `go get -u` refresh.

## Remaining source findings

The default source scan reports two findings.

### GO-2026-5932: `x/crypto/openpgp`

This is a real reachable dependency with no fixed `x/crypto` version.
Cosmos SDK 0.53.8 imports only `openpgp/armor` in its `crypto` package to
encode and decode ASCII-armored Tendermint key material. Zerone does not use
the package as an OpenPGP signature or message-encryption protocol, but the
Go vulnerability record intentionally marks every symbol in the unmaintained
package.

The isolated Sigstore compiler scan also reports this advisory through its
Sigstore and certificate-transparency dependency paths. The compiler handles
Sigstore bundles and X.509 certificates rather than OpenPGP messages, but the
dependency remains reachable to the scanner and is not suppressed.

Until Cosmos SDK removes or replaces this import:

- treat armored key import as a local operator action;
- do not expose an unauthenticated remote armored-key decoder;
- do not suppress the finding; and
- re-evaluate an upstream Cosmos SDK fix before attempting a local module
  replacement.

Reference: <https://pkg.go.dev/vuln/GO-2026-5932>

### GO-2024-2584: Cosmos SDK slashing

This is a vulnerability-database range error for Cosmos SDK 0.53.8. The
official Cosmos advisory lists affected releases as `<= 0.47.9` and
`<= 0.50.4`, with fixes in 0.47.10 and 0.50.5. The Go vulnerability record
adds an `introduced: 0.50.0` range but omits the corresponding
`fixed: 0.50.5` event, so it incorrectly treats every later release as
affected. Cosmos SDK 0.53.8 contains the fixed undelegation-after-redelegation
slashing path.

This exception is documented rather than hidden from scanner output.

References:
<https://github.com/cosmos/cosmos-sdk/security/advisories/GHSA-86h5-xcpx-cfqc>,
<https://vuln.go.dev/ID/GO-2024-2584.json>

## Module-only findings

Verbose scans also report findings for modules without a reachable affected
package or symbol:

- GO-2025-3442 for CometBFT 0.38.23. The database lists only CometBFT 1.0.1
  as fixed; that major line is not compatible with this Cosmos SDK 0.53
  migration and must be handled as a later coordinated stack upgrade.
- GO-2023-1881 and GO-2023-1821 for Cosmos SDK `x/crisis`. Zerone does not
  import or register `x/crisis`; the package is present only inside the
  required Cosmos SDK module.

The isolated compiler reports these three module-only findings plus
GO-2024-2584 as module-only, because it requires the root module but does not
call the slashing path.

These are not treated as cleared. Re-run the scanner whenever the consensus
stack changes.

## Verification

```sh
go mod verify
go test ./... -count=1
go build ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

The Sigstore compiler is verified separately from its nested module with
Go 1.25.12, including tests, vet, race tests, and a reproducible
Linux/AMD64 build.

Any validator rollout still requires one reviewed binary and a coordinated
upgrade height. A dependency-only change is not safe for a mixed-version
validator set merely because application source did not change.
