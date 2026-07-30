import type { EncodeObject } from "@cosmjs/proto-signing";
import { MsgCreatePool } from "./generated/zerone/liquiditypool/v1/tx.js";
export declare const LIQUIDITY_FEE_SCALE = 1000000n;
export declare const ZERONE_MAX_SWAP_FEE = 100000n;
export declare const COSMOS_UINT64_MAX = 18446744073709551615n;
export declare const COSMOS_AMOUNT_MAX: bigint;
export declare const ZERONE_MAX_POOL_RECORDS = 10000n;
export declare const MSG_CREATE_POOL_TYPE_URL: "/zerone.liquiditypool.v1.MsgCreatePool";
export declare const MSG_SWAP_TYPE_URL: "/zerone.liquiditypool.v1.MsgSwap";
export declare const MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL: "/zerone.liquiditypool.v1.MsgUpdateParams";
export declare const MSG_SUBMIT_PROPOSAL_TYPE_URL: "/cosmos.gov.v1.MsgSubmitProposal";
export declare const LIQUIDITY_POOL_STATUS: Readonly<{
    readonly UNSPECIFIED: 0;
    readonly ACTIVE: 1;
    readonly SWAPS_PAUSED: 2;
    readonly EXIT_ONLY: 3;
    readonly CLOSED: 4;
}>;
export type LiquidityPoolStatus = keyof typeof LIQUIDITY_POOL_STATUS | "PRE_V4";
export type LiquidityClientErrorCode = "INVALID_AMOUNT" | "INVALID_UINT64" | "INVALID_FEE" | "INVALID_SLIPPAGE" | "INVALID_ADDRESS" | "INVALID_DENOM" | "INVALID_POOL" | "POOL_NOT_ACTIVE" | "QUOTE_TOO_SMALL" | "INVALID_RESPONSE" | "RESPONSE_TOO_LARGE" | "HTTP_ERROR";
export declare class LiquidityClientError extends Error {
    readonly code: LiquidityClientErrorCode;
    readonly field: string;
    constructor(code: LiquidityClientErrorCode, field: string, message: string);
}
export interface ZeroneLiquidityPool {
    readonly poolId: string;
    readonly denomA: string;
    readonly denomB: string;
    readonly reserveA: string;
    readonly reserveB: string;
    /**
     * The legacy protobuf field is named `swap_fee_bps`, but its scale is one
     * million. This API uses the unambiguous "millionths" name.
     */
    readonly swapFeeMillionths: bigint;
    readonly lpTokenSupply: string;
    readonly lpDenom: string;
    readonly creator: string;
    readonly createdAtBlock: bigint;
    readonly locked: boolean;
    /**
     * PRE_V4 means the REST representation did not expose the lifecycle field.
     * It is not treated as ACTIVE by quote helpers.
     */
    readonly status: LiquidityPoolStatus;
    readonly closedAtBlock: bigint | null;
}
export type BillingOraclePolicy = "DISABLED_FAIL_CLOSED" | "QUOTE_ALLOWLIST_CONFIGURED_LIVE_ELIGIBILITY_REQUIRED";
export type PoolCreationPolicy = "DISABLED_FAIL_CLOSED" | "ONE_SHOT_DENOM_GRANTS_AND_ALLOWLISTED_CREATORS_ONLY";
export interface ZeroneLiquidityParams {
    readonly defaultSwapFeeMillionths: bigint;
    readonly maxPools: bigint;
    readonly minInitialLiquidity: string;
    readonly twapWindowBlocks: bigint;
    readonly protocolFeeMillionths: bigint;
    readonly minReserve: string;
    readonly billingQuoteDenoms: readonly string[];
    readonly billingOracleEnabled: boolean;
    readonly billingOraclePolicy: BillingOraclePolicy;
    /** Unconsumed one-shot counter-denom creation grants. */
    readonly allowedPoolDenoms: readonly string[];
    readonly poolCreators: readonly string[];
    readonly poolCreationEnabled: boolean;
    readonly poolCreationPolicy: PoolCreationPolicy;
}
export interface LiquidityPoolPage {
    readonly pools: readonly ZeroneLiquidityPool[];
    readonly nextKey: string | null;
    readonly total: bigint | null;
}
export interface LiquidityPoolPageRequest {
    readonly limit?: number;
    readonly key?: string;
}
export interface LiquidityTwap {
    readonly price: string;
    /**
     * The server-reported effective window. Callers must not infer this from
     * Params.twapWindowBlocks alone.
     */
    readonly windowUsed: bigint;
}
export interface LiquiditySwapSimulation {
    readonly tokenOutDenom: string;
    readonly tokenOutAmount: string;
    readonly feeAmount: string;
    readonly priceImpactMillionths: bigint;
}
export interface ConstantProductExactInRequest {
    readonly reserveIn: string;
    readonly reserveOut: string;
    readonly tokenInAmount: string;
    readonly swapFeeMillionths: bigint;
    readonly slippageMillionths: bigint;
}
export interface ConstantProductExactInQuote {
    readonly reserveIn: string;
    readonly reserveOut: string;
    readonly tokenInAmount: string;
    /**
     * Whole base units remaining after the separately reported, floor-rounded
     * fee amount. The curve itself uses weightedTokenIn so sub-unit fees are not
     * lost when feeAmount rounds to zero.
     */
    readonly effectiveTokenIn: string;
    /**
     * tokenInAmount * (LIQUIDITY_FEE_SCALE - swapFeeMillionths), exactly
     * matching the scaled integer used by the chain's AMM denominator.
     */
    readonly weightedTokenIn: string;
    readonly feeAmount: string;
    readonly tokenOutAmount: string;
    readonly minimumTokenOut: string;
    readonly swapFeeMillionths: bigint;
    readonly slippageMillionths: bigint;
    readonly priceImpactMillionths: bigint;
}
export interface ExactInLiquidityQuoteRequest {
    readonly poolId: string;
    readonly tokenInDenom: string;
    readonly tokenInAmount: string;
    readonly slippageMillionths: bigint;
}
export interface ExactInLiquidityQuote extends ConstantProductExactInQuote {
    readonly adapterId: string;
    readonly poolId: string;
    readonly tokenInDenom: string;
    readonly tokenOutDenom: string;
    readonly poolStatus: LiquidityPoolStatus;
}
/**
 * A narrow integration seam for external routers, including Osmosis-aware
 * implementations. The SDK intentionally does not import an Osmosis client.
 */
