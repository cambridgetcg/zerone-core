import {
  MsgUpdateParams
} from "./chunk-CXBAXZI7.js";

// src/liquidity.ts
import { fromBech32 } from "@cosmjs/encoding";
var LIQUIDITY_FEE_SCALE = 1000000n;
var ZERONE_MAX_SWAP_FEE = 100000n;
var COSMOS_UINT64_MAX = 18446744073709551615n;
var COSMOS_AMOUNT_MAX = (1n << 256n) - 1n;
var ZERONE_MAX_POOL_RECORDS = 10000n;
var LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE = "fee_collector";
var MSG_CREATE_POOL_TYPE_URL = "/zerone.liquiditypool.v1.MsgCreatePool";
var MSG_SWAP_TYPE_URL = "/zerone.liquiditypool.v1.MsgSwap";
var MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL = "/zerone.liquiditypool.v1.MsgUpdateParams";
var MSG_SUBMIT_PROPOSAL_TYPE_URL = "/cosmos.gov.v1.MsgSubmitProposal";
var LIQUIDITY_POOL_STATUS = Object.freeze({
  UNSPECIFIED: 0,
  ACTIVE: 1,
  SWAPS_PAUSED: 2,
  EXIT_ONLY: 3,
  CLOSED: 4
});
var LiquidityClientError = class extends Error {
  code;
  field;
  constructor(code, field, message) {
    super(`${field}: ${message}`);
    this.name = "LiquidityClientError";
    this.code = code;
    this.field = field;
  }
};
var CANONICAL_UINT_PATTERN = /^(?:0|[1-9][0-9]*)$/;
var CANONICAL_POSITIVE_PATTERN = /^[1-9][0-9]*$/;
var DENOM_PATTERN = /^[A-Za-z][A-Za-z0-9/:._-]{2,127}$/;
var POOL_ID_PATTERN = /^pool-[1-9][0-9]{0,19}$/;
var BASE64_PATTERN = /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;
function fail(code, field, message) {
  throw new LiquidityClientError(code, field, message);
}
function parseCanonicalAmount(value, field, positive) {
  const pattern = positive ? CANONICAL_POSITIVE_PATTERN : CANONICAL_UINT_PATTERN;
  if (typeof value !== "string" || !pattern.test(value)) {
    return fail(
      "INVALID_AMOUNT",
      field,
      positive ? "must be a positive canonical decimal integer string" : "must be a canonical non-negative decimal integer string"
    );
  }
  if (value.length > 78) {
    return fail("INVALID_AMOUNT", field, "exceeds the Cosmos 256-bit amount limit");
  }
  const parsed = BigInt(value);
  if (parsed > COSMOS_AMOUNT_MAX) {
    return fail("INVALID_AMOUNT", field, "exceeds the Cosmos 256-bit amount limit");
  }
  return parsed;
}
function parseCanonicalPositiveAmount(value, field = "amount") {
  return parseCanonicalAmount(value, field, true);
}
function parseUint64(value, field, defaultValue) {
  if (value === void 0 && defaultValue !== void 0) return defaultValue;
  let parsed;
  if (typeof value === "bigint") {
    parsed = value;
  } else if (typeof value === "number" && Number.isSafeInteger(value) && value >= 0) {
    parsed = BigInt(value);
  } else if (typeof value === "string" && CANONICAL_UINT_PATTERN.test(value)) {
    parsed = BigInt(value);
  } else {
    return fail(
      "INVALID_UINT64",
      field,
      "must be a canonical decimal uint64"
    );
  }
  if (parsed < 0n || parsed > COSMOS_UINT64_MAX) {
    return fail("INVALID_UINT64", field, "exceeds uint64");
  }
  return parsed;
}
function validateDenom(value, field) {
  if (typeof value !== "string" || !DENOM_PATTERN.test(value)) {
    return fail(
      "INVALID_DENOM",
      field,
      "must be a canonical 3-128 character Cosmos denomination"
    );
  }
  return value;
}
function validatePoolId(value, field = "poolId") {
  if (typeof value !== "string" || !POOL_ID_PATTERN.test(value)) {
    return fail("INVALID_POOL", field, "must be a bounded canonical pool ID");
  }
  if (BigInt(value.slice("pool-".length)) > ZERONE_MAX_POOL_RECORDS) {
    return fail(
      "INVALID_POOL",
      field,
      `numeric suffix exceeds the ${ZERONE_MAX_POOL_RECORDS} record lifetime cap`
    );
  }
  return value;
}
function validateZeroneAddress(value, field) {
  if (typeof value !== "string" || value !== value.toLowerCase()) {
    return fail(
      "INVALID_ADDRESS",
      field,
      "must be a lowercase Zerone Bech32 account address"
    );
  }
  try {
    const decoded = fromBech32(value);
    if (decoded.prefix !== "zrn" || decoded.data.length !== 20) {
      return fail(
        "INVALID_ADDRESS",
        field,
        "must use the zrn prefix and contain a 20-byte account"
      );
    }
  } catch {
    return fail(
      "INVALID_ADDRESS",
      field,
      "must be a valid Zerone Bech32 account address"
    );
  }
  return value;
}
function parseMillionths(value, field, maximumExclusive, code) {
  if (typeof value !== "bigint" || value < 0n || value >= maximumExclusive) {
    return fail(
      code,
      field,
      `must be an integer from 0 through ${maximumExclusive - 1n} on the 1,000,000 scale`
    );
  }
  return value;
}
function record(value, field) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return fail("INVALID_RESPONSE", field, "must be an object");
  }
  return value;
}
function wireField(object, camelCase, snakeCase, field) {
  const camel = object[camelCase];
  const snake = object[snakeCase];
  if (camel !== void 0 && snake !== void 0 && camel !== snake) {
    return fail(
      "INVALID_RESPONSE",
      field,
      `contains conflicting ${camelCase} and ${snakeCase} values`
    );
  }
  return camel ?? snake;
}
function parsePoolStatus(value, field) {
  if (value === void 0) return "PRE_V4";
  const normalized = typeof value === "string" ? value.toUpperCase() : value;
  switch (normalized) {
    case 0:
    case "0":
    case "UNSPECIFIED":
    case "POOL_STATUS_UNSPECIFIED":
      return "UNSPECIFIED";
    case 1:
    case "1":
    case "ACTIVE":
    case "POOL_STATUS_ACTIVE":
      return "ACTIVE";
    case 2:
    case "2":
    case "SWAPS_PAUSED":
    case "POOL_STATUS_SWAPS_PAUSED":
      return "SWAPS_PAUSED";
    case 3:
    case "3":
    case "EXIT_ONLY":
    case "POOL_STATUS_EXIT_ONLY":
      return "EXIT_ONLY";
    case 4:
    case "4":
    case "CLOSED":
    case "POOL_STATUS_CLOSED":
      return "CLOSED";
    default:
      return fail("INVALID_RESPONSE", field, "contains an unknown pool status");
  }
}
function parsePool(value, field) {
  const object = record(value, field);
  const poolId = validatePoolId(
    wireField(object, "poolId", "pool_id", `${field}.poolId`),
    `${field}.poolId`
  );
  const denomA = validateDenom(
    wireField(object, "denomA", "denom_a", `${field}.denomA`),
    `${field}.denomA`
  );
  const denomB = validateDenom(
    wireField(object, "denomB", "denom_b", `${field}.denomB`),
    `${field}.denomB`
  );
  if (denomA === denomB || denomA !== "uzrn" && denomB !== "uzrn") {
    return fail(
      "INVALID_RESPONSE",
      field,
      "must contain two distinct denoms with one uzrn side"
    );
  }
  const reserveAValue = wireField(
    object,
    "reserveA",
    "reserve_a",
    `${field}.reserveA`
  );
  const reserveBValue = wireField(
    object,
    "reserveB",
    "reserve_b",
    `${field}.reserveB`
  );
  const supplyValue = wireField(
    object,
    "lpTokenSupply",
    "lp_token_supply",
    `${field}.lpTokenSupply`
  );
  const reserveA = parseCanonicalAmount(
    reserveAValue,
    `${field}.reserveA`,
    false
  );
  const reserveB = parseCanonicalAmount(
    reserveBValue,
    `${field}.reserveB`,
    false
  );
  const lpTokenSupply = parseCanonicalAmount(
    supplyValue,
    `${field}.lpTokenSupply`,
    false
  );
  const swapFeeMillionths = parseUint64(
    wireField(
      object,
      "swapFeeBps",
      "swap_fee_bps",
      `${field}.swapFeeBps`
    ),
    `${field}.swapFeeBps`,
    0n
  );
  if (swapFeeMillionths > ZERONE_MAX_SWAP_FEE) {
    return fail(
      "INVALID_RESPONSE",
      `${field}.swapFeeBps`,
      `exceeds Zerone's ${ZERONE_MAX_SWAP_FEE} maximum`
    );
  }
  const status = parsePoolStatus(object.status, `${field}.status`);
  if (status === "UNSPECIFIED") {
    return fail(
      "INVALID_RESPONSE",
      `${field}.status`,
      "explicit UNSPECIFIED is not a valid v4 lifecycle state"
    );
  }
  const closedHeight = parseUint64(
    wireField(
      object,
      "closedAtBlock",
      "closed_at_block",
      `${field}.closedAtBlock`
    ),
    `${field}.closedAtBlock`,
    0n
  );
  const allZero = reserveA === 0n && reserveB === 0n && lpTokenSupply === 0n;
  const allPositive = reserveA > 0n && reserveB > 0n && lpTokenSupply > 0n;
  if (status === "CLOSED") {
    if (!allZero || closedHeight === 0n) {
      return fail(
        "INVALID_RESPONSE",
        field,
        "a CLOSED tombstone must have zero reserves/supply and a closing height"
      );
    }
  } else if (status === "ACTIVE" || status === "SWAPS_PAUSED" || status === "EXIT_ONLY" || status === "PRE_V4") {
    if (!allPositive || closedHeight !== 0n) {
      return fail(
        "INVALID_RESPONSE",
        field,
        "an open pool must have positive reserves/supply and no closing height"
      );
    }
  }
  const locked = object.locked ?? false;
  if (typeof locked !== "boolean") {
    return fail("INVALID_RESPONSE", `${field}.locked`, "must be a boolean");
  }
  const lpDenom = validateDenom(
    wireField(object, "lpDenom", "lp_denom", `${field}.lpDenom`),
    `${field}.lpDenom`
  );
  if (lpDenom !== `lp/${poolId}`) {
    return fail(
      "INVALID_RESPONSE",
      `${field}.lpDenom`,
      `must be lp/${poolId}`
    );
  }
  const createdAtBlock = parseUint64(
    wireField(
      object,
      "createdAtBlock",
      "created_at_block",
      `${field}.createdAtBlock`
    ),
    `${field}.createdAtBlock`,
    0n
  );
  if (status === "CLOSED" && closedHeight < createdAtBlock) {
    return fail(
      "INVALID_RESPONSE",
      `${field}.closedAtBlock`,
      "cannot precede the creation height"
    );
  }
  return Object.freeze({
    poolId,
    denomA,
    denomB,
    reserveA: String(reserveAValue),
    reserveB: String(reserveBValue),
    swapFeeMillionths,
    lpTokenSupply: String(supplyValue),
    lpDenom,
    creator: validateZeroneAddress(object.creator, `${field}.creator`),
    createdAtBlock,
    locked,
    status,
    closedAtBlock: closedHeight === 0n ? null : closedHeight
  });
}
function parseCounterDenomList(value, field) {
  if (!Array.isArray(value) || value.length > 32) {
    return fail(
      "INVALID_RESPONSE",
      field,
      "must contain at most 32 counter-denoms"
    );
  }
  const seen = /* @__PURE__ */ new Set();
  const denoms = value.map((entry, index) => {
    const denom = validateDenom(entry, `${field}[${index}]`);
    if (denom === "uzrn" || seen.has(denom)) {
      return fail(
        "INVALID_RESPONSE",
        `${field}[${index}]`,
        "must be a unique non-uzrn denom"
      );
    }
    seen.add(denom);
    return denom;
  });
  return Object.freeze(denoms);
}
function parsePoolCreatorList(value, field) {
  if (!Array.isArray(value) || value.length > 32) {
    return fail(
      "INVALID_RESPONSE",
      field,
      "must contain at most 32 creator addresses"
    );
  }
  const seen = /* @__PURE__ */ new Set();
  const creators = value.map((entry, index) => {
    const creator = validateZeroneAddress(entry, `${field}[${index}]`);
    if (seen.has(creator)) {
      return fail(
        "INVALID_RESPONSE",
        `${field}[${index}]`,
        "must be unique"
      );
    }
    seen.add(creator);
    return creator;
  });
  return Object.freeze(creators);
}
function parseParams(value) {
  const response = record(value, "$");
  const params = record(response.params, "$.params");
  const defaultSwapFeeMillionths = parseUint64(
    wireField(
      params,
      "defaultSwapFeeBps",
      "default_swap_fee_bps",
      "$.params.defaultSwapFeeBps"
    ),
    "$.params.defaultSwapFeeBps",
    0n
  );
  if (defaultSwapFeeMillionths > ZERONE_MAX_SWAP_FEE) {
    return fail(
      "INVALID_RESPONSE",
      "$.params.defaultSwapFeeBps",
      `exceeds Zerone's ${ZERONE_MAX_SWAP_FEE} maximum`
    );
  }
  const protocolFeeMillionths = parseUint64(
    wireField(
      params,
      "protocolFeeBps",
      "protocol_fee_bps",
      "$.params.protocolFeeBps"
    ),
    "$.params.protocolFeeBps",
    0n
  );
  if (protocolFeeMillionths > LIQUIDITY_FEE_SCALE) {
    return fail(
      "INVALID_RESPONSE",
      "$.params.protocolFeeBps",
      "must not exceed the 1,000,000 fee scale"
    );
  }
  const billingQuoteDenoms = parseCounterDenomList(
    wireField(
      params,
      "billingQuoteDenoms",
      "billing_quote_denoms",
      "$.params.billingQuoteDenoms"
    ) ?? [],
    "$.params.billingQuoteDenoms"
  );
  const allowedPoolDenoms = parseCounterDenomList(
    wireField(
      params,
      "allowedPoolDenoms",
      "allowed_pool_denoms",
      "$.params.allowedPoolDenoms"
    ) ?? [],
    "$.params.allowedPoolDenoms"
  );
  const poolCreators = parsePoolCreatorList(
    wireField(
      params,
      "poolCreators",
      "pool_creators",
      "$.params.poolCreators"
    ) ?? [],
    "$.params.poolCreators"
  );
  const billingOracleEnabled = billingQuoteDenoms.length > 0;
  const maxPools = parseUint64(
    wireField(params, "maxPools", "max_pools", "$.params.maxPools"),
    "$.params.maxPools",
    0n
  );
  if (maxPools > 64n) {
    return fail(
      "INVALID_RESPONSE",
      "$.params.maxPools",
      "must be between 0 and 64 (legacy zero means disabled)"
    );
  }
  const poolCreationEnabled = maxPools > 0n && allowedPoolDenoms.length > 0 && poolCreators.length > 0;
  const twapWindowBlocks = parseUint64(
    wireField(
      params,
      "twapWindowBlocks",
      "twap_window_blocks",
      "$.params.twapWindowBlocks"
    ),
    "$.params.twapWindowBlocks",
    0n
  );
  if (twapWindowBlocks < 1n || twapWindowBlocks > 10000n) {
    return fail(
      "INVALID_RESPONSE",
      "$.params.twapWindowBlocks",
      "must be between 1 and 10000"
    );
  }
  return Object.freeze({
    defaultSwapFeeMillionths,
    maxPools,
    minInitialLiquidity: String(
      parseCanonicalAmount(
        wireField(
          params,
          "minInitialLiquidity",
          "min_initial_liquidity",
          "$.params.minInitialLiquidity"
        ),
        "$.params.minInitialLiquidity",
        true
      )
    ),
    twapWindowBlocks,
    protocolFeeMillionths,
    minReserve: String(
      parseCanonicalAmount(
        wireField(
          params,
          "minReserve",
          "min_reserve",
          "$.params.minReserve"
        ),
        "$.params.minReserve",
        false
      )
    ),
    billingQuoteDenoms,
    billingOracleEnabled,
    billingOraclePolicy: billingOracleEnabled ? "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED" : "DISABLED_FAIL_CLOSED",
    allowedPoolDenoms,
    poolCreators,
    poolCreationEnabled,
    poolCreationPolicy: poolCreationEnabled ? "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY" : "DISABLED_FAIL_CLOSED"
  });
}
function parsePagination(value) {
  if (value === void 0 || value === null) {
    return { nextKey: null, total: null };
  }
  const pagination = record(value, "$.pagination");
  const rawNextKey = wireField(
    pagination,
    "nextKey",
    "next_key",
    "$.pagination.nextKey"
  );
  let nextKey = null;
  if (rawNextKey !== void 0 && rawNextKey !== "") {
    if (typeof rawNextKey !== "string" || rawNextKey.length > 4096 || !BASE64_PATTERN.test(rawNextKey)) {
      return fail(
        "INVALID_RESPONSE",
        "$.pagination.nextKey",
        "must be a bounded base64 cursor"
      );
    }
    nextKey = rawNextKey;
  }
  const rawTotal = pagination.total;
  const total = rawTotal === void 0 ? null : parseUint64(rawTotal, "$.pagination.total");
  if (total !== null && total > ZERONE_MAX_POOL_RECORDS) {
    return fail(
      "INVALID_RESPONSE",
      "$.pagination.total",
      `exceeds the ${ZERONE_MAX_POOL_RECORDS} record lifetime cap`
    );
  }
  return {
    nextKey,
    total
  };
}
function minimumOutputForSlippage(tokenOutAmount, slippageMillionths) {
  const output = parseCanonicalPositiveAmount(
    tokenOutAmount,
    "tokenOutAmount"
  );
  const slippage = parseMillionths(
    slippageMillionths,
    "slippageMillionths",
    LIQUIDITY_FEE_SCALE,
    "INVALID_SLIPPAGE"
  );
  const derived = output * (LIQUIDITY_FEE_SCALE - slippage) / LIQUIDITY_FEE_SCALE;
  return (derived > 0n ? derived : 1n).toString();
}
function discloseLiquiditySwapFee(request) {
  const tokenInDenom = validateDenom(request.tokenInDenom, "tokenInDenom");
  const totalFee = parseCanonicalAmount(request.feeAmount, "feeAmount", false);
  const protocolFeeMillionths = parseMillionths(
    request.protocolFeeMillionths,
    "protocolFeeMillionths",
    LIQUIDITY_FEE_SCALE + 1n,
    "INVALID_FEE"
  );
  const policy = protocolFeeMillionths === 0n ? "NO_PROTOCOL_SKIM" : "LEGACY_ZRN_INPUT_PROTOCOL_SKIM";
  const protocolFee = tokenInDenom === "uzrn" && protocolFeeMillionths > 0n ? totalFee * protocolFeeMillionths / LIQUIDITY_FEE_SCALE : 0n;
  return Object.freeze({
    policy,
    tokenInDenom,
    totalFeeAmount: totalFee.toString(),
    poolRetainedFeeAmount: (totalFee - protocolFee).toString(),
    protocolFeeAmount: protocolFee.toString(),
    protocolFeeMillionths,
    protocolFeeDestinationModule: protocolFee > 0n ? LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE : null
  });
}
function quoteConstantProductExactIn(request) {
  const reserveIn = parseCanonicalPositiveAmount(
    request.reserveIn,
    "reserveIn"
  );
  const reserveOut = parseCanonicalPositiveAmount(
    request.reserveOut,
    "reserveOut"
  );
  const tokenIn = parseCanonicalPositiveAmount(
    request.tokenInAmount,
    "tokenInAmount"
  );
  const swapFee = parseMillionths(
    request.swapFeeMillionths,
    "swapFeeMillionths",
    ZERONE_MAX_SWAP_FEE + 1n,
    "INVALID_FEE"
  );
  const slippage = parseMillionths(
    request.slippageMillionths,
    "slippageMillionths",
    LIQUIDITY_FEE_SCALE,
    "INVALID_SLIPPAGE"
  );
  const feeAmount = tokenIn * swapFee / LIQUIDITY_FEE_SCALE;
  const effectiveTokenIn = tokenIn - feeAmount;
  const weightedTokenIn = tokenIn * (LIQUIDITY_FEE_SCALE - swapFee);
  const tokenOut = reserveOut * weightedTokenIn / (reserveIn * LIQUIDITY_FEE_SCALE + weightedTokenIn);
  if (tokenOut <= 0n) {
    return fail(
      "QUOTE_TOO_SMALL",
      "tokenOutAmount",
      "rounds to zero base units"
    );
  }
  const idealNumerator = tokenIn * (LIQUIDITY_FEE_SCALE - swapFee) * reserveOut;
  const idealDenominator = reserveIn * LIQUIDITY_FEE_SCALE;
  const actualNumerator = tokenOut * idealDenominator;
  const priceImpact = idealNumerator > actualNumerator && idealNumerator > 0n ? (idealNumerator - actualNumerator) * LIQUIDITY_FEE_SCALE / idealNumerator : 0n;
  const output = tokenOut.toString();
  return Object.freeze({
    reserveIn: request.reserveIn,
    reserveOut: request.reserveOut,
    tokenInAmount: request.tokenInAmount,
    effectiveTokenIn: effectiveTokenIn.toString(),
    weightedTokenIn: weightedTokenIn.toString(),
    feeAmount: feeAmount.toString(),
    tokenOutAmount: output,
    minimumTokenOut: minimumOutputForSlippage(output, slippage),
    swapFeeMillionths: swapFee,
    slippageMillionths: slippage,
    priceImpactMillionths: priceImpact
  });
}
function validateRestOptions(options) {
  let baseUrl;
  try {
    baseUrl = new URL(options.baseUrl);
  } catch {
    return fail("INVALID_RESPONSE", "baseUrl", "must be an absolute URL");
  }
  if (baseUrl.protocol !== "http:" && baseUrl.protocol !== "https:" || baseUrl.username !== "" || baseUrl.password !== "" || baseUrl.search !== "" || baseUrl.hash !== "") {
    return fail(
      "INVALID_RESPONSE",
      "baseUrl",
      "must be a credential-free HTTP(S) URL without query or fragment"
    );
  }
  if (!baseUrl.pathname.endsWith("/")) baseUrl.pathname += "/";
  const fetchImplementation = options.fetch ?? globalThis.fetch;
  if (typeof fetchImplementation !== "function") {
    return fail(
      "INVALID_RESPONSE",
      "fetch",
      "must be injected when global fetch is unavailable"
    );
  }
  const timeoutMs = options.timeoutMs ?? 8e3;
  const maximumResponseBytes = options.maximumResponseBytes ?? 1048576;
  if (!Number.isInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 6e4) {
    return fail("INVALID_RESPONSE", "timeoutMs", "must be between 1 and 60000");
  }
  if (!Number.isInteger(maximumResponseBytes) || maximumResponseBytes < 1 || maximumResponseBytes > 8388608) {
    return fail(
      "INVALID_RESPONSE",
      "maximumResponseBytes",
      "must be between 1 and 8388608"
    );
  }
  return {
    baseUrl,
    fetch: fetchImplementation.bind(globalThis),
    timeoutMs,
    maximumResponseBytes
  };
}
function restObservedHeight(headers, field) {
  const candidates = [
    headers.get("x-cosmos-block-height"),
    headers.get("grpc-metadata-x-cosmos-block-height")
  ].filter((value) => value !== null);
  if (candidates.length === 0) return null;
  let observedHeight = null;
  for (const value of candidates) {
    if (!CANONICAL_POSITIVE_PATTERN.test(value)) {
      return fail(
        "INVALID_RESPONSE",
        field,
        "must be a canonical positive decimal uint64"
      );
    }
    const parsed = BigInt(value);
    if (parsed > COSMOS_UINT64_MAX) {
      return fail("INVALID_RESPONSE", field, "exceeds uint64");
    }
    if (observedHeight !== null && parsed !== observedHeight) {
      return fail(
        "INVALID_RESPONSE",
        field,
        "contains conflicting Cosmos block heights"
      );
    }
    observedHeight = parsed;
  }
  return observedHeight;
}
var ZeroneLiquidityRestClient = class {
  id = "zerone/liquiditypool/v1";
  #baseUrl;
  #fetch;
  #timeoutMs;
  #maximumResponseBytes;
  constructor(options) {
    const validated = validateRestOptions(options);
    this.#baseUrl = validated.baseUrl;
    this.#fetch = validated.fetch;
    this.#timeoutMs = validated.timeoutMs;
    this.#maximumResponseBytes = validated.maximumResponseBytes;
  }
  async #jsonObservation(path, search) {
    const url = new URL(path.replace(/^\/+/, ""), this.#baseUrl);
    if (search !== void 0) url.search = search.toString();
    const response = await this.#fetch(url, {
      method: "GET",
      headers: { Accept: "application/json" },
      redirect: "error",
      signal: AbortSignal.timeout(this.#timeoutMs)
    });
    if (!response.ok) {
      return fail(
        "HTTP_ERROR",
        url.pathname,
        `REST query returned HTTP ${response.status}`
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null && (!CANONICAL_UINT_PATTERN.test(declaredLength) || BigInt(declaredLength) > BigInt(this.#maximumResponseBytes))) {
      return fail(
        "RESPONSE_TOO_LARGE",
        url.pathname,
        "REST response exceeded its configured byte limit"
      );
    }
    const text = await response.text();
    if (new TextEncoder().encode(text).byteLength > this.#maximumResponseBytes) {
      return fail(
        "RESPONSE_TOO_LARGE",
        url.pathname,
        "REST response exceeded its configured byte limit"
      );
    }
    let value;
    try {
      value = JSON.parse(text);
    } catch {
      return fail(
        "INVALID_RESPONSE",
        url.pathname,
        "REST response was not valid JSON"
      );
    }
    return Object.freeze({
      value,
      observedHeight: restObservedHeight(
        response.headers,
        `${url.pathname} response height`
      )
    });
  }
  async #json(path, search) {
    return (await this.#jsonObservation(path, search)).value;
  }
  async pool(poolId) {
    const id = validatePoolId(poolId);
    const response = record(
      await this.#json(
        `zerone/liquiditypool/v1/pools/${encodeURIComponent(id)}`
      ),
      "$"
    );
    return parsePool(response.pool, "$.pool");
  }
  async pools(request = {}) {
    const search = new URLSearchParams();
    if (request.limit !== void 0) {
      if (!Number.isInteger(request.limit) || request.limit < 1 || request.limit > 100) {
        return fail(
          "INVALID_RESPONSE",
          "pagination.limit",
          "must be an integer from 1 through 100"
        );
      }
      search.set("pagination.limit", String(request.limit));
    }
    if (request.key !== void 0) {
      if (request.limit === void 0 || request.key.length === 0 || request.key.length > 4096 || !BASE64_PATTERN.test(request.key)) {
        return fail(
          "INVALID_RESPONSE",
          "pagination.key",
          "requires a limit and a bounded base64 cursor"
        );
      }
      search.set("pagination.key", request.key);
    }
    const response = record(
      await this.#json(
        "zerone/liquiditypool/v1/pools",
        search.size === 0 ? void 0 : search
      ),
      "$"
    );
    if (!Array.isArray(response.pools) || response.pools.length > 100) {
      return fail(
        "INVALID_RESPONSE",
        "$.pools",
        "must be an array with at most 100 pools"
      );
    }
    const pagination = parsePagination(response.pagination);
    return Object.freeze({
      pools: Object.freeze(
        response.pools.map(
          (pool, index) => parsePool(pool, `$.pools[${index}]`)
        )
      ),
      nextKey: pagination.nextKey,
      total: pagination.total
    });
  }
  async params() {
    return parseParams(await this.#json("zerone/liquiditypool/v1/params"));
  }
  async twap(poolId, baseDenom, window = 0n) {
    const id = validatePoolId(poolId);
    const denom = validateDenom(baseDenom, "baseDenom");
    const requestedWindow = parseUint64(window, "window");
    const search = new URLSearchParams({
      baseDenom: denom,
      window: requestedWindow.toString()
    });
    const response = record(
      await this.#json(
        `zerone/liquiditypool/v1/twap/${encodeURIComponent(id)}`,
        search
      ),
      "$"
    );
    const rawPrice = response.twap;
    parseCanonicalAmount(rawPrice, "$.twap", false);
    return Object.freeze({
      price: String(rawPrice),
      windowUsed: parseUint64(
        wireField(
          response,
          "windowUsed",
          "window_used",
          "$.windowUsed"
        ),
        "$.windowUsed",
        0n
      )
    });
  }
  async simulateSwap(poolId, tokenInDenom, tokenInAmount) {
    const id = validatePoolId(poolId);
    const denom = validateDenom(tokenInDenom, "tokenInDenom");
    parseCanonicalPositiveAmount(tokenInAmount, "tokenInAmount");
    const search = new URLSearchParams({
      tokenInDenom: denom,
      tokenInAmount
    });
    const path = `zerone/liquiditypool/v1/simulate/${encodeURIComponent(id)}`;
    const observation = await this.#jsonObservation(path, search);
    if (observation.observedHeight === null) {
      return fail(
        "INVALID_RESPONSE",
        `${path} response height`,
        "is required to prepare an expiring transaction"
      );
    }
    const response = record(
      observation.value,
      "$"
    );
    const result = record(response.result, "$.result");
    const tokenOutAmount = result.tokenOutAmount ?? result.token_out_amount;
    const feeAmount = result.feeAmount ?? result.fee_amount;
    parseCanonicalPositiveAmount(
      String(tokenOutAmount),
      "$.result.tokenOutAmount"
    );
    parseCanonicalAmount(feeAmount, "$.result.feeAmount", false);
    const priceImpactMillionths = parseUint64(
      wireField(
        result,
        "priceImpactBps",
        "price_impact_bps",
        "$.result.priceImpactBps"
      ),
      "$.result.priceImpactBps",
      0n
    );
    if (priceImpactMillionths > LIQUIDITY_FEE_SCALE) {
      return fail(
        "INVALID_RESPONSE",
        "$.result.priceImpactBps",
        "must not exceed the 1,000,000 scale"
      );
    }
    return Object.freeze({
      tokenOutDenom: validateDenom(
        result.tokenOutDenom ?? result.token_out_denom,
        "$.result.tokenOutDenom"
      ),
      tokenOutAmount: String(tokenOutAmount),
      feeAmount: String(feeAmount),
      priceImpactMillionths,
      observedHeight: observation.observedHeight
    });
  }
  async prepareExactInSwap(request) {
    const sender = validateZeroneAddress(request.sender, "sender");
    const poolId = validatePoolId(request.poolId);
    const tokenInDenom = validateDenom(
      request.tokenInDenom,
      "tokenInDenom"
    );
    parseCanonicalPositiveAmount(request.tokenInAmount, "tokenInAmount");
    const expectedTokenOutDenom = validateDenom(
      request.expectedTokenOutDenom,
      "expectedTokenOutDenom"
    );
    if (expectedTokenOutDenom === tokenInDenom) {
      return fail(
        "INVALID_DENOM",
        "expectedTokenOutDenom",
        "must differ from tokenInDenom"
      );
    }
    const slippageMillionths = parseMillionths(
      request.slippageMillionths,
      "slippageMillionths",
      LIQUIDITY_FEE_SCALE,
      "INVALID_SLIPPAGE"
    );
    const lifetimeBlocks = parseUint64(
      request.lifetimeBlocks,
      "lifetimeBlocks"
    );
    if (lifetimeBlocks === 0n) {
      return fail(
        "INVALID_UINT64",
        "lifetimeBlocks",
        "must be greater than zero"
      );
    }
    const [simulation, params] = await Promise.all([
      this.simulateSwap(poolId, tokenInDenom, request.tokenInAmount),
      this.params()
    ]);
    if (simulation.tokenOutDenom !== expectedTokenOutDenom) {
      return fail(
        "INVALID_RESPONSE",
        "simulation.tokenOutDenom",
        `expected ${expectedTokenOutDenom}; received ${simulation.tokenOutDenom}`
      );
    }
    const minimumTokenOut = minimumOutputForSlippage(
      simulation.tokenOutAmount,
      slippageMillionths
    );
    const feeDisclosure = discloseLiquiditySwapFee({
      tokenInDenom,
      feeAmount: simulation.feeAmount,
      protocolFeeMillionths: params.protocolFeeMillionths
    });
    const plan = createExactInSwapPlan({
      sender,
      poolId,
      tokenInDenom,
      tokenInAmount: request.tokenInAmount,
      minimumTokenOut,
      timeoutHeight: timeoutHeightAfter(
        simulation.observedHeight,
        lifetimeBlocks
      )
    });
    return Object.freeze({
      simulation,
      feeDisclosure,
      minimumTokenOut,
      plan
    });
  }
  async quoteExactIn(request) {
    const [pool, params] = await Promise.all([
      this.pool(request.poolId),
      this.params()
    ]);
    if (pool.status !== "ACTIVE" || pool.locked) {
      return fail(
        "POOL_NOT_ACTIVE",
        pool.locked ? "pool.locked" : "pool.status",
        pool.locked ? "must be unlocked" : `must be ACTIVE; received ${pool.status}`
      );
    }
    const tokenInDenom = validateDenom(
      request.tokenInDenom,
      "tokenInDenom"
    );
    let reserveIn;
    let reserveOut;
    let tokenOutDenom;
    if (tokenInDenom === pool.denomA) {
      reserveIn = pool.reserveA;
      reserveOut = pool.reserveB;
      tokenOutDenom = pool.denomB;
    } else if (tokenInDenom === pool.denomB) {
      reserveIn = pool.reserveB;
      reserveOut = pool.reserveA;
      tokenOutDenom = pool.denomA;
    } else {
      return fail(
        "INVALID_DENOM",
        "tokenInDenom",
        "is not one of the pool denoms"
      );
    }
    const quote = quoteConstantProductExactIn({
      reserveIn,
      reserveOut,
      tokenInAmount: request.tokenInAmount,
      swapFeeMillionths: pool.swapFeeMillionths,
      slippageMillionths: request.slippageMillionths
    });
    if (BigInt(reserveOut) - BigInt(quote.tokenOutAmount) < BigInt(params.minReserve)) {
      return fail(
        "QUOTE_TOO_SMALL",
        "tokenOutAmount",
        `would leave less than the governed ${params.minReserve} minimum reserve`
      );
    }
    return Object.freeze({
      ...quote,
      adapterId: this.id,
      poolId: pool.poolId,
      tokenInDenom,
      tokenOutDenom,
      poolStatus: pool.status
    });
  }
};
function validateProposalText(value, field, maximumLength, allowEmpty = false) {
  if (typeof value !== "string" || !allowEmpty && value.trim().length === 0 || value.length > maximumLength) {
    return fail(
      "INVALID_RESPONSE",
      field,
      `must be ${allowEmpty ? "a" : "a non-empty"} string of at most ${maximumLength} characters`
    );
  }
  return value;
}
function validatePublicParams(value) {
  const params = record(value, "params");
  const normalized = parseParams({
    params: {
      defaultSwapFeeBps: params.defaultSwapFeeMillionths,
      maxPools: params.maxPools,
      minInitialLiquidity: params.minInitialLiquidity,
      twapWindowBlocks: params.twapWindowBlocks,
      protocolFeeBps: params.protocolFeeMillionths,
      minReserve: params.minReserve,
      billingQuoteDenoms: params.billingQuoteDenoms,
      allowedPoolDenoms: params.allowedPoolDenoms,
      poolCreators: params.poolCreators
    }
  });
  if (normalized.maxPools < 1n || normalized.maxPools > 64n) {
    return fail(
      "INVALID_UINT64",
      "params.maxPools",
      "must be between 1 and 64"
    );
  }
  if (normalized.twapWindowBlocks < 1n || normalized.twapWindowBlocks > 10000n) {
    return fail(
      "INVALID_UINT64",
      "params.twapWindowBlocks",
      "must be between 1 and 10000"
    );
  }
  return normalized;
}
function generatedParams(params, allowedPoolDenoms = params.allowedPoolDenoms, poolCreators = params.poolCreators) {
  return {
    defaultSwapFeeBps: params.defaultSwapFeeMillionths,
    maxPools: params.maxPools,
    minInitialLiquidity: params.minInitialLiquidity,
    twapWindowBlocks: params.twapWindowBlocks,
    protocolFeeBps: params.protocolFeeMillionths,
    minReserve: params.minReserve,
    billingQuoteDenoms: [...params.billingQuoteDenoms],
    allowedPoolDenoms: [...allowedPoolDenoms],
    poolCreators: [...poolCreators]
  };
}
function createLiquidityAdmissionUpdateMessage(request) {
  const authority = validateZeroneAddress(request.authority, "authority");
  const currentParams = validatePublicParams(request.currentParams);
  const allowedPoolDenoms = parseCounterDenomList(
    request.allowedPoolDenoms,
    "allowedPoolDenoms"
  );
  const poolCreators = parsePoolCreatorList(
    request.poolCreators,
    "poolCreators"
  );
  const message = {
    authority,
    params: generatedParams(
      currentParams,
      allowedPoolDenoms,
      poolCreators
    )
  };
  return Object.freeze({
    typeUrl: MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
    value: MsgUpdateParams.encode(message).finish()
  });
}
function createLiquidityAdmissionProposal(request) {
  const proposer = validateZeroneAddress(request.proposer, "proposer");
  const initialDeposit = (request.initialDeposit ?? []).map((coin, index) => {
    const denom = validateDenom(coin.denom, `initialDeposit[${index}].denom`);
    parseCanonicalPositiveAmount(
      coin.amount,
      `initialDeposit[${index}].amount`
    );
    return Object.freeze({ denom, amount: coin.amount });
  });
  const seenDepositDenoms = /* @__PURE__ */ new Set();
  initialDeposit.forEach((coin, index) => {
    if (seenDepositDenoms.has(coin.denom)) {
      fail(
        "INVALID_RESPONSE",
        `initialDeposit[${index}].denom`,
        "must be unique"
      );
    }
    seenDepositDenoms.add(coin.denom);
  });
  const value = Object.freeze({
    messages: Object.freeze([
      createLiquidityAdmissionUpdateMessage(request)
    ]),
    initialDeposit: Object.freeze(initialDeposit),
    proposer,
    metadata: validateProposalText(
      request.metadata ?? "",
      "metadata",
      1e4,
      true
    ),
    title: validateProposalText(request.title, "title", 140),
    summary: validateProposalText(request.summary, "summary", 1e4),
    expedited: request.expedited ?? false
  });
  if (typeof value.expedited !== "boolean") {
    return fail("INVALID_RESPONSE", "expedited", "must be a boolean");
  }
  return Object.freeze({
    typeUrl: MSG_SUBMIT_PROPOSAL_TYPE_URL,
    value
  });
}
function createPoolMessage(request) {
  const params = validatePublicParams(request.params);
  const creator = validateZeroneAddress(request.creator, "creator");
  if (!params.poolCreators.includes(creator)) {
    return fail(
      "INVALID_ADDRESS",
      "creator",
      "is not in the persistent governed pool creator allowlist"
    );
  }
  const denomA = validateDenom(request.denomA, "denomA");
  const denomB = validateDenom(request.denomB, "denomB");
  if (denomA === denomB || denomA !== "uzrn" && denomB !== "uzrn") {
    return fail(
      "INVALID_DENOM",
      "denoms",
      "must be distinct and include one uzrn side"
    );
  }
  const counterDenom = denomA === "uzrn" ? denomB : denomA;
  if (!params.allowedPoolDenoms.includes(counterDenom)) {
    return fail(
      "INVALID_DENOM",
      "denoms",
      "counter-denom has no pending governed one-shot creation grant"
    );
  }
  const amountA = parseCanonicalPositiveAmount(request.amountA, "amountA");
  const amountB = parseCanonicalPositiveAmount(request.amountB, "amountB");
  const zrnAmount = denomA === "uzrn" ? amountA : amountB;
  if (zrnAmount < BigInt(params.minInitialLiquidity)) {
    return fail(
      "INVALID_AMOUNT",
      denomA === "uzrn" ? "amountA" : "amountB",
      `must meet the governed ${params.minInitialLiquidity} uzrn minimum`
    );
  }
  const message = Object.freeze({
    creator,
    denomA,
    denomB,
    amountA: request.amountA,
    amountB: request.amountB,
    // v4 resolves the governed default and rejects every custom fee.
    swapFeeBps: 0n
  });
  return Object.freeze({
    typeUrl: MSG_CREATE_POOL_TYPE_URL,
    value: message
  });
}
function timeoutHeightAfter(currentHeight, lifetimeBlocks) {
  const current = parseUint64(currentHeight, "currentHeight");
  const lifetime = parseUint64(lifetimeBlocks, "lifetimeBlocks");
  if (lifetime === 0n) {
    return fail(
      "INVALID_UINT64",
      "lifetimeBlocks",
      "must be greater than zero"
    );
  }
  const timeoutHeight = current + lifetime;
  if (timeoutHeight > COSMOS_UINT64_MAX) {
    return fail("INVALID_UINT64", "timeoutHeight", "exceeds uint64");
  }
  return timeoutHeight;
}
function withTimeoutHeight(messages, timeoutHeight) {
  if (messages.length === 0) {
    return fail(
      "INVALID_RESPONSE",
      "messages",
      "must contain at least one message"
    );
  }
  const height = parseUint64(timeoutHeight, "timeoutHeight");
  if (height === 0n) {
    return fail(
      "INVALID_UINT64",
      "timeoutHeight",
      "must be greater than zero"
    );
  }
  return Object.freeze({
    messages: Object.freeze([...messages]),
    timeoutHeight: height
  });
}
function createExactInSwapPlan(request) {
  const message = {
    sender: validateZeroneAddress(request.sender, "sender"),
    poolId: validatePoolId(request.poolId),
    tokenInDenom: validateDenom(request.tokenInDenom, "tokenInDenom"),
    tokenInAmount: request.tokenInAmount,
    minTokenOut: request.minimumTokenOut
  };
  parseCanonicalPositiveAmount(message.tokenInAmount, "tokenInAmount");
  parseCanonicalPositiveAmount(message.minTokenOut, "minimumTokenOut");
  return withTimeoutHeight(
    [
      Object.freeze({
        typeUrl: MSG_SWAP_TYPE_URL,
        value: message
      })
    ],
    request.timeoutHeight
  );
}

export {
  LIQUIDITY_FEE_SCALE,
  ZERONE_MAX_SWAP_FEE,
  COSMOS_UINT64_MAX,
  COSMOS_AMOUNT_MAX,
  ZERONE_MAX_POOL_RECORDS,
  LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE,
  MSG_CREATE_POOL_TYPE_URL,
  MSG_SWAP_TYPE_URL,
  MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
  MSG_SUBMIT_PROPOSAL_TYPE_URL,
  LIQUIDITY_POOL_STATUS,
  LiquidityClientError,
  parseCanonicalPositiveAmount,
  minimumOutputForSlippage,
  discloseLiquiditySwapFee,
  quoteConstantProductExactIn,
  ZeroneLiquidityRestClient,
  createLiquidityAdmissionUpdateMessage,
  createLiquidityAdmissionProposal,
  createPoolMessage,
  timeoutHeightAfter,
  withTimeoutHeight,
  createExactInSwapPlan
};
