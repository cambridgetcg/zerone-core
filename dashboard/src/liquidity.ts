import {
  defineZeroneNetwork,
  zeroneAccountId,
} from "@zerone-chain/sdk/caip";

const ZERONE_MAINNET = defineZeroneNetwork("zerone-1");

export interface RawLiquidityPool {
  poolId?: string;
  pool_id?: string;
  denomA?: string;
  denom_a?: string;
  denomB?: string;
  denom_b?: string;
  reserveA?: string;
  reserve_a?: string;
  reserveB?: string;
  reserve_b?: string;
  swapFeeBps?: string | number;
  swap_fee_bps?: string | number;
  lpTokenSupply?: string;
  lp_token_supply?: string;
  lpDenom?: string;
  lp_denom?: string;
  creator?: string;
  createdAtBlock?: string | number;
  created_at_block?: string | number;
  locked?: boolean;
  status?: unknown;
  closedAtBlock?: string | number;
  closed_at_block?: string | number;
}

export interface RawLiquidityParamsResponse {
  params?: {
    defaultSwapFeeBps?: string;
    default_swap_fee_bps?: string;
    protocolFeeBps?: string;
    protocol_fee_bps?: string;
    minInitialLiquidity?: string;
    min_initial_liquidity?: string;
    twapWindowBlocks?: string;
    twap_window_blocks?: string;
    maxPools?: string;
    max_pools?: string;
    minReserve?: string;
    min_reserve?: string;
    billingQuoteDenoms?: unknown;
    billing_quote_denoms?: unknown;
    allowedPoolDenoms?: unknown;
    allowed_pool_denoms?: unknown;
    poolCreators?: unknown;
    pool_creators?: unknown;
  };
}

export type LiquidityPoolLifecycleStatus =
  | "PRE_V4"
  | "ACTIVE"
  | "SWAPS_PAUSED"
  | "EXIT_ONLY"
  | "CLOSED";

export interface LiquidityPool {
  id: string;
  denomA: string;
  denomB: string;
  reserveA: string;
  reserveB: string;
  swapFeeBps: number;
  lpSupply: string;
  lpDenom: string;
  creator: string;
  createdAtBlock: number;
  locked: boolean;
  status: LiquidityPoolLifecycleStatus;
  closedAtBlock: number | null;
}

export interface LiquidityParams {
  defaultSwapFeeBps: number;
  protocolFeeBps: number;
  protocolFeePolicy:
    | "LP_ONLY_NO_PROTOCOL_SKIM"
    | "LEGACY_PROTOCOL_SKIM_CONFIGURED";
  minInitialLiquidity: string;
  twapWindowBlocks: number;
  maxPools: number;
  minReserve: string;
  billingQuoteDenoms: readonly string[];
  billingOracleEnabled: boolean;
  billingOraclePolicy:
    | "DISABLED_FAIL_CLOSED"
    | "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED";
  allowedPoolDenoms: readonly string[];
  poolCreators: readonly string[];
  poolCreationEnabled: boolean;
  poolCreationPolicy:
    | "DISABLED_FAIL_CLOSED"
    | "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY";
}

const MAX_POOL_ID = 10_000n;
const MAX_SDK_AMOUNT = (1n << 256n) - 1n;
const COSMOS_DENOM = /^[A-Za-z][A-Za-z0-9/:._-]{2,127}$/;

function uint(value: unknown): number | null {
  if (typeof value !== "string" && typeof value !== "number") return null;
  if (!/^(?:0|[1-9]\d*)$/.test(String(value))) return null;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed >= 0 ? parsed : null;
}

function amount(value: unknown): string | null {
  if (
    typeof value !== "string" ||
    value.length > 78 ||
    !/^(?:0|[1-9]\d*)$/.test(value)
  ) {
    return null;
  }
  return BigInt(value) <= MAX_SDK_AMOUNT ? value : null;
}

function positiveAmount(value: unknown): string | null {
  const parsed = amount(value);
  return parsed !== null && parsed !== "0" ? parsed : null;
}

function poolId(value: unknown): string | null {
  if (typeof value !== "string") return null;
  const match = /^pool-([1-9]\d*)$/.exec(value);
  if (!match) return null;
  const suffix = match[1];
  if (
    suffix === undefined ||
    suffix.length > 20 ||
    BigInt(suffix) > MAX_POOL_ID
  ) {
    return null;
  }
  return value;
}

function poolDenom(value: unknown): string | null {
  return typeof value === "string" && COSMOS_DENOM.test(value) ? value : null;
}

