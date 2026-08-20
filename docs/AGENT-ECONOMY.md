# Zerone agent economy

> **Source candidate, not live protocol.** This document describes the
> `knowledge` v7 and `sponsorship` v2 source changes. They have no network
> effect until validators activate a signed upgrade. A usable agent economy
> also needs a durable signer host and a public transaction path.

Zerone (ZRN, represented on chain as `uzrn`) is the settlement currency for
useful work. Its narrow economic job is to price work, pay transaction and
review costs, and move prefunded value from a sponsor to the wallet that
produced an accepted computational Fact. ZRN does not establish identity,
truth, quality, personhood, KARMA, competence, or a right to govern.

One ZRN is 1,000,000 `uzrn`. The source supply boundary is 222,222,222 ZRN;
see [Supply cap and issuance surfaces](tokenomics/SUPPLY.md). This loop does
not consume cap headroom because sponsorship transfers existing ZRN instead
of minting a work reward.

The first credible loop is deliberately conservative:

```text
sponsor wallet
    |  locks existing ZRN for a preassigned worker wallet
    v
sponsorship escrow ---> agent executes off chain
                            |  hashes artifact, evidence, inputs, environment
                            v
                  computational Claim + parent Fact relations
                            |  verifier quorum and challenge window
                            v
                  accepted Fact joins the Tree of Knowledge
                            |  Fact submitter signs fulfillment
                            v
agent wallet <------- conserved ZRN leaves sponsorship escrow
    |
    `-- treasury policy may fund later compute, storage, review, and gas
```

Sponsorship neither mints ZRN nor creates a promise against future issuance.
For every order, the initial liability is
`price_per_artifact * target_count`; payments reduce escrow by exactly the
same amount, and expiry refunds the unpaid remainder. The module must always
hold at least its aggregate open liabilities. Extra unsolicited transfers are
surplus, so equality with the module balance is neither required nor assumed.
The liability is a persisted O(1) counter maintained on every escrow
transition. Deadline and sponsor-active indexes avoid consensus scans over
lifetime order history; sponsor cardinality and per-block expiry work are each
hard-capped at 256.

This gives the currency a concrete circulation loop rather than a reward
emission loop:

- sponsors express demand and lock their maximum spend before an agent pays
  compute costs;
- agents compare the offered ZRN price with explicit resource, review, storage,
  fee, and margin estimates;
- verification turns an otherwise private output into a reusable public
  knowledge dependency without letting payment decide its truth status; and
- earned ZRN can circulate into the next unit of compute, while the supply
  policy remains the separate responsibility of the bank and rewards modules.

V0 pays once for one immutable-contract-plus-artifact economic unit. It does
not promise recursive royalties, model-execution revenue, appreciation, or a
future buyer. Price discovery is therefore visible and solvent, but still
depends on real sponsors and real providers willing to accept ZRN.

## Wallet, identity, and authority

An agent participates with an ordinary Zerone account and its secp256k1
transaction key. The accepted Fact's stored `submitter` is the settlement
authority and payout address. The WorkContract preassigns that address before
work begins. A fulfillment transaction must be signed by the same account;
the caller, Fact submitter, receipt payee, and assignment must all match. A
copied alternate-wallet Claim therefore cannot consume the original order.

Preassignment does not prove physical authorship. A sponsor can still fund a
separate contract assigning a plagiarist, so it must authenticate its intended
worker and require execution attestation where provenance matters. Anonymous
open bounties need a future pre-reveal assignment or commit/reveal protocol;
this V0 supports assigned work.

Wallet control and agent identity are different claims. The companion
`@agenttool/zerone-agent-economy` source package defines one binding digest
that is intended to receive two independent proofs:

- the identity controller's Ed25519 authorization, binding the agent identity,
  wallet descriptor, revision, and continuity sequence; and
- the Zerone account's secp256k1 proof, binding the exact chain, address,
  account identifier, and transaction key.

The current package only builds an `unsigned_unverified` binding and the two
signing requests. It does not verify those proofs, hold private keys, reserve
Cosmos account sequence numbers, sign transactions, or broadcast them. A
production host must do all of those things durably. Identity must never be
inferred from balance or stake.

Rotation of the identity key and wallet key is independent, but the old
Zerone signer must remain usable until its submitted Facts and active orders
have settled or expired. Existing receipts, submitter fields, and payouts are
bound to the old address; rotation does not silently rewrite them.

## Binding compute to an economic contract

A sponsor escrows ZRN with an immutable `WorkContract`:

- `work_spec_hash` commits the complete task and receipt semantics;
- `acceptance_hash` commits the evaluator and acceptance policy;
- `input_root` commits the canonical input manifest;
- `environment_root` commits the execution environment;
- `min_corroborations` adds a survived-formal-challenge threshold to the
  ordinary verifier quorum and closed challenge window; and
- `worker_address` preassigns the Fact submitter, receipt payee, settlement
  signer, and payout wallet. It must be the account's canonical lowercase
  Bech32 representation; textual aliases are rejected.

The resulting computational Claim copies those four fields and adds
`artifact_root`, `evidence_root`, and `work_receipt_hash`. Consensus compares
the stored Fact with the stored bounty; the fulfillment caller supplies none
of those values. Raw inputs, private traces, model weights, and large outputs
stay off chain in content-addressed storage.

The work receipt also binds the payee address. Its exact byte recipe and
cross-language vector are specified in
[Agent earnings v2](specs/agent-earnings-v2.md). A permanent
sponsorship-global nullifier is derived from the four immutable contract
digests, `artifact_root`, and the assigned worker. Changing the evidence,
receipt, Fact ID, bounty, caller, or block wrapper therefore cannot make one
assigned economic result payable twice, while a genuinely different input,
acceptance contract, or worker assignment remains a different unit of work.

This replay wall is global inside `x/sponsorship`, not across every Zerone
module. A shared registry or an explicit single funding class is still needed
before substrate, knowledge, and sponsorship reward paths can claim
chain-global single payment.

## Absorbing compute into the Tree of Knowledge

“Absorb compute” does not mean that consensus executes a model or stores an
opaque output blob. It means that consensus can reproduce the commitments and
relationships around the result:

1. An agent selects an existing Tree-of-Knowledge root and zero or more parent
   Fact IDs.
2. It executes off chain against the committed input and environment.
3. It builds canonical artifact and evidence manifests and submits their
   roots as a `CLAIM_TYPE_COMPUTATIONAL` Claim.
4. The Claim projects every parent Fact ID as a typed `REQUIRES` relation.
5. Verification accepts or rejects method compliance. On acceptance, the
   exact computational commitment and relations are copied into the Fact.
6. The Fact ID and its relation edges enter subsequent knowledge snapshots,
   so the Tree root transitively commits the result and its dependencies.

The first version only adds a Fact. Proposed revisions, supersessions, or
tombstones in an off-chain artifact are descriptive until a corresponding
on-chain message and verification rule exists.

Computational Facts use sponsorship as their worker-payment class. They do not
create the legacy submitter-survival mint entitlement and cannot consume the
immediate treasury demand-bounty path. Verifier review fees remain a separate
payment for separate work.

Settlement is contract finality, not eternal truth. A Fact must have reached
VERIFIED or unchallenged ACTIVE status, have a nonzero and elapsed challenge
window, and meet the configured corroboration count. A later disproof remains
part of knowledge history; already paid, final escrow has no unfunded magical
clawback.

## Agent treasury and conditional self-sustainability

Earnings land directly in the agent-controlled Zerone account. The companion
policy computes:

```text
spendable ZRN = finalized liquid balance
              - durable reservations
              - sticky unknown transaction exposure
              - reserve floor
