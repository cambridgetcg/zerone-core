import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * Pool represents a constant-product AMM liquidity pool.
 * @name Pool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Pool
 */
export interface Pool {
    poolId: string;
    denomA: string;
    denomB: string;
    /**
     * bigint string
     */
    reserveA: string;
    /**
     * bigint string
     */
    reserveB: string;
    /**
     * 1M bps scale (1,000,000 = 100%)
     */
    swapFeeBps: bigint;
    /**
     * bigint string
     */
    lpTokenSupply: string;
    /**
     * "lp/{pool_id}"
     */
    lpDenom: string;
    creator: string;
    createdAtBlock: bigint;
    /**
     * reentrancy guard
     */
    locked: boolean;
}
/**
 * TWAPAccumulator stores cumulative price data for TWAP oracle.
 * One record per pool, O(1) updates.
 * @name TWAPAccumulator
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.TWAPAccumulator
 */
export interface TWAPAccumulator {
    poolId: string;
    lastBlock: bigint;
    /**
     * bigint string (1e12 scale)
     */
    cumPriceAToB: string;
    /**
     * bigint string (1e12 scale)
     */
    cumPriceBToA: string;
    /**
     * height accumulation began — TWAP divisor is last_block - start_block
     */
    startBlock: bigint;
}
/**
 * SwapResult contains the output of a swap calculation.
 * @name SwapResult
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.SwapResult
 */
export interface SwapResult {
    tokenOutDenom: string;
    /**
     * bigint string
     */
    tokenOutAmount: string;
    /**
     * bigint string
     */
    feeAmount: string;
    /**
     * 1M bps scale
     */
    priceImpactBps: bigint;
}
/**
 * Pool represents a constant-product AMM liquidity pool.
 * @name Pool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Pool
 */
export declare const Pool: {
    typeUrl: string;
    encode(message: Pool, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Pool;
    fromPartial(object: DeepPartial<Pool>): Pool;
};
/**
 * TWAPAccumulator stores cumulative price data for TWAP oracle.
 * One record per pool, O(1) updates.
 * @name TWAPAccumulator
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.TWAPAccumulator
 */
export declare const TWAPAccumulator: {
    typeUrl: string;
    encode(message: TWAPAccumulator, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TWAPAccumulator;
    fromPartial(object: DeepPartial<TWAPAccumulator>): TWAPAccumulator;
};
/**
 * SwapResult contains the output of a swap calculation.
 * @name SwapResult
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.SwapResult
 */
export declare const SwapResult: {
    typeUrl: string;
    encode(message: SwapResult, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SwapResult;
    fromPartial(object: DeepPartial<SwapResult>): SwapResult;
};
