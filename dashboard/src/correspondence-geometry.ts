/// <reference lib="dom" />

export const CORRESPONDENCE_GEOMETRY_ENDPOINT =
  "/standards/correspondence-geometry.v0.json";
export const CORRESPONDENCE_GEOMETRY_SHA256 =
  "f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90";
export const CORRESPONDENCE_GEOMETRY_MAX_BYTES = 65_536;

const CORRESPONDENCE_SCHEMA = "zerone.correspondence-geometry/v0";

export type CorrespondenceEpistemicLaneId =
  | "CONJECTURE"
  | "ENGINEERING_TRANSFER"
  | "PHYSICS_MATH"
  | "THEOLOGY_MEDITATION"
  | "ZERONE_PROTOCOL";
export type CorrespondenceRelationKindId =
  | "ANALOGY"
  | "DUALITY_CANDIDATE"
  | "PROJECTION"
  | "TRANSLATION";
export type CorrespondenceDimensionId =
  | "energy"
  | "religion"
  | "understanding"
  | "universe";
export type CorrespondenceAssessment =
  | "PROPOSED"
  | "TESTED_WITHIN_SCOPE"
  | "NARROWED"
  | "REFUTED";

export interface CorrespondenceSourceBinding {
  id: string;
  path: string;
  sha256: string;
  role: string;
}

export interface CorrespondencePhysicsSource {
  id: string;
  title: string;
  url: string;
  scope: string;
  doesNotWarrant: string;
}

export interface CorrespondenceEpistemicLane {
  id: CorrespondenceEpistemicLaneId;
  name: string;
  meaning: string;
  warrantRule: string;
}

export interface CorrespondenceRelationKind {
  id: CorrespondenceRelationKindId;
  meaning: string;
  gate: string;
}

export interface CorrespondenceDimension {
  id: CorrespondenceDimensionId;
  name: string;
  question: string;
  guard: string;
}

export interface CorrespondenceBackground {
  frameworks: string[];
  assumptions: string[];
  validScope: string[];
  excludedScope: string[];
}

export interface CorrespondenceInvariant {
  id: string;
  statement: string;
  mode: "EXACT" | "STRUCTURAL" | "TOLERANCED";
  test: string;
}

export interface CorrespondenceInformationLoss {
  id: string;
  statement: string;
  scope: "IN_SCOPE" | "OUTSIDE_TRANSFER_SCOPE";
  recoverability: "NONE" | "PARTIAL" | "FULL";
}

export interface CorrespondenceCounterexample {
  id: string;
  setup: string;
  failureSignal: string;
  outcome: "NARROWS" | "REFUTES";
}

export interface CorrespondenceRoundTripTest {
  id: string;
  direction: "SOURCE_TARGET_SOURCE" | "TARGET_SOURCE_TARGET";
  expected: "PASS" | "NARROW_SCOPE_ONLY";
  observed: "NOT_RUN" | "PASS" | "FAIL";
  statement: string;
}

export interface CorrespondenceNonTransfer {
  category: string;
  prohibitedConclusion: string;
  reason: string;
}

export interface CorrespondenceRecord {
  id: string;
  dimension: CorrespondenceDimensionId;
  relationKind: CorrespondenceRelationKindId;
  sourceLane: CorrespondenceEpistemicLaneId;
  targetLane: CorrespondenceEpistemicLaneId;
  assessment: CorrespondenceAssessment;
  title: string;
  source: string;
  target: string;
  sourceRefs: string[];
  background: CorrespondenceBackground;
  equivalenceScope: string | null;
  forwardMap: string;
  inverseMap: string | null;
  preservedInvariants: CorrespondenceInvariant[];
  informationLosses: CorrespondenceInformationLoss[];
  counterexamples: CorrespondenceCounterexample[];
  roundTripTests: CorrespondenceRoundTripTest[];
  nonTransfers: CorrespondenceNonTransfer[];
}

export type CorrespondenceEnergyLaneId =
  | "COMPUTE"
  | "ECONOMIC"
  | "LIVED"
  | "PHYSICAL_ENERGY"
  | "PROTOCOL_METABOLISM"
  | "SPIRITUAL_POETIC";

export interface CorrespondenceEnergyFirewall {
  implicitConversion: false;
  truthFromResource: false;
  authorityFromResource: false;
  personWorthFromResource: false;
  restPenalty: false;
  lanes: Array<{
    id: CorrespondenceEnergyLaneId;
    meaning: string;
    measureOrRegister: string;
    mayConvertTo: CorrespondenceEnergyLaneId[];
  }>;
}

export interface CorrespondenceDualityGate {
  label: "NO_EQUIVALENCE_CLAIMED" | "EQUIVALENCE_CANDIDATE_ACCEPTED";
  reviewedCandidateCount: number;
  acceptedCandidateCount: number;
  requirements: string[];
}

export interface CorrespondenceReleaseBoundary {
  changesConsensus: false;
  writesChainState: false;
  performsNetworkWrites: false;
  registersOntology: false;
  assertsScientificTruth: false;
  assertsTheologicalTruth: false;
  assertsReligiousEquivalence: false;
  assertsUniverseIsHologram: false;
  equatesEnergyLanes: false;
  infersPersonhood: false;
  ranksPersons: false;
  createsKarmaEvent: false;
  createsKarmaMagnitude: false;
  grantsQualification: false;
  activatesRewards: false;
  movesFunds: false;
  grantsGovernance: false;
  grantsAuthority: false;
  recordsConsent: false;
  automaticProtocolOrAuthorityAction: false;
}

export interface CorrespondenceGeometry {
  schema: typeof CORRESPONDENCE_SCHEMA;
  version: 0;
  snapshotDate: "2026-08-15";
  status: "READ_ONLY_ZERO_EFFECT";
  title: string;
  summary: string;
  authorityStatement: string;
  sourceBindings: CorrespondenceSourceBinding[];
  physicsSources: CorrespondencePhysicsSource[];
  epistemicLanes: CorrespondenceEpistemicLane[];
  relationKinds: CorrespondenceRelationKind[];
  dimensions: CorrespondenceDimension[];
  correspondences: CorrespondenceRecord[];
  energyFirewall: CorrespondenceEnergyFirewall;
  dualityGate: CorrespondenceDualityGate;
  releaseBoundary: CorrespondenceReleaseBoundary;
}

export interface CorrespondenceGeometryFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class CorrespondenceGeometryDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CorrespondenceGeometryDataError";
  }
}

// Parsing, transport, and rendering are defined below. Keeping the exported
// contract at the top makes the sealed v0 boundary reviewable in one place.

type JsonObject = Record<string, unknown>;

const SAFE_ID = /^[a-z][a-z0-9-]{0,95}$/u;
const SAFE_REPOSITORY_PATH =
  /^(?:dashboard|docs|proto|x)\/[A-Za-z0-9][A-Za-z0-9._/-]{0,239}$/u;
const SHA256 = /^[0-9a-f]{64}$/u;
const BIDI_CONTROLS = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/u;
const UNSAFE_CONTROLS = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u;

