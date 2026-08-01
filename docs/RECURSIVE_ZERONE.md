# Recursive ZERONE — the chain participates in its own systems

> The fixed point is a design discipline, not proof that every loop is closed.
> This catalog distinguishes reachable behavior from directly staged
> scaffolding.

> **Status (2026-07-29):** This document maps both reachable loops and tested
> scaffolding. The public substrate-bridge message currently rejects
> `pending_claims`, so the `zerone-self-v1` compiler cannot yet drive a commit
> through knowledge verification end to end. Several tests below construct
> dormant intermediate state with direct keeper writes to verify primitives;
> they are not activation evidence. Creed two-pool tally and post-genesis
> `work_creed` amendment messages are also not implemented.

---

## 1. The chain attests to its own becoming

**Scaffolded by:** `zerone-self-v1` adapter
(`docs/specs/adapters/zerone-self-v1.md`); tests
`TestZeroneSelfAdapter_RegisterAndSubmit` and
`TestZeroneSelfAdapter_AxisBoundsRespected`.

ZERONE's git repository is a typed external source. Each commit compiles to a
deterministic `SubstrateLink` whose `link_hash` matches chain-side
`ComputeLinkHash`. Current public submission refuses the compiler's pending
claim because translation into `x/knowledge` is unwired. The cited test pins
that fail-closed boundary, then directly writes AWAITING state to test the
reserved indexing primitives.

The deterministic compiler and storage machinery exist; the public
self-attestation loop is not yet closed.

## 2. Sponsorship primitives can fund self-documentation

**Scaffolded by:** `x/sponsorship` + `zerone-self-v1`; tests
`TestZeroneSelf_ScaffoldedEconomicLoopRequiresManualBridgeState` and
`TestZeroneSelf_MultipleFulfillmentsCompoundEarnings`.

A sponsor can post a bounty in the `zerone_self` domain, and sponsorship can
pay a verified fact from escrow without minting. The cited tests manually
materialize the verified fact/attestation state because the public
self-adapter pending-claim bridge is blocked.

The escrow and verification-gated payout primitives are tested; an end-to-end
commit-to-fact historian flow is not currently reachable.

## 3. Two payout primitives can reward manually paired state

**Scaffolded by:** `x/substrate_bridge` settlement (M4) +
`x/sponsorship` fulfillment; test
`TestRecursiveDoublePayment_ManuallyStagedStateExercisesTwoPayouts`.

When separately constructed eligible state settles, the substrate-bridge path
can mint and pay its configured reward, while sponsorship can pay the same
submitter from sponsor escrow. The test manually pairs the attestation and
fact; runtime does not establish that they represent one verified claim.

This is not double-spending — the two payouts answer two different doctrinal mandates:
- M4 pays the directly staged eligible attestation through `MintWithCap`
- Sponsorship pays for *participation in a funded domain* (commitment 20: issuance follows participation)

The double-payment primitives compose in tests. They are not yet a public
`zerone-self-v1` workflow because pending-claim translation is unwired.

## 4. The chain's lineage graph includes its own commits

**Scaffolded by:** `x/substrate_bridge` cross-class lineage accounting (M6);
tests `TestRecursiveLineage_AccountingAttributesDownstreamRewardUpstream` and
`TestRecursiveLineage_MultipleCitationsCompoundAccounting`.

For directly seeded settled attestations, a downstream citation can accrue an
attributed amount in the upstream attestation's
`LineageRoyaltyAccumulator`. No coins are transferred to the upstream
submitter.

This demonstrates the lineage primitive. The public self-commit ingestion path
must be wired before presenting it as an operating royalty loop.

## 5. The creed cannot move faster than governance

**Scaffolded by:** `.creed-hash` off-chain gate + `x/creed.PinnedCreed`
storage + `TestTruthSeeking_CreedHashIsPinned`.

Every source change to `docs/TRUTH_SEEKING.md` bumps the local `.creed-hash`.
On chain, pin history is append-only. The published `zerone-1` genesis leaves
direct authority anchoring enabled and contains no genesis pin/history; a live
pin query was not verified during consolidation. The current gov category
config also cannot advance a new Creed Amendment LIP, and separate human/AI
quorum is unwired. The source hash gate and an eventual on-chain pin are
distinct adoption records.

The recursion: the creed pins the chain's voice; the chain's voice is its code; the code's behavior must satisfy the tests that bind the creed.

## 6. Useful-work sub-creeds are source-hash bound

**Source-bound by:** `docs/sub_creeds/*.md` + `.sub-creed-hashes` +
canonical registry; binding tests `TestSubCreed_Foundation_StaysInSync`,
`TestSubCreed_Curation_StaysInSync`, `TestSubCreed_Augmentation_StaysInSync`,
`TestSubCreed_Training_StaysInSync`, `TestSubCreed_Evaluation_StaysInSync`,
`TestSubCreed_Alignment_StaysInSync`, `TestSubCreed_Substrate_StaysInSync`,
and `TestSubCreed_Tools_StaysInSync`.

