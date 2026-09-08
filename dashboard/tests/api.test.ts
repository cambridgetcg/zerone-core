import assert from "node:assert/strict";
import { afterEach, describe, it } from "node:test";
import { createObserverClient, OBSERVER_INTERVAL_MS, observerReadinessAtAge } from "../src/observer-api";
import { assessNetworkReadiness, BLOCK_FRESHNESS_WINDOW_MS } from "../src/onboarding";
import { syntheticProfile, syntheticResponse, syntheticValidator, NATIVE_ADDRESS, BLOCK_HASH, NOW } from "./observer-fixtures";

Object.defineProperty(globalThis, "location", {
  configurable: true,
  value: { origin: "https://dashboard.invalid" },
});

const {
  getLiquidityPoolRegistry,
  LIQUIDITY_POOL_PAGE_LIMIT,
  LIQUIDITY_POOL_RECORD_CAP,
} = await import("../src/api");

const CREATOR = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const originalFetch = globalThis.fetch;

function closedPool(id: number): Record<string, unknown> {
  const poolId = `pool-${id}`;
  return {
    poolId,
    denomA: "uzrn",
    denomB: "uatom",
    reserveA: "0",
    reserveB: "0",
    swapFeeBps: "3000",
    lpTokenSupply: "0",
    lpDenom: `lp/${poolId}`,
    creator: CREATOR,
    createdAtBlock: "42",
    locked: false,
    status: "POOL_STATUS_CLOSED",
    closedAtBlock: "99",
  };
}

afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("gateway-compatible observer API", () => {
  function observerHarness(mode: "preview" | "beta-active" | "beta-archive" = "preview", transform: (path: string, body: Record<string, unknown>) => unknown = (_path, body) => body) {
    let now = NOW;
    const calls: Array<{ path: string; at: number }> = [];
    const profile = syntheticProfile(mode);
    const client = createObserverClient(profile, "https://dashboard.invalid", {
      now: () => now, sleep: async (ms) => { now += ms; },
      fetch: async (input) => {
        const url = new URL(String(input));
        const path = `${url.pathname.replace(/^\/api\/(rpc|rest)/, "")}${url.search}`;
        calls.push({ path, at: now });
        const result = transform(path, syntheticResponse(path, NOW, mode === "beta-archive"));
        return result instanceof Response ? result : Response.json(result);
      },
    });
    return { client, calls };
  }
  it("uses a bounded cached sequential lane, no net_info, blockchain or liquidity scans", async () => {
    const { client, calls } = observerHarness();
    const first = await client.snapshot();
    assert.equal(first.readiness, "ready"); assert.equal(first.supplyUzrn, "100000000");
    assert.equal(first.validators, 1);
    assert.equal((await client.recentBlocks(100)).length, 4);
    const pointCount = calls.filter((call) => call.path.startsWith("/block?")).length;
    assert.equal(pointCount, 4);
    await client.recentBlocks(4); await client.block(10);
    assert.equal(calls.filter((call) => call.path.startsWith("/block?")).length, pointCount);
    await Promise.all([client.balance(NATIVE_ADDRESS), client.balance(NATIVE_ADDRESS)]);
    for (let index = 1; index < calls.length; index += 1) assert.ok(calls[index]!.at - calls[index - 1]!.at >= OBSERVER_INTERVAL_MS);
    assert.ok(calls.every(({ path }) => !/net_info|blockchain|liquidity|pagination/.test(path)));
  });
  it("refuses wrong chain, malformed identity and mismatched point block", async () => {
    for (const change of ["chain", "hash", "time", "incomplete"]) {
      const { client } = observerHarness("preview", (path, body) => {
        if (path !== "/status") return body;
        const result = body.result as { node_info: { network: string }; sync_info: Record<string, unknown> };
        if (change === "chain") result.node_info.network = "wrong-chain";
        if (change === "hash") result.sync_info.latest_block_hash = "f".repeat(64);
        if (change === "time") result.sync_info.latest_block_time = new Date(NOW - 9000).toISOString();
        if (change === "incomplete") delete result.sync_info.catching_up;
        return body;
      });
      await assert.rejects(client.snapshot());
      await assert.rejects(client.balance(NATIVE_ADDRESS), /identity/);
    }
  });
  it("distinguishes stale, future and frozen archive records", async () => {
    for (const time of [NOW - 100_000, NOW + 100_000]) {
      const { client } = observerHarness("beta-active", (path) => syntheticResponse(path, time));
      assert.equal((await client.snapshot()).readiness, "stale");
    }
    const { client } = observerHarness("beta-archive", (path) => syntheticResponse(path, NOW - 10_000_000, true));
    assert.equal((await client.snapshot()).readiness, "archive");
    for (const key of ["latest_block_height", "latest_block_hash", "latest_app_hash", "catching_up"]) {
      const bad = observerHarness("beta-archive", (path, body) => {
        if (path === "/status") {
          const result = body.result as { sync_info: Record<string, unknown> };
          result.sync_info[key] = key === "catching_up" ? false : key === "latest_block_height" ? "11" : "f".repeat(64);
        }
        return body;
      });
      await assert.rejects(bad.client.snapshot(), /Archive checkpoint/);
    }
  });
  it("keeps a height regression stale until the previous high-water mark is reached", async () => {
    let reads = 0;
    const { client } = observerHarness("beta-active", (path, body) => {
      if (path === "/status" && ++reads > 1) {
        const result = body.result as { sync_info: Record<string, unknown> };
        result.sync_info.latest_block_height = "9";
      }
      return body;
    });
    assert.equal((await client.snapshot()).readiness, "ready");
    assert.equal((await client.snapshot()).readiness, "stale");
    assert.equal((await client.snapshot()).readiness, "stale");
  });
  it("requires native base64 entries before counting block transactions", async () => {
    for (const txs of [[null, {}, 123], [""], ["not base64"], ["AR=="], { length: 3 }]) {
      const { client } = observerHarness("preview", (path, body) => {
        if (path.startsWith("/block?")) (body.result as { block: { data: { txs: unknown } } }).block.data.txs = txs;
        return body;
      });
      await assert.rejects(client.snapshot());
    }
    const { client } = observerHarness("preview", (path, body) => {
      if (path.startsWith("/block?")) (body.result as { block: { data: { txs: unknown } } }).block.data.txs = ["AQID", "BA=="];
      return body;
    });
    assert.equal((await client.snapshot()).block.transactionCount, 2);
  });
  it("keeps malformed or duplicate validator entries unknown instead of counting them", async () => {
    const valid = syntheticValidator();
    for (const validators of [[null], [{}], [123], [{ ...valid, address: "synthetic" }],
      [{ ...valid, voting_power: "0" }], [{ ...valid, voting_power: "01" }],
      [{ ...valid, pub_key: { ...valid.pub_key, value: "%%%" } }], [valid, valid]]) {
      const { client } = observerHarness("preview", (path, body) => path.startsWith("/validators?")
        ? { result: { total: String(validators.length), validators } } : body);
      const snapshot = await client.snapshot();
      assert.equal(snapshot.validators, null);
      assert.ok(snapshot.issues.some((issue) => issue.startsWith("Validator count unknown")));
    }
  });
  it("uses the beta gateway age boundary without changing legacy readiness", async () => {
    for (const age of [-10_000, 0, 30_000]) assert.equal(observerReadinessAtAge("ready", age), "ready");
    for (const age of [-10_001, 30_001, 45_000, Number.NaN]) assert.equal(observerReadinessAtAge("ready", age), "stale");
    assert.equal(observerReadinessAtAge("archive", 86_400_000), "archive");
    assert.equal(observerReadinessAtAge("stale", 1000), "stale");
    assert.equal(observerReadinessAtAge("syncing", 1000), "syncing");
    assert.equal(BLOCK_FRESHNESS_WINDOW_MS, 75_000);
    assert.equal(assessNetworkReadiness({ blockAgeMs: 45_000, catchingUp: false, chainMatches: true, regressed: false }), "ready");
    for (const age of [31_000, 45_000]) {
      const { client } = observerHarness("beta-active", (path) => syntheticResponse(path, NOW - age + 1000));
      assert.equal((await client.snapshot()).readiness, "stale");
    }
  });
  it("preserves unknown versus actual zero and returns known tx execution results", async () => {
    const { client } = observerHarness("preview", (path, body) => path.includes("supply") || path.includes("validators") ? new Response("", { status: 503 }) : body);
    const snapshot = await client.snapshot();
    assert.equal(snapshot.supplyUzrn, null); assert.equal(snapshot.validators, null); assert.equal(snapshot.issues.length, 2);
    assert.equal(await client.balance(NATIVE_ADDRESS), "0");
    assert.deepEqual(await client.transaction(BLOCK_HASH), { hash: BLOCK_HASH, height: 9, code: 0 });
    await assert.rejects(client.block(11), /exceeds/);
    await assert.rejects(client.transaction("not a hash"));
  });
  it("does not turn missing transactions, malformed balances or failed identity into zero", async () => {
    for (const response of [new Response("", { status: 404 }), { error: { message: "tx not found" } }, { result: {} }]) {
      const { client } = observerHarness("preview", (path, body) => path.startsWith("/tx?") ? response : body);
      await client.snapshot(); await assert.rejects(client.transaction(BLOCK_HASH));
    }
    const bad = observerHarness("preview", (path, body) => path.includes("/balances/") ? { balance: { denom: "uzrn" } } : body);
    await bad.client.snapshot(); await assert.rejects(bad.client.balance(NATIVE_ADDRESS));
    let calls = 0;
    const client = createObserverClient({ mode: "unconfigured", error: "missing bundle" }, "https://dashboard.invalid", { fetch: async () => { calls += 1; return Response.json({}); } });
    await assert.rejects(client.snapshot(), /unconfigured/); assert.equal(calls, 0);
  });
});

