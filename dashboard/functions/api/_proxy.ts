export type ProxyKind = "rpc" | "rest";

export interface PagesContext {
  request: Request;
  params: Record<string, string | string[] | undefined>;
  waitUntil(promise: Promise<unknown>): void;
}

type ProxyFetch = (
  input: string | URL | Request,
  init?: RequestInit,
) => Promise<Response>;

export interface ProxyCache {
  match(request: Request): Promise<Response | undefined>;
  put(request: Request, response: Response): Promise<void>;
}

export interface ProxyRuntime {
  fetch: ProxyFetch;
  cache: ProxyCache;
  upstreams: Readonly<Record<ProxyKind, string>>;
}

type JsonRecord = Record<string, unknown>;

const UPSTREAMS: Record<ProxyKind, string> = {
  // Workers reject fetches to literal IP addresses. Fly's stable hostname
  // resolves to the same public mainnet machine without adding a third party.
  rpc: "http://zerone-1.fly.dev:26657",
  rest: "http://zerone-1.fly.dev:1317",
};

const MAX_RPC_BODY_BYTES = 300_000;
const MAX_STATUS_BODY_BYTES = 64_000;
const EXPECTED_CHAIN_ID = "zerone-1";
const SYNCING_ROUTE = "cosmos/base/tendermint/v1beta1/syncing";
const ZERONE_ACCOUNT_PATTERN = "zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}";
const ZERONE_ACCOUNT = new RegExp(`^${ZERONE_ACCOUNT_PATTERN}$`);
const FEEGRANT_PAIR = new RegExp(
  `^cosmos/feegrant/v1beta1/allowance/(${ZERONE_ACCOUNT_PATTERN})/(${ZERONE_ACCOUNT_PATTERN})$`,
);
const FEEGRANT_LIST = new RegExp(
  `^cosmos/feegrant/v1beta1/allowances/(${ZERONE_ACCOUNT_PATTERN})$`,
);
const BECH32_CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
const BECH32_GENERATORS = [
  0x3b6a57b2,
  0x26508e6d,
  0x1ea119fa,
  0x3d4233dd,
  0x2a1462b3,
] as const;
const ABCI_PATHS = new Set([
  "/cosmos.auth.v1beta1.Query/Account",
  "/cosmos.bank.v1beta1.Query/Balance",
]);

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, HEAD, POST, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type",
  "Access-Control-Max-Age": "86400",
};

const API_HEADERS = {
  ...CORS_HEADERS,
  "Content-Type": "application/json; charset=utf-8",
  "Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
  "Referrer-Policy": "no-referrer",
  "X-Content-Type-Options": "nosniff",
};

function jsonError(message: string, status: number): Response {
  return Response.json(
    { error: message },
    { status, headers: { ...API_HEADERS, "Cache-Control": "no-store" } },
  );
}

function requestPath(value: string | string[] | undefined): string {
  const joined = Array.isArray(value) ? value.join("/") : value ?? "";
  return joined
    .split("/")
    .filter(Boolean)
    .map((part) => encodeURIComponent(decodeURIComponent(part)))
    .join("/");
}

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: JsonRecord, allowed: readonly string[]): boolean {
  const allowedSet = new Set(allowed);
  return Object.keys(value).every((key) => allowedSet.has(key));
}

function smallPositiveInteger(value: unknown, maximum: number): number | null {
  if (typeof value !== "string" && typeof value !== "number") return null;
  if (!/^\d+$/.test(String(value))) return null;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed > 0 && parsed <= maximum ? parsed : null;
}

function bech32Polymod(values: readonly number[]): number {
  let checksum = 1;
  for (const value of values) {
    const top = checksum >>> 25;
    checksum = (((checksum & 0x1ffffff) << 5) ^ value) >>> 0;
    BECH32_GENERATORS.forEach((generator, index) => {
      if (((top >>> index) & 1) === 1) {
        checksum = (checksum ^ generator) >>> 0;
      }
    });
  }
  return checksum;
}

function validZeroneAccount(value: string): boolean {
  if (!ZERONE_ACCOUNT.test(value)) return false;
  const values = [
    ..."zrn".split("").map((character) => character.charCodeAt(0) >>> 5),
    0,
    ..."zrn".split("").map((character) => character.charCodeAt(0) & 31),
  ];
  for (const character of value.slice(4)) {
    const decoded = BECH32_CHARSET.indexOf(character);
    if (decoded < 0) return false;
    values.push(decoded);
  }
  return bech32Polymod(values) === 1;
}

