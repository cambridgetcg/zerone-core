//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgQualifyByStake
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStake
 */
export interface MsgQualifyByStake {
  validator: string;
  domain: string;
  /**
   * uzrn
   */
  stakeAmount: string;
}
/**
 * @name MsgQualifyByStakeResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStakeResponse
 */
export interface MsgQualifyByStakeResponse {}
/**
 * @name MsgQualifyByTrackRecord
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecord
 */
export interface MsgQualifyByTrackRecord {
  validator: string;
  domain: string;
}
/**
 * @name MsgQualifyByTrackRecordResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecordResponse
 */
export interface MsgQualifyByTrackRecordResponse {}
/**
 * @name MsgQualifyByCrossReference
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReference
 */
export interface MsgQualifyByCrossReference {
  validator: string;
  targetDomain: string;
  sourceDomain: string;
}
/**
 * @name MsgQualifyByCrossReferenceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReferenceResponse
 */
export interface MsgQualifyByCrossReferenceResponse {}
/**
 * @name MsgQualifyByInheritance
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritance
 */
export interface MsgQualifyByInheritance {
  validator: string;
  targetDomain: string;
  parentDomain: string;
}
/**
 * @name MsgQualifyByInheritanceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritanceResponse
 */
export interface MsgQualifyByInheritanceResponse {}
/**
 * @name MsgEndorseQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualification
 */
export interface MsgEndorseQualification {
  endorser: string;
  validator: string;
  domain: string;
  reason: string;
  weight: number;
}
/**
 * @name MsgEndorseQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualificationResponse
 */
export interface MsgEndorseQualificationResponse {
  endorsementId: bigint;
}
/**
 * @name MsgRenewQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualification
 */
export interface MsgRenewQualification {
  validator: string;
  domain: string;
}
/**
 * @name MsgRenewQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualificationResponse
 */
export interface MsgRenewQualificationResponse {}
/**
 * @name MsgWithdrawQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualification
 */
export interface MsgWithdrawQualification {
  validator: string;
  domain: string;
}
/**
 * @name MsgWithdrawQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualificationResponse
 */
export interface MsgWithdrawQualificationResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgQualifyByStake(): MsgQualifyByStake {
  return {
    validator: "",
    domain: "",
    stakeAmount: ""
  };
}
/**
 * @name MsgQualifyByStake
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStake
 */
