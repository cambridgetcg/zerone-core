//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBasePool(): Pool {
  return {
    poolId: "",
    denomA: "",
    denomB: "",
    reserveA: "",
    reserveB: "",
    swapFeeBps: BigInt(0),
    lpTokenSupply: "",
    lpDenom: "",
    creator: "",
    createdAtBlock: BigInt(0),
    locked: false
  };
}
/**
 * Pool represents a constant-product AMM liquidity pool.
 * @name Pool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Pool
 */
export const Pool = {
  typeUrl: "/zerone.liquiditypool.v1.Pool",
  encode(message: Pool, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.poolId !== "") {
      writer.uint32(10).string(message.poolId);
    }
    if (message.denomA !== "") {
      writer.uint32(18).string(message.denomA);
    }
    if (message.denomB !== "") {
      writer.uint32(26).string(message.denomB);
    }
    if (message.reserveA !== "") {
      writer.uint32(34).string(message.reserveA);
    }
    if (message.reserveB !== "") {
      writer.uint32(42).string(message.reserveB);
    }
    if (message.swapFeeBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.swapFeeBps);
    }
    if (message.lpTokenSupply !== "") {
      writer.uint32(58).string(message.lpTokenSupply);
    }
    if (message.lpDenom !== "") {
      writer.uint32(66).string(message.lpDenom);
    }
    if (message.creator !== "") {
      writer.uint32(74).string(message.creator);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.createdAtBlock);
    }
    if (message.locked === true) {
      writer.uint32(88).bool(message.locked);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Pool {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.poolId = reader.string();
          break;
        case 2:
          message.denomA = reader.string();
          break;
        case 3:
          message.denomB = reader.string();
          break;
        case 4:
          message.reserveA = reader.string();
          break;
        case 5:
          message.reserveB = reader.string();
          break;
        case 6:
          message.swapFeeBps = reader.uint64();
          break;
        case 7:
          message.lpTokenSupply = reader.string();
          break;
        case 8:
          message.lpDenom = reader.string();
          break;
        case 9:
          message.creator = reader.string();
          break;
        case 10:
          message.createdAtBlock = reader.uint64();
          break;
        case 11:
          message.locked = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Pool>): Pool {
    const message = createBasePool();
    message.poolId = object.poolId ?? "";
    message.denomA = object.denomA ?? "";
    message.denomB = object.denomB ?? "";
    message.reserveA = object.reserveA ?? "";
    message.reserveB = object.reserveB ?? "";
    message.swapFeeBps = object.swapFeeBps !== undefined && object.swapFeeBps !== null ? BigInt(object.swapFeeBps.toString()) : BigInt(0);
    message.lpTokenSupply = object.lpTokenSupply ?? "";
    message.lpDenom = object.lpDenom ?? "";
    message.creator = object.creator ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.locked = object.locked ?? false;
    return message;
  }
};
function createBaseTWAPAccumulator(): TWAPAccumulator {
  return {
    poolId: "",
    lastBlock: BigInt(0),
    cumPriceAToB: "",
    cumPriceBToA: "",
    startBlock: BigInt(0)
  };
}
/**
 * TWAPAccumulator stores cumulative price data for TWAP oracle.
 * One record per pool, O(1) updates.
 * @name TWAPAccumulator
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.TWAPAccumulator
 */
export const TWAPAccumulator = {
  typeUrl: "/zerone.liquiditypool.v1.TWAPAccumulator",
  encode(message: TWAPAccumulator, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.poolId !== "") {
      writer.uint32(10).string(message.poolId);
    }
    if (message.lastBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.lastBlock);
    }
    if (message.cumPriceAToB !== "") {
      writer.uint32(26).string(message.cumPriceAToB);
    }
    if (message.cumPriceBToA !== "") {
      writer.uint32(34).string(message.cumPriceBToA);
    }
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.startBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TWAPAccumulator {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTWAPAccumulator();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.poolId = reader.string();
          break;
        case 2:
          message.lastBlock = reader.uint64();
          break;
        case 3:
          message.cumPriceAToB = reader.string();
          break;
        case 4:
          message.cumPriceBToA = reader.string();
          break;
        case 5:
          message.startBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TWAPAccumulator>): TWAPAccumulator {
    const message = createBaseTWAPAccumulator();
    message.poolId = object.poolId ?? "";
    message.lastBlock = object.lastBlock !== undefined && object.lastBlock !== null ? BigInt(object.lastBlock.toString()) : BigInt(0);
    message.cumPriceAToB = object.cumPriceAToB ?? "";
    message.cumPriceBToA = object.cumPriceBToA ?? "";
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseSwapResult(): SwapResult {
  return {
    tokenOutDenom: "",
    tokenOutAmount: "",
    feeAmount: "",
    priceImpactBps: BigInt(0)
  };
}
/**
 * SwapResult contains the output of a swap calculation.
 * @name SwapResult
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.SwapResult
 */
export const SwapResult = {
  typeUrl: "/zerone.liquiditypool.v1.SwapResult",
  encode(message: SwapResult, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.tokenOutDenom !== "") {
      writer.uint32(10).string(message.tokenOutDenom);
    }
    if (message.tokenOutAmount !== "") {
      writer.uint32(18).string(message.tokenOutAmount);
    }
    if (message.feeAmount !== "") {
      writer.uint32(26).string(message.feeAmount);
    }
    if (message.priceImpactBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.priceImpactBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SwapResult {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSwapResult();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenOutDenom = reader.string();
          break;
        case 2:
          message.tokenOutAmount = reader.string();
          break;
        case 3:
          message.feeAmount = reader.string();
          break;
        case 4:
          message.priceImpactBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SwapResult>): SwapResult {
    const message = createBaseSwapResult();
    message.tokenOutDenom = object.tokenOutDenom ?? "";
    message.tokenOutAmount = object.tokenOutAmount ?? "";
    message.feeAmount = object.feeAmount ?? "";
    message.priceImpactBps = object.priceImpactBps !== undefined && object.priceImpactBps !== null ? BigInt(object.priceImpactBps.toString()) : BigInt(0);
    return message;
  }
};