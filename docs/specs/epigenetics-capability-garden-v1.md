# Epigenetics Capability Garden v1

Status: experimental static standard; non-authoritative; non-network-observed;
non-reward-bearing.

Canonical public document:
`dashboard/public/standards/epigenetics-capability-garden.v1.json`

Reviewed SHA-256:
`7d04efe9da46309bf97b850c9b80324b1a5c4035edb1008b9ba3ad0df2bcfa63`

## 1. Release boundary

This release publishes an inspectable capability graph and an inactive reward
template. It does not:

- change consensus, genesis, module state, network parameters, or a validator;
- activate a reward, mint, treasury payment, escrow, qualification, or governing
  right;
- move funds or identify a recipient;
- authorize a biological experiment, security test, human intervention, or
  heritable genome intervention;
- publish raw identifiable genomic, medical, or controlled-access data; or
- assert that a node, study, biomarker, intervention, or product is clinically
  valid.

Every release-boundary field in the JSON is exactly `false`. The offline and
browser parsers reject the whole document if any boundary changes.

The existing Constructive Intelligence Tree v1 remains byte-for-byte unchanged.
Its zero-value constructive-receipt bridge continues to pin SHA-256
`8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf`.
Epigenetics is a separate versioned track so scientific iteration cannot rewrite
cryptographic evidence receipts retroactively.

## 2. Why a garden

A conventional skill tree suggests a single trunk, a permanent unlock, and a
linear climb. Experimental life science is less tidy:

- evidence depends on organism, tissue, cell state, developmental window,
  environment, assay, cohort, and analysis lineage;
- biological and technical failure modes recur at every layer;
- correction, contradiction, null results, and supersession are productive;
- a capability can decay when a protocol, reference, cohort, instrument, or
  understanding changes; and
- branches should produce new questions and independent descendants, not a
  permanent status attached to a person.

The v1 garden therefore has two roots, seven growth stages, 25 capabilities, 58
prerequisite edges, and three bounded quests. Nodes describe evidence work, not
human worth.

## 3. Growth stages

| Stage | Function |
| --- | --- |
| `soil` | Consent, data governance, preregistration, provenance, statistics, and reproducibility. |
| `trunk` | Cell identity, gene regulation, chromatin, and methylation foundations. |
| `measurement` | Assay-specific protocols and controls for molecular, single-cell, and spatial evidence. |
| `inference` | Confounding, causal graphs, cell-state trajectories, and multi-omic integration. |
| `intervention` | Targeted epigenome perturbation, rescue, reversal, specificity, and containment. |
| `canopy` | Independent replication, orthogonal challenge, prior-art delta, and descendant value. |
| `quest` | Bounded breakthrough candidates that require the preceding roots and remain template-only. |

The graph is acyclic, reaches every node from the two declared roots, has maximum
depth ten and maximum direct fan-out seven, and refuses prerequisites that point
backward across stages.

`prerequisites` is the sorted union of structural graph edges. The default rule
requires every listed edge. Four reviewed nodes instead carry an exact
`prerequisiteSemantics` overlay: all entries in `allOf` plus the declared
`atLeast.count` members of `atLeast.of`. This makes “any one” a threshold of one
and prevents broad capabilities from silently requiring one assay stack. The
parser requires each overlay to partition the flat edge list exactly and forbids
alternative quest gates.

V1 uses threshold rules only for batch/confounding, multi-omic integration,
single-cell multi-omic measurement, and target-layer epigenome editing. Generic
replication and orthogonal challenge remain claim-specific rather than hard-coded
to a modality. Orthogonal challenge requires at least two assays with
demonstrably nonshared physical principles and failure modes.

## 4. Evidence ladder

Evidence level and lifecycle state are orthogonal. `E4` does not mean “true”; a
claim can be supported, contested, corrected, disproved, or superseded at any
level.

A node's `evidenceContribution` says which ladder transition its artifacts can
help a bounded claim approach. It never says that the node, artifact, or person
has attained that level. The cumulative, claim-scoped evidence review remains
separate. The prior-art/descendant node is marked `cross-cutting` because it
spans the frozen E0 comparison and later descendant value rather than behaving
like one scalar grade.

