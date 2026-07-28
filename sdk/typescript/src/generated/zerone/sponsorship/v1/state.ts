//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** BountyStatus enumerates the lifecycle states of a BountyOrder. */
export enum BountyStatus {
  BOUNTY_STATUS_UNSPECIFIED = 0,
  /** BOUNTY_STATUS_ACTIVE - accepting fulfillments */
  BOUNTY_STATUS_ACTIVE = 1,
  /** BOUNTY_STATUS_FULFILLED - target_count reached */
  BOUNTY_STATUS_FULFILLED = 2,
  /** BOUNTY_STATUS_EXPIRED - currentBlock >= end_block, set by BeginBlocker */
  BOUNTY_STATUS_EXPIRED = 3,
  /** BOUNTY_STATUS_CANCELED - sponsor reclaimed remaining */
  BOUNTY_STATUS_CANCELED = 4,
  UNRECOGNIZED = -1,
}
export function bountyStatusFromJSON(object: any): BountyStatus {
  switch (object) {
    case 0:
    case "BOUNTY_STATUS_UNSPECIFIED":
      return BountyStatus.BOUNTY_STATUS_UNSPECIFIED;
    case 1:
    case "BOUNTY_STATUS_ACTIVE":
      return BountyStatus.BOUNTY_STATUS_ACTIVE;
    case 2:
    case "BOUNTY_STATUS_FULFILLED":
      return BountyStatus.BOUNTY_STATUS_FULFILLED;
    case 3:
    case "BOUNTY_STATUS_EXPIRED":
      return BountyStatus.BOUNTY_STATUS_EXPIRED;
    case 4:
    case "BOUNTY_STATUS_CANCELED":
      return BountyStatus.BOUNTY_STATUS_CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BountyStatus.UNRECOGNIZED;
  }
}
export function bountyStatusToJSON(object: BountyStatus): string {
  switch (object) {
    case BountyStatus.BOUNTY_STATUS_UNSPECIFIED:
      return "BOUNTY_STATUS_UNSPECIFIED";
    case BountyStatus.BOUNTY_STATUS_ACTIVE:
      return "BOUNTY_STATUS_ACTIVE";
    case BountyStatus.BOUNTY_STATUS_FULFILLED:
      return "BOUNTY_STATUS_FULFILLED";
    case BountyStatus.BOUNTY_STATUS_EXPIRED:
      return "BOUNTY_STATUS_EXPIRED";
    case BountyStatus.BOUNTY_STATUS_CANCELED:
      return "BOUNTY_STATUS_CANCELED";
    case BountyStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * BountyOrder is a sponsor's commitment of escrowed funds against a
 * typed bounty for verified work in a specific domain.
 * @name BountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyOrder
 */
export interface BountyOrder {
  id: string;
  sponsor: string;
  domain: string;
  pricePerArtifact: string;
  targetCount: number;
  fulfilledCount: number;
  escrowRemaining: string;
  startBlock: bigint;
  endBlock: bigint;
  status: BountyStatus;
}
/**
 * BountyFulfillment records a single payout from a bounty to a worker
 * (the submitter of the fulfilling fact).
 * @name BountyFulfillment
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyFulfillment
 */
export interface BountyFulfillment {
  bountyId: string;
  factId: string;
  worker: string;
  amountPaid: string;
  fulfilledAtBlock: bigint;
}
/**
 * Params is the governance-tunable configuration.
 * @name Params
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.Params
 */
export interface Params {
  minTargetCount: number;
  minDurationBlocks: bigint;
  maxActiveBountiesPerSponsor: number;
}
function createBaseBountyOrder(): BountyOrder {
  return {
    id: "",
    sponsor: "",
    domain: "",
    pricePerArtifact: "",
    targetCount: 0,
    fulfilledCount: 0,
    escrowRemaining: "",
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    status: 0
  };
}
/**
 * BountyOrder is a sponsor's commitment of escrowed funds against a
 * typed bounty for verified work in a specific domain.
 * @name BountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyOrder
 */
export const BountyOrder = {
  typeUrl: "/zerone.sponsorship.v1.BountyOrder",
  encode(message: BountyOrder, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.sponsor !== "") {
      writer.uint32(18).string(message.sponsor);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.pricePerArtifact !== "") {
      writer.uint32(34).string(message.pricePerArtifact);
    }
    if (message.targetCount !== 0) {
      writer.uint32(40).uint32(message.targetCount);
    }
    if (message.fulfilledCount !== 0) {
      writer.uint32(48).uint32(message.fulfilledCount);
    }
    if (message.escrowRemaining !== "") {
      writer.uint32(58).string(message.escrowRemaining);
    }
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.startBlock);
    }
    if (message.endBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.endBlock);
    }
    if (message.status !== 0) {
      writer.uint32(80).int32(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): BountyOrder {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBountyOrder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.sponsor = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.pricePerArtifact = reader.string();
          break;
        case 5:
          message.targetCount = reader.uint32();
          break;
        case 6:
          message.fulfilledCount = reader.uint32();
          break;
        case 7:
          message.escrowRemaining = reader.string();
          break;
        case 8:
          message.startBlock = reader.uint64();
          break;
        case 9:
          message.endBlock = reader.uint64();
          break;
        case 10:
          message.status = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<BountyOrder>): BountyOrder {
    const message = createBaseBountyOrder();
    message.id = object.id ?? "";
    message.sponsor = object.sponsor ?? "";
    message.domain = object.domain ?? "";
    message.pricePerArtifact = object.pricePerArtifact ?? "";
    message.targetCount = object.targetCount ?? 0;
    message.fulfilledCount = object.fulfilledCount ?? 0;
    message.escrowRemaining = object.escrowRemaining ?? "";
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== undefined && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    return message;
  }
};
function createBaseBountyFulfillment(): BountyFulfillment {
  return {
    bountyId: "",
    factId: "",
    worker: "",
    amountPaid: "",
    fulfilledAtBlock: BigInt(0)
  };
}
/**
 * BountyFulfillment records a single payout from a bounty to a worker
 * (the submitter of the fulfilling fact).
 * @name BountyFulfillment
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyFulfillment
 */
