# Zerone training-provenance predicate v1

**Predicate type:** `https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/attestations/training-provenance-v1.md`

Zerone exposes a flat, sealed training manifest's current
`x/training_provenance` certificate as an unsigned
[in-toto Statement v1][statement]. This is an exact standards projection of
that query response; it is not a projection of the complete manifest or corpus,
and it does not add state, mint rewards, or make a new truth claim.

This URI fixes the semantics below as predicate v1. Editorial clarifications
may not broaden or change those semantics. Any change to manifest eligibility,
coverage, counting, grading, or digest meaning requires a new predicate version
and a new predicate-type URI so existing signed statements remain
interpretable.

## Shape

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [{
    "name": "zerone://zerone-1/training-corpus/<manifest-id>",
    "digest": {"sha256": "<included-ID-set-merkle-root>"}
  }],
  "predicateType": "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/attestations/training-provenance-v1.md",
  "predicate": {
    "sourceChainId": "zerone-1",
    "observedOnChainId": "zerone-1",
    "certificate": {"manifestId": "<manifest-id>"}
  }
}
```

The real `certificate` contains the complete serialized
`ProvenanceCertificate`, including domain coverage, audit counts, the current
trust grade, and the block at which the projection was computed.
`sourceChainId` is the chain pinned when the manifest was created;
`observedOnChainId` is the node serving the current projection. They remain
distinct across exports and relaunches.

The subject digest commits to the manifest's domain-separated, canonical,
sorted included-ID sets. It does **not** hash fact contents or every manifest
metadata/version-pin field. The full manifest remains the authoritative source
for those pins.

## Eligibility and exact v1 semantics

Predicate v1 accepts only non-composed manifests whose status is `FINALIZED`,
`ATTESTED`, or `SUPERSEDED` and whose source chain and SHA-256 root are valid.
It deliberately refuses composed manifests because the current certificate
synthesizer reads the child's direct delta and does not recursively materialize
parent coverage. A future recursive projection requires a new predicate
version.

Certificate fields have these precise v1 meanings:

- `factCount` is the number of fact IDs included directly in this manifest.
- `domains` groups the directly included facts that are still queryable and
  have a nonempty domain. Qualification counts and weights are current values
  for those directly covered domains.
- `privilegedActionCount` counts records whose target exactly matches one of
  the directly included fact IDs. It is not a domain- or pipeline-wide count.
- `incidentCount` is module-global: it counts every incident whose
  `affectedModules` contains `knowledge`, not only incidents attributable to
  this manifest, pipeline, or domain.
- `cartelResolutionCount` counts resolved, upheld challenges in the directly
  covered domains.
- `trustGrade` and `trustExplanation` are recomputed from those current counts;
  they are not historical facts frozen at finalization.

## Query and signing boundary

```sh
curl "$REST/zerone/training_provenance/v1/in-toto/<manifest-id>"
```

An empty ID returns `InvalidArgument`; an unknown ID returns `NotFound`; a
draft, otherwise unsupported, or composed manifest returns
`FailedPrecondition`; malformed sealed state returns `DataLoss`; and an
unwired server or invalid serving-chain context returns `Internal`. None is
misreported as absent.

The response is intentionally **unsigned**. A producer may wrap the exact JSON
payload in a DSSE envelope and sign it with Sigstore. Verifiers must pin their
trusted root and apply an explicit certificate-identity and artifact-digest
policy. A valid signature proves control of the signing identity under that
policy; it does not prove that the predicate is true. Zerone's underlying
manifest root and audit signals remain independently queryable.

[statement]: https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md
