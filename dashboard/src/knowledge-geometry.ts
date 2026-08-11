/// <reference lib="dom" />

export const KNOWLEDGE_GEOMETRY_ENDPOINT = "/api/knowledge";
export const KNOWLEDGE_GEOMETRY_MAX_BYTES = 262_144;
export const KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX = 960;
export const KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX = 8_192;
export const KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX = 24;

const SNAPSHOT_SCHEMA = "zerone.knowledge-geometry-snapshot/v0";
const FACT_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const DECIMAL = /^(?:0|[1-9][0-9]*)$/;
const METHOD = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const BIDI_CONTROLS = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/u;
const UNSAFE_CONTROLS = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u;
const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5));
const CONTENT_MAX_BYTES = 16_384;
const LABEL_MAX_BYTES = 128;
const FACT_STATUSES = new Set([
  "FACT_STATUS_UNSPECIFIED",
  "FACT_STATUS_PENDING",
  "FACT_STATUS_PROVISIONAL",
  "FACT_STATUS_VERIFIED",
  "FACT_STATUS_ACTIVE",
  "FACT_STATUS_CONTESTED",
  "FACT_STATUS_CHALLENGED",
  "FACT_STATUS_SUPERSEDED",
  "FACT_STATUS_EXPIRED",
  "FACT_STATUS_DISPROVEN",
  "FACT_STATUS_REVOKED",
  "FACT_STATUS_AT_RISK",
  "FACT_STATUS_PRUNED",
]);
const CLAIM_TYPES = new Set([
  "CLAIM_TYPE_UNSPECIFIED",
  "CLAIM_TYPE_ASSERTION",
  "CLAIM_TYPE_RELATION",
  "CLAIM_TYPE_DEFINITION",
  "CLAIM_TYPE_CONSTRAINT",
  "CLAIM_TYPE_NEGATION",
  "CLAIM_TYPE_OBSERVATION",
  "CLAIM_TYPE_COMPUTATIONAL",
  "CLAIM_TYPE_CONJECTURE",
]);
const RELATION_TYPES = new Set([
  "RELATION_TYPE_UNSPECIFIED",
  "RELATION_TYPE_SUPPORTS",
  "RELATION_TYPE_CONTRADICTS",
  "RELATION_TYPE_REQUIRES",
  "RELATION_TYPE_REFINES",
  "RELATION_TYPE_GENERALIZES",
  "RELATION_TYPE_SUPERSEDES",
  "RELATION_TYPE_CITES",
  "RELATION_TYPE_REFORMULATES",
]);
const INFERENCE_TYPES = new Set([
  "INFERENCE_TYPE_UNSPECIFIED",
  "INFERENCE_TYPE_DEDUCTIVE",
  "INFERENCE_TYPE_INDUCTIVE",
  "INFERENCE_TYPE_ABDUCTIVE",
  "INFERENCE_TYPE_EMPIRICAL",
  "INFERENCE_TYPE_ANALOGICAL",
  "INFERENCE_TYPE_CITATION",
]);

type JsonObject = Record<string, unknown>;

export interface KnowledgeGeometrySource {
  chainId: "zerone-1";
  blockHeight: string;
  statusHeight: string;
  catchingUp: boolean;
  queryPath: "/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true";
  queryTracked: false;
  writes: false;
  completeness: "NOT_CLAIMED";
  upstreamRecords: number;
  returnedRecords: number;
  truncated: boolean;
}

export interface KnowledgeGeometryFact {
  id: string;
  content: string;
  domain: string;
  category: string;
  status: string;
  claimType: string;
  confidence: number;
  verifiedAtBlock: string;
  lastVerifiedBlock: string;
  energy: number;
  energyCap: number;
  fitnessScore: number;
  methodId: string;
}

export interface KnowledgeGeometryRelation {
  sourceFactId: string;
  targetFactId: string;
  relation: string;
  inference: string;
  inferenceStrengthBps: number;
  createdAtBlock: string;
  methodId: string;
}

export interface KnowledgeGeometrySnapshot {
  schema: typeof SNAPSHOT_SCHEMA;
  source: KnowledgeGeometrySource;
  facts: KnowledgeGeometryFact[];
  relations: KnowledgeGeometryRelation[];
}

export interface KnowledgeGeometryFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export interface KnowledgeGeometryPoint {
  factId: string;
  domain: string;
  domainIndex: number;
  x: number;
  y: number;
}

export interface KnowledgeGeometryDomain {
  id: string;
  index: number;
  x: number;
  y: number;
  factCount: number;
}

export interface KnowledgeGeometryLayout {
  domains: KnowledgeGeometryDomain[];
  points: KnowledgeGeometryPoint[];
}

export class KnowledgeGeometryDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "KnowledgeGeometryDataError";
  }
}

function fail(path: string, message: string): never {
  throw new KnowledgeGeometryDataError(`${path}: ${message}`);
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
}

function denseArray(
  value: unknown,
  path: string,
  maximum: number,
): unknown[] {
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

function text(value: unknown, path: string, maximum: number): string {
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

function boundedUtf8Text(
  value: unknown,
  path: string,
  maximumBytes: number,
  allowEmpty: boolean,
): string {
  if (
    typeof value !== "string" ||
    (!allowEmpty && value.trim().length === 0) ||
    new TextEncoder().encode(value).byteLength > maximumBytes ||
    BIDI_CONTROLS.test(value) ||
    UNSAFE_CONTROLS.test(value)
  ) {
    fail(
      path,
      `must be safe UTF-8 text no longer than ${maximumBytes} bytes${allowEmpty ? "" : " and must not be empty"}`,
    );
  }
  return value;
}

function patterned(
  value: unknown,
  path: string,
  pattern: RegExp,
  maximum: number,
): string {
  const result = text(value, path, maximum);
  if (!pattern.test(result)) fail(path, "has an invalid format");
  return result;
}

function optionalPatterned(
  value: unknown,
  path: string,
  pattern: RegExp,
  maximum: number,
): string {
  if (value === "") return "";
  return patterned(value, path, pattern, maximum);
}

function enumMember(
  value: unknown,
  path: string,
  allowed: ReadonlySet<string>,
): string {
  if (typeof value !== "string" || !allowed.has(value)) {
    fail(path, "is not an accepted protocol enum value");
  }
  return value;
}

function exact<T>(value: unknown, expected: T, path: string): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function metric(value: unknown, path: string, maximum = 1_000_000): number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value < 0 ||
    value > maximum
  ) {
    fail(path, `must be an integer between 0 and ${maximum}`);
  }
  return value;
}

