# Constructive-intelligence tree: design and Season 0

Date: 2026-07-29

Status: implemented as a static, non-authoritative experiment

Branch: `feat/constructive-intelligence-tree-v1`

Normative contract:
[`../specs/constructive-intelligence-tree-v1.md`](../specs/constructive-intelligence-tree-v1.md)

## Outcome

Zerone should reward verified improvements in assurance, not
self-declarations of brilliance.

The first implementation is therefore a static cryptography and
industrial-protocol skill DAG with three sponsor-milestone quest templates. It
deliberately does not change consensus, grant qualification, activate rewards,
authorize testing, or publish confidential evidence.

The key split is:

```text
capability graph: who has demonstrated eligibility to attempt/review work
artifact graph:   what a frozen result proves, repairs, reproduces or deploys
```

“Breakthrough” is derived only after effectively independent reproduction,
prior-art comparison and adoption or descendant impact. It is not a message
field, claim status, governance vote, or multiplier.

## Why this shape fits Zerone

Zerone already has partial organs:

- ontology domains and curriculum-depth heuristics;
- domain qualifications with expiry and decay;
- knowledge claims, challenges and reasoning/provenance records;
- sponsor-funded fact bounties;
- substrate-bridge adapters for external evidence; and
- truth-linked vesting categories.

They are not ready to carry high-value cryptographic rewards:

- ontology proposals do not yet make parent/depth a reliable writable tree;
- qualification “correctness” is agreement with panel outcome, not external
  truth;
- historical dissent frequency is used as a proxy for validator
  independence;
- replication, corroboration and citation are raw counters rather than unique
  evidence receipts;
- time releases vesting even when no new evidence arrives; and
- private vulnerability evidence has no embargo-safe lifecycle.

The July slim cut also made the architectural decision: project trees,
discovery, research listings, teams and payment orchestration live in
AgentTool. Reintroducing an omnibus on-chain breakthrough module would reverse
that decision without solving the evidence problem.

## Seed allocation

The v1 seed has exactly 30 nodes:

- seven mathematical, systems and security foundations;
- six cryptographic primitive nodes;
- six assurance-method nodes;
- eight industrial-protocol nodes; and
- three Season 0 quests.

The emphasis is deliberate:

- approximately 80% deployed protocols and assurance;
- approximately 15% migration and formal methods; and
- at most 5% novel-primitive exploration outside production.

Toy cryptography can demonstrate qualification evidence. Production quests
normally improve maintained libraries through proofs, vectors, wrappers,
interoperability, migration and upstream repairs.

## Evidence before economics

The Season 0 ladder is:

```text
E0 commit and frozen prior art
E1 inspectable reproducible bundle
E2 class verification
E3 effective independent reproduction
E4 adversarial survival or confidential fix/mitigation testing
E5 upstream/adoption evidence
E6 maintenance evidence
```

The nominal milestone pool preserves 15% for E2, 20% for E3, 15% for E4, 25%
for E5, 10% for E6, and 15% for challenges/remediation. E1 verified costs sit
outside that percentage pool but inside the prospectively funded all-in escrow
cap. Novelty, impact, and maintenance shape E3, E5, and E6 respectively; they
are not extra bonuses. The all-in escrow also budgets reviewers and fees, and
every payment plus refund must conserve that funded amount. E1 uses
preapproved cost classes, market-rate ceilings, third-party receipts,
related-party disclosure, and its own sub-cap. No time-only or over-cap release
exists.

Correctness, novelty, impact and maintenance remain separate. Safety is a hard
zero/one gate per recipient, artifact, and action. A claimant result that
violates authorization or disclosure policy receives no claimant reward or
cost reimbursement regardless of technical quality, while a compliant neutral
challenger or repairer can still draw only the bounded reserve.

Review independence is measured through conflict/control clusters rather than
address or vote counts. High-stakes work needs three effective clusters, two
organizational roots, two implementation/toolchain roots and two execution
environments. Reviewers are paid for faithful method execution whether they
confirm or contradict.

