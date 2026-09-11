//@ts-nocheck
import { Claim, VerificationRound, Fact, FactRelation } from "./types";
import { StatusTransition } from "./tok_cascade";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * Reads retained records at the actual query context height. A nonzero requested
 * height must match that context; it does not reconstruct pruned state.
 * @name QueryClaimHistoryRequest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryRequest
 */
export interface QueryClaimHistoryRequest {
  id: string;
  atBlockHeight: bigint;
}
/**
 * An observation of stored state, not a transaction inclusion proof or a
 * scientific verdict. The bounded query either returns its whole scope or fails.
 * @name QueryClaimHistoryResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryResponse
 */
export interface QueryClaimHistoryResponse {
  chainId: string;
  blockHeight: bigint;
  record?: ClaimHistoryRecord;
  relatedClaims: RelatedClaimHistory[];
}
/**
 * Includes all retained rounds and derived facts for this claim ID. The claim
 * itself can be absent when surviving records refer to a historical missing row.
 * @name ClaimHistoryRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryRecord
 */
export interface ClaimHistoryRecord {
  claimId: string;
  claim?: Claim;
  rounds: VerificationRound[];
  facts: ClaimHistoryFact[];
  missingRoundIds: string[];
}
/**
 * Relations are current canonical metadata; transitions are retained history.
 * Neighbor IDs do not expand the query into their fact bodies or descendants.
 * @name ClaimHistoryFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryFact
 */
export interface ClaimHistoryFact {
  fact?: Fact;
  outgoingRelations: FactRelation[];
  incomingRelations: FactRelation[];
  statusTransitions: StatusTransition[];
}
/**
 * A single hop through an explicit stored claim field. This does not establish
 * causation, successful adjudication, credibility, or a completed payment.
 * @name RelatedClaimHistory
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RelatedClaimHistory
 */
export interface RelatedClaimHistory {
  record?: ClaimHistoryRecord;
  links: ClaimHistoryLink[];
}
/**
 * @name ClaimHistoryLink
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryLink
 */
export interface ClaimHistoryLink {
  /**
   * One of provisional_fact_id, challenged_claim_id, relations.contradicts.
   */
  field: string;
  targetId: string;
}
function createBaseQueryClaimHistoryRequest(): QueryClaimHistoryRequest {
  return {
    id: "",
    atBlockHeight: BigInt(0)
  };
}
/**
 * Reads retained records at the actual query context height. A nonzero requested
 * height must match that context; it does not reconstruct pruned state.
 * @name QueryClaimHistoryRequest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryRequest
 */
