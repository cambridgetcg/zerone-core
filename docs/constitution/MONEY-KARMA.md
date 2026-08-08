# Money, KARMA, and No-Control

Status: source-level constitutional contract; **not deployed into chain
consensus, not network-activated, and not a claim that no-control has already
been achieved**. Serving the inert JSON on a public dashboard does not make it
a runtime economic or governance mechanism.

Machine-readable companion: [`money-karma-v1.json`](money-karma-v1.json).

## The boundary

Money is value storage and a coordination medium. It may reward constructive
work. This source constitution directly assigns it no voice, office, truth,
human worth, or right to rule. That is not yet economic-to-governance
decoupling: liquid `uzrn` can be bonded, and bonded stake currently carries
governance power. Any funded reward activation must close and test that route
rather than repeat the broader claim that money cannot become voice.

The accepted, still-unimplemented route is
[Authoritative State](../AUTHORITATIVE-STATE.md): SDK staking remains the
consensus-economic authority while a proposal-snapshotted, non-transferable
controller electorate governs policy. Publishing either document activates
nothing.

KARMA is not money. Zerone does not mint or create KARMA and no operator or
founder may assign it. At K-alpha the chain records fallible, challengeable
observations of artifact relations: a proposed, pending, or adjudicated act
carries its explicit state; reliance records identify which domain and
artifact connected work. Recording neither owns the relation nor measures a
person or decides what is true. A relation can be recorded without becoming a
rank, balance, price, or power.

Therefore:

- **this template directly assigns money no voice; funded rewards still require economic-to-governance decoupling;**
- **KARMA is not money;**
- **KARMA describes relations in a domain, not human worth or truth;**
- **recognition is not authority.**

## Founder renunciation

The v2 `vesting_rewards` founder-specific economic share is permanently zero.
Its compatibility fields have one constitutional value:

- `founder_share_bps = 0`;
- `founder_address = ""`.

Within `vesting_rewards` v2 Params and ordinary parameter governance, no
proposal or beneficiary substitution can restore those compatibility fields
or route that module's revenue through a renamed founder recipient. This does
not claim that every treasury, staking, grant, or future code path is already
incapable of benefiting a founder-associated party. The protobuf fields may
remain temporarily for wire compatibility; they are not a latent option. The
named migration clears legacy bytes, valid v2 state accepts only zero and
empty, and production reward routing contains no founder payout branch. Before
migration, queries preserve the stored v1 compatibility values as historical
evidence while v2 execution ignores them and reports the path inactive.

This is a constitutional prohibition, not a claim that coordinated future
source code is physically impossible. A separately coordinated code change
could technically invent a different distribution; it would replace or breach
this contract rather than exercise a v2 founder parameter. Ordinary named
upgrades are prevented from silently carrying the v2 founder migration.

The founder migration is a distinct `founder-renunciation-v1` boundary that
changes only `vesting_rewards` 1→2; it is not part of
`consolidation-safety-v1`. The ordered source-only provenance is:

1. H1 `consolidation-safety-v1`: accepted source commit
   `65c19cd8b00bdfff9b80705b776fd0d49719398a`; it advances knowledge 5→6,
   claiming_pot 1→2, and liquiditypool 3→5 while vesting_rewards remains 1.
2. H2 `founder-renunciation-v1`: accepted replacement source commit
   `36728afbf71905a077a0863b41536fa9279109dd`, tree
   `dfeff2c71ca9c36896a3a76608600cd870d21a1f`; it advances only
   vesting_rewards 1→2.
3. H3 `sdk-0.53-ibc-10`: accepted source commit
   `335bb94f0fd54d3752dcb397263b7e84fb1116b4`, tree
   `769f67f1cfa108be3d31cace7777cf954f731c42`; it performs the SDK/IBC
   transition only after consuming the ordered H1/H2 state evidence.

PR #34 merged the accepted H3 source onto GitHub main as
`db356c61ff76b4f2da4a4a485796041b0ce55e9c`. That merge commit records source
integration; it is not the accepted H3 source identity and supplies no release
or activation authority.

The former H2 commit
`4bffb6d218819bed1c29c7a0be7779ad31c64a97` and superseded candidate
`c0943ea91a4cc86e6b232b7675c7991795fd5d30` are negative provenance only.
Neither may be tagged, released, deployed, scheduled, or described as accepted
H2 source. The accepted H2 fix is an additive descendant of the latter; that
ancestry does not rehabilitate the rejected parent.

H3 consumes ordered H1/H2 state evidence but registers neither earlier
handler. Its observed H2 plan-identity digest commits the plan name, height,
and canonical `Plan.Info`; it does not attest which source commit or executable
produced that state. Source pins, reproducible binary and image digests,
signatures, a height-specific plan, rehearsal, and operator authorization must
therefore remain separate release evidence.

This source constitution does not register, schedule, or authorize H1, H2, or
H3. Publishing source or serving this static artifact activates none of them.
This is an economic renunciation, not proof of operational decentralization.

