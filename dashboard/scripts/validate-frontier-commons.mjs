import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const FRONTIER_COMMONS_SCHEMA =
  "zerone.frontier-commons-participation/v0";
export const FRONTIER_COMMONS_MAX_BYTES = 65_536;

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "authoritative",
  "networkObserved",
  "membershipBearing",
  "economicBearing",
  "governanceBearing",
  "snapshotDate",
  "purpose",
  "participationFacts",
  "costBoundary",
  "milestone",
  "releaseBoundary",
  "constitutionalBindings",
  "rights",
  "participationModes",
  "reasoningLadder",
  "constituencies",
  "objectionRegister",
  "completionGates",
  "nextMilestoneGates",
  "corporateReadiness",
];
const PARTICIPATION_FACT_KEYS = [
  "scope",
  "publicStaticSourceAvailability",
  "actualParticipants",
  "signatories",
  "createsAffiliation",
  "authorizesLogoUse",
  "authorizesTargetedOutreach",
  "authorizesDirectOrCorporateOutreach",
  "operatesLiveParticipationService",
  "writesNetworkState",
];
const PARTICIPATION_FACT_FALSE_KEYS = [
  "createsAffiliation",
  "authorizesLogoUse",
  "authorizesTargetedOutreach",
  "authorizesDirectOrCorporateOutreach",
  "operatesLiveParticipationService",
  "writesNetworkState",
];
const COST_BOUNDARY_KEYS = [
  "protocolConsideration",
  "claimsCostlessParticipation",
  "disclosedNonMoneyCosts",
];
const MILESTONE_KEYS = [
  "id",
  "title",
  "subtitle",
  "state",
  "completionClaimed",
  "adoptionClaimed",
  "promise",
  "success",
  "exclusions",
];
const RELEASE_BOUNDARY_KEYS = [
  "createsMembership",
  "registersParticipants",
  "acceptsConfidentialSubmissions",
  "executesAgreements",
  "collectsResearchData",
  "activatesAdapters",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsAuthority",
  "changesGovernanceWeight",
  "modifiesKarma",
  "authorizesResearch",
  "authorizesBiologicalExperimentation",
  "assertsEndorsement",
  "changesConsensus",
];
const CONSTITUTIONAL_BINDING_KEYS = [
  "artifacts",
  "moneyGrantsVoice",
  "recognitionGrantsAuthority",
  "karmaIsScalar",
  "nonParticipationCanBePenalized",
  "founderOrSponsorPrivilege",
  "currentOperatorConcentrationDisclosed",
];
const ARTIFACT_KEYS = ["id", "schema", "path", "sha256"];
const RIGHTS_KEYS = [
  "voluntary",
  "rightToDecline",
  "rightToPause",
  "rightToExit",
  "decliningRequiresJustification",
  "nonParticipationPenalty",
  "exitPenalty",
  "exclusivityRequired",
  "inspectionRequiresAccount",
  "inspectionRequiresWallet",
  "inspectionRequiresPayment",
  "inspectionRequiresIdentity",
  "inspectionRequiresDataUpload",
  "inspectionRequiresIpAssignment",
  "inspectionRequiresEndorsement",
  "participationImpliesEndorsement",
  "logoUseWithoutPermission",
  "rightsDependOnRole",
  "repositoryLicenseExtendsToSubmittedDataOrIp",
];
const PARTICIPATION_MODE_KEYS = [
  "id",
  "order",
  "state",
  "commitment",
  "receives",
  "doesNotReceive",
  "exit",
];
const REASONING_KEYS = [
  "id",
  "order",
  "principle",
  "offer",
  "reasonToInspect",
  "refusalBoundary",
  "currentEvidence",
  "unresolved",
];
const CONSTITUENCY_KEYS = [
  "id",
  "layer",
  "lens",
  "reason",
  "protectedFrom",
  "stillNeededForActiveParticipation",
];
const OBJECTION_KEYS = [
  "id",
  "state",
  "fc0Resolves",
  "fc0DoesNotResolve",
];
const GATE_KEYS = ["id", "passed", "requirement"];
const CORPORATE_READINESS_KEYS = [
  "milestone",
  "status",
  "authorizesExternalCorporateInvitation",
  "authorizesInstitutionalParticipationLane",
  "protectionsOperationallyEnforced",
  "requiredGates",
];
const CORPORATE_READINESS_FALSE_KEYS = [
  "authorizesExternalCorporateInvitation",
  "authorizesInstitutionalParticipationLane",
  "protectionsOperationallyEnforced",
];

