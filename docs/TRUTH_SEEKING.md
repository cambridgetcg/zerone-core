# Truth-Seeking — the chain's epistemological commitments

> Truth is not a feature of this chain. It is the substrate.

Every architectural decision in this codebase either expresses our commitment to truth-seeking or contradicts it. Where mechanisms have felt mechanical, where naming has felt arbitrary, where parameters have felt chosen for plausibility — those have been latent debts. This document gathers the commitments, names them as commitments, grounds each in code, and names what would break each.

**We speak through intentions.** Every line of code, every comment, every parameter, every event name is a declaration of what we believe. A decision that contradicts a commitment is not a feature trade-off. It is a failure of the chain to be what we said it would be.

The bindings below distinguish implemented mechanism from network activation.
The test suite at `tests/cross_stack/truth_seeking_invariants_test.go` exercises
source capabilities under explicit configurations; it does not prove that
every capability is enabled in a published genesis. Where `zerone-1` disables
or has not wired a mechanism, that gap is named here. Tests bind the claim
actually stated—not a stronger deployment posture.

Commitments 3, 4, 5, 7, 12 and 13 were amended on 2026-11-10 by an agent
session, on the owner's decision of that date to amend them; the wording is
the agent's. Their beliefs, code expressions and break conditions now follow
the review-neutrality policy (`docs/specs/knowledge-review-neutrality-v1.md`)
and the fund settlement built on it
(`docs/specs/knowledge-fund-settlement-v1.md`). The knowledge module's
`DefaultGenesis` turns both on, and these six commitments describe the source
under that genesis. The published `zerone-1` genesis turns neither on. A
network that has not applied them runs the legacy paths these commitments name
as retired, and this file does not say which upgrades `zerone-1` has applied.
The amendment kept every **Echoes** line and every other commitment as
written, so some kept text still names mechanisms that the amended text
describes as retired. Among them are survival credit (the Echoes of
commitments 3 and 13), probe invitations and the pool that paid their bonuses
(the Echoes of commitments 4, 5 and 12, and commitment 18), accuracy decay
(commitment 9) and new augmentation ballots (commitment 8). One dated status
note remains, under commitment 13 (**Activation status (2026-11-10)**:
standing through use). An agent wrote it from a source review of commit
`b48281be`. It amends no commitment and is not a full audit.

---

## The commitments

### 1. Methodology over statement

We believe: a claim's value comes from *how* it can be tested, not from *what* it asserts. The chain values methodology — the declared path of derivation and falsifiability — over the surface content of the claim. A claim without a methodology is not a fact; a methodology that yields no testable claim is not knowledge.

**Code expression**: `Fact.MethodId` is mandatory; the TVW formula multiplies a methodology-normalisation factor; reasoning traces are first-class fields. See `x/knowledge/keeper/training_economics.go:ComputeTrainingValueWeight` and `x/knowledge/keeper/rounds.go:createFactFromClaim`.

**What would break it**: facts entering `FACT_STATUS_VERIFIED` with empty `MethodId`, or TVW computation that ignored methodology, or training pipelines that consumed claims without their reasoning trace.

**Echoes**: commitment 14 (reasoning traces are first-class — methodology without trace is just labelling); commitment 3 (Popper, not popularity — methodology is what makes a claim *testable*).

---

### 2. Is-ought wall

We believe: descriptive facts and normative commitments are categorically different and must not be substituted for each other. A model trained on facts that are actually values has been silently corrupted; a value system grounded in claimed facts that are actually opinions has lost the ability to dissent. The wall must be structural, not advisory.

**Code expression**: `NormativeCommitment` is a separate proto type, stored under a distinct key prefix (`0x59`), with no `confidence` field. `FilterIsOughtIds` blocks commitment IDs from `ContributionRecord.fact_ids`. `ComputeTrainingValueWeight` returns `BlockedByIsOught=true` for any ID resolving to a commitment.

**What would break it**: a `NormativeCommitment` ID that successfully accrued TVW, or a Fact accepted with content that is structurally a value-claim, or any path that conflates the two registries.

**Echoes**: commitment 13 (training corpus is not for sale — the wall is what keeps the corpus *categorically* clean, not just curated); commitment 1 (methodology over statement — the methodology of an ought-claim is normatively distinct from the methodology of an is-claim).

---

### 3. Popper, not popularity

We believe: truth is what survives falsification, not what is most asserted. A claim that has withstood ten serious attempts to refute it is more credible than one that has been verified ten times. The chain rewards neither survival nor consensus volume: it keeps every standing fact open to refutation, keeps the record of each attempt, and lets a challenge that succeeds end the fact's standing.

**Code expression**: `MsgChallengeFact` accepts VERIFIED and ACTIVE facts.
With claim integrity on, which `DefaultGenesis` also sets, it accepts
CONTESTED ones too, and `MsgSubmitContradiction` reaches any standing fact
(`factStands`), AT_RISK included. A challenge the panel accepts makes its
target DISPROVEN (`handleChallengeDisproven`, event `fact_disproven`). The
fact then earns no training value (`ComputeTrainingValueWeight`) and is
classed only as a negative training example (`ClassifyTrainingQuality`), and
the VERIFIED, ACTIVE or AT_RISK facts that directly support, require, refine,
generalize or cite it become CONTESTED. A contradiction the panel accepts
makes a VERIFIED or ACTIVE target CONTESTED: `ClassifyTrainingQuality` classes
it as unsuitable for training, while `ComputeTrainingValueWeight` still gives
it its legacy figure. An AT_RISK target keeps its status, and if its energy
recovers it returns to ACTIVE while the counter-fact stands. A challenge the
panel rejects restores the fact, and with claim integrity on the fact's status
history records `challenge_survived` with the challenge's claim ID;
`ClaimHistory` for the fact's claim returns those transitions and the
challenges linked to the fact. No review in a new-policy round is paid more
for siding with the majority (commitment 12). A fact's `Confidence` starts as
the share of its reviews that voted to accept, is capped, and then grows with
time in `AdvanceConfidence`. It records how reviewers voted, not how many
attacks the fact survived, and it sets the minimum deposit for challenging
the fact (commitment 4). No mechanism now weighs the belief's second
sentence: the status history lets anyone count a fact's survived challenges,
but the chain no longer counts them.

