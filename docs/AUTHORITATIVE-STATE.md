# Authoritative State — Staking, Governance, and Domains

Status: **accepted source design of record (2026-08-08); not implemented,
scheduled, deployed, or network-activated.**

Decision date: 2026-08-08

Reviewed source baseline: canonical `github/main` commit
`c7802db2d79b29f1d21c44a8d8136dfbc3c7b585`, Go 1.25.12, Cosmos SDK
0.53.8, CometBFT 0.38.25, and IBC-Go 10.7.0.

This document defines Zerone's target ownership of consensus stake, policy
decisions, and domain identity. It is normative for new source design and for
the migration that implements it. It does not change a running chain, register
an upgrade handler, schedule an upgrade, select an electorate, move funds, or
claim that present governance is independent.

Current-source and live-network observations remain distinct from this
decision. Until every activation gate below is satisfied, operators and clients
MUST describe the current dual ledgers and concentrated effective control as
they actually exist.

The words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are normative.

## 1. Decision

Zerone adopts three single-writer authority boundaries:

1. Cosmos SDK `x/staking` is the sole long-lived staking and consensus-
   validator authority.
2. Cosmos SDK `x/gov` is the sole ordinary proposal, tally, decision, and
   atomic execution authority.
3. Zerone `x/ontology` is the sole domain-identity, hierarchy, stratum, and
   lifecycle authority.

Everything else is one of:

- evidence owned by the module that observed it;
- a deterministic projection of canonical state;
- an immutable snapshot used by a bounded process; or
- an explicitly bounded, exit-only legacy claim ledger; or
- explicitly labelled, read-only legacy state.

No projection, cache, compatibility query, tier, reputation score, document,
or event becomes a second writer merely because downstream code consumes it.

### 1.1 Authority table

| State | Sole authority | Explicitly not authoritative |
|---|---|---|
| Validator operator, consensus key, bonded status, delegation shares, unbonding, redelegation, consensus power, consensus jail/slash | SDK `x/staking` | Custom `x/zerone_staking`, verifier tiers, knowledge state |
| Account, operational-key, and self-certifying DID binding | Zerone `x/auth` | A DID label as proof of one independent person or controller |
| Independent controller record, authenticated address/profile bindings, rotations, and merges | `x/controller` | Self-asserted DIDs, balances, validators, profiles, or addresses counted separately |
| Verifier participation history | `x/verifier_profile` | Bank balance, SDK voting power, custom tier alone |
| Per-domain competence and its decay | `x/qualification` | Stake, KARMA, global reputation alone |
| Claims, facts, verification rounds, verdict evidence, and domain analytics | `x/knowledge` | Domain identity or lifecycle |
| Policy proposal, ballot, tally, terminal result, and typed execution | SDK `x/gov` | Custom LIPs, treasury committees, Guardian ceremonies |
| Governance policy, versioned live roster, and immutable per-proposal electorate | `x/electorate` | SDK stake, custom stake, balances, KARMA, verifier tier |
| Domain ID, kind, revision, stratum, hierarchy, lifecycle, alias, and tombstone | `x/ontology` | Knowledge-domain records and client-maintained taxonomies |
| Reconciled legacy monetary claims and bounded withdrawals | `x/legacy_claims` | Legacy aggregate fields, unexplained balances, or old modules mutating history |
| Research-fund balance and scoped disbursement | `research_fund` account through an authority-gated `x/vesting_rewards` handler | Custom LIP execution or a committee keeper writing arbitrary state |
| Emergency quarantine and one exact recovery capability | `x/emergency` | Ordinary policy, treasury, ontology, or electorate authority |

## 2. Why the existing split cannot be retained

This decision does not rest on naming preference. The current source contains
independent state machines whose values can disagree while all remain committed
consensus state.

### 2.1 Two staking ledgers

SDK `x/staking` drives CometBFT validator updates, distribution, slashing,
evidence, and SDK governance. Custom `x/zerone_staking` separately stores
ordinary-account validators, delegations, unbondings, tiers, reputation, and
slash fields. Registering in the custom module does not create an SDK validator
or alter consensus power.

The custom ledger also lacks the conservation properties required for custody:

- registration creates an ordinary self-delegation record, while generic
  undelegation reduces delegated stake rather than self-delegation;
- custom slash paths reduce validator aggregates and move coins without
  reducing individual delegation claims; and
- governance reads individual custom delegations while its denominator reads
  active validator aggregates, so numerator and denominator can diverge.

Passing unit tests do not establish solvency because the custom-staking bank
mock accepts transfers without maintaining real balances. Migration MUST treat
all custom staking claims as untrusted accounting input until reconciled
against the module account.

### 2.2 Two governance systems

SDK governance already owns the application authority address and atomically
executes typed message bundles through the application message router. Custom
LIPs use a separate custom-stake electorate and contain additional research,
seat, phase, parameter, and historical upgrade state.

The custom parameter router is constructed empty. A parameter LIP can be
marked passed before dispatch, while handler absence or execution failure is
logged and does not reverse that terminal result. Other LIP categories contain
explicitly incomplete or absent post-pass dispatch. Custom LIP escrow records
aggregate proposal stake but no per-staker claimant ledger.

A result that says `PASSED` while committing no authorized target mutation is
not an authoritative execution record.

### 2.3 Two domain registries

`x/ontology` and `x/knowledge` both store primary domain records with different
schemas and independent proposal and activation paths. Knowledge claim
admission currently checks only that its own domain record exists; it does not
require the ontology record to exist or be active.

The checked-in genesis material contains 18 knowledge domains and seven
ontology domains, with only five shared. Knowledge initialization also creates
four doctrine namespaces. A complete migration census therefore has 24
candidate IDs before any later proposals, not one automatically
reconcilable taxonomy.

Current ontology export also omits stored logic-zone and incompleteness-
acknowledgment state, while initialization recreates default zones. H4 cannot
claim round-trip preservation without migrating and exporting those keyspaces
explicitly.

## 3. Staking authority

### 3.1 SDK staking owns stake

SDK `x/staking` MUST be the only module that owns:

- validator operator addresses and consensus public keys;
- bonded, unbonded, and unbonding validator status;
- delegation shares, redelegations, and unbonding entries;
- CometBFT validator-update power;
- consensus jailing and consensus-fault slashing; and
- bonded principal and delegation-share issuance and redemption.

No Zerone custom module may present its records as consensus validators,
consensus stake, bonded supply, or consensus jail state. Only SDK staking pools
may hold delegated `uzrn` after legacy claims are drained.

SDK `x/distribution` remains the canonical accounting authority for accrued
validator commission, delegator rewards, and their withdrawals, using SDK
staking state as its power input. That dependent reward ledger is not a second
staking or validator authority.

### 3.2 Controller binding is explicit

Zerone `x/auth` remains the address, key, and self-certifying DID registry. The
target `x/controller` module MUST own the distinct claim that several addresses,
profiles, or validators resolve to one effective controller. Its records MUST
be versioned, challengeable under the active controller policy, and preserve
rotation and merge history. A merge may reduce controller count; splitting one
controller into several voting or panel units requires new independent-
controller evidence and MUST NOT be achieved by key rotation.

`x/controller` is not a permission to claim solved personhood. In
`BOOTSTRAP_DISCLOSED` operation it records an enumerated, openly concentrated
roster. `OPEN_CONTROLLER` requires the later uniqueness and challenge process
defined in section 4.3. `VerifierProfile` does not duplicate `controller_id`;
consumers resolve its `profile_id` through the controller registry. None infers
controller independence from an address, DID, profile, or validator count.

### 3.3 Verifier profile is non-custodial

The target `x/verifier_profile` module MUST own only identity and participation
state:

```text
VerifierProfile {
  profile_id
  sdk_valoper                // optional for non-consensus verifiers
  status                     // ACTIVE, SUSPENDED, RETIRED, LEGACY_ONLY
  verification_totals
  contested_totals
  global_reputation
  created_height
  updated_height
}
```

The profile module MUST NOT own coins, third-party delegations, commission,
unbonding state, a self-delegation field, a user-supplied consensus key,
consensus jail state, an independent active-validator bit, DID/controller
bindings, or a persisted tier that can become authority. Tier labels MAY be
query-time projections of explicit profile fields.

A controller-authenticated message may create or retire its own profile and may
link an SDK validator only after SDK operator ownership and the controller
binding both verify. Finalized knowledge-round receipts update participation
history through a replay-protected keeper interface; no caller supplies its own
counters or reputation. Status transitions follow a versioned deterministic
profile policy. SDK governance may update that policy but MUST NOT arbitrarily
activate an individual `LEGACY_ONLY` or suspended profile.

Any profile linked to an SDK validator MUST resolve the validator operator and
consensus identity from SDK staking by canonical address bytes. The migration
MUST NOT trust the legacy custom `ConsensusPubkey` string because it was not
used to create or authenticate the Comet validator.

Existing reputation cached in custom staking or `x/auth` is legacy evidence,
not a second writer. The migration MUST retire those writers and preserve their
last values as explicitly sourced history if `x/verifier_profile` becomes the
global participation-history authority.

### 3.4 Qualification is non-economic

`x/qualification` MUST describe evidenced, time-bounded competence for one
profile and one immutable domain revision. Stake, balance, validator power, and
payment MUST NOT grant, renew, weight, or preserve a qualification. The legacy
`MsgQualifyByStake` pathway is retired.

Only the qualification keeper may create or change qualification state, by
validating replay-protected evidence receipts against a versioned policy.
Permitted evidence kinds are a disclosed, expiring bootstrap manifest; finalized
domain-assessment results from a controller-deduplicated panel; and adjudicated
domain track record. A controller may withdraw its qualification. SDK
governance may change future policy but cannot directly grant, renew, or score
one profile. Endorsements are evidence only and are controller-deduplicated;
self-endorsement has no effect.