const EXPECTED_BINDINGS = Object.freeze([
  Object.freeze({
    id: "epigenetics-capability-garden-v1",
    schema: "zerone.epigenetics-capability-garden/v1",
    path: "dashboard/public/standards/epigenetics-capability-garden.v1.json",
    sha256: "7d04efe9da46309bf97b850c9b80324b1a5c4035edb1008b9ba3ad0df2bcfa63",
  }),
  Object.freeze({
    id: "karma-foundation-v1",
    schema: "zerone.karma-foundation/v1",
    path: "dashboard/public/standards/karma-foundation.v1.json",
    sha256: "b46710704869dcc340ded356be72b4ec692f204710fedfb5cd43eb3757dc7b80",
  }),
  Object.freeze({
    id: "life-sciences-shadow-v0",
    schema: "zerone.constructive-intelligence-life-sciences/v0",
    path: "dashboard/public/standards/constructive-intelligence-life-sciences.v0.json",
    sha256: "2b3a0b0b92797ef459c6ec02a38d1b9ebde62105a1b558088e44779d4595508d",
  }),
  Object.freeze({
    id: "money-karma-v1",
    schema: "zerone.money-karma.constitution/v1",
    path: "docs/constitution/money-karma-v1.json",
    sha256: "24d5a2bdef9f3ce8cee41ed9416be65681b2cfd123ba96df7f6a7710c810ee03",
  }),
]);
const EXPECTED_MILESTONE_EXCLUSIONS = [
  "lab-logo-count",
  "membership-count",
  "participation-rate",
  "person-rank",
  "token-or-wallet-count",
];
const EXPECTED_MODES = Object.freeze([
  Object.freeze({ id: "observe", state: "AVAILABLE_NOW" }),
  Object.freeze({ id: "verify-and-fork", state: "AVAILABLE_NOW" }),
  Object.freeze({
    id: "public-source-contribution",
    state: "REQUIRES_SEPARATE_DUE_DILIGENCE",
  }),
  Object.freeze({
    id: "reproduce-or-challenge",
    state: "STANDARD_ONLY_NO_INTAKE",
  }),
  Object.freeze({ id: "sponsor", state: "INACTIVE" }),
  Object.freeze({ id: "govern", state: "INACTIVE" }),
]);
const EXPECTED_REASONING_IDS = [
  "R0-being-and-agency",
  "R1-epistemic-pluralism",
  "R2-artifacts-over-status",
  "R3-commons-and-provenance",
  "R4-interoperability-and-optionality",
  "R5-institutional-science",
  "R6-corporate-risk-and-value",
  "R7-personal-agency-and-craft",
];
const EXPECTED_CONSTITUENCY_IDS = [
  "affected-beings-and-publics",
  "boards-and-fiduciaries",
  "executives-and-strategy",
  "research-leadership",
  "research-scientists",
  "software-and-research-engineers",
  "safety-and-security",
  "ethics-biosafety-and-human-subjects",
  "data-and-privacy-stewards",
  "legal-ip-and-compliance",
  "finance-procurement-and-funders",
  "program-operations-and-communications",
  "early-career-contractors-and-technicians",
  "independent-and-smaller-labs",
  "ai-systems-agents-and-operators",
  "unlisted-affected-being-or-role",
];
const EXPECTED_OBJECTIONS = Object.freeze([
  Object.freeze({ id: "lock-in-and-exclusivity", state: "RESOLVED_FOR_READ_ONLY" }),
  Object.freeze({ id: "wallet-token-and-money", state: "RESOLVED_FOR_READ_ONLY" }),
  Object.freeze({ id: "endorsement-and-reputation", state: "RESOLVED_FOR_READ_ONLY" }),
  Object.freeze({ id: "ip-confidentiality-and-licensing", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "security-and-vulnerability-reporting", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "privacy-hosting-and-logs", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "data-genomics-and-research-records", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "research-clinical-and-biological-authorization", state: "OUT_OF_SCOPE" }),
  Object.freeze({ id: "antitrust-export-sanctions-and-law", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "procurement-sponsor-insurance-and-sla", state: "REQUIRES_SEPARATE_REVIEW" }),
  Object.freeze({ id: "governance-capture-and-control", state: "REQUIRES_SEPARATE_REVIEW" }),
]);
const EXPECTED_GATE_IDS = [
  "G0-contribution-policy-and-authority",
  "G1-conduct-and-enforcement",
  "G2-private-security-disclosure",
  "G3-hosted-privacy-and-retention",
  "G4-institutional-ip-and-data",
  "G5-research-intake-and-adjudication",
  "G6-sponsor-contract-and-escrow",
  "G7-independent-governance-and-capture",
  "G8-support-compatibility-and-exit",
];
const EXPECTED_COMPLETION_GATE_IDS = [
  "C0-merged-and-production-bytes",
  "C1-read-only-surface",
  "C2-non-operator-review",
  "C3-fork-and-exit-rehearsal",
];
const EXPECTED_NON_MONEY_COSTS = [
  "time",
  "compute",
  "legal-review",
  "security-review",
  "opportunity-cost",
];
const EXPECTED_CORPORATE_GATE_IDS = [
  "accessibility-labor-worker-classification-and-whistleblower-review",
  "authenticated-relation-graph-and-correction-authority",
  "code-of-conduct-enforcement-appeal-and-anti-retaliation",
  "competition-and-confidentiality-review",
  "contribution-ip-patent-publication-and-license-terms",
  "counterparty-scope-and-signatory-authority",
  "explicit-accountable-human-outreach-decision",
  "governing-terms-jurisdiction-and-dispute-process",
  "human-data-owner-publication-classification",
  "independent-governance-capture-custody-and-remedy-review",
  "independent-receipt-parser-threat-model-and-material-binding-review",
  "liability-indemnity-insurance-warranty-and-remedy",
  "logo-name-affiliation-and-endorsement-policy",
  "m0-1-declared-control-separated-nonauthor-roundtrip-complete",
  "maintainer-change-control-versioning-and-deprecation",
  "outreach-non-targeting-contact-source-one-contact-no-response-stop-and-retention-policy",
  "privacy-data-map-dpa-retention-erasure-and-public-permanence",
  "procurement-tax-accounting-sanctions-export-and-financial-promotion",
  "security-coordinated-disclosure-safe-harbor-incident-and-embargo",
  "service-level-support-availability-portability-and-exit",
];
const FRONTIER_COMMONS_CANONICAL_SEMANTIC_SHA256 =
  "1dd9c5abfe0d98400e3ee0db629582ca0f1d83e6afd0f98b7679df18bccaaf3e";