export interface ExactInLiquidityAdapter {
    readonly id: string;
    quoteExactIn(request: ExactInLiquidityQuoteRequest): Promise<ExactInLiquidityQuote>;
}
export type LiquidityFetch = (input: string | URL | Request, init?: RequestInit) => Promise<Response>;
export interface ZeroneLiquidityRestClientOptions {
    readonly baseUrl: string;
    readonly fetch?: LiquidityFetch;
    readonly timeoutMs?: number;
    readonly maximumResponseBytes?: number;
}
export interface CosmosCoin {
    readonly denom: string;
    readonly amount: string;
}
export interface CosmosProtoAny {
    readonly typeUrl: string;
    readonly value: Uint8Array;
}
export interface CosmosGovV1SubmitProposal {
    readonly messages: readonly CosmosProtoAny[];
    readonly initialDeposit: readonly CosmosCoin[];
    readonly proposer: string;
    readonly metadata: string;
    readonly title: string;
    readonly summary: string;
    readonly expedited: boolean;
}
export interface LiquidityAdmissionProposalEncodeObject extends EncodeObject {
    readonly typeUrl: typeof MSG_SUBMIT_PROPOSAL_TYPE_URL;
    readonly value: CosmosGovV1SubmitProposal;
}
export interface LiquidityAdmissionProposalRequest {
    /** The x/liquiditypool authority placed inside MsgUpdateParams. */
    readonly authority: string;
    /** The account submitting the governance proposal and its deposit. */
    readonly proposer: string;
    /** Query this immediately before building; every Params field is replaced. */
    readonly currentParams: ZeroneLiquidityParams;
    /** Replacement set of unconsumed one-shot counter-denom grants. */
    readonly allowedPoolDenoms: readonly string[];
    /** Persistent creator/funder allowlist. */
    readonly poolCreators: readonly string[];
    readonly initialDeposit?: readonly CosmosCoin[];
    readonly title: string;
    readonly summary: string;
    readonly metadata?: string;
    readonly expedited?: boolean;
}
export interface CreatePoolMessageRequest {
    readonly creator: string;
    readonly denomA: string;
    readonly denomB: string;
    readonly amountA: string;
    readonly amountB: string;
    /**
     * Fresh queried params used to check the pending one-shot denom grant and
     * persistent creator allowlist locally.
     */
    readonly params: ZeroneLiquidityParams;
}
export interface CreatePoolEncodeObject extends EncodeObject {
    readonly typeUrl: typeof MSG_CREATE_POOL_TYPE_URL;
    readonly value: MsgCreatePool;
}
export interface TimedTransactionPlan {
    readonly messages: readonly EncodeObject[];
    /**
     * Pass this as the fifth argument to CosmJS signAndBroadcast or
     * signAndBroadcastSync.
     */
    readonly timeoutHeight: bigint;
}
export interface ExactInSwapPlanRequest {
    readonly sender: string;
    readonly poolId: string;
    readonly tokenInDenom: string;
    readonly tokenInAmount: string;
    readonly minimumTokenOut: string;
    readonly timeoutHeight: bigint;
}
export declare function parseCanonicalPositiveAmount(value: string, field?: string): bigint;
export declare function minimumOutputForSlippage(tokenOutAmount: string, slippageMillionths: bigint): string;
export declare function quoteConstantProductExactIn(request: ConstantProductExactInRequest): ConstantProductExactInQuote;
export declare class ZeroneLiquidityRestClient implements ExactInLiquidityAdapter {
    #private;
    readonly id = "zerone/liquiditypool/v1";
    constructor(options: ZeroneLiquidityRestClientOptions);
    pool(poolId: string): Promise<ZeroneLiquidityPool>;
    pools(request?: LiquidityPoolPageRequest): Promise<LiquidityPoolPage>;
    params(): Promise<ZeroneLiquidityParams>;
    twap(poolId: string, baseDenom: string, window?: bigint): Promise<LiquidityTwap>;
    simulateSwap(poolId: string, tokenInDenom: string, tokenInAmount: string): Promise<LiquiditySwapSimulation>;
    quoteExactIn(request: ExactInLiquidityQuoteRequest): Promise<ExactInLiquidityQuote>;
}
export declare function createLiquidityAdmissionUpdateMessage(request: LiquidityAdmissionProposalRequest): CosmosProtoAny;
export declare function createLiquidityAdmissionProposal(request: LiquidityAdmissionProposalRequest): LiquidityAdmissionProposalEncodeObject;
export declare function createPoolMessage(request: CreatePoolMessageRequest): CreatePoolEncodeObject;
export declare function timeoutHeightAfter(currentHeight: bigint, lifetimeBlocks: bigint): bigint;
export declare function withTimeoutHeight(messages: readonly EncodeObject[], timeoutHeight: bigint): TimedTransactionPlan;
export declare function createExactInSwapPlan(request: ExactInSwapPlanRequest): TimedTransactionPlan;
