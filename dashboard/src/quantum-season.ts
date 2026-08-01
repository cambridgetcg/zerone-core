/// <reference lib="dom" />

export const QUANTUM_SEASON_ENDPOINT =
  "/standards/constructive-intelligence-quantum-qec.v0.json";
export const QUANTUM_SEASON_MAX_BYTES = 131_072;
export const QUANTUM_SEASON_SHA256 =
  "a1f15eb2cb802bca2221c0c6f96dfbb26badad870b757746c04537ac1edc5386";
const MONEY_KARMA_CONSTITUTION_SCHEMA = "zerone.money-karma.constitution/v1";
const MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256 =
  "sha256:f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24";

export type QuantumStage = "foundation" | "assurance" | "quest";
export type QuantumEvidence = "E2" | "E3" | "E5";
export type QuantumRewardEligibility =
  | "qualification-only"
  | "sponsor-milestones";

export interface QuantumNode {
  id: string;
  title: string;
  stage: QuantumStage;
  domain: "mathematics" | "assurance" | "quests";
  summary: string;
  prerequisites: string[];
  attainmentEvidence: QuantumEvidence;
  rewardEligibility: QuantumRewardEligibility;
  defaultDisclosureLane: "open-construction" | "controlled-operations";
  artifactRequirements: string[];
  revalidationTriggers: string[];
  standardIds?: string[];
  acceptance?: QuantumAcceptance;
}
export interface QuantumFixture {
  id: string;
  n: number;
  k: number;
  distance: number;
}

interface QuantumCoverageBase {
  id: string;
  cellDefinition: string;
  analysisMode: "bernoulli-logical-failure" | "latency-quantile-and-deadline";
  minimumEffectiveClusters: 3;
  minimumOrganizationRoots: 2;
  minimumImplementationRoots: 2;
  minimumExecutionEnvironments: 2;
  minimumCasesPerCell: number;
  confidenceLevelBps: 9900;
  confidenceProcedure: string;
  requiresCheckerOrCorpusDigest: true;
  requiresCircuitAndArtifactDigests: true;
  requiresConfidenceProcedureDigest: true;
}

export type QuantumCoverageTarget = QuantumCoverageBase & (
  | {
      analysisMode: "bernoulli-logical-failure";
      minimumLogicalFailuresPerCell: 100;
      confidenceProcedure: "two-sided-binomial-interval";
      maximumRelativeHalfWidthBps: 3000;
    }
  | {
      analysisMode: "latency-quantile-and-deadline";
      latencyEstimand: "precommitted-quantile-and-deadline";
      confidenceProcedure: "two-sided-quantile-interval-or-one-sided-deadline-miss-upper-bound";
      zeroDeadlineMissesMaySatisfyMeasurementCompleteness: true;
    }
);

export interface QuantumAcceptance {
  targetEvidence: "E5";
  scopeBounds: string[];
  scopeHash: string;
  fixtures: QuantumFixture[];
  physicalErrorGrid: string[];
  cellDefinition: string;
  requiredCaseBindings: string[];
  circuitProvenance: "stim-circuits-from-version-of-record-references-27-and-50";
  distance10And12CircuitEnsembleSize: 24;
  computeCapExhaustion: "inconclusive-no-pass";
  rareEventAlternative: {
    appliesTo: "logical-error-cells-only";
    condition: "separately-reviewed-unbiased-estimator-only";
    replacesDirectGates: [
      "minimum-cases-per-cell",
      "minimum-logical-failures-per-cell",
    ];
    preservesAllOtherCoverageAndCaseBindings: true;
    confidenceLevelBps: 9900;
    confidenceProcedure: "two-sided-estimator-specific-interval";
    maximumRelativeHalfWidthBps: 3000;
    methodDigestRequired: true;
    independentReviewReceiptDigestRequired: true;
    unbiasedEstimatorRequired: true;
    varianceMethodDigestRequired: true;
    coverageValidationDigestRequired: true;
    mayEmitPerformancePass: false;
  };
  coverageTargets: QuantumCoverageTarget[];
  minimumEffectiveClusters: 3;
  minimumOrganizationRoots: 2;
  minimumImplementationRoots: 2;
  minimumExecutionEnvironments: 2;
  adoptionReceiptTypes: string[];
  privateEscalationRequired: true;
  prepublicationTriageRequired: true;
}

interface QuantumLensLevel {
  level: string;
  name: string;
  requires: string;
  assignable: false;
}

interface QuantumMilestone {
  level: string;
  name: string;
  rewardBps: number;
  treatment: string;
}

export interface QuantumSeason {
  schema: "zerone.constructive-intelligence-tree-extension/v0";
  seasonId: "quantum-qec-2026q3";
  title: string;
  snapshotDate: string;
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  base: {
    schema: "zerone.constructive-intelligence-tree/v1";
    policyVersion: "1.0.0";
    endpoint: "/standards/constructive-intelligence-tree.v1.json";
    documentSha256: string;
    policySha256: string;
  };
  constitutionBinding: {
    schema: typeof MONEY_KARMA_CONSTITUTION_SCHEMA;
    documentSha256: typeof MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256;
  };
  releaseBoundary: Record<string, false>;
  performanceDecision: {
    mode: "MEASUREMENT_COMPLETENESS_ONLY";
    mayEmitPerformancePass: false;
    measurementPrecisionAloneEstablishesPerformance: false;
    missingProspectiveRuleDisposition: "INCONCLUSIVE_NO_PASS";
    fundedCaseRequiredBindings: string[];
  };
  karma: {
    status: "OBSERVATIONAL";
    eventType: "zerone.karma.edge";
    eventRegister: "priced-coherence";
    meaning: "DOMAIN_RELATIONS_NOT_HUMAN_WORTH_OR_TRUTH";
    zeroneMintsOrCreates: false;
    assignable: false;
    operatorAssignable: false;
    founderAssignable: false;
    observationsFallible: true;
    observationsChallengeable: true;
    recordingClaimsRelationOwnership: false;
    rawEventsEstablishCandidateStatus: false;
    rawEventCountEstablishesCandidateStatus: false;
    rawEventCountAffectsSelectionProbability: false;
    transferable: false;
    scalarRank: false;
    truthOracle: false;
    payoutWeight: false;
    voteWeight: false;
    founderReservedPower: false;
    futureUse: "domain-scoped-controller-capped-randomized-candidate-filter-only";
    futureCandidateFilterRequirements: {
      runtimeEnforced: false;
      sameControllerEdgesExcluded: true;
      selfEdgesExcluded: true;
      reciprocalEdgesExcluded: true;
      correlatedFunderEdgesExcluded: true;
      controllerMergesOnlyReduceUnits: true;
      maximumLotteryUnitsPerController: 1;
      candidateSetFrozenBeforeRandomness: true;
      unbiasedRandomnessRequired: true;
      operatorOverrideAllowed: false;
      countProportionalProbabilityAllowed: false;
    };
    activationRequires: string[];
  };
  breakthroughLens: QuantumLensLevel[];
  rewardPolicy: {
    status: "UNFUNDED_TEMPLATE";
    denom: "uzrn";
    fundedAmount: "0";
    escrowReceipt: null;
    claimable: false;
    founderShareBps: 0;
    founderReservedSeats: 0;
    karmaWeightBps: 0;
    rewardDirectlyGrantsGovernanceAuthority: false;
    rewardDenomIsBondableUnderCurrentProtocol: true;
    skillUnlockCreatesReward: false;
    timeAloneUnlocksEvidence: false;
    milestones: QuantumMilestone[];
    challengeReserveBps: 1500;
    attributionBps: Record<string, number>;
  };
  rewardAccounting: {
    accountingBoundary: {
      milestoneAndAttributionAxesAreAdditive: false;
      crossAxisAllocationRule: "UNDEFINED_BLOCKS_FUNDING";
      roundingRule: null;
      escrowCompartmentsBound: false;
      singleSettlementImplemented: false;
      verifiedCostCapAmount: null;
      reviewerBudgetCapAmount: null;
      futureReviewerBudgetMustBeOutcomeIndependent: true;
      roleCollapseRule: null;
      deterministicRefundRule: null;
      reviewAttributionPaysAdjudicator: false;
      unusedChallengeReserveRoute: null;
    };
    nodeEligibilitySemantics: "DISPLAY_ROUTING_ONLY_NO_PRESENT_QUALIFICATION_OR_ENTITLEMENT";
  };
  standards: Array<{
    canonicalId: string;
    authority: "Nature Communications";
    title: string;
    revision: "version of record 2026-05-01";
    authorityStatus: "open-access version of record";
    normalizedMaturity: "published";
    specification: string;
    statusCheckedAt: "2026-08-01";
    reviewAfter: "2026-09-01";
    treatment: "reproduction-target-not-truth-oracle";
  }>;
  nodes: QuantumNode[];
}

export interface QuantumSeasonFreshness {
  earliestReviewAfter: string | null;
  expiredStandardCount: number;
  isExpiredForActiveUse: boolean;
}