| Level | Name | Inactive sponsor template | Minimum meaning |
| --- | --- | ---: | --- |
| E0 | question frozen | 0% | Question, scope, prior-art search, outcomes, exclusions, and safety lane frozen before results. |
| E1 | inspectable | 0% | Protocol, lineage, permitted-use scope, code, and immutable artifact pointers inspectable. |
| E2 | assay controlled | 15% | Assay-specific controls and internal reproduction survive. |
| E3 | independently reproduced | 25% | A separately rooted effective cluster reproduces the bounded claim. |
| E4 | causally challenged | 25% | Perturbation, orthogonal measurement, rescue, or frozen triangulation survives a serious alternative. |
| E5 | prospectively generalised | 15% | A frozen prediction generalises to new context or useful descendant work. |
| E6 | maintained impact | 5% | The result remains useful through maintenance, challenge, correction, and descendants. |
| — | challenge reserve | 15% | Counterevidence, replication, correction, and unresolved challenge. |

The percentages sum to 10,000 basis points. They specify how a future,
separately funded, voluntary sponsor escrow could be partitioned. They are not an
amount, currency, entitlement, price, protocol issuance rule, treasury mandate,
or active escrow.

Skill unlocks create no reward. Time alone unlocks no evidence. Protocol issuance
is disabled. Ordinary nodes are `capability-evidence-only`: they describe work
artifacts and grant no qualification. The three quest nodes are marked
`sponsor-template-only`, which still grants no reward.

Every quest also freezes a digest-bounded E0/E3/E4/E5 acceptance template. That
still cannot create a funded case. Before any future sponsor money can exist, a
separate `zerone.breakthrough-sponsor-case/v1` must bind the claim and artifact
digests, objective milestone tests, null/failure/contradiction treatment,
reviewer conflicts, correction and recovery, expiry and refund, and proof that
escrow was funded. The dashboard is structurally unable to activate such a case.

## 5. Derived breakthrough rule

An author, sponsor, operator, token holder, majority, citation count, model score,
or dashboard cannot select a breakthrough. E4 can establish a causally challenged
candidate, but breakthrough recognition requires E5. It emerges only when all of
these coexist:

1. at least E4 causally challenged evidence;
2. a frozen, inspectable prior-art delta;
3. effective independent reproduction;
4. a serious causal or counter-hypothesis challenge; and
5. a preserved E5 receipt for that prospective or descendant result.

Novelty alone is insufficient. Powered preregistered null results remain
direction-neutral evidence and may earn separately scoped replication or
challenge funding under a future sponsor agreement. Attention, publication
venue, token transfer, address count, and elapsed time are not substitutes.

## 6. Independence

The v1 minimum is three effective clusters with at least two organisation roots,
two data roots, and two analysis-pipeline roots. Assignment occurs after artifact
freeze where practical, and review compensation must not depend on reaching the
preferred outcome.

An effective cluster is a control and lineage question, not a wallet count. Shared
people, funding, management, samples, preparation, instruments, code, models,
cloud images, decision authority, or undisclosed coordination can collapse many
nominal actors into one cluster.

These floors are necessary but not sufficient. A real funded quest must freeze a
claim-specific independence matrix and controller caps before accepting results.

## 7. Assay-specific evidence

The garden deliberately has no universal “epigenetics quality” scalar. ENCODE
publishes different experimental and data-quality expectations for different
assay classes. Each node therefore carries its own artifacts and failure modes;
passing one assay's threshold cannot certify another assay or a biological
mechanism.

The provenance model keeps donor, biosample, preparation, experiment, file,
analysis step, model, figure, and bounded claim distinct. This follows the shape
of the 4DN data model and prevents a repository page, latent embedding, or result
table from standing in for lineage.

Related primary and official references in v1 include:

- [ENCODE data standards](https://www.encodeproject.org/data-standards/)
- [4DN data model](https://data.4dnucleome.org/help/user-guide/data-model)
- [scNMT-seq](https://doi.org/10.1038/s41467-018-03149-4)
- [CRISPRoff](https://doi.org/10.1016/j.cell.2021.03.025)
- [Two-step epigenetic Mendelian randomization](https://doi.org/10.1093/ije/dyr233)
- [NIH preclinical rigor guidance](https://grants.nih.gov/policy-and-compliance/policy-topics/reproducibility/principles-guidelines-reporting-preclinical-research)
- [NIH rigor-of-prior-research guidance](https://grants.nih.gov/policy-and-compliance/policy-topics/reproducibility/guidance)
- [WIPO PCT novelty guidance](https://www.wipo.int/en/web/pct-system/texts/ispe/12_01_02)
- [DunedinPACE](https://doi.org/10.7554/eLife.73420)
- [Epigenetic-clock reliability analysis](https://doi.org/10.1038/s43587-022-00248-2)
- [Epigenetic-clock demographic and tissue context](https://doi.org/10.1186/s13059-016-1030-0)
- [scPair bimodal single-cell analysis](https://doi.org/10.1038/s41467-024-53971-2)
- [HHS Common Rule](https://www.hhs.gov/ohrp/regulations-and-policy/regulations/common-rule/index.html)
- [NIH biosafety and biosecurity policy](https://osp.od.nih.gov/policies/biosafety-and-biosecurity-policy/)
- [OLAW PHS animal-welfare policy](https://olaw.nih.gov/node/74)
- [WHO Laboratory Biosafety Manual](https://www.who.int/publications/i/item/9789240011311)

Each JSON source states a narrow `supportScope`. References constrain examples
and evidence shape; they do not become universal standards, certify the node as
complete, or imply that an experiment reproduced a cited paper. The WIPO novelty
comparison is borrowed only as an element-by-element prior-art discipline and
makes no legal conclusion about patentability, priority, inventorship, or
ownership.

## 8. Human genomic data and safety

Raw identifiable human data never belongs in the public garden, an artifact
receipt, or chain state. Public evidence is limited to non-identifying metadata,
digests, permitted-use scope, access-lane attestations, aggregate claims, and
links whose publication is independently permitted.

For human genomic material, consent or permitted data-use scope and an explicit
access-lane attestation are mandatory. Controlled access is required when
consent, policy, or law requires it; unrestricted access is possible only when
consent, policy, and the applicable legal basis permit it. Institutional
certification and ethical or regulatory review remain required where applicable.
The policy is
informed by the [NIH Genomic Data Sharing Policy](https://sharing.nih.gov/genomic-data-sharing-policy/about-genomic-data-sharing),
[controlled-access best practices](https://sharing.nih.gov/sites/default/files/flmngr/NIH_Best_Practices_for_Controlled-Access_Data_Subject_to_the_NIH_GDS_Policy.pdf),
and [institutional certification guidance](https://sharing.nih.gov/genomic-data-sharing-policy/institutional-certifications/about-institutional-certifications).
Applicable human-participant review, risk-based biosafety review, containment,
and animal-welfare review remain distinct gates; approval in one lane cannot be
laundered into approval in another.

The garden cannot grant consent, access, ethical approval, clinical validity, or
permission to intervene. Unknown harm escalates to the regulated-human lane.
Unapproved human intervention and heritable human genome intervention are
ineligible by construction. This garden also cannot grant IRB, IBC, IACUC,
regulatory, or equivalent jurisdiction-specific approval.

## 9. The three v1 quests

### Transportable ageing signal

Carry a frozen signal across external cohort, tissue, ancestry, cell composition,
assay, longitudinal, and intervention-aware checks without converting age
prediction into a causal or clinical claim.

### Causal regulatory locus

Carry one fixed locus from prior-art delta and bounded association through
orthogonal assay, targeted perturbation, rescue, independent reproduction, and
useful descendant work.

### Writable and reversible cell-fate memory

Define memory, write, read, time horizon, state transition, erasure or rescue,
and context boundary; then separate persistence from selection, toxicity,
composition, clonal expansion, and irreversible genomic alteration.

## 10. KARMA relationship

Quest evidence may eventually emit contextual recognition edges. Those edges are
not balances, ranks, property, truth, rewards, or votes. The companion
[KARMA foundation](../KARMA.md) freezes that separation and all governance gates
closed.

## 11. Validation and versioning

From `dashboard/`:

```bash
npm run check:life
npm test
```

The validator is offline. It checks exact keys, enums, dates, HTTPS source URLs,
sorted and unique identifiers, graph resolution, reachability, cycles, stage
direction, exact threshold partitions, non-alternative quest gates, bounded
quest acceptance and canonical scope hashes, hard safety and economics literals,
graph bounds, duplicate JSON keys, reviewed shape, and exact SHA-256 pins.

The browser independently fetches both static standards with a same-origin
versioned path, no-store caching, same-origin credentials, redirect refusal,
JSON media-type check, declared and streamed size limits, a live-stream deadline,
strict UTF-8 decoding, exact SHA-256 verification, and fail-closed parsing.

V1 is immutable once public receipts or external work cite its digest. A changed
node, edge, source snapshot, policy, or evidence rule requires a new versioned
document and reviewed runtime pin.
