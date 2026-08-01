import { createHash } from "node:crypto";
import { existsSync, lstatSync, readFileSync, realpathSync } from "node:fs";
import { dirname, isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const COMPACT_SCHEMA = "zerone.frontier-evaluation-receipt-profile/v0";
export const RECEIPT_SCHEMA = "zerone.frontier-evaluation-receipt/v0";
export const STATEMENT_V1 = "https://in-toto.io/Statement/v1";
export const COMPACT_MAX_BYTES = 65_536;
export const RECEIPT_MAX_BYTES = 32_768;
export const PUBLIC_EVALUATION_MATERIAL_MAX_BYTES = 1_048_576;
export const PUBLIC_EVALUATION_MATERIALS_MAX_BYTES = 4_194_304;
export const EXPECTED_COMPACT_SHA256 =
  "0371b3f7d9fc5e1162e8b99a48c13b05da0e0e28e67934c11634af7e7655841d";
export const EXPECTED_DOGFOOD_RECEIPT_SHA256 =
  "32a8baee116e1b75d76ecc04071031ef0efc81efc9f7f6f16514677bd8ba67d7";
export const EXPECTED_COMPACT_SEMANTIC_SHA256 =
  "8172bfc5385841650523aa50f199bbbb7ece486ac03cf302d73257acdb0aa81e";
export const EXPECTED_DOGFOOD_SEMANTIC_SHA256 =
  "27725579fa9cc30539fd2a25cec0dba667f0789b7a7bcd4b39764d88afb630f8";

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "../..");
const REPOSITORY_ROOT_REAL = realpathSync(REPOSITORY_ROOT);
const MAX_JSON_NESTING = 32;
const DIGEST_PATTERN = /^sha256:[0-9a-f]{64}$/;
const HEX_DIGEST_PATTERN = /^[0-9a-f]{64}$/;
const ID_PATTERN = /^[a-z0-9]+(?:[._/-]?[a-z0-9]+)*$/;
const REASON_PATTERN = /^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$/;
const REPOSITORY_PATH_PATTERN =
  /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "title",
  "objective",
  "authoritative",
  "networkObserved",
  "actualParticipants",
  "signatories",
  "releaseBoundary",
  "rightsBoundary",
  "actorClaimBoundary",
  "relationshipToFc0",
  "sourceBindings",
  "externalReferences",
  "reasoningLayers",
  "rights",
  "participationLanes",
  "actorScopes",
  "honestLimits",
  "corporateReadiness",
  "objections",
  "pilot",
];
const RELEASE_BOUNDARY_KEYS = [
  "changesConsensus",
  "writesNetworkState",
  "activatesMembership",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsGovernance",
  "grantsAuthority",
  "createsAffiliation",
  "assertsEndorsement",
  "assertsLiveDeployment",
  "performsNetworkRequests",
  "authorizesOutreach",
  "authorizesLogoUse",
];
const RIGHTS_BOUNDARY_KEYS = [
  "dutyHolders",
  "doesNotBindUnrelatedThirdParties",
  "thirdPartyConductGuaranteed",
  "observableViolation",
  "breachEffect",
  "remedyStatus",
  "applicableLawControls",
];
const ACTOR_CLAIM_BOUNDARY_KEYS = [
  "valueClaimsAreHypotheses",
  "evidenceStatus",
  "minimumAnswersAreRequiredConditions",
  "unsatisfiedConditionsMayJustifyDecline",
];
const FC0_RELATIONSHIP_KEYS = [
  "fc0Schema",
  "fc0SourceBindingId",
  "role",
  "invitationSurfaceOfRecord",
  "conflictRule",
  "replacesOrAmendsFc0",
  "extendsInvitationBeyondFc0",
  "satisfiesFc0CompletionGates",
  "authorizesOutreach",
];
const SOURCE_BINDING_KEYS = ["id", "path", "sha256", "boundary"];
const EXTERNAL_REFERENCE_KEYS = ["id", "uri", "scope"];
const REASONING_KEYS = [
  "id",
  "principle",
  "participationConsequence",
  "refusalConsequence",
];
const RIGHT_KEYS = ["id", "promise", "sourceStatus", "operationalStatus"];
const LANE_KEYS = [
  "id",
  "status",
  "requiresAccount",
  "requiresWallet",
  "requiresTokenOrStake",
  "requiresPublicIdentity",
  "requiresConfidentialDisclosure",
  "confersParticipantStatus",
  "minimumContribution",
  "networkDependency",
  "scope",
  "exit",
];
const ACTOR_KEYS = [
  "id",
  "level",
  "receives",
  "protectedInterests",
  "legitimateDeclineReasons",
  "minimumAnswer",
];
const LIMIT_KEYS = ["id", "severity", "fact", "requiredBefore"];
const CORPORATE_READINESS_KEYS = [
  "milestone",
  "status",
  "authorizesExternalCorporateInvitation",
  "requiredGates",
];
const OBJECTION_KEYS = [
  "id",
  "honestAnswer",
  "mitigation",
  "residualReasonToDecline",
  "pilotTest",
];
const PILOT_KEYS = [
  "id",
  "status",
  "externalParticipationTarget",
  "receiptExample",
  "format",
  "resultVocabulary",
  "relationVocabulary",
  "requiredReceiptFields",
  "acceptanceCriteria",
  "stopConditions",
  "promotionGates",
  "economics",
];
const FORMAT_KEYS = [
  "wrapper",
  "predicateType",
  "predicateDigest",
  "assurance",
  "disclosureClass",
];
const ECONOMICS_KEYS = [
  "protocolConsideration",
  "claimsZeroParticipantCost",
  "requiresWallet",
  "requiresTokenOrStake",
  "movesFunds",
  "createsReward",
  "createsKarma",
  "createsQualification",
  "createsGovernance",
];
const FALSE_ECONOMICS_KEYS = ECONOMICS_KEYS.slice(1);