function count(value: unknown, path: string, maximum: number): number {
  return metric(value, path, maximum);
}

function blockHeight(value: unknown, path: string): string {
  const result = patterned(value, path, DECIMAL, 20);
  const parsed = BigInt(result);
  if (parsed === 0n || parsed > (1n << 64n) - 1n) {
    fail(path, "must be a positive uint64 height");
  }
  return result;
}

function uint64String(value: unknown, path: string): string {
  const result = patterned(value, path, DECIMAL, 20);
  if (BigInt(result) > (1n << 64n) - 1n) {
    fail(path, "must be a canonical uint64 value");
  }
  return result;
}

function rejectDuplicateJsonKeys(raw: string): void {
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
  const scanValue = (path: string, depth = 0): void => {
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
    while (offset < raw.length && !/[\s,\]}]/u.test(raw[offset] ?? "")) {
      offset += 1;
    }
    if (offset === start) fail(path, "contains a malformed JSON value");
  };
  scanValue("$");
  whitespace();
  if (offset !== raw.length) fail("$", "contains trailing JSON data");
}

function parseSource(value: unknown): KnowledgeGeometrySource {
  const source = object(value, "$.source");
  exactKeys(
    source,
    [
      "chainId",
      "blockHeight",
      "statusHeight",
      "catchingUp",
      "queryPath",
      "queryTracked",
      "writes",
      "completeness",
      "upstreamRecords",
      "returnedRecords",
      "truncated",
    ],
    "$.source",
  );
  const block = blockHeight(source.blockHeight, "$.source.blockHeight");
  const status = blockHeight(source.statusHeight, "$.source.statusHeight");
  const blockDelta = BigInt(block) - BigInt(status);
  if (blockDelta > 128n || blockDelta < -128n) {
    fail("$.source", "REST and status heights must remain within 128 blocks");
  }
  const catchingUp = source.catchingUp;
  if (typeof catchingUp !== "boolean") fail("$.source.catchingUp", "must be boolean");
  const truncated = source.truncated;
  if (typeof truncated !== "boolean") fail("$.source.truncated", "must be boolean");
  const upstreamRecords = count(source.upstreamRecords, "$.source.upstreamRecords", 8_192);
  const returnedRecords = count(source.returnedRecords, "$.source.returnedRecords", 128);
  if (returnedRecords > upstreamRecords) {
    fail("$.source.returnedRecords", "must not exceed upstreamRecords");
  }
  if (!truncated && returnedRecords !== upstreamRecords) {
    fail("$.source.truncated", "must disclose record truncation");
  }
  return {
    chainId: exact(source.chainId, "zerone-1", "$.source.chainId"),
    blockHeight: block,
    statusHeight: status,
    catchingUp,
    queryPath: exact(
      source.queryPath,
      "/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true",
      "$.source.queryPath",
    ),
    queryTracked: exact(source.queryTracked, false, "$.source.queryTracked"),
    writes: exact(source.writes, false, "$.source.writes"),
    completeness: exact(
      source.completeness,
      "NOT_CLAIMED",
      "$.source.completeness",
    ),
    upstreamRecords,
    returnedRecords,
    truncated,
  };
}

function parseFact(value: unknown, index: number): KnowledgeGeometryFact {
  const path = `$.facts[${index}]`;
  const fact = object(value, path);
  exactKeys(
    fact,
    [
      "id",
      "content",
      "domain",
      "category",
      "status",
      "claimType",
      "confidence",
      "verifiedAtBlock",
      "lastVerifiedBlock",
      "energy",
      "energyCap",
      "fitnessScore",
      "methodId",
    ],
    path,
  );
  const energyCap = metric(fact.energyCap, `${path}.energyCap`);
  const energy = metric(fact.energy, `${path}.energy`);
  return {
    id: patterned(fact.id, `${path}.id`, FACT_ID, 128),
    content: boundedUtf8Text(
      fact.content,
      `${path}.content`,
      CONTENT_MAX_BYTES,
      false,
    ),
    domain: boundedUtf8Text(
      fact.domain,
      `${path}.domain`,
      LABEL_MAX_BYTES,
      true,
    ),
    category: boundedUtf8Text(
      fact.category,
      `${path}.category`,
      LABEL_MAX_BYTES,
      true,
    ),
    status: enumMember(fact.status, `${path}.status`, FACT_STATUSES),
    claimType: enumMember(fact.claimType, `${path}.claimType`, CLAIM_TYPES),
    confidence: metric(fact.confidence, `${path}.confidence`),
    verifiedAtBlock: uint64String(
      fact.verifiedAtBlock,
      `${path}.verifiedAtBlock`,
    ),
    lastVerifiedBlock: uint64String(
      fact.lastVerifiedBlock,
      `${path}.lastVerifiedBlock`,
    ),
    energy,
    energyCap,
    fitnessScore: metric(fact.fitnessScore, `${path}.fitnessScore`),
    methodId: optionalPatterned(fact.methodId, `${path}.methodId`, METHOD, 128),
  };
}

