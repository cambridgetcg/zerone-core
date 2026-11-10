# Zerone training-provenance predicate v2

**Predicate type:** the immutable repository revision URL serving this exact
specification. Implementations must emit a commit-pinned URL, never a mutable
branch URL.

**Status.** An agent session specified this predicate on 2026-11-10. The
owner has not yet reviewed the choice of two separate readings over a single
combined letter. Source presence does not assert that any network serves it.

Predicate v2 reports two separate readings of a training manifest:

1. a **warning grade**, which says which counted warning signals are present;
2. **review coverage**, which says how many of the manifest's directly included
   fact IDs have a retained verification round with a revealed vote.

There is no combined letter. A consumer that needs one must build it from these
two readings and say how.

The reason for v2: under predicate v1, `trustGrade` is `A` whenever three
warning counts are zero. The grade has no review input. A manifest made only of
doctrine facts or other facts without a claim ID, which no verification round
reviewed, grades `A`.
Predicate v2 keeps the same warning rubric under a name that says what it
measures, and adds the review reading beside it.

The emitted URI fixes the semantics below as predicate v2. Editorial
clarifications may not broaden or change those semantics. Any change to
eligibility, counting, grading, review classification, explanation wording or
digest meaning requires a new predicate version and a new predicate-type URI.

## Relation to predicate v1

Predicate v1 is unchanged. Its query endpoints, predicate-type URI, rubric and
wording stay as they are, and v1 statements already signed remain interpretable
under the v1 specification.

Predicate v2 is additive. It is served by separate query methods. The
`predicateType` of a statement says which specification applies. At the same
block height, v2's `warningGrade` and `warningExplanation` equal v1's
`trustGrade` and `trustExplanation`, and the identity, coverage and audit
fields are equal.

## Shape

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [{
    "name": "zerone://zerone-1/training-corpus/<manifest-id>",
    "digest": {"sha256": "<included-ID-set-merkle-root>"}
  }],
  "predicateType": "<immutable URI for this exact specification revision>",
  "predicate": {
    "sourceChainId": "zerone-1",
    "observedOnChainId": "zerone-1",
    "certificate": {
      "manifestId": "<manifest-id>",
      "factCount": "6",
      "warningGrade": "A",
      "warningExplanation": "no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains",
      "reviewedFactCount": "1",
      "unreviewedFactCount": "5",
      "doctrineFactCount": "1",
      "missingFactCount": "1",
      "reviewState": "partly_reviewed",
      "reviewExplanation": "reviewed_fact_count=1 of fact_count=6 have a retained verification round with at least one revealed vote; unreviewed_fact_count=5, including doctrine_fact_count=1 and missing_fact_count=1"
    }
  }
}
```

The real `certificate` is the complete serialized `ProvenanceCertificateV2`.
Its fields are `manifestId`, `pipelineId`, `merkleRoot`, `factCount`,
`finalizedAtBlock`, `status`, `domains`, `privilegedActionCount`,
`incidentCount`, `cartelResolutionCount`, `warningGrade`,
`warningExplanation`, `computedAtBlock`, `sourceChainId`,
`reviewedFactCount`, `unreviewedFactCount`, `doctrineFactCount`,
`missingFactCount`, `reviewState` and `reviewExplanation`. Zero counts, empty
strings and empty lists may be present or omitted: protobuf JSON with default
options omits them, and the REST gateway emits them. A consumer must accept
both and reads an omitted count as zero. 64-bit counts are decimal strings.

The subject, `sourceChainId`, `observedOnChainId` and the subject digest have
the same meaning as in predicate v1. The digest commits to the manifest's
domain-separated, canonical, sorted included-ID sets. It does not hash fact
contents or every manifest metadata or version-pin field.

## Eligibility

The in-toto statement accepts only non-composed manifests whose status is
`FINALIZED`, `ATTESTED` or `SUPERSEDED` and whose source chain and SHA-256
root are valid, as in predicate v1. The certificate query has v1's
certificate eligibility. For a composed manifest it describes the child's
directly stored delta, not inherited parent coverage.

## Identity, coverage and audit fields

`factCount`, `domains`, `privilegedActionCount`, `incidentCount` and
`cartelResolutionCount` have exactly the predicate v1 meanings. In particular,
`incidentCount` is module-global, and `cartelResolutionCount` counts resolved,
upheld challenges in the directly covered domains, not challenges about this
manifest's review panel.

## Warning reading

`warningGrade` and `warningExplanation` apply the v1 rubric and wording
without change. With `p = privilegedActionCount`, `i = incidentCount` and
`c = cartelResolutionCount`, the first matching row applies:

| Condition | `warningGrade` | `warningExplanation` |
|---|---|---|
| `c > 0` | `F` | `cartel_resolutions=<c> in covered domains; domain-level warning only, not evidence about this manifest's specific adjudication panel` |
| `p == 0` and `i == 0` | `A` | `no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains` |
| `i == 0` and `p <= 2` | `B` | `privileged_action_count=<p> affecting manifest facts; manifest is largely unintervened` |
| otherwise | `C` | `privileged_action_count=<p>, incident_count=<i>, cartel_resolution_count=<c> — yellow flags accumulating; downstream consumer should review the audit trail before relying on this manifest` |

