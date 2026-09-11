import { Claim, VerificationRound, Fact, FactRelation } from "./types.js";
import { StatusTransition } from "./tok_cascade.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * Reads retained records at the actual query context height. A nonzero requested
 * height must match that context; it does not reconstruct pruned state.
 * @name QueryClaimHistoryRequest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryRequest
 */
export declare const QueryClaimHistoryRequest: {
    typeUrl: string;
    encode(message: QueryClaimHistoryRequest, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): QueryClaimHistoryRequest;
    fromPartial(object: DeepPartial<QueryClaimHistoryRequest>): QueryClaimHistoryRequest;
};
/**
 * An observation of stored state, not a transaction inclusion proof or a
 * scientific verdict. The bounded query either returns its whole scope or fails.
 * @name QueryClaimHistoryResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.QueryClaimHistoryResponse
 */
export declare const QueryClaimHistoryResponse: {
    typeUrl: string;
    encode(message: QueryClaimHistoryResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): QueryClaimHistoryResponse;
    fromPartial(object: DeepPartial<QueryClaimHistoryResponse>): QueryClaimHistoryResponse;
};
/**
 * Includes all retained rounds and derived facts for this claim ID. The claim
 * itself can be absent when surviving records refer to a historical missing row.
 * @name ClaimHistoryRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryRecord
 */
export declare const ClaimHistoryRecord: {
    typeUrl: string;
    encode(message: ClaimHistoryRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryRecord;
    fromPartial(object: DeepPartial<ClaimHistoryRecord>): ClaimHistoryRecord;
};
/**
 * Relations are current canonical metadata; transitions are retained history.
 * Neighbor IDs do not expand the query into their fact bodies or descendants.
 * @name ClaimHistoryFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryFact
 */
export declare const ClaimHistoryFact: {
    typeUrl: string;
    encode(message: ClaimHistoryFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryFact;
    fromPartial(object: DeepPartial<ClaimHistoryFact>): ClaimHistoryFact;
};
/**
 * A single hop through an explicit stored claim field. This does not establish
 * causation, successful adjudication, credibility, or a completed payment.
 * @name RelatedClaimHistory
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RelatedClaimHistory
 */
export declare const RelatedClaimHistory: {
    typeUrl: string;
    encode(message: RelatedClaimHistory, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RelatedClaimHistory;
    fromPartial(object: DeepPartial<RelatedClaimHistory>): RelatedClaimHistory;
};
/**
 * @name ClaimHistoryLink
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimHistoryLink
 */
export declare const ClaimHistoryLink: {
    typeUrl: string;
    encode(message: ClaimHistoryLink, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimHistoryLink;
    fromPartial(object: DeepPartial<ClaimHistoryLink>): ClaimHistoryLink;
};