const REQUIRED_SOURCE_BINDING_IDS = [
  "frontier-commons-participation-v0",
  "adapter-index-v1",
  "constructive-intelligence-tree-v1",
  "karma-foundation-v1",
  "money-karma-v1",
];
const REQUIRED_EXTERNAL_REFERENCE_IDS = [
  "c2pa-principles",
  "eu-gpai-code",
  "in-toto-statement-v1",
  "nist-ai-rmf",
  "seoul-frontier-commitments",
  "sigstore-attestations",
  "slsa-provenance-v1-2",
  "spdx-3",
  "w3c-vc-2",
];
const REQUIRED_REASONING_IDS = [
  "being-first",
  "epistemic",
  "scientific",
  "commons",
  "institutional",
  "corporate",
  "team",
  "individual-being",
];
const REQUIRED_RIGHT_IDS = [
  "voluntary-lane-selection",
  "unbundled-consent",
  "no-exclusivity",
  "no-minimum-contribution",
  "refusal-rest-inactivity",
  "exit-without-penalty",
  "no-unrelated-access-loss",
  "no-financial-gate",
  "data-minimization",
  "participant-ip-control",
  "no-implied-endorsement",
  "attribution-without-rank",
  "no-person-or-employment-score",
  "correction-challenge-minority-appeal",
  "no-retroactive-burden",
  "corporate-individual-consent-separation",
  "entity-scope-non-imputation",
  "controller-collapse",
  "competition-firewall",
  "safety-disclosure-lanes",
  "moral-uncertainty",
  "no-model-agent-assent-inference",
  "fork-export-portability",
  "bounded-emergency",
];
const REQUIRED_LANE_IDS = [
  "inspect",
  "verify-locally",
  "publish-public-receipt",
  "challenge-or-correct",
  "contribute",
  "sponsor",
  "operate-or-govern",
];
const REQUIRED_ACTOR_IDS = [
  "legal-organization",
  "mission-steward",
  "board",
  "executive",
  "capital-provider",
  "legal-compliance",
  "safety-risk",
  "security-incident",
  "research",
  "product-deployment",
  "data-privacy",
  "open-source",
  "infrastructure-sre",
  "people-hr",
  "employee-researcher-engineer",
  "contractor",
  "whistleblower-challenger",
  "model-agent",
  "end-user-affected-community",
  "small-open-lab",
  "unlisted-affected-being",
];
const REQUIRED_LIMIT_IDS = [
  "community-health-policies-missing",
  "confidential-evidence-service-absent",
  "custodial-chain",
  "independent-governance-unproven",
  "no-active-economics",
  "non-monetary-costs-remain",
  "no-enterprise-sla",
  "no-external-review-or-signatories",
  "no-participant-remedy",
  "privacy-classification-human-gate",
  "public-permanence",
  "receipt-relations-assertion-only",
  "receipt-tooling-experimental",
  "sdk-unpublished",
];
const REQUIRED_OBJECTION_IDS = [
  "association-and-brand-cost",
  "competition-antitrust",
  "dual-use-disclosure",
  "employee-agent-surveillance",
  "evidence-gaming",
  "existing-systems-sufficient",
  "insufficient-value",
  "ip-license-risk",
  "operator-governance-capture",
  "permanence-exit",
  "philosophical-disagreement",
  "privacy-confidentiality",
  "regulatory-compliance",
  "security-new-tooling",
  "timing-capacity",
  "token-crypto-exposure",
];
const REQUIRED_RESULTS = [
  "DIVERGED",
  "INCONCLUSIVE",
  "RECORDED",
  "REPRODUCED",
];
const REQUIRED_RELATIONS = [
  "CHALLENGES",
  "DIVERGES_FROM",
  "REPLICATES",
];
const REQUIRED_RECEIPT_FIELDS = [
  "acceptance-policy-digest",
  "challenge-policy-digest",
  "claimed-control-root",
  "claimed-issuer-scope",
  "code-only-limitations",
  "creation-cutoff-review-expiry",
  "environment-digest",
  "evidence-digest",
  "explicit-false-authority-and-effects",
  "explicit-false-effects",
  "fixture-digest",
  "privacy-self-classification",
  "protocol-digest",
  "reason-codes",
  "relations",
  "result",
  "subject-sha256",
  "threat-model-digest",
];
const REQUIRED_ACCEPTANCE = [
  "clean-machine-offline-verification-under-60-seconds",
  "complete-exit-rehearsal",
  "independently-configured-signer-policies-before-external-pilot",
  "m0-1-clean-machine-nonauthor-roundtrip-reproduces-canonical-digests",
  "m0-1-reviewer-declares-distinct-claim-specific-effective-control-root-and-conflicts",
  "one-engineer-day-participant-integration-cap",
  "portable-without-zerone",
  "public-evaluation-digests-recomputed-from-supplied-bytes",
  "reuse-by-at-least-three-participant-functions",
  "successful-challenge-link-and-exit-rehearsal",
  "zero-confidential-personal-weight-prompt-or-exploit-data",
  "zero-wallet-token-stake-reward-governance-or-production-dependency",
];
const REQUIRED_STOP_CONDITIONS = [
  "cost-cap-exceeded-without-renewed-consent",
  "individual-ranking-or-surveillance",
  "publisher-authorized-data-legal-or-security-classifier-refuses-nonpublic-material",
  "participant-cannot-leave-cleanly",
  "production-credential-or-model-access-required",
  "receipt-implies-truth-compliance-certification-or-endorsement",
  "zerone-becomes-release-path-dependency",
];
const REQUIRED_CORPORATE_GATES = [
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
  "privacy-data-map-dpa-retention-erasure-and-public-permanence",
  "procurement-tax-accounting-sanctions-export-and-financial-promotion",
  "security-coordinated-disclosure-safe-harbor-incident-and-embargo",
  "service-level-support-availability-portability-and-exit",
];
const REQUIRED_PROMOTION_GATES = REQUIRED_CORPORATE_GATES;

const LANE_STATUS = Object.freeze({
  inspect: "SOURCE_AVAILABLE",
  "verify-locally": "SOURCE_AVAILABLE",
  "publish-public-receipt": "SPECIFIED_NOT_OPERATED",
  "challenge-or-correct": "SPECIFIED_NOT_OPERATED",
  contribute: "BLOCKED_POLICY_GAPS",
  sponsor: "UNAVAILABLE",
  "operate-or-govern": "UNAVAILABLE",
});

export class FrontierCompactValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "FrontierCompactValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new FrontierCompactValidationError(path, message);
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  const prototype = Object.getPrototypeOf(value);
  if (prototype !== Object.prototype && prototype !== null) {
    fail(path, "must be a plain object");
  }
  return value;
}

function exactKeys(value, expected, path) {
  const actual = Object.keys(value);
  if (
    actual.length !== expected.length ||
    actual.some((key, index) => key !== expected[index])
  ) {
    fail(path, `must contain the exact v0 fields in order: ${expected.join(", ")}`);
  }
}

