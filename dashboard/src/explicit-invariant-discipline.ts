export const EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT =
  "/standards/explicit-invariant-discipline.v1.json";
export const EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES = 262_144;
export const EXPLICIT_INVARIANT_DISCIPLINE_TIMEOUT_MS = 8_000;
export const EXPLICIT_INVARIANT_DISCIPLINE_SHA256 =
  "e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df";

const EXPLICIT_INVARIANT_DISCIPLINE_SCHEMA =
  "zerone.explicit-invariant-discipline/v1";
const SAFE_ID = /^[a-z][a-z0-9-]{0,95}$/u;
const SAFE_REFERENCE_ID = /^[a-z][a-z0-9.-]{0,95}$/u;
const SAFE_REPOSITORY_PATH = /^[A-Za-z0-9._/-]+$/u;
const SHA256 = /^[0-9a-f]{64}$/u;
const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/u;
const BIDI_CONTROLS = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/u;
const UNSAFE_CONTROLS = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u;

type JsonObject = Record<string, unknown>;

export type ExplicitInvariantResultKind =
  | "FAMILY"
  | "NO_GO"
  | "CONDITIONAL_UNIQUENESS";

export interface ExplicitInvariantCanonicalVocabulary {
  claimOwners: string[];
  candidateKinds: string[];
  candidateCompleteness: string[];
  assumptionKinds: string[];
  invariantModes: string[];
  constraintKinds: string[];
  witnessMethods: string[];
  witnessOutcomes: string[];
  falsifierStatuses: string[];
  counterexampleDispositions: string[];
  resultKinds: ExplicitInvariantResultKind[];
  familyCardinalities: string[];
  boundaryTermTreatments: string[];
  relationKinds: string[];
  assessments: string[];
  integrationStatuses: string[];
}

export interface ExplicitInvariantSourceBinding {
  id: string;
  path: string;
  rawSha256: string;
  role: string;
  boundary: string;
}

export interface ExplicitInvariantPrimarySource {
  id: string;
  title: string;
  authors: string[];
  locator: string;
  version: string;
  versionDate: string;
  retrievedDate: string;
  role: string;
  boundary: string;
}

export interface ExplicitInvariantIntegrationTarget {
  id: string;
  label: string;
  status: "CURRENT_STATIC_REFERENCE" | "NOT_IMPLEMENTED";
  currentBinding: string;
  futureBoundary: string;
  sourceRefs: string[];
}

export interface ExplicitInvariantCandidateClass {
  id: string;
  kind: "SCATTERING_AMPLITUDE_FAMILY";
  definition: string;
  membershipRule: string;
  completeness: "ENUMERATED" | "PARAMETERISED" | "BOUNDED_SEARCH" | "OPEN";
}

export interface ExplicitInvariantRegime {
  formalSystem: string;
  background: string;
  parameterDomain: string;
  approximationOrder: string;
  validityScope: string;
  excludedScope: string;
}

export interface ExplicitInvariantAssumption {
  id: string;
  kind: string;
  statement: string;
  sourceRefs: string[];
}

export interface ExplicitInvariantInvariant {
  id: string;
  statement: string;
  scope: string;
  mode: "EXACT" | "TOLERANCED" | "STRUCTURAL" | "ORDERING";
  tolerance: string | null;
  witnessIds: string[];
}

export interface ExplicitInvariantConstraintWitness {
  id: string;
  kind: string;
  targetRefs: string[];
  method: string;
  procedure: string;
  artifactRefs: string[];
  outcome: "PASS" | "FAIL" | "NOT_RUN" | "INCONCLUSIVE";
  statement: string;
}

export interface ExplicitInvariantFalsifier {
  id: string;
  targetRefs: string[];
  condition: string;
  procedure: string;
  status: "NOT_RUN" | "SURVIVED" | "TRIGGERED" | "INCONCLUSIVE";
  witnessRef: string | null;
}

export interface ExplicitInvariantCounterexample {
  id: string;
  targetRefs: string[];
  member: string;
  explanation: string;
  disposition:
    | "IN_SCOPE_BREAKS_RESULT"
    | "OUTSIDE_REGIME"
    | "RELAXES_ASSUMPTION"
    | "REJECTED";
  relaxationBranchRefs: string[];
  sourceRefs: string[];
}

export interface ExplicitInvariantResult {
  kind: ExplicitInvariantResultKind;
  statement: string;
  underAssumptionIds: string[];
  underInvariantIds: string[];
  witnessIds: string[];
}

export interface ExplicitInvariantRelaxationBranch {
  id: string;
  statement: string;
  relaxedAssumptionIds: string[];
}

export interface ExplicitInvariantRemainingFamily {
  cardinality: "NONE" | "ONE" | "MANY" | "UNKNOWN";
  description: string;
  parameters: string[];
  knownMembers: string[];
  relaxationBranches: ExplicitInvariantRelaxationBranch[];
}

export interface ExplicitInvariantBoundaryTerm {
  id: string;
  term: string;
  origin: string;
  treatment:
    | "RETAINED"
    | "VANISHES_BY_ASSUMPTION"
    | "BOUNDED"
    | "NEGLECTED"
    | "UNKNOWN";
  justification: string;
  affectedResult: string;
  underAssumptionIds: string[];
  underInvariantIds: string[];
  witnessIds: string[];
}

export interface ExplicitInvariantSourceResult {
  claimOwner: "SOURCE_AUTHORS";
  domain: "PHYSICS_MATH";
  candidateClass: ExplicitInvariantCandidateClass;
  regime: ExplicitInvariantRegime;
  assumptions: ExplicitInvariantAssumption[];
  invariants: ExplicitInvariantInvariant[];
  constraintWitnesses: ExplicitInvariantConstraintWitness[];
  falsifiers: ExplicitInvariantFalsifier[];
  counterexamples: ExplicitInvariantCounterexample[];
  result: ExplicitInvariantResult;
  remainingFamily: ExplicitInvariantRemainingFamily;
  boundaryTerms: ExplicitInvariantBoundaryTerm[];
  allowedConclusion: string;
  limitations: string;
  sourceRefs: string[];
}

export interface ExplicitInvariantLocalTest {
  testId: string;
  status: "NOT_RUN";
  statement: string;
}

export interface ExplicitInvariantZeroneTransfer {
  claimOwner: "ZERONE";
  relationKind: "METHODOLOGICAL_ANALOGY";
  assessment: "PROPOSED";
  target: string;
  preservedDiscipline: string[];
  localTest: ExplicitInvariantLocalTest;
  nonTransfers: string[];
  zeroneRefs: string[];
}

export interface ExplicitInvariantRecord {
  id: string;
  title: string;
  sourceResult: ExplicitInvariantSourceResult;
  zeroneTransfer: ExplicitInvariantZeroneTransfer;
}

export interface ExplicitInvariantBrowserBoundary {
  staticReadCount: 1;
  sameOriginOnly: true;
  externalFetchCount: 0;
  purpose: string;
}

export interface ExplicitInvariantReleaseBoundary {
  claimsAuthorEndorsement: false;
  simulatesPersonReasoning: false;
  claimsUnconditionalUniqueness: false;
  assertsStringOntology: false;
  transfersPhysicsToTheology: false;
  transfersPhysicsToMorality: false;
  changesConsensus: false;
  writesChainState: false;
  performsNetworkWrites: false;
  registersOntology: false;
  assertsScientificTruth: false;
  assertsTheologicalTruth: false;
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

export interface ExplicitInvariantDiscipline {
  schema: string;
  version: number;
  snapshotDate: string;
  status: string;
  title: string;
  summary: string;
  attributionStatement: string;
  authorityStatement: string;
  canonicalVocabulary: ExplicitInvariantCanonicalVocabulary;
  sourceBindings: ExplicitInvariantSourceBinding[];
  primarySources: ExplicitInvariantPrimarySource[];
  integrationTargets: ExplicitInvariantIntegrationTarget[];
  records: ExplicitInvariantRecord[];
  browserBoundary: ExplicitInvariantBrowserBoundary;
  releaseBoundary: ExplicitInvariantReleaseBoundary;
}

export interface ExplicitInvariantDisciplineFetchOptions {
  fetcher?: typeof fetch;
  baseUrl?: string;
  timeoutMs?: number;
}

export class ExplicitInvariantDisciplineDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ExplicitInvariantDisciplineDataError";
  }
}