Every locked legacy qualification amount MUST be reconciled against the
qualification module balance, then either be refunded before H4 or become a
claimant-bound `LegacyQualificationClaim` as part of H4. A qualification created by
the stake pathway is preserved as legacy history but does not become an active
target qualification without independent competence evidence. Claim withdrawal
and competence assessment are separate decisions.

### 3.5 Verification eligibility and panel power

For the current vote-extension transport, eligibility MUST be the intersection
of:

```text
SDK bonded
AND SDK unjailed
AND authenticated Comet voter ↔ SDK valoper binding
AND active verifier profile
AND authenticated profile ↔ controller binding
AND active qualification for the canonical domain
```

An off-consensus verification transport MAY admit non-validator profiles, but
it MUST authenticate their controller identity independently and MUST NOT
pretend they are Comet validators.

Panel selection MUST NOT read liquid balance, SDK tokens, SDK delegation
shares, legacy custom stake, KARMA, or wealth-derived tiers. The initial target
policy deduplicates candidates by `controller_id`, then selects uniformly among
eligible, domain-qualified controllers. One controller may occupy at most one
seat in a round even if it owns multiple addresses, profiles, or validators.
Each selected controller has one whole verdict. A later bounded qualification
policy requires a new version of this design; qualification may not quietly
become a continuous vote-weight multiplier.

Uniform selection MUST be deterministic and manipulation-bounded, not a caller-
supplied VRF claim. A versioned `PanelSelectionPolicy` MUST pin the canonical
raw-byte candidate order, algorithm/source hash, panel size, minimum size,
seed-beacon protocol, lookback/future offset, and missing-contribution/retry
rules. The candidate root and round ID are committed before a later seed is
revealed. No single proposer, claim submitter, block proposer, or selected
verifier may choose or replace the seed. Beacon failure yields no selection,
not a privileged fallback.

Verdict aggregation MUST be equally explicit. Each round snapshots a
`PanelDecisionPolicy` containing its version/source hash, a reduced rational
reveal-quorum threshold, a reduced rational decision threshold strictly greater
than `1/2` and at most `1`, and the `>` or `>=` comparator for each threshold.
The only ballot values in the first target policy are `ACCEPT`, `REJECT`, and
`MALFORMED`; `INCONCLUSIVE` is an aggregate outcome, not a ballot.

For selected seats `S` and valid, on-time, controller-deduplicated reveals
`A`, `R`, and `M`:

```text
P = A + R + M
reveal ratio    = P / S
accept ratio    = A / P
reject ratio    = R / P
malformed ratio = M / P
```

`S = 0`, `P = 0`, or failed reveal quorum yields `INCONCLUSIVE`. After quorum,
the unique option meeting the decision threshold becomes the verdict; if none
does, the verdict is `INCONCLUSIVE`. Because the decision threshold is greater
than `1/2`, two options cannot both qualify. All comparisons use checked integer
cross-multiplication; floating-point values, stake, reputation, confidence,
qualification score, and unrevealed commitments never change ballot power or
the decision. A reported confidence value is derived analytics only and MUST
NOT be fed back into this aggregation rule.

When too few qualified profiles exist, the round MUST enter an explicit
`INSUFFICIENT_PANEL` or equivalent non-verdict state. It MUST NOT silently pad
the panel with unqualified participants.

Round creation MUST snapshot the selection-policy version/hash, candidate root,
seed commitment/proof, selected profile IDs, controller IDs, domain ID and
revision, qualification revision, decision-policy version/hash and exact
thresholds/comparators, transport identity, and one-seat power. Later staking,
qualification, profile, or domain changes MUST NOT alter that round's
electorate or verdict arithmetic.

The target panel MUST remain disabled until every selected transport identity
has an authenticated controller binding. A concentrated bootstrap verifier
roster MAY be used only when every controller/profile/validator relationship is
enumerated and its concentration is published; a self-certifying DID alone is
not sufficient. Selection MUST also remain disabled until the exact policy and
seed-beacon implementation are hash-pinned, independently reviewed, and covered
by deterministic and adversarial vectors. Aggregation MUST remain disabled
until the exact `PanelDecisionPolicy` implementation is covered by golden
quorum, equality, tie, missing-reveal, malformed, and integer-boundary vectors.

### 3.6 Penalty boundary

Penalties MUST follow the fault domain:

- liveness, double-signing, and consensus evidence use SDK slashing;
- incorrect or missed truth work affects verifier profile and qualification;
- an economic truth penalty may forfeit only an escrow explicitly posted by
  that verifier for that round or task; and
- passive SDK delegators MUST NOT be slashed for application-level epistemic
  disagreement.

An explicit work bond is escrow, not staking. Its owning module MUST identify
the claimant, isolate the bond compartment from unrelated funds, and satisfy,
for each denomination:

```text
bond_escrow_balance = active_bonds + pending_refunds + pending_settlements
```

Every settlement transition MUST update the claimant record and move the
matching coins atomically. A completed refund or forfeiture is no longer a
liability. Every pending settlement remains named, claimant-bound, and fully
backed; implementations MUST NOT create an unowned balance bucket.

### 3.7 Staking invariants

1. Only SDK staking emits Comet validator updates.
2. Only SDK staking pools contain delegated `uzrn` after legacy retirement.
3. Verifier-profile and controller state own no coin liability; any associated
   module account is absent or has zero balance.
4. Every validator-backed profile maps one-to-one to a real SDK validator.
5. Bonded and jail status are read from SDK staking, never copied as authority.
6. Profile, tier, qualification, and truth penalties never change Comet power.
7. Token balance and stake never change panel selection or verdict weight.
8. Stake or payment never grants or renews domain qualification.
9. One controller occupies at most one seat in one verification round.
10. Every explicit bond module has an exact balance-to-liability invariant.
11. Every selected controller has one whole panel ballot, and only the
    snapshotted decision policy determines the aggregate verdict.

## 4. Governance authority

### 4.1 SDK governance is the sole ordinary executor

SDK `x/gov` MUST own the ordinary proposal lifecycle, ballots, tally, terminal
decision, and execution. Every governed mutation MUST enter its target module
as a typed message whose authority is the SDK gov module address.

The complete message bundle MUST be classified before voting. Mixed action
classes MUST be rejected rather than using ambiguity to obtain a weaker rule.
Initial action classes are:

- `ORDINARY_PARAMETER`;
- `DOMAIN`;
- `TREASURY`;
- `SOFTWARE_UPGRADE`;
- `ELECTORATE_ACTIVATION`; and
- `CONSTITUTION`.

Each policy version MUST contain an exact allowlist of type URLs for each
class. Generic or nested execution containers, including authz execution,
legacy-content wrappers, and ICA-style arbitrary message containers, MUST be
rejected from governance proposals; they cannot be used to hide an action from
classification. Unknown `Any` values fail closed.

The first authoritative integration MUST reject every proposal with
`Expedited = true`. In SDK 0.53.8, a failed expedited tally is converted to a
regular proposal and requeued after the first tally has deleted its active
votes, without a new voting-start/deposit hook. That fallback is incompatible
with one immutable snapshot and ballot archive. A future upgrade MAY admit
expedited proposals only after separately specifying and reviewing two-stage
snapshot revision, deadlines, first-stage receipts, vote retention, and replay
semantics; it MUST NOT inherit the stock fallback implicitly.

An `AfterProposalSubmission` hook MUST classify the immutable bundle and cause
the extended SDK gov store to persist its bundle hash, class, classifier
version, and governance mode in the submitting transaction; hook failure rolls
back submission. Voting-start integration MUST verify that the bundle remains
allowed in the same class under the recorded mode. In controller mode it also
snapshots the then-active electorate policy. It does not invent an uncommitted
interpretation for the first time, and a policy change does not alter a
proposal already in voting.

H4 installs a deliberately narrow transition mode named
`PRE_H5_STAKE_WEIGHTED`. H4 may enter that mode only with zero SDK proposals in
deposit, voting, or expedited-fallback state. While it is active, submission
accepts only a non-expedited, sole-message `ELECTORATE_ACTIVATION` proposal, one
active proposal at a time. That proposal uses the disclosed stock SDK staking
electorate and tally semantics; it does not receive a fictional controller
snapshot. The extended SDK gov store instead records a canonical
`GovernanceModeReceipt` containing proposal and bundle hashes, classifier
version, action class, `PRE_H5_STAKE_WEIGHTED`, SDK gov parameter hash, voting
window, and final SDK tally inputs/result. H5 atomically changes the mode to
`CONTROLLER_SNAPSHOT`; only proposals entering voting after that change receive
an `ElectorateSnapshot`.

String module/key/value parameter routing is retired. Parameter proposals MUST
carry complete target `MsgUpdateParams` payloads. Domain proposals MUST carry
typed ontology messages. Treasury proposals MUST carry a scoped disbursement
message and cannot mutate any other module.

`ELECTORATE_ACTIVATION` accepts exactly one
`x/electorate.MsgScheduleElectorateActivation`, never a mixed bundle. Its
SDK-gov-authorized handler validates the exact H5 name, height, source/binary,
manifest and policy digests, proves no other active SDK proposal exists,
records `H5_PENDING`, and schedules the matching singleton upgrade plan in the
same cached execution. It has no generic upgrade or message-routing capability.

SDK governance's cached execution semantics are mandatory: a proposal is
`PASSED` only after its entire immediate message bundle commits. If any handler
returns an error or panics, no target write commits and the proposal is
`FAILED` with a persisted reason.

