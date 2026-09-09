# Reproduce the observer builds

The packet distinguishes two executables: the existing network runtime was
reproduced exactly as forensic evidence; the observer candidate applies an
explicit dependency patch to that same application source. The signed release
manifest identifies the executable actually supplied. Neither reproduction nor
a matching checksum is a security clearance or authorization to run a signer.

## Patched observer recipe

`observer-build.py` starts from the exact 657-file production compile closure at
`2e37c4c86c31e67515d6aa07b6fd406083e012ea`. It accepts only the reviewed
`observer-dependencies.patch`, changing `go.mod` and `go.sum`. The application
source, Cosmos SDK v0.50.15 and IBC v8.8.0 remain unchanged. CometBFT advances
from v0.38.20 to v0.38.25, with four minimum-version-selection changes:
`lib/pq` v1.12.0, `highwayhash` v1.0.4, `goid` revision `a731cc31b4fe`, and
`go-deadlock` v0.3.9. Go 1.25.14 replaces the original Go 1.24.13 compiler.

The exact dependency patch SHA-256 is
`3eb79d1fa44f76fe8a5c8edbcf90445b5cd78b7b2d08d51a296143135fdd14ce`.
The pinned official `golang:1.25.14-bookworm` Linux amd64 builder is:

```text
registry-1.docker.io/library/golang@sha256:c268a04d59aea0b180ed9946a658cfab9e7b3391dc90eed6e4969ccff98c851f
```

Its multi-platform index digest is
`sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437`.
Native execution confirmed Go 1.25.14 and GCC
`(Debian 12.2.0-14+deb12u1) 12.2.0`. The patched build uses `GOFLAGS=-mod=readonly`,
`GOWORK=off`, module checksum verification and otherwise the Makefile command
and environment described below.

Use Python 3.11+ and a native Linux amd64 host with Docker available on
`/var/run/docker.sock`. Each build permits at most 3 CPUs and 6 GiB RAM; the
recorded host had 4 CPUs and 8 GiB RAM. Allow disk space for new module and
compilation caches. Builds fetch the official image and public Go modules.
After verifying the packet, run from its directory, copying the executable
SHA-256 from the verified release manifest:

```sh
OBSERVER_SHA256='replace-with-the-verified-manifest-executable-sha256'
python3 observer-build.py \
  --source-archive "$PWD/legacy-source.tar.gz" \
  --dependency-patch "$PWD/observer-dependencies.patch" \
  --expected-sha256 "$OBSERVER_SHA256" \
  --output "$PWD/../zerone-observer-reproduction"
```

The output must be a new sibling directory, outside the verified packet. A
successful comparison writes `provenance.json` under that output with
`source_status: "reproduced-observer-patch"`; the executable is `out/zeroned`.
Omitting `--expected-sha256` produces only `observer-candidate-built` status;
that does not establish independent reproduction or release approval. Repeat
with another new output directory for an independent compilation cache.

The helper authenticates the entire original archive before extraction, then
validates the patch and both resulting dependency files. All other 655 files
remain unchanged. Source mounts are read-only; output, module cache, build cache
and temporary directories are fresh. The container drops capabilities and uses
a read-only root filesystem, running as your numeric UID/GID. UID 0 remains UID
0 when invoked as root. The helper starts no node and submits no transaction.

The release's `provenance.json` binds the patch, actual executable, dependency
inventory and reproduction receipts. Read `VULNERABILITY-REVIEW.json` and the
compatibility evidence before use. Patched observer compatibility does not
approve upgrading the existing validator or changing application state.

## Original runtime: forensic reproduction

The September 9, 2026 reconstruction built the existing `zerone-1` executable
byte for byte from the same source commit. Its SHA-256 is
`94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8`,
and its size is 99,292,776 bytes. This unpatched executable is a forensic
reference, not the recommended public observer build.

The original build did not record its Git revision: its CLI reports `dev` /
`unknown`, and its original VCS revision remains null. Byte identity establishes
reproduction from the pinned source; it does not recover a missing VCS field.
The predecessor `fc60797aae8b7c21a324dc1ad3d7d271f5ac4749` has the same compiled
source; four changed deployment files are outside this export.