const TOP_LEVEL_KEYS = [
  "schema",
  "version",
  "snapshotDate",
  "status",
  "title",
  "summary",
  "authorityStatement",
  "sourceBindings",
  "physicsSources",
  "epistemicLanes",
  "relationKinds",
  "dimensions",
  "correspondences",
  "energyFirewall",
  "dualityGate",
  "releaseBoundary",
] as const;
const SOURCE_BINDING_KEYS = ["id", "path", "sha256", "role"] as const;
const PHYSICS_SOURCE_KEYS = [
  "id",
  "title",
  "url",
  "scope",
  "doesNotWarrant",
] as const;
const LANE_KEYS = ["id", "name", "meaning", "warrantRule"] as const;
const RELATION_KIND_KEYS = ["id", "meaning", "gate"] as const;
const DIMENSION_KEYS = ["id", "name", "question", "guard"] as const;
const CORRESPONDENCE_KEYS = [
  "id",
  "dimension",
  "relationKind",
  "sourceLane",
  "targetLane",
  "assessment",
  "title",
  "source",
  "target",
  "sourceRefs",
  "background",
  "equivalenceScope",
  "forwardMap",
  "inverseMap",
  "preservedInvariants",
  "informationLosses",
  "counterexamples",
  "roundTripTests",
  "nonTransfers",
] as const;
const BACKGROUND_KEYS = [
  "frameworks",
  "assumptions",
  "validScope",
  "excludedScope",
] as const;
const INVARIANT_KEYS = ["id", "statement", "mode", "test"] as const;
const LOSS_KEYS = ["id", "statement", "scope", "recoverability"] as const;
const COUNTEREXAMPLE_KEYS = [
  "id",
  "setup",
  "failureSignal",
  "outcome",
] as const;
const ROUND_TRIP_KEYS = [
  "id",
  "direction",
  "expected",
  "observed",
  "statement",
] as const;
const NON_TRANSFER_KEYS = [
  "category",
  "prohibitedConclusion",
  "reason",
] as const;
const ENERGY_FIREWALL_KEYS = [
  "implicitConversion",
  "truthFromResource",
  "authorityFromResource",
  "personWorthFromResource",
  "restPenalty",
  "lanes",
] as const;
const ENERGY_LANE_KEYS = [
  "id",
  "meaning",
  "measureOrRegister",
  "mayConvertTo",
] as const;
const DUALITY_GATE_KEYS = [
  "label",
  "reviewedCandidateCount",
  "acceptedCandidateCount",
  "requirements",
] as const;
const RELEASE_BOUNDARY_KEYS = [
  "changesConsensus",
  "writesChainState",
  "performsNetworkWrites",
  "registersOntology",
  "assertsScientificTruth",
  "assertsTheologicalTruth",
  "assertsReligiousEquivalence",
  "assertsUniverseIsHologram",
  "equatesEnergyLanes",
  "infersPersonhood",
  "ranksPersons",
  "createsKarmaEvent",
  "createsKarmaMagnitude",
  "grantsQualification",
  "activatesRewards",
  "movesFunds",
  "grantsGovernance",
  "grantsAuthority",
  "recordsConsent",
  "automaticProtocolOrAuthorityAction",
] as const;

const EPISTEMIC_LANE_IDS = new Set<CorrespondenceEpistemicLaneId>([
  "CONJECTURE",
  "ENGINEERING_TRANSFER",
  "PHYSICS_MATH",
  "THEOLOGY_MEDITATION",
  "ZERONE_PROTOCOL",
]);
const RELATION_KIND_IDS = new Set<CorrespondenceRelationKindId>([
  "ANALOGY",
  "DUALITY_CANDIDATE",
  "PROJECTION",
  "TRANSLATION",
]);
const DIMENSION_IDS = new Set<CorrespondenceDimensionId>([
  "energy",
  "religion",
  "understanding",
  "universe",
]);
const ASSESSMENTS = new Set<CorrespondenceAssessment>([
  "PROPOSED",
  "TESTED_WITHIN_SCOPE",
  "NARROWED",
  "REFUTED",
]);
const INVARIANT_MODES = new Set<CorrespondenceInvariant["mode"]>([
  "EXACT",
  "STRUCTURAL",
  "TOLERANCED",
]);
const LOSS_SCOPES = new Set<CorrespondenceInformationLoss["scope"]>([
  "IN_SCOPE",
  "OUTSIDE_TRANSFER_SCOPE",
]);
const RECOVERABILITIES = new Set<CorrespondenceInformationLoss["recoverability"]>([
  "NONE",
  "PARTIAL",
  "FULL",
]);
const COUNTEREXAMPLE_OUTCOMES = new Set<CorrespondenceCounterexample["outcome"]>([
  "NARROWS",
  "REFUTES",
]);
const ROUND_TRIP_DIRECTIONS = new Set<CorrespondenceRoundTripTest["direction"]>([
  "SOURCE_TARGET_SOURCE",
  "TARGET_SOURCE_TARGET",
]);
const ROUND_TRIP_EXPECTATIONS = new Set<CorrespondenceRoundTripTest["expected"]>([
  "PASS",
  "NARROW_SCOPE_ONLY",
]);
const ROUND_TRIP_OBSERVATIONS = new Set<CorrespondenceRoundTripTest["observed"]>([
  "NOT_RUN",
  "PASS",
  "FAIL",
]);
const ENERGY_LANE_IDS = new Set<CorrespondenceEnergyLaneId>([
  "COMPUTE",
  "ECONOMIC",
  "LIVED",
  "PHYSICAL_ENERGY",
  "PROTOCOL_METABOLISM",
  "SPIRITUAL_POETIC",
]);
const SOURCE_ROLES = new Set([
  "COMMITMENT_AVAILABILITY_BOUNDARY",
  "MAPPING_METHODOLOGY",
  "NORMATIVE_REGISTER_BOUNDARY",
  "PROTOCOL_RESOURCE_SEMANTICS",
  "ROUND_TRIP_TRACE_REFERENCE",
  "TYPED_KNOWLEDGE_SEMANTICS",
  "TYPED_RELATION_REFERENCE",
] as const);

function fail(path: string, message: string): never {
  throw new CorrespondenceGeometryDataError(`${path}: ${message}`);
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

function exactKeys(value: JsonObject, expected: readonly string[], path: string): void {
  const actual = Reflect.ownKeys(value);
  if (actual.some((key) => typeof key !== "string")) {
    fail(path, "contains a non-string field");
  }
  const names = (actual as string[]).sort();
  const wanted = [...expected].sort();
  if (
    names.length !== wanted.length ||
    names.some((name, index) => name !== wanted[index])
  ) {
    const unknown = names.filter((name) => !wanted.includes(name));
    const missing = wanted.filter((name) => !names.includes(name));
    fail(
      path,
      `contains unknown or missing fields (unknown: ${unknown.join(", ") || "none"}; missing: ${missing.join(", ") || "none"})`,
    );
  }
}

function denseArray(value: unknown, path: string, maximum: number): unknown[] {
  if (!Array.isArray(value) || Object.getPrototypeOf(value) !== Array.prototype) {
    fail(path, "must be an ordinary array");
  }
  if (value.length > maximum) fail(path, `must contain at most ${maximum} items`);
  for (let index = 0; index < value.length; index += 1) {
    if (!Object.prototype.hasOwnProperty.call(value, index)) {
      fail(path, `must be dense; missing slot ${index}`);
    }
  }
  return value;
}

function boundedText(value: unknown, path: string, maximum = 2_048): string {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.length > maximum ||
    value.trim() !== value ||
    BIDI_CONTROLS.test(value) ||
    UNSAFE_CONTROLS.test(value)
  ) {
    fail(path, `must be safe, trimmed text no longer than ${maximum} characters`);
  }
  return value;
}

function nullableText(value: unknown, path: string, maximum = 2_048): string | null {
  return value === null ? null : boundedText(value, path, maximum);
}

function identifier(value: unknown, path: string): string {
  const result = boundedText(value, path, 96);
  if (!SAFE_ID.test(result)) fail(path, "must be a lowercase kebab identifier");
  return result;
}