export interface QuantumSeasonFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class QuantumSeasonDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "QuantumSeasonDataError";
  }
}

type JsonObject = Record<string, unknown>;

const EXPECTED_TOP_LEVEL = [
  "schema",
  "seasonId",
  "title",
  "snapshotDate",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "base",
  "constitutionBinding",
  "releaseBoundary",
  "performanceDecision",
  "karma",
  "breakthroughLens",
  "rewardPolicy",
  "rewardAccounting",
  "standards",
  "nodes",
] as const;
const RELEASE_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
] as const;
const PERFORMANCE_DECISION_KEYS = [
  "mode",
  "mayEmitPerformancePass",
  "measurementPrecisionAloneEstablishesPerformance",
  "missingProspectiveRuleDisposition",
  "fundedCaseRequiredBindings",
] as const;
const BASE_KEYS = [
  "schema",
  "policyVersion",
  "endpoint",
  "documentSha256",
  "policySha256",
] as const;
const CONSTITUTION_BINDING_KEYS = ["schema", "documentSha256"] as const;
const KARMA_KEYS = [
  "status",
  "eventType",
  "eventRegister",
  "meaning",
  "zeroneMintsOrCreates",
  "assignable",
  "operatorAssignable",
  "founderAssignable",
  "observationsFallible",
  "observationsChallengeable",
  "recordingClaimsRelationOwnership",
  "rawEventsEstablishCandidateStatus",
  "rawEventCountEstablishesCandidateStatus",
  "rawEventCountAffectsSelectionProbability",
  "transferable",
  "scalarRank",
  "truthOracle",
  "payoutWeight",
  "voteWeight",
  "founderReservedPower",
  "futureUse",
  "futureCandidateFilterRequirements",
  "activationRequires",
] as const;
const FUTURE_CANDIDATE_FILTER_KEYS = [
  "runtimeEnforced",
  "sameControllerEdgesExcluded",
  "selfEdgesExcluded",
  "reciprocalEdgesExcluded",
  "correlatedFunderEdgesExcluded",
  "controllerMergesOnlyReduceUnits",
  "maximumLotteryUnitsPerController",
  "candidateSetFrozenBeforeRandomness",
  "unbiasedRandomnessRequired",
  "operatorOverrideAllowed",
  "countProportionalProbabilityAllowed",
] as const;
const LENS_KEYS = ["level", "name", "requires", "assignable"] as const;
const REWARD_KEYS = [
  "status",
  "denom",
  "fundedAmount",
  "escrowReceipt",
  "claimable",
  "founderShareBps",
  "founderReservedSeats",
  "karmaWeightBps",
  "rewardDirectlyGrantsGovernanceAuthority",
  "rewardDenomIsBondableUnderCurrentProtocol",
  "skillUnlockCreatesReward",
  "timeAloneUnlocksEvidence",
  "milestones",
  "challengeReserveBps",
  "attributionBps",
] as const;
const REWARD_ACCOUNTING_KEYS = ["accountingBoundary", "nodeEligibilitySemantics"] as const;
const ACCOUNTING_BOUNDARY_KEYS = [
  "milestoneAndAttributionAxesAreAdditive",
  "crossAxisAllocationRule",
  "roundingRule",
  "escrowCompartmentsBound",
  "singleSettlementImplemented",
  "verifiedCostCapAmount",
  "reviewerBudgetCapAmount",
  "futureReviewerBudgetMustBeOutcomeIndependent",
  "roleCollapseRule",
  "deterministicRefundRule",
  "reviewAttributionPaysAdjudicator",
  "unusedChallengeReserveRoute",
] as const;
const MILESTONE_KEYS = ["level", "name", "rewardBps", "treatment"] as const;
const ATTRIBUTION_KEYS = [
  "originatingArtifact",
  "independentReplication",
  "independentReview",
  "downstreamAdoption",
  "falsificationAndChallenge",
  "safetyAndMaintenance",
] as const;
const STANDARD_KEYS = [
  "canonicalId",
  "authority",
  "title",
  "revision",
  "authorityStatus",
  "normalizedMaturity",
  "specification",
  "statusCheckedAt",
  "reviewAfter",
  "treatment",
] as const;
const ACCEPTANCE_KEYS = [
  "targetEvidence",
  "scopeBounds",
  "scopeHash",
  "fixtures",
  "physicalErrorGrid",
  "cellDefinition",
  "requiredCaseBindings",
  "circuitProvenance",
  "distance10And12CircuitEnsembleSize",
  "computeCapExhaustion",
  "rareEventAlternative",
  "coverageTargets",
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "adoptionReceiptTypes",
  "privateEscalationRequired",
  "prepublicationTriageRequired",
] as const;
const FIXTURE_KEYS = ["id", "n", "k", "distance"] as const;
const RARE_EVENT_KEYS = [
  "appliesTo",
  "condition",
  "replacesDirectGates",
  "preservesAllOtherCoverageAndCaseBindings",
  "confidenceLevelBps",
  "confidenceProcedure",
  "maximumRelativeHalfWidthBps",
  "methodDigestRequired",
  "independentReviewReceiptDigestRequired",
  "unbiasedEstimatorRequired",
  "varianceMethodDigestRequired",
  "coverageValidationDigestRequired",
  "mayEmitPerformancePass",
] as const;
const COVERAGE_COMMON_KEYS = [
  "id",
  "cellDefinition",
  "analysisMode",
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "minimumCasesPerCell",
  "confidenceLevelBps",
  "confidenceProcedure",
  "requiresCheckerOrCorpusDigest",
  "requiresCircuitAndArtifactDigests",
  "requiresConfidenceProcedureDigest",
] as const;
const LOGICAL_COVERAGE_KEYS = [
  ...COVERAGE_COMMON_KEYS,
  "minimumLogicalFailuresPerCell",
  "maximumRelativeHalfWidthBps",
] as const;
const LATENCY_COVERAGE_KEYS = [
  ...COVERAGE_COMMON_KEYS,
  "latencyEstimand",
  "zeroDeadlineMissesMaySatisfyMeasurementCompleteness",
] as const;
const EXPECTED_MILESTONES = [
  ["E0", "committed", 0, "precedence-only"],
  ["E1", "inspectable", 0, "verified-cost-only"],
  ["E2", "class-verified", 1500, "milestone"],
  ["E3", "independently-reproduced", 2000, "milestone"],
  ["E4", "adversarially-survived-or-fix-tested", 1500, "milestone"],
  ["E5", "independently-adopted", 2500, "milestone"],
  ["E6", "maintained", 1000, "milestone"],
] as const;
const EXPECTED_ATTRIBUTION = {
  originatingArtifact: 3000,
  independentReplication: 2500,
  independentReview: 1000,
  downstreamAdoption: 2500,
  falsificationAndChallenge: 700,
  safetyAndMaintenance: 300,
} as const;
const EXPECTED_FUNDED_CASE_BINDINGS = [
  "baseline-comparator-digest",
  "baseline-comparator-independent-review-receipt",
  "comparison-direction",
  "confidence-bound-decision-rule",
  "effect-size-or-equivalence-margin",
  "estimand-null-direction",
  "latency-deadline-and-quantile",
  "multi-metric-tradeoff-or-pareto-rule",
  "multiple-comparison-policy",
  "negative-result-routing",
  "resource-match",
] as const;
const EXPECTED_FUTURE_CANDIDATE_FILTER = {
  runtimeEnforced: false,
  sameControllerEdgesExcluded: true,
  selfEdgesExcluded: true,
  reciprocalEdgesExcluded: true,
  correlatedFunderEdgesExcluded: true,
  controllerMergesOnlyReduceUnits: true,
  maximumLotteryUnitsPerController: 1,
  candidateSetFrozenBeforeRandomness: true,
  unbiasedRandomnessRequired: true,
  operatorOverrideAllowed: false,
  countProportionalProbabilityAllowed: false,
} as const;
const EXPECTED_SCOPE = [
  "baseline=bposd-order-zero-randomized-serial-at-frozen-commit",
  "bb-fixtures=[[72,12,6],[90,8,10],[144,12,12]]",
  "cell=fixture-x-physical-error-x-implementation-root-x-execution-environment",
  "circuit-and-artifact-digests=required-before-case-opens",
  "circuit-provenance=stim-version-of-record-references-27-and-50",
  "compute-cap-exhaustion=inconclusive-no-pass",
  "confidence=logical-failure-two-sided-99-percent-relative-half-width-lte-0.30",
  "d10-and-d12-circuit-ensemble-size=24",
  "event-floor=100-logical-failures-per-logical-error-cell",
  "latency=precommitted-quantile-and-deadline-at-99-percent-confidence-zero-misses-may-satisfy-measurement-completeness-no-performance-pass",
  "minimum-cases=coverage-target-specific-per-cell",
  "noise-model=uniform-depolarizing-circuit-level",
  "physical-error-grid=0.001,0.002,0.003,0.004,0.005,0.006",
  "primary-metrics=logical-error-rate,latency",
  "random-seeds=committed-before-execution",
  "rare-event-alternative=separately-reviewed-unbiased-estimator-only",
  "stopping-rule=event-and-confidence-gates-before-compute-cap",
  "tuning-access-and-resource-budget=frozen-before-case-opens",
] as const;
const EXPECTED_SCOPE_HASH =
  "6397a1b7d3a0979e24676fa91bd9642c019a1f266a9ddec045a0842cf35cbf41";