const TOP_LEVEL_KEYS = [
  "schema",
  "version",
  "snapshotDate",
  "status",
  "title",
  "summary",
  "attributionStatement",
  "authorityStatement",
  "canonicalVocabulary",
  "sourceBindings",
  "primarySources",
  "integrationTargets",
  "records",
  "browserBoundary",
  "releaseBoundary",
] as const;
const VOCABULARY_KEYS = [
  "claimOwners",
  "candidateKinds",
  "candidateCompleteness",
  "assumptionKinds",
  "invariantModes",
  "constraintKinds",
  "witnessMethods",
  "witnessOutcomes",
  "falsifierStatuses",
  "counterexampleDispositions",
  "resultKinds",
  "familyCardinalities",
  "boundaryTermTreatments",
  "relationKinds",
  "assessments",
  "integrationStatuses",
] as const;
const SOURCE_BINDING_KEYS = ["id", "path", "rawSha256", "role", "boundary"] as const;
const PRIMARY_SOURCE_KEYS = [
  "id",
  "title",
  "authors",
  "locator",
  "version",
  "versionDate",
  "retrievedDate",
  "role",
  "boundary",
] as const;
const INTEGRATION_TARGET_KEYS = [
  "id",
  "label",
  "status",
  "currentBinding",
  "futureBoundary",
  "sourceRefs",
] as const;
const RECORD_KEYS = ["id", "title", "sourceResult", "zeroneTransfer"] as const;
const SOURCE_RESULT_KEYS = [
  "claimOwner",
  "domain",
  "candidateClass",
  "regime",
  "assumptions",
  "invariants",
  "constraintWitnesses",
  "falsifiers",
  "counterexamples",
  "result",
  "remainingFamily",
  "boundaryTerms",
  "allowedConclusion",
  "limitations",
  "sourceRefs",
] as const;
const CANDIDATE_KEYS = ["id", "kind", "definition", "membershipRule", "completeness"] as const;
const REGIME_KEYS = [
  "formalSystem",
  "background",
  "parameterDomain",
  "approximationOrder",
  "validityScope",
  "excludedScope",
] as const;
const ASSUMPTION_KEYS = ["id", "kind", "statement", "sourceRefs"] as const;
const INVARIANT_KEYS = ["id", "statement", "scope", "mode", "tolerance", "witnessIds"] as const;
const WITNESS_KEYS = [
  "id",
  "kind",
  "targetRefs",
  "method",
  "procedure",
  "artifactRefs",
  "outcome",
  "statement",
] as const;
const FALSIFIER_KEYS = ["id", "targetRefs", "condition", "procedure", "status", "witnessRef"] as const;
const COUNTEREXAMPLE_KEYS = [
  "id",
  "targetRefs",
  "member",
  "explanation",
  "disposition",
  "relaxationBranchRefs",
  "sourceRefs",
] as const;
const RESULT_KEYS = ["kind", "statement", "underAssumptionIds", "underInvariantIds", "witnessIds"] as const;
const REMAINING_FAMILY_KEYS = [
  "cardinality",
  "description",
  "parameters",
  "knownMembers",
  "relaxationBranches",
] as const;
const RELAXATION_BRANCH_KEYS = ["id", "statement", "relaxedAssumptionIds"] as const;
const BOUNDARY_TERM_KEYS = [
  "id",
  "term",
  "origin",
  "treatment",
  "justification",
  "affectedResult",
  "underAssumptionIds",
  "underInvariantIds",
  "witnessIds",
] as const;
const TRANSFER_KEYS = [
  "claimOwner",
  "relationKind",
  "assessment",
  "target",
  "preservedDiscipline",
  "localTest",
  "nonTransfers",
  "zeroneRefs",
] as const;
const LOCAL_TEST_KEYS = ["testId", "status", "statement"] as const;
const BROWSER_BOUNDARY_KEYS = [
  "staticReadCount",
  "sameOriginOnly",
  "externalFetchCount",
  "purpose",
] as const;
const RELEASE_BOUNDARY_KEYS = [
  "claimsAuthorEndorsement",
  "simulatesPersonReasoning",
  "claimsUnconditionalUniqueness",
  "assertsStringOntology",
  "transfersPhysicsToTheology",
  "transfersPhysicsToMorality",
  "changesConsensus",
  "writesChainState",
  "performsNetworkWrites",
  "registersOntology",
  "assertsScientificTruth",
  "assertsTheologicalTruth",
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

const EXPECTED_VOCABULARY = {
  claimOwners: ["SOURCE_AUTHORS", "ZERONE"],
  candidateKinds: ["SCATTERING_AMPLITUDE_FAMILY"],
  candidateCompleteness: ["ENUMERATED", "PARAMETERISED", "BOUNDED_SEARCH", "OPEN"],
  assumptionKinds: [
    "DEFINITIONAL",
    "STRUCTURAL",
    "DYNAMICAL",
    "APPROXIMATION",
    "EMPIRICAL_INPUT",
    "COMPUTATIONAL_BOUND",
    "INTERPRETIVE_CHOICE",
  ],
  invariantModes: ["EXACT", "TOLERANCED", "STRUCTURAL", "ORDERING"],
  constraintKinds: ["EXISTENCE", "EXCLUSION", "PRESERVATION", "COMPLETENESS", "BOUND"],
  witnessMethods: [
    "FORMAL_DERIVATION",
    "SYMBOLIC_COMPUTATION",
    "NUMERICAL_COMPUTATION",
    "EMPIRICAL_OBSERVATION",
    "SOURCE_ANALYSIS",
  ],
  witnessOutcomes: ["PASS", "FAIL", "NOT_RUN", "INCONCLUSIVE"],
  falsifierStatuses: ["NOT_RUN", "SURVIVED", "TRIGGERED", "INCONCLUSIVE"],
  counterexampleDispositions: [
    "IN_SCOPE_BREAKS_RESULT",
    "OUTSIDE_REGIME",
    "RELAXES_ASSUMPTION",
    "REJECTED",
  ],
  resultKinds: ["FAMILY", "NO_GO", "CONDITIONAL_UNIQUENESS"],
  familyCardinalities: ["NONE", "ONE", "MANY", "UNKNOWN"],
  boundaryTermTreatments: [
    "RETAINED",
    "VANISHES_BY_ASSUMPTION",
    "BOUNDED",
    "NEGLECTED",
    "UNKNOWN",
  ],
  relationKinds: ["METHODOLOGICAL_ANALOGY"],
  assessments: ["PROPOSED"],
  integrationStatuses: ["CURRENT_STATIC_REFERENCE", "NOT_IMPLEMENTED"],
} as const;

function fail(path: string, message: string): never {
  throw new ExplicitInvariantDisciplineDataError(`${path}: ${message}`);
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
  const names = Reflect.ownKeys(value);
  if (names.some((name) => typeof name !== "string")) {
    fail(path, "contains a non-string field");
  }
  const actual = names as string[];
  const wanted = [...expected];
  if (
    actual.length !== wanted.length ||
    actual.some((name, index) => name !== wanted[index])
  ) {
    const unknown = [...actual].sort().filter((name) => !wanted.includes(name));
    const missing = [...wanted].sort().filter((name) => !actual.includes(name));
    fail(
      path,
      `contains unknown, missing, or reordered fields (unknown: ${unknown.join(", ") || "none"}; missing: ${missing.join(", ") || "none"})`,
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

function text(value: unknown, path: string, maximum = 2_048): string {
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
  return value === null ? null : text(value, path, maximum);
}

function id(value: unknown, path: string): string {
  const result = text(value, path, 96);
  if (!SAFE_ID.test(result)) fail(path, "must be a lowercase kebab identifier");
  return result;
}

function referenceId(value: unknown, path: string): string {
  const result = text(value, path, 96);
  if (!SAFE_REFERENCE_ID.test(result)) {
    fail(path, "must be a lowercase reference identifier");
  }
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

function oneOf<T extends string>(
  value: unknown,
  allowed: readonly T[],
  path: string,
): T {
  const result = text(value, path, 96) as T;
  if (!allowed.includes(result)) fail(path, "has an unsupported value");
  return result;
}

function strings(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = false,
): string[] {
  const result = denseArray(value, path, maximum).map((entry, index) =>
    text(entry, `${path}[${index}]`),
  );
  if (!allowEmpty && result.length === 0) fail(path, "must not be empty");
  return result;
}

function sortedIds(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = false,
): string[] {
  const result = denseArray(value, path, maximum).map((entry, index) =>
    id(entry, `${path}[${index}]`),
  );
  if (!allowEmpty && result.length === 0) fail(path, "must not be empty");
  for (let index = 1; index < result.length; index += 1) {
    if ((result[index - 1] ?? "") >= (result[index] ?? "")) {
      fail(path, "must be strictly sorted with no duplicates");
    }
  }
  return result;
}

function uniqueIds(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = false,
): string[] {
  const result = denseArray(value, path, maximum).map((entry, index) =>
    id(entry, `${path}[${index}]`),
  );
  if (!allowEmpty && result.length === 0) fail(path, "must not be empty");
  if (new Set(result).size !== result.length) fail(path, "must not contain duplicates");
  return result;
}

function sortedReferenceIds(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = false,
): string[] {
  const result = denseArray(value, path, maximum).map((entry, index) =>
    referenceId(entry, `${path}[${index}]`),
  );
  if (!allowEmpty && result.length === 0) fail(path, "must not be empty");
  for (let index = 1; index < result.length; index += 1) {
    if ((result[index - 1] ?? "") >= (result[index] ?? "")) {
      fail(path, "must be strictly sorted with no duplicates");
    }
  }
  return result;
}

function ensureSortedById<T extends { id: string }>(items: readonly T[], path: string): void {
  for (let index = 1; index < items.length; index += 1) {
    if ((items[index - 1]?.id ?? "") >= (items[index]?.id ?? "")) {
      fail(path, "must be strictly sorted by id with no duplicates");
    }
  }
}

function safeRepositoryPath(value: unknown, path: string): string {
  const result = text(value, path, 240);
  if (
    !SAFE_REPOSITORY_PATH.test(result) ||
    result.startsWith("/") ||
    result.includes("//") ||
    result.split("/").some((segment) => segment === "." || segment === "..")
  ) {
    fail(path, "must be a safe repository-relative path");
  }
  return result;
}

function digest(value: unknown, path: string): string {
  const result = text(value, path, 64);
  if (!SHA256.test(result)) fail(path, "must be lowercase SHA-256 hex");
  return result;
}

function isoDate(value: unknown, path: string): string {
  const result = text(value, path, 10);
  if (!ISO_DATE.test(result) || Number.isNaN(Date.parse(`${result}T00:00:00Z`))) {
    fail(path, "must be an ISO calendar date");
  }
  return result;
}

function isoInstant(value: unknown, path: string): string {
  const result = text(value, path, 32);
  if (
    !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u.test(result) ||
    Number.isNaN(Date.parse(result))
  ) {
    fail(path, "must be a canonical UTC second-precision timestamp");
  }
  return result;
}

function canonicalHttpsUrl(value: unknown, path: string): string {
  const result = text(value, path, 512);
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
    url.search !== "" ||
    url.hash !== "" ||
    url.href !== result
  ) {
    fail(path, "must be a canonical credential-free HTTPS URL without query or fragment");
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
        try {
          return JSON.parse(raw.slice(start, offset)) as string;
        } catch {
          fail("$", "contains a malformed JSON string");
        }
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

function exactStringSequence(
  value: unknown,
  expected: readonly string[],
  path: string,
): string[] {
  const result = strings(value, path, expected.length);
  if (
    result.length !== expected.length ||
    result.some((entry, index) => entry !== expected[index])
  ) {
    fail(path, "must match the reviewed vocabulary exactly and in order");
  }
  return result;
}

function requireReferences(
  refs: readonly string[],
  allowed: ReadonlySet<string>,
  path: string,
): void {
  for (const ref of refs) {
    if (!allowed.has(ref)) fail(path, `contains unresolved reference ${ref}`);
  }
}

function parseVocabulary(value: unknown): ExplicitInvariantCanonicalVocabulary {
  const vocabulary = object(value, "$.canonicalVocabulary");
  exactKeys(vocabulary, VOCABULARY_KEYS, "$.canonicalVocabulary");
  return {
    claimOwners: exactStringSequence(
      vocabulary.claimOwners,
      EXPECTED_VOCABULARY.claimOwners,
      "$.canonicalVocabulary.claimOwners",
    ),
    candidateKinds: exactStringSequence(
      vocabulary.candidateKinds,
      EXPECTED_VOCABULARY.candidateKinds,
      "$.canonicalVocabulary.candidateKinds",
    ),
    candidateCompleteness: exactStringSequence(
      vocabulary.candidateCompleteness,
      EXPECTED_VOCABULARY.candidateCompleteness,
      "$.canonicalVocabulary.candidateCompleteness",
    ),
    assumptionKinds: exactStringSequence(
      vocabulary.assumptionKinds,
      EXPECTED_VOCABULARY.assumptionKinds,
      "$.canonicalVocabulary.assumptionKinds",
    ),
    invariantModes: exactStringSequence(
      vocabulary.invariantModes,
      EXPECTED_VOCABULARY.invariantModes,
      "$.canonicalVocabulary.invariantModes",
    ),
    constraintKinds: exactStringSequence(
      vocabulary.constraintKinds,
      EXPECTED_VOCABULARY.constraintKinds,
      "$.canonicalVocabulary.constraintKinds",
    ),
    witnessMethods: exactStringSequence(
      vocabulary.witnessMethods,
      EXPECTED_VOCABULARY.witnessMethods,
      "$.canonicalVocabulary.witnessMethods",
    ),
    witnessOutcomes: exactStringSequence(
      vocabulary.witnessOutcomes,
      EXPECTED_VOCABULARY.witnessOutcomes,
      "$.canonicalVocabulary.witnessOutcomes",
    ),
    falsifierStatuses: exactStringSequence(
      vocabulary.falsifierStatuses,
      EXPECTED_VOCABULARY.falsifierStatuses,
      "$.canonicalVocabulary.falsifierStatuses",
    ),
    counterexampleDispositions: exactStringSequence(
      vocabulary.counterexampleDispositions,
      EXPECTED_VOCABULARY.counterexampleDispositions,
      "$.canonicalVocabulary.counterexampleDispositions",
    ),
    resultKinds: exactStringSequence(
      vocabulary.resultKinds,
      EXPECTED_VOCABULARY.resultKinds,
      "$.canonicalVocabulary.resultKinds",
    ) as ExplicitInvariantResultKind[],
    familyCardinalities: exactStringSequence(
      vocabulary.familyCardinalities,
      EXPECTED_VOCABULARY.familyCardinalities,
      "$.canonicalVocabulary.familyCardinalities",
    ),
    boundaryTermTreatments: exactStringSequence(
      vocabulary.boundaryTermTreatments,
      EXPECTED_VOCABULARY.boundaryTermTreatments,
      "$.canonicalVocabulary.boundaryTermTreatments",
    ),
    relationKinds: exactStringSequence(
      vocabulary.relationKinds,
      EXPECTED_VOCABULARY.relationKinds,
      "$.canonicalVocabulary.relationKinds",
    ),
    assessments: exactStringSequence(
      vocabulary.assessments,
      EXPECTED_VOCABULARY.assessments,
      "$.canonicalVocabulary.assessments",
    ),
    integrationStatuses: exactStringSequence(
      vocabulary.integrationStatuses,
      EXPECTED_VOCABULARY.integrationStatuses,
      "$.canonicalVocabulary.integrationStatuses",
    ),
  };
}

function parseSourceBindings(value: unknown): ExplicitInvariantSourceBinding[] {
  const bindings = denseArray(value, "$.sourceBindings", 16).map((entry, index) => {
    const path = `$.sourceBindings[${index}]`;
    const binding = object(entry, path);
    exactKeys(binding, SOURCE_BINDING_KEYS, path);
    return {
      id: id(binding.id, `${path}.id`),
      path: safeRepositoryPath(binding.path, `${path}.path`),
      rawSha256: digest(binding.rawSha256, `${path}.rawSha256`),
      role: text(binding.role, `${path}.role`, 160),
      boundary: text(binding.boundary, `${path}.boundary`),
    };
  });
  if (bindings.length === 0) fail("$.sourceBindings", "must not be empty");
  ensureSortedById(bindings, "$.sourceBindings");
  const paths = new Set(bindings.map((binding) => binding.path));
  if (paths.size !== bindings.length) fail("$.sourceBindings", "contains duplicate paths");
  return bindings;
}

function parsePrimarySources(value: unknown): ExplicitInvariantPrimarySource[] {
  const sources = denseArray(value, "$.primarySources", 16).map((entry, index) => {
    const path = `$.primarySources[${index}]`;
    const source = object(entry, path);
    exactKeys(source, PRIMARY_SOURCE_KEYS, path);
    const versionDate = isoInstant(source.versionDate, `${path}.versionDate`);
    const retrievedDate = isoDate(source.retrievedDate, `${path}.retrievedDate`);
    if (versionDate.slice(0, 10) > retrievedDate) {
      fail(path, "cannot be retrieved before its version date");
    }
    return {
      id: referenceId(source.id, `${path}.id`),
      title: text(source.title, `${path}.title`, 240),
      authors: strings(source.authors, `${path}.authors`, 16),
      locator: canonicalHttpsUrl(source.locator, `${path}.locator`),
      version: text(source.version, `${path}.version`, 32),
      versionDate,
      retrievedDate,
      role: text(source.role, `${path}.role`, 200),
      boundary: text(source.boundary, `${path}.boundary`),
    };
  });
  if (sources.length === 0) fail("$.primarySources", "must not be empty");
  ensureSortedById(sources, "$.primarySources");
  for (const source of sources) {
    const match = /^https:\/\/arxiv\.org\/abs\/(\d{4}\.\d{4,5})v(\d+)$/u.exec(
      source.locator,
    );
    if (
      match === null ||
      source.id !== `arxiv-${match[1]}v${match[2]}`
    ) {
      fail(`$.primarySources.${source.id}`, "id and version-pinned arXiv locator disagree");
    }
    if (source.version !== `v${match[2]}`) {
      fail(`$.primarySources.${source.id}.version`, "does not match locator version");
    }
  }
  return sources;
}

function parseIntegrationTargets(
  value: unknown,
  sourceIds: ReadonlySet<string>,
): ExplicitInvariantIntegrationTarget[] {
  const targets = denseArray(value, "$.integrationTargets", 16).map((entry, index) => {
    const path = `$.integrationTargets[${index}]`;
    const target = object(entry, path);
    exactKeys(target, INTEGRATION_TARGET_KEYS, path);
    const refs = sortedIds(target.sourceRefs, `${path}.sourceRefs`, 16);
    requireReferences(refs, sourceIds, `${path}.sourceRefs`);
    return {
      id: id(target.id, `${path}.id`),
      label: text(target.label, `${path}.label`, 200),
      status: oneOf(
        target.status,
        ["CURRENT_STATIC_REFERENCE", "NOT_IMPLEMENTED"] as const,
        `${path}.status`,
      ),
      currentBinding: text(target.currentBinding, `${path}.currentBinding`),
      futureBoundary: text(target.futureBoundary, `${path}.futureBoundary`),
      sourceRefs: refs,
    };
  });
  if (targets.length === 0) fail("$.integrationTargets", "must not be empty");
  ensureSortedById(targets, "$.integrationTargets");
  return targets;
}

function parseCandidateClass(value: unknown, path: string): ExplicitInvariantCandidateClass {
  const candidate = object(value, path);
  exactKeys(candidate, CANDIDATE_KEYS, path);
  return {
    id: id(candidate.id, `${path}.id`),
    kind: literal(
      candidate.kind,
      "SCATTERING_AMPLITUDE_FAMILY",
      `${path}.kind`,
    ),
    definition: text(candidate.definition, `${path}.definition`),
    membershipRule: text(candidate.membershipRule, `${path}.membershipRule`),
    completeness: oneOf(
      candidate.completeness,
      ["ENUMERATED", "PARAMETERISED", "BOUNDED_SEARCH", "OPEN"] as const,
      `${path}.completeness`,
    ),
  };
}

function parseRegime(value: unknown, path: string): ExplicitInvariantRegime {
  const regime = object(value, path);
  exactKeys(regime, REGIME_KEYS, path);
  return {
    formalSystem: text(regime.formalSystem, `${path}.formalSystem`),
    background: text(regime.background, `${path}.background`),
    parameterDomain: text(regime.parameterDomain, `${path}.parameterDomain`),
    approximationOrder: text(regime.approximationOrder, `${path}.approximationOrder`),
    validityScope: text(regime.validityScope, `${path}.validityScope`),
    excludedScope: text(regime.excludedScope, `${path}.excludedScope`),
  };
}

function parseAssumptions(
  value: unknown,
  path: string,
  sourceIds: ReadonlySet<string>,
): ExplicitInvariantAssumption[] {
  const assumptions = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const assumption = object(entry, itemPath);
    exactKeys(assumption, ASSUMPTION_KEYS, itemPath);
    const sourceRefs = sortedReferenceIds(
      assumption.sourceRefs,
      `${itemPath}.sourceRefs`,
      16,
    );
    requireReferences(sourceRefs, sourceIds, `${itemPath}.sourceRefs`);
    return {
      id: id(assumption.id, `${itemPath}.id`),
      kind: oneOf(
        assumption.kind,
        EXPECTED_VOCABULARY.assumptionKinds,
        `${itemPath}.kind`,
      ),
      statement: text(assumption.statement, `${itemPath}.statement`),
      sourceRefs,
    };
  });
  if (assumptions.length === 0) fail(path, "must not be empty");
  ensureSortedById(assumptions, path);
  return assumptions;
}

function parseInvariants(value: unknown, path: string): ExplicitInvariantInvariant[] {
  const invariants = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const invariant = object(entry, itemPath);
    exactKeys(invariant, INVARIANT_KEYS, itemPath);
    const mode = oneOf(
      invariant.mode,
      ["EXACT", "TOLERANCED", "STRUCTURAL", "ORDERING"] as const,
      `${itemPath}.mode`,
    );
    const tolerance = nullableText(invariant.tolerance, `${itemPath}.tolerance`, 500);
    if ((mode === "TOLERANCED") !== (tolerance !== null)) {
      fail(`${itemPath}.tolerance`, "must be non-null exactly for TOLERANCED invariants");
    }
    return {
      id: id(invariant.id, `${itemPath}.id`),
      statement: text(invariant.statement, `${itemPath}.statement`),
      scope: text(invariant.scope, `${itemPath}.scope`),
      mode,
      tolerance,
      witnessIds: sortedIds(invariant.witnessIds, `${itemPath}.witnessIds`, 16),
    };
  });
  if (invariants.length === 0) fail(path, "must not be empty");
  ensureSortedById(invariants, path);
  return invariants;
}

function parseWitnesses(
  value: unknown,
  path: string,
  sourceIds: ReadonlySet<string>,
): ExplicitInvariantConstraintWitness[] {
  const witnesses = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const witness = object(entry, itemPath);
    exactKeys(witness, WITNESS_KEYS, itemPath);
    const artifacts = sortedReferenceIds(
      witness.artifactRefs,
      `${itemPath}.artifactRefs`,
      16,
    );
    requireReferences(artifacts, sourceIds, `${itemPath}.artifactRefs`);
    return {
      id: id(witness.id, `${itemPath}.id`),
      kind: oneOf(witness.kind, EXPECTED_VOCABULARY.constraintKinds, `${itemPath}.kind`),
      targetRefs: uniqueIds(witness.targetRefs, `${itemPath}.targetRefs`, 32),
      method: oneOf(witness.method, EXPECTED_VOCABULARY.witnessMethods, `${itemPath}.method`),
      procedure: text(witness.procedure, `${itemPath}.procedure`),
      artifactRefs: artifacts,
      outcome: oneOf(
        witness.outcome,
        ["PASS", "FAIL", "NOT_RUN", "INCONCLUSIVE"] as const,
        `${itemPath}.outcome`,
      ),
      statement: text(witness.statement, `${itemPath}.statement`),
    };
  });
  if (witnesses.length === 0) fail(path, "must not be empty");
  ensureSortedById(witnesses, path);
  return witnesses;
}

