import assert from "node:assert/strict";
import test, { describe, it } from "node:test";

import {
  proxyRequest,
  syncingValueFromStatus,
  type PagesContext,
  type ProxyCache,
  type ProxyRuntime,
  validRestRequest,
} from "../functions/api/_proxy";

const TEST_UPSTREAMS = {
  rpc: "https://rpc.invalid",
  rest: "https://rest.invalid",
} as const;

interface FetchCall {
  input: string | URL | Request;
  init?: RequestInit;
}

interface TestHarness {
  calls: FetchCall[];
  cacheMatches: Request[];
  cachePuts: Array<{ request: Request; response: Response }>;
  runtime: ProxyRuntime;
}

function harness(
  fetchResponse: () => Promise<Response> = async () =>
    Response.json({ ok: true }),
): TestHarness {
  const calls: FetchCall[] = [];
  const cacheMatches: Request[] = [];
  const cachePuts: Array<{ request: Request; response: Response }> = [];
  const cache: ProxyCache = {
    async match(request) {
      cacheMatches.push(request);
      return undefined;
    },
    async put(request, response) {
      cachePuts.push({ request, response });
    },
  };

  return {
    calls,
    cacheMatches,
    cachePuts,
    runtime: {
      upstreams: TEST_UPSTREAMS,
      cache,
      async fetch(input, init) {
        calls.push({ input, init });
        return fetchResponse();
      },
    },
  };
}

function pagesContext(
  url: string,
  path: string | string[] | undefined,
  init?: RequestInit,
): { context: PagesContext; waits: Promise<unknown>[] } {
  const waits: Promise<unknown>[] = [];
  return {
    context: {
      request: new Request(url, init),
      params: { path },
      waitUntil(promise) {
        waits.push(promise);
      },
    },
    waits,
  };
}

async function errorMessage(response: Response): Promise<string> {
  const body = (await response.json()) as { error?: unknown };
  if (typeof body.error !== "string") {
    assert.fail("expected a string error response");
  }
  return body.error;
}

test("REST rejects a path outside the public allowlist without fetching upstream", async () => {
  const testHarness = harness();
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rest/cosmos/gov/v1/proposals",
    "cosmos/gov/v1/proposals",
  );

  const response = await proxyRequest(context, "rest", testHarness.runtime);

  assert.equal(response.status, 403);
  assert.equal(
    await errorMessage(response),
    "REST query is not on the public dashboard allowlist",
  );
  assert.equal(response.headers.get("Cache-Control"), "no-store");
  assert.equal(testHarness.calls.length, 0);
  assert.equal(testHarness.cacheMatches.length, 0);
});

test("REST rejects POST before path validation or upstream access", async () => {
  const testHarness = harness();
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rest/zerone/liquiditypool/v1/pools",
    "zerone/liquiditypool/v1/pools",
    { method: "POST" },
  );

  const response = await proxyRequest(context, "rest", testHarness.runtime);

  assert.equal(response.status, 405);
  assert.equal(await errorMessage(response), "The REST edge is read-only");
  assert.equal(testHarness.calls.length, 0);
  assert.equal(testHarness.cacheMatches.length, 0);
});

test("RPC rejects a JSON-RPC method outside the allowlist without fetching upstream", async () => {
  const testHarness = harness();
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rpc",
    undefined,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "unsafe_flush_mempool",
        params: {},
      }),
    },
  );

  const response = await proxyRequest(context, "rpc", testHarness.runtime);

  assert.equal(response.status, 403);
  assert.equal(
    await errorMessage(response),
    "RPC call is not on the public dashboard allowlist",
  );
  assert.equal(testHarness.calls.length, 0);
  assert.equal(testHarness.cacheMatches.length, 0);
});

