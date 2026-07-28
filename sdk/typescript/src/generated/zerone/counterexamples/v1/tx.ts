//@ts-nocheck
import { ErrorType, CounterexampleStatus } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgProposeCounterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexample
 */
export interface MsgProposeCounterexample {
  author: string;
  factId: string;
  wrongClaim: string;
  reasoning: string;
  errorType: ErrorType;
  violatedMethodologyIds: string[];
}
/**
 * @name MsgProposeCounterexampleResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexampleResponse
 */
export interface MsgProposeCounterexampleResponse {
  counterexampleId: string;
}
/**
 * @name MsgValidate
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidate
 */
export interface MsgValidate {
  validator: string;
  counterexampleId: string;
  affirm: boolean;
  reason: string;
}
/**
 * @name MsgValidateResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidateResponse
 */
export interface MsgValidateResponse {
  validationId: bigint;
  /**
   * True if this vote caused the counterexample to resolve.
   */
  resolved: boolean;
  /**
   * Final status if resolved.
   */
  status: CounterexampleStatus;
}
/**
 * @name MsgUpdateParams
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgProposeCounterexample(): MsgProposeCounterexample {
  return {
    author: "",
    factId: "",
    wrongClaim: "",
    reasoning: "",
    errorType: 0,
    violatedMethodologyIds: []
  };
}
/**
 * @name MsgProposeCounterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexample
 */
export const MsgProposeCounterexample = {
  typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
  encode(message: MsgProposeCounterexample, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.author !== "") {
      writer.uint32(10).string(message.author);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.wrongClaim !== "") {
      writer.uint32(26).string(message.wrongClaim);
    }
    if (message.reasoning !== "") {
      writer.uint32(34).string(message.reasoning);
    }
    if (message.errorType !== 0) {
      writer.uint32(40).int32(message.errorType);
    }
    for (const v of message.violatedMethodologyIds) {
      writer.uint32(50).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeCounterexample {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeCounterexample();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.author = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.wrongClaim = reader.string();
          break;
        case 4:
          message.reasoning = reader.string();
          break;
        case 5:
          message.errorType = reader.int32() as any;
          break;
        case 6:
          message.violatedMethodologyIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeCounterexample>): MsgProposeCounterexample {
    const message = createBaseMsgProposeCounterexample();
    message.author = object.author ?? "";
    message.factId = object.factId ?? "";
    message.wrongClaim = object.wrongClaim ?? "";
    message.reasoning = object.reasoning ?? "";
    message.errorType = object.errorType ?? 0;
    message.violatedMethodologyIds = object.violatedMethodologyIds?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgProposeCounterexampleResponse(): MsgProposeCounterexampleResponse {
  return {
    counterexampleId: ""
  };
}
/**
 * @name MsgProposeCounterexampleResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexampleResponse
 */
export const MsgProposeCounterexampleResponse = {
  typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexampleResponse",
  encode(message: MsgProposeCounterexampleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.counterexampleId !== "") {
      writer.uint32(10).string(message.counterexampleId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeCounterexampleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeCounterexampleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.counterexampleId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeCounterexampleResponse>): MsgProposeCounterexampleResponse {
    const message = createBaseMsgProposeCounterexampleResponse();
    message.counterexampleId = object.counterexampleId ?? "";
    return message;
  }
};
function createBaseMsgValidate(): MsgValidate {
  return {
    validator: "",
    counterexampleId: "",
    affirm: false,
    reason: ""
  };
}
/**
 * @name MsgValidate
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidate
 */
export const MsgValidate = {
  typeUrl: "/zerone.counterexamples.v1.MsgValidate",
  encode(message: MsgValidate, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.counterexampleId !== "") {
      writer.uint32(18).string(message.counterexampleId);
    }
    if (message.affirm === true) {
      writer.uint32(24).bool(message.affirm);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgValidate {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgValidate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.counterexampleId = reader.string();
          break;
        case 3:
          message.affirm = reader.bool();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgValidate>): MsgValidate {
    const message = createBaseMsgValidate();
    message.validator = object.validator ?? "";
    message.counterexampleId = object.counterexampleId ?? "";
    message.affirm = object.affirm ?? false;
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgValidateResponse(): MsgValidateResponse {
  return {
    validationId: BigInt(0),
    resolved: false,
    status: 0
  };
}
/**
 * @name MsgValidateResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidateResponse
 */
export const MsgValidateResponse = {
  typeUrl: "/zerone.counterexamples.v1.MsgValidateResponse",
  encode(message: MsgValidateResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validationId !== BigInt(0)) {
      writer.uint32(8).uint64(message.validationId);
    }
    if (message.resolved === true) {
      writer.uint32(16).bool(message.resolved);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgValidateResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgValidateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validationId = reader.uint64();
          break;
        case 2:
          message.resolved = reader.bool();
          break;
        case 3:
          message.status = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgValidateResponse>): MsgValidateResponse {
    const message = createBaseMsgValidateResponse();
    message.validationId = object.validationId !== undefined && object.validationId !== null ? BigInt(object.validationId.toString()) : BigInt(0);
    message.resolved = object.resolved ?? false;
    message.status = object.status ?? 0;
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
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
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
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.counterexamples.v1.MsgUpdateParamsResponse",
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