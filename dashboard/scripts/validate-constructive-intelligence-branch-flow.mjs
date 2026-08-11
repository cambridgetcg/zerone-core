import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const BRANCH_FLOW_SCHEMA =
  "zerone.constructive-intelligence-branch-flow/v1";
export const BRANCH_FLOW_MAX_BYTES = 32_768;
export const BRANCH_FLOW_BASE_TREE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
export const BRANCH_FLOW_REFERENCE_POLICY_SHA256 =
  "cc8601fdc0efd8b0260a5a979fa456f43e45be7a3ebd821ca330b389ebb26684";
export const BRANCH_FLOW_REFERENCE_POLICY_PREIMAGE =
  '{"base_commons_ppm":0,"direct_ppm":600000,"domain_id":"general","domain_revision":1,"downstream_continuation_ppm":500000,"downstream_max_depth":5,"downstream_ppm":300000,"envelope_controller_cap_ppm":1000000,"min_projected_payout_uzrn":"0","program_window_cap_uzrn":"","upstream_continuation_ppm":500000,"upstream_max_depth":5,"upstream_ppm":100000}';

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "..", "..");
const SCALE_PPM = 1_000_000;
const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "assurance",
  "economic_effect",
  "moves_funds",
  "integration_ready",
  "authoritative",
  "network_observed",
  "reward_bearing",
  "snapshot_date",
  "specification",
  "purpose",
  "base_tree",
  "reference_policy",
  "milestone_boundary",
  "allocation_invariants",
  "authority_boundary",
  "release_boundary",
  "release_gates",
];
const BASE_TREE_KEYS = ["schema", "endpoint", "sha256"];
const PROFILE_KEYS = [
  "policy_digest",
  "scale_ppm",
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
  "decay",
  "absolute_depth_buckets",
  "empty_depths_renormalize",
  "recursive_issuance",
  "post_admission_terminal_mass_renormalizes",
  "post_settlement_unattributed_routes_to_terminal_destination",
];
const SPLIT_KEYS = [
  "direct_ppm",
  "upstream_ppm",
  "downstream_ppm",
  "base_commons_ppm",
];
const EXPECTED_SPLIT = Object.freeze({
  direct_ppm: 600_000,
  upstream_ppm: 100_000,
  downstream_ppm: 300_000,
  base_commons_ppm: 0,
});
const REFERENCE_POLICY_PREIMAGE_KEYS = [
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
const DECAY_KEYS = [
  "direction",
  "continuation_ppm",
  "max_depth",
  "depth_weights_ppm",
  "tail_ppm",
];
const EXPECTED_DEPTH_WEIGHTS = [500_000, 250_000, 125_000, 62_500, 31_250];
const MILESTONE_BOUNDARY_KEYS = [
  "milestones",
  "challenge_and_remediation_reserve_bps",
  "total_outcome_pool_bps",
  "downstream_consequence_levels",
  "novelty_level",
  "attack_disproof_repair_level",
  "branch_flow_is_additive_bonus",
  "skill_unlock_creates_entitlement",
  "breakthrough_creates_funds",
];
const MILESTONE_KEYS = ["level", "outcome_pool_bps", "constraint"];
const EXPECTED_MILESTONES = Object.freeze([
  Object.freeze({
    level: "E2",
    outcome_pool_bps: 1_500,
    constraint: "CLASS_VERIFICATION",
  }),
  Object.freeze({
    level: "E3",
    outcome_pool_bps: 2_000,
    constraint: "NOVELTY_MAY_SHAPE_ONLY_HERE",
  }),
  Object.freeze({
    level: "E4",
    outcome_pool_bps: 1_500,
    constraint: "ATTACK_DISPROOF_OR_REPAIR_COMPARTMENT",
  }),
  Object.freeze({
    level: "E5",
    outcome_pool_bps: 2_500,
    constraint: "DOWNSTREAM_CONSEQUENCE_MAY_SHAPE",
  }),
  Object.freeze({
    level: "E6",
    outcome_pool_bps: 1_000,
    constraint: "MAINTAINED_CONSEQUENCE_MAY_SHAPE",
  }),
]);
const ALLOCATION_INVARIANT_KEYS = [
  "funded_cluster_is_economic_subject",
  "breakthrough_is_allocation_input",
  "breakthrough_creates_prize",
  "one_conserved_envelope",
  "additive_bonus",
  "edge_orientation",
  "graph_input",
  "edge_share_formula",
  "descendant_impact_share_formula",
  "graph_multiplication_rounding",
  "graph_floor_residue_is_reserved_terminal_mass",
  "independent_abundance_may_fill_fixed_bucket",
  "absolute_sparse_depths",
  "admitted_impact_dispositions",
  "terminal_disposition_preserves_capacity",
  "terminal_disposition_consumes_receipt",
  "economic_receipt_use",
  "accepted_receipt_use",
  "zero_projection_consumes_accepted_receipt",
  "invalid_request_consumes_receipt",
  "controller_aggregation",
  "funded_controller_descendant_credit_eligible",
  "mixed_control_descendant_independent_credits_remain_evaluable",
  "hidden_or_correlated_control_requires_external_adjudication",
  "monetary_apportionment",
  "exact_tie_priority",
  "terminal_reasons",
  "implemented_adapter",
  "reference_milestone",
  "impact_milestone_must_match_funded_milestone",
  "pre_e5_downstream_ppm",
  "pre_e5_descendant_impacts_allowed",
  "tc6_training_revenue",
];
const TC6_KEYS = [
  "implemented",
  "separate_ledger_required",
  "separate_receipt_namespace_required",
];
const AUTHORITY_BOUNDARY_KEYS = [
  "allocator_is_authority",
  "accepts_adjudicated_inputs_only",
  "selects_policy",
  "selects_winners",
  "reads_store",
  "writes_store",
  "uses_bank",
  "uses_clock",
  "uses_network",
  "uses_signer",
  "changes_governance",
  "changes_qualification",
  "changes_karma",
  "changes_ontology",
];
const RELEASE_BOUNDARY_KEYS = [
  "assurance",
  "economic_effect",
  "amount_uzrn",
  "integration_ready",
  "adds_consensus_behavior",
  "activates_rewards",
  "moves_funds",
  "creates_entitlement",
  "mints",
  "reserves",
  "transfers",
  "vests",
  "burns",
  "claws_back",
  "reads_chain_state",
  "writes_chain_state",
  "grants_qualification",
  "grants_governance_power",
  "modifies_karma",
  "modifies_ontology",
  "activates_upgrade",
];
const RELEASE_FALSE_KEYS = RELEASE_BOUNDARY_KEYS.filter(
  (key) => !["assurance", "economic_effect", "amount_uzrn"].includes(key),
);
const RELEASE_GATE_KEYS = ["id", "passed"];
const EXPECTED_RELEASE_GATE_IDS = [
  "canonical-semantic-cluster-adjudication",
  "typed-causal-edge-and-consequence-receipts",
  "global-economic-receipt-consumption",
  "authoritative-controller-records-and-merges",
  "class-specific-dependency-and-impact-scorers",
  "fixed-funding-liability-expiry-refund-and-commons-state",
  "exact-milestone-and-observation-window-transitions",
  "authoritative-closed-cohort-membership-and-terminal-tombstones",
  "envelope-and-program-window-controller-caps",
  "atomic-role-commons-replay-liability-and-bank-settlement",
  "backed-vesting-schedules-and-monetary-migration",
  "sdk-governance-only-policy-authority",
  "bounded-performance-storage-replay-and-dos-analysis",
  "two-exact-implementations-and-golden-vectors",
  "adversarial-simulation-and-independent-review",
  "upgrade-rollback-incident-and-production-verification",
  "separately-authorized-network-release",
];

export class BranchFlowValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "BranchFlowValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new BranchFlowValidationError(path, message);
}

