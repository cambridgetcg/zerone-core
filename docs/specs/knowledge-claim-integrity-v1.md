# Knowledge claim integrity v1

`knowledge-claim-integrity-v1` advances knowledge **11 → 12**. Its predecessor
is the completed [fund-settlement](knowledge-fund-settlement-v1.md) state
machine. The source base is `a4272bfa`. Publishing this source upgrades no
network: `zerone-dev-1` still runs knowledge 11, and scheduling its upgrade is a
separate decision. Legacy `zerone-1` is not affected.

The release carries three decisions recorded on 2026-10-16:

1. **Who said what is recorded on chain.** An attribution is an ordinary claim
   that names a work, a voice, an act, a proposition and a locator, using
   hashes only (§2). Every status change on the dispute paths records its exact
   cause (§6.2).
2. **The verdict on a proposition is a separate decision by chain
   participants.** A fact's status changes only through a panel decision on that
   fact, or through an accepted finding that conflicts with it or that it
   depends on (§6). Neither a submitter nor the defender of a disputed fact can
   review (§7). Accepting an attribution never changes a proposition's verdict.
3. **Verdicts can be reversed by the latest findings.** A CONTESTED fact can be
   challenged. A fallen counter-fact releases its target. Surviving a direct
   challenge clears dependencies refuted before it. A rejected, starved or fallen
   proposition can be submitted again as a new record, and the old record is
   kept (§3, §6).

There are no new messages, tx fields or store prefixes. The only proto change
is the genesis flag `claim_integrity_enabled` (§1.2).

## How to read this specification

- *Marker on* means knowledge-12 semantics, selected by store key `0x87`
  (§1.1). *Marker off* means knowledge 11.
- Each statement of behaviour ends with **Tests:**, naming the tests that check
  it. Go tests are in `x/knowledge/keeper` unless a path is given. Most run the
  same scenario with the marker on and with it off. Python test names are in
  `scripts/test_claim_workflow_attribution.py` unless a path is given.
- *Not tested here* marks a statement read from code, or a convention the chain
  does not enforce. Such statements are labelled so that nobody mistakes them
  for checked behaviour.
- The sandbox acceptance run and the two-binary upgrade rehearsal (§7 of
  `docs/plans/2026-10-16-claims-attribution-verdict-integrity.md`) are separate
  verification. This specification does not report their results.

## 1. Activation

### 1.1 Marker 0x87

- The knowledge store key `0x87` holds exactly `{1}` when claim integrity is on.
  An absent key reads as off and `{1}` as on. Any other value is an error, which
  every caller returns; it is never read as off. A transaction is refused, a
  round does not complete, and the BeginBlock expiry branch leaves its targets
  as they are without halting the chain.
  Tests: `TestClaimIntegrityMarkerReadsAreExact`,
  `TestCompleteRoundRefusesMalformedClaimIntegrityMarker`,
  `TestClaimIntegrity_LegacyPolicyStarvedContradictionSettles`,
  `TestClaimIntegrity_ReviewerPartyCheckFailsClosed`,
  `TestAnteIntegration_ClaimIntegrityGateReadErrorRejectsTx`.
- Enabling requires record integrity, review neutrality, claim records and fund
  settlement (markers `0x81`, `0x84`, `0x85`, `0x86`).
  Tests: `TestClaimIntegrityEnableRequiresEveryPredecessor`.
- Only `InitGenesis` and the 11→12 migration write the marker. No message or
  parameter writes it, and nothing removes it. *Not tested here*: read from
  code, where `EnableClaimIntegrity` is its only writer.

### 1.2 Genesis flag

- `GenesisState.claim_integrity_enabled` (field 71). `DefaultGenesis` sets it to
  true, so a chain created by `zeroned init` runs knowledge-12 rules from block 1.
  `InitGenesis` with the flag writes `0x87 = {1}`.
  Tests: `TestClaimIntegrityNativeGenesisWritesMarker`,
  `TestClaimIntegrityNativeSelectionAndRestartReceipt`.
- Genesis validation refuses the flag without `fund_settlement_enabled`.
  Tests: `TestClaimIntegrityGenesisRequiresFundSettlement`.
- Export writes the flag from the marker. Export followed by import preserves
  both true and false.
  Tests: `TestClaimIntegrityGenesisRoundTripsFlag`.
- A knowledge-12 binary refuses native genesis whose flag is missing or false.
  The flag is never derived from `fund_settlement_enabled`, so a pre-v12 export,
  which has no flag, is refused rather than silently switched to knowledge-12
  rules.
  Tests: `TestClaimIntegrityNativeSelectionAndRestartReceipt`.
- Import invents no upgrade receipt: there is no `migration_v12_complete`, the
  done height is 0, and the imported chain restarts as native genesis.
  Tests: `TestClaimIntegrityMigrationPreservesEveryByteAndRestart`.
- With the flag on, `InitGenesis` replays claims in admission order
  (`submitted_at_block`, then ID), so each duplicate-index entry again names the
  latest admitted record (§3). With the flag off the file order is kept.
  Tests: `TestClaimIntegrity_ImportReplaysAdmissionOrder`.

### 1.3 Migration and named upgrade

- `Migrate11to12` requires the upgrade owner `knowledge-claim-integrity-v1`, an
  absent marker, every predecessor, and no `migration_v12_complete`. In one
  cache it writes `0x87 = {1}` and `migration_v12_complete = "true"`, and nothing
  else. The module refuses a migration that no owner runs.
  Tests: `TestMigrate11to12RefusesWithoutOwnerOrUnmarkedSource`,
  `TestMigrate11to12ChangesExactlyMarkerAndReceipt`,
  `TestClaimIntegrityModuleRefusesUnownedMigration`.
- Only `knowledge-claim-integrity-v1` may carry knowledge 11→12. Every older
  plan name is refused for that boundary, and the earlier boundaries keep their
  owners. `BuildChainVersionReport` lists the upgrade.
  Tests: `TestClaimIntegrityOwnsExactVersionBoundary`.
- The handler requires:
  - the exact plan: its name, a positive height, empty `info`, no time and no
    client state;
  - the exact source and target version maps;
  - completed fund settlement;
  - the exact committed H−1/H boundary;
  - a height that is not a skip height.
  Each refusal is checked by its exact text through `ApplyUpgrade`.
  Tests: `TestClaimIntegrityRejectsWrongSourceAndPrematureActivation`.
- Across the upgrade, every byte of every store is unchanged except `0x87`, the
  receipt, and x/upgrade's own bookkeeping (the plan is cleared, and the done
  height, knowledge version and protocol version are recorded). A fact
  CONTESTED before activation is still CONTESTED after it.
  Tests: `TestClaimIntegrityMigrationPreservesEveryByteAndRestart`.
- At startup the v12 binary:
  - refuses v11 state before H−1;
  - accepts v11 state at exactly H−1 with matching on-chain and local plans;
  - accepts a restart at the target, after either an applied upgrade
    (`0 < done ≤ latest` and the receipt present) or native genesis (done 0 and
    no receipt);
  - refuses a missing or malformed marker, and a receipt and done height that
    disagree.
  Tests: `TestClaimIntegrityMigrationPreservesEveryByteAndRestart`,
  `TestClaimIntegrityNativeSelectionAndRestartReceipt`.

### 1.4 Marker off is knowledge 11

Every new rule is behind the marker. With the marker off, the knowledge store
after the knowledge-11 characterization scenario matches, byte for byte, the
digest recorded on `a4272bfa`. The only difference is reads of `0x87`, which
cost gas, on the
admission, round-completion, challenge and commitment paths and in the ante of a
tx that carries a dispute message.
Tests: `TestV11_GoldenKnowledgeStoreDigest`, plus the marker-off half of the
paired tests below, for example `TestClaimIntegrity_ResaveDoesNotMoveIndex`,
`TestClaimIntegrity_ChallengeTransitionsMarkerOffKeepInferredCauses` and
`TestClaimIntegrityVoteExtensionSkipsPartyCommitment`.