test("an allowlisted RPC status call is forwarded only through the injected runtime", async () => {
  const testHarness = harness(async () =>
    new Response('{"result":{"sync_info":{"catching_up":false}}}', {
      status: 200,
      headers: {
        "Content-Type": "text/plain",
        "Set-Cookie": "upstream-secret=1",
        "X-Upstream-Only": "discard",
      },
    }),
  );
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rpc",
    undefined,
    {
      method: "POST",
      headers: {
        Authorization: "Bearer browser-secret",
        Cookie: "browser-secret=1",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: "status-test",
        method: "status",
        params: {},
      }),
    },
  );

  const response = await proxyRequest(context, "rpc", testHarness.runtime);

  assert.equal(response.status, 200);
  assert.equal(testHarness.calls.length, 1);
  const call = testHarness.calls[0];
  assert.ok(call);
  assert.equal(String(call.input), "https://rpc.invalid/");
  assert.equal(call.init?.method, "POST");
  assert.equal(call.init?.redirect, "manual");
  assert.ok(call.init?.signal instanceof AbortSignal);
  const forwardedHeaders = new Headers(call.init?.headers);
  assert.equal(forwardedHeaders.get("Accept"), "application/json");
  assert.equal(forwardedHeaders.get("Content-Type"), "application/json");
  assert.equal(forwardedHeaders.get("Authorization"), null);
  assert.equal(forwardedHeaders.get("Cookie"), null);
  assert.equal(response.headers.get("Content-Type"), "application/json; charset=utf-8");
  assert.equal(response.headers.get("Cache-Control"), "no-store");
  assert.equal(response.headers.get("X-Zerone-Edge"), "rpc");
  assert.equal(response.headers.get("Set-Cookie"), null);
  assert.equal(response.headers.get("X-Upstream-Only"), null);
  assert.equal(testHarness.cacheMatches.length, 0);
});

test("an allowlisted REST read forwards only the query and stores a sanitized response", async () => {
  const testHarness = harness(async () =>
    new Response('{"amount":{"denom":"uzrn","amount":"1"}}', {
      status: 200,
      headers: {
        Location: "https://unexpected.invalid",
        "Set-Cookie": "upstream-secret=1",
      },
    }),
  );
  const { context, waits } = pagesContext(
    "https://dashboard.invalid/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn",
    ["cosmos", "bank", "v1beta1", "supply", "by_denom"],
    {
      headers: {
        Authorization: "Bearer browser-secret",
        Cookie: "browser-secret=1",
      },
    },
  );

  const response = await proxyRequest(context, "rest", testHarness.runtime);
  await Promise.all(waits);

  assert.equal(response.status, 200);
  assert.equal(testHarness.calls.length, 1);
  const call = testHarness.calls[0];
  assert.ok(call);
  assert.equal(
    String(call.input),
    "https://rest.invalid/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn",
  );
  assert.equal(call.init?.method, "GET");
  assert.equal(call.init?.redirect, "manual");
  const forwardedHeaders = new Headers(call.init?.headers);
  assert.equal(forwardedHeaders.get("Accept"), "application/json");
  assert.equal(forwardedHeaders.get("Authorization"), null);
  assert.equal(forwardedHeaders.get("Cookie"), null);
  assert.equal(response.headers.get("Location"), null);
  assert.equal(response.headers.get("Set-Cookie"), null);
  assert.equal(response.headers.get("X-Zerone-Edge"), "rest");
  assert.equal(response.headers.get("Cache-Control"), "public, max-age=2, s-maxage=3");

  assert.equal(testHarness.cacheMatches.length, 1);
  assert.equal(
    testHarness.cacheMatches[0]?.url,
    "https://dashboard.invalid/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn",
  );
  assert.equal(testHarness.cachePuts.length, 1);
  assert.equal(testHarness.cachePuts[0]?.response.headers.get("Set-Cookie"), null);
});

test("an upstream network failure becomes a bounded 502 without leaking the error", async () => {
  const testHarness = harness(async () => {
    throw new Error("private upstream detail");
  });
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rpc/status",
    "status",
  );

  const response = await proxyRequest(context, "rpc", testHarness.runtime);
  const message = await errorMessage(response);

  assert.equal(response.status, 502);
  assert.equal(message, "Mainnet endpoint is temporarily unreachable");
  assert.doesNotMatch(message, /private upstream detail/);
  assert.equal(testHarness.calls.length, 1);
});

test("the syncing compatibility route uses only the injected RPC status endpoint", async () => {
  const testHarness = harness(async () =>
    new Response(
      JSON.stringify({
        result: {
          node_info: { network: "zerone-1" },
          sync_info: { catching_up: false },
        },
      }),
      {
        headers: {
          "Set-Cookie": "upstream-secret=1",
          "X-Upstream-Only": "discard",
        },
      },
    ),
  );
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rest/cosmos/base/tendermint/v1beta1/syncing",
    ["cosmos", "base", "tendermint", "v1beta1", "syncing"],
  );

  const response = await proxyRequest(context, "rest", testHarness.runtime);

  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { syncing: false });
  assert.equal(response.headers.get("X-Zerone-Edge"), "rest-syncing-compat");
  assert.equal(response.headers.get("Set-Cookie"), null);
  assert.equal(response.headers.get("X-Upstream-Only"), null);
  assert.equal(testHarness.calls.length, 1);
  const call = testHarness.calls[0];
  assert.ok(call);
  assert.equal(String(call.input), "https://rpc.invalid/status");
  assert.equal(call.init?.method, "GET");
  assert.equal(call.init?.redirect, "manual");
  assert.equal(testHarness.cacheMatches.length, 0);
  assert.equal(testHarness.cachePuts.length, 0);
});

