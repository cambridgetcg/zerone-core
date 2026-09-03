import { Params } from "./state.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgCreateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateSchedule
 */
export interface MsgCreateSchedule {
    creator: string;
    recipient: string;
    amountPerExecutionUzrn: string;
    firstExecutionHeight: bigint;
    intervalBlocks: bigint;
    executionCount: number;
}
/**
 * @name MsgCreateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateScheduleResponse
 */
export interface MsgCreateScheduleResponse {
    scheduleId: string;
    escrowedUzrn: string;
}
/**
 * UpdateSchedule replaces all not-yet-executed terms. The two expected values
 * provide compare-and-swap protection against an occurrence executing or a
 * prior amendment committing before this transaction.
 * @name MsgUpdateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateSchedule
 */
export interface MsgUpdateSchedule {
    creator: string;
    scheduleId: string;
    expectedRevision: bigint;
    expectedExecutionCount: number;
    recipient: string;
    amountPerExecutionUzrn: string;
    nextExecutionHeight: bigint;
    intervalBlocks: bigint;
    remainingExecutions: number;
}
/**
 * @name MsgUpdateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateScheduleResponse
 */
export interface MsgUpdateScheduleResponse {
    revision: bigint;
    escrowDeltaUzrn: string;
    refunded: boolean;
}
/**
 * @name MsgCancelSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelSchedule
 */
export interface MsgCancelSchedule {
    creator: string;
    scheduleId: string;
    expectedRevision: bigint;
    expectedExecutionCount: number;
}
/**
 * @name MsgCancelScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelScheduleResponse
 */
export interface MsgCancelScheduleResponse {
    refundedUzrn: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * @name MsgCreateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateSchedule
 */
export declare const MsgCreateSchedule: {
    typeUrl: string;
    encode(message: MsgCreateSchedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateSchedule;
    fromPartial(object: DeepPartial<MsgCreateSchedule>): MsgCreateSchedule;
};
/**
 * @name MsgCreateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCreateScheduleResponse
 */
export declare const MsgCreateScheduleResponse: {
    typeUrl: string;
    encode(message: MsgCreateScheduleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateScheduleResponse;
    fromPartial(object: DeepPartial<MsgCreateScheduleResponse>): MsgCreateScheduleResponse;
};
/**
 * UpdateSchedule replaces all not-yet-executed terms. The two expected values
 * provide compare-and-swap protection against an occurrence executing or a
 * prior amendment committing before this transaction.
 * @name MsgUpdateSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateSchedule
 */
export declare const MsgUpdateSchedule: {
    typeUrl: string;
    encode(message: MsgUpdateSchedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateSchedule;
    fromPartial(object: DeepPartial<MsgUpdateSchedule>): MsgUpdateSchedule;
};
/**
 * @name MsgUpdateScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateScheduleResponse
 */
export declare const MsgUpdateScheduleResponse: {
    typeUrl: string;
    encode(message: MsgUpdateScheduleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateScheduleResponse;
    fromPartial(object: DeepPartial<MsgUpdateScheduleResponse>): MsgUpdateScheduleResponse;
};
/**
 * @name MsgCancelSchedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelSchedule
 */
export declare const MsgCancelSchedule: {
    typeUrl: string;
    encode(message: MsgCancelSchedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelSchedule;
    fromPartial(object: DeepPartial<MsgCancelSchedule>): MsgCancelSchedule;
};
/**
 * @name MsgCancelScheduleResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgCancelScheduleResponse
 */
export declare const MsgCancelScheduleResponse: {
    typeUrl: string;
    encode(message: MsgCancelScheduleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelScheduleResponse;
    fromPartial(object: DeepPartial<MsgCancelScheduleResponse>): MsgCancelScheduleResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