function parseRelation(
  value: unknown,
  index: number,
): KnowledgeGeometryRelation {
  const path = `$.relations[${index}]`;
  const relation = object(value, path);
  exactKeys(
    relation,
    [
      "sourceFactId",
      "targetFactId",
      "relation",
      "inference",
      "inferenceStrengthBps",
      "createdAtBlock",
      "methodId",
    ],
    path,
  );
  return {
    sourceFactId: patterned(
      relation.sourceFactId,
      `${path}.sourceFactId`,
      FACT_ID,
      128,
    ),
    targetFactId: patterned(
      relation.targetFactId,
      `${path}.targetFactId`,
      FACT_ID,
      128,
    ),
    relation: enumMember(
      relation.relation,
      `${path}.relation`,
      RELATION_TYPES,
    ),
    inference: enumMember(
      relation.inference,
      `${path}.inference`,
      INFERENCE_TYPES,
    ),
    inferenceStrengthBps: metric(
      relation.inferenceStrengthBps,
      `${path}.inferenceStrengthBps`,
    ),
    createdAtBlock: uint64String(
      relation.createdAtBlock,
      `${path}.createdAtBlock`,
    ),
    methodId: optionalPatterned(
      relation.methodId,
      `${path}.methodId`,
      METHOD,
      128,
    ),
  };
}

export function parseKnowledgeGeometry(value: unknown): KnowledgeGeometrySnapshot {
  const root = object(value, "$");
  exactKeys(root, ["schema", "source", "facts", "relations"], "$");
  const source = parseSource(root.source);
  const facts = denseArray(root.facts, "$.facts", 128).map(parseFact);
  if (facts.length !== source.returnedRecords) {
    fail("$.facts", "length must equal source.returnedRecords");
  }
  const ids = new Set<string>();
  let previousFactId: string | undefined;
  facts.forEach((fact, index) => {
    if (ids.has(fact.id)) fail(`$.facts[${index}].id`, "must be unique");
    if (previousFactId !== undefined && previousFactId >= fact.id) {
      fail(`$.facts[${index}].id`, "must remain in deterministic lexical order");
    }
    ids.add(fact.id);
    previousFactId = fact.id;
    if (
      BigInt(fact.verifiedAtBlock) > BigInt(source.blockHeight) ||
      BigInt(fact.verifiedAtBlock) > BigInt(source.statusHeight) ||
      BigInt(fact.lastVerifiedBlock) > BigInt(source.blockHeight) ||
      BigInt(fact.lastVerifiedBlock) > BigInt(source.statusHeight)
    ) {
      fail(`$.facts[${index}]`, "contains a height after the source snapshot");
    }
    if (BigInt(fact.verifiedAtBlock) > BigInt(fact.lastVerifiedBlock)) {
      fail(`$.facts[${index}]`, "contains inconsistent verification heights");
    }
  });
  const relations = denseArray(root.relations, "$.relations", 512).map(
    parseRelation,
  );
  const relationKeys = new Set<string>();
  let previousRelationIdentity: string | undefined;
  relations.forEach((relation, index) => {
    if (!ids.has(relation.sourceFactId) && !ids.has(relation.targetFactId)) {
      fail(`$.relations[${index}]`, "must touch at least one returned fact");
    }
    if (
      BigInt(relation.createdAtBlock) > BigInt(source.blockHeight) ||
      BigInt(relation.createdAtBlock) > BigInt(source.statusHeight)
    ) {
      fail(`$.relations[${index}].createdAtBlock`, "is after the source snapshot");
    }
    const key = [relation.sourceFactId, relation.targetFactId].join("\u0000");
    if (relationKeys.has(key)) fail(`$.relations[${index}]`, "is duplicated");
    relationKeys.add(key);
    const identity = [
      relation.sourceFactId,
      relation.targetFactId,
      relation.relation,
      relation.inference,
      String(relation.inferenceStrengthBps),
      relation.createdAtBlock,
      relation.methodId,
    ].join("\u0000");
    if (previousRelationIdentity !== undefined && previousRelationIdentity >= identity) {
      fail(`$.relations[${index}]`, "must remain in deterministic lexical order");
    }
    previousRelationIdentity = identity;
  });
  return {
    schema: exact(root.schema, SNAPSHOT_SCHEMA, "$.schema"),
    source,
    facts,
    relations,
  };
}

export function parseKnowledgeGeometryJson(raw: string): KnowledgeGeometrySnapshot {
  if (new TextEncoder().encode(raw).byteLength > KNOWLEDGE_GEOMETRY_MAX_BYTES) {
    fail("$", `exceeds ${KNOWLEDGE_GEOMETRY_MAX_BYTES} bytes`);
  }
  rejectDuplicateJsonKeys(raw);
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch {
    fail("$", "is not valid JSON");
  }
  return parseKnowledgeGeometry(parsed);
}

async function readBoundedResponse(
  response: Response,
  maximumBytes: number,
  signal: AbortSignal,
): Promise<string> {
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!DECIMAL.test(declaredLength) || Number(declaredLength) > maximumBytes)
  ) {
    fail("$", `response exceeds ${maximumBytes} bytes`);
  }
  const reader = response.body?.getReader();
  if (!reader) return "";
  const chunks: Uint8Array[] = [];
  let total = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(
      signal.reason ??
        new DOMException("Knowledge projection request timed out", "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      total += value.byteLength;
      if (total > maximumBytes) {
        void reader.cancel().catch(() => undefined);
        fail("$", `response exceeds ${maximumBytes} bytes`);
      }
      chunks.push(value);
    }
  } catch (error) {
    void reader.cancel(signal.reason).catch(() => undefined);
    if (signal.aborted) {
      throw new KnowledgeGeometryDataError("$: projection request timed out");
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A hostile pending read is abandoned after the refusal boundary wins.
    }
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  chunks.forEach((chunk) => {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  });
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    fail("$", "response is not valid UTF-8");
  }
}