### 1.5 Existing state at activation: nothing is rewritten

| State before activation | What happens after it | Tests |
|---|---|---|
| Duplicate-index entries naming terminal claims or fallen facts | They stop blocking, because the check reads the record's status (§3). | `TestClaimIntegrity_ClaimStatusDecidesWhetherEntryBlocks`, `TestClaimIntegrity_FactStatusDecidesWhetherEntryBlocks` |
| Canonical hashes in form v1 | They do not interact with form v2. A reworded twin of a pre-v12 structured claim can be admitted once. Exact text is still caught by the content hash, and v1 forms still find pre-v12 facts. | `TestClaimIntegrity_LegacyV1CanonicalEntriesDoNotInteract` |
| A fact CONTESTED by a knowledge-11 admission flip | When that contradiction ends, the contest-reason check (§6.3) releases the fact. A standing accepted counter-fact holds it instead. | `TestClaimIntegrity_PreActivationContestSettlesThroughPredicate` |
| A fact laundered to VERIFIED while a counter-fact stands | It stays VERIFIED. Either side can be challenged. When a challenge of it ends without refuting it, it returns to CONTESTED. | `TestClaimIntegrity_LaunderedFactChallengeFollowsStandingCounterFact` |
| A reviewer's commitment to their own claim | It still reveals and settles, and the reviewer is paid. New commitments are refused (§7). | `TestClaimIntegrity_PreActivationSelfCommitStillSettles` |
| Claims, rounds, facts, relations, status transitions, payments | Never rewritten. | `TestClaimIntegrityMigrationPreservesEveryByteAndRestart`, `TestMigrate11to12ChangesExactlyMarkerAndReceipt` |

The rows on contested, laundered and self-reviewed records are each checked
after an in-place upgrade and after an import.

## 2. Who said what: the attribution convention

### 2.1 Two records, two verdicts

1. **Attribution claim.** "Work W, in voice V, performs act A on proposition P
   at locator L." Its panel judges fidelity only, never truth.
2. **Proposition claim.** An ordinary claim stating P in the extractor's own
   words. Its panel judges truth or support.

They are joined by a hash: the attribution's `structure.object` is
`zrn-content:` followed by the proposition claim's on-chain content hash,
`sha256("ZRN.claim.v1:" + domain + ":" + text)`. That hash can be computed
before the proposition is submitted. No Reference or Relation joins the two
records. An acceptance contests another fact only along a stored CONTRADICTS
edge, so accepting an attribution never changes a proposition's verdict. An
attribution can stay VERIFIED while its proposition is DISPROVEN: the source
did say it.
Tests: `test_hermes_fixture_produces_exact_attribution_fields` (references and
relations are empty), `test_content_hash_matches_go_vectors`,
`TestContestAcceptedTargets`.

*Not tested here* (the reason for having no edge, read from code): a Reference
would clamp the proposition's confidence to the attribution's through
`computeProvenance`, and a CITES edge would let a refuted attribution cascade
into the proposition.

### 2.2 Field mapping

An attribution is an ordinary `MsgSubmitClaim`. No message, field or store key
is added for it.

| Field | Value |
|---|---|
| `claim_type` | `CLAIM_TYPE_RELATION` |
| `domain` | the proposition's domain, so both records share one content-hash namespace |
| `category` | `empirical` (`historical` is not an accepted category) |
| `method_id` | `M-HISTORICAL` (default; check that it is registered on the target chain) |
| `structure.subject` | lowercase work ID: `doi:<doi>`, `arxiv:<id>v<n>`, `isbn:<13 digits>` or `url-sha256:<64 hex>` |
| `structure.predicate` | `<act>:<voice>`; act is one of `states`, `reports`, `hedges`, `denies`; voice is `author` or `quoted:<speaker-ref>` |
| `structure.object` | `zrn-content:<64 hex>`, the proposition's content hash |
| `structure.scope` | `ver=sha256:<64 hex>;loc=<locator>`: the digest of the retrieved source bytes and a lowercase locator such as `p4.para2`, `sec3.2.para4`, `fig2` or `t=00:12:31` |
| `structure.temporal_scope` | publication date of the pinned version, `YYYY-MM-DD`, or empty |
| `structure.tags` | `attr-v1`, `stmt:<first 40 hex of the StatementOccurrence hash>`, `zc:<first 40 hex of the content hash>` |
| `fact_content` | `Attribution zerone.attribution/v1: <subject> <predicate> <object> at <scope>` (a fixed template, so no excerpt can get in) |
| `reasoning_trace` | RFC 8785 (JCS) JSON with hashes only, keys `schema`, `source_version_record`, `statement_occurrence`, `excerpt_sha256`, `proposition_record`, `extractor` |
| `references`, `relations` | empty |
| `canonical_form` | omitted; the chain derives form v2 from the structure (§4) |

A `speaker-ref` is `orcid:…`, `wikidata:q…`, or the opaque `hermes:<64 hex>`.
Private persons appear only as opaque `hermes:` references. Structure fields and
tags must match `^[a-z0-9][a-z0-9:._/;=+-]*$`.

The encoder is `scripts/claim-workflow.py attribute`. It is a dry run by
default; `--dry-run` is an explicit no-op and conflicts with `--broadcast`.
`--broadcast` goes through the ordinary submit path with the structure flags and
never sends `--canonical`.
Tests: `test_hermes_fixture_produces_exact_attribution_fields`,
`test_fact_content_template_is_byte_exact`,
`test_reasoning_trace_is_jcs_with_hashes_only`,
`test_jcs_matches_rfc8785_ordering_and_string_rules`,
`test_content_hash_matches_go_vectors`,
`test_quoted_voices_and_other_work_identifiers`,
`test_uppercase_or_free_text_in_structure_fields_is_refused`,
`test_unknown_missing_or_misversioned_intake_fields_are_refused`,
`test_method_and_category_defaults_and_bounded_overrides`,
`test_broadcast_uses_submit_path_with_structure_flags_and_never_canonical`,
`test_signing_bridge_maps_relation_and_still_refuses_other_types`,
`test_submit_arguments_cannot_drift_from_retained_structure`,
`test_dry_run_prints_the_attribution_fields_offline`,
`test_explicit_dry_run_flag_is_a_no_op_and_conflicts_with_broadcast`.

The chain does not check this convention. A third-party client can submit
anything an ordinary claim may hold; reviewers push back by voting MALFORMED
(§2.6). *Not tested here.*

### 2.3 Example (synthetic)

Proposition claim P1, type ASSERTION, domain `chemistry`, category `empirical`:

```
subject   boiling point of compound Z
predicate equals degrees C at 1 atm
object    78.4
scope     pure sample
text      Compound Z boils at 78.4 °C at 1 atm for a pure sample.
content   f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026
form v2   zrn.cf.v2/CLAIM_TYPE_ASSERTION(domain="chemistry",subject="boiling point of compound Z",predicate="equals degrees C at 1 atm",object="78.4",scope="pure sample",temporal_scope="")
canonical 7c754ede41adc2cd62468985e42079ad673e090a77be7fedd4373cbac26cc790
```

Attribution A1, "Smith 2019 states P1 as author, on p. 4, paragraph 2":

