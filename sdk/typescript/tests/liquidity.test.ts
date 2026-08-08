import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { toBech32 } from "@cosmjs/encoding";

import {
  COSMOS_AMOUNT_MAX,
  LIQUIDITY_FEE_SCALE,
  LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE,
  LiquidityClientError,
  MSG_CREATE_POOL_TYPE_URL,
  MSG_SUBMIT_PROPOSAL_TYPE_URL,
  MSG_SWAP_TYPE_URL,
  MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
  ZeroneLiquidityRestClient,
  createExactInSwapPlan,
  createLiquidityAdmissionProposal,
  createPoolMessage,
  discloseLiquiditySwapFee,
  minimumOutputForSlippage,
  parseCanonicalPositiveAmount,
  quoteConstantProductExactIn,
  timeoutHeightAfter,
  type ExactInLiquidityAdapter,
  type ExactInLiquidityQuote,
  type ExactInLiquidityQuoteRequest,
  type LiquidityClientErrorCode,
  type ZeroneLiquidityParams,
} from "../src/liquidity";
import {
  MsgCreatePool,
  MsgUpdateParams,
  type MsgSwap,
} from "../src/generated/zerone/liquiditypool/v1/tx";

const AUTHORITY = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const PROPOSER = "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r";

function liquidityParams(
  overrides: Partial<ZeroneLiquidityParams> = {},
): ZeroneLiquidityParams {
  return {
    defaultSwapFeeMillionths: 3_000n,
    maxPools: 16n,
    minInitialLiquidity: "1000000",
    twapWindowBlocks: 120n,
    protocolFeeMillionths: 450_000n,
    minReserve: "1",
    billingQuoteDenoms: [],
    billingOracleEnabled: false,
    billingOraclePolicy: "DISABLED_FAIL_CLOSED",
    allowedPoolDenoms: [],
    poolCreators: [],
    poolCreationEnabled: false,
    poolCreationPolicy: "DISABLED_FAIL_CLOSED",
    ...overrides,
  };
}

function assertLiquidityError(
  operation: () => unknown,
  code: LiquidityClientErrorCode,
): void {
  assert.throws(
    operation,
    (error: unknown) =>
      error instanceof LiquidityClientError && error.code === code,
  );
}

function pool(
  overrides: Readonly<Record<string, unknown>> = {},
): Record<string, unknown> {
  return {
    poolId: "pool-7",
    denomA: "uzrn",
    denomB: "uatom",
    reserveA: "1000000",
    reserveB: "2000000",
    swapFeeBps: "3000",
    lpTokenSupply: "1414213",
    lpDenom: "lp/pool-7",
    creator: AUTHORITY,
    createdAtBlock: "42",
    locked: false,
    status: "POOL_STATUS_ACTIVE",
    closedAtBlock: "0",
    ...overrides,
  };
}

function paramsResponse(
  overrides: Readonly<Record<string, unknown>> = {},
): Record<string, unknown> {
  return {
    params: {
      defaultSwapFeeBps: "3000",
      maxPools: "16",
      minInitialLiquidity: "1000000",
      twapWindowBlocks: "120",
      protocolFeeBps: "450000",
      minReserve: "1",
      billingQuoteDenoms: [],
      allowedPoolDenoms: [],
      poolCreators: [],
      ...overrides,
    },
  };
}

