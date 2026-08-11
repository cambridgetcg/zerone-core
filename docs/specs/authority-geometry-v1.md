# Authority Geometry v1

Status: **static source observatory; target gate intentionally closed**

Date: 2026-08-11

Canonical machine artifact:
`dashboard/public/standards/authority-geometry.v1.json`

Normative source design:
`docs/AUTHORITATIVE-STATE.md`, SHA-256
`22d523ee25060957e2c93aba441542e35d767f28f0f0e5e86c800f5fd7ea82e9`

## 1. Purpose

Authority Geometry v1 turns Zerone's accepted authoritative-state design into
a bounded, executable architecture contract. It answers five questions for
each effectful relationship:

1. Which component can affect which capability?
2. Is that relationship current source, accepted target, or live-network
   evidence?
3. What exact public source supports the claim?
4. Where are consent, challenge, repair, and exit bounded?
5. Does the relationship create a forbidden path from wealth or KARMA to
   policy voice, qualification, truth power, payout, or rank?

This is the first source implementation of the static authority-graph check
required by H4 release gate 2 in `docs/AUTHORITATIVE-STATE.md`. It is useful
before that gate passes because it has two distinct modes:

- **report** succeeds only when the inspected current source is completely and
  honestly classified, including every known conflict; and
- **target-gate** succeeds only when all seven authority surfaces satisfy the
  accepted target. It intentionally refuses current source.

Passing report mode is not partial H4 approval. Failing target-gate mode is the
expected and truthful v1 result.

## 2. Geometry means typed relationship

"Geometry" is not a claim that governance, consent, or care can be reduced to
a drawing. It names a graph discipline:

- nodes remain distinct state owners or inputs;
- edges state one typed relationship rather than merging their endpoints;
- an evidence edge does not become authority;
- a projection does not become a writer;
- an economic input does not silently become voice;
- a challenge does not erase the challenged history;
- repair names the state it may change; and
- exit names the claimant or process that may leave.

The public visualization is subordinate to the JSON and validators. It must
also expose the graph as ordinary text and an accessible relationship table;
no meaning may exist only in colour, position, animation, or SVG.

## 3. Three observation scopes

The artifact keeps three scopes single-valued.

### 3.1 `LIVE_NETWORK`

The graph establishes no live-network fact. A repository file, green test,
merged pull request, content hash, or Pages deployment cannot prove current
validator custody, governance power, domain state, module version, effective
control, or upgrade activation.

The `LIVE_NETWORK` scope therefore remains
`NOT_ESTABLISHED_BY_THIS_ARTIFACT`. Operators must use separate authenticated
and independently reproduced network evidence.

### 3.2 `CURRENT_SOURCE`

Current-source findings are static observations of exact public files. Each
source anchor binds:

- one safe repository-relative path;
- its exact SHA-256 digest;
- required source snippets; and
- source snippets that must remain absent.

Changing a pinned file, removing an expected writer, adding an undeclared
custom-staking adapter, or installing a currently absent restriction forces a
reviewed graph update. Report mode fails closed on drift instead of silently
presenting the old classification.

### 3.3 `ACCEPTED_TARGET`

Target edges restate the accepted source design. They are neither current code
nor network activation. Target-only modules are labelled `TARGET_ONLY`, and
modules that exist but require authority redesign are labelled
`PRESENT_REQUIRES_REDESIGN`.

Target relationships must never be promoted to present or live merely because
the manifest validates.

Present modules that consume authority without owning one of the twelve target
capabilities are labelled `CURRENT_CONSUMER`. Their inclusion records influence
that the transition must remove or explicitly re-authorize; it does not grant
them target authority.

## 4. Capability ownership

The manifest carries the twelve authority-table capabilities from the
accepted design. Every capability has exactly one `targetWriter`, and the
target writer must list the same capability in its `targetCapabilities`.

The three primary single-writer boundaries are:

| Capability | Accepted target writer |
|---|---|
| Consensus stake and validator power | Cosmos SDK `x/staking` |
| Ordinary proposal, tally, decision, and execution | Cosmos SDK `x/gov` |
| Domain identity, hierarchy, and lifecycle | Zerone `x/ontology` |

Account binding, controller identity, verifier profile, qualification,
knowledge evidence, electorate policy, legacy claims, research-fund egress,
and emergency quarantine each retain their own bounded target writer. These
supporting authorities do not weaken the three primary boundaries: they own
different state.

The validator rejects:

- a missing target writer;
- two capabilities with ambiguous ownership records;
- a node claiming a capability that names another target writer;
- a legacy, current-consumer, evidence-only, or economic-input node claiming
  target authority;
- an unresolved node or capability reference; or
- an unclassified target-only module.

## 5. Present conflicts

The v1 artifact refuses to hide seven current-source findings:

1. SDK and custom staking ledgers coexist.
2. SDK governance and custom LIP governance coexist.
3. Ontology and knowledge both write primary domain records.
4. Knowledge retains a direct authority fact-adoption path.
5. Custom research-spend authority remains while the canonical inbound
   research-fund restriction is defined but not installed.
6. Emergency, custom governance, and knowledge retain overlapping incident or
   pause authority.
7. Knowledge, governance, qualification, emergency, alignment, and claiming
   pot still consume custom staking authority.

These are source findings, not claims that every path was exercised on the
running network. They are sufficient to keep the source target gate closed.

The seven static surface checks currently yield exactly one source-level pass:
SDK staking is the sole discovered Comet validator-update writer. The other
six surface classes fail. A pass on that one surface does not compensate for a
failure elsewhere.