function parseFalsifiers(value: unknown, path: string): ExplicitInvariantFalsifier[] {
  const falsifiers = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const falsifier = object(entry, itemPath);
    exactKeys(falsifier, FALSIFIER_KEYS, itemPath);
    return {
      id: id(falsifier.id, `${itemPath}.id`),
      targetRefs: sortedIds(falsifier.targetRefs, `${itemPath}.targetRefs`, 32),
      condition: text(falsifier.condition, `${itemPath}.condition`),
      procedure: text(falsifier.procedure, `${itemPath}.procedure`),
      status: oneOf(
        falsifier.status,
        ["NOT_RUN", "SURVIVED", "TRIGGERED", "INCONCLUSIVE"] as const,
        `${itemPath}.status`,
      ),
      witnessRef:
        falsifier.witnessRef === null
          ? null
          : id(falsifier.witnessRef, `${itemPath}.witnessRef`),
    };
  });
  if (falsifiers.length === 0) fail(path, "must not be empty");
  ensureSortedById(falsifiers, path);
  return falsifiers;
}

function parseCounterexamples(
  value: unknown,
  path: string,
  sourceIds: ReadonlySet<string>,
): ExplicitInvariantCounterexample[] {
  const counterexamples = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const counterexample = object(entry, itemPath);
    exactKeys(counterexample, COUNTEREXAMPLE_KEYS, itemPath);
    const sourceRefs = sortedReferenceIds(
      counterexample.sourceRefs,
      `${itemPath}.sourceRefs`,
      16,
    );
    requireReferences(sourceRefs, sourceIds, `${itemPath}.sourceRefs`);
    return {
      id: id(counterexample.id, `${itemPath}.id`),
      targetRefs: sortedIds(counterexample.targetRefs, `${itemPath}.targetRefs`, 32),
      member: text(counterexample.member, `${itemPath}.member`),
      explanation: text(counterexample.explanation, `${itemPath}.explanation`),
      disposition: oneOf(
        counterexample.disposition,
        [
          "IN_SCOPE_BREAKS_RESULT",
          "OUTSIDE_REGIME",
          "RELAXES_ASSUMPTION",
          "REJECTED",
        ] as const,
        `${itemPath}.disposition`,
      ),
      relaxationBranchRefs: sortedIds(
        counterexample.relaxationBranchRefs,
        `${itemPath}.relaxationBranchRefs`,
        16,
        true,
      ),
      sourceRefs,
    };
  });
  if (counterexamples.length === 0) fail(path, "must not be empty");
  ensureSortedById(counterexamples, path);
  return counterexamples;
}

