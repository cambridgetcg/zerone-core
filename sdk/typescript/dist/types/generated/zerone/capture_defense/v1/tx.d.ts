import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgRecordVerification
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerification
 */
export interface MsgRecordVerification {
    authority: string;
    domain: string;
    roundId: string;
    validators: string[];
    verdicts: boolean[];
    submitBlocks: bigint[];
}
/**
 * @name MsgRecordVerificationResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerificationResponse
 */
export interface MsgRecordVerificationResponse {
}
/**
 * @name MsgAnalyzeDomain
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomain
 */
export interface MsgAnalyzeDomain {
    sender: string;
    domain: string;
}
/**
 * @name MsgAnalyzeDomainResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomainResponse
 */
export interface MsgAnalyzeDomainResponse {
    riskScore: bigint;
    flagged: boolean;
}
/**
 * @name MsgUpdateParams
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgRecordVerification
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerification
 */
export declare const MsgRecordVerification: {
    typeUrl: string;
    encode(message: MsgRecordVerification, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordVerification;
    fromPartial(object: DeepPartial<MsgRecordVerification>): MsgRecordVerification;
};
/**
 * @name MsgRecordVerificationResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgRecordVerificationResponse
 */
export declare const MsgRecordVerificationResponse: {
    typeUrl: string;
    encode(_: MsgRecordVerificationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordVerificationResponse;
    fromPartial(_: DeepPartial<MsgRecordVerificationResponse>): MsgRecordVerificationResponse;
};
/**
 * @name MsgAnalyzeDomain
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomain
 */
export declare const MsgAnalyzeDomain: {
    typeUrl: string;
    encode(message: MsgAnalyzeDomain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAnalyzeDomain;
    fromPartial(object: DeepPartial<MsgAnalyzeDomain>): MsgAnalyzeDomain;
};
/**
 * @name MsgAnalyzeDomainResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgAnalyzeDomainResponse
 */
export declare const MsgAnalyzeDomainResponse: {
    typeUrl: string;
    encode(message: MsgAnalyzeDomainResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAnalyzeDomainResponse;
    fromPartial(object: DeepPartial<MsgAnalyzeDomainResponse>): MsgAnalyzeDomainResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
