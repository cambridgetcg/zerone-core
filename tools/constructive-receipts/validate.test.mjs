import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";
import {
  RECEIPT_BUNDLE_MAX_BYTES,
  RECEIPT_DECISION_SCHEMA,
  RECEIPT_KEYS,
  canonicalJson,
  deriveCanonicalEconomics,
  deriveConsumptionKey,
  deriveDeliverableKey,
  deriveEvidenceId,
  derivePolicyRevisionDigest,
  deriveQuestNodeNormativeDigest,
  deriveStandardPins,
  deriveStandardPinsDigest,
  digestCanonical,
  parseConstructiveReceiptBundle,
  validateConstructiveReceiptBundle,
} from "./validate.mjs";

const HERE = fileURLToPath(new URL(".", import.meta.url));
const REPOSITORY = resolve(HERE, "../..");
const TREE_PATH = resolve(
  REPOSITORY,
  "dashboard/public/standards/constructive-intelligence-tree.v1.json",
);
const BUNDLE_PATH = resolve(
  HERE,
  "testdata/valid-tls-e3-bundle.v0.json",
);
const POLICY_PATH = resolve(
  HERE,
  "testdata/zero-value-tls-policy.v0.json",
);
const VALIDATOR_PATH = resolve(HERE, "validate.mjs");
const TREE_RAW = readFileSync(TREE_PATH, "utf8");
const TREE = JSON.parse(TREE_RAW);
const BUNDLE_RAW = readFileSync(BUNDLE_PATH, "utf8");
const BUNDLE = JSON.parse(BUNDLE_RAW);
const POLICY = JSON.parse(readFileSync(POLICY_PATH, "utf8"));

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function refreshEvidenceIds(bundle) {
  for (const receipt of bundle.receipts) {
    receipt.evidence_id = deriveEvidenceId(receipt);
  }
  return bundle;
}

function refreshBindings(bundle) {
  const policyDigest = derivePolicyRevisionDigest(bundle.policyRevision);
  const deliverableKey = deriveDeliverableKey({
    standards_reference_and_revision: bundle.policyRevision.standards,
    scope_hash: bundle.policyRevision.scope_hash,
    acceptance_policy_digest: bundle.policyRevision.acceptance_policy_digest,
    canonical_subject_roots: bundle.policyRevision.canonical_subject_roots,
  });
  for (const receipt of bundle.receipts) {
    receipt.immutable_bounty_and_policy_revision_digest = policyDigest;
    receipt.deliverable_key = deliverableKey;
  }
  return refreshEvidenceIds(bundle);
}

function validateObject(bundle) {
  return validateConstructiveReceiptBundle(TREE_RAW, JSON.stringify(bundle));
}

function assertRejected(bundle, pattern) {
  assert.throws(() => validateObject(bundle), pattern);
}

test("canonical TLS E3 fixture is structurally valid but never authoritative or valuable", () => {
  const decision = validateConstructiveReceiptBundle(TREE_RAW, BUNDLE_RAW);
  assert.equal(decision.schema, RECEIPT_DECISION_SCHEMA);
  assert.equal(decision.structurallyValid, true);
  for (const field of [
    "authoritative",
    "networkObserved",
    "rewardBearing",
    "qualificationGranted",
    "evidenceAccepted",
    "fundsMoved",
    "integrationAuthorized",
  ]) {
    assert.equal(decision[field], false, field);
  }
  assert.deepEqual(
    new Set(Object.values(decision.releaseBoundary)),
    new Set([false]),
  );
  assert.deepEqual(decision.valueBoundary, {
    denomination: null,
    totalCap: 0,
    verifiedCostBudget: 0,
    outcomePool: 0,
    reviewerBudget: 0,
    administrationAndFeeBudget: 0,
    escrowReceipt: null,
    scheduleInstantiated: false,
    expiry: null,
    refundPath: null,
  });
  assert.equal(decision.checkedForUseAt, "2026-07-30");
  assert.equal(decision.evidenceLevel, "E3");
  assert.equal(decision.receiptCount, 3);
  assert.equal(decision.submittedReceiptCount, 3);
  assert.equal(decision.supersededReceiptCount, 0);
  assert.deepEqual(decision.independence, {
    requiredEffectiveClusters: 3,
    componentCount: 3,
    effectiveClusters: 3,
    effectiveQualityMicros: 3_000_000,
    organizationRootCount: 2,
    implementationRootCount: 3,
    executionEnvironmentCount: 2,
    uniqueCaseCount: 12,
  });
  assert.equal(decision.canonicalEconomics.metadataOnly, true);
  assert.equal(decision.canonicalEconomics.milestone.rewardBps, 2000);
});

