//@ts-nocheck
import { FactStatus } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * CascadeReplaySelector walks the disproval-graph from a DISPROVEN root.
 * Returns the disproven fact + first-hop cascaded descendants + every
 * CascadeEvent the chain recorded when the disproof landed. Optionally
 * includes vindication records and supersession-chain pointers.
 *
 * TC4 (the graph carries its disprovals) is the doctrinal binding.
 * Plan 1's RootedSubtree/AncestorCone deliberately filter out
 * CONTRADICTS|SUPERSEDES edges; CascadeReplay deliberately includes them.
 * The two selectors compose: a trainer that wants both the support-graph
 * and the disproval-graph for a fact issues two BundleToK calls.
 * @name CascadeReplaySelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeReplaySelector
 */
export interface CascadeReplaySelector {
  /**
   * must reference a fact with Status=DISPROVEN
   */
  disprovenFactId: string;
  /**
   * cascade depth; 0 = chain default 1; cap 3
   */
  maxDepth: number;
  /**
   * ship VindicationRecord entries
   */
  includeVindications: boolean;
  /**
   * walk SUPERSEDES chain (8-deep cap)
   */
  includeSupersessions: boolean;
  /**
   * ship StatusTransition timeline per node
   */
  includeStatusHistory: boolean;
}
/**
 * CascadeEvent is the persistent record of a single descendant status flip
 * during a falsification cascade. The chain emits a `falsification_cascade`
 * voice event when this fires; the record is what makes TC4 bundle-able.
 *
 * One CascadeEvent is written per descendant per disproof. Forward-only
 * (commitment 10): records are append-only, never amended in place.
 * @name CascadeEvent
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeEvent
 */
export interface CascadeEvent {
  /**
   * monotonic per disproven_fact_id
   */
  seq: bigint;
  /**
   * root of the cascade
   */
  disprovenFactId: string;
  /**
   * direct descendant whose status flipped
   */
  descendantFactId: string;
  /**
   * challenge that triggered the cascade
   */
  challengeClaimId: string;
  /**
   * SUPPORTS|REQUIRES|REFINES|GENERALIZES|CITES
   */
  edgeRelation: string;
  /**
   * VERIFIED|ACTIVE|AT_RISK
   */
  priorStatus: FactStatus;
  /**
   * CONTESTED (current cascade always)
   */
  newStatus: FactStatus;
  /**
   * height the cascade fired
   */
  blockHeight: bigint;
}
/**
 * StatusTransition is the persistent record of a single status change on
 * a fact. Forward-only: once written, never modified. The full history of
 * a fact's status is reconstructable by iterating these records by factID.
 *
 * TC4: "Fact.status is preserved in graph manifests with full transition
 * history." This is the record that makes "full transition history" real.
 * @name StatusTransition
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.StatusTransition
 */
export interface StatusTransition {
  /**
   * monotonic per fact_id
   */
  seq: bigint;
  factId: string;
  /**
   * status before this transition
   */
  priorStatus: FactStatus;
  /**
   * status after this transition
   */
  newStatus: FactStatus;
  blockHeight: bigint;
  /**
   * "verification" | "challenge_disproven" | "cascade" | "supersession"
   */
  causeEventType: string;
  /**
   * round_id | challenge_claim_id | disproven_fact_id | superseding_fact_id
   */
  causeId: string;
}
/**
 * ToKVindicationRecord is the proto mirror of the existing Go struct
 * VindicationRecord in x/knowledge/types/vindication.go. The keeper
 * writes/reads JSON for backward compatibility but converts to/from this
 * proto for bundle assembly (TC4). Named with a ToK prefix to avoid a
 * collision with the pre-existing Go struct in the same package.
 * @name ToKVindicationRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ToKVindicationRecord
 */
export interface ToKVindicationRecord {
  verifier: string;
  factId: string;
  refundAmount: string;
  bonusAmount: string;
  vindicatedAt: bigint;
  disprovenBy: string;
  roundId: string;
}
function createBaseCascadeReplaySelector(): CascadeReplaySelector {
  return {
    disprovenFactId: "",
    maxDepth: 0,
    includeVindications: false,
    includeSupersessions: false,
    includeStatusHistory: false
  };
}
/**
 * CascadeReplaySelector walks the disproval-graph from a DISPROVEN root.
 * Returns the disproven fact + first-hop cascaded descendants + every
 * CascadeEvent the chain recorded when the disproof landed. Optionally
 * includes vindication records and supersession-chain pointers.
 *
 * TC4 (the graph carries its disprovals) is the doctrinal binding.
 * Plan 1's RootedSubtree/AncestorCone deliberately filter out
 * CONTRADICTS|SUPERSEDES edges; CascadeReplay deliberately includes them.
 * The two selectors compose: a trainer that wants both the support-graph
 * and the disproval-graph for a fact issues two BundleToK calls.
 * @name CascadeReplaySelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeReplaySelector
 */
