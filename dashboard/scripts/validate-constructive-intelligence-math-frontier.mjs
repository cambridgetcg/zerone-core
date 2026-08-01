import { createHash } from "node:crypto";
import { lstatSync, readFileSync, realpathSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  parseAndValidateConstructiveIntelligenceTree,
} from "./validate-constructive-intelligence-tree.mjs";

export const MATH_FRONTIER_SCHEMA =
  "zerone.constructive-intelligence-math-frontier/v0";
export const MATH_PROBLEM_SCHEMA =
  "zerone.constructive-intelligence-math-problem/v0";
export const MATH_FRONTIER_MAX_BYTES = 262_144;
export const MATH_PROBLEM_MAX_BYTES = 65_536;

const REVIEWED_FRONTIER_BYTE_SHA256 =
  "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456";
const REVIEWED_FRONTIER_CANONICAL_SHA256 =
  "b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503";
const REVIEWED_QUEST_TEMPLATE_SHA256 =
  "7af1dc87b98f6b80bb798aa1427c82c5b2c049f8cdccd9b914188cba50718313";
const REVIEWED_PROBLEM_PACKET_SHA256 =
  "f0961813b83cbd9f127290c19cf5bc98c07cad2dc1158bbab26243edd7af9ae3";
const REVIEWED_BASE_TREE_BYTE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
const REVIEWED_BASE_POLICY_SHA256 =
  "36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7";

const TOP_LEVEL_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "governanceBearing",
  "snapshotDate",
  "policyVersion",
  "baseTree",
  "releaseBoundary",
  "constitution",
  "karma",
  "rewardTemplate",
  "stages",
  "nodes",
  "questTemplate",
];
const BASE_TREE_KEYS = [
  "schema",
  "policyVersion",
  "documentSha256",
  "policySha256",
  "requiredCapabilities",
];
const RELEASE_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsGovernancePower",
  "ranksPersons",
  "authorizesSecurityTesting",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
];
const CONSTITUTION_KEYS = [
  "identityLabelInvariant",
  "reservedSeatsAllowed",
  "reservedSharesAllowed",
  "creatorPrivilege",
  "operatorPrivilege",
  "humanAiClassMultiplier",
  "stakeAffectsValidity",
  "stakeAffectsVoice",
  "wealthAffectsEligibility",
  "wealthAffectsReward",
  "rewardBalanceAffectsVoice",
  "karmaMagnitudeAffectsVoice",
  "karmaEligibilityOnly",
  "controllerMergeCanOnlyReduceVoice",
  "controllerConflictRecusalRequired",
  "recusalQuorumRecomputed",
  "selectedControllerVoice",
  "currentActivationAuthority",
  "futureGovernance",
  "ordinaryStakeVoteCanActivate",
  "emergencyAuthorityCanActivate",
  "emergencyAuthorityScope",
];
const KARMA_KEYS = [
  "mode",
  "eventSchema",
  "state",
  "register",
  "truthOracle",
  "onChainRecognition",
  "magnitude",
  "transferable",
  "spendable",
  "rewardMultiplier",
  "governanceWeight",
  "economicEffect",
  "controlEffect",
  "futureRecognitionRequiresSignedReceipts",
  "futureRecognitionRequiresControllerResolution",
  "excludedFromRecognition",
];
const REWARD_KEYS = [
  "mode",
  "economicEffect",
  "liveAmount",
  "skillUnlockCreatesEntitlement",
  "breakthroughCreatesEntitlement",
  "protocolIssuanceAllowed",
  "dedicatedEscrowRequired",
  "policyFrozenBeforeAdmission",
  "singleSettlementRequired",
  "verifiedCostsTreatment",
  "milestones",
  "challengeReserveBps",
  "roles",
  "disproofDisposition",
  "unallocatedDisposition",
  "openLiabilityCap",
  "specializedResearchSpendPathAllowed",
];
const MILESTONE_KEYS = ["level", "name", "outcomePoolBps", "evidence"];
const NODE_KEYS = [
  "id",
  "title",
  "stage",
  "summary",
  "prerequisites",
  "evidenceTarget",
  "capabilityClass",
  "artifacts",
  "unlocksReward",
  "questEligible",
];
const QUEST_KEYS = [
  "id",
  "status",
  "requiredCoreCapabilities",
  "allowedDomainCapabilities",
  "domainSelectionMode",
  "selectedDomainEvidenceMinimum",
  "selectedDomainReceiptRequired",
  "allowedArtifactRelations",
  "requiredPacketBindings",
  "milestoneOrder",
  "breakthroughStatus",
  "selfDeclaredBreakthrough",
  "popularityCanOverrideValidity",
  "relationSpecificValidityRequired",
  "implementsEstablishesTheoremValidity",
  "semanticRootReplayAllowed",
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "minimumKernelFamilies",
  "counterexampleEligible",
  "sunsetRequired",
  "economicEffect",
  "controlEffect",
];
const PROBLEM_KEYS = [
  "schema",
  "synthetic",
  "frontierDocumentSha256",
  "questTemplateSha256",
  "problemId",
  "domainCapabilityId",
  "selectedDomainEvidence",
  "selectedDomainReceiptDigest",
  "artifactRelation",
  "statementDigest",
  "formalizationDigest",
  "axiomPolicyDigest",
  "semanticRoot",
  "priorArtCutoff",
  "priorArtManifestDigest",
  "checkerKernelDigests",
  "falsifierSuiteDigest",
  "breakthroughStatus",
  "karmaProjection",
  "economicEffect",
  "controlEffect",
  "qualification",
  "liveAmount",
];
const PROBLEM_KARMA_KEYS = ["schema", "state", "register", "magnitude"];

