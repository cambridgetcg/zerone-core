export interface KnowledgePagesContext {
  request: Request;
  waitUntil(promise: Promise<unknown>): void;
}

type KnowledgeFetch = (
  input: string | URL | Request,
  init?: RequestInit,
) => Promise<Response>;

export interface KnowledgeCache {
  match(request: Request): Promise<Response | undefined>;
  put(request: Request, response: Response): Promise<void>;
}

export interface KnowledgeRuntime {
  fetch: KnowledgeFetch;
  cache: KnowledgeCache;
  upstreams: Readonly<{
    rest: string;
    rpc: string;
  }>;
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
  schema: "zerone.knowledge-geometry-snapshot/v0";
  source: {
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
  };
  facts: KnowledgeGeometryFact[];
  relations: KnowledgeGeometryRelation[];
}

type JsonRecord = Record<string, unknown>;

export const KNOWLEDGE_SCHEMA = "zerone.knowledge-geometry-snapshot/v0" as const;
export const KNOWLEDGE_FACTS_QUERY_PATH =
  "/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true" as const;
export const KNOWLEDGE_FACT_CAP = 128;
export const KNOWLEDGE_RELATION_CAP = 512;
export const KNOWLEDGE_FACTS_BODY_MAX_BYTES = 384 * 1024;
export const KNOWLEDGE_STATUS_BODY_MAX_BYTES = 64 * 1024;
export const KNOWLEDGE_OUTPUT_MAX_BYTES = 256 * 1024;

const EXPECTED_CHAIN_ID = "zerone-1" as const;
const ENDPOINT_PATH = "/api/knowledge";
const STATUS_PATH = "/status";
const UPSTREAM_TIMEOUT_MS = 8_000;
const MAX_HEIGHT_DELTA = 128n;
const MAX_UINT64 = 18_446_744_073_709_551_615n;
const MAX_METRIC = 1_000_000;
const MAX_CONTENT_BYTES = 16_384;
const MAX_LABEL_BYTES = 128;
const MAX_METHOD_ID_BYTES = 128;
const MAX_UPSTREAM_RECORDS = 8_192;
const FACT_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const FORBIDDEN_TEXT_CONTROL = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u;
const FORBIDDEN_BIDI_CONTROL = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/u;

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

const DEFAULT_UPSTREAMS = {
  // This independently read-only HTTPS edge exposes only bounded query paths;
  // the dashboard never needs a direct route to the signer node.
  rest: "https://zerone-rpc.fly.dev/rest",
  rpc: "https://zerone-rpc.fly.dev/rpc",
} as const;

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, HEAD, OPTIONS",
  "Access-Control-Max-Age": "86400",
};

const SECURITY_HEADERS = {
  "Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
  "Referrer-Policy": "no-referrer",
  "X-Content-Type-Options": "nosniff",
};

const SUCCESS_CACHE_CONTROL = "public, max-age=30, s-maxage=60";

class KnowledgeUpstreamError extends Error {}

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function responseHeaders(cacheControl: string): Headers {
  return new Headers({
    ...CORS_HEADERS,
    ...SECURITY_HEADERS,
    "Cache-Control": cacheControl,
    "Content-Type": "application/json; charset=utf-8",
    "X-Zerone-Edge": "knowledge-geometry",
  });
}

function errorResponse(message: string, status: number, head: boolean): Response {
  const headers = responseHeaders("no-store");
  headers.set("Allow", "GET, HEAD, OPTIONS");
  const serialized = JSON.stringify({ error: message });
  headers.set("Content-Length", String(utf8Length(serialized)));
  return new Response(head ? null : serialized, {
    status,
    headers,
  });
}

function optionsResponse(): Response {
  const headers = new Headers({
    ...CORS_HEADERS,
    ...SECURITY_HEADERS,
    Allow: "GET, HEAD, OPTIONS",
    "Cache-Control": "no-store",
  });
  return new Response(null, { status: 204, headers });
}

function utf8Length(value: string): number {
  return new TextEncoder().encode(value).byteLength;
}

function boundedText(
  value: unknown,
  maximumBytes: number,
  allowEmpty = false,
): string {
  if (
    typeof value !== "string" ||
    (!allowEmpty && value.trim().length === 0) ||
    FORBIDDEN_TEXT_CONTROL.test(value) ||
    FORBIDDEN_BIDI_CONTROL.test(value) ||
    utf8Length(value) > maximumBytes
  ) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid fact");
  }
  return value;
}

function factId(value: unknown, relation = false): string {
  if (typeof value !== "string" || !FACT_ID.test(value)) {
    throw new KnowledgeUpstreamError(
      relation
        ? "Mainnet facts response contains an invalid relation"
        : "Mainnet facts response contains an invalid fact",
    );
  }
  return value;
}

