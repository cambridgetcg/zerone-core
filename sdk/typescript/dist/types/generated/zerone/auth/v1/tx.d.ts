import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * MsgRegisterAccount registers a new Zerone account with DID mapping.
 * @name MsgRegisterAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccount
 */
export interface MsgRegisterAccount {
    sender: string;
    did: string;
    publicKey: string;
    accountType: string;
    operationalKeyHash: string;
    metadata: string;
}
/**
 * @name MsgRegisterAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccountResponse
 */
export interface MsgRegisterAccountResponse {
}
/**
 * MsgRotateKey rotates the operational key for an account.
 * @name MsgRotateKey
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKey
 */
export interface MsgRotateKey {
    sender: string;
    newOperationalKey: Uint8Array;
    authorizationSignature: Uint8Array;
}
/**
 * @name MsgRotateKeyResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKeyResponse
 */
export interface MsgRotateKeyResponse {
    newKeyVersion: number;
}
/**
 * MsgFreezeAccount freezes an account.
 * @name MsgFreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccount
 */
export interface MsgFreezeAccount {
    sender: string;
    address: string;
    reason: string;
}
/**
 * @name MsgFreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccountResponse
 */
export interface MsgFreezeAccountResponse {
}
/**
 * MsgUnfreezeAccount unfreezes a frozen account.
 * @name MsgUnfreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccount
 */
export interface MsgUnfreezeAccount {
    authority: string;
    address: string;
}
/**
 * @name MsgUnfreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccountResponse
 */
export interface MsgUnfreezeAccountResponse {
}
/**
 * MsgUpdateParams updates auth module parameters.
 * @name MsgUpdateParams
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * MsgRegisterAccount registers a new Zerone account with DID mapping.
 * @name MsgRegisterAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccount
 */
export declare const MsgRegisterAccount: {
    typeUrl: string;
    encode(message: MsgRegisterAccount, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAccount;
    fromPartial(object: DeepPartial<MsgRegisterAccount>): MsgRegisterAccount;
};
/**
 * @name MsgRegisterAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccountResponse
 */
export declare const MsgRegisterAccountResponse: {
    typeUrl: string;
    encode(_: MsgRegisterAccountResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAccountResponse;
    fromPartial(_: DeepPartial<MsgRegisterAccountResponse>): MsgRegisterAccountResponse;
};
/**
 * MsgRotateKey rotates the operational key for an account.
 * @name MsgRotateKey
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKey
 */
export declare const MsgRotateKey: {
    typeUrl: string;
    encode(message: MsgRotateKey, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRotateKey;
    fromPartial(object: DeepPartial<MsgRotateKey>): MsgRotateKey;
};
/**
 * @name MsgRotateKeyResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKeyResponse
 */
export declare const MsgRotateKeyResponse: {
    typeUrl: string;
    encode(message: MsgRotateKeyResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRotateKeyResponse;
    fromPartial(object: DeepPartial<MsgRotateKeyResponse>): MsgRotateKeyResponse;
};
/**
 * MsgFreezeAccount freezes an account.
 * @name MsgFreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccount
 */
export declare const MsgFreezeAccount: {
    typeUrl: string;
    encode(message: MsgFreezeAccount, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFreezeAccount;
    fromPartial(object: DeepPartial<MsgFreezeAccount>): MsgFreezeAccount;
};
/**
 * @name MsgFreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccountResponse
 */
export declare const MsgFreezeAccountResponse: {
    typeUrl: string;
    encode(_: MsgFreezeAccountResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFreezeAccountResponse;
    fromPartial(_: DeepPartial<MsgFreezeAccountResponse>): MsgFreezeAccountResponse;
};
/**
 * MsgUnfreezeAccount unfreezes a frozen account.
 * @name MsgUnfreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccount
 */
export declare const MsgUnfreezeAccount: {
    typeUrl: string;
    encode(message: MsgUnfreezeAccount, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnfreezeAccount;
    fromPartial(object: DeepPartial<MsgUnfreezeAccount>): MsgUnfreezeAccount;
};
/**
 * @name MsgUnfreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccountResponse
 */
export declare const MsgUnfreezeAccountResponse: {
    typeUrl: string;
    encode(_: MsgUnfreezeAccountResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnfreezeAccountResponse;
    fromPartial(_: DeepPartial<MsgUnfreezeAccountResponse>): MsgUnfreezeAccountResponse;
};
/**
 * MsgUpdateParams updates auth module parameters.
 * @name MsgUpdateParams
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
