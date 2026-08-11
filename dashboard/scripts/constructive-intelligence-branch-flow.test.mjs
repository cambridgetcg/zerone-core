import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  BRANCH_FLOW_BASE_TREE_SHA256,
  BRANCH_FLOW_MAX_BYTES,
  BRANCH_FLOW_REFERENCE_POLICY_PREIMAGE,
  BRANCH_FLOW_REFERENCE_POLICY_SHA256,
  BranchFlowValidationError,
  parseAndValidateBranchFlow,
} from "./validate-constructive-intelligence-branch-flow.mjs";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-branch-flow.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw);
const baseTreeRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-tree.v1.json",
    import.meta.url,
  ),
);
const CANONICAL_SHA256 =
  "6b83912450fec94772dad8a7bde11c980c4dea6bdb6b867e495e4480d4cc55aa";
const POLICY_PREIMAGE_KEYS = [
  "base_commons_ppm",
  "direct_ppm",
  "domain_id",
  "domain_revision",
  "downstream_continuation_ppm",
  "downstream_max_depth",
  "downstream_ppm",
  "envelope_controller_cap_ppm",
  "min_projected_payout_uzrn",
  "program_window_cap_uzrn",
  "upstream_continuation_ppm",
  "upstream_max_depth",
  "upstream_ppm",
];

function copyStandard() {
  return structuredClone(canonical);
}

