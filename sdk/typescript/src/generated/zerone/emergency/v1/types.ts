//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * EmergencyHaltProposal is the data attached to a halt ceremony.
 * @name EmergencyHaltProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyHaltProposal
 */
export interface EmergencyHaltProposal {
  id: string;
  proposer: string;
  reason: string;
  evidenceHash: string;
  proposedAtBlock: bigint;
  suggestedRevertBlock: bigint;
}
/**
 * EmergencyRevertProposal is the data attached to a revert ceremony.
 * @name EmergencyRevertProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRevertProposal
 */
export interface EmergencyRevertProposal {
  id: string;
  proposer: string;
  targetBlockNumber: bigint;
  targetBlockHash: string;
  targetStateRoot: string;
  justification: string;
  haltCeremonyId: string;
}
/**
 * EmergencyResumeProposal is the data attached to a resume ceremony.
 * @name EmergencyResumeProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyResumeProposal
 */
export interface EmergencyResumeProposal {
  id: string;
  proposer: string;
  resumeFromBlock: bigint;
  resumeFromHash: string;
  resumeStateRoot: string;
  haltCeremonyId: string;
  revertCeremonyId: string;
}
/**
 * EmergencyVote represents a prevote (yes/no).
 * @name EmergencyVote
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyVote
 */
export interface EmergencyVote {
  voter: string;
  approve: boolean;
}
/**
 * EmergencyPrecommit represents a precommit (commit to yes prevote).
 * @name EmergencyPrecommit
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyPrecommit
 */
export interface EmergencyPrecommit {
  voter: string;
}
/**
 * PrevoteEntry is a map entry for ceremony prevotes.
 * @name PrevoteEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrevoteEntry
 */
export interface PrevoteEntry {
  key: string;
  value?: EmergencyVote;
}
/**
 * PrecommitEntry is a map entry for ceremony precommits.
 * @name PrecommitEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrecommitEntry
 */
export interface PrecommitEntry {
  key: string;
  value?: EmergencyPrecommit;
}
/**
 * EmergencyCeremony tracks a 2-phase BFT ceremony (prevote → precommit).
 * @name EmergencyCeremony
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyCeremony
 */
export interface EmergencyCeremony {
  id: string;
  /**
   * halt, revert, resume
   */
  type: string;
  /**
   * prevote, precommit, finalized, failed
   */
  phase: string;
  proposalData: Uint8Array;
  startBlock: bigint;
  prevoteDeadline: bigint;
  precommitDeadline: bigint;
  timeoutDeadline: bigint;
  prevotes: PrevoteEntry[];
  precommits: PrecommitEntry[];
  /**
   * bigint string
   */
  yesPrevoteStake: string;
  /**
   * bigint string
   */
  noPrevoteStake: string;
  /**
   * bigint string
   */
  precommitStake: string;
  failureReason: string;
}
/**
 * EmergencyAuditEntry records a single emergency action for the audit trail.
 * @name EmergencyAuditEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyAuditEntry
 */
export interface EmergencyAuditEntry {
  timestamp: bigint;
  blockNumber: bigint;
  action: string;
  actor: string;
  ceremonyId: string;
  details: string;
}
function createBaseEmergencyHaltProposal(): EmergencyHaltProposal {
  return {
    id: "",
    proposer: "",
    reason: "",
    evidenceHash: "",
    proposedAtBlock: BigInt(0),
    suggestedRevertBlock: BigInt(0)
  };
}
/**
 * EmergencyHaltProposal is the data attached to a halt ceremony.
 * @name EmergencyHaltProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyHaltProposal
 */
