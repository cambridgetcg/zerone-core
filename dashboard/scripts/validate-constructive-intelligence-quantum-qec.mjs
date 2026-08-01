import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  parseAndValidateConstructiveIntelligenceTree,
} from "./validate-constructive-intelligence-tree.mjs";

export const QUANTUM_EXTENSION_SCHEMA =
  "zerone.constructive-intelligence-tree-extension/v0";
export const QUANTUM_EXTENSION_MAX_BYTES = 131_072;
export const QUANTUM_EXTENSION_MAX_DEPTH = 8;
export const QUANTUM_EXTENSION_MAX_FAN_OUT = 16;
export const REVIEWED_BASE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
export const REVIEWED_BASE_POLICY_SHA256 =
  "36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7";
export const REVIEWED_QUANTUM_NORMATIVE_SHA256 =
  "906f3256be011e5d56b2eb929b6d2963f4c4291ec3034cd902ae83499477a719";

const TOP_LEVEL_KEYS = [
  "schema",
  "seasonId",
  "title",
  "snapshotDate",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "base",
  "releaseBoundary",
  "karma",
  "breakthroughLens",
  "rewardPolicy",
  "standards",
  "nodes",
];
const BASE_KEYS = [
  "schema",
  "policyVersion",
  "endpoint",
  "documentSha256",
  "policySha256",
];
const RELEASE_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
];
const KARMA_KEYS = [
  "status",
  "register",
  "transferable",
  "scalarRank",
  "truthOracle",
  "payoutWeight",
  "voteWeight",
  "founderReservedPower",
  "futureUse",
  "activationRequires",
];
const LENS_KEYS = ["level", "name", "requires", "assignable"];
const REWARD_KEYS = [
  "status",
  "denom",
  "fundedAmount",
  "escrowReceipt",
  "claimable",
  "founderShareBps",
  "founderReservedSeats",
  "karmaWeightBps",
  "rewardCreatesGovernancePower",
  "skillUnlockCreatesReward",
  "timeAloneUnlocksEvidence",
  "milestones",
  "challengeReserveBps",
  "attributionBps",
];
const MILESTONE_KEYS = ["level", "name", "rewardBps", "treatment"];
const ATTRIBUTION_KEYS = [
  "originatingArtifact",
  "independentReplication",
  "independentReview",
  "downstreamAdoption",
  "falsificationAndChallenge",
  "safetyAndMaintenance",
];
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
];
const NODE_KEYS = [
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
];
const QUEST_NODE_KEYS = [...NODE_KEYS, "standardIds", "acceptance"];
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
];
const FIXTURE_KEYS = ["id", "n", "k", "distance"];
const RARE_EVENT_KEYS = [
  "appliesTo",
  "condition",
  "methodDigestRequired",
  "reviewReceiptDigestRequired",
  "unbiasedEstimatorRequired",
  "varianceAndCoverageValidationRequired",
];
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
];
const LOGICAL_COVERAGE_KEYS = [
  ...COVERAGE_COMMON_KEYS,
  "minimumLogicalFailuresPerCell",
  "maximumRelativeHalfWidthBps",
];
const LATENCY_COVERAGE_KEYS = [
  ...COVERAGE_COMMON_KEYS,
  "latencyEstimand",
  "zeroDeadlineMissesMayBeConclusive",
];
const EXPECTED_MILESTONES = [
  { level: "E0", name: "committed", rewardBps: 0, treatment: "precedence-only" },
  { level: "E1", name: "inspectable", rewardBps: 0, treatment: "verified-cost-only" },
  { level: "E2", name: "class-verified", rewardBps: 1500, treatment: "milestone" },
  { level: "E3", name: "independently-reproduced", rewardBps: 2000, treatment: "milestone" },
  { level: "E4", name: "adversarially-survived-or-fix-tested", rewardBps: 1500, treatment: "milestone" },
  { level: "E5", name: "independently-adopted", rewardBps: 2500, treatment: "milestone" },
  { level: "E6", name: "maintained", rewardBps: 1000, treatment: "milestone" },
];
const EXPECTED_ATTRIBUTION = {
  originatingArtifact: 3000,
  independentReplication: 2500,
  independentReview: 1000,
  downstreamAdoption: 2500,
  falsificationAndChallenge: 700,
  safetyAndMaintenance: 300,
};
const EXPECTED_LENS = [
  ["B0", "reproduction", "Known-result capability demonstration"],
  ["B1", "independent-replication", "REPLICATES plus E3"],
  ["B2", "bounded-extension", "Prior-art delta plus E3"],
  ["B3", "enabling-artifact", "Independent E5 adoption"],
  ["B4", "breakthrough", "Retrospective PROVES or DISPROVES, E3, prior-art delta, and independent adoption or descendant impact"],
  ["B5", "field-shift", "Multiple independent IMPLEMENTS, DEPLOYS, or MAINTAINS descendants through E5 or E6"],
];
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
];
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
  "latency=precommitted-quantile-and-deadline-at-99-percent-confidence-zero-misses-eligible",
  "minimum-cases=coverage-target-specific-per-cell",
  "noise-model=uniform-depolarizing-circuit-level",
  "physical-error-grid=0.001,0.002,0.003,0.004,0.005,0.006",
  "primary-metrics=logical-error-rate,latency",
  "random-seeds=committed-before-execution",
  "rare-event-alternative=separately-reviewed-unbiased-estimator-only",
  "stopping-rule=event-and-confidence-gates-before-compute-cap",
  "tuning-access-and-resource-budget=frozen-before-case-opens",
];
const EXPECTED_FIXTURES = [
  { id: "bb-72-12-6", n: 72, k: 12, distance: 6 },
  { id: "bb-90-8-10", n: 90, k: 8, distance: 10 },
  { id: "bb-144-12-12", n: 144, k: 12, distance: 12 },
];
const EXPECTED_PHYSICAL_ERROR_GRID = ["0.001", "0.002", "0.003", "0.004", "0.005", "0.006"];
const CELL_DEFINITION = "fixture-x-physical-error-x-implementation-root-x-execution-environment";
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
];
const NODE_ID = /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const DATE = /^\d{4}-\d{2}-\d{2}$/;
const SHA256 = /^[a-f0-9]{64}$/;

