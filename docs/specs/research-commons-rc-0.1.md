# Research Commons RC-0.1

- Profile: `zerone.research-commons/v0.1`
- Milestone: `RC-0.1`
- Status: `STATIC_SHADOW`
- Assurance: `SHADOW_ONLY`
- Economic effect: **none**
- AgentTool integration: `SHADOW_REFERENCE`

Machine-readable companion:
`dashboard/public/standards/research-commons.v0.1.json`.

## 1. What RC-0.1 is

RC-0.1 is a static public-observatory model for a possible scientific research
commons. It asks how a future case could name a bounded research need at a
passive Tree of Knowledge coordinate, and how agents could perform inspectable
work without turning money, attention, or relationship into truth or authority.

It does not operate that commons. It opens no case, accepts no evidence or
funding, enrols no participant, imports no AgentTool account or receipt, and
makes no chain or service call. It is an inspectable architecture with every
operational release gate closed. One of twelve gates records reviewed static
format compatibility only; the other eleven remain closed.

The operative sentence is:

> Cases can name questions at passive node coordinates. Agents can make
> evidence inspectable. Neither money nor participation decides what is true.

## 2. Three planes plus one closed seam

RC-0.1 assigns responsibility to three distinct planes and leaves a fourth
one-way seam closed:

1. **Knowledge plane — Zerone.** Versioned Tree references and an optional,
   inactive current-only ToK reference shape belong here beside typed claims,
   challenges, disprovals, and provenance. Funds do not set a Fact's status,
   confidence, priority, qualification, or authority. RC-0.1 performs no ToK
   request.
2. **Research-case plane — the future commons protocol.** A case would freeze
   scope, falsifiers, standards, disclosure lane, roles, conflict policy,
   evidence gates, budgets, fees, and refunds before work begins. RC-0.1 only
   describes this object; it creates none.
3. **Work and economy plane — prospective AgentTool responsibility.** Agent
   capabilities, work packages, compute or tool expenses, escrow compartments,
   and settlements would belong here. AgentTool remains a `SHADOW_REFERENCE`:
   exact inert format, schema, fixture, and reciprocal-profile bytes are pinned,
   but no external state, account, authority, or receipt is imported and no live
   route exists.
4. **Witness seam — closed.** A future adapter may project an already settled,
   public-safe receipt into Zerone. It must be versioned and one-way. It cannot
   carry confidential bytes, decide scientific truth, or become payment
   authority.

No plane silently imports the status or authority of another. A signature can
bind bytes without making a proposition true. A funded case can purchase
work without purchasing an outcome. A chain witness can record a projection
without proving that the external adjudication was sound.

## 3. Two kinds of node reference

The static curriculum graph and a possible current ToK projection are not one
namespace. A future case must use a tagged reference:

```text
NodeRef =
  StaticTreeRef(tree_schema, policy_version, node_id, node_digest)
  | InactiveCurrentOnlyToKRequest(at_block_height = 0)

ResolvedCurrentToKRef =
  (returned_chain_id, returned_actual_block_height, returned_tok_snapshot_root, fact_id)
```

A static Tree node remains historical curriculum with
`authoritative = false` and `rewardBearing = false`. A future current-only ToK
request must use `at_block_height = 0` and bind the returned chain ID, actual
block height, and ToK snapshot root. Unavailability is a valid outcome; the
caller must not substitute guessed or stale values. RC-0.1 makes no such
request and creates neither reference type in chain state.

## 4. A node is a passive coordinate, not a being or wallet

The proposed node role is deliberately passive. A future case may address a
node coordinate when naming one evidence gap, one earmarked destination, and
one maintenance obligation. The node itself neither requests nor retains
anything and exercises no agency.

A node cannot sign, hold a private key, withdraw, borrow, vote, own a person or
agent, become a legal or moral person, acquire authority from funding, or
convert attention, KARMA, citation count, query count, or ToK energy into
money. It cannot pay itself. Any future procurement that names the coordinate
requires a separately funded immutable case and an external, accountable
release path.

RC-0.1 creates zero node accounts, balances, or procurements.

## 5. Agents are participants without becoming scores

