import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * MsgRegisterValidator registers a new validator.
 * @name MsgRegisterValidator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidator
 */
export interface MsgRegisterValidator {
    operator: string;
    consensusPubkey: string;
    did: string;
    moniker: string;
    /**
     * uzrn
     */
    selfDelegation: string;
    commissionBps: bigint;
    website: string;
    details: string;
}
/**
 * @name MsgRegisterValidatorResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidatorResponse
 */
export interface MsgRegisterValidatorResponse {
    initialTier: number;
}
/**
 * MsgDelegate delegates tokens to a validator.
 * @name MsgDelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegate
 */
export interface MsgDelegate {
    delegator: string;
    validator: string;
    /**
     * uzrn
     */
    amount: string;
}
/**
 * @name MsgDelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegateResponse
 */
export interface MsgDelegateResponse {
    /**
     * total delegation after operation
     */
    newDelegation: string;
}
/**
 * MsgUndelegate initiates unbonding from a validator.
 * @name MsgUndelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegate
 */
export interface MsgUndelegate {
    delegator: string;
    validator: string;
    /**
     * uzrn
     */
    amount: string;
}
/**
 * @name MsgUndelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegateResponse
 */
export interface MsgUndelegateResponse {
    unbondingId: string;
    completesAtHeight: bigint;
}
/**
 * MsgRedelegate moves delegation between validators.
 * @name MsgRedelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegate
 */
export interface MsgRedelegate {
    delegator: string;
    srcValidator: string;
    dstValidator: string;
    /**
     * uzrn
     */
    amount: string;
}
/**
 * @name MsgRedelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegateResponse
 */
export interface MsgRedelegateResponse {
}
/**
 * MsgUpdateValidatorStake increases or decreases self-delegation.
 * @name MsgUpdateValidatorStake
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStake
 */
export interface MsgUpdateValidatorStake {
    operator: string;
    /**
     * uzrn
     */
    amount: string;
    increase: boolean;
}
/**
 * @name MsgUpdateValidatorStakeResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStakeResponse
 */
export interface MsgUpdateValidatorStakeResponse {
}
/**
 * MsgUpdateParams updates module parameters (governance-gated).
 * @name MsgUpdateParams
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * MsgRegisterValidator registers a new validator.
 * @name MsgRegisterValidator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidator
 */
export declare const MsgRegisterValidator: {
    typeUrl: string;
    encode(message: MsgRegisterValidator, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterValidator;
    fromPartial(object: DeepPartial<MsgRegisterValidator>): MsgRegisterValidator;
};
/**
 * @name MsgRegisterValidatorResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidatorResponse
 */
export declare const MsgRegisterValidatorResponse: {
    typeUrl: string;
    encode(message: MsgRegisterValidatorResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterValidatorResponse;
    fromPartial(object: DeepPartial<MsgRegisterValidatorResponse>): MsgRegisterValidatorResponse;
};
/**
 * MsgDelegate delegates tokens to a validator.
 * @name MsgDelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegate
 */
export declare const MsgDelegate: {
    typeUrl: string;
    encode(message: MsgDelegate, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegate;
    fromPartial(object: DeepPartial<MsgDelegate>): MsgDelegate;
};
/**
 * @name MsgDelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegateResponse
 */
export declare const MsgDelegateResponse: {
    typeUrl: string;
    encode(message: MsgDelegateResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegateResponse;
    fromPartial(object: DeepPartial<MsgDelegateResponse>): MsgDelegateResponse;
};
/**
 * MsgUndelegate initiates unbonding from a validator.
 * @name MsgUndelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegate
 */
export declare const MsgUndelegate: {
    typeUrl: string;
    encode(message: MsgUndelegate, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegate;
    fromPartial(object: DeepPartial<MsgUndelegate>): MsgUndelegate;
};
/**
 * @name MsgUndelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegateResponse
 */
export declare const MsgUndelegateResponse: {
    typeUrl: string;
    encode(message: MsgUndelegateResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegateResponse;
    fromPartial(object: DeepPartial<MsgUndelegateResponse>): MsgUndelegateResponse;
};
/**
 * MsgRedelegate moves delegation between validators.
 * @name MsgRedelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegate
 */
export declare const MsgRedelegate: {
    typeUrl: string;
    encode(message: MsgRedelegate, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRedelegate;
    fromPartial(object: DeepPartial<MsgRedelegate>): MsgRedelegate;
};
/**
 * @name MsgRedelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegateResponse
 */
export declare const MsgRedelegateResponse: {
    typeUrl: string;
    encode(_: MsgRedelegateResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRedelegateResponse;
    fromPartial(_: DeepPartial<MsgRedelegateResponse>): MsgRedelegateResponse;
};
/**
 * MsgUpdateValidatorStake increases or decreases self-delegation.
 * @name MsgUpdateValidatorStake
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStake
 */
export declare const MsgUpdateValidatorStake: {
    typeUrl: string;
    encode(message: MsgUpdateValidatorStake, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateValidatorStake;
    fromPartial(object: DeepPartial<MsgUpdateValidatorStake>): MsgUpdateValidatorStake;
};
/**
 * @name MsgUpdateValidatorStakeResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStakeResponse
 */
export declare const MsgUpdateValidatorStakeResponse: {
    typeUrl: string;
    encode(_: MsgUpdateValidatorStakeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateValidatorStakeResponse;
    fromPartial(_: DeepPartial<MsgUpdateValidatorStakeResponse>): MsgUpdateValidatorStakeResponse;
};
/**
 * MsgUpdateParams updates module parameters (governance-gated).
 * @name MsgUpdateParams
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