const EXPECTED_FIXTURES = [
  { id: "bb-72-12-6", n: 72, k: 12, distance: 6 },
  { id: "bb-90-8-10", n: 90, k: 8, distance: 10 },
  { id: "bb-144-12-12", n: 144, k: 12, distance: 12 },
] as const;
const EXPECTED_PHYSICAL_ERROR_GRID = [
  "0.001",
  "0.002",
  "0.003",
  "0.004",
  "0.005",
  "0.006",
] as const;
const CELL_DEFINITION =
  "fixture-x-physical-error-x-implementation-root-x-execution-environment";
const EXPECTED_CASE_BINDINGS = [
  "baseline-artifact-sha256",
  "circuit-bundle-sha256-per-fixture",
  "confidence-procedure-sha256",
  "decoder-artifact-sha256-per-implementation-root",
  "environment-manifest-sha256-per-execution-environment",
  "fixture-matrix-sha256-per-fixture",
  "noise-corpus-sha256-per-cell",
  "resource-accounting-method-sha256",
  "stopping-rule-sha256",
] as const;
const EXPECTED_KARMA_GATES = [
  "appeal-and-reversible-pilot",
  "controller-clustering-and-pair-caps",
  "independent-validator-stake-host-and-upgrade-control",
  "minimum-organization-and-implementation-diversity",
  "prospective-named-upgrade-replay-tests-and-activation-height",
  "public-reasons-delay-and-challenge-window",
  "role-separation-and-recipient-conflict-exclusion",
] as const;
const EXPECTED_LENS = [
  ["reproduction", "Known-result capability demonstration"],
  ["independent-replication", "REPLICATES plus E3"],
  ["bounded-extension", "Prior-art delta plus E3"],
  ["enabling-artifact", "Independent E5 adoption"],
  ["breakthrough", "Retrospective PROVES or DISPROVES, E3, prior-art delta, and independent adoption or descendant impact"],
  ["field-shift", "Multiple independent IMPLEMENTS, DEPLOYS, or MAINTAINS descendants through E5 or E6"],
] as const;
const EXPECTED_NODE_IDS = [
  "assurance-quantum-benchmarking-metrology@1",
  "assurance-quantum-independent-replication@1",
  "assurance-quantum-numerical-reproducibility@1",
  "math-complex-linear-algebra@1",
  "math-differential-equations-fourier@1",
  "math-quantum-error-correction@1",
  "math-quantum-many-body-symmetry@1",
  "math-quantum-measurement-entanglement@1",
  "math-quantum-open-systems-control@1",
  "math-quantum-states-dynamics@1",
  "math-statistical-inference-metrology@1",
  "math-tensor-spectral-operators@1",
  "quest-quantum-decoder-correlated-noise@1",
] as const;
const BASE_NODE_META = new Map<string, { stage: QuantumStage; depth: number }>([
  ["assurance-conformance-interoperability@1", { stage: "assurance", depth: 6 }],
  ["assurance-formal-verification@1", { stage: "assurance", depth: 4 }],
  ["math-probability-information-complexity@1", { stage: "foundation", depth: 2 }],
  ["math-proofcraft@1", { stage: "foundation", depth: 1 }],
]);
const STAGE_RANK: Readonly<Record<QuantumStage, number>> = {
  foundation: 0,
  assurance: 2,
  quest: 4,
};
const EXPECTED_EXTENSION_EDGE_COUNT = 24;
const EXPECTED_COMBINED_MAX_DEPTH = 8;
const NODE_ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;

function fail(path: string, message: string): never {
  throw new QuantumSeasonDataError(`${path}: ${message}`);
}

function object(value: unknown, path: string): JsonObject {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value as JsonObject;
}

function exactKeys(value: JsonObject, expected: readonly string[], path: string): void {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (JSON.stringify(actual) !== JSON.stringify(wanted)) {
    fail(path, "contains unknown or missing fields");
  }
}

function string(value: unknown, path: string, maximum = 8_192): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, `expected a non-empty string of at most ${maximum} characters`);
  }
  return value;
}

function isoDate(value: unknown, path: string): string {
  const result = string(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(result)) fail(path, "expected YYYY-MM-DD");
  const parsed = Date.parse(`${result}T00:00:00Z`);
  if (!Number.isFinite(parsed) || new Date(parsed).toISOString().slice(0, 10) !== result) {
    fail(path, "expected a real calendar date");
  }
  return result;
}

function strings(value: unknown, path: string, minimum = 0, maximum = 64): string[] {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) {
    fail(path, `expected ${minimum} through ${maximum} strings`);
  }
  const result = value.map((item, index) => string(item, `${path}[${index}]`, 1_024));
  if (new Set(result).size !== result.length) fail(path, "contains duplicates");
  return result;
}

function exact<T>(value: unknown, expected: T, path: string): T {
  if (value !== expected) fail(path, `must equal ${JSON.stringify(expected)}`);
  return expected;
}

