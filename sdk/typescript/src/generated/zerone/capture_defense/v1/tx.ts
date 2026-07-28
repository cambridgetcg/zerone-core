//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgRecordVerification
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerification
 */
export interface MsgRecordVerification {
  authority: string;
  domain: string;
  roundId: string;
  validators: string[];
  verdicts: boolean[];
  submitBlocks: bigint[];
}
/**
 * @name MsgRecordVerificationResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerificationResponse
 */
export interface MsgRecordVerificationResponse {}
/**
 * @name MsgAnalyzeDomain
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomain
 */
export interface MsgAnalyzeDomain {
  sender: string;
  domain: string;
}
/**
 * @name MsgAnalyzeDomainResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomainResponse
 */
export interface MsgAnalyzeDomainResponse {
  riskScore: bigint;
  flagged: boolean;
}
/**
 * @name MsgUpdateParams
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgRecordVerification(): MsgRecordVerification {
  return {
    authority: "",
    domain: "",
    roundId: "",
    validators: [],
    verdicts: [],
    submitBlocks: []
  };
}
/**
 * @name MsgRecordVerification
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerification
 */
export const MsgRecordVerification = {
  typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
  encode(message: MsgRecordVerification, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.roundId !== "") {
      writer.uint32(26).string(message.roundId);
    }
    for (const v of message.validators) {
      writer.uint32(34).string(v!);
    }
    writer.uint32(42).fork();
    for (const v of message.verdicts) {
      writer.bool(v);
    }
    writer.ldelim();
    writer.uint32(50).fork();
    for (const v of message.submitBlocks) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordVerification {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordVerification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.roundId = reader.string();
          break;
        case 4:
          message.validators.push(reader.string());
          break;
        case 5:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdicts.push(reader.bool());
            }
          } else {
            message.verdicts.push(reader.bool());
          }
          break;
        case 6:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.submitBlocks.push(reader.uint64());
            }
          } else {
            message.submitBlocks.push(reader.uint64());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRecordVerification>): MsgRecordVerification {
    const message = createBaseMsgRecordVerification();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.roundId = object.roundId ?? "";
    message.validators = object.validators?.map(e => e) || [];
    message.verdicts = object.verdicts?.map(e => e) || [];
    message.submitBlocks = object.submitBlocks?.map(e => BigInt(e.toString())) || [];
    return message;
  }
};
function createBaseMsgRecordVerificationResponse(): MsgRecordVerificationResponse {
  return {};
}
/**
 * @name MsgRecordVerificationResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerificationResponse
 */
export const MsgRecordVerificationResponse = {
  typeUrl: "/zerone.capture_defense.v1.MsgRecordVerificationResponse",
  encode(_: MsgRecordVerificationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordVerificationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordVerificationResponse();
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
  fromPartial(_: DeepPartial<MsgRecordVerificationResponse>): MsgRecordVerificationResponse {
    const message = createBaseMsgRecordVerificationResponse();
    return message;
  }
};
function createBaseMsgAnalyzeDomain(): MsgAnalyzeDomain {
  return {
    sender: "",
    domain: ""
  };
}
/**
 * @name MsgAnalyzeDomain
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomain
 */
export const MsgAnalyzeDomain = {
  typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
  encode(message: MsgAnalyzeDomain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAnalyzeDomain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAnalyzeDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAnalyzeDomain>): MsgAnalyzeDomain {
    const message = createBaseMsgAnalyzeDomain();
    message.sender = object.sender ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgAnalyzeDomainResponse(): MsgAnalyzeDomainResponse {
  return {
    riskScore: BigInt(0),
    flagged: false
  };
}
/**
 * @name MsgAnalyzeDomainResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomainResponse
 */
export const MsgAnalyzeDomainResponse = {
  typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomainResponse",
  encode(message: MsgAnalyzeDomainResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.riskScore !== BigInt(0)) {
      writer.uint32(8).uint64(message.riskScore);
    }
    if (message.flagged === true) {
      writer.uint32(16).bool(message.flagged);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAnalyzeDomainResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAnalyzeDomainResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.riskScore = reader.uint64();
          break;
        case 2:
          message.flagged = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAnalyzeDomainResponse>): MsgAnalyzeDomainResponse {
    const message = createBaseMsgAnalyzeDomainResponse();
    message.riskScore = object.riskScore !== undefined && object.riskScore !== null ? BigInt(object.riskScore.toString()) : BigInt(0);
    message.flagged = object.flagged ?? false;
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
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
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
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.capture_defense.v1.MsgUpdateParamsResponse",
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