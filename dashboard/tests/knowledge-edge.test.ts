import assert from "node:assert/strict";
import test, { describe, it } from "node:test";

import {
  KNOWLEDGE_FACTS_BODY_MAX_BYTES,
  KNOWLEDGE_FACTS_QUERY_PATH,
  KNOWLEDGE_FACT_CAP,
  KNOWLEDGE_OUTPUT_MAX_BYTES,
  KNOWLEDGE_RELATION_CAP,
  knowledgeRequest,
  type KnowledgeCache,
  type KnowledgePagesContext,
  type KnowledgeRuntime,
} from "../functions/api/_knowledge";

const ENDPOINT = "https://dashboard.invalid/api/knowledge";
const UPSTREAMS = {
  rest: "https://upstream.invalid/rest",
  rpc: "https://upstream.invalid/rpc",
} as const;

interface FetchCall {
  input: string | URL | Request;
  init?: RequestInit;
}

interface Harness {
  calls: FetchCall[];
  cacheMatches: Request[];
  cachePuts: Array<{ request: Request; response: Response }>;
  waits: Promise<unknown>[];
  runtime: KnowledgeRuntime;
  context(method?: string, url?: string): KnowledgePagesContext;
}

type FetchResponder = (
  target: URL,
  init: RequestInit | undefined,
) => Promise<Response>;

function relation(
  sourceFactId = "fact-b",
  targetFactId = "fact-a",
): Record<string, unknown> {
  return {
    sourceFactId,
    targetFactId,
    relation: "RELATION_TYPE_SUPPORTS",
    inference: "INFERENCE_TYPE_EMPIRICAL",
    inferenceStrengthBps: "750000",
    createdAtBlock: "900",
    methodId: "",
    creator: "zrn1private",
  };
}

function fact(
  id: string,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    id,
    content: `Knowledge carried by ${id}`,
    domain: "general",
    category: "empirical",
    status: "FACT_STATUS_VERIFIED",
    claimType: "CLAIM_TYPE_ASSERTION",
    confidence: "880000",
    verifiedAtBlock: "800",
    lastVerifiedBlock: "800",
    energy: "500000",
    energyCap: "1000000",
    fitnessScore: "75000",
    methodId: "M-EMPIRICAL",
    outgoingRelations: [],
    incomingRelations: [],
    submitter: "zrn1private",
    nicheRank: "1",
    personRank: 99,
    ...overrides,
  };
}

function factsBody(facts: unknown[], pagination: unknown = null): unknown {
  return { facts, pagination };
}

function factsResponse(
  body: unknown,
  options: {
    blockHeight?: string;
    contentType?: string;
    extraHeaders?: Record<string, string>;
    raw?: boolean;
    status?: number;
  } = {},
): Response {
  const headers = new Headers({
    "Content-Type": options.contentType ?? "application/json; charset=utf-8",
    "grpc-metadata-x-cosmos-block-height": options.blockHeight ?? "1000",
    "Set-Cookie": "upstream-facts-secret=1",
    "X-Upstream-Facts": "discard",
    ...options.extraHeaders,
  });
  return new Response(options.raw ? String(body) : JSON.stringify(body), {
    status: options.status ?? 200,
    headers,
  });
}

function statusResponse(
  options: {
    chainId?: string;
    catchingUp?: boolean;
    contentType?: string;
    height?: string;
    raw?: string;
    status?: number;
  } = {},
): Response {
  const body =
    options.raw ??
    JSON.stringify({
      jsonrpc: "2.0",
      id: -1,
      result: {
        node_info: { network: options.chainId ?? "zerone-1" },
        sync_info: {
          latest_block_height: options.height ?? "1002",
          catching_up: options.catchingUp ?? false,
        },
      },
    });
  return new Response(body, {
    status: options.status ?? 200,
    headers: {
      "Content-Type": options.contentType ?? "application/json",
      "Set-Cookie": "upstream-status-secret=1",
      "X-Upstream-Status": "discard",
    },
  });
}

