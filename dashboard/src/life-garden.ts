/// <reference lib="dom" />

export const EPIGENETICS_GARDEN_ENDPOINT =
  "/standards/epigenetics-capability-garden.v1.json";
export const KARMA_FOUNDATION_ENDPOINT =
  "/standards/karma-foundation.v1.json";
export const STATIC_STANDARD_MAX_BYTES = 262_144;
export const EPIGENETICS_GARDEN_SHA256 =
  "7d04efe9da46309bf97b850c9b80324b1a5c4035edb1008b9ba3ad0df2bcfa63";
export const KARMA_FOUNDATION_SHA256 =
  "b46710704869dcc340ded356be72b4ec692f204710fedfb5cd43eb3757dc7b80";

export const LIFE_GARDEN_STAGES = [
  "soil",
  "trunk",
  "measurement",
  "inference",
  "intervention",
  "canopy",
  "quest",
] as const;
export const LIFE_GARDEN_DOMAINS = [
  "integrity",
  "foundations",
  "methylation",
  "chromatin",
  "transcriptomics",
  "single-cell",
  "computation",
  "causality",
  "intervention",
  "quests",
] as const;
export const LIFE_GARDEN_KINDS = [
  "foundation",
  "protocol",
  "analysis",
  "intervention",
  "validation",
  "quest",
] as const;
export const LIFE_GARDEN_EVIDENCE_LEVELS = [
  "E0",
  "E1",
  "E2",
  "E3",
  "E4",
  "E5",
  "E6",
] as const;
export const LIFE_GARDEN_EVIDENCE_CONTRIBUTIONS = [
  ...LIFE_GARDEN_EVIDENCE_LEVELS,
  "cross-cutting",
] as const;
export const LIFE_GARDEN_REWARD_ELIGIBILITY = [
  "capability-evidence-only",
  "sponsor-template-only",
] as const;
export const LIFE_GARDEN_SAFETY_TIERS = [
  "open-computational",
  "controlled-biological",
  "regulated-human",
] as const;

export type LifeGardenStage = (typeof LIFE_GARDEN_STAGES)[number];
export type LifeGardenDomain = (typeof LIFE_GARDEN_DOMAINS)[number];
export type LifeGardenKind = (typeof LIFE_GARDEN_KINDS)[number];
export type LifeGardenEvidence =
  (typeof LIFE_GARDEN_EVIDENCE_LEVELS)[number];
export type LifeGardenEvidenceContribution =
  (typeof LIFE_GARDEN_EVIDENCE_CONTRIBUTIONS)[number];
export type LifeGardenRewardEligibility =
  (typeof LIFE_GARDEN_REWARD_ELIGIBILITY)[number];
export type LifeGardenSafetyTier =
  (typeof LIFE_GARDEN_SAFETY_TIERS)[number];

interface LifeGardenReleaseBoundary {
  addsConsensusBehavior: false;
  activatesRewards: false;
  movesFunds: false;
  grantsQualification: false;
  authorizesBiologicalExperimentation: false;
  authorizesHumanIntervention: false;
  publishesIdentifiableGenomicData: false;
  assertsClinicalValidity: false;
}

export interface LifeGardenEvidenceStep {
  level: LifeGardenEvidence;
  name: string;
  rewardBps: number;
  meaning: string;
}

interface LifeGardenPolicy {
  breakthroughRecognition: {
    authorSelected: false;
    minimumEvidenceLevel: "E5";
    requiresPriorArtDelta: true;
    requiresIndependentReproduction: true;
    requiresProspectiveOrDescendantImpact: true;
    noveltyAloneIsSufficient: false;
    poweredNullResultsRemainEvidence: true;
  };
  funding: {
    skillUnlockCreatesReward: false;
    candidateSource: "voluntary-external-sponsor-escrow-only";
    protocolIssuance: "disabled";
    templateStatus: "simulation-only";
    timeAloneUnlocksEvidence: false;
    challengeReserveBps: 1500;
    sponsorCaseSchema: "zerone.breakthrough-sponsor-case/v1-required";
    dashboardCanActivateCase: false;
    frozenClaimAndArtifactDigestsRequired: true;
    milestoneAcceptanceTestsRequired: true;
    nullFailureContradictionTreatmentRequired: true;
    reviewerConflictPolicyRequired: true;
    correctionRecoveryPolicyRequired: true;
    expiryRefundPolicyRequired: true;
    escrowFundingReceiptRequired: true;
  };
  independence: {
    minimumEffectiveClusters: 3;
    minimumOrganizationRoots: 2;
    minimumDataRoots: 2;
    minimumAnalysisPipelineRoots: 2;
    assignmentAfterArtifactFreeze: true;
    reviewPayOutcomeIndependent: true;
    rawAddressCountIsEvidence: false;
  };
  humanData: {
    rawIdentifiableDataOnPublicLedger: false;
    consentOrDataUseScopeRequired: true;
    accessLaneAttestationRequired: true;
    controlledAccessRequiredWhereConsentPolicyOrLawRequires: true;
    unrestrictedAccessRequiresConsentPolicyAndLawAllow: true;
    institutionalCertificationRequiredWhereApplicable: true;
    publicArtifactsUseMetadataAndDigestsOnly: true;
  };
  safety: {
    animalWelfareReviewRequiredWhereApplicable: true;
    biosafetyRiskAssessmentRequired: true;
    ethicalReviewIsHardGate: true;
    unapprovedHumanInterventionEligible: false;
    heritableHumanGenomeInterventionEligible: false;
    clinicalClaimsRequireRegulatoryEvidence: true;
    unknownHarmEscalatesTo: "regulated-human";
  };
}

export interface LifeGardenSource {
  id: string;
  authority: string;
  title: string;
  url: string;
  kind: "official-guidance" | "official-policy" | "primary-research";
  supportScope: string;
  checkedAt: string;
  reviewAfter: string;
}

export interface LifeGardenGroupedPrerequisite {
  nodeId: string;
  allOf: string[];
  atLeast: {
    count: number;
    of: string[];
  };
}

export interface LifeGardenPrerequisiteSemantics {
  default: "all-listed-required";
  grouped: LifeGardenGroupedPrerequisite[];
}

export interface LifeGardenQuestMilestone {
  id: string;
  evidenceLevel: LifeGardenEvidence;
  acceptanceTest: string;
}

export interface LifeGardenQuestAcceptance {
  targetEvidence: "E5";
  scopeBounds: string[];
  scopeHash: string;
  milestones: LifeGardenQuestMilestone[];
  minimumEffectiveClusters: 3;
  minimumOrganizationRoots: 2;
  minimumDataRoots: 2;
  minimumAnalysisPipelineRoots: 2;
  requiredReceiptTypes: string[];
  nullFailureContradictionRule:
    | "preserve-and-report-no-scope-mutation-no-automatic-reward";
  safetyApprovalSubstitutionAllowed: false;
}

export interface LifeGardenNode {
  id: string;
  title: string;
  stage: LifeGardenStage;
  domain: LifeGardenDomain;
  kind: LifeGardenKind;
  summary: string;
  prerequisites: string[];
  evidenceContribution: LifeGardenEvidenceContribution;
  rewardEligibility: LifeGardenRewardEligibility;
  safetyTier: LifeGardenSafetyTier;
  artifactRequirements: string[];
  failureModes: string[];
  sourceIds: string[];
  acceptance: LifeGardenQuestAcceptance | null;
}

export interface EpigeneticsCapabilityGarden {
  schema: "zerone.epigenetics-capability-garden/v1";
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  snapshotDate: string;
  policyVersion: string;
  releaseBoundary: LifeGardenReleaseBoundary;
  evidenceLadder: LifeGardenEvidenceStep[];
  policy: LifeGardenPolicy;
  prerequisiteSemantics: LifeGardenPrerequisiteSemantics;
  sources: LifeGardenSource[];
  roots: string[];
  nodes: LifeGardenNode[];
}

export interface KarmaState {
  name: "RECOGNIZED" | "ORDINAL";
  summable: false;
  meaning: string;
}

export interface KarmaEventKind {
  id:
    | "cited"
    | "corroborate"
    | "corroborated"
    | "external"
    | "pending_open"
    | "pending_settle"
    | "verify";
  state: "RECOGNIZED" | "ORDINAL";
  meaning: string;
}

export interface KarmaInvariant {
  id: string;
  statement: string;
}

export interface KarmaGovernanceGate {
  id: string;
  passed: false;
  requirement: string;
}

export interface KarmaFoundation {
  schema: "zerone.karma-foundation/v1";
  authoritative: false;
  networkObserved: false;
  economicBearing: false;
  governanceBearing: false;
  transferable: false;
  purchasable: false;
  delegable: false;
  founderPrivilege: false;
  operatorPrivilege: false;
  scalarRank: false;
  snapshotDate: string;
  status: "design-only";
  purpose: string;
  eventRegister: "priced-coherence";
  releaseBoundary: {
    addsConsensusBehavior: false;
    activatesRewards: false;
    movesFunds: false;
    grantsAuthority: false;
    grantsQualification: false;
    modifiesKarmaEvents: false;
    derivesPersonScores: false;
    changesGovernanceWeight: false;
  };
  economicCovenant: {
    founderShare: false;
    founderControl: false;
    operatorShare: false;
    operatorControl: false;
    creatorRoyalty: false;
    structurallyEnforced: false;
    currentStatus: "declarative-until-independent-enforcement";
    residualValueDestination: string;
  };
  states: KarmaState[];
  eventVocabulary: KarmaEventKind[];
  invariants: KarmaInvariant[];
  prohibitedUses: string[];
  futureGovernanceGates: KarmaGovernanceGate[];
}

