import { CHAIN_ID, DECIMALS, DENOM, REST_ENDPOINT, RPC_ENDPOINT } from "./config";
import type { FeeGrantAllowance } from "./feegrant";

interface RpcStatusResponse {
  result?: {
    node_info?: { network?: string; version?: string; moniker?: string };
    sync_info?: {
      latest_block_height?: string;
      latest_block_time?: string;
      catching_up?: boolean;
    };
  };
}

interface RpcNetInfoResponse {
  result?: { n_peers?: string };
}

interface SupplyResponse {
  amount?: { denom?: string; amount?: string };
}

interface RpcValidatorsResponse {
  result?: {
    validators?: Array<{ address?: string; voting_power?: string }>;
    total?: string;
  };
}

interface RawPool {
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
}

interface PoolsResponse {
  pools?: RawPool[];
}

interface ParamsResponse {
  params?: {
    defaultSwapFeeBps?: string;
    default_swap_fee_bps?: string;
    protocolFeeBps?: string;
    protocol_fee_bps?: string;
    minInitialLiquidity?: string;
    min_initial_liquidity?: string;
    twapWindowBlocks?: string;
    twap_window_blocks?: string;
  };
}

interface RpcBlockchainResponse {
  result?: {
    block_metas?: Array<{
      block_id?: { hash?: string };
      header?: { height?: string; time?: string };
      num_txs?: string;
    }>;
  };
}

interface RawAccountIdentifier {
  namespace?: unknown;
  reference?: unknown;
  rawChainId?: unknown;
  raw_chain_id?: unknown;
  accountId?: unknown;
  account_id?: unknown;
  address?: unknown;
  did?: unknown;
  accountType?: unknown;
  account_type?: unknown;
  frozen?: unknown;
  createdAtBlock?: unknown;
  created_at_block?: unknown;
}

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
}

export interface LiquidityParams {
  defaultSwapFeeBps: number;
  protocolFeeBps: number;
  minInitialLiquidity: string;
  twapWindowBlocks: number;
}

export interface NetworkSnapshot {
  chainId: string;
  height: number;
  blockTime: string;
  catchingUp: boolean;
  cometVersion: string;
  peers: number | null;
  supplyUzrn: string | null;
  validators: number | null;
  validatorMonikers: string[];
  pools: LiquidityPool[] | null;
  liquidityParams: LiquidityParams | null;
}

export interface RecentBlock {
  height: number;
  time: string;
  transactionCount: number;
  hash: string;
}

export interface AccountIdentifier {
  namespace: string;
  reference: string;
  rawChainId: string;
  accountId: string;
  address: string;
  did: string;
  accountType: string;
  frozen: boolean;
  createdAtBlock: string;
}

async function fetchJson<T>(url: string, timeoutMs = 8_000): Promise<T> {
  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(timeoutMs),
  });
  if (!response.ok) throw new Error(`Mainnet returned HTTP ${response.status}`);
  return (await response.json()) as T;
}

async function boundedResponseJson(
  response: Response,
  maximumBytes = 1_048_576,
): Promise<unknown> {
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!/^\d+$/.test(declaredLength) ||
      Number(declaredLength) > maximumBytes)
  ) {
    throw new Error("Mainnet response exceeded its size limit.");
  }
  const raw = await response.text();
  if (new TextEncoder().encode(raw).byteLength > maximumBytes) {
    throw new Error("Mainnet response exceeded its size limit.");
  }
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    throw new Error("Mainnet returned malformed JSON.");
  }
}

function rpcUrl(path: string, params?: Record<string, string>): string {
  const url = new URL(`${RPC_ENDPOINT}/${path.replace(/^\//, "")}`);
  Object.entries(params ?? {}).forEach(([key, value]) => url.searchParams.set(key, value));
  return url.toString();
}

function restUrl(path: string): string {
  return `${REST_ENDPOINT}/${path.replace(/^\//, "")}`;
}

function uint(value: unknown): number | null {
  if (typeof value !== "string" && typeof value !== "number") return null;
  if (!/^\d+$/.test(String(value))) return null;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed >= 0 ? parsed : null;
}

function amount(value: unknown): string | null {
  return typeof value === "string" && /^\d+$/.test(value) ? value : null;
}