function normalFacts(): unknown {
  const edge = relation();
  return factsBody([
    fact("fact-b", { outgoingRelations: [edge] }),
    fact("fact-a", {
      content: "  exact surrounding whitespace is preserved  ",
      domain: "",
      category: "",
      status: "FACT_STATUS_EXPIRED",
      claimType: "CLAIM_TYPE_UNSPECIFIED",
      verifiedAtBlock: "0",
      lastVerifiedBlock: "0",
      energy: "1000000",
      energyCap: "0",
      methodId: "",
      incomingRelations: [edge],
    }),
  ]);
}

function createHarness(
  responder: FetchResponder = async (target) =>
    target.pathname.endsWith("/status")
      ? statusResponse()
      : factsResponse(normalFacts()),
  cached?: Response,
): Harness {
  const calls: FetchCall[] = [];
  const cacheMatches: Request[] = [];
  const cachePuts: Array<{ request: Request; response: Response }> = [];
  const waits: Promise<unknown>[] = [];
  const cache: KnowledgeCache = {
    async match(request) {
      cacheMatches.push(request);
      return cached;
    },
    async put(request, response) {
      cachePuts.push({ request, response });
    },
  };

  return {
    calls,
    cacheMatches,
    cachePuts,
    waits,
    runtime: {
      upstreams: UPSTREAMS,
      cache,
      async fetch(input, init) {
        calls.push({ input, init });
        return responder(new URL(String(input)), init);
      },
    },
    context(method = "GET", url = ENDPOINT) {
      return {
        request: new Request(url, {
          method,
          headers: {
            Authorization: "Bearer browser-secret",
            Cookie: "browser-secret=1",
            "X-Browser-Only": "discard",
          },
        }),
        waitUntil(promise) {
          waits.push(promise);
        },
      };
    },
  };
}

async function errorMessage(response: Response): Promise<string> {
  const value = (await response.json()) as { error?: unknown };
  assert.equal(typeof value.error, "string");
  return value.error as string;
}

function utf8Bytes(value: string): number {
  return new TextEncoder().encode(value).byteLength;
}