For `CONTROLLER_SNAPSHOT`, the pinned SDK gov extension MUST inject a non-
spoofable execution context that binds proposal ID, bundle hash, action class,
and electorate snapshot to every target handler call. The sole H5 scheduling
handler instead requires a `PRE_H5_STAKE_WEIGHTED` context bound to the final SDK
tally receipt and rejects any controller-snapshot or unlabelled context. No
other target handler accepts the transition mode. A target derives
`authority_ref = SDK_PROPOSAL` from the verified context and gov authority
address; it MUST reject caller-supplied provenance. Genesis initialization and
H4 migration use distinct keeper-internal `GENESIS` and `MIGRATION` authority
references.

A delayed target action MUST separately record `APPROVED`, `PENDING`,
`EXECUTED`, `CANCELLED`, `EXPIRED`, or `FAILED`. Governance approval MUST NOT
be reported as completed delayed execution.

Stock SDK 0.53.8 proposal cancellation deletes the proposal and active votes,
so it is not the accepted cancellation contract. A proposer MAY cancel only
during deposit period. The integration MUST replace deletion with an immutable
`CANCELLED` receipt in the extended SDK gov store, recording the proposal hash,
actor, reason, height, and deposit disposition. Once voting starts, proposer
cancellation is forbidden; only the exact emergency capability in section 4.6
may cancel, and it MUST persist the same receipt plus its ceremony reference.

Stock failed-minimum-deposit processing also deletes proposal content before
calling a hook whose error is ignored. The pinned SDK gov extension MUST first
persist an immutable `DROPPED` receipt with canonical proposal bytes, bundle
hash, proposer, timestamps, reason, and deposit disposition, then remove only
the active queue/state. It MUST NOT rely on `AfterProposalFailedMinDeposit` for
that authority-critical archive.

### 4.2 Electorate state is non-economic

The new `x/electorate` module MUST own versioned governance policy and one
immutable snapshot for every proposal that enters voting in
`CONTROLLER_SNAPSHOT` mode:

```text
ElectoratePolicy {
  policy_version
  action_class
  quorum
  threshold
  veto_threshold
  quorum_comparison
  threshold_comparison
  veto_comparison
  activation_epoch
}

ElectorateRoster {
  roster_version
  mode
  members_root
  activation_epoch
  sunset_epoch
}

ElectorateMember {
  roster_version
  controller_id
  voting_address
  fixed_power = 1
}

ElectorateSnapshot {
  proposal_id
  bundle_hash
  classifier_version
  policy_version
  roster_version
  action_class
  members_root
  total_power
  quorum
  threshold
  veto_threshold
  quorum_comparison
  threshold_comparison
  veto_comparison
  voting_start
  voting_end
}

MemberPower {
  proposal_id
  controller_id
  voting_address
  fixed_power = 1
}
```

In `CONTROLLER_SNAPSHOT` mode, the snapshot MUST be created atomically when
voting starts. On SDK 0.53.8 an idempotent `AfterProposalDeposit` hook MUST
detect the first transition into `VOTING_PERIOD`, verify the bundle
classification and recorded mode, and create the snapshot in the same
transaction. Hook failure MUST roll back the activating deposit. Authority-
critical writes MUST NOT depend on
`AfterProposalVotingPeriodEnded`, whose error is deliberately ignored by the
SDK.

Snapshot membership MUST be derived only from the `ElectorateRoster` active at
voting start. Every roster entry must resolve to the same controller/address
binding in `x/controller`; duplicate controllers, duplicate voting addresses,
missing bindings, and root mismatches fail snapshot creation.

All thresholds are reduced non-negative rational pairs, not rounded decimal
strings. For fixed-power ballot counts `Y`, `N`, `A`, and `V`, the tally is:

```text
T = total eligible snapshot power
P = Y + N + A + V                 // participation
C = Y + N + V                     // decision power; abstain excluded
quorum ratio   = P / T
approval ratio = Y / C
veto ratio     = V / C
```

`T = 0`, failed quorum, or `C = 0` rejects the proposal. After quorum, a veto
comparison that meets its configured rejection boundary rejects; otherwise the
approval comparison decides passage. Ratios MUST be compared by checked integer
cross-multiplication with no floating-point conversion or implicit rounding.
The snapshot pins each comparison as `>` or `>=`; every equality, all-abstain,
zero, and one-unit boundary requires a golden vector.

The numerator and denominator MUST use the same frozen member set and policy.
In `CONTROLLER_SNAPSHOT` mode, an `AfterProposalVote` admission hook MUST reject
a voter absent from the snapshot, a duplicate controller address, or anything
other than one whole `YES`, `NO`, `ABSTAIN`, or `NO_WITH_VETO` option; hook
failure rolls back the SDK vote write. Weighted-split ballots are rejected in
that mode. The explicitly labelled stake-weighted transition modes retain stock
SDK vote admission, including its weighted ballots, but MUST archive every
accepted vote and replacement as a `BallotRevision` under the mode receipt.

Revoting before the deadline is permitted as an explicit replacement. The
extended SDK gov store, not `x/electorate`, MUST append an immutable
`BallotRevision` after the SDK writes the new active vote but before the cached
transaction commits; the prior accepted vote was already archived as its own
revision. Rejected hook execution rolls both writes back. Only the latest
accepted SDK gov revision counts. `x/electorate` owns policy, roster, and
snapshot state, not a second ballot status.

Governance power is non-transferable, non-delegable, and not a bank or staking
balance. The eligibility and tally implementation MUST NOT read:

- liquid or vesting balance;
- SDK stake or delegation shares;
- legacy custom stake;
- KARMA edges or counts;
- verifier rewards; or
- verifier tier or reputation as continuous voting weight.

Proposal deposits are bounded spam bonds only. Paying or augmenting a deposit
MUST NOT create membership or vote power.

Cosmos SDK 0.53.8 exposes a custom vote-calculation callback, but its enclosing
tally still enumerates SDK validators, reads total bonded tokens, and applies
global SDK quorum, threshold, veto, and strict-comparison rules. The callback
alone cannot implement this design.

The implementation MUST use a narrow, hash-pinned and independently reviewed
SDK/app extension that replaces the complete proposal-specific tally policy.
It supplies final ballot counts, total eligible snapshot power, quorum,
threshold, veto threshold, and explicit inclusive/exclusive comparison
semantics from the same `ElectorateSnapshot`. That path MUST NOT enumerate SDK
validators or read bonded tokens. It MUST NOT emulate non-economic governance
with a fake staking keeper or an opaque stake-scaling adapter.

The same pinned extension MUST augment the cached SDK terminalization path. It
atomically persists the `ProposalTerminalReceipt`, final ballot root, complete
ballot-revision archive, and final tally inputs/results before active votes are
cleared. None of those writes may rely on the ignored voting-ended hook.
Proposal, cancellation, ballot, and tally history remains queryable after
terminalization.

### 4.3 Electorate modes and control honesty

The current self-certifying DID mapping proves a key binding, not one
independent controller. Open one-DID-one-vote is therefore forbidden.

The first permitted mode is `BOOTSTRAP_DISCLOSED`:

- every effective controller and voting address is enumerated;
- each controller has exactly one unit;
- controller overlap and concentration are queryable;
- a sunset height or epoch is committed before activation; and
- the chain MUST NOT describe this mode as decentralized or independent when
  the effective controllers are correlated.

The later `OPEN_CONTROLLER` mode may activate only after the release packet
demonstrates a versioned controller-uniqueness, challenge, merge, rotation, and
exit process. Controller merges may only reduce voting units. If bootstrap
sunset arrives before the open-mode predicates are met, ordinary policy
governance freezes; authority does not silently revert to a founder,
household, operator, validator, or wealth exception.

Changing electorate roster, electorate policy, or classifier policy is a typed
`CONSTITUTION` action authorized only by SDK gov. A change MUST activate at a
future epoch and MUST NOT alter the snapshot of the proposal that approved it.
The initial disabled roster is installed by the H4 manifest; all later roster
changes append a new version and preserve prior versions for proposal replay.

### 4.4 Custom LIPs become historical only

At authority unification:

- every custom LIP, vote, seat, phase, attachment, and research record remains
  queryable as explicitly labelled legacy state;
- all custom LIP submission, staking, voting, stage advancement, seat,
  parameter, phase, research-spend, and execution messages return stable typed
  retirement errors;
- the custom BeginBlock execution paths and parameter router are removed; and
- a passed-but-inert legacy action is recorded as `legacy_no_effect` and MUST
  NOT be replayed automatically.

Future deliberation MAY occur off-chain or in a separately reviewed advisory
module. It MUST link by content hash to the SDK proposal and MUST NOT maintain a
second decision status.

The raw research proposal/vote/execute surface in `x/knowledge` is retired at
the same boundary. Its proposal and vote keys remain explicitly labelled legacy
evidence; its event-only execution message gains no target authority. Likewise,
legacy `MsgAddFact` and its guardian-delayed `PendingFactInjection` queue are
retired because they bypass verification. Every queued injection is cancelled
with an immutable receipt and the legacy BeginBlock materializer is disabled.
A future direct fact-adoption path requires a separately accepted classified
policy; it is not inferred from gov authority or old guardian parameters.

### 4.5 Treasury scope

The `research_fund` module account remains the canonical balance. A new typed,
SDK-gov-authority message in `x/vesting_rewards` MUST be the only ordinary
disbursement path after migration. It may validate recipient, exact coins,
purpose metadata, remaining balance, and any delay policy, then move only
research-fund coins.