describe("canonical liquidity amounts and quotes", () => {
  it("accepts exact positive 256-bit amounts without JS number coercion", () => {
    assert.equal(parseCanonicalPositiveAmount("1"), 1n);
    assert.equal(
      parseCanonicalPositiveAmount(COSMOS_AMOUNT_MAX.toString()),
      COSMOS_AMOUNT_MAX,
    );
  });

  it("rejects zero, signs, whitespace, decimals, leading zeroes, and overflow", () => {
    for (const invalid of [
      "",
      "0",
      "00",
      "01",
      "+1",
      "-1",
      " 1",
      "1 ",
      "1.0",
      "1e3",
      (COSMOS_AMOUNT_MAX + 1n).toString(),
    ]) {
      assertLiquidityError(
        () => parseCanonicalPositiveAmount(invalid),
        "INVALID_AMOUNT",
      );
    }
  });

  it("matches the chain's floor-rounded constant-product exact-in formula", () => {
    const quote = quoteConstantProductExactIn({
      reserveIn: "1000000",
      reserveOut: "2000000",
      tokenInAmount: "100000",
      swapFeeMillionths: 3_000n,
      slippageMillionths: 5_000n,
    });

    assert.deepEqual(quote, {
      reserveIn: "1000000",
      reserveOut: "2000000",
      tokenInAmount: "100000",
      effectiveTokenIn: "99700",
      weightedTokenIn: "99700000000",
      feeAmount: "300",
      tokenOutAmount: "181322",
      minimumTokenOut: "180415",
      swapFeeMillionths: 3_000n,
      slippageMillionths: 5_000n,
      priceImpactMillionths: 90_661n,
    });
    assert.equal(LIQUIDITY_FEE_SCALE, 1_000_000n);
  });

  it("preserves a fractional curve fee when the reported fee rounds to zero", () => {
    const quote = quoteConstantProductExactIn({
      reserveIn: "1000",
      reserveOut: "2004",
      tokenInAmount: "1",
      swapFeeMillionths: 3_000n,
      slippageMillionths: 5_000n,
    });

    assert.deepEqual(quote, {
      reserveIn: "1000",
      reserveOut: "2004",
      tokenInAmount: "1",
      effectiveTokenIn: "1",
      weightedTokenIn: "997000",
      feeAmount: "0",
      tokenOutAmount: "1",
      minimumTokenOut: "1",
      swapFeeMillionths: 3_000n,
      slippageMillionths: 5_000n,
      priceImpactMillionths: 499_496n,
    });
  });

  it("always derives a nonzero minimum output and rejects unsafe controls", () => {
    assert.equal(minimumOutputForSlippage("1", 999_999n), "1");
    assert.equal(minimumOutputForSlippage("100", 10_000n), "99");
    assertLiquidityError(
      () => minimumOutputForSlippage("100", 1_000_000n),
      "INVALID_SLIPPAGE",
    );
    assertLiquidityError(
      () =>
        quoteConstantProductExactIn({
          reserveIn: "999999999999",
          reserveOut: "1",
          tokenInAmount: "1",
          swapFeeMillionths: 0n,
          slippageMillionths: 0n,
        }),
      "QUOTE_TOO_SMALL",
    );
  });

  it("discloses v5 no-skim and legacy directional protocol fees", () => {
    assert.deepEqual(
      discloseLiquiditySwapFee({
        tokenInDenom: "uzrn",
        feeAmount: "30",
        protocolFeeMillionths: 0n,
      }),
      {
        policy: "NO_PROTOCOL_SKIM",
        tokenInDenom: "uzrn",
        totalFeeAmount: "30",
        poolRetainedFeeAmount: "30",
        protocolFeeAmount: "0",
        protocolFeeMillionths: 0n,
        protocolFeeDestinationModule: null,
      },
    );

    assert.deepEqual(
      discloseLiquiditySwapFee({
        tokenInDenom: "uzrn",
        feeAmount: "30",
        protocolFeeMillionths: 450_000n,
      }),
      {
        policy: "LEGACY_ZRN_INPUT_PROTOCOL_SKIM",
        tokenInDenom: "uzrn",
        totalFeeAmount: "30",
        poolRetainedFeeAmount: "17",
        protocolFeeAmount: "13",
        protocolFeeMillionths: 450_000n,
        protocolFeeDestinationModule:
          LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE,
      },
    );

    assert.deepEqual(
      discloseLiquiditySwapFee({
        tokenInDenom: "uatom",
        feeAmount: "30",
        protocolFeeMillionths: 450_000n,
      }),
      {
        policy: "LEGACY_ZRN_INPUT_PROTOCOL_SKIM",
        tokenInDenom: "uatom",
        totalFeeAmount: "30",
        poolRetainedFeeAmount: "30",
        protocolFeeAmount: "0",
        protocolFeeMillionths: 450_000n,
        protocolFeeDestinationModule: null,
      },
    );

    assert.equal(
      discloseLiquiditySwapFee({
        tokenInDenom: "uzrn",
        feeAmount: "1",
        protocolFeeMillionths: 450_000n,
      }).protocolFeeAmount,
      "0",
    );
    assertLiquidityError(
      () =>
        discloseLiquiditySwapFee({
          tokenInDenom: "uzrn",
          feeAmount: "30",
          protocolFeeMillionths: 1_000_001n,
        }),
      "INVALID_FEE",
    );
  });
});