function enumValue(
  value: unknown,
  allowed: ReadonlySet<string>,
  relation = false,
): string {
  if (typeof value !== "string" || !allowed.has(value)) {
    throw new KnowledgeUpstreamError(
      relation
        ? "Mainnet facts response contains an invalid relation"
        : "Mainnet facts response contains an invalid fact",
    );
  }
  return value;
}

function safeMetric(value: unknown, relation = false): number {
  let parsed: number;
  if (typeof value === "number") {
    parsed = value;
  } else if (
    typeof value === "string" &&
    /^(?:0|[1-9]\d*)$/.test(value)
  ) {
    parsed = Number(value);
  } else {
    throw new KnowledgeUpstreamError(
      relation
        ? "Mainnet facts response contains an invalid relation"
        : "Mainnet facts response contains an invalid fact",
    );
  }
  if (!Number.isSafeInteger(parsed) || parsed < 0 || parsed > MAX_METRIC) {
    throw new KnowledgeUpstreamError(
      relation
        ? "Mainnet facts response contains an invalid relation"
        : "Mainnet facts response contains an invalid fact",
    );
  }
  return parsed;
}

function canonicalHeight(
  value: unknown,
  errorMessage: string,
  positive = false,
): string {
  if (
    typeof value !== "string" ||
    !/^(?:0|[1-9]\d{0,19})$/.test(value)
  ) {
    throw new KnowledgeUpstreamError(errorMessage);
  }
  const parsed = BigInt(value);
  if (parsed > MAX_UINT64 || (positive && parsed === 0n)) {
    throw new KnowledgeUpstreamError(errorMessage);
  }
  return value;
}

function heightAtMost(value: string, maximum: string, message: string): void {
  if (BigInt(value) > BigInt(maximum)) {
    throw new KnowledgeUpstreamError(message);
  }
}

function parseRelation(
  value: unknown,
  anchorFactId: string,
  direction: "outgoing" | "incoming",
  blockHeight: string,
  statusHeight: string,
): KnowledgeGeometryRelation {
  if (!isRecord(value)) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid relation");
  }
  const sourceFactId = factId(value.sourceFactId, true);
  const targetFactId = factId(value.targetFactId, true);
  if (
    (direction === "outgoing" && sourceFactId !== anchorFactId) ||
    (direction === "incoming" && targetFactId !== anchorFactId)
  ) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid relation");
  }
  const createdAtBlock = canonicalHeight(
    value.createdAtBlock,
    "Mainnet facts response contains an invalid relation",
  );
  heightAtMost(
    createdAtBlock,
    blockHeight,
    "Mainnet facts response contains a future relation",
  );
  heightAtMost(
    createdAtBlock,
    statusHeight,
    "Mainnet facts response contains a future relation",
  );
  const methodId = boundedText(value.methodId, MAX_METHOD_ID_BYTES, true);
  if (methodId !== "" && !FACT_ID.test(methodId)) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid relation");
  }
  return {
    sourceFactId,
    targetFactId,
    relation: enumValue(value.relation, RELATION_TYPES, true),
    inference: enumValue(value.inference, INFERENCE_TYPES, true),
    inferenceStrengthBps: safeMetric(value.inferenceStrengthBps, true),
    createdAtBlock,
    methodId,
  };
}