It MUST NOT change params, staking, domains, electorate, software upgrades, or
other treasury accounts. Legacy designated voters, community seats, and phase
records remain historical and confer no post-migration authority.

The custom research-fund restriction exists in source and tests but is not
installed in production app wiring. The generic module-account denylist blocks
ordinary bank `MsgSend` into `research_fund`, but keeper/module inbound transfers
and all egress lack the intended scoped guard. H4 MUST install a canonical
inbound router and an outbound bank-layer guard, or equivalent unforgeable
scoped capabilities. Only approved deposit sources may send into
`research_fund`, and egress is permitted only while the typed
`x/vesting_rewards` handler executes. Every legacy
`DisburseFromResearchFund` caller and direct module-account send path is retired
or rejected.

### 4.6 Emergency authority is a circuit breaker

`x/emergency` remains separate from ordinary governance because containment
must be available while ordinary messages are quarantined. Its authority is
strictly bounded.

The current `x/knowledge` incident, `ModulePause`, and surgical-correction
stores are not a second circuit breaker. Their pause helper has no production
consumer outside that package, so its current records do not prove a working
quarantine. H4 preserves them as legacy audit evidence and retires their
open/remediate/resolve/close, pause/unpause, and correction mutations. Target
handlers consume canonical `x/emergency` incident evidence. A future surgical
repair message requires its own classified, deterministic policy; gov authority
alone does not revive the legacy escape hatch.

An emergency electorate MUST snapshot an explicit responder roster and its
controller bindings, with at most one responder per controller and one whole
vote. Operational eligibility MAY require a responder to be an SDK bonded and
unjailed validator, but stake MUST NOT weight the ceremony. Custom-stake
Guardian tier MUST NOT confer emergency authority.

Every ceremony MUST snapshot an `EmergencyPolicy` version, action type,
responder/controller root, total eligible units, rational quorum and approval
thresholds with explicit comparators, action digest, start/expiry, and maximum
quarantine or capability lifetime. Responders cast one whole `YES` or `NO`.
Quorum is participation divided by snapshotted eligible units; approval is
`YES / (YES + NO)`, compared by checked integer cross-multiplication. Zero
eligible units, zero decision ballots, expired snapshots, and root mismatch
fail closed.

Every action is bound to one active incident ID, evidence digest, quarantined
message/type scope, and cumulative incident counters. Policy caps total
quarantine duration, renewals, and cancellation set size. A renewal requires
new evidence and cannot exceed the cumulative cap; reopening the same evidence
under another incident ID cannot reset it. Proposal cancellation may name only
a finite precommitted set of active proposals whose classified messages match
that incident scope. Unrelated policy proposals cannot be cancelled.

Changing the live responder roster or policy is a future-epoch `CONSTITUTION`
action and MUST NOT alter an active ceremony's snapshot. H4 MUST install an
explicit disabled state when no reviewed roster and policy exists; it must not
derive one from custom Guardian tier. Because `PRE_H5_STAKE_WEIGHTED` admits
only the H5 scheduler, a network intending to proceed directly to H5 MUST
install a reviewed active responder roster/policy at H4. A network that chooses
disabled emergency state cannot schedule H5 until another separately accepted
upgrade supplies that state; the transition proposal cannot bootstrap its own
cancellation authority.

Emergency authority may:

- place or renew a scoped transaction quarantine;
- resume after required evidence and reconciliation;
- cancel the exact bounded active-proposal set committed by the incident
  ceremony while persisting each cancellation receipt defined in section 4.1;
- revoke an unsafe pending recovery capability; or
- issue one single-use capability consumed only by the singleton upgrade gate
  for an exact digest-bound software-upgrade plan or plan cancellation.

Before the H5 upgrade height, consuming an authorized cancellation capability
for the exact scheduled H5 plan MUST atomically cancel that plan, persist an
immutable plan-cancellation receipt with its incident/ceremony evidence, clear
the matching `H5_PENDING` latch, and return governance to
`PRE_H5_STAKE_WEIGHTED`. A digest mismatch or partial transition fails closed.
Once the H5 handler begins or commits, cancellation cannot rewind it; only a
new forward recovery upgrade under the bounded emergency rules is admissible.

It MUST NOT update ordinary params, spend treasury funds, amend creed or
electorate policy, create or mutate domains, register adapters, or execute an
arbitrary message bundle.

A resume MUST either clear a verified-reconciled governance hold atomically or
prove that the exact reconciliation upgrade is already scheduled. It MUST NOT
leave ordinary governance indefinitely dependent on another responder quorum.

### 4.7 Governance invariants

1. The SDK gov module address is the sole ordinary authority address.
2. SDK gov is the sole ordinary tally-to-execution transition.
3. Every governed target write is a typed message with full target validation.
4. In controller mode, proposal power and quorum use one immutable electorate
   snapshot.
5. In controller mode, money, stake, reward, KARMA, tier, and reputation do not
   scale policy power; the two stake-weighted schedulers are labelled transition
   exceptions only.
6. Every controller has at most one voting unit in one snapshot.
7. A proposal cannot amend its own electorate or action-class policy.
8. Any immediate execution failure commits zero target writes.
9. Custom LIPs and legacy treasury committees confer no post-migration power.
10. Emergency capability is action-bound, time-bounded, and non-general.
11. Generic execution wrappers cannot bypass action classification.
12. Cancellation and every accepted ballot replacement remain durably
    queryable.

## 5. Domain authority

### 5.1 Ontology owns the canonical registry

`x/ontology` MUST be the only module that creates, revises, freezes,
deprecates, archives, merges, reparents, aliases, or resolves a canonical
domain.

Canonical v2 state uses new versioned types rather than changing the existing
v1 protobuf meaning in place:

```text
DomainRecord {
  domain_id                 // immutable canonical identifier
  current_revision
  lifecycle                 // ACTIVE, FROZEN, DEPRECATED, ARCHIVED, MERGED
  merged_into               // set only when lifecycle = MERGED
  created_height
  updated_height
  latest_transition_sequence
}

DomainRevision {
  domain_id
  revision
  display_name
  description
  kind                      // EPISTEMIC or SYSTEM_DOCTRINE initially
  stratum_ref { id, revision } // required only for EPISTEMIC
  parent_ref { id, revision }  // optional
  computed_depth            // derived and invariant-checked
  effective_height
  authority_ref             // GENESIS, MIGRATION, or SDK_PROPOSAL + id/hash
  content_hash
}

DomainTransition {
  domain_id
  sequence
  from_lifecycle
  to_lifecycle
  height
  authority_ref             // GENESIS, MIGRATION, or SDK_PROPOSAL + id/hash
  reason_hash
}
```

Kind, stratum, parent, and depth live authoritatively in immutable revisions,
not as independently mutable `DomainRecord` fields. A query MAY expose a
current projection, but the projection MUST name its source revision and be
recomputed by an invariant.

Stratum definitions MUST also be versioned. Every epistemic `DomainRevision`
pins a `stratum_id` and `stratum_revision`, including valid stratum ID zero, so
a later confidence-ceiling or decay change cannot alter old facts
retroactively.

Ontology v2 MUST also preserve and version the adjacent ontology state that
affects knowledge semantics:

- every `CrossStratumLinkRevision` pins canonical source and target
  `DomainRef`s plus its relation, effective height, and authority reference;
- every `LogicZoneRevision` pins completeness, decidability, Gödel applicability,
  confidence ceiling, and other admission semantics; and
- every fact or acknowledgment that uses a logic zone pins
  `{zone_id, zone_revision}`.

`IncompletenessAcknowledgment` MUST become append-only evidence in ontology,
with an immutable acknowledgment ID/sequence, exact fact ID, and logic-zone
revision; it is not a mutable fact verdict. The current fact-ID-only overwrite
key is retired after its last value is preserved as legacy evidence. Logic-zone
creation or policy revision is a typed SDK-gov ontology action.

The exact v2 `domain_id` grammar is ASCII lower snake case:

```text
^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$
maximum 64 bytes
```

This admits every checked-in canonical ID. Transaction messages MUST reject,
not trim, lowercase, Unicode-normalize, or otherwise rewrite, a noncanonical
ID. Domain strings participate in hashes and store keys. Display names may be
Unicode and may change by revision.

Domain IDs are immutable. A display-name change does not rename the ID. A
merge leaves the source as a permanent tombstone pointing to the target; it
does not rewrite historical facts. Aliases are query and discovery aids, not
alternate mutation addresses.

Doctrine namespaces are `SYSTEM_DOCTRINE` domains with an explicit system
write policy. They MUST NOT be forced into an empirical stratum merely to fit
the old knowledge-domain schema. Doctrine entries are governance-adopted
system assertions, not epistemic evidence: they MUST NOT earn epistemic
confidence, truth-work rewards, or training-value weight, and MUST NOT provide
factual citation inheritance. Default factual and training-corpus exports MUST
exclude them. A separately labelled doctrine export MAY include them without
representing them as empirically verified facts.

### 5.2 Proposal and lifecycle state

A candidate domain belongs to an SDK governance proposal or an ontology-owned,
bounded slug reservation. A reservation confers no admission authority and
expires deterministically. The candidate is not inserted into the canonical
domain registry until the typed ontology creation message executes
successfully.

The initial lifecycle transitions are exactly:

```text
create -> ACTIVE
ACTIVE <-> FROZEN
FROZEN -> DEPRECATED
DEPRECATED -> ARCHIVED | MERGED
ARCHIVED and MERGED are terminal
```

- `ACTIVE` admits new domain-scoped work.
- `FROZEN` admits no new work; already snapshotted settlement and narrowly
  scoped integrity challenges continue under their frozen domain revision.
