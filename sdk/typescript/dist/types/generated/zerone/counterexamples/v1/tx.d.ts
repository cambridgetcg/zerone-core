import { ErrorType, CounterexampleStatus } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgProposeCounterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexample
 */
export interface MsgProposeCounterexample {
    author: string;
    factId: string;
    wrongClaim: string;
    reasoning: string;
    errorType: ErrorType;
    violatedMethodologyIds: string[];
}
/**
 * @name MsgProposeCounterexampleResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexampleResponse
 */
export interface MsgProposeCounterexampleResponse {
    counterexampleId: string;
}
/**
 * @name MsgValidate
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidate
 */
export interface MsgValidate {
    validator: string;
    counterexampleId: string;
    affirm: boolean;
    reason: string;
}
/**
 * @name MsgValidateResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidateResponse
 */
export interface MsgValidateResponse {
    validationId: bigint;
    /**
     * True if this vote caused the counterexample to resolve.
     */
    resolved: boolean;
    /**
     * Final status if resolved.
     */
    status: CounterexampleStatus;
}
/**
 * @name MsgUpdateParams
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgProposeCounterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexample
 */
export declare const MsgProposeCounterexample: {
    typeUrl: string;
    encode(message: MsgProposeCounterexample, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeCounterexample;
    fromPartial(object: DeepPartial<MsgProposeCounterexample>): MsgProposeCounterexample;
};
/**
 * @name MsgProposeCounterexampleResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgProposeCounterexampleResponse
 */
export declare const MsgProposeCounterexampleResponse: {
    typeUrl: string;
    encode(message: MsgProposeCounterexampleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeCounterexampleResponse;
    fromPartial(object: DeepPartial<MsgProposeCounterexampleResponse>): MsgProposeCounterexampleResponse;
};
/**
 * @name MsgValidate
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidate
 */
export declare const MsgValidate: {
    typeUrl: string;
    encode(message: MsgValidate, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgValidate;
    fromPartial(object: DeepPartial<MsgValidate>): MsgValidate;
};
/**
 * @name MsgValidateResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgValidateResponse
 */
export declare const MsgValidateResponse: {
    typeUrl: string;
    encode(message: MsgValidateResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgValidateResponse;
    fromPartial(object: DeepPartial<MsgValidateResponse>): MsgValidateResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