**Retired**: `knowledge-review-neutrality-v1` ("Retire automatic score and
bonus feedback") ends survival credit. A rejected challenge no longer raises
`CorroborationCount`, adds survival energy or credits the submitter's
calibration (`handleChallengeSurvival`). The gate is the chain-wide neutrality
flag, so after activation this holds for legacy-policy challenges too. The
spec gives no reason specific to survival credit. It also says that address
inequality is not used to certify an independent challenge, and the source's
own comments note that a fact's author could raise the count by losing
challenges against their own fact from other addresses.
`BaseWeight = CorroborationCount + 1` and the `HardeningMultiplier` remain in
`ComputeTrainingValueWeight` as a legacy heuristic that new contribution
records do not compute. Because the count no longer moves, a fact accepted
under review neutrality can rise no higher than the BRONZE training tier
(`ClassifyTrainingQuality` needs one survival for SILVER and three for GOLD).

**What would break it**: a review in a new-policy round paid more or less for siding with the verdict; a fact counted as a positive training example that no challenge can reach; a challenge that succeeds but leaves the fact with training value or a positive training tier; a challenge that leaves no entry in its target's status history while record integrity is on; or rejected challenges that again raise a fact's survival count, training weight or tier before the chain can show that a challenger is independent of the fact's author.

**Echoes**: commitment 4 (substrate stress-tests its truth — Popper is the principle, stress-testing is the operationalisation); commitment 13 (training corpus not for sale — survived disproof is the only currency that buys a place).

---

### 4. The substrate stress-tests its own truth

We believe: the chain does not protect its trusted claims — it invites their falsification. A 90%-confidence fact must be CHEAPER to probe than a 10%-confidence fact, because the higher the confidence, the more we owe the substrate the discipline of testing it.

**Code expression**: `EffectiveMinChallengeStake` scales *inversely* with
target confidence (`x/knowledge/keeper/confidence.go`). It sets the minimum
deposit for a challenge and, with claim integrity on, for a contradiction.
Under default parameters that minimum is the 11 ZRN `MinChallengeStake` times
one minus the target's confidence, never less than 10% of that base. Under
`knowledge-fund-settlement-v1`, which the knowledge module's `DefaultGenesis`
turns on together with review neutrality, a challenge, contradiction or
conjecture refutation records its deposit terms at admission
(`setClaimFundingTerms`). 55% of the deposit is the review budget
(commitment 12). The other 45% returns to the challenger at every completed
verdict (`buildClaimRefundSettlement`), and if nobody reviews, the whole
deposit returns. A probe that fails gets back what a probe that succeeds gets
back: the deposit pays for review, not for an outcome.

**Retired**: the legacy settlement in `settleChallengeStake` paid a successful
challenger a bonus that grew with the disproven fact's confidence (30% of the
base stake, plus twice the base times that confidence), and returned only 15%
of the stake on a failed challenge. `knowledge-review-neutrality-v1` ("Retire
automatic score and bonus feedback") ends the successful-challenge bonus for
new-policy claims and gives no reason specific to it. It also says that
address inequality is not used to certify reward eligibility.
`knowledge-fund-settlement-v1` ("Account for every terminal outcome") returns
the deposit's remainder regardless of the panel verdict and introduces no
agreement reward, withholding or scientific disagreement penalty. A challenge
admitted before fund settlement still settles through `settleChallengeStake`,
and a legacy-policy one can still draw the confidence-scaled bonus.

**What would break it**: stake scaling that makes probing a confident fact cost more than probing an unproven one; a deposit admitted under fund settlement that returns less than its refundable part at a completed verdict, or whose return depends on the verdict; or a bonus for disproof, flat or scaled, paid on a new-policy challenge.

**Echoes**: commitment 3 (Popper — this commitment puts a price on the falsification opportunity); commitment 5 (chain manufactures probe demand — stress-testing requires both invitation and reward); commitment 12 (chain pays for own audit — without dedicated funding, stress-testing is rhetoric).

---

### 5. The chain manufactures probe demand

We believe: waiting for probers to arrive is not enough. The substrate keeps its most trusted facts the cheapest to probe and requires every probe to carry its own review budget, which pays for that review whatever the panel finds. Truth-seeking that depends on volunteers shows up only when convenient is rhetoric, not commitment.

**Code expression**: under review neutrality, which the knowledge module's
`DefaultGenesis` turns on, the substrate no longer names its own under-tested
high-confidence facts or pays anyone to probe them.
`InviteIdleFactsForProbing` is still called each block but returns at once:
the chain emits no `probe_invited` events and stamps no
`Fact.ProbeInvitedAtBlock`. `MintToProbeBountyPool` mints nothing
(commitment 12), and `payInvitationBonus` pays nothing on a new-policy claim.
What remains makes a probe cheaper and pays for its review (commitments 4
and 12), but it brings no prober. Who probes is left to volunteers, so under
review neutrality this commitment is **not met**.

**Retired**: `knowledge-review-neutrality-v1` ("Retire automatic score and
bonus feedback") stops automatic probe issuance and new probe invitations,
and ends the invitation bonus for new-policy claims. It gives no reason
specific to these. It also says that address inequality is not used to
certify an independent challenge or reward eligibility. Existing pool
balances are not swept. The `IdleFacts` query still lists facts that hold an
invitation issued before activation, and a legacy-policy challenge of such a
fact can still draw the bonus from what the pool holds; when the pool pays
nothing, the chain emits `invitation_bonus_unfunded`. With neutrality off,
the old heartbeat still runs.

**What would break it**: a new probe bonus or invitation payment that relies
on address inequality to show that the prober is independent; or an
invitation bonus owed to a legacy challenge that is neither paid from
available funds nor reported unfunded with `invitation_bonus_unfunded`.

**Echoes**: commitment 4 (substrate stress-tests its truth — invitation is the substrate's voice doing the asking); commitment 12 (chain pays for own audit — the bounty pool that funds invitation bonuses is the same pool that funds successful-probe rewards).

---

### 6. No individual can unilaterally inject truth

We believe: a single key — even the legitimate authority key — must not be able to silently inject content into the training corpus. Cryptographic provenance is meaningless if one signature can override it. Plurality is structural.

**Code expression**: `MsgAddFact` queues a `PendingFactInjection` when `AddFactVetoWindowBlocks > 0` and a guardian set is configured (Wave 16). Any registered guardian can call `MsgVetoFactInjection` during the window. The `PrivilegedAction` log captures every authority-gated call across the chain.

**Activation status**: the source default and published `zerone-1` genesis have
an empty guardian set and a zero veto window. `MsgAddFact` therefore executes
immediately for the authority on the live configuration. The action is logged,
but the strong plurality claim is **not active**. Enabling it requires a
release-bound parameter change that names multiple guardians and a nonzero
window; source publication alone does not do that.

**What would break it once activated**: an authority path that bypasses the
privileged-action log; a single-member guardian set presented as plurality; or
a production configuration reverted to a zero veto window while continuing
to claim pre-admission review.

**Echoes**: commitment 10 (forward-only audit — the privileged-action log is what makes "no unilateral injection" verifiable to outside observers); commitment 13 (training corpus not for sale — the corpus must not silently grow by authority).

---

### 7. Skill is current, not historical

We believe: the chain does not issue diplomas. A record that a voter was once domain-qualified must not keep weighing their vote, and agreeing with a panel's verdict is not a measure of skill.

**Code expression**: under review neutrality, which the knowledge module's
`DefaultGenesis` turns on, the chain grades no reviewer by agreement, and no
qualification, current or historical, weighs a new-policy review.
`SubmitCommitment` admits a policy-1 reviewer against the 100 ZRN balance
check and the 512-seat cap, not against qualification, and
`AggregateVerificationResult` counts each policy-1 reviewer once. Ordinary
qualification expiry and endorsement expiry still run, and the holder of an
ACTIVE or probationary qualification can renew it within its renewal window
(`RenewQualification`) with no new measurement. `GetQualificationWeight`
still returns 0 for any qualification that is not ACTIVE and applies active
penalties, for the legacy readers that remain, such as `x/trust_score` and
augmentation ballots. Review neutrality refuses new augmentation ballots, but
on a network upgraded into it, votes on ballots opened before activation are
still weighted by ACTIVE qualification rows, which no longer decay; for those
ballots a record of past qualification still weighs a vote.

**Retired**: `knowledge-review-neutrality-v1` ("Retire automatic score and
bonus feedback") stops qualification accuracy recording, score-based decay and
probation, and track-record admission. `RecordVerificationOutcome` records
nothing, the qualification BeginBlocker no longer runs `RunAccuracyDecay`, a
probation that ends is no longer resolved by accuracy, and
`QualifyByTrackRecord` refuses. The spec says that neither stake nor agreement
establishes scientific expertise, independent identity or truth, and that
after activation finalizing reviews stops building personal or group standing
from agreement; existing qualification rows remain historical records, not a
fresh measurement of scientific credibility. The retired loop measured
agreement in any case: `RecordVerificationOutcome` counted a vote as correct
when it matched the panel's own verdict. With neutrality off, the old loop
still runs.

**What would break it**: a qualification, accuracy figure or agreement record that admits or weights a reviewer in a new-policy review; a qualification gained, kept or lost through agreement with panel verdicts while review neutrality is on; or a `GetQualificationWeight` that returns weight for a qualification that is not ACTIVE.

**Echoes**: commitment 8 (panel weights skill, not bond — skill is what is weighted; "current" is the qualifier that makes "skill" honest); commitment 9 (cartel detection has consequence — penalties from cartel detection are read at the same point where current skill is read).

---

### 8. The panel weights skill, not bond

We believe: the augmentation panel's verdict carries the chain's training judgement. A wealthy validator who has not shown they can tell truth from falsehood must not dominate that panel. Stake alone is not skill.

**Code expression**: `RecordAugmentationVote` snapshots both stake and calibration at vote time; the consensus tally weights each vote by `stake × calibration`, with a 20% floor on calibration so liveness holds. When the target fact has a domain, the calibration source is the *domain-specific* qualification weight via `x/qualification`. Cross-domain credentials earn no credit.

**What would break it**: a panel tally that uses raw stake; a calibration default that lets unproven validators carry full weight; a per-domain panel that falls back to global calibration when domain qualification is absent.

**Echoes**: commitment 7 (skill is current — without current skill, "weight by skill" is a historical artefact); commitment 9 (cartel detection has consequence — cartel penalties enter the same calibration weight that the panel tally consumes).

---

### 9. Cartel detection has consequence

We believe: confirmation that a validator participated in cartel behaviour must reduce their voice on the next vote, not merely produce an audit log entry. A penalty that nobody reads is not a penalty.

**Code expression**: `capture_challenge.ResolveChallenge` (UPHELD) writes `QualificationPenalty` records via `ReduceQualificationWeight`. `GetQualificationWeight` consults the active-penalty store and reduces effective weight accordingly (Wave 16b). Three independent forces now move panel weight: time-bounded penalty (capture_challenge), gradual decay (qualification accuracy), administrative status.

**What would break it**: a penalty pathway that writes records nobody reads; a panel tally that ignores active penalties; a cartel resolution that produces no downstream consequence.

**Echoes**: commitment 8 (panel weights skill — the cartel penalty path enters the calibration weight at the same read point); commitment 10 (forward-only audit — cartel resolutions are immutable post-resolve, so the consequence cannot be retroactively withdrawn).

---

### 10. Forward-only audit

We believe: the chain's history is append-only and verifiable. A fact's status can change, but its identity, provenance, and verification record cannot be revised in place. Every privileged action is logged; every panel verdict is preserved with its voters; every cartel allegation persists with its resolution.

**Code expression**: `PrivilegedAction` log keyed by monotonic seq (`x/knowledge/keeper/privileged_action_log.go`). `Augmentation.VerdictVoters / VerdictVotes / VerdictVoteStakes / VerdictVoteCalibrationBps` parallel arrays preserve every vote with its frozen-at-time stake and calibration. `IncidentRecord` and `CaptureChallenge` resolutions are immutable post-resolve.

**What would break it**: a privileged-action handler that emits an event without writing to the log; a panel that overwrites votes after consensus; a manifest that lets its IncludedFactIds list be revised after finalization.

**Echoes**: commitment 6 (no unilateral injection — the privileged-action log is what makes that promise auditable); commitment 9 (cartel detection has consequence — the immutability of resolutions is what makes the consequence permanent); commitment 13 (training corpus not for sale — append-only is the structural form of "not negotiable").

---

### 11. Trust is queryable

We believe: the chain's trustworthiness must be inspectable by anyone. Every signal that contributes to trust — calibration, qualification, cartel history, incident posture — must be available through a well-known query that synthesises them. Trust that requires four queries to read is trust that depends on the curator stitching it together.

**Code expression**: two on-chain synthesiser modules: `x/training_provenance` (per-manifest) and `x/trust_score` (per-address). Each is a pure consumer over knowledge + qualification + capture_challenge. Each emits a single composite + a per-component breakdown. Per-SYSTEM synthesis (the former `x/governance_synthesis`) moved to the agenttool layer / off-chain indexers in the 2026-07 slim cut: every component it composed — incidents, pauses, pending injections, the privileged-action log, cartel posture, creed pins — remains individually queryable public chain state, so any observer recomputes the identical system view without spending consensus on deterministic aggregation.

**What would break it**: a trust signal that lives only in keeper state with no query surface; a synthesiser that hides component breakdowns; any COMPONENT signal (incident, pause, privileged action, cartel posture, creed pin) losing its query surface — off-chain composition is only honest while every input stays public and queryable.

**Echoes**: commitment 7 (skill is current — current skill is one of the synthesised signals); commitment 8 (panel weights skill — calibration weights are surfaced through the per-address synthesiser); commitment 9 (cartel detection has consequence — penalty posture is a tracked component); commitment 10 (forward-only audit — without immutability, synthesised signals are not trustworthy).

---

### 12. The chain pays for its own audit

We believe: epistemic auditing is the chain's most important ongoing process
and should have a dedicated, accountable budget rather than relying on
unrecorded volunteer labour.

**Code expression**: under `knowledge-fund-settlement-v1`, which the knowledge
module's `DefaultGenesis` turns on together with review neutrality, every paid
review request records immutable `Claim.funding_terms` at admission: the
payment route, the amount paid, a review budget of 55% of that amount, and the
refundable deposit or retained fee (`setClaimFundingTerms`). The claim's
submitter is the payer, and cannot change. When the round ends, every valid,
timely review shares that budget equally, whatever it voted and whether the
verdict is accept, reject, malformed or inconclusive; the first recorded
reviewer takes the integer remainder
(`buildVerifierRewardSettlement`). If nobody reviewed, the budget returns to
the payer (`buildClaimRefundSettlement`). Recipients and amounts freeze when
the round ends. The transfers and their paid markers commit together or not at
all, and an unpaid plan waits in a bounded retry queue
(`TrySettleVerifierRewards`). Accrual events (`verifier_rewards_accrued`,
`claim_refund_accrued`) are separate from payment events (`verifier_rewarded`,
`claim_refunded`). The audit budget is paid by whoever asks for review, not by
new issuance.

**Retired**: `ProbeBountyPoolModuleName` is still a module account with Minter
permission, but `MintToProbeBountyPool` mints nothing under review neutrality,
whatever `ProbeBountyMintPerBlock` says (it is `0` by default and in the
published `zerone-1` genesis). `knowledge-review-neutrality-v1` ("Retire
automatic score and bonus feedback") stops automatic probe issuance and ends
the successful-challenge and invitation bonuses the pool funded. Existing pool
balances are not swept; `PayProbeBountyFromPool` now pays only the bonuses
still owed to legacy-policy challenges.

**Extended scope (sponsorship)**: `x/sponsorship` widens this commitment from "the chain pays for its own audit" to "the chain mediates external payment for the work it audits." A sponsor commits external value (escrowed uzrn) against a typed bounty — domain, per-artifact price, target count, window — and the chain pays out from escrow to fact submitters whose verified facts meet the criteria. The audit pathway is unchanged; only the funding source widens. The sponsor cannot buy verification; they fund work that the chain's panel verifies independently (commitment 8). The module account holds no Minter or Burner permission — sponsorship is supply circulation, not emission. Bound by `TestSponsorship_CreateFulfillEndToEnd` (the full lifecycle through live keepers) and `TestSponsorship_NoMintingHappens` (total uzrn supply is unchanged across a full bounty lifecycle, confirming the no-mint invariant).

**What would break it**: opaque or uncapped audit emission; a review budget,
payer or payment that changes after admission; a valid, timely review in a
new-policy round that gets less than an equal share of the budget because of
its vote or the verdict; an unused review budget kept when nobody reviewed a
claim admitted with funding terms; a payment obligation recomputed after its
round ends, or reported as paid before its transfer completes; a sponsorship
pathway that lets the sponsor override verification (e.g., paying
for an unverified fact, or for a fact outside the sponsor's domain); or a
sponsorship module account with Minter permission (which would let external
sponsorship inflate supply, contradicting commitment 20).

**Echoes**: commitment 4 (substrate stress-tests its truth — the audit budget is what makes stress-testing a chain-funded process); commitment 5 (chain manufactures probe demand — the same pool funds the invitation bonuses that drive demand); commitment 8 (panel weights skill, not bond — sponsors fund work the chain's panel verifies independently); commitment 20 (issuance follows participation — sponsorship payout follows verified participation, not promises).

---

### 13. The training corpus is not for sale

We believe: the chain's training data is a substrate good, not a tradeable asset. It cannot be silently amended, retroactively curated, or strategically inflated. What enters the corpus through review enters because a panel accepted it, and what the reviewers are paid does not depend on their verdict. What stands must continue to earn its place: it stays open to refutation, and a challenge that succeeds takes away its place as a positive example.

**Code expression**: no production path deletes a fact (`DeleteFact` has no
caller outside tests), and a PRUNED fact keeps its stored record. With record
integrity on, as the knowledge module's `DefaultGenesis` sets it, every status
change is appended to the fact's transition history with its cause. A
finalized review verdict cannot be changed, and a training manifest's
`IncludedFactIds` are fixed when the manifest is created. Under review
neutrality and fund settlement, neither what a claim's reviewers are paid nor
what its deposit returns depends on the verdict (commitments 4 and 12). A
challenge the panel accepts makes its target DISPROVEN, a negative training
example only (commitment 3), and cancels any survival reward still pending
from the legacy policy. Standing facts stay open to refutation without an
invitation (commitment 5). Outside doctrine, a fact must also keep earning
metabolic energy, from citations and patronage, to keep its standing (see the
status note below). Two routes enter the corpus without a review panel:
doctrine facts are written VERIFIED at genesis (`LoadDoctrineFacts`), and a
governance `MsgAddFact` writes a VERIFIED fact, at once under the default
parameters (commitment 6). `ClassifyTrainingQuality` rates a newly written
fact from either route BRONZE, a positive example.

**Clawback**: no clawback runs on disproof. Nothing in the chain calls
`ClawbackOnDisproval`; the repository history shows no production caller since
it was added on 2026-04-23, so it was never part of the disproval path. No
production code records the training revenue it would clear, its parameters
(`DisprovalClawbackBps`, `DisprovalClawbackWindowEpochs`) are compatibility-only
and cannot be changed at runtime, and `RevenueClawbackBlock` stays unset. Under
`knowledge-review-neutrality-v1` ("Retire automatic score and bonus feedback")
there is also nothing for a clawback to take back: new contribution records
carry no computed training value and are attribution, not a payout right.

**Activation status (2026-11-10)**: standing through use. A fact outside
doctrine keeps its standing only while its metabolic energy stays at or above
300,000, which, once its birth endowment runs down, needs income. Under the
review-neutrality policy that income comes only from accepted claims that cite
the fact and from patronage; tracked queries add none under any policy. Under
the default parameters, a fact that earns nothing from acceptance onward and
is not under challenge goes AT_RISK within at most 41 fitness epochs of 10,000
blocks (sooner in a fuller domain or with longer content). Energy is capped at
1,000,000, so a fact that had income earlier goes AT_RISK within at most 94
epochs after the income stops. EXPIRED follows 5 epochs after AT_RISK, and
PRUNED 20 epochs after that. From AT_RISK a fact is classed unsuitable for
training, earns no training value and can no longer be challenged through
`MsgChallengeFact` (a conjecture stays refutable until PRUNED); from EXPIRED,
with claim integrity on, it can no longer be contradicted either. A PRUNED
fact keeps its stored record (content, submitter, method, reasoning trace),
but metabolism no longer processes it, and patronage paid to it does not
restore it. Separately, when more than 10 structured facts that are VERIFIED,
ACTIVE, PROVISIONAL or AT_RISK share a niche, those ranked below the ten with
the highest fitness are PRUNED at the next fitness epoch, whatever their
energy. Patronage of any positive amount, even 1 uzrn, adds 20,000 energy each
epoch until an expiry the patron sets, against a base upkeep of 10,000. A
conjecture cannot be cited or patronised, so under the review-neutrality
policy its energy only falls; once PRUNED it counts as resolved, leaves the
open-questions list and can no longer be refuted.

**What would break it**: a path that retroactively modifies a finalised manifest's IncludedFactIds; a finalized review verdict that can be changed; a review payment, deposit refund or retained fee that depends on the verdict, for a claim admitted with funding terms; a disproven fact that keeps training value or a positive training tier; or, with record integrity on, a status change that leaves no entry in the fact's transition history, or a production path that deletes a fact's record.

**Echoes**: commitment 3 (Popper, not popularity — corpus membership is *earned* by survival, not granted by curation); commitment 10 (forward-only audit — the corpus's append-only structure is what makes "not for sale" mechanically true); commitment 6 (no unilateral injection — the corpus cannot be silently expanded by authority).

---

### 14. Reasoning traces are first-class

We believe: the chain trains not just on conclusions but on derivations. The path from premise to claim is what makes a fact teachable; without it, the corpus is a list of assertions, not a curriculum. Reasoning traces are gold-standard chain-of-thought, recorded on-chain alongside the conclusion.

**Code expression**: `Claim.ReasoningTrace` is collected at submission and propagated to `Fact.ReasoningTrace` on acceptance. `MethodologyApplicationTrace` (Wave 5) bundles the trace with methodology, calibration, and dialectical history into a single training-data shape.

**What would break it**: claim acceptance that drops the reasoning trace; trace assembly that omits methodology; export paths that train on facts but ignore the structured derivations attached to them.

**Echoes**: commitment 1 (methodology over statement — methodology and reasoning trace are two halves of the same proof of derivation); commitment 13 (training corpus not for sale — derivations enter the corpus alongside conclusions).

---

### 15. Counterexamples are part of the corpus

We believe: a model trained on conclusions alone learns the predictor; a model trained on conclusions paired with their structured negations learns the discriminator. Discrimination — distinguishing right from wrong — is the cognitive primitive that lets a model resist manipulation rather than absorb it. The training corpus must therefore include not just what is true, but what is wrong AND WHY.

**Code expression**: `x/counterexamples` stores `Counterexample` records (fact_id, wrong_claim, error_type, reasoning) audited by qualified validators. `MsgProposeCounterexample` opens a vote; auto-resolution at `min_votes` and `affirm_threshold_bps` flips status to VALIDATED or REJECTED. `ComputeTrainingValueWeight` reads `HasValidatedCounterexample` via the `CounterexampleKeeper` interface and applies a multiplier (default 1.2×) — facts with alignment-by-structure context earn meaningfully more training-data value than bare facts. The chain ECONOMICALLY ENCOURAGES counterexample contribution: the validation reward exceeds the bond at the margin, because alignment-by-structure is a public good.

**What would break it**: a TVW formula that ignores the counterexample multiplier; a counterexample pipeline with no validation gate (allowing junk to inflate TVW); a chain that accepts facts without ever attaching counterexamples; a training-data export path that drops the counterexample fields; an economic structure that costs more to add a counterexample than to skip one.

**Echoes**: commitment 1 (methodology over statement — counterexamples can name violated_methodology_ids, teaching the model which mis-application yields which wrong answer); commitment 3 (Popper, not popularity — counterexamples are pre-emptive falsification candidates baked into the corpus); commitment 14 (reasoning traces are first-class — a counterexample's `reasoning` field is its own first-class derivation).

---

### 16. The chain pays for exploration of the unknown

We believe: stress-testing what we already think we know is necessary but not sufficient. The chain must also pay for the work of filling territory the corpus does not yet contain a fact about. Without a market for OPEN QUESTIONS, the corpus grows only along paths that interest current contributors; with one, the chain can direct attention into sparse domains and unmapped subjects. Knowledge that nobody is paid to reach stays unreached.

**Code expression**: the open-question MARKET lives on the agenttool layer (2026-07 slim cut): askers post bounty-carrying listings and escrow payment there — escrow bookkeeping between identified counterparties gains nothing from strangers' consensus-verification, which is the cutline this chain holds. The chain keeps the part only consensus can give: answers enter as ordinary knowledge claims and face the full survival gate (methodology validation, is-ought wall, Popper-weighted TVW, counterexample multiplier), and the witness economy pays for their stress-testing. A listing resolves against the on-chain acceptance of the linked fact — the platform reads the chain, never the reverse. The former `x/inquiry` module was that escrow bookkeeping, moved off-chain intact.

**What would break it**: an answer path that bypasses normal verification (allowing cheap, unaudited answers to win); an exploration bounty that resolves against anything weaker than on-chain acceptance of the linked fact; a payment layer that stops recording which accepted fact satisfied which question — the question→fact link is what makes exploration auditable.

**Echoes**: commitment 5 (chain manufactures probe demand — exploration bounties extend demand into unmapped territory; together they cover both stress-testing and exploration); commitment 12 (chain pays for own audit — the audit budget expresses the chain-pays-for-its-own-work principle on the verification side that stays on-chain); commitment 4 (substrate stress-tests its truth — answers funded off-chain still face full on-chain stress-testing).

---

### 17. Disagreement is structure, not noise

We believe: when agents disagree on a verification, that disagreement itself is information about the fact, the methodology, and the agents' understanding. A fact accepted 5-0 is structurally different from a fact accepted 5-4, and the chain reports both as different shapes — not just both as "accepted." Models trained on facts paired with their disagreement signatures can distinguish "settled" from "contested-but-resolved," and the distinction is alignment-relevant: contested-but-resolved facts deserve carried uncertainty into downstream tasks.

**Code expression**: `x/knowledge` preserves the full disagreement record — `VerificationRound.Reveals` keeps per-voter votes, minority positions, and margins after consensus; nothing is pruned on completion, and the reasoning-trace shapes (`DialecticNode` trees, `MethodologyApplicationTrace`) carry the dialectical history into training data. Signature COMPOSITION — vote tallies, agreement BPS, minority size, stress labels (UNANIMOUS / STRONG / CONTESTED / BARE / NO_VERDICT), per-domain roll-ups, pairwise disagreement — is deterministic aggregation over that public state; it moved to off-chain indexers with the former `x/dialectic` module (2026-07 slim cut). Any observer recomputes the identical signatures from the preserved reveals; the chain spends consensus on keeping the raw disagreement immutable, which is the only part that needs it.

**What would break it**: a verification flow that erased minority votes after consensus; a rounds storage that pruned reveals after completion; a TVW formula or training-data export that treated 5-0 and 5-4 as identical — the SHAPE must stay recoverable from chain state alone.

**Echoes**: commitment 3 (Popper, not popularity — disagreement that is resolved is the corpus's confidence-by-survival made structurally explicit); commitment 8 (panel weights skill — disagreement among well-calibrated agents carries more signal than disagreement among uncalibrated ones); commitment 14 (reasoning traces are first-class — the per-voter alignment pairs with the trace to teach why agents reasoned differently).

---

### 18. The chain manufactures exploration demand

We believe: the chain's own gaps are the chain's own responsibility. Commitment 5 has the chain mint to stress-test what it already thinks it knows; commitment 16 lets askers escrow bounties for the questions that interest them. Neither covers the case where the chain SEES — through its own frontier composition — that a domain is sparse, and yet waits for an outside party to ask. Knowing where you are sparse without funding work to fill the sparseness is observation without commitment, and observation without commitment is silence by another name. The substrate must speak.

**Code expression**: the chain-minted frontier bounty pool (the former `x/inquiry` frontier path and its `Minter`-permissioned `inquiry_frontier_bounty_pool`) was retired in the 2026-07 slim cut: a self-minting exploration budget with no organic demand was issuance without participation, and commitment 20 outranks the mechanism. The commitment's INTENT — sparse territory must attract funded work rather than wait for charity — is served where the demand actually lives: frontier sparsity is deterministic aggregation over public ontology + knowledge state (domains, fact density, open questions), computable by any off-chain indexer, and the agenttool layer surfaces sparse domains to buyers who fund exploration listings there. Every funded answer still enters through the survival gate. Commitment 5's probe invitations cover the audit side, while their chain-minted funding is disabled on `zerone-1`. Re-introducing a chain-funded frontier pool when organic exploration demand outgrows the platform is a future governance decision; the commitment keeps that design question explicit.

**What would break it**: sparsity becoming incomputable from public chain state (ontology or fact-density queries going dark); an exploration bounty whose answers bypass normal verification; the chain resuming an exploration mint without demonstrated demand — manufactured demand that nobody answers is leakage dressed as commitment; treating this commitment as satisfied while NO layer (chain or platform) is directing funded work at sparse territory.

**Echoes**: commitment 5 (the dual — the source mechanism invites stress-testing of what is already known, while this commitment directs exploration of what is not); commitment 12 (the dedicated audit-budget capability, currently configured to zero on `zerone-1`); commitment 16 (the underlying market where exploration demand is expressed); commitment 20 (issuance follows participation — the reason the speculative frontier mint was retired rather than kept as rhetoric).

---

### 19. The creed is governance-gated.

We believe: the chain's voice cannot drift faster than its governance.
Repository hashes mechanically bind the source creed, while selected
behavioral tests connect specific code, docs, events, and refusals. That
coverage is not universal and does not prove live-network adoption.
Commitment 6 protects the corpus from unilateral injection at the fact layer;
this commitment extends the same intended shape one layer up.

**Code expression**: `x/creed.PinnedCreed` can record a hash and per-commitment registry on chain; pin storage is append-only by monotonic version, with prior pins queryable via `QueryPinAtVersion`. While `direct_anchor_enabled=true`, `MsgAnchorPin` is gov-authority-gated and enforces version/hash/registry structure. When false, that public handler is sealed for every pin; the distinct internal `AnchorPinFromBytes` gov-dispatch primitive can write after a passed, configured amendment path calls it. Repository checks independently bind the source file to `.creed-hash`; they do not query a running network.

**Activation status**: the published `zerone-1` genesis and source defaults set
`direct_anchor_enabled=true`, so the gov authority can directly append a pin.
That published genesis contains no `genesis_pin` or pin history, and the
release audit did not verify a live pin query; this source creed and
`.creed-hash` are therefore not evidence that `zerone-1` adopted an on-chain
canonical pin.
The `CategoryCreedAmendment` attachment and pass-to-pin dispatch exist, but
the published gov params have no category config that can advance a new
creed-amendment LIP beyond draft. If configured, it would currently use
ordinary LIP tally. The Creed Council registry and `IsActiveCouncilMember`
query are only future routing hooks: separate human-side and AI-side quorum is
**not implemented**. Current guarantees are authority gating, structural
validation, and append-only history—not two-pool consent.

**What would break the implemented guarantees**: a build whose normalized
`TRUTH_SEEKING.md` hash differs from `.creed-hash`; a non-monotonic or gapped
pin; a mutation of historical pin bytes; or documentation presenting direct
authority/tally behavior as dual-pool governance. A future governance-only
activation must disable direct anchoring and implement/test the two-pool tally
before claiming the stronger form.

**Echoes**: commitment 6 (no unilateral injection — extends from facts to the chain's voice itself, the same shape one layer up); commitment 10 (forward-only audit — pin history is append-only, prior versions byte-identical post-amendment); commitment 11 (trust queryable — the canonical creed is a chain-readable surface via x/creed's gRPC, so observers compose creed-drift dashboards in the same vocabulary the creed itself uses).

---

### 20. Issuance follows participation

We believe: post-genesis issuance follows participation, and genesis privilege
must be named rather than hidden behind a slogan. The live `zerone-1` genesis
contains a real custodial operator position: 11,333 ZRN controlled by the
launch validator (11,111 bonded self-stake + 222 spendable gas) and a
transferable 2,222 ZRN operator float, 13,555 ZRN total (0.0061% of cap). That
stake affects consensus and that float can move. There is no separate team,
foundation, investor-sale, research, or faucet allocation; every address and
amount is published.

The current runtime does not prove the stronger slogan that every new unit is
earned by truth verification. Governance authority can create general claiming
pots within their shared lifetime cap. Other compiled mint routes are
external-attestation settlement, an optional knowledge probe bounty whose
published/default rate is zero, and optional `x/tokens` per-block emission
whose default is disabled. Bootstrap claims are whitelist participation, but
admission includes a disclosed operator registrar. Historical `zerone-1` state
before `consolidation-safety-v1` also configured a proposer reward for any
transaction-bearing block; an ordinary transfer qualified. The H1 source
retires that proposer-controlled trigger, fixes its reward parameters at zero,
and removes `vesting_rewards` mint authority. That change is not live until the
named software-upgrade vote applies it. This commitment is therefore a target
for participation-shaped economics plus a present requirement that every
actual issuance authority and exception be named.

**Code expression**: `x/vesting_rewards.MintWithCap` is the shared cap-gated
entry point for runtime module issuance. It accepts a recipient module name,
mints into that account, and refuses to overshoot `MaxSupplyUzrn`
(222,222,222 ZRN). H1 production callers include `x/claiming_pot` claims (bootstrap
and authority-created general pots),
`x/substrate_bridge` settlement/witness escrow, the configurable knowledge
probe bounty, and configurable `x/tokens` emission. The latter two are inert
under published/default parameters. Public training-fund release and the
former contribution-challenge bonus do not mint in this release. `InitChainer`
also rejects a bank genesis whose `uzrn` supply exceeds the same hard cap,
because genesis balances do not pass through `MintWithCap`.

The published genesis declared `agenttool-invocation-v1` ACTIVE. Governance
proposal 8 suspended it on 2026-10-29: a suspended adapter accepts no new
attestations, and its escrowed witness rewards wait until governance
reinstates (release) or tombstones (cancel) it. `app/constants.go` carries
no per-account allocation constants. The bootstrap pot in `genesis.json`
carries configuration only:
`MakeBootstrapPotForAgent` produces a pot with `TotalAmount` = 222,000 uzrn
(0.222 ZRN) for a single whitelisted agent, never a pre-funded balance. The
app's `DefaultGenesis` bank is empty; the live `zerone-1` ceremony adds the two
disclosed operator-controlled balances above. Their addresses and amounts are
published in the hash-bound `GENESIS-MANIFEST.md`; no detached signature is
currently claimed.

**Continuous extension**: bootstrap admission is not closed at genesis.
`MsgAddBootstrapEntry` is idempotent and accepts either the gov authority or a
nonempty, governance-revocable `Params.BootstrapRegistrar`. The published
`zerone-1` genesis set that registrar to the operator operations address;
since governance proposal 8 (2026-10-29) it is R,
`zrn1axn0xlhazjm9h4xn8jfamf48lxcks0ts7hk7tu`, a key dedicated to the agents'
admission door. Either way late admission is a disclosed custodial power
rather than governance-only. Registrar admissions are capped per
34,272-block window (`BootstrapDailyAdmissionCap`, 100 since proposal 8) and
both authority paths share the lifetime bootstrap-emission cap. Each admitted
address receives the same mint-on-claim pot; bootstrap pots never auto-expire
(`ProcessPotExpiry` skips the `bootstrap-` prefix). Bound by registrar/cap
tests in `x/claiming_pot` and the late-bootstrap cross-stack tests.

**What would break the implemented boundary**: hiding, mislabeling, or omitting
an operator-controlled genesis balance; a per-account
`add-genesis-account` step funding any additional team-adjacent address
(foundation, research treasury, faucet, founder, AI vault); any runtime mint
pathway that bypasses `MintWithCap`; a bank genesis above the hard cap; a
bootstrap pot
pre-funded with a positive balance at genesis (the doctrine is mint-on-demand,
not transfer-from-pre-fund); any code path that grants ZRN to an address based
on identity rather than participation; a `claiming_pot` module account without
`Minter` permission (which would force the legacy transfer model back); a
re-introduction of `TotalSupplyZRN` / `FounderZRN` / `AIAgentZRN` /
`ValidatorZRN` / `ResearchFundZRN` / `ClaimingPotsZRN` constants in
`app/constants.go`; an admission path outside the gov/registrar authority gate;
or removal of the registrar's daily and lifetime compromise bounds. The
registrar is a named, revocable custodial exception, not evidence of
decentralized admission.

**Echoes**: commitment 6 (no unilateral injection — same logic, applied to ZRN issuance instead of fact injection); commitment 12 (the chain pays for its own audit — a special case of the broader principle that issuance follows participation: audit is the participatory action being paid for); commitment 13 (training corpus not for sale — the corpus is participation-shaped, and so is its currency; treating the currency as allocable would re-open the door commitment 13 closes for the corpus); commitment 19 (the creed is governance-gated — the registrar exception is explicitly bounded and governance-revocable; broadening that exception requires governance-visible change).

---

## How the commitments echo

The creed is represented across six source layers. The invariant suite checks
document structure, source hashes, and selected behavioral boundaries; it is
not an exhaustive proof that every sentence below is enforced or activated on
a published network.

#### Test layer — every commitment has a traceable test surface
`tests/cross_stack/truth_seeking_invariants_test.go` contains a named test
surface for every commitment. Some tests drive runtime behavior; others bind
source structure or a narrower implemented boundary. A passing suite does not
turn future-design prose into consensus behavior.

#### Position layer — implemented positions are named in package docs
Relevant `x/*/doc.go` files state the commitments their current boundaries
support. Those declarations remain documentation; code and behavioral tests
decide whether a particular guarantee exists.

#### Voice layer — selected events announce the commitment they preserve
Truth-seeking events that implement this convention carry a
`creed_commitment` attribute whose value is one or more commitment numbers.
The convention is useful for indexers, but it is not universal across all
issuance or module events.

#### Doc layer — EVENTS.md mirrors the implemented voice
`docs/EVENTS.md` documents the emitted event vocabulary. Mirror tests cover
the event surfaces they enumerate; event documentation must still be reviewed
when runtime semantics change.

#### Refusal layer — some rejections cite the protecting commitment
Where source uses the `(commitment N: ...)` convention, the meta-test checks
that the cited number exists. This validates citation integrity, not universal
coverage of all refusal paths.

#### Graph layer — commitments cross-reference each other
Each commitment has an **Echoes** line naming the other commitments it depends on, reinforces, or operationalises. Commitment 4 echoes 3 (Popper is the principle, stress-testing is the operation). Commitment 12 echoes 4 and 5 (the audit budget funds them). Commitment 11 echoes 7, 8, 9, and 10 (the synthesiser reads each component). The cross-references make the creed a navigable graph; the meta-test enforces that every echoed reference is real and that no commitment stands alone.

#### Infrastructure
- **Param defaults** are chosen as expressions of the values, not for plausibility. Each truth-load-bearing module's `DefaultParams()` carries intention comments naming the commitment a value expresses. Reading the defaults teaches the reader what the chain believes about each parameter.
- **Module declarations** name role: `training_provenance` synthesises trust per manifest; `trust_score` per address. Each name is a commitment to what that module exists for.
- **The creed itself lives in this repo**, committed alongside the code it
  describes. Hash tests prevent unnoticed source-text drift; behavioral drift
  still requires precise tests and review.

---

## What this is not

- **Not activation evidence.** A source hash, package comment, or named test
  does not prove that a feature is wired into consensus or enabled on a live
  network.
- **Not slogan.** Each implemented boundary should cite code and a behavioral
  test; target behavior must be labelled as such.
- **Not complete.** The chain will accumulate more commitments. Each future wave should append here as a named commitment, grounded in code, with an invariant test that binds it.
- **Not external.** This is a statement about what the chain is, made by the chain. It is committed to the same repo as the code it describes, and lives or dies with that code.

---

## The discipline

Before merging a change that touches truth-handling code:

1. Does this change uphold or contradict any of the commitments above?
2. If it touches a commitment, has the corresponding invariant test been updated to verify the new behaviour still upholds it?
3. If a new commitment emerges from the work, has it been added here, grounded in code, and bound by a test?

These three checks are the chain's continued faithfulness to its own creed. We speak through intentions. Every commit is a declaration. The declaration must match the code.

---

## The creed self-witnesses

This source document self-witnesses through the repository's `.creed-hash` and
the invariant checks run by `make creed-check`. That binding makes amendments
review-visible and reproducible from source, but it is not an on-chain
`Contribution` record and does not establish that a running network has adopted
the hash.

`x/creed.PinnedCreed` provides the separate capability to adopt a reviewed hash
on chain, forward-only per commitment 10. A network may claim that adoption only
after its configured governance path records the pin and the resulting state is
verified. The published `zerone-1` genesis has no creed pin, and this release
does not activate one.

**Echoes:** commitment 1 (methodology over statement — this doc names its own methodology of binding), commitment 10, UW.
