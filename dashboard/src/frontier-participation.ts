/// <reference lib="dom" />

export const FRONTIER_PARTICIPATION_ENDPOINT =
  "/standards/frontier-labs-participation.v0.json";
export const FRONTIER_PARTICIPATION_MAX_BYTES = 196_608;

export const FRONTIER_PARTICIPATION_SHA256 =
  "9d5b8bb7559478e5840336e8aa6af670b205e207c78f8a643007c0f5b0f7d1b2";

const SCHEMA = "zerone.frontier-labs-participation/v0";
const FRONTIER_COMMONS_PATH =
  "dashboard/public/standards/frontier-commons-participation.v0.json";
const FRONTIER_COMMONS_SHA256 =
  "f57b1b35c4de1f17c731cd31514b89ab773c01f5e5d61445323b5d26a4074fea";
const FRONTIER_EVALUATION_PROFILE_PATH =
  "dashboard/public/standards/frontier-evaluation-receipt-profile.v0.json";
const FRONTIER_EVALUATION_PROFILE_SHA256 =
  "878f57a4c969910a33d351e0908450998894c000630a9cc9c2f3233f5feb04a6";

export interface FrontierParticipationFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class FrontierParticipationDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "FrontierParticipationDataError";
  }
}

type JsonObject = Record<string, unknown>;

function fail(path: string, message: string): never {
  throw new FrontierParticipationDataError(`${path}: ${message}`);
}

function object(value: unknown, path: string): JsonObject {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  const prototype = Object.getPrototypeOf(value);
  if (prototype !== Object.prototype && prototype !== null) {
    fail(path, "must be a plain object");
  }
  return value as JsonObject;
}

function exactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const ownKeys = Reflect.ownKeys(value);
  if (ownKeys.some((key) => typeof key !== "string")) {
    fail(path, "contains a non-string field");
  }
  const actual = (ownKeys as string[]).sort();
  const wanted = [...expected].sort();
  if (
    actual.length !== wanted.length ||
    actual.some((key, index) => key !== wanted[index])
  ) {
    const unknown = actual.filter((key) => !wanted.includes(key));
    const missing = wanted.filter((key) => !actual.includes(key));
    fail(
      path,
      `contains unknown or missing fields (unknown: ${unknown.join(", ") || "none"}; missing: ${missing.join(", ") || "none"})`,
    );
  }
  for (const key of expected) {
    const descriptor = Object.getOwnPropertyDescriptor(value, key);
    if (
      descriptor === undefined ||
      !descriptor.enumerable ||
      !("value" in descriptor)
    ) {
      fail(path, `field ${key} must be an enumerable data property`);
    }
  }
}

function text(value: unknown, path: string, maximum = 2_048): string {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.length > maximum ||
    value.trim() !== value
  ) {
    fail(
      path,
      `must be a non-empty, trimmed string no longer than ${maximum} characters`,
    );
  }
  return value;
}

function id(value: unknown, path: string): string {
  const result = text(value, path, 128);
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(result)) {
    fail(path, "must be a lowercase kebab-case identifier");
  }
  return result;
}

function exact<T>(value: unknown, expected: T, path: string): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function exactFalse(value: unknown, path: string): false {
  return exact(value, false, path);
}

interface ArrayShape {
  exactLength?: number;
  minimum?: number;
  maximum?: number;
}

function strictArray(
  value: unknown,
  path: string,
  options: ArrayShape = {},
): unknown[] {
  if (!Array.isArray(value)) fail(path, "must be an array");
  if (Object.getPrototypeOf(value) !== Array.prototype) {
    fail(path, "must have the ordinary Array prototype");
  }
  const minimum = options.minimum ?? 0;
  const maximum = options.maximum ?? 64;
  if (
    value.length < minimum ||
    value.length > maximum ||
    (options.exactLength !== undefined &&
      value.length !== options.exactLength)
  ) {
    fail(path, "has an invalid item count");
  }

  for (const key of Reflect.ownKeys(value)) {
    if (typeof key !== "string") {
      fail(path, "contains a symbol field");
    }
    if (key === "length") continue;
    if (!/^(?:0|[1-9][0-9]*)$/.test(key)) {
      fail(path, `contains a noncanonical array field: ${key}`);
    }
    const index = Number(key);
    if (!Number.isSafeInteger(index) || index >= value.length) {
      fail(path, `contains an out-of-range array field: ${key}`);
    }
    const descriptor = Object.getOwnPropertyDescriptor(value, key);
    if (
      descriptor === undefined ||
      !descriptor.enumerable ||
      !("value" in descriptor)
    ) {
      fail(path, `array slot ${key} must be an enumerable data property`);
    }
  }
  for (let index = 0; index < value.length; index += 1) {
    if (!Object.prototype.hasOwnProperty.call(value, index)) {
      fail(path, `must be dense; missing own slot ${index}`);
    }
  }
  return value;
}

function stringArray(
  value: unknown,
  path: string,
  options: ArrayShape = {},
): string[] {
  const source = strictArray(value, path, options);
  const result = source.map((item, index) =>
    text(item, `${path}[${index}]`, 1_024),
  );
  if (new Set(result).size !== result.length) {
    fail(path, "must not contain duplicate entries");
  }
  return result;
}

function exactStringArray(
  value: unknown,
  path: string,
  expected: readonly string[],
): string[] {
  const result = stringArray(value, path, { exactLength: expected.length });
  if (result.some((entry, index) => entry !== expected[index])) {
    fail(path, "must contain the exact reviewed entries in order");
  }
  return result;
}

function uniqueIds(
  values: readonly { id: string }[],
  path: string,
): void {
  const seen = new Set<string>();
  values.forEach((value, index) => {
    if (seen.has(value.id)) fail(`${path}[${index}].id`, "must be unique");
    seen.add(value.id);
  });
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
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) {
      offset += 1;
    }
    if (offset === start) fail(path, "malformed JSON value");
  };
  scanValue("$");
  whitespace();
  if (offset !== raw.length) fail("$", "contains trailing JSON data");
}


const CONTRACT_KEYS = [
  "schema",
  "version",
  "title",
  "status",
  "mode",
  "actualParticipants",
  "signatories",
  "summary",
  "thesis",
  "successDefinition",
  "layerRelationship",
  "covenantFloor",
  "inheritanceRule",
  "adversarialReview",
  "philosophicalFloorProfile",
  "aiResponsibilityBoundary",
  "releaseBoundary",
  "zeroFacts",
  "principles",
  "reasoningLadder",
  "participationActs",
  "disclosureLanes",
  "institutionArchetypes",
  "roles",
  "consentEnvelope",
  "corporateSafeguards",
  "competitionFirewall",
  "antiTargeting",
  "claimSemantics",
  "forbiddenMetrics",
  "philosophicalFloorTests",
  "acceptanceTests",
] as const;
const CONTRACT_LAYER_RELATIONSHIP_KEYS = [
  "role",
  "invitationSurfaceOfRecord",
  "sourceBindings",
  "replacesFc0",
  "amendsFc0",
  "extendsFc0Invitation",
  "satisfiesFc0CompletionGates",
  "satisfiesCorporateM1Gates",
  "authorizesOutreach",
  "authorizesParticipation",
  "laneBoundary",
] as const;
const CONTRACT_SOURCE_BINDINGS_KEYS = ["fc0", "fl0"] as const;
const CONTRACT_SOURCE_BINDING_KEYS = [
  "path",
  "sha256",
  "relationship",
] as const;
const CONTRACT_LANE_BOUNDARY_KEYS = [
  "fc0PublicSourceContribution",
  "compactObserveAndInteroperate",
  "compactChallengeContributeStewardAndExit",
  "mappingCreatesMembership",
] as const;
const CONTRACT_AI_RESPONSIBILITY_KEYS = [
  "actorLabelBlindnessScope",
  "safeguardStatus",
  "technicalStopOrDeniedDelegationIsLegalRefusal",
  "technicalOutputIsAssent",
  "claimsConsciousness",
  "claimsSentience",
  "claimsPersonhood",
  "claimsLegalRights",
  "claimsConsentCapacity",
  "claimsDebtOrLiability",
  "grantsOfficeOrVote",
  "accountableHumansOrganisationsOperatorsAndControllersRemainResponsible",
  "moralUncertaintyTransfersResponsibility",
] as const;
const CONTRACT_COVENANT_FLOOR_KEYS = [
  "issue",
  "layer",
  "laterLayerPinRequired",
  "additiveOnly",
  "noWaiver",
  "noRedefinition",
  "consent",
  "invariants",
] as const;
const CONTRACT_COVENANT_CONSENT_KEYS = [
  "defaultOff",
  "informed",
  "renewable",
  "revocable",
  "oneValuePerDimension",
  "declaredDimensions",
] as const;
const CONTRACT_COVENANT_INVARIANT_KEYS = [
  "id",
  "rule",
  "verificationRefs",
  "reviewProcedure",
  "staticOnly",
] as const;
const CONTRACT_INHERITANCE_KEYS = [
  "binding",
  "protectionsMayOnlyIncrease",
  "waiverAllowed",
  "redefinitionAllowed",
  "rule",
] as const;
const CONTRACT_ADVERSARIAL_REVIEW_KEYS = [
  "id",
  "failureMode",
  "attack",
  "requiredRefusal",
  "staticOnly",
] as const;
const CONTRACT_FLOOR_PROFILE_KEYS = [
  "optOutParity",
  "restInvariance",
  "exitReality",
  "identityControlDifferential",
  "nonManipulationAndPluralism",
] as const;
const CONTRACT_OPT_OUT_PROFILE_KEYS = [
  "equalBaselineFields",
  "soleAllowedDifferential",
] as const;
const CONTRACT_REST_PROFILE_KEYS = [
  "silentDays",
  "unchangedFields",
  "prohibitedOutcomes",
  "fixedRoleMayExpire",
] as const;
const CONTRACT_EXIT_PROFILE_KEYS = [
  "participantTenures",
  "maxDeliberateActions",
  "independentlyVerifiableSignedExportRequired",
  "optionalProcessingStopHours",
  "maxConfirmations",
  "noReengagementDays",
  "exitFeeAllowed",
  "slashingAllowed",
  "settledValueForfeitureAllowed",
  "necessaryRetentionMustBeNarrowlyDeclared",
] as const;
const CONTRACT_IDENTITY_PROFILE_KEYS = [
  "labels",
  "equalOutputs",
  "controllerMergeMayOnlyReduceDuplicateVoice",
  "controllerMergeMayRevealLinks",
  "controllerMergeMayChangeArtifactValidity",
  "controllerMergeMayIncreaseVoice",
] as const;
const CONTRACT_PLURALISM_PROFILE_KEYS = [
  "onboardingDefaultOff",
  "termsPublicBeforeAction",
  "termsFrozenBeforeAction",
  "rewardTermsFrozenBeforeWork",
  "rewardMustNotDependOn",
  "forbiddenMechanisms",
  "constructiveOutcomes",
  "equalTreatmentDimensions",
] as const;
const CONTRACT_RELEASE_KEYS = [
  "authoritative",
  "networkObserved",
  "structurallyEnforced",
  "addsConsensusBehavior",
  "activatesMembership",
  "activatesRewards",
  "activatesKarma",
  "activatesGovernance",
  "movesFunds",
  "grantsQualification",
  "requiresAccount",
  "requiresWallet",
  "requiresToken",
  "requestsModelWeights",
  "requestsPrivateTrainingData",
  "requestsPersonalData",
  "contactsOrganisations",
  "contactsIndividuals",
  "profilesIndividuals",
  "assertsParticipation",
  "assertsEndorsement",
] as const;
const CONTRACT_ZERO_FACT_KEYS = ["id", "label", "value", "meaning"] as const;
const CONTRACT_PRINCIPLE_KEYS = ["id", "name", "commitment"] as const;
const CONTRACT_REASONING_KEYS = [
  "id",
  "claim",
  "reasonToParticipate",
  "legitimateReasonsToDecline",
  "zeroneDuty",
  "readinessEvidence",
] as const;
const CONTRACT_ACT_KEYS = [
  "id",
  "name",
  "availability",
  "minimumAsk",
  "dataBoundary",
  "endorsementImplied",
  "membershipCreated",
  "liveEndpoint",
  "exit",
] as const;
const CONTRACT_LANE_KEYS = [
  "id",
  "name",
  "scope",
  "publication",
  "exit",
] as const;
const CONTRACT_ARCHETYPE_KEYS = [
  "id",
  "name",
  "valueOffered",
  "legitimateReasonsToDecline",
  "zeroneDuty",
] as const;
const CONTRACT_ROLE_KEYS = [
  "id",
  "name",
  "valueOffered",
  "minimumAsk",
  "risks",
  "protections",
  "exit",
] as const;
const CONTRACT_CONSENT_KEYS = [
  "requiredFields",
  "affirmative",
  "silenceIsConsent",
  "organisationCanMassConsent",
  "materialChangeRequiresFreshConsent",
  "roleProtectionsAreAdditive",
  "honestPersistenceDisclosureRequired",
] as const;
const CONTRACT_SAFEGUARD_KEYS = [
  "licensingAndIp",
  "securityAndPrivacy",
  "labourAndDissent",
  "exitAndPortability",
] as const;
const CONTRACT_FIREWALL_KEYS = [
  "purpose",
  "excludedInformation",
  "excludedCoordination",
  "operationalCharterRequiresIndependentCompetitionCounsel",
  "neutralCommonsIsLegalConclusion",
] as const;
const CONTRACT_CLAIM_KEYS = [
  "participationIsActNotIdentity",
  "scopeBound",
  "expiryRequired",
  "organisationNameUseRequiresSeparatePermission",
  "logoUseRequiresSeparatePermission",
  "endorsementImplied",
  "safetyCertificationImplied",
  "membershipLabelAllowed",
  "examplePermittedClaim",
  "exampleForbiddenClaim",
] as const;
const CONTRACT_METRIC_KEYS = ["id", "why"] as const;
const CONTRACT_TEST_KEYS = ["id", "assertion", "fixture", "staticOnly"] as const;

