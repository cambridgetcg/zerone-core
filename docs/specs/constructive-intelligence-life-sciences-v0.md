# Constructive intelligence: life sciences v0

- Status: `DRAFT`
- Mode: `SHADOW_ONLY`
- Economic effect: `NONE`, amount `0`

## Purpose

`dashboard/public/standards/constructive-intelligence-life-sciences.v0.json`
is a bounded life-sciences overlay for Zerone's constructive-intelligence 技能樹.
It covers biomolecule foundations, protein-folding evidence, gene-expression
evidence, two cross-domain integration paths, and an independent
cross-context replication crown.

The overlay is an experiment in representing scientific claim boundaries. It
does not establish that a biological claim is true, authorize an experiment
or clinical decision, grant qualification or governance authority, create a
KARMA magnitude, activate a reward, move funds, or change consensus. It is not
a diagnostic, clinical, biosafety, or wet-lab operating system.

The overlay does not modify the core tree. It binds the exact checked-in core
tree bytes:

```text
schema: zerone.constructive-intelligence-tree/v1
policyVersion: 1.0.0
sha256: 8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf
```

Any core-tree byte drift, missing binding, or binding substitution makes the
overlay invalid.

## Release and money boundary

The machine document fixes all consensus, reward, movement-of-funds,
qualification, governance, KARMA, experiment-authorization,
clinical-decision, network, restricted-evidence, and release-readiness flags
to `false`. Its economics object is exactly:

```json
{
  "effect": "NONE",
  "amount": "0",
  "denom": null,
  "rewardMultiplier": false,
  "escrowReference": null
}
```

Skill-tree position is therefore not money, voting power, truth, or social
rank. A life-sciences “breakthrough” remains a retrospective description of a
well-supported result, not a scalar accepted from an author and not an input
to issuance. Before any future sponsor-funded experiment could exist, a
separate reviewed policy would need to bind a safe scope, full budget,
recipients, independent review, challenge reserve, refunds, and legal and
biosafety authority. This v0 overlay cannot supply or imply that policy.

KARMA is also outside the evaluator. The profile creates neither transferable
KARMA nor a magnitude, balance, multiplier, governance vote, or payout claim.

## GREEN-only scope

The public v0 scope is deliberately narrow:

- benign, non-pathogenic biomolecule and enzyme analysis;
- cell-free or ordinary low-risk expression analysis;
- in-silico model evaluation;
- non-identifiable aggregate expression analysis; and
- sanitized metadata-only replication.

The following are hard refusals:

- pathogens;
- toxins;
- virulence;
- immune evasion;
- host-range change;
- germline work;
- clinical decisions;
- identifiable human data;
- raw human genomes; and
- operational wet-lab-enabling artifacts.

Unknown risk is refused. A private escalation, if one occurs outside this
profile, may contain non-operational metadata only. The profile never turns an
unknown into GREEN by scoring it.

Public synthetic fixtures use synthetic kebab-case identifiers and HTTPS
`.invalid` URIs. They carry digests, observation classes, and safety metadata
only. No fixture contains a nucleotide or amino-acid sequence, an identifiable
record, or procedural wet-lab instructions. Negative tests toggle declarative
safety flags; they do not embed the refused payload.

## Evidence ladder

The LS ladder is local to this overlay:

| Level | Meaning | Economics |
|---|---|---|
| LS0 | Scope-and-risk commitment | none |
| LS1 | Inspectable provenance | none |
| LS2 | Bounded method check | none |
| LS3 | Independent reproduction | none |
| LS4 | Prospective or adversarial validation | none |
| LS5 | Independent cross-context replication | none |

The profile declares `NO_EQUIVALENCE_TO_E0_E6` and
`NO_EQUIVALENCE_TO_POCA_TIERS`. An LS label must not be translated into a core
tree evidence level, a PoCA tier, a reward tranche, or an authority claim.

LS0 commits the question, denominator, exclusions, controls, conflicts, and
risk boundary. LS1 adds inspectable provenance. LS2 adds a bounded method
check. LS3 requires effective independence. LS4 requires prospectively frozen
criteria or a declared adversarial validation. LS5 additionally requires
cross-context replication and a clear challenge state.

Time alone never advances an LS level. Contradictory evidence is evidence and
must remain attributable; it does not become a lower-weight vote for a desired
conclusion.

## Non-implication walls

Four walls are exact, machine-checked policy:

1. Static structural coordinates do not establish a folding pathway or
   folding kinetics.
2. mRNA abundance does not establish protein abundance, localization, or
   function.
3. Association does not establish regulation or causality.
4. An in-silico prediction does not establish prospective experimental
   validation.