function validPagination(search: URLSearchParams): boolean {
  const allowed = new Set(["pagination.limit", "pagination.key"]);
  for (const key of search.keys()) {
    if (!allowed.has(key) || search.getAll(key).length !== 1) return false;
  }
  const limit = search.get("pagination.limit");
  if (limit === null || smallPositiveInteger(limit, 50) === null) return false;
  const key = search.get("pagination.key");
  if (
    key !== null &&
    (key.length === 0 ||
      key.length > 344 ||
      !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(key))
  ) {
    return false;
  }
  return true;
}

function validPoolPagination(search: URLSearchParams): boolean {
  const allowed = new Set([
    "pagination.limit",
    "pagination.key",
    "pagination.count_total",
  ]);
  for (const key of search.keys()) {
    if (!allowed.has(key) || search.getAll(key).length !== 1) return false;
  }
  if (
    smallPositiveInteger(search.get("pagination.limit"), 100) === null
  ) {
    return false;
  }
  const key = search.get("pagination.key");
  const countTotal = search.get("pagination.count_total");
  if (key === null) {
    return countTotal === "true" && search.size === 2;
  }
  return (
    countTotal === null &&
    search.size === 2 &&
    key.length <= 344 &&
    /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(
      key,
    ) &&
    key.length > 0
  );
}

async function readLimitedBody(request: Request): Promise<string | null> {
  const declaredLength = request.headers.get("content-length");
  if (declaredLength !== null) {
    const declared = Number(declaredLength);
    if (!Number.isInteger(declared) || declared < 0 || declared > MAX_RPC_BODY_BYTES) {
      return null;
    }
  }

  const reader = request.clone().body?.getReader();
  if (!reader) return "";
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > MAX_RPC_BODY_BYTES) {
      await reader.cancel();
      return null;
    }
    chunks.push(value);
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
    return null;
  }
}

type LimitedResponseBody =
  | { value: string }
  | { error: "too-large" | "unreadable" };

async function readLimitedResponseBody(
  response: Response,
  maximumBytes: number,
): Promise<LimitedResponseBody> {
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!/^\d+$/.test(declaredLength) || Number(declaredLength) > maximumBytes)
  ) {
    return { error: "too-large" };
  }

  const reader = response.body?.getReader();
  if (!reader) return { value: "" };
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > maximumBytes) {
        try {
          await reader.cancel();
        } catch {
          // The size limit has already been enforced.
        }
        return { error: "too-large" };
      }
      chunks.push(value);
    }
  } catch {
    return { error: "unreadable" };
  }

  const bytes = new Uint8Array(total);
  let offset = 0;
  chunks.forEach((chunk) => {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  });
  try {
    return {
      value: new TextDecoder("utf-8", { fatal: true }).decode(bytes),
    };
  } catch {
    return { error: "unreadable" };
  }
}

function validRpcCall(call: JsonRecord): boolean {
  if (
    !hasOnlyKeys(call, ["jsonrpc", "id", "method", "params"]) ||
    call.jsonrpc !== "2.0" ||
    (typeof call.id !== "number" && typeof call.id !== "string") ||
    typeof call.method !== "string"
  ) {
    return false;
  }
  const params = call.params;

  if (call.method === "status") {
    return params === undefined || (isRecord(params) && Object.keys(params).length === 0);
  }

  if (call.method === "abci_query") {
    if (!isRecord(params) || !hasOnlyKeys(params, ["path", "data", "prove"])) return false;
    return (
      typeof params.path === "string" &&
      ABCI_PATHS.has(params.path) &&
      typeof params.data === "string" &&
      /^[A-Fa-f0-9]{2,4096}$/.test(params.data) &&
      params.prove === false
    );
  }

  if (call.method === "broadcast_tx_sync") {
    if (!isRecord(params) || !hasOnlyKeys(params, ["tx"])) return false;
    return (
      typeof params.tx === "string" &&
      params.tx.length <= 280_000 &&
      /^[A-Za-z0-9+/]+={0,2}$/.test(params.tx)
    );
  }

  if (call.method === "tx_search") {
    if (
      !isRecord(params) ||
      !hasOnlyKeys(params, ["query", "prove", "page", "per_page", "order_by"])
    ) {
      return false;
    }
    const page = params.page === undefined ? 1 : smallPositiveInteger(params.page, 1);
    const perPage =
      params.per_page === undefined ? 10 : smallPositiveInteger(params.per_page, 10);
    return (
      typeof params.query === "string" &&
      /^tx\.hash='[A-Fa-f0-9]{64}'$/.test(params.query) &&
      (params.prove === undefined || params.prove === false) &&
      page === 1 &&
      perPage !== null &&
      (params.order_by === undefined || params.order_by === "asc" || params.order_by === "desc")
    );
  }

  return false;
}