export const CascadeReplaySelector = {
  typeUrl: "/zerone.knowledge.v1.CascadeReplaySelector",
  encode(message: CascadeReplaySelector, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.disprovenFactId !== "") {
      writer.uint32(10).string(message.disprovenFactId);
    }
    if (message.maxDepth !== 0) {
      writer.uint32(16).uint32(message.maxDepth);
    }
    if (message.includeVindications === true) {
      writer.uint32(24).bool(message.includeVindications);
    }
    if (message.includeSupersessions === true) {
      writer.uint32(32).bool(message.includeSupersessions);
    }
    if (message.includeStatusHistory === true) {
      writer.uint32(40).bool(message.includeStatusHistory);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CascadeReplaySelector {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCascadeReplaySelector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.disprovenFactId = reader.string();
          break;
        case 2:
          message.maxDepth = reader.uint32();
          break;
        case 3:
          message.includeVindications = reader.bool();
          break;
        case 4:
          message.includeSupersessions = reader.bool();
          break;
        case 5:
          message.includeStatusHistory = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CascadeReplaySelector>): CascadeReplaySelector {
    const message = createBaseCascadeReplaySelector();
    message.disprovenFactId = object.disprovenFactId ?? "";
    message.maxDepth = object.maxDepth ?? 0;
    message.includeVindications = object.includeVindications ?? false;
    message.includeSupersessions = object.includeSupersessions ?? false;
    message.includeStatusHistory = object.includeStatusHistory ?? false;
    return message;
  }
};
function createBaseCascadeEvent(): CascadeEvent {
  return {
    seq: BigInt(0),
    disprovenFactId: "",
    descendantFactId: "",
    challengeClaimId: "",
    edgeRelation: "",
    priorStatus: 0,
    newStatus: 0,
    blockHeight: BigInt(0)
  };
}
/**
 * CascadeEvent is the persistent record of a single descendant status flip
 * during a falsification cascade. The chain emits a `falsification_cascade`
 * voice event when this fires; the record is what makes TC4 bundle-able.
 *
 * One CascadeEvent is written per descendant per disproof. Forward-only
 * (commitment 10): records are append-only, never amended in place.
 * @name CascadeEvent
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeEvent
 */
export const CascadeEvent = {
  typeUrl: "/zerone.knowledge.v1.CascadeEvent",
  encode(message: CascadeEvent, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.seq !== BigInt(0)) {
      writer.uint32(8).uint64(message.seq);
    }
    if (message.disprovenFactId !== "") {
      writer.uint32(18).string(message.disprovenFactId);
    }
    if (message.descendantFactId !== "") {
      writer.uint32(26).string(message.descendantFactId);
    }
    if (message.challengeClaimId !== "") {
      writer.uint32(34).string(message.challengeClaimId);
    }
    if (message.edgeRelation !== "") {
      writer.uint32(42).string(message.edgeRelation);
    }
    if (message.priorStatus !== 0) {
      writer.uint32(48).int32(message.priorStatus);
    }
    if (message.newStatus !== 0) {
      writer.uint32(56).int32(message.newStatus);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(64).uint64(message.blockHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CascadeEvent {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCascadeEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.seq = reader.uint64();
          break;
        case 2:
          message.disprovenFactId = reader.string();
          break;
        case 3:
          message.descendantFactId = reader.string();
          break;
        case 4:
          message.challengeClaimId = reader.string();
          break;
        case 5:
          message.edgeRelation = reader.string();
          break;
        case 6:
          message.priorStatus = reader.int32() as any;
          break;
        case 7:
          message.newStatus = reader.int32() as any;
          break;
        case 8:
          message.blockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CascadeEvent>): CascadeEvent {
    const message = createBaseCascadeEvent();
    message.seq = object.seq !== undefined && object.seq !== null ? BigInt(object.seq.toString()) : BigInt(0);
    message.disprovenFactId = object.disprovenFactId ?? "";
    message.descendantFactId = object.descendantFactId ?? "";
    message.challengeClaimId = object.challengeClaimId ?? "";
    message.edgeRelation = object.edgeRelation ?? "";
    message.priorStatus = object.priorStatus ?? 0;
    message.newStatus = object.newStatus ?? 0;
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseStatusTransition(): StatusTransition {
  return {
    seq: BigInt(0),
    factId: "",
    priorStatus: 0,
    newStatus: 0,
    blockHeight: BigInt(0),
    causeEventType: "",
    causeId: ""
  };
}
/**
 * StatusTransition is the persistent record of a single status change on
 * a fact. Forward-only: once written, never modified. The full history of
 * a fact's status is reconstructable by iterating these records by factID.
 *
 * TC4: "Fact.status is preserved in graph manifests with full transition
 * history." This is the record that makes "full transition history" real.
 * @name StatusTransition
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.StatusTransition
 */
export const StatusTransition = {
  typeUrl: "/zerone.knowledge.v1.StatusTransition",
  encode(message: StatusTransition, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.seq !== BigInt(0)) {
      writer.uint32(8).uint64(message.seq);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.priorStatus !== 0) {
      writer.uint32(24).int32(message.priorStatus);
    }
    if (message.newStatus !== 0) {
      writer.uint32(32).int32(message.newStatus);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(40).uint64(message.blockHeight);
    }
    if (message.causeEventType !== "") {
      writer.uint32(50).string(message.causeEventType);
    }
    if (message.causeId !== "") {
      writer.uint32(58).string(message.causeId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): StatusTransition {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStatusTransition();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.seq = reader.uint64();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.priorStatus = reader.int32() as any;
          break;
        case 4:
          message.newStatus = reader.int32() as any;
          break;
        case 5:
          message.blockHeight = reader.uint64();
          break;
        case 6:
          message.causeEventType = reader.string();
          break;
        case 7:
          message.causeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<StatusTransition>): StatusTransition {
    const message = createBaseStatusTransition();
    message.seq = object.seq !== undefined && object.seq !== null ? BigInt(object.seq.toString()) : BigInt(0);
    message.factId = object.factId ?? "";
    message.priorStatus = object.priorStatus ?? 0;
    message.newStatus = object.newStatus ?? 0;
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    message.causeEventType = object.causeEventType ?? "";
    message.causeId = object.causeId ?? "";
    return message;
  }
};
function createBaseToKVindicationRecord(): ToKVindicationRecord {
  return {
    verifier: "",
    factId: "",
    refundAmount: "",
    bonusAmount: "",
    vindicatedAt: BigInt(0),
    disprovenBy: "",
    roundId: ""
  };
}
/**
 * ToKVindicationRecord is the proto mirror of the existing Go struct
 * VindicationRecord in x/knowledge/types/vindication.go. The keeper
 * writes/reads JSON for backward compatibility but converts to/from this
 * proto for bundle assembly (TC4). Named with a ToK prefix to avoid a
 * collision with the pre-existing Go struct in the same package.
 * @name ToKVindicationRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ToKVindicationRecord
 */
export const ToKVindicationRecord = {
  typeUrl: "/zerone.knowledge.v1.ToKVindicationRecord",
  encode(message: ToKVindicationRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.refundAmount !== "") {
      writer.uint32(26).string(message.refundAmount);
    }
    if (message.bonusAmount !== "") {
      writer.uint32(34).string(message.bonusAmount);
    }
    if (message.vindicatedAt !== BigInt(0)) {
      writer.uint32(40).uint64(message.vindicatedAt);
    }
    if (message.disprovenBy !== "") {
      writer.uint32(50).string(message.disprovenBy);
    }
    if (message.roundId !== "") {
      writer.uint32(58).string(message.roundId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ToKVindicationRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseToKVindicationRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.refundAmount = reader.string();
          break;
        case 4:
          message.bonusAmount = reader.string();
          break;
        case 5:
          message.vindicatedAt = reader.uint64();
          break;
        case 6:
          message.disprovenBy = reader.string();
          break;
        case 7:
          message.roundId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ToKVindicationRecord>): ToKVindicationRecord {
    const message = createBaseToKVindicationRecord();
    message.verifier = object.verifier ?? "";
    message.factId = object.factId ?? "";
    message.refundAmount = object.refundAmount ?? "";
    message.bonusAmount = object.bonusAmount ?? "";
    message.vindicatedAt = object.vindicatedAt !== undefined && object.vindicatedAt !== null ? BigInt(object.vindicatedAt.toString()) : BigInt(0);
    message.disprovenBy = object.disprovenBy ?? "";
    message.roundId = object.roundId ?? "";
    return message;
  }
};