- `DEPRECATED` admits no new work but permits settlement and challenges against
  existing records.
- `ARCHIVED` remains queryable but starts no new process.
- `MERGED` is a terminal source tombstone. A mutation against it fails with a
  typed error naming the target; the signed message is never silently
  rewritten.

Archive or merge MUST require zero in-flight processes and zero unresolved
escrow, or close/refund them atomically under a proposal-specific disposition.

Every structural mutation appends a `DomainRevision`. Every lifecycle mutation
appends a `DomainTransition`. The current pointers, immutable history, graph
indexes, and one canonical event update atomically. No production caller may
directly overwrite them outside the ontology lifecycle engine.

A reparent operation MUST atomically append revisions for the moved domain and
every descendant whose computed depth changes. It MUST fail before writing if
the affected set exceeds an explicit bound; descendant depths MUST NOT be
silently rewritten outside the revision history.

### 5.3 Read contract and fail-closed admission

All domain-bearing modules MUST consume one read-only contract:

```text
Resolve(domain_id)
RequireKnown(domain_id, use)
RequirePrimaryActive(domain_id, use)
GetCurrentRevision(domain_id)
GetRevision(domain_id, revision)
GetStratum(domain_id, revision)
GetDepth(domain_id, revision)
Iterate(filter)
```

`Resolve` may follow aliases and merged tombstones for query/display only. New
claims, facts, qualifications, sponsorships, bounties, capture-defense records,
and capture challenges MUST name a primary canonical ID directly and pass the
use-specific lifecycle policy before any coins move or state is written.

Unknown or failed ontology lookup MUST fail closed. Consensus logic MUST NOT
substitute depth one, an empty stratum, no confidence ceiling, or an omitted
capture adjustment.

Every persistent domain-bearing record MUST store:

```text
DomainRef {
  primary_domain_id
  primary_domain_revision
  original_domain_id          // optional legacy/alias ID for audit
}

StratumRef {
  stratum_id
  stratum_revision
}
```

This includes qualifications and endorsements, knowledge claims/facts/rounds,
capture histories and challenges, sponsorship orders and bounty pools,
substrate-adapter requirements, and pending external claims. Existing processes
continue against the frozen revision. Historical reads MAY resolve aliases for
display, but MUST preserve both the original ID and the canonical primary
revision used by migration or admission. New writes require
`original_domain_id` to be empty or equal to the primary ID.

Any record independently scoped by stratum MUST store `StratumRef` rather than
an unversioned scalar or string. This includes cached fact stratum, capture-
defense stratum reputation, and cross-stratum requirements. A display cache MAY
retain a label only as an explicitly derived projection whose source revision
is named and invariant-checked.

### 5.4 Derived domain views

Fact count, claim count, diversity, temperature, carrying capacity,
qualification activity, and verification activity belong to their producing
modules. They are derived views keyed by canonical domain ID and MUST NOT be
independently writable registry metadata.

The legacy knowledge-domain query surface MAY remain for one compatibility
release, but it MUST project ontology metadata plus knowledge-owned metrics.
Its response MUST identify the canonical source and projection height. The old
knowledge domain KV records remain read-only evidence and do not authorize
admission.

### 5.5 Domain invariants

1. One canonical ID resolves to one primary record.
2. Noncanonical transaction IDs fail; legacy collisions fail migration.
3. Every epistemic domain revision references an existing stratum revision.
4. Every parent exists, is not self, and produces an acyclic bounded graph.
5. Stored depth equals depth computed from the canonical parent graph.
6. Every lifecycle transition follows the explicit transition matrix.
7. No referenced domain is deleted.
8. Every new domain-bearing record passes a use-specific registry gate.
9. Historical IDs resolve to a canonical record, alias, or immutable
   tombstone.
10. Every secondary index exactly matches canonical primary state.
11. Derived counts reconcile with their producing module's primary records.
12. Genesis and export/import preserve domain/stratum revisions, transitions,
    tombstones, cross-stratum links, logic zones, acknowledgments, references,
    and indexes.
13. Doctrine cannot enter epistemic confidence, reward, factual-support, or
    default training paths.
14. Every merge target is a primary active domain, and alias/tombstone
    resolution is acyclic and bounded.
15. Every cross-stratum endpoint and logic-zone-bearing record pins an exact,
    existing revision.

## 6. Snapshot discipline

The authority model distinguishes live canonical state from process authority.
A target process that spans blocks MUST snapshot every mutable input that can
change its outcome:

| Process | Required immutable snapshot |
|---|---|
| Knowledge verification round | verifier/controller IDs, transport identity, qualification revision, domain revision, one-seat power, deadlines |
| Controller-mode governance proposal | bundle hash, classifier/policy versions, action class, member root, member power, denominator, thresholds/comparators, voting window |
| Emergency ceremony | responder roster, controller IDs, action type, action bytes/hash, target proposal/plan, expiry |
| Domain-scoped claim or fact | canonical domain ID and revision |
| Work bond | claimant, purpose, amount, denom, release/slash conditions, deadline |

Changing live stake, profile, qualification, domain, or roster state after
snapshot creation MUST NOT retroactively change the process outcome.

The sole `PRE_H4_STAKE_WEIGHTED` unification scheduler and sole
`PRE_H5_STAKE_WEIGHTED` electorate scheduler are explicitly disclosed
transition exceptions: they retain SDK stake-at-tally semantics and receive
mode/tally receipts rather than controller snapshots. No ordinary policy or
target-module proposal is admitted in either mode.

## 7. Migration and activation

### 7.1 Precedence and provisional names

Accepted H1 `consolidation-safety-v1`, H2 `founder-renunciation-v1`, and H3
`sdk-0.53-ibc-10` source identities remain unchanged. This document does not
amend, combine, register, schedule, or authorize them.

For an existing network, the provisional later boundaries are:

1. H4-F `authority-freeze-v1`: an earlier on-chain drain/freeze latch for
   conflict-bearing legacy state;
2. H4 `authority-unification-v1`: single-writer state migration and retirement
   of alternate execution paths; and
3. H5 `non-economic-governance-v1`: activation of the published electorate
   policy after H4 is independently verified.

These names are design vocabulary only. None of these handlers exists merely
because this document names it. They MUST NOT be proposed or scheduled until exact
source, binary, migration, rollback, rehearsal, and authority evidence exists.

A fresh successor genesis MAY start directly in the target model, but its
genesis MUST contain the same canonical domain manifest, explicit electorate,
controller-concentration disclosure, and zero-legacy-liability proofs. Calling
the network new does not relax any authority invariant.

### 7.2 Freeze, F census, and commitment

An operator promise to stop submitting transactions is not a state freeze.
H4-F MUST first enter a bounded `DRAINING` mode that rejects new custom stake,
qualification stake, LIP, domain-proposal, research-spend, cross-boundary round,
or vindication-escrow obligations while allowing only enumerated withdrawals,
refunds, and settlements. It also retires the knowledge research-proposal/vote/
execute messages and rejects `MsgAddFact`. Legacy pending fact injections are
cancelled with immutable receipts, and their BeginBlock materializer is disabled;
they are never silently admitted as facts. Knowledge incident, pause/unpause,
and surgical-correction mutations are frozen as legacy audit evidence, and no
pause helper remains in an authoritative call path.

At the same boundary an SDK-governance drain gate rejects new proposal
submissions and records the complete already-active proposal set. Only deposits,
stock SDK ballots, cancellation where still permitted, and terminal processing
for those recorded proposal IDs may continue. Archive replay preserves any
pre-H4-F vote history that the old SDK store cannot provide. Once that set
reaches zero, `H4_GOV_FROZEN` rejects every SDK proposal mutation until the
singleton scheduler below is opened.

At a published cutoff the legacy latch enters `FROZEN`, with every conflict-
bearing and domain-bearing message, BeginBlock, and EndBlock writer disabled.
Bank-layer guards reject unenumerated inbound or outbound transfers for each
frozen legacy account. The cutoff requires every process that must be quiescent
to be terminal or have a deterministic published disposition. The census begins
only after both legacy state and SDK governance have reached and held their
frozen states.

The first fully committed block in that condition is the fixed census point
`F`. Its height and app hash are then knowable. The release process MUST census
and commitment-bind:

- the `F` height/app hash, chain ID, source version map, and relevant store and
  account roots;
- SDK validators, consensus keys, delegations, unbondings, and bonded supply;
- every custom validator aggregate, delegation, unbonding, qualification stake,
  claimant, and relevant module coin;
- all knowledge rounds and selected participants; raw knowledge research-
  proposal/vote keys; every pending or cancelled fact injection; and the
  disabled materialization queue; all knowledge incident, module-pause,
  remediation, privileged-action, and surgical-correction records;
- all custom LIPs, votes, proposal stake, research proposals, seats, phase
  transitions, attachments, and module balance;
- both domain registries, every ontology and knowledge domain proposal, their
  coin-transfer evidence and module-account balances, strata, indexes, and
  every persistent domain label in downstream modules;
- every knowledge and training-fund challenge, contradiction, patronage,
  `KnowledgeBounty`, augmentation, contribution, live
  `TrainingFundEscrowLocked`, vesting liability, and pending
  `SurvivalPendingReward`; every capture-challenge stake and domain-bounty
  entry; every sponsorship escrow; and every nonterminal substrate-bridge bond,
  plus every substrate-bridge `WitnessPendingReward`, together with their
  account balances or mint entitlements as applicable;
- the `vindication_escrow` account, every surviving `VindicationPending`, every
  final `VindicationRecord`, and archive evidence for deleted pending entries
  and attempted refunds or sweeps;
- cross-stratum links, logic zones, incompleteness acknowledgments, and every
  persistent stratum and logic-zone reference;
