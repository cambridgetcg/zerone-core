type ProxyKind = "rpc" | "rest";

interface PagesContext {
  request: Request;
  params: Record<string, string | string[] | undefined>;
  waitUntil(promise: Promise<unknown>): void;
}

type JsonRecord = Record<string, unknown>;

const UPSTREAMS: Record<ProxyKind, string> = {
  // Workers reject fetches to literal IP addresses. Fly's stable hostname
  // resolves to the same public mainnet machine without adding a third party.
  rpc: "http://zerone-1.fly.dev:26657",
  rest: "http://zerone-1.fly.dev:1317",
};

const MAX_RPC_BODY_BYTES = 300_000;
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

function validRestRequest(path: string, search: URLSearchParams): boolean {
  if (path === "zerone/liquiditypool/v1/pools") return search.size === 0;
  if (path === "zerone/liquiditypool/v1/params") return search.size === 0;
  if (path === "cosmos/bank/v1beta1/denoms_metadata/uzrn") return search.size === 0;
  if (path === "cosmos/bank/v1beta1/supply/by_denom") {
    return search.size === 1 && search.get("denom") === "uzrn";
  }
  if (/^cosmos\/bank\/v1beta1\/balances\/zrn1[0-9a-z]+\/by_denom$/.test(path)) {
    return search.size === 1 && search.get("denom") === "uzrn";
  }
  return false;
}

export async function proxyMainnet(context: PagesContext, kind: ProxyKind): Promise<Response> {
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
  const edgeCache = (globalThis.caches as unknown as { default: Cache }).default;
  const cacheKey = canCache ? new Request(incoming.toString(), { method: "GET" }) : null;
  if (cacheKey) {
    const cached = await edgeCache.match(cacheKey);
    if (cached) return cached;
  }

  const target = new URL(`/${path}`, UPSTREAMS[kind]);
  target.search = incoming.search;
  const upstreamHeaders = new Headers({ Accept: "application/json" });
  if (method === "POST") upstreamHeaders.set("Content-Type", "application/json");

  let upstream: Response;
  try {
    upstream = await fetch(target, {
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
    context.waitUntil(edgeCache.put(cacheKey, response.clone()));
  }
  return response;
}
