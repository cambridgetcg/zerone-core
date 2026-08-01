/// <reference lib="dom" />

export const MATH_FRONTIER_ENDPOINT =
  "/standards/constructive-intelligence-math-frontier.v0.json";
export const MATH_FRONTIER_MAX_BYTES = 131_072;
export const MATH_FRONTIER_SHA256 =
  "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456";

export const MATH_FRONTIER_STAGES = [
  "ground",
  "craft",
  "assurance",
  "frontier",
] as const;
export const MATH_FRONTIER_EVIDENCE_LEVELS = [
  "E0",
  "E1",
  "E2",
  "E3",
  "E4",
  "E5",
  "E6",
] as const;
export const MATH_FRONTIER_CAPABILITY_CLASSES = [
  "foundation",
  "domain",
  "method",
  "assurance",
  "quest",
] as const;

export type MathFrontierStage = (typeof MATH_FRONTIER_STAGES)[number];
export type MathFrontierEvidence =
  (typeof MATH_FRONTIER_EVIDENCE_LEVELS)[number];
export type MathFrontierCapabilityClass =
  (typeof MATH_FRONTIER_CAPABILITY_CLASSES)[number];

interface MathFrontierBaseTree {
  schema: "zerone.constructive-intelligence-tree/v1";
  policyVersion: string;
  documentSha256: string;
  policySha256: string;
  requiredCapabilities: string[];
}

interface MathFrontierReleaseBoundary {
  addsConsensusBehavior: false;
  activatesRewards: false;
  movesFunds: false;
  grantsQualification: false;
  grantsGovernancePower: false;
  ranksPersons: false;
  authorizesSecurityTesting: false;
  performsNetworkRequests: false;
  publishesConfidentialEvidence: false;
}

interface MathFrontierConstitution {
  identityLabelInvariant: true;
  reservedSeatsAllowed: false;
  reservedSharesAllowed: false;
  creatorPrivilege: false;
  operatorPrivilege: false;
  humanAiClassMultiplier: false;
  stakeAffectsValidity: false;
  stakeAffectsVoice: false;
  wealthAffectsEligibility: false;
  wealthAffectsReward: false;
  rewardBalanceAffectsVoice: false;
  karmaMagnitudeAffectsVoice: false;
  karmaEligibilityOnly: true;
  controllerMergeCanOnlyReduceVoice: true;
  controllerConflictRecusalRequired: true;
  recusalQuorumRecomputed: true;
  selectedControllerVoice: "ONE_CONTROLLER_ONE_VOICE";
  currentActivationAuthority: "NONE";
  futureGovernance: "KARMA_ELIGIBILITY_AND_SORTITION_NOT_IMPLEMENTED";
  ordinaryStakeVoteCanActivate: false;
  emergencyAuthorityCanActivate: false;
  emergencyAuthorityScope: "PAUSE_ONLY";
}

interface MathFrontierKarma {
  mode: "ORDINAL_SHADOW_ONLY";
  eventSchema: "zerone.karma.shadow-edge/v0";
  state: "ORDINAL";
  register: "priced-coherence";
  truthOracle: false;
  onChainRecognition: false;
  magnitude: "NONE";
  transferable: false;
  spendable: false;
  rewardMultiplier: false;
  governanceWeight: false;
  economicEffect: "NONE";
  controlEffect: "NONE";
  futureRecognitionRequiresSignedReceipts: true;
  futureRecognitionRequiresControllerResolution: true;
  excludedFromRecognition: string[];
}

export interface MathFrontierMilestone {
  level: MathFrontierEvidence;
  name: string;
  outcomePoolBps: number;
  evidence: string;
}

interface MathFrontierRewardTemplate {
  mode: "PROSPECTIVE_SPONSOR_ESCROW_ONLY";
  economicEffect: "NONE";
  liveAmount: "0";
  skillUnlockCreatesEntitlement: false;
  breakthroughCreatesEntitlement: false;
  protocolIssuanceAllowed: false;
  dedicatedEscrowRequired: true;
  policyFrozenBeforeAdmission: true;
  singleSettlementRequired: true;
  verifiedCostsTreatment: "SEPARATE_PREAUTHORIZED_CAP";
  milestones: MathFrontierMilestone[];
  challengeReserveBps: number;
  roles: string[];
  disproofDisposition: "E4_TO_COMPLIANT_FALSIFIER_CLAIMANT_UNPAID";
  unallocatedDisposition: "PROSPECTIVELY_NAMED_REFUND_OR_COMMONS";
  openLiabilityCap: "DEDICATED_ESCROW";
  specializedResearchSpendPathAllowed: false;
}

export interface MathFrontierNode {
  id: string;
  title: string;
  stage: MathFrontierStage;
  summary: string;
  prerequisites: string[];
  evidenceTarget: MathFrontierEvidence;
  capabilityClass: MathFrontierCapabilityClass;
  artifacts: string[];
  unlocksReward: false;
  questEligible: boolean;
}

interface MathFrontierQuestTemplate {
  id: string;
  status: "TEMPLATE_ONLY";
  requiredCoreCapabilities: string[];
  allowedDomainCapabilities: string[];
  domainSelectionMode: "EXACTLY_ONE_AT_PACKET_FREEZE";
  selectedDomainEvidenceMinimum: "E2";
  selectedDomainReceiptRequired: true;
  allowedArtifactRelations: string[];
  requiredPacketBindings: string[];
  milestoneOrder: MathFrontierEvidence[];
  breakthroughStatus: "DERIVED_ONLY_AFTER_E5";
  selfDeclaredBreakthrough: false;
  popularityCanOverrideValidity: false;
  relationSpecificValidityRequired: true;
  implementsEstablishesTheoremValidity: false;
  semanticRootReplayAllowed: false;
  minimumEffectiveClusters: number;
  minimumOrganizationRoots: number;
  minimumImplementationRoots: number;
  minimumExecutionEnvironments: number;
  minimumKernelFamilies: number;
  counterexampleEligible: true;
  sunsetRequired: true;
  economicEffect: "NONE";
  controlEffect: "NONE";
}

export interface MathFrontier {
  schema: "zerone.constructive-intelligence-math-frontier/v0";
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  governanceBearing: false;
  snapshotDate: string;
  policyVersion: string;
  baseTree: MathFrontierBaseTree;
  releaseBoundary: MathFrontierReleaseBoundary;
  constitution: MathFrontierConstitution;
  karma: MathFrontierKarma;
  rewardTemplate: MathFrontierRewardTemplate;
  stages: MathFrontierStage[];
  nodes: MathFrontierNode[];
  questTemplate: MathFrontierQuestTemplate;
}

export interface MathFrontierIndex {
  byId: ReadonlyMap<string, MathFrontierNode>;
  dependentsById: ReadonlyMap<string, readonly MathFrontierNode[]>;
}

export interface MathFrontierFilters {
  query: string;
  stage: MathFrontierStage | "all";
  capabilityClass: MathFrontierCapabilityClass | "all";
}

