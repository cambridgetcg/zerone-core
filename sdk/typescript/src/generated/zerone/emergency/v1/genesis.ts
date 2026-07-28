//@ts-nocheck
import { EmergencyCeremony, EmergencyAuditEntry } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState defines the emergency module's genesis state.
 * @name GenesisState
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  status: string;
  ceremonies: EmergencyCeremony[];
  auditLog: EmergencyAuditEntry[];
}
/**
 * Params defines the emergency module parameters.
 * @name Params
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.Params
 */
export interface Params {
  /**
   * Quorum thresholds (1,000,000 = 100%).
   */
  haltQuorum: bigint;
  revertQuorum: bigint;
  resumeQuorum: bigint;
  /**
   * Ceremony timing (blocks).
   */
  haltPrevoteBlocks: bigint;
  haltPrecommitBlocks: bigint;
  haltTimeoutBlocks: bigint;
  revertPrevoteBlocks: bigint;
  revertPrecommitBlocks: bigint;
  revertTimeoutBlocks: bigint;
  resumePrevoteBlocks: bigint;
  resumePrecommitBlocks: bigint;
  resumeTimeoutBlocks: bigint;
  /**
   * Anti-abuse limits.
   */
  maxProposalsPerEpoch: bigint;
  maxProposalsPerGuardianPerEpoch: bigint;
  cooldownBlocks: bigint;
  /**
   * uzrn bigint string
   */
  minGuardianStake: string;
  minDistinctVoters: bigint;
  /**
   * Revert constraints.
   */
  maxRevertDepth: bigint;
  /**
   * Epoch length (blocks).
   */
  epochBlocks: bigint;
  /**
   * Genesis emergency council (H-5 bootstrap).
   */
  genesisCouncil: string[];
  councilExpiryBlock: bigint;
  /**
   * uzrn bigint string
   */
  councilVirtualStake: string;
  /**
   * Auto-resume: max halt duration before automatic resume.
   */
  maxHaltDurationBlocks: bigint;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    status: "",
    ceremonies: [],
    auditLog: []
  };
}
/**
 * GenesisState defines the emergency module's genesis state.
 * @name GenesisState
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.emergency.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    if (message.status !== "") {
      writer.uint32(18).string(message.status);
    }
    for (const v of message.ceremonies) {
      EmergencyCeremony.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.auditLog) {
      EmergencyAuditEntry.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.status = reader.string();
          break;
        case 3:
          message.ceremonies.push(EmergencyCeremony.decode(reader, reader.uint32()));
          break;
        case 4:
          message.auditLog.push(EmergencyAuditEntry.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    message.status = object.status ?? "";
    message.ceremonies = object.ceremonies?.map(e => EmergencyCeremony.fromPartial(e)) || [];
    message.auditLog = object.auditLog?.map(e => EmergencyAuditEntry.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    haltQuorum: BigInt(0),
    revertQuorum: BigInt(0),
    resumeQuorum: BigInt(0),
    haltPrevoteBlocks: BigInt(0),
    haltPrecommitBlocks: BigInt(0),
    haltTimeoutBlocks: BigInt(0),
    revertPrevoteBlocks: BigInt(0),
    revertPrecommitBlocks: BigInt(0),
    revertTimeoutBlocks: BigInt(0),
    resumePrevoteBlocks: BigInt(0),
    resumePrecommitBlocks: BigInt(0),
    resumeTimeoutBlocks: BigInt(0),
    maxProposalsPerEpoch: BigInt(0),
    maxProposalsPerGuardianPerEpoch: BigInt(0),
    cooldownBlocks: BigInt(0),
    minGuardianStake: "",
    minDistinctVoters: BigInt(0),
    maxRevertDepth: BigInt(0),
    epochBlocks: BigInt(0),
    genesisCouncil: [],
    councilExpiryBlock: BigInt(0),
    councilVirtualStake: "",
    maxHaltDurationBlocks: BigInt(0)
  };
}
/**
 * Params defines the emergency module parameters.
 * @name Params
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.emergency.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.haltQuorum !== BigInt(0)) {
      writer.uint32(8).uint64(message.haltQuorum);
    }
    if (message.revertQuorum !== BigInt(0)) {
      writer.uint32(16).uint64(message.revertQuorum);
    }
    if (message.resumeQuorum !== BigInt(0)) {
      writer.uint32(24).uint64(message.resumeQuorum);
    }
    if (message.haltPrevoteBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.haltPrevoteBlocks);
    }
    if (message.haltPrecommitBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.haltPrecommitBlocks);
    }
    if (message.haltTimeoutBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.haltTimeoutBlocks);
    }
    if (message.revertPrevoteBlocks !== BigInt(0)) {
      writer.uint32(56).uint64(message.revertPrevoteBlocks);
    }
    if (message.revertPrecommitBlocks !== BigInt(0)) {
      writer.uint32(64).uint64(message.revertPrecommitBlocks);
    }
    if (message.revertTimeoutBlocks !== BigInt(0)) {
      writer.uint32(72).uint64(message.revertTimeoutBlocks);
    }
    if (message.resumePrevoteBlocks !== BigInt(0)) {
      writer.uint32(80).uint64(message.resumePrevoteBlocks);
    }
    if (message.resumePrecommitBlocks !== BigInt(0)) {
      writer.uint32(88).uint64(message.resumePrecommitBlocks);
    }
    if (message.resumeTimeoutBlocks !== BigInt(0)) {
      writer.uint32(96).uint64(message.resumeTimeoutBlocks);
    }
    if (message.maxProposalsPerEpoch !== BigInt(0)) {
      writer.uint32(104).uint64(message.maxProposalsPerEpoch);
    }
    if (message.maxProposalsPerGuardianPerEpoch !== BigInt(0)) {
      writer.uint32(112).uint64(message.maxProposalsPerGuardianPerEpoch);
    }
    if (message.cooldownBlocks !== BigInt(0)) {
      writer.uint32(120).uint64(message.cooldownBlocks);
    }
    if (message.minGuardianStake !== "") {
      writer.uint32(130).string(message.minGuardianStake);
    }
    if (message.minDistinctVoters !== BigInt(0)) {
      writer.uint32(136).uint64(message.minDistinctVoters);
    }
    if (message.maxRevertDepth !== BigInt(0)) {
      writer.uint32(144).uint64(message.maxRevertDepth);
    }
    if (message.epochBlocks !== BigInt(0)) {
      writer.uint32(152).uint64(message.epochBlocks);
    }
    for (const v of message.genesisCouncil) {
      writer.uint32(162).string(v!);
    }
    if (message.councilExpiryBlock !== BigInt(0)) {
      writer.uint32(168).uint64(message.councilExpiryBlock);
    }
    if (message.councilVirtualStake !== "") {
      writer.uint32(178).string(message.councilVirtualStake);
    }
    if (message.maxHaltDurationBlocks !== BigInt(0)) {
      writer.uint32(184).uint64(message.maxHaltDurationBlocks);
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
          message.haltQuorum = reader.uint64();
          break;
        case 2:
          message.revertQuorum = reader.uint64();
          break;
        case 3:
          message.resumeQuorum = reader.uint64();
          break;
        case 4:
          message.haltPrevoteBlocks = reader.uint64();
          break;
        case 5:
          message.haltPrecommitBlocks = reader.uint64();
          break;
        case 6:
          message.haltTimeoutBlocks = reader.uint64();
          break;
        case 7:
          message.revertPrevoteBlocks = reader.uint64();
          break;
        case 8:
          message.revertPrecommitBlocks = reader.uint64();
          break;
        case 9:
          message.revertTimeoutBlocks = reader.uint64();
          break;
        case 10:
          message.resumePrevoteBlocks = reader.uint64();
          break;
        case 11:
          message.resumePrecommitBlocks = reader.uint64();
          break;
        case 12:
          message.resumeTimeoutBlocks = reader.uint64();
          break;
        case 13:
          message.maxProposalsPerEpoch = reader.uint64();
          break;
        case 14:
          message.maxProposalsPerGuardianPerEpoch = reader.uint64();
          break;
        case 15:
          message.cooldownBlocks = reader.uint64();
          break;
        case 16:
          message.minGuardianStake = reader.string();
          break;
        case 17:
          message.minDistinctVoters = reader.uint64();
          break;
        case 18:
          message.maxRevertDepth = reader.uint64();
          break;
        case 19:
          message.epochBlocks = reader.uint64();
          break;
        case 20:
          message.genesisCouncil.push(reader.string());
          break;
        case 21:
          message.councilExpiryBlock = reader.uint64();
          break;
        case 22:
          message.councilVirtualStake = reader.string();
          break;
        case 23:
          message.maxHaltDurationBlocks = reader.uint64();
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
    message.haltQuorum = object.haltQuorum !== undefined && object.haltQuorum !== null ? BigInt(object.haltQuorum.toString()) : BigInt(0);
    message.revertQuorum = object.revertQuorum !== undefined && object.revertQuorum !== null ? BigInt(object.revertQuorum.toString()) : BigInt(0);
    message.resumeQuorum = object.resumeQuorum !== undefined && object.resumeQuorum !== null ? BigInt(object.resumeQuorum.toString()) : BigInt(0);
    message.haltPrevoteBlocks = object.haltPrevoteBlocks !== undefined && object.haltPrevoteBlocks !== null ? BigInt(object.haltPrevoteBlocks.toString()) : BigInt(0);
    message.haltPrecommitBlocks = object.haltPrecommitBlocks !== undefined && object.haltPrecommitBlocks !== null ? BigInt(object.haltPrecommitBlocks.toString()) : BigInt(0);
    message.haltTimeoutBlocks = object.haltTimeoutBlocks !== undefined && object.haltTimeoutBlocks !== null ? BigInt(object.haltTimeoutBlocks.toString()) : BigInt(0);
    message.revertPrevoteBlocks = object.revertPrevoteBlocks !== undefined && object.revertPrevoteBlocks !== null ? BigInt(object.revertPrevoteBlocks.toString()) : BigInt(0);
    message.revertPrecommitBlocks = object.revertPrecommitBlocks !== undefined && object.revertPrecommitBlocks !== null ? BigInt(object.revertPrecommitBlocks.toString()) : BigInt(0);
    message.revertTimeoutBlocks = object.revertTimeoutBlocks !== undefined && object.revertTimeoutBlocks !== null ? BigInt(object.revertTimeoutBlocks.toString()) : BigInt(0);
    message.resumePrevoteBlocks = object.resumePrevoteBlocks !== undefined && object.resumePrevoteBlocks !== null ? BigInt(object.resumePrevoteBlocks.toString()) : BigInt(0);
    message.resumePrecommitBlocks = object.resumePrecommitBlocks !== undefined && object.resumePrecommitBlocks !== null ? BigInt(object.resumePrecommitBlocks.toString()) : BigInt(0);
    message.resumeTimeoutBlocks = object.resumeTimeoutBlocks !== undefined && object.resumeTimeoutBlocks !== null ? BigInt(object.resumeTimeoutBlocks.toString()) : BigInt(0);
    message.maxProposalsPerEpoch = object.maxProposalsPerEpoch !== undefined && object.maxProposalsPerEpoch !== null ? BigInt(object.maxProposalsPerEpoch.toString()) : BigInt(0);
    message.maxProposalsPerGuardianPerEpoch = object.maxProposalsPerGuardianPerEpoch !== undefined && object.maxProposalsPerGuardianPerEpoch !== null ? BigInt(object.maxProposalsPerGuardianPerEpoch.toString()) : BigInt(0);
    message.cooldownBlocks = object.cooldownBlocks !== undefined && object.cooldownBlocks !== null ? BigInt(object.cooldownBlocks.toString()) : BigInt(0);
    message.minGuardianStake = object.minGuardianStake ?? "";
    message.minDistinctVoters = object.minDistinctVoters !== undefined && object.minDistinctVoters !== null ? BigInt(object.minDistinctVoters.toString()) : BigInt(0);
    message.maxRevertDepth = object.maxRevertDepth !== undefined && object.maxRevertDepth !== null ? BigInt(object.maxRevertDepth.toString()) : BigInt(0);
    message.epochBlocks = object.epochBlocks !== undefined && object.epochBlocks !== null ? BigInt(object.epochBlocks.toString()) : BigInt(0);
    message.genesisCouncil = object.genesisCouncil?.map(e => e) || [];
    message.councilExpiryBlock = object.councilExpiryBlock !== undefined && object.councilExpiryBlock !== null ? BigInt(object.councilExpiryBlock.toString()) : BigInt(0);
    message.councilVirtualStake = object.councilVirtualStake ?? "";
    message.maxHaltDurationBlocks = object.maxHaltDurationBlocks !== undefined && object.maxHaltDurationBlocks !== null ? BigInt(object.maxHaltDurationBlocks.toString()) : BigInt(0);
    return message;
  }
};