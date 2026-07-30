/// <reference lib="dom" />

export const CONSTRUCTIVE_TREE_ENDPOINT =
  "/standards/constructive-intelligence-tree.v1.json";
export const CONSTRUCTIVE_TREE_MAX_BYTES = 262_144;

export const CONSTRUCTIVE_TREE_STAGES = [
  "foundation",
  "primitive",
  "assurance",
  "protocol",
  "quest",
] as const;
export const CONSTRUCTIVE_TREE_DOMAINS = [
  "mathematics",
  "security",
  "systems",
  "cryptography",
  "assurance",
  "protocols",
  "quests",
] as const;
export const CONSTRUCTIVE_TREE_EVIDENCE_LEVELS = [
  "E0",
  "E1",
  "E2",
  "E3",
  "E4",
  "E5",
  "E6",
] as const;
export const CONSTRUCTIVE_TREE_REWARD_ELIGIBILITY = [
  "qualification-only",
  "sponsor-milestones",
] as const;
export const CONSTRUCTIVE_TREE_DISCLOSURE_LANES = [
  "open-construction",
  "private-coordinated-repair",
  "controlled-operations",
] as const;

export type ConstructiveTreeStage =
  (typeof CONSTRUCTIVE_TREE_STAGES)[number];
export type ConstructiveTreeDomain =
  (typeof CONSTRUCTIVE_TREE_DOMAINS)[number];
export type ConstructiveTreeEvidence =
  (typeof CONSTRUCTIVE_TREE_EVIDENCE_LEVELS)[number];
export type ConstructiveTreeRewardEligibility =
  (typeof CONSTRUCTIVE_TREE_REWARD_ELIGIBILITY)[number];
export type ConstructiveTreeDisclosureLane =
  (typeof CONSTRUCTIVE_TREE_DISCLOSURE_LANES)[number];

interface ConstructiveTreeReleaseBoundary {
  addsConsensusBehavior: false;
  activatesRewards: false;
  movesFunds: false;
  grantsQualification: false;
  authorizesSecurityTesting: false;
  assertsProtocolSecurity: false;
  performsNetworkRequests: false;
  publishesConfidentialEvidence: false;
}

interface ConstructiveTreeMilestone {
  level: ConstructiveTreeEvidence;
  name: string;
  rewardBps: number;
  treatment: string;
}

interface ConstructiveTreeStandard {
  canonicalId: string;
  authority: string;
  title: string;
  revision: string;
  authorityStatus: string;
  normalizedMaturity: string;
  specification: string;
  statusCheckedAt: string;
  reviewAfter: string;
}

interface ConstructiveTreeCoverageTarget {
  id: string;
  minimumEffectiveClusters: number;
  minimumOrganizationRoots: number;
  minimumImplementationRoots: number;
  minimumExecutionEnvironments: number;
  minimumCases: number;
  requiresCheckerOrCorpusDigest: boolean;
}

interface ConstructiveTreeAcceptance {
  targetEvidence: ConstructiveTreeEvidence;
  scopeBounds: string[];
  scopeHash: string;
  coverageTargets: ConstructiveTreeCoverageTarget[];
  minimumEffectiveClusters: number;
  minimumOrganizationRoots: number;
  minimumImplementationRoots: number;
  minimumExecutionEnvironments: number;
  adoptionReceiptTypes: string[];
  privateEscalationRequired: boolean;
  prepublicationTriageRequired: boolean;
}

export interface ConstructiveTreeNode {
  id: string;
  title: string;
  stage: ConstructiveTreeStage;
  domain: ConstructiveTreeDomain;
  summary: string;
  prerequisites: string[];
  attainmentEvidence: ConstructiveTreeEvidence;
  rewardEligibility: ConstructiveTreeRewardEligibility;
  defaultDisclosureLane: ConstructiveTreeDisclosureLane;
  artifactRequirements: string[];
  revalidationTriggers: string[];
  standards: ConstructiveTreeStandard[];
  repositoryReferences: string[];
  acceptance: ConstructiveTreeAcceptance | null;
}

interface ConstructiveTreePolicy {
  artifactEdgeTypes: string[];
  breakthroughRecognition: {
    authorSelected: false;
    minimumEvidenceLevel: ConstructiveTreeEvidence;
    requiresPriorArtDelta: boolean;
    requiresAdoptionOrDescendantImpact: boolean;
  };
  funding: {
    skillUnlockCreatesReward: false;
    externalWorkDefault: string;
    protocolIssuanceGate: string;
    timeAloneUnlocksEvidence: false;
  };
  independence: {
    minimumEffectiveClusters: number;
    minimumOrganizationRoots: number;
    minimumImplementationRoots: number;
    minimumExecutionEnvironments: number;
    assignmentAfterArtifactFreeze: boolean;
    reviewPayOutcomeIndependent: boolean;
    rawAddressCountIsEvidence: false;
  };
  disclosure: {
    safetyIsHardGate: true;
    unknownSecurityImpactEscalatesTo: ConstructiveTreeDisclosureLane;
    publicExploitPlaintextAllowed: false;
    vendorHasPayoutVeto: false;
  };
  milestones: ConstructiveTreeMilestone[];
  challengeReserveBps: number;
}

export interface ConstructiveIntelligenceTree {
  schema: "zerone.constructive-intelligence-tree/v1";
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  snapshotDate: string;
  policyVersion: string;
  releaseBoundary: ConstructiveTreeReleaseBoundary;
  policy: ConstructiveTreePolicy;
  roots: string[];
  nodes: ConstructiveTreeNode[];
}

export interface ConstructiveTreeIndex {
  byId: ReadonlyMap<string, ConstructiveTreeNode>;
  dependentsById: ReadonlyMap<string, readonly ConstructiveTreeNode[]>;
}

export interface ConstructiveTreeFilters {
  query: string;
  stage: ConstructiveTreeStage | "all";
  domain: ConstructiveTreeDomain | "all";
  rewardEligibility: ConstructiveTreeRewardEligibility | "all";
}

export interface ConstructiveTreeFreshness {
  earliestReviewAfter: string | null;
  expiredStandardCount: number;
  isExpiredForActiveUse: boolean;
}

export interface ConstructiveTreeRewardSummary {
  qualificationOnlyCount: number;
  sponsorMilestoneCount: number;
  milestoneBps: number;
  challengeReserveBps: number;
  totalOutcomePoolBps: number;
  rewardBearing: false;
}

export interface ConstructiveTreeFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
}

export class ConstructiveTreeDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ConstructiveTreeDataError";
  }
}

type JsonObject = Record<string, unknown>;

const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
] as const;
const MAX_NODES = 64;
const MAX_ARRAY_ITEMS = 128;
const MAX_STRING_LENGTH = 8_192;
const NODE_ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const REPOSITORY_REFERENCE_PATTERN =
  /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;

function fail(path: string, message: string): never {
  throw new ConstructiveTreeDataError(`${path}: ${message}`);
}

function asObject(value: unknown, path: string): JsonObject {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value as JsonObject;
}

function asArray(value: unknown, path: string, maximum = MAX_ARRAY_ITEMS): unknown[] {
  if (!Array.isArray(value)) fail(path, "expected an array");
  if (value.length > maximum) fail(path, `must contain at most ${maximum} items`);
  return value;
}

function asString(value: unknown, path: string, maximum = MAX_STRING_LENGTH): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, `expected a non-empty string of at most ${maximum} characters`);
  }
  return value;
}