export interface MathFrontierFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class MathFrontierDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "MathFrontierDataError";
  }
}

type JsonObject = Record<string, unknown>;

const MAX_ARRAY_ITEMS = 128;
const MAX_NODES = 64;
const MAX_STRING_LENGTH = 8_192;
const NODE_ID_PATTERN =
  /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const SHA256_PATTERN = /^sha256:[a-f0-9]{64}$/;
const MATH_FRONTIER_SNAPSHOT_DATE = "2026-08-01";
const MATH_FRONTIER_POLICY_VERSION = "0.1.0";
const BASE_TREE_POLICY_VERSION = "1.0.0";
const BASE_TREE_DOCUMENT_SHA256 =
  "sha256:8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
const BASE_TREE_POLICY_SHA256 =
  "sha256:36116220c7f17dd06f8bda2217d79a000aaac771075a709004686b233402abc7";
const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsGovernancePower",
  "ranksPersons",
  "authorizesSecurityTesting",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
] as const;
const REQUIRED_BASE_CAPABILITIES = [
  "math-algebra-finite-fields@1",
  "math-lattices-polynomial-rings@1",
  "math-probability-information-complexity@1",
  "math-proofcraft@1",
] as const;
const KARMA_EXCLUSIONS = [
  "external-unverified",
  "reciprocal-cycle",
  "same-controller",
  "self",
] as const;
const REWARD_ROLES = [
  "claimant",
  "falsifier",
  "formalizer",
  "independent-reproducer",
  "integrator",
  "maintainer",
] as const;
const ARTIFACT_RELATIONS = [
  "DISPROVES",
  "IMPLEMENTS",
  "PROVES",
] as const;
const PACKET_BINDINGS = [
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
] as const;
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
] as const;

function fail(path: string, message: string): never {
  throw new MathFrontierDataError(path + ": " + message);
}

function asObject(value: unknown, path: string): JsonObject {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
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
    if (!expectedSet.has(key)) fail(path + "." + key, "unknown field");
  }
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) fail(path + "." + key, "missing field");
  }
}

function asArray(
  value: unknown,
  path: string,
  maximum = MAX_ARRAY_ITEMS,
): unknown[] {
  if (!Array.isArray(value)) fail(path, "expected an array");
  if (value.length > maximum) {
    fail(path, "must contain at most " + maximum + " items");
  }
  return value;
}

function asString(
  value: unknown,
  path: string,
  maximum = MAX_STRING_LENGTH,
): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, "expected a non-empty string of at most " + maximum + " characters");
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
    fail(path, "expected an integer from " + minimum + " to " + maximum);
  }
  return value;
}

function asLiteral<T extends string | boolean | number>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) {
    fail(path, "must remain " + String(expected));
  }
  return expected;
}

function asEnum<T extends readonly string[]>(
  value: unknown,
  values: T,
  path: string,
): T[number] {
  if (typeof value !== "string" || !values.includes(value)) {
    fail(path, "expected one of " + values.join(", "));
  }
  return value as T[number];
}

function asIsoDate(value: unknown, path: string): string {
  const result = asString(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(result)) fail(path, "expected YYYY-MM-DD");
  const parsed = Date.parse(result + "T00:00:00Z");
  if (
    !Number.isFinite(parsed) ||
    new Date(parsed).toISOString().slice(0, 10) !== result
  ) {
    fail(path, "expected a real calendar date");
  }
  return result;
}

function asVersionedId(value: unknown, path: string): string {
  const id = asString(value, path, 256);
  if (!NODE_ID_PATTERN.test(id)) {
    fail(path, "expected a versioned lowercase capability identifier");
  }
  return id;
}

function asSha256(value: unknown, path: string): string {
  const digest = asString(value, path, 71);
  if (!SHA256_PATTERN.test(digest)) {
    fail(path, "expected a sha256: lowercase digest");
  }
  return digest;
}

function asStringArray(
  value: unknown,
  path: string,
  maximum = MAX_ARRAY_ITEMS,
): string[] {
  const items = asArray(value, path, maximum).map((item, index) =>
    asString(item, path + "[" + index + "]"),
  );
  if (new Set(items).size !== items.length) fail(path, "contains a duplicate");
  return items;
}

function assertOrdered(
  actual: readonly string[],
  expected: readonly string[],
  path: string,
): void {
  if (
    actual.length !== expected.length ||
    actual.some((value, index) => value !== expected[index])
  ) {
    fail(path, "must exactly equal " + expected.join(", "));
  }
}

function assertLexicallySorted(values: readonly string[], path: string): void {
  const sorted = [...values].sort((left, right) => left.localeCompare(right, "en"));
  if (values.some((value, index) => value !== sorted[index])) {
    fail(path, "must be sorted");
  }
}

function parseBaseTree(value: unknown, path: string): MathFrontierBaseTree {
  const base = asObject(value, path);
  assertExactKeys(
    base,
    [
      "schema",
      "policyVersion",
      "documentSha256",
      "policySha256",
      "requiredCapabilities",
    ],
    path,
  );
  const requiredCapabilities = asStringArray(
    base.requiredCapabilities,
    path + ".requiredCapabilities",
    16,
  ).map((id, index) =>
    asVersionedId(id, path + ".requiredCapabilities[" + index + "]"),
  );
  assertOrdered(
    requiredCapabilities,
    REQUIRED_BASE_CAPABILITIES,
    path + ".requiredCapabilities",
  );
  return {
    schema: asLiteral(
      base.schema,
      "zerone.constructive-intelligence-tree/v1",
      path + ".schema",
    ),
    policyVersion: asLiteral(
      base.policyVersion,
      BASE_TREE_POLICY_VERSION,
      path + ".policyVersion",
    ),
    documentSha256: asLiteral(
      asSha256(base.documentSha256, path + ".documentSha256"),
      BASE_TREE_DOCUMENT_SHA256,
      path + ".documentSha256",
    ),
    policySha256: asLiteral(
      asSha256(base.policySha256, path + ".policySha256"),
      BASE_TREE_POLICY_SHA256,
      path + ".policySha256",
    ),
    requiredCapabilities,
  };
}

function parseReleaseBoundary(
  value: unknown,
  path: string,
): MathFrontierReleaseBoundary {
  const boundary = asObject(value, path);
  assertExactKeys(boundary, RELEASE_BOUNDARY_KEYS, path);
  for (const key of RELEASE_BOUNDARY_KEYS) {
    asLiteral(boundary[key], false, path + "." + key);
  }
  return {
    addsConsensusBehavior: false,
    activatesRewards: false,
    movesFunds: false,
    grantsQualification: false,
    grantsGovernancePower: false,
    ranksPersons: false,
    authorizesSecurityTesting: false,
    performsNetworkRequests: false,
    publishesConfidentialEvidence: false,
  };
}