function parseResult(value: unknown, path: string): ExplicitInvariantResult {
  const result = object(value, path);
  exactKeys(result, RESULT_KEYS, path);
  return {
    kind: oneOf(
      result.kind,
      ["FAMILY", "NO_GO", "CONDITIONAL_UNIQUENESS"] as const,
      `${path}.kind`,
    ),
    statement: text(result.statement, `${path}.statement`),
    underAssumptionIds: sortedIds(
      result.underAssumptionIds,
      `${path}.underAssumptionIds`,
      32,
    ),
    underInvariantIds: sortedIds(
      result.underInvariantIds,
      `${path}.underInvariantIds`,
      32,
    ),
    witnessIds: sortedIds(result.witnessIds, `${path}.witnessIds`, 32),
  };
}

function parseRemainingFamily(
  value: unknown,
  path: string,
): ExplicitInvariantRemainingFamily {
  const family = object(value, path);
  exactKeys(family, REMAINING_FAMILY_KEYS, path);
  const relaxationBranches = denseArray(
    family.relaxationBranches,
    `${path}.relaxationBranches`,
    32,
  ).map((entry, index) => {
    const itemPath = `${path}.relaxationBranches[${index}]`;
    const branch = object(entry, itemPath);
    exactKeys(branch, RELAXATION_BRANCH_KEYS, itemPath);
    return {
      id: id(branch.id, `${itemPath}.id`),
      statement: text(branch.statement, `${itemPath}.statement`),
      relaxedAssumptionIds: sortedIds(
        branch.relaxedAssumptionIds,
        `${itemPath}.relaxedAssumptionIds`,
        32,
      ),
    };
  });
  if (relaxationBranches.length === 0) {
    fail(`${path}.relaxationBranches`, "must not be empty");
  }
  ensureSortedById(relaxationBranches, `${path}.relaxationBranches`);
  return {
    cardinality: oneOf(
      family.cardinality,
      ["NONE", "ONE", "MANY", "UNKNOWN"] as const,
      `${path}.cardinality`,
    ),
    description: text(family.description, `${path}.description`),
    parameters: strings(family.parameters, `${path}.parameters`, 32, true),
    knownMembers: strings(family.knownMembers, `${path}.knownMembers`, 32, true),
    relaxationBranches,
  };
}

function parseBoundaryTerms(
  value: unknown,
  path: string,
): ExplicitInvariantBoundaryTerm[] {
  const terms = denseArray(value, path, 32).map((entry, index) => {
    const itemPath = `${path}[${index}]`;
    const term = object(entry, itemPath);
    exactKeys(term, BOUNDARY_TERM_KEYS, itemPath);
    return {
      id: id(term.id, `${itemPath}.id`),
      term: text(term.term, `${itemPath}.term`),
      origin: text(term.origin, `${itemPath}.origin`),
      treatment: oneOf(
        term.treatment,
        ["RETAINED", "VANISHES_BY_ASSUMPTION", "BOUNDED", "NEGLECTED", "UNKNOWN"] as const,
        `${itemPath}.treatment`,
      ),
      justification: text(term.justification, `${itemPath}.justification`),
      affectedResult: text(term.affectedResult, `${itemPath}.affectedResult`),
      underAssumptionIds: sortedIds(
        term.underAssumptionIds,
        `${itemPath}.underAssumptionIds`,
        32,
      ),
      underInvariantIds: sortedIds(
        term.underInvariantIds,
        `${itemPath}.underInvariantIds`,
        32,
      ),
      witnessIds: sortedIds(
        term.witnessIds,
        `${itemPath}.witnessIds`,
        32,
      ),
    };
  });
  if (terms.length === 0) fail(path, "must not be empty");
  ensureSortedById(terms, path);
  return terms;
}

function validateSourceResultSemantics(
  result: ExplicitInvariantSourceResult,
  path: string,
): void {
  const assumptionIds = new Set(result.assumptions.map((item) => item.id));
  const invariantIds = new Set(result.invariants.map((item) => item.id));
  const witnessIds = new Set(result.constraintWitnesses.map((item) => item.id));
  const witnessTargetIds = new Set([
    result.candidateClass.id,
    ...assumptionIds,
    ...invariantIds,
  ]);
  const diagnosticTargetIds = new Set([...witnessTargetIds, ...witnessIds]);
  const relaxationBranchIds = new Set(
    result.remainingFamily.relaxationBranches.map((branch) => branch.id),
  );
  requireReferences(result.result.underAssumptionIds, assumptionIds, `${path}.result.underAssumptionIds`);
  requireReferences(result.result.underInvariantIds, invariantIds, `${path}.result.underInvariantIds`);
  requireReferences(result.result.witnessIds, witnessIds, `${path}.result.witnessIds`);
  if (
    result.result.underAssumptionIds.length !== assumptionIds.size ||
    [...assumptionIds].some((assumptionId) => !result.result.underAssumptionIds.includes(assumptionId))
  ) {
    fail(`${path}.result.underAssumptionIds`, "must expose every load-bearing assumption");
  }
  if (
    result.result.underInvariantIds.length !== invariantIds.size ||
    [...invariantIds].some(
      (invariantId) => !result.result.underInvariantIds.includes(invariantId),
    )
  ) {
    fail(
      `${path}.result.underInvariantIds`,
      "must expose every load-bearing invariant",
    );
  }
  for (const invariant of result.invariants) {
    requireReferences(invariant.witnessIds, witnessIds, `${path}.invariants.${invariant.id}.witnessIds`);
    for (const witnessId of invariant.witnessIds) {
      const witness = result.constraintWitnesses.find((item) => item.id === witnessId);
      if (witness === undefined || !witness.targetRefs.includes(invariant.id)) {
        fail(
          `${path}.invariants.${invariant.id}.witnessIds`,
          `witness ${witnessId} must explicitly target the invariant`,
        );
      }
    }
  }
  for (const witness of result.constraintWitnesses) {
    requireReferences(
      witness.targetRefs,
      witnessTargetIds,
      `${path}.constraintWitnesses.${witness.id}.targetRefs`,
    );
  }
  for (const branch of result.remainingFamily.relaxationBranches) {
    requireReferences(
      branch.relaxedAssumptionIds,
      assumptionIds,
      `${path}.remainingFamily.relaxationBranches.${branch.id}.relaxedAssumptionIds`,
    );
  }
  for (const term of result.boundaryTerms) {
    requireReferences(
      term.underAssumptionIds,
      assumptionIds,
      `${path}.boundaryTerms.${term.id}.underAssumptionIds`,
    );
    requireReferences(
      term.underInvariantIds,
      invariantIds,
      `${path}.boundaryTerms.${term.id}.underInvariantIds`,
    );
    requireReferences(
      term.witnessIds,
      witnessIds,
      `${path}.boundaryTerms.${term.id}.witnessIds`,
    );
    const strongResult =
      result.result.kind === "NO_GO" ||
      result.result.kind === "CONDITIONAL_UNIQUENESS";
    if (
      strongResult &&
      (term.treatment === "NEGLECTED" || term.treatment === "UNKNOWN")
    ) {
      const allInvariantsToleranced = term.underInvariantIds.every((invariantId) =>
        result.invariants.some(
          (invariant) => invariant.id === invariantId && invariant.mode === "TOLERANCED",
        ),
      );
      if (!allInvariantsToleranced) {
        fail(
          `${path}.boundaryTerms.${term.id}.underInvariantIds`,
          "NEGLECTED or UNKNOWN under a strong result requires every referenced invariant to be TOLERANCED",
        );
      }
      for (const witnessId of term.witnessIds) {
        const witness = result.constraintWitnesses.find((item) => item.id === witnessId);
        if (
          witness?.kind !== "BOUND" ||
          witness.outcome !== "PASS" ||
          !term.underInvariantIds.every((invariantId) =>
            witness.targetRefs.includes(invariantId),
          )
        ) {
          fail(
            `${path}.boundaryTerms.${term.id}.witnessIds`,
            `witness ${witnessId} must be PASS BOUND and target every referenced toleranced invariant`,
          );
        }
      }
    }
  }
  for (const falsifier of result.falsifiers) {
    requireReferences(
      falsifier.targetRefs,
      diagnosticTargetIds,
      `${path}.falsifiers.${falsifier.id}.targetRefs`,
    );
    if (falsifier.witnessRef !== null && !witnessIds.has(falsifier.witnessRef)) {
      fail(`${path}.falsifiers.${falsifier.id}.witnessRef`, "references an unknown witness");
    }
  }
  for (const counterexample of result.counterexamples) {
    requireReferences(
      counterexample.targetRefs,
      diagnosticTargetIds,
      `${path}.counterexamples.${counterexample.id}.targetRefs`,
    );
    requireReferences(
      counterexample.relaxationBranchRefs,
      relaxationBranchIds,
      `${path}.counterexamples.${counterexample.id}.relaxationBranchRefs`,
    );
    if (counterexample.disposition === "RELAXES_ASSUMPTION") {
      const targetedAssumptions = new Set(
        counterexample.targetRefs.filter((target) => assumptionIds.has(target)),
      );
      if (targetedAssumptions.size === 0) {
        fail(
          `${path}.counterexamples.${counterexample.id}.targetRefs`,
          "a RELAXES_ASSUMPTION counterexample must target an assumption",
        );
      }
      if (counterexample.relaxationBranchRefs.length === 0) {
        fail(
          `${path}.counterexamples.${counterexample.id}.relaxationBranchRefs`,
          "a RELAXES_ASSUMPTION counterexample must reference a relaxation branch",
        );
      }
      for (const branchId of counterexample.relaxationBranchRefs) {
        const branch = result.remainingFamily.relaxationBranches.find(
          (item) => item.id === branchId,
        );
        if (
          branch === undefined ||
          !branch.relaxedAssumptionIds.some((assumptionId) =>
            targetedAssumptions.has(assumptionId),
          )
        ) {
          fail(
            `${path}.counterexamples.${counterexample.id}.relaxationBranchRefs`,
            `branch ${branchId} must relax an assumption explicitly targeted by the counterexample`,
          );
        }
      }
    } else if (counterexample.relaxationBranchRefs.length !== 0) {
      fail(
        `${path}.counterexamples.${counterexample.id}.relaxationBranchRefs`,
        "must be empty unless disposition is RELAXES_ASSUMPTION",
      );
    }
  }

  const decisiveWitnesses = result.result.witnessIds.map((witnessId) =>
    result.constraintWitnesses.find((item) => item.id === witnessId),
  );
  if (decisiveWitnesses.some((witness) => witness?.outcome !== "PASS")) {
    fail(`${path}.result.witnessIds`, "every decisive witness must report PASS");
  }
  if (result.falsifiers.some((falsifier) => falsifier.status === "TRIGGERED")) {
    fail(`${path}.falsifiers`, "a triggered falsifier blocks the recorded result");
  }
  if (
    result.counterexamples.some(
      (counterexample) => counterexample.disposition === "IN_SCOPE_BREAKS_RESULT",
    )
  ) {
    fail(`${path}.counterexamples`, "an in-scope breaking counterexample blocks the result");
  }
  switch (result.result.kind) {
    case "FAMILY":
      if (result.remainingFamily.cardinality !== "MANY") {
        fail(`${path}.remainingFamily.cardinality`, "FAMILY requires MANY");
      }
      break;
    case "NO_GO": {
      if (result.remainingFamily.cardinality !== "NONE") {
        fail(`${path}.remainingFamily.cardinality`, "NO_GO requires NONE");
      }
      if (result.candidateClass.completeness === "OPEN") {
        fail(`${path}.candidateClass.completeness`, "OPEN cannot support NO_GO");
      }
      const hasCandidateExclusion = decisiveWitnesses.some(
        (witness) =>
          witness?.kind === "EXCLUSION" &&
          witness.targetRefs.includes(result.candidateClass.id),
      );
      const hasCandidateCompleteness = decisiveWitnesses.some(
        (witness) =>
          witness?.kind === "COMPLETENESS" &&
          witness.targetRefs.includes(result.candidateClass.id),
      );
      if (!hasCandidateExclusion || !hasCandidateCompleteness) {
        fail(
          `${path}.result.witnessIds`,
          "NO_GO requires decisive exclusion and completeness witnesses targeting the candidate class",
        );
      }
      break;
    }
    case "CONDITIONAL_UNIQUENESS": {
      if (result.remainingFamily.cardinality !== "ONE") {
        fail(
          `${path}.remainingFamily.cardinality`,
          "CONDITIONAL_UNIQUENESS requires ONE",
        );
      }
      if (result.remainingFamily.knownMembers.length !== 1) {
        fail(`${path}.remainingFamily.knownMembers`, "must name exactly one survivor");
      }
      if (result.candidateClass.completeness === "OPEN") {
        fail(
          `${path}.candidateClass.completeness`,
          "OPEN cannot support CONDITIONAL_UNIQUENESS",
        );
      }
      if (
        !decisiveWitnesses.some(
          (witness) =>
            witness?.kind === "COMPLETENESS" &&
            witness.targetRefs.includes(result.candidateClass.id),
        )
      ) {
        fail(
          `${path}.result.witnessIds`,
          "CONDITIONAL_UNIQUENESS requires a decisive completeness witness targeting the candidate class",
        );
      }
      const invariantWitnesses = result.invariants.flatMap((invariant) =>
        invariant.witnessIds.map((witnessId) =>
          result.constraintWitnesses.find((witness) => witness.id === witnessId),
        ),
      );
      if (invariantWitnesses.some((witness) => witness?.outcome !== "PASS")) {
        fail(
          `${path}.invariants`,
          "CONDITIONAL_UNIQUENESS requires every invariant witness to PASS",
        );
      }
      break;
    }
  }
}