function record(value, path) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value;
}

function exactKeys(value, expected, path) {
  const expectedSet = new Set(expected);
  for (const key of Object.keys(value)) {
    if (!expectedSet.has(key)) fail(`${path}.${key}`, "unknown field");
  }
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "missing field");
  }
}

function exact(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${JSON.stringify(expected)}`);
  return expected;
}

function falseOnly(value, path) {
  return exact(value, false, path);
}

function trueOnly(value, path) {
  return exact(value, true, path);
}

function integer(value, path, minimum, maximum) {
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(path, `must be an integer from ${minimum} to ${maximum}`);
  }
  return value;
}

function exactArray(value, expected, path) {
  if (!Array.isArray(value) || value.length !== expected.length) {
    fail(path, `must contain exactly ${expected.length} entries`);
  }
  for (const [index, expectedValue] of expected.entries()) {
    exact(value[index], expectedValue, `${path}[${index}]`);
  }
  return value;
}

function validateReferenceProfile(value) {
  const profile = record(value, "$.reference_policy");
  exactKeys(profile, PROFILE_KEYS, "$.reference_policy");
  exact(profile.scale_ppm, SCALE_PPM, "$.reference_policy.scale_ppm");

  let splitTotal = 0;
  for (const key of SPLIT_KEYS) {
    const amount = integer(
      profile[key],
      `$.reference_policy.${key}`,
      0,
      SCALE_PPM,
    );
    exact(
      amount,
      EXPECTED_SPLIT[key],
      `$.reference_policy.${key}`,
    );
    splitTotal += amount;
  }
  exact(splitTotal, SCALE_PPM, "$.reference_policy");
  exact(profile.domain_id, "general", "$.reference_policy.domain_id");
  exact(profile.domain_revision, 1, "$.reference_policy.domain_revision");
  exact(
    profile.downstream_continuation_ppm,
    500_000,
    "$.reference_policy.downstream_continuation_ppm",
  );
  exact(
    profile.downstream_max_depth,
    5,
    "$.reference_policy.downstream_max_depth",
  );
  exact(
    profile.envelope_controller_cap_ppm,
    SCALE_PPM,
    "$.reference_policy.envelope_controller_cap_ppm",
  );
  exact(
    profile.min_projected_payout_uzrn,
    "0",
    "$.reference_policy.min_projected_payout_uzrn",
  );
  exact(
    profile.program_window_cap_uzrn,
    "",
    "$.reference_policy.program_window_cap_uzrn",
  );
  exact(
    profile.upstream_continuation_ppm,
    500_000,
    "$.reference_policy.upstream_continuation_ppm",
  );
  exact(
    profile.upstream_max_depth,
    5,
    "$.reference_policy.upstream_max_depth",
  );
  const policyPreimage = {};
  for (const key of REFERENCE_POLICY_PREIMAGE_KEYS) {
    policyPreimage[key] = profile[key];
  }
  const encodedPolicy = JSON.stringify(policyPreimage);
  exact(
    encodedPolicy,
    BRANCH_FLOW_REFERENCE_POLICY_PREIMAGE,
    "$.reference_policy",
  );
  const policyDigest = createHash("sha256").update(encodedPolicy).digest("hex");
  exact(
    policyDigest,
    BRANCH_FLOW_REFERENCE_POLICY_SHA256,
    "$.reference_policy.policy_digest",
  );
  exact(
    profile.policy_digest,
    `sha256:${policyDigest}`,
    "$.reference_policy.policy_digest",
  );

  if (!Array.isArray(profile.decay) || profile.decay.length !== 2) {
    fail("$.reference_policy.decay", "must contain upstream and downstream");
  }
  for (const [index, direction] of ["UPSTREAM", "DOWNSTREAM"].entries()) {
    const path = `$.reference_policy.decay[${index}]`;
    const decay = record(profile.decay[index], path);
    exactKeys(decay, DECAY_KEYS, path);
    exact(decay.direction, direction, `${path}.direction`);
    exact(decay.continuation_ppm, 500_000, `${path}.continuation_ppm`);
    exact(decay.max_depth, 5, `${path}.max_depth`);
    exactArray(
      decay.depth_weights_ppm,
      EXPECTED_DEPTH_WEIGHTS,
      `${path}.depth_weights_ppm`,
    );
    exact(decay.tail_ppm, 31_250, `${path}.tail_ppm`);
    const conserved = decay.depth_weights_ppm.reduce(
      (sum, weight) => sum + integer(weight, path, 0, SCALE_PPM),
      decay.tail_ppm,
    );
    exact(conserved, SCALE_PPM, path);
  }
  trueOnly(
    profile.absolute_depth_buckets,
    "$.reference_policy.absolute_depth_buckets",
  );
  falseOnly(
    profile.empty_depths_renormalize,
    "$.reference_policy.empty_depths_renormalize",
  );
  falseOnly(
    profile.recursive_issuance,
    "$.reference_policy.recursive_issuance",
  );
  falseOnly(
    profile.post_admission_terminal_mass_renormalizes,
    "$.reference_policy.post_admission_terminal_mass_renormalizes",
  );
  trueOnly(
    profile.post_settlement_unattributed_routes_to_terminal_destination,
    "$.reference_policy.post_settlement_unattributed_routes_to_terminal_destination",
  );
}

function validateMilestoneBoundary(value) {
  const boundary = record(value, "$.milestone_boundary");
  exactKeys(boundary, MILESTONE_BOUNDARY_KEYS, "$.milestone_boundary");
  if (
    !Array.isArray(boundary.milestones) ||
    boundary.milestones.length !== EXPECTED_MILESTONES.length
  ) {
    fail("$.milestone_boundary.milestones", "must contain exactly E2 through E6");
  }
  for (const [index, expected] of EXPECTED_MILESTONES.entries()) {
    const path = `$.milestone_boundary.milestones[${index}]`;
    const milestone = record(boundary.milestones[index], path);
    exactKeys(milestone, MILESTONE_KEYS, path);
    for (const key of MILESTONE_KEYS) {
      exact(milestone[key], expected[key], `${path}.${key}`);
    }
  }
  const milestoneBps = boundary.milestones.reduce(
    (sum, milestone) => sum + milestone.outcome_pool_bps,
    0,
  );
  exact(
    boundary.challenge_and_remediation_reserve_bps,
    1_500,
    "$.milestone_boundary.challenge_and_remediation_reserve_bps",
  );
  exact(
    boundary.total_outcome_pool_bps,
    10_000,
    "$.milestone_boundary.total_outcome_pool_bps",
  );
  exact(
    milestoneBps + boundary.challenge_and_remediation_reserve_bps,
    boundary.total_outcome_pool_bps,
    "$.milestone_boundary",
  );
  exactArray(
    boundary.downstream_consequence_levels,
    ["E5", "E6"],
    "$.milestone_boundary.downstream_consequence_levels",
  );
  exact(boundary.novelty_level, "E3", "$.milestone_boundary.novelty_level");
  exact(
    boundary.attack_disproof_repair_level,
    "E4",
    "$.milestone_boundary.attack_disproof_repair_level",
  );
  for (const key of [
    "branch_flow_is_additive_bonus",
    "skill_unlock_creates_entitlement",
    "breakthrough_creates_funds",
  ]) {
    falseOnly(boundary[key], `$.milestone_boundary.${key}`);
  }
}

function validateAuthorityBoundary(value) {
  const boundary = record(value, "$.authority_boundary");
  exactKeys(boundary, AUTHORITY_BOUNDARY_KEYS, "$.authority_boundary");
  for (const key of AUTHORITY_BOUNDARY_KEYS) {
    if (key === "accepts_adjudicated_inputs_only") {
      trueOnly(boundary[key], `$.authority_boundary.${key}`);
    } else {
      falseOnly(boundary[key], `$.authority_boundary.${key}`);
    }
  }
}

function validateAllocationInvariants(value) {
  const path = "$.allocation_invariants";
  const invariants = record(value, path);
  exactKeys(invariants, ALLOCATION_INVARIANT_KEYS, path);
  trueOnly(
    invariants.funded_cluster_is_economic_subject,
    `${path}.funded_cluster_is_economic_subject`,
  );
  falseOnly(
    invariants.breakthrough_is_allocation_input,
    `${path}.breakthrough_is_allocation_input`,
  );
  falseOnly(
    invariants.breakthrough_creates_prize,
    `${path}.breakthrough_creates_prize`,
  );
  trueOnly(invariants.one_conserved_envelope, `${path}.one_conserved_envelope`);
  falseOnly(invariants.additive_bonus, `${path}.additive_bonus`);
  exact(invariants.edge_orientation, "CHILD_TO_PARENT", `${path}.edge_orientation`);
  exact(
    invariants.graph_input,
    "ADJUDICATED_ACYCLIC_DAG",
    `${path}.graph_input`,
  );
  exact(
    invariants.edge_share_formula,
    "floor(S*a/max(S,sum_a))",
    `${path}.edge_share_formula`,
  );
  exact(
    invariants.descendant_impact_share_formula,
    "floor(S*m/max(S,sum_m_per_semantic_descendant))",
    `${path}.descendant_impact_share_formula`,
  );
  exact(
    invariants.graph_multiplication_rounding,
    "FLOOR_EACH_STEP",
    `${path}.graph_multiplication_rounding`,
  );
  falseOnly(
    invariants.graph_floor_residue_is_reserved_terminal_mass,
    `${path}.graph_floor_residue_is_reserved_terminal_mass`,
  );
  trueOnly(
    invariants.independent_abundance_may_fill_fixed_bucket,
    `${path}.independent_abundance_may_fill_fixed_bucket`,
  );
  trueOnly(invariants.absolute_sparse_depths, `${path}.absolute_sparse_depths`);
  exactArray(
    invariants.admitted_impact_dispositions,
    ["PAYABLE", "TERMINAL"],
    `${path}.admitted_impact_dispositions`,
  );
  trueOnly(
    invariants.terminal_disposition_preserves_capacity,
    `${path}.terminal_disposition_preserves_capacity`,
  );
  trueOnly(
    invariants.terminal_disposition_consumes_receipt,
    `${path}.terminal_disposition_consumes_receipt`,
  );
  exact(
    invariants.economic_receipt_use,
    "GLOBAL_EXCLUSIVE",
    `${path}.economic_receipt_use`,
  );
  exact(
    invariants.accepted_receipt_use,
    "CONSUME_ON_SUCCESSFUL_EVALUATION",
    `${path}.accepted_receipt_use`,
  );
  trueOnly(
    invariants.zero_projection_consumes_accepted_receipt,
    `${path}.zero_projection_consumes_accepted_receipt`,
  );
  falseOnly(
    invariants.invalid_request_consumes_receipt,
    `${path}.invalid_request_consumes_receipt`,
  );
  exact(
    invariants.controller_aggregation,
    "BEFORE_CAPS_AND_ROUNDING",
    `${path}.controller_aggregation`,
  );
  falseOnly(
    invariants.funded_controller_descendant_credit_eligible,
    `${path}.funded_controller_descendant_credit_eligible`,
  );
  trueOnly(
    invariants.mixed_control_descendant_independent_credits_remain_evaluable,
    `${path}.mixed_control_descendant_independent_credits_remain_evaluable`,
  );
  trueOnly(
    invariants.hidden_or_correlated_control_requires_external_adjudication,
    `${path}.hidden_or_correlated_control_requires_external_adjudication`,
  );
  exact(
    invariants.monetary_apportionment,
    "FLOOR_PER_CONTROLLER_RESIDUAL_TO_TERMINAL_INNER_HAMILTON",
    `${path}.monetary_apportionment`,
  );
  exact(
    invariants.exact_tie_priority,
    "COMMONS_THEN_CANONICAL_KEY",
    `${path}.exact_tie_priority`,
  );
  exactArray(
    invariants.terminal_reasons,
    [
      "BASE",
      "ADMITTED_TERMINAL",
      "UNATTRIBUTED",
      "CONTROLLER_INELIGIBILITY",
      "ROUNDING",
      "CAP",
      "DUST",
      "TAIL",
    ],
    `${path}.terminal_reasons`,
  );
  exact(
    invariants.implemented_adapter,
    "OUTCOME_REWARD_ONLY",
    `${path}.implemented_adapter`,
  );
  exact(invariants.reference_milestone, "E5", `${path}.reference_milestone`);
  trueOnly(
    invariants.impact_milestone_must_match_funded_milestone,
    `${path}.impact_milestone_must_match_funded_milestone`,
  );
  exact(invariants.pre_e5_downstream_ppm, 0, `${path}.pre_e5_downstream_ppm`);
  falseOnly(
    invariants.pre_e5_descendant_impacts_allowed,
    `${path}.pre_e5_descendant_impacts_allowed`,
  );
  const tc6Path = `${path}.tc6_training_revenue`;
  const tc6 = record(invariants.tc6_training_revenue, tc6Path);
  exactKeys(tc6, TC6_KEYS, tc6Path);
  falseOnly(tc6.implemented, `${tc6Path}.implemented`);
  trueOnly(tc6.separate_ledger_required, `${tc6Path}.separate_ledger_required`);
  trueOnly(
    tc6.separate_receipt_namespace_required,
    `${tc6Path}.separate_receipt_namespace_required`,
  );
}

function validateReleaseBoundary(value) {
  const boundary = record(value, "$.release_boundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.release_boundary");
  exact(boundary.assurance, "SHADOW_ONLY", "$.release_boundary.assurance");
  exact(
    boundary.economic_effect,
    "NONE",
    "$.release_boundary.economic_effect",
  );
  exact(boundary.amount_uzrn, "0", "$.release_boundary.amount_uzrn");
  for (const key of RELEASE_FALSE_KEYS) {
    falseOnly(boundary[key], `$.release_boundary.${key}`);
  }
}

function validateReleaseGates(value) {
  if (!Array.isArray(value) || value.length !== EXPECTED_RELEASE_GATE_IDS.length) {
    fail(
      "$.release_gates",
      `must contain exactly ${EXPECTED_RELEASE_GATE_IDS.length} gates`,
    );
  }
  value.forEach((candidate, index) => {
    const path = `$.release_gates[${index}]`;
    const gate = record(candidate, path);
    exactKeys(gate, RELEASE_GATE_KEYS, path);
    exact(gate.id, EXPECTED_RELEASE_GATE_IDS[index], `${path}.id`);
    falseOnly(gate.passed, `${path}.passed`);
  });
}

function validateBranchFlow(value) {
  const root = record(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  exact(root.schema, BRANCH_FLOW_SCHEMA, "$.schema");
  exact(root.status, "SHADOW_ONLY", "$.status");
  exact(root.assurance, "SHADOW_ONLY", "$.assurance");
  exact(root.economic_effect, "NONE", "$.economic_effect");
  for (const key of [
    "moves_funds",
    "integration_ready",
    "authoritative",
    "network_observed",
    "reward_bearing",
  ]) {
    falseOnly(root[key], `$.${key}`);
  }
  exact(root.snapshot_date, "2026-08-11", "$.snapshot_date");
  exact(
    root.specification,
    "docs/specs/constructive-intelligence-branch-flow-v1.md",
    "$.specification",
  );
  if (typeof root.purpose !== "string" || root.purpose.length < 80 || root.purpose.length > 1_024) {
    fail("$.purpose", "must be a bounded explanatory string");
  }

  const baseTree = record(root.base_tree, "$.base_tree");
  exactKeys(baseTree, BASE_TREE_KEYS, "$.base_tree");
  exact(
    baseTree.schema,
    "zerone.constructive-intelligence-tree/v1",
    "$.base_tree.schema",
  );
  exact(
    baseTree.endpoint,
    "/standards/constructive-intelligence-tree.v1.json",
    "$.base_tree.endpoint",
  );
  exact(baseTree.sha256, BRANCH_FLOW_BASE_TREE_SHA256, "$.base_tree.sha256");

  validateReferenceProfile(root.reference_policy);
  validateMilestoneBoundary(root.milestone_boundary);
  validateAllocationInvariants(root.allocation_invariants);
  validateAuthorityBoundary(root.authority_boundary);
  validateReleaseBoundary(root.release_boundary);
  validateReleaseGates(root.release_gates);

  return Object.freeze({
    schema: BRANCH_FLOW_SCHEMA,
    status: "SHADOW_ONLY",
    splitPpm: SCALE_PPM,
    maxDepth: 5,
    releaseGateCount: EXPECTED_RELEASE_GATE_IDS.length,
    passedReleaseGateCount: 0,
    economicEffect: "NONE",
    movesFunds: false,
    integrationReady: false,
  });
}

function rejectDuplicateJsonKeys(raw) {
  let offset = 0;
  const whitespace = () => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = () => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") {
        offset += 2;
        continue;
      }
      if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset));
      }
      offset += 1;
    }
    fail("$", "unterminated JSON string");
  };
  const scanValue = (path) => {
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        const key = scanString();
        const keyPath = `${path}.${key}`;
        if (keys.has(key)) fail(keyPath, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        offset += 1;
        scanValue(keyPath);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        offset += 1;
      }
      return;
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") {
        offset += 1;
        return;
      }
      let index = 0;
      while (offset < raw.length) {
        scanValue(`${path}[${index}]`);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        offset += 1;
        index += 1;
      }
      return;
    }
    if (token === '"') {
      scanString();
      return;
    }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset])) offset += 1;
  };
  scanValue("$");
}

function rejectExcessiveJsonNesting(raw) {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (const character of raw) {
    if (inString) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') inString = false;
      continue;
    }
    if (character === '"') inString = true;
    else if (character === "{" || character === "[") {
      depth += 1;
      if (depth > 24) fail("$", "JSON nesting exceeds the branch-flow limit of 24");
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
}

export function parseAndValidateBranchFlow(raw) {
  if (typeof raw !== "string") fail("$", "input must be a JSON string");
  if (Buffer.byteLength(raw, "utf8") > BRANCH_FLOW_MAX_BYTES) {
    fail("$", `must be at most ${BRANCH_FLOW_MAX_BYTES} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw);
  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    fail("$", `input must be valid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw);
  return validateBranchFlow(value);
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === SCRIPT_PATH) {
  const standardPath = resolve(
    process.cwd(),
    process.argv[2] ??
      "public/standards/constructive-intelligence-branch-flow.v1.json",
  );
  const baseTreePath = resolve(
    process.cwd(),
    process.argv[3] ??
      "public/standards/constructive-intelligence-tree.v1.json",
  );
  const raw = readFileSync(standardPath, "utf8");
  const result = parseAndValidateBranchFlow(raw);
  const baseTreeDigest = createHash("sha256")
    .update(readFileSync(baseTreePath))
    .digest("hex");
  if (baseTreeDigest !== BRANCH_FLOW_BASE_TREE_SHA256) {
    fail("$.base_tree.sha256", "does not match the supplied base tree bytes");
  }
  const specificationPath = resolve(
    REPOSITORY_ROOT,
    "docs/specs/constructive-intelligence-branch-flow-v1.md",
  );
  if (!existsSync(specificationPath)) {
    fail("$.specification", "does not resolve inside the repository");
  }
  const digest = createHash("sha256").update(raw).digest("hex");
  console.log(
    `branch flow ${result.schema} is valid (${result.splitPpm} ppm conserved, depth ${result.maxDepth}, ${result.passedReleaseGateCount}/${result.releaseGateCount} release gates passed, economic effect ${result.economicEffect}, sha256:${digest})`,
  );
}
