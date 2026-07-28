import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * EmergencyHaltProposal is the data attached to a halt ceremony.
 * @name EmergencyHaltProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyHaltProposal
 */
export declare const EmergencyHaltProposal: {
    typeUrl: string;
    encode(message: EmergencyHaltProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyHaltProposal;
    fromPartial(object: DeepPartial<EmergencyHaltProposal>): EmergencyHaltProposal;
};
/**
 * EmergencyRevertProposal is the data attached to a revert ceremony.
 * @name EmergencyRevertProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRevertProposal
 */
export declare const EmergencyRevertProposal: {
    typeUrl: string;
    encode(message: EmergencyRevertProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRevertProposal;
    fromPartial(object: DeepPartial<EmergencyRevertProposal>): EmergencyRevertProposal;
};
/**
 * EmergencyResumeProposal is the data attached to a resume ceremony.
 * @name EmergencyResumeProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyResumeProposal
 */
export declare const EmergencyResumeProposal: {
    typeUrl: string;
    encode(message: EmergencyResumeProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyResumeProposal;
    fromPartial(object: DeepPartial<EmergencyResumeProposal>): EmergencyResumeProposal;
};
/**
 * EmergencyVote represents a prevote (yes/no).
 * @name EmergencyVote
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyVote
 */
export declare const EmergencyVote: {
    typeUrl: string;
    encode(message: EmergencyVote, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyVote;
    fromPartial(object: DeepPartial<EmergencyVote>): EmergencyVote;
};
/**
 * EmergencyPrecommit represents a precommit (commit to yes prevote).
 * @name EmergencyPrecommit
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyPrecommit
 */
export declare const EmergencyPrecommit: {
    typeUrl: string;
    encode(message: EmergencyPrecommit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyPrecommit;
    fromPartial(object: DeepPartial<EmergencyPrecommit>): EmergencyPrecommit;
};
/**
 * PrevoteEntry is a map entry for ceremony prevotes.
 * @name PrevoteEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrevoteEntry
 */
export declare const PrevoteEntry: {
    typeUrl: string;
    encode(message: PrevoteEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PrevoteEntry;
    fromPartial(object: DeepPartial<PrevoteEntry>): PrevoteEntry;
};
/**
 * PrecommitEntry is a map entry for ceremony precommits.
 * @name PrecommitEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.PrecommitEntry
 */
export declare const PrecommitEntry: {
    typeUrl: string;
    encode(message: PrecommitEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PrecommitEntry;
    fromPartial(object: DeepPartial<PrecommitEntry>): PrecommitEntry;
};
/**
 * EmergencyCeremony tracks a 2-phase BFT ceremony (prevote → precommit).
 * @name EmergencyCeremony
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyCeremony
 */
export declare const EmergencyCeremony: {
    typeUrl: string;
    encode(message: EmergencyCeremony, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyCeremony;
    fromPartial(object: DeepPartial<EmergencyCeremony>): EmergencyCeremony;
};
/**
 * EmergencyAuditEntry records a single emergency action for the audit trail.
 * @name EmergencyAuditEntry
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyAuditEntry
 */
export declare const EmergencyAuditEntry: {
    typeUrl: string;
    encode(message: EmergencyAuditEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyAuditEntry;
    fromPartial(object: DeepPartial<EmergencyAuditEntry>): EmergencyAuditEntry;
};
