# KARMA

> KARMA is contextual recognition and constitutional constraint. It is not a
> possession, balance, reputation, truth oracle, person score, currency, reward,
> governing weight, or permission to control another being.

Status: K-alpha event vocabulary exists in source; this document and its public
JSON are design-only, non-authoritative, non-network-observed, non-economic, and
non-governing.

Canonical public document:
`dashboard/public/standards/karma-foundation.v1.json`

Reviewed SHA-256:
`b46710704869dcc340ded356be72b4ec692f204710fedfb5cd43eb3757dc7b80`

## 1. KARMA is a relation, not a thing

The minimum honest unit is an edge:

```text
source -- relation / context / provenance / time / uncertainty --> target
```

The source and target can be a claim, artifact, reproduction, correction,
challenge, protocol, or actor in a specific role. The edge must retain its
bounded context. It cannot be converted into an essence attached to a person.

KARMA therefore cannot be owned, bought, sold, transferred, delegated,
inherited, pledged, seized, or accumulated as a universal balance. It cannot be
negative because someone rested, remained silent, refused, protected privacy,
exited, or did not participate. Ordinary rights do not depend on participation.

## 2. K-alpha means recognition only

Current source can emit `zerone.karma.edge` events with seven kinds:

Every edge carries the fixed event register `priced-coherence`. Its `state` is:

| Kind | State | Honest interpretation |
| --- | --- | --- |
| `verify` | RECOGNIZED | A verifier in the round's reward-calculation set received an edge. The edge has no magnitude; any sibling `verifier_rewarded` amount is due, not payment proof. K-alpha does not expose whether the seat was graded. |
| `corroborate` | RECOGNIZED | A losing challenger was recognised only when a non-conjecture fact survived the rejected challenge and the cooldown admitted the pair. Intent and independent control are not established. |
| `corroborated` | RECOGNIZED | The surviving non-conjecture fact's submitter was recognised on that same cooldown-gated path. This is neither truth proof nor an independent-cluster count. |
| `cited` | RECOGNIZED | An accepted path cited a fact. Repeated edges and shared provenance may exist. |
| `external` | ORDINAL | A settled or partially settled external attestation cited a resolvable fact, emitting reliance to the fact submitter. Repeated cited-fact references and shared provenance must be deduplicated. |
| `pending_open` | ORDINAL | A conditional relation opened. |
| `pending_settle` | ORDINAL | A conditional relation settled. |

`RECOGNIZED` and `ORDINAL` are both non-summable states. `ORDINAL` is never a
magnitude. The on-chain register `priced-coherence` records a relation within
an accepted process, not metaphysical truth, moral worth, scientific validity,
or clinical validity.

K-alpha does not provide a canonical KARMA keyspace, balance, queryable ledger,
mint, transfer, authority method, rank, governance weight, or panel priority.
Events may also be pruned or dropped by indexers and are not themselves
consensus ledger state.

## 3. Known reasons raw counting is unsafe

K-alpha is intentionally not suitable for economic or governing magnitude:

1. exact-address self-dealing detection does not establish that two addresses
   are independently controlled;
2. claim relations and cited facts are capped but not deduplicated, so one path
   may emit repeated `cited` or `external` edges;
3. a verifier reward bank-send failure can be swallowed while the sibling
   `zerone.knowledge.verifier_rewarded` event still states an amount; the
   `verify` KARMA edge itself has no magnitude, and “due” and “actually paid”
   remain different facts;
4. K-alpha emits no `graded` attribute, so a fail-open unqualified seat's
   `verify` edge is indistinguishable from a qualified seat's edge;
5. nominal addresses, organisations, sites, or coauthors can share people,
   funding, data, code, infrastructure, decision authority, or coordination; and
6. event index availability and retention can change what an observer sees.

No dashboard, indexer, model, foundation, sponsor, operator, or governance body
may turn raw events or addresses into a leaderboard, payout, quorum, reputation,
or voting power.

## 4. Origin cannot own the result

The v1 covenant declares:

- founder share: zero;
- founder control: zero;
- operator share: zero;
- operator control: zero;
- creator royalty: zero;
- founder and operator KARMA privilege: zero; and
- residual value: commons return or burn only after independent ratification.