export async function fetchKnowledgeGeometry(
  options: KnowledgeGeometryFetchOptions = {},
): Promise<KnowledgeGeometrySnapshot> {
  const fetcher = options.fetcher ?? globalThis.fetch.bind(globalThis);
  const timeoutMs = options.timeoutMs ?? 8_000;
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? "https://zerone.ai" : window.location.origin);
  let expected: URL;
  try {
    expected = new URL(KNOWLEDGE_GEOMETRY_ENDPOINT, baseUrl);
  } catch {
    throw new KnowledgeGeometryDataError(
      "$: the knowledge projection URL is invalid",
    );
  }
  if (
    (expected.protocol !== "https:" && expected.protocol !== "http:") ||
    expected.username !== "" ||
    expected.password !== ""
  ) {
    throw new KnowledgeGeometryDataError(
      "$: the knowledge projection URL is invalid",
    );
  }
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(() => {
    controller.abort(
      new DOMException("Knowledge projection request timed out", "TimeoutError"),
    );
  }, timeoutMs);
  const signal = controller.signal;
  try {
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        signal.reason ??
          new DOMException("Knowledge projection request timed out", "TimeoutError"),
      );
    };
    signal.addEventListener("abort", onFetchAbort, { once: true });
    if (signal.aborted) onFetchAbort();
    let response: Response;
    try {
      response = await Promise.race([
        fetcher(expected, {
          method: "GET",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          redirect: "manual",
          signal,
        }),
        fetchAborted,
      ]);
    } catch {
      throw new KnowledgeGeometryDataError(
        signal.aborted
          ? "$: projection request timed out"
          : "$: the bounded zerone-1 knowledge projection is unavailable",
      );
    } finally {
      signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok || response.status !== 200) {
      fail("$", `projection returned HTTP ${response.status}`);
    }
    if (response.redirected) fail("$", "projection redirects are refused");
    if (response.url) {
      let actual: URL;
      try {
        actual = new URL(response.url);
      } catch {
        fail("$", "projection response URL is invalid");
      }
      if (
        actual.origin !== expected.origin ||
        actual.username !== "" ||
        actual.password !== "" ||
        actual.pathname !== expected.pathname ||
        actual.search !== "" ||
        actual.hash !== ""
      ) {
        fail("$", "projection response URL does not match the requested endpoint");
      }
    }
    const mediaType = response.headers
      .get("content-type")
      ?.split(";", 1)[0]
      ?.trim()
      .toLowerCase();
    if (mediaType !== "application/json") {
      fail("$", "projection must use application/json");
    }
    const raw = await readBoundedResponse(
      response,
      KNOWLEDGE_GEOMETRY_MAX_BYTES,
      signal,
    );
    return parseKnowledgeGeometryJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function stableHash(value: string): number {
  let hash = 0x811c9dc5;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash >>> 0;
}

function lexicalCompare(left: string, right: string): number {
  return left < right ? -1 : left > right ? 1 : 0;
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.max(minimum, Math.min(maximum, value));
}

export function buildKnowledgeGeometryLayout(
  facts: readonly KnowledgeGeometryFact[],
): KnowledgeGeometryLayout {
  const grouped = new Map<string, KnowledgeGeometryFact[]>();
  [...facts]
    .sort((left, right) => lexicalCompare(left.id, right.id))
    .forEach((fact) => {
      const group = grouped.get(fact.domain) ?? [];
      group.push(fact);
      grouped.set(fact.domain, group);
    });
  const domainIds = [...grouped.keys()].sort();
  const domains: KnowledgeGeometryDomain[] = [];
  const points: KnowledgeGeometryPoint[] = [];
  domainIds.forEach((domain, domainIndex) => {
    const angle =
      domainIds.length === 1
        ? 0
        : -Math.PI / 2 + (domainIndex * Math.PI * 2) / domainIds.length;
    const centerX = domainIds.length === 1 ? 50 : 50 + Math.cos(angle) * 32;
    const centerY = domainIds.length === 1 ? 50 : 50 + Math.sin(angle) * 29;
    const entries = grouped.get(domain) ?? [];
    domains.push({
      id: domain,
      index: domainIndex,
      x: centerX,
      y: centerY,
      factCount: entries.length,
    });
    const seed = ((stableHash(domain) % 360) * Math.PI) / 180;
    entries.forEach((fact, factIndex) => {
      const radius = factIndex === 0 ? 0 : Math.min(12, 3.05 * Math.sqrt(factIndex));
      const localAngle = seed + factIndex * GOLDEN_ANGLE;
      points.push({
        factId: fact.id,
        domain,
        domainIndex,
        x: clamp(centerX + Math.cos(localAngle) * radius, 3, 97),
        y: clamp(centerY + Math.sin(localAngle) * radius, 5, 95),
      });
    });
  });
  return { domains, points };
}