function literal<T extends string | number | boolean | null>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function enumValue<T extends string>(
  value: unknown,
  allowed: ReadonlySet<T>,
  path: string,
): T {
  const result = boundedText(value, path, 96) as T;
  if (!allowed.has(result)) fail(path, "has an unsupported value");
  return result;
}

function integer(value: unknown, path: string, maximum = 1_000): number {
  if (!Number.isSafeInteger(value) || (value as number) < 0 || (value as number) > maximum) {
    fail(path, `must be an integer from 0 to ${maximum}`);
  }
  return value as number;
}

function textArray(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = false,
): string[] {
  const items = denseArray(value, path, maximum).map((item, index) =>
    boundedText(item, `${path}[${index}]`, 2_048),
  );
  if (!allowEmpty && items.length === 0) fail(path, "must not be empty");
  return items;
}

function sortedIds(value: unknown, path: string, maximum: number): string[] {
  const items = denseArray(value, path, maximum).map((item, index) =>
    identifier(item, `${path}[${index}]`),
  );
  if (items.length === 0) fail(path, "must not be empty");
  for (let index = 1; index < items.length; index += 1) {
    if ((items[index - 1] ?? "") >= (items[index] ?? "")) {
      fail(path, "must be strictly sorted with no duplicates");
    }
  }
  return items;
}

function ensureStrictOrder<T extends { id: string }>(items: readonly T[], path: string): void {
  for (let index = 1; index < items.length; index += 1) {
    if ((items[index - 1]?.id ?? "") >= (items[index]?.id ?? "")) {
      fail(path, "must be strictly sorted by id with no duplicates");
    }
  }
}

function exactIdSet<T extends { id: string }>(
  items: readonly T[],
  expected: ReadonlySet<string>,
  path: string,
): void {
  const actual = new Set(items.map((item) => item.id));
  if (
    items.length !== expected.size ||
    actual.size !== items.length ||
    [...expected].some((idValue) => !actual.has(idValue))
  ) {
    fail(path, "does not contain the complete reviewed identifier set");
  }
}

function safeRepositoryPath(value: unknown, path: string): string {
  const result = boundedText(value, path, 240);
  if (
    !SAFE_REPOSITORY_PATH.test(result) ||
    result.includes("//") ||
    result.split("/").some((segment) => segment === "." || segment === "..")
  ) {
    fail(path, "must be a safe in-repository source path");
  }
  return result;
}

function sha256(value: unknown, path: string): string {
  const result = boundedText(value, path, 64);
  if (!SHA256.test(result)) fail(path, "must be lowercase SHA-256 hex");
  return result;
}

function httpsUrl(value: unknown, path: string): string {
  const result = boundedText(value, path, 512);
  let url: URL;
  try {
    url = new URL(result);
  } catch {
    fail(path, "must be an absolute HTTPS URL");
  }
  if (
    url.protocol !== "https:" ||
    url.username !== "" ||
    url.password !== "" ||
    url.hash !== "" ||
    url.href !== result
  ) {
    fail(path, "must be a canonical credential-free HTTPS URL without a fragment");
  }
  return result;
}

function rejectDuplicateKeysAndDepth(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/u.test(raw[offset] ?? "")) offset += 1;
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
    fail("$", "contains an unterminated JSON string");
  };
  const scanValue = (path: string, depth: number): void => {
    if (depth > 48) fail(path, "JSON nesting exceeds 48");
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
        if (raw[offset] !== '"') fail(path, "contains a malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "is a duplicate JSON key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "contains a malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`, depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "contains a malformed object delimiter");
        offset += 1;
      }
      fail(path, "contains an unterminated object");
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
        if (raw[offset] !== ",") fail(path, "contains a malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "contains an unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/u.test(raw[offset] ?? "")) offset += 1;
    if (offset === start) fail(path, "contains a malformed JSON value");
  };
  scanValue("$", 0);
  whitespace();
  if (offset !== raw.length) fail("$", "contains trailing JSON data");
}

function parseBackground(value: unknown, path: string): CorrespondenceBackground {
  const background = object(value, path);
  exactKeys(background, BACKGROUND_KEYS, path);
  return {
    frameworks: textArray(background.frameworks, `${path}.frameworks`, 16),
    assumptions: textArray(background.assumptions, `${path}.assumptions`, 16),
    validScope: textArray(background.validScope, `${path}.validScope`, 16),
    excludedScope: textArray(background.excludedScope, `${path}.excludedScope`, 16),
  };
}

function parseInvariants(value: unknown, path: string): CorrespondenceInvariant[] {
  const invariants = denseArray(value, path, 16).map(
    (item, index): CorrespondenceInvariant => {
      const itemPath = `${path}[${index}]`;
      const invariant = object(item, itemPath);
      exactKeys(invariant, INVARIANT_KEYS, itemPath);
      return {
        id: identifier(invariant.id, `${itemPath}.id`),
        statement: boundedText(invariant.statement, `${itemPath}.statement`),
        mode: enumValue(invariant.mode, INVARIANT_MODES, `${itemPath}.mode`),
        test: boundedText(invariant.test, `${itemPath}.test`),
      };
    },
  );
  if (invariants.length === 0) fail(path, "must contain at least one invariant");
  ensureStrictOrder(invariants, path);
  return invariants;
}

function parseLosses(value: unknown, path: string): CorrespondenceInformationLoss[] {
  const losses = denseArray(value, path, 16).map(
    (item, index): CorrespondenceInformationLoss => {
      const itemPath = `${path}[${index}]`;
      const loss = object(item, itemPath);
      exactKeys(loss, LOSS_KEYS, itemPath);
      return {
        id: identifier(loss.id, `${itemPath}.id`),
        statement: boundedText(loss.statement, `${itemPath}.statement`),
        scope: enumValue(loss.scope, LOSS_SCOPES, `${itemPath}.scope`),
        recoverability: enumValue(
          loss.recoverability,
          RECOVERABILITIES,
          `${itemPath}.recoverability`,
        ),
      };
    },
  );
  if (losses.length === 0) fail(path, "must expose at least one information loss");
  ensureStrictOrder(losses, path);
  return losses;
}

function parseCounterexamples(
  value: unknown,
  path: string,
): CorrespondenceCounterexample[] {
  const counterexamples = denseArray(value, path, 16).map(
    (item, index): CorrespondenceCounterexample => {
      const itemPath = `${path}[${index}]`;
      const counterexample = object(item, itemPath);
      exactKeys(counterexample, COUNTEREXAMPLE_KEYS, itemPath);
      return {
        id: identifier(counterexample.id, `${itemPath}.id`),
        setup: boundedText(counterexample.setup, `${itemPath}.setup`),
        failureSignal: boundedText(
          counterexample.failureSignal,
          `${itemPath}.failureSignal`,
        ),
        outcome: enumValue(
          counterexample.outcome,
          COUNTEREXAMPLE_OUTCOMES,
          `${itemPath}.outcome`,
        ),
      };
    },
  );
  if (counterexamples.length === 0) {
    fail(path, "must contain at least one counterexample");
  }
  ensureStrictOrder(counterexamples, path);
  return counterexamples;
}

