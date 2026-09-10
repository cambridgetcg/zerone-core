**Knowledge record integrity v1**

`knowledge-record-integrity-v1` is the named consensus boundary for faithful
review records, reviewer-bound commitments, atomic fact/history writes and
explicit verifier payment obligations. It advances knowledge **7 → 8** from the
complete frozen `survivalHandoffTargetVersionMap()`. Other module versions stay
unchanged. The existing SDK staking/governance authority design and its adoption
gates remain in force.

The purpose is to preserve who asserted what, who reviewed it, their stated
reasons and evidence, and subsequent corrections. A signature authenticates an
account's assertion. It does not establish discovery, independent control,
scientific expertise or that a described experiment was performed.

**Commitments and signed records**

New rounds created after activation use commitment scheme **2** and retain their
creation chain ID. Existing scheme **0** rounds keep the original v1 hash and
existing commitments. Unknown schemes refuse execution; a new round cannot
fall back to the legacy scheme. Migration does not rehash, relabel or rewrite
old rounds. Scheme and chain context are immutable after creation.

The scheme-2 digest commits, in fixed order, to a domain/version tag, chain ID,
round ID, canonical reviewer address bytes, vote, confidence, salt, method ID,
reason, scope and ordered evidence references. Variable-length fields use
explicit length framing; confidence uses a fixed-width integer. The salt is
16–64 bytes. Confidence is on the existing 0–1,000,000 scale and is retained in
the reveal record; it is a declared assessment, not a calibrated probability.
Copying another reviewer's hash and revealed preimage fails because the
reviewer's address changes the digest. Cooperating reviewers can still agree
or coordinate; this is not a proof of independent work.

The new reveal includes a required, nonblank reason and optional method, scope
and evidence references. Limits are 4,096 UTF-8 bytes for reason, 1,024 for scope,
128 for method ID, sixteen evidence references of at most 256 bytes each, and
8,192 total attestation text bytes. Exact signed text is retained. Unknown
attestation fields are refused so a signer cannot believe an unbound extension
was included in the commitment. Legacy rounds refuse these new unbound fields.

Ordinary claims gain method ID and reasoning input. Challenge records retain
reason, evidence IDs and method; provisional challenges also retain their
validated target claim ID and counter-claim. Accepted graph relations retain
their submitted method ID. Input validation closes unknown claim/relation
enums, unsupported categories, dangling references and the ordinary-conjecture
bypass. User-declared methods cannot claim the privileged doctrine-authorship
lifecycle exemption. Claim reasoning is bounded to 8,192 bytes, and legacy claim
references to 64. Canonical structured input must agree with the supplied
structure; no hash is represented as a proof of semantic novelty.

CLI commitment construction queries the round's actual scheme and context.
Offline construction requires explicit scheme/context. CLI and TypeScript SDK
use the same versioned digest and retain the exact reveal fields. An imported
scheme-2 round keeps its original commitment chain context; a transaction sent
on the importing chain is still signed for that chain through SDK transaction
signing. Genesis import is not proof of original network execution.

**Payments and history**

Activated ordinary claim submission caches fee collection, sponsorship usage,
the fee split and claim/round writes. A checked failure rolls back that attempt.
Amounts outside the existing uint64 review-pool arithmetic are refused rather
than truncated. This does not change the review-fee split or reward formula.

A completed round records a frozen verifier payment plan in its existing primary
record. It includes creation height, each recipient's amount and withholding,
the total withholding and a paid height. Paid height zero means pending. The
scientific verdict can finalize while payment remains pending. Every payment
leg, the paid marker and removal from the pending index commit in one SDK cache;
a failure transfers nothing and preserves the full obligation. Retries do not
recalculate amounts from later reputation or parameter values. Paid events are
emitted only with successful batch settlement. Recognition of a review labels
its basis as panel agreement and does not itself claim a transfer.

A derived cursor processes at most sixteen pending records and 513 possible
transfer legs per block. It leaves an over-budget batch for the next block and
rotates past failed payment attempts, rather than starving later records. The
existing primary round is the obligation; the index can be reconstructed. No
plan is inferred for an old completed round whose actual payment is unknown.
These bounds apply to pending retries; ordinary round completion still has its
existing wider lifecycle costs.