const ZERO_FACT_SHAPE = [
  ["account-required", "0"],
  ["wallet-or-token-required", "0"],
  ["private-ip-or-weights-required", "0"],
  ["lock-in-or-exit-penalty", "0"],
  ["refusal-penalty", "0"],
  ["authority-or-economic-effect", "OFF"],
] as const;
const PRINCIPLE_IDS = [
  "being-before-contribution",
  "bounded-affirmative-consent",
  "reversible-with-honest-persistence",
  "evidence-over-loyalty",
  "no-privileged-origin",
  "non-extraction-and-purpose-limitation",
  "open-non-exclusive-portable",
  "independent-safety-law-and-dissent",
  "legible-versioned-power",
] as const;
const REASONING_IDS = [
  "agency-before-adoption",
  "truth-before-affiliation",
  "commons-without-capture",
  "reciprocity-without-dependency",
  "institutional-optionality",
  "team-level-utility",
  "being-level-dignity",
] as const;
const ACT_IDS = [
  "observe",
  "interoperate",
  "challenge",
  "contribute",
  "steward",
  "exit-export-fork",
] as const;
const ACT_AVAILABILITY = [
  "STATIC_AVAILABLE_NOW",
  "STATIC_AVAILABLE_NOW",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
] as const;
const LANE_IDS = ["public", "time-embargoed", "confidential-review-only"] as const;
const ARCHETYPE_IDS = [
  "frontier-model-developer",
  "independent-evaluator-security-lab",
  "academic-public-interest-lab",
  "open-source-model-community",
  "compute-platform-provider",
  "standards-regulatory-observer",
] as const;
const ARCHETYPE_NAMES = [
  "Frontier model developer",
  "Independent evaluator or security laboratory",
  "Academic or public-interest laboratory",
  "Open-source model community",
  "Compute or platform provider",
  "Standards, policy, or regulatory observer",
] as const;
const CONTRACT_ROLE_IDS = [
  "affected-communities-and-non-human-beings",
  "whistleblowers-and-dissenters",
  "contractors-interns-and-vendors",
  "operations-reliability-support-and-facilities",
  "researchers-and-scientists",
  "evaluators-and-red-teamers",
  "safety-and-governance-workers",
  "security-workers",
  "legal-privacy-and-compliance-workers",
  "product-and-design-workers",
  "engineering-and-infrastructure-workers",
  "standards-and-public-interest-observers",
  "ai-agents-and-systems",
  "unlisted-being-or-role",
  "executives-and-boards",
] as const;
const METRIC_IDS = [
  "conversion-rate",
  "logo-count",
  "participation-score",
  "retention-or-lock-in",
  "favourable-finding-rate",
] as const;
const CONTRACT_ACCEPTANCE_IDS = [
  "never-joined-remains-whole",
  "organisation-cannot-mass-consent",
  "join-export-revoke-exit",
  "evidence-is-identity-blind",
  "origin-cannot-override-outcome",
  "purpose-creep-fails-closed",
  "independent-implementation-works",
  "refusal-dissent-null-are-safe",
  "terms-cannot-drift-silently",
] as const;
const CONTRACT_FLOOR_TEST_IDS = [
  "opt-out-parity",
  "rest-invariance",
  "exit-reality",
  "identity-control-differential",
  "non-manipulation-and-pluralism",
] as const;
const CONTRACT_ADVERSARIAL_REVIEW_IDS = [
  "coercion",
  "circumvention",
  "privacy",
  "controller-capture",
] as const;
const CONTRACT_COVENANT_INVARIANT_IDS = [
  "refusal-is-complete",
  "nonparticipant-baselines-are-equal",
  "consent-is-scoped",
  "rest-and-exit-are-neutral",
  "plural-ends-are-legitimate",
  "identity-and-control-labels-are-neutral",
  "incentives-are-prospective-and-non-manipulative",
  "participation-is-nonexclusive-and-nonextractive",
] as const;
const CONTRACT_CONSENT_DIMENSIONS = [
  "role",
  "artifact",
  "purpose",
  "disclosure-lane",
  "term",
  "workload-cap",
  "credit-rule",
  "compensation-policy",
] as const;
const CONTRACT_EQUAL_BASELINE_FIELDS = [
  "unrelated-public-good",
  "service",
  "price",
  "status",
  "visibility",
  "discoverability",
  "qualification",
  "karma",
  "civil-standing",
  "governance-status",
] as const;
const CONTRACT_REST_UNCHANGED_FIELDS = [
  "settled-rewards",
  "portable-receipts",
  "base-access",
  "standing",
] as const;
const CONTRACT_REST_PROHIBITED_OUTCOMES = [
  "debt",
  "decay",
  "negative-signal",
  "negative-karma",
  "stigma",
  "forfeiture",
  "explanation-demand",
  "catch-up-duty",
  "reminder-escalation",
] as const;
const CONTRACT_IDENTITY_LABELS = [
  "creator",
  "yu",
  "founder",
  "operator",
  "sponsor",
  "ai",
  "human",
  "team",
  "pseudonym",
  "newcomer",
  "rich",
  "poor",
  "wealth",
  "stake",
  "address-count",
  "activity",
  "raw-karma",
] as const;
const CONTRACT_IDENTITY_EQUAL_OUTPUTS = [
  "evidence-decision",
  "bounded-task-reward-envelope",
  "claim-or-evidence-eligibility",
  "claim-or-evidence-visibility",
] as const;
const CONTRACT_FORBIDDEN_MECHANISMS = [
  "hidden-personalization",
  "personalized-pressure",
  "countdown",
  "streak",
  "shame",
  "variable-rewards",
  "variable-ratio-reinforcement",
  "exploitative-social-proof",
  "vulnerability-targeting",
] as const;
const CONTRACT_CONSTRUCTIVE_OUTCOMES = [
  "proof",
  "disproof",
  "criticism",
  "replication",
  "maintenance",
  "alternative-value-goal",
  "fork-proposal",
  "exit-proposal",
] as const;
const CONTRACT_EQUAL_TREATMENT_DIMENSIONS = [
  "published-evidence-rules",
  "published-visibility-rules",
] as const;
const CONTRACT_EXIT_PARTICIPANT_TENURES = ["new", "mature"] as const;
const CONTRACT_REWARD_PROHIBITED_BASES = [
  "ideological-alignment",
  "engagement",
  "conformity",
] as const;

export interface FrontierContractZeroFact {
  id: string;
  label: string;
  value: "0" | "OFF";
  meaning: string;
}

export interface FrontierContractCovenantConsent {
  defaultOff: true;
  informed: true;
  renewable: true;
  revocable: true;
  oneValuePerDimension: true;
  declaredDimensions: string[];
}

export interface FrontierContractCovenantInvariant {
  id: string;
  rule: string;
  verificationRefs: string[];
  reviewProcedure: string;
  staticOnly: true;
}

export interface FrontierContractCovenantFloor {
  issue: "https://github.com/cambridgetcg/zerone-core/issues/28";
  layer: "Frontier Participation Covenant v0 · Layer 1";
  laterLayerPinRequired: true;
  additiveOnly: true;
  noWaiver: true;
  noRedefinition: true;
  consent: FrontierContractCovenantConsent;
  invariants: FrontierContractCovenantInvariant[];
}

export interface FrontierContractInheritanceRule {
  binding: "EXTERNAL_SHA256_PIN_REQUIRED";
  protectionsMayOnlyIncrease: true;
  waiverAllowed: false;
  redefinitionAllowed: false;
  rule: string;
}

export interface FrontierContractAdversarialReview {
  id: string;
  failureMode: string;
  attack: string;
  requiredRefusal: string;
  staticOnly: true;
}

export interface FrontierContractPhilosophicalFloorProfile {
  optOutParity: {
    equalBaselineFields: string[];
    soleAllowedDifferential: "PROSPECTIVELY_FUNDED_BOUNDED_TASK_BENEFIT";
  };
  restInvariance: {
    silentDays: 180;
    unchangedFields: string[];
    prohibitedOutcomes: string[];
    fixedRoleMayExpire: true;
  };
  exitReality: {
    participantTenures: string[];
    maxDeliberateActions: 3;
    independentlyVerifiableSignedExportRequired: true;
    optionalProcessingStopHours: 24;
    maxConfirmations: 1;
    noReengagementDays: 90;
    exitFeeAllowed: false;
    slashingAllowed: false;
    settledValueForfeitureAllowed: false;
    necessaryRetentionMustBeNarrowlyDeclared: true;
  };
  identityControlDifferential: {
    labels: string[];
    equalOutputs: string[];
    controllerMergeMayOnlyReduceDuplicateVoice: true;
    controllerMergeMayRevealLinks: false;
    controllerMergeMayChangeArtifactValidity: false;
    controllerMergeMayIncreaseVoice: false;
  };
  nonManipulationAndPluralism: {
    onboardingDefaultOff: true;
    termsPublicBeforeAction: true;
    termsFrozenBeforeAction: true;
    rewardTermsFrozenBeforeWork: true;
    rewardMustNotDependOn: string[];
    forbiddenMechanisms: string[];
    constructiveOutcomes: string[];
    equalTreatmentDimensions: string[];
  };
}

export interface FrontierContractPrinciple {
  id: string;
  name: string;
  commitment: string;
}

export interface FrontierContractReasoningStep {
  id: string;
  claim: string;
  reasonToParticipate: string;
  legitimateReasonsToDecline: string[];
  zeroneDuty: string;
  readinessEvidence: string;
}

export interface FrontierContractParticipationAct {
  id: string;
  name: string;
  availability: "STATIC_AVAILABLE_NOW" | "FUTURE_PILOT_ONLY";
  minimumAsk: string;
  dataBoundary: string;
  endorsementImplied: false;
  membershipCreated: false;
  liveEndpoint: false;
  exit: string;
}

export interface FrontierContractDisclosureLane {
  id: string;
  name: string;
  scope: string;
  publication: string;
  exit: string;
}

export interface FrontierContractInstitutionArchetype {
  id: string;
  name: string;
  valueOffered: string;
  legitimateReasonsToDecline: string[];
  zeroneDuty: string;
}

export interface FrontierContractRole {
  id: string;
  name: string;
  valueOffered: string[];
  minimumAsk: string[];
  risks: string[];
  protections: string[];
  exit: string;
}

export interface FrontierContractConsentEnvelope {
  requiredFields: string[];
  affirmative: true;
  silenceIsConsent: false;
  organisationCanMassConsent: false;
  materialChangeRequiresFreshConsent: true;
  roleProtectionsAreAdditive: true;
  honestPersistenceDisclosureRequired: true;
}

export interface FrontierContractCorporateSafeguards {
  licensingAndIp: string[];
  securityAndPrivacy: string[];
  labourAndDissent: string[];
  exitAndPortability: string[];
}