export const QueryClaimHistoryRequest = {
  typeUrl: "/zerone.knowledge.v1.QueryClaimHistoryRequest",
  encode(message: QueryClaimHistoryRequest, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.atBlockHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.atBlockHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): QueryClaimHistoryRequest {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryClaimHistoryRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.atBlockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<QueryClaimHistoryRequest>): QueryClaimHistoryRequest {
    const message = createBaseQueryClaimHistoryRequest();
    message.id = object.id ?? "";
    message.atBlockHeight = object.atBlockHeight !== undefined && object.atBlockHeight !== null ? BigInt(object.atBlockHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseQueryClaimHistoryResponse(): QueryClaimHistoryResponse {
  return {
    chainId: "",
    blockHeight: BigInt(0),
    record: undefined,
    relatedClaims: []
  };
}
/**
 * An observation of stored state, not a transaction inclusion proof or a
 * scientific verdict. The bounded query either returns its whole scope or fails.
 * @name QueryClaimHistoryResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryResponse
 */
export const QueryClaimHistoryResponse = {
  typeUrl: "/zerone.knowledge.v1.QueryClaimHistoryResponse",
  encode(message: QueryClaimHistoryResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.chainId !== "") {
      writer.uint32(10).string(message.chainId);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.blockHeight);
    }
    if (message.record !== undefined) {
      ClaimHistoryRecord.encode(message.record, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.relatedClaims) {
      RelatedClaimHistory.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): QueryClaimHistoryResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryClaimHistoryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainId = reader.string();
          break;
        case 2:
          message.blockHeight = reader.uint64();
          break;
        case 3:
          message.record = ClaimHistoryRecord.decode(reader, reader.uint32());
          break;
        case 4:
          message.relatedClaims.push(RelatedClaimHistory.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<QueryClaimHistoryResponse>): QueryClaimHistoryResponse {
    const message = createBaseQueryClaimHistoryResponse();
    message.chainId = object.chainId ?? "";
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    message.record = object.record !== undefined && object.record !== null ? ClaimHistoryRecord.fromPartial(object.record) : undefined;
    message.relatedClaims = object.relatedClaims?.map(e => RelatedClaimHistory.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimHistoryRecord(): ClaimHistoryRecord {
  return {
    claimId: "",
    claim: undefined,
    rounds: [],
    facts: [],
    missingRoundIds: []
  };
}
/**
 * Includes all retained rounds and derived facts for this claim ID. The claim
 * itself can be absent when surviving records refer to a historical missing row.
 * @name ClaimHistoryRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryRecord
 */
export const ClaimHistoryRecord = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryRecord",
  encode(message: ClaimHistoryRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimId !== "") {
      writer.uint32(10).string(message.claimId);
    }
    if (message.claim !== undefined) {
      Claim.encode(message.claim, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.rounds) {
      VerificationRound.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.facts) {
      ClaimHistoryFact.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.missingRoundIds) {
      writer.uint32(42).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimId = reader.string();
          break;
        case 2:
          message.claim = Claim.decode(reader, reader.uint32());
          break;
        case 3:
          message.rounds.push(VerificationRound.decode(reader, reader.uint32()));
          break;
        case 4:
          message.facts.push(ClaimHistoryFact.decode(reader, reader.uint32()));
          break;
        case 5:
          message.missingRoundIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimHistoryRecord>): ClaimHistoryRecord {
    const message = createBaseClaimHistoryRecord();
    message.claimId = object.claimId ?? "";
    message.claim = object.claim !== undefined && object.claim !== null ? Claim.fromPartial(object.claim) : undefined;
    message.rounds = object.rounds?.map(e => VerificationRound.fromPartial(e)) || [];
    message.facts = object.facts?.map(e => ClaimHistoryFact.fromPartial(e)) || [];
    message.missingRoundIds = object.missingRoundIds?.map(e => e) || [];
    return message;
  }
};
function createBaseClaimHistoryFact(): ClaimHistoryFact {
  return {
    fact: undefined,
    outgoingRelations: [],
    incomingRelations: [],
    statusTransitions: []
  };
}
/**
 * Relations are current canonical metadata; transitions are retained history.
 * Neighbor IDs do not expand the query into their fact bodies or descendants.
 * @name ClaimHistoryFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryFact
 */
export const ClaimHistoryFact = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryFact",
  encode(message: ClaimHistoryFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.fact !== undefined) {
      Fact.encode(message.fact, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.outgoingRelations) {
      FactRelation.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.incomingRelations) {
      FactRelation.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.statusTransitions) {
      StatusTransition.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.fact = Fact.decode(reader, reader.uint32());
          break;
        case 2:
          message.outgoingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 3:
          message.incomingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.statusTransitions.push(StatusTransition.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimHistoryFact>): ClaimHistoryFact {
    const message = createBaseClaimHistoryFact();
    message.fact = object.fact !== undefined && object.fact !== null ? Fact.fromPartial(object.fact) : undefined;
    message.outgoingRelations = object.outgoingRelations?.map(e => FactRelation.fromPartial(e)) || [];
    message.incomingRelations = object.incomingRelations?.map(e => FactRelation.fromPartial(e)) || [];
    message.statusTransitions = object.statusTransitions?.map(e => StatusTransition.fromPartial(e)) || [];
    return message;
  }
};
function createBaseRelatedClaimHistory(): RelatedClaimHistory {
  return {
    record: undefined,
    links: []
  };
}
/**
 * A single hop through an explicit stored claim field. This does not establish
 * causation, successful adjudication, credibility, or a completed payment.
 * @name RelatedClaimHistory
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RelatedClaimHistory
 */
export const RelatedClaimHistory = {
  typeUrl: "/zerone.knowledge.v1.RelatedClaimHistory",
  encode(message: RelatedClaimHistory, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.record !== undefined) {
      ClaimHistoryRecord.encode(message.record, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.links) {
      ClaimHistoryLink.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RelatedClaimHistory {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelatedClaimHistory();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.record = ClaimHistoryRecord.decode(reader, reader.uint32());
          break;
        case 2:
          message.links.push(ClaimHistoryLink.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RelatedClaimHistory>): RelatedClaimHistory {
    const message = createBaseRelatedClaimHistory();
    message.record = object.record !== undefined && object.record !== null ? ClaimHistoryRecord.fromPartial(object.record) : undefined;
    message.links = object.links?.map(e => ClaimHistoryLink.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimHistoryLink(): ClaimHistoryLink {
  return {
    field: "",
    targetId: ""
  };
}
/**
 * @name ClaimHistoryLink
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryLink
 */
export const ClaimHistoryLink = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryLink",
  encode(message: ClaimHistoryLink, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.field !== "") {
      writer.uint32(10).string(message.field);
    }
    if (message.targetId !== "") {
      writer.uint32(18).string(message.targetId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryLink {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryLink();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.field = reader.string();
          break;
        case 2:
          message.targetId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimHistoryLink>): ClaimHistoryLink {
    const message = createBaseClaimHistoryLink();
    message.field = object.field ?? "";
    message.targetId = object.targetId ?? "";
    return message;
  }
};