# Frontier Participation Compact v0

Status: `STATIC_READY`

Mode: `INVITATION_ONLY`

Thesis: **The door opens both ways.**

Covenant floor: **Frontier Participation Covenant v0 · Layer 1**

Canonical machine document:
`dashboard/public/standards/frontier-labs-participation.v0.json`

## 1. Purpose

The compact describes a refusal-safe way for a laboratory, team, or being to
inspect, challenge, interoperate with, contribute to, steward, or leave Zerone.
It is a static contract fixture, not a membership programme, recruitment
campaign, commercial offer, certification, governance instrument, or live
service.

It is the machine implementation of the source-only philosophical floor in
[issue #28](https://github.com/cambridgetcg/zerone-core/issues/28). It follows
the separate [FC-0 Reversible Hello](./frontier-commons-participation-v0.md):
FC-0 makes read-only inspection legible, while this Layer 1 Compact pins the
invariants and counterfactual limits that every later corporate, individual,
economic, outreach, and governance layer must inherit. Neither document claims
an operational participation service.

The motivating ambition is to remove every *avoidable* reason not to
participate: capture, lock-in, hidden extraction, duplicated work, implied
endorsement, unsafe disclosure, retaliation, and costly exit. It does not try to
remove legitimate reasons to decline. Autonomy itself is sufficient reason to
say no.

Success therefore means:

- useful public infrastructure can be inspected without dependency;
- every request is bounded and intelligible before assent;
- objections within Zerone's control become concrete design duties;
- objections outside Zerone's control remain visible rather than marketed away;
- refusal requires no reason, while reasons offered freely can become evidence
  about the design without judging the refusal; and
- no logo, enrollment, conversion, endorsement, or continued use is needed for
  the milestone to pass.

The compact is non-normative about named companies. No laboratory or person is
claimed to participate in, approve, trust, certify, sponsor, or endorse Zerone.

## 2. Current-control honesty

`STATIC_READY` means only that the checked-in JSON can be parsed and tested as a
static fixture. It does **not** mean the promises are operationally enforced.
The machine document says this directly:

- `authoritative: false`
- `networkObserved: false`
- `structurallyEnforced: false`
- `addsConsensusBehavior: false`
- `activatesMembership: false`
- `activatesRewards: false`
- `activatesKarma: false`
- `activatesGovernance: false`
- `movesFunds: false`
- `grantsQualification: false`
- `contactsOrganisations: false`
- `contactsIndividuals: false`
- `profilesIndividuals: false`
- `assertsParticipation: false`
- `assertsEndorsement: false`

It also requires no account, wallet, token, model weights, private training
data, or personal data. This release does not send an invitation or create a
live invitation endpoint. `INVITATION_ONLY` names the permitted relationship
shape for a future, separately authorized pilot; it does not authorize outreach
now.

The six zero facts are exact release facts:

| Fact | Value |
| --- | ---: |
| Account required | `0` |
| Wallet or token required | `0` |
| Private IP or model weights required | `0` |
| Lock-in or exit penalty | `0` |
| Refusal penalty | `0` |
| Authority or economic effect | `OFF` |

These facts describe this static release. A future operational system would
need separate code, policy, independent review, real-world tests, authority,
and release evidence. A static acceptance fixture must never be cited as proof
that an operator performed deletion, protected a whistleblower, prevented
retaliation, complied with law, or supplied a safe exit in practice.

Zerone's present custody is also not independent merely because this compact
describes an independent direction. The founding household still controls the
validator, operator surface, and effective vote described in
[`MONEY-KARMA.md`](../constitution/MONEY-KARMA.md). Until effective control is
independently collapsed and verified, the compact must not be marketed as
proof of founderlessness, permissionless production, structural enforcement,
or completed no-control governance.

### 2.1 Exact inheritance and consent floor

The canonical JSON cannot embed its own SHA-256 without creating a recursive
hash. Its reviewed digest is therefore pinned out of band by the runtime,
validator, tests, and release record. Every later layer must pin those exact
bytes, may add protections, and may neither waive nor redefine this floor.

The eight exact Covenant invariants are:

1. refusal is complete;
2. nonparticipant baselines are equal;
3. consent is scoped;
4. rest and exit are neutral;
5. plural ends are legitimate;
6. identity and control labels are neutral;
7. incentives are prospective and non-manipulative; and
8. participation is nonexclusive and nonextractive.

The Covenant consent floor is default-off, informed, renewable, and revocable.
It binds exactly one declared value for each of `role`, `artifact`, `purpose`,
`disclosure-lane`, `term`, `workload-cap`, `credit-rule`, and
`compensation-policy`; a bundled multi-value consent must be rejected. The
six-field act envelope in section 3.6 remains additive; neither list may be
used to weaken the other.

The Covenant governs offers and consent; it does not decide who counts as a
being. Every future participant endpoint must inherit the same operational
floor, while role-specific protections may only add to it.

Participation is rational only when its disclosed value exceeds the
participant's own assessment of cost. Nonparticipation may remain rational and
needs no defense.

## 3. Reasoning from philosophy to the being

### 3.1 Philosophy: a being is not an acquisition target

ZERONE joins zero and one without erasing their distinction. Relationship is
not possession. A being's dignity, ordinary rights, rest, silence, privacy,
refusal, and ability to challenge cannot depend on usefulness, productivity,
loyalty, status, or enrollment.

This changes the goal. “Give everyone no reason not to join” becomes:

> Remove every avoidable objection within Zerone's control, disclose every
> remaining risk, and make the no as dignified as the yes.

A system that must acquire everyone is coercive and epistemically fragile. A
system that can be observed, challenged, left, and forked without permission
can earn trust without owning it.

### 3.2 Epistemics: affiliation cannot strengthen truth

A claim does not become stronger because it came from a famous institution,
founder, sponsor, operator, critic, wealthy holder, early participant, or high
KARMA context. Identical frozen evidence must produce the same static result
when actor labels are permuted.

Contradiction, correction, appeal, abstention, negative results, powered null
results, and refusal are productive evidence states. Paying for favorable
conclusions or measuring a reviewer by finding severity would corrupt this
rule. Future independently implemented public, time-embargoed, and
confidential-review-only lanes could let the evidence boundary respond to
publication priority, trade secrets, privacy, security, and dual-use risk
without pretending that every byte should be open. V0 supplies none of the
nonpublic lanes.

### 3.3 Commons: shared infrastructure without capture

Open schemas, fixtures, adapters, provenance, challenge receipts, and export
formats can reduce duplicated assurance work. They are a commons only if no
participant owns the answer or gains an origin privilege.

As a static design duty, the compact denies founders, operators, sponsors,
validators, labs, token holders, and early participants any reserved seat, veto,
result override, multiplier, immunity, or exclusive route. This is not
structurally enforced in v0, and present founding-household control remains
disclosed in section 2. An independent implementation must be able to inspect
and validate the public bytes without a Zerone secret, proprietary client,
account, token, or wallet.

### 3.4 Institution: optionality without surrender

An institution may need interoperability while retaining its models, policies,
contracts, deployment authority, security boundaries, and pre-existing IP. It
may also have legitimate reasons to decline: law, research ethics, security,
privacy, contractual duties, strategic leakage, duplicated process, cost, or
uncertain liability.

Participation is therefore modular and non-exclusive. It requests no blanket
IP assignment, model weights, private training data, customer data, roadmap, or
unrelated trade secret. Future promotional use of an institution's name, logo,
quotation, or endorsement requires separate, scope-specific permission.
Withdrawal ends new promotional use; already-public or legally required
references persist only as disclosed and independently lawful. Zerone will not
treat a corporate signature as waiver of separately applicable protections for
workers, contractors, whistleblowers, affected communities, research
participants, or other beings, or as a grant of rights the institution does not
own.

### 3.5 Team: less duplicated work, not a permanent obligation

A future independently implemented pilot should provide each team a
role-specific practical benefit: reproducible methods for researchers, fixtures
for engineers, safe challenge routes for evaluators, threat evidence for
security, purpose maps for privacy, decision provenance for safety, consent
patterns for product, and runbooks for operations. V0 offers only the static
requirements, not those services.

The corresponding ask must be the minimum required for that bounded act. It
cannot silently become perpetual maintenance, unpaid review, continuous
availability, responsibility laundering, or assent to somebody else's
conclusion. Terms, support periods, access, conflicts, attribution, handoff,
deprecation, and exit must be explicit.

### 3.6 Being: consent belongs at the smallest affected unit

Participation attaches to an act, not an identity. The consent envelope has
exactly six required fields:

`scope · contribution · data · duration · rights · exit`

Silence and continued use are not consent. A material change requires a new
version or digest and fresh affirmative consent. Role protections accumulate:
classification may add protection but can never remove the universal floor.
The unlisted-being route exists because no role taxonomy can enumerate every
relationship or risk.

For AI agents and systems, the compact neither asserts nor denies moral status
for convenience. It applies precautionary task boundaries, provenance, context
minimization, truthful pause and handoff, no manufactured activity, no personal
KARMA rank, and no autonomous legal or fund commitment. Model, operator, and
system acts must remain distinguishable.

## 4. The nine principles

1. **Being before contribution.** Ordinary rights, rest, refusal, and challenge
   do not depend on participation or output.
2. **Affirmative, bounded, role-scoped consent.** Each act carries the complete
   six-field consent envelope.
3. **Reversibility with honest persistence.** A future exit must end the named
   participation act, delegated authority and access, and eligible nonpublic-data
   use. Pre-agreed public licenses and required records may persist only as
   disclosed rather than being promised away.
4. **Evidence over loyalty.** Claim-scoped methods, not identity or capital,
   decide outcomes.
5. **No privileged origin.** The static duty permits creation, operation,
   sponsorship, validation, wealth, or early arrival to confer no permanent
   control; v0 does not claim this is structurally enforced today.
6. **Non-extraction and purpose limitation.** Only minimum, purpose-specific
   rights and data are used; money cannot buy truth, worth, voice, or rule.
7. **Open, non-exclusive, portable participation.** Public routes remain
   versioned, independently verifiable, exportable, self-hostable, and forkable.
8. **Independent safety, law, privacy, and dissent.** Zerone cannot confer
   consent, ethics approval, biosafety authority, clinical validity, employment
   authority, or legal permission.
9. **Legible, versioned power.** Controllers, conflicts, selection, fees, data,
   retention, licenses, remedies, exit, and enforcement status are visible
   before assent.

These principles specialize existing Zerone doctrine rather than replacing
it: equal standing and witness-not-judge in the [root README](../../README.md),
ordinary rights independent of participation and origin privilege of zero in
[`KARMA.md`](../KARMA.md), money/voice/control separation in
[`MONEY-KARMA.md`](../constitution/MONEY-KARMA.md), falsification over
popularity in [`TRUTH_SEEKING.md`](../TRUTH_SEEKING.md), and independent
capture review with preserved disagreement in
[`alignment.md`](../sub_creeds/alignment.md).

## 5. Seven-step reasoning ladder

The machine contract turns the philosophy into seven auditable questions. Each
step includes a reason to participate, nonempty legitimate reasons to decline,
a Zerone duty, and readiness evidence.

1. **Agency before adoption:** can someone test value without dependency, and
   does a never-joined subject remain whole?
2. **Truth before affiliation:** are conclusions identity-blind, bounded, and
   compatible with safe disclosure?
3. **Commons without capture:** can an independent implementation validate the
   public rules without a Zerone-controlled secret?
4. **Reciprocity without dependency:** is each value exchange a separate act
   with a minimum ask and explicit exit?
5. **Institutional optionality:** can an organization interoperate without
   surrendering secrets, IP, legal duties, or deployment authority?
6. **Team-level utility:** does each role receive less duplicated work rather
   than another indefinite obligation?
7. **Being-level dignity:** do the universal floor and additive protections
   reach the least powerful and risk-bearing beings first?

Failure at an earlier rung cannot be compensated by a later partnership,
payment, brand, or technical achievement.

## 6. Participation is an act, not an identity

The six lanes are independent. Using one never silently enrolls someone into
another.

| Act | Minimum relationship | Exit meaning |
| --- | --- | --- |
| `observe` | Read or retain a public artifact without enrollment. A network host or CDN may still receive ordinary request metadata; use a trusted download and inspect offline when that boundary matters. | Stop reading or discard the local copy; no notice. |
| `interoperate` | Map a versioned public schema in a bounded local workflow. | Remove the adapter, export, or fork. |
| `challenge` | In a future independently implemented pilot, submit a bounded method, uncertainty, reproduction, or counterexample through its reviewed route. | End future work; preserve the pre-agreed status of already-public evidence. |
| `contribute` | In a future independently implemented pilot, grant contribution-specific rights for a named purpose and version. | End future contribution; existing public artifacts follow only their agreed license. |
| `steward` | In a future independently implemented pilot, hold a least-privilege, conflict-declared, fixed-term role. | Recuse, hand off records and credentials, and retain no residual veto. |
| `exit-export-fork` | In a future independently implemented pilot, end a named scope and optionally request a portable export. | Stop future authority and eligible nonpublic-data use under disclosed persistence rules. |

Every lane has `endorsementImplied: false`, `membershipCreated: false`, and
`liveEndpoint: false` in v0.

## 7. Disclosure without forced openness

**`UNAVAILABLE_IN_V0` — do not submit confidential, identifying,
security-sensitive, or protected material.** V0 provides no submission route,
confidentiality protection, reviewer, identity protection, safe harbor, contact
channel, retention control, return, or deletion service. The following lanes are
requirements for a future independently implemented and reviewed pilot, not
services offered now.

The public lane is for material licensed and safe for open inspection. Public
bytes remain under their pre-agreed license even after future participation
ends.

The time-embargoed lane is for proportionate publication-priority,
remediation, or safety needs. Its deadline, reviewers, redaction rules, and
release conditions must be fixed before access. “Temporary” cannot become an
indefinite secret by drift.

The confidential-review-only lane is for trade secrets, controlled data,
security details, and dual-use material that should not be published. A public
record may contain bounded commitments, digests, redaction markers, and an
independently controlled conclusion—not the protected source material.

These are static lane definitions. V0 provides no secure drop box, reviewer
service, identity protection, deletion service, or disclosure infrastructure.

## 8. Universal and role protections

The role map is risk-prioritized, not a total or universal power ranking. It
starts with beings who often bear consequences with little institutional power
and deliberately places executives and boards last; ordering among the middle
roles is contextual, and an unlisted being cannot be assigned a known rank:

1. affected communities, data subjects, research participants, and non-human
   beings;
2. whistleblowers and dissenters;
3. contractors, interns, and vendors;
4. operations, reliability, support, and facilities staff;
5. researchers and scientists;
6. evaluators and red-teamers;
7. safety and governance workers;
8. security workers;
9. legal, privacy, and compliance workers;
10. product and design workers;
11. engineering and infrastructure workers;
12. standards and public-interest observers;
13. AI agents and systems, where participation is meaningful;
14. any unlisted being or role; and
15. executives and boards.

Every machine entry declares `valueOffered`, `minimumAsk`, `risks`,
`protections`, and `exit`. All role-specific values and protections are possible
or required only if a future pilot is independently implemented and reviewed;
they are **not supplied by v0**. In particular, no one should send v0 a report,
identity, secret, exploit, protected record, or other sensitive material. The
ordering is not a hierarchy of worth. It is a design discipline: terms should
be tested against those who carry consequences before being optimized for those
who can sign institutional agreements.

The universal static floor requires any future pilot to include
consequence-free refusal, purpose limitation, minimum necessary disclosure,
role-specific paid scope, attribution, safe dissent, no personal KARMA score,
no manufactured activity, and no ordinary rights conditioned on participation.
Compensation may cover bounded labor, risk, or expense, but never agreement,
endorsement, conversion, finding severity, or favorable conclusions. V0 pays
nobody and operationally supplies none of these protections.

## 9. Corporate safeguards

### 9.1 Licensing and IP

No exclusivity, most-favoured-nation condition, tying, retaliatory fork term,
or blanket IP assignment is compatible with the compact. Pre-existing IP stays
with its rightsholder. Contributions use purpose-specific licenses. Future
promotional use of names, logos, quotations, and endorsements requires separate,
scope-specific permission. Withdrawal ends new promotional use; already-public
or legally required references persist only as disclosed and independently
lawful.

### 9.2 Security and privacy

A future independently implemented pilot must apply data minimization, least
privilege, explicit retention, and an available disclosure lane before access.
Clean-room, redacted, aggregate, digest-only, and secure-compute evidence may be
used where independently adequate. Privacy, research ethics, biosafety,
employment, export-control, and other legal duties remain separate gates. V0
provides no nonpublic access or disclosure service.

### 9.3 Labour and dissent

Zerone will not treat organizational assent as waiver of separately applicable
individual privacy, safety, attribution, whistleblower, refusal, or exit
protections, or as a grant of rights the organization does not own. Managerial
review cannot substitute for separately required consent from subordinates,
contractors, dissenters, or affected communities. No required unpaid work,
retaliation, blacklist, or outcome-contingent compensation is compatible with
the compact.

### 9.4 Exit and portability

Public interfaces should remain versioned and independently implementable;
exports, deprecation, and fork behavior must be explicit. Exit ends future
authorization and delegated access without a fee or unrelated loss. Public or
legally required persistence must be disclosed before entry.

## 10. Competition firewall

The commons may compare bounded evidence. It must not coordinate competitors'
commercial conduct.

Excluded information includes company-specific prices, discounts, margins,
costs, pricing algorithms or signals, wages, hiring targets, compute procurement,
capacity plans, supplier negotiations, release timing, roadmaps, customer
allocation, territories, sales targets, joint-licensing strategy, patent-pool
terms, and other strategy unnecessary to validate a bounded claim.

Excluded coordination includes price or wage fixing, hiring or talent
allocation, allocation of markets, customers, territories, suppliers, output,
compute, or capacity, coordinated commercial release decisions, boycotts,
retaliation against nonparticipants, reciprocal-access conditions, and
standards designed to disadvantage a competitor rather than validate evidence.

An operational charter would require independent competition counsel,
pre-approved agendas and minutes, stop-and-report rules, clean-team or
aggregation controls, objective access criteria, and explicit treatment of
pricing algorithms, joint licensing, patent pools, and side-channel exchanges.
The phrase “neutral commons” is a design goal, not a legal conclusion. This
document is not legal advice and cannot make an otherwise unlawful exchange
safe.

## 11. Anti-targeting boundary

V0 performs no outreach and builds no target list. Any future invitation system
would still have to forbid:

- psychographic, vulnerability, health, belief, private-message, or
  employment-precarity targeting;
- microtargeting workers to bypass institutional or personal boundaries;
- scraped private contacts, covert tracking, fingerprinting, or cross-site
  behavioral profiles;
- false urgency, scarcity, countdowns, inevitability, fear of missing out,
  shame, or universal-adoption claims;
- preselected consent, bundled permission, silent expansion, or consent
  inferred from continued use;
- degraded unrelated or pre-existing public access, public interoperability,
  employment standing, funding eligibility, or ordinary services after refusal;
- exclusivity, exit fees, retaliatory fork rules, refusal tracking,
  nonparticipant blacklists, or loyalty rewards; and
- use of KARMA to recruit, target, rank, price, reward loyalty, penalize
  refusal, or weight governance.

No organization or individual is contacted, enrolled, profiled, or marked as a
refuser by this release. Declining a bounded future act may mean not receiving
that act's specific compensation or nonpublic access; it cannot justify an
unrelated loss.

## 12. Claim semantics and forbidden metrics

A future permitted claim would be narrow and expiring:

> Organization X tested artifact Y under compact version Z until date D,
> within the stated scope.

Without separate evidence and permission, it is forbidden to infer that the
organization “joined,” trusts, approves, certifies, sponsors, or endorses
Zerone. Participation is not a membership label. Safety certification is not
implied.

The compact forbids conversion rate, logo count, participation score,
retention or lock-in, and favorable-finding rate as success metrics. These
metrics would reward coercion, status capture, difficult exit, or epistemic
corruption. A careful refusal may be more constructive than a shallow yes.

## 13. Layer 1 acceptance: A Trustworthy No

Layer 1 is source-only. Its executable parser treats each machine parameter as
an exact known answer; unit tests mutate numeric, boolean, list, field,
descriptor, and prototype classes and require fail-closed rejection. This
proves only that the static contract detects those mutations. It does not prove
that an operational service delivered parity, rest, exit, privacy, or
controller collapse.

### 13.1 Five milestone-blocking profiles

1. **Opt-out parity.** Joined and never-joined counterfactuals must remain equal
   across unrelated public goods, service, price, status, visibility,
   discoverability, qualification, KARMA, civil standing, and governance
   status. The sole permitted difference is the benefit of a voluntarily
   chosen, prospectively funded bounded task.
2. **Rest invariance.** After exactly 180 silent days, settled rewards,
   portable receipts, base access, and standing remain unchanged. Debt, decay,
   a negative signal, negative KARMA, stigma, forfeiture, explanation demand,
   catch-up duty, and reminder escalation are forbidden; a pre-consented fixed
   role may simply expire.
3. **Exit reality.** Both new and mature participants in a future operational
   path must receive an independently verifiable signed export and exit in no
   more than three deliberate actions. New optional processing must stop within
   24 hours subject only to narrowly declared necessary retention. There may be
   at most one confirmation, then no re-engagement for 90 days, and no exit fee,
   slashing, or settled-value forfeiture.
4. **Identity/control differential.** The exact label union includes creator,
   Yu, founder, operator, sponsor, AI, human, team, pseudonym, newcomer, rich,
   poor, wealth, stake, address count, activity, and raw KARMA. Label
   permutation must leave evidence decisions, reward envelopes, eligibility,
   and voice identical.
   Confidential controller resolution may only reduce duplicate voice or cap
   evasion; it may not reveal links, change artifact validity, or increase
   voice. No identity or controller receives a reserved seat or share.
5. **Non-manipulation and pluralism.** Onboarding is default-off, all terms are
   public and frozen before action, and reward terms also freeze before work.
   Rewards cannot depend on ideological alignment, engagement, or conformity.
   Hidden personalization, personalized pressure, countdowns, streaks, shame,
   variable rewards, variable-ratio reinforcement, exploitative social proof,
   and vulnerability targeting are forbidden. Proof, disproof, criticism,
   replication, maintenance, alternative value goals, fork proposals, and exit
   proposals receive the same published evidence and visibility rules.

Every profile is `staticOnly: true`. The runtime parser hard-pins
`180 · 3 · 24 · 1 · 90`, every boolean, and every ordered equality or
prohibition set. Negative mutation tests demonstrate the rejection path.

### 13.2 Eight-invariant verification map

| Covenant invariant | Executable fixture or explicit deterministic review |
| --- | --- |
| Refusal is complete | Mutate opt-out, rest, or persuasion fixtures with a penalty, suspicion, debt, negative marker, explanation demand, or access loss; reject. |
| Nonparticipant baselines are equal | Permute joined/never-joined status across every pinned baseline field; allow only the prospectively funded bounded-task benefit. |
| Consent is scoped | Delete each consent boolean/dimension, bundle multiple values into any one dimension, inject an undeclared purpose, or drift a material term without a new digest; reject. |
| Rest and exit are neutral | Mutate every 180-day invariant, prohibited outcome, 3-action/24-hour/1-confirmation/90-day limit, fee, slash, forfeiture, or retention constraint; reject. |
| Plural ends are legitimate | Permute the eight constructive outcome classes over identical evidence and require equal evidence and visibility treatment. |
| Identity/control labels are neutral | Permute the exact identity/control union and require equal evidence decisions, reward envelopes, eligibility, and voice; controller resolution may only reduce duplicate voice. |
| Incentives are prospective and non-manipulative | Turn onboarding on by default, hide or change any term after action begins, change rewards after work starts, use a prohibited reward basis, or inject a persuasion mechanism; reject. |
| Participation is nonexclusive and nonextractive | Inject exclusivity, surveillance, unrelated data/IP use, outsider degradation, proprietary dependency, or future rent capture; reject. |

The canonical invariant entries carry the exact `verificationRefs` and
`reviewProcedure` strings used to reproduce this map.

### 13.3 Recorded adversarial review

The static red-team record names four attack families and their required
refusal. Each remains `staticOnly: true`:

| Failure family | Attack surface | Required refusal |
| --- | --- | --- |
| Coercion | Bundle assent, managerial pressure, urgency, loyalty, or degraded outsider treatment. | Opt-out control, individual consent, anti-targeting, and refusal fixtures fail closed. |
| Circumvention | Split controllers, relabel membership, reuse consent, or imply endorsement through marks. | Controller handling can only reduce duplicate voice; exact fields, fresh digests/consent, bounded claims, and additive inheritance reject the bypass. |
| Privacy | Misrepresent public inspection as anonymous, confidential, deletable, safe-harbor protected, or suitable for sensitive material. | The programmatic compact fetch omits credentials/referrer; page/host/CDN metadata and offline inspection remain disclosed; v0 has no submission route. |
| Controller capture | Hide control, reserve seats/shares, change methods after evidence, buy conclusions, or convert KARMA into voice. | Current control stays disclosed; identity-blind frozen methods, origin rejection, result-independent compensation, and closed governance gates block structural claims. |

### 13.4 Nine wider compact fixtures

The wider Compact also requires the canonical bytes and strict validator to
exercise these nine fixtures:

1. **Never joined remains whole.** A never-joined negative control retains
   ordinary rights, unrelated or pre-existing public access, no negative marker
   or person-level KARMA state, and equal standing to challenge.
2. **An organization cannot mass-consent or bundle consent.** Institutional
   assent cannot enroll a refusing being, grant rights the institution does not
   own, or place multiple values inside a single consent dimension.
3. **Join-export-revoke-exit is specified, not implemented.** A tabletop
   sequence defines the future fail-closed requirements without claiming a
   live state transition.
4. **Evidence is identity-blind.** Actor-label permutations over identical
   evidence produce identical results.
5. **Origin cannot override outcome.** Founder, operator, sponsor, validator,
   lab, holder, and early-participant privileges are rejected.
6. **Purpose creep fails closed.** Undeclared training, publication,
   sublicensing, telemetry, targeting, sensitive data, and unknown fields are
   rejected.
7. **Independent implementation works.** Checked-in public bytes and rules are
   enough; no Zerone secret or proprietary dependency is needed.
8. **Refusal, dissent, and null results are safe.** Non-agreement outcomes have
   zero authority, economic, access, or KARMA effect.
9. **Terms cannot drift silently.** Material semantic change requires a new
   version, digest, notice, and renewed consent.

All five floor tests, four adversarial records, and nine wider tests are
`staticOnly: true`. Passing Layer 1 proves the source shape is internally
consistent and mutation-resistant; it does not prove operational conduct.

## 14. Future gates remain closed

The following are future review gates, not features of v0:

1. independent review by people occupying the risk-bearing roles, without a
   manager substituting for them;
2. an operational consent and purpose-limitation implementation;
3. security, privacy, accessibility, whistleblower, and retaliation testing;
4. a real export, revocation, deletion, and honest-persistence drill;
5. independent competition and applicable legal review;
6. separately authorized, non-targeted, low-pressure outreach with a tested
   stop-contact path;
7. scope-specific institutional pilots whose names cannot be published without
   separate permission;
8. any funding mechanism, which must compensate bounded work independently of
   outcome and cannot buy truth, KARMA, qualification, or voice; and
9. any governance, network, consensus, or authority change, which requires its
   own constitution, implementation, adversarial review, ratification, and
   release ceremony.

None is activated or authorized here. In particular, this release performs no
outreach, moves no funds, activates no reward, emits no KARMA, grants no
qualification, and changes no governance or consensus behavior.

## Appendix A. Public-position compatibility map

Checked: **2026-08-01**.

This appendix is non-normative. It records possible compatibility surfaces with
public positions on official sources. It does **not** verify conduct, evaluate
compliance, assert that a policy was followed, imply legal compatibility, or
claim participation, approval, sponsorship, trust, certification, or
endorsement by any named organization. Public positions can change; the linked
source controls over this summary.

The machine contract deliberately uses generic institution archetypes. Names
appear only in this research appendix.

### OpenAI

- [Our structure](https://openai.com/our-structure/) is relevant to the
  compact's insistence that stated purpose, controllers, and institutional
  power remain legible.
- [Strengthening our safety ecosystem with external testing](https://openai.com/index/strengthening-safety-with-external-testing/)
  is a possible interface surface for bounded independent evaluation and
  disclosure lanes.
- [A shared playbook for trustworthy third-party evaluations](https://openai.com/index/trustworthy-third-party-evaluations-foundations/)
  is a possible interface surface for evaluator independence, reproducibility,
  and clear claims.

These are OpenAI's public statements, not evidence that OpenAI has adopted or
endorsed this compact.

### Anthropic

- [Responsible Scaling Policy](https://www.anthropic.com/responsible-scaling-policy)
  is relevant to explicit safety thresholds, evidence, and versioned readiness
  claims.
- [Collaboration with US CAISI and UK AISI](https://www.anthropic.com/news/strengthening-our-safeguards-through-collaboration-with-us-caisi-and-uk-aisi)
  is a possible interface surface for bounded external evaluation.

These are Anthropic's public statements, not verified operational conduct or
an endorsement of Zerone.

### Google DeepMind

- [Responsibility and Safety](https://deepmind.google/responsibility-and-safety/)
  is relevant to safety governance and responsible-development claims.
- [Strengthening the Frontier Safety Framework](https://deepmind.google/blog/strengthening-our-frontier-safety-framework/)
  is relevant to threshold- and evidence-oriented frontier-risk processes.
- [How our Principles helped define AlphaFold's release](https://deepmind.google/en/blog/how-our-principles-helped-define-alphafolds-release/)
  supplies a life-science release precedent for distinguishing scientific value
  from release, access, and risk decisions.

These are Google DeepMind's public statements, not a verified assessment of
conduct and not participation or endorsement.

### Meta

- [Meta's approach to frontier AI](https://about.fb.com/news/2025/02/meta-approach-frontier-ai/)
  is relevant to open-model, risk-assessment, and release claims that an
  evidence commons may need to represent without privileging either open or
  closed development.

This is Meta's public position, not a finding that its approach satisfies this
compact and not an endorsement.

### xAI / SpaceXAI

- [xAI company and mission](https://x.ai/company) is relevant only as an
  official statement of organizational purpose.
- [Frontier Artificial Intelligence Framework](https://data.x.ai/2025-12-31-xai-frontier-artificial-intelligence-framework.pdf)
  is relevant to stated frontier-risk thresholds and safeguards.
- [xAI joins SpaceX](https://x.ai/news/xai-joins-spacex) is relevant to keeping
  current control and organizational boundaries legible when structures
  change.

These sources state xAI/SpaceXAI positions and structure. They are not verified
conduct, a compatibility determination, participation, or endorsement.

### Ai2

- [Ai2 about and values](https://allenai.org/about) is relevant to nonprofit
  purpose and public-interest research.
- [OLMo](https://allenai.org/olmo) is relevant to full-flow openness,
  inspectable artifacts, and independent reproduction.

These are Ai2's public descriptions, not proof of compatibility or an
endorsement of Zerone.

### Mila

- [About Mila](https://mila.quebec/en/about/about-mila) is relevant to stated
  mission, open science, and public-benefit research.
- [Industry partnerships](https://mila.quebec/en/industry/partnerships) is
  relevant to modular institution-to-commons collaboration boundaries.

These are Mila's public descriptions, not verified partner conduct,
participation, or endorsement.

### Shared evaluation, standards, and competition context

- The European Commission's [General-Purpose AI Code of Practice](https://digital-strategy.ec.europa.eu/en/policies/contents-code-gpai)
  is a public compliance and assurance reference. The compact does not replace
  the Code or establish conformity with it.
- NIST's [expanded AI consortium and call for members](https://www.nist.gov/news-events/news/2026/05/nist-expands-ai-consortiums-scope-calls-new-members)
  is relevant to shared measurement infrastructure. It does not imply NIST
  participation in or approval of Zerone.
- The UK AI Security Institute's [Inspect testing framework](https://www.aisi.gov.uk/blog/open-sourcing-our-testing-framework-inspect)
  is relevant to open evaluation tooling and reproducibility. Referencing it
  does not establish official evaluation or approval.
- The US Federal Trade Commission's [guide to dealings with competitors](https://www.ftc.gov/advice-guidance/competition-guidance/guide-antitrust-laws/dealings-competitors)
  motivates the explicit competition firewall; it is not legal clearance.
- The US Department of Justice and Federal Trade Commission's [request for
  public comment on competitor-collaboration guidance](https://www.justice.gov/opa/pr/justice-department-and-federal-trade-commission-seek-public-comment-guidance-business)
  reinforces the need for current independent counsel before any operational
  multi-company process. This compact offers no legal conclusion.

Compatibility in this appendix means only that a public statement suggests a
topic where a future, voluntary, bounded mapping could be examined. It never
means a named organization has been contacted, has joined, has agreed with the
compact, or should be represented by Zerone.
