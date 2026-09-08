//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * ScheduleStatus is the durable lifecycle of a committed schedule. A schedule
 * message only reaches ACTIVE after the surrounding transaction is committed;
 * the node-local mempool is never authoritative schedule state.
 */
export enum ScheduleStatus {
  SCHEDULE_STATUS_UNSPECIFIED = 0,
  SCHEDULE_STATUS_ACTIVE = 1,
  SCHEDULE_STATUS_COMPLETED = 2,
  SCHEDULE_STATUS_CANCELLED = 3,
  SCHEDULE_STATUS_FAILED = 4,
  UNRECOGNIZED = -1,
}
export function scheduleStatusFromJSON(object: any): ScheduleStatus {
  switch (object) {
    case 0:
    case "SCHEDULE_STATUS_UNSPECIFIED":
      return ScheduleStatus.SCHEDULE_STATUS_UNSPECIFIED;
    case 1:
    case "SCHEDULE_STATUS_ACTIVE":
      return ScheduleStatus.SCHEDULE_STATUS_ACTIVE;
    case 2:
    case "SCHEDULE_STATUS_COMPLETED":
      return ScheduleStatus.SCHEDULE_STATUS_COMPLETED;
    case 3:
    case "SCHEDULE_STATUS_CANCELLED":
      return ScheduleStatus.SCHEDULE_STATUS_CANCELLED;
    case 4:
    case "SCHEDULE_STATUS_FAILED":
      return ScheduleStatus.SCHEDULE_STATUS_FAILED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ScheduleStatus.UNRECOGNIZED;
  }
}
export function scheduleStatusToJSON(object: ScheduleStatus): string {
  switch (object) {
    case ScheduleStatus.SCHEDULE_STATUS_UNSPECIFIED:
      return "SCHEDULE_STATUS_UNSPECIFIED";
    case ScheduleStatus.SCHEDULE_STATUS_ACTIVE:
      return "SCHEDULE_STATUS_ACTIVE";
    case ScheduleStatus.SCHEDULE_STATUS_COMPLETED:
      return "SCHEDULE_STATUS_COMPLETED";
    case ScheduleStatus.SCHEDULE_STATUS_CANCELLED:
      return "SCHEDULE_STATUS_CANCELLED";
    case ScheduleStatus.SCHEDULE_STATUS_FAILED:
      return "SCHEDULE_STATUS_FAILED";
    case ScheduleStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * ExecutionOutcome records the terminal result of one deterministic
 * occurrence. An occurrence is never retried after a receipt is committed.
 */
export enum ExecutionOutcome {
  EXECUTION_OUTCOME_UNSPECIFIED = 0,
  EXECUTION_OUTCOME_SUCCEEDED = 1,
  EXECUTION_OUTCOME_FAILED_AND_REFUNDED = 2,
  UNRECOGNIZED = -1,
}
export function executionOutcomeFromJSON(object: any): ExecutionOutcome {
  switch (object) {
    case 0:
    case "EXECUTION_OUTCOME_UNSPECIFIED":
      return ExecutionOutcome.EXECUTION_OUTCOME_UNSPECIFIED;
    case 1:
    case "EXECUTION_OUTCOME_SUCCEEDED":
      return ExecutionOutcome.EXECUTION_OUTCOME_SUCCEEDED;
    case 2:
    case "EXECUTION_OUTCOME_FAILED_AND_REFUNDED":
      return ExecutionOutcome.EXECUTION_OUTCOME_FAILED_AND_REFUNDED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ExecutionOutcome.UNRECOGNIZED;
  }
}
export function executionOutcomeToJSON(object: ExecutionOutcome): string {
  switch (object) {
    case ExecutionOutcome.EXECUTION_OUTCOME_UNSPECIFIED:
      return "EXECUTION_OUTCOME_UNSPECIFIED";
    case ExecutionOutcome.EXECUTION_OUTCOME_SUCCEEDED:
      return "EXECUTION_OUTCOME_SUCCEEDED";
    case ExecutionOutcome.EXECUTION_OUTCOME_FAILED_AND_REFUNDED:
      return "EXECUTION_OUTCOME_FAILED_AND_REFUNDED";
    case ExecutionOutcome.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
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
function createBaseParams(): Params {
  return {
    acceptNewSchedules: false,
    minScheduleDelayBlocks: BigInt(0),
    minIntervalBlocks: BigInt(0),
    maxExecutionsPerSchedule: 0,
    maxActiveSchedulesPerCreator: 0,
    maxDueRecordsPerBlock: 0,
    maxQueryLimit: 0,
    executionFeeUzrn: "",
    maxTransferPerExecutionUzrn: ""
  };
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
export const Params = {
  typeUrl: "/zerone.schedule.v2.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.acceptNewSchedules === true) {
      writer.uint32(8).bool(message.acceptNewSchedules);
    }
    if (message.minScheduleDelayBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.minScheduleDelayBlocks);
    }
    if (message.minIntervalBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.minIntervalBlocks);
    }
    if (message.maxExecutionsPerSchedule !== 0) {
      writer.uint32(32).uint32(message.maxExecutionsPerSchedule);
    }
    if (message.maxActiveSchedulesPerCreator !== 0) {
      writer.uint32(40).uint32(message.maxActiveSchedulesPerCreator);
    }
    if (message.maxDueRecordsPerBlock !== 0) {
      writer.uint32(48).uint32(message.maxDueRecordsPerBlock);
    }
    if (message.maxQueryLimit !== 0) {
      writer.uint32(56).uint32(message.maxQueryLimit);
    }
    if (message.executionFeeUzrn !== "") {
      writer.uint32(66).string(message.executionFeeUzrn);
    }
    if (message.maxTransferPerExecutionUzrn !== "") {
      writer.uint32(74).string(message.maxTransferPerExecutionUzrn);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.acceptNewSchedules = reader.bool();
          break;
        case 2:
          message.minScheduleDelayBlocks = reader.uint64();
          break;
        case 3:
          message.minIntervalBlocks = reader.uint64();
          break;
        case 4:
          message.maxExecutionsPerSchedule = reader.uint32();
          break;
        case 5:
          message.maxActiveSchedulesPerCreator = reader.uint32();
          break;
        case 6:
          message.maxDueRecordsPerBlock = reader.uint32();
          break;
        case 7:
          message.maxQueryLimit = reader.uint32();
          break;
        case 8:
          message.executionFeeUzrn = reader.string();
          break;
        case 9:
          message.maxTransferPerExecutionUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.acceptNewSchedules = object.acceptNewSchedules ?? false;
    message.minScheduleDelayBlocks = object.minScheduleDelayBlocks !== undefined && object.minScheduleDelayBlocks !== null ? BigInt(object.minScheduleDelayBlocks.toString()) : BigInt(0);
    message.minIntervalBlocks = object.minIntervalBlocks !== undefined && object.minIntervalBlocks !== null ? BigInt(object.minIntervalBlocks.toString()) : BigInt(0);
    message.maxExecutionsPerSchedule = object.maxExecutionsPerSchedule ?? 0;
    message.maxActiveSchedulesPerCreator = object.maxActiveSchedulesPerCreator ?? 0;
    message.maxDueRecordsPerBlock = object.maxDueRecordsPerBlock ?? 0;
    message.maxQueryLimit = object.maxQueryLimit ?? 0;
    message.executionFeeUzrn = object.executionFeeUzrn ?? "";
    message.maxTransferPerExecutionUzrn = object.maxTransferPerExecutionUzrn ?? "";
    return message;
  }
};
function createBaseSchedule(): Schedule {
  return {
    id: "",
    creator: "",
    revision: BigInt(0),
    status: 0,
    recipient: "",
    amountPerExecutionUzrn: "",
    executionFeeUzrn: "",
    nextExecutionHeight: BigInt(0),
    intervalBlocks: BigInt(0),
    executionCount: 0,
    remainingExecutions: 0,
    principalRemainingUzrn: "",
    feeRemainingUzrn: "",
    createdHeight: BigInt(0),
    updatedHeight: BigInt(0),
    lastExecutionHeight: BigInt(0),
    terminalReason: ""
  };
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
export const Schedule = {
  typeUrl: "/zerone.schedule.v2.Schedule",
  encode(message: Schedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.creator !== "") {
      writer.uint32(18).string(message.creator);
    }
    if (message.revision !== BigInt(0)) {
      writer.uint32(24).uint64(message.revision);
    }
    if (message.status !== 0) {
      writer.uint32(32).int32(message.status);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    if (message.amountPerExecutionUzrn !== "") {
      writer.uint32(50).string(message.amountPerExecutionUzrn);
    }
    if (message.executionFeeUzrn !== "") {
      writer.uint32(58).string(message.executionFeeUzrn);
    }
    if (message.nextExecutionHeight !== BigInt(0)) {
      writer.uint32(64).uint64(message.nextExecutionHeight);
    }
    if (message.intervalBlocks !== BigInt(0)) {
      writer.uint32(72).uint64(message.intervalBlocks);
    }
    if (message.executionCount !== 0) {
      writer.uint32(80).uint32(message.executionCount);
    }
    if (message.remainingExecutions !== 0) {
      writer.uint32(88).uint32(message.remainingExecutions);
    }
    if (message.principalRemainingUzrn !== "") {
      writer.uint32(98).string(message.principalRemainingUzrn);
    }
    if (message.feeRemainingUzrn !== "") {
      writer.uint32(106).string(message.feeRemainingUzrn);
    }
    if (message.createdHeight !== BigInt(0)) {
      writer.uint32(112).uint64(message.createdHeight);
    }
    if (message.updatedHeight !== BigInt(0)) {
      writer.uint32(120).uint64(message.updatedHeight);
    }
    if (message.lastExecutionHeight !== BigInt(0)) {
      writer.uint32(128).uint64(message.lastExecutionHeight);
    }
    if (message.terminalReason !== "") {
      writer.uint32(138).string(message.terminalReason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Schedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.creator = reader.string();
          break;
        case 3:
          message.revision = reader.uint64();
          break;
        case 4:
          message.status = reader.int32() as any;
          break;
        case 5:
          message.recipient = reader.string();
          break;
        case 6:
          message.amountPerExecutionUzrn = reader.string();
          break;
        case 7:
          message.executionFeeUzrn = reader.string();
          break;
        case 8:
          message.nextExecutionHeight = reader.uint64();
          break;
        case 9:
          message.intervalBlocks = reader.uint64();
          break;
        case 10:
          message.executionCount = reader.uint32();
          break;
        case 11:
          message.remainingExecutions = reader.uint32();
          break;
        case 12:
          message.principalRemainingUzrn = reader.string();
          break;
        case 13:
          message.feeRemainingUzrn = reader.string();
          break;
        case 14:
          message.createdHeight = reader.uint64();
          break;
        case 15:
          message.updatedHeight = reader.uint64();
          break;
        case 16:
          message.lastExecutionHeight = reader.uint64();
          break;
        case 17:
          message.terminalReason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Schedule>): Schedule {
    const message = createBaseSchedule();
    message.id = object.id ?? "";
    message.creator = object.creator ?? "";
    message.revision = object.revision !== undefined && object.revision !== null ? BigInt(object.revision.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    message.recipient = object.recipient ?? "";
    message.amountPerExecutionUzrn = object.amountPerExecutionUzrn ?? "";
    message.executionFeeUzrn = object.executionFeeUzrn ?? "";
    message.nextExecutionHeight = object.nextExecutionHeight !== undefined && object.nextExecutionHeight !== null ? BigInt(object.nextExecutionHeight.toString()) : BigInt(0);
    message.intervalBlocks = object.intervalBlocks !== undefined && object.intervalBlocks !== null ? BigInt(object.intervalBlocks.toString()) : BigInt(0);
    message.executionCount = object.executionCount ?? 0;
    message.remainingExecutions = object.remainingExecutions ?? 0;
    message.principalRemainingUzrn = object.principalRemainingUzrn ?? "";
    message.feeRemainingUzrn = object.feeRemainingUzrn ?? "";
    message.createdHeight = object.createdHeight !== undefined && object.createdHeight !== null ? BigInt(object.createdHeight.toString()) : BigInt(0);
    message.updatedHeight = object.updatedHeight !== undefined && object.updatedHeight !== null ? BigInt(object.updatedHeight.toString()) : BigInt(0);
    message.lastExecutionHeight = object.lastExecutionHeight !== undefined && object.lastExecutionHeight !== null ? BigInt(object.lastExecutionHeight.toString()) : BigInt(0);
    message.terminalReason = object.terminalReason ?? "";
    return message;
  }
};
function createBaseExecutionReceipt(): ExecutionReceipt {
  return {
    occurrenceId: "",
    scheduleId: "",
    revision: BigInt(0),
    sequence: 0,
    dueHeight: BigInt(0),
    executedHeight: BigInt(0),
    recipient: "",
    amountUzrn: "",
    feeUzrn: "",
    actionSha256: "",
    outcome: 0,
    failureCode: ""
  };
}
/**
 * ExecutionReceipt is immutable evidence that one occurrence was processed.
 * occurrence_id and action_sha256 are lowercase hex SHA-256 digests over
 * canonical, domain-separated encodings defined by x/schedule.
 * @name ExecutionReceipt
 * @package zerone.schedule.v2
 * @see proto type: zerone.schedule.v2.ExecutionReceipt
 */
export const ExecutionReceipt = {
  typeUrl: "/zerone.schedule.v2.ExecutionReceipt",
  encode(message: ExecutionReceipt, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.occurrenceId !== "") {
      writer.uint32(10).string(message.occurrenceId);
    }
    if (message.scheduleId !== "") {
      writer.uint32(18).string(message.scheduleId);
    }
    if (message.revision !== BigInt(0)) {
      writer.uint32(24).uint64(message.revision);
    }
    if (message.sequence !== 0) {
      writer.uint32(32).uint32(message.sequence);
    }
    if (message.dueHeight !== BigInt(0)) {
      writer.uint32(40).uint64(message.dueHeight);
    }
    if (message.executedHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.executedHeight);
    }
    if (message.recipient !== "") {
      writer.uint32(58).string(message.recipient);
    }
    if (message.amountUzrn !== "") {
      writer.uint32(66).string(message.amountUzrn);
    }
    if (message.feeUzrn !== "") {
      writer.uint32(74).string(message.feeUzrn);
    }
    if (message.actionSha256 !== "") {
      writer.uint32(82).string(message.actionSha256);
    }
    if (message.outcome !== 0) {
      writer.uint32(88).int32(message.outcome);
    }
    if (message.failureCode !== "") {
      writer.uint32(98).string(message.failureCode);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ExecutionReceipt {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExecutionReceipt();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.occurrenceId = reader.string();
          break;
        case 2:
          message.scheduleId = reader.string();
          break;
        case 3:
          message.revision = reader.uint64();
          break;
        case 4:
          message.sequence = reader.uint32();
          break;
        case 5:
          message.dueHeight = reader.uint64();
          break;
        case 6:
          message.executedHeight = reader.uint64();
          break;
        case 7:
          message.recipient = reader.string();
          break;
        case 8:
          message.amountUzrn = reader.string();
          break;
        case 9:
          message.feeUzrn = reader.string();
          break;
        case 10:
          message.actionSha256 = reader.string();
          break;
        case 11:
          message.outcome = reader.int32() as any;
          break;
        case 12:
          message.failureCode = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ExecutionReceipt>): ExecutionReceipt {
    const message = createBaseExecutionReceipt();
    message.occurrenceId = object.occurrenceId ?? "";
    message.scheduleId = object.scheduleId ?? "";
    message.revision = object.revision !== undefined && object.revision !== null ? BigInt(object.revision.toString()) : BigInt(0);
    message.sequence = object.sequence ?? 0;
    message.dueHeight = object.dueHeight !== undefined && object.dueHeight !== null ? BigInt(object.dueHeight.toString()) : BigInt(0);
    message.executedHeight = object.executedHeight !== undefined && object.executedHeight !== null ? BigInt(object.executedHeight.toString()) : BigInt(0);
    message.recipient = object.recipient ?? "";
    message.amountUzrn = object.amountUzrn ?? "";
    message.feeUzrn = object.feeUzrn ?? "";
    message.actionSha256 = object.actionSha256 ?? "";
    message.outcome = object.outcome ?? 0;
    message.failureCode = object.failureCode ?? "";
    return message;
  }
};