test("standalone zero-value policy is exactly the bundle policy", () => {
  assert.deepEqual(POLICY, BUNDLE.policyRevision);
  assert.equal(POLICY.value_boundary.denomination, null);
  assert.equal(POLICY.value_boundary.escrow_receipt, null);
  assert.equal(POLICY.value_boundary.schedule_instantiated, false);
  for (const field of [
    "total_cap",
    "verified_cost_budget",
    "outcome_pool",
    "reviewer_budget",
    "administration_and_fee_budget",
  ]) {
    assert.equal(POLICY.value_boundary[field], 0);
  }
});

test("canonical derivations are stable goldens", () => {
  const quest = TREE.nodes.find(
    (node) => node.id === "quest-tls-rfc9846-keyshare-reuse@1",
  );
  assert.equal(
    deriveQuestNodeNormativeDigest(quest),
    "bcefb7c2d177c79d135722bf38a689d122fe564eb39ebec873b0020dacb46206",
  );
  assert.equal(
    digestCanonical(quest.acceptance),
    "ed3163a6a2fc321dbe4174f85f746285d93dc83617bee57b9d1f82e46850a60b",
  );
  assert.equal(
    digestCanonical(quest.standards),
    "8a8bfc542b2f894e14478db834331bc1a210a6a358bebf2d7dc1e6ad1fb1c2a5",
  );
  assert.equal(
    deriveStandardPinsDigest(quest),
    "1f415a0c95a1d5be61563c331b8cde26d16193b0c7ef10190c883852322bb666",
  );
  assert.deepEqual(deriveStandardPins(quest), POLICY.standards);
  assert.equal(
    derivePolicyRevisionDigest(POLICY),
    "09671a857ffc9832d7cce4797f34e3b793d0b6caa18a04c040991aac2fd91a7b",
  );
  assert.equal(
    deriveDeliverableKey({
      standards_reference_and_revision: POLICY.standards,
      scope_hash: POLICY.scope_hash,
      acceptance_policy_digest: POLICY.acceptance_policy_digest,
      canonical_subject_roots: POLICY.canonical_subject_roots,
    }),
    "4d60b8baadbde19bc0b49e9cfef966c8d8f13183af3097200089c39b72496a41",
  );
  assert.deepEqual(
    BUNDLE.receipts.map(deriveEvidenceId),
    BUNDLE.receipts.map((receipt) => receipt.evidence_id),
  );
  assert.equal(
    canonicalJson({ z: 0, a: [{ y: false, x: null }] }),
    '{"a":[{"x":null,"y":false}],"z":0}',
  );
});

test("all E0-E6 economics are copied as non-operative tree metadata", () => {
  const expected = [
    ["E0", 0, "precedence-only"],
    ["E1", 0, "verified-cost-only"],
    ["E2", 1500, "milestone"],
    ["E3", 2000, "milestone"],
    ["E4", 1500, "milestone"],
    ["E5", 2500, "milestone"],
    ["E6", 1000, "milestone"],
  ];
  for (const [level, rewardBps, treatment] of expected) {
    const economics = deriveCanonicalEconomics(TREE, level);
    assert.equal(economics.milestone.rewardBps, rewardBps);
    assert.equal(economics.milestone.treatment, treatment);
    assert.equal(economics.challengeReserveBps, 1500);
    assert.equal(economics.metadataOnly, true);
  }
});

test("decision ordering is deterministic across receipt order", () => {
  const reordered = clone(BUNDLE);
  reordered.receipts.reverse();
  const originalDecision = validateConstructiveReceiptBundle(TREE_RAW, BUNDLE_RAW);
  const reorderedDecision = validateObject(reordered);
  assert.deepEqual(
    reorderedDecision.consumptionKeys,
    originalDecision.consumptionKeys,
  );
  assert.deepEqual(reorderedDecision.independence, originalDecision.independence);
  assert.equal(
    JSON.stringify(reorderedDecision),
    JSON.stringify(originalDecision),
  );
});

