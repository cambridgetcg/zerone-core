# A compatible observer for the existing zerone-1 ledger

This package follows the existing custodial chain with a fresh, zero-power
node. The original live executable was reproduced byte for byte from source
commit `2e37c4c86c31e67515d6aa07b6fd406083e012ea`, using Go 1.24.13. Its SHA-256 is
`94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8`.
That forensic result identifies the base application. The observer candidate
applies a separately reviewed dependency patch and newer Go toolchain; its
actual executable hash, base commit and patch are bound by the signed manifest.
It must not be described as byte-identical to the installed live executable.
The separately versioned tooling creates the observer. Read the particular
release's signed manifest, provenance and rehearsal report before starting.

Use a Linux amd64 host with Python 3.11+, `gpgv`, and a local Docker daemon,
at least 2 available CPUs, 6 GiB available memory, and 20 GiB free disk for an
initial trial. Disk usage grows: this observer retains new application states
and blocks. These are trial resource bounds, not a long-term capacity promise.
The first release has no tested macOS, ARM, Kubernetes, or remote Docker path.

## Verify before executing package code

Use the immutable tooling commit linked on the GitHub release page. Fetch the
source checkout and detach at that full commit; do not use a moving branch.
Download the named observer archive from the same release. The release page
provides its exact SHA-256 and a separately signed rehearsal receipt binding
the tested manifest. There is no curl-to-shell installer.

The established main release authority has this full OpenPGP fingerprint:

```text
09327B031F8FF2C2EE49B18F2234027FC5B68C19
```

Confirm that pin through a previously trusted Zerone source. A public key
shipped beside a download cannot establish its own identity. The source-pinned
verifier fixes this fingerprint, uses an isolated public keyring, checks the
detached manifest signature, and verifies every artifact's size and SHA-256.
It rejects symlinks, extra files, duplicate JSON fields and expired bootstrap
windows. Do not execute a verifier fetched only from an unverified archive.

From the source checkout, with your downloaded archive at the indicated path:

```sh
python3 -I -B deploy/legacy-observer/package.py unpack \
  --archive /absolute/path/to/observer-release.tar.gz \
  --output /absolute/path/to/new-observer-package \
  --gpgv /usr/bin/gpgv
```

This creates a new flat directory, checks archive entry bounds, and verifies
the signed contents without executing the node. An error leaves the directory
for inspection and never overwrites it. Change `/usr/bin/gpgv` if your trusted
installation uses another absolute path. A `PASS` authenticates the operator's
statement and files; canonical checkpoint validation also runs during `init`.

Download `REHEARSAL-RECEIPT.json` and `REHEARSAL-RECEIPT.json.sig` from the
same immutable release, keeping them outside the flat package directory. From
the source checkout, verify the signed account of the compatibility trial:

```sh
python3 -I -B deploy/legacy-observer/verify-rehearsal.py \
  --bundle /absolute/path/to/new-observer-package \
  --archive /absolute/path/to/observer-release.tar.gz \
  --receipt /absolute/path/to/REHEARSAL-RECEIPT.json \
  --signature /absolute/path/to/REHEARSAL-RECEIPT.json.sig \
  --gpgv /usr/bin/gpgv
```

This verifies the main authority's attestation against the exact archive,
manifest, checkpoint, advancing heights and state-root relationship. It does
not repeat the trial or turn operator testimony into independent observation.
Proceed to initialization only after both verification commands pass.

## Create and follow

Choose a new dedicated home, separate from any validator or other node. Keep
the verified package directory for future restarts. Paths below are examples
and must be replaced with your actual absolute paths.

```sh
python3 -I -B /absolute/path/to/new-observer-package/observer.py init \
  --home /absolute/path/to/new-observer-home

python3 -I -B /absolute/path/to/new-observer-package/observer.py start \
  --home /absolute/path/to/new-observer-home
```

Initialization runs offline, generates fresh random node identities, copies
the exact public genesis, and fixes the peer, checkpoint and observer settings.
It creates no account or funds. Never import an old validator identity.