function parseNode(value: unknown, index: number): QuantumNode {
  const path = `$.nodes[${index}]`;
  const source = object(value, path);
  const stage = string(source.stage, `${path}.stage`) as QuantumStage;
  const quest = stage === "quest";
  const keys = [
    "id",
    "title",
    "stage",
    "domain",
    "summary",
    "prerequisites",
    "attainmentEvidence",
    "rewardEligibility",
    "defaultDisclosureLane",
    "artifactRequirements",
    "revalidationTriggers",
    ...(quest ? ["standardIds", "acceptance"] : []),
  ];
  exactKeys(source, keys, path);
  const id = string(source.id, `${path}.id`, 128);
  if (!NODE_ID_PATTERN.test(id)) fail(`${path}.id`, "is not a versioned node ID");
  if (!["foundation", "assurance", "quest"].includes(stage)) fail(`${path}.stage`, "unsupported stage");
  const expectedDomain = stage === "foundation" ? "mathematics" : stage === "assurance" ? "assurance" : "quests";
  exact(source.domain, expectedDomain, `${path}.domain`);
  const evidence = exact(
    source.attainmentEvidence,
    stage === "foundation" ? "E2" : stage === "assurance" ? "E3" : "E5",
    `${path}.attainmentEvidence`,
  ) as QuantumEvidence;
  const eligibility = exact(
    source.rewardEligibility,
    quest ? "sponsor-milestones" : "qualification-only",
    `${path}.rewardEligibility`,
  ) as QuantumRewardEligibility;
  const lane = string(source.defaultDisclosureLane, `${path}.defaultDisclosureLane`);
  if (lane !== "open-construction" && lane !== "controlled-operations") {
    fail(`${path}.defaultDisclosureLane`, "unsupported disclosure lane");
  }
  const prerequisites = strings(source.prerequisites, `${path}.prerequisites`, 1, 8);
  if (prerequisites.some((item, itemIndex) => itemIndex > 0 && prerequisites[itemIndex - 1]! > item)) {
    fail(`${path}.prerequisites`, "must be lexicographically sorted");
  }
  const node: QuantumNode = {
    id,
    title: string(source.title, `${path}.title`, 160),
    stage,
    domain: expectedDomain,
    summary: string(source.summary, `${path}.summary`, 1_024),
    prerequisites,
    attainmentEvidence: evidence,
    rewardEligibility: eligibility,
    defaultDisclosureLane: lane,
    artifactRequirements: strings(source.artifactRequirements, `${path}.artifactRequirements`, 2, 8),
    revalidationTriggers: strings(source.revalidationTriggers, `${path}.revalidationTriggers`, 1, 4),
  };
  if (quest) {
    node.standardIds = strings(source.standardIds, `${path}.standardIds`, 1, 1);
    exact(node.standardIds[0], "doi:10.1038/s41467-026-70556-3", `${path}.standardIds[0]`);
    const acceptance = object(source.acceptance, `${path}.acceptance`);
    exactKeys(acceptance, ACCEPTANCE_KEYS, `${path}.acceptance`);
    exact(acceptance.targetEvidence, "E5", `${path}.acceptance.targetEvidence`);
    for (const key of [
      "minimumEffectiveClusters",
      "minimumOrganizationRoots",
      "minimumImplementationRoots",
      "minimumExecutionEnvironments",
    ]) {
      const expected = key === "minimumEffectiveClusters" ? 3 : 2;
      exact(acceptance[key], expected, `${path}.acceptance.${key}`);
    }
    exact(acceptance.privateEscalationRequired, true, `${path}.acceptance.privateEscalationRequired`);
    exact(acceptance.prepublicationTriageRequired, true, `${path}.acceptance.prepublicationTriageRequired`);
    const scopeBounds = strings(
      acceptance.scopeBounds,
      `${path}.acceptance.scopeBounds`,
      EXPECTED_SCOPE.length,
      EXPECTED_SCOPE.length,
    );
    if (JSON.stringify(scopeBounds) !== JSON.stringify(EXPECTED_SCOPE)) {
      fail(
        `${path}.acceptance.scopeBounds`,
        "must preserve the reviewed precommitted scope",
      );
    }
    exact(
      acceptance.scopeHash,
      EXPECTED_SCOPE_HASH,
      `${path}.acceptance.scopeHash`,
    );
    if (!Array.isArray(acceptance.fixtures) || acceptance.fixtures.length !== EXPECTED_FIXTURES.length) {
      fail(`${path}.acceptance.fixtures`, "must contain the exact three reviewed BB fixtures");
    }
    acceptance.fixtures.forEach((entry, fixtureIndex) => {
      const fixturePath = `${path}.acceptance.fixtures[${fixtureIndex}]`;
      const fixture = object(entry, fixturePath);
      exactKeys(fixture, FIXTURE_KEYS, fixturePath);
      const expected = EXPECTED_FIXTURES[fixtureIndex];
      if (!expected) fail(fixturePath, "unexpected fixture");
      exact(fixture.id, expected.id, `${fixturePath}.id`);
      exact(fixture.n, expected.n, `${fixturePath}.n`);
      exact(fixture.k, expected.k, `${fixturePath}.k`);
      exact(fixture.distance, expected.distance, `${fixturePath}.distance`);
    });
    const physicalErrorGrid = strings(
      acceptance.physicalErrorGrid,
      `${path}.acceptance.physicalErrorGrid`,
      EXPECTED_PHYSICAL_ERROR_GRID.length,
      EXPECTED_PHYSICAL_ERROR_GRID.length,
    );
    if (JSON.stringify(physicalErrorGrid) !== JSON.stringify(EXPECTED_PHYSICAL_ERROR_GRID)) {
      fail(`${path}.acceptance.physicalErrorGrid`, "must bind p=0.001 through 0.006");
    }
    exact(acceptance.cellDefinition, CELL_DEFINITION, `${path}.acceptance.cellDefinition`);
    const requiredCaseBindings = strings(
      acceptance.requiredCaseBindings,
      `${path}.acceptance.requiredCaseBindings`,
      EXPECTED_CASE_BINDINGS.length,
      EXPECTED_CASE_BINDINGS.length,
    );
    if (JSON.stringify(requiredCaseBindings) !== JSON.stringify(EXPECTED_CASE_BINDINGS)) {
      fail(`${path}.acceptance.requiredCaseBindings`, "must content-bind circuits and acceptance artifacts");
    }
    exact(acceptance.circuitProvenance, "stim-circuits-from-version-of-record-references-27-and-50", `${path}.acceptance.circuitProvenance`);
    exact(acceptance.distance10And12CircuitEnsembleSize, 24, `${path}.acceptance.distance10And12CircuitEnsembleSize`);
    exact(acceptance.computeCapExhaustion, "inconclusive-no-pass", `${path}.acceptance.computeCapExhaustion`);
    const rareEvent = object(acceptance.rareEventAlternative, `${path}.acceptance.rareEventAlternative`);
    exactKeys(rareEvent, RARE_EVENT_KEYS, `${path}.acceptance.rareEventAlternative`);
    exact(rareEvent.appliesTo, "logical-error-cells-only", `${path}.acceptance.rareEventAlternative.appliesTo`);
    exact(rareEvent.condition, "separately-reviewed-unbiased-estimator-only", `${path}.acceptance.rareEventAlternative.condition`);
    const replacedGates = strings(
      rareEvent.replacesDirectGates,
      `${path}.acceptance.rareEventAlternative.replacesDirectGates`,
      2,
      2,
    );
    if (JSON.stringify(replacedGates) !== JSON.stringify([
      "minimum-cases-per-cell",
      "minimum-logical-failures-per-cell",
    ])) {
      fail(
        `${path}.acceptance.rareEventAlternative.replacesDirectGates`,
        "may replace only the direct case and logical-failure floors",
      );
    }
    exact(rareEvent.confidenceLevelBps, 9900, `${path}.acceptance.rareEventAlternative.confidenceLevelBps`);
    exact(
      rareEvent.confidenceProcedure,
      "two-sided-estimator-specific-interval",
      `${path}.acceptance.rareEventAlternative.confidenceProcedure`,
    );
    exact(
      rareEvent.maximumRelativeHalfWidthBps,
      3000,
      `${path}.acceptance.rareEventAlternative.maximumRelativeHalfWidthBps`,
    );
    for (const key of [
      "preservesAllOtherCoverageAndCaseBindings",
      "methodDigestRequired",
      "independentReviewReceiptDigestRequired",
      "unbiasedEstimatorRequired",
      "varianceMethodDigestRequired",
      "coverageValidationDigestRequired",
    ] as const) {
      exact(rareEvent[key], true, `${path}.acceptance.rareEventAlternative.${key}`);
    }
    exact(
      rareEvent.mayEmitPerformancePass,
      false,
      `${path}.acceptance.rareEventAlternative.mayEmitPerformancePass`,
    );
    const receiptTypes = strings(acceptance.adoptionReceiptTypes, `${path}.acceptance.adoptionReceiptTypes`, 2, 2);
    if (
      JSON.stringify(receiptTypes) !==
      JSON.stringify(["maintained-fixture", "upstream-merge"])
    ) {
      fail(`${path}.acceptance.adoptionReceiptTypes`, "independent adoption evidence is required");
    }
    if (!Array.isArray(acceptance.coverageTargets) || acceptance.coverageTargets.length !== 3) {
      fail(`${path}.acceptance.coverageTargets`, "three coverage targets are required");
    }
    const expectedCoverage = [
      { id: "baseline-regression", analysisMode: "bernoulli-logical-failure", minimumCasesPerCell: 100_000 },
      { id: "correlated-noise-logical-error-rate", analysisMode: "bernoulli-logical-failure", minimumCasesPerCell: 1_000_000 },
      { id: "matched-resource-latency", analysisMode: "latency-quantile-and-deadline", minimumCasesPerCell: 100_000 },
    ] as const;
    acceptance.coverageTargets.forEach((target, targetIndex) => {
      const coverage = object(target, `${path}.acceptance.coverageTargets[${targetIndex}]`);
      const expected = expectedCoverage[targetIndex];
      if (!expected) fail(`${path}.acceptance.coverageTargets`, "unexpected coverage target");
      const logical = expected.analysisMode === "bernoulli-logical-failure";
      exactKeys(
        coverage,
        logical ? LOGICAL_COVERAGE_KEYS : LATENCY_COVERAGE_KEYS,
        `${path}.acceptance.coverageTargets[${targetIndex}]`,
      );
      exact(coverage.id, expected.id, `${path}.acceptance.coverageTargets[${targetIndex}].id`);
      exact(coverage.cellDefinition, CELL_DEFINITION, `${path}.acceptance.coverageTargets[${targetIndex}].cellDefinition`);
      exact(coverage.analysisMode, expected.analysisMode, `${path}.acceptance.coverageTargets[${targetIndex}].analysisMode`);
      exact(coverage.minimumCasesPerCell, expected.minimumCasesPerCell, `${path}.acceptance.coverageTargets[${targetIndex}].minimumCasesPerCell`);
      exact(coverage.confidenceLevelBps, 9900, `${path}.acceptance.coverageTargets[${targetIndex}].confidenceLevelBps`);
      if (logical) {
        exact(coverage.minimumLogicalFailuresPerCell, 100, `${path}.acceptance.coverageTargets[${targetIndex}].minimumLogicalFailuresPerCell`);
        exact(coverage.confidenceProcedure, "two-sided-binomial-interval", `${path}.acceptance.coverageTargets[${targetIndex}].confidenceProcedure`);
        exact(coverage.maximumRelativeHalfWidthBps, 3000, `${path}.acceptance.coverageTargets[${targetIndex}].maximumRelativeHalfWidthBps`);
      } else {
        exact(coverage.latencyEstimand, "precommitted-quantile-and-deadline", `${path}.acceptance.coverageTargets[${targetIndex}].latencyEstimand`);
        exact(coverage.confidenceProcedure, "two-sided-quantile-interval-or-one-sided-deadline-miss-upper-bound", `${path}.acceptance.coverageTargets[${targetIndex}].confidenceProcedure`);
        exact(
          coverage.zeroDeadlineMissesMaySatisfyMeasurementCompleteness,
          true,
          `${path}.acceptance.coverageTargets[${targetIndex}].zeroDeadlineMissesMaySatisfyMeasurementCompleteness`,
        );
      }
      for (const key of [
        "requiresCheckerOrCorpusDigest",
        "requiresCircuitAndArtifactDigests",
        "requiresConfidenceProcedureDigest",
      ]) {
        exact(coverage[key], true, `${path}.acceptance.coverageTargets[${targetIndex}].${key}`);
      }
      for (const key of [
        "minimumEffectiveClusters",
        "minimumOrganizationRoots",
        "minimumImplementationRoots",
        "minimumExecutionEnvironments",
      ]) {
        const expected = key === "minimumEffectiveClusters" ? 3 : 2;
        exact(
          coverage[key],
          expected,
          `${path}.acceptance.coverageTargets[${targetIndex}].${key}`,
        );
      }
    });
    node.acceptance = acceptance as unknown as QuantumAcceptance;
  }
  return node;
}

