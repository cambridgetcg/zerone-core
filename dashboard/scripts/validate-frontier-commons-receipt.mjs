import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { parseAndValidateFrontierCommons } from "./validate-frontier-commons.mjs";

export const FRONTIER_COMMONS_RECEIPT_SCHEMA =
  "zerone.frontier-evaluation-receipt/v0";
export const FRONTIER_COMMONS_RECEIPT_MAX_BYTES = 32_768;
export const EXPECTED_FRONTIER_COMMONS_SHA256 =
  "c642e09f46dcaf0a1960f140996969688b936a806c2d33ebf0b6c3efa6a70d2a";
export const EXPECTED_FRONTIER_COMMONS_RECEIPT_SHA256 =
  "3e36185eaba207d1defec4b488f9070683f184fffe7ee5b6f487ac1e15c95b92";

const STATEMENT_V1 = "https://in-toto.io/Statement/v1";
const PREDICATE_TYPE =
  "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/frontier-commons-receipt-v0.md#fc-01-self-receipt-profile";
const RECEIPT_KIND = "ZERONE_SELF_DOGFOOD";
const ASSURANCE = "UNSIGNED_UNVERIFIED_DECLARATION";
const SUBJECT_NAME = "frontier-commons-participation.v0.json";
const DISCLOSURE_CLASS = "PUBLIC_METADATA_ONLY";
const RESULT = "INCONCLUSIVE";
const MAX_JSON_NESTING = 32;
const SCRIPT_PATH = fileURLToPath(import.meta.url);

const STATEMENT_KEYS = ["_type", "subject", "predicateType", "predicate"];
const SUBJECT_KEYS = ["name", "digest"];
const SUBJECT_DIGEST_KEYS = ["sha256"];
const PREDICATE_KEYS = [
  "schema",
  "receiptKind",
  "assurance",
  "issuer",
  "evaluation",
  "privacy",
  "effects",
  "correction",
];
const ISSUER_KEYS = [
  "id",
  "authorityScope",
  "identityDisclosure",
  "controlRootClaim",
  "authorityAssurance",
  "controlRootAssurance",
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
  "limitations",
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
const PRIVACY_KEYS = [
  "containsConfidentialData",
  "containsPersonalData",
  "containsModelWeights",
  "containsTrainingData",
  "containsPrivatePrompts",
  "containsExploitDetails",
  "containsSecrets",
  "containsExportControlledMaterial",
];
const EFFECT_KEYS = [
  "truth",
  "safety",
  "compliance",
  "certification",
  "endorsement",
  "membership",
  "economic",
  "reward",
  "karma",
  "qualification",
  "authority",
  "governance",
  "privacyRights",
  "networkWrite",
];
const CORRECTION_KEYS = [
  "mode",
  "futureRelianceMayBeWithdrawn",
  "historicalPublicCopiesDeletable",
  "exitAffectsUnrelatedAccess",
];
const REQUIRED_REASON_CODES = [
  "NO_BOUND_EVALUATION_MATERIALS",
  "NO_EXTERNAL_REVIEW",
  "NO_OPERATIONAL_ENFORCEMENT",
  "NO_SIGNATORIES",
];

export class FrontierCommonsReceiptValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "FrontierCommonsReceiptValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new FrontierCommonsReceiptValidationError(path, message);
}

function sha256(raw) {
  return createHash("sha256").update(raw).digest("hex");
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
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

function exactString(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${expected}`);
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

function exactStringArray(value, expected, path) {
  if (!Array.isArray(value) || value.length !== expected.length) {
    fail(path, `must contain exactly ${expected.length} reviewed values`);
  }
  for (const [index, expectedValue] of expected.entries()) {
    const actual = boundedString(value[index], `${path}[${index}]`, 1_024);
    if (actual !== expectedValue) {
      fail(`${path}[${index}]`, `must equal ${expectedValue}`);
    }
  }
}

function falseOnly(value, path) {
  if (value !== false) fail(path, "must remain false in FC-0.1");
}

function trueOnly(value, path) {
  if (value !== true) fail(path, "must remain true in FC-0.1");
}

function nullOnly(value, path) {
  if (value !== null) {
    fail(path, "must remain explicitly null for the FC-0.1 self-receipt");
  }
}

function isoDate(value, path) {
  const text = boundedString(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) fail(path, "must be YYYY-MM-DD");
  const date = new Date(`${text}T00:00:00.000Z`);
  if (Number.isNaN(date.valueOf()) || date.toISOString().slice(0, 10) !== text) {
    fail(path, "must be a real UTC calendar date");
  }
  return text;
}

function temporalStatus(asOf, createdOn, reviewAfterOn, expiresOn) {
  if (asOf === undefined) return "NOT_EVALUATED";
  const date = isoDate(asOf, "$asOf");
  if (date < createdOn) return "NOT_YET_CREATED";
  if (date > expiresOn) return "EXPIRED";
  if (date >= reviewAfterOn) return "REVIEW_DUE";
  return "CURRENT";
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
      if (depth > MAX_JSON_NESTING) {
        fail(label, `JSON nesting exceeds the FC-0.1 limit of ${MAX_JSON_NESTING}`);
      }
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
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
        const key = scanString();
        const keyPath = `${path}.${key}`;
        if (keys.has(key)) fail(keyPath, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object delimiter");
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
  scanValue(label);
  whitespace();
  if (offset !== raw.length) fail(label, "contains trailing JSON data");
}

function parseReceiptRaw(raw) {
  if (typeof raw !== "string") fail("$receipt", "must be a JSON string");
  if (Buffer.byteLength(raw, "utf8") > FRONTIER_COMMONS_RECEIPT_MAX_BYTES) {
    fail(
      "$receipt",
      `must be at most ${FRONTIER_COMMONS_RECEIPT_MAX_BYTES} UTF-8 bytes`,
    );
  }
  rejectExcessiveJsonNesting(raw, "$receipt");
  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    fail("$receipt", `must be valid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw, "$receipt");
  return value;
}