describe("knowledge geometry edge contract", () => {
  it("projects a typed, uniformly shaped, relational snapshot from exactly two reads", async () => {
    const harness = createHarness();
    const response = await knowledgeRequest(harness.context(), harness.runtime);
    await Promise.all(harness.waits);

    assert.equal(response.status, 200);
    const raw = await response.text();
    assert.equal(Number(response.headers.get("Content-Length")), utf8Bytes(raw));
    assert.ok(utf8Bytes(raw) <= KNOWLEDGE_OUTPUT_MAX_BYTES);
    const snapshot = JSON.parse(raw) as Record<string, unknown>;
    assert.deepEqual(Object.keys(snapshot), ["schema", "source", "facts", "relations"]);
    assert.equal(snapshot.schema, "zerone.knowledge-geometry-snapshot/v0");
    assert.deepEqual(snapshot.source, {
      chainId: "zerone-1",
      blockHeight: "1000",
      statusHeight: "1002",
      catchingUp: false,
      queryPath: KNOWLEDGE_FACTS_QUERY_PATH,
      queryTracked: false,
      writes: false,
      completeness: "NOT_CLAIMED",
      upstreamRecords: 2,
      returnedRecords: 2,
      truncated: false,
    });

    const projectedFacts = snapshot.facts as Array<Record<string, unknown>>;
    assert.deepEqual(projectedFacts.map(({ id }) => id), ["fact-a", "fact-b"]);
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
    ];
    assert.ok(projectedFacts.every((projected) =>
      JSON.stringify(Object.keys(projected)) === JSON.stringify(factKeys)));
    assert.equal(projectedFacts[0]?.content, "  exact surrounding whitespace is preserved  ");
    assert.equal(projectedFacts[0]?.domain, "");
    assert.equal(projectedFacts[0]?.methodId, "");
    assert.equal(projectedFacts[0]?.energy, 1_000_000);
    assert.equal(projectedFacts[0]?.energyCap, 0);
    assert.equal(typeof projectedFacts[1]?.confidence, "number");
    assert.equal(projectedFacts[1]?.submitter, undefined);
    assert.equal(projectedFacts[1]?.nicheRank, undefined);
    assert.equal(projectedFacts[1]?.personRank, undefined);

    const projectedRelations = snapshot.relations as Array<Record<string, unknown>>;
    assert.equal(projectedRelations.length, 1);
    assert.deepEqual(projectedRelations[0], {
      sourceFactId: "fact-b",
      targetFactId: "fact-a",
      relation: "RELATION_TYPE_SUPPORTS",
      inference: "INFERENCE_TYPE_EMPIRICAL",
      inferenceStrengthBps: 750000,
      createdAtBlock: "900",
      methodId: "",
    });

    assert.equal(harness.calls.length, 2);
    const factsCall = harness.calls.find(({ input }) =>
      new URL(String(input)).pathname.includes("/facts"));
    const statusCall = harness.calls.find(({ input }) =>
      new URL(String(input)).pathname.endsWith("/status"));
    assert.ok(factsCall);
    assert.ok(statusCall);
    assert.equal(
      String(factsCall.input),
      `https://upstream.invalid/rest${KNOWLEDGE_FACTS_QUERY_PATH}`,
    );
    assert.equal(String(statusCall.input), "https://upstream.invalid/rpc/status");
    for (const call of harness.calls) {
      assert.equal(call.init?.method, "GET");
      assert.equal(call.init?.redirect, "manual");
      assert.ok(call.init?.signal instanceof AbortSignal);
      const headers = new Headers(call.init?.headers);
      assert.equal(headers.get("Accept"), "application/json");
      assert.equal(headers.get("Authorization"), null);
      assert.equal(headers.get("Cookie"), null);
      assert.equal(headers.get("X-Browser-Only"), null);
    }

    assert.equal(response.headers.get("Cache-Control"), "public, max-age=30, s-maxage=60");
    assert.equal(response.headers.get("Content-Type"), "application/json; charset=utf-8");
    assert.equal(response.headers.get("Set-Cookie"), null);
    assert.equal(response.headers.get("X-Upstream-Facts"), null);
    assert.equal(response.headers.get("X-Upstream-Status"), null);
    assert.equal(harness.cacheMatches.length, 1);
    assert.equal(harness.cacheMatches[0]?.url, ENDPOINT);
    assert.equal(harness.cacheMatches[0]?.method, "GET");
    assert.equal(harness.cacheMatches[0]?.headers.get("Authorization"), null);
    assert.equal(harness.cachePuts.length, 1);
    assert.equal(harness.cachePuts[0]?.request.url, ENDPOINT);
    assert.equal(harness.cachePuts[0]?.response.headers.get("Set-Cookie"), null);
    assert.equal(
      harness.cachePuts[0]?.response.headers.get("Content-Length"),
      response.headers.get("Content-Length"),
    );
  });

  it("serves HEAD with no body and the exact GET representation length", async () => {
    const getHarness = createHarness();
    const getResponse = await knowledgeRequest(getHarness.context(), getHarness.runtime);
    const getBody = await getResponse.text();

    const headHarness = createHarness();
    const headResponse = await knowledgeRequest(
      headHarness.context("HEAD"),
      headHarness.runtime,
    );

    assert.equal(headResponse.status, 200);
    assert.equal(await headResponse.text(), "");
    assert.equal(
      headResponse.headers.get("Content-Length"),
      String(utf8Bytes(getBody)),
    );
    assert.equal(headHarness.calls.length, 2);
    assert.ok(headHarness.calls.every(({ init }) => init?.method === "GET"));
    assert.equal(headHarness.cacheMatches.length, 0);
    assert.equal(headHarness.cachePuts.length, 0);
  });

  it("allows only the exact path, empty query, and GET/HEAD/OPTIONS methods", async () => {
    const optionsHarness = createHarness();
    const options = await knowledgeRequest(
      optionsHarness.context("OPTIONS"),
      optionsHarness.runtime,
    );
    assert.equal(options.status, 204);
    assert.equal(options.headers.get("Allow"), "GET, HEAD, OPTIONS");
    assert.equal(options.headers.get("Access-Control-Allow-Methods"), "GET, HEAD, OPTIONS");
    assert.equal(optionsHarness.calls.length, 0);
    assert.equal(optionsHarness.cacheMatches.length, 0);

    for (const [method, url, status, message] of [
      ["POST", ENDPOINT, 405, "supports only"],
      ["GET", `${ENDPOINT}?domain=general`, 400, "does not accept query"],
      ["GET", "https://dashboard.invalid/api/knowledge/extra", 400, "path is invalid"],
    ] as const) {
      const harness = createHarness();
      const response = await knowledgeRequest(harness.context(method, url), harness.runtime);
      assert.equal(response.status, status);
      assert.match(await errorMessage(response), new RegExp(message));
      assert.equal(response.headers.get("Cache-Control"), "no-store");
      assert.equal(harness.calls.length, 0);
      assert.equal(harness.cacheMatches.length, 0);
    }
  });

  it("revalidates the exact credential-free cache entry and strips cached headers", async () => {
    const seedHarness = createHarness();
    const seedResponse = await knowledgeRequest(
      seedHarness.context(),
      seedHarness.runtime,
    );
    const body = await seedResponse.text();
    const cached = new Response(body, {
      headers: {
        "Content-Length": String(utf8Bytes(body)),
        "Content-Type": "application/json",
        "Set-Cookie": "cache-secret=1",
        "X-Cache-Secret": "discard",
      },
    });
    const harness = createHarness(undefined, cached);

    const response = await knowledgeRequest(harness.context(), harness.runtime);

    assert.equal(response.status, 200);
    assert.equal(await response.text(), body);
    assert.equal(response.headers.get("Content-Length"), String(utf8Bytes(body)));
    assert.equal(response.headers.get("Set-Cookie"), null);
    assert.equal(response.headers.get("X-Cache-Secret"), null);
    assert.equal(response.headers.get("Cache-Control"), "public, max-age=30, s-maxage=60");
    assert.equal(harness.cacheMatches.length, 1);
    assert.equal(harness.cacheMatches[0]?.url, ENDPOINT);
    assert.equal(harness.cacheMatches[0]?.headers.get("Cookie"), null);
    assert.equal(harness.calls.length, 0);
    assert.equal(harness.cachePuts.length, 0);

    const invalidBody = JSON.stringify({ cached: true });
    const invalidCache = new Response(invalidBody, {
      headers: {
        "Content-Length": String(utf8Bytes(invalidBody)),
        "Content-Type": "application/json",
      },
    });
    const fallbackHarness = createHarness(undefined, invalidCache);
    const fallbackResponse = await knowledgeRequest(
      fallbackHarness.context(),
      fallbackHarness.runtime,
    );
    assert.equal(fallbackResponse.status, 200);
    assert.equal(
      (await fallbackResponse.json() as { schema?: unknown }).schema,
      "zerone.knowledge-geometry-snapshot/v0",
    );
    assert.equal(fallbackHarness.calls.length, 2);
    assert.equal(fallbackHarness.cachePuts.length, 1);
  });
});