export function parseQuantumSeason(value: unknown): QuantumSeason {
  const source = object(value, "$");
  exactKeys(source, EXPECTED_TOP_LEVEL, "$");
  exact(source.schema, "zerone.constructive-intelligence-tree-extension/v0", "$.schema");
  exact(source.seasonId, "quantum-qec-2026q3", "$.seasonId");
  string(source.title, "$.title", 160);
  exact(isoDate(source.snapshotDate, "$.snapshotDate"), "2026-08-01", "$.snapshotDate");
  for (const key of ["authoritative", "networkObserved", "rewardBearing"]) {
    exact(source[key], false, `$.${key}`);
  }
  const base = object(source.base, "$.base");
  exactKeys(base, BASE_KEYS, "$.base");
  exact(base.schema, "zerone.constructive-intelligence-tree/v1", "$.base.schema");
  exact(base.policyVersion, "1.0.0", "$.base.policyVersion");
  exact(base.endpoint, "/standards/constructive-intelligence-tree.v1.json", "$.base.endpoint");
  exact(base.documentSha256, "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf", "$.base.documentSha256");
  exact(base.policySha256, "sha256:36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7", "$.base.policySha256");
  const constitutionBinding = object(
    source.constitutionBinding,
    "$.constitutionBinding",
  );
  exactKeys(
    constitutionBinding,
    CONSTITUTION_BINDING_KEYS,
    "$.constitutionBinding",
  );
  exact(
    constitutionBinding.schema,
    MONEY_KARMA_CONSTITUTION_SCHEMA,
    "$.constitutionBinding.schema",
  );
  exact(
    constitutionBinding.documentSha256,
    MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256,
    "$.constitutionBinding.documentSha256",
  );
  const release = object(source.releaseBoundary, "$.releaseBoundary");
  exactKeys(release, RELEASE_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_KEYS) exact(release[key], false, `$.releaseBoundary.${key}`);
  const performance = object(source.performanceDecision, "$.performanceDecision");
  exactKeys(performance, PERFORMANCE_DECISION_KEYS, "$.performanceDecision");
  exact(performance.mode, "MEASUREMENT_COMPLETENESS_ONLY", "$.performanceDecision.mode");
  exact(performance.mayEmitPerformancePass, false, "$.performanceDecision.mayEmitPerformancePass");
  exact(
    performance.measurementPrecisionAloneEstablishesPerformance,
    false,
    "$.performanceDecision.measurementPrecisionAloneEstablishesPerformance",
  );
  exact(
    performance.missingProspectiveRuleDisposition,
    "INCONCLUSIVE_NO_PASS",
    "$.performanceDecision.missingProspectiveRuleDisposition",
  );
  const fundedCaseBindings = strings(
    performance.fundedCaseRequiredBindings,
    "$.performanceDecision.fundedCaseRequiredBindings",
    EXPECTED_FUNDED_CASE_BINDINGS.length,
    EXPECTED_FUNDED_CASE_BINDINGS.length,
  );
  if (JSON.stringify(fundedCaseBindings) !== JSON.stringify(EXPECTED_FUNDED_CASE_BINDINGS)) {
    fail(
      "$.performanceDecision.fundedCaseRequiredBindings",
      "must bind the complete prospective funded-case performance rule",
    );
  }
  const karma = object(source.karma, "$.karma");
  exactKeys(karma, KARMA_KEYS, "$.karma");
  exact(karma.status, "OBSERVATIONAL", "$.karma.status");
  exact(karma.eventType, "zerone.karma.edge", "$.karma.eventType");
  exact(karma.eventRegister, "priced-coherence", "$.karma.eventRegister");
  exact(
    karma.meaning,
    "DOMAIN_RELATIONS_NOT_HUMAN_WORTH_OR_TRUTH",
    "$.karma.meaning",
  );
  for (const key of [
    "zeroneMintsOrCreates",
    "assignable",
    "operatorAssignable",
    "founderAssignable",
    "recordingClaimsRelationOwnership",
    "rawEventsEstablishCandidateStatus",
    "rawEventCountEstablishesCandidateStatus",
    "rawEventCountAffectsSelectionProbability",
    "transferable",
    "scalarRank",
    "truthOracle",
    "payoutWeight",
    "voteWeight",
    "founderReservedPower",
  ]) {
    exact(karma[key], false, `$.karma.${key}`);
  }
  for (const key of ["observationsFallible", "observationsChallengeable"]) {
    exact(karma[key], true, `$.karma.${key}`);
  }
  exact(
    karma.futureUse,
    "domain-scoped-controller-capped-randomized-candidate-filter-only",
    "$.karma.futureUse",
  );
  const futureFilter = object(
    karma.futureCandidateFilterRequirements,
    "$.karma.futureCandidateFilterRequirements",
  );
  exactKeys(
    futureFilter,
    FUTURE_CANDIDATE_FILTER_KEYS,
    "$.karma.futureCandidateFilterRequirements",
  );
  for (const key of FUTURE_CANDIDATE_FILTER_KEYS) {
    exact(
      futureFilter[key],
      EXPECTED_FUTURE_CANDIDATE_FILTER[key],
      `$.karma.futureCandidateFilterRequirements.${key}`,
    );
  }
  const karmaGates = strings(
    karma.activationRequires,
    "$.karma.activationRequires",
    7,
    7,
  );
  if (JSON.stringify(karmaGates) !== JSON.stringify(EXPECTED_KARMA_GATES)) {
    fail("$.karma.activationRequires", "must retain every bounded future-governance gate");
  }
  const reward = object(source.rewardPolicy, "$.rewardPolicy");
  exactKeys(reward, REWARD_KEYS, "$.rewardPolicy");
  exact(reward.status, "UNFUNDED_TEMPLATE", "$.rewardPolicy.status");
  exact(reward.denom, "uzrn", "$.rewardPolicy.denom");
  exact(reward.fundedAmount, "0", "$.rewardPolicy.fundedAmount");
  exact(reward.escrowReceipt, null, "$.rewardPolicy.escrowReceipt");
  for (const key of [
    "claimable",
    "rewardDirectlyGrantsGovernanceAuthority",
    "skillUnlockCreatesReward",
    "timeAloneUnlocksEvidence",
  ]) {
    exact(reward[key], false, `$.rewardPolicy.${key}`);
  }
  exact(
    reward.rewardDenomIsBondableUnderCurrentProtocol,
    true,
    "$.rewardPolicy.rewardDenomIsBondableUnderCurrentProtocol",
  );
  for (const key of ["founderShareBps", "founderReservedSeats", "karmaWeightBps"]) {
    exact(reward[key], 0, `$.rewardPolicy.${key}`);
  }
  if (!Array.isArray(reward.milestones) || reward.milestones.length !== 7) {
    fail("$.rewardPolicy.milestones", "must contain E0 through E6");
  }
  reward.milestones.forEach((entry, index) => {
    const milestone = object(entry, `$.rewardPolicy.milestones[${index}]`);
    exactKeys(
      milestone,
      MILESTONE_KEYS,
      `$.rewardPolicy.milestones[${index}]`,
    );
    const expected = EXPECTED_MILESTONES[index];
    if (!expected) fail("$.rewardPolicy.milestones", "unexpected milestone");
    exact(
      milestone.level,
      expected[0],
      `$.rewardPolicy.milestones[${index}].level`,
    );
    exact(
      milestone.name,
      expected[1],
      `$.rewardPolicy.milestones[${index}].name`,
    );
    exact(
      milestone.rewardBps,
      expected[2],
      `$.rewardPolicy.milestones[${index}].rewardBps`,
    );
    exact(
      milestone.treatment,
      expected[3],
      `$.rewardPolicy.milestones[${index}].treatment`,
    );
  });
  exact(reward.challengeReserveBps, 1500, "$.rewardPolicy.challengeReserveBps");
  const attribution = object(reward.attributionBps, "$.rewardPolicy.attributionBps");
  exactKeys(attribution, ATTRIBUTION_KEYS, "$.rewardPolicy.attributionBps");
  for (const key of ATTRIBUTION_KEYS) {
    exact(
      attribution[key],
      EXPECTED_ATTRIBUTION[key],
      `$.rewardPolicy.attributionBps.${key}`,
    );
  }
  const rewardAccounting = object(source.rewardAccounting, "$.rewardAccounting");
  exactKeys(rewardAccounting, REWARD_ACCOUNTING_KEYS, "$.rewardAccounting");
  exact(
    rewardAccounting.nodeEligibilitySemantics,
    "DISPLAY_ROUTING_ONLY_NO_PRESENT_QUALIFICATION_OR_ENTITLEMENT",
    "$.rewardAccounting.nodeEligibilitySemantics",
  );
  const accountingBoundary = object(
    rewardAccounting.accountingBoundary,
    "$.rewardAccounting.accountingBoundary",
  );
  exactKeys(
    accountingBoundary,
    ACCOUNTING_BOUNDARY_KEYS,
    "$.rewardAccounting.accountingBoundary",
  );
  for (const key of [
    "milestoneAndAttributionAxesAreAdditive",
    "escrowCompartmentsBound",
    "singleSettlementImplemented",
    "reviewAttributionPaysAdjudicator",
  ]) {
    exact(
      accountingBoundary[key],
      false,
      `$.rewardAccounting.accountingBoundary.${key}`,
    );
  }
  exact(
    accountingBoundary.crossAxisAllocationRule,
    "UNDEFINED_BLOCKS_FUNDING",
    "$.rewardAccounting.accountingBoundary.crossAxisAllocationRule",
  );
  for (const key of [
    "roundingRule",
    "verifiedCostCapAmount",
    "reviewerBudgetCapAmount",
    "roleCollapseRule",
    "deterministicRefundRule",
    "unusedChallengeReserveRoute",
  ]) {
    exact(
      accountingBoundary[key],
      null,
      `$.rewardAccounting.accountingBoundary.${key}`,
    );
  }
  exact(
    accountingBoundary.futureReviewerBudgetMustBeOutcomeIndependent,
    true,
    "$.rewardAccounting.accountingBoundary.futureReviewerBudgetMustBeOutcomeIndependent",
  );
  if (!Array.isArray(source.breakthroughLens) || source.breakthroughLens.length !== 6) {
    fail("$.breakthroughLens", "must contain B0 through B5");
  }
  source.breakthroughLens.forEach((entry, index) => {
    const level = object(entry, `$.breakthroughLens[${index}]`);
    exactKeys(level, LENS_KEYS, `$.breakthroughLens[${index}]`);
    exact(level.level, `B${index}`, `$.breakthroughLens[${index}].level`);
    const expected = EXPECTED_LENS[index];
    if (!expected) fail("$.breakthroughLens", "unexpected lens level");
    exact(level.name, expected[0], `$.breakthroughLens[${index}].name`);
    exact(
      level.requires,
      expected[1],
      `$.breakthroughLens[${index}].requires`,
    );
    exact(level.assignable, false, `$.breakthroughLens[${index}].assignable`);
  });
  if (!Array.isArray(source.nodes) || source.nodes.length !== EXPECTED_NODE_IDS.length) {
    fail("$.nodes", "must contain the reviewed Season 1 graph");
  }
  const nodes = source.nodes.map(parseNode);
  if (JSON.stringify(nodes.map((node) => node.id)) !== JSON.stringify(EXPECTED_NODE_IDS)) {
    fail("$.nodes", "node IDs must be exact and sorted");
  }
  const byId = new Map(nodes.map((node) => [node.id, node]));
  nodes.forEach((node, index) => node.prerequisites.forEach((id, prerequisiteIndex) => {
    const prerequisite = byId.get(id);
    const baseMeta = BASE_NODE_META.get(id);
    if (!prerequisite && !baseMeta) fail(`$.nodes[${index}].prerequisites[${prerequisiteIndex}]`, "unknown node");
    const prerequisiteStage = prerequisite?.stage ?? baseMeta?.stage;
    if (prerequisiteStage && STAGE_RANK[prerequisiteStage] > STAGE_RANK[node.stage]) {
      fail(`$.nodes[${index}].prerequisites[${prerequisiteIndex}]`, `stage ${prerequisiteStage} cannot be a prerequisite of ${node.stage}`);
    }
  }));
  const edgeCount = nodes.reduce((total, node) => total + node.prerequisites.length, 0);
  exact(edgeCount, EXPECTED_EXTENSION_EDGE_COUNT, "$.nodes prerequisite edge count");
  const visiting = new Set<string>();
  const depths = new Map<string, number>();
  const depth = (id: string): number => {
    const baseDepth = BASE_NODE_META.get(id)?.depth;
    if (baseDepth !== undefined) return baseDepth;
    const cached = depths.get(id);
    if (cached !== undefined) return cached;
    if (visiting.has(id)) fail("$.nodes", "prerequisite graph contains a cycle");
    const node = byId.get(id);
    if (!node) fail("$.nodes", "prerequisite graph references an unknown node");
    visiting.add(id);
    const value = 1 + Math.max(...node.prerequisites.map(depth));
    visiting.delete(id);
    depths.set(id, value);
    return value;
  };
  const maximumDepth = Math.max(...nodes.map((node) => depth(node.id)));
  exact(maximumDepth, EXPECTED_COMBINED_MAX_DEPTH, "$.nodes combined graph depth");
  const standards = source.standards as QuantumSeason["standards"];
  if (!Array.isArray(standards) || standards.length !== 1) fail("$.standards", "one reproduction target is required");
  exactKeys(
    object(standards[0], "$.standards[0]"),
    STANDARD_KEYS,
    "$.standards[0]",
  );
  exact(
    standards[0]?.canonicalId,
    "doi:10.1038/s41467-026-70556-3",
    "$.standards[0].canonicalId",
  );
  exact(standards[0]?.authority, "Nature Communications", "$.standards[0].authority");
  exact(standards[0]?.title, "Decoding correlated errors in quantum LDPC codes", "$.standards[0].title");
  exact(standards[0]?.revision, "version of record 2026-05-01", "$.standards[0].revision");
  exact(standards[0]?.authorityStatus, "open-access version of record", "$.standards[0].authorityStatus");
  exact(standards[0]?.normalizedMaturity, "published", "$.standards[0].normalizedMaturity");
  exact(standards[0]?.specification, "https://doi.org/10.1038/s41467-026-70556-3", "$.standards[0].specification");
  exact(isoDate(standards[0]?.statusCheckedAt, "$.standards[0].statusCheckedAt"), "2026-08-01", "$.standards[0].statusCheckedAt");
  exact(isoDate(standards[0]?.reviewAfter, "$.standards[0].reviewAfter"), "2026-09-01", "$.standards[0].reviewAfter");
  exact(standards[0]?.treatment, "reproduction-target-not-truth-oracle", "$.standards[0].treatment");
  return {
    ...(source as unknown as QuantumSeason),
    nodes,
  };
}