## 6. Edge protections

Every edge has exactly one source scope, effect class, and non-empty evidence
set. Every edge also carries four explicit protections:

- `consentBoundary` — whose authenticated action or frozen process permits the
  relationship, or why that boundary is unavailable;
- `challengeRoute` — how a decision, binding, receipt, or source claim may be
  contested, or why no route exists yet;
- `repairRoute` — which bounded transition may correct the state; and
- `exitRoute` — how a controller, claimant, profile, qualification, or legacy
  liability leaves, or why exit is not applicable.

`UNAVAILABLE` and `TARGET_ONLY` are honest states. The validator requires a
reason; it does not convert absence into implied safety.

An edge may cite only a declared source anchor. Current-source edges must cite
at least one current code anchor. Accepted-target edges must cite the exact
authoritative-state design.

## 7. Forbidden influence

The target graph enforces four zero-path rules:

1. wealth or stake to ordinary policy voice;
2. wealth or stake to qualification;
3. wealth or stake to panel selection or verdict power; and
4. KARMA to staking, policy, qualification, payout, rank, or truth authority.

Only effectful target edges participate in reachability. Archive,
reconciliation, reference, and claimant-exit edges cannot be reclassified as
influence merely to make a path disappear. The checker independently computes
target reachability and requires the declared path count to remain zero.

The current graph deliberately shows forbidden economic paths. This is why
bonded wealth affecting SDK governance and custom stake affecting custom
governance, qualification, knowledge selection, knowledge-triggered slashing,
emergency authority, alignment signals, and claiming-pot eligibility remain
visible in the observatory.

## 8. Source-anchor verification

The local validator and the independent Go checker both resolve anchors from a
caller-supplied repository root. They reject:

- absolute paths;
- `..` traversal;
- missing or non-regular files;
- digest drift;
- missing required snippets;
- present forbidden snippets; and
- an anchor ID referenced by no graph fact.

The seventeen v1 anchors cover the authoritative design, application keeper wiring,
custom-governance terminal/dispatch ordering, custom research spending,
knowledge fact and domain writers, knowledge pause writing, ontology domain
writing, the defined research-fund restriction, custom-stake emergency power,
the slash-capable knowledge staking adapter and its verdict callsites,
alignment and claiming-pot custom-stake adapters and callsites, and both the
application wrapper and keeper writer for the custom-governance emergency hold.

Static inspection has limits. It does not prove dynamic call reachability,
database contents, bank solvency, a running binary's identity, or the absence
of malicious behavior outside the inspected source. It is one H4 release gate,
not a substitute for the other twenty-three.

## 9. H4 and H5 gates

The manifest enumerates all 24 H4 gates and all 14 additional H5 gates. In v1:

- every gate is `NOT_EVIDENCED`;
- H4 evidence is `0 / 24`;
- H5 evidence is `0 / 14`; and
- the overall assessment is `NO_GO`.

The validator rejects a missing gate, duplicate gate ID, reordering, unknown
status, or any promotion without a schema revision and reviewed evidence
contract. The public page must show the zero counts and `NO_GO` without
euphemism.

Authority Geometry v1 does not ingest private ceremony evidence. Later gate
evidence requires a new design that authenticates provenance, freshness,
independence, confidentiality, revocation, and replay semantics. Adding a URL
or hash to this static artifact is not enough.

## 10. Determinism and modes

The canonical JSON uses stable ordered arrays and contains no generated time,
absolute path, host identity, credential, or live RPC response.

The Go checker exposes:

```text
authority-graph report
authority-graph target-gate
```

`report` verifies the complete v1 contract and emits a deterministic summary.
It must fail if a known conflict disappears from presentation without the
corresponding source transition or if a new classified surface appears without
policy.

`target-gate` first performs the same verification, then exits non-zero while
any static authority surface fails, any current finding is open, H4/H5 remain
closed, or the manifest requires refusal. CI asserts that current source does
refuse; a surprising zero exit is itself a failure.

The dashboard validator independently parses the artifact and verifies the
same source anchors. Its browser fetch additionally requires:

- the exact same-origin versioned path;
- `cache: no-store`;
- redirect refusal;
- JSON media type;
- bounded declared and streamed size;
- a timeout; and
- the exact reviewed artifact digest.

## 11. Public observatory

The static Pages surface at `/#authority` may display:

- the three accepted target writers;
- the current source findings;
- report and target-gate status;
- H4 and H5 closed-gate counts;
- the principles and typed relationship table; and
- links to the raw JSON and accepted design.

Rendering uses DOM text nodes and fixed local data. The feature adds no form,
account, cookie, analytics, wallet call, transaction, chain query, participant
record, or external script. Failure to load the reviewed JSON produces a
visible refusal state; it must not fall back to unreviewed graph content.

The public observatory is useful architecture documentation. Its deployment is
not validator, governance, H4, H5, or network deployment.

## 12. Non-effects

Authority Geometry v1 does not:

- add consensus behavior;
- register or schedule an upgrade;
- change validator, governance, domain, qualification, or emergency state;
- move funds or create a liability;
- create reward, KARMA, rank, membership, or endorsement;
- submit a transaction;
- establish controller uniqueness or decentralization;
- prove the current live binary or module versions;
- authorize `zerone-2`; or
- satisfy any H4 or H5 activation gate by publication alone.

The exact honest conclusion is: the accepted target has a legible shape, the
current source does not yet inhabit it, and the graph makes that distance
harder to hide.