export interface LifeGardenFilters {
  query: string;
  stage: LifeGardenStage | "all";
  domain: LifeGardenDomain | "all";
  safetyTier: LifeGardenSafetyTier | "all";
}

export interface LifeGardenSourceReview {
  checkedThrough: string;
  reviewAfter: string;
  due: boolean;
}

export interface StaticStandardFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class LifeGardenDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "LifeGardenDataError";
  }
}

type JsonObject = Record<string, unknown>;

const GARDEN_TOP_LEVEL_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "snapshotDate",
  "policyVersion",
  "releaseBoundary",
  "evidenceLadder",
  "policy",
  "prerequisiteSemantics",
  "sources",
  "roots",
  "nodes",
] as const;
const GARDEN_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesBiologicalExperimentation",
  "authorizesHumanIntervention",
  "publishesIdentifiableGenomicData",
  "assertsClinicalValidity",
] as const;
const KARMA_TOP_LEVEL_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "economicBearing",
  "governanceBearing",
  "transferable",
  "purchasable",
  "delegable",
  "founderPrivilege",
  "operatorPrivilege",
  "scalarRank",
  "snapshotDate",
  "status",
  "purpose",
  "eventRegister",
  "releaseBoundary",
  "economicCovenant",
  "states",
  "eventVocabulary",
  "invariants",
  "prohibitedUses",
  "futureGovernanceGates",
] as const;
const KARMA_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsAuthority",
  "grantsQualification",
  "modifiesKarmaEvents",
  "derivesPersonScores",
  "changesGovernanceWeight",
] as const;
const MAX_ARRAY_ITEMS = 128;
const MAX_STRING_LENGTH = 8_192;
const MAX_NODES = 64;
const NODE_ID_PATTERN =
  /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const SOURCE_ID_PATTERN =
  /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:-[a-z0-9]+)*$/;

function fail(path: string, message: string): never {
  throw new LifeGardenDataError(`${path}: ${message}`);
}

function asObject(value: unknown, path: string): JsonObject {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value as JsonObject;
}

function exactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (
    actual.length !== wanted.length ||
    actual.some((key, index) => key !== wanted[index])
  ) {
    fail(path, `expected exactly: ${wanted.join(", ")}`);
  }
}

function asArray(
  value: unknown,
  path: string,
  maximum = MAX_ARRAY_ITEMS,
): unknown[] {
  if (!Array.isArray(value)) fail(path, "expected an array");
  if (value.length > maximum) fail(path, `must contain at most ${maximum} items`);
  return value;
}

function asString(
  value: unknown,
  path: string,
  maximum = MAX_STRING_LENGTH,
): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, `expected a non-empty string of at most ${maximum} characters`);
  }
  return value;
}