The foreground launcher starts a constrained container and reports `SYNCING`
while it restores a snapshot. It reports `FOLLOWING` only after the matching
zero-power observer catches up to fresh blocks and then advances. This may
take several minutes. Press Ctrl-C to stop the owned container cleanly; keep
the home. Repeat `start` with the same package and home to resume.

In another terminal:

```sh
python3 -I -B /absolute/path/to/new-observer-package/observer.py status \
  --home /absolute/path/to/new-observer-home
curl --fail --max-time 10 http://127.0.0.1:27657/status
curl --fail --max-time 10 http://127.0.0.1:27657/abci_info
```

`init --port 27658` can choose another loopback RPC port. The package publishes
no P2P, REST, gRPC, profiling or metrics host port. The two state-sync RPC URLs
reach the **same upstream**; they are not independent witnesses. The HTTPS
website gateway does not provide the Comet POST interface required for sync.

The `nop` mempool rejects all three broadcast RPC methods and disables
transaction gossip. Raw local `check_tx` remains an ABCI check with transient
local effects; the raw RPC is not a strictly query-only API. Keep the RPC on
loopback and do not expose it through a reverse proxy. The helper itself exposes
only status reads and no transaction command.

## Trust, scope and expiry

The release signature authenticates an operator-selected chain checkpoint.
The checkpoint verifier checks the canonical Comet header, complete validator
set and commit signatures against the disclosed sole-validator pin. This does
not create an independent trust anchor. The legacy validator custody finding
remains open; another replica does not repair compromised signing authority.

State sync verifies and resumes from a recent application snapshot. It does
not replay every historical transaction or prove the completeness of older
knowledge settlement history. Compare application state at height H to the
app hash committed in header H+1 when checking state-root evidence.

The manifest's `expires_at` closes **new bootstrap**. It is at most seven days
from the checkpoint header, below the observed 21-day staking unbonding period.
Obtain a new signed release/checkpoint when this window expires. There is no
automatic unsigned checkpoint refresh. An already initialized and successfully
synced home can resume with its original authenticated package after this
deadline; integrity, identity, config and checkpoint signatures are still
checked. An unsynced home cannot use that exception.

The original CometBFT 0.38.20 runtime is affected by the critical
[Tachyon advisory](https://github.com/cometbft/cometbft/security/advisories/GHSA-c32p-wcqj-j677).
The observer candidate uses CometBFT 0.38.25 and Go 1.25.14 while retaining the
legacy Zerone application source, SDK and IBC generation. Stricter validation
may refuse a malformed block accepted by an unpatched peer; do not relax it.
This does not patch the running validator or repair its custody finding.
Reproduction and a successful compatibility trial do not establish an absence
of vulnerabilities. Read the release's vulnerability assessment and keep this
experimental observer isolated from valuable keys and other workloads.

The package grants no validator, account admission, reward, transaction,
protocol upgrade, reset, halt, or successor-chain authority. Current `main`
uses a different SDK/IBC generation and must not replace this live-compatible
executable. Validator joining and the H1/H2/H3 or zerone-2 release gates remain
separate. See [JOIN](../mainnet/JOIN.md), [TRUST](../mainnet/TRUST.md) and
[the validator safety guide](../../docs/VALIDATOR-GUIDE.md) in the source tree.

## Reproduction and release maintenance

`README-build.md`, `provenance.py`, `legacy-source.tar.gz` and the signed
`provenance.json` describe the executable reconstruction. The checkpoint
verifier's exact Go source/module pins are in
`checkpoint-verifier-source.tar.gz`. The tooling source is bound by
`OBSERVER-RELEASE.json.tooling.commit`.

Maintainers assemble explicit public inputs with `package.py assemble`, review
and sign the exact canonical manifest using the main release authority, then
rehearse that exact signed bundle. The published rehearsal receipt must bind
its manifest digest and cover snapshot restore, fresh block following,
same-home restart, exact state-root comparison, zero power and local broadcast
refusal. Archive only after signature and artifact verification with
`package.py archive`. Never transfer node homes, private keys, old validator
images, cached layers, or a broad workspace directory into a public package.