function parseFact(
  value: unknown,
  blockHeight: string,
  statusHeight: string,
): {
  fact: KnowledgeGeometryFact;
  relations: KnowledgeGeometryRelation[];
} {
  if (!isRecord(value)) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid fact");
  }
  const id = factId(value.id);
  const verifiedAtBlock = canonicalHeight(
    value.verifiedAtBlock,
    "Mainnet facts response contains an invalid fact height",
  );
  const lastVerifiedBlock = canonicalHeight(
    value.lastVerifiedBlock,
    "Mainnet facts response contains an invalid fact height",
  );
  heightAtMost(
    verifiedAtBlock,
    blockHeight,
    "Mainnet facts response contains a future fact height",
  );
  heightAtMost(
    lastVerifiedBlock,
    blockHeight,
    "Mainnet facts response contains a future fact height",
  );
  heightAtMost(
    verifiedAtBlock,
    statusHeight,
    "Mainnet facts response contains a future fact height",
  );
  heightAtMost(
    lastVerifiedBlock,
    statusHeight,
    "Mainnet facts response contains a future fact height",
  );
  heightAtMost(
    verifiedAtBlock,
    lastVerifiedBlock,
    "Mainnet facts response contains inconsistent fact heights",
  );
  const methodId = boundedText(value.methodId, MAX_METHOD_ID_BYTES, true);
  if (methodId !== "" && !FACT_ID.test(methodId)) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid fact");
  }
  if (!Array.isArray(value.outgoingRelations) || !Array.isArray(value.incomingRelations)) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains an invalid fact");
  }

  const relations = [
    ...value.outgoingRelations.map((relation) =>
      parseRelation(relation, id, "outgoing", blockHeight, statusHeight),
    ),
    ...value.incomingRelations.map((relation) =>
      parseRelation(relation, id, "incoming", blockHeight, statusHeight),
    ),
  ];

  return {
    fact: {
      id,
      content: boundedText(value.content, MAX_CONTENT_BYTES),
      domain: boundedText(value.domain, MAX_LABEL_BYTES, true),
      category: boundedText(value.category, MAX_LABEL_BYTES, true),
      status: enumValue(value.status, FACT_STATUSES),
      claimType: enumValue(value.claimType, CLAIM_TYPES),
      confidence: safeMetric(value.confidence),
      verifiedAtBlock,
      lastVerifiedBlock,
      energy: safeMetric(value.energy),
      energyCap: safeMetric(value.energyCap),
      fitnessScore: safeMetric(value.fitnessScore),
      methodId,
    },
    relations,
  };
}

function relationIdentity(relation: KnowledgeGeometryRelation): string {
  return [
    relation.sourceFactId,
    relation.targetFactId,
    relation.relation,
    relation.inference,
    String(relation.inferenceStrengthBps),
    relation.createdAtBlock,
    relation.methodId,
  ].join("\u0000");
}

function relationPairIdentity(relation: KnowledgeGeometryRelation): string {
  return `${relation.sourceFactId}\u0000${relation.targetFactId}`;
}

function lexicalCompare(left: string, right: string): number {
  return left < right ? -1 : left > right ? 1 : 0;
}

function maximumFittingPrefix<T>(
  values: readonly T[],
  fits: (prefix: T[]) => boolean,
): T[] {
  let lower = 0;
  let upper = values.length;
  while (lower < upper) {
    const middle = Math.ceil((lower + upper) / 2);
    if (fits(values.slice(0, middle))) lower = middle;
    else upper = middle - 1;
  }
  return values.slice(0, lower);
}

function paginationClaimsMore(value: unknown, observedRecords: number): boolean {
  if (value === null || value === undefined) return false;
  if (!isRecord(value)) {
    throw new KnowledgeUpstreamError("Mainnet facts response has invalid pagination metadata");
  }

  if (value.nextKey !== undefined && value.next_key !== undefined) {
    throw new KnowledgeUpstreamError("Mainnet facts response has invalid pagination metadata");
  }
  const nextKey = value.nextKey ?? value.next_key;
  if (
    nextKey !== undefined &&
    nextKey !== null &&
    (typeof nextKey !== "string" || nextKey.length > 344)
  ) {
    throw new KnowledgeUpstreamError("Mainnet facts response has invalid pagination metadata");
  }

  let totalClaimsMore = false;
  if (value.total !== undefined && value.total !== null) {
    const total = value.total;
    let parsed: number;
    if (typeof total === "number") parsed = total;
    else if (typeof total === "string" && /^(?:0|[1-9]\d*)$/.test(total)) {
      parsed = Number(total);
    } else {
      throw new KnowledgeUpstreamError("Mainnet facts response has invalid pagination metadata");
    }
    if (!Number.isSafeInteger(parsed) || parsed < observedRecords) {
      throw new KnowledgeUpstreamError("Mainnet facts response has invalid pagination metadata");
    }
    totalClaimsMore = parsed > observedRecords;
  }

  return (typeof nextKey === "string" && nextKey.length > 0) || totalClaimsMore;
}

