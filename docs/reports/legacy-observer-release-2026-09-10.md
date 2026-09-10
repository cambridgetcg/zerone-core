# zerone-1 observer release — 2026-09-10

Signed version 2 passed the native sync, retained-home restart, offline root and signed-receipt checks. Version 1 failed safely before syncing and was never published. Public download and website checks are recorded separately.

**Experimental legacy trial:** the seven-day bootstrap window is not a maintenance or security-support promise. Go 1.25.14 is outside Go's two currently supported major releases after Go 1.27 shipped on 2026-08-19 ([Go release policy and history](https://go.dev/doc/devel/release)). [Cosmos Labs lists SDK 0.50 as end of life](https://docs.cosmos.network/sdk/latest/release-family), with maintenance, security patches and compatibility support ended. The September 9 scan reported no standard-library advisory for this executable, but that does not restore upstream support or establish the absence of defects. No new exploitable standard-library flaw has been established here. A future release needs review and reproduction on a maintained Go baseline. This release is not presented as a supported production runtime.

The package provides a separate Linux amd64 observer for people and agents who want to keep their own copy of the existing zerone-1 ledger. It uses fresh local node identities and zero voting power. The source-based local sandbox remains a separate network with test funds.

The package targets a new dedicated home and a local Docker daemon. It publishes RPC only on `127.0.0.1:27657`, disables the REST and gRPC servers, and configures the `nop` mempool. The helper preserves state across stop and restart. The exact signed v2 package restored a snapshot, followed three observed heights and refused all three broadcast methods. Both the initial run and retained-home restart exited cleanly with zero voting power and unused local signing state. Raw local `check_tx` is not promised to be query-only.

## Exact v2 package

| Record | Pin or status |
| --- | --- |
| Release ID | `zerone-1-observer-20260909.2` |
| Tooling source | `ed364bc0afec062612b61a309002f23da85acf2c` |
| Application base | `2e37c4c86c31e67515d6aa07b6fd406083e012ea` |
| Dependency patch SHA-256 | `3eb79d1fa44f76fe8a5c8edbcf90445b5cd78b7b2d08d51a296143135fdd14ce` |
| Observer executable SHA-256 | `d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443` |
| Signed manifest SHA-256 | `1300f9381d2e290e1468075806734c586fd92e4df7d0ec55e2f7c7a858df7d92` |
| Archive SHA-256 | `5d874b4c38427117f1567ea0152fc4782e066e4926f3294f1eb776ff6ce22d08` |
| Signed rehearsal receipt SHA-256 | `c99a1bac24b9ccc71a2769ce10d72f0a947eec8cb4b4afe9df22df2886c0ed0b` |
| Rehearsal signature SHA-256 | `b6c47423166cff29731741b71e9e9a1d5e30820eab93a41c06f26a3fd8b8c56f` |
| New-bootstrap expiry | `2026-09-16T19:19:13Z` |