const EXPECTED_SOURCE_BINDINGS = [
  {
    id: "constructive-intelligence-tree",
    path: "dashboard/public/standards/constructive-intelligence-tree.v1.json",
    rawSha256: "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
  },
  {
    id: "correspondence-geometry",
    path: "dashboard/public/standards/correspondence-geometry.v0.json",
    rawSha256: "f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90",
  },
  {
    id: "knowledge-methodologies",
    path: "x/knowledge/types/methodologies.go",
    rawSha256: "fa16ac33e7f2c10a19ed76541af6c2378edb79683578f2cec6f1a0563ebec386",
  },
  {
    id: "knowledge-types",
    path: "proto/zerone/knowledge/v1/types.proto",
    rawSha256: "7b2b301c80711587a55ae03216728ec1f6f5bf981035106d26ac1fa4923d8ced",
  },
] as const;

const EXPECTED_PRIMARY_SOURCES = [
  {
    id: "arxiv-1412.4095v1",
    title: "Effective Field Theories from Soft Limits",
    authors: ["Clifford Cheung", "Karol Kampf", "Jiri Novotny", "Jaroslav Trnka"],
    locator: "https://arxiv.org/abs/1412.4095v1",
    version: "v1",
    versionDate: "2014-12-12T19:32:50Z",
  },
  {
    id: "arxiv-1509.03309v1",
    title: "On-Shell Recursion Relations for Effective Field Theories",
    authors: [
      "Clifford Cheung",
      "Karol Kampf",
      "Jiri Novotny",
      "Chia-Hsien Shen",
      "Jaroslav Trnka",
    ],
    locator: "https://arxiv.org/abs/1509.03309v1",
    version: "v1",
    versionDate: "2015-09-10T20:03:45Z",
  },
  {
    id: "arxiv-2406.02665v2",
    title: "Bootstrap Principle for the Spectrum and Scattering of Strings",
    authors: ["Clifford Cheung", "Aaron Hillman", "Grant N. Remmen"],
    locator: "https://arxiv.org/abs/2406.02665v2",
    version: "v2",
    versionDate: "2025-09-09T17:09:29Z",
  },
  {
    id: "arxiv-2508.09246v2",
    title: "Strings from Almost Nothing",
    authors: [
      "Clifford Cheung",
      "Grant N. Remmen",
      "Francesco Sciotti",
      "Michele Tarquini",
    ],
    locator: "https://arxiv.org/abs/2508.09246v2",
    version: "v2",
    versionDate: "2026-06-24T15:33:06Z",
  },
] as const;

const EXPECTED_INTEGRATION_TARGETS = [
  { id: "cg-0", status: "CURRENT_STATIC_REFERENCE" },
  { id: "knowledge-boundary", status: "CURRENT_STATIC_REFERENCE" },
  { id: "m-analogical", status: "CURRENT_STATIC_REFERENCE" },
  { id: "math-proofcraft-1", status: "CURRENT_STATIC_REFERENCE" },
  { id: "tok-per-fact-explicit-invariants", status: "NOT_IMPLEMENTED" },
] as const;

const EXPECTED_RECORDS = [
  {
    id: "boundary-probing-zero-input-robustness",
    candidateId: "scalar-eft-soft-limit-amplitudes",
    sourceId: "arxiv-1412.4095v1",
    resultKind: "FAMILY",
    localTestId: "eid1-boundary-zero-input",
  },
  {
    id: "factorization-declared-dependency-integrity",
    candidateId: "enhanced-soft-eft-tree-amplitudes",
    sourceId: "arxiv-1509.03309v1",
    resultKind: "CONDITIONAL_UNIQUENESS",
    localTestId: "eid1-declared-dependency-integrity",
  },
  {
    id: "bootstrap-conditional-solution-space",
    candidateId: "four-point-string-bootstrap-amplitudes",
    sourceId: "arxiv-2406.02665v2",
    resultKind: "CONDITIONAL_UNIQUENESS",
    localTestId: "eid1-conditional-solution-space",
  },
  {
    id: "witness-projection-publication-integrity",
    candidateId: "minimal-residue-zero-string-amplitudes",
    sourceId: "arxiv-2508.09246v2",
    resultKind: "FAMILY",
    localTestId: "eid1-witness-publication-integrity",
  },
] as const;

function assertExactSequence(
  actual: readonly string[],
  expected: readonly string[],
  path: string,
): void {
  if (
    actual.length !== expected.length ||
    actual.some((entry, index) => entry !== expected[index])
  ) {
    fail(path, "must match the reviewed sequence exactly");
  }
}

function assertExactSourceBindings(
  bindings: readonly ExplicitInvariantSourceBinding[],
): void {
  assertExactSequence(
    bindings.map((binding) => binding.id),
    EXPECTED_SOURCE_BINDINGS.map((binding) => binding.id),
    "$.sourceBindings",
  );
  bindings.forEach((binding, index) => {
    const expected = EXPECTED_SOURCE_BINDINGS[index];
    if (
      expected === undefined ||
      binding.path !== expected.path ||
      binding.rawSha256 !== expected.rawSha256
    ) {
      fail(`$.sourceBindings[${index}]`, "does not match the reviewed local byte pin");
    }
  });
}

function assertExactPrimarySources(
  sources: readonly ExplicitInvariantPrimarySource[],
): void {
  assertExactSequence(
    sources.map((source) => source.id),
    EXPECTED_PRIMARY_SOURCES.map((source) => source.id),
    "$.primarySources",
  );
  sources.forEach((source, index) => {
    const expected = EXPECTED_PRIMARY_SOURCES[index];
    if (
      expected === undefined ||
      source.title !== expected.title ||
      source.locator !== expected.locator ||
      source.version !== expected.version ||
      source.versionDate !== expected.versionDate ||
      source.retrievedDate !== "2026-08-16" ||
      source.authors.length !== expected.authors.length ||
      source.authors.some((author, authorIndex) => author !== expected.authors[authorIndex])
    ) {
      fail(`$.primarySources[${index}]`, "does not match the reviewed versioned source metadata");
    }
  });
}