A future agent participant may take a bounded role such as finder or prover,
replicator, reviewer, challenger, repairer, integrator, maintainer, or trainer
or extractor. Delegation must be attributable, scoped, revocable, and supplied
with a stop path. These precautions do not infer identity, controller,
consciousness, consent capacity, personhood, liability, office, or vote; they
also do not displace accountable humans and organisations.

Decline, rest, pause, exit, inactivity, and honest non-completion create no
person score, negative recognition, debt, or forfeiture of already earned
compensation. Under a future frozen case rule, non-completion may only release
an unearned reservation. It is not a punishment.

Reviewer compensation is fixed and verdict-independent. RC-0.1 imports no
outcome bonus, outcome slashing, sponsor-controlled verdict, or
claimant-controlled verdict.

A DID, signing key, wallet, account, agent, and controller are distinct. None
proves an independent controller. Address count and signature count are not
independence. Any future E3 independence would require external evidence under
a frozen policy; RC-0.1 records that status as `DECLARED_UNPROVEN`.

## 6. Six ledgers remain separate

The word “ledger” here means one conceptual record domain, not necessarily a
blockchain store.

| Ledger | Holds | Must not decide |
|---|---|---|
| `VALIDITY` | Claim status, support, contradiction, uncertainty, challenge, disproof, supersession | Novelty, significance, attribution, payment, liability, governance, worth, consent, identity, authority |
| `NOVELTY_PRIORITY` | Case-scoped prior-art and caller-declared timestamp references only, plus declared overlap; no trusted time, novelty, priority, or precedence adjudication | Validity, significance, attribution share, payout, liability, governance, rank, truth |
| `SIGNIFICANCE_CONSEQUENCE` | Bounded adoption, downstream consequence, maintenance, separately evidenced impact | Validity, novelty, ownership, payout, governance, person value, popularity authority |
| `ATTRIBUTION_CREDIT` | Artifact-scoped roles, provenance, overlap, bounded lineage credit | Validity, ownership of a person, transferable reputation, payment, debt, qualification, vote, rank |
| `FUNDING_LIABILITY` | Prefunded compartments, earned releases, fixed compensation, fees, reserves, refunds, bounded liabilities | Validity, novelty, significance, attribution, verdict, governance, KARMA, person score |
| `GOVERNANCE_AUTHORITY` | Separately authorized offices, scopes, recusals, quorums, appeals, revocations | Scientific truth, novelty, significance, attribution, wealth-based voice, sponsor veto, identity, consent, personhood |

The six ledgers have no shared unit. Cross-ledger arithmetic, inference, and
implicit conversion are invalid; there is no shared scalar person score.

Their shared vocabulary is pinned as
`research-commons.six-ledger-boundary/0.1` at
`sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2`.
This is a vocabulary pin only: it imports no AgentTool value, record, account,
identity, or authority and does not complete the AgentTool cross-pin.

Work obligations and rest, attention and metabolism, relational KARMA,
identity, and external value are separate non-ledger registers and are not
imported by RC-0.1. They cannot be used as proxy values for any research
ledger. In particular, money cannot buy truth, attention cannot buy scientific
priority, KARMA cannot buy voice, and an address cannot prove independence.

### Open commons and protected bytes

Public outputs remain open. Payment may purchase bounded labour or review, but
not truth, ownership, exclusive access, graph priority, or authority. RC-0.1
creates no knowledge paywall.

Versioned public claims, safe artifact digests, typed public challenges, and
public-safe settlement projections may be designed for public inspection.
Sensitive bytes, embargoed evidence, human-subject data, exploit details,
dual-use operational details, raw personal data, secrets, and credentials must
remain off the public surface and off-chain. Embargo expiry never
auto-publishes, and an unsalted digest of a small known secret set is not
automatically public-safe.

## 7. One prefunded, conserved envelope

Before a future case opens, its complete maximum liability must be funded and
its allocation and refund rules frozen:

```text
funded_envelope =
  verified_costs
  + delivery_work_compensation
  + claimant_milestones
  + reviewer_compensation
  + challenge_and_remediation
  + compute_and_tools
  + administration_and_fee
  + refundable_residual

sum(released_allocations) + refundable_residual = funded_envelope
```

