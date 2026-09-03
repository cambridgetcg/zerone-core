//@ts-nocheck
import { Params } from "./state";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgCreateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateSchedule
 */
export interface MsgCreateSchedule {
  creator: string;
  recipient: string;
  amountPerExecutionUzrn: string;
  firstExecutionHeight: bigint;
  intervalBlocks: bigint;
  executionCount: number;
}
/**
 * @name MsgCreateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateScheduleResponse
 */
export interface MsgCreateScheduleResponse {
  scheduleId: string;
  escrowedUzrn: string;
}
/**
 * UpdateSchedule replaces all not-yet-executed terms. The two expected values
 * provide compare-and-swap protection against an occurrence executing or a
 * prior amendment committing before this transaction.
 * @name MsgUpdateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateSchedule
 */
export interface MsgUpdateSchedule {
  creator: string;
  scheduleId: string;
  expectedRevision: bigint;
  expectedExecutionCount: number;
  recipient: string;
  amountPerExecutionUzrn: string;
  nextExecutionHeight: bigint;
  intervalBlocks: bigint;
  remainingExecutions: number;
}
/**
 * @name MsgUpdateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateScheduleResponse
 */
export interface MsgUpdateScheduleResponse {
  revision: bigint;
  escrowDeltaUzrn: string;
  refunded: boolean;
}
/**
 * @name MsgCancelSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelSchedule
 */
export interface MsgCancelSchedule {
  creator: string;
  scheduleId: string;
  expectedRevision: bigint;
  expectedExecutionCount: number;
}
/**
 * @name MsgCancelScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelScheduleResponse
 */