function boundedText(value: unknown, maxLength = 256): string | null {
  return typeof value === "string" && value.length > 0 && value.length <= maxLength
    ? value
    : null;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function uintString(value: unknown): string | null {
  if (typeof value !== "string" && typeof value !== "number") return null;
  const normalized = String(value);
  return /^\d+$/.test(normalized) ? normalized : null;
}

function normalizeAccountIdentifier(
  response: unknown,
): AccountIdentifier | null {
  if (!isRecord(response)) return null;
  const identifier = response.identifier;
  if (!isRecord(identifier)) return null;
  const raw = identifier as RawAccountIdentifier;

  const namespace = boundedText(raw.namespace, 32);
  const reference = boundedText(raw.reference, 32);
  const rawChainId = boundedText(
    raw.rawChainId ?? raw.raw_chain_id,
    128,
  );
  const accountId = boundedText(
    raw.accountId ?? raw.account_id,
    256,
  );
  const address = boundedText(raw.address, 128);
  const did = boundedText(raw.did, 256);
  const accountType = boundedText(
    raw.accountType ?? raw.account_type,
    128,
  );
  const createdAtBlock = uintString(
    raw.createdAtBlock ?? raw.created_at_block,
  );
  const frozen = raw.frozen === undefined ? false : raw.frozen;
  if (
    namespace === null ||
    reference === null ||
    rawChainId === null ||
    accountId === null ||
    address === null ||
    did === null ||
    accountType === null ||
    typeof frozen !== "boolean" ||
    createdAtBlock === null
  ) {
    return null;
  }

  return {
    namespace,
    reference,
    rawChainId,
    accountId,
    address,
    did,
    accountType,
    frozen,
    createdAtBlock,
  };
}

function normalizePool(pool: RawPool): LiquidityPool | null {
  const id = boundedText(pool.poolId ?? pool.pool_id);
  const denomA = boundedText(pool.denomA ?? pool.denom_a, 160);
  const denomB = boundedText(pool.denomB ?? pool.denom_b, 160);
  const reserveA = amount(pool.reserveA ?? pool.reserve_a);
  const reserveB = amount(pool.reserveB ?? pool.reserve_b);
  const fee = uint(pool.swapFeeBps ?? pool.swap_fee_bps);
  const lpSupply = amount(pool.lpTokenSupply ?? pool.lp_token_supply);
  const createdAtBlock = uint(pool.createdAtBlock ?? pool.created_at_block);
  if (
    !id ||
    !denomA ||
    !denomB ||
    reserveA === null ||
    reserveB === null ||
    fee === null ||
    lpSupply === null ||
    createdAtBlock === null
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
    lpDenom: boundedText(pool.lpDenom ?? pool.lp_denom, 256) ?? "—",
    creator: boundedText(pool.creator, 128) ?? "—",
    createdAtBlock,
    locked: Boolean(pool.locked),
  };
}

function normalizeParams(response: ParamsResponse): LiquidityParams | null {
  const params = response.params;
  if (!params) return null;
  const defaultSwapFeeBps = uint(
    params.defaultSwapFeeBps ?? params.default_swap_fee_bps,
  );
  const protocolFeeBps = uint(params.protocolFeeBps ?? params.protocol_fee_bps);
  const minInitialLiquidity = amount(
    params.minInitialLiquidity ?? params.min_initial_liquidity,
  );
  const twapWindowBlocks = uint(params.twapWindowBlocks ?? params.twap_window_blocks);
  if (
    defaultSwapFeeBps === null ||
    protocolFeeBps === null ||
    minInitialLiquidity === null ||
    twapWindowBlocks === null
  ) {
    return null;
  }
  return {
    defaultSwapFeeBps,
    protocolFeeBps,
    minInitialLiquidity,
    twapWindowBlocks,
  };
}

export async function getNetworkSnapshot(): Promise<NetworkSnapshot> {
  const status = await fetchJson<RpcStatusResponse>(rpcUrl("status"));
  const statusResult = status.result;
  const chainId = statusResult?.node_info?.network;
  const height = uint(statusResult?.sync_info?.latest_block_height);
  const blockTime = statusResult?.sync_info?.latest_block_time;
  const catchingUp = statusResult?.sync_info?.catching_up;
  if (
    !statusResult ||
    chainId !== CHAIN_ID ||
    height === null ||
    height <= 0 ||
    typeof blockTime !== "string" ||
    !Number.isFinite(Date.parse(blockTime)) ||
    typeof catchingUp !== "boolean"
  ) {
    throw new Error(`Expected a complete ${CHAIN_ID} mainnet status response`);
  }

  const [netInfoResult, supplyResult, validatorsResult, poolsResult, paramsResult] =
    await Promise.allSettled([
      fetchJson<RpcNetInfoResponse>(rpcUrl("net_info")),
      fetchJson<SupplyResponse>(
        restUrl(`/cosmos/bank/v1beta1/supply/by_denom?denom=${DENOM}`),
      ),
      fetchJson<RpcValidatorsResponse>(rpcUrl("validators", { page: "1", per_page: "100" })),
      fetchJson<PoolsResponse>(restUrl("/zerone/liquiditypool/v1/pools")),
      fetchJson<ParamsResponse>(restUrl("/zerone/liquiditypool/v1/params")),
    ]);

  const netInfo = netInfoResult.status === "fulfilled" ? netInfoResult.value : {};
  const supply = supplyResult.status === "fulfilled" ? supplyResult.value : {};
  const validators = validatorsResult.status === "fulfilled" ? validatorsResult.value : {};
  const pools = poolsResult.status === "fulfilled" ? poolsResult.value : {};
  const params = paramsResult.status === "fulfilled" ? paramsResult.value : {};
  const validatorList = Array.isArray(validators.result?.validators)
    ? validators.result.validators
    : null;
  const peerCount = uint(netInfo.result?.n_peers);
  const issuedSupply =
    supply.amount?.denom === DENOM ? amount(supply.amount.amount) : null;
  const validatorTotal = uint(validators.result?.total);
  const rawPools = Array.isArray(pools.pools) ? pools.pools : null;
  const normalizedPools = rawPools?.map(normalizePool) ?? null;
  const poolsValid = normalizedPools?.every(
    (pool): pool is LiquidityPool => pool !== null,
  );

  return {
    chainId,
    height,
    blockTime,
    catchingUp,
    cometVersion: statusResult.node_info?.version ?? "unknown",
    peers:
      netInfoResult.status === "fulfilled" && peerCount !== null ? peerCount : null,
    supplyUzrn:
      supplyResult.status === "fulfilled" ? issuedSupply : null,
    validators:
      validatorsResult.status === "fulfilled" && validatorList !== null
        ? (validatorTotal ?? validatorList.length)
        : null,
    validatorMonikers: statusResult.node_info?.moniker
      ? [statusResult.node_info.moniker]
      : [],
    pools:
      poolsResult.status === "fulfilled" && poolsValid ? normalizedPools : null,
    liquidityParams:
      paramsResult.status === "fulfilled" ? normalizeParams(params) : null,
  };
}

export async function getRecentBlocks(latestHeight: number, count = 5): Promise<RecentBlock[]> {
  const safeCount = Math.min(8, Math.max(1, count));
  const minHeight = Math.max(1, latestHeight - safeCount + 1);
  const response = await fetchJson<RpcBlockchainResponse>(
    rpcUrl("blockchain", {
      minHeight: String(minHeight),
      maxHeight: String(latestHeight),
    }),
  );
  const metas = response.result?.block_metas;
  if (!Array.isArray(metas)) return [];

  return metas.flatMap((meta) => {
    const height = uint(meta.header?.height);
    const transactionCount = uint(meta.num_txs);
    const time = meta.header?.time;
    const hash = meta.block_id?.hash;
    if (
      height === null ||
      transactionCount === null ||
      typeof time !== "string" ||
      !Number.isFinite(Date.parse(time)) ||
      typeof hash !== "string" ||
      !/^[A-F0-9]{64}$/i.test(hash)
    ) {
      return [];
    }
    return [{ height, time, transactionCount, hash }];
  });
}

export async function getWalletBalance(address: string): Promise<string> {
  const response = await fetchJson<{
    balance?: { denom?: string; amount?: string };
  }>(restUrl(`/cosmos/bank/v1beta1/balances/${encodeURIComponent(address)}/by_denom?denom=${DENOM}`));
  const value = amount(response.balance?.amount);
  if (response.balance?.denom !== DENOM || value === null) {
    throw new Error("Wallet balance response was incomplete");
  }
  return value;
}

export async function getAccountIdentifier(
  address: string,
): Promise<AccountIdentifier | null> {
  let response: Response;
  try {
    response = await fetch(
      restUrl(
        `/zerone/auth/v1/account_identifier/${encodeURIComponent(address)}`,
      ),
      {
        headers: { Accept: "application/json" },
        signal: AbortSignal.timeout(8_000),
      },
    );
  } catch {
    // The first deployed dashboard may briefly run against a node binary that
    // predates this read-only query. Local CAIP derivation remains available.
    return null;
  }

  if (!response.ok) return null;

  let body: unknown;
  try {
    body = await response.json();
  } catch {
    throw new Error(
      "Mainnet returned a successful but malformed account identifier response.",
    );
  }
  const identifier = normalizeAccountIdentifier(body);
  if (!identifier) {
    throw new Error(
      "Mainnet returned a successful but incomplete account identifier response.",
    );
  }
  return identifier;
}

async function parseFeeGrantResponse(value: unknown): Promise<{
  allowances: FeeGrantAllowance[];
  nextKey: string | null;
}> {
  const { parseFeeGrantPage } = await import("./feegrant");
  const page = parseFeeGrantPage(value);
  return { allowances: page.allowances, nextKey: page.nextKey };
}

export async function getFeeGrantsByGrantee(
  grantee: string,
): Promise<FeeGrantAllowance[]> {
  const grants: FeeGrantAllowance[] = [];
  const seenKeys = new Set<string>();
  let nextKey: string | null = null;
  for (let pageNumber = 0; pageNumber < 10; pageNumber += 1) {
    const query = new URLSearchParams({ "pagination.limit": "50" });
    if (nextKey !== null) query.set("pagination.key", nextKey);
    const response = await fetchJson<unknown>(
      restUrl(
        `/cosmos/feegrant/v1beta1/allowances/${encodeURIComponent(grantee)}?${query.toString()}`,
      ),
    );
    const page = await parseFeeGrantResponse(response);
    if (page.allowances.some((grant) => grant.grantee !== grantee)) {
      throw new Error(
        "Mainnet returned a fee grant for a different grantee.",
      );
    }
    grants.push(...page.allowances);
    if (page.nextKey === null) return grants;
    if (seenKeys.has(page.nextKey)) {
      throw new Error("Fee grant pagination repeated a cursor.");
    }
    seenKeys.add(page.nextKey);
    nextKey = page.nextKey;
  }
  throw new Error("Fee grant response exceeded the page limit.");
}

export async function getFeeGrant(
  granter: string,
  grantee: string,
): Promise<FeeGrantAllowance | null> {
  const response = await fetch(
    restUrl(
      `/cosmos/feegrant/v1beta1/allowance/${encodeURIComponent(granter)}/${encodeURIComponent(grantee)}`,
    ),
    {
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(8_000),
    },
  );
  if (response.status === 404) return null;
  let body: unknown;
  try {
    body = await boundedResponseJson(response);
  } catch (error) {
    if (!response.ok) {
      throw new Error(`Mainnet returned HTTP ${response.status}`);
    }
    throw error;
  }
  if (response.status === 500) {
    const { isMissingFeeGrantError } = await import("./feegrant");
    if (isMissingFeeGrantError(body)) return null;
  }
  if (!response.ok) {
    throw new Error(`Mainnet returned HTTP ${response.status}`);
  }
  if (!isRecord(body) || !isRecord(body.allowance)) {
    throw new Error("Mainnet returned an incomplete fee grant response.");
  }
  const parsed = await parseFeeGrantResponse({
    allowances: [body.allowance],
    pagination: { next_key: null },
  });
  const grant = parsed.allowances[0] ?? null;
  if (
    grant !== null &&
    (grant.granter !== granter || grant.grantee !== grantee)
  ) {
    throw new Error(
      "Mainnet returned a fee grant for a different account pair.",
    );
  }
  return grant;
}

export function microToDisplay(amount: string, maximumFractionDigits = DECIMALS): string {
  const numeric = Number(amount) / 10 ** DECIMALS;
  if (!Number.isFinite(numeric)) return "0";
  return new Intl.NumberFormat("en-GB", {
    minimumFractionDigits: 0,
    maximumFractionDigits,
  }).format(numeric);
}
