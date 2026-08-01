import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import {
  FRONTIER_PARTICIPATION_MAX_BYTES,
  FRONTIER_PARTICIPATION_SHA256,
  FrontierParticipationDataError,
  parseFrontierParticipationJson,
  type FrontierParticipationContract,
} from "../src/frontier-participation";

const MAX_JSON_NESTING = 64;

const EXPECTED_RELEASE_FLAGS = [
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

const EXPECTED_ZERO_FACTS = [
  ["account-required", "0"],
  ["wallet-or-token-required", "0"],
  ["private-ip-or-weights-required", "0"],
  ["lock-in-or-exit-penalty", "0"],
  ["refusal-penalty", "0"],
  ["authority-or-economic-effect", "OFF"],
] as const;

const EXPECTED_REASONING_IDS = [
  "agency-before-adoption",
  "truth-before-affiliation",
  "commons-without-capture",
  "reciprocity-without-dependency",
  "institutional-optionality",
  "team-level-utility",
  "being-level-dignity",
] as const;

const EXPECTED_ACT_AVAILABILITY = [
  "STATIC_AVAILABLE_NOW",
  "STATIC_AVAILABLE_NOW",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
  "FUTURE_PILOT_ONLY",
] as const;

const EXPECTED_ROLE_IDS = [
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

const EXPECTED_ARCHETYPE_IDS = [
  "frontier-model-developer",
  "independent-evaluator-security-lab",
  "academic-public-interest-lab",
  "open-source-model-community",
  "compute-platform-provider",
  "standards-regulatory-observer",
] as const;

const EXPECTED_ACCEPTANCE_TEST_IDS = [
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

const EXPECTED_FLOOR_TEST_IDS = [
  "opt-out-parity",
  "rest-invariance",
  "exit-reality",
  "identity-control-differential",
  "non-manipulation-and-pluralism",
] as const;

const EXPECTED_COVENANT_INVARIANT_IDS = [
  "refusal-is-complete",
  "nonparticipant-baselines-are-equal",
  "consent-is-scoped",
  "rest-and-exit-are-neutral",
  "plural-ends-are-legitimate",
  "identity-and-control-labels-are-neutral",
  "incentives-are-prospective-and-non-manipulative",
  "participation-is-nonexclusive-and-nonextractive",
] as const;

const EXPECTED_ADVERSARIAL_IDS = [
  "coercion",
  "circumvention",
  "privacy",
  "controller-capture",
] as const;

const EXPECTED_CONSENT_VALUES = {
  affirmative: true,
  silenceIsConsent: false,
  organisationCanMassConsent: false,
  materialChangeRequiresFreshConsent: true,
  roleProtectionsAreAdditive: true,
  honestPersistenceDisclosureRequired: true,
} as const;

const EXPECTED_CONSENT_FIELDS = [
  "scope",
  "contribution",
  "data",
  "duration",
  "rights",
  "exit",
] as const;

const EXPECTED_FIREWALL_VALUES = {
  operationalCharterRequiresIndependentCompetitionCounsel: true,
  neutralCommonsIsLegalConclusion: false,
} as const;

const EXPECTED_CLAIM_VALUES = {
  participationIsActNotIdentity: true,
  scopeBound: true,
  expiryRequired: true,
  organisationNameUseRequiresSeparatePermission: true,
  logoUseRequiresSeparatePermission: true,
  endorsementImplied: false,
  safetyCertificationImplied: false,
  membershipLabelAllowed: false,
} as const;

const EXPECTED_FORBIDDEN_METRIC_IDS = [
  "conversion-rate",
  "logo-count",
  "participation-score",
  "retention-or-lock-in",
  "favourable-finding-rate",
] as const;

const KNOWN_COMPANY_NAMES = [
  /\bopenai\b/i,
  /\banthropic\b/i,
  /\bgoogle deepmind\b/i,
  /\bmeta\b/i,
  /\bxai\b/i,
  /\bai2\b/i,
  /\bmila\b/i,
] as const;

const NORMATIVE_ENDORSEMENT =
  /\b(?:join(?:ed|s)?|participat(?:e|es|ed|ing)|endors(?:e|es|ed|ing)|support(?:s|ed|ing)?|partner(?:s|ed|ing)?|affiliat(?:e|es|ed|ion)|approv(?:e|es|ed|al)|certif(?:y|ies|ied|ication)|signator(?:y|ies))\b/i;

function fail(path: string, message: string): never {
  throw new FrontierParticipationDataError(`${path}: ${message}`);
}

function rejectExcessiveJsonNesting(raw: string): void {
  let depth = 0;
  let inString = false;
  let escaped = false;

  for (const character of raw) {
    if (inString) {
      if (escaped) {
        escaped = false;
      } else if (character === "\\") {
        escaped = true;
      } else if (character === '"') {
        inString = false;
      }
      continue;
    }

    if (character === '"') {
      inString = true;
    } else if (character === "{" || character === "[") {
      depth += 1;
      if (depth > MAX_JSON_NESTING) {
        fail("$", `JSON nesting exceeds ${MAX_JSON_NESTING}`);
      }
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
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
    return fail("$", "unterminated JSON string");
  };

  const scanValue = (path: string): void => {
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
        const keyPath = `${path}.${key}`;
        if (keys.has(key)) fail(keyPath, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object separator");
        offset += 1;
        scanValue(keyPath);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed object delimiter");
        offset += 1;
      }
      return fail(path, "unterminated object");
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
        if (raw[offset] !== ",") fail(path, "malformed array delimiter");
        offset += 1;
        index += 1;
      }
      return fail(path, "unterminated array");
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

function preflight(raw: string): void {
  if (Buffer.byteLength(raw, "utf8") > FRONTIER_PARTICIPATION_MAX_BYTES) {
    fail(
      "$",
      `document exceeds ${FRONTIER_PARTICIPATION_MAX_BYTES} UTF-8 bytes`,
    );
  }
  try {
    JSON.parse(raw);
  } catch (error) {
    fail(
      "$",
      `invalid JSON: ${error instanceof Error ? error.message : "unknown error"}`,
    );
  }
  rejectExcessiveJsonNesting(raw);
  rejectDuplicateJsonKeys(raw);
}

function sha256(raw: string): string {
  return createHash("sha256").update(raw, "utf8").digest("hex");
}

type UnknownRecord = Record<string, unknown>;

function record(value: unknown, path: string): UnknownRecord {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value as UnknownRecord;
}

function array(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value)) fail(path, "must be an array");
  return value;
}

function nonEmptyString(value: unknown, path: string): string {
  if (typeof value !== "string" || value.trim().length === 0) {
    fail(path, "must be a non-empty string");
  }
  return value;
}

function stringList(value: unknown, path: string): string[] {
  return array(value, path).map((entry, index) =>
    nonEmptyString(entry, `${path}[${index}]`),
  );
}

function exactKeySet(
  value: UnknownRecord,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (
    actual.length !== wanted.length ||
    actual.some((key, index) => key !== wanted[index])
  ) {
    fail(path, `must contain exactly ${wanted.join(", ")}`);
  }
}

function exactBooleanValues(
  value: UnknownRecord,
  expected: Readonly<Record<string, boolean>>,
  path: string,
): void {
  for (const [key, expectedValue] of Object.entries(expected)) {
    if (value[key] !== expectedValue) {
      fail(`${path}.${key}`, `must remain ${String(expectedValue)}`);
    }
  }
}

function uniqueIds(values: readonly UnknownRecord[], path: string): string[] {
  const ids = values.map((value, index) =>
    nonEmptyString(value.id, `${path}[${index}].id`),
  );
  if (new Set(ids).size !== ids.length) fail(path, "IDs must be unique");
  return ids;
}

function exactIdSet(
  actual: readonly string[],
  expected: readonly string[],
  path: string,
): void {
  const found = [...actual].sort();
  const wanted = [...expected].sort();
  if (
    found.length !== wanted.length ||
    found.some((id, index) => id !== wanted[index])
  ) {
    fail(path, `must cover exactly ${wanted.join(", ")}`);
  }
}

function objectList(value: unknown, path: string): UnknownRecord[] {
  return array(value, path).map((entry, index) =>
    record(entry, `${path}[${index}]`),
  );
}

function collectStrings(value: unknown, result: string[] = []): string[] {
  if (typeof value === "string") {
    result.push(value);
  } else if (Array.isArray(value)) {
    for (const entry of value) collectStrings(entry, result);
  } else if (value !== null && typeof value === "object") {
    for (const entry of Object.values(value as UnknownRecord)) {
      collectStrings(entry, result);
    }
  }
  return result;
}

function requireConcept(
  text: string,
  expression: RegExp,
  label: string,
  path: string,
): void {
  if (!expression.test(text)) fail(path, `must explicitly exclude ${label}`);
}

export interface FrontierParticipationValidationSummary {
  covenantInvariantCount: number;
  adversarialReviewCount: number;
  releaseFlagCount: number;
  zeroFactCount: number;
  reasoningStepCount: number;
  roleCount: number;
  consentFieldCount: number;
  forbiddenMetricCount: number;
  philosophicalFloorTestCount: number;
  acceptanceTestCount: number;
  digest: string;
}

function validateSemantics(
  contract: FrontierParticipationContract,
  digest: string,
): FrontierParticipationValidationSummary {
  const root = contract as unknown as UnknownRecord;
  if (root.version !== "0.0.0") fail("$.version", "must remain 0.0.0");
  if (root.status !== "STATIC_READY") {
    fail("$.status", "must remain STATIC_READY");
  }
  if (root.mode !== "INVITATION_ONLY") {
    fail("$.mode", "must remain INVITATION_ONLY");
  }

  const covenantFloor = record(root.covenantFloor, "$.covenantFloor");
  for (const flag of [
    "laterLayerPinRequired",
    "additiveOnly",
    "noWaiver",
    "noRedefinition",
  ]) {
    if (covenantFloor[flag] !== true) {
      fail(`$.covenantFloor.${flag}`, "must remain true");
    }
  }
  const covenantConsent = record(
    covenantFloor.consent,
    "$.covenantFloor.consent",
  );
  for (const flag of ["defaultOff", "informed", "renewable", "revocable"]) {
    if (covenantConsent[flag] !== true) {
      fail(`$.covenantFloor.consent.${flag}`, "must remain true");
    }
  }
  const covenantInvariants = objectList(
    covenantFloor.invariants,
    "$.covenantFloor.invariants",
  );
  exactIdSet(
    uniqueIds(covenantInvariants, "$.covenantFloor.invariants"),
    EXPECTED_COVENANT_INVARIANT_IDS,
    "$.covenantFloor.invariants",
  );
  covenantInvariants.forEach((invariant, index) => {
    const path = `$.covenantFloor.invariants[${index}]`;
    nonEmptyString(invariant.rule, `${path}.rule`);
    if (stringList(invariant.verificationRefs, `${path}.verificationRefs`).length === 0) {
      fail(`${path}.verificationRefs`, "must not be empty");
    }
    nonEmptyString(invariant.reviewProcedure, `${path}.reviewProcedure`);
    if (invariant.staticOnly !== true) fail(`${path}.staticOnly`, "must remain true");
  });

  const adversarialReview = objectList(
    root.adversarialReview,
    "$.adversarialReview",
  );
  exactIdSet(
    uniqueIds(adversarialReview, "$.adversarialReview"),
    EXPECTED_ADVERSARIAL_IDS,
    "$.adversarialReview",
  );
  adversarialReview.forEach((review, index) => {
    const path = `$.adversarialReview[${index}]`;
    for (const field of ["failureMode", "attack", "requiredRefusal"]) {
      nonEmptyString(review[field], `${path}.${field}`);
    }
    if (review.staticOnly !== true) fail(`${path}.staticOnly`, "must remain true");
  });

  const floorProfile = record(
    root.philosophicalFloorProfile,
    "$.philosophicalFloorProfile",
  );
  const restProfile = record(
    floorProfile.restInvariance,
    "$.philosophicalFloorProfile.restInvariance",
  );
  if (restProfile.silentDays !== 180) {
    fail("$.philosophicalFloorProfile.restInvariance.silentDays", "must remain 180");
  }
  const exitProfile = record(
    floorProfile.exitReality,
    "$.philosophicalFloorProfile.exitReality",
  );
  for (const [field, expected] of [
    ["maxDeliberateActions", 3],
    ["optionalProcessingStopHours", 24],
    ["maxConfirmations", 1],
    ["noReengagementDays", 90],
  ] as const) {
    if (exitProfile[field] !== expected) {
      fail(`$.philosophicalFloorProfile.exitReality.${field}`, `must remain ${expected}`);
    }
  }
  const pluralismProfile = record(
    floorProfile.nonManipulationAndPluralism,
    "$.philosophicalFloorProfile.nonManipulationAndPluralism",
  );
  for (const flag of [
    "onboardingDefaultOff",
    "termsPublicBeforeAction",
    "rewardTermsFrozenBeforeWork",
  ]) {
    if (pluralismProfile[flag] !== true) {
      fail(
        `$.philosophicalFloorProfile.nonManipulationAndPluralism.${flag}`,
        "must remain true",
      );
    }
  }

  const releaseBoundary = record(root.releaseBoundary, "$.releaseBoundary");
  exactKeySet(releaseBoundary, EXPECTED_RELEASE_FLAGS, "$.releaseBoundary");
  for (const flag of EXPECTED_RELEASE_FLAGS) {
    if (releaseBoundary[flag] !== false) {
      fail(`$.releaseBoundary.${flag}`, "every release flag must remain false");
    }
  }

  const zeroFacts = objectList(root.zeroFacts, "$.zeroFacts");
  if (zeroFacts.length !== EXPECTED_ZERO_FACTS.length) {
    fail("$.zeroFacts", "must contain exactly six zero facts");
  }
  const zeroFactIds = uniqueIds(zeroFacts, "$.zeroFacts");
  exactIdSet(
    zeroFactIds,
    EXPECTED_ZERO_FACTS.map(([id]) => id),
    "$.zeroFacts",
  );
  const expectedZeroValues = new Map<string, string>(EXPECTED_ZERO_FACTS);
  zeroFacts.forEach((fact, index) => {
    const id = zeroFactIds[index];
    if (id === undefined || fact.value !== expectedZeroValues.get(id)) {
      fail(`$.zeroFacts[${index}].value`, "reviewed zero fact changed");
    }
  });

  const principles = objectList(root.principles, "$.principles");
  if (principles.length !== 9) {
    fail("$.principles", "must contain exactly nine principles");
  }
  uniqueIds(principles, "$.principles");

  const reasoningSteps = objectList(root.reasoningLadder, "$.reasoningLadder");
  if (reasoningSteps.length !== EXPECTED_REASONING_IDS.length) {
    fail("$.reasoningLadder", "must contain exactly seven reasoning steps");
  }
  const reasoningIds = uniqueIds(reasoningSteps, "$.reasoningLadder");
  exactIdSet(reasoningIds, EXPECTED_REASONING_IDS, "$.reasoningLadder");
  reasoningSteps.forEach((step, index) => {
    const path = `$.reasoningLadder[${index}]`;
    nonEmptyString(step.reasonToParticipate, `${path}.reasonToParticipate`);
    const declines = stringList(
      step.legitimateReasonsToDecline,
      `${path}.legitimateReasonsToDecline`,
    );
    if (declines.length === 0) {
      fail(
        `${path}.legitimateReasonsToDecline`,
        "must preserve at least one legitimate reason to decline",
      );
    }
    nonEmptyString(step.zeroneDuty, `${path}.zeroneDuty`);
    nonEmptyString(step.readinessEvidence, `${path}.readinessEvidence`);
  });

  const participationActs = objectList(
    root.participationActs,
    "$.participationActs",
  );
  if (participationActs.length !== 6) {
    fail("$.participationActs", "must contain exactly six bounded acts");
  }
  uniqueIds(participationActs, "$.participationActs");
  participationActs.forEach((act, index) => {
    if (act.availability !== EXPECTED_ACT_AVAILABILITY[index]) {
      fail(
        `$.participationActs[${index}].availability`,
        "reviewed static availability changed",
      );
    }
    for (const flag of [
      "endorsementImplied",
      "membershipCreated",
      "liveEndpoint",
    ]) {
      if (act[flag] !== false) {
        fail(`$.participationActs[${index}].${flag}`, "must remain false");
      }
    }
  });

  const disclosureLanes = objectList(
    root.disclosureLanes,
    "$.disclosureLanes",
  );
  if (disclosureLanes.length !== 3) {
    fail("$.disclosureLanes", "must contain exactly three disclosure lanes");
  }
  exactIdSet(
    uniqueIds(disclosureLanes, "$.disclosureLanes"),
    ["public", "time-embargoed", "confidential-review-only"],
    "$.disclosureLanes",
  );

  const archetypes = objectList(
    root.institutionArchetypes,
    "$.institutionArchetypes",
  );
  if (archetypes.length !== 6) {
    fail(
      "$.institutionArchetypes",
      "must contain exactly six generic institution archetypes",
    );
  }
  exactIdSet(
    uniqueIds(archetypes, "$.institutionArchetypes"),
    EXPECTED_ARCHETYPE_IDS,
    "$.institutionArchetypes",
  );

  const roles = objectList(root.roles, "$.roles");
  if (roles.length !== EXPECTED_ROLE_IDS.length) {
    fail("$.roles", "must contain exactly fifteen role protections");
  }
  const roleIds = uniqueIds(roles, "$.roles");
  exactIdSet(roleIds, EXPECTED_ROLE_IDS, "$.roles");
  roles.forEach((role, index) => {
    const path = `$.roles[${index}]`;
    for (const field of [
      "valueOffered",
      "minimumAsk",
      "risks",
      "protections",
    ]) {
      if (stringList(role[field], `${path}.${field}`).length === 0) {
        fail(`${path}.${field}`, "must not be empty");
      }
    }
    nonEmptyString(role.exit, `${path}.exit`);
  });

  const consent = record(root.consentEnvelope, "$.consentEnvelope");
  exactBooleanValues(consent, EXPECTED_CONSENT_VALUES, "$.consentEnvelope");
  const consentFields = stringList(
    consent.requiredFields,
    "$.consentEnvelope.requiredFields",
  );
  exactIdSet(
    consentFields,
    EXPECTED_CONSENT_FIELDS,
    "$.consentEnvelope.requiredFields",
  );

  const firewall = record(root.competitionFirewall, "$.competitionFirewall");
  exactBooleanValues(
    firewall,
    EXPECTED_FIREWALL_VALUES,
    "$.competitionFirewall",
  );
  const excludedInformation = stringList(
    firewall.excludedInformation,
    "$.competitionFirewall.excludedInformation",
  );
  const excludedCoordination = stringList(
    firewall.excludedCoordination,
    "$.competitionFirewall.excludedCoordination",
  );
  const firewallText = [...excludedInformation, ...excludedCoordination]
    .join("\n")
    .toLowerCase();
  requireConcept(
    firewallText,
    /\bpric(?:e|es|ing)\b/,
    "price coordination",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bwages?\b|\bcompensation\b/,
    "wage coordination",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bhiring\b|\bno-hire\b/,
    "hiring coordination",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bcompute\b/,
    "compute coordination",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\brelease\b/,
    "release coordination",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bcustomer\b/,
    "customer allocation",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bmarket\b/,
    "market allocation",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\ballocation\b/,
    "customer or market allocation",
    "$.competitionFirewall",
  );
  requireConcept(
    firewallText,
    /\bboycotts?\b/,
    "boycotts",
    "$.competitionFirewall",
  );

  const antiTargeting = stringList(root.antiTargeting, "$.antiTargeting");
  const antiTargetingText = antiTargeting.join("\n").toLowerCase();
  for (const [label, expression] of [
    ["individual targeting", /\btarget(?:ing|ed)?\b|\bmicrotargeting\b/],
    ["profiling", /\bprofil(?:e|es|ed|ing)\b/],
    ["fear-of-missing-out pressure", /\bfear of missing out\b|\bscarcity\b/],
    ["refusal penalties", /\bdeclining\b|\brefusal\b/],
    ["tracking refusers", /\btracked as a refuser\b|\bcovert tracking\b/],
  ] as const) {
    requireConcept(antiTargetingText, expression, label, "$.antiTargeting");
  }

  const claimSemantics = record(root.claimSemantics, "$.claimSemantics");
  exactBooleanValues(
    claimSemantics,
    EXPECTED_CLAIM_VALUES,
    "$.claimSemantics",
  );

  for (const statement of collectStrings(root)) {
    if (
      KNOWN_COMPANY_NAMES.some((pattern) => pattern.test(statement)) &&
      NORMATIVE_ENDORSEMENT.test(statement)
    ) {
      fail(
        "$",
        "named-company participation or endorsement language is forbidden",
      );
    }
  }

  const forbiddenMetrics = objectList(
    root.forbiddenMetrics,
    "$.forbiddenMetrics",
  );
  if (forbiddenMetrics.length !== EXPECTED_FORBIDDEN_METRIC_IDS.length) {
    fail("$.forbiddenMetrics", "must contain exactly five forbidden metrics");
  }
  const forbiddenMetricIds = uniqueIds(
    forbiddenMetrics,
    "$.forbiddenMetrics",
  );
  exactIdSet(
    forbiddenMetricIds,
    EXPECTED_FORBIDDEN_METRIC_IDS,
    "$.forbiddenMetrics",
  );
  forbiddenMetrics.forEach((metric, index) => {
    nonEmptyString(metric.why, `$.forbiddenMetrics[${index}].why`);
  });

  const philosophicalFloorTests = objectList(
    root.philosophicalFloorTests,
    "$.philosophicalFloorTests",
  );
  exactIdSet(
    uniqueIds(philosophicalFloorTests, "$.philosophicalFloorTests"),
    EXPECTED_FLOOR_TEST_IDS,
    "$.philosophicalFloorTests",
  );
  philosophicalFloorTests.forEach((test, index) => {
    const path = `$.philosophicalFloorTests[${index}]`;
    nonEmptyString(test.assertion, `${path}.assertion`);
    nonEmptyString(test.fixture, `${path}.fixture`);
    if (test.staticOnly !== true) fail(`${path}.staticOnly`, "must remain true");
  });

  const acceptanceTests = objectList(
    root.acceptanceTests,
    "$.acceptanceTests",
  );
  if (acceptanceTests.length !== EXPECTED_ACCEPTANCE_TEST_IDS.length) {
    fail("$.acceptanceTests", "must contain exactly nine acceptance tests");
  }
  const acceptanceTestIds = uniqueIds(
    acceptanceTests,
    "$.acceptanceTests",
  );
  exactIdSet(
    acceptanceTestIds,
    EXPECTED_ACCEPTANCE_TEST_IDS,
    "$.acceptanceTests",
  );
  acceptanceTests.forEach((test, index) => {
    const path = `$.acceptanceTests[${index}]`;
    nonEmptyString(test.assertion, `${path}.assertion`);
    nonEmptyString(test.fixture, `${path}.fixture`);
    if (test.staticOnly !== true) {
      fail(`${path}.staticOnly`, "must remain true");
    }
  });

  return {
    covenantInvariantCount: covenantInvariants.length,
    adversarialReviewCount: adversarialReview.length,
    releaseFlagCount: EXPECTED_RELEASE_FLAGS.length,
    zeroFactCount: zeroFacts.length,
    reasoningStepCount: reasoningSteps.length,
    roleCount: roles.length,
    consentFieldCount: consentFields.length,
    forbiddenMetricCount: forbiddenMetrics.length,
    philosophicalFloorTestCount: philosophicalFloorTests.length,
    acceptanceTestCount: acceptanceTests.length,
    digest,
  };
}

export function validateFrontierParticipationRaw(
  raw: string,
): FrontierParticipationValidationSummary {
  preflight(raw);
  const actualDigest = sha256(raw);
  if (actualDigest !== FRONTIER_PARTICIPATION_SHA256) {
    fail("$", "document digest differs from the reviewed runtime pin");
  }
  return validateSemantics(parseFrontierParticipationJson(raw), actualDigest);
}

function runCli(): void {
  if (process.argv.length !== 3) {
    console.error(
      "usage: tsx scripts/validate-frontier-participation.ts COMPACT_JSON",
    );
    process.exitCode = 2;
    return;
  }

  const artifactPath = process.argv[2];
  if (artifactPath === undefined) {
    process.exitCode = 2;
    return;
  }

  try {
    const summary = validateFrontierParticipationRaw(
      readFileSync(resolve(artifactPath), "utf8"),
    );
    console.log(
      `frontier participation: PASS (${summary.releaseFlagCount} release flags closed; ${summary.covenantInvariantCount} Covenant invariants, ${summary.adversarialReviewCount} adversarial reviews, ${summary.zeroFactCount} zero facts, ${summary.reasoningStepCount} reasoning steps, ${summary.roleCount} roles, ${summary.consentFieldCount} consent fields, ${summary.forbiddenMetricCount} forbidden metrics, ${summary.philosophicalFloorTestCount} floor tests, ${summary.acceptanceTestCount} compact tests; sha256 ${summary.digest})`,
    );
  } catch (error) {
    console.error(
      `frontier participation: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
    );
    process.exitCode = 1;
  }
}

if (
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
