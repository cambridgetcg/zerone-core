# frontier-intake

Client-side intake pipeline for the math breakthrough register: validated
register entries become `x/knowledge` claims, panel-verified into facts on the
live zerone networks. Shells the `zeroned` CLI; contains no consensus code and
persists a distinct Ed25519 identity key for each registered operator key.
Normative context: `docs/specs/math-frontier-absorption-v0.md`.

```
bun intake.ts validate                       # register schema + admission rules (no network)
bun intake.ts plan   --network zerone-testnet-1                 # free novelty/fee preview
bun intake.ts setup  --network zerone-testnet-1 --live-ack=zerone-testnet-1
bun intake.ts submit --network zerone-testnet-1 --live-ack=zerone-testnet-1 --batch
bun intake.ts panel  --network zerone-testnet-1 --live-ack=zerone-testnet-1 --watch
bun intake.ts status --network zerone-testnet-1
```

On macOS, build the descriptor-bound ACL inspector once from the repository
root before running setup or loading identity state:

```
make frontier-intake-darwin-acl-helper
```

The target builds a universal, ad-hoc-signed Mach-O at
`tools/frontier-intake/build/darwin-acl-check`. The binary is a generated
release artifact, not committed source; a production Bun bundle must remain in
that same `build/` directory. Frontier intake refuses to touch identity bytes
when the helper is absent, unsafe, times out, or returns anything except its
exact versioned clear result. Linux uses its mode-bit-enforced POSIX ACL mask
and neither builds nor loads the Darwin helper. The helper targets macOS 12,
matching the pinned Go 1.25 runtime baseline; its exact protocol, packaging,
and installation-tree trust boundary are documented in
`tools/darwin-acl-check/README.md`.

## Release posture

Build the macOS production unit from the repository root on macOS:

```
make frontier-intake-macos-package
make frontier-intake-macos-package-check
```

The first target builds the Bun entry point and helper, then atomically
publishes `build/frontier-intake-macos.tar`. The archive has one read-only
`frontier-intake-macos/` directory. It preserves the runtime-relative layout:
the exact bundle and universal helper are colocated at
`tools/frontier-intake/build/{frontier-intake.js,darwin-acl-check}`, while the
lexically latest canonical register is under `docs/research/` where the bundle
expects it. `MANIFEST.json` sits at the archive root. The deterministic,
sorted-key manifest records the source commit and Bun version and binds the
register and both executable files by path, mode, size, and SHA-256. Fixed
archive ownership, ordering, mtimes, and ustar encoding make successive builds
with the same inputs and toolchain byte-identical. The
byte-identical detached manifest and transport checksum are
`build/frontier-intake-macos.manifest.json` and
`build/frontier-intake-macos.tar.sha256`. `make release` includes this set.
CI pins Bun 1.3.5 and regression-checks two consecutive builds, the exact
member list and modes, payload equality, extraction layout, and execution from
the extracted directory.

A SHA-256 checksum detects corruption; it does not establish who produced an
archive. Ordinary CI never publishes the unsigned handoff. An explicitly
authorized `sign_release_components` dispatch passes the exact same-run
archive to the protected `zerone-production-signing` environment and emits a
Sigstore bundle named `FRONTIER-INTAKE-MACOS-SIGNATURE-BUNDLE.json`. Before
extraction or execution, verify the archive itself against the expected
workflow identity, issuer, release commit, transparency log, and signed
timestamp, then check its sidecar:

```
cosign verify-blob \
  --bundle FRONTIER-INTAKE-MACOS-SIGNATURE-BUNDLE.json \
  --certificate-identity \
    https://github.com/cambridgetcg/zerone-core/.github/workflows/ci.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-sha <40-hex-release-commit> \
  --use-signed-timestamps \
  frontier-intake-macos.tar
LC_ALL=C LANG=C /usr/bin/shasum -a 256 -c \
  frontier-intake-macos.tar.sha256
```

