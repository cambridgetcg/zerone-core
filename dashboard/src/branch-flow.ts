/// <reference lib="dom" />

export const BRANCH_FLOW_ENDPOINT =
  "/standards/constructive-intelligence-branch-flow.v1.json";
export const BRANCH_FLOW_MAX_BYTES = 32_768;
export const BRANCH_FLOW_SHA256 =
  "6b83912450fec94772dad8a7bde11c980c4dea6bdb6b867e495e4480d4cc55aa";
export const BRANCH_FLOW_REFERENCE_POLICY_DIGEST =
  "sha256:cc8601fdc0efd8b0260a5a979fa456f43e45be7a3ebd821ca330b389ebb26684";

export type BranchFlowDirection = "UPSTREAM" | "DOWNSTREAM";
export type BranchFlowMilestoneLevel = "E2" | "E3" | "E4" | "E5" | "E6";

export interface BranchFlowDecay {
  direction: BranchFlowDirection;
  continuationPpm: 500_000;
  maxDepth: 5;
  depthWeightsPpm: readonly [500_000, 250_000, 125_000, 62_500, 31_250];
  tailPpm: 31_250;
}

export interface BranchFlowReferencePolicy {
  policyDigest: typeof BRANCH_FLOW_REFERENCE_POLICY_DIGEST;
  scalePpm: 1_000_000;
  directPpm: 600_000;
  upstreamPpm: 100_000;
  downstreamPpm: 300_000;
  baseCommonsPpm: 0;
  domainId: "general";
  domainRevision: 1;
  envelopeControllerCapPpm: 1_000_000;
  minProjectedPayoutUzrn: "0";
  programWindowCapUzrn: "";
  decay: readonly [BranchFlowDecay, BranchFlowDecay];
  absoluteDepthBuckets: true;
  emptyDepthsRenormalize: false;
  recursiveIssuance: false;
  postAdmissionTerminalMassRenormalizes: false;
  postSettlementUnattributedRoutesToTerminalDestination: true;
}

export interface BranchFlowMilestone {
  level: BranchFlowMilestoneLevel;
  outcomePoolBps: number;
  constraint: string;
}

export interface BranchFlowAllocationInvariants {
  fundedClusterIsEconomicSubject: true;
  breakthroughIsAllocationInput: false;
  breakthroughCreatesPrize: false;
  oneConservedEnvelope: true;
  additiveBonus: false;
  edgeOrientation: "CHILD_TO_PARENT";
  graphInput: "ADJUDICATED_ACYCLIC_DAG";
  edgeShareFormula: "floor(S*a/max(S,sum_a))";
  descendantImpactShareFormula: "floor(S*m/max(S,sum_m_per_semantic_descendant))";
  graphMultiplicationRounding: "FLOOR_EACH_STEP";
  graphFloorResidueIsReservedTerminalMass: false;
  independentAbundanceMayFillFixedBucket: true;
  absoluteSparseDepths: true;
  admittedImpactDispositions: readonly ["PAYABLE", "TERMINAL"];
  terminalDispositionPreservesCapacity: true;
  terminalDispositionConsumesReceipt: true;
  economicReceiptUse: "GLOBAL_EXCLUSIVE";
  acceptedReceiptUse: "CONSUME_ON_SUCCESSFUL_EVALUATION";
  zeroProjectionConsumesAcceptedReceipt: true;
  invalidRequestConsumesReceipt: false;
  controllerAggregation: "BEFORE_CAPS_AND_ROUNDING";
  fundedControllerDescendantCreditEligible: false;
  mixedControlDescendantIndependentCreditsRemainEvaluable: true;
  hiddenOrCorrelatedControlRequiresExternalAdjudication: true;
  monetaryApportionment: "FLOOR_PER_CONTROLLER_RESIDUAL_TO_TERMINAL_INNER_HAMILTON";
  exactTiePriority: "COMMONS_THEN_CANONICAL_KEY";
  terminalReasons: readonly [
    "BASE",
    "ADMITTED_TERMINAL",
    "UNATTRIBUTED",
    "CONTROLLER_INELIGIBILITY",
    "ROUNDING",
    "CAP",
    "DUST",
    "TAIL",
  ];
  implementedAdapter: "OUTCOME_REWARD_ONLY";
  referenceMilestone: "E5";
  impactMilestoneMustMatchFundedMilestone: true;
  preE5DownstreamPpm: 0;
  preE5DescendantImpactsAllowed: false;
  tc6TrainingRevenue: {
    implemented: false;
    separateLedgerRequired: true;
    separateReceiptNamespaceRequired: true;
  };
}