function asBoolean(value: unknown, path: string): boolean {
  if (typeof value !== "boolean") fail(path, "expected a boolean");
  return value;
}

function asInteger(
  value: unknown,
  path: string,
  minimum = 0,
  maximum = 10_000,
): number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value < minimum ||
    value > maximum
  ) {
    fail(path, `expected an integer from ${minimum} to ${maximum}`);
  }
  return value;
}

function asEnum<T extends readonly string[]>(
  value: unknown,
  values: T,
  path: string,
): T[number] {
  if (typeof value !== "string" || !values.includes(value)) {
    fail(path, `expected one of ${values.join(", ")}`);
  }
  return value as T[number];
}

function asIsoDate(value: unknown, path: string): string {
  const result = asString(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(result)) fail(path, "expected YYYY-MM-DD");
  const parsed = Date.parse(`${result}T00:00:00Z`);
  if (
    !Number.isFinite(parsed) ||
    new Date(parsed).toISOString().slice(0, 10) !== result
  ) {
    fail(path, "expected a real calendar date");
  }
  return result;
}

function asStringArray(
  value: unknown,
  path: string,
  maximum = MAX_ARRAY_ITEMS,
): string[] {
  const items = asArray(value, path, maximum).map((item, index) =>
    asString(item, `${path}[${index}]`),
  );
  if (new Set(items).size !== items.length) fail(path, "contains a duplicate");
  return items;
}

function parseStandard(value: unknown, path: string): ConstructiveTreeStandard {
  const standard = asObject(value, path);
  const specification = asString(
    standard.specification,
    `${path}.specification`,
    2_048,
  );
  let parsedUrl: URL;
  try {
    parsedUrl = new URL(specification);
  } catch {
    fail(`${path}.specification`, "expected an absolute URL");
  }
  if (
    parsedUrl.protocol !== "https:" ||
    parsedUrl.username !== "" ||
    parsedUrl.password !== "" ||
    parsedUrl.search !== "" ||
    parsedUrl.hash !== ""
  ) {
    fail(
      `${path}.specification`,
      "expected a credential-free HTTPS URL without query or fragment",
    );
  }
  return {
    canonicalId: asString(standard.canonicalId, `${path}.canonicalId`, 256),
    authority: asString(standard.authority, `${path}.authority`, 128),
    title: asString(standard.title, `${path}.title`, 512),
    revision: asString(standard.revision, `${path}.revision`, 256),
    authorityStatus: asString(
      standard.authorityStatus,
      `${path}.authorityStatus`,
      512,
    ),
    normalizedMaturity: asString(
      standard.normalizedMaturity,
      `${path}.normalizedMaturity`,
      128,
    ),
    specification: parsedUrl.toString(),
    statusCheckedAt: asIsoDate(
      standard.statusCheckedAt,
      `${path}.statusCheckedAt`,
    ),
    reviewAfter: asIsoDate(standard.reviewAfter, `${path}.reviewAfter`),
  };
}

function parseCoverageTarget(
  value: unknown,
  path: string,
): ConstructiveTreeCoverageTarget {
  const target = asObject(value, path);
  return {
    id: asString(target.id, `${path}.id`, 256),
    minimumEffectiveClusters: asInteger(
      target.minimumEffectiveClusters,
      `${path}.minimumEffectiveClusters`,
      1,
      64,
    ),
    minimumOrganizationRoots: asInteger(
      target.minimumOrganizationRoots,
      `${path}.minimumOrganizationRoots`,
      1,
      64,
    ),
    minimumImplementationRoots: asInteger(
      target.minimumImplementationRoots,
      `${path}.minimumImplementationRoots`,
      1,
      64,
    ),
    minimumExecutionEnvironments: asInteger(
      target.minimumExecutionEnvironments,
      `${path}.minimumExecutionEnvironments`,
      1,
      64,
    ),
    minimumCases: asInteger(
      target.minimumCases,
      `${path}.minimumCases`,
      1,
      100_000,
    ),
    requiresCheckerOrCorpusDigest: asBoolean(
      target.requiresCheckerOrCorpusDigest,
      `${path}.requiresCheckerOrCorpusDigest`,
    ),
  };
}

function parseAcceptance(
  value: unknown,
  path: string,
): ConstructiveTreeAcceptance | null {
  if (value === null) return null;
  const acceptance = asObject(value, path);
  const scopeHash = asString(acceptance.scopeHash, `${path}.scopeHash`, 64);
  if (!/^[a-f0-9]{64}$/.test(scopeHash)) {
    fail(`${path}.scopeHash`, "expected a lowercase SHA-256 digest");
  }
  const coverageTargets = asArray(
    acceptance.coverageTargets,
    `${path}.coverageTargets`,
    32,
  ).map((target, index) =>
    parseCoverageTarget(target, `${path}.coverageTargets[${index}]`),
  );
  if (coverageTargets.length === 0) {
    fail(`${path}.coverageTargets`, "must not be empty");
  }
  return {
    targetEvidence: asEnum(
      acceptance.targetEvidence,
      CONSTRUCTIVE_TREE_EVIDENCE_LEVELS,
      `${path}.targetEvidence`,
    ),
    scopeBounds: asStringArray(
      acceptance.scopeBounds,
      `${path}.scopeBounds`,
      64,
    ),
    scopeHash,
    coverageTargets,
    minimumEffectiveClusters: asInteger(
      acceptance.minimumEffectiveClusters,
      `${path}.minimumEffectiveClusters`,
      1,
      64,
    ),
    minimumOrganizationRoots: asInteger(
      acceptance.minimumOrganizationRoots,
      `${path}.minimumOrganizationRoots`,
      1,
      64,
    ),
    minimumImplementationRoots: asInteger(
      acceptance.minimumImplementationRoots,
      `${path}.minimumImplementationRoots`,
      1,
      64,
    ),
    minimumExecutionEnvironments: asInteger(
      acceptance.minimumExecutionEnvironments,
      `${path}.minimumExecutionEnvironments`,
      1,
      64,
    ),
    adoptionReceiptTypes: asStringArray(
      acceptance.adoptionReceiptTypes,
      `${path}.adoptionReceiptTypes`,
      32,
    ),
    privateEscalationRequired: asBoolean(
      acceptance.privateEscalationRequired,
      `${path}.privateEscalationRequired`,
    ),
    prepublicationTriageRequired: asBoolean(
      acceptance.prepublicationTriageRequired,
      `${path}.prepublicationTriageRequired`,
    ),
  };
}