function parseFactsSnapshot(
  value: unknown,
  blockHeight: string,
  statusHeight: string,
  catchingUp: boolean,
): KnowledgeGeometrySnapshot {
  if (!isRecord(value) || !Array.isArray(value.facts)) {
    throw new KnowledgeUpstreamError("Mainnet facts response is incomplete");
  }

  const upstreamRecords = value.facts.length;
  if (upstreamRecords > MAX_UPSTREAM_RECORDS) {
    throw new KnowledgeUpstreamError("Mainnet facts response contains too many records");
  }
  const parsedFacts: Array<{
    fact: KnowledgeGeometryFact;
    relations: KnowledgeGeometryRelation[];
  }> = [];
  const factIds = new Set<string>();

  for (const candidate of value.facts) {
    const parsed = parseFact(candidate, blockHeight, statusHeight);
    if (factIds.has(parsed.fact.id)) {
      throw new KnowledgeUpstreamError("Mainnet facts response contains duplicate fact IDs");
    }
    factIds.add(parsed.fact.id);
    parsedFacts.push(parsed);
  }

  parsedFacts.sort((left, right) => lexicalCompare(left.fact.id, right.fact.id));
  const selected = parsedFacts.slice(0, KNOWLEDGE_FACT_CAP);
  const facts = selected.map(({ fact }) => fact);
  // Validate pair authority across the whole bounded upstream response before
  // selecting the visible prefix. Otherwise a conflicting copy embedded on a
  // fact just after the cap could contradict the copy carried by a retained
  // endpoint without being noticed.
  const relationCandidates = parsedFacts.flatMap(({ relations }) => relations);

  const relationByPair = new Map<
    string,
    { identity: string; relation: KnowledgeGeometryRelation }
  >();
  for (const relation of relationCandidates) {
    const pair = relationPairIdentity(relation);
    const identity = relationIdentity(relation);
    const previous = relationByPair.get(pair);
    if (previous) {
      if (previous.identity !== identity) {
        throw new KnowledgeUpstreamError(
          "Mainnet facts response contains conflicting duplicate relations",
        );
      }
    } else {
      relationByPair.set(pair, { identity, relation });
    }
  }
  const allRelations = [...relationByPair.values()]
    .sort((left, right) => lexicalCompare(left.identity, right.identity))
    .map(({ relation }) => relation);
  const selectedFactIds = new Set(facts.map(({ id }) => id));
  const selectedRelations = allRelations.filter(
    ({ sourceFactId, targetFactId }) =>
      selectedFactIds.has(sourceFactId) || selectedFactIds.has(targetFactId),
  );

  const paginationTruncated = paginationClaimsMore(value.pagination, upstreamRecords);
  const baseTruncated =
    upstreamRecords > KNOWLEDGE_FACT_CAP ||
    selectedRelations.length > KNOWLEDGE_RELATION_CAP ||
    paginationTruncated;
  const source: KnowledgeGeometrySnapshot["source"] = {
    chainId: EXPECTED_CHAIN_ID,
    blockHeight,
    statusHeight,
    catchingUp,
    queryPath: KNOWLEDGE_FACTS_QUERY_PATH,
    queryTracked: false,
    writes: false,
    completeness: "NOT_CLAIMED",
    upstreamRecords,
    returnedRecords: facts.length,
    truncated: baseTruncated,
  };
  const assemble = (
    boundedFacts: KnowledgeGeometryFact[],
    boundedRelations: KnowledgeGeometryRelation[],
    outputTruncated: boolean,
  ): KnowledgeGeometrySnapshot => ({
    schema: KNOWLEDGE_SCHEMA,
    source: {
      ...source,
      returnedRecords: boundedFacts.length,
      truncated: outputTruncated,
    },
    facts: boundedFacts,
    relations: boundedRelations,
  });
  const fits = (snapshot: KnowledgeGeometrySnapshot): boolean =>
    utf8Length(JSON.stringify(snapshot)) <= KNOWLEDGE_OUTPUT_MAX_BYTES;

  let boundedFacts = facts;
  if (!fits(assemble(boundedFacts, [], baseTruncated))) {
    boundedFacts = maximumFittingPrefix(facts, (prefix) =>
      fits(assemble(prefix, [], true)),
    );
  }
  const boundedFactIds = new Set(boundedFacts.map(({ id }) => id));
  const eligibleRelations = allRelations.filter(
    ({ sourceFactId, targetFactId }) =>
      boundedFactIds.has(sourceFactId) || boundedFactIds.has(targetFactId),
  );
  const cappedRelations = eligibleRelations.slice(0, KNOWLEDGE_RELATION_CAP);
  const outputAlreadyTruncated =
    baseTruncated ||
    boundedFacts.length !== facts.length ||
    eligibleRelations.length > KNOWLEDGE_RELATION_CAP;
  const boundedRelations = maximumFittingPrefix(cappedRelations, (prefix) =>
    fits(
      assemble(
        boundedFacts,
        prefix,
        outputAlreadyTruncated || prefix.length !== eligibleRelations.length,
      ),
    ),
  );
  return assemble(
    boundedFacts,
    boundedRelations,
    outputAlreadyTruncated || boundedRelations.length !== eligibleRelations.length,
  );
}