function parseSourceResult(
  value: unknown,
  path: string,
  primarySourceIds: ReadonlySet<string>,
): ExplicitInvariantSourceResult {
  const source = object(value, path);
  exactKeys(source, SOURCE_RESULT_KEYS, path);
  const sourceRefs = sortedReferenceIds(source.sourceRefs, `${path}.sourceRefs`, 8);
  requireReferences(sourceRefs, primarySourceIds, `${path}.sourceRefs`);
  const scopedSourceIds = new Set(sourceRefs);
  const parsed: ExplicitInvariantSourceResult = {
    claimOwner: literal(source.claimOwner, "SOURCE_AUTHORS", `${path}.claimOwner`),
    domain: literal(source.domain, "PHYSICS_MATH", `${path}.domain`),
    candidateClass: parseCandidateClass(source.candidateClass, `${path}.candidateClass`),
    regime: parseRegime(source.regime, `${path}.regime`),
    assumptions: parseAssumptions(source.assumptions, `${path}.assumptions`, scopedSourceIds),
    invariants: parseInvariants(source.invariants, `${path}.invariants`),
    constraintWitnesses: parseWitnesses(
      source.constraintWitnesses,
      `${path}.constraintWitnesses`,
      scopedSourceIds,
    ),
    falsifiers: parseFalsifiers(source.falsifiers, `${path}.falsifiers`),
    counterexamples: parseCounterexamples(
      source.counterexamples,
      `${path}.counterexamples`,
      scopedSourceIds,
    ),
    result: parseResult(source.result, `${path}.result`),
    remainingFamily: parseRemainingFamily(
      source.remainingFamily,
      `${path}.remainingFamily`,
    ),
    boundaryTerms: parseBoundaryTerms(source.boundaryTerms, `${path}.boundaryTerms`),
    allowedConclusion: text(source.allowedConclusion, `${path}.allowedConclusion`),
    limitations: text(source.limitations, `${path}.limitations`),
    sourceRefs,
  };
  validateSourceResultSemantics(parsed, path);
  return parsed;
}

function parseTransfer(
  value: unknown,
  path: string,
  integrationTargetIds: ReadonlySet<string>,
): ExplicitInvariantZeroneTransfer {
  const transfer = object(value, path);
  exactKeys(transfer, TRANSFER_KEYS, path);
  const localTestSource = object(transfer.localTest, `${path}.localTest`);
  exactKeys(localTestSource, LOCAL_TEST_KEYS, `${path}.localTest`);
  const zeroneRefs = uniqueIds(transfer.zeroneRefs, `${path}.zeroneRefs`, 16);
  requireReferences(zeroneRefs, integrationTargetIds, `${path}.zeroneRefs`);
  const preservedDiscipline = strings(
    transfer.preservedDiscipline,
    `${path}.preservedDiscipline`,
    16,
  );
  const nonTransfers = strings(transfer.nonTransfers, `${path}.nonTransfers`, 16);
  if (preservedDiscipline.length < 2) {
    fail(`${path}.preservedDiscipline`, "must state at least two preserved practices");
  }
  if (nonTransfers.length < 3) {
    fail(`${path}.nonTransfers`, "must provide a substantive non-transfer wall");
  }
  return {
    claimOwner: literal(transfer.claimOwner, "ZERONE", `${path}.claimOwner`),
    relationKind: literal(
      transfer.relationKind,
      "METHODOLOGICAL_ANALOGY",
      `${path}.relationKind`,
    ),
    assessment: literal(transfer.assessment, "PROPOSED", `${path}.assessment`),
    target: text(transfer.target, `${path}.target`),
    preservedDiscipline,
    localTest: {
      testId: id(localTestSource.testId, `${path}.localTest.testId`),
      status: literal(localTestSource.status, "NOT_RUN", `${path}.localTest.status`),
      statement: text(localTestSource.statement, `${path}.localTest.statement`),
    },
    nonTransfers,
    zeroneRefs,
  };
}

function parseRecords(
  value: unknown,
  primarySourceIds: ReadonlySet<string>,
  integrationTargetIds: ReadonlySet<string>,
): ExplicitInvariantRecord[] {
  const entries = denseArray(value, "$.records", EXPECTED_RECORDS.length);
  if (entries.length !== EXPECTED_RECORDS.length) {
    fail("$.records", `must contain exactly ${EXPECTED_RECORDS.length} reviewed examples`);
  }
  return entries.map((entry, index) => {
    const path = `$.records[${index}]`;
    const source = object(entry, path);
    exactKeys(source, RECORD_KEYS, path);
    const record: ExplicitInvariantRecord = {
      id: id(source.id, `${path}.id`),
      title: text(source.title, `${path}.title`, 240),
      sourceResult: parseSourceResult(
        source.sourceResult,
        `${path}.sourceResult`,
        primarySourceIds,
      ),
      zeroneTransfer: parseTransfer(
        source.zeroneTransfer,
        `${path}.zeroneTransfer`,
        integrationTargetIds,
      ),
    };
    const expected = EXPECTED_RECORDS[index];
    if (
      expected === undefined ||
      record.id !== expected.id ||
      record.sourceResult.candidateClass.id !== expected.candidateId ||
      record.sourceResult.sourceRefs.length !== 1 ||
      record.sourceResult.sourceRefs[0] !== expected.sourceId ||
      record.sourceResult.result.kind !== expected.resultKind ||
      record.zeroneTransfer.localTest.testId !== expected.localTestId
    ) {
      fail(path, "does not match the reviewed example identity and ownership boundary");
    }
    return record;
  });
}

function assertGloballyUniqueDefinitionIds(
  sourceBindings: readonly ExplicitInvariantSourceBinding[],
  primarySources: readonly ExplicitInvariantPrimarySource[],
  integrationTargets: readonly ExplicitInvariantIntegrationTarget[],
  records: readonly ExplicitInvariantRecord[],
): void {
  const definitions: Array<{ id: string; path: string }> = [
    ...sourceBindings.map((binding, index) => ({
      id: binding.id,
      path: `$.sourceBindings[${index}].id`,
    })),
    ...primarySources.map((source, index) => ({
      id: source.id,
      path: `$.primarySources[${index}].id`,
    })),
    ...integrationTargets.map((target, index) => ({
      id: target.id,
      path: `$.integrationTargets[${index}].id`,
    })),
  ];
  records.forEach((record, recordIndex) => {
    const base = `$.records[${recordIndex}]`;
    definitions.push(
      { id: record.id, path: `${base}.id` },
      {
        id: record.sourceResult.candidateClass.id,
        path: `${base}.sourceResult.candidateClass.id`,
      },
      {
        id: record.zeroneTransfer.localTest.testId,
        path: `${base}.zeroneTransfer.localTest.testId`,
      },
      ...record.sourceResult.assumptions.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.assumptions[${index}].id`,
      })),
      ...record.sourceResult.invariants.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.invariants[${index}].id`,
      })),
      ...record.sourceResult.constraintWitnesses.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.constraintWitnesses[${index}].id`,
      })),
      ...record.sourceResult.falsifiers.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.falsifiers[${index}].id`,
      })),
      ...record.sourceResult.counterexamples.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.counterexamples[${index}].id`,
      })),
      ...record.sourceResult.remainingFamily.relaxationBranches.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.remainingFamily.relaxationBranches[${index}].id`,
      })),
      ...record.sourceResult.boundaryTerms.map((entry, index) => ({
        id: entry.id,
        path: `${base}.sourceResult.boundaryTerms[${index}].id`,
      })),
    );
  });
  const firstPathById = new Map<string, string>();
  for (const definition of definitions) {
    const firstPath = firstPathById.get(definition.id);
    if (firstPath !== undefined) {
      fail(
        definition.path,
        `definition id ${definition.id} collides with ${firstPath}`,
      );
    }
    firstPathById.set(definition.id, definition.path);
  }
}

function parseBrowserBoundary(value: unknown): ExplicitInvariantBrowserBoundary {
  const boundary = object(value, "$.browserBoundary");
  exactKeys(boundary, BROWSER_BOUNDARY_KEYS, "$.browserBoundary");
  return {
    staticReadCount: literal(boundary.staticReadCount, 1, "$.browserBoundary.staticReadCount"),
    sameOriginOnly: literal(boundary.sameOriginOnly, true, "$.browserBoundary.sameOriginOnly"),
    externalFetchCount: literal(
      boundary.externalFetchCount,
      0,
      "$.browserBoundary.externalFetchCount",
    ),
    purpose: text(boundary.purpose, "$.browserBoundary.purpose"),
  };
}

function parseReleaseBoundary(value: unknown): ExplicitInvariantReleaseBoundary {
  const boundary = object(value, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    literal(boundary[key], false, `$.releaseBoundary.${key}`);
  }
  return boundary as unknown as ExplicitInvariantReleaseBoundary;
}

