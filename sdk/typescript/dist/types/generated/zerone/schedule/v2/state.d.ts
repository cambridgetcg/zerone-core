import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * ScheduleStatus is the durable lifecycle of a committed schedule. A schedule
 * message only reaches ACTIVE after the surrounding transaction is committed;
 * the node-local mempool is never authoritative schedule state.
 */
export declare enum ScheduleStatus {
    SCHEDULE_STATUS_UNSPECIFIED = 0,
    SCHEDULE_STATUS_ACTIVE = 1,
    SCHEDULE_STATUS_COMPLETED = 2,
    SCHEDULE_STATUS_CANCELLED = 3,
    SCHEDULE_STATUS_FAILED = 4,
    UNRECOGNIZED = -1
}
export declare function scheduleStatusFromJSON(object: any): ScheduleStatus;
export declare function scheduleStatusToJSON(object: ScheduleStatus): string;
/**
 * ExecutionOutcome records the terminal result of one deterministic
 * occurrence. An occurrence is never retried after a receipt is committed.
 */
export declare enum ExecutionOutcome {
    EXECUTION_OUTCOME_UNSPECIFIED = 0,
    EXECUTION_OUTCOME_SUCCEEDED = 1,
    EXECUTION_OUTCOME_FAILED_AND_REFUNDED = 2,
    UNRECOGNIZED = -1
}
export declare function executionOutcomeFromJSON(object: any): ExecutionOutcome;
export declare function executionOutcomeToJSON(object: ExecutionOutcome): string;
/**
 * Params bounds per-creator active admission and per-block due processing below
 * immutable source ceilings. Terminal records and receipts remain append-only.
 * New schedule admission is closed by default so source deployment does not
 * silently activate a new economic surface.
 * @name Params
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.Params
 */
export interface Params {
    acceptNewSchedules: boolean;
    minScheduleDelayBlocks: bigint;
    minIntervalBlocks: bigint;
    maxExecutionsPerSchedule: number;
    maxActiveSchedulesPerCreator: number;
    maxDueRecordsPerBlock: number;
    maxQueryLimit: number;
    executionFeeUzrn: string;
    maxTransferPerExecutionUzrn: string;
}
/**
 * Schedule is a prefunded native-token transfer commitment. This initial
 * bounded design is intentionally finite and height based: it cannot execute
 * arbitrary SDK or contract messages, read validator wall clocks, or depend on
 * local mempools.
 * @name Schedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.Schedule
 */
export interface Schedule {
    id: string;
    creator: string;
    revision: bigint;
    status: ScheduleStatus;
    recipient: string;
    amountPerExecutionUzrn: string;
    executionFeeUzrn: string;
    nextExecutionHeight: bigint;
    intervalBlocks: bigint;
    executionCount: number;
    remainingExecutions: number;
    principalRemainingUzrn: string;
    feeRemainingUzrn: string;
    createdHeight: bigint;
    updatedHeight: bigint;
    lastExecutionHeight: bigint;
    terminalReason: string;
}
/**
 * ExecutionReceipt is immutable evidence that one occurrence was processed.
 * occurrence_id and action_sha256 are lowercase hex SHA-256 digests over
 * canonical, domain-separated encodings defined by x/schedule.
 * @name ExecutionReceipt
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.ExecutionReceipt
 */
export interface ExecutionReceipt {
    occurrenceId: string;
    scheduleId: string;
    revision: bigint;
    sequence: number;
    dueHeight: bigint;
    executedHeight: bigint;
    recipient: string;
    amountUzrn: string;
    feeUzrn: string;
    actionSha256: string;
    outcome: ExecutionOutcome;
    failureCode: string;
}
/**
 * Params bounds per-creator active admission and per-block due processing below
 * immutable source ceilings. Terminal records and receipts remain append-only.
 * New schedule admission is closed by default so source deployment does not
 * silently activate a new economic surface.
 * @name Params
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * Schedule is a prefunded native-token transfer commitment. This initial
 * bounded design is intentionally finite and height based: it cannot execute
 * arbitrary SDK or contract messages, read validator wall clocks, or depend on
 * local mempools.
 * @name Schedule
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.Schedule
 */
export declare const Schedule: {
    typeUrl: string;
    encode(message: Schedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Schedule;
    fromPartial(object: DeepPartial<Schedule>): Schedule;
};
/**
 * ExecutionReceipt is immutable evidence that one occurrence was processed.
 * occurrence_id and action_sha256 are lowercase hex SHA-256 digests over
 * canonical, domain-separated encodings defined by x/schedule.
 * @name ExecutionReceipt
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.ExecutionReceipt
 */
export declare const ExecutionReceipt: {
    typeUrl: string;
    encode(message: ExecutionReceipt, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ExecutionReceipt;
    fromPartial(object: DeepPartial<ExecutionReceipt>): ExecutionReceipt;
};