| Original input | Exact value |
| --- | --- |
| Source commit | `2e37c4c86c31e67515d6aa07b6fd406083e012ea` |
| Full Git tree | `5c85dd9f4f3c62a2494142509bc013e4678b0936` |
| Exported compile closure | 657 tracked regular files |
| Uncompressed tar SHA-256 | `62054eed39981c367c9c6d5bdf7bb19e54766afe73d336c2ea7f4a45cbfb400f` |
| Go | `go1.24.13` |
| Platform | Native Linux amd64, `GOAMD64=v1`, `CGO_ENABLED=1` |
| C compiler | `gcc (Debian 12.2.0-14+deb12u1) 12.2.0` |
| Builder | `registry-1.docker.io/library/golang@sha256:98d673f18a1aac43da744209873cb79323e11706f909251bcfb131828b95559d` |
| Dynamic runtime | `libc.so.6`, interpreter `/lib64/ld-linux-x86-64.so.2` |
| SDK / IBC / Comet dependency | `v0.50.15` / `v8.8.0` / `v0.38.20` |

The original builder is the official `golang:1.24.13-bookworm` Linux amd64
manifest, from index
`sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac`.
CometBFT v0.38.20 hardcodes `TMCoreSemVer = "0.38.19"`; that explains the
existing node's RPC version string. The original ELF contains 178 linked Go
dependencies. A Go dependency inventory is not a full operating-system SBOM.

To reproduce the original unpatched reference separately:

```sh
python3 provenance.py build \
  --source-archive "$PWD/legacy-source.tar.gz" \
  --output "$PWD/../zerone-base-reproduction"
```

Success requires the original SHA-256 above and writes `reproduced-exact`
status. The Makefile command is `make build VERSION=dev COMMIT=unknown`:

```sh
go build -trimpath -ldflags '-s -w -X github.com/cosmos/cosmos-sdk/version.Name=zerone -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned -X github.com/cosmos/cosmos-sdk/version.Version=dev -X github.com/cosmos/cosmos-sdk/version.Commit=unknown' -o build/zeroned ./cmd/zeroned
```

The explicit environment is `GOTOOLCHAIN=local`, `GOMAXPROCS=3`,
`GOCACHE=/cache/build`, `GOMODCACHE=/cache/mod`, `GOPATH=/cache/gopath`,
`CGO_ENABLED=1`, `GOOS=linux`, `GOARCH=amd64`, `GOAMD64=v1`. The patched recipe
adds `GOWORK=off` and `GOFLAGS=-mod=readonly`. Both helpers record their full build
script and image in the resulting receipt.

The first historical reconstruction's post-build CLI metadata inspection tried
to create `/.zeroned` on the read-only container root and failed. Compilation,
exact hash comparison and source checks had succeeded. A corrected offline
inspection with `--home /tmp/metadata-home` passed; the original failure remains
in the forensic evidence. Both public recipes use the corrected invocation.

## Export the original source independently

From a clone of `https://github.com/cambridgetcg/zerone-core` containing the
historical commit, invoke the reviewed release helper:

```sh
python3 deploy/legacy-observer/provenance.py export \
  --repository "$PWD" --output "$PWD/../zerone-legacy-source-export"
```

Export reads pinned Git objects, ignoring working-tree changes. It includes
production Go under `app`, `cmd` and `x`, module files, Makefile, license, and
three Swagger embed resources. It excludes tests, deployment seeds, signer
files, node homes, keyrings, caches and `.git`. `source-closure.json` records
all exported Git blob identities and SHA-256 hashes.

## Vulnerability evidence remains separate

The original CometBFT v0.38.20 is affected by the upstream critical
[Tachyon advisory](https://github.com/cometbft/cometbft/security/advisories/GHSA-c32p-wcqj-j677).
A nop mempool, zero voting power and loopback RPC do not fix that dependency.
The observer patch includes a version beyond the upstream v0.38.21 fix, and a
newer Go toolchain. It preserves other legacy dependencies and is not a claim
that every vulnerability has been removed.

The dated `VULNERABILITY-REVIEW.json` identifies the actual supplied executable,
scanner results and review limits. Binary scanning does not establish runtime
exploitability or a complete call graph. Some reported module branches and
function names require correction against primary advisories and the actual
ELF function table. Application or further dependency changes require a
separate compatibility decision.
