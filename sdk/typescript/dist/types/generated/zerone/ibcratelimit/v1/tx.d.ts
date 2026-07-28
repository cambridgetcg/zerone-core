import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgAddRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimit
 */
export interface MsgAddRateLimit {
    /**
     * governance address
     */
    authority: string;
    channelId: string;
    denom: string;
    /**
     * bigint string
     */
    maxSend: string;
    /**
     * bigint string
     */
    maxRecv: string;
    windowBlocks: bigint;
}
/**
 * @name MsgAddRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimitResponse
 */
export interface MsgAddRateLimitResponse {
}
/**
 * @name MsgRemoveRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimit
 */
export interface MsgRemoveRateLimit {
    /**
     * governance address
     */
    authority: string;
    channelId: string;
    denom: string;
}
/**
 * @name MsgRemoveRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse
 */
export interface MsgRemoveRateLimitResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    /**
     * governance address
     */
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgAddRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimit
 */
export declare const MsgAddRateLimit: {
    typeUrl: string;
    encode(message: MsgAddRateLimit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddRateLimit;
    fromPartial(object: DeepPartial<MsgAddRateLimit>): MsgAddRateLimit;
};
/**
 * @name MsgAddRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgAddRateLimitResponse
 */
export declare const MsgAddRateLimitResponse: {
    typeUrl: string;
    encode(_: MsgAddRateLimitResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddRateLimitResponse;
    fromPartial(_: DeepPartial<MsgAddRateLimitResponse>): MsgAddRateLimitResponse;
};
/**
 * @name MsgRemoveRateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimit
 */
export declare const MsgRemoveRateLimit: {
    typeUrl: string;
    encode(message: MsgRemoveRateLimit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveRateLimit;
    fromPartial(object: DeepPartial<MsgRemoveRateLimit>): MsgRemoveRateLimit;
};
/**
 * @name MsgRemoveRateLimitResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse
 */
export declare const MsgRemoveRateLimitResponse: {
    typeUrl: string;
    encode(_: MsgRemoveRateLimitResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveRateLimitResponse;
    fromPartial(_: DeepPartial<MsgRemoveRateLimitResponse>): MsgRemoveRateLimitResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