async function validRpcBody(request: Request): Promise<boolean> {
  try {
    const raw = await readLimitedBody(request);
    if (raw === null) return false;
    const body = JSON.parse(raw) as unknown;
    return isRecord(body) && validRpcCall(body);
  } catch {
    return false;
  }
}

function validRpcGet(path: string, search: URLSearchParams): boolean {
  if (path === "status" || path === "net_info") return search.size === 0;

  if (path === "block") {
    return (
      search.size === 1 && smallPositiveInteger(search.get("height"), 1_000_000_000) !== null
    );
  }

  if (path === "blockchain") {
    if (search.size !== 2) return false;
    const minimum = smallPositiveInteger(search.get("minHeight"), 1_000_000_000);
    const maximum = smallPositiveInteger(search.get("maxHeight"), 1_000_000_000);
    return minimum !== null && maximum !== null && maximum >= minimum && maximum - minimum <= 7;
  }

  if (path === "validators") {
    return (
      search.size === 2 &&
      smallPositiveInteger(search.get("page"), 1) === 1 &&
      smallPositiveInteger(search.get("per_page"), 100) === 100
    );
  }

  return false;
}

export function validRestRequest(path: string, search: URLSearchParams): boolean {
  if (path === SYNCING_ROUTE) return search.size === 0;
  if (path === "zerone/liquiditypool/v1/pools") {
    return validPoolPagination(search);
  }
  if (path === "zerone/liquiditypool/v1/params") return search.size === 0;
  if (path === "cosmos/bank/v1beta1/denoms_metadata/uzrn") return search.size === 0;
  if (path === "cosmos/bank/v1beta1/supply/by_denom") {
    return search.size === 1 && search.get("denom") === "uzrn";
  }
  const balanceMatch = new RegExp(
    `^cosmos/bank/v1beta1/balances/(${ZERONE_ACCOUNT_PATTERN})/by_denom$`,
  ).exec(path);
  if (balanceMatch) {
    return (
      validZeroneAccount(balanceMatch[1] ?? "") &&
      search.size === 1 &&
      search.get("denom") === "uzrn"
    );
  }
  const identityAddress = path.slice("zerone/auth/v1/account_identifier/".length);
  if (
    path.startsWith("zerone/auth/v1/account_identifier/") &&
    validZeroneAccount(identityAddress)
  ) {
    return search.size === 0;
  }
  const feeGrantPair = FEEGRANT_PAIR.exec(path);
  if (feeGrantPair) {
    return (
      validZeroneAccount(feeGrantPair[1] ?? "") &&
      validZeroneAccount(feeGrantPair[2] ?? "") &&
      search.size === 0
    );
  }
  const feeGrantList = FEEGRANT_LIST.exec(path);
  if (feeGrantList) {
    return (
      validZeroneAccount(feeGrantList[1] ?? "") &&
      validPagination(search)
    );
  }
  return false;
}

export function syncingValueFromStatus(status: unknown): boolean | null {
  if (!isRecord(status) || !isRecord(status.result)) return null;
  const nodeInfo = status.result.node_info;
  const syncInfo = status.result.sync_info;
  if (
    !isRecord(nodeInfo) ||
    nodeInfo.network !== EXPECTED_CHAIN_ID ||
    !isRecord(syncInfo) ||
    typeof syncInfo.catching_up !== "boolean"
  ) {
    return null;
  }
  return syncInfo.catching_up;
}