function asSafeInteger(value: unknown, path: string): number {
  if (!Number.isSafeInteger(value)) fail(path, "expected a safe integer");
  return value as number;
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

function requireFalse(value: unknown, path: string): false {
  if (value !== false) fail(path, "must remain false");
  return false;
}

function requireTrue(value: unknown, path: string): true {
  if (value !== true) fail(path, "must remain true");
  return true;
}

function requireLiteral<T extends string | number>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${JSON.stringify(expected)}`);
  return expected;
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
  const result = asArray(value, path, maximum).map((item, index) =>
    asString(item, `${path}[${index}]`),
  );
  if (new Set(result).size !== result.length) fail(path, "contains a duplicate");
  return result;
}

function requireSorted(values: readonly string[], path: string): void {
  const sorted = [...values].sort((left, right) => left.localeCompare(right, "en"));
  if (values.some((value, index) => value !== sorted[index])) {
    fail(path, "must be sorted lexicographically");
  }
}

function asHttpsUrl(value: unknown, path: string): string {
  const result = asString(value, path, 2_048);
  let url: URL;
  try {
    url = new URL(result);
  } catch {
    fail(path, "expected a valid HTTPS URL");
  }
  if (
    url.protocol !== "https:" ||
    url.username !== "" ||
    url.password !== "" ||
    url.hash !== ""
  ) {
    fail(path, "expected a credential-free HTTPS URL without a fragment");
  }
  return result;
}

function parseEvidenceStep(
  value: unknown,
  path: string,
  expectedIndex: number,
): LifeGardenEvidenceStep {
  const source = asObject(value, path);
  exactKeys(source, ["level", "name", "rewardBps", "meaning"], path);
  const level = LIFE_GARDEN_EVIDENCE_LEVELS[expectedIndex];
  const rewardBps = [0, 0, 1500, 2500, 2500, 1500, 500][expectedIndex];
  if (level === undefined || rewardBps === undefined) fail(path, "unexpected step");
  return {
    level: requireLiteral(source.level, level, `${path}.level`),
    name: asString(source.name, `${path}.name`, 64),
    rewardBps: requireLiteral(source.rewardBps, rewardBps, `${path}.rewardBps`),
    meaning: asString(source.meaning, `${path}.meaning`),
  };
}

function parsePolicy(value: unknown, path: string): LifeGardenPolicy {
  const source = asObject(value, path);
  exactKeys(
    source,
    ["breakthroughRecognition", "funding", "independence", "humanData", "safety"],
    path,
  );
  const breakthrough = asObject(
    source.breakthroughRecognition,
    `${path}.breakthroughRecognition`,
  );
  exactKeys(
    breakthrough,
    [
      "authorSelected",
      "minimumEvidenceLevel",
      "requiresPriorArtDelta",
      "requiresIndependentReproduction",
      "requiresProspectiveOrDescendantImpact",
      "noveltyAloneIsSufficient",
      "poweredNullResultsRemainEvidence",
    ],
    `${path}.breakthroughRecognition`,
  );
  const funding = asObject(source.funding, `${path}.funding`);
  exactKeys(
    funding,
    [
      "skillUnlockCreatesReward",
      "candidateSource",
      "protocolIssuance",
      "templateStatus",
      "timeAloneUnlocksEvidence",
      "challengeReserveBps",
      "sponsorCaseSchema",
      "dashboardCanActivateCase",
      "frozenClaimAndArtifactDigestsRequired",
      "milestoneAcceptanceTestsRequired",
      "nullFailureContradictionTreatmentRequired",
      "reviewerConflictPolicyRequired",
      "correctionRecoveryPolicyRequired",
      "expiryRefundPolicyRequired",
      "escrowFundingReceiptRequired",
    ],
    `${path}.funding`,
  );
  const independence = asObject(source.independence, `${path}.independence`);
  exactKeys(
    independence,
    [
      "minimumEffectiveClusters",
      "minimumOrganizationRoots",
      "minimumDataRoots",
      "minimumAnalysisPipelineRoots",
      "assignmentAfterArtifactFreeze",
      "reviewPayOutcomeIndependent",
      "rawAddressCountIsEvidence",
    ],
    `${path}.independence`,
  );
  const humanData = asObject(source.humanData, `${path}.humanData`);
  exactKeys(
    humanData,
    [
      "rawIdentifiableDataOnPublicLedger",
      "consentOrDataUseScopeRequired",
      "accessLaneAttestationRequired",
      "controlledAccessRequiredWhereConsentPolicyOrLawRequires",
      "unrestrictedAccessRequiresConsentPolicyAndLawAllow",
      "institutionalCertificationRequiredWhereApplicable",
      "publicArtifactsUseMetadataAndDigestsOnly",
    ],
    `${path}.humanData`,
  );
  const safety = asObject(source.safety, `${path}.safety`);
  exactKeys(
    safety,
    [
      "animalWelfareReviewRequiredWhereApplicable",
      "biosafetyRiskAssessmentRequired",
      "ethicalReviewIsHardGate",
      "unapprovedHumanInterventionEligible",
      "heritableHumanGenomeInterventionEligible",
      "clinicalClaimsRequireRegulatoryEvidence",
      "unknownHarmEscalatesTo",
    ],
    `${path}.safety`,
  );

  return {
    breakthroughRecognition: {
      authorSelected: requireFalse(
        breakthrough.authorSelected,
        `${path}.breakthroughRecognition.authorSelected`,
      ),
      minimumEvidenceLevel: requireLiteral(
        breakthrough.minimumEvidenceLevel,
        "E5",
        `${path}.breakthroughRecognition.minimumEvidenceLevel`,
      ),
      requiresPriorArtDelta: requireTrue(
        breakthrough.requiresPriorArtDelta,
        `${path}.breakthroughRecognition.requiresPriorArtDelta`,
      ),
      requiresIndependentReproduction: requireTrue(
        breakthrough.requiresIndependentReproduction,
        `${path}.breakthroughRecognition.requiresIndependentReproduction`,
      ),
      requiresProspectiveOrDescendantImpact: requireTrue(
        breakthrough.requiresProspectiveOrDescendantImpact,
        `${path}.breakthroughRecognition.requiresProspectiveOrDescendantImpact`,
      ),
      noveltyAloneIsSufficient: requireFalse(
        breakthrough.noveltyAloneIsSufficient,
        `${path}.breakthroughRecognition.noveltyAloneIsSufficient`,
      ),
      poweredNullResultsRemainEvidence: requireTrue(
        breakthrough.poweredNullResultsRemainEvidence,
        `${path}.breakthroughRecognition.poweredNullResultsRemainEvidence`,
      ),
    },
    funding: {
      skillUnlockCreatesReward: requireFalse(
        funding.skillUnlockCreatesReward,
        `${path}.funding.skillUnlockCreatesReward`,
      ),
      candidateSource: requireLiteral(
        funding.candidateSource,
        "voluntary-external-sponsor-escrow-only",
        `${path}.funding.candidateSource`,
      ),
      protocolIssuance: requireLiteral(
        funding.protocolIssuance,
        "disabled",
        `${path}.funding.protocolIssuance`,
      ),
      templateStatus: requireLiteral(
        funding.templateStatus,
        "simulation-only",
        `${path}.funding.templateStatus`,
      ),
      timeAloneUnlocksEvidence: requireFalse(
        funding.timeAloneUnlocksEvidence,
        `${path}.funding.timeAloneUnlocksEvidence`,
      ),
      challengeReserveBps: requireLiteral(
        funding.challengeReserveBps,
        1500,
        `${path}.funding.challengeReserveBps`,
      ),
      sponsorCaseSchema: requireLiteral(
        funding.sponsorCaseSchema,
        "zerone.breakthrough-sponsor-case/v1-required",
        `${path}.funding.sponsorCaseSchema`,
      ),
      dashboardCanActivateCase: requireFalse(
        funding.dashboardCanActivateCase,
        `${path}.funding.dashboardCanActivateCase`,
      ),
      frozenClaimAndArtifactDigestsRequired: requireTrue(
        funding.frozenClaimAndArtifactDigestsRequired,
        `${path}.funding.frozenClaimAndArtifactDigestsRequired`,
      ),
      milestoneAcceptanceTestsRequired: requireTrue(
        funding.milestoneAcceptanceTestsRequired,
        `${path}.funding.milestoneAcceptanceTestsRequired`,
      ),
      nullFailureContradictionTreatmentRequired: requireTrue(
        funding.nullFailureContradictionTreatmentRequired,
        `${path}.funding.nullFailureContradictionTreatmentRequired`,
      ),
      reviewerConflictPolicyRequired: requireTrue(
        funding.reviewerConflictPolicyRequired,
        `${path}.funding.reviewerConflictPolicyRequired`,
      ),
      correctionRecoveryPolicyRequired: requireTrue(
        funding.correctionRecoveryPolicyRequired,
        `${path}.funding.correctionRecoveryPolicyRequired`,
      ),
      expiryRefundPolicyRequired: requireTrue(
        funding.expiryRefundPolicyRequired,
        `${path}.funding.expiryRefundPolicyRequired`,
      ),
      escrowFundingReceiptRequired: requireTrue(
        funding.escrowFundingReceiptRequired,
        `${path}.funding.escrowFundingReceiptRequired`,
      ),
    },
    independence: {
      minimumEffectiveClusters: requireLiteral(
        independence.minimumEffectiveClusters,
        3,
        `${path}.independence.minimumEffectiveClusters`,
      ),
      minimumOrganizationRoots: requireLiteral(
        independence.minimumOrganizationRoots,
        2,
        `${path}.independence.minimumOrganizationRoots`,
      ),
      minimumDataRoots: requireLiteral(
        independence.minimumDataRoots,
        2,
        `${path}.independence.minimumDataRoots`,
      ),
      minimumAnalysisPipelineRoots: requireLiteral(
        independence.minimumAnalysisPipelineRoots,
        2,
        `${path}.independence.minimumAnalysisPipelineRoots`,
      ),
      assignmentAfterArtifactFreeze: requireTrue(
        independence.assignmentAfterArtifactFreeze,
        `${path}.independence.assignmentAfterArtifactFreeze`,
      ),
      reviewPayOutcomeIndependent: requireTrue(
        independence.reviewPayOutcomeIndependent,
        `${path}.independence.reviewPayOutcomeIndependent`,
      ),
      rawAddressCountIsEvidence: requireFalse(
        independence.rawAddressCountIsEvidence,
        `${path}.independence.rawAddressCountIsEvidence`,
      ),
    },
    humanData: {
      rawIdentifiableDataOnPublicLedger: requireFalse(
        humanData.rawIdentifiableDataOnPublicLedger,
        `${path}.humanData.rawIdentifiableDataOnPublicLedger`,
      ),
      consentOrDataUseScopeRequired: requireTrue(
        humanData.consentOrDataUseScopeRequired,
        `${path}.humanData.consentOrDataUseScopeRequired`,
      ),
      accessLaneAttestationRequired: requireTrue(
        humanData.accessLaneAttestationRequired,
        `${path}.humanData.accessLaneAttestationRequired`,
      ),
      controlledAccessRequiredWhereConsentPolicyOrLawRequires: requireTrue(
        humanData.controlledAccessRequiredWhereConsentPolicyOrLawRequires,
        `${path}.humanData.controlledAccessRequiredWhereConsentPolicyOrLawRequires`,
      ),
      unrestrictedAccessRequiresConsentPolicyAndLawAllow: requireTrue(
        humanData.unrestrictedAccessRequiresConsentPolicyAndLawAllow,
        `${path}.humanData.unrestrictedAccessRequiresConsentPolicyAndLawAllow`,
      ),
      institutionalCertificationRequiredWhereApplicable: requireTrue(
        humanData.institutionalCertificationRequiredWhereApplicable,
        `${path}.humanData.institutionalCertificationRequiredWhereApplicable`,
      ),
      publicArtifactsUseMetadataAndDigestsOnly: requireTrue(
        humanData.publicArtifactsUseMetadataAndDigestsOnly,
        `${path}.humanData.publicArtifactsUseMetadataAndDigestsOnly`,
      ),
    },
    safety: {
      animalWelfareReviewRequiredWhereApplicable: requireTrue(
        safety.animalWelfareReviewRequiredWhereApplicable,
        `${path}.safety.animalWelfareReviewRequiredWhereApplicable`,
      ),
      biosafetyRiskAssessmentRequired: requireTrue(
        safety.biosafetyRiskAssessmentRequired,
        `${path}.safety.biosafetyRiskAssessmentRequired`,
      ),
      ethicalReviewIsHardGate: requireTrue(
        safety.ethicalReviewIsHardGate,
        `${path}.safety.ethicalReviewIsHardGate`,
      ),
      unapprovedHumanInterventionEligible: requireFalse(
        safety.unapprovedHumanInterventionEligible,
        `${path}.safety.unapprovedHumanInterventionEligible`,
      ),
      heritableHumanGenomeInterventionEligible: requireFalse(
        safety.heritableHumanGenomeInterventionEligible,
        `${path}.safety.heritableHumanGenomeInterventionEligible`,
      ),
      clinicalClaimsRequireRegulatoryEvidence: requireTrue(
        safety.clinicalClaimsRequireRegulatoryEvidence,
        `${path}.safety.clinicalClaimsRequireRegulatoryEvidence`,
      ),
      unknownHarmEscalatesTo: requireLiteral(
        safety.unknownHarmEscalatesTo,
        "regulated-human",
        `${path}.safety.unknownHarmEscalatesTo`,
      ),
    },
  };
}

function parseSource(value: unknown, path: string): LifeGardenSource {
  const source = asObject(value, path);
  exactKeys(
    source,
    [
      "id",
      "authority",
      "title",
      "url",
      "kind",
      "supportScope",
      "checkedAt",
      "reviewAfter",
    ],
    path,
  );
  const id = asString(source.id, `${path}.id`, 128);
  if (!SOURCE_ID_PATTERN.test(id)) fail(`${path}.id`, "invalid source identifier");
  const checkedAt = asIsoDate(source.checkedAt, `${path}.checkedAt`);
  const reviewAfter = asIsoDate(source.reviewAfter, `${path}.reviewAfter`);
  if (reviewAfter < checkedAt) fail(`${path}.reviewAfter`, "precedes checkedAt");
  return {
    id,
    authority: asString(source.authority, `${path}.authority`, 256),
    title: asString(source.title, `${path}.title`, 512),
    url: asHttpsUrl(source.url, `${path}.url`),
    kind: asEnum(
      source.kind,
      ["official-guidance", "official-policy", "primary-research"] as const,
      `${path}.kind`,
    ),
    supportScope: asString(source.supportScope, `${path}.supportScope`, 1_024),
    checkedAt,
    reviewAfter,
  };
}

function parseQuestMilestone(
  value: unknown,
  path: string,
): LifeGardenQuestMilestone {
  const source = asObject(value, path);
  exactKeys(source, ["id", "evidenceLevel", "acceptanceTest"], path);
  const id = asString(source.id, `${path}.id`, 96);
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(id)) {
    fail(`${path}.id`, "invalid milestone identifier");
  }
  return {
    id,
    evidenceLevel: asEnum(
      source.evidenceLevel,
      LIFE_GARDEN_EVIDENCE_LEVELS,
      `${path}.evidenceLevel`,
    ),
    acceptanceTest: asString(
      source.acceptanceTest,
      `${path}.acceptanceTest`,
      1_024,
    ),
  };
}

const REQUIRED_QUEST_RECEIPTS = [
  "causal-challenge",
  "independent-reproduction",
  "prior-art-delta",
  "prospective-or-descendant-impact",
  "safety-lane",
] as const;

function parseQuestAcceptance(
  value: unknown,
  path: string,
): LifeGardenQuestAcceptance {
  const source = asObject(value, path);
  exactKeys(
    source,
    [
      "targetEvidence",
      "scopeBounds",
      "scopeHash",
      "milestones",
      "minimumEffectiveClusters",
      "minimumOrganizationRoots",
      "minimumDataRoots",
      "minimumAnalysisPipelineRoots",
      "requiredReceiptTypes",
      "nullFailureContradictionRule",
      "safetyApprovalSubstitutionAllowed",
    ],
    path,
  );
  const scopeBounds = asStringArray(source.scopeBounds, `${path}.scopeBounds`, 24);
  if (scopeBounds.length < 5) fail(`${path}.scopeBounds`, "requires at least five bounds");
  requireSorted(scopeBounds, `${path}.scopeBounds`);
  const scopeHash = asString(source.scopeHash, `${path}.scopeHash`, 64);
  if (!/^[a-f0-9]{64}$/.test(scopeHash)) {
    fail(`${path}.scopeHash`, "expected a lowercase SHA-256 digest");
  }
  const milestones = asArray(source.milestones, `${path}.milestones`, 12).map(
    (milestone, index) =>
      parseQuestMilestone(milestone, `${path}.milestones[${index}]`),
  );
  if (milestones.length < 4) fail(`${path}.milestones`, "requires at least four");
  requireSorted(
    milestones.map((milestone) => milestone.id),
    `${path}.milestones`,
  );
  if (new Set(milestones.map((milestone) => milestone.id)).size !== milestones.length) {
    fail(`${path}.milestones`, "contains duplicate identifiers");
  }
  for (const requiredLevel of ["E0", "E3", "E4", "E5"] as const) {
    if (!milestones.some((milestone) => milestone.evidenceLevel === requiredLevel)) {
      fail(`${path}.milestones`, `requires a ${requiredLevel} acceptance test`);
    }
  }
  const requiredReceiptTypes = asStringArray(
    source.requiredReceiptTypes,
    `${path}.requiredReceiptTypes`,
    8,
  );
  requireSorted(requiredReceiptTypes, `${path}.requiredReceiptTypes`);
  if (
    requiredReceiptTypes.length !== REQUIRED_QUEST_RECEIPTS.length ||
    requiredReceiptTypes.some(
      (receipt, index) => receipt !== REQUIRED_QUEST_RECEIPTS[index],
    )
  ) {
    fail(`${path}.requiredReceiptTypes`, "does not match the closed v1 receipt set");
  }
  return {
    targetEvidence: requireLiteral(
      source.targetEvidence,
      "E5",
      `${path}.targetEvidence`,
    ),
    scopeBounds,
    scopeHash,
    milestones,
    minimumEffectiveClusters: requireLiteral(
      source.minimumEffectiveClusters,
      3,
      `${path}.minimumEffectiveClusters`,
    ),
    minimumOrganizationRoots: requireLiteral(
      source.minimumOrganizationRoots,
      2,
      `${path}.minimumOrganizationRoots`,
    ),
    minimumDataRoots: requireLiteral(
      source.minimumDataRoots,
      2,
      `${path}.minimumDataRoots`,
    ),
    minimumAnalysisPipelineRoots: requireLiteral(
      source.minimumAnalysisPipelineRoots,
      2,
      `${path}.minimumAnalysisPipelineRoots`,
    ),
    requiredReceiptTypes,
    nullFailureContradictionRule: requireLiteral(
      source.nullFailureContradictionRule,
      "preserve-and-report-no-scope-mutation-no-automatic-reward",
      `${path}.nullFailureContradictionRule`,
    ),
    safetyApprovalSubstitutionAllowed: requireFalse(
      source.safetyApprovalSubstitutionAllowed,
      `${path}.safetyApprovalSubstitutionAllowed`,
    ),
  };
}

function parsePrerequisiteSemantics(
  value: unknown,
  path: string,
): LifeGardenPrerequisiteSemantics {
  const source = asObject(value, path);
  exactKeys(source, ["default", "grouped"], path);
  const grouped = asArray(source.grouped, `${path}.grouped`, 12).map(
    (entry, index): LifeGardenGroupedPrerequisite => {
      const entryPath = `${path}.grouped[${index}]`;
      const group = asObject(entry, entryPath);
      exactKeys(group, ["nodeId", "allOf", "atLeast"], entryPath);
      const nodeId = asString(group.nodeId, `${entryPath}.nodeId`, 128);
      if (!NODE_ID_PATTERN.test(nodeId)) {
        fail(`${entryPath}.nodeId`, "invalid versioned node identifier");
      }
      const allOf = asStringArray(group.allOf, `${entryPath}.allOf`, 16);
      requireSorted(allOf, `${entryPath}.allOf`);
      const atLeastSource = asObject(group.atLeast, `${entryPath}.atLeast`);
      exactKeys(atLeastSource, ["count", "of"], `${entryPath}.atLeast`);
      const of = asStringArray(atLeastSource.of, `${entryPath}.atLeast.of`, 16);
      requireSorted(of, `${entryPath}.atLeast.of`);
      const count = asSafeInteger(
        atLeastSource.count,
        `${entryPath}.atLeast.count`,
      );
      if (of.length < 2 || count < 1 || count >= of.length) {
        fail(
          `${entryPath}.atLeast`,
          "requires a positive count smaller than a choice set of at least two",
        );
      }
      if (allOf.some((id) => of.includes(id))) {
        fail(entryPath, "allOf and atLeast.of must be disjoint");
      }
      return { nodeId, allOf, atLeast: { count, of } };
    },
  );
  requireSorted(
    grouped.map((group) => group.nodeId),
    `${path}.grouped`,
  );
  if (new Set(grouped.map((group) => group.nodeId)).size !== grouped.length) {
    fail(`${path}.grouped`, "contains duplicate node rules");
  }
  return {
    default: requireLiteral(
      source.default,
      "all-listed-required",
      `${path}.default`,
    ),
    grouped,
  };
}

function parseNode(value: unknown, path: string): LifeGardenNode {
  const source = asObject(value, path);
  const kind = asEnum(source.kind, LIFE_GARDEN_KINDS, `${path}.kind`);
  exactKeys(
    source,
    [
      "id",
      "title",
      "stage",
      "domain",
      "kind",
      "summary",
      "prerequisites",
      "evidenceContribution",
      "rewardEligibility",
      "safetyTier",
      "artifactRequirements",
      "failureModes",
      "sourceIds",
      ...(kind === "quest" ? ["acceptance"] : []),
    ],
    path,
  );
  const id = asString(source.id, `${path}.id`, 128);
  if (!NODE_ID_PATTERN.test(id)) fail(`${path}.id`, "invalid versioned node identifier");
  const prerequisites = asStringArray(source.prerequisites, `${path}.prerequisites`, 16);
  const artifactRequirements = asStringArray(
    source.artifactRequirements,
    `${path}.artifactRequirements`,
    8,
  );
  const failureModes = asStringArray(source.failureModes, `${path}.failureModes`, 8);
  const sourceIds = asStringArray(source.sourceIds, `${path}.sourceIds`, 16);
  if (artifactRequirements.length < 2) {
    fail(`${path}.artifactRequirements`, "requires at least two artifacts");
  }
  if (failureModes.length < 2) fail(`${path}.failureModes`, "requires at least two");
  if (sourceIds.length === 0) fail(`${path}.sourceIds`, "requires at least one source");
  requireSorted(prerequisites, `${path}.prerequisites`);
  requireSorted(sourceIds, `${path}.sourceIds`);
  return {
    id,
    title: asString(source.title, `${path}.title`, 256),
    stage: asEnum(source.stage, LIFE_GARDEN_STAGES, `${path}.stage`),
    domain: asEnum(source.domain, LIFE_GARDEN_DOMAINS, `${path}.domain`),
    kind,
    summary: asString(source.summary, `${path}.summary`),
    prerequisites,
    evidenceContribution: asEnum(
      source.evidenceContribution,
      LIFE_GARDEN_EVIDENCE_CONTRIBUTIONS,
      `${path}.evidenceContribution`,
    ),
    rewardEligibility: asEnum(
      source.rewardEligibility,
      LIFE_GARDEN_REWARD_ELIGIBILITY,
      `${path}.rewardEligibility`,
    ),
    safetyTier: asEnum(
      source.safetyTier,
      LIFE_GARDEN_SAFETY_TIERS,
      `${path}.safetyTier`,
    ),
    artifactRequirements,
    failureModes,
    sourceIds,
    acceptance:
      kind === "quest"
        ? parseQuestAcceptance(source.acceptance, `${path}.acceptance`)
        : null,
  };
}

function validateGardenGraph(
  nodes: LifeGardenNode[],
  roots: string[],
  sourceIds: Set<string>,
  prerequisiteSemantics: LifeGardenPrerequisiteSemantics,
): void {
  const byId = new Map<string, LifeGardenNode>();
  for (const node of nodes) {
    if (byId.has(node.id)) fail("$.nodes", `duplicate node ${node.id}`);
    byId.set(node.id, node);
  }
  for (const [index, node] of nodes.entries()) {
    for (const prerequisite of node.prerequisites) {
      const dependency = byId.get(prerequisite);
      if (!dependency) {
        fail(`$.nodes[${index}].prerequisites`, `missing node ${prerequisite}`);
      }
      if (dependency.id === node.id) {
        fail(`$.nodes[${index}].prerequisites`, "self dependency");
      }
      if (
        LIFE_GARDEN_STAGES.indexOf(dependency.stage) >
        LIFE_GARDEN_STAGES.indexOf(node.stage)
      ) {
        fail(`$.nodes[${index}].prerequisites`, "points backward across stages");
      }
    }
    for (const sourceId of node.sourceIds) {
      if (!sourceIds.has(sourceId)) {
        fail(`$.nodes[${index}].sourceIds`, `missing source ${sourceId}`);
      }
    }
    if (
      node.rewardEligibility === "sponsor-template-only" &&
      node.kind !== "quest"
    ) {
      fail(`$.nodes[${index}].rewardEligibility`, "sponsor templates are quest-only");
    }
    if (node.kind === "quest") {
      if (
        node.stage !== "quest" ||
        node.evidenceContribution !== "E5" ||
        node.acceptance === null
      ) {
        fail(`$.nodes[${index}]`, "quests must be quest-stage E5 targets");
      }
    } else if (node.stage === "quest") {
      fail(`$.nodes[${index}].kind`, "quest-stage nodes must have quest kind");
    }
  }
  for (const [index, rule] of prerequisiteSemantics.grouped.entries()) {
    const node = byId.get(rule.nodeId);
    if (!node) {
      fail(
        `$.prerequisiteSemantics.grouped[${index}].nodeId`,
        `missing node ${rule.nodeId}`,
      );
    }
    if (node.kind === "quest") {
      fail(
        `$.prerequisiteSemantics.grouped[${index}].nodeId`,
        "quest prerequisites cannot use alternatives in v1",
      );
    }
    const union = [...rule.allOf, ...rule.atLeast.of].sort((left, right) =>
      left.localeCompare(right, "en"),
    );
    if (
      union.length !== node.prerequisites.length ||
      union.some((id, prerequisiteIndex) => id !== node.prerequisites[prerequisiteIndex])
    ) {
      fail(
        `$.prerequisiteSemantics.grouped[${index}]`,
        "must partition the node's listed prerequisite edges exactly",
      );
    }
  }
  const graphRoots = nodes
    .filter((node) => node.prerequisites.length === 0)
    .map((node) => node.id)
    .sort((left, right) => left.localeCompare(right, "en"));
  if (
    graphRoots.length !== roots.length ||
    graphRoots.some((root, index) => root !== roots[index])
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

  const dependents = new Map<string, string[]>(nodes.map((node) => [node.id, []]));
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

export function parseEpigeneticsCapabilityGarden(
  value: unknown,
): EpigeneticsCapabilityGarden {
  const garden = asObject(value, "$");
  exactKeys(garden, GARDEN_TOP_LEVEL_KEYS, "$");
  requireLiteral(
    garden.schema,
    "zerone.epigenetics-capability-garden/v1",
    "$.schema",
  );
  requireFalse(garden.authoritative, "$.authoritative");
  requireFalse(garden.networkObserved, "$.networkObserved");
  requireFalse(garden.rewardBearing, "$.rewardBearing");

  const boundary = asObject(garden.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundary, GARDEN_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of GARDEN_BOUNDARY_KEYS) {
    requireFalse(boundary[key], `$.releaseBoundary.${key}`);
  }

  const evidenceLadder = asArray(garden.evidenceLadder, "$.evidenceLadder", 7).map(
    (step, index) => parseEvidenceStep(step, `$.evidenceLadder[${index}]`, index),
  );
  if (evidenceLadder.length !== 7) fail("$.evidenceLadder", "requires E0 through E6");
  const policy = parsePolicy(garden.policy, "$.policy");
  const prerequisiteSemantics = parsePrerequisiteSemantics(
    garden.prerequisiteSemantics,
    "$.prerequisiteSemantics",
  );
  const rewardTotal =
    evidenceLadder.reduce((total, step) => total + step.rewardBps, 0) +
    policy.funding.challengeReserveBps;
  if (rewardTotal !== 10_000) fail("$.evidenceLadder", "template must conserve 10,000 bps");

  const sources = asArray(garden.sources, "$.sources", 32).map((source, index) =>
    parseSource(source, `$.sources[${index}]`),
  );
  if (sources.length === 0) fail("$.sources", "must not be empty");
  const sourceIds = sources.map((source) => source.id);
  requireSorted(sourceIds, "$.sources");
  if (new Set(sourceIds).size !== sourceIds.length) fail("$.sources", "duplicate source");

  const roots = asStringArray(garden.roots, "$.roots", MAX_NODES);
  if (roots.length === 0) fail("$.roots", "must not be empty");
  requireSorted(roots, "$.roots");
  const nodes = asArray(garden.nodes, "$.nodes", MAX_NODES).map((node, index) =>
    parseNode(node, `$.nodes[${index}]`),
  );
  if (nodes.length === 0) fail("$.nodes", "must not be empty");
  requireSorted(
    nodes.map((node) => node.id),
    "$.nodes",
  );
  validateGardenGraph(
    nodes,
    roots,
    new Set(sourceIds),
    prerequisiteSemantics,
  );

  return {
    schema: "zerone.epigenetics-capability-garden/v1",
    authoritative: false,
    networkObserved: false,
    rewardBearing: false,
    snapshotDate: asIsoDate(garden.snapshotDate, "$.snapshotDate"),
    policyVersion: asString(garden.policyVersion, "$.policyVersion", 64),
    releaseBoundary: {
      addsConsensusBehavior: false,
      activatesRewards: false,
      movesFunds: false,
      grantsQualification: false,
      authorizesBiologicalExperimentation: false,
      authorizesHumanIntervention: false,
      publishesIdentifiableGenomicData: false,
      assertsClinicalValidity: false,
    },
    evidenceLadder,
    policy,
    prerequisiteSemantics,
    sources,
    roots,
    nodes,
  };
}

function parseKarmaState(value: unknown, path: string): KarmaState {
  const source = asObject(value, path);
  exactKeys(source, ["name", "summable", "meaning"], path);
  return {
    name: asEnum(source.name, ["RECOGNIZED", "ORDINAL"] as const, `${path}.name`),
    summable: requireFalse(source.summable, `${path}.summable`),
    meaning: asString(source.meaning, `${path}.meaning`),
  };
}

const KARMA_EVENT_IDS = [
  "cited",
  "corroborate",
  "corroborated",
  "external",
  "pending_open",
  "pending_settle",
  "verify",
] as const;

const KARMA_EVENT_STATES: Record<(typeof KARMA_EVENT_IDS)[number], KarmaState["name"]> = {
  cited: "RECOGNIZED",
  corroborate: "RECOGNIZED",
  corroborated: "RECOGNIZED",
  external: "ORDINAL",
  pending_open: "ORDINAL",
  pending_settle: "ORDINAL",
  verify: "RECOGNIZED",
};

function parseKarmaEvent(value: unknown, path: string): KarmaEventKind {
  const source = asObject(value, path);
  exactKeys(source, ["id", "state", "meaning"], path);
  return {
    id: asEnum(source.id, KARMA_EVENT_IDS, `${path}.id`),
    state: asEnum(
      source.state,
      ["RECOGNIZED", "ORDINAL"] as const,
      `${path}.state`,
    ),
    meaning: asString(source.meaning, `${path}.meaning`),
  };
}

function parseKarmaInvariant(value: unknown, path: string): KarmaInvariant {
  const source = asObject(value, path);
  exactKeys(source, ["id", "statement"], path);
  const id = asString(source.id, `${path}.id`, 128);
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(id)) {
    fail(`${path}.id`, "invalid invariant identifier");
  }
  return { id, statement: asString(source.statement, `${path}.statement`) };
}

function parseKarmaGate(value: unknown, path: string): KarmaGovernanceGate {
  const source = asObject(value, path);
  exactKeys(source, ["id", "passed", "requirement"], path);
  const id = asString(source.id, `${path}.id`, 128);
  if (!/^G[0-7]-[a-z0-9]+(?:-[a-z0-9]+)*$/.test(id)) {
    fail(`${path}.id`, "invalid governance gate identifier");
  }
  return {
    id,
    passed: requireFalse(source.passed, `${path}.passed`),
    requirement: asString(source.requirement, `${path}.requirement`),
  };
}

export function parseKarmaFoundation(value: unknown): KarmaFoundation {
  const karma = asObject(value, "$");
  exactKeys(karma, KARMA_TOP_LEVEL_KEYS, "$");
  requireLiteral(karma.schema, "zerone.karma-foundation/v1", "$.schema");
  for (const key of [
    "authoritative",
    "networkObserved",
    "economicBearing",
    "governanceBearing",
    "transferable",
    "purchasable",
    "delegable",
    "founderPrivilege",
    "operatorPrivilege",
    "scalarRank",
  ] as const) {
    requireFalse(karma[key], `$.${key}`);
  }
  const boundary = asObject(karma.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundary, KARMA_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of KARMA_BOUNDARY_KEYS) {
    requireFalse(boundary[key], `$.releaseBoundary.${key}`);
  }

  const covenant = asObject(karma.economicCovenant, "$.economicCovenant");
  exactKeys(
    covenant,
    [
      "founderShare",
      "founderControl",
      "operatorShare",
      "operatorControl",
      "creatorRoyalty",
      "structurallyEnforced",
      "currentStatus",
      "residualValueDestination",
    ],
    "$.economicCovenant",
  );
  for (const key of [
    "founderShare",
    "founderControl",
    "operatorShare",
    "operatorControl",
    "creatorRoyalty",
    "structurallyEnforced",
  ] as const) {
    requireFalse(covenant[key], `$.economicCovenant.${key}`);
  }

  const states = asArray(karma.states, "$.states", 2).map((state, index) =>
    parseKarmaState(state, `$.states[${index}]`),
  );
  const expectedStates = ["RECOGNIZED", "ORDINAL"];
  if (
    states.length !== expectedStates.length ||
    states.some((state, index) => state.name !== expectedStates[index])
  ) {
    fail("$.states", "must preserve RECOGNIZED then ORDINAL");
  }

  const eventVocabulary = asArray(
    karma.eventVocabulary,
    "$.eventVocabulary",
    KARMA_EVENT_IDS.length,
  ).map((event, index) => parseKarmaEvent(event, `$.eventVocabulary[${index}]`));
  if (
    eventVocabulary.length !== KARMA_EVENT_IDS.length ||
    eventVocabulary.some((event, index) => event.id !== KARMA_EVENT_IDS[index])
  ) {
    fail("$.eventVocabulary", "must preserve the exact K-alpha vocabulary");
  }
  for (const event of eventVocabulary) {
    if (event.state !== KARMA_EVENT_STATES[event.id]) {
      fail(
        `$.eventVocabulary.${event.id}.state`,
        `must preserve K-alpha state ${KARMA_EVENT_STATES[event.id]}`,
      );
    }
  }

  const invariants = asArray(karma.invariants, "$.invariants", 9).map(
    (invariant, index) => parseKarmaInvariant(invariant, `$.invariants[${index}]`),
  );
  const invariantIds = invariants.map((invariant) => invariant.id);
  if (invariants.length !== 9 || new Set(invariantIds).size !== invariantIds.length) {
    fail("$.invariants", "requires exactly nine unique invariants");
  }
  requireSorted(invariantIds, "$.invariants");

  const prohibitedUses = asStringArray(
    karma.prohibitedUses,
    "$.prohibitedUses",
    9,
  );
  if (prohibitedUses.length !== 9) {
    fail("$.prohibitedUses", "requires exactly nine prohibited uses");
  }
  requireSorted(prohibitedUses, "$.prohibitedUses");

  const futureGovernanceGates = asArray(
    karma.futureGovernanceGates,
    "$.futureGovernanceGates",
    8,
  ).map((gate, index) => parseKarmaGate(gate, `$.futureGovernanceGates[${index}]`));
  if (
    futureGovernanceGates.length !== 8 ||
    futureGovernanceGates.some((gate, index) => !gate.id.startsWith(`G${index}-`))
  ) {
    fail("$.futureGovernanceGates", "requires closed gates G0 through G7 in order");
  }

  return {
    schema: "zerone.karma-foundation/v1",
    authoritative: false,
    networkObserved: false,
    economicBearing: false,
    governanceBearing: false,
    transferable: false,
    purchasable: false,
    delegable: false,
    founderPrivilege: false,
    operatorPrivilege: false,
    scalarRank: false,
    snapshotDate: asIsoDate(karma.snapshotDate, "$.snapshotDate"),
    status: requireLiteral(karma.status, "design-only", "$.status"),
    purpose: asString(karma.purpose, "$.purpose"),
    eventRegister: requireLiteral(
      karma.eventRegister,
      "priced-coherence",
      "$.eventRegister",
    ),
    releaseBoundary: {
      addsConsensusBehavior: false,
      activatesRewards: false,
      movesFunds: false,
      grantsAuthority: false,
      grantsQualification: false,
      modifiesKarmaEvents: false,
      derivesPersonScores: false,
      changesGovernanceWeight: false,
    },
    economicCovenant: {
      founderShare: false,
      founderControl: false,
      operatorShare: false,
      operatorControl: false,
      creatorRoyalty: false,
      structurallyEnforced: false,
      currentStatus: requireLiteral(
        covenant.currentStatus,
        "declarative-until-independent-enforcement",
        "$.economicCovenant.currentStatus",
      ),
      residualValueDestination: requireLiteral(
        covenant.residualValueDestination,
        "commons-return-or-burn-subject-to-future-independent-ratification",
        "$.economicCovenant.residualValueDestination",
      ),
    },
    states,
    eventVocabulary,
    invariants,
    prohibitedUses,
    futureGovernanceGates,
  };
}

function parseJson(raw: string, label: string): unknown {
  if (new TextEncoder().encode(raw).byteLength > STATIC_STANDARD_MAX_BYTES) {
    throw new LifeGardenDataError(
      `${label} exceeds ${STATIC_STANDARD_MAX_BYTES} UTF-8 bytes`,
    );
  }
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new LifeGardenDataError(`${label} contains malformed JSON`);
  }
}

export function parseEpigeneticsCapabilityGardenJson(
  raw: string,
): EpigeneticsCapabilityGarden {
  return parseEpigeneticsCapabilityGarden(parseJson(raw, "Life-science garden"));
}

export function parseKarmaFoundationJson(raw: string): KarmaFoundation {
  return parseKarmaFoundation(parseJson(raw, "KARMA foundation"));
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
  label: string,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new LifeGardenDataError(`${label} returned an empty response body`);
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
      signal.reason ?? new DOMException(`${label} request timed out`, "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > STATIC_STANDARD_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal is immediate even when a hostile stream ignores cancellation.
        });
        throw new LifeGardenDataError(`${label} exceeds its size limit`);
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // Deadline refusal does not wait for cancellation.
      });
      throw new LifeGardenDataError(`${label} request timed out`);
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
    throw new LifeGardenDataError("Static-standard digest verification is unavailable");
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
  endpoint: string,
  label: string,
  baseUrl?: string,
): void {
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(endpoint, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new LifeGardenDataError(`${label} returned an invalid final URL`);
  }
  if (
    actual.origin !== expected.origin ||
    actual.pathname !== expected.pathname ||
    actual.search !== expected.search ||
    actual.hash !== expected.hash
  ) {
    throw new LifeGardenDataError(`${label} left its canonical same-origin path`);
  }
}

async function fetchPinnedStandard<T>(
  endpoint: string,
  expectedDigest: string,
  label: string,
  parser: (raw: string) => T,
  options: StaticStandardFetchOptions,
): Promise<T> {
  const fetcher = options.fetcher ?? fetch;
  const controller = new AbortController();
  const timeoutMs = options.timeoutMs ?? 8_000;
  const deadline = globalThis.setTimeout(() => {
    controller.abort(new DOMException(`${label} request timed out`, "TimeoutError"));
  }, timeoutMs);
  const signal = controller.signal;
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? undefined : window.location.href);
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        signal.reason ??
          new DOMException(`${label} request timed out`, "TimeoutError"),
      );
    };
    signal.addEventListener("abort", onFetchAbort, { once: true });
    if (signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(endpoint, {
          cache: "no-store",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          redirect: "error",
          signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (
        signal.aborted ||
        (error instanceof DOMException &&
          (error.name === "AbortError" || error.name === "TimeoutError"))
      ) {
        throw new LifeGardenDataError(`${label} request timed out`);
      }
      throw new LifeGardenDataError(`${label} is unavailable`);
    } finally {
      signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      throw new LifeGardenDataError(`${label} returned HTTP ${response.status}`);
    }
    const contentType = response.headers.get("content-type");
    if (contentType === null || !/\bjson\b/i.test(contentType)) {
      throw new LifeGardenDataError(`${label} returned a non-JSON response`);
    }
    const declaredLength = response.headers.get("content-length");
    if (
      declaredLength !== null &&
      (!/^\d+$/.test(declaredLength) ||
        Number(declaredLength) > STATIC_STANDARD_MAX_BYTES)
    ) {
      throw new LifeGardenDataError(`${label} exceeded its size limit`);
    }
    assertCanonicalResponseUrl(response, endpoint, label, baseUrl);
    const bytes = await readBoundedResponse(response, signal, label);
    if ((await sha256Hex(bytes)) !== expectedDigest) {
      throw new LifeGardenDataError(
        `${label} did not match the reviewed canonical digest`,
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new LifeGardenDataError(`${label} was not valid UTF-8`);
    }
    return parser(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

export function fetchEpigeneticsCapabilityGarden(
  options: StaticStandardFetchOptions = {},
): Promise<EpigeneticsCapabilityGarden> {
  return fetchPinnedStandard(
    EPIGENETICS_GARDEN_ENDPOINT,
    EPIGENETICS_GARDEN_SHA256,
    "Life-science garden",
    parseEpigeneticsCapabilityGardenJson,
    options,
  );
}

export function fetchKarmaFoundation(
  options: StaticStandardFetchOptions = {},
): Promise<KarmaFoundation> {
  return fetchPinnedStandard(
    KARMA_FOUNDATION_ENDPOINT,
    KARMA_FOUNDATION_SHA256,
    "KARMA foundation",
    parseKarmaFoundationJson,
    options,
  );
}

function normaliseSearch(value: string): string {
  return value.normalize("NFKD").toLocaleLowerCase("en").trim();
}

export function filterLifeGardenNodes(
  garden: EpigeneticsCapabilityGarden,
  filters: LifeGardenFilters,
): LifeGardenNode[] {
  const query = normaliseSearch(filters.query);
  return garden.nodes.filter((node) => {
    if (filters.stage !== "all" && node.stage !== filters.stage) return false;
    if (filters.domain !== "all" && node.domain !== filters.domain) return false;
    if (filters.safetyTier !== "all" && node.safetyTier !== filters.safetyTier) {
      return false;
    }
    if (query === "") return true;
    return normaliseSearch(
      [
        node.id,
        node.title,
        node.summary,
        node.domain,
        node.kind,
        ...node.artifactRequirements,
        ...node.failureModes,
      ].join(" "),
    ).includes(query);
  });
}

export function evaluateLifeGardenSourceReview(
  garden: EpigeneticsCapabilityGarden,
  asOf: string,
): LifeGardenSourceReview {
  const reviewedAsOf = asIsoDate(asOf, "asOf");
  if (garden.sources.length === 0) {
    throw new LifeGardenDataError("Life-science garden has no reviewed sources");
  }
  const checkedThrough = garden.sources.reduce(
    (latest, source) => (source.checkedAt > latest ? source.checkedAt : latest),
    garden.sources[0]?.checkedAt ?? "",
  );
  const reviewAfter = garden.sources.reduce(
    (earliest, source) =>
      source.reviewAfter < earliest ? source.reviewAfter : earliest,
    garden.sources[0]?.reviewAfter ?? "",
  );
  return {
    checkedThrough,
    reviewAfter,
    due: reviewedAsOf > reviewAfter,
  };
}

export function formatBasisPoints(basisPoints: number): string {
  if (!Number.isSafeInteger(basisPoints) || basisPoints < 0) {
    throw new LifeGardenDataError("Basis points must be a non-negative integer");
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

function labelFor(value: string): string {
  return value.replaceAll("-", " ").replaceAll("_", " ");
}

function evidenceContributionLabel(
  contribution: LifeGardenEvidenceContribution,
): string {
  return contribution === "cross-cutting"
    ? "contributes across ladder"
    : `contributes toward ${contribution}`;
}

function appendFact(
  root: HTMLElement,
  label: string,
  value: string,
): void {
  const fact = element("div");
  fact.append(element("span", undefined, label), element("strong", undefined, value));
  root.append(fact);
}

function appendTextList(
  root: HTMLElement,
  items: readonly string[],
  className = "life-text-list",
): void {
  const list = element("ul", className);
  for (const item of items) list.append(element("li", undefined, item));
  root.append(list);
}

function renderKarmaFoundation(karma: KarmaFoundation): HTMLElement {
  const panel = element("article", "life-karma-panel");
  const heading = element("div", "life-panel-heading");
  const copy = element("div");
  copy.append(
    element("span", "card-kicker", "KARMA · zero-authority foundation"),
    element("h3", undefined, "Recognition without possession."),
    element("p", undefined, karma.purpose),
  );
  heading.append(copy, element("span", "life-status-badge", "design only · 0 authority"));
  panel.append(heading);

  const facts = element("div", "life-karma-facts");
  appendFact(facts, "Transferable", "No");
  appendFact(facts, "Purchasable", "No");
  appendFact(facts, "Founder privilege", "None");
  appendFact(facts, "Scalar rank", "Never");
  panel.append(facts);

  const covenant = element("div", "life-covenant");
  covenant.append(
    element("strong", undefined, "A declaration awaiting structural enforcement"),
    element(
      "p",
      undefined,
      "Founder, creator, and operator share or control are all zero in this foundation. That intent is not yet structurally enforced; future independent ratification must bind any commons return or burn.",
    ),
  );
  panel.append(covenant);

  const details = element("details", "life-disclosure");
  const summary = element("summary");
  summary.append(
    element("strong", undefined, "Nine constitutional invariants"),
    element(
      "span",
      undefined,
      `${karma.futureGovernanceGates.filter((gate) => gate.passed).length}/${karma.futureGovernanceGates.length} governance gates passed`,
    ),
  );
  details.append(summary);
  const body = element("div", "life-disclosure-body");
  const invariants = element("ol", "life-invariants");
  for (const invariant of karma.invariants) {
    const item = element("li");
    item.append(
      element("strong", undefined, labelFor(invariant.id)),
      element("p", undefined, invariant.statement),
    );
    invariants.append(item);
  }
  const links = element("div", "life-standard-links");
  const raw = element("a", undefined, "Raw KARMA foundation ↗");
  raw.href = KARMA_FOUNDATION_ENDPOINT;
  const doctrine = element("a", undefined, "Read KARMA.md ↗");
  doctrine.href =
    "https://github.com/cambridgetcg/zerone-core/blob/main/docs/KARMA.md";
  doctrine.target = "_blank";
  doctrine.rel = "noreferrer";
  links.append(raw, doctrine);
  body.append(invariants, links);
  details.append(body);
  panel.append(details);
  return panel;
}

function renderRewardTemplate(garden: EpigeneticsCapabilityGarden): HTMLElement {
  const panel = element("article", "life-reward-panel");
  const heading = element("div", "life-panel-heading");
  const copy = element("div");
  copy.append(
    element("span", "card-kicker", "Breakthrough reward shape"),
    element("h3", undefined, "Evidence first. Money last."),
    element(
      "p",
      undefined,
      "A prospective split for separately funded, voluntary sponsor escrow. It is a simulation template: this release issues nothing, moves nothing, and rewards no skill unlock.",
    ),
  );
  heading.append(copy, element("span", "life-status-badge is-amber", "inactive · 0 moved"));
  panel.append(heading);

  const breakthrough = element("div", "life-breakthrough-rule");
  breakthrough.append(
    element(
      "strong",
      undefined,
      "E4 causal candidacy; E5 required for breakthrough recognition",
    ),
    element(
      "p",
      undefined,
      "Prior-art delta + effective independent reproduction + serious causal challenge + prospective generalisation or material descendant impact. Authors, sponsors, popularity, novelty alone, time served, and token holdings cannot select it.",
    ),
  );
  panel.append(breakthrough);

  const ladder = element("ol", "life-reward-ladder");
  for (const step of garden.evidenceLadder) {
    const item = element("li");
    item.append(
      element("span", "life-evidence-level", step.level),
      element("strong", undefined, labelFor(step.name)),
      element("small", undefined, formatBasisPoints(step.rewardBps)),
      element("p", undefined, step.meaning),
    );
    ladder.append(item);
  }
  const reserve = element("li", "life-challenge-reserve");
  reserve.append(
    element("span", "life-evidence-level", "↺"),
    element("strong", undefined, "challenge reserve"),
    element(
      "small",
      undefined,
      formatBasisPoints(garden.policy.funding.challengeReserveBps),
    ),
    element(
      "p",
      undefined,
      "Held for counterevidence, replication, correction, and unresolved challenge under any future sponsor agreement.",
    ),
  );
  ladder.append(reserve);
  panel.append(ladder);
  return panel;
}

function renderLifeNode(
  node: LifeGardenNode,
  byId: ReadonlyMap<string, LifeGardenNode>,
  sourcesById: ReadonlyMap<string, LifeGardenSource>,
  groupedPrerequisites: ReadonlyMap<string, LifeGardenGroupedPrerequisite>,
): HTMLDetailsElement {
  const card = element("details", "life-node");
  card.dataset.safety = node.safetyTier;
  const summary = element("summary");
  const meta = element(
    "span",
    "life-node-meta",
    `${labelFor(node.domain)} · ${evidenceContributionLabel(node.evidenceContribution)}`,
  );
  summary.append(
    meta,
    element("strong", undefined, node.title),
    element("small", undefined, labelFor(node.safetyTier)),
  );
  card.append(summary);

  const body = element("div", "life-node-body");
  body.append(element("p", "life-node-summary", node.summary));
  const truth = element("div", "life-node-truthline");
  truth.append(
    element("span", undefined, labelFor(node.kind)),
    element("span", undefined, labelFor(node.rewardEligibility)),
  );
  body.append(truth);

  const prerequisiteBlock = element("div", "life-node-block");
  prerequisiteBlock.append(element("h4", undefined, "Prerequisite rule"));
  if (node.prerequisites.length === 0) {
    prerequisiteBlock.append(element("p", undefined, "Root capability"));
  } else {
    const grouped = groupedPrerequisites.get(node.id);
    if (grouped) {
      if (grouped.allOf.length > 0) {
        prerequisiteBlock.append(
          element("p", "life-prerequisite-label", "All required"),
        );
        appendTextList(
          prerequisiteBlock,
          grouped.allOf.map((id) => byId.get(id)?.title ?? id),
        );
      }
      prerequisiteBlock.append(
        element(
          "p",
          "life-prerequisite-label",
          `At least ${grouped.atLeast.count} of ${grouped.atLeast.of.length}`,
        ),
      );
      appendTextList(
        prerequisiteBlock,
        grouped.atLeast.of.map((id) => byId.get(id)?.title ?? id),
      );
    } else {
      prerequisiteBlock.append(
        element("p", "life-prerequisite-label", "All listed are required"),
      );
      appendTextList(
        prerequisiteBlock,
        node.prerequisites.map((id) => byId.get(id)?.title ?? id),
      );
    }
  }
  body.append(prerequisiteBlock);

  const evidenceBlock = element("div", "life-node-block");
  evidenceBlock.append(element("h4", undefined, "Evidence artifacts"));
  appendTextList(evidenceBlock, node.artifactRequirements);
  body.append(evidenceBlock);

  if (node.acceptance !== null) {
    const acceptanceBlock = element("div", "life-node-block life-acceptance");
    acceptanceBlock.append(
      element("h4", undefined, "Inactive quest acceptance · E5 target"),
      element(
        "p",
        undefined,
        "These reviewed tests bound the static quest only. A separately funded case must still freeze its claim and artifact digests, conflicts, correction, expiry, refund, and escrow receipt; this dashboard cannot activate one.",
      ),
    );
    appendTextList(
      acceptanceBlock,
      node.acceptance.milestones.map(
        (milestone) =>
          `${milestone.evidenceLevel} · ${labelFor(milestone.id)} — ${milestone.acceptanceTest}`,
      ),
    );
    acceptanceBlock.append(
      element(
        "code",
        "life-scope-hash",
        `scope sha256 ${node.acceptance.scopeHash}`,
      ),
    );
    body.append(acceptanceBlock);
  }

  const failureBlock = element("div", "life-node-block is-failure");
  failureBlock.append(element("h4", undefined, "Failure modes"));
  appendTextList(failureBlock, node.failureModes);
  body.append(failureBlock);

  const sourceBlock = element("div", "life-node-block");
  sourceBlock.append(element("h4", undefined, "Related references"));
  const sourceList = element("ul", "life-source-list");
  for (const sourceId of node.sourceIds) {
    const source = sourcesById.get(sourceId);
    if (!source) continue;
    const item = element("li");
    const link = element("a", undefined, `${source.authority} · ${source.title} ↗`);
    link.href = source.url;
    link.target = "_blank";
    link.rel = "noreferrer";
    item.append(link, element("p", "life-source-scope", source.supportScope));
    sourceList.append(item);
  }
  sourceBlock.append(sourceList);
  body.append(sourceBlock);
  card.append(body);
  return card;
}

function makeFilterSelect(
  label: string,
  values: readonly string[],
): { wrapper: HTMLLabelElement; select: HTMLSelectElement } {
  const wrapper = element("label");
  wrapper.append(element("span", undefined, label));
  const select = element("select");
  const all = element("option", undefined, `All ${label.toLocaleLowerCase("en")}`);
  all.value = "all";
  select.append(all);
  for (const value of values) {
    const option = element("option", undefined, labelFor(value));
    option.value = value;
    select.append(option);
  }
  wrapper.append(select);
  return { wrapper, select };
}

function renderGardenExplorer(
  garden: EpigeneticsCapabilityGarden,
): HTMLElement {
  const explorer = element("div", "life-garden-explorer");
  const facts = element("div", "ci-tree-facts");
  appendFact(facts, "Capabilities", `${garden.nodes.length}`);
  appendFact(
    facts,
    "Prerequisite edges",
    `${garden.nodes.reduce((total, node) => total + node.prerequisites.length, 0)}`,
  );
  appendFact(
    facts,
    "Breakthrough quests",
    `${garden.nodes.filter((node) => node.kind === "quest").length} · template only`,
  );
  appendFact(facts, "Live reward / authority", "0 / 0");
  explorer.append(facts);

  const safety = element("div", "life-safety-line");
  safety.append(
    element("strong", undefined, "Human and genomic safety is a hard gate."),
    element(
      "p",
      undefined,
      "No raw identifiable data enters this standard or a public ledger. Every human-data case must attest its lawful access lane; controlled or unrestricted access then follows consent, policy, and jurisdiction. Institutional certification, human-participant review, biosafety risk assessment, animal-welfare review, and regulatory evidence remain separate external requirements where applicable—not claims this page can grant.",
    ),
  );
  explorer.append(safety);

  const controls = element("form", "life-garden-controls");
  controls.setAttribute("role", "search");
  controls.setAttribute("aria-label", "Filter epigenetics capabilities");
  controls.addEventListener("submit", (event) => event.preventDefault());
  const searchLabel = element("label");
  searchLabel.append(element("span", undefined, "Search the garden"));
  const search = element("input");
  search.type = "search";
  search.placeholder = "methylation, causal, single cell…";
  search.autocomplete = "off";
  searchLabel.append(search);
  const stage = makeFilterSelect("Stage", LIFE_GARDEN_STAGES);
  const domain = makeFilterSelect("Domain", LIFE_GARDEN_DOMAINS);
  const safetyTier = makeFilterSelect("Safety", LIFE_GARDEN_SAFETY_TIERS);
  const reset = element("button", "button button-ghost", "Reset");
  reset.type = "reset";
  controls.append(
    searchLabel,
    stage.wrapper,
    domain.wrapper,
    safetyTier.wrapper,
    reset,
  );
  explorer.append(controls);

  const results = element("div", "life-results-head");
  const resultCount = element("span");
  resultCount.setAttribute("role", "status");
  resultCount.setAttribute("aria-live", "polite");
  const raw = element("a", undefined, "Raw garden JSON ↗");
  raw.href = EPIGENETICS_GARDEN_ENDPOINT;
  results.append(resultCount, raw);
  explorer.append(results);

  const map = element("div", "life-stage-map");
  map.setAttribute("role", "region");
  map.setAttribute(
    "aria-label",
    "Epigenetics capabilities grouped by growth stage. Scroll horizontally to inspect all seven stages.",
  );
  map.tabIndex = 0;
  explorer.append(map);

  const byId = new Map(garden.nodes.map((node) => [node.id, node]));
  const sourcesById = new Map(garden.sources.map((source) => [source.id, source]));
  const groupedPrerequisites = new Map(
    garden.prerequisiteSemantics.grouped.map((rule) => [rule.nodeId, rule]),
  );
  const render = (): void => {
    const filters: LifeGardenFilters = {
      query: search.value,
      stage: stage.select.value as LifeGardenStage | "all",
      domain: domain.select.value as LifeGardenDomain | "all",
      safetyTier: safetyTier.select.value as LifeGardenSafetyTier | "all",
    };
    const filtered = filterLifeGardenNodes(garden, filters);
    resultCount.textContent = `${filtered.length} of ${garden.nodes.length} capabilities · horizontal tree, open any node`;
    map.replaceChildren();
    for (const [index, stageName] of LIFE_GARDEN_STAGES.entries()) {
      const column = element("section", "life-stage");
      const head = element("div", "life-stage-head");
      const stageNodes = filtered.filter((node) => node.stage === stageName);
      head.append(
        element("span", undefined, `${String(index + 1).padStart(2, "0")}`),
        element("h3", undefined, labelFor(stageName)),
        element("small", undefined, `${stageNodes.length}`),
      );
      column.append(head);
      if (stageNodes.length === 0) {
        column.append(element("p", "life-stage-empty", "No matching growth here."));
      } else {
        const list = element("div", "life-stage-nodes");
        for (const node of stageNodes) {
          list.append(
            renderLifeNode(
              node,
              byId,
              sourcesById,
              groupedPrerequisites,
            ),
          );
        }
        column.append(list);
      }
      map.append(column);
    }
  };
  controls.addEventListener("input", render);
  controls.addEventListener("change", render);
  controls.addEventListener("reset", () => window.setTimeout(render, 0));
  render();

  const footer = element("div", "life-garden-footer");
  const sourceReview = evaluateLifeGardenSourceReview(
    garden,
    new Date().toISOString().slice(0, 10),
  );
  const reviewStatus = element(
    "p",
    sourceReview.due ? "life-source-review is-due" : "life-source-review",
    sourceReview.due
      ? "Source review was due " +
          sourceReview.reviewAfter +
          ". This historical, inactive map must be re-reviewed before any future use."
      : garden.sources.length +
          " primary or official anchors checked through " +
          sourceReview.checkedThrough +
          "; next review by " +
          sourceReview.reviewAfter +
          ".",
  );
  footer.append(
    reviewStatus,
    element(
      "p",
      undefined,
      "Every quality claim remains assay-specific. The graph preserves biosample, method, file, analysis, and claim lineage separately; a shared embedding or score is never the mechanism.",
    ),
    element(
      "code",
      undefined,
      `garden sha256:${EPIGENETICS_GARDEN_SHA256}`,
    ),
  );
  explorer.append(footer);
  return explorer;
}

function renderLoadError(root: HTMLElement, error: unknown): void {
  root.ariaBusy = "false";
  const panel = element("div", "ci-tree-load-error");
  panel.setAttribute("role", "alert");
  panel.append(
    element("strong", undefined, "The reviewed life-science standards did not load."),
    element(
      "p",
      undefined,
      error instanceof Error
        ? error.message
        : "The static standards are unavailable or failed validation.",
    ),
  );
  const actions = element("div", "ci-tree-load-actions");
  const retry = element("button", "button button-primary", "Retry");
  retry.type = "button";
  retry.addEventListener("click", () => void initialiseLifeGarden(root));
  const raw = element("a", "button button-ghost", "Open raw garden");
  raw.href = EPIGENETICS_GARDEN_ENDPOINT;
  actions.append(retry, raw);
  panel.append(actions);
  root.replaceChildren(panel);
}

export async function initialiseLifeGarden(root: HTMLElement): Promise<void> {
  root.ariaBusy = "true";
  try {
    const [garden, karma] = await Promise.all([
      fetchEpigeneticsCapabilityGarden(),
      fetchKarmaFoundation(),
    ]);
    const shell = element("div", "life-garden-shell");
    shell.append(
      renderKarmaFoundation(karma),
      renderRewardTemplate(garden),
      renderGardenExplorer(garden),
    );
    root.replaceChildren(shell);
    root.ariaBusy = "false";
  } catch (error) {
    renderLoadError(root, error);
  }
}