function parseStatus(value: unknown): { statusHeight: string; catchingUp: boolean } {
  if (!isRecord(value) || !isRecord(value.result)) {
    throw new KnowledgeUpstreamError("Mainnet status response is incomplete");
  }
  const nodeInfo = value.result.node_info;
  const syncInfo = value.result.sync_info;
  if (!isRecord(nodeInfo) || nodeInfo.network !== EXPECTED_CHAIN_ID) {
    throw new KnowledgeUpstreamError("Mainnet status did not prove chain zerone-1");
  }
  if (!isRecord(syncInfo) || typeof syncInfo.catching_up !== "boolean") {
    throw new KnowledgeUpstreamError("Mainnet status response is incomplete");
  }
  return {
    statusHeight: canonicalHeight(
      syncInfo.latest_block_height,
      "Mainnet status returned an invalid block height",
      true,
    ),
    catchingUp: syncInfo.catching_up,
  };
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
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
  };
  const scanValue = (depth = 0): void => {
    if (depth > 64) {
      throw new KnowledgeUpstreamError("Mainnet response JSON nesting is excessive");
    }
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
        if (raw[offset] !== '"') {
          throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
        }
        const key = scanString();
        if (keys.has(key)) {
          throw new KnowledgeUpstreamError("Mainnet response contains duplicate JSON keys");
        }
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") {
          throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
        }
        offset += 1;
        scanValue(depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") {
          throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
        }
        offset += 1;
      }
      throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        scanValue(depth + 1);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") {
          throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
        }
        offset += 1;
      }
      throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/u.test(raw[offset] ?? "")) {
      offset += 1;
    }
    if (offset === start) {
      throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
    }
  };

  scanValue();
  whitespace();
  if (offset !== raw.length) {
    throw new KnowledgeUpstreamError("Mainnet response contains malformed JSON");
  }
}

function parseBoundedJson(raw: string, label: "facts" | "status"): unknown {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch {
    throw new KnowledgeUpstreamError(`Mainnet ${label} response contains malformed JSON`);
  }
  try {
    rejectDuplicateJsonKeys(raw);
  } catch (error) {
    if (
      error instanceof KnowledgeUpstreamError &&
      error.message.includes("duplicate JSON keys")
    ) {
      throw new KnowledgeUpstreamError(
        `Mainnet ${label} response contains duplicate JSON keys`,
      );
    }
    throw new KnowledgeUpstreamError(`Mainnet ${label} response contains malformed JSON`);
  }
  return parsed;
}

function hasExactKeys(value: JsonRecord, expected: readonly string[]): boolean {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  return (
    actual.length === wanted.length &&
    actual.every((key, index) => key === wanted[index])
  );
}

function normalizedMetric(value: unknown, relation = false): number {
  if (typeof value !== "number") {
    throw new KnowledgeUpstreamError("Cached knowledge projection is not normalized");
  }
  return safeMetric(value, relation);
}

