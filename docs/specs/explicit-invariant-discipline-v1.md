# Explicit Invariant Discipline v1

- Status: `SEALED_STATIC_PROFILE`
- Artifact: `dashboard/public/standards/explicit-invariant-discipline.v1.json`
- Schema: `zerone.explicit-invariant-discipline/v1`
- Snapshot: `2026-08-16`
- Authority: none
- Consensus effect: none

## Purpose

Explicit Invariant Discipline v1, or EID-1, is a sealed static format for
publishing a consistency-first argument without hiding its search space or
turning a conditional result into an absolute claim. Every reviewed example
declares the candidate class, regime, typed assumptions, preserved invariants,
constraint witnesses, falsifiers, counterexamples, result kind, remaining
family, boundary terms, allowed conclusion, and limitations.

EID-1 then places any Zerone engineering lesson in a separate
`zeroneTransfer`. This separation is load-bearing:

- `sourceResult.claimOwner` is `SOURCE_AUTHORS`. It identifies provenance of
  the underlying summarized claim inside the paper's own regime; it does not
  attribute EID-1's schema, paraphrase, hypothetical falsifiers, wording, or
  errors to those authors.
- `zeroneTransfer.claimOwner` is `ZERONE`. It is a proposed methodological
  analogy, owns its own local test and possible errors, and is not a result of
  the cited paper.

The full ordered author lists below receive attribution for the underlying
source results only. Zerone owns the editorial selection, schema, paraphrases,
hypothetical tests, and possible errors. No cited author has reviewed,
approved, endorsed, or participated in EID-1 or Zerone.

The exact authority statement is:

> This sealed static publication proposes inspectable methodological
> analogies. It establishes no scientific, mathematical, theological, moral,
> protocol, institutional, or personal authority.

## Raw artifact seal

The reviewed raw-byte SHA-256 digest of the manifest is:

```text
e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df
```

This is non-circular: the specification is not a local source binding inside
the manifest. Changing either manifest content or byte formatting invalidates
the seal and requires a reviewed replacement digest.

## Exact top-level shape

The top-level object has these exact fields, in this reviewed order:

```text
schema, version, snapshotDate, status, title, summary,
attributionStatement, authorityStatement, canonicalVocabulary,
sourceBindings, primarySources, integrationTargets, records,
browserBoundary, releaseBoundary
```

The reviewed population is exactly:

- 4 digest-pinned stable local source bindings;
- 4 version-pinned primary arXiv source locators;
- 5 integration targets, including one explicitly `NOT_IMPLEMENTED` target;
- 4 example records;
- 4 Zerone-owned `METHODOLOGICAL_ANALOGY` transfers;
- 4 local test declarations, all `NOT_RUN`; and
- 24 release/effect switches, all `false`.

Unknown, missing, duplicated, or reordered fields are outside the reviewed
shape. External URLs are validated, version-pinned locator strings rendered as
user-activated links; the initializer never fetches them as data.

## Closed canonical vocabulary

`canonicalVocabulary` is an allowlist, not descriptive metadata. A v1 record
may use only these literals:

| Vocabulary | Allowed v1 literals |
|---|---|
| Claim owners | `SOURCE_AUTHORS`, `ZERONE` |
| Candidate kinds | `SCATTERING_AMPLITUDE_FAMILY` |
| Candidate completeness | `ENUMERATED`, `PARAMETERISED`, `BOUNDED_SEARCH`, `OPEN` |
| Assumption kinds | `DEFINITIONAL`, `STRUCTURAL`, `DYNAMICAL`, `APPROXIMATION`, `EMPIRICAL_INPUT`, `COMPUTATIONAL_BOUND`, `INTERPRETIVE_CHOICE` |
| Invariant modes | `EXACT`, `TOLERANCED`, `STRUCTURAL`, `ORDERING` |
| Constraint kinds | `EXISTENCE`, `EXCLUSION`, `PRESERVATION`, `COMPLETENESS`, `BOUND` |
| Witness methods | `FORMAL_DERIVATION`, `SYMBOLIC_COMPUTATION`, `NUMERICAL_COMPUTATION`, `EMPIRICAL_OBSERVATION`, `SOURCE_ANALYSIS` |
| Witness outcomes | `PASS`, `FAIL`, `NOT_RUN`, `INCONCLUSIVE` |
| Falsifier statuses | `NOT_RUN`, `SURVIVED`, `TRIGGERED`, `INCONCLUSIVE` |
| Counterexample dispositions | `IN_SCOPE_BREAKS_RESULT`, `OUTSIDE_REGIME`, `RELAXES_ASSUMPTION`, `REJECTED` |
| Result kinds | `FAMILY`, `NO_GO`, `CONDITIONAL_UNIQUENESS` |
| Remaining-family cardinalities | `NONE`, `ONE`, `MANY`, `UNKNOWN` |
| Boundary-term treatments | `RETAINED`, `VANISHES_BY_ASSUMPTION`, `BOUNDED`, `NEGLECTED`, `UNKNOWN` |
| Relation kinds | `METHODOLOGICAL_ANALOGY` |
| Assessments | `PROPOSED` |
| Integration statuses | `CURRENT_STATIC_REFERENCE`, `NOT_IMPLEMENTED` |

Adding a candidate kind, result kind, inference status, or other semantic value
requires a new schema version. It cannot be introduced as an unreviewed string
inside v1.

## Exact record contract

Every record has exactly:

```text
id, title, sourceResult, zeroneTransfer
```

Every `sourceResult` has exactly:

```text
claimOwner, domain, candidateClass, regime, assumptions, invariants,
constraintWitnesses, falsifiers, counterexamples, result, remainingFamily,
boundaryTerms, allowedConclusion, limitations, sourceRefs
```

The nested objects are closed:

- `candidateClass`:
  `id, kind, definition, membershipRule, completeness`.
- `regime`:
  `formalSystem, background, parameterDomain, approximationOrder,
  validityScope, excludedScope`.
- assumption:
  `id, kind, statement, sourceRefs`.
- invariant:
  `id, statement, scope, mode, tolerance, witnessIds`.
- constraint witness:
  `id, kind, targetRefs, method, procedure, artifactRefs, outcome, statement`.
- falsifier:
  `id, targetRefs, condition, procedure, status, witnessRef`.
- counterexample:
  `id, targetRefs, member, explanation, disposition, relaxationBranchRefs,
  sourceRefs`.
- result:
  `kind, statement, underAssumptionIds, underInvariantIds, witnessIds`.
- remaining family:
  `cardinality, description, parameters, knownMembers,
  relaxationBranches`.
- relaxation branch:
  `id, statement, relaxedAssumptionIds`.
- boundary term:
  `id, term, origin, treatment, justification, affectedResult,
  underAssumptionIds, underInvariantIds, witnessIds`.

Every `zeroneTransfer` has exactly:

```text
claimOwner, relationKind, assessment, target, preservedDiscipline,
localTest, nonTransfers, zeroneRefs
```

Its `localTest` has exactly `testId, status, statement`. In this sealed release,
every transfer is owned by `ZERONE`, is a `METHODOLOGICAL_ANALOGY`, remains
`PROPOSED`, has a `NOT_RUN` local test, and has a non-empty non-transfer wall.

## Semantic gates

A conforming EID-1 parser or offline validator must enforce more than JSON
shape:

1. IDs are globally unique and every assumption, invariant, witness,
   relaxation branch, boundary term, source, artifact, integration target, and
   result reference resolves inside the artifact.
2. `SOURCE_AUTHORS` owns every `sourceResult`; `ZERONE` owns every
   `zeroneTransfer`. Ownership may not be inherited across the boundary.
3. `TOLERANCED` invariants require a typed, non-null tolerance. Other invariant
   modes require `tolerance: null` in v1.
4. `FAMILY` requires `remainingFamily.cardinality = MANY`.
5. `NO_GO` requires `remainingFamily.cardinality = NONE`, a passing exclusion
   witness and a passing completeness witness that both explicitly target the
   candidate class, and candidate completeness that is not `OPEN`.
