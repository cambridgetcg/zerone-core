import { AxisProjection, ExternalSource, CitationType } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * SubstrateLink commits caller-declared external provenance to ToK fact IDs.
 * cited_facts MUST exist in x/knowledge at commit time.
 * pending_claims is a reserved future integration surface: current
 * MsgSubmitExternalAttestation rejects a nonempty list because translation
 * into x/knowledge is not wired.
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
     * sha256 of canonical declared fields
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
 * PendingClaim is the reserved shape for a future x/knowledge translation.
 * It is not accepted by current public attestation submission. When that
 * integration is implemented, claim_relations may cite existing facts only;
 * they are not recursive pending claims.
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
    /**
     * reserved path; 10,000 scale
     */
    inferenceStrengthBps: number;
}
/**
 * SubstrateLink commits caller-declared external provenance to ToK fact IDs.
 * cited_facts MUST exist in x/knowledge at commit time.
 * pending_claims is a reserved future integration surface: current
 * MsgSubmitExternalAttestation rejects a nonempty list because translation
 * into x/knowledge is not wired.
 * @name SubstrateLink
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SubstrateLink
 */
export declare const SubstrateLink: {
    typeUrl: string;
    encode(message: SubstrateLink, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SubstrateLink;
    fromPartial(object: DeepPartial<SubstrateLink>): SubstrateLink;
};
/**
 * FactCitation is one outgoing edge in the substrate-link. citation_type
 * drives lineage propagation weight (M6).
 * @name FactCitation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.FactCitation
 */
export declare const FactCitation: {
    typeUrl: string;
    encode(message: FactCitation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): FactCitation;
    fromPartial(object: DeepPartial<FactCitation>): FactCitation;
};
/**
 * PendingClaim is the reserved shape for a future x/knowledge translation.
 * It is not accepted by current public attestation submission. When that
 * integration is implemented, claim_relations may cite existing facts only;
 * they are not recursive pending claims.
 * @name PendingClaim
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.PendingClaim
 */
export declare const PendingClaim: {
    typeUrl: string;
    encode(message: PendingClaim, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PendingClaim;
    fromPartial(object: DeepPartial<PendingClaim>): PendingClaim;
};
/**
 * ClaimRelation is a citation from a pending claim to an existing fact.
 * Mirrors x/knowledge.ClaimRelation. Pending claims cannot cite OTHER
 * pending claims — they cite existing verified facts only. This keeps
 * the resolution graph a tree (one-hop deferral).
 * @name ClaimRelation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ClaimRelation
 */
export declare const ClaimRelation: {
    typeUrl: string;
    encode(message: ClaimRelation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimRelation;
    fromPartial(object: DeepPartial<ClaimRelation>): ClaimRelation;
};