describe("Zerone liquidity REST adapter", () => {
  it("parses v4 pool, pagination, params, TWAP, and simulation queries", async () => {
    const requested: URL[] = [];
    const client = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example/api/",
      fetch: async (input) => {
        const url = new URL(String(input));
        requested.push(url);
        if (url.pathname.endsWith("/pools/pool-7")) {
          return Response.json({ pool: pool() });
        }
        if (url.pathname.endsWith("/pools")) {
          return Response.json({
            pools: [
              pool(),
              pool({
                poolId: "pool-2",
                reserveA: "0",
                reserveB: "0",
                lpTokenSupply: "0",
                lpDenom: "lp/pool-2",
                status: 4,
                closedAtBlock: "99",
              }),
            ],
            pagination: { nextKey: "AQ==", total: "2" },
          });
        }
        if (url.pathname.endsWith("/params")) {
          return Response.json({
            params: {
              default_swap_fee_bps: "3000",
              max_pools: "16",
              min_initial_liquidity: "1000000",
              twap_window_blocks: "120",
              protocol_fee_bps: "450000",
              min_reserve: "1",
              billing_quote_denoms: ["uatom"],
              allowed_pool_denoms: ["uatom"],
              pool_creators: [PROPOSER],
            },
          });
        }
        if (url.pathname.endsWith("/twap/pool-7")) {
          return Response.json({ twap: "2000000", window_used: "90" });
        }
        if (url.pathname.endsWith("/simulate/pool-7")) {
          return Response.json(
            {
              result: {
                token_out_denom: "uatom",
                token_out_amount: "181322",
                fee_amount: "300",
                price_impact_bps: "90661",
              },
            },
            {
              headers: {
                "Grpc-Metadata-X-Cosmos-Block-Height": "1234",
              },
            },
          );
        }
        return new Response("", { status: 404 });
      },
    });

    const queriedPool = await client.pool("pool-7");
    assert.equal(queriedPool.status, "ACTIVE");
    assert.equal(queriedPool.swapFeeMillionths, 3_000n);

    const page = await client.pools({ limit: 16, key: "AQ==" });
    assert.equal(page.pools[1]?.status, "CLOSED");
    assert.equal(page.pools[1]?.closedAtBlock, 99n);
    assert.equal(page.nextKey, "AQ==");
    assert.equal(page.total, 2n);

    const params = await client.params();
    assert.equal(params.maxPools, 16n);
    assert.equal(params.minReserve, "1");
    assert.deepEqual(params.billingQuoteDenoms, ["uatom"]);
    assert.equal(params.billingOracleEnabled, true);
    assert.equal(
      params.billingOraclePolicy,
      "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED",
    );
    assert.deepEqual(params.allowedPoolDenoms, ["uatom"]);
    assert.deepEqual(params.poolCreators, [PROPOSER]);
    assert.equal(params.poolCreationEnabled, true);

    assert.deepEqual(await client.twap("pool-7", "uzrn", 90n), {
      price: "2000000",
      windowUsed: 90n,
    });
    assert.deepEqual(
      await client.simulateSwap("pool-7", "uzrn", "100000"),
      {
        tokenOutDenom: "uatom",
        tokenOutAmount: "181322",
        feeAmount: "300",
        priceImpactMillionths: 90_661n,
        observedHeight: 1_234n,
      },
    );

    const poolsUrl = requested.find((url) => url.pathname.endsWith("/pools"));
    assert.equal(poolsUrl?.searchParams.get("pagination.limit"), "16");
    assert.equal(poolsUrl?.searchParams.get("pagination.key"), "AQ==");
    const twapUrl = requested.find((url) =>
      url.pathname.endsWith("/twap/pool-7"),
    );
    assert.equal(twapUrl?.searchParams.get("baseDenom"), "uzrn");
    assert.equal(twapUrl?.searchParams.get("window"), "90");
    const simulationUrl = requested.find((url) =>
      url.pathname.endsWith("/simulate/pool-7"),
    );
    assert.equal(simulationUrl?.searchParams.get("tokenInDenom"), "uzrn");
    assert.equal(
      simulationUrl?.searchParams.get("tokenInAmount"),
      "100000",
    );
  });

  it("prepares an expiring exact-in plan from authoritative simulation", async () => {
    const requestedPaths: string[] = [];
    const client = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async (input) => {
        const url = new URL(String(input));
        requestedPaths.push(url.pathname);
        if (url.pathname.endsWith("/params")) {
          return Response.json(paramsResponse({ protocolFeeBps: "0" }));
        }
        if (url.pathname.endsWith("/simulate/pool-7")) {
          return Response.json(
            {
              result: {
                tokenOutDenom: "uatom",
                // Deliberately differs from the local reserve-vector quote.
                tokenOutAmount: "190000",
                feeAmount: "300",
                priceImpactBps: "50000",
              },
            },
            { headers: { "X-Cosmos-Block-Height": "1000" } },
          );
        }
        return new Response("", { status: 404 });
      },
    });

    const prepared = await client.prepareExactInSwap({
      sender: PROPOSER,
      poolId: "pool-7",
      tokenInDenom: "uzrn",
      tokenInAmount: "100000",
      expectedTokenOutDenom: "uatom",
      slippageMillionths: 5_000n,
      lifetimeBlocks: 20n,
    });

    assert.equal(prepared.simulation.observedHeight, 1_000n);
    assert.equal(prepared.minimumTokenOut, "189050");
    assert.deepEqual(prepared.feeDisclosure, {
      policy: "NO_PROTOCOL_SKIM",
      tokenInDenom: "uzrn",
      totalFeeAmount: "300",
      poolRetainedFeeAmount: "300",
      protocolFeeAmount: "0",
      protocolFeeMillionths: 0n,
      protocolFeeDestinationModule: null,
    });
    assert.equal(prepared.plan.timeoutHeight, 1_020n);
    assert.deepEqual(prepared.plan.messages[0]?.value as MsgSwap, {
      sender: PROPOSER,
      poolId: "pool-7",
      tokenInDenom: "uzrn",
      tokenInAmount: "100000",
      minTokenOut: "189050",
    });
    assert.equal(
      requestedPaths.some((path) => path.endsWith("/pools/pool-7")),
      false,
      "transaction preparation must not substitute a local pool quote",
    );
  });

  it("rejects missing, malformed, conflicting, and mismatched observations", async () => {
    const invalidHeightHeaders: HeadersInit[] = [
      {},
      { "X-Cosmos-Block-Height": "0" },
      { "X-Cosmos-Block-Height": "01" },
      {
        "X-Cosmos-Block-Height": "1000",
        "Grpc-Metadata-X-Cosmos-Block-Height": "1001",
      },
    ];
    for (const headers of invalidHeightHeaders) {
      const client = new ZeroneLiquidityRestClient({
        baseUrl: "https://rest.example",
        fetch: async () =>
          Response.json(
            {
              result: {
                tokenOutDenom: "uatom",
                tokenOutAmount: "190000",
                feeAmount: "300",
                priceImpactBps: "50000",
              },
            },
            { headers },
          ),
      });
      await assert.rejects(
        () => client.simulateSwap("pool-7", "uzrn", "100000"),
        (error: unknown) =>
          error instanceof LiquidityClientError &&
          error.code === "INVALID_RESPONSE",
      );
    }

    const mismatch = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async (input) =>
        new URL(String(input)).pathname.endsWith("/params")
          ? Response.json(paramsResponse({ protocolFeeBps: "0" }))
          : Response.json(
              {
                result: {
                  tokenOutDenom: "uosmo",
                  tokenOutAmount: "190000",
                  feeAmount: "300",
                  priceImpactBps: "50000",
                },
              },
              { headers: { "X-Cosmos-Block-Height": "1000" } },
            ),
    });
    await assert.rejects(
      () =>
        mismatch.prepareExactInSwap({
          sender: PROPOSER,
          poolId: "pool-7",
          tokenInDenom: "uzrn",
          tokenInAmount: "100000",
          expectedTokenOutDenom: "uatom",
          slippageMillionths: 5_000n,
          lifetimeBlocks: 20n,
        }),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "INVALID_RESPONSE",
    );
  });

  it("computes a local quote only for an explicitly ACTIVE v4 pool", async () => {
    const active = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async (input) =>
        new URL(String(input)).pathname.endsWith("/params")
          ? Response.json(paramsResponse())
          : Response.json({ pool: pool() }),
    });
    const quote = await active.quoteExactIn({
      poolId: "pool-7",
      tokenInDenom: "uzrn",
      tokenInAmount: "100000",
      slippageMillionths: 5_000n,
    });
    assert.equal(quote.adapterId, "zerone/liquiditypool/v1");
    assert.equal(quote.poolStatus, "ACTIVE");
    assert.equal(quote.tokenOutDenom, "uatom");
    assert.equal(quote.minimumTokenOut, "180415");

    const preV4 = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async (input) => {
        if (new URL(String(input)).pathname.endsWith("/params")) {
          return Response.json(paramsResponse());
        }
        const legacy = pool();
        delete legacy.status;
        delete legacy.closedAtBlock;
        return Response.json({ pool: legacy });
      },
    });
    await assert.rejects(
      () =>
        preV4.quoteExactIn({
          poolId: "pool-7",
          tokenInDenom: "uzrn",
          tokenInAmount: "100000",
          slippageMillionths: 5_000n,
        }),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "POOL_NOT_ACTIVE",
    );
  });

  it("enforces pool identity, lock state, and the governed reserve floor", async () => {
    const overflowId = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () => {
        throw new Error("invalid pool IDs must fail before fetching");
      },
    });
    for (const impossibleId of [
      "pool-10001",
      "pool-18446744073709551616",
    ]) {
      await assert.rejects(
        () => overflowId.pool(impossibleId),
        (error: unknown) =>
          error instanceof LiquidityClientError &&
          error.code === "INVALID_POOL",
      );
    }

    const mismatchedLpDenom = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () =>
        Response.json({ pool: pool({ lpDenom: "lp/pool-8" }) }),
    });
    await assert.rejects(
      () => mismatchedLpDenom.pool("pool-7"),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "INVALID_RESPONSE",
    );

    for (const testCase of [
      {
        name: "persistent lock",
        pool: pool({ locked: true }),
        params: paramsResponse(),
        code: "POOL_NOT_ACTIVE" as const,
      },
      {
        name: "minimum reserve",
        pool: pool(),
        params: paramsResponse({ minReserve: "1900000" }),
        code: "QUOTE_TOO_SMALL" as const,
      },
    ]) {
      const client = new ZeroneLiquidityRestClient({
        baseUrl: "https://rest.example",
        fetch: async (input) =>
          new URL(String(input)).pathname.endsWith("/params")
            ? Response.json(testCase.params)
            : Response.json({ pool: testCase.pool }),
      });
      await assert.rejects(
        () =>
          client.quoteExactIn({
            poolId: "pool-7",
            tokenInDenom: "uzrn",
            tokenInAmount: "100000",
            slippageMillionths: 5_000n,
          }),
        (error: unknown) =>
          error instanceof LiquidityClientError &&
          error.code === testCase.code,
        testCase.name,
      );
    }
  });

  it("bounds payloads and rejects malformed lifecycle state", async () => {
    const tooLarge = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      maximumResponseBytes: 10,
      fetch: async () =>
        new Response('{"params":{}}', {
          headers: { "Content-Length": "13" },
        }),
    });
    await assert.rejects(
      () => tooLarge.params(),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "RESPONSE_TOO_LARGE",
    );

    const malformed = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () =>
        Response.json({
          pool: pool({
            status: "POOL_STATUS_CLOSED",
            closedAtBlock: "99",
          }),
        }),
    });
    await assert.rejects(
      () => malformed.pool("pool-7"),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "INVALID_RESPONSE",
    );

    const unspecified = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () =>
        Response.json({
          pool: pool({ status: "POOL_STATUS_UNSPECIFIED" }),
        }),
    });
    await assert.rejects(
      () => unspecified.pool("pool-7"),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "INVALID_RESPONSE",
    );

    const impossibleTotal = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () =>
        Response.json({
          pools: [],
          pagination: { total: "10001" },
        }),
    });
    await assert.rejects(
      () => impossibleTotal.pools(),
      (error: unknown) =>
        error instanceof LiquidityClientError &&
        error.code === "INVALID_RESPONSE",
    );

    const legacyDisabled = new ZeroneLiquidityRestClient({
      baseUrl: "https://rest.example",
      fetch: async () => Response.json(paramsResponse({
        maxPools: "0",
        allowedPoolDenoms: ["uatom"],
        poolCreators: [PROPOSER],
      })),
    });
    const observedLegacy = await legacyDisabled.params();
    assert.equal(observedLegacy.maxPools, 0n);
    assert.equal(observedLegacy.poolCreationEnabled, false);

    for (const invalidParams of [
      paramsResponse({ maxPools: "65" }),
      paramsResponse({ twapWindowBlocks: "0" }),
      paramsResponse({ twapWindowBlocks: "10001" }),
    ]) {
      const client = new ZeroneLiquidityRestClient({
        baseUrl: "https://rest.example",
        fetch: async () => Response.json(invalidParams),
      });
      await assert.rejects(
        () => client.params(),
        (error: unknown) =>
          error instanceof LiquidityClientError &&
          error.code === "INVALID_RESPONSE",
      );
    }
  });

  it("accepts an injected external adapter without an Osmosis dependency", async () => {
    const injected: ExactInLiquidityAdapter = {
      id: "example/osmosis-router",
      async quoteExactIn(
        request: ExactInLiquidityQuoteRequest,
      ): Promise<ExactInLiquidityQuote> {
        return {
          adapterId: this.id,
          poolId: request.poolId,
          tokenInDenom: request.tokenInDenom,
          tokenOutDenom: "uosmo",
          poolStatus: "ACTIVE",
          reserveIn: "1",
          reserveOut: "1",
          tokenInAmount: request.tokenInAmount,
          effectiveTokenIn: request.tokenInAmount,
          weightedTokenIn: (
            BigInt(request.tokenInAmount) * LIQUIDITY_FEE_SCALE
          ).toString(),
          feeAmount: "0",
          tokenOutAmount: "1",
          minimumTokenOut: "1",
          swapFeeMillionths: 0n,
          slippageMillionths: request.slippageMillionths,
          priceImpactMillionths: 0n,
        };
      },
    };

    const quote = await injected.quoteExactIn({
      poolId: "1",
      tokenInDenom: "uzrn",
      tokenInAmount: "1",
      slippageMillionths: 10_000n,
    });
    assert.equal(quote.adapterId, "example/osmosis-router");
  });
});

