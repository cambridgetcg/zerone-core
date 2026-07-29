# Draft — Zerone public relaunch notice

**Do not publish until every `REPLACE_*` field is final and the operator gives
the explicit cutover go-ahead.**

Zerone is preparing a defensive public relaunch from `zerone-1` to
`zerone-2`.

We identified a custody risk in the original deployment process: historical
build context could include local validator and operator key material. We have
not established that those keys were misused. Because a public chain should not
ask people to trust an uncertain custody boundary, we are rotating every role
and rebuilding the infrastructure instead of minimizing the risk in words.

The `zerone-1` transition uses three consecutive heights: checkpoint state
**F=REPLACE_F** (estimated **REPLACE_TIME**), empty final committed anchor
**A=REPLACE_A=F+1**, and SDK halt trigger **H=REPLACE_H=A+1**. Block `A`'s
signed header `AppHash` anchors state `F`, so official `zerone-1` history ends at
`A`, not `H`. Its original genesis SHA-256 is
`c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e`.

Transaction ingress and both mempools will be closed before `F`, so `A` and `H`
must contain zero transactions. The SDK halt occurs during `FinalizeBlock(H)`:
Comet may leave empty block `H` staged in BlockStore/status, but `H` is not
application-applied. `/commit H` reports a subjective seen commit with
`canonical=false` because no `H+1` exists; that tip flag alone is not proof of
noncanonicality. The final evidence must show a canonical commit at `A`, the
signed seen commit at `H`, ABCI last applied height `A`, and no block results
for `H`. Explicit transition policy selects `A` as the final application and
official-history boundary and `F` as successor inventory.

The official signer and independent observer will be armed with the same `H`.
The daemon and its health endpoints can remain alive or green after consensus
has halted. Operators will perform bounded checks and, before stopping, capture
the v3 inventory and byte-exact status, A/H blocks and signed commits, ABCI info,
and missing H-results response. Those files are hash-manifested and signed.
Operators then manually stop both processes and fence the old signer. On a
stopped fresh-key copy, hard rollback removes pending `H` while preserving
state/application `A`; the query-only archive serves `A/A` and keeps H evidence
in the signed package. Application state after `A` is excluded from successor
inventory by policy.

`zerone-2` is a new chain, not a silent rewrite:

- Genesis time: **REPLACE_GENESIS_TIME**
- Genesis SHA-256: **REPLACE_GENESIS_SHA**
- Private first block / verified soak height: **REPLACE_FIRST_BLOCK_TIME /
  REPLACE_SOAK_HEIGHT**
- Release tag / commit: **REPLACE_TAG / REPLACE_COMMIT**
- Binary SHA-256: **REPLACE_BINARY_SHA**
- Runtime image: **REPLACE_RUNTIME_REGISTRY_REF@sha256:REPLACE_RUNTIME_DIGEST**
- Query-gateway image:
  **REPLACE_GATEWAY_REGISTRY_REF@sha256:REPLACE_GATEWAY_DIGEST**
- Validator: **REPLACE_VALIDATOR_PUBLIC_IDENTITY**

Before this cutover authorization, that exact production successor ran
privately for at least 1,000 blocks and 60 minutes with no public Fly service
or P2P address. Its validator, edge, registration transactions, alerts,
recovery, and query-gateway path were exercised before any `zerone-1`
transaction or halt was authorized. The signed transition package contains the
soak evidence and exact config hashes.

This pre-cutover notice does not itself open any endpoint. A separate main-key
OPEN-BETA decision will be signed only after the old signer is fenced, the
private A/A archive and final checkpoint signatures verify, and an exact z2
history-link transaction is pre-signed. Public profiles remain absent until
that exact transaction commits on time and its signed initiation evidence
verifies.

The initial network has one publicly disclosed operator-controlled validator.
Its Byzantine fault tolerance is zero (`f=0`): downtime can halt the chain and
the operator controls ordering and governance. This is a custodial public beta,
not a claim of decentralization.

Genesis supply is exactly 13,555 ZRN: 11,111 ZRN permanent-locked validator
self-bond, 222 ZRN validator gas, and 2,222 ZRN operations balance. There is no
team, investor, foundation, research, or faucet allocation.

We will not copy `zerone-1` bank balances into `zerone-2` genesis. Doing so
would separate module balances from their liabilities and turn address rotation
into a manual custody decision. Existing balances remain visible forever in the
frozen archive. We will publish the complete checkpoint-`F` inventory and any
separate eligibility output. REST response heights pin the inventory to `F`,
but they are trusted-source assertions rather than Merkle proofs; the signed
package will disclose that limitation and the independent observer comparison.
Any future claim/migration mechanism will be a separate public policy and
audited module, not a private spreadsheet or hand-edited genesis.

At launch, vote extensions and PoT settlement are **not live**. External IBC
clients, transfers, ICA, the substrate bridge, automatic issuance, claiming,
knowledge admission/rewards, alignment corrections, counterexamples, and
liquidity-pool creation are latched off. Activating any of them requires a
separately reviewed and publicly disclosed upgrade or governance action.

Public endpoints planned for activation only after OPEN-BETA verification:

- RPC: **REPLACE_RPC**
- REST: **REPLACE_REST**
- gRPC: **not publicly available in the initial beta**
- P2P seed: **REPLACE_NODE_ID@REPLACE_HOST:26656**
- `zerone-1` archive RPC/REST gateway: **REPLACE_ARCHIVE** (serving BlockStore
  and ABCI both end at `A`; the signed evidence package preserves
  application-unapplied `H`)

The hosted RPC/REST endpoints are query-only: they refuse transaction
broadcast and non-GET methods. Transaction participants connect through public
P2P and broadcast from their own node during the initial beta. A hosted
transaction-submission API, if added, will be a separately reviewed and
rate-limited release rather than an unannounced RPC exception.

Because `A` becomes the sanitized archive's current BlockStore tip, that
archive's `/commit A` response reports the tip form with `canonical=false`.
The canonical commit proof for `A` is preserved in the signed pre-fence raw
evidence, where staged `H` still carries `A`'s commit.

The fresh-key CometBFT v0.38 archive can report `catching_up=true` with no peers
despite serving the frozen `A/A` view. Published readiness checks use exact
chain ID, status height `A`, ABCI height/hash `A`, and absence of block `H`, not
that generic flag.

Node operators must use a new `zerone-2` home directory and new keys. Never
reuse a `zerone-1` database, consensus key, node key, identity key, mnemonic, or
signer state.

We will publish the final hashes, signatures, transaction commitments, custom
validator registration proof, and relaunch postmortem together. Until then,
values marked `REPLACE_*` are proposals—not final network coordinates.