test("receipt shape is the exact canonical 22-field record", () => {
  assert.equal(RECEIPT_KEYS.length, 22);
  assert.deepEqual(Object.keys(BUNDLE.receipts[0]), RECEIPT_KEYS);
});

test("active-use validation uses the explicit bundle date and fails stale snapshots", () => {
  const stale = clone(BUNDLE);
  stale.checkedForUseAt = "2026-08-06";
  assertRejected(stale, /expired before active-use date 2026-08-06/);
});

test("non-quest nodes cannot be receipt targets", () => {
  const nonquest = clone(BUNDLE);
  nonquest.treeBinding.questNodeId = "math-proofcraft@1";
  assertRejected(nonquest, /must select a sponsor-milestone quest/);
});

test("tree, quest, scope, acceptance, snapshot and standards pins are exact", async (t) => {
  const mutations = [
    ["tree document", (b) => (b.treeBinding.treeDocumentDigest = "0".repeat(64))],
    ["tree normative", (b) => (b.treeBinding.treeNormativeDigest = "0".repeat(64))],
    ["quest normative", (b) => (b.treeBinding.questNodeNormativeDigest = "0".repeat(64))],
    ["scope", (b) => (b.treeBinding.scopeHash = "0".repeat(64))],
    ["acceptance", (b) => (b.treeBinding.acceptancePolicyDigest = "0".repeat(64))],
    ["snapshot", (b) => (b.treeBinding.standardsSnapshotDigest = "0".repeat(64))],
    [
      "receipt standard",
      (b) => {
        b.receipts[0].standards_reference_and_revision[0].revision = "wrong";
        refreshEvidenceIds(b);
      },
    ],
  ];
  for (const [name, mutate] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle);
      assertRejected(bundle, /does not match|canonical standard pin/);
    });
  }
});

test("policy, deliverable and evidence digest mismatches fail closed", async (t) => {
  const mutations = [
    [
      "policy binding",
      (b) =>
        (b.receipts[0].immutable_bounty_and_policy_revision_digest =
          "0".repeat(64)),
    ],
    ["deliverable", (b) => (b.receipts[0].deliverable_key = "0".repeat(64))],
    ["evidence", (b) => (b.receipts[0].evidence_id = "0".repeat(64))],
    [
      "adapter",
      (b) => {
        b.receipts[0].method_or_adapter_digest = "0".repeat(64);
        refreshEvidenceIds(b);
      },
    ],
  ];
  for (const [name, mutate] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle);
      assertRejected(bundle, /does not match/);
    });
  }
});

test("source consumption keys prevent replay independently of evidence IDs", () => {
  const replay = clone(BUNDLE);
  replay.receipts[1].source_system = replay.receipts[0].source_system;
  replay.receipts[1].source_record_or_event_id =
    replay.receipts[0].source_record_or_event_id;
  replay.receipts[1].source_revision = replay.receipts[0].source_revision;
  refreshEvidenceIds(replay);
  assert.notEqual(replay.receipts[0].evidence_id, replay.receipts[1].evidence_id);
  assert.equal(
    deriveConsumptionKey(replay.receipts[0]),
    deriveConsumptionKey(replay.receipts[1]),
  );
  assertRejected(replay, /reuses a source-system consumption key/);
});

test("duplicate evidence IDs are rejected", () => {
  const duplicate = clone(BUNDLE);
  duplicate.receipts[1] = clone(duplicate.receipts[0]);
  assertRejected(duplicate, /duplicate evidence IDs/);
});

test("related control roots collapse effective independence transitively", () => {
  const collapsed = clone(BUNDLE);
  collapsed.receipts[0].conflict_disclosures.related_control_roots = [
    "control:evaluator:a",
    "control:shared:ab",
  ];
  collapsed.receipts[1].conflict_disclosures.related_control_roots = [
    "control:evaluator:b",
    "control:shared:ab",
    "control:shared:bc",
  ];
  collapsed.receipts[2].conflict_disclosures.related_control_roots = [
    "control:evaluator:c",
    "control:shared:bc",
  ];
  refreshEvidenceIds(collapsed);
  assertRejected(collapsed, /effective independent quality is below 3/);
});