function parseNode(value: unknown, path: string): ConstructiveTreeNode {
  const node = asObject(value, path);
  const id = asString(node.id, `${path}.id`, 256);
  if (!NODE_ID_PATTERN.test(id)) {
    fail(`${path}.id`, "expected a versioned lowercase capability identifier");
  }
  const stage = asEnum(node.stage, CONSTRUCTIVE_TREE_STAGES, `${path}.stage`);
  const rewardEligibility = asEnum(
    node.rewardEligibility,
    CONSTRUCTIVE_TREE_REWARD_ELIGIBILITY,
    `${path}.rewardEligibility`,
  );
  const acceptance = parseAcceptance(node.acceptance, `${path}.acceptance`);
  if (stage === "quest" && acceptance === null) {
    fail(`${path}.acceptance`, "quest nodes require acceptance bounds");
  }
  if (stage !== "quest" && acceptance !== null) {
    fail(`${path}.acceptance`, "only quest nodes may define acceptance bounds");
  }
  if (
    (rewardEligibility === "sponsor-milestones") !==
    (stage === "quest")
  ) {
    fail(
      `${path}.rewardEligibility`,
      "only quest nodes may use sponsor milestones",
    );
  }
  const standards = asArray(node.standards, `${path}.standards`, 16).map(
    (standard, index) =>
      parseStandard(standard, `${path}.standards[${index}]`),
  );
  const repositoryReferences = asStringArray(
    node.repositoryReferences,
    `${path}.repositoryReferences`,
    32,
  );
  repositoryReferences.forEach((reference, index) => {
    if (!REPOSITORY_REFERENCE_PATTERN.test(reference)) {
      fail(
        `${path}.repositoryReferences[${index}]`,
        "expected a safe repository-relative path",
      );
    }
  });
  const attainmentEvidence = asEnum(
    node.attainmentEvidence,
    CONSTRUCTIVE_TREE_EVIDENCE_LEVELS,
    `${path}.attainmentEvidence`,
  );
  if (
    acceptance !== null &&
    acceptance.targetEvidence !== attainmentEvidence
  ) {
    fail(
      `${path}.acceptance.targetEvidence`,
      "must match the node attainment evidence",
    );
  }
  return {
    id,
    title: asString(node.title, `${path}.title`, 512),
    stage,
    domain: asEnum(
      node.domain,
      CONSTRUCTIVE_TREE_DOMAINS,
      `${path}.domain`,
    ),
    summary: asString(node.summary, `${path}.summary`, 2_048),
    prerequisites: asStringArray(
      node.prerequisites,
      `${path}.prerequisites`,
      16,
    ),
    attainmentEvidence,
    rewardEligibility,
    defaultDisclosureLane: asEnum(
      node.defaultDisclosureLane,
      CONSTRUCTIVE_TREE_DISCLOSURE_LANES,
      `${path}.defaultDisclosureLane`,
    ),
    artifactRequirements: asStringArray(
      node.artifactRequirements,
      `${path}.artifactRequirements`,
      32,
    ),
    revalidationTriggers: asStringArray(
      node.revalidationTriggers,
      `${path}.revalidationTriggers`,
      32,
    ),
    standards,
    repositoryReferences,
    acceptance,
  };
}

function parsePolicy(value: unknown, path: string): ConstructiveTreePolicy {
  const policy = asObject(value, path);
  const breakthrough = asObject(
    policy.breakthroughRecognition,
    `${path}.breakthroughRecognition`,
  );
  const funding = asObject(policy.funding, `${path}.funding`);
  const independence = asObject(policy.independence, `${path}.independence`);
  const disclosure = asObject(policy.disclosure, `${path}.disclosure`);
  const milestones = asArray(policy.milestones, `${path}.milestones`, 7).map(
    (value, index): ConstructiveTreeMilestone => {
      const milestone = asObject(value, `${path}.milestones[${index}]`);
      return {
        level: asEnum(
          milestone.level,
          CONSTRUCTIVE_TREE_EVIDENCE_LEVELS,
          `${path}.milestones[${index}].level`,
        ),
        name: asString(
          milestone.name,
          `${path}.milestones[${index}].name`,
          128,
        ),
        rewardBps: asInteger(
          milestone.rewardBps,
          `${path}.milestones[${index}].rewardBps`,
        ),
        treatment: asString(
          milestone.treatment,
          `${path}.milestones[${index}].treatment`,
          128,
        ),
      };
    },
  );
  if (
    milestones.length !== CONSTRUCTIVE_TREE_EVIDENCE_LEVELS.length ||
    milestones.some(
      (milestone, index) =>
        milestone.level !== CONSTRUCTIVE_TREE_EVIDENCE_LEVELS[index],
    )
  ) {
    fail(`${path}.milestones`, "expected one ordered E0 through E6 ladder");
  }
  const e0 = milestones[0];
  const e1 = milestones[1];
  if (!e0 || !e1 || e0.rewardBps !== 0 || e1.rewardBps !== 0) {
    fail(`${path}.milestones`, "E0 and E1 must not allocate outcome-pool bps");
  }
  const challengeReserveBps = asInteger(
    policy.challengeReserveBps,
    `${path}.challengeReserveBps`,
  );
  const milestoneBps = milestones.reduce(
    (total, milestone) => total + milestone.rewardBps,
    0,
  );
  if (milestoneBps + challengeReserveBps !== 10_000) {
    fail(
      path,
      "milestone allocations plus challenge reserve must equal 10,000 bps",
    );
  }

  const authorSelected = asBoolean(
    breakthrough.authorSelected,
    `${path}.breakthroughRecognition.authorSelected`,
  );
  const skillUnlockCreatesReward = asBoolean(
    funding.skillUnlockCreatesReward,
    `${path}.funding.skillUnlockCreatesReward`,
  );
  const timeAloneUnlocksEvidence = asBoolean(
    funding.timeAloneUnlocksEvidence,
    `${path}.funding.timeAloneUnlocksEvidence`,
  );
  const rawAddressCountIsEvidence = asBoolean(
    independence.rawAddressCountIsEvidence,
    `${path}.independence.rawAddressCountIsEvidence`,
  );
  const safetyIsHardGate = asBoolean(
    disclosure.safetyIsHardGate,
    `${path}.disclosure.safetyIsHardGate`,
  );
  const publicExploitPlaintextAllowed = asBoolean(
    disclosure.publicExploitPlaintextAllowed,
    `${path}.disclosure.publicExploitPlaintextAllowed`,
  );
  const vendorHasPayoutVeto = asBoolean(
    disclosure.vendorHasPayoutVeto,
    `${path}.disclosure.vendorHasPayoutVeto`,
  );
  if (
    authorSelected ||
    skillUnlockCreatesReward ||
    timeAloneUnlocksEvidence ||
    rawAddressCountIsEvidence ||
    !safetyIsHardGate ||
    publicExploitPlaintextAllowed ||
    vendorHasPayoutVeto
  ) {
    fail(path, "critical recognition, funding, independence, or safety gate changed");
  }

  return {
    artifactEdgeTypes: asStringArray(
      policy.artifactEdgeTypes,
      `${path}.artifactEdgeTypes`,
      32,
    ),
    breakthroughRecognition: {
      authorSelected: false,
      minimumEvidenceLevel: asEnum(
        breakthrough.minimumEvidenceLevel,
        CONSTRUCTIVE_TREE_EVIDENCE_LEVELS,
        `${path}.breakthroughRecognition.minimumEvidenceLevel`,
      ),
      requiresPriorArtDelta: asBoolean(
        breakthrough.requiresPriorArtDelta,
        `${path}.breakthroughRecognition.requiresPriorArtDelta`,
      ),
      requiresAdoptionOrDescendantImpact: asBoolean(
        breakthrough.requiresAdoptionOrDescendantImpact,
        `${path}.breakthroughRecognition.requiresAdoptionOrDescendantImpact`,
      ),
    },
    funding: {
      skillUnlockCreatesReward: false,
      externalWorkDefault: asString(
        funding.externalWorkDefault,
        `${path}.funding.externalWorkDefault`,
        128,
      ),
      protocolIssuanceGate: asString(
        funding.protocolIssuanceGate,
        `${path}.funding.protocolIssuanceGate`,
        128,
      ),
      timeAloneUnlocksEvidence: false,
    },
    independence: {
      minimumEffectiveClusters: asInteger(
        independence.minimumEffectiveClusters,
        `${path}.independence.minimumEffectiveClusters`,
        1,
        64,
      ),
      minimumOrganizationRoots: asInteger(
        independence.minimumOrganizationRoots,
        `${path}.independence.minimumOrganizationRoots`,
        1,
        64,
      ),
      minimumImplementationRoots: asInteger(
        independence.minimumImplementationRoots,
        `${path}.independence.minimumImplementationRoots`,
        1,
        64,
      ),
      minimumExecutionEnvironments: asInteger(
        independence.minimumExecutionEnvironments,
        `${path}.independence.minimumExecutionEnvironments`,
        1,
        64,
      ),
      assignmentAfterArtifactFreeze: asBoolean(
        independence.assignmentAfterArtifactFreeze,
        `${path}.independence.assignmentAfterArtifactFreeze`,
      ),
      reviewPayOutcomeIndependent: asBoolean(
        independence.reviewPayOutcomeIndependent,
        `${path}.independence.reviewPayOutcomeIndependent`,
      ),
      rawAddressCountIsEvidence: false,
    },
    disclosure: {
      safetyIsHardGate: true,
      unknownSecurityImpactEscalatesTo: asEnum(
        disclosure.unknownSecurityImpactEscalatesTo,
        CONSTRUCTIVE_TREE_DISCLOSURE_LANES,
        `${path}.disclosure.unknownSecurityImpactEscalatesTo`,
      ),
      publicExploitPlaintextAllowed: false,
      vendorHasPayoutVeto: false,
    },
    milestones,
    challengeReserveBps,
  };
}

