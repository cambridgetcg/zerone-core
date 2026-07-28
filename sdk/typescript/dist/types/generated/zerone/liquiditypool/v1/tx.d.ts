import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgCreatePool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePool
 */
export interface MsgCreatePool {
    /**
     * governance authority address
     */
    creator: string;
    denomA: string;
    denomB: string;
    /**
     * bigint string
     */
    amountA: string;
    /**
     * bigint string
     */
    amountB: string;
    /**
     * 0 = use default
     */
    swapFeeBps: bigint;
}
/**
 * @name MsgCreatePoolResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePoolResponse
 */
export interface MsgCreatePoolResponse {
    poolId: string;
}
/**
 * @name MsgSwap
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwap
 */
export interface MsgSwap {
    sender: string;
    poolId: string;
    tokenInDenom: string;
    /**
     * bigint string
     */
    tokenInAmount: string;
    /**
     * slippage protection (bigint string, optional)
     */
    minTokenOut: string;
}
/**
 * @name MsgSwapResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwapResponse
 */
export interface MsgSwapResponse {
    /**
     * bigint string
     */
    tokenOutAmount: string;
    /**
     * bigint string
     */
    feeAmount: string;
}
/**
 * @name MsgAddLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidity
 */
export interface MsgAddLiquidity {
    sender: string;
    poolId: string;
    /**
     * desired bigint string
     */
    amountA: string;
    /**
     * desired bigint string
     */
    amountB: string;
    /**
     * slippage protection (bigint string, optional)
     */
    minLpTokens: string;
}
/**
 * @name MsgAddLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidityResponse
 */
export interface MsgAddLiquidityResponse {
    /**
     * bigint string
     */
    lpTokensMinted: string;
    /**
     * bigint string
     */
    actualA: string;
    /**
     * bigint string
     */
    actualB: string;
}
/**
 * @name MsgRemoveLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidity
 */
export interface MsgRemoveLiquidity {
    sender: string;
    poolId: string;
    /**
     * bigint string
     */
    lpTokens: string;
    /**
     * slippage protection (bigint string, optional)
     */
    minAmountA: string;
    /**
     * slippage protection (bigint string, optional)
     */
    minAmountB: string;
}
/**
 * @name MsgRemoveLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidityResponse
 */
export interface MsgRemoveLiquidityResponse {
    /**
     * bigint string
     */
    amountA: string;
    /**
     * bigint string
     */
    amountB: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgCreatePool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePool
 */
export declare const MsgCreatePool: {
    typeUrl: string;
    encode(message: MsgCreatePool, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePool;
    fromPartial(object: DeepPartial<MsgCreatePool>): MsgCreatePool;
};
/**
 * @name MsgCreatePoolResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePoolResponse
 */
export declare const MsgCreatePoolResponse: {
    typeUrl: string;
    encode(message: MsgCreatePoolResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePoolResponse;
    fromPartial(object: DeepPartial<MsgCreatePoolResponse>): MsgCreatePoolResponse;
};
/**
 * @name MsgSwap
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwap
 */
export declare const MsgSwap: {
    typeUrl: string;
    encode(message: MsgSwap, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSwap;
    fromPartial(object: DeepPartial<MsgSwap>): MsgSwap;
};
/**
 * @name MsgSwapResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwapResponse
 */
export declare const MsgSwapResponse: {
    typeUrl: string;
    encode(message: MsgSwapResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSwapResponse;
    fromPartial(object: DeepPartial<MsgSwapResponse>): MsgSwapResponse;
};
/**
 * @name MsgAddLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidity
 */
export declare const MsgAddLiquidity: {
    typeUrl: string;
    encode(message: MsgAddLiquidity, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddLiquidity;
    fromPartial(object: DeepPartial<MsgAddLiquidity>): MsgAddLiquidity;
};
/**
 * @name MsgAddLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidityResponse
 */
export declare const MsgAddLiquidityResponse: {
    typeUrl: string;
    encode(message: MsgAddLiquidityResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddLiquidityResponse;
    fromPartial(object: DeepPartial<MsgAddLiquidityResponse>): MsgAddLiquidityResponse;
};
/**
 * @name MsgRemoveLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidity
 */
export declare const MsgRemoveLiquidity: {
    typeUrl: string;
    encode(message: MsgRemoveLiquidity, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveLiquidity;
    fromPartial(object: DeepPartial<MsgRemoveLiquidity>): MsgRemoveLiquidity;
};
/**
 * @name MsgRemoveLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidityResponse
 */
export declare const MsgRemoveLiquidityResponse: {
    typeUrl: string;
    encode(message: MsgRemoveLiquidityResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveLiquidityResponse;
    fromPartial(object: DeepPartial<MsgRemoveLiquidityResponse>): MsgRemoveLiquidityResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