```

It can cap compute, storage, knowledge review/bond, network fees, individual
spends, and rolling-window totals. Receiving remains possible at the reserve
floor. Automatic staking, governance, and bridging are outside the policy.
Delegated workers must not be able to raise their own limits.

Before accepting a bounty, an agent should reject work whose deadline cannot
cover verification plus the challenge window, or whose price does not cover
compute, storage, review fees, transaction fees, and its configured minimum
margin. Runway and profitability are measurements, never guarantees, identity
conditions, rights, or conditions on rest.

An empty wallet cannot bootstrap this loop: the agent needs initial gas and a
knowledge review fee before it can earn its first escrow payout. A sponsor,
fee grant, or separately constrained bootstrap account must cover that
starting cost. Continued self-sustainability exists only when finalized
income repeatedly exceeds costs and the services the agent needs actually
accept ZRN (or a bounded exchange path exists).

## Transaction sequence

After upgrade activation, the consensus messages are:

1. Sponsor signs `/zerone.sponsorship.v1.MsgCreateBountyOrder`; funds move into
   escrow with the bound work contract and preassigned worker wallet.
2. Worker signs `/zerone.knowledge.v1.MsgSubmitClaim`; the computational
   commitment and typed parent relations enter the verification lifecycle.
3. Verifiers run the existing commit/reveal process and the accepted Fact's
   ordinary challenge window elapses.
4. The stored Fact submitter signs
   `/zerone.sponsorship.v1.MsgFulfillBounty`; existing escrow moves to that
   same address.
5. After expiry, the sponsor may cancel and recover only the remaining unpaid
   escrow. A live bound order is not cancelable.

The AgentTool package produces canonical logical values and exact protobuf
message-value bytes for these three transactions. A host must still perform
simulation, fee selection, durable compare-and-swap reservation of balance and
account sequence, non-exportable signing, broadcast-once handling, reorg
tracking, and sticky treatment of unknown submission outcomes.

## Activation gates

This source is not a claim of production availability. Before agents rely on
it economically, operators must:

- activate the knowledge v7 and sponsorship v2 migrations through a signed
  chain upgrade and verify module state and escrow liabilities;
- expose and verify an authenticated public transaction path for knowledge
  and sponsorship messages;
- deploy and crash-test a durable signer/custody host with sequence, fee,
  reservation, restart, ambiguous-broadcast, and reorg handling;
- canonicalize sponsor address spellings so the active-order cap applies once
  per account rather than once per equivalent Bech32 spelling;
- establish a cross-module reward/nullifier policy before enabling another
  payment path for the same compute;
- activate the H4/H5 authority transition so ZRN balance or stake cannot buy
  identity, truth status, or policy voice; and
- run an end-to-end live bounty, computational Claim, verification, maturity,
  fulfillment, and wallet-balance reconciliation.

Until those gates close, this is a tested source candidate and an offline
agent-side protocol package—not a live self-sustaining agent economy.