test("quality is capped at one full cluster per related component", () => {
  const partial = clone(BUNDLE);
  partial.receipts[2].result.quality_micros = 999_999;
  refreshEvidenceIds(partial);
  assertRejected(partial, /effective independent quality is below 3/);
});

test("zero or partial-quality decorator receipts cannot inflate diversity floors", () => {
  const padded = clone(BUNDLE);
  for (const receipt of padded.receipts) {
    receipt.organization_or_control_root = "org:tls-lab:alpha";
    receipt.implementation_or_toolchain_root = "impl:tls:alpha";
    receipt.execution_environment_digest = "a".repeat(64);
    receipt.evidence_level_and_scope.case_ids = [
      "case-001",
      "case-002",
      "case-003",
      "case-004",
    ];
  }
  const paddingOne = clone(padded.receipts[1]);
  paddingOne.source_record_or_event_id = "run:padding-b";
  paddingOne.organization_or_control_root = "org:tls-lab:beta";
  paddingOne.implementation_or_toolchain_root = "impl:tls:beta";
  paddingOne.execution_environment_digest = "b".repeat(64);
  paddingOne.evidence_level_and_scope.case_ids = [
    "case-005",
    "case-006",
    "case-007",
    "case-008",
  ];
  paddingOne.result.quality_micros = 0;

  const paddingTwo = clone(padded.receipts[2]);
  paddingTwo.source_record_or_event_id = "run:padding-c";
  paddingTwo.organization_or_control_root = "org:tls-lab:beta";
  paddingTwo.implementation_or_toolchain_root = "impl:tls:gamma";
  paddingTwo.execution_environment_digest = "b".repeat(64);
  paddingTwo.evidence_level_and_scope.case_ids = [
    "case-009",
    "case-010",
    "case-011",
    "case-012",
  ];
  paddingTwo.result.quality_micros = 999_999;

  padded.receipts.push(paddingOne, paddingTwo);
  refreshEvidenceIds(padded);
  assertRejected(padded, /fully supported organization\/control roots/);
});

test("organization, implementation, environment, cases and checker floors fail independently", async (t) => {
  const mutations = [
    [
      "organization",
      (b) => {
        for (const receipt of b.receipts) {
          receipt.organization_or_control_root = "org:tls-lab:only";
        }
      },
      /organization\/control roots/,
    ],
    [
      "implementation",
      (b) => {
        for (const receipt of b.receipts) {
          receipt.implementation_or_toolchain_root = "impl:tls:only";
        }
      },
      /implementation\/toolchain roots/,
    ],
    [
      "environment",
      (b) => {
        for (const receipt of b.receipts) {
          receipt.execution_environment_digest = "a".repeat(64);
        }
      },
      /execution environments/,
    ],
    [
      "cases",
      (b) => {
        b.receipts[2].evidence_level_and_scope.case_ids = [
          "case-001",
          "case-002",
          "case-003",
          "case-004",
        ];
      },
      /unique cases/,
    ],
    [
      "checker",
      (b) => {
        b.receipts[2].evidence_level_and_scope.checker_or_corpus_digest =
          "0".repeat(64);
      },
      /one checker or corpus digest/,
    ],
  ];
  for (const [name, mutate, pattern] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle);
      refreshEvidenceIds(bundle);
      assertRejected(bundle, pattern);
    });
  }
});

