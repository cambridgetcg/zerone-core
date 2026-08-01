import { EmergencyCeremony, EmergencyAuditEntry, EmergencyRecoveryAuthorization } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
    /**
     * Identifier of the halt ceremony (or deterministic legacy quarantine
     * marker) that an evidence-bound resume must reference.
     */
    activeHaltCeremonyId: string;
    /**
     * Block at which transaction quarantine began. This is an escalation
     * clock, never an automatic resume deadline.
     */
    haltStartBlock: bigint;
    /**
     * Per-Guardian and global proposal counters are consensus anti-abuse state,
     * not disposable node-local telemetry. They must survive export/import.
     */
    guardianProposalCounts: GuardianProposalCount[];
    epochProposalCount: bigint;
    lastProposalBlock: bigint;
    /**
     * Most recent quarantine escalation boundary already reported. Persisting
     * it makes diagnostics idempotent across restart and resume voting.
     */
    lastHaltEscalationBlock: bigint;
    /**
     * Block in which an affirmative resume finalized. Transaction admission
     * remains quarantined through this entire block and reopens at H+1.
     */
    quarantineReleaseBlock: bigint;
    /**
     * Finalized Guardian authorization for one exact SDK governance recovery
     * proposal in the currently active quarantine incident.
     */
    recoveryAuthorization?: EmergencyRecoveryAuthorization;
}
/**
 * GuardianProposalCount is one canonical, sorted anti-abuse counter.
 * @name GuardianProposalCount
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.GuardianProposalCount
 */
export interface GuardianProposalCount {
    guardian: string;
    count: bigint;
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
     * One shared anti-abuse budget for halt, resume, recovery-authorization,
     * and recovery-revocation ceremonies. A Guardian cannot bypass a limit by
     * switching ceremony lanes.
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
     * Escalation deadline. Crossing it alerts operators but never resumes
     * transaction admission automatically.
     */
    maxHaltDurationBlocks: bigint;
}
/**
 * GenesisState defines the emergency module's genesis state.
 * @name GenesisState
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * GuardianProposalCount is one canonical, sorted anti-abuse counter.
 * @name GuardianProposalCount
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.GuardianProposalCount
 */
export declare const GuardianProposalCount: {
    typeUrl: string;
    encode(message: GuardianProposalCount, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GuardianProposalCount;
    fromPartial(object: DeepPartial<GuardianProposalCount>): GuardianProposalCount;
};
/**
 * Params defines the emergency module parameters.
 * @name Params
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