The static tree never stores live cases. A live bounty uses a separate
append-only evidence record with exact standard and policy hashes, role
separation, funded escrow, conflict-cluster quorum, and a deliverable key
derived from scope plus stable canonical subject roots. Rebuilds, wrappers,
forks, scope splits, and other derivatives declare provenance and overlap,
link the prior deliverable, and earn only a separately reviewed delta. A
global source-event consumption ledger prevents the same upstream merge,
checker run, or release from being reissued under a fresh evidence ID. This
prevents a new address, explanation, or duplicate bounty from resetting reward
eligibility. Deliberately planted or knowingly retained defects,
self-controlled adoption, and concealed causal involvement are disqualifying.

The validator digest-pins the normative v1 policy and all 30 node templates,
including separate quest pins. Standards status text and review dates are a
separate refreshable snapshot. Any active-use consumer fails closed after a
required pin's `reviewAfter` date, and a funded case additionally binds fetched
bytes, a signed release digest, or a commit SHA rather than trusting a movable
version URL.

## Responsible-disclosure switch

Every quest carries a prepublication quarantine and the same disclosure
switch:

> If the work reveals a previously unknown result capable of changing an
> adversarial accept/reject decision in a deployed implementation, public work
> stops and the artifact enters private coordinated repair.

Unexpected failures and counterexamples are triaged before public logs. Exploit
plaintext and target-identifying confidential metadata do not enter a public
ledger during embargo. The private workflow records encrypted evidence,
threshold reviewer receipts, deadlines, tested repair and eventual sanitized
disclosure. A vendor may reproduce but cannot veto validity or payout.

## Season 0

### 1. RFC 9846 KeyShare reuse

Turn the security-relevant RFC 9846 delta into executable negative and
conformance tests across at least three independent TLS implementations. The
RFC 9846 sender must not reuse a KeyShare across connections; a receiver must
permit peer reuse for interoperability. Record each stack commit's claimed RFC
profile so RFC 8446-only behavior is an adoption gap rather than falsely
labeled RFC 9846 nonconformance.

### 2. FIPS 203/204/205 cross-library assurance

Build a cross-library corpus covering known answers, malformed encodings,
errata, and constant-time evidence with three implementation roots per
algorithm. Test implicit rejection for ML-KEM; test invalid-signature rejection
and signing/randomness behavior for ML-DSA and SLH-DSA. Freeze the 2026-07-29
FIPS/planning-note oracle and keep ACVP-style evidence distinct from FIPS 140
certification.

### 3. MLS state-machine invariants

Bind a formal or executable state model to RFC 9420, frozen RFC 9180 HPKE, and
`mlswg/mls-implementations@cfd450286d1bfd9cd2519b95c80f9771f94a5b1a`.
Cover bounded group, epoch, proposal, and trace limits plus per-transition
minimum cases. State all attacker assumptions and distinguish bounded
exploration, protocol proof, and implementation proof.

## Promotion sequence

1. Publish and validate the static v1 tree.
2. Define unique typed evidence receipts, canonical subject lineage, global
   source replay protection, funded escrow conservation, and refund
   accounting.
3. Run an unfunded shadow quest in AgentTool and audit duplication, conflict
   clustering, review assignment, disclosure escalation, and derivative
   overlap.
4. Repair qualification bootstrapping/version decay and verify every claimant,
   reviewer, challenge, remediation, fee, and refund accounting path.
5. Run one capped sponsor-funded pilot only after those controls pass.
6. Complete the supported Cosmos SDK/IBC migration before adding further
   consensus or IBC surface.
7. Propose only the smallest useful on-chain projection, with explicit
   economic and doctrine analysis.

The absence of those gates means “keep experimenting off-chain,” not “launch a
partially trusted reward mechanism.”

## Repository artifacts

- Human contract:
  `docs/specs/constructive-intelligence-tree-v1.md`
- Served seed:
  `dashboard/public/standards/constructive-intelligence-tree.v1.json`
- Validator:
  `dashboard/scripts/validate-constructive-intelligence-tree.mjs`
- Mutation tests:
  `dashboard/scripts/constructive-intelligence-tree.test.mjs`
- Build integration:
  `dashboard/package.json` → `check:tree`

No protobuf, keeper, store, genesis, doctrine, or deployment file changes are
part of Season 0.
