//@ts-nocheck
import { ChallengeOutcome } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgSubmitChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallenge
 */
export interface MsgSubmitChallenge {
  challenger: string;
  domain: string;
  accusedValidators: string[];
  /**
   * uzrn
   */
  stake: string;
  reason: string;
}
/**
 * @name MsgSubmitChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallengeResponse
 */
export interface MsgSubmitChallengeResponse {
  challengeId: string;
}
/**
 * @name MsgAddEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidence
 */
export interface MsgAddEvidence {
  challenger: string;
  challengeId: string;
  description: string;
  dataHash: string;
}
/**
 * @name MsgAddEvidenceResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidenceResponse
 */
export interface MsgAddEvidenceResponse {}
/**
 * @name MsgResolveChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallenge
 */
export interface MsgResolveChallenge {
  authority: string;
  challengeId: string;
  outcome: ChallengeOutcome;
  reason: string;
}
/**
 * @name MsgResolveChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallengeResponse
 */
export interface MsgResolveChallengeResponse {}
/**
 * @name MsgFundBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPool
 */
export interface MsgFundBountyPool {
  sender: string;
  domain: string;
  /**
   * uzrn
   */
  amount: string;
}
/**
 * @name MsgFundBountyPoolResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPoolResponse
 */
export interface MsgFundBountyPoolResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgSubmitChallenge(): MsgSubmitChallenge {
  return {
    challenger: "",
    domain: "",
    accusedValidators: [],
    stake: "",
    reason: ""
  };
}
/**
 * @name MsgSubmitChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallenge
 */
export const MsgSubmitChallenge = {
  typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
  encode(message: MsgSubmitChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    for (const v of message.accusedValidators) {
      writer.uint32(26).string(v!);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.accusedValidators.push(reader.string());
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitChallenge>): MsgSubmitChallenge {
    const message = createBaseMsgSubmitChallenge();
    message.challenger = object.challenger ?? "";
    message.domain = object.domain ?? "";
    message.accusedValidators = object.accusedValidators?.map(e => e) || [];
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSubmitChallengeResponse(): MsgSubmitChallengeResponse {
  return {
    challengeId: ""
  };
}
/**
 * @name MsgSubmitChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgSubmitChallengeResponse
 */
export const MsgSubmitChallengeResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallengeResponse",
  encode(message: MsgSubmitChallengeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challengeId !== "") {
      writer.uint32(10).string(message.challengeId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitChallengeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitChallengeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challengeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitChallengeResponse>): MsgSubmitChallengeResponse {
    const message = createBaseMsgSubmitChallengeResponse();
    message.challengeId = object.challengeId ?? "";
    return message;
  }
};
function createBaseMsgAddEvidence(): MsgAddEvidence {
  return {
    challenger: "",
    challengeId: "",
    description: "",
    dataHash: ""
  };
}
/**
 * @name MsgAddEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidence
 */
export const MsgAddEvidence = {
  typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
  encode(message: MsgAddEvidence, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.dataHash !== "") {
      writer.uint32(34).string(message.dataHash);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddEvidence {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddEvidence();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.dataHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddEvidence>): MsgAddEvidence {
    const message = createBaseMsgAddEvidence();
    message.challenger = object.challenger ?? "";
    message.challengeId = object.challengeId ?? "";
    message.description = object.description ?? "";
    message.dataHash = object.dataHash ?? "";
    return message;
  }
};
function createBaseMsgAddEvidenceResponse(): MsgAddEvidenceResponse {
  return {};
}
/**
 * @name MsgAddEvidenceResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgAddEvidenceResponse
 */
export const MsgAddEvidenceResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidenceResponse",
  encode(_: MsgAddEvidenceResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddEvidenceResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddEvidenceResponse();
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
  fromPartial(_: DeepPartial<MsgAddEvidenceResponse>): MsgAddEvidenceResponse {
    const message = createBaseMsgAddEvidenceResponse();
    return message;
  }
};
function createBaseMsgResolveChallenge(): MsgResolveChallenge {
  return {
    authority: "",
    challengeId: "",
    outcome: 0,
    reason: ""
  };
}
/**
 * @name MsgResolveChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallenge
 */
export const MsgResolveChallenge = {
  typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
  encode(message: MsgResolveChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.outcome !== 0) {
      writer.uint32(24).int32(message.outcome);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.outcome = reader.int32() as any;
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
  fromPartial(object: DeepPartial<MsgResolveChallenge>): MsgResolveChallenge {
    const message = createBaseMsgResolveChallenge();
    message.authority = object.authority ?? "";
    message.challengeId = object.challengeId ?? "";
    message.outcome = object.outcome ?? 0;
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgResolveChallengeResponse(): MsgResolveChallengeResponse {
  return {};
}
/**
 * @name MsgResolveChallengeResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgResolveChallengeResponse
 */
export const MsgResolveChallengeResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallengeResponse",
  encode(_: MsgResolveChallengeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveChallengeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveChallengeResponse();
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
  fromPartial(_: DeepPartial<MsgResolveChallengeResponse>): MsgResolveChallengeResponse {
    const message = createBaseMsgResolveChallengeResponse();
    return message;
  }
};
function createBaseMsgFundBountyPool(): MsgFundBountyPool {
  return {
    sender: "",
    domain: "",
    amount: ""
  };
}
/**
 * @name MsgFundBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPool
 */
export const MsgFundBountyPool = {
  typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
  encode(message: MsgFundBountyPool, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFundBountyPool {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFundBountyPool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFundBountyPool>): MsgFundBountyPool {
    const message = createBaseMsgFundBountyPool();
    message.sender = object.sender ?? "";
    message.domain = object.domain ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgFundBountyPoolResponse(): MsgFundBountyPoolResponse {
  return {};
}
/**
 * @name MsgFundBountyPoolResponse
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgFundBountyPoolResponse
 */
export const MsgFundBountyPoolResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPoolResponse",
  encode(_: MsgFundBountyPoolResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFundBountyPoolResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFundBountyPoolResponse();
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
  fromPartial(_: DeepPartial<MsgFundBountyPoolResponse>): MsgFundBountyPoolResponse {
    const message = createBaseMsgFundBountyPoolResponse();
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
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
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
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParamsResponse",
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