async function syncingCompatibilityResponse(
  method: string,
  runtime: ProxyRuntime,
): Promise<Response> {
  let upstream: Response;
  try {
    upstream = await runtime.fetch(new URL("/status", runtime.upstreams.rpc), {
      method: "GET",
      headers: { Accept: "application/json" },
      redirect: "manual",
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    return jsonError("Mainnet endpoint is temporarily unreachable", 502);
  }
  if (!upstream.ok) {
    return jsonError("Mainnet status is temporarily unavailable", 502);
  }

  const limitedBody = await readLimitedResponseBody(
    upstream,
    MAX_STATUS_BODY_BYTES,
  );
  if ("error" in limitedBody) {
    return jsonError(
      limitedBody.error === "too-large"
        ? "Mainnet status response exceeded its limit"
        : "Mainnet status response could not be read",
      502,
    );
  }

  let status: unknown;
  try {
    status = JSON.parse(limitedBody.value);
  } catch {
    return jsonError("Mainnet returned malformed status JSON", 502);
  }
  const syncing = syncingValueFromStatus(status);
  if (syncing === null) {
    return jsonError("Mainnet returned an incomplete status response", 502);
  }

  const headers = new Headers(API_HEADERS);
  headers.set("Cache-Control", "public, max-age=2, s-maxage=3");
  headers.set("X-Zerone-Edge", "rest-syncing-compat");
  return new Response(
    method === "HEAD" ? null : JSON.stringify({ syncing }),
    { status: 200, headers },
  );
}

export async function proxyRequest(
  context: PagesContext,
  kind: ProxyKind,
  runtime: ProxyRuntime,
): Promise<Response> {
  const { request } = context;
  const method = request.method.toUpperCase();

  if (method === "OPTIONS") {
    return new Response(null, { status: 204, headers: CORS_HEADERS });
  }

  if (kind === "rest" && !["GET", "HEAD"].includes(method)) {
    return jsonError("The REST edge is read-only", 405);
  }
  if (kind === "rpc" && !["GET", "HEAD", "POST"].includes(method)) {
    return jsonError("Unsupported RPC method", 405);
  }

  const incoming = new URL(request.url);
  let path: string;
  try {
    path = requestPath(context.params.path);
  } catch {
    return jsonError("Malformed request path", 400);
  }

  if (kind === "rest" && !validRestRequest(path, incoming.searchParams)) {
    return jsonError("REST query is not on the public dashboard allowlist", 403);
  }
  if (kind === "rest" && path === SYNCING_ROUTE) {
    return syncingCompatibilityResponse(method, runtime);
  }
  if (kind === "rpc" && method === "POST") {
    if (path !== "" || !(await validRpcBody(request))) {
      return jsonError("RPC call is not on the public dashboard allowlist", 403);
    }
  }
  if (
    kind === "rpc" &&
    (method === "GET" || method === "HEAD") &&
    !validRpcGet(path, incoming.searchParams)
  ) {
    return jsonError("RPC query is not on the public dashboard allowlist", 403);
  }

  const canCache = method === "GET";
  const cacheKey = canCache ? new Request(incoming.toString(), { method: "GET" }) : null;
  if (cacheKey) {
    const cached = await runtime.cache.match(cacheKey);
    if (cached) return cached;
  }

  const target = new URL(`/${path}`, runtime.upstreams[kind]);
  target.search = incoming.search;
  const upstreamHeaders = new Headers({ Accept: "application/json" });
  if (method === "POST") upstreamHeaders.set("Content-Type", "application/json");

  let upstream: Response;
  try {
    upstream = await runtime.fetch(target, {
      method,
      headers: upstreamHeaders,
      body: method === "POST" ? request.body : undefined,
      redirect: "manual",
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    return jsonError("Mainnet endpoint is temporarily unreachable", 502);
  }

  const headers = new Headers(API_HEADERS);
  headers.set("Cache-Control", canCache ? "public, max-age=2, s-maxage=3" : "no-store");
  headers.set("X-Zerone-Edge", kind);
  const response = new Response(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers,
  });

  if (cacheKey && upstream.ok) {
    context.waitUntil(runtime.cache.put(cacheKey, response.clone()));
  }
  return response;
}

export async function proxyMainnet(context: PagesContext, kind: ProxyKind): Promise<Response> {
  const edgeCache = (globalThis.caches as unknown as { default: ProxyCache }).default;
  return proxyRequest(context, kind, {
    fetch: globalThis.fetch.bind(globalThis),
    cache: edgeCache,
    upstreams: UPSTREAMS,
  });
}
