import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * PoolStatus is the governed lifecycle state of a pool. CLOSED is an
 * immutable tombstone produced only by the final LP exit.
 */
export declare enum PoolStatus {
    POOL_STATUS_UNSPECIFIED = 0,
    POOL_STATUS_ACTIVE = 1,
    POOL_STATUS_SWAPS_PAUSED = 2,
    POOL_STATUS_EXIT_ONLY = 3,
    POOL_STATUS_CLOSED = 4,
    UNRECOGNIZED = -1
}
export declare function poolStatusFromJSON(object: any): PoolStatus;
export declare function poolStatusToJSON(object: PoolStatus): string;
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
    status: PoolStatus;
    /**
     * zero unless status is CLOSED
     */
    closedAtBlock: bigint;
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
 * TWAPObservation is a retained cumulative-price checkpoint. One checkpoint
 * is stored per open pool per block, bounded by Params.twap_window_blocks.
 * @name TWAPObservation
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.TWAPObservation
 */
export interface TWAPObservation {
    poolId: string;
    blockHeight: bigint;
    /**
     * bigint string (1e12 scale)
     */
    cumPriceAToB: string;
    /**
     * bigint string (1e12 scale)
     */
    cumPriceBToA: string;
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
 * TWAPObservation is a retained cumulative-price checkpoint. One checkpoint
 * is stored per open pool per block, bounded by Params.twap_window_blocks.
 * @name TWAPObservation
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.TWAPObservation
 */
export declare const TWAPObservation: {
    typeUrl: string;
    encode(message: TWAPObservation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TWAPObservation;
    fromPartial(object: DeepPartial<TWAPObservation>): TWAPObservation;
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