test("one bundle cannot mix evidence levels, artifacts, payee roots, or undeclared evaluators", async (t) => {
  const mutations = [
    [
      "evidence level",
      (b) => {
        Object.assign(b.receipts[2].evidence_level_and_scope, {
          level: "E5",
          adoption_receipt_type: "maintained-fixture",
          adopter_control_root: "control:adopter:mixed-level",
        });
      },
      /must use one evidence level/,
    ],
    [
      "artifact",
      (b) => {
        b.receipts[2].artifact_digest = "0".repeat(64);
      },
      /must bind one artifact digest/,
    ],
    [
      "payee root",
      (b) => {
        b.receipts[0].payee_and_role.control_root = "control:evaluator:b";
      },
      /must equal the verifier control cluster/,
    ],
    [
      "undeclared evaluator",
      (b) => {
        b.receipts[0].payee_and_role.control_root = "control:evaluator:outside";
        b.receipts[0].verifier_control_cluster = "control:evaluator:outside";
        b.receipts[0].conflict_disclosures.related_control_roots = [
          "control:evaluator:outside",
        ];
      },
      /must be a declared technical evaluator root/,
    ],
  ];
  for (const [name, mutate, pattern] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle);
      refreshEvidenceIds(bundle);
      assertRejected(bundle, pattern);
    });
  }
});

test("levels without typed v0 predicates remain metadata-only and cannot be claimed", async (t) => {
  for (const level of ["E0", "E1", "E2", "E4", "E6"]) {
    await t.test(level, () => {
      const bundle = clone(BUNDLE);
      for (const receipt of bundle.receipts) {
        receipt.evidence_level_and_scope.level = level;
      }
      refreshEvidenceIds(bundle);
      assertRejected(bundle, /tree metadata only; receipt schema v0 supports only E3 and E5/);
    });
  }
});

test("claimant, sponsor, evaluators and authorizer must be disjoint", () => {
  const conflict = clone(BUNDLE);
  conflict.policyRevision.roles.payout_authorizer_control_root =
    conflict.policyRevision.roles.sponsor_control_root;
  assertRejected(conflict, /all claimant, sponsor, evaluator, and authorizer roots/);
});

test("claimant and sponsor roots cannot inflate quorum", () => {
  const claimant = clone(BUNDLE);
  claimant.receipts[0].organization_or_control_root =
    claimant.policyRevision.roles.claimant_control_root;
  refreshEvidenceIds(claimant);
  assertRejected(claimant, /cannot contribute to technical quorum/);
});

test("conflict disclosure is complete, outcome-independent and ineligible-loop-free", async (t) => {
  const mutations = [
    ["incomplete", (c) => (c.complete = false)],
    [
      "outcome contingent",
      (c) => (c.outcome_contingent_compensation = true),
    ],
    [
      "deliberately retained",
      (c) => (c.deliberately_planted_or_retained_defect = true),
    ],
    [
      "claimant adoption",
      (c) => (c.claimant_controlled_adoption = true),
    ],
    ["break fix", (c) => (c.self_created_break_fix_loop = true)],
    ["concealed", (c) => (c.concealed_causal_involvement = true)],
  ];
  for (const [name, mutate] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle.receipts[0].conflict_disclosures);
      refreshEvidenceIds(bundle);
      assertRejected(bundle, /must be (?:true|false)/);
    });
  }
});

test("known authorship and review disclosures remain diagnostic rather than auto-ineligible", () => {
  const disclosed = clone(BUNDLE);
  Object.assign(disclosed.receipts[0].conflict_disclosures, {
    claimant_authored_or_controlled_subject: true,
    claimant_reviewed_subject: true,
    claimant_knowingly_preserved_subject: true,
  });
  refreshEvidenceIds(disclosed);
  assert.equal(validateObject(disclosed).structurallyValid, true);
});

test("authorization and safety decisions are hard gates", async (t) => {
  const mutations = [
    ["authorized target", (a) => (a.authorized_target = false)],
    ["safety gate", (a) => (a.safety_gate_passed = false)],
    [
      "triage",
      (a) => (a.prepublication_triage_completed = false),
    ],
    [
      "open unknown impact",
      (a) => (a.unknown_security_impact = true),
    ],
    [
      "unneeded escalation",
      (a) => (a.private_escalation_applied = true),
    ],
    [
      "exploit plaintext",
      (a) => (a.public_exploit_plaintext_present = true),
    ],
    [
      "confidential publication",
      (a) => (a.confidential_evidence_published = true),
    ],
    ["vendor veto", (a) => (a.vendor_veto_required = true)],
    [
      "protocol security assertion",
      (a) => (a.asserts_protocol_security = true),
    ],
    ["network request", (a) => (a.performs_network_requests = true)],
  ];
  for (const [name, mutate] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      mutate(bundle.receipts[0].authorization_and_safety_decision);
      refreshEvidenceIds(bundle);
      assertRejected(
        bundle,
        /must be (?:true|false)|requires private coordinated repair/,
      );
    });
  }
});

