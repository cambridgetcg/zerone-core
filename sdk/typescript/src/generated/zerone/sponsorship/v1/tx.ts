//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgCreateBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrder
 */
export interface MsgCreateBountyOrder {
  sponsor: string;
  domain: string;
  pricePerArtifact: string;
  targetCount: number;
  durationBlocks: bigint;
}
/**
 * @name MsgCreateBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrderResponse
 */
export interface MsgCreateBountyOrderResponse {
  bountyId: string;
}
/**
 * @name MsgFulfillBounty
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBounty
 */
export interface MsgFulfillBounty {
  caller: string;
  bountyId: string;
  factId: string;
}
/**
 * @name MsgFulfillBountyResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBountyResponse
 */
export interface MsgFulfillBountyResponse {
  worker: string;
  amountPaid: string;
  bountyNowFulfilled: boolean;
}
/**
 * @name MsgCancelBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrder
 */
export interface MsgCancelBountyOrder {
  sponsor: string;
  bountyId: string;
}
/**
 * @name MsgCancelBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrderResponse
 */
export interface MsgCancelBountyOrderResponse {
  refundedAmount: string;
}
function createBaseMsgCreateBountyOrder(): MsgCreateBountyOrder {
  return {
    sponsor: "",
    domain: "",
    pricePerArtifact: "",
    targetCount: 0,
    durationBlocks: BigInt(0)
  };
}
/**
 * @name MsgCreateBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrder
 */
export const MsgCreateBountyOrder = {
  typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
  encode(message: MsgCreateBountyOrder, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.pricePerArtifact !== "") {
      writer.uint32(26).string(message.pricePerArtifact);
    }
    if (message.targetCount !== 0) {
      writer.uint32(32).uint32(message.targetCount);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.durationBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateBountyOrder {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateBountyOrder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.pricePerArtifact = reader.string();
          break;
        case 4:
          message.targetCount = reader.uint32();
          break;
        case 5:
          message.durationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateBountyOrder>): MsgCreateBountyOrder {
    const message = createBaseMsgCreateBountyOrder();
    message.sponsor = object.sponsor ?? "";
    message.domain = object.domain ?? "";
    message.pricePerArtifact = object.pricePerArtifact ?? "";
    message.targetCount = object.targetCount ?? 0;
    message.durationBlocks = object.durationBlocks !== undefined && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateBountyOrderResponse(): MsgCreateBountyOrderResponse {
  return {
    bountyId: ""
  };
}
/**
 * @name MsgCreateBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrderResponse
 */
export const MsgCreateBountyOrderResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrderResponse",
  encode(message: MsgCreateBountyOrderResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.bountyId !== "") {
      writer.uint32(10).string(message.bountyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateBountyOrderResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateBountyOrderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bountyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateBountyOrderResponse>): MsgCreateBountyOrderResponse {
    const message = createBaseMsgCreateBountyOrderResponse();
    message.bountyId = object.bountyId ?? "";
    return message;
  }
};
function createBaseMsgFulfillBounty(): MsgFulfillBounty {
  return {
    caller: "",
    bountyId: "",
    factId: ""
  };
}
/**
 * @name MsgFulfillBounty
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBounty
 */
export const MsgFulfillBounty = {
  typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
  encode(message: MsgFulfillBounty, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.caller !== "") {
      writer.uint32(10).string(message.caller);
    }
    if (message.bountyId !== "") {
      writer.uint32(18).string(message.bountyId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFulfillBounty {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFulfillBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.caller = reader.string();
          break;
        case 2:
          message.bountyId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFulfillBounty>): MsgFulfillBounty {
    const message = createBaseMsgFulfillBounty();
    message.caller = object.caller ?? "";
    message.bountyId = object.bountyId ?? "";
    message.factId = object.factId ?? "";
    return message;
  }
};
function createBaseMsgFulfillBountyResponse(): MsgFulfillBountyResponse {
  return {
    worker: "",
    amountPaid: "",
    bountyNowFulfilled: false
  };
}
/**
 * @name MsgFulfillBountyResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBountyResponse
 */
export const MsgFulfillBountyResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgFulfillBountyResponse",
  encode(message: MsgFulfillBountyResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.worker !== "") {
      writer.uint32(10).string(message.worker);
    }
    if (message.amountPaid !== "") {
      writer.uint32(18).string(message.amountPaid);
    }
    if (message.bountyNowFulfilled === true) {
      writer.uint32(24).bool(message.bountyNowFulfilled);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFulfillBountyResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFulfillBountyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.worker = reader.string();
          break;
        case 2:
          message.amountPaid = reader.string();
          break;
        case 3:
          message.bountyNowFulfilled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFulfillBountyResponse>): MsgFulfillBountyResponse {
    const message = createBaseMsgFulfillBountyResponse();
    message.worker = object.worker ?? "";
    message.amountPaid = object.amountPaid ?? "";
    message.bountyNowFulfilled = object.bountyNowFulfilled ?? false;
    return message;
  }
};
function createBaseMsgCancelBountyOrder(): MsgCancelBountyOrder {
  return {
    sponsor: "",
    bountyId: ""
  };
}
/**
 * @name MsgCancelBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrder
 */
export const MsgCancelBountyOrder = {
  typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
  encode(message: MsgCancelBountyOrder, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.bountyId !== "") {
      writer.uint32(18).string(message.bountyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelBountyOrder {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelBountyOrder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.bountyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCancelBountyOrder>): MsgCancelBountyOrder {
    const message = createBaseMsgCancelBountyOrder();
    message.sponsor = object.sponsor ?? "";
    message.bountyId = object.bountyId ?? "";
    return message;
  }
};
function createBaseMsgCancelBountyOrderResponse(): MsgCancelBountyOrderResponse {
  return {
    refundedAmount: ""
  };
}
/**
 * @name MsgCancelBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrderResponse
 */
export const MsgCancelBountyOrderResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrderResponse",
  encode(message: MsgCancelBountyOrderResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.refundedAmount !== "") {
      writer.uint32(10).string(message.refundedAmount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelBountyOrderResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelBountyOrderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.refundedAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCancelBountyOrderResponse>): MsgCancelBountyOrderResponse {
    const message = createBaseMsgCancelBountyOrderResponse();
    message.refundedAmount = object.refundedAmount ?? "";
    return message;
  }
};