```
claim_type  RELATION   domain chemistry   category empirical   method_id M-HISTORICAL
subject     doi:10.1000/p6-smith-2019
predicate   states:author
object      zrn-content:f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026
scope       ver=sha256:0580a051899e6a386f8213633badf896d07f21a370a8ba915f2bf93ba054bce3;loc=p4.para2
temporal    2019-03-01
tags        attr-v1, stmt:e25fa6b012331b0f8681c04b88d4ee7eacaa089d, zc:f789edd890d61613208c09ebc74248b09c2f170e
fact_content (243 bytes)
  Attribution zerone.attribution/v1: doi:10.1000/p6-smith-2019 states:author zrn-content:<64 hex> at ver=sha256:<64 hex>;loc=p4.para2
```

In the real `fact_content`, each `<64 hex>` is the full value shown above.
Tests: `TestCanonicalFormV2_PlanExample`, `test_content_hash_matches_go_vectors`,
`test_fact_content_template_is_byte_exact`.

If a second source reports a different value (81.0), submit it as its own
proposition P2; form v2 keeps it apart from P1 (§4). For P2 to contest P1,
submit it with a CONTRADICTS relation to P1's fact, or as
`MsgSubmitContradiction`. P1 becomes CONTESTED only when P2 is accepted (§6).
Tests: `TestClaimIntegrity_CanonicalV2SeparatesDistinctFindings`,
`TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted`.

### 2.4 Roles and Hermes binding

| Role | Where it is on chain |
|---|---|
| author or quoted speaker | `subject`, plus the voice in `predicate` |
| extractor | `reasoning_trace.extractor`, bound by the StatementOccurrence hash |
| submitter | `Claim.submitter`, signed |
| reviewers | round commitments and reveals with `ReviewAttestation{method_id, reason, evidence_ids, scope}`; `scope` is `attribution-fidelity` or `proposition-truth` |

Hermes records are referenced as `sha256:<64 lowercase hex>` over their
canonical (JCS) bytes:

- SourceVersion is the `ver=` digest in `scope`, plus `source_version_record`.
- StatementOccurrence is `statement_occurrence`, `excerpt_sha256` and the
  `stmt:` tag. The excerpt and the full role list stay offline.
- Proposition is the proposition claim's text and `proposition_record`.
- EvidenceAssessment and RelationReview go in reviewer `evidence_ids`
  (at most 16, each at most 256 bytes) as `hermes:ea:sha256:<hex>` or
  `hermes:rr:sha256:<hex>`, or in `MsgSubmitContradiction.evidence_ids`.
- After inclusion, Hermes keeps a receipt
  `{chain_id, height, txhash, claim_id, content_hash}`. If Hermes later redacts
  an offline record, the on-chain hash points at nothing, which is harmless.

These are the adopted defaults for the Hermes contract. *Not tested here*,
except the intake shape that the encoder accepts:
`test_hermes_fixture_produces_exact_attribution_fields`,
`test_unknown_missing_or_misversioned_intake_fields_are_refused`.

### 2.5 Finding attributions

`ClaimHistory(P)` lists, one hop from the root claim, every related claim whose:

- `content_hash` equals the root's (a resubmission, or a contradiction carrying
  the same text and domain);
- `canonical_hash` equals the root's non-empty canonical hash;
- `structure.object` is exactly `zrn-content:` + the root's content hash (an
  attribution).

