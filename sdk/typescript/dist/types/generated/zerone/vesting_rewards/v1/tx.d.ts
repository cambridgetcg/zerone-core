import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** VestingCategory classifies the purpose of a vesting schedule. */
export declare enum VestingCategory {
    VESTING_CATEGORY_UNSPECIFIED = 0,
    VESTING_CATEGORY_VERIFICATION_REWARD = 1,
    VESTING_CATEGORY_BLOCK_REWARD = 2,
    VESTING_CATEGORY_BOUNTY_REWARD = 3,
    VESTING_CATEGORY_DISPUTE_REWARD = 4,
    VESTING_CATEGORY_RESEARCH_GRANT = 5,
    VESTING_CATEGORY_BOOTSTRAP = 6,
    UNRECOGNIZED = -1
}
export declare function vestingCategoryFromJSON(object: any): VestingCategory;
export declare function vestingCategoryToJSON(object: VestingCategory): string;
/**
 * @name MsgCreateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVesting
 */
export interface MsgCreateVesting {
    authority: string;
    beneficiary: string;
    amount: string;
    category: VestingCategory;
    linkedFactId: string;
    startHeight: bigint;
}
/**
 * @name MsgCreateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVestingResponse
 */
export interface MsgCreateVestingResponse {
    vestingId: string;
}
/**
 * @name MsgClaimVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVesting
 */
export interface MsgClaimVesting {
    claimer: string;
    vestingIds: string[];
}
/**
 * @name MsgClaimVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVestingResponse
 */
export interface MsgClaimVestingResponse {
    totalClaimed: string;
    vestingsClaimed: number;
}
/**
 * @name MsgPauseVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVesting
 */
export interface MsgPauseVesting {
    authority: string;
    vestingId: string;
    reason: string;
}
/**
 * @name MsgPauseVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVestingResponse
 */
export interface MsgPauseVestingResponse {
}
/**
 * @name MsgResumeVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVesting
 */
export interface MsgResumeVesting {
    authority: string;
    vestingId: string;
}
/**
 * @name MsgResumeVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVestingResponse
 */
export interface MsgResumeVestingResponse {
}
/**
 * @name MsgAccelerateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVesting
 */
export interface MsgAccelerateVesting {
    authority: string;
    vestingId: string;
    accelerationFactor: number;
}
/**
 * @name MsgAccelerateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVestingResponse
 */
export interface MsgAccelerateVestingResponse {
}
/**
 * @name MsgFalsifyVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVesting
 */
export interface MsgFalsifyVesting {
    challenger: string;
    vestingId: string;
    reason: string;
    counterEvidenceHash: string;
}
/**
 * @name MsgFalsifyVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVestingResponse
 */
export interface MsgFalsifyVestingResponse {
    vestingPaused: boolean;
}
/**
 * @name MsgCompleteVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVesting
 */
export interface MsgCompleteVesting {
    authority: string;
    vestingId: string;
}
/**
 * @name MsgCompleteVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVestingResponse
 */
export interface MsgCompleteVestingResponse {
    remainingAmount: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgCreateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVesting
 */
export declare const MsgCreateVesting: {
    typeUrl: string;
    encode(message: MsgCreateVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateVesting;
    fromPartial(object: DeepPartial<MsgCreateVesting>): MsgCreateVesting;
};
/**
 * @name MsgCreateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVestingResponse
 */
export declare const MsgCreateVestingResponse: {
    typeUrl: string;
    encode(message: MsgCreateVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateVestingResponse;
    fromPartial(object: DeepPartial<MsgCreateVestingResponse>): MsgCreateVestingResponse;
};
/**
 * @name MsgClaimVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVesting
 */
export declare const MsgClaimVesting: {
    typeUrl: string;
    encode(message: MsgClaimVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimVesting;
    fromPartial(object: DeepPartial<MsgClaimVesting>): MsgClaimVesting;
};
/**
 * @name MsgClaimVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVestingResponse
 */
export declare const MsgClaimVestingResponse: {
    typeUrl: string;
    encode(message: MsgClaimVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimVestingResponse;
    fromPartial(object: DeepPartial<MsgClaimVestingResponse>): MsgClaimVestingResponse;
};
/**
 * @name MsgPauseVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVesting
 */
export declare const MsgPauseVesting: {
    typeUrl: string;
    encode(message: MsgPauseVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseVesting;
    fromPartial(object: DeepPartial<MsgPauseVesting>): MsgPauseVesting;
};
/**
 * @name MsgPauseVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVestingResponse
 */
export declare const MsgPauseVestingResponse: {
    typeUrl: string;
    encode(_: MsgPauseVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseVestingResponse;
    fromPartial(_: DeepPartial<MsgPauseVestingResponse>): MsgPauseVestingResponse;
};
/**
 * @name MsgResumeVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVesting
 */
export declare const MsgResumeVesting: {
    typeUrl: string;
    encode(message: MsgResumeVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResumeVesting;
    fromPartial(object: DeepPartial<MsgResumeVesting>): MsgResumeVesting;
};
/**
 * @name MsgResumeVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVestingResponse
 */
export declare const MsgResumeVestingResponse: {
    typeUrl: string;
    encode(_: MsgResumeVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResumeVestingResponse;
    fromPartial(_: DeepPartial<MsgResumeVestingResponse>): MsgResumeVestingResponse;
};
/**
 * @name MsgAccelerateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVesting
 */
export declare const MsgAccelerateVesting: {
    typeUrl: string;
    encode(message: MsgAccelerateVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAccelerateVesting;
    fromPartial(object: DeepPartial<MsgAccelerateVesting>): MsgAccelerateVesting;
};
/**
 * @name MsgAccelerateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVestingResponse
 */
export declare const MsgAccelerateVestingResponse: {
    typeUrl: string;
    encode(_: MsgAccelerateVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAccelerateVestingResponse;
    fromPartial(_: DeepPartial<MsgAccelerateVestingResponse>): MsgAccelerateVestingResponse;
};
/**
 * @name MsgFalsifyVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVesting
 */
export declare const MsgFalsifyVesting: {
    typeUrl: string;
    encode(message: MsgFalsifyVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFalsifyVesting;
    fromPartial(object: DeepPartial<MsgFalsifyVesting>): MsgFalsifyVesting;
};
/**
 * @name MsgFalsifyVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVestingResponse
 */
export declare const MsgFalsifyVestingResponse: {
    typeUrl: string;
    encode(message: MsgFalsifyVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFalsifyVestingResponse;
    fromPartial(object: DeepPartial<MsgFalsifyVestingResponse>): MsgFalsifyVestingResponse;
};
/**
 * @name MsgCompleteVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVesting
 */
export declare const MsgCompleteVesting: {
    typeUrl: string;
    encode(message: MsgCompleteVesting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCompleteVesting;
    fromPartial(object: DeepPartial<MsgCompleteVesting>): MsgCompleteVesting;
};
/**
 * @name MsgCompleteVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVestingResponse
 */
export declare const MsgCompleteVestingResponse: {
    typeUrl: string;
    encode(message: MsgCompleteVestingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCompleteVestingResponse;
    fromPartial(object: DeepPartial<MsgCompleteVestingResponse>): MsgCompleteVestingResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
