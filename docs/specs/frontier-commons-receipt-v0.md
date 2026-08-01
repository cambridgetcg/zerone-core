# Frontier Commons receipt v0 — One Bounded Inconclusive Self-Receipt

Status: `INTERNAL_DOGFOOD_ONLY`; unsigned, self-declared, source-level,
non-authoritative, and recording no external participant or signatory.

Canonical participation contract:
[`zerone.frontier-commons-participation/v0`](frontier-commons-participation-v0.md)
and its
[machine-readable companion](../../dashboard/public/standards/frontier-commons-participation.v0.json).

Receipt schema: `zerone.frontier-evaluation-receipt/v0`.

Canonical public fixture: `/standards/frontier-commons-self-receipt.v0.json`
(repository source:
`dashboard/public/standards/frontier-commons-self-receipt.v0.json`).

## FC-0.1 self-receipt profile

This anchored profile is the only receipt profile authorized by this document.
Its reviewed `predicateType` is
`https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/frontier-commons-receipt-v0.md#fc-01-self-receipt-profile`.

### 1. Authority and subordination

This profile is an evidence accessory to FC-0. It is not another participation
compact, membership instrument, invitation system, corporate agreement, or
governance document. The canonical FC-0 contract controls participation terms.
This document controls only the shape and deliberately narrow interpretation of
its receipt.

The receipt cannot change FC-0's canonical state of `SET_NOT_MET`, satisfy any
of C0-C3, activate a successor gate, or establish adoption. A valid receipt
authenticates neither truth nor organizational authority merely because its
bytes pass validation.

Static publication of the FC-0 page, JSON, specification, validator, or receipt
fixture only makes public bytes available for inspection. It does not:

- authorize targeted contact or an external or corporate invitation;
- register a participant, member, partner, supporter, signatory, or endorser;
- request evidence, confidential information, model access, or research data;
- authorize name or logo use, research, security testing, biological work,
  governance, qualification, rewards, or network writes; or
- create consideration, payment, a protocol economic effect, or a production
  dependency.

### 2. Exact raw-byte identity

The internal fixture is one in-toto
[Statement v1](https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md)
with exactly one subject. That subject is the final canonical
`frontier-commons-participation.v0.json` file and carries its lowercase SHA-256
digest.

The digest is calculated over the exact raw UTF-8 bytes. JSON reserialization,
key reordering, whitespace changes, newline changes, Unicode normalization, or
an otherwise semantically equivalent object produces different bytes and must
not satisfy the reviewed pin. The receipt's own raw bytes are independently
SHA-256-pinned by its source validator and adversarial tests.

The reviewed envelope has:

- `_type`: `https://in-toto.io/Statement/v1`;
- one subject named `frontier-commons-participation.v0.json`;
- a subject `sha256` equal to the exact canonical FC-0 bytes;
- the exact reviewed `predicateType` URI
  `https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/frontier-commons-receipt-v0.md#fc-01-self-receipt-profile`;
- predicate schema `zerone.frontier-evaluation-receipt/v0`; and
- receipt kind `ZERONE_SELF_DOGFOOD`.

The current receipt is not DSSE-wrapped and has no signature or signatory. Its
issuer and control-root fields are explicitly project-role self-declarations
with artifact-only scope and unverified authority. Even a future valid outer
signature would establish only control of a signing key over particular bytes;
it would not prove predicate truth, independence, safety, compliance,
employment authority, or consent by another being or organization.

### 3. Deliberately absent evaluation material

The internal dogfood receipt binds no evaluation bundle. All seven
evaluation-material digest fields are required to be exactly `null`:

1. `protocolDigest`;
2. `threatModelDigest`;
3. `fixtureDigest`;
4. `acceptancePolicyDigest`;
5. `environmentDigest`;
6. `evidenceDigest`; and
7. `challengePolicyDigest`.

`null` means absent. It is not an omitted secret, private attachment, wildcard,
placeholder, or assertion that material exists elsewhere. A hash-shaped string
in any of these fields must fail the current self-dogfood profile.