function parseConstitution(
  value: unknown,
  path: string,
): MathFrontierConstitution {
  const constitution = asObject(value, path);
  assertExactKeys(
    constitution,
    [
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
    ],
    path,
  );
  asLiteral(
    constitution.identityLabelInvariant,
    true,
    path + ".identityLabelInvariant",
  );
  for (const key of [
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
    "ordinaryStakeVoteCanActivate",
    "emergencyAuthorityCanActivate",
  ] as const) {
    asLiteral(constitution[key], false, path + "." + key);
  }
  for (const key of [
    "karmaEligibilityOnly",
    "controllerMergeCanOnlyReduceVoice",
    "controllerConflictRecusalRequired",
    "recusalQuorumRecomputed",
  ] as const) {
    asLiteral(constitution[key], true, path + "." + key);
  }
  return {
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
    selectedControllerVoice: asLiteral(
      constitution.selectedControllerVoice,
      "ONE_CONTROLLER_ONE_VOICE",
      path + ".selectedControllerVoice",
    ),
    currentActivationAuthority: asLiteral(
      constitution.currentActivationAuthority,
      "NONE",
      path + ".currentActivationAuthority",
    ),
    futureGovernance: asLiteral(
      constitution.futureGovernance,
      "KARMA_ELIGIBILITY_AND_SORTITION_NOT_IMPLEMENTED",
      path + ".futureGovernance",
    ),
    ordinaryStakeVoteCanActivate: false,
    emergencyAuthorityCanActivate: false,
    emergencyAuthorityScope: asLiteral(
      constitution.emergencyAuthorityScope,
      "PAUSE_ONLY",
      path + ".emergencyAuthorityScope",
    ),
  };
}

function parseKarma(value: unknown, path: string): MathFrontierKarma {
  const karma = asObject(value, path);
  assertExactKeys(
    karma,
    [
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
    ],
    path,
  );
  for (const key of [
    "truthOracle",
    "onChainRecognition",
    "transferable",
    "spendable",
    "rewardMultiplier",
    "governanceWeight",
  ] as const) {
    asLiteral(karma[key], false, path + "." + key);
  }
  for (const key of [
    "futureRecognitionRequiresSignedReceipts",
    "futureRecognitionRequiresControllerResolution",
  ] as const) {
    asLiteral(karma[key], true, path + "." + key);
  }
  const excludedFromRecognition = asStringArray(
    karma.excludedFromRecognition,
    path + ".excludedFromRecognition",
    16,
  );
  assertOrdered(
    excludedFromRecognition,
    KARMA_EXCLUSIONS,
    path + ".excludedFromRecognition",
  );
  return {
    mode: asLiteral(karma.mode, "ORDINAL_SHADOW_ONLY", path + ".mode"),
    eventSchema: asLiteral(
      karma.eventSchema,
      "zerone.karma.shadow-edge/v0",
      path + ".eventSchema",
    ),
    state: asLiteral(karma.state, "ORDINAL", path + ".state"),
    register: asLiteral(
      karma.register,
      "priced-coherence",
      path + ".register",
    ),
    truthOracle: false,
    onChainRecognition: false,
    magnitude: asLiteral(karma.magnitude, "NONE", path + ".magnitude"),
    transferable: false,
    spendable: false,
    rewardMultiplier: false,
    governanceWeight: false,
    economicEffect: asLiteral(
      karma.economicEffect,
      "NONE",
      path + ".economicEffect",
    ),
    controlEffect: asLiteral(
      karma.controlEffect,
      "NONE",
      path + ".controlEffect",
    ),
    futureRecognitionRequiresSignedReceipts: true,
    futureRecognitionRequiresControllerResolution: true,
    excludedFromRecognition,
  };
}

function parseMilestone(
  value: unknown,
  path: string,
): MathFrontierMilestone {
  const milestone = asObject(value, path);
  assertExactKeys(
    milestone,
    ["level", "name", "outcomePoolBps", "evidence"],
    path,
  );
  return {
    level: asEnum(
      milestone.level,
      MATH_FRONTIER_EVIDENCE_LEVELS,
      path + ".level",
    ),
    name: asString(milestone.name, path + ".name", 128),
    outcomePoolBps: asInteger(
      milestone.outcomePoolBps,
      path + ".outcomePoolBps",
    ),
    evidence: asString(milestone.evidence, path + ".evidence", 2_048),
  };
}

function parseRewardTemplate(
  value: unknown,
  path: string,
): MathFrontierRewardTemplate {
  const reward = asObject(value, path);
  assertExactKeys(
    reward,
    [
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
    ],
    path,
  );
  for (const key of [
    "skillUnlockCreatesEntitlement",
    "breakthroughCreatesEntitlement",
    "protocolIssuanceAllowed",
    "specializedResearchSpendPathAllowed",
  ] as const) {
    asLiteral(reward[key], false, path + "." + key);
  }
  for (const key of [
    "dedicatedEscrowRequired",
    "policyFrozenBeforeAdmission",
    "singleSettlementRequired",
  ] as const) {
    asLiteral(reward[key], true, path + "." + key);
  }
  const milestones = asArray(reward.milestones, path + ".milestones", 5).map(
    (milestone, index) =>
      parseMilestone(milestone, path + ".milestones[" + index + "]"),
  );
  assertOrdered(
    milestones.map((milestone) => milestone.level),
    ["E2", "E3", "E4", "E5", "E6"],
    path + ".milestones",
  );
  const expectedMilestoneBps = [1_500, 2_000, 1_500, 2_500, 1_000];
  milestones.forEach((milestone, index) => {
    if (milestone.outcomePoolBps !== expectedMilestoneBps[index]) {
      fail(
        path + ".milestones[" + index + "].outcomePoolBps",
        "must match the reviewed prospective allocation",
      );
    }
  });
  const challengeReserveBps = asLiteral(
    asInteger(
      reward.challengeReserveBps,
      path + ".challengeReserveBps",
    ),
    1_500,
    path + ".challengeReserveBps",
  );
  const outcomeBps = milestones.reduce(
    (total, milestone) => total + milestone.outcomePoolBps,
    0,
  );
  if (outcomeBps + challengeReserveBps !== 10_000) {
    fail(path, "milestones plus challenge reserve must equal 10,000 bps");
  }
  const roles = asStringArray(reward.roles, path + ".roles", 16);
  assertOrdered(roles, REWARD_ROLES, path + ".roles");
  return {
    mode: asLiteral(
      reward.mode,
      "PROSPECTIVE_SPONSOR_ESCROW_ONLY",
      path + ".mode",
    ),
    economicEffect: asLiteral(
      reward.economicEffect,
      "NONE",
      path + ".economicEffect",
    ),
    liveAmount: asLiteral(reward.liveAmount, "0", path + ".liveAmount"),
    skillUnlockCreatesEntitlement: false,
    breakthroughCreatesEntitlement: false,
    protocolIssuanceAllowed: false,
    dedicatedEscrowRequired: true,
    policyFrozenBeforeAdmission: true,
    singleSettlementRequired: true,
    verifiedCostsTreatment: asLiteral(
      reward.verifiedCostsTreatment,
      "SEPARATE_PREAUTHORIZED_CAP",
      path + ".verifiedCostsTreatment",
    ),
    milestones,
    challengeReserveBps,
    roles,
    disproofDisposition: asLiteral(
      reward.disproofDisposition,
      "E4_TO_COMPLIANT_FALSIFIER_CLAIMANT_UNPAID",
      path + ".disproofDisposition",
    ),
    unallocatedDisposition: asLiteral(
      reward.unallocatedDisposition,
      "PROSPECTIVELY_NAMED_REFUND_OR_COMMONS",
      path + ".unallocatedDisposition",
    ),
    openLiabilityCap: asLiteral(
      reward.openLiabilityCap,
      "DEDICATED_ESCROW",
      path + ".openLiabilityCap",
    ),
    specializedResearchSpendPathAllowed: false,
  };
}