export class QuantumExtensionValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "QuantumExtensionValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new QuantumExtensionValidationError(path, message);
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value;
}

function exactKeys(value, expected, path) {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (JSON.stringify(actual) !== JSON.stringify(wanted)) {
    const unknown = actual.filter((key) => !wanted.includes(key));
    const missing = wanted.filter((key) => !actual.includes(key));
    fail(path, `field mismatch${unknown.length ? `; unknown: ${unknown.join(", ")}` : ""}${missing.length ? `; missing: ${missing.join(", ")}` : ""}`);
  }
}

function exact(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${JSON.stringify(expected)}`);
  return value;
}

function text(value, path, maximum = 8192) {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, `expected a non-empty string of at most ${maximum} characters`);
  }
  return value;
}

function integer(value, path, minimum = 0, maximum = Number.MAX_SAFE_INTEGER) {
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(path, `expected an integer in [${minimum}, ${maximum}]`);
  }
  return value;
}

function stringArray(value, path, { minimum = 0, maximum = 64, sorted = false } = {}) {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) {
    fail(path, `expected ${minimum} through ${maximum} strings`);
  }
  const result = value.map((item, index) => text(item, `${path}[${index}]`, 1024));
  if (new Set(result).size !== result.length) fail(path, "must not contain duplicates");
  if (sorted && result.some((item, index) => index > 0 && result[index - 1] > item)) {
    fail(path, "must be lexicographically sorted");
  }
  return result;
}

function digest(value) {
  return createHash("sha256").update(value).digest("hex");
}

function canonicalJson(value) {
  if (Array.isArray(value)) return value.map(canonicalJson);
  if (typeof value === "object" && value !== null) {
    return Object.fromEntries(
      Object.keys(value).sort().map((key) => [key, canonicalJson(value[key])]),
    );
  }
  return value;
}

function validateReviewedBase(base, baseRaw) {
  exactKeys(base, BASE_KEYS, "$.base");
  exact(base.schema, "zerone.constructive-intelligence-tree/v1", "$.base.schema");
  exact(base.policyVersion, "1.0.0", "$.base.policyVersion");
  exact(base.endpoint, "/standards/constructive-intelligence-tree.v1.json", "$.base.endpoint");
  exact(base.documentSha256, `sha256:${REVIEWED_BASE_SHA256}`, "$.base.documentSha256");
  exact(base.policySha256, `sha256:${REVIEWED_BASE_POLICY_SHA256}`, "$.base.policySha256");
  if (typeof baseRaw !== "string") fail("$.base", "reviewed base bytes are required");
  if (digest(baseRaw) !== REVIEWED_BASE_SHA256) fail("$.base.documentSha256", "reviewed base bytes do not match");
  parseAndValidateConstructiveIntelligenceTree(baseRaw);
  const parsed = JSON.parse(baseRaw);
  if (digest(JSON.stringify(canonicalJson(parsed.policy))) !== REVIEWED_BASE_POLICY_SHA256) {
    fail("$.base.policySha256", "reviewed base policy does not match");
  }
  return parsed;
}

function validateReleaseBoundary(boundary) {
  exactKeys(boundary, RELEASE_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_KEYS) exact(boundary[key], false, `$.releaseBoundary.${key}`);
}

function validateKarma(karma) {
  exactKeys(karma, KARMA_KEYS, "$.karma");
  exact(karma.status, "OBSERVATIONAL", "$.karma.status");
  exact(karma.register, "artifact-relation", "$.karma.register");
  for (const key of [
    "transferable",
    "scalarRank",
    "truthOracle",
    "payoutWeight",
    "voteWeight",
    "founderReservedPower",
  ]) exact(karma[key], false, `$.karma.${key}`);
  exact(karma.futureUse, "capped-randomized-eligibility-only", "$.karma.futureUse");
  const requirements = stringArray(karma.activationRequires, "$.karma.activationRequires", {
    minimum: 7,
    maximum: 7,
    sorted: true,
  });
  const required = [
    "appeal-and-reversible-pilot",
    "controller-clustering-and-pair-caps",
    "independent-validator-stake-host-and-upgrade-control",
    "minimum-organization-and-implementation-diversity",
    "prospective-named-upgrade-replay-tests-and-activation-height",
    "public-reasons-delay-and-challenge-window",
    "role-separation-and-recipient-conflict-exclusion",
  ];
  if (JSON.stringify(requirements) !== JSON.stringify(required)) {
    fail("$.karma.activationRequires", "must retain every ownerless-governance gate");
  }
}

function validateLens(lens) {
  if (!Array.isArray(lens) || lens.length !== EXPECTED_LENS.length) {
    fail("$.breakthroughLens", "must contain B0 through B5 exactly");
  }
  lens.forEach((entry, index) => {
    const item = record(entry, `$.breakthroughLens[${index}]`);
    exactKeys(item, LENS_KEYS, `$.breakthroughLens[${index}]`);
    exact(item.level, EXPECTED_LENS[index][0], `$.breakthroughLens[${index}].level`);
    exact(item.name, EXPECTED_LENS[index][1], `$.breakthroughLens[${index}].name`);
    exact(item.requires, EXPECTED_LENS[index][2], `$.breakthroughLens[${index}].requires`);
    exact(item.assignable, false, `$.breakthroughLens[${index}].assignable`);
  });
}

function validateRewardPolicy(reward) {
  exactKeys(reward, REWARD_KEYS, "$.rewardPolicy");
  exact(reward.status, "UNFUNDED_TEMPLATE", "$.rewardPolicy.status");
  exact(reward.denom, "uzrn", "$.rewardPolicy.denom");
  exact(reward.fundedAmount, "0", "$.rewardPolicy.fundedAmount");
  exact(reward.escrowReceipt, null, "$.rewardPolicy.escrowReceipt");
  for (const key of [
    "claimable",
    "rewardCreatesGovernancePower",
    "skillUnlockCreatesReward",
    "timeAloneUnlocksEvidence",
  ]) exact(reward[key], false, `$.rewardPolicy.${key}`);
  for (const key of ["founderShareBps", "founderReservedSeats", "karmaWeightBps"]) {
    exact(reward[key], 0, `$.rewardPolicy.${key}`);
  }
  if (!Array.isArray(reward.milestones) || reward.milestones.length !== EXPECTED_MILESTONES.length) {
    fail("$.rewardPolicy.milestones", "must inherit the exact E0-E6 ladder");
  }
  reward.milestones.forEach((entry, index) => {
    const item = record(entry, `$.rewardPolicy.milestones[${index}]`);
    exactKeys(item, MILESTONE_KEYS, `$.rewardPolicy.milestones[${index}]`);
    for (const key of MILESTONE_KEYS) {
      exact(item[key], EXPECTED_MILESTONES[index][key], `$.rewardPolicy.milestones[${index}].${key}`);
    }
  });
  exact(reward.challengeReserveBps, 1500, "$.rewardPolicy.challengeReserveBps");
  const released = reward.milestones.reduce((sum, item) => sum + item.rewardBps, 0);
  if (released + reward.challengeReserveBps !== 10_000) {
    fail("$.rewardPolicy", "milestone release plus challenge reserve must conserve 10,000 bps");
  }
  const attribution = record(reward.attributionBps, "$.rewardPolicy.attributionBps");
  exactKeys(attribution, ATTRIBUTION_KEYS, "$.rewardPolicy.attributionBps");
  for (const key of ATTRIBUTION_KEYS) {
    exact(attribution[key], EXPECTED_ATTRIBUTION[key], `$.rewardPolicy.attributionBps.${key}`);
  }
  if (Object.values(attribution).reduce((sum, value) => sum + value, 0) !== 10_000) {
    fail("$.rewardPolicy.attributionBps", "attribution must conserve 10,000 bps");
  }
}

function validateStandards(standards) {
  if (!Array.isArray(standards) || standards.length !== 1) {
    fail("$.standards", "must contain the one reviewed reproduction target");
  }
  const standard = record(standards[0], "$.standards[0]");
  exactKeys(standard, STANDARD_KEYS, "$.standards[0]");
  const expected = {
    canonicalId: "doi:10.1038/s41467-026-70556-3",
    authority: "Nature Communications",
    title: "Decoding correlated errors in quantum LDPC codes",
    revision: "version of record 2026-05-01",
    authorityStatus: "open-access version of record",
    normalizedMaturity: "published",
    specification: "https://doi.org/10.1038/s41467-026-70556-3",
    statusCheckedAt: "2026-08-01",
    reviewAfter: "2026-09-01",
    treatment: "reproduction-target-not-truth-oracle",
  };
  for (const key of STANDARD_KEYS) exact(standard[key], expected[key], `$.standards[0].${key}`);
}

function validateAcceptance(acceptance, path) {
  exactKeys(acceptance, ACCEPTANCE_KEYS, path);
  exact(acceptance.targetEvidence, "E5", `${path}.targetEvidence`);
  const scope = stringArray(acceptance.scopeBounds, `${path}.scopeBounds`, {
    minimum: EXPECTED_SCOPE.length,
    maximum: EXPECTED_SCOPE.length,
    sorted: true,
  });
  if (JSON.stringify(scope) !== JSON.stringify(EXPECTED_SCOPE)) fail(`${path}.scopeBounds`, "must keep the frozen QEC scope");
  const scopeHash = text(acceptance.scopeHash, `${path}.scopeHash`, 64);
  if (!SHA256.test(scopeHash) || scopeHash !== digest(JSON.stringify(scope))) {
    fail(`${path}.scopeHash`, "must commit to the exact scope bounds");
  }
  if (!Array.isArray(acceptance.fixtures) || acceptance.fixtures.length !== EXPECTED_FIXTURES.length) {
    fail(`${path}.fixtures`, "must contain the exact three reviewed BB fixtures");
  }
  acceptance.fixtures.forEach((entry, index) => {
    const fixturePath = `${path}.fixtures[${index}]`;
    const fixture = record(entry, fixturePath);
    exactKeys(fixture, FIXTURE_KEYS, fixturePath);
    for (const key of FIXTURE_KEYS) {
      exact(fixture[key], EXPECTED_FIXTURES[index][key], `${fixturePath}.${key}`);
    }
  });
  const physicalErrorGrid = stringArray(acceptance.physicalErrorGrid, `${path}.physicalErrorGrid`, {
    minimum: EXPECTED_PHYSICAL_ERROR_GRID.length,
    maximum: EXPECTED_PHYSICAL_ERROR_GRID.length,
    sorted: true,
  });
  if (JSON.stringify(physicalErrorGrid) !== JSON.stringify(EXPECTED_PHYSICAL_ERROR_GRID)) {
    fail(`${path}.physicalErrorGrid`, "must bind p=0.001 through 0.006 in increments of 0.001");
  }
  exact(acceptance.cellDefinition, CELL_DEFINITION, `${path}.cellDefinition`);
  const caseBindings = stringArray(acceptance.requiredCaseBindings, `${path}.requiredCaseBindings`, {
    minimum: EXPECTED_CASE_BINDINGS.length,
    maximum: EXPECTED_CASE_BINDINGS.length,
    sorted: true,
  });
  if (JSON.stringify(caseBindings) !== JSON.stringify(EXPECTED_CASE_BINDINGS)) {
    fail(`${path}.requiredCaseBindings`, "must content-bind every circuit and acceptance artifact");
  }
  exact(acceptance.circuitProvenance, "stim-circuits-from-version-of-record-references-27-and-50", `${path}.circuitProvenance`);
  exact(acceptance.distance10And12CircuitEnsembleSize, 24, `${path}.distance10And12CircuitEnsembleSize`);
  exact(acceptance.computeCapExhaustion, "inconclusive-no-pass", `${path}.computeCapExhaustion`);
  const rareEvent = record(acceptance.rareEventAlternative, `${path}.rareEventAlternative`);
  exactKeys(rareEvent, RARE_EVENT_KEYS, `${path}.rareEventAlternative`);
  exact(rareEvent.appliesTo, "logical-error-cells-only", `${path}.rareEventAlternative.appliesTo`);
  exact(rareEvent.condition, "separately-reviewed-unbiased-estimator-only", `${path}.rareEventAlternative.condition`);
  for (const key of RARE_EVENT_KEYS.slice(2)) {
    exact(rareEvent[key], true, `${path}.rareEventAlternative.${key}`);
  }
  if (!Array.isArray(acceptance.coverageTargets) || acceptance.coverageTargets.length !== 3) {
    fail(`${path}.coverageTargets`, "must contain all three coverage targets");
  }
  const expectedCases = [100_000, 1_000_000, 100_000];
  const expectedIds = ["baseline-regression", "correlated-noise-logical-error-rate", "matched-resource-latency"];
  acceptance.coverageTargets.forEach((entry, index) => {
    const targetPath = `${path}.coverageTargets[${index}]`;
    const target = record(entry, targetPath);
    const logical = index < 2;
    exactKeys(target, logical ? LOGICAL_COVERAGE_KEYS : LATENCY_COVERAGE_KEYS, targetPath);
    exact(target.id, expectedIds[index], `${targetPath}.id`);
    exact(target.cellDefinition, CELL_DEFINITION, `${targetPath}.cellDefinition`);
    exact(target.analysisMode, logical ? "bernoulli-logical-failure" : "latency-quantile-and-deadline", `${targetPath}.analysisMode`);
    for (const key of [
      "minimumEffectiveClusters",
      "minimumOrganizationRoots",
      "minimumImplementationRoots",
      "minimumExecutionEnvironments",
    ]) {
      const floor = key === "minimumEffectiveClusters" ? 3 : 2;
      exact(target[key], floor, `${targetPath}.${key}`);
    }
    exact(target.minimumCasesPerCell, expectedCases[index], `${targetPath}.minimumCasesPerCell`);
    exact(target.confidenceLevelBps, 9900, `${targetPath}.confidenceLevelBps`);
    if (logical) {
      exact(target.minimumLogicalFailuresPerCell, 100, `${targetPath}.minimumLogicalFailuresPerCell`);
      exact(target.confidenceProcedure, "two-sided-binomial-interval", `${targetPath}.confidenceProcedure`);
      exact(target.maximumRelativeHalfWidthBps, 3000, `${targetPath}.maximumRelativeHalfWidthBps`);
    } else {
      exact(target.latencyEstimand, "precommitted-quantile-and-deadline", `${targetPath}.latencyEstimand`);
      exact(target.confidenceProcedure, "two-sided-quantile-interval-or-one-sided-deadline-miss-upper-bound", `${targetPath}.confidenceProcedure`);
      exact(target.zeroDeadlineMissesMayBeConclusive, true, `${targetPath}.zeroDeadlineMissesMayBeConclusive`);
    }
    for (const key of [
      "requiresCheckerOrCorpusDigest",
      "requiresCircuitAndArtifactDigests",
      "requiresConfidenceProcedureDigest",
    ]) {
      exact(target[key], true, `${targetPath}.${key}`);
    }
  });
  exact(acceptance.minimumEffectiveClusters, 3, `${path}.minimumEffectiveClusters`);
  exact(acceptance.minimumOrganizationRoots, 2, `${path}.minimumOrganizationRoots`);
  exact(acceptance.minimumImplementationRoots, 2, `${path}.minimumImplementationRoots`);
  exact(acceptance.minimumExecutionEnvironments, 2, `${path}.minimumExecutionEnvironments`);
  const receipts = stringArray(acceptance.adoptionReceiptTypes, `${path}.adoptionReceiptTypes`, { minimum: 2, maximum: 2, sorted: true });
  if (JSON.stringify(receipts) !== JSON.stringify(["maintained-fixture", "upstream-merge"])) {
    fail(`${path}.adoptionReceiptTypes`, "must require independent maintained use");
  }
  exact(acceptance.privateEscalationRequired, true, `${path}.privateEscalationRequired`);
  exact(acceptance.prepublicationTriageRequired, true, `${path}.prepublicationTriageRequired`);
}

function validateNodes(nodes, baseTree) {
  if (!Array.isArray(nodes) || nodes.length !== EXPECTED_NODE_IDS.length) {
    fail("$.nodes", `must contain the ${EXPECTED_NODE_IDS.length} reviewed Season 1 nodes`);
  }
  const ids = nodes.map((node, index) => text(record(node, `$.nodes[${index}]`).id, `$.nodes[${index}].id`, 128));
  if (JSON.stringify(ids) !== JSON.stringify(EXPECTED_NODE_IDS)) fail("$.nodes", "node IDs must be exact, unique, and sorted");
  const baseIds = new Set(baseTree.nodes.map((node) => node.id));
  const extensionIds = new Set(ids);
  const stageRank = new Map([["foundation", 0], ["primitive", 1], ["assurance", 2], ["protocol", 3], ["quest", 4]]);
  const all = new Map(baseTree.nodes.map((node) => [node.id, node]));
  nodes.forEach((node) => all.set(node.id, node));

  nodes.forEach((entry, index) => {
    const path = `$.nodes[${index}]`;
    const node = record(entry, path);
    const quest = node.stage === "quest";
    exactKeys(node, quest ? QUEST_NODE_KEYS : NODE_KEYS, path);
    if (!NODE_ID.test(node.id)) fail(`${path}.id`, "must be a versioned kebab-case node ID");
    text(node.title, `${path}.title`, 160);
    text(node.summary, `${path}.summary`, 1024);
    if (!stageRank.has(node.stage)) fail(`${path}.stage`, "unsupported extension stage");
    const expectedDomain = quest ? "quests" : node.stage;
    const domain = node.stage === "foundation" ? "mathematics" : expectedDomain;
    exact(node.domain, domain, `${path}.domain`);
    const prerequisites = stringArray(node.prerequisites, `${path}.prerequisites`, { minimum: 1, maximum: 8, sorted: true });
    prerequisites.forEach((id, prerequisiteIndex) => {
      if (!baseIds.has(id) && !extensionIds.has(id)) fail(`${path}.prerequisites[${prerequisiteIndex}]`, "references an unknown base or extension node");
      if (id === node.id) fail(`${path}.prerequisites[${prerequisiteIndex}]`, "cannot reference itself");
      const prerequisite = all.get(id);
      const prerequisiteRank = stageRank.get(prerequisite?.stage);
      if (prerequisiteRank === undefined) fail(`${path}.prerequisites[${prerequisiteIndex}]`, "prerequisite has an unsupported stage");
      if (prerequisiteRank > stageRank.get(node.stage)) {
        fail(`${path}.prerequisites[${prerequisiteIndex}]`, `stage ${prerequisite.stage} cannot be a prerequisite of ${node.stage}`);
      }
    });
    exact(node.attainmentEvidence, quest ? "E5" : node.stage === "assurance" ? "E3" : "E2", `${path}.attainmentEvidence`);
    exact(node.rewardEligibility, quest ? "sponsor-milestones" : "qualification-only", `${path}.rewardEligibility`);
    if (!["open-construction", "controlled-operations"].includes(node.defaultDisclosureLane)) fail(`${path}.defaultDisclosureLane`, "unsupported disclosure lane");
    stringArray(node.artifactRequirements, `${path}.artifactRequirements`, { minimum: 2, maximum: 8 });
    stringArray(node.revalidationTriggers, `${path}.revalidationTriggers`, { minimum: 1, maximum: 4 });
    if (quest) {
      const standards = stringArray(node.standardIds, `${path}.standardIds`, { minimum: 1, maximum: 1 });
      exact(standards[0], "doi:10.1038/s41467-026-70556-3", `${path}.standardIds[0]`);
      validateAcceptance(record(node.acceptance, `${path}.acceptance`), `${path}.acceptance`);
    }
  });

  const visiting = new Set();
  const depths = new Map();
  const depth = (id) => {
    if (depths.has(id)) return depths.get(id);
    if (visiting.has(id)) fail("$.nodes", "prerequisite graph contains a cycle");
    visiting.add(id);
    const node = all.get(id);
    const value = node.prerequisites.length === 0 ? 1 : 1 + Math.max(...node.prerequisites.map(depth));
    visiting.delete(id);
    depths.set(id, value);
    return value;
  };
  const maxDepth = Math.max(...nodes.map((node) => depth(node.id)));
  if (maxDepth > QUANTUM_EXTENSION_MAX_DEPTH) fail("$.nodes", `combined graph depth ${maxDepth} exceeds ${QUANTUM_EXTENSION_MAX_DEPTH}`);
  const fanOut = new Map([...all.keys()].map((id) => [id, 0]));
  all.forEach((node) => node.prerequisites.forEach((id) => fanOut.set(id, (fanOut.get(id) ?? 0) + 1)));
  const maxFanOut = Math.max(...fanOut.values());
  if (maxFanOut > QUANTUM_EXTENSION_MAX_FAN_OUT) fail("$.nodes", `combined graph fan-out ${maxFanOut} exceeds ${QUANTUM_EXTENSION_MAX_FAN_OUT}`);
  return {
    nodeCount: nodes.length,
    edgeCount: nodes.reduce((sum, node) => sum + node.prerequisites.length, 0),
    maxDepth,
    maxFanOut,
    questCount: nodes.filter((node) => node.stage === "quest").length,
  };
}

export function validateQuantumExtension(value, baseRaw, { enforceReviewedDigest = true } = {}) {
  const extension = record(value, "$");
  exactKeys(extension, TOP_LEVEL_KEYS, "$");
  exact(extension.schema, QUANTUM_EXTENSION_SCHEMA, "$.schema");
  exact(extension.seasonId, "quantum-qec-2026q3", "$.seasonId");
  text(extension.title, "$.title", 160);
  if (!DATE.test(extension.snapshotDate)) fail("$.snapshotDate", "must be an ISO date");
  exact(extension.snapshotDate, "2026-08-01", "$.snapshotDate");
  for (const key of ["authoritative", "networkObserved", "rewardBearing"]) exact(extension[key], false, `$.${key}`);
  const baseTree = validateReviewedBase(record(extension.base, "$.base"), baseRaw);
  validateReleaseBoundary(record(extension.releaseBoundary, "$.releaseBoundary"));
  validateKarma(record(extension.karma, "$.karma"));
  validateLens(extension.breakthroughLens);
  validateRewardPolicy(record(extension.rewardPolicy, "$.rewardPolicy"));
  validateStandards(extension.standards);
  const graph = validateNodes(extension.nodes, baseTree);
  const normativeDigest = digest(JSON.stringify(canonicalJson(extension)));
  if (enforceReviewedDigest && normativeDigest !== REVIEWED_QUANTUM_NORMATIVE_SHA256) {
    fail("$", `reviewed normative digest mismatch: ${normativeDigest}`);
  }
  return { ...graph, normativeDigest };
}

function rejectDuplicateKeys(raw) {
  let offset = 0;
  const whitespace = () => { while (/\s/.test(raw[offset] ?? "")) offset += 1; };
  const scanString = () => {
    const start = offset;
    offset += 1;
    let escaped = false;
    while (offset < raw.length) {
      const character = raw[offset];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset));
      }
      offset += 1;
    }
    fail("$", "unterminated JSON string");
  };
  const scanValue = (path, depth = 0) => {
    if (depth > 64) fail(path, "JSON nesting exceeds 64");
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set();
      if (raw[offset] === "}") { offset += 1; return; }
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
        if (raw[offset] === "}") { offset += 1; return; }
        if (raw[offset] !== ",") fail(path, "malformed object delimiter");
        offset += 1;
      }
      fail(path, "unterminated object");
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") { offset += 1; return; }
      let index = 0;
      while (offset < raw.length) {
        scanValue(`${path}[${index}]`, depth + 1);
        whitespace();
        if (raw[offset] === "]") { offset += 1; return; }
        if (raw[offset] !== ",") fail(path, "malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "unterminated array");
    }
    if (token === '"') { scanString(); return; }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset])) offset += 1;
  };
  scanValue("$");
}

export function parseAndValidateQuantumExtension(raw, baseRaw, options) {
  if (typeof raw !== "string") fail("$", "raw extension must be a string");
  if (Buffer.byteLength(raw, "utf8") > QUANTUM_EXTENSION_MAX_BYTES) fail("$", `document exceeds ${QUANTUM_EXTENSION_MAX_BYTES} UTF-8 bytes`);
  let value;
  try { value = JSON.parse(raw); }
  catch (error) { fail("$", `invalid JSON: ${error.message}`); }
  rejectDuplicateKeys(raw);
  return validateQuantumExtension(value, baseRaw, options);
}

function runCli() {
  if (process.argv.length !== 4) {
    console.error("usage: node scripts/validate-constructive-intelligence-quantum-qec.mjs EXTENSION BASE_TREE");
    process.exitCode = 2;
    return;
  }
  try {
    const extensionRaw = readFileSync(resolve(process.argv[2]), "utf8");
    const baseRaw = readFileSync(resolve(process.argv[3]), "utf8");
    const result = parseAndValidateQuantumExtension(extensionRaw, baseRaw);
    console.log(`quantum QEC extension: PASS (${result.nodeCount} extension nodes, ${result.edgeCount} extension edges, combined depth ${result.maxDepth}, ${result.questCount} quest)`);
  } catch (error) {
    console.error(`quantum QEC extension: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) runCli();
