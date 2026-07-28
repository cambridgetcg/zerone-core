//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * RateLimit defines a per-(channel, denom) flow cap with a sliding window.
 * @name RateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.RateLimit
 */
export interface RateLimit {
  channelId: string;
  denom: string;
  /**
   * bigint string (uzrn)
   */
  maxSend: string;
  /**
   * bigint string (uzrn)
   */
  maxRecv: string;
  windowBlocks: bigint;
  /**
   * bigint string — accumulated in current window
   */
  currentSend: string;
  /**
   * bigint string — accumulated in current window
   */
  currentRecv: string;
  /**
   * block height at which the current window began
   */
  windowStart: bigint;
}
/**
 * PacketFlow records an outbound packet for quota reversal on timeout/error ack.
 * @name PacketFlow
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.PacketFlow
 */
export interface PacketFlow {
  channelId: string;
  sequence: bigint;
  denom: string;
  /**
   * bigint string
   */
  amount: string;
}
function createBaseRateLimit(): RateLimit {
  return {
    channelId: "",
    denom: "",
    maxSend: "",
    maxRecv: "",
    windowBlocks: BigInt(0),
    currentSend: "",
    currentRecv: "",
    windowStart: BigInt(0)
  };
}
/**
 * RateLimit defines a per-(channel, denom) flow cap with a sliding window.
 * @name RateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.RateLimit
 */
export const RateLimit = {
  typeUrl: "/zerone.ibcratelimit.v1.RateLimit",
  encode(message: RateLimit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.channelId !== "") {
      writer.uint32(10).string(message.channelId);
    }
    if (message.denom !== "") {
      writer.uint32(18).string(message.denom);
    }
    if (message.maxSend !== "") {
      writer.uint32(26).string(message.maxSend);
    }
    if (message.maxRecv !== "") {
      writer.uint32(34).string(message.maxRecv);
    }
    if (message.windowBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.windowBlocks);
    }
    if (message.currentSend !== "") {
      writer.uint32(50).string(message.currentSend);
    }
    if (message.currentRecv !== "") {
      writer.uint32(58).string(message.currentRecv);
    }
    if (message.windowStart !== BigInt(0)) {
      writer.uint32(64).uint64(message.windowStart);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RateLimit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRateLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.channelId = reader.string();
          break;
        case 2:
          message.denom = reader.string();
          break;
        case 3:
          message.maxSend = reader.string();
          break;
        case 4:
          message.maxRecv = reader.string();
          break;
        case 5:
          message.windowBlocks = reader.uint64();
          break;
        case 6:
          message.currentSend = reader.string();
          break;
        case 7:
          message.currentRecv = reader.string();
          break;
        case 8:
          message.windowStart = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RateLimit>): RateLimit {
    const message = createBaseRateLimit();
    message.channelId = object.channelId ?? "";
    message.denom = object.denom ?? "";
    message.maxSend = object.maxSend ?? "";
    message.maxRecv = object.maxRecv ?? "";
    message.windowBlocks = object.windowBlocks !== undefined && object.windowBlocks !== null ? BigInt(object.windowBlocks.toString()) : BigInt(0);
    message.currentSend = object.currentSend ?? "";
    message.currentRecv = object.currentRecv ?? "";
    message.windowStart = object.windowStart !== undefined && object.windowStart !== null ? BigInt(object.windowStart.toString()) : BigInt(0);
    return message;
  }
};
function createBasePacketFlow(): PacketFlow {
  return {
    channelId: "",
    sequence: BigInt(0),
    denom: "",
    amount: ""
  };
}
/**
 * PacketFlow records an outbound packet for quota reversal on timeout/error ack.
 * @name PacketFlow
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.PacketFlow
 */
export const PacketFlow = {
  typeUrl: "/zerone.ibcratelimit.v1.PacketFlow",
  encode(message: PacketFlow, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.channelId !== "") {
      writer.uint32(10).string(message.channelId);
    }
    if (message.sequence !== BigInt(0)) {
      writer.uint32(16).uint64(message.sequence);
    }
    if (message.denom !== "") {
      writer.uint32(26).string(message.denom);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PacketFlow {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePacketFlow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.channelId = reader.string();
          break;
        case 2:
          message.sequence = reader.uint64();
          break;
        case 3:
          message.denom = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PacketFlow>): PacketFlow {
    const message = createBasePacketFlow();
    message.channelId = object.channelId ?? "";
    message.sequence = object.sequence !== undefined && object.sequence !== null ? BigInt(object.sequence.toString()) : BigInt(0);
    message.denom = object.denom ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};