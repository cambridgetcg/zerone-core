# Darwin descriptor ACL inspector

`main.c` is the auditable native boundary used by both `zeroned tx zerone_auth
onboard` and Frontier intake on macOS. The target object is inherited as file
descriptor 0. The helper accepts no path and no arguments, so a rename,
symlink, or `/dev/fd` pathname race cannot redirect the ACL decision.

The complete versioned protocol is:

- no extended ACL: stdout `zerone-darwin-acl-v1 clear\n`, empty stderr, exit 0;
- local or inherited extended ACL: stdout `zerone-darwin-acl-v1 present\n`,
  empty stderr, exit 10;
- inspection or protocol error: exit 70 (with a diagnostic on stderr when
  possible).

Callers accept only the first exact tuple, reject the second, and fail closed
on every other result, timeout, signal, unsafe helper metadata, or missing
helper. Linux does not execute this helper because its mode-bit check also
constrains the effective POSIX ACL mask.

Run `make darwin-acl-helper` on macOS. It produces a deterministic universal
arm64/x86_64, ad-hoc-signed generated artifact at `build/darwin-acl-check` and
copies the identical artifact beside the production Frontier Bun bundle at
`tools/frontier-intake/build/darwin-acl-check`. No Mach-O is committed. The
helper deployment target is macOS 12, matching the repository's pinned Go
1.25 runtime baseline. `make build` and `make install` place it beside
`zeroned`; the Darwin release target includes it in release checksums.
Because this native companion cannot be produced by the repository's CGO-free
Linux cross-compile, `build-darwin-arm64`, and therefore aggregate `build-all`
and `release`, deliberately require a macOS release host. The two explicit
Linux build targets remain portable and unchanged.

Ad-hoc code signing is an execution-format requirement, not provenance. The
current `make release` emits SHA-256 sidecars, which establish integrity but
not producer identity. A production distributor must cryptographically bind
both `zeroned-darwin-arm64` and `darwin-acl-check` into its signed release
evidence; installing one without the other makes onboarding fail closed. For
Frontier intake, `make frontier-intake-macos-package` puts the helper and exact
Bun bundle in one deterministic archive, and the protected production signing
workflow signs that archive as a single object. See
`tools/frontier-intake/README.md` for the verification boundary.

Both callers open and validate the helper and its non-group/world-writable
containing directory before execution, hold those descriptors, and re-open and
compare them after execution. Bun and Go still execute the helper by its
absolute bundle-relative path because portable descriptor execution is not
available through Bun's stable spawn API. The installation tree is therefore
a code-integrity trust boundary: an attacker able to mutate it as the
installing user could replace the caller itself as well as the helper. The
pre/post inode, metadata, and directory checks detect ordinary path swaps and
fail closed. Distributor signing and installation integrity must address
same-user code replacement.
