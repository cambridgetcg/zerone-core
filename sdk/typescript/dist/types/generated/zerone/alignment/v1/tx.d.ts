import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgUpdateParams
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgActivate
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgActivate
 */
export interface MsgActivate {
    authority: string;
    enabled: boolean;
}
/**
 * @name MsgActivateResponse
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgActivateResponse
 */
export interface MsgActivateResponse {
}
/**
 * @name MsgUpdateParams
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
/**
 * @name MsgActivate
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgActivate
 */
export declare const MsgActivate: {
    typeUrl: string;
    encode(message: MsgActivate, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgActivate;
    fromPartial(object: DeepPartial<MsgActivate>): MsgActivate;
};
/**
 * @name MsgActivateResponse
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.MsgActivateResponse
 */
export declare const MsgActivateResponse: {
    typeUrl: string;
    encode(_: MsgActivateResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgActivateResponse;
    fromPartial(_: DeepPartial<MsgActivateResponse>): MsgActivateResponse;
};
