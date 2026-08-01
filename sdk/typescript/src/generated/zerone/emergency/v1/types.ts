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
  justification: string;
  /**
   * SHA-256 of the canonical, signed recovery manifest reviewed by voters.
   */
  recoveryManifestSha256: string;
}
/**
 * EmergencyRecoveryAuthorizationProposal asks the immutable Guardian
 * electorate to authorize one exact SDK-governance recovery action while
 * transaction admission remains quarantined.
 * @name EmergencyRecoveryAuthorizationProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorizationProposal
 */
export interface EmergencyRecoveryAuthorizationProposal {
  id: string;
  proposer: string;
  haltCeremonyId: string;
  sdkGovProposalId: bigint;
  /**
   * SHA-256 over the domain-separated TypeURL and raw Any value bytes of the
   * proposal's sole MsgSoftwareUpgrade or MsgCancelUpgrade action.
   */
  actionSha256: string;
  recoveryManifestSha256: string;
  justification: string;
  /**
   * Canonical digest of the plan being scheduled or, for cancellation, the
   * exact currently scheduled plan whose removal Guardians authorize.
   */
  upgradePlanSha256: string;
  /**
   * Account allowed to submit the pre-authorized next SDK proposal, preventing
   * mempool observers from front-running the recovery envelope.
   */
  authorizedSubmitter: string;
  /**
   * "software_upgrade", "cancel_upgrade", or "revoke".
   */
  actionType: string;
  generation: bigint;
}
/**
 * EmergencyRecoveryAuthorization is the finalized, incident-bound capability
 * for one exact SDK-governance proposal. It does not resume transactions.
 * @name EmergencyRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorization
 */