- emergency incidents, holds, ceremonies, snapshots, and scheduled plan; and
- the canonical resolution/migration manifest and all archive-evidence roots.

After `F`, the governance gate may open only `PRE_H4_STAKE_WEIGHTED`: one non-
expedited, sole-message standard SDK software-upgrade proposal that schedules
the exact H4 plan. Its `Plan.Info` MUST bind `F`, the manifest and evidence
roots, H4 source/binary digests, height, a cancellation cutoff, and the
rollback/rehearsal packet. The cutoff MUST leave the full voting and published
operational safety window before H4. The narrow H4-F integration archives its
stock SDK ballots and replacements and
persists drop, cancellation, tally, and terminal mode receipts. Successful
cached execution MUST reverify all frozen commitments, schedule the exact H4
plan, set `H4_PENDING` to the same digest, and close every other route
atomically. A failed or dropped scheduler leaves an immutable receipt and may
be retried; it cannot authorize any other message or target mutation. The H4
height MUST leave at least the published maximum deposit/voting plus operational
safety window after scheduling.

While `H4_PENDING` and before the committed cutoff, the gate may admit only one
non-expedited, sole-message SDK upgrade-cancellation proposal naming that exact
plan/digest. It MUST include the full minimum deposit at submission, enter
voting atomically, and have a voting end no later than the cutoff; otherwise
submission is rejected. If it passes, cached execution MUST atomically cancel
the plan, persist the canonical proposal/plan cancellation receipt, clear only
the matching `H4_PENDING`, return to `H4_GOV_FROZEN`, and reopen the singleton
H4 scheduler. At the cutoff the lane MUST be terminal and return to zero; no
later cancellation submission is accepted. A mismatch or partial transition
fails closed. Once the H4 handler begins or commits, cancellation cannot rewind
it; recovery is a new forward upgrade.

No frozen store or account commitment may change between `F` and H4; unrelated
canonical state may continue normally. H4 MUST verify the historical `F`
height/app hash, source version map, current equality of every frozen
commitment, exact `H4_PENDING`/plan digest, and zero active SDK proposals before
writing target state. The runtime H4-1 state artifact MUST be captured for
rehearsal and forensics, but its future full app hash is not falsely claimed to
have been known when the scheduling proposal was created. The H4-F/F/scheduler
sequence is mandatory; H4 MUST refuse to commit if any step or commitment is
absent.

### 7.3 Custom staking reconciliation

Let:

```text
B = bank balance(zerone_staking, uzrn)
D = sum(every active delegation KV claim, including operator self-delegation)
U = sum(pending unbonding claim amounts)
B = D + U
```

The census MUST also reconcile every validator's self, delegated, and total
aggregates against individual claims. Because known execution paths can make
these values disagree, state alone may not identify a unique fair claimant
ledger. Automatic migration requires the displayed equality; aggregate
validator fields do not add liabilities or substitute for individual records.

H4 MUST refuse automatic migration when balances, claimant records, or
aggregates do not reconcile. It MUST NOT mint a deficit, silently haircut,
assign surplus, or infer claimants from aggregate fields. Resolution requires
a separately approved and published manifest choosing an explicit
recapitalization, deterministic claimant adjustment, or successor-genesis
path.

All reconciled liabilities belonging to retired legacy pathways MUST move
atomically at H4 from their source module accounts into source-labelled
compartments in the new `x/legacy_claims` module:

```text
LegacyClaim {
  source_kind
  source_claim_id
  claimant
  coins
  status                 // OUTSTANDING, PENDING_WITHDRAWAL, PAID
  manifest_hash
}
```

For each source and denomination, the compartment balance MUST equal
outstanding claims plus pending withdrawals plus any manifest-labelled
`UnresolvedLegacyBalance`. The latter has no withdrawal or spending route and
requires a later explicit network policy; unexplained residual is still a
hard NO-GO.

The only ordinary value-changing message is a claimant-signed, replay-protected
withdrawal that atomically marks one claim paid and transfers its exact coins.
Claims are non-transferable, do not expire or escheat, and grant no staking,
qualification, proposal, domain, or voting rights. The old modules and their
historical records become strictly read-only once their resolved balances move.

Reconciled active custom delegations and pending unbondings become
`LegacyBondClaim` records in `x/legacy_claims`. They MUST NOT become automatic
SDK delegations, because SDK delegation has different slashing, reward, and
unbonding semantics to which the claimant did not consent. A claimant withdraws
legacy coins and may opt into SDK staking separately.

Legacy profile history migrates independently from monetary claims. Unmatched
custom identities become historical profiles without validator or emergency
authority.

Qualification custody is reconciled separately. Its KV state is not a claimant
ledger: expiry retains `StakedAmount`, attempts the bank refund with an ignored
error, and does not record whether a successful refund occurred. Therefore an
expired qualification may describe either paid or unpaid coins.

Let `BQ` be the frozen qualification-module `uzrn` balance. An archive-complete,
deterministic replay of successful `MsgQualifyByStake`, explicit withdrawals,
and expiry-time bank results, or a separately approved exact manifest derived
from equivalent evidence, MUST establish the outstanding claimant total `Q`.
H4 requires `BQ = Q`; status plus `StakedAmount` is insufficient. Reconciled
amounts become `LegacyQualificationClaim` records in `x/legacy_claims`.
Stake-pathway qualifications become inactive legacy evidence, not target
competence. Missing replay evidence, failed automatic unlocks, duplicate
records, deficits, and unexplained surplus all fail closed.

H4 MUST also remove every runtime dependency on custom staking. Current app
wiring supplies that keeper to knowledge, custom governance, qualification,
emergency, alignment, and claiming pot. Each consumer MUST be rewired to its
target authority or retired explicitly. In particular, custom Guardian tier,
custom delegated amount, and custom active-validator flags confer no residual
panel, governance, emergency, qualification, alignment, or reward authority.

### 7.4 Governance migration

Custom LIP KV stores only aggregate `StakedAmount` and proposer data, not each
`MsgStakeLIP` contributor and amount. It cannot produce a claimant root by
itself. An archive-complete census MUST bind every successful `MsgSubmitLIP`
and `MsgStakeLIP` transfer as `(chain_id, height, tx_hash, msg_index, lip_id,
claimant, amount, tx_result)` and reconcile the resulting outstanding total
against the custom gov module balance. Missing or ambiguous contributor history
is a hard NO-GO unless a separately approved exact claimant manifest resolves
every transfer and balance difference.

H4 MUST:

- require custom LIP escrow to be zero or exactly matched by that
  archive-proven claimant manifest, then move resolved liabilities to
  `LegacyLIPClaim` records in `x/legacy_claims`;
- preserve legacy proposal and ballot bytes without treating them as current
  authority;
- disable every custom governance mutation and target dispatch;
- cancel or explicitly migrate non-terminal custom phases and research spends;
- preserve the knowledge research proposal/vote keys as legacy evidence and
  retire its proposal, vote, and event-only execute messages;
- require no SDK proposal in deposit, voting, or expedited-fallback state;
- install the electorate state and controller tally integration in disabled
  mode; and
- consume the exact `H4_PENDING` latch while preserving its receipt and enter
  `PRE_H5_STAKE_WEIGHTED`, in which only the exact transition proposal is
  admissible.

Passed-but-inert LIPs are not replayed. A desired action requires a new typed
SDK proposal under the authoritative execution path.

H5 MUST publish before activation:

- the complete initial controller set and member root;
- address bindings, overlap declarations, and controller-merger policy;
- concentration and effective-control report;
- action-class policies and exact thresholds;
- bootstrap sunset and open-mode predicates;
- the reviewed active emergency responder roster/policy and proof that it can
  issue the exact H5 plan-cancellation capability without legacy Guardian
  authority;
- deterministic tally and exact-rational boundary vectors proving balance and
  stake changes have no effect;
- roster derivation, future-epoch mutation, ballot-revision, and durable
  drop/cancel/terminal-receipt vectors; and
- independent review of the SDK/app classification, tally, voting-start, and
  cached-terminalization integration.

The exact H5 scheduling proposal is the final stake-weighted SDK governance
decision. It uses the dedicated `ELECTORATE_ACTIVATION` class and a
`PRE_H5_STAKE_WEIGHTED` mode receipt, not an `ElectorateSnapshot`. When that
proposal's sole typed handler executes successfully, it atomically schedules
the plan and enters `H5_PENDING`; SDK gov marks the proposal passed only after
that cached execution commits. The latch rejects every later proposal
submission, deposit, ballot, replacement, and ordinary cancellation. The
handler MUST fail if any other SDK proposal is in deposit, voting, or
expedited-fallback state.

The H5 upgrade handler MUST independently require zero active SDK proposals and
the exact pending digest before atomically changing governance mode and tally
policy to `CONTROLLER_SNAPSHOT`. A proposal MUST NOT begin under stake weighting
and finish under the controller electorate. The release packet MUST disclose
that H5 scheduling was stake-weighted and identify its voters, bonded-power
denominator, and outcome.

Until H5 commits, SDK governance remains stake-weighted and MUST be described
that way. H4 is not economic-to-governance decoupling by itself.

### 7.5 Domain migration

Domain-reference migration crosses live custody beyond domain proposals. H4
MUST first partition, for every denomination, all primary liabilities and
reserved balances in:

- the knowledge module and `knowledge_training_fund`, including challenge and
  contradiction stakes, patronage, demand and `KnowledgeBounty` records,
  augmentation escrow and live `TrainingFundEscrowLocked` amounts, contribution
  bonds, vesting obligations, and `SurvivalPendingReward` mint entitlements;