function validateParsedReceipt(statement, asOf) {
  const receipt = record(statement, "$receipt");
  exactKeys(receipt, STATEMENT_KEYS, "$receipt");
  exactString(receipt._type, STATEMENT_V1, "$receipt._type");

  if (!Array.isArray(receipt.subject) || receipt.subject.length !== 1) {
    fail("$receipt.subject", "must contain FC-0 as its sole subject");
  }
  const subject = record(receipt.subject[0], "$receipt.subject[0]");
  exactKeys(subject, SUBJECT_KEYS, "$receipt.subject[0]");
  exactString(subject.name, SUBJECT_NAME, "$receipt.subject[0].name");
  const subjectDigest = record(subject.digest, "$receipt.subject[0].digest");
  exactKeys(subjectDigest, SUBJECT_DIGEST_KEYS, "$receipt.subject[0].digest");
  exactString(
    subjectDigest.sha256,
    EXPECTED_FRONTIER_COMMONS_SHA256,
    "$receipt.subject[0].digest.sha256",
  );
  exactString(receipt.predicateType, PREDICATE_TYPE, "$receipt.predicateType");

  const predicate = record(receipt.predicate, "$receipt.predicate");
  exactKeys(predicate, PREDICATE_KEYS, "$receipt.predicate");
  exactString(predicate.schema, FRONTIER_COMMONS_RECEIPT_SCHEMA, "$receipt.predicate.schema");
  exactString(predicate.receiptKind, RECEIPT_KIND, "$receipt.predicate.receiptKind");
  exactString(predicate.assurance, ASSURANCE, "$receipt.predicate.assurance");

  const issuer = record(predicate.issuer, "$receipt.predicate.issuer");
  exactKeys(issuer, ISSUER_KEYS, "$receipt.predicate.issuer");
  exactString(issuer.id, "zerone-fc0-source-draft", "$receipt.predicate.issuer.id");
  exactString(
    issuer.authorityScope,
    "ARTIFACT_ONLY",
    "$receipt.predicate.issuer.authorityScope",
  );
  exactString(
    issuer.identityDisclosure,
    "PROJECT_ROLE",
    "$receipt.predicate.issuer.identityDisclosure",
  );
  exactString(
    issuer.controlRootClaim,
    "zerone-current-operator",
    "$receipt.predicate.issuer.controlRootClaim",
  );
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

  const evaluation = record(predicate.evaluation, "$receipt.predicate.evaluation");
  exactKeys(evaluation, EVALUATION_KEYS, "$receipt.predicate.evaluation");
  const createdOn = isoDate(
    evaluation.createdOn,
    "$receipt.predicate.evaluation.createdOn",
  );
  const evidenceCutoffOn = isoDate(
    evaluation.evidenceCutoffOn,
    "$receipt.predicate.evaluation.evidenceCutoffOn",
  );
  const reviewAfterOn = isoDate(
    evaluation.reviewAfterOn,
    "$receipt.predicate.evaluation.reviewAfterOn",
  );
  const expiresOn = isoDate(
    evaluation.expiresOn,
    "$receipt.predicate.evaluation.expiresOn",
  );
  if (evidenceCutoffOn > createdOn) {
    fail("$receipt.predicate.evaluation.evidenceCutoffOn", "cannot follow creation");
  }
  if (reviewAfterOn <= createdOn) {
    fail("$receipt.predicate.evaluation.reviewAfterOn", "must follow creation");
  }
  if (expiresOn < reviewAfterOn) {
    fail("$receipt.predicate.evaluation.expiresOn", "cannot precede review-after");
  }
  const status = temporalStatus(asOf, createdOn, reviewAfterOn, expiresOn);

  for (const key of EVALUATION_DIGEST_KEYS) {
    nullOnly(evaluation[key], `$receipt.predicate.evaluation.${key}`);
  }
  exactString(
    evaluation.disclosureClass,
    DISCLOSURE_CLASS,
    "$receipt.predicate.evaluation.disclosureClass",
  );
  exactString(evaluation.result, RESULT, "$receipt.predicate.evaluation.result");
  exactStringArray(
    evaluation.reasonCodes,
    REQUIRED_REASON_CODES,
    "$receipt.predicate.evaluation.reasonCodes",
  );
  if (
    !Array.isArray(evaluation.limitations) ||
    evaluation.limitations.length !== 6
  ) {
    fail("$receipt.predicate.evaluation.limitations", "must contain six reviewed limitations");
  }
  for (const [index, limitation] of evaluation.limitations.entries()) {
    boundedString(limitation, `$receipt.predicate.evaluation.limitations[${index}]`, 1_024);
  }
  if (!Array.isArray(evaluation.relations) || evaluation.relations.length !== 0) {
    fail("$receipt.predicate.evaluation.relations", "must remain empty in FC-0.1");
  }

  const privacy = record(predicate.privacy, "$receipt.predicate.privacy");
  exactKeys(privacy, PRIVACY_KEYS, "$receipt.predicate.privacy");
  for (const key of PRIVACY_KEYS) {
    falseOnly(privacy[key], `$receipt.predicate.privacy.${key}`);
  }

  const effects = record(predicate.effects, "$receipt.predicate.effects");
  exactKeys(effects, EFFECT_KEYS, "$receipt.predicate.effects");
  for (const key of EFFECT_KEYS) {
    falseOnly(effects[key], `$receipt.predicate.effects.${key}`);
  }

  const correction = record(predicate.correction, "$receipt.predicate.correction");
  exactKeys(correction, CORRECTION_KEYS, "$receipt.predicate.correction");
  exactString(
    correction.mode,
    "APPEND_ONLY_SUPERSESSION_OR_WITHDRAWAL",
    "$receipt.predicate.correction.mode",
  );
  trueOnly(
    correction.futureRelianceMayBeWithdrawn,
    "$receipt.predicate.correction.futureRelianceMayBeWithdrawn",
  );
  falseOnly(
    correction.historicalPublicCopiesDeletable,
    "$receipt.predicate.correction.historicalPublicCopiesDeletable",
  );
  falseOnly(
    correction.exitAffectsUnrelatedAccess,
    "$receipt.predicate.correction.exitAffectsUnrelatedAccess",
  );

  return status;
}