export interface ConstructiveIntelligenceBranchFlow {
  schema: "zerone.constructive-intelligence-branch-flow/v1";
  status: "SHADOW_ONLY";
  assurance: "SHADOW_ONLY";
  economicEffect: "NONE";
  movesFunds: false;
  integrationReady: false;
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  snapshotDate: "2026-08-11";
  specification: "docs/specs/constructive-intelligence-branch-flow-v1.md";
  purpose: string;
  baseTree: {
    schema: "zerone.constructive-intelligence-tree/v1";
    endpoint: "/standards/constructive-intelligence-tree.v1.json";
    sha256: "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
  };
  referencePolicy: BranchFlowReferencePolicy;
  milestones: readonly BranchFlowMilestone[];
  challengeAndRemediationReserveBps: 1_500;
  totalOutcomePoolBps: 10_000;
  allocationInvariants: BranchFlowAllocationInvariants;
  releaseAmountUzrn: "0";
  releaseGates: readonly { id: string; passed: false }[];
}

export interface BranchFlowFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class BranchFlowDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "BranchFlowDataError";
  }
}

type JsonObject = Record<string, unknown>;

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
] as const;
const REFERENCE_POLICY_KEYS = [
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
] as const;
const DECAY_KEYS = [
  "direction",
  "continuation_ppm",
  "max_depth",
  "depth_weights_ppm",
  "tail_ppm",
] as const;
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
] as const;
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
] as const;
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
] as const;
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
] as const;
const EXPECTED_MILESTONES = [
  ["E2", 1_500, "CLASS_VERIFICATION"],
  ["E3", 2_000, "NOVELTY_MAY_SHAPE_ONLY_HERE"],
  ["E4", 1_500, "ATTACK_DISPROOF_OR_REPAIR_COMPARTMENT"],
  ["E5", 2_500, "DOWNSTREAM_CONSEQUENCE_MAY_SHAPE"],
  ["E6", 1_000, "MAINTAINED_CONSEQUENCE_MAY_SHAPE"],
] as const;
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
] as const;

function fail(path: string, message: string): never {
  throw new BranchFlowDataError(`${path}: ${message}`);
}

function asObject(value: unknown, path: string): JsonObject {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value as JsonObject;
}

function assertExactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const expectedSet = new Set(expected);
  for (const key of Object.keys(value)) {
    if (!expectedSet.has(key)) fail(`${path}.${key}`, "unknown field");
  }
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "missing field");
  }
}

function asLiteral<T extends string | number | boolean>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function asString(value: unknown, path: string, maximum = 1_024): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, "expected a bounded non-empty string");
  }
  return value;
}

function asArray(value: unknown, path: string, length: number): unknown[] {
  if (!Array.isArray(value) || value.length !== length) {
    fail(path, `must contain exactly ${length} entries`);
  }
  return value;
}

function assertLiteralArray<T extends string | number>(
  value: unknown,
  expected: readonly T[],
  path: string,
): readonly T[] {
  const items = asArray(value, path, expected.length);
  expected.forEach((entry, index) =>
    asLiteral(items[index], entry, `${path}[${index}]`),
  );
  return expected;
}

function parseDecay(
  value: unknown,
  direction: BranchFlowDirection,
  path: string,
): BranchFlowDecay {
  const decay = asObject(value, path);
  assertExactKeys(decay, DECAY_KEYS, path);
  asLiteral(decay.direction, direction, `${path}.direction`);
  asLiteral(decay.continuation_ppm, 500_000, `${path}.continuation_ppm`);
  asLiteral(decay.max_depth, 5, `${path}.max_depth`);
  const weights = assertLiteralArray(
    decay.depth_weights_ppm,
    [500_000, 250_000, 125_000, 62_500, 31_250] as const,
    `${path}.depth_weights_ppm`,
  );
  asLiteral(decay.tail_ppm, 31_250, `${path}.tail_ppm`);
  if (
    weights.reduce<number>((sum, weight) => sum + weight, 31_250) !==
    1_000_000
  ) {
    fail(path, "depth weights and tail must conserve one direction tranche");
  }
  return {
    direction,
    continuationPpm: 500_000,
    maxDepth: 5,
    depthWeightsPpm: [500_000, 250_000, 125_000, 62_500, 31_250],
    tailPpm: 31_250,
  };
}

