# Join the Zerone-2 read beta — STAGED, NOT LIVE

This guide is preparation, not an announcement that public Zerone-2 is open.
There are deliberately no public endpoint addresses here. Source publication,
a passing fixture rehearsal, a dashboard profile, and migration policy design
are **not** signed OPEN authority. The default dashboard network remains legacy
until the signed OPEN gates are satisfied.

## Before joining

Obtain the published, signed release and OPEN-BETA artifact set through the
release's authenticated publication channel. Follow [CANONICAL-SIGNING.md](CANONICAL-SIGNING.md)
and [AUTHORITY-BUNDLE.md](AUTHORITY-BUNDLE.md), with an independently established
operator signer fingerprint. Verify the complete required authority chain, not
merely a checksum obtained from the same server as its file. Stop if any required
signature, artifact, pinned image/binary digest, genesis hash, chain ID, or OPEN
binding is missing or disagrees. Examples and unsigned templates are not inputs.

Use only the binary/platform and genesis bytes bound by that verified release.
Pin the exact genesis SHA-256 and `chain_id = zerone-2`; retain the public
`network-manifest.json` and `GENESIS-MANIFEST.md` for their supply, custody and
launch-policy disclosures. Do not substitute legacy `zerone-1` genesis or a
locally generated `drill` genesis. Never use public rehearsal keys with real
assets or production nodes.

The operator must publish the exact permitted non-validator edge peer IDs and
addresses under OPEN. If that publication is absent, **wait**; do not guess an
endpoint, repurpose a legacy endpoint, or connect directly to the validator.

## Use your own fresh, non-validator node

After those gates pass, use a new home and the verified binary. Keep the home
outside a public web root. Never copy a validator home, `data/` directory,
`priv_validator_key.json`, `priv_validator_state.json`, or someone else's keys.
Do not import a snapshot as a substitute for the fresh replay check.

The following is a template, not a ready-to-run launch command. Supply paths and
peer values only from your own setup and the verified OPEN publication. Check
that the local ports are unoccupied first; do not terminate an existing process
to make room.

```bash
# VERIFIED_BINARY, VERIFIED_GENESIS, JOIN_HOME and SIGNED_EDGE_PEERS are required.
# VERIFIED_GENESIS must already match the signed genesis SHA-256.
test ! -e "${JOIN_HOME:?choose a new home}" || exit 1
"${VERIFIED_BINARY:?}" init read-beta --chain-id zerone-2 --home "$JOIN_HOME"
cp "${VERIFIED_GENESIS:?}" "$JOIN_HOME/config/genesis.json"
"$VERIFIED_BINARY" genesis validate "$JOIN_HOME/config/genesis.json"

# Keep local query/broadcast access on loopback. No hosted broadcast gateway.
"$VERIFIED_BINARY" start --home "$JOIN_HOME" \
  --rpc.laddr tcp://127.0.0.1:47657 --rpc.unsafe=false --rpc.pprof_laddr "" \
  --p2p.laddr tcp://127.0.0.1:47656 \
  --p2p.persistent_peers "${SIGNED_EDGE_PEERS:?verified OPEN peers required}" \
  --p2p.pex=false --api.enable=false --grpc.enable=false --grpc-web.enable=false \
  --minimum-gas-prices 1uzrn
```

Check your node's `/status` for `network = zerone-2`, advancing height and
`catching_up = false`. Compare your local node's genesis content with the
**authenticated, release-pinned genesis artifact** obtained above, accounting
for the SDK/Comet JSON envelope difference rather than accepting a different
genesis. The public edge exposes P2P only; its RPC is private, and the public
query gateway deliberately serves neither `/genesis` nor chunked genesis.
Do not open those listeners or widen the gateway to complete this check.

At the **same retained height**, compare block ID and header AppHash with the
published authenticated checkpoint and the permitted `/block` or `/commit`
point reads through the published query gateway. AppHash in header H anchors
state H−1; do not compare independently sampled tips or call a REST balance a
Merkle proof. Matching responses from several nodes following one validator
do not create several independent consensus authorities.

Stop your own node cleanly, retain the same home, restart it, and confirm it
catches up and still returns the same historical checkpoint. That is a restart
and replay diagnostic, not a stopped-directory production disaster-recovery
proof. Keep RPC private; never enable unsafe RPC just to join.

## Reads first; no automatic funding

Joining or registering an identity does not grant coins, a faucet entitlement,
a validator position, or an automatic reward. Use your own local node for
queries and any explicitly authorized signed transaction. Do not send keys,
identity secrets, recovery phrases or signed transactions to a hosted dashboard.
The read observer is not a hosted transaction broadcaster.

Only **after signed OPEN**, an operator may conduct the separately authorized
one-off pilot: the total OPERATIONS float debit, **including the funding
transaction's fees**, must be at most **10 ZRN = 10,000,000 uzrn**. This is a
ceiling for one pilot, not a recurring allowance, faucet or permission to fund
all joiners. The operator must approve the recipient and exact amount, reconcile
the float before and after, and record transaction hashes, success codes,
sequence changes and fee/balance arithmetic. User onboarding/DID registration
and an ordinary MsgSend must fit inside that funded amount. Never run the user
pilot during real DARK registration.

The launch operator's identity onboarding and required **111 ZRN custom-validator
escrow** are separate launch-bootstrap obligations funded from the validator
allocation. They are not paid from, waived by, or counted as this user pilot.
The local dress rehearsal models that separation using public fixture accounts;
its successful transactions are not real funding or a public launch claim.

## What this beta does not promise

- **One custodial validator; f = 0.** There is no Byzantine fault tolerance or
  independent validator quorum. Losing the validator stops progress; an edge or
  your own node is not a replacement signer or an availability guarantee.
- **Reset/history boundary.** Zerone-2 is a new genesis and chain ID, not a
  continuous import of the predecessor ledger. Follow the signed final-checkpoint
  and archive disclosures for predecessor history. Retain the genesis/checkpoint
  you trusted; any later reset requires an explicit new disclosure, not silent
  substitution of a home or genesis file.
- **No automatic asset migration.** Historical snapshot allocation and nominal
  1:1 are policy-design terms only, not an activated claim, reserve, balance copy,
  legacy-asset extinguishment, price guarantee or user entitlement. Keep legacy
  assets and custody separate until a separately authorized mechanism exists.
- **Dark economic surfaces remain dark.** No substrate adapters or claiming pots
  are enabled by joining. Reward settings and capped-mint telemetry must retain
  the released dark profile; fee movement is not new issuance. Fixed supply in
  a short fixture run is not an audit of every future economic path.

See [GO-NO-GO.md](GO-NO-GO.md), [CUTOVER.md](CUTOVER.md) and
[LAUNCH-COMMUNICATIONS.md](LAUNCH-COMMUNICATIONS.md) for the authoritative gates
and limits. Stop on disagreement; this guide never overrides them.