describe("liquidity transaction helpers", () => {
  it("builds a strict swap plan with an explicit TxBody timeout height", () => {
    const timeoutHeight = timeoutHeightAfter(1_000n, 20n);
    const plan = createExactInSwapPlan({
      sender: PROPOSER,
      poolId: "pool-7",
      tokenInDenom: "uzrn",
      tokenInAmount: "100000",
      minimumTokenOut: "180415",
      timeoutHeight,
    });

    assert.equal(plan.timeoutHeight, 1_020n);
    assert.equal(plan.messages[0]?.typeUrl, MSG_SWAP_TYPE_URL);
    assert.deepEqual(plan.messages[0]?.value as MsgSwap, {
      sender: PROPOSER,
      poolId: "pool-7",
      tokenInDenom: "uzrn",
      tokenInAmount: "100000",
      minTokenOut: "180415",
    });
    assertLiquidityError(
      () =>
        createExactInSwapPlan({
          sender: PROPOSER,
          poolId: "pool-7",
          tokenInDenom: "uzrn",
          tokenInAmount: "100000",
          minimumTokenOut: "0",
          timeoutHeight,
        }),
      "INVALID_AMOUNT",
    );
  });

  it("wraps a full admission Params update in standard Cosmos SDK governance", () => {
    const proposal = createLiquidityAdmissionProposal({
      authority: AUTHORITY,
      proposer: PROPOSER,
      currentParams: liquidityParams(),
      allowedPoolDenoms: ["uatom"],
      poolCreators: [PROPOSER],
      initialDeposit: [{ denom: "uzrn", amount: "5000000" }],
      title: "Admit the ZRN / ATOM pool",
      summary: "Admit one counter-denom and one creator without changing fees.",
    });

    assert.equal(proposal.typeUrl, MSG_SUBMIT_PROPOSAL_TYPE_URL);
    assert.equal(
      proposal.value.messages[0]?.typeUrl,
      MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
    );
    const update = MsgUpdateParams.decode(proposal.value.messages[0]!.value);
    assert.equal(update.authority, AUTHORITY);
    assert.equal(update.params?.defaultSwapFeeBps, 3_000n);
    assert.equal(update.params?.maxPools, 16n);
    assert.deepEqual(update.params?.allowedPoolDenoms, ["uatom"]);
    assert.deepEqual(update.params?.poolCreators, [PROPOSER]);
    assert.deepEqual(proposal.value.initialDeposit, [
      { denom: "uzrn", amount: "5000000" },
    ]);
    assert.equal(proposal.value.proposer, PROPOSER);
  });

  it("builds the separately creator-signed pool message with governed fee", () => {
    const message = createPoolMessage({
      creator: PROPOSER,
      denomA: "uzrn",
      denomB: "uatom",
      amountA: "1000000",
      amountB: "2000000",
      params: liquidityParams({
        allowedPoolDenoms: ["uatom"],
        poolCreators: [PROPOSER],
        poolCreationEnabled: true,
        poolCreationPolicy:
          "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY",
      }),
    });
    assert.equal(message.typeUrl, MSG_CREATE_POOL_TYPE_URL);
    assert.deepEqual(message.value as MsgCreatePool, {
      creator: PROPOSER,
      denomA: "uzrn",
      denomB: "uatom",
      amountA: "1000000",
      amountB: "2000000",
      swapFeeBps: 0n,
    });
  });

  it("rejects unadmitted creators/denoms and noncanonical pool amounts", () => {
    assertLiquidityError(
      () =>
        createPoolMessage({
          creator: toBech32("zrn", new Uint8Array(64), 200),
          denomA: "uzrn",
          denomB: "uatom",
          amountA: "1000000",
          amountB: "1",
          params: liquidityParams(),
        }),
      "INVALID_ADDRESS",
    );
    assertLiquidityError(
      () =>
        createPoolMessage({
          creator: PROPOSER,
          denomA: "uzrn",
          denomB: "uatom",
          amountA: "1000000",
          amountB: "1",
          params: liquidityParams(),
        }),
      "INVALID_ADDRESS",
    );
    assertLiquidityError(
      () =>
        createPoolMessage({
          creator: PROPOSER,
          denomA: "uzrn",
          denomB: "uatom",
          amountA: "01",
          amountB: "1",
          params: liquidityParams({
            allowedPoolDenoms: ["uatom"],
            poolCreators: [PROPOSER],
            poolCreationEnabled: true,
            poolCreationPolicy:
              "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY",
          }),
        }),
      "INVALID_AMOUNT",
    );
  });
});