function parseReferencePolicy(value: unknown): BranchFlowReferencePolicy {
  const path = "$.reference_policy";
  const policy = asObject(value, path);
  assertExactKeys(policy, REFERENCE_POLICY_KEYS, path);
  asLiteral(
    policy.policy_digest,
    BRANCH_FLOW_REFERENCE_POLICY_DIGEST,
    `${path}.policy_digest`,
  );
  asLiteral(policy.scale_ppm, 1_000_000, `${path}.scale_ppm`);
  asLiteral(policy.base_commons_ppm, 0, `${path}.base_commons_ppm`);
  asLiteral(policy.direct_ppm, 600_000, `${path}.direct_ppm`);
  asLiteral(policy.domain_id, "general", `${path}.domain_id`);
  asLiteral(policy.domain_revision, 1, `${path}.domain_revision`);
  asLiteral(
    policy.downstream_continuation_ppm,
    500_000,
    `${path}.downstream_continuation_ppm`,
  );
  asLiteral(policy.downstream_max_depth, 5, `${path}.downstream_max_depth`);
  asLiteral(policy.downstream_ppm, 300_000, `${path}.downstream_ppm`);
  asLiteral(
    policy.envelope_controller_cap_ppm,
    1_000_000,
    `${path}.envelope_controller_cap_ppm`,
  );
  asLiteral(
    policy.min_projected_payout_uzrn,
    "0",
    `${path}.min_projected_payout_uzrn`,
  );
  asLiteral(
    policy.program_window_cap_uzrn,
    "",
    `${path}.program_window_cap_uzrn`,
  );
  asLiteral(
    policy.upstream_continuation_ppm,
    500_000,
    `${path}.upstream_continuation_ppm`,
  );
  asLiteral(policy.upstream_max_depth, 5, `${path}.upstream_max_depth`);
  asLiteral(policy.upstream_ppm, 100_000, `${path}.upstream_ppm`);
  if (600_000 + 100_000 + 300_000 + 0 !== 1_000_000) {
    fail(path, "reference split does not conserve one envelope");
  }
  const decay = asArray(policy.decay, `${path}.decay`, 2);
  const upstream = parseDecay(decay[0], "UPSTREAM", `${path}.decay[0]`);
  const downstream = parseDecay(
    decay[1],
    "DOWNSTREAM",
    `${path}.decay[1]`,
  );
  asLiteral(
    policy.absolute_depth_buckets,
    true,
    `${path}.absolute_depth_buckets`,
  );
  asLiteral(
    policy.empty_depths_renormalize,
    false,
    `${path}.empty_depths_renormalize`,
  );
  asLiteral(
    policy.recursive_issuance,
    false,
    `${path}.recursive_issuance`,
  );
  asLiteral(
    policy.post_admission_terminal_mass_renormalizes,
    false,
    `${path}.post_admission_terminal_mass_renormalizes`,
  );
  asLiteral(
    policy.post_settlement_unattributed_routes_to_terminal_destination,
    true,
    `${path}.post_settlement_unattributed_routes_to_terminal_destination`,
  );
  return {
    policyDigest: BRANCH_FLOW_REFERENCE_POLICY_DIGEST,
    scalePpm: 1_000_000,
    directPpm: 600_000,
    upstreamPpm: 100_000,
    downstreamPpm: 300_000,
    baseCommonsPpm: 0,
    domainId: "general",
    domainRevision: 1,
    envelopeControllerCapPpm: 1_000_000,
    minProjectedPayoutUzrn: "0",
    programWindowCapUzrn: "",
    decay: [upstream, downstream],
    absoluteDepthBuckets: true,
    emptyDepthsRenormalize: false,
    recursiveIssuance: false,
    postAdmissionTerminalMassRenormalizes: false,
    postSettlementUnattributedRoutesToTerminalDestination: true,
  };
}

function parseMilestones(value: unknown): {
  milestones: BranchFlowMilestone[];
  challengeAndRemediationReserveBps: 1_500;
  totalOutcomePoolBps: 10_000;
} {
  const path = "$.milestone_boundary";
  const boundary = asObject(value, path);
  assertExactKeys(boundary, MILESTONE_BOUNDARY_KEYS, path);
  const rawMilestones = asArray(
    boundary.milestones,
    `${path}.milestones`,
    EXPECTED_MILESTONES.length,
  );
  const milestones = EXPECTED_MILESTONES.map(
    ([level, outcomePoolBps, constraint], index): BranchFlowMilestone => {
      const milestonePath = `${path}.milestones[${index}]`;
      const milestone = asObject(rawMilestones[index], milestonePath);
      assertExactKeys(
        milestone,
        ["level", "outcome_pool_bps", "constraint"],
        milestonePath,
      );
      asLiteral(milestone.level, level, `${milestonePath}.level`);
      asLiteral(
        milestone.outcome_pool_bps,
        outcomePoolBps,
        `${milestonePath}.outcome_pool_bps`,
      );
      asLiteral(milestone.constraint, constraint, `${milestonePath}.constraint`);
      return { level, outcomePoolBps, constraint };
    },
  );
  asLiteral(
    boundary.challenge_and_remediation_reserve_bps,
    1_500,
    `${path}.challenge_and_remediation_reserve_bps`,
  );
  asLiteral(
    boundary.total_outcome_pool_bps,
    10_000,
    `${path}.total_outcome_pool_bps`,
  );
  assertLiteralArray(
    boundary.downstream_consequence_levels,
    ["E5", "E6"] as const,
    `${path}.downstream_consequence_levels`,
  );
  asLiteral(boundary.novelty_level, "E3", `${path}.novelty_level`);
  asLiteral(
    boundary.attack_disproof_repair_level,
    "E4",
    `${path}.attack_disproof_repair_level`,
  );
  for (const key of [
    "branch_flow_is_additive_bonus",
    "skill_unlock_creates_entitlement",
    "breakthrough_creates_funds",
  ] as const) {
    asLiteral(boundary[key], false, `${path}.${key}`);
  }
  const conserved = milestones.reduce(
    (sum, milestone) => sum + milestone.outcomePoolBps,
    1_500,
  );
  if (conserved !== 10_000) fail(path, "outcome pool must conserve 10,000 bps");
  return {
    milestones,
    challengeAndRemediationReserveBps: 1_500,
    totalOutcomePoolBps: 10_000,
  };
}