function validateGraph(
  nodes: ConstructiveTreeNode[],
  roots: string[],
): void {
  const byId = new Map<string, ConstructiveTreeNode>();
  for (const node of nodes) {
    if (byId.has(node.id)) fail("$.nodes", `duplicate node ${node.id}`);
    byId.set(node.id, node);
  }
  for (const [index, node] of nodes.entries()) {
    for (const prerequisite of node.prerequisites) {
      if (!byId.has(prerequisite)) {
        fail(
          `$.nodes[${index}].prerequisites`,
          `missing node ${prerequisite}`,
        );
      }
      if (prerequisite === node.id) {
        fail(`$.nodes[${index}].prerequisites`, "self dependency");
      }
    }
  }
  const graphRoots = nodes
    .filter((node) => node.prerequisites.length === 0)
    .map((node) => node.id)
    .sort();
  const declaredRoots = [...roots].sort();
  if (
    graphRoots.length !== declaredRoots.length ||
    graphRoots.some((root, index) => root !== declaredRoots[index])
  ) {
    fail("$.roots", "must exactly match nodes without prerequisites");
  }

  const state = new Map<string, "visiting" | "visited">();
  const visit = (id: string): void => {
    const current = state.get(id);
    if (current === "visiting") fail("$.nodes", `prerequisite cycle at ${id}`);
    if (current === "visited") return;
    state.set(id, "visiting");
    const node = byId.get(id);
    if (!node) fail("$.nodes", `missing node ${id}`);
    node.prerequisites.forEach(visit);
    state.set(id, "visited");
  };
  nodes.forEach((node) => visit(node.id));

  const dependents = new Map<string, string[]>(
    nodes.map((node) => [node.id, []]),
  );
  nodes.forEach((node) => {
    node.prerequisites.forEach((prerequisite) => {
      dependents.get(prerequisite)?.push(node.id);
    });
  });
  const reachable = new Set<string>();
  const queue = [...roots];
  while (queue.length > 0) {
    const id = queue.shift();
    if (!id || reachable.has(id)) continue;
    reachable.add(id);
    queue.push(...(dependents.get(id) ?? []));
  }
  if (reachable.size !== nodes.length) {
    fail("$.nodes", "every node must be reachable from a declared root");
  }
}

export function parseConstructiveIntelligenceTree(
  value: unknown,
): ConstructiveIntelligenceTree {
  const tree = asObject(value, "$");
  if (tree.schema !== "zerone.constructive-intelligence-tree/v1") {
    fail("$.schema", "unsupported constructive-intelligence tree schema");
  }
  for (const key of ["authoritative", "networkObserved", "rewardBearing"] as const) {
    if (tree[key] !== false) fail(`$.${key}`, "must remain false");
  }
  const boundarySource = asObject(tree.releaseBoundary, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    if (boundarySource[key] !== false) {
      fail(`$.releaseBoundary.${key}`, "must remain false");
    }
  }
  const roots = asStringArray(tree.roots, "$.roots", MAX_NODES);
  if (roots.length === 0) fail("$.roots", "must not be empty");
  const nodes = asArray(tree.nodes, "$.nodes", MAX_NODES).map((node, index) =>
    parseNode(node, `$.nodes[${index}]`),
  );
  if (nodes.length === 0) fail("$.nodes", "must not be empty");
  validateGraph(nodes, roots);
  return {
    schema: "zerone.constructive-intelligence-tree/v1",
    authoritative: false,
    networkObserved: false,
    rewardBearing: false,
    snapshotDate: asIsoDate(tree.snapshotDate, "$.snapshotDate"),
    policyVersion: asString(tree.policyVersion, "$.policyVersion", 64),
    releaseBoundary: {
      addsConsensusBehavior: false,
      activatesRewards: false,
      movesFunds: false,
      grantsQualification: false,
      authorizesSecurityTesting: false,
      assertsProtocolSecurity: false,
      performsNetworkRequests: false,
      publishesConfidentialEvidence: false,
    },
    policy: parsePolicy(tree.policy, "$.policy"),
    roots,
    nodes,
  };
}

export function parseConstructiveIntelligenceTreeJson(
  raw: string,
): ConstructiveIntelligenceTree {
  if (new TextEncoder().encode(raw).byteLength > CONSTRUCTIVE_TREE_MAX_BYTES) {
    fail("$", `document exceeds ${CONSTRUCTIVE_TREE_MAX_BYTES} UTF-8 bytes`);
  }
  let value: unknown;
  try {
    value = JSON.parse(raw) as unknown;
  } catch {
    fail("$", "malformed JSON");
  }
  return parseConstructiveIntelligenceTree(value);
}

export async function fetchConstructiveIntelligenceTree(
  options: ConstructiveTreeFetchOptions = {},
): Promise<ConstructiveIntelligenceTree> {
  const fetcher = options.fetcher ?? fetch;
  const timeoutMs = options.timeoutMs ?? 8_000;
  let response: Response;
  try {
    response = await fetcher(CONSTRUCTIVE_TREE_ENDPOINT, {
      cache: "no-cache",
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(timeoutMs),
    });
  } catch (error) {
    if (
      error instanceof DOMException &&
      (error.name === "AbortError" || error.name === "TimeoutError")
    ) {
      throw new ConstructiveTreeDataError("Static curriculum request timed out");
    }
    throw new ConstructiveTreeDataError("Static curriculum is unavailable");
  }
  if (!response.ok) {
    throw new ConstructiveTreeDataError(
      `Static curriculum returned HTTP ${response.status}`,
    );
  }
  const contentType = response.headers.get("content-type");
  if (contentType !== null && !/\bjson\b/i.test(contentType)) {
    throw new ConstructiveTreeDataError(
      "Static curriculum returned a non-JSON response",
    );
  }
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!/^\d+$/.test(declaredLength) ||
      Number(declaredLength) > CONSTRUCTIVE_TREE_MAX_BYTES)
  ) {
    throw new ConstructiveTreeDataError(
      "Static curriculum exceeded its size limit",
    );
  }
  return parseConstructiveIntelligenceTreeJson(await response.text());
}