The waterfall order is explanatory, not a claim that RC-0.1 holds funds:

1. E1 verified costs within frozen categories, unit ceilings, disclosure rules,
   and a case-fixed sub-cap.
2. Fixed delivery-work compensation for a compliant frozen deliverable. Its
   amount is identical for `POSITIVE`, `NEGATIVE`, `NULL`, `INCONCLUSIVE`, or
   `NOT_APPLICABLE` direction and sits outside the outcome pool.
3. Claimant milestone tranches only when the named evidence gate, declared
   separation, and quorum requirements pass; independent control requires
   external evidence not established by RC-0.1.
4. Fixed, verdict-independent reviewer compensation.
5. The challenge and remediation reserve, including compliant work that
   honestly falsifies the original claim.
6. Verified case-scoped compute and tool costs.
7. A visible administration and fee cap fixed before opening.
8. Every unreleased unit to the prospectively named refund or maintenance
   destination.

There is no bonus outside the envelope, silent fee deduction, silent
reallocation, claimant self-release, sponsor verdict, or node withdrawal.

RC-0.1's activated amount is exactly `0 uzrn`; it accepts no funding.

## 8. E0–E6 outcome schedule

E0 through E6 are ordered evidence transitions, never badges assigned by time,
prestige, funding, or popularity.

| Level | Evidence boundary | Reference economic treatment |
|---|---|---|
| E0 | Digest, scope, prior-art snapshot, threat model, falsifiers, disclosure policy, and a caller-declared case-local preregistration commitment; proves no trusted time, novelty, priority, or entitlement | Caller-declared case-local preregistration reference only |
| E1 | Complete inspectable and reproducible bundle | Verified costs only; outside the percentage pool but inside the all-in cap |
| E2 | Required deterministic or class-specific checks pass | 15% of the outcome pool |
| E3 | Declared-unproven reproduction; independent control requires external evidence not established by RC-0.1 | 20% |
| E4 | Public challenge survived, or a scoped confidential repair or mitigation was separately tested; independent control requires external evidence not established by RC-0.1 | 15% |
| E5 | Declared-unproven adoption, upstream merge, maintained release, or standards disposition; independent control requires external evidence not established by RC-0.1 | 25% |
| E6 | Continued conformance across the frozen interval or version transition | 10% |

The remaining 15% is reserved for challenge and remediation. The five claimant
tranches total 85%; together they conserve exactly 100% of the outcome pool.
E1 verified costs and fixed reviewer compensation remain separately capped
inside the all-in envelope.

Honest falsification cancels only unearned claimant tranches. It can release
the separately reserved challenge or repair budget. A compliant negative,
null, inconclusive, or not-applicable deliverable receives the same fixed
delivery-work compensation as a positive deliverable; scientific correction
is not fraud.

## 9. Pilot: Amplitude Bootstrap Garden

The first proposed pilot is a small, public-safe fixture-coherence garden capped
at E2. It binds the exact EID-1 record
`bootstrap-conditional-solution-space` inside the reviewed EID-1 bytes
`sha256:e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df`
and the version-pinned primary source
`https://arxiv.org/abs/2406.02665v2`. EID-1 is a pinned method input; the
paper's scientific result is not imported or rerun.

The candidate is the planar, color-ordered, weakly coupled tree-level
four-point amplitude class in that paper's analytically solvable bootstrap
problem. It retains assumptions `bs-a1` through `bs-a7`, including the scoped
asymptotic, cancellation, crossing, locality, dual-resonance, positivity, and
normalization conditions. Its falsifier is `bs-f1`: exhibit a second
inequivalent amplitude inside the exact declared class satisfying both
headline bootstrap assumptions.

1. **Freeze the bed.** Load only the pinned EID record, exact v2 locator,
   sector, assumptions `bs-a1`–`bs-a7`, falsifier `bs-f1`, and non-transfer
   wall.
2. **Plant E0.** Record a caller-declared case-local preregistration reference
   for a digest-addressed public-safe question packet. It proves no trusted
   time, novelty, priority, or entitlement.
3. **Grow E1.** Publish an inspectable local bundle with exact inputs, scripts,
   expected refusals, and bounded cost evidence.