export const MsgQualifyByStake = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
  encode(message: MsgQualifyByStake, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.stakeAmount !== "") {
      writer.uint32(26).string(message.stakeAmount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByStake {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByStake();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.stakeAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgQualifyByStake>): MsgQualifyByStake {
    const message = createBaseMsgQualifyByStake();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.stakeAmount = object.stakeAmount ?? "";
    return message;
  }
};
function createBaseMsgQualifyByStakeResponse(): MsgQualifyByStakeResponse {
  return {};
}
/**
 * @name MsgQualifyByStakeResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStakeResponse
 */
export const MsgQualifyByStakeResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByStakeResponse",
  encode(_: MsgQualifyByStakeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByStakeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByStakeResponse();
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
  fromPartial(_: DeepPartial<MsgQualifyByStakeResponse>): MsgQualifyByStakeResponse {
    const message = createBaseMsgQualifyByStakeResponse();
    return message;
  }
};
function createBaseMsgQualifyByTrackRecord(): MsgQualifyByTrackRecord {
  return {
    validator: "",
    domain: ""
  };
}
/**
 * @name MsgQualifyByTrackRecord
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecord
 */
export const MsgQualifyByTrackRecord = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
  encode(message: MsgQualifyByTrackRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByTrackRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByTrackRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
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
  fromPartial(object: DeepPartial<MsgQualifyByTrackRecord>): MsgQualifyByTrackRecord {
    const message = createBaseMsgQualifyByTrackRecord();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByTrackRecordResponse(): MsgQualifyByTrackRecordResponse {
  return {};
}
/**
 * @name MsgQualifyByTrackRecordResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecordResponse
 */
export const MsgQualifyByTrackRecordResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecordResponse",
  encode(_: MsgQualifyByTrackRecordResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByTrackRecordResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByTrackRecordResponse();
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
  fromPartial(_: DeepPartial<MsgQualifyByTrackRecordResponse>): MsgQualifyByTrackRecordResponse {
    const message = createBaseMsgQualifyByTrackRecordResponse();
    return message;
  }
};
function createBaseMsgQualifyByCrossReference(): MsgQualifyByCrossReference {
  return {
    validator: "",
    targetDomain: "",
    sourceDomain: ""
  };
}
/**
 * @name MsgQualifyByCrossReference
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReference
 */
export const MsgQualifyByCrossReference = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
  encode(message: MsgQualifyByCrossReference, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.targetDomain !== "") {
      writer.uint32(18).string(message.targetDomain);
    }
    if (message.sourceDomain !== "") {
      writer.uint32(26).string(message.sourceDomain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByCrossReference {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByCrossReference();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.targetDomain = reader.string();
          break;
        case 3:
          message.sourceDomain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgQualifyByCrossReference>): MsgQualifyByCrossReference {
    const message = createBaseMsgQualifyByCrossReference();
    message.validator = object.validator ?? "";
    message.targetDomain = object.targetDomain ?? "";
    message.sourceDomain = object.sourceDomain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByCrossReferenceResponse(): MsgQualifyByCrossReferenceResponse {
  return {};
}
/**
 * @name MsgQualifyByCrossReferenceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReferenceResponse
 */
export const MsgQualifyByCrossReferenceResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReferenceResponse",
  encode(_: MsgQualifyByCrossReferenceResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByCrossReferenceResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByCrossReferenceResponse();
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
  fromPartial(_: DeepPartial<MsgQualifyByCrossReferenceResponse>): MsgQualifyByCrossReferenceResponse {
    const message = createBaseMsgQualifyByCrossReferenceResponse();
    return message;
  }
};
function createBaseMsgQualifyByInheritance(): MsgQualifyByInheritance {
  return {
    validator: "",
    targetDomain: "",
    parentDomain: ""
  };
}
/**
 * @name MsgQualifyByInheritance
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritance
 */
export const MsgQualifyByInheritance = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
  encode(message: MsgQualifyByInheritance, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.targetDomain !== "") {
      writer.uint32(18).string(message.targetDomain);
    }
    if (message.parentDomain !== "") {
      writer.uint32(26).string(message.parentDomain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByInheritance {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByInheritance();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.targetDomain = reader.string();
          break;
        case 3:
          message.parentDomain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgQualifyByInheritance>): MsgQualifyByInheritance {
    const message = createBaseMsgQualifyByInheritance();
    message.validator = object.validator ?? "";
    message.targetDomain = object.targetDomain ?? "";
    message.parentDomain = object.parentDomain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByInheritanceResponse(): MsgQualifyByInheritanceResponse {
  return {};
}
/**
 * @name MsgQualifyByInheritanceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritanceResponse
 */
export const MsgQualifyByInheritanceResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritanceResponse",
  encode(_: MsgQualifyByInheritanceResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByInheritanceResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByInheritanceResponse();
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
  fromPartial(_: DeepPartial<MsgQualifyByInheritanceResponse>): MsgQualifyByInheritanceResponse {
    const message = createBaseMsgQualifyByInheritanceResponse();
    return message;
  }
};
function createBaseMsgEndorseQualification(): MsgEndorseQualification {
  return {
    endorser: "",
    validator: "",
    domain: "",
    reason: "",
    weight: 0
  };
}
/**
 * @name MsgEndorseQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualification
 */
export const MsgEndorseQualification = {
  typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
  encode(message: MsgEndorseQualification, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.endorser !== "") {
      writer.uint32(10).string(message.endorser);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    if (message.weight !== 0) {
      writer.uint32(40).uint32(message.weight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseQualification {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorser = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        case 5:
          message.weight = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgEndorseQualification>): MsgEndorseQualification {
    const message = createBaseMsgEndorseQualification();
    message.endorser = object.endorser ?? "";
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.reason = object.reason ?? "";
    message.weight = object.weight ?? 0;
    return message;
  }
};
function createBaseMsgEndorseQualificationResponse(): MsgEndorseQualificationResponse {
  return {
    endorsementId: BigInt(0)
  };
}
/**
 * @name MsgEndorseQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualificationResponse
 */
export const MsgEndorseQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgEndorseQualificationResponse",
  encode(message: MsgEndorseQualificationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.endorsementId !== BigInt(0)) {
      writer.uint32(8).uint64(message.endorsementId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseQualificationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseQualificationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorsementId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgEndorseQualificationResponse>): MsgEndorseQualificationResponse {
    const message = createBaseMsgEndorseQualificationResponse();
    message.endorsementId = object.endorsementId !== undefined && object.endorsementId !== null ? BigInt(object.endorsementId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRenewQualification(): MsgRenewQualification {
  return {
    validator: "",
    domain: ""
  };
}
/**
 * @name MsgRenewQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualification
 */
export const MsgRenewQualification = {
  typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
  encode(message: MsgRenewQualification, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRenewQualification {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRenewQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
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
  fromPartial(object: DeepPartial<MsgRenewQualification>): MsgRenewQualification {
    const message = createBaseMsgRenewQualification();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgRenewQualificationResponse(): MsgRenewQualificationResponse {
  return {};
}
/**
 * @name MsgRenewQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualificationResponse
 */
export const MsgRenewQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgRenewQualificationResponse",
  encode(_: MsgRenewQualificationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRenewQualificationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRenewQualificationResponse();
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
  fromPartial(_: DeepPartial<MsgRenewQualificationResponse>): MsgRenewQualificationResponse {
    const message = createBaseMsgRenewQualificationResponse();
    return message;
  }
};
function createBaseMsgWithdrawQualification(): MsgWithdrawQualification {
  return {
    validator: "",
    domain: ""
  };
}
/**
 * @name MsgWithdrawQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualification
 */
export const MsgWithdrawQualification = {
  typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
  encode(message: MsgWithdrawQualification, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawQualification {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
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
  fromPartial(object: DeepPartial<MsgWithdrawQualification>): MsgWithdrawQualification {
    const message = createBaseMsgWithdrawQualification();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgWithdrawQualificationResponse(): MsgWithdrawQualificationResponse {
  return {};
}
/**
 * @name MsgWithdrawQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualificationResponse
 */
export const MsgWithdrawQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgWithdrawQualificationResponse",
  encode(_: MsgWithdrawQualificationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawQualificationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawQualificationResponse();
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
  fromPartial(_: DeepPartial<MsgWithdrawQualificationResponse>): MsgWithdrawQualificationResponse {
    const message = createBaseMsgWithdrawQualificationResponse();
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
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
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
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.qualification.v1.MsgUpdateParamsResponse",
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