function validate(standard) {
  return parseAndValidateBranchFlow(`${JSON.stringify(standard, null, 2)}\n`);
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof BranchFlowValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("constructive-intelligence branch-flow v1 standard", () => {
  it("accepts the exact shadow profile and pins its reviewed bytes", () => {
    assert.deepEqual(parseAndValidateBranchFlow(canonicalRaw), {
      schema: "zerone.constructive-intelligence-branch-flow/v1",
      status: "SHADOW_ONLY",
      splitPpm: 1_000_000,
      maxDepth: 5,
      releaseGateCount: 17,
      passedReleaseGateCount: 0,
      economicEffect: "NONE",
      movesFunds: false,
      integrationReady: false,
    });
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      CANONICAL_SHA256,
    );
  });

  it("exposes only raw-string validation", () => {
    let getterReads = 0;
    const getterObject = {};
    Object.defineProperty(getterObject, "schema", {
      enumerable: true,
      get() {
        getterReads += 1;
        return canonical.schema;
      },
    });
    assertInvalid(() => parseAndValidateBranchFlow(getterObject), "$");
    assert.equal(getterReads, 0);
    assertInvalid(() => parseAndValidateBranchFlow(new String(canonicalRaw)), "$" );
  });

  it("binds the exact canonical reference-policy preimage and digest", () => {
    const preimage = {};
    for (const key of POLICY_PREIMAGE_KEYS) {
      preimage[key] = canonical.reference_policy[key];
    }
    const encoded = JSON.stringify(preimage);
    assert.equal(encoded, BRANCH_FLOW_REFERENCE_POLICY_PREIMAGE);
    assert.equal(
      createHash("sha256").update(encoded).digest("hex"),
      BRANCH_FLOW_REFERENCE_POLICY_SHA256,
    );
    assert.equal(
      canonical.reference_policy.policy_digest,
      `sha256:${BRANCH_FLOW_REFERENCE_POLICY_SHA256}`,
    );

    const digestDrift = copyStandard();
    digestDrift.reference_policy.policy_digest = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () => validate(digestDrift),
      "$.reference_policy.policy_digest",
    );

    const capDrift = copyStandard();
    capDrift.reference_policy.envelope_controller_cap_ppm = 600_000;
    assertInvalid(
      () => validate(capDrift),
      "$.reference_policy.envelope_controller_cap_ppm",
    );
  });

  it("conserves the exact 60/10/30/0 split without recursive issuance", () => {
    const policy = canonical.reference_policy;
    assert.equal(
      policy.direct_ppm +
        policy.upstream_ppm +
        policy.downstream_ppm +
        policy.base_commons_ppm,
      policy.scale_ppm,
    );
    for (const key of [
      "direct_ppm",
      "upstream_ppm",
      "downstream_ppm",
      "base_commons_ppm",
    ]) {
      const changed = copyStandard();
      changed.reference_policy[key] += 1;
      assertInvalid(
        () => validate(changed),
        `$.reference_policy.${key}`,
      );
    }
    for (const key of [
      "recursive_issuance",
      "post_admission_terminal_mass_renormalizes",
    ]) {
      const changed = copyStandard();
      changed.reference_policy[key] = true;
      assertInvalid(
        () => validate(changed),
        `$.reference_policy.${key}`,
      );
    }
    const relativeDepths = copyStandard();
    relativeDepths.reference_policy.absolute_depth_buckets = false;
    assertInvalid(
      () => validate(relativeDepths),
      "$.reference_policy.absolute_depth_buckets",
    );
  });

  it("pins half-per-hop absolute decay through depth five and its terminal tail", () => {
    for (const decay of canonical.reference_policy.decay) {
      assert.equal(decay.continuation_ppm, 500_000);
      assert.equal(decay.max_depth, 5);
      assert.deepEqual(decay.depth_weights_ppm, [
        500_000,
        250_000,
        125_000,
        62_500,
        31_250,
      ]);
      assert.equal(
        decay.depth_weights_ppm.reduce((sum, weight) => sum + weight, 0) +
          decay.tail_ppm,
        1_000_000,
      );
    }

    const enrichedTail = copyStandard();
    enrichedTail.reference_policy.decay[0].tail_ppm += 1;
    assertInvalid(
      () => validate(enrichedTail),
      "$.reference_policy.decay[0].tail_ppm",
    );

    const swappedDirections = copyStandard();
    swappedDirections.reference_policy.decay.reverse();
    assertInvalid(
      () => validate(swappedDirections),
      "$.reference_policy.decay[0].direction",
    );

    const renormalized = copyStandard();
    renormalized.reference_policy.empty_depths_renormalize = true;
    assertInvalid(
      () => validate(renormalized),
      "$.reference_policy.empty_depths_renormalize",
    );
  });

  it("keeps E2-E6 inside the existing 10,000-bps outcome-pool boundary", () => {
    assert.deepEqual(
      canonical.milestone_boundary.milestones.map(
        ({ level, outcome_pool_bps }) => [level, outcome_pool_bps],
      ),
      [
        ["E2", 1_500],
        ["E3", 2_000],
        ["E4", 1_500],
        ["E5", 2_500],
        ["E6", 1_000],
      ],
    );
    assert.equal(
      canonical.milestone_boundary.milestones.reduce(
        (sum, milestone) => sum + milestone.outcome_pool_bps,
        canonical.milestone_boundary.challenge_and_remediation_reserve_bps,
      ),
      10_000,
    );

    const additive = copyStandard();
    additive.milestone_boundary.branch_flow_is_additive_bonus = true;
    assertInvalid(
      () => validate(additive),
      "$.milestone_boundary.branch_flow_is_additive_bonus",
    );

    const prematureDownstream = copyStandard();
    prematureDownstream.milestone_boundary.downstream_consequence_levels[0] =
      "E3";
    assertInvalid(
      () => validate(prematureDownstream),
      "$.milestone_boundary.downstream_consequence_levels[0]",
    );
  });

  it("pins the funded-cluster graph, receipt, rounding, and adapter invariants", () => {
    const invariants = canonical.allocation_invariants;
    assert.equal(invariants.funded_cluster_is_economic_subject, true);
    assert.equal(invariants.breakthrough_is_allocation_input, false);
    assert.equal(invariants.breakthrough_creates_prize, false);
    assert.equal(invariants.edge_orientation, "CHILD_TO_PARENT");
    assert.equal(invariants.graph_input, "ADJUDICATED_ACYCLIC_DAG");
    assert.equal(invariants.edge_share_formula, "floor(S*a/max(S,sum_a))");
    assert.equal(
      invariants.descendant_impact_share_formula,
      "floor(S*m/max(S,sum_m_per_semantic_descendant))",
    );
    assert.equal(invariants.graph_multiplication_rounding, "FLOOR_EACH_STEP");
    assert.equal(
      invariants.graph_floor_residue_is_reserved_terminal_mass,
      false,
    );
    assert.equal(invariants.independent_abundance_may_fill_fixed_bucket, true);
    assert.equal(invariants.absolute_sparse_depths, true);
    assert.deepEqual(invariants.admitted_impact_dispositions, [
      "PAYABLE",
      "TERMINAL",
    ]);
    assert.equal(invariants.terminal_disposition_preserves_capacity, true);
    assert.equal(invariants.terminal_disposition_consumes_receipt, true);
    assert.equal(invariants.economic_receipt_use, "GLOBAL_EXCLUSIVE");
    assert.equal(
      invariants.accepted_receipt_use,
      "CONSUME_ON_SUCCESSFUL_EVALUATION",
    );
    assert.equal(invariants.zero_projection_consumes_accepted_receipt, true);
    assert.equal(invariants.invalid_request_consumes_receipt, false);
    assert.equal(
      invariants.controller_aggregation,
      "BEFORE_CAPS_AND_ROUNDING",
    );
    assert.equal(
      invariants.funded_controller_descendant_credit_eligible,
      false,
    );
    assert.equal(
      invariants.mixed_control_descendant_independent_credits_remain_evaluable,
      true,
    );
    assert.equal(
      invariants.hidden_or_correlated_control_requires_external_adjudication,
      true,
    );
    assert.equal(
      invariants.monetary_apportionment,
      "FLOOR_PER_CONTROLLER_RESIDUAL_TO_TERMINAL_INNER_HAMILTON",
    );
    assert.equal(invariants.exact_tie_priority, "COMMONS_THEN_CANONICAL_KEY");
    assert.deepEqual(invariants.terminal_reasons, [
      "BASE",
      "ADMITTED_TERMINAL",
      "UNATTRIBUTED",
      "CONTROLLER_INELIGIBILITY",
      "ROUNDING",
      "CAP",
      "DUST",
      "TAIL",
    ]);
    assert.equal(invariants.implemented_adapter, "OUTCOME_REWARD_ONLY");
    assert.equal(invariants.reference_milestone, "E5");
    assert.equal(
      invariants.impact_milestone_must_match_funded_milestone,
      true,
    );
    assert.equal(invariants.pre_e5_downstream_ppm, 0);
    assert.equal(invariants.pre_e5_descendant_impacts_allowed, false);
    assert.deepEqual(invariants.tc6_training_revenue, {
      implemented: false,
      separate_ledger_required: true,
      separate_receipt_namespace_required: true,
    });

    for (const key of [
      "breakthrough_is_allocation_input",
      "breakthrough_creates_prize",
      "additive_bonus",
    ]) {
      const enabled = copyStandard();
      enabled.allocation_invariants[key] = true;
      assertInvalid(
        () => validate(enabled),
        `$.allocation_invariants.${key}`,
      );
    }
    const recursiveReceiptUse = copyStandard();
    recursiveReceiptUse.allocation_invariants.economic_receipt_use =
      "PER_ENVELOPE";
    assertInvalid(
      () => validate(recursiveReceiptUse),
      "$.allocation_invariants.economic_receipt_use",
    );
    const splitImpact = copyStandard();
    splitImpact.allocation_invariants.descendant_impact_share_formula =
      "m_per_receipt";
    assertInvalid(
      () => validate(splitImpact),
      "$.allocation_invariants.descendant_impact_share_formula",
    );
    const replayZero = copyStandard();
    replayZero.allocation_invariants.zero_projection_consumes_accepted_receipt =
      false;
    assertInvalid(
      () => validate(replayZero),
      "$.allocation_invariants.zero_projection_consumes_accepted_receipt",
    );
    const crossMilestone = copyStandard();
    crossMilestone.allocation_invariants.reference_milestone = "E3";
    assertInvalid(
      () => validate(crossMilestone),
      "$.allocation_invariants.reference_milestone",
    );
    const selfAdoption = copyStandard();
    selfAdoption.allocation_invariants.funded_controller_descendant_credit_eligible =
      true;
    assertInvalid(
      () => validate(selfAdoption),
      "$.allocation_invariants.funded_controller_descendant_credit_eligible",
    );
    const eraseIndependentCredit = copyStandard();
    eraseIndependentCredit.allocation_invariants.mixed_control_descendant_independent_credits_remain_evaluable =
      false;
    assertInvalid(
      () => validate(eraseIndependentCredit),
      "$.allocation_invariants.mixed_control_descendant_independent_credits_remain_evaluable",
    );
    const inferHiddenControl = copyStandard();
    inferHiddenControl.allocation_invariants.hidden_or_correlated_control_requires_external_adjudication =
      false;
    assertInvalid(
      () => validate(inferHiddenControl),
      "$.allocation_invariants.hidden_or_correlated_control_requires_external_adjudication",
    );
    const preE5Impact = copyStandard();
    preE5Impact.allocation_invariants.pre_e5_descendant_impacts_allowed = true;
    assertInvalid(
      () => validate(preE5Impact),
      "$.allocation_invariants.pre_e5_descendant_impacts_allowed",
    );
    const activeTc6 = copyStandard();
    activeTc6.allocation_invariants.tc6_training_revenue.implemented = true;
    assertInvalid(
      () => validate(activeTc6),
      "$.allocation_invariants.tc6_training_revenue.implemented",
    );
  });

  it("keeps every authority, economic, release, and integration switch closed", () => {
    for (const key of [
      "moves_funds",
      "integration_ready",
      "authoritative",
      "network_observed",
      "reward_bearing",
    ]) {
      const standard = copyStandard();
      standard[key] = true;
      assertInvalid(() => validate(standard), `$.${key}`);
    }
    for (const key of Object.keys(canonical.authority_boundary)) {
      const standard = copyStandard();
      standard.authority_boundary[key] =
        key === "accepts_adjudicated_inputs_only" ? false : true;
      assertInvalid(
        () => validate(standard),
        `$.authority_boundary.${key}`,
      );
    }
    for (const key of Object.keys(canonical.release_boundary).filter(
      (key) => typeof canonical.release_boundary[key] === "boolean",
    )) {
      const standard = copyStandard();
      standard.release_boundary[key] = true;
      assertInvalid(
        () => validate(standard),
        `$.release_boundary.${key}`,
      );
    }
  });

  it("requires all seventeen ordered release gates to remain unmet", () => {
    assert.equal(canonical.release_gates.length, 17);
    assert.ok(canonical.release_gates.every(({ passed }) => !passed));

    const opened = copyStandard();
    opened.release_gates[12].passed = true;
    assertInvalid(() => validate(opened), "$.release_gates[12].passed");

    const reordered = copyStandard();
    [reordered.release_gates[0], reordered.release_gates[1]] = [
      reordered.release_gates[1],
      reordered.release_gates[0],
    ];
    assertInvalid(() => validate(reordered), "$.release_gates[0].id");

    const omitted = copyStandard();
    omitted.release_gates.pop();
    assertInvalid(() => validate(omitted), "$.release_gates");
  });

  it("pins the untouched canonical tree bytes", () => {
    assert.equal(
      createHash("sha256").update(baseTreeRaw).digest("hex"),
      BRANCH_FLOW_BASE_TREE_SHA256,
    );
    assert.equal(canonical.base_tree.sha256, BRANCH_FLOW_BASE_TREE_SHA256);
  });

  it("rejects unknown fields, duplicate keys, malformed JSON, nesting, and oversize", () => {
    const unknown = copyStandard();
    unknown.payout_address = "zrn1notallowed";
    assertInvalid(() => validate(unknown), "$.payout_address");

    const nestedUnknown = copyStandard();
    nestedUnknown.reference_policy.royalty_rate_ppm = 1;
    assertInvalid(
      () => validate(nestedUnknown),
      "$.reference_policy.royalty_rate_ppm",
    );

    const duplicateStatus = canonicalRaw.replace(
      '  "status": "SHADOW_ONLY",',
      '  "status": "SHADOW_ONLY",\n  "status": "ACTIVE",',
    );
    assertInvalid(() => parseAndValidateBranchFlow(duplicateStatus), "$.status");
    assertInvalid(() => parseAndValidateBranchFlow("{"), "$" );
    assertInvalid(
      () => parseAndValidateBranchFlow(" ".repeat(BRANCH_FLOW_MAX_BYTES + 1)),
      "$",
    );
    const tooDeep = `${"[".repeat(25)}0${"]".repeat(25)}`;
    assertInvalid(() => parseAndValidateBranchFlow(tooDeep), "$" );
  });
});