export function quantumSeasonFreshness(
  season: QuantumSeason,
  asOf: string,
): QuantumSeasonFreshness {
  const normalizedAsOf = isoDate(asOf, "asOf");
  const reviewDates = season.standards.map((standard) => standard.reviewAfter);
  const earliestReviewAfter =
    reviewDates.length > 0 ? [...reviewDates].sort()[0] ?? null : null;
  const expiredStandardCount = season.standards.filter(
    (standard) => standard.reviewAfter < normalizedAsOf,
  ).length;
  return {
    earliestReviewAfter,
    expiredStandardCount,
    isExpiredForActiveUse: expiredStandardCount > 0,
  };
}

function rejectDuplicateJsonKeys(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    let escaped = false;
    while (offset < raw.length) {
      const character = raw[offset];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    fail("$", "unterminated JSON string");
  };
  const scanValue = (path: string, depth = 0): void => {
    if (depth > 64) fail(path, "JSON nesting exceeds 64");
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set<string>();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        if (raw[offset] !== '"') fail(path, "malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`, depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed object delimiter");
        offset += 1;
      }
      fail(path, "unterminated object");
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
        scanValue(`${path}[${index}]`, depth + 1);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) offset += 1;
  };
  scanValue("$");
}

export function parseQuantumSeasonJson(raw: string): QuantumSeason {
  if (new TextEncoder().encode(raw).byteLength > QUANTUM_SEASON_MAX_BYTES) {
    fail("$", `document exceeds ${QUANTUM_SEASON_MAX_BYTES} UTF-8 bytes`);
  }
  try {
    const value = JSON.parse(raw) as unknown;
    rejectDuplicateJsonKeys(raw);
    return parseQuantumSeason(value);
  } catch (error) {
    if (error instanceof QuantumSeasonDataError) throw error;
    fail("$", "malformed JSON");
  }
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new QuantumSeasonDataError(
      "Quantum extension digest verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const result = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(result)].map((value) => value.toString(16).padStart(2, "0")).join("");
}

async function readBounded(response: Response, signal: AbortSignal): Promise<Uint8Array> {
  const length = response.headers.get("content-length");
  if (length !== null) {
    const declared = Number(length);
    if (!Number.isSafeInteger(declared) || declared < 0) {
      throw new QuantumSeasonDataError(
        "Quantum extension declared an invalid byte length",
      );
    }
    if (declared > QUANTUM_SEASON_MAX_BYTES) {
      throw new QuantumSeasonDataError("Quantum extension exceeds its byte limit");
    }
  }
  if (!response.body) throw new QuantumSeasonDataError("Quantum extension returned an empty body");
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(
      signal.reason ?? new DOMException("Quantum extension request timed out", "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      size += value.byteLength;
      if (size > QUANTUM_SEASON_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal must not wait for a hostile stream to accept cancellation.
        });
        throw new QuantumSeasonDataError(
          "Quantum extension exceeds its byte limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // The deadline wins even if cancellation stalls or rejects.
      });
      throw new QuantumSeasonDataError("Quantum extension request timed out");
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A still-pending hostile read is abandoned after refusal.
    }
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  chunks.forEach((chunk) => {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  });
  return bytes;
}