function parseNode(value: unknown, path: string): MathFrontierNode {
  const node = asObject(value, path);
  assertExactKeys(
    node,
    [
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
    ],
    path,
  );
  const prerequisites = asStringArray(
    node.prerequisites,
    path + ".prerequisites",
    16,
  ).map((id, index) =>
    asVersionedId(id, path + ".prerequisites[" + index + "]"),
  );
  assertLexicallySorted(prerequisites, path + ".prerequisites");
  const artifacts = asStringArray(node.artifacts, path + ".artifacts", 16);
  if (artifacts.length === 0) fail(path + ".artifacts", "must not be empty");
  return {
    id: asVersionedId(node.id, path + ".id"),
    title: asString(node.title, path + ".title", 512),
    stage: asEnum(node.stage, MATH_FRONTIER_STAGES, path + ".stage"),
    summary: asString(node.summary, path + ".summary", 2_048),
    prerequisites,
    evidenceTarget: asEnum(
      node.evidenceTarget,
      MATH_FRONTIER_EVIDENCE_LEVELS,
      path + ".evidenceTarget",
    ),
    capabilityClass: asEnum(
      node.capabilityClass,
      MATH_FRONTIER_CAPABILITY_CLASSES,
      path + ".capabilityClass",
    ),
    artifacts,
    unlocksReward: asLiteral(
      node.unlocksReward,
      false,
      path + ".unlocksReward",
    ),
    questEligible: asBoolean(node.questEligible, path + ".questEligible"),
  };
}

function validateGraph(
  nodes: MathFrontierNode[],
  importedCapabilities: readonly string[],
): void {
  const byId = new Map<string, MathFrontierNode>();
  for (const node of nodes) {
    if (byId.has(node.id)) fail("$.nodes", "duplicate node " + node.id);
    if (importedCapabilities.includes(node.id)) {
      fail("$.nodes", "extension node duplicates imported capability " + node.id);
    }
    byId.set(node.id, node);
  }
  for (const [index, node] of nodes.entries()) {
    for (const prerequisite of node.prerequisites) {
      if (!byId.has(prerequisite) && !importedCapabilities.includes(prerequisite)) {
        fail(
          "$.nodes[" + index + "].prerequisites",
          "missing capability " + prerequisite,
        );
      }
      if (prerequisite === node.id) {
        fail("$.nodes[" + index + "].prerequisites", "self dependency");
      }
    }
  }

  const state = new Map<string, "visiting" | "visited">();
  const visit = (id: string): void => {
    const current = state.get(id);
    if (current === "visiting") fail("$.nodes", "prerequisite cycle at " + id);
    if (current === "visited") return;
    const node = byId.get(id);
    if (!node) return;
    state.set(id, "visiting");
    node.prerequisites.forEach(visit);
    state.set(id, "visited");
  };
  nodes.forEach((node) => visit(node.id));

  const reachable = new Set(importedCapabilities);
  let changed = true;
  while (changed) {
    changed = false;
    for (const node of nodes) {
      if (
        !reachable.has(node.id) &&
        node.prerequisites.every((prerequisite) => reachable.has(prerequisite))
      ) {
        reachable.add(node.id);
        changed = true;
      }
    }
  }
  const unreachable = nodes.find((node) => !reachable.has(node.id));
  if (unreachable) {
    fail("$.nodes", "node is not reachable from the imported base tree: " + unreachable.id);
  }
}

