import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgCreateBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrder
 */
export interface MsgCreateBountyOrder {
    sponsor: string;
    domain: string;
    pricePerArtifact: string;
    targetCount: number;
    durationBlocks: bigint;
}
/**
 * @name MsgCreateBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrderResponse
 */
export interface MsgCreateBountyOrderResponse {
    bountyId: string;
}
/**
 * @name MsgFulfillBounty
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBounty
 */
export interface MsgFulfillBounty {
    caller: string;
    bountyId: string;
    factId: string;
}
/**
 * @name MsgFulfillBountyResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBountyResponse
 */
export interface MsgFulfillBountyResponse {
    worker: string;
    amountPaid: string;
    bountyNowFulfilled: boolean;
}
/**
 * @name MsgCancelBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrder
 */
export interface MsgCancelBountyOrder {
    sponsor: string;
    bountyId: string;
}
/**
 * @name MsgCancelBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrderResponse
 */
export interface MsgCancelBountyOrderResponse {
    refundedAmount: string;
}
/**
 * @name MsgCreateBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrder
 */
export declare const MsgCreateBountyOrder: {
    typeUrl: string;
    encode(message: MsgCreateBountyOrder, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateBountyOrder;
    fromPartial(object: DeepPartial<MsgCreateBountyOrder>): MsgCreateBountyOrder;
};
/**
 * @name MsgCreateBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCreateBountyOrderResponse
 */
export declare const MsgCreateBountyOrderResponse: {
    typeUrl: string;
    encode(message: MsgCreateBountyOrderResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateBountyOrderResponse;
    fromPartial(object: DeepPartial<MsgCreateBountyOrderResponse>): MsgCreateBountyOrderResponse;
};
/**
 * @name MsgFulfillBounty
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBounty
 */
export declare const MsgFulfillBounty: {
    typeUrl: string;
    encode(message: MsgFulfillBounty, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFulfillBounty;
    fromPartial(object: DeepPartial<MsgFulfillBounty>): MsgFulfillBounty;
};
/**
 * @name MsgFulfillBountyResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgFulfillBountyResponse
 */
export declare const MsgFulfillBountyResponse: {
    typeUrl: string;
    encode(message: MsgFulfillBountyResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFulfillBountyResponse;
    fromPartial(object: DeepPartial<MsgFulfillBountyResponse>): MsgFulfillBountyResponse;
};
/**
 * @name MsgCancelBountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrder
 */
export declare const MsgCancelBountyOrder: {
    typeUrl: string;
    encode(message: MsgCancelBountyOrder, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelBountyOrder;
    fromPartial(object: DeepPartial<MsgCancelBountyOrder>): MsgCancelBountyOrder;
};
/**
 * @name MsgCancelBountyOrderResponse
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.MsgCancelBountyOrderResponse
 */
export declare const MsgCancelBountyOrderResponse: {
    typeUrl: string;
    encode(message: MsgCancelBountyOrderResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelBountyOrderResponse;
    fromPartial(object: DeepPartial<MsgCancelBountyOrderResponse>): MsgCancelBountyOrderResponse;
};