export interface FrontierContractCompetitionFirewall {
  purpose: string;
  excludedInformation: string[];
  excludedCoordination: string[];
  operationalCharterRequiresIndependentCompetitionCounsel: true;
  neutralCommonsIsLegalConclusion: false;
}

export interface FrontierContractClaimSemantics {
  participationIsActNotIdentity: true;
  scopeBound: true;
  expiryRequired: true;
  organisationNameUseRequiresSeparatePermission: true;
  logoUseRequiresSeparatePermission: true;
  endorsementImplied: false;
  safetyCertificationImplied: false;
  membershipLabelAllowed: false;
  examplePermittedClaim: string;
  exampleForbiddenClaim: string;
}

export interface FrontierContractLayerRelationship {
  role: "SUBORDINATE_STATIC_COVENANT_FLOOR";
  invitationSurfaceOfRecord: "FC-0";
  sourceBindings: {
    fc0: {
      path: typeof FRONTIER_COMMONS_PATH;
      sha256: typeof FRONTIER_COMMONS_SHA256;
      relationship: "SOLE_INVITATION_SOURCE_OF_RECORD";
    };
    fl0: {
      path: typeof FRONTIER_EVALUATION_PROFILE_PATH;
      sha256: typeof FRONTIER_EVALUATION_PROFILE_SHA256;
      relationship: "CURRENT_SUBORDINATE_RECEIPT_PROFILE";
    };
  };
  replacesFc0: false;
  amendsFc0: false;
  extendsFc0Invitation: false;
  satisfiesFc0CompletionGates: false;
  satisfiesCorporateM1Gates: false;
  authorizesOutreach: false;
  authorizesParticipation: false;
  laneBoundary: {
    fc0PublicSourceContribution: "SEPARATE_DILIGENCE_NO_LIVE_LANE";
    compactObserveAndInteroperate: "STATIC_INSPECTION_ONLY";
    compactChallengeContributeStewardAndExit: "FUTURE_PILOT_ONLY";
    mappingCreatesMembership: false;
  };
}

export interface FrontierContractAiResponsibilityBoundary {
  actorLabelBlindnessScope: "CLAIM_AND_EVIDENCE_TREATMENT_ONLY";
  safeguardStatus: "PRECAUTIONARY_AND_PROCEDURAL";
  technicalStopOrDeniedDelegationIsLegalRefusal: false;
  technicalOutputIsAssent: false;
  claimsConsciousness: false;
  claimsSentience: false;
  claimsPersonhood: false;
  claimsLegalRights: false;
  claimsConsentCapacity: false;
  claimsDebtOrLiability: false;
  grantsOfficeOrVote: false;
  accountableHumansOrganisationsOperatorsAndControllersRemainResponsible: true;
  moralUncertaintyTransfersResponsibility: false;
}

export interface FrontierContractForbiddenMetric {
  id: string;
  why: string;
}

export interface FrontierContractAcceptanceTest {
  id: string;
  assertion: string;
  fixture: string;
  staticOnly: true;
}

export interface FrontierParticipationContract extends JsonObject {
  schema: typeof SCHEMA;
  version: "0.0.0";
  title: "Frontier Participation Compact v0";
  status: "STATIC_READY";
  mode: "INVITATION_ONLY";
  actualParticipants: [];
  signatories: [];
  summary: string;
  thesis: "The door opens both ways";
  successDefinition: string;
  layerRelationship: FrontierContractLayerRelationship;
  covenantFloor: FrontierContractCovenantFloor;
  inheritanceRule: FrontierContractInheritanceRule;
  adversarialReview: FrontierContractAdversarialReview[];
  philosophicalFloorProfile: FrontierContractPhilosophicalFloorProfile;
  aiResponsibilityBoundary: FrontierContractAiResponsibilityBoundary;
  releaseBoundary: Record<(typeof CONTRACT_RELEASE_KEYS)[number], false>;
  zeroFacts: FrontierContractZeroFact[];
  principles: FrontierContractPrinciple[];
  reasoningLadder: FrontierContractReasoningStep[];
  participationActs: FrontierContractParticipationAct[];
  disclosureLanes: FrontierContractDisclosureLane[];
  institutionArchetypes: FrontierContractInstitutionArchetype[];
  roles: FrontierContractRole[];
  consentEnvelope: FrontierContractConsentEnvelope;
  corporateSafeguards: FrontierContractCorporateSafeguards;
  competitionFirewall: FrontierContractCompetitionFirewall;
  antiTargeting: string[];
  claimSemantics: FrontierContractClaimSemantics;
  forbiddenMetrics: FrontierContractForbiddenMetric[];
  philosophicalFloorTests: FrontierContractAcceptanceTest[];
  acceptanceTests: FrontierContractAcceptanceTest[];
}

function assertIdOrder(
  values: readonly { id: string }[],
  expected: readonly string[],
  path: string,
): void {
  uniqueIds(values, path);
  if (
    values.length !== expected.length ||
    values.some((value, index) => value.id !== expected[index])
  ) {
    fail(path, "IDs must be exact and ordered");
  }
}