export function buildConstructiveTreeIndex(
  tree: ConstructiveIntelligenceTree,
): ConstructiveTreeIndex {
  const byId = new Map(tree.nodes.map((node) => [node.id, node]));
  const dependents = new Map<string, ConstructiveTreeNode[]>(
    tree.nodes.map((node) => [node.id, []]),
  );
  tree.nodes.forEach((node) => {
    node.prerequisites.forEach((prerequisite) => {
      dependents.get(prerequisite)?.push(node);
    });
  });
  dependents.forEach((nodes) =>
    nodes.sort((left, right) => left.title.localeCompare(right.title, "en")),
  );
  return { byId, dependentsById: dependents };
}

function sortNodesForPresentation(
  nodes: ConstructiveTreeNode[],
): ConstructiveTreeNode[] {
  return [...nodes].sort((left, right) => {
    const stageDelta =
      CONSTRUCTIVE_TREE_STAGES.indexOf(left.stage) -
      CONSTRUCTIVE_TREE_STAGES.indexOf(right.stage);
    return stageDelta || left.title.localeCompare(right.title, "en");
  });
}

export function prerequisiteClosure(
  index: ConstructiveTreeIndex,
  nodeId: string,
): ConstructiveTreeNode[] {
  const node = index.byId.get(nodeId);
  if (!node) throw new ConstructiveTreeDataError(`Unknown capability ${nodeId}`);
  const found = new Set<string>();
  const visit = (id: string): void => {
    const current = index.byId.get(id);
    if (!current) return;
    current.prerequisites.forEach((prerequisite) => {
      if (found.has(prerequisite)) return;
      found.add(prerequisite);
      visit(prerequisite);
    });
  };
  visit(node.id);
  return sortNodesForPresentation(
    [...found].flatMap((id) => {
      const prerequisite = index.byId.get(id);
      return prerequisite ? [prerequisite] : [];
    }),
  );
}

function normaliseSearch(value: string): string {
  return value.normalize("NFKD").toLocaleLowerCase("en").trim();
}

export function filterConstructiveTreeNodes(
  tree: ConstructiveIntelligenceTree,
  filters: ConstructiveTreeFilters,
): ConstructiveTreeNode[] {
  const query = normaliseSearch(filters.query);
  return sortNodesForPresentation(
    tree.nodes.filter((node) => {
      if (filters.stage !== "all" && node.stage !== filters.stage) return false;
      if (filters.domain !== "all" && node.domain !== filters.domain) return false;
      if (
        filters.rewardEligibility !== "all" &&
        node.rewardEligibility !== filters.rewardEligibility
      ) {
        return false;
      }
      if (query === "") return true;
      const searchable = [
        node.id,
        node.title,
        node.summary,
        ...node.artifactRequirements,
        ...node.standards.flatMap((standard) => [
          standard.canonicalId,
          standard.authority,
          standard.title,
        ]),
      ]
        .join(" ")
        .normalize("NFKD")
        .toLocaleLowerCase("en");
      return searchable.includes(query);
    }),
  );
}

export function constructiveTreeFreshness(
  tree: ConstructiveIntelligenceTree,
  asOf: string,
): ConstructiveTreeFreshness {
  const normalizedAsOf = asIsoDate(asOf, "asOf");
  const standards = new Map<string, ConstructiveTreeStandard>();
  tree.nodes.forEach((node) => {
    node.standards.forEach((standard) => {
      standards.set(standard.canonicalId, standard);
    });
  });
  const reviewDates = [...standards.values()].map(
    (standard) => standard.reviewAfter,
  );
  const earliestReviewAfter =
    reviewDates.length > 0 ? [...reviewDates].sort()[0] ?? null : null;
  const expiredStandardCount = [...standards.values()].filter(
    (standard) => standard.reviewAfter < normalizedAsOf,
  ).length;
  return {
    earliestReviewAfter,
    expiredStandardCount,
    isExpiredForActiveUse: expiredStandardCount > 0,
  };
}

export function constructiveTreeRewardSummary(
  tree: ConstructiveIntelligenceTree,
): ConstructiveTreeRewardSummary {
  const milestoneBps = tree.policy.milestones.reduce(
    (total, milestone) => total + milestone.rewardBps,
    0,
  );
  return {
    qualificationOnlyCount: tree.nodes.filter(
      (node) => node.rewardEligibility === "qualification-only",
    ).length,
    sponsorMilestoneCount: tree.nodes.filter(
      (node) => node.rewardEligibility === "sponsor-milestones",
    ).length,
    milestoneBps,
    challengeReserveBps: tree.policy.challengeReserveBps,
    totalOutcomePoolBps: milestoneBps + tree.policy.challengeReserveBps,
    rewardBearing: false,
  };
}