function cachedProjection(raw: string): KnowledgeGeometrySnapshot | null {
  try {
    const value = parseBoundedJson(raw, "facts");
    if (
      !isRecord(value) ||
      JSON.stringify(value) !== raw ||
      !hasExactKeys(value, ["schema", "source", "facts", "relations"]) ||
      value.schema !== KNOWLEDGE_SCHEMA ||
      !isRecord(value.source) ||
      !Array.isArray(value.facts) ||
      !Array.isArray(value.relations)
    ) {
      return null;
    }

    const source = value.source;
    if (
      !hasExactKeys(source, [
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
      ]) ||
      source.chainId !== EXPECTED_CHAIN_ID ||
      source.queryPath !== KNOWLEDGE_FACTS_QUERY_PATH ||
      source.queryTracked !== false ||
      source.writes !== false ||
      source.completeness !== "NOT_CLAIMED" ||
      typeof source.catchingUp !== "boolean" ||
      typeof source.truncated !== "boolean" ||
      typeof source.upstreamRecords !== "number" ||
      !Number.isSafeInteger(source.upstreamRecords) ||
      source.upstreamRecords < 0 ||
      source.upstreamRecords > MAX_UPSTREAM_RECORDS ||
      typeof source.returnedRecords !== "number" ||
      !Number.isSafeInteger(source.returnedRecords) ||
      source.returnedRecords < 0 ||
      source.returnedRecords > KNOWLEDGE_FACT_CAP ||
      source.returnedRecords > source.upstreamRecords ||
      (!source.truncated && source.returnedRecords !== source.upstreamRecords)
    ) {
      return null;
    }
    const blockHeight = canonicalHeight(
      source.blockHeight,
      "Cached knowledge projection has an invalid block height",
      true,
    );
    const statusHeight = canonicalHeight(
      source.statusHeight,
      "Cached knowledge projection has an invalid status height",
      true,
    );
    const heightDelta =
      BigInt(blockHeight) >= BigInt(statusHeight)
        ? BigInt(blockHeight) - BigInt(statusHeight)
        : BigInt(statusHeight) - BigInt(blockHeight);
    if (heightDelta > MAX_HEIGHT_DELTA || value.facts.length !== source.returnedRecords) {
      return null;
    }

    const factKeys = [
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
    ] as const;
    const factIds = new Set<string>();
    let previousFactId: string | undefined;
    for (const candidate of value.facts) {
      if (!isRecord(candidate) || !hasExactKeys(candidate, factKeys)) return null;
      const id = factId(candidate.id);
      if (factIds.has(id) || (previousFactId !== undefined && previousFactId >= id)) {
        return null;
      }
      factIds.add(id);
      previousFactId = id;
      boundedText(candidate.content, MAX_CONTENT_BYTES);
      boundedText(candidate.domain, MAX_LABEL_BYTES, true);
      boundedText(candidate.category, MAX_LABEL_BYTES, true);
      enumValue(candidate.status, FACT_STATUSES);
      enumValue(candidate.claimType, CLAIM_TYPES);
      normalizedMetric(candidate.confidence);
      normalizedMetric(candidate.energy);
      normalizedMetric(candidate.energyCap);
      normalizedMetric(candidate.fitnessScore);
      const methodId = boundedText(candidate.methodId, MAX_METHOD_ID_BYTES, true);
      if (methodId !== "" && !FACT_ID.test(methodId)) return null;
      const verifiedAtBlock = canonicalHeight(
        candidate.verifiedAtBlock,
        "Cached knowledge projection has an invalid fact height",
      );
      const lastVerifiedBlock = canonicalHeight(
        candidate.lastVerifiedBlock,
        "Cached knowledge projection has an invalid fact height",
      );
      if (
        BigInt(verifiedAtBlock) > BigInt(lastVerifiedBlock) ||
        BigInt(verifiedAtBlock) > BigInt(blockHeight) ||
        BigInt(verifiedAtBlock) > BigInt(statusHeight) ||
        BigInt(lastVerifiedBlock) > BigInt(blockHeight) ||
        BigInt(lastVerifiedBlock) > BigInt(statusHeight)
      ) {
        return null;
      }
    }

    if (value.relations.length > KNOWLEDGE_RELATION_CAP) return null;
    const relationKeys = [
      "sourceFactId",
      "targetFactId",
      "relation",
      "inference",
      "inferenceStrengthBps",
      "createdAtBlock",
      "methodId",
    ] as const;
    const relationPairs = new Set<string>();
    let previousRelationIdentity: string | undefined;
    for (const candidate of value.relations) {
      if (!isRecord(candidate) || !hasExactKeys(candidate, relationKeys)) return null;
      const sourceFactId = factId(candidate.sourceFactId, true);
      const targetFactId = factId(candidate.targetFactId, true);
      if (!factIds.has(sourceFactId) && !factIds.has(targetFactId)) return null;
      const relation = enumValue(candidate.relation, RELATION_TYPES, true);
      const inference = enumValue(candidate.inference, INFERENCE_TYPES, true);
      const inferenceStrengthBps = normalizedMetric(
        candidate.inferenceStrengthBps,
        true,
      );
      const createdAtBlock = canonicalHeight(
        candidate.createdAtBlock,
        "Cached knowledge projection has an invalid relation height",
      );
      if (
        BigInt(createdAtBlock) > BigInt(blockHeight) ||
        BigInt(createdAtBlock) > BigInt(statusHeight)
      ) {
        return null;
      }
      const methodId = boundedText(candidate.methodId, MAX_METHOD_ID_BYTES, true);
      if (methodId !== "" && !FACT_ID.test(methodId)) return null;
      const pair = `${sourceFactId}\u0000${targetFactId}`;
      if (relationPairs.has(pair)) return null;
      relationPairs.add(pair);
      const identity = [
        sourceFactId,
        targetFactId,
        relation,
        inference,
        String(inferenceStrengthBps),
        createdAtBlock,
        methodId,
      ].join("\u0000");
      if (previousRelationIdentity !== undefined && previousRelationIdentity >= identity) {
        return null;
      }
      previousRelationIdentity = identity;
    }
    return value as unknown as KnowledgeGeometrySnapshot;
  } catch {
    return null;
  }
}