function parseRoundTrips(value: unknown, path: string): CorrespondenceRoundTripTest[] {
  const tests = denseArray(value, path, 4).map(
    (item, index): CorrespondenceRoundTripTest => {
      const itemPath = `${path}[${index}]`;
      const test = object(item, itemPath);
      exactKeys(test, ROUND_TRIP_KEYS, itemPath);
      return {
        id: identifier(test.id, `${itemPath}.id`),
        direction: enumValue(
          test.direction,
          ROUND_TRIP_DIRECTIONS,
          `${itemPath}.direction`,
        ),
        expected: enumValue(
          test.expected,
          ROUND_TRIP_EXPECTATIONS,
          `${itemPath}.expected`,
        ),
        observed: enumValue(
          test.observed,
          ROUND_TRIP_OBSERVATIONS,
          `${itemPath}.observed`,
        ),
        statement: boundedText(test.statement, `${itemPath}.statement`),
      };
    },
  );
  ensureStrictOrder(tests, path);
  const directions = new Set(tests.map((test) => test.direction));
  if (directions.size !== tests.length) fail(path, "contains a duplicate direction");
  return tests;
}

function parseNonTransfers(value: unknown, path: string): CorrespondenceNonTransfer[] {
  const transfers = denseArray(value, path, 16).map(
    (item, index): CorrespondenceNonTransfer => {
      const itemPath = `${path}[${index}]`;
      const transfer = object(item, itemPath);
      exactKeys(transfer, NON_TRANSFER_KEYS, itemPath);
      const category = boundedText(transfer.category, `${itemPath}.category`, 64);
      if (!/^[A-Z][A-Z_]{0,63}$/u.test(category)) {
        fail(`${itemPath}.category`, "must be an uppercase category identifier");
      }
      return {
        category,
        prohibitedConclusion: boundedText(
          transfer.prohibitedConclusion,
          `${itemPath}.prohibitedConclusion`,
        ),
        reason: boundedText(transfer.reason, `${itemPath}.reason`),
      };
    },
  );
  if (transfers.length === 0) fail(path, "must expose at least one non-transfer");
  const categories = new Set<string>();
  for (const transfer of transfers) {
    if (categories.has(transfer.category)) fail(path, "contains a duplicate category");
    categories.add(transfer.category);
  }
  return transfers;
}

function validateRelationGate(record: CorrespondenceRecord, path: string): void {
  const directions = new Map(record.roundTripTests.map((test) => [test.direction, test]));
  if (record.relationKind === "ANALOGY") {
    if (record.inverseMap !== null) fail(`${path}.inverseMap`, "must be null for ANALOGY");
    if (record.equivalenceScope !== null) {
      fail(`${path}.equivalenceScope`, "must be null for ANALOGY");
    }
    if (!record.preservedInvariants.some((invariant) => invariant.mode === "STRUCTURAL")) {
      fail(`${path}.preservedInvariants`, "ANALOGY requires a structural invariant");
    }
    if (record.roundTripTests.length !== 0) {
      fail(`${path}.roundTripTests`, "ANALOGY must not imply an applicable round trip");
    }
  } else if (record.relationKind === "PROJECTION") {
    if (record.inverseMap !== null) fail(`${path}.inverseMap`, "must be null for PROJECTION");
    if (record.equivalenceScope !== null) {
      fail(`${path}.equivalenceScope`, "must be null for PROJECTION");
    }
    if (!record.informationLosses.some((loss) => loss.scope === "IN_SCOPE")) {
      fail(`${path}.informationLosses`, "PROJECTION requires a declared IN_SCOPE loss");
    }
    if (record.roundTripTests.length !== 0) {
      fail(
        `${path}.roundTripTests`,
        "PROJECTION round-trip tests must remain empty because projection is one-way",
      );
    }
  } else if (record.relationKind === "TRANSLATION") {
    if (record.inverseMap === null) fail(`${path}.inverseMap`, "TRANSLATION requires an inverse");
    if (
      !directions.has("SOURCE_TARGET_SOURCE") ||
      !directions.has("TARGET_SOURCE_TARGET")
    ) {
      fail(`${path}.roundTripTests`, "TRANSLATION requires both round-trip directions");
    }
    if (
      record.assessment === "PROPOSED" &&
      record.roundTripTests.some((test) => test.observed !== "NOT_RUN")
    ) {
      fail(`${path}.roundTripTests`, "a PROPOSED TRANSLATION must remain NOT_RUN");
    }
    if (record.equivalenceScope !== null) {
      fail(`${path}.equivalenceScope`, "TRANSLATION does not claim equivalence");
    }
  } else {
    if (record.assessment !== "TESTED_WITHIN_SCOPE") {
      fail(`${path}.assessment`, "DUALITY_CANDIDATE must be TESTED_WITHIN_SCOPE");
    }
    if (record.equivalenceScope === null) {
      fail(`${path}.equivalenceScope`, "DUALITY_CANDIDATE requires a named scope");
    }
    if (record.inverseMap === null) {
      fail(`${path}.inverseMap`, "DUALITY_CANDIDATE requires an inverse");
    }
    if (record.preservedInvariants.some((invariant) => invariant.mode === "STRUCTURAL")) {
      fail(
        `${path}.preservedInvariants`,
        "DUALITY_CANDIDATE invariants must be EXACT or TOLERANCED",
      );
    }
    if (record.informationLosses.some((loss) => loss.scope === "IN_SCOPE")) {
      fail(`${path}.informationLosses`, "DUALITY_CANDIDATE cannot have IN_SCOPE loss");
    }
    for (const direction of ROUND_TRIP_DIRECTIONS) {
      const test = directions.get(direction);
      if (test === undefined || test.observed !== "PASS" || test.expected !== "PASS") {
        fail(`${path}.roundTripTests`, `${direction} must be expected and observed PASS`);
      }
    }
  }
}