- capture-challenge stakes and `DomainBountyPool` entries;
- `vindication_escrow`, including every pending, refunded, swept, or deleted
  vindication entry and final record;
- sponsorship `EscrowRemaining`; and
- every nonterminal substrate-bridge `BondUzrn` and cap-gated
  `WitnessPendingReward` mint entitlement.

For each module account or separately identified compartment, frozen bank
balance MUST equal its backed outstanding, pending-settlement, and
manifest-labelled unresolved liabilities. Status fields alone are
insufficient: knowledge challenge settlement can log a failed transfer after a
round terminalizes while retaining stake data; capture-challenge expiry ignores
refund errors while retaining stake; and a rejected capture challenge can move
stake to `development_fund` while also increasing an unfunded domain-bounty KV
amount.

`SurvivalPendingReward` is not proof of an unpaid mint entitlement. Its release
path ignores failure from vesting-schedule creation, then deletes the pending
reward and emits a release event. Archive-complete replay or an exact manifest
MUST prove, for every attempted release, whether the vesting entitlement was
created. H4 MUST neither mint it twice nor discard it; any unproven disposition
is a hard NO-GO.

Vindication custody has the inverse missing-record risk. Expiry attempts to send
the summed escrow to `development_fund`, and execution attempts individual
refunds, but both paths can log transfer failure and then delete
`VindicationPending` entries. A final `VindicationRecord` may also report a
refund that did not move. Therefore neither surviving pending state nor final
records prove the `vindication_escrow` claimant ledger. Archive-complete replay
or an exact independently approved manifest MUST reconcile its bank balance and
every attempted transfer. Retired unpaid amounts become claimant-bound
`LegacyVindicationClaim` records; an amount whose claimant cannot be proven is
manifest-labelled unresolved balance with no release route, and any unexplained
balance is a hard NO-GO.

The migration MUST use archive-complete replay or an exact independently
approved manifest to classify every paid, unpaid, duplicated, and unfunded
entry. A backed continuing obligation may migrate in place only with its exact
canonical references and escrow. A retired unpaid obligation becomes an
`x/legacy_claims` record. An unfunded counter is quarantined as non-spendable
evidence unless separately recapitalized; it is never treated as a bank-backed
claim. Missing evidence, deficit, double liability, or unexplained balance is a
hard NO-GO before any `DomainRef` key is rewritten.

Legacy domain-proposal state is not a claimant ledger. In `x/knowledge`, a
successful `MsgProposeDomain` can transfer positive `msg.stake` into the
commingled knowledge module account, but the stored `Domain` retains neither
amount, expiry, resolution/refund status, nor transfer receipt. Empty,
malformed, or non-positive stake can create the same stored proposal shape;
three endorsements can make it active; neither path refunds it. Status,
proposer, endorsers, and module balance therefore prove neither claimant amount
nor outstanding liability.

Before H4, an archive-complete census of successful transaction bytes MUST bind
every legacy knowledge `MsgProposeDomain` as `(chain_id, height, tx_hash,
msg_index, proposer, domain, requested_amount, tx_result)` in a canonical
claimant manifest. Every positive claim requires block/transaction inclusion
and successful-execution evidence. Its total MUST reconcile with every other
liability and reserved balance in the commingled knowledge account.

Ontology is also ambiguous: `DomainProposal.Stake` remains populated after a
successful refund, expiry does not invoke the refund, and pass-time refund
failure is only logged. `status + stake` is not proof of an outstanding
ontology claim; archive replay or an exact claimant manifest is required.

H4 MUST require proof that no successful positive-stake proposal occurred or a
separately approved, balance-proven manifest covering every such transfer in
both modules. Missing history, ambiguity, duplication, deficit, unexplained
surplus, or an unmatched proposed/endorsed domain is a hard NO-GO. H4 MUST NOT
infer, pro-rate, burn, sweep, send to SDK governance, or refund an amount from
KV state, events alone, or account balance.

Validated amounts become one-time `LegacyDomainProposalClaim` records in
`x/legacy_claims`, keyed by transaction identity. They do not become SDK-
governance deposits or domain voting power and do not expire or escheat. Only a
manifest-labelled residual with no provable claimant may enter
`UnresolvedLegacyBalance`; unexplained or unmatched coins are a hard NO-GO.
Refund eligibility and canonical admission of the proposed domain are
independent decisions.

The domain migration MUST use a reviewed resolution manifest rather than an
automatic union heuristic.

The manifest MUST:

- enumerate both registries, cross-stratum links, logic zones,
  incompleteness acknowledgments, and all downstream references;
- preserve the five shared IDs with ontology structural fields as the starting
  canonical value;
- explicitly classify the 13 knowledge-only domains;
- preserve ontology-only `protocol` and `history`;
- create the four doctrine namespaces as `SYSTEM_DOCTRINE`;
- treat proto3 JSON's omitted ontology stratum zero as valid `AXIOMATIC`, not
  as an unknown value;
- resolve status, stratum, parent, ID-grammar, and alias conflicts;
- give every candidate an explicit primary, alias, tombstone, or quarantine
  disposition without automatically approving it as active; and
- abort on every unlisted ID or collision.

The migration order is normative:

1. verify the exact chain ID, committed `F` height/app hash, source version map,
   current equality of the frozen commitments, and manifest hash;
2. reconcile every legacy economic liability before deciding registry status;
3. build ontology v2 domain records/revisions/transitions, stratum and logic-
   zone revisions, cross-stratum-link revisions, acknowledgments, aliases,
   tombstones, and graph indexes;
4. backfill every persistent `DomainRef`, `StratumRef`, and logic-zone revision
   reference, quarantining empty or ambiguous references record-by-record
   rather than silently mapping them to `general`;
5. retire every runtime read and write of the knowledge legacy-domain prefix;
6. rebuild consumer secondary indexes from their primary records;
7. run the full reference, graph, index, doctrine, and balance census inside
   the handler and abort the block on mismatch; and
8. only then set target consensus versions.

Old knowledge-domain keys remain untouched but read-only for at least one
compatibility release. Mutating those bytes in an adversarial test MUST have no
effect on admission, lifecycle, confidence, capture policy, or qualification.
The same migration replayed from the captured runtime H4-1 artifact on two
independent machines MUST produce the same post-state root.

Legacy knowledge domain mutation type URLs return a stable
`ErrLegacyDomainMutationDisabled` or equivalent error naming the canonical
ontology route. They MUST NOT silently forward because the two proposal,
stratum, lifecycle, and escrow models are not wire-equivalent.

### 7.6 Consensus versions and store retirement

The implementation plan MUST include, at minimum:

- custom `zerone_staking` v1→v2 as legacy evidence/history with all mutations
  retired;
- new `legacy_claims` v1;
- new `controller` v1;
- new `verifier_profile` v1;
- `qualification` v1→v2 for non-economic, revision-pinned competence and
  immutable legacy pathway evidence;
- custom `zerone_gov` v2→v3 as historical/read-only state;
- new `electorate` v1;
- SDK `gov` v5→v6 for classification, ballot revisions, durable drop/cancel/
  terminal receipts, transition modes, and the complete tally/terminalization
  integration, with the transition receipt store installed at H4-F;
- `emergency` v2→v3 for controller-deduplicated responder authority with no
  custom Guardian dependency;
- `zerone_ontology` v1→v2;
- `knowledge` v6→v7;
- `vesting_rewards` v2→v3 for the sole typed research-fund disbursement and its
  inbound/outbound guard wiring;
- `capture_defense`, `capture_challenge`, `sponsorship`, and
  `substrate_bridge` v1→v2 for persistent domain references, including
  `StratumRef` where applicable; and
- an app-level coordinated migration that can validate all cross-store
  invariants and rewire knowledge, custom governance, qualification, emergency,
  alignment, and claiming pot before committing.

Version numbers above are target migration identities, not current registered
handlers. Generated types, store loaders, and compatibility services remain an
implementation review.

The implementation MUST also repair current round-trip gaps before claiming
the export/import gate: capture-defense export omits verification-history
entries, capture-challenge export omits paused-domain keys, and ontology export
omits logic zones and incompleteness acknowledgments. Knowledge genesis omits
`VindicationPending`, `VindicationRecord`, `KnowledgeBounty`, live
`TrainingFundEscrowLocked`, and `SurvivalPendingReward` state even though
related balances or entitlements can survive. These are known examples, not an
exhaustive allowlist: knowledge v7 MUST inventory every committed key prefix,
round-trip every primary and economic keyspace, deterministically rebuild every
secondary index, and abort on an unclassified key. Target export MUST preserve
all of this state after its reference and liability migrations.

The first migration MUST NOT delete legacy stores. After every legacy claim is
paid and one full export/import release proves compatibility, a later separate
upgrade MAY remove legacy module-account permissions and stores. Removal
requires zero balance, zero liability, query-retention or archive evidence,
and its own rollback rehearsal. The pre-H4 snapshot is coordinated-fork
evidence, not an automatic rollback after H4 commits; the old binary is invalid
at and after the H4 height.

## 8. Compatibility contract

For at least one full release after H4:

- legacy staking, LIP, research, seat, and knowledge-domain queries remain
  available;
- every response identifies itself as legacy or projected and names its
  canonical source where one exists;
- old mutation messages fail with stable typed retirement errors;
- knowledge domain queries project ontology lifecycle and stratum plus
  knowledge-owned metrics;
- historical events and records remain exportable; and
- clients receive a documented replacement message or query route.

Compatibility MUST NOT restore authority. A legacy route may read old bytes;
it may not create new legacy state or make a legacy record eligible for a new
process.

## 9. Release gates

