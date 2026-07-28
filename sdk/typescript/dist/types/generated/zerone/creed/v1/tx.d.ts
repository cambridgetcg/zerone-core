import { PinnedCreed, CreedCouncilMember } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgAnchorPin
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPin
 */
export interface MsgAnchorPin {
    authority: string;
    /**
     * The new pinned creed. version MUST equal currentPin.version+1.
     * canonical_hash MUST be non-empty. commitments MUST satisfy:
     *   - all numbers unique
     *   - numbers are 1..N for some N (no gaps unless the missing
     *     number's prior entry is archived in this same version)
     *   - introduced_at_height ≤ block_height for all entries
     */
    pin?: PinnedCreed;
    /**
     * The LIP id this pin came from. Required when params.
     * direct_anchor_enabled is false. Empty allowed only for
     * genesis-equivalent pre-launch pins.
     */
    sourceLip: string;
}
/**
 * @name MsgAnchorPinResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPinResponse
 */
export interface MsgAnchorPinResponse {
    newVersion: number;
}
/**
 * @name MsgUpdateParams
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgUpdateCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMember
 */
export interface MsgUpdateCouncilMember {
    authority: string;
    /**
     * The seat to upsert. address is the key; existing seats with
     * the same address are updated in place (their admitted_at_height
     * is preserved if unchanged).
     */
    member?: CreedCouncilMember;
    /**
     * LIP id that authorized this change. Required when params.
     * direct_anchor_enabled is false. Empty allowed only for
     * pre-launch council seeding outside of gov.
     */
    sourceLip: string;
}
/**
 * @name MsgUpdateCouncilMemberResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMemberResponse
 */
export interface MsgUpdateCouncilMemberResponse {
}
/**
 * @name MsgAnchorPin
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPin
 */
export declare const MsgAnchorPin: {
    typeUrl: string;
    encode(message: MsgAnchorPin, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAnchorPin;
    fromPartial(object: DeepPartial<MsgAnchorPin>): MsgAnchorPin;
};
/**
 * @name MsgAnchorPinResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgAnchorPinResponse
 */
export declare const MsgAnchorPinResponse: {
    typeUrl: string;
    encode(message: MsgAnchorPinResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAnchorPinResponse;
    fromPartial(object: DeepPartial<MsgAnchorPinResponse>): MsgAnchorPinResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
/**
 * @name MsgUpdateCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMember
 */
export declare const MsgUpdateCouncilMember: {
    typeUrl: string;
    encode(message: MsgUpdateCouncilMember, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateCouncilMember;
    fromPartial(object: DeepPartial<MsgUpdateCouncilMember>): MsgUpdateCouncilMember;
};
/**
 * @name MsgUpdateCouncilMemberResponse
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.MsgUpdateCouncilMemberResponse
 */
export declare const MsgUpdateCouncilMemberResponse: {
    typeUrl: string;
    encode(_: MsgUpdateCouncilMemberResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateCouncilMemberResponse;
    fromPartial(_: DeepPartial<MsgUpdateCouncilMemberResponse>): MsgUpdateCouncilMemberResponse;
};