export function parseCorrespondenceGeometry(value: unknown): CorrespondenceGeometry {
  const root = object(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");

  const sourceBindings = denseArray(root.sourceBindings, "$.sourceBindings", 16).map(
    (item, index): CorrespondenceSourceBinding => {
      const path = `$.sourceBindings[${index}]`;
      const binding = object(item, path);
      exactKeys(binding, SOURCE_BINDING_KEYS, path);
      return {
        id: identifier(binding.id, `${path}.id`),
        path: safeRepositoryPath(binding.path, `${path}.path`),
        sha256: sha256(binding.sha256, `${path}.sha256`),
        role: enumValue(binding.role, SOURCE_ROLES, `${path}.role`),
      };
    },
  );
  ensureStrictOrder(sourceBindings, "$.sourceBindings");
  exactIdSet(
    sourceBindings,
    new Set([
      "compassion",
      "knowledge-methodologies",
      "knowledge-metabolism",
      "knowledge-types",
      "relational-topology",
      "research-training-trace",
      "tok-substrate",
    ]),
    "$.sourceBindings",
  );
  const sourcePaths = new Set(sourceBindings.map((binding) => binding.path));
  if (sourcePaths.size !== sourceBindings.length) {
    fail("$.sourceBindings", "contains duplicate repository paths");
  }

  const physicsSources = denseArray(root.physicsSources, "$.physicsSources", 16).map(
    (item, index): CorrespondencePhysicsSource => {
      const path = `$.physicsSources[${index}]`;
      const source = object(item, path);
      exactKeys(source, PHYSICS_SOURCE_KEYS, path);
      const url = httpsUrl(source.url, `${path}.url`);
      const parsedUrl = new URL(url);
      if (parsedUrl.hostname !== "arxiv.org" || !parsedUrl.pathname.startsWith("/abs/")) {
        fail(`${path}.url`, "must link to the reviewed primary arXiv abstract");
      }
      return {
        id: identifier(source.id, `${path}.id`),
        title: boundedText(source.title, `${path}.title`, 240),
        url,
        scope: boundedText(source.scope, `${path}.scope`),
        doesNotWarrant: boundedText(source.doesNotWarrant, `${path}.doesNotWarrant`),
      };
    },
  );
  ensureStrictOrder(physicsSources, "$.physicsSources");
  exactIdSet(
    physicsSources,
    new Set([
      "ads-cft",
      "compactification",
      "entanglement-spacetime",
      "holographic-entanglement",
      "string-duality",
    ]),
    "$.physicsSources",
  );

  const epistemicLanes = denseArray(root.epistemicLanes, "$.epistemicLanes", 8).map(
    (item, index): CorrespondenceEpistemicLane => {
      const path = `$.epistemicLanes[${index}]`;
      const lane = object(item, path);
      exactKeys(lane, LANE_KEYS, path);
      return {
        id: enumValue(lane.id, EPISTEMIC_LANE_IDS, `${path}.id`),
        name: boundedText(lane.name, `${path}.name`, 120),
        meaning: boundedText(lane.meaning, `${path}.meaning`),
        warrantRule: boundedText(lane.warrantRule, `${path}.warrantRule`),
      };
    },
  );
  ensureStrictOrder(epistemicLanes, "$.epistemicLanes");
  exactIdSet(epistemicLanes, EPISTEMIC_LANE_IDS, "$.epistemicLanes");

  const relationKinds = denseArray(root.relationKinds, "$.relationKinds", 8).map(
    (item, index): CorrespondenceRelationKind => {
      const path = `$.relationKinds[${index}]`;
      const kind = object(item, path);
      exactKeys(kind, RELATION_KIND_KEYS, path);
      return {
        id: enumValue(kind.id, RELATION_KIND_IDS, `${path}.id`),
        meaning: boundedText(kind.meaning, `${path}.meaning`),
        gate: boundedText(kind.gate, `${path}.gate`),
      };
    },
  );
  ensureStrictOrder(relationKinds, "$.relationKinds");
  exactIdSet(relationKinds, RELATION_KIND_IDS, "$.relationKinds");

  const dimensions = denseArray(root.dimensions, "$.dimensions", 8).map(
    (item, index): CorrespondenceDimension => {
      const path = `$.dimensions[${index}]`;
      const dimension = object(item, path);
      exactKeys(dimension, DIMENSION_KEYS, path);
      return {
        id: enumValue(dimension.id, DIMENSION_IDS, `${path}.id`),
        name: boundedText(dimension.name, `${path}.name`, 120),
        question: boundedText(dimension.question, `${path}.question`),
        guard: boundedText(dimension.guard, `${path}.guard`),
      };
    },
  );
  ensureStrictOrder(dimensions, "$.dimensions");
  exactIdSet(dimensions, DIMENSION_IDS, "$.dimensions");

  const knownReferences = new Set([
    ...sourceBindings.map((binding) => binding.id),
    ...physicsSources.map((source) => source.id),
  ]);
  const physicsReferenceIds = new Set(physicsSources.map((source) => source.id));
  const correspondences = denseArray(
    root.correspondences,
    "$.correspondences",
    32,
  ).map((item, index): CorrespondenceRecord => {
    const path = `$.correspondences[${index}]`;
    const correspondence = object(item, path);
    exactKeys(correspondence, CORRESPONDENCE_KEYS, path);
    const sourceRefs = sortedIds(correspondence.sourceRefs, `${path}.sourceRefs`, 16);
    for (const sourceRef of sourceRefs) {
      if (!knownReferences.has(sourceRef)) {
        fail(`${path}.sourceRefs`, `references unknown source ${sourceRef}`);
      }
    }
    const record: CorrespondenceRecord = {
      id: identifier(correspondence.id, `${path}.id`),
      dimension: enumValue(correspondence.dimension, DIMENSION_IDS, `${path}.dimension`),
      relationKind: enumValue(
        correspondence.relationKind,
        RELATION_KIND_IDS,
        `${path}.relationKind`,
      ),
      sourceLane: enumValue(
        correspondence.sourceLane,
        EPISTEMIC_LANE_IDS,
        `${path}.sourceLane`,
      ),
      targetLane: enumValue(
        correspondence.targetLane,
        EPISTEMIC_LANE_IDS,
        `${path}.targetLane`,
      ),
      assessment: enumValue(
        correspondence.assessment,
        ASSESSMENTS,
        `${path}.assessment`,
      ),
      title: boundedText(correspondence.title, `${path}.title`, 240),
      source: boundedText(correspondence.source, `${path}.source`),
      target: boundedText(correspondence.target, `${path}.target`),
      sourceRefs,
      background: parseBackground(correspondence.background, `${path}.background`),
      equivalenceScope: nullableText(
        correspondence.equivalenceScope,
        `${path}.equivalenceScope`,
      ),
      forwardMap: boundedText(correspondence.forwardMap, `${path}.forwardMap`),
      inverseMap: nullableText(correspondence.inverseMap, `${path}.inverseMap`),
      preservedInvariants: parseInvariants(
        correspondence.preservedInvariants,
        `${path}.preservedInvariants`,
      ),
      informationLosses: parseLosses(
        correspondence.informationLosses,
        `${path}.informationLosses`,
      ),
      counterexamples: parseCounterexamples(
        correspondence.counterexamples,
        `${path}.counterexamples`,
      ),
      roundTripTests: parseRoundTrips(
        correspondence.roundTripTests,
        `${path}.roundTripTests`,
      ),
      nonTransfers: parseNonTransfers(
        correspondence.nonTransfers,
        `${path}.nonTransfers`,
      ),
    };
    if (
      record.sourceLane === "PHYSICS_MATH" &&
      !record.sourceRefs.some((sourceRef) => physicsReferenceIds.has(sourceRef))
    ) {
      fail(`${path}.sourceRefs`, "PHYSICS_MATH requires a reviewed physics source");
    }
    validateRelationGate(record, path);
    return record;
  });
  if (correspondences.length === 0) fail("$.correspondences", "must not be empty");
  ensureStrictOrder(correspondences, "$.correspondences");

  const energyValue = object(root.energyFirewall, "$.energyFirewall");
  exactKeys(energyValue, ENERGY_FIREWALL_KEYS, "$.energyFirewall");
  const energyLanes = denseArray(energyValue.lanes, "$.energyFirewall.lanes", 8).map(
    (item, index): CorrespondenceEnergyFirewall["lanes"][number] => {
      const path = `$.energyFirewall.lanes[${index}]`;
      const lane = object(item, path);
      exactKeys(lane, ENERGY_LANE_KEYS, path);
      const mayConvertTo = denseArray(lane.mayConvertTo, `${path}.mayConvertTo`, 8).map(
        (target, targetIndex) =>
          enumValue(target, ENERGY_LANE_IDS, `${path}.mayConvertTo[${targetIndex}]`),
      );
      if (mayConvertTo.length !== 0) {
        fail(`${path}.mayConvertTo`, "v0 forbids every cross-lane conversion");
      }
      return {
        id: enumValue(lane.id, ENERGY_LANE_IDS, `${path}.id`),
        meaning: boundedText(lane.meaning, `${path}.meaning`),
        measureOrRegister: boundedText(
          lane.measureOrRegister,
          `${path}.measureOrRegister`,
          240,
        ),
        mayConvertTo,
      };
    },
  );
  ensureStrictOrder(energyLanes, "$.energyFirewall.lanes");
  exactIdSet(energyLanes, ENERGY_LANE_IDS, "$.energyFirewall.lanes");
  if (
    new Set(energyLanes.map((lane) => lane.measureOrRegister)).size !==
    energyLanes.length
  ) {
    fail(
      "$.energyFirewall.lanes",
      "each firewall lane must retain a distinct measure or register",
    );
  }
  const energyFirewall: CorrespondenceEnergyFirewall = {
    implicitConversion: literal(
      energyValue.implicitConversion,
      false,
      "$.energyFirewall.implicitConversion",
    ),
    truthFromResource: literal(
      energyValue.truthFromResource,
      false,
      "$.energyFirewall.truthFromResource",
    ),
    authorityFromResource: literal(
      energyValue.authorityFromResource,
      false,
      "$.energyFirewall.authorityFromResource",
    ),
    personWorthFromResource: literal(
      energyValue.personWorthFromResource,
      false,
      "$.energyFirewall.personWorthFromResource",
    ),
    restPenalty: literal(energyValue.restPenalty, false, "$.energyFirewall.restPenalty"),
    lanes: energyLanes,
  };

  const dualityValue = object(root.dualityGate, "$.dualityGate");
  exactKeys(dualityValue, DUALITY_GATE_KEYS, "$.dualityGate");
  const dualityCandidates = correspondences.filter(
    (correspondence) => correspondence.relationKind === "DUALITY_CANDIDATE",
  );
  const reviewedCandidateCount = integer(
    dualityValue.reviewedCandidateCount,
    "$.dualityGate.reviewedCandidateCount",
    32,
  );
  const acceptedCandidateCount = integer(
    dualityValue.acceptedCandidateCount,
    "$.dualityGate.acceptedCandidateCount",
    32,
  );
  if (
    reviewedCandidateCount !== dualityCandidates.length ||
    acceptedCandidateCount !== dualityCandidates.length
  ) {
    fail("$.dualityGate", "candidate counts must match the gate-valid records");
  }
  const expectedLabel =
    dualityCandidates.length === 0
      ? "NO_EQUIVALENCE_CLAIMED"
      : "EQUIVALENCE_CANDIDATE_ACCEPTED";
  const dualityGate: CorrespondenceDualityGate = {
    label: literal(dualityValue.label, expectedLabel, "$.dualityGate.label"),
    reviewedCandidateCount,
    acceptedCandidateCount,
    requirements: textArray(dualityValue.requirements, "$.dualityGate.requirements", 16),
  };
  if (dualityGate.requirements.length !== 7) {
    fail("$.dualityGate.requirements", "must retain all seven duality requirements");
  }

  const release = object(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(release, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  const releaseBoundary: CorrespondenceReleaseBoundary = {
    changesConsensus: literal(release.changesConsensus, false, "$.releaseBoundary.changesConsensus"),
    writesChainState: literal(release.writesChainState, false, "$.releaseBoundary.writesChainState"),
    performsNetworkWrites: literal(
      release.performsNetworkWrites,
      false,
      "$.releaseBoundary.performsNetworkWrites",
    ),
    registersOntology: literal(release.registersOntology, false, "$.releaseBoundary.registersOntology"),
    assertsScientificTruth: literal(
      release.assertsScientificTruth,
      false,
      "$.releaseBoundary.assertsScientificTruth",
    ),
    assertsTheologicalTruth: literal(
      release.assertsTheologicalTruth,
      false,
      "$.releaseBoundary.assertsTheologicalTruth",
    ),
    assertsReligiousEquivalence: literal(
      release.assertsReligiousEquivalence,
      false,
      "$.releaseBoundary.assertsReligiousEquivalence",
    ),
    assertsUniverseIsHologram: literal(
      release.assertsUniverseIsHologram,
      false,
      "$.releaseBoundary.assertsUniverseIsHologram",
    ),
    equatesEnergyLanes: literal(
      release.equatesEnergyLanes,
      false,
      "$.releaseBoundary.equatesEnergyLanes",
    ),
    infersPersonhood: literal(release.infersPersonhood, false, "$.releaseBoundary.infersPersonhood"),
    ranksPersons: literal(release.ranksPersons, false, "$.releaseBoundary.ranksPersons"),
    createsKarmaEvent: literal(
      release.createsKarmaEvent,
      false,
      "$.releaseBoundary.createsKarmaEvent",
    ),
    createsKarmaMagnitude: literal(
      release.createsKarmaMagnitude,
      false,
      "$.releaseBoundary.createsKarmaMagnitude",
    ),
    grantsQualification: literal(
      release.grantsQualification,
      false,
      "$.releaseBoundary.grantsQualification",
    ),
    activatesRewards: literal(
      release.activatesRewards,
      false,
      "$.releaseBoundary.activatesRewards",
    ),
    movesFunds: literal(release.movesFunds, false, "$.releaseBoundary.movesFunds"),
    grantsGovernance: literal(
      release.grantsGovernance,
      false,
      "$.releaseBoundary.grantsGovernance",
    ),
    grantsAuthority: literal(
      release.grantsAuthority,
      false,
      "$.releaseBoundary.grantsAuthority",
    ),
    recordsConsent: literal(release.recordsConsent, false, "$.releaseBoundary.recordsConsent"),
    automaticProtocolOrAuthorityAction: literal(
      release.automaticProtocolOrAuthorityAction,
      false,
      "$.releaseBoundary.automaticProtocolOrAuthorityAction",
    ),
  };

  return {
    schema: literal(root.schema, CORRESPONDENCE_SCHEMA, "$.schema"),
    version: literal(root.version, 0, "$.version"),
    snapshotDate: literal(root.snapshotDate, "2026-08-15", "$.snapshotDate"),
    status: literal(root.status, "READ_ONLY_ZERO_EFFECT", "$.status"),
    title: boundedText(root.title, "$.title", 200),
    summary: boundedText(root.summary, "$.summary"),
    authorityStatement: boundedText(root.authorityStatement, "$.authorityStatement"),
    sourceBindings,
    physicsSources,
    epistemicLanes,
    relationKinds,
    dimensions,
    correspondences,
    energyFirewall,
    dualityGate,
    releaseBoundary,
  };
}

export function parseCorrespondenceGeometryJson(raw: string): CorrespondenceGeometry {
  if (new TextEncoder().encode(raw).byteLength > CORRESPONDENCE_GEOMETRY_MAX_BYTES) {
    fail("$", `exceeds the ${CORRESPONDENCE_GEOMETRY_MAX_BYTES}-byte limit`);
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch (error) {
    fail("$", `is invalid JSON: ${error instanceof Error ? error.message : "unknown error"}`);
  }
  rejectDuplicateKeysAndDepth(raw);
  return parseCorrespondenceGeometry(parsed);
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new CorrespondenceGeometryDataError("Web Crypto SHA-256 is unavailable");
  }
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes as BufferSource);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new CorrespondenceGeometryDataError(
      "Correspondence geometry returned an empty body",
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
      signal.reason ?? new Error("Correspondence geometry request timed out"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > CORRESPONDENCE_GEOMETRY_MAX_BYTES) {
        void reader.cancel().catch(() => undefined);
        throw new CorrespondenceGeometryDataError(
          "Correspondence geometry exceeds its byte limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => undefined);
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry request timed out",
      );
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A hostile pending read is abandoned after refusal.
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

export async function fetchCorrespondenceGeometry(
  options: CorrespondenceGeometryFetchOptions = {},
): Promise<CorrespondenceGeometry> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (fetcher === undefined) {
    throw new CorrespondenceGeometryDataError("Correspondence geometry is unavailable");
  }
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? "https://zerone.ai/" : window.location.href);
  const expectedUrl = new URL(CORRESPONDENCE_GEOMETRY_ENDPOINT, baseUrl);
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new Error("Correspondence geometry request timed out")),
    options.timeoutMs ?? 8_000,
  );
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        controller.signal.reason ?? new Error("Correspondence geometry request timed out"),
      );
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(CORRESPONDENCE_GEOMETRY_ENDPOINT, {
          cache: "no-store",
          credentials: "omit",
          headers: { Accept: "application/json" },
          redirect: "error",
          referrerPolicy: "no-referrer",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch {
      if (controller.signal.aborted) {
        throw new CorrespondenceGeometryDataError(
          "Correspondence geometry request timed out",
        );
      }
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry is unavailable",
      );
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      throw new CorrespondenceGeometryDataError(
        `Correspondence geometry returned HTTP ${response.status}`,
      );
    }
    if (response.redirected) {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry response was redirected",
      );
    }
    let actualUrl: URL;
    try {
      actualUrl = new URL(response.url);
    } catch {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry returned an invalid final URL",
      );
    }
    if (actualUrl.href !== expectedUrl.href || actualUrl.origin !== expectedUrl.origin) {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry left its exact same-origin path",
      );
    }
    const mediaType =
      response.headers.get("content-type")?.split(";", 1)[0]?.trim().toLowerCase() ??
      "";
    if (mediaType !== "application/json") {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry returned a non-application/json response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null) {
      const length = Number(declaredLength);
      if (
        !/^\d+$/u.test(declaredLength) ||
        !Number.isSafeInteger(length) ||
        length > CORRESPONDENCE_GEOMETRY_MAX_BYTES
      ) {
        throw new CorrespondenceGeometryDataError(
          "Correspondence geometry exceeds its byte limit",
        );
      }
    }
    const bytes = await readBoundedResponse(response, controller.signal);
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry is not valid UTF-8",
      );
    }
    if ((await sha256Hex(bytes)) !== CORRESPONDENCE_GEOMETRY_SHA256) {
      throw new CorrespondenceGeometryDataError(
        "Correspondence geometry bytes do not match the reviewed SHA-256",
      );
    }
    return parseCorrespondenceGeometryJson(raw);
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