export const EmergencyHaltProposal = {
  typeUrl: "/zerone.emergency.v1.EmergencyHaltProposal",
  encode(message: EmergencyHaltProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.evidenceHash !== "") {
      writer.uint32(34).string(message.evidenceHash);
    }
    if (message.proposedAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.proposedAtBlock);
    }
    if (message.suggestedRevertBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.suggestedRevertBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyHaltProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyHaltProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.proposer = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.evidenceHash = reader.string();
          break;
        case 5:
          message.proposedAtBlock = reader.uint64();
          break;
        case 6:
          message.suggestedRevertBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyHaltProposal>): EmergencyHaltProposal {
    const message = createBaseEmergencyHaltProposal();
    message.id = object.id ?? "";
    message.proposer = object.proposer ?? "";
    message.reason = object.reason ?? "";
    message.evidenceHash = object.evidenceHash ?? "";
    message.proposedAtBlock = object.proposedAtBlock !== undefined && object.proposedAtBlock !== null ? BigInt(object.proposedAtBlock.toString()) : BigInt(0);
    message.suggestedRevertBlock = object.suggestedRevertBlock !== undefined && object.suggestedRevertBlock !== null ? BigInt(object.suggestedRevertBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseEmergencyRevertProposal(): EmergencyRevertProposal {
  return {
    id: "",
    proposer: "",
    targetBlockNumber: BigInt(0),
    targetBlockHash: "",
    targetStateRoot: "",
    justification: "",
    haltCeremonyId: ""
  };
}
/**
 * EmergencyRevertProposal is the data attached to a revert ceremony.
 * @name EmergencyRevertProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRevertProposal
 */
export const EmergencyRevertProposal = {
  typeUrl: "/zerone.emergency.v1.EmergencyRevertProposal",
  encode(message: EmergencyRevertProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.targetBlockNumber !== BigInt(0)) {
      writer.uint32(24).uint64(message.targetBlockNumber);
    }
    if (message.targetBlockHash !== "") {
      writer.uint32(34).string(message.targetBlockHash);
    }
    if (message.targetStateRoot !== "") {
      writer.uint32(42).string(message.targetStateRoot);
    }
    if (message.justification !== "") {
      writer.uint32(50).string(message.justification);
    }
    if (message.haltCeremonyId !== "") {
      writer.uint32(58).string(message.haltCeremonyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRevertProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyRevertProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.proposer = reader.string();
          break;
        case 3:
          message.targetBlockNumber = reader.uint64();
          break;
        case 4:
          message.targetBlockHash = reader.string();
          break;
        case 5:
          message.targetStateRoot = reader.string();
          break;
        case 6:
          message.justification = reader.string();
          break;
        case 7:
          message.haltCeremonyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyRevertProposal>): EmergencyRevertProposal {
    const message = createBaseEmergencyRevertProposal();
    message.id = object.id ?? "";
    message.proposer = object.proposer ?? "";
    message.targetBlockNumber = object.targetBlockNumber !== undefined && object.targetBlockNumber !== null ? BigInt(object.targetBlockNumber.toString()) : BigInt(0);
    message.targetBlockHash = object.targetBlockHash ?? "";
    message.targetStateRoot = object.targetStateRoot ?? "";
    message.justification = object.justification ?? "";
    message.haltCeremonyId = object.haltCeremonyId ?? "";
    return message;
  }
};
function createBaseEmergencyResumeProposal(): EmergencyResumeProposal {
  return {
    id: "",
    proposer: "",
    resumeFromBlock: BigInt(0),
    resumeFromHash: "",
    resumeStateRoot: "",
    haltCeremonyId: "",
    revertCeremonyId: ""
  };
}
/**
 * EmergencyResumeProposal is the data attached to a resume ceremony.
 * @name EmergencyResumeProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyResumeProposal
 */
export const EmergencyResumeProposal = {
  typeUrl: "/zerone.emergency.v1.EmergencyResumeProposal",
  encode(message: EmergencyResumeProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.resumeFromBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.resumeFromBlock);
    }
    if (message.resumeFromHash !== "") {
      writer.uint32(34).string(message.resumeFromHash);
    }
    if (message.resumeStateRoot !== "") {
      writer.uint32(42).string(message.resumeStateRoot);
    }
    if (message.haltCeremonyId !== "") {
      writer.uint32(50).string(message.haltCeremonyId);
    }
    if (message.revertCeremonyId !== "") {
      writer.uint32(58).string(message.revertCeremonyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyResumeProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyResumeProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.proposer = reader.string();
          break;
        case 3:
          message.resumeFromBlock = reader.uint64();
          break;
        case 4:
          message.resumeFromHash = reader.string();
          break;
        case 5:
          message.resumeStateRoot = reader.string();
          break;
        case 6:
          message.haltCeremonyId = reader.string();
          break;
        case 7:
          message.revertCeremonyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyResumeProposal>): EmergencyResumeProposal {
    const message = createBaseEmergencyResumeProposal();
    message.id = object.id ?? "";
    message.proposer = object.proposer ?? "";
    message.resumeFromBlock = object.resumeFromBlock !== undefined && object.resumeFromBlock !== null ? BigInt(object.resumeFromBlock.toString()) : BigInt(0);
    message.resumeFromHash = object.resumeFromHash ?? "";
    message.resumeStateRoot = object.resumeStateRoot ?? "";
    message.haltCeremonyId = object.haltCeremonyId ?? "";
    message.revertCeremonyId = object.revertCeremonyId ?? "";
    return message;
  }
};
function createBaseEmergencyVote(): EmergencyVote {
  return {
    voter: "",
    approve: false
  };
}
/**
 * EmergencyVote represents a prevote (yes/no).
 * @name EmergencyVote
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyVote
 */
export const EmergencyVote = {
  typeUrl: "/zerone.emergency.v1.EmergencyVote",
  encode(message: EmergencyVote, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.approve === true) {
      writer.uint32(16).bool(message.approve);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyVote {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyVote>): EmergencyVote {
    const message = createBaseEmergencyVote();
    message.voter = object.voter ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseEmergencyPrecommit(): EmergencyPrecommit {
  return {
    voter: ""
  };
}
/**
 * EmergencyPrecommit represents a precommit (commit to yes prevote).
 * @name EmergencyPrecommit
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyPrecommit
 */
export const EmergencyPrecommit = {
  typeUrl: "/zerone.emergency.v1.EmergencyPrecommit",
  encode(message: EmergencyPrecommit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyPrecommit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyPrecommit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyPrecommit>): EmergencyPrecommit {
    const message = createBaseEmergencyPrecommit();
    message.voter = object.voter ?? "";
    return message;
  }
};
function createBasePrevoteEntry(): PrevoteEntry {
  return {
    key: "",
    value: undefined
  };
}
/**
 * PrevoteEntry is a map entry for ceremony prevotes.
 * @name PrevoteEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrevoteEntry
 */
export const PrevoteEntry = {
  typeUrl: "/zerone.emergency.v1.PrevoteEntry",
  encode(message: PrevoteEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      EmergencyVote.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PrevoteEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrevoteEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = EmergencyVote.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PrevoteEntry>): PrevoteEntry {
    const message = createBasePrevoteEntry();
    message.key = object.key ?? "";
    message.value = object.value !== undefined && object.value !== null ? EmergencyVote.fromPartial(object.value) : undefined;
    return message;
  }
};
function createBasePrecommitEntry(): PrecommitEntry {
  return {
    key: "",
    value: undefined
  };
}
/**
 * PrecommitEntry is a map entry for ceremony precommits.
 * @name PrecommitEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrecommitEntry
 */
export const PrecommitEntry = {
  typeUrl: "/zerone.emergency.v1.PrecommitEntry",
  encode(message: PrecommitEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      EmergencyPrecommit.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PrecommitEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrecommitEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = EmergencyPrecommit.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PrecommitEntry>): PrecommitEntry {
    const message = createBasePrecommitEntry();
    message.key = object.key ?? "";
    message.value = object.value !== undefined && object.value !== null ? EmergencyPrecommit.fromPartial(object.value) : undefined;
    return message;
  }
};
function createBaseEmergencyCeremony(): EmergencyCeremony {
  return {
    id: "",
    type: "",
    phase: "",
    proposalData: new Uint8Array(),
    startBlock: BigInt(0),
    prevoteDeadline: BigInt(0),
    precommitDeadline: BigInt(0),
    timeoutDeadline: BigInt(0),
    prevotes: [],
    precommits: [],
    yesPrevoteStake: "",
    noPrevoteStake: "",
    precommitStake: "",
    failureReason: ""
  };
}
/**
 * EmergencyCeremony tracks a 2-phase BFT ceremony (prevote → precommit).
 * @name EmergencyCeremony
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyCeremony
 */
export const EmergencyCeremony = {
  typeUrl: "/zerone.emergency.v1.EmergencyCeremony",
  encode(message: EmergencyCeremony, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.type !== "") {
      writer.uint32(18).string(message.type);
    }
    if (message.phase !== "") {
      writer.uint32(26).string(message.phase);
    }
    if (message.proposalData.length !== 0) {
      writer.uint32(34).bytes(message.proposalData);
    }
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.startBlock);
    }
    if (message.prevoteDeadline !== BigInt(0)) {
      writer.uint32(48).uint64(message.prevoteDeadline);
    }
    if (message.precommitDeadline !== BigInt(0)) {
      writer.uint32(56).uint64(message.precommitDeadline);
    }
    if (message.timeoutDeadline !== BigInt(0)) {
      writer.uint32(64).uint64(message.timeoutDeadline);
    }
    for (const v of message.prevotes) {
      PrevoteEntry.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.precommits) {
      PrecommitEntry.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    if (message.yesPrevoteStake !== "") {
      writer.uint32(90).string(message.yesPrevoteStake);
    }
    if (message.noPrevoteStake !== "") {
      writer.uint32(98).string(message.noPrevoteStake);
    }
    if (message.precommitStake !== "") {
      writer.uint32(106).string(message.precommitStake);
    }
    if (message.failureReason !== "") {
      writer.uint32(114).string(message.failureReason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyCeremony {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyCeremony();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.type = reader.string();
          break;
        case 3:
          message.phase = reader.string();
          break;
        case 4:
          message.proposalData = reader.bytes();
          break;
        case 5:
          message.startBlock = reader.uint64();
          break;
        case 6:
          message.prevoteDeadline = reader.uint64();
          break;
        case 7:
          message.precommitDeadline = reader.uint64();
          break;
        case 8:
          message.timeoutDeadline = reader.uint64();
          break;
        case 9:
          message.prevotes.push(PrevoteEntry.decode(reader, reader.uint32()));
          break;
        case 10:
          message.precommits.push(PrecommitEntry.decode(reader, reader.uint32()));
          break;
        case 11:
          message.yesPrevoteStake = reader.string();
          break;
        case 12:
          message.noPrevoteStake = reader.string();
          break;
        case 13:
          message.precommitStake = reader.string();
          break;
        case 14:
          message.failureReason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyCeremony>): EmergencyCeremony {
    const message = createBaseEmergencyCeremony();
    message.id = object.id ?? "";
    message.type = object.type ?? "";
    message.phase = object.phase ?? "";
    message.proposalData = object.proposalData ?? new Uint8Array();
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.prevoteDeadline = object.prevoteDeadline !== undefined && object.prevoteDeadline !== null ? BigInt(object.prevoteDeadline.toString()) : BigInt(0);
    message.precommitDeadline = object.precommitDeadline !== undefined && object.precommitDeadline !== null ? BigInt(object.precommitDeadline.toString()) : BigInt(0);
    message.timeoutDeadline = object.timeoutDeadline !== undefined && object.timeoutDeadline !== null ? BigInt(object.timeoutDeadline.toString()) : BigInt(0);
    message.prevotes = object.prevotes?.map(e => PrevoteEntry.fromPartial(e)) || [];
    message.precommits = object.precommits?.map(e => PrecommitEntry.fromPartial(e)) || [];
    message.yesPrevoteStake = object.yesPrevoteStake ?? "";
    message.noPrevoteStake = object.noPrevoteStake ?? "";
    message.precommitStake = object.precommitStake ?? "";
    message.failureReason = object.failureReason ?? "";
    return message;
  }
};
function createBaseEmergencyAuditEntry(): EmergencyAuditEntry {
  return {
    timestamp: BigInt(0),
    blockNumber: BigInt(0),
    action: "",
    actor: "",
    ceremonyId: "",
    details: ""
  };
}
/**
 * EmergencyAuditEntry records a single emergency action for the audit trail.
 * @name EmergencyAuditEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyAuditEntry
 */
export const EmergencyAuditEntry = {
  typeUrl: "/zerone.emergency.v1.EmergencyAuditEntry",
  encode(message: EmergencyAuditEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.timestamp !== BigInt(0)) {
      writer.uint32(8).int64(message.timestamp);
    }
    if (message.blockNumber !== BigInt(0)) {
      writer.uint32(16).uint64(message.blockNumber);
    }
    if (message.action !== "") {
      writer.uint32(26).string(message.action);
    }
    if (message.actor !== "") {
      writer.uint32(34).string(message.actor);
    }
    if (message.ceremonyId !== "") {
      writer.uint32(42).string(message.ceremonyId);
    }
    if (message.details !== "") {
      writer.uint32(50).string(message.details);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyAuditEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyAuditEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.timestamp = reader.int64();
          break;
        case 2:
          message.blockNumber = reader.uint64();
          break;
        case 3:
          message.action = reader.string();
          break;
        case 4:
          message.actor = reader.string();
          break;
        case 5:
          message.ceremonyId = reader.string();
          break;
        case 6:
          message.details = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyAuditEntry>): EmergencyAuditEntry {
    const message = createBaseEmergencyAuditEntry();
    message.timestamp = object.timestamp !== undefined && object.timestamp !== null ? BigInt(object.timestamp.toString()) : BigInt(0);
    message.blockNumber = object.blockNumber !== undefined && object.blockNumber !== null ? BigInt(object.blockNumber.toString()) : BigInt(0);
    message.action = object.action ?? "";
    message.actor = object.actor ?? "";
    message.ceremonyId = object.ceremonyId ?? "";
    message.details = object.details ?? "";
    return message;
  }
};