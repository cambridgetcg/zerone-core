import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ADAPTER_INDEX_SCHEMA = "zerone.adapter-index/v1";
export const ADAPTER_INDEX_MAX_BYTES = 65_536;

const TOP_LEVEL_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "releaseBoundary",
  "entries",
];
const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesAdapters",
  "performsNetworkRequests",
  "assertsLiveDeployment",
];
const ENTRY_KEYS = [
  "id",
  "kind",
  "status",
  "availability",
  "standards",
  "repositoryReferences",
  "capabilities",
  "trust",
  "boundaries",
];
const STANDARD_KEYS = ["name", "specification"];
const TRUST_KEYS = [
  "networkInput",
  "outputAuthentication",
  "liveDeploymentVerified",
];

const KINDS = new Set([
  "agent-discovery",
  "external-witness",
  "payment",
  "provenance-consumer",
  "provenance-verifier",
]);
const STATUSES = new Set([
  "planned",
  "implemented-source",
  "experimental-unregistered",
]);
const AVAILABILITIES = new Set([
  "unavailable",
  "source-only",
  "local-tool",
  "external-service-required",
]);
const NETWORK_INPUTS = new Set([
  "none",
  "unauthenticated-read",
  "authenticated-external-api",
]);
const OUTPUT_AUTHENTICATION = new Set(["none", "signature-policy-only"]);
const REQUIRED_IDS = [
  "a2a-agent-card",
  "agenttool-invocation-v1",
  "sigstore-in-toto-v1",
  "training-provenance-in-toto-v1",
  "x402-zerone",
];
const ENTRY_POLICIES = Object.freeze({
  "a2a-agent-card": Object.freeze({
    kind: "agent-discovery",
    status: "planned",
    availability: "unavailable",
    capabilities: Object.freeze([]),
    networkInput: "none",
    outputAuthentication: "none",
  }),
  "agenttool-invocation-v1": Object.freeze({
    kind: "external-witness",
    status: "implemented-source",
    availability: "external-service-required",
    capabilities: Object.freeze(["external-invocation-witness-relay"]),
    networkInput: "authenticated-external-api",
    outputAuthentication: "none",
  }),
  "sigstore-in-toto-v1": Object.freeze({
    kind: "provenance-verifier",
    status: "experimental-unregistered",
    availability: "local-tool",
    capabilities: Object.freeze([
      "local-dsse-signature-policy",
      "witness-only-link-compilation",
    ]),
    networkInput: "none",
    outputAuthentication: "signature-policy-only",
  }),
  "training-provenance-in-toto-v1": Object.freeze({
    kind: "provenance-consumer",
    status: "implemented-source",
    availability: "source-only",
    capabilities: Object.freeze([
      "offline-unsigned-profile-parse",
      "read-only-query-source",
    ]),
    networkInput: "none",
    outputAuthentication: "none",
  }),
  "x402-zerone": Object.freeze({
    kind: "payment",
    status: "planned",
    availability: "unavailable",
    capabilities: Object.freeze([]),
    networkInput: "none",
    outputAuthentication: "none",
  }),
});
const ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const CAPABILITY_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const REPOSITORY_PATH_PATTERN =
  /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;
const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "../..");

export class AdapterIndexValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "AdapterIndexValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new AdapterIndexValidationError(path, message);
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
    if (!allowedSet.has(key)) fail(`${path}.${key}`, "is not part of schema v1");
  }
  for (const key of allowed) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is required");
  }
}

function boundedString(value, path, maxBytes = 512) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function stringArray(value, path, { maxItems = 16, pattern } = {}) {
  if (!Array.isArray(value) || value.length > maxItems) {
    fail(path, `must be an array with at most ${maxItems} items`);
  }
  const result = value.map((item, index) => {
    const parsed = boundedString(item, `${path}[${index}]`);
    if (pattern && !pattern.test(parsed)) {
      fail(`${path}[${index}]`, "has an invalid format");
    }
    return parsed;
  });
  if (new Set(result).size !== result.length) fail(path, "must not contain duplicates");
  return result;
}

function falseOnly(value, path) {
  if (value !== false) fail(path, "must be false in this non-authoritative index");
}

function parseHttpsUrl(value, path) {
  const text = boundedString(value, path);
  let parsed;
  try {
    parsed = new URL(text);
  } catch {
    fail(path, "must be an absolute URL");
  }
  if (
    parsed.protocol !== "https:" ||
    parsed.hostname !== "github.com" ||
    parsed.username !== "" ||
    parsed.password !== "" ||
    parsed.search !== "" ||
    parsed.hash !== ""
  ) {
    fail(
      path,
      "must be an HTTPS github.com source without credentials, query parameters, or a fragment",
    );
  }
  return text;
}

function validateTrust(value, path) {
  const trust = record(value, path);
  exactKeys(trust, TRUST_KEYS, path);
  if (!NETWORK_INPUTS.has(trust.networkInput)) {
    fail(`${path}.networkInput`, "has an unknown value");
  }
  if (!OUTPUT_AUTHENTICATION.has(trust.outputAuthentication)) {
    fail(`${path}.outputAuthentication`, "has an unknown value");
  }
  falseOnly(trust.liveDeploymentVerified, `${path}.liveDeploymentVerified`);
  return trust;
}