export interface EmergencyRecoveryAuthorization {
  haltCeremonyId: string;
  authorizationCeremonyId: string;
  sdkGovProposalId: bigint;
  actionSha256: string;
  recoveryManifestSha256: string;
  authorizedAtBlock: bigint;
  upgradePlanSha256: string;
  /**
   * Set after SDK governance reaches a terminal result. An authorization with
   * a non-empty outcome cannot admit further votes, deposits, or execution.
   */
  terminalAtBlock: bigint;
  outcome: string;
  authorizedSubmitter: string;
  actionType: string;
  generation: bigint;
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
 * EmergencyElectorateMember is one address and its immutable voting power for
 * a ceremony. The complete, sorted electorate is snapshotted when the
 * ceremony opens so staking, council-expiry, and parameter changes cannot
 * alter an in-flight decision.
 * @name EmergencyElectorateMember
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyElectorateMember
 */
export interface EmergencyElectorateMember {
  address: string;
  /**
   * positive bigint string
   */
  power: string;
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
  /**
   * Version 1 snapshots the exact electorate and quorum policy at creation.
   * A non-terminal legacy ceremony without a complete snapshot fails closed.
   */
  electorateSnapshotVersion: number;
  electorate: EmergencyElectorateMember[];
  /**
   * positive bigint string
   */
  electorateTotalPower: string;
  /**
   * 1,000,000 = 100%
   */
  quorumThreshold: bigint;
  minDistinctVoters: bigint;
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
    revertCeremonyId: "",
    justification: "",
    recoveryManifestSha256: ""
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
    if (message.justification !== "") {
      writer.uint32(66).string(message.justification);
    }
    if (message.recoveryManifestSha256 !== "") {
      writer.uint32(74).string(message.recoveryManifestSha256);
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
        case 8:
          message.justification = reader.string();
          break;
        case 9:
          message.recoveryManifestSha256 = reader.string();
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
    message.justification = object.justification ?? "";
    message.recoveryManifestSha256 = object.recoveryManifestSha256 ?? "";
    return message;
  }
};
function createBaseEmergencyRecoveryAuthorizationProposal(): EmergencyRecoveryAuthorizationProposal {
  return {
    id: "",
    proposer: "",
    haltCeremonyId: "",
    sdkGovProposalId: BigInt(0),
    actionSha256: "",
    recoveryManifestSha256: "",
    justification: "",
    upgradePlanSha256: "",
    authorizedSubmitter: "",
    actionType: "",
    generation: BigInt(0)
  };
}
/**
 * EmergencyRecoveryAuthorizationProposal asks the immutable Guardian
 * electorate to authorize one exact SDK-governance recovery action while
 * transaction admission remains quarantined.
 * @name EmergencyRecoveryAuthorizationProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorizationProposal
 */
export const EmergencyRecoveryAuthorizationProposal = {
  typeUrl: "/zerone.emergency.v1.EmergencyRecoveryAuthorizationProposal",
  encode(message: EmergencyRecoveryAuthorizationProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.haltCeremonyId !== "") {
      writer.uint32(26).string(message.haltCeremonyId);
    }
    if (message.sdkGovProposalId !== BigInt(0)) {
      writer.uint32(32).uint64(message.sdkGovProposalId);
    }
    if (message.actionSha256 !== "") {
      writer.uint32(42).string(message.actionSha256);
    }
    if (message.recoveryManifestSha256 !== "") {
      writer.uint32(50).string(message.recoveryManifestSha256);
    }
    if (message.justification !== "") {
      writer.uint32(58).string(message.justification);
    }
    if (message.upgradePlanSha256 !== "") {
      writer.uint32(66).string(message.upgradePlanSha256);
    }
    if (message.authorizedSubmitter !== "") {
      writer.uint32(74).string(message.authorizedSubmitter);
    }
    if (message.actionType !== "") {
      writer.uint32(82).string(message.actionType);
    }
    if (message.generation !== BigInt(0)) {
      writer.uint32(88).uint64(message.generation);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRecoveryAuthorizationProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyRecoveryAuthorizationProposal();
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
          message.haltCeremonyId = reader.string();
          break;
        case 4:
          message.sdkGovProposalId = reader.uint64();
          break;
        case 5:
          message.actionSha256 = reader.string();
          break;
        case 6:
          message.recoveryManifestSha256 = reader.string();
          break;
        case 7:
          message.justification = reader.string();
          break;
        case 8:
          message.upgradePlanSha256 = reader.string();
          break;
        case 9:
          message.authorizedSubmitter = reader.string();
          break;
        case 10:
          message.actionType = reader.string();
          break;
        case 11:
          message.generation = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyRecoveryAuthorizationProposal>): EmergencyRecoveryAuthorizationProposal {
    const message = createBaseEmergencyRecoveryAuthorizationProposal();
    message.id = object.id ?? "";
    message.proposer = object.proposer ?? "";
    message.haltCeremonyId = object.haltCeremonyId ?? "";
    message.sdkGovProposalId = object.sdkGovProposalId !== undefined && object.sdkGovProposalId !== null ? BigInt(object.sdkGovProposalId.toString()) : BigInt(0);
    message.actionSha256 = object.actionSha256 ?? "";
    message.recoveryManifestSha256 = object.recoveryManifestSha256 ?? "";
    message.justification = object.justification ?? "";
    message.upgradePlanSha256 = object.upgradePlanSha256 ?? "";
    message.authorizedSubmitter = object.authorizedSubmitter ?? "";
    message.actionType = object.actionType ?? "";
    message.generation = object.generation !== undefined && object.generation !== null ? BigInt(object.generation.toString()) : BigInt(0);
    return message;
  }
};
function createBaseEmergencyRecoveryAuthorization(): EmergencyRecoveryAuthorization {
  return {
    haltCeremonyId: "",
    authorizationCeremonyId: "",
    sdkGovProposalId: BigInt(0),
    actionSha256: "",
    recoveryManifestSha256: "",
    authorizedAtBlock: BigInt(0),
    upgradePlanSha256: "",
    terminalAtBlock: BigInt(0),
    outcome: "",
    authorizedSubmitter: "",
    actionType: "",
    generation: BigInt(0)
  };
}
/**
 * EmergencyRecoveryAuthorization is the finalized, incident-bound capability
 * for one exact SDK-governance proposal. It does not resume transactions.
 * @name EmergencyRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorization
 */
export const EmergencyRecoveryAuthorization = {
  typeUrl: "/zerone.emergency.v1.EmergencyRecoveryAuthorization",
  encode(message: EmergencyRecoveryAuthorization, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.haltCeremonyId !== "") {
      writer.uint32(10).string(message.haltCeremonyId);
    }
    if (message.authorizationCeremonyId !== "") {
      writer.uint32(18).string(message.authorizationCeremonyId);
    }
    if (message.sdkGovProposalId !== BigInt(0)) {
      writer.uint32(24).uint64(message.sdkGovProposalId);
    }
    if (message.actionSha256 !== "") {
      writer.uint32(34).string(message.actionSha256);
    }
    if (message.recoveryManifestSha256 !== "") {
      writer.uint32(42).string(message.recoveryManifestSha256);
    }
    if (message.authorizedAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.authorizedAtBlock);
    }
    if (message.upgradePlanSha256 !== "") {
      writer.uint32(58).string(message.upgradePlanSha256);
    }
    if (message.terminalAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.terminalAtBlock);
    }
    if (message.outcome !== "") {
      writer.uint32(74).string(message.outcome);
    }
    if (message.authorizedSubmitter !== "") {
      writer.uint32(82).string(message.authorizedSubmitter);
    }
    if (message.actionType !== "") {
      writer.uint32(90).string(message.actionType);
    }
    if (message.generation !== BigInt(0)) {
      writer.uint32(96).uint64(message.generation);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRecoveryAuthorization {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyRecoveryAuthorization();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.haltCeremonyId = reader.string();
          break;
        case 2:
          message.authorizationCeremonyId = reader.string();
          break;
        case 3:
          message.sdkGovProposalId = reader.uint64();
          break;
        case 4:
          message.actionSha256 = reader.string();
          break;
        case 5:
          message.recoveryManifestSha256 = reader.string();
          break;
        case 6:
          message.authorizedAtBlock = reader.uint64();
          break;
        case 7:
          message.upgradePlanSha256 = reader.string();
          break;
        case 8:
          message.terminalAtBlock = reader.uint64();
          break;
        case 9:
          message.outcome = reader.string();
          break;
        case 10:
          message.authorizedSubmitter = reader.string();
          break;
        case 11:
          message.actionType = reader.string();
          break;
        case 12:
          message.generation = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyRecoveryAuthorization>): EmergencyRecoveryAuthorization {
    const message = createBaseEmergencyRecoveryAuthorization();
    message.haltCeremonyId = object.haltCeremonyId ?? "";
    message.authorizationCeremonyId = object.authorizationCeremonyId ?? "";
    message.sdkGovProposalId = object.sdkGovProposalId !== undefined && object.sdkGovProposalId !== null ? BigInt(object.sdkGovProposalId.toString()) : BigInt(0);
    message.actionSha256 = object.actionSha256 ?? "";
    message.recoveryManifestSha256 = object.recoveryManifestSha256 ?? "";
    message.authorizedAtBlock = object.authorizedAtBlock !== undefined && object.authorizedAtBlock !== null ? BigInt(object.authorizedAtBlock.toString()) : BigInt(0);
    message.upgradePlanSha256 = object.upgradePlanSha256 ?? "";
    message.terminalAtBlock = object.terminalAtBlock !== undefined && object.terminalAtBlock !== null ? BigInt(object.terminalAtBlock.toString()) : BigInt(0);
    message.outcome = object.outcome ?? "";
    message.authorizedSubmitter = object.authorizedSubmitter ?? "";
    message.actionType = object.actionType ?? "";
    message.generation = object.generation !== undefined && object.generation !== null ? BigInt(object.generation.toString()) : BigInt(0);
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
function createBaseEmergencyElectorateMember(): EmergencyElectorateMember {
  return {
    address: "",
    power: ""
  };
}
/**
 * EmergencyElectorateMember is one address and its immutable voting power for
 * a ceremony. The complete, sorted electorate is snapshotted when the
 * ceremony opens so staking, council-expiry, and parameter changes cannot
 * alter an in-flight decision.
 * @name EmergencyElectorateMember
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyElectorateMember
 */
export const EmergencyElectorateMember = {
  typeUrl: "/zerone.emergency.v1.EmergencyElectorateMember",
  encode(message: EmergencyElectorateMember, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.power !== "") {
      writer.uint32(18).string(message.power);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyElectorateMember {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyElectorateMember();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.power = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyElectorateMember>): EmergencyElectorateMember {
    const message = createBaseEmergencyElectorateMember();
    message.address = object.address ?? "";
    message.power = object.power ?? "";
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
    failureReason: "",
    electorateSnapshotVersion: 0,
    electorate: [],
    electorateTotalPower: "",
    quorumThreshold: BigInt(0),
    minDistinctVoters: BigInt(0)
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
    if (message.electorateSnapshotVersion !== 0) {
      writer.uint32(120).uint32(message.electorateSnapshotVersion);
    }
    for (const v of message.electorate) {
      EmergencyElectorateMember.encode(v!, writer.uint32(130).fork()).ldelim();
    }
    if (message.electorateTotalPower !== "") {
      writer.uint32(138).string(message.electorateTotalPower);
    }
    if (message.quorumThreshold !== BigInt(0)) {
      writer.uint32(144).uint64(message.quorumThreshold);
    }
    if (message.minDistinctVoters !== BigInt(0)) {
      writer.uint32(152).uint64(message.minDistinctVoters);
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
        case 15:
          message.electorateSnapshotVersion = reader.uint32();
          break;
        case 16:
          message.electorate.push(EmergencyElectorateMember.decode(reader, reader.uint32()));
          break;
        case 17:
          message.electorateTotalPower = reader.string();
          break;
        case 18:
          message.quorumThreshold = reader.uint64();
          break;
        case 19:
          message.minDistinctVoters = reader.uint64();
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
    message.electorateSnapshotVersion = object.electorateSnapshotVersion ?? 0;
    message.electorate = object.electorate?.map(e => EmergencyElectorateMember.fromPartial(e)) || [];
    message.electorateTotalPower = object.electorateTotalPower ?? "";
    message.quorumThreshold = object.quorumThreshold !== undefined && object.quorumThreshold !== null ? BigInt(object.quorumThreshold.toString()) : BigInt(0);
    message.minDistinctVoters = object.minDistinctVoters !== undefined && object.minDistinctVoters !== null ? BigInt(object.minDistinctVoters.toString()) : BigInt(0);
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