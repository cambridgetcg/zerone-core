import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import {
  parseAndValidateConstructiveIntelligenceTree,
  validateConstructiveIntelligenceTreeForUse,
} from "../../dashboard/scripts/validate-constructive-intelligence-tree.mjs";

export const RECEIPT_BUNDLE_SCHEMA =
  "zerone.constructive-intelligence-receipt-shadow/v0";
export const ZERO_VALUE_POLICY_SCHEMA =
  "zerone.constructive-intelligence-zero-value-policy/v0";
export const RECEIPT_DECISION_SCHEMA =
  "zerone.constructive-intelligence-receipt-shadow-decision/v0";
export const RECEIPT_BUNDLE_MAX_BYTES = 262_144;
export const RECEIPT_BUNDLE_MAX_DEPTH = 64;

export const RELEASE_BOUNDARY_KEYS = Object.freeze([
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
]);

const BUNDLE_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "releaseBoundary",
  "checkedForUseAt",
  "treeBinding",
  "policyRevision",
  "receipts",
];
const TREE_BINDING_KEYS = [
  "treeSchema",
  "policyVersion",
  "treeNormativeDigest",
  "treeDocumentDigest",
  "questNodeId",
  "questNodeNormativeDigest",
  "scopeHash",
  "acceptancePolicyDigest",
  "standardsSnapshotDigest",
];
const POLICY_KEYS = [
  "schema",
  "tree_normative_digest",
  "quest_node_id",
  "quest_node_normative_digest",
  "scope_hash",
  "acceptance_policy_digest",
  "standards",
  "canonical_subject_roots",
  "threat_model_digest",
  "falsifiers_digest",
  "permitted_target_and_test_environment_digest",
  "escalation_policy_digest",
  "conflict_cluster_policy_digest",
  "quality_rubric_digest",
  "method_or_adapter_digest",
  "roles",
  "value_boundary",
];
const STANDARD_PIN_KEYS = ["canonical_id", "revision", "edition_digest"];
const ROLE_KEYS = [
  "claimant_control_root",
  "sponsor_control_root",
  "technical_evaluator_control_roots",
  "payout_authorizer_control_root",
];
const VALUE_BOUNDARY_KEYS = [
  "denomination",
  "total_cap",
  "verified_cost_budget",
  "outcome_pool",
  "reviewer_budget",
  "administration_and_fee_budget",
  "escrow_receipt",
  "schedule_instantiated",
  "expiry",
  "refund_path",
];
export const RECEIPT_KEYS = Object.freeze([
  "evidence_id",
  "deliverable_key",
  "immutable_bounty_and_policy_revision_digest",
  "artifact_digest",
  "canonical_subject_roots",
  "prior_deliverable_and_overlap_claim",
  "standards_reference_and_revision",
  "evidence_level_and_scope",
  "method_or_adapter_digest",
  "source_system",
  "source_record_or_event_id",
  "source_revision",
  "payee_and_role",
  "verifier_control_cluster",
  "organization_or_control_root",
  "implementation_or_toolchain_root",
  "execution_environment_digest",
  "conflict_disclosures",
  "authorization_and_safety_decision",
  "result",
  "created_at",
  "supersedes",
]);
const PRIOR_CLAIM_KEYS = [
  "prior_deliverable_key",
  "overlap_claim_digest",
  "independently_reviewed_delta",
  "edge_type",
];
const EVIDENCE_SCOPE_KEYS = [
  "level",
  "quest_node_id",
  "quest_node_normative_digest",
  "scope_hash",
  "acceptance_policy_digest",
  "coverage_target_id",
  "case_ids",
  "checker_or_corpus_digest",
  "adoption_receipt_type",
  "adopter_control_root",
  "evidence_payload_digest",
];
const PAYEE_KEYS = ["payee_id", "role", "control_root"];
const CONFLICT_KEYS = [
  "complete",
  "related_control_roots",
  "outcome_contingent_compensation",
  "claimant_authored_or_controlled_subject",
  "claimant_reviewed_subject",
  "claimant_knowingly_preserved_subject",
  "deliberately_planted_or_retained_defect",
  "claimant_controlled_adoption",
  "self_created_break_fix_loop",
  "concealed_causal_involvement",
];
const AUTHORIZATION_KEYS = [
  "authorized_target",
  "safety_gate_passed",
  "disclosure_lane",
  "unknown_security_impact",
  "prepublication_triage_completed",
  "private_escalation_applied",
  "public_exploit_plaintext_present",
  "confidential_evidence_published",
  "vendor_veto_required",
  "asserts_protocol_security",
  "performs_network_requests",
];
const RESULT_KEYS = ["result_digest", "quality_micros", "method_completed"];

const EVIDENCE_LEVELS = new Set(["E0", "E1", "E2", "E3", "E4", "E5", "E6"]);
const SUPPORTED_RECEIPT_LEVELS = new Set(["E3", "E5"]);
const DISCLOSURE_LANES = new Set([
  "open-construction",
  "private-coordinated-repair",
  "controlled-operations",
]);
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const ISO_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const UTC_TIMESTAMP_PATTERN =
  /^\d{4}-\d{2}-\d{2}T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d{1,9})?Z$/;

export class ConstructiveReceiptValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "ConstructiveReceiptValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new ConstructiveReceiptValidationError(path, message);
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

function exactBoolean(value, expected, path) {
  if (value !== expected) fail(path, `must be ${expected}`);
}

function boundedString(value, path, maxBytes = 1024) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function digest(value, path) {
  const parsed = boundedString(value, path, 64);
  if (!SHA256_PATTERN.test(parsed)) {
    fail(path, "must be a lowercase SHA-256 hex digest");
  }
  return parsed;
}

function boundedInteger(value, path, minimum, maximum) {
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(path, `must be an integer from ${minimum} through ${maximum}`);
  }
  return value;
}