const STAGES = ["ground", "craft", "assurance", "frontier"];
const STAGE_RANK = new Map(STAGES.map((stage, index) => [stage, index]));
const CAPABILITY_CLASSES = new Set([
  "foundation",
  "domain",
  "method",
  "assurance",
  "quest",
]);
const EVIDENCE_LEVELS = new Set(["E1", "E2", "E3", "E5"]);
const NODE_ID_PATTERN =
  /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const PROBLEM_ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const ISO_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const SHA256_PATTERN = /^sha256:[0-9a-f]{64}$/;

const EXPECTED_MILESTONES = [
  ["E2", "deterministic-validity", 1500],
  ["E3", "independent-reproduction", 2000],
  ["E4", "adversarial-survival", 1500],
  ["E5", "independent-downstream-use", 2500],
  ["E6", "maintained-recheck", 1000],
];

export class MathFrontierValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "MathFrontierValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new MathFrontierValidationError(path, message);
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value;
}

function exactKeys(value, allowed, path) {
  const allowedSet = new Set(allowed);
  for (const key of Object.keys(value)) {
    if (!allowedSet.has(key)) fail(`${path}.${key}`, "is not part of schema v0");
  }
  for (const key of allowed) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is required");
  }
}

function exact(value, expected, path) {
  if (value !== expected) fail(path, `must be ${JSON.stringify(expected)}`);
}

function boundedString(value, path, maximum = 2048) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maximum) {
    fail(path, `must be at most ${maximum} UTF-8 bytes`);
  }
  return value;
}

function positiveInteger(value, path, maximum = 1_000_000) {
  if (!Number.isSafeInteger(value) || value <= 0 || value > maximum) {
    fail(path, `must be an integer in [1, ${maximum}]`);
  }
  return value;
}

function digest(value, path) {
  if (typeof value !== "string" || !SHA256_PATTERN.test(value)) {
    fail(path, "must be a lowercase sha256 digest");
  }
  return value;
}

function sortedStrings(value, path, { minimum = 0, maximum = 64 } = {}) {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) {
    fail(path, `must contain between ${minimum} and ${maximum} strings`);
  }
  const parsed = value.map((item, index) =>
    boundedString(item, `${path}[${index}]`),
  );
  if (new Set(parsed).size !== parsed.length) fail(path, "must not contain duplicates");
  const sorted = [...parsed].sort();
  if (parsed.some((item, index) => item !== sorted[index])) {
    fail(path, "must be sorted");
  }
  return parsed;
}

