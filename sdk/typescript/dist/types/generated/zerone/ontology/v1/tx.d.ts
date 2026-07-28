import { LogicZoneProperties } from "./state.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgProposeDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomain
 */
export interface MsgProposeDomain {
    proposer: string;
    name: string;
    displayName: string;
    description: string;
    stratum: number;
    /**
     * bigint as string (uzrn)
     */
    stake: string;
}
/**
 * @name MsgProposeDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomainResponse
 */
export interface MsgProposeDomainResponse {
    proposalId: string;
}
/**
 * @name MsgVoteDomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposal
 */
export interface MsgVoteDomainProposal {
    voter: string;
    proposalId: string;
    approve: boolean;
}
/**
 * @name MsgVoteDomainProposalResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposalResponse
 */
export interface MsgVoteDomainProposalResponse {
}
/**
 * @name MsgUpdateDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomain
 */
export interface MsgUpdateDomain {
    authority: string;
    domainName: string;
    displayName: string;
    description: string;
    status: string;
}
/**
 * @name MsgUpdateDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomainResponse
 */
export interface MsgUpdateDomainResponse {
}
/**
 * @name MsgRegisterLogicZone
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZone
 */
export interface MsgRegisterLogicZone {
    authority: string;
    zoneProperties?: LogicZoneProperties;
}
/**
 * @name MsgRegisterLogicZoneResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZoneResponse
 */
export interface MsgRegisterLogicZoneResponse {
}
/**
 * @name MsgAcknowledgeIncompleteness
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompleteness
 */
export interface MsgAcknowledgeIncompleteness {
    submitter: string;
    factId: string;
    zone: string;
    reason: string;
}
/**
 * @name MsgAcknowledgeIncompletenessResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse
 */
export interface MsgAcknowledgeIncompletenessResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgProposeDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomain
 */
export declare const MsgProposeDomain: {
    typeUrl: string;
    encode(message: MsgProposeDomain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomain;
    fromPartial(object: DeepPartial<MsgProposeDomain>): MsgProposeDomain;
};
/**
 * @name MsgProposeDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomainResponse
 */
export declare const MsgProposeDomainResponse: {
    typeUrl: string;
    encode(message: MsgProposeDomainResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomainResponse;
    fromPartial(object: DeepPartial<MsgProposeDomainResponse>): MsgProposeDomainResponse;
};
/**
 * @name MsgVoteDomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposal
 */
export declare const MsgVoteDomainProposal: {
    typeUrl: string;
    encode(message: MsgVoteDomainProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteDomainProposal;
    fromPartial(object: DeepPartial<MsgVoteDomainProposal>): MsgVoteDomainProposal;
};
/**
 * @name MsgVoteDomainProposalResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposalResponse
 */
export declare const MsgVoteDomainProposalResponse: {
    typeUrl: string;
    encode(_: MsgVoteDomainProposalResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteDomainProposalResponse;
    fromPartial(_: DeepPartial<MsgVoteDomainProposalResponse>): MsgVoteDomainProposalResponse;
};
/**
 * @name MsgUpdateDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomain
 */
export declare const MsgUpdateDomain: {
    typeUrl: string;
    encode(message: MsgUpdateDomain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateDomain;
    fromPartial(object: DeepPartial<MsgUpdateDomain>): MsgUpdateDomain;
};
/**
 * @name MsgUpdateDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomainResponse
 */
export declare const MsgUpdateDomainResponse: {
    typeUrl: string;
    encode(_: MsgUpdateDomainResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateDomainResponse;
    fromPartial(_: DeepPartial<MsgUpdateDomainResponse>): MsgUpdateDomainResponse;
};
/**
 * @name MsgRegisterLogicZone
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZone
 */
export declare const MsgRegisterLogicZone: {
    typeUrl: string;
    encode(message: MsgRegisterLogicZone, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterLogicZone;
    fromPartial(object: DeepPartial<MsgRegisterLogicZone>): MsgRegisterLogicZone;
};
/**
 * @name MsgRegisterLogicZoneResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZoneResponse
 */
export declare const MsgRegisterLogicZoneResponse: {
    typeUrl: string;
    encode(_: MsgRegisterLogicZoneResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterLogicZoneResponse;
    fromPartial(_: DeepPartial<MsgRegisterLogicZoneResponse>): MsgRegisterLogicZoneResponse;
};
/**
 * @name MsgAcknowledgeIncompleteness
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompleteness
 */
export declare const MsgAcknowledgeIncompleteness: {
    typeUrl: string;
    encode(message: MsgAcknowledgeIncompleteness, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeIncompleteness;
    fromPartial(object: DeepPartial<MsgAcknowledgeIncompleteness>): MsgAcknowledgeIncompleteness;
};
/**
 * @name MsgAcknowledgeIncompletenessResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse
 */
export declare const MsgAcknowledgeIncompletenessResponse: {
    typeUrl: string;
    encode(_: MsgAcknowledgeIncompletenessResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeIncompletenessResponse;
    fromPartial(_: DeepPartial<MsgAcknowledgeIncompletenessResponse>): MsgAcknowledgeIncompletenessResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