export function knowledgeGeometryPlaneSize(
  points: readonly KnowledgeGeometryPoint[],
): number {
  for (const point of points) {
    if (!Number.isFinite(point.x) || !Number.isFinite(point.y)) {
      fail("$.geometry", "layout points must use finite coordinates");
    }
  }
  if (points.length < 2) return KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX;

  let minimumDistance = Number.POSITIVE_INFINITY;
  for (let left = 0; left < points.length; left += 1) {
    for (let right = left + 1; right < points.length; right += 1) {
      const distance = Math.hypot(
        points[left]!.x - points[right]!.x,
        points[left]!.y - points[right]!.y,
      );
      if (distance === 0) {
        fail("$.geometry", "layout points must not coincide");
      }
      minimumDistance = Math.min(minimumDistance, distance);
    }
  }

  const requiredSize = Math.ceil(
    (KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX * 100) / minimumDistance,
  );
  if (
    !Number.isFinite(requiredSize) ||
    requiredSize > KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX
  ) {
    fail(
      "$.geometry",
      `${KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX}px record-node separation would require a plane larger than ${KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX}px`,
    );
  }
  return Math.max(KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX, requiredSize);
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  content?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (content !== undefined) node.textContent = content;
  return node;
}

function svgElement<K extends keyof SVGElementTagNameMap>(
  tag: K,
): SVGElementTagNameMap[K] {
  return document.createElementNS("http://www.w3.org/2000/svg", tag);
}

function humanEnum(value: string, prefix: string): string {
  return value
    .replace(prefix, "")
    .toLowerCase()
    .replaceAll("_", " ");
}

function declaredLabel(value: string): string {
  return value === "" ? "undeclared" : value;
}

function confidence(value: number): string {
  return `${new Intl.NumberFormat("en-GB", { maximumFractionDigits: 2 }).format(value / 10_000)}%`;
}

function shortId(value: string): string {
  return `${value.slice(0, 8)}…${value.slice(-6)}`;
}

function relationTone(value: string): string {
  if (value === "RELATION_TYPE_CONTRADICTS") return "contradicts";
  if (value === "RELATION_TYPE_SUPPORTS") return "supports";
  return "structural";
}

export interface KnowledgeGeometryRenderOptions {
  onRefresh?: () => void | Promise<void>;
}

