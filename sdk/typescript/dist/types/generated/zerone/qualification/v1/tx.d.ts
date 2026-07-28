import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgQualifyByStake
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStake
 */
export interface MsgQualifyByStake {
    validator: string;
    domain: string;
    /**
     * uzrn
     */
    stakeAmount: string;
}
/**
 * @name MsgQualifyByStakeResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStakeResponse
 */
export interface MsgQualifyByStakeResponse {
}
/**
 * @name MsgQualifyByTrackRecord
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecord
 */
export interface MsgQualifyByTrackRecord {
    validator: string;
    domain: string;
}
/**
 * @name MsgQualifyByTrackRecordResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecordResponse
 */
export interface MsgQualifyByTrackRecordResponse {
}
/**
 * @name MsgQualifyByCrossReference
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReference
 */
export interface MsgQualifyByCrossReference {
    validator: string;
    targetDomain: string;
    sourceDomain: string;
}
/**
 * @name MsgQualifyByCrossReferenceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReferenceResponse
 */
export interface MsgQualifyByCrossReferenceResponse {
}
/**
 * @name MsgQualifyByInheritance
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritance
 */
export interface MsgQualifyByInheritance {
    validator: string;
    targetDomain: string;
    parentDomain: string;
}
/**
 * @name MsgQualifyByInheritanceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritanceResponse
 */
export interface MsgQualifyByInheritanceResponse {
}
/**
 * @name MsgEndorseQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualification
 */
export interface MsgEndorseQualification {
    endorser: string;
    validator: string;
    domain: string;
    reason: string;
    weight: number;
}
/**
 * @name MsgEndorseQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualificationResponse
 */
export interface MsgEndorseQualificationResponse {
    endorsementId: bigint;
}
/**
 * @name MsgRenewQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualification
 */
export interface MsgRenewQualification {
    validator: string;
    domain: string;
}
/**
 * @name MsgRenewQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualificationResponse
 */
export interface MsgRenewQualificationResponse {
}
/**
 * @name MsgWithdrawQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualification
 */
export interface MsgWithdrawQualification {
    validator: string;
    domain: string;
}
/**
 * @name MsgWithdrawQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualificationResponse
 */
export interface MsgWithdrawQualificationResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgQualifyByStake
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStake
 */
export declare const MsgQualifyByStake: {
    typeUrl: string;
    encode(message: MsgQualifyByStake, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByStake;
    fromPartial(object: DeepPartial<MsgQualifyByStake>): MsgQualifyByStake;
};
/**
 * @name MsgQualifyByStakeResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByStakeResponse
 */
export declare const MsgQualifyByStakeResponse: {
    typeUrl: string;
    encode(_: MsgQualifyByStakeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByStakeResponse;
    fromPartial(_: DeepPartial<MsgQualifyByStakeResponse>): MsgQualifyByStakeResponse;
};
/**
 * @name MsgQualifyByTrackRecord
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecord
 */
export declare const MsgQualifyByTrackRecord: {
    typeUrl: string;
    encode(message: MsgQualifyByTrackRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByTrackRecord;
    fromPartial(object: DeepPartial<MsgQualifyByTrackRecord>): MsgQualifyByTrackRecord;
};
/**
 * @name MsgQualifyByTrackRecordResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByTrackRecordResponse
 */
export declare const MsgQualifyByTrackRecordResponse: {
    typeUrl: string;
    encode(_: MsgQualifyByTrackRecordResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByTrackRecordResponse;
    fromPartial(_: DeepPartial<MsgQualifyByTrackRecordResponse>): MsgQualifyByTrackRecordResponse;
};
/**
 * @name MsgQualifyByCrossReference
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReference
 */
export declare const MsgQualifyByCrossReference: {
    typeUrl: string;
    encode(message: MsgQualifyByCrossReference, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByCrossReference;
    fromPartial(object: DeepPartial<MsgQualifyByCrossReference>): MsgQualifyByCrossReference;
};
/**
 * @name MsgQualifyByCrossReferenceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByCrossReferenceResponse
 */
export declare const MsgQualifyByCrossReferenceResponse: {
    typeUrl: string;
    encode(_: MsgQualifyByCrossReferenceResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByCrossReferenceResponse;
    fromPartial(_: DeepPartial<MsgQualifyByCrossReferenceResponse>): MsgQualifyByCrossReferenceResponse;
};
/**
 * @name MsgQualifyByInheritance
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritance
 */
export declare const MsgQualifyByInheritance: {
    typeUrl: string;
    encode(message: MsgQualifyByInheritance, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByInheritance;
    fromPartial(object: DeepPartial<MsgQualifyByInheritance>): MsgQualifyByInheritance;
};
/**
 * @name MsgQualifyByInheritanceResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgQualifyByInheritanceResponse
 */
export declare const MsgQualifyByInheritanceResponse: {
    typeUrl: string;
    encode(_: MsgQualifyByInheritanceResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgQualifyByInheritanceResponse;
    fromPartial(_: DeepPartial<MsgQualifyByInheritanceResponse>): MsgQualifyByInheritanceResponse;
};
/**
 * @name MsgEndorseQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualification
 */
export declare const MsgEndorseQualification: {
    typeUrl: string;
    encode(message: MsgEndorseQualification, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseQualification;
    fromPartial(object: DeepPartial<MsgEndorseQualification>): MsgEndorseQualification;
};
/**
 * @name MsgEndorseQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgEndorseQualificationResponse
 */
export declare const MsgEndorseQualificationResponse: {
    typeUrl: string;
    encode(message: MsgEndorseQualificationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseQualificationResponse;
    fromPartial(object: DeepPartial<MsgEndorseQualificationResponse>): MsgEndorseQualificationResponse;
};
/**
 * @name MsgRenewQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualification
 */
export declare const MsgRenewQualification: {
    typeUrl: string;
    encode(message: MsgRenewQualification, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRenewQualification;
    fromPartial(object: DeepPartial<MsgRenewQualification>): MsgRenewQualification;
};
/**
 * @name MsgRenewQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgRenewQualificationResponse
 */
export declare const MsgRenewQualificationResponse: {
    typeUrl: string;
    encode(_: MsgRenewQualificationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRenewQualificationResponse;
    fromPartial(_: DeepPartial<MsgRenewQualificationResponse>): MsgRenewQualificationResponse;
};
/**
 * @name MsgWithdrawQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualification
 */
export declare const MsgWithdrawQualification: {
    typeUrl: string;
    encode(message: MsgWithdrawQualification, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawQualification;
    fromPartial(object: DeepPartial<MsgWithdrawQualification>): MsgWithdrawQualification;
};
/**
 * @name MsgWithdrawQualificationResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgWithdrawQualificationResponse
 */
export declare const MsgWithdrawQualificationResponse: {
    typeUrl: string;
    encode(_: MsgWithdrawQualificationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawQualificationResponse;
    fromPartial(_: DeepPartial<MsgWithdrawQualificationResponse>): MsgWithdrawQualificationResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