function parseAllocationInvariants(
  value: unknown,
): BranchFlowAllocationInvariants {
  const path = "$.allocation_invariants";
  const invariants = asObject(value, path);
  assertExactKeys(invariants, ALLOCATION_INVARIANT_KEYS, path);
  asLiteral(
    invariants.funded_cluster_is_economic_subject,
    true,
    `${path}.funded_cluster_is_economic_subject`,
  );
  asLiteral(
    invariants.breakthrough_is_allocation_input,
    false,
    `${path}.breakthrough_is_allocation_input`,
  );
  asLiteral(
    invariants.breakthrough_creates_prize,
    false,
    `${path}.breakthrough_creates_prize`,
  );
  asLiteral(invariants.one_conserved_envelope, true, `${path}.one_conserved_envelope`);
  asLiteral(invariants.additive_bonus, false, `${path}.additive_bonus`);
  asLiteral(invariants.edge_orientation, "CHILD_TO_PARENT", `${path}.edge_orientation`);
  asLiteral(
    invariants.graph_input,
    "ADJUDICATED_ACYCLIC_DAG",
    `${path}.graph_input`,
  );
  asLiteral(
    invariants.edge_share_formula,
    "floor(S*a/max(S,sum_a))",
    `${path}.edge_share_formula`,
  );
  asLiteral(
    invariants.descendant_impact_share_formula,
    "floor(S*m/max(S,sum_m_per_semantic_descendant))",
    `${path}.descendant_impact_share_formula`,
  );
  asLiteral(
    invariants.graph_multiplication_rounding,
    "FLOOR_EACH_STEP",
    `${path}.graph_multiplication_rounding`,
  );
  asLiteral(
    invariants.graph_floor_residue_is_reserved_terminal_mass,
    false,
    `${path}.graph_floor_residue_is_reserved_terminal_mass`,
  );
  asLiteral(
    invariants.independent_abundance_may_fill_fixed_bucket,
    true,
    `${path}.independent_abundance_may_fill_fixed_bucket`,
  );
  asLiteral(invariants.absolute_sparse_depths, true, `${path}.absolute_sparse_depths`);
  assertLiteralArray(
    invariants.admitted_impact_dispositions,
    ["PAYABLE", "TERMINAL"] as const,
    `${path}.admitted_impact_dispositions`,
  );
  asLiteral(
    invariants.terminal_disposition_preserves_capacity,
    true,
    `${path}.terminal_disposition_preserves_capacity`,
  );
  asLiteral(
    invariants.terminal_disposition_consumes_receipt,
    true,
    `${path}.terminal_disposition_consumes_receipt`,
  );
  asLiteral(
    invariants.economic_receipt_use,
    "GLOBAL_EXCLUSIVE",
    `${path}.economic_receipt_use`,
  );
  asLiteral(
    invariants.accepted_receipt_use,
    "CONSUME_ON_SUCCESSFUL_EVALUATION",
    `${path}.accepted_receipt_use`,
  );
  asLiteral(
    invariants.zero_projection_consumes_accepted_receipt,
    true,
    `${path}.zero_projection_consumes_accepted_receipt`,
  );
  asLiteral(
    invariants.invalid_request_consumes_receipt,
    false,
    `${path}.invalid_request_consumes_receipt`,
  );
  asLiteral(
    invariants.controller_aggregation,
    "BEFORE_CAPS_AND_ROUNDING",
    `${path}.controller_aggregation`,
  );
  asLiteral(
    invariants.funded_controller_descendant_credit_eligible,
    false,
    `${path}.funded_controller_descendant_credit_eligible`,
  );
  asLiteral(
    invariants.mixed_control_descendant_independent_credits_remain_evaluable,
    true,
    `${path}.mixed_control_descendant_independent_credits_remain_evaluable`,
  );
  asLiteral(
    invariants.hidden_or_correlated_control_requires_external_adjudication,
    true,
    `${path}.hidden_or_correlated_control_requires_external_adjudication`,
  );
  asLiteral(
    invariants.monetary_apportionment,
    "FLOOR_PER_CONTROLLER_RESIDUAL_TO_TERMINAL_INNER_HAMILTON",
    `${path}.monetary_apportionment`,
  );
  asLiteral(
    invariants.exact_tie_priority,
    "COMMONS_THEN_CANONICAL_KEY",
    `${path}.exact_tie_priority`,
  );
  assertLiteralArray(
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
    ] as const,
    `${path}.terminal_reasons`,
  );
  asLiteral(
    invariants.implemented_adapter,
    "OUTCOME_REWARD_ONLY",
    `${path}.implemented_adapter`,
  );
  asLiteral(
    invariants.reference_milestone,
    "E5",
    `${path}.reference_milestone`,
  );
  asLiteral(
    invariants.impact_milestone_must_match_funded_milestone,
    true,
    `${path}.impact_milestone_must_match_funded_milestone`,
  );
  asLiteral(
    invariants.pre_e5_downstream_ppm,
    0,
    `${path}.pre_e5_downstream_ppm`,
  );
  asLiteral(
    invariants.pre_e5_descendant_impacts_allowed,
    false,
    `${path}.pre_e5_descendant_impacts_allowed`,
  );
  const tc6Path = `${path}.tc6_training_revenue`;
  const tc6 = asObject(invariants.tc6_training_revenue, tc6Path);
  assertExactKeys(
    tc6,
    ["implemented", "separate_ledger_required", "separate_receipt_namespace_required"],
    tc6Path,
  );
  asLiteral(tc6.implemented, false, `${tc6Path}.implemented`);
  asLiteral(tc6.separate_ledger_required, true, `${tc6Path}.separate_ledger_required`);
  asLiteral(
    tc6.separate_receipt_namespace_required,
    true,
    `${tc6Path}.separate_receipt_namespace_required`,
  );
  return {
    fundedClusterIsEconomicSubject: true,
    breakthroughIsAllocationInput: false,
    breakthroughCreatesPrize: false,
    oneConservedEnvelope: true,
    additiveBonus: false,
    edgeOrientation: "CHILD_TO_PARENT",
    graphInput: "ADJUDICATED_ACYCLIC_DAG",
    edgeShareFormula: "floor(S*a/max(S,sum_a))",
    descendantImpactShareFormula:
      "floor(S*m/max(S,sum_m_per_semantic_descendant))",
    graphMultiplicationRounding: "FLOOR_EACH_STEP",
    graphFloorResidueIsReservedTerminalMass: false,
    independentAbundanceMayFillFixedBucket: true,
    absoluteSparseDepths: true,
    admittedImpactDispositions: ["PAYABLE", "TERMINAL"],
    terminalDispositionPreservesCapacity: true,
    terminalDispositionConsumesReceipt: true,
    economicReceiptUse: "GLOBAL_EXCLUSIVE",
    acceptedReceiptUse: "CONSUME_ON_SUCCESSFUL_EVALUATION",
    zeroProjectionConsumesAcceptedReceipt: true,
    invalidRequestConsumesReceipt: false,
    controllerAggregation: "BEFORE_CAPS_AND_ROUNDING",
    fundedControllerDescendantCreditEligible: false,
    mixedControlDescendantIndependentCreditsRemainEvaluable: true,
    hiddenOrCorrelatedControlRequiresExternalAdjudication: true,
    monetaryApportionment:
      "FLOOR_PER_CONTROLLER_RESIDUAL_TO_TERMINAL_INNER_HAMILTON",
    exactTiePriority: "COMMONS_THEN_CANONICAL_KEY",
    terminalReasons: [
      "BASE",
      "ADMITTED_TERMINAL",
      "UNATTRIBUTED",
      "CONTROLLER_INELIGIBILITY",
      "ROUNDING",
      "CAP",
      "DUST",
      "TAIL",
    ],
    implementedAdapter: "OUTCOME_REWARD_ONLY",
    referenceMilestone: "E5",
    impactMilestoneMustMatchFundedMilestone: true,
    preE5DownstreamPpm: 0,
    preE5DescendantImpactsAllowed: false,
    tc6TrainingRevenue: {
      implemented: false,
      separateLedgerRequired: true,
      separateReceiptNamespaceRequired: true,
    },
  };
}