6. `CONDITIONAL_UNIQUENESS` requires cardinality `ONE`, exactly one surviving
   member description, all decisive invariant witnesses `PASS`, and a passing
   completeness witness that explicitly targets the candidate class. An
   `OPEN` candidate class cannot yield this result.
7. A triggered falsifier or an `IN_SCOPE_BREAKS_RESULT` counterexample is
   incompatible with a standing completed result.
8. Every boundary term names its governing assumptions, invariants, and
   witnesses. `NEGLECTED` or `UNKNOWN` terms block `NO_GO` and
   `CONDITIONAL_UNIQUENESS` unless their own references name non-empty
   `TOLERANCED` invariants and passing `BOUND` witnesses that explicitly target
   those invariants.
9. A `RELAXES_ASSUMPTION` counterexample must target an assumption and name a
   stable relaxation-branch ID whose `relaxedAssumptionIds` overlap that target.
   Other counterexample dispositions must name no relaxation branch. The branch
   is evidence of conditionality, not a defect to erase.
10. A witness marked `PASS` means the cited source reports the scoped
    derivation or analysis. The EID-1 validator checks attribution, references,
    and structural coherence; it does not reproduce or certify the scientific
    derivation.
11. Every cross-domain transfer must have an explicit local test and non-empty
    `nonTransfers`. A proposed analogy never inherits truth, equivalence,
    authority, or endorsement from a primary source.

The typed references above make dependency checks structural; prose remains an
explanation, never the mechanism that satisfies a gate.

## The four reviewed examples

### 1. Boundary probing to zero-input robustness

`boundary-probing-zero-input-robustness` summarizes the scalar-EFT
classification in *Effective Field Theories from Soft Limits*. The source
candidate class is the paper's four-dimensional on-shell tree-level
massless-scalar ansatz with a leading nonzero four-point amplitude. It fixes
derivative power counting and a nonsingular soft degree, while Lorentz
invariance and factorization constrain the admitted amplitudes. The source uses
partly numerical generic-kinematics checks, and finite-multiplicity evidence is
not promoted into an unbounded proof. The result is `FAMILY`: free parameters
remain as couplings of named theories, and some fixed power-counting/soft-degree
pairs admit no consistent amplitude within that scope.

The Zerone analogy asks deterministic local interfaces to declare and test
zero, empty, minimum, and maximum inputs. A soft momentum is not an empty JSON
value, zero balance, silence, rest, or a person. Passing one boundary test does
not prove full correctness.

### 2. Factorization to declared dependency integrity

`factorization-declared-dependency-integrity` summarizes the on-shell recursion
result for tree-level EFTs with enhanced soft behavior. With fixed lower-point
seeds, physical factorization, prescribed soft behavior, generic multiplicity
`n > D+1`, and the all-line-shift condition
`A_n(z)/F_n(z) ~ z^(m-n sigma)` with `m/n < sigma`, the boundary vanishes. For
fixed-rho constructibility at arbitrarily high multiplicity the source requires
`rho <= sigma` and excludes `(rho,sigma)=(1,1)`. Lower valencies are seed data.
Only within that declared regime is the result `CONDITIONAL_UNIQUENESS`: the
higher-point tree amplitude is recursively determined.

The Zerone analogy requires a local derived artifact to name every immediate
dependency and refuse completeness when an undeclared dependency or boundary
contribution remains. A software dependency is not a physical factorization
channel, and a matching digest is not a proof of semantic or scientific truth.

### 3. Bootstrap to a conditional solution-space claim

