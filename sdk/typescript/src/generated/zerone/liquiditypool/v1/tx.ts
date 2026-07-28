//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
export interface MsgUpdateParamsResponse {}
function createBaseMsgCreatePool(): MsgCreatePool {
  return {
    creator: "",
    denomA: "",
    denomB: "",
    amountA: "",
    amountB: "",
    swapFeeBps: BigInt(0)
  };
}
/**
 * @name MsgCreatePool
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePool
 */
export const MsgCreatePool = {
  typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
  encode(message: MsgCreatePool, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.denomA !== "") {
      writer.uint32(18).string(message.denomA);
    }
    if (message.denomB !== "") {
      writer.uint32(26).string(message.denomB);
    }
    if (message.amountA !== "") {
      writer.uint32(34).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(42).string(message.amountB);
    }
    if (message.swapFeeBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.swapFeeBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePool {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.denomA = reader.string();
          break;
        case 3:
          message.denomB = reader.string();
          break;
        case 4:
          message.amountA = reader.string();
          break;
        case 5:
          message.amountB = reader.string();
          break;
        case 6:
          message.swapFeeBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreatePool>): MsgCreatePool {
    const message = createBaseMsgCreatePool();
    message.creator = object.creator ?? "";
    message.denomA = object.denomA ?? "";
    message.denomB = object.denomB ?? "";
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    message.swapFeeBps = object.swapFeeBps !== undefined && object.swapFeeBps !== null ? BigInt(object.swapFeeBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreatePoolResponse(): MsgCreatePoolResponse {
  return {
    poolId: ""
  };
}
/**
 * @name MsgCreatePoolResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgCreatePoolResponse
 */
export const MsgCreatePoolResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgCreatePoolResponse",
  encode(message: MsgCreatePoolResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.poolId !== "") {
      writer.uint32(10).string(message.poolId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePoolResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePoolResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.poolId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreatePoolResponse>): MsgCreatePoolResponse {
    const message = createBaseMsgCreatePoolResponse();
    message.poolId = object.poolId ?? "";
    return message;
  }
};
function createBaseMsgSwap(): MsgSwap {
  return {
    sender: "",
    poolId: "",
    tokenInDenom: "",
    tokenInAmount: "",
    minTokenOut: ""
  };
}
/**
 * @name MsgSwap
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwap
 */
export const MsgSwap = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
  encode(message: MsgSwap, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.tokenInDenom !== "") {
      writer.uint32(26).string(message.tokenInDenom);
    }
    if (message.tokenInAmount !== "") {
      writer.uint32(34).string(message.tokenInAmount);
    }
    if (message.minTokenOut !== "") {
      writer.uint32(42).string(message.minTokenOut);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSwap {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSwap();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.tokenInDenom = reader.string();
          break;
        case 4:
          message.tokenInAmount = reader.string();
          break;
        case 5:
          message.minTokenOut = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSwap>): MsgSwap {
    const message = createBaseMsgSwap();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.tokenInDenom = object.tokenInDenom ?? "";
    message.tokenInAmount = object.tokenInAmount ?? "";
    message.minTokenOut = object.minTokenOut ?? "";
    return message;
  }
};
function createBaseMsgSwapResponse(): MsgSwapResponse {
  return {
    tokenOutAmount: "",
    feeAmount: ""
  };
}
/**
 * @name MsgSwapResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgSwapResponse
 */
export const MsgSwapResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSwapResponse",
  encode(message: MsgSwapResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.tokenOutAmount !== "") {
      writer.uint32(10).string(message.tokenOutAmount);
    }
    if (message.feeAmount !== "") {
      writer.uint32(18).string(message.feeAmount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSwapResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSwapResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenOutAmount = reader.string();
          break;
        case 2:
          message.feeAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSwapResponse>): MsgSwapResponse {
    const message = createBaseMsgSwapResponse();
    message.tokenOutAmount = object.tokenOutAmount ?? "";
    message.feeAmount = object.feeAmount ?? "";
    return message;
  }
};
function createBaseMsgAddLiquidity(): MsgAddLiquidity {
  return {
    sender: "",
    poolId: "",
    amountA: "",
    amountB: "",
    minLpTokens: ""
  };
}
/**
 * @name MsgAddLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidity
 */
export const MsgAddLiquidity = {
  typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
  encode(message: MsgAddLiquidity, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.amountA !== "") {
      writer.uint32(26).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(34).string(message.amountB);
    }
    if (message.minLpTokens !== "") {
      writer.uint32(42).string(message.minLpTokens);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddLiquidity {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddLiquidity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.amountA = reader.string();
          break;
        case 4:
          message.amountB = reader.string();
          break;
        case 5:
          message.minLpTokens = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddLiquidity>): MsgAddLiquidity {
    const message = createBaseMsgAddLiquidity();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    message.minLpTokens = object.minLpTokens ?? "";
    return message;
  }
};
function createBaseMsgAddLiquidityResponse(): MsgAddLiquidityResponse {
  return {
    lpTokensMinted: "",
    actualA: "",
    actualB: ""
  };
}
/**
 * @name MsgAddLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgAddLiquidityResponse
 */
export const MsgAddLiquidityResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidityResponse",
  encode(message: MsgAddLiquidityResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.lpTokensMinted !== "") {
      writer.uint32(10).string(message.lpTokensMinted);
    }
    if (message.actualA !== "") {
      writer.uint32(18).string(message.actualA);
    }
    if (message.actualB !== "") {
      writer.uint32(26).string(message.actualB);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddLiquidityResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddLiquidityResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lpTokensMinted = reader.string();
          break;
        case 2:
          message.actualA = reader.string();
          break;
        case 3:
          message.actualB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddLiquidityResponse>): MsgAddLiquidityResponse {
    const message = createBaseMsgAddLiquidityResponse();
    message.lpTokensMinted = object.lpTokensMinted ?? "";
    message.actualA = object.actualA ?? "";
    message.actualB = object.actualB ?? "";
    return message;
  }
};
function createBaseMsgRemoveLiquidity(): MsgRemoveLiquidity {
  return {
    sender: "",
    poolId: "",
    lpTokens: "",
    minAmountA: "",
    minAmountB: ""
  };
}
/**
 * @name MsgRemoveLiquidity
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidity
 */
export const MsgRemoveLiquidity = {
  typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
  encode(message: MsgRemoveLiquidity, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.lpTokens !== "") {
      writer.uint32(26).string(message.lpTokens);
    }
    if (message.minAmountA !== "") {
      writer.uint32(34).string(message.minAmountA);
    }
    if (message.minAmountB !== "") {
      writer.uint32(42).string(message.minAmountB);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveLiquidity {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveLiquidity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.lpTokens = reader.string();
          break;
        case 4:
          message.minAmountA = reader.string();
          break;
        case 5:
          message.minAmountB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRemoveLiquidity>): MsgRemoveLiquidity {
    const message = createBaseMsgRemoveLiquidity();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.lpTokens = object.lpTokens ?? "";
    message.minAmountA = object.minAmountA ?? "";
    message.minAmountB = object.minAmountB ?? "";
    return message;
  }
};
function createBaseMsgRemoveLiquidityResponse(): MsgRemoveLiquidityResponse {
  return {
    amountA: "",
    amountB: ""
  };
}
/**
 * @name MsgRemoveLiquidityResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgRemoveLiquidityResponse
 */
export const MsgRemoveLiquidityResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidityResponse",
  encode(message: MsgRemoveLiquidityResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.amountA !== "") {
      writer.uint32(10).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(18).string(message.amountB);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveLiquidityResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveLiquidityResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amountA = reader.string();
          break;
        case 2:
          message.amountB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRemoveLiquidityResponse>): MsgRemoveLiquidityResponse {
    const message = createBaseMsgRemoveLiquidityResponse();
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    return message;
  }
};
function createBaseMsgUpdateParams(): MsgUpdateParams {
  return {
    authority: "",
    params: undefined
  };
}
/**
 * @name MsgUpdateParams
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
  encode(message: MsgUpdateParams, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams {
    const message = createBaseMsgUpdateParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse(): MsgUpdateParamsResponse {
  return {};
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParamsResponse",
  encode(_: MsgUpdateParamsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse {
    const message = createBaseMsgUpdateParamsResponse();
    return message;
  }
};