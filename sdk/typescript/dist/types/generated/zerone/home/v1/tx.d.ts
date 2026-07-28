import { HomeGuardian, DeadmanConfig } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgCreateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHome
 */
export interface MsgCreateHome {
    owner: string;
    name: string;
    initialGuardianConfig?: HomeGuardian;
}
/**
 * @name MsgCreateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHomeResponse
 */
export interface MsgCreateHomeResponse {
    homeId: string;
}
/**
 * @name MsgUpdateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHome
 */
export interface MsgUpdateHome {
    owner: string;
    homeId: string;
    name: string;
    status: string;
}
/**
 * @name MsgUpdateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHomeResponse
 */
export interface MsgUpdateHomeResponse {
}
/**
 * @name MsgUpdateMemoryCID
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCID
 */
export interface MsgUpdateMemoryCID {
    owner: string;
    homeId: string;
    cid: string;
}
/**
 * @name MsgUpdateMemoryCIDResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCIDResponse
 */
export interface MsgUpdateMemoryCIDResponse {
}
/**
 * @name MsgStartSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSession
 */
export interface MsgStartSession {
    signer: string;
    homeId: string;
    keyHash: string;
    requestedPermissions: string[];
}
/**
 * @name MsgStartSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSessionResponse
 */
export interface MsgStartSessionResponse {
    sessionId: string;
}
/**
 * @name MsgEndSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSession
 */
export interface MsgEndSession {
    signer: string;
    homeId: string;
    sessionId: string;
}
/**
 * @name MsgEndSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSessionResponse
 */
export interface MsgEndSessionResponse {
}
/**
 * @name MsgRegisterKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKey
 */
export interface MsgRegisterKey {
    owner: string;
    homeId: string;
    keyHash: string;
    keyType: string;
    role: string;
    permissions: string[];
    expiresAt: bigint;
}
/**
 * @name MsgRegisterKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKeyResponse
 */
export interface MsgRegisterKeyResponse {
}
/**
 * @name MsgRevokeKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKey
 */
export interface MsgRevokeKey {
    owner: string;
    homeId: string;
    keyHash: string;
}
/**
 * @name MsgRevokeKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKeyResponse
 */
export interface MsgRevokeKeyResponse {
}
/**
 * @name MsgConfigureGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardian
 */
export interface MsgConfigureGuardian {
    owner: string;
    homeId: string;
    defenseStrategy: string;
    autoDefend: boolean;
    deadman?: DeadmanConfig;
    recoveryAddresses: string[];
    recoveryThreshold: number;
    guardianAddress: string;
}
/**
 * @name MsgConfigureGuardianResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardianResponse
 */
export interface MsgConfigureGuardianResponse {
}
/**
 * @name MsgAcknowledgeAlert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlert
 */
export interface MsgAcknowledgeAlert {
    signer: string;
    homeId: string;
    alertId: string;
}
/**
 * @name MsgAcknowledgeAlertResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlertResponse
 */
export interface MsgAcknowledgeAlertResponse {
}
/**
 * @name MsgSetSpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimit
 */
export interface MsgSetSpendingLimit {
    owner: string;
    homeId: string;
    keyType: string;
    maxAmount: string;
    periodBlocks: bigint;
}
/**
 * @name MsgSetSpendingLimitResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimitResponse
 */
export interface MsgSetSpendingLimitResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgCreateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHome
 */
