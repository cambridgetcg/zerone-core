# Adapter specification: `sigstore-in-toto-v1`

Status: experimental and unregistered — promotion requires a successful
production-policy end-to-end cryptographic fixture

## 1. Purpose and boundary

This adapter anchors a cryptographically verified Sigstore in-toto attestation
as a witness-only `x/substrate_bridge` source. It records that a particular
identity signed a particular DSSE payload under a pinned trust policy.

It does not interpret the predicate as truth, create knowledge claims, cite
facts, project recursion weight, or grant an economic reward. Predicate-specific
meaning belongs in a separately governed adapter version.

## 2. Fixed compiler

- Adapter ID: `sigstore-in-toto-v1`
- Implementation:
  `tools/sigstore-substrate-compiler`
- Runtime network access: none
- Sigstore SDK: `github.com/sigstore/sigstore-go` v1.2.2
- Tool module: Go 1.25.8, isolated from the Go 1.24 validator module

Version 1.2.2 is intentionally pinned: v1.2.1 includes the upstream
GHSA-wqqc-jjcq-vfxm remediation and v1.2.2 tightens empty certificate-identity
criteria. Any SDK, policy, or output change requires a new compiler binary hash
and governance review; a semantic change should use a new adapter ID.

## 3. Required invocation policy

Every run supplies:

1. a local Sigstore bundle JSON path;
2. a local, reviewed trusted-root JSON path;
3. one exact certificate issuer;
4. one exact certificate SAN;
5. one artifact digest in canonical
   `sha256:<64 lowercase hex>` form;
6. one exact predicate-type URI;
7. one public HTTPS source URL with no userinfo, query, or fragment; and
8. optionally, the Zerone block height at which the source was fetched.

Regex identity matching, remote trusted-root discovery, TUF updates, network
lookups, current-time certificate validation, key-only verification, missing
artifact policies, and missing identity policies are not exposed.

Governance must record the full approved invocation policy (including the
SHA-256 of the trusted-root file) outside the link. These policy values are not
fields in the current `SubstrateLink` protobuf.

## 4. Verification algorithm

The compiler must:

1. read bounded, regular local bundle and trusted-root files;
2. parse a Sigstore bundle with version at least v0.3;
3. require a DSSE envelope with payload type
   `application/vnd.in-toto+json`;
4. create the verifier from only the supplied local trusted root;
5. require at least one verified SCT, one transparency-log entry, and one
   observer timestamp;
6. enforce the exact issuer and SAN with no regex matcher;
7. bind the signature to the required SHA-256 artifact digest through the
   in-toto subjects;
8. require every Statement subject to contain at least one non-empty digest
   algorithm and value;
9. require `_type == "https://in-toto.io/Statement/v1"`;
10. require exact equality with the configured `predicateType`;
11. return an opaque verified-attestation value that keeps the exact accepted
    bundle bytes paired with the exact decoded, verified DSSE payload; and
12. pass only that opaque value to the public compiler API.

The compiler does not validate the internal schema or assertions of the
predicate beyond its exact type URI.

## 5. Deterministic output

Let `P` be the exact decoded DSSE payload byte string, let `B` be the exact raw
bundle JSON byte string accepted by the verifier, and:

```text
HP = SHA-256(P)
HB = SHA-256(B)
```

The output is:

```text
adapter_id      = "sigstore-in-toto-v1"
cited_facts     = []
pending_claims  = []
recursion_weight = nil
source.adapter_id       = "sigstore-in-toto-v1"
source.source_id        = "sha256:" + lowercase_hex(HP)
source.source_url       = required audit URL
source.content_hash     = HB
source.fetched_at_block = caller value, default 0
link_hash               = keeper.ComputeLinkHash(link)
```

No JSON normalization occurs before hashing. A whitespace-only bundle change
therefore changes `content_hash` and `link_hash`. A payload-byte change also
changes `source_id`, even when parsed JSON semantics are otherwise equivalent.

The current chain canonical hash does not include `source_url`; it is
unauthenticated audit metadata and can be changed without changing
`link_hash`. It must therefore never carry credentials or drive a trust
decision. A challenger may fetch it as a convenience, but accepts the returned
bytes only when their SHA-256 matches `source.content_hash`, then re-verifies
the payload identity and policy. Governance remains responsible for immutable
bundle retention independently of this locator.

## 6. Future governance registration template

This adapter is experimental and must remain unregistered until a
production-policy end-to-end fixture cryptographically verifies with Zerone's
selected trusted root, exact identity, artifact digest, and predicate policy.
After that gate and governance review, any initial registration must explicitly
declare zero axis bounds and zero witness reward:

```json
{
  "adapter_id": "sigstore-in-toto-v1",
  "source_type": "sigstore_in_toto_dsse",
  "version": "1.0.0",
  "compiler_binary_hash": "<base64 SHA-256 of the reproducible compiler binary>",
  "axis_bounds": {
    "axis_substrate_max": "0",
    "axis_verification_max": "0",
    "axis_classification_max": "0",
    "axis_attribution_max": "0",
    "axis_tooling_max": "0",
    "axis_interface_max": "0"
  },
  "min_attestation_bond_uzrn": "<governance-selected non-zero bond>",
  "min_per_claim_bond_uzrn": "0",
  "slash_gradient": {
    "compiler_drift_bps": 10000,
    "axis_overflow_bps": 10000,
    "fraud_bps": 10000
  },
  "required_qualification_domain": "<governance-selected domain>",
  "min_qualification_status": "QUALIFICATION_STATUS_ACTIVE",
  "allowed_class_ids": [],
  "status": "ADAPTER_STATUS_ACTIVE",
  "registered_via_lip_id": "<LIP ID>",
  "witness_reward_uzrn": "0"
}
```

Before registration, governance must publish:

- the passing production-policy cryptographic fixture and reproducible test
  command;
- the Zerone source commit;
- Go toolchain and target tuple;
- reproducible build command and binary SHA-256;
- trusted-root JSON and its SHA-256;
- exact issuer, SAN, artifact-digest selection rule, and predicate type;
- immutable bundle retention locations; and
- the independent re-verification/challenge procedure.

## 7. Threat model

The adapter protects against payload or proof-bundle mutation, signature
forgery under the configured trust root, use of an unapproved certificate
identity, removal of required transparency evidence, artifact substitution,
statement-version downgrade, and predicate-type substitution.

It does not protect against a compromised authorized signer or builder, false
predicate contents, a maliciously approved trusted root, governance choosing a
weak identity, loss of the bundle, or disagreement about caller-supplied policy
that governance failed to pin. Sigstore identity is evidence of control, not a
general endorsement of the signed claim.

Because the chain verifies the structural link hash rather than Sigstore
cryptography in consensus, challengers must re-run the registered compiler
with the published policy. Keep the adapter unregistered and
`witness_reward_uzrn` at `"0"` until the production fixture and challenge path
are operational.