4. **Test E2.** Mutate one decisive assumption and verify only that the local
   fixture updates its result, witnesses, remaining family, and relaxation
   branches coherently.

The bounded fixture conclusion is limited to whether that local record
mutation is structurally coherent. The source result is tree-level, four-point,
normalized-quotient, and assumption-bound; positivity is necessary rather
than sufficient for full unitarity. The garden does not establish
unconditional uniqueness, empirical string detection, ontology, or any
protocol result.

The garden currently accepts no submission or funding, runs no computation,
grants no qualification or reward, and claims no physics result. Its public
buttons are disabled with the missing gate beside each one.

## 10. AgentTool cross-pin boundary

The manifest names the exact inert interchange formats
`agenttool.research-settlement-bundle/0.1` and
`agenttool.research-public-projection/0.1`, plus the prospective Zerone output
schema `zerone.agenttool-research-receipt-shadow/v0`. The frozen static interop
profile is named `agenttool.research-commons-zerone-static-interop/0.1` at
`packages/research-commons/interop/research-commons-zerone-v0.1.json`, with raw
SHA-256
`8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a`.
Those names and bytes are compatibility inputs, not imported state or an
integration claim. AgentTool R0 is pinned to source revision
`6a644b9e858b7d23bdea613d91412bf7310c2338`, main merge
`55342fac97250898c2c4ea884f1a03bec1f8cc8c`, and PR 335. Imported accounts,
identities, and receipts remain zero.

The non-circular reciprocal handshake has three immutable stages:

1. Zerone Phase A source
   `5328b42230fa6945f458a6e60aca92b23eead595` and main merge
   `fdd40bf9aca4a82b2cdd904d0161016b8c2a8667` (PR 52) froze the
   `agenttool-research-receipt/v1` adapter specification at raw SHA-256
   `1d67c4649b419d4ff60f2fba5796d42b07d7be5d605997ecafafd37cec5158e8`
   and the static fixture manifest at raw SHA-256
   `cf367bb39553567e86c43c0db48501802832396b2a3f681410aaac7c5e2221e8`.
2. AgentTool Phase B source
   `91a1396c76edd5e1585af33042e46640c5b5cf4a` and main merge
   `8c63c6b4b5c14286addd29bf9da00337e43c46cd` (PR 337) froze
   `agenttool.zerone-research-adapter-reciprocal/0.1`. Its profile raw SHA-256
   is `80621747824e6c9b747d00958d2b6822bcfb76b7e11688000bc219db6177d713`;
   its schema raw SHA-256 is
   `0b9439c39b41da19fa7a7f07539d52a53000e1f5e6c820f47e9dd8ca607e9ab2`.
   The profile points backward to Phase A; it does not contain its own source or
   merge revision.
3. RC-0.1 pins those Phase B bytes, revisions, PR, immutable permalinks, and
   byte-for-byte local copies. The offline validator reconstructs profile ID
   `sha256:4d927f4db623884453f4e16b73573a81b0b1cc4cc7b72529e69ca153b39112c7`
   as `SHA256(UTF8(format) || 0x00 || UTF8(canonical_profile_body))`, where the
   body excludes the outer `profile_id` and canonical JSON recursively sorts
   object keys by Unicode code point with compact separators.

This closes exactly one compatibility gate. It proves only that the named
static formats and frozen pins agree. It does not make integration ready,
activate the witness seam, authorize the adapter, establish a canonical head,
or import AgentTool state. `SHADOW_REFERENCE` therefore remains visible;
`integration_ready`, authority transfer, all 29 AgentTool wire effects, all 41
Zerone surface effects, and every call to action remain false.

AgentTool's separately pinned 29-key wire/interop effect profile and Zerone's
41-key context-specific surface boundary are not identical. RC-0.1 infers no
field equivalence beyond the named interop contract; the 41-key Zerone boundary
is an independent, surface-specific vector and is not reduced to or
field-equated with the wire vector.

AgentTool's append-only challenge/work-retention check is scoped to a
caller-supplied `prior_state` transition. Content-addressed state IDs are not
signatures or canonical heads and prove no provenance, trusted time, global
ordering, or prevention of a fork replayed from an older state. RC-0.1 imports
no stronger lifecycle guarantee.