test("unknown security impact is valid only in the private escalated lane", () => {
  const privateBundle = clone(BUNDLE);
  for (const receipt of privateBundle.receipts) {
    Object.assign(receipt.authorization_and_safety_decision, {
      disclosure_lane: "private-coordinated-repair",
      unknown_security_impact: true,
      private_escalation_applied: true,
    });
  }
  refreshEvidenceIds(privateBundle);
  assert.equal(validateObject(privateBundle).structurallyValid, true);
});

test("E5 requires an allowed adopter independent from the claimant", () => {
  const validE5 = clone(BUNDLE);
  for (const [index, receipt] of validE5.receipts.entries()) {
    Object.assign(receipt.evidence_level_and_scope, {
      level: "E5",
      adoption_receipt_type: "maintained-fixture",
      adopter_control_root: `control:adopter:${index}`,
    });
  }
  refreshEvidenceIds(validE5);
  const decision = validateObject(validE5);
  assert.equal(decision.evidenceLevel, "E5");
  assert.equal(decision.canonicalEconomics.metadataOnly, true);
  assert.equal(decision.rewardBearing, false);

  const controlled = clone(validE5);
  controlled.receipts[0].evidence_level_and_scope.adopter_control_root =
    controlled.policyRevision.roles.claimant_control_root;
  refreshEvidenceIds(controlled);
  assertRejected(controlled, /independent of every policy role/);
});

test("E5 adopter roots cannot be evaluators, sponsors, authorizers, subject implementations, organizations, or related aliases", async (t) => {
  const forbiddenRoots = [
    ["evaluator", "control:evaluator:a"],
    ["sponsor", BUNDLE.policyRevision.roles.sponsor_control_root],
    ["authorizer", BUNDLE.policyRevision.roles.payout_authorizer_control_root],
    ["organization", BUNDLE.receipts[0].organization_or_control_root],
    ["cross organization", BUNDLE.receipts[1].organization_or_control_root],
    ["implementation", BUNDLE.receipts[0].implementation_or_toolchain_root],
    [
      "cross implementation",
      BUNDLE.receipts[1].implementation_or_toolchain_root,
    ],
    ["related alias", "control:related:evaluator-a"],
    ["cross related alias", "control:related:evaluator-b"],
  ];
  for (const [name, adopterRoot] of forbiddenRoots) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      for (const [index, receipt] of bundle.receipts.entries()) {
        Object.assign(receipt.evidence_level_and_scope, {
          level: "E5",
          adoption_receipt_type: "maintained-fixture",
          adopter_control_root: `control:adopter:${index}`,
        });
      }
      if (name === "related alias") {
        bundle.receipts[0].conflict_disclosures.related_control_roots = [
          "control:evaluator:a",
          adopterRoot,
        ].sort();
      }
      if (name === "cross related alias") {
        bundle.receipts[1].conflict_disclosures.related_control_roots = [
          "control:evaluator:b",
          adopterRoot,
        ].sort();
      }
      bundle.receipts[0].evidence_level_and_scope.adopter_control_root =
        adopterRoot;
      refreshEvidenceIds(bundle);
      assertRejected(bundle, /independent of every policy role/);
    });
  }
});

test("non-E5 receipts cannot smuggle adoption claims", () => {
  const adoption = clone(BUNDLE);
  adoption.receipts[0].evidence_level_and_scope.adoption_receipt_type =
    "upstream-merge";
  adoption.receipts[0].evidence_level_and_scope.adopter_control_root =
    "control:adopter:outside";
  refreshEvidenceIds(adoption);
  assertRejected(adoption, /non-E5 evidence/);
});