Each such link targets the root claim ID. An empty root hash, or an absent root
claim, yields none of these links. The existing budgets, sort order and one-hop
scope are unchanged. See also
[claim records](knowledge-claim-records-v1.md#read-the-retained-claim-history).
Tests: `TestClaimHistoryLinksResubmissionsCanonicalTwinsAndAttributions`,
`TestClaimHistoryHashLinksRequireRetainedNonEmptyRootHashes`.

The repository's strict readers accept these three link fields and still refuse
any other unknown name: `test_v12_discovery_links_are_retained_and_the_set_stays_closed`
(`tools/research-review/test_workflow.py`),
`"knowledge v12 discovery links to the root claim are accepted; other link names stay refused"`
(`sdk/typescript/tests/claim-history.test.ts`) and
`"accepts knowledge v12 discovery links to the root claim and keeps the link set closed"`
(`dashboard/tests/development.test.ts`).

For accepted attributions only, `FactsByTag("zc:<40 hex>")` also works, because
tags are indexed when a fact is created. *Not tested here.*

### 2.6 What attribution reviewers judge

This rubric is guidance for reviewers. The chain records their votes and
attestations; it does not check the rubric. *Not tested here.*

- **ACCEPT** when all of these hold:
  - the source version matches `ver=`;
  - at the locator there is an exact excerpt whose NFC sha256 equals
    `excerpt_sha256`;
  - the excerpt is uttered in the stated voice;
  - the act is faithful: a hedge stays `hedges`, and reporting someone else's
    result is `reports`;
  - the proposition text renders the excerpt fairly, neither stronger nor
    narrower.
- **REJECT** for misattribution, a wrong locator or distortion.
- **MALFORMED** when the hashes cannot be checked, a field holds free text, or
  any verbatim excerpt appears on chain.

Reviewers never vote on whether the proposition is true.

### 2.7 Quote and licensing guard

The chain has no excerpt field and cannot detect quotation, and anything written
is permanent. The guard therefore runs in the client before signing. Version:
`zerone.quote-guard/v1`.

- The excerpt is read from a local file (`--excerpt-file`) and never leaves it:
  it is not in argv, receipts, saved records or output. Receipts keep only the
  guard version and `excerpt_sha256` (sha256 of the NFC-normalised exact file
  text).
- Guarded text:
  - `submit`: the claim text and reasoning trace;
  - `challenge`: the contradiction text, reason and each evidence ID;
  - `commit`: the review reason, scope and each evidence ID;
  - `attribute`: `fact_content`, `reasoning_trace`, every structure field and
    every tag.
- Tokenizing, applied identically to the excerpt and to each field:
  1. NFKD, then casefold, then NFKD;
  2. delete every combining mark (M\*), every format character (Cf) and every
     Default_Ignorable_Code_Point;
  3. turn every remaining character that is not a letter (L\*) or decimal digit
     (Nd) into a space;
  4. split on whitespace.
  Deleting, rather than spacing, means a hidden character cannot split a word.
  Folding to base letters means a composed, decomposed or unaccented copy matches
  the same window.
- A field is refused if it contains any 8 consecutive excerpt tokens, or the
  whole excerpt when the excerpt has fewer than 8 tokens. An excerpt with no
  tokens is refused. Refusals name the field, never the text, and happen before
  any CLI call or saved record.
- `attribute --broadcast` requires a passing guard and an excerpt whose digest
  equals the record's `excerpt_sha256`.
Tests: `test_refuses_eight_token_overlap_after_normalisation`,
`test_accepts_seven_tokens_and_a_paraphrase`,
`test_refuses_a_whole_short_excerpt_only`,
`test_excerpt_digest_is_nfc_and_excerpt_must_have_tokens`,
`test_refusal_message_never_repeats_excerpt_text`,
`test_soft_hyphen_in_a_field_does_not_hide_a_quote`,
`test_soft_hyphen_in_the_excerpt_does_not_hide_a_quote`,
`test_zero_width_and_other_default_ignorables_are_deleted`,
`test_combining_marks_left_after_nfkc_are_deleted`,
`test_marks_inside_words_keep_scripts_whole_and_normalisation_is_idempotent`,
`test_guarded_submit_challenge_and_commit_record_only_version_and_digest`,
`test_guarded_attempt_keeps_snapshot_view_compatible`,
`test_guard_refuses_before_any_cli_call_or_saved_record`,
`test_parsers_offer_excerpt_file_on_submit_challenge_and_commit`,
`test_eight_excerpt_words_in_a_field_are_refused_before_signing`,
`test_broadcast_requires_the_excerpt_and_a_matching_digest`.

Limits: character categories come from the running Python's `unicodedata`
(Unicode 14 on Python 3.11), so a code point assigned in a later Unicode version
can tokenize differently between runtimes. A third-party client can skip the
guard; only reviewers voting MALFORMED push back on chain. The Hermes intake
should apply the same algorithm.

### 2.8 Known limits of attribution records

*Not tested here* (existing behaviour, unchanged by this release):

- Attribution facts are subject to metabolism (AT_RISK, EXPIRED, PRUNED). The
  durable record is the claim, its verdict and its history, not the fact's
  metabolic status.
- Facts in one niche (same domain, subject, claim type and scope) are pruned
  above `CompetitionMaxNicheSize`. The scope includes the version digest and the
  locator, so in practice a niche is one locator in one version.
- A tag is a 160-bit prefix of a hash. The full hashes are in `object` and
  `reasoning_trace`.

## 3. Duplicate checks and resubmission

Two index entries block a duplicate: `0x09` (content hash of text and domain)
and `0x34` (canonical hash). With the marker on, an entry blocks admission only
while the record it names is live or standing:

| Record the entry names | Blocks? |
|---|---|
| A claim that is PENDING, PENDING_EVALUATION, EVALUATED, PROVISIONAL, IN_VERIFICATION, CHALLENGED, CONTESTED, UNSPECIFIED or unknown | yes |
| A claim that is REJECTED, MALFORMED, INSUFFICIENT or EXPIRED | no |
| An ACCEPTED claim | It resolves to its fact `GenerateFactID(claim, round.verdict_block)`, and blocks if that fact blocks. A missing round or fact blocks (fails closed). |
| A fact that is PENDING, PROVISIONAL, VERIFIED, ACTIVE, CONTESTED, CHALLENGED, AT_RISK, UNSPECIFIED or unknown | yes |
| A fact that is SUPERSEDED, EXPIRED, DISPROVEN, REVOKED or PRUNED | no |
| Nothing | no |
| An unreadable record, or an empty entry value | the tx is refused |

Tests: `TestClaimIntegrity_ClaimStatusDecidesWhetherEntryBlocks`,
`TestClaimIntegrity_FactStatusDecidesWhetherEntryBlocks`,
`TestClaimIntegrity_DanglingEntryAllowsAdmission`,
`TestClaimIntegrity_CorruptEntryFailsClosed`.

- A starved claim releases its exact text. The resubmission is a new record with
  a new ID, and the old record keeps its status and history.
  Tests: `TestClaimIntegrity_StarvedClaimAllowsResubmission`.
- REJECTED and MALFORMED verdicts release the text.
  Tests: `TestClaimIntegrity_NegativeVerdictsReleaseText`.
- A fact refuted by a direct challenge releases both of its entries, so the
  proposition can be asserted again. The old fact keeps its status and history.
  Tests: `TestClaimIntegrity_StandingFactBlocksAndDisprovenReleases`.
- Only a claim's first save (its admission) writes its entries, so the index
  always names the latest admitted record. A re-save never moves an entry.
  Tests: `TestClaimIntegrity_ResaveDoesNotMoveIndex`,
  `TestClaimIntegrity_StarvedClaimAllowsResubmission`.
- After acceptance, `0x34` names the fact, so `FactByCanonical` resolves it
  through the index. An accepted claim's fact takes the entry only if it is
  absent or still names that claim, so a late acceptance never takes it from a
  newer record.
  Tests: `TestClaimIntegrity_AcceptedFactKeepsCanonicalEntry`.
- The refusal keeps its prefix (`duplicate claim: content hash matches existing
  claim <id>` or `canonical duplicate: matches existing claim <id>`) and adds the
  status that decided, for example `(status CLAIM_STATUS_IN_VERIFICATION)` or
  `(status CLAIM_STATUS_ACCEPTED; fact <id> FACT_STATUS_VERIFIED)`.
  Tests: `TestClaimIntegrity_StarvedClaimAllowsResubmission`,
  `TestClaimIntegrity_ContradictionCannotTakeLiveClaimText`.
- `MsgPostConjecture` and `MsgSubmitContradiction` apply the same check to their
  text. A contradiction cannot take a live claim's `0x09` entry, so two
  same-hash claims cannot be admitted in one block.
  Tests: `TestClaimIntegrity_ConjectureDedupParity`,
  `TestClaimIntegrity_ContradictionCannotTakeLiveClaimText`.
- The usual fee and cooldown apply to a resubmission, and the existing
  `claim-workflow.py --resubmit-after <txhash> --max-review-fee-uzrn N` flow
  needs no client change. *Not tested here*: neither has a unit test; the flow
  is checked only by the sandbox run (plan §7, row 1).

With the marker off, every entry blocks for good and every save rewrites it, as
in knowledge 11.
Tests: `TestV11_StarvedClaimLocksExactText`,
`TestV11_ContradictionOverwritesContentIndex`, and the marker-off halves of the
tests above.

## 4. Canonical form v2

With the marker on, the chain derives a claim's canonical form from its type and
structure:

```
zrn.cf.v2/<CLAIM_TYPE_…>(domain=q(domain),subject=q(subject),predicate=q(predicate),object=q(object),scope=q(scope),temporal_scope=q(temporal_scope))
q(v) = strconv.QuoteToASCII(collapse(TrimSpace(NFC(v))))
```

`collapse` turns each run of `unicode.IsSpace` into one U+0020.

- Every field takes part for every claim type, and each is named and quoted on
  its own. Case is kept, so `5 mW` and `5 MW` differ. Tags and `negatable` do not
  take part. UNSPECIFIED is treated as ASSERTION.
  Tests: `TestCanonicalFormV2_EveryFieldChangesTheHashForEveryType`,
  `TestCanonicalFormV2_KeepsCase`,
  `TestCanonicalFormV2_ObjectOnlyAndScopeOnlyDiffer`,
  `TestCanonicalFormV2_ClaimTypeTakesPart`,
  `TestCanonicalFormV2_DelimitersNeverCollide`.
- The form is empty when there is no structure or the trimmed subject is empty,
  and such a claim has no canonical hash.
  Tests: `TestCanonicalFormV2_EmptyWithoutStructureOrSubject`.
- Equivalent spellings (NFC or NFD, whitespace runs, padding) give one form, and
  normalisation is idempotent. Every built form is ASCII, and the lookup
  normaliser leaves it exactly as built.
  Tests: `TestCanonicalFormV2_NormalizationIsIdempotent`,
  `TestClaimIntegrity_CanonicalV2RefusesIdenticalStructure`.
- Distinct findings (another object, temporal scope, claim type or letter case)
  are admitted under their own hashes, and an identical structure is still
  refused.
  Tests: `TestClaimIntegrity_CanonicalV2SeparatesDistinctFindings`,
  `TestClaimIntegrity_CanonicalV2RefusesIdenticalStructure`.
- A declared `canonical_form` without a structure is refused with `canonical
  form requires claim structure`. A declared form must equal the derived v2 form
  after lookup normalisation. The chain always stores the derived form, never
  the declared text, so no claim can reserve another's slot.
  Tests: `TestClaimIntegrity_CanonicalFormRequiresStructure`,
  `TestClaimIntegrity_DeclaredCanonicalFormMustMatchStructure`.
- `FactByCanonical` hashes a form with the `zrn.cf.v2/` prefix verbatim, after
  NFC and trimming. Any other form keeps the v1 normalisation. The v2 form of an
  accepted claim returns its fact.
  Tests: `TestNormalizeCanonicalFormForLookup`,
  `TestClaimIntegrity_FactByCanonicalFindsV2Form`.
- Form v1 (`BuildCanonicalForm`) is unchanged, so pre-v12 records and the
  marker-off path hash exactly as in knowledge 11.
  Tests: `TestCanonicalFormV1_GoldenFormsUnchanged`,
  `TestV11_CanonicalDropsObject`, `TestV11_UnstructuredCanonicalSquat`.

## 5. Admitting a dispute

### 5.1 Contradiction admission

With the marker on, `MsgSubmitContradiction` is checked as follows, before any
collateral moves:

1. The target must stand: VERIFIED, ACTIVE, CONTESTED, CHALLENGED or AT_RISK.
   Conjectures were already refused. Otherwise the refusal reads
   `fact <id> is not standing (status: <STATUS>); …`.
2. The stake must be at least the minimum for challenging the target
   (`EffectiveMinChallengeStake`, the base `min_challenge_stake` scaled by the
   target's confidence; the base is 11 ZRN in `DefaultGenesis`). Otherwise the
   refusal reads
   `contradiction stake <s> below effective minimum <m> (fact confidence <c>; …)`.
3. The counter-claim text must not repeat a live or standing record's text in
   its domain (§3).

A refusal moves no coins and writes nothing. Admission changes no fact (§6).
With the marker off every positive stake and every target status is admitted.
Tests: `TestClaimIntegrity_ContradictionStakeMeetsChallengeMinimum`,
`TestClaimIntegrity_ContradictionTargetMustStand`,
`TestClaimIntegrity_ContradictionCannotTakeLiveClaimText`.

### 5.2 Ante gate for dispute messages

While the marker is on, the ante treats `MsgSubmitContradiction` and
`MsgChallengeProvisionalFact` exactly like `MsgChallengeFact`:

- an unregistered signer is refused with the onboard hint
  (`… is not registered with zerone_auth — one-shot fix: zeroned tx zerone_auth onboard agent --from <your-key>`);
- a registered account without `CanChallenge` is refused;
- human and agent accounts are allowed.

The refusal texts are exactly those of `MsgChallengeFact`.

The gate is read only for a tx that carries a dispute message, at most once per
tx, before the account lookup. A read error refuses the tx (`read claim
integrity gate for dispute message: …`) and is never treated as off. Simulation
never reads it. With the marker off, both messages keep the legacy
default-allow. `MsgPostConjecture` is deliberately not gated.
Tests: `TestAnteIntegration_ClaimIntegrityGateWiredToKnowledgeMarker`,
`TestAnteIntegration_ClaimIntegrityGateTreatsDisputesAsChallenges`,
`TestAnteIntegration_ClaimIntegrityGateReadErrorRejectsTx`,
`TestAnteIntegration_ClaimIntegrityGateReadLazily` (all in
`app/ante_integration_test.go`).

## 6. Verdicts and reversals

### 6.1 Fact state machine

Each row records the cause shown (§6.2).

| From | To | Trigger | Cause | Tests |
|---|---|---|---|---|
| — | VERIFIED (PROVISIONAL for a conjecture) | claim ACCEPTED | `verification` / claim ID | `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses` |
| VERIFIED, ACTIVE | CONTESTED | A claim carrying CONTRADICTS→F is **accepted**, and the edge stored from its counter-fact K to F is CONTRADICTS | `contradiction_accepted` / counter-claim ID | `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted`, `TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted`, `TestClaimIntegrity_AcceptedContestFollowsStoredEdge`, `TestContestAcceptedTargets` |
| VERIFIED, ACTIVE, AT_RISK | CONTESTED | a fact F depends on is DISPROVEN (the existing cascade, unchanged) | `cascade` / disproven fact ID | `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses`, `TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt` |
| CONTESTED | VERIFIED (PROVISIONAL for a conjecture) | a contradiction ends without acceptance (REJECT, MALFORMED, INCONCLUSIVE or starved) **and** no contest reason remains | `contradiction_reversed` / that claim's ID | `TestClaimIntegrity_PreActivationContestSettlesThroughPredicate`, `TestSettleContradictionTargetsSinceLastSurvival`, `TestClaimIntegrity_ConjectureShapedTargetReturnsToProvisional` |
| CONTESTED | VERIFIED (PROVISIONAL for a conjecture) | counter-fact K becomes DISPROVEN **and** no contest reason remains | `counterfact_disproven` / K | `TestClaimIntegrity_DisprovenCounterFactReleasesTarget`, `TestSettleCounterFactTargets` |
| VERIFIED, ACTIVE, **CONTESTED** | CHALLENGED | `MsgChallengeFact` | `challenge_opened` / challenge claim ID | `TestClaimIntegrity_ChallengeFactAdmitsContestedTarget` |
| PROVISIONAL | CHALLENGED | `MsgChallengeProvisionalFact` | `challenge_opened` / challenge claim ID | `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses` |
| CHALLENGED | DISPROVEN | the challenge is ACCEPTED | `challenge_disproven` / challenge claim ID | `TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands` |
| CHALLENGED | ACTIVE, or CONTESTED while a contest reason remains | the challenge is REJECTED (survived) | `challenge_survived` / challenge claim ID | `TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt`, `TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands`, `TestClaimIntegrity_SurvivalClearsNothingRefutedWhileTheChallengeWasOpen` |
| CHALLENGED | ACTIVE, or CONTESTED while a contest reason remains | the challenge is INCONCLUSIVE, MALFORMED or starved | `challenge_inconclusive` / challenge claim ID | `TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt`, `TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands`, `TestClaimIntegrity_SameBlockRefutationAfterSurvivalIsNotCleared` |
| CHALLENGED (conjecture) | PROVISIONAL | any ending except ACCEPT | `challenge_survived` or `challenge_inconclusive` / challenge claim ID | `TestClaimIntegrity_ConjectureChallengeEndsProvisional`, `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses` |
| metabolic statuses | — | unchanged | inferred, as in knowledge 11 | *Not tested here* |

These events change no fact:

- **Admitting a contradiction**, by either message. A counter-claim carrying
  CONTRADICTS changes nothing until it is accepted, and one that ends without
  acceptance leaves a VERIFIED target exactly as it was.
  Tests: `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted`,
  `TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted`,
  `TestClaimIntegrity_UnacceptedContradictionChangesNothing`.
- **Accepting a counter-claim, as far as the target's verdict goes.** The target
  only becomes CONTESTED, never DISPROVEN, and the counter-fact stays VERIFIED.
  Tests: `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted`.
- **Accepting an attribution** (§2.1).

DISPROVEN comes only from an accepted direct challenge of that fact. *Not
tested here* as a universal statement: read from code, where the only DISPROVEN
writes are in the two versions of `handleChallengeDisproven`.

A claim can name a target twice. The relation store keeps one edge per
(source, target) pair, and the last relation written wins. So
`[CONTRADICTS F, CITES F]` stores CITES and contests nothing.
Tests: `TestClaimIntegrity_AcceptedContestFollowsStoredEdge`.

### 6.2 Cause vocabulary

`StatusTransition.cause_event_type` and `cause_id`:

| `cause_event_type` | `cause_id` | Recorded when (marker on) | Knowledge 11 recorded |
|---|---|---|---|
| `verification` | the accepted claim | a fact is created, VERIFIED or PROVISIONAL | the same for VERIFIED; `unknown` with no ID for PROVISIONAL |
| `contradiction_accepted` | the accepted counter-claim | an acceptance contests F | `cascade` with no ID, at admission |
| `cascade` | the disproven fact | a dependency is refuted | the same |
| `contradiction_reversed` | the contradiction that ended | F is released when it ends | `verification` with F's own claim ID |
| `counterfact_disproven` | the disproven counter-fact K | F is released when K falls | nothing (F stayed CONTESTED) |
| `challenge_opened` | the challenge claim | a challenge opens | `unknown`, no ID |
| `challenge_survived` | the challenge claim | a challenge is REJECTED | `unknown`, no ID |
| `challenge_inconclusive` | the challenge claim | a challenge is INCONCLUSIVE, MALFORMED or starved | `unknown`, no ID |
| `challenge_disproven` | the challenge claim | a challenge is ACCEPTED | `challenge_disproven`, no ID |

Tests: `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses`,
`TestClaimIntegrity_ChallengeTransitionsMarkerOffKeepInferredCauses`,
`TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted`,
`TestClaimIntegrity_PreActivationContestSettlesThroughPredicate`,
`TestClaimIntegrity_DisprovenCounterFactReleasesTarget`,
`TestV11_TransitionCausesInferred`.

- A cause is attached to one fact's next status write only. Any other fact
  written in the same context, such as a cascade descendant, keeps its own
  cause.
  Tests: `TestTransitionCauseIsScopedToItsFact`,
  `TestClaimIntegrity_ChallengeTransitionsRecordExactCauses`.
- Export and import keep every transition and counter byte for byte.
  Tests: `TestClaimIntegrity_TransitionCausesSurviveExportImport`.
- Status writes that this release does not touch (metabolism, pruning and other
  paths) keep the inferred cause of knowledge 11. *Not tested here.*

### 6.3 Contest reasons

`contestReasonRemains(F, survivedAt)` decides whether F must stay CONTESTED. It
never returns an error and writes nothing.

- **R1, standing conflict.** Some incoming edge to F is CONTRADICTS, and its
  source fact has not fallen: it is VERIFIED, ACTIVE, CONTESTED, CHALLENGED,
  AT_RISK or EXPIRED. An EXPIRED counter-fact counts because patronage or
  metabolism can revive it; only SUPERSEDED, DISPROVEN, REVOKED and PRUNED
  have fallen. Other incoming types, and F's own outgoing CONTRADICTS edges,
  are not reasons.
  Tests: `TestContestReasonStandingCounterFact`,
  `TestClaimIntegrity_ExpiredCounterFactStillHoldsTarget`.
- **R2, refuted dependency.** Some outgoing edge from F is SUPPORTS, REQUIRES,
  REFINES, GENERALIZES or CITES (the cascade set), its target D is DISPROVEN,
  and `survivedAt == 0` or D's latest transition to DISPROVEN is at or after
  `survivedAt`. A fall at that height counts against F: heights cannot order
  two events of one block. A D with no recorded fall is cleared by any
  survival.
  Tests: `TestContestReasonRefutedDependencyIsOrderedBySurvival`,
  `TestClaimIntegrity_SameBlockRefutationAfterSurvivalIsNotCleared`.
- **`survivedAt`.** It is the evidence cutoff of F's last survival: the height
  the surviving challenge opened, its challenge claim's `SubmittedAtBlock`, not
  the height it ended. A refutation while that challenge was open is therefore
  never cleared by it. When a challenge of F is REJECTED, it is that
  challenge's `SubmittedAtBlock`. Otherwise (a release, or a challenge that ends
  any other way) it is read from the challenge claim that F's latest
  `challenge_survived` transition names, or 0 if there is none, so an
  inconclusive or starved challenge clears nothing. A knowledge-11 survival was
  recorded as `unknown`, so it clears nothing. A survival whose challenge claim
  is missing, unreadable, names another fact or opened after the survival makes
  the check incomplete.
  Tests: `TestContestReasonSinceLastSurvival`,
  `TestSettleContradictionTargetsSinceLastSurvival`,
  `TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt`,
  `TestClaimIntegrity_SurvivalClearsNothingRefutedWhileTheChallengeWasOpen`,
  `TestChallengeEndStatus`.
- **Order.** R2 is read first, then R1, within one budget. A reason found by
  either rule is conclusive, whatever the other could read. R1 reads every
  incoming edge of every type, which any later fact can add, so no number of
  facts citing F can hide a refuted dependency behind the bound.
  Tests: `TestContestReasonFindingIsConclusive`,
  `TestClaimIntegrity_ShieldChallengeOfHeavilyCitedFactLaundersNothing`.
- **Entries naming another fact.** F's forward-index prefix holds F's own
  edges (from its claim, from genesis, and from an augmentation whose ID is
  F's), and it can also hold edges whose source is another ID that starts
  with `<F>/`. A passing augmentation writes a REFORMULATES edge whose source
  is its user-chosen ID, so an augmentation named `<F>/0` puts one there. The
  stored key is checked against the payload, and the payload names the edge's
  owner. R2 skips an entry whose source is not F, and R1 skips an entry whose
  target is not F; neither makes the check incomplete. A skipped entry still
  counts toward the bound, so a refuted dependency can hide behind the bound
  only if F's forward prefix nears it by itself, which would take about 100,000
  passing augmentations admitted before review neutrality, which refuses new
  ones.
  Tests: `TestContestReasonSkipsForeignEdgesUnderTheFactsPrefix`,
  `TestClaimIntegrity_ShieldChallengeBehindAugmentationEdgeLaundersNothing`.
- **Failure handling.** When neither rule finds a reason, a read error, a
  malformed record, an edge naming a missing fact, or more than 100,000 records
  or 64 MiB read in one check all mean the reason remains, so no CONTESTED fact
  is released. The check emits
  `zerone.knowledge.contest_check_incomplete{fact_id, reason}`, exactly once per
  failed check; the reason names every failed read. The bound counts every
  incoming edge, of any type. At a challenge's end an incomplete check changes
  nothing (§6.5).
  Tests: `TestContestReasonFailsTowardContested`,
  `TestClaimIntegrity_IncompleteContestCheckKeepsTargetContested`,
  `TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases`.
- **Pending contradictions are not reasons.** A contradiction has no fact, and
  so no stored edge, until it is accepted. *Not tested here* as a separate
  case: no test ends one contradiction while another is still pending. Read
  from code: relation edges are written only when a fact is created.

The check reads stored edges and recorded transitions, never events and never
other pending claims. That way a fact the cascade skipped, because it was
already CONTESTED, is still held by its refuted dependency.
Tests: `TestClaimIntegrity_DisprovenCounterFactLeavesRefutedDependencyHolding`,
`TestClaimIntegrity_CascadeContestedTargetStaysContested`.

### 6.4 Contradictions take effect when accepted

- **Acceptance.** For each CONTRADICTS target that is VERIFIED or ACTIVE, and
  whose stored edge from the new counter-fact is CONTRADICTS, the target becomes
  CONTESTED and `zerone.knowledge.contradiction_accepted` is emitted. The
  counter-fact stays VERIFIED. A stored edge that cannot be read is an error,
  and nothing is written.
  Tests: `TestContestAcceptedTargets`,
  `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted`,
  `TestClaimIntegrity_RelationContradictionContestsOnlyWhenAccepted`.
- **Any other ending.** Each CONTESTED target is released (§6.1) only if no
  contest reason remains, and then `contradiction_reversed` is emitted.
  Tests: `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted` (a starved
  B leaves F CONTESTED while K stands),
  `TestClaimIntegrity_CascadeContestedTargetStaysContested`,
  `TestClaimIntegrity_PreActivationContestSettlesThroughPredicate`,
  `TestSettleContradictionTargetsSinceLastSurvival`.
  A target record that cannot be read is reported with
  `contest_check_incomplete` and left as it is, so the ending round still
  completes, and the releases are written in their own cache, so a write error
  leaves every target unchanged. *Not tested here*: read from code
  (`settleContradictionTargets`); the counter-fact path's equivalent is tested
  in §6.5.
- **Legacy-policy rounds.** A round admitted under the legacy review policy that
  expires in BeginBlock settles its targets through the same check. A marker
  read error there leaves every target unchanged and does not halt the chain.
  Tests: `TestClaimIntegrity_LegacyPolicyStarvedContradictionSettles`.
- **Conjectures.** A released conjecture returns to PROVISIONAL. With the marker
  on, a conjecture cannot be contradicted or contested by an acceptance, so only
  legacy records reach this.
  Tests: `TestClaimIntegrity_ConjectureShapedTargetReturnsToProvisional`,
  `TestContestAcceptedTargets`.

With the marker off, knowledge 11 contests the target at admission and restores
it when the contradiction fails, whatever else holds it.
Tests: `TestV11_StarvedContradictionRestoresUpheldTarget`,
`TestSubmitContradiction_StarvedDoesNotLockFact`,
`TestSubmitClaim_ContradictionAutoContests`, and the marker-off halves of the
tests above.

### 6.5 Participants decide the target

- **Challenging a CONTESTED fact.** `MsgChallengeFact` accepts VERIFIED, ACTIVE
  and CONTESTED targets, at the same confidence-scaled minimum stake. Any other
  status is refused with `… (commitment 4: only verified, active or contested
  facts accept probes; …)`. A refusal moves no coins and writes nothing. With
  the marker off a CONTESTED fact is refused as in knowledge 11.
  Tests: `TestClaimIntegrity_ChallengeFactAdmitsContestedTarget`.
- **Ending a challenge of F.** ACCEPT makes F DISPROVEN. Any other ending returns
  F to ACTIVE, or to CONTESTED while a contest reason remains (`survivedAt` as in
  §6.3). A survival never clears a standing counter-fact, so the counter-fact
  must itself be refuted, and never clears a dependency refuted while it was
  open. A starved challenge launders nothing, however many facts cite F.
  Tests: `TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands`,
  `TestClaimIntegrity_SurvivalClearsOnlyDependenciesRefutedBeforeIt`,
  `TestClaimIntegrity_SurvivalClearsNothingRefutedWhileTheChallengeWasOpen`,
  `TestClaimIntegrity_ShieldChallengeOfHeavilyCitedFactLaundersNothing`,
  `TestClaimIntegrity_SameBlockRefutationAfterSurvivalIsNotCleared`,
  `TestClaimIntegrity_LaunderedFactChallengeFollowsStandingCounterFact`.
- **A counter-fact falls.** When K becomes DISPROVEN, in the same cache and
  after the DISPROVEN write and the cascade, each CONTESTED target of K's stored
  CONTRADICTS edges is released if no contest reason remains. Another standing
  counter-fact, or a dependency refuted since F's last survival, keeps F
  CONTESTED. The release emits `contradiction_reversed` with
  `reverted_by_fact = K`.
  Tests: `TestClaimIntegrity_DisprovenCounterFactReleasesTarget`,
  `TestClaimIntegrity_DisprovingOneOfTwoCounterFactsKeepsTargetContested`,
  `TestClaimIntegrity_DisprovenCounterFactLeavesRefutedDependencyHolding`,
  `TestSettleCounterFactTargets`.
- **Failures on these paths never release.** A challenged fact whose check fails
  keeps its standing from before the challenge: it returns to CONTESTED if its
  `challenge_opened` transition for this challenge has prior status CONTESTED,
  or if that transition is absent or unreadable, and to ACTIVE otherwise.
  `contest_check_incomplete` names it once. So one refunded, starved challenge
  cannot freeze CONTESTED a fact that has no refuted dependency and whose check
  is too large to finish, and an incomplete check releases nothing. A refuted
  dependency is read before the incoming edges, and entries naming another
  fact are skipped (§6.3), so it is lost this way only if F's forward prefix
  alone exceeds the bound. Otherwise the cost is limited to R1: a fact against
  which a counter-fact was accepted while it was CHALLENGED, or a fact
  laundered before activation, returns to ACTIVE if its check also cannot
  finish. The residual: a fact
  whose check is too large to finish, once CONTESTED, never returns to
  VERIFIED or ACTIVE, even after surviving a challenge opened after its
  refutation, because every release check reads the incoming edges; only a
  challenge that refutes it moves it. If K's edges
  cannot be listed, no target is released and `contest_check_incomplete` names
  K. A target that cannot be read is reported and left as it is, while the
  other targets still settle. The disproval stands in every case. A write error
  is returned, and the round's cache discards it.
  Tests: `TestClaimIntegrity_IncompleteCheckOnVerdictPathsNeverReleases`,
  `TestClaimIntegrity_IncompleteCheckAtChallengeEndKeepsPriorStanding`,
  `TestChallengeEndStatus`,
  `TestSettleCounterFactTargetsFailsTowardContested`.
- **Conjectures.** A challenge ending that does not refute a conjecture returns
  it to PROVISIONAL, never ACTIVE or CONTESTED. A CONTESTED conjecture left by a
  legacy record can be challenged at the unscaled minimum, and is released to
  PROVISIONAL when its counter-fact falls.
  Tests: `TestClaimIntegrity_ConjectureChallengeEndsProvisional`.
- Stakes, refunds and survival rewards on challenge endings are unchanged.
  *Not tested here*: read from code; this release changes only the status a
  challenged fact returns to and its recorded cause.

With the marker off, nothing re-evaluates F when K falls, and F stays CONTESTED.
Tests: `TestClaimIntegrity_DisprovenCounterFactReleasesTarget`.

## 7. Conflicted reviewers

With the marker on, `StoreCommitmentInRound` refuses a commitment from a party
to the round's claim. The check runs after the phase and scheme checks and
before the duplicate and equivocation lookup. It covers both
`MsgSubmitCommitment` and commitments injected from vote extensions. The parties
are:

- the claim's submitter;
- the defender of the one fact the claim disputes, meaning that fact's
  submitter. The disputed fact is the claim's `provisional_fact_id`
  (`MsgChallengeFact` or `MsgChallengeProvisionalFact`). Failing that, it is the
  single CONTRADICTS target of a `CHALLENGE_DEPOSIT` claim with exactly one
  relation, which only `MsgSubmitContradiction` creates.

A `MsgSubmitClaim`'s relation list, chosen by its submitter, is never read, so
naming facts recuses none of their defenders.
Tests: `TestClaimIntegrity_SubmitterAndDefenderCannotReview`,
`TestClaimIntegrity_DisputeDefenderCannotReview`,
`TestClaimIntegrity_SubmitClaimRelationsRecuseNoDefender`, `TestDisputedFactID`.

- Two addresses are one actor when the strings are equal and non-empty, or when
  both decode as bech32 to the same non-empty bytes, whatever the prefix or
  case. So an operator (valoper) or upper-case encoding of the submitter's key is
  refused.
  Tests: `TestClaimIntegrity_PartyMatchesEveryEncodingOfItsKey`, `TestSameActor`.
- Refusal: `reviewer <r> is a party to claim <id> (submitter)` or
  `(defender of fact <F>)`, followed by `: verifier not qualified for domain`.
  Only non-parties complete the round and are paid.
  Tests: `TestClaimIntegrity_SubmitterAndDefenderCannotReview`.
- On the vote-extension path a refused commitment is skipped and logged at
  Debug level (`skipped commitment from vote extension`, with the refusal text as
  `reason`), and the block goes on. In this source a release latch keeps that
  path closed, so the test drives its per-commitment helper directly.
  Tests: `TestClaimIntegrityVoteExtensionSkipsPartyCommitment`
  (`app/claim_integrity_review_test.go`).
- An unreadable claim, an unreadable disputed fact or a malformed marker
  refuses the commitment and writes nothing. An absent record names no party: a
  legacy round may have no claim, and facts are never deleted on chain, so an
  absent disputed fact means store damage. In that case the submitter is still
  refused.
  Tests: `TestClaimIntegrity_ReviewerPartyCheckFailsClosed`,
  `TestClaimIntegrity_AbsentDisputedFactNamesNoDefender`.
- Reveals are not guarded, so a commitment made before activation still reveals
  and settles.
  Tests: `TestClaimIntegrity_PreActivationSelfCommitStillSettles`.

With the marker off, self-review is allowed, as in knowledge 11.
Tests: `TestV11_SubmitterMayReviewOwnClaim`, and the marker-off halves of the
tests above.

## 8. Events

Full attribute lists are in [`docs/EVENTS.md`](../EVENTS.md); its completeness is
checked by `TestEventAudit_DocumentationCompleteness`.

| Event | Marker-on meaning | Tests |
|---|---|---|
| `zerone.knowledge.contradiction_accepted{fact_id, counter_claim_id, counter_fact_id}` | New. An accepted contradiction contested `fact_id` (§6.4). | `TestContestAcceptedTargets` |
| `zerone.knowledge.contradiction_reversed{fact_id, reverted_by_claim}` | `fact_id` was released when that contradiction ended without acceptance and no contest reason remained. It is not emitted while a reason remains. | `TestClaimIntegrity_PreActivationContestSettlesThroughPredicate`, `TestClaimIntegrity_ContradictionContestsOnlyWhenAccepted` |
| `zerone.knowledge.contradiction_reversed{fact_id, reverted_by_fact}` | New form. `fact_id` was released when counter-fact `reverted_by_fact` fell. | `TestClaimIntegrity_DisprovenCounterFactReleasesTarget`, `TestSettleCounterFactTargets` |
| `zerone.knowledge.contest_check_incomplete{fact_id, reason}` | New. A contest check could not finish, so no CONTESTED fact is released; at a challenge's end the fact keeps its standing from before the challenge (§6.5). `fact_id` is that fact, or a fallen counter-fact whose edges could not be listed. | `TestContestReasonFailsTowardContested`, `TestSettleCounterFactTargetsFailsTowardContested`, `TestClaimIntegrity_IncompleteCheckAtChallengeEndKeepsPriorStanding` |
| `zerone.knowledge.challenge_fact` | Unchanged attributes. A CONTESTED fact can now be challenged, so `fact_id` may name one. | `TestClaimIntegrity_ChallengeFactAdmitsContestedTarget` |
| `zerone.knowledge.challenge_inconclusive_restored{fact_id, challenge_claim_id}` | The fact may return to CONTESTED rather than ACTIVE. | `TestClaimIntegrity_ContestedFactChallengeWhileCounterFactStands` |

## 9. What consumers read

- **The current verdict on a fact.** Read `fact.status`, its status transitions
  with their causes (§6.2), and its incoming CONTRADICTS edges together with each
  source fact's status.
- **The evidence behind a verdict.** `ClaimHistory(claim_id)` gives the rounds,
  commitments, reveals and attestations (who reviewed, the vote, the method, the
  reason and the evidence hashes). It also gives the related claims, linked by
  `provisional_fact_id`, `challenged_claim_id`, `relations.contradicts`,
  `content_hash`, `canonical_hash` and `structure.object` (§2.5).
- **Who said P.** Read `ClaimHistory(P's claim ID)` and take the related claims
  linked by `structure.object`. Each is an attribution claim with its own
  fidelity verdict.
- **A proposition by its structure.** Use `FactByCanonical` with the v2 form
  (§4).

| Status | Meaning |
|---|---|
| VERIFIED, ACTIVE | currently accepted |
| CONTESTED | Accepted, but a standing accepted finding conflicts with it, or a fact it depends on was refuted. The transition cause says which. It does **not** mean false. |
| CHALLENGED | under adjudication |
| DISPROVEN | refuted by a direct challenge |
| AT_RISK, EXPIRED, PRUNED | metabolic, never epistemic |

A pending contradiction appears only in `ClaimHistory`. It is an allegation,
not a verdict. A panel verdict proves neither truth nor reviewer independence.

### 9.1 Reading legacy records (never rewritten)

Before activation height H:

- A transition recorded as `cascade` with an empty `cause_id` was a
  contradiction flip at admission.
- A transition recorded as `verification` right after CONTESTED, carrying the
  fact's own claim ID, was a contradiction reversal.
- A conjecture's creation (UNSPECIFIED→PROVISIONAL) was recorded as `unknown`
  with an empty `cause_id`. Its claim is the fact's own `claim_id`.
- Challenge transitions were recorded as `unknown`, and `challenge_disproven`
  had an empty `cause_id`.
- Two VERIFIED facts joined by CONTRADICTS are legacy laundering. Treat them as
  an unresolved dispute; either side can now be challenged.

Tests: `TestV11_TransitionCausesInferred`,
`TestClaimIntegrity_ChallengeTransitionsMarkerOffKeepInferredCauses`,
`TestClaimIntegrity_PreActivationContestSettlesThroughPredicate`,
`TestV11_StarvedContradictionRestoresUpheldTarget`,
`TestClaimIntegrity_LaunderedFactChallengeFollowsStandingCounterFact`.

## 10. What the chain still cannot prevent

These are disclosures, not behaviour this release adds. *Not tested here.*

- **One controller holding many DIDs.** Registration is permissionless, and
  there is no controller field. For `CHALLENGE_DEPOSIT` disputes, sybil review
  costs nothing net: 45% is refunded whatever the verdict, and the 55% review
  budget pays the sybils. For `REVIEW_FEE` claims, sybil self-review loses 45% of
  the fee. The party check (§7) excludes only the same key, in any encoding, and
  the single defender.
- **Deliberately starved disputes.** A starved challenge or contradiction is
  refunded in full, so its cost is locked capital and gas. After activation a
  starved contradiction changes no fact's status, and a starved challenge
  clears no contest reason (§6.1, §6.3).
- **Mempool copying of claim text.**
- **Verbatim quotes from third-party clients.** Only reviewers voting MALFORMED
  push back on chain (§2.7).
- **Reviewer independence, and the truth of anything a panel accepts.**

## 11. Decisions and scope

The defaults adopted on 2026-10-16:

- Contradictions take effect when accepted, not when admitted (§6.4).
- An accepted contrary finding is asymmetric: the target becomes CONTESTED and
  the counter-fact stays VERIFIED. A target that survives a direct challenge
  stays CONTESTED while the counter-fact stands (§6.5).
- No-verdict, negative-verdict and fallen records all release text for
  resubmission (§3).
- The Hermes contract: the attribution `object` is the on-chain content hash,
  records are JCS plus sha256 with a `sha256:` prefix, plus the
  `reasoning_trace` keys and locator grammar of §2.2.
- No verbatim text on chain, an 8-token guard threshold, and private persons
  only as opaque `hermes:` IDs (§2.7).
- `M-HISTORICAL` is the attribution methodology.

Out of scope for this release: any `zerone-dev-1` or `zerone-1` upgrade or
deploy tooling; query energy reaching metabolism; `method_id` on
`MsgSubmitContradiction`; gating `MsgPostConjecture`; identity limits beyond
same-key and single-defender exclusion; dispute economics (refunds, cooldowns);
metabolic re-checks of CONTESTED; exempting attribution facts from niche
competition or metabolism; challenging AT_RISK, EXPIRED or PRUNED facts;
repairing legacy index pointers; on-chain quote detection; a client-side
standing fold or a `PropositionStanding` query; and rewriting historical
records.