export declare const MsgCreateHome: {
    typeUrl: string;
    encode(message: MsgCreateHome, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateHome;
    fromPartial(object: DeepPartial<MsgCreateHome>): MsgCreateHome;
};
/**
 * @name MsgCreateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHomeResponse
 */
export declare const MsgCreateHomeResponse: {
    typeUrl: string;
    encode(message: MsgCreateHomeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateHomeResponse;
    fromPartial(object: DeepPartial<MsgCreateHomeResponse>): MsgCreateHomeResponse;
};
/**
 * @name MsgUpdateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHome
 */
export declare const MsgUpdateHome: {
    typeUrl: string;
    encode(message: MsgUpdateHome, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateHome;
    fromPartial(object: DeepPartial<MsgUpdateHome>): MsgUpdateHome;
};
/**
 * @name MsgUpdateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHomeResponse
 */
export declare const MsgUpdateHomeResponse: {
    typeUrl: string;
    encode(_: MsgUpdateHomeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateHomeResponse;
    fromPartial(_: DeepPartial<MsgUpdateHomeResponse>): MsgUpdateHomeResponse;
};
/**
 * @name MsgUpdateMemoryCID
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCID
 */
export declare const MsgUpdateMemoryCID: {
    typeUrl: string;
    encode(message: MsgUpdateMemoryCID, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateMemoryCID;
    fromPartial(object: DeepPartial<MsgUpdateMemoryCID>): MsgUpdateMemoryCID;
};
/**
 * @name MsgUpdateMemoryCIDResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCIDResponse
 */
export declare const MsgUpdateMemoryCIDResponse: {
    typeUrl: string;
    encode(_: MsgUpdateMemoryCIDResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateMemoryCIDResponse;
    fromPartial(_: DeepPartial<MsgUpdateMemoryCIDResponse>): MsgUpdateMemoryCIDResponse;
};
/**
 * @name MsgStartSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSession
 */
export declare const MsgStartSession: {
    typeUrl: string;
    encode(message: MsgStartSession, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgStartSession;
    fromPartial(object: DeepPartial<MsgStartSession>): MsgStartSession;
};
/**
 * @name MsgStartSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSessionResponse
 */
export declare const MsgStartSessionResponse: {
    typeUrl: string;
    encode(message: MsgStartSessionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgStartSessionResponse;
    fromPartial(object: DeepPartial<MsgStartSessionResponse>): MsgStartSessionResponse;
};
/**
 * @name MsgEndSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSession
 */
export declare const MsgEndSession: {
    typeUrl: string;
    encode(message: MsgEndSession, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndSession;
    fromPartial(object: DeepPartial<MsgEndSession>): MsgEndSession;
};
/**
 * @name MsgEndSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSessionResponse
 */
export declare const MsgEndSessionResponse: {
    typeUrl: string;
    encode(_: MsgEndSessionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndSessionResponse;
    fromPartial(_: DeepPartial<MsgEndSessionResponse>): MsgEndSessionResponse;
};
/**
 * @name MsgRegisterKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKey
 */
export declare const MsgRegisterKey: {
    typeUrl: string;
    encode(message: MsgRegisterKey, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterKey;
    fromPartial(object: DeepPartial<MsgRegisterKey>): MsgRegisterKey;
};
/**
 * @name MsgRegisterKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKeyResponse
 */
export declare const MsgRegisterKeyResponse: {
    typeUrl: string;
    encode(_: MsgRegisterKeyResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterKeyResponse;
    fromPartial(_: DeepPartial<MsgRegisterKeyResponse>): MsgRegisterKeyResponse;
};
/**
 * @name MsgRevokeKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKey
 */
export declare const MsgRevokeKey: {
    typeUrl: string;
    encode(message: MsgRevokeKey, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRevokeKey;
    fromPartial(object: DeepPartial<MsgRevokeKey>): MsgRevokeKey;
};
/**
 * @name MsgRevokeKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKeyResponse
 */
export declare const MsgRevokeKeyResponse: {
    typeUrl: string;
    encode(_: MsgRevokeKeyResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRevokeKeyResponse;
    fromPartial(_: DeepPartial<MsgRevokeKeyResponse>): MsgRevokeKeyResponse;
};
/**
 * @name MsgConfigureGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardian
 */
export declare const MsgConfigureGuardian: {
    typeUrl: string;
    encode(message: MsgConfigureGuardian, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgConfigureGuardian;
    fromPartial(object: DeepPartial<MsgConfigureGuardian>): MsgConfigureGuardian;
};
/**
 * @name MsgConfigureGuardianResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardianResponse
 */
export declare const MsgConfigureGuardianResponse: {
    typeUrl: string;
    encode(_: MsgConfigureGuardianResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgConfigureGuardianResponse;
    fromPartial(_: DeepPartial<MsgConfigureGuardianResponse>): MsgConfigureGuardianResponse;
};
/**
 * @name MsgAcknowledgeAlert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlert
 */
export declare const MsgAcknowledgeAlert: {
    typeUrl: string;
    encode(message: MsgAcknowledgeAlert, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeAlert;
    fromPartial(object: DeepPartial<MsgAcknowledgeAlert>): MsgAcknowledgeAlert;
};
/**
 * @name MsgAcknowledgeAlertResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlertResponse
 */
export declare const MsgAcknowledgeAlertResponse: {
    typeUrl: string;
    encode(_: MsgAcknowledgeAlertResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeAlertResponse;
    fromPartial(_: DeepPartial<MsgAcknowledgeAlertResponse>): MsgAcknowledgeAlertResponse;
};
/**
 * @name MsgSetSpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimit
 */
export declare const MsgSetSpendingLimit: {
    typeUrl: string;
    encode(message: MsgSetSpendingLimit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSetSpendingLimit;
    fromPartial(object: DeepPartial<MsgSetSpendingLimit>): MsgSetSpendingLimit;
};
/**
 * @name MsgSetSpendingLimitResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimitResponse
 */
export declare const MsgSetSpendingLimitResponse: {
    typeUrl: string;
    encode(_: MsgSetSpendingLimitResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSetSpendingLimitResponse;
    fromPartial(_: DeepPartial<MsgSetSpendingLimitResponse>): MsgSetSpendingLimitResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
