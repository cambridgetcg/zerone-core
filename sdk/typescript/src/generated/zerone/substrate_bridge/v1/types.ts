//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * CitationType distinguishes citation strengths for lineage propagation
 * (M6 generalized). Mirrors the ToK relation-type semantics applied
 * across work classes.
 */
export enum CitationType {
  CITATION_TYPE_UNSPECIFIED = 0,
  /** CITATION_TYPE_CITES - 1× base weight */
  CITATION_TYPE_CITES = 1,
  /** CITATION_TYPE_SUPPORTS - 2× base weight */
  CITATION_TYPE_SUPPORTS = 2,
  /** CITATION_TYPE_EXTENDS - 3× base weight */
  CITATION_TYPE_EXTENDS = 3,
  /** CITATION_TYPE_REFINES - 3× base weight */
  CITATION_TYPE_REFINES = 4,
  /** CITATION_TYPE_GENERALIZES - 4× base weight */
  CITATION_TYPE_GENERALIZES = 5,
  UNRECOGNIZED = -1,
}
export function citationTypeFromJSON(object: any): CitationType {
  switch (object) {
    case 0:
    case "CITATION_TYPE_UNSPECIFIED":
      return CitationType.CITATION_TYPE_UNSPECIFIED;
    case 1:
    case "CITATION_TYPE_CITES":
      return CitationType.CITATION_TYPE_CITES;
    case 2:
    case "CITATION_TYPE_SUPPORTS":
      return CitationType.CITATION_TYPE_SUPPORTS;
    case 3:
    case "CITATION_TYPE_EXTENDS":
      return CitationType.CITATION_TYPE_EXTENDS;
    case 4:
    case "CITATION_TYPE_REFINES":
      return CitationType.CITATION_TYPE_REFINES;
    case 5:
    case "CITATION_TYPE_GENERALIZES":
      return CitationType.CITATION_TYPE_GENERALIZES;
    case -1:
    case "UNRECOGNIZED":
    default:
      return CitationType.UNRECOGNIZED;
  }
}
export function citationTypeToJSON(object: CitationType): string {
  switch (object) {
    case CitationType.CITATION_TYPE_UNSPECIFIED:
      return "CITATION_TYPE_UNSPECIFIED";
    case CitationType.CITATION_TYPE_CITES:
      return "CITATION_TYPE_CITES";
    case CitationType.CITATION_TYPE_SUPPORTS:
      return "CITATION_TYPE_SUPPORTS";
    case CitationType.CITATION_TYPE_EXTENDS:
      return "CITATION_TYPE_EXTENDS";
    case CitationType.CITATION_TYPE_REFINES:
      return "CITATION_TYPE_REFINES";
    case CitationType.CITATION_TYPE_GENERALIZES:
      return "CITATION_TYPE_GENERALIZES";
    case CitationType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * ExternalSource is a typed reference to off-chain content that an
 * adapter has fetched. The content_hash is the cryptographic anchor:
 * substrate-link re-derivation matches if and only if the source's
 * content_hash matches what the adapter binary produced.
 * @name ExternalSource
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ExternalSource
 */
export interface ExternalSource {
  adapterId: string;
  /**
   * e.g. Wikipedia article ID
   */
  sourceId: string;
  /**
   * optional; for audit
   */
  sourceUrl: string;
  /**
   * sha256 of fetched content
   */
  contentHash: Uint8Array;
  fetchedAtBlock: bigint;
}
/**
 * AxisProjection is the per-axis recursion-weight contribution of an
 * external work artifact, in the order fixed by USEFUL_WORK.md
 * section "The six recursive axes". Units are uint64 weights, bounded
 * by an adapter's AxisBounds.
 * @name AxisProjection
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisProjection
 */
export interface AxisProjection {
  axisSubstrate: bigint;
  axisVerification: bigint;
  axisClassification: bigint;
  axisAttribution: bigint;
  axisTooling: bigint;
  axisInterface: bigint;
}
/**
 * AxisBounds caps the per-axis projection an adapter is allowed to
 * claim. Gov-approved at adapter registration; enforced at attestation
 * submit.
 * @name AxisBounds
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisBounds
 */
export interface AxisBounds {
  axisSubstrateMax: bigint;
  axisVerificationMax: bigint;
  axisClassificationMax: bigint;
  axisAttributionMax: bigint;
  axisToolingMax: bigint;
  axisInterfaceMax: bigint;
}
function createBaseExternalSource(): ExternalSource {
  return {
    adapterId: "",
    sourceId: "",
    sourceUrl: "",
    contentHash: new Uint8Array(),
    fetchedAtBlock: BigInt(0)
  };
}
/**
 * ExternalSource is a typed reference to off-chain content that an
 * adapter has fetched. The content_hash is the cryptographic anchor:
 * substrate-link re-derivation matches if and only if the source's
 * content_hash matches what the adapter binary produced.
 * @name ExternalSource
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ExternalSource
 */
export const ExternalSource = {
  typeUrl: "/zerone.substrate_bridge.v1.ExternalSource",
  encode(message: ExternalSource, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.adapterId !== "") {
      writer.uint32(10).string(message.adapterId);
    }
    if (message.sourceId !== "") {
      writer.uint32(18).string(message.sourceId);
    }
    if (message.sourceUrl !== "") {
      writer.uint32(26).string(message.sourceUrl);
    }
    if (message.contentHash.length !== 0) {
      writer.uint32(34).bytes(message.contentHash);
    }
    if (message.fetchedAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.fetchedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ExternalSource {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExternalSource();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.adapterId = reader.string();
          break;
        case 2:
          message.sourceId = reader.string();
          break;
        case 3:
          message.sourceUrl = reader.string();
          break;
        case 4:
          message.contentHash = reader.bytes();
          break;
        case 5:
          message.fetchedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ExternalSource>): ExternalSource {
    const message = createBaseExternalSource();
    message.adapterId = object.adapterId ?? "";
    message.sourceId = object.sourceId ?? "";
    message.sourceUrl = object.sourceUrl ?? "";
    message.contentHash = object.contentHash ?? new Uint8Array();
    message.fetchedAtBlock = object.fetchedAtBlock !== undefined && object.fetchedAtBlock !== null ? BigInt(object.fetchedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAxisProjection(): AxisProjection {
  return {
    axisSubstrate: BigInt(0),
    axisVerification: BigInt(0),
    axisClassification: BigInt(0),
    axisAttribution: BigInt(0),
    axisTooling: BigInt(0),
    axisInterface: BigInt(0)
  };
}
/**
 * AxisProjection is the per-axis recursion-weight contribution of an
 * external work artifact, in the order fixed by USEFUL_WORK.md
 * section "The six recursive axes". Units are uint64 weights, bounded
 * by an adapter's AxisBounds.
 * @name AxisProjection
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisProjection
 */
export const AxisProjection = {
  typeUrl: "/zerone.substrate_bridge.v1.AxisProjection",
  encode(message: AxisProjection, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.axisSubstrate !== BigInt(0)) {
      writer.uint32(8).uint64(message.axisSubstrate);
    }
    if (message.axisVerification !== BigInt(0)) {
      writer.uint32(16).uint64(message.axisVerification);
    }
    if (message.axisClassification !== BigInt(0)) {
      writer.uint32(24).uint64(message.axisClassification);
    }
    if (message.axisAttribution !== BigInt(0)) {
      writer.uint32(32).uint64(message.axisAttribution);
    }
    if (message.axisTooling !== BigInt(0)) {
      writer.uint32(40).uint64(message.axisTooling);
    }
    if (message.axisInterface !== BigInt(0)) {
      writer.uint32(48).uint64(message.axisInterface);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AxisProjection {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAxisProjection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.axisSubstrate = reader.uint64();
          break;
        case 2:
          message.axisVerification = reader.uint64();
          break;
        case 3:
          message.axisClassification = reader.uint64();
          break;
        case 4:
          message.axisAttribution = reader.uint64();
          break;
        case 5:
          message.axisTooling = reader.uint64();
          break;
        case 6:
          message.axisInterface = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AxisProjection>): AxisProjection {
    const message = createBaseAxisProjection();
    message.axisSubstrate = object.axisSubstrate !== undefined && object.axisSubstrate !== null ? BigInt(object.axisSubstrate.toString()) : BigInt(0);
    message.axisVerification = object.axisVerification !== undefined && object.axisVerification !== null ? BigInt(object.axisVerification.toString()) : BigInt(0);
    message.axisClassification = object.axisClassification !== undefined && object.axisClassification !== null ? BigInt(object.axisClassification.toString()) : BigInt(0);
    message.axisAttribution = object.axisAttribution !== undefined && object.axisAttribution !== null ? BigInt(object.axisAttribution.toString()) : BigInt(0);
    message.axisTooling = object.axisTooling !== undefined && object.axisTooling !== null ? BigInt(object.axisTooling.toString()) : BigInt(0);
    message.axisInterface = object.axisInterface !== undefined && object.axisInterface !== null ? BigInt(object.axisInterface.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAxisBounds(): AxisBounds {
  return {
    axisSubstrateMax: BigInt(0),
    axisVerificationMax: BigInt(0),
    axisClassificationMax: BigInt(0),
    axisAttributionMax: BigInt(0),
    axisToolingMax: BigInt(0),
    axisInterfaceMax: BigInt(0)
  };
}
/**
 * AxisBounds caps the per-axis projection an adapter is allowed to
 * claim. Gov-approved at adapter registration; enforced at attestation
 * submit.
 * @name AxisBounds
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisBounds
 */
export const AxisBounds = {
  typeUrl: "/zerone.substrate_bridge.v1.AxisBounds",
  encode(message: AxisBounds, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.axisSubstrateMax !== BigInt(0)) {
      writer.uint32(8).uint64(message.axisSubstrateMax);
    }
    if (message.axisVerificationMax !== BigInt(0)) {
      writer.uint32(16).uint64(message.axisVerificationMax);
    }
    if (message.axisClassificationMax !== BigInt(0)) {
      writer.uint32(24).uint64(message.axisClassificationMax);
    }
    if (message.axisAttributionMax !== BigInt(0)) {
      writer.uint32(32).uint64(message.axisAttributionMax);
    }
    if (message.axisToolingMax !== BigInt(0)) {
      writer.uint32(40).uint64(message.axisToolingMax);
    }
    if (message.axisInterfaceMax !== BigInt(0)) {
      writer.uint32(48).uint64(message.axisInterfaceMax);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AxisBounds {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAxisBounds();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.axisSubstrateMax = reader.uint64();
          break;
        case 2:
          message.axisVerificationMax = reader.uint64();
          break;
        case 3:
          message.axisClassificationMax = reader.uint64();
          break;
        case 4:
          message.axisAttributionMax = reader.uint64();
          break;
        case 5:
          message.axisToolingMax = reader.uint64();
          break;
        case 6:
          message.axisInterfaceMax = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AxisBounds>): AxisBounds {
    const message = createBaseAxisBounds();
    message.axisSubstrateMax = object.axisSubstrateMax !== undefined && object.axisSubstrateMax !== null ? BigInt(object.axisSubstrateMax.toString()) : BigInt(0);
    message.axisVerificationMax = object.axisVerificationMax !== undefined && object.axisVerificationMax !== null ? BigInt(object.axisVerificationMax.toString()) : BigInt(0);
    message.axisClassificationMax = object.axisClassificationMax !== undefined && object.axisClassificationMax !== null ? BigInt(object.axisClassificationMax.toString()) : BigInt(0);
    message.axisAttributionMax = object.axisAttributionMax !== undefined && object.axisAttributionMax !== null ? BigInt(object.axisAttributionMax.toString()) : BigInt(0);
    message.axisToolingMax = object.axisToolingMax !== undefined && object.axisToolingMax !== null ? BigInt(object.axisToolingMax.toString()) : BigInt(0);
    message.axisInterfaceMax = object.axisInterfaceMax !== undefined && object.axisInterfaceMax !== null ? BigInt(object.axisInterfaceMax.toString()) : BigInt(0);
    return message;
  }
};