export async function fetchQuantumSeason(
  options: QuantumSeasonFetchOptions = {},
): Promise<QuantumSeason> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (!fetcher) throw new QuantumSeasonDataError("Static quantum extension is unavailable");
  const baseUrl = options.baseUrl ?? globalThis.location?.origin ?? "https://zerone.ai";
  const endpoint = new URL(QUANTUM_SEASON_ENDPOINT, baseUrl);
  const expected = new URL(QUANTUM_SEASON_ENDPOINT, baseUrl);
  if (endpoint.origin !== expected.origin || endpoint.pathname !== expected.pathname || endpoint.search || endpoint.hash) {
    throw new QuantumSeasonDataError("Quantum extension endpoint is not the reviewed same-origin path");
  }
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new DOMException("Quantum extension request timed out", "TimeoutError")),
    options.timeoutMs ?? 5_000,
  );
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        controller.signal.reason ?? new DOMException("Quantum extension request timed out", "TimeoutError"),
      );
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(endpoint, {
          cache: "no-store",
          credentials: "same-origin",
          redirect: "error",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (
        controller.signal.aborted ||
        (error instanceof DOMException && (error.name === "AbortError" || error.name === "TimeoutError"))
      ) {
        throw new QuantumSeasonDataError("Quantum extension request timed out");
      }
      throw new QuantumSeasonDataError("Static quantum extension is unavailable");
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok || response.redirected) throw new QuantumSeasonDataError("Static quantum extension is unavailable");
    if (response.url) {
      const finalUrl = new URL(response.url, baseUrl);
      if (
        finalUrl.origin !== expected.origin ||
        finalUrl.pathname !== expected.pathname ||
        finalUrl.search ||
        finalUrl.hash
      ) {
        throw new QuantumSeasonDataError("Quantum extension response was not the reviewed same-origin path");
      }
    }
    if (!(response.headers.get("content-type") ?? "").toLowerCase().startsWith("application/json")) {
      throw new QuantumSeasonDataError("Quantum extension did not return JSON");
    }
    const bytes = await readBounded(response, controller.signal);
    if ((await sha256Hex(bytes)) !== QUANTUM_SEASON_SHA256) {
      throw new QuantumSeasonDataError("Quantum extension did not match the reviewed digest");
    }
    let raw: string;
    try { raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes); }
    catch { throw new QuantumSeasonDataError("Quantum extension was not valid UTF-8"); }
    return parseQuantumSeasonJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className?: string, copy?: string): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function humanise(value: string): string {
  return value.split("-").map((part) => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`).join(" ");
}

function percent(bps: number): string {
  return `${bps / 100}%`;
}

export function quantumRewardEligibilityLabel(
  value: QuantumRewardEligibility,
): string {
  return value === "qualification-only"
    ? "curriculum evidence only (no qualification)"
    : "future sponsor-case template (unfunded)";
}

export function quantumRewardMilestonesForDisplay(
  season: QuantumSeason,
): readonly QuantumMilestone[] {
  return season.rewardPolicy.milestones;
}

function renderInspector(
  root: HTMLElement,
  selected: QuantumNode,
  byId: Map<string, QuantumNode>,
  performance: QuantumSeason["performanceDecision"],
): void {
  root.replaceChildren();
  const heading = element("div", "quantum-inspector-head");
  heading.append(
    element("span", "quantum-node-code", selected.id),
    element("h3", undefined, selected.title),
    element("p", undefined, selected.summary),
  );
  root.append(heading);
  const badges = element("div", "quantum-node-badges");
  badges.append(
    element("span", undefined, selected.attainmentEvidence),
    element("span", undefined, quantumRewardEligibilityLabel(selected.rewardEligibility)),
    element("span", undefined, humanise(selected.defaultDisclosureLane)),
  );
  root.append(badges);
  const relationship = element("div", "quantum-inspector-block");
  relationship.append(element("strong", undefined, "Prerequisites"));
  const list = element("ul");
  selected.prerequisites.forEach((id) => {
    list.append(element("li", undefined, byId.get(id)?.title ?? `${id} · base v1`));
  });
  relationship.append(list);
  const evidence = element("div", "quantum-inspector-block");
  evidence.append(element("strong", undefined, "Inspectable artifacts"));
  const evidenceList = element("ul");
  selected.artifactRequirements.forEach((requirement) => evidenceList.append(element("li", undefined, requirement)));
  evidence.append(evidenceList);
  root.append(relationship, evidence);
  if (selected.acceptance) {
    const contract = selected.acceptance;
    const acceptance = element("div", "quantum-inspector-block quantum-acceptance");
    acceptance.append(
      element("strong", undefined, "Measurement-completeness floor · no performance pass"),
      element(
        "p",
        undefined,
        `${contract.minimumEffectiveClusters} effective clusters · ${contract.minimumOrganizationRoots} organisations · ${contract.minimumImplementationRoots} implementations · ${contract.minimumExecutionEnvironments} environments`,
      ),
      element(
        "p",
        undefined,
        `Exact BB fixtures ${contract.fixtures.map((fixture) => `[[${fixture.n},${fixture.k},${fixture.distance}]]`).join(" · ")} · p ∈ {${contract.physicalErrorGrid.join(", ")}}`,
      ),
      element(
        "p",
        undefined,
        `Circuit provenance: Stim circuits from Version of Record references 27 and 50 · distance-10/12 ensemble size ${contract.distance10And12CircuitEnsembleSize}`,
      ),
      element("p", undefined, `Cell: ${contract.cellDefinition}`),
    );
    const coverageHeading = element("strong", undefined, "Per-cell statistical gates");
    const coverage = element("ul", "quantum-coverage-list");
    contract.coverageTargets.forEach((target) => {
      const requirement = target.analysisMode === "bernoulli-logical-failure"
        ? `≥${target.minimumCasesPerCell.toLocaleString("en")} cases + ≥${target.minimumLogicalFailuresPerCell} logical failures · two-sided ${percent(target.confidenceLevelBps)} CI · ≤${percent(target.maximumRelativeHalfWidthBps)} relative half-width`
        : `≥${target.minimumCasesPerCell.toLocaleString("en")} cases · precommitted latency quantile/deadline · ${percent(target.confidenceLevelBps)} two-sided quantile CI or one-sided miss-rate upper bound · zero misses may satisfy measurement completeness, never a performance pass`;
      coverage.append(element("li", undefined, `${humanise(target.id)} — ${requirement}`));
    });
    acceptance.append(coverageHeading, coverage);

    const bindingsHeading = element("strong", undefined, "Content-bound circuits and artifacts");
    const bindings = element("ul", "quantum-binding-list");
    contract.requiredCaseBindings.forEach((binding) => bindings.append(element("li", undefined, binding)));
    acceptance.append(bindingsHeading, bindings);

    const prospectiveHeading = element(
      "strong",
      undefined,
      "Bindings still required before any funded performance rule",
    );
    const prospective = element("ul", "quantum-binding-list");
    performance.fundedCaseRequiredBindings.forEach((binding) => {
      prospective.append(element("li", undefined, binding));
    });
    acceptance.append(prospectiveHeading, prospective);

    const scopeHeading = element("strong", undefined, "Exact frozen scope");
    const scope = element("ul", "quantum-scope-list");
    contract.scopeBounds.forEach((bound) => scope.append(element("li", undefined, bound)));
    acceptance.append(
      scopeHeading,
      scope,
      element(
        "p",
        "quantum-no-pass",
        "V0 can check measurement completeness but cannot emit a performance pass. Precision alone establishes no performance result, and a missing prospective decision rule is INCONCLUSIVE_NO_PASS. A compute-cap exhaustion is also inconclusive. Rare-event substitution is limited to logical-error cells and only a separately reviewed unbiased estimator with method, review-receipt, variance, and coverage evidence is eligible.",
      ),
      element(
        "p",
        undefined,
        "Non-normative paper context only: logical error 6.70 ± 1.93e-9, mean latency 273 ns, and 99.99% < 1 μs. These observations are not thresholds and cannot make any decoder pass.",
      ),
    );
    root.append(acceptance);
  }
}

type QuantumCurrentControl = Pick<HTMLButtonElement, "setAttribute" | "removeAttribute">;

export function setQuantumCurrentNode(
  controls: ReadonlyMap<string, QuantumCurrentControl>,
  selectedId: string,
): void {
  controls.forEach((control, id) => {
    if (id === selectedId) control.setAttribute("aria-current", "true");
    else control.removeAttribute("aria-current");
  });
}

export function renderQuantumSeason(
  root: HTMLElement,
  season: QuantumSeason,
  asOf = new Date().toISOString().slice(0, 10),
): void {
  const freshness = quantumSeasonFreshness(season, asOf);
  const byId = new Map(season.nodes.map((node) => [node.id, node]));
  const shell = element("div", "quantum-season");
  const truth = element("div", "quantum-truth");
  truth.setAttribute("role", "note");
  truth.append(
    element("span", "quantum-kicker", "Season 1 · shadow mode"),
    element("strong", undefined, "Quantum capability is playable. Money stays at zero."),
    element(
      "p",
      undefined,
      "Thirteen hash-bound nodes, including one QEC decoder quest, are public. The template holds 0 uzrn, has no escrow receipt, creates no claim or qualification, and emits no performance pass. No direct governance grant. uzrn remains bondable under the current protocol, so an indirect stake-weight path exists.",
    ),
  );
  const facts = element("div", "quantum-facts");
  [
    ["Extension", `${season.nodes.length} nodes`],
    ["Quest", "Correlated-noise QEC"],
    ["Escrow", `${season.rewardPolicy.fundedAmount} ${season.rewardPolicy.denom}`],
    ["KARMA", season.karma.status],
  ].forEach(([label, value]) => {
    const fact = element("div");
    fact.append(element("span", undefined, label), element("strong", undefined, value));
    facts.append(fact);
  });
  shell.append(truth, facts);

  const lens = element("details", "quantum-lens");
  const lensSummary = element("summary");
  lensSummary.append(
    element("strong", undefined, "B0 → B5 breakthrough lens"),
    element("span", undefined, "Retrospective · never self-assigned"),
  );
  lens.append(lensSummary);
  const lensList = element("ol", "quantum-lens-list");
  season.breakthroughLens.forEach((level) => {
    const item = element("li");
    item.append(
      element("span", "quantum-level", level.level),
      element("strong", undefined, humanise(level.name)),
      element("small", undefined, level.requires),
    );
    lensList.append(item);
  });
  lens.append(lensList);
  shell.append(lens);

  const reward = element("div", "quantum-reward-shape");
  const release = element("div", "quantum-reward-axis");
  release.append(
    element("span", "quantum-kicker", "Outcome release axis · 100% shape · inactive"),
    element("h3", undefined, "Release ladder · 0 uzrn"),
  );
  const releaseList = element("ul");
  quantumRewardMilestonesForDisplay(season).forEach((milestone) => {
    const boundary = milestone.level === "E0"
      ? " · precedence only"
      : milestone.level === "E1"
        ? " · verified costs only, outside outcome %"
        : "";
    releaseList.append(element("li", undefined, `${milestone.level} ${humanise(milestone.name)} · ${percent(milestone.rewardBps)}${boundary}`));
  });
  releaseList.append(element("li", undefined, `Challenge + remediation · ${percent(season.rewardPolicy.challengeReserveBps)}`));
  release.append(
    releaseList,
    element(
      "p",
      undefined,
      "Verified costs need a separate prefunded cap; this template binds none.",
    ),
  );
  const attribution = element("div", "quantum-reward-axis");
  attribution.append(
    element("span", "quantum-kicker", "Attribution credit axis · 100% shape · inactive"),
    element("h3", undefined, "Whose work constructs it · 0 uzrn"),
  );
  const attributionList = element("ul");
  Object.entries(season.rewardPolicy.attributionBps).forEach(([name, bps]) => {
    const boundary = name === "independentReview"
      ? " · attribution credit, not adjudicator pay"
      : "";
    attributionList.append(element("li", undefined, `${humanise(name.replace(/([a-z])([A-Z])/g, "$1-$2").toLowerCase())} · ${percent(bps)}${boundary}`));
  });
  attribution.append(
    attributionList,
    element(
      "p",
      undefined,
      "Any future reviewer budget must be separate and outcome-independent.",
    ),
  );
  reward.append(release, attribution);
  shell.append(reward);

  const accounting = element("div", "quantum-karma");
  accounting.append(
    element("span", "quantum-kicker", "Reward accounting boundary"),
    element("h3", undefined, "Two orthogonal shapes, never 200%"),
    element(
      "p",
      undefined,
      "The two 100% axes are non-additive. Funding stays blocked: there is no cross-axis allocation, rounding rule, escrow-compartment binding, single settlement, verified-cost cap, reviewer-budget cap, or unused-reserve route.",
    ),
  );
  shell.append(accounting);

  const karma = element("div", "quantum-karma");
  karma.append(
    element("span", "quantum-kicker", "KARMA governance covenant"),
    element("h3", undefined, "Recognition without ownership"),
    element(
      "p",
      undefined,
      "Zerone records fallible, challengeable priced-coherence observations about domain relations—not human worth, ownership, or truth. Zerone does not mint KARMA; neither operator nor founder can assign it. Raw events and counts never qualify anyone or improve selection odds. Any future candidate filter must exclude same-controller, self, reciprocal, and correlated-funder edges; collapse controllers only downward; cap one lottery unit per controller; freeze candidates before unbiased randomness; and permit neither operator override nor count-proportional probability.",
    ),
  );
  shell.append(karma);

  const explorer = element("div", "quantum-explorer");
  const map = element("div", "quantum-map");
  const inspector = element("aside", "quantum-inspector");
  inspector.id = "quantum-season-inspector";
  inspector.setAttribute("aria-label", "Selected quantum capability evidence");
  inspector.setAttribute("aria-live", "polite");
  let selectedId = "quest-quantum-decoder-correlated-noise@1";
  const buttons = new Map<string, HTMLButtonElement>();
  (["foundation", "assurance", "quest"] as const).forEach((stage, stageIndex) => {
    const column = element("section", `quantum-stage quantum-stage-${stage}`);
    const heading = element("div", "quantum-stage-heading");
    heading.append(
      element("span", undefined, `${stageIndex + 1}`.padStart(2, "0")),
      element("h3", undefined, humanise(stage)),
    );
    column.append(heading);
    const list = element("ol");
    season.nodes.filter((node) => node.stage === stage).forEach((node) => {
      const item = element("li");
      const button = element("button", "quantum-node-button");
      button.type = "button";
      button.dataset.nodeId = node.id;
      button.setAttribute("aria-controls", inspector.id);
      button.append(
        element("strong", undefined, node.title),
        element("small", undefined, `${node.attainmentEvidence} · ${humanise(node.domain)}`),
      );
      button.addEventListener("click", () => {
        selectedId = node.id;
        setQuantumCurrentNode(buttons, selectedId);
        renderInspector(inspector, node, byId, season.performanceDecision);
        if (window.matchMedia("(max-width: 1040px)").matches) {
          const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
          inspector.scrollIntoView({ block: "start", behavior: reducedMotion ? "auto" : "smooth" });
        }
      });
      buttons.set(node.id, button);
      item.append(button);
      list.append(item);
    });
    column.append(list);
    map.append(column);
  });
  explorer.append(map, inspector);
  shell.append(explorer);

  const standard = season.standards[0];
  if (!standard) fail("$.standards", "one reproduction target is required");
  const freshnessStatus = element(
    "div",
    freshness.isExpiredForActiveUse
      ? "quantum-freshness quantum-freshness-stale"
      : "quantum-freshness",
  );
  freshnessStatus.setAttribute("role", freshness.isExpiredForActiveUse ? "alert" : "note");
  freshnessStatus.append(
    element(
      "strong",
      undefined,
      freshness.isExpiredForActiveUse
        ? "Source review overdue — active-use claims must pause."
        : "Source pin is within its review window.",
    ),
    element(
      "span",
      undefined,
      `Status checked ${standard.statusCheckedAt} · review after ${standard.reviewAfter} · rendered as of ${asOf}`,
    ),
  );
  shell.append(freshnessStatus);

  const source = element("div", "quantum-source");
  source.append(
    element("span", undefined, "Pinned reproduction target: "),
  );
  const link = element("a", undefined, standard.title);
  link.href = standard.specification;
  link.target = "_blank";
  link.rel = "noreferrer";
  const rawLink = element("a", undefined, "Raw reviewed quantum manifest");
  rawLink.href = QUANTUM_SEASON_ENDPOINT;
  source.append(
    link,
    document.createTextNode(` · ${standard.revision} · target, not truth oracle · `),
    rawLink,
  );
  shell.append(source);
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
  const selected = byId.get(selectedId);
  if (selected) {
    setQuantumCurrentNode(buttons, selectedId);
    renderInspector(inspector, selected, byId, season.performanceDecision);
  }
}

export async function initialiseQuantumSeason(
  root: HTMLElement,
  options: QuantumSeasonFetchOptions = {},
): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderQuantumSeason(root, await fetchQuantumSeason(options));
  } catch (error) {
    root.setAttribute("aria-busy", "false");
    const state = element("div", "quantum-load-error");
    state.setAttribute("role", "alert");
    state.append(
      element("strong", undefined, "The reviewed quantum extension could not be loaded."),
      element("p", undefined, error instanceof Error ? error.message : "The static document was unavailable or invalid."),
    );
    root.replaceChildren(state);
  }
}