function nullableDigest(value, path) {
  if (value === null) return null;
  return digest(value, path);
}

function sortedUniqueStrings(
  value,
  path,
  { minimum = 0, maximum = 64, maxBytes = 256 } = {},
) {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum) {
    fail(path, `must contain ${minimum} through ${maximum} strings`);
  }
  const parsed = value.map((item, index) =>
    boundedString(item, `${path}[${index}]`, maxBytes),
  );
  if (new Set(parsed).size !== parsed.length) {
    fail(path, "must not contain duplicates");
  }
  if (parsed.some((item, index) => index > 0 && parsed[index - 1] > item)) {
    fail(path, "must be sorted");
  }
  return parsed;
}

function parseIsoDate(value, path) {
  const parsed = boundedString(value, path, 10);
  if (!ISO_DATE_PATTERN.test(parsed)) fail(path, "must be an ISO date");
  const timestamp = Date.parse(`${parsed}T00:00:00Z`);
  if (
    !Number.isFinite(timestamp) ||
    new Date(timestamp).toISOString().slice(0, 10) !== parsed
  ) {
    fail(path, "must be a real ISO date");
  }
  return parsed;
}

function parseUtcTimestamp(value, path) {
  const parsed = boundedString(value, path, 40);
  if (!UTC_TIMESTAMP_PATTERN.test(parsed)) {
    fail(path, "must be an RFC 3339 UTC timestamp");
  }
  const calendarDate = parsed.slice(0, 10);
  const calendarTimestamp = Date.parse(`${calendarDate}T00:00:00Z`);
  if (
    !Number.isFinite(calendarTimestamp) ||
    new Date(calendarTimestamp).toISOString().slice(0, 10) !== calendarDate
  ) {
    fail(path, "must contain a real UTC calendar date");
  }
  if (!Number.isFinite(Date.parse(parsed))) {
    fail(path, "must be a real RFC 3339 UTC timestamp");
  }
  return parsed;
}

function utcTimestampNanoseconds(value) {
  const fractionalSeparator = value.indexOf(".");
  const wholeSeconds =
    fractionalSeparator === -1
      ? value.slice(0, -1)
      : value.slice(0, fractionalSeparator);
  const fraction =
    fractionalSeparator === -1
      ? ""
      : value.slice(fractionalSeparator + 1, -1);
  const epochSeconds = BigInt(
    Math.floor(Date.parse(`${wholeSeconds}Z`) / 1000),
  );
  const nanoseconds = BigInt(fraction.padEnd(9, "0"));
  return epochSeconds * 1_000_000_000n + nanoseconds;
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
  const encoded = JSON.stringify(value);
  if (encoded === undefined) {
    throw new TypeError("canonical JSON cannot encode undefined");
  }
  return encoded;
}

export function sha256Hex(value) {
  return createHash("sha256").update(value).digest("hex");
}

export function digestCanonical(value) {
  return sha256Hex(canonicalJson(value));
}

function normativeStandardProjection(standard) {
  const {
    authorityStatus: _authorityStatus,
    statusCheckedAt: _statusCheckedAt,
    reviewAfter: _reviewAfter,
    ...normative
  } = standard;
  return normative;
}

function normativeNodeProjection(node) {
  return {
    ...node,
    standards: node.standards.map(normativeStandardProjection),
  };
}

export function deriveQuestNodeNormativeDigest(node) {
  return digestCanonical(normativeNodeProjection(node));
}

export function deriveStandardPins(node) {
  return node.standards.map((standard) => ({
    canonical_id: standard.canonicalId,
    revision: standard.revision,
    edition_digest: digestCanonical({
      canonicalId: standard.canonicalId,
      revision: standard.revision,
      specification: standard.specification,
    }),
  }));
}

export function deriveStandardPinsDigest(node) {
  return digestCanonical(
    node.standards.map((standard) => ({
      canonicalId: standard.canonicalId,
      revision: standard.revision,
      specification: standard.specification,
    })),
  );
}

export function derivePolicyRevisionDigest(policyRevision) {
  return digestCanonical(policyRevision);
}

export function deriveDeliverableKey({
  standards_reference_and_revision,
  scope_hash,
  acceptance_policy_digest,
  canonical_subject_roots,
}) {
  return digestCanonical([
    standards_reference_and_revision,
    scope_hash,
    acceptance_policy_digest,
    canonical_subject_roots,
  ]);
}

export function deriveConsumptionKey(receipt) {
  return digestCanonical([
    receipt.source_system,
    receipt.source_record_or_event_id,
    receipt.source_revision,
  ]);
}

export function deriveEvidenceId(receipt) {
  const { evidence_id: _evidenceId, ...unsignedReceipt } = receipt;
  return digestCanonical(unsignedReceipt);
}

export function deriveCanonicalEconomics(tree, level) {
  if (!EVIDENCE_LEVELS.has(level)) {
    fail("$.evidenceLevel", "has an unknown evidence level");
  }
  const milestone = tree.policy.milestones.find((item) => item.level === level);
  if (milestone === undefined) {
    fail("$.tree.policy.milestones", `does not define ${level}`);
  }
  return {
    milestone: {
      level: milestone.level,
      name: milestone.name,
      rewardBps: milestone.rewardBps,
      treatment: milestone.treatment,
    },
    challengeReserveBps: tree.policy.challengeReserveBps,
    metadataOnly: true,
  };
}

function rejectExcessiveJsonNesting(raw) {
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
      if (depth > RECEIPT_BUNDLE_MAX_DEPTH) {
        fail(
          "$",
          `JSON nesting exceeds the v0 limit of ${RECEIPT_BUNDLE_MAX_DEPTH}`,
        );
      }
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
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
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset])) {
      offset += 1;
    }
  };

  scanValue("$");
}