function parseAuthorityBoundary(value: unknown): void {
  const path = "$.authority_boundary";
  const boundary = asObject(value, path);
  assertExactKeys(boundary, AUTHORITY_BOUNDARY_KEYS, path);
  for (const key of AUTHORITY_BOUNDARY_KEYS) {
    asLiteral(
      boundary[key],
      key === "accepts_adjudicated_inputs_only",
      `${path}.${key}`,
    );
  }
}

function parseReleaseBoundary(value: unknown): "0" {
  const path = "$.release_boundary";
  const boundary = asObject(value, path);
  assertExactKeys(boundary, RELEASE_BOUNDARY_KEYS, path);
  asLiteral(boundary.assurance, "SHADOW_ONLY", `${path}.assurance`);
  asLiteral(boundary.economic_effect, "NONE", `${path}.economic_effect`);
  asLiteral(boundary.amount_uzrn, "0", `${path}.amount_uzrn`);
  for (const key of RELEASE_BOUNDARY_KEYS) {
    if (["assurance", "economic_effect", "amount_uzrn"].includes(key)) continue;
    asLiteral(boundary[key], false, `${path}.${key}`);
  }
  return "0";
}

function parseReleaseGates(value: unknown): { id: string; passed: false }[] {
  const gates = asArray(value, "$.release_gates", EXPECTED_RELEASE_GATE_IDS.length);
  return EXPECTED_RELEASE_GATE_IDS.map((id, index) => {
    const path = `$.release_gates[${index}]`;
    const gate = asObject(gates[index], path);
    assertExactKeys(gate, ["id", "passed"], path);
    asLiteral(gate.id, id, `${path}.id`);
    asLiteral(gate.passed, false, `${path}.passed`);
    return { id, passed: false };
  });
}