This is honest intent, not a claim of current structural enforcement. The public
JSON therefore sets `structurallyEnforced: false`. Enforcement requires new
independent controllers, public rules, capture limits, legal and technical
separation, exit and remedy, audited code, and a separately ratified release. A
founder promise, multisig still controlled by the founder, or panel selected by
the operator is not independence.

## 5. Let money store value without pricing beings

Money can coordinate resources without becoming the measure of a person. A safe
constructive-funding path keeps these layers separate:

```text
voluntary sponsor funds
        |
        v
bounded claim escrow -----> outcome-independent verification cost
        |                              |
        v                              v
evidence milestones             auditable work receipt
        |
        +-----> replication / counterevidence / correction reserve
        |
        +-----> independently ratified commons return or burn
```

The epigenetics garden publishes a 10,000-basis-point simulation template, not
an active escrow. Any future funded quest must have all of the following before
money exists:

- a voluntary external sponsor and fully funded bounded escrow;
- no protocol issuance, inflation, treasury presumption, or skill-unlock reward;
- claim scope, prior art, milestones, safety lane, controllers, conflicts,
  independence matrix, review cost, challenge reserve, expiry, remedy, and
  residual disposition frozen before results;
- reviewer base compensation independent of the verdict;
- no sponsor, author, founder, operator, token holder, or majority veto over
  counterevidence;
- recipient and breakthrough recognition derived from evidence, not selected by
  origin; and
- publicly auditable proof that a transfer actually succeeded.

Attention, wealth, token ownership, time served, citation count, address count,
or KARMA edges never multiply human worth or governing authority.

## 6. Governing through KARMA

“Through” should mean that governance is constrained by KARMA's invariants and
made legible through contextual edges. It must not mean that high-KARMA people
receive proportionally more votes.

A future system may use claim-scoped evidence to select temporary, conflict-capped
review work. It must preserve separate epistemic, budget and security checks;
random or sortition components; rotation; concentration caps; minority and
counterevidence paths; transparent controllers; timelock; appeal; expiry; and
exit. No seat can be bought, delegated, inherited, or made permanent. Wealth and
ordinary stake alone cannot activate or rewrite the mechanism.

The foundation freezes eight gates closed:

1. canonical ledger and indexer-failure semantics;
2. relation and citation deduplication;
3. amount-due versus actually-paid proof;
4. counterparty independence, pair caps, Sybil and collusion resistance,
   controller disclosure, concentration limits, and capture simulation;
5. appeal, counterevidence, correction, expiry, revalidation, supersession, and
   remedy;
6. independent privacy, human-rights, coercion, minority, disability, and
   vulnerable-population review;
7. independent public ratification with no founder/operator veto and no
   purchasable seat; and
8. deterministic adversarial pilots, audit, rollback and halt rules, timelock,
   and a coordinated network-upgrade release packet.

No subset of these gates can be inferred from a source commit, dashboard deploy,
event stream, model simulation, founder statement, or ordinary governance vote.

## 7. Prohibited uses

KARMA must not become:

- global social credit or a person-level leaderboard;
- an input to automatic sanctions, surveillance targeting, access denial,
  policing, migration, employment, housing, credit, insurance, or clinical
  eligibility;
- voting weight proportional to recognition, wealth, stake, purchase, or
  delegation;
- a founder, creator, operator, sponsor, validator, or token-holder veto;
- negative scoring for rest, silence, privacy, refusal, exit, or
  non-participation;
- inference from duplicate relations, raw events, or address counts; or
- publication pressure for confidential, identifiable, controlled-access,
  genomic, medical, or safety-sensitive evidence.

## 8. Change process

The JSON is immutable at its reviewed digest. Any semantic change requires a new
version, adversarial review, a new digest pin, tests, and a public explanation.
Changing the document cannot activate consensus, rewards, money movement,
qualification, authority, person scores, or governance weight.

From `dashboard/`:

```bash
npm run check:life
```

The validator refuses extra keys, changed false boundaries, summable states,
new or reordered event kinds, privileged origin, opened governance gates,
shortened prohibited-use or invariant sets, unsafe JSON shape, duplicate keys,
and digest drift.