export function renderKnowledgeGeometry(
  root: HTMLElement,
  snapshot: KnowledgeGeometrySnapshot,
  options: KnowledgeGeometryRenderOptions = {},
): void {
  const layout = buildKnowledgeGeometryLayout(snapshot.facts);
  const planeSize = knowledgeGeometryPlaneSize(layout.points);
  const factsById = new Map(snapshot.facts.map((fact) => [fact.id, fact]));
  const pointsById = new Map(layout.points.map((point) => [point.factId, point]));
  const relationsByFact = new Map<string, KnowledgeGeometryRelation[]>();
  snapshot.relations.forEach((relation) => {
    for (const id of new Set([relation.sourceFactId, relation.targetFactId])) {
      const entries = relationsByFact.get(id) ?? [];
      entries.push(relation);
      relationsByFact.set(id, entries);
    }
  });

  const shell = element("article", "knowledge-geometry-shell");
  const toolbar = element("div", "knowledge-geometry-toolbar");
  const sourceState = element("div", "knowledge-geometry-source");
  const sourceDot = element("span", "knowledge-geometry-source-dot");
  sourceDot.setAttribute("aria-hidden", "true");
  sourceState.append(
    sourceDot,
    element(
      "span",
      "",
      `zerone-1 · height ${BigInt(snapshot.source.blockHeight).toLocaleString("en-GB")}${snapshot.source.catchingUp ? " · node catching up" : ""}`,
    ),
  );
  if (snapshot.source.catchingUp) sourceState.dataset.state = "catching-up";
  const refresh = element("button", "button button-ghost", "Refresh snapshot");
  refresh.type = "button";
  if (options.onRefresh) {
    refresh.addEventListener("click", () => void options.onRefresh?.());
  } else {
    refresh.hidden = true;
  }
  toolbar.append(sourceState, refresh);

  const controls = element("div", "knowledge-geometry-controls");
  const searchLabel = element("label");
  const search = element("input", "knowledge-geometry-search");
  search.type = "search";
  search.placeholder = "Search record, domain, ID";
  search.autocomplete = "off";
  search.setAttribute("aria-label", "Search returned knowledge records");
  searchLabel.append(search);
  const domainLabel = element("label");
  const domain = element("select", "knowledge-geometry-domain");
  domain.setAttribute("aria-label", "Filter by knowledge domain");
  const allDomains = element("option", "", "All returned domains");
  allDomains.value = "";
  domain.append(allDomains);
  layout.domains.forEach((entry) => {
    const option = element(
      "option",
      "",
      `${declaredLabel(entry.id).replaceAll("_", " ")} · ${entry.factCount}`,
    );
    option.value = entry.id;
    domain.append(option);
  });
  domainLabel.append(domain);
  controls.append(searchLabel, domainLabel);

  const factsMetric = element("strong", "", String(snapshot.facts.length));
  const domainsMetric = element("strong", "", String(layout.domains.length));
  const relationsMetric = element("strong", "", String(snapshot.relations.length));
  const summary = element("dl", "knowledge-geometry-summary");
  for (const [label, node, note] of [
    ["Returned records", factsMetric, "completeness not claimed"],
    ["Populated domains", domainsMetric, "layout groups only"],
    ["Declared relations", relationsMetric, "typed edges only"],
  ] as const) {
    const item = element("div");
    item.append(element("dt", "", label), node, element("dd", "", note));
    summary.append(item);
  }

  const body = element("div", "knowledge-geometry-body");
  const mapPanel = element("div", "knowledge-geometry-map-panel");
  const mapScroll = element("div", "knowledge-geometry-map-scroll");
  const map = element("div", "knowledge-geometry-map");
  map.style.setProperty(
    "--kg-plane-size",
    `${planeSize}px`,
  );
  map.setAttribute("role", "group");
  map.setAttribute(
    "aria-label",
    "Returned zerone-1 knowledge records grouped by domain; only declared relations are drawn",
  );
  const relationSvg = svgElement("svg");
  relationSvg.setAttribute("class", "knowledge-geometry-relations");
  relationSvg.setAttribute("viewBox", "0 0 100 100");
  relationSvg.setAttribute("preserveAspectRatio", "none");
  relationSvg.setAttribute("aria-hidden", "true");
  map.append(relationSvg);

  layout.domains.forEach((entry) => {
    const halo = element("div", "knowledge-geometry-domain-halo");
    halo.dataset.domainIndex = String(entry.index % 8);
    halo.style.setProperty("--kg-x", `${entry.x}%`);
    halo.style.setProperty("--kg-y", `${entry.y}%`);
    halo.setAttribute("aria-hidden", "true");
    const label = element(
      "span",
      "knowledge-geometry-domain-label",
      declaredLabel(entry.id).replaceAll("_", " "),
    );
    label.style.setProperty("--kg-x", `${entry.x}%`);
    label.style.setProperty("--kg-y", `${entry.y}%`);
    label.dataset.domainIndex = String(entry.index % 8);
    map.append(halo, label);
  });

  const nodeById = new Map<string, HTMLButtonElement>();
  layout.points.forEach((point) => {
    const fact = factsById.get(point.factId);
    if (!fact) return;
    const node = element("button", "knowledge-geometry-node");
    node.type = "button";
    node.tabIndex = -1;
    node.dataset.factId = fact.id;
    node.dataset.domain = fact.domain;
    node.dataset.domainIndex = String(point.domainIndex % 8);
    node.dataset.status = humanEnum(fact.status, "FACT_STATUS_").replaceAll(" ", "-");
    node.style.setProperty("--kg-x", `${point.x}%`);
    node.style.setProperty("--kg-y", `${point.y}%`);
    node.setAttribute(
      "aria-label",
      `${declaredLabel(fact.domain)}: ${fact.content.slice(0, 140)}${fact.content.length > 140 ? "…" : ""}`,
    );
    node.setAttribute("aria-controls", "knowledge-geometry-inspector");
    node.title = `${fact.domain} · ${shortId(fact.id)}`;
    map.append(node);
    nodeById.set(fact.id, node);
  });

  const geometryNote = element("p", "knowledge-geometry-map-note");
  mapScroll.append(map);
  mapPanel.append(mapScroll, geometryNote);

  const inspector = element("aside", "knowledge-geometry-inspector");
  inspector.setAttribute("id", "knowledge-geometry-inspector");
  inspector.setAttribute("aria-live", "polite");

  const resultStatus = element("p", "knowledge-geometry-result-status");
  resultStatus.setAttribute("id", "knowledge-geometry-result-status");
  resultStatus.setAttribute("role", "status");
  resultStatus.setAttribute("aria-live", "polite");
  resultStatus.setAttribute("aria-atomic", "true");
  map.setAttribute("aria-describedby", "knowledge-geometry-result-status");

  let selectedId = snapshot.facts[0]?.id ?? "";
  const inspect = (id: string): void => {
    const fact = factsById.get(id);
    if (!fact) return;
    selectedId = id;
    nodeById.forEach((node, factId) => {
      node.setAttribute("aria-pressed", String(factId === id));
    });
    inspector.replaceChildren();
    const head = element("div", "knowledge-geometry-inspector-head");
    head.append(
      element(
        "span",
        "knowledge-geometry-domain-chip",
        declaredLabel(fact.domain).replaceAll("_", " "),
      ),
      element("span", "knowledge-geometry-status-chip", humanEnum(fact.status, "FACT_STATUS_")),
    );
    const title = element("h3", "", "Recorded proposition");
    const content = element("p", "knowledge-geometry-fact-content", fact.content);
    const metrics = element("dl", "knowledge-geometry-fact-metrics");
    const metricEntries = [
      ["Protocol confidence", confidence(fact.confidence)],
      ["Claim shape", humanEnum(fact.claimType, "CLAIM_TYPE_")],
      ["Category", declaredLabel(fact.category).replaceAll("_", " ")],
      [
        "Method",
        fact.methodId || "empty · source protocol treats as M-LEGACY",
      ],
      [
        "Verified at",
        fact.verifiedAtBlock === "0"
          ? "legacy metadata absent"
          : `block ${BigInt(fact.verifiedAtBlock).toLocaleString("en-GB")}`,
      ],
      ["Energy", `${fact.energy.toLocaleString("en-GB")} / ${fact.energyCap.toLocaleString("en-GB")}`],
    ];
    metricEntries.forEach(([label, value]) => {
      const item = element("div");
      item.append(element("dt", "", label), element("dd", "", value));
      metrics.append(item);
    });
    const relationTitle = element("h4", "", "Declared edges");
    const relationList = element("ul", "knowledge-geometry-edge-list");
    const related = relationsByFact.get(id) ?? [];
    if (related.length === 0) {
      relationList.append(
        element(
          "li",
          "knowledge-geometry-empty-edge",
          "None in this returned snapshot. Isolation is shown, not repaired by inference.",
        ),
      );
    } else {
      related.forEach((relation) => {
        const outgoing = relation.sourceFactId === id;
        const otherId = outgoing ? relation.targetFactId : relation.sourceFactId;
        const other = factsById.get(otherId);
        const item = element("li");
        item.dataset.tone = relationTone(relation.relation);
        item.append(
          element(
            "span",
            "",
            `${outgoing ? "→" : "←"} ${humanEnum(relation.relation, "RELATION_TYPE_")}`,
          ),
          element(
            "strong",
            "",
            other
              ? `${declaredLabel(other.domain)} · ${shortId(other.id)}`
              : `outside view · ${shortId(otherId)}`,
          ),
          element(
            "small",
            "",
            `${humanEnum(relation.inference, "INFERENCE_TYPE_")} · strength ${confidence(relation.inferenceStrengthBps)} · ${relation.createdAtBlock === "0" ? "creation height absent (legacy)" : `block ${BigInt(relation.createdAtBlock).toLocaleString("en-GB")}`} · method ${relation.methodId || "empty · inherits by source protocol"}`,
          ),
        );
        relationList.append(item);
      });
    }
    const idLine = element("p", "knowledge-geometry-id-line");
    idLine.append(element("span", "", "Fact ID"), element("code", "", fact.id));
    inspector.append(head, title, content, metrics, relationTitle, relationList, idLine);
  };

  let rovingId = selectedId;
  const visibleNodeEntries = (): Array<[string, HTMLButtonElement]> =>
    [...nodeById.entries()].filter(([, node]) => !node.hidden);
  const setRovingTabStop = (id: string): void => {
    rovingId = id;
    nodeById.forEach((node, factId) => {
      node.tabIndex = id !== "" && factId === id && !node.hidden ? 0 : -1;
    });
  };
  const selectNode = (id: string, moveFocus: boolean): void => {
    const node = nodeById.get(id);
    if (!node || node.hidden) return;
    setRovingTabStop(id);
    inspect(id);
    if (moveFocus) {
      node.focus({ preventScroll: true });
      node.scrollIntoView({ block: "nearest", inline: "nearest" });
    }
  };

  nodeById.forEach((node, id) => {
    node.addEventListener("click", () => selectNode(id, false));
    node.addEventListener("keydown", (event) => {
      if (
        event.key !== "ArrowLeft" &&
        event.key !== "ArrowUp" &&
        event.key !== "ArrowRight" &&
        event.key !== "ArrowDown" &&
        event.key !== "Home" &&
        event.key !== "End"
      ) {
        return;
      }
      const visible = visibleNodeEntries();
      if (visible.length === 0) return;
      event.preventDefault();
      const currentIndex = Math.max(
        0,
        visible.findIndex(([factId]) => factId === id),
      );
      let nextIndex: number;
      if (event.key === "Home") nextIndex = 0;
      else if (event.key === "End") nextIndex = visible.length - 1;
      else {
        const direction =
          event.key === "ArrowLeft" || event.key === "ArrowUp" ? -1 : 1;
        nextIndex = (currentIndex + direction + visible.length) % visible.length;
      }
      const nextId = visible[nextIndex]?.[0];
      if (nextId !== undefined) selectNode(nextId, true);
    });
  });

  const drawRelations = (visibleIds: Set<string>): void => {
    relationSvg.replaceChildren();
    const definitions = svgElement("defs");
    for (const tone of ["structural", "supports", "contradicts"] as const) {
      const marker = svgElement("marker");
      marker.id = `knowledge-geometry-arrow-${tone}`;
      marker.setAttribute("viewBox", "0 0 6 6");
      marker.setAttribute("refX", "5.5");
      marker.setAttribute("refY", "3");
      marker.setAttribute("markerWidth", "2");
      marker.setAttribute("markerHeight", "2");
      marker.setAttribute("orient", "auto");
      marker.setAttribute("markerUnits", "userSpaceOnUse");
      const arrow = svgElement("path");
      arrow.setAttribute("d", "M 0 0 L 6 3 L 0 6 Z");
      arrow.setAttribute(
        "class",
        `knowledge-geometry-arrow knowledge-geometry-arrow-${tone}`,
      );
      marker.append(arrow);
      definitions.append(marker);
    }
    relationSvg.append(definitions);
    snapshot.relations.forEach((relation) => {
      if (
        !visibleIds.has(relation.sourceFactId) ||
        !visibleIds.has(relation.targetFactId)
      ) {
        return;
      }
      const source = pointsById.get(relation.sourceFactId);
      const target = pointsById.get(relation.targetFactId);
      if (!source || !target) return;
      const line = svgElement("line");
      line.setAttribute("x1", String(source.x));
      line.setAttribute("y1", String(source.y));
      line.setAttribute("x2", String(target.x));
      line.setAttribute("y2", String(target.y));
      line.setAttribute(
        "class",
        `knowledge-geometry-edge knowledge-geometry-edge-${relationTone(relation.relation)}`,
      );
      line.setAttribute(
        "marker-end",
        `url(#knowledge-geometry-arrow-${relationTone(relation.relation)})`,
      );
      relationSvg.append(line);
    });
  };

  const applyFilter = (): void => {
    const needle = search.value.trim().toLocaleLowerCase("en-GB");
    const selectedDomain =
      domain.selectedIndex === 0 ? null : (domain.options[domain.selectedIndex]?.value ?? null);
    const visibleIds = new Set<string>();
    snapshot.facts.forEach((fact) => {
      const matchesDomain = selectedDomain === null || fact.domain === selectedDomain;
      const matchesSearch =
        needle === "" ||
        fact.id.toLocaleLowerCase("en-GB").includes(needle) ||
        fact.domain.toLocaleLowerCase("en-GB").includes(needle) ||
        fact.category.toLocaleLowerCase("en-GB").includes(needle) ||
        fact.content.toLocaleLowerCase("en-GB").includes(needle);
      const visible = matchesDomain && matchesSearch;
      const node = nodeById.get(fact.id);
      if (node) node.hidden = !visible;
      if (visible) visibleIds.add(fact.id);
    });
    factsMetric.textContent = String(visibleIds.size);
    const visibleDomains = new Set(
      snapshot.facts
        .filter((fact) => visibleIds.has(fact.id))
        .map((fact) => fact.domain),
    ).size;
    domainsMetric.textContent = String(visibleDomains);
    const visibleRelations = snapshot.relations.filter(
      (relation) =>
        visibleIds.has(relation.sourceFactId) && visibleIds.has(relation.targetFactId),
    ).length;
    relationsMetric.textContent = String(visibleRelations);
    drawRelations(visibleIds);
    geometryNote.textContent =
      visibleIds.size === 0
        ? "No returned record is visible under the current local filters. No broader search or inference was performed."
        : visibleRelations === 0
        ? "No typed FactRelation edge joins the visible records. Domain proximity is layout only—not evidence, agreement, or inferred meaning."
        : `${visibleRelations} declared ${visibleRelations === 1 ? "edge is" : "edges are"} drawn with direction and type; proximity still carries no semantic meaning.`;
    resultStatus.textContent = `${visibleIds.size} of ${snapshot.facts.length} returned ${snapshot.facts.length === 1 ? "record" : "records"} visible across ${visibleDomains} ${visibleDomains === 1 ? "domain" : "domains"}; ${visibleRelations} declared ${visibleRelations === 1 ? "relation" : "relations"} visible. Arrow keys move between visible record nodes; Home and End jump to the first and last.`;
    if (!visibleIds.has(selectedId)) {
      const next = snapshot.facts.find((fact) => visibleIds.has(fact.id));
      if (next) {
        inspect(next.id);
      } else {
        selectedId = "";
        nodeById.forEach((node) => node.setAttribute("aria-pressed", "false"));
        inspector.replaceChildren(
          element("h3", "", "No records in this view"),
          element(
            "p",
            "knowledge-geometry-empty-edge",
            "The current local filters match no returned record. No broader search or inference was performed.",
          ),
        );
      }
    }
    const nextRovingId = visibleIds.has(rovingId)
      ? rovingId
      : visibleIds.has(selectedId)
        ? selectedId
        : (visibleNodeEntries()[0]?.[0] ?? "");
    setRovingTabStop(nextRovingId);
  };
  search.addEventListener("input", applyFilter);
  domain.addEventListener("change", applyFilter);

  body.append(mapPanel, inspector);
  const boundary = element("div", "knowledge-geometry-boundary");
  boundary.append(
    element("strong", "", "A map of records, not minds."),
    element(
      "p",
      "",
      "Equal-size nodes prevent protocol confidence, energy, wealth, identity, or popularity from becoming visual worth. A Fact status is a chain record—not a truth certificate. This read creates no query receipt, transaction, KARMA, qualification, reward, authority, or claim that anyone understands anyone else.",
    ),
  );
  const footer = element("div", "knowledge-geometry-footer");
  const raw = element("a", "button button-ghost", "Inspect bounded JSON ↗");
  raw.href = KNOWLEDGE_GEOMETRY_ENDPOINT;
  footer.append(
    raw,
    element(
      "span",
      "",
      snapshot.source.truncated
        ? `${snapshot.source.returnedRecords} of ${snapshot.source.upstreamRecords} upstream records returned · completeness not claimed`
        : `${snapshot.source.returnedRecords} upstream records returned · completeness still not claimed`,
    ),
  );
  shell.append(toolbar, controls, resultStatus, summary, body, boundary, footer);
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
  if (selectedId) inspect(selectedId);
  applyFilter();
  const selectedPoint = pointsById.get(selectedId);
  const planeWidth = map.scrollWidth;
  const planeHeight = map.scrollHeight;
  const viewportWidth = mapScroll.clientWidth;
  const viewportHeight = mapScroll.clientHeight;
  if (
    Number.isFinite(planeWidth) &&
    Number.isFinite(planeHeight) &&
    Number.isFinite(viewportWidth) &&
    Number.isFinite(viewportHeight)
  ) {
    const targetX = planeWidth * ((selectedPoint?.x ?? 50) / 100);
    const targetY = planeHeight * ((selectedPoint?.y ?? 50) / 100);
    mapScroll.scrollLeft = clamp(
      targetX - viewportWidth / 2,
      0,
      Math.max(0, planeWidth - viewportWidth),
    );
    mapScroll.scrollTop = clamp(
      targetY - viewportHeight / 2,
      0,
      Math.max(0, planeHeight - viewportHeight),
    );
  }
}

