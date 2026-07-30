import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  normalizeLiquidityParams,
  normalizeLiquidityPool,
} from "../src/liquidity";

const CREATOR = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";

function pool(
  overrides: Readonly<Record<string, unknown>> = {},
): Record<string, unknown> {
  return {
    poolId: "pool-1",
    denomA: "uzrn",
    denomB: "uatom",
    reserveA: "1000000",
    reserveB: "2000000",
    swapFeeBps: "3000",
    lpTokenSupply: "1414213",
    lpDenom: "lp/pool-1",
    creator: CREATOR,
    createdAtBlock: "42",
    locked: false,
    ...overrides,
  };
}

function params(
  overrides: Readonly<Record<string, unknown>> = {},
): {
  params: Record<string, unknown>;
} {
  return {
    params: {
      defaultSwapFeeBps: "3000",
      protocolFeeBps: "450000",
      minInitialLiquidity: "1000000",
      twapWindowBlocks: "120",
      maxPools: "16",
      minReserve: "1",
      billingQuoteDenoms: [],
      allowedPoolDenoms: [],
      poolCreators: [],
      ...overrides,
    },
  };
}

describe("dashboard liquidity lifecycle parsing", () => {
  it("labels a legacy response without inventing an ACTIVE lifecycle", () => {
    const parsed = normalizeLiquidityPool(pool());
    assert.ok(parsed);
    assert.equal(parsed.status, "PRE_V4");
    assert.equal(parsed.closedAtBlock, null);
  });

  it("parses every valid v4 lifecycle name and numeric representation", () => {
    for (const [wire, expected] of [
      ["POOL_STATUS_ACTIVE", "ACTIVE"],
      ["SWAPS_PAUSED", "SWAPS_PAUSED"],
      [3, "EXIT_ONLY"],
    ] as const) {
      assert.equal(normalizeLiquidityPool(pool({ status: wire }))?.status, expected);
    }

    const closed = normalizeLiquidityPool(
      pool({
        reserveA: "0",
        reserveB: "0",
        lpTokenSupply: "0",
        status: "POOL_STATUS_CLOSED",
        closedAtBlock: "99",
      }),
    );
    assert.ok(closed);
    assert.equal(closed.status, "CLOSED");
    assert.equal(closed.closedAtBlock, 99);
  });

  it("rejects every explicit UNSPECIFIED lifecycle representation", () => {
    for (const status of [
      0,
      "0",
      "UNSPECIFIED",
      "POOL_STATUS_UNSPECIFIED",
    ]) {
      assert.equal(normalizeLiquidityPool(pool({ status })), null);
    }
  });

  it("enforces canonical pool/LP identities and the lifetime ID cap", () => {
    const maximumId = "pool-10000";
    assert.ok(
      normalizeLiquidityPool(
        pool({ poolId: maximumId, lpDenom: `lp/${maximumId}` }),
      ),
    );
    for (const invalidId of [
      "pool-0",
      "pool-01",
      "pool-10001",
      "pool-18446744073709551616",
      "pool-id",
    ]) {
      assert.equal(
        normalizeLiquidityPool(
          pool({ poolId: invalidId, lpDenom: `lp/${invalidId}` }),
        ),
        null,
      );
    }
    assert.equal(
      normalizeLiquidityPool(pool({ lpDenom: "lp/pool-2" })),
      null,
    );
  });

  it("enforces the ZRN pair, creator, fee, and amount bounds", () => {
    assert.equal(
      normalizeLiquidityPool(pool({ denomA: "uatom", denomB: "uosmo" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(pool({ denomB: "uzrn" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(pool({ denomB: "x" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(pool({ creator: "zrn1invalid" })),
      null,
    );
    assert.ok(normalizeLiquidityPool(pool({ swapFeeBps: "100000" })));
    assert.equal(
      normalizeLiquidityPool(pool({ swapFeeBps: "100001" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(
        pool({ reserveA: (1n << 256n).toString() }),
      ),
      null,
    );
  });

  it("rejects malformed amounts, locks, and contradictory tombstones", () => {
    assert.equal(
      normalizeLiquidityPool(pool({ reserveA: "01", status: "ACTIVE" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(pool({ locked: "false", status: "ACTIVE" })),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(
        pool({ closedAtBlock: null, status: "ACTIVE" }),
      ),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(
        pool({ status: "CLOSED", closedAtBlock: "99" }),
      ),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(
        pool({
          reserveA: "0",
          reserveB: "0",
          lpTokenSupply: "0",
          status: "CLOSED",
          createdAtBlock: "100",
          closedAtBlock: "99",
        }),
      ),
      null,
    );
    assert.equal(
      normalizeLiquidityPool(
        pool({ reserveA: "0", status: "EXIT_ONLY" }),
      ),
      null,
    );
  });
});

describe("dashboard liquidity admission and oracle params", () => {
  it("shows empty quote and creation allowlists as fail-closed", () => {
    const parsed = normalizeLiquidityParams(params());
    assert.ok(parsed);
    assert.equal(parsed.maxPools, 16);
    assert.equal(parsed.minReserve, "1");
    assert.deepEqual(parsed.billingQuoteDenoms, []);
    assert.equal(parsed.billingOracleEnabled, false);
    assert.equal(parsed.billingOraclePolicy, "DISABLED_FAIL_CLOSED");
    assert.deepEqual(parsed.allowedPoolDenoms, []);
    assert.deepEqual(parsed.poolCreators, []);
    assert.equal(parsed.poolCreationEnabled, false);
    assert.equal(parsed.poolCreationPolicy, "DISABLED_FAIL_CLOSED");
  });

  it("exposes both governed admission lists without broadening them", () => {
    const parsed = normalizeLiquidityParams(
      params({
        billing_quote_denoms: ["uatom"],
        billingQuoteDenoms: undefined,
        allowed_pool_denoms: ["uatom", "uosmo"],
        allowedPoolDenoms: undefined,
        pool_creators: [CREATOR],
        poolCreators: undefined,
      }),
    );
    assert.ok(parsed);
    assert.deepEqual(parsed.billingQuoteDenoms, ["uatom"]);
    assert.deepEqual(parsed.allowedPoolDenoms, ["uatom", "uosmo"]);
    assert.deepEqual(parsed.poolCreators, [CREATOR]);
    assert.equal(parsed.billingOracleEnabled, true);
    assert.equal(parsed.poolCreationEnabled, true);
    assert.equal(
      parsed.poolCreationPolicy,
      "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY",
    );
    assert.equal(
      parsed.billingOraclePolicy,
      "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED",
    );
  });

  it("rejects duplicate, native, malformed, and noncanonical allowlist values", () => {
    assert.equal(
      normalizeLiquidityParams(
        params({ allowedPoolDenoms: ["uatom", "uatom"] }),
      ),
      null,
    );
    assert.equal(
      normalizeLiquidityParams(params({ allowedPoolDenoms: ["uzrn"] })),
      null,
    );
    assert.equal(
      normalizeLiquidityParams(params({ poolCreators: ["zrn1invalid"] })),
      null,
    );
    assert.equal(
      normalizeLiquidityParams(
        params({ poolCreators: [`${CREATOR.slice(0, -1)}q`] }),
      ),
      null,
    );
    assert.equal(
      normalizeLiquidityParams(params({ minReserve: "01" })),
      null,
    );
  });

  it("enforces v4 fee, open-pool, TWAP, and amount bounds", () => {
    assert.ok(
      normalizeLiquidityParams(
        params({
          defaultSwapFeeBps: "100000",
          maxPools: "64",
          twapWindowBlocks: "10000",
        }),
      ),
    );
    for (const overrides of [
      { defaultSwapFeeBps: "100001" },
      { protocolFeeBps: "1000001" },
      { maxPools: "0" },
      { maxPools: "65" },
      { twapWindowBlocks: "0" },
      { twapWindowBlocks: "10001" },
      { minInitialLiquidity: (1n << 256n).toString() },
      { minReserve: (1n << 256n).toString() },
    ]) {
      assert.equal(normalizeLiquidityParams(params(overrides)), null);
    }
  });
});