export interface MsgCancelScheduleResponse {
  refundedUzrn: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgCreateSchedule(): MsgCreateSchedule {
  return {
    creator: "",
    recipient: "",
    amountPerExecutionUzrn: "",
    firstExecutionHeight: BigInt(0),
    intervalBlocks: BigInt(0),
    executionCount: 0
  };
}
/**
 * @name MsgCreateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateSchedule
 */
export const MsgCreateSchedule = {
  typeUrl: "/zerone.schedule.v2.MsgCreateSchedule",
  encode(message: MsgCreateSchedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.recipient !== "") {
      writer.uint32(18).string(message.recipient);
    }
    if (message.amountPerExecutionUzrn !== "") {
      writer.uint32(26).string(message.amountPerExecutionUzrn);
    }
    if (message.firstExecutionHeight !== BigInt(0)) {
      writer.uint32(32).uint64(message.firstExecutionHeight);
    }
    if (message.intervalBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.intervalBlocks);
    }
    if (message.executionCount !== 0) {
      writer.uint32(48).uint32(message.executionCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateSchedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.recipient = reader.string();
          break;
        case 3:
          message.amountPerExecutionUzrn = reader.string();
          break;
        case 4:
          message.firstExecutionHeight = reader.uint64();
          break;
        case 5:
          message.intervalBlocks = reader.uint64();
          break;
        case 6:
          message.executionCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateSchedule>): MsgCreateSchedule {
    const message = createBaseMsgCreateSchedule();
    message.creator = object.creator ?? "";
    message.recipient = object.recipient ?? "";
    message.amountPerExecutionUzrn = object.amountPerExecutionUzrn ?? "";
    message.firstExecutionHeight = object.firstExecutionHeight !== undefined && object.firstExecutionHeight !== null ? BigInt(object.firstExecutionHeight.toString()) : BigInt(0);
    message.intervalBlocks = object.intervalBlocks !== undefined && object.intervalBlocks !== null ? BigInt(object.intervalBlocks.toString()) : BigInt(0);
    message.executionCount = object.executionCount ?? 0;
    return message;
  }
};
function createBaseMsgCreateScheduleResponse(): MsgCreateScheduleResponse {
  return {
    scheduleId: "",
    escrowedUzrn: ""
  };
}
/**
 * @name MsgCreateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateScheduleResponse
 */
export const MsgCreateScheduleResponse = {
  typeUrl: "/zerone.schedule.v2.MsgCreateScheduleResponse",
  encode(message: MsgCreateScheduleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.scheduleId !== "") {
      writer.uint32(10).string(message.scheduleId);
    }
    if (message.escrowedUzrn !== "") {
      writer.uint32(18).string(message.escrowedUzrn);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateScheduleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateScheduleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.scheduleId = reader.string();
          break;
        case 2:
          message.escrowedUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateScheduleResponse>): MsgCreateScheduleResponse {
    const message = createBaseMsgCreateScheduleResponse();
    message.scheduleId = object.scheduleId ?? "";
    message.escrowedUzrn = object.escrowedUzrn ?? "";
    return message;
  }
};
function createBaseMsgUpdateSchedule(): MsgUpdateSchedule {
  return {
    creator: "",
    scheduleId: "",
    expectedRevision: BigInt(0),
    expectedExecutionCount: 0,
    recipient: "",
    amountPerExecutionUzrn: "",
    nextExecutionHeight: BigInt(0),
    intervalBlocks: BigInt(0),
    remainingExecutions: 0
  };
}
/**
 * UpdateSchedule replaces all not-yet-executed terms. The two expected values
 * provide compare-and-swap protection against an occurrence executing or a
 * prior amendment committing before this transaction.
 * @name MsgUpdateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateSchedule
 */
export const MsgUpdateSchedule = {
  typeUrl: "/zerone.schedule.v2.MsgUpdateSchedule",
  encode(message: MsgUpdateSchedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.scheduleId !== "") {
      writer.uint32(18).string(message.scheduleId);
    }
    if (message.expectedRevision !== BigInt(0)) {
      writer.uint32(24).uint64(message.expectedRevision);
    }
    if (message.expectedExecutionCount !== 0) {
      writer.uint32(32).uint32(message.expectedExecutionCount);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    if (message.amountPerExecutionUzrn !== "") {
      writer.uint32(50).string(message.amountPerExecutionUzrn);
    }
    if (message.nextExecutionHeight !== BigInt(0)) {
      writer.uint32(56).uint64(message.nextExecutionHeight);
    }
    if (message.intervalBlocks !== BigInt(0)) {
      writer.uint32(64).uint64(message.intervalBlocks);
    }
    if (message.remainingExecutions !== 0) {
      writer.uint32(72).uint32(message.remainingExecutions);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateSchedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.scheduleId = reader.string();
          break;
        case 3:
          message.expectedRevision = reader.uint64();
          break;
        case 4:
          message.expectedExecutionCount = reader.uint32();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        case 6:
          message.amountPerExecutionUzrn = reader.string();
          break;
        case 7:
          message.nextExecutionHeight = reader.uint64();
          break;
        case 8:
          message.intervalBlocks = reader.uint64();
          break;
        case 9:
          message.remainingExecutions = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateSchedule>): MsgUpdateSchedule {
    const message = createBaseMsgUpdateSchedule();
    message.creator = object.creator ?? "";
    message.scheduleId = object.scheduleId ?? "";
    message.expectedRevision = object.expectedRevision !== undefined && object.expectedRevision !== null ? BigInt(object.expectedRevision.toString()) : BigInt(0);
    message.expectedExecutionCount = object.expectedExecutionCount ?? 0;
    message.recipient = object.recipient ?? "";
    message.amountPerExecutionUzrn = object.amountPerExecutionUzrn ?? "";
    message.nextExecutionHeight = object.nextExecutionHeight !== undefined && object.nextExecutionHeight !== null ? BigInt(object.nextExecutionHeight.toString()) : BigInt(0);
    message.intervalBlocks = object.intervalBlocks !== undefined && object.intervalBlocks !== null ? BigInt(object.intervalBlocks.toString()) : BigInt(0);
    message.remainingExecutions = object.remainingExecutions ?? 0;
    return message;
  }
};
function createBaseMsgUpdateScheduleResponse(): MsgUpdateScheduleResponse {
  return {
    revision: BigInt(0),
    escrowDeltaUzrn: "",
    refunded: false
  };
}
/**
 * @name MsgUpdateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateScheduleResponse
 */
export const MsgUpdateScheduleResponse = {
  typeUrl: "/zerone.schedule.v2.MsgUpdateScheduleResponse",
  encode(message: MsgUpdateScheduleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.revision !== BigInt(0)) {
      writer.uint32(8).uint64(message.revision);
    }
    if (message.escrowDeltaUzrn !== "") {
      writer.uint32(18).string(message.escrowDeltaUzrn);
    }
    if (message.refunded === true) {
      writer.uint32(24).bool(message.refunded);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateScheduleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateScheduleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.revision = reader.uint64();
          break;
        case 2:
          message.escrowDeltaUzrn = reader.string();
          break;
        case 3:
          message.refunded = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateScheduleResponse>): MsgUpdateScheduleResponse {
    const message = createBaseMsgUpdateScheduleResponse();
    message.revision = object.revision !== undefined && object.revision !== null ? BigInt(object.revision.toString()) : BigInt(0);
    message.escrowDeltaUzrn = object.escrowDeltaUzrn ?? "";
    message.refunded = object.refunded ?? false;
    return message;
  }
};
function createBaseMsgCancelSchedule(): MsgCancelSchedule {
  return {
    creator: "",
    scheduleId: "",
    expectedRevision: BigInt(0),
    expectedExecutionCount: 0
  };
}
/**
 * @name MsgCancelSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelSchedule
 */
export const MsgCancelSchedule = {
  typeUrl: "/zerone.schedule.v2.MsgCancelSchedule",
  encode(message: MsgCancelSchedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.scheduleId !== "") {
      writer.uint32(18).string(message.scheduleId);
    }
    if (message.expectedRevision !== BigInt(0)) {
      writer.uint32(24).uint64(message.expectedRevision);
    }
    if (message.expectedExecutionCount !== 0) {
      writer.uint32(32).uint32(message.expectedExecutionCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelSchedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.scheduleId = reader.string();
          break;
        case 3:
          message.expectedRevision = reader.uint64();
          break;
        case 4:
          message.expectedExecutionCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCancelSchedule>): MsgCancelSchedule {
    const message = createBaseMsgCancelSchedule();
    message.creator = object.creator ?? "";
    message.scheduleId = object.scheduleId ?? "";
    message.expectedRevision = object.expectedRevision !== undefined && object.expectedRevision !== null ? BigInt(object.expectedRevision.toString()) : BigInt(0);
    message.expectedExecutionCount = object.expectedExecutionCount ?? 0;
    return message;
  }
};
function createBaseMsgCancelScheduleResponse(): MsgCancelScheduleResponse {
  return {
    refundedUzrn: ""
  };
}
/**
 * @name MsgCancelScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelScheduleResponse
 */
export const MsgCancelScheduleResponse = {
  typeUrl: "/zerone.schedule.v2.MsgCancelScheduleResponse",
  encode(message: MsgCancelScheduleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.refundedUzrn !== "") {
      writer.uint32(10).string(message.refundedUzrn);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelScheduleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelScheduleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.refundedUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCancelScheduleResponse>): MsgCancelScheduleResponse {
    const message = createBaseMsgCancelScheduleResponse();
    message.refundedUzrn = object.refundedUzrn ?? "";
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
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.schedule.v2.MsgUpdateParams",
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
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.schedule.v2.MsgUpdateParamsResponse",
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