function parseQuestTemplate(
  value: unknown,
  path: string,
  nodes: readonly MathFrontierNode[],
  baseCapabilities: readonly string[],
): MathFrontierQuestTemplate {
  const quest = asObject(value, path);
  assertExactKeys(
    quest,
    [
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
    ],
    path,
  );
  const id = asVersionedId(quest.id, path + ".id");
  const questNode = nodes.find((node) => node.id === id);
  if (
    !questNode ||
    questNode.stage !== "frontier" ||
    questNode.capabilityClass !== "quest"
  ) {
    fail(path + ".id", "must reference the frontier quest node");
  }
  const requiredCoreCapabilities = asStringArray(
    quest.requiredCoreCapabilities,
    path + ".requiredCoreCapabilities",
    16,
  );
  requiredCoreCapabilities.forEach((capability, index) => {
    if (!baseCapabilities.includes(capability)) {
      fail(
        path + ".requiredCoreCapabilities[" + index + "]",
        "must reference an imported base capability",
      );
    }
  });
  const allowedDomainCapabilities = asStringArray(
    quest.allowedDomainCapabilities,
    path + ".allowedDomainCapabilities",
    16,
  );
  assertLexicallySorted(
    allowedDomainCapabilities,
    path + ".allowedDomainCapabilities",
  );
  allowedDomainCapabilities.forEach((capability, index) => {
    const domainNode = nodes.find((node) => node.id === capability);
    if (!domainNode || domainNode.capabilityClass !== "domain") {
      fail(
        path + ".allowedDomainCapabilities[" + index + "]",
        "must reference a domain capability",
      );
    }
  });
  const domainSelectionMode = asLiteral(
    quest.domainSelectionMode,
    "EXACTLY_ONE_AT_PACKET_FREEZE",
    path + ".domainSelectionMode",
  );
  const selectedDomainEvidenceMinimum = asLiteral(
    quest.selectedDomainEvidenceMinimum,
    "E2",
    path + ".selectedDomainEvidenceMinimum",
  );
  asLiteral(
    quest.selectedDomainReceiptRequired,
    true,
    path + ".selectedDomainReceiptRequired",
  );
  const allowedArtifactRelations = asStringArray(
    quest.allowedArtifactRelations,
    path + ".allowedArtifactRelations",
    8,
  );
  assertOrdered(
    allowedArtifactRelations,
    ARTIFACT_RELATIONS,
    path + ".allowedArtifactRelations",
  );
  const requiredPacketBindings = asStringArray(
    quest.requiredPacketBindings,
    path + ".requiredPacketBindings",
    32,
  );
  assertOrdered(
    requiredPacketBindings,
    PACKET_BINDINGS,
    path + ".requiredPacketBindings",
  );
  const milestoneOrder = asArray(
    quest.milestoneOrder,
    path + ".milestoneOrder",
    7,
  ).map((level, index) =>
    asEnum(
      level,
      MATH_FRONTIER_EVIDENCE_LEVELS,
      path + ".milestoneOrder[" + index + "]",
    ),
  );
  assertOrdered(
    milestoneOrder,
    MATH_FRONTIER_EVIDENCE_LEVELS,
    path + ".milestoneOrder",
  );
  for (const key of [
    "selfDeclaredBreakthrough",
    "popularityCanOverrideValidity",
    "implementsEstablishesTheoremValidity",
    "semanticRootReplayAllowed",
  ] as const) {
    asLiteral(quest[key], false, path + "." + key);
  }
  for (const key of [
    "selectedDomainReceiptRequired",
    "relationSpecificValidityRequired",
    "counterexampleEligible",
    "sunsetRequired",
  ] as const) {
    asLiteral(quest[key], true, path + "." + key);
  }
  return {
    id,
    status: asLiteral(quest.status, "TEMPLATE_ONLY", path + ".status"),
    requiredCoreCapabilities,
    allowedDomainCapabilities,
    domainSelectionMode,
    selectedDomainEvidenceMinimum,
    selectedDomainReceiptRequired: true,
    allowedArtifactRelations,
    requiredPacketBindings,
    milestoneOrder,
    breakthroughStatus: asLiteral(
      quest.breakthroughStatus,
      "DERIVED_ONLY_AFTER_E5",
      path + ".breakthroughStatus",
    ),
    selfDeclaredBreakthrough: false,
    popularityCanOverrideValidity: false,
    relationSpecificValidityRequired: true,
    implementsEstablishesTheoremValidity: false,
    semanticRootReplayAllowed: false,
    minimumEffectiveClusters: asLiteral(
      asInteger(
        quest.minimumEffectiveClusters,
        path + ".minimumEffectiveClusters",
        1,
        64,
      ),
      3,
      path + ".minimumEffectiveClusters",
    ),
    minimumOrganizationRoots: asLiteral(
      asInteger(
        quest.minimumOrganizationRoots,
        path + ".minimumOrganizationRoots",
        1,
        64,
      ),
      2,
      path + ".minimumOrganizationRoots",
    ),
    minimumImplementationRoots: asLiteral(
      asInteger(
        quest.minimumImplementationRoots,
        path + ".minimumImplementationRoots",
        1,
        64,
      ),
      2,
      path + ".minimumImplementationRoots",
    ),
    minimumExecutionEnvironments: asLiteral(
      asInteger(
        quest.minimumExecutionEnvironments,
        path + ".minimumExecutionEnvironments",
        1,
        64,
      ),
      2,
      path + ".minimumExecutionEnvironments",
    ),
    minimumKernelFamilies: asLiteral(
      asInteger(
        quest.minimumKernelFamilies,
        path + ".minimumKernelFamilies",
        1,
        64,
      ),
      2,
      path + ".minimumKernelFamilies",
    ),
    counterexampleEligible: true,
    sunsetRequired: true,
    economicEffect: asLiteral(
      quest.economicEffect,
      "NONE",
      path + ".economicEffect",
    ),
    controlEffect: asLiteral(
      quest.controlEffect,
      "NONE",
      path + ".controlEffect",
    ),
  };
}

export function parseMathFrontier(value: unknown): MathFrontier {
  const frontier = asObject(value, "$");
  assertExactKeys(frontier, TOP_LEVEL_KEYS, "$");
  asLiteral(
    frontier.schema,
    "zerone.constructive-intelligence-math-frontier/v0",
    "$.schema",
  );
  for (const key of [
    "authoritative",
    "networkObserved",
    "rewardBearing",
    "governanceBearing",
  ] as const) {
    asLiteral(frontier[key], false, "$." + key);
  }
  const baseTree = parseBaseTree(frontier.baseTree, "$.baseTree");
  const stages = asArray(frontier.stages, "$.stages", 4).map((stage, index) =>
    asEnum(stage, MATH_FRONTIER_STAGES, "$.stages[" + index + "]"),
  );
  assertOrdered(stages, MATH_FRONTIER_STAGES, "$.stages");
  const nodes = asArray(frontier.nodes, "$.nodes", MAX_NODES).map(
    (node, index) => parseNode(node, "$.nodes[" + index + "]"),
  );
  if (nodes.length === 0) fail("$.nodes", "must not be empty");
  assertLexicallySorted(
    nodes.map((node) => node.id),
    "$.nodes",
  );
  validateGraph(nodes, baseTree.requiredCapabilities);
  const questTemplate = parseQuestTemplate(
    frontier.questTemplate,
    "$.questTemplate",
    nodes,
    baseTree.requiredCapabilities,
  );
  return {
    schema: "zerone.constructive-intelligence-math-frontier/v0",
    authoritative: false,
    networkObserved: false,
    rewardBearing: false,
    governanceBearing: false,
    snapshotDate: asLiteral(
      asIsoDate(frontier.snapshotDate, "$.snapshotDate"),
      MATH_FRONTIER_SNAPSHOT_DATE,
      "$.snapshotDate",
    ),
    policyVersion: asLiteral(
      frontier.policyVersion,
      MATH_FRONTIER_POLICY_VERSION,
      "$.policyVersion",
    ),
    baseTree,
    releaseBoundary: parseReleaseBoundary(
      frontier.releaseBoundary,
      "$.releaseBoundary",
    ),
    constitution: parseConstitution(frontier.constitution, "$.constitution"),
    karma: parseKarma(frontier.karma, "$.karma"),
    rewardTemplate: parseRewardTemplate(
      frontier.rewardTemplate,
      "$.rewardTemplate",
    ),
    stages,
    nodes,
    questTemplate,
  };
}

