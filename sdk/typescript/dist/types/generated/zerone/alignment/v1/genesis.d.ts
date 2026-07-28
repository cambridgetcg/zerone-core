import { AlignmentState, AlignmentObservation, DimensionScores, AlignmentHealthIndex, CorrectionRecord } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * GenesisState defines the alignment module's genesis state.
 * @name GenesisState
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    state?: AlignmentState;
    observations: AlignmentObservation[];
    scores: DimensionScores[];
    healthIndices: AlignmentHealthIndex[];
    corrections: CorrectionRecord[];
}
/**
 * Params defines the alignment module parameters.
 * @name Params
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.Params
 */
export interface Params {
    /**
     * How often to observe (in blocks).
     */
    observationIntervalBlocks: bigint;
    /**
     * Dimension weights (BPS, must sum to 1,000,000).
     */
    weightKnowledgeQuality: bigint;
    weightEconomicStability: bigint;
    weightGovernanceParticipation: bigint;
    weightNetworkSecurity: bigint;
    weightStakingRatio: bigint;
    /**
     * Health category thresholds (BPS).
     */
    criticalThreshold: bigint;
    degradedThreshold: bigint;
    healthyThreshold: bigint;
    /**
     * Module enabled flag.
     */
    enabled: boolean;
    /**
     * Maximum correction magnitude (BPS) for auto-apply. Above this requires governance.
     */
    maxAutoApplyMagnitudeBps: bigint;
    /**
     * Correction confidence params (R29-4).
     */
    correctionConfidenceWindowSize: bigint;
    correctionConfidenceMinSamples: bigint;
    minConfidenceForAutoApply: bigint;
    correctionBoundsMinMultiplierBps: bigint;
    correctionBoundsMaxMultiplierBps: bigint;
    /**
     * Correction banding (L7). Corrections smaller than this are *advisory only*:
     * logged + event emitted, but not forwarded to autopoiesis. Prevents the
     * module from chattering at every small deviation.
     */
    advisoryMagnitudeBps: bigint;
}
/**
 * GenesisState defines the alignment module's genesis state.
 * @name GenesisState
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * Params defines the alignment module parameters.
 * @name Params
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