function renderKnowledgeGeometryError(
  root: HTMLElement,
  message: string,
  onRetry: () => void | Promise<void>,
): void {
  const state = element("div", "knowledge-geometry-error");
  state.setAttribute("role", "alert");
  state.setAttribute("aria-live", "assertive");
  state.setAttribute("aria-atomic", "true");
  const retry = element("button", "button button-ghost", "Retry bounded read");
  retry.type = "button";
  retry.addEventListener("click", async () => {
    retry.disabled = true;
    retry.textContent = "Retrying bounded read…";
    await onRetry();
  });
  state.append(
    element("span", "knowledge-geometry-error-mark", "∅"),
    element("h3", "", "The geometry refused an unverified snapshot."),
    element("p", "", message),
    element(
      "p",
      "",
      "No records or relations have been inferred. The rest of zerone.ai remains available.",
    ),
    retry,
  );
  root.replaceChildren(state);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseKnowledgeGeometry(
  root: HTMLElement,
  options: KnowledgeGeometryFetchOptions = {},
): Promise<void> {
  let loadEpoch = 0;
  const load = async (): Promise<void> => {
    const epoch = ++loadEpoch;
    root.setAttribute("aria-busy", "true");
    try {
      const snapshot = await fetchKnowledgeGeometry(options);
      if (epoch !== loadEpoch) return;
      renderKnowledgeGeometry(root, snapshot, { onRefresh: load });
    } catch (error) {
      if (epoch !== loadEpoch) return;
      renderKnowledgeGeometryError(
        root,
        error instanceof KnowledgeGeometryDataError
          ? error.message.replace(/^\$:\s*/u, "")
          : "The read-only projection is unavailable.",
        load,
      );
    }
  };
  await load();
}
