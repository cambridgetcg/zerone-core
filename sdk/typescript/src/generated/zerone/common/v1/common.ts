//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * BasisPoints represents a value on a 1,000,000 scale (100% = 1,000,000).
 * Fields typed as this common message use that scale. Some legacy scalar
 * fields named *_bps declare a 10,000 scale in their own module contracts;
 * callers must follow the field-specific protobuf documentation.
 * @name BasisPoints
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.BasisPoints
 */
export interface BasisPoints {
  value: bigint;
}
/**
 * RevenueSplit defines how protocol revenue is allocated — governance-adjustable.
 * @name RevenueSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.RevenueSplit
 */
export interface RevenueSplit {
  /**
   * default 550,000 (55%)
   */
  contributorBps: bigint;
  /**
   * default 220,000 (22%)
   */
  protocolBps: bigint;
  /**
   * default  33,300 (3.33%)
   */
  researchBps: bigint;
  /**
   * default 196,700 (19.67%) — bug bounties, truth discovery, protocol development
   */
  developmentBps: bigint;
}
/**
 * ProtocolSubSplit defines how the protocol share is divided.
 * @name ProtocolSubSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.ProtocolSubSplit
 */
export interface ProtocolSubSplit {
  /**
   * default 500,000 (50% of protocol)
   */
  citationBps: bigint;
  /**
   * default 300,000 (30% of protocol)
   */
  verificationBps: bigint;
  /**
   * default 200,000 (20% of protocol)
   */
  treasuryBps: bigint;
}
function createBaseBasisPoints(): BasisPoints {
  return {
    value: BigInt(0)
  };
}
/**
 * BasisPoints represents a value on a 1,000,000 scale (100% = 1,000,000).
 * Fields typed as this common message use that scale. Some legacy scalar
 * fields named *_bps declare a 10,000 scale in their own module contracts;
 * callers must follow the field-specific protobuf documentation.
 * @name BasisPoints
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.BasisPoints
 */
export const BasisPoints = {
  typeUrl: "/zerone.common.v1.BasisPoints",
  encode(message: BasisPoints, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.value !== BigInt(0)) {
      writer.uint32(8).uint64(message.value);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): BasisPoints {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBasisPoints();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.value = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<BasisPoints>): BasisPoints {
    const message = createBaseBasisPoints();
    message.value = object.value !== undefined && object.value !== null ? BigInt(object.value.toString()) : BigInt(0);
    return message;
  }
};
function createBaseRevenueSplit(): RevenueSplit {
  return {
    contributorBps: BigInt(0),
    protocolBps: BigInt(0),
    researchBps: BigInt(0),
    developmentBps: BigInt(0)
  };
}
/**
 * RevenueSplit defines how protocol revenue is allocated — governance-adjustable.
 * @name RevenueSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.RevenueSplit
 */
export const RevenueSplit = {
  typeUrl: "/zerone.common.v1.RevenueSplit",
  encode(message: RevenueSplit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.contributorBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.contributorBps);
    }
    if (message.protocolBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.protocolBps);
    }
    if (message.researchBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.researchBps);
    }
    if (message.developmentBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.developmentBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RevenueSplit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRevenueSplit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contributorBps = reader.uint64();
          break;
        case 2:
          message.protocolBps = reader.uint64();
          break;
        case 3:
          message.researchBps = reader.uint64();
          break;
        case 4:
          message.developmentBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RevenueSplit>): RevenueSplit {
    const message = createBaseRevenueSplit();
    message.contributorBps = object.contributorBps !== undefined && object.contributorBps !== null ? BigInt(object.contributorBps.toString()) : BigInt(0);
    message.protocolBps = object.protocolBps !== undefined && object.protocolBps !== null ? BigInt(object.protocolBps.toString()) : BigInt(0);
    message.researchBps = object.researchBps !== undefined && object.researchBps !== null ? BigInt(object.researchBps.toString()) : BigInt(0);
    message.developmentBps = object.developmentBps !== undefined && object.developmentBps !== null ? BigInt(object.developmentBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseProtocolSubSplit(): ProtocolSubSplit {
  return {
    citationBps: BigInt(0),
    verificationBps: BigInt(0),
    treasuryBps: BigInt(0)
  };
}
/**
 * ProtocolSubSplit defines how the protocol share is divided.
 * @name ProtocolSubSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.ProtocolSubSplit
 */
export const ProtocolSubSplit = {
  typeUrl: "/zerone.common.v1.ProtocolSubSplit",
  encode(message: ProtocolSubSplit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.citationBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.citationBps);
    }
    if (message.verificationBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.verificationBps);
    }
    if (message.treasuryBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.treasuryBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ProtocolSubSplit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProtocolSubSplit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.citationBps = reader.uint64();
          break;
        case 2:
          message.verificationBps = reader.uint64();
          break;
        case 3:
          message.treasuryBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ProtocolSubSplit>): ProtocolSubSplit {
    const message = createBaseProtocolSubSplit();
    message.citationBps = object.citationBps !== undefined && object.citationBps !== null ? BigInt(object.citationBps.toString()) : BigInt(0);
    message.verificationBps = object.verificationBps !== undefined && object.verificationBps !== null ? BigInt(object.verificationBps.toString()) : BigInt(0);
    message.treasuryBps = object.treasuryBps !== undefined && object.treasuryBps !== null ? BigInt(object.treasuryBps.toString()) : BigInt(0);
    return message;
  }
};