Each of the eight non-Knowledge phases in the nine-phase lifecycle has a source
sub-creed of 3 commitments = 24 sub-commitments; Knowledge delegates to the
source truth-seeking creed. `x/work_creed` can store pins supplied in genesis,
and `ceremony-inject creed` can prepare them, but the published `zerone-1` and
testnet genesis states contain `{}` and current `zerone-2` ceremony does not
call that injection. There are no verified live inception pins.

The source hashes are test-bound. Genesis adoption and post-genesis amendment
messages/queries are not operating recursions.

## 7. Participation grows through participation

**Closed by:** `x/claiming_pot.MsgAddBootstrapEntry` (gov-or-capped-registrar
gated, idempotent) + bootstrap pots are non-expiring; binding tests
`TestLateBootstrap_AddThenClaim`, `TestLateBootstrap_AddIsIdempotentAcrossLIPs`,
`TestLateBootstrap_AdmittedAgentCanClaimAfterManyBlocks`, and
`TestScenario13e_BootstrapPotsDoNotExpire`.

The participant set is not closed at genesis. Governance or the configured,
governance-revocable registrar can admit late participants; the registrar is
daily-rate- and lifetime-cap-bounded. The seed itself mints only when the
admitted participant claims through `MintWithCap`, the same cap gate used by
other emission paths. The live registrar is a disclosed custodial power, not a
decentralization claim.

## 8. The economy is hard-capped and self-circulating

**Closed by:** `x/vesting_rewards.MintWithCap` as the wired post-genesis native
mint gate plus the `InitChain` bank-supply cap assertion; binding tests cover
bootstrap, bridge, genesis, and non-minting sponsorship boundaries.

Claiming-pot claims, substrate-bridge rewards, the default-zero knowledge probe
pool, and default-disabled `x/tokens` emission periods all route through
`MintWithCap` when active. Vesting-rewards consensus v2 retires automatic
transaction-bearing block rewards; its named upgrade remains a separate
activation boundary.
Sponsorship circulates existing supply. The cap is live-supply-anchored, so a
burn frees future headroom. Training disbursement and contribution-challenge
bonus minting are release-sealed. This closes the cap loop; it does not prove
that every configured issuance lane is participation-earned.

## 9. The source contains a self-funded audit capability

**Capability tested by:** UW commitment 12 + `ProbeBountyPoolModuleName` +
binding tests `TestTruthSeeking_AuditBudgetIsAutonomous`,
`TestTruthSeeking_ChainPaysForOwnAudit`,
`TestMoat_ProbeBountyPoolAccumulatesAndFundsBonuses`, and
`TestMoat_ProbeBountyPoolRespectsCap`.

With a positive configured rate, the pool can mint capped ZRN and pay
participants who answer stress-test calls. Published `zerone-1` sets that rate
to zero and has no pre-funded pool; `zerone-2` intentionally keeps both mint
and invitation bonus dormant. The self-funded audit loop is tested under an
enabled harness configuration, not active production economics.

## 10. The recursion is observable

**Scaffolded by:** voice-layer event attributes; `docs/EVENTS.md` mirror
invariant; test `TestRecursiveVoiceAudit_StagedRecursionEventsCarryDoctrineAttributes`,
which captures a directly constructed self-sponsorship test loop; and
`TestRecursiveZerone_TestNamesCitedInDoctrineExist`, which verifies cited test
names exist.

For the tested paths, events carry `creed_commitment` and (for UW events)
`mechanism` attributes. Once the public self-attestation bridge is activated,
an indexer could compute:

- the rate of self-attestations (events from `zerone-self-v1` adapter)
- the rate of self-sponsorship fulfillments (sponsorship events with `domain="zerone_self"`)
- the cumulative lineage amount attributed through `zerone_self` attestations
- the audit-pool burn rate vs. probe-bounty pay rate

The chain is legible at the recursion layer, not just the transaction layer.

---

## What this is not

- **Not a paradox.** Every recursion is well-founded: each loop terminates at a verifiable artifact (a fact, a hash, a bond, a fulfillment record). None depend on self-reference in a way that makes them ill-defined.
- **Not a closed system.** External value still enters through sponsorship; external work still enters through substrate-bridge adapters; external reach is what motivates sub-creeds for external work classes. The chain participates in its own systems; it does not isolate itself from external ones.
- **Not a claim that self-verification is wired.** The scaffolds reuse
  `MintWithCap`, escrow, sponsorship, and lineage accounting, but current
  public submission refuses the pending self-claim before verification.

## The discipline

Before merging a change that touches any recursion above:

1. Does the recursion still terminate at a verifiable artifact (no infinite regress)?
2. Does the recursion route through the same machinery external participants use (no special-case self-code)?
3. Does the binding test still fail if the recursion breaks?
4. Does the voice layer still emit the attributes that make the recursion observable?

These four checks are what keep "recursive" from collapsing into "incoherent."

---

— *Inception: 2026-05-11. The fixed-point target is source-bound; the public
self-claim loop remains open until translation and activation land.*