describe("dashboard liquidity registry pagination", () => {
  it("consumes total and cursor metadata until the registry is complete", async () => {
    const urls: URL[] = [];
    const pages = [
      {
        pools: [closedPool(1)],
        pagination: { next_key: "AQID", total: "2" },
      },
      {
        pools: [closedPool(2)],
        pagination: { next_key: null, total: "0" },
      },
    ];
    globalThis.fetch = async (input) => {
      urls.push(new URL(String(input)));
      return Response.json(pages.shift());
    };

    const registry = await getLiquidityPoolRegistry();

    assert.equal(registry.complete, true);
    assert.equal(registry.total, "2");
    assert.equal(registry.recordCap, LIQUIDITY_POOL_RECORD_CAP);
    assert.deepEqual(
      registry.pools.map((pool) => pool.id),
      ["pool-1", "pool-2"],
    );
    assert.equal(
      urls[0]?.searchParams.get("pagination.limit"),
      String(LIQUIDITY_POOL_PAGE_LIMIT),
    );
    assert.equal(
      urls[0]?.searchParams.get("pagination.count_total"),
      "true",
    );
    assert.equal(urls[0]?.searchParams.get("pagination.key"), null);
    assert.equal(urls[1]?.searchParams.get("pagination.key"), "AQID");
    assert.equal(urls[1]?.searchParams.get("pagination.count_total"), null);
  });

  it("stops at 500 records and preserves the exact larger chain total", async () => {
    let pageNumber = 0;
    globalThis.fetch = async () => {
      const firstId = pageNumber * LIQUIDITY_POOL_PAGE_LIMIT + 1;
      const pools = Array.from(
        { length: LIQUIDITY_POOL_PAGE_LIMIT },
        (_, index) => closedPool(firstId + index),
      );
      pageNumber += 1;
      return Response.json({
        pools,
        pagination: {
          next_key: Buffer.from([pageNumber]).toString("base64"),
          total: pageNumber === 1 ? "501" : "0",
        },
      });
    };

    const registry = await getLiquidityPoolRegistry();

    assert.equal(pageNumber, 5);
    assert.equal(registry.pools.length, LIQUIDITY_POOL_RECORD_CAP);
    assert.equal(registry.total, "501");
    assert.equal(registry.complete, false);
  });

  it("fails closed on repeated cursors, duplicate records, and totals", async () => {
    for (const pages of [
      [
        {
          pools: [closedPool(1)],
          pagination: { next_key: "AQID", total: "3" },
        },
        {
          pools: [closedPool(2)],
          pagination: { next_key: "AQID", total: "0" },
        },
      ],
      [
        {
          pools: [closedPool(1)],
          pagination: { next_key: "AQID", total: "2" },
        },
        {
          pools: [closedPool(1)],
          pagination: { next_key: null, total: "0" },
        },
      ],
      [
        {
          pools: [closedPool(1)],
          pagination: { next_key: null, total: "2" },
        },
      ],
      [
        {
          pools: [closedPool(1)],
          pagination: { next_key: null, total: "10001" },
        },
      ],
    ]) {
      const queue = [...pages];
      globalThis.fetch = async () => Response.json(queue.shift());
      await assert.rejects(getLiquidityPoolRegistry());
    }
  });
});