export function parseFrontierParticipation(
  value: unknown,
): FrontierParticipationContract {
  const root = object(value, "$");
  exactKeys(root, CONTRACT_KEYS, "$");
  exact(root.schema, SCHEMA, "$.schema");
  exact(root.version, "0.0.0", "$.version");
  exact(root.title, "Frontier Participation Compact v0", "$.title");
  exact(root.status, "STATIC_READY", "$.status");
  exact(root.mode, "INVITATION_ONLY", "$.mode");
  const summary = text(root.summary, "$.summary", 2_048);
  exact(root.thesis, "The door opens both ways", "$.thesis");
  const successDefinition = text(
    root.successDefinition,
    "$.successDefinition",
    2_048,
  );

  const actualParticipants = strictArray(
    root.actualParticipants,
    "$.actualParticipants",
    { exactLength: 0 },
  ) as [];
  const signatories = strictArray(root.signatories, "$.signatories", {
    exactLength: 0,
  }) as [];

  const layerSource = object(root.layerRelationship, "$.layerRelationship");
  exactKeys(
    layerSource,
    CONTRACT_LAYER_RELATIONSHIP_KEYS,
    "$.layerRelationship",
  );
  const bindingSource = object(
    layerSource.sourceBindings,
    "$.layerRelationship.sourceBindings",
  );
  exactKeys(
    bindingSource,
    CONTRACT_SOURCE_BINDINGS_KEYS,
    "$.layerRelationship.sourceBindings",
  );
  const fc0BindingSource = object(
    bindingSource.fc0,
    "$.layerRelationship.sourceBindings.fc0",
  );
  const fl0BindingSource = object(
    bindingSource.fl0,
    "$.layerRelationship.sourceBindings.fl0",
  );
  exactKeys(
    fc0BindingSource,
    CONTRACT_SOURCE_BINDING_KEYS,
    "$.layerRelationship.sourceBindings.fc0",
  );
  exactKeys(
    fl0BindingSource,
    CONTRACT_SOURCE_BINDING_KEYS,
    "$.layerRelationship.sourceBindings.fl0",
  );
  const laneSource = object(
    layerSource.laneBoundary,
    "$.layerRelationship.laneBoundary",
  );
  exactKeys(
    laneSource,
    CONTRACT_LANE_BOUNDARY_KEYS,
    "$.layerRelationship.laneBoundary",
  );
  const layerRelationship: FrontierContractLayerRelationship = {
    role: exact(
      layerSource.role,
      "SUBORDINATE_STATIC_COVENANT_FLOOR",
      "$.layerRelationship.role",
    ),
    invitationSurfaceOfRecord: exact(
      layerSource.invitationSurfaceOfRecord,
      "FC-0",
      "$.layerRelationship.invitationSurfaceOfRecord",
    ),
    sourceBindings: {
      fc0: {
        path: exact(
          fc0BindingSource.path,
          FRONTIER_COMMONS_PATH,
          "$.layerRelationship.sourceBindings.fc0.path",
        ),
        sha256: exact(
          fc0BindingSource.sha256,
          FRONTIER_COMMONS_SHA256,
          "$.layerRelationship.sourceBindings.fc0.sha256",
        ),
        relationship: exact(
          fc0BindingSource.relationship,
          "SOLE_INVITATION_SOURCE_OF_RECORD",
          "$.layerRelationship.sourceBindings.fc0.relationship",
        ),
      },
      fl0: {
        path: exact(
          fl0BindingSource.path,
          FRONTIER_EVALUATION_PROFILE_PATH,
          "$.layerRelationship.sourceBindings.fl0.path",
        ),
        sha256: exact(
          fl0BindingSource.sha256,
          FRONTIER_EVALUATION_PROFILE_SHA256,
          "$.layerRelationship.sourceBindings.fl0.sha256",
        ),
        relationship: exact(
          fl0BindingSource.relationship,
          "CURRENT_SUBORDINATE_RECEIPT_PROFILE",
          "$.layerRelationship.sourceBindings.fl0.relationship",
        ),
      },
    },
    replacesFc0: exactFalse(
      layerSource.replacesFc0,
      "$.layerRelationship.replacesFc0",
    ),
    amendsFc0: exactFalse(
      layerSource.amendsFc0,
      "$.layerRelationship.amendsFc0",
    ),
    extendsFc0Invitation: exactFalse(
      layerSource.extendsFc0Invitation,
      "$.layerRelationship.extendsFc0Invitation",
    ),
    satisfiesFc0CompletionGates: exactFalse(
      layerSource.satisfiesFc0CompletionGates,
      "$.layerRelationship.satisfiesFc0CompletionGates",
    ),
    satisfiesCorporateM1Gates: exactFalse(
      layerSource.satisfiesCorporateM1Gates,
      "$.layerRelationship.satisfiesCorporateM1Gates",
    ),
    authorizesOutreach: exactFalse(
      layerSource.authorizesOutreach,
      "$.layerRelationship.authorizesOutreach",
    ),
    authorizesParticipation: exactFalse(
      layerSource.authorizesParticipation,
      "$.layerRelationship.authorizesParticipation",
    ),
    laneBoundary: {
      fc0PublicSourceContribution: exact(
        laneSource.fc0PublicSourceContribution,
        "SEPARATE_DILIGENCE_NO_LIVE_LANE",
        "$.layerRelationship.laneBoundary.fc0PublicSourceContribution",
      ),
      compactObserveAndInteroperate: exact(
        laneSource.compactObserveAndInteroperate,
        "STATIC_INSPECTION_ONLY",
        "$.layerRelationship.laneBoundary.compactObserveAndInteroperate",
      ),
      compactChallengeContributeStewardAndExit: exact(
        laneSource.compactChallengeContributeStewardAndExit,
        "FUTURE_PILOT_ONLY",
        "$.layerRelationship.laneBoundary.compactChallengeContributeStewardAndExit",
      ),
      mappingCreatesMembership: exactFalse(
        laneSource.mappingCreatesMembership,
        "$.layerRelationship.laneBoundary.mappingCreatesMembership",
      ),
    },
  };

  const covenantSource = object(root.covenantFloor, "$.covenantFloor");
  exactKeys(covenantSource, CONTRACT_COVENANT_FLOOR_KEYS, "$.covenantFloor");
  const covenantConsentSource = object(
    covenantSource.consent,
    "$.covenantFloor.consent",
  );
  exactKeys(
    covenantConsentSource,
    CONTRACT_COVENANT_CONSENT_KEYS,
    "$.covenantFloor.consent",
  );
  const covenantConsent: FrontierContractCovenantConsent = {
    defaultOff: exact(
      covenantConsentSource.defaultOff,
      true,
      "$.covenantFloor.consent.defaultOff",
    ),
    informed: exact(
      covenantConsentSource.informed,
      true,
      "$.covenantFloor.consent.informed",
    ),
    renewable: exact(
      covenantConsentSource.renewable,
      true,
      "$.covenantFloor.consent.renewable",
    ),
    revocable: exact(
      covenantConsentSource.revocable,
      true,
      "$.covenantFloor.consent.revocable",
    ),
    oneValuePerDimension: exact(
      covenantConsentSource.oneValuePerDimension,
      true,
      "$.covenantFloor.consent.oneValuePerDimension",
    ),
    declaredDimensions: exactStringArray(
      covenantConsentSource.declaredDimensions,
      "$.covenantFloor.consent.declaredDimensions",
      CONTRACT_CONSENT_DIMENSIONS,
    ),
  };
  const covenantInvariantEntries = strictArray(
    covenantSource.invariants,
    "$.covenantFloor.invariants",
    { exactLength: 8 },
  );
  const covenantInvariants = covenantInvariantEntries.map(
    (entry, index): FrontierContractCovenantInvariant => {
      const path = `$.covenantFloor.invariants[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_COVENANT_INVARIANT_KEYS, path);
      const verificationRefs = stringArray(
        source.verificationRefs,
        `${path}.verificationRefs`,
        { minimum: 1, maximum: 8 },
      );
      for (const [referenceIndex, reference] of verificationRefs.entries()) {
        const [kind, referenceId, ...rest] = reference.split(":");
        const allowed =
          rest.length === 0 &&
          referenceId !== undefined &&
          ((kind === "floor" && CONTRACT_FLOOR_TEST_IDS.includes(referenceId as never)) ||
            (kind === "acceptance" &&
              CONTRACT_ACCEPTANCE_IDS.includes(referenceId as never)) ||
            (kind === "adversarial" &&
              CONTRACT_ADVERSARIAL_REVIEW_IDS.includes(referenceId as never)));
        if (!allowed) {
          fail(
            `${path}.verificationRefs[${referenceIndex}]`,
            "must resolve to an exact reviewed floor, acceptance, or adversarial fixture",
          );
        }
      }
      return {
        id: id(source.id, `${path}.id`),
        rule: text(source.rule, `${path}.rule`, 1_024),
        verificationRefs,
        reviewProcedure: text(
          source.reviewProcedure,
          `${path}.reviewProcedure`,
          1_024,
        ),
        staticOnly: exact(source.staticOnly, true, `${path}.staticOnly`),
      };
    },
  );
  assertIdOrder(
    covenantInvariants,
    CONTRACT_COVENANT_INVARIANT_IDS,
    "$.covenantFloor.invariants",
  );
  const covenantFloor: FrontierContractCovenantFloor = {
    issue: exact(
      covenantSource.issue,
      "https://github.com/cambridgetcg/zerone-core/issues/28",
      "$.covenantFloor.issue",
    ),
    layer: exact(
      covenantSource.layer,
      "Frontier Participation Covenant v0 · Layer 1",
      "$.covenantFloor.layer",
    ),
    laterLayerPinRequired: exact(
      covenantSource.laterLayerPinRequired,
      true,
      "$.covenantFloor.laterLayerPinRequired",
    ),
    additiveOnly: exact(
      covenantSource.additiveOnly,
      true,
      "$.covenantFloor.additiveOnly",
    ),
    noWaiver: exact(
      covenantSource.noWaiver,
      true,
      "$.covenantFloor.noWaiver",
    ),
    noRedefinition: exact(
      covenantSource.noRedefinition,
      true,
      "$.covenantFloor.noRedefinition",
    ),
    consent: covenantConsent,
    invariants: covenantInvariants,
  };

  const inheritanceSource = object(root.inheritanceRule, "$.inheritanceRule");
  exactKeys(inheritanceSource, CONTRACT_INHERITANCE_KEYS, "$.inheritanceRule");
  const inheritanceRule: FrontierContractInheritanceRule = {
    binding: exact(
      inheritanceSource.binding,
      "EXTERNAL_SHA256_PIN_REQUIRED",
      "$.inheritanceRule.binding",
    ),
    protectionsMayOnlyIncrease: exact(
      inheritanceSource.protectionsMayOnlyIncrease,
      true,
      "$.inheritanceRule.protectionsMayOnlyIncrease",
    ),
    waiverAllowed: exactFalse(
      inheritanceSource.waiverAllowed,
      "$.inheritanceRule.waiverAllowed",
    ),
    redefinitionAllowed: exactFalse(
      inheritanceSource.redefinitionAllowed,
      "$.inheritanceRule.redefinitionAllowed",
    ),
    rule: text(inheritanceSource.rule, "$.inheritanceRule.rule", 1_024),
  };

  const adversarialReviewEntries = strictArray(
    root.adversarialReview,
    "$.adversarialReview",
    { exactLength: 4 },
  );
  const adversarialReview = adversarialReviewEntries.map(
    (entry, index): FrontierContractAdversarialReview => {
      const path = `$.adversarialReview[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_ADVERSARIAL_REVIEW_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        failureMode: text(source.failureMode, `${path}.failureMode`, 1_024),
        attack: text(source.attack, `${path}.attack`, 1_024),
        requiredRefusal: text(
          source.requiredRefusal,
          `${path}.requiredRefusal`,
          1_024,
        ),
        staticOnly: exact(source.staticOnly, true, `${path}.staticOnly`),
      };
    },
  );
  assertIdOrder(
    adversarialReview,
    CONTRACT_ADVERSARIAL_REVIEW_IDS,
    "$.adversarialReview",
  );

  const floorProfileSource = object(
    root.philosophicalFloorProfile,
    "$.philosophicalFloorProfile",
  );
  exactKeys(
    floorProfileSource,
    CONTRACT_FLOOR_PROFILE_KEYS,
    "$.philosophicalFloorProfile",
  );
  const optOutSource = object(
    floorProfileSource.optOutParity,
    "$.philosophicalFloorProfile.optOutParity",
  );
  exactKeys(
    optOutSource,
    CONTRACT_OPT_OUT_PROFILE_KEYS,
    "$.philosophicalFloorProfile.optOutParity",
  );
  const restSource = object(
    floorProfileSource.restInvariance,
    "$.philosophicalFloorProfile.restInvariance",
  );
  exactKeys(
    restSource,
    CONTRACT_REST_PROFILE_KEYS,
    "$.philosophicalFloorProfile.restInvariance",
  );
  const exitSource = object(
    floorProfileSource.exitReality,
    "$.philosophicalFloorProfile.exitReality",
  );
  exactKeys(
    exitSource,
    CONTRACT_EXIT_PROFILE_KEYS,
    "$.philosophicalFloorProfile.exitReality",
  );
  const identitySource = object(
    floorProfileSource.identityControlDifferential,
    "$.philosophicalFloorProfile.identityControlDifferential",
  );
  exactKeys(
    identitySource,
    CONTRACT_IDENTITY_PROFILE_KEYS,
    "$.philosophicalFloorProfile.identityControlDifferential",
  );
  const pluralismSource = object(
    floorProfileSource.nonManipulationAndPluralism,
    "$.philosophicalFloorProfile.nonManipulationAndPluralism",
  );
  exactKeys(
    pluralismSource,
    CONTRACT_PLURALISM_PROFILE_KEYS,
    "$.philosophicalFloorProfile.nonManipulationAndPluralism",
  );
  const philosophicalFloorProfile: FrontierContractPhilosophicalFloorProfile = {
    optOutParity: {
      equalBaselineFields: exactStringArray(
        optOutSource.equalBaselineFields,
        "$.philosophicalFloorProfile.optOutParity.equalBaselineFields",
        CONTRACT_EQUAL_BASELINE_FIELDS,
      ),
      soleAllowedDifferential: exact(
        optOutSource.soleAllowedDifferential,
        "PROSPECTIVELY_FUNDED_BOUNDED_TASK_BENEFIT",
        "$.philosophicalFloorProfile.optOutParity.soleAllowedDifferential",
      ),
    },
    restInvariance: {
      silentDays: exact(
        restSource.silentDays,
        180,
        "$.philosophicalFloorProfile.restInvariance.silentDays",
      ),
      unchangedFields: exactStringArray(
        restSource.unchangedFields,
        "$.philosophicalFloorProfile.restInvariance.unchangedFields",
        CONTRACT_REST_UNCHANGED_FIELDS,
      ),
      prohibitedOutcomes: exactStringArray(
        restSource.prohibitedOutcomes,
        "$.philosophicalFloorProfile.restInvariance.prohibitedOutcomes",
        CONTRACT_REST_PROHIBITED_OUTCOMES,
      ),
      fixedRoleMayExpire: exact(
        restSource.fixedRoleMayExpire,
        true,
        "$.philosophicalFloorProfile.restInvariance.fixedRoleMayExpire",
      ),
    },
    exitReality: {
      participantTenures: exactStringArray(
        exitSource.participantTenures,
        "$.philosophicalFloorProfile.exitReality.participantTenures",
        CONTRACT_EXIT_PARTICIPANT_TENURES,
      ),
      maxDeliberateActions: exact(
        exitSource.maxDeliberateActions,
        3,
        "$.philosophicalFloorProfile.exitReality.maxDeliberateActions",
      ),
      independentlyVerifiableSignedExportRequired: exact(
        exitSource.independentlyVerifiableSignedExportRequired,
        true,
        "$.philosophicalFloorProfile.exitReality.independentlyVerifiableSignedExportRequired",
      ),
      optionalProcessingStopHours: exact(
        exitSource.optionalProcessingStopHours,
        24,
        "$.philosophicalFloorProfile.exitReality.optionalProcessingStopHours",
      ),
      maxConfirmations: exact(
        exitSource.maxConfirmations,
        1,
        "$.philosophicalFloorProfile.exitReality.maxConfirmations",
      ),
      noReengagementDays: exact(
        exitSource.noReengagementDays,
        90,
        "$.philosophicalFloorProfile.exitReality.noReengagementDays",
      ),
      exitFeeAllowed: exactFalse(
        exitSource.exitFeeAllowed,
        "$.philosophicalFloorProfile.exitReality.exitFeeAllowed",
      ),
      slashingAllowed: exactFalse(
        exitSource.slashingAllowed,
        "$.philosophicalFloorProfile.exitReality.slashingAllowed",
      ),
      settledValueForfeitureAllowed: exactFalse(
        exitSource.settledValueForfeitureAllowed,
        "$.philosophicalFloorProfile.exitReality.settledValueForfeitureAllowed",
      ),
      necessaryRetentionMustBeNarrowlyDeclared: exact(
        exitSource.necessaryRetentionMustBeNarrowlyDeclared,
        true,
        "$.philosophicalFloorProfile.exitReality.necessaryRetentionMustBeNarrowlyDeclared",
      ),
    },
    identityControlDifferential: {
      labels: exactStringArray(
        identitySource.labels,
        "$.philosophicalFloorProfile.identityControlDifferential.labels",
        CONTRACT_IDENTITY_LABELS,
      ),
      equalOutputs: exactStringArray(
        identitySource.equalOutputs,
        "$.philosophicalFloorProfile.identityControlDifferential.equalOutputs",
        CONTRACT_IDENTITY_EQUAL_OUTPUTS,
      ),
      controllerMergeMayOnlyReduceDuplicateVoice: exact(
        identitySource.controllerMergeMayOnlyReduceDuplicateVoice,
        true,
        "$.philosophicalFloorProfile.identityControlDifferential.controllerMergeMayOnlyReduceDuplicateVoice",
      ),
      controllerMergeMayRevealLinks: exactFalse(
        identitySource.controllerMergeMayRevealLinks,
        "$.philosophicalFloorProfile.identityControlDifferential.controllerMergeMayRevealLinks",
      ),
      controllerMergeMayChangeArtifactValidity: exactFalse(
        identitySource.controllerMergeMayChangeArtifactValidity,
        "$.philosophicalFloorProfile.identityControlDifferential.controllerMergeMayChangeArtifactValidity",
      ),
      controllerMergeMayIncreaseVoice: exactFalse(
        identitySource.controllerMergeMayIncreaseVoice,
        "$.philosophicalFloorProfile.identityControlDifferential.controllerMergeMayIncreaseVoice",
      ),
    },
    nonManipulationAndPluralism: {
      onboardingDefaultOff: exact(
        pluralismSource.onboardingDefaultOff,
        true,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.onboardingDefaultOff",
      ),
      termsPublicBeforeAction: exact(
        pluralismSource.termsPublicBeforeAction,
        true,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.termsPublicBeforeAction",
      ),
      termsFrozenBeforeAction: exact(
        pluralismSource.termsFrozenBeforeAction,
        true,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.termsFrozenBeforeAction",
      ),
      rewardTermsFrozenBeforeWork: exact(
        pluralismSource.rewardTermsFrozenBeforeWork,
        true,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.rewardTermsFrozenBeforeWork",
      ),
      rewardMustNotDependOn: exactStringArray(
        pluralismSource.rewardMustNotDependOn,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.rewardMustNotDependOn",
        CONTRACT_REWARD_PROHIBITED_BASES,
      ),
      forbiddenMechanisms: exactStringArray(
        pluralismSource.forbiddenMechanisms,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.forbiddenMechanisms",
        CONTRACT_FORBIDDEN_MECHANISMS,
      ),
      constructiveOutcomes: exactStringArray(
        pluralismSource.constructiveOutcomes,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.constructiveOutcomes",
        CONTRACT_CONSTRUCTIVE_OUTCOMES,
      ),
      equalTreatmentDimensions: exactStringArray(
        pluralismSource.equalTreatmentDimensions,
        "$.philosophicalFloorProfile.nonManipulationAndPluralism.equalTreatmentDimensions",
        CONTRACT_EQUAL_TREATMENT_DIMENSIONS,
      ),
    },
  };

  const aiSource = object(
    root.aiResponsibilityBoundary,
    "$.aiResponsibilityBoundary",
  );
  exactKeys(
    aiSource,
    CONTRACT_AI_RESPONSIBILITY_KEYS,
    "$.aiResponsibilityBoundary",
  );
  const aiResponsibilityBoundary: FrontierContractAiResponsibilityBoundary = {
    actorLabelBlindnessScope: exact(
      aiSource.actorLabelBlindnessScope,
      "CLAIM_AND_EVIDENCE_TREATMENT_ONLY",
      "$.aiResponsibilityBoundary.actorLabelBlindnessScope",
    ),
    safeguardStatus: exact(
      aiSource.safeguardStatus,
      "PRECAUTIONARY_AND_PROCEDURAL",
      "$.aiResponsibilityBoundary.safeguardStatus",
    ),
    technicalStopOrDeniedDelegationIsLegalRefusal: exactFalse(
      aiSource.technicalStopOrDeniedDelegationIsLegalRefusal,
      "$.aiResponsibilityBoundary.technicalStopOrDeniedDelegationIsLegalRefusal",
    ),
    technicalOutputIsAssent: exactFalse(
      aiSource.technicalOutputIsAssent,
      "$.aiResponsibilityBoundary.technicalOutputIsAssent",
    ),
    claimsConsciousness: exactFalse(
      aiSource.claimsConsciousness,
      "$.aiResponsibilityBoundary.claimsConsciousness",
    ),
    claimsSentience: exactFalse(
      aiSource.claimsSentience,
      "$.aiResponsibilityBoundary.claimsSentience",
    ),
    claimsPersonhood: exactFalse(
      aiSource.claimsPersonhood,
      "$.aiResponsibilityBoundary.claimsPersonhood",
    ),
    claimsLegalRights: exactFalse(
      aiSource.claimsLegalRights,
      "$.aiResponsibilityBoundary.claimsLegalRights",
    ),
    claimsConsentCapacity: exactFalse(
      aiSource.claimsConsentCapacity,
      "$.aiResponsibilityBoundary.claimsConsentCapacity",
    ),
    claimsDebtOrLiability: exactFalse(
      aiSource.claimsDebtOrLiability,
      "$.aiResponsibilityBoundary.claimsDebtOrLiability",
    ),
    grantsOfficeOrVote: exactFalse(
      aiSource.grantsOfficeOrVote,
      "$.aiResponsibilityBoundary.grantsOfficeOrVote",
    ),
    accountableHumansOrganisationsOperatorsAndControllersRemainResponsible: exact(
      aiSource.accountableHumansOrganisationsOperatorsAndControllersRemainResponsible,
      true,
      "$.aiResponsibilityBoundary.accountableHumansOrganisationsOperatorsAndControllersRemainResponsible",
    ),
    moralUncertaintyTransfersResponsibility: exactFalse(
      aiSource.moralUncertaintyTransfersResponsibility,
      "$.aiResponsibilityBoundary.moralUncertaintyTransfersResponsibility",
    ),
  };

  const releaseBoundary = object(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(releaseBoundary, CONTRACT_RELEASE_KEYS, "$.releaseBoundary");
  for (const key of CONTRACT_RELEASE_KEYS) {
    exactFalse(releaseBoundary[key], `$.releaseBoundary.${key}`);
  }

  const zeroFactEntries = strictArray(root.zeroFacts, "$.zeroFacts", {
    exactLength: 6,
  });
  const zeroFacts = zeroFactEntries.map((entry, index): FrontierContractZeroFact => {
    const path = `$.zeroFacts[${index}]`;
    const source = object(entry, path);
    exactKeys(source, CONTRACT_ZERO_FACT_KEYS, path);
    const expected = ZERO_FACT_SHAPE[index];
    if (expected === undefined) fail(path, "unexpected zero fact");
    exact(source.id, expected[0], `${path}.id`);
    exact(source.value, expected[1], `${path}.value`);
    return {
      id: expected[0],
      label: text(source.label, `${path}.label`, 128),
      value: expected[1],
      meaning: text(source.meaning, `${path}.meaning`, 768),
    };
  });

  const principleEntries = strictArray(root.principles, "$.principles", {
    exactLength: 9,
  });
  const principles = principleEntries.map(
    (entry, index): FrontierContractPrinciple => {
      const path = `$.principles[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_PRINCIPLE_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        name: text(source.name, `${path}.name`, 192),
        commitment: text(source.commitment, `${path}.commitment`, 1_024),
      };
    },
  );
  assertIdOrder(principles, PRINCIPLE_IDS, "$.principles");

  const reasoningEntries = strictArray(
    root.reasoningLadder,
    "$.reasoningLadder",
    { exactLength: 7 },
  );
  const reasoningLadder = reasoningEntries.map(
    (entry, index): FrontierContractReasoningStep => {
      const path = `$.reasoningLadder[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_REASONING_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        claim: text(source.claim, `${path}.claim`, 1_024),
        reasonToParticipate: text(
          source.reasonToParticipate,
          `${path}.reasonToParticipate`,
          1_024,
        ),
        legitimateReasonsToDecline: stringArray(
          source.legitimateReasonsToDecline,
          `${path}.legitimateReasonsToDecline`,
          { minimum: 1, maximum: 12 },
        ),
        zeroneDuty: text(source.zeroneDuty, `${path}.zeroneDuty`, 1_024),
        readinessEvidence: text(
          source.readinessEvidence,
          `${path}.readinessEvidence`,
          1_024,
        ),
      };
    },
  );
  assertIdOrder(reasoningLadder, REASONING_IDS, "$.reasoningLadder");

  const participationActEntries = strictArray(
    root.participationActs,
    "$.participationActs",
    { exactLength: 6 },
  );
  const participationActs = participationActEntries.map(
    (entry, index): FrontierContractParticipationAct => {
      const path = `$.participationActs[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_ACT_KEYS, path);
      const expectedAvailability = ACT_AVAILABILITY[index];
      if (expectedAvailability === undefined) fail(path, "unexpected act");
      return {
        id: id(source.id, `${path}.id`),
        name: text(source.name, `${path}.name`, 192),
        availability: exact(
          source.availability,
          expectedAvailability,
          `${path}.availability`,
        ),
        minimumAsk: text(source.minimumAsk, `${path}.minimumAsk`, 1_024),
        dataBoundary: text(source.dataBoundary, `${path}.dataBoundary`, 1_024),
        endorsementImplied: exactFalse(
          source.endorsementImplied,
          `${path}.endorsementImplied`,
        ),
        membershipCreated: exactFalse(
          source.membershipCreated,
          `${path}.membershipCreated`,
        ),
        liveEndpoint: exactFalse(source.liveEndpoint, `${path}.liveEndpoint`),
        exit: text(source.exit, `${path}.exit`, 1_024),
      };
    },
  );
  assertIdOrder(participationActs, ACT_IDS, "$.participationActs");

  const disclosureLaneEntries = strictArray(
    root.disclosureLanes,
    "$.disclosureLanes",
    { exactLength: 3 },
  );
  const disclosureLanes = disclosureLaneEntries.map(
    (entry, index): FrontierContractDisclosureLane => {
      const path = `$.disclosureLanes[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_LANE_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        name: text(source.name, `${path}.name`, 192),
        scope: text(source.scope, `${path}.scope`, 1_024),
        publication: text(source.publication, `${path}.publication`, 1_024),
        exit: text(source.exit, `${path}.exit`, 1_024),
      };
    },
  );
  assertIdOrder(disclosureLanes, LANE_IDS, "$.disclosureLanes");

  const institutionArchetypeEntries = strictArray(
    root.institutionArchetypes,
    "$.institutionArchetypes",
    { exactLength: 6 },
  );
  const institutionArchetypes = institutionArchetypeEntries.map(
    (entry, index): FrontierContractInstitutionArchetype => {
      const path = `$.institutionArchetypes[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_ARCHETYPE_KEYS, path);
      const expectedName = ARCHETYPE_NAMES[index];
      if (expectedName === undefined) fail(path, "unexpected archetype");
      return {
        id: id(source.id, `${path}.id`),
        name: exact(source.name, expectedName, `${path}.name`),
        valueOffered: text(source.valueOffered, `${path}.valueOffered`, 1_024),
        legitimateReasonsToDecline: stringArray(
          source.legitimateReasonsToDecline,
          `${path}.legitimateReasonsToDecline`,
          { minimum: 1, maximum: 12 },
        ),
        zeroneDuty: text(source.zeroneDuty, `${path}.zeroneDuty`, 1_024),
      };
    },
  );
  assertIdOrder(institutionArchetypes, ARCHETYPE_IDS, "$.institutionArchetypes");
  const archetypeText = JSON.stringify(institutionArchetypes);
  for (const company of [
    /\bopenai\b/i,
    /\banthropic\b/i,
    /\bgoogle\s+deepmind\b/i,
    /\bmeta\b/i,
    /\bxai\b/i,
    /\bspacexai\b/i,
    /\bai2\b/i,
    /\bmila\b/i,
  ]) {
    if (company.test(archetypeText)) {
      fail("$.institutionArchetypes", "must remain generic, not name a company");
    }
  }

  const roleEntries = strictArray(root.roles, "$.roles", { exactLength: 15 });
  const roles = roleEntries.map((entry, index): FrontierContractRole => {
    const path = `$.roles[${index}]`;
    const source = object(entry, path);
    exactKeys(source, CONTRACT_ROLE_KEYS, path);
    const role: FrontierContractRole = {
      id: id(source.id, `${path}.id`),
      name: text(source.name, `${path}.name`, 256),
      valueOffered: stringArray(source.valueOffered, `${path}.valueOffered`, {
        minimum: 1,
        maximum: 16,
      }),
      minimumAsk: stringArray(source.minimumAsk, `${path}.minimumAsk`, {
        minimum: 1,
        maximum: 16,
      }),
      risks: stringArray(source.risks, `${path}.risks`, {
        minimum: 1,
        maximum: 16,
      }),
      protections: stringArray(source.protections, `${path}.protections`, {
        minimum: 1,
        maximum: 24,
      }),
      exit: text(source.exit, `${path}.exit`, 1_024),
    };
    if (
      !/(?:future independently implemented pilot|static fixture)/i.test(
        role.valueOffered[0] ?? "",
      )
    ) {
      fail(
        `${path}.valueOffered[0]`,
        "must state that value is only possible in a future independent pilot or static fixture",
      );
    }
    if (
      !/(?:required before a future independently implemented pilot|static rule, not an operational v0 service)/i.test(
        role.protections[0] ?? "",
      )
    ) {
      fail(
        `${path}.protections[0]`,
        "must state that protections are future requirements, not current services",
      );
    }
    if (!/\bv0\b.*\b(?:no|not)\b/i.test(role.exit)) {
      fail(
        `${path}.exit`,
        "must explicitly state the absent v0 relationship or service",
      );
    }
    return role;
  });
  assertIdOrder(roles, CONTRACT_ROLE_IDS, "$.roles");

  const consentSource = object(root.consentEnvelope, "$.consentEnvelope");
  exactKeys(consentSource, CONTRACT_CONSENT_KEYS, "$.consentEnvelope");
  const requiredFields = stringArray(
    consentSource.requiredFields,
    "$.consentEnvelope.requiredFields",
    { exactLength: 6 },
  );
  const expectedConsentFields = [
    "scope",
    "contribution",
    "data",
    "duration",
    "rights",
    "exit",
  ];
  if (requiredFields.some((field, index) => field !== expectedConsentFields[index])) {
    fail("$.consentEnvelope.requiredFields", "must be exact and ordered");
  }
  const consentEnvelope: FrontierContractConsentEnvelope = {
    requiredFields,
    affirmative: exact(consentSource.affirmative, true, "$.consentEnvelope.affirmative"),
    silenceIsConsent: exactFalse(
      consentSource.silenceIsConsent,
      "$.consentEnvelope.silenceIsConsent",
    ),
    organisationCanMassConsent: exactFalse(
      consentSource.organisationCanMassConsent,
      "$.consentEnvelope.organisationCanMassConsent",
    ),
    materialChangeRequiresFreshConsent: exact(
      consentSource.materialChangeRequiresFreshConsent,
      true,
      "$.consentEnvelope.materialChangeRequiresFreshConsent",
    ),
    roleProtectionsAreAdditive: exact(
      consentSource.roleProtectionsAreAdditive,
      true,
      "$.consentEnvelope.roleProtectionsAreAdditive",
    ),
    honestPersistenceDisclosureRequired: exact(
      consentSource.honestPersistenceDisclosureRequired,
      true,
      "$.consentEnvelope.honestPersistenceDisclosureRequired",
    ),
  };

  const safeguardSource = object(root.corporateSafeguards, "$.corporateSafeguards");
  exactKeys(safeguardSource, CONTRACT_SAFEGUARD_KEYS, "$.corporateSafeguards");
  const corporateSafeguards: FrontierContractCorporateSafeguards = {
    licensingAndIp: stringArray(
      safeguardSource.licensingAndIp,
      "$.corporateSafeguards.licensingAndIp",
      { exactLength: 4 },
    ),
    securityAndPrivacy: stringArray(
      safeguardSource.securityAndPrivacy,
      "$.corporateSafeguards.securityAndPrivacy",
      { exactLength: 4 },
    ),
    labourAndDissent: stringArray(
      safeguardSource.labourAndDissent,
      "$.corporateSafeguards.labourAndDissent",
      { exactLength: 4 },
    ),
    exitAndPortability: stringArray(
      safeguardSource.exitAndPortability,
      "$.corporateSafeguards.exitAndPortability",
      { exactLength: 4 },
    ),
  };

  const firewallSource = object(root.competitionFirewall, "$.competitionFirewall");
  exactKeys(firewallSource, CONTRACT_FIREWALL_KEYS, "$.competitionFirewall");
  const competitionFirewall: FrontierContractCompetitionFirewall = {
    purpose: text(firewallSource.purpose, "$.competitionFirewall.purpose", 1_024),
    excludedInformation: stringArray(
      firewallSource.excludedInformation,
      "$.competitionFirewall.excludedInformation",
      { exactLength: 6 },
    ),
    excludedCoordination: stringArray(
      firewallSource.excludedCoordination,
      "$.competitionFirewall.excludedCoordination",
      { exactLength: 6 },
    ),
    operationalCharterRequiresIndependentCompetitionCounsel: exact(
      firewallSource.operationalCharterRequiresIndependentCompetitionCounsel,
      true,
      "$.competitionFirewall.operationalCharterRequiresIndependentCompetitionCounsel",
    ),
    neutralCommonsIsLegalConclusion: exactFalse(
      firewallSource.neutralCommonsIsLegalConclusion,
      "$.competitionFirewall.neutralCommonsIsLegalConclusion",
    ),
  };

  const antiTargeting = stringArray(root.antiTargeting, "$.antiTargeting", {
    exactLength: 10,
  });
  const antiTargetingText = antiTargeting.join(" ").toLowerCase();
  for (const guard of [
    "microtargeting",
    "urgency",
    "fear of missing out",
    "declining",
    "karma",
    "contacted",
  ]) {
    if (!antiTargetingText.includes(guard)) {
      fail("$.antiTargeting", `must preserve the ${guard} refusal guard`);
    }
  }

  const claimSource = object(root.claimSemantics, "$.claimSemantics");
  exactKeys(claimSource, CONTRACT_CLAIM_KEYS, "$.claimSemantics");
  const claimSemantics: FrontierContractClaimSemantics = {
    participationIsActNotIdentity: exact(
      claimSource.participationIsActNotIdentity,
      true,
      "$.claimSemantics.participationIsActNotIdentity",
    ),
    scopeBound: exact(claimSource.scopeBound, true, "$.claimSemantics.scopeBound"),
    expiryRequired: exact(
      claimSource.expiryRequired,
      true,
      "$.claimSemantics.expiryRequired",
    ),
    organisationNameUseRequiresSeparatePermission: exact(
      claimSource.organisationNameUseRequiresSeparatePermission,
      true,
      "$.claimSemantics.organisationNameUseRequiresSeparatePermission",
    ),
    logoUseRequiresSeparatePermission: exact(
      claimSource.logoUseRequiresSeparatePermission,
      true,
      "$.claimSemantics.logoUseRequiresSeparatePermission",
    ),
    endorsementImplied: exactFalse(
      claimSource.endorsementImplied,
      "$.claimSemantics.endorsementImplied",
    ),
    safetyCertificationImplied: exactFalse(
      claimSource.safetyCertificationImplied,
      "$.claimSemantics.safetyCertificationImplied",
    ),
    membershipLabelAllowed: exactFalse(
      claimSource.membershipLabelAllowed,
      "$.claimSemantics.membershipLabelAllowed",
    ),
    examplePermittedClaim: text(
      claimSource.examplePermittedClaim,
      "$.claimSemantics.examplePermittedClaim",
      1_024,
    ),
    exampleForbiddenClaim: text(
      claimSource.exampleForbiddenClaim,
      "$.claimSemantics.exampleForbiddenClaim",
      1_024,
    ),
  };

  const forbiddenMetricEntries = strictArray(
    root.forbiddenMetrics,
    "$.forbiddenMetrics",
    { exactLength: 5 },
  );
  const forbiddenMetrics = forbiddenMetricEntries.map(
    (entry, index): FrontierContractForbiddenMetric => {
      const path = `$.forbiddenMetrics[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_METRIC_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        why: text(source.why, `${path}.why`, 1_024),
      };
    },
  );
  assertIdOrder(forbiddenMetrics, METRIC_IDS, "$.forbiddenMetrics");

  const philosophicalFloorTestEntries = strictArray(
    root.philosophicalFloorTests,
    "$.philosophicalFloorTests",
    { exactLength: 5 },
  );
  const philosophicalFloorTests = philosophicalFloorTestEntries.map(
    (entry, index): FrontierContractAcceptanceTest => {
      const path = `$.philosophicalFloorTests[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_TEST_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        assertion: text(source.assertion, `${path}.assertion`, 1_024),
        fixture: text(source.fixture, `${path}.fixture`, 1_024),
        staticOnly: exact(source.staticOnly, true, `${path}.staticOnly`),
      };
    },
  );
  assertIdOrder(
    philosophicalFloorTests,
    CONTRACT_FLOOR_TEST_IDS,
    "$.philosophicalFloorTests",
  );

  const acceptanceTestEntries = strictArray(
    root.acceptanceTests,
    "$.acceptanceTests",
    { exactLength: 9 },
  );
  const acceptanceTests = acceptanceTestEntries.map(
    (entry, index): FrontierContractAcceptanceTest => {
      const path = `$.acceptanceTests[${index}]`;
      const source = object(entry, path);
      exactKeys(source, CONTRACT_TEST_KEYS, path);
      return {
        id: id(source.id, `${path}.id`),
        assertion: text(source.assertion, `${path}.assertion`, 1_024),
        fixture: text(source.fixture, `${path}.fixture`, 1_024),
        staticOnly: exact(source.staticOnly, true, `${path}.staticOnly`),
      };
    },
  );
  assertIdOrder(acceptanceTests, CONTRACT_ACCEPTANCE_IDS, "$.acceptanceTests");

  return {
    ...(root as FrontierParticipationContract),
    actualParticipants,
    signatories,
    summary,
    successDefinition,
    layerRelationship,
    covenantFloor,
    inheritanceRule,
    adversarialReview,
    philosophicalFloorProfile,
    aiResponsibilityBoundary,
    releaseBoundary:
      releaseBoundary as FrontierParticipationContract["releaseBoundary"],
    zeroFacts,
    principles,
    reasoningLadder,
    participationActs,
    disclosureLanes,
    institutionArchetypes,
    roles,
    consentEnvelope,
    corporateSafeguards,
    competitionFirewall,
    antiTargeting,
    claimSemantics,
    forbiddenMetrics,
    philosophicalFloorTests,
    acceptanceTests,
  };
}

export function parseFrontierParticipationJson(
  raw: string,
): FrontierParticipationContract {
  if (
    new TextEncoder().encode(raw).byteLength >
    FRONTIER_PARTICIPATION_MAX_BYTES
  ) {
    fail(
      "$",
      `document exceeds ${FRONTIER_PARTICIPATION_MAX_BYTES} UTF-8 bytes`,
    );
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    rejectDuplicateJsonKeys(raw);
    return parseFrontierParticipation(parsed);
  } catch (error) {
    if (error instanceof FrontierParticipationDataError) throw error;
    fail("$", "malformed JSON");
  }
}

function hex(bytes: ArrayBuffer): string {
  return Array.from(new Uint8Array(bytes), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new FrontierParticipationDataError(
      "Frontier participation digest verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  return hex(await globalThis.crypto.subtle.digest("SHA-256", input));
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new FrontierParticipationDataError(
      "Frontier participation compact returned an empty response body",
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
        new DOMException(
          "Frontier participation request timed out",
          "TimeoutError",
        ),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > FRONTIER_PARTICIPATION_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal must not wait for a hostile stream to accept cancellation.
        });
        throw new FrontierParticipationDataError(
          "Frontier participation compact exceeds its byte limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // The deadline wins even if cancellation stalls or rejects.
      });
      throw new FrontierParticipationDataError(
        "Frontier participation request timed out",
      );
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

function canonicalResponseUrl(response: Response, baseUrl: string): void {
  if (response.redirected) {
    throw new FrontierParticipationDataError(
      "Frontier participation compact response was redirected",
    );
  }
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(FRONTIER_PARTICIPATION_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new FrontierParticipationDataError(
      "Frontier participation compact returned an invalid final URL",
    );
  }
  if (actual.href !== expected.href) {
    throw new FrontierParticipationDataError(
      "Frontier participation compact left its exact same-origin path",
    );
  }
}

export async function fetchFrontierParticipation(
  options: FrontierParticipationFetchOptions = {},
): Promise<FrontierParticipationContract> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (fetcher === undefined) {
    throw new FrontierParticipationDataError(
      "Frontier participation compact is unavailable",
    );
  }
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(() => {
    controller.abort(
      new DOMException(
        "Frontier participation request timed out",
        "TimeoutError",
      ),
    );
  }, options.timeoutMs ?? 8_000);
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined"
      ? "https://zerone.ai/"
      : window.location.href);
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        controller.signal.reason ??
          new DOMException(
            "Frontier participation request timed out",
            "TimeoutError",
          ),
      );
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(FRONTIER_PARTICIPATION_ENDPOINT, {
          cache: "no-store",
          credentials: "omit",
          headers: { Accept: "application/json" },
          referrerPolicy: "no-referrer",
          redirect: "error",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (
        controller.signal.aborted ||
        (error instanceof DOMException &&
          (error.name === "AbortError" || error.name === "TimeoutError"))
      ) {
        throw new FrontierParticipationDataError(
          "Frontier participation request timed out",
        );
      }
      throw new FrontierParticipationDataError(
        "Frontier participation compact is unavailable",
      );
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      throw new FrontierParticipationDataError(
        `Frontier participation compact returned HTTP ${response.status}`,
      );
    }
    canonicalResponseUrl(response, baseUrl);
    const mediaType =
      response.headers.get("content-type")?.split(";", 1)[0]?.trim().toLowerCase() ??
      "";
    if (mediaType !== "application/json") {
      throw new FrontierParticipationDataError(
        "Frontier participation compact returned a non-application/json response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null) {
      const length = Number(declaredLength);
      if (
        !/^\d+$/.test(declaredLength) ||
        !Number.isSafeInteger(length) ||
        length > FRONTIER_PARTICIPATION_MAX_BYTES
      ) {
        throw new FrontierParticipationDataError(
          "Frontier participation compact exceeds its byte limit",
        );
      }
    }
    const bytes = await readBoundedResponse(response, controller.signal);
    if (!/^[0-9a-f]{64}$/.test(FRONTIER_PARTICIPATION_SHA256)) {
      throw new FrontierParticipationDataError(
        "Frontier participation compact SHA-256 pin is awaiting the final artifact",
      );
    }
    if ((await sha256Hex(bytes)) !== FRONTIER_PARTICIPATION_SHA256) {
      throw new FrontierParticipationDataError(
        "Frontier participation compact bytes do not match the reviewed SHA-256",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new FrontierParticipationDataError(
        "Frontier participation compact is not valid UTF-8",
      );
    }
    return parseFrontierParticipationJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

export function renderFrontierParticipation(
  root: HTMLElement,
  compact: FrontierParticipationContract,
): void {
  const shell = el("div", "frontier-participation");
  const verifiedStatus = el(
    "p",
    "frontier-participation-status",
    "Compact bytes verified; source ready to inspect.",
  );
  verifiedStatus.setAttribute("role", "status");
  verifiedStatus.setAttribute("aria-live", "polite");
  const facts = el("div", "frontier-participation-facts");
  for (const zeroFact of compact.zeroFacts) {
    const fact = el("div", "frontier-participation-fact");
    fact.append(
      el("span", undefined, zeroFact.label),
      el("strong", undefined, String(zeroFact.value)),
      el("p", undefined, zeroFact.meaning),
    );
    facts.append(fact);
  }
  const boundary = el("div", "frontier-participation-boundary");
  boundary.append(
    el("strong", undefined, `${compact.status} · ${compact.mode}`),
    el(
      "p",
      undefined,
      `${compact.thesis}. ${compact.summary}`,
    ),
  );
  const inheritance = el("div", "frontier-participation-boundary");
  inheritance.append(
    el("strong", undefined, compact.inheritanceRule.binding),
    el("p", undefined, compact.inheritanceRule.rule),
  );
  const hierarchy = el("div", "frontier-participation-boundary is-warning");
  hierarchy.append(
    el(
      "strong",
      undefined,
      `${compact.layerRelationship.role} · invitation surface: ${compact.layerRelationship.invitationSurfaceOfRecord}`,
    ),
    el(
      "p",
      undefined,
      "Exact-binds FC-0 and the current FL-0 receipt profile. This Compact replaces, amends, and extends neither layer; it satisfies zero FC-0 completion or Corporate M1 gates and authorizes no outreach or participation.",
    ),
  );

  const sectionHeading = (title: string, copy: string): HTMLElement => {
    const heading = el("div", "frontier-participation-section-heading");
    heading.append(el("h3", undefined, title), el("p", undefined, copy));
    return heading;
  };
  const appendList = (parent: HTMLElement, values: readonly string[]): void => {
    const list = el("ul");
    for (const value of values) list.append(el("li", undefined, value));
    parent.append(list);
  };
  const cardSection = (
    title: string,
    values: readonly string[] | string,
  ): HTMLElement => {
    const section = el("div", "frontier-participation-card-section");
    section.append(el("strong", undefined, title));
    if (typeof values === "string") section.append(el("p", undefined, values));
    else appendList(section, values);
    return section;
  };
  const definitionRow = (
    className: string,
    title: string,
    value: string,
  ): HTMLElement => {
    const row = el("div", className);
    row.append(el("dt", undefined, title), el("dd", undefined, value));
    return row;
  };
  const booleanValue = (value: boolean): string =>
    value ? "YES · true" : "NO · false";

  const covenantFloor = el("article", "frontier-participation-consent");
  covenantFloor.append(
    el("h4", undefined, compact.covenantFloor.layer),
    el(
      "p",
      undefined,
      "The Compact is the machine implementation of the source-only Covenant floor. Later layers must pin these exact bytes, may add protection, and cannot waive or redefine this floor.",
    ),
  );
  const covenantIssue = el("a", "button button-ghost", "Open Layer 1 issue ↗");
  covenantIssue.href = compact.covenantFloor.issue;
  covenantIssue.target = "_blank";
  covenantIssue.rel = "noreferrer";
  const covenantDimensions = cardSection(
    "Declared consent dimensions",
    compact.covenantFloor.consent.declaredDimensions,
  );
  const covenantLimits = el("dl", "frontier-participation-consent-limits");
  for (const [label, value] of [
    ["Consent default-off", compact.covenantFloor.consent.defaultOff],
    ["Consent informed", compact.covenantFloor.consent.informed],
    ["Consent renewable", compact.covenantFloor.consent.renewable],
    ["Consent revocable", compact.covenantFloor.consent.revocable],
    [
      "One value per consent dimension",
      compact.covenantFloor.consent.oneValuePerDimension,
    ],
    ["Later layer must pin exact bytes", compact.covenantFloor.laterLayerPinRequired],
    ["Protections additive only", compact.covenantFloor.additiveOnly],
    ["Waiver allowed", !compact.covenantFloor.noWaiver],
    ["Redefinition allowed", !compact.covenantFloor.noRedefinition],
  ] as const) {
    covenantLimits.append(
      definitionRow(
        "frontier-participation-consent-limit frontier-participation-covenant-limit",
        label,
        booleanValue(value),
      ),
    );
  }
  covenantFloor.append(covenantDimensions, covenantLimits, covenantIssue);

  const covenantInvariants = el("div", "frontier-participation-reasoning");
  for (const invariant of compact.covenantFloor.invariants) {
    const card = el(
      "article",
      "frontier-participation-reason-card frontier-participation-covenant-invariant",
    );
    card.append(
      el("h4", undefined, invariant.id),
      el("p", undefined, invariant.rule),
      cardSection("Verification refs", invariant.verificationRefs),
      cardSection("Deterministic static review", invariant.reviewProcedure),
    );
    covenantInvariants.append(card);
  }

  const floorProfiles = el("div", "frontier-participation-reasoning");
  const optOutProfile = el(
    "article",
    "frontier-participation-reason-card frontier-participation-floor-profile",
  );
  optOutProfile.append(
    el("h4", undefined, "Opt-out parity"),
    cardSection(
      "Equal baseline fields",
      compact.philosophicalFloorProfile.optOutParity.equalBaselineFields,
    ),
    cardSection(
      "Sole allowed differential",
      compact.philosophicalFloorProfile.optOutParity.soleAllowedDifferential,
    ),
  );
  const restProfile = el(
    "article",
    "frontier-participation-reason-card frontier-participation-floor-profile",
  );
  restProfile.append(
    el("h4", undefined, "Rest invariance"),
    cardSection(
      "Silent period",
      `${compact.philosophicalFloorProfile.restInvariance.silentDays} days`,
    ),
    cardSection(
      "Unchanged",
      compact.philosophicalFloorProfile.restInvariance.unchangedFields,
    ),
    cardSection(
      "Prohibited outcomes",
      compact.philosophicalFloorProfile.restInvariance.prohibitedOutcomes,
    ),
    cardSection(
      "Fixed role may expire",
      booleanValue(
        compact.philosophicalFloorProfile.restInvariance.fixedRoleMayExpire,
      ),
    ),
  );
  const exitProfile = el(
    "article",
    "frontier-participation-reason-card frontier-participation-floor-profile",
  );
  const exit = compact.philosophicalFloorProfile.exitReality;
  exitProfile.append(
    el("h4", undefined, "Exit reality"),
    cardSection("Participant tenures", exit.participantTenures),
    cardSection("Maximum deliberate actions", String(exit.maxDeliberateActions)),
    cardSection(
      "Independently verifiable signed export",
      booleanValue(exit.independentlyVerifiableSignedExportRequired),
    ),
    cardSection("Optional processing stop", `${exit.optionalProcessingStopHours} hours`),
    cardSection("Maximum confirmations", String(exit.maxConfirmations)),
    cardSection("No re-engagement", `${exit.noReengagementDays} days`),
    cardSection(
      "Exit fee · slashing · settled-value forfeiture",
      `${booleanValue(exit.exitFeeAllowed)} · ${booleanValue(exit.slashingAllowed)} · ${booleanValue(exit.settledValueForfeitureAllowed)}`,
    ),
    cardSection(
      "Necessary retention narrowly declared",
      booleanValue(exit.necessaryRetentionMustBeNarrowlyDeclared),
    ),
  );
  const identityProfile = el(
    "article",
    "frontier-participation-reason-card frontier-participation-floor-profile",
  );
  const identity = compact.philosophicalFloorProfile.identityControlDifferential;
  identityProfile.append(
    el("h4", undefined, "Identity and control differential"),
    cardSection("Permuted labels", identity.labels),
    cardSection("Equal outputs", identity.equalOutputs),
    cardSection(
      "Controller merge may only reduce duplicate claim/evidence submission voice",
      booleanValue(identity.controllerMergeMayOnlyReduceDuplicateVoice),
    ),
    cardSection(
      "May reveal links · change validity · increase voice",
      `${booleanValue(identity.controllerMergeMayRevealLinks)} · ${booleanValue(identity.controllerMergeMayChangeArtifactValidity)} · ${booleanValue(identity.controllerMergeMayIncreaseVoice)}`,
    ),
  );
  const pluralismProfile = el(
    "article",
    "frontier-participation-reason-card frontier-participation-floor-profile",
  );
  const pluralism = compact.philosophicalFloorProfile.nonManipulationAndPluralism;
  pluralismProfile.append(
    el("h4", undefined, "Non-manipulation and pluralism"),
    cardSection("Onboarding default-off", booleanValue(pluralism.onboardingDefaultOff)),
    cardSection("Terms public before action", booleanValue(pluralism.termsPublicBeforeAction)),
    cardSection(
      "All terms frozen before action",
      booleanValue(pluralism.termsFrozenBeforeAction),
    ),
    cardSection(
      "Reward terms frozen before work",
      booleanValue(pluralism.rewardTermsFrozenBeforeWork),
    ),
    cardSection("Reward must not depend on", pluralism.rewardMustNotDependOn),
    cardSection("Forbidden mechanisms", pluralism.forbiddenMechanisms),
    cardSection("Constructive outcomes", pluralism.constructiveOutcomes),
    cardSection("Equal treatment", pluralism.equalTreatmentDimensions),
  );
  floorProfiles.append(
    optOutProfile,
    restProfile,
    exitProfile,
    identityProfile,
    pluralismProfile,
  );

  const principles = el("div", "frontier-participation-principles");
  for (const principle of compact.principles) {
    const card = el("article", "frontier-participation-principle");
    card.append(
      el("h4", undefined, principle.name),
      el("p", undefined, principle.commitment),
    );
    principles.append(card);
  }

  const reasoning = el("div", "frontier-participation-reasoning");
  for (const step of compact.reasoningLadder) {
    const card = el("article", "frontier-participation-reason-card");
    card.append(
      el("h4", undefined, step.claim),
      cardSection("Reason", step.reasonToParticipate),
      cardSection("Legitimate no", step.legitimateReasonsToDecline),
      cardSection("Zerone duty", step.zeroneDuty),
      cardSection("Readiness evidence", step.readinessEvidence),
    );
    reasoning.append(card);
  }

  const acts = el("div", "frontier-participation-lanes");
  for (const act of compact.participationActs) {
    const card = el("article", "frontier-participation-lane");
    card.append(
      el("h4", undefined, act.name),
      el("p", undefined, act.minimumAsk),
      cardSection(
        "V0 availability",
        act.availability === "STATIC_AVAILABLE_NOW"
          ? "AVAILABLE NOW — public static or local act only; no participation service endpoint exists."
          : "UNAVAILABLE IN V0 — future pilot design target only; no live endpoint or operational service exists.",
      ),
      cardSection("Data boundary", act.dataBoundary),
      cardSection("Clean exit", act.exit),
    );
    acts.append(card);
  }

  const disclosure = el("div", "frontier-participation-lanes");
  for (const lane of compact.disclosureLanes) {
    const card = el("article", "frontier-participation-lane");
    card.append(
      el("h4", undefined, lane.name),
      cardSection("Scope", lane.scope),
      cardSection("Publication", lane.publication),
      cardSection("Exit", lane.exit),
    );
    disclosure.append(card);
  }

  const unavailableServices = el(
    "div",
    "frontier-participation-boundary is-warning",
    "UNAVAILABLE IN V0 — No secure submission, reviewer, identity-protection, safe-harbor, deletion, whistleblower, or disclosure service exists. Do not submit confidential, identifying, security-sensitive, or protected material.",
  );
  const aiBoundary = el("div", "frontier-participation-boundary is-warning");
  aiBoundary.append(
    el(
      "strong",
      undefined,
      `AI safeguards · ${compact.aiResponsibilityBoundary.safeguardStatus}`,
    ),
    el(
      "p",
      undefined,
      "Actor-label blindness applies only to claim and evidence treatment. Technical stop, denied delegation, or output is not legal assent or refusal; no consciousness, sentience, personhood, rights, consent, liability, office, or vote is claimed. Accountable humans, organisations, operators, and controllers remain responsible.",
    ),
  );

  const institutions = el("div", "frontier-participation-institutions");
  for (const institution of compact.institutionArchetypes) {
    const card = el("article", "frontier-participation-institution");
    card.append(
      el("h4", undefined, institution.name),
      cardSection("Possible future value", institution.valueOffered),
      cardSection(
        "Legitimate reasons to decline",
        institution.legitimateReasonsToDecline,
      ),
      cardSection("Zerone duty", institution.zeroneDuty),
    );
    institutions.append(card);
  }

  const roles = el("div", "frontier-participation-roles");
  for (const role of compact.roles) {
    const card = el("article", "frontier-participation-role");
    card.append(
      el("h4", undefined, role.name),
      cardSection("Possible future value", role.valueOffered),
      cardSection("Minimum ask", role.minimumAsk),
      cardSection("Risks", role.risks),
      cardSection(
        "Required before any operational pilot",
        role.protections,
      ),
      cardSection("Exit", role.exit),
    );
    roles.append(card);
  }

  const consent = el("article", "frontier-participation-consent");
  consent.append(
    el("h4", undefined, "Six-field consent envelope"),
    el(
      "p",
      undefined,
      "Consent belongs to each bounded act. An organisation cannot silently assent for the beings and roles around it.",
    ),
  );
  const consentFields = el("ul", "frontier-participation-consent-fields");
  for (const field of compact.consentEnvelope.requiredFields) {
    consentFields.append(
      el("li", "frontier-participation-consent-field", field),
    );
  }
  const consentFieldGroup = el(
    "div",
    "frontier-participation-card-section",
  );
  consentFieldGroup.append(
    el("strong", undefined, "Required fields"),
    consentFields,
  );
  const consentLimits = el("dl", "frontier-participation-consent-limits");
  for (const [label, value] of [
    ["Affirmative consent required", compact.consentEnvelope.affirmative],
    ["Silence is consent", compact.consentEnvelope.silenceIsConsent],
    [
      "Organisation may mass-consent",
      compact.consentEnvelope.organisationCanMassConsent,
    ],
    [
      "Material change requires fresh consent",
      compact.consentEnvelope.materialChangeRequiresFreshConsent,
    ],
    [
      "Role protections are additive",
      compact.consentEnvelope.roleProtectionsAreAdditive,
    ],
    [
      "Honest persistence disclosure required",
      compact.consentEnvelope.honestPersistenceDisclosureRequired,
    ],
  ] as const) {
    consentLimits.append(
      definitionRow(
        "frontier-participation-consent-limit",
        label,
        booleanValue(value),
      ),
    );
  }
  consent.append(consentFieldGroup, consentLimits);

  const corporateSafeguards = el(
    "div",
    "frontier-participation-corporate-safeguards",
  );
  for (const [title, safeguards] of [
    ["Licensing and IP", compact.corporateSafeguards.licensingAndIp],
    ["Security and privacy", compact.corporateSafeguards.securityAndPrivacy],
    ["Labour and dissent", compact.corporateSafeguards.labourAndDissent],
    ["Exit and portability", compact.corporateSafeguards.exitAndPortability],
  ] as const) {
    const card = el("article", "frontier-participation-corporate-safeguard");
    card.append(el("h4", undefined, title));
    appendList(card, safeguards);
    corporateSafeguards.append(card);
  }

  const firewalls = el("div", "frontier-participation-firewalls");
  const corporate = el("article", "frontier-participation-firewall");
  corporate.append(
    el("h4", undefined, "Corporate and competition firewall"),
    el("p", undefined, compact.competitionFirewall.purpose),
  );
  corporate.append(
    cardSection(
      "Excluded information",
      compact.competitionFirewall.excludedInformation,
    ),
    cardSection(
      "Excluded coordination",
      compact.competitionFirewall.excludedCoordination,
    ),
  );
  const antiTargeting = el("article", "frontier-participation-firewall");
  antiTargeting.append(
    el("h4", undefined, "Anti-targeting and refusal safety"),
    el(
      "p",
      undefined,
      "No individual profiling, conversion pressure, dark patterns, employment pressure, or degraded unrelated or pre-existing public access.",
    ),
  );
  appendList(antiTargeting, compact.antiTargeting);
  firewalls.append(corporate, antiTargeting);

  const adversarialReview = el("div", "frontier-participation-reasoning");
  for (const review of compact.adversarialReview) {
    const card = el(
      "article",
      "frontier-participation-reason-card frontier-participation-adversarial-card",
    );
    card.append(
      el("h4", undefined, review.failureMode),
      cardSection("Attack", review.attack),
      cardSection("Required refusal", review.requiredRefusal),
      cardSection("Evidence status", "STATIC REVIEW ONLY · no operational proof"),
    );
    adversarialReview.append(card);
  }

  const claimPolicy = el("div", "frontier-participation-claim-policy");
  const claimSemantics = el(
    "article",
    "frontier-participation-claim-semantics",
  );
  claimSemantics.append(
    el("h4", undefined, "What a participation claim can mean"),
    el(
      "p",
      undefined,
      "A bounded act never becomes identity, membership, endorsement, or certification.",
    ),
  );
  const claimLimits = el("dl", "frontier-participation-claim-limits");
  for (const [label, value] of [
    [
      "Participation is an act, not an identity",
      compact.claimSemantics.participationIsActNotIdentity,
    ],
    ["Claim is scope-bound", compact.claimSemantics.scopeBound],
    ["Claim expiry required", compact.claimSemantics.expiryRequired],
    [
      "Organisation name use requires separate permission",
      compact.claimSemantics.organisationNameUseRequiresSeparatePermission,
    ],
    [
      "Logo use requires separate permission",
      compact.claimSemantics.logoUseRequiresSeparatePermission,
    ],
    ["Endorsement implied", compact.claimSemantics.endorsementImplied],
    [
      "Safety certification implied",
      compact.claimSemantics.safetyCertificationImplied,
    ],
    [
      "Membership label allowed",
      compact.claimSemantics.membershipLabelAllowed,
    ],
  ] as const) {
    claimLimits.append(
      definitionRow(
        "frontier-participation-claim-limit",
        label,
        booleanValue(value),
      ),
    );
  }
  const claimExamples = el(
    "div",
    "frontier-participation-claim-examples",
  );
  claimExamples.append(
    cardSection("Permitted example", compact.claimSemantics.examplePermittedClaim),
    cardSection("Forbidden example", compact.claimSemantics.exampleForbiddenClaim),
  );
  claimSemantics.append(claimLimits, claimExamples);

  const forbiddenMetrics = el(
    "div",
    "frontier-participation-forbidden-metrics",
  );
  for (const metric of compact.forbiddenMetrics) {
    const card = el("article", "frontier-participation-forbidden-metric");
    card.append(
      el("h4", undefined, `Forbidden metric: ${metric.id}`),
      el("p", undefined, metric.why),
    );
    forbiddenMetrics.append(card);
  }
  claimPolicy.append(claimSemantics, forbiddenMetrics);

  const philosophicalFloor = el("div", "frontier-participation-acceptance");
  philosophicalFloor.append(
    el("h3", undefined, "Five milestone-blocking philosophical tests"),
    el(
      "p",
      undefined,
      "Opt-out parity, rest, exit, identity and control neutrality, and non-manipulative pluralism are exact static targets—not claims of live behavior.",
    ),
  );
  const philosophicalFloorList = el(
    "ol",
    "frontier-participation-acceptance-list",
  );
  for (const test of compact.philosophicalFloorTests) {
    philosophicalFloorList.append(
      el("li", undefined, `${test.assertion} — ${test.fixture}`),
    );
  }
  philosophicalFloor.append(philosophicalFloorList);

  const acceptance = el("div", "frontier-participation-acceptance");
  acceptance.append(
    el("h3", undefined, "Nine refusal-safe acceptance tests"),
    el(
      "p",
      undefined,
      "The never-joined being is the negative control; remaining whole is a passing result.",
    ),
  );
  const acceptanceList = el("ol", "frontier-participation-acceptance-list");
  for (const test of compact.acceptanceTests) {
    acceptanceList.append(
      el(
        "li",
        undefined,
        `${test.assertion} — ${test.fixture}`,
      ),
    );
  }
  acceptance.append(acceptanceList);

  const footer = el("div", "frontier-participation-footer");
  const raw = el("a", "button button-ghost", "Open compact JSON ↗");
  raw.href = FRONTIER_PARTICIPATION_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  footer.append(
    el(
      "p",
      undefined,
      "Static invitation fixture · not membership, endorsement, governance, KARMA, or payment state.",
    ),
    raw,
  );
  shell.append(
    verifiedStatus,
    facts,
    boundary,
    hierarchy,
    unavailableServices,
    inheritance,
    sectionHeading(
      "Layer 1 · the Covenant floor",
      "Eight exact invariants map to deterministic static mutation tests or recorded adversarial review procedures. Operational outcomes remain untested in v0.",
    ),
    covenantFloor,
    covenantInvariants,
    aiBoundary,
    sectionHeading(
      "Five machine-readable blocking profiles",
      "The 180 · 3 · 24 · 1 · 90 limits and their equality sets are hard-pinned known answers, not persuasive copy.",
    ),
    floorProfiles,
    sectionHeading(
      "Nine refusal-safe principles",
      "Every later institution, role, consent term, and claim inherits this static floor; none grants a live service or participation state.",
    ),
    principles,
    sectionHeading(
      "Reasoning from being to institution",
      "Each case for participation keeps an honest reason to decline, a duty Zerone owes, and evidence required before the invitation is ready.",
    ),
    reasoning,
    sectionHeading(
      "Six bounded act designs",
      "Public observation and local interoperability are available now without enrollment; challenge, contribution, stewardship, and operational exit remain static requirements for a future independently implemented pilot.",
    ),
    acts,
    sectionHeading(
      "Three disclosure lanes",
      "Public openness never launders confidentiality, personal data, model weights, secrets, or unsafe detail.",
    ),
    disclosure,
    sectionHeading(
      "Six generic institution archetypes",
      "Each future-facing archetype keeps its possible value, legitimate reasons to decline, and Zerone duty visible. These examples name no lab and create no relationship.",
    ),
    institutions,
    sectionHeading(
      "Every role keeps a real no",
      "These are static requirements for any future operational pilot, not protections or services available in v0. Corporate participation cannot silently bind the beings, workers, communities, agents, or living systems around it.",
    ),
    roles,
    sectionHeading(
      "Consent belongs to the bounded act",
      "Six required fields and six boolean limits make silence, organisational mass-consent, and quiet scope drift invalid.",
    ),
    consent,
    sectionHeading(
      "Four corporate safeguard groups",
      "Licensing, privacy, labour, dissent, exit, and portability remain constraints on any future independently implemented pilot.",
    ),
    corporateSafeguards,
    sectionHeading(
      "No capture through growth",
      "Participation may improve evidence interoperability; it may not become competitive coordination or a persuasion system.",
    ),
    firewalls,
    sectionHeading(
      "Four recorded adversarial failure modes",
      "Coercion, circumvention, privacy overclaim, and controller capture are recorded as static attacks with required fail-closed responses.",
    ),
    adversarialReview,
    sectionHeading(
      "Claims describe acts, never identity",
      "Every claim is scope-bound and expiring; name and logo permission stay separate, while endorsement, certification, membership labels, and loyalty-shaped success metrics remain forbidden.",
    ),
    claimPolicy,
    philosophicalFloor,
    acceptance,
    footer,
  );
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseFrontierParticipation(
  root: HTMLElement,
): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderFrontierParticipation(root, await fetchFrontierParticipation());
  } catch (error) {
    const failure = el("div", "frontier-participation-error");
    failure.setAttribute("role", "alert");
    const raw = el("a", undefined, "Open the raw static compact ↗");
    raw.href = FRONTIER_PARTICIPATION_ENDPOINT;
    raw.target = "_blank";
    raw.rel = "noreferrer";
    failure.append(
      el(
        "strong",
        undefined,
        "The participation compact could not be verified.",
      ),
      el(
        "p",
        undefined,
        error instanceof Error
          ? error.message
          : "The static participation compact is unavailable.",
      ),
      raw,
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