H4 is **NO-GO** unless all of the following pass for the exact release:

1. H4-F has frozen every committed legacy writer; the fixed `F` commitments,
   migration manifest, successful `PRE_H4_STAKE_WEIGHTED` scheduler receipt, and
   exact `H4_PENDING` plan all match. The SDK-governance gate has held its
   drained set at zero.
2. A static authority-graph check finds no alternate validator-update,
   ordinary governance-execution, domain-registry, direct-fact-adoption, or
   research-disbursement writer, no alternate quarantine/pause authority, and
   no remaining runtime consumer of custom staking authority.
3. Custom staking balance, individual claims, pending unbondings, and
   aggregates satisfy `B = D + U`, or an explicit reviewed resolution manifest
   covers every difference without an unexplained balance.
4. Archive-complete evidence proves `BQ = Q` for legacy qualification custody;
   missing or failed refunds are claimant-bound, and stake-pathway records
   confer no target qualification.
5. Custom governance escrow is zero or equals the complete archive-proven
   contributor manifest, with no unresolved escrow.
6. Legacy domain-proposal liabilities are zero or equal complete,
   balance-proven claimant manifests for both source modules, with no
   unexplained or unmatched coins.
7. Every knowledge/training, vindication, capture, sponsorship, and substrate-
   bridge compartment reconciles per denomination to backed continuing
   obligations, legacy claims, and manifest-labelled unresolved balances;
   deleted pending entries, failed transfers, survival-reward vesting attempts,
   and bridge witness mint entitlements are replayed to a proven disposition,
   while deficits, double liabilities, duplicate mint entitlements, unfunded
   counters presented as claims, and unexplained balances are absent.
8. Every retired-path liability migrated to a source-labelled
   `x/legacy_claims` compartment has a bank balance equal to outstanding claims,
   pending withdrawals, and manifest-labelled unresolved balance; only the
   exact claimant withdrawal route can release backed coins.
9. Active cross-boundary processes are absent or deterministically migrated;
   every pending fact injection is receipted as cancelled and no legacy
   materializer can write it after H4. Knowledge incident, pause, and surgical-
   correction records are legacy audit evidence with no authoritative writer or
   consumer.
10. No SDK proposal remains in deposit, voting, or expedited-fallback state and
    `H4_PENDING` matches the exact successful transition receipt. H4 enters
    `PRE_H5_STAKE_WEIGHTED`, admits only the exact non-expedited transition
    proposal, and backfills no unverifiable pre-extension history.
11. Golden migrations account for all 24 checked-in candidate IDs with an
   explicit primary, alias, tombstone, or quarantine disposition; candidate
   count alone does not authorize active status.
12. Cross-stratum links, logic-zone revisions, incompleteness acknowledgments,
    aliases, tombstones, and graph indexes survive migration without being
    recreated from defaults or silently overwritten.
13. Every domain-, stratum-, and logic-zone-bearing primary record and
    secondary key maps to an exact revision-pinned reference; empty or
    ambiguous references are quarantined or resolved record-by-record.
14. One ontology creation admits a claim without dual seeding, while proposed,
    frozen, deprecated, archived, merged, aliased, malformed, and unknown IDs
    exercise their exact fail-closed action policy.
15. Changing legacy knowledge-domain bytes cannot affect admission, lifecycle,
    confidence, capture policy, or qualification.
16. Doctrine is excluded from epistemic confidence, reward, factual-support,
    and default training paths.
17. Controller/address/profile/validator bindings and profile lifecycle
    transitions are authenticated, replay-protected, and controller-
    deduplicated; qualification grants consume only permitted evidence.
18. Random staking operations cannot affect profile history or truth power
    outside their explicit SDK consensus effects. Panel processing remains
    disabled unless the hash-pinned selection, beacon, and decision policies
    pass deterministic adversarial, quorum, equality, tie, malformed, and
    missing-reveal vectors with one whole ballot per controller.
19. Every work-bond property test preserves balance-to-liability equality using
    a real bank keeper.
20. The canonical research-fund inbound router and scoped egress capability are
    installed; unapproved deposits, every legacy disbursement caller, and every
    out-of-handler egress fail in integration tests.
21. Fault-injection and restart tests prove SDK proposal, ballot-revision,
    dropped, cancelled, and terminal history commits atomically before active
    state is cleared and does not depend on ignored hooks. The H4 cancellation
    lane rejects partial deposits and late deadlines, reaches zero at its
    cutoff, and atomically clears only a matching plan/latch.
22. Export/import preserves SDK validators, legacy-claim compartments and
    receipts, vindication records and reconciled claims, profiles, governance
    proposals/ballot revisions/terminal receipts, electorate state,
    domain/stratum/logic-zone revisions, cross-stratum links, incompleteness
    acknowledgments, transitions, aliases, all knowledge primary/economic
    keyspaces, deterministically rebuilt indexes, all persistent references,
    capture verification history, and paused domains; no store key remains
    unclassified.
23. Two independent replays from the committed artifact produce the same
    post-state root.
24. Emergency quorum, approval, incident-scope, renewal, cancellation-set, and
    cumulative-cap vectors fail closed under duplicate controllers, repeated
    evidence, expiry, and zero denominators. Cancelling the exact pre-height H5
    plan atomically receipts the cancellation, clears only its matching latch,
    and restores `PRE_H5_STAKE_WEIGHTED`; halt → exact recovery → resume restores
    ordinary governance without an indefinite hidden Guardian gate.

H5 is additionally **NO-GO** unless:

1. `H5_PENDING` names the exact successful final stake-weighted scheduling
   proposal and plan, and no SDK proposal remains in deposit, voting, or
   expedited-fallback state;
2. the scheduling proposal has a complete `PRE_H5_STAKE_WEIGHTED` mode/tally
   receipt, contains only the dedicated activation message, and generic
   wrappers, mixed classes, unknown messages, and expedited proposals fail
   classification;
3. each snapshot derives from one active versioned roster, validates every
   controller/address binding and member root, and is unchanged by future-epoch
   roster or policy mutations;
4. the full tally path reads no SDK validator or bonded-token state;
5. changing a member's liquid balance or SDK/custom delegation cannot change
   eligibility, ballot power, quorum, or outcome;
6. numerator, denominator, thresholds, veto, and comparison rules come from the
   same immutable snapshot;
7. checked cross-multiplication passes equality, all-abstain, zero, one-unit,
   quorum, approval, and veto boundary vectors under every comparator;
8. two addresses assigned to one controller still produce at most one unit;
9. weighted, ineligible, duplicate-controller, and post-deadline ballots fail
   closed, while every accepted replacement remains archived;
10. drop, cancellation, and terminalization leave immutable canonical receipts,
    ballot history, and final tally inputs across faults and restarts, never
    deleting authoritative history;
11. a policy or roster change cannot modify its own proposal or any proposal
    already in voting;
12. bootstrap concentration and sunset are publicly queryable;
13. a reviewed emergency responder roster/policy is active and can exercise the
    exact H5 cancellation transition; and
14. the full transition has independent source, binary, migration, operational,
    and effective-control review.

Source publication, green tests, a merged pull request, or this document's
content hash satisfies none of those network activation gates by itself.

## 10. Consequences

This decision deliberately separates four kinds of power:

- stake secures block consensus;
- qualification and profile state establish epistemic eligibility;
- a non-transferable controller electorate governs policy; and
- ontology defines the vocabulary in which knowledge is admitted.

It removes the ability to treat money as policy voice or truth weight, but it
does not erase current custody concentration. It increases migration work,
introduces explicit controller, profile, and electorate modules, redesigns
qualification, and requires long-lived legacy query support. Those costs are
accepted because silently divergent consensus ledgers are more dangerous than
visible migration complexity.

## 11. Rejected alternatives

- **Keep two staking planes:** rejected because the custom ledger is neither
  consensus staking nor conservationally safe.
- **Convert custom delegations directly into SDK delegations:** rejected
  because it changes claimant risk and consent.
- **Use a fake staking keeper for non-economic SDK governance:** rejected
  because it hides electorate semantics behind a consensus-economic API.
- **Keep custom LIPs as a second executor:** rejected because two terminal
  decision records can disagree and custom execution is incomplete.
- **Use one DID-one vote now:** rejected because a self-certifying key binding
  is not independent-controller uniqueness.
- **Let ontology and knowledge synchronize opportunistically:** rejected
  because two writable primaries remain two authorities even when a happy-path
  hook copies values.
- **Automatically union domain taxonomies:** rejected because their strata,
  statuses, proposal economics, and doctrine treatment differ.
- **Delete legacy stores in H4:** rejected because audit, claimant, rollback,
  and client-compatibility evidence must survive the first transition.

## 12. Supersession and non-effects

This record supersedes future-target claims that custom Zerone staking should
remain a second delegation system, that custom LIPs should become the eventual
ordinary executor, or that knowledge should retain an independent writable
domain registry. Existing documents may continue to describe current or
historical behavior when explicitly labelled as such.

It does not supersede:

- the current operational observations in
  [UPGRADE_AND_INCIDENT_OPERATIONS.md](UPGRADE_AND_INCIDENT_OPERATIONS.md);
- accepted H1, H2, or H3 source identities;
- the current live network's queried state;
- the no-money-as-voice boundary in
  [MONEY-KARMA.md](constitution/MONEY-KARMA.md); or
- the requirement for separate governance, release, deployment, and
  production verification.

Until implementation and activation, the honest statement remains: Zerone's
current source has dual custom/SDK authority surfaces, bonded wealth still
affects SDK governance, and the founding household retains concentrated
effective control. This accepted design names the route out; it is not evidence
that the route has already been travelled.