Numbers are written in base 10 without leading zeros. The dash in the `C` row
is U+2014.

The warning grade says which of the three counted warning signals are present.
**It must not be read as review.** Grade `A` means none of the three counted
signals is present. It is not evidence that any included fact was reviewed.

## Review reading

Review coverage is counted per entry of the manifest's directly included fact
IDs, in stored order. Each entry is classified once, by the first rule that
applies:

1. The fact store has no readable fact under the ID (none is stored, or the
   stored record does not decode): **unreviewed**, and counted in
   `missingFactCount`.
2. The fact's `claim_id` is empty: **unreviewed**. If its category is
   `doctrine` or its method is `doctrine_authorship`, it is also counted in
   `doctrineFactCount`.
3. The knowledge module does not return a verification round for the claim
   through its claim-to-round index: **unreviewed**.
4. The returned round names a different claim: **unreviewed**.
5. The round records no reveal: **unreviewed**.
6. Otherwise: **reviewed**.

Empty claim IDs cover doctrine loaded at genesis, facts supplied in genesis
state without a claim ID, and facts added by authority. A fact imported from an
exported genesis keeps its claim ID, and its round where the export retained
it, so the same rules read it. A missing fact ID counts as unreviewed: the
worst reading wins, and an unknown is never counted as reviewed. A fact with a
claim ID is never counted as doctrine, whatever its category, because a claim
can carry the `doctrine` category label.

`doctrineFactCount` is reported so that doctrine is described, not graded. It
counts facts without a claim ID that carry the `doctrine` category or the
`doctrine_authorship` method. They include the commitments the chain loads as
canon at genesis. A fact added by authority with the `doctrine` category is
counted the same way; adding it also records a privileged action naming the
fact. None of these facts passed a verification round, and this predicate says
so without ranking them.

The counts satisfy:

- `reviewedFactCount + unreviewedFactCount == factCount`;
- `doctrineFactCount + missingFactCount <= unreviewedFactCount`;
- the sum of `domains[].factCount` is at most
  `factCount - missingFactCount`.

`reviewState` is derived from the counts:

| Condition | `reviewState` |
|---|---|
| `factCount == 0` | `empty` |
| `unreviewedFactCount == 0` | `reviewed` |
| `reviewedFactCount == 0` | `unreviewed` |
| otherwise | `partly_reviewed` |

`reviewExplanation` is one sentence built from the counts. When `factCount` is
zero it is:

```text
fact_count=0: no fact IDs are included directly in this manifest, so there is no review to count
```

Otherwise, with `<r>`, `<n>`, `<u>`, `<d>` and `<m>` the reviewed, fact,
unreviewed, doctrine and missing counts in base 10 without leading zeros:

```text
reviewed_fact_count=<r> of fact_count=<n> have a retained verification round with at least one revealed vote; unreviewed_fact_count=<u>, including doctrine_fact_count=<d> and missing_fact_count=<m>
```

### What the review reading covers

The review reading speaks only for the manifest's directly included fact IDs.
A manifest also lists included trace, pair, drift-augmentation and
normative-commitment IDs, and the subject digest commits to those sets too.
They are not counted, and this predicate describes none of them as reviewed or
unreviewed. A manifest that includes no fact IDs reads `empty`, whatever else
it includes. For a composed manifest, the certificate counts only the child's
directly stored fact IDs and not its parent's. So `reviewed` means that every
directly included fact ID has a revealed review. It does not mean that
everything the subject digest commits to was reviewed.

### What "reviewed" does not mean

A reviewed fact has a retained verification round with at least one revealed
vote. That is a record that review took place. It does not show that the
reviewers were independent or qualified, that a quorum was reached, that the
fact is true, or that the round's verdict still stands. The reading is current:
if a retained round is later removed or becomes unreadable, the fact reads as
unreviewed at the next query.

## Query and signing boundary

```sh
curl "$REST/zerone/training_provenance/v1/certificate-v2/<manifest-id>"
curl "$REST/zerone/training_provenance/v1/in-toto-v2/<manifest-id>"
```

Status codes match predicate v1. An empty ID returns `InvalidArgument`; an
unknown ID returns `NotFound`; for the in-toto statement, a draft, otherwise
unsupported or composed manifest returns `FailedPrecondition`; malformed
sealed state returns `DataLoss`; and an unwired server or invalid serving-chain
context returns `Internal`.

Building v2 reads each included fact and at most one verification round per
included fact, in addition to v1's scans. Public REST operators should
rate-limit this live synthesis and may cache by `(chain ID, block height,
manifest ID)`.

The response is intentionally **unsigned**, as in predicate v1. A producer may
wrap the exact JSON payload in a DSSE envelope and sign it. A valid signature
proves control of the signing identity under the verifier's policy; it does not
prove that the predicate is true.