test("the syncing compatibility route returns no body for HEAD", async () => {
  const testHarness = harness(async () =>
    Response.json({
      result: {
        node_info: { network: "zerone-1" },
        sync_info: { catching_up: true },
      },
    }),
  );
  const { context } = pagesContext(
    "https://dashboard.invalid/api/rest/cosmos/base/tendermint/v1beta1/syncing",
    "cosmos/base/tendermint/v1beta1/syncing",
    { method: "HEAD" },
  );

  const response = await proxyRequest(context, "rest", testHarness.runtime);

  assert.equal(response.status, 200);
  assert.equal(await response.text(), "");
  assert.equal(testHarness.calls.length, 1);
});

test("the syncing compatibility route bounds declared and streamed status bodies", async () => {
  for (const upstream of [
    new Response("", { headers: { "Content-Length": "64001" } }),
    new Response(
      new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(new Uint8Array(64_001));
          controller.close();
        },
      }),
    ),
  ]) {
    const testHarness = harness(async () => upstream);
    const { context } = pagesContext(
      "https://dashboard.invalid/api/rest/cosmos/base/tendermint/v1beta1/syncing",
      "cosmos/base/tendermint/v1beta1/syncing",
    );

    const response = await proxyRequest(context, "rest", testHarness.runtime);

    assert.equal(response.status, 502);
    assert.equal(
      await errorMessage(response),
      "Mainnet status response exceeded its limit",
    );
    assert.equal(testHarness.calls.length, 1);
  }
});

const GRANTER = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const GRANTEE = "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r";
const ADDRESS_WITH_V = "zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89";

describe("feegrant REST edge allowlist", () => {
  it("allows only exact-pair and bounded grantee query shapes", () => {
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowance/${GRANTER}/${GRANTEE}`,
        new URLSearchParams(),
      ),
      true,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${ADDRESS_WITH_V}`,
        new URLSearchParams({ "pagination.limit": "1" }),
      ),
      true,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        new URLSearchParams({
          "pagination.limit": "50",
          "pagination.key": "AQID",
        }),
      ),
      true,
    );
  });

  it("rejects malformed accounts, unbounded pagination, duplicates, and unknown routes", () => {
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowance/${GRANTER}/${GRANTEE.slice(0, -1)}`,
        new URLSearchParams(),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowance/${GRANTER}/${GRANTEE.slice(0, -1)}q`,
        new URLSearchParams(),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        new URLSearchParams(),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/issued/${GRANTER}`,
        new URLSearchParams({ "pagination.limit": "50" }),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        new URLSearchParams({ "pagination.limit": "51" }),
      ),
      false,
    );
    const duplicate = new URLSearchParams();
    duplicate.append("pagination.limit", "10");
    duplicate.append("pagination.limit", "20");
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        duplicate,
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        new URLSearchParams({ "pagination.key": "***" }),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/allowances/${GRANTEE}`,
        new URLSearchParams({
          "pagination.limit": "50",
          "pagination.key": "",
        }),
      ),
      false,
    );
    assert.equal(
      validRestRequest(
        `cosmos/feegrant/v1beta1/grants/${GRANTEE}`,
        new URLSearchParams(),
      ),
      false,
    );
  });
});

describe("standard syncing compatibility response", () => {
  it("accepts only a typed zerone-1 status projection", () => {
    assert.equal(
      syncingValueFromStatus({
        result: {
          node_info: { network: "zerone-1" },
          sync_info: { catching_up: false },
        },
      }),
      false,
    );
    assert.equal(
      syncingValueFromStatus({
        result: {
          node_info: { network: "cosmoshub-4" },
          sync_info: { catching_up: false },
        },
      }),
      null,
    );
    assert.equal(
      syncingValueFromStatus({
        result: {
          node_info: { network: "zerone-1" },
          sync_info: { catching_up: "false" },
        },
      }),
      null,
    );
  });
});
