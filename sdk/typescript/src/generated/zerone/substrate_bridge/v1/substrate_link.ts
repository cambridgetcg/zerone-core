//@ts-nocheck
import { AxisProjection, ExternalSource, CitationType } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * SubstrateLink is the deterministic provenance from external content
 * to ToK fact-IDs (existing + pending). Two sections — cited_facts MUST
 * exist in x/knowledge at commit time; pending_claims are auto-submitted
 * as Claims and the attestation is held in AWAITING_RESOLUTION until
 * they resolve. M2 satisfied: every pending claim becomes a real
 * on-chain claim with full provenance.
 * @name SubstrateLink
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SubstrateLink
 */
export interface SubstrateLink {
  citedFacts: FactCitation[];
  pendingClaims: PendingClaim[];
  recursionWeight?: AxisProjection;
  adapterId: string;
  source?: ExternalSource;
  /**
   * sha256 of canonical form
   */
  linkHash: Uint8Array;
}
/**
 * FactCitation is one outgoing edge in the substrate-link. citation_type
 * drives lineage propagation weight (M6).
 * @name FactCitation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.FactCitation
 */
export interface FactCitation {
  factId: string;
  citationType: CitationType;
  /**
   * optional excerpt for audit
   */
  citationContext: string;
}
/**
 * PendingClaim is a Claim auto-submitted at commit phase. Shape mirrors
 * x/knowledge.Claim so the substrate_bridge keeper can call
 * x/knowledge.SetClaim directly. claim_relations cite existing facts;
 * they are NOT recursive pending claims (one-hop deferral only).
 * @name PendingClaim
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.PendingClaim
 */
export interface PendingClaim {
  claimContent: string;
  /**
   * optional; chain assigns if empty
   */
  proposedFactId: string;
  domain: string;
  methodologyId: string;
  relations: ClaimRelation[];
}
/**
 * ClaimRelation is a citation from a pending claim to an existing fact.
 * Mirrors x/knowledge.ClaimRelation. Pending claims cannot cite OTHER
 * pending claims — they cite existing verified facts only. This keeps
 * the resolution graph a tree (one-hop deferral).
 * @name ClaimRelation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ClaimRelation
 */
export interface ClaimRelation {
  targetFactId: string;
  /**
   * SUPPORTS | REQUIRES | etc. (mirrors x/knowledge)
   */
  relation: string;
  /**
   * DEDUCTIVE | INDUCTIVE | etc.
   */
  inference: string;
  inferenceStrengthBps: number;
}
function createBaseSubstrateLink(): SubstrateLink {
  return {
    citedFacts: [],
    pendingClaims: [],
    recursionWeight: undefined,
    adapterId: "",
    source: undefined,
    linkHash: new Uint8Array()
  };
}
/**
 * SubstrateLink is the deterministic provenance from external content
 * to ToK fact-IDs (existing + pending). Two sections — cited_facts MUST
 * exist in x/knowledge at commit time; pending_claims are auto-submitted
 * as Claims and the attestation is held in AWAITING_RESOLUTION until
 * they resolve. M2 satisfied: every pending claim becomes a real
 * on-chain claim with full provenance.
 * @name SubstrateLink
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SubstrateLink
 */