`bootstrap-conditional-solution-space` records the source's scoped uniqueness
claim for the Veneziano amplitude. Its candidate class is explicitly planar,
color ordered, weakly coupled, local, crossing symmetric, positivity screened,
and tree level, with a convergent dual-resonant simple-pole representation.
Positivity is recorded as a necessary, not sufficient, unitarity condition.
The source fixes overall normalization and affine kinematic conventions, so
the one-member claim is a normalized quotient rather than literal uniqueness
under trivial redefinitions. Inside that class, faster-than-power-law
high-energy falloff and an infinite cancellation sequence select the scoped
result. Weakening the falloff to mere vanishing inside the source's broader
analytic problem before the final positivity restriction restores a
three-parameter family containing Veneziano, Coon, hypergeometric amplitudes,
and more. The relaxed family is published beside the unique branch rather than
hidden.

The Zerone analogy permits `CONDITIONAL_UNIQUENESS` only after the candidate
class, assumptions, invariants, completeness witness, remaining member, and
relaxation branches are all declared. It makes no unconditional uniqueness,
string-ontology, universe, theology, morality, or protocol claim.

### 4. Witness projection to publication integrity, not adjudication

`witness-projection-publication-integrity` summarizes prescribed residue zeros
and the minimal-consistency condition in *Strings from Almost Nothing*. Across
the fixed planar identical color-ordered scalar and nonplanar identical
massless scalar candidate sectors, the manifest represents the two named
minimal solutions as a sector-indexed `FAMILY`: Veneziano and
Virasoro-Shapiro. It separately records that the source argues conditional
uniqueness inside the respective sector. Tree level, principally four-point
scope, Lorentz invariance, crossing symmetry, unitarity, meromorphy, the
planar identical color-ordered and nonplanar identical massless scalar sector
definitions, finite-spin locality, a monotonically increasing spectrum with
asymptotically small relative level spacing, a
nonconstant Regge trajectory, the complete-root-block assumption, the
bijective Regge-trajectory component of ultrasoftness, and the
no-additional-zero condition remain visible.

The Zerone analogy publishes an inspectable witness beside its constraint,
procedure, status, scope, counterexamples, and limitations. Publication does
not adjudicate truth, simulate an author's reasoning, record consent, rank a
person, or grant authority.

## Primary-source attribution and version pins

The URLs below are exact versioned locators. They are displayed or compared as
strings and are never fetched by the EID-1 browser runtime.