function stringList(value, path, { minimum = 0, maximum = 64 } = {}) {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) {
    fail(path, `must contain between ${minimum} and ${maximum} strings`);
  }
  const parsed = value.map((item, index) =>
    boundedString(item, `${path}[${index}]`),
  );
  if (new Set(parsed).size !== parsed.length) fail(path, "must not contain duplicates");
  return parsed;
}

export function canonicalJson(value) {
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalJson(item)).join(",")}]`;
  }
  if (typeof value === "object" && value !== null) {
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${canonicalJson(value[key])}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}

function sha256(value) {
  return createHash("sha256").update(value).digest("hex");
}

function validateExactRecord(value, expected, keys, path) {
  const parsed = record(value, path);
  exactKeys(parsed, keys, path);
  for (const [key, expectedValue] of Object.entries(expected)) {
    exact(parsed[key], expectedValue, `${path}.${key}`);
  }
  return parsed;
}

function validateBaseBinding(value, baseTreeRaw) {
  const base = record(value, "$.baseTree");
  exactKeys(base, BASE_TREE_KEYS, "$.baseTree");
  exact(base.schema, "zerone.constructive-intelligence-tree/v1", "$.baseTree.schema");
  exact(base.policyVersion, "1.0.0", "$.baseTree.policyVersion");
  exact(
    base.documentSha256,
    `sha256:${REVIEWED_BASE_TREE_BYTE_SHA256}`,
    "$.baseTree.documentSha256",
  );
  exact(
    base.policySha256,
    `sha256:${REVIEWED_BASE_POLICY_SHA256}`,
    "$.baseTree.policySha256",
  );
  const required = sortedStrings(base.requiredCapabilities, "$.baseTree.requiredCapabilities", {
    minimum: 1,
    maximum: 16,
  });
  if (typeof baseTreeRaw !== "string") {
    fail("$.baseTree", "exact base-tree bytes are required for validation");
  }
  try {
    parseAndValidateConstructiveIntelligenceTree(baseTreeRaw);
  } catch (error) {
    fail(
      "$.baseTree",
      `base tree is invalid: ${error instanceof Error ? error.message : String(error)}`,
    );
  }
  exact(
    sha256(baseTreeRaw),
    REVIEWED_BASE_TREE_BYTE_SHA256,
    "$.baseTree.documentSha256",
  );
  const tree = JSON.parse(baseTreeRaw);
  exact(
    sha256(canonicalJson(tree.policy)),
    REVIEWED_BASE_POLICY_SHA256,
    "$.baseTree.policySha256",
  );
  const byId = new Map(tree.nodes.map((node) => [node.id, node]));
  for (const [index, id] of required.entries()) {
    const node = byId.get(id);
    if (!node) fail(`$.baseTree.requiredCapabilities[${index}]`, "is absent from the base tree");
    if (node.domain !== "mathematics") {
      fail(`$.baseTree.requiredCapabilities[${index}]`, "must import a mathematics capability");
    }
    if (node.rewardEligibility !== "qualification-only") {
      fail(`$.baseTree.requiredCapabilities[${index}]`, "must remain qualification-only");
    }
  }
  return new Set(required);
}

function validateReleaseBoundary(value) {
  const boundary = record(value, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_KEYS) exact(boundary[key], false, `$.releaseBoundary.${key}`);
}

function validateConstitution(value) {
  const expected = {
    identityLabelInvariant: true,
    reservedSeatsAllowed: false,
    reservedSharesAllowed: false,
    creatorPrivilege: false,
    operatorPrivilege: false,
    humanAiClassMultiplier: false,
    stakeAffectsValidity: false,
    stakeAffectsVoice: false,
    wealthAffectsEligibility: false,
    wealthAffectsReward: false,
    rewardBalanceAffectsVoice: false,
    karmaMagnitudeAffectsVoice: false,
    karmaEligibilityOnly: true,
    controllerMergeCanOnlyReduceVoice: true,
    controllerConflictRecusalRequired: true,
    recusalQuorumRecomputed: true,
    selectedControllerVoice: "ONE_CONTROLLER_ONE_VOICE",
    currentActivationAuthority: "NONE",
    futureGovernance: "KARMA_ELIGIBILITY_AND_SORTITION_NOT_IMPLEMENTED",
    ordinaryStakeVoteCanActivate: false,
    emergencyAuthorityCanActivate: false,
    emergencyAuthorityScope: "PAUSE_ONLY",
  };
  validateExactRecord(value, expected, CONSTITUTION_KEYS, "$.constitution");
}

function validateKarma(value) {
  const karma = validateExactRecord(
    value,
    {
      mode: "ORDINAL_SHADOW_ONLY",
      eventSchema: "zerone.karma.shadow-edge/v0",
      state: "ORDINAL",
      register: "priced-coherence",
      truthOracle: false,
      onChainRecognition: false,
      magnitude: "NONE",
      transferable: false,
      spendable: false,
      rewardMultiplier: false,
      governanceWeight: false,
      economicEffect: "NONE",
      controlEffect: "NONE",
      futureRecognitionRequiresSignedReceipts: true,
      futureRecognitionRequiresControllerResolution: true,
    },
    KARMA_KEYS,
    "$.karma",
  );
  const excluded = sortedStrings(
    karma.excludedFromRecognition,
    "$.karma.excludedFromRecognition",
    { minimum: 4, maximum: 4 },
  );
  exact(
    JSON.stringify(excluded),
    JSON.stringify([
      "external-unverified",
      "reciprocal-cycle",
      "same-controller",
      "self",
    ]),
    "$.karma.excludedFromRecognition",
  );
}

function validateRewardTemplate(value) {
  const reward = record(value, "$.rewardTemplate");
  exactKeys(reward, REWARD_KEYS, "$.rewardTemplate");
  const expected = {
    mode: "PROSPECTIVE_SPONSOR_ESCROW_ONLY",
    economicEffect: "NONE",
    liveAmount: "0",
    skillUnlockCreatesEntitlement: false,
    breakthroughCreatesEntitlement: false,
    protocolIssuanceAllowed: false,
    dedicatedEscrowRequired: true,
    policyFrozenBeforeAdmission: true,
    singleSettlementRequired: true,
    verifiedCostsTreatment: "SEPARATE_PREAUTHORIZED_CAP",
    challengeReserveBps: 1500,
    unallocatedDisposition: "PROSPECTIVELY_NAMED_REFUND_OR_COMMONS",
    openLiabilityCap: "DEDICATED_ESCROW",
    specializedResearchSpendPathAllowed: false,
    disproofDisposition: "E4_TO_COMPLIANT_FALSIFIER_CLAIMANT_UNPAID",
  };
  for (const [key, expectedValue] of Object.entries(expected)) {
    exact(reward[key], expectedValue, `$.rewardTemplate.${key}`);
  }
  if (!Array.isArray(reward.milestones) || reward.milestones.length !== 5) {
    fail("$.rewardTemplate.milestones", "must contain the five reviewed milestones");
  }
  let total = reward.challengeReserveBps;
  reward.milestones.forEach((item, index) => {
    const milestone = record(item, `$.rewardTemplate.milestones[${index}]`);
    exactKeys(milestone, MILESTONE_KEYS, `$.rewardTemplate.milestones[${index}]`);
    const [level, name, bps] = EXPECTED_MILESTONES[index];
    exact(milestone.level, level, `$.rewardTemplate.milestones[${index}].level`);
    exact(milestone.name, name, `$.rewardTemplate.milestones[${index}].name`);
    exact(milestone.outcomePoolBps, bps, `$.rewardTemplate.milestones[${index}].outcomePoolBps`);
    boundedString(milestone.evidence, `$.rewardTemplate.milestones[${index}].evidence`);
    total += milestone.outcomePoolBps;
  });
  exact(total, 10_000, "$.rewardTemplate.milestones");
  const roles = sortedStrings(reward.roles, "$.rewardTemplate.roles", {
    minimum: 6,
    maximum: 6,
  });
  exact(
    JSON.stringify(roles),
    JSON.stringify([
      "claimant",
      "falsifier",
      "formalizer",
      "independent-reproducer",
      "integrator",
      "maintainer",
    ]),
    "$.rewardTemplate.roles",
  );
}

function validateNodes(value, imported) {
  if (!Array.isArray(value) || value.length !== 13) {
    fail("$.nodes", "must contain the 13 reviewed nodes");
  }
  const nodes = value.map((item, index) => {
    const path = `$.nodes[${index}]`;
    const node = record(item, path);
    exactKeys(node, NODE_KEYS, path);
    const id = boundedString(node.id, `${path}.id`, 128);
    if (!NODE_ID_PATTERN.test(id)) fail(`${path}.id`, "must be a versioned lowercase identifier");
    boundedString(node.title, `${path}.title`, 160);
    if (!STAGE_RANK.has(node.stage)) fail(`${path}.stage`, "is not a reviewed stage");
    boundedString(node.summary, `${path}.summary`, 512);
    const prerequisites = sortedStrings(node.prerequisites, `${path}.prerequisites`, {
      minimum: 1,
      maximum: 8,
    });
    if (!EVIDENCE_LEVELS.has(node.evidenceTarget)) {
      fail(`${path}.evidenceTarget`, "is not a reviewed evidence target");
    }
    if (!CAPABILITY_CLASSES.has(node.capabilityClass)) {
      fail(`${path}.capabilityClass`, "is not a reviewed capability class");
    }
    stringList(node.artifacts, `${path}.artifacts`, { minimum: 3, maximum: 3 });
    exact(node.unlocksReward, false, `${path}.unlocksReward`);
    if (typeof node.questEligible !== "boolean") fail(`${path}.questEligible`, "must be boolean");
    if (node.stage === "frontier" && node.capabilityClass !== "quest") {
      fail(`${path}.capabilityClass`, "frontier nodes must be quests");
    }
    if (node.capabilityClass === "quest" && node.stage !== "frontier") {
      fail(`${path}.stage`, "quest nodes must be frontier nodes");
    }
    return { ...node, prerequisites };
  });
  const ids = nodes.map((node) => node.id);
  if (new Set(ids).size !== ids.length) fail("$.nodes", "must not contain duplicate IDs");
  const sortedIds = [...ids].sort();
  if (ids.some((id, index) => id !== sortedIds[index])) fail("$.nodes", "must be sorted by ID");
  const byId = new Map(nodes.map((node) => [node.id, node]));
  for (const [index, node] of nodes.entries()) {
    for (const prerequisite of node.prerequisites) {
      if (!imported.has(prerequisite) && !byId.has(prerequisite)) {
        fail(`$.nodes[${index}].prerequisites`, `unknown prerequisite ${prerequisite}`);
      }
      const internal = byId.get(prerequisite);
      if (internal && STAGE_RANK.get(internal.stage) > STAGE_RANK.get(node.stage)) {
        fail(`$.nodes[${index}].prerequisites`, "cannot depend on a later stage");
      }
    }
  }
  const visiting = new Set();
  const visited = new Set();
  const visit = (id) => {
    if (imported.has(id)) return;
    if (visiting.has(id)) fail("$.nodes", "prerequisite graph contains a cycle");
    if (visited.has(id)) return;
    visiting.add(id);
    const node = byId.get(id);
    if (!node) fail("$.nodes", `missing node ${id}`);
    node.prerequisites.forEach(visit);
    visiting.delete(id);
    visited.add(id);
  };
  ids.forEach(visit);
  return byId;
}

function validateQuestTemplate(value, byId, imported) {
  const quest = record(value, "$.questTemplate");
  exactKeys(quest, QUEST_KEYS, "$.questTemplate");
  exact(quest.id, "quest-math-formal-construction@1", "$.questTemplate.id");
  const node = byId.get(quest.id);
  if (!node || node.capabilityClass !== "quest") fail("$.questTemplate.id", "must resolve to the frontier quest node");
  exact(quest.status, "TEMPLATE_ONLY", "$.questTemplate.status");
  const core = sortedStrings(quest.requiredCoreCapabilities, "$.questTemplate.requiredCoreCapabilities", {
    minimum: 1,
    maximum: 4,
  });
  core.forEach((id, index) => {
    if (!imported.has(id)) fail(`$.questTemplate.requiredCoreCapabilities[${index}]`, "must be imported from the base tree");
  });
  const domains = sortedStrings(quest.allowedDomainCapabilities, "$.questTemplate.allowedDomainCapabilities", {
    minimum: 1,
    maximum: 8,
  });
  domains.forEach((id, index) => {
    const domain = byId.get(id);
    if (!domain || domain.capabilityClass !== "domain" || !domain.questEligible) {
      fail(`$.questTemplate.allowedDomainCapabilities[${index}]`, "must be a quest-eligible domain node");
    }
  });
  exact(
    quest.domainSelectionMode,
    "EXACTLY_ONE_AT_PACKET_FREEZE",
    "$.questTemplate.domainSelectionMode",
  );
  exact(
    quest.selectedDomainEvidenceMinimum,
    "E2",
    "$.questTemplate.selectedDomainEvidenceMinimum",
  );
  exact(
    quest.selectedDomainReceiptRequired,
    true,
    "$.questTemplate.selectedDomainReceiptRequired",
  );
  const relations = sortedStrings(quest.allowedArtifactRelations, "$.questTemplate.allowedArtifactRelations", {
    minimum: 3,
    maximum: 3,
  });
  exact(JSON.stringify(relations), JSON.stringify(["DISPROVES", "IMPLEMENTS", "PROVES"]), "$.questTemplate.allowedArtifactRelations");
  const bindings = sortedStrings(quest.requiredPacketBindings, "$.questTemplate.requiredPacketBindings", {
    minimum: 11,
    maximum: 11,
  });
  exact(
    JSON.stringify(bindings),
    JSON.stringify([
      "axiom_policy_digest",
      "checker_kernel_digests",
      "domain_capability_id",
      "falsifier_suite_digest",
      "formalization_digest",
      "prior_art_cutoff",
      "prior_art_manifest_digest",
      "selected_domain_evidence",
      "selected_domain_receipt_digest",
      "semantic_root",
      "statement_digest",
    ]),
    "$.questTemplate.requiredPacketBindings",
  );
  const order = sortedStrings(quest.milestoneOrder, "$.questTemplate.milestoneOrder", {
    minimum: 7,
    maximum: 7,
  });
  // Milestones are an order, not a set sorted lexicographically.
  exact(JSON.stringify(order), JSON.stringify(["E0", "E1", "E2", "E3", "E4", "E5", "E6"]), "$.questTemplate.milestoneOrder");
  const expected = {
    breakthroughStatus: "DERIVED_ONLY_AFTER_E5",
    selfDeclaredBreakthrough: false,
    popularityCanOverrideValidity: false,
    relationSpecificValidityRequired: true,
    implementsEstablishesTheoremValidity: false,
    semanticRootReplayAllowed: false,
    minimumEffectiveClusters: 3,
    minimumOrganizationRoots: 2,
    minimumImplementationRoots: 2,
    minimumExecutionEnvironments: 2,
    minimumKernelFamilies: 2,
    counterexampleEligible: true,
    sunsetRequired: true,
    economicEffect: "NONE",
    controlEffect: "NONE",
  };
  for (const [key, expectedValue] of Object.entries(expected)) {
    if (Number.isInteger(expectedValue)) positiveInteger(quest[key], `$.questTemplate.${key}`, 16);
    exact(quest[key], expectedValue, `$.questTemplate.${key}`);
  }
  exact(
    sha256(canonicalJson(quest)),
    REVIEWED_QUEST_TEMPLATE_SHA256,
    "$.questTemplate",
  );
}

export function validateConstructiveIntelligenceMathFrontier(
  value,
  { baseTreeRaw, exactRaw } = {},
) {
  const frontier = record(value, "$");
  exactKeys(frontier, TOP_LEVEL_KEYS, "$");
  exact(frontier.schema, MATH_FRONTIER_SCHEMA, "$.schema");
  for (const key of [
    "authoritative",
    "networkObserved",
    "rewardBearing",
    "governanceBearing",
  ]) {
    exact(frontier[key], false, `$.${key}`);
  }
  exact(frontier.snapshotDate, "2026-08-01", "$.snapshotDate");
  exact(frontier.policyVersion, "0.1.0", "$.policyVersion");
  const imported = validateBaseBinding(frontier.baseTree, baseTreeRaw);
  validateReleaseBoundary(frontier.releaseBoundary);
  validateConstitution(frontier.constitution);
  validateKarma(frontier.karma);
  validateRewardTemplate(frontier.rewardTemplate);
  exact(
    JSON.stringify(frontier.stages),
    JSON.stringify(STAGES),
    "$.stages",
  );
  const byId = validateNodes(frontier.nodes, imported);
  validateQuestTemplate(frontier.questTemplate, byId, imported);
  exact(
    sha256(canonicalJson(frontier)),
    REVIEWED_FRONTIER_CANONICAL_SHA256,
    "$",
  );
  if (typeof exactRaw === "string") {
    exact(sha256(exactRaw), REVIEWED_FRONTIER_BYTE_SHA256, "$");
  }
  return {
    schema: frontier.schema,
    policyVersion: frontier.policyVersion,
    nodeCount: frontier.nodes.length,
    stageCount: frontier.stages.length,
    liveAmount: frontier.rewardTemplate.liveAmount,
    karmaState: frontier.karma.state,
    economicEffect: frontier.rewardTemplate.economicEffect,
    controlEffect: frontier.karma.controlEffect,
    canonicalSha256: REVIEWED_FRONTIER_CANONICAL_SHA256,
    documentSha256: REVIEWED_FRONTIER_BYTE_SHA256,
  };
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
      if (depth > 64) fail("$", "JSON nesting exceeds the v0 limit of 64");
    } else if (character === "}" || character === "]") depth -= 1;
  }
}

function strictParse(raw, maximum) {
  if (typeof raw !== "string") fail("$", "raw document must be a string");
  if (Buffer.byteLength(raw, "utf8") > maximum) {
    fail("$", `raw document exceeds ${maximum} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw);
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    fail("$", `invalid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw);
  return parsed;
}

export function parseAndValidateConstructiveIntelligenceMathFrontier(
  raw,
  baseTreeRaw,
) {
  return validateConstructiveIntelligenceMathFrontier(
    strictParse(raw, MATH_FRONTIER_MAX_BYTES),
    { baseTreeRaw, exactRaw: raw },
  );
}

export function validateMathProblemPacket(value, frontier, frontierRaw) {
  const packet = record(value, "$");
  exactKeys(packet, PROBLEM_KEYS, "$");
  exact(packet.schema, MATH_PROBLEM_SCHEMA, "$.schema");
  exact(packet.synthetic, true, "$.synthetic");
  if (typeof frontierRaw !== "string") fail("$.frontierDocumentSha256", "exact frontier bytes are required");
  const documentDigest = `sha256:${sha256(frontierRaw)}`;
  exact(packet.frontierDocumentSha256, documentDigest, "$.frontierDocumentSha256");
  exact(packet.frontierDocumentSha256, `sha256:${REVIEWED_FRONTIER_BYTE_SHA256}`, "$.frontierDocumentSha256");
  exact(packet.questTemplateSha256, `sha256:${REVIEWED_QUEST_TEMPLATE_SHA256}`, "$.questTemplateSha256");
  const id = boundedString(packet.problemId, "$.problemId", 128);
  if (!PROBLEM_ID_PATTERN.test(id)) fail("$.problemId", "must be a lowercase problem identifier");
  if (!frontier.questTemplate.allowedDomainCapabilities.includes(packet.domainCapabilityId)) {
    fail("$.domainCapabilityId", "is not allowed by the quest template");
  }
  exact(
    packet.selectedDomainEvidence,
    frontier.questTemplate.selectedDomainEvidenceMinimum,
    "$.selectedDomainEvidence",
  );
  digest(packet.selectedDomainReceiptDigest, "$.selectedDomainReceiptDigest");
  if (!frontier.questTemplate.allowedArtifactRelations.includes(packet.artifactRelation)) {
    fail("$.artifactRelation", "is not allowed by the quest template");
  }
  for (const key of [
    "statementDigest",
    "formalizationDigest",
    "axiomPolicyDigest",
    "semanticRoot",
    "priorArtManifestDigest",
    "falsifierSuiteDigest",
  ]) {
    digest(packet[key], `$.${key}`);
  }
  if (typeof packet.priorArtCutoff !== "string" || !ISO_DATE_PATTERN.test(packet.priorArtCutoff)) {
    fail("$.priorArtCutoff", "must be an ISO date");
  }
  if (packet.priorArtCutoff > frontier.snapshotDate) {
    fail("$.priorArtCutoff", "cannot be after the frontier snapshot");
  }
  const kernels = sortedStrings(packet.checkerKernelDigests, "$.checkerKernelDigests", {
    minimum: frontier.questTemplate.minimumKernelFamilies,
    maximum: 8,
  });
  kernels.forEach((value, index) => digest(value, `$.checkerKernelDigests[${index}]`));
  exact(packet.breakthroughStatus, "NOT_EVALUATED", "$.breakthroughStatus");
  validateExactRecord(
    packet.karmaProjection,
    {
      schema: "zerone.karma.shadow-edge/v0",
      state: "ORDINAL",
      register: "priced-coherence",
      magnitude: "NONE",
    },
    PROBLEM_KARMA_KEYS,
    "$.karmaProjection",
  );
  exact(packet.economicEffect, "NONE", "$.economicEffect");
  exact(packet.controlEffect, "NONE", "$.controlEffect");
  exact(packet.qualification, "NONE", "$.qualification");
  exact(packet.liveAmount, "0", "$.liveAmount");
  const packetSha256 = sha256(canonicalJson(packet));
  exact(packetSha256, REVIEWED_PROBLEM_PACKET_SHA256, "$");
  return {
    schema: packet.schema,
    problemId: packet.problemId,
    domainCapabilityId: packet.domainCapabilityId,
    artifactRelation: packet.artifactRelation,
    karmaState: packet.karmaProjection.state,
    economicEffect: packet.economicEffect,
    controlEffect: packet.controlEffect,
    packetSha256,
  };
}

export function parseAndValidateMathProblemPacket(raw, frontier, frontierRaw) {
  return validateMathProblemPacket(
    strictParse(raw, MATH_PROBLEM_MAX_BYTES),
    frontier,
    frontierRaw,
  );
}

function readRegularFile(path) {
  const resolved = resolve(path);
  const stat = lstatSync(resolved);
  if (!stat.isFile() || stat.isSymbolicLink()) {
    throw new Error(`${path}: must be a regular non-symlink file`);
  }
  return readFileSync(realpathSync(resolved), "utf8");
}

function runCli() {
  if (process.argv.length !== 5) {
    console.error(
      "usage: node scripts/validate-constructive-intelligence-math-frontier.mjs FRONTIER BASE_TREE PROBLEM_PACKET",
    );
    process.exitCode = 2;
    return;
  }
  try {
    const frontierRaw = readRegularFile(process.argv[2]);
    const baseTreeRaw = readRegularFile(process.argv[3]);
    const packetRaw = readRegularFile(process.argv[4]);
    const frontier = strictParse(frontierRaw, MATH_FRONTIER_MAX_BYTES);
    const frontierSummary = validateConstructiveIntelligenceMathFrontier(
      frontier,
      { baseTreeRaw, exactRaw: frontierRaw },
    );
    const packetSummary = parseAndValidateMathProblemPacket(
      packetRaw,
      frontier,
      frontierRaw,
    );
    console.log(JSON.stringify({ frontier: frontierSummary, problem: packetSummary }, null, 2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}

const invokedPath = process.argv[1] ? pathToFileURL(resolve(process.argv[1])).href : "";
if (invokedPath === import.meta.url) runCli();