function validateEntry(value, index) {
  const path = `$.entries[${index}]`;
  const entry = record(value, path);
  exactKeys(entry, ENTRY_KEYS, path);

  const id = boundedString(entry.id, `${path}.id`, 96);
  if (!ID_PATTERN.test(id)) fail(`${path}.id`, "must be a kebab-case identifier");
  if (!KINDS.has(entry.kind)) fail(`${path}.kind`, "has an unknown value");
  if (!STATUSES.has(entry.status)) fail(`${path}.status`, "has an unknown value");
  if (!AVAILABILITIES.has(entry.availability)) {
    fail(`${path}.availability`, "has an unknown value");
  }
  if (entry.status === "planned" && entry.availability !== "unavailable") {
    fail(`${path}.availability`, "planned entries must be unavailable");
  }
  if (entry.status === "experimental-unregistered" && entry.availability !== "local-tool") {
    fail(`${path}.availability`, "experimental unregistered entries must remain local tools");
  }

  if (!Array.isArray(entry.standards) || entry.standards.length > 8) {
    fail(`${path}.standards`, "must contain at most eight standards");
  }
  for (const [standardIndex, candidate] of entry.standards.entries()) {
    const standardPath = `${path}.standards[${standardIndex}]`;
    const standard = record(candidate, standardPath);
    exactKeys(standard, STANDARD_KEYS, standardPath);
    boundedString(standard.name, `${standardPath}.name`, 128);
    parseHttpsUrl(standard.specification, `${standardPath}.specification`);
  }

  const references = stringArray(
    entry.repositoryReferences,
    `${path}.repositoryReferences`,
    { maxItems: 16, pattern: REPOSITORY_PATH_PATTERN },
  );
  if (references.length === 0) {
    fail(`${path}.repositoryReferences`, "must cite at least one repository path");
  }
  for (const [referenceIndex, reference] of references.entries()) {
    if (!existsSync(resolve(REPOSITORY_ROOT, reference))) {
      fail(
        `${path}.repositoryReferences[${referenceIndex}]`,
        "must resolve to an existing repository path",
      );
    }
  }
  const capabilities = stringArray(entry.capabilities, `${path}.capabilities`, {
    maxItems: 16,
    pattern: CAPABILITY_PATTERN,
  });
  const trust = validateTrust(entry.trust, `${path}.trust`);
  const boundaries = stringArray(entry.boundaries, `${path}.boundaries`, {
    maxItems: 16,
  });
  if (boundaries.length === 0) {
    fail(`${path}.boundaries`, "must state at least one boundary");
  }
  if (entry.status === "planned" && capabilities.length !== 0) {
    fail(`${path}.capabilities`, "planned entries cannot advertise capabilities");
  }

  return { id, entry, capabilities, trust };
}

function requireExactPolicy(entries, id, expected) {
  const candidate = entries.find((item) => item.id === id);
  if (!candidate) fail("$.entries", `missing required entry ${id}`);
  const path = `$.entries[${entries.indexOf(candidate)}]`;
  for (const [key, value] of Object.entries(expected)) {
    if (candidate.entry[key] !== value) {
      fail(`${path}.${key}`, `${id} must use ${JSON.stringify(value)}`);
    }
  }
  return { candidate, path };
}

export function validateAdapterIndex(value) {
  const topLevel = record(value, "$");
  exactKeys(topLevel, TOP_LEVEL_KEYS, "$");
  if (topLevel.schema !== ADAPTER_INDEX_SCHEMA) {
    fail("$.schema", `must equal ${ADAPTER_INDEX_SCHEMA}`);
  }
  falseOnly(topLevel.authoritative, "$.authoritative");
  falseOnly(topLevel.networkObserved, "$.networkObserved");

  const releaseBoundary = record(topLevel.releaseBoundary, "$.releaseBoundary");
  exactKeys(releaseBoundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    falseOnly(releaseBoundary[key], `$.releaseBoundary.${key}`);
  }

  if (!Array.isArray(topLevel.entries) || topLevel.entries.length > 32) {
    fail("$.entries", "must be an array with at most 32 entries");
  }
  const entries = topLevel.entries.map(validateEntry);
  const ids = entries.map(({ id }) => id);
  if (new Set(ids).size !== ids.length) fail("$.entries", "IDs must be unique");
  if (ids.join("\n") !== REQUIRED_IDS.join("\n")) {
    fail("$.entries", "must contain exactly the required v1 IDs in sorted order");
  }

  for (const id of REQUIRED_IDS) {
    const policy = ENTRY_POLICIES[id];
    const { candidate, path } = requireExactPolicy(entries, id, {
      kind: policy.kind,
      status: policy.status,
      availability: policy.availability,
    });
    if (
      candidate.capabilities.join("\n") !== policy.capabilities.join("\n")
    ) {
      fail(`${path}.capabilities`, `${id} must use its exact reviewed capability set`);
    }
    if (candidate.trust.networkInput !== policy.networkInput) {
      fail(`${path}.trust.networkInput`, `${id} has the wrong network boundary`);
    }
    if (
      candidate.trust.outputAuthentication !== policy.outputAuthentication
    ) {
      fail(
        `${path}.trust.outputAuthentication`,
        `${id} has the wrong authentication boundary`,
      );
    }
  }

  return Object.freeze({ schema: ADAPTER_INDEX_SCHEMA, entryCount: entries.length });
}

export function parseAndValidateAdapterIndex(json) {
  if (typeof json !== "string") fail("$", "input must be a JSON string");
  if (Buffer.byteLength(json, "utf8") > ADAPTER_INDEX_MAX_BYTES) {
    fail("$", `must be at most ${ADAPTER_INDEX_MAX_BYTES} UTF-8 bytes`);
  }
  let parsed;
  try {
    parsed = JSON.parse(json);
  } catch {
    fail("$", "input must be valid JSON");
  }
  return validateAdapterIndex(parsed);
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === SCRIPT_PATH) {
  const indexPath = resolve(
    process.cwd(),
    process.argv[2] ?? "public/standards/adapter-index.v1.json",
  );
  const result = parseAndValidateAdapterIndex(readFileSync(indexPath, "utf8"));
  console.log(
    `adapter index ${result.schema} is valid (${result.entryCount} entries)`,
  );
}