describe("knowledge upstream trust boundary", () => {
  it("refuses malformed and duplicate-key JSON from either upstream", async () => {
    const duplicateFact = JSON.stringify(fact("fact-a")).replace(
      '"confidence":"880000"',
      '"confidence":"1","\\u0063onfidence":"880000"',
    );
    const cases: Array<{ responder: FetchResponder; message: RegExp }> = [
      {
        responder: async (target) =>
          target.pathname.endsWith("/status")
            ? statusResponse()
            : factsResponse('{"facts":[', { raw: true }),
        message: /facts response contains malformed JSON/,
      },
      {
        responder: async (target) =>
          target.pathname.endsWith("/status")
            ? statusResponse()
            : factsResponse(`{"facts":[${duplicateFact}],"pagination":null}`, { raw: true }),
        message: /facts response contains duplicate JSON keys/,
      },
      {
        responder: async (target) =>
          target.pathname.endsWith("/status")
            ? statusResponse({
                raw: '{"result":{"node_info":{"network":"zerone-1","\\u006eetwork":"evil"},"sync_info":{"latest_block_height":"1002","catching_up":false}}}',
              })
            : factsResponse(normalFacts()),
        message: /status response contains duplicate JSON keys/,
      },
    ];

    for (const scenario of cases) {
      const harness = createHarness(scenario.responder);
      const response = await knowledgeRequest(harness.context(), harness.runtime);
      assert.equal(response.status, 502);
      assert.match(await errorMessage(response), scenario.message);
      assert.equal(response.headers.get("Cache-Control"), "no-store");
      assert.equal(harness.cachePuts.length, 0);
    }
  });

  it("refuses redirects, final-URL drift, non-JSON media, and upstream errors", async () => {
    const redirected = statusResponse();
    Object.defineProperty(redirected, "redirected", { value: true });
    const drifted = factsResponse(normalFacts());
    Object.defineProperty(drifted, "url", {
      value: "https://different.invalid/knowledge",
    });

    const cases: Array<{ responseFor: "facts" | "status"; response: Response; message: RegExp }> = [
      {
        responseFor: "facts",
        response: new Response("", {
          status: 302,
          headers: { Location: "https://different.invalid" },
        }),
        message: /facts endpoint refused a redirect/,
      },
      { responseFor: "status", response: redirected, message: /status endpoint refused a redirect/ },
      { responseFor: "facts", response: drifted, message: /facts endpoint refused a redirect/ },
      {
        responseFor: "facts",
        response: factsResponse(normalFacts(), { contentType: "text/plain" }),
        message: /facts endpoint returned a non-JSON response/,
      },
      {
        responseFor: "status",
        response: statusResponse({ contentType: "application/problem+json" }),
        message: /status endpoint returned a non-JSON response/,
      },
      {
        responseFor: "facts",
        response: factsResponse({ error: "private" }, { status: 503 }),
        message: /facts endpoint is temporarily unavailable/,
      },
    ];

    for (const scenario of cases) {
      const harness = createHarness(async (target) => {
        const label = target.pathname.endsWith("/status") ? "status" : "facts";
        if (label === scenario.responseFor) return scenario.response;
        return label === "status" ? statusResponse() : factsResponse(normalFacts());
      });
      const response = await knowledgeRequest(harness.context(), harness.runtime);
      assert.equal(response.status, 502);
      const message = await errorMessage(response);
      assert.match(message, scenario.message);
      assert.doesNotMatch(message, /private/);
      assert.equal(harness.cachePuts.length, 0);
    }
  });

  it("enforces declared and streamed byte ceilings", async () => {
    const declared = factsResponse("{}", {
      raw: true,
      extraHeaders: {
        "Content-Length": String(KNOWLEDGE_FACTS_BODY_MAX_BYTES + 1),
      },
    });
    const streamed = new Response(
      new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(new Uint8Array(KNOWLEDGE_FACTS_BODY_MAX_BYTES + 1));
          controller.close();
        },
      }),
      {
        headers: {
          "Content-Type": "application/json",
          "grpc-metadata-x-cosmos-block-height": "1000",
        },
      },
    );
    let hostileCancelled = false;
    const hostileCancellation = new Response(
      new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(new Uint8Array(KNOWLEDGE_FACTS_BODY_MAX_BYTES + 1));
        },
        cancel() {
          hostileCancelled = true;
          return new Promise<void>(() => undefined);
        },
      }),
      {
        headers: {
          "Content-Type": "application/json",
          "grpc-metadata-x-cosmos-block-height": "1000",
        },
      },
    );

    for (const upstream of [declared, streamed, hostileCancellation]) {
      const harness = createHarness(async (target) =>
        target.pathname.endsWith("/status") ? statusResponse() : upstream);
      const response = await knowledgeRequest(harness.context(), harness.runtime);
      assert.equal(response.status, 502);
      assert.match(await errorMessage(response), /exceeded its byte limit/);
      assert.equal(harness.cachePuts.length, 0);
    }
    assert.equal(hostileCancelled, true);
  });

  it("requires a zerone-1 status proof and close, canonical positive metadata heights", async () => {
    const cases: Array<{
      facts: Response;
      status: Response;
      message: RegExp;
    }> = [
      {
        facts: factsResponse(normalFacts()),
        status: statusResponse({ chainId: "evil-1" }),
        message: /did not prove chain zerone-1/,
      },
      {
        facts: factsResponse(normalFacts(), { blockHeight: "1000" }),
        status: statusResponse({ height: "1129" }),
        message: /heights are too far apart/,
      },
      {
        facts: factsResponse(normalFacts(), { blockHeight: "01000" }),
        status: statusResponse(),
        message: /omitted a valid block height/,
      },
      {
        facts: factsResponse(normalFacts()),
        status: statusResponse({ height: "0" }),
        message: /invalid block height/,
      },
    ];

    for (const scenario of cases) {
      const harness = createHarness(async (target) =>
        target.pathname.endsWith("/status") ? scenario.status : scenario.facts);
      const response = await knowledgeRequest(harness.context(), harness.runtime);
      assert.equal(response.status, 502);
      assert.match(await errorMessage(response), scenario.message);
    }
  });

  it("rejects future, noncanonical, inconsistent, and oversized fact/relation values", async () => {
    const invalidFacts: unknown[] = [
      factsBody([fact("fact-a", { verifiedAtBlock: "1001", lastVerifiedBlock: "1001" })]),
      factsBody([fact("fact-a", { verifiedAtBlock: "0800" })]),
      factsBody([fact("fact-a", { verifiedAtBlock: "900", lastVerifiedBlock: "800" })]),
      factsBody([fact("fact-a", { confidence: "1000001" })]),
      factsBody([fact("fact-a", { status: "FACT_STATUS_PERSON_RANKED" })]),
      factsBody([fact("fact-a", { content: `safe\u202eevil` })]),
      factsBody([fact("fact-a"), fact("fact-a")]),
      factsBody([
        fact("fact-a", {
          outgoingRelations: [relation("not-the-anchor", "fact-a")],
        }),
      ]),
      factsBody([
        fact("fact-a"),
        fact("fact-b", {
          outgoingRelations: [
            { ...relation(), createdAtBlock: "1001" },
          ],
        }),
      ]),
    ];

    for (const body of invalidFacts) {
      const harness = createHarness(async (target) =>
        target.pathname.endsWith("/status")
          ? statusResponse()
          : factsResponse(body));
      const response = await knowledgeRequest(harness.context(), harness.runtime);
      assert.equal(response.status, 502);
      assert.match(await errorMessage(response), /invalid|future|inconsistent|duplicate/);
      assert.equal(harness.cachePuts.length, 0);
    }
  });

  it("deduplicates exact directional pairs but rejects conflicting copies", async () => {
    const edge = relation();
    const exact = factsBody([
      fact("fact-a", { incomingRelations: [edge] }),
      fact("fact-b", { outgoingRelations: [edge, edge] }),
    ]);
    const exactHarness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(exact));
    const exactResponse = await knowledgeRequest(
      exactHarness.context(),
      exactHarness.runtime,
    );
    assert.equal(exactResponse.status, 200);
    const exactSnapshot = (await exactResponse.json()) as { relations: unknown[] };
    assert.equal(exactSnapshot.relations.length, 1);

    const conflict = factsBody([
      fact("fact-a", { incomingRelations: [edge] }),
      fact("fact-b", {
        outgoingRelations: [
          edge,
          { ...edge, inferenceStrengthBps: "749999" },
        ],
      }),
    ]);
    const conflictHarness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(conflict));
    const conflictResponse = await knowledgeRequest(
      conflictHarness.context(),
      conflictHarness.runtime,
    );
    assert.equal(conflictResponse.status, 502);
    assert.match(await errorMessage(conflictResponse), /conflicting duplicate relations/);

    const crossBoundaryEdge = relation("fact-128", "fact-000");
    const crossBoundaryFacts = Array.from(
      { length: KNOWLEDGE_FACT_CAP + 1 },
      (_, index) => {
        const id = `fact-${String(index).padStart(3, "0")}`;
        if (id === "fact-000") {
          return fact(id, { incomingRelations: [crossBoundaryEdge] });
        }
        if (id === "fact-128") {
          return fact(id, {
            outgoingRelations: [
              { ...crossBoundaryEdge, inferenceStrengthBps: "749999" },
            ],
          });
        }
        return fact(id);
      },
    );
    const crossBoundaryHarness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(factsBody(crossBoundaryFacts)));
    const crossBoundaryResponse = await knowledgeRequest(
      crossBoundaryHarness.context(),
      crossBoundaryHarness.runtime,
    );
    assert.equal(crossBoundaryResponse.status, 502);
    assert.match(
      await errorMessage(crossBoundaryResponse),
      /conflicting duplicate relations/,
    );

    const outsideEdge = relation("y998", "z999");
    const outsideFacts = [
      ...Array.from({ length: KNOWLEDGE_FACT_CAP }, (_, index) =>
        fact(`a-${String(index).padStart(3, "0")}`)),
      fact("y998", { outgoingRelations: [outsideEdge] }),
      fact("z999", {
        incomingRelations: [
          { ...outsideEdge, inferenceStrengthBps: "749999" },
        ],
      }),
    ];
    const outsideHarness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(factsBody(outsideFacts)));
    const outsideResponse = await knowledgeRequest(
      outsideHarness.context(),
      outsideHarness.runtime,
    );
    assert.equal(outsideResponse.status, 502);
    assert.match(
      await errorMessage(outsideResponse),
      /conflicting duplicate relations/,
    );
  });
});