The protected workflow also requires the internal manifest's
`source_commit` to equal that release commit before signing. In a matching
source checkout, `make frontier-intake-macos-package-check
COMMIT=<40-hex-release-commit>` independently checks the checksum, exact
archive surface, internal/detached manifest equality, payload digests and
modes, universal slices, fixed helper identifier, code-signature structure,
and descriptor protocol. The helper's ad-hoc Mach-O signature remains an
execution-format property; only the separately verified distributor Sigstore
bundle supplies provenance.

Extract the verified archive as a unit and keep its directory read-only. Run
it as
`bun frontier-intake-macos/tools/frontier-intake/build/frontier-intake.js ...`;
moving only the Bun file or only the helper breaks the custody boundary and
setup fails closed.

Broadcasting commands (`setup`, `submit`, `panel`) refuse to run without
`--live-ack=<exact-chain-id>` — the explicit live-network authorization this
repo's broadcast-guard discipline requires (cf. `tools/agenttool-relay`,
which refuses live chains entirely; this tool's purpose IS live intake, so
the guard is an explicit per-invocation acknowledgment instead of a refusal).
Live reruns after the 2026-08 drill should revalidate: effective fees and
pacing (`plan`), verifier balance headroom (the gate reads spendable balance
and panel keys drift — never reuse a key with a day job, e.g. the relay's
`relayer`), RPC endpoints, and the phase parameters read live at startup.

## State

Runtime state lives outside the repo at
`~/.zerone-agent/frontier-intake/<network>.state.json` (0600), updated
read-merge-write under an exclusive lock so `submit` and a concurrent
`panel --watch` never drop each other's rounds. `submit` interleaves panel
passes during cooldown waits, so a single process also covers a full batch.

Rounds record honest stages: `submitted`, `pending_inclusion` (broadcast
accepted but inclusion unobserved — backfilled later, never lost),
`resolved_accepted`, `resolved_other`, `missed_window`, `abandoned`, `error`.
Claim statuses are classified numerically per the live enum (ACCEPTED = 6);
in-flight statuses keep waiting and transient RPC failures never produce a
terminal verdict.

Setup stores each identity key beside that state as
`<network>.<key>.ed25519.json` (0600). Its PKCS#8 private key signs the canonical
`zerone.auth/register-account/v1` proof over the chain ID, Cosmos sender, full
`did:zrn:<64-lowercase-hex-key>`, account type, and metadata. Back up and
protect these files as long-lived identity key material; an existing file whose
public and private halves do not match is rejected instead of silently replaced.
Reuse opens the file without following links and rejects non-regular,
wrong-owner, group/world-accessible, extended-ACL-bearing (on macOS), or
oversized files. Creation is exclusive and mode 0600; the key file and its
private parent directory are synced before the registration transaction can be
broadcast. Missing custody directories are created one component at a time
from an owned, private, descriptor-verified ancestor. Every new directory and
its containing directory are fsynced before key generation; unsafe components,
symlinks, and path-identity races fail closed.

Setup treats an existing on-chain account as a reconciliation obligation, not
an idempotent shortcut. It securely loads the local key without creating a
replacement, proves the public/private pair, and checks the exact address,
full self-certifying DID and identity key, `agent` role, current operational
public key and SHA-256 hash, and positive uint32 key version. A rotated-away
operational key is reported as unavailable rather than green. Transport errors,
malformed success output, and other ambiguous query failures never authorize
new key creation: only the auth module's exact NotFound diagnostic does. All
operator identities complete this preflight before the first funding or
registration broadcast.

## Tests

`bun test` — register validation (admission, disclosure, aggregator-only
sources, refutation wording), commit-hash conformance against the proven
first-truth ceremony recipe, exact Go-compatible registration proof vectors,
strict account-query/reconciliation negatives and no-broadcast guarantees, the
proof-bearing `zeroned` setup invocation, crash-durable custody directories,
and the pure panel phase logic including missed-window honesty. Wired into CI
as the `frontier-intake` job.