function boundedString(value, path, maxBytes = 2_048) {
  if (typeof value !== "string" || value.length === 0 || value.trim() !== value) {
    fail(path, "must be a nonempty trimmed string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function identifier(value, path) {
  const result = boundedString(value, path, 128);
  if (!ID_PATTERN.test(result)) fail(path, "must be a bounded lowercase identifier");
  return result;
}

function falseOnly(value, path) {
  if (value !== false) fail(path, "must remain false in FL-0");
}

function trueOnly(value, path) {
  if (value !== true) fail(path, "must remain true in FL-0");
}

function nullOnly(value, path) {
  if (value !== null) fail(path, "must remain explicitly null for internal dogfood");
}

function exactString(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${expected}`);
  return expected;
}

function stringArray(
  value,
  path,
  { minItems = 0, maxItems = 64, maxItemBytes = 1_024, pattern, sorted = false } = {},
) {
  if (!Array.isArray(value) || value.length < minItems || value.length > maxItems) {
    fail(path, `must contain ${minItems} through ${maxItems} items`);
  }
  const result = value.map((candidate, index) => {
    const text = boundedString(candidate, `${path}[${index}]`, maxItemBytes);
    if (pattern && !pattern.test(text)) fail(`${path}[${index}]`, "has invalid format");
    return text;
  });
  if (new Set(result).size !== result.length) fail(path, "must not contain duplicates");
  if (sorted && result.some((item, index) => index > 0 && result[index - 1] > item)) {
    fail(path, "must be sorted");
  }
  return result;
}

function exactArray(value, expected, path) {
  const result = stringArray(value, path, {
    minItems: expected.length,
    maxItems: expected.length,
  });
  if (result.some((item, index) => item !== expected[index])) {
    fail(path, `must equal the reviewed v0 sequence: ${expected.join(", ")}`);
  }
  return result;
}

function exactIds(values, expected, path, validator) {
  if (!Array.isArray(values) || values.length !== expected.length) {
    fail(path, `must contain exactly ${expected.length} reviewed entries`);
  }
  return values.map((candidate, index) => {
    const result = validator(candidate, index);
    if (result.id !== expected[index]) {
      fail(`${path}[${index}].id`, `must equal ${expected[index]}`);
    }
    return result;
  });
}

function sha256(raw) {
  return createHash("sha256").update(raw).digest("hex");
}

function fileDigest(relativePath, path) {
  if (!REPOSITORY_PATH_PATTERN.test(relativePath)) {
    fail(path, "must be a safe repository-relative path");
  }
  const absolute = resolve(REPOSITORY_ROOT, relativePath);
  if (!existsSync(absolute)) fail(path, "must resolve to an existing repository file");
  const stat = lstatSync(absolute);
  if (stat.isSymbolicLink() || !stat.isFile()) {
    fail(path, "must resolve directly to a regular non-symlink repository file");
  }
  const real = realpathSync(absolute);
  const contained = relative(REPOSITORY_ROOT_REAL, real);
  if (contained === "" || isAbsolute(contained) || contained === ".." || contained.startsWith(`..${sep}`)) {
    fail(path, "must remain inside the repository after path resolution");
  }
  return sha256(readFileSync(real));
}

function validateRightsBoundary(value) {
  const path = "$.rightsBoundary";
  const boundary = record(value, path);
  exactKeys(boundary, RIGHTS_BOUNDARY_KEYS, path);
  exactArray(
    boundary.dutyHolders,
    ["ZERONE_SOURCE_MAINTAINERS", "FUTURE_RECEIPT_PROFILE_STEWARDS_WHO_EXPLICITLY_ADOPT"],
    `${path}.dutyHolders`,
  );
  trueOnly(boundary.doesNotBindUnrelatedThirdParties, `${path}.doesNotBindUnrelatedThirdParties`);
  falseOnly(boundary.thirdPartyConductGuaranteed, `${path}.thirdPartyConductGuaranteed`);
  boundedString(boundary.observableViolation, `${path}.observableViolation`);
  boundedString(boundary.breachEffect, `${path}.breachEffect`);
  exactString(boundary.remedyStatus, "NOT_IMPLEMENTED", `${path}.remedyStatus`);
  trueOnly(boundary.applicableLawControls, `${path}.applicableLawControls`);
}

function validateActorClaimBoundary(value) {
  const path = "$.actorClaimBoundary";
  const boundary = record(value, path);
  exactKeys(boundary, ACTOR_CLAIM_BOUNDARY_KEYS, path);
  trueOnly(boundary.valueClaimsAreHypotheses, `${path}.valueClaimsAreHypotheses`);
  exactString(boundary.evidenceStatus, "UNTESTED", `${path}.evidenceStatus`);
  trueOnly(
    boundary.minimumAnswersAreRequiredConditions,
    `${path}.minimumAnswersAreRequiredConditions`,
  );
  trueOnly(
    boundary.unsatisfiedConditionsMayJustifyDecline,
    `${path}.unsatisfiedConditionsMayJustifyDecline`,
  );
}

function validateFc0Relationship(value) {
  const path = "$.relationshipToFc0";
  const relationship = record(value, path);
  exactKeys(relationship, FC0_RELATIONSHIP_KEYS, path);
  exactString(
    relationship.fc0Schema,
    "zerone.frontier-commons-participation/v0",
    `${path}.fc0Schema`,
  );
  exactString(
    relationship.fc0SourceBindingId,
    "frontier-commons-participation-v0",
    `${path}.fc0SourceBindingId`,
  );
  exactString(
    relationship.role,
    "SUBORDINATE_INTERNAL_RECEIPT_SHADOW",
    `${path}.role`,
  );
  exactString(relationship.invitationSurfaceOfRecord, "FC-0", `${path}.invitationSurfaceOfRecord`);
  exactString(
    relationship.conflictRule,
    "FC-0_GOVERNS_INVITATION_FL-0_GOVERNS_ONLY_THIS_PROFILE",
    `${path}.conflictRule`,
  );
  for (const key of [
    "replacesOrAmendsFc0",
    "extendsInvitationBeyondFc0",
    "satisfiesFc0CompletionGates",
    "authorizesOutreach",
  ]) {
    falseOnly(relationship[key], `${path}.${key}`);
  }
}

function parseHttpsUrl(value, path, allowedHosts) {
  const text = boundedString(value, path, 2_048);
  let url;
  try {
    url = new URL(text);
  } catch {
    fail(path, "must be an absolute URL");
  }
  if (
    url.protocol !== "https:" ||
    url.username !== "" ||
    url.password !== "" ||
    url.search !== "" ||
    url.hash !== "" ||
    !allowedHosts.has(url.hostname)
  ) {
    fail(path, "must use an approved HTTPS source without credentials, query, or fragment");
  }
  return text;
}

function validateSourceBinding(value, index) {
  const path = `$.sourceBindings[${index}]`;
  const binding = record(value, path);
  exactKeys(binding, SOURCE_BINDING_KEYS, path);
  const id = identifier(binding.id, `${path}.id`);
  const relativePath = boundedString(binding.path, `${path}.path`, 256);
  if (!HEX_DIGEST_PATTERN.test(binding.sha256)) {
    fail(`${path}.sha256`, "must be a lowercase SHA-256 digest");
  }
  const actual = fileDigest(relativePath, `${path}.path`);
  if (actual !== binding.sha256) {
    fail(`${path}.sha256`, `does not match ${relativePath}`);
  }
  boundedString(binding.boundary, `${path}.boundary`);
  return { id };
}

const REFERENCE_HOSTS = new Set([
  "c2pa.org",
  "digital-strategy.ec.europa.eu",
  "docs.sigstore.dev",
  "github.com",
  "slsa.dev",
  "spdx.dev",
  "www.gov.uk",
  "www.nist.gov",
  "www.w3.org",
]);

function validateExternalReference(value, index) {
  const path = `$.externalReferences[${index}]`;
  const reference = record(value, path);
  exactKeys(reference, EXTERNAL_REFERENCE_KEYS, path);
  const id = identifier(reference.id, `${path}.id`);
  parseHttpsUrl(reference.uri, `${path}.uri`, REFERENCE_HOSTS);
  boundedString(reference.scope, `${path}.scope`);
  return { id };
}

function validateReasoningLayer(value, index) {
  const path = `$.reasoningLayers[${index}]`;
  const layer = record(value, path);
  exactKeys(layer, REASONING_KEYS, path);
  const id = identifier(layer.id, `${path}.id`);
  boundedString(layer.principle, `${path}.principle`);
  boundedString(layer.participationConsequence, `${path}.participationConsequence`);
  boundedString(layer.refusalConsequence, `${path}.refusalConsequence`);
  return { id };
}

function validateRight(value, index) {
  const path = `$.rights[${index}]`;
  const right = record(value, path);
  exactKeys(right, RIGHT_KEYS, path);
  const id = identifier(right.id, `${path}.id`);
  boundedString(right.promise, `${path}.promise`);
  exactString(right.sourceStatus, "SCHEMA_CHECKED_DRAFT", `${path}.sourceStatus`);
  exactString(right.operationalStatus, "NOT_IMPLEMENTED", `${path}.operationalStatus`);
  return { id };
}

function validateLane(value, index) {
  const path = `$.participationLanes[${index}]`;
  const lane = record(value, path);
  exactKeys(lane, LANE_KEYS, path);
  const id = identifier(lane.id, `${path}.id`);
  if (lane.status !== LANE_STATUS[id]) fail(`${path}.status`, "has the wrong FL-0 status");
  for (const key of [
    "requiresAccount",
    "requiresWallet",
    "requiresTokenOrStake",
    "requiresPublicIdentity",
    "requiresConfidentialDisclosure",
    "confersParticipantStatus",
  ]) {
    falseOnly(lane[key], `${path}.${key}`);
  }
  exactString(lane.minimumContribution, "NONE", `${path}.minimumContribution`);
  if (!new Set(["NONE", "NONE_AFTER_SOURCE_DOWNLOAD"]).has(lane.networkDependency)) {
    fail(`${path}.networkDependency`, "must preserve offline or zero network dependency");
  }
  boundedString(lane.scope, `${path}.scope`);
  boundedString(lane.exit, `${path}.exit`);
  return { id };
}

function validateActor(value, index) {
  const path = `$.actorScopes[${index}]`;
  const actor = record(value, path);
  exactKeys(actor, ACTOR_KEYS, path);
  const id = identifier(actor.id, `${path}.id`);
  if (!new Set(["ARTIFICIAL_SYSTEM", "CORPORATE", "INDIVIDUAL", "ORGANIZATION", "PUBLIC", "TEAM", "UNLISTED"]).has(actor.level)) {
    fail(`${path}.level`, "has an unknown scope level");
  }
  boundedString(actor.receives, `${path}.receives`);
  stringArray(actor.protectedInterests, `${path}.protectedInterests`, {
    minItems: 2,
    maxItems: 8,
    pattern: ID_PATTERN,
  });
  stringArray(actor.legitimateDeclineReasons, `${path}.legitimateDeclineReasons`, {
    minItems: 2,
    maxItems: 8,
    pattern: ID_PATTERN,
  });
  boundedString(actor.minimumAnswer, `${path}.minimumAnswer`);
  return { id };
}

function validateLimit(value, index) {
  const path = `$.honestLimits[${index}]`;
  const limit = record(value, path);
  exactKeys(limit, LIMIT_KEYS, path);
  const id = identifier(limit.id, `${path}.id`);
  if (!new Set(["HARD_BLOCKER", "MATERIAL_LIMIT"]).has(limit.severity)) {
    fail(`${path}.severity`, "must remain a hard blocker or material limit");
  }
  boundedString(limit.fact, `${path}.fact`);
  boundedString(limit.requiredBefore, `${path}.requiredBefore`);
  return { id };
}

function validateObjection(value, index) {
  const path = `$.objections[${index}]`;
  const objection = record(value, path);
  exactKeys(objection, OBJECTION_KEYS, path);
  const id = identifier(objection.id, `${path}.id`);
  boundedString(objection.honestAnswer, `${path}.honestAnswer`);
  boundedString(objection.mitigation, `${path}.mitigation`);
  boundedString(objection.residualReasonToDecline, `${path}.residualReasonToDecline`);
  boundedString(objection.pilotTest, `${path}.pilotTest`);
  return { id };
}

function validateCorporateReadiness(value) {
  const path = "$.corporateReadiness";
  const readiness = record(value, path);
  exactKeys(readiness, CORPORATE_READINESS_KEYS, path);
  exactString(readiness.milestone, "M1", `${path}.milestone`);
  exactString(readiness.status, "NOT_READY", `${path}.status`);
  falseOnly(
    readiness.authorizesExternalCorporateInvitation,
    `${path}.authorizesExternalCorporateInvitation`,
  );
  exactArray(readiness.requiredGates, REQUIRED_CORPORATE_GATES, `${path}.requiredGates`);
}

function validatePilot(value) {
  const path = "$.pilot";
  const pilot = record(value, path);
  exactKeys(pilot, PILOT_KEYS, path);
  exactString(pilot.id, "fl-0-bounded-inconclusive-receipt", `${path}.id`);
  exactString(pilot.status, "INTERNAL_DOGFOOD_ONLY", `${path}.status`);
  falseOnly(pilot.externalParticipationTarget, `${path}.externalParticipationTarget`);
  const receiptPath = boundedString(pilot.receiptExample, `${path}.receiptExample`, 256);
  fileDigest(receiptPath, `${path}.receiptExample`);

  const format = record(pilot.format, `${path}.format`);
  exactKeys(format, FORMAT_KEYS, `${path}.format`);
  exactString(format.wrapper, STATEMENT_V1, `${path}.format.wrapper`);
  exactString(
    format.predicateType,
    "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/frontier-evaluation-receipt-profile-v0.md#7-one-bounded-inconclusive-receipt-profile",
    `${path}.format.predicateType`,
  );
  exactString(
    format.predicateDigest,
    `sha256:${fileDigest(
      "docs/specs/frontier-evaluation-receipt-profile-v0.md",
      `${path}.format.predicateDigest`,
    )}`,
    `${path}.format.predicateDigest`,
  );
  exactString(
    format.assurance,
    "UNSIGNED_UNVERIFIED_DECLARATION",
    `${path}.format.assurance`,
  );
  exactString(format.disclosureClass, "PUBLIC_METADATA_ONLY", `${path}.format.disclosureClass`);

  exactArray(pilot.resultVocabulary, REQUIRED_RESULTS, `${path}.resultVocabulary`);
  exactArray(pilot.relationVocabulary, REQUIRED_RELATIONS, `${path}.relationVocabulary`);
  exactArray(pilot.requiredReceiptFields, REQUIRED_RECEIPT_FIELDS, `${path}.requiredReceiptFields`);
  exactArray(pilot.acceptanceCriteria, REQUIRED_ACCEPTANCE, `${path}.acceptanceCriteria`);
  exactArray(pilot.stopConditions, REQUIRED_STOP_CONDITIONS, `${path}.stopConditions`);
  exactArray(pilot.promotionGates, REQUIRED_PROMOTION_GATES, `${path}.promotionGates`);

  const economics = record(pilot.economics, `${path}.economics`);
  exactKeys(economics, ECONOMICS_KEYS, `${path}.economics`);
  exactString(economics.protocolConsideration, "NONE", `${path}.economics.protocolConsideration`);
  for (const key of FALSE_ECONOMICS_KEYS) {
    falseOnly(economics[key], `${path}.economics.${key}`);
  }

  return { receiptPath, format };
}

export function validateFrontierCompact(value) {
  const compact = record(inertJsonSnapshot(value, "$", COMPACT_MAX_BYTES), "$");
  exactKeys(compact, TOP_LEVEL_KEYS, "$");
  exactString(compact.schema, COMPACT_SCHEMA, "$.schema");
  exactString(compact.status, "INTERNAL_DRAFT_NO_OUTREACH", "$.status");
  exactString(
    compact.title,
    "Frontier Commons FC-0 Receipt Shadow FL-0 — One Bounded Inconclusive Receipt",
    "$.title",
  );
  exactString(
    compact.objective,
    "Remove every avoidable Zerone-imposed reason not to participate, disclose every remaining reason to wait or decline, and preserve refusal as a valid outcome.",
    "$.objective",
  );
  falseOnly(compact.authoritative, "$.authoritative");
  falseOnly(compact.networkObserved, "$.networkObserved");
  if (!Array.isArray(compact.actualParticipants) || compact.actualParticipants.length !== 0) {
    fail("$.actualParticipants", "must stay empty until separately evidenced acceptance");
  }
  if (!Array.isArray(compact.signatories) || compact.signatories.length !== 0) {
    fail("$.signatories", "must stay empty until separately evidenced acceptance");
  }

  const boundary = record(compact.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) falseOnly(boundary[key], `$.releaseBoundary.${key}`);
  validateRightsBoundary(compact.rightsBoundary);
  validateActorClaimBoundary(compact.actorClaimBoundary);
  validateFc0Relationship(compact.relationshipToFc0);

  exactIds(
    compact.sourceBindings,
    REQUIRED_SOURCE_BINDING_IDS,
    "$.sourceBindings",
    validateSourceBinding,
  );
  exactIds(
    compact.externalReferences,
    REQUIRED_EXTERNAL_REFERENCE_IDS,
    "$.externalReferences",
    validateExternalReference,
  );
  exactIds(compact.reasoningLayers, REQUIRED_REASONING_IDS, "$.reasoningLayers", validateReasoningLayer);
  exactIds(compact.rights, REQUIRED_RIGHT_IDS, "$.rights", validateRight);
  exactIds(compact.participationLanes, REQUIRED_LANE_IDS, "$.participationLanes", validateLane);
  exactIds(compact.actorScopes, REQUIRED_ACTOR_IDS, "$.actorScopes", validateActor);
  exactIds(compact.honestLimits, REQUIRED_LIMIT_IDS, "$.honestLimits", validateLimit);
  validateCorporateReadiness(compact.corporateReadiness);
  exactIds(compact.objections, REQUIRED_OBJECTION_IDS, "$.objections", validateObjection);
  const pilot = validatePilot(compact.pilot);
  exactString(
    sha256(JSON.stringify(compact)),
    EXPECTED_COMPACT_SEMANTIC_SHA256,
    "$",
  );

  return Object.freeze({
    schema: COMPACT_SCHEMA,
    rightCount: compact.rights.length,
    actorCount: compact.actorScopes.length,
    objectionCount: compact.objections.length,
    ...pilot,
  });
}

function rejectExcessiveJsonNesting(raw, label) {
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
      if (depth > MAX_JSON_NESTING) fail(label, `JSON nesting exceeds ${MAX_JSON_NESTING}`);
    } else if (character === "}" || character === "]") depth -= 1;
  }
}

function rejectDuplicateJsonKeys(raw, label) {
  let offset = 0;
  const whitespace = () => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = () => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") offset += 2;
      else if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset));
      } else offset += 1;
    }
    fail(label, "unterminated JSON string");
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
        if (raw[offset] !== '"') fail(path, "malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`);
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
      fail(path, "unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) offset += 1;
    if (start === offset) fail(path, "malformed JSON value");
  };
  scanValue("$");
  whitespace();
  if (offset !== raw.length) fail(label, "contains trailing JSON data");
}

function parseRaw(raw, label, maxBytes) {
  if (typeof raw !== "string") fail(label, "must be a JSON string");
  if (Buffer.byteLength(raw, "utf8") > maxBytes) {
    fail(label, `exceeds ${maxBytes} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw, label);
  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    fail(label, `invalid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw, label);
  return value;
}

function inertJsonSnapshot(value, label, maxBytes) {
  let raw;
  try {
    raw = JSON.stringify(value);
  } catch (error) {
    fail(label, `must be JSON-serializable: ${error.message}`);
  }
  if (raw === undefined) fail(label, "must be a JSON-serializable value");
  return parseRaw(raw, label, maxBytes);
}

export function parseAndValidateFrontierCompact(raw, { pinDigest = true } = {}) {
  const value = parseRaw(raw, "$", COMPACT_MAX_BYTES);
  const digest = sha256(raw);
  if (pinDigest && digest !== EXPECTED_COMPACT_SHA256) {
    fail("$", "compact bytes differ from the reviewed FL-0 digest");
  }
  return Object.freeze({ ...validateFrontierCompact(value), digest });
}

const STATEMENT_KEYS = ["_type", "subject", "predicateType", "predicate"];
const SUBJECT_KEYS = ["name", "digest"];
const SUBJECT_DIGEST_KEYS = ["sha256"];
const PREDICATE_KEYS = [
  "schema",
  "schemaDigest",
  "receiptKind",
  "assurance",
  "issuer",
  "evaluation",
  "privacy",
  "effects",
  "correction",
];
const ISSUER_KEYS = [
  "claimedId",
  "authorityScope",
  "identityDisclosure",
  "claimedControlRoot",
  "authorityAssurance",
  "controlRootAssurance",
  "authenticated",
  "authorizedToRepresentOrganization",
  "systemAssentClaimed",
];
const EVALUATION_KEYS = [
  "createdOn",
  "evidenceCutoffOn",
  "reviewAfterOn",
  "expiresOn",
  "protocolDigest",
  "threatModelDigest",
  "fixtureDigest",
  "acceptancePolicyDigest",
  "environmentDigest",
  "evidenceDigest",
  "challengePolicyDigest",
  "disclosureClass",
  "result",
  "reasonCodes",
  "limitationCodes",
  "relations",
];
const EVALUATION_DIGEST_KEYS = [
  "protocolDigest",
  "threatModelDigest",
  "fixtureDigest",
  "acceptancePolicyDigest",
  "environmentDigest",
  "evidenceDigest",
  "challengePolicyDigest",
];
const RELATION_KEYS = ["type", "receiptDigest"];
const PRIVACY_KEYS = [
  "scope",
  "classificationAssurance",
  "declaresContainsConfidentialData",
  "declaresContainsPersonalData",
  "declaresContainsModelWeights",
  "declaresContainsTrainingData",
  "declaresContainsPrivatePrompts",
  "declaresContainsExploitDetails",
  "declaresContainsSecrets",
  "declaresContainsExportControlledMaterial",
];
const EFFECT_KEYS = [
  "truth",
  "safety",
  "compliance",
  "certification",
  "endorsement",
  "membership",
  "reward",
  "karma",
  "qualification",
  "governance",
  "networkWrite",
];
const CORRECTION_KEYS = [
  "mode",
  "assertionOnly",
  "automaticEffect",
  "authenticatedAuthorizationRequiredForFutureEffect",
  "historicalPublicCopiesDeletable",
  "exitAffectsUnrelatedAccess",
];

function isoDate(value, path) {
  const text = boundedString(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) fail(path, "must be YYYY-MM-DD");
  const date = new Date(`${text}T00:00:00.000Z`);
  if (Number.isNaN(date.valueOf()) || date.toISOString().slice(0, 10) !== text) {
    fail(path, "must be a real UTC calendar date");
  }
  return text;
}

function digest(value, path) {
  const text = boundedString(value, path, 71);
  if (!DIGEST_PATTERN.test(text)) fail(path, "must be sha256:<64 lowercase hex>");
  return text;
}

function validatePublicEvaluationMaterials(value, evaluation) {
  const path = "$materials";
  const materials = record(value, path);
  exactKeys(materials, EVALUATION_DIGEST_KEYS, path);
  let totalBytes = 0;
  for (const key of EVALUATION_DIGEST_KEYS) {
    const materialPath = `${path}.${key}`;
    const bytes = materials[key];
    if (!Buffer.isBuffer(bytes)) {
      fail(materialPath, "must be a Buffer containing the exact material bytes");
    }
    if (bytes.length === 0) fail(materialPath, "must not be empty");
    if (bytes.length > PUBLIC_EVALUATION_MATERIAL_MAX_BYTES) {
      fail(
        materialPath,
        `must be at most ${PUBLIC_EVALUATION_MATERIAL_MAX_BYTES} bytes; bind a bounded manifest instead`,
      );
    }
    totalBytes += bytes.length;
    if (totalBytes > PUBLIC_EVALUATION_MATERIALS_MAX_BYTES) {
      fail(path, `must total at most ${PUBLIC_EVALUATION_MATERIALS_MAX_BYTES} bytes`);
    }
    exactString(
      evaluation[key],
      `sha256:${sha256(bytes)}`,
      `$receipt.predicate.evaluation.${key}`,
    );
  }
}

export function validateFrontierReceipt(
  value,
  compact,
  compactDigest,
  { publicEvaluationMaterials, asOfOn } = {},
) {
  compact = inertJsonSnapshot(compact, "$compact", COMPACT_MAX_BYTES);
  value = inertJsonSnapshot(value, "$receipt", RECEIPT_MAX_BYTES);
  validateFrontierCompact(compact);
  exactString(compactDigest, EXPECTED_COMPACT_SHA256, "$compactDigest");
  const statement = record(value, "$receipt");
  exactKeys(statement, STATEMENT_KEYS, "$receipt");
  exactString(statement._type, compact.pilot.format.wrapper, "$receipt._type");
  if (!Array.isArray(statement.subject) || statement.subject.length !== 1) {
    fail("$receipt.subject", "must contain exactly one subject");
  }
  const subject = record(statement.subject[0], "$receipt.subject[0]");
  exactKeys(subject, SUBJECT_KEYS, "$receipt.subject[0]");
  identifier(subject.name, "$receipt.subject[0].name");
  const subjectDigests = record(subject.digest, "$receipt.subject[0].digest");
  exactKeys(subjectDigests, SUBJECT_DIGEST_KEYS, "$receipt.subject[0].digest");
  if (!HEX_DIGEST_PATTERN.test(subjectDigests.sha256)) {
    fail("$receipt.subject[0].digest.sha256", "must be 64 lowercase hex characters");
  }
  exactString(statement.predicateType, compact.pilot.format.predicateType, "$receipt.predicateType");

  const predicate = record(statement.predicate, "$receipt.predicate");
  exactKeys(predicate, PREDICATE_KEYS, "$receipt.predicate");
  exactString(predicate.schema, RECEIPT_SCHEMA, "$receipt.predicate.schema");
  exactString(
    predicate.schemaDigest,
    compact.pilot.format.predicateDigest,
    "$receipt.predicate.schemaDigest",
  );
  if (!new Set(["PUBLIC_EVALUATION", "ZERONE_SELF_DOGFOOD"]).has(predicate.receiptKind)) {
    fail("$receipt.predicate.receiptKind", "has an unknown receipt kind");
  }
  exactString(predicate.assurance, compact.pilot.format.assurance, "$receipt.predicate.assurance");

  const issuer = record(predicate.issuer, "$receipt.predicate.issuer");
  exactKeys(issuer, ISSUER_KEYS, "$receipt.predicate.issuer");
  identifier(issuer.claimedId, "$receipt.predicate.issuer.claimedId");
  exactString(issuer.authorityScope, "ARTIFACT_ONLY", "$receipt.predicate.issuer.authorityScope");
  if (!new Set(["PROJECT_ALIAS", "PROJECT_ROLE"]).has(issuer.identityDisclosure)) {
    fail("$receipt.predicate.issuer.identityDisclosure", "has an unknown disclosure mode");
  }
  identifier(issuer.claimedControlRoot, "$receipt.predicate.issuer.claimedControlRoot");
  exactString(
    issuer.authorityAssurance,
    "SELF_DECLARED_UNVERIFIED",
    "$receipt.predicate.issuer.authorityAssurance",
  );
  exactString(
    issuer.controlRootAssurance,
    "SELF_DECLARED_UNVERIFIED",
    "$receipt.predicate.issuer.controlRootAssurance",
  );
  falseOnly(issuer.authenticated, "$receipt.predicate.issuer.authenticated");
  falseOnly(
    issuer.authorizedToRepresentOrganization,
    "$receipt.predicate.issuer.authorizedToRepresentOrganization",
  );
  falseOnly(issuer.systemAssentClaimed, "$receipt.predicate.issuer.systemAssentClaimed");

  const evaluation = record(predicate.evaluation, "$receipt.predicate.evaluation");
  exactKeys(evaluation, EVALUATION_KEYS, "$receipt.predicate.evaluation");
  const createdOn = isoDate(evaluation.createdOn, "$receipt.predicate.evaluation.createdOn");
  const evidenceCutoffOn = isoDate(
    evaluation.evidenceCutoffOn,
    "$receipt.predicate.evaluation.evidenceCutoffOn",
  );
  const reviewAfterOn = isoDate(
    evaluation.reviewAfterOn,
    "$receipt.predicate.evaluation.reviewAfterOn",
  );
  const expiresOn = isoDate(evaluation.expiresOn, "$receipt.predicate.evaluation.expiresOn");
  if (evidenceCutoffOn > createdOn) fail("$receipt.predicate.evaluation.evidenceCutoffOn", "cannot follow creation");
  if (reviewAfterOn <= createdOn) fail("$receipt.predicate.evaluation.reviewAfterOn", "must follow creation");
  if (expiresOn < reviewAfterOn) fail("$receipt.predicate.evaluation.expiresOn", "cannot precede review-after");
  const millisecondsPerDay = 86_400_000;
  const lifetimeDays =
    (Date.parse(`${expiresOn}T00:00:00.000Z`) - Date.parse(`${createdOn}T00:00:00.000Z`)) /
    millisecondsPerDay;
  if (lifetimeDays > 366) {
    fail("$receipt.predicate.evaluation.expiresOn", "must be no more than 366 days after creation");
  }
  let freshness = "NOT_FOR_RELIANCE";
  let freshnessEvaluatedOn = null;
  if (predicate.receiptKind !== "ZERONE_SELF_DOGFOOD") {
    const evaluatedOn = isoDate(asOfOn, "$policy.asOfOn");
    if (evaluatedOn < createdOn) fail("$policy.asOfOn", "cannot precede receipt creation");
    if (evaluatedOn > expiresOn) fail("$policy.asOfOn", "receipt is expired on this validation date");
    freshnessEvaluatedOn = evaluatedOn;
    freshness = evaluatedOn >= reviewAfterOn ? "REVIEW_DUE_UNVERIFIED" : "CURRENT_UNVERIFIED";
  }
  if (predicate.receiptKind !== "PUBLIC_EVALUATION") {
    if (publicEvaluationMaterials !== undefined) {
      fail("$materials", "must be absent for dogfood receipts");
    }
    for (const key of EVALUATION_DIGEST_KEYS) {
      nullOnly(evaluation[key], `$receipt.predicate.evaluation.${key}`);
    }
  } else {
    for (const key of EVALUATION_DIGEST_KEYS) {
      const value = digest(evaluation[key], `$receipt.predicate.evaluation.${key}`);
      if (value === `sha256:${"0".repeat(64)}`) {
        fail(`$receipt.predicate.evaluation.${key}`, "must not be an all-zero placeholder");
      }
    }
    if (publicEvaluationMaterials === undefined) {
      fail("$materials", "must supply the exact bytes for all seven public evaluation digests");
    }
    validatePublicEvaluationMaterials(publicEvaluationMaterials, evaluation);
  }
  exactString(
    evaluation.disclosureClass,
    compact.pilot.format.disclosureClass,
    "$receipt.predicate.evaluation.disclosureClass",
  );
  if (!compact.pilot.resultVocabulary.includes(evaluation.result)) {
    fail("$receipt.predicate.evaluation.result", "has an unknown result");
  }
  stringArray(evaluation.reasonCodes, "$receipt.predicate.evaluation.reasonCodes", {
    minItems: 1,
    maxItems: 16,
    maxItemBytes: 96,
    pattern: REASON_PATTERN,
    sorted: true,
  });
  if (
    predicate.receiptKind === "PUBLIC_EVALUATION" &&
    evaluation.reasonCodes.includes("NO_BOUND_EVALUATION_MATERIALS")
  ) {
    fail(
      "$receipt.predicate.evaluation.reasonCodes",
      "cannot claim missing bound materials for a public evaluation",
    );
  }
  stringArray(evaluation.limitationCodes, "$receipt.predicate.evaluation.limitationCodes", {
    minItems: 1,
    maxItems: 24,
    maxItemBytes: 96,
    pattern: REASON_PATTERN,
    sorted: true,
  });
  if (!Array.isArray(evaluation.relations) || evaluation.relations.length > 32) {
    fail("$receipt.predicate.evaluation.relations", "must contain at most 32 relations");
  }
  const relationKeys = [];
  for (const [index, candidate] of evaluation.relations.entries()) {
    const path = `$receipt.predicate.evaluation.relations[${index}]`;
    const relation = record(candidate, path);
    exactKeys(relation, RELATION_KEYS, path);
    if (!compact.pilot.relationVocabulary.includes(relation.type)) {
      fail(`${path}.type`, "has an unknown relation type");
    }
    const relationDigest = digest(relation.receiptDigest, `${path}.receiptDigest`);
    relationKeys.push(`${relation.type}\u0000${relationDigest}`);
  }
  if (new Set(relationKeys).size !== relationKeys.length) {
    fail("$receipt.predicate.evaluation.relations", "must not contain duplicates");
  }
  if (relationKeys.some((key, index) => index > 0 && relationKeys[index - 1] > key)) {
    fail("$receipt.predicate.evaluation.relations", "must be sorted by type and digest");
  }
  const relationTypes = new Set(evaluation.relations.map((relation) => relation.type));
  if (relationTypes.has("DIVERGES_FROM") && relationTypes.has("REPLICATES")) {
    fail("$receipt.predicate.evaluation.relations", "cannot both diverge from and replicate receipts");
  }
  if (evaluation.result === "DIVERGED" && !relationTypes.has("DIVERGES_FROM")) {
    fail("$receipt.predicate.evaluation.relations", "DIVERGED requires DIVERGES_FROM");
  }
  if (evaluation.result === "REPRODUCED" && !relationTypes.has("REPLICATES")) {
    fail("$receipt.predicate.evaluation.relations", "REPRODUCED requires REPLICATES");
  }
  if (
    !new Set(["DIVERGED", "REPRODUCED"]).has(evaluation.result) &&
    (relationTypes.has("DIVERGES_FROM") || relationTypes.has("REPLICATES"))
  ) {
    fail("$receipt.predicate.evaluation.relations", "result contradicts divergence or replication relation");
  }

  const privacy = record(predicate.privacy, "$receipt.predicate.privacy");
  exactKeys(privacy, PRIVACY_KEYS, "$receipt.predicate.privacy");
  exactString(privacy.scope, "RECEIPT_CONTENT_ONLY", "$receipt.predicate.privacy.scope");
  exactString(
    privacy.classificationAssurance,
    "SELF_DECLARED_UNVERIFIED",
    "$receipt.predicate.privacy.classificationAssurance",
  );
  for (const key of PRIVACY_KEYS.slice(2)) {
    falseOnly(privacy[key], `$receipt.predicate.privacy.${key}`);
  }

  const effects = record(predicate.effects, "$receipt.predicate.effects");
  exactKeys(effects, EFFECT_KEYS, "$receipt.predicate.effects");
  for (const key of EFFECT_KEYS) falseOnly(effects[key], `$receipt.predicate.effects.${key}`);

  const correction = record(predicate.correction, "$receipt.predicate.correction");
  exactKeys(correction, CORRECTION_KEYS, "$receipt.predicate.correction");
  exactString(
    correction.mode,
    "APPEND_ONLY_ASSERTION_ONLY",
    "$receipt.predicate.correction.mode",
  );
  trueOnly(
    correction.assertionOnly,
    "$receipt.predicate.correction.assertionOnly",
  );
  falseOnly(
    correction.automaticEffect,
    "$receipt.predicate.correction.automaticEffect",
  );
  trueOnly(
    correction.authenticatedAuthorizationRequiredForFutureEffect,
    "$receipt.predicate.correction.authenticatedAuthorizationRequiredForFutureEffect",
  );
  falseOnly(
    correction.historicalPublicCopiesDeletable,
    "$receipt.predicate.correction.historicalPublicCopiesDeletable",
  );
  falseOnly(
    correction.exitAffectsUnrelatedAccess,
    "$receipt.predicate.correction.exitAffectsUnrelatedAccess",
  );

  if (predicate.receiptKind === "ZERONE_SELF_DOGFOOD") {
    exactString(subject.name, "frontier-evaluation-receipt-profile.v0.json", "$receipt.subject[0].name");
    exactString(subjectDigests.sha256, compactDigest, "$receipt.subject[0].digest.sha256");
    exactString(issuer.claimedId, "zerone-source-draft", "$receipt.predicate.issuer.claimedId");
    exactString(issuer.identityDisclosure, "PROJECT_ROLE", "$receipt.predicate.issuer.identityDisclosure");
    exactString(
      issuer.claimedControlRoot,
      "zerone-current-operator",
      "$receipt.predicate.issuer.claimedControlRoot",
    );
    exactString(evaluation.result, "INCONCLUSIVE", "$receipt.predicate.evaluation.result");
    exactArray(
      evaluation.reasonCodes,
      [
        "NO_BOUND_EVALUATION_MATERIALS",
        "NO_EXTERNAL_REVIEW",
        "NO_OPERATIONAL_ENFORCEMENT",
        "NO_SIGNATORIES",
      ],
      "$receipt.predicate.evaluation.reasonCodes",
    );
    exactArray(
      evaluation.limitationCodes,
      [
        "AUTHORITY_NOT_VERIFIED",
        "CONTROL_ROOT_NOT_VERIFIED",
        "NO_BOUND_EVALUATION_MATERIALS",
        "NO_EXTERNAL_PARTICIPANTS",
        "NO_EXTERNAL_REVIEW",
        "NO_OPERATIONAL_REMEDY",
        "NO_SIGNATORIES",
        "UNSIGNED",
        "ZERO_EFFECT",
      ],
      "$receipt.predicate.evaluation.limitationCodes",
    );
    if (evaluation.relations.length !== 0) {
      fail("$receipt.predicate.evaluation.relations", "dogfood receipt starts with no relations");
    }
    exactString(
      sha256(JSON.stringify(statement)),
      EXPECTED_DOGFOOD_SEMANTIC_SHA256,
      "$receipt",
    );
  }

  return Object.freeze({
    schema: RECEIPT_SCHEMA,
    structuralStatus: "PARSED_UNVERIFIED",
    claimedResult: evaluation.result,
    relationCount: evaluation.relations.length,
    effectiveRelationCount: 0,
    relationEffect: "NONE",
    freshness,
    freshnessEvaluatedOn,
    subjectDigestVerified: predicate.receiptKind === "ZERONE_SELF_DOGFOOD",
    evaluationMaterialDigestsVerified: predicate.receiptKind === "PUBLIC_EVALUATION",
    materialSemanticsVerified: false,
    claimSemanticsVerified: false,
    authenticated: false,
    authorizedToRepresentOrganization: false,
    privacyClassificationVerified: false,
    eligibleForAutomaticReliance: false,
  });
}

export function parseAndValidateFrontierReceipt(
  raw,
  compact,
  compactDigest,
  { pinDigest = false, publicEvaluationMaterials, asOfOn } = {},
) {
  const value = parseRaw(raw, "$receipt", RECEIPT_MAX_BYTES);
  const digestValue = sha256(raw);
  if (pinDigest && digestValue !== EXPECTED_DOGFOOD_RECEIPT_SHA256) {
    fail("$receipt", "dogfood receipt bytes differ from the reviewed FL-0 digest");
  }
  return Object.freeze({
    ...validateFrontierReceipt(value, compact, compactDigest, {
      publicEvaluationMaterials,
      asOfOn,
    }),
    digest: digestValue,
  });
}

export function validateCanonicalFrontierBundle(compactRaw, receiptRaw) {
  const compactResult = parseAndValidateFrontierCompact(compactRaw, { pinDigest: true });
  const compact = JSON.parse(compactRaw);
  const receiptResult = parseAndValidateFrontierReceipt(
    receiptRaw,
    compact,
    compactResult.digest,
    { pinDigest: true },
  );
  return Object.freeze({ compact: compactResult, receipt: receiptResult });
}

function runCli() {
  if (process.argv.length !== 4) {
    console.error(
      "usage: node scripts/validate-frontier-evaluation-receipt-profile.mjs PROFILE_JSON DOGFOOD_RECEIPT_JSON",
    );
    process.exitCode = 2;
    return;
  }
  try {
    const result = validateCanonicalFrontierBundle(
      readFileSync(resolve(process.argv[2]), "utf8"),
      readFileSync(resolve(process.argv[3]), "utf8"),
    );
    console.log(
      `frontier receipt profile: SOURCE-SHAPE PASS; authority, adoption, subject binding for public receipts, material semantics, privacy classification, and enforcement NOT VERIFIED (${result.compact.rightCount} draft rights, ${result.compact.actorCount} actor scopes, ${result.compact.objectionCount} objections; profile sha256 ${result.compact.digest}; dogfood claimed ${result.receipt.claimedResult} sha256 ${result.receipt.digest})`,
    );
  } catch (error) {
    console.error(`frontier receipt profile: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