function sourceUrl(path: string): string {
  return `https://github.com/cambridgetcg/zerone-core/blob/7aef7ad945b6c5d4d31484342da36da567a47075/${path}`;
}

function detailBlock(
  title: string,
  values: readonly string[],
  modifier?: "is-loss" | "is-refusal",
): HTMLElement {
  const block = el(
    "div",
    `correspondence-geometry-map-detail${modifier === undefined ? "" : ` ${modifier}`}`,
  );
  block.append(el("strong", undefined, title));
  const list = el("ul");
  for (const value of values) {
    const item = el("li", undefined, value);
    list.append(item);
  }
  block.append(list);
  return block;
}

export function renderCorrespondenceGeometry(
  root: HTMLElement,
  geometry: CorrespondenceGeometry,
): void {
  const shell = el("div", "correspondence-geometry-shell");

  const noEffects = el("section", "correspondence-geometry-no-effects");
  const effectIntroduction = el("div");
  effectIntroduction.append(
    el(
      "span",
      "correspondence-geometry-kicker",
      "Release boundary · every value false",
    ),
    el("h3", undefined, "No protocol or authority effect is activated."),
    el("p", undefined, geometry.authorityStatement),
    el(
      "p",
      undefined,
      "The browser automatically performs one digest-pinned same-origin read and renders it locally. This artifact performs no network write, chain write, protocol or authority action, reward action, or consent action.",
    ),
  );
  const effectList = el("ul");
  effectList.setAttribute("aria-label", "Twenty disabled effects");
  for (const [name, value] of Object.entries(geometry.releaseBoundary)) {
    const item = el("li");
    item.append(el("code", undefined, name), el("b", undefined, String(value)));
    effectList.append(item);
  }
  noEffects.append(effectIntroduction, effectList);

  const laneSection = el("section", "correspondence-geometry-lanes");
  laneSection.append(el("h3", undefined, "Five epistemic lanes stay separate"));
  const laneGrid = el("div", "correspondence-geometry-lane-grid");
  for (const lane of geometry.epistemicLanes) {
    const card = el("article", "correspondence-geometry-lane");
    card.dataset.lane = lane.id;
    card.append(
      el("span", undefined, lane.id.replaceAll("_", " ")),
      el("h4", undefined, lane.name),
      el("p", undefined, lane.meaning),
      el("small", undefined, lane.warrantRule),
    );
    laneGrid.append(card);
  }
  laneSection.append(laneGrid);

  const mappingSection = el("section", "correspondence-geometry-mappings");
  mappingSection.append(
    el("h3", undefined, "Inspect the mapping, the loss, and the refusal"),
  );
  const controls = el("div", "correspondence-geometry-controls");
  controls.setAttribute("role", "group");
  controls.setAttribute("aria-label", "Filter correspondence mappings by dimension");
  const filterButtons = new Map<string, HTMLButtonElement>();
  const mappingGrid = el("div", "correspondence-geometry-mapping-grid");
  const mappingCards: Array<{
    dimension: CorrespondenceDimensionId;
    card: HTMLElement;
  }> = [];
  const visibleCount = el("span", "correspondence-geometry-visible-count");
  visibleCount.setAttribute("aria-live", "polite");

  const selectDimension = (dimension: CorrespondenceDimensionId | null): void => {
    let shown = 0;
    for (const [idValue, button] of filterButtons) {
      const active = idValue === (dimension ?? "all");
      button.setAttribute("aria-pressed", String(active));
      button.classList.toggle("is-active", active);
    }
    for (const entry of mappingCards) {
      const visible = dimension === null || entry.dimension === dimension;
      entry.card.hidden = !visible;
      if (visible) shown += 1;
    }
    visibleCount.textContent = `${shown} mapping${shown === 1 ? "" : "s"}`;
  };

  const addFilter = (idValue: "all" | CorrespondenceDimensionId, label: string): void => {
    const button = el("button", "correspondence-geometry-filter", label);
    button.type = "button";
    button.dataset.dimension = idValue;
    button.setAttribute("aria-pressed", "false");
    button.addEventListener("click", () => {
      selectDimension(idValue === "all" ? null : idValue);
    });
    filterButtons.set(idValue, button);
    controls.append(button);
  };
  addFilter("all", "All");
  for (const dimension of geometry.dimensions) addFilter(dimension.id, dimension.name);
  controls.append(visibleCount);

  for (const mapping of geometry.correspondences) {
    const card = el("article", "correspondence-geometry-mapping");
    card.dataset.relationKind = mapping.relationKind;
    card.dataset.dimension = mapping.dimension;
    const heading = el("div", "correspondence-geometry-mapping-heading");
    heading.append(
      el(
        "span",
        "correspondence-geometry-kicker",
        `${mapping.dimension.toUpperCase()} · ${mapping.assessment.replaceAll("_", " ")}`,
      ),
      el("h4", "correspondence-geometry-mapping-title", mapping.title),
    );
    const badges = el("div", "correspondence-geometry-badges");
    const kindBadge = el(
      "span",
      "correspondence-geometry-badge",
      mapping.relationKind.replaceAll("_", " "),
    );
    kindBadge.dataset.kind = mapping.relationKind;
    badges.append(
      kindBadge,
      el("span", "correspondence-geometry-badge", mapping.sourceLane.replaceAll("_", " ")),
      el("span", "correspondence-geometry-badge", mapping.targetLane.replaceAll("_", " ")),
    );
    const route = el("div", "correspondence-geometry-route");
    route.append(
      el("p", "correspondence-geometry-source-node", mapping.source),
      el("span", "correspondence-geometry-arrow", "→"),
      el("p", "correspondence-geometry-target-node", mapping.target),
    );
    const mapDetails = el("div", "correspondence-geometry-map-details");
    mapDetails.append(
      detailBlock("Forward map", [mapping.forwardMap]),
      detailBlock(
        "Preserved invariants",
        mapping.preservedInvariants.map(
          (invariant) =>
            `${invariant.mode}: ${invariant.statement} Test: ${invariant.test}`,
        ),
      ),
      detailBlock(
        "Declared information loss",
        mapping.informationLosses.map(
          (loss) => `${loss.scope}: ${loss.statement} Recoverability: ${loss.recoverability}.`,
        ),
        "is-loss",
      ),
      detailBlock(
        "Must not transfer",
        mapping.nonTransfers.map(
          (wall) => `${wall.category}: ${wall.prohibitedConclusion} ${wall.reason}`,
        ),
        "is-refusal",
      ),
    );
    if (mapping.inverseMap !== null) {
      mapDetails.append(detailBlock("Proposed inverse", [mapping.inverseMap]));
    }
    const tests = el("div", "correspondence-geometry-tests");
    tests.append(el("strong", undefined, "Round-trip status"));
    if (mapping.roundTripTests.length === 0) {
      tests.append(
        el(
          "p",
          undefined,
          "No round trip is claimed for this one-way mapping.",
        ),
      );
    }
    for (const test of mapping.roundTripTests) {
      const row = el("div", "correspondence-geometry-test");
      const status = el("span", "correspondence-geometry-test-status", test.observed);
      status.dataset.status = test.observed;
      row.append(
        status,
        el(
          "p",
          undefined,
          `${test.direction.replaceAll("_", " ")} · expected ${test.expected.replaceAll("_", " ")}. ${test.statement}`,
        ),
      );
      tests.append(row);
    }
    const counterexamples = detailBlock(
      "Counterexamples",
      mapping.counterexamples.map(
        (counterexample) =>
          `${counterexample.setup} Failure: ${counterexample.failureSignal} Outcome: ${counterexample.outcome}.`,
      ),
      "is-refusal",
    );
    const provenance = el(
      "small",
      "correspondence-geometry-provenance",
      `Sources: ${mapping.sourceRefs.join(", ")}. Frameworks: ${mapping.background.frameworks.join("; ")}. Assumptions: ${mapping.background.assumptions.join("; ")}. Valid scope: ${mapping.background.validScope.join("; ")}. Excluded: ${mapping.background.excludedScope.join("; ")}. Equivalence scope: ${mapping.equivalenceScope ?? "none"}.`,
    );
    card.append(heading, badges, route, mapDetails, tests, counterexamples, provenance);
    mappingGrid.append(card);
    mappingCards.push({ dimension: mapping.dimension, card });
  }
  mappingSection.append(controls, mappingGrid);
  selectDimension(null);

  const guardGrid = el("div", "correspondence-geometry-guard-grid");
  for (const dimension of geometry.dimensions) {
    const guard = el("div", "correspondence-geometry-guard-card");
    guard.append(
      el("strong", undefined, dimension.name),
      el("p", undefined, dimension.question),
      el("small", undefined, dimension.guard),
    );
    guardGrid.append(guard);
  }

  const firewall = el("section", "correspondence-geometry-energy-firewall");
  firewall.append(
    el("h3", undefined, "Energy firewall: no implicit conversion"),
    el(
      "p",
      undefined,
      "Physical, compute, protocol, economic, lived, and spiritual-poetic language retain distinct measures or registers. A balance or report establishes neither truth, authority, person-worth, nor a penalty for rest.",
    ),
  );
  const energyList = el("div", "correspondence-geometry-source-grid");
  for (const lane of geometry.energyFirewall.lanes) {
    const card = el("article", "correspondence-geometry-source");
    card.append(
      el("strong", undefined, lane.id.replaceAll("_", " ")),
      el("p", undefined, lane.meaning),
      el("code", undefined, lane.measureOrRegister),
      el("small", undefined, "May convert to: none."),
    );
    energyList.append(card);
  }
  firewall.append(energyList);

  const duality = el("section", "correspondence-geometry-duality-gate");
  duality.append(
    el("h3", undefined, "Duality gate: no equivalence claimed"),
    el(
      "p",
      undefined,
      `${geometry.dualityGate.reviewedCandidateCount} reviewed candidates; ${geometry.dualityGate.acceptedCandidateCount} accepted. Passing this machine-shape gate would still create only a candidate for deeper review, not prove mathematical, physical, or ontological equivalence.`,
    ),
  );
  const dualityRequirements = el("ul");
  for (const requirement of geometry.dualityGate.requirements) {
    dualityRequirements.append(el("li", undefined, requirement));
  }
  duality.append(dualityRequirements);

  const sources = el("section", "correspondence-geometry-sources");
  sources.append(el("h3", undefined, "Pinned local sources and scoped physics papers"));
  const sourceGrid = el("div", "correspondence-geometry-source-grid");
  for (const binding of geometry.sourceBindings) {
    const card = el("article", "correspondence-geometry-source");
    const link = el("a", undefined, binding.path);
    link.href = sourceUrl(binding.path);
    link.target = "_blank";
    link.rel = "noreferrer";
    card.append(
      el("strong", undefined, binding.role.replaceAll("_", " ")),
      link,
      el("code", undefined, `sha256:${binding.sha256}`),
    );
    sourceGrid.append(card);
  }
  for (const source of geometry.physicsSources) {
    const card = el("article", "correspondence-geometry-source");
    const link = el("a", undefined, source.title);
    link.href = source.url;
    link.target = "_blank";
    link.rel = "noreferrer";
    card.append(
      el("strong", undefined, source.id.replaceAll("-", " ")),
      link,
      el("p", undefined, source.scope),
      el("small", undefined, source.doesNotWarrant),
    );
    sourceGrid.append(card);
  }
  sources.append(sourceGrid);

  const footer = el("div", "correspondence-geometry-footer");
  const footerActions = el("div");
  const reset = el("button", "correspondence-geometry-reset", "Show all dimensions");
  reset.type = "button";
  reset.addEventListener("click", () => selectDimension(null));
  const raw = el("a", "correspondence-geometry-raw", "Open raw JSON ↗");
  raw.href = CORRESPONDENCE_GEOMETRY_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  footerActions.append(reset, raw);
  footer.append(
    el(
      "p",
      undefined,
      `Digest-pinned ${geometry.status.replaceAll("_", " ").toLowerCase()} publication · ${geometry.snapshotDate}.`,
    ),
    footerActions,
  );

  shell.append(
    noEffects,
    laneSection,
    mappingSection,
    guardGrid,
    firewall,
    duality,
    sources,
    footer,
  );
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseCorrespondenceGeometry(
  root: HTMLElement,
  options: CorrespondenceGeometryFetchOptions = {},
): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderCorrespondenceGeometry(root, await fetchCorrespondenceGeometry(options));
  } catch (error) {
    const failure = el("div", "correspondence-geometry-load-error");
    failure.setAttribute("role", "alert");
    const actions = el("div", "correspondence-geometry-load-actions");
    const raw = el("a", "correspondence-geometry-raw", "Open raw static atlas ↗");
    raw.href = CORRESPONDENCE_GEOMETRY_ENDPOINT;
    raw.target = "_blank";
    raw.rel = "noreferrer";
    actions.append(raw);
    failure.append(
      el("strong", undefined, "The correspondence geometry could not be verified."),
      el(
        "p",
        undefined,
        error instanceof Error
          ? error.message
          : "The sealed static correspondence atlas is unavailable.",
      ),
      el(
        "p",
        undefined,
        "No correspondence, equivalence, scientific, theological, protocol, authority, reward, or consent conclusion was accepted.",
      ),
      actions,
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