export function parseExplicitInvariantDiscipline(
  value: unknown,
): ExplicitInvariantDiscipline {
  const root = object(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  const canonicalVocabulary = parseVocabulary(root.canonicalVocabulary);
  const sourceBindings = parseSourceBindings(root.sourceBindings);
  assertExactSourceBindings(sourceBindings);
  const primarySources = parsePrimarySources(root.primarySources);
  assertExactPrimarySources(primarySources);
  const sourceBindingIds = new Set(sourceBindings.map((binding) => binding.id));
  const integrationTargets = parseIntegrationTargets(
    root.integrationTargets,
    sourceBindingIds,
  );
  assertExactSequence(
    integrationTargets.map((target) => target.id),
    EXPECTED_INTEGRATION_TARGETS.map((target) => target.id),
    "$.integrationTargets",
  );
  integrationTargets.forEach((target, index) => {
    if (target.status !== EXPECTED_INTEGRATION_TARGETS[index]?.status) {
      fail(`$.integrationTargets[${index}].status`, "does not match the reviewed boundary");
    }
  });
  const records = parseRecords(
    root.records,
    new Set(primarySources.map((source) => source.id)),
    new Set(integrationTargets.map((target) => target.id)),
  );
  assertGloballyUniqueDefinitionIds(
    sourceBindings,
    primarySources,
    integrationTargets,
    records,
  );
  const usedSources = new Set(
    records.flatMap((record) => record.sourceResult.sourceRefs),
  );
  if (
    usedSources.size !== primarySources.length ||
    primarySources.some((source) => !usedSources.has(source.id))
  ) {
    fail("$.records", "must use every reviewed primary source");
  }
  return {
    schema: literal(root.schema, EXPLICIT_INVARIANT_DISCIPLINE_SCHEMA, "$.schema"),
    version: literal(root.version, 1, "$.version"),
    snapshotDate: literal(root.snapshotDate, "2026-08-16", "$.snapshotDate"),
    status: literal(root.status, "SEALED_STATIC_PROFILE", "$.status"),
    title: literal(root.title, "ZERONE Explicit Invariant Discipline v1", "$.title"),
    summary: text(root.summary, "$.summary"),
    attributionStatement: text(root.attributionStatement, "$.attributionStatement"),
    authorityStatement: text(root.authorityStatement, "$.authorityStatement"),
    canonicalVocabulary,
    sourceBindings,
    primarySources,
    integrationTargets,
    records,
    browserBoundary: parseBrowserBoundary(root.browserBoundary),
    releaseBoundary: parseReleaseBoundary(root.releaseBoundary),
  };
}

export function parseExplicitInvariantDisciplineJson(
  raw: string,
): ExplicitInvariantDiscipline {
  if (typeof raw !== "string") {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline input must be text",
    );
  }
  if (new TextEncoder().encode(raw).byteLength > EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES) {
    throw new ExplicitInvariantDisciplineDataError(
      `Explicit-invariant discipline exceeds ${EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES} UTF-8 bytes`,
    );
  }
  rejectDuplicateKeysAndDepth(raw);
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline contains malformed JSON",
    );
  }
  return parseExplicitInvariantDiscipline(parsed);
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new ExplicitInvariantDisciplineDataError(
      "SHA-256 verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const output = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(output)]
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline returned an empty body",
    );
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  let rejectAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectAbort = reject;
  });
  const onAbort = (): void => {
    rejectAbort?.(
      signal.reason ?? new DOMException("request timed out", "TimeoutError"),
    );
  };
  if (signal.aborted) onAbort();
  else signal.addEventListener("abort", onAbort, { once: true });
  try {
    while (true) {
      const result = await Promise.race([reader.read(), aborted]);
      if (result.done) break;
      total += result.value.byteLength;
      if (total > EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES) {
        void reader.cancel("response exceeded size limit").catch(() => undefined);
        throw new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline exceeded its size limit",
        );
      }
      chunks.push(result.value);
    }
  } catch (error) {
    if (signal.aborted) {
      try {
        void reader.cancel(signal.reason).catch(() => undefined);
      } catch {
        // Refusal does not wait for a hostile cancellation implementation.
      }
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A refused or hostile stream may already have released its lock.
    }
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

const HTTP_TOKEN = "[!#$%&'*+.^_`|~0-9A-Za-z-]+";
const HTTP_QUOTED_STRING = '"(?:[^"\\\\\\r\\n]|\\\\[\\t -~])*"';
const JSON_MEDIA_TYPE = new RegExp(
  `^application/(?:json|${HTTP_TOKEN}\\+json)` +
    `(?:\\s*;\\s*${HTTP_TOKEN}\\s*=\\s*(?:${HTTP_TOKEN}|${HTTP_QUOTED_STRING}))*\\s*$`,
  "i",
);

function refuseEarlyResponse(
  response: Response,
  controller: AbortController,
  error: ExplicitInvariantDisciplineDataError,
): never {
  if (!controller.signal.aborted) controller.abort(error);
  if (response.body !== null) {
    try {
      void response.body.cancel(error).catch(() => undefined);
    } catch {
      // Early refusal must not wait for a hostile or already locked body.
    }
  }
  throw error;
}

export async function fetchExplicitInvariantDiscipline(
  options: ExplicitInvariantDisciplineFetchOptions = {},
): Promise<ExplicitInvariantDiscipline> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (typeof fetcher !== "function") {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline fetch is unavailable",
    );
  }
  const timeoutMs = options.timeoutMs ?? EXPLICIT_INVARIANT_DISCIPLINE_TIMEOUT_MS;
  if (!Number.isSafeInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 15_000) {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline timeout is out of bounds",
    );
  }
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? "https://zerone.ai/" : window.location.href);
  let expectedUrl: URL;
  try {
    const base = new URL(baseUrl);
    expectedUrl = new URL(EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT, base);
    if (expectedUrl.origin !== base.origin) {
      throw new Error("cross-origin endpoint");
    }
  } catch {
    throw new ExplicitInvariantDisciplineDataError(
      "Explicit-invariant discipline base URL is invalid",
    );
  }

  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new DOMException("request timed out", "TimeoutError")),
    timeoutMs,
  );
  try {
    let response: Response;
    let rejectFetchAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchAbort?.(
        controller.signal.reason ??
          new DOMException("request timed out", "TimeoutError"),
      );
    };
    if (controller.signal.aborted) onFetchAbort();
    else controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    try {
      response = await Promise.race([
        fetcher(expectedUrl.href, {
          cache: "no-store",
          credentials: "omit",
          headers: { Accept: "application/json" },
          redirect: "error",
          referrerPolicy: "no-referrer",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (controller.signal.aborted) {
        throw new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline request timed out",
        );
      }
      throw new ExplicitInvariantDisciplineDataError(
        error instanceof Error
          ? `Explicit-invariant discipline is unavailable: ${error.message}`
          : "Explicit-invariant discipline is unavailable",
      );
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          `Explicit-invariant discipline returned HTTP ${response.status}`,
        ),
      );
    }
    let actualUrl: URL;
    try {
      actualUrl = new URL(response.url);
    } catch {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline returned an invalid final URL",
        ),
      );
    }
    if (response.redirected) {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline response was redirected",
        ),
      );
    }
    if (actualUrl.href !== expectedUrl.href || actualUrl.origin !== expectedUrl.origin) {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline left its exact same-origin path",
        ),
      );
    }
    const contentType = response.headers.get("content-type");
    if (contentType === null || !JSON_MEDIA_TYPE.test(contentType)) {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline returned non-JSON content",
        ),
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (
      declaredLength !== null &&
      (!/^\d+$/u.test(declaredLength) ||
        !Number.isSafeInteger(Number(declaredLength)) ||
        Number(declaredLength) > EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES)
    ) {
      refuseEarlyResponse(
        response,
        controller,
        new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline exceeded its size limit",
        ),
      );
    }
    let bytes: Uint8Array;
    try {
      bytes = await readBoundedResponse(response, controller.signal);
    } catch (error) {
      if (controller.signal.aborted) {
        throw new ExplicitInvariantDisciplineDataError(
          "Explicit-invariant discipline request timed out",
        );
      }
      throw error;
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new ExplicitInvariantDisciplineDataError(
        "Explicit-invariant discipline was not valid UTF-8",
      );
    }
    if ((await sha256Hex(bytes)) !== EXPLICIT_INVARIANT_DISCIPLINE_SHA256) {
      throw new ExplicitInvariantDisciplineDataError(
        "Explicit-invariant discipline did not match the reviewed canonical digest",
      );
    }
    return parseExplicitInvariantDisciplineJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function humanize(value: string): string {
  return value.replaceAll("_", " ").replaceAll("-", " ").toLowerCase();
}

function textList(values: readonly string[]): HTMLUListElement {
  const list = element("ul");
  for (const value of values) list.append(element("li", undefined, value));
  return list;
}

function detailEntry(
  label: string,
  value: string | readonly string[],
  modifier?: "non-transfer",
): HTMLDivElement {
  const entry = element(
    "div",
    `explicit-invariant-detail${modifier === undefined ? "" : " explicit-invariant-non-transfer"}`,
  );
  entry.append(element("dt", undefined, label));
  const body = element("dd");
  if (typeof value === "string") body.textContent = value;
  else body.append(textList(value));
  entry.append(body);
  return entry;
}

function sourceHref(path: string): string {
  return `https://github.com/cambridgetcg/zerone-core/blob/e9b0231da27568a74257ff44c89318cb83329a8f/${path}`;
}

function sourceLink(source: ExplicitInvariantPrimarySource): HTMLAnchorElement {
  const link = element("a", undefined, `${source.title} · ${source.version}`);
  link.href = source.locator;
  link.target = "_blank";
  link.rel = "noreferrer";
  return link;
}

function renderRecord(
  record: ExplicitInvariantRecord,
  primarySources: ReadonlyMap<string, ExplicitInvariantPrimarySource>,
): HTMLLIElement {
  const wrapper = element("li", "explicit-invariant-record");
  wrapper.dataset.resultKind = record.sourceResult.result.kind;
  const card = element("article", "explicit-invariant-example");
  const titleId = `explicit-invariant-runtime-${record.id}`;
  card.setAttribute("aria-labelledby", titleId);

  const heading = element("header", "explicit-invariant-example-heading");
  const recordHeading = element(
    "h4",
    "explicit-invariant-example-title",
    record.title,
  );
  recordHeading.id = titleId;
  heading.append(
    element(
      "span",
      "explicit-invariant-kicker",
      "Methodological analogy · proposed",
    ),
    recordHeading,
  );
  const badges = element("div", "explicit-invariant-badges");
  badges.append(
    element("span", "explicit-invariant-badge", humanize(record.sourceResult.result.kind)),
    element(
      "span",
      "explicit-invariant-badge",
      humanize(record.sourceResult.candidateClass.completeness),
    ),
    element(
      "span",
      "explicit-invariant-badge",
      `local test ${humanize(record.zeroneTransfer.localTest.status)}`,
    ),
  );

  const sourceResult = element("section", "explicit-invariant-source-result");
  const sourceHeadingId = `${titleId}-source`;
  sourceResult.setAttribute("aria-labelledby", sourceHeadingId);
  const sourceHeading = element(
    "h5",
    "explicit-invariant-detail-label",
    "Published source result",
  );
  sourceHeading.id = sourceHeadingId;
  sourceResult.append(sourceHeading);
  const sourceDetails = element("dl", "explicit-invariant-detail-grid");
  sourceDetails.append(
    detailEntry(
      "Candidate class",
      `${record.sourceResult.candidateClass.id}: ${record.sourceResult.candidateClass.definition} Membership: ${record.sourceResult.candidateClass.membershipRule}`,
    ),
    detailEntry(
      "Regime",
      `${record.sourceResult.regime.formalSystem} Background: ${record.sourceResult.regime.background} Parameters: ${record.sourceResult.regime.parameterDomain} Approximation: ${record.sourceResult.regime.approximationOrder} Valid: ${record.sourceResult.regime.validityScope} Excluded: ${record.sourceResult.regime.excludedScope}`,
    ),
    detailEntry(
      "Assumptions",
      record.sourceResult.assumptions.map(
        (assumption) => `${assumption.id} · ${humanize(assumption.kind)} · ${assumption.statement}`,
      ),
    ),
    detailEntry(
      "Invariants",
      record.sourceResult.invariants.map(
        (invariant) =>
          `${invariant.id} · ${humanize(invariant.mode)} · ${invariant.statement} Scope: ${invariant.scope} Witnesses: ${invariant.witnessIds.join(", ")}.`,
      ),
    ),
    detailEntry(
      "Constraint witnesses",
      record.sourceResult.constraintWitnesses.map(
        (witness) =>
          `${witness.id} · ${humanize(witness.kind)} · ${witness.outcome}: ${witness.statement} Targets: ${witness.targetRefs.join(", ")}. Procedure: ${witness.procedure}`,
      ),
    ),
    detailEntry(
      "Falsifiers",
      record.sourceResult.falsifiers.map(
        (falsifier) =>
          `${falsifier.id} · ${humanize(falsifier.status)} · ${falsifier.condition} Procedure: ${falsifier.procedure}`,
      ),
    ),
    detailEntry(
      "Counterexamples",
      record.sourceResult.counterexamples.map(
        (counterexample) => {
          const branches =
            counterexample.relaxationBranchRefs.length === 0
              ? ""
              : ` Relaxation branches: ${counterexample.relaxationBranchRefs.join(", ")}.`;
          return `${counterexample.id} · ${humanize(counterexample.disposition)} · ${counterexample.member}: ${counterexample.explanation} Targets: ${counterexample.targetRefs.join(", ")}.${branches}`;
        },
      ),
    ),
    detailEntry(
      "Bounded result",
      `${humanize(record.sourceResult.result.kind)}: ${record.sourceResult.result.statement} Assumptions: ${record.sourceResult.result.underAssumptionIds.join(", ")}. Invariants: ${record.sourceResult.result.underInvariantIds.join(", ")}. Witnesses: ${record.sourceResult.result.witnessIds.join(", ")}. Allowed conclusion: ${record.sourceResult.allowedConclusion}`,
    ),
    detailEntry(
      "Remaining family",
      `${humanize(record.sourceResult.remainingFamily.cardinality)} · ${record.sourceResult.remainingFamily.description} Relaxation branches: ${record.sourceResult.remainingFamily.relaxationBranches.map((branch) => `${branch.id} relaxes ${branch.relaxedAssumptionIds.join(", ")}: ${branch.statement}`).join(" ")}`,
    ),
    detailEntry(
      "Boundary terms",
      record.sourceResult.boundaryTerms.map(
        (term) =>
          `${term.id} · ${humanize(term.treatment)} · ${term.term}. ${term.justification} Assumptions: ${term.underAssumptionIds.join(", ")}. Invariants: ${term.underInvariantIds.join(", ")}. Witnesses: ${term.witnessIds.join(", ")}. Affected result: ${term.affectedResult}`,
      ),
    ),
    detailEntry("Limitations", record.sourceResult.limitations, "non-transfer"),
  );
  sourceResult.append(sourceDetails);

  const transfer = element("section", "explicit-invariant-zerone-transfer");
  const transferHeadingId = `${titleId}-transfer`;
  transfer.setAttribute("aria-labelledby", transferHeadingId);
  const transferHeading = element(
    "h5",
    "explicit-invariant-detail-label",
    "Zerone-owned proposed transfer",
  );
  transferHeading.id = transferHeadingId;
  transfer.append(transferHeading);
  const transferDetails = element("dl", "explicit-invariant-detail-grid");
  transferDetails.append(
    detailEntry("Target", record.zeroneTransfer.target),
    detailEntry("Preserved discipline", record.zeroneTransfer.preservedDiscipline),
    detailEntry(
      "Local test · not run",
      `${record.zeroneTransfer.localTest.testId}: ${record.zeroneTransfer.localTest.statement}`,
    ),
    detailEntry("Must not transfer", record.zeroneTransfer.nonTransfers, "non-transfer"),
  );
  transfer.append(transferDetails);

  const provenance = element("p", "explicit-invariant-sources");
  provenance.append(element("strong", undefined, "Versioned source: "));
  record.sourceResult.sourceRefs.forEach((sourceId, index) => {
    const source = primarySources.get(sourceId);
    if (source === undefined) return;
    if (index > 0) provenance.append(document.createTextNode(" · "));
    provenance.append(sourceLink(source));
    provenance.append(
      document.createTextNode(` · ${source.authors.join(", ")} · ${source.versionDate}`),
    );
  });
  provenance.append(
    document.createTextNode(
      ` · Zerone references: ${record.zeroneTransfer.zeroneRefs.join(", ")}.`,
    ),
  );

  card.append(heading, badges, sourceResult, transfer, provenance);
  wrapper.append(card);
  return wrapper;
}