export const BountyFulfillment = {
  typeUrl: "/zerone.sponsorship.v1.BountyFulfillment",
  encode(message: BountyFulfillment, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.bountyId !== "") {
      writer.uint32(10).string(message.bountyId);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.worker !== "") {
      writer.uint32(26).string(message.worker);
    }
    if (message.amountPaid !== "") {
      writer.uint32(34).string(message.amountPaid);
    }
    if (message.fulfilledAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.fulfilledAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): BountyFulfillment {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBountyFulfillment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bountyId = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.worker = reader.string();
          break;
        case 4:
          message.amountPaid = reader.string();
          break;
        case 5:
          message.fulfilledAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<BountyFulfillment>): BountyFulfillment {
    const message = createBaseBountyFulfillment();
    message.bountyId = object.bountyId ?? "";
    message.factId = object.factId ?? "";
    message.worker = object.worker ?? "";
    message.amountPaid = object.amountPaid ?? "";
    message.fulfilledAtBlock = object.fulfilledAtBlock !== undefined && object.fulfilledAtBlock !== null ? BigInt(object.fulfilledAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams(): Params {
  return {
    minTargetCount: 0,
    minDurationBlocks: BigInt(0),
    maxActiveBountiesPerSponsor: 0
  };
}
/**
 * Params is the governance-tunable configuration.
 * @name Params
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.sponsorship.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minTargetCount !== 0) {
      writer.uint32(8).uint32(message.minTargetCount);
    }
    if (message.minDurationBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.minDurationBlocks);
    }
    if (message.maxActiveBountiesPerSponsor !== 0) {
      writer.uint32(24).uint32(message.maxActiveBountiesPerSponsor);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minTargetCount = reader.uint32();
          break;
        case 2:
          message.minDurationBlocks = reader.uint64();
          break;
        case 3:
          message.maxActiveBountiesPerSponsor = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.minTargetCount = object.minTargetCount ?? 0;
    message.minDurationBlocks = object.minDurationBlocks !== undefined && object.minDurationBlocks !== null ? BigInt(object.minDurationBlocks.toString()) : BigInt(0);
    message.maxActiveBountiesPerSponsor = object.maxActiveBountiesPerSponsor ?? 0;
    return message;
  }
};