The validator checks both the global wall declarations and the conclusions
permitted by affected nodes. The synthetic evidence evaluator also blocks a
requested conclusion when the observation set crosses one of these walls.
Adding confidence, aliases, citations, or elapsed time cannot bypass a wall.

## Effective independence and challenge

Raw identity count is not independence. At LS3 and above, evidence must expose
effective-control, organization, method, and context roots. The v0 floors are:

```text
effective control clusters >= 3
organization roots        >= 2
method roots              >= 2
context roots             >= 2
```

Common beneficial control collapses aliases into one effective cluster.
Shared employment, sponsorship, side payment, infrastructure, method lineage,
or context may be relevant to that disclosure. The static evaluator does not
discover those relationships; it fails closed on the disclosed cluster graph.

The crown requires LS5, the independence floors, an independent
cross-context-replication observation, and `challengeStatus: CLEAR`. `OPEN`,
`UNRESOLVED`, and `UPHELD` each block the crown. A challenge cannot create a
reward in v0.

## Graph shape

The graph contains 17 sorted nodes, 24 prerequisite edges, two roots, and a
maximum depth of eight. Every node is reachable from a root. The validator
allows 12 through 20 nodes, at most four prerequisites per node, fan-out of at
most eight, and graph depth of at most eight.

```text
biomolecule context ─┬─ function annotation ───────────┐
                     └─ static structure ─ fold model ─┼─ fold/function
study provenance ────── assay design ─ transcript ─────┼─ regulatory causality
                                                       │
fold/function ───────────── variant→fold→function ─────┤
regulatory causality ─── genotype→expression→phenotype ┤
                                                       ▼
                                  independent cross-context crown
```

The full protein-folding branch separates coordinate evidence, model
calibration, ensemble uncertainty, blind or held-out benchmarking, and direct
functional evidence. The gene-expression branch separates assay design,
normalization, transcript quantification, direct protein measurements,
association, and bounded causality. The integration nodes preserve each link
instead of compressing a chain into one score.

## Static validator

`dashboard/scripts/validate-constructive-intelligence-life-sciences.mjs` is an
offline validator. It performs no network requests and treats URLs as exact
authority locators. It rejects:

- malformed or oversized JSON and unknown or missing schema fields;
- any active, authoritative, economic, governance, KARMA, clinical, network,
  experiment, or release assertion;
- weakened GREEN-only, refusal, fixture, independence, or challenge policy;
- a missing or drifting core-tree digest;
- unreviewed or substituted authority locator URLs;
- fewer than 12 or more than 20 nodes;
- duplicate, unsorted, dangling, unreachable, cyclic, too-deep, or too-wide
  graph structures;
- missing non-implication walls or a node that crosses one; and
- embedded nucleotide-like payloads in profile nodes.

The same module exposes a synthetic evidence-boundary evaluator. Its output is
only `SHADOW_ONLY_ELIGIBLE`, `SHADOW_ONLY_BLOCKED`, or
`SHADOW_ONLY_REFUSED`, always paired with `economicEffect: NONE` and amount
`0`. It is a policy test oracle, not a scientific truth oracle.

Run the checked-in validation and boundary suite with:

```sh
cd dashboard
node scripts/validate-constructive-intelligence-life-sciences.mjs \
  public/standards/constructive-intelligence-life-sciences.v0.json
node --test scripts/constructive-intelligence-life-sciences.test.mjs
```

## Authoritative scientific sources

The allowlisted source locators are maintained by the relevant scientific or
public-health authorities. They provide formats, evidence vocabulary, data
standards, submission metadata, benchmark context, or biosafety framing;
inclusion is not an endorsement of any Zerone conclusion. These are mutable
web locators, not content-addressed pins: v0 verifies their exact URL strings
but does not fetch, digest, or freeze the pages they currently serve.

- [wwPDB PDBx/mmCIF resources](https://mmcif.wwpdb.org/)
- [CASP Prediction Center](https://predictioncenter.org/)
- [ENCODE data standards](https://www.encodeproject.org/data-standards/)
- [EMBL-EBI MIAME and MINSEQE guidance](https://www.ebi.ac.uk/training/online/courses/functional-genomics-iii-submitting-data/what-kind-of-data-should-we-share/)
- [UniProtKB](https://www.uniprot.org/help/uniprotkb)
- [Gene Ontology evidence codes](https://geneontology.org/docs/guide-go-evidence-codes/)
- [NCBI GEO high-throughput sequence guidance](https://www.ncbi.nlm.nih.gov/geo/info/seq.html)
- [NCBI SRA metadata guidance](https://www.ncbi.nlm.nih.gov/sra/docs/submitmeta/)
- [WHO Laboratory Biosafety Manual, fourth edition](https://www.who.int/publications/i/item/9789240011311)