export const SubstrateLink = {
  typeUrl: "/zerone.substrate_bridge.v1.SubstrateLink",
  encode(message: SubstrateLink, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    for (const v of message.citedFacts) {
      FactCitation.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.pendingClaims) {
      PendingClaim.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.recursionWeight !== undefined) {
      AxisProjection.encode(message.recursionWeight, writer.uint32(26).fork()).ldelim();
    }
    if (message.adapterId !== "") {
      writer.uint32(34).string(message.adapterId);
    }
    if (message.source !== undefined) {
      ExternalSource.encode(message.source, writer.uint32(42).fork()).ldelim();
    }
    if (message.linkHash.length !== 0) {
      writer.uint32(50).bytes(message.linkHash);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SubstrateLink {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSubstrateLink();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.citedFacts.push(FactCitation.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pendingClaims.push(PendingClaim.decode(reader, reader.uint32()));
          break;
        case 3:
          message.recursionWeight = AxisProjection.decode(reader, reader.uint32());
          break;
        case 4:
          message.adapterId = reader.string();
          break;
        case 5:
          message.source = ExternalSource.decode(reader, reader.uint32());
          break;
        case 6:
          message.linkHash = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SubstrateLink>): SubstrateLink {
    const message = createBaseSubstrateLink();
    message.citedFacts = object.citedFacts?.map(e => FactCitation.fromPartial(e)) || [];
    message.pendingClaims = object.pendingClaims?.map(e => PendingClaim.fromPartial(e)) || [];
    message.recursionWeight = object.recursionWeight !== undefined && object.recursionWeight !== null ? AxisProjection.fromPartial(object.recursionWeight) : undefined;
    message.adapterId = object.adapterId ?? "";
    message.source = object.source !== undefined && object.source !== null ? ExternalSource.fromPartial(object.source) : undefined;
    message.linkHash = object.linkHash ?? new Uint8Array();
    return message;
  }
};
function createBaseFactCitation(): FactCitation {
  return {
    factId: "",
    citationType: 0,
    citationContext: ""
  };
}
/**
 * FactCitation is one outgoing edge in the substrate-link. citation_type
 * drives lineage propagation weight (M6).
 * @name FactCitation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.FactCitation
 */
export const FactCitation = {
  typeUrl: "/zerone.substrate_bridge.v1.FactCitation",
  encode(message: FactCitation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.factId !== "") {
      writer.uint32(10).string(message.factId);
    }
    if (message.citationType !== 0) {
      writer.uint32(16).int32(message.citationType);
    }
    if (message.citationContext !== "") {
      writer.uint32(26).string(message.citationContext);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): FactCitation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFactCitation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.factId = reader.string();
          break;
        case 2:
          message.citationType = reader.int32() as any;
          break;
        case 3:
          message.citationContext = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<FactCitation>): FactCitation {
    const message = createBaseFactCitation();
    message.factId = object.factId ?? "";
    message.citationType = object.citationType ?? 0;
    message.citationContext = object.citationContext ?? "";
    return message;
  }
};
function createBasePendingClaim(): PendingClaim {
  return {
    claimContent: "",
    proposedFactId: "",
    domain: "",
    methodologyId: "",
    relations: []
  };
}
/**
 * PendingClaim is a Claim auto-submitted at commit phase. Shape mirrors
 * x/knowledge.Claim so the substrate_bridge keeper can call
 * x/knowledge.SetClaim directly. claim_relations cite existing facts;
 * they are NOT recursive pending claims (one-hop deferral only).
 * @name PendingClaim
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.PendingClaim
 */
export const PendingClaim = {
  typeUrl: "/zerone.substrate_bridge.v1.PendingClaim",
  encode(message: PendingClaim, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimContent !== "") {
      writer.uint32(10).string(message.claimContent);
    }
    if (message.proposedFactId !== "") {
      writer.uint32(18).string(message.proposedFactId);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.methodologyId !== "") {
      writer.uint32(34).string(message.methodologyId);
    }
    for (const v of message.relations) {
      ClaimRelation.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PendingClaim {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePendingClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimContent = reader.string();
          break;
        case 2:
          message.proposedFactId = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.methodologyId = reader.string();
          break;
        case 5:
          message.relations.push(ClaimRelation.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PendingClaim>): PendingClaim {
    const message = createBasePendingClaim();
    message.claimContent = object.claimContent ?? "";
    message.proposedFactId = object.proposedFactId ?? "";
    message.domain = object.domain ?? "";
    message.methodologyId = object.methodologyId ?? "";
    message.relations = object.relations?.map(e => ClaimRelation.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimRelation(): ClaimRelation {
  return {
    targetFactId: "",
    relation: "",
    inference: "",
    inferenceStrengthBps: 0
  };
}
/**
 * ClaimRelation is a citation from a pending claim to an existing fact.
 * Mirrors x/knowledge.ClaimRelation. Pending claims cannot cite OTHER
 * pending claims — they cite existing verified facts only. This keeps
 * the resolution graph a tree (one-hop deferral).
 * @name ClaimRelation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ClaimRelation
 */
export const ClaimRelation = {
  typeUrl: "/zerone.substrate_bridge.v1.ClaimRelation",
  encode(message: ClaimRelation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.targetFactId !== "") {
      writer.uint32(10).string(message.targetFactId);
    }
    if (message.relation !== "") {
      writer.uint32(18).string(message.relation);
    }
    if (message.inference !== "") {
      writer.uint32(26).string(message.inference);
    }
    if (message.inferenceStrengthBps !== 0) {
      writer.uint32(32).uint32(message.inferenceStrengthBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimRelation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimRelation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetFactId = reader.string();
          break;
        case 2:
          message.relation = reader.string();
          break;
        case 3:
          message.inference = reader.string();
          break;
        case 4:
          message.inferenceStrengthBps = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimRelation>): ClaimRelation {
    const message = createBaseClaimRelation();
    message.targetFactId = object.targetFactId ?? "";
    message.relation = object.relation ?? "";
    message.inference = object.inference ?? "";
    message.inferenceStrengthBps = object.inferenceStrengthBps ?? 0;
    return message;
  }
};