export function parseBranchFlow(value: unknown): ConstructiveIntelligenceBranchFlow {
  const root = asObject(value, "$");
  assertExactKeys(root, TOP_LEVEL_KEYS, "$");
  asLiteral(
    root.schema,
    "zerone.constructive-intelligence-branch-flow/v1",
    "$.schema",
  );
  asLiteral(root.status, "SHADOW_ONLY", "$.status");
  asLiteral(root.assurance, "SHADOW_ONLY", "$.assurance");
  asLiteral(root.economic_effect, "NONE", "$.economic_effect");
  for (const key of [
    "moves_funds",
    "integration_ready",
    "authoritative",
    "network_observed",
    "reward_bearing",
  ] as const) {
    asLiteral(root[key], false, `$.${key}`);
  }
  asLiteral(root.snapshot_date, "2026-08-11", "$.snapshot_date");
  asLiteral(
    root.specification,
    "docs/specs/constructive-intelligence-branch-flow-v1.md",
    "$.specification",
  );
  const purpose = asString(root.purpose, "$.purpose");
  if (purpose.length < 80) fail("$.purpose", "must explain the bounded profile");

  const baseTreePath = "$.base_tree";
  const baseTree = asObject(root.base_tree, baseTreePath);
  assertExactKeys(baseTree, ["schema", "endpoint", "sha256"], baseTreePath);
  asLiteral(
    baseTree.schema,
    "zerone.constructive-intelligence-tree/v1",
    `${baseTreePath}.schema`,
  );
  asLiteral(
    baseTree.endpoint,
    "/standards/constructive-intelligence-tree.v1.json",
    `${baseTreePath}.endpoint`,
  );
  asLiteral(
    baseTree.sha256,
    "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
    `${baseTreePath}.sha256`,
  );

  const referencePolicy = parseReferencePolicy(root.reference_policy);
  const milestoneBoundary = parseMilestones(root.milestone_boundary);
  const allocationInvariants = parseAllocationInvariants(
    root.allocation_invariants,
  );
  parseAuthorityBoundary(root.authority_boundary);
  const releaseAmountUzrn = parseReleaseBoundary(root.release_boundary);
  const releaseGates = parseReleaseGates(root.release_gates);

  return {
    schema: "zerone.constructive-intelligence-branch-flow/v1",
    status: "SHADOW_ONLY",
    assurance: "SHADOW_ONLY",
    economicEffect: "NONE",
    movesFunds: false,
    integrationReady: false,
    authoritative: false,
    networkObserved: false,
    rewardBearing: false,
    snapshotDate: "2026-08-11",
    specification: "docs/specs/constructive-intelligence-branch-flow-v1.md",
    purpose,
    baseTree: {
      schema: "zerone.constructive-intelligence-tree/v1",
      endpoint: "/standards/constructive-intelligence-tree.v1.json",
      sha256: "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
    },
    referencePolicy,
    milestones: milestoneBoundary.milestones,
    challengeAndRemediationReserveBps:
      milestoneBoundary.challengeAndRemediationReserveBps,
    totalOutcomePoolBps: milestoneBoundary.totalOutcomePoolBps,
    allocationInvariants,
    releaseAmountUzrn,
    releaseGates,
  };
}

