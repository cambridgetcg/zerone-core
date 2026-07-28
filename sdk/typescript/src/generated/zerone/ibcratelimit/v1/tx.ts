//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgAddRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimit
 */
export interface MsgAddRateLimit {
  /**
   * governance address
   */
  authority: string;
  channelId: string;
  denom: string;
  /**
   * bigint string
   */
  maxSend: string;
  /**
   * bigint string
   */
  maxRecv: string;
  windowBlocks: bigint;
}
/**
 * @name MsgAddRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimitResponse
 */
export interface MsgAddRateLimitResponse {}
/**
 * @name MsgRemoveRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimit
 */
export interface MsgRemoveRateLimit {
  /**
   * governance address
   */
  authority: string;
  channelId: string;
  denom: string;
}
/**
 * @name MsgRemoveRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse
 */
export interface MsgRemoveRateLimitResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  /**
   * governance address
   */
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgAddRateLimit(): MsgAddRateLimit {
  return {
    authority: "",
    channelId: "",
    denom: "",
    maxSend: "",
    maxRecv: "",
    windowBlocks: BigInt(0)
  };
}
/**
 * @name MsgAddRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimit
 */
export const MsgAddRateLimit = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
  encode(message: MsgAddRateLimit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.denom !== "") {
      writer.uint32(26).string(message.denom);
    }
    if (message.maxSend !== "") {
      writer.uint32(34).string(message.maxSend);
    }
    if (message.maxRecv !== "") {
      writer.uint32(42).string(message.maxRecv);
    }
    if (message.windowBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.windowBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddRateLimit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddRateLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.denom = reader.string();
          break;
        case 4:
          message.maxSend = reader.string();
          break;
        case 5:
          message.maxRecv = reader.string();
          break;
        case 6:
          message.windowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddRateLimit>): MsgAddRateLimit {
    const message = createBaseMsgAddRateLimit();
    message.authority = object.authority ?? "";
    message.channelId = object.channelId ?? "";
    message.denom = object.denom ?? "";
    message.maxSend = object.maxSend ?? "";
    message.maxRecv = object.maxRecv ?? "";
    message.windowBlocks = object.windowBlocks !== undefined && object.windowBlocks !== null ? BigInt(object.windowBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddRateLimitResponse(): MsgAddRateLimitResponse {
  return {};
}
/**
 * @name MsgAddRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimitResponse
 */
export const MsgAddRateLimitResponse = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimitResponse",
  encode(_: MsgAddRateLimitResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddRateLimitResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddRateLimitResponse();
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
  fromPartial(_: DeepPartial<MsgAddRateLimitResponse>): MsgAddRateLimitResponse {
    const message = createBaseMsgAddRateLimitResponse();
    return message;
  }
};
function createBaseMsgRemoveRateLimit(): MsgRemoveRateLimit {
  return {
    authority: "",
    channelId: "",
    denom: ""
  };
}
/**
 * @name MsgRemoveRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimit
 */
export const MsgRemoveRateLimit = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
  encode(message: MsgRemoveRateLimit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.denom !== "") {
      writer.uint32(26).string(message.denom);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveRateLimit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveRateLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRemoveRateLimit>): MsgRemoveRateLimit {
    const message = createBaseMsgRemoveRateLimit();
    message.authority = object.authority ?? "";
    message.channelId = object.channelId ?? "";
    message.denom = object.denom ?? "";
    return message;
  }
};
function createBaseMsgRemoveRateLimitResponse(): MsgRemoveRateLimitResponse {
  return {};
}
/**
 * @name MsgRemoveRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse
 */
export const MsgRemoveRateLimitResponse = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse",
  encode(_: MsgRemoveRateLimitResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveRateLimitResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveRateLimitResponse();
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
  fromPartial(_: DeepPartial<MsgRemoveRateLimitResponse>): MsgRemoveRateLimitResponse {
    const message = createBaseMsgRemoveRateLimitResponse();
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
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
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
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParamsResponse",
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