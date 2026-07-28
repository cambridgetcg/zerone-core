import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** EmergencyCategory classifies the emergency reason. */
export declare enum EmergencyCategory {
    EMERGENCY_CATEGORY_UNSPECIFIED = 0,
    EMERGENCY_CATEGORY_SECURITY_BREACH = 1,
    EMERGENCY_CATEGORY_CONSENSUS_FAILURE = 2,
    EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT = 3,
    EMERGENCY_CATEGORY_STATE_CORRUPTION = 4,
    UNRECOGNIZED = -1
}
export declare function emergencyCategoryFromJSON(object: any): EmergencyCategory;
export declare function emergencyCategoryToJSON(object: EmergencyCategory): string;
/**
 * @name MsgProposeHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHalt
 */
export interface MsgProposeHalt {
    proposer: string;
    reason: string;
    category: EmergencyCategory;
}
/**
 * @name MsgProposeHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHaltResponse
 */
export interface MsgProposeHaltResponse {
    proposalId: string;
}
/**
 * @name MsgVoteHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHalt
 */
export interface MsgVoteHalt {
    voter: string;
    proposalId: string;
    approve: boolean;
}
/**
 * @name MsgVoteHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHaltResponse
 */
export interface MsgVoteHaltResponse {
    quorumReached: boolean;
    chainHalted: boolean;
}
/**
 * @name MsgProposeRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevert
 */
export interface MsgProposeRevert {
    proposer: string;
    revertToHeight: bigint;
    justification: string;
}
/**
 * @name MsgProposeRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevertResponse
 */
export interface MsgProposeRevertResponse {
    proposalId: string;
}
/**
 * @name MsgVoteRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevert
 */
export interface MsgVoteRevert {
    voter: string;
    proposalId: string;
    approve: boolean;
}
/**
 * @name MsgVoteRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevertResponse
 */
export interface MsgVoteRevertResponse {
    quorumReached: boolean;
    revertExecuted: boolean;
}
/**
 * @name MsgProposeResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResume
 */
export interface MsgProposeResume {
    proposer: string;
    justification: string;
}
/**
 * @name MsgProposeResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResumeResponse
 */
export interface MsgProposeResumeResponse {
    proposalId: string;
}
/**
 * @name MsgVoteResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResume
 */
export interface MsgVoteResume {
    voter: string;
    proposalId: string;
    approve: boolean;
}
/**
 * @name MsgVoteResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResumeResponse
 */
export interface MsgVoteResumeResponse {
    quorumReached: boolean;
    chainResumed: boolean;
}
/**
 * @name MsgUpdateParams
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgProposeHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHalt
 */
export declare const MsgProposeHalt: {
    typeUrl: string;
    encode(message: MsgProposeHalt, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeHalt;
    fromPartial(object: DeepPartial<MsgProposeHalt>): MsgProposeHalt;
};
/**
 * @name MsgProposeHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHaltResponse
 */
export declare const MsgProposeHaltResponse: {
    typeUrl: string;
    encode(message: MsgProposeHaltResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeHaltResponse;
    fromPartial(object: DeepPartial<MsgProposeHaltResponse>): MsgProposeHaltResponse;
};
/**
 * @name MsgVoteHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHalt
 */
export declare const MsgVoteHalt: {
    typeUrl: string;
    encode(message: MsgVoteHalt, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteHalt;
    fromPartial(object: DeepPartial<MsgVoteHalt>): MsgVoteHalt;
};
/**
 * @name MsgVoteHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHaltResponse
 */
export declare const MsgVoteHaltResponse: {
    typeUrl: string;
    encode(message: MsgVoteHaltResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteHaltResponse;
    fromPartial(object: DeepPartial<MsgVoteHaltResponse>): MsgVoteHaltResponse;
};
/**
 * @name MsgProposeRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevert
 */
export declare const MsgProposeRevert: {
    typeUrl: string;
    encode(message: MsgProposeRevert, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRevert;
    fromPartial(object: DeepPartial<MsgProposeRevert>): MsgProposeRevert;
};
/**
 * @name MsgProposeRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevertResponse
 */
export declare const MsgProposeRevertResponse: {
    typeUrl: string;
    encode(message: MsgProposeRevertResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRevertResponse;
    fromPartial(object: DeepPartial<MsgProposeRevertResponse>): MsgProposeRevertResponse;
};
/**
 * @name MsgVoteRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevert
 */
export declare const MsgVoteRevert: {
    typeUrl: string;
    encode(message: MsgVoteRevert, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRevert;
    fromPartial(object: DeepPartial<MsgVoteRevert>): MsgVoteRevert;
};
/**
 * @name MsgVoteRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevertResponse
 */
export declare const MsgVoteRevertResponse: {
    typeUrl: string;
    encode(message: MsgVoteRevertResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRevertResponse;
    fromPartial(object: DeepPartial<MsgVoteRevertResponse>): MsgVoteRevertResponse;
};
/**
 * @name MsgProposeResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResume
 */
export declare const MsgProposeResume: {
    typeUrl: string;
    encode(message: MsgProposeResume, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResume;
    fromPartial(object: DeepPartial<MsgProposeResume>): MsgProposeResume;
};
/**
 * @name MsgProposeResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResumeResponse
 */
export declare const MsgProposeResumeResponse: {
    typeUrl: string;
    encode(message: MsgProposeResumeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResumeResponse;
    fromPartial(object: DeepPartial<MsgProposeResumeResponse>): MsgProposeResumeResponse;
};
/**
 * @name MsgVoteResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResume
 */
export declare const MsgVoteResume: {
    typeUrl: string;
    encode(message: MsgVoteResume, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResume;
    fromPartial(object: DeepPartial<MsgVoteResume>): MsgVoteResume;
};
/**
 * @name MsgVoteResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResumeResponse
 */
export declare const MsgVoteResumeResponse: {
    typeUrl: string;
    encode(message: MsgVoteResumeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResumeResponse;
    fromPartial(object: DeepPartial<MsgVoteResumeResponse>): MsgVoteResumeResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