export function formatBasisPoints(basisPoints: number): string {
  if (!Number.isSafeInteger(basisPoints) || basisPoints < 0) {
    throw new ConstructiveTreeDataError("Basis points must be a non-negative integer");
  }
  const whole = Math.floor(basisPoints / 100);
  const remainder = basisPoints % 100;
  return remainder === 0
    ? `${whole}%`
    : `${whole}.${remainder.toString().padStart(2, "0").replace(/0$/, "")}%`;
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

function humanise(value: string): string {
  return value
    .split("-")
    .map((part) => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
    .join(" ");
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${value}T00:00:00Z`));
}

function appendTextList(parent: HTMLElement, items: readonly string[]): void {
  const list = element("ul", "ci-tree-text-list");
  items.forEach((item) => {
    list.append(element("li", undefined, item));
  });
  parent.append(list);
}

function detailBlock(title: string): HTMLDetailsElement {
  const details = element("details", "ci-tree-detail-block");
  details.append(element("summary", undefined, title));
  return details;
}

function disclosureCopy(lane: ConstructiveTreeDisclosureLane): string {
  if (lane === "controlled-operations") {
    return "Controlled operations require owned infrastructure or explicit authorization. This tree authorizes no testing.";
  }
  if (lane === "private-coordinated-repair") {
    return "Evidence stays in coordinated private repair until safe disclosure. Exploit plaintext and target-identifying material do not belong here.";
  }
  return "Construction may begin openly, but an unexpected security-impacting result must be quarantined and escalated before publication.";
}

function eligibilityCopy(
  eligibility: ConstructiveTreeRewardEligibility,
): string {
  return eligibility === "sponsor-milestones"
    ? "Sponsor milestone template only. A separate immutable, funded case and escrow receipt are required before any payout can exist."
    : "Qualification-only curriculum. Demonstrating this capability creates no ZRN claim.";
}

function repositoryUrl(reference: string): string {
  const encoded = reference
    .split("/")
    .map((segment) => encodeURIComponent(segment))
    .join("/");
  return `https://github.com/cambridgetcg/zerone-core/blob/main/${encoded}`;
}

function renderPolicy(
  tree: ConstructiveIntelligenceTree,
  asOf: string,
): HTMLElement {
  const summary = constructiveTreeRewardSummary(tree);
  const freshness = constructiveTreeFreshness(tree, asOf);
  const wrapper = element("div", "ci-tree-policy-wrap");
  const facts = element("div", "ci-tree-facts");
  const factData: [string, string][] = [
    ["Release", "Static · non-authoritative"],
    ["Capabilities", `${tree.nodes.length} across 5 stages`],
    ["Funding", `${summary.sponsorMilestoneCount} sponsor quest templates`],
    ["Snapshot", `${formatDate(tree.snapshotDate)} · policy ${tree.policyVersion}`],
  ];
  factData.forEach(([label, value]) => {
    const fact = element("div");
    fact.append(
      element("span", undefined, label),
      element("strong", undefined, value),
    );
    facts.append(fact);
  });
  wrapper.append(facts);

  const freshnessNote = element(
    "div",
    freshness.isExpiredForActiveUse
      ? "ci-tree-freshness is-expired"
      : "ci-tree-freshness",
  );
  freshnessNote.setAttribute(
    "role",
    freshness.isExpiredForActiveUse ? "alert" : "note",
  );
  if (freshness.isExpiredForActiveUse) {
    freshnessNote.textContent =
      `${freshness.expiredStandardCount} authority snapshot` +
      `${freshness.expiredStandardCount === 1 ? " is" : "s are"} past review. ` +
      "This remains a historical viewer, but active qualification or funding must fail closed until revalidated.";
  } else if (freshness.earliestReviewAfter) {
    freshnessNote.textContent =
      `Historical standards snapshot. Earliest authority review is due ${formatDate(freshness.earliestReviewAfter)}. ` +
      "The explorer does not make this data active qualification or funding state.";
  }
  wrapper.append(freshnessNote);

  const policy = element("details", "ci-tree-policy");
  policy.open = true;
  const policySummary = element("summary");
  policySummary.append(
    element("strong", undefined, "Evidence and sponsor-milestone policy"),
    element("span", undefined, "Rewards inactive"),
  );
  policy.append(policySummary);

  const policyBody = element("div", "ci-tree-policy-body");
  const boundary = element("div", "ci-tree-boundary");
  boundary.setAttribute("role", "note");
  boundary.append(
    element("strong", undefined, "No skill unlock creates a reward."),
    element(
      "p",
      undefined,
      `${summary.qualificationOnlyCount} nodes are qualification-only and ${summary.sponsorMilestoneCount} are sponsor templates. ` +
        "This file adds no consensus behavior, moves no funds, grants no qualification, activates no reward, authorizes no security testing, " +
        "asserts no protocol security, performs no network requests itself, and publishes no confidential evidence.",
    ),
  );
  policyBody.append(boundary);

  const ladder = element("ol", "ci-tree-ladder");
  tree.policy.milestones.forEach((milestone) => {
    const item = element("li");
    const treatment =
      milestone.level === "E0"
        ? "Precedence only"
        : milestone.level === "E1"
          ? "Verified costs only · outside outcome %"
          : `${formatBasisPoints(milestone.rewardBps)} prospective outcome pool`;
    item.append(
      element("span", "ci-tree-level", milestone.level),
      element("strong", undefined, humanise(milestone.name)),
      element("small", undefined, treatment),
    );
    ladder.append(item);
  });
  const reserve = element("li", "ci-tree-reserve");
  reserve.append(
    element("span", "ci-tree-level", "R"),
    element("strong", undefined, "Challenge + remediation"),
    element(
      "small",
      undefined,
      `${formatBasisPoints(tree.policy.challengeReserveBps)} prospective reserve`,
    ),
  );
  ladder.append(reserve);
  policyBody.append(ladder);

  const breakthrough = element("div", "ci-tree-breakthrough");
  breakthrough.append(
    element("strong", undefined, "Breakthrough is retrospective."),
    element(
      "p",
      undefined,
      `It is never author-selected: at least ${tree.policy.breakthroughRecognition.minimumEvidenceLevel}, ` +
        "a delta against frozen prior art, and independent adoption or descendant impact. " +
        "Those facts recognize an outcome; they do not create money outside a prospectively funded escrow.",
    ),
  );
  const rewardModel = element(
    "a",
    "ci-tree-policy-link",
    "Pre-consensus reward model · inactive ↗",
  );
  rewardModel.href =
    "https://github.com/cambridgetcg/zerone-core/blob/main/docs/tokenomics/CONSTRUCTIVE-INTELLIGENCE-REWARDS.md";
  rewardModel.target = "_blank";
  rewardModel.rel = "noreferrer";
  policyBody.append(breakthrough, rewardModel);
  policy.append(policyBody);
  wrapper.append(policy);
  return wrapper;
}

function selectOption(
  value: string,
  label: string,
): HTMLOptionElement {
  const option = element("option", undefined, label);
  option.value = value;
  return option;
}

function renderStandard(
  standard: ConstructiveTreeStandard,
  asOf: string,
): HTMLElement {
  const card = element(
    "li",
    standard.reviewAfter < asOf
      ? "ci-tree-standard is-expired"
      : "ci-tree-standard",
  );
  const link = element("a");
  link.href = standard.specification;
  link.target = "_blank";
  link.rel = "noreferrer";
  link.append(
    element("strong", undefined, standard.title),
    element("span", undefined, "↗"),
  );
  card.append(
    link,
    element(
      "p",
      undefined,
      `${standard.authority} · ${standard.revision} · ${standard.authorityStatus}`,
    ),
    element("code", undefined, standard.canonicalId),
    element(
      "small",
      undefined,
      `Status checked ${formatDate(standard.statusCheckedAt)} · review after ${formatDate(standard.reviewAfter)}`,
    ),
  );
  return card;
}

function renderAcceptance(acceptance: ConstructiveTreeAcceptance): HTMLElement {
  const block = detailBlock("Quest acceptance bounds");
  const content = element("div", "ci-tree-detail-content");
  const floors = element("dl", "ci-tree-mini-facts");
  const values: [string, string][] = [
    ["Target evidence", acceptance.targetEvidence],
    ["Effective clusters", `${acceptance.minimumEffectiveClusters}+`],
    ["Organisation roots", `${acceptance.minimumOrganizationRoots}+`],
    ["Implementation roots", `${acceptance.minimumImplementationRoots}+`],
    ["Execution environments", `${acceptance.minimumExecutionEnvironments}+`],
  ];
  values.forEach(([term, description]) => {
    const row = element("div");
    row.append(
      element("dt", undefined, term),
      element("dd", undefined, description),
    );
    floors.append(row);
  });
  content.append(floors);
  content.append(
    element(
      "p",
      "ci-tree-safety-copy",
      `Private escalation: ${acceptance.privateEscalationRequired ? "required" : "not required"} · ` +
        `Prepublication triage: ${acceptance.prepublicationTriageRequired ? "required" : "not required"}`,
    ),
    element("h4", undefined, "Frozen scope"),
  );
  appendTextList(content, acceptance.scopeBounds);
  content.append(
    element("p", "ci-tree-digest-label", "Scope SHA-256"),
    element("code", "ci-tree-digest", acceptance.scopeHash),
    element("h4", undefined, "Coverage targets"),
  );
  const targets = element("ul", "ci-tree-coverage");
  acceptance.coverageTargets.forEach((target) => {
    const item = element("li");
    item.append(
      element("strong", undefined, target.id),
      element(
        "span",
        undefined,
        `${target.minimumCases}+ cases · ${target.minimumEffectiveClusters}+ clusters · ` +
          `${target.minimumImplementationRoots}+ implementations · ${target.minimumExecutionEnvironments}+ environments`,
      ),
      element(
        "small",
        undefined,
        target.requiresCheckerOrCorpusDigest
          ? "Checker or corpus digest required"
          : "No checker digest required",
      ),
    );
    targets.append(item);
  });
  content.append(
    targets,
    element("h4", undefined, "Independent adoption receipts"),
  );
  appendTextList(content, acceptance.adoptionReceiptTypes.map(humanise));
  block.append(content);
  return block;
}

function renderNodeDialog(
  dialog: HTMLDialogElement,
  node: ConstructiveTreeNode,
  tree: ConstructiveIntelligenceTree,
  index: ConstructiveTreeIndex,
  asOf: string,
  selectRelated: (id: string) => void,
): void {
  const body = element("div", "ci-tree-dialog-body");
  const head = element("div", "dialog-head");
  const heading = element("div");
  heading.append(
    element(
      "span",
      "card-kicker",
      `${humanise(node.stage)} · ${humanise(node.domain)}`,
    ),
  );
  const title = element("h2", undefined, node.title);
  title.id = "ci-tree-dialog-title";
  title.tabIndex = -1;
  heading.append(title);
  const close = element("button", "dialog-close", "×");
  close.type = "button";
  close.setAttribute("aria-label", "Close capability details");
  close.addEventListener("click", () => dialog.close());
  head.append(heading, close);
  body.append(head, element("p", "ci-tree-dialog-summary", node.summary));

  const badges = element("div", "ci-tree-badges");
  [
    `Requires ${node.attainmentEvidence}`,
    humanise(node.rewardEligibility),
    humanise(node.defaultDisclosureLane),
    `${prerequisiteClosure(index, node.id).length} total prerequisites`,
  ].forEach((value) => badges.append(element("span", undefined, value)));
  body.append(badges);

  const truth = element("div", "ci-tree-node-truth");
  truth.setAttribute("role", "note");
  truth.append(
    element("strong", undefined, eligibilityCopy(node.rewardEligibility)),
    element("p", undefined, disclosureCopy(node.defaultDisclosureLane)),
  );
  body.append(truth);

  const relationships = element("div", "ci-tree-relationships");
  const relationshipData: [
    string,
    readonly ConstructiveTreeNode[],
    string,
  ][] = [
    [
      "Direct prerequisites",
      node.prerequisites.flatMap((id) => {
        const prerequisite = index.byId.get(id);
        return prerequisite ? [prerequisite] : [];
      }),
      "This is a root capability.",
    ],
    [
      "Leads directly to",
      index.dependentsById.get(node.id) ?? [],
      "No direct dependents in v1.",
    ],
  ];
  relationshipData.forEach(([label, related, emptyCopy]) => {
    const group = element("div");
    group.append(element("h3", undefined, label));
    if (related.length === 0) {
      group.append(element("p", undefined, emptyCopy));
    } else {
      const list = element("ul");
      related.forEach((relatedNode) => {
        const item = element("li");
        const button = element("button", undefined, relatedNode.title);
        button.type = "button";
        button.addEventListener("click", () => selectRelated(relatedNode.id));
        item.append(button);
        list.append(item);
      });
      group.append(list);
    }
    relationships.append(group);
  });
  body.append(relationships);

  const artifacts = detailBlock("Required evidence artifacts");
  const artifactContent = element("div", "ci-tree-detail-content");
  appendTextList(artifactContent, node.artifactRequirements);
  artifacts.append(artifactContent);
  body.append(artifacts);

  const revalidation = detailBlock("Revalidate when");
  const revalidationContent = element("div", "ci-tree-detail-content");
  appendTextList(revalidationContent, node.revalidationTriggers);
  revalidation.append(revalidationContent);
  body.append(revalidation);

  if (node.standards.length > 0) {
    const standards = detailBlock(`Pinned standards (${node.standards.length})`);
    const standardsContent = element("div", "ci-tree-detail-content");
    const standardsList = element("ul", "ci-tree-standards");
    node.standards.forEach((standard) =>
      standardsList.append(renderStandard(standard, asOf)),
    );
    standardsContent.append(standardsList);
    standards.append(standardsContent);
    body.append(standards);
  }

  if (node.acceptance) body.append(renderAcceptance(node.acceptance));

  const references = detailBlock("Repository references");
  const referencesContent = element("div", "ci-tree-detail-content");
  const referencesList = element("ul", "ci-tree-references");
  node.repositoryReferences.forEach((reference) => {
    const item = element("li");
    const link = element("a");
    link.href = repositoryUrl(reference);
    link.target = "_blank";
    link.rel = "noreferrer";
    link.append(
      element("code", undefined, reference),
      element("span", undefined, " ↗"),
    );
    item.append(link);
    referencesList.append(item);
  });
  referencesContent.append(referencesList);
  references.append(referencesContent);
  body.append(references);

  const footer = element("div", "ci-tree-dialog-footer");
  footer.append(
    element(
      "p",
      undefined,
      `Static snapshot ${tree.snapshotDate} · policy ${tree.policyVersion} · not qualification or payment state.`,
    ),
  );
  const done = element("button", "button button-primary", "Back to the tree");
  done.type = "button";
  done.addEventListener("click", () => dialog.close());
  footer.append(done);
  body.append(footer);
  dialog.replaceChildren(body);
  dialog.setAttribute("aria-labelledby", title.id);
}

function renderExplorer(
  root: HTMLElement,
  tree: ConstructiveIntelligenceTree,
  asOf: string,
): void {
  const index = buildConstructiveTreeIndex(tree);
  const container = element("div", "ci-tree-explorer");
  container.append(renderPolicy(tree, asOf));

  const controls = element("form", "ci-tree-controls");
  controls.setAttribute("role", "search");
  controls.setAttribute("aria-label", "Filter constructive-intelligence capabilities");
  controls.addEventListener("submit", (event) => event.preventDefault());

  const searchLabel = element("label");
  const search = element("input");
  search.type = "search";
  search.placeholder = "TLS, lattice, fuzzing…";
  search.autocomplete = "off";
  searchLabel.append(element("span", undefined, "Search"), search);

  const stageLabel = element("label");
  const stage = element("select");
  stage.append(selectOption("all", "All stages"));
  CONSTRUCTIVE_TREE_STAGES.forEach((value) =>
    stage.append(selectOption(value, humanise(value))),
  );
  stageLabel.append(element("span", undefined, "Stage"), stage);

  const domainLabel = element("label");
  const domain = element("select");
  domain.append(selectOption("all", "All domains"));
  CONSTRUCTIVE_TREE_DOMAINS.forEach((value) =>
    domain.append(selectOption(value, humanise(value))),
  );
  domainLabel.append(element("span", undefined, "Domain"), domain);

  const fundingLabel = element("label");
  const funding = element("select");
  funding.append(
    selectOption("all", "All funding classes"),
    selectOption("qualification-only", "Qualification-only"),
    selectOption("sponsor-milestones", "Sponsor quest templates"),
  );
  fundingLabel.append(element("span", undefined, "Funding class"), funding);

  const reset = element("button", "button button-ghost compact", "Reset");
  reset.type = "button";
  controls.append(searchLabel, stageLabel, domainLabel, fundingLabel, reset);
  container.append(controls);

  const resultHead = element("div", "ci-tree-results-head");
  const resultStatus = element("p");
  resultStatus.setAttribute("role", "status");
  resultStatus.setAttribute("aria-live", "polite");
  const rawLink = element("a", undefined, "Open the raw standard ↗");
  rawLink.href = CONSTRUCTIVE_TREE_ENDPOINT;
  rawLink.target = "_blank";
  rawLink.rel = "noreferrer";
  resultHead.append(resultStatus, rawLink);
  container.append(resultHead);

  const map = element("div", "ci-tree-map");
  map.setAttribute(
    "aria-label",
    "Capability prerequisites grouped by progression stage",
  );
  container.append(map);

  const dialog = element("dialog", "ci-tree-dialog");
  container.append(dialog);
  root.replaceChildren(container);
  root.setAttribute("aria-busy", "false");

  let selectedId = tree.roots[0] ?? tree.nodes[0]?.id ?? "";
  const buttons = new Map<string, HTMLButtonElement>();

  const filters = (): ConstructiveTreeFilters => ({
    query: search.value,
    stage: stage.value as ConstructiveTreeStage | "all",
    domain: domain.value as ConstructiveTreeDomain | "all",
    rewardEligibility: funding.value as
      | ConstructiveTreeRewardEligibility
      | "all",
  });

  const updateRelationships = (): void => {
    const selected = index.byId.get(selectedId);
    const prerequisiteIds = new Set(selected?.prerequisites ?? []);
    const dependentIds = new Set(
      (index.dependentsById.get(selectedId) ?? []).map((node) => node.id),
    );
    buttons.forEach((button, id) => {
      let relationship = "";
      if (id === selectedId) relationship = "selected";
      else if (prerequisiteIds.has(id)) relationship = "prerequisite";
      else if (dependentIds.has(id)) relationship = "dependent";
      if (relationship) button.dataset.relationship = relationship;
      else delete button.dataset.relationship;
      if (id === selectedId) button.setAttribute("aria-current", "true");
      else button.removeAttribute("aria-current");
      const relation = button.querySelector<HTMLElement>(
        ".ci-tree-node-relationship",
      );
      if (relation) {
        relation.textContent =
          relationship === "selected"
            ? "Selected"
            : relationship === "prerequisite"
              ? "Direct prerequisite"
              : relationship === "dependent"
                ? "Direct dependent"
                : "";
        relation.hidden = relationship === "";
      }
    });
  };

  const selectNode = (
    id: string,
    openDialog: boolean,
    focusTitle = false,
  ): void => {
    const node = index.byId.get(id);
    if (!node) return;
    selectedId = id;
    updateRelationships();
    renderNodeDialog(dialog, node, tree, index, asOf, (relatedId) =>
      selectNode(relatedId, false, true),
    );
    if (openDialog && !dialog.open) dialog.showModal();
    if (focusTitle) {
      requestAnimationFrame(() => {
        dialog.querySelector<HTMLElement>("#ci-tree-dialog-title")?.focus();
      });
    }
  };

  const renderMap = (): void => {
    const visible = filterConstructiveTreeNodes(tree, filters());
    resultStatus.textContent =
      `${visible.length} of ${tree.nodes.length} capabilities` +
      (visible.length === tree.nodes.length ? "" : " match these filters");
    buttons.clear();
    map.replaceChildren();
    if (visible.length === 0) {
      const empty = element("div", "ci-tree-empty");
      empty.append(
        element("strong", undefined, "No capabilities match."),
        element(
          "p",
          undefined,
          "Try a broader search or reset the stage, domain, and funding filters.",
        ),
      );
      map.append(empty);
      return;
    }
    if (!visible.some((node) => node.id === selectedId)) {
      selectedId = visible[0]?.id ?? selectedId;
    }
    CONSTRUCTIVE_TREE_STAGES.forEach((stageName) => {
      const nodes = visible.filter((node) => node.stage === stageName);
      const section = element("section", "ci-tree-stage");
      const stageId = `ci-tree-stage-${stageName}`;
      const header = element("div", "ci-tree-stage-head");
      const heading = element("h3", undefined, humanise(stageName));
      heading.id = stageId;
      header.append(
        element(
          "span",
          undefined,
          `${CONSTRUCTIVE_TREE_STAGES.indexOf(stageName) + 1}`.padStart(2, "0"),
        ),
        heading,
        element("small", undefined, `${nodes.length}`),
      );
      section.setAttribute("aria-labelledby", stageId);
      section.append(header);
      if (nodes.length === 0) {
        section.append(
          element("p", "ci-tree-stage-empty", "No matches in this stage."),
        );
      } else {
        const list = element("ol", "ci-tree-nodes");
        nodes.forEach((node) => {
          const item = element("li");
          const button = element("button", "ci-tree-node");
          button.type = "button";
          button.setAttribute("aria-haspopup", "dialog");
          button.setAttribute("aria-controls", "ci-tree-node-dialog");
          button.dataset.nodeId = node.id;
          button.append(
            element("strong", undefined, node.title),
            element(
              "span",
              "ci-tree-node-meta",
              `${node.attainmentEvidence} · ${humanise(node.domain)}`,
            ),
            element(
              "span",
              "ci-tree-node-relationship",
              "",
            ),
          );
          const relation = button.lastElementChild;
          if (relation instanceof HTMLElement) relation.hidden = true;
          button.addEventListener("click", () => selectNode(node.id, true));
          buttons.set(node.id, button);
          item.append(button);
          list.append(item);
        });
        section.append(list);
      }
      map.append(section);
    });
    updateRelationships();
  };

  [search, stage, domain, funding].forEach((control) => {
    control.addEventListener(
      control === search ? "input" : "change",
      renderMap,
    );
  });
  reset.addEventListener("click", () => {
    search.value = "";
    stage.value = "all";
    domain.value = "all";
    funding.value = "all";
    renderMap();
    search.focus();
  });
  dialog.id = "ci-tree-node-dialog";
  dialog.addEventListener("click", (event) => {
    if (event.target === dialog) dialog.close();
  });
  dialog.addEventListener("close", () => {
    buttons.get(selectedId)?.focus({ preventScroll: true });
  });
  renderMap();
  const initial = index.byId.get(selectedId);
  if (initial) {
    renderNodeDialog(dialog, initial, tree, index, asOf, (relatedId) =>
      selectNode(relatedId, false, true),
    );
  }
}

function todayUtc(): string {
  return new Date().toISOString().slice(0, 10);
}

export async function initialiseConstructiveTree(
  root: HTMLElement,
  options: ConstructiveTreeFetchOptions & { asOf?: string } = {},
): Promise<void> {
  const load = async (): Promise<void> => {
    root.setAttribute("aria-busy", "true");
    try {
      const tree = await fetchConstructiveIntelligenceTree(options);
      renderExplorer(root, tree, options.asOf ?? todayUtc());
      if (window.location.hash === "#skills") {
        requestAnimationFrame(() => {
          root.closest<HTMLElement>("#skills")?.scrollIntoView({
            block: "start",
          });
        });
      }
    } catch (error) {
      root.setAttribute("aria-busy", "false");
      const state = element("div", "ci-tree-load-error");
      state.setAttribute("role", "alert");
      state.append(
        element("strong", undefined, "The static curriculum could not be loaded."),
        element(
          "p",
          undefined,
          error instanceof Error
            ? error.message
            : "The response was unavailable or invalid.",
        ),
      );
      const actions = element("div", "ci-tree-load-actions");
      const retry = element("button", "button button-primary", "Try again");
      retry.type = "button";
      retry.addEventListener("click", () => {
        retry.disabled = true;
        retry.textContent = "Trying again…";
        void load();
      });
      const source = element("a", "button button-ghost", "Open raw JSON");
      source.href = CONSTRUCTIVE_TREE_ENDPOINT;
      source.target = "_blank";
      source.rel = "noreferrer";
      actions.append(retry, source);
      state.append(actions);
      root.replaceChildren(state);
    }
  };
  await load();
}