The result is exactly `INCONCLUSIVE`, with reasons that include:

- no bound evaluation materials;
- no external review;
- no operational enforcement; and
- no signatories.

The receipt is therefore a bounded test of the envelope, parser, byte pin, and
semantic walls. It is not an evaluation of a frontier laboratory, a favorable
finding, or evidence that FC-0 is institutionally ready.

### 4. Privacy, effects, and non-implications

The disclosure class is `PUBLIC_METADATA_ONLY`. The self-receipt declares false
for containing confidential data, personal data, model weights, training data,
private prompts, exploit details, secrets, or export-controlled material. The
validator checks the declaration and reviewed shape; it cannot infer safe
classification from arbitrary content. Classification remains the
responsibility of an authorized privacy, security, legal, or data owner.

Every effect is false:

- truth;
- safety;
- compliance;
- certification;
- endorsement;
- membership;
- economic effect;
- reward;
- KARMA;
- qualification;
- authority;
- governance;
- privacy-rights effect; and
- network write.

The receipt records no actual participant or external signatory. It grants no
authority, affiliation, publicity right, liability allocation, service level,
IP transfer, research permission, wallet relationship, or financial claim.
Time, compute, legal review, security review, and opportunity cost can still be
real; the absence of protocol consideration must not be described as zero
participant cost or zero risk.

### 5. Temporal and correction semantics

The receipt records four ISO calendar dates:

- `evidenceCutoffOn` must not follow `createdOn`;
- `reviewAfterOn` must follow `createdOn`; and
- `expiresOn` must not precede `reviewAfterOn`.

Freshness is derived only when a verifier supplies an explicit `asOf` date:

- before `createdOn`: `NOT_YET_CREATED`;
- from `createdOn` until `reviewAfterOn`: `CURRENT`;
- from `reviewAfterOn` through `expiresOn`: `REVIEW_DUE`; and
- after `expiresOn`: `EXPIRED`.

Without `asOf`, freshness is `NOT_EVALUATED`. An expired receipt remains
inspectable history but is not current evidence and cannot silently refresh
itself. A returned temporal status describes the caller-selected `asOf`; it is
not independent proof of current wall-clock freshness.

Correction is append-only. The current self-receipt requires no relations. A
separately reviewed future profile or version may admit bounded relations for
`CHALLENGES`, `DIVERGES_FROM`, `REPAIRS`, `REPLICATES`, `SUPERSEDES`, and
`WITHDRAWS_FUTURE_RELIANCE`; this v0 validator admits none. Such a later
receipt may correct interpretation or withdraw future reliance, but it cannot
promise deletion of repository history, mirrors, forks, caches, or
transparency logs. Exit from future use cannot remove unrelated public access
or create a negative record.

### 6. Current validator boundary

The current source validator accepts only the exact reviewed
`ZERONE_SELF_DOGFOOD` bundle. It fails closed on malformed or duplicate-key
JSON, unknown or reordered fields, excessive size or nesting, byte-pin drift,
wrong subject, predicate or receipt kind, non-null material digests, a result
other than `INCONCLUSIVE`, fabricated participants or signatories, opened
privacy or effect flags, incoherent dates or relations, and weakened correction
semantics.

`PUBLIC_EVALUATION` is rejected even when it supplies hash-shaped values. That
lane is not implemented or authorized. Renaming the receipt kind, filling the
seven digests, adding a signature, or publishing the file cannot activate it.
A public-evaluation validator requires a separately reviewed version and
activation decision.

### 7. FC-0.1 independent-roundtrip requirements

The current operation status remains `INTERNAL_DOGFOOD_ONLY`; the next evidence
status remains `INDEPENDENT_ROUNDTRIP_NOT_RUN`. FC-0.1 is complete only after a
separately consenting and independently controlled evaluator performs a
bounded clean-machine roundtrip under a newly reviewed public-evaluation
profile that:

1. supplies the exact bounded raw bytes for the public subject;
2. supplies the exact raw bytes for all seven evaluation materials;
3. supplies every related receipt's exact raw bytes;
4. recomputes each declared digest offline rather than trusting hash-shaped
   metadata;
5. uses independently configured signer and effective-control-root policy;
6. enforces issuer scope, subject, predicate, result, reason, relation, cutoff,
   review, expiry, and correction coherence;
7. uses classification approved by an authorized privacy or security owner and
   never inferred by the hash validator;
8. fails closed on mutation, replay, wrong signer, wrong subject, wrong
   predicate, missing material, duplicate keys, oversize input, expiry, and
   stale policy;
9. rehearses challenge, correction, supersession, withdrawal of future
   reliance, portability without Zerone, and clean exit; and
10. requires no confidential or personal data, weights, private prompts,
    exploit detail, wallet, token, stake, reward, consideration, qualification,
    governance, production credential, or favorable result.

`REFUSED`, `DIVERGED`, and `INCONCLUSIVE` satisfy the outcome-independence
requirement. FC-0.1 success does not make the evaluator, its employer, or any
laboratory a participant or endorser, and it does not by itself satisfy a
Corporate M1 gate other than the independent-roundtrip gate.

### 8. Corporate M1 requirements

Corporate M1 remains `NOT_READY` and authorizes no institutional invitation.
Before any bounded corporate approach, all 18 gates must be closed through
separately reviewed evidence for the exact counterparty, people, artifact,
jurisdiction, and scope:

1. accessibility, labor, worker-classification, and whistleblower review;
2. code of conduct, proportionate enforcement, appeal, anti-retaliation, and
   protected reporting;
3. competition and confidentiality review for the exact scope;
4. contribution, IP, patent, publication, and license terms;
5. counterparty scope and signatory authority;
6. an explicit, accountable human decision authorizing the bounded outreach;
7. governing terms, governing law, jurisdiction, and dispute process;
8. independent governance, capture, custody, and remedy review;
9. independent receipt-parser, threat-model, and raw-material-binding review;
10. liability, warranty, indemnity, insurance, and participant-remedy terms;
11. logo, name, affiliation, endorsement, and publicity rules;
12. a successful FC-0.1 independent roundtrip;
13. maintainer, change-control, versioning, and deprecation rules;
14. non-targeting outreach conduct: no profiling or identity targeting,
    minimized lawful contact sources, at most one bounded contact, stop on
    silence or decline, and finite contact-data retention;
15. privacy data map, DPA decision, retention, erasure, and public-permanence
    review;
16. procurement, tax, accounting, sanctions, export-control, and financial-
    promotion review;
17. security policy, coordinated disclosure, safe harbor, incident response,
    and embargo rules; and
18. service-level, support, availability, portability, and exit terms.

No source publication, self-receipt, issue, pull request, favorable result,
public mention, or organization count can substitute for these gates. A legal
or security refusal, unsafe public classification, unclean exit, need for model
or production credentials, implied truth or endorsement, individual ranking or
surveillance, release-path dependency, or cost-cap overrun without renewed
consent is a stop condition.

### 9. Ordinary source work is not institutional participation

Anyone may inspect, hash-verify, copy, modify, and fork licensed repository
works under Apache-2.0. A person may also intentionally offer an ordinary public
patch under the repository's current review process after considering the
license, including Apache-2.0 section 5, and the policy gaps. Those acts do not
create membership, endorsement, KARMA, reward, governance, a corporate receipt,
or authority to bind an employer.

An institutional source, policy, model, dataset, research, security, or
confidential contribution is a different lane. It remains unavailable until
the applicable Corporate M1 gates establish counterparty authority, IP and data
terms, conduct and remedy, security and disclosure handling, competition
review, publication scope, and clean exit. Neither a public repository nor
this receipt silently supplies those missing terms.

No stage sets a calendar deadline, adoption quota, named prospect, logo target,
conversion goal, favorable-result requirement, or presumption of consent. A
decline or no response ends any later authorized approach without penalty,
loss of unrelated access, negative KARMA, or an adverse record.