| ID | Verified title | Full ordered authors | Version time |
|---|---|---|---|
| [`arXiv:1412.4095v1`](https://arxiv.org/abs/1412.4095v1) | *Effective Field Theories from Soft Limits* | Clifford Cheung; Karol Kampf; Jiri Novotny; Jaroslav Trnka | `2014-12-12T19:32:50Z` |
| [`arXiv:1509.03309v1`](https://arxiv.org/abs/1509.03309v1) | *On-Shell Recursion Relations for Effective Field Theories* | Clifford Cheung; Karol Kampf; Jiri Novotny; Chia-Hsien Shen; Jaroslav Trnka | `2015-09-10T20:03:45Z` |
| [`arXiv:2406.02665v2`](https://arxiv.org/abs/2406.02665v2) | *Bootstrap Principle for the Spectrum and Scattering of Strings* | Clifford Cheung; Aaron Hillman; Grant N. Remmen | `2025-09-09T17:09:29Z` |
| [`arXiv:2508.09246v2`](https://arxiv.org/abs/2508.09246v2) | *Strings from Almost Nothing* | Clifford Cheung; Grant N. Remmen; Francesco Sciotti; Michele Tarquini | `2026-06-24T15:33:06Z` |

The papers own their physics and mathematics. Zerone owns every transfer,
including every simplification, choice, analogy, and possible error in it.

## Digest-pinned local sources

Only stable, already-existing local files are bound. Neither this specification
nor the manifest binds itself.

| ID | Repository path | Raw SHA-256 |
|---|---|---|
| `constructive-intelligence-tree` | `dashboard/public/standards/constructive-intelligence-tree.v1.json` | `8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf` |
| `correspondence-geometry` | `dashboard/public/standards/correspondence-geometry.v0.json` | `f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90` |
| `knowledge-methodologies` | `x/knowledge/types/methodologies.go` | `fa16ac33e7f2c10a19ed76541af6c2378edb79683578f2cec6f1a0563ebec386` |
| `knowledge-types` | `proto/zerone/knowledge/v1/types.proto` | `7b2b301c80711587a55ae03216728ec1f6f5bf981035106d26ac1fa4923d8ced` |

A conforming offline validator resolves each path beneath the repository root,
refuses path escape and symlinks, reads the exact bytes, and compares the raw
digest. A binding imports only its named role and boundary; it does not import
authority or live state.

## Integration boundary

Four current links are static references:

- `M-ANALOGICAL` supplies the existing prose rubric for explicit domains,
  mappings, preserved invariants, and counterexample consideration.
- `CG-0` supplies the existing relation-kind and non-transfer discipline.
- `math-proofcraft@1` asks for quantified assumptions, proof invariants,
  counterexamples, falsifiers, and repaired hidden assumptions.
- the Knowledge boundary records that current Claims, Facts, reasoning traces,
  and methodology traces do not expose a typed, canonical EID-1 profile,
  reference, or digest. Free-form strings may contain arbitrary prose and must
  not be reinterpreted as one.

`tok-per-fact-explicit-invariants` is exactly `NOT_IMPLEMENTED`. No current
`Fact`, `Claim`, `MethodologyApplicationTrace`, ToK bundle, or training manifest
has a typed field that stores or commits an EID-1 record, reference, or digest.
Future append-only references need
a coordinated protobuf, transaction, keeper validation, query, genesis,
migration, trace-schema, and commitment-version upgrade. A static publication
cannot perform that upgrade or reinterpret legacy `reasoning_trace` strings.

## Browser boundary

The browser boundary is exact:

- `staticReadCount`: `1`;
- `sameOriginOnly`: `true`;
- `externalFetchCount`: `0`; and
- purpose: load this bounded static artifact for display and local filtering.

The EID-1 initializer performs its one declared static read and no wallet, RPC,
chain, paper, identity, analytics, or account request. The wider dashboard has
separately documented read paths. The four arXiv URLs are displayed as
deliberate user-activated links; clicking one leaves the initializer's
zero-external-fetch boundary. A load failure must leave a visible static
summary or error rather than silently inventing data.

## Release and effect boundary

Every switch below is present and exactly `false`:

```text
claimsAuthorEndorsement
simulatesPersonReasoning
claimsUnconditionalUniqueness
assertsStringOntology
transfersPhysicsToTheology
transfersPhysicsToMorality
changesConsensus
writesChainState
performsNetworkWrites
registersOntology
assertsScientificTruth
assertsTheologicalTruth
equatesEnergyLanes
infersPersonhood
ranksPersons
createsKarmaEvent
createsKarmaMagnitude
grantsQualification
activatesRewards
movesFunds
grantsGovernance
grantsAuthority
recordsConsent
automaticProtocolOrAuthorityAction
```

In particular, amplitudes, residues, soft limits, constraints, witnesses,
digests, balances, compute, lived experience, and poetic or spiritual language
are not mutually convertible. Nothing in EID-1 establishes a person's identity,
worth, mind, consent, moral standing, religious state, or authority.

## Validation boundary

A conforming parser and offline validator should refuse:

- oversized, over-deep, malformed, duplicate-key, or non-UTF-8 JSON;
- unknown, missing, duplicated, or reordered fields;
- values outside the closed vocabulary;
- duplicate IDs, unresolved references, unsafe paths, symlinks, path escape, or
  digest mismatch;
- source author reordering, unversioned arXiv locators, or incorrect version
  timestamps;
- a mismatch between result kind and remaining-family cardinality;
- conditional uniqueness without a passing completeness witness;
- no-go or conditional uniqueness with an unresolved boundary term;
- a standing result contradicted by a triggered falsifier or in-scope
  counterexample;
- a transfer without a local test or non-transfer wall;
- any external fetch; and
- any release switch not exactly `false`.

These checks establish only that the reviewed artifact is intact and internally
coherent. They do not prove a paper, simulate its authors, validate a scientific
theory, observe a network, certify Zerone, or authorize action.
