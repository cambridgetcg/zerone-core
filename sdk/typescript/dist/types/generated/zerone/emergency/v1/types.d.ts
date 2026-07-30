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
 * EmergencyRecoveryAuthorizationProposal asks the immutable Guardian
 * electorate to authorize one exact SDK-governance recovery action while
 * transaction admission remains quarantined.
 * @name EmergencyRecoveryAuthorizationProposal
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorizationProposal
 */
export declare const EmergencyRecoveryAuthorizationProposal: {
    typeUrl: string;
    encode(message: EmergencyRecoveryAuthorizationProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRecoveryAuthorizationProposal;
    fromPartial(object: DeepPartial<EmergencyRecoveryAuthorizationProposal>): EmergencyRecoveryAuthorizationProposal;
};
/**
 * EmergencyRecoveryAuthorization is the finalized, incident-bound capability
 * for one exact SDK-governance proposal. It does not resume transactions.
 * @name EmergencyRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyRecoveryAuthorization
 */
export declare const EmergencyRecoveryAuthorization: {
    typeUrl: string;
    encode(message: EmergencyRecoveryAuthorization, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyRecoveryAuthorization;
    fromPartial(object: DeepPartial<EmergencyRecoveryAuthorization>): EmergencyRecoveryAuthorization;
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
 * EmergencyElectorateMember is one address and its immutable voting power for
 * a ceremony. The complete, sorted electorate is snapshotted when the
 * ceremony opens so staking, council-expiry, and parameter changes cannot
 * alter an in-flight decision.
 * @name EmergencyElectorateMember
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.EmergencyElectorateMember
 */
export declare const EmergencyElectorateMember: {
    typeUrl: string;
    encode(message: EmergencyElectorateMember, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyElectorateMember;
    fromPartial(object: DeepPartial<EmergencyElectorateMember>): EmergencyElectorateMember;
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