The public release authority fingerprint is `09327B031F8FF2C2EE49B18F2234027FC5B68C19`. A user must confirm this pin through an already trusted source before executing downloaded package code. The [pinned setup guide](https://github.com/cambridgetcg/zerone-core/blob/ed364bc0afec062612b61a309002f23da85acf2c/deploy/legacy-observer/README.md) explains signature, file and checkpoint verification. The separately signed rehearsal receipt binds the exact manifest and archive; it remains an operator attestation, not independent observation.

GitHub’s automatically generated “Source code” ZIP/tar files follow the release tag. They contain the modern repository and tooling, **not the legacy executable source tree**. To reproduce this observer, use the authenticated `legacy-source.tar.gz` and `observer-dependencies.patch` inside `observer-release.tar.gz`, following the [pinned build guide](https://github.com/cambridgetcg/zerone-core/blob/ed364bc0afec062612b61a309002f23da85acf2c/deploy/legacy-observer/README-build.md). The release tag may identify the final website/report commit; the package tooling remains pinned separately to `ed364bc0afec062612b61a309002f23da85acf2c`.

## Source and retained risk

The original live executable, SHA-256 `94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8`, was reproduced byte for byte from the stated application base using Go 1.24.13. Its original VCS metadata was absent; reproduction does not recover the original authoring checkout.

The distributed observer is a different executable. It applies the manifest-authenticated two-file `go.mod`/`go.sum` patch and uses Go 1.25.14 with CometBFT 0.38.25. The other 655 files in the 657-file compile closure remain unchanged. Two fresh-cache builds on the same native host produced the executable digest above. This is repeatability by the same operator, not reproduction by an independent organization.

The dependency change addresses the upstream [Tachyon consensus issue](https://github.com/cometbft/cometbft/security/advisories/GHSA-c32p-wcqj-j677). It does not patch the running validator. The dated binary assessment retains 42 reported advisory IDs, including 11 with exact function-name matches; function presence alone is not a runtime exploitability finding, and absence of a matching name does not clear a dependency. SDK/IBC and consensus-sensitive application semantics are retained. The manifest-authenticated `VULNERABILITY-REVIEW.json` discloses remaining findings; no complete application or operating-system security audit is claimed.

## Trust and verification status

| Check | Status for this exact package |
| --- | --- |
| Source closure, two-file patch and repeated executable build | Reviewed; matching captured results |
| Manifest and package signature/file verification | Local signature/file verification passed |
| Snapshot restore and fresh applied-state advancement | v2 followed 1,262,486 → 1,262,487 → 1,262,488; clean first stop recorded |
| Application root at H matched to authenticated header H+1 | Passed at H 1,262,488 and H 1,262,498 |
| Same-home clean restart, zero power and unused signing state | Passed: same identity and controls; observed heights 1,262,496 → 1,262,497 → 1,262,498; exit 0, no OOM, owned container removed |
| `broadcast_tx_async`, `broadcast_tx_sync`, `broadcast_tx_commit` refusal | All three returned `nop` mempool refusals in both phases |
| Signed rehearsal receipt and local archive equality | Main signature, exact manifest/archive binding and required checks verified; receipt completed 2026-09-09 23:37:45 UTC |
| Active validator and production authority unchanged | Checked at 2026-09-09 23:35:17 UTC, live H 1,262,500: original executable and sole-validator key/power unchanged |

The offline root check reuses the packaged CometBFT 0.38.25 checkpoint verifier for both captured commits and complete validator sets. Its pair wrapper also checks each block body and part-set, the full adjacent block ID, the carried previous commit, validator continuity, and equality between the operator-observed application root at H and the signed header at H+1. It does not itself inspect the durable local database or establish an ancestry path back to the bootstrap checkpoint. The first phase completed at 2026-09-09 23:28:52 UTC. Its applied root `f3cea15b4a45737d284bfd1eb8bd51b4e7df6c09d853b1b4199801f8c847f6d6` at H 1,262,488 equals the app root in signed header 1,262,489, block `92579EA29991C790323F6618ED53124CF8CA0D852CD612A7C3E469F376FD3C26`. The restart phase completed at 2026-09-09 23:33:51 UTC. Its applied root `89f2d7236b108d4e04c0a7eee047cc5304203eb376aef71dad7728ccfede2709` at H 1,262,498 equals the app root in signed header 1,262,499, block `F16AFF7B73241C74485269F5A73D2B04E001067BDFB8CDFB911907F98C57BE19`.

A separate [prior-census revalidation](legacy-observer-census-revalidation-2026-09-10.json) reran the patched signature/address checks for the five retained snapshot/census headers and their adjacent links. The captured census root at H 1,261,575 still matches signed header 1,261,576; the specific retained commitments passed. This recheck did not reopen the census database, replay history, or establish ancestry between the earlier snapshot and census. The companion JSON SHA-256 is `32e2201d0bbe9293728d7c8f4907178ba6495e3359daa974c6d63032bede12df`.

The checkpoint is selected by the existing operator and checked against the disclosed sole-validator key. Both state-sync RPC aliases share one upstream. Another replica does not provide an independent trust anchor or repair the legacy signer-custody finding. Snapshot synchronization is not a full historical replay or proof of every historical payment.

The package grants no validator membership, new account, funds, reward, transaction admission or upgrade authority. It does not modify the current validator or activate a successor network. Current main is not a drop-in replacement for the live legacy executable.

New bootstrap closes at the signed expiry. A previously initialized and successfully synced home may resume with its original verified package; an unsynced home cannot use that exception. There is no automatic unsigned checkpoint refresh.

This report records source, build, local signature verification and exact-package rehearsal evidence. Public release-asset downloads, website deployment and production browser checks are separate publication checks, recorded after publication; they are not claimed here. The website release record is admitted only after the signed rehearsal passes. Existing public reads, wallet tools and the local sandbox remain separate participation choices.

The first signed candidate stopped before state sync because its helper rejected the real daemon's omitted zero ABCI height and SHA256-empty startup hash. The two failed attempts stopped cleanly without local signatures or an owned running container. The reviewed fix accepts only the exact empty startup state while keeping it unready; chain/identity/power checks and two advancing positive observations are unchanged. Version 1 remains immutable and unpublished. Version 2 authenticates this helper correction and the maintenance qualification; its actual sync and retained-home restart subsequently passed as recorded above.

For provenance, the executable build recipe and both reproduced ELF receipts remain pinned to `84a1f7304515b52da002ebd7e2da782b5aef1634`; the v2 package/helper source is `ed364bc0afec062612b61a309002f23da85acf2c`. The v2 archive is 43,975,496 bytes. The unpublished v1 record is release `zerone-1-observer-20260909.1`, manifest `5179893f2c831c689e5a7d34e57047530407006d3f2707e40cd3caee98a71a34` and archive `44e499ed1ba15d70bb7f92a91c944d6714577e5a0a528dcecb522df9140c7528`; none of those hashes are being relabeled as v2.

The separate offline pair review used 14 passing test groups (70 cases including subtests, with the packaged checkpoint tests reused). Its first-phase review receipt SHA-256 is `f8d21022bc61e837868c098c9fc3dcee3fcc1c995ff1e44bc588ef114c3908a2`; its restart review receipt is `34e25d836cdc5c35e56551f1c9cea923c2598edd67366ea9ed471c50ef178d46`. These are integrity references to retained review evidence, not claims of independent control of the network.
