import { EmergencyCeremony, EmergencyAuditEntry } from "./types.js";
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