Activated fact writes commit the fact, its status transition and relevant
indexes together. Falsification cascades propagate checked history failures.
Status sequence counters cannot silently reset on malformed data or overwrite
an existing transition. Ordered cascade keys supply the next sequence without
rescanning the entire prior cascade for every descendant. Strict readers and
exports report corrupt records instead of silently omitting them. Existing
history gaps remain gaps; the release does not invent missing past events.

Genesis now explicitly transports completed/expired rounds, status transitions,
cascade events and status counters, alongside existing active rounds and claims.
Import reconstructs derived indexes and preserves pending and paid plans. It
does not record an imported fact's current status as if that were a newly
observed historical transition. The selected claim, round and history inventories
refuse unsupported fields, malformed records and inconsistent identities. This
is not a complete raw-store audit of every other knowledge namespace.

**ToK query meaning and resource limits**

ToK bundles remain derived queries. Extracting a bundle adds no transaction or
persistent root. Existing v1/v2 digest algorithms retain their historical byte
meaning: graph identifiers and selected relationships/history, excluding full
fact payload and chain/height context. Query events describe a derived digest;
they are not chain-authenticated pin receipts. Full payload authentication,
selector-completeness proofs and independent trusted-header verification are
separate work and are not claimed by this release.

A requested nonzero height must equal the actual SDK query-context height.
The parameter does not load an old version. Clients must select an available
historical SDK query context, or receive a refusal. Ancestor selection enforces
its branch limit, and frontier selection orders by verification height with a
deterministic ID tie break.

Whole-query ceilings are 8,192 nodes, 32,768 edges, 65,536 examined key/value
entries, 16 MiB of examined key/value bytes and 8 MiB of encoded output. Exceeding
a ceiling refuses the response with ResourceExhausted rather than returning an
apparently complete partial graph. Corrupt state refuses with an integrity
error. These limits do not promise historical availability or expose new routes
through the public website's REST allowlist.

Training traces no longer manufacture intermediate confidence histories where
no such event log exists. Missing history stays absent. A challenge node describes
the challenge verdict: rejection means the original survived. Completed
inconclusive outcomes remain inconclusive; unresolved challenges have no panel
verdict leaf. Unknown challenge/rebuttal occurrence heights remain absent, while
verdict and vindication heights come from retained records. A contradicting edge
alone is not presented as proof of which challenge caused a disproof.

**Execution and compatibility**

The upgrade plan has this exact name, a positive height H, empty `info`, and no
deprecated time/client fields. The target binary admits an existing predecessor
database only at the exact committed H−1 boundary with matching on-chain and
local plans. The source map is knowledge 7/vesting 3 plus every unchanged
survival-predecessor module version. A historical handler cannot carry the new
knowledge 7→8 delta; unsafe-skip, mixed maps, early startup and forged lineage
are refused.

The named handler validates old rounds and history, then enables the separate
record-integrity marker and completes the module migration in one cache.
Selected activation/export inventories are bounded to 100,000 records and
64 MiB per inventory; exceeding a bound refuses the operation. Existing raw
records are not rewritten. The activation flag is separate from the applied
upgrade marker/done height. Native or explicitly imported current genesis sets
the flag directly, without manufacturing an applied-upgrade receipt. Restart
requires coherent source/target versions and flag/marker/done metadata.

The deployed legacy `zerone-1` and its signed observer package are a different
application lineage. This source is not their drop-in replacement and does not
schedule their upgrade, change custody, open validator admission or activate
reward research. Network adoption requires the applicable preceding releases
and verification with the actual intended state and binaries.

**Remaining consolidation**

This patch preserves the current economic formulas. Balance-only reviewer
admission, promised slash liability, related-party bounty eligibility,
agreement-to-qualification/reputation feedback and global fitness/metabolism
work still require their own explicit consolidation. Reasons and evidence are
retained assertions, not an automatic assessment of their quality. The next
economic work should use these records and actual funded tasks; it should not
add another score to compensate for unresolved authority or independence.

Validation combines adversarial commitment and codec vectors, actual SDK bank
rollback/retry tests, bounded graph/history failure fixtures, named-upgrade
restart/export tests and the separate two-binary rehearsal. Rehearsals use
disclosed synthetic local accounts and data; they establish compatibility and
execution behavior, not independent scientific review or production adoption.
