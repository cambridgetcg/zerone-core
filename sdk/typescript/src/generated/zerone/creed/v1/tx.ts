//@ts-nocheck
import { PinnedCreed, CreedCouncilMember } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgAnchorPin
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPin
 */
export interface MsgAnchorPin {
  authority: string;
  /**
   * The new pinned creed. version MUST equal currentPin.version+1.
   * canonical_hash MUST be non-empty. commitments MUST satisfy:
   *   - all numbers unique
   *   - numbers are 1..N for some N (no gaps unless the missing
   *     number's prior entry is archived in this same version)
   *   - introduced_at_height ≤ block_height for all entries
   */
  pin?: PinnedCreed;
  /**
   * The LIP id this pin came from. Required when params.
   * direct_anchor_enabled is false. Empty allowed only for
   * genesis-equivalent pre-launch pins.
   */
  sourceLip: string;
}
/**
 * @name MsgAnchorPinResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPinResponse
 */
export interface MsgAnchorPinResponse {
  newVersion: number;
}
/**
 * @name MsgUpdateParams
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
/**
 * @name MsgUpdateCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMember
 */
export interface MsgUpdateCouncilMember {
  authority: string;
  /**
   * The seat to upsert. address is the key; existing seats with
   * the same address are updated in place (their admitted_at_height
   * is preserved if unchanged).
   */
  member?: CreedCouncilMember;
  /**
   * LIP id that authorized this change. Required when params.
   * direct_anchor_enabled is false. Empty allowed only for
   * pre-launch council seeding outside of gov.
   */
  sourceLip: string;
}
/**
 * @name MsgUpdateCouncilMemberResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMemberResponse
 */
export interface MsgUpdateCouncilMemberResponse {}
function createBaseMsgAnchorPin(): MsgAnchorPin {
  return {
    authority: "",
    pin: undefined,
    sourceLip: ""
  };
}
/**
 * @name MsgAnchorPin
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPin
 */
export const MsgAnchorPin = {
  typeUrl: "/zerone.creed.v1.MsgAnchorPin",
  encode(message: MsgAnchorPin, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.pin !== undefined) {
      PinnedCreed.encode(message.pin, writer.uint32(18).fork()).ldelim();
    }
    if (message.sourceLip !== "") {
      writer.uint32(26).string(message.sourceLip);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAnchorPin {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAnchorPin();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.pin = PinnedCreed.decode(reader, reader.uint32());
          break;
        case 3:
          message.sourceLip = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAnchorPin>): MsgAnchorPin {
    const message = createBaseMsgAnchorPin();
    message.authority = object.authority ?? "";
    message.pin = object.pin !== undefined && object.pin !== null ? PinnedCreed.fromPartial(object.pin) : undefined;
    message.sourceLip = object.sourceLip ?? "";
    return message;
  }
};
function createBaseMsgAnchorPinResponse(): MsgAnchorPinResponse {
  return {
    newVersion: 0
  };
}
/**
 * @name MsgAnchorPinResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPinResponse
 */
export const MsgAnchorPinResponse = {
  typeUrl: "/zerone.creed.v1.MsgAnchorPinResponse",
  encode(message: MsgAnchorPinResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newVersion !== 0) {
      writer.uint32(8).uint32(message.newVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAnchorPinResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAnchorPinResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAnchorPinResponse>): MsgAnchorPinResponse {
    const message = createBaseMsgAnchorPinResponse();
    message.newVersion = object.newVersion ?? 0;
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
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.creed.v1.MsgUpdateParams",
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
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.creed.v1.MsgUpdateParamsResponse",
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
function createBaseMsgUpdateCouncilMember(): MsgUpdateCouncilMember {
  return {
    authority: "",
    member: undefined,
    sourceLip: ""
  };
}
/**
 * @name MsgUpdateCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMember
 */
export const MsgUpdateCouncilMember = {
  typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
  encode(message: MsgUpdateCouncilMember, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.member !== undefined) {
      CreedCouncilMember.encode(message.member, writer.uint32(18).fork()).ldelim();
    }
    if (message.sourceLip !== "") {
      writer.uint32(26).string(message.sourceLip);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateCouncilMember {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateCouncilMember();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.member = CreedCouncilMember.decode(reader, reader.uint32());
          break;
        case 3:
          message.sourceLip = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateCouncilMember>): MsgUpdateCouncilMember {
    const message = createBaseMsgUpdateCouncilMember();
    message.authority = object.authority ?? "";
    message.member = object.member !== undefined && object.member !== null ? CreedCouncilMember.fromPartial(object.member) : undefined;
    message.sourceLip = object.sourceLip ?? "";
    return message;
  }
};
function createBaseMsgUpdateCouncilMemberResponse(): MsgUpdateCouncilMemberResponse {
  return {};
}
/**
 * @name MsgUpdateCouncilMemberResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMemberResponse
 */
export const MsgUpdateCouncilMemberResponse = {
  typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMemberResponse",
  encode(_: MsgUpdateCouncilMemberResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateCouncilMemberResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateCouncilMemberResponse();
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
  fromPartial(_: DeepPartial<MsgUpdateCouncilMemberResponse>): MsgUpdateCouncilMemberResponse {
    const message = createBaseMsgUpdateCouncilMemberResponse();
    return message;
  }
};