async function readLimitedBody(response: Response, maximumBytes: number): Promise<string> {
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!/^(?:0|[1-9]\d*)$/.test(declaredLength) ||
      Number(declaredLength) > maximumBytes)
  ) {
    throw new KnowledgeUpstreamError("Mainnet response exceeded its byte limit");
  }

  const reader = response.body?.getReader();
  if (!reader) return "";
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > maximumBytes) {
        void reader.cancel().catch(() => {
          // Refusal does not await an uncooperative upstream stream.
        });
        throw new KnowledgeUpstreamError("Mainnet response exceeded its byte limit");
      }
      chunks.push(value);
    }
  } catch (error) {
    void reader.cancel().catch(() => undefined);
    if (error instanceof KnowledgeUpstreamError) throw error;
    throw new KnowledgeUpstreamError("Mainnet response body could not be read");
  }

  const bytes = new Uint8Array(total);
  let writeOffset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, writeOffset);
    writeOffset += chunk.byteLength;
  }
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new KnowledgeUpstreamError("Mainnet response body is not valid UTF-8");
  }
}

function validJsonMediaType(response: Response): boolean {
  const contentType = response.headers.get("content-type");
  if (contentType === null) return false;
  return contentType.split(";", 1)[0]?.trim().toLowerCase() === "application/json";
}

async function fetchJson(
  runtime: KnowledgeRuntime,
  target: URL,
  label: "facts" | "status",
  maximumBytes: number,
): Promise<unknown> {
  let response: Response;
  try {
    response = await runtime.fetch(target, {
      method: "GET",
      headers: { Accept: "application/json" },
      redirect: "manual",
      signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
    });
  } catch {
    throw new KnowledgeUpstreamError(`Mainnet ${label} endpoint is temporarily unreachable`);
  }

  if (
    response.redirected ||
    (response.status >= 300 && response.status < 400) ||
    (response.url !== "" && response.url !== target.toString())
  ) {
    throw new KnowledgeUpstreamError(`Mainnet ${label} endpoint refused a redirect`);
  }
  if (!response.ok) {
    throw new KnowledgeUpstreamError(`Mainnet ${label} endpoint is temporarily unavailable`);
  }
  if (!validJsonMediaType(response)) {
    throw new KnowledgeUpstreamError(`Mainnet ${label} endpoint returned a non-JSON response`);
  }
  const raw = await readLimitedBody(response, maximumBytes);
  return parseBoundedJson(raw, label);
}

function restBlockHeight(responseValue: string | null): string {
  return canonicalHeight(
    responseValue,
    "Mainnet facts response omitted a valid block height",
    true,
  );
}

function upstreamTarget(base: string, path: string): URL {
  let target: URL;
  try {
    target = new URL(base);
  } catch {
    throw new KnowledgeUpstreamError("Knowledge upstream configuration is invalid");
  }
  if (
    (target.protocol !== "https:" && target.protocol !== "http:") ||
    target.username !== "" ||
    target.password !== "" ||
    target.search !== "" ||
    target.hash !== ""
  ) {
    throw new KnowledgeUpstreamError("Knowledge upstream configuration is invalid");
  }
  const suffix = new URL(path, "https://knowledge-path.invalid");
  target.pathname = `${target.pathname.replace(/\/$/, "")}${suffix.pathname}`;
  target.search = suffix.search;
  return target;
}

async function fetchKnowledgeSnapshot(runtime: KnowledgeRuntime): Promise<KnowledgeGeometrySnapshot> {
  const factsTarget = upstreamTarget(runtime.upstreams.rest, KNOWLEDGE_FACTS_QUERY_PATH);
  const statusTarget = upstreamTarget(runtime.upstreams.rpc, STATUS_PATH);

  const factsResponsePromise = (async (): Promise<Response> => {
    try {
      return await runtime.fetch(factsTarget, {
        method: "GET",
        headers: { Accept: "application/json" },
        redirect: "manual",
        signal: AbortSignal.timeout(UPSTREAM_TIMEOUT_MS),
      });
    } catch {
      throw new KnowledgeUpstreamError("Mainnet facts endpoint is temporarily unreachable");
    }
  })();
  const [factsResponse, statusValue] = await Promise.all([
    factsResponsePromise,
    fetchJson(
      runtime,
      statusTarget,
      "status",
      KNOWLEDGE_STATUS_BODY_MAX_BYTES,
    ),
  ]);

  if (
    factsResponse.redirected ||
    (factsResponse.status >= 300 && factsResponse.status < 400) ||
    (factsResponse.url !== "" && factsResponse.url !== factsTarget.toString())
  ) {
    throw new KnowledgeUpstreamError("Mainnet facts endpoint refused a redirect");
  }
  if (!factsResponse.ok) {
    throw new KnowledgeUpstreamError("Mainnet facts endpoint is temporarily unavailable");
  }
  if (!validJsonMediaType(factsResponse)) {
    throw new KnowledgeUpstreamError("Mainnet facts endpoint returned a non-JSON response");
  }

  const blockHeight = restBlockHeight(
    factsResponse.headers.get("grpc-metadata-x-cosmos-block-height") ??
      factsResponse.headers.get("x-cosmos-block-height"),
  );
  const factsRaw = await readLimitedBody(
    factsResponse,
    KNOWLEDGE_FACTS_BODY_MAX_BYTES,
  );
  const factsValue = parseBoundedJson(factsRaw, "facts");
  const status = parseStatus(statusValue);
  const heightDelta =
    BigInt(blockHeight) >= BigInt(status.statusHeight)
      ? BigInt(blockHeight) - BigInt(status.statusHeight)
      : BigInt(status.statusHeight) - BigInt(blockHeight);
  if (heightDelta > MAX_HEIGHT_DELTA) {
    throw new KnowledgeUpstreamError(
      "Mainnet facts and status block heights are too far apart",
    );
  }
  return parseFactsSnapshot(
    factsValue,
    blockHeight,
    status.statusHeight,
    status.catchingUp,
  );
}

