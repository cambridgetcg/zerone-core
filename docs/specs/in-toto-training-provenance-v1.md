# Zerone in-toto training-provenance predicate v1

- Implementation: query-side projection
- Predicate type: `https://zerone.money/attestations/training-provenance/v1`
- Statement type: `https://in-toto.io/Statement/v1`
- Normative local copy: this file

The predicate TypeURI is a stable, versioned identifier. in-toto permits an
unresolvable TypeURI, but recommends that it resolve to a human-readable
definition. The `zerone.money` route is not published yet; public envelope
issuance should wait until it serves this v1 definition. Breaking changes must
use a new predicate URI.

## Purpose

This predicate projects Zerone's live `x/training_provenance`
`ProvenanceCertificate` into an
[in-toto Statement v1](https://github.com/in-toto/attestation/blob/v1.2.0/spec/v1/statement.md).
It lets model builders, auditors, and artifact registries consume a Zerone
training-manifest commitment without learning Zerone's protobuf schema.

The projection is read-only. It creates no store entry, changes no consensus
root, and does not alter the native certificate. It creates a Statement layer,
not a signed in-toto Envelope.

## Statement mapping

The Statement has exactly one subject:

| in-toto field | Zerone source |
|---|---|
| `subject[0].name` | `training-manifest/<manifest_id>` |
| `subject[0].digest.sha256` | `ProvenanceCertificate.merkle_root` |
| `subject[0].annotations.sourceChain` | CAIP-2 Cosmos identifier derived from the node-reported chain ID |
| `predicateType` | Zerone predicate URI above |
| `predicate` | Mapping defined below |

The subject digest is Zerone's domain-separated, lowercase, hex-encoded SHA-256
training-manifest commitment. It commits to selected identifier sets, not
arbitrary model bytes, a packaged dataset, or necessarily the content behind
every identifier. Composed child manifests retain their existing recursive
commitment semantics.

A finalized root is intended to remain fixed, but Zerone has an incident-bound
root-correction path. A corrected root is a distinct in-toto subject digest.
Consumers must retain the source height and review correction history rather
than treating `subject.name` alone as immutable.

## Predicate shape

```json
{
  "source": {
    "chain": "cosmos:zerone-2",
    "computedAtBlock": "731",
    "module": "training_provenance"
  },
  "manifest": {
    "factCount": "22",
    "finalizedAtBlock": "700",
    "id": "manifest-7",
    "pipelineId": "pipeline-3",
    "status": "finalized"
  },
  "domainCoverage": [
    {
      "activeVoterCount": "3",
      "avgQualifiedWeight": "800000",
      "domain": "sciences",
      "factCount": "12"
    }
  ],
  "trust": {
    "explanation": "no privileged actions ...",
    "grade": "A",
    "signals": {
      "coveredDomainCartelResolutionCount": "0",
      "includedFactPrivilegedActionCount": "0",
      "knowledgeModuleIncidentCount": "0"
    }
  }
}
```

Every field shown above is required. Additional fields are not defined by v1.
The generation rules are:

- every unsigned integer is a decimal string matching
  `^(0|[1-9][0-9]*)$`;
- `manifest.status` is one of `finalized`, `attested`, or `superseded`;
- draft and unspecified manifests are not exportable;
- `trust.grade` is one of `A`, `B`, `C`, or `F`;
- the subject `sha256` digest is exactly 64 lowercase hexadecimal characters;
- each `domainCoverage.domain` is non-empty and unique; and
- `domainCoverage` is ordered lexicographically by `domain`, with the numeric
  fields used as deterministic tie-breakers.

Integer strings avoid precision loss because `google.protobuf.Struct`
represents JSON numbers as `float64`. Domain ordering normalizes equivalent
certificate inputs, but the emitted JSON is not a cross-version canonical
serialization. A DSSE verifier must verify the payload bytes carried by the
Envelope and must not reserialize them for signature verification.

The signal names deliberately expose the native synthesizer's current scope:

- `includedFactPrivilegedActionCount` counts privileged actions whose `target`
  exactly equals an included fact ID. It does not count every domain-, module-,
  or manifest-level privileged action.
- `knowledgeModuleIncidentCount` counts all incidents that list `knowledge` in
  `affected_modules`; it is module-wide, not manifest-specific.
- `coveredDomainCartelResolutionCount` counts upheld cartel resolutions in a
  domain covered by the manifest; it is domain-wide.

Qualification coverage, all three signal counts, the explanation, and the
grade are live at `source.computedAtBlock`.

## CAIP-2 source label

`source.chain` follows [CAIP-2](https://standards.chainagnostic.org/CAIPs/caip-2).
The core CAIP is Final; the
[Cosmos namespace profile](https://namespaces.chainagnostic.org/cosmos/caip2)
is Draft.

The CLI reads the chain ID reported by the same node connection used for the
certificate query. IDs matching `[-a-zA-Z0-9]{1,32}` are represented directly
unless they begin with reserved prefix `hashed-`. Other IDs use the Cosmos
profile's `hashed-` plus the first 16 lowercase hex characters of
`SHA-256(UTF-8(chain_id))`.

CAIP-2 labels the network claimed by the queried node and lets consumers reject
an unexpected source. It does not cryptographically bind the subject digest to
that network or establish RPC honesty. Origin assurance requires a trusted
query or state proof plus, where authentication is needed, a signed Envelope
and signer policy.

## Creation

The node CLI exposes the native certificate and the standard projection:

```sh
zeroned query training_provenance certificate manifest-7 \
  --node tcp://127.0.0.1:26657

zeroned query training_provenance in-toto-statement manifest-7 \
  --node tcp://127.0.0.1:26657 > statement.json
```

The native query is available over gRPC/ABCI. A public REST route is not mounted
in this change: certificate synthesis scans manifest facts and global audit
streams, so HTTP exposure needs indexing, scale tests, and operator rate
limits first.

Go consumers can build the projection directly:

```go
statement, err := intoto.BuildStatement("zerone-2", certificate)
if err != nil {
    return err
}
payload, err := intoto.MarshalStatementJSON(statement)
```

The implementation uses the official
[`github.com/in-toto/attestation/go/v1`](https://pkg.go.dev/github.com/in-toto/attestation/go/v1)
types. Zerone adds stricter checks for exact Statement v1, a lowercase SHA-256
digest, exportable manifest status, supported trust grade, and domain
uniqueness. The upstream `Statement.Validate()` provides basic structure and
known-digest encoding/length checks; it does not recompute the manifest root,
validate this predicate, prove chain inclusion, or authenticate the Statement.

## Authentication and verification

The CLI output is an **unsigned Statement**, not an authenticated attestation.
For DSSE, use the emitted JSON bytes as `payload`, set `payloadType` to
`application/vnd.in-toto+json`, and sign DSSE's pre-authentication encoding
`PAE(payloadType, payload)`. The
[in-toto Envelope specification](https://github.com/in-toto/attestation/blob/v1.2.0/spec/v1/envelope.md)
defines the requirements. A signature proves control of the signing key and
integrity of the authenticated payload; it does not prove the manifest is true,
safe, available, or suitable for a model.

A relying party should:

1. if an Envelope is present, verify its signature over DSSE PAE before parsing
   the payload;
2. parse the authenticated payload, or the explicitly unauthenticated raw
   object, as exact in-toto Statement v1;
3. require this exact `predicateType` and validate the predicate rules above;
4. require an expected CAIP-2 source chain;
5. obtain native data from a trusted Zerone node or verify a suitable state
   proof at `source.computedAtBlock`;
6. re-derive the training-manifest commitment from native manifest data; and
7. evaluate the correction history, live audit counts, and trust grade under
   its own policy.

The certificate and grade are live snapshots. A later block may contain new
incidents, privileged actions, qualifications, or capture resolutions and
therefore produce a different predicate for the same manifest digest.
