import assert from "node:assert/strict";
import { afterEach, describe, it } from "node:test";

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