function exactCacheKey(incoming: URL): Request {
  return new Request(new URL(ENDPOINT_PATH, incoming.origin).toString(), {
    method: "GET",
  });
}

function successResponse(body: BodyInit | null, contentLength: number): Response {
  const headers = responseHeaders(SUCCESS_CACHE_CONTROL);
  headers.set("Content-Length", String(contentLength));
  return new Response(body, {
    status: 200,
    headers,
  });
}

export async function knowledgeRequest(
  context: KnowledgePagesContext,
  runtime: KnowledgeRuntime,
): Promise<Response> {
  const method = context.request.method.toUpperCase();
  const head = method === "HEAD";
  let incoming: URL;
  try {
    incoming = new URL(context.request.url);
  } catch {
    return errorResponse("Knowledge endpoint received a malformed URL", 400, head);
  }

  if (incoming.pathname !== ENDPOINT_PATH) {
    return errorResponse("Knowledge endpoint path is invalid", 400, head);
  }
  if (incoming.search !== "") {
    return errorResponse("Knowledge endpoint does not accept query parameters", 400, head);
  }
  if (method === "OPTIONS") return optionsResponse();
  if (method !== "GET" && method !== "HEAD") {
    return errorResponse(
      "Knowledge endpoint supports only GET, HEAD, and OPTIONS",
      405,
      false,
    );
  }

  const cacheKey = exactCacheKey(incoming);
  if (method === "GET") {
    try {
      const cached = await runtime.cache.match(cacheKey);
      if (cached?.ok) {
        const declaredLength = cached.headers.get("content-length");
        if (
          declaredLength !== null &&
          /^(?:0|[1-9]\d*)$/.test(declaredLength) &&
          Number(declaredLength) <= KNOWLEDGE_OUTPUT_MAX_BYTES &&
          validJsonMediaType(cached)
        ) {
          const raw = await readLimitedBody(cached, KNOWLEDGE_OUTPUT_MAX_BYTES);
          const contentLength = utf8Length(raw);
          if (
            contentLength === Number(declaredLength) &&
            cachedProjection(raw) !== null
          ) {
            return successResponse(raw, contentLength);
          }
        }
      }
    } catch {
      // Cache availability never weakens the upstream validation boundary.
    }
  }

  try {
    const snapshot = await fetchKnowledgeSnapshot(runtime);
    const serialized = JSON.stringify(snapshot);
    const contentLength = utf8Length(serialized);
    if (contentLength > KNOWLEDGE_OUTPUT_MAX_BYTES) {
      throw new KnowledgeUpstreamError("Knowledge projection exceeded its byte limit");
    }
    const response = successResponse(head ? null : serialized, contentLength);
    if (method === "GET") {
      try {
        const cacheWrite = runtime.cache
          .put(cacheKey, successResponse(serialized, contentLength))
          .catch(() => undefined);
        context.waitUntil(cacheWrite);
      } catch {
        // A cache write is an optimization, never part of snapshot truth.
      }
    }
    return response;
  } catch (error) {
    const message =
      error instanceof KnowledgeUpstreamError
        ? error.message
        : "Knowledge snapshot could not be assembled safely";
    return errorResponse(message, 502, head);
  }
}

export async function knowledgeMainnet(context: KnowledgePagesContext): Promise<Response> {
  const edgeCache = (globalThis.caches as unknown as { default: KnowledgeCache }).default;
  return knowledgeRequest(context, {
    fetch: globalThis.fetch.bind(globalThis),
    cache: edgeCache,
    upstreams: DEFAULT_UPSTREAMS,
  });
}