describe("knowledge projection caps", () => {
  it("returns at most 128 sorted facts and 512 sorted directional relations", async () => {
    const facts = Array.from({ length: KNOWLEDGE_FACT_CAP + 1 }, (_, index) =>
      fact(`fact-${String(index).padStart(3, "0")}`));
    let relationNumber = 0;
    for (let source = 0; source < 32 && relationNumber <= KNOWLEDGE_RELATION_CAP; source += 1) {
      const outgoing: unknown[] = [];
      for (let target = 0; target < 32 && relationNumber <= KNOWLEDGE_RELATION_CAP; target += 1) {
        if (source === target) continue;
        outgoing.push(
          relation(
            `fact-${String(source).padStart(3, "0")}`,
            `fact-${String(target).padStart(3, "0")}`,
          ),
        );
        relationNumber += 1;
      }
      facts[source] = { ...facts[source], outgoingRelations: outgoing };
    }
    facts.reverse();

    const harness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(factsBody(facts)));
    const response = await knowledgeRequest(harness.context(), harness.runtime);
    const snapshot = (await response.json()) as {
      source: { upstreamRecords: number; returnedRecords: number; truncated: boolean };
      facts: Array<{ id: string }>;
      relations: Array<{ sourceFactId: string; targetFactId: string }>;
    };

    assert.equal(response.status, 200);
    assert.equal(snapshot.source.upstreamRecords, KNOWLEDGE_FACT_CAP + 1);
    assert.equal(snapshot.source.returnedRecords, KNOWLEDGE_FACT_CAP);
    assert.equal(snapshot.source.truncated, true);
    assert.equal(snapshot.facts.length, KNOWLEDGE_FACT_CAP);
    assert.equal(snapshot.facts[0]?.id, "fact-000");
    assert.equal(snapshot.facts.at(-1)?.id, "fact-127");
    assert.equal(snapshot.relations.length, KNOWLEDGE_RELATION_CAP);
    const relationKeys = snapshot.relations.map(
      ({ sourceFactId, targetFactId }) => `${sourceFactId}\u0000${targetFactId}`,
    );
    assert.deepEqual(relationKeys, [...relationKeys].sort());
    assert.ok(Number(response.headers.get("Content-Length")) <= KNOWLEDGE_OUTPUT_MAX_BYTES);
  });

  it("shrinks a deterministic fact prefix until serialized output fits 262144 bytes", async () => {
    const facts = Array.from({ length: 20 }, (_, index) =>
      fact(`large-${String(index).padStart(2, "0")}`, {
        content: `${String(index).padStart(2, "0")}:${"x".repeat(15_900)}`,
      }));
    facts.reverse();
    const harness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(factsBody(facts)));

    const response = await knowledgeRequest(harness.context(), harness.runtime);
    const raw = await response.text();
    const snapshot = JSON.parse(raw) as {
      source: { upstreamRecords: number; returnedRecords: number; truncated: boolean };
      facts: Array<{ id: string }>;
    };

    assert.equal(response.status, 200);
    assert.ok(utf8Bytes(raw) <= KNOWLEDGE_OUTPUT_MAX_BYTES);
    assert.equal(Number(response.headers.get("Content-Length")), utf8Bytes(raw));
    assert.equal(snapshot.source.upstreamRecords, 20);
    assert.ok(snapshot.source.returnedRecords > 0);
    assert.ok(snapshot.source.returnedRecords < 20);
    assert.equal(snapshot.source.truncated, true);
    assert.deepEqual(
      snapshot.facts.map(({ id }) => id),
      Array.from(
        { length: snapshot.source.returnedRecords },
        (_, index) => `large-${String(index).padStart(2, "0")}`,
      ),
    );
  });

  it("uses the truthful truncation flag at the exact 262144-byte boundary", async () => {
    const contentLengths = [
      ...Array<number>(5).fill(16_384),
      16_017,
      ...Array<number>(10).fill(16_000),
    ];
    const facts = contentLengths.map((length, index) =>
      fact(`boundary-${String(index).padStart(2, "0")}`, {
        content: "x".repeat(length),
      }));
    const harness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse()
        : factsResponse(factsBody(facts)));

    const response = await knowledgeRequest(harness.context(), harness.runtime);
    const raw = await response.text();
    const snapshot = JSON.parse(raw) as {
      source: { upstreamRecords: number; returnedRecords: number; truncated: boolean };
    };

    assert.equal(response.status, 200);
    assert.ok(utf8Bytes(raw) <= KNOWLEDGE_OUTPUT_MAX_BYTES);
    assert.equal(Number(response.headers.get("Content-Length")), utf8Bytes(raw));
    assert.equal(snapshot.source.upstreamRecords, contentLengths.length);
    assert.ok(snapshot.source.returnedRecords < contentLengths.length);
    assert.equal(snapshot.source.truncated, true);
  });

  it("marks advertised pagination without claiming completeness", async () => {
    const harness = createHarness(async (target) =>
      target.pathname.endsWith("/status")
        ? statusResponse({ catchingUp: true })
        : factsResponse(
            factsBody([fact("fact-a")], { nextKey: "AQID", total: "10" }),
          ));
    const response = await knowledgeRequest(harness.context(), harness.runtime);
    const snapshot = (await response.json()) as {
      source: { completeness: string; upstreamRecords: number; truncated: boolean; catchingUp: boolean };
    };

    assert.equal(response.status, 200);
    assert.deepEqual(snapshot.source, {
      chainId: "zerone-1",
      blockHeight: "1000",
      statusHeight: "1002",
      catchingUp: true,
      queryPath: KNOWLEDGE_FACTS_QUERY_PATH,
      queryTracked: false,
      writes: false,
      completeness: "NOT_CLAIMED",
      upstreamRecords: 1,
      returnedRecords: 1,
      truncated: true,
    });
  });
});

test("network failures become bounded errors without leaking private details", async () => {
  const harness = createHarness(async (target) => {
    if (target.pathname.includes("/facts")) {
      throw new Error("private signer topology");
    }
    return statusResponse();
  });
  const response = await knowledgeRequest(harness.context(), harness.runtime);
  const message = await errorMessage(response);

  assert.equal(response.status, 502);
  assert.equal(message, "Mainnet facts endpoint is temporarily unreachable");
  assert.doesNotMatch(message, /private signer topology/);
  assert.equal(harness.cachePuts.length, 0);
});