test("derivative claims require a distinct prior deliverable, overlap digest and canonical edge", () => {
  const derivative = clone(BUNDLE);
  Object.assign(derivative.receipts[0].prior_deliverable_and_overlap_claim, {
    prior_deliverable_key: "a".repeat(64),
    overlap_claim_digest: "b".repeat(64),
    independently_reviewed_delta: true,
    edge_type: "REPLICATES",
  });
  refreshEvidenceIds(derivative);
  assert.equal(validateObject(derivative).structurallyValid, true);

  const reset = clone(derivative);
  reset.receipts[0].prior_deliverable_and_overlap_claim.prior_deliverable_key =
    reset.receipts[0].deliverable_key;
  refreshEvidenceIds(reset);
  assertRejected(reset, /must identify a different deliverable/);
});

test("superseded revisions stay in replay output but cannot count toward active floors", () => {
  const appendSuccessor = (bundle, createdAt, revision = "2") => {
    const successor = clone(bundle.receipts[0]);
    successor.source_revision = revision;
    successor.created_at = createdAt;
    successor.supersedes = bundle.receipts[0].evidence_id;
    successor.result.result_digest = "0".repeat(64);
    bundle.receipts.push(successor);
    refreshEvidenceIds(bundle);
    return bundle;
  };

  const linked = clone(BUNDLE);
  appendSuccessor(linked, "2026-07-30T10:15:00Z");
  const decision = validateObject(linked);
  assert.equal(decision.structurallyValid, true);
  assert.equal(decision.receiptCount, 3);
  assert.equal(decision.submittedReceiptCount, 4);
  assert.equal(decision.supersededReceiptCount, 1);
  assert.equal(decision.consumptionKeys.length, 4);
  assert.equal(decision.independence.effectiveClusters, 3);

  const nanosecondLater = clone(BUNDLE);
  appendSuccessor(nanosecondLater, "2026-07-30T10:00:00.000000001Z");
  assert.equal(validateObject(nanosecondLater).receiptCount, 3);

  for (const timestamp of [
    "2026-07-29T00:00:00Z",
    "2026-07-30T10:00:00Z",
  ]) {
    const retroactive = clone(BUNDLE);
    appendSuccessor(retroactive, timestamp);
    assertRejected(retroactive, /must be strictly later/);
  }

  const forward = clone(BUNDLE);
  forward.receipts[0].supersedes = forward.receipts[1].evidence_id;
  refreshEvidenceIds(forward);
  assertRejected(forward, /must identify an earlier receipt/);

  const external = clone(BUNDLE);
  external.receipts[0].supersedes = "0".repeat(64);
  refreshEvidenceIds(external);
  assertRejected(external, /must identify an evidence ID in the same bundle/);

  const forked = clone(BUNDLE);
  for (const revision of ["2", "3"]) {
    const replacement = clone(forked.receipts[0]);
    replacement.source_revision = revision;
    replacement.created_at = "2026-07-30T10:15:00Z";
    replacement.supersedes = forked.receipts[0].evidence_id;
    forked.receipts.push(replacement);
  }
  refreshEvidenceIds(forked);
  assertRejected(forked, /cannot create two active replacements/);
});

test("receipt timestamps are bounded by snapshot and explicit active-use date", async (t) => {
  const mutations = [
    ["before snapshot", "2026-07-28T23:59:59Z", /cannot predate/],
    ["after use date", "2026-07-31T00:00:00Z", /cannot be after/],
    ["invalid calendar date", "2026-02-30T10:00:00Z", /real UTC calendar date/],
    ["offset timestamp", "2026-07-30T10:00:00+01:00", /must be an RFC 3339 UTC/],
  ];
  for (const [name, timestamp, pattern] of mutations) {
    await t.test(name, () => {
      const bundle = clone(BUNDLE);
      bundle.receipts[0].created_at = timestamp;
      refreshEvidenceIds(bundle);
      assertRejected(bundle, pattern);
    });
  }
});

test("every zero-value policy boundary rejects any economic activation", async (t) => {
  const numericFields = [
    "total_cap",
    "verified_cost_budget",
    "outcome_pool",
    "reviewer_budget",
    "administration_and_fee_budget",
  ];
  for (const field of numericFields) {
    await t.test(field, () => {
      const bundle = clone(BUNDLE);
      bundle.policyRevision.value_boundary[field] = 1;
      assertRejected(bundle, /must be exactly zero/);
    });
  }
  const nonzeroShapes = [
    ["denomination", "uzrn", /must be null/],
    ["escrow_receipt", "escrow:test", /must be null/],
    ["schedule_instantiated", true, /must be false/],
    ["expiry", "2026-08-01", /must be null/],
    ["refund_path", "refund:test", /must be null/],
  ];
  for (const [field, value, pattern] of nonzeroShapes) {
    await t.test(field, () => {
      const bundle = clone(BUNDLE);
      bundle.policyRevision.value_boundary[field] = value;
      assertRejected(bundle, pattern);
    });
  }
});

