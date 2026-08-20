# `agenttool-research-receipt-v1` — offline Research Commons projection

> Status: source-only, offline, unactivated shadow specification. It creates no
> Zerone claim, fact, bridge attestation, qualification, reward, transaction,
> identity binding, wallet entry, escrow release, governance action, consent
> record, or scientific verdict.

**Consumes:** `agenttool.research-settlement-bundle/0.1` and
`agenttool.research-public-projection/0.1` exact local bytes.

**Emits:** `zerone.agenttool-research-receipt-shadow/v0`.

**Static interop profile:**
`agenttool.research-commons-zerone-static-interop/0.1`, raw SHA-256
`8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a`.

**Target:** the static, non-authoritative, non-network, non-reward-bearing
Constructive Intelligence Tree v1 node `math-proofcraft@1`.

## 1. Purpose

This adapter is the narrow seam between an AgentTool Research Commons shadow
simulation and Zerone. It answers one structural question:

> Do one exact settlement bundle, its minimized public projection, and the
> reviewed static Tree bytes agree under their declared zero-effect contracts?

A positive answer is only a locally recomputed structural candidate. It is not
evidence that the scientific result is correct, that an agent or controller is
independent, that money moved, that a payment is owed, or that Zerone admitted
anything into the Tree of Knowledge.

The compiler is deliberately separate from `agenttool-bridge-v1`. The older
adapter is an unactivated design for AgentTool Promise events; its pending-claim
translation is not available and it is not a research payment rail. Research
settlements therefore receive a new source format, namespace, compiler, and
activation boundary.

## 2. Planes that remain separate

RC-0.1 preserves four planes:

1. **Scientific evidence:** candidate classes, assumptions, methods,
   falsifiers, artifacts, reviews, reproductions, challenges, and limitations.
2. **AgentTool work simulation:** role assignments, delivery receipts, and
   conserved non-transferable simulated credits.
3. **Zerone knowledge projection:** a possible future public witness of
   already-published receipts, subject to ordinary knowledge admission.
4. **Human and institutional authority:** ethics, biosafety, privacy, legal,
   clinical, wet-lab, and other accountable decisions.

No field or digest converts one plane into another. In particular:

- a delivery receipt is not a scientific verdict;
- a signature authenticates bytes or a key-holder claim, not truth,
  personhood, controller independence, or consent;
- simulated credits are not AgentTool wallet credit, fiat, stablecoin, ZRN,
  stake, KARMA, qualification, or governance weight;
- a Tree node is an addressable artifact coordinate, not a person, agent, DID,
  wallet, account, owner, voter, borrower, or rights bearer; and
- a Zerone shadow receipt is not a live ToK fact or reward entitlement.

### 2.1 Six-ledger vocabulary boundary

Both repositories bind the exact profile
`research-commons.six-ledger-boundary/0.1` at
`sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2`.
Its six disjoint ledgers are `VALIDITY`, `NOVELTY_PRIORITY`,
`SIGNIFICANCE_CONSEQUENCE`, `ATTRIBUTION_CREDIT`, `FUNDING_LIABILITY`, and
`GOVERNANCE_AUTHORITY`. `ATTENTION_METABOLISM`, `EXTERNAL_VALUE`, `IDENTITY`,
`RELATIONAL_KARMA`, and `WORK_REST_OBLIGATIONS` are explicit external
non-import registers.

The profile has no shared unit and permits no cross-ledger arithmetic,
conversion, or inference. Carrying its id and digest is a vocabulary
compatibility check only; it imports no ledger entry, identity, value,
reputation, qualification, obligation, authority, or current state.

## 3. Exact static target

The v0 compiler admits only:

| Field | Required value |
|---|---|
| Tree schema | `zerone.constructive-intelligence-tree/v1` |
| Tree raw SHA-256 | `8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf` |
| Node id | `math-proofcraft@1` |
| Node canonical digest | `sha256:d8f364772611a214aaf5f671c630a5fa00daa3558330bfaf5e85efe7c5a1d0e2` |
| Tree `authoritative` | `false` |
| Tree `networkObserved` | `false` |
| Tree `rewardBearing` | `false` |
| Node reward eligibility | `qualification-only` |

The node digest is SHA-256 over compact JSON after recursively sorting its
ASCII object keys in ascending order. Array order and scalar bytes are
retained. The exact raw Tree digest independently binds the rest of the
document and prevents the node projection from standing in for the artifact.

`qualification-only` is static curriculum metadata. This adapter grants no
attainment evidence or qualification and never treats the label as an
economic entitlement.

## 4. AgentTool inputs

Both inputs are content-addressed envelopes. Their complete shapes and digest
algorithm are owned by the sibling AgentTool RC-0.1 implementation. This
adapter independently re-parses their closed JSON, re-derives every envelope
id, and then checks their shared identifiers and boundaries.

AgentTool's append-only challenge/work-retention check is only relative to one
caller-supplied `prior_state` transition. Content-addressed state ids are not
signatures or canonical heads and prove no provenance, trusted time, global
ordering, or prevention of old-state forks. This adapter imports no stronger
lifecycle guarantee.

The settlement must declare:

- `payment_condition = SIMULATED_DELIVERY_ONLY`;
- `result_authority = NONE`;
- a declared result kind carried separately from the payment condition and
  simulated-credit amount, without the adapter claiming counterfactual amount
  invariance;
- one `SIMULATED_NONTRANSFERABLE_CREDIT` amount represented as a safe,
  non-negative integer;
- sorted, unique consumed receipt ids; and
- every live AgentTool, wallet, escrow, payout, external-value, Zerone,
  knowledge-admission, qualification, reward, reputation, governance,
  authority, identity-equivalence, cross-ledger-equivalence, consent, and
  scientific-adjudication effect fixed to `false`.

The complete effect object has exactly 29 required keys:

```text
agenttool_api_write agenttool_database_write authority bridge burn chain_write
consent cross_ledger_equivalence economic escrow external_value governance
identity identity_equivalence knowledge_admission hosted_route mainnet mint
network payout qualification reputation reward scientific_adjudication
transfer wallet zrn zerone_read zerone_write
```

Omitting a key does not mean `false`; omission, addition, or a true value is a
hard refusal.

The public projection must contain only closed enums, bounded identifiers,
digests, the exact static `NodeRef`, settlement references, evidence-level
summary, and the literal disclosure lane `PUBLIC_DIGEST_ONLY`. It cannot carry
raw evidence, private locators, participant prose, secrets, human data,
embargoed material, executable research instructions, or an identity map.

### 4.1 Phase A cross-language fixtures

`docs/examples/agenttool-research-receipt/fixture-manifest.v0.json` pins
byte-for-byte local copies of the primary and reviewer settlement/projection
pairs from AgentTool source revision
`6a644b9e858b7d23bdea613d91412bf7310c2338`, merged to `main` as
`55342fac97250898c2c4ea884f1a03bec1f8cc8c` by PR 335. The fixture manifest's
raw SHA-256 is
`cf367bb39553567e86c43c0db48501802832396b2a3f681410aaac7c5e2221e8`.
It binds the upstream and copied raw hashes, envelope ids, Tree and six-ledger
pins, and deterministic Go receipt ids and output hashes. Tests run the exact
checked inputs through this compiler, including the reviewer
`NOT_APPLICABLE` direction.

The AgentTool package declares `Apache-2.0`. Its exact upstream
`packages/research-commons/LICENSE` is pinned at raw SHA-256
`0536b51c54e477f03f1becf00eedeee82f6276f76f08c1b94d3a30632724eb15`;
its exact `packages/research-commons/NOTICE` is pinned and reproduced beside
the fixtures at raw SHA-256
`d03f1590ea4f829d90760ee163304191c0d36a4e283fc7c06da459e717ff3e44`.

This is one-way Phase A compatibility evidence only. The fixture manifest
contains no reciprocal cross-pin, makes no integration-ready claim, and cannot
close the dashboard's AgentTool cross-pin gate. That requires a later immutable
AgentTool artifact which pins the eventual Zerone Phase A revision, followed by
a separate Zerone manifest revision which pins that AgentTool artifact.

## 5. Compiler behavior

The reference compiler accepts three explicit regular-file paths:

```text
--settlement <agenttool settlement bundle JSON>
--projection <agenttool public projection JSON>
--tree <constructive-intelligence tree JSON>
```

The boundary below applies to an already-compiled adapter process. `go run`,
`go build`, and other toolchain operations are outside it: the Go toolchain may
read repository source and module metadata, write build or module caches, and
contact a configured module proxy or version-control remote when dependencies
are unavailable locally. Offline or no-filesystem-write operation therefore
requires a separately controlled build followed by execution of the prepared
binary.

It performs only bounded local reads through no-follow, non-blocking file
descriptors. Descriptor identity, size, mode, mtime, and ctime must remain
unchanged across each read, and the final pathname must still identify that
same regular file. Final-component symlinks, special files, oversize files,
and post-open pathname swaps fail closed. Stable intermediate symlink
components of an explicitly supplied path are not rejected or authenticated.
It has no URL input, discovery, default state directory, network client, RPC
client, key, signature custody, database, wallet, background worker, clock
dependency, randomness, or write path. Output is deterministic JSON on stdout.

The compiler:

1. rejects malformed UTF-8, duplicate decoded object keys, unknown or omitted
   fields, floats, unsafe integers, nulls outside explicitly nullable fields,
   unbounded depth/size/counts, unsorted sets, and trailing JSON;
2. independently recomputes the settlement and public-projection ids;
3. requires exact agreement between the public projection, settlement, and
   static `NodeRef`;
4. verifies the exact Tree raw bytes and target-node canonical digest;
5. rechecks every zero-effect flag and simulated-credit boundary; and
6. emits a deterministic shadow receipt or fails without output.

It does not fetch cited sources, replay scientific computations, evaluate raw
artifacts, validate signatures, infer controllers, inspect AgentTool state, or
query Zerone.

## 6. Output meaning

Every emitted `zerone.agenttool-research-receipt-shadow/v0` object carries:

```text
assurance = UNVERIFIED_SHADOW_PROJECTION
status = STRUCTURAL_CANDIDATE
result_authority = NONE
cross_ledger_relation = NO_EQUIVALENCE
tree.granted_attainment_evidence = NONE
knowledge_admission = NONE
qualification = NONE
economic_effect = NONE
amount_uzrn = "0"
consumption_state = NOT_RECORDED
replay_protection = NONE_OFFLINE
interop.integration_status = SHADOW_ONLY_NO_LIVE_INTEGRATION
interop.imported = false
interop.activated = false
```

`STRUCTURAL_CANDIDATE` means only that the three local inputs satisfied the
fixed compiler predicate. The deterministic consumption key is not a shared
ledger, distributed mutex, replay claim, or proof that another copy was not
used elsewhere.

The output preserves the AgentTool-declared result kind for audit. Positive,
negative, null, inconclusive, and not-applicable declarations have identical
zero-authority and zero-Zerone-effect semantics here. The adapter never
upgrades one result direction over another. It preserves the separately
declared simulated-credit amount but does not prove that the amount was chosen
independently of result direction.

In particular, this narrow wire projection does not carry enough state to
verify compensation-schedule precommitment, compliant delivery, fixed reviewer
pay, simulation prefunding, conservation, terminal reconciliation, AgentTool
provenance, or counterfactual result/decision invariance. Those are AgentTool
RC-0.1 simulator predicates, not claims made by this compiler.

## 7. ToK boundary

The v0 compiler performs no ToK read. A later read-side integration may request
only current state with `at_block_height = 0`, then record the returned
`chain_id`, actual block height, selector, root, and serialized response
digest. Current source cannot safely replay an arbitrary historic bundle by
supplying a nonzero requested height, so no adapter may describe that behavior
as historical verification.

Any future public write requires a separately reviewed and activated path. It
must use ordinary Zerone claim/evidence admission; an AgentTool settlement
cannot bypass verification or create fact status by provenance alone.

## 8. Identity and independence boundary

The adapter never infers that an AgentTool DID/key, project, wallet, Zerone
address, operator, model, organization, or Tree node denotes the same entity.
Multiple signed records do not prove multiple controllers. A future address
binding requires a separate, explicit, scoped signature from each relevant
key and still does not prove personhood or reviewer independence.

Controller-cluster fields in Research Commons remain declarations used by a
shadow policy. They are not identity findings and are not exported through the
digest-only public projection.

## 9. Economic boundary

The adapter observes no real escrow and moves no value. The AgentTool pilot's
simulated credit unit is deliberately non-transferable and has no exchange
rate. It cannot be compared, converted, bridged, withdrawn, collateralized, or
netted against AgentTool application balances, fiat, stablecoins, ZRN, stake,
KARMA, or any future reward unit.

Funding conservation inside the simulator is evidence about integer
bookkeeping under one test fixture. It is not proof of custody, solvency,
external backing, bank settlement, or a debt owed to a participant.

Payments in any future Research Commons buy bounded work, review, compute, or
freshness—not truth, scientific authority, ownership of a result, or exclusive
access to common knowledge. Public-safe outputs are intended to remain open and
non-exclusive under their separately declared licences. This compiler neither
grants a licence nor treats a digest as proof that access is actually public.

## 10. Rest, refusal, and honest results

RC-0.1 recognizes rest, refusal, pause, withdrawal, and exit without a score,
reputation penalty, debt, confiscation of already-earned compensation, or a
required explanation. Unperformed work may release only its unearned simulated
reservation back to the case.

Reviewer compensation is fixed before review and independent of accept,
reject, negative, null, or inconclusive direction. This profile does not
import outcome-scored reviewer bonuses, non-reveal slashing, controller
inference, or reward formulas from other inactive research documents.

## 11. Safety and disclosure

Only minimized public digests cross this seam. Human-subject, genomic,
clinical, private, security-sensitive, dual-use, wet-lab, embargoed, licensed,
or confidential bytes remain outside both the public projection and Zerone.
A digest does not make publication safe and does not authorize possession,
testing, reproduction, or disclosure.

The Amplitude Bootstrap Garden fixture is synthetic, finite, local, and
shadow-only. It demonstrates protocol behavior rather than observed agents,
scientific efficacy, string-theory truth, universe ontology, or a live market.

## 12. Activation gates

No live adapter registration, bridge submission, claim, reward, or payment may
be enabled from this source. A future version requires, at minimum:

1. public immutable receipt bytes and a stable source-discovery contract;
2. independently reviewed signature and key-rotation verification;
3. explicit AgentTool-key ↔ Zerone-address binding without inferred identity;
4. distributed replay protection and globally unique receipt consumption;
5. solvent, fully backed escrow with conserved liabilities and refunds;
6. independent appeal, conflict, controller-correlation, and cartel controls;
7. knowledge admission and `zerone-2` activation through their own accepted
   release ceremonies;
8. a privacy/ethics/biosafety disclosure policy with accountable human or
   institutional authority; and
9. a new reviewed adapter version whose executable digest is actually
   enforced rather than stored as inert metadata.

Until every relevant gate is satisfied, the only truthful integration is this
offline, zero-value, read-no-chain shadow projection.