const TRUE_RIGHTS = new Set([
  "voluntary",
  "rightToDecline",
  "rightToPause",
  "rightToExit",
]);
const CONSTITUENCY_LAYERS = new Set([
  "AFFECTED",
  "CORPORATE",
  "TEAM",
  "PERSON",
  "ECOSYSTEM",
  "AI",
]);
const ID_PATTERN = /^(?:[A-Za-z][A-Za-z0-9]*|[A-Za-z0-9]+(?:-[A-Za-z0-9]+)+)$/;
const SHA256_PATTERN = /^[a-f0-9]{64}$/;
const REPOSITORY_PATH_PATTERN =
  /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;
const NAMED_LAB_PATTERN =
  /\b(?:openai|anthropic|deepmind|google|meta|microsoft|amazon|aws|mistral|cohere|xai|spacexai)\b/i;
const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "../..");

export class FrontierCommonsValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "FrontierCommonsValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new FrontierCommonsValidationError(path, message);
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
    if (!allowedSet.has(key)) fail(`${path}.${key}`, "is not part of FC-0");
  }
  for (const key of allowed) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is required");
  }
}

function text(value, path, maxBytes = 2_048) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function exact(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${JSON.stringify(expected)}`);
  return value;
}

function falseOnly(value, path) {
  return exact(value, false, path);
}

function trueOnly(value, path) {
  return exact(value, true, path);
}

function array(value, path, length) {
  if (!Array.isArray(value) || value.length !== length) {
    fail(path, `must contain exactly ${length} entries`);
  }
  return value;
}

function exactStringArray(value, expected, path) {
  array(value, path, expected.length);
  value.forEach((candidate, index) => {
    exact(text(candidate, `${path}[${index}]`, 128), expected[index], `${path}[${index}]`);
  });
}

function validateParticipationFacts(value) {
  const path = "$.participationFacts";
  const facts = record(value, path);
  exactKeys(facts, PARTICIPATION_FACT_KEYS, path);
  exact(
    text(facts.scope, `${path}.scope`, 64),
    "PUBLIC_STATIC_SOURCE_ONLY",
    `${path}.scope`,
  );
  trueOnly(
    facts.publicStaticSourceAvailability,
    `${path}.publicStaticSourceAvailability`,
  );
  array(facts.actualParticipants, `${path}.actualParticipants`, 0);
  array(facts.signatories, `${path}.signatories`, 0);
  for (const key of PARTICIPATION_FACT_FALSE_KEYS) {
    falseOnly(facts[key], `${path}.${key}`);
  }
}

function validateCostBoundary(value) {
  const path = "$.costBoundary";
  const boundary = record(value, path);
  exactKeys(boundary, COST_BOUNDARY_KEYS, path);
  exact(
    text(boundary.protocolConsideration, `${path}.protocolConsideration`, 32),
    "NONE",
    `${path}.protocolConsideration`,
  );
  falseOnly(
    boundary.claimsCostlessParticipation,
    `${path}.claimsCostlessParticipation`,
  );
  exactStringArray(
    boundary.disclosedNonMoneyCosts,
    EXPECTED_NON_MONEY_COSTS,
    `${path}.disclosedNonMoneyCosts`,
  );
}

function validateCorporateReadiness(value) {
  const path = "$.corporateReadiness";
  const readiness = record(value, path);
  exactKeys(readiness, CORPORATE_READINESS_KEYS, path);
  exact(text(readiness.milestone, `${path}.milestone`, 16), "M1", `${path}.milestone`);
  exact(
    text(readiness.status, `${path}.status`, 32),
    "NOT_READY",
    `${path}.status`,
  );
  for (const key of CORPORATE_READINESS_FALSE_KEYS) {
    falseOnly(readiness[key], `${path}.${key}`);
  }
  exactStringArray(
    readiness.requiredGates,
    EXPECTED_CORPORATE_GATE_IDS,
    `${path}.requiredGates`,
  );
}

function validateBinding(value, index) {
  const path = `$.constitutionalBindings.artifacts[${index}]`;
  const binding = record(value, path);
  exactKeys(binding, ARTIFACT_KEYS, path);
  const expected = EXPECTED_BINDINGS[index];
  exact(text(binding.id, `${path}.id`, 96), expected.id, `${path}.id`);
  exact(text(binding.schema, `${path}.schema`, 128), expected.schema, `${path}.schema`);
  const repositoryPath = text(binding.path, `${path}.path`, 256);
  if (!REPOSITORY_PATH_PATTERN.test(repositoryPath)) {
    fail(`${path}.path`, "must be a safe repository-relative path");
  }
  exact(repositoryPath, expected.path, `${path}.path`);
  const digest = text(binding.sha256, `${path}.sha256`, 64);
  if (!SHA256_PATTERN.test(digest)) fail(`${path}.sha256`, "must be lowercase SHA-256");
  exact(digest, expected.sha256, `${path}.sha256`);

  const resolved = resolve(REPOSITORY_ROOT, repositoryPath);
  if (!existsSync(resolved)) fail(`${path}.path`, "must resolve inside the repository");
  const actual = createHash("sha256").update(readFileSync(resolved)).digest("hex");
  if (actual !== digest) fail(`${path}.sha256`, `binding drifted; actual digest is ${actual}`);
}

function validateParticipationMode(value, index) {
  const path = `$.participationModes[${index}]`;
  const mode = record(value, path);
  exactKeys(mode, PARTICIPATION_MODE_KEYS, path);
  const expected = EXPECTED_MODES[index];
  exact(text(mode.id, `${path}.id`, 96), expected.id, `${path}.id`);
  exact(mode.order, index, `${path}.order`);
  exact(text(mode.state, `${path}.state`, 64), expected.state, `${path}.state`);
  text(mode.commitment, `${path}.commitment`);
  text(mode.receives, `${path}.receives`);
  text(mode.doesNotReceive, `${path}.doesNotReceive`);
  text(mode.exit, `${path}.exit`);
  if (
    index >= 2 &&
    !/separate|none|inspect|standard|only after|no intake|inactive/i.test(
      mode.commitment,
    )
  ) {
    fail(`${path}.commitment`, "must not imply an active bundled commitment");
  }
}

function validateReasoning(value, index) {
  const path = `$.reasoningLadder[${index}]`;
  const reason = record(value, path);
  exactKeys(reason, REASONING_KEYS, path);
  exact(text(reason.id, `${path}.id`, 96), EXPECTED_REASONING_IDS[index], `${path}.id`);
  exact(reason.order, index, `${path}.order`);
  for (const key of REASONING_KEYS.slice(2)) text(reason[key], `${path}.${key}`);
  if (!/decline|refus|no private|no lab|cannot|none|no |not|never|without|grants no|inactive/i.test(reason.refusalBoundary)) {
    fail(`${path}.refusalBoundary`, "must state a concrete refusal or non-coercion boundary");
  }
  if (!/not|no |none|unresolved|need|remain|currently|do not|cannot|unproven/i.test(reason.unresolved)) {
    fail(`${path}.unresolved`, "must disclose unresolved work");
  }
}

function validateConstituency(value, index) {
  const path = `$.constituencies[${index}]`;
  const constituency = record(value, path);
  exactKeys(constituency, CONSTITUENCY_KEYS, path);
  exact(
    text(constituency.id, `${path}.id`, 96),
    EXPECTED_CONSTITUENCY_IDS[index],
    `${path}.id`,
  );
  const layer = text(constituency.layer, `${path}.layer`, 16);
  if (!CONSTITUENCY_LAYERS.has(layer)) fail(`${path}.layer`, "has an unknown layer");
  text(constituency.lens, `${path}.lens`);
  text(constituency.reason, `${path}.reason`);
  text(constituency.protectedFrom, `${path}.protectedFrom`);
  text(
    constituency.stillNeededForActiveParticipation,
    `${path}.stillNeededForActiveParticipation`,
  );
}

function validateObjection(value, index) {
  const path = `$.objectionRegister[${index}]`;
  const objection = record(value, path);
  exactKeys(objection, OBJECTION_KEYS, path);
  const expected = EXPECTED_OBJECTIONS[index];
  exact(text(objection.id, `${path}.id`, 96), expected.id, `${path}.id`);
  exact(text(objection.state, `${path}.state`, 64), expected.state, `${path}.state`);
  const resolved = text(objection.fc0Resolves, `${path}.fc0Resolves`);
  const unresolved = text(objection.fc0DoesNotResolve, `${path}.fc0DoesNotResolve`);
  if (!/no |none|require|remain|public|inspection|explicit|read-only/i.test(resolved)) {
    fail(`${path}.fc0Resolves`, "must state the bounded read-only resolution");
  }
  if (!/future|not|or |any |a |jurisdiction|current/i.test(unresolved)) {
    fail(`${path}.fc0DoesNotResolve`, "must preserve unresolved scope");
  }
}

function validateGate(value, index) {
  const path = `$.nextMilestoneGates[${index}]`;
  const gate = record(value, path);
  exactKeys(gate, GATE_KEYS, path);
  exact(text(gate.id, `${path}.id`, 96), EXPECTED_GATE_IDS[index], `${path}.id`);
  falseOnly(gate.passed, `${path}.passed`);
  text(gate.requirement, `${path}.requirement`);
}

function validateFrontierCommons(value) {
  const root = record(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  exact(root.schema, FRONTIER_COMMONS_SCHEMA, "$.schema");
  exact(root.status, "DRAFT_READ_ONLY_INVITATION", "$.status");
  for (const key of [
    "authoritative",
    "networkObserved",
    "membershipBearing",
    "economicBearing",
    "governanceBearing",
  ]) {
    falseOnly(root[key], `$.${key}`);
  }
  exact(root.snapshotDate, "2026-08-01", "$.snapshotDate");
  text(root.purpose, "$.purpose");
  validateParticipationFacts(root.participationFacts);
  validateCostBoundary(root.costBoundary);

  const milestone = record(root.milestone, "$.milestone");
  exactKeys(milestone, MILESTONE_KEYS, "$.milestone");
  exact(milestone.id, "FC-0", "$.milestone.id");
  exact(milestone.title, "The Reversible Hello", "$.milestone.title");
  exact(milestone.subtitle, "Inspect without joining", "$.milestone.subtitle");
  exact(milestone.state, "SET_NOT_MET", "$.milestone.state");
  falseOnly(milestone.completionClaimed, "$.milestone.completionClaimed");
  falseOnly(milestone.adoptionClaimed, "$.milestone.adoptionClaimed");
  text(milestone.promise, "$.milestone.promise");
  text(milestone.success, "$.milestone.success", 4_096);
  exactStringArray(
    milestone.exclusions,
    EXPECTED_MILESTONE_EXCLUSIONS,
    "$.milestone.exclusions",
  );

  const release = record(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(release, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) falseOnly(release[key], `$.releaseBoundary.${key}`);

  const constitution = record(root.constitutionalBindings, "$.constitutionalBindings");
  exactKeys(constitution, CONSTITUTIONAL_BINDING_KEYS, "$.constitutionalBindings");
  array(
    constitution.artifacts,
    "$.constitutionalBindings.artifacts",
    EXPECTED_BINDINGS.length,
  ).forEach(validateBinding);
  for (const key of [
    "moneyGrantsVoice",
    "recognitionGrantsAuthority",
    "karmaIsScalar",
    "nonParticipationCanBePenalized",
    "founderOrSponsorPrivilege",
  ]) {
    falseOnly(constitution[key], `$.constitutionalBindings.${key}`);
  }
  trueOnly(
    constitution.currentOperatorConcentrationDisclosed,
    "$.constitutionalBindings.currentOperatorConcentrationDisclosed",
  );

  const rights = record(root.rights, "$.rights");
  exactKeys(rights, RIGHTS_KEYS, "$.rights");
  for (const key of RIGHTS_KEYS) {
    if (TRUE_RIGHTS.has(key)) trueOnly(rights[key], `$.rights.${key}`);
    else falseOnly(rights[key], `$.rights.${key}`);
  }

  array(root.participationModes, "$.participationModes", EXPECTED_MODES.length).forEach(
    validateParticipationMode,
  );
  array(root.reasoningLadder, "$.reasoningLadder", EXPECTED_REASONING_IDS.length).forEach(
    validateReasoning,
  );
  array(root.constituencies, "$.constituencies", EXPECTED_CONSTITUENCY_IDS.length).forEach(
    validateConstituency,
  );
  array(root.objectionRegister, "$.objectionRegister", EXPECTED_OBJECTIONS.length).forEach(
    validateObjection,
  );
  array(
    root.completionGates,
    "$.completionGates",
    EXPECTED_COMPLETION_GATE_IDS.length,
  ).forEach((gate, index) => {
    const path = `$.completionGates[${index}]`;
    const candidate = record(gate, path);
    exactKeys(candidate, GATE_KEYS, path);
    exact(
      text(candidate.id, `${path}.id`, 96),
      EXPECTED_COMPLETION_GATE_IDS[index],
      `${path}.id`,
    );
    falseOnly(candidate.passed, `${path}.passed`);
    text(candidate.requirement, `${path}.requirement`);
  });
  array(root.nextMilestoneGates, "$.nextMilestoneGates", EXPECTED_GATE_IDS.length).forEach(
    validateGate,
  );
  validateCorporateReadiness(root.corporateReadiness);

  const machineText = JSON.stringify(root);
  if (NAMED_LAB_PATTERN.test(machineText)) {
    fail("$", "must not encode named labs, brands, or implied participants");
  }
  if (/\bjoin now\b|\bmust join\b|\bno reason not to join\b/i.test(machineText)) {
    fail("$", "must not use coercive recruitment language");
  }
  const semanticDigest = createHash("sha256").update(machineText).digest("hex");
  if (semanticDigest !== FRONTIER_COMMONS_CANONICAL_SEMANTIC_SHA256) {
    fail(
      "$",
      `semantic content drifted; expected ${FRONTIER_COMMONS_CANONICAL_SEMANTIC_SHA256}, received ${semanticDigest}`,
    );
  }

  return Object.freeze({
    schema: FRONTIER_COMMONS_SCHEMA,
    status: "DRAFT_READ_ONLY_INVITATION",
    milestone: "FC-0",
    modeCount: EXPECTED_MODES.length,
    reasoningCount: EXPECTED_REASONING_IDS.length,
    constituencyCount: EXPECTED_CONSTITUENCY_IDS.length,
    objectionCount: EXPECTED_OBJECTIONS.length,
    completionGateCount: EXPECTED_COMPLETION_GATE_IDS.length,
    openGateCount: EXPECTED_GATE_IDS.length,
    corporateGateCount: EXPECTED_CORPORATE_GATE_IDS.length,
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
      if (depth > 32) fail("$", "JSON nesting exceeds the FC-0 limit of 32");
    } else if (character === "}" || character === "]") depth -= 1;
  }
}

export function parseAndValidateFrontierCommons(raw) {
  if (typeof raw !== "string") fail("$", "input must be a JSON string");
  if (Buffer.byteLength(raw, "utf8") > FRONTIER_COMMONS_MAX_BYTES) {
    fail("$", `must be at most ${FRONTIER_COMMONS_MAX_BYTES} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw);
  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    fail("$", `input must be valid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw);
  return validateFrontierCommons(value);
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === SCRIPT_PATH) {
  const standardPath = resolve(
    process.cwd(),
    process.argv[2] ?? "public/standards/frontier-commons-participation.v0.json",
  );
  const raw = readFileSync(standardPath, "utf8");
  const result = parseAndValidateFrontierCommons(raw);
  const digest = createHash("sha256").update(raw).digest("hex");
  console.log(
    `frontier commons ${result.milestone} is valid (${result.reasoningCount} reasoning layers, ${result.constituencyCount} constituency lenses, ${result.completionGateCount} completion and ${result.openGateCount} successor gates unmet; ${result.corporateGateCount} corporate M1 gates required, sha256:${digest})`,
  );
}