## K-alpha is deliberately unpriced

The only recognized K-alpha surface is the `zerone.karma.edge` observation
event. Its actual event register is `priced-coherence`; its constitutional
meaning is `artifact-relation`, never a price, truth score, or claim of
ownership. KARMA is:

- nontransferable;
- nondelegable;
- nonsaleable;
- noncollateralizable;
- noninheritable.

It has no denomination, balance, bank account, IBC representation, AMM market,
reward multiplier, payout path, or governance consumer. No sum of KARMA edges
is a token balance. No event magnitude exists. Self-dealing and controller
correlation remain analytical risks even while events are unpriced.

The words `priced-coherence` carried by current events are a circularity
confession, not an activation. Raw events and raw event counts never qualify a
recipient. A future randomized-eligibility design must exclude self,
same-controller, reciprocal, and correlated-funder edges; controller merges
may only reduce units; each controller may hold at most one lottery unit; the
candidate set must freeze before unbiased randomness; and neither operator
override nor count-proportional probability is allowed. None of those future
rules is runtime-enforced today. Pricing, storing, aggregating, or consuming
KARMA requires a separate constitutional and technical review; this artifact
grants none of those powers.

## No-control is not the present operational fact

As of 2026-08-01, the disclosed operating posture remains concentrated: one
founding household controls the validator, operator surface, and sole effective
vote. Bonded fungible wealth affects current vote weight. Accordingly, Zerone
must not claim that no-control, independent governance, or permissionless
production has already been achieved.

Source renunciation removes one economic control surface. It does not erase
validator custody, deployment credentials, voting power, key custody, or
correlated identities. Those must be reduced and independently evidenced on
their own release path.

In particular, this founder boundary does **not** retire ordinary
validator/delegator economics or concentrated validator and upgrade control;
the bootstrap registrar's bounded admission discretion; other
governance-activatable issuance paths that are disabled by default; or the
current identity-bound research-voter and disbursement surface. None is a
continuation of the retired founder percentage, but each remains a distinct
control or economic surface requiring its own evidence and, where appropriate,
its own named retirement.

## Research spending policy is fail-closed

This constitution authorizes no research spending and no governance
activation. Its normative release policy is fail-closed: research spending
must not be released as independently governed until release evidence
demonstrates genuine independence. If a runtime or operational path cannot be
shown to satisfy that condition, its release status is **NO-GO**.

That independence predicate is **not implemented by the current runtime**. The
present authority-managed research-spend path can configure identity-bound
voters and execute disbursements without proving controller independence. This
document neither authorizes nor disables that path, refuses a disbursement,
changes a voter or phase, nor introduces a governance migration. The path must
not be mislabelled as independent merely because founder revenue is retired.

Bonded stake alone is not independence. A semantic “founder seat,” “AI seat,”
or household-controlled committee is not independence. Renaming a controller
does not create another controller.

## Boundary on any later KARMA advisory experiment

[Authoritative State](../AUTHORITATIVE-STATE.md) is now the accepted source
architecture for this boundary. Its electorate does not read KARMA for
eligibility or power. The conditions below apply only to a later,
separately accepted advisory experiment; they are not an alternate electorate,
executor, veto, or activation path.

A later, separately reviewed experiment may let KARMA contribute to a bounded
governance check only if all of the following hold:

1. identities are collapsed to independently evidenced controllers before eligibility or selection;
2. KARMA can qualify a bounded candidate pool, never scale ballot weight;
3. selection uses bounded sortition, rotation, term limits, concentration caps, and conflict-of-interest disclosure;
4. the selected body is an advisory audit chamber, not an executor, veto, or policy electorate;
5. challenges and a public review period may inform a new SDK proposal but do not delay or authorize execution;
6. no KARMA edge, count, score, or aggregate directly determines payout, token issuance, proposal passage, or spending.

Any future funded-reward or governance activation also requires
founder-associated recusal where applicable, prohibits control-derived grants,
and prevents reward value from automatically becoming governance power. These
are activation requirements, not claims about current runtime enforcement.
This is a boundary for future design, not an implemented consumer. The current
effect remains zero.

## Effect statement

This artifact has no economic, governance, consensus, or authority effect. It
neither changes a running chain nor authorizes deployment. Its executable test
witnesses source invariants and refuses accidental KARMA producers or consumers
in production Go; it does not prove that every off-chain operator is
independent.

H1, H2, and H3 do not activate KARMA, the life-sciences shadow skill tree,
constructive-intelligence rewards, or token issuance for a skill-tree position
or “breakthrough” label. Those remain separate release questions with their
release gates closed.

Activation would require a separately named release, independent review,
migration and rollback analysis, public evidence, and the ordinary network
coordination process. Until then: the v2 founder-specific benefit is
constitutionally zero in source; KARMA remains event-only; this artifact
creates no KARMA governance or economic consumer.