function creatorAddress(value: unknown): string | null {
  if (typeof value !== "string") return null;
  try {
    zeroneAccountId(ZERONE_MAINNET, value);
    return value;
  } catch {
    return null;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function normalizePoolStatus(
  value: unknown,
): LiquidityPoolLifecycleStatus | null {
  if (value === undefined) return "PRE_V4";
  const normalized =
    typeof value === "string" ? value.toUpperCase() : value;
  switch (normalized) {
    case 0:
    case "0":
    case "UNSPECIFIED":
    case "POOL_STATUS_UNSPECIFIED":
      return null;
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
      return null;
  }
}

export function normalizeLiquidityPool(value: unknown): LiquidityPool | null {
  if (!isRecord(value)) return null;
  const pool = value as RawLiquidityPool;
  const id = poolId(pool.poolId ?? pool.pool_id);
  const denomA = poolDenom(pool.denomA ?? pool.denom_a);
  const denomB = poolDenom(pool.denomB ?? pool.denom_b);
  const reserveA = amount(pool.reserveA ?? pool.reserve_a);
  const reserveB = amount(pool.reserveB ?? pool.reserve_b);
  const fee = uint(pool.swapFeeBps ?? pool.swap_fee_bps);
  const lpSupply = amount(pool.lpTokenSupply ?? pool.lp_token_supply);
  const lpDenom = poolDenom(pool.lpDenom ?? pool.lp_denom);
  const creator = creatorAddress(pool.creator);
  const createdAtBlock = uint(pool.createdAtBlock ?? pool.created_at_block);
  const status = normalizePoolStatus(pool.status);
  const rawClosedAtBlock =
    pool.closedAtBlock !== undefined
      ? pool.closedAtBlock
      : pool.closed_at_block;
  const closedAtBlock = uint(
    rawClosedAtBlock === undefined ? 0 : rawClosedAtBlock,
  );
  const locked = pool.locked === undefined ? false : pool.locked;
  if (
    !id ||
    !denomA ||
    !denomB ||
    denomA === denomB ||
    (denomA !== "uzrn" && denomB !== "uzrn") ||
    reserveA === null ||
    reserveB === null ||
    fee === null ||
    fee > 100_000 ||
    lpSupply === null ||
    lpDenom !== `lp/${id}` ||
    creator === null ||
    createdAtBlock === null ||
    status === null ||
    closedAtBlock === null ||
    typeof locked !== "boolean"
  ) {
    return null;
  }
  const allZero = reserveA === "0" && reserveB === "0" && lpSupply === "0";
  const allPositive =
    reserveA !== "0" && reserveB !== "0" && lpSupply !== "0";
  if (
    (status === "CLOSED" &&
      (!allZero ||
        closedAtBlock === 0 ||
        closedAtBlock < createdAtBlock)) ||
    (status !== "CLOSED" && closedAtBlock !== 0) ||
    (status !== "CLOSED" && !allPositive)
  ) {
    return null;
  }
  return {
    id,
    denomA,
    denomB,
    reserveA,
    reserveB,
    swapFeeBps: fee,
    lpSupply,
    lpDenom,
    creator,
    createdAtBlock,
    locked,
    status,
    closedAtBlock: closedAtBlock === 0 ? null : closedAtBlock,
  };
}

function parseCounterDenomList(value: unknown): string[] | null {
  if (!Array.isArray(value) || value.length > 32) return null;
  const parsed = value.map((entry) =>
    poolDenom(entry) !== null && entry !== "uzrn"
      ? entry
      : null,
  );
  if (
    parsed.some((entry) => entry === null) ||
    new Set(parsed).size !== parsed.length
  ) {
    return null;
  }
  return parsed as string[];
}

function parsePoolCreatorList(value: unknown): string[] | null {
  if (!Array.isArray(value) || value.length > 32) return null;
  const parsed = value.map((entry) => {
    if (typeof entry !== "string") return null;
    try {
      zeroneAccountId(ZERONE_MAINNET, entry);
      return entry;
    } catch {
      return null;
    }
  });
  if (
    parsed.some((entry) => entry === null) ||
    new Set(parsed).size !== parsed.length
  ) {
    return null;
  }
  return parsed as string[];
}

export function normalizeLiquidityParams(
  value: unknown,
): LiquidityParams | null {
  if (!isRecord(value)) return null;
  const response = value as RawLiquidityParamsResponse;
  const params = response.params;
  if (!isRecord(params)) return null;
  const defaultSwapFeeBps = uint(
    params.defaultSwapFeeBps ?? params.default_swap_fee_bps,
  );
  const protocolFeeBps = uint(params.protocolFeeBps ?? params.protocol_fee_bps);
  const minInitialLiquidity = positiveAmount(
    params.minInitialLiquidity ?? params.min_initial_liquidity,
  );
  const twapWindowBlocks = uint(params.twapWindowBlocks ?? params.twap_window_blocks);
  const maxPools = uint(params.maxPools ?? params.max_pools);
  const minReserve = amount(params.minReserve ?? params.min_reserve);
  const billingQuoteDenoms = parseCounterDenomList(
    params.billingQuoteDenoms ?? params.billing_quote_denoms ?? [],
  );
  const allowedPoolDenoms = parseCounterDenomList(
    params.allowedPoolDenoms ?? params.allowed_pool_denoms ?? [],
  );
  const poolCreators = parsePoolCreatorList(
    params.poolCreators ?? params.pool_creators ?? [],
  );
  if (
    defaultSwapFeeBps === null ||
    protocolFeeBps === null ||
    minInitialLiquidity === null ||
    twapWindowBlocks === null ||
    maxPools === null ||
    minReserve === null ||
    billingQuoteDenoms === null ||
    allowedPoolDenoms === null ||
    poolCreators === null ||
    defaultSwapFeeBps > 100_000 ||
    protocolFeeBps > 1_000_000 ||
    maxPools > 64 ||
    twapWindowBlocks < 1 ||
    twapWindowBlocks > 10_000
  ) {
    return null;
  }
  const billingOracleEnabled = billingQuoteDenoms.length > 0;
  const poolCreationEnabled =
    allowedPoolDenoms.length > 0 && poolCreators.length > 0;
  return {
    defaultSwapFeeBps,
    protocolFeeBps,
    protocolFeePolicy:
      protocolFeeBps === 0
        ? "LP_ONLY_NO_PROTOCOL_SKIM"
        : "LEGACY_PROTOCOL_SKIM_CONFIGURED",
    minInitialLiquidity,
    twapWindowBlocks,
    maxPools,
    minReserve,
    billingQuoteDenoms,
    billingOracleEnabled,
    billingOraclePolicy: billingOracleEnabled
      ? "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED"
      : "DISABLED_FAIL_CLOSED",
    allowedPoolDenoms,
    poolCreators,
    poolCreationEnabled,
    poolCreationPolicy: poolCreationEnabled
      ? "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY"
      : "DISABLED_FAIL_CLOSED",
  };
}