export function parseBranchFlowJson(
  raw: string,
): ConstructiveIntelligenceBranchFlow {
  if (new TextEncoder().encode(raw).byteLength > BRANCH_FLOW_MAX_BYTES) {
    fail("$", `document exceeds ${BRANCH_FLOW_MAX_BYTES} UTF-8 bytes`);
  }
  let value: unknown;
  try {
    value = JSON.parse(raw) as unknown;
  } catch {
    fail("$", "malformed JSON");
  }
  return parseBranchFlow(value);
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new BranchFlowDataError(
      "Static branch-flow profile returned an empty response body",
    );
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(
      signal.reason ??
        new DOMException("Static branch-flow request timed out", "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > BRANCH_FLOW_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Cancellation is best-effort; the byte limit is already decisive.
        });
        throw new BranchFlowDataError(
          "Static branch-flow profile exceeds its size limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // A hostile cancellation must not extend the deadline.
      });
      throw new BranchFlowDataError("Static branch-flow request timed out");
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A pending hostile read is abandoned after refusal.
    }
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new BranchFlowDataError(
      "Static branch-flow digest verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

function assertCanonicalResponseUrl(response: Response, baseUrl?: string): void {
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(BRANCH_FLOW_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new BranchFlowDataError(
      "Static branch-flow profile returned an invalid final URL",
    );
  }
  if (
    actual.origin !== expected.origin ||
    actual.pathname !== expected.pathname ||
    actual.search !== expected.search ||
    actual.hash !== expected.hash
  ) {
    throw new BranchFlowDataError(
      "Static branch-flow profile left its canonical same-origin path",
    );
  }
}

export async function fetchBranchFlow(
  options: BranchFlowFetchOptions = {},
): Promise<ConstructiveIntelligenceBranchFlow> {
  const fetcher = options.fetcher ?? fetch;
  const timeoutMs = options.timeoutMs ?? 8_000;
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(() => {
    controller.abort(
      new DOMException("Static branch-flow request timed out", "TimeoutError"),
    );
  }, timeoutMs);
  const signal = controller.signal;
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? undefined : window.location.href);
  try {
    let response: Response;
    try {
      let rejectOnAbort: ((reason?: unknown) => void) | undefined;
      const aborted = new Promise<never>((_resolve, reject) => {
        rejectOnAbort = reject;
      });
      const onAbort = (): void => {
        rejectOnAbort?.(
          signal.reason ??
            new DOMException("Static branch-flow request timed out", "TimeoutError"),
        );
      };
      signal.addEventListener("abort", onAbort, { once: true });
      if (signal.aborted) onAbort();
      try {
        response = await Promise.race([
          fetcher(BRANCH_FLOW_ENDPOINT, {
            cache: "no-store",
            credentials: "same-origin",
            headers: { Accept: "application/json" },
            redirect: "error",
            signal,
          }),
          aborted,
        ]);
      } finally {
        signal.removeEventListener("abort", onAbort);
      }
    } catch (error) {
      if (
        error instanceof DOMException &&
        (error.name === "AbortError" || error.name === "TimeoutError")
      ) {
        throw new BranchFlowDataError("Static branch-flow request timed out");
      }
      throw new BranchFlowDataError("Static branch-flow profile is unavailable");
    }
    if (!response.ok) {
      throw new BranchFlowDataError(
        `Static branch-flow profile returned HTTP ${response.status}`,
      );
    }
    const contentType = response.headers.get("content-type");
    if (contentType === null || !/\bjson\b/i.test(contentType)) {
      throw new BranchFlowDataError(
        "Static branch-flow profile did not return an explicit JSON response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (
      declaredLength !== null &&
      (!/^\d+$/.test(declaredLength) ||
        Number(declaredLength) > BRANCH_FLOW_MAX_BYTES)
    ) {
      throw new BranchFlowDataError(
        "Static branch-flow profile exceeded its size limit",
      );
    }
    assertCanonicalResponseUrl(response, baseUrl);
    const bytes = await readBoundedResponse(response, signal);
    if ((await sha256Hex(bytes)) !== BRANCH_FLOW_SHA256) {
      throw new BranchFlowDataError(
        "Static branch-flow profile did not match the reviewed canonical digest",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new BranchFlowDataError(
        "Static branch-flow profile was not valid UTF-8",
      );
    }
    return parseBranchFlowJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function formatPpm(value: number): string {
  return `${new Intl.NumberFormat("en-GB", {
    maximumFractionDigits: 4,
  }).format(value / 10_000)}%`;
}

function renderAssurance(flow: ConstructiveIntelligenceBranchFlow): HTMLElement {
  const facts = element("dl", "branch-flow-assurance");
  const values: readonly [string, string][] = [
    ["Assurance", flow.assurance.replaceAll("_", " ")],
    ["Economic effect", flow.economicEffect],
    [
      "Integration gates",
      `${flow.releaseGates.filter(({ passed }) => passed).length} / ${flow.releaseGates.length}`,
    ],
    ["Activated amount (static)", `${flow.releaseAmountUzrn} uzrn`],
  ];
  for (const [term, description] of values) {
    const item = element("div");
    item.append(
      element("dt", undefined, term),
      element("dd", undefined, description),
    );
    facts.append(item);
  }
  return facts;
}

function renderSplit(flow: ConstructiveIntelligenceBranchFlow): HTMLElement {
  const section = element("section", "branch-flow-block");
  section.append(
    element("span", "card-kicker", "One conserved envelope"),
    element("h4", undefined, "60 / 10 / 30 / 0"),
  );
  const grid = element("div", "branch-flow-split");
  const legs: readonly [string, string, number][] = [
    ["DIRECT", "Current milestone roles", flow.referencePolicy.directPpm],
    ["UPSTREAM", "Load-bearing ancestors", flow.referencePolicy.upstreamPpm],
    [
      "DOWNSTREAM",
      "Independent descendants",
      flow.referencePolicy.downstreamPpm,
    ],
    [
      "BASE_COMMONS",
      "Base commons",
      flow.referencePolicy.baseCommonsPpm,
    ],
  ];
  for (const [leg, label, ppm] of legs) {
    const card = element("article", "branch-flow-leg");
    card.dataset.leg = leg;
    card.append(
      element("span", undefined, label),
      element("strong", undefined, formatPpm(ppm)),
      element("code", undefined, `${ppm.toLocaleString("en-GB")} ppm`),
    );
    grid.append(card);
  }
  section.append(
    grid,
    element(
      "p",
      "branch-flow-note",
      "The funded semantic cluster is the economic subject. Breakthrough is retrospective only: never an allocation input, separate prize, or source of new funds.",
    ),
  );
  return section;
}

function renderDecay(flow: ConstructiveIntelligenceBranchFlow): HTMLElement {
  const section = element("section", "branch-flow-block");
  section.append(
    element("span", "card-kicker", "Absolute geometric depth"),
    element("h4", undefined, "Half per hop · depth five"),
  );
  const directions = element("div", "branch-flow-decay-grid");
  for (const decay of flow.referencePolicy.decay) {
    const card = element("article", "branch-flow-decay");
    card.dataset.direction = decay.direction;
    card.append(
      element("strong", undefined, decay.direction.toLowerCase()),
      element(
        "span",
        undefined,
        `${formatPpm(decay.continuationPpm)} continuation · max depth ${decay.maxDepth}`,
      ),
    );
    const levels = element("ol", "branch-flow-depths");
    decay.depthWeightsPpm.forEach((weight, index) => {
      const item = element("li");
      const label = element("span", undefined, `Depth ${index + 1}`);
      const progress = element("progress");
      progress.max = flow.referencePolicy.scalePpm;
      progress.value = weight;
      progress.setAttribute(
        "aria-label",
        `${decay.direction.toLowerCase()} depth ${index + 1}: ${formatPpm(weight)}`,
      );
      item.append(label, progress, element("code", undefined, formatPpm(weight)));
      levels.append(item);
    });
    const tail = element("li", "branch-flow-tail");
    tail.append(
      element("span", undefined, "Terminal tail"),
      element("code", undefined, formatPpm(decay.tailPpm)),
    );
    levels.append(tail);
    card.append(levels);
    directions.append(card);
  }
  section.append(
    directions,
    element(
      "p",
      "branch-flow-note",
      "Empty nearer depths never enlarge a farther bucket. Missing evidence, caps, dust, and the tail go only to the prospectively named commons or refund destination.",
    ),
  );
  return section;
}

function renderBranchFlow(
  root: HTMLElement,
  flow: ConstructiveIntelligenceBranchFlow,
): void {
  const panel = element("article", "branch-flow-panel");
  panel.dataset.assurance = flow.assurance;
  const header = element("div", "branch-flow-panel-head");
  const copy = element("div");
  copy.append(
    element("span", "card-kicker", "Reference policy · static v1"),
    element("h3", undefined, "Value may flow. The envelope may not grow."),
    element("p", undefined, flow.purpose),
  );
  header.append(copy, renderAssurance(flow));
  const body = element("div", "branch-flow-body");
  body.append(renderSplit(flow), renderDecay(flow));

  const footer = element("footer", "branch-flow-footer");
  const digests = element("div", "branch-flow-digests");
  digests.append(
    element("span", undefined, "Reviewed static bytes"),
    element("code", undefined, `sha256:${BRANCH_FLOW_SHA256}`),
    element("span", undefined, "Reference policy"),
    element("code", undefined, flow.referencePolicy.policyDigest),
  );
  const links = element("div", "branch-flow-links");
  const raw = element("a", "button button-ghost", "Raw standard ↗");
  raw.href = BRANCH_FLOW_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  const specification = element("a", "button button-ghost", "Read the design ↗");
  specification.href =
    `https://github.com/cambridgetcg/zerone-core/blob/main/${flow.specification}`;
  specification.target = "_blank";
  specification.rel = "noreferrer";
  links.append(raw, specification);
  footer.append(digests, links);
  panel.append(header, body, footer);
  root.replaceChildren(panel);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseBranchFlow(
  root: HTMLElement,
  options: BranchFlowFetchOptions = {},
): Promise<void> {
  const load = async (): Promise<void> => {
    root.setAttribute("aria-busy", "true");
    try {
      const flow = await fetchBranchFlow(options);
      renderBranchFlow(root, flow);
    } catch (error) {
      root.setAttribute("aria-busy", "false");
      const state = element("div", "branch-flow-load-error");
      state.setAttribute("role", "alert");
      state.append(
        element("strong", undefined, "The static Branch Flow profile could not be loaded."),
        element(
          "p",
          undefined,
          error instanceof Error
            ? error.message
            : "The response was unavailable or invalid.",
        ),
      );
      const actions = element("div", "branch-flow-load-actions");
      const retry = element("button", "button button-primary", "Try again");
      retry.type = "button";
      retry.addEventListener("click", () => {
        retry.disabled = true;
        retry.textContent = "Trying again…";
        void load();
      });
      const source = element("a", "button button-ghost", "Open raw JSON");
      source.href = BRANCH_FLOW_ENDPOINT;
      source.target = "_blank";
      source.rel = "noreferrer";
      actions.append(retry, source);
      state.append(actions);
      root.replaceChildren(state);
    }
  };
  await load();
}
