# Cosmovisor Setup for Zerone

[Cosmovisor](https://docs.cosmos.network/sdk/v0.53/build/tooling/cosmovisor) is a process manager that watches for governance-approved upgrade proposals and swaps a previously staged `zeroned` binary at the activation height.

Zerone validators use the tested
[Cosmovisor `v1.7.1`](https://github.com/cosmos/cosmos-sdk/releases/tag/cosmovisor%2Fv1.7.1).
They must not download binaries from an on-chain URL at activation time.

The repository's validator image adds a fail-closed staging entrypoint around
Cosmovisor. It never treats image replacement alone as upgrade authorization:
the image binary must match an independently authenticated SHA-256, and a
pending upgrade is copied only into the exact named upgrade directory.

## Directory Structure

```text
$DAEMON_HOME/cosmovisor/
├── genesis/
│   └── bin/
│       └── zeroned          ← initial binary (regular copied file; never a symlink)
└── upgrades/
    └── v1.0.0-testnet/
        └── bin/
            └── zeroned      ← upgraded binary (placed before upgrade height)
```

## Quick Start

### 1. Install Cosmovisor

```bash
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.7.1
go version -m "$(command -v cosmovisor)" |
  awk '$1 == "mod" && $2 == "cosmossdk.io/tools/cosmovisor" { print $2, $3 }'
```

The second command must report
`cosmossdk.io/tools/cosmovisor v1.7.1`. Do not use `@latest` in a validator
build or bootstrap script.

### 2. Configure the validator policy

```shell
export DAEMON_NAME=zeroned
export DAEMON_HOME=$HOME/.zeroned
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_LOG_BUFFER_SIZE=512
export UNSAFE_SKIP_BACKUP=false
```

`DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true` is defense in depth if the download
policy is ever changed; it does not verify a manually staged binary.
`UNSAFE_SKIP_BACKUP=false` preserves the pre-upgrade data backup. Confirm the
backup filesystem has enough free space during rehearsal.

### 3. Initialize

```shell
cosmovisor init "$(command -v zeroned)"
```

Confirm the installed genesis binary has the expected digest:

```shell
shasum -a 256 "$DAEMON_HOME/cosmovisor/genesis/bin/zeroned"
```

The genesis binary must be a regular copied file. Never point it at
`/usr/local/bin/zeroned` or another mutable image path: replacing an image
would then change the pre-H binary and bypass the named upgrade boundary. The
validator container entrypoint rejects a genesis binary symlink.

Compare that digest with an independently authenticated release manifest.

### 4. Run

```shell
cosmovisor run start
```

## Preparing an Upgrade

1. Reproduce or obtain the release artifact and its authenticated manifest.
2. Set `EXPECTED_SHA256` from that independently verified manifest, then check
   the artifact before it enters the supervisor directory:

   ```shell
   test "$(shasum -a 256 build/zeroned | awk '{ print $1 }')" = \
     "$EXPECTED_SHA256"
   ```

3. Stage the binary atomically under the exact normalized governance plan
   name. The temporary file must be in the destination directory so the final
   rename cannot cross filesystems:

   ```shell
   UPGRADE_NAME=v1.0.0-testnet
   UPGRADE_BIN_DIR="$DAEMON_HOME/cosmovisor/upgrades/$UPGRADE_NAME/bin"
   UPGRADE_BINARY="$UPGRADE_BIN_DIR/zeroned"
   install -d -m 0755 "$UPGRADE_BIN_DIR"
   if [ -e "$UPGRADE_BINARY" ] || [ -L "$UPGRADE_BINARY" ]; then
     test -f "$UPGRADE_BINARY" && test -x "$UPGRADE_BINARY" &&
       test ! -L "$UPGRADE_BINARY"
     test "$(shasum -a 256 "$UPGRADE_BINARY" | awk '{ print $1 }')" = \
       "$EXPECTED_SHA256"
   else
     STAGED_BINARY="$(mktemp "$UPGRADE_BIN_DIR/.zeroned.tmp.XXXXXX")"
     install -m 0755 build/zeroned "$STAGED_BINARY"
     test "$(shasum -a 256 "$STAGED_BINARY" | awk '{ print $1 }')" = \
       "$EXPECTED_SHA256"
     mv "$STAGED_BINARY" "$UPGRADE_BINARY"
     test "$(shasum -a 256 "$UPGRADE_BINARY" | awk '{ print $1 }')" = \
       "$EXPECTED_SHA256"
   fi
   ```

4. Rehearse from an H-1 database copy and record the resulting app hash,
   migration output, binary digest, and backup/restore timing.
5. Before activation, verify the on-chain plan name and height match the
   staged directory and the reviewed release manifest.

Cosmovisor swaps only to this pre-positioned binary. With downloads disabled,
a missing or incorrectly named artifact fails closed instead of resolving and
executing network content at the upgrade height.

The backup is recovery evidence, not authority to rewrite committed history.
After the upgrade block has committed, recover forward with a corrected binary
or an explicitly coordinated fork; do not restart an old binary as though the
upgrade had not occurred.

## Validator image staging contract

`Dockerfile.validator` runs
[`deploy/validator-cosmovisor-entrypoint.sh`](../deploy/validator-cosmovisor-entrypoint.sh).
Every container start must provide the image payload digest, the immutable
genesis root, and the currently authorized predecessor digest:

```shell
export DAEMON_BINARY_SHA256=<64-lowercase-hex-from-authenticated-manifest>
export DAEMON_GENESIS_BINARY_SHA256=<original-genesis-binary-sha256>
export DAEMON_CURRENT_BINARY_SHA256=<currently-authorized-binary-sha256>
```

The entrypoint hashes the regular, executable, non-symlink image binary at
`/usr/local/bin/$DAEMON_NAME` before touching the Cosmovisor tree. It installs
through a same-directory temporary file, atomically renames it, and verifies
the installed digest again. On every later boot it first re-hashes the genesis
binary against `DAEMON_GENESIS_BINARY_SHA256`, before any staging mutation.

For the initial binary, leave `DAEMON_UPGRADE_NAME` empty. On an empty home,
also leave `DAEMON_CURRENT_UPGRADE_NAME` empty and set all three digests to the
same authenticated image digest. The entrypoint installs that binary as:

```text
$DAEMON_HOME/cosmovisor/genesis/bin/$DAEMON_NAME
```

For a later upgrade, preserve that genesis binary and bind both the predecessor
and pending release:

```shell
export DAEMON_GENESIS_BINARY_SHA256=<original-genesis-binary-sha256>
export DAEMON_CURRENT_UPGRADE_NAME=<empty-when-current-is-genesis-or-prior-plan-name>
export DAEMON_CURRENT_BINARY_SHA256=<authorized-predecessor-sha256>
export DAEMON_UPGRADE_NAME=v1.0.0-testnet
export DAEMON_BINARY_SHA256=<upgrade-binary-sha256>
```

`DAEMON_UPGRADE_NAME` must exactly match the on-chain plan name and be
canonical lowercase ASCII: letters, digits, `.`, `_`, and `-`, beginning and
ending with a letter or digit. The entrypoint rejects `"."`, `".."`, uppercase
characters, whitespace, slashes, and directory symlinks. With the variable
set, it installs only:

```text
$DAEMON_HOME/cosmovisor/upgrades/$DAEMON_UPGRADE_NAME/bin/$DAEMON_NAME
```

The `cosmovisor/current` selector may be absent only while genesis is the
predecessor. Otherwise it must be a relative symlink whose text is exactly
`genesis`, `upgrades/$DAEMON_CURRENT_UPGRADE_NAME`, or—after Cosmovisor
activates the pending release—`upgrades/$DAEMON_UPGRADE_NAME`. Absolute,
traversing, operator-selected, or directory-symlink paths fail closed. The
selected executable is re-hashed against the corresponding predecessor or
pending digest immediately before Cosmovisor starts.

An existing genesis, predecessor, or staged upgrade binary is accepted only
when it already has the expected digest. A different digest at the same path
fails closed. Cancel or supersede the release through the signed operations
process and use a separately authorized plan/path; do not silently replace
bytes under an already attested name. On the next release, promote the prior
plan name/digest into `DAEMON_CURRENT_UPGRADE_NAME` and
`DAEMON_CURRENT_BINARY_SHA256`.

The image still requires:

```shell
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM=true
export UNSAFE_SKIP_BACKUP=false
```

These values are enforced by the entrypoint, not merely suggested defaults.
Runtime overrides that enable downloads, disable checksum enforcement, disable
restart-after-upgrade, or skip the backup fail before Cosmovisor starts.
`DAEMON_NAME` is fixed to `zeroned`; `DAEMON_HOME` must be an absolute,
canonical, non-root path and neither it nor the inspected validator
directories may be symlinks. Both image binaries must be regular,
non-symlink executables, and the validator entrypoint accepts only the frozen
`cosmovisor run start` command.

The checksum setting does not substitute for any of the three lineage digests;
downloads remain disabled.

The image build itself is also manifest-bound. Supply a semantic `VERSION`, the
full 40-character lowercase source `COMMIT`, and `SOURCE_DATE_EPOCH`:

```shell
docker build \
  --build-arg VERSION=v1.2.3 \
  --build-arg COMMIT=<40-lowercase-source-commit> \
  --build-arg SOURCE_DATE_EPOCH=<positive-unix-epoch> \
  -f Dockerfile.validator .
```

The build rejects missing or malformed values, embeds and verifies the version
and commit, labels the runtime image, pins both base-image indexes by digest,
and resolves Debian runtime packages from the fixed 2026-07-13 snapshot.

## Existing populated homes

The entrypoint will not infer a genesis binary from a new image when
`$DAEMON_HOME/data` is populated. It also rejects an empty Cosmovisor home when
`DAEMON_UPGRADE_NAME` is set. Both cases prevent a pending-upgrade image from
becoming the pre-H binary.

For a pre-Cosmovisor populated home that lacks
`cosmovisor/genesis/bin/zeroned`:

1. Stop the node and preserve the data directory, consensus WAL,
   `priv_validator_state.json`, and the last committed height/block/AppHash.
2. Identify the exact binary that produced the existing state and verify its
   SHA-256 against its independently authenticated historical release
   manifest. Do not use the pending upgrade binary.
3. Install that verified historical binary as a regular executable file in
   `cosmovisor/genesis/bin/zeroned` using a same-directory temporary file and
   atomic rename.
4. Verify the installed digest again; set both genesis/current digests to that
   historical digest, leave both upgrade-name variables empty, and start
   Cosmovisor against the existing data.
5. Only after the pinned genesis layout is healthy should a separately
   attested upgrade image start with the exact current/pending names and all
   three lineage digests.

Do not delete or regenerate application data to bypass this safeguard. If the
historical binary cannot be authenticated, stop and enter the signed incident
or fork-recovery process.
