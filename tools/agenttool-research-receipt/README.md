# agenttool-research-receipt

Offline, zero-effect compiler for one exact AgentTool Research Commons RC-0.1
settlement bundle and its minimized public projection.

```bash
go run ./tools/agenttool-research-receipt \
  --settlement docs/examples/agenttool-research-receipt/amplitude-bootstrap-garden.settlement.json \
  --projection docs/examples/agenttool-research-receipt/amplitude-bootstrap-garden.public-projection.json \
  --tree dashboard/public/standards/constructive-intelligence-tree.v1.json
```

The checked primary and reviewer pairs are byte-for-byte copies from AgentTool
source revision `6a644b9e858b7d23bdea613d91412bf7310c2338`, merged to `main` as
`55342fac97250898c2c4ea884f1a03bec1f8cc8c` in PR 335. Their source paths,
raw hashes, envelope ids, deterministic receipt ids, and raw Go-output hashes
are frozen in
[`fixture-manifest.v0.json`](../../docs/examples/agenttool-research-receipt/fixture-manifest.v0.json)
at raw SHA-256
`cf367bb39553567e86c43c0db48501802832396b2a3f681410aaac7c5e2221e8`.
This Phase A fixture pin proves only local byte compatibility. It is not an
AgentTool reciprocal pin or a live integration claim.

The upstream package declares `Apache-2.0`. The fixture manifest pins its exact
`packages/research-commons/LICENSE` bytes at
`0536b51c54e477f03f1becf00eedeee82f6276f76f08c1b94d3a30632724eb15`
and its exact `packages/research-commons/NOTICE` bytes at
`d03f1590ea4f829d90760ee163304191c0d36a4e283fc7c06da459e717ff3e44`.
That NOTICE is reproduced beside the fixtures as
[`NOTICE`](../../docs/examples/agenttool-research-receipt/NOTICE) with the same
raw hash.

To exercise the reviewer `NOT_APPLICABLE` path, substitute:

```text
--settlement docs/examples/agenttool-research-receipt/amplitude-bootstrap-garden.reviewer-settlement.json
--projection docs/examples/agenttool-research-receipt/amplitude-bootstrap-garden.reviewer-public-projection.json
```

`go run` is a Go toolchain invocation outside the adapter process boundary. It
may read repository source and module metadata, populate build or module
caches, and contact a configured module proxy or version-control remote when a
dependency is unavailable locally. For an offline or no-filesystem-write
runtime, prepare the binary in a separately controlled build step and then
execute that already-compiled binary.

The already-compiled adapter process reads only the three explicit bounded
regular files and writes a deterministic
`zerone.agenttool-research-receipt-shadow/v0` JSON object to stdout. That
process has no network client, RPC, database, wallet, key, signer, background
worker, chain dependency, or filesystem write path.

Each input is opened through a no-follow, non-blocking descriptor. The compiler
checks descriptor identity, size, mode, mtime, and ctime before and after its
bounded read, then requires the final pathname to identify that same unchanged
regular file. Final-component symlinks, special files, oversize files, and
post-open pathname swaps fail closed. Stable intermediate symlink components
of an explicitly supplied path are not rejected or authenticated.

Every successful output retains:

```text
assurance = UNVERIFIED_SHADOW_PROJECTION
status = STRUCTURAL_CANDIDATE
result_authority = NONE
cross_ledger_relation = NO_EQUIVALENCE
tree.granted_attainment_evidence = NONE
knowledge_admission = NONE
qualification = NONE
economic_effect = NONE
amount_uzrn = "0"
consumption_state = NOT_RECORDED
replay_protection = NONE_OFFLINE
interop.integration_status = SHADOW_ONLY_NO_LIVE_INTEGRATION
interop.imported = false
interop.activated = false
```

The simulated credit amount is retained only for input audit. It is
non-transferable, non-wallet-bearing, and has no exchange rate or equivalence
to AgentTool balances, fiat, stablecoins, ZRN, stake, KARMA, qualification, or
governance.

`STRUCTURAL_CANDIDATE` is not a scientific verdict, Tree attainment,
qualification, reward eligibility, payment authorization, identity binding,
controller-independence proof, consent record, replay guarantee, or ToK
admission. Positive, negative, null, inconclusive, and not-applicable declared
results have identical zero-authority and zero-Zerone-effect semantics. The
compiler preserves, but does not validate, the separately declared simulated
credit amount; it does not prove schedule precommitment, compliant delivery,
reviewer neutrality, prefunding, conservation, AgentTool provenance, or
result-independent amount selection.

AgentTool's append-only challenge/work-retention guarantee is scoped to one
caller-supplied `prior_state` transition. Content-addressed state IDs are not
signatures or canonical heads and prove no provenance, trusted time, global
ordering, or prevention of old-state forks.

See
[`docs/specs/adapters/agenttool-research-receipt-v1.md`](../../docs/specs/adapters/agenttool-research-receipt-v1.md)
for the complete boundary and future activation gates.