/**
 * Validate only canonical serialized FC-0 and FC-0.1 receipt bytes.
 * Object-taking validation is intentionally not exported: parsing creates a
 * data-only snapshot before any field is inspected.
 */
export function parseAndValidateFrontierCommonsReceipt(
  commonsRaw,
  receiptRaw,
  { asOf } = {},
) {
  if (typeof commonsRaw !== "string") fail("$commons", "must be a JSON string");

  let commons;
  try {
    commons = parseAndValidateFrontierCommons(commonsRaw);
  } catch (error) {
    const sourcePath = typeof error?.path === "string" ? error.path : "$";
    const path =
      sourcePath === "$" ? "$commons" : `$commons${sourcePath.slice(1)}`;
    fail(path, `FC-0 standard is invalid: ${error.message}`);
  }
  if (commons.schema !== "zerone.frontier-commons-participation/v0") {
    fail("$commons.schema", "must be the FC-0 Frontier Commons standard");
  }
  if (commons.milestone !== "FC-0") {
    fail("$commons.milestone", "must equal FC-0");
  }
  const commonsDigest = sha256(commonsRaw);
  if (commonsDigest !== EXPECTED_FRONTIER_COMMONS_SHA256) {
    fail("$commons", "bytes differ from the canonical FC-0 standard");
  }

  const receipt = parseReceiptRaw(receiptRaw);
  const status = validateParsedReceipt(receipt, asOf);
  const receiptDigest = sha256(receiptRaw);
  if (receiptDigest !== EXPECTED_FRONTIER_COMMONS_RECEIPT_SHA256) {
    fail("$receipt", "bytes differ from the canonical FC-0.1 self-receipt");
  }

  return Object.freeze({
    schema: FRONTIER_COMMONS_RECEIPT_SCHEMA,
    receiptKind: RECEIPT_KIND,
    subjectDigest: EXPECTED_FRONTIER_COMMONS_SHA256,
    receiptDigest,
    result: RESULT,
    relationCount: 0,
    temporalStatus: status,
  });
}

function runCli() {
  if (process.argv.length !== 4) {
    console.error(
      "usage: node scripts/validate-frontier-commons-receipt.mjs FRONTIER_COMMONS_JSON SELF_RECEIPT_JSON",
    );
    process.exitCode = 2;
    return;
  }

  try {
    const result = parseAndValidateFrontierCommonsReceipt(
      readFileSync(resolve(process.argv[2]), "utf8"),
      readFileSync(resolve(process.argv[3]), "utf8"),
    );
    console.log(
      `frontier commons receipt: PASS (FC-0 sha256 ${result.subjectDigest}; self-receipt sha256 ${result.receiptDigest}; ${result.result}/${result.temporalStatus}; unsigned self-dogfood)`,
    );
  } catch (error) {
    console.error(`frontier commons receipt: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === SCRIPT_PATH) runCli();