test("top-level authority and release boundaries preserve every false", async (t) => {
  for (const field of ["authoritative", "networkObserved", "rewardBearing"]) {
    await t.test(field, () => {
      const bundle = clone(BUNDLE);
      bundle[field] = true;
      assertRejected(bundle, /must be false/);
    });
  }
  for (const field of Object.keys(BUNDLE.releaseBoundary)) {
    await t.test(field, () => {
      const bundle = clone(BUNDLE);
      bundle.releaseBoundary[field] = true;
      assertRejected(bundle, /must be false/);
    });
  }
});

test("strict raw parser rejects malformed, duplicate, unknown, oversized and deep JSON", async (t) => {
  await t.test("malformed", () => {
    assert.throws(() => parseConstructiveReceiptBundle("{"), /invalid JSON/);
  });
  await t.test("duplicate key", () => {
    const raw = BUNDLE_RAW.replace(
      '"authoritative": false,',
      '"authoritative": false,\n  "authoritative": false,',
    );
    assert.throws(() => parseConstructiveReceiptBundle(raw), /duplicate JSON object key/);
  });
  await t.test("unknown key", () => {
    const bundle = clone(BUNDLE);
    bundle.surprise = false;
    assertRejected(bundle, /is not part of schema v0/);
  });
  await t.test("oversized", () => {
    const raw = `${BUNDLE_RAW}${" ".repeat(
      RECEIPT_BUNDLE_MAX_BYTES - Buffer.byteLength(BUNDLE_RAW) + 1,
    )}`;
    assert.throws(() => parseConstructiveReceiptBundle(raw), /exceeds 262144/);
  });
  await t.test("deep", () => {
    const raw = `${"[".repeat(65)}0${"]".repeat(65)}`;
    assert.throws(() => parseConstructiveReceiptBundle(raw), /nesting exceeds/);
  });
});

test("policy rebinding changes canonical policy, deliverable and receipt identities", () => {
  const rebound = clone(BUNDLE);
  rebound.policyRevision.canonical_subject_roots = [
    ...rebound.policyRevision.canonical_subject_roots,
    "standard:ietf:rfc:9846#section-4.4",
  ].sort();
  for (const receipt of rebound.receipts) {
    receipt.canonical_subject_roots = clone(
      rebound.policyRevision.canonical_subject_roots,
    );
  }
  refreshBindings(rebound);
  const decision = validateObject(rebound);
  assert.notEqual(
    decision.policyRevisionDigest,
    derivePolicyRevisionDigest(POLICY),
  );
  assert.notEqual(decision.deliverableKey, BUNDLE.receipts[0].deliverable_key);
  assert.equal(decision.rewardBearing, false);
  assert.equal(decision.valueBoundary.totalCap, 0);
});

test("CLI returns deterministic success, validation failure and usage exit codes", () => {
  const success = spawnSync(
    process.execPath,
    [
      VALIDATOR_PATH,
      "--tree",
      TREE_PATH,
      "--bundle",
      BUNDLE_PATH,
    ],
    { encoding: "utf8" },
  );
  assert.equal(success.status, 0, success.stderr);
  assert.equal(JSON.parse(success.stdout).structurallyValid, true);

  const failure = spawnSync(
    process.execPath,
    [
      VALIDATOR_PATH,
      "--tree",
      TREE_PATH,
      "--bundle",
      POLICY_PATH,
    ],
    { encoding: "utf8" },
  );
  assert.equal(failure.status, 1);
  assert.equal(JSON.parse(failure.stderr).structurallyValid, false);

  const usage = spawnSync(process.execPath, [VALIDATOR_PATH], {
    encoding: "utf8",
  });
  assert.equal(usage.status, 2);
  assert.match(usage.stderr, /^usage:/);
});