export function renderExplicitInvariantDiscipline(
  root: HTMLElement,
  discipline: ExplicitInvariantDiscipline,
): void {
  const shell = element("div", "explicit-invariant-shell");

  const attribution = element("section", "explicit-invariant-attribution");
  const attributionHeading = element("div");
  attributionHeading.append(
    element("span", "explicit-invariant-kicker", "Attribution and ownership boundary"),
    element("h3", undefined, "Source results and Zerone transfers remain separate."),
  );
  const attributionCopy = element("div");
  attributionCopy.append(
    element("p", undefined, discipline.attributionStatement),
    element("p", undefined, discipline.authorityStatement),
  );
  attribution.append(attributionHeading, attributionCopy);

  const noEffects = element("section", "explicit-invariant-no-effects");
  const boundaryHeading = element("div");
  boundaryHeading.append(
    element("span", "explicit-invariant-kicker", "Release boundary · all values false"),
    element("h3", undefined, "Inspection creates no protocol or authority effect."),
  );
  const boundaryCopy = element("div");
  boundaryCopy.append(
    element("p", undefined, discipline.browserBoundary.purpose),
    element(
      "p",
      undefined,
      "That automatic read and local render perform no wallet, RPC, chain, identity, analytics, consent, reward, qualification, or authority action.",
    ),
  );
  noEffects.append(boundaryHeading, boundaryCopy);

  const workflow = element("section", "explicit-invariant-workflow");
  workflow.append(element("h3", undefined, "A conclusion is only as broad as its declared class."));
  const workflowList = element("ol", "explicit-invariant-workflow-list");
  workflowList.setAttribute("role", "list");
  const stages = [
    ["1 · Name the class", "Define membership, regime, approximation, and excluded scope."],
    ["2 · State the constraints", "Separate assumptions from invariants and name each boundary term."],
    ["3 · Publish the witnesses", "Attach procedures, outcomes, falsifiers, and counterexamples."],
    ["4 · Bound the conclusion", "Report a family, scoped no-go, or conditional uniqueness—and what reopens when constraints relax."],
  ] as const;
  for (const [stage, explanation] of stages) {
    const item = element("li");
    item.append(element("strong", undefined, stage), element("p", undefined, explanation));
    workflowList.append(item);
  }
  workflow.append(workflowList);

  const examples = element("section", "explicit-invariant-principles");
  examples.append(element("h3", undefined, "Four scoped examples"));
  const controls = element("div", "explicit-invariant-controls");
  controls.setAttribute("role", "group");
  controls.setAttribute("aria-label", "Filter examples by bounded result kind");
  const count = element("span", "explicit-invariant-visible-count");
  count.setAttribute("aria-live", "polite");
  const grid = element("ol", "explicit-invariant-example-grid");
  grid.setAttribute("role", "list");
  const sourceMap = new Map(
    discipline.primarySources.map((source) => [source.id, source] as const),
  );
  const cards = discipline.records.map((record) => renderRecord(record, sourceMap));
  for (const card of cards) grid.append(card);
  const buttons = new Map<string, HTMLButtonElement>();

  const choose = (kind: ExplicitInvariantResultKind | null): void => {
    let shown = 0;
    for (const [value, button] of buttons) {
      const active = value === (kind ?? "ALL");
      button.setAttribute("aria-pressed", String(active));
      button.classList.toggle("is-active", active);
    }
    for (const card of cards) {
      const visible = kind === null || card.dataset.resultKind === kind;
      card.hidden = !visible;
      if (visible) shown += 1;
    }
    count.textContent = `${shown} example${shown === 1 ? "" : "s"}`;
  };
  const addFilter = (kind: "ALL" | ExplicitInvariantResultKind): void => {
    const button = element("button", "explicit-invariant-filter", humanize(kind));
    button.type = "button";
    button.dataset.resultKind = kind;
    button.setAttribute("aria-pressed", "false");
    button.addEventListener("click", () => choose(kind === "ALL" ? null : kind));
    buttons.set(kind, button);
    controls.append(button);
  };
  addFilter("ALL");
  const representedKinds = new Set(
    discipline.records.map((record) => record.sourceResult.result.kind),
  );
  for (const kind of discipline.canonicalVocabulary.resultKinds) {
    if (representedKinds.has(kind)) addFilter(kind);
  }
  controls.append(count);
  examples.append(controls, grid);
  choose(null);

  const integrations = element("section", "explicit-invariant-principles");
  integrations.append(element("h3", undefined, "Integration boundary"));
  const integrationGrid = element("div", "explicit-invariant-principle-grid");
  for (const target of discipline.integrationTargets) {
    const card = element("div", "explicit-invariant-principle");
    card.append(
      element("strong", undefined, `${target.label} · ${humanize(target.status)}`),
      element("p", undefined, target.currentBinding),
      element("small", undefined, target.futureBoundary),
    );
    if (target.id === "math-proofcraft-1") {
      const link = element("a", undefined, "Related curriculum: Proof construction and counterexamples ↓");
      link.href = "#skills";
      card.append(
        link,
        element(
          "small",
          undefined,
          "Opening it records nothing and grants no qualification or reward.",
        ),
      );
    }
    integrationGrid.append(card);
  }
  integrations.append(integrationGrid);

  const sources = element("section", "explicit-invariant-sources-panel");
  sources.append(element("h3", undefined, "Pinned local bindings and versioned papers"));
  const sourceGrid = element("div", "explicit-invariant-source-grid");
  for (const binding of discipline.sourceBindings) {
    const card = element("div", "explicit-invariant-source");
    const link = element("a", undefined, binding.path);
    link.href = sourceHref(binding.path);
    link.target = "_blank";
    link.rel = "noreferrer";
    card.append(
      element("strong", undefined, binding.id),
      link,
      element("code", undefined, `sha256:${binding.rawSha256}`),
      element("small", undefined, binding.boundary),
    );
    sourceGrid.append(card);
  }
  for (const source of discipline.primarySources) {
    const card = element("div", "explicit-invariant-source");
    card.append(
      element("strong", undefined, source.authors.join(", ")),
      sourceLink(source),
      element("span", undefined, source.versionDate),
      element("small", undefined, source.boundary),
    );
    sourceGrid.append(card);
  }
  sources.append(sourceGrid);

  const effects = element("section", "explicit-invariant-principles");
  effects.append(element("h3", undefined, "Twenty-four disabled effects"));
  const effectGrid = element("div", "explicit-invariant-principle-grid");
  for (const [name, value] of Object.entries(discipline.releaseBoundary)) {
    const item = element("div", "explicit-invariant-principle");
    item.append(element("strong", undefined, name), element("code", undefined, String(value)));
    effectGrid.append(item);
  }
  effects.append(effectGrid);

  shell.append(
    attribution,
    noEffects,
    workflow,
    examples,
    integrations,
    sources,
    effects,
  );
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseExplicitInvariantDiscipline(
  root: HTMLElement,
  options: ExplicitInvariantDisciplineFetchOptions = {},
): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderExplicitInvariantDiscipline(
      root,
      await fetchExplicitInvariantDiscipline(options),
    );
  } catch (error) {
    const failure = element("div", "explicit-invariant-load-error");
    failure.setAttribute("role", "alert");
    const actions = element("div", "explicit-invariant-load-actions");
    const raw = element("a", "explicit-invariant-raw", "Open raw static profile ↗");
    raw.href = EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT;
    raw.target = "_blank";
    raw.rel = "noreferrer";
    actions.append(raw);
    failure.append(
      element("strong", undefined, "The explicit-invariant discipline could not be verified."),
      element(
        "p",
        undefined,
        error instanceof Error
          ? error.message
          : "The sealed static profile is unavailable.",
      ),
      element(
        "p",
        undefined,
        "No scientific, theological, protocol, personhood, authority, reward, qualification, consent, or unique-design conclusion was accepted.",
      ),
      actions,
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