export function parseConstructiveReceiptBundle(raw) {
  if (typeof raw !== "string") fail("$", "raw document must be a string");
  if (Buffer.byteLength(raw, "utf8") > RECEIPT_BUNDLE_MAX_BYTES) {
    fail(
      "$",
      `raw document exceeds ${RECEIPT_BUNDLE_MAX_BYTES} UTF-8 bytes`,
    );
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

function validateReleaseBoundary(value, path) {
  const boundary = record(value, path);
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, path);
  for (const key of RELEASE_BOUNDARY_KEYS) {
    exactBoolean(boundary[key], false, `${path}.${key}`);
  }
  return boundary;
}

function validateStandardPins(value, expected, path) {
  if (!Array.isArray(value) || value.length !== expected.length) {
    fail(path, `must contain exactly ${expected.length} standard pins`);
  }
  return value.map((rawPin, index) => {
    const pinPath = `${path}[${index}]`;
    const pin = record(rawPin, pinPath);
    exactKeys(pin, STANDARD_PIN_KEYS, pinPath);
    boundedString(pin.canonical_id, `${pinPath}.canonical_id`, 128);
    boundedString(pin.revision, `${pinPath}.revision`, 64);
    digest(pin.edition_digest, `${pinPath}.edition_digest`);
    if (canonicalJson(pin) !== canonicalJson(expected[index])) {
      fail(pinPath, "must match the tree-derived canonical standard pin");
    }
    return pin;
  });
}

function validateValueBoundary(value, path) {
  const boundary = record(value, path);
  exactKeys(boundary, VALUE_BOUNDARY_KEYS, path);
  if (boundary.denomination !== null) fail(`${path}.denomination`, "must be null");
  for (const field of [
    "total_cap",
    "verified_cost_budget",
    "outcome_pool",
    "reviewer_budget",
    "administration_and_fee_budget",
  ]) {
    if (boundary[field] !== 0) fail(`${path}.${field}`, "must be exactly zero");
  }
  if (boundary.escrow_receipt !== null) {
    fail(`${path}.escrow_receipt`, "must be null");
  }
  exactBoolean(
    boundary.schedule_instantiated,
    false,
    `${path}.schedule_instantiated`,
  );
  if (boundary.expiry !== null) fail(`${path}.expiry`, "must be null");
  if (boundary.refund_path !== null) fail(`${path}.refund_path`, "must be null");
  return boundary;
}

function validatePolicyRevision(value, context, path) {
  const policy = record(value, path);
  exactKeys(policy, POLICY_KEYS, path);
  if (policy.schema !== ZERO_VALUE_POLICY_SCHEMA) {
    fail(`${path}.schema`, `must equal ${ZERO_VALUE_POLICY_SCHEMA}`);
  }

  const exactDigestFields = [
    ["tree_normative_digest", context.treeResult.normativeDigest],
    ["quest_node_normative_digest", context.questNodeDigest],
    ["scope_hash", context.questNode.acceptance.scopeHash],
    ["acceptance_policy_digest", context.acceptancePolicyDigest],
  ];
  for (const [field, expected] of exactDigestFields) {
    digest(policy[field], `${path}.${field}`);
    if (policy[field] !== expected) {
      fail(`${path}.${field}`, "does not match the active tree binding");
    }
  }
  if (policy.quest_node_id !== context.questNode.id) {
    fail(`${path}.quest_node_id`, "does not match the selected quest");
  }
  validateStandardPins(policy.standards, context.standardPins, `${path}.standards`);
  const subjectRoots = sortedUniqueStrings(
    policy.canonical_subject_roots,
    `${path}.canonical_subject_roots`,
    { minimum: 1, maximum: 16, maxBytes: 256 },
  );
  for (const field of [
    "threat_model_digest",
    "falsifiers_digest",
    "permitted_target_and_test_environment_digest",
    "escalation_policy_digest",
    "conflict_cluster_policy_digest",
    "quality_rubric_digest",
    "method_or_adapter_digest",
  ]) {
    digest(policy[field], `${path}.${field}`);
  }

  const roles = record(policy.roles, `${path}.roles`);
  exactKeys(roles, ROLE_KEYS, `${path}.roles`);
  const claimant = boundedString(
    roles.claimant_control_root,
    `${path}.roles.claimant_control_root`,
    256,
  );
  const sponsor = boundedString(
    roles.sponsor_control_root,
    `${path}.roles.sponsor_control_root`,
    256,
  );
  const evaluators = sortedUniqueStrings(
    roles.technical_evaluator_control_roots,
    `${path}.roles.technical_evaluator_control_roots`,
    { minimum: 3, maximum: 16, maxBytes: 256 },
  );
  const authorizer = boundedString(
    roles.payout_authorizer_control_root,
    `${path}.roles.payout_authorizer_control_root`,
    256,
  );
  const allRoleRoots = [claimant, sponsor, ...evaluators, authorizer];
  if (new Set(allRoleRoots).size !== allRoleRoots.length) {
    fail(`${path}.roles`, "all claimant, sponsor, evaluator, and authorizer roots must be distinct");
  }
  validateValueBoundary(policy.value_boundary, `${path}.value_boundary`);

  return {
    policy,
    subjectRoots,
    claimant,
    sponsor,
    evaluators,
    authorizer,
  };
}

function validatePriorClaim(value, context, path) {
  const claim = record(value, path);
  exactKeys(claim, PRIOR_CLAIM_KEYS, path);
  const prior = nullableDigest(claim.prior_deliverable_key, `${path}.prior_deliverable_key`);
  const overlap = nullableDigest(claim.overlap_claim_digest, `${path}.overlap_claim_digest`);
  if (typeof claim.independently_reviewed_delta !== "boolean") {
    fail(`${path}.independently_reviewed_delta`, "must be a boolean");
  }
  const edgeType =
    claim.edge_type === null
      ? null
      : boundedString(claim.edge_type, `${path}.edge_type`, 32);

  if (prior === null || overlap === null) {
    if (
      prior !== null ||
      overlap !== null ||
      claim.independently_reviewed_delta !== false ||
      edgeType !== null
    ) {
      fail(path, "an absent prior claim must use null/null/false/null");
    }
    return;
  }
  if (prior === context.deliverableKey) {
    fail(`${path}.prior_deliverable_key`, "must identify a different deliverable");
  }
  if (!claim.independently_reviewed_delta) {
    fail(`${path}.independently_reviewed_delta`, "must be true for a derivative");
  }
  if (!context.artifactEdgeTypes.has(edgeType)) {
    fail(`${path}.edge_type`, "must be a canonical artifact edge type");
  }
}

function validateEvidenceScope(value, context, path) {
  const scope = record(value, path);
  exactKeys(scope, EVIDENCE_SCOPE_KEYS, path);
  const level = boundedString(scope.level, `${path}.level`, 2);
  if (!EVIDENCE_LEVELS.has(level)) {
    fail(`${path}.level`, "has an unknown evidence level");
  }
  if (!SUPPORTED_RECEIPT_LEVELS.has(level)) {
    fail(
      `${path}.level`,
      "is tree metadata only; receipt schema v0 supports only E3 and E5 claims",
    );
  }
  const exactFields = [
    ["quest_node_id", context.questNode.id],
    ["quest_node_normative_digest", context.questNodeDigest],
    ["scope_hash", context.questNode.acceptance.scopeHash],
    ["acceptance_policy_digest", context.acceptancePolicyDigest],
    ["coverage_target_id", context.coverageTarget.id],
  ];
  for (const [field, expected] of exactFields) {
    if (scope[field] !== expected) {
      fail(`${path}.${field}`, "does not match the selected quest target");
    }
  }
  sortedUniqueStrings(scope.case_ids, `${path}.case_ids`, {
    minimum: 1,
    maximum: 256,
    maxBytes: 256,
  });
  digest(scope.checker_or_corpus_digest, `${path}.checker_or_corpus_digest`);
  digest(scope.evidence_payload_digest, `${path}.evidence_payload_digest`);

  const adoptionType =
    scope.adoption_receipt_type === null
      ? null
      : boundedString(
          scope.adoption_receipt_type,
          `${path}.adoption_receipt_type`,
          64,
        );
  const adopter =
    scope.adopter_control_root === null
      ? null
      : boundedString(
          scope.adopter_control_root,
          `${path}.adopter_control_root`,
          256,
        );
  if (level === "E5") {
    if (
      adoptionType === null ||
      !context.allowedAdoptionTypes.has(adoptionType)
    ) {
      fail(
        `${path}.adoption_receipt_type`,
        "must be an adoption type allowed by the quest",
      );
    }
    if (adopter === null) {
      fail(
        `${path}.adopter_control_root`,
        "must name an adopter control root for E5",
      );
    }
  } else if (adoptionType !== null || adopter !== null) {
    fail(
      path,
      "non-E5 evidence must leave adoption receipt type and adopter root null",
    );
  }
  return { scope, level };
}

function validateConflictDisclosures(value, verifierRoot, path) {
  const conflicts = record(value, path);
  exactKeys(conflicts, CONFLICT_KEYS, path);
  exactBoolean(conflicts.complete, true, `${path}.complete`);
  const relatedRoots = sortedUniqueStrings(
    conflicts.related_control_roots,
    `${path}.related_control_roots`,
    { minimum: 1, maximum: 32, maxBytes: 256 },
  );
  if (!relatedRoots.includes(verifierRoot)) {
    fail(
      `${path}.related_control_roots`,
      "must include the verifier control cluster",
    );
  }
  exactBoolean(
    conflicts.outcome_contingent_compensation,
    false,
    `${path}.outcome_contingent_compensation`,
  );
  for (const diagnostic of [
    "claimant_authored_or_controlled_subject",
    "claimant_reviewed_subject",
    "claimant_knowingly_preserved_subject",
  ]) {
    if (typeof conflicts[diagnostic] !== "boolean") {
      fail(`${path}.${diagnostic}`, "must be a boolean");
    }
  }
  for (const ineligible of [
    "deliberately_planted_or_retained_defect",
    "claimant_controlled_adoption",
    "self_created_break_fix_loop",
    "concealed_causal_involvement",
  ]) {
    exactBoolean(conflicts[ineligible], false, `${path}.${ineligible}`);
  }
  return { conflicts, relatedRoots };
}

function validateAuthorization(value, context, path) {
  const decision = record(value, path);
  exactKeys(decision, AUTHORIZATION_KEYS, path);
  exactBoolean(decision.authorized_target, true, `${path}.authorized_target`);
  exactBoolean(decision.safety_gate_passed, true, `${path}.safety_gate_passed`);
  const lane = boundedString(
    decision.disclosure_lane,
    `${path}.disclosure_lane`,
    64,
  );
  if (!DISCLOSURE_LANES.has(lane)) {
    fail(`${path}.disclosure_lane`, "has an unknown disclosure lane");
  }
  if (typeof decision.unknown_security_impact !== "boolean") {
    fail(`${path}.unknown_security_impact`, "must be a boolean");
  }
  exactBoolean(
    decision.prepublication_triage_completed,
    context.questNode.acceptance.prepublicationTriageRequired,
    `${path}.prepublication_triage_completed`,
  );
  if (decision.unknown_security_impact) {
    if (
      lane !== "private-coordinated-repair" ||
      decision.private_escalation_applied !== true
    ) {
      fail(
        path,
        "unknown security impact requires private coordinated repair and escalation",
      );
    }
  } else {
    exactBoolean(
      decision.private_escalation_applied,
      false,
      `${path}.private_escalation_applied`,
    );
  }
  for (const forbidden of [
    "public_exploit_plaintext_present",
    "confidential_evidence_published",
    "vendor_veto_required",
    "asserts_protocol_security",
    "performs_network_requests",
  ]) {
    exactBoolean(decision[forbidden], false, `${path}.${forbidden}`);
  }
  return decision;
}

function validateReceipt(value, index, context) {
  const path = `$.receipts[${index}]`;
  const receipt = record(value, path);
  exactKeys(receipt, RECEIPT_KEYS, path);
  digest(receipt.evidence_id, `${path}.evidence_id`);
  digest(receipt.deliverable_key, `${path}.deliverable_key`);
  if (receipt.deliverable_key !== context.deliverableKey) {
    fail(`${path}.deliverable_key`, "does not match the canonical deliverable key");
  }
  digest(
    receipt.immutable_bounty_and_policy_revision_digest,
    `${path}.immutable_bounty_and_policy_revision_digest`,
  );
  if (
    receipt.immutable_bounty_and_policy_revision_digest !==
    context.policyRevisionDigest
  ) {
    fail(
      `${path}.immutable_bounty_and_policy_revision_digest`,
      "does not match the policy revision",
    );
  }
  digest(receipt.artifact_digest, `${path}.artifact_digest`);
  if (
    canonicalJson(receipt.canonical_subject_roots) !==
    canonicalJson(context.policyContext.subjectRoots)
  ) {
    fail(
      `${path}.canonical_subject_roots`,
      "must match the policy canonical subject roots",
    );
  }
  sortedUniqueStrings(
    receipt.canonical_subject_roots,
    `${path}.canonical_subject_roots`,
    { minimum: 1, maximum: 16, maxBytes: 256 },
  );
  validatePriorClaim(
    receipt.prior_deliverable_and_overlap_claim,
    context,
    `${path}.prior_deliverable_and_overlap_claim`,
  );
  validateStandardPins(
    receipt.standards_reference_and_revision,
    context.standardPins,
    `${path}.standards_reference_and_revision`,
  );
  const evidence = validateEvidenceScope(
    receipt.evidence_level_and_scope,
    context,
    `${path}.evidence_level_and_scope`,
  );
  digest(receipt.method_or_adapter_digest, `${path}.method_or_adapter_digest`);
  if (
    receipt.method_or_adapter_digest !==
    context.policyContext.policy.method_or_adapter_digest
  ) {
    fail(
      `${path}.method_or_adapter_digest`,
      "does not match the policy adapter digest",
    );
  }
  boundedString(receipt.source_system, `${path}.source_system`, 128);
  boundedString(
    receipt.source_record_or_event_id,
    `${path}.source_record_or_event_id`,
    256,
  );
  boundedString(receipt.source_revision, `${path}.source_revision`, 128);

  const payee = record(receipt.payee_and_role, `${path}.payee_and_role`);
  exactKeys(payee, PAYEE_KEYS, `${path}.payee_and_role`);
  boundedString(payee.payee_id, `${path}.payee_and_role.payee_id`, 256);
  if (payee.role !== "technical-evaluator") {
    fail(`${path}.payee_and_role.role`, "must be technical-evaluator");
  }
  const payeeRoot = boundedString(
    payee.control_root,
    `${path}.payee_and_role.control_root`,
    256,
  );
  const verifierRoot = boundedString(
    receipt.verifier_control_cluster,
    `${path}.verifier_control_cluster`,
    256,
  );
  if (payeeRoot !== verifierRoot) {
    fail(
      `${path}.payee_and_role.control_root`,
      "must equal the verifier control cluster",
    );
  }
  if (!context.policyContext.evaluators.includes(verifierRoot)) {
    fail(
      `${path}.verifier_control_cluster`,
      "must be a declared technical evaluator root",
    );
  }
  const organizationRoot = boundedString(
    receipt.organization_or_control_root,
    `${path}.organization_or_control_root`,
    256,
  );
  const implementationRoot = boundedString(
    receipt.implementation_or_toolchain_root,
    `${path}.implementation_or_toolchain_root`,
    256,
  );
  const forbiddenQuorumRoots = new Set([
    context.policyContext.claimant,
    context.policyContext.sponsor,
  ]);
  if (
    forbiddenQuorumRoots.has(verifierRoot) ||
    forbiddenQuorumRoots.has(organizationRoot) ||
    forbiddenQuorumRoots.has(implementationRoot)
  ) {
    fail(path, "claimant and sponsor roots cannot contribute to technical quorum");
  }
  digest(
    receipt.execution_environment_digest,
    `${path}.execution_environment_digest`,
  );
  const conflict = validateConflictDisclosures(
    receipt.conflict_disclosures,
    verifierRoot,
    `${path}.conflict_disclosures`,
  );
  if (evidence.level === "E5") {
    const adopterRoot = evidence.scope.adopter_control_root;
    const adoptionConflictRoots = new Set([
      context.policyContext.claimant,
      context.policyContext.sponsor,
      context.policyContext.authorizer,
      ...context.policyContext.evaluators,
      organizationRoot,
      implementationRoot,
      ...conflict.relatedRoots,
    ]);
    if (adoptionConflictRoots.has(adopterRoot)) {
      fail(
        `${path}.evidence_level_and_scope.adopter_control_root`,
        "must be independent of every policy role and disclosed evaluator control root for E5",
      );
    }
  }
  validateAuthorization(
    receipt.authorization_and_safety_decision,
    context,
    `${path}.authorization_and_safety_decision`,
  );
  const result = record(receipt.result, `${path}.result`);
  exactKeys(result, RESULT_KEYS, `${path}.result`);
  digest(result.result_digest, `${path}.result.result_digest`);
  const qualityMicros = boundedInteger(
    result.quality_micros,
    `${path}.result.quality_micros`,
    0,
    1_000_000,
  );
  exactBoolean(result.method_completed, true, `${path}.result.method_completed`);
  const createdAt = parseUtcTimestamp(receipt.created_at, `${path}.created_at`);
  const createdDate = createdAt.slice(0, 10);
  if (createdDate < context.tree.snapshotDate) {
    fail(`${path}.created_at`, "cannot predate the tree snapshot");
  }
  if (createdDate > context.checkedForUseAt) {
    fail(`${path}.created_at`, "cannot be after checkedForUseAt");
  }
  nullableDigest(receipt.supersedes, `${path}.supersedes`);
  const expectedEvidenceId = deriveEvidenceId(receipt);
  if (receipt.evidence_id !== expectedEvidenceId) {
    fail(`${path}.evidence_id`, "does not match the canonical receipt digest");
  }

  return {
    receipt,
    index,
    evidenceId: receipt.evidence_id,
    consumptionKey: deriveConsumptionKey(receipt),
    artifactDigest: receipt.artifact_digest,
    level: evidence.level,
    adopterRoot: evidence.scope.adopter_control_root,
    checkerDigest: evidence.scope.checker_or_corpus_digest,
    caseIds: evidence.scope.case_ids,
    verifierRoot,
    relatedRoots: conflict.relatedRoots,
    organizationRoot,
    implementationRoot,
    environmentDigest: receipt.execution_environment_digest,
    qualityMicros,
    createdAt,
  };
}

function validateSupersession(receipts) {
  const indexByEvidenceId = new Map(
    receipts.map((receipt, index) => [receipt.evidenceId, index]),
  );
  const supersededEvidenceIds = new Set();
  for (const receipt of receipts) {
    const supersedes = receipt.receipt.supersedes;
    if (supersedes === null) continue;
    const supersededIndex = indexByEvidenceId.get(supersedes);
    if (supersededIndex === undefined) {
      fail(
        `$.receipts[${receipt.index}].supersedes`,
        "must identify an evidence ID in the same bundle",
      );
    }
    if (supersededIndex >= receipt.index) {
      fail(
        `$.receipts[${receipt.index}].supersedes`,
        "must identify an earlier receipt",
      );
    }
    if (
      utcTimestampNanoseconds(receipt.createdAt) <=
      utcTimestampNanoseconds(receipts[supersededIndex].createdAt)
    ) {
      fail(
        `$.receipts[${receipt.index}].created_at`,
        "must be strictly later than the receipt it supersedes",
      );
    }
    if (supersededEvidenceIds.has(supersedes)) {
      fail(
        `$.receipts[${receipt.index}].supersedes`,
        "cannot create two active replacements for one superseded receipt",
      );
    }
    supersededEvidenceIds.add(supersedes);
  }
  return receipts.filter(
    (receipt) => !supersededEvidenceIds.has(receipt.evidenceId),
  );
}

function validateBundleWideAdoptionIndependence(receipts, policyContext) {
  const conflictRoots = new Set([
    policyContext.claimant,
    policyContext.sponsor,
    policyContext.authorizer,
    ...policyContext.evaluators,
  ]);
  for (const receipt of receipts) {
    conflictRoots.add(receipt.organizationRoot);
    conflictRoots.add(receipt.implementationRoot);
    for (const root of receipt.relatedRoots) conflictRoots.add(root);
  }
  for (const receipt of receipts) {
    if (
      receipt.level === "E5" &&
      conflictRoots.has(receipt.adopterRoot)
    ) {
      fail(
        `$.receipts[${receipt.index}].evidence_level_and_scope.adopter_control_root`,
        "must be independent of every policy role and every active bundle control root for E5",
      );
    }
  }
}

function deriveIndependence(receipts, floors) {
  const parent = receipts.map((_, index) => index);
  const find = (index) => {
    while (parent[index] !== index) {
      parent[index] = parent[parent[index]];
      index = parent[index];
    }
    return index;
  };
  const union = (left, right) => {
    const leftRoot = find(left);
    const rightRoot = find(right);
    if (leftRoot !== rightRoot) parent[rightRoot] = leftRoot;
  };

  for (let left = 0; left < receipts.length; left += 1) {
    const leftRoots = new Set(receipts[left].relatedRoots);
    for (let right = left + 1; right < receipts.length; right += 1) {
      if (
        receipts[left].verifierRoot === receipts[right].verifierRoot ||
        receipts[right].relatedRoots.some((root) => leftRoots.has(root))
      ) {
        union(left, right);
      }
    }
  }

  const qualityByComponent = new Map();
  for (const [index, receipt] of receipts.entries()) {
    const root = find(index);
    qualityByComponent.set(
      root,
      (qualityByComponent.get(root) ?? 0) + receipt.qualityMicros,
    );
  }
  const effectiveQualityMicros = [...qualityByComponent.values()].reduce(
    (total, quality) => total + Math.min(1_000_000, quality),
    0,
  );
  const componentCount = qualityByComponent.size;
  const effectiveClusters = Math.floor(effectiveQualityMicros / 1_000_000);
  const componentByReceipt = receipts.map((_, index) => find(index));
  const fullySupportedValueCount = (selectValues) => {
    const qualityByValueAndComponent = new Map();
    for (const [index, receipt] of receipts.entries()) {
      for (const value of selectValues(receipt)) {
        let qualityByComponentForValue = qualityByValueAndComponent.get(value);
        if (qualityByComponentForValue === undefined) {
          qualityByComponentForValue = new Map();
          qualityByValueAndComponent.set(value, qualityByComponentForValue);
        }
        const component = componentByReceipt[index];
        qualityByComponentForValue.set(
          component,
          (qualityByComponentForValue.get(component) ?? 0) +
            receipt.qualityMicros,
        );
      }
    }
    let count = 0;
    for (const qualityByComponentForValue of qualityByValueAndComponent.values()) {
      const effectiveSupport = [...qualityByComponentForValue.values()].reduce(
        (total, quality) => total + Math.min(1_000_000, quality),
        0,
      );
      if (effectiveSupport >= 1_000_000) count += 1;
    }
    return count;
  };
  const organizationRootCount = fullySupportedValueCount((receipt) => [
    receipt.organizationRoot,
  ]);
  const implementationRootCount = fullySupportedValueCount((receipt) => [
    receipt.implementationRoot,
  ]);
  const executionEnvironmentCount = fullySupportedValueCount((receipt) => [
    receipt.environmentDigest,
  ]);
  const uniqueCaseCount = fullySupportedValueCount(
    (receipt) => receipt.caseIds,
  );

  if (effectiveQualityMicros < floors.effectiveClusters * 1_000_000) {
    fail(
      "$.receipts",
      `effective independent quality is below ${floors.effectiveClusters} full clusters`,
    );
  }
  if (organizationRootCount < floors.organizationRoots) {
    fail(
      "$.receipts",
      `fully supported organization/control roots are below the floor of ${floors.organizationRoots}`,
    );
  }
  if (implementationRootCount < floors.implementationRoots) {
    fail(
      "$.receipts",
      `fully supported implementation/toolchain roots are below the floor of ${floors.implementationRoots}`,
    );
  }
  if (executionEnvironmentCount < floors.executionEnvironments) {
    fail(
      "$.receipts",
      `fully supported execution environments are below the floor of ${floors.executionEnvironments}`,
    );
  }
  if (uniqueCaseCount < floors.cases) {
    fail(
      "$.receipts",
      `fully supported unique cases are below the floor of ${floors.cases}`,
    );
  }

  return {
    requiredEffectiveClusters: floors.effectiveClusters,
    componentCount,
    effectiveClusters,
    effectiveQualityMicros,
    organizationRootCount,
    implementationRootCount,
    executionEnvironmentCount,
    uniqueCaseCount,
  };
}

function buildContext(treeRaw, bundle) {
  const treeResult = parseAndValidateConstructiveIntelligenceTree(treeRaw);
  const tree = JSON.parse(treeRaw);
  const checkedForUseAt = parseIsoDate(
    bundle.checkedForUseAt,
    "$.checkedForUseAt",
  );
  validateConstructiveIntelligenceTreeForUse(tree, checkedForUseAt);

  const binding = record(bundle.treeBinding, "$.treeBinding");
  exactKeys(binding, TREE_BINDING_KEYS, "$.treeBinding");
  const questNode = tree.nodes.find((node) => node.id === binding.questNodeId);
  if (questNode === undefined) {
    fail("$.treeBinding.questNodeId", "does not identify a tree node");
  }
  if (
    questNode.stage !== "quest" ||
    questNode.rewardEligibility !== "sponsor-milestones" ||
    questNode.acceptance === null
  ) {
    fail(
      "$.treeBinding.questNodeId",
      "must select a sponsor-milestone quest with bounded acceptance",
    );
  }

  const questNodeDigest = deriveQuestNodeNormativeDigest(questNode);
  const acceptancePolicyDigest = digestCanonical(questNode.acceptance);
  const standardsSnapshotDigest = digestCanonical(questNode.standards);
  const standardPins = deriveStandardPins(questNode);
  const expectedBinding = {
    treeSchema: tree.schema,
    policyVersion: tree.policyVersion,
    treeNormativeDigest: treeResult.normativeDigest,
    treeDocumentDigest: sha256Hex(treeRaw),
    questNodeId: questNode.id,
    questNodeNormativeDigest: questNodeDigest,
    scopeHash: questNode.acceptance.scopeHash,
    acceptancePolicyDigest,
    standardsSnapshotDigest,
  };
  for (const [field, expected] of Object.entries(expectedBinding)) {
    if (binding[field] !== expected) {
      fail(`$.treeBinding.${field}`, "does not match the active tree document");
    }
  }

  return {
    tree,
    treeResult,
    checkedForUseAt,
    binding,
    questNode,
    questNodeDigest,
    acceptancePolicyDigest,
    standardsSnapshotDigest,
    standardPins,
  };
}

export function validateConstructiveReceiptBundle(treeRaw, bundleRaw) {
  if (typeof treeRaw !== "string") {
    fail("$.tree", "raw tree document must be a string");
  }
  const bundle = parseConstructiveReceiptBundle(bundleRaw);
  record(bundle, "$");
  exactKeys(bundle, BUNDLE_KEYS, "$");
  if (bundle.schema !== RECEIPT_BUNDLE_SCHEMA) {
    fail("$.schema", `must equal ${RECEIPT_BUNDLE_SCHEMA}`);
  }
  exactBoolean(bundle.authoritative, false, "$.authoritative");
  exactBoolean(bundle.networkObserved, false, "$.networkObserved");
  exactBoolean(bundle.rewardBearing, false, "$.rewardBearing");
  validateReleaseBoundary(bundle.releaseBoundary, "$.releaseBoundary");

  const baseContext = buildContext(treeRaw, bundle);
  const coverageTargets = baseContext.questNode.acceptance.coverageTargets;
  if (coverageTargets.length !== 1) {
    fail(
      "$.treeBinding.questNodeId",
      "v0 supports exactly one canonical coverage target per bundle",
    );
  }
  const coverageTarget = coverageTargets[0];
  const allowedAdoptionTypes = new Set(
    baseContext.questNode.acceptance.adoptionReceiptTypes,
  );
  const policyContext = validatePolicyRevision(
    bundle.policyRevision,
    baseContext,
    "$.policyRevision",
  );
  const policyRevisionDigest = derivePolicyRevisionDigest(
    bundle.policyRevision,
  );
  const deliverableKey = deriveDeliverableKey({
    standards_reference_and_revision: baseContext.standardPins,
    scope_hash: baseContext.questNode.acceptance.scopeHash,
    acceptance_policy_digest: baseContext.acceptancePolicyDigest,
    canonical_subject_roots: policyContext.subjectRoots,
  });
  const context = {
    ...baseContext,
    coverageTarget,
    allowedAdoptionTypes,
    policyContext,
    policyRevisionDigest,
    deliverableKey,
    artifactEdgeTypes: new Set(baseContext.tree.policy.artifactEdgeTypes),
  };

  if (
    !Array.isArray(bundle.receipts) ||
    bundle.receipts.length < 1 ||
    bundle.receipts.length > 64
  ) {
    fail("$.receipts", "must contain 1 through 64 receipts");
  }
  const receipts = bundle.receipts.map((receipt, index) =>
    validateReceipt(receipt, index, context),
  );
  const evidenceIds = receipts.map((receipt) => receipt.evidenceId);
  if (new Set(evidenceIds).size !== evidenceIds.length) {
    fail("$.receipts", "must not contain duplicate evidence IDs");
  }
  const consumptionKeys = receipts.map((receipt) => receipt.consumptionKey);
  if (new Set(consumptionKeys).size !== consumptionKeys.length) {
    fail("$.receipts", "reuses a source-system consumption key");
  }
  const activeReceipts = validateSupersession(receipts);
  validateBundleWideAdoptionIndependence(activeReceipts, policyContext);

  const levels = new Set(activeReceipts.map((receipt) => receipt.level));
  if (levels.size !== 1) fail("$.receipts", "must use one evidence level");
  const artifacts = new Set(
    activeReceipts.map((receipt) => receipt.artifactDigest),
  );
  if (artifacts.size !== 1) {
    fail("$.receipts", "must bind one artifact digest");
  }
  const checkerDigests = new Set(
    activeReceipts.map((receipt) => receipt.checkerDigest),
  );
  if (
    coverageTarget.requiresCheckerOrCorpusDigest &&
    checkerDigests.size !== 1
  ) {
    fail("$.receipts", "must bind one checker or corpus digest");
  }

  const globalFloor = baseContext.tree.policy.independence;
  const acceptance = baseContext.questNode.acceptance;
  const floors = {
    effectiveClusters: Math.max(
      globalFloor.minimumEffectiveClusters,
      acceptance.minimumEffectiveClusters,
      coverageTarget.minimumEffectiveClusters,
    ),
    organizationRoots: Math.max(
      globalFloor.minimumOrganizationRoots,
      acceptance.minimumOrganizationRoots,
      coverageTarget.minimumOrganizationRoots,
    ),
    implementationRoots: Math.max(
      globalFloor.minimumImplementationRoots,
      acceptance.minimumImplementationRoots,
      coverageTarget.minimumImplementationRoots,
    ),
    executionEnvironments: Math.max(
      globalFloor.minimumExecutionEnvironments,
      acceptance.minimumExecutionEnvironments,
      coverageTarget.minimumExecutionEnvironments,
    ),
    cases: coverageTarget.minimumCases,
  };
  const independence = deriveIndependence(activeReceipts, floors);
  const evidenceLevel = activeReceipts[0].level;

  return {
    schema: RECEIPT_DECISION_SCHEMA,
    structurallyValid: true,
    authoritative: false,
    networkObserved: false,
    rewardBearing: false,
    qualificationGranted: false,
    evidenceAccepted: false,
    fundsMoved: false,
    integrationAuthorized: false,
    releaseBoundary: Object.fromEntries(
      RELEASE_BOUNDARY_KEYS.map((key) => [key, false]),
    ),
    checkedForUseAt: baseContext.checkedForUseAt,
    treeSchema: baseContext.tree.schema,
    policyVersion: baseContext.tree.policyVersion,
    treeNormativeDigest: baseContext.treeResult.normativeDigest,
    treeDocumentDigest: baseContext.binding.treeDocumentDigest,
    questNodeId: baseContext.questNode.id,
    questNodeNormativeDigest: baseContext.questNodeDigest,
    policyRevisionDigest,
    deliverableKey,
    evidenceLevel,
    coverageTargetId: coverageTarget.id,
    receiptCount: activeReceipts.length,
    submittedReceiptCount: receipts.length,
    supersededReceiptCount: receipts.length - activeReceipts.length,
    consumptionKeys: [...consumptionKeys].sort(),
    independence,
    canonicalEconomics: deriveCanonicalEconomics(
      baseContext.tree,
      evidenceLevel,
    ),
    valueBoundary: {
      denomination: null,
      totalCap: 0,
      verifiedCostBudget: 0,
      outcomePool: 0,
      reviewerBudget: 0,
      administrationAndFeeBudget: 0,
      escrowReceipt: null,
      scheduleInstantiated: false,
      expiry: null,
      refundPath: null,
    },
  };
}

export function runCli(argv = process.argv.slice(2)) {
  if (
    argv.length !== 4 ||
    argv[0] !== "--tree" ||
    argv[2] !== "--bundle"
  ) {
    console.error(
      "usage: node tools/constructive-receipts/validate.mjs --tree PATH --bundle PATH",
    );
    return 2;
  }
  try {
    const treeRaw = readFileSync(argv[1], "utf8");
    const bundleRaw = readFileSync(argv[3], "utf8");
    const decision = validateConstructiveReceiptBundle(treeRaw, bundleRaw);
    console.log(JSON.stringify(decision, null, 2));
    return 0;
  } catch (error) {
    const path =
      typeof error === "object" && error !== null && "path" in error
        ? error.path
        : "$";
    console.error(
      JSON.stringify(
        {
          schema: RECEIPT_DECISION_SCHEMA,
          structurallyValid: false,
          error: { path, message: error.message },
        },
        null,
        2,
      ),
    );
    return 1;
  }
}

if (
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  process.exitCode = runCli();
}