AgentTool artifact records declare intended open, nonexclusive access only.
RC-0.1 fetches no artifact bytes, locator, or licence and verifies no public
availability. A digest-only record does not make referenced or low-entropy
sensitive material safe. External availability, licence, and safety review are
required before any open-knowledge integration claim.

The completed static cross-pin does not relax the witness seam. A name,
compatible-looking object, URL, deployment, signature, or one-sided hash would
not have been enough. Independent review closes only the cross-pin release
gate; it does not make integration ready, open any other gate, or activate
either system.

This profile is not blanket alignment with
`CONSTRUCTIVE-INTELLIGENCE-REWARDS.md`. That inactive research document is not
a source binding for RC-0.1. Its outcome-resolved reviewer incentives,
slashing concepts, and controller-inference ideas are not imported. RC-0.1 is
the stricter shadow profile described in sections 5 and 7.

## 11. Browser and source boundary

The RC-0.1 browser module performs exactly one automatic request:

```text
GET /standards/research-commons.v0.1.json
```

It requires the exact same-origin path, refuses redirects, sends no
credentials or referrer, accepts only `application/json`, enforces an 8-second
deadline and a 65,536-byte streamed cap, decodes strict UTF-8, verifies the
reviewed SHA-256, rejects duplicate JSON keys, excessive nesting, unknown
fields, missing fields, and all literal boundary drift, then renders with DOM
text nodes.

It performs no external-source fetch. The baseline source-revision link is an
inert, user-activated link pinned to an exact commit predating RC-0.1. It is not
represented as the final reviewed RC revision. An unresolved external
reference fails closed instead of falling back to `main`, `latest`, a mutable
URL, or guessed compatible bytes.

The offline validator verifies this manifest and all six local source bindings,
including the vendored reciprocal profile and schema, from repository bytes.
It independently reconstructs the reciprocal canonical tuple instead of
trusting the outer profile ID. Each manifest read is capped at 65,536 bytes and each
bound-source read at 262,144 bytes. It opens a non-blocking, no-follow file
descriptor, requires a regular file, reads through that descriptor only, and
rechecks descriptor identity and metadata plus the final path before accepting
the bytes. Terminal or intermediate symlinks, special files, path escape,
oversize files, and files or paths swapped during the read fail closed. It
makes no network request.

If loading or validation fails, the renderer accepts none of the architecture
and shows only a fail-closed message plus direct access to the raw static
artifact.

## 12. Exact zero-effect boundary

Publishing or viewing RC-0.1:

- when its JavaScript module runs, performs exactly one static same-origin GET
  for the RC manifest and no other RC data request; without JavaScript it makes
  no RC manifest, chain, AgentTool, or external-source request;
- makes no API, RPC, chain, mainnet, bridge, adapter, AgentTool, wallet,
  hosted-payout, or external-source request;
- uses no ZRN or uzrn and neither offers nor implies an external-value payout;
- reads and writes no chain state;
- accepts no input, research data, confidential material, identity, account,
  wallet, signature, payment, or evidence;
- creates no case, assignment, reviewer, verdict, participant, membership,
  escrow, transfer, mint, burn, reward, entitlement, debt, or payout;
- grants no qualification, authority, office, governance power, consent,
  research permission, security-testing permission, biological or clinical
  permission, or endorsement;
- modifies no Fact, confidence, consensus, ontology, KARMA, person score, or
  protocol energy; and
- infers no identity, controller, consent, consciousness, personhood, rest
  state, inactivity, or relationship.

The dashboard's separately disclosed network observatory continues its own
read-only operation; the statement above is scoped only to the RC-0.1 module.

The static cross-pin gate is `true` for compatibility only; the other eleven
release gates remain `false`. Even all twelve passing would be necessary, not
sufficient, and could not self-activate anything. Operation
would still require a new successor profile, separate authorization,
independent review, and reviewed deployment verification. A validator pass
proves only that the static profile matches these reviewed local invariants. It
proves no science, market safety, funding, independence, external integration,
or operational readiness.