export function parseMathFrontierJson(raw: string): MathFrontier {
  if (new TextEncoder().encode(raw).byteLength > MATH_FRONTIER_MAX_BYTES) {
    fail("$", "document exceeds " + MATH_FRONTIER_MAX_BYTES + " UTF-8 bytes");
  }
  let value: unknown;
  try {
    value = JSON.parse(raw) as unknown;
  } catch {
    fail("$", "malformed JSON");
  }
  return parseMathFrontier(value);
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new MathFrontierDataError(
      "Math Frontier returned an empty response body",
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
        new DOMException("Math Frontier request timed out", "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const result = await Promise.race([reader.read(), aborted]);
      if (result.done) break;
      length += result.value.byteLength;
      if (length > MATH_FRONTIER_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal must not await a hostile stream cancellation.
        });
        throw new MathFrontierDataError(
          "Math Frontier exceeds its size limit",
        );
      }
      chunks.push(result.value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // The request deadline wins even if cancellation stalls.
      });
      throw new MathFrontierDataError("Math Frontier request timed out");
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
    throw new MathFrontierDataError(
      "Math Frontier digest verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

function assertCanonicalResponseUrl(
  response: Response,
  baseUrl?: string,
): void {
  if (response.redirected) {
    throw new MathFrontierDataError("Math Frontier response was redirected");
  }
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(MATH_FRONTIER_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new MathFrontierDataError(
      "Math Frontier returned an invalid final URL",
    );
  }
  if (
    actual.origin !== expected.origin ||
    actual.pathname !== expected.pathname ||
    actual.search !== expected.search ||
    actual.hash !== expected.hash
  ) {
    throw new MathFrontierDataError(
      "Math Frontier left its canonical same-origin path",
    );
  }
}

export async function fetchMathFrontier(
  options: MathFrontierFetchOptions = {},
): Promise<MathFrontier> {
  const fetcher = options.fetcher ?? fetch;
  const timeoutMs = options.timeoutMs ?? 8_000;
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(() => {
    controller.abort(
      new DOMException("Math Frontier request timed out", "TimeoutError"),
    );
  }, timeoutMs);
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? undefined : window.location.href);
  try {
    let response: Response;
    try {
      response = await fetcher(MATH_FRONTIER_ENDPOINT, {
        cache: "no-store",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        redirect: "error",
        signal: controller.signal,
      });
    } catch (error) {
      if (
        error instanceof DOMException &&
        (error.name === "AbortError" || error.name === "TimeoutError")
      ) {
        throw new MathFrontierDataError("Math Frontier request timed out");
      }
      throw new MathFrontierDataError("Math Frontier is unavailable");
    }
    if (!response.ok) {
      throw new MathFrontierDataError(
        "Math Frontier returned HTTP " + response.status,
      );
    }
    const contentType = response.headers.get("content-type");
    if (contentType !== null && !/\bjson\b/i.test(contentType)) {
      throw new MathFrontierDataError(
        "Math Frontier returned a non-JSON response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (
      declaredLength !== null &&
      (!/^\d+$/.test(declaredLength) ||
        Number(declaredLength) > MATH_FRONTIER_MAX_BYTES)
    ) {
      throw new MathFrontierDataError(
        "Math Frontier exceeded its size limit",
      );
    }
    assertCanonicalResponseUrl(response, baseUrl);
    const bytes = await readBoundedResponse(response, controller.signal);
    if ((await sha256Hex(bytes)) !== MATH_FRONTIER_SHA256) {
      throw new MathFrontierDataError(
        "Math Frontier did not match the reviewed canonical digest",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new MathFrontierDataError("Math Frontier was not valid UTF-8");
    }
    return parseMathFrontierJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

export function buildMathFrontierIndex(
  frontier: MathFrontier,
): MathFrontierIndex {
  const byId = new Map(frontier.nodes.map((node) => [node.id, node]));
  const dependents = new Map<string, MathFrontierNode[]>(
    frontier.nodes.map((node) => [node.id, []]),
  );
  for (const node of frontier.nodes) {
    for (const prerequisite of node.prerequisites) {
      dependents.get(prerequisite)?.push(node);
    }
  }
  for (const nodes of dependents.values()) {
    nodes.sort((left, right) => left.title.localeCompare(right.title, "en"));
  }
  return { byId, dependentsById: dependents };
}

export function filterMathFrontierNodes(
  frontier: MathFrontier,
  filters: MathFrontierFilters,
): MathFrontierNode[] {
  const query = filters.query.trim().toLocaleLowerCase("en");
  return frontier.nodes.filter((node) => {
    if (filters.stage !== "all" && node.stage !== filters.stage) return false;
    if (
      filters.capabilityClass !== "all" &&
      node.capabilityClass !== filters.capabilityClass
    ) {
      return false;
    }
    if (query.length === 0) return true;
    return [
      node.id,
      node.title,
      node.summary,
      node.stage,
      node.capabilityClass,
      node.evidenceTarget,
      ...node.artifacts,
    ]
      .join(" ")
      .toLocaleLowerCase("en")
      .includes(query);
  });
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
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatBasisPoints(basisPoints: number): string {
  const whole = Math.floor(basisPoints / 100);
  const remainder = basisPoints % 100;
  return remainder === 0
    ? whole + "%"
    : whole + "." + remainder.toString().padStart(2, "0").replace(/0$/, "") + "%";
}

function boundaryCard(
  label: string,
  value: string,
  copy: string,
  modifier?: string,
): HTMLElement {
  const card = element(
    "div",
    "math-frontier-boundary" + (modifier ? " " + modifier : ""),
  );
  card.append(
    element("span", undefined, label),
    element("strong", undefined, value),
    element("p", undefined, copy),
  );
  return card;
}

function renderBoundaryPanel(frontier: MathFrontier): HTMLElement {
  const wrap = element("div");
  const boundaries = element("div", "math-frontier-boundaries");
  boundaries.setAttribute(
    "aria-label",
    "Inactive economic, KARMA, control, and governance boundaries",
  );
  boundaries.append(
    boundaryCard(
      "Economics",
      frontier.rewardTemplate.liveAmount + " ZRN · inactive",
      "No escrow is funded. Skill unlocks and breakthrough labels create no entitlement.",
    ),
    boundaryCard(
      "KARMA",
      frontier.karma.state + " shadow only",
      "An ordinal observation is not recognition, truth, a score, or a ranking.",
      "is-ordinal",
    ),
    boundaryCard(
      "Control",
      frontier.karma.controlEffect + " · no vote",
      "KARMA magnitude, wealth, stake, and reward balances confer no voice.",
    ),
    boundaryCard(
      "Activation",
      frontier.constitution.currentActivationAuthority,
      "No ordinary stake vote or emergency authority can activate this template.",
    ),
  );
  const constitution = element("aside", "math-frontier-constitution");
  constitution.append(
    element(
      "strong",
      undefined,
      "One evidence rule. No reserved seat, share, creator privilege, operator privilege, or person ranking.",
    ),
    element(
      "p",
      undefined,
      "A future direction names KARMA eligibility plus controller-level sortition, never proportional voice. It is not implemented. Controller merges can only reduce voice; conflicts require recusal and quorum is recomputed.",
    ),
  );
  wrap.append(boundaries, constitution);
  return wrap;
}

function renderRewardTemplate(frontier: MathFrontier): HTMLElement {
  const section = element("section", "math-frontier-reward");
  section.setAttribute("aria-labelledby", "math-frontier-reward-title");
  const heading = element("div", "math-frontier-reward-head");
  const headingCopy = element("div");
  headingCopy.append(
    element("span", undefined, "Prospective sponsor-escrow shape"),
    element(
      "h4",
      undefined,
      "What independently checked breakthrough work could unlock later",
    ),
  );
  headingCopy.lastElementChild?.setAttribute(
    "id",
    "math-frontier-reward-title",
  );
  heading.append(
    headingCopy,
    element("strong", undefined, "0 ZRN · ECONOMIC EFFECT NONE"),
  );

  const milestones = element("ol", "math-frontier-milestones");
  for (const milestone of frontier.rewardTemplate.milestones) {
    const item = element("li");
    item.append(
      element("span", undefined, milestone.level),
      element("strong", undefined, humanise(milestone.name)),
      element(
        "b",
        undefined,
        formatBasisPoints(milestone.outcomePoolBps),
      ),
      element("p", undefined, milestone.evidence),
    );
    milestones.append(item);
  }
  const reserve = element("li", "is-reserve");
  reserve.append(
    element("span", undefined, "Reserve"),
    element("strong", undefined, "Challenge and remediation"),
    element(
      "b",
      undefined,
      formatBasisPoints(frontier.rewardTemplate.challengeReserveBps),
    ),
    element(
      "p",
      undefined,
      "A future funded packet would preserve this share for bounded challenge and repair.",
    ),
  );
  milestones.append(reserve);

  const truth = element("p", "math-frontier-reward-truth");
  truth.append(
    element(
      "strong",
      undefined,
      "Template proportions, not money:",
    ),
    document.createTextNode(
      " these percentages apply only inside a future, already-funded, prospectively frozen sponsor escrow. Protocol issuance is forbidden here; E0 is precedence-only and E1 can only use a separate preauthorised verified-cost cap. If the frozen claim is disproved, E4 belongs to the compliant falsifier and the failed claimant remains unpaid.",
    ),
  );
  section.append(heading, milestones, truth);
  return section;
}

function renderFacts(frontier: MathFrontier): HTMLElement {
  const facts = element("p", "math-frontier-facts");
  const edgeCount = frontier.nodes.reduce(
    (total, node) => total + node.prerequisites.length,
    0,
  );
  const entries = [
    ["Snapshot", frontier.snapshotDate],
    ["Policy", frontier.policyVersion],
    ["Capabilities", String(frontier.nodes.length)],
    ["Prerequisite edges", String(edgeCount)],
    ["Imported roots", String(frontier.baseTree.requiredCapabilities.length)],
    ["Base tree", frontier.baseTree.policyVersion],
  ];
  entries.forEach(([label, value], index) => {
    const item = element("span");
    item.append(
      document.createTextNode((label ?? "") + " "),
      element("strong", undefined, value ?? ""),
    );
    facts.append(item);
    if (index < entries.length - 1) {
      facts.append(document.createTextNode("·"));
    }
  });
  return facts;
}

function relationshipLabel(
  node: MathFrontierNode,
  selected: MathFrontierNode,
  index: MathFrontierIndex,
): string | null {
  if (node.id === selected.id) return "selected";
  if (selected.prerequisites.includes(node.id)) return "prerequisite";
  if (
    index.dependentsById
      .get(selected.id)
      ?.some((dependent) => dependent.id === node.id)
  ) {
    return "dependent";
  }
  return null;
}

function renderRelatedList(
  ids: readonly string[],
  index: MathFrontierIndex,
  emptyCopy: string,
  onSelect: (id: string) => void,
): HTMLElement {
  const list = element("ul");
  if (ids.length === 0) {
    list.append(element("li", undefined, emptyCopy));
    return list;
  }
  for (const id of ids) {
    const item = element("li");
    const related = index.byId.get(id);
    if (related) {
      const button = element(
        "button",
        "math-frontier-related",
        related.title,
      );
      button.type = "button";
      button.addEventListener("click", () => onSelect(id));
      item.append(button);
    } else {
      const code = element("code", undefined, id);
      item.append(code, document.createTextNode(" · imported base capability"));
    }
    list.append(item);
  }
  return list;
}

function renderNodeDetail(
  detail: HTMLElement,
  node: MathFrontierNode,
  frontier: MathFrontier,
  index: MathFrontierIndex,
  onSelect: (id: string) => void,
): void {
  const head = element("div", "math-frontier-detail-head");
  const titleWrap = element("div");
  titleWrap.append(
    element("span", "card-kicker", "Selected capability"),
    element("h4", undefined, node.title),
    element("code", undefined, node.id),
  );
  head.append(titleWrap);

  const badges = element("div", "math-frontier-badges");
  [
    humanise(node.stage),
    humanise(node.capabilityClass),
    node.evidenceTarget + " target",
    node.questEligible ? "future packet candidate" : "curriculum only",
  ].forEach((label) => badges.append(element("span", undefined, label)));

  const zero = element(
    "p",
    "math-frontier-zero",
    "This capability unlocks 0 ZRN. Its KARMA presentation is an ORDINAL shadow observation only: no recognition, magnitude, ranking, qualification, reward multiplier, or vote.",
  );

  const grid = element("div", "math-frontier-detail-grid");
  const artifacts = element("section", "math-frontier-detail-block");
  artifacts.append(element("h5", undefined, "Evidence artifacts"));
  const artifactList = element("ul");
  node.artifacts.forEach((artifact) =>
    artifactList.append(element("li", undefined, artifact)),
  );
  artifacts.append(artifactList);

  const prerequisites = element("section", "math-frontier-detail-block");
  prerequisites.append(
    element("h5", undefined, "Direct prerequisites"),
    renderRelatedList(
      node.prerequisites,
      index,
      "No prerequisite.",
      onSelect,
    ),
  );

  const dependents = element("section", "math-frontier-detail-block");
  dependents.append(
    element("h5", undefined, "Direct next steps"),
    renderRelatedList(
      (index.dependentsById.get(node.id) ?? []).map(
        (dependent) => dependent.id,
      ),
      index,
      "No extension capability depends on this node.",
      onSelect,
    ),
  );
  grid.append(artifacts, prerequisites, dependents);

  const breakthrough =
    node.id === frontier.questTemplate.id
      ? element(
          "p",
          "math-frontier-zero",
          "Exactly one domain path and its E2 attainment receipt must be frozen into the packet. Breakthrough is derived only after E5: relation-specific deterministic validity, independent reproduction, adversarial survival or honest disproof, and independently controlled downstream use. IMPLEMENTS alone does not establish theorem truth. The label cannot be self-declared, and popularity cannot override failed mathematics.",
        )
      : null;
  detail.replaceChildren(
    head,
    element("p", "math-frontier-detail-summary", node.summary),
    badges,
    zero,
    grid,
  );
  if (breakthrough) detail.append(breakthrough);
}

function option(value: string, label: string): HTMLOptionElement {
  const item = element("option", undefined, label);
  item.value = value;
  return item;
}

function renderExplorer(root: HTMLElement, frontier: MathFrontier): void {
  const explorer = element("div", "math-frontier-explorer");
  explorer.append(
    renderBoundaryPanel(frontier),
    renderRewardTemplate(frontier),
    renderFacts(frontier),
  );

  const controls = element("form", "math-frontier-controls");
  controls.setAttribute("role", "search");
  const queryLabel = element("label");
  queryLabel.append(element("span", undefined, "Search mathematics"));
  const queryInput = element("input");
  queryInput.type = "search";
  queryInput.placeholder = "proof, counterexample, kernel…";
  queryInput.autocomplete = "off";
  queryInput.setAttribute("aria-controls", "math-frontier-map");
  queryLabel.append(queryInput);

  const stageLabel = element("label");
  stageLabel.append(element("span", undefined, "Stage"));
  const stageSelect = element("select");
  stageSelect.setAttribute("aria-controls", "math-frontier-map");
  stageSelect.append(option("all", "All stages"));
  for (const stage of MATH_FRONTIER_STAGES) {
    stageSelect.append(option(stage, humanise(stage)));
  }
  stageLabel.append(stageSelect);

  const classLabel = element("label");
  classLabel.append(element("span", undefined, "Capability class"));
  const classSelect = element("select");
  classSelect.setAttribute("aria-controls", "math-frontier-map");
  classSelect.append(option("all", "All classes"));
  for (const capabilityClass of MATH_FRONTIER_CAPABILITY_CLASSES) {
    classSelect.append(option(capabilityClass, humanise(capabilityClass)));
  }
  classLabel.append(classSelect);

  const reset = element("button", "button button-ghost", "Reset");
  reset.type = "reset";
  controls.append(queryLabel, stageLabel, classLabel, reset);

  const resultsHead = element("div", "math-frontier-results-head");
  const resultCount = element("span");
  resultCount.setAttribute("aria-live", "polite");
  const raw = element("a", undefined, "Open raw JSON ↗");
  raw.href = MATH_FRONTIER_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  resultsHead.append(resultCount, raw);

  const map = element("div", "math-frontier-map");
  map.id = "math-frontier-map";
  map.setAttribute("aria-label", "Math Frontier capability graph");
  const detail = element("section", "math-frontier-detail");
  detail.tabIndex = -1;
  detail.setAttribute("aria-live", "polite");
  detail.setAttribute("aria-label", "Selected Math Frontier capability");

  explorer.append(controls, resultsHead, map, detail);
  root.replaceChildren(explorer);
  root.setAttribute("aria-busy", "false");

  const index = buildMathFrontierIndex(frontier);
  let selectedId = frontier.questTemplate.id;
  const filters: MathFrontierFilters = {
    query: "",
    stage: "all",
    capabilityClass: "all",
  };

  const selectNode = (id: string): void => {
    const node = index.byId.get(id);
    if (!node) return;
    selectedId = id;
    if (!filterMathFrontierNodes(frontier, filters).some((item) => item.id === id)) {
      filters.query = "";
      filters.stage = "all";
      filters.capabilityClass = "all";
      queryInput.value = "";
      stageSelect.value = "all";
      classSelect.value = "all";
    }
    update();
  };

  const update = (): void => {
    const selected =
      index.byId.get(selectedId) ??
      index.byId.get(frontier.questTemplate.id) ??
      frontier.nodes[0];
    if (!selected) return;
    const visible = filterMathFrontierNodes(frontier, filters);
    const visibleIds = new Set(visible.map((node) => node.id));
    resultCount.textContent =
      visible.length +
      " of " +
      frontier.nodes.length +
      " capabilities · selected " +
      selected.evidenceTarget;
    map.replaceChildren();

    for (const stage of MATH_FRONTIER_STAGES) {
      const section = element("section", "math-frontier-stage");
      const stageNodes = frontier.nodes.filter(
        (node) => node.stage === stage && visibleIds.has(node.id),
      );
      const heading = element("div", "math-frontier-stage-head");
      heading.append(
        element(
          "span",
          undefined,
          String(MATH_FRONTIER_STAGES.indexOf(stage) + 1).padStart(2, "0"),
        ),
        element("h4", undefined, humanise(stage)),
        element("small", undefined, String(stageNodes.length)),
      );
      section.append(heading);
      if (stageNodes.length === 0) {
        section.append(
          element(
            "p",
            "math-frontier-stage-empty",
            "No matching capabilities.",
          ),
        );
      } else {
        const list = element("ol", "math-frontier-nodes");
        for (const node of stageNodes) {
          const item = element("li");
          const button = element("button", "math-frontier-node");
          button.type = "button";
          button.setAttribute("aria-label", "Inspect " + node.title);
          const relationship = relationshipLabel(node, selected, index);
          if (relationship) button.dataset.relationship = relationship;
          button.append(
            element("strong", undefined, node.title),
            element(
              "span",
              "math-frontier-node-meta",
              node.capabilityClass + " · " + node.evidenceTarget,
            ),
            element(
              "span",
              "math-frontier-node-karma",
              "KARMA mode · ordinal shadow",
            ),
          );
          if (relationship) {
            button.append(
              element(
                "span",
                "math-frontier-node-relation",
                relationship,
              ),
            );
          }
          button.addEventListener("click", () => selectNode(node.id));
          item.append(button);
          list.append(item);
        }
        section.append(list);
      }
      map.append(section);
    }
    renderNodeDetail(detail, selected, frontier, index, selectNode);
  };

  queryInput.addEventListener("input", () => {
    filters.query = queryInput.value;
    update();
  });
  stageSelect.addEventListener("change", () => {
    filters.stage = asEnum(
      stageSelect.value,
      ["all", ...MATH_FRONTIER_STAGES] as const,
      "stage filter",
    );
    update();
  });
  classSelect.addEventListener("change", () => {
    filters.capabilityClass = asEnum(
      classSelect.value,
      ["all", ...MATH_FRONTIER_CAPABILITY_CLASSES] as const,
      "capability-class filter",
    );
    update();
  });
  controls.addEventListener("reset", () => {
    filters.query = "";
    filters.stage = "all";
    filters.capabilityClass = "all";
    globalThis.setTimeout(update, 0);
  });
  controls.addEventListener("submit", (event) => event.preventDefault());
  update();
}

export async function initialiseMathFrontier(
  root: HTMLElement,
  options: MathFrontierFetchOptions = {},
): Promise<void> {
  const load = async (): Promise<void> => {
    root.setAttribute("aria-busy", "true");
    try {
      const frontier = await fetchMathFrontier(options);
      renderExplorer(root, frontier);
      if (window.location.hash === "#math-frontier") {
        requestAnimationFrame(() => {
          root.closest<HTMLElement>("#math-frontier")?.scrollIntoView({
            block: "start",
            behavior: "instant",
          });
        });
      }
    } catch (error) {
      root.setAttribute("aria-busy", "false");
      const state = element("div", "math-frontier-load-error");
      state.setAttribute("role", "alert");
      state.append(
        element("strong", undefined, "The Math Frontier could not be loaded."),
        element(
          "p",
          undefined,
          error instanceof Error
            ? error.message
            : "The response was unavailable or invalid.",
        ),
      );
      const actions = element("div", "math-frontier-load-actions");
      const retry = element("button", "button button-primary", "Try again");
      retry.type = "button";
      retry.addEventListener("click", () => {
        retry.disabled = true;
        retry.textContent = "Trying again…";
        void load();
      });
      const source = element("a", "button button-ghost", "Open raw JSON");
      source.href = MATH_FRONTIER_ENDPOINT;
      source.target = "_blank";
      source.rel = "noreferrer";
      actions.append(retry, source);
      state.append(actions);
      root.replaceChildren(state);
    }
  };
  await load();
}
